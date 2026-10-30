package events

import (
	"fmt"
	"math/big"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

var eventTypes = map[string]bool{
	"country_event": true, "news_event": true, "state_event": true,
	"unit_leader_event": true, "operative_leader_event": true,
}

var eventKnownFields = map[string]bool{
	"id": true, "title": true, "desc": true, "picture": true,
	"is_triggered_only": true, "fire_only_once": true, "hidden": true,
	"major": true, "trigger": true, "mean_time_to_happen": true,
	"immediate": true, "after": true, "option": true, "timeout_days": true,
}

func parseEventOption(optBlock []*paradox.Entry) map[string]any {
	result := map[string]any{
		"name": "", "trigger": "", "effects": "",
		"hidden_effects": "", "ai_chance": map[string]any{},
	}

	var effectLines []string
	for _, entry := range optBlock {
		key := ""
		if entry.Key != nil {
			key = *entry.Key
		}

		switch {
		case key == "name":
			result["name"] = pyStrEntryValue(entry)
		case key == "trigger" && entry.Block != nil:
			result["trigger"] = importsvc.BlockToText(entry.Block, 0)
		case key == "hidden_effect" && entry.Block != nil:
			result["hidden_effects"] = importsvc.BlockToText(entry.Block, 0)
		case key == "ai_chance" && entry.Block != nil:
			ai := map[string]any{}
			modifiers := []any{}
			for _, aiEntry := range entry.Block.Entries {
				akey := ""
				if aiEntry.Key != nil {
					akey = *aiEntry.Key
				}
				switch {

				case akey == "base":
					if z, ok := pyIntCoerce(aiEntry.Scalar); ok && aiEntry.Block == nil {
						ai["base"] = z
					} else {
						ai["base"] = big.NewInt(1)
					}
				case akey == "factor":
					if aiEntry.Block == nil {
						if z, ok := pyIntCoerce(aiEntry.Scalar); ok {
							ai["factor"] = z
						}
					}
				case akey == "modifier" && aiEntry.Block != nil:
					mod := map[string]any{}
					for _, m := range aiEntry.Block.Entries {
						mk := ""
						if m.Key != nil {
							mk = *m.Key
						}
						switch {
						case mk == "factor":

							if m.Block == nil {
								if f, ok := pyFloatStr(m.Scalar); ok {
									mod["factor"] = f
								}
							}
						case mk == "add":
							if m.Block == nil {
								if f, ok := pyFloatStr(m.Scalar); ok {
									mod["add"] = f
								}
							}
						case m.Block != nil:

							mod["trigger"] = importsvc.BlockToText(
								&paradox.Block{Entries: []*paradox.Entry{m}}, 0)
						default:

							mop := "="
							if m.Op != nil && *m.Op != "" {
								mop = *m.Op
							}
							line := mk + " " + mop + " " + importsvc.PyScalarStr(m.Scalar)
							cur, _ := mod["trigger"].(string)
							if cur != "" {
								mod["trigger"] = cur + "\n" + line
							} else {
								mod["trigger"] = line
							}
						}
					}
					modifiers = append(modifiers, mod)
				}
			}
			if len(modifiers) > 0 {
				ai["modifiers"] = modifiers
			}
			result["ai_chance"] = ai
		default:

			if entry.Block != nil {
				effectLines = append(effectLines, key+" = {")
				effectLines = append(effectLines, importsvc.BlockToText(entry.Block, 1))
				effectLines = append(effectLines, "}")
			} else {

				eop := "="
				if entry.Op != nil && *entry.Op != "" {
					eop = *entry.Op
				}
				effectLines = append(effectLines,
					key+" "+eop+" "+importsvc.PyScalarStr(entry.Scalar))
			}
		}
	}

	if len(effectLines) > 0 {
		result["effects"] = strings.Join(effectLines, "\n")
	}
	return result
}

func ImportEvents(text string) (namespaces []string, results []map[string]any, summary *importsvc.ImportSummary, err error) {
	summary = importsvc.NewImportSummary("event")
	ast, _, perr := paradox.Parse(text)
	if perr != nil {
		return nil, nil, summary, perr
	}

	expandAtConstants(ast)

	namespaces = []string{}
	for _, entry := range ast.Entries {
		if entry.Key != nil && *entry.Key == "add_namespace" && entry.Block == nil {
			namespaces = append(namespaces, importsvc.PyScalarStr(entry.Scalar))
		}
	}

	results = []map[string]any{}
	for _, entry := range ast.Entries {
		eventType := ""
		if entry.Key != nil {
			eventType = *entry.Key
		}
		if !eventTypes[eventType] {
			continue
		}
		if entry.Block == nil {
			continue
		}

		rawExtras := importsvc.NewRawExtras()
		ev := map[string]any{
			"event_type":          eventType,
			"event_id":            "",
			"event_number":        big.NewInt(0),
			"title":               "",
			"description":         "",
			"picture":             "",
			"is_triggered_only":   false,
			"fire_only_once":      false,
			"hidden":              false,
			"major":               false,
			"trigger":             "",
			"mean_time_to_happen": "",
			"immediate":           "",
			"after":               "",
			"options":             []any{},
			"timeout_days":        big.NewInt(0),
			"raw_extras":          rawExtras,
		}
		var titles, descs []any
		titleEn, hasTitleEn := "", false
		descEn, hasDescEn := "", false
		var scopeTriggerBlocks [][]*paradox.Entry

		for _, fe := range entry.Block.Entries {
			key := ""
			if fe.Key != nil {
				key = *fe.Key
			}
			isBlock := fe.Block != nil

			switch {
			case key == "id":
				s := pyStrEntryValue(fe)
				ev["event_id"] = s

				parts := strings.Split(s, ".")
				if len(parts) >= 2 {
					if z, ok := pyIntStr(parts[len(parts)-1]); ok {
						ev["event_number"] = z
					}
				}
			case key == "title":
				if isBlock {

					t := map[string]any{}
					for _, sub := range fe.Block.Entries {
						sk := ""
						if sub.Key != nil {
							sk = *sub.Key
						}
						if sk == "text" {
							t["text"] = pyStrEntryValue(sub)
						} else if sk == "trigger" && sub.Block != nil {
							t["trigger"] = importsvc.BlockToText(sub.Block, 0)
						}
					}
					titles = append(titles, t)
				} else {
					s := importsvc.PyScalarStr(fe.Scalar)
					ev["title"] = s

					if !hasTitleEn || titleEn == "" {
						titleEn = s
						hasTitleEn = true
					}
				}
			case key == "desc":
				if isBlock {
					d := map[string]any{}
					for _, sub := range fe.Block.Entries {
						sk := ""
						if sub.Key != nil {
							sk = *sub.Key
						}
						if sk == "text" {
							d["text"] = pyStrEntryValue(sub)
						} else if sk == "trigger" && sub.Block != nil {
							d["trigger"] = importsvc.BlockToText(sub.Block, 0)
						}
					}
					descs = append(descs, d)
				} else {
					s := importsvc.PyScalarStr(fe.Scalar)
					ev["description"] = s
					if !hasDescEn || descEn == "" {
						descEn = s
						hasDescEn = true
					}
				}
			case key == "picture":
				ev["picture"] = pyStrEntryValue(fe)
			case key == "is_triggered_only":
				ev["is_triggered_only"] = pyYesEntry(fe)
			case key == "fire_only_once":
				ev["fire_only_once"] = pyYesEntry(fe)
			case key == "hidden":
				ev["hidden"] = pyYesEntry(fe)
			case key == "major":
				ev["major"] = pyYesEntry(fe)
			case key == "timeout_days":

				if !isBlock {
					if z, ok := pyIntCoerce(fe.Scalar); ok {
						ev["timeout_days"] = z
					}
				}
			case key == "trigger" && isBlock:

				scopeTriggerBlocks = append(scopeTriggerBlocks, fe.Block.Entries)
				ev["trigger"] = importsvc.BlockToText(fe.Block, 0)
			case key == "mean_time_to_happen" && isBlock:
				ev["mean_time_to_happen"] = importsvc.BlockToText(fe.Block, 0)
			case key == "immediate" && isBlock:
				ev["immediate"] = importsvc.BlockToText(fe.Block, 0)
			case key == "after" && isBlock:
				ev["after"] = importsvc.BlockToText(fe.Block, 0)
			case key == "option" && isBlock:
				ev["options"] = append(ev["options"].([]any),
					parseEventOption(fe.Block.Entries))
			case !eventKnownFields[key]:
				if isBlock {
					importsvc.AccumulateRawExtra(rawExtras, key,
						importsvc.BlockToText(fe.Block, 0))
				} else {
					importsvc.AccumulateRawExtra(rawExtras, key,
						importsvc.PyScalarStr(fe.Scalar))
				}
				summary.AddPreserved(key)
			}

		}

		if hasTitleEn {
			ev["title_en"] = titleEn
		}
		if hasDescEn {
			ev["desc_en"] = descEn
		}

		if titles != nil {
			ev["title"] = titles
		}
		if descs != nil {
			ev["description"] = descs
		}

		ev["_scope_verdict"] = importsvc.ClassifyScope(scopeTriggerBlocks)

		results = append(results, ev)
		summary.Total++
	}

	if len(namespaces) == 0 && len(results) == 0 {
		summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
			"no_content_events", nil,
			"A file in the Events folder had no events in it — skipped.",
		))
	}

	return namespaces, results, summary, nil
}

