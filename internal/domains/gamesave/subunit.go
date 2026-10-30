package gamesave

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"haganeworks/internal/domains/military"
	"haganeworks/internal/project"
)

const subUnitTable = "sub_units"

type subUnitSaveIndex interface {
	SubUnitSource(id string) (string, []byte, bool)
}

type subUnitSaveInput struct {
	ID    string             `json:"id"`
	Stats map[string]float64 `json:"stats"`
	Need  map[string]float64 `json:"need"`
}

const subUnitRawFileNote = "游戏界面营的数值(整份文件收进项目)"

var subUnitStatKeys = map[string]bool{
	"combat_width": true, "max_strength": true, "max_organisation": true, "default_morale": true, "manpower": true,
	"training_time": true, "suppression": true, "weight": true, "supply_consumption": true,
}

var errSubUnitMissing = errors.New("the battalion is not in its units file")

func subUnitEdit(text, id string, stats, need map[string]float64) (string, error) {
	words := scriptWords(text)
	for i := 0; i+2 < len(words); i++ {
		if !strings.EqualFold(words[i].text, "sub_units") || words[i+1].text != "=" || words[i+2].text != "{" {
			continue
		}
		unitsClose := closeOf(words, i+2)
		for j := i + 3; j+2 < unitsClose; j++ {
			if words[j].depth != words[i+2].depth+1 || !strings.EqualFold(words[j].text, id) || words[j+1].text != "=" || words[j+2].text != "{" {
				continue
			}
			open, close := j+2, closeOf(words, j+2)
			if close < 0 {
				return "", errSubUnitMissing
			}
			values := map[string]*float64{}
			for key, value := range stats {
				values[key] = &value
			}
			edits := setNumbers(text, words, open, close, values, blockInner(text, words, open, close))
			if len(need) > 0 {
				key, needClose, found := childBlock(words, open, close, "need")
				if !found {
					return "", errors.New("the battalion needs no equipment")
				}
				needOpen := key + 2
				have := assignments(words, needOpen, needClose)
				amounts := map[string]*float64{}
				for equipment, amount := range need {
					if _, listed := have[equipment]; !listed {
						return "", fmt.Errorf("the battalion does not need %s", equipment)
					}
					amounts[equipment] = &amount
				}
				edits = append(edits, setNumbers(text, words, needOpen, needClose, amounts, blockInner(text, words, needOpen, needClose))...)
			}
			return applyEdits(text, edits), nil
		}
	}
	return "", errSubUnitMissing
}

func saveSubUnit(st *store, index subUnitSaveIndex, in subUnitSaveInput) (map[string]any, error) {
	id := strings.ToLower(strings.TrimSpace(in.ID))
	if id == "" || strings.ContainsAny(id, " \t\r\n{}=\"#/\\") {
		return nil, saveError{422, "a battalion is required"}
	}
	if len(in.Stats) == 0 && len(in.Need) == 0 {
		return nil, saveError{422, "nothing to change"}
	}
	for key, value := range in.Stats {
		if !subUnitStatKeys[key] || math.IsNaN(value) || value < 0 || value > 1e6 {
			return nil, saveError{422, fmt.Sprintf("%s is not a battalion number the panel sets, or is out of range", key)}
		}
	}
	need := map[string]float64{}
	for equipment, amount := range in.Need {
		equipment = strings.ToLower(strings.TrimSpace(equipment))
		if equipment == "" || strings.ContainsAny(equipment, " \t\r\n{}=\"#") || math.IsNaN(amount) || amount < 0 || amount > 1e6 || amount != math.Trunc(amount) {
			return nil, saveError{422, "equipment amounts are whole numbers from 0"}
		}
		need[equipment] = amount
	}
	if out, handled, err := saveSubUnitRow(st, id, in.Stats, need); handled || err != nil {
		return out, err
	}
	file, body, ok := index.SubUnitSource(id)
	if !ok {
		return nil, saveError{404, "the battalion's units file is not available"}
	}
	text, err := subUnitEdit(string(body), id, in.Stats, need)
	if err != nil {
		return nil, saveError{422, err.Error()}
	}
	dest, err := sanitizedPath(file)
	if err != nil {
		return nil, err
	}
	if err := gateText(dest, text); err != nil {
		return nil, err
	}
	created, err := st.raw.put(dest, text, subUnitRawFileNote)
	if err != nil {
		return nil, err
	}
	return map[string]any{"file": dest, "imported": created}, nil
}

type subUnitRow struct {
	SubUnitID  string         `json:"sub_unit_id"`
	Source     string         `json:"source"`
	SourceFile string         `json:"source_file"`
	Ordering   int            `json:"ordering"`
	UnitTypes  []string       `json:"unit_types"`
	Categories []string       `json:"categories"`
	Need       map[string]any `json:"need"`
	Stats      map[string]any `json:"stats"`
}

func saveSubUnitRow(st *store, id string, stats, need map[string]float64) (map[string]any, bool, error) {
	rows := st.sortedRows(subUnitTable, nil, func(a, b project.Row) bool {
		if x, y := text(a, "source_file"), text(b, "source_file"); x != y {
			return x < y
		}
		if x, y := number(a["ordering"]), number(b["ordering"]); x != y {
			return x < y
		}
		return text(a, "sub_unit_id") < text(b, "sub_unit_id")
	})
	for _, row := range rows {
		var current subUnitRow
		decodeColumn(row, &current)
		if current.Source == "" {
			current.Source = "created"
		}
		if !strings.EqualFold(current.SubUnitID, id) || current.Source == "imported" {
			continue
		}

		if err := military.ValidateSubUnitClassification(current.UnitTypes, current.Categories); err != nil {
			return nil, true, saveError{422, err.Error()}
		}
		changes := map[string]any{}
		if len(stats) > 0 {
			merged := map[string]any{}
			for key, value := range current.Stats {
				merged[key] = value
			}
			for key, value := range stats {
				merged[key] = value
			}
			changes["stats"] = jsonValue(merged)
		}
		if len(need) > 0 {
			merged := map[string]any{}
			for key, value := range current.Need {
				merged[key] = value
			}
			for equipment, amount := range need {
				if _, listed := current.Need[equipment]; !listed {
					return nil, true, saveError{422, fmt.Sprintf("the battalion does not need %s", equipment)}
				}
				merged[equipment] = amount
			}
			changes["need"] = jsonValue(merged)
		}
		rowID := project.Int64(row["id"])
		if _, err := st.update(subUnitTable, rowID, changes); err != nil {
			return nil, true, err
		}
		return map[string]any{"sub_unit": rowID, "imported": false}, true, nil
	}
	return nil, false, nil
}

func (h *handlers) subUnitSave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in subUnitSaveInput
	if !decodeBody(w, r, 1<<16, &in, "invalid battalion payload") {
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}
	out, err := write(p, []string{rawFilesTable, subUnitTable}, func(st *store) (map[string]any, error) {
		out, err := saveSubUnit(st, index, in)
		return st.raw.withUndo(out), err
	})
	h.finish(w, p, out, err, "sub unit save", true)
}
