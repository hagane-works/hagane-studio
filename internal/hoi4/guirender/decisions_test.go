package guirender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const decisionCategoryFile = `
political_actions = {
	icon = generic_propaganda
	visible = {

	}
}
resource_prospecting = {
	icon = GFX_decision_category_generic_prospecting
	picture = GFX_decision_cat_picture_generic_prospecting
	priority = {
		base = 8
	}
}
TAG_mefo_bills_category = {
	icon = tag_mefo_bills
	picture = GFX_decision_cat_generic_mefo_bills
	priority = 12
	allowed = {
		original_tag = TAG
	}
	visible = {
		has_completed_focus = TAG_rearmament
	}
}
TAG_empty_category = {
	icon = generic_propaganda
	visible_when_empty = yes
	allowed = {
		original_tag = TAG
	}
}
debug_decisions = {
	icon = generic_research
	visible = {
		is_debug = yes
		is_ai = no
	}
}
`

const decisionsShared = `
political_actions = {
	improved_worker_conditions = {
		icon = generic_industry
		cost = 50
		days_re_enable = 540
		fire_only_once = no
		available = {
			has_stability < 1.0
		}
	}
	targeted_race_for_the_bomb = {
		icon = generic_research
		cost = @BOMB_COST
		target_trigger = {
			FROM = {
				NOT = { is_in_faction_with = ROOT }
			}
		}
	}
	buy_from_the_neighbours = {
		icon = decision_generic_trade
		cost = TAG_trade_deal_cost
		allowed = {
			NOT = { tag = TAG }
		}
	}
}
resource_prospecting = {
	prospect_for_oil = {
		icon = generic_oil
		cost = TAG_prospecting_cost?25
		days_re_enable = 90
		fire_only_once = yes
	}
}
`

const decisionsTag = `
@BOMB_COST = 150

political_actions = {
	TAG_denounce_the_treaty = {
		name = TAG_shared_denouncement
		icon = {
			key = TAG_denounce_armed
			trigger = {
				has_army_size = { size > 50 }
			}
		}
		icon = {
			key = TAG_denounce_weak
		}
		cost = 25
		allowed = {
			original_tag = TAG
		}
	}
}
TAG_mefo_bills_category = {
	TAG_mefo_bills_mission = {
		icon = tag_mefo_bills
		fire_only_once = yes
	}
	DEBUG_TAG_give_mefo_bills = {
		priority = 999
		visible = {
			is_debug = yes
		}
	}
}
debug_decisions = {
	DEBUG_TAG_reset = {
		visible = {
			is_debug = yes
		}
	}
}
`

const decisionsLoc = `l_english:
 political_actions: "Political Actions"
 resource_prospecting: "Resource Prospecting"
 TAG_mefo_bills_category: "MEFO Bills"
 TAG_mefo_bills_category_desc: "Off-the-books [ROOT.GetAdjective] bills pay for the rearmament."
 TAG_ADJ: "Taggish"
 improved_worker_conditions: "Improved Worker Conditions"
 TAG_shared_denouncement: "Denounce the Treaty"
 TAG_mefo_bills_mission: "MEFO Bills Due"
`

func decisionIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/decisions/categories/00_decision_categories.txt": []byte(decisionCategoryFile),
		"common/decisions/_shared_decisions.txt":                 []byte(decisionsShared),
		"common/decisions/TAG.txt":                               []byte(decisionsTag),
		"localisation/english/decisions_l_english.yml":           []byte(decisionsLoc),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func decisionNamed(payload map[string]any, category, name string) map[string]any {
	categories, _ := payload["categories"].([]map[string]any)
	for _, item := range categories {
		if item["name"] != category {
			continue
		}
		rows, _ := item["decisions"].([]map[string]any)
		for _, row := range rows {
			if row["name"] == name {
				return row
			}
		}
	}
	return nil
}

func categoryNamed(payload map[string]any, name string) map[string]any {
	categories, _ := payload["categories"].([]map[string]any)
	for _, item := range categories {
		if item["name"] == name {
			return item
		}
	}
	return nil
}

