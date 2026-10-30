package clausewitz

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScriptPositionParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "position_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name     string          `json:"name"`
			Text     string          `json:"text"`
			RootPath []string        `json:"root_path"`
			Report   json.RawMessage `json:"report"`
		} `json:"cases"`
		Registries []struct {
			Name        string          `json:"name"`
			Payload     json.RawMessage `json:"payload"`
			GameVersion string          `json:"game_version"`
			Verified    bool            `json:"verified"`
			Source      *string         `json:"verification_source"`
			FatalKinds  [][]string      `json:"fatal_kinds"`
		} `json:"registries"`
		Errors []struct {
			Name    string          `json:"name"`
			Payload json.RawMessage `json:"payload"`
			Error   *string         `json:"error"`
		} `json:"errors"`
		Evidence []struct {
			Requested *string         `json:"requested"`
			Evidence  json.RawMessage `json:"evidence"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	src := "f.txt"
	for _, c := range fx.Cases {
		rep, err := AnalyzeScriptPositions(c.Text, c.RootPath, &src, nil)
		if err != nil {
			t.Errorf("case %s: %v", c.Name, err)
			continue
		}
		got := jsonNorm(t, rep.ToMap())
		var want any
		if err := json.Unmarshal(c.Report, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("case %s mismatch\n got %s\nwant %s", c.Name, gb, c.Report)
		}
	}

	for _, r := range fx.Registries {
		reg, err := ParseFatalRegistry(r.Payload)
		if err != nil {
			t.Errorf("registry %s: %v", r.Name, err)
			continue
		}
		if reg.GameVersion != r.GameVersion || reg.Verified != r.Verified {
			t.Errorf("registry %s: got (%s,%v) want (%s,%v)",
				r.Name, reg.GameVersion, reg.Verified, r.GameVersion, r.Verified)
		}
		var gotSrc *string
		if reg.VerificationSource != nil {
			gotSrc = reg.VerificationSource
		}
		if (gotSrc == nil) != (r.Source == nil) ||
			(gotSrc != nil && r.Source != nil && *gotSrc != *r.Source) {
			t.Errorf("registry %s: source mismatch", r.Name)
		}
		if len(reg.FatalKinds) != len(r.FatalKinds) {
			t.Errorf("registry %s: kinds %d want %d", r.Name, len(reg.FatalKinds), len(r.FatalKinds))
		}
		for _, k := range r.FatalKinds {
			if !reg.FatalKinds[[2]string{k[0], k[1]}] {
				t.Errorf("registry %s: missing kind %v", r.Name, k)
			}
		}
	}

	for _, e := range fx.Errors {
		_, err := ParseFatalRegistry(e.Payload)
		if e.Error == nil {
			if err != nil {
				t.Errorf("registry error case %s: Go 报错 %v,Python 没报", e.Name, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("registry error case %s: 应当报 %q", e.Name, *e.Error)
		} else if err.Error() != *e.Error {
			t.Errorf("registry error case %s\n got %q\nwant %q", e.Name, err.Error(), *e.Error)
		}
	}
}

func TestScriptPositionRejectsStaleAnalysis(t *testing.T) {
	stale := AnalyzeAssignments("add_idea = a", nil, nil)
	if _, err := AnalyzeScriptPositions("完全不同的文本", nil, nil, &stale); err == nil {
		t.Error("source hash 不符时应当报错")
	}
}
