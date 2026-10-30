package content

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"haganeworks/internal/apiutil"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/project"
)

const (
	panelsTable  = "ui_panels"
	panelMissing = "Panel not found"
)

const panelFileNS = "zz_uipanel_"

var panelNameRE = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func panelNameSlug(raw, fallback string) string {
	s := strings.Trim(panelNameRE.ReplaceAllString(strings.TrimSpace(raw), "_"), "_")
	if s == "" {
		return fallback
	}
	return s
}

func derivePanelNames(token, controlName string, withControl bool) map[string]string {
	t := panelNameSlug(token, "panel")
	names := map[string]string{
		"token":            t,
		"scripted_gui_key": t,
		"window_name":      t + "_panel",
		"gui_path":         "interface/" + panelFileNS + t + ".gui",
		"txt_path":         "common/scripted_guis/" + panelFileNS + t + ".txt",
		"loc_basename":     t + "_panel",
		"title_loc_key":    t + "_title",
	}
	if !withControl {
		return names
	}
	c := panelNameSlug(controlName, "control")
	full := t + "_" + c
	names["control_short"] = c
	names["control_name"] = full
	names["visible_trigger_key"] = full + "_visible"
	names["text_loc_key"] = full + "_loc"
	names["click_key"] = full + "_click"
	names["click_enabled_key"] = full + "_click_enabled"
	names["tooltip_loc_key"] = full + "_tt"
	return names
}

func anyToMapSlice(v any) []map[string]any {
	items, ok := v.([]any)
	if !ok {
		if direct, ok := v.([]map[string]any); ok {
			return direct
		}
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if value, ok := item.(map[string]any); ok {
			out = append(out, value)
		}
	}
	return out
}

func walkPanelControls(controls []map[string]any) []map[string]any {
	out := []map[string]any{}
	for _, ctrl := range controls {
		if ctrl == nil {
			continue
		}
		out = append(out, ctrl)
		if strings.ToLower(strings.TrimSpace(anyToString(ctrl["kind"]))) == "container" {
			out = append(out, walkPanelControls(anyToMapSlice(ctrl["children"]))...)
		}
	}
	return out
}

func derivedForPanel(token string, controls []map[string]any) map[string]any {
	derived := map[string]any{}
	for k, v := range derivePanelNames(token, "", false) {
		derived[k] = v
	}
	byControl := map[string]any{}
	for _, ctrl := range walkPanelControls(controls) {
		short := anyToString(ctrl["name"])
		n := derivePanelNames(token, short, true)
		byControl[short] = map[string]any{
			"control_name":        n["control_name"],
			"visible_trigger_key": n["visible_trigger_key"],
			"text_loc_key":        n["text_loc_key"],
			"click_key":           n["click_key"],
			"click_enabled_key":   n["click_enabled_key"],
			"tooltip_loc_key":     n["tooltip_loc_key"],
		}
	}
	derived["controls"] = byControl
	return derived
}

type uiPanel struct {
	ID                int64            `json:"id"`
	ProjectID         int64            `json:"project_id"`
	CountryTag        string           `json:"country_tag"`
	Token             string           `json:"token"`
	NameEn            string           `json:"name_en"`
	NameZh            string           `json:"name_zh"`
	DisplayNameLocs   map[string]any   `json:"display_name_locs"`
	ContextType       string           `json:"context_type"`
	ParentWindowToken string           `json:"parent_window_token"`
	VisibleTrigger    string           `json:"visible_trigger"`
	WindowProps       map[string]any   `json:"window_props"`
	Controls          []map[string]any `json:"controls"`
	RawExtras         map[string]any   `json:"raw_extras"`
	Source            string           `json:"source"`
	CreatedAt         string           `json:"created_at"`
	UpdatedAt         string           `json:"updated_at"`
	Derived           map[string]any   `json:"derived"`
	Warnings          []string         `json:"warnings"`
}

func panelFromRow(row project.Row) uiPanel {
	item := uiPanel{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		CountryTag: text(row, "country_tag"), Token: text(row, "token"),
		NameEn: text(row, "name_en"), NameZh: text(row, "name_zh"),
		DisplayNameLocs:   anyMap(row["display_name_locs"]),
		ContextType:       textOr(row, "context_type", "decision_category"),
		ParentWindowToken: text(row, "parent_window_token"),
		VisibleTrigger:    text(row, "visible_trigger"),
		WindowProps:       anyMap(row["window_props"]), RawExtras: anyMap(row["raw_extras"]),
		Source:    textOr(row, "source", "created"),
		CreatedAt: isoTime(row["created_at"]), UpdatedAt: isoTime(row["updated_at"]),
		Warnings: []string{},
	}
	decode(row["controls"], &item.Controls)
	if item.Controls == nil {
		item.Controls = []map[string]any{}
	}
	item.Derived = derivedForPanel(item.Token, item.Controls)
	return item
}

