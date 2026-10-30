package emitmisc

import (
	"fmt"
	"sort"
	"strings"
)

type Env struct {
	ModDir      string
	Tag         string
	IsSubmod    bool
	LangTriples []LangTriple
	Gate        func(string) string
	Project     *NS

	VanillaFocusIcons map[string]bool

	BuildFocusGfxEntries func(nodes []*NS) []FocusGfxEntry
	BuildTechGfxEntries  func(techs []*NS) []SpritePair
	BuildCatGfxEntries   func(cats []*NS) []SpritePair
	BuildDecGfxEntries   func(decs []*NS) []SpritePair

	PngToDDS func(sourceLabel string) string

	MakePlaceholderDDS func(frameName string) string

	SEPictureObs map[int64]string

	ResolveSEPicture func(obsKey string) bool

	RawFileScanChunks []string

	Writes              []WriteRec
	YmlWarnings         []string
	SkipCounter         *OD
	StructuredUnitPaths map[string]bool
}

type WriteRec struct {
	Path string
	Text string
}

type FocusGfxEntry struct {
	Name  string
	Path  string
	Shine bool
}

func (env *Env) write(path, text string) {
	env.Writes = append(env.Writes, WriteRec{path, text})
}

const bom = "\uFEFF"

func EmitTreesTechMilitary(env *Env, trees, techs, equipment, subUnits,
	divisionTemplates, divisionDeployments, airWingGroups, fleets, rawFiles []*NS) {

	modDir := env.ModDir
	tag := env.Tag

	if len(trees) > 0 {
		treesByFile := FocusTreeFilenames(trees)

		treeBuckets := BucketByCountry(trees, tag)
		focusGfxByCountry := map[string][]FocusGfxEntry{}
		for _, ctag := range treeBuckets.Keys {
			var nodesInCountry []*NS
			for _, tree := range treeBuckets.Get(ctag) {
				nodesInCountry = append(nodesInCountry, nsList(tree.List("nodes"))...)
			}
			focusGfxByCountry[ctag] = env.BuildFocusGfxEntries(nodesInCountry)
		}
		registeredFocusIcons := map[string]bool{}
		for _, entries := range focusGfxByCountry {
			for _, e := range entries {
				registeredFocusIcons[e.Name] = true
			}
		}
		var focusValidIcons map[string]bool
		if env.IsSubmod {
			focusValidIcons = nil
		} else {
			focusValidIcons = map[string]bool{}
			for k := range env.VanillaFocusIcons {
				focusValidIcons[k] = true
			}
			for k := range registeredFocusIcons {
				focusValidIcons[k] = true
			}
		}

		for _, filename := range treesByFile.Keys {
			grouped := treesByFile.Get(filename)
			sort.SliceStable(grouped, func(i, j int) bool {
				a, b := grouped[i], grouped[j]
				an := a.Str("tree_kind") != "primary"
				bn := b.Str("tree_kind") != "primary"
				if an != bn {
					return !an
				}
				return a.Str("tree_id") < b.Str("tree_id")
			})
			parts := make([]string, 0, len(grouped))
			for _, t := range grouped {
				parts = append(parts, env.Gate(GenerateFocusTreeScript(t, focusValidIcons)))
			}
			script := pyRStrip(strings.Join(parts, "\n\n")) + "\n"
			env.write(modDir+"common/national_focus/"+tag+"_"+filename, script)
		}

		for _, ctag := range treeBuckets.Keys {
			treesInCountry := treeBuckets.Get(ctag)
			focusLocByLang := newLocByLang()
			for _, lt := range env.LangTriples {
				focusLoc := GenerateFocusLocalisation(treesInCountry, lt.Code)
				focusLocByLang.set(lt.Code, focusLoc)
				env.write(modDir+"localisation/"+lt.Subdir+"/"+ctag+"_focus_"+lt.Suffix+".yml",
					bom+focusLoc)
			}
			env.YmlWarnings = append(env.YmlWarnings,
				assertYmlSymmetricMulti(focusLocByLang, "focus/"+ctag)...)
		}

		for _, ctag := range treeBuckets.Keys {
			focusGfxEntries := focusGfxByCountry[ctag]
			if len(focusGfxEntries) > 0 {
				spritePairs := make([]SpritePair, 0, len(focusGfxEntries))
				for _, e := range focusGfxEntries {
					spritePairs = append(spritePairs, SpritePair{Name: e.Name, Path: e.Path})
				}
				env.write(modDir+"interface/"+ctag+"_focus.gfx", EmitSpriteBlock(spritePairs))
				var shinePairs []SpritePair
				for _, e := range focusGfxEntries {
					if e.Shine {
						shinePairs = append(shinePairs, SpritePair{Name: e.Name, Path: e.Path})
					}
				}
				if len(shinePairs) > 0 {
					env.write(modDir+"interface/"+ctag+"_focus_shine.gfx",
						EmitFocusShineBlock(shinePairs))
				}
				for _, e := range focusGfxEntries {
					ddsBytes := env.PngToDDS(e.Name)
					if ddsBytes != "" {
						env.write(modDir+e.Path, ddsBytes)
					}
				}
			}
		}
	}

	if len(techs) > 0 {
		techGroups := GroupBySourceFile(techs, func(t *NS) string {
			return "common/technologies/" + tag + "_" + t.Str("tech_line") + ".txt"
		}, "common/technologies")
		for _, relPath := range techGroups.Keys {
			groupTechs := techGroups.Get(relPath)
			line := groupTechs[0].Str("tech_line")
			script := GenerateTechLineScript(groupTechs, line, env.Project, techs)
			env.write(modDir+relPath, script)
		}

		techLocByLang := newLocByLang()
		for _, lt := range env.LangTriples {
			techLoc := GenerateTechLocalisation(techs, lt.Code)
			techLocByLang.set(lt.Code, techLoc)
			env.write(modDir+"localisation/"+lt.Subdir+"/"+tag+"_technology_"+lt.Suffix+".yml",
				bom+techLoc)
		}
		env.YmlWarnings = append(env.YmlWarnings,
			assertYmlSymmetricMulti(techLocByLang, "technology/"+tag)...)

		techGfxEntries := env.BuildTechGfxEntries(techs)
		if len(techGfxEntries) > 0 {
			env.write(modDir+"interface/"+tag+"_technologies.gfx", EmitSpriteBlock(techGfxEntries))
			for _, e := range techGfxEntries {
				ddsBytes := env.PngToDDS(e.Name)
				if ddsBytes != "" {
					env.write(modDir+e.Path, ddsBytes)
				}
			}
		}
	}

	if len(equipment) > 0 || len(subUnits) > 0 {
		EmitUnits(env, equipment, subUnits, rawFiles)
	}
	if len(equipment) > 0 {
		eqLocByLang := newLocByLang()
		for _, lt := range env.LangTriples {
			eqLoc := GenerateEquipmentLocalisation(equipment, lt.Code)
			eqLocByLang.set(lt.Code, eqLoc)
			env.write(modDir+"localisation/"+lt.Subdir+"/"+tag+"_equipment_"+lt.Suffix+".yml",
				bom+eqLoc)
		}
		env.YmlWarnings = append(env.YmlWarnings,
			assertYmlSymmetricMulti(eqLocByLang, "equipment/"+tag)...)
	}
	if len(subUnits) > 0 {
		suLocByLang := newLocByLang()
		for _, lt := range env.LangTriples {
			suLoc := GenerateSubUnitLocalisation(subUnits, lt.Code)
			suLocByLang.set(lt.Code, suLoc)
			env.write(modDir+"localisation/"+lt.Subdir+"/"+tag+"_sub_units_"+lt.Suffix+".yml",
				bom+suLoc)
		}
		env.YmlWarnings = append(env.YmlWarnings,
			assertYmlSymmetricMulti(suLocByLang, "sub_units/"+tag)...)
	}

	if len(divisionTemplates) > 0 || len(divisionDeployments) > 0 ||
		len(airWingGroups) > 0 || len(fleets) > 0 {
		tplsByOob := newFileGroups()
		for _, tpl := range divisionTemplates {
			tplsByOob.append(OOBDestination(tpl), tpl)
		}
		depsByOob := newFileGroups()
		for _, dep := range divisionDeployments {
			depsByOob.append(OOBDestination(dep), dep)
		}
		airByOob := newFileGroups()
		for _, group := range airWingGroups {
			airByOob.append(OOBDestination(group), group)
		}
		fleetsByOob := newFileGroups()
		for _, fleet := range fleets {
			fleetsByOob.append(OOBDestination(fleet), fleet)
		}
		pathSet := map[string]bool{}
		var allPaths []string
		for _, g := range []*FileGroups{tplsByOob, depsByOob, airByOob, fleetsByOob} {
			for _, p := range g.Keys {
				if !pathSet[p] {
					pathSet[p] = true
					allPaths = append(allPaths, p)
				}
			}
		}
		overridePaths := AuthorRawOverridePaths(rawFiles)
		for _, relativePath := range allPaths {
			tpls := tplsByOob.Get(relativePath)
			deps := depsByOob.Get(relativePath)
			airGroups := airByOob.Get(relativePath)
			oobFleets := fleetsByOob.Get(relativePath)
			tail := ""
			for _, t := range tpls {
				if pyStrip(t.Str("oob_units_tail")) != "" {
					tail = t.Str("oob_units_tail")
					break
				}
			}
			var parts []string
			templateScript := pyStrip(GenerateOOBFileScript(tpls, ""))
			if templateScript != "" {
				parts = append(parts, templateScript)
			}

			airRaw, airNotices := GenerateAirWingsBlockTolerant(airGroups, 0)
			env.YmlWarnings = append(env.YmlWarnings, airNotices...)
			if airScript := pyStrip(airRaw); airScript != "" {
				parts = append(parts, airScript)
			}
			unitsScript, unitNotices := MergeUnitsBlocksTolerant(deps, oobFleets)
			env.YmlWarnings = append(env.YmlWarnings, unitNotices...)
			if unitsScript != "" {
				parts = append(parts, unitsScript)
			}

			if tail = pyStrip(strings.TrimPrefix(pyStrip(tail), "\ufeff")); tail != "" {
				parts = append(parts, tail)
			}
			if !overridePaths[relativePath] {
				env.write(modDir+relativePath, strings.Join(parts, "\n\n")+"\n")
				env.StructuredUnitPaths[relativePath] = true
			}
		}
	}
}

