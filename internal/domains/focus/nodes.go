package focus

import (
	"fmt"
	"strings"

	"haganeworks/internal/hoi4/savediagnostics"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const (
	maxPersistedFocusNodeID       = 2_147_483_647
	maxFocusNodeCollisionAttempts = 1000
	focusIDConflictError          = "error"
	focusIDConflictSuffix         = "suffix"
)

var focusWriteTables = []string{"focus_trees", "focus_nodes"}

func focusIDExistsDetail(focusID string) string {
	return fmt.Sprintf("focus_id %s already exists in this project", pyRepr(focusID))
}

func checkPersistedFocusNodeID(nodeID int64) error {
	if nodeID <= 0 || nodeID > maxPersistedFocusNodeID {
		return errNodeNotPersisted
	}
	return nil
}

type mutation struct {
	tx     *project.Tx
	tree   FocusTree
	before []SaveDiagnostic
}

func beginNodeMutation(tx *project.Tx, rv *revisions, treeID int64) (*mutation, error) {
	row := tx.Find("focus_trees", treeID)
	if row == nil {
		return nil, errTreeMissing
	}
	tree := treeFromRow(row)
	if _, err := rv.advance(tx, treeID); err != nil {
		return nil, err
	}
	return &mutation{tx: tx, tree: tree}, nil
}

func (m *mutation) evaluateBefore() {
	m.before = treeDiagnostics(txData(m.tx), m.tree)
}

func (m *mutation) after() []SaveDiagnostic {
	current := treeDiagnostics(txData(m.tx), m.tree)
	return savediagnostics.Compare(m.before, current)
}

func (m *mutation) node(id int64) FocusNode {
	revision := formatRevision(revisionOf(m.tx.Find("focus_trees", m.tree.ID)))
	return nodeFromRow(m.tx.Find("focus_nodes", id), revision)
}

func (m *mutation) markTreeEdited() error {
	if m.tree.Source != "imported" {
		return nil
	}
	_, err := m.tx.Update("focus_trees", m.tree.ID, func(r project.Row) { r["source"] = "created" })
	return err
}

func (m *mutation) treeNodes() []project.Row { return txData(m.tx).nodesOf(m.tree.ID) }

func focusIDTaken(tx *project.Tx, focusID string, except int64) bool {
	for _, row := range tx.Rows("focus_nodes") {
		if project.Int64(row["id"]) != except && colStr(row, "focus_id", "") == focusID {
			if tx.Find("focus_trees", project.Int64(row["focus_tree_id"])) != nil {
				return true
			}
		}
	}
	return false
}

type FocusNodeCreateInput struct {
	FocusID                string            `json:"focus_id"`
	Name                   *string           `json:"name"`
	Icon                   *string           `json:"icon"`
	Cost                   *float64          `json:"cost"`
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
	AiWillDo               *string           `json:"ai_will_do"`
	SearchFilters          []string          `json:"search_filters"`
	CancelIfInvalid        *bool             `json:"cancel_if_invalid"`
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
	Source                 *string           `json:"source"`
	Offset                 []map[string]any  `json:"offset"`
	DynamicIcon            []map[string]any  `json:"dynamic_icon"`
	TextIcon               string            `json:"text_icon"`

	OnConflict string `json:"on_conflict"`
}

func (in *FocusNodeCreateInput) normalize() {
	if in.Name == nil {
		v := "New Focus"
		in.Name = &v
	}
	if in.Icon == nil {
		v := "GFX_goal_unknown"
		in.Icon = &v
	}
	if in.Cost == nil {
		v := 10.0
		in.Cost = &v
	}
	if in.AiWillDo == nil {
		v := "factor = 1"
		in.AiWillDo = &v
	}
	if in.CancelIfInvalid == nil {
		v := true
		in.CancelIfInvalid = &v
	}
	if in.Prerequisites == nil {
		in.Prerequisites = [][]string{}
	}
	if in.MutuallyExclusive == nil {
		in.MutuallyExclusive = []string{}
	}
	if in.SearchFilters == nil {
		in.SearchFilters = []string{}
	}
	if in.WillLeadToWarWith == nil {
		in.WillLeadToWarWith = []string{}
	}
	if in.AllowBranchAst == nil {
		in.AllowBranchAst = map[string]any{}
	}
	if in.LoadFocusTreeRules == nil {
		in.LoadFocusTreeRules = []map[string]any{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	if in.EffectTooltipLocs == nil {
		in.EffectTooltipLocs = map[string]string{}
	}
	if in.Offset == nil {
		in.Offset = []map[string]any{}
	}
	if in.DynamicIcon == nil {
		in.DynamicIcon = []map[string]any{}
	}
}

func createNode(p *project.Project, rv *revisions, treeID int64, in FocusNodeCreateInput) (FocusNode, error) {
	in.normalize()
	if err := canonicalBlockTextFocusCreate(&in); err != nil {
		return FocusNode{}, err
	}
	var node FocusNode
	err := p.Write(focusWriteTables, func(tx *project.Tx) error {
		m, err := beginNodeMutation(tx, rv, treeID)
		if err != nil {
			return err
		}
		onConflict := strings.TrimSpace(in.OnConflict)
		switch onConflict {
		case "", focusIDConflictError, focusIDConflictSuffix:
		default:
			return DetailError{Status: 422, Detail: fmt.Sprintf(
				"on_conflict must be %q or %q, got %s", focusIDConflictError, focusIDConflictSuffix, pyRepr(onConflict))}
		}
		focusID := strings.TrimSpace(in.FocusID)

		suffixOnCollision := focusID == "" || onConflict == focusIDConflictSuffix
		if focusID == "" {
			tag := "None"
			if m.tree.CountryTag != nil {
				tag = *m.tree.CountryTag
			}
			focusID = fmt.Sprintf("%s_%s", tag, strings.TrimSpace(strings.ReplaceAll(*in.Name, " ", "_")))
		}
		original := focusID
		suffix := 0
		for focusIDTaken(tx, focusID, 0) {
			if !suffixOnCollision {
				return DetailError{Status: 409, Detail: focusIDExistsDetail(original)}
			}
			suffix++
			if suffix > maxFocusNodeCollisionAttempts {
				return DetailError{Status: 500, Detail: fmt.Sprintf(
					"focus_id auto-suffix exhausted %d attempts starting at %s · pick manually.",
					maxFocusNodeCollisionAttempts, pyRepr(original))}
			}
			focusID = fmt.Sprintf("%s_%d", original, suffix)
		}
		m.evaluateBefore()

		nameEn := in.NameEn
		if nameEn == "" {
			nameEn = *in.Name
		}
		source := strDefault(in.Source, "created")
		row := tx.Insert("focus_nodes", project.Row{
			"focus_tree_id": treeID, "focus_id": focusID, "name": *in.Name, "icon": *in.Icon,
			"cost": *in.Cost, "x": in.X, "y": in.Y, "relative_position_id": in.RelativePositionID,
			"prerequisites": plain(in.Prerequisites), "mutually_exclusive": plain(in.MutuallyExclusive),
			"completion_reward": in.CompletionReward, "available": in.Available, "bypass": in.Bypass,
			"cancel": in.Cancel, "select_effect": in.SelectEffect, "ai_will_do": *in.AiWillDo,
			"search_filters": plain(in.SearchFilters), "cancel_if_invalid": *in.CancelIfInvalid,
			"continue_if_invalid": in.ContinueIfInvalid, "available_if_capitulated": in.AvailableIfCapitulated,
			"will_lead_to_war_with": plain(in.WillLeadToWarWith), "allow_branch_ast": plain(in.AllowBranchAst),
			"load_focus_tree_rules": plain(in.LoadFocusTreeRules), "raw_extras": map[string]any{},
			"name_en": nameEn, "name_zh": in.NameZh, "desc_en": in.DescEn, "desc_zh": in.DescZh,
			"display_name_locs": plain(in.DisplayNameLocs), "description_locs": plain(in.DescriptionLocs),
			"effect_tooltip_locs": plain(in.EffectTooltipLocs), "source": source, "source_snapshot": nil,
			"offset": plain(in.Offset), "dynamic_icon": plain(in.DynamicIcon), "text_icon": in.TextIcon,
		})

		if source != "imported" {
			if err := m.markTreeEdited(); err != nil {
				return err
			}
		}
		node = m.node(project.Int64(row["id"]))
		if suffix > 0 {
			node.Warnings = []string{fmt.Sprintf("focus_id %s collided · auto-suffixed to %s.", pyRepr(original), pyRepr(focusID))}
		}
		diagnostics := m.after()
		node.Diagnostics = focusRowDiagnostics(diagnostics, node)
		node.TreeDiagnostics = currentTreeDiagnostics(diagnostics)
		return nil
	})
	if err != nil {
		return FocusNode{}, err
	}
	rv.commit(p)
	return node, nil
}

type FocusNodePatch struct {
	FocusID                workbench.Optional[string]            `json:"focus_id"`
	Name                   workbench.Optional[string]            `json:"name"`
	Icon                   workbench.Optional[string]            `json:"icon"`
	Cost                   workbench.Optional[float64]           `json:"cost"`
	X                      workbench.Optional[int]               `json:"x"`
	Y                      workbench.Optional[int]               `json:"y"`
	RelativePositionID     workbench.Optional[string]            `json:"relative_position_id"`
	Prerequisites          workbench.Optional[[][]string]        `json:"prerequisites"`
	MutuallyExclusive      workbench.Optional[[]string]          `json:"mutually_exclusive"`
	CompletionReward       workbench.Optional[string]            `json:"completion_reward"`
	Available              workbench.Optional[string]            `json:"available"`
	Bypass                 workbench.Optional[string]            `json:"bypass"`
	Cancel                 workbench.Optional[string]            `json:"cancel"`
	SelectEffect           workbench.Optional[string]            `json:"select_effect"`
	AiWillDo               workbench.Optional[string]            `json:"ai_will_do"`
	SearchFilters          workbench.Optional[[]string]          `json:"search_filters"`
	CancelIfInvalid        workbench.Optional[bool]              `json:"cancel_if_invalid"`
	ContinueIfInvalid      workbench.Optional[bool]              `json:"continue_if_invalid"`
	AvailableIfCapitulated workbench.Optional[bool]              `json:"available_if_capitulated"`
	WillLeadToWarWith      workbench.Optional[[]string]          `json:"will_lead_to_war_with"`
	AllowBranchAst         workbench.Optional[map[string]any]    `json:"allow_branch_ast"`
	LoadFocusTreeRules     workbench.Optional[[]map[string]any]  `json:"load_focus_tree_rules"`
	NameEn                 workbench.Optional[string]            `json:"name_en"`
	NameZh                 workbench.Optional[string]            `json:"name_zh"`
	DescEn                 workbench.Optional[string]            `json:"desc_en"`
	DescZh                 workbench.Optional[string]            `json:"desc_zh"`
	DisplayNameLocs        workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs        workbench.Optional[map[string]string] `json:"description_locs"`
	EffectTooltipLocs      workbench.Optional[map[string]string] `json:"effect_tooltip_locs"`
	RawExtras              workbench.Optional[map[string]any]    `json:"raw_extras"`
	Source                 workbench.Optional[string]            `json:"source"`
	Offset                 workbench.Optional[[]map[string]any]  `json:"offset"`
	DynamicIcon            workbench.Optional[[]map[string]any]  `json:"dynamic_icon"`
	TextIcon               workbench.Optional[string]            `json:"text_icon"`
}

func (p FocusNodePatch) apply(row project.Row) {
	setStr := func(o workbench.Optional[string], column string, trim bool) {
		if !o.Set {
			return
		}
		v := optStr(o)
		if trim {
			v = strings.TrimSpace(v)
		}
		row[column] = v
	}
	setValue := func(set bool, column string, value any) {
		if set {
			row[column] = value
		}
	}
	setStr(p.FocusID, "focus_id", true)
	setStr(p.Name, "name", false)
	setStr(p.Icon, "icon", false)
	setValue(p.Cost.Set, "cost", optValue(p.Cost))
	setValue(p.X.Set, "x", optValue(p.X))
	setValue(p.Y.Set, "y", optValue(p.Y))
	setStr(p.RelativePositionID, "relative_position_id", false)
	setValue(p.Prerequisites.Set, "prerequisites", optValue(p.Prerequisites))
	setValue(p.MutuallyExclusive.Set, "mutually_exclusive", optValue(p.MutuallyExclusive))
	setStr(p.CompletionReward, "completion_reward", false)
	setStr(p.Available, "available", false)
	setStr(p.Bypass, "bypass", false)
	setStr(p.Cancel, "cancel", false)
	setStr(p.SelectEffect, "select_effect", false)
	setStr(p.AiWillDo, "ai_will_do", false)
	setValue(p.SearchFilters.Set, "search_filters", optValue(p.SearchFilters))
	setValue(p.CancelIfInvalid.Set, "cancel_if_invalid", optValue(p.CancelIfInvalid))
	setValue(p.ContinueIfInvalid.Set, "continue_if_invalid", optValue(p.ContinueIfInvalid))
	setValue(p.AvailableIfCapitulated.Set, "available_if_capitulated", optValue(p.AvailableIfCapitulated))
	setValue(p.WillLeadToWarWith.Set, "will_lead_to_war_with", optValue(p.WillLeadToWarWith))
	setValue(p.AllowBranchAst.Set, "allow_branch_ast", optValue(p.AllowBranchAst))
	setValue(p.LoadFocusTreeRules.Set, "load_focus_tree_rules", optValue(p.LoadFocusTreeRules))
	setStr(p.NameEn, "name_en", false)
	setStr(p.NameZh, "name_zh", false)
	setStr(p.DescEn, "desc_en", false)
	setStr(p.DescZh, "desc_zh", false)
	setValue(p.DisplayNameLocs.Set, "display_name_locs", optValue(p.DisplayNameLocs))
	setValue(p.DescriptionLocs.Set, "description_locs", optValue(p.DescriptionLocs))
	setValue(p.EffectTooltipLocs.Set, "effect_tooltip_locs", optValue(p.EffectTooltipLocs))
	setValue(p.RawExtras.Set, "raw_extras", optValue(p.RawExtras))
	setStr(p.Source, "source", false)
	setValue(p.Offset.Set, "offset", optValue(p.Offset))
	setValue(p.DynamicIcon.Set, "dynamic_icon", optValue(p.DynamicIcon))
	setStr(p.TextIcon, "text_icon", false)
}

func updateNode(p *project.Project, rv *revisions, treeID, nodeID int64, patch FocusNodePatch) (FocusNode, error) {
	if err := checkPersistedFocusNodeID(nodeID); err != nil {
		return FocusNode{}, err
	}
	if err := canonicalBlockTextFocusPatch(&patch); err != nil {
		return FocusNode{}, err
	}
	var node FocusNode
	err := p.Write(focusWriteTables, func(tx *project.Tx) error {
		m, err := beginNodeMutation(tx, rv, treeID)
		if err != nil {
			return err
		}
		target := tx.Find("focus_nodes", nodeID)
		if target == nil || project.Int64(target["focus_tree_id"]) != treeID {
			return errNodeMissing
		}
		if patch.FocusID.Set {
			if patch.FocusID.Value == nil || strings.TrimSpace(*patch.FocusID.Value) == "" {
				return DetailError{Status: 400, Detail: "focus_id cannot be empty"}
			}
			requested := strings.TrimSpace(*patch.FocusID.Value)
			if requested != colStr(target, "focus_id", "") && focusIDTaken(tx, requested, nodeID) {
				return DetailError{Status: 409, Detail: focusIDExistsDetail(requested)}
			}
		}
		m.evaluateBefore()
		if _, err := tx.Update("focus_nodes", nodeID, patch.apply); err != nil {
			return err
		}

		if err := m.markTreeEdited(); err != nil {
			return err
		}
		node = m.node(nodeID)
		diagnostics := m.after()
		node.Diagnostics = focusRowDiagnostics(diagnostics, node)
		node.TreeDiagnostics = currentTreeDiagnostics(diagnostics)
		return nil
	})
	if err != nil {
		return FocusNode{}, err
	}
	rv.commit(p)
	return node, nil
}

type FocusNodePosition struct {
	ID int64 `json:"id"`
	X  int   `json:"x"`
	Y  int   `json:"y"`
}

func bulkPositions(p *project.Project, rv *revisions, treeID int64, positions []FocusNodePosition) ([]FocusNode, error) {
	out := []FocusNode{}
	err := p.Write(focusWriteTables, func(tx *project.Tx) error {
		m, err := beginNodeMutation(tx, rv, treeID)
		if err != nil {
			return err
		}
		if len(positions) == 0 {
			delete(rv.advanced, treeID)
			return errNoChange
		}
		wanted := map[int64]FocusNodePosition{}
		for _, pos := range positions {
			wanted[pos.ID] = pos
		}
		var matched []int64
		for _, row := range m.treeNodes() {
			if _, ok := wanted[project.Int64(row["id"])]; ok {
				matched = append(matched, project.Int64(row["id"]))
			}
		}
		if len(matched) == 0 {
			delete(rv.advanced, treeID)
			return errNoChange
		}
		m.evaluateBefore()
		for _, id := range matched {
			pos := wanted[id]
			if _, err := tx.Update("focus_nodes", id, func(r project.Row) { r["x"], r["y"] = pos.X, pos.Y }); err != nil {
				return err
			}
		}
		if err := m.markTreeEdited(); err != nil {
			return err
		}
		diagnostics := m.after()
		for i, id := range matched {
			node := m.node(id)
			node.Diagnostics = focusRowDiagnostics(diagnostics, node)
			if i == 0 {
				node.TreeDiagnostics = currentTreeDiagnostics(diagnostics)
			}
			out = append(out, node)
		}
		return nil
	})
	if err == errNoChange {
		return []FocusNode{}, nil
	}
	if err != nil {
		return nil, err
	}
	rv.commit(p)
	return out, nil
}

var errNoChange = fmt.Errorf("no change")

func deleteNode(p *project.Project, rv *revisions, treeID, nodeID int64) error {
	if err := checkPersistedFocusNodeID(nodeID); err != nil {
		return err
	}
	err := p.Write(focusWriteTables, func(tx *project.Tx) error {
		m, err := beginNodeMutation(tx, rv, treeID)
		if err != nil {
			return err
		}
		removedRow := tx.Find("focus_nodes", nodeID)
		if removedRow == nil || project.Int64(removedRow["focus_tree_id"]) != treeID {
			return errNodeMissing
		}
		removed := nodeFromRow(removedRow, "")
		if removed.FocusID != "" {
			for _, siblingRow := range m.treeNodes() {
				siblingID := project.Int64(siblingRow["id"])
				if siblingID == nodeID {
					continue
				}
				sibling := nodeFromRow(siblingRow, "")
				changes := project.Row{}
				changedPrereqs := false
				newPrereqs := [][]string{}
				for _, group := range sibling.Prerequisites {
					stripped := []string{}
					for _, ref := range group {
						if ref != removed.FocusID {
							stripped = append(stripped, ref)
						}
					}
					if len(stripped) != len(group) {
						changedPrereqs = true
					}
					if len(stripped) > 0 {
						newPrereqs = append(newPrereqs, stripped)
					}
				}
				if changedPrereqs {
					changes["prerequisites"] = plain(newPrereqs)
				}
				inMutex := false
				for _, ref := range sibling.MutuallyExclusive {
					if ref == removed.FocusID {
						inMutex = true
						break
					}
				}
				if inMutex {
					stripped := []string{}
					for _, ref := range sibling.MutuallyExclusive {
						if ref != removed.FocusID {
							stripped = append(stripped, ref)
						}
					}
					changes["mutually_exclusive"] = plain(stripped)
				}
				if sibling.RelativePositionID == removed.FocusID {
					changes["relative_position_id"] = removed.RelativePositionID
					changes["x"] = sibling.X + removed.X
					changes["y"] = sibling.Y + removed.Y
				}
				if len(changes) > 0 {
					if _, err := tx.Update("focus_nodes", siblingID, func(r project.Row) {
						for k, v := range changes {
							r[k] = v
						}
					}); err != nil {
						return err
					}
				}
			}
		}
		tx.Delete("focus_nodes", func(r project.Row) bool { return project.Int64(r["id"]) == nodeID })

		return m.markTreeEdited()
	})
	if err != nil {
		return err
	}
	rv.commit(p)
	return nil
}

func forkNode(p *project.Project, rv *revisions, treeID, nodeID int64, newFocusID string) (FocusNode, error) {
	if err := checkPersistedFocusNodeID(nodeID); err != nil {
		return FocusNode{}, err
	}
	var node FocusNode
	err := p.Write(focusWriteTables, func(tx *project.Tx) error {
		srcRow := tx.Find("focus_nodes", nodeID)
		if srcRow == nil || project.Int64(srcRow["focus_tree_id"]) != treeID {
			return errSourceMissing
		}
		if tx.Find("focus_trees", treeID) == nil {
			return errTreeNotInProject
		}
		m, err := beginNodeMutation(tx, rv, treeID)
		if err != nil {
			return err
		}
		trimmed := strings.TrimSpace(newFocusID)
		if trimmed == "" {
			return DetailError{Status: 400, Detail: "new_focus_id cannot be empty"}
		}
		if trimmed == colStr(srcRow, "focus_id", "") {
			return DetailError{Status: 400, Detail: "new_focus_id must differ from the original"}
		}
		if focusIDTaken(tx, trimmed, 0) {
			return DetailError{Status: 409, Detail: focusIDExistsDetail(trimmed)}
		}
		m.evaluateBefore()
		copied := project.Row{}
		for column, value := range srcRow {
			switch column {
			case "id", "created_at", "updated_at", "source", "focus_id":
				continue
			}
			copied[column] = plain(value)
		}
		copied["focus_id"] = trimmed
		copied["source"] = "created"
		copied["x"] = colInt(srcRow, "x", 0) + 2
		if srcRow["x"] == nil {
			copied["x"] = nil
		}
		row := tx.Insert("focus_nodes", copied)
		if err := m.markTreeEdited(); err != nil {
			return err
		}
		node = m.node(project.Int64(row["id"]))
		diagnostics := m.after()
		node.Diagnostics = focusRowDiagnostics(diagnostics, node)
		node.TreeDiagnostics = currentTreeDiagnostics(diagnostics)
		return nil
	})
	if err != nil {
		return FocusNode{}, err
	}
	rv.commit(p)
	return node, nil
}

func setMutualExclusion(p *project.Project, rv *revisions, treeID, nodeID, otherNodeID int64, enabled bool) ([]FocusNode, error) {
	if err := checkPersistedFocusNodeID(nodeID); err != nil {
		return nil, err
	}
	if err := checkPersistedFocusNodeID(otherNodeID); err != nil {
		return nil, err
	}
	if nodeID == otherNodeID {
		return nil, DetailError{Status: 422, Detail: "mutual exclusion requires two distinct focus nodes"}
	}
	var out []FocusNode
	err := p.Write(focusWriteTables, func(tx *project.Tx) error {
		m, err := beginNodeMutation(tx, rv, treeID)
		if err != nil {
			return err
		}
		left, right := tx.Find("focus_nodes", nodeID), tx.Find("focus_nodes", otherNodeID)
		if left == nil || right == nil || project.Int64(left["focus_tree_id"]) != treeID || project.Int64(right["focus_tree_id"]) != treeID {
			return errNodeMissing
		}
		m.evaluateBefore()
		setMember := func(existing []string, target string) []string {
			out := []string{}
			found := false
			for _, id := range existing {
				if id != target {
					out = append(out, id)
				} else if enabled && !found {
					out = append(out, id)
					found = true
				}
			}
			if enabled && !found {
				out = append(out, target)
			}
			return out
		}
		leftNode, rightNode := nodeFromRow(left, ""), nodeFromRow(right, "")
		updates := map[int64][]string{
			nodeID:      setMember(leftNode.MutuallyExclusive, rightNode.FocusID),
			otherNodeID: setMember(rightNode.MutuallyExclusive, leftNode.FocusID),
		}
		for _, id := range []int64{nodeID, otherNodeID} {
			list := updates[id]
			if _, err := tx.Update("focus_nodes", id, func(r project.Row) {
				r["mutually_exclusive"] = plain(list)
				if colStr(r, "source", "") == "imported" {
					r["source"] = "created"
				}
			}); err != nil {
				return err
			}
		}
		if err := m.markTreeEdited(); err != nil {
			return err
		}
		diagnostics := m.after()
		for _, id := range []int64{nodeID, otherNodeID} {
			node := m.node(id)
			node.Diagnostics = focusRowDiagnostics(diagnostics, node)
			node.TreeDiagnostics = currentTreeDiagnostics(diagnostics)
			out = append(out, node)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	rv.commit(p)
	return out, nil
}
