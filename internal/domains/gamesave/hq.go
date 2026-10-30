package gamesave

import (
	"fmt"
	"sort"
	"strings"
)

type hqSaveIndex interface {
	ArmyHQSource(tag, date, name string) (string, []byte, int, bool)
}

const hqRawFileNote = "游戏界面 Army HQ 模板(整份文件收进项目)"

func saveHQTemplate(st *store, index hqSaveIndex, in designerSaveInput) (map[string]any, bool, error) {
	tag := upperTag(in.Tag)
	file, body, line, ok := index.ArmyHQSource(tag, in.Date, in.Template)
	if !ok {
		return nil, false, nil
	}
	if len(in.RegimentalSupport) > 0 {
		return nil, true, saveError{422, "Army HQ templates do not have regimental support slots"}
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 255 || strings.ContainsAny(name, "\"\r\n") {
		return nil, true, saveError{422, "a template name is required"}
	}
	if len(in.Regiments) == 0 && len(in.Support) == 0 {
		return nil, true, saveError{422, "a template needs at least one battalion"}
	}
	for _, cell := range append(append([]designerSaveCell{}, in.Regiments...), in.Support...) {
		if strings.TrimSpace(cell.Type) == "" || strings.ContainsAny(cell.Type, " \t\r\n{}=\"#") ||
			cell.X < 0 || cell.Y < 0 || cell.X > 10 || cell.Y > 10 {
			return nil, true, saveError{422, "invalid battalion cell"}
		}
	}
	dest, err := sanitizedDest(file)
	if err != nil {
		return nil, true, err
	}
	text, err := hqTemplateEdit(string(body), line, in.Template, name, in.Regiments, in.Support)
	if err != nil {
		return nil, true, saveError{422, err.Error()}
	}
	if err := gateText(dest, text); err != nil {
		return nil, true, err
	}
	created, err := st.raw.put(dest, text, hqRawFileNote)
	if err != nil {
		return nil, true, err
	}
	return map[string]any{
		"template": map[string]any{"template_name": name, "country_tag": tag, "is_army_hq": true},
		"oob_file": dest,
		"imported": created,
	}, true, nil
}

func cellsText(key string, cells []designerSaveCell, indent string) string {
	sorted := append([]designerSaveCell{}, cells...)
	sort.SliceStable(sorted, func(a, b int) bool {
		if sorted[a].X != sorted[b].X {
			return sorted[a].X < sorted[b].X
		}
		return sorted[a].Y < sorted[b].Y
	})
	var b strings.Builder
	b.WriteString(key + " = {\n")
	for _, cell := range sorted {
		fmt.Fprintf(&b, "%s\t%s = { x = %d y = %d }\n", indent, strings.TrimSpace(cell.Type), cell.X, cell.Y)
	}
	b.WriteString(indent + "}")
	return b.String()
}

func hqTemplateEdit(text string, line int, oldName, name string, regiments, support []designerSaveCell, regimental ...[]designerSaveCell) (string, error) {
	from := lineStart(text, line)
	words := scriptWords(text)
	open := -1
	for i, word := range words {
		if word.start >= from && word.text == "division_template" && i+2 < len(words) && words[i+1].text == "=" && words[i+2].text == "{" {
			open = i + 2
			break
		}
	}
	if open < 0 {
		return "", fmt.Errorf("the Army HQ template is not where its file says (line %d)", line)
	}
	inner := words[open].depth + 1
	closeAt := -1
	var nameValue, regimentsBlock, supportBlock, regimentalBlock *span
	for i := open + 1; i < len(words); i++ {
		word := words[i]
		if word.text == "}" && word.depth == inner-1 {
			closeAt = i
			break
		}
		if word.depth != inner || i+2 >= len(words) || words[i+1].text != "=" {
			continue
		}
		switch strings.ToLower(word.text) {
		case "name":
			nameValue = &span{words[i+2].start, words[i+2].end}
		case "regiments", "support", "regimental_support":
			if words[i+2].text != "{" {
				continue
			}
			end := i + 3
			for end < len(words) && !(words[end].text == "}" && words[end].depth == inner) {
				end++
			}
			if end >= len(words) {
				return "", fmt.Errorf("the template's %s block does not close", word.text)
			}
			block := &span{word.start, words[end].end}
			if strings.EqualFold(word.text, "regiments") {
				regimentsBlock = block
			} else if strings.EqualFold(word.text, "support") {
				supportBlock = block
			} else {
				regimentalBlock = block
			}
		}
	}
	if closeAt < 0 {
		return "", fmt.Errorf("the Army HQ template does not close")
	}
	fieldIndent := indentOf(text, words[open].start) + "\t"
	type change struct {
		at   span
		with string
	}
	changes := []change{}
	if name != oldName {
		if nameValue == nil {
			return "", fmt.Errorf("this template's name comes from localisation (the generic template); it cannot be renamed here")
		}
		changes = append(changes, change{*nameValue, `"` + name + `"`})
	}
	closing := words[closeAt].start
	type part struct {
		key   string
		cells []designerSaveCell
		at    *span
	}
	parts := []part{{"regiments", regiments, regimentsBlock}, {"support", support, supportBlock}}
	if len(regimental) > 0 {
		parts = append(parts, part{"regimental_support", regimental[0], regimentalBlock})
	}
	for _, part := range parts {
		switch {
		case part.at != nil && len(part.cells) > 0:
			changes = append(changes, change{*part.at, cellsText(part.key, part.cells, indentOf(text, part.at.start))})
		case part.at != nil:

			start := strings.LastIndexByte(text[:part.at.start], '\n') + 1
			end := part.at.end
			if end < len(text) && text[end] == '\r' {
				end++
			}
			if end < len(text) && text[end] == '\n' {
				end++
			}
			changes = append(changes, change{span{start, end}, ""})
		case len(part.cells) > 0:

			at := strings.LastIndexByte(text[:closing], '\n') + 1
			changes = append(changes, change{span{at, at}, fieldIndent + cellsText(part.key, part.cells, fieldIndent) + "\n"})
		}
	}
	sort.Slice(changes, func(a, b int) bool { return changes[a].at.start > changes[b].at.start })
	out := text
	for _, c := range changes {
		out = out[:c.at.start] + c.with + out[c.at.end:]
	}
	return out, nil
}
