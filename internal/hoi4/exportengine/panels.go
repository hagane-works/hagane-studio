package exportengine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	panelVarTokenRE    = regexp.MustCompile(`[^A-Za-z0-9_.@^]+`)
	panelSingleTokenRE = regexp.MustCompile(`^[A-Za-z0-9_.@^\-]+$`)
	panelBareScalarRE  = regexp.MustCompile(`^[A-Za-z0-9_.@^%+\-\[\]|]+$`)
)

func panelSlug(value, fallback string) string {
	var out []rune
	underscore := false
	for _, r := range strings.TrimSpace(value) {

		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' ||
			r >= '0' && r <= '9' || r == '_' {
			out = append(out, r)
			underscore = false
		} else if len(out) > 0 && !underscore {
			out = append(out, '_')
			underscore = true
		}
	}
	result := strings.Trim(string(out), "_")
	if result == "" {
		return fallback
	}
	return result
}

func panelToken(value any) string {
	raw := strings.TrimSpace(textValue(value))
	if raw == "" {
		return ""
	}
	return panelSlug(raw, "panel")
}

func panelInt(value any, fallback int) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		if n, err := strconv.Atoi(v.String()); err == nil {
			return n
		}
	}
	return fallback
}

func panelMap(value any) map[string]any {
	if result, ok := value.(map[string]any); ok {
		return result
	}
	return map[string]any{}
}

func panelTruthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case float64:
		return v != 0
	case float32:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	case json.Number:
		f, err := v.Float64()
		return err != nil || f != 0
	default:
		return true
	}
}

func panelVarToken(value any) string {
	return panelVarTokenRE.ReplaceAllString(strings.TrimSpace(textValue(value)), "")
}

func panelRawScalar(value any) (string, bool) {
	switch v := value.(type) {
	case bool:
		if v {
			return "yes", true
		}
		return "no", true
	case string:
		if panelBareScalarRE.MatchString(v) && v != "" {
			return v, true
		}
		return strconv.Quote(v), true
	case json.Number:
		return v.String(), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	case int:
		return strconv.Itoa(v), true
	case int8:
		return strconv.FormatInt(int64(v), 10), true
	case int16:
		return strconv.FormatInt(int64(v), 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint8:
		return strconv.FormatUint(uint64(v), 10), true
	case uint16:
		return strconv.FormatUint(uint64(v), 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	default:
		return "", false
	}
}

func panelRawMap(value any) map[string]any {
	raw, ok := value.(map[string]any)
	if !ok || raw == nil {
		return nil
	}
	out := make(map[string]any, len(raw))
	for key, value := range raw {
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "" {
			continue
		}
		if _, ok := panelRawScalar(value); ok {
			out[key] = value
			continue
		}
		block, ok := value.(map[string]any)
		if !ok || len(block) == 0 {
			continue
		}
		valid := true
		for _, child := range block {
			if _, ok := panelRawScalar(child); !ok {
				valid = false
				break
			}
		}
		if valid {
			out[key] = block
		}
	}
	return out
}

func panelSortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func panelAppendGUIRaw(lines []string, raw map[string]any, prefix string) []string {
	for _, key := range panelSortedKeys(raw) {
		value := raw[key]
		if block, ok := value.(map[string]any); ok {
			lines = append(lines, prefix+key+" = {")
			for _, childKey := range panelSortedKeys(block) {
				scalar, _ := panelRawScalar(block[childKey])
				lines = append(lines, prefix+"\t"+strings.TrimSpace(childKey)+" = "+scalar)
			}
			lines = append(lines, prefix+"}")
			continue
		}
		scalar, _ := panelRawScalar(value)
		lines = append(lines, prefix+key+" = "+scalar)
	}
	return lines
}

func panelButtonTooltipPresent(control map[string]any) bool {
	if strings.TrimSpace(textValue(control["tooltip"])) != "" {
		return true
	}
	for _, value := range panelMap(control["tooltip_locs"]) {
		if strings.TrimSpace(textValue(value)) != "" {
			return true
		}
	}
	return false
}

func panelList(value any) []any {
	if result, ok := value.([]any); ok {
		return result
	}
	return nil
}

func panelIndent(text string, tabs int) []string {
	prefix := strings.Repeat("\t", tabs)
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result = append(result, "")
		} else {
			result = append(result, prefix+strings.TrimSpace(line))
		}
	}
	return result
}

