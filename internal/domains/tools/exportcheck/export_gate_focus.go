package exportcheck

import (
	"fmt"
	"haganeworks/internal/hoi4/clausewitz"
	"haganeworks/internal/hoi4/importsvc"
	treeparser "haganeworks/internal/hoi4/importsvc/trees"
	"haganeworks/internal/hoi4/paradox"
	"sort"
	"strings"
)

type FocusGateProject struct {
	CountryTag        string
	ExportLanguages   []string
	Dependencies      []string
	BasemodTemplateID any
}

type FocusGateRawFile struct {
	ID                  any    `json:"id"`
	DestPath            string `json:"dest_path"`
	Content             string `json:"content"`
	IsBinary            bool   `json:"is_binary"`
	ContentSHA256       string `json:"content_sha256"`
	GenerateReplacePath bool   `json:"generate_replace_path"`
	Source              string `json:"source"`
}

type FocusGatePolicyRow struct {
	ID     int64
	Policy FocusFilePolicy
}

type FocusFindingsInput struct {
	Project       FocusGateProject
	Trees         []FocusTree
	Countries     []string
	RawFiles      []FocusGateRawFile
	ProviderTrees []FocusTree
	FilePolicies  []FocusGatePolicyRow
	ReplacePaths  []string
}

func policyFindingDict(f FocusPolicyFinding) map[string]any {
	evidence := f.Evidence
	if evidence == nil {
		evidence = map[string]any{}
	}
	return map[string]any{
		"code": f.Code, "severity": f.Severity, "source": f.Source,
		"evidence": evidence, "repair_route": f.RepairRoute,
	}
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (p FocusGateProject) namespace() string {
	tag := p.CountryTag
	if tag == "" {
		tag = "UNK"
	}
	return strings.ToUpper(tag)
}

func sharedPoolClosure(directRefs map[string]map[string]any,
	deps func(focusID string) ([]string, bool)) []string {
	included := map[string]bool{}
	pending := []string{}
	for id := range directRefs {
		included[id] = true
		pending = append(pending, id)
	}
	for len(pending) > 0 {
		focusID := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		dependencies, ok := deps(focusID)
		if !ok {
			continue
		}
		for _, dependency := range dependencies {
			if dependency == "" || included[dependency] {
				continue
			}
			if _, exists := deps(dependency); !exists {
				continue
			}
			included[dependency] = true
			pending = append(pending, dependency)
		}
	}
	out := make([]string, 0, len(included))
	for id := range included {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func refInt(reference map[string]any, key string) int {
	if reference == nil {
		return 0
	}
	return coerceInt(reference[key])
}

func refString(reference map[string]any, key string) string {
	if reference == nil {
		return ""
	}
	return anyToString(reference[key])
}

func FocusFindings(in FocusFindingsInput) ([]map[string]any, error) {
	treeRows := in.Trees
	providerRows := in.ProviderTrees
	if providerRows == nil {
		providerRows = treeRows
	}
	findings := []map[string]any{}

	selectedReplacePaths := append([]string{}, in.ReplacePaths...)
	for _, raw := range in.RawFiles {
		if !raw.GenerateReplacePath {
			continue
		}
		path := SafeRawPath(raw.DestPath)
		directory := ""
		if i := strings.LastIndex(path, "/"); i >= 0 {
			directory = path[:i]
		}
		if directory == "" {
			continue
		}
		seen := false
		for _, existing := range selectedReplacePaths {
			if existing == directory {
				seen = true
				break
			}
		}
		if !seen {
			selectedReplacePaths = append(selectedReplacePaths, directory)
		}
	}

	projectTags := projectTags(CountryHistoryBuildInput{
		ProjectCountryTag: in.Project.CountryTag, CountryTags: in.Countries})
	enabledLocales := in.Project.ExportLanguages
	if len(enabledLocales) == 0 {
		enabledLocales = []string{"en"}
	}

	poolNodes := map[string]FocusNode{}
	for _, pool := range providerRows {
		if pool.TreeKind != "shared_pool" {
			continue
		}
		for _, node := range pool.Nodes {
			poolNodes[node.FocusID] = node
		}
	}

	policyRows := make([]FocusPolicyRow, 0, len(in.FilePolicies))
	for _, row := range in.FilePolicies {
		policyRows = append(policyRows, FocusPolicyRow{ID: row.ID, TreeIDs: row.Policy.TreeIDs})
	}
	policyByTreeID := PolicyRowsByTreeID(policyRows)
	persistedByID := map[int64]FocusFilePolicy{}
	for _, row := range in.FilePolicies {
		persistedByID[row.ID] = row.Policy
	}

	selectedTreeIDs := map[string]bool{}
	for _, tree := range treeRows {
		selectedTreeIDs[tree.TreeID] = true
	}

	for _, row := range in.FilePolicies {
		selectedMembers, missingMembers := map[string]bool{}, map[string]bool{}
		for _, treeID := range row.Policy.TreeIDs {
			if selectedTreeIDs[treeID] {
				selectedMembers[treeID] = true
			} else {
				missingMembers[treeID] = true
			}
		}
		if len(selectedMembers) == 0 || len(missingMembers) == 0 {
			continue
		}
		findings = append(findings, map[string]any{
			"code": "focus_file_policy_cohort_incomplete", "severity": "blocker",
			"source": map[string]any{
				"kind": "focus_file_policy", "identifier": row.Policy.SourceFile,
			},
			"evidence": map[string]any{
				"selected_tree_ids": sortedKeys(selectedMembers),
				"missing_tree_ids":  sortedKeys(missingMembers),
			},
			"repair_route": "focus.file_policy.expand_cohort",
		})
	}
	duplicates := DuplicatePolicyTreeIDs(policyRows)
	for _, treeID := range sortedMapKeys(duplicates) {
		ids := []any{}
		for _, row := range duplicates[treeID] {
			ids = append(ids, row.ID)
		}
		findings = append(findings, map[string]any{
			"code": "focus_file_policy_duplicate_membership", "severity": "blocker",
			"source": map[string]any{
				"kind": "focus_file_policy", "identifier": treeID,
			},
			"evidence":     map[string]any{"tree_id": treeID, "policy_ids": ids},
			"repair_route": "focus.file_policy.repair_membership",
		})
	}

	emissions := []FocusEmission{}

	var checkedRegistry *FocusTreePathRegistry
	var registryIndex *RegistryIndex
	registry, regErr := LoadFocusTreePathRegistry()
	if regErr != nil {
		findings = append(findings, map[string]any{
			"code": "focus_registry_unavailable", "severity": "blocker",
			"source": map[string]any{
				"kind": "focus_registry", "identifier": "hoi4",
			},
			"evidence": map[string]any{
				"error_type": pyExceptionName(pyErrFocusRegistry), "error": regErr.Error(),
			},
			"repair_route": "focus_registry_refresh",
		})
	} else {
		checkedRegistry = registry
		index := BuildPrimaryRegistry(registry.PolicySnapshot(), "")
		registryIndex = &index
	}

	expectedPolicyByTreeID := map[string]materializedFocusPolicy{}
	if checkedRegistry != nil {
		policyTrees := make([]FocusPolicyTree, 0, len(providerRows))
		for _, tree := range providerRows {
			policyTrees = append(policyTrees, FocusPolicyTree{
				TreeID: tree.TreeID, TreeKind: tree.TreeKind,

				OriginKind: "", SourceFile: tree.SourceFile,
			})
		}
		if expected, err := deriveFocusFilePolicies(policyTrees, in.Project.namespace()); err == nil {
			for _, policy := range expected {
				for _, treeID := range policy.TreeIDs {
					expectedPolicyByTreeID[treeID] = policy
				}
			}
		}
	}

	for _, tree := range treeRows {
		scope := MaterializeFocusBundle(FocusScopeCandidate{
			TreeKind: tree.TreeKind, TreeID: tree.TreeID, ScopeKind: tree.ScopeKind,
			CountryTag: derefString(tree.CountryTag), SharedCountryTags: tree.SharedCountryTags,
		}, projectTags, nil, "focus_tree:"+tree.TreeID)
		for _, f := range scope.Findings {
			findings = append(findings, policyFindingDict(f))
		}

		initial := normalizeInitialShowPosition(
			tree.InitialShowPosition, tree.InitialShowFocus, "", tree.SourceFile)
		findings = append(findings, initial.Findings...)

		mode := "created_independent"
		if strings.TrimSpace(tree.SourceFile) != "" {
			mode = "replacement"
		}

		directRefs := map[string]map[string]any{}
		for _, reference := range tree.SharedFocusRefs {
			if id := strings.TrimSpace(anyToString(reference["focus_id"])); id != "" {
				directRefs[id] = reference
			}
		}
		included := sharedPoolClosure(directRefs, func(focusID string) ([]string, bool) {
			shared, ok := poolNodes[focusID]
			if !ok {
				return nil, false
			}
			deps := []string{strings.TrimSpace(shared.RelativePositionID)}
			for _, group := range shared.Prerequisites {
				for _, value := range group {
					deps = append(deps, strings.TrimSpace(value))
				}
			}
			return deps, true
		})

		sharedProjections := []SharedProjectionPolicy{}
		for _, focusID := range included {
			shared, ok := poolNodes[focusID]
			if !ok {
				continue
			}
			reference := directRefs[focusID]
			dx, dy := FocusTreeConditionalOffset(shared.Offset, tree.TreeID)
			sharedProjections = append(sharedProjections, SharedProjectionPolicy{
				FocusID: focusID, BaseX: shared.X + dx, BaseY: shared.Y + dy,
				RelativePositionID: shared.RelativePositionID,
				OffsetX:            refInt(reference, "offset_x"),
				OffsetY:            refInt(reference, "offset_y"),
				Source:             "shared_pool",
				Condition:          refString(reference, "condition"),
			})
		}

		baselineNodes := []LayoutNode{}
		for _, node := range tree.Nodes {
			snapshot := node.sourceSnapshot
			if snapshot == nil {
				continue
			}
			complete := true
			for _, key := range []string{"focus_id", "x", "y"} {
				if _, ok := snapshot[key]; !ok {
					complete = false
					break
				}
			}
			if !complete {
				continue
			}
			baselineNodes = append(baselineNodes, LayoutNode{
				FocusID:            anyToString(snapshot["focus_id"]),
				X:                  coerceInt(snapshot["x"]),
				Y:                  coerceInt(snapshot["y"]),
				RelativePositionID: anyToString(snapshot["relative_position_id"]),
				Source:             "import_snapshot",
			})
		}

		layoutNodes := make([]LayoutNode, 0, len(tree.Nodes))
		for _, node := range tree.Nodes {
			source := node.Source
			if source == "" {
				source = "project"
			}
			layoutNodes = append(layoutNodes, LayoutNode{
				FocusID: node.FocusID, X: node.X, Y: node.Y,
				RelativePositionID: node.RelativePositionID, Source: source,
			})
		}
		layout := ValidateCandidateLayout(layoutNodes, mode, sharedProjections,
			baselineNodes, len(baselineNodes) > 0, "focus_layout:"+tree.TreeID)
		for _, f := range layout.Findings {
			findings = append(findings, policyFindingDict(f))
		}

		if initial.Value != nil && anyToString(initial.Value["kind"]) == "focus" {
			target := anyToString(initial.Value["focus_id"])
			if _, ok := layout.Positions[target]; !ok {
				findings = append(findings, map[string]any{
					"code": "focus_initial_position_target_missing", "severity": "blocker",
					"source": map[string]any{
						"kind": "focus_tree", "identifier": tree.TreeID,
						"path": nullableString(tree.SourceFile),
					},
					"evidence": map[string]any{
						"focus_id":            target,
						"available_focus_ids": sortedPositionIDs(layout.Positions),
					},
					"repair_route": "focus.initial_position.choose_target",
				})
			}
		}

		for _, node := range tree.Nodes {
			if strings.TrimSpace(node.CompletionReward) != "" {
				focusID, sourceName := node.FocusID, tree.SourceFile
				reward := AnalyzeCompletionReward(CompletionRewardInput{
					Text: node.CompletionReward, FocusID: &focusID,
					SourceName: nullableStringPtr(sourceName),
				})
				for _, f := range reward.Findings {
					findings = append(findings, analysisFindingDict(f))
				}
			}
			source := node.Source
			if source == "" {
				source = "created"
			}
			locs := map[string]any{}
			for k, v := range node.DisplayNameLocs {
				locs[k] = v
			}
			display := AnalyzeFocusDisplayName(FocusDisplayNameInput{
				FocusID: node.FocusID, Source: source, DisplayNameLocs: locs,
				NameEN: node.NameEn, NameZH: node.NameZh,
				EnabledContentLocales: enabledLocales,

				IsNewID:               source != "imported",
				MissingLocaleSeverity: DisplayNameSeverityPolicy{Fixed: "warning"},
				SourceName:            nullableStringPtr(tree.SourceFile),
			})
			findings = append(findings, display.Findings...)
		}

		if registryIndex == nil {
			continue
		}
		decision := DecideFocusFilePolicy(FocusPolicyTree{
			TreeID: tree.TreeID, TreeKind: tree.TreeKind,

			OriginKind: "", SourceFile: tree.SourceFile,
		}, *registryIndex, in.Project.namespace(), "")
		for _, f := range decision.Findings {
			findings = append(findings, policyFindingDict(f))
		}
		emissionPath := decision.TargetPath
		persistedRow, hasPersisted := policyByTreeID[tree.TreeID]

		if !hasPersisted {
			origin := "created"
			if strings.TrimSpace(tree.SourceFile) != "" {
				origin = "imported"
			}
			findings = append(findings, map[string]any{
				"code": "focus_file_policy_missing", "severity": "blocker",
				"source": map[string]any{
					"kind": "focus_tree", "identifier": tree.TreeID,
					"path": nullableString(tree.SourceFile),
				},
				"evidence": map[string]any{
					"origin": origin, "target_path": decision.TargetPath,
				},
				"repair_route": "focus.file_policy.reimport_or_repair",
			})
		} else if checkedRegistry != nil {
			persisted := persistedByID[persistedRow.ID]
			if persisted.TargetPath != "" {
				emissionPath = persisted.TargetPath
			}

			actualContract := map[string]any{
				"source_file":      persisted.SourceFile,
				"origin_kind":      persisted.OriginKind,
				"tree_ids":         sortedCopy(persisted.TreeIDs),
				"matched_tree_ids": sortedCopy(persisted.MatchedTreeIDs),
				"emit_mode":        persisted.EmitMode,
				"target_path":      persisted.TargetPath,
				"target_layer":     persisted.TargetLayer,
				"decision_source":  persisted.DecisionSource,
			}
			expectedContract := map[string]any{}
			if expected, ok := expectedPolicyByTreeID[tree.TreeID]; ok {
				expectedContract = map[string]any{
					"source_file":      expected.SourceFile,
					"origin_kind":      expected.OriginKind,
					"tree_ids":         sortedCopy(expected.TreeIDs),
					"matched_tree_ids": sortedCopy(expected.MatchedTreeIDs),
					"emit_mode":        expected.EmitMode,
					"target_path":      expected.TargetPath,
					"target_layer":     expected.TargetLayer,
					"decision_source":  expected.DecisionSource,
				}
			}
			drifted := !contractsEqual(actualContract, expectedContract)

			code, severity, repairRoute := "focus_file_policy_applied", "info", ""
			if drifted {
				code, severity = "focus_file_policy_drift", "blocker"
				repairRoute = "focus.file_policy.reimport_or_repair"
			}
			identifier := persisted.SourceFile
			if identifier == "" {
				identifier = tree.TreeID
			}
			treeIDs := persisted.TreeIDs
			if treeIDs == nil {
				treeIDs = []string{}
			}
			findings = append(findings, map[string]any{
				"code": code, "severity": severity,
				"source": map[string]any{
					"kind": "focus_file_policy", "identifier": identifier,
					"path": nullableString(persisted.TargetPath),
				},
				"evidence": map[string]any{
					"tree_ids":          treeIDs,
					"target_path":       persisted.TargetPath,
					"target_layer":      persisted.TargetLayer,
					"emit_mode":         persisted.EmitMode,
					"decision_source":   persisted.DecisionSource,
					"expected_contract": expectedContract,
					"mandatory":         persisted.EmitMode == "mandatory_shadow_same_path",
				},
				"repair_route": repairRoute,
			})
		}

		if emissionPath != "" {
			treeKind := tree.TreeKind
			if treeKind == "" {
				treeKind = "primary"
			}
			emissions = append(emissions, FocusEmission{
				TreeID: tree.TreeID, TreeKind: treeKind, Path: emissionPath,
				Provider:     decision.DecisionSource,
				ArtifactID:   "structured:" + strings.ToLower(emissionPath),
				ContentHash:  "",
				IsDefault:    tree.IsDefault,
				SourceBacked: strings.TrimSpace(tree.SourceFile) != "",
			})
		}
	}

	type rawArtifact struct {
		raw     FocusGateRawFile
		path    string
		bundles []*importsvc.TreeBundle
	}
	rawArtifacts := []rawArtifact{}
	for _, raw := range in.RawFiles {
		path := SafeRawPath(raw.DestPath)
		lower := strings.ToLower(path)
		if !strings.HasPrefix(lower, "common/national_focus/") ||
			!strings.HasSuffix(lower, ".txt") || raw.IsBinary {
			continue
		}

		_, recoveries, perr := paradox.Parse(raw.Content)
		if failed, message := clausewitz.ParseErrorMessage(recoveries, perr); failed {
			findings = append(findings, map[string]any{
				"code": "focus_raw_file_parse_failed", "severity": "blocker",
				"source": map[string]any{
					"kind": "focus_raw_file", "identifier": path, "path": path,
				},
				"evidence": map[string]any{
					"error_type": pyExceptionName(pyErrParse), "error": message,
				},
				"repair_route": "focus_raw_file_repair",
			})
			continue
		}
		bundles, _, err := treeparser.ImportFocusFile(raw.Content, path)
		if err != nil {
			findings = append(findings, map[string]any{
				"code": "focus_raw_file_parse_failed", "severity": "blocker",
				"source": map[string]any{
					"kind": "focus_raw_file", "identifier": path, "path": path,
				},
				"evidence": map[string]any{
					"error_type": pyExceptionName(pyErrParse), "error": err.Error(),
				},
				"repair_route": "focus_raw_file_repair",
			})
			continue
		}
		rawArtifacts = append(rawArtifacts, rawArtifact{raw, path, bundles})
	}

	rawPoolNodes := map[string]map[string]any{}
	for _, artifact := range rawArtifacts {
		for _, bundle := range artifact.bundles {
			if bundle.TreeKind != "shared_pool" {
				continue
			}
			for _, node := range bundle.NodesData {
				if id := strings.TrimSpace(anyToString(node["focus_id"])); id != "" {
					rawPoolNodes[anyToString(node["focus_id"])] = node
				}
			}
		}
	}

	for _, artifact := range rawArtifacts {
		raw, path, bundles := artifact.raw, artifact.path, artifact.bundles
		rawID := anyToString(raw.ID)
		if raw.ID == nil || rawID == "" || rawID == "0" {
			rawID = path
		}
		rawArtifactID := fmt.Sprintf("raw:%s:%s", rawID, strings.ToLower(path))

		rawHasPrimary := len(bundles) == 0
		for _, bundle := range bundles {
			if bundle.TreeKind != "shared_pool" {
				rawHasPrimary = true
			}
		}
		rawTreeKind := "shared_pool"
		if rawHasPrimary {
			rawTreeKind = "primary"
		}
		emissions = append(emissions, FocusEmission{
			TreeID: "", TreeKind: rawTreeKind, Path: path, Provider: "project_raw",
			ArtifactID: rawArtifactID, ArtifactOnly: true,
			ContentHash: raw.ContentSHA256, SourceBacked: rawHasPrimary,
		})

		for _, bundle := range bundles {
			if bundle.TreeKind == "shared_pool" {
				continue
			}
			treeData := map[string]any{}
			for k, v := range bundle.TreeData {
				treeData[k] = v
			}
			treeData["tree_kind"] = bundle.TreeKind
			treeData["source_file"] = path
			treeID := anyToString(treeData["tree_id"])

			var verdict *FocusScopeVerdict
			if parsed, ok := treeData["_scope_verdict"].(importsvc.ScopeVerdict); ok {
				verdict = &FocusScopeVerdict{
					Kind: string(parsed.Kind), CountryTags: parsed.CountryTags,
				}
			}
			scope := MaterializeFocusBundle(FocusScopeCandidate{
				TreeKind: bundle.TreeKind, TreeID: treeID,
				ScopeKind:         anyToString(treeData["scope_kind"]),
				CountryTag:        anyToString(treeData["country_tag"]),
				SharedCountryTags: anyToStringSlice(treeData["shared_country_tags"]),
			}, projectTags, verdict, fmt.Sprintf("focus_raw:%s:%s", path, treeID))
			for _, f := range scope.Findings {
				findings = append(findings, policyFindingDict(f))
			}

			initialValue, _ := treeData["initial_show_position"].(map[string]any)
			if len(initialValue) == 0 {
				initialValue = nil
			}
			initial := normalizeInitialShowPosition(initialValue,
				anyToString(treeData["initial_show_focus"]), "", path)
			findings = append(findings, initial.Findings...)

			directRefs := map[string]map[string]any{}
			for _, reference := range bundle.SharedFocusRefs {
				if id := strings.TrimSpace(anyToString(reference["focus_id"])); id != "" {
					directRefs[id] = reference
				}
			}
			included := sharedPoolClosure(directRefs, func(focusID string) ([]string, bool) {
				shared, ok := rawPoolNodes[focusID]
				if !ok {
					return nil, false
				}
				deps := []string{strings.TrimSpace(anyToString(shared["relative_position_id"]))}
				for _, group := range anyToGroups(shared["prerequisites"]) {
					for _, value := range group {
						deps = append(deps, strings.TrimSpace(value))
					}
				}
				return deps, true
			})

			sharedProjections := []SharedProjectionPolicy{}
			for _, focusID := range included {
				shared, ok := rawPoolNodes[focusID]
				if !ok {
					continue
				}
				reference := directRefs[focusID]
				dx, dy := FocusTreeConditionalOffset(anyToMapSlice(shared["offset"]), treeID)
				sharedProjections = append(sharedProjections, SharedProjectionPolicy{
					FocusID:            focusID,
					BaseX:              coerceInt(shared["x"]) + dx,
					BaseY:              coerceInt(shared["y"]) + dy,
					RelativePositionID: anyToString(shared["relative_position_id"]),
					OffsetX:            refInt(reference, "offset_x"),
					OffsetY:            refInt(reference, "offset_y"),
					Source:             "raw_shared_pool",
					Condition:          refString(reference, "condition"),
				})
			}

			layoutNodes := make([]LayoutNode, 0, len(bundle.NodesData))
			for _, node := range bundle.NodesData {
				layoutNodes = append(layoutNodes, LayoutNode{
					FocusID:            anyToString(node["focus_id"]),
					X:                  coerceInt(node["x"]),
					Y:                  coerceInt(node["y"]),
					RelativePositionID: anyToString(node["relative_position_id"]),
					Source:             "raw",
				})
			}

			layout := ValidateCandidateLayout(layoutNodes, "replacement",
				sharedProjections, nil, false,
				fmt.Sprintf("focus_raw_layout:%s:%s", path, treeID))
			for _, f := range layout.Findings {
				findings = append(findings, policyFindingDict(f))
			}

			if initial.Value != nil && anyToString(initial.Value["kind"]) == "focus" {
				target := anyToString(initial.Value["focus_id"])
				if _, ok := layout.Positions[target]; !ok {
					findings = append(findings, map[string]any{
						"code": "focus_initial_position_target_missing", "severity": "blocker",
						"source": map[string]any{
							"kind": "focus_raw_file", "identifier": treeID, "path": path,
						},
						"evidence": map[string]any{
							"focus_id":            target,
							"available_focus_ids": sortedPositionIDs(layout.Positions),
						},
						"repair_route": "focus.initial_position.choose_target",
					})
				}
			}

			for _, node := range bundle.NodesData {
				rewardText := anyToString(node["completion_reward"])
				if strings.TrimSpace(rewardText) == "" {
					continue
				}
				focusID, sourceName := anyToString(node["focus_id"]), path
				reward := AnalyzeCompletionReward(CompletionRewardInput{
					Text: rewardText, FocusID: &focusID, SourceName: &sourceName,
				})
				for _, f := range reward.Findings {
					findings = append(findings, analysisFindingDict(f))
				}
			}

			isDefault := false
			if v, ok := bundle.TreeData["is_default"]; ok {
				isDefault = anyTruthy(v)
			}
			emissions = append(emissions, FocusEmission{
				TreeID: treeID, TreeKind: "primary", Path: path, Provider: "project_raw",
				ArtifactID: rawArtifactID, ContentHash: raw.ContentSHA256,
				IsDefault: isDefault, SourceBacked: true,
			})
		}
	}

	for _, f := range ValidateFocusTreeEmissions(emissions, "") {
		findings = append(findings, policyFindingDict(f))
	}

	knownDefaultPaths := map[string]bool{}
	vanillaGenericPath := ""
	if registryIndex != nil {
		if generic, ok := registryIndex.ByTreeID["generic_focus"]; ok {
			knownDefaultPaths[strings.ToLower(generic.CanonicalPath)] = true
			vanillaGenericPath = generic.CanonicalPath
		}
	}
	defaultSelection := SelectEffectiveFocusDefaults(FocusDefaultSelectionInput{
		Emissions: emissions, VanillaGenericPath: vanillaGenericPath,
		ReplacePaths: selectedReplacePaths, Source: "focus_defaults:effective_artifact",
	})

	dependencyNames := []string{}
	for _, value := range in.Project.Dependencies {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			dependencyNames = append(dependencyNames, trimmed)
		}
	}
	hasBaseLink := len(dependencyNames) > 0 || anyTruthy(in.Project.BasemodTemplateID)
	canChangeBaseDefault := false
	for _, emission := range emissions {
		treeKind := emission.TreeKind
		if treeKind == "" {
			treeKind = "primary"
		}
		if treeKind == "shared_pool" {
			continue
		}
		if emission.IsDefault || knownDefaultPaths[strings.ToLower(emission.Path)] ||
			emission.SourceBacked {
			canChangeBaseDefault = true
			break
		}
	}

	if hasBaseLink && canChangeBaseDefault && !defaultSelection.FocusDirectoryReplaced {
		identifier := strings.Join(dependencyNames, ",")
		if identifier == "" {
			identifier = anyToString(in.Project.BasemodTemplateID)
			if identifier == "" {
				identifier = "linked-base-mod"
			}
		}
		findings = append(findings, map[string]any{
			"code": "focus_base_default_provider_unresolved", "severity": "blocker",
			"source": map[string]any{
				"kind": "base_mod_focus_registry", "identifier": identifier,
			},
			"evidence": map[string]any{
				"dependencies":        dependencyNames,
				"basemod_template_id": in.Project.BasemodTemplateID,
				"reason": "the registered base-mod index does not yet contain " +
					"tree-level default/path/hash evidence",
			},
			"repair_route": "focus.base_mod_registry.refresh",
		})
	}
	for _, f := range defaultSelection.Findings {
		findings = append(findings, policyFindingDict(f))
	}
	return findings, nil
}

func sortedPositionIDs(positions map[string][2]int) []string {
	out := make([]string, 0, len(positions))
	for k := range positions {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedCopy(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	if out == nil {
		return []string{}
	}
	return out
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func nullableStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
