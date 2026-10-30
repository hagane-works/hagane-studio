package clausewitz

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/paradox"
)

type ValueKind string

const (
	ValueScalar  ValueKind = "scalar"
	ValueBlock   ValueKind = "block"
	ValueMissing ValueKind = "missing"
)

type SourceSpan struct {
	StartOffset int `json:"start_offset"`
	EndOffset   int `json:"end_offset"`
	StartLine   int `json:"start_line"`
	StartColumn int `json:"start_column"`
	EndLine     int `json:"end_line"`
	EndColumn   int `json:"end_column"`
}

type AnalysisFinding struct {
	Code        string         `json:"code"`
	Severity    string         `json:"severity"`
	Source      map[string]any `json:"source"`
	Evidence    map[string]any `json:"evidence"`
	RepairRoute string         `json:"repair_route"`
}

type ScalarValue struct {
	Kind string
	Str  string
	Int  int64
	Flt  float64
}

func (v ScalarValue) MarshalJSON() ([]byte, error) {
	switch v.Kind {
	case "str":
		return json.Marshal(v.Str)
	case "int":
		return []byte(strconv.FormatInt(v.Int, 10)), nil
	case "float":
		return []byte(pyFloatRepr(v.Flt)), nil
	}
	return []byte("null"), nil
}

func pyFloatRepr(f float64) string {
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	if math.IsNaN(f) {
		return "NaN"
	}
	sci := strconv.FormatFloat(f, 'e', -1, 64)
	exp := 0
	if i := strings.IndexByte(sci, 'e'); i >= 0 {
		exp, _ = strconv.Atoi(sci[i+1:])
	}
	if exp >= -4 && exp < 16 {
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if !strings.ContainsRune(s, '.') {
			s += ".0"
		}
		return s
	}
	return sci
}

type AssignmentIR struct {
	Key         string
	ValueKind   ValueKind
	ParentPath  []string
	Span        SourceSpan
	Operator    string
	Value       ScalarValue
	Excerpt     string
	KeyQuoted   bool
	ValueQuoted bool
}

func (a AssignmentIR) ToMap() map[string]any {
	path := a.ParentPath
	if path == nil {
		path = []string{}
	}
	return map[string]any{
		"key":          a.Key,
		"value_kind":   string(a.ValueKind),
		"parent_path":  path,
		"source_span":  a.Span,
		"operator":     a.Operator,
		"value":        a.Value,
		"excerpt":      a.Excerpt,
		"key_quoted":   a.KeyQuoted,
		"value_quoted": a.ValueQuoted,
	}
}

type Analysis struct {
	Assignments  []AssignmentIR
	Findings     []AnalysisFinding
	SourceSHA256 string
}

func (a Analysis) ToMap() map[string]any {
	as := make([]map[string]any, 0, len(a.Assignments))
	for _, x := range a.Assignments {
		as = append(as, x.ToMap())
	}
	fs := make([]AnalysisFinding, 0, len(a.Findings))
	fs = append(fs, a.Findings...)
	return map[string]any{
		"assignments":   as,
		"findings":      fs,
		"source_sha256": a.SourceSHA256,
	}
}

var qualifiedBlockWords = map[string]bool{"rgb": true, "hsv": true, "RGB": true, "HSV": true}

func pyIsSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return true
	}
	return unicode.IsSpace(r)
}

