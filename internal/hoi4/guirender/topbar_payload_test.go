package guirender

import (
	"encoding/json"
	"strings"
	"testing"

	"haganeworks/internal/reftables"
)

func datedFactionIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/factions/templates/axis.txt": []byte(testFactionTemplates),
		"history/countries/GER - Germany.txt": []byte("capital = 1\n" +
			"create_faction_from_template = faction_template_axis\nadd_to_faction = GER\n"),
		"history/countries/ITA - Italy.txt": []byte("capital = 2\n" +
			"1939.1.1 = { GER = { add_to_faction = ITA } }\n"),
		"history/countries/HUN - Hungary.txt": []byte("capital = 3\n" +
			"IF = { limit = { has_dlc = \"Man the Guns\" } GER = { add_to_faction = PREV } }\n" +
			"ELSE = { GER = { add_to_faction = ROM } }\n"),
		"interface/factions.gfx": []byte(`spriteTypes = { spriteType = { name = "GFX_faction_logo_axis_miniature" texturefile = "gfx/x.dds" } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestFactionMembershipFollowsTheDate(t *testing.T) {
	idx := datedFactionIndex(t)
	if name, _, _ := idx.FactionOf("ITA", "1936.1.1"); name != "" {
		t.Errorf("ITA is in %q on the 1936 start; it joins in its 1939 block", name)
	}
	if name, _, leader := idx.FactionOf("ITA", "1939.8.14"); name != "Axis" || leader != "GER" {
		t.Errorf("ITA 1939 = %q led by %q, want the Axis led by GER", name, leader)
	}

	if name, _, _ := idx.FactionOf("HUN", "1936.1.1"); name != "Axis" {
		t.Errorf("HUN = %q, want the Axis through add_to_faction = PREV", name)
	}
	if name, _, _ := idx.FactionOf("ROM", "1936.1.1"); name != "" {
		t.Errorf("ROM joined through an ELSE that does not run: %q", name)
	}
	if _, found := idx.factionMemberOf("PREV", "1936.1.1"); found {
		t.Errorf("PREV was recorded as a country")
	}
}

func TestTopbarFactionBadge(t *testing.T) {
	idx := datedFactionIndex(t)
	badge, _ := idx.CountryStatsPayload("GER", "1936.1.1")["faction"].(map[string]any)
	if badge == nil || badge["name"] != "Axis" || badge["icon"] != "GFX_faction_logo_axis_miniature" ||
		badge["initiative"] != 0 {
		t.Errorf("GER badge = %v", badge)
	}
	if badge, _ := idx.CountryStatsPayload("ITA", "1936.1.1")["faction"].(map[string]any); badge != nil {
		t.Errorf("ITA 1936 badge = %v, want none", badge)
	}

	plain, err := NewIndex(map[string][]byte{
		"common/factions/templates/axis.txt": []byte(testFactionTemplates),
		"history/countries/GER - Germany.txt": []byte("capital = 1\n" +
			"create_faction_from_template = faction_template_axis\nadd_to_faction = GER\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if badge, _ := plain.CountryStatsPayload("GER", "1936.1.1")["faction"].(map[string]any); badge["icon"] != "" {
		t.Errorf("icon = %v, want empty when no _miniature sprite exists", badge["icon"])
	}

	named, err := NewIndex(map[string][]byte{
		"history/countries/BRN - Brunei.txt": []byte("capital = 1\ncreate_faction = \"Petro-Idol Co-Prosperity Sphere\"\n"),
		"interface/factions.gfx":             []byte(`spriteTypes = { spriteType = { name = "GFX_faction_logo_generic_miniature" texturefile = "gfx/g.dds" } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	badge, _ = named.CountryStatsPayload("BRN", "1936.1.1")["faction"].(map[string]any)
	if badge == nil || badge["name"] != "Petro-Idol Co-Prosperity Sphere" || badge["icon"] != "GFX_faction_logo_generic_miniature" {
		t.Errorf("BRN badge = %v, want the named faction with the generic miniature", badge)
	}
}

const alertLoc = "l_english:\n" +
	" ALERT_RIGHTCLICK:0 \"§GRight-click to dismiss§!\"\n" +
	" alert_no_research_instant:0 \"§YResearch Slot Available§!\"\n" +
	" alert_no_research_delayed:0 \"You have $COUNT|Y$ Research Slot(s) available.\"\n" +
	" alert_decision_new_instant:0 \"§HDecisions Available§!\"\n" +
	" alert_decision_new_delayed:0 \"Next decisions:\\n$LIST$§GRight-click to dismiss§!\"\n" +
	" ALERT_LOSING_TRAINS:0 \"Losing Trains!\"\n"

func TestAlertsPayloadListsEveryEngineAlert(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"localisation/english/alerts_l_english.yml": []byte(alertLoc),
		"common/alerts.txt": []byte("alerts = { alert_lack_of_resources = { category = HIGH }\n" +
			"alert_few_manpower = { category = MEDIUM } }\n"),
		"interface/alerts.gui": []byte(`guiTypes = {
			containerWindowType = { name = "global_alerticon_window" size = { width = 43 height = 38 }
				buttonType = { name = "icon" spriteType = "GFX_mod_alert_icons" } }
			positionType = { name = "alerticon_startposition_extended" position = { x = 718 y = 38 } }
			positionType = { name = "alerticon_offset" position = { x = 48 y = 44 } }
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := idx.AlertsPayload()
	if payload["sprite"] != "GFX_mod_alert_icons" {
		t.Errorf("sprite = %v, want the template's own", payload["sprite"])
	}
	positions, _ := payload["positions"].(map[string]any)
	if start, _ := positions["alerticon_startposition_extended"].(map[string]any); start["x"] != 718 {
		t.Errorf("positions = %v", positions)
	}
	alerts, _ := payload["alerts"].([]map[string]any)
	if len(alerts) != 70 {
		t.Fatalf("%d alerts, want the 70 the strip has frames for", len(alerts))
	}
	frames := map[int]bool{}
	for _, alert := range alerts {
		frames[alert["frame"].(int)] = true
	}
	for frame := 1; frame <= 70; frame++ {
		if !frames[frame] {
			t.Errorf("frame %d is drawn by no alert", frame)
		}
	}

	if alerts[0]["name"] != "alert_no_research" || alerts[0]["frame"] != 1 {
		t.Errorf("first = %v", alerts[0])
	}
	if last := alerts[len(alerts)-1]; last["name"] != "alert_lack_of_resources" || last["category"] != "HIGH" {
		t.Errorf("last = %v, want the file's only HIGH alert", last)
	}
	if alerts[len(alerts)-2]["name"] != "alert_few_manpower" {
		t.Errorf("MEDIUM does not sit before HIGH: %v", alerts[len(alerts)-2])
	}
	byName := map[string]map[string]any{}
	for _, alert := range alerts {
		byName[alert["name"].(string)] = alert
	}

	research := byName["alert_no_research"]
	if research["instant"] != "§YResearch Slot Available§!" ||
		research["delayed"] != "You have $COUNT|Y$ Research Slot(s) available.\n§GRight-click to dismiss§!" {
		t.Errorf("research = %q / %q", research["instant"], research["delayed"])
	}

	if delayed := byName["alert_decision_new"]["delayed"].(string); strings.Count(delayed, "Right-click") != 1 {
		t.Errorf("decision_new delayed = %q", delayed)
	}

	if trains := byName["alert_losing_trains"]; trains["instant"] != "Losing Trains!" || trains["delayed"] != "" {
		t.Errorf("losing trains = %v", trains)
	}
}

func TestAlertCategoriesComeFromAlertsTxt(t *testing.T) {
	base, err := NewIndex(map[string][]byte{
		"common/alerts.txt": []byte("alerts = { alert_lack_of_resources = { category = HIGH } }\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewIndexWithFallback(map[string][]byte{}, base)
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.alertCategory("alert_lack_of_resources"); got != "HIGH" {
		t.Errorf("lack_of_resources = %q", got)
	}
	if got := idx.alertCategory("alert_decision_new"); got != "LOW" {
		t.Errorf("an unlisted alert = %q, want LOW", got)
	}
	none, err := NewIndex(map[string][]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if got := none.alertCategory("alert_lack_of_resources"); got != "LOW" {
		t.Errorf("without alerts.txt = %q, want LOW", got)
	}
}

func TestAlertsWithAnEmptyTable(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"localisation/english/alerts_l_english.yml": []byte(alertLoc),
		"interface/alerts.gfx":                      []byte(`spriteTypes = { spriteType = { name = "GFX_global_alert_icons" texturefile = "gfx/interface/alerts.dds" noOfFrames = 70 } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	frameUsages := func() []map[string]any {
		usage := idx.SpriteUsagePayload([]string{"GFX_global_alert_icons"})["GFX_global_alert_icons"].(map[string]any)
		return usage["frame_usages"].([]map[string]any)
	}
	if len(frameUsages()) != 70 {
		t.Fatalf("%d frame usages with the built-in table", len(frameUsages()))
	}
	defer reftables.Use(map[string][]byte{reftables.AlertsFile: []byte(`{"engine": [], "text_keys": {}}`)})()
	if alerts, _ := idx.AlertsPayload()["alerts"].([]map[string]any); len(alerts) != 0 {
		t.Errorf("%d alerts drawn without the table", len(alerts))
	}
	if instant, delayed := idx.alertTexts("alert_losing_trains"); instant != "" || delayed != "" {
		t.Errorf("losing trains = %q / %q, want its <name>_instant / _delayed (none)", instant, delayed)
	}
	if frames := frameUsages(); len(frames) != 0 {
		t.Errorf("%d frame usages without the table", len(frames))
	}
}

func TestWindowCarriesItsTooltipTexts(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"interface/topbar.gui": []byte(`guiTypes = { containerWindowType = { name = "topbar"
			buttonType = { name = "button_speeddown" spriteType = "GFX_zoom_out" pdx_tooltip = "SPEED_DECREASE" }
			buttonType = { name = "menu_button" spriteType = "GFX_menu" pdx_tooltip = "MAIN_MENU" pdx_tooltip_delayed = "MENU_DELAYED" }
			iconType = { name = "unknown" spriteType = "GFX_x" pdx_tooltip = "NO_SUCH_KEY" }
		} }`),
		"localisation/english/core_l_english.yml": []byte("l_english:\n" +
			" SPEED_DECREASE:0 \"§TDecrease speed§!\"\n" +
			" MAIN_MENU:0 \"Main menu $SHORTCUT$\"\n" +
			" SHORTCUT:0 \"Shortcut: $KEY$\"\n" +
			" MENU_DELAYED:0 \"$FACTION|Y$ stays\"\n" +
			" FACTION:0 \"Faction\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	win, ok := idx.Window("topbar")
	if !ok {
		t.Fatal("no topbar window")
	}
	texts, _ := win["texts"].(map[string]any)
	want := map[string]string{
		"SPEED_DECREASE": "§TDecrease speed§!",

		"MAIN_MENU": "Main menu Shortcut: $KEY$",

		"MENU_DELAYED": "$FACTION|Y$ stays",
	}
	for key, display := range want {
		entry, _ := texts[key].(map[string]any)
		if entry["display"] != display || entry["kind"] != "loc" {
			t.Errorf("%s = %v, want %q", key, entry, display)
		}
	}
	if entry, _ := texts["NO_SUCH_KEY"].(map[string]any); entry["kind"] != "key" {
		t.Errorf("unresolved key = %v", entry)
	}

	node, _ := win["node"].(map[string]any)
	raw, _ := nodeChildren(node)[0]["raw"].(map[string]any)
	if raw["pdx_tooltip"] != "SPEED_DECREASE" {
		t.Errorf("raw = %v", raw)
	}
}

func TestProgressBarSpriteCarriesSizeAndBothTextures(t *testing.T) {
	manifest, err := json.Marshal(VanillaManifest{GameVersion: "t", Textures: map[string][]int{
		"gfx/interface/topbar/fuel_bar.dds":    {45, 24},
		"gfx/interface/topbar/fuel_bar_bg.dds": {45, 8},
	}})
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewVanillaIndex(map[string][]byte{
		VanillaManifestPath: manifest,
		"interface/topbar.gfx": []byte(`spriteTypes = { progressbartype = { name = "GFX_topbar_fuel_bar"
			textureFile1 = "gfx/interface/topbar/fuel_bar.dds" textureFile2 = "gfx/interface/topbar/fuel_bar_bg.dds"
			size = { x = 45 y = 8 } effectFile = "gfx/FX/progressbar.lua" } }`),
	}, "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	payload := idx.spritePayload("GFX_topbar_fuel_bar")
	size, _ := payload["size"].(map[string]any)
	if size["x"] != 45 || size["y"] != 8 {
		t.Errorf("size = %v, want 45x8", payload["size"])
	}
	if payload["width"] != 45 || payload["height"] != 24 {
		t.Errorf("width/height = %v/%v changed; they stay the texture's", payload["width"], payload["height"])
	}
	urls, _ := payload["texture_urls"].([]string)
	if len(urls) != 2 || !strings.HasSuffix(urls[0], "/fuel_bar.png") || !strings.HasSuffix(urls[1], "/fuel_bar_bg.png") {
		t.Errorf("texture_urls = %v", payload["texture_urls"])
	}
}

func TestButtonQuadTextureReplacesSpriteType(t *testing.T) {
	windows, err := ParseGUI(`guiTypes = { containerWindowType = { name = "w"
		buttonType = { name = "threat_value_button" spriteType = "GFX_world_tension_icon_big_strip"
			quadTextureSprite = "GFX_tiled_window_transparent" }
		iconType = { name = "icon" spriteType = "GFX_first" quadTextureSprite = "GFX_second" }
	} }`)
	if err != nil || len(windows) != 1 {
		t.Fatalf("parse: %v", err)
	}
	children := nodeChildren(windows[0])
	if children[0]["sprite"] != "GFX_tiled_window_transparent" || children[0]["sprite_key"] != "quadTextureSprite" {
		t.Errorf("button = %v/%v", children[0]["sprite"], children[0]["sprite_key"])
	}
	if children[1]["sprite"] != "GFX_first" {
		t.Errorf("icon = %v, want its first key", children[1]["sprite"])
	}
}

func TestTakeableDecisionsAreTheOnesTheFilesDecide(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"history/countries/GER - Germany.txt": []byte("capital = 1\n" +
			"set_politics = { ruling_party = fascism }\n" +
			"add_dynamic_modifier = { modifier = GER_mefo_bills_modifier }\n" +
			"1939.1.1 = { add_political_power = 100 complete_national_focus = GER_done set_country_flag = GER_flag }\n"),
		"common/decisions/categories/cats.txt": []byte(`
			open_cat = { }
			debug_cat = { allowed = { is_debug = yes } }
			other_cat = { allowed = { original_tag = FRA } }
		`),
		"common/decisions/ger.txt": []byte(`open_cat = {
			GER_cancel_mefos = { cost = 0 available = { } visible = { has_dynamic_modifier = { modifier = GER_mefo_bills_modifier } } }
			GER_after_focus = { cost = 50 visible = { has_completed_focus = GER_done has_government = fascism } }
			GER_flagged = { visible = { NOT = { has_country_flag = GER_flag } } }
			GER_expensive = { cost = 500 }
			GER_run_time = { available = { threat > 0.25 } }
			GER_scripted = { cost = @UNKNOWN }
			GER_mission = { days_mission_timeout = 30 }
			GER_targeted = { target_trigger = { FROM = { exists = yes } } }
		}
		debug_cat = { DEBUG_button = { } }
		other_cat = { FRA_only = { } }
		`),
		"localisation/english/d_l_english.yml": []byte("l_english:\n GER_cancel_mefos:0 \"Cancel MEFO Bills\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	names := func(date string) []string {
		out := []string{}
		for _, item := range idx.TakeableDecisions("GER", date) {
			out = append(out, item["id"].(string))
		}
		return out
	}

	if got := strings.Join(names("1936.1.1"), ","); got != "GER_cancel_mefos,GER_flagged" {
		t.Errorf("1936 = %s", got)
	}

	if got := strings.Join(names("1939.8.14"), ","); got != "GER_cancel_mefos,GER_after_focus" {
		t.Errorf("1939 = %s", got)
	}
	stats := idx.CountryStatsPayload("GER", "1939.8.14")
	list, _ := stats["decisions_takeable"].([]map[string]any)
	if stats["decisions_takeable_count"] != 2 || list[0]["name"] != "Cancel MEFO Bills" {
		t.Errorf("payload = %v / %v", stats["decisions_takeable_count"], list)
	}
}

func TestDefaultScenarioDate(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/bookmarks/a.txt": []byte(`bookmarks = { bookmark = { name = BLITZ date = 1939.8.14.12 } }`),
		"common/bookmarks/b.txt": []byte(`bookmarks = { bookmark = { name = STORM date = 1936.1.1.12 default = yes } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.DefaultScenarioDate(); got != "1936.1.1.12" {
		t.Errorf("default date = %q", got)
	}
	marks, _ := idx.BookmarksPayload()["bookmarks"].([]map[string]any)
	if len(marks) != 2 || marks[0]["default"] != true || marks[1]["default"] != false {
		t.Errorf("bookmarks = %v", marks)
	}
	empty, _ := NewIndex(map[string][]byte{})
	if got := empty.DefaultScenarioDate(); got != "1936.1.1.12" {
		t.Errorf("no bookmarks = %q", got)
	}
}

func TestTakeableDecisionsHonourEveryLayersGate(t *testing.T) {
	history := []byte("capital = 1\nset_politics = { ruling_party = fascism }\n")
	base, err := NewIndex(map[string][]byte{
		"history/countries/GER - Germany.txt":  history,
		"common/decisions/categories/cats.txt": []byte(`RAJ_cat = { allowed = { original_tag = RAJ } }`),
		"common/decisions/raj.txt":             []byte(`RAJ_cat = { RAJ_effort = { cost = 0 } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	top, err := NewIndexWithFallback(map[string][]byte{
		"common/decisions/categories/cats.txt": []byte(`RAJ_cat = { }`),
		"common/decisions/raj.txt":             []byte(`RAJ_cat = { RAJ_effort = { cost = 0 } }`),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	if got := top.TakeableDecisions("GER", "1936.1.1"); len(got) != 0 {
		t.Errorf("Germany offered %v: the gate below was ignored", got)
	}

	openBase, err := NewIndex(map[string][]byte{
		"history/countries/GER - Germany.txt":  history,
		"common/decisions/categories/cats.txt": []byte(`open_cat = { }`),
		"common/decisions/open.txt":            []byte(`open_cat = { shared_effort = { cost = 0 } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	gatedTop, err := NewIndexWithFallback(map[string][]byte{
		"common/decisions/open.txt": []byte(`open_cat = { shared_effort = { cost = 0 visible = { original_tag = RAJ } } }`),
	}, openBase)
	if err != nil {
		t.Fatal(err)
	}
	if got := openBase.TakeableDecisions("GER", "1936.1.1"); len(got) != 1 {
		t.Fatalf("the open decision is not offered on its own layer: %v", got)
	}
	if got := gatedTop.TakeableDecisions("GER", "1936.1.1"); len(got) != 0 {
		t.Errorf("Germany offered %v past the mod's gate", got)
	}
}
