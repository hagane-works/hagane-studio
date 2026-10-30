package guirender

import (
	"strings"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/paradox"
)

type continuousPalette struct {
	id       string
	scores   map[string]float64
	base     float64
	fallback bool
	position *focusPosition
	focuses  []*continuousFocus
}

type continuousFocus struct {
	id        string
	icon      string
	dailyCost float64
	available *paradox.Block
}

type focusPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func parseFocusPosition(block *paradox.Block) *focusPosition {
	if block == nil {
		return nil
	}
	return &focusPosition{X: toInt(ciScalarValue(block.Entries, "x"), 0), Y: toInt(ciScalarValue(block.Entries, "y"), 0)}
}

func (idx *Index) indexContinuousFocusFile(body []byte) {
	text, ok := gfx.DecodeScriptText(body)
	if !ok {
		return
	}
	root, _, err := paradox.Parse(text)
	if err != nil {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "continuous_focus_palette") {
			continue
		}
		palette := &continuousPalette{id: ciScalar(entry.Block.Entries, "id")}
		if palette.id == "" {
			continue
		}
		for _, child := range entry.Block.Entries {
			if child == nil || child.Key == nil {
				continue
			}
			switch key := strings.ToLower(*child.Key); {
			case key == "country" && child.Block != nil:
				palette.scores = focusTreeScores(child.Block.Entries, nil, nil)
				palette.base, _ = toFloat(ciScalarValue(child.Block.Entries, "factor"))
			case key == "default" && child.Block == nil:
				palette.fallback = strings.EqualFold(child.Scalar.StrValue(), "yes")
			case key == "position" && child.Block != nil:
				palette.position = parseFocusPosition(child.Block)
			case key == "focus" && child.Block != nil:
				focus := &continuousFocus{id: ciScalar(child.Block.Entries, "id"), icon: ciScalar(child.Block.Entries, "icon")}
				if focus.id == "" {
					continue
				}
				focus.dailyCost, _ = toFloat(ciScalarValue(child.Block.Entries, "daily_cost"))
				for _, field := range child.Block.Entries {
					if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, "available") {
						focus.available = field.Block
					}
				}
				palette.focuses = append(palette.focuses, focus)
			}
		}
		idx.continuousPalettes[palette.id] = palette
	}
}

func (idx *Index) continuousPaletteFor(tag string) *continuousPalette {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	seen := map[string]bool{}
	var best, fallback *continuousPalette
	bestScore := 0.0
	for layer := idx; layer != nil; layer = layer.fallback {
		for id, palette := range layer.continuousPalettes {
			if seen[id] {
				continue
			}
			seen[id] = true
			score, named := palette.scores[tag]
			if !named {
				score = palette.base
			}
			if score > 0 && (best == nil || score > bestScore || (score == bestScore && id < best.id)) {
				best, bestScore = palette, score
			}
			if palette.fallback && (fallback == nil || id < fallback.id) {
				fallback = palette
			}
		}
	}
	if best != nil {
		return best
	}
	return fallback
}

func (idx *Index) ContinuousFocusPanel(tag, date, treeID string) map[string]any {
	palette := idx.continuousPaletteFor(tag)
	if palette == nil {
		return nil
	}
	position := palette.position
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.focusTrees[treeID]; ok {
			if record.continuousPosition != nil {
				position = record.continuousPosition
			}
			break
		}
	}
	if position == nil {
		position = &focusPosition{}
	}
	state := idx.decisionStartOf(strings.ToUpper(strings.TrimSpace(tag)), date)
	focuses := make([]map[string]any, 0, len(palette.focuses))
	for _, focus := range palette.focuses {
		if holds, decided := state.verdict(focus.available); decided && !holds {
			continue
		}
		icon := focus.icon
		if icon == "" {
			icon = "GFX_goal_unknown"
		}
		name := focus.id
		if label, ok := idx.Localise(focus.id); ok {
			name = label
		}
		focuses = append(focuses, map[string]any{
			"id": focus.id, "name": name, "icon": icon, "icon_sprite": idx.spritePayload(icon),
			"daily_cost": focus.dailyCost,
		})
	}
	return map[string]any{"palette": palette.id, "position": position, "focuses": focuses}
}
