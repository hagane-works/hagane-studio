package gfx

import (
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/paradox"
)

const fallbackSuspicionSize = 4096

func pySpace(r rune) bool {
	return (r >= 0x1c && r <= 0x1f) || unicode.IsSpace(r)
}

func pyAlpha(r rune) bool { return unicode.IsLetter(r) }

func pyAlnum(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }

func pyWord(r rune) bool { return pyAlnum(r) || r == '_' }

func asciiKeyStart(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func asciiKeyChar(r rune) bool { return asciiKeyStart(r) || (r >= '0' && r <= '9') }

func stripComments(text []rune) []rune {
	out := make([]rune, 0, len(text))
	i, n := 0, len(text)
	inString := false
	for i < n {
		c := text[i]
		if inString {
			out = append(out, c)
			if c == '\\' && i+1 < n {
				out = append(out, text[i+1])
				i += 2
				continue
			}
			if c == '"' {
				inString = false
			}
			i++
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			i++
			continue
		}
		if c == '#' {
			for i < n && text[i] != '\n' {
				i++
			}
			out = append(out, ' ')
			continue
		}
		out = append(out, c)
		i++
	}
	return out
}

func StripCommentsAndSkipQuotes(text string) string {
	return string(stripComments([]rune(text)))
}

func findMatchingBrace(text []rune, start int) int {
	if start >= len(text) || text[start] != '{' {
		return -1
	}
	depth := 0
	i, n := start, len(text)
	inString := false
	for i < n {
		c := text[i]
		if inString {
			if c == '\\' && i+1 < n {
				i += 2
				continue
			}
			if c == '"' {
				inString = false
			}
			i++
			continue
		}
		if c == '"' {
			inString = true
			i++
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

func matchHeaderAt(t []rune, i int, tail string) (int, bool) {
	if t[i] != 's' && t[i] != 'S' {
		return 0, false
	}
	if i > 0 && pyWord(t[i-1]) {
		return 0, false
	}
	j := i + 1
	for _, r := range tail {
		if j >= len(t) || t[j] != r {
			return 0, false
		}
		j++
	}
	for j < len(t) && pySpace(t[j]) {
		j++
	}
	if j >= len(t) || t[j] != '=' {
		return 0, false
	}
	j++
	for j < len(t) && pySpace(t[j]) {
		j++
	}
	if j >= len(t) || t[j] != '{' {
		return 0, false
	}
	return j + 1, true
}

func assignMatch(t []rune, i int) (key, value string, end int, ok bool) {
	if i >= len(t) || !asciiKeyStart(t[i]) {
		return "", "", 0, false
	}
	j := i + 1
	for j < len(t) && asciiKeyChar(t[j]) {
		j++
	}
	key = string(t[i:j])
	k := j
	for k < len(t) && pySpace(t[k]) {
		k++
	}
	if k >= len(t) || t[k] != '=' {
		return "", "", 0, false
	}
	k++
	for k < len(t) && pySpace(t[k]) {
		k++
	}
	if k >= len(t) {
		return "", "", 0, false
	}
	if t[k] == '"' {
		m := k + 1
		for m < len(t) && t[m] != '"' {
			m++
		}
		if m >= len(t) {
			return "", "", 0, false
		}
		return key, string(t[k+1 : m]), m + 1, true
	}
	m := k
	for m < len(t) {
		r := t[m]
		if pySpace(r) || r == '{' || r == '}' || r == '"' || r == '#' {
			break
		}
		m++
	}
	if m == k {
		return "", "", 0, false
	}
	return key, string(t[k:m]), m, true
}

func extractNameTexturefile(block []rune, mapping map[string]string) {
	name, texturefile := "", ""
	i, bn := 0, len(block)
	for i < bn {
		c := block[i]
		if c == '{' {
			close := findMatchingBrace(block, i)
			if close < 0 {
				break
			}
			i = close + 1
			continue
		}
		if c == '"' {
			i++
			for i < bn && block[i] != '"' {
				if block[i] == '\\' && i+1 < bn {
					i += 2
				} else {
					i++
				}
			}
			if i < bn {
				i++
			}
			continue
		}
		if pyAlpha(c) || c == '_' {
			if key, val, end, ok := assignMatch(block, i); ok {
				if key == "name" {
					name = val
				} else if key == "texturefile" {
					texturefile = val
				}
				i = end
				continue
			}
			for i < bn && (pyAlnum(block[i]) || block[i] == '_') {
				i++
			}
			continue
		}
		i++
	}
	if name != "" && texturefile != "" {
		mapping[name] = texturefile
	}
}

func parseGfxFast(text []rune) map[string]string {
	mapping := map[string]string{}
	cleaned := stripComments(text)

	type span struct{ start, end int }
	var roots []span
	for i := 0; i < len(cleaned); {
		end, ok := matchHeaderAt(cleaned, i, "priteTypes")
		if !ok {
			i++
			continue
		}
		rootOpen := end - 1
		if rootClose := findMatchingBrace(cleaned, rootOpen); rootClose >= 0 {
			roots = append(roots, span{rootOpen + 1, rootClose})
		}
		i = end
	}

	for _, rt := range roots {
		rootText := cleaned[rt.start:rt.end]
		for i := 0; i < len(rootText); {
			end, ok := matchHeaderAt(rootText, i, "priteType")
			if !ok {
				i++
				continue
			}
			blockOpen := end - 1
			blockClose := findMatchingBrace(rootText, blockOpen)
			i = end
			if blockClose < 0 {
				continue
			}
			extractNameTexturefile(rootText[blockOpen+1:blockClose], mapping)
		}
	}
	return mapping
}

func blockEntries(entries []*paradox.Entry, key string) []*paradox.Entry {
	for _, e := range entries {
		if e.Key != nil && *e.Key == key {
			if e.Block == nil {
				return nil
			}
			return e.Block.Entries
		}
	}
	return nil
}

func scalarStr(entries []*paradox.Entry, key string) string {
	for _, e := range entries {
		if e.Key != nil && *e.Key == key {
			if e.Block != nil {
				return ""
			}
			return e.Scalar.StrValue()
		}
	}
	return ""
}

func parseGfxViaAST(text string) (map[string]string, error) {
	mapping := map[string]string{}
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, err
	}

	st := blockEntries(ast.Entries, "spriteTypes")
	if len(st) == 0 {
		st = blockEntries(ast.Entries, "SpriteTypes")
	}
	if len(st) == 0 {
		return mapping, nil
	}

	for _, variant := range []string{"spriteType", "SpriteType"} {
		for _, e := range st {
			if e.Key == nil || *e.Key != variant || e.Block == nil {
				continue
			}
			body := e.Block.Entries
			name := scalarStr(body, "name")
			texturefile := scalarStr(body, "texturefile")
			if name != "" && texturefile != "" {

				mapping[strings.Trim(name, `"`)] = strings.Trim(texturefile, `"`)
			}
		}
	}
	return mapping, nil
}

func ParseGfxMappings(text string) map[string]string {
	if text == "" {
		return map[string]string{}
	}
	runes := []rune(text)
	mapping := parseGfxFast(runes)
	if len(mapping) == 0 && len(runes) >= fallbackSuspicionSize {
		astMapping, err := parseGfxViaAST(text)
		if err != nil {
			return mapping
		}
		return astMapping
	}
	return mapping
}

func ParseMultipleGfxFiles(texts []string) map[string]string {
	merged := map[string]string{}
	for _, t := range texts {
		for k, v := range ParseGfxMappings(t) {
			merged[k] = v
		}
	}
	return merged
}
