package focus

import (
	"strings"
	"testing"
)

func TestEveryFocusScopeFindingCodeHasMessage(t *testing.T) {
	for _, code := range FocusFindingCodes {
		finding := FocusPolicyFinding{
			Code: code,
			Evidence: map[string]any{
				"country_tag":          "ENG",
				"scope_kind":           "wat",
				"country_tags":         []string{"e", "ENG"},
				"project_country_tags": []string{"GER", "SOV", "USA"},
			},
		}
		message := focusFindingMessage(finding)
		if strings.TrimSpace(message) == "" {
			t.Errorf("[FAIL] %s 没有人话。加 code 就要同时加一句 —— "+
				"只给 code 和 evidence 等于让用户自己悟。", code)
			continue
		}
		if strings.HasSuffix(strings.TrimSpace(message), "?") {
			t.Errorf("%s: 消息以问号结尾 —— 拦截要给出路,不是反问用户", code)
		}
	}
}

func TestForeignCountryBlockNamesTheRemedyNotJustTheEvidence(t *testing.T) {
	allowed := []string{"GER", "SOV", "USA"}
	m := MaterializeFocusBundle(FocusScopeCandidate{
		TreeKind: "primary", TreeID: "pvp_eng_tree", ScopeKind: "own", CountryTag: "ENG",
	}, allowed, nil, "focus_tree:create")

	if !m.Blocked() {
		t.Fatal("expected a blocker: the project has no ENG")
	}
	var found *FocusPolicyFinding
	for i := range m.Findings {
		if m.Findings[i].Code == "focus_scope_own_foreign_country" {
			found = &m.Findings[i]
		}
	}
	if found == nil {
		t.Fatal("focus_scope_own_foreign_country not among the findings")
	}
	for _, want := range []string{"ENG", "3", "project_country_tags"} {
		if !strings.Contains(found.Message, want) {
			t.Errorf("message %q does not mention %q", found.Message, want)
		}
	}

	for _, tag := range allowed {
		if strings.Count(found.Message, tag) > 0 && tag != "ENG" {
			t.Errorf("message lists project tag %q verbatim — say how many, point at evidence", tag)
		}
	}
}

func TestFocusFindingCodesCoversEveryCodeThePolicyEmits(t *testing.T) {
	emitted := map[string]bool{}
	cases := []struct {
		candidate FocusScopeCandidate
		allowed   []string
	}{
		{FocusScopeCandidate{TreeKind: "shared_pool", ScopeKind: "own", CountryTag: "GER"}, nil},
		{FocusScopeCandidate{TreeKind: "primary", ScopeKind: "nonsense"}, nil},
		{FocusScopeCandidate{TreeKind: "primary", ScopeKind: "own"}, nil},
		{FocusScopeCandidate{TreeKind: "primary", ScopeKind: "explicit"}, nil},
		{FocusScopeCandidate{TreeKind: "primary", ScopeKind: "global_except"}, nil},
		{FocusScopeCandidate{TreeKind: "primary", ScopeKind: "explicit", SharedCountryTags: []string{"toolongtag"}}, nil},
		{FocusScopeCandidate{TreeKind: "primary", ScopeKind: "own", CountryTag: "ENG"}, []string{"GER"}},
	}
	for _, tc := range cases {
		for _, f := range MaterializeFocusBundle(tc.candidate, tc.allowed, nil, "test").Findings {
			emitted[f.Code] = true
			if strings.TrimSpace(f.Message) == "" {
				t.Errorf("[FAIL] %s came out of the policy with an empty Message", f.Code)
			}
		}
	}
	for _, code := range FocusFindingCodes {
		if !emitted[code] {
			t.Logf("note: %s not exercised by this table (message coverage still checked above)", code)
		}
	}
	for code := range emitted {
		listed := false
		for _, known := range FocusFindingCodes {
			if known == code {
				listed = true
			}
		}
		if !listed {
			t.Errorf("[FAIL] policy emitted %q but FocusFindingCodes does not list it — "+
				"the completeness check is blind to it", code)
		}
	}
}
