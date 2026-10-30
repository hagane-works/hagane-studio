package guirender

import (
	"reflect"
	"testing"
)

const testDLCHistory = `
capital = 8
set_politics = { ruling_party = democratic }
IF = {
	limit = { has_dlc = "Arms Against Tyranny" }
	add_ideas = qa_with_aat
}
ELSE = {
	add_ideas = qa_without_aat
}
IF = {
	limit = { NOT = { has_dlc = "By Blood Alone" } }
	add_ideas = qa_without_bba
}
IF = {
	limit = { has_dlc = "La Resistance" }
	create_intelligence_agency = yes
}
`

func dlcIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"history/countries/LUX - Luxemburg.txt": []byte(testDLCHistory),
		"history/states/1032 - Yanan.txt":       []byte(testYananFile),
	})
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestPreviewFollowsTheProjectsDLCs(t *testing.T) {
	for _, c := range []struct {
		name       string
		all        bool
		owned      []string
		ideas      []string
		yanan      string
		agencyMade bool
	}{
		{"every DLC (not chosen)", true, nil, []string{"qa_with_aat"}, "XIC", true},
		{"no DLC", false, nil, []string{"qa_without_aat", "qa_without_bba"}, "PRC", false},
		{"AAT and NCNS only", false, []string{"Arms Against Tyranny", "No Compromise, No Surrender"},
			[]string{"qa_with_aat", "qa_without_bba"}, "XIC", false},
		{"La Resistance and BBA", false, []string{"La Resistance", "By Blood Alone"},
			[]string{"qa_without_aat"}, "PRC", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			idx := dlcIndex(t)
			idx.SetOwnedDLCs(c.owned, c.all)
			if got := idx.StartingIdeas("LUX", "1936.1.1"); !reflect.DeepEqual(got, c.ideas) {
				t.Errorf("starting ideas %v, want %v", got, c.ideas)
			}
			for _, state := range idx.effectiveStates() {
				if state.ID == 1032 {
					if got := idx.ownerOfState(state, "1936.1.1"); got != c.yanan {
						t.Errorf("Yan'an starts as %q's, want %q's", got, c.yanan)
					}

					if got := idx.ownerOfState(state, "1938.10.25"); got != "PRC" {
						t.Errorf("Yan'an in 1938 is %q's, want PRC's", got)
					}
				}
			}
			if got := idx.IntelAgencyPayload("LUX", "1936.1.1")["created"]; got != c.agencyMade {
				t.Errorf("agency created at the start = %v, want %v", got, c.agencyMade)
			}
		})
	}
}

func TestLimitGuardReadsOnlyDLCConditions(t *testing.T) {
	for _, c := range []struct {
		limit string
		ok    bool
		want  dlcGuard
	}{
		{`has_dlc = "La Resistance"`, true, dlcGuard{Need: []string{"La Resistance"}}},
		{`NOT = { has_dlc = "By Blood Alone" }`, true, dlcGuard{Without: []string{"By Blood Alone"}}},
		{`OR = { has_dlc = "A" has_dlc = "B" }`, true, dlcGuard{AnyOf: [][]string{{"A", "B"}}}},
		{`has_dlc = "A" NOT = { has_dlc = "B" }`, true, dlcGuard{Need: []string{"A"}, Without: []string{"B"}}},
		{`has_dlc = "A" tag = GER`, false, dlcGuard{}},
		{`NOT = { has_dlc = "A" tag = GER }`, false, dlcGuard{}},
	} {
		root, ok := parseScript([]byte("IF = { limit = { " + c.limit + " } }"))
		if !ok || len(root.Entries) != 1 {
			t.Fatalf("parse %q", c.limit)
		}
		got, decidable := limitGuard(root.Entries[0].Block.Entries)
		if decidable != c.ok || (c.ok && !reflect.DeepEqual(got, c.want)) {
			t.Errorf("%s → %+v %v, want %+v %v", c.limit, got, decidable, c.want, c.ok)
		}
	}
	owned := map[string]bool{"a": true}
	if !(dlcGuard{AnyOf: [][]string{{"A", "B"}}}).holds(owned) || (dlcGuard{Need: []string{"B"}}).holds(owned) {
		t.Error("guard holds() disagrees with the owned set")
	}
	if !(dlcGuard{Need: []string{"Anything"}}).holds(nil) || (dlcGuard{Without: []string{"Anything"}}).holds(nil) {
		t.Error("an unset DLC choice must mean every DLC")
	}
}

