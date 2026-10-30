package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/paradox"
)

const (
	FocusSpacingX = 96
	FocusSpacingY = 130
)

type Focus struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Icon     string     `json:"icon"`
	X        int        `json:"x"`
	Y        int        `json:"y"`
	Cost     float64    `json:"cost"`
	Prereq   [][]string `json:"prereq"`
	Mutex    []string   `json:"mutex"`
	Relative string     `json:"relative_to,omitempty"`

	allowBranch []*paradox.Entry
}

type FocusTree struct {
	ID        string           `json:"id"`
	Focuses   []*Focus         `json:"focuses"`
	Shortcuts []*FocusShortcut `json:"shortcuts,omitempty"`
}

type FocusShortcut struct {
	Name   string `json:"name"`
	Label  string `json:"label"`
	Target string `json:"target"`

	Zoom float64 `json:"zoom,omitempty"`

	dlc dlcGuard
}

type focusTreeRecord struct {
	id      string
	focuses []*Focus
	shared  []string

	scores map[string]float64

	country []*paradox.Entry

	base float64

	fallback bool

	shortcuts []*FocusShortcut

	continuousPosition *focusPosition
	fileName           string
}

func (idx *Index) indexFocusTreeFile(body []byte) {
	text, ok := gfx.DecodeScriptText(body)
	if !ok {
		return
	}
	root, _, err := paradox.Parse(text)
	if err != nil {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		switch strings.ToLower(*entry.Key) {
		case "focus_tree":
			record := &focusTreeRecord{id: ciScalar(entry.Block.Entries, "id")}
			if record.id == "" {
				continue
			}
			for _, child := range entry.Block.Entries {
				if child == nil || child.Key == nil {
					continue
				}
				key := strings.ToLower(*child.Key)
				if key == "country" && child.Block != nil {
					record.country = child.Block.Entries
					record.scores = focusTreeScores(child.Block.Entries, nil, nil)
					record.base, _ = toFloat(ciScalarValue(child.Block.Entries, "factor"))
				}
				if key == "default" && child.Block == nil {
					record.fallback = strings.EqualFold(child.Scalar.StrValue(), "yes")
				}
				if key == "continuous_focus_position" && child.Block != nil {
					record.continuousPosition = parseFocusPosition(child.Block)
				}
				if key == "shortcut" && child.Block != nil {
					if shortcut := parseFocusShortcut(child.Block.Entries); shortcut != nil {
						record.shortcuts = append(record.shortcuts, shortcut)
					}
				}
				if key == "focus" && child.Block != nil {
					if focus := parseFocus(child.Block.Entries); focus != nil {
						record.focuses = append(record.focuses, focus)
						idx.focusIcons[focus.ID] = focus.Icon
						idx.focusDefs[focus.ID] = focus
						idx.recordFocusReward(focus.ID, child.Block.Entries)
					}
				}

				if key == "shared_focus" && child.Block == nil {
					record.shared = append(record.shared, child.Scalar.StrValue())
				}
			}
			idx.focusTrees[record.id] = record
		case "shared_focus":
			if focus := parseFocus(entry.Block.Entries); focus != nil {
				idx.sharedFocuses[focus.ID] = focus
				idx.focusIcons[focus.ID] = focus.Icon
				idx.focusDefs[focus.ID] = focus
				idx.recordFocusReward(focus.ID, entry.Block.Entries)
			}
		}
	}
}

func parseFocusShortcut(body []*paradox.Entry) *FocusShortcut {
	shortcut := &FocusShortcut{Name: ciScalar(body, "name"), Target: ciScalar(body, "target")}
	if shortcut.Target == "" {
		return nil
	}
	if zoom, ok := toFloat(ciScalarValue(body, "scroll_wheel_factor")); ok {
		shortcut.Zoom = zoom
	}
	for _, entry := range body {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "trigger") {
			continue
		}
		if guard, holds, ok := shortcutTriggerGuard(entry.Block.Entries); ok {
			if !holds {
				return nil
			}
			shortcut.dlc = shortcut.dlc.and(guard)
			continue
		}
		if holds, decidable := shortcutTriggerHolds(entry.Block.Entries); decidable && !holds {
			return nil
		}
	}
	return shortcut
}

