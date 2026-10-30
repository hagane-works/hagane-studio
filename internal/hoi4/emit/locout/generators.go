package locout

import "strings"

func attrKey(o *Obj, name string) string { return PyStr(o.Attr(name)) }

func GenerateTraitOverlayLocalisation(namespace, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	var title, desc string
	if lang == "zh" {

		title = "特质覆盖(隐藏)"
		desc = "游戏启动时内部触发," +
			"用于将自定义领袖、将领、" +
			"科学家特质赋予现有原版角色," +
			"无需替换角色文件。"
	} else {
		title = "Trait Overlay (hidden)"
		desc = "Internally fired at game start to apply custom country leader, " +
			"unit leader, and scientist traits to existing vanilla characters " +
			"without replacing the character file."
	}
	lines = append(lines, YmlLine(namespace+".1.t", Text{S: title}))
	lines = append(lines, YmlLine(namespace+".1.d", Text{S: desc}))
	return strings.Join(lines, "\n")
}

func GenerateCharacterLocalisation(characters []*Obj, countryTag, lang string) string {
	_ = countryTag
	lines := []string{langHeader(lang) + ":"}
	for _, char := range characters {
		tagID := char.Attr("tag")
		nameEn := resolveMeaningfulDisplayName(char, "en", tagID, legacyName, "name", true)
		if nameEn.Truthy() {
			var nameVal Text
			if lang == "en" {
				nameVal = nameEn
			} else {
				nameLangResolved := resolveMeaningfulDisplayName(char, lang, tagID, legacyName, "name", true)
				nameVal = orText(nameLangResolved, nameEn)
			}
			lines = append(lines, YmlLine(CharacterLocKey(char), nameVal))
		}
		descAny := resolveDescriptionAnyLang(char, legacyDesc)
		var descVal Text
		if descAny != "" {
			descEn := resolveDescription(char, "en", legacyDesc)
			if descEn == "" {
				descEn = descAny
			}
			if lang == "en" {
				descVal = Text{S: descEn}
			} else {
				dl := resolveDescription(char, lang, legacyDesc)
				if dl == "" {
					dl = descEn
				}
				descVal = Text{S: dl}
			}
		} else {
			if lang == "en" || !nameEn.Truthy() {
				descVal = nameEn
			} else {
				descVal = orText(
					resolveMeaningfulDisplayName(char, lang, tagID, legacyName, "name", true),
					nameEn,
				)
			}
		}
		if descVal.Truthy() {
			lines = append(lines, YmlLine(CharacterLocKey(char)+"_desc", descVal))
		}
	}
	return strings.Join(lines, "\n")
}

func DetectCharacterNameFallback(characters []*Obj) []string {
	warnings := []string{}
	for _, char := range characters {
		if PyTruthy(char.Attr("replaces_vanilla_id")) {
			continue
		}
		tagID := char.Attr("tag")
		if !HasMeaningfulDisplayName(char, tagID) {
			country := char.Attr("country_tag")
			countryStr := "?"
			if PyTruthy(country) {
				countryStr = PyStr(country)
			}
			warnings = append(warnings,
				"character "+pyReprValue(tagID)+" (country "+countryStr+") · "+
					"no display name — the export skips the name loc line, so HOI4 "+
					"shows the raw tag verbatim in-game. Fill the English name.")
		}
	}
	return warnings
}

func pyReprValue(v Value) string {
	if v.Kind == KStr {
		return pyStrRepr(v.Str)
	}
	return PyStr(v)
}

