package worldmap

import (
	"fmt"
	"strings"

	"haganeworks/internal/hoi4/mapsketch"
)

func normalizeWorldTerrainSource(sourceKind, unmatched string, rules []worldImageRuleIn, areas []worldImageAreaIn) (string, string, error) {
	sourceKind = strings.ToLower(strings.TrimSpace(sourceKind))
	if sourceKind == "" || sourceKind == "terrain" {
		return sourceKind, unmatched, nil
	}
	if sourceKind != "political" {
		return sourceKind, unmatched, fmt.Errorf("source_kind must be political or terrain; omit it only for the legacy terrain interpretation")
	}
	if unmatched != "" && strings.ToLower(strings.TrimSpace(unmatched)) != "plains" {
		return sourceKind, unmatched, fmt.Errorf("source_kind=political requires unmatched=plains; country colours are ownership, not terrain")
	}
	hasOcean := false
	for i, rule := range rules {
		switch strings.ToLower(strings.TrimSpace(rule.Terrain)) {
		case "ocean":
			hasOcean = true
		case "ignore":
		default:
			return sourceKind, unmatched, fmt.Errorf("source_kind=political permits only ocean/ignore rules; %q would reinterpret a country's political colour as terrain", rule.Terrain)
		}
		if rule.Tolerance < 0 {
			return sourceKind, unmatched, fmt.Errorf("political rule tolerance cannot be negative; omit it for conservative default 12")
		}
		if rule.Tolerance == 0 {
			rules[i].Tolerance = 12
		}
	}
	for _, area := range areas {
		if class := strings.ToLower(strings.TrimSpace(area.Terrain)); class != "ocean" && class != "ignore" {
			return sourceKind, unmatched, fmt.Errorf("source_kind=political permits only ocean/ignore areas; land remains plains")
		}
	}
	if !hasOcean {
		return sourceKind, unmatched, fmt.Errorf("source_kind=political requires an explicit ocean colour rule; nearest_terrain is a palette colour distance, not evidence that blue water is lakes")
	}
	return sourceKind, "plains", nil
}

func resolveWorldCountrySeeds(picture *mapsketch.Picture, inputs []worldImageSeedIn, land func(mapsketch.Point) bool) ([]mapsketch.Seed, []map[string]any, error) {
	seeds := make([]mapsketch.Seed, 0, len(inputs))
	matches := make([]map[string]any, 0, len(inputs))
	for i, input := range inputs {
		tag := strings.ToUpper(strings.TrimSpace(input.Tag))
		match := map[string]any{"tag": tag, "warnings": []string{}}
		var point mapsketch.Point
		if input.Colour != "" {
			colour, ok := mapsketch.ParseHex(input.Colour)
			if !ok {
				return seeds, matches, fmt.Errorf("seed %d (%s): colour must be #rrggbb", i+1, tag)
			}
			selection, err := mapsketch.FindColourSeed(picture, mapsketch.ColourSeedOptions{
				Colour: colour, Tolerance: input.ColourTolerance, Box: input.Box, At: input.At, Land: land,
			})
			match["mode"], match["selection"] = "colour", selection
			match["requested_colour"], match["source_colour"] = selection.RequestedColour, selection.SourceColour
			match["actual_seed"], match["match_share"], match["warnings"] = selection.ActualSeed, selection.MatchShare, selection.Warnings
			matches = append(matches, match)
			if err != nil {
				return seeds, matches, fmt.Errorf("seed %d (%s): %w", i+1, tag, err)
			}
			point = selection.ActualSeed
		} else {
			if input.Box != nil || input.ColourTolerance != 0 {
				return seeds, matches, fmt.Errorf("seed %d (%s): box/colour_tolerance require colour", i+1, tag)
			}
			switch {
			case input.At != nil:
				point = *input.At
			case input.X != nil && input.Y != nil:
				point = mapsketch.Point{X: *input.X, Y: *input.Y}
			default:
				return seeds, matches, fmt.Errorf("seed %d (%s): give colour (preferred), at, or both legacy x and y", i+1, tag)
			}
			match["mode"], match["actual_seed"] = "point", point
			matches = append(matches, match)
		}
		seeds = append(seeds, mapsketch.Seed{At: point, Tag: tag})
	}
	return seeds, matches, nil
}
