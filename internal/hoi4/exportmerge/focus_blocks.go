package exportmerge

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/focusorder"
	treesvc "haganeworks/internal/hoi4/importsvc/trees"
)

type FocusSourceFile struct {
	Source   string
	Warnings []string
	text     string
	blocks   []*focusSourceBlock
	byID     map[string][]*focusSourceNode
}

type focusSourceBlock struct {
	treeID string
	kind   string
	nodes  []*focusSourceNode
}

type focusSourceNode struct {
	span  focusSpan
	row   map[string]any
	block *focusSourceBlock
}

type focusSpan struct {
	key         string
	start, end  int
	open, close int
}

func ParseFocusSourceFile(text, source string) (*FocusSourceFile, error) {
	tokens, err := focusTokens(text)
	if err != nil {
		return nil, err
	}
	bundles, summary, err := treesvc.ImportFocusFile(text, source)
	if err != nil {
		return nil, err
	}
	f := &FocusSourceFile{Source: source, text: text, byID: map[string][]*focusSourceNode{}}
	if summary != nil {
		f.Warnings = append([]string(nil), summary.Warnings...)
	}
	var primaryRows [][]map[string]any
	var primaryIDs []string
	var poolRows []map[string]any
	for _, bundle := range bundles {
		rows := make([]map[string]any, len(bundle.NodesData))
		for i, raw := range bundle.NodesData {
			rows[i] = nativeValue(raw).(map[string]any)
		}
		if bundle.TreeKind == "shared_pool" {
			poolRows = append(poolRows, rows...)
		} else {
			primaryRows = append(primaryRows, rows)
			primaryIDs = append(primaryIDs, asString(bundle.TreeData["tree_id"]))
		}
	}
	addNode := func(block *focusSourceBlock, span focusSpan, row map[string]any) {
		row["tree_id"] = block.treeID
		node := &focusSourceNode{span: span, row: row, block: block}
		block.nodes = append(block.nodes, node)
		id := stringField(row, "focus_id")
		f.byID[id] = append(f.byID[id], node)
	}
	primaryIndex, poolIndex := 0, 0
	var pool *focusSourceBlock
	for _, span := range focusEntries(tokens, 0, len(tokens)) {
		if span.open >= 0 && (span.key == "shared_focus" || span.key == "joint_focus") {
			if poolIndex >= len(poolRows) {
				return nil, fmt.Errorf("focus span/import mismatch in %s", source)
			}
			if pool == nil {
				pool = &focusSourceBlock{kind: "shared_pool"}
				f.blocks = append(f.blocks, pool)
			}
			addNode(pool, span, poolRows[poolIndex])
			poolIndex++
			continue
		}

		pool = nil
		if span.key != "focus_tree" || span.open < 0 {
			continue
		}
		if primaryIndex >= len(primaryRows) {
			return nil, fmt.Errorf("focus tree span/import mismatch in %s", source)
		}
		block := &focusSourceBlock{treeID: primaryIDs[primaryIndex], kind: "primary"}
		rows := primaryRows[primaryIndex]
		primaryIndex++
		f.blocks = append(f.blocks, block)
		nodeIndex := 0
		for _, child := range focusEntries(tokens, span.open+1, span.close) {
			if child.key != "focus" || child.open < 0 {
				continue
			}
			if nodeIndex >= len(rows) {
				return nil, fmt.Errorf("focus node span/import mismatch in %s", source)
			}
			addNode(block, child, rows[nodeIndex])
			nodeIndex++
		}
		if nodeIndex != len(rows) {
			return nil, fmt.Errorf("focus node span/import mismatch in %s", source)
		}
	}
	if primaryIndex != len(primaryRows) || poolIndex != len(poolRows) {
		return nil, fmt.Errorf("focus block span/import mismatch in %s", source)
	}
	return f, nil
}

func (f *FocusSourceFile) Rows() []map[string]any {
	var rows []map[string]any
	for _, block := range f.blocks {
		for _, node := range block.nodes {
			rows = append(rows, cloneJSON(node.row).(map[string]any))
		}
	}
	return rows
}

func (f *FocusSourceFile) Count(id string) int { return len(f.byID[id]) }

func ValidateFocusTarget(files map[string]*FocusSourceFile, id string) error {
	count := 0
	for _, file := range files {
		if file != nil {
			count += file.Count(id)
		}
	}
	if count > 1 {
		return fmt.Errorf("focus_node/%s is ambiguous: %d source definitions", id, count)
	}
	return nil
}

func ValidateFocusOverride(files map[string]*FocusSourceFile, id string, row map[string]any) error {
	if err := ValidateFocusTarget(files, id); err != nil {
		return err
	}
	if newID := stringField(row, "focus_id"); newID != "" && newID != id {
		for _, file := range files {
			if file != nil && file.Count(newID) > 0 {
				return fmt.Errorf("focus_node/%s would collide with focus ID %s", id, newID)
			}
		}
	}
	return nil
}

