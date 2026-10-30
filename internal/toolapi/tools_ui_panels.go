package toolapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strings"
)

const (
	maxUIPanelControls   = 500
	maxUIPanelDepth      = 16
	maxUIPanelOperations = 100
)

var (
	uiPanelIdentifierRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	uiPanelVarInvalidRE = regexp.MustCompile(`[^A-Za-z0-9_.@^]+`)
	uiPanelContexts     = map[string]bool{
		"player_context": true, "selected_country_context": true,
		"selected_state_context": true, "decision_category": true,
		"diplomatic_action": true, "national_focus_context": true,
		"country_mapicon": true, "state_mapicon": true,
	}
	uiPanelControlKinds = map[string]bool{
		"container": true, "icon": true, "text": true,
		"progressbar": true, "button": true,
	}
)

var uiPanelCapabilityGaps = []string{
	"The current ui-panels route compiles files but does not verify that a panel is mounted and visible in a running HOI4 context.",
}

type uiPanelDiagnostic struct {
	Severity    string `json:"severity"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	ControlName string `json:"control_name,omitempty"`
	ControlPath string `json:"control_path,omitempty"`
}

type getUIPanelIn struct {
	ProjectID int64 `json:"project_id"`
	PanelID   int64 `json:"panel_id" jsonschema:"REQUIRED database id returned by list_ui_panels/create_ui_panel"`
}

type updateUIPanelIn struct {
	ProjectID         int64              `json:"project_id"`
	PanelID           int64              `json:"panel_id" jsonschema:"REQUIRED database id returned by get_ui_panel"`
	ExpectedUpdatedAt string             `json:"expected_updated_at" jsonschema:"REQUIRED exact updated_at returned by get_ui_panel; stale values are rejected before writing"`
	Token             *string            `json:"token,omitempty"`
	CountryTag        *string            `json:"country_tag,omitempty"`
	NameEn            *string            `json:"name_en,omitempty"`
	NameZh            *string            `json:"name_zh,omitempty"`
	DisplayNameLocs   *map[string]string `json:"display_name_locs,omitempty" jsonschema:"merged into the saved names locale by locale; send only the locales that change"`
	ContextType       *string            `json:"context_type,omitempty"`
	ParentWindowToken *string            `json:"parent_window_token,omitempty"`
	VisibleTrigger    *string            `json:"visible_trigger,omitempty"`
	WindowProps       *map[string]any    `json:"window_props,omitempty" jsonschema:"merged into the saved window_props key by key, e.g. {x:50} keeps width/height/background; null removes a key"`
	RawExtras         *map[string]any    `json:"raw_extras,omitempty" jsonschema:"merged into the saved raw_extras key by key; null removes a key"`
}

type uiPanelOperation struct {
	Operation         string         `json:"operation" jsonschema:"REQUIRED add | update | delete | move | reorder"`
	ControlName       string         `json:"control_name,omitempty" jsonschema:"short stable control name; use control_path when names in legacy data are ambiguous"`
	ControlPath       string         `json:"control_path,omitempty" jsonschema:"slash-separated short-name path, e.g. body/status/value"`
	ParentControlName string         `json:"parent_control_name,omitempty" jsonschema:"destination container short name; blank parent name/path means panel root"`
	ParentControlPath string         `json:"parent_control_path,omitempty" jsonschema:"destination container slash path; blank parent name/path means panel root"`
	Index             *int           `json:"index,omitempty" jsonschema:"zero-based final index; omitted add/move appends"`
	Control           map[string]any `json:"control,omitempty" jsonschema:"add only: one complete control object; nested children allowed only on kind=container"`
	Fields            map[string]any `json:"fields,omitempty" jsonschema:"update only: partial scalar/map fields; kind and children cannot be replaced; null removes an optional field"`
	Cascade           bool           `json:"cascade,omitempty" jsonschema:"delete only: required true to delete a non-empty container"`
}

type applyUIPanelOperationsIn struct {
	ProjectID         int64              `json:"project_id"`
	PanelID           int64              `json:"panel_id" jsonschema:"REQUIRED database id returned by get_ui_panel"`
	ExpectedUpdatedAt string             `json:"expected_updated_at" jsonschema:"REQUIRED exact updated_at returned by get_ui_panel"`
	Operations        []uiPanelOperation `json:"operations" jsonschema:"REQUIRED 1..100 ordered operations, committed by one final PUT"`
}

type uiPanelOperationResult struct {
	Operation   string `json:"operation"`
	ControlName string `json:"control_name"`
	BeforePath  string `json:"before_path,omitempty"`
	AfterPath   string `json:"after_path,omitempty"`
}

func uiPanelPath(projectID, panelID int64) string {
	return fmt.Sprintf("/api/projects/%d/ui-panels/%d", projectID, panelID)
}

func copyStringMap(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func uiPanelString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func cloneJSONMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if json.Unmarshal(raw, &out) != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func cloneJSONMaps(in []map[string]any) []map[string]any {
	if in == nil {
		return []map[string]any{}
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return []map[string]any{}
	}
	var out []map[string]any
	if json.Unmarshal(raw, &out) != nil || out == nil {
		return []map[string]any{}
	}
	return out
}

func uiPanelTextChecksum(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func uiPanelFileChecksums(files map[string]any) map[string]string {
	checksums := map[string]string{}
	if file, body := strings.TrimSpace(stringField(files, "gui_path")), stringField(files, "gui_text"); file != "" {
		checksums[file] = uiPanelTextChecksum(body)
	}
	if file, body := strings.TrimSpace(stringField(files, "txt_path")), stringField(files, "txt_text"); file != "" {
		checksums[file] = uiPanelTextChecksum(body)
	}
	if loc, ok := files["loc"].(map[string]any); ok {
		for language, raw := range loc {
			checksums["localisation:"+language] = uiPanelTextChecksum(uiPanelString(raw))
		}
	}
	return checksums
}

func uiPanelMapSlice(value any) ([]map[string]any, bool) {
	switch rows := value.(type) {
	case nil:
		return []map[string]any{}, true
	case []map[string]any:
		return cloneJSONMaps(rows), true
	case []any:
		out := make([]map[string]any, 0, len(rows))
		for _, raw := range rows {
			row, ok := raw.(map[string]any)
			if !ok {
				return nil, false
			}
			out = append(out, cloneJSONMap(row))
		}
		return out, true
	default:
		return nil, false
	}
}

func uiPanelReadback(resp map[string]any, projectID int64) map[string]any {
	out := cloneJSONMap(resp)
	id := int64Field(resp, "id")
	out["id"] = id
	out["panel_id"] = id
	out["project_id"] = projectID
	for _, key := range []string{"display_name_locs", "window_props", "raw_extras", "derived"} {
		if out[key] == nil {
			out[key] = map[string]any{}
		}
	}
	if out["controls"] == nil {
		out["controls"] = []any{}
	}
	if out["warnings"] == nil {
		out["warnings"] = []any{}
	}
	if windowName := uiPanelWindowName(resp); windowName != "" {
		out["window_name"] = windowName
	}
	out["revision_guard"] = map[string]any{
		"client_preflight": true,
		"forwarded_to_put": true,
		"server_enforced":  true,
		"updated_at":       stringField(resp, "updated_at"),
		"conflict_status":  409,
		"conflict_code":    "ui_panel_revision_conflict",
	}
	out["capability_gaps"] = append([]string(nil), uiPanelCapabilityGaps...)
	return out
}

func validateUIPanelID(projectID, panelID int64) error {
	if projectID <= 0 {
		return fmt.Errorf("project_id must be a positive integer")
	}
	if panelID <= 0 {
		return fmt.Errorf("panel_id must be the positive database id returned by list_ui_panels")
	}
	return nil
}

func validateUIPanelIdentifier(value, field string, allowEmpty bool) error {
	value = strings.TrimSpace(value)
	if value == "" && allowEmpty {
		return nil
	}
	if value == "" {
		return fmt.Errorf("%s required", field)
	}
	if !uiPanelIdentifierRE.MatchString(value) {
		return fmt.Errorf("%s must be an ASCII identifier beginning with a letter and containing only letters, digits, or underscore", field)
	}
	return nil
}

func uiPanelIntegerValue(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		if v <= uint64(^uint(0)>>1) {
			return int(v), true
		}
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) && math.Trunc(v) == v {
			return int(v), true
		}
	case float32:
		f := float64(v)
		if !math.IsNaN(f) && !math.IsInf(f, 0) && math.Trunc(f) == f {
			return int(v), true
		}
	case json.Number:
		n, err := v.Int64()
		return int(n), err == nil
	}
	return 0, false
}

func addUIPanelDiagnostic(diags *[]uiPanelDiagnostic, severity, code, message, name, path string) {
	*diags = append(*diags, uiPanelDiagnostic{
		Severity: severity, Code: code, Message: message,
		ControlName: name, ControlPath: path,
	})
}

var uiPanelCommonControlFields = map[string]bool{
	"kind": true, "name": true, "x": true, "y": true,
	"w": true, "h": true, "width": true, "height": true,
	"visible_trigger": true, "raw_extras": true,
}

var uiPanelKindControlFields = map[string]map[string]bool{

	"container": {"children": true, "sprite": true, "scroll": true,
		"array": true, "columns": true, "slot_w": true, "slot_h": true, "change_scope": true},
	"icon":        {"sprite": true, "frame": true, "frame_var": true, "image_expr": true},
	"progressbar": {"sprite": true, "frame": true, "frame_var": true, "image_expr": true},
	"text":        {"text": true, "text_locs": true, "font": true, "format": true},

	"button": {
		"sprite": true, "text": true, "text_locs": true, "font": true,
		"tooltip": true, "tooltip_locs": true, "click_effect": true,
		"click_enabled_trigger": true, "frame_var": true,
	},
}

func uiPanelRawScalar(value any) bool {
	switch typed := value.(type) {
	case string, bool, json.Number,
		int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return !math.IsNaN(typed) && !math.IsInf(typed, 0)
	case float32:
		value := float64(typed)
		return !math.IsNaN(value) && !math.IsInf(value, 0)
	default:
		return false
	}
}

func validateUIPanelRawExtras(value any, panelLevel bool, diagnostics *[]uiPanelDiagnostic, name, controlPath string) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	extras, ok := value.(map[string]any)
	if !ok {
		addUIPanelDiagnostic(diagnostics, "error", "panel.raw_extras_type", "raw_extras must be an object", name, controlPath)
		return map[string]any{}
	}
	extras = cloneJSONMap(extras)
	seenKeys := map[string]string{}
	for key, raw := range extras {
		trimmedKey := strings.TrimSpace(key)
		foldedKey := strings.ToLower(trimmedKey)
		if previous, duplicate := seenKeys[foldedKey]; duplicate {
			addUIPanelDiagnostic(diagnostics, "error", "panel.raw_extras_key_duplicate", fmt.Sprintf("raw_extras keys %q and %q differ only by case; the exporter/engine cannot resolve them deterministically", previous, key), name, controlPath)
		} else {
			seenKeys[foldedKey] = key
		}
		if err := validateUIPanelIdentifier(trimmedKey, "raw_extras key", false); err != nil {
			addUIPanelDiagnostic(diagnostics, "error", "panel.raw_extras_key", err.Error(), name, controlPath)
			continue
		}
		if panelLevel {

			if !uiPanelRawScalar(raw) {
				addUIPanelDiagnostic(diagnostics, "error", "panel.raw_extras_value", "panel raw_extras values must be scalar script text", name, controlPath)
			} else if _, isText := raw.(string); isText {
				validateUIPanelScript(raw, "raw_extras."+trimmedKey, name, controlPath, diagnostics)
			}
			continue
		}
		if uiPanelRawScalar(raw) {
			if _, isText := raw.(string); isText {
				validateUIPanelScript(raw, "raw_extras."+trimmedKey, name, controlPath, diagnostics)
			}
			continue
		}
		block, blockOK := raw.(map[string]any)
		if !blockOK || len(block) == 0 {
			addUIPanelDiagnostic(diagnostics, "error", "panel.raw_extras_value", "control raw_extras supports only scalars or one non-empty level of scalar fields", name, controlPath)
			continue
		}
		seenChildKeys := map[string]string{}
		for childKey, childValue := range block {
			trimmedChildKey := strings.TrimSpace(childKey)
			foldedChildKey := strings.ToLower(trimmedChildKey)
			if previous, duplicate := seenChildKeys[foldedChildKey]; duplicate {
				addUIPanelDiagnostic(diagnostics, "error", "panel.raw_extras_key_duplicate", fmt.Sprintf("nested raw_extras keys %q and %q differ only by case; the exporter/engine cannot resolve them deterministically", previous, childKey), name, controlPath)
			} else {
				seenChildKeys[foldedChildKey] = childKey
			}
			if err := validateUIPanelIdentifier(trimmedChildKey, "raw_extras nested key", false); err != nil || !uiPanelRawScalar(childValue) {
				addUIPanelDiagnostic(diagnostics, "error", "panel.raw_extras_value", "control raw_extras supports only one non-empty level of scalar fields", name, controlPath)
				continue
			}
			if _, isText := childValue.(string); isText {
				validateUIPanelScript(childValue, "raw_extras."+trimmedKey+"."+trimmedChildKey, name, controlPath, diagnostics)
			}
		}
	}
	return extras
}

func validateUIPanelStringMap(value any, field, name, controlPath string, diagnostics *[]uiPanelDiagnostic) {
	if value == nil {
		return
	}
	values, ok := value.(map[string]any)
	if !ok {
		addUIPanelDiagnostic(diagnostics, "error", "panel.control_"+field, field+" must be an object of locale to text", name, controlPath)
		return
	}
	for locale, raw := range values {
		if strings.TrimSpace(locale) == "" {
			addUIPanelDiagnostic(diagnostics, "error", "panel.control_"+field, field+" locale keys cannot be empty", name, controlPath)
		}
		if _, ok := raw.(string); !ok {
			addUIPanelDiagnostic(diagnostics, "error", "panel.control_"+field, field+" values must be strings", name, controlPath)
		}
	}
}

func normalizeUIPanelTextLocales(control map[string]any, name, path string, diagnostics *[]uiPanelDiagnostic) {
	values, ok := control["text"].(map[string]any)
	if !ok {
		return
	}
	if len(values) == 0 {
		addUIPanelDiagnostic(diagnostics, "error", "panel.control_text", "a text locale object must contain at least one locale; use text_locs for multilingual text", name, path)
		return
	}
	for locale, raw := range values {
		text, ok := raw.(string)
		if strings.TrimSpace(locale) == "" || !ok || len(text) > 200_000 || strings.ContainsRune(text, '\x00') {
			addUIPanelDiagnostic(diagnostics, "error", "panel.control_text", "a text locale object must have nonempty locale keys and string values of at most 200000 bytes; use text_locs for multilingual text", name, path)
			return
		}
	}
	merged := map[string]any{}
	if raw := control["text_locs"]; raw != nil {
		explicit, ok := raw.(map[string]any)
		if !ok {
			return
		}
		for locale, value := range explicit {
			merged[locale] = value
		}
	}
	for locale, text := range values {
		if previous, exists := merged[locale]; exists && previous != text {
			addUIPanelDiagnostic(diagnostics, "error", "panel.control_text_locs_conflict", "text and text_locs contain different values for locale "+locale+"; supply one value per locale", name, path)
			return
		}
		merged[locale] = text
	}
	control["text_locs"] = merged
	delete(control, "text")
}

func validateUIPanelScript(value any, field, name, controlPath string, diagnostics *[]uiPanelDiagnostic) {
	if value == nil {
		return
	}
	text, ok := value.(string)
	if !ok || len(text) > 200_000 || strings.ContainsRune(text, '\x00') {
		addUIPanelDiagnostic(diagnostics, "error", "panel.control_"+field, field+" must be script text of at most 200000 bytes", name, controlPath)
		return
	}
	depth := 0
	quoted := false
	escaped := false
	comment := false
	for _, char := range text {

		if comment {
			comment = char != '\n'
			continue
		}
		if quoted {
			if escaped {
				escaped = false
			} else if char == '\\' {
				escaped = true
			} else if char == '"' {
				quoted = false
			}
			continue
		}
		switch char {
		case '#':
			comment = true
		case '"':
			quoted = true
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				addUIPanelDiagnostic(diagnostics, "error", "panel.control_"+field, field+" contains unbalanced braces", name, controlPath)
				return
			}
		}
	}
	if quoted || depth != 0 {
		addUIPanelDiagnostic(diagnostics, "error", "panel.control_"+field, field+" contains an unclosed quote or unbalanced braces", name, controlPath)
	}
}

func normalizeUIPanelControls(controls []map[string]any) ([]map[string]any, []uiPanelDiagnostic) {
	controls = cloneJSONMaps(controls)
	diagnostics := []uiPanelDiagnostic{}
	seen := map[string]string{}
	count := 0

	var walk func([]map[string]any, []string, int) []map[string]any
	walk = func(rows []map[string]any, parents []string, depth int) []map[string]any {
		out := make([]map[string]any, 0, len(rows))
		for _, source := range rows {
			control := cloneJSONMap(source)
			count++
			kind := strings.ToLower(strings.TrimSpace(uiPanelString(control["kind"])))
			name := strings.TrimSpace(uiPanelString(control["name"]))
			control["kind"] = kind
			control["name"] = name
			pathParts := append(append([]string(nil), parents...), name)
			path := strings.Join(pathParts, "/")
			if name == "" {
				path = strings.Join(append(append([]string(nil), parents...), "<unnamed>"), "/")
			}
			if depth > maxUIPanelDepth {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.control_depth", fmt.Sprintf("control nesting exceeds %d levels", maxUIPanelDepth), name, path)
			}
			if !uiPanelControlKinds[kind] {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.control_kind", "kind must be container, icon, text, progressbar, or button", name, path)
			}
			allowed := map[string]bool{}
			for key := range uiPanelCommonControlFields {
				allowed[key] = true
			}
			for key := range uiPanelKindControlFields[kind] {
				allowed[key] = true
			}
			for key := range control {
				if !allowed[key] {
					addUIPanelDiagnostic(&diagnostics, "error", "panel.control_field_unknown", fmt.Sprintf("field %q is not exported for %s controls; move engine-native GUI fields into raw_extras", key, kind), name, path)
				}
			}
			if err := validateUIPanelIdentifier(name, "control name", false); err != nil {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.control_name", err.Error(), name, path)
			} else if previous, duplicate := seen[name]; duplicate {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.control_name_duplicate", fmt.Sprintf("control name is already used at %s; names must be panel-wide unique", previous), name, path)
			} else {
				seen[name] = path
			}

			for _, axis := range []string{"x", "y"} {
				if raw, exists := control[axis]; exists {
					if n, ok := uiPanelIntegerValue(raw); !ok || n < -100000 || n > 100000 {
						addUIPanelDiagnostic(&diagnostics, "error", "panel.control_geometry", axis+" must be an integer between -100000 and 100000", name, path)
					} else {
						control[axis] = n
					}
				} else {
					control[axis] = 0
				}
			}
			for _, pair := range [][2]string{{"width", "w"}, {"height", "h"}} {
				if _, hasCanonical := control[pair[1]]; !hasCanonical {
					if alias, hasAlias := control[pair[0]]; hasAlias {
						control[pair[1]] = alias
					}
				}
				delete(control, pair[0])
			}
			for _, dimension := range []string{"w", "h"} {
				if raw, exists := control[dimension]; exists && raw != nil {
					if n, ok := uiPanelIntegerValue(raw); !ok || n <= 0 || n > 100000 {
						addUIPanelDiagnostic(&diagnostics, "error", "panel.control_geometry", dimension+" must be a positive integer no greater than 100000", name, path)
					} else {
						control[dimension] = n
					}
				}
			}
			if raw, exists := control["frame"]; exists && raw != nil {
				if n, ok := uiPanelIntegerValue(raw); !ok || n < 1 {
					addUIPanelDiagnostic(&diagnostics, "error", "panel.control_frame", "frame must be an integer of at least 1", name, path)
				} else {
					control["frame"] = n
				}
			}
			for _, flag := range []string{"scroll", "change_scope"} {
				if raw, exists := control[flag]; exists && raw != nil {
					if _, ok := raw.(bool); !ok {
						addUIPanelDiagnostic(&diagnostics, "error", "panel.control_"+flag, flag+" must be true or false", name, path)
					}
				}
			}
			for _, field := range []string{"columns", "slot_w", "slot_h"} {
				if raw, exists := control[field]; exists && raw != nil {
					if n, ok := uiPanelIntegerValue(raw); !ok || n < 1 || n > 10000 {
						addUIPanelDiagnostic(&diagnostics, "error", "panel.control_list", field+" must be a positive integer", name, path)
					} else {
						control[field] = n
					}
				}
			}
			if array := strings.TrimSpace(uiPanelString(control["array"])); array != "" && uiPanelVarInvalidRE.ReplaceAllString(array, "") != array {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.control_array", "array contains characters the exporter would silently remove", name, path)
			}
			if format, exists := control["format"]; exists {
				value := strings.ToLower(strings.TrimSpace(uiPanelString(format)))
				if value != "left" && value != "right" && value != "center" {
					addUIPanelDiagnostic(&diagnostics, "error", "panel.control_format", "format must be left, right, or center", name, path)
				} else {
					control["format"] = value
				}
			}
			if kind == "text" || kind == "button" {
				normalizeUIPanelTextLocales(control, name, path, &diagnostics)
			}
			for _, field := range []string{"text", "font", "sprite", "tooltip", "visible_trigger", "click_effect", "click_enabled_trigger", "frame_var", "image_expr", "array"} {
				if raw, exists := control[field]; exists && raw != nil {
					if value, ok := raw.(string); !ok || len(value) > 200_000 || strings.ContainsRune(value, '\x00') {
						addUIPanelDiagnostic(&diagnostics, "error", "panel.control_"+field, field+" must be a string of at most 200000 bytes", name, path)
					}
				}
			}
			validateUIPanelStringMap(control["text_locs"], "text_locs", name, path, &diagnostics)
			validateUIPanelStringMap(control["tooltip_locs"], "tooltip_locs", name, path, &diagnostics)
			for _, field := range []string{"visible_trigger", "click_effect", "click_enabled_trigger"} {
				validateUIPanelScript(control[field], field, name, path, &diagnostics)
			}
			if frameVar := strings.TrimSpace(uiPanelString(control["frame_var"])); frameVar != "" {
				clean := uiPanelVarInvalidRE.ReplaceAllString(frameVar, "")
				if clean != frameVar {
					addUIPanelDiagnostic(&diagnostics, "error", "panel.control_frame_var", "frame_var contains characters the exporter would silently remove", name, path)
				}
			}
			if imageExpr := strings.TrimSpace(uiPanelString(control["image_expr"])); imageExpr != "" && (len(imageExpr) > 1024 || strings.ContainsAny(imageExpr, "\r\n\x00\"{}")) {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.control_image_expr", "image_expr must be a single safe HOI4 image expression without quotes, braces, or newlines", name, path)
			}
			control["raw_extras"] = validateUIPanelRawExtras(control["raw_extras"], false, &diagnostics, name, path)
			if sprite := strings.TrimSpace(uiPanelString(control["sprite"])); sprite != "" && !strings.HasPrefix(sprite, "GFX_") {
				addUIPanelDiagnostic(&diagnostics, "warning", "panel.sprite_format", "sprite normally names a registered GFX_* sprite", name, path)
			}

			rawChildren, childrenPresent := control["children"]
			children, childrenOK := uiPanelMapSlice(rawChildren)
			if childrenPresent && !childrenOK {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.control_children", "children must be an array of control objects", name, path)
				children = []map[string]any{}
			}
			if kind != "container" && len(children) > 0 {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.control_children", "only container controls may have children", name, path)
			}
			if kind == "container" {
				control["children"] = walk(children, pathParts, depth+1)
			} else {
				delete(control, "children")
			}
			if (kind == "icon" || kind == "progressbar") && strings.TrimSpace(uiPanelString(control["sprite"])) == "" {
				addUIPanelDiagnostic(&diagnostics, "warning", "panel.sprite_empty", "icon/progressbar has no sprite and may be invisible", name, path)
			}
			if kind == "progressbar" && strings.TrimSpace(uiPanelString(control["frame_var"])) == "" && strings.TrimSpace(uiPanelString(control["image_expr"])) == "" {
				addUIPanelDiagnostic(&diagnostics, "warning", "panel.progressbar_unbound", "progressbar has neither frame_var nor image_expr; it will not reflect changing game state", name, path)
			}
			if strings.TrimSpace(uiPanelString(control["frame_var"])) != "" && strings.TrimSpace(uiPanelString(control["image_expr"])) != "" {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.property_binding_ambiguous", "set only one of frame_var or image_expr", name, path)
			}
			if kind == "button" && strings.TrimSpace(uiPanelString(control["click_effect"])) == "" {
				addUIPanelDiagnostic(&diagnostics, "warning", "panel.button_no_effect", "button has no click_effect", name, path)
			}
			out = append(out, control)
		}
		return out
	}

	normalized := walk(controls, nil, 1)
	if count > maxUIPanelControls {
		addUIPanelDiagnostic(&diagnostics, "error", "panel.control_count", fmt.Sprintf("panel has %d controls; maximum is %d", count, maxUIPanelControls), "", "")
	}
	return normalized, diagnostics
}

func validateUIPanelData(panel map[string]any) (map[string]any, []uiPanelDiagnostic) {
	panel = cloneJSONMap(panel)
	diagnostics := []uiPanelDiagnostic{}
	token := strings.TrimSpace(uiPanelString(panel["token"]))
	if err := validateUIPanelIdentifier(token, "token", false); err != nil {
		addUIPanelDiagnostic(&diagnostics, "error", "panel.token", err.Error(), "", "")
	}
	panel["token"] = token
	contextType := strings.TrimSpace(uiPanelString(panel["context_type"]))
	if contextType == "" {
		contextType = "decision_category"
	}
	if !uiPanelContexts[contextType] {
		allowed := make([]string, 0, len(uiPanelContexts))
		for value := range uiPanelContexts {
			allowed = append(allowed, value)
		}
		sort.Strings(allowed)
		addUIPanelDiagnostic(&diagnostics, "error", "panel.context_type", "context_type must be one of: "+strings.Join(allowed, ", "), "", "")
	}
	panel["context_type"] = contextType
	parent := strings.TrimSpace(uiPanelString(panel["parent_window_token"]))
	if err := validateUIPanelIdentifier(parent, "parent_window_token", true); err != nil {
		addUIPanelDiagnostic(&diagnostics, "error", "panel.parent_window_token", err.Error(), "", "")
	}
	panel["parent_window_token"] = parent
	validateUIPanelScript(panel["visible_trigger"], "visible_trigger", "", "", &diagnostics)
	panel["raw_extras"] = validateUIPanelRawExtras(panel["raw_extras"], true, &diagnostics, "", "")

	props, ok := panel["window_props"].(map[string]any)
	if panel["window_props"] == nil {
		props, ok = map[string]any{}, true
	}
	if !ok {
		addUIPanelDiagnostic(&diagnostics, "error", "panel.window_props", "window_props must be an object", "", "")
		props = map[string]any{}
	}
	props = cloneJSONMap(props)
	windowFields := map[string]bool{
		"x": true, "y": true, "width": true, "height": true,
		"orientation": true, "origo": true, "moveable": true, "background_sprite": true,
	}
	for field := range props {
		if !windowFields[field] {
			addUIPanelDiagnostic(&diagnostics, "error", "panel.window_field_unknown", fmt.Sprintf("window_props field %q is not exported", field), "", "")
		}
	}
	for _, field := range []string{"x", "y", "width", "height"} {
		if raw, exists := props[field]; exists && raw != nil {
			n, valid := uiPanelIntegerValue(raw)
			if !valid || ((field == "width" || field == "height") && n <= 0) || n < -100000 || n > 100000 {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.window_geometry", field+" must be an integer in range; width/height must be positive", "", "")
			} else {
				props[field] = n
			}
		}
	}
	for _, field := range []string{"orientation", "origo"} {
		if raw, exists := props[field]; exists && raw != nil {
			value, valid := raw.(string)
			value = strings.ToUpper(strings.TrimSpace(value))
			if !valid || (value != "" && !guiOverlayAnchors[value]) {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.window_anchor", field+" must be a canonical 3x3 anchor or empty", "", "")
			} else {
				props[field] = value
			}
		}
	}
	if raw, exists := props["moveable"]; exists && raw != nil {
		if _, valid := raw.(bool); !valid {
			addUIPanelDiagnostic(&diagnostics, "error", "panel.window_moveable", "moveable must be a boolean", "", "")
		}
	}
	if raw, exists := props["background_sprite"]; exists && raw != nil {
		value, valid := raw.(string)
		if !valid || len(value) > 512 || strings.ContainsAny(value, "\r\n\x00") {
			addUIPanelDiagnostic(&diagnostics, "error", "panel.window_background", "background_sprite must be a safe string", "", "")
		} else if value != "" && !strings.HasPrefix(value, "GFX_") {
			addUIPanelDiagnostic(&diagnostics, "warning", "panel.sprite_format", "background_sprite normally names a registered GFX_* sprite", "", "")
		}
	}
	panel["window_props"] = props
	controls, ok := uiPanelMapSlice(panel["controls"])
	if !ok {
		addUIPanelDiagnostic(&diagnostics, "error", "panel.controls", "controls must be an array of control objects", "", "")
		controls = []map[string]any{}
	}
	controls, controlDiagnostics := normalizeUIPanelControls(controls)
	panel["controls"] = controls
	diagnostics = append(diagnostics, controlDiagnostics...)
	return panel, diagnostics
}

func diagnosticsHaveErrors(diagnostics []uiPanelDiagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			return true
		}
	}
	return false
}

func diagnosticsError(prefix string, diagnostics []uiPanelDiagnostic) error {
	messages := []string{}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			message := diagnostic.Code + ": " + diagnostic.Message
			if diagnostic.ControlPath != "" {
				message += " (" + diagnostic.ControlPath + ")"
			}
			messages = append(messages, message)
		}
		if len(messages) == 8 {
			break
		}
	}
	return fmt.Errorf("%s: %s", prefix, strings.Join(messages, "; "))
}

func uiPanelSpriteReferences(panel map[string]any) map[string]string {
	refs := map[string]string{}
	if props, ok := panel["window_props"].(map[string]any); ok {
		if sprite := strings.TrimSpace(uiPanelString(props["background_sprite"])); sprite != "" {
			refs[sprite] = "window background"
		}
	}
	controls, _ := uiPanelMapSlice(panel["controls"])
	var walk func([]map[string]any, []string)
	walk = func(rows []map[string]any, parents []string) {
		for _, control := range rows {
			name := strings.TrimSpace(uiPanelString(control["name"]))
			path := strings.Join(append(append([]string(nil), parents...), name), "/")
			if sprite := strings.TrimSpace(uiPanelString(control["sprite"])); sprite != "" {
				refs[sprite] = path
			}
			children, _ := uiPanelMapSlice(control["children"])
			walk(children, append(append([]string(nil), parents...), name))
		}
	}
	walk(controls, nil)
	return refs
}

func validateUIPanelSpriteReferences(ctx context.Context, proxy *RESTProxy, projectID, panelID int64, panel map[string]any, diagnostics *[]uiPanelDiagnostic) bool {
	refs := uiPanelSpriteReferences(panel)
	if len(refs) == 0 {
		return true
	}
	names := make([]string, 0, len(refs))
	for name := range refs {
		names = append(names, name)
	}
	sort.Strings(names)
	resp, err := proxyAuthedCall(ctx, proxy, "GET", fmt.Sprintf("/api/projects/%d/ui-panels/%d/sprite-references", projectID, panelID), nil)
	if err != nil {
		addUIPanelDiagnostic(diagnostics, "error", "panel.sprite_check_failed", "could not verify referenced GFX sprites: "+err.Error(), "", "")
		return false
	}
	payload, _ := resp["sprites"].(map[string]any)
	valid := true
	for _, name := range names {
		meta, _ := payload[name].(map[string]any)
		known := boolField(meta, "resolved") || boolField(meta, "overridden") || boolField(meta, "procedural")
		if !known || boolField(meta, "missing_file") {
			valid = false
			addUIPanelDiagnostic(diagnostics, "error", "panel.sprite_unresolved", fmt.Sprintf("sprite %q is not resolvable in this project", name), "", refs[name])
		}
	}
	return valid
}

func validateUIPanelCompiledReadback(panel, files map[string]any, diagnostics *[]uiPanelDiagnostic) bool {
	guiText := stringField(files, "gui_text")
	txtText := stringField(files, "txt_text")
	token := strings.TrimSpace(uiPanelString(panel["token"]))
	valid := true
	controls, _ := uiPanelMapSlice(panel["controls"])
	var walk func([]map[string]any, []string)
	walk = func(rows []map[string]any, parents []string) {
		for _, control := range rows {
			shortName := strings.TrimSpace(uiPanelString(control["name"]))
			controlName := token + "_" + shortName
			controlPath := strings.Join(append(append([]string(nil), parents...), shortName), "/")
			if !uiPanelGUIHasName(guiText, controlName) {
				valid = false
				addUIPanelDiagnostic(diagnostics, "error", "panel.compile_control_missing", "compiled GUI omitted the control name "+controlName, shortName, controlPath)
			}
			if sprite := strings.TrimSpace(uiPanelString(control["sprite"])); sprite != "" && !strings.Contains(guiText, `"`+sprite+`"`) {
				valid = false
				addUIPanelDiagnostic(diagnostics, "error", "panel.compile_sprite_missing", "compiled GUI omitted sprite "+sprite, shortName, controlPath)
			}
			if strings.TrimSpace(uiPanelString(control["visible_trigger"])) != "" && !uiPanelScriptHasBlock(txtText, controlName+"_visible") {
				valid = false
				addUIPanelDiagnostic(diagnostics, "error", "panel.compile_trigger_missing", "compiled scripted_gui omitted the control visibility trigger", shortName, controlPath)
			}
			kind := strings.ToLower(strings.TrimSpace(uiPanelString(control["kind"])))
			if kind == "button" && !uiPanelScriptHasBlock(txtText, controlName+"_click") {
				valid = false
				addUIPanelDiagnostic(diagnostics, "error", "panel.compile_effect_missing", "compiled scripted_gui omitted the button click effect", shortName, controlPath)
			}
			if (strings.TrimSpace(uiPanelString(control["frame_var"])) != "" || strings.TrimSpace(uiPanelString(control["image_expr"])) != "") && !uiPanelScriptHasBlock(txtText, controlName) {
				valid = false
				addUIPanelDiagnostic(diagnostics, "error", "panel.compile_property_missing", "compiled scripted_gui omitted the control property binding", shortName, controlPath)
			}
			children, _ := uiPanelMapSlice(control["children"])
			walk(children, append(append([]string(nil), parents...), shortName))
		}
	}
	walk(controls, nil)
	return valid
}

func uiPanelGUIHasName(guiText, name string) bool {
	return regexp.MustCompile(`(^|[\s{])name\s*=\s*"` + regexp.QuoteMeta(name) + `"`).MatchString(guiText)
}

func uiPanelScriptHasBlock(txtText, key string) bool {
	return regexp.MustCompile(`(^|[\s{])` + regexp.QuoteMeta(key) + `\s*=\s*\{`).MatchString(txtText)
}

func requireExpectedUpdatedAt(expected string, current map[string]any) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return fmt.Errorf("expected_updated_at is required; call get_ui_panel and pass its exact updated_at value")
	}
	actual := stringField(current, "updated_at")
	if actual == "" {
		return fmt.Errorf("ui-panels backend did not return updated_at, so a safe partial write cannot be attempted")
	}
	if expected != actual {
		return fmt.Errorf("stale ui-panel snapshot: expected_updated_at=%q but current updated_at=%q; re-read with get_ui_panel and re-plan the edit", expected, actual)
	}
	return nil
}

func mergeUIPanelObject(saved any, patch map[string]any) map[string]any {
	out := map[string]any{}
	if current, ok := saved.(map[string]any); ok {
		for key, value := range current {
			out[key] = value
		}
	}
	for key, value := range patch {
		if value == nil {
			delete(out, key)
			continue
		}
		out[key] = value
	}
	return out
}

func uiPanelMutationError(err error) error {
	var restErr *RESTError
	if errors.As(err, &restErr) && restErr.Status == 409 {

		if strings.Contains(restErr.Detail, "stale ui-panel revision") {
			return fmt.Errorf("ui_panel_revision_conflict: %s; call get_ui_panel again and re-plan the edit instead of replaying the stale request", restErr.Detail)
		}
		return fmt.Errorf("ui-panel write refused: %s; this is not a stale revision, so re-reading will not help — change the request (for example choose an unused token)", restErr.Detail)
	}
	return err
}

type uiPanelControlNode struct {
	Fields   map[string]any
	Children []*uiPanelControlNode
}

func uiPanelNodes(rows []map[string]any) []*uiPanelControlNode {
	out := make([]*uiPanelControlNode, 0, len(rows))
	for _, row := range rows {
		fields := cloneJSONMap(row)
		children, ok := uiPanelMapSlice(fields["children"])
		if ok || fields["children"] == nil {
			delete(fields, "children")
		}

		out = append(out, &uiPanelControlNode{Fields: fields, Children: uiPanelNodes(children)})
	}
	return out
}

func uiPanelNodeMaps(nodes []*uiPanelControlNode) []map[string]any {
	out := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		row := cloneJSONMap(node.Fields)

		if strings.EqualFold(strings.TrimSpace(uiPanelString(row["kind"])), "container") || len(node.Children) > 0 {
			row["children"] = uiPanelNodeMaps(node.Children)
		}
		out = append(out, row)
	}
	return out
}

type uiPanelControlLocation struct {
	Node   *uiPanelControlNode
	Parent *uiPanelControlNode
	Index  int
	Path   string
}

func findUIPanelControls(nodes []*uiPanelControlNode, name, path string) []uiPanelControlLocation {
	name = strings.TrimSpace(name)
	path = strings.Trim(strings.TrimSpace(path), "/")
	matches := []uiPanelControlLocation{}
	var walk func([]*uiPanelControlNode, *uiPanelControlNode, []string)
	walk = func(rows []*uiPanelControlNode, parent *uiPanelControlNode, parents []string) {
		for index, node := range rows {
			nodeName := strings.TrimSpace(uiPanelString(node.Fields["name"]))
			nodePath := strings.Join(append(append([]string(nil), parents...), nodeName), "/")
			nameMatches := name == "" || nodeName == name
			pathMatches := path == "" || nodePath == path
			if nameMatches && pathMatches && (name != "" || path != "") {
				matches = append(matches, uiPanelControlLocation{Node: node, Parent: parent, Index: index, Path: nodePath})
			}
			walk(node.Children, node, append(append([]string(nil), parents...), nodeName))
		}
	}
	walk(nodes, nil, nil)
	return matches
}

func resolveUIPanelControl(nodes []*uiPanelControlNode, name, path, field string) (uiPanelControlLocation, error) {
	if strings.TrimSpace(name) == "" && strings.Trim(strings.TrimSpace(path), "/") == "" {
		return uiPanelControlLocation{}, fmt.Errorf("%s requires control_name or control_path", field)
	}
	matches := findUIPanelControls(nodes, name, path)
	if len(matches) == 0 {
		return uiPanelControlLocation{}, fmt.Errorf("%s did not match a control (name=%q path=%q)", field, name, path)
	}
	if len(matches) > 1 {
		return uiPanelControlLocation{}, fmt.Errorf("%s matched %d controls; supply an exact control_path", field, len(matches))
	}
	return matches[0], nil
}

func uiPanelChildrenList(roots *[]*uiPanelControlNode, parent *uiPanelControlNode) *[]*uiPanelControlNode {
	if parent == nil {
		return roots
	}
	return &parent.Children
}

func insertUIPanelControl(list *[]*uiPanelControlNode, index *int, node *uiPanelControlNode) error {
	position := len(*list)
	if index != nil {
		position = *index
	}
	if position < 0 || position > len(*list) {
		return fmt.Errorf("index %d is outside the allowed final range 0..%d", position, len(*list))
	}
	*list = append(*list, nil)
	copy((*list)[position+1:], (*list)[position:])
	(*list)[position] = node
	return nil
}

func detachUIPanelControl(roots *[]*uiPanelControlNode, location uiPanelControlLocation) *uiPanelControlNode {
	list := uiPanelChildrenList(roots, location.Parent)
	node := (*list)[location.Index]
	copy((*list)[location.Index:], (*list)[location.Index+1:])
	*list = (*list)[:len(*list)-1]
	return node
}

func uiPanelNodeContains(root, candidate *uiPanelControlNode) bool {
	if root == candidate {
		return true
	}
	for _, child := range root.Children {
		if uiPanelNodeContains(child, candidate) {
			return true
		}
	}
	return false
}

func resolveUIPanelParent(nodes []*uiPanelControlNode, name, path string) (*uiPanelControlNode, error) {
	if strings.TrimSpace(name) == "" && strings.Trim(strings.TrimSpace(path), "/") == "" {
		return nil, nil
	}
	location, err := resolveUIPanelControl(nodes, name, path, "destination parent")
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(uiPanelString(location.Node.Fields["kind"])), "container") {
		return nil, fmt.Errorf("destination parent %q is not a container", location.Path)
	}
	return location.Node, nil
}

var uiPanelMutableControlFields = map[string]bool{
	"name": true, "x": true, "y": true, "w": true, "h": true,
	"width": true, "height": true, "sprite": true, "frame": true,
	"text": true, "text_locs": true, "visible_trigger": true,
	"frame_var": true, "image_expr": true, "tooltip": true,
	"tooltip_locs": true, "click_effect": true,
	"click_enabled_trigger": true, "font": true, "format": true,
	"raw_extras": true,
}

var uiPanelOperationFields = map[string][]string{
	"add":     {"parent_control_name", "parent_control_path", "index", "control"},
	"update":  {"control_name", "control_path", "fields"},
	"delete":  {"control_name", "control_path", "cascade"},
	"move":    {"control_name", "control_path", "parent_control_name", "parent_control_path", "index"},
	"reorder": {"control_name", "control_path", "index"},
}

func checkUIPanelOperationFields(at int, action string, op uiPanelOperation) error {
	allowed, known := uiPanelOperationFields[action]
	if !known {
		return nil
	}
	given := []struct {
		name    string
		present bool
	}{
		{"control_name", strings.TrimSpace(op.ControlName) != ""},
		{"control_path", strings.TrimSpace(op.ControlPath) != ""},
		{"parent_control_name", strings.TrimSpace(op.ParentControlName) != ""},
		{"parent_control_path", strings.TrimSpace(op.ParentControlPath) != ""},
		{"index", op.Index != nil},
		{"control", len(op.Control) > 0},
		{"fields", len(op.Fields) > 0},
		{"cascade", op.Cascade},
	}
	for _, field := range given {
		if field.present && !slices.Contains(allowed, field.name) {
			hint := ""
			if field.name == "fields" {
				hint = "; change layout or other fields with a separate update operation"
			}
			return fmt.Errorf("operations[%d] %s does not take %s%s", at, action, field.name, hint)
		}
	}
	return nil
}

func applyUIPanelControlOperations(
	controls []map[string]any, operations []uiPanelOperation,
) ([]map[string]any, []uiPanelOperationResult, error) {
	if len(operations) == 0 {
		return nil, nil, fmt.Errorf("operations must contain at least one operation")
	}
	if len(operations) > maxUIPanelOperations {
		return nil, nil, fmt.Errorf("operations contains %d entries; maximum is %d", len(operations), maxUIPanelOperations)
	}
	roots := uiPanelNodes(controls)
	results := make([]uiPanelOperationResult, 0, len(operations))
	for at, operation := range operations {
		action := strings.ToLower(strings.TrimSpace(operation.Operation))
		result := uiPanelOperationResult{Operation: action, ControlName: strings.TrimSpace(operation.ControlName)}
		if err := checkUIPanelOperationFields(at, action, operation); err != nil {
			return nil, nil, err
		}
		switch action {
		case "add":
			if len(operation.Control) == 0 {
				return nil, nil, fmt.Errorf("operations[%d] add requires control", at)
			}
			control := cloneJSONMap(operation.Control)
			name := strings.TrimSpace(uiPanelString(control["name"]))
			if operation.ControlName != "" && operation.ControlName != name {
				return nil, nil, fmt.Errorf("operations[%d] control_name must equal control.name for add", at)
			}
			if _, err := resolveUIPanelControl(roots, name, "", "new control duplicate check"); err == nil {
				return nil, nil, fmt.Errorf("operations[%d] cannot add duplicate control name %q", at, name)
			}
			parent, err := resolveUIPanelParent(roots, operation.ParentControlName, operation.ParentControlPath)
			if err != nil {
				return nil, nil, fmt.Errorf("operations[%d]: %w", at, err)
			}
			nodeRows, ok := uiPanelMapSlice([]any{control})
			if !ok || len(nodeRows) != 1 {
				return nil, nil, fmt.Errorf("operations[%d] control is invalid", at)
			}
			node := uiPanelNodes(nodeRows)[0]
			if err := insertUIPanelControl(uiPanelChildrenList(&roots, parent), operation.Index, node); err != nil {
				return nil, nil, fmt.Errorf("operations[%d]: %w", at, err)
			}
			result.ControlName = name
			if added, err := resolveUIPanelControl(roots, name, "", "added control"); err == nil {
				result.AfterPath = added.Path
			}

		case "update":
			location, err := resolveUIPanelControl(roots, operation.ControlName, operation.ControlPath, fmt.Sprintf("operations[%d] update", at))
			if err != nil {
				return nil, nil, err
			}
			if len(operation.Fields) == 0 {
				return nil, nil, fmt.Errorf("operations[%d] update requires at least one field", at)
			}
			for field, value := range operation.Fields {
				if !uiPanelMutableControlFields[field] {
					return nil, nil, fmt.Errorf("operations[%d] cannot update field %q; kind/children replacement is forbidden", at, field)
				}
				canonical := field
				if field == "width" {
					canonical = "w"
				}
				if field == "height" {
					canonical = "h"
				}
				if value == nil {
					delete(location.Node.Fields, canonical)
				} else {
					location.Node.Fields[canonical] = value
				}
			}
			result.BeforePath = location.Path
			result.ControlName = strings.TrimSpace(uiPanelString(location.Node.Fields["name"]))
			if updated, err := resolveUIPanelControl(roots, result.ControlName, "", "updated control"); err == nil {
				result.AfterPath = updated.Path
			}

		case "delete":
			location, err := resolveUIPanelControl(roots, operation.ControlName, operation.ControlPath, fmt.Sprintf("operations[%d] delete", at))
			if err != nil {
				return nil, nil, err
			}
			if len(location.Node.Children) > 0 && !operation.Cascade {
				return nil, nil, fmt.Errorf("operations[%d] refuses to delete non-empty container %q without cascade=true", at, location.Path)
			}
			detachUIPanelControl(&roots, location)
			result.ControlName = strings.TrimSpace(uiPanelString(location.Node.Fields["name"]))
			result.BeforePath = location.Path

		case "move":
			location, err := resolveUIPanelControl(roots, operation.ControlName, operation.ControlPath, fmt.Sprintf("operations[%d] move", at))
			if err != nil {
				return nil, nil, err
			}
			parent, err := resolveUIPanelParent(roots, operation.ParentControlName, operation.ParentControlPath)
			if err != nil {
				return nil, nil, fmt.Errorf("operations[%d]: %w", at, err)
			}
			if parent != nil && uiPanelNodeContains(location.Node, parent) {
				return nil, nil, fmt.Errorf("operations[%d] cannot move a control into itself or its descendant", at)
			}
			node := detachUIPanelControl(&roots, location)
			if err := insertUIPanelControl(uiPanelChildrenList(&roots, parent), operation.Index, node); err != nil {
				return nil, nil, fmt.Errorf("operations[%d]: %w", at, err)
			}
			result.ControlName = strings.TrimSpace(uiPanelString(node.Fields["name"]))
			result.BeforePath = location.Path
			if moved, err := resolveUIPanelControl(roots, result.ControlName, "", "moved control"); err == nil {
				result.AfterPath = moved.Path
			}

		case "reorder":
			if operation.Index == nil {
				return nil, nil, fmt.Errorf("operations[%d] reorder requires index", at)
			}
			if operation.ParentControlName != "" || operation.ParentControlPath != "" {
				return nil, nil, fmt.Errorf("operations[%d] reorder keeps the current parent; use move to change parents", at)
			}
			location, err := resolveUIPanelControl(roots, operation.ControlName, operation.ControlPath, fmt.Sprintf("operations[%d] reorder", at))
			if err != nil {
				return nil, nil, err
			}
			parent := location.Parent
			node := detachUIPanelControl(&roots, location)
			if err := insertUIPanelControl(uiPanelChildrenList(&roots, parent), operation.Index, node); err != nil {
				return nil, nil, fmt.Errorf("operations[%d]: %w", at, err)
			}
			result.ControlName = strings.TrimSpace(uiPanelString(node.Fields["name"]))
			result.BeforePath = location.Path
			if reordered, err := resolveUIPanelControl(roots, result.ControlName, "", "reordered control"); err == nil {
				result.AfterPath = reordered.Path
			}

		default:
			return nil, nil, fmt.Errorf("operations[%d].operation must be add, update, delete, move, or reorder", at)
		}
		results = append(results, result)
	}
	return uiPanelNodeMaps(roots), results, nil
}

func registerUIPanelTools(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "get_ui_panel",
		Description: "Read one ui-panel with every persisted field, the complete controls tree, derived names, backend warnings, updated_at and explicit capability gaps. Read this immediately before update_ui_panel or apply_ui_panel_operations.",
	}, func(ctx context.Context, _ *CallToolRequest, in getUIPanelIn) (*CallToolResult, map[string]any, error) {
		if err := validateUIPanelID(in.ProjectID, in.PanelID); err != nil {
			return nil, nil, err
		}
		resp, err := proxyAuthedCall(ctx, proxy, "GET", uiPanelPath(in.ProjectID, in.PanelID), nil)
		if err != nil {
			return nil, nil, err
		}
		return nil, uiPanelReadback(resp, in.ProjectID), nil
	})

	AddTool(s, &Tool{
		Name:        "update_ui_panel",
		Description: "Partially update panel-level fields without accepting/replacing controls. Omitted fields stay unchanged. Requires the exact updated_at from get_ui_panel and returns a full readback plus diagnostics. Use apply_ui_panel_operations for controls. Renaming token re-derives every emitted window/file/loc name, so inspect get_ui_panel_files afterward.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateUIPanelIn) (*CallToolResult, map[string]any, error) {
		if err := validateUIPanelID(in.ProjectID, in.PanelID); err != nil {
			return nil, nil, err
		}
		current, err := proxyAuthedCall(ctx, proxy, "GET", uiPanelPath(in.ProjectID, in.PanelID), nil)
		if err != nil {
			return nil, nil, err
		}
		if err := requireExpectedUpdatedAt(in.ExpectedUpdatedAt, current); err != nil {
			return nil, nil, err
		}
		body := map[string]any{}
		set(body, "token", in.Token)
		set(body, "country_tag", in.CountryTag)
		set(body, "name_en", in.NameEn)
		set(body, "name_zh", in.NameZh)

		if in.DisplayNameLocs != nil {
			body["display_name_locs"] = mergeUIPanelObject(current["display_name_locs"], copyStringMap(*in.DisplayNameLocs))
		}
		set(body, "context_type", in.ContextType)
		set(body, "parent_window_token", in.ParentWindowToken)
		set(body, "visible_trigger", in.VisibleTrigger)
		if in.WindowProps != nil {
			body["window_props"] = mergeUIPanelObject(current["window_props"], *in.WindowProps)
		}
		if in.RawExtras != nil {
			body["raw_extras"] = mergeUIPanelObject(current["raw_extras"], *in.RawExtras)
		}
		if len(body) == 0 {
			return nil, nil, fmt.Errorf("update_ui_panel was called with no fields to change")
		}

		body["expected_updated_at"] = in.ExpectedUpdatedAt
		candidate := cloneJSONMap(current)
		for key, value := range body {
			candidate[key] = value
		}
		normalizedPanel, diagnostics := validateUIPanelData(candidate)
		if diagnosticsHaveErrors(diagnostics) {
			return nil, nil, diagnosticsError("ui-panel update rejected", diagnostics)
		}

		for key := range body {
			if value, ok := normalizedPanel[key]; ok && key != "expected_updated_at" {
				body[key] = value
			}
		}
		resp, err := proxyAuthedCall(ctx, proxy, "PUT", uiPanelPath(in.ProjectID, in.PanelID), body)
		if err != nil {
			return nil, nil, uiPanelMutationError(err)
		}
		readback := uiPanelReadback(resp, in.ProjectID)
		return nil, map[string]any{
			"project_id": in.ProjectID, "panel_id": in.PanelID,
			"token": stringField(resp, "token"), "readback": readback,
			"diagnostics":         diagnostics,
			"previous_updated_at": stringField(current, "updated_at"),
			"updated_at":          stringField(resp, "updated_at"),
			"capability_gaps":     append([]string(nil), uiPanelCapabilityGaps...),
		}, nil
	})

	AddTool(s, &Tool{
		Name:        "apply_ui_panel_operations",
		Description: "Atomically plan add/update/delete/move/reorder edits against the current controls tree, validate the final tree, then persist it with one revision-guarded PUT. Select targets by control_name or exact control_path. Never accepts a replacement controls tree. Requires expected_updated_at and returns ordered operation results, full readback and diagnostics.",
	}, func(ctx context.Context, _ *CallToolRequest, in applyUIPanelOperationsIn) (*CallToolResult, map[string]any, error) {
		if err := validateUIPanelID(in.ProjectID, in.PanelID); err != nil {
			return nil, nil, err
		}
		if len(in.Operations) == 0 || len(in.Operations) > maxUIPanelOperations {
			return nil, nil, fmt.Errorf("operations must contain 1..%d entries", maxUIPanelOperations)
		}
		current, err := proxyAuthedCall(ctx, proxy, "GET", uiPanelPath(in.ProjectID, in.PanelID), nil)
		if err != nil {
			return nil, nil, err
		}
		if err := requireExpectedUpdatedAt(in.ExpectedUpdatedAt, current); err != nil {
			return nil, nil, err
		}
		controls, ok := uiPanelMapSlice(current["controls"])
		if !ok {
			return nil, nil, fmt.Errorf("ui-panels backend returned a non-array controls value; refusing to overwrite it")
		}
		controls, operationResults, err := applyUIPanelControlOperations(controls, in.Operations)
		if err != nil {
			return nil, nil, err
		}
		candidate := cloneJSONMap(current)
		candidate["controls"] = controls
		normalized, diagnostics := validateUIPanelData(candidate)
		if diagnosticsHaveErrors(diagnostics) {
			return nil, nil, diagnosticsError("ui-panel operations rejected", diagnostics)
		}
		normalizedControls, _ := uiPanelMapSlice(normalized["controls"])
		resp, err := proxyAuthedCall(ctx, proxy, "PUT", uiPanelPath(in.ProjectID, in.PanelID), map[string]any{
			"controls":            normalizedControls,
			"expected_updated_at": in.ExpectedUpdatedAt,
		})
		if err != nil {
			return nil, nil, uiPanelMutationError(err)
		}
		readback := uiPanelReadback(resp, in.ProjectID)
		return nil, map[string]any{
			"project_id": in.ProjectID, "panel_id": in.PanelID,
			"token":              stringField(resp, "token"),
			"applied_operations": operationResults,
			"applied_count":      len(operationResults),
			"readback":           readback, "diagnostics": diagnostics,
			"previous_updated_at": stringField(current, "updated_at"),
			"updated_at":          stringField(resp, "updated_at"),
			"capability_gaps":     append([]string(nil), uiPanelCapabilityGaps...),
		}, nil
	})

	AddTool(s, &Tool{
		Name:        "get_ui_panel_files",
		Description: "Compile and return the exact interface .gui, scripted_gui .txt and per-language localisation text exported for one ui-panel. Use this after every write to inspect the real emitted files rather than assuming the structured save is sufficient.",
	}, func(ctx context.Context, _ *CallToolRequest, in getUIPanelIn) (*CallToolResult, map[string]any, error) {
		if err := validateUIPanelID(in.ProjectID, in.PanelID); err != nil {
			return nil, nil, err
		}
		resp, err := proxyAuthedCall(ctx, proxy, "GET", uiPanelPath(in.ProjectID, in.PanelID)+"/files", nil)
		if err != nil {
			return nil, nil, err
		}
		out := cloneJSONMap(resp)
		out["project_id"] = in.ProjectID
		out["panel_id"] = in.PanelID
		out["checksums"] = uiPanelFileChecksums(resp)
		return nil, out, nil
	})

	AddTool(s, &Tool{
		Name:        "validate_ui_panel",
		Description: "Validate one panel's strict structured schema, script snippets and project GFX references, then call the production compile endpoint and read back every control/binding from the emitted files. Returns export validity separately from the explicit mount/runtime capability gap; it never claims in-game verification.",
	}, func(ctx context.Context, _ *CallToolRequest, in getUIPanelIn) (*CallToolResult, map[string]any, error) {
		if err := validateUIPanelID(in.ProjectID, in.PanelID); err != nil {
			return nil, nil, err
		}
		panel, err := proxyAuthedCall(ctx, proxy, "GET", uiPanelPath(in.ProjectID, in.PanelID), nil)
		if err != nil {
			return nil, nil, err
		}
		normalized, diagnostics := validateUIPanelData(panel)
		structurallyValid := !diagnosticsHaveErrors(diagnostics)
		referencesValid := validateUIPanelSpriteReferences(ctx, proxy, in.ProjectID, in.PanelID, normalized, &diagnostics)
		compiled := false
		compiledReadbackValid := false
		filesSummary := map[string]any{}
		files, compileErr := proxyAuthedCall(ctx, proxy, "GET", uiPanelPath(in.ProjectID, in.PanelID)+"/files", nil)
		if compileErr != nil {
			addUIPanelDiagnostic(&diagnostics, "error", "panel.compile_failed", compileErr.Error(), "", "")
		} else {
			guiText := stringField(files, "gui_text")
			txtText := stringField(files, "txt_text")
			guiPath := stringField(files, "gui_path")
			txtPath := stringField(files, "txt_path")
			windowName := stringField(files, "window_name")
			compiled = guiText != "" && txtText != "" && guiPath != "" && txtPath != ""
			if !compiled {
				addUIPanelDiagnostic(&diagnostics, "error", "panel.compile_incomplete", "compile endpoint returned an empty path or file body", "", "")
			}
			if windowName != "" && (!strings.Contains(guiText, windowName) || !strings.Contains(txtText, windowName)) {
				compiled = false
				addUIPanelDiagnostic(&diagnostics, "error", "panel.window_contract", "derived window_name is not present in both compiled GUI and scripted_gui text", "", "")
			}
			compiledReadbackValid = validateUIPanelCompiledReadback(normalized, files, &diagnostics)
			compiled = compiled && compiledReadbackValid
			filesSummary = map[string]any{
				"window_name": windowName, "gui_path": guiPath, "txt_path": txtPath,
				"loc_paths": files["loc_paths"], "gui_bytes": len(guiText), "txt_bytes": len(txtText),
			}
		}
		valid := structurallyValid && referencesValid && compiled && !diagnosticsHaveErrors(diagnostics)
		status := "export_valid_with_runtime_gap"
		if !valid {
			status = "failed"
		}
		return nil, map[string]any{
			"project_id": in.ProjectID, "panel_id": in.PanelID,
			"token": uiPanelString(normalized["token"]),
			"valid": valid, "verification_status": status,
			"structurally_valid": structurallyValid,
			"references_valid":   referencesValid,
			"compiled":           compiled, "compiled_readback_valid": compiledReadbackValid,
			"export_valid": valid, "runtime_verified": false,
			"diagnostics": diagnostics, "files": filesSummary,
			"warnings":        panel["warnings"],
			"capability_gaps": append([]string(nil), uiPanelCapabilityGaps...),
		}, nil
	})
}
