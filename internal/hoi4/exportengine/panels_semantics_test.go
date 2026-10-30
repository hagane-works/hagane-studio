package exportengine

import (
	"strings"
	"testing"
)

func advancedPanelRow() map[string]any {
	return map[string]any{
		"token":               "TST_meter",
		"context_type":        "decision_category",
		"parent_window_token": "decision_tab",
		"visible_trigger":     "has_dlc = \"Test DLC\"",
		"window_props": map[string]any{
			"x": 12, "y": 34, "width": 500, "height": 160,
			"orientation": "upper_left", "origo": "center", "moveable": true,
			"background_sprite": "",
		},
		"controls": []any{
			map[string]any{
				"kind": "container", "name": "body", "x": 3, "y": 4, "w": 450, "h": 130,
				"raw_extras": map[string]any{
					"AlwaysTransparent": true,
					"margin":            map[string]any{"x": 2, "y": 3},
					"nested_is_rejected": map[string]any{
						"inner": map[string]any{"x": 1},
					},
				},
				"children": []any{
					map[string]any{
						"kind": "text", "name": "caption", "x": 8, "y": 9,
						"w": 250, "h": 28, "font": "hoi_20b", "format": "CENTER",
					},
					map[string]any{
						"kind": "button", "name": "commit", "x": 8, "y": 45,
						"font": "hoi_18b", "tooltip_locs": map[string]any{"zh": "执行"},
						"click_effect": "add_political_power = 5",
						"raw_extras": map[string]any{
							"pdx_tooltip": "WRONG_TOOLTIP",
							"shortcut":    "ENTER",
						},
					},
					map[string]any{
						"kind": "icon", "name": "dynamic_icon", "x": 300, "y": 5,
						"image_expr": "[GetTSTMeterIcon]",
					},
					map[string]any{
						"kind": "progressbar", "name": "meter", "x": 300, "y": 45,
						"frame_var": " global.tst_meter ; } ",
					},
					map[string]any{
						"kind": "icon", "name": "frame_wins", "x": 350, "y": 45,
						"frame_var": "TST_frame", "image_expr": "[IgnoredImage]",
					},
				},
			},
		},
		"raw_extras": map[string]any{
			"visible":          "is_ai = no",
			"triggers":         "TST_external_visible = { always = yes }",
			"effects":          "TST_external_click = { add_stability = 0.01 }",
			"properties":       "TST_external = { frame = TST_external_frame }",
			"ai_test_interval": "24",
			"dirty":            "has_variable = TST_dirty\ncheck_variable = { TST_dirty > 0 }",
		},
	}
}

func TestCompileUIPanelAdvancedGUISemantics(t *testing.T) {
	gui, _, err := CompileUIPanel(advancedPanelRow())
	if err != nil {
		t.Fatalf("CompileUIPanel: %v", err)
	}
	for _, want := range []string{
		"orientation = UPPER_LEFT",
		"origo = CENTER",
		"moveable = yes",
		"containerWindowType = {\n\t\t\tname = \"TST_meter_body\"",
		"format = center",
		"font = \"hoi_18b\"",
		"pdx_tooltip = TST_meter_commit_tt",
		"shortcut = ENTER",
		"alwaystransparent = yes",
		"margin = {\n\t\t\t\tx = 2\n\t\t\t\ty = 3",
	} {
		if !strings.Contains(gui, want) {
			t.Errorf("GUI missing %q:\n%s", want, gui)
		}
	}
	if strings.Contains(gui, "GFX_tiled_window_small") {
		t.Errorf("explicit empty background_sprite must suppress the background:\n%s", gui)
	}
	if strings.Contains(gui, "WRONG_TOOLTIP") {
		t.Errorf("derived tooltip key must override raw_extras value:\n%s", gui)
	}
	if strings.Contains(gui, "nested_is_rejected") {
		t.Errorf("nested non-scalar raw_extras must not be emitted:\n%s", gui)
	}
}

func TestCompileUIPanelAdvancedScriptedGUISemantics(t *testing.T) {
	_, txt, err := CompileUIPanel(advancedPanelRow())
	if err != nil {
		t.Fatalf("CompileUIPanel: %v", err)
	}
	for _, want := range []string{
		"has_dlc = \"Test DLC\"\n\t\t\tis_ai = no",
		"TST_meter_commit_click = {",
		"TST_external_click = { add_stability = 0.01 }",
		"TST_external_visible = { always = yes }",
		"TST_meter_dynamic_icon = {\n\t\t\t\timage = \"[GetTSTMeterIcon]\"",
		"TST_meter_meter = {\n\t\t\t\tframe = global.tst_meter",
		"TST_meter_frame_wins = {\n\t\t\t\tframe = TST_frame",
		"TST_external = { frame = TST_external_frame }",
		"ai_test_interval = 24",
		"dirty = {\n\t\t\thas_variable = TST_dirty",
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("scripted GUI missing %q:\n%s", want, txt)
		}
	}
	for _, block := range []string{"\n\t\tvisible = {", "\n\t\ttriggers = {", "\n\t\teffects = {", "\n\t\tproperties = {"} {
		if got := strings.Count(txt, block); got != 1 {
			t.Errorf("merged %q must be emitted once, got %d:\n%s", strings.TrimSpace(block), got, txt)
		}
	}
	if strings.Contains(txt, "[IgnoredImage]") {
		t.Errorf("frame_var must take precedence over image_expr:\n%s", txt)
	}
}

