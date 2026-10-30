package gamesave

import (
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/project"
)

const specialProjectsTable = "special_projects"

type specialProjectSaveIndex interface {
	SpecialProjectSource(name string) (string, []byte, int, bool)
}

type specialProjectSaveInput struct {
	ID string `json:"id"`

	BreakthroughCost *map[string]float64 `json:"breakthrough_cost"`
	PrototypeTime    *string             `json:"prototype_time"`
	Complexity       *string             `json:"complexity"`
	Allowed          *string             `json:"allowed"`
	Available        *string             `json:"available"`
	DesignBonus      *designBonusInput   `json:"design_bonus"`
}

type designBonusInput struct {
	Name       string  `json:"name"`
	Uses       float64 `json:"uses"`
	CostFactor float64 `json:"cost_factor"`
}

const specialProjectRawFileNote = "游戏界面特殊项目(整份文件收进项目)"

var (
	specialProjectConstant = regexp.MustCompile(`^sp_(time\.prototype|complexity)\.[a-z_]+$`)
	specializationID       = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

func costScript(cost map[string]float64) string {
	keys := make([]string, 0, len(cost))
	for key, amount := range cost {
		if amount > 0 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+" = "+numberText(cost[key]))
	}
	return strings.Join(lines, "\n")
}

func setToken(words []scriptWord, open, close int, key, value string) ([]edit, []string) {
	if at, found := assignments(words, open, close)[key]; found {
		return []edit{{words[at[1]].start, words[at[1]].end, value}}, nil
	}
	return nil, []string{key + " = " + value}
}

func designBonusEdit(text string, bonus designBonusInput) (string, int) {
	words := scriptWords(text)
	edits := []edit{}
	found := 0
	for i := 0; i+2 < len(words); i++ {
		if !strings.EqualFold(words[i].text, "add_design_template_bonus") || words[i+1].text != "=" || words[i+2].text != "{" {
			continue
		}
		open, close := i+2, closeOf(words, i+2)
		if close < 0 {
			break
		}
		i = close
		if at, ok := assignments(words, open, close)["name"]; !ok || !strings.EqualFold(strings.Trim(words[at[1]].text, `"`), bonus.Name) {
			continue
		}
		found++
		uses, factor := bonus.Uses, bonus.CostFactor
		edits = append(edits, setNumbers(text, words, open, close, map[string]*float64{"uses": &uses, "cost_factor": &factor}, blockInner(text, words, open, close))...)
	}
	return applyEdits(text, edits), found
}

var errDesignBonusMissing = errors.New("the project's output has no such design discount")

func specialProjectEdit(text string, line int, name string, in specialProjectSaveInput) (string, error) {
	words := scriptWords(text)
	open, close, err := blockBounds(words, text, line, name)
	if err != nil {
		return "", err
	}
	inner := blockInner(text, words, open, close)
	edits := []edit{}
	added := []string{}
	for _, part := range []struct {
		key   string
		value *string
	}{{"prototype_time", in.PrototypeTime}, {"complexity", in.Complexity}} {
		if part.value == nil {
			continue
		}
		more, add := setToken(words, open, close, part.key, *part.value)
		edits = append(edits, more...)
		added = append(added, add...)
	}
	var cost *string
	if in.BreakthroughCost != nil {
		script := costScript(*in.BreakthroughCost)
		cost = &script
	}
	for _, part := range []struct {
		key   string
		value *string
	}{{"breakthrough_cost", cost}, {"allowed", in.Allowed}, {"available", in.Available}} {
		more, add := scriptBlockEdits(text, words, open, close, part.key, part.value, inner)
		edits = append(edits, more...)
		added = append(added, add...)
	}
	if in.DesignBonus != nil {
		outputs := childBlocks(words, open, close, "project_output")
		if len(outputs) == 0 {
			return "", errDesignBonusMissing
		}
		at := outputs[len(outputs)-1]
		from, to := words[at[0]].start, words[at[1]].end
		changed, found := designBonusEdit(text[from:to], *in.DesignBonus)
		if found == 0 {
			return "", errDesignBonusMissing
		}
		edits = append(edits, edit{from, to, changed})
	}
	edits = append(edits, insertBeforeClose(text, words, close, added, inner)...)
	return applyEdits(text, edits), nil
}

