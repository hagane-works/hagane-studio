package guirender

import "testing"

func TestDecisionOriginNamesTheLayer(t *testing.T) {
	base, err := newIndex(map[string][]byte{
		"common/decisions/categories/base.txt": []byte(`political_actions = { }`),
		"common/decisions/base.txt": []byte(`political_actions = {
			keep = { cost = 10 }
			ban_party = { cost = 50 }
		}`),
	}, "vanilla")
	if err != nil {
		t.Fatal(err)
	}
	project, err := NewIndexWithFallback(map[string][]byte{
		"common/decisions/categories/mod.txt": []byte(`mod_tab = { }`),
		"common/decisions/mod.txt": []byte(`mod_tab = { mod_action = { cost = 5 } }
political_actions = {
	ban_party = { cost = 99 }
	extra = { cost = 1 }
}`),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		category, decision, want string
	}{
		{"political_actions", "", "vanilla"},
		{"mod_tab", "", "mod"},
		{"political_actions", "keep", "vanilla"},
		{"political_actions", "ban_party", "mod"},
		{"political_actions", "extra", "mod"},
		{"mod_tab", "mod_action", "mod"},
		{"political_actions", "missing", ""},
	} {
		got := project.decisionCategoryOrigin(tc.category)
		if tc.decision != "" {
			got = project.decisionOrigin(tc.category, tc.decision)
		}
		if got != tc.want {
			t.Errorf("%s/%s origin = %q, want %q", tc.category, tc.decision, got, tc.want)
		}
	}

	payload := project.DecisionsPayload("TAG")
	if origin := categoryNamed(payload, "political_actions")["origin"]; origin != "vanilla" {
		t.Errorf("payload category origin = %v", origin)
	}
	keep := decisionNamed(payload, "political_actions", "keep")
	if keep["origin"] != "vanilla" || keep["label_key"] != "keep" {
		t.Errorf("payload decision = %v, want origin vanilla and label_key keep", keep)
	}
	if origin := decisionNamed(payload, "mod_tab", "mod_action")["origin"]; origin != "mod" {
		t.Errorf("payload mod decision origin = %v", origin)
	}
}
