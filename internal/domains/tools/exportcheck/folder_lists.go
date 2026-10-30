package exportcheck

import (
	"context"
	"encoding/json"
	"haganeworks/internal/project"
	"sort"
)

func plainMap(row project.Row) map[string]any {
	raw, err := json.Marshal(row)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func (s *Store) ListCharacters(_ context.Context, _ int64, countryTag string) ([]Character, error) {
	rows, err := s.rows("characters")
	if err != nil {
		return nil, err
	}
	out := []Character{}
	for _, row := range rows {
		if countryTag != "" && colStr(row, "country_tag") != countryTag {
			continue
		}

		c := Character{Skill: 1, AttackSkill: 1, DefenseSkill: 1, PlanningSkill: 1, LogisticsSkill: 1,
			ManeuveringSkill: 1, CoordinationSkill: 1, AdvisorCost: 150, AdvisorRemovalCost: -1,
			CanBeCaptured: true, OperativeSkillLevel: 1}
		decodeRow(row, &c)
		if c.Traits == nil {
			c.Traits = []string{}
		}
		if c.AdvisorTraits == nil {
			c.AdvisorTraits = []string{}
		}
		if c.ScientistData == nil {
			c.ScientistData = map[string]any{}
		}
		if c.OperativeTraits == nil {
			c.OperativeTraits = []string{}
		}
		if c.OperativeNationalities == nil {
			c.OperativeNationalities = []string{}
		}
		if c.RawExtras == nil {
			c.RawExtras = map[string]any{}
		}
		if c.DisplayNameLocs == nil {
			c.DisplayNameLocs = map[string]string{}
		}
		if c.DescriptionLocs == nil {
			c.DescriptionLocs = map[string]string{}
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *Store) ListEvents(_ context.Context, _ int64, namespaceID *int64, countryTag string) ([]Event, error) {
	rows, err := s.rows("events")
	if err != nil {
		return nil, err
	}
	out := []Event{}
	for _, row := range rows {
		if namespaceID != nil && colInt(row, "namespace_id") != *namespaceID {
			continue
		}
		if countryTag != "" && colStr(row, "country_tag") != countryTag {
			continue
		}
		var e Event
		decodeRow(row, &e)
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].EventNumber < out[j].EventNumber })
	return out, nil
}

func (s *Store) ListDecisions(_ context.Context, _ int64, categoryID *int64, countryTag string) ([]Decision, error) {
	rows, err := s.rows("decisions")
	if err != nil {
		return nil, err
	}
	out := []Decision{}
	for _, row := range rows {
		if categoryID != nil && colInt(row, "category_id") != *categoryID {
			continue
		}
		if countryTag != "" && colStr(row, "country_tag") != countryTag {
			continue
		}
		var d Decision
		decodeRow(row, &d)
		out = append(out, d)
	}
	return out, nil
}

func (s *Store) ListStates(_ context.Context, _ int64) ([]StateRow, error) {
	rows, err := s.rows("states")
	if err != nil {
		return nil, err
	}
	out := make([]StateRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, StateRow(plainMap(row)))
	}
	sort.SliceStable(out, func(i, j int) bool { return coerceInt(out[i]["state_id"]) < coerceInt(out[j]["state_id"]) })
	return out, nil
}

func (s *Store) ListHistoryFiles(_ context.Context, _ int64) ([]HistoryFile, error) {
	rows, err := s.rows("history_files")
	if err != nil {
		return nil, err
	}
	out := make([]HistoryFile, 0, len(rows))
	for _, row := range rows {
		out = append(out, HistoryFile(plainMap(row)))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return anyToString(out[i]["country_tag"]) < anyToString(out[j]["country_tag"])
	})
	return out, nil
}

func (s *Store) ListBookmarks(_ context.Context, _ int64) ([]Bookmark, error) {
	out, err := decodeRows[Bookmark](s, "bookmarks")
	if err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].SelectableCountries == nil {
			out[i].SelectableCountries = []map[string]any{}
		}
		if out[i].Filters == nil {
			out[i].Filters = []string{}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].BookmarkID < out[j].BookmarkID })
	return out, nil
}

func (s *Store) ListCountries(_ context.Context, _ int64) ([]Country, error) {
	return decodeRows[Country](s, "countries")
}

