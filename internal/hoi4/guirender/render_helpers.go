package guirender

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	gridNameRE  = regexp.MustCompile(`(?i)(_grid|_gridbox)$`)
	entryNameRE = regexp.MustCompile(`(?i)(_entry|_item|_row|_slot|_element)(_\d+)?$|entry$`)
	stateNameRE = regexp.MustCompile(`(?i)(loading|popup|tooltip|modal|confirm)`)
	fontSizeRE  = regexp.MustCompile(`\d{2}`)
	colorRE     = regexp.MustCompile(`§.`)
	scriptedRE  = regexp.MustCompile(`\[([^\]]+)\]`)
	substRE     = regexp.MustCompile(`\$([^$]+)\$`)
	literalNum  = regexp.MustCompile(`^[\d\s:./%+\-?]+$`)
	literalX    = regexp.MustCompile(`^[Xx]{2,}$`)
	locKeyRE    = regexp.MustCompile(`^(?:[A-Za-z][A-Za-z0-9]*_[A-Za-z0-9_]*|[A-Z][A-Z0-9]{1,})$`)
)

func classifyWindow(name string, node map[string]any) string {
	if gridNameRE.MatchString(name) {
		return "grid"
	}
	if entryNameRE.MatchString(name) {
		return "entry"
	}
	w, h, percent := dimensions(node)
	background, _ := node["background"].(map[string]any)
	_, hasBackground := background["sprite"].(string)
	if (hasBackground && w >= 400 && h >= 300) || (hasBackground && percent) || w >= 900 || h >= 700 {
		return "screen"
	}
	return "fragment"
}

func dimensions(node map[string]any) (int, int, bool) {
	size, _ := node["size"].(map[string]any)
	if size == nil {
		return 0, 0, false
	}
	w, wok := size["width"]
	if !wok {
		w = size["x"]
	}
	h, hok := size["height"]
	if !hok {
		h = size["y"]
	}
	wi, _ := w.(int)
	hi, _ := h.(int)
	if wi <= 0 {
		wi = 0
	}
	if hi <= 0 {
		hi = 0
	}
	_, wPercent := w.(string)
	_, hPercent := h.(string)
	return wi, hi, wPercent || hPercent
}

func nodeChildren(node map[string]any) []map[string]any {
	children, _ := node["children"].([]map[string]any)
	return children
}

func resolvePercent(value any, base int) int {
	if text, ok := value.(string); ok {
		trimmed := strings.TrimSpace(text)
		percent := strings.HasSuffix(trimmed, "%")
		number := strings.TrimRight(trimmed, "%")
		if number == "" {
			return 0
		}
		f, err := strconv.ParseFloat(number, 64)
		if err != nil {
			if percent {
				return base
			}
			return 0
		}
		if percent {
			return int(float64(base) * f / 100)
		}
		return int(f)
	}
	return toInt(value, 0)
}

func resolveSize(value any, base int) int {
	n := resolvePercent(value, base)
	if n < 0 {
		n = base + n
	}
	if n < 0 {
		return 0
	}
	return n
}

func anchor(orientation string, px, py, pw, ph int) (int, int) {
	o := strings.ToUpper(strings.TrimSpace(orientation))
	if o == "" {
		o = "UPPER_LEFT"
	}
	ax, ay := px, py
	if strings.Contains(o, "RIGHT") {
		ax = px + pw
	} else if strings.Contains(o, "CENTER") {
		ax = px + pw/2
	}
	if strings.Contains(o, "LOWER") || strings.Contains(o, "DOWN") || strings.Contains(o, "BOTTOM") {
		ay = py + ph
	} else if strings.Contains(o, "CENTER") && !strings.Contains(o, "UP") && !strings.Contains(o, "DOWN") {
		ay = py + ph/2
	}
	return ax, ay
}

type rect struct{ x, y, w, h int }

func childRect(child map[string]any, width, height int) rect {
	size, _ := child["size"].(map[string]any)
	w, h := width, height
	if size != nil {
		rawW, ok := size["width"]
		if !ok {
			rawW = size["x"]
		}
		rawH, ok := size["height"]
		if !ok {
			rawH = size["y"]
		}
		w, h = resolveSize(rawW, width), resolveSize(rawH, height)
	}
	position, _ := child["position"].(map[string]any)
	orientation, _ := child["orientation"].(string)
	x, y := anchor(orientation, 0, 0, width, height)
	if position != nil {
		x += toInt(position["x"], 0)
		y += toInt(position["y"], 0)
	}
	if centered, _ := child["centerposition"].(bool); centered {
		x -= w / 2
		y -= h / 2
	}
	return rect{x, y, max(1, w), max(1, h)}
}