func ValidateFocusOverrides(files map[string]*FocusSourceFile, resolved map[string]map[string]any) error {
	var keys []string
	for key := range resolved {
		if strings.HasPrefix(key, "focus_node/") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	newIDs := map[string]string{}
	for _, key := range keys {
		id := strings.TrimPrefix(key, "focus_node/")
		row := resolved[key]
		if err := ValidateFocusOverride(files, id, row); err != nil {
			return err
		}
		if newID := stringField(row, "focus_id"); newID != "" {
			if prior := newIDs[newID]; prior != "" {
				return fmt.Errorf("%s and %s would both create focus ID %s", prior, key, newID)
			}
			newIDs[newID] = key
		}
	}
	return nil
}

func (f *FocusSourceFile) Render(overrides map[string]map[string]any) (string, error) {
	patched := map[*focusSourceNode]map[string]any{}
	touched := map[*focusSourceBlock]bool{}
	newIDs := map[string]string{}
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		kind, id, ok := strings.Cut(key, "/")
		if !ok || kind != "focus_node" || id == "" {
			return "", fmt.Errorf("invalid focus override %q", key)
		}
		nodes := f.byID[id]
		if len(nodes) != 1 {
			return "", fmt.Errorf("%s cannot be uniquely located in %s: %d definitions", key, f.Source, len(nodes))
		}
		row := overrides[key]
		if row == nil {
			return "", fmt.Errorf("%s requires a patched row, not a node deletion", key)
		}
		if newID := stringField(row, "focus_id"); newID != "" && newID != id && len(f.byID[newID]) != 0 {
			return "", fmt.Errorf("%s would collide with focus ID %s in %s", key, newID, f.Source)
		}
		if newID := stringField(row, "focus_id"); newID != "" {
			if prior := newIDs[newID]; prior != "" {
				return "", fmt.Errorf("%s and %s would both create focus ID %s", prior, key, newID)
			}
			newIDs[newID] = key
		}
		patched[nodes[0]] = row
		touched[nodes[0].block] = true
	}
	newline := "\n"
	if strings.Contains(f.text, "\r\n") {
		newline = "\r\n"
	}
	var replacements []focusReplacement
	for _, block := range f.blocks {
		if !touched[block] {
			continue
		}
		planNodes := make([]focusorder.Node, len(block.nodes))
		texts := make([]string, len(block.nodes))
		for i, node := range block.nodes {
			row := node.row
			text := f.text[node.span.start:node.span.end]
			if next, ok := patched[node]; ok {
				var err error
				text, err = focusPatchNode(text, row, next, newline)
				if err != nil {
					return "", fmt.Errorf("focus_node/%s: %w", stringField(row, "focus_id"), err)
				}
				row = next
			}
			texts[i] = text
			planNodes[i] = focusorder.Node{FocusID: stringField(row, "focus_id"), RelativePositionID: stringField(row, "relative_position_id")}
		}
		for slot, index := range focusorder.Plan(planNodes).Order {
			span := block.nodes[slot].span
			replacements = append(replacements, focusReplacement{start: span.start, end: span.end, text: texts[index]})
		}
	}
	return focusReplace(f.text, replacements), nil
}

type focusReplacement struct {
	start, end int
	text       string
}

func focusReplace(text string, replacements []focusReplacement) string {
	sort.Slice(replacements, func(i, j int) bool { return replacements[i].start < replacements[j].start })
	var out strings.Builder
	last := 0
	for _, replacement := range replacements {
		out.WriteString(text[last:replacement.start])
		out.WriteString(replacement.text)
		last = replacement.end
	}
	out.WriteString(text[last:])
	return out.String()
}

