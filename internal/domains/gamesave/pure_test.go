package gamesave

import (
	"errors"
	"strings"
	"testing"
)

const hqFixture = `# Default Army HQ templates
every_possible_country = {
	limit = { has_dlc = "Thunder at Our Gates" }

	# GER
	if = {
		limit = { tag = GER }
		division_template = {
			name = "AOK"
			template_counter = 121
			is_army_hq = yes

			regiments = {
				hq_light_armor = { x = 0 y = 0 }
				hq_light_armor = { x = 0 y = 1 }
			}

			support = {
				hq_support_company = { x = 0 y = 0 }
				# hq_logistics added via on_research_complete
			}
		}
	}

	if = {
		limit = { NOT = { tag = GER } }
		division_template = {
			localization_key = "ARMY_HQ_TEMPLATE_NAME"
			is_army_hq = yes
			regiments = { hq_infantry = { x = 0 y = 0 } }
		}
	}
}
`

func TestHQTemplateEditRewritesOnlyThatTemplate(t *testing.T) {
	got, err := hqTemplateEdit(hqFixture, 8, "AOK", "Heeresgruppe",
		[]designerSaveCell{{"hq_light_armor", 0, 1}, {"hq_light_armor", 0, 0}, {"hq_medium_armor", 0, 2}},
		[]designerSaveCell{{"hq_support_company", 0, 0}, {"hq_air_liaison", 0, 1}})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(strings.Replace(strings.Replace(hqFixture, `name = "AOK"`, `name = "Heeresgruppe"`, 1),
		"\t\t\t\thq_light_armor = { x = 0 y = 1 }\n\t\t\t}", "\t\t\t\thq_light_armor = { x = 0 y = 1 }\n\t\t\t\thq_medium_armor = { x = 0 y = 2 }\n\t\t\t}", 1),
		"\t\t\t\thq_support_company = { x = 0 y = 0 }\n\t\t\t\t# hq_logistics added via on_research_complete\n\t\t\t}",
		"\t\t\t\thq_support_company = { x = 0 y = 0 }\n\t\t\t\thq_air_liaison = { x = 0 y = 1 }\n\t\t\t}", 1)
	if got != want {
		t.Errorf("edited file:\n%s\nwant:\n%s", got, want)
	}
}

