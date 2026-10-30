package exportengine

import (
	"strings"
	"testing"
)

func TestPanelLocalisationKeepsMultiLineTextOnItsLine(t *testing.T) {
	rows := []map[string]any{{
		"token": "CC1_status", "name_en": "Status",
		"controls": []any{map[string]any{
			"kind": "text", "name": "body", "text": "line one\r\nFOO_KEY:0 \"injected\"",
		}},
	}}
	out := GenerateUIPanelsLocalisation(rows, "en")
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header, title and one text line, got %d:\n%s", len(lines), out)
	}
	if want := ` CC1_status_body_loc:0 "line one\nFOO_KEY:0 \"injected\""`; lines[2] != want {
		t.Fatalf("text line = %q, want %q", lines[2], want)
	}
}

func TestPanelRawExtraBooleanIsYesNo(t *testing.T) {
	_, txt, err := CompileUIPanel(map[string]any{
		"token": "CC1_status", "context_type": "decision_category",
		"raw_extras": map[string]any{"dirty": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(txt, "dirty = yes") || strings.Contains(txt, "= true") {
		t.Fatalf("boolean extra not written as yes: %s", txt)
	}
}
