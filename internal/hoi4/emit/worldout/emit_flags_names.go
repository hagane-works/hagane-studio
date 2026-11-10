package worldout

import "strings"

func (c *emitCtx) emitCountryFlagsNames() {
	regRaw, _ := c.s.Project.Get("ideology_registry")
	ideologyKeys := mergeWithVanilla(regRaw).groups

	for _, country := range c.s.Countries {
		ctag := toUpper(eStr(country, "tag"))
		for _, tr := range c.triples {
			lang, subdir, suffix := tr.Code, tr.Subdir, tr.Suffix
			baseName := resolveCountryExportName(country, lang)
			if lang == "en" {
				if baseName == "" {
					baseName = pyStrip(eStrOrEmpty(country, "name_en"))
				}
			} else if lang == "zh" {
				if baseName == "" {
					baseName = pyStrip(eStrOrEmpty(country, "name_zh"))
				}
			}
			strippedOverride := func(field string) string {
				locs := eMap(country, "ideology_name_locs")
				if locs != nil {
					if ideologyLocs, ok := locs.Get(strings.TrimPrefix(field, "name_")); ok {
						if ideologyMap, isMap := ideologyLocs.(*OMap); isMap {
							if v, found := ideologyMap.Get(lang); found {
								if stripped := pyStrip(pyStrOrEmpty(v)); stripped != "" {
									return stripped
								}
							}
						}
					}
				}
				return resolveJSONLocalisation(
					country, lang, "", field+"_en", field+"_zh",
				)
			}
			overrides := make(map[string]string, len(ideologyKeys))
			for _, ideology := range ideologyKeys {
				overrides[ideology] = strippedOverride("name_" + ideology)
			}
			anySet := false
			for _, k := range ideologyKeys {
				if overrides[k] != "" {
					anySet = true
					break
				}
			}
			if baseName == "" && !anySet {
				continue
			}
			ymlLines := []string{suffix + ":"}
			fallbackValue := baseName
			if fallbackValue == "" {
				for _, k := range ideologyKeys {
					if overrides[k] != "" {
						fallbackValue = overrides[k]
						break
					}
				}
			}
			adjective := resolveJSONLocalisation(
				country, lang, "adjective_locs", "adjective_en", "adjective_zh",
			)
			if adjective == "" {
				adjective = fallbackValue
			}
			// The definite (_DEF) form and the per-ideology adjective are not
			// part of the country model; they are filled from the plain name
			// and adjective. Like cosmetic names, such a filled-in value does
			// not replace text the project's own localisation defines for the
			// same key ("the Republic of China" for CHI_DEF).
			preserved := c.s.PreservedLocKeys[suffix]
			filled := func(key, value string) {
				if !preserved[key] {
					ymlLines = append(ymlLines, ymlLine(key, value, false))
				}
			}
			ymlLines = append(ymlLines, ymlLine(ctag, fallbackValue, false))
			filled(ctag+"_DEF", fallbackValue)
			ymlLines = append(ymlLines, ymlLine(ctag+"_ADJ", adjective, false))
			for _, ideology := range ideologyKeys {
				value := overrides[ideology]
				key := ctag + "_" + ideology
				if value == "" {
					filled(key, fallbackValue)
					value = fallbackValue
				} else {
					ymlLines = append(ymlLines, ymlLine(key, value, false))
				}
				filled(key+"_DEF", value)
				filled(key+"_ADJ", adjective)
			}
			ymlLines = append(ymlLines, "")
			c.write(
				c.modDir+"localisation/"+subdir+"/"+ctag+"_country_"+suffix+".yml",
				bomStr+strings.Join(ymlLines, "\n"),
			)
		}
	}
}
