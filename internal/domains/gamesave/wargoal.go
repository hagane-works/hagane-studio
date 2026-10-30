package gamesave

import (
	"net/http"
	"strings"

	"haganeworks/internal/domains/content"
	"haganeworks/internal/project"
)

const wargoalsTable = "wargoal_types"

type wargoalSaveIndex interface {
	WargoalSource(name string) (string, []byte, int, bool)
}

type wargoalSaveInput struct {
	Name   string              `json:"name"`
	Values map[string]*float64 `json:"values"`
}

var wargoalNumberKeys = map[string]bool{
	"generate_base_cost": true, "generate_per_state_cost": true, "take_states_limit": true, "threat": true,
}

const wargoalRawFileNote = "游戏界面战争目标(整份文件收进项目)"

func saveWargoal(st *store, index wargoalSaveIndex, in wargoalSaveInput) (map[string]any, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || strings.ContainsAny(name, " \t\r\n{}=\"#") {
		return nil, saveError{422, "a war goal name is required"}
	}
	if len(in.Values) == 0 {
		return nil, saveError{422, "nothing to change"}
	}
	for key, value := range in.Values {
		if !wargoalNumberKeys[key] {
			return nil, saveError{422, "not a war goal number: " + key}
		}
		if value != nil && *value < 0 {
			return nil, saveError{422, key + " cannot be negative"}
		}
	}
	if err := validEquipmentNumbers(in.Values); err != nil {
		return nil, saveError{422, err.Error()}
	}
	if out, found, err := saveWargoalRow(st, name, in.Values); err != nil || found {
		return out, err
	}
	file, body, line, ok := index.WargoalSource(name)
	if !ok {
		return nil, saveError{404, "the war goal is not declared in any wargoals file"}
	}
	dest, err := sanitizedDest(file)
	if err != nil {
		return nil, err
	}
	text, err := equipmentEdit(string(body), line, name, in.Values, nil)
	if err != nil {
		return nil, saveError{422, err.Error()}
	}
	if err := gateText(dest, text); err != nil {
		return nil, err
	}
	created, err := st.raw.put(dest, text, wargoalRawFileNote)
	if err != nil {
		return nil, err
	}
	return map[string]any{"name": name, "file": dest, "imported": created}, nil
}

func saveWargoalRow(st *store, name string, values map[string]*float64) (map[string]any, bool, error) {
	for _, row := range st.sortedRows(wargoalsTable, nil, byText("wargoal_key")) {
		if !strings.EqualFold(text(row, "wargoal_key"), name) || textOr(row, "source", "created") == "imported" {
			continue
		}
		scalars := map[string]string{}
		decodeColumn(row["safe_scalars"], &scalars)
		for key, value := range values {
			if value == nil {
				delete(scalars, key)
			} else {
				scalars[key] = numberText(*value)
			}
		}
		cleaned, err := content.ValidateWargoalSafeScalars(scalars)
		if err != nil {
			return nil, true, saveError{422, err.Error()}
		}
		id := project.Int64(row["id"])
		if _, err := st.update(wargoalsTable, id, map[string]any{"safe_scalars": jsonValue(cleaned)}); err != nil {
			return nil, true, err
		}
		return map[string]any{"name": name, "row": id, "imported": false}, true, nil
	}
	return nil, false, nil
}

func (h *handlers) wargoalSave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in wargoalSaveInput
	if !decodeBody(w, r, 1<<16, &in, "invalid war goal payload") {
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}
	out, err := write(p, []string{rawFilesTable, wargoalsTable}, func(st *store) (map[string]any, error) {
		out, err := saveWargoal(st, index, in)
		return st.raw.withUndo(out), err
	})
	h.finish(w, p, out, err, "war goal save", true)
}
