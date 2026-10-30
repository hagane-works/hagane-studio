package rawfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"haganeworks/internal/hoi4/clausewitz"
)

const (
	RawContentBraceCode        = "raw_content_brace_imbalance"
	RawContentBraceRepairRoute = "raw_content.fix_brace_balance"
	RawContentLiteralNewline   = "raw_content.literal_escape_newline"
	rawContentNewlineRepair    = "raw_content.replace_literal_newline"
	rawContentFindingLimit     = 20
)

var RawContentBraceCodes = map[string]bool{
	"clausewitz.stray_closing_brace": true,
	"clausewitz.unclosed_block":      true,
	"clausewitz.unterminated_quote":  true,
}

func ClassifyRawPath(destPath string) string {
	p := strings.ToLower(strings.TrimSpace(destPath))
	if strings.HasSuffix(p, ".yml") || strings.HasSuffix(p, ".yaml") {
		return "localisation"
	}
	if strings.HasSuffix(p, ".txt") || strings.HasSuffix(p, ".gui") || strings.HasSuffix(p, ".gfx") {
		return "script"
	}
	return "other"
}

type RawContentReport struct {
	Kind         string                       `json:"kind"`
	SourceSHA256 string                       `json:"source_sha256"`
	NetDelta     int                          `json:"net_delta"`
	Findings     []clausewitz.AnalysisFinding `json:"findings"`
}

func (r RawContentReport) BraceBlockers() []clausewitz.AnalysisFinding {
	out := []clausewitz.AnalysisFinding{}
	for _, f := range r.Findings {
		if RawContentBraceCodes[f.Code] {
			out = append(out, f)
		}
	}
	return out
}

func (r RawContentReport) Warnings() []clausewitz.AnalysisFinding {
	out := []clausewitz.AnalysisFinding{}
	for _, f := range r.Findings {
		if f.Severity == "warning" {
			out = append(out, f)
		}
	}
	return out
}

func (r RawContentReport) IsValid() bool {
	for _, f := range r.Findings {
		if f.Severity == "blocker" {
			return false
		}
	}
	return true
}

type RawContentError struct {
	Code        string                       `json:"code"`
	Message     string                       `json:"message"`
	TargetPath  string                       `json:"target_path"`
	NetDelta    int                          `json:"net_delta"`
	Findings    []clausewitz.AnalysisFinding `json:"findings"`
	RepairRoute string                       `json:"repair_route"`
}

func (e *RawContentError) Error() string { return e.Message }

func (e *RawContentError) AsTombstone() *TombstoneError {
	findings := make([]map[string]any, 0, len(e.Findings))
	for _, f := range e.Findings {
		findings = append(findings, map[string]any{
			"code":         f.Code,
			"severity":     f.Severity,
			"source":       f.Source,
			"evidence":     f.Evidence,
			"repair_route": f.RepairRoute,
		})
	}
	return &TombstoneError{
		Code:        e.Code,
		Message:     e.Message,
		Findings:    findings,
		NetDelta:    e.NetDelta,
		RepairRoute: e.RepairRoute,
	}
}

func (e *RawContentError) Detail(rawKind string) map[string]any {
	return map[string]any{
		"code":         e.Code,
		"message":      e.Message,
		"raw_kind":     rawKind,
		"target_path":  e.TargetPath,
		"net_delta":    e.NetDelta,
		"findings":     e.Findings,
		"repair_route": e.RepairRoute,
	}
}

func lineExcerpt(text []rune, offset int, width int) string {
	lineStart := 0
	for i := offset - 1; i >= 0; i-- {
		if text[i] == '\n' {
			lineStart = i + 1
			break
		}
	}
	lineEnd := len(text)
	for i := offset; i < len(text); i++ {
		if text[i] == '\n' {
			lineEnd = i
			break
		}
	}
	line := text[lineStart:lineEnd]
	if len(line) <= width {
		return string(line)
	}
	rel := offset - lineStart
	start := rel - width/2
	if start > len(line)-width {
		start = len(line) - width
	}
	if start < 0 {
		start = 0
	}
	prefix := ""
	if start > 0 {
		prefix = "…"
	}
	return prefix + string(line[start:start+width])
}

func rawFinding(text []rune, code, severity string, start, end int, rootPath []string,
	sourceName *string, evidence map[string]any, repairRoute string) clausewitz.AnalysisFinding {
	span := clausewitz.Span(text, start, end)
	fp := []string{}
	fp = append(fp, rootPath...)
	var file any
	if sourceName != nil {
		file = *sourceName
	}
	ev := map[string]any{
		"line":           span.StartLine,
		"source_excerpt": lineExcerpt(text, start, 200),
	}
	for k, v := range evidence {
		ev[k] = v
	}
	return clausewitz.AnalysisFinding{
		Code:     code,
		Severity: severity,
		Source: map[string]any{
			"file":       file,
			"field_path": fp,
			"span":       span,
		},
		Evidence:    ev,
		RepairRoute: repairRoute,
	}
}

