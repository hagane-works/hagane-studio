package guirender

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestDesignerProductionCostOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)

	one := idx.DesignerPayloadFor("GER", "1936.1.1.12", "-", "r:0:0:infantry")
	checkRange(t, "one rifle battalion", one, 43, 50)
	checkResources(t, "one rifle battalion", one, "steel")

	light := idx.DesignerPayloadFor("GER", "1936.1.1.12", "-",
		"r:0:0:motorized,r:1:0:armored_car,s:0:0:light_tank_recon,s:0:1:engineer")
	equipment, _ := light["equipment"].([]map[string]any)
	wantLabels := []string{"Armored Car", "Infantry Eq.", "Truck", "Support Eq.", "Light Tank"}
	wantAmounts := []float64{60, 110, 35, 30, 24}
	if len(equipment) != len(wantLabels) {
		t.Fatalf("the light template's equipment = %v", equipment)
	}
	for i, item := range equipment {
		if item["short_label"] != wantLabels[i] || item["amount"] != wantAmounts[i] {
			t.Errorf("equipment row %d = %v %v, want %s %v", i, item["short_label"], item["amount"], wantLabels[i], wantAmounts[i])
		}
	}
}

func checkRange(t *testing.T, what string, payload map[string]any, low, high float64) {
	t.Helper()
	cost, _ := payload["production_cost"].(map[string]float64)
	if math.Floor(cost["min"]) != low || math.Floor(cost["max"]) != high {
		t.Errorf("%s costs %v to %v, want %v to %v", what, cost["min"], cost["max"], low, high)
	}
}

func checkResources(t *testing.T, what string, payload map[string]any, want ...string) {
	t.Helper()
	resources, _ := payload["resources"].([]map[string]any)
	got := []string{}
	for _, item := range resources {
		got = append(got, item["resource"].(string))
	}
	if len(got) != len(want) {
		t.Fatalf("%s needs %v, want %v", what, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s needs %v, want %v", what, got, want)
			break
		}
	}
}

func TestDesignerNewTemplateHeaderOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	fresh := idx.DesignerPayloadFor("GER", "1936.1.1.12", "-", "")
	header, _ := fresh["header_stats"].(map[string]float64)
	for key, want := range map[string]float64{"max_organisation": 50.9, "max_strength": 227.2, "speed": 4, "combat_width": 18} {
		if math.Abs(header[key]-want) > 0.05 {
			t.Errorf("New Division header %s = %v, want %v (Infanterie-Division's)", key, header[key], want)
		}
	}
	if one := idx.DesignerPayloadFor("GER", "1936.1.1.12", "-", "r:0:0:infantry"); one["header_stats"] != nil && len(one["header_stats"].(map[string]float64)) > 0 {
		t.Errorf("a template with a battalion keeps the old header: %v", one["header_stats"])
	}

	groups, _ := fresh["names_groups"].([]map[string]any)
	byName := map[string]map[string]any{}
	for _, group := range groups {
		byName[group["name"].(string)] = group
	}
	for _, name := range []string{"Late/Alternate Infantry Divisions", "Light Divisions", "Mountain Divisions", "Cavalry Divisions", "Marine Divisions", "Garrison Divisions", "Paratrooper Divisions"} {
		if byName[name] == nil {
			t.Errorf("the dropdown lacks %q; it has %d groups", name, len(groups))
		}
	}
	light := byName["Light Divisions"]
	if light == nil {
		t.FailNow()
	}
	examples, _ := light["examples"].([]string)
	want := []string{"4. Leichte-Division", "1. Leichte-Division", "2. Leichte-Division", "3. Leichte-Division", "5. Leichte-Division"}
	if len(examples) != len(want) || examples[0] != want[0] || examples[4] != want[4] {
		t.Errorf("Light Divisions examples = %v, want %v", examples, want)
	}
	if light["fallback"] != "123. Leichte-Division" {
		t.Errorf("Light Divisions fallback = %v", light["fallback"])
	}
	mountain, _ := byName["Mountain Divisions"]["examples"].([]string)
	if len(mountain) != 10 || mountain[7] != "188. Gebirgs-Division" || mountain[8] != "1. Ski-Jäger-Division" {
		t.Errorf("Mountain Divisions examples = %v", mountain)
	}

	if byName["Penal Units"] != nil {
		t.Errorf("Penal Units is listed before its focus")
	}
}

func TestDesignerTerrainOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	terrains := func(cells string) []string {
		out := []string{}
		items, _ := idx.DesignerPayloadFor("GER", "1936.1.1.12", "-", cells)["terrain"].([]map[string]any)
		for _, item := range items {
			out = append(out, item["terrain"].(string))
		}
		return out
	}
	for cells, want := range map[string][]string{
		"r:0:0:mountaineers": {"hills", "mountain"},
		"r:0:0:infantry":     {},
		"r:0:0:cavalry":      {"forest", "hills", "mountain", "urban", "jungle", "amphibious"},
	} {
		got := terrains(cells)
		if len(got) != len(want) {
			t.Errorf("%s: terrain %v, want %v", cells, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: terrain %v, want %v", cells, got, want)
				break
			}
		}
	}

	aok := []string{}
	aokItems, _ := idx.DesignerPayloadFor("GER", "1936.1.1.12", "AOK", "")["terrain"].([]map[string]any)
	for _, item := range aokItems {
		aok = append(aok, item["terrain"].(string))
	}
	if want := "forest mountain urban jungle marsh river amphibious"; strings.Join(aok, " ") != want {
		t.Errorf("AOK terrain %v, want %s", aok, want)
	}

	items, _ := idx.DesignerPayloadFor("GER", "1936.1.1.12", "-", "r:0:0:mountaineers,r:0:1:infantry")["terrain"].([]map[string]any)
	if len(items) == 0 || math.Abs(items[0]["attack"].(float64)-0.1) > 1e-6 {
		t.Errorf("mountaineers + infantry hills = %v, want attack 0.1", items)
	}
}

func TestOfficerSpiritSlotsOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	payload, ok := idx.OfficerCorpsPayload("GER", "1936.1.1.12")
	if !ok {
		t.Fatal("no officer corps for GER")
	}
	raw, _ := payload["spirit_slots"].(map[string][]map[string]any)
	slots := map[string][]string{}
	for command, items := range raw {
		for _, item := range items {
			slots[command] = append(slots[command], item["group"].(string))

			if item["affordable"] != false {
				t.Errorf("%s %v affordable on 1 Jan 1936", command, item["group"])
			}
		}
	}
	if raw["army"][0]["cost"] != 20.0 {
		t.Errorf("academy spirit costs %v, want 20", raw["army"][0]["cost"])
	}
	want := map[string][]string{
		"army": {"academy_spirit", "army_spirit", "division_command_spirit"},
		"navy": {"naval_academy_spirit", "navy_spirit", "naval_command_spirit"},
		"air":  {"air_force_spirit", "air_force_command_spirit"},
	}
	for command, groups := range want {
		got := slots[command]
		if len(got) != len(groups) {
			t.Errorf("%s spirits = %v, want %v", command, got, groups)
			continue
		}
		for i := range groups {
			if got[i] != groups[i] {
				t.Errorf("%s spirits = %v, want %v", command, got, groups)
				break
			}
		}
	}
}

func TestArmyHQTemplateOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	templates := idx.ArmyHQTemplatesPayload("GER", "1936.1.1.12")
	if len(templates) != 1 || templates[0]["name"] != "AOK" {
		t.Fatalf("GER Army HQ templates = %v", templates)
	}
	if templates[0]["deploy_days"] != 3 || templates[0]["command_power"] != 9.0 {
		t.Errorf("AOK deploys in %v days for %v command power, want 3 and 9", templates[0]["deploy_days"], templates[0]["command_power"])
	}
	if templates[0]["icon"] != "GFX_div_templ_121_large" {
		t.Errorf("AOK symbol = %v", templates[0]["icon"])
	}
	for _, template := range idx.DivisionTemplatesOf("GER", "1936.1.1.12") {
		if template.IsArmyHQ {
			t.Errorf("the division templates list AOK")
		}
	}
	payload := idx.DesignerPayloadFor("GER", "1936.1.1.12", "AOK", "")
	if payload["is_army_hq"] != true {
		t.Fatalf("AOK does not open as an Army HQ")
	}
	bonuses, _ := payload["commander_bonuses"].([]string)
	want := []string{
		"Enables Ability: §HDefense in Depth§!",
		"Airborne Invasion Preparation Time: §G-5.0%§!",
		"Reconnaissance: §G+10.0%§!",
		"Damage Reduction Against CAS: §G+5.0%§!",
		"Air Superiority Bonus in Combat: §G+5.0%§!",
		"Leader Skill Bonuses: §G+20.0%§!",
		"Terrain traits XP Gain: §G+20.00%§!",
	}
	if len(bonuses) != len(want) {
		t.Fatalf("AOK commander bonuses = %q", bonuses)
	}
	for i := range want {
		if bonuses[i] != want[i] {
			t.Errorf("bonus %d = %q, want %q", i, bonuses[i], want[i])
		}
	}
}

func TestArmyHQPickerOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	payload := idx.DesignerPayloadFor("GER", "1936.1.1.12", "AOK", "")
	units, _ := payload["available"].([]map[string]any)
	byType := map[string]map[string]any{}
	support := []string{}
	for _, unit := range units {
		byType[unit["type"].(string)] = unit
		if unit["group"] == "support" {
			switch unit["type"] {
			case "hq_support_company", "hq_air_liaison", "hq_recon":
			default:
				support = append(support, unit["label"].(string))
			}
		}
	}
	if got := strings.Join(support, " / "); got != "Naval Liaison / Special Operations Coordination / Engineer Staff" {
		t.Errorf("HQ support on offer = %s", got)
	}
	light := byType["hq_light_armor"]
	if light == nil || light["deployment_cost"] != 3.0 {
		t.Errorf("Light Armor HQ Security = %v, want deployment cost 3", light)
	}
	staff := byType["hq_engineer"]
	if staff == nil {
		t.Fatalf("no Engineer Staff among %v", units)
	}
	if staff["deployment_cost"] != 1.0 {
		t.Errorf("Engineer Staff deployment cost = %v", staff["deployment_cost"])
	}
	bonuses, _ := staff["commander_bonuses"].([]string)
	if got := strings.Join(bonuses, " | "); got != "Invasion Defense: §G+5%§! | Fort Damage from Combat: §G+20.0%§!" {
		t.Errorf("Engineer Staff bonuses = %s", got)
	}
	if abilities, _ := staff["abilities"].([]string); len(abilities) != 1 || abilities[0] != "Prepare Fallback Positions" {
		t.Errorf("Engineer Staff abilities = %v", abilities)
	}
	withStaff := idx.DesignerPayloadFor("GER", "1936.1.1.12", "AOK",
		"r:0:0:hq_light_armor,r:0:1:hq_light_armor,s:0:0:hq_support_company,s:0:1:hq_air_liaison,s:0:2:hq_recon,s:0:3:hq_engineer")
	list, _ := withStaff["commander_bonuses"].([]string)
	want := []string{
		"Enables Ability: §HDefense in Depth§!",
		"Enables Ability: §HPrepare Fallback Positions§!",
		"Airborne Invasion Preparation Time: §G-5.0%§!",
		"Invasion Defense: §G+5%§!",
		"Reconnaissance: §G+10.0%§!",
		"Damage Reduction Against CAS: §G+5.0%§!",
		"Air Superiority Bonus in Combat: §G+5.0%§!",
		"Leader Skill Bonuses: §G+20.0%§!",
		"Fort Damage from Combat: §G+20.0%§!",
		"Terrain traits XP Gain: §G+20.00%§!",
	}
	if strings.Join(list, "\n") != strings.Join(want, "\n") {
		t.Errorf("AOK with Engineer Staff:\n%s\nwant\n%s", strings.Join(list, "\n"), strings.Join(want, "\n"))
	}
}

func TestDesignerEquipmentTypesOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	groups, _ := idx.DesignerPayloadFor("GER", "1936.1.1.12", "AOK", "")["equipment_types"].([]map[string]any)
	got := []string{}
	for _, group := range groups {
		line := group["label"].(string) + ":"
		for _, row := range group["types"].([]map[string]any) {
			line += fmt.Sprintf(" %s %d/%d;", row["label"], row["allowed"], row["total"])
		}
		got = append(got, line)
	}
	want := []string{
		"Infantry Eq.: Infantry Equipment I 1/1; Basic Infantry Equipment 1/1;",
		"Light Tank: Improved Light Tank 2/2; Basic Light Tank 1/1; Inter-War Light Tank 1/1;",
		"Support Eq.: Support Equipment 1/1;",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("equipment panel:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestDesignerTankDesignsOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	cut := func(value float64, digits int) float64 {
		scale := math.Pow(10, float64(digits))
		return math.Trunc(value*scale+1e-6) / scale
	}
	support := ",s:0:0:light_tank_recon,s:0:1:engineer"
	cases := []struct {
		name, template, cells string
		want                  map[string]float64
		low, high             float64
	}{
		{"Neubaufahrzeug", "-", "r:0:0:heavy_armor", map[string]float64{
			"speed": 6.2, "soft_attack": 34, "hard_attack": 5, "defense": 2.4, "breakthrough": 35, "armor_value": 60,
			"ap_attack": 10, "reliability": 61.6, "fuel_capacity": 110.4, "fuel_use": 2.30, "hardness": 95}, 480, 848},
		{"Panzer II", "-", "r:0:0:light_armor", map[string]float64{
			"speed": 8.6, "soft_attack": 15, "hard_attack": 5, "defense": 2.3, "breakthrough": 20.8, "armor_value": 26.1,
			"ap_attack": 20, "reliability": 91, "fuel_capacity": 103.2, "fuel_use": 2.15}, 192, 398},
		{"lorried, recon, engineers", "-", "r:0:0:motorized" + support, map[string]float64{
			"speed": 8.6, "soft_attack": 15, "hard_attack": 4.2, "defense": 47.1, "breakthrough": 19.7, "armor_value": 6.2,
			"ap_attack": 13.5, "reliability": 87, "fuel_capacity": 160.8, "fuel_use": 3.35, "hardness": 10}, 331, 421},
		{"with an armoured car", "-", "r:0:0:motorized,r:1:0:armored_car" + support, nil, 511, 601},
		{"with two armoured cars", "-", "r:0:0:motorized,r:1:0:armored_car,r:2:0:armored_car" + support, map[string]float64{
			"soft_attack": 28.2, "hard_attack": 8.6, "defense": 51.1, "breakthrough": 43.7, "armor_value": 6.1,
			"ap_attack": 12.8, "reliability": 84.2, "fuel_capacity": 237.6, "fuel_use": 4.95, "hardness": 46.6}, 691, 781},
		{"AOK", "AOK", "", map[string]float64{
			"speed": 12.6, "soft_attack": 48.6, "hard_attack": 13.1, "defense": 81.6, "breakthrough": 50.9, "armor_value": 16.7,
			"ap_attack": 16.3, "reliability": 87.2, "fuel_capacity": 206.4, "fuel_use": 4.30, "hardness": 80}, 323, 538},
	}
	for _, c := range cases {
		payload := idx.DesignerPayloadFor("GER", "1936.1.1.12", c.template, c.cells)
		stats := payload["stats"].(DivisionStats)
		for stat, want := range c.want {
			value, digits := stats[stat], 1
			switch stat {
			case "reliability", "hardness":
				value *= 100
			case "fuel_use":
				digits = 2
			}
			if got := cut(value, digits); math.Abs(got-want) > 1e-9 {
				t.Errorf("%s %s = %v (%v), want %v", c.name, stat, got, stats[stat], want)
			}
		}
		cost, _ := payload["production_cost"].(map[string]float64)
		if math.Trunc(cost["min"]) != c.low || math.Trunc(cost["max"]) != c.high {
			t.Errorf("%s production cost = %v to %v, want %v to %v", c.name, cost["min"], cost["max"], c.low, c.high)
		}
	}
}

func TestFuelCapacityDefinesFromTheFile(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/defines/00_defines.lua": []byte("NDefines = {\nNMilitary = {\n\tARMY_FUEL_COST_MULT = 0.5,\n" +
			"\tFUEL_CAPACITY_DEFAULT_HOURS = 96,\n},\n}\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if idx.defineNumber("NMilitary.ARMY_FUEL_COST_MULT") != 0.5 || idx.defineNumber("NMilitary.FUEL_CAPACITY_DEFAULT_HOURS") != 96 {
		t.Errorf("fuel defines = %v, %v", idx.defineNumber("NMilitary.ARMY_FUEL_COST_MULT"), idx.defineNumber("NMilitary.FUEL_CAPACITY_DEFAULT_HOURS"))
	}
}

func TestTemplateOOBOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	oob, ok := idx.TemplateOOB("GER", "1936.1.1.12", "Infanterie-Division")
	if !ok || !strings.HasPrefix(oob, "GER_1936") {
		t.Fatalf("Infanterie-Division is in %q (%v)", oob, ok)
	}
	if first, ok := idx.TemplateOOB("GER", "1936.1.1.12", ""); !ok || first != oob {
		t.Errorf("a new template goes into %q, want %q", first, oob)
	}
	if _, ok := idx.TemplateOOB("GER", "1936.1.1.12", "AOK"); ok {
		t.Error("AOK is not in a battle order file")
	}
	source, ok := idx.OOBSource(oob)
	if !ok || !strings.Contains(string(source), `name = "Infanterie-Division"`) || !strings.Contains(string(source), "units") {
		t.Errorf("GER_1936 text = %d bytes (%v)", len(source), ok)
	}
}

func TestDivisionsDeployedCountsOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	templates, _ := idx.DivisionsPayload("GER", "1936.1.1.12")["templates"].([]map[string]any)
	counts := map[string]int{}
	for _, template := range templates {
		counts[template["name"].(string)] = template["deployed"].(int)
	}
	if counts["Infanterie-Division"] != 24 {
		t.Errorf("deployed = %v, want 24 Infanterie-Division", counts)
	}
}

func TestAutonomyLevelsOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	levels := idx.AutonomyLevels()
	names := map[string]string{}
	for _, level := range levels {
		names[level["id"].(string)] = level["label"].(string)
	}
	if len(levels) < 5 || names["autonomy_dominion"] != "Dominion" || names["autonomy_puppet"] == "" {
		t.Errorf("autonomy levels = %v", levels)
	}
}
