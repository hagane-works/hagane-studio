package lint

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/project"
)

func text(row project.Row, col string) string {
	switch v := row[col].(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func textPtr(row project.Row, col string) *string {
	if row[col] == nil {
		return nil
	}
	v := text(row, col)
	return &v
}

func boolOr(row project.Row, col string, def bool) bool {
	switch v := row[col].(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	case json.Number:
		return v.String() != "0"
	}
	return def
}

func rowID(row project.Row, col string) (int64, bool) {
	if row[col] == nil {
		return 0, false
	}
	return project.Int64(row[col]), true
}

func jsonMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok && m != nil {
		return m
	}
	return map[string]any{}
}

func anyToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case bool:
		if t {
			return "True"
		}
		return "False"
	}
	return ""
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

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func itoaInt(i int) string  { return fmt.Sprint(i) }
func itoa64(i int64) string { return fmt.Sprint(i) }
func sortedBy(rows []project.Row, key func(project.Row) string) {
	sort.SliceStable(rows, func(i, j int) bool { return key(rows[i]) < key(rows[j]) })
}

func loadPanels(p *project.Project) ([]Panel, []string, error) {
	rows, err := p.Rows("ui_panels")
	if err != nil {
		return nil, nil, err
	}
	sortedBy(rows, func(row project.Row) string { return text(row, "token") })
	panels := make([]Panel, 0, len(rows))
	for _, row := range rows {
		controls := anyToMapSlice(row["controls"])
		if controls == nil {
			controls = []map[string]any{}
		}
		panels = append(panels, Panel{
			CountryTag: text(row, "country_tag"), Token: text(row, "token"),
			NameEn: text(row, "name_en"), NameZh: text(row, "name_zh"),
			DisplayNameLocs: jsonMap(row["display_name_locs"]),
			VisibleTrigger:  text(row, "visible_trigger"),
			Controls:        controls, RawExtras: jsonMap(row["raw_extras"]),
		})
	}
	snippets, err := p.Rows("scripted_snippets")
	if err != nil {
		return nil, nil, err
	}
	names := make([]string, 0, len(snippets))
	for _, row := range snippets {
		names = append(names, text(row, "name"))
	}
	return panels, names, nil
}

func buildStartDateReport(p *project.Project) (map[string]any, error) {
	startDate := textPtr(p.Record(), "start_date")
	bookmarks, err := p.Rows("bookmarks")
	if err != nil {
		return nil, err
	}

	created := []project.Row{}
	for _, row := range bookmarks {
		if text(row, "source") == "created" {
			created = append(created, row)
		}
	}
	sortedBy(created, func(row project.Row) string { return text(row, "bookmark_id") })
	var structured []StructuredBookmarkInput
	for _, row := range created {
		bid := text(row, "bookmark_id")
		src := text(row, "source")
		if src == "" {
			src = "project_structured"
		}
		structured = append(structured, StructuredBookmarkInput{
			BookmarkID: bid, Date: text(row, "date"),
			Path:    "common/bookmarks/" + strings.TrimSpace(bid) + ".txt",
			Source:  src,
			Emitted: strings.TrimSpace(bid) != "",
		})
	}
	rawRows, err := p.Rows("project_raw_files")
	if err != nil {
		return nil, err
	}

	kept := []project.Row{}
	for _, row := range rawRows {
		if row["source"] == nil || text(row, "source") != "imported" {
			kept = append(kept, row)
		}
	}
	sortedBy(kept, func(row project.Row) string { return text(row, "dest_path") })
	var raws []PreservedRawBookmarkInput
	for _, row := range kept {
		source := text(row, "source")
		if source == "" {
			source = "project_preserved_raw"
		}
		raws = append(raws, PreservedRawBookmarkInput{
			Path: text(row, "dest_path"), Content: text(row, "content"), Source: source, Emitted: true,
		})
	}
	plan := BuildTimelineExportPlan(deref(startDate), structured, raws)
	findings := LintStartDates(plan)
	var startDateOut any
	if startDate != nil {
		startDateOut = *startDate
	}
	return LintFindingsReport(findings, map[string]any{
		"start_date":   startDateOut,
		"custom_start": IsCustomStart(deref(startDate)),
	}), nil
}

