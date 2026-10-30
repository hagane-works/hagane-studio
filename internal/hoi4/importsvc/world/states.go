package world

import (
	"math/big"
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

var provinceIDRE = regexp.MustCompile(`\A\p{Nd}+\z`)

func splitStateBuildings(buildingsBlock *paradox.Block) (*PyDict, []*paradox.Entry, *PyRaise) {
	stateInts := NewPyDict()
	var provinceEntries []*paradox.Entry
	for _, e := range buildingsBlock.Entries {
		if e.Key == nil {
			provinceEntries = append(provinceEntries, e)
			continue
		}
		if e.Block != nil {
			provinceEntries = append(provinceEntries, e)
		} else {
			v, exc := coerceInt(scalarAny(e.Scalar), big.NewInt(0))
			if exc != nil {
				return nil, nil, exc
			}
			stateInts.Set(*e.Key, v)
		}
	}
	return stateInts, provinceEntries, nil
}

func isSimpleProvinceBlock(entries []*paradox.Entry) bool {
	for _, sub := range entries {
		if sub.Key == nil || sub.Block != nil {
			return false
		}
	}
	return true
}

func DeserializeProvinceBuildings(buildingsText string) (*PyDict, error) {
	text := pyStrip(buildingsText)
	if text == "" {
		return NewPyDict(), nil
	}
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return NewPyDict(), nil
	}
	out := NewPyDict()
	for _, ent := range ast.Entries {
		if ent.Key == nil || ent.Block == nil {
			continue
		}
		if !provinceIDRE.MatchString(pyStrip(*ent.Key)) {
			continue
		}
		if !isSimpleProvinceBlock(ent.Block.Entries) {
			continue
		}
		block := NewPyDict()
		for _, sub := range ent.Block.Entries {
			v, exc := coerceInt(scalarAny(sub.Scalar), big.NewInt(0))
			if exc != nil {
				return nil, exc
			}
			block.Set(*sub.Key, v)
		}
		out.Set(pyStrip(*ent.Key), block)
	}
	return out, nil
}

type PBItem struct {
	Key string
	Val any
}

type PBBlockInput struct {
	PID    string
	IsDict bool
	Items  []PBItem
}

func SerializeProvinceBuildings(pb []PBBlockInput, existingText string) (string, error) {

	var preservedChunks []string
	text := pyStrip(existingText)
	if text != "" {
		ast, _, err := paradox.Parse(text)
		if err == nil {
			for _, ent := range ast.Entries {
				if ent.Key != nil && ent.Block != nil &&
					provinceIDRE.MatchString(pyStrip(*ent.Key)) &&
					isSimpleProvinceBlock(ent.Block.Entries) {
					continue
				}
				preservedChunks = append(preservedChunks, pyRStripChar(
					importsvc.BlockToText(&paradox.Block{Entries: []*paradox.Entry{ent}}, 0), '\n'))
			}
		} else {

			preservedChunks = append(preservedChunks, text)
		}
	}

	var structChunks []string
	for _, blk := range pb {
		if !blk.IsDict {
			continue
		}
		var bodyLines []string
		for _, kv := range blk.Items {
			if pyStrip(kv.Key) == "" {
				continue
			}
			iv, exc := coerceInt(kv.Val, big.NewInt(0))
			if exc != nil {
				return "", exc
			}
			bodyLines = append(bodyLines, "\t"+kv.Key+" = "+iv.(*big.Int).String())
		}
		if len(bodyLines) == 0 {
			continue
		}
		structChunks = append(structChunks, blk.PID+" = {\n"+strings.Join(bodyLines, "\n")+"\n}")
	}

	allChunks := append(structChunks, preservedChunks...)
	return strings.Join(allChunks, "\n"), nil
}

func appendHistoryRaw(rawExtras *PyDict, key string, ent *paradox.Entry) {
	rk := "history." + key
	var piece any
	if ent.Block != nil {
		piece = importsvc.BlockToText(ent.Block, 0)
	} else {
		piece = importsvc.ScalarToText(ent)
	}
	rawExtras.Accumulate(rk, piece)
}

