package exportcheck

import (
	"fmt"
	"haganeworks/internal/hoi4/clausewitz"
	"strings"
)

const RawContentBraceCode = "raw_content_brace_imbalance"

const RawContentBraceRepairRoute = "raw_content.fix_brace_balance"

func (r RawContentReport) BraceBlockers() []clausewitz.AnalysisFinding {
	out := []clausewitz.AnalysisFinding{}
	for _, f := range r.Findings {
		if RawContentBraceCodes[f.Code] {
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