func TestDecisionCategoriesKeepDeclarationOrder(t *testing.T) {
	payload := decisionIndex(t).DecisionsPayload("TAG")
	categories, _ := payload["categories"].([]map[string]any)
	if len(categories) != 4 {
		t.Fatalf("categories = %d, want the four the file declares", len(categories))
	}
	names := []string{}
	for _, item := range categories {
		names = append(names, item["name"].(string))
	}
	if strings.Join(names, ",") != "political_actions,resource_prospecting,TAG_mefo_bills_category,TAG_empty_category" {
		t.Errorf("category order = %v, want the order the file declares them in", names)
	}
	if categories[0]["label"] != "Political Actions" {
		t.Errorf("label = %v, want the localised category name", categories[0]["label"])
	}

	if categories[2]["icon"] != "GFX_decision_category_tag_mefo_bills" {
		t.Errorf("category icon = %v", categories[2]["icon"])
	}
	if categories[1]["icon"] != "GFX_decision_category_generic_prospecting" {
		t.Errorf("a GFX_-prefixed icon should be left alone, got %v", categories[1]["icon"])
	}
	if categories[1]["picture"] != "GFX_decision_cat_picture_generic_prospecting" {
		t.Errorf("category picture = %v", categories[1]["picture"])
	}
	if _, hasPicture := categories[0]["picture"]; hasPicture {
		t.Errorf("a category with no picture should not carry the key: %v", categories[0])
	}
}

func TestCategoryPriorityAndVisibleBlocks(t *testing.T) {
	payload := decisionIndex(t).DecisionsPayload("TAG")
	if priority := categoryNamed(payload, "TAG_mefo_bills_category")["priority"]; priority != 12 {
		t.Errorf("priority = %v, want the plain number", priority)
	}

	if priority := categoryNamed(payload, "resource_prospecting")["priority"]; priority != 8 {
		t.Errorf("scored priority = %v, want its base", priority)
	}
	if priority := categoryNamed(payload, "political_actions")["priority"]; priority != 0 {
		t.Errorf("a category with no priority = %v", priority)
	}

	if _, gated := categoryNamed(payload, "political_actions")["conditional"]; gated {
		t.Errorf("an empty visible block should not mark the tab conditional")
	}
	if conditional := categoryNamed(payload, "TAG_mefo_bills_category")["conditional"]; conditional != true {
		t.Errorf("a visible block with a trigger should mark the tab conditional, got %v", conditional)
	}
}

func TestDebugOnlyTabsAndRowsStayOut(t *testing.T) {
	payload := decisionIndex(t).DecisionsPayload("TAG")

	if categoryNamed(payload, "debug_decisions") != nil {
		t.Errorf("the debug category reached the screen")
	}
	if decisionNamed(payload, "TAG_mefo_bills_category", "DEBUG_TAG_give_mefo_bills") != nil {
		t.Errorf("a debug decision reached a player's category")
	}
	if decisionNamed(payload, "TAG_mefo_bills_category", "TAG_mefo_bills_mission") == nil {
		t.Errorf("its neighbour in the same category was dropped with it")
	}
}

func TestDecisionsAreFiledUnderTheirOuterBlockKey(t *testing.T) {
	idx := decisionIndex(t)
	names := []string{}
	for _, decision := range idx.DecisionsIn("political_actions") {
		names = append(names, decision.Name)
	}

	if len(names) != 4 {
		t.Fatalf("political_actions = %v, want every file's contribution", names)
	}
	if !contains(names, "improved_worker_conditions") || !contains(names, "TAG_denounce_the_treaty") {
		t.Errorf("political_actions = %v", names)
	}
	for _, decision := range idx.DecisionsIn("TAG_mefo_bills_category") {
		if decision.Category != "TAG_mefo_bills_category" {
			t.Errorf("%s was filed under %s", decision.Name, decision.Category)
		}
	}
	if rows := idx.DecisionsIn("no_such_category"); len(rows) != 0 {
		t.Errorf("an unknown category returned %v", rows)
	}
}

