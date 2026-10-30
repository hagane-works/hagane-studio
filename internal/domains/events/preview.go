package events

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
	eventimport "haganeworks/internal/hoi4/importsvc/events"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

func previewID(r *http.Request, name, field string, missing error) (int64, error) {
	raw := r.PathValue(name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, previewValidationError("int_parsing", []any{"path", field},
			"Input should be a valid integer, unable to parse string as an integer", raw, nil)
	}
	if id <= 0 {
		return 0, missing
	}
	return id, nil
}

func (h *handlers) namespacePreview(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	nsID, err := previewID(r, "nsid", "ns_id", errNamespaceMissing)
	if err != nil {
		return err
	}
	namespace, err := getNamespace(p, nsID)
	if err != nil {
		return err
	}
	events, err := listEvents(p, &nsID, "")
	if err != nil {
		return err
	}
	script, err := emitEventsPreview(namespace.Namespace, events)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]string{"script": script})
	return nil
}

func (h *handlers) eventPreview(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	eventID, err := previewID(r, "eid", "event_db_id", errEventMissing)
	if err != nil {
		return err
	}
	event, err := getEvent(p, eventID)
	if err != nil {
		return err
	}
	namespaceName := "unknown"
	if namespace, err := getNamespace(p, event.NamespaceID); err == nil {
		namespaceName = namespace.Namespace
	} else if !errors.Is(err, errNamespaceMissing) {
		return err
	}
	script, err := emitEventsPreview(namespaceName, []eventResponse{event})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]string{"script": script})
	return nil
}

func emitEventsPreview(namespace string, events []eventResponse) (string, error) {
	body, err := json.Marshal(map[string]any{
		"namespace": namespace, "events": events, "valid_pictures": nil,
	})
	if err != nil {
		return "", err
	}
	return previewemit.EmitEventsJSON(body)
}

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

var eventUpdateFields = map[string]bool{
	"namespace_id": true, "event_id": true, "event_number": true,
	"event_type": true, "title": true, "description": true,
	"picture": true, "is_triggered_only": true, "fire_only_once": true,
	"hidden": true, "major": true, "trigger": true,
	"mean_time_to_happen": true, "immediate": true, "after": true,
	"options": true, "timeout_days": true, "country_tag": true,
	"scope_kind": true, "shared_country_tags": true, "title_en": true,
	"title_zh": true, "desc_en": true, "desc_zh": true,
	"raw_extras": true, "auto_fire_on_startup": true,
	"auto_fire_after_days": true, "on_action_attach": true, "source": true,
	"display_name_locs": true, "description_locs": true,
}

func (h *handlers) parsePreview(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	eventID, err := previewID(r, "eid", "event_db_id", errEventMissing)
	if err != nil {
		return err
	}
	script, err := decodeParsePreviewBody(r)
	if err != nil {
		return err
	}
	if _, err := getEvent(p, eventID); err != nil {
		return err
	}
	result, err := parseEventPreview(script)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}

func decodeParsePreviewBody(r *http.Request) (string, error) {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20))
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

func parseEventPreview(script string) (parsePreviewResponse, error) {
	empty := parsePreviewResponse{
		Entity: nil, Extras: map[string]any{}, Warnings: []string{},
		ParseErrors: []scriptParseError{},
	}
	if strings.TrimSpace(script) == "" {
		empty.ParseErrors = []scriptParseError{{Line: 1, Msg: "Script is empty", Severity: "info"}}
		return empty, nil
	}
	_, events, summary, err := eventimport.ImportEvents(script)
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
	if len(events) == 0 {
		empty.Warnings = warnings
		empty.ParseErrors = []scriptParseError{{
			Line:     1,
			Msg:      "No event found in script — wrap your edits with the namespace declaration: add_namespace = <ns> followed by <event_type> = { ... }.",
			Severity: "error",
		}}
		return empty, nil
	}
	entity, extras := map[string]any{}, map[string]any{}
	for key, value := range events[0] {
		value = parseJSONValue(value)
		if eventUpdateFields[key] {
			entity[key] = value
		} else {
			extras[key] = value
		}
	}
	if len(events) > 1 {
		id := "?"
		if s, ok := parseJSONValue(events[0]["event_id"]).(string); ok && s != "" {
			id = s
		}
		warnings = append(warnings, "Script defined "+strconv.Itoa(len(events))+
			" events; only the first ("+id+") was used. Apply this and edit additional events separately.")
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
