package world

import (
	"math"
	"math/big"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func superEventNamespaceCtag(namespaceName string) string {
	ns := pyLower(pyStrip(namespaceName))
	if !strings.HasSuffix(ns, "_se") {
		return ""
	}
	prefix := ns[:len(ns)-3]
	if prefix == "" {
		return ""
	}
	return pyUpper(prefix)
}

func seAiChance(baseVal any) (any, *PyRaise) {
	f, exc := pyFloatAny(baseVal)
	if exc != nil {
		if exc.Type == "OverflowError" {
			return nil, exc
		}
		return nil, nil
	}
	if math.IsInf(f, 0) {
		return nil, &PyRaise{Type: "OverflowError"}
	}
	if math.IsNaN(f) {
		return nil, nil
	}
	if math.Trunc(f) == f {
		z, _ := pyIntFromFloat(f)
		return z, nil
	}
	return f, nil
}

func ImportSuperEvents(text string) ([]*PyDict, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("super_event")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		summary.Warnings = append(summary.Warnings, importsvc.ParseSkipWarning(summary, ""))
		return []*PyDict{}, summary, nil
	}
	expandAtConstants(ast)

	seNamespaces := map[string]bool{}
	for _, ent := range ast.Entries {
		if keyIs(ent, "add_namespace") {
			nsName := pyStripChar(pyStrip(pyStr(entVal(ent))), '"')
			if strings.HasSuffix(pyLower(nsName), "_se") {
				seNamespaces[pyLower(nsName)] = true
			}
		}
	}
	if len(seNamespaces) == 0 {
		return []*PyDict{}, summary, nil
	}

	results := []*PyDict{}
	for _, ent := range ast.Entries {
		if !keyIs(ent, "country_event") && !keyIs(ent, "news_event") {
			continue
		}
		if ent.Block == nil {
			continue
		}
		body := ent.Block.Entries
		eventID := ""
		if idV, ok := ppScalar(body, "id"); ok {
			eventID = pyStripChar(pyStrip(pyStr(idV)), '"')
		}
		if !strings.Contains(eventID, ".") {
			continue
		}
		ri := strings.LastIndex(eventID, ".")
		nsName, numStr := eventID[:ri], eventID[ri+1:]
		if !seNamespaces[pyLower(nsName)] {
			continue
		}
		number, okInt := pyIntFromStr(numStr)
		if !okInt {
			summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
				"super_event_bad_id",
				[]importsvc.WarnParam{{Key: "id", Value: eventID}},
				"Skipped a super-event with an unusual id ("+eventID+") — nothing else was affected.",
			))
			continue
		}

		ctag := superEventNamespaceCtag(nsName)

		data := NewPyDict()
		data.Set("number", number)
		data.Set("country_tag", ctag)
		data.Set("title_en", "")
		data.Set("title_zh", "")
		data.Set("desc_en", "")
		data.Set("desc_zh", "")
		data.Set("quote_en", "")
		data.Set("quote_zh", "")
		data.Set("btn_en", "")
		data.Set("btn_zh", "")
		data.Set("picture_path", "")
		data.Set("song_token", "")
		data.Set("auto_fire_on_startup", false)
		data.Set("auto_fire_after_days", big.NewInt(0))
		data.Set("options", []any{})

		if immediate := ppBlock(body, "immediate"); immediate != nil && len(immediate.Entries) > 0 {
			if songRaw, ok := ppScalar(immediate.Entries, "play_song"); ok && truthy(songRaw) {
				data.Set("song_token", pyStripChar(pyStrip(pyStr(songRaw)), '"'))
			}
		}

		for _, optBlock := range ppFindAll(body, "option") {
			if optBlock.Block == nil {
				continue
			}
			opt := NewPyDict()
			opt.Set("name_en", "")
			opt.Set("name_zh", "")
			opt.Set("effect", "")
			opt.Set("ai_chance", nil)
			opt.Set("_name_loc_key", "")
			if nameVal, ok := ppScalar(optBlock.Block.Entries, "name"); ok && truthy(nameVal) {
				opt.Set("_name_loc_key", pyStripChar(pyStrip(pyStr(nameVal)), '"'))
			}

			if aic := ppBlock(optBlock.Block.Entries, "ai_chance"); aic != nil && len(aic.Entries) > 0 {
				if baseVal, ok := ppScalar(aic.Entries, "base"); ok {
					ai, exc := seAiChance(baseVal)
					if exc != nil {
						return nil, nil, exc
					}
					opt.Set("ai_chance", ai)
				}
			}

			var effectEntries []*paradox.Entry
			for _, sub := range optBlock.Block.Entries {
				if sub.Key != nil && (*sub.Key == "name" || *sub.Key == "ai_chance") {
					continue
				}
				effectEntries = append(effectEntries, sub)
			}
			if len(effectEntries) > 0 {
				opt.Set("effect", pyStrip(importsvc.ToScriptEntries(effectEntries, 0)))
			}
			data.appendTo("options", opt)
		}

		results = append(results, data)
		summary.AddPreserved("super_event")
	}

	summary.Total = len(results)
	return results, summary, nil
}
