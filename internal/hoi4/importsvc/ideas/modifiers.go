package ideas

import (
	"math/big"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func ImportOpinionModifiers(text string) ([]*Dict, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("opinion_modifier")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, nil, err
	}
	expandAtConstants(ast.Entries)

	omEntries, ok := ppBlock(ast.Entries, "opinion_modifiers")
	if !ok {
		return []*Dict{}, summary, nil
	}

	results := []*Dict{}
	for _, entry := range omEntries {
		if entry.Block == nil {

			continue
		}
		modifierID := keyStr(entry)
		body := NewDict()
		rawExtras := NewDict()
		for _, child := range entry.Block.Entries {
			ck := keyStr(child)
			if child.Block != nil {
				accumulateRawExtra(rawExtras, ck, blockText(child.Block))
				summary.AddPreserved(ck)
			} else {
				body.Set(ck, coerceDotNum(importsvc.PyScalarStr(child.Scalar)))
			}
		}
		rec := NewDict()
		rec.Set("modifier_id", modifierID)
		rec.Set("body", body)
		rec.Set("raw_extras", rawExtras)
		results = append(results, rec)
	}

	summary.Total = len(results)
	return results, summary, nil
}

func ImportDynamicModifiers(text string) ([]*Dict, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("dynamic_modifier")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, nil, err
	}
	expandAtConstants(ast.Entries)

	results := []*Dict{}
	for _, entry := range ast.Entries {
		if !hasKey(entry) || entry.Block == nil {
			continue
		}
		dmID := *entry.Key
		icon := ""
		enable := ""
		removeTrigger := ""
		modifier := NewDict()
		rawExtras := NewDict()
		for _, child := range entry.Block.Entries {
			ck := keyOrEmpty(child)
			if ck == "" {
				continue
			}
			if child.Block != nil {
				bt := blockText(child.Block)
				switch ck {
				case "enable":
					enable = bt
				case "remove_trigger":
					removeTrigger = bt
				default:
					accumulateRawExtra(rawExtras, ck, bt)
					summary.AddPreserved(ck)
				}
			} else if ck == "icon" {
				icon = importsvc.PyScalarStr(child.Scalar)
			} else {
				modifier.Set(ck, coerceDotNum(importsvc.PyScalarStr(child.Scalar)))
			}
		}
		rec := NewDict()
		rec.Set("dm_id", dmID)
		rec.Set("icon", icon)
		rec.Set("enable", enable)
		rec.Set("remove_trigger", removeTrigger)
		rec.Set("modifier", modifier)
		rec.Set("raw_extras", rawExtras)
		results = append(results, rec)
	}

	summary.Total = len(results)
	return results, summary, nil
}

var autonomyAIKeys = map[string]bool{
	"ai_subject_wants_higher":    true,
	"ai_overlord_wants_lower":    true,
	"ai_overlord_wants_garrison": true,
}

func ImportAutonomyStates(text string) ([]*Dict, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("autonomy_state")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, nil, err
	}
	expandAtConstants(ast.Entries)

	results := []*Dict{}
	for _, entry := range ast.Entries {
		if keyStr(entry) != "autonomy_state" || entry.Block == nil {
			continue
		}
		autonomyID := ""
		isPuppet := false
		isDefault := false
		var manpowerInfluence any
		var minFreedomLevel any = big.NewInt(0)
		rule := ""
		aiWeights := NewDict()
		modifier := NewDict()
		rawExtras := NewDict()
		for _, child := range entry.Block.Entries {
			ck := keyOrEmpty(child)
			if ck == "" {
				continue
			}
			if child.Block != nil {
				bt := blockText(child.Block)
				switch {
				case ck == "rule":
					rule = bt
				case ck == "modifier":

					for _, m := range child.Block.Entries {
						mk := keyOrEmpty(m)
						if mk == "" {
							continue
						}
						if m.Block != nil {
							accumulateRawExtra(rawExtras, mk, blockText(m.Block))
						} else {
							modifier.Set(mk, coerceDotNum(importsvc.PyScalarStr(m.Scalar)))
						}
					}
				case autonomyAIKeys[ck]:
					aiWeights.Set(ck, bt)
				default:
					accumulateRawExtra(rawExtras, ck, bt)
					summary.AddPreserved(ck)
				}
			} else {
				sv := importsvc.PyScalarStr(child.Scalar)
				switch ck {
				case "id":
					autonomyID = sv
				case "is_puppet":
					isPuppet = yesTrue1(sv)
				case "min_freedom_level":

					if f, fok := pyFloat(sv); fok {
						z, ckind := pyRoundToInt(f * 1000)
						switch ckind {
						case coerceOverflowError:
							return nil, nil, ErrOverflow
						case coerceOK:
							minFreedomLevel = z
						default:
							accumulateRawExtra(rawExtras, ck, sv)
						}
					} else {
						accumulateRawExtra(rawExtras, ck, sv)
					}
				case "default":
					isDefault = yesTrue1(sv)
				case "manpower_influence":
					if f, fok := pyFloat(sv); fok {
						manpowerInfluence = f
					} else {
						accumulateRawExtra(rawExtras, ck, sv)
					}
				default:

					accumulateRawExtra(rawExtras, ck, sv)
					summary.AddPreserved(ck)
				}
			}
		}
		if autonomyID == "" {

			continue
		}
		rec := NewDict()
		rec.Set("autonomy_id", autonomyID)
		rec.Set("is_puppet", isPuppet)
		rec.Set("is_default", isDefault)
		rec.Set("manpower_influence", manpowerInfluence)
		rec.Set("min_freedom_level", minFreedomLevel)
		rec.Set("rule", rule)
		rec.Set("ai_weights", aiWeights)
		rec.Set("modifier", modifier)
		rec.Set("raw_extras", rawExtras)
		results = append(results, rec)
	}

	summary.Total = len(results)
	return results, summary, nil
}
