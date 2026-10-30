package guirender

import (
	"sort"
	"strings"
)

func (idx *Index) indexScientistTraitFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	if idx.scientistTraitIcons == nil {
		idx.scientistTraitIcons = map[string]string{}
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		icon := strings.Trim(ciScalar(entry.Block.Entries, "icon"), "\"")
		if icon == "" {
			icon = "GFX_" + *entry.Key
		}
		idx.scientistTraitIcons[strings.ToLower(*entry.Key)] = icon
	}
}

func (idx *Index) scientistTraitIcon(trait string) string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if icon, ok := layer.scientistTraitIcons[strings.ToLower(trait)]; ok {
			return icon
		}
	}
	return "GFX_" + trait
}

func (idx *Index) ScientistsPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	rows := []map[string]any{}
	pictures := []string{}
	order := map[string]int{}
	for at, spec := range idx.specializationList() {
		order[spec.Name] = at
	}
	for _, person := range idx.peopleOf(tag, date) {
		if person == nil || !containsFold(person.Roles, "scientist") {
			continue
		}
		name := idx.localiseOr(person.Name, person.Name)
		traits := []map[string]any{}
		for _, trait := range person.ScientistTraits {
			icon := idx.scientistTraitIcon(trait)
			pictures = append(pictures, icon)
			traits = append(traits, map[string]any{"id": trait, "label": idx.localiseOr(trait, trait), "icon": icon})
		}
		specs := make([]string, 0, len(person.ScientistSkills))
		for spec := range person.ScientistSkills {
			specs = append(specs, spec)
		}
		sort.Slice(specs, func(a, b int) bool { return order[specs[a]] < order[specs[b]] })
		skills := []map[string]any{}
		for _, spec := range specs {
			icon := "GFX_" + spec
			pictures = append(pictures, icon)
			skills = append(skills, map[string]any{
				"specialization": spec, "label": idx.localiseOr(spec+"_short", spec), "level": person.ScientistSkills[spec], "icon": icon,
			})
		}
		row := map[string]any{"token": person.Token, "name": name, "traits": traits, "skills": skills}
		if portrait := person.Large; portrait != "" {
			row["portrait"] = portrait
			pictures = append(pictures, portrait)
		}
		rows = append(rows, row)
	}
	specializations := []map[string]any{}
	for _, spec := range idx.specializationList() {
		icon := "GFX_" + spec.Name
		pictures = append(pictures, icon)
		specializations = append(specializations, map[string]any{
			"id": spec.Name, "icon": icon, "label": idx.localiseOr(spec.Name+"_short", spec.Name),
		})
	}

	catalog := []map[string]any{}
	seenTrait := map[string]bool{}
	for layer := idx; layer != nil; layer = layer.fallback {
		names := make([]string, 0, len(layer.scientistTraitIcons))
		for name := range layer.scientistTraitIcons {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if seenTrait[name] {
				continue
			}
			seenTrait[name] = true
			catalog = append(catalog, map[string]any{"id": name, "label": idx.localiseOr(name, name)})
		}
	}
	return map[string]any{
		"tag": tag, "scientists": rows, "specializations": specializations, "trait_catalog": catalog,
		"title":   idx.localiseOr("SCIENTIST_ROSTER_WINDOW_TITLE", "Scientists"),
		"sprites": idx.SpritesPayload(pictures),
	}
}
