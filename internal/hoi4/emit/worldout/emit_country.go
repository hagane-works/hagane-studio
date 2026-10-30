package worldout

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

var graphicalCultureTo2D = map[string]string{
	"western_european_gfx": "western_european_2d",
	"eastern_european_gfx": "eastern_european_2d",
	"middle_eastern_gfx":   "middle_eastern_2d",
	"asian_gfx":            "asian_2d",
	"african_gfx":          "african_2d",
	"southamerican_gfx":    "southamerican_2d",
	"commonwealth_gfx":     "commonwealth_2d",
	"sami_gfx":             "sami_2d",
	"asian_european_gfx":   "asian_2d",
}

const (
	defaultGraphicalCulture   = "western_european_gfx"
	defaultGraphicalCulture2D = "western_european_2d"
)

func deriveGraphicalCulture2D(gfx string) string {
	gfx = pyStrip(gfx)
	if gfx == "" {
		return defaultGraphicalCulture2D
	}
	if v, ok := graphicalCultureTo2D[gfx]; ok && v != "" {
		return v
	}
	return strings.ReplaceAll(gfx, "_gfx", "_2d")
}

func validRGB(value Value) ([3]int64, bool) {
	list, ok := value.([]Value)
	if !ok || len(list) != 3 {
		return [3]int64{}, false
	}
	var out [3]int64
	for i, ch := range list {
		var iv int64
		switch t := ch.(type) {
		case bool:
			if t {
				iv = 1
			}
		case json.Number:
			f, err := t.Float64()
			if err != nil {
				return [3]int64{}, false
			}
			iv = int64(f)
		case string:
			n, err := strconv.ParseInt(pyStrip(t), 10, 64)
			if err != nil {
				return [3]int64{}, false
			}
			iv = n
		default:
			return [3]int64{}, false
		}
		if iv < 0 || iv > 255 {
			return [3]int64{}, false
		}
		out[i] = iv
	}
	return out, true
}

type countryDefinition struct {
	graphicalCulture   string
	graphicalCulture2D string
	color              [3]int64
	legacyAIPP         Value
}

func resolveCountryDefinition(country *OMap, vanillaMap map[string]*OMap) *countryDefinition {
	tag := toUpper(pyStrip(eStrOrEmpty(country, "tag")))
	if tag == "" {
		return nil
	}
	vrec, isVanilla := vanillaMap[tag]

	modelGC := pyStrip(eStrOrEmpty(country, "graphical_culture"))
	modelGC2 := pyStrip(eStrOrEmpty(country, "graphical_culture_2d"))
	modelColor, hasModelColor := validRGB(eGet(country, "color"))

	legacyAIPP, _ := country.Get("use_legacy_ai_pp_spend")
	hasLegacyAIPP := legacyAIPP != nil

	hasCustomGC := modelGC != "" || modelGC2 != ""
	var vrecColor Value
	if vrec != nil {
		vrecColor, _ = vrec.Get("color")
	}
	vanillaColor, hasVanillaColor := validRGB(vrecColor)
	hasCustomColor := hasModelColor &&
		(!isVanilla || !hasVanillaColor || modelColor != vanillaColor)

	if isVanilla && !hasCustomGC && !hasCustomColor && !hasLegacyAIPP {
		return nil
	}

	var gc, gc2 string
	if modelGC != "" {
		gc = modelGC
		gc2 = modelGC2
		if gc2 == "" {
			gc2 = deriveGraphicalCulture2D(gc)
		}
	} else {
		gc = ""
		if vrec != nil {
			gc = pyStrOrEmpty(eGet(vrec, "graphical_culture"))
		}
		if gc == "" {
			gc = defaultGraphicalCulture
		}
		gc2 = modelGC2
		if gc2 == "" && vrec != nil {
			gc2 = pyStrOrEmpty(eGet(vrec, "graphical_culture_2d"))
		}
		if gc2 == "" {
			gc2 = deriveGraphicalCulture2D(gc)
		}
	}

	color := modelColor
	if !hasModelColor {
		if hasVanillaColor {
			color = vanillaColor
		} else {
			color = deriveTagColor(tag)
		}
	}

	return &countryDefinition{
		graphicalCulture:   gc,
		graphicalCulture2D: gc2,
		color:              color,
		legacyAIPP:         legacyAIPP,
	}
}

var cosmeticSplitRe = regexp.MustCompile(`[_\s]+`)

