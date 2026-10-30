package worldout

import (
	"encoding/json"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/exportmsg"
)

func oobState(owner string, provinces ...int64) *OMap {
	m := NewOMap()
	m.Set("owner", owner)
	list := make([]Value, 0, len(provinces))
	for _, p := range provinces {
		list = append(list, json.Number(fmtInt(p)))
	}
	m.Set("provinces", list)
	return m
}

func oobStateNumbered(stateID int64, owner string, provinces ...int64) *OMap {
	m := oobState(owner, provinces...)
	m.Set("state_id", json.Number(fmtInt(stateID)))
	return m
}

func gunState570() *OMap {
	return oobState("GUN", 1733, 1790, 1860, 4366, 4734, 4806, 4840, 4866,
		7274, 7687, 7722, 7825, 10555, 10652, 12606, 12679)
}

func countLines(lines []string, want string) int {
	n := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == want {
			n++
		}
	}
	return n
}

func oobOwners(states ...*OMap) *oobOwnerIndex { return buildProvinceOwner(states) }

func joinWarnings(w []exportmsg.Message) string {
	lines := make([]string, len(w))
	for i, m := range w {
		lines[i] = exportmsg.For("en").Render(m)
	}
	return strings.Join(lines, "\n")
}

func TestOOBWiringAddsMissingReference(t *testing.T) {
	files := []OOBFile{{Path: "history/units/new_1978.txt", Tags: []string{"GUN"}}}
	plan := planOOBWiring(files, map[string][]string{}, map[string]bool{"GUN": true},
		oobOwners(), "GUN")
	if len(plan.Actions) != 1 {
		t.Fatalf("want exactly 1 wire action, got %d (%+v)", len(plan.Actions), plan.Actions)
	}
	got := plan.Actions[0]
	if got.Tag != "GUN" || got.Stem != "new_1978" || got.Layer != 1 {
		t.Fatalf("wired %+v, want {GUN new_1978 layer 1}", got)
	}
	if !strings.Contains(joinWarnings(plan.Warnings), `set_oob = "new_1978"`) {
		t.Fatalf("author was not told what was added: %q", plan.Warnings)
	}
}

func TestOOBWiringDoesNotDuplicateExistingReference(t *testing.T) {
	files := []OOBFile{{Path: "history/units/new_1978.txt", Tags: []string{"GUN"}}}
	for _, existing := range []string{"new_1978"} {
		plan := planOOBWiring(files, map[string][]string{"GUN": {existing}},
			map[string]bool{"GUN": true}, oobOwners(), "GUN")
		if len(plan.Actions) != 0 {
			t.Fatalf("stem %q already referenced but %d action(s) planned: %+v",
				existing, len(plan.Actions), plan.Actions)
		}
		if len(plan.Warnings) != 0 {
			t.Fatalf("nothing happened, so nothing should be said: %q", plan.Warnings)
		}
	}

	plan := planOOBWiring(files, map[string][]string{"TOM": {"new_1978"}},
		map[string]bool{"GUN": true, "TOM": true}, oobOwners(), "GUN")
	if len(plan.Actions) != 0 {
		t.Fatalf("cross-country reference ignored: %+v", plan.Actions)
	}
}

func TestOOBWiringGunPlaceholderStillWiresAndReportsDangling(t *testing.T) {
	files := []OOBFile{{Path: "history/units/new_1978.txt", Tags: []string{"GUN"}}}
	plan := planOOBWiring(files, map[string][]string{"GUN": {"TAG_1936"}},
		map[string]bool{"GUN": true, "TOM": true}, oobOwners(), "GUN")
	if len(plan.Actions) != 1 || plan.Actions[0].Stem != "new_1978" || plan.Actions[0].Tag != "GUN" {
		t.Fatalf("a reference to another stem must not count as wired: %+v", plan.Actions)
	}
	all := joinWarnings(plan.Warnings)
	if !strings.Contains(all, "TAG_1936") || !strings.Contains(all, "no `history/units/TAG_1936.txt`") {
		t.Fatalf("dangling TAG_1936 not reported as actionable: %q", plan.Warnings)
	}
	if !strings.Contains(all, "new_1978") {
		t.Fatalf("dangling report must point at a real OOB file: %q", plan.Warnings)
	}
}

