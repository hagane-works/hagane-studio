package countries

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sort"
	"sync"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
)

var countryTagTables = []struct{ Label, Table string }{
	{"characters", "characters"},
	{"focus_trees", "focus_trees"},
	{"ideas", "ideas"},
	{"custom_traits", "custom_traits"},
	{"event_namespaces", "event_namespaces"},
	{"events", "events"},
	{"decision_categories", "decision_categories"},
	{"decisions", "decisions"},
	{"technologies", "technologies"},
	{"opinion_modifiers", "opinion_modifiers"},
	{"dynamic_modifiers", "dynamic_modifiers"},
	{"super_events", "super_events"},
	{"military_industrial_organizations", "military_industrial_organizations"},
	{"history_files", "history_files"},
}

var renameOnlyTagColumns = []struct{ Table, Column string }{
	{"states", "owner"},
	{"states", "controller"},
	{"autonomy_states", "country_tag"},
	{"ui_panels", "country_tag"},
	{"special_project_specializations", "country_tag"},
	{"special_projects", "country_tag"},
	{"equipments", "country_tag"},
	{"sub_units", "country_tag"},
	{"division_templates", "country_tag"},
	{"division_deployments", "country_tag"},
	{"oob_air_wing_groups", "country_tag"},
	{"oob_air_wings", "owner"},
	{"oob_fleets", "country_tag"},
	{"oob_ships", "owner"},
	{"intelligence_agencies", "country_tag"},
}

var stateTagListColumns = []string{"add_core_of", "add_claim_by", "remove_core_of"}

var sharedTagsTables = []string{
	"focus_trees", "ideas", "event_namespaces", "events",
	"decision_categories", "decisions", "military_industrial_organizations",
	"intelligence_agencies",
}

func renameTables() []string {
	seen := map[string]bool{}
	var out []string
	add := func(table string) {
		if !seen[table] {
			seen[table] = true
			out = append(out, table)
		}
	}
	add("countries")
	add("bookmarks")
	for _, t := range countryTagTables {
		add(t.Table)
	}
	for _, c := range renameOnlyTagColumns {
		add(c.Table)
	}
	for _, t := range sharedTagsTables {
		add(t)
	}
	return out
}

func renameCountryTagCascade(tx *project.Tx, rv *revisions, oldTag, newTag string) error {
	for _, row := range tx.Rows("focus_trees") {
		if str(row["country_tag"]) == oldTag || containsJSONString(row["shared_country_tags"], oldTag) {
			if err := rv.advance(tx, "focus_trees", project.Int64(row["id"])); err != nil {
				return err
			}
		}
	}
	for _, row := range tx.Rows("bookmarks") {
		if bookmarkReferences(row, oldTag) {
			if err := rv.advance(tx, "bookmarks", project.Int64(row["id"])); err != nil {
				return err
			}
		}
	}
	scalar := func(table, column string) {
		rewriteRows(tx, table, func(row project.Row) project.Row {
			if str(row[column]) != oldTag || row[column] == nil {
				return nil
			}
			next := cloneShallow(row)
			next[column] = newTag
			return next
		})
	}
	for _, t := range countryTagTables {
		scalar(t.Table, "country_tag")
	}
	for _, c := range renameOnlyTagColumns {
		scalar(c.Table, c.Column)
	}
	rewriteRows(tx, "states", func(row project.Row) project.Row {
		var next project.Row
		for _, column := range stateTagListColumns {
			if list, ok := renameTagList(row[column], oldTag, newTag); ok {
				if next == nil {
					next = cloneShallow(row)
				}
				next[column] = list
			}
		}
		if list, ok := renameObjectField(row["set_province_controller"], "controller", oldTag, newTag); ok {
			if next == nil {
				next = cloneShallow(row)
			}
			next["set_province_controller"] = list
		}
		return next
	})
	for _, table := range sharedTagsTables {
		rewriteRows(tx, table, func(row project.Row) project.Row {
			list, ok := renameTagList(row["shared_country_tags"], oldTag, newTag)
			if !ok {
				return nil
			}
			next := cloneShallow(row)
			next["shared_country_tags"] = list
			return next
		})
	}

	rewriteRows(tx, "bookmarks", func(row project.Row) project.Row {
		return renameBookmarkTags(row, oldTag, newTag)
	})
	return nil
}