func ScanBraceStructure(src string, rootPath []string, sourceName *string) ([]clausewitz.AnalysisFinding, int) {
	text := []rune(src)
	n := len(text)
	findings := []clausewitz.AnalysisFinding{}
	openStack := []int{}
	commented := []int{}
	stray := 0
	inComment := false
	quoteStart := -1
	for i := 0; i < n; i++ {
		c := text[i]
		switch {
		case inComment:
			if c == '\n' {
				inComment = false
			} else if (c == '{' || c == '}') && len(commented) < 64 {
				commented = append(commented, i)
			}
		case quoteStart >= 0:
			if c == '\\' && i+1 < n {
				i++
			} else if c == '"' {
				quoteStart = -1
			}
		case c == '#':
			inComment = true
		case c == '"':
			quoteStart = i
		case c == '{':
			openStack = append(openStack, i)
		case c == '}':
			if len(openStack) > 0 {
				openStack = openStack[:len(openStack)-1]
			} else {
				stray++
				if len(findings) < rawContentFindingLimit {
					findings = append(findings, rawFinding(text, "clausewitz.stray_closing_brace", "blocker",
						i, i+1, rootPath, sourceName, nil, "clausewitz.remove_or_match_closing_brace"))
				}
			}
		}
	}
	if quoteStart >= 0 && len(findings) < rawContentFindingLimit {
		findings = append(findings, rawFinding(text, "clausewitz.unterminated_quote", "blocker",
			quoteStart, minInt(n, quoteStart+1), rootPath, sourceName, nil, "clausewitz.close_quote"))
	}
	for k := len(openStack) - 1; k >= 0; k-- {
		if len(findings) >= rawContentFindingLimit {
			break
		}
		openAt := openStack[k]
		suspects := []map[string]any{}
		seen := map[int]bool{}
		for _, off := range commented {
			if off <= openAt || len(suspects) >= 3 {
				continue
			}
			ln := clausewitz.Span(text, off, off+1).StartLine
			if seen[ln] {
				continue
			}
			seen[ln] = true
			suspects = append(suspects, map[string]any{
				"line":           ln,
				"source_excerpt": lineExcerpt(text, off, 200),
			})
		}
		findings = append(findings, rawFinding(text, "clausewitz.unclosed_block", "blocker",
			openAt, minInt(n, openAt+1), rootPath, sourceName, map[string]any{
				"hint": "If a line inside this block ends with an inline '# comment', every '}' " +
					"after the '#' is part of the comment and closes nothing - move the comment after the braces.",
				"commented_brace_lines": suspects,
			}, "clausewitz.close_block"))
	}
	return findings, len(openStack) - stray
}

func LiteralNewlineFindings(src string, rootPath []string, sourceName *string) []clausewitz.AnalysisFinding {
	text := []rune(src)
	n := len(text)
	findings := []clausewitz.AnalysisFinding{}
	inComment := false
	for i := 0; i < n-1; i++ {
		c := text[i]
		switch {
		case c == '\n':
			inComment = false
		case c == '#' && !inComment:
			inComment = true
		case !inComment && c == '\\' && text[i+1] == 'n':
			findings = append(findings, rawFinding(text, RawContentLiteralNewline, "warning",
				i, i+2, rootPath, sourceName, map[string]any{
					"reason": "HOI4 script does not interpret backslash escapes; the two characters '\\n' " +
						"are emitted verbatim. Use a real line break, or move the text into a localisation " +
						"yml value where \\n is a newline.",
				}, rawContentNewlineRepair))
			if len(findings) >= rawContentFindingLimit {
				return findings
			}
			i++
		}
	}
	return findings
}

func AnalyzeRawContent(src, destPath, kind string) RawContentReport {
	if kind == "" {
		kind = ClassifyRawPath(destPath)
	}
	sum := sha256.Sum256([]byte(src))
	report := RawContentReport{Kind: kind, SourceSHA256: hex.EncodeToString(sum[:]),
		Findings: []clausewitz.AnalysisFinding{}}
	if kind != "script" {
		return report
	}
	var name *string
	if destPath != "" {
		name = &destPath
	}
	findings, net := ScanBraceStructure(src, nil, name)
	report.Findings = append(report.Findings, findings...)
	report.Findings = append(report.Findings, LiteralNewlineFindings(src, nil, name)...)
	report.NetDelta = net
	return report
}

func SummarizeRawFindings(findings []clausewitz.AnalysisFinding, limit int) string {
	parts := []string{}
	for _, f := range findings {
		where := "?"
		if span, ok := f.Source["span"].(clausewitz.SourceSpan); ok {
			where = fmt.Sprintf("line %d:%d", span.StartLine, span.StartColumn)
		}
		if raw, ok := f.Evidence["commented_brace_lines"].([]map[string]any); ok && len(raw) > 0 {
			lines := []string{}
			for i, s := range raw {
				if i >= 3 {
					break
				}
				lines = append(lines, fmt.Sprint(s["line"]))
			}
			where += " (braces inside a '#' comment at line " + strings.Join(lines, ", ") + ")"
		}
		parts = append(parts, f.Code+" @ "+where)
		if len(parts) >= limit {
			break
		}
	}
	return strings.Join(parts, "; ")
}

func GateRawContent(destPath, content string, isBinary bool) ([]clausewitz.AnalysisFinding, *RawContentError) {
	if isBinary || content == "" {
		return []clausewitz.AnalysisFinding{}, nil
	}
	report := AnalyzeRawContent(content, destPath, "")
	brace := report.BraceBlockers()
	if len(brace) > 0 {
		return nil, &RawContentError{
			Code: RawContentBraceCode,
			Message: fmt.Sprintf("raw file content does not close its braces (net %+d); HOI4 would nest "+
				"every later block of this file into the wrong scope: %s",
				report.NetDelta, SummarizeRawFindings(brace, 5)),
			TargetPath:  destPath,
			NetDelta:    report.NetDelta,
			Findings:    brace,
			RepairRoute: RawContentBraceRepairRoute,
		}
	}
	return report.Warnings(), nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
