package guirender

import (
	"sort"
	"strings"
)

const buildingIconSprite = "GFX_buildings_strip"

type buildingRecord struct {
	Name string

	BaseCost int

	IconFrame int

	Specialization string

	StateMax    int
	ProvinceMax int
	IsProvince  bool

	SharesSlots bool
	Facility    bool

	Buildable bool

	HideIfMissingTech bool

	Order int

	CountryModifiers map[string]float64
}

func (record *buildingRecord) maxLevel() int {
	if record.IsProvince {
		return record.ProvinceMax
	}
	return record.StateMax
}

func (record *buildingRecord) group() string {
	switch {
	case record.Facility:
		return "facility"
	case record.IsProvince:
		return "province"
	case record.SharesSlots:
		return "shared"
	default:
		return "state"
	}
}

func (idx *Index) indexBuildingFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}

	expandConstants(root)
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil ||
			!strings.EqualFold(*entry.Key, "buildings") {
			continue
		}
		for _, building := range entry.Block.Entries {
			if building == nil || building.Key == nil || building.Block == nil ||
				strings.HasPrefix(*building.Key, "@") {
				continue
			}
			name := strings.ToLower(*building.Key)
			if _, seen := idx.buildings[name]; seen {
				continue
			}
			record := &buildingRecord{
				Name:              *building.Key,
				Buildable:         !strings.EqualFold(ciScalar(building.Block.Entries, "is_buildable"), "no"),
				HideIfMissingTech: strings.EqualFold(ciScalar(building.Block.Entries, "hide_if_missing_tech"), "yes"),
				Order:             len(idx.buildings),
			}
			if cost, ok := toFloat(ciScalarValue(building.Block.Entries, "base_cost")); ok {
				record.BaseCost = int(cost)
			}
			if frame, ok := strictInt(ciScalarValue(building.Block.Entries, "icon_frame")); ok {
				record.IconFrame = frame
			}
			for _, field := range building.Block.Entries {
				if field == nil || field.Key == nil || field.Block == nil {
					continue
				}
				switch strings.ToLower(*field.Key) {
				case "level_cap":
					if value, ok := strictInt(ciScalarValue(field.Block.Entries, "state_max")); ok {
						record.StateMax = value
					}
					if value, ok := strictInt(ciScalarValue(field.Block.Entries, "province_max")); ok {
						record.ProvinceMax, record.IsProvince = value, true
					}
					record.SharesSlots = strings.EqualFold(ciScalar(field.Block.Entries, "shares_slots"), "yes")
				case "country_modifiers":
					for _, group := range field.Block.Entries {
						if group == nil || group.Key == nil || group.Block == nil ||
							!strings.EqualFold(*group.Key, "modifiers") {
							continue
						}
						for _, modifier := range group.Block.Entries {
							if modifier == nil || modifier.Key == nil || modifier.Block != nil {
								continue
							}
							if value, ok := toFloat(scalarValue(modifier.Scalar)); ok {
								if record.CountryModifiers == nil {
									record.CountryModifiers = map[string]float64{}
								}
								record.CountryModifiers[strings.ToLower(*modifier.Key)] += value
							}
						}
					}
				case "specialization":
					if tokens := blockTokens(field.Block.Entries); len(tokens) > 0 {
						record.Specialization = strings.ToLower(tokens[0])
					}
				case "tags":

					for _, tag := range blockTokens(field.Block.Entries) {
						if strings.EqualFold(tag, "facility") {
							record.Facility = true
						}
					}
				}
			}
			idx.buildings[name] = record
		}
	}
}

func (idx *Index) builtBuildings(tag, date string) map[string]int {
	out := map[string]int{}
	for _, state := range idx.effectiveStates() {
		if idx.ownerOfState(state, date) != tag {
			continue
		}
		for name, level := range state.buildingsAt(date) {
			out[name] += level
		}
	}
	return out
}

func (idx *Index) BuildingsPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	built := idx.builtBuildings(tag, date)
	enabled := map[string]bool{}
	for _, name := range idx.ResearchedBy(tag, date) {
		if record := idx.technologyFor(name); record != nil {
			for _, building := range record.EnableBuildings {
				enabled[building] = true
			}
		}
	}

	seen := map[string]bool{}
	records := []*buildingRecord{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for key, record := range layer.buildings {
			if seen[key] {
				continue
			}
			seen[key] = true
			if !record.Buildable {
				continue
			}
			if record.HideIfMissingTech && !enabled[key] {
				continue
			}
			records = append(records, record)
		}
	}
	sort.Slice(records, func(a, b int) bool {
		if records[a].Order != records[b].Order {
			return records[a].Order < records[b].Order
		}
		return records[a].Name < records[b].Name
	})

	items := make([]map[string]any, 0, len(records))
	for _, record := range records {
		item := map[string]any{
			"name": record.Name,

			"icon":        buildingIconSprite,
			"icon_frame":  record.IconFrame,
			"base_cost":   record.BaseCost,
			"max_level":   record.maxLevel(),
			"is_province": record.IsProvince,
			"group":       record.group(),
			"built":       built[strings.ToLower(record.Name)],
		}

		if label, ok := idx.Localise(record.Name); ok {
			item["label"] = label
		}
		items = append(items, item)
	}
	return map[string]any{"tag": tag, "buildings": items}
}
