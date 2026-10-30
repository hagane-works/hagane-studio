package gamesave

import (
	"errors"
	"math"
	"math/big"
	"net/http"
	"strings"

	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

const autonomyStatesTable = "autonomy_states"

type autonomySaveIndex interface {
	AutonomySource(id string) (string, []byte, int, bool)
}

type autonomySaveInput struct {
	ID string `json:"id"`

	Values   map[string]*float64 `json:"values"`
	Rule     *string             `json:"rule"`
	Modifier *string             `json:"modifier"`
}

var autonomyNumberKeys = map[string]bool{"min_freedom_level": true, "manpower_influence": true}

const autonomyRawFileNote = "游戏界面自治等级(整份文件收进项目)"

func autonomyEdit(text string, line int, values map[string]*float64, rule, modifier *string) (string, error) {
	words := scriptWords(text)
	open, close, err := blockBounds(words, text, line, "autonomy_state")
	if err != nil {
		return "", err
	}
	inner := blockInner(text, words, open, close)
	edits := setNumbers(text, words, open, close, values, inner)
	added := []string{}
	for _, part := range []struct {
		key   string
		value *string
	}{{"rule", rule}, {"modifier", modifier}} {
		more, add := scriptBlockEdits(text, words, open, close, part.key, part.value, inner)
		edits = append(edits, more...)
		added = append(added, add...)
	}
	edits = append(edits, insertBeforeClose(text, words, close, added, inner)...)
	return applyEdits(text, edits), nil
}

func modifierTable(script string) (map[string]any, error) {
	out := map[string]any{}
	if strings.TrimSpace(script) == "" {
		return out, nil
	}
	root, _, err := paradox.Parse(script)
	if err != nil || root == nil {
		return nil, errors.New("the modifiers are not valid script")
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block != nil {
			return nil, errors.New("modifiers with blocks in them are edited in the autonomy level workbench")
		}
		switch entry.Scalar.Kind {
		case paradox.ScalarInt:
			value, _ := new(big.Float).SetInt(entry.Scalar.Int).Float64()
			out[*entry.Key] = value
		case paradox.ScalarFloat:
			out[*entry.Key] = entry.Scalar.Float
		default:
			out[*entry.Key] = strings.Trim(entry.Scalar.Str, `"`)
		}
	}
	return out, nil
}

func saveAutonomy(st *store, index autonomySaveIndex, in autonomySaveInput) (map[string]any, error) {
	id := strings.ToLower(strings.TrimSpace(in.ID))
	if id == "" || strings.ContainsAny(id, " \t\r\n{}=\"#") {
		return nil, saveError{422, "an autonomy level id is required"}
	}
	if len(in.Values) == 0 && in.Rule == nil && in.Modifier == nil {
		return nil, saveError{422, "nothing to change"}
	}
	for key, value := range in.Values {
		if !autonomyNumberKeys[key] {
			return nil, saveError{422, "not an autonomy level number: " + key}
		}
		if value == nil && key == "min_freedom_level" {
			return nil, saveError{422, "min_freedom_level cannot be dropped"}
		}
		if value != nil && (*value < 0 || *value > 1) {
			return nil, saveError{422, key + " must be between 0 and 1"}
		}
	}
	if err := validEquipmentNumbers(in.Values); err != nil {
		return nil, saveError{422, err.Error()}
	}
	for _, script := range []*string{in.Rule, in.Modifier} {
		if script == nil {
			continue
		}
		cleaned, err := cleanScript(*script)
		if err != nil {
			return nil, saveError{422, err.Error()}
		}
		*script = cleaned
	}
	for _, row := range st.sortedRows(autonomyStatesTable, nil, nil) {
		if !strings.EqualFold(text(row, "autonomy_id"), id) || textOr(row, "source", "created") == "imported" {
			continue
		}
		changes := map[string]any{}
		if value, ok := in.Values["min_freedom_level"]; ok && value != nil {
			changes["min_freedom_level"] = int64(math.Round(*value * 1000))
		}
		if value, ok := in.Values["manpower_influence"]; ok {
			if value == nil {
				changes["manpower_influence"] = nil
			} else {
				changes["manpower_influence"] = *value
			}
		}
		if in.Rule != nil {
			changes["rule"] = *in.Rule
		}
		if in.Modifier != nil {
			table, err := modifierTable(*in.Modifier)
			if err != nil {
				return nil, saveError{422, err.Error()}
			}
			changes["modifier"] = jsonValue(table)
		}
		rowID := project.Int64(row["id"])
		if _, err := st.update(autonomyStatesTable, rowID, changes); err != nil {
			return nil, err
		}
		return map[string]any{"row": rowID, "imported": false}, nil
	}
	file, body, line, ok := index.AutonomySource(id)
	if !ok {
		return nil, saveError{404, "the autonomy level is not declared in any autonomous_states file"}
	}
	dest, err := sanitizedDest(file)
	if err != nil {
		return nil, err
	}
	text, err := autonomyEdit(string(body), line, in.Values, in.Rule, in.Modifier)
	if err != nil {
		return nil, saveError{422, err.Error()}
	}
	if err := gateText(dest, text); err != nil {
		return nil, err
	}
	created, err := st.raw.put(dest, text, autonomyRawFileNote)
	if err != nil {
		return nil, err
	}
	return map[string]any{"file": dest, "imported": created}, nil
}

func (h *handlers) autonomySave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in autonomySaveInput
	if !decodeBody(w, r, 1<<16, &in, "invalid autonomy level payload") {
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}
	out, err := write(p, []string{rawFilesTable, autonomyStatesTable}, func(st *store) (map[string]any, error) {
		out, err := saveAutonomy(st, index, in)
		return st.raw.withUndo(out), err
	})
	h.finish(w, p, out, err, "autonomy level save", true)
}