var decisionCategoryKnownFields = map[string]bool{
	"icon": true, "picture": true, "priority": true,
	"allowed": true, "visible": true, "available": true, "cancel_trigger": true,
	"visible_when_empty": true, "visibility_type": true,
	"custom_icon": true, "highlight_states": true, "on_map_area": true,
	"scripted_gui": true,
}

var decisionMarkerFields = map[string]bool{
	"cost": true, "fire_only_once": true, "days_remove": true,
	"days_re_enable": true, "complete_effect": true, "remove_effect": true,
	"cancel_trigger": true, "cancel_effect": true, "available": true,
	"modifier": true, "ai_will_do": true, "target_trigger": true,
	"targets": true, "target_array": true, "target_root_trigger": true,
}

func looksLikeDecisionBody(body []*paradox.Entry) bool {
	for _, sub := range body {
		if sub.Key != nil && *sub.Key != "" && decisionMarkerFields[*sub.Key] {
			return true
		}
	}
	return false
}

func ImportDecisionCategories(text string) (results []map[string]any, summary *importsvc.ImportSummary, err error) {
	summary = importsvc.NewImportSummary("decision_category")
	ast, _, perr := paradox.Parse(text)
	if perr != nil {
		return nil, summary, perr
	}

	expandAtConstants(ast)

	results = []map[string]any{}
	for _, entry := range ast.Entries {
		if entry.Key == nil || entry.Block == nil {
			continue
		}

		catID := *entry.Key
		body := entry.Block.Entries

		priority := big.NewInt(0)
		if sc := ppScalar(body, "priority"); sc != nil {
			z, ek := pyIntFloatStr(*sc)
			if ek != "" {
				return nil, summary, &PyError{Type: ek}
			}
			priority = z
		}

		onMapArea := ""
		if blk := ppBlock(body, "on_map_area"); blk != nil {
			onMapArea = importsvc.BlockToText(blk, 0)
		}

		rawExtras := importsvc.NewRawExtras()
		cat := map[string]any{
			"category_id": catID,
			"icon":        ppScalarStr(body, "icon"),
			"picture":     ppScalarStr(body, "picture"),
			"priority":    priority,
			"allowed":     "",
			"visible":     "",
			"on_map_area": onMapArea,
			"raw_extras":  rawExtras,
		}

		var catScopeBlocks [][]*paradox.Entry
		for _, fieldName := range []string{"allowed", "visible"} {
			blk := ppBlock(body, fieldName)
			if blockTruthy(blk) {
				cat[fieldName] = importsvc.BlockToText(blk, 0)
				catScopeBlocks = append(catScopeBlocks, blk.Entries)
			}
		}
		cat["_scope_verdict"] = importsvc.ClassifyScope(catScopeBlocks)

		for _, sub := range body {
			if sub.Key == nil || *sub.Key == "" {
				continue
			}
			k := *sub.Key
			if decisionCategoryKnownFields[k] {
				continue
			}
			if sub.Block != nil {
				importsvc.AccumulateRawExtra(rawExtras, k,
					importsvc.BlockToText(sub.Block, 0))
			} else {
				importsvc.AccumulateRawExtra(rawExtras, k, scalarAny(sub.Scalar))
			}
			summary.AddPreserved(k)
		}

		results = append(results, cat)
	}

	summary.Total = len(results)
	if len(results) == 0 {
		summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
			"no_content_decision_categories", nil,
			"A file in the Decisions folder had no decision categories in it — skipped.",
		))
	}
	return results, summary, nil
}

