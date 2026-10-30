package guirender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const headerSubUnits = `
sub_units = {
	infantry = {
		sprite = infantry  priority = 600  type = { infantry }
		need = { infantry_equipment = 100 }
	}
	militia = {
		sprite = infantry  priority = 400  type = { infantry }
		need = { infantry_equipment = 100 }
	}
	motorized = {
		sprite = motorized  priority = 599  type = { motorized }
		need = { infantry_equipment = 100 motorized_equipment = 35 }
	}
	light_armor = {
		sprite = light_armor  priority = 2501  type = { armor }
		need = { light_tank_chassis = 60 }
	}
	artillery_brigade = {
		sprite = artillery  priority = 1198  type = { infantry artillery }
		need = { artillery_equipment = 24 }
	}
	artillery = {
		sprite = artillery  priority = 0  group = support  type = { support }
		need = { artillery_equipment = 12 }
	}
	engineer = {
		sprite = infantry  priority = 0  group = support  type = { infantry support }
		need = { infantry_equipment = 10 support_equipment = 30 }
	}
	mod_gun = {
		priority = 5
		need = { mod_gun_chassis = 5 }
	}
}`

const headerEquipment = `
equipments = {
	infantry_equipment   = { year = 1936 is_archetype = yes type = infantry }
	motorized_equipment  = { year = 1936 is_archetype = yes type = { motorized } }
	light_tank_chassis   = { year = 1936 is_archetype = yes type = armor }
	artillery_equipment  = { year = 1936 is_archetype = yes type = { artillery infantry } }
	support_equipment    = { year = 1936 is_archetype = yes type = { infantry motorized support } }
	mod_gun_chassis      = { year = 1936 is_archetype = yes }
}`

const headerOOB = `
division_template = {
	name = "Infanterie-Division"
	division_names_group = TAG_Inf_01
	regiments = { infantry = { x = 0 y = 0 } infantry = { x = 0 y = 1 } infantry = { x = 1 y = 0 } }
	support = { engineer = { x = 0 y = 0 } artillery = { x = 0 y = 1 } }
}
division_template = {
	name = "Leichte Division"
	division_names_group = TAG_LTARM_01
	regiments = {
		light_armor = { x = 0 y = 0 }
		motorized = { x = 1 y = 0 } motorized = { x = 1 y = 1 }
		motorized = { x = 2 y = 0 } motorized = { x = 2 y = 1 }
	}
}
division_template = {
	name = "Infanterie-Division (mot.)"
	regiments = { motorized = { x = 0 y = 0 } motorized = { x = 0 y = 1 } }
}
division_template = {
	name = "Artillerie"
	regiments = { artillery_brigade = { x = 0 y = 0 } }
}
division_template = {
	name = "Afrikanische Schutztruppe"
	division_names_group = TAG_MIL_02
	template_counter = 98
	regiments = { militia = { x = 0 y = 0 } militia = { x = 0 y = 1 } }
}
division_template = {
	name = "Sicherungs-Division"
	template_counter = 7
	regiments = { infantry = { x = 0 y = 0 } }
	priority = 0
}
division_template = {
	name = "SS-Regiment"
	division_names_group = TAG_NOWHERE
	regiments = { motorized = { x = 0 y = 0 } }
	priority = 2
}
division_template = {
	name = "Mod-Division"
	regiments = { mod_gun = { x = 0 y = 0 } }
}`

const headerNames = `
# Division template historical names system.
TAG_Inf_01 = {
	name = "Infantry Divisions"
	for_countries = { TAG }
	can_use = { always = yes }
	division_types = { "infantry" }
	fallback_name = "%d. Infanterie-Division"
	ordered = {
		1 = { "%d. Infanterie-Division" }
		2 = { "%d. Infanterie-Division" "TAG_INF_2_DESC" }
	}
}
TAG_LTARM_01 = { name = "Light Divisions" ordered = { 1 = { "%d. Leichte Division" } } }
TAG_MIL_02 = { name = TAG_MIL_02_NAME }
`

const headerNamesAppend = `TAG_Inf_01 = { ordered = { 3 = { "%d. Infanterie-Division" } } }`

const headerIcons = `spriteTypes = {
	spriteType = { name = "GFX_unit_infantry_icon_medium" textureFile = "gfx/interface/counters/divisions_large/unit_infantry_icon.dds" noOfFrames = 2 }
	spriteType = { name = "GFX_unit_militia_icon_medium" textureFile = "gfx/interface/counters/divisions_large/unit_militia_icon.dds" noOfFrames = 2 }
	spriteType = { name = "GFX_unit_motorized_icon_medium" textureFile = "gfx/interface/counters/divisions_large/unit_motorized_icon.dds" noOfFrames = 2 }
	spriteType = { name = "GFX_unit_light_armor_icon_medium" textureFile = "gfx/interface/counters/divisions_large/unit_light_tank_icon.dds" noOfFrames = 2 }
	spriteType = { name = "GFX_div_templ_98_large" textureFile = "gfx/interface/counters/division_templates_large/custom_template_098.dds" }
}`

