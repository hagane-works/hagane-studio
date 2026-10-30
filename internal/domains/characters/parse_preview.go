package characters

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
	"haganeworks/internal/hoi4/importsvc"
	characterimport "haganeworks/internal/hoi4/importsvc/characters"
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

var (
	parseErrorLineRE     = regexp.MustCompile(`line\s+(\d+)`)
	previewQuotedTokenRE = regexp.MustCompile(`'([^']+)'`)
)

var characterUpdateFields = map[string]bool{
	"name": true, "tag": true, "localisation_key": true,
	"country_tag": true, "description": true, "role_type": true,
	"has_advisor_role": true, "is_replacement": true,
	"replaces_vanilla_id": true, "skill": true, "attack_skill": true,
	"defense_skill": true, "planning_skill": true, "logistics_skill": true,
	"maneuvering_skill": true, "coordination_skill": true,
	"military_visible": true, "traits": true, "advisor_traits": true,
	"advisor_slot": true, "advisor_idea_token": true, "advisor_cost": true,
	"advisor_available": true, "advisor_visible": true,
	"advisor_removal_cost": true, "advisor_modifier": true,
	"advisor_on_add": true, "advisor_on_remove": true, "gender": true,
	"can_be_captured": true, "always_show_on_actions_tooltip": true,
	"scientist_data": true, "scientist_visible": true,
	"operative_skill_level": true, "operative_traits": true,
	"operative_nationalities": true, "operative_available": true,
	"operative_bg": true, "ideology": true, "expire": true,
	"name_en": true, "name_zh": true, "desc_en": true, "desc_zh": true,
	"display_name_locs": true, "description_locs": true,
	"raw_extras": true, "source": true, "source_file": true,
	"is_portrait_modified": true,
}

func characterParsePreview(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, ok := pathID(r, "cid")
	if !ok {
		return errCharacterMissing
	}
	script, err := decodeParsePreviewBody(w, r)
	if err != nil {
		return err
	}
	row, err := findRow(p, table, cid)
	if err != nil {
		return err
	}
	if row == nil {
		return errCharacterMissing
	}
	result, err := parseCharacterPreview(script)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}

func parseCharacterPreview(script string) (parsePreviewResponse, error) {
	empty := emptyParsePreviewResponse()
	if strings.TrimSpace(script) == "" {
		empty.ParseErrors = []scriptParseError{{Line: 1, Msg: "Script is empty", Severity: "info"}}
		return empty, nil
	}
	characters, summary, err := characterimport.ImportCharacters(script)
	if err != nil {
		var pyErr *characterimport.PyError
		if !errors.As(err, &pyErr) {
			return empty, err
		}
		switch pyErr.Type {
		case "ParseError":
			return parsePreviewSyntaxError(empty, pyErr.Msg, 0), nil
		case "ValueError", "TypeError", "KeyError", "AttributeError":
			return parsePreviewFieldError(empty, script, pyErr.Msg, 0), nil
		default:
			return empty, err
		}
	}
	warnings := []string{}
	if summary != nil {
		warnings = append(warnings, summary.Warnings...)
	}
	if len(characters) == 0 {
		empty.Warnings = warnings
		empty.ParseErrors = []scriptParseError{{
			Line:     1,
			Msg:      "No 'characters = { TAG = { ... } }' block found — wrap your edits in the standard HOI4 character file structure.",
			Severity: "error",
		}}
		return empty, nil
	}
	entity, extras := splitParseEntity(characters[0], characterUpdateFields)
	if len(characters) > 1 {
		tag := parseString(characters[0]["tag"], "?")
		warnings = append(warnings, "Script defined "+strconv.Itoa(len(characters))+
			" characters; only the first ("+tag+") was used. Apply this and edit additional characters separately.")
	}
	return parsePreviewResponse{Entity: entity, Extras: extras, Warnings: warnings, ParseErrors: []scriptParseError{}}, nil
}

func emptyParsePreviewResponse() parsePreviewResponse {
	return parsePreviewResponse{
		Entity: nil, Extras: map[string]any{}, Warnings: []string{},
		ParseErrors: []scriptParseError{},
	}
}

func decodeParsePreviewBody(w http.ResponseWriter, r *http.Request) (string, error) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
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
		return "", previewValidationError("string_type", []any{"body", "script"},
			"Input should be a valid string", value, nil)
	}
	return script, nil
}

func previewValidationError(errorType string, loc []any, msg string, input any, ctx map[string]any) error {
	item := map[string]any{"type": errorType, "loc": loc, "msg": msg, "input": input}
	if ctx != nil {
		item["ctx"] = ctx
	}
	return &apiutil.StatusError{Status: http.StatusUnprocessableEntity, Body: map[string]any{"detail": []map[string]any{item}}}
}

func previewJSONError(err error, offset int) error {
	if syntaxErr := new(json.SyntaxError); errors.As(err, &syntaxErr) && syntaxErr.Offset > 0 {
		offset = int(syntaxErr.Offset)
	}
	return previewValidationError("json_invalid", []any{"body", offset},
		"JSON decode error", map[string]any{}, map[string]any{"error": err.Error()})
}

func parsePreviewSyntaxError(empty parsePreviewResponse, msg string, lineOffset int) parsePreviewResponse {
	line := 1
	if match := parseErrorLineRE.FindStringSubmatch(msg); len(match) == 2 {
		if n, err := strconv.Atoi(match[1]); err == nil && n > 0 {
			line = n
		}
	}
	line -= lineOffset
	if line < 1 {
		line = 1
	}
	empty.ParseErrors = []scriptParseError{{Line: line, Msg: msg, Severity: "error"}}
	return empty
}

func parsePreviewFieldError(empty parsePreviewResponse, script, rawMsg string, lineOffset int) parsePreviewResponse {
	line := 0
	if match := previewQuotedTokenRE.FindStringSubmatch(rawMsg); len(match) == 2 {
		line = locateTokenLine(script, match[1])
	}
	if line == 0 {
		if match := parseErrorLineRE.FindStringSubmatch(rawMsg); len(match) == 2 {
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
	empty.ParseErrors = []scriptParseError{{
		Line:     line,
		Msg:      "Field value error: " + rawMsg + ". A previous field's value was likely missing — the parser then consumed the next identifier (often shown above) as that value.",
		Severity: "error",
	}}
	return empty
}

func locateTokenLine(script, token string) int {
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

func splitParseEntity(source map[string]any, allowed map[string]bool) (map[string]any, map[string]any) {
	entity := map[string]any{}
	extras := map[string]any{}
	for key, value := range source {
		value = parseJSONValue(value)
		if allowed[key] {
			entity[key] = value
		} else {
			extras[key] = value
		}
	}
	return entity, extras
}

func parseString(value any, fallback string) string {
	if text, ok := parseJSONValue(value).(string); ok && text != "" {
		return text
	}
	return fallback
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
