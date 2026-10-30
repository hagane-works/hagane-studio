package gamesave

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/domains/military"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

type divisionSaveInput struct {
	OOB        string   `json:"oob"`
	Ordinal    int      `json:"ordinal"`
	Name       *string  `json:"name"`
	Template   *string  `json:"template"`
	Experience *float64 `json:"experience"`
	Equipment  *float64 `json:"equipment"`
	Location   *int     `json:"location"`

	Officer *string `json:"officer"`
	Remove  bool    `json:"remove"`
}

var errDivisionMissing = errors.New("the division is not among the battle order file's units")

var errDivisionAdded = errors.New("this division was added in Recruit & Deploy; change it there")

var errTailNotRead = errors.New("the battle order file changed since it was read; refresh and try again")

func tailDivisionOrdinal(source, tail string, ordinal int) (int, error) {
	tail = strings.TrimSpace(tail)
	at := strings.LastIndex(source, tail)
	if tail == "" || at < 0 {
		return 0, errTailNotRead
	}
	words := scriptWords(source)
	seen, inTail := -1, 0
	for i := 0; i+2 < len(words); i++ {
		if !strings.EqualFold(words[i].text, "division") || words[i+1].text != "=" || words[i+2].text != "{" {
			continue
		}
		seen++
		if seen == ordinal {
			if words[i].start < at {
				return 0, errDivisionAdded
			}
			return inTail, nil
		}
		if words[i].start >= at {
			inTail++
		}
	}
	return 0, errDivisionMissing
}

func divisionEdit(text string, ordinal int, in divisionSaveInput) (string, error) {
	words := scriptWords(text)
	seen := -1
	for i := 0; i+2 < len(words); i++ {
		if !strings.EqualFold(words[i].text, "division") || words[i+1].text != "=" || words[i+2].text != "{" {
			continue
		}
		seen++
		if seen != ordinal {
			continue
		}
		open, close := i+2, closeOf(words, i+2)
		if close < 0 {
			return "", errDivisionMissing
		}
		if in.Remove {
			return applyEdits(text, []edit{removal(text, words, i, close)}), nil
		}
		inner := blockInner(text, words, open, close)
		edits := []edit{}
		added := []string{}
		have := assignments(words, open, close)
		setText := func(key, value string) {
			if at, found := have[key]; found {
				edits = append(edits, edit{words[at[1]].start, words[at[1]].end, value})
				return
			}
			added = append(added, key+" = "+value)
		}
		if in.Name != nil {

			for _, at := range childBlocks(words, open, close, "division_name") {
				edits = append(edits, removal(text, words, at[0], at[1]))
			}
			setText("name", `"`+*in.Name+`"`)
		}
		if in.Template != nil {
			setText("division_template", `"`+*in.Template+`"`)
		}
		if in.Location != nil {
			setText("location", strconv.Itoa(*in.Location))
		}
		if in.Officer != nil {
			officers := childBlocks(words, open, close, "officer")
			switch {
			case *in.Officer == "":
				for _, at := range officers {
					edits = append(edits, removal(text, words, at[0], at[1]))
				}
			case len(officers) > 0:

				first, last := officers[0][0]+2, officers[0][1]
				if at, found := assignments(words, first, last)["name"]; found {
					edits = append(edits, edit{words[at[1]].start, words[at[1]].end, `"` + *in.Officer + `"`})
				} else {
					edits = append(edits, edit{words[first].end, words[first].end, ` name = "` + *in.Officer + `"`})
				}
			default:
				added = append(added, `officer = { name = "`+*in.Officer+`" }`)
			}
		}
		values := map[string]*float64{}
		if in.Experience != nil {
			values["start_experience_factor"] = in.Experience
		}
		if in.Equipment != nil {
			values["start_equipment_factor"] = in.Equipment
		}
		if len(values) > 0 {
			edits = append(edits, setNumbers(text, words, open, close, values, inner)...)
		}
		edits = append(edits, insertBeforeClose(text, words, close, added, inner)...)
		return applyEdits(text, edits), nil
	}
	return "", errDivisionMissing
}

func saveDivision(st *store, index designerSaveIndex, in divisionSaveInput) (map[string]any, error) {
	oob := strings.Trim(strings.TrimSpace(in.OOB), `"`)
	if oob == "" || strings.ContainsAny(oob, " \t\r\n{}=\"#/\\") || in.Ordinal < 0 {
		return nil, saveError{422, "a battle order file and a division are required"}
	}
	if !in.Remove && in.Name == nil && in.Template == nil && in.Experience == nil && in.Equipment == nil && in.Location == nil && in.Officer == nil {
		return nil, saveError{422, "nothing to change"}
	}
	if in.Location != nil && *in.Location <= 0 {
		return nil, saveError{422, "a province id is required"}
	}
	if in.Officer != nil {
		officer := strings.TrimSpace(*in.Officer)
		if strings.ContainsAny(officer, "\"\r\n{}") {
			return nil, saveError{422, "a commander's name is written without quotes or braces"}
		}
		in.Officer = &officer
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" || strings.ContainsAny(name, "\"\r\n{}") {
			return nil, saveError{422, "a division name is required, without quotes or braces"}
		}
		in.Name = &name
	}
	for _, value := range []*float64{in.Experience, in.Equipment} {
		if value != nil && (*value < 0 || *value > 1) {
			return nil, saveError{422, "experience and equipment are shares from 0 to 1"}
		}
	}

	source, _ := index.OOBSource(oob)
	rows, imported, err := takeOOBIn(st, index, oob)
	if err != nil {
		return nil, err
	}
	if in.Template != nil {
		known := false
		for _, row := range rows {
			known = known || row.TemplateName == *in.Template
		}
		if !known {
			return nil, saveError{422, "the template is not in the battle order file"}
		}
	}
	for _, row := range rows {
		if !strings.Contains(row.OobUnitsTail, "division") {
			continue
		}
		ordinal, err := divisionOrdinal(st, oob, string(source), row.OobUnitsTail, in.Ordinal, imported)
		if err != nil {
			return nil, err
		}
		text, err := divisionEdit(row.OobUnitsTail, ordinal, in)
		if err != nil {
			return nil, saveError{422, err.Error()}
		}
		if _, err := military.UpdateDivisionTemplate(st.tx, row.ID, military.DivisionTemplatePatch{
			OobUnitsTail: workbench.Optional[string]{Set: true, Value: &text},
		}); err != nil {
			return nil, refusal(err)
		}
		return map[string]any{"oob": oob, "row": row.ID}, nil
	}
	return nil, saveError{422, errDivisionMissing.Error()}
}

func divisionOrdinal(st *store, oob, source, tail string, ordinal int, imported bool) (int, error) {
	if imported || source == "" {
		return ordinal, nil
	}
	mapped, err := tailDivisionOrdinal(source, tail, ordinal)
	if err == nil {
		return mapped, nil
	}
	if errors.Is(err, errTailNotRead) && len(st.deployments(oob)) == 0 {
		return ordinal, nil
	}
	return 0, saveError{422, err.Error()}
}

func (h *handlers) divisionSave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in divisionSaveInput
	if !decodeBody(w, r, 1<<16, &in, "invalid division payload") {
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}

	out, err := write(p, []string{divisionTemplateTable, divisionDeploymentTable}, func(st *store) (map[string]any, error) {
		return saveDivision(st, index, in)
	})
	h.finish(w, p, out, err, "division save", true)
}
