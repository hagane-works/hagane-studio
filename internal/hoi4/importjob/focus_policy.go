package importjob

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"haganeworks/internal/project"
)

type importedFocusPolicyTree struct {
	treeID     string
	treeKind   string
	sourceFile string
}

type importedFocusFilePolicy struct {
	sourceFile     string
	originKind     string
	treeIDs        []string
	matchedTreeIDs []string
	emitMode       string
	targetPath     string
	targetLayer    string
	decisionSource string
}

type importedFocusDecision struct {
	treeID     string
	treeKind   string
	sourceFile string
	targetPath string
	emitMode   string
}

func normaliseImportedFocusPath(value string) (string, error) {
	text := strings.ReplaceAll(strings.TrimSpace(value), `\`, "/")
	if text == "" || strings.Contains(text, ":") || strings.HasPrefix(text, "/") {
		return "", fmt.Errorf("invalid focus source path %q", value)
	}
	for _, char := range text {
		if char < 32 {
			return "", fmt.Errorf("invalid focus source path %q", value)
		}
	}
	parts := strings.Split(text, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid focus source path %q", value)
		}
	}
	if len(parts) == 1 {
		text = "common/national_focus/" + text
		parts = strings.Split(text, "/")
	}
	if len(parts) != 3 || parts[0] != "common" || parts[1] != "national_focus" || !strings.EqualFold(path.Ext(parts[2]), ".txt") {
		return "", fmt.Errorf("invalid focus source path %q", value)
	}
	return strings.Join(parts, "/"), nil
}

func deriveImportedFocusFilePolicies(trees []importedFocusPolicyTree, registry *focusRegistry) ([]importedFocusFilePolicy, error) {
	vanillaFocusPathByTreeID := registry.pathByTreeID
	registeredPaths := make(map[string]string, len(vanillaFocusPathByTreeID))
	for _, registryPath := range vanillaFocusPathByTreeID {
		registeredPaths[strings.ToLower(registryPath)] = registryPath
	}

	grouped := make(map[string][]importedFocusDecision)
	for _, tree := range trees {
		sourceFile, err := normaliseImportedFocusPath(tree.sourceFile)
		if err != nil {
			return nil, fmt.Errorf("focus tree %q: %w", tree.treeID, err)
		}
		decision := importedFocusDecision{
			treeID:     strings.TrimSpace(tree.treeID),
			treeKind:   strings.TrimSpace(tree.treeKind),
			sourceFile: sourceFile,
		}
		if decision.treeKind == "" {
			decision.treeKind = "primary"
		}
		switch decision.treeKind {
		case "shared_pool":
			decision.targetPath = sourceFile
			decision.emitMode = "shared_pool_same_path"
		case "primary":
			if canonical, ok := vanillaFocusPathByTreeID[decision.treeID]; ok {
				decision.targetPath = canonical
			} else {
				decision.targetPath = sourceFile
			}
			decision.emitMode = "canonical_same_path"
		default:
			return nil, fmt.Errorf("focus tree %q has unknown tree kind %q", decision.treeID, decision.treeKind)
		}
		grouped[strings.ToLower(sourceFile)] = append(grouped[strings.ToLower(sourceFile)], decision)
	}

	policies := make([]importedFocusFilePolicy, 0, len(grouped))
	for _, decisions := range grouped {
		targetVariants := map[string]map[string]struct{}{}
		for _, decision := range decisions {
			key := strings.ToLower(decision.targetPath)
			if targetVariants[key] == nil {
				targetVariants[key] = map[string]struct{}{}
			}
			targetVariants[key][decision.targetPath] = struct{}{}
		}
		if len(targetVariants) != 1 {
			targets := make([]string, 0, len(decisions))
			for _, variants := range targetVariants {
				for target := range variants {
					targets = append(targets, target)
				}
			}
			sort.Strings(targets)
			return nil, fmt.Errorf("one source file resolved to multiple target paths: %s", strings.Join(targets, ", "))
		}

		var targetKey string
		var variants map[string]struct{}
		for targetKey, variants = range targetVariants {
		}
		targetPath, registered := registeredPaths[targetKey]
		if !registered {
			choices := make([]string, 0, len(variants))
			for target := range variants {
				choices = append(choices, target)
			}
			sort.Strings(choices)
			targetPath = choices[0]
		}

		policy := importedFocusFilePolicy{sourceFile: targetPath, targetPath: targetPath}
		sharedOnly := true
		for _, decision := range decisions {
			policy.treeIDs = append(policy.treeIDs, decision.treeID)
			if _, ok := vanillaFocusPathByTreeID[decision.treeID]; ok {
				policy.matchedTreeIDs = append(policy.matchedTreeIDs, decision.treeID)
			}
			sharedOnly = sharedOnly && strings.HasPrefix(decision.emitMode, "shared_pool")
		}
		sort.Strings(policy.treeIDs)
		sort.Strings(policy.matchedTreeIDs)
		if len(policy.matchedTreeIDs) > 0 || registered {
			policy.originKind = "imported_vanilla"
			policy.emitMode = "mandatory_shadow_same_path"
			policy.targetLayer = "vanilla"
			policy.decisionSource = "derived_exact_match"
		} else {
			policy.originKind = "legacy_unknown"
			policy.emitMode = "mandatory_shadow_same_path"
			if sharedOnly {
				policy.emitMode = "shared_pool_same_path"
			}
			policy.targetLayer = "project"
			policy.decisionSource = "legacy_repair"
		}
		policies = append(policies, policy)
	}
	sort.Slice(policies, func(i, j int) bool {
		left, right := strings.ToLower(policies[i].sourceFile), strings.ToLower(policies[j].sourceFile)
		if left == right {
			return policies[i].sourceFile < policies[j].sourceFile
		}
		return left < right
	})
	return policies, nil
}

func (s *session) persistImportedFocusFilePolicies(tx *project.Tx) error {

	touchedSources := map[string]struct{}{}
	touchedTreeIDs := map[string]struct{}{}
	for _, tree := range s.trees.trees {
		source, err := normaliseImportedFocusPath(tree.data.str("source_file"))
		if err != nil {
			return err
		}
		touchedSources[strings.ToLower(source)] = struct{}{}
		touchedTreeIDs[tree.treeID] = struct{}{}
	}
	if len(touchedSources) == 0 {
		return nil
	}

	var trees []importedFocusPolicyTree
	for _, row := range tx.Rows("focus_trees") {
		tree := importedFocusPolicyTree{treeKind: "primary"}
		tree.treeID, _ = row["tree_id"].(string)
		if kind, ok := row["tree_kind"].(string); ok {
			tree.treeKind = kind
		}
		tree.sourceFile, _ = row["source_file"].(string)
		source, err := normaliseImportedFocusPath(tree.sourceFile)
		if err != nil {
			continue
		}
		if _, touched := touchedSources[strings.ToLower(source)]; touched {
			trees = append(trees, tree)
		}
	}
	sort.SliceStable(trees, func(i, j int) bool {
		left, right := strings.ToLower(trees[i].sourceFile), strings.ToLower(trees[j].sourceFile)
		if left != right {
			return left < right
		}
		return trees[i].treeID < trees[j].treeID
	})

	facts, err := s.gameFacts()
	if err != nil {
		return err
	}
	if facts.focusErr != nil {
		return facts.focusErr
	}
	policies, err := deriveImportedFocusFilePolicies(trees, facts.focus)
	if err != nil {
		return err
	}
	currentSources := map[string]struct{}{}
	for _, policy := range policies {
		currentSources[strings.ToLower(policy.sourceFile)] = struct{}{}
	}

	type existingPolicy struct {
		id         int64
		sourceFile string
		treeIDs    []string
		deleted    bool
	}
	var existing []existingPolicy
	for _, row := range tx.Rows("focus_file_policies") {
		policy := existingPolicy{id: project.Int64(row["id"])}
		policy.sourceFile, _ = row["source_file"].(string)
		raw, err := json.Marshal(row["tree_ids"])
		if err == nil {
			err = json.Unmarshal(raw, &policy.treeIDs)
		}
		if err != nil {
			return fmt.Errorf("decode focus file policy %d tree ids: %w", policy.id, err)
		}
		existing = append(existing, policy)
	}
	sort.SliceStable(existing, func(i, j int) bool { return existing[i].id < existing[j].id })

	stale := map[int64]bool{}
	for i := range existing {
		if _, current := currentSources[strings.ToLower(existing[i].sourceFile)]; current {
			continue
		}
		for _, treeID := range existing[i].treeIDs {
			if _, touched := touchedTreeIDs[treeID]; touched {
				stale[existing[i].id] = true
				existing[i].deleted = true
				break
			}
		}
	}
	if len(stale) > 0 {
		tx.Delete("focus_file_policies", func(row project.Row) bool { return stale[project.Int64(row["id"])] })
	}

	rowsBySource := map[string]int64{}
	for _, row := range existing {
		if !row.deleted {
			key := strings.ToLower(row.sourceFile)
			if _, present := rowsBySource[key]; !present {
				rowsBySource[key] = row.id
			}
		}
	}
	staged := stageImportedFocusPolicies(policies, rowsBySource)
	written := map[string]bool{}

	for _, policy := range staged {
		if policy.existingID == 0 {
			continue
		}
		if _, err := tx.Update("focus_file_policies", policy.existingID, func(row project.Row) {
			for column, value := range policy.row {
				row[column] = value
			}
		}); err != nil {
			return fmt.Errorf("update imported focus file policies: %w", err)
		}
		written[policy.row["source_file"].(string)] = true
	}

	for _, policy := range staged {
		if policy.existingID == 0 {
			tx.Insert("focus_file_policies", policy.row)
			written[policy.row["source_file"].(string)] = true
		}
	}

	count := map[string]int{}
	for _, row := range tx.Rows("focus_file_policies") {
		if source, ok := row["source_file"].(string); ok {
			count[source]++
		}
	}
	for source := range written {
		if count[source] > 1 {
			return fmt.Errorf("write imported focus file policies: duplicate key value violates unique constraint \"uq_focus_file_policy_project_source\" (source_file %q)", source)
		}
	}
	return nil
}

type importedFocusPolicyRow struct {
	existingID int64
	row        project.Row
}

func stageImportedFocusPolicies(policies []importedFocusFilePolicy, rowsBySource map[string]int64) []importedFocusPolicyRow {
	out := make([]importedFocusPolicyRow, 0, len(policies))
	for _, policy := range policies {
		out = append(out, importedFocusPolicyRow{
			existingID: rowsBySource[strings.ToLower(policy.sourceFile)],
			row: project.Row{
				"source_file": policy.sourceFile, "origin_kind": policy.originKind,
				"tree_ids": importedFocusIDList(policy.treeIDs), "matched_tree_ids": importedFocusIDList(policy.matchedTreeIDs),
				"registry_version": "", "registry_sha256": "",
				"emit_mode": policy.emitMode, "target_path": policy.targetPath,
				"target_layer": policy.targetLayer, "decision_source": policy.decisionSource,
			},
		})
	}
	return out
}

func importedFocusIDList(ids []string) any {
	if ids == nil {
		return nil
	}
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}
