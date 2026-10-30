package gamesave

import (
	"fmt"
	"strings"

	"haganeworks/internal/domains/gameview"
)

const designerRawFileNote = "游戏界面师编辑器(模板写在战斗序列文件里)"

type oobTemplate struct {
	name string
	line int
}

func oobTemplates(text string) []oobTemplate {
	words := scriptWords(text)
	out := []oobTemplate{}
	for i := 0; i+2 < len(words); i++ {
		if words[i].depth != 0 || words[i].text != "division_template" || words[i+1].text != "=" || words[i+2].text != "{" {
			continue
		}
		name := ""
		for j := i + 3; j < len(words); j++ {
			if words[j].text == "}" && words[j].depth == 0 {
				break
			}
			if words[j].depth == 1 && strings.EqualFold(words[j].text, "name") && j+2 < len(words) && words[j+1].text == "=" {
				name = strings.Trim(words[j+2].text, `"`)
				break
			}
		}
		out = append(out, oobTemplate{name: name, line: strings.Count(text[:words[i].start], "\n") + 1})
	}
	return out
}

func templateScalarEdit(text string, line int, key, value, skipIfMissing string) (string, error) {
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
		return "", fmt.Errorf("the template is not where the file says (line %d)", line)
	}
	inner := words[open].depth + 1
	after := words[open].end
	for i := open + 1; i < len(words); i++ {
		word := words[i]
		if word.text == "}" && word.depth == inner-1 {
			break
		}
		if word.depth != inner || i+2 >= len(words) || words[i+1].text != "=" {
			continue
		}
		if strings.EqualFold(word.text, key) {
			if words[i+2].text == "{" {
				return "", fmt.Errorf("the template's %s is a block", key)
			}
			return text[:words[i+2].start] + value + text[words[i+2].end:], nil
		}
		if strings.EqualFold(word.text, "name") {
			after = words[i+2].end
		}
	}
	if skipIfMissing != "" && value == skipIfMissing {
		return text, nil
	}
	end := strings.IndexByte(text[after:], '\n')
	if end < 0 {
		return text + "\n" + indentOf(text, words[open].start) + "\t" + key + " = " + value, nil
	}
	at := after + end + 1
	return text[:at] + indentOf(text, words[open].start) + "\t" + key + " = " + value + "\n" + text[at:], nil
}

func newTemplateText(in designerSaveInput, name string) string {
	var b strings.Builder
	b.WriteString("division_template = {\n")
	fmt.Fprintf(&b, "\tname = %q\n", name)
	if group := strings.TrimSpace(in.NamesGroup); group != "" {
		fmt.Fprintf(&b, "\tdivision_names_group = %s\n", group)
	}
	if in.Priority != nil && *in.Priority != 1 {
		fmt.Fprintf(&b, "\tpriority = %d\n", *in.Priority)
	}
	for _, part := range []struct {
		key   string
		cells []designerSaveCell
	}{{"regiments", in.Regiments}, {"support", in.Support}, {"regimental_support", in.RegimentalSupport}} {
		if len(part.cells) > 0 {
			b.WriteString("\t" + cellsText(part.key, part.cells, "\t") + "\n")
		}
	}
	b.WriteString("}\n\n")
	return b.String()
}

func oobTemplateEdit(text string, in designerSaveInput, name string) (string, error) {
	templates := oobTemplates(text)
	var target *oobTemplate
	for i := range templates {
		if in.Template != "" && strings.EqualFold(templates[i].name, in.Template) {
			target = &templates[i]
		}
	}
	for _, t := range templates {
		if strings.EqualFold(t.name, name) && (target == nil || t.line != target.line) {
			return "", saveError{409, "a template with this name already exists"}
		}
	}
	if group := strings.TrimSpace(in.NamesGroup); strings.ContainsAny(group, " \t\r\n{}=\"#") {
		return "", saveError{422, "invalid division names group"}
	}
	if target == nil {
		if in.Template != "" {
			return "", saveError{404, "template not found in the battle order file"}
		}
		block := newTemplateText(in, name)
		at := len(text)
		for _, word := range scriptWords(text) {
			if word.depth == 0 && strings.EqualFold(word.text, "units") {
				at = strings.LastIndexByte(text[:word.start], '\n') + 1
				break
			}
		}
		if at == len(text) && at > 0 && text[at-1] != '\n' {
			block = "\n" + block
		}
		return text[:at] + block + text[at:], nil
	}
	var extra [][]designerSaveCell
	if in.RegimentalSupport != nil {
		extra = append(extra, in.RegimentalSupport)
	}
	out, err := hqTemplateEdit(text, target.line, target.name, name, in.Regiments, in.Support, extra...)
	if err != nil {
		return "", saveError{422, err.Error()}
	}
	if group := strings.TrimSpace(in.NamesGroup); group != "" {
		if out, err = templateScalarEdit(out, target.line, "division_names_group", group, ""); err != nil {
			return "", saveError{422, err.Error()}
		}
	}
	if in.Priority != nil {
		if out, err = templateScalarEdit(out, target.line, "priority", fmt.Sprint(*in.Priority), "1"); err != nil {
			return "", saveError{422, err.Error()}
		}
	}
	if target.name != name {
		out = renameTemplateRefs(out, target.name, name)
	}
	return out, nil
}

func saveDesignerTemplateInFile(st *store, index designerSaveIndex, in designerSaveInput, tag, name, oob string) (designerSaveResult, bool, error) {
	dest, err := sanitizedPath("history/units/" + oob + ".txt")
	if err != nil {
		return designerSaveResult{}, false, nil
	}
	var raw *gameview.RawFile
	for _, file := range st.raw.list() {
		if strings.EqualFold(file.DestPath, dest) && !file.IsBinary &&
			(file.RawKind == "" || file.RawKind == "verbatim_override") {
			raw = &file
			break
		}
	}
	if raw == nil {
		return designerSaveResult{}, false, nil
	}
	for _, cell := range append(append(append([]designerSaveCell{}, in.Regiments...), in.Support...), in.RegimentalSupport...) {
		if strings.ContainsAny(cell.Type, " \t\r\n{}=\"#") {
			return designerSaveResult{}, true, saveError{422, "invalid battalion cell"}
		}
	}

	text := raw.Content
	if body, ok := index.OOBSource(oob); ok {
		text = string(body)
	}
	if strings.TrimSpace(text) == "" {
		return designerSaveResult{}, true, saveError{422, "the battle order file's text is not available"}
	}
	out, err := oobTemplateEdit(text, in, name)
	if err != nil {
		return designerSaveResult{}, true, err
	}
	if err := gateText(raw.DestPath, out); err != nil {
		return designerSaveResult{}, true, err
	}
	note := raw.Note
	if note == "" {
		note = designerRawFileNote
	}
	if _, err := st.raw.update(raw.ID, gameview.RawFileIntent{
		RawKind: raw.RawKind, DestPath: raw.DestPath, Content: out, GenerateReplacePath: raw.GenerateReplacePath,
		TargetLayer: raw.TargetLayer, TargetSHA256: raw.TargetSHA256,
	}, &note); err != nil {
		return designerSaveResult{}, true, err
	}
	return designerSaveResult{
		Template: divisionTemplate{TemplateName: name, CountryTag: tag, OobFile: oob},
		OobFile:  oob,
	}, true, nil
}
