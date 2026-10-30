package exportcheck

import (
	"fmt"
	"sort"
	"strings"
)

func pyRepr(s string) string {
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		return `"` + s + `"`
	}
	return "'" + s + "'"
}

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

type focusPolicyConflict struct{ msg string }

func (e focusPolicyConflict) Error() string { return e.msg }

type focusPolicyRow struct {
	ID     int64
	Policy FocusFilePolicy
}

type materializedFocusPolicy struct {
	FocusFilePolicy
}

func deriveFocusFilePolicies(trees []FocusPolicyTree, namespace string) ([]materializedFocusPolicy, error) {
	registry, err := LoadFocusTreePathRegistry()
	if err != nil {
		return nil, err
	}
	index := BuildPrimaryRegistry(registry.PolicySnapshot(), "")
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
	out := make([]materializedFocusPolicy, 0, len(order))
	for _, key := range order {
		policy, err := materializeFocusPolicyGroup(grouped[key], registry, index)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	return out, nil
}

func materializeFocusPolicyGroup(
	decisions []FocusFileDecision,
	registry *FocusTreePathRegistry,
	index RegistryIndex,
) (materializedFocusPolicy, error) {
	if len(decisions) == 0 {
		return materializedFocusPolicy{}, focusPolicyConflict{msg: "empty focus file policy group"}
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
		return materializedFocusPolicy{}, focusPolicyConflict{
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
		return materializedFocusPolicy{}, focusPolicyConflict{
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

	return materializedFocusPolicy{FocusFilePolicy: FocusFilePolicy{
		SourceFile:     targetPath,
		OriginKind:     originKind,
		TreeIDs:        treeIDs,
		MatchedTreeIDs: matched,
		EmitMode:       emitMode,
		TargetPath:     targetPath,
		TargetLayer:    targetLayer,
		DecisionSource: decisionSource,
	}}, nil
}