func (s *Store) ListSubUnits(_ context.Context, _ int64, countryTag, sourceFile *string) ([]SubUnit, error) {
	all, err := decodeRows[SubUnit](s, "sub_units")
	if err != nil {
		return nil, err
	}
	out := all[:0]
	for _, u := range all {
		if countryTag != nil && u.CountryTag != *countryTag || sourceFile != nil && u.SourceFile != *sourceFile {
			continue
		}
		out = append(out, u)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.SourceFile != b.SourceFile {
			return a.SourceFile < b.SourceFile
		}
		if a.Ordering != b.Ordering {
			return a.Ordering < b.Ordering
		}
		return a.SubUnitID < b.SubUnitID
	})
	return out, nil
}

func (s *Store) ListDivisionTemplates(_ context.Context, _ int64, countryTag, oobFile *string) ([]DivisionTemplate, error) {
	all, err := decodeRows[DivisionTemplate](s, "division_templates")
	if err != nil {
		return nil, err
	}
	out := all[:0]
	for _, d := range all {
		if countryTag != nil && d.CountryTag != *countryTag || oobFile != nil && d.OobFile != *oobFile {
			continue
		}
		out = append(out, d)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.OobFile != b.OobFile {
			return a.OobFile < b.OobFile
		}
		if a.Ordering != b.Ordering {
			return a.Ordering < b.Ordering
		}
		return a.TemplateName < b.TemplateName
	})
	return out, nil
}

func (s *Store) ListDoctrines(_ context.Context, _ int64, entityType string) ([]Doctrine, error) {
	all, err := decodeRows[Doctrine](s, "doctrines")
	if err != nil {
		return nil, err
	}
	out := all[:0]
	for _, d := range all {
		if entityType != "" && d.EntityType != entityType {
			continue
		}
		out = append(out, d)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].EntityType != out[j].EntityType {
			return out[i].EntityType < out[j].EntityType
		}
		return out[i].EntityKey < out[j].EntityKey
	})
	return out, nil
}

func (s *Store) ListAirWingGroups(_ context.Context, _ int64, oobFile *string) ([]AirWingGroup, error) {
	all, err := decodeRows[AirWingGroup](s, "oob_air_wing_groups")
	if err != nil {
		return nil, err
	}
	out := all[:0]
	for _, g := range all {
		if oobFile != nil && g.OobFile != *oobFile {
			continue
		}
		if g.Layout == nil {
			g.Layout = []map[string]any{}
		}
		if g.Warnings == nil {
			g.Warnings = []string{}
		}
		out = append(out, g)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Ordering < out[j].Ordering })
	return out, nil
}

func (s *Store) ListAirWings(_ context.Context, _ int64, groupID *int64) ([]AirWing, error) {
	all, err := decodeRows[AirWing](s, "oob_air_wings")
	if err != nil {
		return nil, err
	}
	out := all[:0]
	for _, w := range all {
		if groupID != nil && w.GroupID != *groupID {
			continue
		}
		out = append(out, w)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Ordering < out[j].Ordering })
	return out, nil
}

