package focus

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	focusNamespaceRE    = regexp.MustCompile(`[^A-Za-z0-9_]+`)
	focusHasFocusTreeRE = regexp.MustCompile(`(?i)\bhas_focus_tree\s*=\s*"?([A-Za-z0-9_.:-]+)"?`)
)

func normaliseFocusPath(value string) (string, bool) {
	text := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if text == "" || strings.Contains(text, ":") {
		return "", false
	}
	for _, r := range text {
		if r < 32 {
			return "", false
		}
	}
	if strings.HasPrefix(text, "/") {
		return "", false
	}
	var parts []string
	for _, seg := range strings.Split(text, "/") {
		if seg == "" || seg == "." {
			continue
		}
		if seg == ".." {
			return "", false
		}
		parts = append(parts, seg)
	}
	switch len(parts) {
	case 1:
		parts = []string{"common", "national_focus", parts[0]}
	case 3:
		if parts[0] != "common" || parts[1] != "national_focus" {
			return "", false
		}
	default:
		return "", false
	}
	base := parts[2]
	dot := strings.LastIndexByte(base, '.')
	if dot <= 0 || dot >= len(base)-1 || !strings.EqualFold(base[dot:], ".txt") {
		return "", false
	}
	return strings.Join(parts, "/"), true
}

func safeFocusComponent(value, fallback string) string {
	cleaned := strings.Trim(focusNamespaceRE.ReplaceAllString(strings.TrimSpace(value), "_"), "_")
	if cleaned == "" {
		return fallback
	}
	return cleaned
}

func FocusTreeConditionalOffset(offsets []map[string]any, treeID string) (int, int) {
	dx, dy := 0, 0
	wanted := strings.TrimSpace(treeID)
	for _, item := range offsets {
		if item == nil {
			continue
		}
		trigger, _ := item["trigger"].(string)
		matched := false
		for _, m := range focusHasFocusTreeRE.FindAllStringSubmatch(trigger, -1) {
			if m[1] == wanted {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		dx += coerceInt(item["x"])
		dy += coerceInt(item["y"])
	}
	return dx, dy
}

func coerceInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return int(n)
		}
		if f, err := t.Float64(); err == nil {
			return int(f)
		}
	case string:
		var n int
		if _, err := fmt.Sscanf(strings.TrimSpace(t), "%d", &n); err == nil {
			return n
		}
	}
	return 0
}

type RegistryRecord struct {
	TreeID        string
	CanonicalPath string
	TreeKind      string
	TargetLayer   string
	ContentHash   string
}

type RegistrySnapshot struct {
	Records []RegistryRecord
}

func (r *FocusTreePathRegistry) PolicySnapshot() RegistrySnapshot {
	records := make([]RegistryRecord, 0, len(r.Records))
	for _, rec := range r.Records {
		records = append(records, RegistryRecord{
			TreeID:        rec.TreeID,
			CanonicalPath: rec.CanonicalPath,
			TreeKind:      "primary",
			TargetLayer:   "vanilla",
			ContentHash:   rec.ContentSHA256,
		})
	}
	return RegistrySnapshot{Records: records}
}

type RegistryIndex struct {
	Snapshot RegistrySnapshot
	ByTreeID map[string]RegistryRecord
	ByPath   map[string]RegistryRecord
	Findings []FocusPolicyFinding
}

func (idx RegistryIndex) Blocked() bool {
	for _, f := range idx.Findings {
		if f.Severity == "blocker" {
			return true
		}
	}
	return false
}

