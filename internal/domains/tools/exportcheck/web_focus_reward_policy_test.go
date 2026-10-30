package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"haganeworks/internal/hoi4/clausewitz"
)

func TestCompletionRewardPolicyParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "reward_policy_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name  string `json:"name"`
			Input struct {
				Text         string  `json:"text"`
				FocusID      *string `json:"focus_id"`
				SourceName   *string `json:"source_name"`
				CurrentScope string  `json:"current_scope"`
			} `json:"input"`
			Expected json.RawMessage `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Cases) == 0 {
		t.Fatal("fixture 为空")
	}

	for _, c := range fx.Cases {
		got := jsonNorm(t, AnalyzeCompletionReward(CompletionRewardInput{
			Text:         c.Input.Text,
			FocusID:      c.Input.FocusID,
			SourceName:   c.Input.SourceName,
			CurrentScope: c.Input.CurrentScope,
		}).ToDict())
		var want any
		if err := json.Unmarshal(c.Expected, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.MarshalIndent(got, "", " ")
			t.Errorf("case %s\n got %s\nwant %s", c.Name, gb, c.Expected)
		}
	}
}

func TestLimitGuardsEveryDepth(t *testing.T) {
	deep := AnalyzeCompletionReward(CompletionRewardInput{
		Text: "if = { limit = { 278 = { GER = { has_country_flag = f } } } }"})
	for _, f := range deep.Findings {
		if f.Code == "focus.reward.trigger_in_effect_position" {
			t.Fatal("limit 下的 trigger 不该被判成 effect 位置")
		}
	}
	bare := AnalyzeCompletionReward(CompletionRewardInput{
		Text: "278 = { has_country_flag = f }"})
	found := false
	for _, f := range bare.Findings {
		if f.Code == "focus.reward.trigger_in_effect_position" {
			found = true
		}
	}
	if !found {
		t.Error("没有 limit 包裹时,trigger 写在 state 包装里应当被拦")
	}
}

func TestRandomListWeightKeepsScope(t *testing.T) {
	weight := AnalyzeCompletionReward(CompletionRewardInput{
		Text: "random_list = { 50 = { add_political_power = 10 } }"})
	if len(weight.Findings) != 0 {
		t.Errorf("random_list 权重下的 country effect 不该报错,得到 %v", weight.Findings)
	}
	nested := AnalyzeCompletionReward(CompletionRewardInput{
		Text: "random_list = { 50 = { 278 = { add_political_power = 10 } } }"})
	if len(nested.Findings) != 1 ||
		nested.Findings[0].Code != "focus.reward.unsupported_scope" {
		t.Errorf("权重下再套数字包装应当是 state 作用域,得到 %v", nested.Findings)
	}
}

func TestScalarOrBlockEffectsNeverBlock(t *testing.T) {
	cases := map[string][2]string{
		"promote_character": {
			"promote_character = ROM_king_michael",
			"promote_character = { character = ROM_king_michael ideology = liberalism }",
		},
		"load_focus_tree": {
			"load_focus_tree = polish_focus",
			"load_focus_tree = { tree = polish_focus keep_completed = yes }",
		},
		"set_country_flag": {
			"set_country_flag = BUL_soviet_influence_flag",
			"set_country_flag = { flag = BUL_soviet_influence_flag value = 1 days = 30 }",
		},
	}
	for name, texts := range cases {
		rule, ok := DefaultEffectRules[name]
		if !ok {
			t.Fatalf("%s 必须留在登记表里(作用域校验仍要跑)", name)
		}
		if !rule.acceptsKind(clausewitz.ValueScalar) || !rule.acceptsKind(clausewitz.ValueBlock) {
			t.Errorf("%s 应接受 {scalar, block},得到 %v", name, rule.ValueKinds)
		}
		for _, text := range texts {

			for _, wrapped := range []string{text, "else = { " + text + " }", "hidden_effect = { " + text + " }"} {
				got := AnalyzeCompletionReward(CompletionRewardInput{Text: wrapped})
				if got.Blocking() || len(got.RepairPlans) != 0 {
					t.Errorf("%s: %q 不该是 blocker,findings=%v plans=%d", name, wrapped, got.Findings, len(got.RepairPlans))
				}
			}
		}
	}
}

func TestScalarOnlyEffectsBlockFormStillBlocks(t *testing.T) {
	cases := map[string]string{
		"set_country_leader_ideology": "ideology",
		"set_cosmetic_tag":            "tag",
		"retire_character":            "character",
		"add_to_faction":              "faction",
		"add_political_power":         "value",
		"add_stability":               "value",
	}
	for name, param := range cases {
		text := name + " = { " + param + " = TARGET }"
		got := AnalyzeCompletionReward(CompletionRewardInput{Text: text})
		var hit int
		for _, f := range got.Findings {
			if f.Code == "focus.reward.scalar_effect_block_value" && f.Severity == "blocker" {
				hit++
			}
		}
		if hit != 1 {
			t.Errorf("%s: 块形应恰好报 1 条 scalar_effect_block_value,得到 %v", name, got.Findings)
		}
		if len(got.RepairPlans) != 1 || !got.RepairPlans[0].Applicable ||
			got.RepairPlans[0].PreviewAfter == nil || *got.RepairPlans[0].PreviewAfter != name+" = TARGET" {
			t.Errorf("%s: 应有一条可应用的确定性修复,得到 %+v", name, got.RepairPlans)
		}
		scalar := AnalyzeCompletionReward(CompletionRewardInput{Text: name + " = TARGET"})
		if scalar.Blocking() {
			t.Errorf("%s: 标量写法不该 blocker: %v", name, scalar.Findings)
		}
	}
}
