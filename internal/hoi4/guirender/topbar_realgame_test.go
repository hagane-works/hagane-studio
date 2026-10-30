package guirender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func realTopbarIndex(t *testing.T, wrap bool) *Index {
	t.Helper()
	root := strings.TrimSpace(os.Getenv("HOI4_GAME_ROOT"))
	if root == "" {
		t.Skip("HOI4_GAME_ROOT unset; set it to a Hearts of Iron IV install to run the real-corpus gate")
	}
	files := map[string][]byte{}
	for _, sub := range []string{
		"history/countries", "history/states", "common/ideas", "common/idea_tags",
		"common/national_focus", "common/factions", "common/buildings", "common/decisions",
		"common/bookmarks", "common/defines", "common/country_tags", "common/ideologies",
		"common/scripted_localisation", "common/characters",
		"common/units", "history/units", "history/general", "common/equipment_groups", "common/military_industrial_organization",
		"common/unit_leader", "common/country_leader", "common/doctrines",
		"common/technologies", "common/dynamic_modifiers", "common/modifiers", "common/autonomous_states", "common/resources", "common/wargoals", "common/intelligence_agencies", "common/intelligence_agency_upgrades", "common/special_projects", "common/scientist_traits", "map/strategicregions",
		"localisation/english", "interface",
	} {
		base := filepath.Join(root, filepath.FromSlash(sub))
		err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".txt", ".lua", ".yml", ".gui", ".gfx":
			default:
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			files[filepath.ToSlash(rel)] = body
			return nil
		})
		if err != nil {
			t.Fatalf("read %s: %v", sub, err)
		}
	}

	if body, err := os.ReadFile(filepath.Join(root, "documentation", "modifiers_documentation.md")); err == nil {
		files["documentation/modifiers_documentation.md"] = body
	}
	if body, err := os.ReadFile(filepath.Join(root, "common", "alerts.txt")); err == nil {
		files["common/alerts.txt"] = body
	}
	idx, err := NewIndex(files)
	if err != nil {
		t.Fatal(err)
	}
	if !wrap {
		return idx
	}
	top, err := NewIndexWithFallback(files, idx)
	if err != nil {
		t.Fatal(err)
	}
	return top
}

func TestRealGameTopbarNumbers(t *testing.T) {
	idx := realTopbarIndex(t, true)

	at1936 := idx.CountryStatsPayload("GER", "1936.1.1.12")
	if at1936["civilian_factories"] != 35 || at1936["military_factories"] != 28 || at1936["dockyards"] != 10 {
		t.Errorf("GER 1936 factories = %v/%v/%v, want 35/28/10",
			at1936["civilian_factories"], at1936["military_factories"], at1936["dockyards"])
	}
	if at1936["factories_total"] != 73 || at1936["population"] != 65513907 {
		t.Errorf("GER 1936 total %v population %v, want 73 and 65,513,907", at1936["factories_total"], at1936["population"])
	}
	if resources, _ := at1936["resources"].(map[string]int); resources["steel"] != 160 {
		t.Errorf("GER 1936 steel = %v, want 160", resources["steel"])
	}
	states, _ := idx.StatesOfPayload("GER", "1936.1.1.12", 0)["states"].([]map[string]any)
	if len(states) != 24 {
		t.Errorf("GER 1936 lists %d states, want 24 (one row per state id)", len(states))
	}
	if fuel, _ := at1936["fuel"].(map[string]any); fuel["ratio"] != 0.8 {
		t.Errorf("GER fuel ratio = %v, want the history's 0.8", fuel["ratio"])
	}
	if at1936["world_tension"] != 0.0 {
		t.Errorf("1936 world tension = %v", at1936["world_tension"])
	}
	if faction, _ := at1936["faction"].(map[string]any); faction == nil ||
		faction["icon"] != "GFX_faction_logo_axis_miniature" || faction["initiative"] != 0 {
		t.Errorf("GER 1936 faction = %v, want the Axis miniature at initiative 0", at1936["faction"])
	}

	at1939 := idx.CountryStatsPayload("GER", "1939.8.14.12")
	if at1939["civilian_factories"] != 75 || at1939["military_factories"] != 67 || at1939["dockyards"] != 13 {
		t.Errorf("GER 1939 factories = %v/%v/%v, want 75/67/13",
			at1939["civilian_factories"], at1939["military_factories"], at1939["dockyards"])
	}

	if at1939["command_power"] != 130.0 || at1939["air_experience"] != 25 || at1939["navy_experience"] != 0 {
		t.Errorf("GER 1939 cp/air/navy = %v/%v/%v, want 130/25/0",
			at1939["command_power"], at1939["air_experience"], at1939["navy_experience"])
	}

	base1939, _ := idx.countryStartAt("GER", "1939.8.14.12")
	if at1939["research_slots"] != 5 || at1939["convoys"] != 200 ||
		!near(base1939.WarSupport, 0.75) || !near(base1939.Stability, 0.65) {
		t.Errorf("GER 1939 slots/convoys/war support/stability = %v/%v/%v/%v, want 5/200/0.75/0.65",
			at1939["research_slots"], at1939["convoys"], base1939.WarSupport, base1939.Stability)
	}

	if !near(at1936["stability"], 0.81) || !near(at1936["war_support"], 0.35) {
		t.Errorf("GER 1936 stability/war support = %v/%v, want 0.81/0.35", at1936["stability"], at1936["war_support"])
	}
	if decisions, _ := at1939["decisions_takeable"].([]map[string]any); len(decisions) == 0 ||
		decisions[0]["name"] != "Cancel MEFO Bills" {
		t.Errorf("GER 1939 takeable decisions = %v, want at least Cancel MEFO Bills", decisions)
	}
	if at1939["world_tension"] != 42.0 {
		t.Errorf("1939 world tension = %v, want 40 (GER) + 2 (ITA)", at1939["world_tension"])
	}
	t.Logf("GER 1936: pp=%v cp=%v stab=%v ws=%v fuel=%v decisions=%v",
		at1936["political_power"], at1936["command_power"], at1936["stability"], at1936["war_support"],
		at1936["fuel"], at1936["decisions_takeable_count"])
	t.Logf("GER 1939: pp=%v army=%v fuel=%v decisions=%v %v",
		at1939["political_power"], at1939["army_experience"], at1939["fuel"],
		at1939["decisions_takeable_count"], at1939["decisions_takeable"])

	for _, tc := range []struct{ tag, date, want, leader string }{
		{"ITA", "1936.1.1.12", "", ""}, {"FRA", "1936.1.1.12", "", ""}, {"CHI", "1936.1.1.12", "", ""},
		{"ITA", "1939.8.14.12", "Axis", "GER"}, {"FRA", "1939.8.14.12", "Allies", "ENG"},
	} {
		name, _, leader := idx.FactionOf(tc.tag, tc.date)
		if name != tc.want || (tc.want != "" && leader != tc.leader) {
			t.Errorf("%s %s faction = %q led by %q, want %q led by %q", tc.tag, tc.date, name, leader, tc.want, tc.leader)
		}
	}
	if jap, _ := idx.CountryStatsPayload("JAP", "1936.1.1.12")["fuel"].(map[string]any); jap["ratio"] != 1.0 || jap["days"] != nil {
		t.Errorf("JAP fuel = %v, want a full stock", jap)
	}
	eng := idx.CountryStatsPayload("ENG", "1936.1.1.12")
	t.Logf("ENG 1936: slots=%v stab=%v ws=%v cp=%v", eng["research_slots"], eng["stability"], eng["war_support"], eng["command_power"])
}