func TestParseOOBRefsReadsRawScriptAndSkipsComments(t *testing.T) {
	raw := "# 作战序列 — 原样写入本国 history 文件\n" +
		"set_oob = \"TAG_1936\"\n" +
		"# units = { division = { name = \"1. Division\" location = 1 } }\n" +
		"# set_oob = \"commented_out\"\n"
	got := parseOOBRefs(raw)
	if len(got) != 1 || got[0] != "TAG_1936" {
		t.Fatalf("parseOOBRefs(raw_script) = %v, want [TAG_1936]", got)
	}
	multi := parseOOBRefs("oob = \"PAK_1936\"\nset_air_oob = \"A\"\nset_naval_oob = N_1936\n")
	want := []string{"PAK_1936", "A", "N_1936"}
	if strings.Join(got, ",") == "" || strings.Join(multi, ",") != strings.Join(want, ",") {
		t.Fatalf("parseOOBRefs = %v, want %v", multi, want)
	}
	if refs := parseOOBRefs("set_technology = { tech_oob = 1 }"); len(refs) != 0 {
		t.Fatalf("a token merely ending in _oob is not a reference: %v", refs)
	}
}

func TestOOBWiringAttributesByDeploymentProvince(t *testing.T) {
	files := []OOBFile{{
		Path:      "history/units/new_1978.txt",
		Tags:      []string{"NEW", "NEW"},
		Locations: []int64{7274, 7274, 7274, 7274, 1794, 1794, 4866, 1790, 10555, 4840, 7722},
	}}
	owners := buildProvinceOwner([]*OMap{gunState570(), oobState("GUN", 1794)})
	if owners.provinceOwner(7274) != "GUN" {
		t.Fatalf("province 7274 must resolve to GUN via state 570, got %q",
			owners.provinceOwner(7274))
	}
	plan := planOOBWiring(files, map[string][]string{}, map[string]bool{"GUN": true, "TOM": true},
		owners, "TOM")
	if len(plan.Actions) != 1 {
		t.Fatalf("want 1 action, got %+v", plan.Actions)
	}
	if plan.Actions[0].Tag != "GUN" || plan.Actions[0].Layer != 3 {
		t.Fatalf("wired to %+v — the file-name prefix tag NEW must not win over "+
			"the states the divisions actually spawn in", plan.Actions[0])
	}

	if !strings.Contains(joinWarnings(plan.Warnings),
		"GUN was chosen from the owner of the states its divisions spawn in.") {
		t.Fatalf("land attribution wording changed: %q", plan.Warnings)
	}
}

func TestOOBWiringFallsBackToProjectTagThenGivesUp(t *testing.T) {
	files := []OOBFile{{Path: "history/units/orphan.txt", Tags: []string{"ZZZ"}, Locations: []int64{999}}}
	plan := planOOBWiring(files, map[string][]string{}, map[string]bool{"GUN": true},
		oobOwners(), "GUN")
	if len(plan.Actions) != 1 || plan.Actions[0].Tag != "GUN" || plan.Actions[0].Layer != 4 {
		t.Fatalf("project-tag fallback failed: %+v", plan.Actions)
	}
	none := planOOBWiring(files, map[string][]string{}, map[string]bool{}, oobOwners(), "")
	if len(none.Actions) != 0 {
		t.Fatalf("nothing to attribute to, so nothing may be wired: %+v", none.Actions)
	}
	if !strings.Contains(joinWarnings(none.Warnings), "could not tell which country") {
		t.Fatalf("silent give-up is the bug being fixed: %q", none.Warnings)
	}
}

