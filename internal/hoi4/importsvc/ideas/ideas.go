package ideas

import (
	"math/big"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

var designerTypes = map[string]bool{
	"tank_manufacturer": true, "naval_manufacturer": true,
	"aircraft_manufacturer": true, "materiel_manufacturer": true,
	"industrial_concern": true,
}

var ideaKnownFields = map[string]bool{
	"picture": true, "modifier": true, "removal_cost": true, "allowed": true,
	"allowed_civil_war": true, "designer": true, "research_bonus": true,
	"traits": true, "ai_will_do": true, "name": true,
	"available": true, "visible": true, "allowed_to_remove": true,
	"cancel": true, "do_effect": true, "on_add": true, "on_remove": true,
	"cancel_if_invalid": true,
	"targeted_modifier": true, "equipment_bonus": true, "rule": true,
	"level": true, "ledger": true,
	"cost": true,
}

var ideaScriptBlockKeys = map[string]bool{
	"available": true, "visible": true, "allowed_to_remove": true,
	"cancel": true, "do_effect": true, "on_add": true, "on_remove": true,
}

func ImportIdeas(text string) ([]*Dict, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("idea")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, nil, err
	}
	expandAtConstants(ast.Entries)

	ideasEntries, ok := ppBlock(ast.Entries, "ideas")
	if !ok {
		summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
			"no_content_ideas", nil,
			"A file in the Ideas folder had no ideas in it — skipped.",
		))
		return []*Dict{}, summary, nil
	}

	results := []*Dict{}

	for _, catEntry := range ideasEntries {
		if catEntry.Block == nil {
			continue
		}
		ideaType := keyStr(catEntry)
		categoryIsDesigner := designerTypes[ideaType]

		categoryFlags := NewDict()
		for _, child := range catEntry.Block.Entries {
			if child.Block != nil {
				continue
			}

			categoryFlags.Set(keyStr(child), child.Scalar)
		}

		for _, ideaEntry := range catEntry.Block.Entries {
			if ideaEntry.Block == nil {
				continue
			}
			ideaDict, err := mapIdea(keyStr(ideaEntry), ideaType,
				ideaEntry.Block.Entries, categoryIsDesigner, summary)
			if err != nil {
				return nil, nil, err
			}
			if categoryFlags.Len() > 0 {
				ideaDict.Set("category_flags", categoryFlags.Copy())
			}
			results = append(results, ideaDict)
		}
	}

	summary.Total = len(results)
	return results, summary, nil
}