func lineStarts(text []rune) []int {
	starts := []int{0}
	for i, c := range text {
		if c == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func lineColumn(starts []int, offset int) (int, int) {
	idx := sort.SearchInts(starts, offset+1) - 1
	if idx < 0 {
		idx = 0
	}
	return idx + 1, offset - starts[idx] + 1
}

func Span(text []rune, start, end int) SourceSpan {
	return spanWith(lineStarts(text), len(text), start, end)
}

func spanWith(starts []int, n, start, end int) SourceSpan {
	safeStart := start
	if safeStart < 0 {
		safeStart = 0
	}
	if safeStart > n {
		safeStart = n
	}
	safeEnd := end
	if safeEnd > n {
		safeEnd = n
	}
	if safeEnd < safeStart {
		safeEnd = safeStart
	}
	sl, sc := lineColumn(starts, safeStart)
	el, ec := lineColumn(starts, safeEnd)
	return SourceSpan{
		StartOffset: safeStart, EndOffset: safeEnd,
		StartLine: sl, StartColumn: sc, EndLine: el, EndColumn: ec,
	}
}

func sourceRef(span SourceSpan, sourceName *string, fieldPath []string) map[string]any {
	path := fieldPath
	if path == nil {
		path = []string{}
	}
	var file any
	if sourceName != nil {
		file = *sourceName
	}
	return map[string]any{
		"file":       file,
		"field_path": path,
		"span":       span,
	}
}

func skipTrivia(text []rune, cursor int) int {
	n := len(text)
	for cursor < n {
		c := text[cursor]
		if c == '\uFEFF' || pyIsSpace(c) {
			cursor++
			continue
		}
		if c == '#' {
			nl := -1
			for i := cursor; i < n; i++ {
				if text[i] == '\n' {
					nl = i
					break
				}
			}
			if nl < 0 {
				cursor = n
			} else {
				cursor = nl + 1
			}
			continue
		}
		break
	}
	return cursor
}

func quotedEnd(text []rune, start int) (int, bool) {
	cursor := start + 1
	for cursor < len(text) {
		if text[cursor] == '\\' && cursor+1 < len(text) {
			cursor += 2
			continue
		}
		if text[cursor] == '"' {
			return cursor + 1, true
		}
		cursor++
	}
	return len(text), false
}

type positionedToken struct {
	tok        *paradox.Token
	start, end int
}

type blockFrame struct {
	key             string
	assignmentIndex int
	openStart       int
}

func locateTokens(text []rune, tokens []*paradox.Token, rootPath []string,
	sourceName *string) ([]positionedToken, []AnalysisFinding) {

	starts := lineStarts(text)
	n := len(text)
	var positioned []positionedToken
	var findings []AnalysisFinding
	cursor := 0

	for _, tok := range tokens {
		if tok.Type == "EOF" {
			positioned = append(positioned, positionedToken{tok, n, n})
			break
		}
		cursor = skipTrivia(text, cursor)
		start := cursor
		closedQuote := true
		var end int

		if tok.Quoted {
			if cursor >= n || text[cursor] != '"' {
				end = cursor
			} else {
				end, closedQuote = quotedEnd(text, cursor)
			}
		} else {
			expected := []rune(tok.Value)
			end = cursor + len(expected)
			if end > n || string(text[cursor:min(end, n)]) != tok.Value {

				sp := spanWith(starts, n, cursor, min(n, cursor+1))
				findings = append(findings, AnalysisFinding{
					Code: "clausewitz.analysis_token_alignment", Severity: "blocker",
					Source: sourceRef(sp, sourceName, rootPath),
					Evidence: map[string]any{
						"token_type":     tok.Type,
						"token_value":    tok.Value,
						"source_excerpt": string(text[cursor:min(n, cursor+32)]),
					},
					RepairRoute: "clausewitz.reparse_source",
				})
				end = min(n, cursor+max(1, len(expected)))
			}
		}

		positioned = append(positioned, positionedToken{tok, start, end})
		cursor = end

		if tok.Quoted && !closedQuote {
			sp := spanWith(starts, n, start, end)
			findings = append(findings, AnalysisFinding{
				Code: "clausewitz.unterminated_quote", Severity: "blocker",
				Source:      sourceRef(sp, sourceName, rootPath),
				Evidence:    map[string]any{"source_excerpt": string(text[start:end])},
				RepairRoute: "clausewitz.close_quote",
			})
		}
	}
	return positioned, findings
}

func astAssignmentCount(b *paradox.Block) int {
	if b == nil {
		return 0
	}
	count := 0
	for _, e := range b.Entries {
		if e.Key != nil {
			count++
		}
		if e.Block != nil {
			count += astAssignmentCount(e.Block)
		}
	}
	return count
}

func AnalyzeAssignments(src string, rootPath []string, sourceName *string) Analysis {
	sum := sha256.Sum256([]byte(src))
	text := []rune(src)
	n := len(text)
	starts := lineStarts(text)
	root := append([]string{}, rootPath...)

	tokens := paradox.Tokenize(text)
	positioned, findings := locateTokens(text, tokens, root, sourceName)

	assignments := []AssignmentIR{}
	var stack []blockFrame

	currentPath := func() []string {
		out := append([]string{}, root...)
		for _, f := range stack {
			out = append(out, f.key)
		}
		return out
	}
	appendAssignment := func(keyTok, opTok positionedToken, kind ValueKind,
		end int, value ScalarValue, valueQuoted bool) int {
		sp := spanWith(starts, n, keyTok.start, end)
		assignments = append(assignments, AssignmentIR{
			Key: keyTok.tok.Value, ValueKind: kind, ParentPath: currentPath(),
			Span: sp, Operator: opTok.tok.Value, Value: value,
			Excerpt:   string(text[sp.StartOffset:sp.EndOffset]),
			KeyQuoted: keyTok.tok.Quoted, ValueQuoted: valueQuoted,
		})
		return len(assignments) - 1
	}

	index := 0
	for index < len(positioned) {
		current := positioned[index]
		tok := current.tok
		if tok.Type == "EOF" {
			break
		}

		if (tok.Type == "STRING" || tok.Type == "NUMBER") && index+1 < len(positioned) {
			operator := positioned[index+1]
			if operator.tok.Type == "OP" {
				afterIndex := index + 2
				after := positioned[afterIndex]

				if after.tok.Type == "LBRACE" {
					ai := appendAssignment(current, operator, ValueBlock, after.end, ScalarValue{}, false)
					stack = append(stack, blockFrame{tok.Value, ai, after.start})
					index = afterIndex + 1
					continue
				}
				if after.tok.Type == "STRING" && qualifiedBlockWords[after.tok.Value] &&
					afterIndex+1 < len(positioned) && positioned[afterIndex+1].tok.Type == "LBRACE" {
					openBrace := positioned[afterIndex+1]
					ai := appendAssignment(current, operator, ValueBlock, openBrace.end,
						ScalarValue{Kind: "str", Str: after.tok.Value}, false)
					stack = append(stack, blockFrame{tok.Value, ai, openBrace.start})
					index = afterIndex + 2
					continue
				}
				if after.tok.Type == "RBRACE" || after.tok.Type == "EOF" {
					ai := appendAssignment(current, operator, ValueMissing, operator.end, ScalarValue{}, false)
					missing := assignments[ai]
					fp := append(append([]string{}, missing.ParentPath...), missing.Key)
					findings = append(findings, AnalysisFinding{
						Code: "clausewitz.missing_assignment_value", Severity: "blocker",
						Source:      sourceRef(missing.Span, sourceName, fp),
						Evidence:    map[string]any{"assignment": missing.Excerpt},
						RepairRoute: "clausewitz.fill_assignment_value",
					})
					index = afterIndex
					continue
				}

				var scalar ScalarValue
				if after.tok.Type == "NUMBER" {
					if strings.ContainsRune(after.tok.Value, '.') {
						f, _ := strconv.ParseFloat(after.tok.Value, 64)
						scalar = ScalarValue{Kind: "float", Flt: f}
					} else {
						iv, _ := strconv.ParseInt(after.tok.Value, 10, 64)
						scalar = ScalarValue{Kind: "int", Int: iv}
					}
				} else {
					scalar = ScalarValue{Kind: "str", Str: after.tok.Value}
				}
				appendAssignment(current, operator, ValueScalar, after.end, scalar, after.tok.Quoted)
				index = afterIndex + 1
				continue
			}
		}

		if tok.Type == "LBRACE" {
			stack = append(stack, blockFrame{"{anonymous}", -1, current.start})
			index++
			continue
		}
		if tok.Type == "RBRACE" {
			if len(stack) == 0 {
				sp := spanWith(starts, n, current.start, current.end)
				findings = append(findings, AnalysisFinding{
					Code: "clausewitz.stray_closing_brace", Severity: "blocker",
					Source:      sourceRef(sp, sourceName, currentPath()),
					Evidence:    map[string]any{"source_excerpt": string(text[current.start:current.end])},
					RepairRoute: "clausewitz.remove_or_match_closing_brace",
				})
			} else {
				frame := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if frame.assignmentIndex >= 0 {
					old := assignments[frame.assignmentIndex]
					closed := spanWith(starts, n, old.Span.StartOffset, current.end)
					old.Span = closed
					old.Excerpt = string(text[closed.StartOffset:closed.EndOffset])
					assignments[frame.assignmentIndex] = old
				}
			}
			index++
			continue
		}
		index++
	}

	for i := len(stack) - 1; i >= 0; i-- {
		frame := stack[i]
		sp := spanWith(starts, n, frame.openStart, n)
		fp := append([]string{}, root...)
		for _, item := range stack {
			fp = append(fp, item.key)
		}
		findings = append(findings, AnalysisFinding{
			Code: "clausewitz.unclosed_block", Severity: "blocker",
			Source:      sourceRef(sp, sourceName, fp),
			Evidence:    map[string]any{"source_excerpt": string(text[sp.StartOffset:sp.EndOffset])},
			RepairRoute: "clausewitz.close_block",
		})
		if frame.assignmentIndex >= 0 {
			old := assignments[frame.assignmentIndex]
			open := spanWith(starts, n, old.Span.StartOffset, n)
			old.Span = open
			old.Excerpt = string(text[open.StartOffset:open.EndOffset])
			assignments[frame.assignmentIndex] = old
		}
	}

	ast, recoveries, err := paradox.Parse(src)
	parseFailed := err != nil
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	if failed, m := ParseErrorMessage(recoveries, err); failed {
		parseFailed = true
		msg = m
	}
	if parseFailed {
		sp := spanWith(starts, n, 0, n)
		findings = append(findings, AnalysisFinding{
			Code: "clausewitz.parse_error", Severity: "blocker",
			Source:      sourceRef(sp, sourceName, root),
			Evidence:    map[string]any{"error": msg},
			RepairRoute: "clausewitz.fix_syntax",
		})
	} else {
		astCount := astAssignmentCount(ast)
		positionedCount := 0
		for _, a := range assignments {
			if a.ValueKind != ValueMissing {
				positionedCount++
			}
		}
		hasBlocker := false
		for _, f := range findings {
			if strings.HasPrefix(f.Code, "clausewitz.") && f.Severity == "blocker" {
				hasBlocker = true
				break
			}
		}
		if !hasBlocker && astCount != positionedCount {
			sp := spanWith(starts, n, 0, n)
			findings = append(findings, AnalysisFinding{
				Code: "clausewitz.analysis_ast_mismatch", Severity: "blocker",
				Source: sourceRef(sp, sourceName, root),
				Evidence: map[string]any{
					"parser_assignment_count":     astCount,
					"positioned_assignment_count": positionedCount,
				},
				RepairRoute: "clausewitz.reparse_source",
			})
		}
	}

	return Analysis{
		Assignments:  assignments,
		Findings:     findings,
		SourceSHA256: hex.EncodeToString(sum[:]),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ParseErrorMessage(recoveries []paradox.Recovery, err error) (bool, string) {
	if err != nil {
		return true, err.Error()
	}
	for _, r := range recoveries {
		if r.Code != paradox.RecoveryUnclosedBlock {
			continue
		}
		line := 0
		switch v := r.Details["eof_line"].(type) {
		case int:
			line = v
		case int64:
			line = int(v)
		case float64:
			line = int(v)
		}
		return true, "Expected RBRACE, got EOF '' at line " + strconv.Itoa(line)
	}
	return false, ""
}

func PyIsSpace(r rune) bool { return pyIsSpace(r) }

func (v ScalarValue) PyStr() string {
	switch v.Kind {
	case "str":
		return v.Str
	case "int":
		return strconv.FormatInt(v.Int, 10)
	case "float":
		return pyFloatRepr(v.Flt)
	}
	return ""
}
