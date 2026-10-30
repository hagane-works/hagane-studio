package worldout

import (
	"encoding/json"
	"strings"
	"testing"

	"haganeworks/internal/reftables"
)

func managedRegistry() *OMap {
	root := NewOMap()
	root.Set("id", "technocracy")
	root.Set("body", "types = {\n}\ncolor = { 35 75 120 }")
	rootName := NewOMap()
	rootName.Set("en", "Technocracy")
	rootName.Set("zh", "技术统治")
	root.Set("name", rootName)
	rootDesc := NewOMap()
	rootDesc.Set("en", "Technocratic regime")
	rootDesc.Set("zh", "技术统治政体")
	root.Set("desc", rootDesc)

	sub := NewOMap()
	sub.Set("id", "cybernetics")
	sub.Set("group", "technocracy")
	sub.Set("name", NewOMap())

	registry := NewOMap()
	registry.Set("root_definitions", []Value{root})
	registry.Set("sub_ideologies", []Value{sub})
	return registry
}

func TestProjectManagedCustomRootFullReplacement(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatal(err)
	}
	registry := managedRegistry()
	normalized := normalizeRegistry(registry)
	if len(normalized.roots) != 1 || normalized.roots[0].id != "technocracy" {
		t.Fatalf("custom ROOT must survive registry normalization: %#v", normalized.roots)
	}

	full := generateFullIdeologiesFile(registry, tables.IdeologyBaseText)
	if full == "" {
		t.Fatal("project-managed custom ROOT must produce the complete ideologies replacement")
	}
	for _, root := range []string{"democratic", "communism", "fascism", "neutrality", "technocracy"} {
		if got := strings.Count(full, "\t"+root+" = {"); got != 1 {
			t.Errorf("ROOT %q definitions = %d, want exactly one", root, got)
		}
	}
	for _, want := range []string{
		"\ttechnocracy = {", "\t\tcybernetics = {", "color = { 35 75 120 }",
	} {
		if !strings.Contains(full, want) {
			t.Errorf("complete replacement missing %q", want)
		}
	}
	block, ok := extractVanillaGroupBlock(full, "technocracy")
	if !ok || !typeIDsInGroupBlock(block, "technocracy")["cybernetics"] {
		t.Fatalf("custom subtype is not nested inside the custom ROOT types block:\n%s", block)
	}
}

func TestInlineEmptyTypesReceivesSubtypeInsideTypes(t *testing.T) {
	block := "\ttechnocracy = {\n\t\ttypes = { }\n\t\tcolor = { 42 110 145 }\n\t}"
	sub := NewOMap()
	sub.Set("id", "civic_technocracy")
	sub.Set("random_disabled", true)
	injected, ok := injectSubsIntoTypes(block, customSubsText([]*OMap{sub}))
	if !ok || !typeIDsInGroupBlock(injected, "technocracy")["civic_technocracy"] {
		t.Fatalf("inline types injection escaped its types block:\n%s", injected)
	}
	if strings.Contains(injected, "\n\t\tcivic_technocracy = {") {
		t.Fatalf("subtype became a direct ROOT child:\n%s", injected)
	}
}

func TestCustomSubtypeRandomSelectionFollowsRegistry(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatal(err)
	}
	registry := managedRegistry()
	allowed := registry.m["sub_ideologies"].([]Value)[0].(*OMap)
	allowed.Set("random_disabled", false)
	disabled := NewOMap()
	disabled.Set("id", "closed_cybernetics")
	disabled.Set("group", "technocracy")
	disabled.Set("random_disabled", true)
	registry.Set("sub_ideologies", []Value{allowed, disabled})

	full := generateFullIdeologiesFile(registry, tables.IdeologyBaseText)
	allowedBlock := "\t\t\tcybernetics = {\n\t\t\t}\n"
	if !strings.Contains(full, allowedBlock) {
		t.Fatalf("random-enabled subtype was not emitted as an empty selectable block:\n%s", full)
	}
	disabledBlock := "\t\t\tclosed_cybernetics = {\n\t\t\t\tcan_be_randomly_selected = no\n\t\t\t}\n"
	if !strings.Contains(full, disabledBlock) {
		t.Fatalf("random-disabled subtype lost its engine marker:\n%s", full)
	}
}