func mapIdea(ideaID, ideaType string, block []*paradox.Entry,
	isDesigner bool, summary *importsvc.ImportSummary) (*Dict, error) {

	modifier := NewDict()
	researchBonus := NewDict()
	rawExtras := NewDict()
	equipmentBonus := NewDict()
	ruleDict := NewDict()

	result := NewDict()
	result.Set("idea_id", ideaID)
	result.Set("idea_type", ideaType)
	result.Set("name", ideaID)
	result.Set("picture", "")
	result.Set("modifier", modifier)
	result.Set("ai_will_do", "")
	result.Set("removal_cost", big.NewInt(-1))
	result.Set("allowed", "")
	result.Set("allowed_civil_war", "")
	result.Set("is_designer", isDesigner)
	result.Set("research_bonus", researchBonus)
	result.Set("designer_traits", []any{})
	result.Set("raw_extras", rawExtras)
	result.Set("available", "")
	result.Set("visible", "")
	result.Set("allowed_to_remove", "")
	result.Set("cancel", "")
	result.Set("do_effect", "")
	result.Set("on_add", "")
	result.Set("on_remove", "")
	result.Set("cancel_if_invalid", true)
	result.Set("targeted_modifier", []any{})
	result.Set("equipment_bonus", equipmentBonus)
	result.Set("rule", ruleDict)
	result.Set("level", big.NewInt(0))
	result.Set("ledger", "")
	result.Set("cost", big.NewInt(0))

	var allowedEntries []*paradox.Entry
	hasAllowed := false

	for _, entry := range block {
		key := keyStr(entry)
		isBlock := entry.Block != nil

		switch {
		case key == "picture":

			result.Set("picture", valueStr(entry))

		case key == "name":

			locKey := pyStrip(valueStr(entry))
			if locKey != "" && locKey != ideaID {
				result.Set("idea_token", locKey)
				rawExtras.Set("name", locKey)
			}

		case key == "removal_cost":
			z, ck := intFloatStr(valueStr(entry))
			switch ck {
			case coerceOverflowError:
				return nil, ErrOverflow
			case coerceOK:
				result.Set("removal_cost", z)
			default:
				result.Set("removal_cost", big.NewInt(-1))
			}

		case key == "modifier" && isBlock:
			for _, modEntry := range entry.Block.Entries {
				mk := keyStr(modEntry)
				if modEntry.Block != nil {

					modifier.Set(mk, blockText(modEntry.Block))
					continue
				}
				sv := importsvc.PyScalarStr(modEntry.Scalar)
				if f, fok := pyFloat(sv); fok {
					modifier.Set(mk, f)
				} else {
					modifier.Set(mk, sv)
				}
			}

		case key == "allowed" && isBlock:
			allowedEntries = entry.Block.Entries
			hasAllowed = true
			result.Set("allowed", blockText(entry.Block))

		case key == "allowed_civil_war" && isBlock:
			result.Set("allowed_civil_war", blockText(entry.Block))

		case key == "research_bonus" && isBlock:
			for _, rb := range entry.Block.Entries {
				if rb.Block != nil {
					continue
				}
				if f, fok := pyFloat(importsvc.PyScalarStr(rb.Scalar)); fok {
					researchBonus.Set(keyStr(rb), f)
				}
			}

		case key == "traits" && isBlock:
			traits := []any{}
			for _, t := range entry.Block.Entries {
				if t.Key == nil {
					traits = append(traits, valueStr(t))
				} else {
					traits = append(traits, *t.Key)
				}
			}
			result.Set("designer_traits", traits)

		case key == "designer":
			result.Set("is_designer", true)

		case key == "ai_will_do" && isBlock:
			result.Set("ai_will_do", blockText(entry.Block))

		case ideaScriptBlockKeys[key] && isBlock:
			result.Set(key, blockText(entry.Block))

		case key == "cancel_if_invalid":
			v := pyLower(pyStrip(valueStr(entry)))
			result.Set("cancel_if_invalid", v != "no" && v != "false" && v != "0")

		case key == "level":
			z, ck := intFloatStr(valueStr(entry))
			switch ck {
			case coerceOverflowError:
				return nil, ErrOverflow
			case coerceOK:
				result.Set("level", z)
			default:
				result.Set("level", big.NewInt(0))
			}

		case key == "cost":
			z, ck := intFloatStr(valueStr(entry))
			switch ck {
			case coerceOverflowError:
				return nil, ErrOverflow
			case coerceOK:
				result.Set("cost", z)
			default:
				result.Set("cost", big.NewInt(0))
			}

		case key == "ledger":
			result.Set("ledger", pyStrip(valueStr(entry)))

		case key == "targeted_modifier" && isBlock:
			tagVal := ""
			mods := NewDict()
			for _, sub := range entry.Block.Entries {
				sk := keyStr(sub)
				if sk == "tag" {

					tagVal = pyUpper(pyStrip(valueStr(sub)))
				} else if sub.Block != nil {
					mods.Set(sk, blockText(sub.Block))
				} else {
					sv := importsvc.PyScalarStr(sub.Scalar)
					if f, fok := pyFloat(sv); fok {
						mods.Set(sk, f)
					} else {
						mods.Set(sk, sv)
					}
				}
			}
			if tagVal != "" {
				listAny, _ := result.Get("targeted_modifier")
				list := listAny.([]any)
				var existing *Dict
				for _, item := range list {
					if d, dok := item.(*Dict); dok {
						if tv, tok := d.Get("tag"); tok {
							if s, sok := tv.(string); sok && s == tagVal {
								existing = d
								break
							}
						}
					}
				}
				if existing != nil {

					merged := NewDict()
					if mv, mok := existing.Get("modifier"); mok {
						if md, dok := mv.(*Dict); dok {
							merged = md.Copy()
						}
					}
					for _, k := range mods.Keys() {
						v, _ := mods.Get(k)
						merged.Set(k, v)
					}
					existing.Set("modifier", merged)
				} else {
					nd := NewDict()
					nd.Set("tag", tagVal)
					nd.Set("modifier", mods)
					result.Set("targeted_modifier", append(list, nd))
				}
			}

		case key == "equipment_bonus" && isBlock:
			for _, sub := range entry.Block.Entries {
				sk := keyStr(sub)
				if sk == "instant" {

					sv := pyLower(pyStrip(valueStr(sub)))
					equipmentBonus.Set("instant", sv == "yes" || sv == "true" || sv == "1")
					continue
				}
				if sub.Block != nil {
					inner := NewDict()
					for _, be := range sub.Block.Entries {
						bk := keyStr(be)
						raw := valueStr(be)
						if f, fok := pyFloat(raw); fok {
							inner.Set(bk, f)
						} else {
							inner.Set(bk, raw)
						}
					}
					equipmentBonus.Set(sk, inner)
				}

			}

		case key == "rule" && isBlock:
			for _, sub := range entry.Block.Entries {
				sk := keyStr(sub)
				sv := pyStrip(valueStr(sub))
				lower := pyLower(sv)
				switch {
				case lower == "yes" || lower == "true":
					ruleDict.Set(sk, true)
				case lower == "no" || lower == "false":
					ruleDict.Set(sk, false)
				default:
					if f, fok := pyFloat(sv); fok {
						ruleDict.Set(sk, f)
					} else {
						ruleDict.Set(sk, sv)
					}
				}
			}

		case ideaKnownFields[key]:

		default:

			if isBlock {
				accumulateRawExtra(rawExtras, key, blockText(entry.Block))
			} else {
				accumulateRawExtra(rawExtras, key, importsvc.PyScalarStr(entry.Scalar))
			}
			summary.AddPreserved(key)
		}
	}

	var scopeBlocks [][]*paradox.Entry
	if hasAllowed {
		scopeBlocks = [][]*paradox.Entry{allowedEntries}
	}
	result.Set("_scope_verdict", importsvc.ClassifyScope(scopeBlocks))

	return result, nil
}
