package moddesc

import (
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/emit/locout"
)

func pyIsSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return true
	}
	return unicode.IsSpace(r)
}

func pyStrip(s string) string { return strings.TrimFunc(s, pyIsSpace) }

func pyLower(s string) string { return strings.ToLower(s) }

func GenerateDescriptor(project *locout.Obj, replacePaths []string) string {
	version := pyStrip(strAttrOr(project, "version"))
	if version == "" {
		version = "1.0"
	}
	supportedVersion := pyStrip(strAttrOr(project, "supported_version"))
	if supportedVersion == "" {
		supportedVersion = "1.19.*"
	}
	picture := pyStrip(strAttrOr(project, "picture"))

	tagsVal := project.Attr("tags")
	var tags []locout.Value
	if tagsVal.Kind == locout.KArr && len(tagsVal.Arr) > 0 {
		tags = tagsVal.Arr
	} else {
		tags = []locout.Value{locout.Str("Alternative History")}
	}
	depsVal := project.Attr("dependencies")
	var deps []locout.Value
	if locout.PyTruthy(depsVal) && depsVal.Kind == locout.KArr {
		deps = depsVal.Arr
	}

	lines := []string{`version="` + version + `"`}
	lines = append(lines, "tags={")
	for _, t := range tags {
		if t.Kind == locout.KStr && t.Str != "" {
			lines = append(lines, "\t\""+t.Str+"\"")
		}
	}
	lines = append(lines, "}")
	nameVal := project.Attr("name")
	name := ""
	if locout.PyTruthy(nameVal) && nameVal.Kind == locout.KStr {
		name = nameVal.Str
	}
	safeName := strings.ReplaceAll(name, "\\", "\\\\")
	safeName = strings.ReplaceAll(safeName, "\"", "\\\"")
	lines = append(lines, `name="`+safeName+`"`)
	lines = append(lines, `supported_version="`+supportedVersion+`"`)
	if picture != "" {
		lines = append(lines, `picture="`+picture+`"`)
	}
	cleaned := []string{}
	for _, d := range deps {
		if d.Kind == locout.KStr && pyStrip(d.Str) != "" {
			cleaned = append(cleaned, d.Str)
		}
	}
	if len(cleaned) > 0 {
		lines = append(lines, "dependencies={")
		for _, d := range cleaned {
			lines = append(lines, "\t\""+d+"\"")
		}
		lines = append(lines, "}")
	}
	for _, rp := range replacePaths {
		lines = append(lines, `replace_path="`+rp+`"`)
	}
	return strings.Join(lines, "\n")
}

func strAttrOr(o *locout.Obj, attr string) string {
	v := o.Attr(attr)
	if !locout.PyTruthy(v) {
		return ""
	}
	if v.Kind == locout.KStr {
		return v.Str
	}
	return locout.PyStr(v)
}

func IsOnActionsPath(path string) bool {
	norm := strings.ReplaceAll(path, "\\", "/")
	return strings.Contains(norm, "common/on_actions/") &&
		strings.HasSuffix(pyLower(norm), ".txt")
}

func HaganeOnActionsSibling(path string) string {
	norm := strings.ReplaceAll(path, "\\", "/")
	if strings.Contains(norm, "_hagane_on_actions.txt") {
		return norm
	}
	if strings.Contains(norm, "_on_actions.txt") {
		return strings.Replace(norm, "_on_actions.txt", "_hagane_on_actions.txt", 1)
	}
	if strings.HasSuffix(pyLower(norm), ".txt") {
		return norm[:len(norm)-4] + "_hagane.txt"
	}
	return norm + "_hagane"
}

func IsDescriptorPath(path string) bool {
	norm := strings.ReplaceAll(path, "\\", "/")
	return strings.Contains(norm, "/") &&
		strings.HasSuffix(pyLower(norm), "/descriptor.mod")
}

type ParsedDescriptor struct {
	Scalars     map[string]string
	ScalarOrder []string
	Blocks      map[string][]string
	BlockOrder  []string
}

func isDescKeyRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_'
}

