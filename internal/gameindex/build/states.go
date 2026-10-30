package build

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/gfx"
)

const (
	tokLBrace = iota
	tokRBrace
	tokEquals
	tokString
	tokWord
)

type token struct {
	kind  int
	value string
}

func tokenize(text string) []token {
	var out []token
	i, n := 0, len(text)
	for i < n {
		c := text[i]
		switch c {
		case ' ', '\t', '\r', '\n':
			i++
			continue
		case '#':
			if nl := strings.IndexByte(text[i:], '\n'); nl >= 0 {
				i += nl + 1
			} else {
				i = n
			}
			continue
		case '{':
			out = append(out, token{tokLBrace, "{"})
			i++
			continue
		case '}':
			out = append(out, token{tokRBrace, "}"})
			i++
			continue
		case '=':
			out = append(out, token{tokEquals, "="})
			i++
			continue
		case '"':
			end := strings.IndexByte(text[i+1:], '"')
			if end < 0 {
				end = strings.IndexByte(text[i+1:], '\n')
				if end < 0 {
					end = n - i - 1
				}
				out = append(out, token{tokString, text[i+1 : i+1+end]})
				i = i + 1 + end
				continue
			}
			out = append(out, token{tokString, text[i+1 : i+1+end]})
			i = i + 1 + end + 1
			continue
		}
		start := i
		for i < n && !strings.ContainsRune(" \t\r\n{}=#\"", rune(text[i])) {
			i++
		}
		if i == start {
			i++
			continue
		}
		out = append(out, token{tokWord, text[start:i]})
	}
	return out
}

type tokenBlock struct {
	key   string
	inner []token
}

func topLevelBlocks(tokens []token) []tokenBlock {
	var out []tokenBlock
	for i, total := 0, len(tokens); i < total; {
		t := tokens[i]
		if t.kind == tokWord && i+2 < total && tokens[i+1].kind == tokEquals && tokens[i+2].kind == tokLBrace {
			depth, cursor := 0, i+2
			for cursor < total {
				if tokens[cursor].kind == tokLBrace {
					depth++
				} else if tokens[cursor].kind == tokRBrace {
					depth--
					if depth == 0 {
						break
					}
				}
				cursor++
			}
			out = append(out, tokenBlock{t.value, tokens[i+3 : min(cursor, total)]})
			i = cursor + 1
			continue
		}
		i++
	}
	return out
}

func directChildInt(tokens []token, keyword string) (int, bool) {
	depth := 0
	for i, t := range tokens {
		switch t.kind {
		case tokLBrace:
			depth++
			continue
		case tokRBrace:
			depth--
			continue
		}
		if depth != 0 || t.kind != tokWord || t.value != keyword {
			continue
		}
		if i+2 < len(tokens) && tokens[i+1].kind == tokEquals {
			if v := tokens[i+2]; v.kind == tokWord {
				n, err := strconv.Atoi(v.value)
				return n, err == nil
			}
		}
	}
	return 0, false
}

func scalarAssignments(tokens []token) [][2]string {
	var out [][2]string
	for i := 0; i+2 < len(tokens); {
		k, eq, v := tokens[i], tokens[i+1], tokens[i+2]
		if k.kind == tokWord && eq.kind == tokEquals && (v.kind == tokWord || v.kind == tokString) {
			out = append(out, [2]string{k.value, v.value})
			i += 3
			continue
		}
		i++
	}
	return out
}

func countWords(tokens []token, words ...string) int {
	n := 0
	for _, t := range tokens {
		if t.kind != tokWord {
			continue
		}
		for _, w := range words {
			if t.value == w {
				n++
				break
			}
		}
	}
	return n
}

func readStateTokens(in *gameindex.Input) [][]token {
	var out [][]token
	for _, rel := range baseFiles(in, "history/states", ".txt") {
		raw, err := readRaw(in, rel)
		if err != nil {
			continue
		}
		text, ok := gfx.DecodeScriptText(raw)
		if !ok {
			text = pyText(raw)
		}
		out = append(out, tokenize(text))
	}
	return out
}

type stateContent struct {
	NavalBases     int      `json:"navalBases,omitempty"`
	Bunkers        int      `json:"bunkers,omitempty"`
	CoastalBunkers int      `json:"coastalBunkers,omitempty"`
	Dams           int      `json:"dams,omitempty"`
	Landmarks      []string `json:"landmarks,omitempty"`
	DatedBlocks    int      `json:"datedBlocks,omitempty"`
	HasDLC         int      `json:"hasDlc,omitempty"`
	ForceLink      bool     `json:"forceLink,omitempty"`
	Resistance     int      `json:"resistance,omitempty"`
	OtherContent   int      `json:"otherContent,omitempty"`
}

