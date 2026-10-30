package preview

import "strings"

func hasLocalizedText(ent *OD, dictAttr string, legacyAttrs [2]string) bool {
	if locs := getOD(ent, dictAttr); locs != nil {
		for _, k := range locs.Keys() {
			v, _ := locs.Get(k)
			if s, ok := v.(string); ok && pyStrip(s) != "" {
				return true
			}
		}
	}
	for _, a := range legacyAttrs {
		val := orEmptyStr(attr(ent, a))
		if pyStrip(val) != "" {
			return true
		}
	}
	return false
}

func generateSingleEventScriptLines(ev *OD, validPictures map[string]bool, gatePictures bool) []string {
	var lines []string
	lines = append(lines, getStr(ev, "event_type")+" = {")
	lines = append(lines, "\tid = "+getStr(ev, "event_id"))
	eventID := getStr(ev, "event_id")

	title := attr(ev, "title")
	if titleList, ok := title.([]any); ok {
		for _, entryAny := range titleList {
			entry := asOD(entryAny)
			if entry == nil {
				continue
			}
			lines = append(lines, "\ttitle = {")
			if pyTruthy(entry.GetDef("trigger", nil)) {
				lines = append(lines, "\t\ttrigger = {")
				lines = append(lines, indentBlock(pyStr(entry.GetDef("trigger", nil)), 3)...)
				lines = append(lines, "\t\t}")
			}
			if pyTruthy(entry.GetDef("text", nil)) {
				lines = append(lines, "\t\ttext = "+pyStr(entry.GetDef("text", nil)))
			}
			lines = append(lines, "\t}")
		}
	} else if hasLocalizedText(ev, "display_name_locs", [2]string{"title_en", "title_zh"}) {
		lines = append(lines, "\ttitle = "+eventID+".t")
	} else if s, ok := title.(string); ok && s != "" {
		lines = append(lines, "\ttitle = "+s)
	}

	desc := attr(ev, "description")
	if descList, ok := desc.([]any); ok {
		for _, entryAny := range descList {
			entry := asOD(entryAny)
			if entry == nil {
				continue
			}
			lines = append(lines, "\tdesc = {")
			if pyTruthy(entry.GetDef("trigger", nil)) {
				lines = append(lines, "\t\ttrigger = {")
				lines = append(lines, indentBlock(pyStr(entry.GetDef("trigger", nil)), 3)...)
				lines = append(lines, "\t\t}")
			}
			if pyTruthy(entry.GetDef("text", nil)) {
				lines = append(lines, "\t\ttext = "+pyStr(entry.GetDef("text", nil)))
			}
			lines = append(lines, "\t}")
		}
	} else if hasLocalizedText(ev, "description_locs", [2]string{"desc_en", "desc_zh"}) {
		lines = append(lines, "\tdesc = "+eventID+".d")
	} else if s, ok := desc.(string); ok && s != "" {
		lines = append(lines, "\tdesc = "+s)
	}

	if pyTruthy(attr(ev, "picture")) {
		if !gatePictures || validPictures[pyStrip(pyStr(attr(ev, "picture")))] {
			lines = append(lines, "\tpicture = "+pyStr(attr(ev, "picture")))
		}
	}

	lines = append(lines, "")

	if pyTruthy(attr(ev, "major")) {
		lines = append(lines, "\tmajor = yes")
	}
	if pyTruthy(attr(ev, "is_triggered_only")) {
		lines = append(lines, "\tis_triggered_only = yes")
	}
	if pyTruthy(attr(ev, "fire_only_once")) {
		lines = append(lines, "\tfire_only_once = yes")
	}
	if pyTruthy(attr(ev, "hidden")) {
		lines = append(lines, "\thidden = yes")
	}
	timeoutDays := attr(ev, "timeout_days")
	if pyTruthy(timeoutDays) && valGTZero(timeoutDays) {
		lines = append(lines, "\ttimeout_days = "+pyStr(timeoutDays))
	}

	if pyTruthy(attr(ev, "trigger")) && pyStrip(getStr(ev, "trigger")) != "" {
		lines = append(lines, "")
		lines = append(lines, "\ttrigger = {")
		lines = append(lines, indentBlock(getStr(ev, "trigger"), 2)...)
		lines = append(lines, "\t}")
	}

	if pyTruthy(attr(ev, "mean_time_to_happen")) && pyStrip(getStr(ev, "mean_time_to_happen")) != "" {
		if pyTruthy(attr(ev, "is_triggered_only")) {
			lines = append(lines, "")
			lines = append(lines, "\t# mean_time_to_happen dropped: invalid with is_triggered_only = yes")
		} else {
			lines = append(lines, "")
			lines = append(lines, "\tmean_time_to_happen = {")
			lines = append(lines, indentBlock(getStr(ev, "mean_time_to_happen"), 2)...)
			lines = append(lines, "\t}")
		}
	}

	if pyTruthy(attr(ev, "immediate")) && pyStrip(getStr(ev, "immediate")) != "" {
		lines = append(lines, "")
		lines = append(lines, "\timmediate = {")
		lines = append(lines, indentBlock(getStr(ev, "immediate"), 2)...)
		lines = append(lines, "\t}")
	}

	for optIdx, optAny := range getList(ev, "options") {
		opt := asOD(optAny)
		if opt == nil {
			continue
		}
		lines = append(lines, "")
		lines = append(lines, "\toption = {")
		if pyTruthy(opt.GetDef("name", nil)) {
			optKey, optLiteral := EventOptionLocKey(opt, eventID, optIdx)
			hasLabel := pyStrip(orEmptyStr(opt.GetDef("name_en", nil))) != "" ||
				pyStrip(orEmptyStr(opt.GetDef("name_zh", nil))) != "" ||
				pyStrip(orEmptyStr(opt.GetDef("display_text", nil))) != ""
			if optLiteral != "" && !hasLabel {
				lines = append(lines, "\t\tname = \""+optLiteral+"\"")
			} else {
				lines = append(lines, "\t\tname = "+optKey)
			}
		}

		if pyTruthy(opt.GetDef("trigger", nil)) {
			lines = append(lines, "\t\ttrigger = {")
			lines = append(lines, indentBlock(pyStr(opt.GetDef("trigger", nil)), 3)...)
			lines = append(lines, "\t\t}")
		}

		if pyTruthy(opt.GetDef("effects", nil)) {
			lines = append(lines, indentBlock(pyStr(opt.GetDef("effects", nil)), 2)...)
		}

		etEn := pyStrip(orEmptyStr(opt.GetDef("effect_tooltip_en", nil)))
		etZh := pyStrip(orEmptyStr(opt.GetDef("effect_tooltip_zh", nil)))
		if etEn != "" || etZh != "" {
			ttKey, _ := EventOptionLocKey(opt, eventID, optIdx)
			lines = append(lines, "\t\tcustom_effect_tooltip = "+ttKey+"_tt")
		}

		if pyTruthy(opt.GetDef("hidden_effects", nil)) {
			lines = append(lines, "\t\thidden_effect = {")
			lines = append(lines, indentBlock(pyStr(opt.GetDef("hidden_effects", nil)), 3)...)
			lines = append(lines, "\t\t}")
		}

		ai := opt.GetDef("ai_chance", nil)
		if aiOD := asOD(ai); aiOD != nil && aiOD.Len() > 0 {
			lines = append(lines, "\t\tai_chance = {")
			if v, ok := aiOD.Get("base"); ok {
				lines = append(lines, "\t\t\tbase = "+pyStr(v))
			}
			if v, ok := aiOD.Get("factor"); ok {
				lines = append(lines, "\t\t\tfactor = "+pyStr(v))
			}
			if pyTruthy(aiOD.GetDef("modifiers", nil)) {
				for _, modAny := range asList(aiOD.GetDef("modifiers", nil)) {
					mod := asOD(modAny)
					lines = append(lines, "\t\t\tmodifier = {")
					if mod != nil {
						if v, ok := mod.Get("factor"); ok {
							lines = append(lines, "\t\t\t\tfactor = "+pyStr(v))
						}
						if v, ok := mod.Get("add"); ok {
							lines = append(lines, "\t\t\t\tadd = "+pyStr(v))
						}
						if v, ok := mod.Get("trigger"); ok {
							lines = append(lines, indentBlock(pyStr(v), 4)...)
						}
					}
					lines = append(lines, "\t\t\t}")
				}
			}
			lines = append(lines, "\t\t}")
		} else if s, ok := ai.(string); ok && pyStrip(s) != "" {
			lines = append(lines, "\t\tai_chance = {")
			lines = append(lines, indentBlock(s, 3)...)
			lines = append(lines, "\t\t}")
		} else {
			lines = append(lines, "\t\tai_chance = { base = 100 }")
		}

		lines = append(lines, "\t}")
	}

	if pyTruthy(attr(ev, "after")) && pyStrip(getStr(ev, "after")) != "" {
		lines = append(lines, "")
		lines = append(lines, "\tafter = {")
		lines = append(lines, indentBlock(getStr(ev, "after"), 2)...)
		lines = append(lines, "\t}")
	}

	if rawExtras := getOD(ev, "raw_extras"); rawExtras != nil {
		for _, key := range rawExtras.Keys() {
			value, _ := rawExtras.Get(key)
			emitRawExtras(&lines, key, value, "\t")
		}
	}

	lines = append(lines, "}")
	return lines
}

func GenerateEventsPreview(namespace string, events []*OD, validPictures map[string]bool, gatePictures bool) string {
	var lines []string
	lines = append(lines, "add_namespace = "+namespace)
	lines = append(lines, "")

	for _, ev := range events {
		lines = append(lines, generateSingleEventScriptLines(ev, validPictures, gatePictures)...)
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
