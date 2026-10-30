package guirender

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func industrialOrgIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tag.txt": []byte("capital = 1\n"),
		"common/units/equipment/tanks.txt": []byte(`equipments = {
			heavy_chassis = { is_archetype = yes type = armor interface_category = interface_category_armor }
			heavy_chassis_1 = { archetype = heavy_chassis }
			rifle = { is_archetype = yes type = infantry interface_category = interface_category_land }
		}`),
		"common/equipment_groups/groups.txt": []byte(`
			all_heavy = { equipment_type = { heavy_chassis } }
			all_rifles = { equipment_type = { rifle } }
		`),
		"common/military_industrial_organization/organizations/00_generic.txt": []byte(`
			generic_heavy = {
				icon = GFX_generic_heavy
				allowed = { always = no }
				equipment_type = { all_heavy }
				tree_header_text = { text = header_one x = 1 }
				initial_trait = {
					name = heavy_designer
					equipment_bonus = { armor_value = 0.05 }
				}
				trait = { token = wide_tracks position = { x = 1 y = 0 } }
				trait = { token = cast_turret position = { x = 4 y = 0 } }
			}
			generic_rifles = {
				allowed = { NOT = { original_tag = TAG } }
				equipment_type = { all_rifles }
				initial_trait = { name = rifle_maker production_bonus = { production_cost_factor = -0.05 } }
			}
			closed_example = { include = generic_heavy }
		`),
		"common/military_industrial_organization/organizations/TAG.txt": []byte(`
			TAG_works = {
				include = generic_heavy
				icon = GFX_tag_works
				allowed = { original_tag = TAG }
				initial_trait = { organization_modifier = { military_industrial_organization_research_bonus = 0.05 } }
				override_trait = { token = wide_tracks position = { x = 0 y = 0 } }
				remove_trait = cast_turret
				add_trait = { token = tag_special position = { x = 9 y = 0 } }
			}
			TAG_hidden = {
				include = generic_heavy
				allowed = { original_tag = TAG }
				visible = { FROM = { has_country_flag = never_set } }
			}
			OTHER_works = { include = generic_heavy allowed = { original_tag = OTH } }
		`),
		"localisation/english/mio_l_english.yml": []byte("l_english:\n TAG_works:0 \"$works$\"\n works:0 \"Works\"\n heavy_designer:0 \"Heavy Designer\"\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func orgTokens(payload map[string]any) []string {
	out := []string{}
	for _, item := range payload["items"].([]map[string]any) {
		out = append(out, item["token"].(string))
	}
	return out
}

func TestIndustrialOrgsListWhatTheFirstDayAllows(t *testing.T) {
	payload := industrialOrgIndex(t).IndustrialOrgsPayload("TAG", "1936.1.1.12")
	if got := strings.Join(orgTokens(payload), ","); got != "TAG_works" {
		t.Fatalf("listed %s, want TAG_works", got)
	}
}

func TestIndustrialOrgRowComesFromTheArchetype(t *testing.T) {
	item := industrialOrgIndex(t).IndustrialOrgsPayload("TAG", "1936.1.1.12")["items"].([]map[string]any)[0]
	if item["name"] != "Works" || item["icon"] != "GFX_tag_works" || item["initial_trait"] != "Heavy Designer" {
		t.Errorf("row = %v", item)
	}
	if item["category"] != "tank" || item["size"] != 0 {
		t.Errorf("category %v size %v", item["category"], item["size"])
	}
	equipment := item["equipment"].([]map[string]any)
	if len(equipment) != 1 || equipment[0]["group"] != "all_heavy" ||
		equipment[0]["icon"] != "GFX_military_industrial_organization_all_heavy" ||
		equipment[0]["design"] != 1 || equipment[0]["production"] != 0 || equipment[0]["level"] != 1 {
		t.Errorf("equipment = %v", equipment)
	}
}

