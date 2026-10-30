package gamesave

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/project"
)

const (
	fleetTable        = "oob_fleets"
	taskForceTable    = "oob_task_forces"
	airWingGroupTable = "oob_air_wing_groups"
	airWingTable      = "oob_air_wings"
)

type oobUnitSaveIndex interface {
	OOBSource(name string) ([]byte, bool)
}

type oobUnitSaveInput struct {
	Kind string `json:"kind"`
	OOB  string `json:"oob"`

	Ordinal    int      `json:"ordinal"`
	State      int      `json:"state"`
	StateIndex int      `json:"state_index"`
	RawName    string   `json:"raw_name"`
	Name       *string  `json:"name"`
	Amount     *float64 `json:"amount"`
}

const oobUnitRawFileNote = "游戏界面舰队 / 空军联队(整份文件收进项目)"

var errOOBUnitMissing = errors.New("the unit is not in the battle order file")

func taskForceEdit(text string, ordinal int, name string) (string, error) {
	words := scriptWords(text)
	seen := -1
	for i := 0; i+2 < len(words); i++ {
		if !strings.EqualFold(words[i].text, "task_force") || words[i+1].text != "=" || words[i+2].text != "{" {
			continue
		}
		if seen++; seen != ordinal {
			continue
		}
		open, close := i+2, closeOf(words, i+2)
		if close < 0 {
			break
		}
		if at, found := assignments(words, open, close)["name"]; found {
			return applyEdits(text, []edit{{words[at[1]].start, words[at[1]].end, strconv.Quote(name)}}), nil
		}
		inner := blockInner(text, words, open, close)
		return applyEdits(text, insertBeforeClose(text, words, close, []string{"name = " + strconv.Quote(name)}, inner)), nil
	}
	return "", errOOBUnitMissing
}

func airWingEdit(text string, ordinal int, name *string, amount *float64) (string, error) {
	words := scriptWords(text)
	seen := -1
	for i := 0; i+2 < len(words); i++ {
		if !strings.EqualFold(words[i].text, "air_wings") || words[i+1].text != "=" || words[i+2].text != "{" {
			continue
		}
		wingsClose := closeOf(words, i+2)
		for j := i + 3; j+2 < wingsClose; j++ {

			if words[j].depth != words[i+2].depth+2 || words[j+1].text != "=" || words[j+2].text != "{" {
				continue
			}
			open, close := j+2, closeOf(words, j+2)
			if close < 0 {
				break
			}
			if _, isWing := assignments(words, open, close)["amount"]; !isWing {
				j = close
				continue
			}
			if seen++; seen != ordinal {
				j = close
				continue
			}
			edits := []edit{}
			if amount != nil {
				edits = append(edits, setNumbers(text, words, open, close, map[string]*float64{"amount": amount}, blockInner(text, words, open, close))...)
			}
			if name != nil {
				if close+3 < len(words) && strings.EqualFold(words[close+1].text, "name") && words[close+2].text == "=" {
					edits = append(edits, edit{words[close+3].start, words[close+3].end, strconv.Quote(*name)})
				} else {
					edits = append(edits, edit{words[close].end, words[close].end, "\n" + indentOf(text, words[j].start) + "name = " + strconv.Quote(*name)})
				}
			}
			return applyEdits(text, edits), nil
		}
	}
	return "", errOOBUnitMissing
}