func GenerateFocusLocalisation(trees []*Obj, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, tree := range trees {
		nodesVal := tree.Attr("nodes")
		if nodesVal.Kind != KArr {
			continue
		}
		for _, nodeV := range nodesVal.Arr {
			node := nodeV.Obj
			focusID := node.Attr("focus_id")
			key := PyStr(focusID)
			nameEn := resolveMeaningfulDisplayName(node, "en", focusID, legacyName, "name", true)
			if nameEn.Truthy() {
				var nameVal Text
				if lang == "en" {
					nameVal = nameEn
				} else {
					nameLang := resolveMeaningfulDisplayName(node, lang, focusID, legacyName, "name", true)
					nameVal = orText(nameLang, nameEn)
				}
				lines = append(lines, YmlLine(key, nameVal))
			}
			descEn := resolveDescription(node, "en", legacyDesc)
			descLang := descEn
			if lang != "en" {
				descLang = resolveDescription(node, lang, legacyDesc)
			}
			if descEn != "" || descLang != "" {
				descVal := descEn
				if lang != "en" {
					if descLang != "" {
						descVal = descLang
					}
				}
				lines = append(lines, YmlLine(key+"_desc", Text{S: descVal}))
			}
			ttEn := resolveEffectTooltip(node, "en")
			ttLang := ttEn
			if lang != "en" {
				ttLang = resolveEffectTooltip(node, lang)
			}
			if ttEn != "" || ttLang != "" {
				ttVal := ttEn
				if lang != "en" {
					if ttLang != "" {
						ttVal = ttLang
					}
				}
				lines = append(lines, YmlLine(key+"_et_tt", Text{S: ttVal}))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func idNameDescEmit(lines []string, entity *Obj, idAttr, lang, literalAttr string) []string {
	idVal := entity.Attr(idAttr)
	key := PyStr(idVal)
	nameEn := resolveMeaningfulDisplayName(entity, "en", idVal, legacyName, literalAttr, true)
	if nameEn.Truthy() {
		var nameVal Text
		if lang == "en" {
			nameVal = nameEn
		} else {
			nameLang := resolveMeaningfulDisplayName(entity, lang, idVal, legacyName, literalAttr, true)
			nameVal = orText(nameLang, nameEn)
		}
		lines = append(lines, YmlLine(key, nameVal))
	}
	descEn := resolveDescription(entity, "en", legacyDesc)
	descLang := descEn
	if lang != "en" {
		descLang = resolveDescription(entity, lang, legacyDesc)
	}
	if descEn != "" || descLang != "" {
		descVal := descEn
		if lang != "en" && descLang != "" {
			descVal = descLang
		}
		lines = append(lines, YmlLine(key+"_desc", Text{S: descVal}))
	}
	return lines
}

func GenerateTechLocalisation(techs []*Obj, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, tech := range techs {
		lines = idNameDescEmit(lines, tech, "tech_id", lang, "name")
	}
	return strings.Join(lines, "\n")
}

func GenerateIdeasLocalisation(ideas []*Obj, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, idea := range ideas {
		lines = idNameDescEmit(lines, idea, "idea_id", lang, "name")
	}
	return strings.Join(lines, "\n")
}

func GenerateCustomTraitsLocalisation(traits []*Obj, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, t := range traits {
		lines = idNameDescEmit(lines, t, "trait_id", lang, "")
	}
	return strings.Join(lines, "\n")
}

func GenerateEventsLocalisation(namespaces, events []*Obj, lang string) string {
	_ = namespaces
	lines := []string{langHeader(lang) + ":"}
	for _, ev := range events {
		eventID := PyStr(ev.Attr("event_id"))

		titleEn := resolveDisplayName(ev, "en", legacyTitle)
		titleLang := titleEn
		if lang != "en" {
			titleLang = resolveDisplayName(ev, lang, legacyTitle)
		}
		if titleEn != "" || titleLang != "" {
			val := titleEn
			if lang != "en" && titleLang != "" {
				val = titleLang
			}
			lines = append(lines, YmlLine(eventID+".t", Text{S: val}))
		}

		descEn := resolveDescription(ev, "en", legacyDesc)
		descLang := descEn
		if lang != "en" {
			descLang = resolveDescription(ev, lang, legacyDesc)
		}
		if descEn != "" || descLang != "" {
			val := descEn
			if lang != "en" && descLang != "" {
				val = descLang
			}
			lines = append(lines, YmlLine(eventID+".d", Text{S: val}))
		}

		optsVal := ev.Attr("options")
		var opts []Value
		if PyTruthy(optsVal) && optsVal.Kind == KArr {
			opts = optsVal.Arr
		}
		for optIdx, opt := range opts {
			if opt.Kind != KObj {
				continue
			}
			optKey, _ := EventOptionLocKey(opt, eventID, optIdx)
			displayText := pyStrip(strOrEmpty(opt.Obj.Attr("display_text")))
			nameEn := resolveJSONLocalisation(
				opt.Obj, "en", "display_name_locs", legacyName,
			)
			if nameEn == "" {
				nameEn = displayText
			}
			nameLang := resolveJSONLocalisation(
				opt.Obj, lang, "display_name_locs", legacyName,
			)
			val := nameEn
			if lang != "en" && nameLang != "" {
				val = nameLang
			}
			nameAny := firstJSONLocalisation(opt.Obj, "display_name_locs", legacyName)
			if nameAny == "" {
				nameAny = displayText
			}

			if nameAny != "" {
				lines = append(lines, YmlLine(optKey, Text{S: val}))
			}
			ttLegacy := legacyAttrs{"effect_tooltip_en", "effect_tooltip_zh"}
			ttEn := resolveJSONLocalisation(opt.Obj, "en", "effect_tooltip_locs", ttLegacy)
			ttLang := resolveJSONLocalisation(opt.Obj, lang, "effect_tooltip_locs", ttLegacy)
			ttAny := firstJSONLocalisation(opt.Obj, "effect_tooltip_locs", ttLegacy)
			if ttAny != "" {
				ttVal := ttEn
				if lang != "en" {
					ttVal = ttLang
					if ttVal == "" {
						ttVal = ttEn
					}
					if ttVal == "" {
						ttVal = ttAny
					}
				}
				lines = append(lines, YmlLine(optKey+"_tt", Text{S: ttVal}))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func idOnlyNameEmit(lines []string, entity *Obj, idAttr, lang string) []string {
	idVal := entity.Attr(idAttr)
	if !PyTruthy(idVal) {
		return lines
	}
	nameEn := resolveMeaningfulDisplayName(entity, "en", idVal, legacyName, "", true)
	if nameEn.Truthy() {
		var nameVal Text
		if lang == "en" {
			nameVal = nameEn
		} else {
			nameLang := resolveMeaningfulDisplayName(entity, lang, idVal, legacyName, "", true)
			nameVal = orText(nameLang, nameEn)
		}
		lines = append(lines, YmlLine(PyStr(idVal), nameVal))
	}
	return lines
}

func GenerateDynamicModifiersLocalisation(modifiers []*Obj, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, m := range modifiers {
		lines = idOnlyNameEmit(lines, m, "dm_id", lang)
	}
	return strings.Join(lines, "\n")
}

func GenerateAutonomyStatesLocalisation(states []*Obj, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, s := range states {
		lines = idOnlyNameEmit(lines, s, "autonomy_id", lang)
	}
	return strings.Join(lines, "\n")
}

func GenerateMIOLocalisation(mios []*Obj, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, m := range mios {
		mioID := m.Attr("mio_id")
		if !PyTruthy(mioID) {
			continue
		}
		nameEn := resolveMeaningfulDisplayName(m, "en", mioID, legacyName, "", true)
		if nameEn.Truthy() {
			var val Text
			if lang == "en" {
				val = nameEn
			} else {
				nameLang := resolveMeaningfulDisplayName(m, lang, mioID, legacyName, "", true)
				val = orText(nameLang, nameEn)
			}
			lines = append(lines, YmlLine(PyStr(mioID), val))
		}
		seen := map[string]bool{}
		traitsVal := m.Attr("traits")
		var traitArr []Value
		if PyTruthy(traitsVal) && traitsVal.Kind == KArr {
			traitArr = traitsVal.Arr
		}
		for _, trait := range traitArr {
			if trait.Kind != KObj {
				continue
			}

			cand := trait.Obj.Attr("name")
			if !PyTruthy(cand) {
				cand = trait.Obj.Attr("token")
			}
			key := pyStrip(strOrPyStr(cand))
			if key == "" || strings.Contains(key, " ") || seen[key] {
				continue
			}
			seen[key] = true
			lines = append(lines, YmlLine(key, Text{S: HumanizeMIOToken(key)}))
		}
	}
	return strings.Join(lines, "\n")
}

func GenerateDoctrineLocalisation(doctrines []*Obj, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	seen := map[string]bool{}
	for _, d := range doctrines {
		nameKey := pyStrip(strOrEmpty(d.Attr("name")))
		if nameKey != "" && !strings.Contains(nameKey, " ") && !seen[nameKey] {
			textEn := resolveMeaningfulDisplayName(d, "en", Str(nameKey), legacyName, "", true)
			var val Text
			if lang == "en" {
				val = textEn
			} else {
				textLang := resolveMeaningfulDisplayName(d, lang, Str(nameKey), legacyName, "", true)
				val = orText(textLang, textEn)
			}
			if val.Truthy() {
				seen[nameKey] = true
				lines = append(lines, YmlLine(nameKey, val))
			}
		}
		descKey := pyStrip(strOrEmpty(d.Attr("description")))
		if descKey != "" && !strings.Contains(descKey, " ") && !seen[descKey] {
			dtextEn := resolveDescription(d, "en", legacyDesc)
			dtextLang := dtextEn
			if lang != "en" {
				dtextLang = resolveDescription(d, lang, legacyDesc)
			}
			dval := dtextEn
			if lang != "en" && dtextLang != "" {
				dval = dtextLang
			}
			if dval != "" {
				seen[descKey] = true
				lines = append(lines, YmlLine(descKey, Text{S: dval}))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func GenerateIntelAgencyLocalisation(agencies []*Obj, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	seen := map[string]bool{}
	for _, a := range agencies {
		authoredEn := resolveMeaningfulDisplayName(a, "en", Null(), legacyName, "", true)
		var authored Text
		if lang == "en" {
			authored = authoredEn
		} else {
			authoredLang := resolveMeaningfulDisplayName(a, lang, Null(), legacyName, "", true)
			authored = orText(authoredLang, authoredEn)
		}
		firstKeyUsed := false
		for _, n := range EffectiveIntelligenceAgencyNames(a) {
			key := pyStrip(strOrPyStr(n))
			if key == "" || !fullmatchIdentDot(key) || seen[key] {
				continue
			}
			seen[key] = true
			if !firstKeyUsed && authored.Truthy() &&
				!IsPlaceholderDisplayName(authored.S, Str(key)) {
				lines = append(lines, YmlLine(key, authored))
				firstKeyUsed = true
			} else {
				lines = append(lines, YmlLine(key, Text{S: HumanizeMIOToken(key)}))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func fullmatchIdentDot(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

func GenerateDecisionsLocalisation(categories, decisions []*Obj, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, cat := range categories {
		catID := cat.Attr("category_id")
		key := PyStr(catID)
		nameEn := resolveMeaningfulDisplayName(cat, "en", catID, legacyName, "", true)
		if nameEn.Truthy() {
			var val Text
			if lang == "en" {
				val = nameEn
			} else {
				nameLang := resolveMeaningfulDisplayName(cat, lang, catID, legacyName, "", true)
				val = orText(nameLang, nameEn)
			}
			lines = append(lines, YmlLine(key, val))
		}
		descEn := resolveDescription(cat, "en", legacyDesc)
		descLang := descEn
		if lang != "en" {
			descLang = resolveDescription(cat, lang, legacyDesc)
		}
		if descEn != "" || descLang != "" {
			val := descEn
			if lang != "en" && descLang != "" {
				val = descLang
			}
			lines = append(lines, YmlLine(key+"_desc", Text{S: val}))
		}
	}
	for _, dec := range decisions {
		decID := dec.Attr("decision_id")
		key := PyStr(decID)
		nameEn := resolveMeaningfulDisplayName(dec, "en", decID, legacyName, "", true)
		if nameEn.Truthy() {
			var val Text
			if lang == "en" {
				val = nameEn
			} else {
				nameLang := resolveMeaningfulDisplayName(dec, lang, decID, legacyName, "", true)
				val = orText(nameLang, nameEn)
			}
			lines = append(lines, YmlLine(key, val))
		}
		descEn := resolveDescription(dec, "en", legacyDesc)
		descLang := descEn
		if lang != "en" {
			descLang = resolveDescription(dec, lang, legacyDesc)
		}
		if descEn != "" || descLang != "" {
			val := descEn
			if lang != "en" && descLang != "" {
				val = descLang
			}
			lines = append(lines, YmlLine(key+"_desc", Text{S: val}))
		}
		ttEn := resolveEffectTooltip(dec, "en")
		ttLang := ttEn
		if lang != "en" {
			ttLang = resolveEffectTooltip(dec, lang)
		}
		if ttEn != "" || ttLang != "" {
			val := ttEn
			if lang != "en" && ttLang != "" {
				val = ttLang
			}
			lines = append(lines, YmlLine(key+"_et_tt", Text{S: val}))
		}
	}
	return strings.Join(lines, "\n")
}

func GenerateStatesLocalisation(states []*Obj, lang string) string {
	lines := []string{}
	for _, st := range states {
		sid := st.Attr("state_id")
		if sid.Kind == KNull || sid.Kind == KMissing {
			continue
		}
		locsVal := st.Attr("display_name_locs")
		var val string
		if locsVal.Kind == KObj {
			val = pyStrip(strOrPyStr(locsVal.Obj.Attr(lang)))
		} else {
			val = pyStrip(strOrPyStr(Value{Kind: KMissing}))
		}
		key := "STATE_" + PyStr(sid)
		if IsPlaceholderDisplayName(val, Str(key)) {
			continue
		}
		lines = append(lines, YmlLine(key, Text{S: val}))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(append([]string{langHeader(lang) + ":"}, lines...), "\n")
}
