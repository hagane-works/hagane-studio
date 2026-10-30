package guirender

import (
	"strings"
	"testing"
)

func hoverIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/ideas/USA.txt": []byte(`ideas = { country = {
	great_depression = {
		modifier = {
			political_power_cost = 1
			consumer_goods_factor = 0.60
			conscription_factor = -0.5
			join_faction_tension = 0.20
			local_resources_factor = -0.2
			stability_factor = 0.05
		}
	}
} }`),
		"common/national_focus/germany.txt": []byte(`focus_tree = { id = german_focus
	focus = { id = GER_tend_the_future cost = 10 x = 1 y = 0 }
	focus = { id = GER_proletarian cost = 10 x = 3 y = 1 }
	focus = {
		id = GER_monarchist_sentiment
		cost = 2
		x = 1 y = 1
		prerequisite = { focus = GER_tend_the_future }
		mutually_exclusive = { focus = GER_proletarian }
		completion_reward = {
			add_popularity = { ideology = neutrality popularity = 0.03 }
			add_political_power = 50
			add_ideas = great_depression
			hidden_effect = { add_stability = 1 }
		}
	}
	focus = {
		id = GER_either
		cost = 5 x = 2 y = 2
		prerequisite = { focus = GER_tend_the_future focus = GER_proletarian }
	}
}`),
		"documentation/modifiers_documentation.md": []byte("## political_power_cost\n\n* Number with 2 decimal places\n\n" +
			"## consumer_goods_factor\n\n* Number with 1 decimal places\n\n## conscription_factor\n\n* Number with 0 decimal places\n\n" +
			"## join_faction_tension\n\n* Number with 1 decimal places\n\n## local_resources_factor\n\n* Number with 2 decimal places\n"),
		"localisation/english/hover_l_english.yml": []byte("\ufeffl_english:\n" +
			" great_depression: \"The Great Depression\"\n" +
			" great_depression_desc: \"Although a worldwide phenomenon, the Great Depression started in the United States.\"\n" +
			" MODIFIER_POLITICAL_POWER_COST: \"Daily Political Power Cost\"\n" +
			" MODIFIER_CONSUMER_GOODS_FACTOR: \"Consumer Goods Factories factor\"\n" +
			" MODIFIER_CONSCRIPTION_FACTOR: \"Recruitable Population\"\n" +
			" MODIFIER_CONSCRIPTION_TOTAL_FACTOR: \"Recruitable Population Factor\"\n" +
			" MODIFIER_JOIN_FACTION_TENSION_LIMIT: \"Join faction tension limit\"\n" +
			" MODIFIER_LOCAL_RESOURCES_FACTOR: \"Resource Gain Efficiency\"\n" +
			" MODIFIER_STABILITY_FACTOR: \"Stability\"\n" +
			" GER_tend_the_future: \"Tend to the Future of Germany\"\n" +
			" GER_proletarian: \"Start The Proletarian Revolution\"\n" +
			" GER_monarchist_sentiment: \"Monarchist Sentiment\"\n" +
			" neutrality_noun: \"Non-Aligned\"\n" +
			" FOCUS_STATUS_TOOLTIP: \"Completion Time: $DAYS|0H$ days\"\n" +
			" FOCUS_REWARD_TOOLTIP: \"§HEffect§!:\"\n" +
			" FOCUS_SINGLE_PREREQ: \"§RRequires $FOCUS$§!\"\n" +
			" NATIONAL_FOCUS_EXCLUSIVE_INFO_DESC: \"Mutually exclusive with\"\n" +
			" POLITICS_ADD_POLITICAL_POWER: \"Political Power: $VAL|=+0$.\"\n" +
			" POLITICS_IDEOLOGY_POPULARITY_CHANGE: \"Change in popularity of $IDEOLOGY$: $VALUE|+2%$\"\n" +
			" EFFECT_ADD_IDEA_NATIONAL_SPIRIT_TOOLTIP_DETAILED: \"Gains National Spirit $NAME|Y$, which grants ($DESC$).\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestIdeaTooltipAsTheGamePrintsIt(t *testing.T) {
	got := hoverIndex(t).IdeaTooltip("great_depression")
	want := strings.Join([]string{
		"§HThe Great Depression§!",
		"",
		"Daily Political Power Cost: §R+1.00§!",
		"Consumer Goods Factories factor: §Y60.0%§!",
		"Recruitable Population Factor: §R-50%§!",
		"Join faction tension limit: §R+20.0%§!",
		"Resource Gain Efficiency: §R-20.00%§!",
		"Stability: §G+5.0%§!",
		"--------------",
		"Although a worldwide phenomenon, the Great Depression started in the United States.",
	}, "\n")
	if got != want {
		t.Errorf("tooltip =\n%s\nwant\n%s", got, want)
	}
}

func TestFocusTooltipAsTheGamePrintsIt(t *testing.T) {
	idx := hoverIndex(t)
	got := idx.FocusTooltip("GER_monarchist_sentiment")
	for _, line := range []string{
		"§HMonarchist Sentiment§!",
		"Completion Time: §H14§! days",
		"§RRequires Tend to the Future of Germany§!",
		"Mutually exclusive with §YStart The Proletarian Revolution§!",
		"§HEffect§!:",
		"Change in popularity of Non-Aligned: §G3.00%§!",
		"Political Power: §G+50§!.",
		"Gains National Spirit §YThe Great Depression§!, which grants (Daily Political Power Cost: §R+1.00§!, ",
	} {
		if !strings.Contains(got, line) {
			t.Errorf("tooltip lacks %q:\n%s", line, got)
		}
	}

	if strings.Contains(got, "Stability") && strings.Contains(got, "Gain Base Stability") {
		t.Errorf("hidden effect printed:\n%s", got)
	}

	either := idx.FocusTooltip("GER_either")
	if !strings.Contains(either, "Requires one of the following:") || !strings.Contains(either, "§R- Tend to the Future of Germany") {
		t.Errorf("either tooltip:\n%s", either)
	}

	payload, ok := idx.FocusTreePayload("german_focus")
	if !ok {
		t.Fatal("no tree")
	}
	for _, focus := range payload["focuses"].([]map[string]any) {
		if focus["id"] == "GER_monarchist_sentiment" && focus["tooltip"] != got {
			t.Errorf("payload tooltip = %v", focus["tooltip"])
		}
	}
}

func TestFormatTooltipValue(t *testing.T) {
	for _, c := range []struct {
		value  any
		format string
		want   string
	}{
		{50.0, "=+0", "§G+50§!"},
		{-0.05, "+=%", "§R-5%§!"},
		{0.03, "+2%", "§G3.00%§!"},
		{14.0, "0H", "§H14§!"},
		{0.1, "-%", "§R10%§!"},
		{"Germany", "Y", "§YGermany§!"},
	} {
		if got := formatTooltipValue(c.value, c.format); got != c.want {
			t.Errorf("formatTooltipValue(%v, %q) = %q, want %q", c.value, c.format, got, c.want)
		}
	}
}

func TestProjectRawTextureResolvesSprite(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"interface/sg119.gfx": []byte(`spriteTypes = { spriteType = { name = "GFX_idea_SG_ancient_march" texturefile = "gfx/interface/sg119/ancient_march_modifier.dds" } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload := idx.spritePayload("GFX_idea_SG_ancient_march"); payload["resolved"] != false {
		t.Fatalf("before: %v", payload)
	}
	idx.SetProjectRawTextures([]ProjectRawTexture{{Path: "gfx/interface/sg119/Ancient_March_Modifier.dds", Width: 64, Height: 64, Version: "d4dd404e7e9b", ObjectKey: "raw/1"}})
	payload := idx.spritePayload("GFX_idea_SG_ancient_march")
	if payload["resolved"] != true || payload["missing_file"] != false || payload["width"] != 64 || payload["version"] != "d4dd404e7e9b" {
		t.Fatalf("after: %v", payload)
	}
	texture, ok := idx.ProjectRawSpriteTexture("GFX_idea_SG_ancient_march")
	if !ok || texture.ObjectKey != "raw/1" {
		t.Fatalf("raw sprite texture = %+v, %v", texture, ok)
	}
}