func humanizeCosmeticName(tag string) string {
	var words []string
	for _, w := range cosmeticSplitRe.Split(pyStrip(tag), -1) {
		if w != "" {
			words = append(words, w)
		}
	}
	var out []string
	for _, w := range words {
		if pyIsUpper(w) && len([]rune(w)) <= 4 {
			out = append(out, w)
		} else {
			out = append(out, pyCapWord(w))
		}
	}
	joined := strings.Join(out, " ")
	if joined == "" {
		return tag
	}
	return joined
}

func rgbStr(c [3]int64) string {
	return fmtInt(c[0]) + " " + fmtInt(c[1]) + " " + fmtInt(c[2])
}

type countryDefEntry struct {
	tag        string
	country    *OMap
	definition *countryDefinition
}

func buildCountryColorsTxt(defs []countryDefEntry, modName string) (string, bool) {
	var blocks []string
	for _, e := range defs {
		colorUI, ok := validRGB(eGet(e.country, "color_ui"))
		if !ok {
			continue
		}
		blocks = append(blocks, e.tag+" = {")
		blocks = append(blocks, "\tcolor = rgb { "+rgbStr(e.definition.color)+" }")
		blocks = append(blocks, "\tcolor_ui = rgb { "+rgbStr(colorUI)+" }")
		blocks = append(blocks, "}")
		blocks = append(blocks, "")
	}
	if len(blocks) == 0 {
		return "", false
	}
	lines := []string{
		"#reload countrycolors",
		"# Country UI colors · emitted by " + modName + " export",
		"# vanilla idiom · `common/countries/colors.txt` (color = map, color_ui = list)",
		"",
	}
	lines = append(lines, blocks...)
	return strings.Join(lines, "\n") + "\n", true
}

