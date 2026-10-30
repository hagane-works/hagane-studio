package focus

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/savediagnostics"
	"haganeworks/internal/project"
)

type SaveDiagnostic = savediagnostics.Diagnostic

type FocusFilePolicy struct {
	SourceFile     string   `json:"source_file"`
	OriginKind     string   `json:"origin_kind"`
	TreeIDs        []string `json:"tree_ids"`
	MatchedTreeIDs []string `json:"matched_tree_ids"`
	EmitMode       string   `json:"emit_mode"`
	TargetPath     string   `json:"target_path"`
	TargetLayer    string   `json:"target_layer"`
	DecisionSource string   `json:"decision_source"`
}

type FocusTree struct {
	EditRevision             string            `json:"edit_revision"`
	Diagnostics              *[]SaveDiagnostic `json:"diagnostics,omitempty"`
	ID                       int64             `json:"id"`
	ProjectID                int64             `json:"project_id"`
	TreeID                   string            `json:"tree_id"`
	CountryTag               *string           `json:"country_tag"`
	IsDefault                bool              `json:"is_default"`
	TreeKind                 string            `json:"tree_kind"`
	SharedFocusRefs          []map[string]any  `json:"shared_focus_refs"`
	SourceFile               string            `json:"source_file"`
	ContinuousFocusPositionX int               `json:"continuous_focus_position_x"`
	ContinuousFocusPositionY int               `json:"continuous_focus_position_y"`
	InitialShowPosition      map[string]any    `json:"initial_show_position"`
	InitialShowFocus         string            `json:"initial_show_focus"`
	ScopeKind                string            `json:"scope_kind"`
	SharedCountryTags        []string          `json:"shared_country_tags"`
	Source                   string            `json:"source"`
	FilePolicy               *FocusFilePolicy  `json:"file_policy"`
	RawExtras                map[string]any    `json:"raw_extras"`
	Nodes                    []FocusNode       `json:"nodes"`
	Warnings                 []string          `json:"warnings"`
}

type FocusNode struct {
	TreeEditRevision       string            `json:"tree_edit_revision"`
	Diagnostics            *[]SaveDiagnostic `json:"diagnostics,omitempty"`
	TreeDiagnostics        *[]SaveDiagnostic `json:"tree_diagnostics,omitempty"`
	ID                     int64             `json:"id"`
	FocusTreeID            int64             `json:"focus_tree_id"`
	FocusID                string            `json:"focus_id"`
	Name                   string            `json:"name"`
	Icon                   string            `json:"icon"`
	Cost                   float64           `json:"cost"`
	X                      int               `json:"x"`
	Y                      int               `json:"y"`
	RelativePositionID     string            `json:"relative_position_id"`
	Prerequisites          [][]string        `json:"prerequisites"`
	MutuallyExclusive      []string          `json:"mutually_exclusive"`
	CompletionReward       string            `json:"completion_reward"`
	Available              string            `json:"available"`
	Bypass                 string            `json:"bypass"`
	Cancel                 string            `json:"cancel"`
	SelectEffect           string            `json:"select_effect"`
	AiWillDo               string            `json:"ai_will_do"`
	SearchFilters          []string          `json:"search_filters"`
	CancelIfInvalid        bool              `json:"cancel_if_invalid"`
	ContinueIfInvalid      bool              `json:"continue_if_invalid"`
	AvailableIfCapitulated bool              `json:"available_if_capitulated"`
	WillLeadToWarWith      []string          `json:"will_lead_to_war_with"`
	AllowBranchAst         map[string]any    `json:"allow_branch_ast"`
	LoadFocusTreeRules     []map[string]any  `json:"load_focus_tree_rules"`
	NameEn                 string            `json:"name_en"`
	NameZh                 string            `json:"name_zh"`
	DescEn                 string            `json:"desc_en"`
	DescZh                 string            `json:"desc_zh"`
	DisplayNameLocs        map[string]string `json:"display_name_locs"`
	DescriptionLocs        map[string]string `json:"description_locs"`
	EffectTooltipLocs      map[string]string `json:"effect_tooltip_locs"`
	RawExtras              map[string]any    `json:"raw_extras"`
	Source                 string            `json:"source"`
	Offset                 []map[string]any  `json:"offset"`
	DynamicIcon            []map[string]any  `json:"dynamic_icon"`
	TextIcon               string            `json:"text_icon"`
	Warnings               []string          `json:"warnings"`

	sourceSnapshot map[string]any
}