func TestProjectManagedPoliticalReferencesCloseOverRootAndSubtype(t *testing.T) {
	registry := managedRegistry()
	if got, ok := resolveRulingPartyGroup(registry, false, "cybernetics"); !ok || got != "technocracy" {
		t.Fatalf("subtype ruling_party must resolve to custom ROOT, got %q ok=%v", got, ok)
	}
	if got, ok := resolveRulingPartyGroup(registry, false, "technocracy"); !ok || got != "technocracy" {
		t.Fatalf("ROOT ruling_party must remain stable, got %q ok=%v", got, ok)
	}
	if _, ok := resolveRulingPartyGroup(registry, true, "technocracy"); ok {
		t.Fatal("base-mod export must not write or take over a custom ROOT")
	}

	leaders := emitHistoryFileLeaderReplacementBlock("TCH", []*OMap{func() *OMap {
		leader := NewOMap()
		leader.Set("char_tag", "TCH_director")
		leader.Set("ideology", "cybernetics")
		return leader
	}()})
	if !strings.Contains(strings.Join(leaders, "\n"), "\tideology = cybernetics") {
		t.Fatal("country leader must retain the subtype rather than rewriting it to the ROOT")
	}
}

func TestEmitScenarioClosesProjectManagedCountryPolitics(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatal(err)
	}
	project := NewOMap()
	project.Set("ideology_registry", managedRegistry())
	project.Set("export_languages", []Value{"en"})
	country := NewOMap()
	country.Set("tag", "TCH")
	country.Set("ideology", "cybernetics")
	leader := NewOMap()
	leader.Set("char_tag", "TCH_director")
	leader.Set("ideology", "cybernetics")
	leaders := NewOMap()
	leaders.Set("TCH", []Value{leader})
	scenario := &Scenario{
		Project: project, ModName: "Test Mod", ModDirName: "test_mod",
		Countries: []*OMap{country}, ClByCtag: leaders,
	}
	records, _ := EmitScenario(tables, scenario)
	joined := ""
	for _, record := range records {
		joined += "\n-- " + record.Path + "\n" + record.Text
	}
	for _, want := range []string{
		"common/ideologies/00_ideologies.txt", "\ttechnocracy = {",
		"\t\tcybernetics = {", "ruling_party = technocracy",
		"ideology = cybernetics", `technocracy:0 "Technocracy"`,
		`technocracy_noun:0 "Technocracy"`,
		`technocracy_desc:0 "Technocratic regime"`,
		"portraits/zz_test_mod_political.txt", "TCH = {",
		"\t\ttechnocracy = {", `"GFX_Portrait_Europe_Generic_2"`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("project-managed country export missing %q:\n%s", want, joined)
		}
	}
}

