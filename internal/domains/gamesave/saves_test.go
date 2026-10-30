package gamesave

import (
	"strings"
	"testing"

	"haganeworks/internal/domains/gameview"
	"haganeworks/internal/project"
)

func saveMap(p *project.Project, fn func(st *store) (map[string]any, error)) (map[string]any, error) {
	return run(p, func(st *store) (map[string]any, error) {
		out, err := fn(st)
		return st.raw.withUndo(out), err
	})
}

func TestSaveDecisionOverrideTakesTheFileInOnce(t *testing.T) {
	p := newProject(t)
	base := index(t, map[string]string{"common/decisions/political_actions.txt": decisionFileText})
	out, err := saveMap(p, func(st *store) (map[string]any, error) {
		return saveDecisionOverride(st, base, decisionSaveInput{Category: "political_actions", Name: "improved_worker_conditions", Cost: floatPtr(75)})
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["file"] != "common/decisions/political_actions.txt" || out["imported"] != true || out["undo"] == nil {
		t.Fatalf("answer = %v", out)
	}
	files := rawFiles(t, p)
	if len(files) != 1 || files[0].RawKind != "verbatim_override" || files[0].Note != decisionRawFileNote {
		t.Fatalf("raw files = %+v", files)
	}
	saved := files[0].Content
	if !strings.Contains(saved, "cost = 75") || !strings.Contains(saved, "ban_party = { cost = 50 }") {
		t.Fatalf("saved file = %s", saved)
	}

	top := indexOver(t, map[string]string{"common/decisions/political_actions.txt": saved}, base)
	out, err = saveMap(p, func(st *store) (map[string]any, error) {
		return saveDecisionOverride(st, top, decisionSaveInput{Category: "political_actions", Name: "improved_worker_conditions", DaysReEnable: intPtr(60)})
	})
	if err != nil || out["imported"] != false {
		t.Fatalf("second save: %v %v", out, err)
	}
	if files := rawFiles(t, p); len(files) != 1 || !strings.Contains(files[0].Content, "cost = 75") || !strings.Contains(files[0].Content, "days_re_enable = 60") {
		t.Fatalf("second save lost the first: %+v", files)
	}
	_, err = saveMap(p, func(st *store) (map[string]any, error) {
		return saveDecisionOverride(st, base, decisionSaveInput{Category: "political_actions", Name: "nope", Cost: floatPtr(1)})
	})
	if statusOf(err) != 404 {
		t.Fatalf("an unknown decision: %v", err)
	}
}

type fakeDesignIndex struct{}

func (fakeDesignIndex) DesignSlotNames(typeName string) ([]string, bool) {
	if typeName == "light_tank_chassis_1" {
		return []string{"main_armament_slot", "turret_type_slot", "engine_type_slot"}, true
	}
	return nil, false
}

func (fakeDesignIndex) KnownEquipmentModule(name string) bool {
	return name == "tank_small_cannon" || name == "tank_light_one_man_tank_turret" || name == "tank_gasoline_engine"
}

func TestDesignSaveGoesIntoTheCountrysRow(t *testing.T) {
	p := newProject(t)
	ids := seed(t, p, historyFilesTable,
		project.Row{"country_tag": "ITA", "raw_script": "", "source": "created"},
		project.Row{"country_tag": "GER", "raw_script": designHistory, "source": "imported"})
	save := func(index designSaveIndex, in designSaveInput) (map[string]any, error) {
		return run(p, func(st *store) (map[string]any, error) { return saveDesign(st, index, in) })
	}
	in := designSaveInput{Tag: "ger", Name: "Panzer III", Type: "light_tank_chassis_1", Modules: designModules()}
	out, err := save(fakeDesignIndex{}, in)
	germany := rows(t, p, historyFilesTable)[1]
	if err != nil || out["added"] != true || out["history_file_id"] != ids[1] || !strings.Contains(text(germany, "raw_script"), `name = "Panzer III"`) {
		t.Fatalf("save: %v %v %v", out, err, germany)
	}
	if text(germany, "source") != "created" {
		t.Errorf("an edited imported row stays imported: %v", germany["source"])
	}

	bad := in
	bad.Modules = []designSaveModule{{"wing_slot", "tank_small_cannon"}}
	if _, err := save(fakeDesignIndex{}, bad); statusOf(err) != 422 {
		t.Errorf("bad slot: %v", err)
	}

	gone, err := save(nil, designSaveInput{Tag: "GER", Original: "Panzer III", Delete: true})
	if err != nil || gone["deleted"] != true || strings.Contains(text(rows(t, p, historyFilesTable)[1], "raw_script"), "Panzer III") {
		t.Errorf("delete: %v %v", gone, err)
	}
	other := in
	other.Tag = "FRA"
	if _, err := save(fakeDesignIndex{}, other); statusOf(err) != 409 {
		t.Errorf("no row: %v", err)
	}
}

type oobUnitIndexFake struct{}

func (oobUnitIndexFake) OOBSource(name string) ([]byte, bool) {
	switch name {
	case "GER_1936_naval":
		return []byte(germanFleets), true
	case "GER_1936":
		return []byte(germanWings), true
	}
	return nil, false
}

func TestOOBUnitSaveTakesTheFileIn(t *testing.T) {
	p := newProject(t)
	save := func(in oobUnitSaveInput) (map[string]any, error) {
		return saveMap(p, func(st *store) (map[string]any, error) { return saveOOBUnit(st, oobUnitIndexFake{}, in) })
	}
	name := "Kampfgeschwader 51"
	out, err := save(oobUnitSaveInput{Kind: "air_wing", OOB: "GER_1936", Ordinal: 2, Name: &name})
	files := rawFiles(t, p)
	if err != nil || out["file"] != "history/units/GER_1936.txt" || len(files) != 1 || files[0].RawKind != "verbatim_override" {
		t.Fatalf("save: %v %v %+v", out, err, files)
	}
	if !strings.Contains(files[0].Content, `name = "Kampfgeschwader 51"`) {
		t.Errorf("content:\n%s", files[0].Content)
	}
	for _, wrong := range []oobUnitSaveInput{
		{Kind: "division", OOB: "GER_1936", Name: &name},
		{Kind: "air_wing", OOB: "GER_1936", Amount: floatPtr(12.5)},
		{Kind: "air_wing", OOB: "GER_1936", Amount: floatPtr(0)},
		{Kind: "task_force", OOB: "GER_1936_naval", Amount: floatPtr(10)},
		{Kind: "task_force", OOB: "GER_1936_naval"},
		{Kind: "task_force", OOB: "../GER_1936_naval", Name: &name},
		{Kind: "task_force", OOB: "GER_1936_naval", Ordinal: 5, Name: &name},
	} {
		if _, err := save(wrong); statusOf(err) != 422 {
			t.Errorf("%+v: %v", wrong, err)
		}
	}
}

func TestOOBUnitSaveIntoTheWorkbenchRows(t *testing.T) {
	p := newProject(t)
	fleet := seed(t, p, fleetTable, project.Row{"oob_file": "GER_1936_naval", "name": "Hochseeflotte", "ordering": int64(0)})[0]
	forces := seed(t, p, taskForceTable,
		project.Row{"fleet_id": fleet, "name": "Hochseeflotte", "ordering": int64(0)},
		project.Row{"fleet_id": fleet, "name": `"Ostseeflotte"`, "ordering": int64(1)})
	groups := seed(t, p, airWingGroupTable,
		project.Row{"oob_file": "GER_1936", "location": "54", "ordering": int64(0)},
		project.Row{"oob_file": "GER_1936", "location": "64", "ordering": int64(1)})
	wings := seed(t, p, airWingTable,
		project.Row{"group_id": groups[0], "equipment_type": "medium_plane_airframe_0", "amount": "40", "ordering": int64(0)},
		project.Row{"group_id": groups[1], "equipment_type": "small_plane_airframe_0", "amount": "40", "ordering": int64(0)},
		project.Row{"group_id": groups[1], "equipment_type": "small_plane_cas_airframe_0", "amount": "24", "ordering": int64(1)})
	save := func(in oobUnitSaveInput) (map[string]any, error) {
		return saveMap(p, func(st *store) (map[string]any, error) { return saveOOBUnit(st, oobUnitIndexFake{}, in) })
	}
	name := "Ostseegeschwader"
	if out, err := save(oobUnitSaveInput{Kind: "task_force", OOB: "GER_1936_naval", Ordinal: 1, RawName: "Ostseeflotte", Name: &name}); err != nil || out["task_force"] != forces[1] {
		t.Fatalf("task force: %v %v", out, err)
	}
	if got := text(rows(t, p, taskForceTable)[1], "name"); got != name {
		t.Errorf("task force name = %q", got)
	}
	if out, err := save(oobUnitSaveInput{Kind: "air_wing", OOB: "GER_1936", Ordinal: 1, State: 64, StateIndex: 1, Amount: floatPtr(30)}); err != nil || out["air_wing"] != wings[2] {
		t.Fatalf("wing: %v %v", out, err)
	}
	if got := text(rows(t, p, airWingTable)[2], "amount"); got != "30" {
		t.Errorf("wing amount = %q", got)
	}
	if _, err := save(oobUnitSaveInput{Kind: "task_force", OOB: "GER_1936_naval", RawName: "Mittelmeer", Name: &name}); statusOf(err) != 404 {
		t.Errorf("unknown task force: %v", err)
	}
	if len(rawFiles(t, p)) != 0 {
		t.Errorf("a file came in: %+v", rawFiles(t, p))
	}

	q := newProject(t)
	fleet = seed(t, q, fleetTable, project.Row{"oob_file": "GER_1936_naval", "name": "Hochseeflotte", "source": "imported"})[0]
	seed(t, q, taskForceTable, project.Row{"fleet_id": fleet, "name": "Hochseeflotte", "source": "imported"})
	out, err := saveMap(q, func(st *store) (map[string]any, error) {
		return saveOOBUnit(st, oobUnitIndexFake{}, oobUnitSaveInput{Kind: "task_force", OOB: "GER_1936_naval", RawName: "Hochseeflotte", Name: &name})
	})
	if err != nil || out["imported"] != true || text(rows(t, q, taskForceTable)[0], "name") != "Hochseeflotte" {
		t.Errorf("imported rows: %v %v", out, err)
	}
}

type subUnitIndexFake struct{}

func (subUnitIndexFake) SubUnitSource(id string) (string, []byte, bool) {
	if id != "infantry" && id != "bicycle_battalion" {
		return "", nil, false
	}
	return "common/units/infantry.txt", []byte(infantryFile), true
}

func TestSubUnitSaveTakesTheFileIn(t *testing.T) {
	p := newProject(t)
	save := func(in subUnitSaveInput) (map[string]any, error) {
		return saveMap(p, func(st *store) (map[string]any, error) { return saveSubUnit(st, subUnitIndexFake{}, in) })
	}
	out, err := save(subUnitSaveInput{ID: "bicycle_battalion", Stats: map[string]float64{"max_organisation": 45}})
	if err != nil || out["file"] != "common/units/infantry.txt" || len(rawFiles(t, p)) != 1 {
		t.Fatalf("save: %v %v", out, err)
	}
	if content := rawFiles(t, p)[0].Content; !strings.Contains(content, "max_organisation = 45") || !strings.Contains(content, "max_organisation = 60") {
		t.Errorf("content:\n%s", content)
	}
	for _, wrong := range []subUnitSaveInput{
		{ID: "infantry"},
		{ID: "infantry", Stats: map[string]float64{"priority": 1}},
		{ID: "infantry", Stats: map[string]float64{"manpower": -1}},
		{ID: "infantry", Need: map[string]float64{"infantry_equipment": 12.5}},
		{ID: "infantry", Need: map[string]float64{"support_equipment": 10}},
		{ID: "../infantry", Stats: map[string]float64{"manpower": 900}},
	} {
		if _, err := save(wrong); statusOf(err) != 422 {
			t.Errorf("%+v: %v", wrong, err)
		}
	}
}

func TestSubUnitSaveIntoTheWorkbenchRow(t *testing.T) {
	p := newProject(t)
	ids := seed(t, p, subUnitTable,
		project.Row{"sub_unit_id": "infantry", "source": "imported", "source_file": "a", "stats": map[string]any{"combat_width": 2.0}},
		project.Row{"sub_unit_id": "Infantry", "source": "created", "source_file": "b", "unit_types": []any{"infantry"},
			"stats": map[string]any{"combat_width": 2.0, "weight": 0.5}, "need": map[string]any{"infantry_equipment": 100.0}})
	out, err := saveMap(p, func(st *store) (map[string]any, error) {
		return saveSubUnit(st, subUnitIndexFake{}, subUnitSaveInput{ID: "infantry", Stats: map[string]float64{"combat_width": 3}, Need: map[string]float64{"infantry_equipment": 80}})
	})
	if err != nil || out["sub_unit"] != ids[1] || len(rawFiles(t, p)) != 0 {
		t.Fatalf("save: %v %v", out, err)
	}
	var saved subUnitRow
	decodeColumn(rows(t, p, subUnitTable)[1], &saved)
	if saved.Stats["combat_width"] != 3.0 || saved.Stats["weight"] != 0.5 || saved.Need["infantry_equipment"] != 80.0 {
		t.Errorf("row = %+v", saved)
	}

	q := newProject(t)
	seed(t, q, subUnitTable, project.Row{"sub_unit_id": "infantry", "source": "created", "unit_types": []any{"cavalry"}})
	if _, err := saveMap(q, func(st *store) (map[string]any, error) {
		return saveSubUnit(st, subUnitIndexFake{}, subUnitSaveInput{ID: "infantry", Stats: map[string]float64{"combat_width": 3}})
	}); statusOf(err) != 422 {
		t.Errorf("bad classification: %v", err)
	}

	r := newProject(t)
	seed(t, r, subUnitTable, project.Row{"sub_unit_id": "infantry", "source": "imported"})
	if out, err := saveMap(r, func(st *store) (map[string]any, error) {
		return saveSubUnit(st, subUnitIndexFake{}, subUnitSaveInput{ID: "infantry", Stats: map[string]float64{"combat_width": 3}})
	}); err != nil || out["imported"] != true {
		t.Errorf("imported row: %v %v", out, err)
	}
}

const projectsFile = `sp_naval_support_ships = {
	specialization = specialization_naval

	allowed = {
		#FREE
	}

	available = {
		FROM = {
			has_tech = transport
		}
	}

	breakthrough_cost = {
		specialization_naval = 1
	}

	prototype_time = sp_time.prototype.medium

	complexity = sp_complexity.medium
}
`

type fakeProjectIndex struct{ text string }

func (f fakeProjectIndex) SpecialProjectSource(name string) (string, []byte, int, bool) {
	if !strings.HasPrefix(f.text, name+" = {") {
		return "", nil, 0, false
	}
	return "common/special_projects/projects/naval_projects.txt", []byte(f.text), 1, true
}

func TestSpecialProjectSaveRewritesItsBlockOnly(t *testing.T) {
	p := newProject(t)
	save := func(in specialProjectSaveInput) (map[string]any, error) {
		return saveMap(p, func(st *store) (map[string]any, error) {
			return saveSpecialProject(st, fakeProjectIndex{projectsFile}, in)
		})
	}
	cost := map[string]float64{"specialization_naval": 2, "specialization_air": 1}
	out, err := save(specialProjectSaveInput{ID: "sp_naval_support_ships", BreakthroughCost: &cost, PrototypeTime: strPtr("sp_time.prototype.short"), Available: strPtr("")})
	if err != nil || out["imported"] != true || len(rawFiles(t, p)) != 1 {
		t.Fatalf("save: %v %v", out, err)
	}
	got := rawFiles(t, p)[0].Content
	for _, want := range []string{
		"prototype_time = sp_time.prototype.short", "complexity = sp_complexity.medium", "#FREE",
		"breakthrough_cost = {\n\t\tspecialization_air = 1\n\t\tspecialization_naval = 2\n\t}",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "has_tech = transport") || strings.Contains(got, "available") {
		t.Errorf("available left in:\n%s", got)
	}
	if _, err := save(specialProjectSaveInput{ID: "sp_naval_support_ships", PrototypeTime: strPtr("sp_time.prototype.short ) x")}); statusOf(err) != 422 {
		t.Errorf("bad constant: %v", err)
	}
	half := map[string]float64{"specialization_naval": 1.5}
	if _, err := save(specialProjectSaveInput{ID: "sp_naval_support_ships", BreakthroughCost: &half}); statusOf(err) != 422 {
		t.Errorf("half a point: %v", err)
	}
}

func TestSpecialProjectSaveIntoTheWorkbenchRow(t *testing.T) {
	p := newProject(t)
	ids := seed(t, p, specialProjectsTable, project.Row{
		"project_token": "sp_naval_support_ships", "source": "created",
		"raw_extras": map[string]any{"ai_will_do": map[string]any{"base": 1.0}, "breakthrough_cost": map[string]any{"specialization_naval": 1.0}},
	})
	cost := map[string]float64{"specialization_naval": 3}
	out, err := saveMap(p, func(st *store) (map[string]any, error) {
		return saveSpecialProject(st, fakeProjectIndex{projectsFile}, specialProjectSaveInput{
			ID: "sp_naval_support_ships", BreakthroughCost: &cost, Complexity: strPtr("sp_complexity.large"),
		})
	})
	if err != nil || out["row"] != ids[0] || len(rawFiles(t, p)) != 0 {
		t.Fatalf("row save: %v %v", out, err)
	}
	row := rows(t, p, specialProjectsTable)[0]
	extras, _ := row["raw_extras"].(map[string]any)
	breakthrough, _ := extras["breakthrough_cost"].(map[string]any)
	if text(row, "complexity") != "sp_complexity.large" || extras["ai_will_do"] == nil || number(breakthrough["specialization_naval"]) != 3 {
		t.Errorf("row = %v", row)
	}
}

const superHeavyFile = `sp_naval_super_heavy_battleship = {
	specialization = specialization_naval
	project_output = {
		country_effects = {
			if = {
				limit = {
				NOT = {has_dlc = "Man the Guns"}
			}
			custom_effect_tooltip = {
				localization_key = SP_UNLOCK_TECH
				TECH = heavy_battleship2
				}
			}
			else = {
				add_design_template_bonus = {
					name = sp_naval_super_heavy_battleship_free_design
					uses = 1
					cost_factor = 1
					equipment = ship_hull_super_heavy_1
				}
			}
		}
		enable_equipments = {
			ship_hull_super_heavy_1
		}
	}
}
`

func TestSpecialProjectSaveDesignDiscount(t *testing.T) {
	p := newProject(t)
	bonus := &designBonusInput{Name: "sp_naval_super_heavy_battleship_free_design", Uses: 2, CostFactor: 0.5}
	save := func(p *project.Project, bonus *designBonusInput) (map[string]any, error) {
		return saveMap(p, func(st *store) (map[string]any, error) {
			return saveSpecialProject(st, fakeProjectIndex{superHeavyFile}, specialProjectSaveInput{ID: "sp_naval_super_heavy_battleship", DesignBonus: bonus})
		})
	}
	if _, err := save(p, bonus); err != nil || len(rawFiles(t, p)) != 1 {
		t.Fatalf("save: %v", err)
	}
	want := strings.Replace(strings.Replace(superHeavyFile, "uses = 1", "uses = 2", 1), "cost_factor = 1", "cost_factor = 0.5", 1)
	if got := rawFiles(t, p)[0].Content; got != want {
		t.Errorf("file =\n%s", got)
	}
	for _, wrong := range []*designBonusInput{
		{Name: "sp_something_else", Uses: 1, CostFactor: 1},
		{Name: "sp_naval_super_heavy_battleship_free_design", Uses: 0, CostFactor: 1},
		{Name: "sp_naval_super_heavy_battleship_free_design", Uses: 1, CostFactor: 1.5},
	} {
		if _, err := save(newProject(t), wrong); statusOf(err) != 422 {
			t.Errorf("%+v: %v", *wrong, err)
		}
	}

	q := newProject(t)
	ids := seed(t, q, specialProjectsTable, project.Row{
		"project_token": "sp_naval_super_heavy_battleship", "source": "created",
		"project_output": "country_effects = {\n\tadd_design_template_bonus = {\n\t\tname = sp_naval_super_heavy_battleship_free_design\n\t\tuses = 1\n\t\tcost_factor = 1\n\t}\n}",
	})
	if out, err := save(q, bonus); err != nil || out["row"] != ids[0] {
		t.Fatalf("row save: %v %v", out, err)
	}
	if got := text(rows(t, q, specialProjectsTable)[0], "project_output"); got != "country_effects = {\n\tadd_design_template_bonus = {\n\t\tname = sp_naval_super_heavy_battleship_free_design\n\t\tuses = 2\n\t\tcost_factor = 0.5\n\t}\n}" {
		t.Errorf("row project_output = %q", got)
	}
}

const intelFile = `# List of names and logos for intelligence agencies

intelligence_agency = {

	picture = GFX_intelligence_agency_logo_ger
	names = { "Abwehr" }

	default = {
		tag = GER
		NOT = { has_government = communism }
	}
	available = {
		original_tag = GER
		NOT = { has_government = communism }
	}
}

intelligence_agency = {

	picture = GFX_intelligence_agency_logo_rom
	names = { "Serviciul Special de Informatii" }

	available = { always = yes }

	default = { tag = ROM }
	available = { original_tag = ROM }
}
`

const intelPath = "common/intelligence_agencies/00_intelligence_agencies.txt"

type fakeIntelIndex struct{}

func (fakeIntelIndex) IntelAgencySource(path string, ordinal int) (string, []byte, int, bool) {
	lines := map[int]int{0: 3, 1: 18}
	line, ok := lines[ordinal]
	if path != intelPath || !ok {
		return "", nil, 0, false
	}
	return path, []byte(intelFile), line, true
}

func saveIntel(p *project.Project, in intelSaveInput) (map[string]any, error) {
	return saveMap(p, func(st *store) (map[string]any, error) { return saveIntelAgency(st, fakeIntelIndex{}, in) })
}

func TestIntelSaveRewritesItsBlockOnly(t *testing.T) {
	p := newProject(t)
	names := []string{"Abwehr", "Amt Ausland"}
	out, err := saveIntel(p, intelSaveInput{File: intelPath, Ordinal: 0, Picture: "GFX_intelligence_agency_logo_ger",
		OldNames: []string{"Abwehr"}, Names: &names, Available: strPtr(""), Default: strPtr("tag = GER")})
	if err != nil || out["imported"] != true || len(rawFiles(t, p)) != 1 {
		t.Fatalf("save: %v %v", out, err)
	}
	got := rawFiles(t, p)[0].Content
	for _, want := range []string{
		`names = { "Abwehr" "Amt Ausland" }`, "default = { tag = GER }",
		"# List of names and logos", `names = { "Serviciul Special de Informatii" }`, "available = { original_tag = ROM }",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "original_tag = GER") {
		t.Errorf("available not taken out:\n%s", got)
	}
}

func TestIntelSaveKeepsOneAvailable(t *testing.T) {
	p := newProject(t)
	if _, err := saveIntel(p, intelSaveInput{File: intelPath, Ordinal: 1, Picture: "GFX_intelligence_agency_logo_rom",
		Available: strPtr("original_tag = ROM\nhas_government = fascism")}); err != nil {
		t.Fatal(err)
	}
	got := rawFiles(t, p)[0].Content
	if strings.Contains(got, "always = yes") || !strings.Contains(got, "available = {\n\t\toriginal_tag = ROM\n\t\thas_government = fascism\n\t}") {
		t.Errorf("text:\n%s", got)
	}

	if _, err := saveIntel(p, intelSaveInput{File: intelPath, Ordinal: 1, Picture: "GFX_intelligence_agency_logo_ger", Default: strPtr("")}); statusOf(err) != 409 {
		t.Errorf("changed file: %v", err)
	}

	if _, err := saveIntel(p, intelSaveInput{File: intelPath, Ordinal: 1, Picture: "GFX_intelligence_agency_logo_rom", Default: strPtr("tag = { ROM")}); statusOf(err) != 422 {
		t.Errorf("bad script: %v", err)
	}
}

func TestIntelSaveIntoTheWorkbenchRow(t *testing.T) {
	p := newProject(t)
	ids := seed(t, p, agenciesTable,
		project.Row{"agency_id": "ger_sss", "picture": "GFX_intelligence_agency_logo_generic_2", "names": []any{"SSS"}, "source": "created", "available": "tag = GER"},
		project.Row{"agency_id": "abwehr", "picture": "GFX_intelligence_agency_logo_ger", "names": []any{"Abwehr"}, "source": "imported"})
	names := []string{"SD"}
	out, err := saveIntel(p, intelSaveInput{File: "common/intelligence_agencies/hagane_intelligence_agencies.txt", Ordinal: 0,
		Picture: "GFX_intelligence_agency_logo_generic_2", OldNames: []string{"SSS"}, Names: &names})
	if err != nil || out["row"] != ids[0] || len(rawFiles(t, p)) != 0 {
		t.Fatalf("row save: %v %v", out, err)
	}
	row := rows(t, p, agenciesTable)[0]
	if got := agencyNames(row); len(got) != 1 || got[0] != "SD" || text(row, "available") != "tag = GER" {
		t.Errorf("row = %v", row)
	}
	if _, err := saveIntel(p, intelSaveInput{File: intelPath, Ordinal: 0, Picture: "GFX_intelligence_agency_logo_ger",
		OldNames: []string{"Abwehr"}, Default: strPtr("tag = GER")}); err != nil || len(rawFiles(t, p)) != 1 {
		t.Errorf("imported row: %v", err)
	}
	if text(rows(t, p, agenciesTable)[1], "default_trigger") != "" {
		t.Errorf("the imported row changed: %v", rows(t, p, agenciesTable)[1])
	}
}

func TestIntelCreateForTheCountry(t *testing.T) {
	p := newProject(t)
	seed(t, p, agenciesTable, project.Row{"agency_id": "ger_sss"})
	create := func(in intelCreateInput) (map[string]any, error) {
		return run(p, func(st *store) (map[string]any, error) { return createIntelAgency(st, in) })
	}
	out, err := create(intelCreateInput{Tag: "ger", Picture: "GFX_intelligence_agency_logo_generic_2", Name: "SSS"})
	if err != nil || out["agency_id"] != "ger_sss_2" {
		t.Fatalf("create: %v %v", out, err)
	}
	made := rows(t, p, agenciesTable)[1]
	if text(made, "available") != "original_tag = GER" || text(made, "country_tag") != "GER" || agencyNames(made)[0] != "SSS" ||
		text(made, "picture") != "GFX_intelligence_agency_logo_generic_2" || text(made, "scope_kind") != "own" || text(made, "source") != "created" ||
		out["row"] != project.Int64(made["id"]) {
		t.Errorf("row = %v", made)
	}
	if _, err := create(intelCreateInput{Tag: "GER", Picture: "../x.dds", Name: "SSS"}); statusOf(err) != 422 {
		t.Errorf("bad picture: %v", err)
	}
}

const upgradesFile = `branch_defense = {

	upgrade_passive_defense = {
		picture = GFX_agency_passive_defense
		frame = GFX_upgrade_frame_defense

		modifiers_during_progress = {
			civilian_factory_use = 5
		}

		level = {
			modifier = {
				foreign_intel_network_decay_factor = 0.1
			}
		}
		level = {
			modifier = {
				foreign_intel_network_decay_factor = 0.1
			}
		}
	}

	upgrade_anti_partisan = {
		picture = GFX_agency_anti_partisan
		modifiers_during_progress = {
			civilian_factory_use = 5
		}
		level = {
			modifier = { root_out_resistance_effectiveness_factor = 0.25 }
		}
	}
}
`

type fakeUpgradeIndex struct{}

func (fakeUpgradeIndex) IntelUpgradeSource(name string) (string, []byte, int, bool) {
	switch name {
	case "upgrade_passive_defense":
		return "common/intelligence_agency_upgrades/intelligence_agency_upgrades.txt", []byte(upgradesFile), 3, true
	case "upgrade_anti_partisan":
		return "common/intelligence_agency_upgrades/intelligence_agency_upgrades.txt", []byte(upgradesFile), 22, true
	}
	return "", nil, 0, false
}

func TestIntelUpgradeSaveRewritesItsBlockOnly(t *testing.T) {
	p := newProject(t)
	save := func(p *project.Project, in intelUpgradeSaveInput) (map[string]any, error) {
		return saveMap(p, func(st *store) (map[string]any, error) { return saveIntelUpgrade(st, fakeUpgradeIndex{}, in) })
	}
	levels := []string{"modifier = {\n\tforeign_intel_network_decay_factor = 0.3\n}"}
	out, err := save(p, intelUpgradeSaveInput{ID: "upgrade_passive_defense", Factories: floatPtr(8), Available: strPtr("has_tech = radio"), Levels: &levels})
	if err != nil || out["imported"] != true || len(rawFiles(t, p)) != 1 {
		t.Fatalf("save: %v %v", out, err)
	}
	got := rawFiles(t, p)[0].Content
	passive := got[:strings.Index(got, "upgrade_anti_partisan")]
	for _, want := range []string{"civilian_factory_use = 8", "available = { has_tech = radio }", "foreign_intel_network_decay_factor = 0.3"} {
		if !strings.Contains(passive, want) {
			t.Errorf("missing %q in:\n%s", want, passive)
		}
	}
	if strings.Count(passive, "level = {") != 1 || strings.Contains(passive, "0.1") {
		t.Errorf("levels not replaced:\n%s", passive)
	}
	if rest := got[strings.Index(got, "upgrade_anti_partisan"):]; rest != upgradesFile[strings.Index(upgradesFile, "upgrade_anti_partisan"):] {
		t.Errorf("anti-partisan changed:\n%s", rest)
	}
	for _, wrong := range []intelUpgradeSaveInput{
		{ID: "upgrade_passive_defense"},
		{ID: "upgrade_passive_defense", Levels: &[]string{}},
		{ID: "upgrade_passive_defense", Levels: &[]string{"modifier = {"}},
		{ID: "upgrade_passive_defense", Factories: floatPtr(-1)},
		{ID: "upgrade_nothing", Factories: floatPtr(8)},
	} {
		if _, err := save(newProject(t), wrong); statusOf(err) != 422 && statusOf(err) != 404 {
			t.Errorf("%+v: %v", wrong, err)
		}
	}
}

func TestIntelUpgradeSaveIntoTheWorkbenchRow(t *testing.T) {
	p := newProject(t)
	ids := seed(t, p, agenciesTable,
		project.Row{"agency_id": "a", "source": "imported", "upgrade_branches": []any{map[string]any{"branch": "branch_defense", "upgrades": []any{map[string]any{"token": "upgrade_mine"}}}}},
		project.Row{"agency_id": "b", "source": "created", "upgrade_branches": []any{map[string]any{"branch": "branch_defense", "upgrades": []any{
			map[string]any{"token": "upgrade_other"},
			map[string]any{"token": "upgrade_mine", "modifiers_during_progress": "civilian_factory_use = 5", "levels": []any{"modifier = { x = 1 }"}},
		}}}})
	levels := []string{"modifier = { x = 2 }", "modifier = { x = 3 }"}
	out, err := saveMap(p, func(st *store) (map[string]any, error) {
		return saveIntelUpgrade(st, fakeUpgradeIndex{}, intelUpgradeSaveInput{ID: "upgrade_mine", Factories: floatPtr(12), Levels: &levels})
	})
	if err != nil || out["row"] != ids[1] {
		t.Fatalf("row save: %v %v", out, err)
	}
	var branches []map[string]any
	decodeColumn(rows(t, p, agenciesTable)[1]["upgrade_branches"], &branches)
	upgrades := branches[0]["upgrades"].([]any)
	mine := upgrades[1].(map[string]any)
	if upgrades[0].(map[string]any)["token"] != "upgrade_other" || mine["modifiers_during_progress"] != "civilian_factory_use = 12" || len(mine["levels"].([]any)) != 2 {
		t.Errorf("row upgrades = %v", upgrades)
	}
	if text(rows(t, p, agenciesTable)[0], "source") != "imported" {
		t.Error("the imported agency changed")
	}
}

const colonyFile = `autonomy_state = {
	id = autonomy_colony

	is_puppet = yes

	min_freedom_level = 0.60

	manpower_influence = 0.7

	rule = {
		can_not_declare_war = yes
		can_decline_call_to_war = no
	}

	modifier = {
		autonomy_manpower_share = 0.4
		extra_trade_to_overlord_factor = 0.5
	}
}
`

type fakeAutonomyIndex struct{}

func (fakeAutonomyIndex) AutonomySource(id string) (string, []byte, int, bool) {
	if id != "autonomy_colony" {
		return "", nil, 0, false
	}
	return "common/autonomous_states/colony.txt", []byte(colonyFile), 1, true
}

func TestAutonomySaveRewritesItsBlockOnly(t *testing.T) {
	p := newProject(t)
	save := func(in autonomySaveInput) (map[string]any, error) {
		return saveMap(p, func(st *store) (map[string]any, error) { return saveAutonomy(st, fakeAutonomyIndex{}, in) })
	}
	out, err := save(autonomySaveInput{ID: "autonomy_colony", Values: map[string]*float64{"min_freedom_level": floatPtr(0.55), "manpower_influence": nil},
		Rule: strPtr("can_not_declare_war = yes\ncan_decline_call_to_war = yes"), Modifier: strPtr("")})
	if err != nil || out["imported"] != true || len(rawFiles(t, p)) != 1 {
		t.Fatalf("save: %v %v", out, err)
	}
	got := rawFiles(t, p)[0].Content
	for _, want := range []string{"min_freedom_level = 0.55", "is_puppet = yes", "\t\tcan_decline_call_to_war = yes\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, gone := range []string{"manpower_influence", "modifier", "extra_trade_to_overlord_factor", "0.60"} {
		if strings.Contains(got, gone) {
			t.Errorf("%q left in:\n%s", gone, got)
		}
	}

	if _, err := save(autonomySaveInput{ID: "autonomy_colony", Values: map[string]*float64{"min_freedom_level": floatPtr(1.5)}}); statusOf(err) != 422 {
		t.Errorf("out of range: %v", err)
	}
}

func TestAutonomySaveIntoTheWorkbenchRow(t *testing.T) {
	p := newProject(t)
	ids := seed(t, p, autonomyStatesTable, project.Row{"autonomy_id": "autonomy_colony", "source": "created", "manpower_influence": 0.7})
	save := func(in autonomySaveInput) (map[string]any, error) {
		return saveMap(p, func(st *store) (map[string]any, error) { return saveAutonomy(st, fakeAutonomyIndex{}, in) })
	}
	out, err := save(autonomySaveInput{ID: "autonomy_colony", Values: map[string]*float64{"min_freedom_level": floatPtr(0.65), "manpower_influence": nil},
		Modifier: strPtr("extra_trade_to_overlord_factor = 0.25\nautonomy_manpower_share = 1")})
	if err != nil || out["row"] != ids[0] || len(rawFiles(t, p)) != 0 {
		t.Fatalf("row save: %v %v", out, err)
	}
	row := rows(t, p, autonomyStatesTable)[0]
	modifier, _ := row["modifier"].(map[string]any)
	if project.Int64(row["min_freedom_level"]) != 650 || row["manpower_influence"] != nil ||
		number(modifier["extra_trade_to_overlord_factor"]) != 0.25 || number(modifier["autonomy_manpower_share"]) != 1 {
		t.Errorf("row = %v", row)
	}
	if _, err := save(autonomySaveInput{ID: "autonomy_colony", Modifier: strPtr("custom_modifier_tooltip = { x = 1 }")}); statusOf(err) != 422 {
		t.Errorf("nested modifier: %v", err)
	}
}

const wargoalFile = `wargoal_types = {

	take_state = {
		# PREV = original target country
		allowed = {

		}
		generate_base_cost = 200
		generate_per_state_cost = 50

		take_states_limit = 5
		threat = 2
	}
}
`

type fakeWargoalIndex struct{}

func (fakeWargoalIndex) WargoalSource(name string) (string, []byte, int, bool) {
	if name != "take_state" {
		return "", nil, 0, false
	}
	return "common/wargoals/00_invasion.txt", []byte(wargoalFile), 3, true
}

func saveWargoalIn(p *project.Project, days float64) (map[string]any, error) {
	return saveMap(p, func(st *store) (map[string]any, error) {
		return saveWargoal(st, fakeWargoalIndex{}, wargoalSaveInput{Name: "take_state", Values: map[string]*float64{"generate_base_cost": &days}})
	})
}

func TestWargoalSaveRewritesItsNumbersOnly(t *testing.T) {
	p := newProject(t)
	out, err := saveWargoalIn(p, 150)
	if err != nil || out["imported"] != true || len(rawFiles(t, p)) != 1 {
		t.Fatalf("save: %v %v", out, err)
	}
	if text := rawFiles(t, p)[0].Content; !strings.Contains(text, "generate_base_cost = 150") || !strings.Contains(text, "generate_per_state_cost = 50") ||
		!strings.Contains(text, "# PREV = original target country") {
		t.Errorf("text:\n%s", text)
	}

	if _, err := saveMap(p, func(st *store) (map[string]any, error) {
		return saveWargoal(st, fakeWargoalIndex{}, wargoalSaveInput{Name: "take_state", Values: map[string]*float64{"expire": floatPtr(1)}})
	}); statusOf(err) != 422 {
		t.Errorf("other key: %v", err)
	}
}

func TestWargoalSaveIntoTheWorkbenchRow(t *testing.T) {
	p := newProject(t)
	ids := seed(t, p, wargoalsTable, project.Row{"wargoal_key": "take_state", "source": "created",
		"safe_scalars": map[string]any{"generate_base_cost": "200", "expire": "60"}})
	out, err := saveMap(p, func(st *store) (map[string]any, error) {
		return saveWargoal(st, fakeWargoalIndex{}, wargoalSaveInput{Name: "take_state",
			Values: map[string]*float64{"generate_base_cost": floatPtr(150), "take_states_limit": floatPtr(3)}})
	})
	if err != nil || out["row"] != ids[0] || len(rawFiles(t, p)) != 0 {
		t.Fatalf("save: %v %v", out, err)
	}
	scalars := map[string]string{}
	decodeColumn(rows(t, p, wargoalsTable)[0]["safe_scalars"], &scalars)
	if scalars["generate_base_cost"] != "150" || scalars["take_states_limit"] != "3" || scalars["expire"] != "60" {
		t.Errorf("scalars = %v", scalars)
	}
	q := newProject(t)
	seed(t, q, wargoalsTable, project.Row{"wargoal_key": "take_state", "source": "imported", "safe_scalars": map[string]any{}})
	if out, err := saveWargoalIn(q, 150); err != nil || out["imported"] != true || len(rawFiles(t, q)) != 1 {
		t.Errorf("imported row: %v %v", out, err)
	}
}

const tinyMIOFile = "common/military_industrial_organization/organizations/TAG_organization.txt"

func TestMIOTakeBringsTheWholeFileInOnce(t *testing.T) {
	p := newProject(t)
	idx := index(t, map[string]string{tinyMIOFile: "@x = 3\n" +
		"TAG_one_organization = {\n\tallowed = { original_tag = TAG }\n" +
		"\tinitial_trait = { name = TAG_one_trait equipment_bonus = { soft_attack = 0.05 } }\n" +
		"\tadd_trait = { token = TAG_far name = TAG_far position = { x = @x y = 0 } }\n}\n" +
		"TAG_two_organization = {\n\tinclude = generic_organization\n\tremove_trait = generic_trait\n}\n"})
	take := func(id string) (string, int, error) {
		var count int
		file, err := run(p, func(st *store) (string, error) {
			file, n, err := takeMIOFileIn(st, idx, id)
			count = n
			return file, err
		})
		return file, count, err
	}
	file, count, err := take("TAG_two_organization")
	if err != nil || file != tinyMIOFile || count != 2 {
		t.Fatalf("take = %q, %d, %v; want the whole file, 2 organizations", file, count, err)
	}
	made := rows(t, p, mioTable)
	for _, row := range made {
		extras, _ := row["raw_extras"].(map[string]any)
		if extras[mioSourceKey] != tinyMIOFile || text(row, "source") != "created" {
			t.Errorf("%s: marker %v, source %v", text(row, "mio_id"), extras[mioSourceKey], row["source"])
		}
	}

	var traits []map[string]any
	decodeColumn(made[0]["traits"], &traits)
	if extras, _ := traits[1]["raw_extras"].(map[string]any); !strings.Contains(extras["position"].(string), "x = 3") {
		t.Errorf("TAG_far = %v", traits[1])
	}
	if text(made[0], "country_tag") != "TAG" {
		t.Errorf("country = %q", text(made[0], "country_tag"))
	}

	if again, n, err := take("TAG_one_organization"); err != nil || n != 0 || again != tinyMIOFile {
		t.Fatalf("second take = %q %d, %v", again, n, err)
	}
	if _, _, err := take("nobody_organization"); statusOf(err) != 422 {
		t.Fatalf("an organization no file declares: %v", err)
	}
}

func undoSteps(t *testing.T, out map[string]any) []rawFileStep {
	t.Helper()
	undo, _ := out["undo"].(map[string]any)
	steps, _ := undo["files"].([]rawFileStep)
	if len(steps) != 1 {
		t.Fatalf("undo steps: %#v", out["undo"])
	}
	return steps
}

func undo(p *project.Project, steps []rawFileStep) error {
	_, err := run(p, func(st *store) (struct{}, error) { return struct{}{}, undoRawFiles(st, steps) })
	return err
}

func TestUndoRemovesAFileTheSaveTookIn(t *testing.T) {
	p := newProject(t)
	out, err := saveWargoalIn(p, 150)
	if err != nil {
		t.Fatal(err)
	}
	steps := undoSteps(t, out)
	if steps[0].Before != nil || len(rawFiles(t, p)) != 1 {
		t.Fatalf("taken in: %#v", steps[0])
	}
	if err := undo(p, steps); err != nil {
		t.Fatal(err)
	}
	if files := rawFiles(t, p); len(files) != 0 {
		t.Errorf("still in the project: %#v", files)
	}

	if err := undo(p, steps); err != nil {
		t.Errorf("second undo: %v", err)
	}
}

func TestUndoPutsAFileBackWordForWord(t *testing.T) {
	p := newProject(t)
	before := seedRaw(t, p, gameview.RawFileInsert{DestPath: "common/wargoals/00_invasion.txt", Content: wargoalFile, RawKind: "verbatim_override", Note: "导入的"})
	out, err := saveWargoalIn(p, 150)
	if err != nil {
		t.Fatal(err)
	}
	steps := undoSteps(t, out)
	if saved := rawFiles(t, p)[0]; !strings.Contains(saved.Content, "generate_base_cost = 150") || saved.Note != "导入的" {
		t.Fatalf("saved: %+v", saved)
	}
	if err := undo(p, steps); err != nil {
		t.Fatal(err)
	}
	if after := rawFiles(t, p)[0]; after.Content != before.Content || after.Note != before.Note || after.DestPath != before.DestPath {
		t.Errorf("not put back: %#v", after)
	}
}

func TestUndoLeavesAFileChangedSinceAlone(t *testing.T) {
	p := newProject(t)
	seedRaw(t, p, gameview.RawFileInsert{DestPath: "common/wargoals/00_invasion.txt", Content: wargoalFile, RawKind: "verbatim_override"})
	out, err := saveWargoalIn(p, 150)
	if err != nil {
		t.Fatal(err)
	}
	steps := undoSteps(t, out)
	later := rawFiles(t, p)[0]
	setRawContent(t, p, later, later.Content+"\n# edited in the workbench\n")
	if err := undo(p, steps); statusOf(err) != 409 {
		t.Fatalf("want 409, got %v", err)
	}
	if got := rawFiles(t, p)[0].Content; !strings.HasSuffix(got, "# edited in the workbench\n") {
		t.Errorf("touched anyway: %s", got)
	}
	if err := undo(p, nil); statusOf(err) != 422 {
		t.Errorf("nothing to undo: %v", err)
	}
}