func mapStateHistory(historyBlock *paradox.Block, data, rawExtras *PyDict, summary *importsvc.ImportSummary) *PyRaise {
	var provinceBuildingEntries []*paradox.Entry
	for _, ent := range historyBlock.Entries {
		switch {
		case keyIs(ent, "owner"):
			data.Set("owner", pyStripChar(pyStrip(pyStr(entVal(ent))), '"'))
		case keyIs(ent, "controller"):
			data.Set("controller", pyStripChar(pyStrip(pyStr(entVal(ent))), '"'))
		case keyIs(ent, "add_core_of"):
			if val := pyStripChar(pyStrip(pyStr(entVal(ent))), '"'); val != "" {
				data.appendTo("add_core_of", val)
			}
		case keyIs(ent, "add_claim_by"):
			if val := pyStripChar(pyStrip(pyStr(entVal(ent))), '"'); val != "" {
				data.appendTo("add_claim_by", val)
			}
		case keyIs(ent, "remove_core_of"):
			if val := pyStripChar(pyStrip(pyStr(entVal(ent))), '"'); val != "" {
				data.appendTo("remove_core_of", val)
			}
		case keyIs(ent, "set_province_controller") && ent.Block != nil:
			var pidV any
			ctrlV := ""
			ctrlSet := false
			for _, sub := range ent.Block.Entries {
				if keyIs(sub, "province") {
					v, exc := coerceInt(entVal(sub), nil)
					if exc != nil {
						return exc
					}
					pidV = v
				} else if keyIs(sub, "controller") {
					ctrlV = pyStripChar(pyStrip(pyStr(entVal(sub))), '"')
					ctrlSet = true
				}
			}
			if pidV != nil && ctrlSet && ctrlV != "" {
				d := NewPyDict()
				d.Set("province_id", pidV)
				d.Set("controller", ctrlV)
				data.appendTo("set_province_controller", d)
			} else {

				appendHistoryRaw(rawExtras, "set_province_controller", ent)
			}
		case keyIs(ent, "victory_points") && ent.Block != nil:
			vals := ppBareValues(ent.Block.Entries)
			if len(vals) >= 2 {
				pid, exc := coerceInt(vals[0], vals[0])
				if exc != nil {
					return exc
				}
				value, exc := coerceInt(vals[1], vals[1])
				if exc != nil {
					return exc
				}
				d := NewPyDict()
				d.Set("province_id", pid)
				d.Set("value", value)
				data.appendTo("victory_points", d)
			} else if len(vals) > 0 {

				appendHistoryRaw(rawExtras, "victory_points", ent)
			}
		case keyIs(ent, "buildings") && ent.Block != nil:
			stateInts, provinceEntries, exc := splitStateBuildings(ent.Block)
			if exc != nil {
				return exc
			}

			buildings := data.MustGet("buildings").(*PyDict)
			for _, k := range stateInts.Keys() {
				v, _ := stateInts.Get(k)
				buildings.Set(k, v)
			}
			provinceBuildingEntries = append(provinceBuildingEntries, provinceEntries...)
		case ent.Key != nil:

			appendHistoryRaw(rawExtras, *ent.Key, ent)
			summary.AddPreserved("state.history." + *ent.Key)
		}
	}

	if len(provinceBuildingEntries) > 0 {
		rawExtras.Set("buildings", importsvc.BlockToText(
			&paradox.Block{Entries: provinceBuildingEntries}, 0))
		summary.AddPreserved("state.history.buildings.province_level")
	}
	return nil
}

