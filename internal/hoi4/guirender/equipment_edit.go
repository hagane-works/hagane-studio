package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

var notEquipmentStats = map[string]bool{"year": true, "priority": true, "visual_level": true}

func (idx *Index) equipmentBlock(name string) (file string, body []byte, entry *paradox.Entry, ok bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	if want == "" {
		return "", nil, nil, false
	}
	for layer := idx; layer != nil; layer = layer.fallback {
		paths := []string{}
		for path := range layer.files {
			lower := strings.ToLower(path)
			if strings.HasPrefix(lower, "common/units/equipment/") && strings.HasSuffix(lower, ".txt") {
				paths = append(paths, path)
			}
		}
		sort.Strings(paths)
		for _, path := range paths {
			root, parsed := parseScript(layer.files[path])
			if !parsed {
				continue
			}
			for _, group := range root.Entries {
				if group == nil || group.Key == nil || group.Block == nil || !strings.EqualFold(*group.Key, "equipments") {
					continue
				}
				for _, item := range group.Block.Entries {
					if item != nil && item.Key != nil && item.Block != nil && strings.EqualFold(*item.Key, want) {
						return path, layer.files[path], item, true
					}
				}
			}
		}
	}
	return "", nil, nil, false
}

func (idx *Index) EquipmentSource(name string) (string, []byte, int, bool) {
	file, body, entry, ok := idx.equipmentBlock(name)
	if !ok {
		return "", nil, 0, false
	}
	return file, body, entry.Line, true
}

func equipmentNumbers(entries []*paradox.Entry) (map[string]float64, map[string]float64) {
	numbers, resources := map[string]float64{}, map[string]float64{}
	for _, field := range entries {
		if field == nil || field.Key == nil {
			continue
		}
		key := strings.ToLower(*field.Key)
		if field.Block != nil {
			if key == "resources" {
				for _, resource := range field.Block.Entries {
					if resource != nil && resource.Key != nil && resource.Block == nil {
						if amount, ok := toFloat(scalarValue(resource.Scalar)); ok {
							resources[strings.ToLower(*resource.Key)] = amount
						}
					}
				}
			}
			continue
		}
		if notEquipmentStats[key] {
			continue
		}
		if value, ok := toFloat(scalarValue(field.Scalar)); ok {
			numbers[key] = value
		}
	}
	return numbers, resources
}

func (idx *Index) EquipmentPayload(tag, name string) map[string]any {
	file, _, entry, ok := idx.equipmentBlock(name)
	if !ok {
		return map[string]any{"name": name, "found": false}
	}
	record := idx.equipmentFor(name)
	own, ownResources := equipmentNumbers(entry.Block.Entries)
	label, _ := idx.Localise(*entry.Key)
	out := map[string]any{
		"name": *entry.Key, "found": true, "label": firstNonEmpty(label, *entry.Key), "file": file,
		"numbers": own, "resources": ownResources,
		"inherited": map[string]float64{}, "inherited_resources": map[string]float64{},
	}
	if tag = strings.ToUpper(strings.TrimSpace(tag)); tag != "" {
		out["country_key"] = tag + "_" + *entry.Key
		if own, found := idx.Localise(tag + "_" + *entry.Key); found {
			out["country_label"] = own
		}
	}
	if record != nil {
		out["archetype"] = record.Archetype
		out["is_archetype"] = record.IsArchetype
		out["year"] = record.Year
		out["parent"] = record.Parent
	}
	if record != nil && record.Archetype != "" {
		if _, _, archetype, found := idx.equipmentBlock(record.Archetype); found {
			inherited, inheritedResources := equipmentNumbers(archetype.Block.Entries)
			out["inherited"] = inherited
			out["inherited_resources"] = inheritedResources
			archetypeLabel, _ := idx.Localise(record.Archetype)
			out["archetype_label"] = firstNonEmpty(archetypeLabel, record.Archetype)
		}
	}
	return out
}
