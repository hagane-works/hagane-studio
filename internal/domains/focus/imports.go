package focus

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/drawio"
	"haganeworks/internal/hoi4/importsvc"
	treeimport "haganeworks/internal/hoi4/importsvc/trees"
	"haganeworks/internal/hoi4/loc"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

const maxSingleImportBytes = 256 << 20

type FocusTreeImportResult struct {
	TreeID          string         `json:"tree_id,omitempty"`
	Total           int            `json:"total"`
	TreeAction      string         `json:"tree_action"`
	CreatedNodes    int            `json:"created_nodes"`
	PreservedFields map[string]int `json:"preserved_fields"`
	Warnings        []string       `json:"warnings"`
	ConflictPolicy  string         `json:"conflict_policy"`
}

func singleImportPolicy(raw string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(raw))
	if p == "" {
		return "overwrite", nil
	}
	if p != "overwrite" && p != "skip" {
		return "", apiutil.BadRequestError(fmt.Sprintf("conflict_policy must be one of {'overwrite', 'skip'}, got %q", raw))
	}
	return p, nil
}

func planSingleImportBatch(keys []string, existing map[string]bool, policy string) (selected []int, actions []string) {
	actions = make([]string, len(keys))
	seen := make(map[string]bool, len(existing)+len(keys))
	for key, present := range existing {
		seen[key] = present
	}
	selectedByKey := map[string]int{}
	for index, key := range keys {
		if key == "" {
			continue
		}
		if seen[key] {
			if policy == "skip" {
				actions[index] = "skipped"
				continue
			}
			actions[index] = "updated"
		} else {
			actions[index] = "created"
			seen[key] = true
		}
		selectedByKey[key] = index
	}
	for _, index := range selectedByKey {
		selected = append(selected, index)
	}
	sort.Ints(selected)
	return selected, actions
}

func nativeTreeValue(value any) any {
	switch v := value.(type) {
	case *treeimport.RawDict:
		out := make(map[string]any, len(v.Keys))
		for _, key := range v.Keys {
			item, _ := v.Get(key)
			out[key] = nativeTreeValue(item)
		}
		return out
	case paradox.Scalar:
		switch v.Kind {
		case paradox.ScalarInt:
			if v.Int.IsInt64() {
				return v.Int.Int64()
			}
			return v.Int.String()
		case paradox.ScalarFloat:
			return v.Float
		default:
			return v.Str
		}
	case *big.Int:
		if v.IsInt64() {
			return v.Int64()
		}
		return v.String()
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = nativeTreeValue(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = nativeTreeValue(v[i])
		}
		return out
	case []map[string]any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = nativeTreeValue(v[i])
		}
		return out
	default:
		return value
	}
}

func singleString(row map[string]any, key, fallback string) string {
	if value, ok := row[key].(string); ok {
		return value
	}
	return fallback
}

func singleBool(row map[string]any, key string, fallback bool) bool {
	if value, ok := row[key].(bool); ok {
		return value
	}
	return fallback
}

func singleValue(row map[string]any, key string, fallback any) any {
	if value, ok := row[key]; ok && value != nil {
		return value
	}
	return fallback
}

func intValue(v any, fallback int) int {
	return colInt(project.Row{"v": plain(v)}, "v", fallback)
}

var focusNodeSnapshotKeys = []string{
	"ai_will_do", "allow_branch_ast", "available", "bypass", "cancel",
	"completion_reward", "cost", "description_locs", "desc_en", "desc_zh",
	"display_name_locs", "dynamic_icon", "effect_tooltip_locs", "focus_id",
	"icon", "load_focus_tree_rules", "mutually_exclusive", "name", "name_en",
	"name_zh", "offset", "prerequisites", "raw_extras", "relative_position_id",
	"search_filters", "select_effect", "text_icon", "x", "y",
}