var decisionKnownFields = map[string]bool{
	"icon": true, "allowed": true, "available": true, "visible": true,
	"cost": true, "fire_only_once": true, "days_remove": true,
	"days_re_enable": true, "complete_effect": true, "remove_effect": true,
	"cancel_trigger": true, "cancel_effect": true, "modifier": true,
	"ai_will_do": true,

	"activation": true, "timeout_effect": true, "days_mission_timeout": true,
	"selectable_mission": true, "is_good": true,

	"state_target": true, "target_trigger": true, "targets": true,
	"target_array": true, "target_root_trigger": true, "priority": true,
	"targeted_modifier": true, "custom_cost_trigger": true,
	"custom_cost_text": true, "war_with_on_add": true, "war_with_on_remove": true,

	"cancel_if_not_visible": true, "on_map_mode": true, "highlight_states": true,

	"target_non_existing": true,
}

var decisionColumnFields = map[string]bool{
	"icon": true, "allowed": true, "available": true, "visible": true,
	"cost": true, "fire_only_once": true, "days_remove": true,
	"days_re_enable": true, "complete_effect": true, "remove_effect": true,
	"cancel_trigger": true, "cancel_effect": true, "modifier": true,
	"ai_will_do": true,
	"activation": true, "timeout_effect": true, "days_mission_timeout": true,
	"selectable_mission": true, "is_good": true,
	"targeted_modifier": true, "targets": true, "target_array": true,
	"target_root_trigger": true, "target_trigger": true, "state_target": true,
	"priority": true, "custom_cost_trigger": true, "custom_cost_text": true,
	"war_with_on_add": true, "war_with_on_remove": true,
	"cancel_if_not_visible": true, "on_map_mode": true, "highlight_states": true,
}

