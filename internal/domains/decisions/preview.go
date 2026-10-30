package decisions

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/importsvc"
	eventimport "haganeworks/internal/hoi4/importsvc/events"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

type scriptParseError struct {
	Line     int    `json:"line"`
	Msg      string `json:"msg"`
	Severity string `json:"severity"`
}

type parsePreviewResponse struct {
	Entity      map[string]any     `json:"entity"`
	Extras      map[string]any     `json:"extras"`
	Warnings    []string           `json:"warnings"`
	ParseErrors []scriptParseError `json:"parse_errors"`
}

var parseErrorLineRE = regexp.MustCompile(`line\s+(\d+)`)

var decisionUpdateFields = map[string]bool{
	"category_id": true, "decision_id": true, "icon": true,
	"allowed": true, "available": true, "visible": true, "cost": true,
	"fire_only_once": true, "cancel_if_not_visible": true,
	"days_remove": true, "days_re_enable": true, "complete_effect": true,
	"remove_effect": true, "cancel_trigger": true, "cancel_effect": true,
	"modifier": true, "activation": true, "timeout_effect": true,
	"days_mission_timeout": true, "selectable_mission": true,
	"is_good": true, "targeted_modifier": true, "targets": true,
	"target_array": true, "target_root_trigger": true,
	"target_trigger": true, "state_target": true, "highlight_states": true,
	"on_map_mode": true, "priority": true, "custom_cost_trigger": true,
	"custom_cost_text": true, "war_with_on_add": true,
	"war_with_on_remove": true, "ai_will_do": true, "country_tag": true,
	"scope_kind": true, "shared_country_tags": true, "name_en": true,
	"name_zh": true, "desc_en": true, "desc_zh": true,
	"display_name_locs": true, "description_locs": true,
	"effect_tooltip_locs": true, "raw_extras": true, "source": true,
}

func internalError(w http.ResponseWriter, what string, err error) {
	log.Printf("decisions %s: %v", what, err)
	detail(w, http.StatusInternalServerError, "Internal error")
}

func emitDecisionsPreview(categories []DecisionCategory, decisions []Decision) (string, error) {
	body, err := json.Marshal(map[string]any{
		"categories": categories, "decisions": decisions, "icon_gates": nil,
	})
	if err != nil {
		return "", err
	}
	return emitmisc.EmitDecisionsJSON(body)
}

func categoryPreview(w http.ResponseWriter, p *project.Project, id int64) {
	category, err := getCategory(p, id)
	if errors.Is(err, errCategoryMissing) {
		detail(w, http.StatusNotFound, "Category not found")
		return
	}
	if err != nil {
		internalError(w, "category preview", err)
		return
	}
	decisions, err := listDecisions(p, &id, "")
	if err != nil {
		internalError(w, "category preview list", err)
		return
	}
	script, err := emitDecisionsPreview([]DecisionCategory{category}, decisions)
	if err != nil {
		internalError(w, "category preview emit", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"script": script})
}

func decisionPreview(w http.ResponseWriter, p *project.Project, id int64) {
	decision, err := getDecision(p, id)
	if errors.Is(err, errDecisionMissing) {
		detail(w, http.StatusNotFound, "Decision not found")
		return
	}
	if err != nil {
		internalError(w, "preview", err)
		return
	}
	categories := []DecisionCategory{}
	category, err := getCategory(p, decision.CategoryID)
	if err == nil {
		categories = append(categories, category)
	} else if !errors.Is(err, errCategoryMissing) {
		internalError(w, "preview category", err)
		return
	}
	script, err := emitDecisionsPreview(categories, []Decision{decision})
	if err != nil {
		internalError(w, "preview emit", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"script": script})
}

func decisionParsePreview(w http.ResponseWriter, r *http.Request, p *project.Project, id int64) {
	script, ok := decodeParsePreviewBody(w, r)
	if !ok {
		return
	}
	if _, err := findDecisionRow(p, id); errors.Is(err, errDecisionMissing) {
		detail(w, http.StatusNotFound, "Decision not found")
		return
	} else if err != nil {
		internalError(w, "parse-preview lookup", err)
		return
	}
	result, err := parseDecisionPreview(script)
	if err != nil {
		internalError(w, "parse-preview", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func decodeParsePreviewBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20))
	dec.UseNumber()
	var payload any
	if err := dec.Decode(&payload); errors.Is(err, io.EOF) {
		writePreviewValidationError(w, "missing", []any{"body"}, "Field required", nil, nil)
		return "", false
	} else if err != nil {
		writePreviewJSONError(w, err, 1)
		return "", false
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		writePreviewJSONError(w, errors.New("Extra data"), int(dec.InputOffset())+1)
		return "", false
	}
	body, ok := payload.(map[string]any)
	if payload == nil {
		writePreviewValidationError(w, "missing", []any{"body"}, "Field required", nil, nil)
		return "", false
	}
	if !ok {
		writePreviewValidationError(w, "model_attributes_type", []any{"body"},
			"Input should be a valid dictionary or object to extract fields from", payload, nil)
		return "", false
	}
	value, present := body["script"]
	if !present {
		writePreviewValidationError(w, "missing", []any{"body", "script"}, "Field required", body, nil)
		return "", false
	}
	script, ok := value.(string)
	if !ok {
		writePreviewValidationError(w, "string_type", []any{"body", "script"},
			"Input should be a valid string", value, nil)
		return "", false
	}
	return script, true
}

