package focus

import (
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/project"
)

type focusPolicyConflict struct{ msg string }

func (e focusPolicyConflict) Error() string { return e.msg }

func policyConflictDetail(err error) error {
	if conflict, ok := err.(focusPolicyConflict); ok {
		return DetailError{Status: 422, Detail: map[string]any{
			"error": "focus_file_policy_conflict", "message": conflict.msg,
		}}
	}
	return err
}

type focusPolicyRow struct {
	ID     int64
	Policy FocusFilePolicy
}

func policyFromRow(row project.Row) focusPolicyRow {
	r := focusPolicyRow{ID: project.Int64(row["id"]), Policy: FocusFilePolicy{
		SourceFile:     colStr(row, "source_file", ""),
		OriginKind:     colStr(row, "origin_kind", ""),
		EmitMode:       colStr(row, "emit_mode", ""),
		TargetPath:     colStr(row, "target_path", ""),
		TargetLayer:    colStr(row, "target_layer", ""),
		DecisionSource: colStr(row, "decision_source", ""),
	}}
	decodeInto(row["tree_ids"], &r.Policy.TreeIDs)
	decodeInto(row["matched_tree_ids"], &r.Policy.MatchedTreeIDs)
	if r.Policy.TreeIDs == nil {
		r.Policy.TreeIDs = []string{}
	}
	if r.Policy.MatchedTreeIDs == nil {
		r.Policy.MatchedTreeIDs = []string{}
	}
	return r
}

func loadFocusPolicyRows(rows []project.Row) []focusPolicyRow {
	out := make([]focusPolicyRow, 0, len(rows))
	for _, row := range sortByID(append([]project.Row(nil), rows...)) {
		out = append(out, policyFromRow(row))
	}
	return out
}

