package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateProjectComposeParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "validate_compose_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name  string `json:"name"`
			Input struct {
				ProjectName   string   `json:"project_name"`
				CountryTag    string   `json:"country_tag"`
				Dependencies  []string `json:"dependencies"`
				StageStateIDs []int64  `json:"stage_state_ids"`
				States        []struct {
					StateID   int64   `json:"state_id"`
					Owner     string  `json:"owner"`
					Name      string  `json:"name"`
					Provinces []int64 `json:"provinces"`
				} `json:"states"`
				HistoryFiles []struct {
					CountryTag string `json:"country_tag"`
					Capital    *int64 `json:"capital"`
					Ruling     string `json:"set_politics_ruling_party"`
				} `json:"history_files"`
				Techs []struct {
					ID            int64    `json:"id"`
					TechID        string   `json:"tech_id"`
					Prerequisites []string `json:"prerequisites"`
				} `json:"techs"`
			} `json:"input"`
			Issues json.RawMessage `json:"issues"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Cases) == 0 {
		t.Fatal("fixture 为空")
	}

	for _, c := range fx.Cases {
		in := ValidateProjectInput{
			Core: ValidateInput{
				ProjectID: 7, ProjectName: c.Input.ProjectName,
				CountryTag: c.Input.CountryTag, Dependencies: c.Input.Dependencies,
			},
			StageStateIDs: c.Input.StageStateIDs,
			IsSubmod:      len(c.Input.Dependencies) > 0,
		}
		for _, s := range c.Input.States {
			in.States = append(in.States, ValidateState{
				StateID: s.StateID, Owner: s.Owner, Name: s.Name, Provinces: s.Provinces})
		}
		for _, h := range c.Input.HistoryFiles {
			row := ValidateHistoryFile{CountryTag: h.CountryTag,
				SetPoliticsRulingParty: h.Ruling}
			if h.Capital != nil {
				row.Capital = *h.Capital
			}
			in.HistoryFiles = append(in.HistoryFiles, row)
		}
		for _, tc := range c.Input.Techs {
			in.Core.Techs = append(in.Core.Techs, ValidateTech{
				ID: tc.ID, TechID: tc.TechID, Prerequisites: tc.Prerequisites})
		}
		compareJSON(t, "validate-compose "+c.Name, ValidateProject(in), c.Issues)
	}
}

func TestOutOfStageIronRules(t *testing.T) {
	issues := []map[string]any{
		{"level": "error", "severity": "blocker",
			"entity_ref": map[string]any{"state_id": 999}},
		{"level": "warning", "severity": "warning",
			"entity_ref": map[string]any{"state_id": 999}},
		{"level": "warning", "severity": "warning",
			"entity_ref": map[string]any{"state_id": 16}},
	}
	tagOutOfStageIssues(issues, []int64{16})
	if len(issues) != 3 {
		t.Fatalf("后置只打标不删除,条数应仍为 3,得到 %d", len(issues))
	}
	if _, tagged := issues[0]["out_of_stage"]; tagged {
		t.Error("error/blocker 永不打标")
	}
	if v, _ := issues[1]["out_of_stage"].(bool); !v {
		t.Error("舞台外的 warning 应当打标")
	}
	if _, tagged := issues[2]["out_of_stage"]; tagged {
		t.Error("舞台内的 warning 不该打标")
	}
}

func TestStateIDRefRejectsBool(t *testing.T) {
	if _, ok := stateIDFromRef(true); ok {
		t.Error("true 不是 state id")
	}
	if _, ok := stateIDFromRef(false); ok {
		t.Error("false 不是 state id")
	}
	if id, ok := stateIDFromRef(16); !ok || id != 16 {
		t.Errorf("int 16 应当被接受,得到 %v,%v", id, ok)
	}
	if id, ok := stateIDFromRef(float64(16)); !ok || id != 16 {
		t.Errorf("JSON 往返来的 16.0 应当被接受,得到 %v,%v", id, ok)
	}
	if _, ok := stateIDFromRef(16.5); ok {
		t.Error("非整值不是 state id")
	}
	if _, ok := stateIDFromRef("16"); ok {
		t.Error("字符串不是 state id")
	}
}
