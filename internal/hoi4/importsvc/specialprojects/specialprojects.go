package specialprojects

import (
	"math"
	"math/big"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func pyIsSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return true
	}
	return unicode.IsSpace(r)
}

func pyStrip(s string) string { return strings.TrimFunc(s, pyIsSpace) }

func stripQuotes(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return r == '"' })
}

type OverflowError struct{ Msg string }

func (e *OverflowError) Error() string { return e.Msg }

func entryKeyStr(e *paradox.Entry) string {
	if e.Key == nil {
		return "None"
	}
	return *e.Key
}

func bareStripped(entries []*paradox.Entry) []string {
	out := []string{}
	for _, c := range entries {
		if c.Key == nil && c.Block == nil {
			out = append(out, pyStrip(importsvc.PyScalarStr(c.Scalar)))
		}
	}
	return out
}

func expandAtConstants(entries []*paradox.Entry) {
	registry := map[string]paradox.Scalar{}
	for _, e := range entries {
		if e.Block != nil {
			continue
		}
		if e.Key == nil || *e.Key == "" || !strings.HasPrefix(*e.Key, "@") {
			continue
		}
		registry[*e.Key] = e.Scalar
	}
	if len(registry) == 0 {
		return
	}
	var walk func(es []*paradox.Entry)
	walk = func(es []*paradox.Entry) {
		for _, e := range es {
			if e.Block != nil {
				walk(e.Block.Entries)
				continue
			}
			if e.Scalar.Kind == paradox.ScalarStr && strings.HasPrefix(e.Scalar.Str, "@") {
				if v, ok := registry[e.Scalar.Str]; ok {
					e.Scalar = v
				}
			}
		}
	}
	walk(entries)
}

func colorInts(nums []paradox.Scalar) ([]*big.Int, error) {
	out := make([]*big.Int, 0, len(nums))
	for _, n := range nums {
		s := paradox.Scalar{Kind: paradox.ScalarStr, Str: importsvc.PyScalarStr(n)}
		f, _ := importsvc.CoerceFloat(&s, math.NaN())
		if math.IsNaN(f) {

			return []*big.Int{}, nil
		}
		if math.IsInf(f, 0) {
			return nil, &OverflowError{Msg: "cannot convert float infinity to integer"}
		}
		z, _ := new(big.Float).SetFloat64(f).Int(nil)
		out = append(out, z)
	}
	if len(out) > 3 {
		out = out[:3]
	}
	return out, nil
}

type Specialization struct {
	SpecializationID   string
	NameEn             string
	NameZh             string
	DescEn             string
	DescZh             string
	Color              []*big.Int
	BlueprintImage     string
	ProgramBackground  string
	FacilityModel      string
	SubSpecializations []string
	RawExtras          *importsvc.RawExtras
}

func ImportSpecializations(text string) ([]*Specialization, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("special_project_specialization")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, nil, err
	}
	expandAtConstants(ast.Entries)
	results := []*Specialization{}
	for _, entry := range ast.Entries {
		if entry.Key == nil || *entry.Key == "" || entry.Block == nil {
			continue
		}
		spec, err := mapSpecialization(*entry.Key, entry.Block.Entries, summary)
		if err != nil {
			return nil, nil, err
		}
		results = append(results, spec)
	}
	summary.Total = len(results)
	return results, summary, nil
}

func mapSpecialization(specID string, block []*paradox.Entry,
	summary *importsvc.ImportSummary) (*Specialization, error) {
	result := &Specialization{
		SpecializationID:   specID,
		Color:              []*big.Int{},
		SubSpecializations: []string{},
		RawExtras:          importsvc.NewRawExtras(),
	}
	for _, entry := range block {
		key := entryKeyStr(entry)
		isBlock := entry.Block != nil
		switch {
		case key == "color" && isBlock:
			var nums []paradox.Scalar
			for _, c := range entry.Block.Entries {
				if c.Key == nil && c.Block == nil {
					nums = append(nums, c.Scalar)
				}
			}
			color, err := colorInts(nums)
			if err != nil {
				return nil, err
			}
			result.Color = color
		case (key == "blueprint_image" || key == "program_background" ||
			key == "facility_model") && !isBlock:

			v := stripQuotes(pyStrip(importsvc.PyScalarStr(entry.Scalar)))
			switch key {
			case "blueprint_image":
				result.BlueprintImage = v
			case "program_background":
				result.ProgramBackground = v
			case "facility_model":
				result.FacilityModel = v
			}
		case key == "special_project_sub_specializations" && isBlock:
			result.SubSpecializations = bareStripped(entry.Block.Entries)
		case !isBlock:
			importsvc.AccumulateRawExtra(result.RawExtras, key,
				pyStrip(importsvc.PyScalarStr(entry.Scalar)))
			summary.AddPreserved(key)
		default:
			importsvc.AccumulateRawExtra(result.RawExtras, key,
				importsvc.BlockToText(entry.Block, 0))
			summary.AddPreserved(key)
		}
	}
	return result, nil
}

