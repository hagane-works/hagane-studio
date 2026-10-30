package lint

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/hoi4/vanilladata"
)

var panelScriptWrapKeys = map[string]bool{
	"if": true, "else": true, "else_if": true, "limit": true,
	"AND": true, "OR": true, "NOT": true, "and": true, "or": true, "not": true,
	"any": true, "all": true, "all_owned_state": true, "any_owned_state": true,
	"any_country": true, "all_country": true, "any_other_country": true,
	"FROM": true, "ROOT": true, "THIS": true, "PREV": true, "OWNER": true, "CONTROLLER": true,
	"var": true, "value": true, "option": true, "ai_chance": true, "base": true,
	"modifier": true, "factor": true, "add": true, "trigger": true, "immediate": true,
	"after": true, "fire_only_once": true, "is_triggered_only": true, "hidden": true,
	"major": true, "id": true, "title": true, "desc": true, "picture": true,
	"name": true, "scope": true, "tag": true, "original_tag": true, "owner": true,
	"controller": true, "random_list": true, "random_country": true, "random_state": true,
	"global_event_target": true, "event_target": true, "is_border_war": true,
	"show_ideas_tooltip": true, "war_with_on_create": true, "instant": true,
	"popup": true, "reversed": true,
}

func panelKnownSymbol(key string) bool {
	return vanilladata.IsVanillaEffect(key) || vanilladata.IsVanillaTrigger(key) || vanilladata.IsStructuralField(key)
}

type Panel struct {
	CountryTag      string
	Token           string
	NameEn          string
	NameZh          string
	DisplayNameLocs map[string]any
	VisibleTrigger  string
	Controls        []map[string]any
	RawExtras       map[string]any
}

func panelValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func panelMapValue(value any) map[string]any {
	if out, ok := value.(map[string]any); ok && out != nil {
		return out
	}
	return map[string]any{}
}

func panelFirstTruthy(values ...any) string {
	for _, value := range values {
		text := panelValue(value)
		if text != "" {
			return text
		}
	}
	return ""
}

func panelWordless(text any, locs map[string]any) bool {
	if strings.TrimSpace(panelValue(text)) != "" {
		return false
	}
	for _, value := range locs {
		if strings.TrimSpace(panelValue(value)) != "" {
			return false
		}
	}
	return true
}

func panelPyRepr(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "'", "\\'")
	value = strings.ReplaceAll(value, "\r", "\\r")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\t", "\\t")
	return "'" + value + "'"
}

