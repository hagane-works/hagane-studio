package exportcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"haganeworks/internal/hoi4/clausewitz"
	"strings"
)

const RawContentLiteralNewline = "raw_content.literal_escape_newline"

const rawContentNewlineRepair = "raw_content.replace_literal_newline"

const rawContentFindingLimit = 20

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

func (r RawContentReport) Warnings() []clausewitz.AnalysisFinding {
	out := []clausewitz.AnalysisFinding{}
	for _, f := range r.Findings {
		if f.Severity == "warning" {
			out = append(out, f)
		}
	}
	return out
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type RawScriptRawFile struct {
	ID       int64
	DestPath string
	Content  string
	IsBinary bool
	Source   string
}

const RawFileGateMaxChars = 2_000_000

func rawScriptFieldFindings(text, entityKind, entityID string, dbID any, field string, sourceName *string) []map[string]any {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	report := AnalyzeRawContent(text, "", "script")
	if sourceName != nil {
		report = analyzeRawScriptNamed(text, sourceName)
	}
	out := []map[string]any{}
	for _, f := range report.Findings {
		source := map[string]any{}
		for k, v := range f.Source {
			source[k] = v
		}
		source["field_path"] = []string{entityKind, entityID, field}
		if _, ok := source["kind"]; !ok {
			source["kind"] = entityKind
		}
		if _, ok := source["identifier"]; !ok {
			source["identifier"] = entityID
		}
		source["entity"] = map[string]any{
			"type": entityKind, "id": entityID, "db_id": dbID, "field": field,
		}
		evidence := map[string]any{}
		for k, v := range f.Evidence {
			evidence[k] = v
		}
		evidence["field"] = field
		evidence["brace_imbalance"] = RawContentBraceCodes[f.Code]
		out = append(out, map[string]any{
			"code":         f.Code,
			"severity":     f.Severity,
			"source":       source,
			"evidence":     evidence,
			"repair_route": f.RepairRoute,
		})
	}
	return out
}

func analyzeRawScriptNamed(text string, name *string) RawContentReport {
	sum := sha256.Sum256([]byte(text))
	report := RawContentReport{Kind: "script", SourceSHA256: hex.EncodeToString(sum[:])}
	findings, net := ScanBraceStructure(text, nil, name)
	report.Findings = append(findings, LiteralNewlineFindings(text, nil, name)...)
	report.NetDelta = net
	return report
}

func RawScriptFindings(events []Event, decisions []Decision, ideas []Idea, rawFiles []RawScriptRawFile) []map[string]any {
	findings := []map[string]any{}
	for _, ev := range events {
		eid := ev.EventID
		if eid == "" {
			eid = "(unnamed)"
		}
		for field, text := range map[string]string{
			"trigger": ev.Trigger, "mean_time_to_happen": ev.MeanTimeToHappen,
			"immediate": ev.Immediate, "after": ev.After,
		} {
			findings = append(findings, rawScriptFieldFindings(text, "event", eid, ev.ID, field, nil)...)
		}
		for index, raw := range ev.Options {
			option, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			for _, ofield := range []string{"trigger", "effects", "hidden_effects"} {
				text, _ := option[ofield].(string)
				findings = append(findings, rawScriptFieldFindings(text, "event", eid, ev.ID,
					fmt.Sprintf("options[%d].%s", index, ofield), nil)...)
			}
		}
	}
	for _, dec := range decisions {
		did := dec.DecisionID
		if did == "" {
			did = "(unnamed)"
		}
		var name *string
		for _, kv := range [][2]string{
			{"allowed", dec.Allowed}, {"available", dec.Available}, {"visible", dec.Visible},
			{"complete_effect", dec.CompleteEffect}, {"remove_effect", dec.RemoveEffect},
			{"cancel_trigger", dec.CancelTrigger}, {"cancel_effect", dec.CancelEffect},
			{"modifier", dec.Modifier}, {"activation", dec.Activation},
			{"timeout_effect", dec.TimeoutEffect}, {"target_root_trigger", dec.TargetRootTrigger},
			{"target_trigger", dec.TargetTrigger}, {"custom_cost_trigger", dec.CustomCostTrigger},
			{"ai_will_do", dec.AIWillDo},
		} {
			findings = append(findings, rawScriptFieldFindings(kv[1], "decision", did, dec.ID, kv[0], name)...)
		}
	}
	for _, idea := range ideas {
		iid := idea.IdeaID
		if iid == "" {
			iid = "(unnamed)"
		}
		for _, kv := range [][2]string{
			{"allowed", idea.Allowed}, {"allowed_civil_war", idea.AllowedCivilWar},
			{"available", idea.Available}, {"visible", idea.Visible},
			{"allowed_to_remove", idea.AllowedToRemove}, {"cancel", idea.Cancel},
			{"do_effect", idea.DoEffect}, {"on_add", idea.OnAdd}, {"on_remove", idea.OnRemove},
			{"ai_will_do", idea.AIWillDo},
		} {
			findings = append(findings, rawScriptFieldFindings(kv[1], "idea", iid, idea.ID, kv[0], nil)...)
		}
	}
	for _, raw := range rawFiles {
		if raw.IsBinary || (raw.Source != "" && raw.Source != "created") {
			continue
		}
		if ClassifyRawPath(raw.DestPath) != "script" {
			continue
		}
		if strings.TrimSpace(raw.Content) == "" || len(raw.Content) > RawFileGateMaxChars {
			continue
		}
		dest := raw.DestPath
		findings = append(findings, rawScriptFieldFindings(raw.Content, "raw_file", dest, raw.ID, "content", &dest)...)
	}
	return findings
}
