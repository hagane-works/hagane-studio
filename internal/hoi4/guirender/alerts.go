package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/reftables"
)

var alertCategoryRank = map[string]int{"LOW": 0, "MEDIUM": 1, "HIGH": 2}

func (idx *Index) indexAlertFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "alerts") {
			continue
		}
		for _, alert := range entry.Block.Entries {
			if alert == nil || alert.Key == nil || alert.Block == nil {
				continue
			}
			if category := strings.ToUpper(ciScalar(alert.Block.Entries, "category")); category != "" {
				idx.alertCategories[strings.ToLower(*alert.Key)] = category
			}
		}
	}
}

func (idx *Index) alertCategory(name string) string {
	name = strings.ToLower(name)
	for layer := idx; layer != nil; layer = layer.fallback {
		if len(layer.alertCategories) == 0 {
			continue
		}
		if category, ok := layer.alertCategories[name]; ok {
			return category
		}
		return "LOW"
	}
	return "LOW"
}

func (idx *Index) alertTexts(name string) (instant, delayed string) {
	if keys, custom := reftables.Alerts().TextKeys[name]; custom {
		instant, _ = idx.LocaliseMarkup(keys[0])
		if keys[1] != "" {
			delayed, _ = idx.LocaliseMarkup(keys[1])
		}
		return instant, delayed
	}
	instant, _ = idx.LocaliseMarkup(name + "_instant")
	text, ok := idx.LocaliseMarkup(name + "_delayed")
	if !ok {
		return instant, ""
	}

	if hint, ok := idx.LocaliseMarkup("ALERT_RIGHTCLICK"); ok && hint != "" && !strings.Contains(text, hint) {
		text += "\n" + hint
	}
	return instant, text
}

func (idx *Index) alertTemplateSprite() string {
	if win, ok := idx.Window("global_alerticon_window"); ok {
		if node, _ := win["node"].(map[string]any); node != nil {
			for _, child := range nodeChildren(node) {
				if name, _ := child["name"].(string); name == "icon" {
					if sprite, _ := child["sprite"].(string); sprite != "" {
						return sprite
					}
				}
			}
		}
	}
	return "GFX_global_alert_icons"
}

func (idx *Index) alertPositions() map[string]any {
	return idx.guiPositions("interface/alerts.gui")
}

func (idx *Index) guiPositions(file string) map[string]any {
	out := map[string]any{}
	body, ok := idx.Source(file)
	if !ok {
		return out
	}
	root, ok := parseScript(body)
	if !ok {
		return out
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "guitypes") {
			continue
		}
		for _, item := range entry.Block.Entries {
			if item == nil || item.Key == nil || item.Block == nil || !strings.EqualFold(*item.Key, "positiontype") {
				continue
			}
			name := strings.Trim(ciScalar(item.Block.Entries, "name"), "\"")
			if point := ciPoint(item.Block.Entries, "position"); name != "" && point != nil {
				out[name] = point
			}
		}
	}
	return out
}

func (idx *Index) AlertsPayload() map[string]any {
	type row struct {
		alert    reftables.EngineAlert
		category string
	}
	engine := reftables.Alerts().Engine
	rows := make([]row, 0, len(engine))
	for _, alert := range engine {
		rows = append(rows, row{alert, idx.alertCategory(alert.Name)})
	}
	sort.SliceStable(rows, func(a, b int) bool {
		return alertCategoryRank[rows[a].category] < alertCategoryRank[rows[b].category]
	})
	alerts := make([]map[string]any, 0, len(rows))
	for _, item := range rows {
		instant, delayed := idx.alertTexts(item.alert.Name)
		alerts = append(alerts, map[string]any{
			"name": item.alert.Name, "frame": item.alert.Frame,
			"instant": instant, "delayed": delayed,
			"category": item.category, "type": item.alert.Type,
		})
	}
	return map[string]any{
		"sprite": idx.alertTemplateSprite(), "alerts": alerts,
		"positions": idx.alertPositions(),
	}
}
