package rawfiles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type rawFilesFixture struct {
	SanitizeCases []struct {
		Input   string `json:"input"`
		Output  string `json:"output"`
		Error   bool   `json:"error"`
		Message string `json:"message"`
	} `json:"sanitize_cases"`
	IntentCases []struct {
		RawKind             string  `json:"raw_kind"`
		DestPath            string  `json:"dest_path"`
		Content             string  `json:"content"`
		IsBinary            bool    `json:"is_binary"`
		GenerateReplacePath bool    `json:"generate_replace_path"`
		GameVersion         string  `json:"game_version"`
		TargetLayer         *string `json:"target_layer"`
		TargetSHA256        *string `json:"target_sha256"`
		Decision            *struct {
			RawKind             string  `json:"raw_kind"`
			DestPath            string  `json:"dest_path"`
			Content             string  `json:"content"`
			GenerateReplacePath bool    `json:"generate_replace_path"`
			TargetLayer         *string `json:"target_layer"`
			TargetSHA256        *string `json:"target_sha256"`
		} `json:"decision"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"intent_cases"`
}

func loadRawFilesFixture(t *testing.T) rawFilesFixture {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "raw_files_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx rawFilesFixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	return fx
}

func TestSanitizeDestPathParity(t *testing.T) {
	fx := loadRawFilesFixture(t)
	if len(fx.SanitizeCases) < 8 {
		t.Fatalf("few cases: %d", len(fx.SanitizeCases))
	}
	for _, c := range fx.SanitizeCases {
		got, errMsg := SanitizeDestPath(c.Input)
		if c.Error {
			if errMsg == "" {
				t.Errorf("%q: want error %q, got ok %q", c.Input, c.Message, got)
			} else if errMsg != c.Message {
				t.Errorf("%q: error mismatch\n got %q\nwant %q", c.Input, errMsg, c.Message)
			}
			continue
		}
		if errMsg != "" || got != c.Output {
			t.Errorf("%q: got (%q, err=%q) want %q", c.Input, got, errMsg, c.Output)
		}
	}
}

func TestValidateRawFileIntentParity(t *testing.T) {
	fx := loadRawFilesFixture(t)
	if len(fx.IntentCases) < 6 {
		t.Fatalf("few cases: %d", len(fx.IntentCases))
	}
	for i, c := range fx.IntentCases {
		layer, sha := "", ""
		if c.TargetLayer != nil {
			layer = *c.TargetLayer
		}
		if c.TargetSHA256 != nil {
			sha = *c.TargetSHA256
		}
		got, terr := ValidateRawFileIntent(c.RawKind, c.DestPath, c.Content,
			c.IsBinary, c.GenerateReplacePath, c.GameVersion, layer, sha)
		if c.Error != nil {
			if terr == nil {
				t.Errorf("case %d: want error %s, got decision %+v", i, c.Error.Code, got)
			} else if terr.Code != c.Error.Code || terr.Message != c.Error.Message {
				t.Errorf("case %d: error mismatch\n got (%s, %q)\nwant (%s, %q)",
					i, terr.Code, terr.Message, c.Error.Code, c.Error.Message)
			}
			continue
		}
		if terr != nil {
			t.Errorf("case %d: unexpected error (%s, %q)", i, terr.Code, terr.Message)
			continue
		}
		d := c.Decision
		gl, gs := "", ""
		if got.TargetLayer != nil {
			gl = *got.TargetLayer
		}
		if got.TargetSHA256 != nil {
			gs = *got.TargetSHA256
		}
		wl, ws := "", ""
		if d.TargetLayer != nil {
			wl = *d.TargetLayer
		}
		if d.TargetSHA256 != nil {
			ws = *d.TargetSHA256
		}
		if got.RawKind != d.RawKind || got.DestPath != d.DestPath || got.Content != d.Content ||
			got.GenerateReplacePath != d.GenerateReplacePath || gl != wl || gs != ws {
			t.Errorf("case %d: decision mismatch\n got %+v\nwant %+v", i, got, d)
		}
	}
}

func TestRejectReservedDest(t *testing.T) {
	if msg := RejectReservedDest("descriptor.mod"); msg == "" {
		t.Error("descriptor.mod should be reserved")
	}
	if msg := RejectReservedDest("Thumbnail.PNG"); msg == "" {
		t.Error("thumbnail.png should be reserved (case-insensitive)")
	}
	if msg := RejectReservedDest("common/descriptor.mod"); msg != "" {
		t.Errorf("nested descriptor.mod should be allowed: %q", msg)
	}
	for _, path := range []string{
		"_HAGANE_EXPORT_FINDINGS.json",
		"reports/_hagane_export_findings.JSON",
		`reports\_HAGANE_EXPORT_FINDINGS.json`,
		"_REFERENCE_INTEGRITY.log",
		"reports/_reference_integrity.LOG",
	} {
		if msg := RejectReservedDest(path); msg == "" {
			t.Errorf("reserved findings basename should be rejected: %q", path)
		}
	}
}