func ParseDescriptorMod(text string) ParsedDescriptor {
	p := ParsedDescriptor{
		Scalars: map[string]string{},
		Blocks:  map[string][]string{},
	}
	replacePaths := []string{}
	runes := []rune(text)
	n := len(runes)
	pos := 0
	for pos < n {

		if !isDescKeyRune(runes[pos]) {
			pos++
			continue
		}
		runStart := pos
		for pos < n && isDescKeyRune(runes[pos]) {
			pos++
		}
		runEnd := pos

		k := runEnd
		for k < n && pyIsSpace(runes[k]) {
			k++
		}
		if k >= n || runes[k] != '=' {
			continue
		}
		k++
		for k < n && pyIsSpace(runes[k]) {
			k++
		}
		key := string(runes[runStart:runEnd])
		vstart := k
		if vstart < n && runes[vstart] == '{' {
			depth := 0
			j := vstart
			for j < n {
				if runes[j] == '{' {
					depth++
				} else if runes[j] == '}' {
					depth--
					if depth == 0 {
						j++
						break
					}
				}
				j++
			}

			bodyEnd := j - 1
			if bodyEnd < vstart+1 {
				bodyEnd = vstart + 1
			}
			body := string(runes[vstart+1 : bodyEnd])
			items := findAllQuoted(body)
			if len(items) == 0 {
				items = splitWS(body)
			}
			if _, seen := p.Blocks[key]; !seen {
				p.BlockOrder = append(p.BlockOrder, key)
			}
			p.Blocks[key] = items
			pos = j
		} else {

			lineEnd := vstart
			for lineEnd < n && runes[lineEnd] != '\n' {
				lineEnd++
			}
			raw := pyStrip(string(runes[vstart:lineEnd]))
			if len(raw) >= 2 && strings.HasPrefix(raw, `"`) && strings.HasSuffix(raw, `"`) {
				raw = raw[1 : len(raw)-1]
			}
			if key == "replace_path" {
				replacePaths = append(replacePaths, raw)
			} else {
				if _, seen := p.Scalars[key]; !seen {
					p.ScalarOrder = append(p.ScalarOrder, key)
				}
				p.Scalars[key] = raw
			}
			pos = lineEnd
		}
	}
	if len(replacePaths) > 0 {
		if _, seen := p.Blocks["replace_path"]; !seen {
			p.BlockOrder = append(p.BlockOrder, "replace_path")
		}
		p.Blocks["replace_path"] = replacePaths
	}
	return p
}

func findAllQuoted(body string) []string {
	out := []string{}
	runes := []rune(body)
	n := len(runes)
	i := 0
	for i < n {
		if runes[i] != '"' {
			i++
			continue
		}
		j := i + 1
		for j < n && runes[j] != '"' {
			j++
		}
		if j >= n {
			break
		}
		out = append(out, string(runes[i+1:j]))
		i = j + 1
	}
	return out
}

func splitWS(s string) []string {
	fields := strings.FieldsFunc(s, pyIsSpace)
	if fields == nil {
		return []string{}
	}
	return fields
}

func MergeDescriptorMod(baseText, platformText string) string {
	base := ParseDescriptorMod(baseText)
	plat := ParseDescriptorMod(platformText)

	scalar := func(key string) (string, bool) {
		if v, ok := base.Scalars[key]; ok && v != "" {
			return v, true
		}
		v, ok := plat.Scalars[key]
		return v, ok
	}
	union := func(key string) []string {
		out := append([]string{}, base.Blocks[key]...)
		seen := map[string]bool{}
		for _, x := range out {
			seen[x] = true
		}
		for _, item := range plat.Blocks[key] {
			if !seen[item] {
				seen[item] = true
				out = append(out, item)
			}
		}
		return out
	}

	lines := []string{}
	if version, ok := scalar("version"); ok {
		lines = append(lines, `version="`+version+`"`)
	}
	tags := union("tags")
	if len(tags) > 0 {
		lines = append(lines, "tags={")
		for _, t := range tags {
			lines = append(lines, "\t\""+t+"\"")
		}
		lines = append(lines, "}")
	}
	if name, ok := scalar("name"); ok {
		lines = append(lines, `name="`+name+`"`)
	}
	if sv, ok := scalar("supported_version"); ok {
		lines = append(lines, `supported_version="`+sv+`"`)
	}
	if picture, ok := scalar("picture"); ok && picture != "" {
		lines = append(lines, `picture="`+picture+`"`)
	}
	deps := union("dependencies")
	if len(deps) > 0 {
		lines = append(lines, "dependencies={")
		for _, d := range deps {
			lines = append(lines, "\t\""+d+"\"")
		}
		lines = append(lines, "}")
	}
	for _, rp := range union("replace_path") {
		lines = append(lines, `replace_path="`+rp+`"`)
	}
	return strings.Join(lines, "\n") + "\n"
}

func CommonTopLevelPrefix(paths []string) (string, bool) {
	tops := map[string]bool{}
	for _, p := range paths {
		norm := strings.ReplaceAll(p, "\\", "/")
		i := strings.Index(norm, "/")
		if i < 0 {
			continue
		}
		tops[norm[:i]+"/"] = true
	}
	if len(tops) == 1 {
		for t := range tops {
			return t, true
		}
	}
	return "", false
}