var allowedPanelContextTypes = map[string]bool{
	"player_context":           true,
	"selected_country_context": true,
	"selected_state_context":   true,
	"decision_category":        true,
	"diplomatic_action":        true,
	"national_focus_context":   true,
	"country_mapicon":          true,
	"state_mapicon":            true,
}

func normalizePanelContextType(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "decision_category"
	}
	if allowedPanelContextTypes[value] {
		return value, nil
	}
	allowed := make([]string, 0, len(allowedPanelContextTypes))
	for item := range allowedPanelContextTypes {
		allowed = append(allowed, item)
	}
	sort.Strings(allowed)
	return "", statusErr(http.StatusUnprocessableEntity, fmt.Sprintf(
		"context_type %s is not a valid scripted_gui context (allowed: %v)", pyRepr(value), allowed))
}

type panelInput struct {
	ExpectedUpdatedAt *string           `json:"expected_updated_at,omitempty"`
	Token             *string           `json:"token"`
	CountryTag        *string           `json:"country_tag"`
	NameEn            *string           `json:"name_en"`
	NameZh            *string           `json:"name_zh"`
	DisplayNameLocs   *map[string]any   `json:"display_name_locs"`
	ContextType       *string           `json:"context_type"`
	ParentWindowToken *string           `json:"parent_window_token"`
	VisibleTrigger    *string           `json:"visible_trigger"`
	WindowProps       *map[string]any   `json:"window_props"`
	Controls          *[]map[string]any `json:"controls"`
	RawExtras         *map[string]any   `json:"raw_extras"`
	Source            *string           `json:"source"`
}

func orDefault[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}

func (h *handlers) uiPanels(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	if len(rest) == 2 {
		id, err := strconv.ParseInt(rest[0], 10, 64)
		if err != nil || id <= 0 || rest[1] != "files" {
			detail(w, http.StatusNotFound, panelMissing)
			return
		}
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET only"})
			return
		}
		files, err := panelFiles(p, id)
		if err != nil {
			fail(w, err, "store unavailable")
			return
		}
		writeJSON(w, http.StatusOK, files)
		return
	}
	crud{
		listKey: "panels", missing: panelMissing, unavailable: "store unavailable",
		list: func(p *project.Project, _ *http.Request) (any, int, error) {
			items, err := listRows(p, panelsTable, byText("token"), panelFromRow)
			return items, len(items), err
		},
		create: createPanel,
		get: func(p *project.Project, id int64) (any, error) {
			row, err := getRow(p, panelsTable, id, panelMissing)
			if err != nil {
				return nil, err
			}
			return panelFromRow(row), nil
		},
		update: updatePanel,
		remove: removeRow(panelsTable, panelMissing),
	}.serve(p, w, r, rest)
}

func createPanel(p *project.Project, r *http.Request) (any, error) {
	var in panelInput
	if err := decodeBody(r, &in); err != nil {
		return nil, errInvalidPayload
	}
	token := strings.TrimSpace(orDefault(in.Token, ""))
	if token == "" {
		return nil, statusErr(http.StatusUnprocessableEntity, "token required")
	}
	contextType, err := normalizePanelContextType(orDefault(in.ContextType, ""))
	if err != nil {
		return nil, err
	}
	var unique string
	row, err := writeRow(p, panelsTable, func(tx *project.Tx) (project.Row, error) {
		unique = suffixed(tx.Rows(panelsTable), "token", token)
		return insert(tx, panelsTable, project.Row{
			"token": unique, "country_tag": strings.ToUpper(strings.TrimSpace(orDefault(in.CountryTag, ""))),
			"name_en": orDefault(in.NameEn, ""), "name_zh": orDefault(in.NameZh, ""),
			"display_name_locs":   jsonValue(orDefault(in.DisplayNameLocs, map[string]any{})),
			"context_type":        contextType,
			"parent_window_token": strings.TrimSpace(orDefault(in.ParentWindowToken, "")),
			"visible_trigger":     orDefault(in.VisibleTrigger, ""),
			"window_props":        jsonValue(orDefault(in.WindowProps, map[string]any{})),
			"controls":            jsonValue(orDefault(in.Controls, []map[string]any{})),
			"raw_extras":          jsonValue(orDefault(in.RawExtras, map[string]any{})),
			"source":              firstNonEmpty(orDefault(in.Source, ""), "created"),
		})
	})
	if err != nil {
		return nil, err
	}
	item := panelFromRow(row)
	if unique != token {
		item.Warnings = append(item.Warnings, "token "+pyRepr(token)+" collided · auto-suffixed to "+pyRepr(unique)+".")
	}
	return item, nil
}