func intersectionOverUnion(a, b rect) float64 {
	ix1, iy1 := max(a.x, b.x), max(a.y, b.y)
	ix2, iy2 := min(a.x+a.w, b.x+b.w), min(a.y+a.h, b.y+b.h)
	iw, ih := max(0, ix2-ix1), max(0, iy2-iy1)
	intersection := iw * ih
	if intersection == 0 {
		return 0
	}
	union := a.w*a.h + b.w*b.h - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func descendantCount(node map[string]any) int {
	count := 1
	for _, child := range nodeChildren(node) {
		count += descendantCount(child)
	}
	return count
}

func detectOverlapGroups(node map[string]any) []map[string]any {
	width, height, _ := dimensions(node)
	if width == 0 {
		width = 1920
	}
	if height == 0 {
		height = 1080
	}
	if size, ok := node["size"].(map[string]any); ok {
		rawW := size["width"]
		if rawW == nil {
			rawW = size["x"]
		}
		if _, ok := rawW.(string); ok {
			if v := resolvePercent(rawW, 1920); v != 0 {
				width = v
			}
		}
		rawH := size["height"]
		if rawH == nil {
			rawH = size["y"]
		}
		if _, ok := rawH.(string); ok {
			if v := resolvePercent(rawH, 1080); v != 0 {
				height = v
			}
		}
	}
	type candidate struct {
		index int
		rect  rect
	}
	candidates := make([]candidate, 0)
	children := nodeChildren(node)
	minimumArea := float64(width*height) * .20
	for index, child := range children {
		if child["type"] != "containerwindowtype" {
			continue
		}
		r := childRect(child, width, height)
		if float64(r.w*r.h) >= minimumArea {
			candidates = append(candidates, candidate{index, r})
		}
	}
	if len(candidates) < 2 {
		return []map[string]any{}
	}
	parent := make([]int, len(candidates))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for i := range candidates {
		for j := i + 1; j < len(candidates); j++ {
			if intersectionOverUnion(candidates[i].rect, candidates[j].rect) > .7 {
				parent[find(i)] = find(j)
			}
		}
	}
	components := map[int][]int{}
	for i := range candidates {
		root := find(i)
		components[root] = append(components[root], i)
	}
	roots := make([]int, 0, len(components))
	for root := range components {
		roots = append(roots, root)
	}
	sort.Ints(roots)
	groups := make([]map[string]any, 0)
	for _, root := range roots {
		members := components[root]
		if len(members) < 2 {
			continue
		}
		anchorRect := candidates[members[0]].rect
		memberRows := make([]map[string]any, 0, len(members))
		defaultID := ""
		for _, member := range members {
			index := candidates[member].index
			child := children[index]
			name, _ := child["name"].(string)
			if name == "" {
				name = "child" + strconv.Itoa(index)
			}
			kind := "tab"
			raw, _ := child["raw"].(map[string]any)
			for _, key := range []string{"show_animation_type", "hide_animation_type", "show_position"} {
				if _, ok := raw[key]; ok {
					kind = "overlay"
				}
			}
			if kind != "overlay" && stateNameRE.MatchString(name) && descendantCount(child) <= 3 {
				kind = "state"
			}
			id := "root/" + strconv.Itoa(index)
			if defaultID == "" || (kind == "tab" && !hasTabMember(memberRows)) {
				defaultID = id
			}
			memberRows = append(memberRows, map[string]any{"id": id, "name": name, "kind": kind})
		}
		groups = append(groups, map[string]any{
			"members":     memberRows,
			"anchor_rect": map[string]any{"x": anchorRect.x, "y": anchorRect.y, "w": anchorRect.w, "h": anchorRect.h},
			"default":     defaultID,
		})
	}
	return groups
}

func hasTabMember(rows []map[string]any) bool {
	for _, row := range rows {
		if row["kind"] == "tab" {
			return true
		}
	}
	return false
}

func (idx *Index) resolveText(raw string) map[string]any {
	if strings.Contains(raw, "[") && strings.Contains(raw, "]") {
		return map[string]any{"display": raw, "kind": "runtime"}
	}
	if value, ok := idx.locValue(raw); ok {
		out := map[string]any{"display": cleanMarkup(value), "kind": "loc"}

		if strings.ContainsAny(value, "§£$") {
			out["markup"], _ = idx.LocaliseMarkup(raw)
		}
		return out
	}
	if !looksLikeLocKey(raw) {
		return map[string]any{"display": raw, "kind": "literal"}
	}
	return map[string]any{"display": raw, "kind": "key"}
}

func cleanMarkup(value string) string {
	value = colorRE.ReplaceAllString(value, "")
	value = scriptedRE.ReplaceAllStringFunc(value, func(match string) string {
		parts := scriptedRE.FindStringSubmatch(match)
		inner := strings.TrimSpace(strings.Split(strings.TrimPrefix(parts[1], "?"), "|")[0])
		return "{" + inner + "}"
	})
	value = substRE.ReplaceAllString(value, "$1")
	return strings.TrimSpace(value)
}

func looksLikeLocKey(value string) bool {
	token := strings.TrimSpace(value)
	if token == "" || literalNum.MatchString(token) || literalX.MatchString(token) {
		return false
	}
	if strings.ToLower(token) == token && !strings.Contains(token, "_") {
		return false
	}
	return locKeyRE.MatchString(token)
}

func fontFamily(name string) any {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lower, "aldrich"):
		return "aldrich"
	case strings.HasPrefix(lower, "bomb"):
		return "bomb"
	case strings.Contains(lower, "times") || strings.Contains(lower, "typewriter"):
		return "times"
	default:
		return nil
	}
}

func (idx *Index) fontMeta(name string) map[string]any {
	size := 16
	if match := fontSizeRE.FindString(name); match != "" {
		size, _ = strconv.Atoi(match)
	}
	key := strings.ToLower(name)
	info, ok := idx.fontRecordFor(key)
	if !ok {
		info, ok = idx.fontRecordFor(strings.ReplaceAll(key, "_closing", ""))
	}
	outline := true
	ratio := 1.0
	if ok {
		outline = info.Outline
		if info.Line != 0 && info.Size != 0 {
			ratio = float64(int((float64(info.Line)/float64(info.Size))*1000+.5)) / 1000
		}
	} else {
		for _, hint := range []string{"numbers", "futura", "mapfont", "tnr_"} {
			if strings.Contains(key, hint) {
				outline = false
			}
		}
	}
	meta := map[string]any{"family": fontFamily(name), "size": size, "outline": outline, "line_ratio": ratio}
	if bitmap := idx.bitmapFontPayload(name); bitmap != nil {

		meta["bitmap"] = bitmap
	}
	return meta
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