func saveOOBUnit(st *store, index oobUnitSaveIndex, in oobUnitSaveInput) (map[string]any, error) {
	oob := strings.Trim(strings.TrimSpace(in.OOB), `"`)
	if oob == "" || strings.ContainsAny(oob, " \t\r\n{}=\"#/\\") || in.Ordinal < 0 {
		return nil, saveError{422, "a battle order file and a unit are required"}
	}
	if in.Kind != "task_force" && in.Kind != "air_wing" {
		return nil, saveError{422, "a unit is a task force or an air wing"}
	}
	if in.Name == nil && in.Amount == nil {
		return nil, saveError{422, "nothing to change"}
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" || strings.ContainsAny(name, "\"\r\n{}") {
			return nil, saveError{422, "a name is required, without quotes or braces"}
		}
		in.Name = &name
	}
	if in.Amount != nil && (in.Kind != "air_wing" || *in.Amount < 1 || *in.Amount > 1000 || *in.Amount != float64(int(*in.Amount))) {
		return nil, saveError{422, "a wing has 1 to 1000 planes"}
	}
	if out, handled, err := saveOOBUnitRow(st, oob, in); handled || err != nil {
		return out, err
	}
	body, ok := index.OOBSource(oob)
	if !ok {
		return nil, saveError{404, "the battle order file's text is not available"}
	}
	var text string
	var err error
	if in.Kind == "task_force" {
		text, err = taskForceEdit(string(body), in.Ordinal, *in.Name)
	} else {
		text, err = airWingEdit(string(body), in.Ordinal, in.Name, in.Amount)
	}
	if err != nil {
		return nil, saveError{422, err.Error()}
	}
	dest, err := sanitizedPath("history/units/" + oob + ".txt")
	if err != nil {
		return nil, err
	}
	if err := gateText(dest, text); err != nil {
		return nil, err
	}
	created, err := st.raw.put(dest, text, oobUnitRawFileNote)
	if err != nil {
		return nil, err
	}
	return map[string]any{"file": dest, "imported": created}, nil
}

func madeRows(rows []project.Row) []project.Row {
	out := rows[:0:0]
	for _, row := range rows {
		if text(row, "source") != "imported" {
			out = append(out, row)
		}
	}
	return out
}

func saveOOBUnitRow(st *store, oob string, in oobUnitSaveInput) (map[string]any, bool, error) {
	ofFile := func(row project.Row) bool { return text(row, "oob_file") == oob }
	if in.Kind == "task_force" {
		fleets := madeRows(st.sortedRows(fleetTable, ofFile, byOrdering))
		if len(fleets) == 0 {
			return nil, false, nil
		}
		for _, fleet := range fleets {
			fleetID := project.Int64(fleet["id"])
			forces := st.sortedRows(taskForceTable, func(row project.Row) bool {
				return row["fleet_id"] != nil && project.Int64(row["fleet_id"]) == fleetID
			}, byOrdering)
			for _, force := range forces {
				if strings.Trim(text(force, "name"), `"`) != in.RawName {
					continue
				}
				id := project.Int64(force["id"])
				if _, err := st.update(taskForceTable, id, map[string]any{"name": *in.Name}); err != nil {
					return nil, true, err
				}
				return map[string]any{"task_force": id, "imported": false}, true, nil
			}
		}
		return nil, true, saveError{404, "the task force is not among the workbench's rows of the file"}
	}
	groups := madeRows(st.sortedRows(airWingGroupTable, ofFile, byOrdering))
	if len(groups) == 0 {
		return nil, false, nil
	}
	for _, group := range groups {
		if strings.TrimSpace(text(group, "location")) != strconv.Itoa(in.State) {
			continue
		}
		groupID := project.Int64(group["id"])
		wings := st.sortedRows(airWingTable, func(row project.Row) bool {
			return row["group_id"] != nil && project.Int64(row["group_id"]) == groupID
		}, byOrdering)
		if in.StateIndex < 0 || in.StateIndex >= len(wings) {
			break
		}
		changes := map[string]any{}
		if in.Name != nil {
			changes["name"] = *in.Name
		}
		if in.Amount != nil {

			amount := strconv.Itoa(int(*in.Amount))
			if _, err := importsvc.ParseHOI4ScalarToken(amount, true); err != nil {
				return nil, true, saveError{422, err.Error()}
			}
			changes["amount"] = amount
		}
		id := project.Int64(wings[in.StateIndex]["id"])
		if _, err := st.update(airWingTable, id, changes); err != nil {
			return nil, true, err
		}
		return map[string]any{"air_wing": id, "imported": false}, true, nil
	}
	return nil, true, saveError{404, "the wing is not among the workbench's rows of the file"}
}

func (h *handlers) oobUnitSave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in oobUnitSaveInput
	if !decodeBody(w, r, 1<<16, &in, "invalid unit payload") {
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}
	tables := []string{rawFilesTable, fleetTable, taskForceTable, airWingGroupTable, airWingTable}
	out, err := write(p, tables, func(st *store) (map[string]any, error) {
		out, err := saveOOBUnit(st, index, in)
		return st.raw.withUndo(out), err
	})
	h.finish(w, p, out, err, "oob unit save", true)
}