func shortcutTriggerGuard(entries []*paradox.Entry) (guard dlcGuard, holds, ok bool) {
	rest := make([]*paradox.Entry, 0, len(entries))
	holds = true
	for _, entry := range entries {
		if entry != nil && entry.Key != nil && entry.Block == nil && strings.EqualFold(*entry.Key, "always") {
			holds = holds && strings.EqualFold(entry.Scalar.StrValue(), "yes")
			continue
		}
		rest = append(rest, entry)
	}
	guard, ok = conditionGuard(rest)
	return guard, holds, ok
}

func shortcutTriggerHolds(entries []*paradox.Entry) (holds, decidable bool) {
	rest := make([]*paradox.Entry, 0, len(entries))
	result := true
	for _, entry := range entries {
		if entry != nil && entry.Key != nil && entry.Block == nil && strings.EqualFold(*entry.Key, "always") {
			result = result && strings.EqualFold(entry.Scalar.StrValue(), "yes")
			continue
		}
		rest = append(rest, entry)
	}
	inner, ok := evalCondition(rest)
	if !ok {
		return false, false
	}
	return result && inner, true
}

func focusTreeScores(body []*paradox.Entry, triggers func(string) *paradox.Block, owned map[string]bool) map[string]float64 {
	scores := map[string]float64{}
	base, _ := toFloat(ciScalarValue(body, "factor"))
	for _, entry := range body {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "modifier") {
			continue
		}
		add, ok := toFloat(ciScalarValue(entry.Block.Entries, "add"))
		if !ok {
			continue
		}
		if owned != nil && !modifierGuard(entry.Block.Entries, triggers, 0).holds(owned) {
			continue
		}
		for _, tag := range modifierTags(entry.Block.Entries, triggers, 0) {
			scores[tag] += base + add
		}
	}
	return scores
}

func modifierTags(entries []*paradox.Entry, triggers func(string) *paradox.Block, depth int) []string {
	out := []string{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		if entry.Block != nil {
			if key == "not" {
				continue
			}
			out = append(out, modifierTags(entry.Block.Entries, triggers, depth)...)
			continue
		}
		if key == "tag" || key == "original_tag" {
			if tag := strings.ToUpper(toString(scalarValue(entry.Scalar))); len(tag) == 3 {
				out = append(out, tag)
			}
			continue
		}

		if triggers != nil && depth < 8 && strings.EqualFold(entry.Scalar.StrValue(), "yes") {
			if body := triggers(*entry.Key); body != nil {
				out = append(out, modifierTags(body.Entries, triggers, depth+1)...)
			}
		}
	}
	return out
}

func modifierGuard(entries []*paradox.Entry, triggers func(string) *paradox.Block, depth int) dlcGuard {
	guard := dlcGuard{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		switch {
		case key == "has_dlc" && entry.Block == nil:
			guard.Need = append(guard.Need, toString(scalarValue(entry.Scalar)))
		case (key == "not" || key == "nor") && entry.Block != nil:
			if names, ok := plainDLCs(entry.Block.Entries); ok {
				guard.Without = append(guard.Without, names...)
			}
		case key == "or" && entry.Block != nil:
			if names, ok := plainDLCs(entry.Block.Entries); ok && len(names) > 0 {
				guard.AnyOf = append(guard.AnyOf, names)
			}
		case key == "and" && entry.Block != nil:
			guard = guard.and(modifierGuard(entry.Block.Entries, triggers, depth))
		case entry.Block == nil && triggers != nil && depth < 8 && strings.EqualFold(entry.Scalar.StrValue(), "yes"):
			if body := triggers(*entry.Key); body != nil {
				guard = guard.and(modifierGuard(body.Entries, triggers, depth+1))
			}
		}
	}
	return guard
}

func callsYes(entries []*paradox.Entry) bool {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		if entry.Block != nil {
			if callsYes(entry.Block.Entries) {
				return true
			}
			continue
		}
		if strings.EqualFold(entry.Scalar.StrValue(), "yes") {
			return true
		}
	}
	return false
}

func (idx *Index) focusTreeScoresOf(record *focusTreeRecord) map[string]float64 {
	if record.country == nil || (idx.ownedDLC == nil && !callsYes(record.country)) {
		return record.scores
	}
	return focusTreeScores(record.country, idx.scriptedTrigger, idx.ownedDLC)
}

