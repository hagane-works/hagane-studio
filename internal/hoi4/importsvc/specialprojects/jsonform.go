package specialprojects

import (
	"errors"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func pyExcString(err error) string {
	var pe *paradox.ParseError
	if errors.As(err, &pe) {
		return "ParseError: " + pe.Msg
	}
	var oe *OverflowError
	if errors.As(err, &oe) {
		return "OverflowError: " + oe.Msg
	}
	return "Exception: " + err.Error()
}

func encJSONSummary(s *importsvc.ImportSummary) map[string]any {
	preserved := []any{}
	for _, k := range s.PreservedFields.Keys {
		preserved = append(preserved, []any{k, s.PreservedFields.Get(k)})
	}
	warnings := []any{}
	for _, w := range s.Warnings {
		warnings = append(warnings, w)
	}
	return map[string]any{
		"object_type": s.ObjectType, "total": s.Total,
		"preserved": preserved, "warnings": warnings,
	}
}

func encJSONRawExtras(re *importsvc.RawExtras) []any {
	out := []any{}
	for _, k := range re.Keys {
		v, _ := re.Get(k)
		out = append(out, []any{k, v})
	}
	return out
}

func strListJSON(xs []string) []any {
	out := []any{}
	for _, x := range xs {
		out = append(out, x)
	}
	return out
}

func SpecializationsJSON(text string) map[string]any {
	results, summary, err := ImportSpecializations(text)
	if err != nil {
		return map[string]any{"error": pyExcString(err)}
	}
	rows := []any{}
	for _, r := range results {
		color := []any{}
		for _, c := range r.Color {
			color = append(color, c.String())
		}
		rows = append(rows, map[string]any{
			"specialization_id": r.SpecializationID,
			"name_en":           r.NameEn, "name_zh": r.NameZh,
			"desc_en": r.DescEn, "desc_zh": r.DescZh,
			"color":               color,
			"blueprint_image":     r.BlueprintImage,
			"program_background":  r.ProgramBackground,
			"facility_model":      r.FacilityModel,
			"sub_specializations": strListJSON(r.SubSpecializations),
			"raw_extras":          encJSONRawExtras(r.RawExtras),
		})
	}
	return map[string]any{"results": rows, "summary": encJSONSummary(summary)}
}

func SpecialProjectsJSON(text string) map[string]any {
	results, summary, err := ImportSpecialProjects(text)
	if err != nil {
		return map[string]any{"error": pyExcString(err)}
	}
	rows := []any{}
	for _, r := range results {
		rows = append(rows, map[string]any{
			"project_token":  r.ProjectToken,
			"specialization": r.Specialization,
			"name_en":        r.NameEn, "name_zh": r.NameZh,
			"desc_en": r.DescEn, "desc_zh": r.DescZh,
			"icon": r.Icon, "blueprint_image": r.BlueprintImage,
			"complexity": r.Complexity, "prototype_time": r.PrototypeTime,
			"project_tags":              strListJSON(r.ProjectTags),
			"special_project_parent":    strListJSON(r.SpecialProjectParent),
			"allowed":                   r.Allowed,
			"visible":                   r.Visible,
			"available":                 r.Available,
			"resource_cost":             r.ResourceCost,
			"project_output":            r.ProjectOutput,
			"generic_prototype_rewards": strListJSON(r.GenericPrototypeRewards),
			"raw_extras":                encJSONRawExtras(r.RawExtras),
		})
	}
	return map[string]any{"results": rows, "summary": encJSONSummary(summary)}
}
