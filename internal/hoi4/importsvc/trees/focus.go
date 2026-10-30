package trees

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

var focusKnownFields = map[string]bool{
	"id": true, "icon": true, "text_icon": true, "offset": true, "cost": true,
	"x": true, "y": true, "relative_position_id": true,
	"prerequisite": true, "mutually_exclusive": true,
	"completion_reward": true, "available": true, "bypass": true, "cancel": true,
	"select_effect": true, "ai_will_do": true, "search_filters": true,
	"allow_branch":      true,
	"bypass_effect":     true,
	"cancel_if_invalid": true, "continue_if_invalid": true,
	"available_if_capitulated": true,
	"will_lead_to_war_with":    true,
}

var treeKnownFields = map[string]bool{
	"id": true, "country": true, "default": true,
	"continuous_focus_position": true,
	"initial_show_position":     true, "focus": true, "shared_focus": true,
}

type varsLookup interface {
	get(name string) (any, bool)
}

type focusVars map[string]float64

func (v focusVars) get(n string) (any, bool) {
	f, ok := v[n]
	if !ok {
		return nil, false
	}
	return f, ok
}

func resolveVar(val paradox.Scalar, vars varsLookup) (any, error) {
	if val.Kind == paradox.ScalarStr && strings.HasPrefix(val.Str, "@") {
		if v, ok := vars.get(val.Str); ok {
			return v, nil
		}
		return big.NewInt(0), nil
	}
	if !pyTruthy(val) {
		return big.NewInt(0), nil
	}
	f, err := pyFloatScalar(val)
	if err != nil {
		if err == errFloatOverflow {
			return nil, err
		}
		return big.NewInt(0), nil
	}
	if z, ok := pyIntFromFloat(f); ok {
		return z, nil
	}
	if math.IsNaN(f) {
		return big.NewInt(0), nil
	}
	return nil, errIntInf
}

func resolveVarFloat(val paradox.Scalar, vars varsLookup) (float64, error) {
	if val.Kind == paradox.ScalarStr && strings.HasPrefix(val.Str, "@") {
		if v, ok := vars.get(val.Str); ok {
			switch t := v.(type) {
			case float64:
				return t, nil
			case *big.Int:
				if f, ok2 := pyFloatFromBigInt(t); ok2 {
					return f, nil
				}

				return 0, errFloatOverflow
			}
		}
		return 0.0, nil
	}
	if !pyTruthy(val) {
		return 0.0, nil
	}
	f, err := pyFloatScalar(val)
	if err != nil {
		if err == errFloatOverflow {
			return 0, err
		}
		return 0.0, nil
	}
	return f, nil
}

func ParseConditionAST(entries []*paradox.Entry) []any {
	children := make([]any, 0)
	for _, e := range entries {
		if e.Key == nil {
			continue
		}
		if e.Block != nil {
			children = append(children, map[string]any{
				"op":       *e.Key,
				"kind":     "block",
				"children": ParseConditionAST(e.Block.Entries),
			})
		} else {
			children = append(children, map[string]any{
				"op":    *e.Key,
				"kind":  "scalar",
				"value": pyStr(e.Scalar),
			})
		}
	}
	return children
}

func extractLoadFocusTreeRules(entries []*paradox.Entry) []map[string]any {
	rules := []map[string]any{}
	var walk func(es []*paradox.Entry, limitAST map[string]any)
	walk = func(es []*paradox.Entry, limitAST map[string]any) {
		for _, e := range es {
			if e.Key != nil && *e.Key == "load_focus_tree" && e.Block == nil {
				ca := limitAST
				if ca == nil {
					ca = map[string]any{}
				}
				rules = append(rules, map[string]any{
					"target_tree_id": pyStrip(pyStr(e.Scalar)),
					"condition_ast":  ca,
				})
				continue
			}
			if e.Block == nil {
				continue
			}
			if e.Key != nil && (*e.Key == "if" || *e.Key == "else_if") {
				var nested map[string]any
				for _, inner := range e.Block.Entries {
					if inner.Key != nil && *inner.Key == "limit" && inner.Block != nil {
						nested = map[string]any{
							"children": ParseConditionAST(inner.Block.Entries),
						}
						break
					}
				}
				if nested != nil {
					walk(e.Block.Entries, nested)
				} else {
					walk(e.Block.Entries, limitAST)
				}
			} else {

				walk(e.Block.Entries, limitAST)
			}
		}
	}
	walk(entries, nil)
	return rules
}