func headerFiles() map[string][]byte {
	return map[string][]byte{
		"common/units/infantry.txt":                                   []byte(headerSubUnits),
		"common/units/equipment/infantry.txt":                         []byte(headerEquipment),
		"common/units/names_divisions/TAG_names_divisions.txt":        []byte(headerNames),
		"common/units/names_divisions/ZZZ_append_names_divisions.txt": []byte(headerNamesAppend),
		"history/units/TAG_1939.txt":                                  []byte(headerOOB),
		"history/countries/TAG - Tagland.txt": []byte(
			"capital = 64\nset_politics = { ruling_party = fascism elections_allowed = no }\nset_oob = \"TAG_1939\"\n"),
		"interface/subuniticons.gfx": []byte(headerIcons),
		"localisation/english/equipment_l_english.yml": []byte("l_english:\n" +
			" EQUIPMENT_INFANTRY: \"Infantry\"\n EQUIPMENT_MOTORIZED: \"Truck\"\n" +
			" EQUIPMENT_ARMOR: \"Armor\"\n EQUIPMENT_ARTILLERY: \"Artillery\"\n" +
			" TAG_MIL_02_NAME: \"Schutztruppe Divisions\"\n"),
	}
}

func headerIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(headerFiles())
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestTemplateSymbolIsItsOwnTemplateCounterFirst(t *testing.T) {
	idx := headerIndex(t)
	icons := map[string]any{}
	templates, _ := idx.DivisionsPayload("TAG", "1939.8.14")["templates"].([]map[string]any)
	for _, template := range templates {
		icons[template["name"].(string)] = template["icon"]
	}

	if icons["Afrikanische Schutztruppe"] != "GFX_div_templ_98_large" {
		t.Errorf("Schutztruppe row icon = %v, want its template_counter's symbol", icons["Afrikanische Schutztruppe"])
	}
	if got := idx.DesignerPayload("TAG", "1939.8.14", "Afrikanische Schutztruppe")["icon"]; got != "GFX_div_templ_98_large" {
		t.Errorf("Schutztruppe designer icon = %v", got)
	}

	if icons["Sicherungs-Division"] != "GFX_unit_infantry_icon_medium" {
		t.Errorf("Sicherungs-Division icon = %v, want the lead battalion's counter", icons["Sicherungs-Division"])
	}
	if icons["Leichte Division"] != "GFX_unit_light_armor_icon_medium" {
		t.Errorf("the list and the designer disagree on Leichte Division: %v", icons["Leichte Division"])
	}
}

func TestDesignerNamesTheTemplatesNamesGroup(t *testing.T) {
	idx := headerIndex(t)
	payload := idx.DesignerPayload("TAG", "1939.8.14", "Infanterie-Division")

	if payload["names_group"] != "Infantry Divisions" || payload["division_names_group"] != "TAG_Inf_01" {
		t.Errorf("names group = %v (%v), want Infantry Divisions", payload["names_group"], payload["division_names_group"])
	}

	if got := idx.DesignerPayload("TAG", "1939.8.14", "Afrikanische Schutztruppe")["names_group"]; got != "Schutztruppe Divisions" {
		t.Errorf("localised names group = %v", got)
	}

	for _, template := range []string{"SS-Regiment", "Artillerie"} {
		if got := idx.DesignerPayload("TAG", "1939.8.14", template)["names_group"]; got != "" {
			t.Errorf("%s names group = %v, want none", template, got)
		}
	}

	if idx.subUnitFor("TAG_Inf_01") != nil {
		t.Error("a names group was indexed as a sub-unit")
	}
}

func TestAModRenamesANamesGroupOverTheBaseGame(t *testing.T) {
	base := headerIndex(t)
	mod, err := NewIndexWithFallback(map[string][]byte{
		"common/units/names_divisions/TAG_names_divisions.txt": []byte(`TAG_Inf_01 = { name = "Grenadier Divisions" }`),
	}, base)
	if err != nil {
		t.Fatalf("NewIndexWithFallback: %v", err)
	}
	if got := mod.divisionNamesGroup("TAG_Inf_01"); got != "Grenadier Divisions" {
		t.Errorf("mod title = %q, want the mod's", got)
	}
	if got := mod.divisionNamesGroup("TAG_LTARM_01"); got != "Light Divisions" {
		t.Errorf("base title through the mod = %q", got)
	}
}