func panelWalk(controls []any, visit func(map[string]any)) {
	for _, raw := range controls {
		control, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		visit(control)
		if strings.EqualFold(textValue(control["kind"]), "container") {
			panelWalk(panelList(control["children"]), visit)
		}
	}
}

func panelGUIControl(token string, control map[string]any, tabs int, entries *[]string) []string {
	kind := strings.ToLower(strings.TrimSpace(textValue(control["kind"])))
	name := token + "_" + panelSlug(textValue(control["name"]), "control")
	p := strings.Repeat("\t", tabs)
	x, y := panelInt(control["x"], 0), panelInt(control["y"], 0)
	var typeName string
	switch kind {
	case "container":
		typeName = "containerWindowType"
	case "icon", "progressbar":
		typeName = "iconType"
	case "text":
		typeName = "instantTextBoxType"
	case "button":
		typeName = "buttonType"
	default:
		return nil
	}
	lines := []string{p + typeName + " = {", p + "\tname = \"" + name + "\"", fmt.Sprintf("%s\tposition = { x = %d y = %d }", p, x, y)}
	switch kind {
	case "container":
		w, h := panelInt(control["w"], 100), panelInt(control["h"], 100)
		lines = append(lines, fmt.Sprintf("%s\tsize = { width = %d height = %d }", p, w, h))

		if sprite := strings.TrimSpace(textValue(control["sprite"])); sprite != "" {
			lines = append(lines, p+"\tbackground = {", p+"\t\tname = \"Background\"",
				p+"\t\tquadTextureSprite = \""+strings.ReplaceAll(sprite, "\"", "")+"\"", p+"\t}")
		}
		children := panelList(control["children"])
		if array := panelVarToken(control["array"]); array != "" {
			lines = append(lines, panelListGUI(token, name, control, children, w, p, entries)...)
			break
		}
		childTabs := tabs + 1

		if panelTruthy(control["scroll"]) {
			cw, ch := panelContentSize(children)
			lines = append(lines,
				p+"\tverticalScrollbar = \"right_vertical_slider\"",
				p+"\tscroll_wheel_factor = 40",
				p+"\tsmooth_scrolling = yes",
				p+"\tcontainerWindowType = {",
				p+"\t\tname = \""+name+"_content\"",
				p+"\t\tposition = { x = 0 y = 0 }",
				fmt.Sprintf("%s\t\tsize = { width = %d height = %d }", p, max(cw, 1), max(ch, 1)))
			childTabs = tabs + 2
		}
		for _, child := range children {
			if childMap, ok := child.(map[string]any); ok {
				lines = append(lines, panelGUIControl(token, childMap, childTabs, entries)...)
			}
		}
		if panelTruthy(control["scroll"]) {
			lines = append(lines, p+"\t}")
		}
	case "icon", "progressbar":
		if sprite := strings.TrimSpace(textValue(control["sprite"])); sprite != "" {
			lines = append(lines, p+"\tspriteType = \""+strings.ReplaceAll(sprite, "\"", "")+"\"")
		}
		lines = append(lines, fmt.Sprintf("%s\tframe = %d", p, panelInt(control["frame"], 1)))
	case "text":
		lines = append(lines, p+"\ttext = \""+name+"_loc\"", p+"\tfont = \""+panelString(control["font"], "hoi_18mbs")+"\"")
		lines = append(lines, p+"\tformat = "+strings.ToLower(panelString(control["format"], "left")))
		lines = append(lines, fmt.Sprintf("%s\tmaxWidth = %d", p, panelInt(control["w"], 200)), fmt.Sprintf("%s\tmaxHeight = %d", p, panelInt(control["h"], 24)))
	case "button":
		lines = append(lines, p+"\tquadTextureSprite = \""+panelString(control["sprite"], "GFX_button_261x34")+"\"", p+"\tbuttonText = \""+name+"_loc\"", p+"\tfont = \""+panelString(control["font"], "hoi_18mbs")+"\"")

		if w, h := panelInt(control["w"], 0), panelInt(control["h"], 0); w > 0 && h > 0 {
			lines = append(lines, fmt.Sprintf("%s\tsize = { x = %d y = %d }", p, w, h))
		}
	}
	raw := panelRawMap(control["raw_extras"])
	if kind == "button" && panelButtonTooltipPresent(control) {
		if raw == nil {
			raw = map[string]any{}
		}

		raw["pdx_tooltip"] = name + "_tt"
	}
	lines = panelAppendGUIRaw(lines, raw, p+"\t")
	return append(lines, p+"}")
}