func TestOOBWiringSkipsAuthorRawOverriddenPath(t *testing.T) {
	files := []OOBFile{{
		Path: "history/units/new_1978.txt", Tags: []string{"GUN"}, RawOverridden: true,
	}}
	plan := planOOBWiring(files, map[string][]string{}, map[string]bool{"GUN": true},
		oobOwners(), "GUN")
	if len(plan.Actions) != 0 {
		t.Fatalf("wired a path emit/misc never writes: %+v", plan.Actions)
	}
	if len(plan.Warnings) != 0 {
		t.Fatalf("no synthesis happened, so nothing to report: %q", plan.Warnings)
	}
	referenced := planOOBWiring(files, map[string][]string{"GUN": {"new_1978"}},
		map[string]bool{"GUN": true}, oobOwners(), "GUN")
	if len(referenced.Warnings) != 0 {
		t.Fatalf("the author's own file occupies that path — not dangling: %q",
			referenced.Warnings)
	}
}

func TestOOBWiringMixedFileEmitsExactlyOneReference(t *testing.T) {

	files := []OOBFile{{
		Path: "history/units/mixed_1936.txt",

		Tags:      []string{"GER", "GER", "GER", "GER"},
		Locations: []int64{11},
	}}
	plan := planOOBWiring(files, map[string][]string{}, map[string]bool{"GER": true},
		oobOwners(oobState("GER", 11)), "GER")
	if len(plan.Actions) != 1 {
		t.Fatalf("one file must produce one reference, got %d: %+v",
			len(plan.Actions), plan.Actions)
	}

	rendered := renderOOBScenario(t, files, nil)
	body := rendered["GER"]
	if got := countLines(body, `set_oob = "mixed_1936"`); got != 1 {
		t.Fatalf("emitted %d set_oob lines for one mixed file, want exactly 1:\n%s",
			got, strings.Join(body, "\n"))
	}
	for _, forbidden := range []string{"set_air_oob", "set_naval_oob"} {
		if strings.Contains(strings.Join(body, "\n"), forbidden) {
			t.Fatalf("%s must never be synthesized (one file = one load):\n%s",
				forbidden, strings.Join(body, "\n"))
		}
	}
}

func TestOOBWiringEndToEndFoldsAndCreates(t *testing.T) {
	files := []OOBFile{
		{Path: "history/units/new_1978.txt", Tags: []string{"GUN"}},
		{Path: "history/units/tom_army.txt", Tags: []string{"TOM"}},
	}
	hf := NewOMap()
	hf.Set("country_tag", "GUN")
	hf.Set("capital", json.Number("570"))
	rendered := renderOOBScenario(t, files, []*OMap{hf})

	gun := strings.Join(rendered["GUN"], "\n")
	if !strings.Contains(gun, "capital = 570") {
		t.Fatalf("folding lost the pre-existing history content:\n%s", gun)
	}
	if countLines(rendered["GUN"], `set_oob = "new_1978"`) != 1 {
		t.Fatalf("GUN's existing history file was not wired:\n%s", gun)
	}
	tom := rendered["TOM"]
	if countLines(tom, `set_oob = "tom_army"`) != 1 {
		t.Fatalf("TOM had no history file and none was created:\n%s", strings.Join(tom, "\n"))
	}
	if !strings.Contains(strings.Join(tom, "\n"), "guides/division.md") {
		t.Fatalf("the synthesized line must explain itself:\n%s", strings.Join(tom, "\n"))
	}
}

type oobRenderSpec struct {
	Files        []OOBFile
	HistoryFiles []*OMap
	States       []*OMap
	PrimaryTag   string
	Countries    []string
}

