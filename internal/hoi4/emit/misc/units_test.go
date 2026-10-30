package emitmisc

import (
	"sort"
	"strings"
	"testing"
)

func testEnv(tag string) *Env {
	return &Env{ModDir: "mod/", Tag: tag, StructuredUnitPaths: map[string]bool{}}
}

func equip(id, source, sourceFile string) *NS {
	return NewNS().
		Set("equipment_id", id).
		Set("source", source).
		Set("source_file", sourceFile).
		Set("year", nil).
		Set("is_archetype", false).
		Set("picture", "").
		Set("is_buildable", nil).
		Set("equipment_type", "").
		Set("archetype", "").
		Set("parent", "").
		Set("manpower", nil).
		Set("can_license", nil).
		Set("carrier_capable", nil).
		Set("resources", NewOD()).
		Set("raw_extras", NewOD())
}

func subUnit(id, source, sourceFile string, ordering int64) *NS {
	return NewNS().
		Set("sub_unit_id", id).
		Set("source", source).
		Set("source_file", sourceFile).
		Set("ordering", ordering).
		Set("unit_types", []any{}).
		Set("categories", []any{}).
		Set("essential", []any{}).
		Set("need", NewOD()).
		Set("stats", NewOD()).
		Set("terrain_modifiers", NewOD())
}

func rawFile(destPath, source string) *NS {
	return NewNS().Set("dest_path", destPath).Set("source", source)
}

func writtenPaths(env *Env) []string {
	out := make([]string, 0, len(env.Writes))
	for _, w := range env.Writes {
		out = append(out, w.Path)
	}
	sort.Strings(out)
	return out
}

func writeAt(t *testing.T, env *Env, path string) string {
	t.Helper()
	for _, w := range env.Writes {
		if w.Path == path {
			return w.Text
		}
	}
	t.Fatalf("no write at %q; got %v", path, writtenPaths(env))
	return ""
}

