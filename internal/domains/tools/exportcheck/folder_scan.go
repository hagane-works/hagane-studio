package exportcheck

import (
	"context"
	"haganeworks/internal/project"
	"haganeworks/internal/schema"
	"strconv"
	"strings"
)

func (s *Store) tableColumns(_ context.Context, table string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, c := range schema.Columns(table) {
		out[c.Name] = true
	}
	return out, nil
}

func closureColumnsFor(spec contentEntitySpec, data *contentLanguageSpecs) []string {
	seen := map[string]bool{}
	cols := []string{}
	add := func(name string) {
		if name == "" || seen[name] || !sqlIdentifierRE.MatchString(name) {
			return
		}
		seen[name] = true
		cols = append(cols, name)
	}
	for _, f := range spec.LocaleFields {
		add(f)
	}
	for _, f := range spec.NestedLocaleFields {
		add(f)
	}
	for _, f := range spec.DisplayIDFields {
		add(f)
	}
	for _, child := range spec.Children {
		if len(child) == 2 {
			add(child[0])
		}
	}

	for _, extra := range []string{"id", "name", "description", "names", "agency_id",
		"bookmark_id", "token", "specialization_id", "project_token", "dm_id",
		"autonomy_id", "mio_id", "state_id", "tag", "replaces_vanilla_id",
		"role_type"} {
		add(extra)
	}
	return cols
}

func (s *Store) loadClosureRows(
	ctx context.Context, _ int64, table string,
	spec contentEntitySpec, data *contentLanguageSpecs, filterImported bool,
) ([]map[string]any, error) {
	present, _ := s.tableColumns(ctx, table)
	selected := []string{}
	for _, c := range closureColumnsFor(spec, data) {
		if present[c] {
			selected = append(selected, c)
		}
	}
	if len(selected) == 0 {
		return []map[string]any{}, nil
	}
	rows, err := s.rows(table)
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, raw := range rows {
		if filterImported && present["source"] {
			if src := colStr(raw, "source"); src != "" && src != "created" {
				continue
			}
		}
		plain := plainMap(raw)
		row := map[string]any{}
		for _, c := range selected {
			row[c] = plain[c]
		}
		out = append(out, row)
	}
	return out, nil
}

func (s *Store) attachFocusNodes(ctx context.Context, _ int64, trees []map[string]any) error {
	if len(trees) == 0 {
		return nil
	}
	present, _ := s.tableColumns(ctx, "focus_nodes")
	cols := []string{"focus_id"}
	for _, c := range []string{"display_name_locs", "description_locs", "effect_tooltip_locs"} {
		if present[c] {
			cols = append(cols, c)
		}
	}
	nodes, err := s.rows("focus_nodes")
	if err != nil {
		return err
	}
	byTree := map[int64][]map[string]any{}
	for _, raw := range nodes {
		plain := plainMap(raw)
		node := map[string]any{}
		for _, c := range cols {
			node[c] = plain[c]
		}
		byTree[colInt(raw, "focus_tree_id")] = append(byTree[colInt(raw, "focus_tree_id")], node)
	}
	for _, tree := range trees {
		id := int64(coerceInt(tree["id"]))
		if list := byTree[id]; len(list) > 0 {
			items := make([]any, 0, len(list))
			for _, n := range list {
				items = append(items, n)
			}
			tree["nodes"] = items
		}
	}
	return nil
}

func (s *Store) locrefDefinedKeys() (map[string]bool, error) {
	rows, err := s.rows("project_loc_entries")
	if err != nil {
		return nil, err
	}
	defined := map[string]bool{}
	for _, row := range rows {
		defined[colStr(row, "key")] = true
	}
	return defined, nil
}

func (s *Store) scanLocRefTable(
	ctx context.Context, _ int64, spec locrefSpec,
	idCol string, cols, extra []string, includeImported bool,
	sources *[]ScriptLocSource, defined map[string]bool,
) error {
	all := append(append([]string{}, cols...), extra...)
	present, _ := s.tableColumns(ctx, spec.Table)
	for _, c := range all {
		if sqlIdentifierRE.MatchString(c) && !present[c] {
			return nil
		}
	}
	if spec.HasSource && !present["source"] {
		return nil
	}
	rows, err := s.rows(spec.Table)
	if err != nil {
		return nil
	}
	for _, raw := range rows {
		plain := plainMap(raw)
		byName := map[string]any{}
		for _, c := range all {
			byName[c] = plain[c]
		}
		if spec.HasSource && !includeImported && colStr(raw, "source") == "imported" {
			continue
		}
		rowID := locrefRowID(spec.Kind, idCol, byName)
		for _, field := range spec.Fields {
			value := byName[field]
			if value == nil {
				continue
			}
			*sources = append(*sources, ScriptLocSource{
				EntityType: spec.Kind, EntityID: rowID, Field: field, Value: value,
			})
		}

		if anyTruthy(byName["display_name_locs"]) ||
			anyToString(byName["name_en"]) != "" || anyToString(byName["name_zh"]) != "" {
			defined[rowID] = true
		}
		if anyTruthy(byName["description_locs"]) ||
			anyToString(byName["desc_en"]) != "" || anyToString(byName["desc_zh"]) != "" {
			defined[rowID+"_desc"] = true
		}
		if spec.Kind == "decision" && anyTruthy(byName["effect_tooltip_locs"]) {
			defined[rowID+"_et_tt"] = true
		}
	}
	return nil
}