func (idx *Index) scriptedTrigger(name string) *paradox.Block {
	for layer := idx; layer != nil; layer = layer.fallback {
		if body, ok := layer.scriptedTriggers[name]; ok {
			return body
		}
		if body, ok := layer.scriptedTriggers[strings.ToLower(name)]; ok {
			return body
		}
	}
	return nil
}

func (idx *Index) indexScriptedTriggerFile(body []byte) {
	text, ok := gfx.DecodeScriptText(body)
	if !ok {
		return
	}
	root, _, err := paradox.Parse(text)
	if err != nil {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		idx.scriptedTriggers[*entry.Key] = entry.Block
		if lower := strings.ToLower(*entry.Key); lower != *entry.Key {
			if _, taken := idx.scriptedTriggers[lower]; !taken {
				idx.scriptedTriggers[lower] = entry.Block
			}
		}
	}
}

func (idx *Index) FocusTreeForTag(tag string) string {
	if choices := idx.FocusTreeChoices(tag); len(choices) > 0 {
		return choices[0].ID
	}
	return ""
}

type FocusTreeChoice struct {
	ID      string  `json:"id"`
	Score   float64 `json:"score"`
	Focuses int     `json:"focuses"`
}

func (idx *Index) FocusTreeChoices(tag string) []FocusTreeChoice {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	seen := map[string]bool{}
	var out, fallbacks []FocusTreeChoice
	for layer := idx; layer != nil; layer = layer.fallback {
		for id, record := range layer.focusTrees {
			if seen[id] {
				continue
			}
			seen[id] = true
			score, named := idx.focusTreeScoresOf(record)[tag]
			if !named {
				score = record.base
			}
			choice := FocusTreeChoice{ID: id, Score: score, Focuses: len(record.focuses) + len(record.shared)}
			if score > 0 {
				out = append(out, choice)
			} else if record.fallback {
				fallbacks = append(fallbacks, choice)
			}
		}
	}
	if len(out) == 0 {
		out = fallbacks
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Focuses != b.Focuses {
			return a.Focuses > b.Focuses
		}
		return a.ID < b.ID
	})
	return out
}

func parseFocus(body []*paradox.Entry) *Focus {
	focus := &Focus{
		ID:       ciScalar(body, "id"),
		Icon:     ciScalar(body, "icon"),
		Relative: ciScalar(body, "relative_position_id"),
	}
	if focus.ID == "" {
		return nil
	}
	focus.X = toInt(ciScalarValue(body, "x"), 0)
	focus.Y = toInt(ciScalarValue(body, "y"), 0)
	if cost, ok := toFloat(ciScalarValue(body, "cost")); ok {
		focus.Cost = cost
	}
	for _, entry := range body {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		switch strings.ToLower(*entry.Key) {
		case "allow_branch":
			focus.allowBranch = append(focus.allowBranch, entry.Block.Entries...)
		case "prerequisite":
			group := []string{}
			for _, ref := range entry.Block.Entries {
				if ref != nil && ref.Key != nil && strings.EqualFold(*ref.Key, "focus") {
					group = append(group, ref.Scalar.StrValue())
				}
			}
			if len(group) > 0 {
				focus.Prereq = append(focus.Prereq, group)
			}
		case "mutually_exclusive":
			for _, ref := range entry.Block.Entries {
				if ref != nil && ref.Key != nil && strings.EqualFold(*ref.Key, "focus") {
					focus.Mutex = append(focus.Mutex, ref.Scalar.StrValue())
				}
			}
		}
	}
	return focus
}