func TestCustomRootPoliticalPortraitsAreTagScopedAndDeterministic(t *testing.T) {
	registry := managedRegistry()
	got := buildCustomRootPoliticalPortraits(
		[]string{"tch", "TCH", " B02 ", "bad-tag"},
		customRootDefinitions(registry),
	)
	for _, want := range []string{
		"B02 = {", "TCH = {", "\t\ttechnocracy = {",
		`"GFX_Portrait_Europe_Generic_2"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("political portrait fallback missing %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "TCH = {") != 1 || strings.Contains(got, "bad-tag") ||
		strings.Index(got, "B02 = {") > strings.Index(got, "TCH = {") {
		t.Fatalf("portrait fallback must deduplicate, validate, and sort tags:\n%s", got)
	}
}

func TestUnsafeCustomRootAndBaseModTakeoverAreNotEmitted(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatal(err)
	}
	unsafe := managedRegistry()
	root := unsafe.m["root_definitions"].([]Value)[0].(*OMap)
	root.Set("body", "types = { }\ncolor = { 1 2 3 }\n}\nevil = {")
	if got := generateFullIdeologiesFile(unsafe, tables.IdeologyBaseText); got != "" {
		t.Fatal("an escaping ROOT body must never be concatenated into export output")
	}

	project := NewOMap()
	project.Set("ideology_registry", managedRegistry())
	scenario := &Scenario{Project: project, ModDirName: "base_mod", IsSubmod: true}
	records, _ := EmitScenario(tables, scenario)
	for _, record := range records {
		if strings.Contains(record.Path, "common/ideologies/") {
			t.Fatalf("base-mod export must inherit ideologies, not emit ROOT takeover: %s", record.Path)
		}
	}
}

func TestHistoryPopularityLinesPreserveCustomRootMap(t *testing.T) {
	history := NewOMap()
	popularities := NewOMap()
	popularities.Set("technocracy", json.Number("60"))
	popularities.Set("democratic", json.Number("40"))
	history.Set("set_popularities", popularities)

	got := strings.Join(historyPopularityLines(history), "\n")
	want := "\tdemocratic = 40\n\ttechnocracy = 60"
	if got != want {
		t.Fatalf("custom ROOT popularity map changed during export:\n got %q\nwant %q", got, want)
	}
}

func TestCustomRootIdeologiesFileBesideInheritedSource(t *testing.T) {
	registry := managedRegistry()
	text, kept := CustomRootIdeologiesFile(registry, map[string]bool{})
	if len(kept) != 1 || pyStrOrEmpty(eGet(kept[0], "id")) != "technocracy" {
		t.Fatalf("kept ROOTs = %v", kept)
	}
	for _, want := range []string{"ideologies = {\n\ttechnocracy = {", "\t\tcybernetics = {", "color = { 35 75 120 }"} {
		if !strings.Contains(text, want) {
			t.Errorf("ROOT file missing %q:\n%s", want, text)
		}
	}
	for _, vanilla := range []string{"democratic", "communism", "fascism", "neutrality"} {
		if strings.Contains(text, "\t"+vanilla+" = {") {
			t.Errorf("ROOT file redefines the source's %s", vanilla)
		}
	}
	if text, kept := CustomRootIdeologiesFile(registry, map[string]bool{"technocracy": true}); text != "" || kept != nil {
		t.Errorf("a ROOT the source defines was written again:\n%s", text)
	}
	if text, _ := CustomRootIdeologiesFile(NewOMap(), nil); text != "" {
		t.Errorf("no ROOT, still a file:\n%s", text)
	}
}

func TestIdeologyExportWithAnEmptyTable(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatal(err)
	}
	defer reftables.Use(map[string][]byte{reftables.IdeologiesFile: []byte(`{"groups": [], "sub_ideologies": []}`)})()
	registry := managedRegistry()
	if merged := mergeWithVanilla(registry); len(merged.groups) != 1 || len(merged.subs) != 1 {
		t.Fatalf("merged = %v groups, %d subs", merged.groups, len(merged.subs))
	}
	if got, ok := resolveRulingPartyGroup(registry, false, "cybernetics"); !ok || got != "technocracy" {
		t.Fatalf("project subtype = %q %v", got, ok)
	}
	for _, vanilla := range []string{"democratic", "conservatism"} {
		if got, ok := resolveRulingPartyGroup(registry, false, vanilla); ok {
			t.Errorf("%s resolved to %q without the table", vanilla, got)
		}
	}
	full := generateFullIdeologiesFile(registry, tables.IdeologyBaseText)
	if strings.Count(full, "\ttechnocracy = {") != 1 || !strings.Contains(full, "\t\tcybernetics = {") {
		t.Fatalf("the project's ROOT was not exported:\n%s", full)
	}
}
