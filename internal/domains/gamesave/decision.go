package gamesave

import (
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/project"
)

type decisionSaveInput struct {
	Category     string   `json:"category"`
	Name         string   `json:"name"`
	Cost         *float64 `json:"cost"`
	Icon         *string  `json:"icon"`
	DaysRemove   *int     `json:"days_remove"`
	DaysReEnable *int     `json:"days_re_enable"`
	FireOnlyOnce *bool    `json:"fire_only_once"`
}

type decisionSaveIndex interface {
	DecisionSource(category, name string) (string, []byte, bool)
}

const decisionRawFileNote = "游戏界面 · 修改原版决议(整份文件收进项目)"

var decisionScriptToken = regexp.MustCompile(`^[A-Za-z0-9_.\-]+$`)

type decisionScalar struct{ key, value string }

func decisionSaveScalars(in decisionSaveInput) ([]decisionScalar, error) {
	if !decisionScriptToken.MatchString(in.Category) || !decisionScriptToken.MatchString(in.Name) {
		return nil, saveError{422, "a decision is named by its category and its script id"}
	}
	out := []decisionScalar{}
	if in.Cost != nil {
		cost := *in.Cost
		if math.IsNaN(cost) || cost < 0 || cost > 100000 {
			return nil, saveError{422, "cost must be between 0 and 100000"}
		}
		out = append(out, decisionScalar{"cost", strconv.FormatFloat(cost, 'f', -1, 64)})
	}
	if in.Icon != nil {
		icon := strings.TrimSpace(*in.Icon)
		if !decisionScriptToken.MatchString(icon) {
			return nil, saveError{422, "the icon is a sprite or icon name (letters, digits, _ . -)"}
		}
		out = append(out, decisionScalar{"icon", icon})
	}
	for _, days := range []struct {
		key   string
		value *int
	}{{"days_remove", in.DaysRemove}, {"days_re_enable", in.DaysReEnable}} {
		if days.value == nil {
			continue
		}
		if *days.value < 0 || *days.value > 36500 {
			return nil, saveError{422, days.key + " must be between 0 and 36500"}
		}
		out = append(out, decisionScalar{days.key, strconv.Itoa(*days.value)})
	}
	if in.FireOnlyOnce != nil {
		value := "no"
		if *in.FireOnlyOnce {
			value = "yes"
		}
		out = append(out, decisionScalar{"fire_only_once", value})
	}
	if len(out) == 0 {
		return nil, saveError{422, "nothing to change"}
	}
	return out, nil
}

func decisionFieldsEdit(text, category, name string, scalars []decisionScalar) (string, error) {
	words := scriptWords(text)
	opener := func(i, depth int, key string) bool {
		return i+2 < len(words) && words[i].depth == depth && strings.EqualFold(words[i].text, key) &&
			words[i+1].text == "=" && words[i+2].text == "{"
	}
	open := -1
	for i := 0; i < len(words) && open < 0; i++ {
		if !opener(i, 0, category) {
			continue
		}
		for j := i + 3; j < len(words); j++ {
			if words[j].text == "}" && words[j].depth == 0 {
				break
			}
			if opener(j, 1, name) {
				open = j + 2
				break
			}
		}
	}
	if open < 0 {
		return "", fmt.Errorf("decision %s is not in its category %s in this file", name, category)
	}
	inner := words[open].depth + 1
	closeAt := -1
	values := map[string]span{}
	for i := open + 1; i < len(words); i++ {
		word := words[i]
		if word.text == "}" && word.depth == inner-1 {
			closeAt = i
			break
		}
		if word.depth != inner || i+2 >= len(words) || words[i+1].text != "=" {
			continue
		}
		key := strings.ToLower(word.text)
		for _, scalar := range scalars {
			if key != scalar.key {
				continue
			}
			if words[i+2].text == "{" {
				return "", fmt.Errorf("%s of %s is a script block, not a value; change it in the workbench", word.text, name)
			}
			values[key] = span{words[i+2].start, words[i+2].end}
		}
	}
	if closeAt < 0 {
		return "", fmt.Errorf("decision %s does not close", name)
	}
	fieldIndent := indentOf(text, words[open].start) + "\t"
	closing := words[closeAt].start
	type change struct {
		at   span
		with string
	}
	changes := []change{}
	for _, scalar := range scalars {
		if at, ok := values[scalar.key]; ok {
			changes = append(changes, change{at, scalar.value})
			continue
		}

		lineAt := strings.LastIndexByte(text[:closing], '\n') + 1
		if strings.TrimSpace(text[lineAt:closing]) == "" {
			changes = append(changes, change{span{lineAt, lineAt}, fieldIndent + scalar.key + " = " + scalar.value + "\n"})
		} else {
			changes = append(changes, change{span{closing, closing}, scalar.key + " = " + scalar.value + " "})
		}
	}

	order := make([]int, len(changes))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		if changes[order[a]].at.start != changes[order[b]].at.start {
			return changes[order[a]].at.start > changes[order[b]].at.start
		}
		return order[a] > order[b]
	})
	out := text
	for _, at := range order {
		c := changes[at]
		out = out[:c.at.start] + c.with + out[c.at.end:]
	}
	return out, nil
}

func saveDecisionOverride(st *store, index decisionSaveIndex, in decisionSaveInput) (map[string]any, error) {
	scalars, err := decisionSaveScalars(in)
	if err != nil {
		return nil, err
	}
	file, body, ok := index.DecisionSource(in.Category, in.Name)
	if !ok {
		return nil, saveError{404, "the game files do not declare this decision"}
	}
	dest, err := sanitizedDest(file)
	if err != nil {
		return nil, err
	}
	text, err := decisionFieldsEdit(string(body), in.Category, in.Name, scalars)
	if err != nil {
		return nil, saveError{422, err.Error()}
	}
	if err := gateText(dest, text); err != nil {
		return nil, err
	}
	created, err := st.raw.put(dest, text, decisionRawFileNote)
	if err != nil {
		return nil, err
	}
	return map[string]any{"decision": in.Name, "file": dest, "imported": created}, nil
}

func (h *handlers) decisionSave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in decisionSaveInput
	if !decodeBody(w, r, 1<<16, &in, "invalid decision payload") {
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}

	out, err := write(p, []string{rawFilesTable}, func(st *store) (map[string]any, error) {
		out, err := saveDecisionOverride(st, index, in)
		return st.raw.withUndo(out), err
	})
	h.finish(w, p, out, err, "decision save", true)
}