func focusNodeImportRow(treeID int64, parsed map[string]any) project.Row {
	focusID := singleString(parsed, "focus_id", "")
	record := map[string]any{
		"focus_id": focusID, "name": singleString(parsed, "name", focusID),
		"name_en": singleString(parsed, "name_en", focusID), "name_zh": "",
		"desc_en": "", "desc_zh": "", "display_name_locs": map[string]any{},
		"description_locs": map[string]any{}, "effect_tooltip_locs": map[string]any{},
		"icon":         singleString(parsed, "icon", "GFX_goal_unknown"),
		"dynamic_icon": singleValue(parsed, "dynamic_icon", []any{}),
		"text_icon":    singleString(parsed, "text_icon", ""),
		"offset":       singleValue(parsed, "offset", []any{}),
		"cost":         singleValue(parsed, "cost", float64(10)),
		"x":            singleValue(parsed, "x", int64(0)), "y": singleValue(parsed, "y", int64(0)),
		"relative_position_id":     singleString(parsed, "relative_position_id", ""),
		"prerequisites":            singleValue(parsed, "prerequisites", []any{}),
		"mutually_exclusive":       singleValue(parsed, "mutually_exclusive", []any{}),
		"completion_reward":        singleString(parsed, "completion_reward", ""),
		"available":                singleString(parsed, "available", ""),
		"bypass":                   singleString(parsed, "bypass", ""),
		"cancel":                   singleString(parsed, "cancel", ""),
		"select_effect":            singleString(parsed, "select_effect", ""),
		"ai_will_do":               singleString(parsed, "ai_will_do", "factor = 1"),
		"search_filters":           singleValue(parsed, "search_filters", []any{}),
		"cancel_if_invalid":        singleBool(parsed, "cancel_if_invalid", true),
		"continue_if_invalid":      singleBool(parsed, "continue_if_invalid", false),
		"available_if_capitulated": singleBool(parsed, "available_if_capitulated", false),
		"will_lead_to_war_with":    singleValue(parsed, "will_lead_to_war_with", []any{}),
		"allow_branch_ast":         singleValue(parsed, "allow_branch_ast", map[string]any{}),
		"load_focus_tree_rules":    singleValue(parsed, "load_focus_tree_rules", []any{}),
		"raw_extras":               singleValue(parsed, "raw_extras", map[string]any{}),
	}
	snapshot := make(map[string]any, len(focusNodeSnapshotKeys))
	for _, key := range focusNodeSnapshotKeys {
		snapshot[key] = record[key]
	}
	row := project.Row{"focus_tree_id": treeID, "source": "created", "source_snapshot": plain(snapshot)}
	for key, value := range record {
		row[key] = plain(value)
	}
	row["cost"] = colFloat(project.Row{"v": plain(record["cost"])}, "v", 10)
	row["x"], row["y"] = intValue(record["x"], 0), intValue(record["y"], 0)
	return row
}

