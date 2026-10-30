package guirender

import (
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

const tradeLawGroup = "trade_laws"

func (idx *Index) minExportOf(law string) float64 {
	for layer := idx; layer != nil; layer = layer.fallback {
		if value, ok := layer.ideaMinExport[law]; ok {
			return value
		}
	}
	return 0
}

func (idx *Index) TradeLawOf(tag, date string) (string, float64) {
	law := ""
	for _, token := range idx.StartingIdeas(tag, date) {
		if idx.ideaCategoryOf(token) == tradeLawGroup {
			law = token
			break
		}
	}
	if law == "" {
		law = idx.DefaultIdeaOf(tradeLawGroup)
	}
	return law, idx.minExportOf(law)
}

func (idx *Index) resourcesByOwner(date string) map[string]map[string]int {
	out := map[string]map[string]int{}
	for _, state := range idx.effectiveStates() {
		owner := idx.ownerOfState(state, date)
		if owner == "" || len(state.Resources) == 0 {
			continue
		}
		into := out[owner]
		if into == nil {
			into = map[string]int{}
			out[owner] = into
		}
		for name, amount := range state.Resources {
			into[name] += amount
		}
	}
	return out
}

func (idx *Index) TradePayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	world := idx.resourcesByOwner(date)
	frames := idx.ResourceFrames()

	names := make([]string, 0, len(frames))
	for name := range frames {
		names = append(names, name)
	}
	sort.Slice(names, func(a, b int) bool {
		if frames[names[a]] != frames[names[b]] {
			return frames[names[a]] < frames[names[b]]
		}
		return names[a] < names[b]
	})

	law, export := idx.TradeLawOf(tag, date)

	balance := idx.resourceBalance(tag, date)
	columns := make([]map[string]any, 0, len(names))
	for _, name := range names {
		columns = append(columns, map[string]any{
			"resource":   name,
			"frame":      frames[name],
			"produced":   balance["extracted"][name],
			"imported":   balance["imported"][name],
			"exported":   balance["exported"][name],
			"projects":   balance["projects"][name],
			"production": balance["production"][name],
			"surplus":    balance["surplus"][name],
		})
	}

	offered := map[string]map[string]int{}
	add := func(who, resource string, amount int) {
		if amount <= 0 {
			return
		}
		into := offered[who]
		if into == nil {
			into = map[string]int{}
			offered[who] = into
		}
		into[resource] += amount
	}
	for other, resources := range world {
		_, share := idx.TradeLawOf(other, date)
		if share <= 0 {
			continue
		}
		overlord, toOverlord := idx.overlordShare(other)
		for name, amount := range resources {
			sellable := float64(amount) * share
			if overlord != "" && toOverlord > 0 {
				add(overlord, name, int(sellable*toOverlord))
				sellable *= 1 - toOverlord
			}
			add(other, name, int(sellable))
		}
	}

	sellers := make([]map[string]any, 0, len(offered))
	for other, available := range offered {
		if other == tag {
			continue
		}
		total := 0
		for _, amount := range available {
			total += amount
		}
		if total == 0 {
			continue
		}
		row := map[string]any{
			"tag":       other,
			"available": available,
			"total":     total,
		}
		if overlord, _ := idx.overlordShare(other); overlord != "" {
			row["overlord"] = overlord
		}
		if name := idx.CountryName(other, idx.RulingPartyOf(other)); name != "" {
			row["name"] = name
		}

		if flag := idx.flagPayload(other, idx.RulingPartyOf(other), "medium"); flag != nil {
			row["flag"] = flag
		}
		sellers = append(sellers, row)
	}

	sort.Slice(sellers, func(a, b int) bool {
		ta, _ := sellers[a]["total"].(int)
		tb, _ := sellers[b]["total"].(int)
		if ta != tb {
			return ta > tb
		}
		sa, _ := sellers[a]["tag"].(string)
		sb, _ := sellers[b]["tag"].(string)
		return sa < sb
	})

	out := map[string]any{
		"tag":             tag,
		"trade_law":       law,
		"export_share":    export,
		"resources":       columns,
		"resource_frames": frames,
		"sellers":         sellers,
	}
	if label, ok := idx.Localise(law); ok {
		out["trade_law_label"] = label
	}
	out["factories"] = idx.tradeFactoryTexts(tag, date)
	return out
}

