package guirender

import (
	"fmt"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type ideaChoice struct {
	Cost          float64
	HasCost       bool
	Level         float64
	HasLevel      bool
	ConsumerGoods float64
	HasConsumer   bool
	Allowed       *paradox.Block
	Visible       *paradox.Block
	Available     *paradox.Block

	Traits []string
}

func (idx *Index) recordIdeaChoice(group, token string, entries []*paradox.Entry) {
	if group == "" {
		return
	}
	if _, seen := idx.ideaChoices[token]; !seen {
		idx.ideaOrder[group] = append(idx.ideaOrder[group], token)
	}
	choice := ideaChoice{}
	if cost, ok := toFloat(ciScalarValue(entries, "cost")); ok {
		choice.Cost, choice.HasCost = cost, true
	}
	if level, ok := toFloat(ciScalarValue(entries, "level")); ok {
		choice.Level, choice.HasLevel = level, true
	}
	for _, field := range entries {
		if field == nil || field.Key == nil || field.Block == nil {
			continue
		}
		switch strings.ToLower(*field.Key) {
		case "allowed":
			choice.Allowed = field.Block
		case "visible":
			choice.Visible = field.Block
		case "available":
			choice.Available = field.Block
		case "traits":
			choice.Traits = blockTokens(field.Block.Entries)
		case "modifier":
			if value, ok := toFloat(ciScalarValue(field.Block.Entries, "consumer_goods_expected_value")); ok {
				choice.ConsumerGoods, choice.HasConsumer = value, true
			}
		}
	}
	idx.ideaChoices[token] = choice
}

func (idx *Index) ideasOfGroup(group string) ([]string, map[string]ideaChoice) {
	var layers []*Index
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
	}
	order := []string{}
	choices := map[string]ideaChoice{}
	for i := len(layers) - 1; i >= 0; i-- {
		for _, token := range layers[i].ideaOrder[group] {
			if _, seen := choices[token]; !seen {
				order = append(order, token)
			}
			choices[token] = layers[i].ideaChoices[token]
		}
	}
	return order, choices
}

func (idx *Index) ideaStats(group, token string, choice ideaChoice) string {
	switch group {
	case "mobilization_laws":
		if value, ok := idx.lookupIdeaConscription(token); ok {
			return fmt.Sprintf("£manpower_texticon §G+%.1f%%§!", value*100)
		}
	case "trade_laws":
		if value, ok := idx.lookupIdeaMinExport(token); ok {
			return fmt.Sprintf("§Y%.0f%%§!", value*100)
		}
	case "economy":
		if choice.HasConsumer {
			return fmt.Sprintf("£consumer_goods_texticon §Y%.0f%%§!", choice.ConsumerGoods*100)
		}
	}
	return ""
}

func (idx *Index) lookupIdeaConscription(token string) (float64, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if value, ok := layer.ideaConscription[token]; ok {
			return value, true
		}
	}
	return 0, false
}

func (idx *Index) lookupIdeaMinExport(token string) (float64, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if value, ok := layer.ideaMinExport[token]; ok {
			return value, true
		}
	}
	return 0, false
}

func (idx *Index) IdeaChoicesPayload(tag, date, group string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	group = strings.ToLower(strings.TrimSpace(group))
	state := idx.decisionStartOf(tag, date)
	current := ""
	for _, token := range idx.StartingIdeas(tag, date) {
		if idx.ideaCategoryOf(token) == group {
			current = token
		}
	}
	if current == "" {
		current = idx.DefaultIdeaOf(group)
	}
	order, choices := idx.ideasOfGroup(group)

	grid := !lawGroups[group]
	ideas, later := []map[string]any{}, []map[string]any{}
	for _, token := range order {
		choice := choices[token]
		if token != current && (!state.holds(choice.Allowed) || !state.holds(choice.Visible)) {
			continue
		}
		item := idx.ideaLabels([]string{token}, "idea")[0]

		if label, ok := idx.LocaliseFor(token, tag, date); ok {
			item["label"] = label
		}
		if choice.HasCost && token != current {
			cost := choice.Cost
			if held := choices[current]; choice.HasLevel && held.HasLevel {
				steps := choice.Level - held.Level
				if steps < 0 {
					steps = -steps
				}
				cost *= steps
			}
			item["cost"] = cost
		}
		if stats := idx.ideaStats(group, token, choice); stats != "" {
			item["stats"] = stats
		}
		if token == current {
			item["current"] = true
		}
		if len(choice.Traits) > 0 {
			if label, ok := idx.Localise(choice.Traits[0]); ok {
				item["trait"] = label
			}
		}
		available := state.holds(choice.Available)
		item["available"] = available
		if grid && !available && token != current {
			later = append(later, item)
			continue
		}
		ideas = append(ideas, item)
	}
	ideas = append(ideas, later...)
	title := group
	if label, ok := idx.Localise(group); ok {
		title = label
	}
	out := map[string]any{"tag": tag, "group": group, "title": title, "ideas": ideas}
	if grid {
		out["layout"] = "grid"
		out["subtitle"] = idx.ideaWindowAddText(title)
	}
	if current != "" {
		out["current"] = current
		if label, ok := idx.Localise(current); ok {
			out["current_label"] = label
			if !grid {

				out["subtitle"] = idx.fillTooltipLoc("IDEA_WINDOW_SWITCH", "Replace §Y$IDEA$?§!", map[string]any{"IDEA": label})
			}
		}
	}
	return out
}

func (idx *Index) ideaWindowAddText(group string) string {
	return idx.fillTooltipLoc("IDEA_WINDOW_ADD", "Select $GROUP|Y$", map[string]any{"GROUP": group})
}

var lawGroups = map[string]bool{"mobilization_laws": true, "trade_laws": true, "economy": true}