func importFocusTreeScript(p *project.Project, rv *revisions, text, sourceFilename, countryTag, conflictPolicy string) (FocusTreeImportResult, error) {
	policy, err := singleImportPolicy(conflictPolicy)
	if err != nil {
		return FocusTreeImportResult{}, err
	}
	bundles, summary, err := treeimport.ImportFocusFile(text, sourceFilename)
	if err != nil {
		return FocusTreeImportResult{}, apiutil.ValidationError(err.Error())
	}
	preserved := map[string]int{}
	if summary.PreservedFields != nil {
		for _, key := range summary.PreservedFields.Keys {
			preserved[key] = summary.PreservedFields.Get(key)
		}
	}
	result := FocusTreeImportResult{Total: summary.Total, TreeAction: "none", PreservedFields: preserved,
		Warnings: append([]string{}, summary.Warnings...), ConflictPolicy: policy}
	if len(bundles) == 0 {
		if len(result.Warnings) == 0 {
			result.Warnings = []string{"No focus_tree found in file"}
		}
		return result, nil
	}
	err = p.Write([]string{"focus_trees", "focus_nodes", "focus_file_policies"}, func(tx *project.Tx) error {
		record := tx.Record()
		projectTags := normaliseFocusTags(projectCountryTags(record))
		type staged struct {
			tree  project.Row
			nodes []map[string]any
		}
		stagedTrees := make([]staged, len(bundles))
		keys := make([]string, len(bundles))
		for i, bundle := range bundles {
			nativeTree := nativeTreeValue(bundle.TreeData).(map[string]any)
			treeID := singleString(nativeTree, "tree_id", "")
			keys[i] = treeID
			result.TreeID = treeID
			var verdict *FocusScopeVerdict
			if parsed, ok := bundle.TreeData["_scope_verdict"].(importsvc.ScopeVerdict); ok {
				verdict = &FocusScopeVerdict{Kind: string(parsed.Kind), CountryTags: parsed.CountryTags}
			}
			if countryTag != "" && bundle.TreeKind == "primary" {
				verdict = &FocusScopeVerdict{Kind: "own", CountryTags: []string{strings.ToUpper(countryTag)}}
			}
			materialized := MaterializeFocusBundle(FocusScopeCandidate{TreeKind: bundle.TreeKind, TreeID: treeID}, projectTags, verdict, sourceFilename)
			if materialized.Blocked() {
				return DetailError{Status: 422, Detail: map[string]any{
					"error": "focus_scope_blocked", "findings": materialized.Findings,
				}}
			}
			var countryValue, sharedTags any
			if materialized.CountryTag != "" {
				countryValue = materialized.CountryTag
			}
			if len(materialized.SharedCountryTags) > 0 {
				sharedTags = plain(materialized.SharedCountryTags)
			}
			nodes := make([]map[string]any, len(bundle.NodesData))
			for j := range bundle.NodesData {
				nodes[j] = nativeTreeValue(bundle.NodesData[j]).(map[string]any)
			}
			treeKind := bundle.TreeKind
			if treeKind == "" {
				treeKind = "primary"
			}
			stagedTrees[i] = staged{nodes: nodes, tree: project.Row{
				"edit_revision": int64(1), "tree_id": treeID, "country_tag": countryValue,
				"scope_kind": materialized.ScopeKind, "shared_country_tags": sharedTags,
				"is_default": singleBool(nativeTree, "is_default", false), "tree_kind": treeKind,
				"shared_focus_refs":           plain(singleValue(map[string]any{"v": nativeTreeValue(bundle.SharedFocusRefs)}, "v", []any{})),
				"source_file":                 sourceFilename,
				"continuous_focus_position_x": intValue(singleValue(nativeTree, "continuous_focus_position_x", int64(50)), 50),
				"continuous_focus_position_y": intValue(singleValue(nativeTree, "continuous_focus_position_y", int64(1000)), 1000),
				"initial_show_position":       plain(singleValue(nativeTree, "initial_show_position", map[string]any{})),
				"initial_show_focus":          singleString(nativeTree, "initial_show_focus", ""),
				"source":                      "created",
				"raw_extras":                  plain(singleValue(nativeTree, "raw_extras", map[string]any{})),
			}}
		}

		existing := map[string]bool{}
		for _, row := range tx.Rows("focus_trees") {
			existing[colStr(row, "tree_id", "")] = true
		}
		selected, actions := planSingleImportBatch(keys, existing, policy)
		if len(actions) > 0 && actions[0] != "" {
			result.TreeAction = actions[0]
		}
		for i, action := range actions {
			if action != "" && action != "skipped" {
				result.CreatedNodes += len(bundles[i].NodesData)
			}
		}
		selectedKeys := map[string]bool{}
		for _, index := range selected {
			selectedKeys[keys[index]] = true
		}
		if policy == "overwrite" && len(selectedKeys) > 0 {
			replaced := map[int64]bool{}
			for _, row := range append([]project.Row(nil), tx.Rows("focus_trees")...) {
				if selectedKeys[colStr(row, "tree_id", "")] {
					id := project.Int64(row["id"])
					if _, err := rv.advance(tx, id); err != nil {
						return err
					}
					replaced[id] = true
					delete(rv.advanced, id)
				}
			}
			tx.Delete("focus_trees", func(r project.Row) bool { return replaced[project.Int64(r["id"])] })
			tx.Delete("focus_nodes", func(r project.Row) bool { return replaced[project.Int64(r["focus_tree_id"])] })
		}
		for _, index := range selected {
			if exists := treeIDTaken(tx, keys[index]); exists {
				continue
			}
			row := tx.Insert("focus_trees", stagedTrees[index].tree)
			id := project.Int64(row["id"])
			rv.created(id)
			for _, node := range stagedTrees[index].nodes {
				tx.Insert("focus_nodes", focusNodeImportRow(id, node))
			}
		}

		wanted := map[string]bool{}
		for _, key := range keys {
			if key != "" {
				wanted[key] = true
			}
		}
		var policyTrees []FocusPolicyTree
		for _, row := range txData(tx).trees {
			if wanted[colStr(row, "tree_id", "")] {
				policyTrees = append(policyTrees, FocusPolicyTree{
					TreeID: colStr(row, "tree_id", ""), TreeKind: colStr(row, "tree_kind", "primary"),
					SourceFile: colStr(row, "source_file", ""),
				})
			}
		}
		if len(policyTrees) > 0 {
			return persistPolicies(tx, policyTrees, projectNamespace(record))
		}
		return nil
	})
	if err != nil {
		return FocusTreeImportResult{}, err
	}
	rv.commit(p)
	return result, nil
}

func treeIDTaken(tx *project.Tx, treeID string) bool {
	for _, row := range tx.Rows("focus_trees") {
		if colStr(row, "tree_id", "") == treeID {
			return true
		}
	}
	return false
}

