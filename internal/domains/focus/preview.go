package focus

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	previewemit "haganeworks/internal/hoi4/emit/preview"
	"haganeworks/internal/hoi4/importsvc"
	treeimport "haganeworks/internal/hoi4/importsvc/trees"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

var focusNodeUpdateFields = map[string]bool{
	"focus_id": true, "name": true, "icon": true, "cost": true,
	"x": true, "y": true, "relative_position_id": true,
	"prerequisites": true, "mutually_exclusive": true,
	"completion_reward": true, "available": true, "bypass": true,
	"cancel": true, "select_effect": true, "ai_will_do": true,
	"search_filters": true, "cancel_if_invalid": true,
	"continue_if_invalid": true, "available_if_capitulated": true,
	"will_lead_to_war_with": true, "allow_branch_ast": true,
	"load_focus_tree_rules": true, "name_en": true, "name_zh": true,
	"desc_en": true, "desc_zh": true, "display_name_locs": true,
	"description_locs": true, "effect_tooltip_locs": true,
	"raw_extras": true, "source": true, "offset": true,
	"dynamic_icon": true, "text_icon": true,
}

func focusPreviewDTO(tree FocusTree, nodes []FocusNode) *previewemit.FocusTree {
	var countryTag any
	if tree.CountryTag != nil {
		countryTag = *tree.CountryTag
	}
	treeData := map[string]any{
		"tree_id": tree.TreeID, "tree_kind": tree.TreeKind,
		"is_default": tree.IsDefault, "initial_show_position": tree.InitialShowPosition,
		"initial_show_focus":          tree.InitialShowFocus,
		"continuous_focus_position_x": tree.ContinuousFocusPositionX,
		"continuous_focus_position_y": tree.ContinuousFocusPositionY,
		"shared_focus_refs":           tree.SharedFocusRefs, "scope_kind": tree.ScopeKind,
		"country_tag": countryTag, "shared_country_tags": tree.SharedCountryTags,
	}
	nodeData := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		effectTooltips := make(map[string]any, len(node.EffectTooltipLocs))
		for key, value := range node.EffectTooltipLocs {
			effectTooltips[key] = value
		}
		nodeData = append(nodeData, map[string]any{
			"focus_id": node.FocusID, "icon": node.Icon,
			"dynamic_icon": node.DynamicIcon, "text_icon": node.TextIcon,
			"x": node.X, "y": node.Y,
			"relative_position_id": node.RelativePositionID, "offset": node.Offset,
			"cost": node.Cost, "prerequisites": node.Prerequisites,
			"mutually_exclusive": node.MutuallyExclusive,
			"allow_branch_ast":   node.AllowBranchAst, "available": node.Available,
			"bypass": node.Bypass, "cancel": node.Cancel,
			"raw_extras": node.RawExtras, "cancel_if_invalid": node.CancelIfInvalid,
			"continue_if_invalid":      node.ContinueIfInvalid,
			"available_if_capitulated": node.AvailableIfCapitulated,
			"will_lead_to_war_with":    node.WillLeadToWarWith,
			"search_filters":           node.SearchFilters, "select_effect": node.SelectEffect,
			"ai_will_do": node.AiWillDo, "completion_reward": node.CompletionReward,
			"effect_tooltip_locs": effectTooltips,
		})
	}
	return previewemit.BuildFocusTree(&importsvc.TreeBundle{
		TreeData: treeData, NodesData: nodeData, TreeKind: tree.TreeKind,
		SharedFocusRefs: tree.SharedFocusRefs,
	})
}

func previewTree(p *project.Project, w http.ResponseWriter, r *http.Request, treeID int64) error {
	if values, ok := r.URL.Query()["node_id"]; ok {
		raw := ""
		if len(values) > 0 {
			raw = values[len(values)-1]
		}
		nodeID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return previewValidationError("int_parsing", []any{"query", "node_id"},
				"Input should be a valid integer, unable to parse string as an integer", raw, nil)
		}
		if nodeID <= 0 || nodeID > maxPersistedFocusNodeID {
			return errNodeNotPersisted
		}
		node, err := getNode(p, treeID, nodeID)
		if err != nil {
			return err
		}
		dto := focusPreviewDTO(FocusTree{}, []FocusNode{node})
		apiutil.JSON(w, http.StatusOK, map[string]any{
			"script": previewemit.GenerateFocusNodeScript(dto.Nodes[0]), "tree_id": treeID,
			"node_id": nodeID, "tree_edit_revision": node.TreeEditRevision,
		})
		return nil
	}
	tree, err := getTree(p, treeID)
	if err != nil {
		return err
	}
	dto := focusPreviewDTO(tree, tree.Nodes)
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"script": previewemit.GenerateFocusTreeScript(dto, nil), "tree_id": treeID,
		"tree_edit_revision": tree.EditRevision,
	})
	return nil
}

