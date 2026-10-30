package preview

import (
	"encoding/json"
	"math/big"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
)

func AttrStr(ent *OD, k string) string   { return getStr(ent, k) }
func DeriveCountryTag(tag string) string { return deriveCountryTag(tag) }

func BuildCharacter(d map[string]any) *OD { return buildEnt(characterFields, mapSrc(d)) }
func BuildEvent(d map[string]any) *OD     { return buildEnt(eventFields, mapSrc(d)) }
func BuildTech(d map[string]any) *OD      { return buildEnt(techFields, mapSrc(d)) }

func BuildIdea(d interface {
	Get(string) (any, bool)
}) *OD {
	return buildEnt(ideaFields, func(k string) (any, bool) { return d.Get(k) })
}

func BuildFocusTree(b *importsvc.TreeBundle) *FocusTree {
	src := mapSrc(b.TreeData)
	ent := buildEnt(focusTreeFields, src)
	applyVerdict(ent, src)
	ent.Set("tree_kind", b.TreeKind)
	refs := make([]any, 0, len(b.SharedFocusRefs))
	for _, r := range b.SharedFocusRefs {
		refs = append(refs, normalize(r))
	}
	ent.Set("shared_focus_refs", refs)
	var nodes []*OD
	for ni, nd := range b.NodesData {
		n := buildEnt(focusNodeFields, mapSrc(nd))
		n.Set("id", big.NewInt(int64(ni+1)))
		nodes = append(nodes, n)
	}
	return &FocusTree{Ent: ent, Nodes: nodes}
}

func EventOutputs(ns0 string, ents []*OD) []map[string]any {
	gatedAll := map[string]bool{}
	for _, e := range ents {
		if pyTruthy(attr(e, "picture")) {
			gatedAll[pyStrip(pyStr(attr(e, "picture")))] = true
		}
	}
	return []map[string]any{
		{"fn": "generate_events_preview", "args": map[string]any{"variant": "default"},
			"text": GenerateEventsPreview(ns0, ents, nil, false)},
		{"fn": "generate_events_preview", "args": map[string]any{"variant": "gated_empty"},
			"text": GenerateEventsPreview(ns0, ents, map[string]bool{}, true)},
		{"fn": "generate_events_preview", "args": map[string]any{"variant": "gated_all"},
			"text": GenerateEventsPreview(ns0, ents, gatedAll, true)},
	}
}

func DecisionOutputs(catDicts, decDicts []map[string]any) []map[string]any {
	var cats []*DecisionCategory
	pkByName := map[string]int{}
	for i, cd := range catDicts {
		src := mapSrc(cd)
		ent := buildEnt(decisionCategoryFields, src)
		applyVerdict(ent, src)
		cats = append(cats, &DecisionCategory{Ent: ent, ID: i + 1})
		name, _ := cd["category_id"].(string)
		if _, ok := pkByName[name]; !ok {
			pkByName[name] = i + 1
		}
	}
	var decs []*Decision
	for j, dd := range decDicts {
		src := mapSrc(dd)
		ent := buildEnt(decisionFields, src)
		applyVerdict(ent, src)
		name, _ := dd["category_id"].(string)
		decs = append(decs, &Decision{Ent: ent, ID: j + 1, CategoryID: pkByName[name]})
	}
	return decisionVariantOutputs(cats, decs)
}

func decisionVariantOutputs(cats []*DecisionCategory, decs []*Decision) []map[string]any {
	catGatesOff := map[int]bool{}
	for _, c := range cats {
		catGatesOff[c.ID] = false
	}
	decGatesOff := map[int]bool{}
	for _, d := range decs {
		decGatesOff[d.ID] = false
	}
	return []map[string]any{
		{"fn": "generate_decision_categories_script",
			"args": map[string]any{"variant": "default"},
			"text": GenerateDecisionCategoriesScript(cats, nil)},
		{"fn": "generate_decision_categories_script",
			"args": map[string]any{"variant": "gated_off"},
			"text": GenerateDecisionCategoriesScript(cats, catGatesOff)},
		{"fn": "generate_decisions_preview",
			"args": map[string]any{"variant": "default"},
			"text": GenerateDecisionsPreview(cats, decs, nil)},
		{"fn": "generate_decisions_preview",
			"args": map[string]any{"variant": "gated_off"},
			"text": GenerateDecisionsPreview(cats, decs, decGatesOff)},
	}
}