func readSingleImportUpload(w http.ResponseWriter, r *http.Request) (string, string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSingleImportBytes+(4<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return "", "", apiutil.ValidationError("invalid multipart form")
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return "", "", apiutil.ValidationError("file is required")
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxSingleImportBytes+1))
	if err != nil {
		return "", "", apiutil.ValidationError("failed to read upload")
	}
	if len(raw) > maxSingleImportBytes {
		return "", "", statusError(http.StatusRequestEntityTooLarge, "import file too large")
	}
	text, _, ok := loc.DecodeScriptText(raw)
	if !ok {
		return "", "", apiutil.ValidationError("file must be UTF-8 or Windows-1252 text")
	}
	filename := ""
	if header != nil {
		filename = header.Filename
	}
	return text, filename, nil
}

func (h *handlers) importFocusTree(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	text, filename, err := readSingleImportUpload(w, r)
	if err != nil {
		return err
	}
	defer r.MultipartForm.RemoveAll()
	result, err := importFocusTreeScript(p, rv, text, filename, r.URL.Query().Get("country_tag"), r.FormValue("conflict_policy"))
	if err != nil {
		return err
	}
	respond(w, rv, http.StatusOK, result)
	return nil
}

type DrawIOImportResult struct {
	TreeID             string   `json:"tree_id"`
	TreeDBID           int64    `json:"tree_db_id"`
	FocusesCreated     int      `json:"focuses_created"`
	PrereqEdgesCreated int      `json:"prereq_edges_created"`
	Warnings           []string `json:"warnings"`
}

type drawIONode struct {
	FocusID       string
	Name          string
	X, Y          int
	Prerequisites [][]string
}

type drawIOPlan struct {
	TreeBase string
	Tag      string
	Parsed   drawio.Result
	Nodes    []drawIONode
}

func prepareDrawIO(text, effectiveTag, treeName string) (drawIOPlan, error) {
	tag := strings.ToUpper(effectiveTag)
	withoutPrefix, err := drawio.Parse(text, "")
	if err != nil {
		return drawIOPlan{}, err
	}
	usePrefix := false
	if tag != "" && len(withoutPrefix.Nodes) > 0 {
		prefix := tag + "_"
		prefixed := 0
		for _, node := range withoutPrefix.Nodes {
			if len(node.FocusID) >= len(prefix) && strings.EqualFold(node.FocusID[:len(prefix)], prefix) {
				prefixed++
			}
		}
		usePrefix = float64(prefixed) < float64(len(withoutPrefix.Nodes))/2
	}
	prefix := ""
	if usePrefix {
		prefix = tag
	}
	parsed, err := drawio.Parse(text, prefix)
	if err != nil {
		return drawIOPlan{}, err
	}
	prerequisites := map[string][]string{}
	for _, edge := range parsed.Edges {
		prerequisites[edge.Target] = append(prerequisites[edge.Target], edge.Source)
	}
	nodes := make([]drawIONode, 0, len(parsed.Nodes))
	for _, node := range parsed.Nodes {
		groups := make([][]string, 0, len(prerequisites[node.FocusID]))
		for _, source := range prerequisites[node.FocusID] {
			groups = append(groups, []string{source})
		}
		nodes = append(nodes, drawIONode{FocusID: node.FocusID, Name: node.Name, X: node.X, Y: node.Y, Prerequisites: groups})
	}
	base := treeName
	if base == "" {
		base = "drawio_imported"
	}
	base = drawio.SanitizeFocusID(base)
	if tag != "" && !strings.HasPrefix(strings.ToLower(base), strings.ToLower(tag)) {
		base = strings.ToLower(tag) + "_" + base
	}
	return drawIOPlan{TreeBase: base, Tag: tag, Parsed: parsed, Nodes: nodes}, nil
}

