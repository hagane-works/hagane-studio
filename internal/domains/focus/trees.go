package focus

import (
	"errors"
	"fmt"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

type DetailError struct {
	Status int
	Detail any
}

func (e DetailError) Error() string { return fmt.Sprintf("http %d", e.Status) }

var (
	errTreeMissing      = apiutil.NotFoundError("Focus tree not found")
	errTreeNotInProject = apiutil.NotFoundError("Focus tree not found in this project")
	errNodeMissing      = apiutil.NotFoundError("Focus node not found")
	errNodeNotPersisted = apiutil.NotFoundError("Focus node not persisted")
	errSourceMissing    = apiutil.NotFoundError("Source focus node not found")
)

func findingsToAny(findings []FocusPolicyFinding) []any {
	out := make([]any, 0, len(findings))
	for _, f := range findings {
		row := map[string]any{
			"code": f.Code, "severity": f.Severity, "source": f.Source,
			"evidence": f.Evidence, "repair_route": f.RepairRoute,
		}
		if f.Message != "" {
			row["message"] = f.Message
		}
		out = append(out, row)
	}
	return out
}

func projectCountryTags(record project.Row) []string {
	return append(workbench.StringList(record["country_tags"]), colStr(record, "country_tag", ""))
}

func listTrees(p *project.Project, countryTag, treeKind string) ([]FocusTree, error) {
	d, err := readData(p)
	if err != nil {
		return nil, err
	}
	shared := d.sharedPoolNodes()
	byTree := d.nodesByTree()
	trees := []FocusTree{}
	for _, row := range d.trees {
		if countryTag != "" && colStr(row, "country_tag", "\x00") != countryTag {
			continue
		}
		if treeKind != "" && colStr(row, "tree_kind", "") != treeKind {
			continue
		}
		t := treeFromRow(row)
		for _, n := range byTree[t.ID] {
			t.Nodes = append(t.Nodes, nodeFromRow(n, t.EditRevision))
		}
		trees = append(trees, t)
	}
	pointers := make([]*FocusTree, len(trees))
	for i := range trees {
		pointers[i] = &trees[i]
	}
	attachFilePolicies(d.policies, pointers)
	for i := range trees {
		attachFocusDiagnostics(&trees[i], shared)
	}
	return trees, nil
}

func getTree(p *project.Project, treeID int64) (FocusTree, error) {
	d, err := readData(p)
	if err != nil {
		return FocusTree{}, err
	}
	row := d.tree(treeID)
	if row == nil {
		return FocusTree{}, errTreeMissing
	}
	return treeResponse(d, row), nil
}

func treeByScriptID(p *project.Project, scriptID string) (*FocusTree, error) {
	d, err := readData(p)
	if err != nil {
		return nil, err
	}
	scriptID = strings.TrimSpace(scriptID)
	for _, row := range d.trees {
		if colStr(row, "tree_id", "") == scriptID {
			t := treeResponse(d, row)
			return &t, nil
		}
	}
	return nil, nil
}

func getNode(p *project.Project, treeID, nodeID int64) (FocusNode, error) {
	tree, err := getTree(p, treeID)
	if errors.Is(err, errTreeMissing) {
		return FocusNode{}, errNodeMissing
	}
	if err != nil {
		return FocusNode{}, err
	}
	for _, node := range tree.Nodes {
		if node.ID == nodeID {
			return node, nil
		}
	}
	return FocusNode{}, errNodeMissing
}

type FocusTreeCreateInput struct {
	OnConflict               string           `json:"on_conflict,omitempty"`
	TreeID                   string           `json:"tree_id"`
	CountryTag               string           `json:"country_tag"`
	IsDefault                bool             `json:"is_default"`
	TreeKind                 string           `json:"tree_kind"`
	SharedFocusRefs          []map[string]any `json:"shared_focus_refs"`
	SourceFile               string           `json:"source_file"`
	ContinuousFocusPositionX *int             `json:"continuous_focus_position_x"`
	ContinuousFocusPositionY *int             `json:"continuous_focus_position_y"`
	InitialShowPosition      map[string]any   `json:"-"`
	RawInitialShowPosition   any              `json:"initial_show_position"`
	InitialShowFocus         string           `json:"initial_show_focus"`
	ScopeKind                *string          `json:"scope_kind"`
	SharedCountryTags        []string         `json:"shared_country_tags"`
	Source                   *string          `json:"source"`
	RawExtras                map[string]any   `json:"raw_extras"`
}

func createTree(p *project.Project, rv *revisions, in FocusTreeCreateInput) (FocusTree, error) {
	if in.OnConflict != "" && in.OnConflict != "error" && in.OnConflict != "suffix" {
		return FocusTree{}, DetailError{Status: 422, Detail: "on_conflict must be error or suffix"}
	}
	var created FocusTree
	err := p.Write([]string{"focus_trees", "focus_nodes", "focus_file_policies"}, func(tx *project.Tx) error {
		record := tx.Record()
		source := strDefault(in.Source, "created")
		if strings.TrimSpace(in.SourceFile) != "" || source != "created" {
			return DetailError{Status: 422, Detail: map[string]any{
				"error": "focus_origin_metadata_read_only",
				"message": "New trees cannot claim imported source custody; import " +
					"metadata is assigned only by the import pipeline.",
			}}
		}
		countryTag := in.CountryTag
		if countryTag == "" {
			countryTag = colStr(record, "country_tag", "")
		}
		scopeKind := "own"
		if in.ScopeKind != nil {
			scopeKind = *in.ScopeKind
		}
		treeKind := in.TreeKind
		if treeKind == "" {
			treeKind = "primary"
		}
		materialized := MaterializeFocusBundle(FocusScopeCandidate{
			TreeKind: treeKind, TreeID: in.TreeID, ScopeKind: scopeKind,
			CountryTag: countryTag, SharedCountryTags: in.SharedCountryTags,
		}, projectCountryTags(record), nil, "focus_tree:create")
		if materialized.Blocked() {
			return DetailError{Status: 422, Detail: map[string]any{
				"error": "focus_scope_blocked", "findings": findingsToAny(materialized.Findings),
			}}
		}
		if in.IsDefault {
			return DetailError{Status: 422, Detail: map[string]any{
				"error": "focus_created_default_conflicts_with_effective_base",
				"message": "A new namespaced focus file cannot become the default " +
					"while the vanilla/base default file remains effective.",
			}}
		}

		existing := map[string]bool{}
		for _, row := range tx.Rows("focus_trees") {
			existing[colStr(row, "tree_id", "")] = true
		}
		var collisionWarning string
		treeID := in.TreeID
		if treeID != "" {
			if existing[treeID] && in.OnConflict == "error" {
				return DetailError{Status: 409, Detail: fmt.Sprintf("tree_id %s already exists in this project", pyRepr(treeID))}
			}
			original := treeID
			for suffix := 2; existing[treeID]; suffix++ {
				treeID = fmt.Sprintf("%s_%d", original, suffix)
			}
			if treeID != original {
				collisionWarning = fmt.Sprintf("tree_id %s collided · auto-suffixed to %s.", pyRepr(original), pyRepr(treeID))
			}
		} else {
			registry, err := LoadFocusTreePathRegistry()
			if err != nil {
				return err
			}
			unavailable := map[string]bool{}
			for id := range existing {
				unavailable[id] = true
			}
			for id := range registry.ByTreeID {
				unavailable[id] = true
			}
			base := strings.ToLower(countryTag)
			if base == "" {
				base = "new"
			}
			base += "_custom_focus"
			treeID = base
			for suffix := 2; unavailable[treeID]; suffix++ {
				treeID = fmt.Sprintf("%s_%d", base, suffix)
			}
		}

		initialPosition, err := materializeInitialPosition(in.InitialShowPosition, in.InitialShowFocus, "focus_tree:create")
		if err != nil {
			return err
		}
		refs := in.SharedFocusRefs
		if refs == nil {
			refs = []map[string]any{}
		}
		rawExtras := in.RawExtras
		if rawExtras == nil {
			rawExtras = map[string]any{}
		}
		continuousX, continuousY := 50, 1000
		if in.ContinuousFocusPositionX != nil {
			continuousX = *in.ContinuousFocusPositionX
		}
		if in.ContinuousFocusPositionY != nil {
			continuousY = *in.ContinuousFocusPositionY
		}
		var countryTagValue, sharedTagsValue any
		if materialized.CountryTag != "" {
			countryTagValue = materialized.CountryTag
		}
		if len(materialized.SharedCountryTags) > 0 {
			sharedTagsValue = plain(materialized.SharedCountryTags)
		}
		row := tx.Insert("focus_trees", project.Row{
			"edit_revision": int64(1), "tree_id": treeID, "country_tag": countryTagValue,
			"is_default": false, "tree_kind": treeKind, "shared_focus_refs": plain(refs),
			"source_file": "", "continuous_focus_position_x": continuousX,
			"continuous_focus_position_y": continuousY, "initial_show_position": plain(initialPosition),
			"initial_show_focus": "", "scope_kind": materialized.ScopeKind,
			"shared_country_tags": sharedTagsValue, "source": "created", "raw_extras": plain(rawExtras),
		})
		newID := project.Int64(row["id"])
		rv.created(newID)
		if err := persistPolicies(tx, []FocusPolicyTree{{TreeID: treeID, TreeKind: treeKind}}, projectNamespace(record)); err != nil {
			return err
		}
		d := txData(tx)
		created = treeResponse(d, d.tree(newID))
		if in.TreeID != "" {
			if warning := focusVanillaCollisionWarning(in.TreeID, countryTag, source); warning != nil {
				created.Warnings = append(created.Warnings, *warning)
			}
		}
		if collisionWarning != "" {
			created.Warnings = append(created.Warnings, collisionWarning)
		}
		return nil
	})
	if err != nil {
		return FocusTree{}, err
	}
	rv.commit(p)
	return created, nil
}

func focusVanillaCollisionWarning(treeID, countryTag, source string) *string {
	if treeID == "" || source == "imported" {
		return nil
	}
	registry, err := LoadFocusTreePathRegistry()
	if err != nil {
		return nil
	}
	if _, ok := registry.ByTreeID[treeID]; !ok {
		return nil
	}
	prefix := "MOD_"
	if countryTag != "" {
		prefix = strings.ToUpper(countryTag) + "_"
	}
	suggested := treeID
	if !strings.HasPrefix(treeID, prefix) {
		suggested = prefix + treeID
	}
	w := fmt.Sprintf("%s 跟 vanilla focus_tree 同名 · HOI4 mod 加载会 silent override "+
		"vanilla · 建议改名 %s(同 country TAG prefix 防 collision)", pyRepr(treeID), pyRepr(suggested))
	return &w
}

type FocusTreePatch struct {
	TreeID                   workbench.Optional[string]           `json:"tree_id"`
	CountryTag               workbench.Optional[string]           `json:"country_tag"`
	IsDefault                workbench.Optional[bool]             `json:"is_default"`
	TreeKind                 workbench.Optional[string]           `json:"tree_kind"`
	SharedFocusRefs          workbench.Optional[[]map[string]any] `json:"shared_focus_refs"`
	SourceFile               workbench.Optional[string]           `json:"source_file"`
	ContinuousFocusPositionX workbench.Optional[int]              `json:"continuous_focus_position_x"`
	ContinuousFocusPositionY workbench.Optional[int]              `json:"continuous_focus_position_y"`
	InitialShowPosition      workbench.Optional[map[string]any]   `json:"initial_show_position"`
	InitialShowFocus         workbench.Optional[string]           `json:"initial_show_focus"`
	ScopeKind                workbench.Optional[string]           `json:"scope_kind"`
	SharedCountryTags        workbench.Optional[[]string]         `json:"shared_country_tags"`
	Source                   workbench.Optional[string]           `json:"source"`
	RawExtras                workbench.Optional[map[string]any]   `json:"raw_extras"`
}

func optStr(o workbench.Optional[string]) string {
	if o.Value != nil {
		return *o.Value
	}
	return ""
}

func optValue[T any](o workbench.Optional[T]) any {
	if o.Value == nil {
		return nil
	}
	return plain(*o.Value)
}

func updateTree(p *project.Project, rv *revisions, treeRowID int64, patch FocusTreePatch) (FocusTree, error) {
	var updated FocusTree
	err := p.Write([]string{"focus_trees", "focus_nodes", "focus_file_policies"}, func(tx *project.Tx) error {
		row := tx.Find("focus_trees", treeRowID)
		if row == nil {
			return errTreeMissing
		}
		if _, err := rv.advance(tx, treeRowID); err != nil {
			return err
		}
		tree := treeFromRow(row)
		record := tx.Record()

		if patch.TreeID.Set && strings.TrimSpace(optStr(patch.TreeID)) != tree.TreeID {
			return DetailError{Status: 422, Detail: map[string]any{
				"error": "focus_tree_id_read_only",
				"message": "A focus tree id is stable custody metadata. Create a " +
					"new tree instead of retargeting an existing source file.",
			}}
		}
		if patch.SourceFile.Set && optStr(patch.SourceFile) != tree.SourceFile {
			return DetailError{Status: 422, Detail: map[string]any{"error": "focus_origin_metadata_read_only"}}
		}
		if patch.Source.Set && patch.Source.Value != nil && *patch.Source.Value == "imported" {
			return DetailError{Status: 422, Detail: map[string]any{"error": "focus_origin_metadata_read_only"}}
		}

		touchedInitial := patch.InitialShowPosition.Set || patch.InitialShowFocus.Set
		var initialPosition map[string]any
		if touchedInitial {
			var value map[string]any
			if patch.InitialShowPosition.Set {
				if patch.InitialShowPosition.Value != nil {
					value = *patch.InitialShowPosition.Value
				}
			} else {
				value = tree.InitialShowPosition
			}
			legacy := tree.InitialShowFocus
			if patch.InitialShowFocus.Set {
				legacy = optStr(patch.InitialShowFocus)
			}
			var err error
			initialPosition, err = materializeInitialPosition(value, legacy, fmt.Sprintf("focus_tree:%s", tree.TreeID))
			if err != nil {
				return err
			}
		}

		candidateKind := tree.TreeKind
		if patch.TreeKind.Set {
			candidateKind = optStr(patch.TreeKind)
		}
		candidateScope := tree.ScopeKind
		if patch.ScopeKind.Set {
			candidateScope = optStr(patch.ScopeKind)
		}
		candidateCountry := ""
		if tree.CountryTag != nil {
			candidateCountry = *tree.CountryTag
		}
		if patch.CountryTag.Set {
			candidateCountry = optStr(patch.CountryTag)
		}
		candidateShared := tree.SharedCountryTags
		if patch.SharedCountryTags.Set {
			candidateShared = nil
			if patch.SharedCountryTags.Value != nil {
				candidateShared = *patch.SharedCountryTags.Value
			}
		}
		materialized := MaterializeFocusBundle(FocusScopeCandidate{
			TreeKind: candidateKind, TreeID: tree.TreeID, ScopeKind: candidateScope,
			CountryTag: candidateCountry, SharedCountryTags: candidateShared,
		}, projectCountryTags(record), nil, "focus_tree:update")
		if materialized.Blocked() {
			return DetailError{Status: 422, Detail: map[string]any{
				"error": "focus_scope_blocked", "findings": findingsToAny(materialized.Findings),
			}}
		}

		effectiveDefault := tree.IsDefault
		if patch.IsDefault.Set && patch.IsDefault.Value != nil {
			effectiveDefault = *patch.IsDefault.Value
		}
		if effectiveDefault {
			if strings.TrimSpace(tree.SourceFile) == "" {
				return DetailError{Status: 422, Detail: map[string]any{
					"error": "focus_created_default_conflicts_with_effective_base",
					"message": "A namespaced project tree cannot become default " +
						"while the effective base default remains loaded.",
				}}
			}
			for _, other := range tx.Rows("focus_trees") {
				if project.Int64(other["id"]) != treeRowID && colBool(other, "is_default", false) {
					return DetailError{Status: 422, Detail: map[string]any{
						"error":   "focus_default_multiple",
						"message": "Only one project focus tree may be default.",
					}}
				}
			}
		}

		policyShapeChanged := false
		if patch.TreeKind.Set {
			newKind, oldKind := optStr(patch.TreeKind), tree.TreeKind
			if newKind == "" {
				newKind = "primary"
			}
			if oldKind == "" {
				oldKind = "primary"
			}
			policyShapeChanged = newKind != oldKind
		}

		if _, err := tx.Update("focus_trees", treeRowID, func(r project.Row) {
			if patch.IsDefault.Set {
				r["is_default"] = optValue(patch.IsDefault)
			}
			if patch.TreeKind.Set {
				r["tree_kind"] = optValue(patch.TreeKind)
			}
			if patch.SharedFocusRefs.Set {
				r["shared_focus_refs"] = optValue(patch.SharedFocusRefs)
			}
			if patch.ContinuousFocusPositionX.Set {
				r["continuous_focus_position_x"] = optValue(patch.ContinuousFocusPositionX)
			}
			if patch.ContinuousFocusPositionY.Set {
				r["continuous_focus_position_y"] = optValue(patch.ContinuousFocusPositionY)
			}
			if touchedInitial {
				r["initial_show_position"] = plain(initialPosition)
				r["initial_show_focus"] = ""
			}
			if patch.RawExtras.Set {
				r["raw_extras"] = optValue(patch.RawExtras)
			}
			var countryTagValue, sharedTagsValue any
			if materialized.CountryTag != "" {
				countryTagValue = materialized.CountryTag
			}
			if len(materialized.SharedCountryTags) > 0 {
				sharedTagsValue = plain(materialized.SharedCountryTags)
			}
			r["country_tag"] = countryTagValue
			r["scope_kind"] = materialized.ScopeKind
			r["shared_country_tags"] = sharedTagsValue
			if tree.Source == "imported" {
				r["source"] = "created"
			}
		}); err != nil {
			return err
		}

		if policyShapeChanged {
			cohort := policyCohortTrees(txData(tx).trees, treeRowID, tree.SourceFile)
			if err := persistPolicies(tx, cohort, projectNamespace(record)); err != nil {
				return err
			}
		}
		d := txData(tx)
		updated = treeResponse(d, d.tree(treeRowID))
		return nil
	})
	if err != nil {
		return FocusTree{}, err
	}
	rv.commit(p)
	return updated, nil
}

func deleteTree(p *project.Project, rv *revisions, treeRowID int64) error {
	return p.Write([]string{"focus_trees", "focus_nodes", "focus_file_policies"}, func(tx *project.Tx) error {
		row := tx.Find("focus_trees", treeRowID)
		if row == nil {
			return errTreeMissing
		}
		if _, err := rv.advance(tx, treeRowID); err != nil {
			return err
		}
		tree := treeFromRow(row)
		if err := detachFocusTreePolicy(tx, tree.TreeID); err != nil {
			return err
		}
		tx.Delete("focus_trees", func(r project.Row) bool { return project.Int64(r["id"]) == treeRowID })

		tx.Delete("focus_nodes", func(r project.Row) bool { return project.Int64(r["focus_tree_id"]) == treeRowID })
		delete(rv.advanced, treeRowID)
		cohort := policyCohortTrees(txData(tx).trees, treeRowID, tree.SourceFile)
		if len(cohort) > 0 {
			return persistPolicies(tx, cohort, projectNamespace(tx.Record()))
		}
		return nil
	})
}