type contentScriptParseError struct {
	Line     int    `json:"line"`
	Msg      string `json:"msg"`
	Severity string `json:"severity"`
}

type contentParsePreviewResponse struct {
	Entity      map[string]any            `json:"entity"`
	Extras      map[string]any            `json:"extras"`
	Warnings    []string                  `json:"warnings"`
	ParseErrors []contentScriptParseError `json:"parse_errors"`
}

var (
	contentParseErrorLineRE = regexp.MustCompile(`line\s+(\d+)`)
	previewQuotedTokenRE    = regexp.MustCompile(`'([^']+)'`)
)

func emptyContentParsePreviewResponse() contentParsePreviewResponse {
	return contentParsePreviewResponse{
		Entity: nil, Extras: map[string]any{}, Warnings: []string{},
		ParseErrors: []contentScriptParseError{},
	}
}

func previewValidationError(errorType string, loc []any, msg string, input any, ctx map[string]any) error {
	detail := map[string]any{"type": errorType, "loc": loc, "msg": msg, "input": input}
	if ctx != nil {
		detail["ctx"] = ctx
	}
	return &apiutil.StatusError{Status: http.StatusUnprocessableEntity, Body: map[string]any{"detail": []map[string]any{detail}}}
}

func previewJSONError(err error, offset int) error {
	if syntaxErr := new(json.SyntaxError); errors.As(err, &syntaxErr) && syntaxErr.Offset > 0 {
		offset = int(syntaxErr.Offset)
	}
	return previewValidationError("json_invalid", []any{"body", offset},
		"JSON decode error", map[string]any{}, map[string]any{"error": err.Error()})
}

