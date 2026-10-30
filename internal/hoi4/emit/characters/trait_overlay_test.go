package characters

import (
	"encoding/json"
	"strings"
	"testing"
)

func overlayTestLanguages() []LocLanguage {
	return []LocLanguage{
		{Code: "en", Subdir: "english/replace", Suffix: "l_english"},
		{Code: "zh", Subdir: "simp_chinese/replace", Suffix: "l_simp_chinese"},
	}
}

func TestTraitOverlayManifestComesFromThreeReplacementFamilies(t *testing.T) {
	cl := &Character{
		ID: 1, Tag: "GER_repl_leader", CountryTag: "GER", RoleType: "country_leader",
		NameEn: "Replacement Leader", NameZh: "替换领袖", Ideology: "fascism",
		Traits: []string{"custom_cl"}, ReplacesVanillaID: true, IsPortraitModified: true,
	}
	ul := &Character{
		ID: 2, Tag: "GER_repl_general", CountryTag: "GER", RoleType: "general",
		NameEn: "Replacement General", Traits: []string{"custom_ul"},
		ReplacesVanillaID: true, IsPortraitModified: true,
	}
	scientist := &Character{
		ID: 3, Tag: "GER_repl_scientist", CountryTag: "GER", RoleType: "scientist",
		NameEn: "Replacement Scientist", Traits: []string{"custom_sc"},
		ReplacesVanillaID: true,
	}
	in := &Input{
		Tag:                       "GER",
		Characters:                []*Character{cl, ul, scientist},
		ReplacementCountryLeaders: []*Character{cl},
		ReplacementUnitLeaders:    []*Character{ul},
		ReplacementScientists:     []*Character{scientist},
		CustomTraits: []CustomTrait{
			{TraitID: "custom_cl", TraitType: "country_leader", CountryTag: "GER"},
			{TraitID: "custom_ul", TraitType: "unit_leader", CountryTag: "GER"},
			{TraitID: "custom_sc", TraitType: "scientist", CountryTag: "GER"},
		},
		ExportLanguages: overlayTestLanguages(),
		PortraitObs: func(id int) string {
			if id == 1 || id == 2 {
				return "obs/portrait"
			}
			return ""
		},
		AdvisorObs:    func(int) string { return "" },
		VanillaTraits: map[string]bool{},
	}

	result := EmitTraitOverlayArtifacts(in)
	bindings := result.TraitOverlayNamespaces()
	if bindings["GER"] != "ger_trait_overlay" {
		t.Fatalf("overlay binding = %q, want ger_trait_overlay; artifacts=%+v", bindings["GER"], result.TraitOverlays["GER"])
	}

	artifacts := result.TraitOverlays["GER"]
	for _, path := range []string{
		"events/GER_trait_overlay.txt",
		"interface/zz_GER_replacement_portraits.gfx",
		"localisation/english/replace/GER_characters_replacement_overrides_l_english.yml",
		"localisation/simp_chinese/replace/GER_characters_replacement_overrides_l_simp_chinese.yml",
		"localisation/english/replace/GER_trait_overlay_l_english.yml",
		"localisation/simp_chinese/replace/GER_trait_overlay_l_simp_chinese.yml",
	} {
		if _, ok := result.Texts[path]; !ok {
			t.Errorf("missing emitted overlay-family path %q; artifacts=%+v", path, artifacts)
		}
	}
	event := result.Texts[artifacts.EventPath]
	for _, want := range []string{
		"create_country_leader = {",
		"add_unit_leader_trait = custom_ul",
		"add_scientist_trait = custom_sc",
		`picture = "GFX_portrait_GER_repl_leader"`,
	} {
		if !strings.Contains(event, want) {
			t.Errorf("event lacks %q:\n%s", want, event)
		}
	}
	if !strings.HasPrefix(result.Texts[artifacts.EventLocalisationPaths[0]], bomUTF8) {
		t.Fatal("trait overlay localisation is not UTF-8 BOM prefixed")
	}
	if !strings.Contains(result.Texts[artifacts.ReplacementLocalisationPaths[0]], `GER_repl_leader: "Replacement Leader"`) {
		t.Fatal("replacement name override is missing the country leader name")
	}
}