func TestAllowedNamesTheCountryTheDecisionBelongsTo(t *testing.T) {
	idx := decisionIndex(t)
	mine := idx.DecisionsPayload("TAG")
	theirs := idx.DecisionsPayload("SOV")

	if categoryNamed(theirs, "TAG_mefo_bills_category") != nil {
		t.Errorf("SOV was given a category allowed only to TAG")
	}
	if categoryNamed(mine, "TAG_mefo_bills_category") == nil {
		t.Errorf("TAG lost its own category")
	}
	if decisionNamed(mine, "TAG_mefo_bills_category", "TAG_mefo_bills_mission") == nil {
		t.Errorf("a decision inside TAG's own category needs no allowed block of its own")
	}

	if decisionNamed(theirs, "political_actions", "TAG_denounce_the_treaty") != nil {
		t.Errorf("SOV was given a decision allowed only to TAG")
	}
	if decisionNamed(mine, "political_actions", "improved_worker_conditions") == nil ||
		decisionNamed(theirs, "political_actions", "improved_worker_conditions") == nil {
		t.Errorf("a decision with no allowed block belongs to everyone")
	}

	if decisionNamed(mine, "political_actions", "buy_from_the_neighbours") != nil {
		t.Errorf("TAG was given a decision its allowed block excludes")
	}
	if decisionNamed(theirs, "political_actions", "buy_from_the_neighbours") == nil {
		t.Errorf("SOV lost a decision only TAG is excluded from")
	}
}

func TestEmptyTagShowsOnlyWhatNoCountryClaims(t *testing.T) {
	payload := decisionIndex(t).DecisionsPayload("")
	if payload["tag"] != "" {
		t.Errorf("tag = %v", payload["tag"])
	}
	if categoryNamed(payload, "TAG_mefo_bills_category") != nil {
		t.Errorf("a country-specific category survived an empty tag")
	}
	if categoryNamed(payload, "political_actions") == nil {
		t.Errorf("the generic category should survive an empty tag")
	}
	if decisionNamed(payload, "political_actions", "TAG_denounce_the_treaty") != nil {
		t.Errorf("a country-specific decision survived an empty tag")
	}
}

func TestEmptyCategoryStaysOnlyWhenItAsksTo(t *testing.T) {
	idx := decisionIndex(t)
	mine := idx.DecisionsPayload("TAG")

	empty := categoryNamed(mine, "TAG_empty_category")
	if empty == nil {
		t.Fatalf("a category that asks to be visible when empty was dropped")
	}
	if rows, _ := empty["decisions"].([]map[string]any); len(rows) != 0 {
		t.Errorf("empty category = %v", rows)
	}

	if categoryNamed(idx.DecisionsPayload("SOV"), "TAG_empty_category") != nil {
		t.Errorf("visible_when_empty outranked the allowed block")
	}
}

func TestDecisionRowCarriesWhatTheScreenPrints(t *testing.T) {
	payload := decisionIndex(t).DecisionsPayload("TAG")
	worker := decisionNamed(payload, "political_actions", "improved_worker_conditions")
	if worker["label"] != "Improved Worker Conditions" {
		t.Errorf("label = %v", worker["label"])
	}
	if worker["icon"] != "GFX_decision_generic_industry" {
		t.Errorf("icon = %v, want the decision prefix", worker["icon"])
	}
	if worker["cost"] != 50.0 {
		t.Errorf("cost = %v", worker["cost"])
	}
	if worker["days_re_enable"] != 540 {
		t.Errorf("days_re_enable = %v", worker["days_re_enable"])
	}
	if worker["targeted"] != false {
		t.Errorf("targeted = %v, want false for a decision with no target_trigger", worker["targeted"])
	}
	if _, once := worker["fire_only_once"]; once {
		t.Errorf("fire_only_once = no should not carry the key: %v", worker)
	}

	oil := decisionNamed(payload, "resource_prospecting", "prospect_for_oil")
	if oil["fire_only_once"] != true {
		t.Errorf("fire_only_once = %v", oil["fire_only_once"])
	}
	if _, labelled := oil["label"]; labelled {
		t.Errorf("a decision with no localisation should not carry a label: %v", oil)
	}
	bomb := decisionNamed(payload, "political_actions", "targeted_race_for_the_bomb")
	if bomb["targeted"] != true {
		t.Errorf("a decision with a target_trigger = %v", bomb["targeted"])
	}
}