func keysOf(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func eqStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPlanImportedOnlyGroupDefersToRawReplay(t *testing.T) {
	env := testEnv("SOV")
	equipment := []*NS{
		equip("sov_rifle", "imported", "common/units/equipment/vanilla_inf.txt"),
		equip("sov_smg", "imported", "common/units/equipment/vanilla_inf.txt"),
	}
	subUnits := []*NS{
		subUnit("sov_infantry", "imported", "common/units/sov_legacy.txt", 0),
	}
	rawFiles := []*NS{
		rawFile("common/units/equipment/vanilla_inf.txt", "imported"),
		rawFile("common/units/sov_legacy.txt", "imported"),
	}

	plan := EmitUnits(env, equipment, subUnits, rawFiles)

	wantReplay := []string{
		"common/units/equipment/vanilla_inf.txt", "common/units/sov_legacy.txt",
	}
	if got := keysOf(plan.RawReplayPaths); !eqStrings(got, wantReplay) {
		t.Fatalf("raw replay paths = %v, want %v", got, wantReplay)
	}
	if len(plan.Files) != 0 {
		t.Fatalf("imported-only groups must not be structured: %v", plan.FileKeys)
	}
	if len(env.Writes) != 0 {
		t.Fatalf("imported-only groups must write nothing: %v", writtenPaths(env))
	}
	if len(env.StructuredUnitPaths) != 0 {
		t.Fatalf("structured_unit_paths must stay empty: %v", keysOf(env.StructuredUnitPaths))
	}
}

func TestPlanImportedOnlyWithoutRawFileStaysStructured(t *testing.T) {
	env := testEnv("SOV")
	equipment := []*NS{equip("sov_orphan", "imported", "common/units/equipment/orphan.txt")}

	plan := EmitUnits(env, equipment, nil, nil)

	if len(plan.RawReplayPaths) != 0 {
		t.Fatalf("nothing to replay, got %v", keysOf(plan.RawReplayPaths))
	}
	if !plan.StructuredPaths["common/units/equipment/orphan.txt"] {
		t.Fatalf("orphan group must be structured, got %v", keysOf(plan.StructuredPaths))
	}
	if got := writtenPaths(env); !eqStrings(got, []string{"mod/common/units/equipment/orphan.txt"}) {
		t.Fatalf("writes = %v", got)
	}
}

func TestPlanMixedGroupPromotesEveryRowInTheGroup(t *testing.T) {
	env := testEnv("USA")
	equipment := []*NS{
		equip("usa_a", "imported", "common/units/equipment/mixed.txt"),
		equip("usa_b", "imported", "common/units/equipment/mixed.txt"),
		equip("usa_c_edited", "created", "common/units/equipment/mixed.txt"),
	}
	subUnits := []*NS{
		subUnit("usa_inf", "imported", "common/units/mixed_units.txt", 0),
		subUnit("usa_marine", "edited", "common/units/mixed_units.txt", 1),
	}
	rawFiles := []*NS{
		rawFile("common/units/equipment/mixed.txt", "imported"),
		rawFile("common/units/mixed_units.txt", "imported"),
	}

	plan := EmitUnits(env, equipment, subUnits, rawFiles)

	if len(plan.RawReplayPaths) != 0 {
		t.Fatalf("a touched group must not replay raw: %v", keysOf(plan.RawReplayPaths))
	}
	script := writeAt(t, env, "mod/common/units/equipment/mixed.txt")
	for _, id := range []string{"usa_a", "usa_b", "usa_c_edited"} {
		if !strings.Contains(script, "\t"+id+" = {") {
			t.Fatalf("promoted group dropped %q:\n%s", id, script)
		}
	}
	subScript := writeAt(t, env, "mod/common/units/mixed_units.txt")
	for _, id := range []string{"usa_inf", "usa_marine"} {
		if !strings.Contains(subScript, "\t"+id+" = {") {
			t.Fatalf("promoted group dropped %q:\n%s", id, subScript)
		}
	}
}

func TestPlanCreatedOnlyUsesTagPrefixedDefaultPaths(t *testing.T) {
	env := testEnv("GER")
	equipment := []*NS{equip("ger_rifle", "created", "")}
	subUnits := []*NS{subUnit("ger_inf", "created", "", 0)}

	plan := EmitUnits(env, equipment, subUnits, nil)

	want := []string{
		"common/units/GER_sub_units.txt",
		"common/units/equipment/zz_GER_equipment.txt",
	}
	if got := keysOf(plan.StructuredPaths); !eqStrings(got, want) {
		t.Fatalf("structured paths = %v, want %v", got, want)
	}
	if got := keysOf(env.StructuredUnitPaths); !eqStrings(got, want) {
		t.Fatalf("structured_unit_paths = %v, want %v", got, want)
	}
	gotWrites := writtenPaths(env)
	wantWrites := []string{
		"mod/common/units/GER_sub_units.txt",
		"mod/common/units/equipment/zz_GER_equipment.txt",
	}
	if !eqStrings(gotWrites, wantWrites) {
		t.Fatalf("writes = %v, want %v", gotWrites, wantWrites)
	}
}

func TestEmitUnitsYieldsPathToAuthorRawOverride(t *testing.T) {
	env := testEnv("FRA")
	equipment := []*NS{equip("fra_rifle", "created", "")}
	subUnits := []*NS{subUnit("fra_inf", "created", "", 0)}
	rawFiles := []*NS{rawFile("common/units/equipment/zz_FRA_equipment.txt", "manual")}

	plan := EmitUnits(env, equipment, subUnits, rawFiles)

	if _, planned := plan.Files["common/units/equipment/zz_FRA_equipment.txt"]; !planned {
		t.Fatal("the overridden path must still be PLANNED (not silently dropped)")
	}
	got := writtenPaths(env)
	if !eqStrings(got, []string{"mod/common/units/FRA_sub_units.txt"}) {
		t.Fatalf("author raw override must win the write: %v", got)
	}
	if env.StructuredUnitPaths["common/units/equipment/zz_FRA_equipment.txt"] {
		t.Fatal("an unwritten path must not be reported as a structured unit path")
	}
}

func TestGenerateEquipmentFileScriptShape(t *testing.T) {
	arch := equip("infantry_equipment", "created", "").
		Set("year", int64(1918)).
		Set("is_archetype", true).
		Set("is_buildable", false).
		Set("equipment_type", "infantry").
		Set("soft_attack", 3.0).
		Set("build_cost_ic", 0.43).
		Set("manpower", int64(100)).
		Set("resources", NewOD().Set("steel", int64(1)))

	variant := equip("z_variant", "created", "").
		Set("archetype", "infantry_equipment").
		Set("maximum_speed", 4.0).
		Set("raw_extras", NewOD().Set("priority", "5"))

	got := GenerateEquipmentFileScript([]*NS{variant, arch})
	want := strings.Join([]string{
		"equipments = {",
		"",
		"\tinfantry_equipment = {",
		"\t\tyear = 1918",
		"\t\tis_archetype = yes",
		"\t\tis_buildable = no",
		"\t\ttype = infantry",
		"\t\tsoft_attack = 3",
		"\t\tbuild_cost_ic = 0.43",
		"\t\tmanpower = 100",
		"\t\tresources = {",
		"\t\t\tsteel = 1",
		"\t\t}",
		"\t}",
		"",
		"\tz_variant = {",
		"\t\tarchetype = infantry_equipment",
		"\t\tmaximum_speed = 4",
		"\t\tpriority = 5",
		"\t}",
		"",
		"}",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestGenerateSubUnitFileScriptShape(t *testing.T) {

	later := subUnit("later", "created", "", 0).
		Set("sprite", "later_entity").
		Set("priority", int64(7)).
		Set("special_forces", true).
		Set("unit_types", []any{"infantry"}).
		Set("need", NewOD().Set("infantry_equipment", int64(100))).
		Set("stats", NewOD().Set("max_organisation", int64(60))).
		Set("terrain_modifiers", NewOD().Set("forest", NewOD().Set("attack", 0.1)))
	earlier := subUnit("earlier", "created", "", 1).
		Set("group", "Support Group")

	got := GenerateSubUnitFileScript([]*NS{earlier, later})
	want := strings.Join([]string{
		"sub_units = {",
		"",
		"\tlater = {",
		"\t\tsprite = later_entity",
		"\t\tpriority = 7",
		"\t\tspecial_forces = yes",
		"\t\ttype = { infantry }",
		"\t\tneed = {",
		"\t\t\tinfantry_equipment = 100",
		"\t\t}",
		"\t\tmax_organisation = 60",
		"\t\tforest = {",
		"\t\t\tattack = 0.1",
		"\t\t}",
		"\t}",
		"",
		"\tearlier = {",
		"\t\tgroup = \"Support Group\"",
		"\t}",
		"",
		"}",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestFmtSubUnitScalar(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{true, "yes"},
		{false, "no"},
		{3.0, "3"},
		{0.000005, "0.000005"},
		{int64(-3), "-3"},
		{"plain", "plain"},
		{"", `""`},
		{"with space", `"with space"`},
		{`a"b`, `"a\"b"`},
		{`gfx/inf`, `"gfx/inf"`},
		{`a\b`, `"a\\b"`},
	}
	for _, c := range cases {
		if got := fmtSubUnitScalar(c.in); got != c.want {
			t.Errorf("fmtSubUnitScalar(%#v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEmitTreesTechMilitaryEmitsUnitScriptsNotJustLocalisation(t *testing.T) {
	env := testEnv("GER")
	env.LangTriples = BuildExportLangTriples([]string{"en"})
	equipment := []*NS{equip("ger_rifle", "created", "").Set("soft_attack", 3.0)}
	subUnits := []*NS{subUnit("ger_inf", "created", "", 0).Set("sprite", "ger_entity")}

	EmitTreesTechMilitary(env, nil, nil, equipment, subUnits, nil, nil, nil, nil, nil)

	got := writtenPaths(env)
	want := []string{
		"mod/common/units/GER_sub_units.txt",
		"mod/common/units/equipment/zz_GER_equipment.txt",

		"mod/localisation/english/replace/GER_equipment_l_english.yml",
		"mod/localisation/english/replace/GER_sub_units_l_english.yml",
	}
	if !eqStrings(got, want) {
		t.Fatalf("writes = %v, want %v", got, want)
	}
	if s := writeAt(t, env, "mod/common/units/equipment/zz_GER_equipment.txt"); !strings.HasPrefix(s, "equipments = {") {
		t.Fatalf("equipment script must open the `equipments` wrapper:\n%s", s)
	}
	if s := writeAt(t, env, "mod/common/units/GER_sub_units.txt"); !strings.HasPrefix(s, "sub_units = {") {
		t.Fatalf("sub-unit script must open the `sub_units` wrapper:\n%s", s)
	}
}
