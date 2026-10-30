package worldout

import (
	"sort"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/exportmsg"
)

const bomStr = "\uFEFF"

type emitCtx struct {
	t       *Tables
	s       *Scenario
	triples []LangTriple
	modDir  string

	fullText string
	ideology IdeologyExportFacts

	inheritedIdeologySource bool
	sourceRoots             map[string]bool

	records  []Record
	politics []exportmsg.Message
	orphan   []exportmsg.Message
	symmetry []exportmsg.Message
	oob      []exportmsg.Message
	skip     map[string]int64
}

func (c *emitCtx) write(path, text string) {
	c.records = append(c.records, Record{Path: path, Text: text})
}

func (c *emitCtx) exportLangs() []string {
	var out []string
	for _, tr := range c.triples {
		out = append(out, tr.Code)
	}
	return out
}

func EmitScenario(t *Tables, s *Scenario) ([]Record, Meta) {
	var langs []string
	if v, ok := s.Project.Get("export_languages"); ok {
		if l, isList := v.([]Value); isList {
			for _, item := range l {
				if str, isStr := item.(string); isStr {
					langs = append(langs, str)
				}
			}
		}
	}
	c := &emitCtx{
		t:       t,
		s:       s,
		triples: buildExportLangTriples(langs),
		modDir:  s.ModDirName + "/",
		skip:    map[string]int64{},
	}
	c.politics = []exportmsg.Message{}
	c.orphan = []exportmsg.Message{}
	c.symmetry = []exportmsg.Message{}
	c.oob = []exportmsg.Message{}
	ideologyPlan := BuildIdeologyExportPlan(t, s.Project, s.IsSubmod, s.RawOverride)
	if s.IdeologyPlan != nil {
		ideologyPlan = *s.IdeologyPlan
	}
	if s.FullTextOverride != nil && ideologyPlan.Facts.Standalone &&
		ideologyPlan.Facts.HasCustomDefinitions && !ideologyPlan.Facts.RawOverride {

		ideologyPlan.FullText = *s.FullTextOverride
		ideologyPlan.Facts.FullFileGenerated = ideologyPlan.FullText != ""
	}
	c.fullText = ideologyPlan.FullText
	c.ideology = ideologyPlan.Facts
	c.inheritedIdeologySource = ideologyPlan.InheritedSource
	c.sourceRoots = ideologyPlan.SourceRoots

	c.emitCountryFlagsNames()
	c.emitHistoryStatesCountryMain()

	return c.records, Meta{
		Politics: c.politics,
		Orphan:   c.orphan,
		Symmetry: c.symmetry,
		Ideology: c.ideology,
		OOB:      c.oob,
		Skip:     c.skip,
	}
}

func pyLStrip(s string) string {
	return strings.TrimLeftFunc(s, unicode.IsSpace)
}

func bookmarkHistoryAnyLang(entry *OMap, langs []string) string {
	historyLocs := eMap(entry, "history_locs")
	if historyLocs != nil {
		probes := []string{"en", "zh"}
		probes = append(probes, langs...)
		probes = append(probes, historyLocs.Keys()...)
		for _, lang := range probes {
			if val, ok := historyLocs.Get(lang); ok {
				if stripped := pyStrip(pyStrOrEmpty(val)); stripped != "" {
					return stripped
				}
			}
		}
	}
	probes := []string{"history_en", "history_zh"}
	for _, l := range langs {
		probes = append(probes, "history_"+l)
	}
	for _, probe := range probes {
		val := pyStrip(pyStrOrEmpty(eGet(entry, probe)))
		if val != "" {
			return val
		}
	}
	return ""
}

