package countries

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/cosmetic"
	"haganeworks/internal/hoi4/ideologysource"
	"haganeworks/internal/hoi4/politicsrefs"
	"haganeworks/internal/project"
	"haganeworks/internal/reftables"
	"haganeworks/internal/workbench"
)

func TestExtractRelationRefsParity(t *testing.T) {
	raw, err := os.ReadFile("testdata/relations_parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Text string     `json:"text"`
		Refs [][]string `json:"refs"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 30 {
		t.Fatalf("suspiciously few cases: %d", len(cases))
	}
	for i, c := range cases {
		gotList := [][]string{}
		for _, r := range extractRelationRefs(c.Text) {
			gotList = append(gotList, []string{r.effect, r.targetType, r.targetID})
		}
		want := c.Refs
		if want == nil {
			want = [][]string{}
		}
		if !reflect.DeepEqual(gotList, want) {
			t.Errorf("case %d %q:\n got %v\nwant %v", i, c.Text, gotList, want)
		}
	}
}

func TestRelationEdgesForCountry(t *testing.T) {
	ger := "GER"
	edges := []refEdge{
		{SourceKind: "focus", SourceID: 1, TargetID: "GER_target", SourceTags: []string{"GER", "POL"}},
		{SourceKind: "idea", SourceID: 2, TargetID: "usa_thing", SourceTags: []string{"USA"}},
		{SourceKind: "event", SourceID: 3, TargetID: "ger_lower", SourceTags: nil, SourceTag: &ger},
	}
	outgoing, incoming := relationEdgesForCountry(edges, "ger")
	if len(outgoing) != 2 || len(incoming) != 2 {
		t.Errorf("ger: outgoing %d incoming %d, want 2 and 2", len(outgoing), len(incoming))
	}
	if outgoing, _ := relationEdgesForCountry(edges, "usa"); len(outgoing) != 1 {
		t.Errorf("usa outgoing: want 1, got %d", len(outgoing))
	}
}

func TestRelBestAndAllTags(t *testing.T) {
	proj := relProject{countryTag: "ger", countryTags: []string{"pol", "ita"}}
	own := "usa"
	if got := relBestTag(&own, []string{"fra"}, proj); got == nil || *got != "USA" {
		t.Errorf("own priority: %v", got)
	}
	if got := relBestTag(nil, []string{"", "fra"}, proj); got == nil || *got != "FRA" {
		t.Errorf("shared priority: %v", got)
	}
	if got := relBestTag(nil, nil, proj); got == nil || *got != "GER" {
		t.Errorf("project fallback: %v", got)
	}
	if got := relBestTag(nil, nil, relProject{countryTags: []string{"pol", "ita"}}); got == nil || *got != "ITA" {
		t.Errorf("country_tags sorted fallback: %v", got)
	}
	if got := relBestTag(nil, nil, relProject{}); got != nil {
		t.Errorf("nothing attributable: %v", got)
	}
	if tags := relAllTags(&own, []string{"fra", "usa"}, proj); !reflect.DeepEqual(tags, []string{"FRA", "USA"}) {
		t.Errorf("all tags: %v", tags)
	}
	if tags := relAllTags(nil, nil, proj); !reflect.DeepEqual(tags, []string{"GER"}) {
		t.Errorf("fallback tags: %v", tags)
	}
}

func TestNormalizeCountryLocsMirrorsPython(t *testing.T) {
	f := &workbench.CountryLocsFields{
		AdjectiveZh:      "德意志的",
		AdjectiveLocs:    map[string]string{"fr": " Allemand ", "en": ""},
		IdeologyNameLocs: map[string]map[string]string{"fascism": {"de": " Deutsches Reich "}},
		IdeologyScalars: map[string]*[2]string{
			"democratic": {"", ""}, "fascism": {"German Reich", ""}, "communism": {"", ""}, "neutrality": {"", ""},
		},
	}
	workbench.NormalizeCountryLocs(f)
	if f.AdjectiveLocs["zh"] != "德意志的" || f.AdjectiveLocs["fr"] != "Allemand" {
		t.Fatalf("adjective_locs=%v", f.AdjectiveLocs)
	}
	if _, ok := f.AdjectiveLocs["en"]; ok {
		t.Fatalf("empty en must be dropped: %v", f.AdjectiveLocs)
	}
	if fas := f.IdeologyNameLocs["fascism"]; fas["en"] != "German Reich" || fas["de"] != "Deutsches Reich" {
		t.Fatalf("fascism locs=%v", fas)
	}
	if _, ok := f.IdeologyNameLocs["democratic"]; ok {
		t.Fatalf("empty group must stay absent: %v", f.IdeologyNameLocs)
	}
}

func TestCountryTagPattern(t *testing.T) {
	for _, ok := range []string{"GER", "USSR", "X01", "FL2", "WU1", "USSR2"} {
		if !workbench.CountryTagRE.MatchString(ok) {
			t.Fatalf("%s should pass", ok)
		}
	}
	for _, bad := range []string{"GE", "ger", "1AB", "TOOLONG", "G-R", ""} {
		if workbench.CountryTagRE.MatchString(bad) {
			t.Fatalf("%s should fail", bad)
		}
	}
}

func TestCountryTagRewriteMatchesPythonFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/country_lifecycle_parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Shared []struct {
			Input    any    `json:"input"`
			Expected any    `json:"expected"`
			OldTag   string `json:"old_tag"`
			NewTag   string `json:"new_tag"`
			Changed  bool   `json:"changed"`
		} `json:"shared"`
		Bookmarks []struct {
			DefaultCountry              any    `json:"default_country"`
			ExpectedDefaultCountry      any    `json:"expected_default_country"`
			SelectableCountries         any    `json:"selectable_countries"`
			ExpectedSelectableCountries any    `json:"expected_selectable_countries"`
			OldTag                      string `json:"old_tag"`
			NewTag                      string `json:"new_tag"`
			Changed                     bool   `json:"changed"`
		} `json:"bookmarks"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for i, item := range fixture.Shared {
		got, changed := renameTagList(item.Input, item.OldTag, item.NewTag)
		var gotValue any = got
		if !changed {
			gotValue = item.Input
		}
		if changed != item.Changed || !reflect.DeepEqual(gotValue, item.Expected) {
			t.Fatalf("shared[%d]: got=%#v changed=%v, want=%#v changed=%v", i, gotValue, changed, item.Expected, item.Changed)
		}
	}
	for i, item := range fixture.Bookmarks {
		row := project.Row{"default_country": item.DefaultCountry, "selectable_countries": item.SelectableCountries}
		next := renameBookmarkTags(row, item.OldTag, item.NewTag)
		if changed := next != nil; changed != item.Changed {
			t.Fatalf("bookmark[%d]: changed=%v", i, changed)
		}
		if next == nil {
			next = row
		}
		if !reflect.DeepEqual(next["default_country"], item.ExpectedDefaultCountry) ||
			!reflect.DeepEqual(next["selectable_countries"], item.ExpectedSelectableCountries) {
			t.Fatalf("bookmark[%d]: default=%v selectable=%#v", i, next["default_country"], next["selectable_countries"])
		}
	}
}

func TestCountryVanillaTag(t *testing.T) {
	if !workbench.VanillaCountryTag("GER") || workbench.VanillaCountryTag("ZZX") {
		t.Fatal("vanilla tag set")
	}
}

func cosmeticPatchOf(t *testing.T, raw string) cosmeticTagPatch {
	t.Helper()
	var in cosmeticTagPatch
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		t.Fatal(err)
	}
	return in
}