func decodeInto(v any, target any) {
	if v == nil {
		return
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = json.Unmarshal(raw, target)
}

func plain(v any) any {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out any
	if decoder.Decode(&out) != nil {
		return nil
	}
	return out
}

func colStr(row project.Row, key, def string) string {
	switch v := row[key].(type) {
	case string:
		return v
	case nil:
		return def
	case json.Number:
		return v.String()
	}
	return def
}

func colInt(row project.Row, key string, def int) int {
	switch v := row[key].(type) {
	case nil:
		return def
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
		if f, err := v.Float64(); err == nil {
			return int(f)
		}
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}
	return def
}

func colFloat(row project.Row, key string, def float64) float64 {
	switch v := row[key].(type) {
	case nil:
		return def
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return def
}

func colBool(row project.Row, key string, def bool) bool {
	if v, ok := row[key].(bool); ok {
		return v
	}
	return def
}

func revisionOf(row project.Row) int64 {
	if n := project.Int64(row["edit_revision"]); n > 0 {
		return n
	}
	return 1
}

func formatRevision(n int64) string { return strconv.FormatInt(n, 10) }

func treeFromRow(row project.Row) FocusTree {
	t := FocusTree{
		EditRevision:             formatRevision(revisionOf(row)),
		ID:                       project.Int64(row["id"]),
		ProjectID:                project.Int64(row["project_id"]),
		TreeID:                   colStr(row, "tree_id", ""),
		IsDefault:                colBool(row, "is_default", false),
		TreeKind:                 colStr(row, "tree_kind", "primary"),
		SourceFile:               colStr(row, "source_file", ""),
		ContinuousFocusPositionX: colInt(row, "continuous_focus_position_x", 50),
		ContinuousFocusPositionY: colInt(row, "continuous_focus_position_y", 1000),
		InitialShowFocus:         colStr(row, "initial_show_focus", ""),
		ScopeKind:                colStr(row, "scope_kind", "own"),
		Source:                   colStr(row, "source", "created"),
	}
	if tag, ok := row["country_tag"].(string); ok {
		t.CountryTag = &tag
	}
	decodeInto(row["shared_focus_refs"], &t.SharedFocusRefs)
	if t.SharedFocusRefs == nil {
		t.SharedFocusRefs = []map[string]any{}
	}
	decodeInto(row["initial_show_position"], &t.InitialShowPosition)
	if len(t.InitialShowPosition) == 0 {
		t.InitialShowPosition = nil
	}
	decodeInto(row["shared_country_tags"], &t.SharedCountryTags)
	decodeInto(row["raw_extras"], &t.RawExtras)
	if t.RawExtras == nil {
		t.RawExtras = map[string]any{}
	}
	t.Nodes = []FocusNode{}
	t.Warnings = []string{}
	return t
}

func nodeFromRow(row project.Row, treeRevision string) FocusNode {
	n := FocusNode{
		TreeEditRevision:       treeRevision,
		ID:                     project.Int64(row["id"]),
		FocusTreeID:            project.Int64(row["focus_tree_id"]),
		FocusID:                colStr(row, "focus_id", ""),
		Name:                   colStr(row, "name", ""),
		Icon:                   colStr(row, "icon", "GFX_goal_unknown"),
		Cost:                   colFloat(row, "cost", 10.0),
		X:                      colInt(row, "x", 0),
		Y:                      colInt(row, "y", 0),
		RelativePositionID:     colStr(row, "relative_position_id", ""),
		CompletionReward:       colStr(row, "completion_reward", ""),
		Available:              colStr(row, "available", ""),
		Bypass:                 colStr(row, "bypass", ""),
		Cancel:                 colStr(row, "cancel", ""),
		SelectEffect:           colStr(row, "select_effect", ""),
		AiWillDo:               colStr(row, "ai_will_do", "factor = 1"),
		CancelIfInvalid:        colBool(row, "cancel_if_invalid", true),
		ContinueIfInvalid:      colBool(row, "continue_if_invalid", false),
		AvailableIfCapitulated: colBool(row, "available_if_capitulated", false),
		NameEn:                 colStr(row, "name_en", ""),
		NameZh:                 colStr(row, "name_zh", ""),
		DescEn:                 colStr(row, "desc_en", ""),
		DescZh:                 colStr(row, "desc_zh", ""),
		Source:                 colStr(row, "source", "created"),
		TextIcon:               colStr(row, "text_icon", ""),
	}
	decodeInto(row["prerequisites"], &n.Prerequisites)
	decodeInto(row["mutually_exclusive"], &n.MutuallyExclusive)
	decodeInto(row["search_filters"], &n.SearchFilters)
	decodeInto(row["will_lead_to_war_with"], &n.WillLeadToWarWith)
	decodeInto(row["allow_branch_ast"], &n.AllowBranchAst)
	decodeInto(row["load_focus_tree_rules"], &n.LoadFocusTreeRules)
	decodeInto(row["display_name_locs"], &n.DisplayNameLocs)
	decodeInto(row["description_locs"], &n.DescriptionLocs)
	decodeInto(row["effect_tooltip_locs"], &n.EffectTooltipLocs)
	decodeInto(row["raw_extras"], &n.RawExtras)
	decodeInto(row["source_snapshot"], &n.sourceSnapshot)
	decodeInto(row["offset"], &n.Offset)
	decodeInto(row["dynamic_icon"], &n.DynamicIcon)
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
	if n.Offset == nil {
		n.Offset = []map[string]any{}
	}
	if n.DynamicIcon == nil {
		n.DynamicIcon = []map[string]any{}
	}
	n.Warnings = []string{}
	return n
}

type treeData struct {
	trees    []project.Row
	nodes    []project.Row
	policies []project.Row
}

func (d treeData) tree(id int64) project.Row {
	for _, row := range d.trees {
		if project.Int64(row["id"]) == id {
			return row
		}
	}
	return nil
}

func (d treeData) nodesOf(treeID int64) []project.Row {
	out := []project.Row{}
	for _, row := range d.nodes {
		if project.Int64(row["focus_tree_id"]) == treeID {
			out = append(out, row)
		}
	}
	return out
}

func (d treeData) nodesByTree() map[int64][]project.Row {
	out := map[int64][]project.Row{}
	for _, row := range d.nodes {
		id := project.Int64(row["focus_tree_id"])
		out[id] = append(out[id], row)
	}
	return out
}

func (d treeData) sharedPoolNodes() []SharedLayoutNode {
	pools := map[int64]bool{}
	for _, tree := range d.trees {
		if colStr(tree, "tree_kind", "") == "shared_pool" {
			pools[project.Int64(tree["id"])] = true
		}
	}
	if len(pools) == 0 {
		return nil
	}
	var out []SharedLayoutNode
	for _, row := range d.nodes {
		if !pools[project.Int64(row["focus_tree_id"])] {
			continue
		}
		n := SharedLayoutNode{
			FocusID:            colStr(row, "focus_id", ""),
			RelativePositionID: colStr(row, "relative_position_id", ""),
			X:                  colInt(row, "x", 0),
			Y:                  colInt(row, "y", 0),
		}
		decodeInto(row["prerequisites"], &n.Prerequisites)
		decodeInto(row["offset"], &n.Offset)
		out = append(out, n)
	}
	return out
}

func sortByID(rows []project.Row) []project.Row {
	sort.SliceStable(rows, func(i, j int) bool { return project.Int64(rows[i]["id"]) < project.Int64(rows[j]["id"]) })
	return rows
}

func readData(p *project.Project) (treeData, error) {
	trees, err := p.Rows("focus_trees")
	if err != nil {
		return treeData{}, err
	}
	nodes, err := p.Rows("focus_nodes")
	if err != nil {
		return treeData{}, err
	}
	policies, err := p.Rows("focus_file_policies")
	if err != nil {
		return treeData{}, err
	}
	return treeData{trees: sortByID(trees), nodes: sortByID(nodes), policies: sortByID(policies)}, nil
}

func txData(tx *project.Tx) treeData {
	copyRows := func(rows []project.Row) []project.Row {
		return sortByID(append([]project.Row(nil), rows...))
	}
	return treeData{
		trees:    copyRows(tx.Rows("focus_trees")),
		nodes:    copyRows(tx.Rows("focus_nodes")),
		policies: copyRows(tx.Rows("focus_file_policies")),
	}
}

func treeResponse(d treeData, row project.Row) FocusTree {
	t := treeFromRow(row)
	for _, n := range d.nodesOf(t.ID) {
		t.Nodes = append(t.Nodes, nodeFromRow(n, t.EditRevision))
	}
	attachFilePolicies(d.policies, []*FocusTree{&t})
	attachFocusDiagnostics(&t, d.sharedPoolNodes())
	return t
}

func pyRepr(s string) string {
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		return `"` + s + `"`
	}
	return "'" + s + "'"
}

func strDefault(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}
