package gamesave

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type span struct{ start, end int }

type scriptWord struct {
	text       string
	start, end int
	depth      int
}

func scriptWords(text string) []scriptWord {
	words := []scriptWord{}
	depth := 0
	for i := 0; i < len(text); {
		c := text[i]
		switch {
		case c == '#':
			for i < len(text) && text[i] != '\n' {
				i++
			}
		case c == '"':
			start := i
			i++
			for i < len(text) && text[i] != '"' {
				if text[i] == '\\' {
					i++
				}
				i++
			}
			i++
			words = append(words, scriptWord{text[start:min(i, len(text))], start, min(i, len(text)), depth})
		case c == '{':
			words = append(words, scriptWord{"{", i, i + 1, depth})
			depth++
			i++
		case c == '}':
			depth--
			words = append(words, scriptWord{"}", i, i + 1, depth})
			i++
		case c == '=' || c == ' ' || c == '\t' || c == '\r' || c == '\n':
			if c == '=' {
				words = append(words, scriptWord{"=", i, i + 1, depth})
			}
			i++
		default:
			start := i
			for i < len(text) && !strings.ContainsRune(" \t\r\n={}#\"", rune(text[i])) {
				i++
			}
			words = append(words, scriptWord{text[start:i], start, i, depth})
		}
	}
	return words
}

func lineStart(text string, line int) int {
	at := 0
	for n := 1; n < line; n++ {
		next := strings.IndexByte(text[at:], '\n')
		if next < 0 {
			return len(text)
		}
		at += next + 1
	}
	return at
}

func indentOf(text string, offset int) string {
	start := strings.LastIndexByte(text[:offset], '\n') + 1
	end := start
	for end < len(text) && (text[end] == ' ' || text[end] == '\t') {
		end++
	}
	return text[start:end]
}

func closeOf(words []scriptWord, open int) int {
	for j := open + 1; j < len(words); j++ {
		if words[j].text == "}" && words[j].depth == words[open].depth {
			return j
		}
	}
	return -1
}

func blockBounds(words []scriptWord, text string, line int, name string) (open, close int, err error) {
	from, to := lineStart(text, line), lineStart(text, line+1)
	for i := 0; i+2 < len(words); i++ {
		if words[i].start < from || words[i].start >= to || !strings.EqualFold(words[i].text, name) || words[i+1].text != "=" || words[i+2].text != "{" {
			continue
		}
		open = i + 2
		for j := open + 1; j < len(words); j++ {
			if words[j].text == "}" && words[j].depth == words[open].depth {
				return open, j, nil
			}
		}
		return 0, 0, fmt.Errorf("the block of %s is not closed", name)
	}
	return 0, 0, fmt.Errorf("%s is not declared on line %d", name, line)
}

func assignments(words []scriptWord, open, close int) map[string][2]int {
	out := map[string][2]int{}
	depth := words[open].depth + 1
	for i := open + 1; i+2 <= close; i++ {
		if words[i].depth != depth || words[i+1].text != "=" || words[i+2].text == "{" {
			continue
		}
		if i > open+1 && words[i-1].text == "=" {
			continue
		}
		out[strings.ToLower(words[i].text)] = [2]int{i, i + 2}
	}
	return out
}

func childBlock(words []scriptWord, open, close int, key string) (int, int, bool) {
	depth := words[open].depth + 1
	for i := open + 1; i+2 < close; i++ {
		if words[i].depth == depth && strings.EqualFold(words[i].text, key) && words[i+1].text == "=" && words[i+2].text == "{" {
			for j := i + 3; j <= close; j++ {
				if words[j].text == "}" && words[j].depth == depth {
					return i, j, true
				}
			}
		}
	}
	return 0, 0, false
}

func childBlocks(words []scriptWord, open, close int, key string) [][2]int {
	out := [][2]int{}
	depth := words[open].depth + 1
	for i := open + 1; i+2 < close; i++ {
		if words[i].depth != depth || !strings.EqualFold(words[i].text, key) || words[i+1].text != "=" || words[i+2].text != "{" {
			continue
		}
		for j := i + 3; j <= close; j++ {
			if words[j].text == "}" && words[j].depth == depth {
				out = append(out, [2]int{i, j})
				i = j
				break
			}
		}
	}
	return out
}

