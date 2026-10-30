package guirender

import "strings"

func (idx *Index) CountryProfilePayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	localise := func(key string) string {
		text, _ := idx.Localise(key)
		return text
	}
	names := map[string]any{}
	groups := []map[string]any{}
	for _, group := range idx.IdeologyGroups() {
		if name := localise(tag + "_" + group.Name); name != "" {
			names[group.Name] = name
		}
		groups = append(groups, map[string]any{"id": group.Name, "label": firstNonEmpty(localise(group.Name), group.Name)})
	}
	ruling := idx.RulingPartyOf(tag)
	out := map[string]any{
		"tag":               tag,
		"name":              localise(tag),
		"adjective":         localise(tag + "_ADJ"),
		"names":             names,
		"groups":            groups,
		"ruling":            ruling,
		"graphical_culture": idx.GraphicalCultureOf(tag),

		"shown": idx.CountryName(tag, ruling),
	}
	if politics := idx.countryPoliticsFor(tag); politics != nil && politics.Capital > 0 {
		capital := map[string]any{"id": politics.Capital}
		for _, state := range idx.effectiveStates() {
			if state.ID == politics.Capital {
				capital["name"] = firstNonEmpty(localise(state.Name), state.Name)
				break
			}
		}
		out["capital"] = capital
	}
	if flag := idx.flagPayload(tag, ruling, ""); flag != nil {
		out["flag"] = flag
	}
	return out
}

func (idx *Index) indexCountryFile(name string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}

	if strings.EqualFold(name, "common/countries/colors.txt") {
		for _, entry := range root.Entries {
			if entry == nil || entry.Key == nil || entry.Block == nil {
				continue
			}
			if color, ok := countryColor(entry.Block.Entries); ok {
				idx.tagColors[strings.ToUpper(*entry.Key)] = color
			}
		}
		return
	}

	key := strings.ToLower(strings.TrimPrefix(name, "common/"))
	if color, ok := countryColor(root.Entries); ok {
		idx.countryColors[key] = color
	}
	culture := ciScalar(root.Entries, "graphical_culture")
	if culture == "" {
		return
	}
	idx.countryCultures[key] = strings.Trim(culture, "\"")
}

func (idx *Index) GraphicalCultureOf(tag string) string {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	for layer := idx; layer != nil; layer = layer.fallback {
		file, ok := layer.countryFiles[tag]
		if !ok {
			continue
		}
		key := strings.ToLower(strings.Trim(file, "\""))
		for inner := idx; inner != nil; inner = inner.fallback {
			if culture, found := inner.countryCultures[key]; found {
				return culture
			}
		}
	}
	return ""
}