func mapOneState(stateBlock *paradox.Block, summary *importsvc.ImportSummary) (*PyDict, *PyRaise) {
	data := NewPyDict()
	data.Set("state_id", nil)
	data.Set("name", "")
	data.Set("manpower", big.NewInt(0))
	data.Set("state_category", "")
	data.Set("resources", NewPyDict())
	data.Set("provinces", []any{})
	data.Set("owner", "")
	data.Set("controller", nil)
	data.Set("add_core_of", []any{})
	data.Set("add_claim_by", []any{})
	data.Set("remove_core_of", []any{})
	data.Set("victory_points", []any{})
	data.Set("set_province_controller", []any{})
	data.Set("buildings", NewPyDict())
	data.Set("local_supplies", nil)
	data.Set("buildings_max_level_factor", nil)
	data.Set("impassable", nil)
	data.Set("impassable_ignored_links", nil)

	data.Set("force_link_ownership_to", nil)
	data.Set("raw_extras", NewPyDict())
	rawExtras := NewPyDict()

	for _, ent := range stateBlock.Entries {
		switch {
		case keyIs(ent, "id"):
			v, exc := coerceInt(entVal(ent), nil)
			if exc != nil {
				return nil, exc
			}
			data.Set("state_id", v)
		case keyIs(ent, "name"):
			data.Set("name", pyStripChar(pyStrip(pyStr(entVal(ent))), '"'))
		case keyIs(ent, "manpower"):
			v, exc := coerceInt(entVal(ent), big.NewInt(0))
			if exc != nil {
				return nil, exc
			}
			data.Set("manpower", v)
		case keyIs(ent, "state_category"):

			data.Set("state_category", pyStripChar(pyStrip(pyStr(entVal(ent))), '"'))
		case keyIs(ent, "resources") && ent.Block != nil:
			res := NewPyDict()
			for _, sub := range ent.Block.Entries {
				if sub.Key != nil && sub.Block == nil {
					v, exc := coerceInt(scalarAny(sub.Scalar), big.NewInt(0))
					if exc != nil {
						return nil, exc
					}
					res.Set(*sub.Key, v)
				}
			}
			data.Set("resources", res)
		case keyIs(ent, "provinces") && ent.Block != nil:

			provinces := []any{}
			for _, bv := range ppBareValues(ent.Block.Entries) {
				v, exc := coerceInt(bv, bv)
				if exc != nil {
					return nil, exc
				}
				provinces = append(provinces, v)
			}
			data.Set("provinces", provinces)
		case keyIs(ent, "impassable"):
			data.Set("impassable", pyLower(pyStrip(pyStr(entVal(ent)))) == "yes")
		case keyIs(ent, "impassable_ignored_links"):
			if ent.Block != nil {
				data.Set("impassable_ignored_links", importsvc.BlockToText(ent.Block, 0))
			} else {
				data.Set("impassable_ignored_links", pyStrip(pyStr(entVal(ent))))
			}
		case keyIs(ent, "force_link_ownership_to") && ent.Block == nil:

			rawExtras.Accumulate("force_link_ownership_to", importsvc.ScalarToText(ent))
			summary.AddPreserved("state.force_link_ownership_to")
			v, exc := coerceInt(entVal(ent), nil)
			if exc != nil {
				return nil, exc
			}
			data.Set("force_link_ownership_to", v)
		case keyIs(ent, "local_supplies"):
			v, exc := coerceFloat(entVal(ent), nil)
			if exc != nil {
				return nil, exc
			}
			data.Set("local_supplies", v)
		case keyIs(ent, "buildings_max_level_factor"):
			v, exc := coerceFloat(entVal(ent), nil)
			if exc != nil {
				return nil, exc
			}
			data.Set("buildings_max_level_factor", v)
		case keyIs(ent, "history") && ent.Block != nil:
			if exc := mapStateHistory(ent.Block, data, rawExtras, summary); exc != nil {
				return nil, exc
			}
		case ent.Key != nil:

			if ent.Block != nil {
				rawExtras.Accumulate(*ent.Key, importsvc.BlockToText(ent.Block, 0))
			} else {
				rawExtras.Accumulate(*ent.Key, importsvc.ScalarToText(ent))
			}
			summary.AddPreserved("state." + *ent.Key)
		}
	}

	if data.MustGet("state_id") == nil {
		return nil, nil
	}

	if rawExtras.Len() > 0 {
		data.Set("raw_extras", rawExtras)
	}
	return data, nil
}

func ImportStateFiles(text, filename string) ([]*PyDict, *importsvc.ImportSummary) {
	summary := importsvc.NewImportSummary("state")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		summary.Warnings = append(summary.Warnings, importsvc.ParseSkipWarning(summary, filename))
		return []*PyDict{}, summary
	}
	expandAtConstants(ast)

	results := []*PyDict{}
	for _, stateEntry := range ppFindAll(ast.Entries, "state") {
		if stateEntry.Block == nil {
			continue
		}
		data, exc := mapOneState(stateEntry.Block, summary)
		if exc != nil {
			where := filename
			if where == "" {
				where = "a state file"
			}
			summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
				"state_skipped",
				[]importsvc.WarnParam{{Key: "file", Value: where}},
				"Skipped one state in "+where+" that couldn't be read — the rest imported fine.",
			))
			continue
		}
		if data != nil {
			results = append(results, data)
		}
	}

	summary.Total = len(results)
	return results, summary
}
