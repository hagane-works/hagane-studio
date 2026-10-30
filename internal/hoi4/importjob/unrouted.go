package importjob

import (
	"io/fs"
	"path/filepath"
	"strings"
)

var routedPrefixes = []string{
	"common/ai_strategy_plans/",
	"common/ai_templates/",
	"common/autonomous_states/",
	"common/bookmarks/",
	"common/characters/",
	"common/countries/",
	"common/country_leader/",
	"common/decisions/",
	"common/doctrines/",
	"common/dynamic_modifiers/",
	"common/factions/",
	"common/ideas/",

	"common/ideologies/",
	"common/intelligence_agencies/",
	"common/military_industrial_organization/",
	"common/modifier_definitions/",
	"common/national_focus/",
	"common/on_actions/",
	"common/opinion_modifiers/",
	"common/raids/",
	"common/scientist_traits/",
	"common/scripted_effects/",
	"common/scripted_guis/",
	"common/scripted_localisation/",
	"common/scripted_triggers/",
	"common/special_projects/",
	"common/technologies/",
	"common/technology_tags/",
	"common/unit_leader/",
	"common/units/",
	"common/wargoals/",
	"events/",
	"history/countries/",
	"history/states/",
	"history/units/",
	"localisation/",
	"gfx/",
	"interface/",
}

var routedRootFiles = map[string]bool{
	"descriptor.mod": true,
	"thumbnail.png":  true,
	"readme.md":      true,
	"changelog.md":   true,
	".gitignore":     true,
	".gitattributes": true,
}

func UnroutedByDir(root string) (map[string]int, error) {
	out := map[string]int{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return nil
		}
		n := strings.ToLower(filepath.ToSlash(rel))
		if !strings.Contains(n, "/") {
			if !routedRootFiles[n] {
				out["<mod root>"]++
			}
			return nil
		}
		for _, prefix := range routedPrefixes {
			if strings.HasPrefix(n, prefix) {
				return nil
			}
		}
		if IsModMapFile(n) {
			return nil
		}
		out[filepath.ToSlash(filepath.Dir(rel))]++
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
