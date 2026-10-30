package guirender

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type provinceDef struct {
	ID        int
	RGB       uint32
	Kind      string
	Coastal   bool
	Terrain   string
	Continent int
}

func parseProvinceDef(line string) (provinceDef, bool) {
	fields := strings.Split(strings.TrimSpace(line), ";")
	if len(fields) < 8 {
		return provinceDef{}, false
	}
	id, err := strconv.Atoi(fields[0])
	if err != nil || id <= 0 {
		return provinceDef{}, false
	}
	var rgb [3]int
	for i := range rgb {
		value, err := strconv.Atoi(strings.TrimSpace(fields[1+i]))
		if err != nil || value < 0 || value > 255 {
			return provinceDef{}, false
		}
		rgb[i] = value
	}
	continent, _ := strconv.Atoi(strings.TrimSpace(fields[7]))
	return provinceDef{
		ID:        id,
		RGB:       uint32(rgb[0])<<16 | uint32(rgb[1])<<8 | uint32(rgb[2]),
		Kind:      strings.ToLower(strings.TrimSpace(fields[4])),
		Coastal:   strings.EqualFold(strings.TrimSpace(fields[5]), "true"),
		Terrain:   strings.TrimSpace(fields[6]),
		Continent: continent,
	}, true
}

func countryColor(entries []*paradox.Entry) ([3]uint8, bool) {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "color") {
			continue
		}
		tokens := blockTokens(entry.Block.Entries)
		if len(tokens) < 3 {
			return [3]uint8{}, false
		}

		var values [3]float64
		atMostOne, notWhole := true, false
		for i := range values {
			value, err := strconv.ParseFloat(tokens[i], 64)
			if err != nil {
				return [3]uint8{}, false
			}
			values[i] = value
			atMostOne = atMostOne && value <= 1
			notWhole = notWhole || value != math.Trunc(value)
		}
		fraction := atMostOne && notWhole
		if entry.Qualifier != nil && strings.EqualFold(*entry.Qualifier, "hsv") {
			return hsvToRGB(values[0], values[1], values[2]), true
		}
		var out [3]uint8
		for i, value := range values {
			if fraction {
				value *= 255
			}
			out[i] = uint8(math.Round(math.Max(0, math.Min(255, value))))
		}
		return out, true
	}
	return [3]uint8{}, false
}

func hsvToRGB(h, s, v float64) [3]uint8 {
	h = math.Mod(h, 1)
	if h < 0 {
		h++
	}
	i := math.Floor(h * 6)
	f := h*6 - i
	p, q, t := v*(1-s), v*(1-f*s), v*(1-(1-f)*s)
	var r, g, b float64
	switch int(i) % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}
	to := func(x float64) uint8 { return uint8(math.Round(math.Max(0, math.Min(1, x)) * 255)) }
	return [3]uint8{to(r), to(g), to(b)}
}

func (idx *Index) CountryMapColor(tag string) ([3]uint8, bool) {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	for layer := idx; layer != nil; layer = layer.fallback {
		if color, ok := layer.tagColors[tag]; ok {
			return color, true
		}
	}
	for layer := idx; layer != nil; layer = layer.fallback {
		file, ok := layer.countryFiles[tag]
		if !ok {
			continue
		}
		key := strings.ToLower(strings.Trim(file, "\""))
		for inner := idx; inner != nil; inner = inner.fallback {
			if color, found := inner.countryColors[key]; found {
				return color, true
			}
		}
	}
	return [3]uint8{}, false
}

func (idx *Index) provinceTable() []provinceDef {
	for layer := idx; layer != nil; layer = layer.fallback {
		if len(layer.provinceDefs) > 0 {
			return layer.provinceDefs
		}
	}
	return nil
}

var provinceKinds = map[string]int{"land": 0, "sea": 1, "lake": 2}

type WorldMapPayload struct {
	Provinces  [][]any             `json:"provinces"`
	States     [][]any             `json:"states"`
	Regions    [][]any             `json:"regions"`
	Points     [][]any             `json:"points"`
	Continents []string            `json:"continents"`
	Colors     map[string][3]uint8 `json:"colors"`
}

func (idx *Index) WorldMap(date string) WorldMapPayload {
	out := WorldMapPayload{Provinces: [][]any{}, States: [][]any{}, Regions: [][]any{}, Points: [][]any{}, Colors: map[string][3]uint8{}}
	stateOf := map[int]int{}
	for _, record := range idx.effectiveStates() {
		owner := idx.ownerOfState(record, date)
		name := record.Name
		if text, ok := idx.Localise(name); ok && text != "" {
			name = text
		}
		out.States = append(out.States, []any{record.ID, name, owner})
		for _, province := range record.Provinces {
			stateOf[province] = record.ID
		}
		points := make([]int, 0, len(record.VictoryPoints))
		for province := range record.VictoryPoints {
			points = append(points, province)
		}
		sort.Ints(points)
		for _, province := range points {
			name, _ := idx.Localise("VICTORY_POINTS_" + strconv.Itoa(province))
			out.Points = append(out.Points, []any{province, record.VictoryPoints[province], name})
		}
		if owner != "" {
			if _, seen := out.Colors[owner]; !seen {
				if color, ok := idx.CountryMapColor(owner); ok {
					out.Colors[owner] = color
				}
			}
		}
	}
	regions := map[int]string{}
	for _, def := range idx.provinceTable() {
		region := 0
		if found := idx.strategicRegionOf(def.ID); found != nil {
			region = found.ID
			if _, seen := regions[region]; !seen {
				name := found.Name
				if text, ok := idx.Localise(name); ok && text != "" {
					name = text
				}
				regions[region] = name
			}
		}
		coastal := 0
		if def.Coastal {
			coastal = 1
		}
		out.Provinces = append(out.Provinces, []any{def.ID, def.RGB, provinceKinds[def.Kind], coastal, def.Terrain, def.Continent, stateOf[def.ID], region})
	}
	ids := make([]int, 0, len(regions))
	for id := range regions {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		out.Regions = append(out.Regions, []any{id, regions[id]})
	}
	for _, name := range idx.continentList() {
		text := name
		if localised, ok := idx.Localise(name); ok && localised != "" {
			text = localised
		}
		out.Continents = append(out.Continents, text)
	}
	if out.Continents == nil {
		out.Continents = []string{}
	}
	return out
}
