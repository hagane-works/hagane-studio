package units

import (
	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func encJSONScalar(s paradox.Scalar) map[string]any {
	switch s.Kind {
	case paradox.ScalarInt:
		return map[string]any{"t": "int", "v": s.Int.String()}
	case paradox.ScalarFloat:
		return map[string]any{"t": "float", "v": importsvc.PyFloatReprStr(s.Float)}
	default:
		return map[string]any{"t": "str", "v": s.Str}
	}
}

func optStrJSON(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func encJSONEntryData(d *EntryData) map[string]any {
	var value any
	if d.IsBlock {
		children := []any{}
		for _, c := range d.Children {
			children = append(children, encJSONEntryData(c))
		}
		value = children
	} else {
		value = encJSONScalar(d.Value)
	}
	return map[string]any{
		"key": optStrJSON(d.Key), "op": optStrJSON(d.Op), "value": value,
		"line": d.Line, "quoted": d.Quoted, "qualifier": optStrJSON(d.Qualifier),
	}
}

func encJSONBodyAST(body []*EntryData) []any {
	out := []any{}
	for _, d := range body {
		out = append(out, encJSONEntryData(d))
	}
	return out
}

func encJSONLayout(items []*LayoutItem) []any {
	out := []any{}
	for _, it := range items {
		switch it.Kind {
		case LayoutRaw:
			out = append(out, map[string]any{"kind": it.Kind, "entry": encJSONEntryData(it.Entry)})
		case LayoutField:
			out = append(out, map[string]any{"kind": it.Kind, "field": it.Field})
		default:
			out = append(out, map[string]any{"kind": it.Kind, "ordinal": it.Ordinal})
		}
	}
	return out
}

func encJSONWing(w *AirWing) map[string]any {
	return map[string]any{
		"equipment_type": w.EquipmentType, "name": w.Name, "owner": w.Owner,
		"amount": w.Amount, "ordering": w.Ordering,
		"body_ast": encJSONBodyAST(w.BodyAST),
	}
}

func encJSONGroup(g *AirWingGroup) map[string]any {
	wings := []any{}
	for _, w := range g.Wings {
		wings = append(wings, encJSONWing(w))
	}
	return map[string]any{
		"location": g.Location, "ordering": g.Ordering,
		"layout": encJSONLayout(g.Layout), "wings": wings,
	}
}

func encJSONShip(s *Ship) map[string]any {
	return map[string]any{
		"name": s.Name, "definition": s.Definition,
		"start_experience_factor": s.StartExperienceFactor,
		"equipment_type":          s.EquipmentType, "amount": s.Amount,
		"owner": s.Owner, "version_name": s.VersionName,
		"ordering": s.Ordering, "body_ast": encJSONBodyAST(s.BodyAST),
	}
}

func encJSONTaskForce(tf *TaskForce) map[string]any {
	ships := []any{}
	for _, s := range tf.Ships {
		ships = append(ships, encJSONShip(s))
	}
	return map[string]any{
		"name": tf.Name, "location": tf.Location, "ordering": tf.Ordering,
		"layout": encJSONLayout(tf.Layout), "ships": ships,
	}
}

func encJSONFleet(f *Fleet) map[string]any {
	tfs := []any{}
	for _, tf := range f.TaskForces {
		tfs = append(tfs, encJSONTaskForce(tf))
	}
	return map[string]any{
		"name": f.Name, "naval_base": f.NavalBase, "ordering": f.Ordering,
		"layout": encJSONLayout(f.Layout), "task_forces": tfs,
	}
}

func AirNavalOOBJSON(text string) map[string]any {
	parsed, warnings := ImportAirNavalOOB(text)
	groups := []any{}
	for _, g := range parsed.AirWingGroups {
		groups = append(groups, encJSONGroup(g))
	}
	fleets := []any{}
	for _, f := range parsed.Fleets {
		fleets = append(fleets, encJSONFleet(f))
	}
	roundtrip := []any{}
	for _, g := range parsed.AirWingGroups {
		for _, w := range g.Wings {
			roundtrip = append(roundtrip,
				importsvc.ToScriptEntries(DataToEntries(w.BodyAST), 0))
		}
	}
	for _, f := range parsed.Fleets {
		for _, tf := range f.TaskForces {
			for _, s := range tf.Ships {
				roundtrip = append(roundtrip,
					importsvc.ToScriptEntries(DataToEntries(s.BodyAST), 0))
			}
		}
	}
	return map[string]any{
		"groups": groups, "fleets": fleets, "warnings": warnings,
		"roundtrip": roundtrip,
	}
}
