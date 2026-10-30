package ideas

import (
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
	ideaimport "haganeworks/internal/hoi4/importsvc/ideas"
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

var updateFields = map[string]bool{
	"idea_id": true, "country_tag": true, "scope_kind": true,
	"shared_country_tags": true, "idea_type": true, "name": true,
	"picture": true, "modifier": true, "ai_will_do": true,
	"removal_cost": true, "allowed": true, "allowed_civil_war": true,
	"is_designer": true, "research_bonus": true, "designer_traits": true,
	"name_en": true, "name_zh": true, "desc_en": true, "desc_zh": true,
	"display_name_locs": true, "description_locs": true, "raw_extras": true,
	"available": true, "visible": true, "allowed_to_remove": true,
	"cancel": true, "do_effect": true, "on_add": true, "on_remove": true,
	"cancel_if_invalid": true, "targeted_modifier": true,
	"equipment_bonus": true, "rule": true, "level": true, "ledger": true,
	"cost": true, "auto_attach_on_start": true, "source": true,
}

func preview(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	if values, ok := r.URL.Query()["idea_id"]; ok {
		raw := ""
		if len(values) > 0 {
			raw = values[len(values)-1]
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fastAPIIntQueryError("idea_id", raw)
		}
		if id <= 0 {
			return errIdeaMissing
		}
		idea, err := getIdea(p, id)
		if err != nil {
			return err
		}
		body, err := json.Marshal(map[string]any{"idea": idea})
		if err != nil {
			return err
		}
		script, err := previewemit.EmitSingleIdeaJSON(body)
		if err != nil {
			return err
		}
		apiutil.JSON(w, http.StatusOK, map[string]string{"script": script})
		return nil
	}
	items, err := listIdeas(p, "", "")
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{"ideas": items, "picture_refs": nil})
	if err != nil {
		return err
	}
	script, err := previewemit.EmitIdeasJSON(body)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]string{"script": script})
	return nil
}

func parsePreview(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := ideaID(r)
	if err != nil {
		return err
	}
	var body struct {
		Script *string `json:"script"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
	if err := decoder.Decode(&body); err != nil || body.Script == nil {
		return fastAPIBodyFieldError("script")
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fastAPIBodyFieldError("script")
	}
	if _, err := getIdea(p, id); err != nil {
		return err
	}
	result, err := parseIdeaScript(*body.Script)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}

func parseIdeaScript(script string) (parsePreviewResponse, error) {
	empty := parsePreviewResponse{
		Entity: nil, Extras: map[string]any{}, Warnings: []string{},
		ParseErrors: []scriptParseError{},
	}
	if strings.TrimSpace(script) == "" {
		empty.ParseErrors = []scriptParseError{{Line: 1, Msg: "Script is empty", Severity: "info"}}
		return empty, nil
	}

	entities, summary, err := ideaimport.ImportIdeas(script)
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
	if len(entities) == 0 {
		empty.Warnings = warnings
		empty.ParseErrors = []scriptParseError{{
			Line:     1,
			Msg:      "No 'ideas = { <category> = { <idea_id> = { ... } } }' block found — wrap your edits in the standard HOI4 ideas file structure.",
			Severity: "error",
		}}
		return empty, nil
	}

	first := entities[0]
	entity := map[string]any{}
	extras := map[string]any{}
	for _, key := range first.Keys() {
		value, _ := first.Get(key)
		value = parseJSONValue(value)
		if updateFields[key] {
			entity[key] = value
		} else {
			extras[key] = value
		}
	}
	if len(entities) > 1 {
		firstID := "?"
		if value, ok := first.Get("idea_id"); ok {
			if s, ok := parseJSONValue(value).(string); ok && s != "" {
				firstID = s
			}
		}
		warnings = append(warnings, "Script defined "+strconv.Itoa(len(entities))+
			" ideas; only the first ("+firstID+") was used. Apply this and edit additional ideas separately.")
	}
	return parsePreviewResponse{
		Entity: entity, Extras: extras, Warnings: warnings,
		ParseErrors: []scriptParseError{},
	}, nil
}

func parseJSONValue(value any) any {
	switch v := value.(type) {
	case nil, string, bool, float64:
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
	case *ideaimport.Dict:
		out := map[string]any{}
		for _, key := range v.Keys() {
			item, _ := v.Get(key)
			out[key] = parseJSONValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, parseJSONValue(item))
		}
		return out
	case []string:
		return append([]string{}, v...)
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = parseJSONValue(item)
		}
		return out
	case importsvc.ScopeVerdict:
		tags := v.TagList()
		if tags == nil {
			tags = []string{}
		}
		return map[string]any{"kind": string(v.Kind), "country_tags": tags}
	default:
		return v
	}
}

func fastAPIIntQueryError(field, input string) error {
	return &apiutil.StatusError{Status: http.StatusUnprocessableEntity, Body: map[string]any{"detail": []map[string]any{{
		"type": "int_parsing", "loc": []any{"query", field},
		"msg":   "Input should be a valid integer, unable to parse string as an integer",
		"input": input,
	}}}}
}

func fastAPIBodyFieldError(field string) error {
	return &apiutil.StatusError{Status: http.StatusUnprocessableEntity, Body: map[string]any{"detail": []map[string]any{{
		"type": "missing", "loc": []any{"body", field}, "msg": "Field required",
	}}}}
}
