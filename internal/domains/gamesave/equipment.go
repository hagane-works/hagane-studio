package gamesave

import (
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/project"
)

type equipmentSaveIndex interface {
	EquipmentSource(name string) (string, []byte, int, bool)
}

type equipmentSaveInput struct {
	Name string `json:"name"`

	Values    map[string]*float64 `json:"values"`
	Resources map[string]*float64 `json:"resources"`
}

const equipmentRawFileNote = "游戏界面装备(整份文件收进项目)"

var equipmentKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func equipmentEdit(text string, line int, name string, values, resources map[string]*float64) (string, error) {
	words := scriptWords(text)
	open, close, err := blockBounds(words, text, line, name)
	if err != nil {
		return "", err
	}
	inner := indentOf(text, words[open].start) + "\t"
	for _, w := range words[open+1 : close] {
		if w.depth == words[open].depth+1 {
			inner = indentOf(text, w.start)
			break
		}
	}
	edits := setNumbers(text, words, open, close, values, inner)
	if len(resources) > 0 {
		if key, end, found := childBlock(words, open, close, "resources"); found {
			keep := 0
			have := assignments(words, key+2, end)
			for resource := range have {
				if value, touched := resources[resource]; !touched || value != nil {
					keep++
				}
			}
			for resource, value := range resources {
				if _, present := have[resource]; !present && value != nil {
					keep++
				}
			}
			if keep == 0 {

				edits = append(edits, removal(text, words, key, end))
			} else {
				edits = append(edits, setNumbers(text, words, key+2, end, resources, inner+"\t")...)
			}
		} else {
			parts := []string{}
			names := make([]string, 0, len(resources))
			for resource := range resources {
				names = append(names, resource)
			}
			sort.Strings(names)
			for _, resource := range names {
				if value := resources[resource]; value != nil {
					parts = append(parts, resource+" = "+numberText(*value))
				}
			}
			if len(parts) > 0 {
				closeAt := words[close].start
				lineFrom := strings.LastIndexByte(text[:closeAt], '\n') + 1
				line := inner + "resources = { " + strings.Join(parts, " ") + " }\n"
				if strings.TrimSpace(text[lineFrom:closeAt]) == "" {
					edits = append(edits, edit{lineFrom, lineFrom, line})
				} else {
					edits = append(edits, edit{closeAt, closeAt, "\n" + line})
				}
			}
		}
	}
	return applyEdits(text, edits), nil
}

func validEquipmentNumbers(values map[string]*float64) error {
	for key, value := range values {
		if !equipmentKeyPattern.MatchString(key) {
			return fmt.Errorf("invalid key %q", key)
		}
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
			return fmt.Errorf("invalid number for %s", key)
		}
	}
	return nil
}

func saveEquipment(st *store, index equipmentSaveIndex, in equipmentSaveInput) (map[string]any, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || strings.ContainsAny(name, " \t\r\n{}=\"#") {
		return nil, saveError{422, "an equipment name is required"}
	}
	if len(in.Values) == 0 && len(in.Resources) == 0 {
		return nil, saveError{422, "nothing to change"}
	}
	if err := validEquipmentNumbers(in.Values); err != nil {
		return nil, saveError{422, err.Error()}
	}
	if err := validEquipmentNumbers(in.Resources); err != nil {
		return nil, saveError{422, err.Error()}
	}
	file, body, line, ok := index.EquipmentSource(name)
	if !ok {
		return nil, saveError{404, "the equipment is not declared in any equipment file"}
	}
	dest, err := sanitizedDest(file)
	if err != nil {
		return nil, err
	}
	text, err := equipmentEdit(string(body), line, name, in.Values, in.Resources)
	if err != nil {
		return nil, saveError{422, err.Error()}
	}
	if err := gateText(dest, text); err != nil {
		return nil, err
	}
	created, err := st.raw.put(dest, text, equipmentRawFileNote)
	if err != nil {
		return nil, err
	}
	return map[string]any{"name": name, "file": dest, "imported": created}, nil
}

func (h *handlers) equipmentSave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in equipmentSaveInput
	if !decodeBody(w, r, 1<<16, &in, "invalid equipment payload") {
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}
	out, err := write(p, []string{rawFilesTable}, func(st *store) (map[string]any, error) {
		out, err := saveEquipment(st, index, in)
		return st.raw.withUndo(out), err
	})
	h.finish(w, p, out, err, "equipment save", true)
}