func FocusTreeOutputs(tree *FocusTree, bundleIdx int) []map[string]any {
	validIconsAll := map[string]bool{}
	for _, n := range tree.Nodes {
		if icon := getStr(n, "icon"); pyTruthy(attr(n, "icon")) {
			validIconsAll[icon] = true
		}
	}
	treeID := getStr(tree.Ent, "tree_id")
	outputs := []map[string]any{
		{"fn": "generate_focus_tree_script",
			"args": map[string]any{"tree": treeID, "bundle": bundleIdx, "variant": "default"},
			"text": GenerateFocusTreeScript(tree, nil)},
		{"fn": "generate_focus_tree_script",
			"args": map[string]any{"tree": treeID, "bundle": bundleIdx, "variant": "gated_empty"},
			"text": GenerateFocusTreeScript(tree, map[string]bool{})},
		{"fn": "generate_focus_tree_script",
			"args": map[string]any{"tree": treeID, "bundle": bundleIdx, "variant": "gated_icons"},
			"text": GenerateFocusTreeScript(tree, validIconsAll)},
	}
	for i, n := range tree.Nodes {
		if i >= 2 {
			break
		}
		outputs = append(outputs, map[string]any{
			"fn": "generate_focus_node_script",
			"args": map[string]any{"tree": treeID, "bundle": bundleIdx,
				"focus": getStr(n, "focus_id"), "variant": "default"},
			"text": GenerateFocusNodeScript(n),
		})
	}
	return outputs
}

func TechOutputs(techs []*OD, techLine string, project *Project, siblings []*OD) []map[string]any {
	outputs := []map[string]any{
		{"fn": "generate_tech_line_script",
			"args": map[string]any{"tech_line": techLine, "variant": "default"},
			"text": GenerateTechLineScript(techs, techLine, nil, siblings)},
		{"fn": "generate_tech_line_script",
			"args": map[string]any{"tech_line": techLine, "variant": "project"},
			"text": GenerateTechLineScript(techs, techLine, project, siblings)},
	}
	for i, t := range techs {
		if i >= 2 {
			break
		}
		outputs = append(outputs, map[string]any{
			"fn":   "generate_single_tech_script",
			"args": map[string]any{"tech": getStr(t, "tech_id"), "variant": "project"},
			"text": GenerateSingleTechScript(t, techs, project),
		})
	}
	return outputs
}

func IdeaOutputs(ideas []*Idea) []map[string]any {
	gatedRefs := map[int]string{}
	for _, e := range ideas {
		if pyTruthy(attr(e.Ent, "picture")) {
			gatedRefs[e.ID] = "spr_" + getStr(e.Ent, "idea_id")
		}
	}
	outputs := []map[string]any{
		{"fn": "generate_ideas_script", "args": map[string]any{"variant": "default"},
			"text": GenerateIdeasScript(ideas, nil)},
		{"fn": "generate_ideas_script", "args": map[string]any{"variant": "gated_empty"},
			"text": GenerateIdeasScript(ideas, map[int]string{})},
		{"fn": "generate_ideas_script", "args": map[string]any{"variant": "gated_refs"},
			"text": GenerateIdeasScript(ideas, gatedRefs)},
	}
	for i, e := range ideas {
		if i >= 2 {
			break
		}
		outputs = append(outputs, map[string]any{
			"fn":   "generate_single_idea_script",
			"args": map[string]any{"idea": getStr(e.Ent, "idea_id"), "variant": "default"},
			"text": GenerateSingleIdeaScript(e),
		})
	}
	return outputs
}

func canonJSON(v any) any {
	switch t := v.(type) {
	case nil, string, bool:
		return t
	case json.Number:
		s := t.String()
		if strings.ContainsAny(s, ".eE") {
			f, _ := pyFloatParse(s)
			return f
		}
		z, ok := new(big.Int).SetString(s, 10)
		if !ok {
			f, _ := pyFloatParse(s)
			return f
		}
		return z
	case []any:
		outList := make([]any, 0, len(t))
		for _, x := range t {
			outList = append(outList, canonJSON(x))
		}
		return outList
	case map[string]any:
		od := NewOD()
		for k, val := range t {
			od.Set(k, canonJSON(val))
		}
		return od
	}
	panic("canonJSON: unexpected type")
}

func odSrc(od *OD) srcGet {
	return func(k string) (any, bool) { return od.Get(k) }
}

func specOD(spec map[string]any, key string) *OD {
	if v, ok := spec[key]; ok {
		if od, ok2 := canonJSON(v).(*OD); ok2 {
			return od
		}
	}
	return nil
}

