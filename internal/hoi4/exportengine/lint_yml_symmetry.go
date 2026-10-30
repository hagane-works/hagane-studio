package exportengine

import (
	"strings"

	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportmsg"
)

const YMLSymmetryLogName = "_YML_SYMMETRIC_WARNINGS.log"

type YMLSymmetryReport struct {
	Lines []exportmsg.Message
	msg   exportmsg.Printer
}

func ScanYMLSymmetryWarnings(meta worldout.Meta, msg exportmsg.Printer) *YMLSymmetryReport {
	report := &YMLSymmetryReport{msg: msg}
	for _, line := range meta.Symmetry {
		if line.Key != "" {
			report.Lines = append(report.Lines, line)
		}
	}
	return report
}

func (r *YMLSymmetryReport) Count() int {
	if r == nil {
		return 0
	}
	return len(r.Lines)
}

func (r *YMLSymmetryReport) Headline() string {
	if r.Count() == 0 {
		return ""
	}
	return r.msg.Text("yml.headline", exportmsg.Args{
		"count": r.Count(), "findings": strings.Join(worldLintRender(r.Lines, r.msg), "; "), "log": YMLSymmetryLogName,
	})
}

func (r *YMLSymmetryReport) LogEntry(modDir string) Entry {
	if r.Count() == 0 {
		return Entry{}
	}
	text := worldLintLogText(commentLines(r.msg.Text("yml.log.header", nil)), worldLintRender(r.Lines, r.msg))
	return worldLintEntry(modDir, YMLSymmetryLogName, text, "yml_symmetry_lint")
}

func (r *YMLSymmetryReport) CoverageTag() string { return "yml_symmetry_lint" }