func TestCosmeticPatchValidation(t *testing.T) {
	for _, raw := range []string{
		`{"expected_revision":0}`, `{"expected_revision":1,"color":[1,2]}`,
		`{"expected_revision":1,"color":[1,2,3,4]}`, `{"expected_revision":1,"color":[1,null,3]}`,
		`{"expected_revision":1,"color":[1,2.5,3]}`, `{"expected_revision":1,"color":[1,256,3]}`,
		`{"expected_revision":1,"name_ops":[{"locale":"english","suffix":"","value":"A"}]}`,
		`{"expected_revision":1,"name_ops":[{"locale":"l_english","suffix":"democratic_WRONG","value":"A"}]}`,
		`{"expected_revision":1,"name_ops":[{"locale":"l_english","suffix":"","value":"A\nB"}]}`,
		`{"expected_revision":1,"name_ops":[{"locale":"l_english","suffix":"","value":"A"},{"locale":"l_english","suffix":"","value":"B"}]}`,
	} {
		if cosmeticPatchOf(t, raw).validate() == nil {
			t.Errorf("accepted invalid patch %s", raw)
		}
	}
	for _, raw := range []string{`{"expected_revision":1}`, `{"expected_revision":1,"color":null}`, `{"expected_revision":1,"color":[0,128,255],"name_ops":[{"locale":"l_simp_chinese","suffix":"fascism_DEF","value":"  帝国  "},{"locale":"l_english","suffix":"ADJ","value":null}]}`} {
		if err := cosmeticPatchOf(t, raw).validate(); err != nil {
			t.Errorf("valid patch rejected: %v", err)
		}
	}
}

