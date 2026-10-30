package lint

import "testing"

func TestPanelContractCoreCoversOnlyDatabaseAuthoritativeChecks(t *testing.T) {
	panels := []Panel{{
		Token: "A B", DisplayNameLocs: map[string]any{},
		Controls: []map[string]any{
			{"kind": "icon", "name": "same name", "sprite": ""},
			{"kind": "text", "name": "same_name", "text": ""},
		},
		VisibleTrigger: "my_missing = yes", RawExtras: map[string]any{},
	}, {Token: "A_B", DisplayNameLocs: map[string]any{}, Controls: []map[string]any{}, RawExtras: map[string]any{}}}
	report := BuildPanelContractCore(panels, nil)
	rows := report["findings"].([]map[string]any)
	codes := map[string]bool{}
	for _, row := range rows {
		codes[row["code"].(string)] = true
	}
	for _, code := range []string{"panel.token_collision", "panel.control_collision", "panel.unknown_scripted", "panel.sprite_empty", "panel.loc_blank"} {
		if !codes[code] {
			t.Errorf("missing %s in %#v", code, codes)
		}
	}
	for _, forbidden := range []string{"panel.window_collision", "panel.file_collision", "panel.sprite_missing"} {
		if codes[forbidden] {
			t.Errorf("environment finding %s leaked into Go core", forbidden)
		}
	}
}

func TestPanelLocBlankSkipsWordlessButtons(t *testing.T) {
	panel := Panel{Token: "GER_politics_tab_ui", DisplayNameLocs: map[string]any{}, RawExtras: map[string]any{},
		Controls: []map[string]any{
			{"kind": "button", "name": "picture", "sprite": "GFX_decision_generic_political_discourse", "text": ""},
			{"kind": "button", "name": "zh_only", "text": "", "text_locs": map[string]any{"zh": "开始"}},
		}}
	findings := panelLocBlankFindings(panel, nil)
	if len(findings) != 1 || findings[0].Detail["control"] != "zh_only" {
		t.Fatalf("want only zh_only flagged, got %#v", findings)
	}
}