func TestRealGameAlerts(t *testing.T) {
	idx := realTopbarIndex(t, false)
	payload := idx.AlertsPayload()
	alerts, _ := payload["alerts"].([]map[string]any)
	if payload["sprite"] != "GFX_global_alert_icons" || len(alerts) != 70 {
		t.Fatalf("sprite %v, %d alerts", payload["sprite"], len(alerts))
	}
	missing := []string{}
	for _, alert := range alerts {
		if alert["instant"] == "" {
			missing = append(missing, alert["name"].(string))
		}
	}
	if len(missing) > 0 {
		t.Errorf("alerts with no instant text: %v", missing)
	}
	if last := alerts[len(alerts)-1]; last["category"] != "HIGH" {
		t.Errorf("the row does not end with its HIGH alerts: %v", last)
	}
	t.Logf("positions: %v", payload["positions"])
	t.Logf("first: %v", alerts[0])
}

func near(value any, want float64) bool {
	got, ok := value.(float64)
	return ok && got > want-1e-9 && got < want+1e-9
}

func TestRealGameAlertCategoriesAreAlertsTxt(t *testing.T) {
	idx := realTopbarIndex(t, false)
	root := strings.TrimSpace(os.Getenv("HOI4_GAME_ROOT"))
	body, err := os.ReadFile(filepath.Join(root, "common", "alerts.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var uncommented strings.Builder
	for _, line := range strings.Split(string(body), "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		uncommented.WriteString(line + "\n")
	}
	text := uncommented.String()
	seen := 0
	for _, chunk := range strings.Split(text, "alert_")[1:] {
		name := "alert_" + strings.TrimSpace(chunk[:strings.Index(chunk, "=")])
		at := strings.Index(chunk, "category")
		if at < 0 {
			continue
		}
		rest := strings.TrimLeft(chunk[at+len("category"):], " \t=")
		category := strings.Fields(rest)[0]
		seen++
		if got := idx.alertCategory(name); got != category {
			t.Errorf("%s: index %q, alerts.txt %q", name, got, category)
		}
	}
	if seen == 0 {
		t.Error("alerts.txt lists no alerts")
	}
}

func TestRealGameStartingLawsReplaceEachOther(t *testing.T) {
	idx := realTopbarIndex(t, false)
	laws := map[string][]string{}
	for _, token := range idx.StartingIdeas("GER", "1939.8.14.12") {
		if group := idx.ideaCategoryOf(token); idx.isSlotCategory(group) {
			laws[group] = append(laws[group], token)
		}
	}
	for group, want := range map[string]string{
		"mobilization_laws": "extensive_conscription", "economy": "war_economy", "trade_laws": "autarkic_economy",
	} {
		if len(laws[group]) != 1 || laws[group][0] != want {
			t.Errorf("GER 1939 %s = %v, want only %s", group, laws[group], want)
		}
	}
}