func ImportDecisions(text string) (categories, decisions []map[string]any, summary *importsvc.ImportSummary, err error) {
	summary = importsvc.NewImportSummary("decision")
	ast, _, perr := paradox.Parse(text)
	if perr != nil {
		return nil, nil, summary, perr
	}

	expandAtConstants(ast)

	categories = []map[string]any{}
	decisions = []map[string]any{}
	var skippedGhosts []string

	for _, entry := range ast.Entries {
		if entry.Key == nil || entry.Block == nil {
			continue
		}

		catID := *entry.Key
		catBody := entry.Block.Entries

		hasCatMarker := ppScalar(catBody, "icon") != nil ||
			ppScalar(catBody, "picture") != nil ||
			ppScalar(catBody, "priority") != nil
		if !hasCatMarker {
			for _, k := range []string{"allowed", "visible"} {
				if blockTruthy(ppBlock(catBody, k)) {
					hasCatMarker = true
					break
				}
			}
		}
		hasDecisionChild := false
		for _, sub := range catBody {
			if sub.Key != nil && *sub.Key != "" && sub.Block != nil &&
				!decisionCategoryKnownFields[*sub.Key] &&
				looksLikeDecisionBody(sub.Block.Entries) {
				hasDecisionChild = true
				break
			}
		}
		if !hasCatMarker && !hasDecisionChild {
			skippedGhosts = append(skippedGhosts, catID)
			continue
		}

		onMapArea := ""
		if blk := ppBlock(catBody, "on_map_area"); blk != nil {
			onMapArea = importsvc.BlockToText(blk, 0)
		}

		rawExtras := importsvc.NewRawExtras()
		cat := map[string]any{
			"category_id": catID,
			"icon":        ppScalarStr(catBody, "icon"),
			"picture":     ppScalarStr(catBody, "picture"),
			"priority":    big.NewInt(0),
			"allowed":     "",
			"visible":     "",
			"on_map_area": onMapArea,
			"raw_extras":  rawExtras,
		}

		if sc := ppScalar(catBody, "priority"); sc != nil {
			z, ek := pyIntFloatStr(*sc)
			if ek == "OverflowError" {
				return nil, nil, summary, &PyError{Type: ek}
			}
			if ek == "" {
				cat["priority"] = z
			}
		}

		var catScopeBlocks [][]*paradox.Entry
		for _, fieldName := range []string{"allowed", "visible"} {
			blk := ppBlock(catBody, fieldName)
			if blockTruthy(blk) {
				cat[fieldName] = importsvc.BlockToText(blk, 0)
				catScopeBlocks = append(catScopeBlocks, blk.Entries)
			}
		}
		cat["_scope_verdict"] = importsvc.ClassifyScope(catScopeBlocks)

		for _, sub := range catBody {
			if sub.Key == nil || *sub.Key == "" {
				continue
			}
			k := *sub.Key
			if !decisionCategoryKnownFields[k] && sub.Block == nil {
				importsvc.AccumulateRawExtra(rawExtras, k, scalarAny(sub.Scalar))
				summary.AddPreserved(k)
			}
		}

		categories = append(categories, cat)

		for _, decEntry := range catBody {
			if decEntry.Key == nil || decEntry.Block == nil {
				continue
			}

			if decisionCategoryKnownFields[*decEntry.Key] {
				continue
			}
			dec, derr := mapOneDecision(catID, *decEntry.Key, decEntry.Block.Entries, summary)
			if derr != nil {
				return nil, nil, summary, derr
			}
			decisions = append(decisions, dec)
		}
	}

	summary.Total = len(decisions)
	if len(skippedGhosts) > 0 {
		head := skippedGhosts
		if len(head) > 5 {
			head = head[:5]
		}
		preview := strings.Join(head, ", ")
		if len(skippedGhosts) > 5 {
			preview += fmt.Sprintf(", ... (+%d more)", len(skippedGhosts)-5)
		}
		summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
			"decision_skipped_blocks",
			[]importsvc.WarnParam{
				{Key: "count", Value: len(skippedGhosts)},
				{Key: "preview", Value: preview},
			},
			fmt.Sprintf("Skipped %d block(s) in a decisions file that "+
				"weren't decision categories: %s.", len(skippedGhosts), preview),
		))
	}
	if len(categories) == 0 && len(decisions) == 0 {
		summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
			"no_content_decisions", nil,
			"A file in the Decisions folder had no decisions in it — skipped.",
		))
	}

	return categories, decisions, summary, nil
}

