package exportengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/cwtools"
)

func useCWTools(t *testing.T) {
	t.Helper()
	dir, _ := filepath.Abs(filepath.Join("..", "..", "..", filepath.FromSlash(cwtools.Folder)))
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(cwtools.ModifiersFile))); err != nil {
		t.Skipf("no CWTools rules: %v", err)
	}
	cwtools.SetDir(dir)
	t.Cleanup(func() { cwtools.SetDir("") })
}

func registryEntry(rel, text string) Entry {
	return Entry{Path: "Demo/" + rel, Bytes: []byte(text), Origin: "test"}
}

func registryArtifact(files ...Entry) []Entry {
	base := []Entry{{Path: "Demo/descriptor.mod", Bytes: []byte(`name="Demo"`), Origin: "test"}}
	return append(base, files...)
}

func registryFindingCodes(report RegistryReport) string {
	var out []string
	for _, finding := range report.Findings {
		out = append(out, finding.Code+":"+finding.Token)
	}
	return strings.Join(out, "\n")
}

func TestRegistryValidationModifierUsesPinnedCWToolsAndModDefinitions(t *testing.T) {
	useCWTools(t)
	report := ScanRegistryValidation(registryArtifact(
		registryEntry("common/modifier_definitions/custom.txt", "my_mod_bonus = { color_type = good }\n"),
		registryEntry("common/ideas/test.txt", "ideas = { country = { good = { modifier = { political_power_factor = 0.1 democratic_drift = 0.1 my_mod_bonus = 0.2 } } bad = { modifier = { skill_card_taught_invalid = 1 } } } }\n"),

		registryEntry("common/dynamic_modifiers/test.txt", "good_dynamic = { icon = GFX_x political_power_factor = 0.1 }\nbad_dynamic = { direct_skill_card_taught_invalid = 1 }\n"),
	), english)
	got := registryFindingCodes(report)
	if !strings.Contains(got, "modifier.unknown:skill_card_taught_invalid") {
		t.Fatalf("unknown modifier was accepted:\n%s", got)
	}
	if !strings.Contains(got, "modifier.unknown:direct_skill_card_taught_invalid") {
		t.Fatalf("unknown direct dynamic modifier was accepted:\n%s", got)
	}
	for _, valid := range []string{"political_power_factor", "democratic_drift", "my_mod_bonus"} {
		if strings.Contains(got, "modifier.unknown:"+valid) {
			t.Fatalf("valid modifier %q was rejected:\n%s", valid, got)
		}
	}
	if report.Catalog.Source.Repository != "https://github.com/cwtools/cwtools-hoi4-config" || report.Catalog.Source.License != "MIT" ||
		(len(report.Catalog.Source.Commit) != 40 && report.Catalog.Source.Commit != "unknown") {
		t.Fatalf("catalog provenance missing: %+v", report.Catalog.Source)
	}
	if log := string(RegistryValidationLogEntry("Demo", report).Bytes); !strings.Contains(log, "cwtools/cwtools-hoi4-config "+report.Catalog.Source.Commit) {
		t.Fatalf("review log does not name the rules' version:\n%s", log)
	}
}

func TestRegistryValidationWithoutCWToolsSkipsOnlyModifierNames(t *testing.T) {
	cwtools.SetDir(t.TempDir())
	t.Cleanup(func() { cwtools.SetDir("") })
	if _, err := cwtools.Load(); err == nil {
		t.Skip("CWTools rules found beside the test binary or in the working folder")
	}
	report := ScanRegistryValidation(registryArtifact(
		registryEntry("common/ideas/test.txt", "ideas = { country = { x = { modifier = { skill_card_taught_invalid = 1 } } } }\n"),
		registryEntry("events/test.txt", "add_namespace = demo\ncountry_event = { id = 事件.2 }\n"),
	), english)
	got := registryFindingCodes(report)
	if strings.Contains(got, "modifier.") || !strings.Contains(got, "script_id.non_ascii:事件.2") {
		t.Fatalf("findings without CWTools:\n%s", got)
	}
	log := string(RegistryValidationLogEntry("Demo", report).Bytes)
	if !strings.Contains(log, "were not found, so modifier names were not checked") || !strings.Contains(log, cwtools.Repository) || strings.Contains(log, "were checked against") {
		t.Fatalf("review log does not say the rules are missing:\n%s", log)
	}
}

func TestRegistryValidationIdeaCategoryAndSlotVanillaUnionModIdeaTags(t *testing.T) {
	report := ScanRegistryValidation(registryArtifact(
		registryEntry("common/idea_tags/custom.txt", "idea_categories = { custom_category = { slot = custom_slot } }\n"),
		registryEntry("common/ideas/test.txt", "ideas = { country = { vanilla = { } } custom_category = { modded = { } } missing_category = { bad = { } } }\n"),
		registryEntry("common/characters/test.txt", "characters = { A = { advisor = { slot = political_advisor } } B = { advisor = { slot = custom_slot } } C = { advisor = { slot = absent_slot } } }\n"),
	), english)
	got := registryFindingCodes(report)
	for _, want := range []string{"idea_category.source_unregistered:missing_category", "idea_slot.source_unregistered:absent_slot"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"idea_category.source_unregistered:country", "idea_category.source_unregistered:custom_category", "idea_slot.source_unregistered:political_advisor", "idea_slot.source_unregistered:custom_slot"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("union rejected %s:\n%s", unwanted, got)
		}
	}
}

