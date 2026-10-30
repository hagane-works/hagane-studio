package toolapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"haganeworks/internal/domains/worldmap"
)

const (
	guiLocalisationTargetKind = "gui_localisation"
	guiTextBatchLimit         = 100
	guiLocEndpointBatchLimit  = 64
	guiTextValueRuneLimit     = 8000
)

type getGUITextsIn struct {
	ProjectID int64    `json:"project_id"`
	Keys      []string `json:"keys" jsonschema:"1 to 100 exact static-GUI localisation target_key values (from get_gui_window); never guess keys or pass structured entity keys"`
	Language  string   `json:"language,omitempty" jsonschema:"English only; omit or use en"`
}

type guiTextReadResult struct {
	Key           string  `json:"key"`
	TargetKind    string  `json:"target_kind"`
	Found         bool    `json:"found"`
	Resolved      bool    `json:"resolved"`
	Overridden    bool    `json:"overridden"`
	Value         string  `json:"value"`
	ResolvedValue string  `json:"resolved_value"`
	OverrideValue *string `json:"override_value,omitempty"`
}

type getGUITextsOut struct {
	ProjectID    int64               `json:"project_id"`
	TargetKind   string              `json:"target_kind"`
	Language     string              `json:"language"`
	ResourceFile string              `json:"resource_file"`
	Texts        []guiTextReadResult `json:"texts"`
}

type guiTextUpdate struct {
	Key       string  `json:"key" jsonschema:"exact static-GUI localisation target_key (from get_gui_window); never guess it or pass a structured entity key"`
	Operation string  `json:"operation" jsonschema:"set_value, set_color, or clear"`
	Value     *string `json:"value,omitempty" jsonschema:"required only for set_value; complete localisation markup"`
	Color     string  `json:"color,omitempty" jsonschema:"required only for set_color; one HOI4 code such as R or §R"`
}

type updateGUITextsIn struct {
	ProjectID int64           `json:"project_id"`
	Updates   []guiTextUpdate `json:"updates" jsonschema:"1 to 100 exact-key localisation updates in stable order"`
	Language  string          `json:"language,omitempty" jsonschema:"English only; omit or use en"`
}

type guiTextUpdateResult struct {
	Index              int     `json:"index"`
	Key                string  `json:"key"`
	TargetKind         string  `json:"target_kind"`
	Operation          string  `json:"operation"`
	OK                 bool    `json:"ok"`
	WriteAccepted      bool    `json:"write_accepted"`
	ReadbackVerified   bool    `json:"readback_verified"`
	Changed            bool    `json:"changed"`
	Found              bool    `json:"found"`
	Overridden         bool    `json:"overridden"`
	PreviousValue      string  `json:"previous_value"`
	SavedValue         string  `json:"saved_value"`
	SavedOverrideValue *string `json:"saved_override_value,omitempty"`
	Error              string  `json:"error,omitempty"`
}

type updateGUITextsOut struct {
	ProjectID    int64                 `json:"project_id"`
	TargetKind   string                `json:"target_kind"`
	Language     string                `json:"language"`
	ResourceFile string                `json:"resource_file"`
	Requested    int                   `json:"requested"`
	Attempted    int                   `json:"attempted"`
	Applied      int                   `json:"applied"`
	Updated      int                   `json:"updated"`
	Failed       int                   `json:"failed"`
	Results      []guiTextUpdateResult `json:"results"`
}

type guiTextState struct {
	Effective map[string]string
	Overrides map[string]string
}

func guiTextResourceFile() string {
	_, file, ok := worldmap.GUITextTarget(defaultContentLocale)
	if !ok {
		return ""
	}
	return file
}

func registerGUITextTools(s *Server, proxy *RESTProxy) {
	registerGetGUITexts(s, proxy)
	registerUpdateGUITexts(s, proxy)
}