func (c *emitCtx) emitCountrySections(cosmeticTagsSeen map[string]string) {
	modDir := c.modDir
	modName := c.s.ModName
	modDirName := c.s.ModDirName
	project := c.s.Project
	vanillaCultures := c.t.CountryCultures

	countriesByTag := map[string]*OMap{}

	var authoredTags map[string]bool
	if !c.s.IncludeImported {
		activeTag := toUpper(pyStrOrEmpty(eGet(project, "country_tag")))
		authoredTags = map[string]bool{}
		for _, country := range c.s.Countries {
			ct := toUpper(pyStrip(eStrOrEmpty(country, "tag")))
			if ct == "" {
				continue
			}
			hasAny := false
			if c.s.ContentByTag != nil {
				if v, ok := c.s.ContentByTag.Get(ct); ok {
					hasAny = pyTruthy(v)
				} else if v, ok := c.s.ContentByTag.Get(eStr(country, "tag")); ok {
					hasAny = pyTruthy(v)
				}
			}
			isActive := activeTag != "" && ct == activeTag
			autoAdded := pyTruthy(eGet(country, "auto_added"))
			if !autoAdded || isActive || hasAny {
				authoredTags[ct] = true
			}
		}
	}

	sortedCountries := append([]*OMap{}, c.s.Countries...)
	sort.SliceStable(sortedCountries, func(i, j int) bool {
		return toUpper(eStrOrEmpty(sortedCountries[i], "tag")) <
			toUpper(eStrOrEmpty(sortedCountries[j], "tag"))
	})
	var countryDefinitions []countryDefEntry
	for _, country := range sortedCountries {
		ctag2 := toUpper(pyStrip(eStrOrEmpty(country, "tag")))
		if ctag2 != "" {
			if _, exists := countriesByTag[ctag2]; !exists {
				countriesByTag[ctag2] = country
			}
		}
		if ctag2 == "" {
			continue
		}
		if authoredTags != nil && !authoredTags[ctag2] {
			continue
		}
		definition := resolveCountryDefinition(country, vanillaCultures)
		if definition == nil {
			continue
		}
		countryDefinitions = append(countryDefinitions,
			countryDefEntry{ctag2, country, definition})
	}

	type emitPlanEntry struct {
		tag          string
		country      *OMap
		definition   *countryDefinition
		defBasename  string
		emitRegistry bool
	}
	var countryEmitPlan []emitPlanEntry
	for _, e := range countryDefinitions {
		shadowName := ""
		if _, isVanilla := vanillaCultures[e.tag]; isVanilla {
			shadowName = c.t.CountryFilenames[e.tag]
		}
		if shadowName != "" {
			countryEmitPlan = append(countryEmitPlan,
				emitPlanEntry{e.tag, e.country, e.definition, shadowName, false})
		} else {
			countryEmitPlan = append(countryEmitPlan,
				emitPlanEntry{e.tag, e.country, e.definition, e.tag + ".txt", true})
		}
	}

	var registryLines [][2]string
	for _, e := range countryEmitPlan {
		if e.emitRegistry {
			registryLines = append(registryLines, [2]string{e.tag, e.defBasename})
		}
	}
	if len(registryLines) > 0 {
		ctLines := []string{
			"# Country tag registry · emitted by " + modName + " export",
			"",
		}
		for _, rl := range registryLines {
			ctLines = append(ctLines, rl[0]+" = \"countries/"+rl[1]+"\"")
		}
		c.write(modDir+"common/country_tags/00_"+modDirName+"_countries.txt",
			strings.Join(ctLines, "\n")+"\n")
	}

	for _, e := range countryEmitPlan {
		defLines := []string{
			"# " + e.tag + " country definition · emitted by " + modName + " export",
			"# common/countries/<file>.txt · graphical_culture (art set) + map color",
			"",
			"graphical_culture = " + e.definition.graphicalCulture,
			"graphical_culture_2d = " + e.definition.graphicalCulture2D,
			"",
			"color = rgb { " + rgbStr(e.definition.color) + " }",
			"",
		}
		if e.definition.legacyAIPP != nil {
			tok := "no"
			if pyTruthy(e.definition.legacyAIPP) {
				tok = "yes"
			}
			defLines = append(defLines, "use_legacy_ai_pp_spend = "+tok)
			defLines = append(defLines, "")
		}
		c.write(modDir+"common/countries/"+e.defBasename,
			strings.Join(defLines, "\n")+"\n")
	}

	if colorsTxt, ok := buildCountryColorsTxt(countryDefinitions, modName); ok {
		c.write(modDir+"common/countries/zz_"+modDirName+"_colors.txt", colorsTxt)
	}

	if !c.s.IsSubmod {
		regRaw, _ := project.Get("ideology_registry")
		emittedSubs := customSubIdeologies(regRaw)
		emittedRoots := customRootDefinitions(regRaw)
		if len(emittedSubs) > 0 || len(emittedRoots) > 0 {
			byGrp := map[string][]string{}
			var grpOrder []string
			for _, s := range emittedSubs {
				grp := pyStrOrEmpty(eGet(s, "group"))
				if _, ok := byGrp[grp]; !ok {
					grpOrder = append(grpOrder, grp)
				}
				byGrp[grp] = append(byGrp[grp], pyStrOrEmpty(eGet(s, "id")))
			}
			var grpParts []string
			for _, g := range grpOrder {
				grpParts = append(grpParts, g+": "+strings.Join(byGrp[g], ", "))
			}
			grpDesc := strings.Join(grpParts, "; ")
			var rootIDs []string
			for _, root := range emittedRoots {
				if id := pyStrOrEmpty(eGet(root, "id")); id != "" {
					rootIDs = append(rootIDs, id)
				}
			}
			rootDesc := strings.Join(rootIDs, ", ")
			if c.fullText != "" {
				c.write(modDir+"common/ideologies/00_ideologies.txt", c.fullText)
				c.emitCustomRootPortraits(modDir, modDirName, emittedRoots)
				c.politics = append(c.politics, exportmsg.Message{Key: "politics.note.ideologies_full", Args: exportmsg.Args{
					"subs": grpDesc, "groups": rootDesc,
				}})

				c.emitCustomIdeologyLoc(modDir, modDirName, emittedRoots, emittedSubs)
			} else if c.s.RawOverride {
				c.politics = append(c.politics, exportmsg.Message{Key: "politics.note.ideologies_raw", Args: exportmsg.Args{
					"subs": grpDesc, "groups": rootDesc,
				}})
			} else if c.inheritedIdeologySource {

				if text, kept := CustomRootIdeologiesFile(regRaw, c.sourceRoots); text != "" && !c.ideology.RawOverride {
					c.write(modDir+"common/ideologies/zz_"+modDirName+"_ideologies.txt", text)
					c.emitCustomRootPortraits(modDir, modDirName, kept)
					keptIDs := map[string]bool{}
					for _, root := range kept {
						keptIDs[pyStrOrEmpty(eGet(root, "id"))] = true
					}
					var keptSubs []*OMap
					for _, sub := range emittedSubs {
						if keptIDs[pyStrOrEmpty(eGet(sub, "group"))] {
							keptSubs = append(keptSubs, sub)
						}
					}
					c.politics = append(c.politics, exportmsg.Message{Key: "politics.note.ideologies_beside", Args: exportmsg.Args{
						"file": "common/ideologies/zz_" + modDirName + "_ideologies.txt", "groups": rootDesc,
					}})
					c.emitCustomIdeologyLoc(modDir, modDirName, kept, keptSubs)
				}
			} else {
				c.politics = append(c.politics, exportmsg.Message{Key: "politics.ideologies_missing", Args: exportmsg.Args{
					"subs": grpDesc, "groups": rootDesc,
				}})
			}
		}
	}

	c.emitCosmeticDefinitions(cosmeticTagsSeen, c.s.CosmeticDefinitions, c.s.PreservedLocKeys)
}