func panelPyList(values []string) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = panelPyRepr(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func panelRef(panel Panel) map[string]any {
	name := strings.TrimSpace(panelFirstTruthy(panel.DisplayNameLocs["en"], panel.NameEn, panel.NameZh, panel.Token))
	var country any
	if tag := strings.TrimSpace(panel.CountryTag); tag != "" {
		country = tag
	}
	return map[string]any{
		"kind": "ui_panel", "entity_id": panel.Token, "name": name, "country_tag": country,
	}
}

func panelFile(value string) *string { return &value }

func BuildPanelContractCore(panels []Panel, snippetNames []string) map[string]any {
	findings := []LintFinding{}
	projectSymbols := map[string]bool{}
	for _, name := range snippetNames {
		projectSymbols[name] = true
	}
	for i := range panels {
		panel := panels[i]
		names := DerivePanelNames(panel.Token, "", false)
		ref := panelRef(panel)

		for j := range panels {
			if i == j || DerivePanelNames(panels[j].Token, "", false)["token"] != names["token"] {
				continue
			}
			findings = append(findings, LintFinding{
				Severity: "error", Code: "panel.token_collision", Title: "Two panels derive the same names",
				Reason: "Panel token " + panelPyRepr(panel.Token) + " and " + panelPyRepr(panels[j].Token) +
					" both normalise to " + panelPyRepr(names["token"]) + " — their .gui/.txt/loc files would overwrite each other on export.",
				Suggestion: "Rename one panel token so the derived names differ.", EntityRef: ref,
				File: panelFile(names["gui_path"]), Detail: map[string]any{"other_token": panels[j].Token, "normalized": names["token"]},
			})
		}

		groups := map[string][]string{}
		order := []string{}
		for _, control := range WalkPanelControls(panel.Controls) {
			raw := panelValue(control["name"])
			slug := DerivePanelNames(panel.Token, raw, true)["control_short"]
			if _, exists := groups[slug]; !exists {
				order = append(order, slug)
			}
			groups[slug] = append(groups[slug], raw)
		}
		for _, slug := range order {
			raws := groups[slug]
			if len(raws) < 2 {
				continue
			}
			findings = append(findings, LintFinding{
				Severity: "error", Code: "panel.control_collision", Title: "Two controls derive the same name",
				Reason: "Controls " + panelPyList(raws) + " all normalise to " + panelPyRepr(slug) +
					" — their .gui nodes, trigger keys and loc keys collide (first-wins; the rest are silently lost). Blank names all fall back to 'control'.",
				Suggestion: "Rename the controls so their derived names differ.", EntityRef: ref,
				File: panelFile(names["gui_path"]), Detail: map[string]any{"control_slug": slug, "controls": raws},
			})
		}

		findings = append(findings, panelUnknownScriptedFindings(panel, projectSymbols, names, ref)...)
		findings = append(findings, panelSpriteEmptyFindings(panel, names, ref)...)
		findings = append(findings, panelLocBlankFindings(panel, ref)...)
	}
	return LintFindingsReport(findings, map[string]any{"scanned_panels": len(panels), "scope": "database_core"})
}

type panelUsage struct {
	name string
	line int
}

func panelUnknownScriptedFindings(panel Panel, projectSymbols map[string]bool, names map[string]string, ref map[string]any) []LintFinding {
	sources := [][2]string{{"visible", panel.VisibleTrigger}}
	for _, control := range WalkPanelControls(panel.Controls) {
		short := panelValue(control["name"])
		cnames := DerivePanelNames(panel.Token, short, true)
		if body := panelValue(control["visible_trigger"]); strings.TrimSpace(body) != "" {
			sources = append(sources, [2]string{"triggers." + cnames["visible_trigger_key"], body})
		}
		if strings.ToLower(strings.TrimSpace(panelValue(control["kind"]))) == "button" {
			if body := panelValue(control["click_effect"]); strings.TrimSpace(body) != "" {
				sources = append(sources, [2]string{"effects." + cnames["click_key"], body})
			}
			if body := panelValue(control["click_enabled_trigger"]); strings.TrimSpace(body) != "" {
				sources = append(sources, [2]string{"triggers." + cnames["click_enabled_key"], body})
			}
		}
	}
	keys := make([]string, 0, len(panel.RawExtras))
	for key := range panel.RawExtras {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if value, ok := panel.RawExtras[key].(string); ok && strings.TrimSpace(value) != "" {
			sources = append(sources, [2]string{"raw_extras." + key, value})
		}
	}

	seen := map[string]bool{}
	out := []LintFinding{}
	for _, source := range sources {
		for _, usage := range panelExtractUsages(source[1]) {
			if projectSymbols[usage.name] || seen[usage.name+"\x00"+source[0]] {
				continue
			}
			seen[usage.name+"\x00"+source[0]] = true
			out = append(out, LintFinding{
				Severity: "warning", Code: "panel.unknown_scripted", Title: "Scripted call with no matching definition",
				Reason: "`" + usage.name + " = yes/no` in " + source[0] + " looks like a scripted effect/trigger call, but " + panelPyRepr(usage.name) +
					" is neither a vanilla symbol nor defined in this project's Scripted Effects & Triggers workbench — in-game it silently evaluates to nothing.",
				Suggestion: "Define " + panelPyRepr(usage.name) + " in the Scripted Effects & Triggers workbench, or replace it with a vanilla trigger/effect.",
				EntityRef:  ref, File: panelFile(names["txt_path"]),
				Detail: map[string]any{"symbol": usage.name, "field": source[0], "line": usage.line},
			})
		}
	}
	return out
}

func panelExtractUsages(text string) []panelUsage {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil
	}
	out := []panelUsage{}
	var walk func([]*paradox.Entry)
	walk = func(entries []*paradox.Entry) {
		for _, entry := range entries {
			if entry == nil {
				continue
			}
			if entry.Key != nil && entry.Block == nil {
				key, scalar := *entry.Key, entry.Scalar.StrValue()
				if (strings.EqualFold(strings.TrimSpace(scalar), "yes") || strings.EqualFold(strings.TrimSpace(scalar), "no")) &&
					!strings.Contains(key, ".") && !panelPythonIsUpper(key) && !panelScriptWrapKeys[key] &&
					!panelKnownSymbol(key) {
					out = append(out, panelUsage{name: key, line: entry.Line})
				}
			}
			if entry.Block != nil {
				walk(entry.Block.Entries)
			}
		}
	}
	walk(ast.Entries)
	return out
}

func panelPythonIsUpper(value string) bool {
	hasUpper := false
	for _, r := range value {
		if unicode.IsLower(r) {
			return false
		}
		if unicode.IsUpper(r) {
			hasUpper = true
		}
	}
	return hasUpper
}