func blockInner(text string, words []scriptWord, open, close int) string {
	for _, w := range words[open+1 : close] {
		if w.depth == words[open].depth+1 {
			return indentOf(text, w.start)
		}
	}
	return indentOf(text, words[open].start) + "\t"
}

type edit struct {
	start, end int
	text       string
}

func applyEdits(text string, edits []edit) string {
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := text
	for _, e := range edits {
		out = out[:e.start] + e.text + out[e.end:]
	}
	return out
}

func removal(text string, words []scriptWord, key, value int) edit {
	lineFrom := strings.LastIndexByte(text[:words[key].start], '\n') + 1
	lineTo := strings.IndexByte(text[words[value].end:], '\n')
	if lineTo < 0 {
		lineTo = len(text)
	} else {
		lineTo += words[value].end + 1
	}
	before := strings.TrimSpace(text[lineFrom:words[key].start])
	after := strings.TrimSpace(text[words[value].end:lineTo])
	if before == "" && (after == "" || strings.HasPrefix(after, "#")) {
		return edit{lineFrom, lineTo, ""}
	}
	return edit{words[key].start, words[value].end, ""}
}

func numberText(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func setNumbers(text string, words []scriptWord, open, close int, values map[string]*float64, indent string) []edit {
	edits := []edit{}
	have := assignments(words, open, close)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	added := ""
	for _, key := range keys {
		value := values[key]
		at, found := have[key]
		switch {
		case found && value == nil:
			edits = append(edits, removal(text, words, at[0], at[1]))
		case found:
			edits = append(edits, edit{words[at[1]].start, words[at[1]].end, numberText(*value)})
		case value != nil:
			added += indent + key + " = " + numberText(*value) + "\n"
		}
	}
	if added != "" {
		closeAt := words[close].start
		lineFrom := strings.LastIndexByte(text[:closeAt], '\n') + 1
		if strings.TrimSpace(text[lineFrom:closeAt]) == "" {
			edits = append(edits, edit{lineFrom, lineFrom, added})
		} else {
			edits = append(edits, edit{closeAt, closeAt, "\n" + added + indent[:max(0, len(indent)-1)]})
		}
	}
	return edits
}

func insertBeforeClose(text string, words []scriptWord, close int, added []string, inner string) []edit {
	if len(added) == 0 {
		return nil
	}
	closeAt := words[close].start
	lineFrom := strings.LastIndexByte(text[:closeAt], '\n') + 1
	block := ""
	for _, line := range added {
		block += inner + line + "\n"
	}
	if strings.TrimSpace(text[lineFrom:closeAt]) == "" {
		return []edit{{lineFrom, lineFrom, block}}
	}
	return []edit{{closeAt, closeAt, "\n" + block}}
}

func conditionText(key, condition, indent string) string {
	if !strings.Contains(condition, "\n") {
		return key + " = { " + condition + " }"
	}
	lines := strings.Split(condition, "\n")
	for i, line := range lines {
		lines[i] = indent + "\t" + line
	}
	return key + " = {\n" + strings.Join(lines, "\n") + "\n" + indent + "}"
}

func scriptBlockEdits(text string, words []scriptWord, open, close int, key string, value *string, inner string) (edits []edit, added []string) {
	if value == nil {
		return nil, nil
	}
	found := childBlocks(words, open, close, key)
	keepLast := *value != ""
	for at, span := range found {
		if keepLast && at == len(found)-1 {
			edits = append(edits, edit{words[span[0]].start, words[span[1]].end, conditionText(key, *value, inner)})
			continue
		}
		edits = append(edits, removal(text, words, span[0], span[1]))
	}
	if keepLast && len(found) == 0 {
		added = append(added, conditionText(key, *value, inner))
	}
	return edits, added
}

func cleanScript(text string) (string, error) {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if text == "" {
		return "", nil
	}
	if _, recoveries, err := paradox.Parse(text); err != nil || len(recoveries) > 0 {
		return "", errors.New("the condition is not valid script")
	}
	return text, nil
}