func registerGetGUITexts(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "get_gui_texts",
		Description: "Read 1-100 exact static .gui localisation targets from the owned project's rendered MOD layer, with HOI4 markup preserved. " +
			"Use only exact gui_localisation target_key values (get_gui_window lists a window's texts); never infer a key from visible prose, panel history, or a raw file. " +
			"The returned resource_file is the replace-localisation MOD resource behind the selection, not the .gui file that references the key. " +
			"This tool is not for national focuses, decisions, ideas, or other structured entity text; edit those with their entity tools.",
	}, func(ctx context.Context, _ *CallToolRequest, in getGUITextsIn) (*CallToolResult, getGUITextsOut, error) {
		keys, err := validateGUITextKeys(in.ProjectID, in.Language, in.Keys, false)
		if err != nil {
			return nil, getGUITextsOut{}, err
		}
		state, err := readGUITextState(ctx, proxy, in.ProjectID, keys)
		if err != nil {
			return nil, getGUITextsOut{}, err
		}
		out := getGUITextsOut{
			ProjectID: in.ProjectID, TargetKind: guiLocalisationTargetKind, Language: defaultContentLocale,
			ResourceFile: guiTextResourceFile(),
			Texts:        make([]guiTextReadResult, 0, len(keys)),
		}
		for _, key := range keys {
			value, found := state.Effective[key]
			override, overridden := state.Overrides[key]
			var overrideValue *string
			if overridden {
				overrideValue = &override
			}
			out.Texts = append(out.Texts, guiTextReadResult{
				Key: key, TargetKind: guiLocalisationTargetKind, Found: found, Resolved: found,
				Overridden: overridden, Value: value, ResolvedValue: value, OverrideValue: overrideValue,
			})
		}
		return nil, out, nil
	})
}

