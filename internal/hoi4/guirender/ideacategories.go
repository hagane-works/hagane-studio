package guirender

import (
	"strings"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/paradox"
)

type IdeaSlot struct {
	Name      string `json:"name"`
	Character bool   `json:"character"`
}

type IdeaCategory struct {
	Name string `json:"name"`

	Index int        `json:"index"`
	Slots []IdeaSlot `json:"slots"`

	Cost int `json:"cost,omitempty"`
}

func (idx *Index) indexIdeaTagFile(body []byte) {
	text, ok := gfx.DecodeScriptText(body)
	if !ok {
		return
	}
	root, _, err := paradox.Parse(text)
	if err != nil {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "idea_categories") {
			continue
		}
		declared := 0
		for _, category := range entry.Block.Entries {
			if category == nil || category.Key == nil || category.Block == nil {
				continue
			}
			declared++
			parsed := IdeaCategory{Name: *category.Key, Index: declared}
			if kind := strings.ToLower(strings.Trim(ciScalar(category.Block.Entries, "type"), "\"")); kind != "" {
				idx.ideaTypes[strings.ToLower(*category.Key)] = kind
			}
			if cost, ok := toFloat(ciScalarValue(category.Block.Entries, "cost")); ok {
				idx.ideaCosts[strings.ToLower(*category.Key)] = cost
			}
			hidden := false
			for _, field := range category.Block.Entries {
				if field == nil || field.Key == nil {
					continue
				}
				switch strings.ToLower(*field.Key) {
				case "hidden":
					hidden = hidden || strings.EqualFold(toString(scalarValue(field.Scalar)), "yes")
				case "ledger":

					hidden = hidden || strings.EqualFold(toString(scalarValue(field.Scalar)), "hidden")
				case "slot":
					parsed.Slots = append(parsed.Slots, IdeaSlot{Name: toString(scalarValue(field.Scalar))})
				case "character_slot":
					parsed.Slots = append(parsed.Slots, IdeaSlot{Name: toString(scalarValue(field.Scalar)), Character: true})
				case "cost":
					if cost, ok := toFloat(scalarValue(field.Scalar)); ok {
						parsed.Cost = int(cost)
					}
				}
			}
			if hidden || len(parsed.Slots) == 0 {
				continue
			}
			idx.ideaCategories = append(idx.ideaCategories, parsed)
		}
	}
}

func (idx *Index) IdeaCategories() []IdeaCategory {
	if len(idx.ideaCategories) > 0 {
		return idx.ideaCategories
	}
	if idx.fallback != nil {
		return idx.fallback.IdeaCategories()
	}
	return nil
}

func (idx *Index) isSlotCategory(name string) bool {
	if name == "" {
		return false
	}
	for _, category := range idx.IdeaCategories() {
		for _, slot := range category.Slots {
			if !slot.Character && strings.EqualFold(slot.Name, name) {
				return true
			}
		}
	}
	return false
}

func (idx *Index) ideaCategoriesPayload() []map[string]any {
	out := []map[string]any{}
	for _, category := range idx.IdeaCategories() {
		label := category.Name
		if localised, ok := idx.Localise(category.Name); ok {
			label = localised
		}
		slots := []map[string]any{}
		for _, slot := range category.Slots {
			slotLabel := slot.Name
			if localised, ok := idx.Localise(slot.Name); ok {
				slotLabel = localised
			}
			slots = append(slots, map[string]any{
				"name": slot.Name, "label": slotLabel, "character": slot.Character,

				"select_text": idx.ideaWindowAddText(slotLabel),
			})
		}
		out = append(out, map[string]any{
			"name": category.Name, "label": label, "index": category.Index, "slots": slots,
		})
	}
	return out
}

func (idx *Index) characterSlotCost(slot string) int {
	for _, category := range idx.IdeaCategories() {
		for _, candidate := range category.Slots {
			if candidate.Character && strings.EqualFold(candidate.Name, slot) {
				return category.Cost
			}
		}
	}
	return 0
}
