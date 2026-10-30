package drawio

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonParityFixtures(t *testing.T) {

	raw, err := os.ReadFile(filepath.Join("testdata", "python_parity.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name     string `json:"name"`
		XML      string `json:"xml"`
		Prefix   string `json:"prefix"`
		Expected Result `json:"expected"`
	}
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			got, err := Parse(fixture.XML, fixture.Prefix)
			if err != nil {
				t.Fatal(err)
			}
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(fixture.Expected)
			if string(gotJSON) != string(wantJSON) {
				t.Fatalf("Go/Python mismatch\n got: %s\nwant: %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestRejectsUnsafeAndMalformedXML(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"xxe", `<!DOCTYPE x [<!ENTITY e SYSTEM "file:///etc/passwd">]><mxGraphModel>&e;</mxGraphModel>`, "unsafe XML construct"},
		{"entity-bomb", `<!DOCTYPE lolz [<!ENTITY lol "lol"><!ENTITY lol2 "&lol;&lol;">]><mxGraphModel/>`, "unsafe XML construct"},
		{"malformed", `<mxGraphModel><root>`, "Draw.io XML parse failed"},
		{"empty", " \n\t", "empty Draw.io payload"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input, "")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestDoctypeTextInsideCommentOrCDATAIsNotADeclaration(t *testing.T) {
	for _, input := range []string{
		`<mxGraphModel><!-- <!DOCTYPE harmless> --><root/></mxGraphModel>`,
		`<mxGraphModel><root><![CDATA[<!DOCTYPE harmless>]]></root></mxGraphModel>`,
	} {
		got, err := Parse(input, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Nodes) != 0 || len(got.Edges) != 0 || len(got.Warnings) != 0 {
			t.Fatalf("unexpected content: %#v", got)
		}
	}
}

func TestRejectsDecodedPayloadOverEightMiB(t *testing.T) {
	input := "<x>" + strings.Repeat("a", MaxBytes) + "</x>"
	_, err := Parse(input, "")
	if err == nil || !strings.Contains(err.Error(), "payload too large") {
		t.Fatalf("error = %v", err)
	}
}

func TestLongUnicodeLabelsMatchPythonRuneCaps(t *testing.T) {
	label := strings.Repeat("界", 300)
	input := `<mxGraphModel><root><mxCell id="n" value="` + label + `" vertex="1"/></root></mxGraphModel>`
	got, err := Parse(input, "")
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(got.Nodes[0].Name)) != maxFocusNameRunes {
		t.Fatalf("name runes = %d", len([]rune(got.Nodes[0].Name)))
	}

	if got.Nodes[0].FocusID != "focus" {
		t.Fatalf("focus id = %q", got.Nodes[0].FocusID)
	}
}

func TestLongASCIILabelCapsIDAndDisplayNameLikePython(t *testing.T) {
	label := strings.Repeat("x", 400)
	input := `<mxGraphModel><root><mxCell id="n" value="` + label + `" vertex="1"/></root></mxGraphModel>`
	got, err := Parse(input, "")
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(got.Nodes[0].FocusID)) != maxFocusIDRunes {
		t.Fatalf("focus id runes = %d", len([]rune(got.Nodes[0].FocusID)))
	}
	if len([]rune(got.Nodes[0].Name)) != maxFocusNameRunes {
		t.Fatalf("name runes = %d", len([]rune(got.Nodes[0].Name)))
	}
}
