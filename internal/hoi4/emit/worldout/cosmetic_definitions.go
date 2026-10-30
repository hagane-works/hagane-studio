package worldout

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/cosmetic"
)

func (c *emitCtx) emitCosmeticDefinitions(references map[string]string, definitions map[string]cosmetic.Definition, preservedLocKeys map[string]map[string]bool) {
	owners := references
	if c.s.CosmeticOwners != nil {

		owners = c.s.CosmeticOwners
	}
	tagSet := make(map[string]bool, len(references)+len(definitions))
	for tag := range references {
		if !c.t.CosmeticTags[tag] {
			tagSet[tag] = true
		}
	}

	for tag := range definitions {
		tagSet[tag] = true
	}
	if len(tagSet) == 0 {
		return
	}
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	countries := make(map[string]*OMap)
	sortedCountries := append([]*OMap(nil), c.s.Countries...)
	sort.SliceStable(sortedCountries, func(i, j int) bool {
		return toUpper(eStrOrEmpty(sortedCountries[i], "tag")) < toUpper(eStrOrEmpty(sortedCountries[j], "tag"))
	})
	for _, country := range sortedCountries {
		tag := toUpper(pyStrip(eStrOrEmpty(country, "tag")))
		if tag != "" && countries[tag] == nil {
			countries[tag] = country
		}
	}
	colorLines := []string{
		"# Cosmetic tag definitions · emitted by " + c.s.ModName + " export",
		"# vanilla idiom · `common/countries/cosmetic.txt` color blocks",
		"",
	}
	for _, tag := range tags {
		def := definitions[tag]
		def.Tag = tag
		if !c.s.IncludeImported {
			def.ImportedColor, def.ImportedColorUI = nil, nil
		}
		ownerColor, ownerUI := CosmeticOwnerColors(countries[owners[tag]], c.t.CountryCultures)
		color, ui := cosmetic.ResolveColors(def, ownerColor, ownerUI)
		colorLines = append(colorLines, tag+" = {",
			"\tcolor = rgb { "+cosmeticRGBString(color)+" }",
			"\tcolor_ui = rgb { "+cosmeticRGBString(ui)+" }", "}", "")
	}

	c.write(c.modDir+"common/countries/cosmetic.txt", strings.Join(colorLines, "\n")+"\n")

	for _, tr := range c.triples {
		lines := []string{tr.Suffix + ":"}
		for _, tag := range tags {
			def := definitions[tag]
			def.Tag = tag

			def.ImportedNameLocs = nil
			for _, suffix := range cosmetic.Suffixes() {
				key := tag
				if suffix != "" {
					key += "_" + suffix
				}
				name := cosmetic.ResolveName(def, tr.Suffix, suffix)
				if name.Source != "authored" && preservedLocKeys[tr.Suffix][key] {
					continue
				}
				lines = append(lines, " "+key+":0 \""+ymlEscapeValue(name.Value)+"\"")
			}
		}
		lines = append(lines, "")

		c.write(c.modDir+"localisation/"+tr.Subdir+"/zz_"+c.s.ModDirName+"_cosmetic_"+tr.Suffix+".yml", bomStr+strings.Join(lines, "\n"))
	}
}

func CosmeticOwnerColors(owner *OMap, vanilla map[string]*OMap) (color, ui *cosmetic.RGB) {
	if owner == nil {
		return nil, nil
	}
	if definition := resolveCountryDefinition(owner, vanilla); definition != nil {
		value := cosmetic.RGB{int(definition.color[0]), int(definition.color[1]), int(definition.color[2])}
		color = &value
	} else if value, ok := validRGB(eGet(owner, "color")); ok {
		converted := cosmetic.RGB{int(value[0]), int(value[1]), int(value[2])}
		color = &converted
	}
	if value, ok := validRGB(eGet(owner, "color_ui")); ok {
		converted := cosmetic.RGB{int(value[0]), int(value[1]), int(value[2])}
		ui = &converted
	}
	return color, ui
}

func cosmeticRGBString(value cosmetic.RGB) string {
	return rgbStr([3]int64{int64(value[0]), int64(value[1]), int64(value[2])})
}
