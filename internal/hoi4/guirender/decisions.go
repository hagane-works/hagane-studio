package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type DecisionCategory struct {
	Name string

	Icon    string
	Picture string

	Priority int

	Gated bool

	Debug bool

	Allowed *paradox.Block
	Visible *paradox.Block

	VisibleWhenEmpty bool

	Allow []string
	Deny  []string
}

type Decision struct {
	Name string

	Category string

	File string

	LabelKey string
	Icon     string

	Cost         string
	DaysReEnable string

	DaysRemove string

	FireOnlyOnce bool

	Targeted bool

	Debug bool

	Allowed    *paradox.Block
	Visible    *paradox.Block
	Available  *paradox.Block
	CustomCost bool
	Mission    bool
	Allow      []string
	Deny       []string

	Activation *paradox.Block

	TargetTrigger *paradox.Block
	Targets       []string
	TargetedOther bool

	Fires []string
}

func (idx *Index) indexDecisionCategoryFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || strings.HasPrefix(*entry.Key, "@") {
			continue
		}
		category := DecisionCategory{
			Name:    *entry.Key,
			Icon:    decisionIconToken(entry.Block.Entries, "icon"),
			Picture: decisionIconToken(entry.Block.Entries, "picture"),
		}
		for _, field := range entry.Block.Entries {
			if field == nil || field.Key == nil {
				continue
			}
			switch strings.ToLower(*field.Key) {
			case "priority":
				if field.Block != nil {

					category.Priority = toInt(ciScalarValue(field.Block.Entries, "base"), 0)
					continue
				}
				category.Priority = toInt(scalarValue(field.Scalar), 0)
			case "visible":

				category.Gated = category.Gated || (field.Block != nil && len(field.Block.Entries) > 0)
				category.Debug = category.Debug || decisionIsDebugOnly(field.Block)
				if field.Block != nil {
					category.Visible = field.Block
				}
			case "visible_when_empty":
				category.VisibleWhenEmpty = strings.EqualFold(toString(scalarValue(field.Scalar)), "yes")
			case "allowed":
				if field.Block != nil {
					collectDecisionTags(field.Block.Entries, false, &category.Allow, &category.Deny)
					category.Allowed = field.Block
				}
			}
		}
		idx.decisionCategories = append(idx.decisionCategories, category)
	}
}

func (idx *Index) indexDecisionFile(file string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}

	for name, value := range fileVariables(root.Entries) {
		idx.decisionConstants[name] = value
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || strings.HasPrefix(*entry.Key, "@") {
			continue
		}
		category := *entry.Key
		for _, block := range entry.Block.Entries {
			if block == nil || block.Key == nil || block.Block == nil {
				continue
			}
			decision := Decision{
				Name:         *block.Key,
				Category:     category,
				File:         file,
				LabelKey:     *block.Key,
				Icon:         decisionIconToken(block.Block.Entries, "icon"),
				Cost:         decisionScalarToken(block.Block.Entries, "cost"),
				DaysReEnable: decisionScalarToken(block.Block.Entries, "days_re_enable"),
				DaysRemove:   decisionScalarToken(block.Block.Entries, "days_remove"),
			}
			if override := strings.Trim(ciScalar(block.Block.Entries, "name"), "\""); override != "" {
				decision.LabelKey = override
			}
			for _, field := range block.Block.Entries {
				if field == nil || field.Key == nil {
					continue
				}
				switch strings.ToLower(*field.Key) {
				case "fire_only_once":
					decision.FireOnlyOnce = strings.EqualFold(toString(scalarValue(field.Scalar)), "yes")
				case "target_trigger":
					decision.Targeted = decision.Targeted || field.Block != nil
					if field.Block != nil {
						decision.TargetTrigger = field.Block
					}
				case "targets":
					if field.Block != nil {
						decision.Targets = blockTokens(field.Block.Entries)
					} else {
						decision.TargetedOther = true
					}
				case "target_array", "state_target", "target_root_trigger":
					decision.TargetedOther = true
				case "visible":
					decision.Debug = decision.Debug || decisionIsDebugOnly(field.Block)
					if field.Block != nil {
						decision.Visible = field.Block
					}
				case "available":
					if field.Block != nil {
						decision.Available = field.Block
					}
				case "custom_cost_trigger":
					decision.CustomCost = true
				case "days_mission_timeout":
					decision.Mission = true
				case "activation":
					decision.Mission = true
					if field.Block != nil {
						decision.Activation = field.Block
					}
				case "allowed":
					if field.Block != nil {
						collectDecisionTags(field.Block.Entries, false, &decision.Allow, &decision.Deny)
						decision.Allowed = field.Block
					}
				}
			}
			key := strings.ToLower(category)
			firedEventIDs(block.Block.Entries, &decision.Fires)
			idx.decisions[key] = append(idx.decisions[key], decision)
		}
	}
}

