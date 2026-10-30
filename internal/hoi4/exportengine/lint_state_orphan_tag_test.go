package exportengine

import (
	"strings"
	"testing"

	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportmsg"
)

func TestScanStateOrphanTagsEmpty(t *testing.T) {
	r := ScanStateOrphanTags(worldout.Meta{}, english)
	if r.Count() != 0 || r.Headline() != "" || r.LogText() != "" || r.LogEntry("Demo").Path != "" {
		t.Fatalf("empty meta must be silent: %+v", r)
	}
	var nilReport *StateOrphanTagReport
	if nilReport.Count() != 0 || nilReport.Headline() != "" || nilReport.States() != nil {
		t.Fatal("nil report must be silent")
	}
}

func orphanLine(state string, name any, field, tag string) exportmsg.Message {
	return exportmsg.Message{Key: "orphan.state", Args: exportmsg.Args{"state": state, "name": name, "field": field, "tag": tag}}
}

func TestScanStateOrphanTagsHeadlineCountsStatesAndNamesTags(t *testing.T) {
	meta := worldout.Meta{Orphan: []exportmsg.Message{
		orphanLine("5", "Testland", "owner", "XQZ"),
		orphanLine("5", "Testland", "add_core_of", "XQZ"),
		orphanLine("9", exportmsg.Message{Key: "orphan.unnamed"}, "controller", "qq1"),
	}}
	r := ScanStateOrphanTags(meta, english)
	if r.Count() != 3 {
		t.Fatalf("count=%d", r.Count())
	}
	if got := strings.Join(r.States(), ","); got != "5,9" {
		t.Fatalf("states=%q", got)
	}
	if got := strings.Join(r.Tags(), ","); got != "XQZ,qq1" {
		t.Fatalf("tags=%q", got)
	}
	want := "State owners: states that name an unknown country tag as owner, controller or core (2; tags: XQZ, qq1). " +
		"HOI4 reports errors when loading them — add the country to the project or fix the tag (details in _STATE_ORPHAN_TAG_LINT.log)."
	if h := r.Headline(); h != want {
		t.Fatalf("headline\n got: %s\nwant: %s", h, want)
	}
	log := r.LogText()
	if !strings.HasPrefix(log, "# States that name a country that does not exist\n") ||
		!strings.Contains(log, "importing the missing country files again.\n\nState 5 (Testland): owner = XQZ") ||
		!strings.Contains(log, "\nState 9 (unnamed): controller = qq1") ||
		!strings.HasSuffix(log, "HOI4 reports an error when loading.\n") {
		t.Fatalf("log shape:\n%s", log)
	}
	entry := r.LogEntry("Demo")
	if entry.Path != "Demo/"+StateOrphanTagLogName || entry.SourceEntity != "state_orphan_tag_lint" {
		t.Fatalf("entry: %+v", entry)
	}
}

func TestScanStateOrphanTagsAgainstWorldEmitter(t *testing.T) {
	meta := worldMetaFor(t, loadLintWorldSnapshot(t), false)
	r := ScanStateOrphanTags(meta, english)
	if r.Count() != 2 || strings.Join(r.Tags(), ",") != "XQZ" || strings.Join(r.States(), ",") != "5" {
		t.Fatalf("unexpected orphan lines: %v", r.Lines)
	}
	for _, line := range r.Lines {
		if tag := line.Args["tag"]; tag == "ZZ1" || tag == "GER" {
			t.Fatalf("known TAG flagged: %v", line)
		}
	}
}