func TestHQTemplateEditAddsAndDropsBlocks(t *testing.T) {

	got, err := hqTemplateEdit(hqFixture, 27, "Army Headquarters", "Army Headquarters",
		[]designerSaveCell{{"hq_infantry", 0, 0}}, []designerSaveCell{{"hq_support_company", 0, 0}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "regiments = {\n\t\t\t\thq_infantry = { x = 0 y = 0 }\n\t\t\t}\n\t\t\tsupport = {\n\t\t\t\thq_support_company = { x = 0 y = 0 }\n\t\t\t}\n\t\t}") {
		t.Errorf("support not added before the template closes:\n%s", got)
	}
	if _, err := hqTemplateEdit(hqFixture, 27, "Army Headquarters", "Stab", []designerSaveCell{{"hq_infantry", 0, 0}}, nil); err == nil {
		t.Error("the generic template was renamed")
	}
	dropped, err := hqTemplateEdit(hqFixture, 8, "AOK", "AOK", []designerSaveCell{{"hq_light_armor", 0, 0}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dropped, "hq_support_company") || !strings.Contains(dropped, "is_army_hq = yes\n\n\t\t\tregiments") {
		t.Errorf("support not dropped cleanly:\n%s", dropped)
	}
	if _, err := hqTemplateEdit(hqFixture, 200, "AOK", "AOK", nil, nil); err == nil {
		t.Error("a line past the file found a template")
	}
}

const rawOOB = `# imported mod battle order
division_template = {
	name = "Infanterie-Division"
	division_names_group = GER_Inf_01
	regiments = {
		infantry = { x = 0 y = 0 }
	}
}

division_template = {
	name = "Panzer-Division"
	division_names_group = GER_Arm_01

	regiments = {
		light_armor = { x = 0 y = 0 }
		light_armor = { x = 0 y = 1 }
		motorized = { x = 1 y = 0 }
	}
	support = {
		engineer = { x = 0 y = 0 }
	}
}

units = {
	division = {
		location = 6521
		division_template = "Panzer-Division"
	}
	division = {
		location = 6334
		division_template = "Infanterie-Division"
	}
}
`

func TestOOBTemplateEditChangesOnlyThatTemplate(t *testing.T) {
	out, err := oobTemplateEdit(rawOOB, designerSaveInput{
		Template: "Panzer-Division", NamesGroup: "GER_Arm_02", Priority: intPtr(2),
		Regiments: []designerSaveCell{{"light_armor", 0, 0}, {"medium_armor", 1, 0}},
		Support:   []designerSaveCell{{"recon", 0, 0}},
	}, "Leichte-Division")
	if err != nil {
		t.Fatal(err)
	}
	templates := oobTemplates(out)
	if len(templates) != 2 || templates[0].name != "Infanterie-Division" || templates[1].name != "Leichte-Division" {
		t.Fatalf("templates = %+v", templates)
	}
	infantry := out[:strings.Index(out, "Leichte-Division")]
	if !strings.Contains(infantry, "infantry = { x = 0 y = 0 }") || !strings.Contains(infantry, "GER_Inf_01") {
		t.Errorf("the other template changed:\n%s", infantry)
	}
	for _, want := range []string{"medium_armor = { x = 1 y = 0 }", "recon = { x = 0 y = 0 }",
		"division_names_group = GER_Arm_02", "priority = 2", `division_template = "Leichte-Division"`,
		`division_template = "Infanterie-Division"`, "# imported mod battle order"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, gone := range []string{"motorized", "engineer", `"Panzer-Division"`, "GER_Arm_01"} {
		if strings.Contains(out, gone) {
			t.Errorf("%q still in:\n%s", gone, out)
		}
	}
}

func TestOOBTemplateEditAddsANewTemplateBeforeTheUnits(t *testing.T) {
	out, err := oobTemplateEdit(rawOOB, designerSaveInput{
		NamesGroup: "GER_Mot_01", Regiments: []designerSaveCell{{"motorized", 0, 0}},
	}, "Schnelle-Division")
	if err != nil {
		t.Fatal(err)
	}
	at, units := strings.Index(out, `name = "Schnelle-Division"`), strings.Index(out, "units = {")
	if at < 0 || at > units || !strings.Contains(out, "division_names_group = GER_Mot_01") {
		t.Fatalf("new template not before the units:\n%s", out)
	}
	if strings.Contains(out[at:units], "priority") {
		t.Error("a default priority is not written")
	}
	if len(oobTemplates(out)) != 3 {
		t.Errorf("templates = %+v", oobTemplates(out))
	}
}

func TestOOBTemplateEditRefusals(t *testing.T) {
	cases := []struct {
		in     designerSaveInput
		name   string
		status int
	}{
		{designerSaveInput{Template: "Panzer-Division", Regiments: []designerSaveCell{{"infantry", 0, 0}}}, "Infanterie-Division", 409},
		{designerSaveInput{Regiments: []designerSaveCell{{"infantry", 0, 0}}}, "Panzer-Division", 409},
		{designerSaveInput{Template: "Gebirgs-Division", Regiments: []designerSaveCell{{"infantry", 0, 0}}}, "Gebirgs-Division", 404},
		{designerSaveInput{Template: "Panzer-Division", NamesGroup: "a b", Regiments: []designerSaveCell{{"infantry", 0, 0}}}, "Panzer-Division", 422},
	}
	for _, c := range cases {
		if _, err := oobTemplateEdit(rawOOB, c.in, c.name); statusOf(err) != c.status {
			t.Errorf("%+v -> %v, want %d", c.in, err, c.status)
		}
	}
}

func TestTemplateScalarEditAddsAMissingKeyAfterTheName(t *testing.T) {
	out, err := templateScalarEdit(rawOOB, oobTemplates(rawOOB)[0].line, "priority", "3", "1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\tname = \"Infanterie-Division\"\n\tpriority = 3\n") {
		t.Errorf("priority not after the name:\n%s", out[:200])
	}

	same, err := templateScalarEdit(rawOOB, oobTemplates(rawOOB)[0].line, "priority", "1", "1")
	if err != nil || same != rawOOB {
		t.Errorf("default priority written: %v", err)
	}
}

func TestRenameTemplateRefsIsExact(t *testing.T) {
	in := `division_template = "Panzer-Division" division_template="Panzer-Division (mot.)" DIVISION_TEMPLATE = "panzer-division"`
	got := renameTemplateRefs(in, "Panzer-Division", "Leichte $1")
	want := `division_template = "Leichte $1" division_template="Panzer-Division (mot.)" DIVISION_TEMPLATE = "Leichte $1"`
	if got != want {
		t.Errorf("renameTemplateRefs = %s", got)
	}
}

const decisionFileText = `# Political actions, as the game writes them
political_actions = {
	improved_worker_conditions = {
		icon = generic_industry
		cost = 100
		available = { has_war = no }
		complete_effect = { add_stability = 0.05 }
	}
	ban_party = { cost = 50 }
}

other_tab = {
	improved_worker_conditions = { cost = 1 }
}
`

func TestDecisionFieldsEditChangesOneBlock(t *testing.T) {
	out, err := decisionFieldsEdit(decisionFileText, "political_actions", "improved_worker_conditions", []decisionScalar{
		{"cost", "150"}, {"icon", "generic_political_actions"}, {"days_re_enable", "30"}, {"fire_only_once", "yes"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"\t\ticon = generic_political_actions\n\t\tcost = 150\n",
		"\t\tdays_re_enable = 30\n\t\tfire_only_once = yes\n\t}\n\tban_party = { cost = 50 }",
		"available = { has_war = no }",
		"# Political actions, as the game writes them",
		"other_tab = {\n\timproved_worker_conditions = { cost = 1 }\n}",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("edited file lacks %q:\n%s", want, out)
		}
	}

	out, err = decisionFieldsEdit(decisionFileText, "political_actions", "ban_party", []decisionScalar{{"days_remove", "10"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ban_party = { cost = 50 days_remove = 10 }") {
		t.Fatalf("one-line block = %s", out)
	}
	if _, err := decisionFieldsEdit(decisionFileText, "political_actions", "missing", []decisionScalar{{"cost", "1"}}); err == nil {
		t.Fatal("a decision the file does not declare must be refused")
	}

	if _, err := decisionFieldsEdit(decisionFileText, "political_actions", "improved_worker_conditions", []decisionScalar{{"available", "1"}}); err == nil {
		t.Fatal("a script block must not be overwritten by a value")
	}
}

func TestDecisionSaveScalarsRefusesBadInput(t *testing.T) {
	for _, in := range []decisionSaveInput{
		{Category: "tab", Name: "x"},
		{Category: "tab x", Name: "x", Cost: floatPtr(1)},
		{Category: "tab", Name: "x", Cost: floatPtr(-1)},
		{Category: "tab", Name: "x", Icon: strPtr("a b")},
		{Category: "tab", Name: "x", DaysRemove: intPtr(-2)},
	} {
		if _, err := decisionSaveScalars(in); err == nil {
			t.Errorf("%+v should be refused", in)
		}
	}
}

const infantryText = `equipments = {
	infantry_equipment = {
		year = 1918
		is_archetype = yes
		#Offensive Abilities
		soft_attack = 3
		build_cost_ic = 0.43
		resources = {
			steel = 1
		}
	}

	infantry_equipment_1 = {
		year = 1936

		archetype = infantry_equipment
		priority = 10

		#Defensive Abilities
		defense = 22
		breakthrough = 3

		#Offensive Abilities
		soft_attack = 6
		build_cost_ic = 0.50
	}

	# Improved weapons  ~1939
	infantry_equipment_2 = {
		year = 1939
		soft_attack = 9
	}
}
`

func TestEquipmentEditChangesDropsAndAddsItsOwnLines(t *testing.T) {
	out, err := equipmentEdit(infantryText, 13, "infantry_equipment_1", map[string]*float64{
		"soft_attack":  floatPtr(7),
		"breakthrough": nil,
		"reliability":  floatPtr(0.9),
		"year":         floatPtr(1937),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"\t\tyear = 1937\n",
		"\t\tsoft_attack = 7\n",
		"\t\tdefense = 22\n\n\t\t#Offensive Abilities",
		"\t\tbuild_cost_ic = 0.50\n\t\treliability = 0.9\n\t}",

		"infantry_equipment_2 = {\n\t\tyear = 1939\n\t\tsoft_attack = 9\n\t}",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if strings.Contains(out, "breakthrough") {
		t.Errorf("breakthrough stayed:\n%s", out)
	}
	if strings.Count(out, "soft_attack = 3") != 1 {
		t.Error("the archetype's own soft_attack changed")
	}
}

func TestEquipmentEditResources(t *testing.T) {

	out, err := equipmentEdit(infantryText, 13, "infantry_equipment_1", nil, map[string]*float64{"steel": floatPtr(2), "chromium": nil})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\t\tbuild_cost_ic = 0.50\n\t\tresources = { steel = 2 }\n\t}") {
		t.Errorf("no resources line:\n%s", out)
	}

	changed, err := equipmentEdit(infantryText, 2, "infantry_equipment", nil, map[string]*float64{"steel": floatPtr(3)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(changed, "resources = {\n\t\t\tsteel = 3\n\t\t}") {
		t.Errorf("steel not changed:\n%s", changed)
	}
	dropped, err := equipmentEdit(infantryText, 2, "infantry_equipment", nil, map[string]*float64{"steel": nil})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dropped, "resources") || !strings.Contains(dropped, "build_cost_ic = 0.43\n\t}") {
		t.Errorf("resources block not dropped:\n%s", dropped)
	}
}

func TestEquipmentEditNeedsTheBlockOnItsLine(t *testing.T) {
	if _, err := equipmentEdit(infantryText, 2, "infantry_equipment_1", map[string]*float64{"soft_attack": floatPtr(1)}, nil); err == nil {
		t.Error("an equipment not on its line was edited")
	}
	if err := validEquipmentNumbers(map[string]*float64{"Soft Attack": floatPtr(1)}); err == nil {
		t.Error("a key with spaces passed")
	}
}

const designHistory = `set_research_slots = 3
if = {
	limit = { has_dlc = "No Step Back" }
	create_equipment_variant = {
		name = "Panzer II"
		type = light_tank_chassis_1
		parent_version = 0 # kept
		modules = {
			main_armament_slot = tank_small_cannon
			turret_type_slot = tank_light_one_man_tank_turret
		}
		upgrades = {
			tank_nsb_engine_upgrade = 1
		}
		icon = "GFX_GER_basic_light_tank_medium"
	}
}
`

func designModules() []designSaveModule {
	return []designSaveModule{
		{"main_armament_slot", ""},
		{"turret_type_slot", "tank_light_one_man_tank_turret"},
		{"engine_type_slot", "tank_gasoline_engine"},
	}
}

func TestDesignSaveRewritesADesignInsideAnIf(t *testing.T) {
	out, added, err := designEdit(designHistory, designSaveInput{
		Tag: "GER", Original: "Panzer II", Name: "Panzer II Ausf. B", Type: "light_tank_chassis_1",
		Modules: designModules(), Upgrades: map[string]int{"tank_nsb_armor_upgrade": 2},
	})
	if err != nil || added {
		t.Fatalf("edit: %v %v", added, err)
	}
	for _, want := range []string{
		`name = "Panzer II Ausf. B"`,
		"\t\tmodules = {\n\t\t\tmain_armament_slot = empty\n\t\t\tturret_type_slot = tank_light_one_man_tank_turret\n\t\t\tengine_type_slot = tank_gasoline_engine\n\t\t}",
		"\t\tupgrades = {\n\t\t\ttank_nsb_armor_upgrade = 2\n\t\t}",
		"parent_version = 0 # kept",
		`icon = "GFX_GER_basic_light_tank_medium"`,
		"set_research_slots = 3",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if strings.Contains(out, "tank_nsb_engine_upgrade") {
		t.Errorf("old upgrades kept:\n%s", out)
	}
}

func TestDesignSaveAddsAndDeletes(t *testing.T) {
	out, added, err := designEdit(designHistory, designSaveInput{
		Tag: "GER", Name: "Panzer III", Type: "light_tank_chassis_1", Modules: designModules(),
	})
	if err != nil || !added {
		t.Fatalf("add: %v %v", added, err)
	}
	if !strings.HasSuffix(out, "create_equipment_variant = {\n\tname = \"Panzer III\"\n\ttype = light_tank_chassis_1\n\tmodules = {\n\t\tmain_armament_slot = empty\n\t\tturret_type_slot = tank_light_one_man_tank_turret\n\t\tengine_type_slot = tank_gasoline_engine\n\t}\n}\n") {
		t.Errorf("added:\n%s", out)
	}

	back, _, err := designEdit(out, designSaveInput{Tag: "GER", Original: "Panzer III", Delete: true})
	if err != nil || back != designHistory {
		t.Errorf("add then delete: %v\n%q", err, back)
	}
	bare := strings.TrimSuffix(designHistory, "\n") + "\n## 1939 Start ##"
	grown, _, _ := designEdit(bare, designSaveInput{Tag: "GER", Name: "Panzer III", Type: "light_tank_chassis_1", Modules: designModules()})
	if back, _, err := designEdit(grown, designSaveInput{Tag: "GER", Original: "Panzer III", Delete: true}); err != nil || back != bare {
		t.Errorf("add then delete after a comment: %v\n%q", err, back)
	}
	gone, _, err := designEdit(out, designSaveInput{Tag: "GER", Original: "Panzer II", Delete: true})
	if err != nil || strings.Contains(gone, "Panzer II\"") || !strings.Contains(gone, "Panzer III") || !strings.Contains(gone, "has_dlc") {
		t.Errorf("delete: %v\n%s", err, gone)
	}
	if _, _, err := designEdit(designHistory, designSaveInput{Tag: "GER", Original: "Tiger", Name: "Tiger", Type: "light_tank_chassis_1"}); err == nil {
		t.Error("a design the history does not have was rewritten")
	}

	if _, _, err := designEdit(out, designSaveInput{Tag: "GER", Name: "Panzer II", Type: "light_tank_chassis_1", Modules: designModules()}); statusOf(err) != 409 {
		t.Errorf("same name added: %v", err)
	}
	if _, _, err := designEdit(out, designSaveInput{Tag: "GER", Original: "Panzer III", Name: "Panzer II", Type: "light_tank_chassis_1", Modules: designModules()}); statusOf(err) != 409 {
		t.Errorf("renamed onto another: %v", err)
	}
}

const germanUnits = `units = {
	division = {
		division_name = {
			is_name_ordered = yes
			name_order = 25
		}
		location = 6334
		division_template = "Infanterie-Division"
		start_experience_factor = 0.3
	}
	division = {
		division_name = {
			is_name_ordered = yes
			name_order = 1
		}
		location = 537
		division_template = "Panzer-Division"
		start_experience_factor = 0.3
	}
}`

func TestDivisionEditRewritesItsBlockOnly(t *testing.T) {
	name, template := "Panzer-Lehr-Division", "Infanterie-Division"
	got, err := divisionEdit(germanUnits, 1, divisionSaveInput{Name: &name, Template: &template, Experience: floatPtr(0.75), Equipment: floatPtr(0.5)})
	if err != nil {
		t.Fatal(err)
	}
	first := germanUnits[:strings.Index(germanUnits, "division = {\n\t\tdivision_name = {\n\t\t\tis_name_ordered = yes\n\t\t\tname_order = 1")]
	if !strings.HasPrefix(got, first) {
		t.Errorf("first division changed:\n%s", got)
	}
	second := got[len(first):]
	for _, want := range []string{`name = "Panzer-Lehr-Division"`, `division_template = "Infanterie-Division"`, "start_experience_factor = 0.75", "start_equipment_factor = 0.5", "location = 537"} {
		if !strings.Contains(second, want) {
			t.Errorf("missing %q in:\n%s", want, second)
		}
	}
	if strings.Contains(second, "name_order") {
		t.Errorf("numbered name kept:\n%s", second)
	}
	if _, err := divisionEdit(germanUnits, 2, divisionSaveInput{Name: &name}); !errors.Is(err, errDivisionMissing) {
		t.Errorf("third division: %v", err)
	}

	officer, gone := "Guderian", ""
	withOfficer, err := divisionEdit(germanUnits, 0, divisionSaveInput{Officer: &officer, Location: intPtr(42)})
	if err != nil || !strings.Contains(withOfficer, `officer = { name = "Guderian" }`) || !strings.Contains(withOfficer, "location = 42") {
		t.Fatalf("officer: %v\n%s", err, withOfficer)
	}
	if without, _ := divisionEdit(withOfficer, 0, divisionSaveInput{Officer: &gone}); strings.Contains(without, "officer") {
		t.Errorf("officer not taken out:\n%s", without)
	}
	removed, err := divisionEdit(germanUnits, 0, divisionSaveInput{Remove: true})
	if err != nil || strings.Count(removed, "division = {") != 1 || strings.Contains(removed, "6334") {
		t.Errorf("remove: %v\n%s", err, removed)
	}
}

func TestTailDivisionOrdinal(t *testing.T) {
	added := "division = { division_template = \"Neu\" }\n"
	source := added + germanUnits
	if got, err := tailDivisionOrdinal(source, germanUnits, 2); err != nil || got != 1 {
		t.Errorf("own division = %d, %v", got, err)
	}
	if _, err := tailDivisionOrdinal(source, germanUnits, 0); !errors.Is(err, errDivisionAdded) {
		t.Errorf("added division: %v", err)
	}
	if _, err := tailDivisionOrdinal("units = { }", germanUnits, 0); !errors.Is(err, errTailNotRead) {
		t.Errorf("stale read: %v", err)
	}
}

const germanFleets = `units = {
	fleet = {
		name = "Hochseeflotte"
		naval_base = 241
		task_force = {
			name = "Hochseeflotte"
			location = 241
			ship = { name = "Deutschland" definition = heavy_cruiser equipment = { ship_hull_cruiser_1 = { amount = 1 owner = GER } } }
		}
		task_force = {
			location = 6389
			ship = { name = "U-1" definition = submarine equipment = { ship_hull_submarine_1 = { amount = 1 owner = GER } } }
		}
	}
}
`

const germanWings = `air_wings = {
	64 = {
		small_plane_airframe_0 = {
			owner = "GER"
			amount = 40
		}
		name = "Jagdgeschwader 1"
		small_plane_cas_airframe_0 = {
			owner = "GER"
			amount = 24
		}
	}
	# Munich
	54 = {
		medium_plane_airframe_0 = {
			owner = "GER"
			amount = 40
		}
	}
}
`

func TestTaskForceEditRenamesItsBlockOnly(t *testing.T) {
	got, err := taskForceEdit(germanFleets, 0, "Flottenverband Nord")
	if err != nil || !strings.Contains(got, `name = "Flottenverband Nord"`+"\n\t\t\tlocation = 241") || !strings.Contains(got, `name = "Hochseeflotte"`+"\n\t\tnaval_base") {
		t.Fatalf("first: %v\n%s", err, got)
	}
	got, err = taskForceEdit(germanFleets, 1, "U-Flottille Weddigen")
	if err != nil || !strings.Contains(got, `name = "U-Flottille Weddigen"`) || strings.Count(got, `name = "Hochseeflotte"`) != 2 {
		t.Fatalf("second: %v\n%s", err, got)
	}
	if _, err := taskForceEdit(germanFleets, 2, "x"); !errors.Is(err, errOOBUnitMissing) {
		t.Errorf("third: %v", err)
	}
}

func TestAirWingEditSetsAmountAndName(t *testing.T) {
	name, amount := "Jagdgeschwader Richthofen", 60.0
	got, err := airWingEdit(germanWings, 0, &name, &amount)
	if err != nil || !strings.Contains(got, "amount = 60\n\t\t}\n\t\tname = \"Jagdgeschwader Richthofen\"") || !strings.Contains(got, "amount = 24") {
		t.Fatalf("first: %v\n%s", err, got)
	}
	stuka := "Sturzkampfgeschwader 2"
	got, err = airWingEdit(germanWings, 1, &stuka, nil)
	if err != nil || !strings.Contains(got, "amount = 24\n\t\t}\n\t\tname = \"Sturzkampfgeschwader 2\"\n\t}") || !strings.Contains(got, `name = "Jagdgeschwader 1"`) {
		t.Fatalf("second: %v\n%s", err, got)
	}
	got, err = airWingEdit(germanWings, 2, nil, &amount)
	if err != nil || !strings.Contains(got, "medium_plane_airframe_0 = {\n\t\t\towner = \"GER\"\n\t\t\tamount = 60") || strings.Count(got, "name =") != 1 {
		t.Fatalf("third: %v\n%s", err, got)
	}
	if _, err := airWingEdit(germanWings, 3, &name, nil); !errors.Is(err, errOOBUnitMissing) {
		t.Errorf("fourth: %v", err)
	}
}

const infantryFile = `sub_units = {

	infantry = {
		abbreviation = "INF"
		combat_width = 2

		#Size Definitions
		max_strength = 25
		max_organisation = 60
		manpower = 1000

		need = {
			infantry_equipment = 100
		}
	}

	bicycle_battalion = {
		combat_width = 2
		max_organisation = 60
		need = {
			infantry_equipment = 100
		}
	}
}
`

func TestSubUnitEditRewritesItsBlockOnly(t *testing.T) {
	got, err := subUnitEdit(infantryFile, "infantry", map[string]float64{"combat_width": 3, "max_organisation": 70, "default_morale": 0.4}, map[string]float64{"infantry_equipment": 120})
	if err != nil {
		t.Fatal(err)
	}
	infantry, bicycle, _ := strings.Cut(got, "bicycle_battalion")
	for _, want := range []string{"combat_width = 3", "max_organisation = 70", "\t\tdefault_morale = 0.4\n", "infantry_equipment = 120", "#Size Definitions", "manpower = 1000"} {
		if !strings.Contains(infantry, want) {
			t.Errorf("missing %q in:\n%s", want, infantry)
		}
	}
	if !strings.Contains(bicycle, "combat_width = 2") || !strings.Contains(bicycle, "infantry_equipment = 100") {
		t.Errorf("bicycles changed:\n%s", bicycle)
	}
	if _, err := subUnitEdit(infantryFile, "infantry", nil, map[string]float64{"support_equipment": 10}); err == nil {
		t.Error("equipment it does not need was set")
	}
	if _, err := subUnitEdit(infantryFile, "cavalry", map[string]float64{"combat_width": 2}, nil); !errors.Is(err, errSubUnitMissing) {
		t.Errorf("cavalry: %v", err)
	}
}
