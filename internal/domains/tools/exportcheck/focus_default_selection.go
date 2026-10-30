package exportcheck

import (
	"fmt"
	"sort"
	"strings"
)

const FocusDirectory = "common/national_focus"

const VanillaGenericTreeID = "vanilla:generic_focus"

func normaliseModPath(value string) (string, bool) {
	text := strings.TrimSpace(value)
	text = strings.ReplaceAll(text, "\\", "/")
	text = strings.Trim(text, "/")
	if text == "" || strings.Contains(text, ":") {
		return "", false
	}
	for _, r := range text {
		if r < 32 {
			return "", false
		}
	}

	parts := []string{}
	for _, seg := range strings.Split(text, "/") {

		if seg == "" || seg == "." {
			continue
		}
		if seg == ".." {
			return "", false
		}
		parts = append(parts, seg)
	}
	if len(parts) == 0 {

		return ".", true
	}
	return strings.Join(parts, "/"), true
}

func normaliseReplacePaths(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		path, ok := normaliseModPath(value)
		if !ok {
			continue
		}
		key := strings.ToLower(path)
		if seen[key] {
			continue
		}
		out = append(out, key)
		seen[key] = true
	}
	return out
}

type EffectiveFocusTree struct {
	TreeID     string `json:"tree_id"`
	Path       string `json:"path"`
	Provider   string `json:"provider"`
	ArtifactID string `json:"artifact_id"`
	IsDefault  bool   `json:"is_default"`
	TreeKind   string `json:"tree_kind"`
}

type FocusEmission struct {
	Selected     *bool  `json:"selected"`
	Effective    *bool  `json:"effective"`
	Shadowed     bool   `json:"shadowed"`
	Path         string `json:"path"`
	Provider     string `json:"provider"`
	ArtifactID   string `json:"artifact_id"`
	ArtifactOnly bool   `json:"artifact_only"`
	TreeKind     string `json:"tree_kind"`
	TreeID       string `json:"tree_id"`
	IsDefault    bool   `json:"is_default"`
	ContentHash  string `json:"content_hash"`
	SourceBacked bool   `json:"source_backed"`
}

func boolOrTrue(v *bool) bool { return v == nil || *v }

type SelectedArtifact struct {
	PathKey    string `json:"path_key"`
	ArtifactID string `json:"artifact_id"`
}

type FocusDefaultSelection struct {
	EffectiveTrees         []EffectiveFocusTree
	DefaultTrees           []EffectiveFocusTree
	ReplacePaths           []string
	SelectedArtifacts      []SelectedArtifact
	VanillaGenericInjected bool
	FocusDirectoryReplaced bool
	Findings               []FocusPolicyFinding
}

type FocusDefaultSelectionInput struct {
	Emissions              []FocusEmission
	VanillaGenericPath     string
	ReplacePaths           []string
	SelectedArtifactByPath map[string]string
	Source                 string
}

