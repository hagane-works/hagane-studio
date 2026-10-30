package guirender

import "testing"

const testFactionTemplates = `
faction_template_axis = {
	name = Axis
	icon = GFX_faction_logo_axis
}
faction_template_allies = {
	name = Allies
	icon = GFX_faction_logo_allies
}
`

const testFactionLeader = `
capital = 1
create_faction_from_template = faction_template_axis
add_to_faction = LED
`

const testFactionAlly = `
capital = 2
LED = { add_to_faction = ALY }
`

const testFactionNeutral = `
capital = 3
`

func factionIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/factions/templates/axis.txt":  []byte(testFactionTemplates),
		"history/countries/LED - Leader.txt":  []byte(testFactionLeader),
		"history/countries/ALY - Ally.txt":    []byte(testFactionAlly),
		"history/countries/NEU - Neutral.txt": []byte(testFactionNeutral),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestFactionFollowsTheScopeNotTheFileName(t *testing.T) {
	idx := factionIndex(t)
	name, icon, leader := idx.FactionOf("LED", "1936.1.1")
	if name != "Axis" || icon != "GFX_faction_logo_axis" || leader != "LED" {
		t.Errorf("leader = %v/%v/%v", name, icon, leader)
	}

	name, _, leader = idx.FactionOf("ALY", "1936.1.1")
	if name != "Axis" || leader != "LED" {
		t.Errorf("ally = %v led by %v, want the Axis led by LED", name, leader)
	}
	if name, _, _ := idx.FactionOf("NEU", "1936.1.1"); name != "" {
		t.Errorf("a country in no faction reports %q", name)
	}
}

func TestFactionScreenCarriesTheNameKey(t *testing.T) {
	out := factionIndex(t).FactionPayload("ALY", "1936.1.1")
	if out["name"] != "Axis" || out["name_key"] != "Axis" {
		t.Errorf("faction screen = %v", out)
	}
}

func TestFactionNameKey(t *testing.T) {
	idx := factionIndex(t)
	if got := idx.FactionNameKey("ALY", "1936.1.1"); got != "Axis" {
		t.Errorf("template faction key = %q, want Axis", got)
	}
	if got := idx.FactionNameKey("NEU", "1936.1.1"); got != "" {
		t.Errorf("no faction, no key: %q", got)
	}
	named, err := NewIndex(map[string][]byte{
		"history/countries/NEW - New.txt":      []byte("capital = 1\ncreate_faction = NEW_pact\nadd_to_faction = NEW\n"),
		"localisation/english/f_l_english.yml": []byte("l_english:\n NEW_pact:0 \"The Pact\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	name, _, _ := named.FactionOf("NEW", "1936.1.1")
	if key := named.FactionNameKey("NEW", "1936.1.1"); name != "The Pact" || key != "NEW_pact" {
		t.Errorf("create_faction by name = (%q, %q), want (The Pact, NEW_pact)", name, key)
	}
}

func TestFactionTemplateNamesAndIcons(t *testing.T) {
	idx := factionIndex(t)

	if _, ok := idx.factionTemplates["faction_template_allies"]; !ok {
		t.Errorf("templates = %v, want both declared ones", idx.factionTemplates)
	}
}