func panelContentSize(children []any) (int, int) {
	right, bottom := 0, 0
	for _, raw := range children {
		child, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		var w, h int
		switch strings.ToLower(strings.TrimSpace(textValue(child["kind"]))) {
		case "container":
			w, h = panelInt(child["w"], 100), panelInt(child["h"], 100)
		case "text":
			w, h = panelInt(child["w"], 200), panelInt(child["h"], 24)
		case "button":
			w, h = panelInt(child["w"], 261), panelInt(child["h"], 34)
		default:
			w, h = panelInt(child["w"], 0), panelInt(child["h"], 0)
		}
		right = max(right, panelInt(child["x"], 0)+w)
		bottom = max(bottom, panelInt(child["y"], 0)+h)
	}
	return right, bottom
}

const panelListScrollbarW = 20

func panelListGUI(token, name string, control map[string]any, children []any, w int, p string, entries *[]string) []string {
	gridW := w
	var lines []string
	if panelTruthy(control["scroll"]) {
		gridW = max(1, w-panelListScrollbarW)
		lines = append(lines, p+"\tverticalScrollbar = \"right_vertical_slider\"", p+"\tscroll_wheel_factor = 40", p+"\tsmooth_scrolling = yes")
	}
	slotW, slotH := panelInt(control["slot_w"], gridW), panelInt(control["slot_h"], 30)
	lines = append(lines,
		p+"\tgridboxtype = {",
		p+"\t\tname = \""+name+"_grid\"",
		p+"\t\tposition = { x = 0 y = 0 }",
		fmt.Sprintf("%s\t\tsize = { width = %d height = %d }", p, gridW, panelInt(control["h"], 100)),
		fmt.Sprintf("%s\t\tslotsize = { width = %d height = %d }", p, slotW, slotH),
		fmt.Sprintf("%s\t\tmax_slots_horizontal = %d", p, max(1, panelInt(control["columns"], 1))),
		p+"\t\tformat = \"UPPER_LEFT\"",
		p+"\t}")
	entry := []string{
		"\tcontainerWindowType = {",
		"\t\tname = \"" + name + "_entry\"",
		"\t\tposition = { x = 0 y = 0 }",
		fmt.Sprintf("\t\tsize = { width = %d height = %d }", slotW, slotH),
	}
	for _, child := range children {
		if childMap, ok := child.(map[string]any); ok {
			entry = append(entry, panelGUIControl(token, childMap, 2, entries)...)
		}
	}
	*entries = append(*entries, append(entry, "\t}")...)
	return lines
}

func panelString(value any, fallback string) string {
	result := strings.TrimSpace(textValue(value))
	if result == "" {
		return fallback
	}
	return strings.ReplaceAll(result, "\"", "")
}