func mapOneDecision(categoryID, decisionID string, body []*paradox.Entry, summary *importsvc.ImportSummary) (map[string]any, error) {
	rawExtras := importsvc.NewRawExtras()
	data := map[string]any{
		"category_id":           categoryID,
		"decision_id":           decisionID,
		"icon":                  ppScalarStr(body, "icon"),
		"allowed":               "",
		"available":             "",
		"visible":               "",
		"cost":                  big.NewInt(0),
		"fire_only_once":        false,
		"days_remove":           big.NewInt(0),
		"days_re_enable":        big.NewInt(0),
		"complete_effect":       "",
		"remove_effect":         "",
		"cancel_trigger":        "",
		"cancel_effect":         "",
		"modifier":              "",
		"activation":            "",
		"timeout_effect":        "",
		"days_mission_timeout":  big.NewInt(0),
		"selectable_mission":    false,
		"is_good":               false,
		"ai_will_do":            "",
		"targeted_modifier":     map[string]any{},
		"targets":               []any{},
		"target_array":          []any{},
		"target_root_trigger":   "",
		"target_trigger":        "",
		"state_target":          "",
		"priority":              big.NewInt(0),
		"custom_cost_trigger":   "",
		"custom_cost_text":      "",
		"war_with_on_add":       "",
		"war_with_on_remove":    "",
		"cancel_if_not_visible": false,
		"on_map_mode":           "",
		"highlight_states":      []any{},
		"raw_extras":            rawExtras,
	}

	if sc := ppScalar(body, "fire_only_once"); sc != nil {
		data["fire_only_once"] = strings.ToLower(importsvc.PyScalarStr(*sc)) == "yes"
	}
	for _, boolField := range []string{"selectable_mission", "is_good", "cancel_if_not_visible"} {
		if sc := ppScalar(body, boolField); sc != nil {
			data[boolField] = strings.ToLower(importsvc.PyScalarStr(*sc)) == "yes"
		}
	}

	for _, intField := range []string{"cost", "days_remove", "days_re_enable",
		"days_mission_timeout"} {
		if sc := ppScalar(body, intField); sc != nil {
			z, ek := pyIntFloatStr(*sc)
			if ek == "OverflowError" {
				return nil, &PyError{Type: ek}
			}
			if ek == "" {
				data[intField] = z
			}
		}
	}

	scopeFieldSet := map[string]bool{"allowed": true, "available": true, "visible": true}
	var decScopeBlocks [][]*paradox.Entry
	for _, fieldName := range []string{"allowed", "available", "visible",
		"complete_effect", "remove_effect",
		"cancel_trigger", "cancel_effect",
		"modifier", "ai_will_do",
		"activation", "timeout_effect"} {
		blk := ppBlock(body, fieldName)
		if blockTruthy(blk) {
			data[fieldName] = importsvc.BlockToText(blk, 0)
			if scopeFieldSet[fieldName] {
				decScopeBlocks = append(decScopeBlocks, blk.Entries)
			}
		}
	}
	data["_scope_verdict"] = importsvc.ClassifyScope(decScopeBlocks)

	if sc := ppScalar(body, "state_target"); sc != nil {
		data["state_target"] = pyStrip(importsvc.PyScalarStr(*sc))
	}
	if sc := ppScalar(body, "priority"); sc != nil {
		z, ek := pyIntFloatStr(*sc)
		if ek == "OverflowError" {
			return nil, &PyError{Type: ek}
		}
		if ek == "" {
			data["priority"] = z
		}
	}
	if sc := ppScalar(body, "custom_cost_text"); sc != nil {
		data["custom_cost_text"] = pyStrip(importsvc.PyScalarStr(*sc))
	}

	if sc := ppScalar(body, "on_map_mode"); sc != nil {
		data["on_map_mode"] = pyStrip(importsvc.PyScalarStr(*sc))
	}

	if hsBlk := ppBlock(body, "highlight_states"); hsBlk != nil {
		if targetsSub := ppBlock(hsBlk.Entries, "highlight_state_targets"); targetsSub != nil {
			lst := []any{}
			for _, e := range targetsSub.Entries {
				if e.Key != nil && *e.Key == "state" {
					if s := pyStrip(pyStrEntryValue(e)); s != "" {
						lst = append(lst, s)
					}
				}
			}
			data["highlight_states"] = lst
		} else {
			importsvc.AccumulateRawExtra(rawExtras, "highlight_states",
				importsvc.BlockToText(hsBlk, 0))
			summary.AddPreserved("highlight_states")
		}
	}
	for _, wField := range []string{"war_with_on_add", "war_with_on_remove"} {
		if sc := ppScalar(body, wField); sc != nil {
			data[wField] = pyStrip(importsvc.PyScalarStr(*sc))
		}
	}
	for _, tField := range []string{"target_root_trigger", "target_trigger", "custom_cost_trigger"} {
		if blk := ppBlock(body, tField); blk != nil {
			data[tField] = importsvc.BlockToText(blk, 0)
		}
	}
	if targetsBlk := ppBlock(body, "targets"); targetsBlk != nil {
		lst := []any{}
		for _, e := range targetsBlk.Entries {
			if e.Key == nil {
				if s := pyStrip(pyStrEntryValue(e)); s != "" {
					lst = append(lst, s)
				}
			}
		}
		data["targets"] = lst
	}

	if sc := ppScalar(body, "target_array"); sc != nil {
		if taS := pyStrip(importsvc.PyScalarStr(*sc)); taS != "" {
			data["target_array"] = []any{taS}
		}
	} else if taBlk := ppBlock(body, "target_array"); taBlk != nil {
		lst := []any{}
		for _, e := range taBlk.Entries {
			if e.Key == nil {
				if s := pyStrip(pyStrEntryValue(e)); s != "" {
					lst = append(lst, s)
				}
			}
		}
		data["target_array"] = lst
	}

	tmodBlocks := []any{}
	for _, sub := range body {
		if sub.Key == nil || *sub.Key != "targeted_modifier" || sub.Block == nil {
			continue
		}
		tagVal := ""
		mods := map[string]any{}
		for _, ms := range sub.Block.Entries {
			mk := ""
			if ms.Key != nil {
				mk = *ms.Key
			}
			if mk == "" {
				continue
			}
			if mk == "tag" {
				tagVal = pyStrip(pyStrEntryValue(ms))
			} else if ms.Block != nil {
				mods[mk] = importsvc.BlockToText(ms.Block, 0)
			} else {
				if f, ok := pyFloatStr(ms.Scalar); ok {
					mods[mk] = f
				} else {
					mods[mk] = importsvc.PyScalarStr(ms.Scalar)
				}
			}
		}
		if tagVal != "" || len(mods) > 0 {
			tmodBlocks = append(tmodBlocks, map[string]any{"tag": tagVal, "modifier": mods})
		}
	}
	if len(tmodBlocks) == 1 {
		data["targeted_modifier"] = tmodBlocks[0]
	} else if len(tmodBlocks) > 1 {
		data["targeted_modifier"] = tmodBlocks
	}

	for _, sub := range body {
		if sub.Key == nil || *sub.Key == "" {
			continue
		}
		k := *sub.Key
		known := decisionKnownFields[k]
		if !known || !decisionColumnFields[k] {
			if sub.Block != nil {
				importsvc.AccumulateRawExtra(rawExtras, k,
					importsvc.BlockToText(sub.Block, 0))
			} else {
				importsvc.AccumulateRawExtra(rawExtras, k, scalarAny(sub.Scalar))
			}
			summary.AddPreserved(k)
		}
	}

	return data, nil
}
