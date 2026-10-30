package guirender

import (
	"sort"
	"strings"
)

func (idx *Index) expandDuplicateArchetypes() {
	names := make([]string, 0, len(idx.equipmentDuplicates))
	for name := range idx.equipmentDuplicates {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		duplicate := idx.equipmentDuplicates[name]
		base := idx.equipmentFor(duplicate.Archetype)
		if base == nil || !base.IsArchetype {
			continue
		}
		if _, own := idx.equipment[name]; !own {
			copied := *base
			copied.Name = name
			copied.DerivedVariantName = ""
			if len(duplicate.Types) > 0 {
				copied.Types = duplicate.Types
			}
			if duplicate.Picture != "" {
				copied.Picture = duplicate.Picture
			}
			idx.equipmentSeq++
			copied.Order = idx.equipmentSeq
			idx.equipment[name] = &copied
		}
		prefix := strings.ToLower(duplicate.Archetype)
		seen := map[string]bool{}
		types := []*equipmentRecord{}
		for layer := idx; layer != nil; layer = layer.fallback {
			for key, record := range layer.equipment {
				if seen[key] || record.IsArchetype || !strings.EqualFold(record.Archetype, duplicate.Archetype) || !strings.HasPrefix(key, prefix) {
					continue
				}
				seen[key] = true
				types = append(types, record)
			}
		}
		sort.Slice(types, func(a, b int) bool { return types[a].Order < types[b].Order })
		for _, record := range types {
			key := strings.ToLower(record.Name)
			copyName := name + key[len(prefix):]
			if _, own := idx.equipment[copyName]; own {
				continue
			}
			copied := *record
			copied.Name = copyName
			copied.Archetype = name
			copied.Copied = true
			copied.DerivedVariantName = duplicate.VariantNames[copyName]
			if parent := strings.ToLower(record.Parent); strings.HasPrefix(parent, prefix) {
				copied.Parent = name + parent[len(prefix):]
			}
			idx.equipmentSeq++
			copied.Order = idx.equipmentSeq
			idx.equipment[copyName] = &copied
		}
	}
}