func TestPanelLocalisationKeepsTooltipKeysSymmetric(t *testing.T) {
	row := advancedPanelRow()
	en := GenerateUIPanelsLocalisation([]map[string]any{row}, "en")
	zh := GenerateUIPanelsLocalisation([]map[string]any{row}, "zh")
	if !strings.Contains(en, ` TST_meter_commit_tt:0 ""`) {
		t.Fatalf("English localisation must retain the tooltip key emitted by the GUI:\n%s", en)
	}
	if !strings.Contains(zh, ` TST_meter_commit_tt:0 "执行"`) {
		t.Fatalf("Chinese localisation must contain the authored tooltip:\n%s", zh)
	}
	if !strings.Contains(en, ` TST_meter_caption_loc:0 ""`) {
		t.Fatalf("a referenced empty text key must still be emitted:\n%s", en)
	}
}

func TestCompileUIPanelContainerBackground(t *testing.T) {
	row := map[string]any{
		"token": "GER_politics_tab_ui", "context_type": "player_context", "parent_window_token": "politics_tab",
		"window_props": map[string]any{"x": 40, "y": 600, "width": 220, "height": 120, "background_sprite": ""},
		"controls": []any{
			map[string]any{"kind": "container", "name": "box", "x": 0, "y": 0, "w": 220, "h": 120, "sprite": "GFX_tiled_window_1b_border",
				"children": []any{map[string]any{"kind": "text", "name": "label", "x": 8, "y": 8}}},
			map[string]any{"kind": "container", "name": "group", "x": 0, "y": 0, "w": 50, "h": 50},
		},
	}
	gui, err := compilePanelGUI(row)
	if err != nil {
		t.Fatal(err)
	}
	box := gui[strings.Index(gui, `name = "GER_politics_tab_ui_box"`):strings.Index(gui, `name = "GER_politics_tab_ui_label"`)]
	if !strings.Contains(box, `quadTextureSprite = "GFX_tiled_window_1b_border"`) || !strings.Contains(box, "background = {") {
		t.Fatalf("the box has no background:\n%s", gui)
	}
	if strings.Count(gui, "background = {") != 1 {
		t.Fatalf("a container without a sprite got a background (or the window did):\n%s", gui)
	}
}

func TestCompileUIPanelButtonSizeAndSwitch(t *testing.T) {
	row := map[string]any{
		"token": "GER_politics_tab_ui", "context_type": "player_context", "parent_window_token": "politics_tab",
		"window_props": map[string]any{"x": 40, "y": 600, "width": 220, "height": 80, "background_sprite": ""},
		"controls": []any{
			map[string]any{"kind": "button", "name": "wide", "x": 0, "y": 0, "w": 220, "h": 30, "sprite": "GFX_tiling_sort_button", "text": "Go"},
			map[string]any{"kind": "button", "name": "switch", "x": 0, "y": 40, "sprite": "GFX_generic_checkbox", "frame_var": "switch_on"},
		},
	}
	gui, err := compilePanelGUI(row)
	if err != nil {
		t.Fatal(err)
	}
	wide := gui[strings.Index(gui, `name = "GER_politics_tab_ui_wide"`):strings.Index(gui, `name = "GER_politics_tab_ui_switch"`)]
	if !strings.Contains(wide, "size = { x = 220 y = 30 }") {
		t.Fatalf("the wide button has no x/y size:\n%s", gui)
	}
	if strings.Contains(wide, "height") {
		t.Fatalf("a button size was written width/height:\n%s", gui)
	}
	if switchPart := gui[strings.Index(gui, `name = "GER_politics_tab_ui_switch"`):]; strings.Contains(switchPart, "size = {") {
		t.Fatalf("a button without w/h got a size:\n%s", gui)
	}
	txt, err := compilePanelTXT(row)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(txt, "GER_politics_tab_ui_switch = {") || !strings.Contains(txt, "frame = switch_on") {
		t.Fatalf("the switch's frame is not bound to its variable:\n%s", txt)
	}
}