func extractCountryBlockTags(treeBlock []*paradox.Entry) []string {
	ctry := ppBlock(treeBlock, "country")
	if !truthyBlock(ctry) {
		return []string{}
	}
	tags := map[string]bool{}
	for _, me := range ctry.Entries {
		if me.Key == nil || *me.Key != "modifier" || me.Block == nil {
			continue
		}
		for _, inner := range me.Block.Entries {
			if inner.Key == nil || inner.Block != nil {
				continue
			}
			k := *inner.Key
			if k == "tag" || k == "original_tag" || k == "has_original_tag" {
				val := pyUpper(pyStrip(pyStr(inner.Scalar)))
				if len([]rune(val)) == 3 && pyIsAlnum(val) {
					tags[val] = true
				}
			}
		}
	}
	out := make([]string, 0, len(tags))
	for k := range tags {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func parseInitialShowPosition(
	isp *paradox.Block,
) (map[string]any, string, error) {
	if isp == nil {
		return map[string]any{}, "", nil
	}
	exactRaw := importsvc.BlockToText(isp, 0)
	invalid := func() (map[string]any, string, error) {
		return map[string]any{
			"kind": "invalid", "raw": exactRaw, "raw_text": exactRaw,
		}, "", nil
	}

	for _, entry := range isp.Entries {
		if entry.Key == nil {
			return invalid()
		}
		switch *entry.Key {
		case "focus", "x", "y", "offset":
		default:
			return invalid()
		}
	}

	focusEntries := findAll(isp.Entries, "focus")
	xEntries := findAll(isp.Entries, "x")
	yEntries := findAll(isp.Entries, "y")
	offsetEntries := findAll(isp.Entries, "offset")
	if len(focusEntries) > 0 {
		if len(focusEntries) != 1 || len(xEntries) > 0 || len(yEntries) > 0 ||
			len(offsetEntries) > 0 || len(isp.Entries) != 1 || focusEntries[0].Block != nil {
			return invalid()
		}
		focusID := pyStrip(pyStr(focusEntries[0].Scalar))
		if focusID == "" {
			return invalid()
		}
		return map[string]any{"kind": "focus", "focus_id": focusID}, focusID, nil
	}

	if len(xEntries) != 1 || len(yEntries) != 1 ||
		xEntries[0].Block != nil || yEntries[0].Block != nil {
		return invalid()
	}
	x, ok := parseInitialPositionInt(xEntries[0].Scalar)
	if !ok {
		return invalid()
	}
	y, ok := parseInitialPositionInt(yEntries[0].Scalar)
	if !ok {
		return invalid()
	}
	offsets := []map[string]any{}
	for _, entry := range offsetEntries {
		if entry.Block == nil {
			return invalid()
		}
		for _, inner := range entry.Block.Entries {
			if inner.Key == nil {
				return invalid()
			}
			switch *inner.Key {
			case "x", "y", "trigger":
			default:
				return invalid()
			}
		}
		oxEntries := findAll(entry.Block.Entries, "x")
		oyEntries := findAll(entry.Block.Entries, "y")
		triggerEntries := findAll(entry.Block.Entries, "trigger")
		if len(oxEntries) != 1 || len(oyEntries) != 1 || len(triggerEntries) != 1 ||
			oxEntries[0].Block != nil || oyEntries[0].Block != nil ||
			triggerEntries[0].Block == nil {
			return invalid()
		}
		ox, ok := parseInitialPositionInt(oxEntries[0].Scalar)
		if !ok {
			return invalid()
		}
		oy, ok := parseInitialPositionInt(oyEntries[0].Scalar)
		if !ok {
			return invalid()
		}
		triggerBlock := triggerEntries[0].Block
		trigger := importsvc.BlockToText(triggerBlock, 0)
		if pyStrip(trigger) == "" {
			return invalid()
		}
		offsets = append(offsets, map[string]any{
			"x": ox, "y": oy, "trigger": trigger,
		})
	}
	return map[string]any{
		"kind": "coordinates", "x": x, "y": y, "offsets": offsets,
	}, "", nil
}

func parseInitialPositionInt(value paradox.Scalar) (*big.Int, bool) {
	if value.Kind == paradox.ScalarInt {
		return new(big.Int).Set(value.Int), true
	}
	if value.Kind != paradox.ScalarStr {
		return nil, false
	}
	token := strings.TrimSpace(value.Str)
	if token == "" {
		return nil, false
	}
	digits := token
	if digits[0] == '+' || digits[0] == '-' {
		digits = digits[1:]
	}
	if digits == "" {
		return nil, false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return nil, false
		}
	}
	parsed, ok := new(big.Int).SetString(token, 10)
	return parsed, ok
}

func mapOneTreeBlock(
	treeBlock []*paradox.Entry,
	summary *importsvc.ImportSummary,
	vars focusVars,
) (map[string]any, []map[string]any, []map[string]any, error) {
	treeData := map[string]any{
		"tree_id":                     pyStr(scalarOr(treeBlock, "id", strScalar("imported_tree"))),
		"is_default":                  pyStr(scalarOr(treeBlock, "default", strScalar("no"))) == "yes",
		"continuous_focus_position_x": big.NewInt(50),
		"continuous_focus_position_y": big.NewInt(1000),
		"initial_show_position":       map[string]any{},
		"initial_show_focus":          "",
		"raw_extras":                  NewRawDict(),
	}

	if cfp := ppBlock(treeBlock, "continuous_focus_position"); truthyBlock(cfp) {
		x, err := pyIntScalar(scalarOr(cfp.Entries, "x", intScalar(50)))
		if err != nil {
			return nil, nil, nil, err
		}
		y, err := pyIntScalar(scalarOr(cfp.Entries, "y", intScalar(1000)))
		if err != nil {
			return nil, nil, nil, err
		}
		treeData["continuous_focus_position_x"] = x
		treeData["continuous_focus_position_y"] = y
	}

	if isp := ppBlock(treeBlock, "initial_show_position"); truthyBlock(isp) {
		position, legacyFocus, err := parseInitialShowPosition(isp)
		if err != nil {
			return nil, nil, nil, err
		}
		treeData["initial_show_position"] = position
		treeData["initial_show_focus"] = legacyFocus
		if position["kind"] == "invalid" {
			summary.Warnings = append(summary.Warnings,
				"focus.initial_position.invalid_shape: imported opening-camera data requires repair")
		}
	}

	var countryEntries []*paradox.Entry
	if cb := ppBlock(treeBlock, "country"); cb != nil {
		countryEntries = cb.Entries
		if countryEntries == nil {
			countryEntries = []*paradox.Entry{}
		}
	}
	treeData["_scope_verdict"] = importsvc.ClassifyFocusTreeCountryBlock(countryEntries)
	treeData["_country_tags_hint"] = extractCountryBlockTags(treeBlock)

	rawExtras := treeData["raw_extras"].(*RawDict)
	for _, e := range treeBlock {
		if e.Key != nil && *e.Key != "" && !treeKnownFields[*e.Key] {
			if e.Block != nil {
				rawExtras.Accumulate(*e.Key, importsvc.BlockToText(e.Block, 0))
			} else {
				rawExtras.Accumulate(*e.Key, e.Scalar)
			}
			summary.AddPreserved(*e.Key)
		}
	}

	nodes := []map[string]any{}
	for _, fe := range findAll(treeBlock, "focus") {
		if fe.Block == nil {
			continue
		}
		node, err := mapOneFocus(fe.Block.Entries, summary, vars)
		if err != nil {
			return nil, nil, nil, err
		}
		nodes = append(nodes, node)
	}

	sharedRefs := []map[string]any{}
	for _, e := range treeBlock {
		if e.Key == nil || *e.Key != "shared_focus" {
			continue
		}
		if e.Block != nil {
			continue
		}
		focusID := pyStrip(pyStr(e.Scalar))
		if focusID != "" {
			sharedRefs = append(sharedRefs, map[string]any{
				"focus_id": focusID,
				"offset_x": big.NewInt(0),
				"offset_y": big.NewInt(0),
			})
		}
	}

	return treeData, nodes, sharedRefs, nil
}

func stemFromFilename(sourceFilename, fallback string) string {
	if sourceFilename == "" {
		return fallback
	}
	base := strings.ReplaceAll(sourceFilename, "\\", "/")
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	stem := base
	if i := strings.LastIndex(base, "."); i >= 0 {
		stem = base[:i]
	}
	if stem == "" {
		return fallback
	}
	return stem
}

func ImportFocusFile(text, sourceFilename string) ([]*importsvc.TreeBundle, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("focus_node")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, nil, err
	}

	expandAtConstants(ast.Entries)

	variables := focusVars{}
	for _, e := range ast.Entries {
		if e.Key != nil && *e.Key != "" && strings.HasPrefix(*e.Key, "@") && e.Block == nil {
			if f, ok := pyFloat(pyStr(e.Scalar)); ok {
				variables[*e.Key] = f
			}
		}
	}

	bundles := []*importsvc.TreeBundle{}

	for _, te := range findAll(ast.Entries, "focus_tree") {
		if te.Block == nil {
			continue
		}
		treeData, nodesData, sharedRefs, err := mapOneTreeBlock(te.Block.Entries, summary, variables)
		if err != nil {
			return nil, nil, err
		}
		bundles = append(bundles, &importsvc.TreeBundle{
			TreeData:        treeData,
			NodesData:       nodesData,
			TreeKind:        "primary",
			SharedFocusRefs: sharedRefs,
		})
	}

	poolNodes := []map[string]any{}
	for _, e := range ast.Entries {
		if e.Block == nil || e.Key == nil {
			continue
		}
		switch *e.Key {
		case "shared_focus", "joint_focus":
			node, err := mapOneFocus(e.Block.Entries, summary, variables)
			if err != nil {
				return nil, nil, err
			}

			node["raw_extras"].(*RawDict).Set("_origin_keyword", *e.Key)
			poolNodes = append(poolNodes, node)
		}
	}

	if len(poolNodes) > 0 {
		poolTreeID := stemFromFilename(sourceFilename, "shared_pool")
		poolTreeData := map[string]any{
			"tree_id":                     poolTreeID,
			"is_default":                  false,
			"continuous_focus_position_x": big.NewInt(50),
			"continuous_focus_position_y": big.NewInt(1000),
			"initial_show_position":       map[string]any{},
			"initial_show_focus":          "",
			"raw_extras":                  NewRawDict(),
		}
		bundles = append(bundles, &importsvc.TreeBundle{
			TreeData:        poolTreeData,
			NodesData:       poolNodes,
			TreeKind:        "shared_pool",
			SharedFocusRefs: []map[string]any{},
		})
	}

	if len(bundles) == 0 {
		summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
			"no_content_focus",
			nil,
			"A file in the focus tree folder had no focus trees in it — skipped.",
		))
	}

	total := 0
	for _, b := range bundles {
		total += len(b.NodesData)
	}
	summary.Total = total
	return bundles, summary, nil
}