func decodeParsePreviewBody(r *http.Request) (string, error) {
	raw, err := apiutil.ReadBody(r)
	if err != nil {
		return "", apiutil.BadRequestError(err.Error())
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var payload any
	if err := dec.Decode(&payload); errors.Is(err, io.EOF) {
		return "", previewValidationError("missing", []any{"body"}, "Field required", nil, nil)
	} else if err != nil {
		return "", previewJSONError(err, 1)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return "", previewJSONError(errors.New("Extra data"), int(dec.InputOffset())+1)
	}
	if payload == nil {
		return "", previewValidationError("missing", []any{"body"}, "Field required", nil, nil)
	}
	body, ok := payload.(map[string]any)
	if !ok {
		return "", previewValidationError("model_attributes_type", []any{"body"},
			"Input should be a valid dictionary or object to extract fields from", payload, nil)
	}
	value, present := body["script"]
	if !present {
		return "", previewValidationError("missing", []any{"body", "script"}, "Field required", body, nil)
	}
	script, ok := value.(string)
	if !ok {
		return "", previewValidationError("string_type", []any{"body", "script"}, "Input should be a valid string", value, nil)
	}
	return script, nil
}

func parsePreviewNode(p *project.Project, r *http.Request, treeID, nodeID int64) (contentParsePreviewResponse, error) {
	script, err := decodeParsePreviewBody(r)
	if err != nil {
		return contentParsePreviewResponse{}, err
	}
	if _, err := getTree(p, treeID); err != nil {
		return contentParsePreviewResponse{}, err
	}
	if _, err := getNode(p, treeID, nodeID); err != nil {
		return contentParsePreviewResponse{}, err
	}
	return parseFocusNodePreview(script)
}

const focusPreviewPreamble = "focus_tree = { id = _scratch_parse\n"
const focusPreviewSuffix = "\n}\n"

func parseFocusNodePreview(script string) (contentParsePreviewResponse, error) {
	empty := emptyContentParsePreviewResponse()
	if strings.TrimSpace(script) == "" {
		empty.ParseErrors = []contentScriptParseError{{Line: 1, Msg: "Script is empty", Severity: "info"}}
		return empty, nil
	}
	wrapped := focusPreviewPreamble + script + focusPreviewSuffix
	bundles, summary, err := treeimport.ImportFocusFile(wrapped, "")
	if err != nil {
		var parseErr *paradox.ParseError
		if errors.As(err, &parseErr) {
			return parsePreviewSyntaxError(empty, parseErr.Error(), 1), nil
		}

		if msg := err.Error(); msg == "cannot convert float infinity to integer" ||
			msg == "cannot convert float NaN to integer" ||
			msg == "int too large to convert to float" {
			return empty, err
		}
		return parsePreviewFieldError(empty, wrapped, err.Error(), 1), nil
	}
	warnings := append([]string{}, summary.Warnings...)
	if len(bundles) == 0 || len(bundles[0].NodesData) == 0 {
		empty.Warnings = warnings
		empty.ParseErrors = []contentScriptParseError{{
			Line:     1,
			Msg:      "No focus block found in script — the editor scope is one `focus = { ... }` block.",
			Severity: "error",
		}}
		return empty, nil
	}
	first := bundles[0].NodesData[0]
	entity, extras := splitFocusParseEntity(first, focusNodeUpdateFields)

	delete(entity, "name")
	delete(entity, "name_en")
	total := 0
	for _, bundle := range bundles {
		total += len(bundle.NodesData)
	}
	if len(bundles) > 1 || len(bundles[0].NodesData) > 1 {
		id := focusParseString(first["focus_id"], "?")
		warnings = append(warnings, "Script defined "+strconv.Itoa(total)+
			" focus nodes; only the first ("+id+") was used. Apply this and edit additional nodes separately.")
	}
	return contentParsePreviewResponse{Entity: entity, Extras: extras, Warnings: warnings, ParseErrors: []contentScriptParseError{}}, nil
}

func parsePreviewSyntaxError(empty contentParsePreviewResponse, msg string, lineOffset int) contentParsePreviewResponse {
	line := 1
	if match := contentParseErrorLineRE.FindStringSubmatch(msg); len(match) == 2 {
		if n, err := strconv.Atoi(match[1]); err == nil && n > 0 {
			line = n
		}
	}
	line -= lineOffset
	if line < 1 {
		line = 1
	}
	empty.ParseErrors = []contentScriptParseError{{Line: line, Msg: msg, Severity: "error"}}
	return empty
}

func parsePreviewFieldError(empty contentParsePreviewResponse, script, rawMsg string, lineOffset int) contentParsePreviewResponse {
	line := 0
	if match := previewQuotedTokenRE.FindStringSubmatch(rawMsg); len(match) == 2 {
		line = locatePreviewTokenLine(script, match[1])
	}
	if line == 0 {
		if match := contentParseErrorLineRE.FindStringSubmatch(rawMsg); len(match) == 2 {
			line, _ = strconv.Atoi(match[1])
		}
	}
	if line <= 0 {
		line = 1
	}
	line -= lineOffset
	if line < 1 {
		line = 1
	}
	empty.ParseErrors = []contentScriptParseError{{
		Line:     line,
		Msg:      "Field value error: " + rawMsg + ". A previous field's value was likely missing — the parser then consumed the next identifier (often shown above) as that value.",
		Severity: "error",
	}}
	return empty
}

func locatePreviewTokenLine(script, token string) int {
	if token == "" {
		return 0
	}
	for i, line := range strings.Split(script, "\n") {
		if strings.Contains(line, token) {
			return i + 1
		}
	}
	return 0
}

func splitFocusParseEntity(source map[string]any, allowed map[string]bool) (map[string]any, map[string]any) {
	entity := map[string]any{}
	extras := map[string]any{}
	for key, value := range source {
		value = focusParseJSONValue(value)
		if allowed[key] {
			entity[key] = value
		} else {
			extras[key] = value
		}
	}
	return entity, extras
}

func focusParseString(value any, fallback string) string {
	if text, ok := focusParseJSONValue(value).(string); ok && text != "" {
		return text
	}
	return fallback
}

func focusParseJSONValue(value any) any {
	switch v := value.(type) {
	case nil, string, bool, float64, int, int64:
		return v
	case *treeimport.RawDict:
		out := map[string]any{}
		for _, key := range v.Keys {
			item, _ := v.Get(key)
			out[key] = focusParseJSONValue(item)
		}
		return out
	case *importsvc.RawExtras:
		out := map[string]any{}
		for _, key := range v.Keys {
			item, _ := v.Get(key)
			out[key] = focusParseJSONValue(item)
		}
		return out
	case importsvc.ScopeVerdict:
		tags := v.TagList()
		if tags == nil {
			tags = []string{}
		}
		return map[string]any{"kind": string(v.Kind), "country_tags": tags}
	case *big.Int:
		return json.Number(v.String())
	case big.Int:
		return json.Number(v.String())
	case paradox.Scalar:
		switch v.Kind {
		case paradox.ScalarInt:
			return json.Number(v.Int.String())
		case paradox.ScalarFloat:
			return v.Float
		default:
			return v.Str
		}
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = focusParseJSONValue(item)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(v))
		for key, item := range v {
			out[key] = item
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, focusParseJSONValue(item))
		}
		return out
	case []map[string]any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, focusParseJSONValue(item))
		}
		return out
	case []string:
		return append([]string{}, v...)
	default:
		return v
	}
}
