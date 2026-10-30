package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestErrorLogParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "errorlog_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Patterns []struct {
			Index  int    `json:"index"`
			Bucket string `json:"bucket"`
			Sub    string `json:"sub"`
			Label  string `json:"label"`
		} `json:"patterns"`
		Matrix []struct {
			Body string `json:"body"`
			Hits []int  `json:"hits"`
		} `json:"matrix"`
		Classify []struct {
			Body   string `json:"body"`
			Bucket string `json:"bucket"`
			Sub    string `json:"sub"`
			Label  string `json:"label"`
		} `json:"classify"`
		Entries []struct {
			Text    string `json:"text"`
			Entries []struct {
				Module    string `json:"module"`
				Body      string `json:"body"`
				Bucket    string `json:"bucket"`
				SubBucket string `json:"sub_bucket"`
				Label     string `json:"label"`
				RawSample string `json:"raw_sample"`
				RawCount  int    `json:"raw_count"`
			} `json:"entries"`
		} `json:"entries"`
		Map []struct {
			Body string          `json:"body"`
			Ref  json.RawMessage `json:"ref"`
			File *string         `json:"file"`
		} `json:"map"`
		Reports []struct {
			Text   string          `json:"text"`
			Report json.RawMessage `json:"report"`
		} `json:"reports"`
		IndexSpec map[string][][]*string `json:"index_spec"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	if len(logClassifiers) != len(fx.Patterns) {
		t.Fatalf("classifier count %d want %d", len(logClassifiers), len(fx.Patterns))
	}
	for i, p := range fx.Patterns {
		got := logClassifiers[i]
		if got.bucket != p.Bucket || got.sub != p.Sub || got.label != p.Label {
			t.Errorf("classifier[%d] = (%s,%s,%q) want (%s,%s,%q)",
				i, got.bucket, got.sub, got.label, p.Bucket, p.Sub, p.Label)
		}
	}

	for _, m := range fx.Matrix {
		var hits []int
		for i, c := range logClassifiers {
			if c.re.MatchString(m.Body) {
				hits = append(hits, i)
			}
		}
		if len(hits) == 0 {
			hits = []int{}
		}
		want := m.Hits
		if want == nil {
			want = []int{}
		}
		if !reflect.DeepEqual(hits, want) {
			t.Errorf("match matrix mismatch for %q\n got %v\nwant %v", m.Body, hits, want)
		}
	}

	for _, c := range fx.Classify {
		b, s, lab := ClassifyLogBody(c.Body)
		if b != c.Bucket || s != c.Sub || lab != c.Label {
			t.Errorf("classify(%q) = (%s,%s,%q) want (%s,%s,%q)",
				c.Body, b, s, lab, c.Bucket, c.Sub, c.Label)
		}
	}

	for ci, ec := range fx.Entries {
		got := AnalyzeErrorLog(ec.Text)
		if len(got) != len(ec.Entries) {
			t.Fatalf("log case %d: entries %d want %d", ci, len(got), len(ec.Entries))
		}
		for i, want := range ec.Entries {
			g := got[i]
			if g.Sig.Module != want.Module || g.Sig.Body != want.Body ||
				g.Bucket != want.Bucket || g.SubBucket != want.SubBucket ||
				g.Label != want.Label || g.RawSample != want.RawSample ||
				g.RawCount != want.RawCount {
				t.Errorf("log case %d entry %d mismatch\n got %+v\nwant %+v", ci, i, g, want)
			}
		}
	}

	ix := logIndexFromSpec(t, fx.IndexSpec)
	for _, mc := range fx.Map {
		b, s, lab := ClassifyLogBody(mc.Body)
		entry := LogEntry{
			Sig: LogSignature{Module: "m", Body: mc.Body}, Bucket: b,
			SubBucket: s, Label: lab, RawSample: mc.Body, RawCount: 1,
		}
		ref, fp := MapLogEntry(entry, ix)
		var gotRef any
		if ref != nil {
			gotRef = ref.toMap()
		}
		gotTree := jsonNorm(t, map[string]any{"ref": gotRef, "file": fp})
		var wantRef any
		if err := json.Unmarshal(mc.Ref, &wantRef); err != nil {
			t.Fatal(err)
		}
		var wantFile any
		if mc.File != nil {
			wantFile = *mc.File
		}
		wantTree := jsonNorm(t, map[string]any{"ref": wantRef, "file": wantFile})
		if !reflect.DeepEqual(gotTree, wantTree) {
			gb, _ := json.Marshal(gotTree)
			wb, _ := json.Marshal(wantTree)
			t.Errorf("map(%q)\n got %s\nwant %s", mc.Body, gb, wb)
		}
	}

	for ci, rc := range fx.Reports {
		gotTree := jsonNorm(t, BuildLogReportFromIndex(ix, rc.Text))
		var wantTree any
		if err := json.Unmarshal(rc.Report, &wantTree); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotTree, wantTree) {
			gb, _ := json.Marshal(gotTree)
			wb, _ := json.Marshal(wantTree)
			t.Errorf("report case %d mismatch\n got %.1200s\nwant %.1200s", ci, gb, wb)
		}
	}
}

func logIndexFromSpec(t *testing.T, spec map[string][][]*string) *LogProjectIndex {
	t.Helper()
	ix := NewLogProjectIndex()
	fill := func(kind string, dst map[string]LogEntityRef, rows [][]*string) {
		for _, r := range rows {
			if len(r) != 3 || r[0] == nil {
				t.Fatalf("bad index spec row %v", r)
			}
			ref := LogEntityRef{Kind: kind, EntityID: *r[0], Name: r[1], CountryTag: r[2]}
			dst[strings.ToLower(*r[0])] = ref
		}
	}
	fill("focus", ix.Focus, spec["focus"])
	fill("event", ix.Event, spec["event"])
	fill("decision", ix.Decision, spec["decision"])
	fill("decision", ix.DecisionCategory, spec["decision_category"])
	fill("character", ix.Character, spec["character"])
	fill("idea", ix.Idea, spec["idea"])
	fill("technology", ix.Technology, spec["technology"])
	fill("state", ix.State, spec["state"])
	fill("history_file", ix.HistoryFile, spec["history_file"])
	return ix
}