func specList(spec map[string]any, key string) []*OD {
	var out []*OD
	if v, ok := spec[key]; ok {
		if lst, ok2 := v.([]any); ok2 {
			for _, item := range lst {
				if od, ok3 := canonJSON(item).(*OD); ok3 {
					out = append(out, od)
				}
			}
		}
	}
	return out
}

func intFromCanon(v any, def int) int {
	if z, ok := v.(*big.Int); ok {
		return int(z.Int64())
	}
	return def
}

func DriveSynthetic(spec map[string]any) []map[string]any {
	kind, _ := spec["kind"].(string)
	switch kind {
	case "character":
		var outputs []map[string]any
		for _, d := range specList(spec, "entities") {
			ent := buildEnt(characterFields, odSrc(d))
			ctag := ""
			if s, ok := spec["country_tag"].(string); ok && s != "" {
				ctag = s
			} else {
				ctag = deriveCountryTag(getStr(ent, "tag"))
			}
			tag := getStr(ent, "tag")
			outputs = append(outputs, map[string]any{
				"fn":   "generate_character_script",
				"args": map[string]any{"tag": tag, "variant": "default"},
				"text": GenerateCharacterScript(ent, ctag, true, true, "", "")})
			outputs = append(outputs, map[string]any{
				"fn":   "generate_character_script",
				"args": map[string]any{"tag": tag, "variant": "nogates"},
				"text": GenerateCharacterScript(ent, ctag, false, false, "", "")})
			lp := getStr(ent, "large_portrait")
			sp := getStr(ent, "small_portrait")
			if lp != "" || sp != "" {
				outputs = append(outputs, map[string]any{
					"fn":   "generate_character_script",
					"args": map[string]any{"tag": tag, "variant": "pathb"},
					"text": GenerateCharacterScript(ent, ctag, true, true, lp, sp)})
			}
		}
		return outputs
	case "events_file":
		var ents []*OD
		for _, d := range specList(spec, "events") {
			ents = append(ents, buildEnt(eventFields, odSrc(d)))
		}
		ns0 := "synth"
		if s, ok := spec["namespace"].(string); ok {
			ns0 = s
		}
		return EventOutputs(ns0, ents)
	case "decisions_file":
		var cats []*DecisionCategory
		for _, cd := range specList(spec, "categories") {
			ent := buildEnt(decisionCategoryFields, odSrc(cd))
			cats = append(cats, &DecisionCategory{
				Ent: ent, ID: intFromCanon(cd.GetDef("id", nil), 0)})
		}
		var decs []*Decision
		for _, dd := range specList(spec, "decisions") {
			ent := buildEnt(decisionFields, odSrc(dd))
			decs = append(decs, &Decision{
				Ent:        ent,
				ID:         intFromCanon(dd.GetDef("id", nil), 0),
				CategoryID: intFromCanon(dd.GetDef("category_pk", nil), 0)})
		}
		return decisionVariantOutputs(cats, decs)
	case "focus_tree":
		treeOD := specOD(spec, "tree")
		ent := buildEnt(focusTreeFields, odSrc(treeOD))
		var nodes []*OD
		for ni, nd := range specList(spec, "nodes") {
			n := buildEnt(focusNodeFields, odSrc(nd))
			n.Set("id", big.NewInt(int64(ni+1)))
			nodes = append(nodes, n)
		}
		tree := &FocusTree{Ent: ent, Nodes: nodes}
		return FocusTreeOutputs(tree, 0)
	case "tech_line":
		var techs []*OD
		for _, d := range specList(spec, "techs") {
			techs = append(techs, buildEnt(techFields, odSrc(d)))
		}
		var siblings []*OD
		if _, ok := spec["siblings"]; ok {
			siblings = append(siblings, techs...)
			for _, d := range specList(spec, "siblings") {
				siblings = append(siblings, buildEnt(techFields, odSrc(d)))
			}
		}
		techLine := "synth"
		if s, ok := spec["tech_line"].(string); ok {
			techLine = s
		}
		var cols []any
		if v, ok := spec["tech_year_columns"]; ok && v != nil {
			if lst, ok2 := canonJSON(v).([]any); ok2 {
				cols = lst
			}
		}
		project := &Project{TechYearColumns: cols}
		return TechOutputs(techs, techLine, project, siblings)
	case "ideas_file":
		var ideas []*Idea
		for i, d := range specList(spec, "ideas") {
			ideas = append(ideas, &Idea{Ent: buildEnt(ideaFields, odSrc(d)), ID: i + 1})
		}
		return IdeaOutputs(ideas)
	}
	panic("unknown synthetic kind " + kind)
}
