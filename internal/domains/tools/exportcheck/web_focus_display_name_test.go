package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFocusDisplayNamePolicyParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "displayname_policy_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name  string `json:"name"`
			Input struct {
				FocusID               string         `json:"focus_id"`
				Source                string         `json:"source"`
				DisplayNameLocs       map[string]any `json:"display_name_locs"`
				NameEN                string         `json:"name_en"`
				NameZH                string         `json:"name_zh"`
				EnabledContentLocales []string       `json:"enabled_content_locales"`
				IsNewID               bool           `json:"is_new_id"`
				ReusesBaseID          bool           `json:"reuses_base_id"`
				MaterialNameChanged   bool           `json:"material_name_changed"`
				FallbackProviders     []struct {
					Locale       string `json:"locale"`
					Key          string `json:"key"`
					ProviderKind string `json:"provider_kind"`
					Path         string `json:"path"`
					ContentHash  string `json:"content_hash"`
				} `json:"fallback_providers"`
				SeverityMap   map[string]string `json:"severity_map"`
				SeverityFixed *string           `json:"severity_fixed"`
				SourceName    *string           `json:"source_name"`
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
		in := FocusDisplayNameInput{
			FocusID:               c.Input.FocusID,
			Source:                c.Input.Source,
			DisplayNameLocs:       c.Input.DisplayNameLocs,
			NameEN:                c.Input.NameEN,
			NameZH:                c.Input.NameZH,
			EnabledContentLocales: c.Input.EnabledContentLocales,
			IsNewID:               c.Input.IsNewID,
			ReusesBaseID:          c.Input.ReusesBaseID,
			MaterialNameChanged:   c.Input.MaterialNameChanged,
			SourceName:            c.Input.SourceName,
		}

		in.MissingLocaleSeverity = DisplayNameSeverityPolicy{Map: c.Input.SeverityMap}
		if c.Input.SeverityFixed != nil {
			in.MissingLocaleSeverity = DisplayNameSeverityPolicy{Fixed: *c.Input.SeverityFixed}
		}
		for _, p := range c.Input.FallbackProviders {
			in.FallbackProviders = append(in.FallbackProviders, DisplayNameFallbackProvider{
				Locale: p.Locale, Key: p.Key, ProviderKind: p.ProviderKind,
				Path: p.Path, ContentHash: p.ContentHash,
			})
		}

		got := jsonNorm(t, AnalyzeFocusDisplayName(in).ToDict())
		var want any
		if err := json.Unmarshal(c.Expected, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("case %s\n got %s\nwant %s", c.Name, gb, c.Expected)
		}
	}
}

func TestDisplayNameProviderProven(t *testing.T) {
	full := DisplayNameFallbackProvider{
		Locale: "en", Key: "f", ProviderKind: "vanilla", Path: "p", ContentHash: "h"}
	if !full.Proven() {
		t.Error("四字段齐全的 vanilla 提供方应当算被证明")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*DisplayNameFallbackProvider)
	}{
		{"kind=created", func(p *DisplayNameFallbackProvider) { p.ProviderKind = "created" }},
		{"no-locale", func(p *DisplayNameFallbackProvider) { p.Locale = "" }},
		{"no-key", func(p *DisplayNameFallbackProvider) { p.Key = "" }},
		{"no-path", func(p *DisplayNameFallbackProvider) { p.Path = "" }},
		{"no-hash", func(p *DisplayNameFallbackProvider) { p.ContentHash = "" }},
	} {
		p := full
		tc.mutate(&p)
		if p.Proven() {
			t.Errorf("%s 不该算被证明", tc.name)
		}
	}
	base := full
	base.ProviderKind = "base_mod"
	if !base.Proven() {
		t.Error("base_mod 也能作证")
	}
}