func TestCosmeticPatchOmittedNullAndIdentity(t *testing.T) {
	for _, raw := range []string{`{"locale":"l_english","suffix":""}`, `{"locale":"l_english","suffix":null,"value":"oops"}`} {
		var op cosmeticNameOp
		if err := json.Unmarshal([]byte(raw), &op); err == nil {
			t.Errorf("incomplete/null operation accepted: %s", raw)
		}
	}
	omitted := cosmeticPatchOf(t, `{"expected_revision":1}`)
	cleared := cosmeticPatchOf(t, `{"expected_revision":1,"color":null}`)
	if len(omitted.Color) != 0 || string(cleared.Color) != "null" {
		t.Fatal("omitted and null collapsed")
	}
	for _, tag := range []string{"ROM_greater_kingdom", "Rom.tag-1", strings.Repeat("a", 100)} {
		if err := cosmetic.ValidateTag(tag); err != nil {
			t.Fatal(err)
		}
	}
	for _, tag := range []string{" ROM", "国", strings.Repeat("a", 101), "ROM/name"} {
		if cosmetic.ValidateTag(tag) == nil {
			t.Fatalf("invalid tag accepted %q", tag)
		}
	}
}

func politicsCommandSnapshot() politicsrefs.Snapshot {
	return politicsrefs.Snapshot{ProjectID: "1", Ideologies: []politicsrefs.Ideology{
		{Kind: politicsrefs.KindRoot, ScriptID: "democratic", Source: politicsrefs.SourceVanilla},
		{Kind: politicsrefs.KindRoot, ScriptID: "civic_coalition", Source: politicsrefs.SourceProjectManaged},
		{Kind: politicsrefs.KindSubtype, ScriptID: "civic_reformism", RootID: "civic_coalition", Source: politicsrefs.SourceProjectManaged},
	}}
}

func politicsCommandRegistry() map[string]any {
	return map[string]any{
		"groups":           []any{"civic_coalition"},
		"root_definitions": []any{map[string]any{"id": "civic_coalition", "body": "types = { civic_reformism = { } }\ncolor = { 1 2 3 }", "name": map[string]any{"en": "Civic Coalition"}}},
		"sub_ideologies":   []any{map[string]any{"id": "civic_reformism", "group": "civic_coalition", "name": map[string]any{"en": "Civic Reformism"}}},
	}
}

func commandCode(err error) string {
	var commandError politicsCommandError
	if errors.As(err, &commandError) {
		return commandError.Code
	}
	return ""
}