func TestDecisionCostFollowsLiteralsConstantsAndFallbacks(t *testing.T) {
	payload := decisionIndex(t).DecisionsPayload("TAG")

	if cost := decisionNamed(payload, "political_actions", "targeted_race_for_the_bomb")["cost"]; cost != 150.0 {
		t.Errorf("@BOMB_COST = %v, want the constant another file declares", cost)
	}

	if cost := decisionNamed(payload, "resource_prospecting", "prospect_for_oil")["cost"]; cost != 25.0 {
		t.Errorf("fallback cost = %v", cost)
	}

	elsewhere := decisionIndex(t).DecisionsPayload("SOV")
	trade := decisionNamed(elsewhere, "political_actions", "buy_from_the_neighbours")
	if _, priced := trade["cost"]; priced {
		t.Errorf("a script variable was reported as a number: %v", trade)
	}
}

func TestConditionalIconTakesTheFirstCandidate(t *testing.T) {
	payload := decisionIndex(t).DecisionsPayload("TAG")
	denounce := decisionNamed(payload, "political_actions", "TAG_denounce_the_treaty")

	if denounce["icon"] != "GFX_decision_TAG_denounce_armed" {
		t.Errorf("icon = %v, want the first candidate block's key", denounce["icon"])
	}

	if denounce["label"] != "Denounce the Treaty" {
		t.Errorf("label = %v, want the key the name field points at", denounce["label"])
	}
}

func TestDecisionIconTokenPrefixes(t *testing.T) {

	if got := decisionIconSprite("decision_generic_trade"); got != "GFX_decision_generic_trade" {
		t.Errorf("decision_-prefixed token = %v", got)
	}
	if got := decisionIconSprite("GFX_decision_eng_install_government"); got != "GFX_decision_eng_install_government" {
		t.Errorf("GFX_-prefixed token = %v", got)
	}
	if got := decisionIconSprite(""); got != "" {
		t.Errorf("an absent icon = %v, want nothing to draw", got)
	}
	if got := decisionCategorySprite("decision_category_ITA_missiolinis"); got != "GFX_decision_category_ITA_missiolinis" {
		t.Errorf("category token = %v", got)
	}
}

const scopedCategory = `
scoped_actions = {
	icon = generic_propaganda
}
`