func TestIndustrialOrgResolvesTraitEdits(t *testing.T) {
	org := industrialOrgIndex(t).resolveIndustrialOrg("TAG_works")
	tokens := []string{}
	for _, trait := range org.Traits {
		tokens = append(tokens, trait.token())
	}
	if strings.Join(tokens, ",") != "wide_tracks,tag_special" {
		t.Fatalf("traits = %v", tokens)
	}
	if position := ciPoint(org.Traits[0].Block.Entries, "position"); position["x"] != 0 {
		t.Errorf("wide_tracks position = %v, want the override's x = 0", position)
	}
	if ciScalar(org.Initial.Entries, "name") != "heavy_designer" || !hasScriptBlock(org.Initial, "organization_modifier") ||
		!hasScriptBlock(org.Initial, "equipment_bonus") {
		t.Errorf("initial trait lost a part: %v", org.Initial.Entries)
	}
	if len(org.Headers) != 1 || org.Headers[0].Text != "header_one" {
		t.Errorf("headers = %v", org.Headers)
	}
}

func TestRealGameIndustrialOrgsMatchTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	payload := idx.IndustrialOrgsPayload("GER", "1936.1.1.12")
	tanks := []map[string]any{}
	for _, item := range payload["items"].([]map[string]any) {
		if item["category"] == "tank" {
			tanks = append(tanks, item)
		}
	}
	want := []struct {
		name, trait string
		icons       int
	}{
		{"Porsche", "Heavy Tank Designer", 1},
		{"Henschel", "Infantry Tank Designer", 1},
		{"MAN", "Medium Tank Designer", 1},
		{"Alkett", "Assault Guns Manufacturer", 4},
		{"Daimler-Benz", "Standardized Production", 2},
	}
	if len(tanks) != len(want) {
		names := []string{}
		for _, item := range tanks {
			names = append(names, item["name"].(string))
		}
		t.Fatalf("tank companies = %v", names)
	}
	for i, w := range want {
		item := tanks[i]
		equipment := item["equipment"].([]map[string]any)
		if item["name"] != w.name || item["initial_trait"] != w.trait || len(equipment) != w.icons {
			t.Errorf("row %d = %v %v %d icons, want %s %s %d", i, item["name"], item["initial_trait"], len(equipment), w.name, w.trait, w.icons)
			continue
		}
		for _, group := range equipment {
			if group["level"] != 1 {
				t.Errorf("%s %s level = %v, want 1", w.name, group["group"], group["level"])
			}
			if _, declared := idx.spriteRecordFor(group["icon"].(string)); !declared {
				t.Errorf("%s %s icon %v not declared", w.name, group["group"], group["icon"])
			}
		}
	}

	byFilter := map[string][]string{}
	levels := map[string]string{}
	for _, item := range payload["items"].([]map[string]any) {
		name := item["name"].(string)
		byFilter[item["category"].(string)] = append(byFilter[item["category"].(string)], name)
		marks := []string{}
		for _, group := range item["equipment"].([]map[string]any) {
			marks = append(marks, fmt.Sprint(group["level"]))
		}
		levels[name] = strings.Join(marks, " ")
	}
	if got := strings.Join(byFilter["plane"], ","); !strings.HasPrefix(got, "Messerschmitt,Junkers,Focke-Wulf,Heinkel,Dornier,Arado Flugzeugwerke") {
		t.Errorf("plane companies = %s", got)
	}
	if got := strings.Join(byFilter["ship"], ","); got != "Deschimag,Germaniawerft,Blohm & Voss,HDW,Deutsche Werke,Marinewerft Wilhelmshaven" {
		t.Errorf("ship companies = %s", got)
	}
	for name, want := range map[string]string{"Focke-Wulf": "1 1 0", "Heinkel": "1 1 0", "Dornier": "1 0", "Messerschmitt": "1 1 1"} {
		if levels[name] != want {
			t.Errorf("%s counts = %q, want %q", name, levels[name], want)
		}
	}
}