func (c *emitCtx) emitCustomRootPortraits(modDir, modDirName string, emittedRoots []*OMap) {
	var portraitTags []string
	for _, country := range c.s.Countries {
		portraitTags = append(portraitTags, eStrOrEmpty(country, "tag"))
	}
	for _, history := range c.s.HistoryFiles {
		portraitTags = append(portraitTags, eStrOrEmpty(history, "country_tag"))
	}
	if portraits := buildCustomRootPoliticalPortraits(portraitTags, emittedRoots); portraits != "" {
		c.write(modDir+"portraits/zz_"+modDirName+"_political.txt", portraits)
	}
}

func (c *emitCtx) emitCustomIdeologyLoc(modDir, modDirName string, emittedRoots, emittedSubs []*OMap) {

	pick := func(m *OMap, lang string) string {
		var order []string
		order = append(order, lang, "en")
		if m != nil {
			order = append(order, m.Keys()...)
		}
		for _, k := range order {
			if m == nil {
				break
			}
			v, _ := m.Get(k)
			if s := pyStrip(pyStrOrEmpty(v)); s != "" {
				return s
			}
		}
		return ""
	}
	ideoLocByLang := map[string]string{}
	var ideoLangs []string
	for _, tr := range c.triples {
		lang, subdir, suffix := tr.Code, tr.Subdir, tr.Suffix
		ymlLines := []string{suffix + ":"}
		for _, root := range emittedRoots {
			rid := pyStrOrEmpty(eGet(root, "id"))
			if rid == "" {
				continue
			}
			nameMap := eMap(root, "name")
			descMap := eMap(root, "desc")
			nm := pick(nameMap, lang)
			if !isPlaceholderDisplayName(nm, rid) {
				escaped := ymlEscapeValue(nm)
				ymlLines = append(ymlLines,
					" "+rid+":0 \""+escaped+"\"",
					" "+rid+"_noun:0 \""+escaped+"\"")
			}
			if dv := pick(descMap, lang); dv != "" {
				ymlLines = append(ymlLines,
					" "+rid+"_desc:0 \""+ymlEscapeValue(dv)+"\"")
			}
		}
		for _, s := range emittedSubs {
			sid := pyStrOrEmpty(eGet(s, "id"))
			if sid == "" {
				continue
			}
			var nameMap, descMap *OMap
			if nm, ok := s.Get("name"); ok {
				nameMap, _ = nm.(*OMap)
			}
			if dm, ok := s.Get("desc"); ok {
				descMap, _ = dm.(*OMap)
			}
			nm := pick(nameMap, lang)
			if !isPlaceholderDisplayName(nm, sid) {
				ymlLines = append(ymlLines,
					" "+sid+":0 \""+ymlEscapeValue(nm)+"\"")
			}
			dv := pick(descMap, lang)
			if dv != "" {
				ymlLines = append(ymlLines,
					" "+sid+"_desc:0 \""+ymlEscapeValue(dv)+"\"")
			}
		}
		ymlLines = append(ymlLines, "")
		c.write(
			modDir+"localisation/"+subdir+"/zz_"+modDirName+"_ideology_"+suffix+".yml",
			bomStr+strings.Join(ymlLines, "\n"))
		ideoLocByLang[lang] = strings.Join(ymlLines, "\n")
		ideoLangs = append(ideoLangs, lang)
	}
	c.symmetry = append(c.symmetry,
		assertYmlSymmetricMulti(ideoLangs, ideoLocByLang, "ideology")...)
}