func renderOOBScenario(t *testing.T, files []OOBFile, historyFiles []*OMap) map[string][]string {
	t.Helper()
	spec := oobRenderSpec{Files: files, HistoryFiles: historyFiles, PrimaryTag: "GER"}
	if len(files) > 0 && len(files[0].Tags) > 0 {
		spec.PrimaryTag = files[0].Tags[0]
	}
	seen := map[string]bool{}
	for _, f := range files {
		for _, ct := range f.Tags {
			if !seen[ct] {
				seen[ct] = true
				spec.Countries = append(spec.Countries, ct)
			}
		}
	}
	return renderOOBSpec(t, spec)
}

func renderOOBSpec(t *testing.T, spec oobRenderSpec) map[string][]string {
	t.Helper()
	tables, err := testTables(t)
	if err != nil {
		t.Fatalf("load tables: %v", err)
	}
	project := NewOMap()
	project.Set("export_languages", []Value{"en"})
	content := NewOMap()
	var countries []*OMap
	for _, ct := range spec.Countries {
		content.Set(ct, true)
		c := NewOMap()
		c.Set("tag", ct)
		countries = append(countries, c)
	}
	scenario := &Scenario{
		Project: project, Tag: spec.PrimaryTag, ModName: "T", ModDirName: "T",
		ExistenceSets: map[string]map[string]bool{"idea": {}, "focus": {}},
		ContentByTag:  content, Countries: countries, States: spec.States,
		HistoryFiles: spec.HistoryFiles, OOBFiles: spec.Files,
	}
	records, _ := EmitScenario(tables, scenario)
	out := map[string][]string{}
	for _, r := range records {
		if !strings.HasPrefix(r.Path, "T/history/countries/") {
			continue
		}
		name := strings.TrimPrefix(r.Path, "T/history/countries/")
		ctag, _, _ := strings.Cut(name, " - ")
		out[ctag] = strings.Split(r.Text, "\n")
	}
	return out
}

func TestCorpusScenarioHasNoOOBFiles(t *testing.T) {
	o := NewOMap()
	o.Set("name", "x")
	o.Set("tag", "GER")
	if s := scenarioFromOMap(o); len(s.OOBFiles) != 0 {
		t.Fatalf("corpus scenarios must carry no OOB files (byte-parity), got %+v", s.OOBFiles)
	}
}

func TestOOBWiringRawSourcedFilesAreNotDangling(t *testing.T) {
	files := []OOBFile{
		{Path: "history/units/my_army.txt", Tags: []string{"GUN"}},

		{Path: "history/units/GER_1936.txt", RawOverridden: true},
		{Path: "history/units/SOV_1936.txt", RawOverridden: true},
	}
	refs := map[string][]string{
		"GER": {"GER_1936"},
		"SOV": {"SOV_1936"},
		"GUN": {"GER_1936"},
	}
	plan := planOOBWiring(files, refs,
		map[string]bool{"GUN": true, "GER": true, "SOV": true}, oobOwners(), "GUN")

	if len(plan.Actions) != 1 || plan.Actions[0].Stem != "my_army" {
		t.Fatalf("only the self-generated file may be wired: %+v", plan.Actions)
	}
	for _, w := range strings.Split(joinWarnings(plan.Warnings), "\n") {
		if strings.Contains(w, "GER_1936") || strings.Contains(w, "SOV_1936") {
			t.Fatalf("raw/imported OOB file reported as dangling: %q", w)
		}
	}

	refs["GUN"] = append(refs["GUN"], "TAG_1936")
	again := planOOBWiring(files, refs,
		map[string]bool{"GUN": true, "GER": true, "SOV": true}, oobOwners(), "GUN")
	if !strings.Contains(joinWarnings(again.Warnings), "TAG_1936") {
		t.Fatalf("a truly missing stem must still be reported: %q", again.Warnings)
	}
}

func TestOOBWiringSilentWhenNothingWasGenerated(t *testing.T) {
	files := []OOBFile{{Path: "history/units/GER_1936.txt", RawOverridden: true}}
	plan := planOOBWiring(files, map[string][]string{"GER": {"GER_1936", "ITA_1936"}},
		map[string]bool{"GER": true}, oobOwners(), "GER")
	if len(plan.Actions) != 0 || len(plan.Warnings) != 0 {
		t.Fatalf("nothing generated → nothing to say, got %+v / %q", plan.Actions, plan.Warnings)
	}
}