func panelSpriteEmptyFindings(panel Panel, names map[string]string, ref map[string]any) []LintFinding {
	out := []LintFinding{}
	for _, control := range WalkPanelControls(panel.Controls) {
		kind := strings.ToLower(strings.TrimSpace(panelValue(control["kind"])))
		if kind != "icon" && kind != "progressbar" && kind != "button" {
			continue
		}
		short := panelValue(control["name"])
		sprite := strings.TrimSpace(panelValue(control["sprite"]))
		if sprite != "" || kind == "button" {
			continue
		}
		out = append(out, LintFinding{
			Severity: "info", Code: "panel.sprite_empty", Title: "Icon control has no sprite",
			Reason:     "Control " + panelPyRepr(short) + " (" + kind + ") has an empty sprite — it renders as nothing in-game.",
			Suggestion: "Set a GFX_* sprite name on the control.", EntityRef: ref,
			File: panelFile(names["gui_path"]), Detail: map[string]any{"control": short, "kind": kind},
		})
	}
	return out
}

func panelLocBlankFindings(panel Panel, ref map[string]any) []LintFinding {
	keyToControl := map[string]string{}
	entries := [][2]string{}
	for _, control := range WalkPanelControls(panel.Controls) {
		kind := strings.ToLower(strings.TrimSpace(panelValue(control["kind"])))
		if kind != "text" && kind != "button" {
			continue
		}
		short := panelValue(control["name"])
		names := DerivePanelNames(panel.Token, short, true)
		keyToControl[names["text_loc_key"]] = short
		textLocs := panelMapValue(control["text_locs"])

		if kind != "button" || !panelWordless(control["text"], textLocs) {
			entries = append(entries, [2]string{names["text_loc_key"], panelFirstTruthy(textLocs["en"], control["text"])})
		}
		if kind == "button" && panelTooltipPresent(control) {
			keyToControl[names["tooltip_loc_key"]] = short
			tooltipLocs := panelMapValue(control["tooltip_locs"])
			entries = append(entries, [2]string{names["tooltip_loc_key"], panelFirstTruthy(tooltipLocs["en"], control["tooltip"])})
		}
	}
	out := []LintFinding{}
	for _, entry := range entries {
		if strings.TrimSpace(entry[1]) != "" {
			continue
		}
		short := keyToControl[entry[0]]
		out = append(out, LintFinding{
			Severity: "warning", Code: "panel.loc_blank", Title: "Localisation value is blank",
			Reason:     "loc key " + panelPyRepr(entry[0]) + " has an empty EN value — the text control renders blank in-game (looks broken, no error).",
			Suggestion: "Fill the control's text (the weld card offers a one-click placeholder), then save.", EntityRef: ref,
			Detail: map[string]any{"control": short, "loc_key": entry[0]},
		})
	}
	return out
}

func panelTooltipPresent(control map[string]any) bool {
	if strings.TrimSpace(panelValue(control["tooltip"])) != "" {
		return true
	}
	for _, value := range panelMapValue(control["tooltip_locs"]) {
		if strings.TrimSpace(panelValue(value)) != "" {
			return true
		}
	}
	return false
}

const panelFileNS = "zz_uipanel_"

var panelNameRE = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func panelNameSlug(raw, fallback string) string {
	s := strings.Trim(panelNameRE.ReplaceAllString(strings.TrimSpace(raw), "_"), "_")
	if s == "" {
		return fallback
	}
	return s
}

func DerivePanelNames(token, controlName string, withControl bool) map[string]string {
	t := panelNameSlug(token, "panel")
	names := map[string]string{
		"token":            t,
		"scripted_gui_key": t,
		"window_name":      t + "_panel",
		"gui_path":         "interface/" + panelFileNS + t + ".gui",
		"txt_path":         "common/scripted_guis/" + panelFileNS + t + ".txt",
		"loc_basename":     t + "_panel",
		"title_loc_key":    t + "_title",
	}
	if !withControl {
		return names
	}
	c := panelNameSlug(controlName, "control")
	full := t + "_" + c
	names["control_short"] = c
	names["control_name"] = full
	names["visible_trigger_key"] = full + "_visible"
	names["text_loc_key"] = full + "_loc"

	names["click_key"] = full + "_click"
	names["click_enabled_key"] = full + "_click_enabled"
	names["tooltip_loc_key"] = full + "_tt"
	return names
}

func WalkPanelControls(controls []map[string]any) []map[string]any {
	out := []map[string]any{}
	for _, ctrl := range controls {
		if ctrl == nil {
			continue
		}
		out = append(out, ctrl)
		if strings.ToLower(strings.TrimSpace(anyToString(ctrl["kind"]))) == "container" {
			out = append(out, WalkPanelControls(anyToMapSlice(ctrl["children"]))...)
		}
	}
	return out
}