func (idx *Index) tradeFactoryTexts(tag, date string) map[string]any {
	construction := idx.ConstructionPayload(tag, date)
	civilian, _ := construction["civilian"].(int)
	consumer, _ := construction["consumer"].(int)
	projects, _ := construction["projects"].(int)
	available := max(0, civilian-consumer-projects)
	text := func(key, fallback, slot string, value int, colour string) string {
		raw, ok := idx.locValue(key)
		if !ok {
			raw = fallback
		}
		return locSlot.ReplaceAllStringFunc(raw, func(found string) string {
			if strings.HasPrefix(found, "$"+slot) {
				return "§" + colour + strconv.Itoa(value) + "§!"
			}
			return found
		})
	}
	return map[string]any{
		"total":          civilian,
		"used":           consumer,
		"available":      available,
		"total_text":     "§G" + strconv.Itoa(civilian) + "§!",
		"used_text":      text("TRADE_CIV_FACTORIES_USED", "Used: $USED|Y$", "USED", consumer, "Y"),
		"available_text": text("TRADE_CIV_FACTORIES_AVAIL", "Available: $AVAIL$", "AVAIL", available, "Y"),
	}
}

type subjectRecord struct {
	Overlord string
	Level    string

	Freedom float64
}

func (idx *Index) indexAutonomyFile(name string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil ||
			!strings.EqualFold(*entry.Key, "autonomy_state") {
			continue
		}
		id := strings.ToLower(ciScalar(entry.Block.Entries, "id"))
		if id == "" {
			continue
		}
		if !containsFold(idx.autonomyIDs, id) {
			idx.autonomyIDs = append(idx.autonomyIDs, id)
		}
		if idx.autonomyDefault == "" &&
			strings.EqualFold(ciScalar(entry.Block.Entries, "default"), "yes") {
			idx.autonomyDefault = id
		}
		idx.indexAutonomyLevel(id, name, entry.Line, entry.Block)
		for _, field := range entry.Block.Entries {
			if field == nil || field.Key == nil || field.Block == nil ||
				!strings.EqualFold(*field.Key, "modifier") {
				continue
			}
			if value, ok := toFloat(ciScalarValue(field.Block.Entries, "extra_trade_to_overlord_factor")); ok {
				idx.autonomyTrade[id] = value
			}
		}
	}
}

func (idx *Index) AutonomyLevels() []map[string]any {
	var layers []*Index
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
	}
	out := []map[string]any{}
	seen := map[string]bool{}
	for i := len(layers) - 1; i >= 0; i-- {
		for _, id := range layers[i].autonomyIDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, map[string]any{"id": id, "label": idx.localiseOr(id, id)})
		}
	}
	return out
}

const defaultAutonomy = "\x00default"

func (idx *Index) indexSubjects(tag string, entries []*paradox.Entry) {
	var walk func(list []*paradox.Entry)
	walk = func(list []*paradox.Entry) {
		for _, entry := range list {
			if entry == nil || entry.Key == nil {
				continue
			}
			key := strings.ToLower(*entry.Key)
			if entry.Block == nil {
				if key == "puppet" {
					target := strings.ToUpper(strings.Trim(toString(scalarValue(entry.Scalar)), "\""))
					if target != "" && target != tag {
						if _, seen := idx.subjectOf[target]; !seen {
							idx.subjectOf[target] = subjectRecord{Overlord: tag}
						}
					}
				}
				continue
			}
			if key == "set_autonomy" {
				target := strings.ToUpper(strings.Trim(ciScalar(entry.Block.Entries, "target"), "\""))
				level := strings.ToLower(ciScalar(entry.Block.Entries, "autonomous_state"))
				freedom, _ := toFloat(ciScalarValue(entry.Block.Entries, "freedom_level"))
				if target != "" && target != tag {
					if _, seen := idx.subjectOf[target]; !seen {
						idx.subjectOf[target] = subjectRecord{Overlord: tag, Level: level, Freedom: freedom}
					}
				}
				continue
			}
			walk(entry.Block.Entries)
		}
	}
	walk(entries)
}

func (idx *Index) overlordShare(tag string) (string, float64) {
	for layer := idx; layer != nil; layer = layer.fallback {
		record, ok := layer.subjectOf[tag]
		if !ok {
			continue
		}
		level := record.Level
		if level == "" {
			level = idx.defaultAutonomyLevel()
		}
		return record.Overlord, idx.autonomyTradeOf(level)
	}
	return "", 0
}

func (idx *Index) defaultAutonomyLevel() string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if layer.autonomyDefault != "" {
			return layer.autonomyDefault
		}
	}
	return ""
}

func (idx *Index) autonomyTradeOf(level string) float64 {
	for layer := idx; layer != nil; layer = layer.fallback {
		if value, ok := layer.autonomyTrade[level]; ok {
			return value
		}
	}
	return 0
}