func TestAdvisorTooltipInThePayload(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/characters/GER.txt": []byte(`characters = {
	GER_hjalmar_schacht = {
		name = GER_hjalmar_schacht
		advisor = { slot = political_advisor idea_token = GER_hjalmar_schacht traits = { captain_of_industry } cost = 150 }
	}
	GER_quiet = {
		name = GER_quiet
		advisor = { slot = political_advisor idea_token = GER_quiet }
	}
}`),
		"common/country_leader/00_traits.txt": []byte(`leader_traits = {
	captain_of_industry = {
		random = no
		sprite = 3
		production_speed_buildings_factor = 0.1
		consumer_goods_factor = -0.05
	}
}`),
		"common/idea_tags/00_idea.txt": []byte(ideaTags),
		"localisation/english/ger_l_english.yml": []byte("\ufeffl_english:\n" +
			" GER_hjalmar_schacht: \"Hjalmar Schacht\"\n GER_quiet: \"Otto Quiet\"\n" +
			" captain_of_industry: \"Captain of Industry\"\n" +
			" MODIFIER_PRODUCTION_SPEED_BUILDINGS_FACTOR: \"Construction Speed\"\n" +
			" MODIFIER_CONSUMER_GOODS_FACTOR: \"Consumer Goods Factories factor\"\n" +
			" CHARACTER_NAME_TOOLTIP: \"$NAME|Y$\"\n political_advisor: \"Political Advisor\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, ok := idx.CharactersPayload("GER")
	if !ok {
		t.Fatal("no GER characters")
	}
	tips := map[string]string{}
	for _, advisor := range payload["advisors"].([]map[string]any) {
		tips[advisor["token"].(string)], _ = advisor["tooltip"].(string)
	}
	want := strings.Join([]string{
		"§YHjalmar Schacht§!",
		"§HCaptain of Industry§!",
		"",
		"Construction Speed: §G+10.0%§!",
		"Consumer Goods Factories factor: §Y-5.0%§!",
		"",
		"Cost (£pol_power ): §R150§!",
	}, "\n")
	if got := tips["GER_hjalmar_schacht"]; got != want {
		t.Errorf("tooltip =\n%s\nwant\n%s", got, want)
	}
	if got := tips["GER_quiet"]; got != "§YOtto Quiet§!" && !strings.HasPrefix(got, "§YOtto Quiet§!\n\nCost (") {
		t.Errorf("an advisor without traits shows their name (and the seat's cost): %q", got)
	}

	for _, category := range payload["categories"].([]map[string]any) {
		for _, slot := range category["slots"].([]map[string]any) {
			if slot["name"] == "political_advisor" && slot["select_text"] != "Select §YPolitical Advisor§!" {
				t.Errorf("political_advisor select_text = %v", slot["select_text"])
			}
		}
	}
}
