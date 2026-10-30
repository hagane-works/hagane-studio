package rawfiles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"haganeworks/internal/hoi4/clausewitz"
)

func TestCrossRefRepairPlanParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "repair_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name              string  `json:"name"`
			Text              string  `json:"text"`
			TargetID          string  `json:"target_id"`
			TargetStartOffset int     `json:"target_start_offset"`
			TargetEndOffset   int     `json:"target_end_offset"`
			ReplacementID     *string `json:"replacement_id"`
			DeleteAssignment  bool    `json:"delete_assignment"`
			SourceSHA256      *string `json:"source_sha256"`
			Error             *struct {
				Code       string         `json:"code"`
				Message    string         `json:"message"`
				StatusCode int            `json:"status_code"`
				Context    map[string]any `json:"context"`
			} `json:"error"`
			Response    json.RawMessage `json:"response"`
			AppliedText *string         `json:"applied_text"`
		} `json:"cases"`
		AllowedFields map[string][]string `json:"allowed_fields"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	got := AllowedCrossRefRepairFields()
	if !reflect.DeepEqual(got, fx.AllowedFields) {
		t.Errorf("字段白名单不一致\n got %v\nwant %v", got, fx.AllowedFields)
	}

	ctx := map[string]any{
		"project_id": 1, "entity_kind": "focus", "entity_id": 3,
		"field": "completion_reward",
	}
	for _, c := range fx.Cases {
		in := CrossRefRepairInput{
			EntityKind: "focus", EntityID: 3, Field: "completion_reward",
			TargetID: c.TargetID, TargetStartOffset: c.TargetStartOffset,
			TargetEndOffset: c.TargetEndOffset, ReplacementID: c.ReplacementID,
			DeleteAssignment: c.DeleteAssignment, SourceSHA256: c.SourceSHA256,
		}
		cand, plan, err := buildRepairPlan(c.Text, in, copyCtx(ctx))

		if c.Error != nil {
			if err == nil {
				t.Errorf("case %s: 应当报 %s,却成功了", c.Name, c.Error.Code)
				continue
			}
			re, ok := err.(*CrossRefRepairError)
			if !ok {
				t.Errorf("case %s: 错误类型不对 %T", c.Name, err)
				continue
			}
			if re.Code != c.Error.Code || re.Message != c.Error.Message ||
				re.StatusCode != c.Error.StatusCode {
				t.Errorf("case %s\n got (%s,%q,%d)\nwant (%s,%q,%d)", c.Name,
					re.Code, re.Message, re.StatusCode,
					c.Error.Code, c.Error.Message, c.Error.StatusCode)
			}

			gotCtx := jsonNorm(t, re.Context)
			wantCtx := jsonNorm(t, c.Error.Context)
			if !reflect.DeepEqual(gotCtx, wantCtx) {
				gb, _ := json.Marshal(gotCtx)
				wb, _ := json.Marshal(wantCtx)
				t.Errorf("case %s context\n got %s\nwant %s", c.Name, gb, wb)
			}
			continue
		}

		if err != nil {
			t.Errorf("case %s: 意外报错 %v", c.Name, err)
			continue
		}
		gotResp := jsonNorm(t, repairPlanResponse(in, cand, plan))
		var wantResp any
		if err := json.Unmarshal(c.Response, &wantResp); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotResp, wantResp) {
			gb, _ := json.Marshal(gotResp)
			t.Errorf("case %s response\n got %s\nwant %s", c.Name, gb, c.Response)
			continue
		}
		if c.AppliedText != nil {
			applied, err := clausewitz.ApplyCrossRefRepair(c.Text, plan)
			if err != nil {
				t.Errorf("case %s apply: %v", c.Name, err)
			} else if applied != *c.AppliedText {
				t.Errorf("case %s applied\n got %q\nwant %q", c.Name, applied, *c.AppliedText)
			}
		}
	}
}

func copyCtx(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func TestRepairFieldWhitelistIsIdentifiersOnly(t *testing.T) {
	for kind, fields := range repairEntitySpecs {
		for f := range fields {
			for i, r := range f {
				ok := r == '_' || (r >= 'a' && r <= 'z') ||
					(i > 0 && r >= '0' && r <= '9')
				if !ok {
					t.Errorf("%s.%s 含非法字符 %q", kind, f, r)
				}
			}
		}
	}
}