func TestIndustrialOrgTreePlacesTraitsOnTheGrid(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tag.txt": []byte("capital = 1\n"),
		"common/defines/00_defines.lua": []byte("NDefines.NIndustrialOrganisation.DESIGN_TEAM_RESEARCH_BONUS = 0.05\n" +
			"NDefines.NIndustrialOrganisation.ASSIGN_DESIGN_TEAM_PP_COST_PER_DAY = 0.1\n"),
		"common/military_industrial_organization/organizations/TAG.txt": []byte(`
			@far_x = 9
			TAG_works = {
				allowed = { original_tag = TAG }
				tree_header_text = { text = first_column x = 0.5 }
				initial_trait = { name = works_first organization_modifier = { military_industrial_organization_research_bonus = 0.05 } }
				trait = { token = root position = { x = 1 y = 0 } equipment_bonus = { armor_value = 0.05 } }
				trait = { token = child position = { x = 1 y = 2 } relative_position_id = root all_parents = { root } production_bonus = { production_cost_factor = -0.05 } }
				trait = { token = either position = { x = -1 y = 1 } relative_position_id = child any_parent = { child hidden } mutually_exclusive = { hidden } }
				trait = { token = far position = { x = @far_x y = 0 } special_trait_background = yes organization_modifier = { military_industrial_organization_funds_gain = 0.1 } }
				trait = { token = hidden position = { x = 3 y = 3 } visible = { FROM = { has_country_flag = never_set } } }
			}
		`),
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := idx.IndustrialOrgPayload("TAG", "1936.1.1.12", "TAG_works")
	cells := map[string]string{}
	byToken := map[string]map[string]any{}
	for _, trait := range payload["traits"].([]map[string]any) {
		cells[trait["token"].(string)] = fmt.Sprintf("%d,%d", trait["x"], trait["y"])
		byToken[trait["token"].(string)] = trait
	}
	want := map[string]string{"root": "1,0", "child": "2,2", "either": "1,3", "far": "9,0"}
	if fmt.Sprint(cells) != fmt.Sprint(want) {
		t.Errorf("cells = %v, want %v", cells, want)
	}
	if got := byToken["either"]; fmt.Sprint(got["any_parent"], got["exclusive"]) != "[child] []" {
		t.Errorf("either links = %v %v", got["any_parent"], got["exclusive"])
	}
	if fmt.Sprint(byToken["child"]["parents"], byToken["child"]["bonuses"]) != "[root] [production]" {
		t.Errorf("child = %v", byToken["child"])
	}
	if byToken["far"]["special"] != true || fmt.Sprint(byToken["far"]["bonuses"]) != "[organization]" {
		t.Errorf("far = %v", byToken["far"])
	}

	if payload["research_bonus"] != 0.1 || payload["assign_cost"] != 0.1 {
		t.Errorf("attributes = %v %v", payload["research_bonus"], payload["assign_cost"])
	}
}

func TestRealGameIndustrialOrgDetailMatchesTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	porsche := idx.IndustrialOrgPayload("GER", "1936.1.1.12", "GER_porsche_organization")
	rows := map[int][]int{}
	for _, trait := range porsche["traits"].([]map[string]any) {
		rows[trait["y"].(int)] = append(rows[trait["y"].(int)], trait["x"].(int))
	}
	for y := range rows {
		sort.Ints(rows[y])
	}
	if got := fmt.Sprint(rows); got != "map[0:[0 3 5 7 9] 1:[4 9] 2:[2 3 6 8] 3:[1] 4:[0 2 3 4 6 8]]" {
		t.Errorf("Porsche's traits by row = %s", got)
	}
	titles := []string{}
	for _, header := range porsche["headers"].([]map[string]any) {
		titles = append(titles, fmt.Sprintf("%v@%v", header["text"], header["x"]))
	}
	if got := strings.Join(titles, ", "); got != "Engine and Drive System@0.5, Chassis@4, Guns and Turret@7" {
		t.Errorf("headers = %s", got)
	}
	if porsche["research_bonus"] != 0.1 || porsche["assign_cost"] != 0.1 || porsche["background"] != "GFX_MIO_details_background_tank" {
		t.Errorf("Porsche = research %v, cost %v, background %v", porsche["research_bonus"], porsche["assign_cost"], porsche["background"])
	}
	texts := porsche["texts"].(map[string]any)
	if fmt.Sprint(texts["trait_points"], "|", texts["attributes"], "|", texts["research_bonus"], "|", texts["assign_cost"], "|", texts["equipment_bonuses"]) !=
		"Trait points: §Y0§!|Attributes|Research Bonus|Design Team Assign Cost|Equipment Bonuses" {
		t.Errorf("texts = %v", texts)
	}
	if label := porsche["initial"].(map[string]any)["label"]; label != "Heavy Tank Designer" {
		t.Errorf("first trait = %v", label)
	}
	if alkett := idx.IndustrialOrgPayload("GER", "1936.1.1.12", "GER_alkett_organization"); alkett["research_bonus"] != 0.05 {
		t.Errorf("Alkett research bonus = %v, want 0.05", alkett["research_bonus"])
	}
}
