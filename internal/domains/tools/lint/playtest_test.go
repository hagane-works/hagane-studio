package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPlaytestParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "playtest_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Math []struct {
			Date    *string `json:"date"`
			Year    *int    `json:"year"`
			GameLog *string `json:"game_log"`
			Loaded  *string `json:"loaded"`
		} `json:"math"`
		Runtime []struct {
			SystemLog string           `json:"system_log"`
			GameLog   string           `json:"game_log"`
			Inventory *string          `json:"inventory"`
			Expected  *expectedFixture `json:"expected"`
			Result    json.RawMessage  `json:"result"`
			Error     *string          `json:"error"`
		} `json:"runtime"`
		Reports []struct {
			Bookmarks       [][]any         `json:"bookmarks"`
			GameLog         string          `json:"game_log"`
			DLCLoad         string          `json:"dlc_load"`
			Exception       string          `json:"exception"`
			SystemLog       string          `json:"system_log"`
			Inventory       *string         `json:"inventory"`
			ExpectedName    string          `json:"expected_name"`
			ExpectedPath    string          `json:"expected_path"`
			ExpectedBuildID string          `json:"expected_build_id"`
			Report          json.RawMessage `json:"report"`
		} `json:"reports"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	for _, m := range fx.Math {
		if m.Date != nil {
			got := PlaytestLeadingYear(*m.Date)
			if (got == nil) != (m.Year == nil) || (got != nil && *got != *m.Year) {
				t.Errorf("PlaytestLeadingYear(%q) = %v want %v", *m.Date, got, m.Year)
			}
		}
		if m.GameLog != nil {
			got := PlaytestLoadedStartDate(*m.GameLog)
			want := ""
			if m.Loaded != nil {
				want = *m.Loaded
			}
			if got != want {
				t.Errorf("PlaytestLoadedStartDate(%q) = %q want %q", *m.GameLog, got, want)
			}
		}
	}

	for i, rc := range fx.Runtime {
		got, err := AnalyzeRuntimeDiagnostics(rc.SystemLog, rc.GameLog,
			rc.Expected.toIdentity(), rc.Inventory)
		if rc.Error != nil {
			if err == nil {
				t.Errorf("runtime case %d: want error %q, got none", i, *rc.Error)
			} else if err.Error() != *rc.Error && !isJSONDecodeCase(*rc.Error) {
				t.Errorf("runtime case %d: error %q want %q", i, err.Error(), *rc.Error)
			}
			continue
		}
		if err != nil {
			t.Errorf("runtime case %d: unexpected error %v", i, err)
			continue
		}
		gotTree := jsonNorm(t, got)
		var wantTree any
		if err := json.Unmarshal(rc.Result, &wantTree); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotTree, wantTree) {
			gb, _ := json.Marshal(gotTree)
			wb, _ := json.Marshal(wantTree)
			t.Errorf("runtime case %d mismatch\n got %.1000s\nwant %.1000s", i, gb, wb)
		}
	}

	for i, rc := range fx.Reports {
		builtDate, bookmarkID := pickBookmark(t, rc.Bookmarks)
		in := PlaytestBundleInput{
			GameLog: strOrNil(rc.GameLog), ExceptionTxt: strOrNil(rc.Exception),
			DLCLoadJSON: strOrNil(rc.DLCLoad), SystemLog: strOrNil(rc.SystemLog),
			ModInventoryJSON: rc.Inventory,
			ExpectedName:     rc.ExpectedName, ExpectedPath: rc.ExpectedPath,
			ExpectedBuildID: rc.ExpectedBuildID,
		}
		gotTree := normJSONDecodeMsg(jsonNorm(t, BuildPlaytestReport(in, builtDate, bookmarkID)))
		var wantTree any
		if err := json.Unmarshal(rc.Report, &wantTree); err != nil {
			t.Fatal(err)
		}
		wantTree = normJSONDecodeMsg(wantTree)
		if !reflect.DeepEqual(gotTree, wantTree) {
			gb, _ := json.Marshal(gotTree)
			wb, _ := json.Marshal(wantTree)
			t.Errorf("report case %d mismatch\n got %s\nwant %s", i, gb, wb)
		}
	}
}

type expectedFixture struct {
	Name    string  `json:"name"`
	Path    string  `json:"path"`
	BuildID *string `json:"build_id"`
}

func (e *expectedFixture) toIdentity() *RuntimeExpectedIdentity {
	if e == nil {
		return nil
	}
	id := &RuntimeExpectedIdentity{Name: e.Name, Path: e.Path}
	if e.BuildID != nil {
		id.BuildID = *e.BuildID
	}
	return id
}

const jsonDecodeMsgPrefix = "descriptor inventory is not valid JSON: "

func isJSONDecodeCase(msg string) bool {
	return strings.HasPrefix(msg, jsonDecodeMsgPrefix)
}

func normJSONDecodeMsg(v any) any {
	switch t := v.(type) {
	case string:
		if isJSONDecodeCase(t) {
			return jsonDecodeMsgPrefix
		}
	case map[string]any:
		for k, sub := range t {
			t[k] = normJSONDecodeMsg(sub)
		}
	case []any:
		for i, sub := range t {
			t[i] = normJSONDecodeMsg(sub)
		}
	}
	return v
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func pickBookmark(t *testing.T, rows [][]any) (string, string) {
	t.Helper()
	if len(rows) == 0 {
		return "", ""
	}
	date, _ := rows[0][0].(string)
	id, _ := rows[0][1].(string)
	for _, r := range rows {
		if flag, _ := r[2].(bool); flag {
			d, _ := r[0].(string)
			i, _ := r[1].(string)
			return d, i
		}
	}
	return date, id
}