func TestOOBWiringAttributesNavyByNavalBase(t *testing.T) {

	owners := oobOwners(oobStateNumbered(570, "TOM", 1790, 4840, 7274, 10555))
	known := map[string]bool{"GUN": true, "TOM": true}

	named := planOOBWiring([]OOBFile{{
		Path:           "history/units/TOM_navy.txt",
		NavalProvinces: []int64{7274},
	}}, map[string][]string{}, known, owners, "GUN")
	if len(named.Actions) != 1 || named.Actions[0].Tag != "TOM" {
		t.Fatalf("a pure-navy file went to the project tag instead of TOM: %+v",
			named.Actions)
	}

	anon := planOOBWiring([]OOBFile{{
		Path:           "history/units/new_1978.txt",
		NavalProvinces: []int64{7274},
	}}, map[string][]string{}, known, owners, "GUN")
	if len(anon.Actions) != 1 {
		t.Fatalf("want 1 action, got %+v", anon.Actions)
	}
	got := anon.Actions[0]
	if got.Tag != "TOM" || got.Layer != 3 {
		t.Fatalf("fleet berthed in TOM's province 7274 was wired to %+v; before "+
			"H-2 this fell through to the project tag GUN", got)
	}
	reason := joinWarnings(anon.Warnings)
	if !strings.Contains(reason, "naval base") {
		t.Fatalf("the author cannot tell WHY this fleet became TOM's: %q",
			anon.Warnings)
	}
	if strings.Contains(reason, "divisions spawn in") {
		t.Fatalf("a navy-only file must not claim divisions decided it: %q",
			anon.Warnings)
	}
}

func TestOOBWiringNavyEndToEndLandsInTOMHistory(t *testing.T) {
	rendered := renderOOBSpec(t, oobRenderSpec{
		Files: []OOBFile{{
			Path:           "history/units/new_1978.txt",
			NavalProvinces: []int64{7274},
		}},
		States:     []*OMap{oobStateNumbered(570, "TOM", 1790, 7274)},
		PrimaryTag: "GUN",
		Countries:  []string{"GUN", "TOM"},
	})
	if countLines(rendered["TOM"], "set_oob = \"new_1978\"") != 1 {
		t.Fatalf("TOM did not get the fleet:\n%s", strings.Join(rendered["TOM"], "\n"))
	}
	if strings.Contains(strings.Join(rendered["GUN"], "\n"), "set_oob") {
		t.Fatalf("the project's main tag stole TOM's fleet:\n%s",
			strings.Join(rendered["GUN"], "\n"))
	}
}

func TestOOBWiringAttributesAirWingByState(t *testing.T) {
	owners := oobOwners(
		oobStateNumbered(570, "TOM", 1790, 7274),
		oobStateNumbered(578, "GUN", 570, 7789),
	)
	if owners.stateOwner(570) != "TOM" || owners.provinceOwner(570) != "GUN" {
		t.Fatalf("fixture is not discriminating: state 570=%q province 570=%q",
			owners.stateOwner(570), owners.provinceOwner(570))
	}
	plan := planOOBWiring([]OOBFile{{
		Path:      "history/units/new_1978.txt",
		AirStates: []int64{570},
	}}, map[string][]string{}, map[string]bool{"GUN": true, "TOM": true},
		owners, "GUN")
	if len(plan.Actions) != 1 {
		t.Fatalf("want 1 action, got %+v", plan.Actions)
	}
	if plan.Actions[0].Tag != "TOM" || plan.Actions[0].Layer != 3 {
		t.Fatalf("air wing group in state 570 wired to %+v — reading location as "+
			"a province gives GUN, which is the wrong answer", plan.Actions[0])
	}
	if !strings.Contains(joinWarnings(plan.Warnings), "air wings are based in") {
		t.Fatalf("air attribution does not explain itself: %q", plan.Warnings)
	}
}