func BuildPrimaryRegistry(snapshot RegistrySnapshot, source string) RegistryIndex {
	if source == "" {
		source = "focus_registry"
	}
	var findings []FocusPolicyFinding
	byTreeID := map[string]RegistryRecord{}
	byPath := map[string]RegistryRecord{}

	for _, record := range snapshot.Records {
		if record.TreeKind == "shared_pool" {
			findings = append(findings, FocusPolicyFinding{
				Code: "focus_registry_shared_pool_excluded", Severity: "info", Source: source,
				Evidence:    map[string]any{"tree_id": record.TreeID, "path": record.CanonicalPath},
				RepairRoute: "focus_shared_pool_registry",
			})
			continue
		}
		treeID := strings.TrimSpace(record.TreeID)
		path, ok := normaliseFocusPath(record.CanonicalPath)
		if treeID == "" || !ok {
			findings = append(findings, FocusPolicyFinding{
				Code: "focus_registry_invalid_record", Severity: "blocker", Source: source,
				Evidence: map[string]any{
					"tree_id":        nullableTag(treeID),
					"canonical_path": record.CanonicalPath,
				},
				RepairRoute: "focus_registry_refresh",
			})
			continue
		}
		normalised := RegistryRecord{
			TreeID: treeID, CanonicalPath: path, TreeKind: "primary",
			TargetLayer: record.TargetLayer, ContentHash: record.ContentHash,
		}
		if prior, dup := byTreeID[treeID]; dup {
			findings = append(findings, FocusPolicyFinding{
				Code: "focus_registry_duplicate_tree_id", Severity: "blocker", Source: source,
				Evidence: map[string]any{
					"tree_id": treeID,
					"paths":   sortedUniqueStrings(prior.CanonicalPath, path),
				},
				RepairRoute: "focus_registry_refresh",
			})
		}
		if prior, dup := byPath[path]; dup {
			findings = append(findings, FocusPolicyFinding{
				Code: "focus_registry_duplicate_path", Severity: "blocker", Source: source,
				Evidence: map[string]any{
					"path":     path,
					"tree_ids": sortedUniqueStrings(prior.TreeID, treeID),
				},
				RepairRoute: "focus_registry_refresh",
			})
		}
		if _, exists := byTreeID[treeID]; !exists {
			byTreeID[treeID] = normalised
		}
		if _, exists := byPath[path]; !exists {
			byPath[path] = normalised
		}
	}

	return RegistryIndex{Snapshot: snapshot, ByTreeID: byTreeID, ByPath: byPath, Findings: findings}
}

