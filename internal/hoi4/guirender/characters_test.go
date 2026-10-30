package guirender

import "testing"

const chiCharacters = `
characters = {
	CHI_chiang_kai_shek = {
		name = CHI_chiang_kai_shek
		portraits = {
			civilian = { large = "GFX_portrait_CHI_chiang_kai_shek_base" }
		}
		country_leader = { ideology = despotism }
		instance = {
			allowed = { has_dlc = "No Compromise, No Surrender" }
			portraits = { civilian = { large = "GFX_portrait_CHI_chiang_kai_shek" } }
		}
	}
	CHI_dai_li = {
		name = CHI_dai_li
		portraits = { civilian = { small = "GFX_portrait_CHI_dai_li_small" } }
		advisor = { slot = political_advisor idea_token = CHI_dai_li }
	}
	CHI_xue_yue = {
		name = "Hsueh Yueh"
		portraits = { army = { small = "GFX_idea_CHI_xue_yue" } }
		corps_commander = { skill = 4 }
	}
}
`

const ideaTags = `
idea_categories = {
	hidden_ideas = { hidden = yes }
	country = { type = national_spirit ledger = hidden }
	government = {
		slot = mobilization_laws
		character_slot = political_advisor
		ledger = civilian
	}
	research_production = {
		slot = industrial_concern
		character_slot = theorist
		ledger = civilian
	}
}
`

const chiLoc = "l_english:\n CHI_chiang_kai_shek: \"Chiang Kai-Shek\"\n CHI_dai_li: \"Dai Li\"\n government: \"Laws & Government\"\n political_advisor: \"Political Advisor\"\n"

func characterIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/characters/CHI.txt":    []byte(chiCharacters),
		"common/idea_tags/00_idea.txt": []byte(ideaTags),
		"localisation/english/chi.yml": []byte(chiLoc),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestCharactersPayloadSortsPeopleByRole(t *testing.T) {
	idx := characterIndex(t)
	payload, ok := idx.CharactersPayload("chi")
	if !ok {
		t.Fatalf("no characters for CHI; tags=%v", idx.ListCharacterTags())
	}
	leader, _ := payload["leader"].(map[string]any)
	if leader == nil {
		t.Fatal("no country leader")
	}
	if got := leader["name"]; got != "Chiang Kai-Shek" {
		t.Errorf("leader name = %v, want the localised one", got)
	}

	if got := leader["portrait"]; got != "GFX_portrait_CHI_chiang_kai_shek_base" {
		t.Errorf("leader portrait = %v", got)
	}
	advisors, _ := payload["advisors"].([]map[string]any)
	if len(advisors) != 1 || advisors[0]["slot"] != "political_advisor" {
		t.Fatalf("advisors = %v", advisors)
	}
	if got := advisors[0]["name"]; got != "Dai Li" {
		t.Errorf("advisor name = %v", got)
	}
	commanders, _ := payload["commanders"].([]map[string]any)
	if len(commanders) != 1 || commanders[0]["name"] != "Hsueh Yueh" {
		t.Fatalf("commanders = %v", commanders)
	}
}

func TestCharacterPortraitResolvesThroughTheSpriteIndex(t *testing.T) {
	idx := characterIndex(t)
	payload, _ := idx.CharactersPayload("CHI")
	advisors, _ := payload["advisors"].([]map[string]any)
	sprite, _ := advisors[0]["portrait_sprite"].(map[string]any)
	if sprite == nil {
		t.Fatal("advisor portrait was not run through the sprite index")
	}

	if sprite["resolved"] != false {
		t.Errorf("unexpected resolution: %v", sprite)
	}
}

func TestIdeaCategoriesKeepDeclarationIndexAndDropHiddenRows(t *testing.T) {
	idx := characterIndex(t)
	categories := idx.IdeaCategories()
	if len(categories) != 2 {
		t.Fatalf("categories = %v, want the two civilian rows", categories)
	}

	if categories[0].Index != 3 || categories[1].Index != 4 {
		t.Errorf("indexes = %d,%d", categories[0].Index, categories[1].Index)
	}
	if categories[0].Name != "government" {
		t.Errorf("first row = %q", categories[0].Name)
	}
	slots := categories[0].Slots
	if len(slots) != 2 || slots[0].Character || !slots[1].Character {
		t.Fatalf("slots = %+v", slots)
	}
	payload, _ := idx.CharactersPayload("CHI")
	rows, _ := payload["categories"].([]map[string]any)
	if len(rows) != 2 || rows[0]["label"] != "Laws & Government" {
		t.Errorf("payload rows = %v", rows)
	}
}

func TestCharacterTraitsFollowTheRoleNotTheBlockOrder(t *testing.T) {
	for _, order := range []string{
		`advisor = { slot = theorist traits = { blitzkrieg_theorist } } corps_commander = { traits = { panzer_leader } skill = 4 }`,
		`corps_commander = { traits = { panzer_leader } skill = 4 } advisor = { slot = theorist traits = { blitzkrieg_theorist } }`,
	} {
		idx, err := NewIndex(map[string][]byte{
			"common/characters/GER.txt":    []byte("characters = { GER_heinz_guderian = { name = GER_heinz_guderian " + order + " } }"),
			"common/idea_tags/00_idea.txt": []byte(ideaTags),
		})
		if err != nil {
			t.Fatal(err)
		}
		payload, ok := idx.CharactersPayload("GER")
		if !ok {
			t.Fatal("no payload")
		}
		advisors := payload["advisors"].([]map[string]any)
		commanders := payload["commanders"].([]map[string]any)
		if len(advisors) != 1 || len(commanders) != 1 {
			t.Fatalf("advisors %v commanders %v", advisors, commanders)
		}
		advisor := advisors[0]["traits"].([]map[string]any)[0]["token"]
		commander := commanders[0]["traits"].([]map[string]any)[0]["token"]
		if advisor != "blitzkrieg_theorist" || commander != "panzer_leader" {
			t.Errorf("%s: advisor shows %v, commander %v", order[:20], advisor, commander)
		}
	}
}

func TestCharactersPayloadMissesUnknownTag(t *testing.T) {
	idx := characterIndex(t)
	if _, ok := idx.CharactersPayload("FRA"); ok {
		t.Error("FRA should have no characters in this fixture")
	}
}
