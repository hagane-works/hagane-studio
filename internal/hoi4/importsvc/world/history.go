package world

import (
	"math/big"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func containsAnyStr(lst []any, s string) bool {
	for _, v := range lst {
		if vs, ok := v.(string); ok && vs == s {
			return true
		}
	}
	return false
}

func ImportHistoryFiles(text, filename string) ([]*PyDict, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("history_file")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		summary.Warnings = append(summary.Warnings, importsvc.ParseSkipWarning(summary, ""))
		return []*PyDict{}, summary, nil
	}
	expandAtConstants(ast)

	ctag := DeriveCountryTagFromHistoryFilename(filename)

	data := NewPyDict()
	data.Set("country_tag", ctag)
	data.Set("capital", big.NewInt(0))
	data.Set("set_politics_ruling_party", "")
	data.Set("set_politics_last_election", "")
	data.Set("set_politics_election_frequency", big.NewInt(0))
	data.Set("set_politics_elections_allowed", nil)
	data.Set("set_popularities_democratic", big.NewInt(0))
	data.Set("set_popularities_communism", big.NewInt(0))
	data.Set("set_popularities_fascism", big.NewInt(0))
	data.Set("set_popularities_neutrality", big.NewInt(0))
	data.Set("set_popularities", NewPyDict())
	data.Set("set_research_slots", big.NewInt(0))
	data.Set("set_convoys", big.NewInt(0))
	data.Set("set_stability", -1.0)
	data.Set("set_war_support", -1.0)
	data.Set("add_ideas_extra", []any{})
	data.Set("remove_ideas", []any{})
	data.Set("complete_national_focus_ids", []any{})
	data.Set("set_technology_ids", []any{})
	data.Set("set_cosmetic_tag", "")
	data.Set("set_oob", "")
	data.Set("set_air_oob", "")
	data.Set("set_naval_oob", "")
	data.Set("startup_flags", []any{})
	data.Set("give_guarantee_tags", []any{})
	data.Set("add_to_faction", "")
	data.Set("add_to_faction_tags", []any{})
	data.Set("set_autonomy", []any{})
	data.Set("diplomatic_relation", []any{})
	data.Set("date_blocks", []any{})
	data.Set("raw_script", "")

	srcLines := strings.Split(text, "\n")
	var rawBounds [][2]int
	var rawFallback []string
	sawAnyKnown := false

	dateSourceByLine := map[int][]importsvc.ExtractedHistoryDateBlock{}
	for _, dated := range importsvc.ExtractHistoryDateBlocks(text) {
		dateSourceByLine[dated.Line] = append(dateSourceByLine[dated.Line], dated)
	}

	for idx, ent := range ast.Entries {
		if ent.Key != nil && ent.Block != nil {
			cands := dateSourceByLine[ent.Line]
			keyStripped := pyStrip(*ent.Key)
			matched := -1
			for i, c := range cands {
				if c.Date == keyStripped {
					matched = i
					break
				}
			}
			if matched >= 0 {
				m := cands[matched]
				dateSourceByLine[ent.Line] = append(cands[:matched:matched], cands[matched+1:]...)
				db := NewPyDict()
				db.Set("date", m.Date)
				db.Set("script", m.Script)
				data.appendTo("date_blocks", db)
				sawAnyKnown = true
				continue
			}
		}
		switch {
		case keyIs(ent, "capital"):
			v, exc := coerceInt(entVal(ent), big.NewInt(0))
			if exc != nil {
				return nil, nil, exc
			}
			data.Set("capital", v)
			sawAnyKnown = true
		case keyIs(ent, "set_politics") && ent.Block != nil:
			for _, sub := range ent.Block.Entries {
				switch {
				case keyIs(sub, "ruling_party"):
					data.Set("set_politics_ruling_party", pyStrip(pyStr(entVal(sub))))
				case keyIs(sub, "last_election"):
					data.Set("set_politics_last_election",
						pyStripChar(pyStrip(pyStr(entVal(sub))), '"'))
				case keyIs(sub, "election_frequency"):
					v, exc := coerceInt(entVal(sub), big.NewInt(0))
					if exc != nil {
						return nil, nil, exc
					}
					data.Set("set_politics_election_frequency", v)
				case keyIs(sub, "elections_allowed"):
					data.Set("set_politics_elections_allowed",
						pyLower(pyStrip(pyStr(entVal(sub)))) == "yes")
				}
			}
			sawAnyKnown = true
		case keyIs(ent, "set_popularities") && ent.Block != nil:
			for _, sub := range ent.Block.Entries {
				if sub.Key == nil {
					continue
				}
				v, exc := coerceInt(entVal(sub), big.NewInt(0))
				if exc != nil {
					return nil, nil, exc
				}
				if popularities, ok := data.MustGet("set_popularities").(*PyDict); ok {
					popularities.Set(pyStrip(*sub.Key), v)
				}
				switch *sub.Key {
				case "democratic", "communism", "fascism", "neutrality":
					data.Set("set_popularities_"+*sub.Key, v)
				}
			}
			sawAnyKnown = true
		case keyIs(ent, "set_research_slots"):
			v, exc := coerceInt(entVal(ent), big.NewInt(0))
			if exc != nil {
				return nil, nil, exc
			}
			data.Set("set_research_slots", v)
			sawAnyKnown = true
		case keyIs(ent, "set_convoys"):
			v, exc := coerceInt(entVal(ent), big.NewInt(0))
			if exc != nil {
				return nil, nil, exc
			}
			data.Set("set_convoys", v)
			sawAnyKnown = true
		case keyIs(ent, "set_stability"):
			v, exc := coerceFloat(entVal(ent), -1.0)
			if exc != nil {
				return nil, nil, exc
			}
			data.Set("set_stability", v)
			sawAnyKnown = true
		case keyIs(ent, "set_war_support"):
			v, exc := coerceFloat(entVal(ent), -1.0)
			if exc != nil {
				return nil, nil, exc
			}
			data.Set("set_war_support", v)
			sawAnyKnown = true
		case keyIs(ent, "add_ideas") && ent.Block != nil:
			for _, bv := range ppBareValues(ent.Block.Entries) {
				data.appendTo("add_ideas_extra", bv)
			}
			sawAnyKnown = true
		case keyIs(ent, "remove_ideas") && ent.Block != nil:
			for _, bv := range ppBareValues(ent.Block.Entries) {
				data.appendTo("remove_ideas", bv)
			}
			sawAnyKnown = true
		case keyIs(ent, "set_technology") && ent.Block != nil:

			for _, sub := range ent.Block.Entries {
				if sub.Key != nil && *sub.Key != "" {
					if tid := pyStrip(*sub.Key); tid != "" {
						data.appendTo("set_technology_ids", tid)
					}
				}
			}
			sawAnyKnown = true
		case keyIs(ent, "complete_national_focus"):
			if val := pyStrip(pyStr(entVal(ent))); val != "" {
				data.appendTo("complete_national_focus_ids", val)
			}
			sawAnyKnown = true
		case keyIs(ent, "set_cosmetic_tag"):
			data.Set("set_cosmetic_tag", pyStrip(pyStr(entVal(ent))))
			sawAnyKnown = true
		case keyIs(ent, "set_oob") || keyIs(ent, "set_air_oob") || keyIs(ent, "set_naval_oob"):
			data.Set(*ent.Key, pyStripChar(pyStrip(pyStr(entVal(ent))), '"'))
			sawAnyKnown = true
		case keyIs(ent, "give_guarantee"):
			if val := pyStripChar(pyStrip(pyStr(entVal(ent))), '"'); val != "" {
				data.appendTo("give_guarantee_tags", val)
			}
			sawAnyKnown = true
		case keyIs(ent, "add_to_faction"):

			if atf := pyStripChar(pyStrip(pyStr(entVal(ent))), '"'); atf != "" {
				data.Set("add_to_faction", atf)
				tags, _ := data.MustGet("add_to_faction_tags").([]any)
				if !containsAnyStr(tags, atf) {
					data.appendTo("add_to_faction_tags", atf)
				}
			}
			sawAnyKnown = true
		case keyIs(ent, "set_autonomy") && ent.Block != nil:
			target := ""
			if v, ok := ppScalar(ent.Block.Entries, "target"); ok {
				target = pyStrip(pyStr(v))
			}
			astate := ""
			if v, ok := ppScalar(ent.Block.Entries, "autonomous_state"); ok {
				astate = pyStrip(pyStr(v))
			}
			if target != "" || astate != "" {
				d := NewPyDict()
				d.Set("target", target)
				d.Set("autonomous_state", astate)
				data.appendTo("set_autonomy", d)
			}
			sawAnyKnown = true
		case keyIs(ent, "diplomatic_relation") && ent.Block != nil:
			country := ""
			if v, ok := ppScalar(ent.Block.Entries, "country"); ok {
				country = pyStrip(pyStr(v))
			}
			relation := ""
			if v, ok := ppScalar(ent.Block.Entries, "relation"); ok {
				relation = pyStrip(pyStr(v))
			}
			activeRaw := "yes"
			if v, ok := ppScalar(ent.Block.Entries, "active"); ok {
				activeRaw = pyStr(v)
			}
			activeRaw = pyLower(pyStrip(activeRaw))
			if country != "" || relation != "" {
				d := NewPyDict()
				d.Set("country", country)
				d.Set("relation", relation)
				d.Set("active", activeRaw != "no")
				data.appendTo("diplomatic_relation", d)
			}
			sawAnyKnown = true
		case keyIs(ent, "set_country_flag"):

			if ent.Block != nil {
				if innerFlag, ok := ppScalar(ent.Block.Entries, "flag"); ok && truthy(innerFlag) {
					if flagStr := pyStrip(pyStr(innerFlag)); flagStr != "" {
						data.appendTo("startup_flags", flagStr)
					}
				}
			} else {
				if flagStr := pyStrip(pyStr(entVal(ent))); flagStr != "" {
					data.appendTo("startup_flags", flagStr)
				}
			}
			sawAnyKnown = true
		case keyIs(ent, "retire_country_leader") || keyIs(ent, "create_country_leader"):

			sawAnyKnown = true
			continue
		case ent.Key != nil:

			if s, e, ok := importsvc.TopLevelLineBounds(srcLines, ast.Entries, idx); ok {
				rawBounds = append(rawBounds, [2]int{s, e})
			} else {
				rawFallback = append(rawFallback,
					importsvc.ToScriptEntries([]*paradox.Entry{ent}, 0))
			}
			summary.AddPreserved("history_file." + *ent.Key)
			sawAnyKnown = true
		}
	}

	var rawParts []string
	if fromBounds := importsvc.EmitRawScriptFromBounds(srcLines, rawBounds); fromBounds != "" {
		rawParts = append(rawParts, fromBounds)
	}
	if len(rawFallback) > 0 {
		rawParts = append(rawParts, pyRStrip(strings.Join(rawFallback, "\n")))
	}
	if len(rawParts) > 0 {
		var kept []string
		for _, p := range rawParts {
			if pyStrip(p) != "" {
				kept = append(kept, p)
			}
		}
		data.Set("raw_script", pyRStrip(strings.Join(kept, "\n\n")))
	}

	if !sawAnyKnown {
		return []*PyDict{}, summary, nil
	}

	summary.Total = 1
	return []*PyDict{data}, summary, nil
}