func registerUpdateGUITexts(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_gui_texts",
		Description: "Update 1-100 exact static .gui gui_localisation targets in the owned project's real English replace-localisation resource. " +
			"set_value saves complete markup; set_color recolours the entire current text while preserving its visible text, variables, icons, and line breaks; clear removes this override. " +
			"Every accepted write is read back from the game view's rendered project layer and reported per key. Use only exact target_key values; never guess keys. " +
			"The returned resource_file is the replace-localisation MOD resource actually written, not the .gui file that references the key. " +
			"This tool is not for national focuses, decisions, ideas, or other structured entity text; edit those with their entity tools.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateGUITextsIn) (*CallToolResult, updateGUITextsOut, error) {
		keys := make([]string, len(in.Updates))
		for index := range in.Updates {
			keys[index] = in.Updates[index].Key
		}
		keys, err := validateGUITextKeys(in.ProjectID, in.Language, keys, true)
		if err != nil {
			return nil, updateGUITextsOut{}, err
		}
		for index := range in.Updates {
			in.Updates[index].Key = keys[index]
			in.Updates[index].Operation = strings.ToLower(strings.TrimSpace(in.Updates[index].Operation))
			if err := validateGUITextUpdate(in.Updates[index]); err != nil {
				return nil, updateGUITextsOut{}, fmt.Errorf("updates[%d]: %w", index, err)
			}
		}

		before, err := readGUITextState(ctx, proxy, in.ProjectID, keys)
		if err != nil {
			return nil, updateGUITextsOut{}, err
		}
		out := updateGUITextsOut{
			ProjectID: in.ProjectID, TargetKind: guiLocalisationTargetKind, Language: defaultContentLocale,
			ResourceFile: guiTextResourceFile(),
			Requested:    len(in.Updates), Results: make([]guiTextUpdateResult, len(in.Updates)),
		}
		expected := make(map[string]string, len(in.Updates))
		written := make(map[string]bool, len(in.Updates))

		for index, update := range in.Updates {
			previous, found := before.Effective[update.Key]
			row := guiTextUpdateResult{
				Index: index, Key: update.Key, TargetKind: guiLocalisationTargetKind,
				Operation: update.Operation, Found: found, PreviousValue: previous,
			}
			out.Attempted++
			var response map[string]any
			switch update.Operation {
			case "set_value":
				expected[update.Key] = *update.Value
				response, err = proxyAuthedCall(ctx, proxy, http.MethodPost,
					fmt.Sprintf("/api/projects/%d/gui-render/text", in.ProjectID),
					map[string]any{"key": update.Key, "value": expected[update.Key], "lang": defaultContentLocale})
			case "set_color":
				if !found {
					err = fmt.Errorf("localisation key was not found in the current rendered project layer")
					break
				}
				expected[update.Key] = applyWholeGUITextColor(previous, update.Color)
				response, err = proxyAuthedCall(ctx, proxy, http.MethodPost,
					fmt.Sprintf("/api/projects/%d/gui-render/text", in.ProjectID),
					map[string]any{"key": update.Key, "value": expected[update.Key], "lang": defaultContentLocale})
			case "clear":
				query := url.Values{"key": {update.Key}, "lang": {defaultContentLocale}}
				response, err = proxyAuthedCall(ctx, proxy, http.MethodDelete,
					fmt.Sprintf("/api/projects/%d/gui-render/text?%s", in.ProjectID, query.Encode()), nil)
			}
			if err != nil {
				row.Error = err.Error()
				out.Results[index] = row
				continue
			}
			row.WriteAccepted = true
			row.Changed = update.Operation != "clear" || boolField(response, "removed")
			written[update.Key] = true
			out.Applied++
			out.Results[index] = row
		}

		if len(written) > 0 {
			afterKeys := make([]string, 0, len(written))
			for _, key := range keys {
				if written[key] {
					afterKeys = append(afterKeys, key)
				}
			}
			after, readErr := readGUITextState(ctx, proxy, in.ProjectID, afterKeys)
			for index, update := range in.Updates {
				if !written[update.Key] {
					continue
				}
				row := out.Results[index]
				if readErr != nil {
					row.Error = "write was accepted, but the game view readback could not be completed"
					out.Results[index] = row
					continue
				}
				row.SavedValue, row.Found = after.Effective[update.Key]
				overrideValue, overridden := after.Overrides[update.Key]
				row.Overridden = overridden
				if overridden {
					row.SavedOverrideValue = &overrideValue
				}
				switch update.Operation {
				case "set_value", "set_color":
					row.OK = overridden && overrideValue == expected[update.Key] && row.Found && row.SavedValue == expected[update.Key]
					if !row.OK {
						row.Error = "write was accepted, but saved override and rendered game view readback did not match the requested value"
					}
				case "clear":
					row.OK = !overridden
					if !row.OK {
						row.Error = "clear was accepted, but the project override is still present"
					}
				}
				row.ReadbackVerified = row.OK
				out.Results[index] = row
			}
		}

		for _, row := range out.Results {
			if row.OK {
				out.Updated++
			} else {
				out.Failed++
			}
		}
		return nil, out, nil
	})
}

