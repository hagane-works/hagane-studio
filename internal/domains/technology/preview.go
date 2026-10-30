package technology

import (
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/importsvc"
	treeimport "haganeworks/internal/hoi4/importsvc/trees"
	"haganeworks/internal/hoi4/paradox"
)

var technologyUpdateFields = map[string]bool{
	"tech_id": true, "country_tag": true, "name": true,
	"tech_line": true, "research_cost": true, "start_year": true,
	"folder": true, "position_x": true, "position_y": true,
	"folder_placements": true, "position_y_is_explicit": true,
	"prerequisites": true, "categories": true, "modifiers": true,
	"ai_will_do": true, "name_en": true, "name_zh": true,
	"desc_en": true, "desc_zh": true, "display_name_locs": true,
	"description_locs": true, "raw_extras": true, "source": true,
	"is_banner": true, "force_use_small_tech_layout": true,
	"enable_equipments": true, "enable_subunits": true,
	"enable_equipment_modules": true, "enable_building": true,
	"enable_tactic": true, "xor": true, "dependencies": true,
	"sub_technologies": true, "doctrine": true, "allow": true,
	"allow_branch": true, "on_research_complete": true,
	"on_research_complete_limit": true, "path": true,
	"xp_research_type": true, "xp_boost_cost": true,
	"xp_research_bonus": true,
}

func internalError(w http.ResponseWriter) {
	detail(w, http.StatusInternalServerError, "Internal error")
}

func (h *handlers) preview(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	techLine := r.PathValue("line")
	techs, err := listTechnologies(p, techLine, "")
	if err != nil {
		internalError(w)
		return
	}
	yearColumns, _ := p.Record()["tech_year_columns"].([]any)
	if yearColumns == nil {
		yearColumns = []any{}
	}
	projectPayload := map[string]any{"tech_year_columns": yearColumns}

	if values, has := r.URL.Query()["tech_id"]; has {
		rawID := ""
		if len(values) > 0 {
			rawID = values[len(values)-1]
		}
		techID, parseErr := strconv.ParseInt(rawID, 10, 64)
		if parseErr != nil {
			writePreviewValidationError(w, "int_parsing", []any{"query", "tech_id"},
				"Input should be a valid integer, unable to parse string as an integer", rawID, nil)
			return
		}
		var selected *Technology
		for i := range techs {
			if techs[i].ID == techID {
				selected = &techs[i]
				break
			}
		}
		if selected == nil {
			detail(w, http.StatusNotFound, "Technology not found")
			return
		}
		body, err := json.Marshal(map[string]any{"tech": selected, "all_techs": techs, "project": projectPayload})
		if err == nil {
			var script string
			if script, err = emitmisc.EmitSingleTechJSON(body); err == nil {
				apiutil.JSON(w, http.StatusOK, map[string]string{"script": script})
				return
			}
		}
		internalError(w)
		return
	}
	body, err := json.Marshal(map[string]any{"techs": techs, "tech_line": techLine, "project": projectPayload, "siblings": nil})
	if err == nil {
		var script string
		if script, err = emitmisc.EmitTechLineJSON(body); err == nil {
			apiutil.JSON(w, http.StatusOK, map[string]string{"script": script})
			return
		}
	}
	internalError(w)
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

var contentParseErrorLineRE = regexp.MustCompile(`line\s+(\d+)`)

func emptyContentParsePreviewResponse() contentParsePreviewResponse {
	return contentParsePreviewResponse{
		Entity: nil, Extras: map[string]any{}, Warnings: []string{},
		ParseErrors: []contentScriptParseError{},
	}
}

func writePreviewValidationError(w http.ResponseWriter, errorType string, loc []any, msg string, input any, ctx map[string]any) {
	item := map[string]any{"type": errorType, "loc": loc, "msg": msg, "input": input}
	if ctx != nil {
		item["ctx"] = ctx
	}
	apiutil.JSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": []map[string]any{item}})
}

func writePreviewJSONError(w http.ResponseWriter, err error, offset int) {
	if syntaxErr := new(json.SyntaxError); errors.As(err, &syntaxErr) && syntaxErr.Offset > 0 {
		offset = int(syntaxErr.Offset)
	}
	writePreviewValidationError(w, "json_invalid", []any{"body", offset},
		"JSON decode error", map[string]any{}, map[string]any{"error": err.Error()})
}

func decodeContentParsePreviewBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, apiutil.MaxJSONBody))
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
	if payload == nil {
		writePreviewValidationError(w, "missing", []any{"body"}, "Field required", nil, nil)
		return "", false
	}
	body, ok := payload.(map[string]any)
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

