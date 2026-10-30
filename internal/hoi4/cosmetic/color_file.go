package cosmetic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"haganeworks/internal/gameindex"
)

func VanillaColorFile(ctx context.Context) ([]byte, error) {
	raw, err := gameindex.Get(ctx, gameindex.Cosmetic)
	if errors.Is(err, gameindex.ErrNoGame) {
		return nil, fmt.Errorf("writing cosmetic.txt needs the base game's own file: %w", err)
	}
	if err != nil {
		return nil, err
	}
	var file struct {
		Bytes []byte `json:"bytes"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, err
	}
	return file.Bytes, nil
}

type colorSpan struct {
	key        string
	start, end int
}

func colorSpans(source []byte) ([]colorSpan, error) {
	type token struct {
		value      string
		start, end int
	}
	var top []token
	var spans []colorSpan
	depth, start := 0, 0
	key := ""
	for i := 0; i < len(source); {
		if i == 0 && bytes.HasPrefix(source, []byte{0xef, 0xbb, 0xbf}) {
			i = 3
			continue
		}
		c := source[i]
		if c == '#' {
			for i < len(source) && source[i] != '\n' {
				i++
			}
			continue
		}
		if strings.ContainsRune(" \t\r\n", rune(c)) {
			i++
			continue
		}
		if c == '{' {
			if depth == 0 {
				if len(top) < 2 || top[len(top)-1].value != "=" {
					return nil, fmt.Errorf("cosmetic block without tag at byte %d", i)
				}
				key, start = top[len(top)-2].value, top[len(top)-2].start
				top = nil
			}
			depth++
			i++
			continue
		}
		if c == '}' {
			if depth == 0 {
				return nil, fmt.Errorf("stray cosmetic closing brace at byte %d", i)
			}
			depth--
			i++
			if depth == 0 {
				spans = append(spans, colorSpan{key, start, i})
				top = nil
			}
			continue
		}
		begin := i
		value := ""
		if c == '"' {
			i++
			var word strings.Builder
			closed := false
			for i < len(source) {
				if source[i] == '\\' && i+1 < len(source) {
					word.WriteByte(source[i+1])
					i += 2
					continue
				}
				if source[i] == '"' {
					i++
					closed = true
					break
				}
				word.WriteByte(source[i])
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated cosmetic string at byte %d", begin)
			}
			value = word.String()
		} else if c == '=' {
			i++
			value = "="
		} else {
			for i < len(source) && !strings.ContainsRune(" \t\r\n#={}\"", rune(source[i])) {
				i++
			}
			value = string(source[begin:i])
		}
		if depth == 0 {
			top = append(top, token{value, begin, i})
			if len(top) > 2 {
				top = top[len(top)-2:]
			}
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unclosed cosmetic block")
	}
	return spans, nil
}

func ColorFileTags(source []byte) ([]string, error) {
	spans, err := colorSpans(source)
	if err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(spans))
	seen := map[string]bool{}
	for _, span := range spans {
		if seen[span.key] {
			return nil, fmt.Errorf("duplicate generated cosmetic tag %q", span.key)
		}
		seen[span.key] = true
		tags = append(tags, span.key)
	}
	return tags, nil
}

func MergeColorFiles(base, generated []byte, tags []string) ([]byte, error) {
	baseSpans, err := colorSpans(base)
	if err != nil {
		return nil, err
	}
	newSpans, err := colorSpans(generated)
	if err != nil {
		return nil, err
	}
	wanted := map[string]bool{}
	for _, tag := range tags {
		wanted[tag] = true
	}
	replacements := map[string][]byte{}
	for _, span := range newSpans {
		if wanted[span.key] {
			if _, exists := replacements[span.key]; exists {
				return nil, fmt.Errorf("duplicate replacement cosmetic tag %q", span.key)
			}
			replacements[span.key] = generated[span.start:span.end]
		}
	}
	for tag := range wanted {
		if replacements[tag] == nil {
			return nil, fmt.Errorf("missing replacement cosmetic tag %q", tag)
		}
	}
	var out bytes.Buffer
	cursor := 0
	written := map[string]bool{}
	for _, span := range baseSpans {
		if !wanted[span.key] {
			continue
		}
		out.Write(base[cursor:span.start])
		if !written[span.key] {
			out.Write(replacements[span.key])
			written[span.key] = true
		}
		cursor = span.end
	}
	out.Write(base[cursor:])
	for _, tag := range tags {
		if !written[tag] {
			out.WriteString("\n")
			out.Write(replacements[tag])
			out.WriteString("\n")
			written[tag] = true
		}
	}
	return out.Bytes(), nil
}