func decisionIconToken(entries []*paradox.Entry, field string) string {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || !strings.EqualFold(*entry.Key, field) {
			continue
		}
		if entry.Block == nil {
			return strings.Trim(toString(scalarValue(entry.Scalar)), "\"")
		}
		if key := strings.Trim(ciScalar(entry.Block.Entries, "key"), "\""); key != "" {
			return key
		}
		for _, candidate := range entry.Block.Entries {
			if candidate != nil && candidate.Key != nil && !strings.HasPrefix(*candidate.Key, "@") {
				return *candidate.Key
			}
		}
	}
	return ""
}

func decisionIsDebugOnly(visible *paradox.Block) bool {
	if visible == nil {
		return false
	}
	for _, entry := range visible.Entries {
		if entry == nil || entry.Key == nil || entry.Block != nil {
			continue
		}
		if strings.EqualFold(*entry.Key, "is_debug") &&
			strings.EqualFold(toString(scalarValue(entry.Scalar)), "yes") {
			return true
		}
	}
	return false
}

func decisionScalarToken(entries []*paradox.Entry, field string) string {
	return strings.Trim(ciScalar(entries, field), "\"")
}

func decisionIconSprite(token string) string { return decisionSpriteName(token, "GFX_decision_") }

func decisionCategorySprite(token string) string {
	return decisionSpriteName(token, "GFX_decision_category_")
}

func decisionSpriteName(token, prefix string) string {
	token = strings.TrimSpace(token)
	switch {
	case token == "":
		return ""
	case strings.HasPrefix(token, "GFX_"):
		return token
	case strings.HasPrefix(token, "decision_"):
		return "GFX_" + token
	default:
		return prefix + token
	}
}

func (idx *Index) decisionNumber(token string) (float64, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, false
	}
	if at := strings.LastIndex(token, "?"); at >= 0 {
		token = strings.TrimSpace(token[at+1:])
	}
	if value, ok := toFloat(token); ok {
		return value, true
	}
	if strings.HasPrefix(token, "@") {
		for layer := idx; layer != nil; layer = layer.fallback {
			if value, ok := layer.decisionConstants[strings.ToLower(token)]; ok {
				return value, true
			}
		}
	}
	return 0, false
}

func collectDecisionTags(entries []*paradox.Entry, negated bool, allow, deny *[]string) {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		if entry.Block != nil {
			switch key {
			case "or", "and":
				collectDecisionTags(entry.Block.Entries, negated, allow, deny)
			case "not", "nor":
				collectDecisionTags(entry.Block.Entries, !negated, allow, deny)
			}
			continue
		}
		if key != "original_tag" && key != "tag" {
			continue
		}
		tag := strings.ToUpper(strings.Trim(toString(scalarValue(entry.Scalar)), "\" "))
		if len(tag) != 3 || strings.ContainsAny(tag, ":.") {
			continue
		}
		if negated {
			*deny = append(*deny, tag)
			continue
		}
		*allow = append(*allow, tag)
	}
}

func decisionTagAllowed(allow, deny []string, tag string) bool {
	for _, denied := range deny {
		if strings.EqualFold(denied, tag) {
			return false
		}
	}
	if len(allow) == 0 {
		return true
	}
	for _, allowed := range allow {
		if strings.EqualFold(allowed, tag) {
			return true
		}
	}
	return false
}

