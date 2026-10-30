package workbench

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/reftables"
)

type Optional[T any] struct {
	Set   bool
	Value *T
}

func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}
func optionalValue[T any](o Optional[T]) any {
	if o.Value == nil {
		return nil
	}
	return *o.Value
}
func optionalJSON[T any](o Optional[T]) any {
	if o.Value == nil {
		return nil
	}
	b, _ := json.Marshal(*o.Value)
	return string(b)
}

type UpdatePatch struct {
	Name                  Optional[string]            `json:"name"`
	CountryTag            Optional[string]            `json:"country_tag"`
	CountryName           Optional[string]            `json:"country_name"`
	CountryTags           Optional[[]string]          `json:"country_tags"`
	ModPath               Optional[string]            `json:"mod_path"`
	IdeologyIconPaths     Optional[map[string]string] `json:"ideology_icon_paths"`
	Version               Optional[string]            `json:"version"`
	Picture               Optional[string]            `json:"picture"`
	Tags                  Optional[[]string]          `json:"tags"`
	Dependencies          Optional[[]string]          `json:"dependencies"`
	SupportedVersion      Optional[string]            `json:"supported_version"`
	ImportedReplacePaths  Optional[[]string]          `json:"imported_descriptor_replace_paths"`
	Loaded                Optional[bool]              `json:"is_loaded_existing_mod"`
	StartDate             Optional[string]            `json:"start_date"`
	EndDate               Optional[string]            `json:"end_date"`
	ScenarioHistoryPolicy Optional[string]            `json:"scenario_history_policy"`
	StageStateIDs         Optional[[]int64]           `json:"stage_state_ids"`
	ExportLanguages       Optional[[]string]          `json:"export_languages"`
	OwnedDLCs             Optional[[]string]          `json:"owned_dlcs"`
	IdeologyRegistry      Optional[map[string]any]    `json:"ideology_registry"`
	TechSurfaceManifest   Optional[map[string]any]    `json:"tech_surface_manifest"`
}

var dateRE = regexp.MustCompile(`^\d{1,4}\.\d{1,2}\.\d{1,2}(?:\.\d{1,2})?$`)
var languages = map[string]bool{"en": true, "zh": true, "ru": true, "fr": true, "de": true, "es": true, "ja": true, "ko": true, "pl": true, "pt": true}

func (p *UpdatePatch) Validate() error {
	if p.StartDate.Set && p.StartDate.Value != nil {
		v, err := validateStartDate(*p.StartDate.Value)
		if err != nil {
			return err
		}
		p.StartDate.Value = v
	}
	if p.EndDate.Set && p.EndDate.Value != nil {
		v, err := validateEndDate(*p.EndDate.Value)
		if err != nil {
			return err
		}
		p.EndDate.Value = v
	}
	if p.ScenarioHistoryPolicy.Set {
		if p.ScenarioHistoryPolicy.Value == nil {
			return fmt.Errorf("scenario_history_policy must be one of %q or %q", ScenarioHistoryPolicyInheritBaseHistory, ScenarioHistoryPolicyProjectOwnsStartingWorld)
		}
		if err := validateScenarioHistoryPolicy(*p.ScenarioHistoryPolicy.Value); err != nil {
			return err
		}
	}
	if p.StageStateIDs.Set && p.StageStateIDs.Value != nil {
		out, err := normalizeStageStateIDs(*p.StageStateIDs.Value)
		if err != nil {
			return err
		}
		if out == nil {
			p.StageStateIDs.Value = nil
		} else {
			p.StageStateIDs.Value = &out
		}
	}
	if p.OwnedDLCs.Set {
		owned, err := NormalizeOwnedDLCs(p.OwnedDLCs.Value)
		if err != nil {
			return err
		}
		p.OwnedDLCs.Value = owned
	}
	if p.ExportLanguages.Set {
		if p.ExportLanguages.Value == nil {
			return fmt.Errorf("export_languages must be a list of short language codes")
		}
		out, err := normalizeExportLanguages(*p.ExportLanguages.Value)
		if err != nil {
			return err
		}
		p.ExportLanguages.Value = &out
	}
	if p.ImportedReplacePaths.Set {
		out := []string{}
		if p.ImportedReplacePaths.Value != nil {
			out = normalizeReplacePaths(*p.ImportedReplacePaths.Value)
		}
		p.ImportedReplacePaths.Value = &out
	}
	return nil
}

var rootIDRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func normalizeRegistry(raw map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range raw {
		out[k] = v
	}
	groups := []any{}
	if xs, ok := raw["groups"].([]any); ok {
		for _, x := range xs {
			if s, ok := x.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					groups = append(groups, s)
				}
			}
		}
	}
	roots := []any{}
	if xs, ok := raw["root_definitions"].([]any); ok {
		for _, x := range xs {
			m, ok := x.(map[string]any)
			if !ok {
				continue
			}
			id, ok := m["id"].(string)
			if !ok {
				continue
			}
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			e := map[string]any{}
			for k, v := range m {
				e[k] = v
			}
			e["id"] = id
			if b, ok := e["body"].(string); ok {
				e["body"] = strings.TrimSpace(b)
			} else {
				e["body"] = ""
			}
			if _, ok := e["name"].(map[string]any); !ok {
				e["name"] = map[string]any{}
			}
			if _, exists := e["desc"]; exists {
				if _, ok := e["desc"].(map[string]any); !ok {
					e["desc"] = map[string]any{}
				}
			}
			roots = append(roots, e)
			found := false
			for _, g := range groups {
				if g == id {
					found = true
				}
			}
			if !found {
				groups = append(groups, id)
			}
		}
	}
	subs := []any{}
	if xs, ok := raw["sub_ideologies"].([]any); ok {
		for _, x := range xs {
			var e map[string]any
			switch v := x.(type) {
			case string:
				e = map[string]any{"id": v, "group": "", "name": map[string]any{}}
			case map[string]any:
				id, ok := v["id"].(string)
				if !ok || strings.TrimSpace(id) == "" {
					continue
				}
				e = map[string]any{}
				for k, z := range v {
					e[k] = z
				}
				e["id"] = strings.TrimSpace(id)
				if g, ok := e["group"].(string); ok {
					e["group"] = strings.TrimSpace(g)
				} else {
					e["group"] = ""
				}
				if _, ok := e["name"].(map[string]any); !ok {
					e["name"] = map[string]any{}
				}
				if _, exists := e["desc"]; exists {
					if _, ok := e["desc"].(map[string]any); !ok {
						e["desc"] = map[string]any{}
					}
				}
			default:
				continue
			}
			subs = append(subs, e)
		}
	}
	out["groups"] = groups
	out["root_definitions"] = roots
	out["sub_ideologies"] = subs
	return out
}

func validateRoot(id, body string) string {
	if !rootIDRE.MatchString(id) {
		return "id must start with a letter or underscore and use only letters, numbers, and underscores"
	}
	if reftables.Ideologies().IsGroup(id) {
		return "vanilla ROOT groups cannot be redefined"
	}
	if body == "" {
		return "group body is required"
	}
	if len(body) > 100000 {
		return "group body exceeds 100000 characters"
	}
	ast, recoveries, err := paradox.Parse("ideologies = {\n" + id + " = {\n" + body + "\n}\n}\n")
	if err != nil || len(recoveries) > 0 || len(ast.Entries) != 1 || ast.Entries[0].Key == nil || *ast.Entries[0].Key != "ideologies" || ast.Entries[0].Block == nil {
		return "body must stay inside exactly one ROOT group"
	}
	outer := ast.Entries[0].Block
	if len(outer.Entries) != 1 || outer.Entries[0].Key == nil || *outer.Entries[0].Key != id || outer.Entries[0].Block == nil {
		return "body must stay inside exactly one ROOT group"
	}
	hasTypes, hasColor := false, false
	for _, e := range outer.Entries[0].Block.Entries {
		if e.Key != nil && e.Block != nil {
			hasTypes = hasTypes || *e.Key == "types"
			hasColor = hasColor || *e.Key == "color"
		}
	}
	if !hasTypes {
		return "group body must contain a types = { ... } block"
	}
	if !hasColor {
		return "group body must contain a color = { r g b } block"
	}
	return ""
}