func TestCompileUIPanelScrollContainer(t *testing.T) {
	row := map[string]any{
		"token": "GER_politics_tab_ui", "context_type": "player_context", "parent_window_token": "politics_tab",
		"window_props": map[string]any{"x": 40, "y": 600, "width": 220, "height": 120, "background_sprite": ""},
		"controls": []any{
			map[string]any{"kind": "container", "name": "list", "x": 0, "y": 0, "w": 220, "h": 120, "scroll": true,
				"sprite": "GFX_tiled_window_1b_border",
				"children": []any{
					map[string]any{"kind": "text", "name": "row1", "x": 10, "y": 10, "w": 150, "h": 24},
					map[string]any{"kind": "text", "name": "row8", "x": 10, "y": 220, "w": 150, "h": 24},
				}},
			map[string]any{"kind": "container", "name": "plain", "x": 0, "y": 130, "w": 50, "h": 50},
		},
	}
	gui, err := compilePanelGUI(row)
	if err != nil {
		t.Fatal(err)
	}
	list := gui[strings.Index(gui, `name = "GER_politics_tab_ui_list"`):strings.Index(gui, `name = "GER_politics_tab_ui_plain"`)]
	for _, want := range []string{
		`verticalScrollbar = "right_vertical_slider"`, "scroll_wheel_factor = 40", "smooth_scrolling = yes",
		`name = "GER_politics_tab_ui_list_content"`, "size = { width = 160 height = 244 }",
	} {
		if !strings.Contains(list, want) {
			t.Fatalf("scroll container lacks %q:\n%s", want, gui)
		}
	}

	content := list[strings.Index(list, "_list_content"):]
	if !strings.Contains(content, `name = "GER_politics_tab_ui_row8"`) {
		t.Fatalf("the rows are not in the content window:\n%s", gui)
	}
	if strings.Count(gui, "verticalScrollbar") != 1 {
		t.Fatalf("a plain container got a scrollbar:\n%s", gui)
	}
}

func TestCompileUIPanelList(t *testing.T) {
	row := map[string]any{
		"token": "GER_politics_tab_ui", "context_type": "player_context", "parent_window_token": "politics_tab",
		"window_props": map[string]any{"x": 40, "y": 600, "width": 240, "height": 150, "background_sprite": ""},
		"controls": []any{
			map[string]any{"kind": "container", "name": "items", "x": 0, "y": 0, "w": 240, "h": 150, "scroll": true,
				"sprite": "GFX_tiled_window_1b_border", "array": "my_items", "columns": 2, "slot_w": 110, "slot_h": 30, "change_scope": true,
				"children": []any{
					map[string]any{"kind": "text", "name": "label", "x": 5, "y": 5, "w": 80, "h": 24, "text": "Item [?v]"},
					map[string]any{"kind": "button", "name": "pick", "x": 85, "y": 0, "w": 34, "h": 30, "sprite": "GFX_generic_checkbox",
						"click_effect": "add_political_power = 1", "frame_var": "v"},
				}},
		},
	}
	gui, err := compilePanelGUI(row)
	if err != nil {
		t.Fatal(err)
	}
	panelEnd := strings.Index(gui, "\n\t}\n")
	if panelEnd < 0 {
		t.Fatalf("no panel window end:\n%s", gui)
	}
	panel, rest := gui[:panelEnd], gui[panelEnd:]
	for _, want := range []string{
		`name = "GER_politics_tab_ui_items"`, `quadTextureSprite = "GFX_tiled_window_1b_border"`, `verticalScrollbar = "right_vertical_slider"`,
		"gridboxtype = {", `name = "GER_politics_tab_ui_items_grid"`, "size = { width = 220 height = 150 }",
		"slotsize = { width = 110 height = 30 }", "max_slots_horizontal = 2",
	} {
		if !strings.Contains(panel, want) {
			t.Fatalf("the panel window lacks %q:\n%s", want, gui)
		}
	}
	if strings.Contains(panel, "GER_politics_tab_ui_label") || strings.Contains(panel, "_items_content") {
		t.Fatalf("the list's components are in the panel window (or a scroll content window was added):\n%s", gui)
	}
	for _, want := range []string{"\tcontainerWindowType = {", `name = "GER_politics_tab_ui_items_entry"`, "size = { width = 110 height = 30 }",
		`name = "GER_politics_tab_ui_label"`, `name = "GER_politics_tab_ui_pick"`} {
		if !strings.Contains(rest, want) {
			t.Fatalf("the entry window lacks %q:\n%s", want, gui)
		}
	}
	txt, err := compilePanelTXT(row)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"dynamic_lists = {", "GER_politics_tab_ui_items_grid = {", "array = my_items",
		`entry_container = "GER_politics_tab_ui_items_entry"`, "change_scope = yes", "GER_politics_tab_ui_pick_click = {", "frame = v"} {
		if !strings.Contains(txt, want) {
			t.Fatalf("the scripted GUI lacks %q:\n%s", want, txt)
		}
	}
}
