package focus

import "testing"

func codesOf(m FocusScopeMaterialization) map[string]bool {
	out := map[string]bool{}
	for _, f := range m.Findings {
		out[f.Code] = true
	}
	return out
}

func TestFocusCustodyOwnMissingCountry(t *testing.T) {
	m := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID:    "FIN_finland",
		TreeKind:  "primary",
		ScopeKind: "own",
	}, []string{"FIN"}, nil, "")
	if !m.Blocked() {
		t.Fatal("own scope without an explicit country must block")
	}
	if !codesOf(m)["focus_scope_own_missing_country"] {
		t.Fatal("own scope blocker code must be stable")
	}
	if m.CountryTag != "" {
		t.Fatal("tree_id must never infer FIN country scope")
	}
}

func TestFocusCustodySemanticVerdictAuthoritative(t *testing.T) {
	m := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID:   "unrelated_name",
		TreeKind: "primary",
	}, []string{"FIN"}, &FocusScopeVerdict{Kind: "own", CountryTags: []string{"FIN"}}, "")
	if m.Blocked() || m.CountryTag != "FIN" {
		t.Fatalf("semantic verdict must be authoritative: %+v", m)
	}

	stale := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID:     "stale",
		TreeKind:   "primary",
		ScopeKind:  "own",
		CountryTag: "FIN",
	}, []string{"FIN"}, &FocusScopeVerdict{Kind: "own"}, "")
	if !stale.Blocked() || stale.CountryTag != "" {
		t.Fatalf("stale persisted scope must not supplement verdict: %+v", stale)
	}
}

func TestFocusCustodyForeignOwn(t *testing.T) {
	m := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID:   "tree",
		TreeKind: "primary",
	}, []string{"FIN"}, &FocusScopeVerdict{Kind: "own", CountryTags: []string{"SWE"}}, "")
	if !m.Blocked() || !codesOf(m)["focus_scope_own_foreign_country"] {
		t.Fatalf("foreign own scope must fail closed: %+v", m)
	}
	for _, f := range m.Findings {
		if f.Code == "focus_scope_own_foreign_country" {
			got, ok := f.Evidence["project_country_tags"].([]string)
			if !ok || len(got) != 1 || got[0] != "FIN" {
				t.Fatalf("evidence project_country_tags must be sorted allowed list: %+v", f.Evidence)
			}
		}
	}
}

func TestFocusCustodyGlobalScope(t *testing.T) {
	m := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID:   "FIN_global-looking",
		TreeKind: "primary",
	}, []string{"FIN"}, &FocusScopeVerdict{Kind: "global"}, "")
	if m.Blocked() || m.CountryTag != "" {
		t.Fatalf("explicit global scope must stay global: %+v", m)
	}
	if m.ScopeKind != "global" {
		t.Fatalf("scope_kind = %q", m.ScopeKind)
	}
}

func TestFocusCustodySharedPool(t *testing.T) {
	m := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID:     "FIN_shared",
		TreeKind:   "shared_pool",
		ScopeKind:  "own",
		CountryTag: "FIN",
	}, []string{"FIN"}, nil, "")
	if m.Blocked() {
		t.Fatal("shared pool scope cleanup is review-only")
	}
	if m.CountryTag != "" || m.ScopeKind != "shared_pool" {
		t.Fatalf("shared pool must not enter primary country custody: %+v", m)
	}
	if !codesOf(m)["focus_shared_pool_scope_ignored"] {
		t.Fatal("discarded shared-pool scope metadata must be visible")
	}

	clean := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID: "pool", TreeKind: "shared_pool",
	}, nil, nil, "")
	if len(clean.Findings) != 0 {
		t.Fatalf("clean shared pool must have no findings: %+v", clean.Findings)
	}
}

func TestFocusCustodyUnknownKindAndInvalidTag(t *testing.T) {
	unknown := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID: "t", TreeKind: "primary", ScopeKind: "mystery",
	}, nil, nil, "")
	if !unknown.Blocked() || !codesOf(unknown)["focus_scope_unknown_kind"] {
		t.Fatalf("unknown scope kind must block: %+v", unknown)
	}
	if unknown.ScopeKind != "mystery" {
		t.Fatalf("unknown kind must be echoed back: %q", unknown.ScopeKind)
	}

	invalid := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID:            "t",
		TreeKind:          "primary",
		ScopeKind:         "explicit",
		SharedCountryTags: []string{"TOOLONGTAG"},
	}, nil, nil, "")
	if !invalid.Blocked() || !codesOf(invalid)["focus_scope_invalid_country_tag"] {
		t.Fatalf("invalid tag must block: %+v", invalid)
	}

	missing := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID: "t", TreeKind: "primary", ScopeKind: "explicit",
	}, nil, nil, "")
	if !codesOf(missing)["focus_scope_explicit_missing_countries"] {
		t.Fatalf("explicit without countries must block: %+v", missing)
	}
	ge := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID: "t", TreeKind: "primary", ScopeKind: "global_except",
	}, nil, nil, "")
	if !codesOf(ge)["focus_scope_global_except_missing_countries"] {
		t.Fatalf("global_except without countries must block: %+v", ge)
	}
}

func TestFocusCustodyTagNormalisation(t *testing.T) {
	m := MaterializeFocusBundle(FocusScopeCandidate{
		TreeID:            "t",
		TreeKind:          "primary",
		ScopeKind:         "explicit",
		SharedCountryTags: []string{" fin ", "FIN", "swe", ""},
	}, nil, nil, "")
	if m.Blocked() {
		t.Fatalf("normalised tags must pass: %+v", m.Findings)
	}
	if len(m.SharedCountryTags) != 2 || m.SharedCountryTags[0] != "FIN" || m.SharedCountryTags[1] != "SWE" {
		t.Fatalf("tags must upper+trim+dedupe preserving order: %v", m.SharedCountryTags)
	}
}
