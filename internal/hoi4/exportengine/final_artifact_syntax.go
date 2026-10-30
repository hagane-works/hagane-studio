package exportengine

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/loc"
	"haganeworks/internal/hoi4/paradox"
)

const FinalArtifactSyntaxCoverage = "final_artifact_syntax"

type FinalArtifactSyntaxFinding struct {
	Code          string `json:"code"`
	Path          string `json:"path"`
	Line          int    `json:"line"`
	Snippet       string `json:"snippet"`
	Severity      string `json:"severity"`
	PlayerImpact  string `json:"player_impact"`
	SuggestedEdit string `json:"suggested_edit"`
	Detail        string `json:"detail,omitempty"`
}

type FinalArtifactSyntaxReport struct {
	Scanned  int                          `json:"scanned"`
	Findings []FinalArtifactSyntaxFinding `json:"findings"`
	msg      exportmsg.Printer
}

func (r *FinalArtifactSyntaxReport) Count() int {
	if r == nil {
		return 0
	}
	return len(r.Findings)
}

func (r *FinalArtifactSyntaxReport) WarningLines() []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.Findings))
	for _, finding := range r.Findings {
		key := "syntax.warning.attention"
		if finding.Severity == "review" {
			key = "syntax.warning.review"
		}
		out = append(out, r.msg.Text(key, exportmsg.Args{
			"path": finding.Path, "line": finding.Line, "code": finding.Code, "detail": finding.Detail,
			"snippet": finding.Snippet, "impact": finding.PlayerImpact, "edit": finding.SuggestedEdit,
		}))
	}
	return out
}

func ApplyFinalArtifactSyntax(artifact *NativeArtifact, msg exportmsg.Printer) *FinalArtifactSyntaxReport {
	if artifact == nil {
		return &FinalArtifactSyntaxReport{msg: msg}
	}
	report := ScanFinalArtifactSyntax(artifact.Entries, msg)
	artifact.Warnings = append(artifact.Warnings, report.WarningLines()...)
	if report.Scanned > 0 {
		artifact.Coverage = appendCoverageOnce(artifact.Coverage, FinalArtifactSyntaxCoverage)
	}
	return report
}

func appendCoverageOnce(coverage []string, value string) []string {
	for _, existing := range coverage {
		if existing == value {
			return coverage
		}
	}
	return append(coverage, value)
}

func ScanFinalArtifactSyntax(entries []Entry, msg exportmsg.Printer) *FinalArtifactSyntaxReport {
	report := &FinalArtifactSyntaxReport{msg: msg}
	ordered := append([]Entry(nil), entries...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return strings.ToLower(ordered[i].Path) < strings.ToLower(ordered[j].Path)
	})
	for _, entry := range ordered {
		relative, kind, ok := finalSyntaxDestination(entry.Path)
		if !ok {
			continue
		}
		report.Scanned++
		if len(entry.Bytes) == 0 {
			continue
		}
		var findings []FinalArtifactSyntaxFinding
		if kind == "localisation" {
			findings = scanFinalLocalisation(relative, entry, msg)
		} else {
			findings = scanFinalClausewitz(relative, entry, msg)
		}
		report.Findings = append(report.Findings, findings...)
	}
	sort.SliceStable(report.Findings, func(i, j int) bool {
		a, b := report.Findings[i], report.Findings[j]
		if strings.ToLower(a.Path) != strings.ToLower(b.Path) {
			return strings.ToLower(a.Path) < strings.ToLower(b.Path)
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Code < b.Code
	})
	return report
}

func finalSyntaxDestination(artifactPath string) (relative, kind string, ok bool) {
	clean := strings.Trim(strings.ReplaceAll(artifactPath, "\\", "/"), "/")
	parts := strings.Split(clean, "/")
	for i, part := range parts {
		lowerPart := strings.ToLower(part)
		if lowerPart != "common" && lowerPart != "events" && lowerPart != "history" && lowerPart != "localisation" {
			continue
		}
		relative = strings.Join(parts[i:], "/")
		lower := strings.ToLower(relative)
		if lowerPart == "localisation" {
			return relative, "localisation", strings.HasSuffix(lower, ".yml")
		}
		return relative, "script", strings.HasSuffix(lower, ".txt")
	}
	return "", "", false
}