func TestPreparePoliticsSubtypeCreateIsNameFirstAndStable(t *testing.T) {
	updated, created, err := preparePoliticsSubtypeCreate(politicsCommandSnapshot(), politicsCommandRegistry(), politicsSubtypeCreateInput{
		ScriptID: "social_liberalism", RootID: "civic_coalition", Name: map[string]string{"en": "Social Liberalism", "zh": "社会自由主义"},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if created.ScriptID != "social_liberalism" || created.RootID != "civic_coalition" || created.Name["zh"] != "社会自由主义" {
		t.Fatalf("unexpected created definition: %+v", created)
	}
	if len(updated["sub_ideologies"].([]any)) != 2 {
		t.Fatal("subtype was not appended")
	}
	_, _, err = preparePoliticsSubtypeCreate(politicsCommandSnapshot(), politicsCommandRegistry(), politicsSubtypeCreateInput{
		ScriptID: "base_child", RootID: "democratic", Name: map[string]string{"en": "Base Child"},
	}, false)
	if commandCode(err) != "politics_source_lock_required" {
		t.Fatalf("expected source-lock error, got %v", err)
	}
}

func TestPreparePoliticsDefinitionPatch(t *testing.T) {
	name := map[string]string{"en": "Renewed Civic Reformism"}
	updated, definition, err := preparePoliticsDefinitionPatch(politicsCommandSnapshot(), politicsCommandRegistry(),
		politicsrefs.Target{Kind: politicsrefs.KindSubtype, ScriptID: "civic_reformism"},
		politicsDefinitionPatch{Name: workbench.Optional[map[string]string]{Set: true, Value: &name}})
	if err != nil || definition.Name["en"] != name["en"] || len(updated["sub_ideologies"].([]any)) != 1 {
		t.Fatalf("patch: %+v %v", definition, err)
	}
	_, _, err = preparePoliticsDefinitionPatch(politicsCommandSnapshot(), politicsCommandRegistry(),
		politicsrefs.Target{Kind: politicsrefs.KindRoot, ScriptID: "democratic"},
		politicsDefinitionPatch{Name: workbench.Optional[map[string]string]{Set: true, Value: &name}})
	if !errors.Is(err, politicsrefs.ErrImmutableSource) {
		t.Fatalf("source-owned definition should be immutable, got %v", err)
	}
	rootID := "democratic"
	_, _, err = preparePoliticsDefinitionPatch(politicsCommandSnapshot(), politicsCommandRegistry(),
		politicsrefs.Target{Kind: politicsrefs.KindSubtype, ScriptID: "civic_reformism"},
		politicsDefinitionPatch{RootID: workbench.Optional[string]{Set: true, Value: &rootID}})
	if commandCode(err) != "politics_parent_stable" {
		t.Fatalf("expected stable-parent refusal, got %v", err)
	}
	registry := politicsCommandRegistry()
	unsafe := "types = { }\ncolor = { 1 2 3 }\n}\nevil = {"
	_, _, err = preparePoliticsDefinitionPatch(politicsCommandSnapshot(), registry,
		politicsrefs.Target{Kind: politicsrefs.KindRoot, ScriptID: "civic_coalition"},
		politicsDefinitionPatch{Body: workbench.Optional[string]{Set: true, Value: &unsafe}})
	if commandCode(err) != "politics_root_invalid" {
		t.Fatalf("expected unsafe-body refusal, got %v", err)
	}
	if registry["root_definitions"].([]any)[0].(map[string]any)["body"] != "types = { civic_reformism = { } }\ncolor = { 1 2 3 }" {
		t.Fatal("rejected patch mutated the caller registry")
	}
	body := "types = { }\ncolor = { 1 2 3 }"
	_, _, err = preparePoliticsDefinitionPatch(politicsCommandSnapshot(), politicsCommandRegistry(),
		politicsrefs.Target{Kind: politicsrefs.KindSubtype, ScriptID: "civic_reformism"},
		politicsDefinitionPatch{Body: workbench.Optional[string]{Set: true, Value: &body}})
	if commandCode(err) != "politics_root_body_invalid" {
		t.Fatalf("expected root-only body refusal, got %v", err)
	}
}

func TestPreparePoliticsRootCreateStandaloneOnly(t *testing.T) {
	input := politicsRootCreateInput{ScriptID: "new_root", Body: "types = { }\ncolor = { 10 20 30 }", Name: map[string]string{"en": "New Root"}}
	updated, created, err := preparePoliticsRootCreate(politicsCommandSnapshot(), politicsCommandRegistry(), nil, input)
	if err != nil || created.Kind != politicsrefs.KindRoot || len(updated["root_definitions"].([]any)) != 2 {
		t.Fatalf("unexpected root result: %+v %v", created, err)
	}
	_, _, err = preparePoliticsRootCreate(politicsCommandSnapshot(), politicsCommandRegistry(), []string{"base"}, input)
	if commandCode(err) != "politics_root_takeover_out_of_scope" {
		t.Fatalf("expected no-takeover error, got %v", err)
	}
}

func TestMergeImportedPoliticsSourceProjectsReadOnlyDefinitions(t *testing.T) {
	updated, err := mergeImportedPoliticsSource(politicsCommandRegistry(), ideologysource.Discovery{Roots: []ideologysource.RootOwner{
		{RootID: "democratic", Body: "types = { conservatism = { } }\ncolor = { 0 0 255 }", SourceKind: ideologysource.SourceDependency, DependencyKey: "imported", SubtypeIDs: []string{"conservatism"}, TypesCount: 1},
		{RootID: "technocracy", Body: "types = { civic_technocracy = { } }\ncolor = { 42 110 145 }", SourceKind: ideologysource.SourceDependency,
			DependencyKey: "imported", SubtypeIDs: []string{"civic_technocracy"}, TypesCount: 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := workbench.ValidateRegistry(updated, nil); err != nil {
		t.Fatalf("imported source projection is not a valid registry: %v", err)
	}
	roots, subs := updated["root_definitions"].([]any), updated["sub_ideologies"].([]any)
	if len(roots) != 2 || len(subs) != 2 {
		t.Fatalf("unexpected registry cardinality: roots=%d subtypes=%d", len(roots), len(subs))
	}
	root := roots[1].(map[string]any)
	if root["id"] != "technocracy" || root["source"] != string(politicsrefs.SourceImported) {
		t.Fatalf("imported root projection=%#v", root)
	}
}

func TestParseIdeologyDefinitionsPreservesRootAndSubtypeIdentity(t *testing.T) {
	definitions, err := parseIdeologyDefinitions(`ideologies = {
		civic_coalition = { types = { civic_reformism = { can_be_randomly_selected = no } } color = { 1 2 3 } }
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 2 || definitions[0].Kind != politicsrefs.KindRoot || definitions[0].ScriptID != "civic_coalition" ||
		definitions[1].Kind != politicsrefs.KindSubtype || definitions[1].RootID != "civic_coalition" || !definitions[1].RandomDisabled {
		t.Fatalf("definitions=%#v", definitions)
	}
}

func TestRemovePoliticsManifestTargetKeepsFutureFields(t *testing.T) {
	var raw any
	_ = json.Unmarshal([]byte(`{"schema_version":1,"state":"ready","source_key":"imported","future":{"keep":true},"entries":[{"script_id":"civic_reformism"},{"script_id":"civic_socialism"}]}`), &raw)
	doc, changed, err := removePoliticsManifestTarget(raw, politicsrefs.Target{Kind: politicsrefs.KindSubtype, ScriptID: "civic_reformism"})
	if err != nil || !changed || len(doc["entries"].([]any)) != 1 || doc["state"] != "ready" || doc["future"] == nil {
		t.Fatalf("doc=%v changed=%v err=%v", doc, changed, err)
	}
	last, changed, err := removePoliticsManifestTarget(doc, politicsrefs.Target{Kind: politicsrefs.KindSubtype, ScriptID: "civic_socialism"})
	if err != nil || !changed || last["state"] != nil || last["source_key"] != nil || last["future"] == nil {
		t.Fatalf("last=%v changed=%v err=%v", last, changed, err)
	}
}

func TestRemovePoliticsRegistryTargetIsExactAndPreservesUnknownFields(t *testing.T) {
	var raw any
	_ = json.Unmarshal([]byte(`{
		"groups":["civic_coalition"],
		"root_definitions":[{"id":"civic_coalition","body":"types = {} color = { 1 2 3 }","future":{"keep":true}}],
		"sub_ideologies":[{"id":"civic_reformism","group":"civic_coalition"},{"id":"civic_socialism","group":"civic_coalition","future":"keep"}],
		"future_top_level":{"keep":true}
	}`), &raw)
	updated, err := removePoliticsRegistryTarget(decodePoliticsRegistry(raw), politicsrefs.Target{Kind: politicsrefs.KindSubtype, ScriptID: "civic_reformism"})
	if err != nil {
		t.Fatal(err)
	}
	subs := updated["sub_ideologies"].([]any)
	if updated["future_top_level"] == nil || len(subs) != 1 || subs[0].(map[string]any)["future"] != "keep" {
		t.Fatalf("updated=%v", updated)
	}
}

func TestUnparseableScriptReferenceScanFailsClosed(t *testing.T) {
	known := map[string]politicsrefs.Ideology{
		"civic_coalition": {Kind: politicsrefs.KindRoot, ScriptID: "civic_coalition", Source: politicsrefs.SourceProjectManaged},
	}
	tokens := scriptIdeologyTokens(`available = { has_government = civic_coalition`, known)
	if len(tokens) != 1 || tokens[0].ID != "civic_coalition" || tokens[0].Context != "conservative token match in unparsed script" {
		t.Fatalf("tokens=%#v", tokens)
	}
}

func seedVanillaRoots() map[string]politicsrefs.Ideology {
	known := map[string]politicsrefs.Ideology{}
	for _, root := range reftables.Ideologies().Groups {
		known[root] = politicsrefs.Ideology{Kind: politicsrefs.KindRoot, ScriptID: root, DisplayName: root, Source: politicsrefs.SourceVanilla}
	}
	return known
}

func TestScanKeepsGoingWhenPresetPutsGroupIdInSubtypeSlot(t *testing.T) {
	known := seedVanillaRoots()
	conflicts := []politicsrefs.IdeologyConflict{}
	for _, id := range []string{"democratic", "communism", "neutrality", "fascism", "nationalist"} {
		addPoliticsIdeology(known, &conflicts, politicsrefs.Ideology{Kind: politicsrefs.KindSubtype, ScriptID: id, DisplayName: id, Source: politicsrefs.SourceProjectManaged})
	}
	if len(conflicts) != 4 {
		t.Fatalf("conflicts = %#v", conflicts)
	}
	for _, id := range []string{"democratic", "communism", "neutrality", "fascism"} {
		if got := known[id]; got.Kind != politicsrefs.KindRoot || got.Source != politicsrefs.SourceVanilla {
			t.Errorf("%q became %#v", id, got)
		}
	}
	if got, ok := known["nationalist"]; !ok || got.Kind != politicsrefs.KindSubtype {
		t.Errorf("clean subtype lost: %#v", got)
	}
	snapshot := politicsrefs.Snapshot{ProjectID: "p", Conflicts: conflicts}
	for _, ideology := range known {
		snapshot.Ideologies = append(snapshot.Ideologies, ideology)
	}
	snapshot = politicsrefs.PartitionValid(snapshot)
	if err := politicsrefs.ValidateSnapshot(snapshot); err != nil || len(snapshot.Conflicts) != 5 {
		t.Fatalf("partition: %v %d", err, len(snapshot.Conflicts))
	}
}