func saveSpecialProject(st *store, index specialProjectSaveIndex, in specialProjectSaveInput) (map[string]any, error) {
	name := strings.TrimSpace(in.ID)
	if name == "" || strings.ContainsAny(name, " \t\r\n{}=\"#") {
		return nil, saveError{422, "a special project id is required"}
	}
	if in.BreakthroughCost == nil && in.PrototypeTime == nil && in.Complexity == nil && in.Allowed == nil && in.Available == nil && in.DesignBonus == nil {
		return nil, saveError{422, "nothing to change"}
	}
	if bonus := in.DesignBonus; bonus != nil {
		if bonus.Name == "" || strings.ContainsAny(bonus.Name, " \t\r\n{}=\"#") {
			return nil, saveError{422, "a design discount is named"}
		}
		if bonus.Uses < 1 || bonus.Uses > 100 || bonus.Uses != float64(int(bonus.Uses)) || bonus.CostFactor < 0 || bonus.CostFactor > 1 {
			return nil, saveError{422, "a design discount is 1 to 100 designs and a share off from 0 to 1"}
		}
	}
	for _, constant := range []*string{in.PrototypeTime, in.Complexity} {
		if constant != nil && !specialProjectConstant.MatchString(strings.TrimSpace(*constant)) {
			return nil, saveError{422, "not a special project constant: " + *constant}
		}
	}
	if in.BreakthroughCost != nil {
		for key, amount := range *in.BreakthroughCost {
			if !specializationID.MatchString(key) || amount < 0 || amount > 100 || amount != float64(int(amount)) {
				return nil, saveError{422, "a breakthrough cost is a whole number of points by specialization"}
			}
		}
	}
	for _, script := range []*string{in.Allowed, in.Available} {
		if script == nil {
			continue
		}
		cleaned, err := cleanScript(*script)
		if err != nil {
			return nil, saveError{422, err.Error()}
		}
		*script = cleaned
	}
	for _, row := range st.sortedRows(specialProjectsTable, nil, byText("project_token")) {
		if !strings.EqualFold(text(row, "project_token"), name) || textOr(row, "source", "created") == "imported" {
			continue
		}
		changes := map[string]any{}
		if in.PrototypeTime != nil {
			changes["prototype_time"] = *in.PrototypeTime
		}
		if in.Complexity != nil {
			changes["complexity"] = *in.Complexity
		}
		if in.Allowed != nil {
			changes["allowed"] = *in.Allowed
		}
		if in.Available != nil {
			changes["available"] = *in.Available
		}
		if in.DesignBonus != nil {
			changed, found := designBonusEdit(text(row, "project_output"), *in.DesignBonus)
			if found == 0 {
				return nil, saveError{422, errDesignBonusMissing.Error()}
			}
			changes["project_output"] = changed
		}
		if in.BreakthroughCost != nil {
			extras := map[string]any{}
			if current, ok := row["raw_extras"].(map[string]any); ok {
				for key, value := range current {
					extras[key] = value
				}
			}
			delete(extras, "breakthrough_cost")
			cost := map[string]any{}
			for key, amount := range *in.BreakthroughCost {
				if amount > 0 {
					cost[key] = amount
				}
			}
			if len(cost) > 0 {
				extras["breakthrough_cost"] = cost
			}
			changes["raw_extras"] = jsonValue(extras)
		}
		id := project.Int64(row["id"])
		if _, err := st.update(specialProjectsTable, id, changes); err != nil {
			return nil, err
		}
		return map[string]any{"row": id, "imported": false}, nil
	}
	file, body, line, ok := index.SpecialProjectSource(name)
	if !ok {
		return nil, saveError{404, "the special project is not declared in any projects file"}
	}
	dest, err := sanitizedDest(file)
	if err != nil {
		return nil, err
	}
	text, err := specialProjectEdit(string(body), line, name, in)
	if err != nil {
		return nil, saveError{422, err.Error()}
	}
	if err := gateText(dest, text); err != nil {
		return nil, err
	}
	created, err := st.raw.put(dest, text, specialProjectRawFileNote)
	if err != nil {
		return nil, err
	}
	return map[string]any{"file": dest, "imported": created}, nil
}

func (h *handlers) specialProjectSave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in specialProjectSaveInput
	if !decodeBody(w, r, 1<<16, &in, "invalid special project payload") {
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}
	out, err := write(p, []string{rawFilesTable, specialProjectsTable}, func(st *store) (map[string]any, error) {
		out, err := saveSpecialProject(st, index, in)
		return st.raw.withUndo(out), err
	})
	h.finish(w, p, out, err, "special project save", true)
}