func scanFinalClausewitz(relative string, entry Entry, msg exportmsg.Printer) []FinalArtifactSyntaxFinding {
	text := strings.TrimPrefix(string(entry.Bytes), "\uFEFF")
	block, recoveries, parseErr := paradox.Parse(text)
	findings := make([]FinalArtifactSyntaxFinding, 0, len(recoveries)+2)
	seen := map[string]bool{}
	add := func(code string, line int, severity, detail string) {
		if line < 1 {
			line = 1
		}
		key := fmt.Sprintf("%s\x00%d", code, line)
		if seen[key] {
			return
		}
		seen[key] = true
		findings = append(findings, syntaxFinding(relative, entry, code, line, severity, detail, text, false, msg))
	}
	if parseErr != nil {
		add("clausewitz.parse_failed", parseErrorLine(parseErr.Error()), "attention", parseErr.Error())
	}
	for _, recovery := range recoveries {
		severity := "attention"

		if recovery.Code == paradox.RecoveryUnclosedBlock {
			severity = "review"
		}
		add("clausewitz."+strings.TrimPrefix(recovery.Code, "parser_recovery_"), recovery.Line,
			severity, recoveryDetail(recovery))
	}
	for _, token := range paradox.Tokenize([]rune(text)) {
		if token.Type == "UNKNOWN" {
			add("clausewitz.unknown_token", token.Line, "attention",
				msg.Text("syntax.detail.unknown_token", exportmsg.Args{"token": token.Value}))
		}
	}
	quoteLine, literalLines := clausewitzLexicalRisks(text)
	if quoteLine > 0 {
		add("clausewitz.unterminated_quote", quoteLine, "attention", msg.Text("syntax.detail.unterminated_quote", nil))
	}
	for _, line := range literalLines {
		add("clausewitz.literal_backslash_n", line, "attention", msg.Text("syntax.detail.literal_newline", nil))
	}
	if block != nil {
		for _, parsed := range block.Entries {
			if parsed.Key != nil {
				continue
			}
			if parsed.Block != nil {
				add("clausewitz.top_level_anonymous_block", parsed.Line, "review",
					msg.Text("syntax.detail.anonymous_block", nil))
			} else {
				add("clausewitz.top_level_bare_token", parsed.Line, "review",
					msg.Text("syntax.detail.bare_token", exportmsg.Args{"token": parsed.Scalar.StrValue()}))
			}
		}
	}
	return findings
}

func recoveryDetail(recovery paradox.Recovery) string {
	detail := strings.ReplaceAll(recovery.Code, "parser_recovery_", "")
	if len(recovery.Details) == 0 {
		return detail
	}
	keys := make([]string, 0, len(recovery.Details))
	for key := range recovery.Details {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, recovery.Details[key]))
	}
	return detail + ": " + strings.Join(parts, ", ")
}

func parseErrorLine(message string) int {
	const marker = " at line "
	idx := strings.LastIndex(message, marker)
	if idx < 0 {
		return 1
	}
	var line int
	if _, err := fmt.Sscanf(message[idx+len(marker):], "%d", &line); err != nil || line < 1 {
		return 1
	}
	return line
}

func clausewitzLexicalRisks(text string) (unterminatedQuoteLine int, literalBackslashNLines []int) {
	runes := []rune(text)
	line := 1
	inQuote := false
	inComment := false
	quoteStart := 0
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if c == '\n' {
			line++
			inComment = false
			continue
		}
		if inComment {
			continue
		}
		if !inQuote && c == '#' {
			inComment = true
			continue
		}
		if c == '\\' {
			if !inQuote && i+1 < len(runes) && runes[i+1] == 'n' {
				literalBackslashNLines = append(literalBackslashNLines, line)
			}
			if inQuote && i+1 < len(runes) {
				i++
			}
			continue
		}
		if c == '"' {
			if inQuote {
				inQuote = false
				quoteStart = 0
			} else {
				inQuote = true
				quoteStart = line
			}
		}
	}
	if inQuote {
		unterminatedQuoteLine = quoteStart
	}
	return unterminatedQuoteLine, uniqueInts(literalBackslashNLines)
}

func uniqueInts(values []int) []int {
	seen := map[int]bool{}
	out := make([]int, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func scanFinalLocalisation(relative string, entry Entry, msg exportmsg.Printer) []FinalArtifactSyntaxFinding {
	raw := string(entry.Bytes)
	text := strings.TrimPrefix(raw, "\uFEFF")
	findings := []FinalArtifactSyntaxFinding{}
	seen := map[string]bool{}
	add := func(code string, line int, severity, detail string) {
		if line < 1 {
			line = 1
		}
		key := fmt.Sprintf("%s\x00%d", code, line)
		if seen[key] {
			return
		}
		seen[key] = true
		findings = append(findings, syntaxFinding(relative, entry, code, line, severity, detail, text, true, msg))
	}
	if !strings.HasPrefix(raw, "\uFEFF") {
		add("localisation.missing_utf8_bom", 1, "review",
			msg.Text("syntax.detail.missing_bom", nil))
	}
	lines := splitSyntaxLines(text)
	headerSeen := false
	contentFindingCount := 0
	for i, line := range lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if language, ok := localisationHeader(line); ok {
			if headerSeen {
				add("localisation.multiple_language_headers", lineNumber, "attention",
					msg.Text("syntax.detail.second_header", nil))
				contentFindingCount++
			}
			headerSeen = true
			if _, supported := loc.LanguageToSuffix(language); !supported {
				add("localisation.unknown_language_header", lineNumber, "attention",
					msg.Text("syntax.detail.unknown_language", exportmsg.Args{"language": language}))
				contentFindingCount++
			}
			continue
		}
		if !headerSeen {
			add("localisation.content_before_language_header", lineNumber, "attention",
				msg.Text("syntax.detail.before_header", nil))
			contentFindingCount++
			continue
		}
		if code, detail := validateLocalisationEntryLine(line, msg); code != "" {
			add(code, lineNumber, "attention", detail)
			contentFindingCount++
		}
	}
	if !headerSeen {
		add("localisation.missing_language_header", 1, "attention", msg.Text("syntax.detail.no_header", nil))
		contentFindingCount++
	}

	_, _, diagnostics := loc.ParseLocText(raw)
	if len(diagnostics) > 0 && contentFindingCount == 0 {
		add("localisation.parser_diagnostic", 1, "attention", strings.Join(diagnostics, "; "))
	}
	return findings
}

func localisationHeader(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if comment := strings.Index(trimmed, "#"); comment >= 0 {
		trimmed = strings.TrimSpace(trimmed[:comment])
	}
	if !strings.HasSuffix(trimmed, ":") {
		return "", false
	}
	language := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
	if !strings.HasPrefix(language, "l_") || len(language) == 2 {
		return "", false
	}
	for _, r := range language[2:] {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return "", false
		}
	}
	return language, true
}