func (idx *Index) DecisionCategories() []DecisionCategory {
	out := []DecisionCategory{}
	seen := map[string]struct{}{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, category := range layer.decisionCategories {
			key := strings.ToLower(category.Name)
			if _, shadowed := seen[key]; shadowed {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, category)
		}
	}
	return out
}

func (idx *Index) DecisionsIn(category string) []Decision {
	key := strings.ToLower(category)
	topmost := map[string]Decision{}
	var layers []*Index
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
		for _, decision := range layer.decisions[key] {
			name := strings.ToLower(decision.Name)
			if _, shadowed := topmost[name]; !shadowed {
				topmost[name] = decision
			}
		}
	}
	out := []Decision{}
	placed := map[string]bool{}
	for i := len(layers) - 1; i >= 0; i-- {
		for _, decision := range layers[i].decisions[key] {
			name := strings.ToLower(decision.Name)
			if placed[name] {
				continue
			}
			placed[name] = true
			out = append(out, topmost[name])
		}
	}
	return out
}

func (idx *Index) DecisionsPayload(tag string) map[string]any {
	return idx.decisionsPayload(tag, "")
}

func (idx *Index) decisionsPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	categories := []map[string]any{}
	for _, category := range idx.DecisionCategories() {

		if category.Debug || !decisionTagAllowed(category.Allow, category.Deny, tag) || !strictTagAllowed(category.Visible, tag) {
			continue
		}
		rows := []map[string]any{}
		for _, decision := range idx.DecisionsIn(category.Name) {
			if decision.Debug {
				continue
			}

			if !decisionTagAllowed(decision.Allow, decision.Deny, tag) || !strictTagAllowed(decision.Visible, tag) {
				continue
			}
			row := idx.decisionPayload(decision, tag, date)
			row["origin"] = idx.decisionOrigin(category.Name, decision.Name)
			rows = append(rows, row)
		}

		if len(rows) == 0 && !category.VisibleWhenEmpty {
			continue
		}
		item := map[string]any{
			"name": category.Name, "priority": category.Priority, "decisions": rows,
			"origin": idx.decisionCategoryOrigin(category.Name),
		}

		if label, ok := idx.LocaliseFor(category.Name, tag, date); ok {
			item["label"] = label
		}

		if desc, ok := idx.LocaliseFor(category.Name+"_desc", tag, date); ok && strings.TrimSpace(desc) != "" {
			item["desc"] = desc
		}
		if sprite := decisionCategorySprite(category.Icon); sprite != "" {
			item["icon"] = sprite
		}
		if sprite := decisionCategorySprite(category.Picture); sprite != "" {
			item["picture"] = sprite
		}
		if category.Gated {
			item["conditional"] = true
		}
		categories = append(categories, item)
	}
	return map[string]any{"tag": tag, "categories": categories}
}

