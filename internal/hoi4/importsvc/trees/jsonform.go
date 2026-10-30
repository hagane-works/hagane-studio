package trees

import (
	"fmt"
	"math/big"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func encJSON(v any) map[string]any {
	switch t := v.(type) {
	case *big.Int:
		return map[string]any{"t": "int", "v": t.String()}
	case float64:
		return map[string]any{"t": "float", "v": importsvc.PyFloatReprStr(t)}
	case string:
		return map[string]any{"t": "str", "v": t}
	case bool:
		return map[string]any{"t": "bool", "v": t}
	case paradox.Scalar:
		switch t.Kind {
		case paradox.ScalarInt:
			return map[string]any{"t": "int", "v": t.Int.String()}
		case paradox.ScalarFloat:
			return map[string]any{"t": "float", "v": importsvc.PyFloatReprStr(t.Float)}
		default:
			return map[string]any{"t": "str", "v": t.Str}
		}
	}
	panic(fmt.Sprintf("trees jsonform enc: unexpected type %T", v))
}

func encJSONExtras(d *RawDict) map[string]any {
	out := map[string]any{}
	for _, k := range d.Keys {
		v, _ := d.Get(k)
		if lst, ok := v.([]any); ok {
			arr := make([]any, 0, len(lst))
			for _, it := range lst {
				arr = append(arr, encJSON(it))
			}
			out[k] = map[string]any{"t": "list", "v": arr}
		} else {
			out[k] = encJSON(v)
		}
	}
	return out
}

func summaryJSON(s *importsvc.ImportSummary) map[string]any {
	preserved := map[string]any{}
	for _, k := range s.PreservedFields.Keys {
		preserved[k] = s.PreservedFields.Get(k)
	}
	return map[string]any{
		"object_type": s.ObjectType,
		"total":       s.Total,
		"preserved":   preserved,
		"warnings":    s.Warnings,
	}
}

func nodeJSON(n map[string]any) map[string]any {
	offsets := make([]any, 0)
	for _, o := range n["offset"].([]map[string]any) {
		offsets = append(offsets, map[string]any{
			"x": encJSON(o["x"]), "y": encJSON(o["y"]), "trigger": o["trigger"],
		})
	}
	return map[string]any{
		"focus_id":                 n["focus_id"],
		"name":                     n["name"],
		"name_en":                  n["name_en"],
		"icon":                     n["icon"],
		"dynamic_icon":             n["dynamic_icon"],
		"text_icon":                n["text_icon"],
		"offset":                   offsets,
		"cost":                     encJSON(n["cost"]),
		"x":                        encJSON(n["x"]),
		"y":                        encJSON(n["y"]),
		"relative_position_id":     n["relative_position_id"],
		"prerequisites":            n["prerequisites"],
		"mutually_exclusive":       n["mutually_exclusive"],
		"completion_reward":        n["completion_reward"],
		"available":                n["available"],
		"bypass":                   n["bypass"],
		"cancel":                   n["cancel"],
		"select_effect":            n["select_effect"],
		"ai_will_do":               n["ai_will_do"],
		"search_filters":           n["search_filters"],
		"cancel_if_invalid":        n["cancel_if_invalid"],
		"continue_if_invalid":      n["continue_if_invalid"],
		"available_if_capitulated": n["available_if_capitulated"],
		"will_lead_to_war_with":    n["will_lead_to_war_with"],
		"allow_branch_ast":         n["allow_branch_ast"],
		"load_focus_tree_rules":    n["load_focus_tree_rules"],
		"raw_extras":               encJSONExtras(n["raw_extras"].(*RawDict)),
	}
}

func treeJSON(td map[string]any) map[string]any {
	var verdictAny any
	if sv, ok := td["_scope_verdict"]; ok {
		v := sv.(importsvc.ScopeVerdict)
		verdictAny = map[string]any{"kind": string(v.Kind), "tags": v.TagList()}
	}
	var hintAny any
	if h, ok := td["_country_tags_hint"]; ok {
		hintAny = h
	}
	return map[string]any{
		"tree_id":    td["tree_id"],
		"is_default": td["is_default"],
		"cfp": []any{
			encJSON(td["continuous_focus_position_x"]),
			encJSON(td["continuous_focus_position_y"]),
		},
		"isp":        initialShowPositionJSON(td["initial_show_position"]),
		"isf":        td["initial_show_focus"],
		"raw_extras": encJSONExtras(td["raw_extras"].(*RawDict)),
		"verdict":    verdictAny,
		"hint":       hintAny,
	}
}

func initialShowPositionJSON(value any) map[string]any {
	position, ok := value.(map[string]any)
	if !ok || len(position) == 0 {
		return map[string]any{}
	}
	kind, _ := position["kind"].(string)
	if kind == "focus" {
		return map[string]any{
			"kind": "focus", "focus_id": position["focus_id"],
		}
	}
	if kind == "invalid" {
		return map[string]any{
			"kind": "invalid", "raw": position["raw"],
			"raw_text": position["raw_text"],
		}
	}
	if kind == "legacy_ambiguous" {
		return map[string]any{
			"kind": "legacy_ambiguous", "raw": position["raw"],
		}
	}
	if kind != "coordinates" {
		return map[string]any{}
	}
	outOffsets := []any{}
	if offsets, ok := position["offsets"].([]map[string]any); ok {
		for _, offset := range offsets {
			outOffsets = append(outOffsets, map[string]any{
				"x": encJSON(offset["x"]), "y": encJSON(offset["y"]),
				"trigger": offset["trigger"],
			})
		}
	}
	return map[string]any{
		"kind": "coordinates",
		"x":    encJSON(position["x"]), "y": encJSON(position["y"]),
		"offsets": outOffsets,
	}
}

func bundleJSON(b *importsvc.TreeBundle) map[string]any {
	nodes := make([]any, 0, len(b.NodesData))
	for _, n := range b.NodesData {
		nodes = append(nodes, nodeJSON(n))
	}
	refs := make([]any, 0, len(b.SharedFocusRefs))
	for _, r := range b.SharedFocusRefs {
		refs = append(refs, map[string]any{
			"focus_id": r["focus_id"],
			"ox":       encJSON(r["offset_x"]),
			"oy":       encJSON(r["offset_y"]),
		})
	}
	return map[string]any{
		"tree_kind": b.TreeKind,
		"tree":      treeJSON(b.TreeData),
		"nodes":     nodes,
		"refs":      refs,
	}
}

func FocusFileJSON(text, sourceFilename string) map[string]any {
	bundles, summary, err := ImportFocusFile(text, sourceFilename)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	bs := make([]any, 0, len(bundles))
	for _, b := range bundles {
		bs = append(bs, bundleJSON(b))
	}
	return map[string]any{"bundles": bs, "summary": summaryJSON(summary)}
}

func FocusTreeLegacyJSON(text string) map[string]any {
	td, nodes, s, err := ImportFocusTree(text)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	var treeID any
	if v, ok := td["tree_id"]; ok {
		treeID = v
	}
	return map[string]any{
		"tree_id":  treeID,
		"n":        len(nodes),
		"total":    s.Total,
		"warnings": s.Warnings,
	}
}

func techFullJSON(t map[string]any) map[string]any {
	deps := map[string]any{}
	for k, v := range t["dependencies"].(map[string]any) {
		deps[k] = encJSON(v)
	}
	xpb := map[string]any{}
	for k, v := range t["xp_research_bonus"].(map[string]any) {
		xpb[k] = encJSON(v)
	}
	mods := map[string]any{}
	for k, v := range t["modifiers"].(map[string]any) {
		if sub, ok := v.(map[string]any); ok {
			inner := map[string]any{}
			for k2, v2 := range sub {
				inner[k2] = encJSON(v2)
			}
			mods[k] = map[string]any{"t": "dict", "v": inner}
		} else {
			mods[k] = encJSON(v)
		}
	}
	paths := make([]any, 0)
	for _, p := range t["path"].([]map[string]any) {
		paths = append(paths, map[string]any{
			"leads_to_tech": p["leads_to_tech"],
			"coeff":         encJSON(p["research_cost_coeff"]),
		})
	}
	var placements []any
	if source, ok := t["folder_placements"].([]map[string]any); ok {
		placements = make([]any, 0, len(source))
		for _, placement := range source {
			placements = append(placements, map[string]any{
				"folder": placement["folder"],
				"x":      encJSON(placement["x"]),
				"y":      encJSON(placement["y"]),
			})
		}
	}
	return map[string]any{
		"tech_id":                     t["tech_id"],
		"name":                        t["name"],
		"name_en":                     t["name_en"],
		"tech_line":                   t["tech_line"],
		"research_cost":               encJSON(t["research_cost"]),
		"start_year":                  encJSON(t["start_year"]),
		"folder":                      t["folder"],
		"position_x":                  encJSON(t["position_x"]),
		"position_y":                  encJSON(t["position_y"]),
		"position_y_is_explicit":      t["position_y_is_explicit"],
		"folder_placements":           placements,
		"prerequisites":               t["prerequisites"],
		"categories":                  t["categories"],
		"modifiers":                   mods,
		"ai_will_do":                  t["ai_will_do"],
		"raw_extras":                  encJSONExtras(t["raw_extras"].(*RawDict)),
		"enable_equipments":           t["enable_equipments"],
		"enable_subunits":             t["enable_subunits"],
		"enable_equipment_modules":    t["enable_equipment_modules"],
		"enable_building":             t["enable_building"],
		"enable_tactic":               t["enable_tactic"],
		"xor":                         t["xor"],
		"dependencies":                deps,
		"sub_technologies":            t["sub_technologies"],
		"doctrine":                    t["doctrine"],
		"allow":                       t["allow"],
		"allow_branch":                t["allow_branch"],
		"on_research_complete":        t["on_research_complete"],
		"on_research_complete_limit":  t["on_research_complete_limit"],
		"path":                        paths,
		"xp_research_type":            t["xp_research_type"],
		"xp_boost_cost":               encJSON(t["xp_boost_cost"]),
		"xp_research_bonus":           xpb,
		"force_use_small_tech_layout": t["force_use_small_tech_layout"],
	}
}

func TechnologiesJSON(text, techLine, folderName string) map[string]any {
	techs, years, s, err := ImportTechnologies(text, techLine, folderName)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	ts := make([]any, 0, len(techs))
	for _, t := range techs {
		ts = append(ts, techFullJSON(t))
	}
	ys := make([]any, 0, len(years))
	for _, y := range years {
		ys = append(ys, []any{y.Name, encJSON(y.Row)})
	}
	return map[string]any{"techs": ts, "years": ys, "summary": summaryJSON(s)}
}