func TestIntegratedDLCsAreAlwaysOwned(t *testing.T) {
	idx := dlcIndex(t)
	idx.SetOwnedDLCs(nil, false)
	for _, name := range IntegratedDLCs {
		if !idx.OwnsDLC(name) {
			t.Errorf("%s ships with the game but counts as not owned", name)
		}
	}
	if idx.OwnsDLC("La Resistance") {
		t.Error("La Resistance owned with nothing chosen")
	}
	if got := sortedDLCs(idx.ownedDLC); len(got) != len(IntegratedDLCs) {
		t.Errorf("owned set %v", got)
	}
}

const testDLCTrees = `
focus_tree = {
	id = qa_afghan_focus
	country = { factor = 0 modifier = { add = 10 tag = AFG has_dlc = "Graveyard of Empires" } }
	shortcut = { name = qa_go_new target = qa_afg_a trigger = { has_dlc = "Gotterdammerung" } }
	shortcut = { name = qa_go_old target = qa_afg_a trigger = { NOT = { has_dlc = "Gotterdammerung" } } }
	focus = { id = qa_afg_a x = 0 y = 0 cost = 10 }
}
focus_tree = {
	id = qa_generic_focus
	country = { factor = 1 }
	default = yes
	focus = { id = qa_gen_a x = 0 y = 0 cost = 10 }
}
focus_tree = {
	id = qa_prc_old
	country = { factor = 0 modifier = { add = 10 AND = { tag = PRC has_dlc = "Waking the Tiger" NOT = { has_dlc = "No Compromise, No Surrender" } } } }
	focus = { id = qa_prc_old_a x = 0 y = 0 cost = 10 }
}
focus_tree = {
	id = qa_prc_sea
	country = { factor = 0 modifier = { add = 1000 tag = PRC has_dlc = "No Compromise, No Surrender" } }
	focus = { id = qa_prc_sea_a x = 0 y = 0 cost = 10 }
}
`

const testDLCAfghanHistory = `
capital = 267
IF = {
	limit = { has_dlc = "Graveyard of Empires" }
	set_technology = { qa_goe_tech = 1 }
}
ELSE = {
	set_technology = { qa_base_tech = 1 }
}
IF = {
	limit = { has_dlc = "Trial of Allegiance" }
	create_faction = qa_pact
}
`

func TestFocusTreesTechAndFactionsFollowTheProjectsDLCs(t *testing.T) {
	for _, c := range []struct {
		name      string
		all       bool
		owned     []string
		afgTree   string
		prcTree   string
		shortcut  string
		tech      string
		inFaction bool
	}{
		{"every DLC (not chosen)", true, nil, "qa_afghan_focus", "qa_prc_sea", "qa_go_new", "qa_goe_tech", true},
		{"no DLC", false, nil, "qa_generic_focus", "qa_prc_old", "qa_go_old", "qa_base_tech", false},
		{"Graveyard of Empires only", false, []string{"Graveyard of Empires"},
			"qa_afghan_focus", "qa_prc_old", "qa_go_old", "qa_goe_tech", false},
		{"NCNS and Trial of Allegiance", false, []string{"No Compromise, No Surrender", "Trial of Allegiance"},
			"qa_generic_focus", "qa_prc_sea", "qa_go_old", "qa_base_tech", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			idx, err := NewIndex(map[string][]byte{
				"common/national_focus/qa_trees.txt":      []byte(testDLCTrees),
				"history/countries/AFG - Afghanistan.txt": []byte(testDLCAfghanHistory),
			})
			if err != nil {
				t.Fatal(err)
			}
			idx.SetOwnedDLCs(c.owned, c.all)
			if got := idx.FocusTreeForTag("AFG"); got != c.afgTree {
				t.Errorf("Afghanistan plays %q, want %q", got, c.afgTree)
			}
			if got := idx.FocusTreeForTag("PRC"); got != c.prcTree {
				t.Errorf("the communists play %q, want %q", got, c.prcTree)
			}
			{
				tree, ok := idx.FocusTreeByID("qa_afghan_focus")
				if !ok {
					t.Fatal("qa_afghan_focus not found")
				}
				var names []string
				for _, shortcut := range tree.Shortcuts {
					names = append(names, shortcut.Name)
				}
				if !reflect.DeepEqual(names, []string{c.shortcut}) {
					t.Errorf("shortcuts %v, want [%s]", names, c.shortcut)
				}
			}
			if got := idx.ResearchedBy("AFG", "1936.1.1"); !reflect.DeepEqual(got, []string{c.tech}) {
				t.Errorf("researched %v, want [%s]", got, c.tech)
			}
			if _, leads := idx.factionTemplateOf("AFG", "1936.1.1"); leads != c.inFaction {
				t.Errorf("Afghanistan leads a faction = %v, want %v", leads, c.inFaction)
			}
		})
	}
}