func (idx *Index) DecisionsPayloadAt(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	payload := idx.decisionsPayload(tag, date)
	items, _ := payload["categories"].([]map[string]any)
	state := idx.decisionStartOf(tag, date)
	defined := map[string]DecisionCategory{}
	for _, category := range idx.DecisionCategories() {
		defined[strings.ToLower(category.Name)] = category
	}
	type ranked struct {
		item  map[string]any
		shown bool
		prio  int
	}
	tabs := make([]ranked, 0, len(items))
	for _, item := range items {
		name, _ := item["name"].(string)
		category := defined[strings.ToLower(name)]
		missions := map[string]bool{}
		for _, decision := range idx.DecisionsIn(name) {
			missions[strings.ToLower(decision.Name)] = decision.Mission
		}

		gate := idx.everyCategoryLayerHolds(state, name)
		rows, _ := item["decisions"].([]map[string]any)
		shownRows, missionRows, hiddenRows := []map[string]any{}, []map[string]any{}, []map[string]any{}
		for _, row := range rows {
			id, _ := row["name"].(string)
			shown := gate && idx.everyDecisionLayerShows(state, name, id)
			row["shown"] = shown
			mission := missions[strings.ToLower(id)]
			if mission {
				row["mission"] = true
			}
			switch {
			case !shown:
				hiddenRows = append(hiddenRows, row)
			case mission:
				missionRows = append(missionRows, row)
			default:
				shownRows = append(shownRows, row)
			}
		}
		shown := gate && (len(shownRows)+len(missionRows) > 0 || category.VisibleWhenEmpty)
		foreignTab := !shown && idx.namedForAnotherCountry(name, tag)
		kept := []map[string]any{}
		for _, row := range hiddenRows {
			if id, _ := row["name"].(string); !foreignTab && !idx.namedForAnotherCountry(id, tag) {
				kept = append(kept, row)
			}
		}
		if !shown && (foreignTab || (len(rows) > 0 && len(kept) == 0)) {
			continue
		}
		item["decisions"] = append(append(shownRows, missionRows...), kept...)
		item["shown"] = shown
		tabs = append(tabs, ranked{item: item, shown: shown, prio: category.Priority})
	}
	sort.SliceStable(tabs, func(a, b int) bool {
		if tabs[a].shown != tabs[b].shown {
			return tabs[a].shown
		}
		return tabs[a].prio > tabs[b].prio
	})
	ordered := make([]map[string]any, 0, len(tabs))
	for _, tab := range tabs {
		ordered = append(ordered, tab.item)
	}
	payload["categories"] = ordered
	return payload
}

func (idx *Index) namedForAnotherCountry(name, tag string) bool {
	if len(name) < 4 || name[3] != '_' {
		return false
	}
	prefix := name[:3]
	if prefix != strings.ToUpper(prefix) || strings.EqualFold(prefix, tag) {
		return false
	}
	return idx.hasCountryTag(prefix)
}

func (idx *Index) decisionCategoryOrigin(category string) string {
	key := strings.ToLower(category)
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, defined := range layer.decisionCategories {
			if strings.ToLower(defined.Name) == key {
				return layerOrigin(layer)
			}
		}
	}
	return ""
}

func (idx *Index) DecisionSource(category, name string) (string, []byte, bool) {
	key, want := strings.ToLower(category), strings.ToLower(name)
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, decision := range layer.decisions[key] {
			if strings.ToLower(decision.Name) != want || decision.File == "" {
				continue
			}
			body, ok := idx.Source(decision.File)
			return decision.File, body, ok
		}
	}
	return "", nil, false
}

func (idx *Index) decisionOrigin(category, name string) string {
	key, want := strings.ToLower(category), strings.ToLower(name)
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, decision := range layer.decisions[key] {
			if strings.ToLower(decision.Name) == want {
				return layerOrigin(layer)
			}
		}
	}
	return ""
}

func layerOrigin(layer *Index) string {
	if layer.source == "vanilla" {
		return "vanilla"
	}
	return "mod"
}

func (idx *Index) decisionPayload(decision Decision, tag, date string) map[string]any {

	item := map[string]any{"name": decision.Name, "targeted": decision.Targeted, "label_key": decision.LabelKey}
	if label, ok := idx.LocaliseFor(decision.LabelKey, tag, date); ok {
		item["label"] = label
	}
	if sprite := decisionIconSprite(decision.Icon); sprite != "" {
		item["icon"] = sprite
	}
	if cost, ok := idx.decisionNumber(decision.Cost); ok {
		item["cost"] = cost
	}
	if days, ok := idx.decisionNumber(decision.DaysReEnable); ok {
		item["days_re_enable"] = int(days)
	}
	if days, ok := idx.decisionNumber(decision.DaysRemove); ok {
		item["days_remove"] = int(days)
	}
	if decision.FireOnlyOnce {
		item["fire_only_once"] = true
	}

	if decision.Icon != "" {
		item["icon_token"] = decision.Icon
	}

	if decision.File != "" {
		item["file"] = decision.File
	}
	return item
}