func TestRegistryValidationScriptIDASCIIPositiveAndNegative(t *testing.T) {
	report := ScanRegistryValidation(registryArtifact(
		registryEntry("common/decisions/categories/test.txt", "valid_category = { icon = generic }\n中文分类 = { icon = generic }\n"),
		registryEntry("events/test.txt", "add_namespace = demo\ncountry_event = { id = demo.1 }\ncountry_event = { id = 事件.2 }\n"),
	), english)
	got := registryFindingCodes(report)
	for _, want := range []string{"script_id.non_ascii:中文分类", "script_id.non_ascii:事件.2"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"script_id.non_ascii:valid_category", "script_id.non_ascii:demo.1"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("ASCII id rejected %s:\n%s", unwanted, got)
		}
	}
}

func TestRegistryValidationCountryNeedsAllThreeFlagSizes(t *testing.T) {
	report := ScanRegistryValidation(registryArtifact(
		registryEntry("common/country_tags/tags.txt", "dynamic_tags = yes\nAAA = \"countries/AAA.txt\"\nBBB = \"countries/BBB.txt\"\n"),
		registryEntry("gfx/flags/AAA.tga", "root"),
		registryEntry("gfx/flags/medium/AAA.tga", "medium"),
		registryEntry("gfx/flags/small/AAA.tga", "small"),
		registryEntry("gfx/flags/BBB.tga", "root"),
	), english)
	got := registryFindingCodes(report)
	if strings.Contains(got, "country_flag.source_unregistered:AAA") {
		t.Fatalf("complete flag set rejected:\n%s", got)
	}
	if !strings.Contains(got, "country_flag.source_unregistered:BBB") {
		t.Fatalf("missing sizes not reported:\n%s", got)
	}
	if strings.Contains(got, "country_flag.source_unregistered:DYNAMIC_TAGS") {
		t.Fatalf("country_tags control field treated as a country:\n%s", got)
	}
	log := RegistryValidationLogEntry("Demo", report)
	if !strings.Contains(string(log.Bytes), "Source not registered") || !strings.Contains(string(log.Bytes), "register or import the base mod") {
		t.Fatalf("large-mod uncertainty is not explained:\n%s", log.Bytes)
	}

	zh := ScanRegistryValidation(registryArtifact(
		registryEntry("common/country_tags/tags.txt", "BBB = \"countries/BBB.txt\"\n"),
	), chinese)
	text := string(RegistryValidationLogEntry("Demo", zh).Bytes)
	if !strings.HasPrefix(text, "# "+chinese.Text("registry.log.title", nil)+"\n") ||
		!strings.Contains(text, chinese.Text("registry.status.unregistered", nil)+" · country_flag.source_unregistered · gfx/flags · ") {
		t.Fatalf("Chinese review log:\n%s", text)
	}
}

func TestRegistryValidationCharacterInstanceAllowedMustBeFirst(t *testing.T) {
	report := ScanRegistryValidation(registryArtifact(
		registryEntry("common/characters/test.txt", "characters = {\n good = { instance = { allowed = { always = yes } name = good } }\n bad = { instance = { name = bad allowed = { always = yes } } }\n absent = { instance = { name = absent } }\n}\n"),
	), english)
	got := registryFindingCodes(report)
	if count := strings.Count(got, "character_instance.allowed_not_first:instance"); count != 2 {
		t.Fatalf("want bad+absent only, got %d:\n%s", count, got)
	}
}

func TestRegistryValidationCharacterCollisionReusesExistingFinalArtifactRule(t *testing.T) {

	ids := loadCrossRefVanillaIDs()
	if !ids["character"]["ROM_ion_antonescu"] {
		t.Fatal("existing vanilla character registry is not available")
	}
	raw := registryEntry("common/characters/raw.txt", "characters = { ROM_ion_antonescu = { name = ROM_ion_antonescu } own_id = { name = own_id } }\n")
	report := ScanRegistryValidation(registryArtifact(raw), english)
	got := registryFindingCodes(report)
	if !strings.Contains(got, "character_id.vanilla_collision:ROM_ion_antonescu") {
		t.Fatalf("raw vanilla collision missed:\n%s", got)
	}
	if strings.Contains(got, "character_id.vanilla_collision:own_id") {
		t.Fatalf("own id false-positive:\n%s", got)
	}

	generated := raw
	generated.Origin, generated.SourceEntity = "generated", "characters"
	delegated := registryFindingCodes(ScanRegistryValidation(registryArtifact(generated), english))
	if strings.Contains(delegated, "character_id.vanilla_collision:ROM_ion_antonescu") {
		t.Fatalf("structured collision duplicated instead of delegated:\n%s", delegated)
	}
}

func TestRegistryValidationReportNeverMutatesInput(t *testing.T) {
	entries := registryArtifact(registryEntry("common/ideas/test.txt", "ideas = { country = { x = { modifier = { skill_card_taught_invalid = 1 } } } }\n"))
	want := append([]byte(nil), entries[1].Bytes...)
	_ = ScanRegistryValidation(entries, english)
	if string(entries[1].Bytes) != string(want) {
		t.Fatal("report-only scan mutated artifact bytes")
	}
}
