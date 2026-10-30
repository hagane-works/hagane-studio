package exportengine

import (
	"strings"

	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportmsg"
)

const PoliticsLogName = "_POLITICS_EXPORT_WARNINGS.log"

const politicsNoteKey = "politics.note."

type PoliticsReport struct {
	Lines []exportmsg.Message
	msg   exportmsg.Printer
}

func ScanPoliticsWarnings(meta worldout.Meta, msg exportmsg.Printer) *PoliticsReport {
	r := &PoliticsReport{msg: msg}
	for _, line := range meta.Politics {
		if line.Key == "" {
			continue
		}
		r.Lines = append(r.Lines, line)
	}
	return r
}

func (r *PoliticsReport) Count() int {
	if r == nil {
		return 0
	}
	return len(r.Lines)
}

func (r *PoliticsReport) Warnings() []exportmsg.Message {
	if r == nil {
		return nil
	}
	var out []exportmsg.Message
	for _, line := range r.Lines {
		if !strings.HasPrefix(line.Key, politicsNoteKey) {
			out = append(out, line)
		}
	}
	return out
}

func (r *PoliticsReport) Notes() []exportmsg.Message {
	if r == nil {
		return nil
	}
	var out []exportmsg.Message
	for _, line := range r.Lines {
		if strings.HasPrefix(line.Key, politicsNoteKey) {
			out = append(out, line)
		}
	}
	return out
}

func (r *PoliticsReport) Headline() string {
	if r.Count() == 0 {
		return ""
	}
	warnings := r.Warnings()
	notes := r.Notes()
	if len(warnings) == 0 {
		return r.msg.Text("politics.headline_notes", exportmsg.Args{"count": len(notes), "log": PoliticsLogName})
	}
	args := exportmsg.Args{"count": len(warnings), "log": PoliticsLogName}
	headline := r.msg.Text("politics.headline_unnamed", args)
	if subjects := worldLintSubjects(warnings); len(subjects) > 0 {
		args["subjects"] = joinCapped(subjects, 5, r.msg)
		headline = r.msg.Text("politics.headline", args)
	}
	if len(notes) > 0 {
		headline += " " + r.msg.Text("politics.headline_also_notes", exportmsg.Args{"count": len(notes)})
	}
	return headline
}

func (r *PoliticsReport) LogText() string {
	if r.Count() == 0 {
		return ""
	}
	lines := make([]string, 0, len(r.Lines))
	for _, line := range r.Lines {
		text := r.msg.Render(line)
		if strings.HasPrefix(line.Key, politicsNoteKey) {
			text = r.msg.Text("politics.log.note", exportmsg.Args{"text": text})
		}
		lines = append(lines, text)
	}
	return worldLintLogText(commentLines(r.msg.Text("politics.log.header", nil)), lines)
}

func (r *PoliticsReport) LogEntry(modDir string) Entry {
	return worldLintEntry(modDir, PoliticsLogName, r.LogText(), "politics_export_lint")
}

func (r *PoliticsReport) CoverageTag() string { return "politics_export_lint" }