func TestOOBWiringAttributesByFileNamePrefix(t *testing.T) {
	plan := planOOBWiring(
		[]OOBFile{{Path: "history/units/TOM_navy.txt"}},
		map[string][]string{}, map[string]bool{"GUN": true, "TOM": true},
		oobOwners(), "GUN")
	if len(plan.Actions) != 1 {
		t.Fatalf("want 1 action, got %+v", plan.Actions)
	}
	if plan.Actions[0].Tag != "TOM" || plan.Actions[0].Layer != 2 {
		t.Fatalf("TOM_navy wired to %+v, want TOM at layer 2", plan.Actions[0])
	}
	if !strings.Contains(joinWarnings(plan.Warnings), "file name starts with") {
		t.Fatalf("layer 2 does not explain itself: %q", plan.Warnings)
	}

	bare := planOOBWiring([]OOBFile{{Path: "history/units/TOM.txt"}},
		map[string][]string{}, map[string]bool{"GUN": true, "TOM": true},
		oobOwners(), "GUN")
	if len(bare.Actions) != 1 || bare.Actions[0].Tag != "TOM" {
		t.Fatalf("TOM.txt should still resolve to TOM: %+v", bare.Actions)
	}
}

func TestOOBWiringFileNamePrefixRejectsUnknownTag(t *testing.T) {
	files := []OOBFile{{Path: "history/units/new_1978.txt", Locations: []int64{7274}}}
	owners := oobOwners(oobStateNumbered(570, "GUN", 1790, 7274))
	plan := planOOBWiring(files, map[string][]string{},
		map[string]bool{"GUN": true, "TOM": true}, owners, "TOM")
	if len(plan.Actions) != 1 {
		t.Fatalf("want 1 action, got %+v", plan.Actions)
	}
	if plan.Actions[0].Tag == "NEW" {
		t.Fatalf("the phantom tag NEW was wired — the knownTags guard is gone")
	}
	if plan.Actions[0].Tag != "GUN" || plan.Actions[0].Layer != 3 {
		t.Fatalf("new_1978 must fall past layer 2 to geography, got %+v",
			plan.Actions[0])
	}

	bare := planOOBWiring([]OOBFile{{Path: "history/units/new_1978.txt"}},
		map[string][]string{}, map[string]bool{"GUN": true, "TOM": true},
		oobOwners(), "TOM")
	if len(bare.Actions) != 1 || bare.Actions[0].Tag != "TOM" || bare.Actions[0].Layer != 4 {
		t.Fatalf("new_1978 with no geography must land on the project tag: %+v",
			bare.Actions)
	}
}

func TestOOBWiringRowTagBeatsFileNamePrefix(t *testing.T) {
	plan := planOOBWiring(
		[]OOBFile{{Path: "history/units/GUN_army.txt", Tags: []string{"TOM", "TOM"}}},
		map[string][]string{}, map[string]bool{"GUN": true, "TOM": true},
		oobOwners(), "GUN")
	if len(plan.Actions) != 1 || plan.Actions[0].Tag != "TOM" || plan.Actions[0].Layer != 1 {
		t.Fatalf("layer 1 must outrank the file name: %+v", plan.Actions)
	}
}

func TestParseOOBRefsReadsLoadOOB(t *testing.T) {
	got := parseOOBRefs("load_oob = \"my_template_shared\"\n")
	if len(got) != 1 || got[0] != "my_template_shared" {
		t.Fatalf("parseOOBRefs(load_oob) = %v, want [my_template_shared]", got)
	}
	if refs := parseOOBRefs("load_oob = new_1978"); len(refs) != 1 || refs[0] != "new_1978" {
		t.Fatalf("unquoted load_oob = %v", refs)
	}

	if refs := parseOOBRefs("set_technology = { tech_oob = 1 }\nunload_oob = 3\n"); len(refs) != 0 {
		t.Fatalf("suffix-only matches leaked back in: %v", refs)
	}
}