func TestTraitOverlayOptOutCannotCreateOnActionsBinding(t *testing.T) {
	char := &Character{
		ID: 7, Tag: "GER_optout", CountryTag: "GER", RoleType: "general",
		NameEn: "Author Managed", Traits: []string{"custom_ul"}, ReplacesVanillaID: true,
		RawExtras: []KV{{Key: "emit_trait_overlay_event", Val: false}},
	}
	result := EmitTraitOverlayArtifacts(&Input{
		Tag:                    "GER",
		ReplacementUnitLeaders: []*Character{char},
		CustomTraits:           []CustomTrait{{TraitID: "custom_ul", TraitType: "unit_leader", CountryTag: "GER"}},
		ExportLanguages:        overlayTestLanguages(),
		PortraitObs:            func(int) string { return "" },
		AdvisorObs:             func(int) string { return "" },
		VanillaTraits:          map[string]bool{},
	})

	if got := result.TraitOverlayNamespaces(); len(got) != 0 {
		t.Fatalf("opted-out character produced on_actions bindings: %v", got)
	}
	if _, ok := result.Texts["events/GER_trait_overlay.txt"]; ok {
		t.Fatal("opted-out character produced a trait-overlay event")
	}

	if _, ok := result.Texts["localisation/english/replace/GER_characters_replacement_overrides_l_english.yml"]; !ok {
		t.Fatal("event opt-out incorrectly suppressed replacement localisation")
	}
}

func TestTraitOverlayNamespaceCollisionIsReflectedInManifest(t *testing.T) {
	char := &Character{
		ID: 8, Tag: "GER_general", CountryTag: "GER", RoleType: "general",
		Traits: []string{"custom_ul"}, ReplacesVanillaID: true,
	}
	result := EmitTraitOverlayArtifacts(&Input{
		Tag:                    "GER",
		ReplacementUnitLeaders: []*Character{char},
		CustomTraits:           []CustomTrait{{TraitID: "custom_ul", TraitType: "unit_leader", CountryTag: "GER"}},
		EventNamespaces:        []EventNamespace{{Namespace: "GER_TRAIT_OVERLAY", ID: 1}},
		PortraitObs:            func(int) string { return "" },
		AdvisorObs:             func(int) string { return "" },
		VanillaTraits:          map[string]bool{},
	})
	if got := result.TraitOverlayNamespaces()["GER"]; got != "ger_trait_overlay_2" {
		t.Fatalf("collision-resolved binding = %q, want ger_trait_overlay_2", got)
	}
}

func TestEmitTraitOverlayRowsHonoursImportedReplacementFilter(t *testing.T) {
	rows := TraitOverlayRows{
		Project: json.RawMessage(`{"country_tag":"GER","export_languages":["zh"]}`),
		Characters: []json.RawMessage{
			json.RawMessage(`{"id":10,"tag":"GER_imported_general","name_en":"Imported General","name_zh":"导入将领","country_tag":"GER","role_type":"general","traits":["custom_ul"],"raw_extras":{},"source":"imported","replaces_vanilla_id":true,"is_portrait_modified":true}`),
		},
		CustomTraits: []json.RawMessage{
			json.RawMessage(`{"id":11,"trait_id":"custom_ul","trait_type":"unit_leader","country_tag":"GER"}`),
		},
		EntityOBSPaths: []json.RawMessage{
			json.RawMessage(`{"entity_type":"character_portrait","entity_id":10,"obs_key":"obs/imported.png"}`),
		},
		OBSAvailable: func(string) (bool, error) { return true, nil },
	}
	excluded, err := EmitTraitOverlayRows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(excluded.TraitOverlayNamespaces()) != 0 || len(excluded.Texts) != 0 {
		t.Fatalf("include_imported=false leaked imported replacement artifacts: %+v", excluded.Texts)
	}

	rows.IncludeImported = true
	result, err := EmitTraitOverlayRows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.TraitOverlayNamespaces()["GER"]; got != "ger_trait_overlay" {
		t.Fatalf("imported replacement binding = %q, want ger_trait_overlay", got)
	}
	if _, ok := result.Texts["interface/zz_GER_replacement_portraits.gfx"]; !ok {
		t.Fatal("imported replacement portrait registration was not emitted")
	}
	if _, ok := result.Texts["localisation/simp_chinese/replace/GER_characters_replacement_overrides_l_simp_chinese.yml"]; !ok {
		t.Fatal("imported replacement localisation was not emitted")
	}
	if _, ok := result.Texts["localisation/english/replace/GER_trait_overlay_l_english.yml"]; !ok {
		t.Fatal("EnsureEnFirst contract was not applied to overlay localisation")
	}
}