func deriveFocusFilePolicies(trees []FocusPolicyTree, namespace string) ([]FocusFilePolicy, error) {
	registry, err := LoadFocusTreePathRegistry()
	if err != nil {
		return nil, err
	}
	index := registryIndex(registry)
	grouped := map[string][]FocusFileDecision{}
	var order []string
	for _, tree := range trees {
		decision := DecideFocusFilePolicy(tree, index, namespace, "focus_file_policy:persist")
		key := decision.SourceFile
		if key == "" {
			key = decision.TargetPath
		}
		if key == "" {
			return nil, focusPolicyConflict{msg: fmt.Sprintf(
				"focus tree %s has no safe source-file key", pyRepr(decision.TreeID))}
		}
		folded := strings.ToLower(key)
		if _, seen := grouped[folded]; !seen {
			order = append(order, folded)
		}
		grouped[folded] = append(grouped[folded], decision)
	}
	out := make([]FocusFilePolicy, 0, len(order))
	for _, key := range order {
		policy, err := materializeFocusPolicyGroup(grouped[key], registry, index)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	return out, nil
}

func materializeFocusPolicyGroup(decisions []FocusFileDecision, registry *FocusTreePathRegistry, index RegistryIndex) (FocusFilePolicy, error) {
	if len(decisions) == 0 {
		return FocusFilePolicy{}, focusPolicyConflict{msg: "empty focus file policy group"}
	}
	blockerSet := map[string]bool{}
	for _, decision := range decisions {
		for _, finding := range decision.Findings {
			if finding.Severity == "blocker" {
				blockerSet[finding.Code] = true
			}
		}
	}
	if len(blockerSet) > 0 {
		return FocusFilePolicy{}, focusPolicyConflict{
			msg: "unsafe focus file policy: " + strings.Join(sortedKeys(blockerSet), ", ")}
	}

	targetsByKey := map[string]map[string]bool{}
	for _, decision := range decisions {
		if decision.TargetPath != "" {
			key := strings.ToLower(decision.TargetPath)
			if targetsByKey[key] == nil {
				targetsByKey[key] = map[string]bool{}
			}
			targetsByKey[key][decision.TargetPath] = true
		}
	}
	if len(targetsByKey) != 1 {
		var all []string
		for _, variants := range targetsByKey {
			all = append(all, sortedKeys(variants)...)
		}
		sort.Strings(all)
		return FocusFilePolicy{}, focusPolicyConflict{
			msg: "one source file resolved to multiple target paths: " + strings.Join(all, ", ")}
	}
	var targetKey string
	var targetVariants map[string]bool
	for k, v := range targetsByKey {
		targetKey, targetVariants = k, v
	}
	registeredPaths := map[string]string{}
	for path := range index.ByPath {
		registeredPaths[strings.ToLower(path)] = path
	}
	targetPath, shadowsRegistered := registeredPaths[targetKey]
	if !shadowsRegistered {
		targetPath = sortedKeys(targetVariants)[0]
	}
	var matched []string
	imported := false
	sharedOnly := true
	var treeIDs []string
	for _, decision := range decisions {
		treeIDs = append(treeIDs, decision.TreeID)
		if _, ok := index.ByTreeID[decision.TreeID]; ok {
			matched = append(matched, decision.TreeID)
		}
		if decision.OriginKind == "imported" {
			imported = true
		}
		if !strings.HasPrefix(decision.EmitMode, "shared_pool") {
			sharedOnly = false
		}
	}
	sort.Strings(matched)
	sort.Strings(treeIDs)
	if matched == nil {
		matched = []string{}
	}

	var originKind, emitMode, targetLayer, decisionSource string
	switch {
	case len(matched) > 0 || (imported && shadowsRegistered):
		originKind = "imported_vanilla"
		emitMode = "mandatory_shadow_same_path"
		targetLayer = "vanilla"
		decisionSource = "derived_exact_match"
	case imported:
		originKind = "legacy_unknown"
		if sharedOnly {
			emitMode = "shared_pool_same_path"
		} else {
			emitMode = "mandatory_shadow_same_path"
		}
		targetLayer = "project"
		decisionSource = "legacy_repair"
	default:
		originKind = "created"
		if sharedOnly {
			emitMode = "shared_pool_namespaced_addition"
		} else {
			emitMode = "namespaced_addition"
		}
		targetLayer = "project"
		decisionSource = "author_created_unique"
	}
	return FocusFilePolicy{
		SourceFile: targetPath, OriginKind: originKind, TreeIDs: treeIDs, MatchedTreeIDs: matched,
		EmitMode: emitMode, TargetPath: targetPath, TargetLayer: targetLayer, DecisionSource: decisionSource,
	}, nil
}

func policyColumns(policy FocusFilePolicy) project.Row {
	return project.Row{
		"source_file": policy.SourceFile, "origin_kind": policy.OriginKind,
		"tree_ids": plain(policy.TreeIDs), "matched_tree_ids": plain(policy.MatchedTreeIDs),
		"registry_version": "", "registry_sha256": "",
		"emit_mode": policy.EmitMode, "target_path": policy.TargetPath,
		"target_layer": policy.TargetLayer, "decision_source": policy.DecisionSource,
	}
}

func persistFocusFilePolicies(tx *project.Tx, trees []FocusPolicyTree, namespace string) ([]FocusFilePolicy, error) {
	materialized, err := deriveFocusFilePolicies(trees, namespace)
	if err != nil {
		return nil, err
	}
	treeIDs := map[string]bool{}
	currentSources := map[string]bool{}
	for _, policy := range materialized {
		for _, id := range policy.TreeIDs {
			treeIDs[id] = true
		}
		currentSources[strings.ToLower(policy.SourceFile)] = true
	}
	existing := loadFocusPolicyRows(tx.Rows("focus_file_policies"))
	deleted := map[int64]bool{}
	if len(treeIDs) > 0 {
		for _, old := range existing {
			overlap := false
			for _, id := range old.Policy.TreeIDs {
				if treeIDs[id] {
					overlap = true
					break
				}
			}
			if overlap && !currentSources[strings.ToLower(old.Policy.SourceFile)] {
				deleted[old.ID] = true
			}
		}
	}
	if len(deleted) > 0 {
		tx.Delete("focus_file_policies", func(row project.Row) bool { return deleted[project.Int64(row["id"])] })
	}
	rowsBySource := map[string]int64{}
	for _, old := range existing {
		if deleted[old.ID] {
			continue
		}
		key := strings.ToLower(old.Policy.SourceFile)
		if _, seen := rowsBySource[key]; !seen {
			rowsBySource[key] = old.ID
		}
	}
	for _, policy := range materialized {
		columns := policyColumns(policy)
		if id, ok := rowsBySource[strings.ToLower(policy.SourceFile)]; ok {
			if _, err := tx.Update("focus_file_policies", id, func(row project.Row) {
				for k, v := range columns {
					row[k] = v
				}
			}); err != nil {
				return nil, err
			}
			continue
		}
		tx.Insert("focus_file_policies", columns)
	}
	return materialized, nil
}

func persistPolicies(tx *project.Tx, trees []FocusPolicyTree, namespace string) error {
	_, err := persistFocusFilePolicies(tx, trees, namespace)
	return policyConflictDetail(err)
}

func detachFocusTreePolicy(tx *project.Tx, treeID string) error {
	deleteIDs := map[int64]bool{}
	for _, row := range loadFocusPolicyRows(tx.Rows("focus_file_policies")) {
		contains := false
		for _, id := range row.Policy.TreeIDs {
			if id == treeID {
				contains = true
				break
			}
		}
		if !contains {
			continue
		}
		remaining := []string{}
		for _, id := range row.Policy.TreeIDs {
			if id != treeID {
				remaining = append(remaining, id)
			}
		}
		if len(remaining) == 0 {
			deleteIDs[row.ID] = true
			continue
		}
		matched := []string{}
		for _, id := range row.Policy.MatchedTreeIDs {
			if id != treeID {
				matched = append(matched, id)
			}
		}
		if _, err := tx.Update("focus_file_policies", row.ID, func(r project.Row) {
			r["tree_ids"], r["matched_tree_ids"] = plain(remaining), plain(matched)
		}); err != nil {
			return err
		}
	}
	if len(deleteIDs) > 0 {
		tx.Delete("focus_file_policies", func(row project.Row) bool { return deleteIDs[project.Int64(row["id"])] })
	}
	return nil
}

func policyCohortTrees(trees []project.Row, treeRowID int64, sourceFile string) []FocusPolicyTree {
	out := []FocusPolicyTree{}
	view := func(row project.Row) FocusPolicyTree {
		return FocusPolicyTree{
			TreeID: colStr(row, "tree_id", ""), TreeKind: colStr(row, "tree_kind", "primary"),
			SourceFile: colStr(row, "source_file", ""),
		}
	}
	source := strings.TrimSpace(sourceFile)
	if source == "" {
		for _, row := range trees {
			if project.Int64(row["id"]) == treeRowID {
				out = append(out, view(row))
			}
		}
		return out
	}
	key := func(value string) string {
		value = strings.TrimSpace(value)
		if normalised, ok := normaliseFocusPath(value); ok {
			value = normalised
		}
		return strings.ToLower(value)
	}
	sourceKey := key(source)
	for _, row := range trees {
		if key(colStr(row, "source_file", "")) == sourceKey {
			out = append(out, view(row))
		}
	}
	return out
}

func attachFilePolicies(policyRows []project.Row, trees []*FocusTree) {
	rows := loadFocusPolicyRows(policyRows)
	byTreeID := map[string]*FocusFilePolicy{}
	for i := range rows {
		for _, treeID := range rows[i].Policy.TreeIDs {
			if _, seen := byTreeID[treeID]; !seen {
				byTreeID[treeID] = &rows[i].Policy
			}
		}
	}
	for _, tree := range trees {
		tree.FilePolicy = byTreeID[tree.TreeID]
	}
}

func projectNamespace(record project.Row) string {
	namespace := strings.ToUpper(colStr(record, "country_tag", ""))
	if namespace == "" {
		namespace = "UNK"
	}
	return namespace
}
