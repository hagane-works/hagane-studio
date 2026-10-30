package exportengine

import (
	"fmt"

	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportmsg"
)

const StateOrphanTagLogName = "_STATE_ORPHAN_TAG_LINT.log"

type StateOrphanTagReport struct {
	Lines []exportmsg.Message
	msg   exportmsg.Printer
}

func ScanStateOrphanTags(meta worldout.Meta, msg exportmsg.Printer) *StateOrphanTagReport {
	r := &StateOrphanTagReport{msg: msg}
	for _, line := range meta.Orphan {
		if line.Key == "" {
			continue
		}
		r.Lines = append(r.Lines, line)
	}
	return r
}

func (r *StateOrphanTagReport) Count() int {
	if r == nil {
		return 0
	}
	return len(r.Lines)
}

func (r *StateOrphanTagReport) States() []string {
	return r.column("state")
}

func (r *StateOrphanTagReport) Tags() []string {
	return r.column("tag")
}

func (r *StateOrphanTagReport) column(arg string) []string {
	if r == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, line := range r.Lines {
		value := fmt.Sprint(line.Args[arg])
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func (r *StateOrphanTagReport) Headline() string {
	if r.Count() == 0 {
		return ""
	}
	return r.msg.Text("orphan.headline", exportmsg.Args{
		"count": len(r.States()), "tags": joinCapped(r.Tags(), 5, r.msg), "log": StateOrphanTagLogName,
	})
}

func (r *StateOrphanTagReport) LogText() string {
	if r.Count() == 0 {
		return ""
	}
	return worldLintLogText(commentLines(r.msg.Text("orphan.log.header", nil)), worldLintRender(r.Lines, r.msg))
}

func (r *StateOrphanTagReport) LogEntry(modDir string) Entry {
	return worldLintEntry(modDir, StateOrphanTagLogName, r.LogText(), "state_orphan_tag_lint")
}

func (r *StateOrphanTagReport) CoverageTag() string { return "state_orphan_tag_lint" }