func compilePanelGUI(row map[string]any) (string, error) {
	token := panelToken(row["token"])
	if token == "" {
		return "", fmt.Errorf("UI panel token is required")
	}
	props := panelMap(row["window_props"])
	w, h := panelInt(props["width"], 400), panelInt(props["height"], 300)
	lines := []string{"# generated-by: ui-panels builder · token=" + token + " · ONE-WAY", "guiTypes = {", "\tcontainerWindowType = {", "\t\tname = \"" + token + "_panel\"", fmt.Sprintf("\t\tposition = { x = %d y = %d }", panelInt(props["x"], 0), panelInt(props["y"], 0)), fmt.Sprintf("\t\tsize = { width = %d height = %d }", w, h)}
	if orientation := strings.TrimSpace(textValue(props["orientation"])); orientation != "" {
		lines = append(lines, "\t\torientation = "+strings.ToUpper(strings.ReplaceAll(orientation, "\"", "")))
	}
	if origo := strings.TrimSpace(textValue(props["origo"])); origo != "" {
		lines = append(lines, "\t\torigo = "+strings.ToUpper(strings.ReplaceAll(origo, "\"", "")))
	}
	if panelTruthy(props["moveable"]) {
		lines = append(lines, "\t\tmoveable = yes")
	}
	if background, exists := props["background_sprite"]; !exists || textValue(background) != "" {
		sprite := "GFX_tiled_window_small"
		if exists {
			sprite = panelString(background, sprite)
		}
		lines = append(lines, "\t\tbackground = {", "\t\t\tname = \"Background\"", "\t\t\tquadTextureSprite = \""+sprite+"\"", "\t\t}")
	}
	var entries []string
	for _, control := range panelList(row["controls"]) {
		if controlMap, ok := control.(map[string]any); ok {
			lines = append(lines, panelGUIControl(token, controlMap, 2, &entries)...)
		}
	}
	lines = append(append(append(lines, "\t}"), entries...), "}")
	return strings.Join(lines, "\n") + "\n", nil
}

func compilePanelTXT(row map[string]any) (string, error) {
	token := panelToken(row["token"])
	if token == "" {
		return "", fmt.Errorf("UI panel token is required")
	}
	contextType := panelSlug(textValue(row["context_type"]), "decision_category")
	lines := []string{"# generated-by: ui-panels builder · token=" + token + " · ONE-WAY", "scripted_gui = {", "\t" + token + " = {", "\t\tcontext_type = " + contextType}
	if parent := panelSlug(textValue(row["parent_window_token"]), ""); parent != "" {
		lines = append(lines, "\t\tparent_window_token = "+parent)
	}
	rawExtras := panelMap(row["raw_extras"])
	mergedRaw := map[string]string{}
	passthroughRaw := map[string]string{}
	for key, value := range rawExtras {
		key = strings.TrimSpace(key)
		body := strings.TrimSpace(textValue(value))
		if flag, ok := value.(bool); ok {
			body, _ = panelRawScalar(flag)
		}
		if key == "" || body == "" {
			continue
		}
		switch strings.ToLower(key) {
		case "visible", "triggers", "effects", "properties":
			mergedRaw[strings.ToLower(key)] = body
		default:
			passthroughRaw[key] = body
		}
	}
	lines = append(lines, "\t\twindow_name = \""+token+"_panel\"", "", "\t\tvisible = {")
	if visible := strings.TrimSpace(textValue(row["visible_trigger"])); visible != "" {
		lines = append(lines, panelIndent(visible, 3)...)
	}
	if visible := mergedRaw["visible"]; visible != "" {
		lines = append(lines, panelIndent(visible, 3)...)
	}
	if strings.TrimSpace(textValue(row["visible_trigger"])) == "" && mergedRaw["visible"] == "" {
		lines = append(lines, "\t\t\talways = yes")
	}
	lines = append(lines, "\t\t}")
	var triggers, effects, properties, lists []string
	panelWalk(panelList(row["controls"]), func(control map[string]any) {
		name := token + "_" + panelSlug(textValue(control["name"]), "control")

		if array := panelVarToken(control["array"]); array != "" && strings.EqualFold(textValue(control["kind"]), "container") {
			lists = append(lists, "\t\t\t"+name+"_grid = {", "\t\t\t\tarray = "+array, "\t\t\t\tentry_container = \""+name+"_entry\"")
			if panelTruthy(control["change_scope"]) {
				lists = append(lists, "\t\t\t\tchange_scope = yes")
			}
			lists = append(lists, "\t\t\t}")
		}
		if visible := strings.TrimSpace(textValue(control["visible_trigger"])); visible != "" {
			triggers = append(triggers, "\t\t\t"+name+"_visible = {", strings.Join(panelIndent(visible, 4), "\n"), "\t\t\t}")
		}
		if strings.EqualFold(textValue(control["kind"]), "button") {
			effect := strings.TrimSpace(textValue(control["click_effect"]))
			effects = append(effects, "\t\t\t"+name+"_click = {", strings.Join(panelIndent(effect, 4), "\n"), "\t\t\t}")
			if enabled := strings.TrimSpace(textValue(control["click_enabled_trigger"])); enabled != "" {
				triggers = append(triggers, "\t\t\t"+name+"_click_enabled = {", strings.Join(panelIndent(enabled, 4), "\n"), "\t\t\t}")
			}
		}
		if frame := panelVarToken(control["frame_var"]); frame != "" {
			properties = append(properties, "\t\t\t"+name+" = {", "\t\t\t\tframe = "+frame, "\t\t\t}")
		} else if image := strings.TrimSpace(textValue(control["image_expr"])); image != "" {
			properties = append(properties, "\t\t\t"+name+" = {", "\t\t\t\timage = \""+strings.ReplaceAll(image, "\"", "")+"\"", "\t\t\t}")
		}
	})
	for _, block := range []struct {
		name  string
		lines []string
	}{{"triggers", triggers}, {"effects", effects}, {"properties", properties}, {"dynamic_lists", lists}} {
		rawBody := mergedRaw[block.name]
		if len(block.lines) > 0 || rawBody != "" {
			lines = append(lines, "", "\t\t"+block.name+" = {")
			lines = append(lines, block.lines...)
			if rawBody != "" {
				lines = append(lines, panelIndent(rawBody, 3)...)
			}
			lines = append(lines, "\t\t}")
		}
	}
	passthroughKeys := make([]string, 0, len(passthroughRaw))
	for key := range passthroughRaw {
		passthroughKeys = append(passthroughKeys, key)
	}
	sort.Strings(passthroughKeys)
	for _, key := range passthroughKeys {
		body := passthroughRaw[key]
		lines = append(lines, "")
		if panelSingleTokenRE.MatchString(body) {
			lines = append(lines, "\t\t"+key+" = "+body)
		} else {
			lines = append(lines, "\t\t"+key+" = {")
			lines = append(lines, panelIndent(body, 3)...)
			lines = append(lines, "\t\t}")
		}
	}
	return strings.Join(append(lines, "\t}", "}"), "\n") + "\n", nil
}