func TestScopeReferencesAreNotCountryTags(t *testing.T) {

	idx, err := NewIndex(map[string][]byte{
		"common/decisions/categories/00_decision_categories.txt": []byte(scopedCategory),
		"common/decisions/scoped.txt": []byte(`
scoped_actions = {
	pressure_the_neighbour = {
		cost = 10
		allowed = {
			tag = ROOT
			tag = event_target:WTT_current_china_leader
			tag = var:SWI.SWI_secret_ally
			OR = {
				original_tag = SOV
				original_tag = PRC
			}
		}
	}
}
`),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	if decisionNamed(idx.DecisionsPayload("SOV"), "scoped_actions", "pressure_the_neighbour") == nil {
		t.Errorf("a tag named inside an OR was lost")
	}
	if decisionNamed(idx.DecisionsPayload("PRC"), "scoped_actions", "pressure_the_neighbour") == nil {
		t.Errorf("the second tag of the OR was lost")
	}
	if decisionNamed(idx.DecisionsPayload("GER"), "scoped_actions", "pressure_the_neighbour") != nil {
		t.Errorf("a country the OR does not name was let in")
	}
}

func TestRealGameDecisionsCorpus(t *testing.T) {
	root := strings.TrimSpace(os.Getenv("HOI4_GAME_ROOT"))
	if root == "" {
		t.Skip("HOI4_GAME_ROOT unset; set it to a Hearts of Iron IV install to run the real-corpus gate")
	}
	files := map[string][]byte{}
	for _, sub := range []string{"common/decisions", "common/decisions/categories", "localisation/english"} {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(sub)))
		if err != nil {
			t.Fatalf("read %s: %v", sub, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			switch strings.ToLower(filepath.Ext(entry.Name())) {
			case ".txt", ".yml":
			default:
				continue
			}
			body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sub), entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			files[sub+"/"+entry.Name()] = body
		}
	}
	idx, err := NewIndex(files)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	if categories := idx.DecisionCategories(); len(categories) < 500 {
		t.Fatalf("categories = %d, want the whole of common/decisions/categories", len(categories))
	}
	germany := idx.DecisionsPayload("GER")
	mefo := categoryNamed(germany, "GER_mefo_bills_category")
	if mefo == nil {
		t.Fatalf("Germany lost its MEFO bills category")
	}
	if mefo["label"] != "MEFO Bills" || mefo["icon"] != "GFX_decision_category_ger_mefo_bills" {
		t.Errorf("MEFO category = %v", mefo)
	}
	if decisionNamed(germany, "GER_mefo_bills_category", "GER_mefo_bills_mission") == nil {
		t.Errorf("the MEFO mission is not in its own category")
	}

	if categoryNamed(germany, "debug_decisions") != nil {
		t.Errorf("the debug category is the first tab on Germany's screen")
	}
	if decisionNamed(idx.DecisionsPayload("ITA"), "ITA_balance_of_power_category", "DEBUG_deactivate_BOP") != nil {
		t.Errorf("a debug button inside Italy's balance-of-power tab reached the screen")
	}

	if categoryNamed(idx.DecisionsPayload("SOV"), "GER_mefo_bills_category") != nil {
		t.Errorf("the Soviet Union was given Germany's MEFO bills")
	}

	shared := decisionNamed(germany, "political_actions", "improved_worker_conditions")
	if shared == nil || shared["cost"] != 100.0 {
		t.Errorf("improved_worker_conditions = %v", shared)
	}
	if decisionNamed(idx.DecisionsPayload("SIA"), "political_actions", "improved_worker_conditions") == nil {
		t.Errorf("a decision with no country filter did not reach every country")
	}
}

func TestEditedDecisionKeepsItsPlace(t *testing.T) {
	base, err := NewIndex(map[string][]byte{
		"common/decisions/base.txt": []byte(`tabs = {
			first = { cost = 10 }
			second = { cost = 20 }
			third = { cost = 30 }
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := NewIndexWithFallback(map[string][]byte{
		"common/decisions/project.txt": []byte(`tabs = {
			added = { cost = 5 }
			second = { cost = 99 }
		}`),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	cost := map[string]string{}
	for _, decision := range project.DecisionsIn("tabs") {
		names = append(names, decision.Name)
		cost[decision.Name] = decision.Cost
	}
	if strings.Join(names, ",") != "first,second,third,added" {
		t.Fatalf("order = %v, want the base order with the new decision after it", names)
	}
	if cost["second"] != "99" {
		t.Fatalf("the edited decision is the project's: cost %v", cost["second"])
	}
}

func TestCategoryDescription(t *testing.T) {
	payload := decisionIndex(t).DecisionsPayload("TAG")

	if desc := categoryNamed(payload, "TAG_mefo_bills_category")["desc"]; desc != "Off-the-books Taggish bills pay for the rearmament." {
		t.Errorf("desc = %v, want the category's _desc text read for TAG", desc)
	}
	if desc, has := categoryNamed(payload, "political_actions")["desc"]; has {
		t.Errorf("a category without a _desc key should not carry one, got %v", desc)
	}
}