func TestDesignerPriorityDefaultsToNormal(t *testing.T) {
	idx := headerIndex(t)
	for template, want := range map[string]int{
		"Infanterie-Division": 1,
		"Sicherungs-Division": 0,
		"SS-Regiment":         2,
	} {
		if got := idx.DesignerPayload("TAG", "1939.8.14", template)["priority"]; got != want {
			t.Errorf("%s priority = %v, want %d", template, got, want)
		}
	}

	empty := idx.DesignerPayload("ZZZ", "1939.8.14", "")
	if empty["priority"] != 1 || empty["icon"] != "" || empty["names_group"] != "" {
		t.Errorf("empty designer = priority %v icon %v names %v",
			empty["priority"], empty["icon"], empty["names_group"])
	}
}

func TestDesignerHeaderOnTheRealInstall(t *testing.T) {
	root := os.Getenv("GAME_ROOT")
	if root == "" {
		t.Skip("no GAME_ROOT")
	}
	idx := installDesignerIndex(t, root)

	infantry := idx.DesignerPayload("GER", "1936.1.1", "Infanterie-Division")
	for key, want := range map[string]any{
		"names_group":          "Infantry Divisions",
		"division_names_group": "GER_Inf_01",
		"priority":             1,
		"lead_unit":            "infantry",
		"auto_category":        "infantry_equipment",
		"auto_category_label":  "Infantry",
		"icon":                 "GFX_unit_infantry_icon_medium",
	} {
		if infantry[key] != want {
			t.Errorf("GER 1936 Infanterie-Division %s = %v, want %v", key, infantry[key], want)
		}
	}

	icons := map[string]any{}
	priorities := map[string]any{}
	templates, _ := idx.DivisionsPayload("GER", "1939.8.14")["templates"].([]map[string]any)
	for _, template := range templates {
		icons[template["name"].(string)] = template["icon"]
		priorities[template["name"].(string)] = template["priority"]
	}
	for name, want := range map[string]string{
		"Gebirgs-Division":           "GFX_unit_mountaineers_icon_medium",
		"Fallschirmjäger-Division":   "GFX_unit_paratrooper_icon_medium",
		"Infanterie-Division (mot.)": "GFX_unit_motorized_icon_medium",
		"Leichte Division":           "GFX_unit_light_armor_icon_medium",
		"Kavallerie-Brigade":         "GFX_unit_cavalry_icon_medium",
	} {
		if icons[name] != want {
			t.Errorf("%s icon = %v, want %s", name, icons[name], want)
		}
	}
	for name, want := range map[string]int{"Infanterie-Division": 1, "Sicherungs-Division": 0, "SS-Regiment": 2} {
		if priorities[name] != want {
			t.Errorf("%s priority = %v, want %d", name, priorities[name], want)
		}
	}

	support, _ := infantry["support"].([]map[string]any)
	want := map[string]string{"engineer": "GFX_unit_engineer_icon_medium", "artillery": "GFX_unit_artillery_icon_medium"}
	if len(support) != len(want) {
		t.Fatalf("GER 1936 Infanterie-Division support = %v", support)
	}
	for _, cell := range support {
		if kind, _ := cell["type"].(string); cell["icon"] != want[kind] {
			t.Errorf("%s cell icon = %v, want %s", kind, cell["icon"], want[kind])
		}
	}
	if got := idx.divisionNamesGroup("GER_MIL_02"); got != "Schutztruppe Divisions" {
		t.Errorf("GER_MIL_02 = %q, want the Schutztruppe row's title", got)
	}

	motorised := idx.DesignerPayload("GER", "1939.8.14", "Infanterie-Division (mot.)")
	t.Logf("mot.: lead %v auto %v (%v)", motorised["lead_unit"], motorised["auto_category"], motorised["auto_category_label"])
	panzer := idx.DesignerPayload("GER", "1939.8.14", "Panzer-Division")
	t.Logf("panzer: lead %v auto %v (%v) names %v", panzer["lead_unit"], panzer["auto_category"], panzer["auto_category_label"], panzer["names_group"])
}

func installDesignerIndex(t *testing.T, root string) *Index {
	t.Helper()
	bases := []string{root}
	for _, group := range []string{"dlc", "integrated_dlc"} {
		entries, err := os.ReadDir(filepath.Join(root, group))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				bases = append(bases, filepath.Join(root, group, entry.Name()))
			}
		}
	}
	files := map[string][]byte{}
	for _, base := range bases {
		for _, sub := range []string{"common/units", "history/units", "history/countries", "interface", "localisation/english"} {
			_ = filepath.Walk(filepath.Join(base, filepath.FromSlash(sub)), func(p string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				switch strings.ToLower(filepath.Ext(p)) {
				case ".txt", ".yml", ".gfx":
				default:
					return nil
				}
				rel, err := filepath.Rel(base, p)
				if err != nil {
					return nil
				}
				if body, err := os.ReadFile(p); err == nil {
					files[filepath.ToSlash(rel)] = body
				}
				return nil
			})
		}
	}
	idx, err := NewIndex(files)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}