func writePreviewValidationError(w http.ResponseWriter, errorType string, loc []any, msg string, input any, ctx map[string]any) {
	item := map[string]any{"type": errorType, "loc": loc, "msg": msg, "input": input}
	if ctx != nil {
		item["ctx"] = ctx
	}
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": []map[string]any{item}})
}

func writePreviewJSONError(w http.ResponseWriter, err error, offset int) {
	if syntaxErr := new(json.SyntaxError); errors.As(err, &syntaxErr) && syntaxErr.Offset > 0 {
		offset = int(syntaxErr.Offset)
	}
	writePreviewValidationError(w, "json_invalid", []any{"body", offset},
		"JSON decode error", map[string]any{}, map[string]any{"error": err.Error()})
}

func writePreviewIntPathError(w http.ResponseWriter, field, input string) {
	writePreviewValidationError(w, "int_parsing", []any{"path", field},
		"Input should be a valid integer, unable to parse string as an integer", input, nil)
}

func parseDecisionPreview(script string) (parsePreviewResponse, error) {
	empty := parsePreviewResponse{
		Entity: nil, Extras: map[string]any{}, Warnings: []string{},
		ParseErrors: []scriptParseError{},
	}
	if strings.TrimSpace(script) == "" {
		empty.ParseErrors = []scriptParseError{{Line: 1, Msg: "Script is empty", Severity: "info"}}
		return empty, nil
	}
	_, decisions, summary, err := eventimport.ImportDecisions(script)
	if err != nil {
		var parseErr *paradox.ParseError
		if !errors.As(err, &parseErr) {
			return empty, err
		}
		line := 1
		if match := parseErrorLineRE.FindStringSubmatch(parseErr.Error()); len(match) == 2 {
			if n, convErr := strconv.Atoi(match[1]); convErr == nil && n > 0 {
				line = n
			}
		}
		empty.ParseErrors = []scriptParseError{{Line: line, Msg: parseErr.Error(), Severity: "error"}}
		return empty, nil
	}
	warnings := append([]string{}, summary.Warnings...)
	if len(decisions) == 0 {
		empty.Warnings = warnings
		empty.ParseErrors = []scriptParseError{{
			Line:     1,
			Msg:      "No decision found in script — wrap your edits in the standard HOI4 decision file structure: <category> = { <decision_id> = { ... } }.",
			Severity: "error",
		}}
		return empty, nil
	}
	entity, extras := map[string]any{}, map[string]any{}
	for key, value := range decisions[0] {
		value = parseJSONValue(value)
		if decisionUpdateFields[key] {
			entity[key] = value
		} else {
			extras[key] = value
		}
	}
	if len(decisions) > 1 {
		id := "?"
		if s, ok := parseJSONValue(decisions[0]["decision_id"]).(string); ok && s != "" {
			id = s
		}
		warnings = append(warnings, "Script defined "+strconv.Itoa(len(decisions))+
			" decisions; only the first ("+id+") was used. Apply this and edit additional decisions separately.")
	}
	return parsePreviewResponse{
		Entity: entity, Extras: extras, Warnings: warnings,
		ParseErrors: []scriptParseError{},
	}, nil
}

func parseJSONValue(value any) any {
	switch v := value.(type) {
	case nil, string, bool, float64, int, int64:
		return v
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
	case *importsvc.RawExtras:
		out := map[string]any{}
		for _, key := range v.Keys {
			item, _ := v.Get(key)
			out[key] = parseJSONValue(item)
		}
		return out
	case importsvc.ScopeVerdict:
		tags := v.TagList()
		if tags == nil {
			tags = []string{}
		}
		return map[string]any{"kind": string(v.Kind), "country_tags": tags}
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, parseJSONValue(item))
		}
		return out
	case []string:
		return append([]string{}, v...)
	case []map[string]any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, parseJSONValue(item))
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = parseJSONValue(item)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(v))
		for key, item := range v {
			out[key] = item
		}
		return out
	default:
		return v
	}
}
