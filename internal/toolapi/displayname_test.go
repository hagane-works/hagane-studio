package toolapi

import (
	"context"
	"testing"
)

func TestResolveDisplayNameMirrorsTheFrontendChain(t *testing.T) {
	locs := map[string]any{"en": "Reform the Army", "ja": "軍制改革", "zh": "整军经武"}

	cases := []struct {
		name         string
		locale       string
		locs         map[string]any
		legacyEn     string
		legacyZh     string
		fallback     string
		wantName     string
		wantResolved string
	}{
		{"active locale wins", "ja", locs, "", "", "fb", "軍制改革", "ja"},
		{"default is english", "en", locs, "", "", "fb", "Reform the Army", "en"},
		{"unknown locale falls back to english", "de", locs, "", "", "fb", "Reform the Army", "en"},
		{"zh reads the zh entry", "zh", locs, "", "", "fb", "整军经武", "zh"},
		{"legacy en column when no map entry", "en", map[string]any{}, "Legacy Name", "旧名", "fb", "Legacy Name", "en"},
		{"legacy zh column only for zh", "zh", map[string]any{}, "Legacy Name", "旧名", "fb", "旧名", "zh"},
		{"whitespace-only map entry counts as missing", "en", map[string]any{"en": "   "}, "Legacy Name", "", "fb", "Legacy Name", "en"},
		{"nothing at all gives the fallback and an empty locale", "en", map[string]any{}, "", "", "han_se.1100", "han_se.1100", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, resolved := resolveDisplayName(tc.locale, tc.locs, tc.legacyEn, tc.legacyZh, tc.fallback)
			if got != tc.wantName || resolved != tc.wantResolved {
				t.Errorf("= (%q, %q), want (%q, %q)", got, resolved, tc.wantName, tc.wantResolved)
			}
		})
	}
}

func TestChineseLegacyTextNeverLeaksIntoOtherLocales(t *testing.T) {
	for _, locale := range []string{"en", "de", "fr", "ja"} {
		got, resolved := resolveDisplayName(locale, map[string]any{}, "", "只有中文", "fallback_id")
		if got != "fallback_id" || resolved != "" {
			t.Errorf("locale %q leaked Chinese legacy text: (%q, %q)", locale, got, resolved)
		}
	}
}

func TestLegacyTitleColumnsAreTreatedAsNameColumns(t *testing.T) {
	entity := map[string]any{
		"display_name_locs": map[string]any{},
		"title_en":          "The Fall of Luoyang",
		"title_zh":          "洛阳陷落",
	}
	if !enrichDisplayName(entity, "en") {
		t.Fatal("entity with a locs map was not enriched")
	}
	if entity["display_name"] != "The Fall of Luoyang" {
		t.Errorf("display_name = %v", entity["display_name"])
	}
}

func TestNonEntityResponsesAreLeftAlone(t *testing.T) {
	resp := map[string]any{"job_id": "1786067597998820689-1", "status": "complete", "total": 3}
	enrichDisplayNames(resp, "en")
	if _, ok := resp["display_name"]; ok {
		t.Fatalf("a non-entity response was enriched: %v", resp)
	}
}

func TestUpstreamDisplayNameIsNotOverwritten(t *testing.T) {
	entity := map[string]any{
		"display_name_locs": map[string]any{"en": "Reconstructed"},
		"display_name":      "Authoritative",
	}
	enrichDisplayName(entity, "en")
	if entity["display_name"] != "Authoritative" {
		t.Fatalf("display_name = %v, want the upstream value", entity["display_name"])
	}
}

func TestEnrichmentFindsEntitiesAtAnyDepth(t *testing.T) {
	row := func(en string) map[string]any {
		return map[string]any{"display_name_locs": map[string]any{"en": en}}
	}
	bare := []any{row("A"), row("B")}
	enrichDisplayNames(bare, "en")
	for i, item := range bare {
		if item.(map[string]any)["display_name"] == nil {
			t.Errorf("bare list item %d not enriched", i)
		}
	}

	wrapped := map[string]any{"characters": []any{row("C")}, "total": 1}
	enrichDisplayNames(wrapped, "en")
	first := wrapped["characters"].([]any)[0].(map[string]any)
	if first["display_name"] != "C" {
		t.Errorf("nested list item not enriched: %v", first)
	}
}

func TestResolvedLocaleIsAlwaysStated(t *testing.T) {
	entity := map[string]any{"display_name_locs": map[string]any{}, "tag": "HAN"}
	enrichDisplayName(entity, "en")
	resolved, ok := entity["resolved_locale"]
	if !ok {
		t.Fatal("resolved_locale missing on an unresolved entity")
	}
	if resolved != "" {
		t.Errorf("resolved_locale = %v, want empty (nothing resolved; display_name is the fallback)", resolved)
	}
	if entity["display_name"] != "HAN" {
		t.Errorf("display_name = %v, want the game-script fallback", entity["display_name"])
	}
}

func TestContentLocaleDefaultsToEnglish(t *testing.T) {
	if got := contentLocaleFrom(context.Background()); got != "en" {
		t.Errorf("default locale = %q, want en (the frontend's own baseline)", got)
	}
	ctx := withContentLocale(context.Background(), "  JA ")
	if got := contentLocaleFrom(ctx); got != "ja" {
		t.Errorf("locale = %q, want ja (trimmed and lowercased)", got)
	}
	if got := contentLocaleFrom(withContentLocale(context.Background(), "")); got != "en" {
		t.Errorf("empty locale must not clear the default, got %q", got)
	}
}
