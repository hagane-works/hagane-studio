package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestValidateStaticParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "validate_static_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name   string `json:"name"`
			States []struct {
				StateID   int64   `json:"state_id"`
				Owner     string  `json:"owner"`
				Name      string  `json:"name"`
				Provinces []int64 `json:"provinces"`
			} `json:"states"`
			HistoryFiles []struct {
				CountryTag string `json:"country_tag"`
				Capital    int64  `json:"capital"`
			} `json:"history_files"`
			Techs []struct {
				ID            int64    `json:"id"`
				TechID        string   `json:"tech_id"`
				Prerequisites []string `json:"prerequisites"`
			} `json:"techs"`
			StageStateIDs []int64         `json:"stage_state_ids"`
			Issues        json.RawMessage `json:"issues"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	for _, c := range fx.Cases {
		states := make([]ValidateState, 0, len(c.States))
		for _, s := range c.States {
			states = append(states, ValidateState{
				StateID: s.StateID, Owner: s.Owner, Name: s.Name, Provinces: s.Provinces})
		}
		hfs := make([]ValidateHistoryFile, 0, len(c.HistoryFiles))
		for _, h := range c.HistoryFiles {
			hfs = append(hfs, ValidateHistoryFile{CountryTag: h.CountryTag, Capital: h.Capital})
		}
		techs := make([]ValidateTech, 0, len(c.Techs))
		for _, tc := range c.Techs {
			techs = append(techs, ValidateTech{
				ID: tc.ID, TechID: tc.TechID, Prerequisites: tc.Prerequisites})
		}

		issues := []map[string]any{}
		issues = append(issues, validateCapitalNotOwned(states, hfs)...)
		issues = append(issues, validateProvinceDoubleMapping(states)...)
		issues = append(issues, validateTechCircularPrereq(techs)...)
		tagOutOfStage(issues, c.StageStateIDs)
		issues = appendSuccessLine(issues)

		got := jsonNorm(t, issues)
		var want any
		if err := json.Unmarshal(c.Issues, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("case %s\n got %s\nwant %s", c.Name, gb, c.Issues)
		}
	}
}

func TestStageTaggingNeverTouchesBlockers(t *testing.T) {
	issues := []map[string]any{
		{"level": "error", "severity": "blocker",
			"entity_ref": map[string]any{"type": "state", "id": int64(1), "state_id": int64(999)}},
		{"level": "warning",
			"entity_ref": map[string]any{"type": "history_file", "id": "GER", "state_id": int64(999)}},
	}
	tagOutOfStage(issues, []int64{16})
	if _, tagged := issues[0]["out_of_stage"]; tagged {
		t.Error("blocker 被打上了 out_of_stage —— 真 CTD 无论舞台都要大声")
	}
	if issues[1]["out_of_stage"] != true {
		t.Error("舞台外的 warning 应当被标记")
	}
	if len(issues) != 2 {
		t.Error("标记只加 key,不许删条目")
	}
}

func TestTechCycleReportsOnlyOnce(t *testing.T) {
	techs := []ValidateTech{
		{ID: 1, TechID: "a", Prerequisites: []string{"b"}},
		{ID: 2, TechID: "b", Prerequisites: []string{"a"}},
		{ID: 3, TechID: "c", Prerequisites: []string{"d"}},
		{ID: 4, TechID: "d", Prerequisites: []string{"c"}},
	}
	if got := validateTechCircularPrereq(techs); len(got) != 1 {
		t.Errorf("两个独立环应当只报 1 条(既有契约),得到 %d 条", len(got))
	}
}

func TestSuccessLineSurvivesInfo(t *testing.T) {
	withInfo := appendSuccessLine([]map[string]any{{"level": "info"}})
	if len(withInfo) != 2 || withInfo[1]["level"] != "success" {
		t.Errorf("info 不该抑制 success 行:%v", withInfo)
	}
	withWarning := appendSuccessLine([]map[string]any{{"level": "warning"}})
	if len(withWarning) != 1 {
		t.Errorf("warning 应当抑制 success 行:%v", withWarning)
	}
}