func validateGUITextKeys(projectID int64, language string, raw []string, unique bool) ([]string, error) {
	if projectID <= 0 {
		return nil, fmt.Errorf("project_id must be greater than 0")
	}
	if lang := strings.ToLower(strings.TrimSpace(language)); lang != "" && lang != defaultContentLocale {
		return nil, fmt.Errorf("language must be en; GUI localisation currently resolves against the English rendered layer")
	}
	if len(raw) == 0 || len(raw) > guiTextBatchLimit {
		return nil, fmt.Errorf("keys must contain between 1 and %d exact localisation keys", guiTextBatchLimit)
	}
	keys := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, value := range raw {
		key := strings.TrimSpace(value)
		if !validGUISelectionLocKey(key) {
			return nil, fmt.Errorf("invalid localisation key %q; use an exact gui_localisation target_key from the current selection", key)
		}
		if seen[key] {
			if unique {
				return nil, fmt.Errorf("duplicate localisation key %q", key)
			}
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	return keys, nil
}

func validGUISelectionLocKey(key string) bool {
	if key == "" || len(key) > 128 {
		return false
	}
	for _, ch := range key {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9', ch == '_', ch == '.', ch == '-':
		default:
			return false
		}
	}
	return true
}

func validateGUITextUpdate(update guiTextUpdate) error {
	update.Operation = strings.ToLower(strings.TrimSpace(update.Operation))
	switch update.Operation {
	case "set_value":
		if update.Value == nil {
			return fmt.Errorf("set_value requires value")
		}
		if update.Color != "" {
			return fmt.Errorf("set_value does not accept color")
		}
		if utf8.RuneCountInString(*update.Value) > guiTextValueRuneLimit || strings.ContainsRune(*update.Value, '\x00') {
			return fmt.Errorf("value must contain at most %d characters and no NUL byte", guiTextValueRuneLimit)
		}
	case "set_color":
		if update.Value != nil {
			return fmt.Errorf("set_color preserves the current text and does not accept value")
		}
		if _, ok := normalizeGUITextColor(update.Color); !ok {
			return fmt.Errorf("color must be one HOI4 text colour code: C, L, W, B, G, R, b, g, Y, H, T, O, 0-9, or t")
		}
	case "clear":
		if update.Value != nil || strings.TrimSpace(update.Color) != "" {
			return fmt.Errorf("clear does not accept value or color")
		}
	default:
		return fmt.Errorf("operation must be set_value, set_color, or clear")
	}
	return nil
}

func normalizeGUITextColor(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "§")
	if utf8.RuneCountInString(value) != 1 {
		return "", false
	}
	if !strings.ContainsRune("CLWBGRbgYHTO0123456789t", []rune(value)[0]) {
		return "", false
	}
	return value, true
}

func applyWholeGUITextColor(value, rawColor string) string {
	color, _ := normalizeGUITextColor(rawColor)
	chars := []rune(value)
	var plain strings.Builder
	plain.Grow(len(value))
	for index := 0; index < len(chars); index++ {
		if chars[index] == '§' && index+1 < len(chars) {
			index++
			continue
		}
		plain.WriteRune(chars[index])
	}
	return "§" + color + plain.String() + "§!"
}

func readGUITextState(ctx context.Context, proxy *RESTProxy, projectID int64, keys []string) (guiTextState, error) {
	state := guiTextState{Effective: map[string]string{}, Overrides: map[string]string{}}
	for start := 0; start < len(keys); start += guiLocEndpointBatchLimit {
		end := start + guiLocEndpointBatchLimit
		if end > len(keys) {
			end = len(keys)
		}
		query := url.Values{"keys": {strings.Join(keys[start:end], ",")}, "markup": {"1"}}
		response, err := proxyAuthedCall(ctx, proxy, http.MethodGet,
			fmt.Sprintf("/api/projects/%d/gui-render/loc?%s", projectID, query.Encode()), nil)
		if err != nil {
			return guiTextState{}, err
		}
		for key, value := range guiTextStringMapField(response, "texts") {
			state.Effective[key] = value
		}
	}
	query := url.Values{"lang": {defaultContentLocale}}
	response, err := proxyAuthedCall(ctx, proxy, http.MethodGet,
		fmt.Sprintf("/api/projects/%d/gui-render/text?%s", projectID, query.Encode()), nil)
	if err != nil {
		return guiTextState{}, err
	}
	state.Overrides = guiTextStringMapField(response, "overrides")
	return state, nil
}

func guiTextStringMapField(value map[string]any, key string) map[string]string {
	out := map[string]string{}
	switch values := value[key].(type) {
	case map[string]any:
		for itemKey, raw := range values {
			if text, ok := raw.(string); ok {
				out[itemKey] = text
			}
		}
	case map[string]string:
		for itemKey, text := range values {
			out[itemKey] = text
		}
	}
	return out
}