func firstNonEmpty(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func panelRevisionConflict(id int64, expected, actual time.Time) error {
	e, a := expected.UTC().Format(time.RFC3339Nano), actual.UTC().Format(time.RFC3339Nano)
	return &apiutil.StatusError{Status: http.StatusConflict, Body: map[string]any{
		"detail":   fmt.Sprintf("stale ui-panel revision: expected_updated_at=%s, actual_updated_at=%s", e, a),
		"code":     "ui_panel_revision_conflict",
		"panel_id": id, "expected_updated_at": e, "actual_updated_at": a,
	}}
}

func updatePanel(p *project.Project, r *http.Request, id int64) (any, error) {
	var in panelInput
	if err := decodeBody(r, &in); err != nil {
		return nil, errInvalidPayload
	}
	var expected *time.Time
	if in.ExpectedUpdatedAt != nil {
		raw := strings.TrimSpace(*in.ExpectedUpdatedAt)
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil || raw == "" {
			return nil, statusErr(http.StatusUnprocessableEntity, "expected_updated_at must be an RFC3339 timestamp")
		}
		parsed = parsed.UTC()
		expected = &parsed
	}
	sets := project.Row{}
	var token string
	if in.Token != nil {
		token = strings.TrimSpace(*in.Token)
		if token == "" {
			return nil, statusErr(http.StatusUnprocessableEntity, "token cannot be empty")
		}
		sets["token"] = token
	}
	if in.CountryTag != nil {
		sets["country_tag"] = strings.ToUpper(strings.TrimSpace(*in.CountryTag))
	}
	if in.NameEn != nil {
		sets["name_en"] = *in.NameEn
	}
	if in.NameZh != nil {
		sets["name_zh"] = *in.NameZh
	}
	if in.DisplayNameLocs != nil {
		sets["display_name_locs"] = jsonValue(*in.DisplayNameLocs)
	}
	if in.ContextType != nil {
		contextType, err := normalizePanelContextType(*in.ContextType)
		if err != nil {
			return nil, err
		}
		sets["context_type"] = contextType
	}
	if in.ParentWindowToken != nil {
		sets["parent_window_token"] = strings.TrimSpace(*in.ParentWindowToken)
	}
	if in.VisibleTrigger != nil {
		sets["visible_trigger"] = *in.VisibleTrigger
	}
	if in.WindowProps != nil {
		sets["window_props"] = jsonValue(*in.WindowProps)
	}
	if in.Controls != nil {
		sets["controls"] = jsonValue(*in.Controls)
	}
	if in.RawExtras != nil {
		sets["raw_extras"] = jsonValue(*in.RawExtras)
	}
	row, err := writeRow(p, panelsTable, func(tx *project.Tx) (project.Row, error) {
		cur := tx.Find(panelsTable, id)
		if cur == nil {
			return nil, statusErr(http.StatusNotFound, panelMissing)
		}
		if in.Token != nil && taken(tx.Rows(panelsTable), id, func(row project.Row) bool { return text(row, "token") == token }) {
			return nil, statusErr(http.StatusConflict, fmt.Sprintf("token %s already exists in this project", pyRepr(token)))
		}
		if expected != nil {
			if actual, ok := parseTimestamp(cur["updated_at"]); !ok || !actual.Equal(*expected) {
				return nil, panelRevisionConflict(id, *expected, actual)
			}
		}

		source := textOr(cur, "source", "")
		if in.Source != nil {
			source = *in.Source
		}
		if source == "imported" {
			source = "created"
		}
		if in.Source != nil || cur["source"] != nil {
			sets["source"] = source
		}
		return update(tx, panelsTable, id, sets)
	})
	if err != nil {
		return nil, err
	}
	return panelFromRow(row), nil
}

func panelFiles(p *project.Project, id int64) (map[string]any, error) {
	row, err := getRow(p, panelsTable, id, panelMissing)
	if err != nil {
		return nil, err
	}
	panel := panelFromRow(row)
	record := p.Record()
	var languages []string
	decode(record["export_languages"], &languages)
	raw, err := json.Marshal(panel)
	if err != nil {
		return nil, err
	}
	var emitterRow map[string]any
	if err := json.Unmarshal(raw, &emitterRow); err != nil {
		return nil, err
	}
	guiText, txtText, err := exportengine.CompileUIPanel(emitterRow)
	if err != nil {
		return nil, err
	}
	names := derivePanelNames(panel.Token, "", false)
	tag := strings.ToUpper(strings.TrimSpace(text(record, "country_tag")))
	if tag == "" {
		tag = "mod"
	}
	loc := map[string]string{}
	locPaths := []string{}
	for _, lang := range emitmisc.BuildExportLangTriples(languages) {
		loc[lang.Code] = exportengine.GenerateUIPanelsLocalisation([]map[string]any{emitterRow}, lang.Code)
		locPaths = append(locPaths, "localisation/"+lang.Subdir+"/"+tag+"_ui_panels_"+lang.Suffix+".yml")
	}
	return map[string]any{
		"token": panel.Token, "window_name": names["window_name"],
		"gui_path": names["gui_path"], "gui_text": guiText,
		"txt_path": names["txt_path"], "txt_text": txtText,
		"loc": loc, "loc_paths": locPaths,
	}, nil
}
