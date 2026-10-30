package gamesave

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/project"
)

const historyFilesTable = "history_files"

type designSaveIndex interface {
	DesignSlotNames(typeName string) ([]string, bool)
	KnownEquipmentModule(name string) bool
}

type designSaveModule struct {
	Slot   string `json:"slot"`
	Module string `json:"module"`
}

type designSaveInput struct {
	Tag string `json:"tag"`

	Original string             `json:"original"`
	Name     string             `json:"name"`
	Type     string             `json:"type"`
	Modules  []designSaveModule `json:"modules"`
	Upgrades map[string]int     `json:"upgrades"`
	Delete   bool               `json:"delete"`
}

var (
	designTagPattern  = regexp.MustCompile(`^[A-Z][A-Z0-9]{2}$`)
	designWordPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

const notInProjectHistory = "this country's history is not in the project"

func validDesign(index designSaveIndex, in designSaveInput) error {
	if !designTagPattern.MatchString(in.Tag) {
		return saveError{422, "a country tag is required"}
	}
	if in.Delete {
		if strings.TrimSpace(in.Original) == "" {
			return saveError{422, "which design to delete"}
		}
		return nil
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 100 || strings.ContainsAny(name, "\"\r\n\\") {
		return saveError{422, "a design name is required"}
	}
	if !designWordPattern.MatchString(in.Type) {
		return saveError{422, "invalid type"}
	}
	slots, ok := index.DesignSlotNames(in.Type)
	if !ok {
		return saveError{422, "the type has no module slots"}
	}
	known := map[string]bool{}
	for _, slot := range slots {
		known[slot] = true
	}
	for _, module := range in.Modules {
		if !known[module.Slot] {
			return saveError{422, fmt.Sprintf("%s has no slot %s", in.Type, module.Slot)}
		}
		if module.Module != "" && module.Module != "empty" &&
			(!designWordPattern.MatchString(module.Module) || !index.KnownEquipmentModule(module.Module)) {
			return saveError{422, fmt.Sprintf("unknown module %s", module.Module)}
		}
	}
	for upgrade, level := range in.Upgrades {
		if !designWordPattern.MatchString(upgrade) || level < 0 || level > 50 {
			return saveError{422, "invalid upgrade"}
		}
	}
	return nil
}

func modulesText(modules []designSaveModule, indent string) string {
	var b strings.Builder
	b.WriteString("modules = {\n")
	for _, module := range modules {
		value := module.Module
		if value == "" {
			value = "empty"
		}
		fmt.Fprintf(&b, "%s\t%s = %s\n", indent, module.Slot, value)
	}
	b.WriteString(indent + "}")
	return b.String()
}

func upgradesText(upgrades map[string]int, indent string) string {
	names := []string{}
	for name, level := range upgrades {
		if level > 0 {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("upgrades = {\n")
	for _, name := range names {
		fmt.Fprintf(&b, "%s\t%s = %d\n", indent, name, upgrades[name])
	}
	b.WriteString(indent + "}")
	return b.String()
}

func variantBlock(words []scriptWord, name string) (key, open, close int, found bool) {
	for i := 0; i+2 < len(words); i++ {
		if !strings.EqualFold(words[i].text, "create_equipment_variant") || words[i+1].text != "=" || words[i+2].text != "{" {
			continue
		}
		end := closeOf(words, i+2)
		if end < 0 {
			return 0, 0, 0, false
		}
		if at, ok := assignments(words, i+2, end)["name"]; ok && strings.Trim(words[at[1]].text, `"`) == name {
			return i, i + 2, end, true
		}
	}
	return 0, 0, 0, false
}

func designEdit(text string, in designSaveInput) (string, bool, error) {
	words := scriptWords(text)
	original := strings.TrimSpace(in.Original)
	key, open, close, found := 0, 0, 0, false
	if original != "" {
		key, open, close, found = variantBlock(words, original)
	}
	if in.Delete {
		if !found {
			return "", false, saveError{404, "the design is not in the project's history"}
		}
		lineFrom := strings.LastIndexByte(text[:words[key].start], '\n') + 1
		lineTo := strings.IndexByte(text[words[close].end:], '\n')
		last := lineTo < 0
		if last {
			lineTo = len(text)
		} else {
			lineTo += words[close].end + 1
		}
		if strings.TrimSpace(text[lineFrom:words[key].start]) != "" {
			lineFrom = words[key].start
		} else if last && lineFrom > 0 {

			lineFrom--
		}
		return text[:lineFrom] + text[lineTo:], false, nil
	}
	name := strings.TrimSpace(in.Name)

	if name != original {
		if _, _, _, taken := variantBlock(words, name); taken {
			return "", false, saveError{409, "a design of that name already exists"}
		}
	}
	if !found {
		if original != "" {
			return "", false, saveError{404, "the design is not in the project's history"}
		}
		block := "create_equipment_variant = {\n\tname = \"" + name + "\"\n\ttype = " + in.Type + "\n\t" + modulesText(in.Modules, "\t") + "\n"
		if upgrades := upgradesText(in.Upgrades, "\t"); upgrades != "" {
			block += "\t" + upgrades + "\n"
		}
		block += "}\n"

		if text != "" && !strings.HasSuffix(text, "\n") {
			return text + "\n" + strings.TrimSuffix(block, "\n"), true, nil
		}
		return text + block, true, nil
	}
	inner := indentOf(text, words[key].start) + "\t"
	for _, w := range words[open+1 : close] {
		if w.depth == words[open].depth+1 {
			inner = indentOf(text, w.start)
			break
		}
	}
	edits := []edit{}
	have := assignments(words, open, close)
	if at, ok := have["name"]; ok {
		edits = append(edits, edit{words[at[1]].start, words[at[1]].end, `"` + name + `"`})
	}
	if at, ok := have["type"]; ok {
		edits = append(edits, edit{words[at[1]].start, words[at[1]].end, in.Type})
	}
	insert := func(line string) {
		closeAt := words[close].start
		lineFrom := strings.LastIndexByte(text[:closeAt], '\n') + 1
		if strings.TrimSpace(text[lineFrom:closeAt]) == "" {
			edits = append(edits, edit{lineFrom, lineFrom, inner + line + "\n"})
		} else {
			edits = append(edits, edit{closeAt, closeAt, "\n" + inner + line + "\n"})
		}
	}
	if from, to, ok := childBlock(words, open, close, "modules"); ok {
		edits = append(edits, edit{words[from].start, words[to].end, modulesText(in.Modules, inner)})
	} else {
		insert(modulesText(in.Modules, inner))
	}
	upgrades := upgradesText(in.Upgrades, inner)
	if from, to, ok := childBlock(words, open, close, "upgrades"); ok {
		if upgrades == "" {
			edits = append(edits, removal(text, words, from, to))
		} else {
			edits = append(edits, edit{words[from].start, words[to].end, upgrades})
		}
	} else if upgrades != "" {
		insert(upgrades)
	}
	return applyEdits(text, edits), false, nil
}

func saveDesign(st *store, index designSaveIndex, in designSaveInput) (map[string]any, error) {
	in.Tag = upperTag(in.Tag)
	in.Type = strings.ToLower(strings.TrimSpace(in.Type))
	if err := validDesign(index, in); err != nil {
		return nil, err
	}
	for _, row := range st.sortedRows(historyFilesTable, nil, byText("country_tag")) {
		if !strings.EqualFold(strings.TrimSpace(text(row, "country_tag")), in.Tag) {
			continue
		}
		script, added, err := designEdit(text(row, "raw_script"), in)
		if err != nil {
			return nil, err
		}
		id := project.Int64(row["id"])
		if _, err := st.update(historyFilesTable, id, map[string]any{"raw_script": script}); err != nil {
			return nil, err
		}
		return map[string]any{"name": strings.TrimSpace(in.Name), "added": added, "deleted": in.Delete, "history_file_id": id}, nil
	}
	return nil, saveError{409, notInProjectHistory}
}

func (h *handlers) designSave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in designSaveInput
	if !decodeBody(w, r, 1<<16, &in, "invalid design payload") {
		return
	}

	var index designSaveIndex
	if !in.Delete {
		loaded, ok := h.loadIndex(w, r, p)
		if !ok {
			return
		}
		index = loaded
	}
	out, err := write(p, []string{historyFilesTable}, func(st *store) (map[string]any, error) {
		return saveDesign(st, index, in)
	})
	h.finish(w, p, out, err, "design save", true)
}