func focusPatchNode(raw string, before, after map[string]any, newline string) (string, error) {
	if reflect.DeepEqual(before, after) {
		return raw, nil
	}
	oldFields, err := focusRenderedFields(before)
	if err != nil {
		return "", err
	}
	newFields, err := focusRenderedFields(after)
	if err != nil {
		return "", err
	}

	aliases := map[string]string{"focus_id": "id", "prerequisites": "prerequisite", "allow_branch_ast": "allow_branch"}
	for field := range before {
		if _, present := after[field]; present {
			continue
		}
		key := field
		if mapped := aliases[field]; mapped != "" {
			key = mapped
		}
		delete(newFields, key)
	}
	changed := map[string]bool{}
	for key, value := range oldFields {
		if !reflect.DeepEqual(value, newFields[key]) {
			changed[key] = true
		}
	}
	for key, value := range newFields {
		if !reflect.DeepEqual(value, oldFields[key]) {
			changed[key] = true
		}
	}
	tokens, err := focusTokens(raw)
	if err != nil {
		return "", err
	}
	root := focusEntries(tokens, 0, len(tokens))
	if len(root) != 1 || root[0].open < 0 {
		return "", fmt.Errorf("expected one author focus definition")
	}
	format := func(fields []string) string { return strings.ReplaceAll(strings.Join(fields, "\n"), "\n", newline) }
	var replacements []focusReplacement
	seen := map[string]bool{}
	for _, span := range focusEntries(tokens, root[0].open+1, root[0].close) {
		if !changed[span.key] {
			continue
		}
		value := ""
		if !seen[span.key] {
			value = format(newFields[span.key])
			seen[span.key] = true
		}
		replacements = append(replacements, focusReplacement{span.start, span.end, value})
	}
	var additions []string
	for key := range changed {
		if !seen[key] {
			additions = append(additions, key)
		}
	}
	sort.Strings(additions)
	var extra strings.Builder
	for _, key := range additions {
		if len(newFields[key]) > 0 {
			extra.WriteString(newline + "\t\t" + format(newFields[key]) + newline)
		}
	}
	if extra.Len() > 0 {
		end := tokens[root[0].close].start
		replacements = append(replacements, focusReplacement{end, end, extra.String()})
	}
	return focusReplace(raw, replacements), nil
}

func focusRenderedFields(row map[string]any) (map[string][]string, error) {
	body, err := json.Marshal(map[string]any{"node": row})
	if err != nil {
		return nil, err
	}
	text, err := emitmisc.EmitFocusNodeJSON(body)
	if err != nil {
		return nil, err
	}
	tokens, err := focusTokens(text)
	if err != nil {
		return nil, err
	}
	root := focusEntries(tokens, 0, len(tokens))
	if len(root) != 1 || root[0].open < 0 {
		return nil, fmt.Errorf("emitter did not produce one focus definition")
	}
	fields := map[string][]string{}
	for _, span := range focusEntries(tokens, root[0].open+1, root[0].close) {
		fields[span.key] = append(fields[span.key], text[span.start:span.end])
	}
	return fields, nil
}

type focusToken struct {
	value      string
	start, end int
	pair       int
	kind       byte
}

func focusTokens(text string) ([]focusToken, error) {
	var tokens []focusToken
	var stack []int
	for i := 0; i < len(text); {
		if i == 0 && strings.HasPrefix(text, "\xef\xbb\xbf") {
			i += 3
			continue
		}
		c := text[i]
		if strings.ContainsRune(" \t\r\n", rune(c)) {
			i++
			continue
		}
		if c == '#' {
			for i < len(text) && text[i] != '\n' {
				i++
			}
			continue
		}
		start, kind := i, byte('w')
		switch c {
		case '{', '}':
			kind = c
			i++
		case '=', '>', '<', '!':
			kind = 'o'
			i++
			if i < len(text) && text[i] == '=' {
				i++
			}
		case '"':
			i++
			closed := false
			for i < len(text) {
				if text[i] == '\\' && i+1 < len(text) {
					i += 2
					continue
				}
				if text[i] == '"' {
					i++
					closed = true
					break
				}
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated focus string at byte %d", start)
			}
		default:
			for i < len(text) && !strings.ContainsRune(" \t\r\n{}=><!\"#", rune(text[i])) {
				i++
			}
		}
		value := text[start:i]
		if c == '"' {
			value = value[1 : len(value)-1]
		}
		index := len(tokens)
		tokens = append(tokens, focusToken{value: value, start: start, end: i, pair: -1, kind: kind})
		if kind == '{' {
			stack = append(stack, index)
		} else if kind == '}' {
			if len(stack) == 0 {
				return nil, fmt.Errorf("unmatched focus closing brace at byte %d", start)
			}
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			tokens[open].pair, tokens[index].pair = index, open
		}
	}
	if len(stack) != 0 {
		return nil, fmt.Errorf("unclosed focus block at byte %d", tokens[stack[len(stack)-1]].start)
	}
	return tokens, nil
}

func focusEntries(tokens []focusToken, start, end int) []focusSpan {
	var entries []focusSpan
	for i := start; i < end; {
		if i+2 < end && tokens[i].kind == 'w' && tokens[i+1].kind == 'o' {
			value := i + 2
			span := focusSpan{key: tokens[i].value, start: tokens[i].start, end: tokens[value].end, open: -1, close: -1}
			if tokens[value].kind == '{' {
				span.open, span.close = value, tokens[value].pair
				span.end = tokens[span.close].end
				i = span.close + 1
			} else {
				i = value + 1
			}
			entries = append(entries, span)
		} else if tokens[i].kind == '{' {
			i = tokens[i].pair + 1
		} else {
			i++
		}
	}
	return entries
}