func (idx *Index) ListFocusTrees() []string {
	seen := map[string]struct{}{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for id := range layer.focusTrees {
			seen[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (idx *Index) FocusTreeByID(id string) (*FocusTree, bool) {
	var record *focusTreeRecord
	for layer := idx; layer != nil && record == nil; layer = layer.fallback {
		if found, ok := layer.focusTrees[id]; ok {
			record = found
		}
	}
	if record == nil {
		return nil, false
	}
	focuses := make([]*Focus, 0, len(record.focuses)+len(record.shared))
	for _, focus := range record.focuses {
		copied := *focus
		focuses = append(focuses, &copied)
	}
	for _, shared := range idx.expandSharedFocuses(record.shared) {
		copied := *shared
		focuses = append(focuses, &copied)
	}
	resolveFocusPositions(focuses)
	focuses = idx.visibleFocusBranches(focuses)
	for _, focus := range focuses {
		if name, ok := idx.Localise(focus.ID); ok {
			focus.Name = name
		} else {
			focus.Name = focus.ID
		}
	}
	sort.Slice(focuses, func(i, j int) bool {
		if focuses[i].Y != focuses[j].Y {
			return focuses[i].Y < focuses[j].Y
		}
		return focuses[i].X < focuses[j].X
	})
	present := make(map[string]bool, len(focuses))
	for _, focus := range focuses {
		present[focus.ID] = true
	}
	var shortcuts []*FocusShortcut
	for _, shortcut := range record.shortcuts {

		if !present[shortcut.Target] || !shortcut.dlc.holds(idx.ownedDLC) {
			continue
		}
		copied := *shortcut
		copied.Label = copied.Name
		if label, ok := idx.Localise(copied.Name); ok {
			copied.Label = label
		}
		shortcuts = append(shortcuts, &copied)
	}
	return &FocusTree{ID: record.id, Focuses: focuses, Shortcuts: shortcuts}, true
}

func resolveFocusPositions(focuses []*Focus) {
	byID := make(map[string]*Focus, len(focuses))
	for _, focus := range focuses {
		byID[focus.ID] = focus
	}
	done := map[string]bool{}
	for _, focus := range focuses {
		if focus.Relative == "" {
			done[focus.ID] = true
		}
	}
	for progress := true; progress; {
		progress = false
		for _, focus := range focuses {
			if done[focus.ID] {
				continue
			}
			anchor, ok := byID[focus.Relative]
			if !ok {
				done[focus.ID] = true
				progress = true
				continue
			}
			if !done[anchor.ID] {
				continue
			}
			focus.X += anchor.X
			focus.Y += anchor.Y
			done[focus.ID] = true
			progress = true
		}
	}
}

func (idx *Index) FocusTreePayload(id string) (map[string]any, bool) {
	tree, ok := idx.FocusTreeByID(id)
	if !ok {
		return nil, false
	}
	art := map[string]any{}

	for key, sprite := range map[string]string{
		"unavailable":    "GFX_focus_unavailable",
		"available":      "GFX_focus_can_start",
		"completed":      "GFX_focus_completed",
		"exclusive":      "GFX_focus_link_exclusive",
		"exclusive_line": "GFX_focus_exclusive_line1",
	} {
		art[key] = idx.spritePayload(sprite)
	}
	focuses := make([]map[string]any, 0, len(tree.Focuses))
	for _, focus := range tree.Focuses {
		icon := focus.Icon
		if icon == "" {
			icon = "GFX_goal_unknown"
		}
		entry := map[string]any{
			"id": focus.ID, "name": focus.Name, "x": focus.X, "y": focus.Y,
			"cost": focus.Cost, "prereq": focus.Prereq, "mutex": focus.Mutex,
			"icon": icon, "icon_sprite": idx.spritePayload(icon),
		}

		if tooltip := idx.FocusTooltip(focus.ID); tooltip != "" {
			entry["tooltip"] = tooltip
		}

		if focus.Relative != "" {
			entry["relative_to"] = focus.Relative
		}
		focuses = append(focuses, entry)
	}
	payload := map[string]any{
		"id": tree.ID, "focuses": focuses, "art": art,
		"spacing": map[string]int{"x": FocusSpacingX, "y": FocusSpacingY},
		"font":    idx.fontMeta("hoi_16mbs"),
	}
	if len(tree.Shortcuts) > 0 {
		payload["shortcuts"] = tree.Shortcuts
	}
	return payload, true
}

func (idx *Index) expandSharedFocuses(seeds []string) []*Focus {
	pool := map[string]*Focus{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for id, focus := range layer.sharedFocuses {
			if _, taken := pool[id]; !taken {
				pool[id] = focus
			}
		}
	}
	included := map[string]bool{}
	for _, id := range seeds {
		if _, ok := pool[id]; ok {
			included[id] = true
		}
	}
	for progress := true; progress; {
		progress = false
		for id, focus := range pool {
			if included[id] {
				continue
			}
			if included[focus.Relative] {
				included[id] = true
				progress = true
				continue
			}
			for _, group := range focus.Prereq {
				for _, parent := range group {
					if included[parent] {
						included[id] = true
						progress = true
					}
				}
			}
		}
	}
	out := make([]*Focus, 0, len(included))
	for id := range included {
		out = append(out, pool[id])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