func CompileUIPanel(row map[string]any) (guiText, txtText string, err error) {
	guiText, err = compilePanelGUI(row)
	if err != nil {
		return "", "", err
	}
	txtText, err = compilePanelTXT(row)
	if err != nil {
		return "", "", err
	}
	return guiText, txtText, nil
}

func panelLocalized(row map[string]any, field, legacy, lang string) string {
	values := panelMap(row[field])
	if textValue(values[lang]) != "" {
		return textValue(values[lang])
	}
	if lang != "en" && textValue(values["en"]) != "" {
		return textValue(values["en"])
	}
	return textValue(row[legacy])
}

func panelLocalisation(rows []map[string]any, lang string) string {
	header := map[string]string{"en": "l_english", "zh": "l_simp_chinese", "ja": "l_japanese", "ko": "l_korean", "ru": "l_russian", "de": "l_german", "fr": "l_french", "es": "l_spanish", "pt": "l_braz_por", "pl": "l_polish"}[lang]
	if header == "" {
		header = "l_english"
	}
	lines := []string{header + ":"}
	for _, row := range rows {
		token := panelSlug(textValue(row["token"]), "panel")
		if title := panelLocalized(row, "display_name_locs", "name_en", lang); title != "" {
			lines = append(lines, " "+token+"_title:0 "+panelLocQuote(title))
		}
		panelWalk(panelList(row["controls"]), func(control map[string]any) {
			kind := strings.ToLower(textValue(control["kind"]))
			if kind != "text" && kind != "button" {
				return
			}
			name := token + "_" + panelSlug(textValue(control["name"]), "control")

			lines = append(lines, " "+name+"_loc:0 "+panelLocQuote(panelLocalized(control, "text_locs", "text", lang)))
			if kind == "button" && panelButtonTooltipPresent(control) {

				lines = append(lines, " "+name+"_tt:0 "+panelLocQuote(panelLocalized(control, "tooltip_locs", "tooltip", lang)))
			}
		})
	}
	return strings.Join(lines, "\n")
}

func GenerateUIPanelsLocalisation(rows []map[string]any, lang string) string {
	return panelLocalisation(rows, lang)
}

func panelLocQuote(value string) string {
	text := strings.ReplaceAll(value, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.ReplaceAll(quoteScript(text), "\n", `\n`)
}