func (s *Store) scanFocusNodeLocRefs(
	_ context.Context, _ int64, includeImported bool,
	sources *[]ScriptLocSource, defined map[string]bool,
) error {
	trees, err := s.rows("focus_trees")
	if err != nil {
		return nil
	}
	treeSource := map[int64]string{}
	for _, t := range trees {
		treeSource[colInt(t, "id")] = colStr(t, "source")
	}
	nodes, err := s.rows("focus_nodes")
	if err != nil {
		return nil
	}
	fields := []string{"completion_reward", "available", "bypass", "cancel",
		"select_effect", "ai_will_do"}
	for _, raw := range nodes {
		tsrc, ok := treeSource[colInt(raw, "focus_tree_id")]
		if !ok {
			continue
		}
		if !includeImported && (tsrc == "imported" || colStr(raw, "source") == "imported") {
			continue
		}
		plain := plainMap(raw)
		nodeID := strings.TrimSpace(colStr(raw, "focus_id"))
		if nodeID == "" {
			nodeID = strconv.FormatInt(colInt(raw, "id"), 10)
		}
		for _, field := range fields {
			*sources = append(*sources, ScriptLocSource{
				EntityType: "focus", EntityID: nodeID, Field: field, Value: colStr(raw, field),
			})
		}
		if extras := plain["raw_extras"]; extras != nil {
			*sources = append(*sources, ScriptLocSource{
				EntityType: "focus", EntityID: nodeID, Field: "raw_extras", Value: extras,
			})
		}
		if anyTruthy(plain["display_name_locs"]) || colStr(raw, "name_en") != "" || colStr(raw, "name_zh") != "" {
			defined[nodeID] = true
		}
		if anyTruthy(plain["description_locs"]) || colStr(raw, "desc_en") != "" || colStr(raw, "desc_zh") != "" {
			defined[nodeID+"_desc"] = true
		}
		if anyTruthy(plain["effect_tooltip_locs"]) {
			defined[nodeID+"_et_tt"] = true
		}
	}
	return nil
}

func ProjectEntityCounts(p *project.Project) (map[string]int64, error) {
	counts := p.Counts()
	out := map[string]int64{}
	for _, table := range schema.Tables() {
		if nonContentTables[table] || !sqlIdentifierRE.MatchString(table) {
			continue
		}
		hasProjectID := false
		for _, c := range schema.Columns(table) {
			if c.Name == "project_id" {
				hasProjectID = true
				break
			}
		}
		if !hasProjectID {
			continue
		}
		key := table
		if alias, ok := countKeyAliases[table]; ok {
			key = alias
		}
		out[key] = counts[table]
	}
	return out, nil
}

func (s *Store) ExportValidateSummary(_ context.Context, _ int64) (map[string]int64, int64, error) {
	counts, err := ProjectEntityCounts(s.p)
	if err != nil {
		return nil, 0, err
	}
	trees, err := s.rows("focus_trees")
	if err != nil {
		return nil, 0, err
	}
	treeIDs := map[int64]bool{}
	for _, t := range trees {
		treeIDs[colInt(t, "id")] = true
	}
	nodes, err := s.rows("focus_nodes")
	if err != nil {
		return nil, 0, err
	}
	var n int64
	for _, node := range nodes {
		if treeIDs[colInt(node, "focus_tree_id")] {
			n++
		}
	}
	return counts, n, nil
}

func (s *Store) LoadAssetObsIndex(_ context.Context, _ int64) (AssetObsIndex, error) {
	idx := AssetObsIndex{ByKind: map[string]map[int64]string{}, Icons: map[string]string{}}
	kinds := map[string]bool{"idea_picture": true, "event_picture": true, "decision_icon": true,
		"tech_icon": true, "character_portrait": true}
	rows, err := s.rows("entity_obs_paths")
	if err != nil {
		return idx, err
	}
	for _, row := range rows {
		kind := colStr(row, "entity_type")
		if !kinds[kind] {
			continue
		}
		if idx.ByKind[kind] == nil {
			idx.ByKind[kind] = map[int64]string{}
		}
		idx.ByKind[kind][colInt(row, "entity_id")] = colStr(row, "obs_key")
	}
	icons, err := s.rows("project_obs_icons")
	if err != nil {
		return idx, err
	}
	for _, row := range icons {
		idx.Icons[colStr(row, "gfx_name")] = colStr(row, "obs_key")
	}
	return idx, nil
}

func locrefRowID(kind, idCol string, byName map[string]any) string {
	if idCol == locrefComposedID && kind == "super_event" {
		tag := strings.TrimSpace(anyToString(byName["country_tag"]))
		if tag == "" {
			tag = "SE"
		}
		return tag + "_se." + anyToString(byName["number"])
	}
	if v := strings.TrimSpace(anyToString(byName[idCol])); v != "" {
		return v
	}
	return anyToString(byName["id"])
}
