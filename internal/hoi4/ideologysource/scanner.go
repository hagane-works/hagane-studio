package ideologysource

import (
	"bytes"
	"fmt"
	"strings"
)

type tokenKind uint8

const (
	tokenWord tokenKind = iota
	tokenEquals
	tokenLBrace
	tokenRBrace
)

type token struct {
	kind       tokenKind
	value      string
	start, end int
	depth      int
}

type blockLocation struct {
	keyStart, openOffset, closeOffset, closeEnd int
	openToken, closeToken                       int
	depth                                       int
}

type rootLocation struct {
	Owner      RootOwner
	Block      blockLocation
	Types      []blockLocation
	SubtypeIDs []string
}

type discoveredFile struct {
	Path          string
	Bytes         []byte
	SourceKind    string
	DependencyKey string
	Roots         []rootLocation
}

func lex(source []byte) ([]token, error) {
	tokens := []token{}
	depth := 0
	for i := 0; i < len(source); {
		c := source[i]
		if i == 0 && len(source) >= 3 && bytes.Equal(source[:3], []byte{0xef, 0xbb, 0xbf}) {
			i = 3
			continue
		}
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
		switch c {
		case '=':
			tokens = append(tokens, token{kind: tokenEquals, start: i, end: i + 1, depth: depth})
			i++
			continue
		case '{':
			tokens = append(tokens, token{kind: tokenLBrace, start: i, end: i + 1, depth: depth})
			depth++
			i++
			continue
		case '}':
			if depth == 0 {
				return nil, fmt.Errorf("stray closing brace at byte %d", i)
			}
			depth--
			tokens = append(tokens, token{kind: tokenRBrace, start: i, end: i + 1, depth: depth})
			i++
			continue
		case '"':
			start := i
			i++
			var value strings.Builder
			closed := false
			for i < len(source) {
				if source[i] == '\\' && i+1 < len(source) {
					value.WriteByte(source[i+1])
					i += 2
					continue
				}
				if source[i] == '"' {
					i++
					closed = true
					break
				}
				value.WriteByte(source[i])
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated quoted string at byte %d", start)
			}
			tokens = append(tokens, token{kind: tokenWord, value: value.String(), start: start, end: i, depth: depth})
			continue
		}
		start := i
		for i < len(source) && !strings.ContainsRune(" \t\r\n#={}", rune(source[i])) {
			i++
		}
		if start == i {
			return nil, fmt.Errorf("unsupported byte %q at byte %d", source[i], i)
		}
		tokens = append(tokens, token{kind: tokenWord, value: string(source[start:i]), start: start, end: i, depth: depth})
	}
	if depth != 0 {
		return nil, fmt.Errorf("unclosed block depth %d", depth)
	}
	return tokens, nil
}

func matchingClose(tokens []token, open int) int {
	if open < 0 || open >= len(tokens) || tokens[open].kind != tokenLBrace {
		return -1
	}
	depth := 1
	for i := open + 1; i < len(tokens); i++ {
		switch tokens[i].kind {
		case tokenLBrace:
			depth++
		case tokenRBrace:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func assignmentBlock(tokens []token, index int) (blockLocation, bool) {
	if index+2 >= len(tokens) || tokens[index].kind != tokenWord || tokens[index+1].kind != tokenEquals || tokens[index+2].kind != tokenLBrace {
		return blockLocation{}, false
	}
	close := matchingClose(tokens, index+2)
	if close < 0 {
		return blockLocation{}, false
	}
	return blockLocation{keyStart: tokens[index].start, openOffset: tokens[index+2].start, closeOffset: tokens[close].start, closeEnd: tokens[close].end, openToken: index + 2, closeToken: close, depth: tokens[index+2].depth}, true
}

func directNamedBlocks(tokens []token, parent blockLocation, name string) []blockLocation {
	result := []blockLocation{}
	for i := parent.openToken + 1; i < parent.closeToken; i++ {
		if tokens[i].depth != parent.depth+1 || tokens[i].kind != tokenWord || (name != "" && tokens[i].value != name) {
			continue
		}
		if block, ok := assignmentBlock(tokens, i); ok && block.closeToken <= parent.closeToken {
			result = append(result, block)
			i = block.closeToken
		}
	}
	return result
}

func scanSource(path string, source []byte, sourceKind, dependencyKey string) (discoveredFile, error) {
	tokens, err := lex(source)
	if err != nil {
		return discoveredFile{}, err
	}
	file := discoveredFile{Path: normalizePath(path), Bytes: append([]byte(nil), source...), SourceKind: sourceKind, DependencyKey: dependencyKey}
	for i := 0; i < len(tokens); i++ {
		if tokens[i].depth != 0 || tokens[i].kind != tokenWord || tokens[i].value != "ideologies" {
			continue
		}
		wrapper, ok := assignmentBlock(tokens, i)
		if !ok {
			return discoveredFile{}, fmt.Errorf("ideologies must be a block")
		}
		for rootIndex := wrapper.openToken + 1; rootIndex < wrapper.closeToken; rootIndex++ {
			if tokens[rootIndex].depth != wrapper.depth+1 || tokens[rootIndex].kind != tokenWord {
				continue
			}
			root, ok := assignmentBlock(tokens, rootIndex)
			if !ok {
				continue
			}
			rootID := tokens[rootIndex].value
			types := directNamedBlocks(tokens, root, "types")
			subtypes := []string{}
			if len(types) == 1 {
				for j := types[0].openToken + 1; j < types[0].closeToken; j++ {
					if tokens[j].depth != types[0].depth+1 || tokens[j].kind != tokenWord {
						continue
					}
					if child, ok := assignmentBlock(tokens, j); ok {
						subtypes = append(subtypes, tokens[j].value)
						j = child.closeToken
					}
				}
			}
			rootBytes := source[root.keyStart:root.closeEnd]
			owner := RootOwner{RootID: rootID, Body: string(source[root.openOffset+1 : root.closeOffset]), SourceKind: sourceKind, DependencyKey: dependencyKey, OwnerPath: file.Path, FileSHA256: Hash(source), RootSHA256: Hash(rootBytes), SubtypeIDs: append([]string(nil), subtypes...), TypesCount: len(types)}
			file.Roots = append(file.Roots, rootLocation{Owner: owner, Block: root, Types: types, SubtypeIDs: subtypes})
			rootIndex = root.closeToken
		}
		i = wrapper.closeToken
	}
	return file, nil
}

func lineIndent(source []byte, offset int) string {
	start := offset
	for start > 0 && source[start-1] != '\n' {
		start--
	}
	end := start
	for end < len(source) && (source[end] == ' ' || source[end] == '\t') {
		end++
	}
	return string(source[start:end])
}

func childIndent(source []byte, block blockLocation) string {
	base := lineIndent(source, block.closeOffset)
	for i := block.openOffset + 1; i < block.closeOffset; i++ {
		if source[i] == '\t' {
			return base + "\t"
		}
		if source[i] == '\n' {
			j := i + 1
			spaces := 0
			for j < block.closeOffset && source[j] == ' ' {
				spaces++
				j++
			}
			if spaces > len(base) {
				return base + strings.Repeat(" ", spaces-len(base))
			}
		}
	}
	return base + "\t"
}

func lineEnding(source []byte) string {
	if bytes.Contains(source, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}