func renameBookmarkTags(row project.Row, oldTag, newTag string) project.Row {
	if !bookmarkReferences(row, oldTag) {
		return nil
	}
	next := cloneShallow(row)
	if str(row["default_country"]) == oldTag {
		next["default_country"] = newTag
	}
	if list, ok := renameObjectField(row["selectable_countries"], "tag", oldTag, newTag); ok {
		next["selectable_countries"] = list
	}
	return next
}

func bookmarkReferences(row project.Row, tag string) bool {
	if str(row["default_country"]) == tag {
		return true
	}
	list, _ := row["selectable_countries"].([]any)
	for _, item := range list {
		if entry, ok := item.(map[string]any); ok {
			if value, ok := entry["tag"].(string); ok && value == tag {
				return true
			}
		}
	}
	return false
}

func containsJSONString(value any, want string) bool {
	list, _ := value.([]any)
	for _, item := range list {
		if s, ok := item.(string); ok && s == want {
			return true
		}
	}
	return false
}

func renameTagList(value any, oldTag, newTag string) ([]any, bool) {
	if !containsJSONString(value, oldTag) {
		return nil, false
	}
	list := value.([]any)
	out := make([]any, 0, len(list))
	seen := map[string]bool{}
	for _, item := range list {
		if s, ok := item.(string); ok && s == oldTag {
			item = newTag
		}
		key, _ := json.Marshal(item)
		if seen[string(key)] {
			continue
		}
		seen[string(key)] = true
		out = append(out, item)
	}
	return out, true
}

func renameObjectField(value any, field, oldTag, newTag string) ([]any, bool) {
	list, ok := value.([]any)
	if !ok {
		return nil, false
	}
	changed := false
	out := make([]any, len(list))
	for i, item := range list {
		out[i] = item
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if s, ok := entry[field].(string); ok && s == oldTag {
			next := make(map[string]any, len(entry))
			for k, v := range entry {
				next[k] = v
			}
			next[field] = newTag
			out[i] = next
			changed = true
		}
	}
	return out, changed
}

func rewriteRows(tx *project.Tx, table string, change func(project.Row) project.Row) {
	rows := tx.Rows(table)
	var out []project.Row
	for i, row := range rows {
		next := change(row)
		if next == nil {
			if out != nil {
				out = append(out, row)
			}
			continue
		}
		if out == nil {
			out = make([]project.Row, 0, len(rows))
			out = append(out, rows[:i]...)
		}
		out = append(out, next)
	}
	if out != nil {
		tx.Replace(table, out)
	}
}

func cloneShallow(row project.Row) project.Row {
	out := make(project.Row, len(row))
	for k, v := range row {
		out[k] = v
	}
	return out
}

func anyMap(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}

func stringMapOf(value any) map[string]string {
	out := map[string]string{}
	m, _ := value.(map[string]any)
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func nestedStringMapOf(value any) map[string]map[string]string {
	out := map[string]map[string]string{}
	m, _ := value.(map[string]any)
	for k, v := range m {
		out[k] = stringMapOf(v)
	}
	return out
}

func sortStrings(values []string) { sort.Strings(values) }

func decodeJSON(r *http.Request, v any, allowEmpty bool) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, apiutil.MaxJSONBody))
	err := decoder.Decode(v)
	if errors.Is(err, io.EOF) && allowEmpty {
		return nil
	}
	return err
}

func decodeStrict(r *http.Request, v any, limit int64) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, limit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

type syncMap struct {
	mu sync.Mutex
	m  map[string]string
}

func (s *syncMap) get(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[key]
}

func (s *syncMap) set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[string]string{}
	}
	s.m[key] = value
}

var logf = log.Printf