func (h *stateContent) empty() bool {
	return h.NavalBases == 0 && h.Bunkers == 0 && h.CoastalBunkers == 0 && h.Dams == 0 && len(h.Landmarks) == 0 &&
		h.DatedBlocks == 0 && h.HasDLC == 0 && !h.ForceLink && h.Resistance == 0 && h.OtherContent == 0
}

var (
	stateDateRe   = regexp.MustCompile(`^\d{3,4}\.\d{1,2}\.\d{1,2}(\.\d{1,2})?$`)
	provinceKeyRe = regexp.MustCompile(`^\d+$`)
)

func scanStateBuildings(tokens []token, h *stateContent) {
	for _, child := range topLevelBlocks(tokens) {
		if !provinceKeyRe.MatchString(child.key) {
			continue
		}
		for _, kv := range scalarAssignments(child.inner) {
			switch kv[0] {
			case "naval_base":
				h.NavalBases++
			case "bunker":
				h.Bunkers++
			case "coastal_bunker":
				h.CoastalBunkers++
			case "dam":
				h.Dams++
			case "level", "has_dlc":
			default:
				h.OtherContent++
			}
		}
		for _, b := range topLevelBlocks(child.inner) {
			if strings.HasPrefix(b.key, "landmark_") {
				h.Landmarks = append(h.Landmarks, b.key)
			} else {
				h.OtherContent++
			}
		}
	}
}

func scanStateHistory(tokens []token, h *stateContent) {
	for _, child := range topLevelBlocks(tokens) {
		if stateDateRe.MatchString(child.key) {
			h.DatedBlocks++
		}
		if child.key == "buildings" {
			scanStateBuildings(child.inner, h)
		} else {
			scanStateHistory(child.inner, h)
		}
	}
}

func buildStateContent(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	states := map[string]*stateContent{}
	order := map[string]int{}
	files := readStateTokens(in)
	if len(files) == 0 {
		return nil, missingFile("history/states")
	}
	for _, tokens := range files {
		for _, block := range topLevelBlocks(tokens) {
			if block.key != "state" {
				continue
			}
			id := ""
			for _, kv := range scalarAssignments(block.inner) {
				if kv[0] == "id" {
					if n, err := strconv.Atoi(kv[1]); err == nil {
						id = strconv.Itoa(n)
					}
					break
				}
			}
			if id == "" {
				continue
			}
			h := &stateContent{}
			h.HasDLC = countWords(block.inner, "has_dlc")
			h.Resistance = countWords(block.inner, "set_compliance", "set_resistance", "start_resistance")
			for _, child := range topLevelBlocks(block.inner) {
				switch child.key {
				case "history":
					scanStateHistory(child.inner, h)
				case "buildings":
					scanStateBuildings(child.inner, h)
				}
			}
			h.ForceLink = countWords(block.inner, "force_link_ownership_to") > 0
			if h.empty() {
				continue
			}
			sort.Strings(h.Landmarks)
			states[id] = h
			n, _ := strconv.Atoi(id)
			order[id] = n
		}
		if err := checkCtx(ctx); err != nil {
			return nil, err
		}
	}
	ids := make([]string, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return order[ids[i]] < order[ids[j]] })
	out := newOrdered()
	for _, id := range ids {
		out.set(id, states[id])
	}
	return marshal(map[string]any{"game_version": in.GameVersion, "states": out})
}

func buildDLCStates(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	states := map[int][]string{}
	names := map[string]bool{}
	files := readStateTokens(in)
	if len(files) == 0 {
		return nil, missingFile("history/states")
	}
	for _, tokens := range files {
		for _, block := range topLevelBlocks(tokens) {
			if block.key != "state" {
				continue
			}
			id, ok := directChildInt(block.inner, "id")
			if !ok {
				continue
			}
			var found []string
			for i, t := range block.inner {
				if t.kind != tokWord || t.value != "has_dlc" || i+2 >= len(block.inner) || block.inner[i+1].kind != tokEquals {
					continue
				}
				v := block.inner[i+2]
				if (v.kind == tokString || v.kind == tokWord) && strings.TrimSpace(v.value) != "" {
					found = append(found, strings.TrimSpace(v.value))
				}
			}
			if len(found) == 0 {
				continue
			}
			states[id] = append(states[id], found...)
			for _, name := range found {
				names[name] = true
			}
		}
	}
	ids := make([]int, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := newOrdered()
	for _, id := range ids {
		out.set(strconv.Itoa(id), states[id])
	}
	return marshal(map[string]any{"game_version": in.GameVersion, "dlc_names": sortedKeys(names), "states": out})
}