func ImportFocusTree(text string) (map[string]any, []map[string]any, *importsvc.ImportSummary, error) {
	bundles, summary, err := ImportFocusFile(text, "")
	if err != nil {
		return nil, nil, nil, err
	}
	if len(bundles) == 0 {
		return map[string]any{}, []map[string]any{}, summary, nil
	}
	if len(bundles) > 1 {
		summary.Warnings = append(summary.Warnings, fmt.Sprintf(
			"This file contains %d focus trees; only the first was "+
				"loaded through this path.", len(bundles)))
	}
	first := bundles[0]
	return first.TreeData, first.NodesData, summary, nil
}

func mapOneFocus(
	body []*paradox.Entry,
	summary *importsvc.ImportSummary,
	vars focusVars,
) (map[string]any, error) {
	cost, err := resolveVarFloat(scalarOr(body, "cost", floatScalar(10.0)), vars)
	if err != nil {
		return nil, err
	}
	if cost == 0 {
		cost = 10.0
	}
	x, err := resolveVar(scalarOr(body, "x", intScalar(0)), vars)
	if err != nil {
		return nil, err
	}
	y, err := resolveVar(scalarOr(body, "y", intScalar(0)), vars)
	if err != nil {
		return nil, err
	}

	focusID := pyStr(scalarOr(body, "id", strScalar("")))
	data := map[string]any{
		"focus_id":                 focusID,
		"name":                     focusID,
		"name_en":                  focusID,
		"icon":                     "GFX_goal_unknown",
		"dynamic_icon":             []map[string]any{},
		"text_icon":                "",
		"offset":                   []map[string]any{},
		"cost":                     cost,
		"x":                        x,
		"y":                        y,
		"relative_position_id":     pyStr(scalarOr(body, "relative_position_id", strScalar(""))),
		"prerequisites":            [][]string{},
		"mutually_exclusive":       []string{},
		"completion_reward":        "",
		"available":                "",
		"bypass":                   "",
		"cancel":                   "",
		"select_effect":            "",
		"ai_will_do":               "",
		"search_filters":           []string{},
		"cancel_if_invalid":        true,
		"continue_if_invalid":      false,
		"available_if_capitulated": false,
		"will_lead_to_war_with":    []string{},
		"allow_branch_ast":         map[string]any{},
		"load_focus_tree_rules":    []map[string]any{},
		"raw_extras":               NewRawDict(),
	}
	rawExtras := data["raw_extras"].(*RawDict)

	iconEntry := findOne(body, "icon")
	if iconEntry != nil && iconEntry.Block != nil {
		dyn := data["dynamic_icon"].([]map[string]any)
		for _, ie := range findAll(body, "icon") {
			if ie.Block == nil {
				continue
			}
			inner := ie.Block.Entries
			oldValue := scalarOr(inner, "value", strScalar(""))
			if pyTruthy(oldValue) {

				gfx := pyStr(oldValue)
				trigText := ""
				if trigBlk := ppBlock(inner, "trigger"); truthyBlock(trigBlk) {
					trigText = importsvc.BlockToText(trigBlk, 0)
				}
				dyn = append(dyn, map[string]any{"icon": gfx, "trigger": trigText})
				if trigText == "" {
					data["icon"] = gfx
				}
			} else {

				for _, child := range inner {
					if child.Key == nil || *child.Key == "" {
						continue
					}
					gfx := *child.Key
					if child.Block != nil {
						trigText := importsvc.BlockToText(child.Block, 0)
						dyn = append(dyn, map[string]any{"icon": gfx, "trigger": trigText})
					} else {
						dyn = append(dyn, map[string]any{"icon": gfx, "trigger": ""})
						if pyLower(pyStrip(pyStr(child.Scalar))) == "yes" {
							data["icon"] = gfx
						}
					}
				}
			}
		}
		data["dynamic_icon"] = dyn
	} else {
		data["icon"] = pyStr(scalarOr(body, "icon", strScalar("GFX_goal_unknown")))
	}

	data["text_icon"] = pyStr(scalarOr(body, "text_icon", strScalar("")))

	offsets := data["offset"].([]map[string]any)
	for _, oe := range findAll(body, "offset") {
		if oe.Block == nil {
			continue
		}
		inner := oe.Block.Entries
		ox, err := resolveVar(scalarOr(inner, "x", intScalar(0)), vars)
		if err != nil {
			return nil, err
		}
		oy, err := resolveVar(scalarOr(inner, "y", intScalar(0)), vars)
		if err != nil {
			return nil, err
		}
		trigText := ""
		if trigBlk := ppBlock(inner, "trigger"); truthyBlock(trigBlk) {
			trigText = importsvc.BlockToText(trigBlk, 0)
		}
		offsets = append(offsets, map[string]any{"x": ox, "y": oy, "trigger": trigText})
	}
	data["offset"] = offsets

	prereqs := data["prerequisites"].([][]string)
	for _, pe := range findAll(body, "prerequisite") {
		if pe.Block == nil {
			continue
		}
		focusIDs := []string{}
		for _, e := range pe.Block.Entries {
			if e.Key != nil && *e.Key == "focus" && e.Block == nil {
				focusIDs = append(focusIDs, pyStr(e.Scalar))
			}
		}
		if len(focusIDs) > 0 {
			prereqs = append(prereqs, focusIDs)
		}
	}
	data["prerequisites"] = prereqs

	if meBlock := ppBlock(body, "mutually_exclusive"); truthyBlock(meBlock) {
		me := []string{}
		for _, e := range meBlock.Entries {
			if e.Key != nil && *e.Key == "focus" && e.Block == nil {
				me = append(me, pyStr(e.Scalar))
			}
		}
		data["mutually_exclusive"] = me
	}

	for _, fieldName := range []string{
		"completion_reward", "available", "bypass", "cancel", "select_effect",
	} {
		if blk := ppBlock(body, fieldName); truthyBlock(blk) {
			data[fieldName] = importsvc.BlockToText(blk, 0)
		}
	}

	if crBlock := ppBlock(body, "completion_reward"); truthyBlock(crBlock) {
		rules := extractLoadFocusTreeRules(crBlock.Entries)
		if len(rules) > 0 {
			data["load_focus_tree_rules"] = rules
		}
	}

	if aiBlock := ppBlock(body, "ai_will_do"); truthyBlock(aiBlock) {
		data["ai_will_do"] = importsvc.BlockToText(aiBlock, 0)
	}

	if sfBlock := ppBlock(body, "search_filters"); truthyBlock(sfBlock) {
		data["search_filters"] = bareValues(sfBlock.Entries)
	}

	if abBlock := ppBlock(body, "allow_branch"); truthyBlock(abBlock) {
		data["allow_branch_ast"] = map[string]any{
			"children": ParseConditionAST(abBlock.Entries),
		}
	}

	if beBlock := ppBlock(body, "bypass_effect"); truthyBlock(beBlock) {
		rawExtras.Set("bypass_effect", importsvc.BlockToText(beBlock, 0))
	}

	for _, flag := range []string{
		"cancel_if_invalid", "continue_if_invalid", "available_if_capitulated",
	} {
		if raw := scalarOpt(body, flag); raw != nil {
			s := pyLower(pyStrip(pyStr(*raw)))
			data[flag] = s != "no" && s != "false" && s != "0"
		}
	}

	wltww := data["will_lead_to_war_with"].([]string)
	for _, e := range findAll(body, "will_lead_to_war_with") {
		if e.Block != nil {
			continue
		}
		tag := pyUpper(pyStrip(pyStr(e.Scalar)))
		if tag == "" {
			continue
		}
		seen := false
		for _, t := range wltww {
			if t == tag {
				seen = true
				break
			}
		}
		if !seen {
			wltww = append(wltww, tag)
		}
	}
	data["will_lead_to_war_with"] = wltww

	for _, e := range body {
		if e.Key != nil && *e.Key != "" && !focusKnownFields[*e.Key] {
			if e.Block != nil {
				rawExtras.Accumulate(*e.Key, importsvc.BlockToText(e.Block, 0))
			} else {
				rawExtras.Accumulate(*e.Key, e.Scalar)
			}
			summary.AddPreserved(*e.Key)
		}
	}

	return data, nil
}