func sortedUniqueStrings(values ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

type FocusPolicyTree struct {
	TreeID     string
	TreeKind   string
	OriginKind string
	SourceFile string
}

type FocusFileDecision struct {
	TreeID         string
	OriginKind     string
	SourceFile     string
	EmitMode       string
	TargetPath     string
	TargetLayer    string
	DecisionSource string
	Findings       []FocusPolicyFinding
}

func (d FocusFileDecision) Blocked() bool {
	for _, f := range d.Findings {
		if f.Severity == "blocker" {
			return true
		}
	}
	return false
}

func DecideFocusFilePolicy(tree FocusPolicyTree, registry RegistryIndex, namespace, source string) FocusFileDecision {
	if source == "" {
		source = "focus_path"
	}
	treeID := strings.TrimSpace(tree.TreeID)
	treeKind := strings.TrimSpace(tree.TreeKind)
	if treeKind == "" {
		treeKind = "primary"
	}
	originKind := strings.TrimSpace(tree.OriginKind)
	if originKind == "" {
		if tree.SourceFile != "" {
			originKind = "imported"
		} else {
			originKind = "created"
		}
	}
	rawSourceFile := tree.SourceFile
	sourceFile, sourceFileOK := normaliseFocusPath(rawSourceFile)
	findings := append([]FocusPolicyFinding{}, registry.Findings...)
	targetPath := ""
	targetLayer := "project"
	decisionSource := ""
	emitMode := "blocked"

	if rawSourceFile != "" && !sourceFileOK {
		findings = append(findings, FocusPolicyFinding{
			Code: "focus_source_file_invalid", Severity: "blocker", Source: source,
			Evidence:    map[string]any{"tree_id": treeID, "source_file": rawSourceFile},
			RepairRoute: "focus_path_repair",
		})
	}

	namespacePart := safeFocusComponent(namespace, "mod")
	treePart := safeFocusComponent(treeID, "focus_tree")
	namespacedID := treePart
	if !strings.HasPrefix(strings.ToLower(treePart), strings.ToLower(namespacePart)+"_") {
		namespacedID = namespacePart + "_" + treePart
	}
	additionPath := "common/national_focus/" + namespacedID + ".txt"

	switch {
	case treeKind == "shared_pool":
		decisionSource = "shared_pool"
		if originKind == "imported" {
			targetPath = sourceFile
		} else {
			targetPath = additionPath
		}
		if originKind == "imported" && sourceFileOK {
			emitMode = "shared_pool_same_path"
		} else {
			emitMode = "shared_pool_namespaced_addition"
		}
		if originKind == "imported" && !sourceFileOK {
			findings = append(findings, FocusPolicyFinding{
				Code: "focus_imported_source_file_missing", Severity: "blocker", Source: source,
				Evidence:    map[string]any{"tree_id": treeID, "tree_kind": treeKind},
				RepairRoute: "focus_path_repair",
			})
			targetPath = ""
			emitMode = "blocked"
		}
	case treeKind != "primary":
		findings = append(findings, FocusPolicyFinding{
			Code: "focus_tree_kind_unknown", Severity: "blocker", Source: source,
			Evidence:    map[string]any{"tree_id": treeID, "tree_kind": treeKind},
			RepairRoute: "focus_path_repair",
		})
	default:
		registryRecord, hasRecord := registry.ByTreeID[treeID]
		switch originKind {
		case "imported":
			if hasRecord {
				targetPath = registryRecord.CanonicalPath
				targetLayer = registryRecord.TargetLayer
				decisionSource = "primary_registry"
				emitMode = "canonical_same_path"
				if sourceFileOK && sourceFile != targetPath {
					findings = append(findings, FocusPolicyFinding{
						Code: "focus_imported_source_registry_mismatch", Severity: "warning", Source: source,
						Evidence: map[string]any{
							"tree_id":        treeID,
							"source_file":    sourceFile,
							"canonical_path": targetPath,
						},
						RepairRoute: "focus_registry_review",
					})
				}
			} else if sourceFileOK {
				targetPath = sourceFile
				decisionSource = "import_metadata"
				emitMode = "canonical_same_path"
			} else {
				findings = append(findings, FocusPolicyFinding{
					Code: "focus_imported_source_file_missing", Severity: "blocker", Source: source,
					Evidence:    map[string]any{"tree_id": treeID, "tree_kind": treeKind},
					RepairRoute: "focus_path_repair",
				})
			}
		case "created":
			decisionSource = "created_tree"
			if hasRecord {
				findings = append(findings, FocusPolicyFinding{
					Code: "focus_created_primary_id_collision", Severity: "blocker", Source: source,
					Evidence: map[string]any{
						"tree_id":        treeID,
						"canonical_path": registryRecord.CanonicalPath,
					},
					RepairRoute: "focus_choose_unique_tree_id",
				})
			} else {
				targetPath = additionPath
				emitMode = "namespaced_addition"
			}
		default:
			findings = append(findings, FocusPolicyFinding{
				Code: "focus_origin_unknown", Severity: "blocker", Source: source,
				Evidence:    map[string]any{"tree_id": treeID, "origin_kind": nullableTag(originKind)},
				RepairRoute: "focus_path_repair",
			})
		}
	}

	return FocusFileDecision{
		TreeID: treeID, OriginKind: originKind, SourceFile: sourceFile,
		EmitMode: emitMode, TargetPath: targetPath, TargetLayer: targetLayer,
		DecisionSource: decisionSource, Findings: findings,
	}
}

type LayoutNode struct {
	FocusID            string
	X, Y               int
	RelativePositionID string
	Source             string
	OffsetX, OffsetY   int
}

type SharedProjectionPolicy struct {
	FocusID            string
	BaseX, BaseY       int
	RelativePositionID string
	OffsetX, OffsetY   int
	Source             string
	Condition          string
}

type LayoutResolution struct {
	Positions map[string][2]int
	Findings  []FocusPolicyFinding
	Mode      string
}

func (r LayoutResolution) Blocked() bool {
	for _, f := range r.Findings {
		if f.Severity == "blocker" {
			return true
		}
	}
	return false
}

func projectionAsLayoutNode(p SharedProjectionPolicy) LayoutNode {
	src := p.Source
	if src == "" {
		src = "shared_pool"
	}
	return LayoutNode{
		FocusID:            p.FocusID,
		X:                  p.BaseX + p.OffsetX,
		Y:                  p.BaseY + p.OffsetY,
		RelativePositionID: p.RelativePositionID,
		Source:             src,
	}
}

func resolveFocusLayout(nodes []LayoutNode, source string) (map[string][2]int, []FocusPolicyFinding) {
	var findings []FocusPolicyFinding
	byID := map[string]LayoutNode{}
	duplicateIDs := map[string]bool{}
	for _, node := range nodes {
		if node.FocusID == "" {
			src := node.Source
			if src == "" {
				src = "project"
			}
			findings = append(findings, FocusPolicyFinding{
				Code: "focus_layout_missing_focus_id", Severity: "blocker", Source: source,
				Evidence:    map[string]any{"node_source": src},
				RepairRoute: "focus_layout_repair",
			})
		} else if _, seen := byID[node.FocusID]; seen {
			duplicateIDs[node.FocusID] = true
		} else {
			byID[node.FocusID] = node
		}
	}
	for _, focusID := range sortedKeys(duplicateIDs) {
		findings = append(findings, FocusPolicyFinding{
			Code: "focus_layout_duplicate_focus_id", Severity: "blocker", Source: source,
			Evidence:    map[string]any{"focus_id": focusID},
			RepairRoute: "focus_layout_repair",
		})
	}

	positions := map[string][2]int{}
	var visiting []string
	reportedCycles := map[string]bool{}

	var resolve func(focusID string) ([2]int, bool)
	resolve = func(focusID string) ([2]int, bool) {
		if pos, ok := positions[focusID]; ok {
			return pos, true
		}
		for i, v := range visiting {
			if v == focusID {
				cycle := append(append([]string{}, visiting[i:]...), focusID)
				key := strings.Join(cycle, "\x00")
				if !reportedCycles[key] {
					reportedCycles[key] = true
					findings = append(findings, FocusPolicyFinding{
						Code: "focus_layout_relative_cycle", Severity: "blocker", Source: source,
						Evidence:    map[string]any{"cycle": cycle},
						RepairRoute: "focus_layout_repair",
					})
				}
				return [2]int{}, false
			}
		}
		node, ok := byID[focusID]
		if !ok {
			return [2]int{}, false
		}
		visiting = append(visiting, focusID)
		var position [2]int
		resolved := false
		if node.RelativePositionID != "" {
			anchor, hasAnchor := byID[node.RelativePositionID]
			if !hasAnchor {
				findings = append(findings, FocusPolicyFinding{
					Code: "focus_layout_missing_anchor", Severity: "blocker", Source: source,
					Evidence: map[string]any{
						"focus_id":             focusID,
						"relative_position_id": node.RelativePositionID,
					},
					RepairRoute: "focus_layout_repair",
				})
			} else if anchorPos, ok := resolve(anchor.FocusID); ok {
				position = [2]int{anchorPos[0] + node.X + node.OffsetX, anchorPos[1] + node.Y + node.OffsetY}
				resolved = true
			}
		} else {
			position = [2]int{node.X + node.OffsetX, node.Y + node.OffsetY}
			resolved = true
		}
		visiting = visiting[:len(visiting)-1]
		if resolved {
			positions[focusID] = position
		}
		return position, resolved
	}

	byIDKeys := make([]string, 0, len(byID))
	for k := range byID {
		byIDKeys = append(byIDKeys, k)
	}
	sort.Strings(byIDKeys)
	for _, focusID := range byIDKeys {
		resolve(focusID)
	}

	byCell := map[[2]int][]string{}
	for focusID, pos := range positions {
		byCell[pos] = append(byCell[pos], focusID)
	}
	cells := make([][2]int, 0, len(byCell))
	for cell := range byCell {
		cells = append(cells, cell)
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i][0] != cells[j][0] {
			return cells[i][0] < cells[j][0]
		}
		return cells[i][1] < cells[j][1]
	})
	for _, cell := range cells {
		ids := byCell[cell]
		if len(ids) > 1 {
			sort.Strings(ids)
			findings = append(findings, FocusPolicyFinding{
				Code: "focus_layout_same_cell", Severity: "blocker", Source: source,
				Evidence: map[string]any{
					"position":  []int{cell[0], cell[1]},
					"focus_ids": ids,
				},
				RepairRoute: "focus_layout_repair",
			})
		}
	}

	ordered := make([]string, 0, len(positions))
	for focusID := range positions {
		ordered = append(ordered, focusID)
	}
	sort.Strings(ordered)
	var nearPairs []any
	for i, leftID := range ordered {
		leftPos := positions[leftID]
		for _, rightID := range ordered[i+1:] {
			rightPos := positions[rightID]
			if leftPos[1] == rightPos[1] {
				distance := leftPos[0] - rightPos[0]
				if distance < 0 {
					distance = -distance
				}
				if distance == 1 {
					nearPairs = append(nearPairs, []any{leftID, rightID, 1})
				}
			}
		}
	}
	if len(nearPairs) > 0 {
		findings = append(findings, FocusPolicyFinding{
			Code: "focus_layout_adjacent_same_row", Severity: "warning", Source: source,
			Evidence:    map[string]any{"pairs": nearPairs},
			RepairRoute: "focus_layout_review",
		})
	}
	return positions, findings
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ValidateCandidateLayout(
	nodes []LayoutNode,
	mode string,
	sharedProjections []SharedProjectionPolicy,
	beforeNodes []LayoutNode,
	hasBefore bool,
	source string,
) LayoutResolution {
	if source == "" {
		source = "focus_layout"
	}
	candidate := append([]LayoutNode{}, nodes...)
	for _, p := range sharedProjections {
		candidate = append(candidate, projectionAsLayoutNode(p))
	}
	positions, findings := resolveFocusLayout(candidate, source)
	if mode != "replacement" && mode != "created_independent" {
		findings = append(findings, FocusPolicyFinding{
			Code: "focus_layout_unknown_mode", Severity: "blocker", Source: source,
			Evidence:    map[string]any{"mode": mode},
			RepairRoute: "focus_layout_repair",
		})
	}

	if mode == "replacement" && hasBefore {

		beforePositions, _ := resolveFocusLayout(beforeNodes, source+":before")
		var commonIDs []string
		for focusID := range beforePositions {
			if _, ok := positions[focusID]; ok {
				commonIDs = append(commonIDs, focusID)
			}
		}
		sort.Strings(commonIDs)
		translations := map[[2]int]bool{}
		for _, focusID := range commonIDs {
			translations[[2]int{
				positions[focusID][0] - beforePositions[focusID][0],
				positions[focusID][1] - beforePositions[focusID][1],
			}] = true
		}
		if len(commonIDs) > 0 && len(translations) == 1 && !translations[[2]int{0, 0}] {
			var shift [2]int
			for t := range translations {
				shift = t
			}
			findings = append(findings, FocusPolicyFinding{
				Code: "focus_layout_replacement_global_shift", Severity: "blocker", Source: source,
				Evidence: map[string]any{
					"focus_ids":   commonIDs,
					"translation": []int{shift[0], shift[1]},
				},
				RepairRoute: "focus_layout_restore_reference_frame",
			})
		}
	}

	return LayoutResolution{Positions: positions, Findings: findings, Mode: mode}
}