func defaultBookmark(p *project.Project) (date, bookmarkID string, err error) {
	rows, err := p.Rows("bookmarks")
	if err != nil {
		return "", "", err
	}
	for i, row := range rows {
		if i == 0 {
			date, bookmarkID = text(row, "date"), text(row, "bookmark_id")
		}
		if boolOr(row, "default_flag", false) {
			return text(row, "date"), text(row, "bookmark_id"), nil
		}
	}
	return date, bookmarkID, nil
}

func loadLogProjectIndex(p *project.Project) (*LogProjectIndex, error) {
	ix := NewLogProjectIndex()
	nilIfEmptyStr := func(v string) *string {
		if v == "" {
			return nil
		}
		return &v
	}

	trees, err := p.Rows("focus_trees")
	if err != nil {
		return nil, err
	}
	treeTag := map[int64]string{}
	for _, tree := range trees {
		id, _ := rowID(tree, "id")
		treeTag[id] = text(tree, "country_tag")
	}
	nodes, err := p.Rows("focus_nodes")
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		treeID, _ := rowID(node, "focus_tree_id")
		ctag, ok := treeTag[treeID]
		fid := text(node, "focus_id")
		if !ok || fid == "" {
			continue
		}
		ix.Focus[strings.ToLower(fid)] = LogEntityRef{
			Kind: "focus", EntityID: fid,
			Name: logRefName(text(node, "name_en"), text(node, "name"), fid), CountryTag: nilIfEmptyStr(ctag),
		}
	}

	events, err := p.Rows("events")
	if err != nil {
		return nil, err
	}
	for _, row := range events {
		eid := text(row, "event_id")
		if eid == "" {
			continue
		}
		ix.Event[strings.ToLower(eid)] = LogEntityRef{
			Kind: "event", EntityID: eid,
			Name: logRefName(text(row, "title_en"), eid), CountryTag: nilIfEmptyStr(text(row, "country_tag")),
		}
	}

	simple := []struct {
		table, idCol, kind string
		dst                map[string]LogEntityRef
		names              []string
	}{
		{"decisions", "decision_id", "decision", ix.Decision, []string{"name_en"}},
		{"decision_categories", "category_id", "decision", ix.DecisionCategory, []string{"name_en"}},
		{"characters", "tag", "character", ix.Character, []string{"name_en", "name"}},
		{"ideas", "idea_id", "idea", ix.Idea, []string{"name_en", "name"}},
		{"technologies", "tech_id", "technology", ix.Technology, []string{"name_en"}},
	}
	for _, sp := range simple {
		rows, err := p.Rows(sp.table)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			id := text(row, sp.idCol)
			if id == "" {
				continue
			}
			candidates := make([]string, 0, len(sp.names)+1)
			for _, col := range sp.names {
				candidates = append(candidates, text(row, col))
			}
			sp.dst[strings.ToLower(id)] = LogEntityRef{
				Kind: sp.kind, EntityID: id,
				Name: logRefName(append(candidates, id)...), CountryTag: nilIfEmptyStr(text(row, "country_tag")),
			}
		}
	}

	states, err := p.Rows("states")
	if err != nil {
		return nil, err
	}
	for _, row := range states {
		sid, ok := rowID(row, "state_id")
		if !ok {
			continue
		}
		idStr := fmt.Sprintf("%d", sid)
		ix.State[idStr] = LogEntityRef{
			Kind: "state", EntityID: idStr,
			Name: logRefName(text(row, "name"), "state "+idStr), CountryTag: nilIfEmptyStr(text(row, "owner")),
		}
	}

	history, err := p.Rows("history_files")
	if err != nil {
		return nil, err
	}
	for _, row := range history {
		ctag := text(row, "country_tag")
		if ctag == "" {
			continue
		}
		ct := ctag
		ix.HistoryFile[strings.ToLower(ctag)] = LogEntityRef{
			Kind: "history_file", EntityID: ctag,
			Name: logRefName(ctag), CountryTag: &ct,
		}
	}
	return ix, nil
}

func logRefName(candidates ...string) *string {
	for _, c := range candidates {
		if s := strings.TrimSpace(c); s != "" {
			return &s
		}
	}
	return nil
}