func EmitDecisionsSuperevents(env *Env, decisions, decisionCategories, superEvents,
	events, trees []*NS) {

	modDir := env.ModDir
	tag := env.Tag

	if len(decisions) > 0 {
		catBuckets := BucketByCountry(decisionCategories, tag)
		catToCtag := map[int64]string{}
		for _, ctag := range catBuckets.Keys {
			for _, cat := range catBuckets.Get(ctag) {
				catToCtag[cat.Int("id", 0)] = ctag
			}
		}
		primaryUpper := pyUpper(tag)
		if primaryUpper == "" {
			primaryUpper = "UNK"
		}
		decBuckets := newBuckets()
		for _, k := range catBuckets.Keys {
			decBuckets.ensure(k)
		}
		decBuckets.ensure(primaryUpper)
		for _, d := range decisions {
			ctag, ok := catToCtag[d.Int("category_id", 0)]
			if !ok {
				ctag = pyUpper(d.Str("country_tag"))
				if ctag == "" {
					ctag = primaryUpper
				}
			}
			decBuckets.append(ctag, d)
		}

		allCtagSet := map[string]bool{}
		var allCtags []string
		for _, k := range catBuckets.Keys {
			if !allCtagSet[k] {
				allCtagSet[k] = true
				allCtags = append(allCtags, k)
			}
		}
		for _, k := range decBuckets.Keys {
			if !allCtagSet[k] {
				allCtagSet[k] = true
				allCtags = append(allCtags, k)
			}
		}
		for _, ctag := range allCtags {
			catsInCountry := catBuckets.Get(ctag)
			decsInCountry := decBuckets.Get(ctag)

			hiddenCatIDs := map[int64]bool{}
			var visibleCats []*NS
			for _, c := range catsInCountry {
				vis := c.StrOr("visibility", "visible")
				if vis == "hidden" {
					hiddenCatIDs[c.Int("id", 0)] = true
				} else {
					visibleCats = append(visibleCats, c)
				}
			}
			var visibleDecs []*NS
			for _, d := range decsInCountry {
				if !hiddenCatIDs[d.Int("category_id", 0)] {
					visibleDecs = append(visibleDecs, d)
				}
			}

			catGfxEntries := env.BuildCatGfxEntries(visibleCats)
			shippedCatSprites := map[string]bool{}
			for _, e := range catGfxEntries {
				shippedCatSprites[e.Name] = true
			}
			catIconGates := map[int64]bool{}
			for _, c := range visibleCats {
				custom := DecisionCategoryIconSpriteName(c.Str("category_id"))
				catIconGates[c.Int("id", 0)] = c.Str("icon") != custom || shippedCatSprites[custom]
			}

			decGfxEntries := env.BuildDecGfxEntries(visibleDecs)
			shippedDecSprites := map[string]bool{}
			for _, e := range decGfxEntries {
				shippedDecSprites[e.Name] = true
			}
			decIconGates := map[int64]bool{}
			for _, d := range visibleDecs {
				custom := DecisionIconSpriteName(d.Str("decision_id"))
				decIconGates[d.Int("id", 0)] = d.Str("icon") != custom || shippedDecSprites[custom]
			}

			if len(visibleCats) > 0 {
				catScript := env.Gate(GenerateDecisionCategoriesScript(visibleCats, catIconGates))
				env.write(modDir+"common/decisions/categories/"+tag+"_"+ctag+"_categories.txt",
					catScript)
			}
			if len(visibleDecs) > 0 {
				decGroups := GroupBySourceFile(visibleDecs, func(*NS) string {
					return "common/decisions/" + tag + "_" + ctag + ".txt"
				}, "common/decisions")
				for _, relPath := range decGroups.Keys {
					fileDecs := decGroups.Get(relPath)
					decScript := env.Gate(GenerateDecisionsPreview(visibleCats, fileDecs, decIconGates))
					env.write(modDir+relPath, decScript)
				}
			}

			decLocByLang := newLocByLang()
			for _, lt := range env.LangTriples {
				decLoc := GenerateDecisionsLocalisation(catsInCountry, decsInCountry, lt.Code)
				decLocByLang.set(lt.Code, decLoc)
				env.write(modDir+"localisation/"+lt.Subdir+"/"+ctag+"_decisions_"+lt.Suffix+".yml",
					bom+decLoc)
			}
			env.YmlWarnings = append(env.YmlWarnings,
				assertYmlSymmetricMulti(decLocByLang, "decisions/"+ctag)...)

			allGfxEntries := append(append([]SpritePair{}, catGfxEntries...), decGfxEntries...)
			if len(allGfxEntries) > 0 {
				env.write(modDir+"interface/"+ctag+"_decisions.gfx", EmitSpriteBlock(allGfxEntries))
				for _, e := range allGfxEntries {
					ddsBytes := env.PngToDDS(e.Name)
					if ddsBytes != "" {
						env.write(modDir+e.Path, ddsBytes)
					}
				}
			}
		}
	}

	if len(superEvents) > 0 {
		var eventScanChunks []string
		for _, ev := range events {
			for _, attr := range []string{"immediate", "after", "trigger", "mean_time_to_happen"} {
				val := ev.Str(attr)
				if val != "" {
					eventScanChunks = append(eventScanChunks, val)
				}
			}
			for _, optAny := range ev.List("options") {
				opt, ok := asOD(optAny)
				if !ok {
					continue
				}
				for _, k := range []string{"effects", "hidden_effects", "trigger"} {
					v, _ := opt.Get(k)
					if Truthy(v) {
						eventScanChunks = append(eventScanChunks, PyStr(v))
					}
				}
			}
		}
		for _, d := range decisions {
			for _, attr := range []string{"complete_effect", "remove_effect", "cancel_effect",
				"timeout_effect", "activation", "available", "visible"} {
				val := d.Str(attr)
				if val != "" {
					eventScanChunks = append(eventScanChunks, val)
				}
			}
		}
		for _, tr := range trees {
			for _, node := range nsList(tr.List("nodes")) {
				for _, attr := range []string{"completion_reward", "select_effect", "bypass", "available"} {
					val := node.Str(attr)
					if val != "" {
						eventScanChunks = append(eventScanChunks, val)
					}
				}
			}
		}
		for _, seOther := range superEvents {
			for _, optAny := range seOther.List("options") {
				if opt, ok := asOD(optAny); ok {
					eff, _ := opt.Get("effect")
					if Truthy(eff) {
						eventScanChunks = append(eventScanChunks, PyStr(eff))
					}
				}
			}
		}

		eventScanChunks = append(eventScanChunks, env.RawFileScanChunks...)
		scanCorpus := strings.Join(eventScanChunks, "\n")

		var liveSuperEvents []*NS
		for _, se := range superEvents {
			ctagSe := pyUpper(se.Str("country_tag"))
			if ctagSe == "" {
				ctagSe = pyUpper(tag)
			}
			if ctagSe == "" {
				ctagSe = "UNK"
			}
			seID := pyLower(ctagSe) + "_se." + PyStr(se.Int("number", 0))
			if SeIsAutoFire(se) {
				liveSuperEvents = append(liveSuperEvents, se)
				continue
			}
			if ScanForEventRef(scanCorpus, seID) {
				liveSuperEvents = append(liveSuperEvents, se)
				continue
			}
			key := "super_event_dormant:" + seID
			cur, _ := env.SkipCounter.Get(key)
			curN := int64(0)
			if cur != nil {
				curN = asInt64(cur)
			}
			env.SkipCounter.Set(key, curN+1)
		}
		superEvents = liveSuperEvents
	}

	if len(superEvents) > 0 {

		for _, frameName := range sePlaceholderFrameNames {
			dds := env.MakePlaceholderDDS(frameName)
			if dds != "" {
				env.write(modDir+"gfx/tvbigevent/"+frameName, dds)
			}
		}

		seBuckets := BucketByCountry(superEvents, tag)
		for _, ctag := range seBuckets.Keys {
			sesInCountry := seBuckets.Get(ctag)

			evLines := []string{"add_namespace = " + pyLower(ctag) + "_se", ""}
			for _, se := range sesInCountry {
				seID := pyLower(ctag) + "_se." + PyStr(se.Int("number", 0))
				evLines = append(evLines, "country_event = {")
				evLines = append(evLines, "\tid = "+seID)
				evLines = append(evLines, "\ttitle = "+seID+".t")
				evLines = append(evLines, "\tdesc = "+seID+".d")
				evLines = append(evLines, "\thidden = yes")
				evLines = append(evLines, "\tis_triggered_only = yes")
				evLines = append(evLines, "\tfire_only_once = yes")
				evLines = append(evLines, "")
				evLines = append(evLines, "\timmediate = {")
				evLines = append(evLines, fmt.Sprintf("\t\tset_variable = { global.%s_se_display = %04d }",
					pyUpper(ctag), se.Int("number", 0)))
				if Truthy(se.A("song_token")) {
					evLines = append(evLines, "\t\tplay_song = \""+se.Str("song_token")+"\"")
				}
				evLines = append(evLines, "\t}")
				evLines = append(evLines, "")
				seOptions := se.List("options")
				if len(seOptions) == 0 {
					evLines = append(evLines, "\toption = {")
					evLines = append(evLines, "\t\tname = "+seID+".a")
					evLines = append(evLines, "\t\tai_chance = { base = 1 }")
					evLines = append(evLines, "\t}")
				} else {
					for idx, optAny := range seOptions {
						opt, _ := asOD(optAny)
						optKey := fmt.Sprintf("%s.opt%d", seID, idx+1)
						evLines = append(evLines, "\toption = {")
						evLines = append(evLines, "\t\tname = "+optKey)
						effect := pyStrip(PyStr(orAny(opt.GetDef("effect", nil), "")))
						if effect != "" {
							gatedEffect := env.Gate(effect)
							if pyStrip(gatedEffect) != "" {
								for _, line := range strings.Split(gatedEffect, "\n") {
									if pyStrip(line) != "" {
										evLines = append(evLines, "\t\t"+line)
									} else {
										evLines = append(evLines, "")
									}
								}
							}
						}
						aiChance := opt.GetDef("ai_chance", nil)
						aiVal := 1.0
						if aiChance != nil {
							if f, ok := asFloat(aiChance); ok {
								aiVal = f
							}
						}
						aiStr := ""
						if isIntegralFloat(aiVal) {
							aiStr = PyStr(pyIntTrunc(aiVal))
						} else {
							aiStr = fmtFixed2(aiVal)
						}
						evLines = append(evLines, "\t\tai_chance = { base = "+aiStr+" }")
						evLines = append(evLines, "\t}")
					}
				}
				evLines = append(evLines, "}")
				evLines = append(evLines, "")
			}
			env.write(modDir+"events/"+ctag+"_se.txt", strings.Join(evLines, "\n"))

			for _, lt := range env.LangTriples {
				lang := lt.Code
				ymlLines := []string{lt.Suffix + ":"}
				for _, se := range sesInCountry {
					n := fmt.Sprintf("%04d", se.Int("number", 0))
					seID := pyLower(ctag) + "_se." + PyStr(se.Int("number", 0))
					ymlLines = append(ymlLines, " "+seID+`.a:0 "(continue)"`)
					for idx, optAny := range se.List("options") {
						opt, _ := asOD(optAny)
						optKey := fmt.Sprintf("%s.opt%d", seID, idx+1)
						optEn := resolveJSONLocalisationOD(
							opt, "en", "display_name_locs", "name_en", "name_zh",
						)
						optName := resolveJSONLocalisationOD(
							opt, lang, "display_name_locs", "name_en", "name_zh",
						)
						optName = orStr(optName, orStr(optEn, "(continue)"))
						ymlLines = append(ymlLines,
							" "+optKey+`:0 "`+YmlEscapeValue(optName)+`"`)
					}
					titleEnResolved := resolveDisplayName(se, "en", "title_en", "title_zh")
					titleLangResolved := titleEnResolved
					if lang != "en" {
						titleLangResolved = resolveDisplayName(se, lang, "title_en", "title_zh")
					}
					title := titleEnResolved
					if lang != "en" {
						title = orStr(titleLangResolved, titleEnResolved)
					}
					descEnVal := resolveDescription(se, "en")
					desc := descEnVal
					if lang != "en" {
						desc = orStr(resolveDescription(se, lang), descEnVal)
					}
					quoteEn := resolveJSONLocalisationNS(
						se, "en", "quote_locs", "quote_en", "quote_zh",
					)
					quote := orStr(resolveJSONLocalisationNS(
						se, lang, "quote_locs", "quote_en", "quote_zh",
					), quoteEn)
					btnEn := resolveJSONLocalisationNS(
						se, "en", "button_locs", "btn_en", "btn_zh",
					)
					btn := orStr(resolveJSONLocalisationNS(
						se, lang, "button_locs", "btn_en", "btn_zh",
					), btnEn)
					ymlLines = append(ymlLines,
						" "+ctag+"_SE_title_"+n+`:0 "`+YmlEscapeValue(title)+`"`)
					ymlLines = append(ymlLines,
						" "+ctag+"_SE_desc_"+n+`:0 "`+YmlEscapeValue(desc)+`"`)
					ymlLines = append(ymlLines,
						" "+ctag+"_SE_quote_"+n+`:0 "`+YmlEscapeValue(quote)+`"`)
					ymlLines = append(ymlLines,
						" "+ctag+"_SE_btn_"+n+`:0 "`+orStr(YmlEscapeValue(btn), "(continue)")+`"`)
					ymlLines = append(ymlLines,
						" "+seID+`.t:0 "`+orStr(YmlEscapeValue(title), "(superevent "+n+")")+`"`)
					ymlLines = append(ymlLines,
						" "+seID+`.d:0 "`+YmlEscapeValue(desc)+`"`)
				}
				ymlLines = append(ymlLines, " "+ctag+`_SE_title_missing:0 "(superevent placeholder)"`)
				ymlLines = append(ymlLines, " "+ctag+`_SE_desc_missing:0 ""`)
				ymlLines = append(ymlLines, " "+ctag+`_SE_quote_missing:0 ""`)
				ymlLines = append(ymlLines, " "+ctag+`_SE_btn_missing:0 "(continue)"`)
				ymlLines = append(ymlLines, "")
				env.write(modDir+"localisation/"+lt.Subdir+"/"+ctag+"_se_"+lt.Suffix+".yml",
					bom+strings.Join(ymlLines, "\n"))
			}

			var gfxEntries []SpritePair
			picturedSeNumbers := map[string]bool{}
			for _, se := range sesInCountry {
				n := fmt.Sprintf("%04d", se.Int("number", 0))
				obsKey := env.SEPictureObs[se.Int("id", 0)]
				if !env.ResolveSEPicture(obsKey) {
					continue
				}
				ddsZipPath := "gfx/tvbigevent/pics/" + ctag + "_SE_" + n + ".dds"
				spriteName := "GFX_" + ctag + "_SE_" + n
				ddsBytes := env.PngToDDS(spriteName)
				if ddsBytes != "" {
					env.write(modDir+ddsZipPath, ddsBytes)
					gfxEntries = append(gfxEntries, SpritePair{Name: spriteName, Path: ddsZipPath})
					picturedSeNumbers[n] = true
				}
			}
			if len(gfxEntries) > 0 {
				env.write(modDir+"interface/"+ctag+"_se_pictures.gfx", EmitSpriteBlock(gfxEntries))
			}

			for _, fp := range []struct{ field, keyPrefix string }{
				{"getpicture", "GFX_" + ctag + "_SE"},
				{"title", ctag + "_SE_title"},
				{"desc", ctag + "_SE_desc"},
				{"quote", ctag + "_SE_quote"},
				{"btn", ctag + "_SE_btn"},
			} {
				routerLines := []string{
					"# Auto-generated by Hagane Works export",
					"defined_text = {",
					"\tname = " + ctag + "_SE_" + fp.field,
					"",
				}
				for _, se := range sesInCountry {
					n := fmt.Sprintf("%04d", se.Int("number", 0))
					if fp.field == "getpicture" && !picturedSeNumbers[n] {
						continue
					}
					routerLines = append(routerLines,
						"\ttext = { trigger = { check_variable = { global."+
							pyUpper(ctag)+"_se_display = "+n+" } } localization_key = "+
							fp.keyPrefix+"_"+n+" }")
				}
				routerLines = append(routerLines, "")
				routerLines = append(routerLines, "\t# Fallback for unmatched "+pyUpper(ctag)+"_se_display values")
				routerLines = append(routerLines, "\ttext = {")
				routerLines = append(routerLines, "\t\ttrigger = { }")
				fallbackKey := ctag + "_SE_" + fp.field + "_missing"
				if fp.field == "getpicture" {
					fallbackKey = "GFX_" + ctag + "_SE_pic_entry"
				}
				routerLines = append(routerLines, "\t\tlocalization_key = "+fallbackKey)
				routerLines = append(routerLines, "\t}")
				routerLines = append(routerLines, "}")
				env.write(modDir+"common/scripted_localisation/"+ctag+"_SE_"+fp.field+".txt",
					strings.Join(routerLines, "\n")+"\n")
			}

			EmitSuperEventBoilerplate(env.write, modDir, ctag)
		}
	}
}

func fmtFixed2(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