func (h *handlers) parsePreview(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	raw := r.PathValue("tid")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writePreviewValidationError(w, "int_parsing", []any{"path", "tech_id"},
			"Input should be a valid integer, unable to parse string as an integer", raw, nil)
		return
	}
	if id <= 0 {
		detail(w, http.StatusNotFound, "Technology not found")
		return
	}
	script, ok := decodeContentParsePreviewBody(w, r)
	if !ok {
		return
	}
	tech, err := getTechnology(p, id)
	if errors.Is(err, errTechMissing) {
		detail(w, http.StatusNotFound, "Technology not found")
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	result, err := parseTechnologyPreview(script, tech.TechLine, tech.Folder)
	if err != nil {
		internalError(w)
		return
	}
	apiutil.JSON(w, http.StatusOK, result)
}

func parseTechnologyPreview(script, techLine, folderName string) (contentParsePreviewResponse, error) {
	empty := emptyContentParsePreviewResponse()
	if strings.TrimSpace(script) == "" {
		empty.ParseErrors = []contentScriptParseError{{Line: 1, Msg: "Script is empty", Severity: "info"}}
		return empty, nil
	}
	techs, _, summary, err := treeimport.ImportTechnologies(script, techLine, folderName)
	if err != nil {
		var parseErr *paradox.ParseError
		if !errors.As(err, &parseErr) {
			return empty, err
		}
		line := 1
		if match := contentParseErrorLineRE.FindStringSubmatch(parseErr.Error()); len(match) == 2 {
			if n, convErr := strconv.Atoi(match[1]); convErr == nil && n > 0 {
				line = n
			}
		}
		empty.ParseErrors = []contentScriptParseError{{Line: line, Msg: parseErr.Error(), Severity: "error"}}
		return empty, nil
	}
	warnings := append([]string{}, summary.Warnings...)
	if len(techs) == 0 {
		empty.Warnings = warnings
		empty.ParseErrors = []contentScriptParseError{{
			Line:     1,
			Msg:      "No 'technologies = { ... }' block found — wrap your edits in the standard HOI4 tech file structure.",
			Severity: "error",
		}}
		return empty, nil
	}
	entity := map[string]any{}
	extras := map[string]any{}
	for key, value := range techs[0] {
		value = technologyParseJSONValue(value)
		if technologyUpdateFields[key] {
			entity[key] = value
		} else {
			extras[key] = value
		}
	}
	if len(techs) > 1 {
		id := "?"
		if text, ok := technologyParseJSONValue(techs[0]["tech_id"]).(string); ok && text != "" {
			id = text
		}
		warnings = append(warnings, "Script defined "+strconv.Itoa(len(techs))+
			" technologies; only the first ("+id+") was used. Apply this and edit additional technologies separately.")
	}
	return contentParsePreviewResponse{
		Entity: entity, Extras: extras, Warnings: warnings,
		ParseErrors: []contentScriptParseError{},
	}, nil
}

func technologyParseJSONValue(value any) any {
	if extras, ok := value.(*treeimport.RawDict); ok {
		out := map[string]any{}
		for _, key := range extras.Keys {
			item, _ := extras.Get(key)
			out[key] = technologyParseJSONValue(item)
		}
		return out
	}
	return contentParseJSONValue(value)
}

func contentParseJSONValue(value any) any {
	switch v := value.(type) {
	case nil, string, bool, float64:
		return v
	case int:
		return v
	case int64:
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
			out[key] = contentParseJSONValue(item)
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
			out = append(out, contentParseJSONValue(item))
		}
		return out
	case []string:
		return append([]string{}, v...)
	case []map[string]any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, contentParseJSONValue(item))
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = contentParseJSONValue(item)
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

type KnownTechnologyFoldersResponse struct {
	Vanilla            map[string]TechFolderMetadata `json:"vanilla"`
	ProjectCustomLines []string                      `json:"project_custom_lines"`
}

func buildKnownTechnologyFolders(vanilla map[string]TechFolderMetadata, projectLines []string) KnownTechnologyFoldersResponse {
	seen := make(map[string]struct{}, len(projectLines))
	custom := make([]string, 0, len(projectLines))
	for _, line := range projectLines {
		if line == "" {
			continue
		}
		if _, vanillaLine := techLineFolders[line]; vanillaLine {
			continue
		}
		if _, duplicate := seen[line]; duplicate {
			continue
		}
		seen[line] = struct{}{}
		custom = append(custom, line)
	}
	sort.Strings(custom)
	if vanilla == nil {
		vanilla = map[string]TechFolderMetadata{}
	}
	return KnownTechnologyFoldersResponse{Vanilla: vanilla, ProjectCustomLines: custom}
}

func (h *handlers) knownFolders(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	rows, err := p.Rows(table)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "technology catalog unavailable")
		return
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, text(row, "tech_line", ""))
	}
	apiutil.JSON(w, http.StatusOK, buildKnownTechnologyFolders(h.game.folders(), lines))
}
