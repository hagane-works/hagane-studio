package worldout

import (
	"fmt"
	"strings"
	"testing"
)

func TestOwnerOverridesStartTransfers(t *testing.T) {
	row := func(owner, source string) *OMap {
		st := NewOMap()
		st.Set("owner", owner)
		st.Set("source", source)
		return st
	}
	for _, c := range []struct {
		owner, source, vanilla string
		want                   bool
	}{
		{"Z01", "created", "CHI", true},
		{"chi", "created", "CHI", false},
		{"Z01", "imported", "CHI", false},
		{"Z01", "created", "", false},
		{"", "created", "CHI", false},
		{"SIC", "created", "CHI", true},
	} {
		if got := ownerOverridesStartTransfers(row(c.owner, c.source), c.vanilla); got != c.want {
			t.Fatalf("%+v: got %v", c, got)
		}
	}
}

func TestStripStartTransfersShapes(t *testing.T) {

	out, targets, ok := stripStartTransfers("IF = {\n\tlimit = { has_dlc = \"X\" }\n\ttransfer_state_to = SIC\n\tadd_core_of = SIC\n}\nELSE = {\n\tadd_core_of = CHI\n}")
	if !ok || fmt.Sprint(targets) != "[SIC]" || strings.Contains(out, "transfer_state_to") ||
		!strings.Contains(out, "IF = {") || !strings.Contains(out, "add_core_of = SIC") || !strings.Contains(out, "ELSE = {") {
		t.Fatalf("if with else: ok=%v targets=%v\n%s", ok, targets, out)
	}

	out, targets, ok = stripStartTransfers("buildings = { infrastructure = 3 }\nIF = { limit = { has_dlc = \"X\" } NXM = { transfer_state = PREV } }")
	if !ok || fmt.Sprint(targets) != "[NXM]" || strings.TrimSpace(out) != "buildings = { infrastructure = 3 }" {
		t.Fatalf("dated block: ok=%v targets=%v\n%q", ok, targets, out)
	}

	broken := "IF = { limit = { has_dlc = \"X\" } transfer_state_to = SIC"
	if out, targets, ok := stripStartTransfers(broken); ok || targets != nil || out != broken {
		t.Fatalf("broken text: ok=%v targets=%v\n%q", ok, targets, out)
	}

	if _, targets, ok := stripStartTransfers("# transfer_state_to = SIC\nset_state_name = \"transfer_state_to = SIC\""); !ok || len(targets) != 0 {
		t.Fatalf("comment/string: ok=%v targets=%v", ok, targets)
	}
}