func importDrawIO(p *project.Project, rv *revisions, text, countryTag, treeName string) (DrawIOImportResult, error) {
	result := DrawIOImportResult{Warnings: []string{}}
	effectiveTag := countryTag
	if effectiveTag == "" {
		effectiveTag = colStr(p.Record(), "country_tag", "")
	}
	plan, err := prepareDrawIO(text, effectiveTag, treeName)
	if err != nil {
		return result, apiutil.BadRequestError(err.Error())
	}
	registry, err := LoadFocusTreePathRegistry()
	if err != nil {
		return result, err
	}
	err = p.Write([]string{"focus_trees", "focus_nodes", "focus_file_policies"}, func(tx *project.Tx) error {
		unavailable := map[string]bool{}
		for _, row := range tx.Rows("focus_trees") {
			unavailable[colStr(row, "tree_id", "")] = true
		}
		for id := range registry.ByTreeID {
			unavailable[id] = true
		}
		treeID := plan.TreeBase
		for suffix := 2; unavailable[treeID]; suffix++ {
			treeID = fmt.Sprintf("%s_%d", plan.TreeBase, suffix)
		}
		var countryValue any
		if plan.Tag != "" {
			countryValue = plan.Tag
		}
		tree := tx.Insert("focus_trees", project.Row{
			"edit_revision": int64(1), "tree_id": treeID, "country_tag": countryValue, "is_default": false,
			"tree_kind": "primary", "shared_focus_refs": []any{}, "source_file": "",
			"continuous_focus_position_x": 50, "continuous_focus_position_y": 1000,
			"initial_show_position": map[string]any{}, "initial_show_focus": "", "scope_kind": "own",
			"shared_country_tags": nil, "source": "created", "raw_extras": map[string]any{},
		})
		result.TreeDBID = project.Int64(tree["id"])
		rv.created(result.TreeDBID)
		for _, node := range plan.Nodes {
			tx.Insert("focus_nodes", project.Row{
				"focus_tree_id": result.TreeDBID, "focus_id": node.FocusID, "name": node.Name,
				"icon": "GFX_goal_unknown", "cost": 10.0, "x": node.X, "y": node.Y, "relative_position_id": "",
				"prerequisites": plain(node.Prerequisites), "mutually_exclusive": []any{},
				"completion_reward": "", "available": "", "bypass": "", "cancel": "", "select_effect": "",
				"ai_will_do": "factor = 1", "search_filters": []any{}, "cancel_if_invalid": true,
				"continue_if_invalid": false, "available_if_capitulated": false, "will_lead_to_war_with": []any{},
				"allow_branch_ast": map[string]any{}, "load_focus_tree_rules": []any{}, "raw_extras": map[string]any{},
				"name_en": "", "name_zh": "", "desc_en": "", "desc_zh": "", "display_name_locs": map[string]any{},
				"description_locs": map[string]any{}, "effect_tooltip_locs": map[string]any{}, "source": "created",
				"source_snapshot": nil, "offset": []any{}, "dynamic_icon": []any{}, "text_icon": "",
			})
		}
		_, err := persistFocusFilePolicies(tx, []FocusPolicyTree{{TreeID: treeID, TreeKind: "primary", OriginKind: "created"}}, plan.Tag)
		var conflict focusPolicyConflict
		if errors.As(err, &conflict) {

			return apiutil.BadRequestError(conflict.Error())
		}
		if err != nil {
			return err
		}
		result.TreeID = treeID
		return nil
	})
	if err != nil {
		return DrawIOImportResult{Warnings: []string{}}, err
	}
	rv.commit(p)
	result.FocusesCreated = len(plan.Nodes)
	result.PrereqEdgesCreated = len(plan.Parsed.Edges)
	if plan.Parsed.Warnings != nil {
		result.Warnings = plan.Parsed.Warnings
	}
	return result, nil
}

func decodeDrawIOBytes(raw []byte) string {
	if len(raw) >= 3 && raw[0] == 0xef && raw[1] == 0xbb && raw[2] == 0xbf {
		if utf8.Valid(raw[3:]) {
			return string(raw[3:])
		}
	} else if utf8.Valid(raw) {
		return string(raw)
	}
	runes := make([]rune, len(raw))
	for i, b := range raw {
		runes[i] = rune(b)
	}
	return string(runes)
}

func (h *handlers) importDrawIO(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, drawio.MaxBytes+(4<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		return apiutil.BadRequestError("failed to read upload: " + err.Error())
	}
	defer r.MultipartForm.RemoveAll()
	file, _, err := r.FormFile("file")
	if err != nil {
		return apiutil.ValidationError("file is required")
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, drawio.MaxBytes+1))
	if err != nil {
		return apiutil.BadRequestError("failed to read upload: " + err.Error())
	}
	if len(raw) > drawio.MaxBytes {
		return apiutil.BadRequestError("Draw.io payload too large: upload exceeds cap of 8388608 bytes")
	}
	result, err := importDrawIO(p, rv, decodeDrawIOBytes(raw), r.FormValue("country_tag"), r.FormValue("tree_name"))
	if err != nil {
		return err
	}
	respond(w, rv, http.StatusOK, result)
	return nil
}