func TestOOBWiringLoadOOBCountsAsReferenced(t *testing.T) {
	hf := NewOMap()
	hf.Set("country_tag", "GUN")
	hf.Set("raw_script", "load_oob = \"new_1978\"\n")
	rendered := renderOOBSpec(t, oobRenderSpec{
		Files:        []OOBFile{{Path: "history/units/new_1978.txt", Tags: []string{"GUN"}}},
		HistoryFiles: []*OMap{hf},
		PrimaryTag:   "GUN",
		Countries:    []string{"GUN"},
	})
	body := strings.Join(rendered["GUN"], "\n")
	if strings.Contains(body, "set_oob") {
		t.Fatalf("load_oob already loads new_1978; a synthesized set_oob loads it "+
			"a second time and doubles the army:\n%s", body)
	}
	if !strings.Contains(body, "load_oob") {
		t.Fatalf("the author's own load_oob line vanished:\n%s", body)
	}
}

func TestOOBWiringLandScenarioIsUnchanged(t *testing.T) {
	files := []OOBFile{{
		Path:      "history/units/new_1978.txt",
		Tags:      []string{"NEW", "NEW"},
		Locations: []int64{7274, 7274, 1794},
	}}
	owners := oobOwners(gunState570(), oobState("GUN", 1794))
	plan := planOOBWiring(files, map[string][]string{},
		map[string]bool{"GUN": true, "TOM": true}, owners, "TOM")
	if len(plan.Actions) != 1 {
		t.Fatalf("want 1 action, got %+v", plan.Actions)
	}
	if plan.Actions[0].Tag != "GUN" || plan.Actions[0].Stem != "new_1978" {
		t.Fatalf("land attribution changed: %+v", plan.Actions[0])
	}
	want := "Order of battle: nothing in your country history loaded " +
		"`history/units/new_1978.txt`, so the game would have ignored everything " +
		"in it. The export added `set_oob = \"new_1978\"` to GUN's history file — GUN " +
		"was chosen from the owner of the states its divisions spawn in. Move " +
		"that line if it belongs to another country."
	if len(plan.Warnings) != 1 || joinWarnings(plan.Warnings) != want {
		t.Fatalf("land warning drifted.\n got: %q\nwant: %q", plan.Warnings, want)
	}
}

func TestOOBWiringGeographyCombinesAndVotes(t *testing.T) {
	owners := oobOwners(
		oobStateNumbered(570, "TOM", 7274),
		oobStateNumbered(578, "GUN", 7789),
	)
	both := planOOBWiring([]OOBFile{{
		Path:           "history/units/mixed.txt",
		Locations:      []int64{7274},
		NavalProvinces: []int64{7274},
		AirStates:      []int64{570},
	}}, map[string][]string{}, map[string]bool{"GUN": true, "TOM": true},
		owners, "GUN")
	msg := joinWarnings(both.Warnings)
	for _, want := range []string{"divisions spawn in", "naval base", "air wings are based in"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("all three evidence kinds should be named, missing %q: %q", want, msg)
		}
	}

	split := planOOBWiring([]OOBFile{{
		Path:           "history/units/split.txt",
		Locations:      []int64{7789, 7789},
		NavalProvinces: []int64{7274},
	}}, map[string][]string{}, map[string]bool{"GUN": true, "TOM": true},
		owners, "TOM")
	if len(split.Actions) != 1 || split.Actions[0].Tag != "GUN" {
		t.Fatalf("the majority of located rows must win: %+v", split.Actions)
	}
	if strings.Contains(joinWarnings(split.Warnings), "naval base") {
		t.Fatalf("only the evidence that BACKS the winner may be cited: %q",
			split.Warnings)
	}
}