func (s *Store) ListFocusTrees(ctx context.Context, projectID int64, countryTag, treeKind string) ([]FocusTree, error) {
	treeRows, err := s.rows("focus_trees")
	if err != nil {
		return nil, err
	}
	nodeRows, err := s.rows("focus_nodes")
	if err != nil {
		return nil, err
	}
	nodes := map[int64][]FocusNode{}
	for _, row := range nodeRows {
		var n FocusNode
		decodeRow(row, &n)
		decode(colBytes(row, "source_snapshot"), &n.sourceSnapshot)
		if n.Prerequisites == nil {
			n.Prerequisites = [][]string{}
		}
		if n.MutuallyExclusive == nil {
			n.MutuallyExclusive = []string{}
		}
		if n.SearchFilters == nil {
			n.SearchFilters = []string{}
		}
		if n.WillLeadToWarWith == nil {
			n.WillLeadToWarWith = []string{}
		}
		if n.AllowBranchAst == nil {
			n.AllowBranchAst = map[string]any{}
		}
		if n.LoadFocusTreeRules == nil {
			n.LoadFocusTreeRules = []map[string]any{}
		}
		if n.DisplayNameLocs == nil {
			n.DisplayNameLocs = map[string]string{}
		}
		if n.DescriptionLocs == nil {
			n.DescriptionLocs = map[string]string{}
		}
		if n.EffectTooltipLocs == nil {
			n.EffectTooltipLocs = map[string]string{}
		}
		if n.RawExtras == nil {
			n.RawExtras = map[string]any{}
		}
		nodes[n.FocusTreeID] = append(nodes[n.FocusTreeID], n)
	}
	trees := []FocusTree{}
	for _, row := range treeRows {
		if countryTag != "" && colStr(row, "country_tag") != countryTag {
			continue
		}
		if treeKind != "" && colStr(row, "tree_kind") != treeKind {
			continue
		}
		var t FocusTree
		decodeRow(row, &t)
		t.FilePolicy = nil
		if t.SharedFocusRefs == nil {
			t.SharedFocusRefs = []map[string]any{}
		}
		if len(t.InitialShowPosition) == 0 {
			t.InitialShowPosition = nil
		}
		if t.RawExtras == nil {
			t.RawExtras = map[string]any{}
		}
		t.Nodes = nodes[t.ID]
		if t.Nodes == nil {
			t.Nodes = []FocusNode{}
		}
		t.Warnings = []string{}
		trees = append(trees, t)
	}
	policies, err := loadFocusPolicyRows(ctx, s, projectID)
	if err != nil {
		return nil, err
	}
	byTreeID := map[string]*FocusFilePolicy{}
	for i := range policies {
		for _, treeID := range policies[i].Policy.TreeIDs {
			if _, seen := byTreeID[treeID]; !seen {
				byTreeID[treeID] = &policies[i].Policy
			}
		}
	}
	for i := range trees {
		trees[i].FilePolicy = byTreeID[trees[i].TreeID]
	}
	return trees, nil
}

func (s *Store) loadGateTrees(ctx context.Context, projectID int64) ([]FocusTree, error) {
	return s.ListFocusTrees(ctx, projectID, "", "")
}

func (s *Store) loadGateIdeas(_ context.Context, _ int64) ([]Idea, error) {
	rows, err := s.rows("ideas")
	if err != nil {
		return nil, err
	}
	out := []Idea{}
	for _, row := range rows {
		i := Idea{ID: colInt(row, "id"), IdeaID: colStr(row, "idea_id"), IsDesigner: colBool(row, "is_designer"),
			Source: colStr(row, "source"), Name: colStr(row, "name")}
		decode(colBytes(row, "modifier"), &i.Modifier)
		decode(colBytes(row, "raw_extras"), &i.RawExtras)
		decode(colBytes(row, "research_bonus"), &i.ResearchBonus)
		out = append(out, i)
	}
	return out, nil
}

func (s *Store) loadGateTechs(_ context.Context, _ int64) ([]Technology, error) {
	rows, err := s.rows("technologies")
	if err != nil {
		return nil, err
	}
	out := []Technology{}
	for _, row := range rows {
		t := Technology{ID: colInt(row, "id"), TechID: colStr(row, "tech_id"), Folder: colStr(row, "folder"),
			Source: colStr(row, "source"), Name: colStr(row, "name"), StartYear: 1936}
		if row["start_year"] != nil {
			t.StartYear = int(colInt(row, "start_year"))
		}
		decode(colBytes(row, "prerequisites"), &t.Prerequisites)
		decode(colBytes(row, "folder_placements"), &t.FolderPlacements)
		decode(colBytes(row, "path"), &t.Path)
		out = append(out, t)
	}
	return out, nil
}

func (s *Store) loadGateRawFiles(_ context.Context, _ int64) ([]gateRawFile, error) {
	rows, err := s.rows("project_raw_files")
	if err != nil {
		return nil, err
	}
	out := []gateRawFile{}
	for _, row := range rows {
		out = append(out, gateRawFile{
			ID: colInt(row, "id"), DestPath: colStr(row, "dest_path"), Content: colStr(row, "content"),
			IsBinary: colBool(row, "is_binary"), ContentSHA256: colStr(row, "content_sha256"),
			GenerateReplacePath: colBool(row, "generate_replace_path"), Source: colStr(row, "source"),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DestPath < out[j].DestPath })
	return out, nil
}

func (s *Store) hasImportedRows(_ context.Context, _ int64) bool {
	for _, table := range []string{"project_raw_files", "characters"} {
		rows, err := s.rows(table)
		if err != nil {
			continue
		}
		for _, row := range rows {
			if colStr(row, "source") == "imported" {
				return true
			}
		}
	}
	return false
}