func validateLocalisationEntryLine(line string, msg exportmsg.Printer) (code, detail string) {
	runes := []rune(line)
	colon := -1
	for i, r := range runes {
		if r == ':' {
			colon = i
			break
		}
	}
	if colon < 0 {
		return "localisation.invalid_entry", msg.Text("syntax.detail.invalid_entry", nil)
	}
	key := strings.TrimSpace(string(runes[:colon]))
	if key == "" {
		return "localisation.invalid_entry", msg.Text("syntax.detail.invalid_entry", nil)
	}
	for _, r := range key {
		if !isLocalisationKeyRune(r) {
			return "localisation.invalid_key", msg.Text("syntax.detail.invalid_key", exportmsg.Args{"key": key})
		}
	}
	i := colon + 1
	for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
		i++
	}
	for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
		i++
	}
	for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
		i++
	}
	if i >= len(runes) || runes[i] != '"' {
		return "localisation.missing_open_quote", msg.Text("syntax.detail.missing_open_quote", nil)
	}
	i++
	closed := false
	for i < len(runes) {
		if runes[i] == '\\' {
			if i+1 < len(runes) {
				i += 2
				continue
			}
			break
		}
		if runes[i] == '"' {
			closed = true
			i++
			break
		}
		i++
	}
	if !closed {
		return "localisation.unterminated_value", msg.Text("syntax.detail.unterminated_value", nil)
	}
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}
	if i < len(runes) && runes[i] != '#' {
		return "localisation.trailing_junk", msg.Text("syntax.detail.trailing_junk", nil)
	}
	return "", ""
}

func isLocalisationKeyRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-'
}

func syntaxFinding(relative string, entry Entry, code string, line int, severity, detail, text string, localisation bool, msg exportmsg.Printer) FinalArtifactSyntaxFinding {
	impactKey := "syntax.impact.script"
	if localisation {
		impactKey = "syntax.impact.localisation"
	}
	if code == "localisation.missing_utf8_bom" {
		impactKey = "syntax.impact.missing_bom"
	}
	return FinalArtifactSyntaxFinding{
		Code: code, Path: relative, Line: line, Snippet: nearbySyntaxSnippet(text, line),
		Severity: severity, PlayerImpact: msg.Text(impactKey, nil),
		SuggestedEdit: msg.Text(finalSyntaxSuggestedEdit(relative, entry), exportmsg.Args{"path": relative}), Detail: detail,
	}
}

func finalSyntaxSuggestedEdit(relative string, entry Entry) string {
	lower := strings.ToLower(relative)
	if strings.HasPrefix(entry.Origin, "raw_") || strings.HasPrefix(entry.SourceEntity, "project_raw_files:") {
		return "syntax.edit.raw"
	}
	switch {
	case strings.HasPrefix(lower, "events/"):
		return "syntax.edit.events"
	case strings.HasPrefix(lower, "common/national_focus/"):
		return "syntax.edit.focus"
	case strings.HasPrefix(lower, "common/ideas/"):
		return "syntax.edit.ideas"
	case strings.HasPrefix(lower, "common/characters/"):
		return "syntax.edit.characters"
	case strings.HasPrefix(lower, "history/"):
		return "syntax.edit.history"
	case strings.HasPrefix(lower, "localisation/"):
		return "syntax.edit.localisation"
	default:
		return "syntax.edit.raw"
	}
}

func nearbySyntaxSnippet(text string, line int) string {
	lines := splitSyntaxLines(text)
	if len(lines) == 0 {
		return "<empty>"
	}
	if line < 1 {
		line = 1
	}
	if line > len(lines) {
		line = len(lines)
	}
	start := line - 2
	if start < 0 {
		start = 0
	}
	end := line + 1
	if end > len(lines) {
		end = len(lines)
	}
	parts := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		value := strings.TrimSpace(lines[i])
		if value == "" {
			value = "<blank>"
		}
		parts = append(parts, fmt.Sprintf("L%d %s", i+1, value))
	}
	snippet := strings.Join(parts, " | ")
	const limit = 260
	if len([]rune(snippet)) > limit {
		runes := []rune(snippet)
		snippet = string(runes[:limit-1]) + "…"
	}
	return snippet
}

func splitSyntaxLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(text, "\n")
}