func SelectEffectiveFocusDefaults(in FocusDefaultSelectionInput) FocusDefaultSelection {
	source := in.Source
	if source == "" {
		source = "focus_defaults:effective_artifact"
	}
	replacePaths := normaliseReplacePaths(in.ReplacePaths)
	focusDirectoryReplaced := false
	for _, p := range replacePaths {
		if p == strings.ToLower(FocusDirectory) {
			focusDirectoryReplaced = true
			break
		}
	}

	explicitSelection := map[string]string{}
	for rawPath, artifactID := range in.SelectedArtifactByPath {
		path, ok := normaliseModPath(rawPath)
		selectedID := strings.TrimSpace(artifactID)
		if ok && selectedID != "" {
			explicitSelection[strings.ToLower(path)] = selectedID
		}
	}

	type record struct {
		pathKey    string
		artifactID string
		emission   FocusEmission
	}
	records := []record{}
	implicitSelection := map[string]string{}
	displayPathByKey := map[string]string{}
	for _, emission := range in.Emissions {
		if !boolOrTrue(emission.Selected) || !boolOrTrue(emission.Effective) || emission.Shadowed {
			continue
		}
		path, ok := normaliseModPath(emission.Path)
		if !ok {
			continue
		}
		pathKey := strings.ToLower(path)
		artifactID := emission.ArtifactID
		if artifactID == "" {
			artifactID = fmt.Sprintf("%s:%s", emission.Provider, pathKey)
		}
		if _, seen := displayPathByKey[pathKey]; !seen {
			displayPathByKey[pathKey] = path
		}
		records = append(records, record{pathKey, artifactID, emission})
		implicitSelection[pathKey] = artifactID
	}

	selectedArtifacts := map[string]string{}
	for k, v := range implicitSelection {
		selectedArtifacts[k] = v
	}

	for k, v := range explicitSelection {
		if _, ok := implicitSelection[k]; ok {
			selectedArtifacts[k] = v
		}
	}

	effective := []EffectiveFocusTree{}
	for _, rec := range records {
		if selectedArtifacts[rec.pathKey] != rec.artifactID {
			continue
		}
		if rec.emission.ArtifactOnly {
			continue
		}
		treeKind := rec.emission.TreeKind
		if treeKind == "" {
			treeKind = "primary"
		}
		if treeKind == "shared_pool" {
			continue
		}
		treeID := strings.TrimSpace(rec.emission.TreeID)
		if treeID == "" {
			continue
		}
		effective = append(effective, EffectiveFocusTree{
			TreeID:     treeID,
			TreeKind:   treeKind,
			Path:       displayPathByKey[rec.pathKey],
			Provider:   rec.emission.Provider,
			ArtifactID: rec.artifactID,
			IsDefault:  rec.emission.IsDefault,
		})
	}

	genericPath, hasGeneric := normaliseModPath(in.VanillaGenericPath)
	genericKey := ""
	if hasGeneric {
		genericKey = strings.ToLower(genericPath)
	}
	_, genericShadowed := selectedArtifacts[genericKey]
	injectGeneric := hasGeneric && !focusDirectoryReplaced && !genericShadowed
	if injectGeneric {
		effective = append(effective, EffectiveFocusTree{
			TreeID:     VanillaGenericTreeID,
			TreeKind:   "primary",
			Path:       genericPath,
			Provider:   "vanilla",
			ArtifactID: "vanilla:" + genericKey,
			IsDefault:  true,
		})
	}

	defaults := []EffectiveFocusTree{}
	for _, tree := range effective {
		if tree.IsDefault {
			defaults = append(defaults, tree)
		}
	}

	effectiveIDs := []string{}
	for _, tree := range effective {
		effectiveIDs = append(effectiveIDs, tree.TreeID)
	}
	findings := []FocusPolicyFinding{}
	newFinding := func(code string, defaults []EffectiveFocusTree) FocusPolicyFinding {
		ids, paths, providers := []string{}, []string{}, []string{}
		for _, tree := range defaults {
			ids = append(ids, tree.TreeID)
			paths = append(paths, tree.Path)
			providers = append(providers, tree.Provider)
		}
		return FocusPolicyFinding{
			Code:     code,
			Severity: "blocker",
			Source:   source,
			Evidence: map[string]any{
				"effective_tree_ids":       effectiveIDs,
				"default_tree_ids":         ids,
				"default_paths":            paths,
				"default_providers":        providers,
				"replace_paths":            replacePaths,
				"vanilla_generic_injected": injectGeneric,
			},
			RepairRoute: "focus_default_selection",
		}
	}
	switch {
	case len(defaults) == 0:
		findings = append(findings, newFinding("focus_effective_default_missing", nil))
	case len(defaults) > 1:
		findings = append(findings, newFinding("focus_effective_default_multiple", defaults))
	}

	selected := []SelectedArtifact{}
	for k, v := range selectedArtifacts {
		selected = append(selected, SelectedArtifact{PathKey: k, ArtifactID: v})
	}
	sort.Slice(selected, func(i, j int) bool {
		if selected[i].PathKey != selected[j].PathKey {
			return selected[i].PathKey < selected[j].PathKey
		}
		return selected[i].ArtifactID < selected[j].ArtifactID
	})

	return FocusDefaultSelection{
		EffectiveTrees:         effective,
		DefaultTrees:           defaults,
		ReplacePaths:           replacePaths,
		SelectedArtifacts:      selected,
		VanillaGenericInjected: injectGeneric,
		FocusDirectoryReplaced: focusDirectoryReplaced,
		Findings:               findings,
	}
}