func (c *emitCtx) emitHistoryStatesCountryMain() {
	modDir := c.modDir
	modName := c.s.ModName
	modDirName := c.s.ModDirName
	project := c.s.Project

	bookmarkGfxEntries := c.s.BookmarkGfx
	resolvedBackdropSprites := map[string]bool{}
	for _, e := range bookmarkGfxEntries {
		resolvedBackdropSprites[e[0]] = true
	}
	if len(bookmarkGfxEntries) > 0 {
		c.write(modDir+"interface/hagane_bookmarks.gfx",
			emitSpriteBlock(bookmarkGfxEntries))

	}

	exportLangs := c.exportLangs()

	for _, bm := range c.s.Bookmarks {
		bmID := pyStrip(eStrOrEmpty(bm, "bookmark_id"))
		if bmID == "" {
			continue
		}
		nameKey := toUpper(bmID) + "_NAME"
		descKey := toUpper(bmID) + "_DESC"

		bmDescAny := resolveDescriptionAnyLang(bm)
		bmNamePlan := planBookmarkNames(bm, exportLangs, bmID, nameKey)
		c.symmetry = append(c.symmetry, bmNamePlan.Findings...)

		bmLines := []string{"bookmarks = {", "\tbookmark = {"}
		bmLines = append(bmLines, "\t\tname = \""+nameKey+"\"")
		if bmDescAny != "" {
			bmLines = append(bmLines, "\t\tdesc = \""+descKey+"\"")
		}
		date := pyStrOrEmpty(eGet(bm, "date"))
		if date == "" {
			date = "1936.1.1.12"
		}
		bmLines = append(bmLines, "\t\tdate = "+date)
		customSprite := bookmarkPictureSpriteName(bmID)
		pictureVal := pyStrip(eStrOrEmpty(bm, "picture"))
		pictureRef, discarded := resolveBookmarkPicture(
			pictureVal, customSprite, resolvedBackdropSprites)
		if discarded {
			c.politics = append(c.politics, exportmsg.Message{Key: "politics.bookmark_picture", Args: exportmsg.Args{
				"subject": bmID, "picture": pyStrip(pictureVal), "fallback": pictureRef,
			}})
		}
		bmLines = append(bmLines, "\t\tpicture = \""+pictureRef+"\"")
		if pyTruthy(eGet(bm, "default_country")) {
			bmLines = append(bmLines,
				"\t\tdefault_country = \""+pyStr(eGet(bm, "default_country"))+"\"")
		}
		if pyTruthy(eGet(bm, "default_flag")) {
			bmLines = append(bmLines, "\t\tdefault = yes")
		}

		var filtersList []string
		for _, f := range eList(bm, "filters") {
			if s := pyStrip(pyStr(f)); s != "" {
				filtersList = append(filtersList, s)
			}
		}
		if len(filtersList) > 0 {
			bmLines = append(bmLines,
				"\t\tfilters = { "+strings.Join(filtersList, " ")+" }")
		}

		for _, entryV := range eList(bm, "selectable_countries") {
			entry, ok := entryV.(*OMap)
			if !ok {
				continue
			}
			ctag := toUpper(pyStrip(pyStrOrEmpty(eGet(entry, "tag"))))
			if ctag == "" {
				continue
			}
			historyKey := ctag + "_" + toUpper(bmID) + "_DESC"
			countryBlockStart := len(bmLines)
			bmLines = append(bmLines, "")
			bmLines = append(bmLines, "\t\t\""+ctag+"\" = {")
			if bookmarkHistoryAnyLang(entry, exportLangs) != "" {
				bmLines = append(bmLines, "\t\t\thistory = \""+historyKey+"\"")
			}
			if pyTruthy(eGet(entry, "minor")) {
				bmLines = append(bmLines, "\t\t\tminor = yes")
			}
			var labelList []string
			for _, l := range eList(entry, "label") {
				if s := pyStrip(pyStr(l)); s != "" {
					labelList = append(labelList, s)
				}
			}
			if len(labelList) > 0 {
				bmLines = append(bmLines,
					"\t\t\tlabel = { "+strings.Join(labelList, " ")+" }")
			}
			ideologyVal := pyStrip(pyStrOrEmpty(eGet(entry, "ideology")))
			if ideologyVal != "" {
				bmLines = append(bmLines, "\t\t\tideology = "+ideologyVal)
			}
			var ideasList []string
			for _, i := range eList(entry, "ideas") {
				if s := pyStrip(pyStr(i)); s != "" {
					ideasList = append(ideasList, s)
				}
			}
			if len(ideasList) > 0 {
				bmLines = append(bmLines, "\t\t\tideas = {")
				for _, ideaID := range ideasList {
					bmLines = append(bmLines, "\t\t\t\t"+ideaID)
				}
				bmLines = append(bmLines, "\t\t\t}")
			}
			var focusList []string
			for _, f := range eList(entry, "focuses") {
				if s := pyStrip(pyStr(f)); s != "" {
					focusList = append(focusList, s)
				}
			}
			if len(focusList) > 0 {
				bmLines = append(bmLines, "\t\t\tfocuses = {")
				max := len(focusList)
				if max > 3 {
					max = 3
				}
				for _, fid := range focusList[:max] {
					bmLines = append(bmLines, "\t\t\t\t"+fid)
				}
				bmLines = append(bmLines, "\t\t\t}")
			}
			availableVal := pyStrip(pyStrOrEmpty(eGet(entry, "available")))
			if availableVal != "" {
				if !strings.Contains(availableVal, "\n") &&
					!strings.Contains(availableVal, "{") &&
					!strings.Contains(availableVal, "=") {
					bmLines = append(bmLines, "\t\t\tavailable = "+availableVal)
				} else {
					bmLines = append(bmLines, "\t\t\tavailable = {")
					for _, vl := range pySplitlines(availableVal) {
						if pyStrip(vl) != "" {
							bmLines = append(bmLines, "\t\t\t\t"+pyStrip(vl))
						}
					}
					bmLines = append(bmLines, "\t\t\t}")
				}
			}
			if countryExtras := eMap(entry, "raw_extras"); countryExtras != nil {
				for _, rk := range countryExtras.Keys() {
					if rk == "label" {
						continue
					}

					if rk == "new_content" {
						continue
					}
					rv, _ := countryExtras.Get(rk)
					emitRawExtrasBlockOrScalar(&bmLines, rk, rv, "\t\t\t")
				}
			}
			bmLines = append(bmLines, "\t\t}")
			if len(bmLines)-countryBlockStart == 3 {
				bmLines = bmLines[:countryBlockStart]
			}
		}

		if rawExtras := eMap(bm, "raw_extras"); rawExtras != nil {
			for _, rk := range rawExtras.Keys() {
				if rk == "filters" {
					continue
				}
				rv, _ := rawExtras.Get(rk)
				emitRawExtrasBlockOrScalar(&bmLines, rk, rv, "\t\t")
			}
		}

		if !bookmarkHasWeather(eMap(bm, "raw_extras")) {
			seed := bookmarkWeatherSeed(bmID)
			bmLines = append(bmLines, "\t\teffect = {")
			bmLines = append(bmLines, "\t\t\trandomize_weather = "+fmtInt(seed))
			bmLines = append(bmLines, "\t\t}")
		}

		bmLines = append(bmLines, "\t}")
		bmLines = append(bmLines, "}")
		c.write(modDir+"common/bookmarks/"+bmID+".txt",
			strings.Join(bmLines, "\n")+"\n")

		bookmarkLocByLang := map[string]string{}
		var bookmarkLangs []string
		for _, tr := range c.triples {
			lang, subdir, suffix := tr.Code, tr.Subdir, tr.Suffix
			ymlLines := []string{suffix + ":"}
			if nameVal := pyStrip(bmNamePlan.ByLang[lang]); nameVal != "" {
				ymlLines = append(ymlLines, ymlLine(nameKey, nameVal, false))
			}
			if bmDescAny != "" {
				descVal := bmDescAny
				if lang != "en" {
					if v := resolveDescription(bm, lang); v != "" {
						descVal = v
					}
				}
				ymlLines = append(ymlLines, ymlLine(descKey, descVal, false))
			}
			for _, entryV := range eList(bm, "selectable_countries") {
				entry, ok := entryV.(*OMap)
				if !ok {
					continue
				}
				ctag := toUpper(pyStrip(pyStrOrEmpty(eGet(entry, "tag"))))
				if ctag == "" {
					continue
				}
				historyKey := ctag + "_" + toUpper(bmID) + "_DESC"
				histAny := bookmarkHistoryAnyLang(entry, exportLangs)
				if histAny == "" {
					continue
				}
				histLang := resolveJSONLocalisation(
					entry, lang, "history_locs", "history_en", "history_zh",
				)
				historyVal := histLang
				if historyVal == "" {
					historyVal = histAny
				}
				ymlLines = append(ymlLines, ymlLine(historyKey, historyVal, false))
			}
			if len(ymlLines) == 1 {
				continue
			}
			ymlLines = append(ymlLines, "")
			bookmarkLocByLang[lang] = strings.Join(ymlLines, "\n")
			bookmarkLangs = append(bookmarkLangs, lang)
			c.write(modDir+"localisation/"+subdir+"/"+bmID+"_"+suffix+".yml",
				bomStr+strings.Join(ymlLines, "\n"))
		}
		c.symmetry = append(c.symmetry,
			assertYmlSymmetricMulti(bookmarkLangs, bookmarkLocByLang, "bookmark")...)
	}

	regRaw, _ := project.Get("ideology_registry")
	resolveRulingPartyGroup := func(value string) (string, bool) {
		return resolveRulingPartyGroup(regRaw, c.s.IsSubmod, value)
	}

	emittedHistoryFileCtags := map[string]bool{}
	emittedSetPoliticsCtags := map[string]bool{}
	cosmeticTagsSeen := map[string]string{}
	recruitedCtags := map[string]bool{}

	var bufferOrder []string
	buffers := map[string][]string{}
	bufSet := func(ctag string, lines []string) {
		if _, ok := buffers[ctag]; !ok {
			bufferOrder = append(bufferOrder, ctag)
		}
		buffers[ctag] = lines
	}

	recruitsFor := func(ctag string) []Value {
		if c.s.RecruitByCtag == nil {
			return nil
		}
		v, ok := c.s.RecruitByCtag.Get(ctag)
		if !ok {
			return nil
		}
		l, _ := v.([]Value)
		return l
	}

	foldRecruits := func(hfLines *[]string, ctag string, leadBlank, trailBlank bool) {
		recruits := recruitsFor(ctag)
		if len(recruits) == 0 {
			return
		}
		if leadBlank {
			*hfLines = append(*hfLines, "")
		}
		for _, rtag := range recruits {
			*hfLines = append(*hfLines, "recruit_character = "+pyStr(rtag))
		}
		if trailBlank {
			*hfLines = append(*hfLines, "")
		}
		recruitedCtags[ctag] = true
	}

	headerLines := func(ctag string) []string {
		return []string{
			"# " + ctag + " history overrides · emitted by " + modName + " export",
			"# Filename idiom `" + ctag + " - z<modname>.txt` · loads after " +
				"vanilla `" + ctag + " - <Name>.txt`(z > U/G/F/etc 字母序覆盖)",
		}
	}

	clFor := func(ctag string) []*OMap {
		if c.s.ClByCtag == nil {
			return nil
		}
		v, ok := c.s.ClByCtag.Get(ctag)
		if !ok {
			return nil
		}
		l, _ := v.([]Value)
		var out []*OMap
		for _, item := range l {
			if m, isMap := item.(*OMap); isMap {
				out = append(out, m)
			}
		}
		return out
	}

	for _, hf := range c.s.HistoryFiles {
		ctag := toUpper(pyStrip(eStrOrEmpty(hf, "country_tag")))
		if ctag == "" {
			continue
		}
		if emittedHistoryFileCtags[ctag] {
			c.politics = append(c.politics, exportmsg.Message{Key: "politics.duplicate_history", Args: exportmsg.Args{
				"subject": ctag, "file": "history/countries/" + ctag + " - z" + modDirName + ".txt",
			}})
			continue
		}
		emittedHistoryFileCtags[ctag] = true
		if pyStrip(eStrOrEmpty(hf, "set_politics_ruling_party")) != "" {
			emittedSetPoliticsCtags[ctag] = true
		}

		hfLines := append(headerLines(ctag), "")

		clBlock := emitHistoryFileLeaderReplacementBlock(ctag, clFor(ctag))
		hfLines = append(hfLines, clBlock...)

		foldRecruits(&hfLines, ctag, false, true)

		if pyTruthy(eGet(hf, "capital")) {
			hfLines = append(hfLines, "capital = "+pyStr(eGet(hf, "capital")))
			hfLines = append(hfLines, "")
		}

		var spLines []string
		rulingPartyRaw := eGet(hf, "set_politics_ruling_party")
		rpGroup, rpOK := "", false
		if pyTruthy(rulingPartyRaw) {
			rpGroup, rpOK = resolveRulingPartyGroup(pyStr(rulingPartyRaw))
			if !rpOK {
				c.politics = append(c.politics, exportmsg.Message{Key: "politics.ruling_party_dropped", Args: exportmsg.Args{
					"subject": ctag, "value": pyStrip(pyStrOrEmpty(rulingPartyRaw)),
				}})
			}
		}
		if rpOK {
			spLines = append(spLines, "\truling_party = "+rpGroup)
			if pyTruthy(eGet(hf, "set_politics_last_election")) {
				spLines = append(spLines,
					"\tlast_election = \""+pyStr(eGet(hf, "set_politics_last_election"))+"\"")
			}
			if ef := eGet(hf, "set_politics_election_frequency"); pyTruthy(ef) {
				if f, ok := numFloat(ef); ok && f > 0 {
					spLines = append(spLines,
						"\telection_frequency = "+pyStr(ef))
				}
			}
			if ea, ok := hf.Get("set_politics_elections_allowed"); ok && ea != nil {
				tok := "no"
				if pyTruthy(ea) {
					tok = "yes"
				}
				spLines = append(spLines, "\telections_allowed = "+tok)
			}
		}
		if len(spLines) > 0 {
			hfLines = append(hfLines, "set_politics = {")
			hfLines = append(hfLines, spLines...)
			hfLines = append(hfLines, "}")
			hfLines = append(hfLines, "")
		}

		popLines := historyPopularityLines(hf)
		if len(popLines) > 0 {
			hfLines = append(hfLines, "set_popularities = {")
			hfLines = append(hfLines, popLines...)
			hfLines = append(hfLines, "}")
			hfLines = append(hfLines, "")
		}

		numPos := func(key string) bool {
			v := eGet(hf, key)
			if !pyTruthy(v) {
				return false
			}
			f, ok := numFloat(v)
			return ok && f > 0
		}
		numGEZero := func(key string) bool {
			v := eGet(hf, key)
			if v == nil {
				return false
			}
			f, ok := numFloat(v)
			return ok && f >= 0
		}
		if numPos("set_research_slots") {
			hfLines = append(hfLines,
				"set_research_slots = "+pyStr(eGet(hf, "set_research_slots")))
		}
		if numPos("set_convoys") {
			hfLines = append(hfLines,
				"set_convoys = "+pyStr(eGet(hf, "set_convoys")))
		}
		if numGEZero("set_stability") {
			hfLines = append(hfLines,
				"set_stability = "+pyG(eGet(hf, "set_stability")))
		}
		if numGEZero("set_war_support") {
			hfLines = append(hfLines,
				"set_war_support = "+pyG(eGet(hf, "set_war_support")))
		}
		if numPos("set_research_slots") || numPos("set_convoys") ||
			numGEZero("set_stability") || numGEZero("set_war_support") {
			hfLines = append(hfLines, "")
		}

		validIdeaSet := c.s.ExistenceSets["idea"]
		var gatedExtras []string
		for _, iv := range eList(hf, "add_ideas_extra") {
			if !pyTruthy(iv) {
				continue
			}
			ideaID := pyStr(iv)
			if len(validIdeaSet) == 0 || validIdeaSet[ideaID] {
				gatedExtras = append(gatedExtras, ideaID)
			} else {
				c.skip["history_file.add_ideas_extra["+ideaID+"]"]++
			}
		}
		if len(gatedExtras) > 0 {
			hfLines = append(hfLines, "add_ideas = {")
			for _, ideaID := range gatedExtras {
				hfLines = append(hfLines, "\t"+ideaID)
			}
			hfLines = append(hfLines, "}")
			hfLines = append(hfLines, "")
		}

		var removes []string
		for _, iv := range eList(hf, "remove_ideas") {
			if pyTruthy(iv) {
				removes = append(removes, pyStr(iv))
			}
		}
		if len(removes) > 0 {
			hfLines = append(hfLines, "remove_ideas = {")
			for _, ideaID := range removes {
				hfLines = append(hfLines, "\t"+ideaID)
			}
			hfLines = append(hfLines, "}")
			hfLines = append(hfLines, "")
		}

		validFocusSet := c.s.ExistenceSets["focus"]
		var gatedFocus []string
		for _, fv := range eList(hf, "complete_national_focus_ids") {
			if !pyTruthy(fv) {
				continue
			}
			fid := pyStr(fv)
			if len(validFocusSet) == 0 || validFocusSet[fid] {
				gatedFocus = append(gatedFocus, fid)
			} else {
				c.skip["history_file.complete_national_focus["+fid+"]"]++
			}
		}
		for _, fid := range gatedFocus {
			hfLines = append(hfLines, "complete_national_focus = "+fid)
		}
		if len(gatedFocus) > 0 {
			hfLines = append(hfLines, "")
		}

		var techIDs []string
		for _, tv := range eList(hf, "set_technology_ids") {
			if pyTruthy(tv) {
				techIDs = append(techIDs, pyStr(tv))
			}
		}
		if len(techIDs) > 0 {
			hfLines = append(hfLines, "set_technology = {")
			for _, tid := range techIDs {
				hfLines = append(hfLines, "\t"+tid+" = 1")
			}
			hfLines = append(hfLines, "}")
			hfLines = append(hfLines, "")
		}

		cosmetic := pyStrip(eStrOrEmpty(hf, "set_cosmetic_tag"))
		if cosmetic != "" {
			hfLines = append(hfLines, "set_cosmetic_tag = "+cosmetic)
			if _, seen := cosmeticTagsSeen[cosmetic]; !seen {
				cosmeticTagsSeen[cosmetic] = ctag
			}
			hfLines = append(hfLines, "")
		}

		oobEmitted := false
		for _, oobKey := range []string{"set_oob", "set_air_oob", "set_naval_oob"} {
			oobVal := pyStrip(eStrOrEmpty(hf, oobKey))
			if oobVal != "" {
				hfLines = append(hfLines, oobKey+" = \""+oobVal+"\"")
				oobEmitted = true
			}
		}
		if oobEmitted {
			hfLines = append(hfLines, "")
		}

		startupFlags := eList(hf, "startup_flags")
		for _, fv := range startupFlags {
			fs, isStr := fv.(string)
			if !isStr {
				continue
			}
			fid := pyStrip(fs)
			if fid == "" {
				continue
			}
			hfLines = append(hfLines, "set_country_flag = "+fid)
		}
		if len(startupFlags) > 0 {
			hfLines = append(hfLines, "")
		}

		diploEmitted := false
		for _, gv := range eList(hf, "give_guarantee_tags") {
			gs, isStr := gv.(string)
			if !isStr {
				continue
			}
			ggv := toUpper(pyStrip(gs))
			if ggv != "" {
				hfLines = append(hfLines, "give_guarantee = "+ggv)
				diploEmitted = true
			}
		}
		var atfTags []string
		for _, tv := range eList(hf, "add_to_faction_tags") {
			ts, isStr := tv.(string)
			if isStr && pyStrip(ts) != "" {
				atfTags = append(atfTags, ts)
			}
		}
		if len(atfTags) == 0 {
			if legacy := pyStrip(eStrOrEmpty(hf, "add_to_faction")); legacy != "" {
				atfTags = []string{legacy}
			}
		}
		for _, atf := range atfTags {
			atfV := toUpper(pyStrip(atf))
			if atfV != "" {
				hfLines = append(hfLines, "add_to_faction = "+atfV)
				diploEmitted = true
			}
		}
		for _, sv := range eList(hf, "set_autonomy") {
			sa, isMap := sv.(*OMap)
			if !isMap {
				continue
			}
			saT := toUpper(pyStrip(pyStrOrEmpty(eGet(sa, "target"))))
			saS := pyStrip(pyStrOrEmpty(eGet(sa, "autonomous_state")))
			if saT == "" || saS == "" {
				continue
			}
			hfLines = append(hfLines, "set_autonomy = {")
			hfLines = append(hfLines, "\ttarget = "+saT)
			hfLines = append(hfLines, "\tautonomous_state = "+saS)
			hfLines = append(hfLines, "}")
			diploEmitted = true
		}
		for _, dv := range eList(hf, "diplomatic_relation") {
			dr, isMap := dv.(*OMap)
			if !isMap {
				continue
			}
			drC := toUpper(pyStrip(pyStrOrEmpty(eGet(dr, "country"))))
			drR := pyStrip(pyStrOrEmpty(eGet(dr, "relation")))
			if drC == "" || drR == "" {
				continue
			}
			activeTok := "yes"
			if av, ok := dr.Get("active"); ok {
				if b, isB := av.(bool); isB && !b {
					activeTok = "no"
				}
			}
			hfLines = append(hfLines, "diplomatic_relation = {")
			hfLines = append(hfLines, "\tcountry = "+drC)
			hfLines = append(hfLines, "\trelation = "+drR)
			hfLines = append(hfLines, "\tactive = "+activeTok)
			hfLines = append(hfLines, "}")
			diploEmitted = true
		}
		if diploEmitted {
			hfLines = append(hfLines, "")
		}

		dated, dateNotices, err := renderHistoryDateBlocksTolerant(eList(hf, "date_blocks"), ctag)
		if err != nil {
			panic(err)
		}
		c.politics = append(c.politics, dateNotices...)
		if dated != "" {
			hfLines = append(hfLines, dated)
			hfLines = append(hfLines, "")
		}

		raw := pyStrip(eStrOrEmpty(hf, "raw_script"))
		if raw != "" {
			hfLines = append(hfLines, "# raw_script (user-defined long tail)")
			hfLines = append(hfLines, raw)
			hfLines = append(hfLines, "")
		}

		nonComment := false
		for _, line := range hfLines {
			s := pyStrip(line)
			if s != "" && !strings.HasPrefix(s, "#") {
				nonComment = true
				break
			}
		}
		if !nonComment {
			continue
		}
		bufSet(ctag, hfLines)
	}

	if c.s.ClByCtag != nil {
		for _, ctag := range c.s.ClByCtag.Keys() {
			if emittedHistoryFileCtags[ctag] {
				continue
			}
			clBlock := emitHistoryFileLeaderReplacementBlock(ctag, clFor(ctag))
			if len(clBlock) == 0 {
				continue
			}
			hfLines := append(headerLines(ctag),
				"# Family 15 minimal · only pre-day-0 leader replacement", "")
			hfLines = append(hfLines, clBlock...)
			foldRecruits(&hfLines, ctag, true, false)
			emittedHistoryFileCtags[ctag] = true
			bufSet(ctag, hfLines)
		}
	}

	for _, country := range c.s.Countries {
		ctag := toUpper(pyStrip(eStrOrEmpty(country, "tag")))
		if ctag == "" {
			continue
		}
		if c.t.VanillaTags[ctag] || pyTruthy(eGet(country, "auto_added")) {
			continue
		}
		ideology := pyStrip(eStrOrEmpty(country, "ideology"))
		if ideology == "" {
			continue
		}
		if emittedSetPoliticsCtags[ctag] {
			continue
		}
		rpGroup, rpOK := resolveRulingPartyGroup(ideology)
		if !rpOK {
			c.politics = append(c.politics, exportmsg.Message{Key: "politics.country_ideology_dropped", Args: exportmsg.Args{
				"subject": ctag, "value": ideology,
			}})
			continue
		}
		setPoliticsBlock := []string{
			"set_politics = {",
			"\truling_party = " + rpGroup,
			"}",
			"",
		}
		if emittedHistoryFileCtags[ctag] {
			existingBuf, hasBuf := buffers[ctag]
			if hasBuf {
				hasSetPolitics := false
				for _, line := range existingBuf {
					if strings.HasPrefix(pyLStrip(line), "set_politics") {
						hasSetPolitics = true
						break
					}
				}
				if !hasSetPolitics {
					if len(existingBuf) > 0 && existingBuf[len(existingBuf)-1] != "" {
						existingBuf = append(existingBuf, "")
					}
					existingBuf = append(existingBuf,
						"# Round-trip · ruling_party folded from "+
							"CountryModel.ideology (recruit file pre-existed)")
					existingBuf = append(existingBuf, setPoliticsBlock...)
					buffers[ctag] = existingBuf
					emittedSetPoliticsCtags[ctag] = true
				}
			}
			continue
		}
		hfLines := append(headerLines(ctag),
			"# Round-trip · ruling_party synthesized from CountryModel.ideology",
			"")
		hfLines = append(hfLines, setPoliticsBlock...)
		foldRecruits(&hfLines, ctag, false, true)
		emittedHistoryFileCtags[ctag] = true
		emittedSetPoliticsCtags[ctag] = true
		bufSet(ctag, hfLines)
	}

	if c.s.RecruitByCtag != nil {
		for _, ctag := range c.s.RecruitByCtag.Keys() {
			recruits := recruitsFor(ctag)
			if len(recruits) == 0 || recruitedCtags[ctag] {
				continue
			}
			if emittedHistoryFileCtags[ctag] {
				continue
			}
			hfLines := append(headerLines(ctag),
				"# P1 minimal · recruit_character for new characters", "")

			foldRecruits(&hfLines, ctag, false, true)
			emittedHistoryFileCtags[ctag] = true
			bufSet(ctag, hfLines)
		}
	}

	if len(c.s.OOBFiles) > 0 {
		knownTags := map[string]bool{}
		for _, country := range c.s.Countries {
			if t := toUpper(pyStrip(eStrOrEmpty(country, "tag"))); t != "" {
				knownTags[t] = true
			}
		}
		for _, hf := range c.s.HistoryFiles {
			if t := toUpper(pyStrip(eStrOrEmpty(hf, "country_tag"))); t != "" {
				knownTags[t] = true
			}
		}
		if c.s.ContentByTag != nil {
			for _, key := range c.s.ContentByTag.Keys() {
				if t := toUpper(pyStrip(key)); t != "" {
					knownTags[t] = true
				}
			}
		}

		refsByTag := map[string][]string{}
		for _, hf := range c.s.HistoryFiles {
			ctag := toUpper(pyStrip(eStrOrEmpty(hf, "country_tag")))
			if ctag == "" {
				continue
			}
			var stems []string
			for _, key := range []string{"set_oob", "set_air_oob", "set_naval_oob"} {
				if v := pyStrip(eStrOrEmpty(hf, key)); v != "" {
					stems = append(stems, OOBStem(v))
				}
			}
			stems = append(stems, parseOOBRefs(eStrOrEmpty(hf, "raw_script"))...)
			if len(stems) > 0 {
				refsByTag[ctag] = append(refsByTag[ctag], stems...)
			}
		}
		plan := planOOBWiring(c.s.OOBFiles, refsByTag, knownTags,
			buildProvinceOwner(c.s.States), c.s.Tag)
		c.oob = append(c.oob, plan.Warnings...)
		for _, action := range plan.Actions {
			note := "# Round-trip · set_oob synthesized — history/units/" +
				action.Stem + ".txt had no country-history reference, so the " +
				"game would never have loaded it (guides/division.md)"
			line := "set_oob = \"" + action.Stem + "\""
			if existing, hasBuf := buffers[action.Tag]; hasBuf {
				if len(existing) > 0 && existing[len(existing)-1] != "" {
					existing = append(existing, "")
				}
				buffers[action.Tag] = append(existing, note, line, "")
				continue
			}
			hfLines := append(headerLines(action.Tag), note, "")
			hfLines = append(hfLines, line, "")
			emittedHistoryFileCtags[action.Tag] = true
			bufSet(action.Tag, hfLines)
		}
	}

	for _, ctag := range bufferOrder {
		c.write(modDir+"history/countries/"+ctag+" - z"+modDirName+".txt",
			strings.Join(buffers[ctag], "\n")+"\n")
	}

	c.emitStates()

	c.emitCountrySections(cosmeticTagsSeen)
}

func historyPopularityLines(hf *OMap) []string {
	if values := eMap(hf, "set_popularities"); values != nil && len(values.Keys()) > 0 {
		keys := append([]string(nil), values.Keys()...)
		sort.Strings(keys)
		total := int64(0)
		lines := make([]string, 0, len(keys))
		for _, key := range keys {
			if !customRootIDRE.MatchString(key) {
				return nil
			}
			value, ok := intVal(eGet(values, key))
			if !ok || value < 0 || value > 100 {
				return nil
			}
			total += value
			lines = append(lines, "\t"+key+" = "+fmtInt(value))
		}
		if total == 100 {
			return lines
		}
		return nil
	}

	lines := []string{}
	for _, key := range []string{"democratic", "communism", "fascism", "neutrality"} {
		v := eGet(hf, "set_popularities_"+key)
		if f, ok := numFloat(v); ok && f > 0 {
			lines = append(lines, "\t"+key+" = "+pyStr(v))
		}
	}
	return lines
}