var spProjectScalarKeys = map[string]bool{
	"specialization": true, "icon": true, "blueprint_image": true,
	"complexity": true, "prototype_time": true,
}

var spProjectRawBlockKeys = map[string]bool{
	"allowed": true, "visible": true, "available": true,
	"resource_cost": true, "project_output": true,
}

type SpecialProject struct {
	ProjectToken            string
	Specialization          string
	NameEn                  string
	NameZh                  string
	DescEn                  string
	DescZh                  string
	Icon                    string
	BlueprintImage          string
	Complexity              string
	PrototypeTime           string
	ProjectTags             []string
	SpecialProjectParent    []string
	Allowed                 string
	Visible                 string
	Available               string
	ResourceCost            string
	ProjectOutput           string
	GenericPrototypeRewards []string
	RawExtras               *importsvc.RawExtras
}

func ImportSpecialProjects(text string) ([]*SpecialProject, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("special_project")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, nil, err
	}
	expandAtConstants(ast.Entries)
	results := []*SpecialProject{}
	for _, entry := range ast.Entries {
		if entry.Key == nil || *entry.Key == "" || entry.Block == nil {
			continue
		}
		results = append(results,
			mapSpecialProject(*entry.Key, entry.Block.Entries, summary))
	}
	summary.Total = len(results)
	return results, summary, nil
}

func mapSpecialProject(token string, block []*paradox.Entry,
	summary *importsvc.ImportSummary) *SpecialProject {
	result := &SpecialProject{
		ProjectToken:            token,
		ProjectTags:             []string{},
		SpecialProjectParent:    []string{},
		GenericPrototypeRewards: []string{},
		RawExtras:               importsvc.NewRawExtras(),
	}
	for _, entry := range block {
		key := entryKeyStr(entry)
		isBlock := entry.Block != nil
		switch {
		case spProjectScalarKeys[key] && !isBlock:
			v := pyStrip(importsvc.PyScalarStr(entry.Scalar))
			switch key {
			case "specialization":
				result.Specialization = v
			case "icon":
				result.Icon = v
			case "blueprint_image":
				result.BlueprintImage = v
			case "complexity":
				result.Complexity = v
			case "prototype_time":
				result.PrototypeTime = v
			}
		case key == "project_tags":

			if isBlock {
				result.ProjectTags = bareStripped(entry.Block.Entries)
			} else {
				result.ProjectTags = []string{pyStrip(importsvc.PyScalarStr(entry.Scalar))}
			}
		case key == "special_project_parent" && isBlock:
			result.SpecialProjectParent = bareStripped(entry.Block.Entries)
		case key == "generic_prototype_rewards" && isBlock:
			result.GenericPrototypeRewards = bareStripped(entry.Block.Entries)
		case spProjectRawBlockKeys[key] && isBlock:

			v := importsvc.ToScript(entry.Block, 0)
			switch key {
			case "allowed":
				result.Allowed = v
			case "visible":
				result.Visible = v
			case "available":
				result.Available = v
			case "resource_cost":
				result.ResourceCost = v
			case "project_output":
				result.ProjectOutput = v
			}
		case !isBlock:
			importsvc.AccumulateRawExtra(result.RawExtras, key,
				pyStrip(importsvc.PyScalarStr(entry.Scalar)))
			summary.AddPreserved(key)
		default:

			importsvc.AccumulateRawExtra(result.RawExtras, key,
				importsvc.BlockToText(entry.Block, 0))
			summary.AddPreserved(key)
		}
	}
	return result
}
