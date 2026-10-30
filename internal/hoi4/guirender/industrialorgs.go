package guirender

import (
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type industrialOrg struct {
	Name string

	File            string
	Include         string
	Icon            string
	EquipmentGroups []string
	HasEquipment    bool

	ResearchCategories    []string
	HasResearchCategories bool
	Allowed               *paradox.Block
	Visible               *paradox.Block
	Available             *paradox.Block
	Initial               *paradox.Block
	Traits                []mioTrait
	AddTraits             []mioTrait
	OverrideTraits        []mioTrait
	RemoveTraits          []string
	Headers               []industrialOrgHeader
}

type mioTrait struct {
	Block *paradox.Block
	Vars  map[string]float64
}

func (trait mioTrait) token() string { return ciScalar(trait.Block.Entries, "token") }

type industrialOrgHeader struct {
	Text string
	X    float64
}

type resolvedOrg struct {
	Name               string
	Icon               string
	EquipmentGroups    []string
	ResearchCategories []string
	Allowed            *paradox.Block
	Visible            *paradox.Block
	Available          *paradox.Block
	Initial            *paradox.Block
	Traits             []mioTrait
	Headers            []industrialOrgHeader
}

func (idx *Index) indexIndustrialOrgFile(file string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	vars := fileVariables(root.Entries)
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || strings.HasPrefix(*entry.Key, "@") {
			continue
		}
		org := &industrialOrg{Name: *entry.Key, File: file}
		for _, field := range entry.Block.Entries {
			if field == nil || field.Key == nil {
				continue
			}
			key := strings.ToLower(*field.Key)
			if field.Block == nil {
				switch key {
				case "include":
					org.Include = toString(scalarValue(field.Scalar))
				case "icon":
					org.Icon = toString(scalarValue(field.Scalar))
				case "remove_trait":
					org.RemoveTraits = append(org.RemoveTraits, toString(scalarValue(field.Scalar)))
				}
				continue
			}
			switch key {
			case "equipment_type":
				org.EquipmentGroups, org.HasEquipment = blockTokens(field.Block.Entries), true
			case "research_categories":
				org.ResearchCategories, org.HasResearchCategories = blockTokens(field.Block.Entries), true
			case "allowed":
				org.Allowed = field.Block
			case "visible":
				org.Visible = field.Block
			case "available":
				org.Available = field.Block
			case "initial_trait":
				org.Initial = field.Block
			case "trait":
				org.Traits = append(org.Traits, mioTrait{field.Block, vars})
			case "add_trait":
				org.AddTraits = append(org.AddTraits, mioTrait{field.Block, vars})
			case "override_trait":
				org.OverrideTraits = append(org.OverrideTraits, mioTrait{field.Block, vars})
			case "remove_trait":
				org.RemoveTraits = append(org.RemoveTraits, blockTokens(field.Block.Entries)...)
			case "tree_header_text":
				x, _ := toFloat(ciScalarValue(field.Block.Entries, "x"))
				org.Headers = append(org.Headers, industrialOrgHeader{Text: ciScalar(field.Block.Entries, "text"), X: x})
			}
		}
		if _, seen := idx.industrialOrgs[org.Name]; !seen {
			idx.industrialOrgOrder = append(idx.industrialOrgOrder, org.Name)
		}
		idx.industrialOrgs[org.Name] = org
	}
}

func (idx *Index) indexEquipmentGroupFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		for _, field := range entry.Block.Entries {
			if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, "equipment_type") {
				idx.equipmentGroups[*entry.Key] = blockTokens(field.Block.Entries)
			}
		}
	}
}

type equipmentDuplicate struct {
	Archetype string
	Types     []string

	Picture      string
	VariantNames map[string]string
}

func (idx *Index) indexDuplicateArchetypes(block *paradox.Block) {
	for _, item := range block.Entries {
		if item == nil || item.Key == nil || item.Block == nil {
			continue
		}
		duplicate := equipmentDuplicate{
			Archetype:    strings.Trim(ciScalar(item.Block.Entries, "archetype"), "\""),
			Picture:      strings.Trim(ciScalar(item.Block.Entries, "picture"), "\""),
			VariantNames: map[string]string{},
		}
		if names := childBlock(item.Block.Entries, "variant_name"); names != nil {
			for _, pair := range names.Entries {
				if pair != nil && pair.Key != nil && pair.Block == nil {
					duplicate.VariantNames[strings.ToLower(*pair.Key)] = strings.Trim(toString(scalarValue(pair.Scalar)), "\"")
				}
			}
		}
		for _, field := range item.Block.Entries {
			if field == nil || field.Key == nil || !strings.EqualFold(*field.Key, "type") {
				continue
			}
			if field.Block != nil {
				for _, token := range blockTokens(field.Block.Entries) {
					duplicate.Types = append(duplicate.Types, strings.ToLower(token))
				}
			} else if token := strings.Trim(toString(scalarValue(field.Scalar)), "\""); token != "" {
				duplicate.Types = []string{strings.ToLower(token)}
			}
		}
		idx.equipmentDuplicates[strings.ToLower(*item.Key)] = duplicate
	}
}

func (idx *Index) equipmentDuplicateFor(name string) (equipmentDuplicate, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if duplicate, ok := layer.equipmentDuplicates[strings.ToLower(name)]; ok {
			return duplicate, true
		}
	}
	return equipmentDuplicate{}, false
}

func (idx *Index) IndustrialOrgSource(name string) (string, []byte, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if org, ok := layer.industrialOrgs[name]; ok && org.File != "" {
			return org.File, layer.files[org.File], true
		}
	}
	return "", nil, false
}

func (idx *Index) industrialOrgFor(name string) *industrialOrg {
	for layer := idx; layer != nil; layer = layer.fallback {
		if org, ok := layer.industrialOrgs[name]; ok {
			return org
		}
	}
	return nil
}

func (idx *Index) equipmentGroupFor(name string) ([]string, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if types, ok := layer.equipmentGroups[name]; ok {
			return types, true
		}
	}
	return nil, false
}

func (idx *Index) industrialOrgNames() []string {
	var layers []*Index
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
	}
	seen := map[string]bool{}
	out := []string{}
	for i := len(layers) - 1; i >= 0; i-- {
		for _, name := range layers[i].industrialOrgOrder {
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out
}

func (idx *Index) resolveIndustrialOrg(name string) *resolvedOrg {
	return idx.resolveIndustrialOrgDepth(name, 0)
}

func (idx *Index) resolveIndustrialOrgDepth(name string, depth int) *resolvedOrg {
	org := idx.industrialOrgFor(name)
	if org == nil || depth > 8 {
		return nil
	}
	out := &resolvedOrg{Name: org.Name}
	if org.Include != "" {
		if base := idx.resolveIndustrialOrgDepth(org.Include, depth+1); base != nil {
			out.Icon = base.Icon
			out.EquipmentGroups = base.EquipmentGroups
			out.ResearchCategories = base.ResearchCategories
			out.Initial = base.Initial
			out.Traits = append(out.Traits, base.Traits...)
			out.Headers = base.Headers

			out.Allowed, out.Visible, out.Available = base.Allowed, base.Visible, base.Available
		}
	}
	if org.Icon != "" {
		out.Icon = org.Icon
	}
	if org.HasEquipment {
		out.EquipmentGroups = org.EquipmentGroups
	}
	if org.HasResearchCategories {
		out.ResearchCategories = org.ResearchCategories
	}
	if org.Allowed != nil {
		out.Allowed = org.Allowed
	}
	if org.Visible != nil {
		out.Visible = org.Visible
	}
	if org.Available != nil {
		out.Available = org.Available
	}
	if org.Initial != nil {
		out.Initial = mergeScriptBlock(out.Initial, org.Initial)
	}
	if len(org.Headers) > 0 {
		out.Headers = org.Headers
	}
	out.Traits = append(out.Traits, org.Traits...)
	for _, over := range org.OverrideTraits {
		for i, trait := range out.Traits {
			if trait.token() == over.token() {
				vars := map[string]float64{}
				for name, value := range trait.Vars {
					vars[name] = value
				}
				for name, value := range over.Vars {
					vars[name] = value
				}
				out.Traits[i] = mioTrait{mergeScriptBlock(trait.Block, over.Block), vars}
			}
		}
	}
	out.Traits = append(out.Traits, org.AddTraits...)
	if len(org.RemoveTraits) > 0 {
		removed := map[string]bool{}
		for _, token := range org.RemoveTraits {
			removed[token] = true
		}
		kept := out.Traits[:0:0]
		for _, trait := range out.Traits {
			if !removed[trait.token()] {
				kept = append(kept, trait)
			}
		}
		out.Traits = kept
	}
	return out
}

func mergeScriptBlock(base, over *paradox.Block) *paradox.Block {
	if base == nil {
		return over
	}
	if over == nil {
		return base
	}
	keys := map[string]bool{}
	for _, entry := range over.Entries {
		if entry != nil && entry.Key != nil {
			keys[strings.ToLower(*entry.Key)] = true
		}
	}
	out := &paradox.Block{}
	for _, entry := range base.Entries {
		if entry != nil && entry.Key != nil && keys[strings.ToLower(*entry.Key)] {
			continue
		}
		out.Entries = append(out.Entries, entry)
	}
	out.Entries = append(out.Entries, over.Entries...)
	return out
}

func (idx *Index) industrialOrgCategory(groups []string) string {
	for _, group := range groups {
		names, ok := idx.equipmentGroupFor(group)
		if !ok {
			names = []string{group}
		}
		for _, name := range names {
			types, category, known := idx.equipmentKind(name)
			if !known {
				continue
			}
			for _, kind := range types {
				if kind == "armor" {
					return "tank"
				}
			}
			switch {
			case category == "interface_category_air":
				return "plane"
			case strings.HasSuffix(category, "_ships"):
				return "ship"
			}
			return "materiel"
		}
	}
	return "materiel"
}

func (idx *Index) equipmentKind(name string) (types []string, category string, known bool) {
	if record := idx.equipmentFor(name); record != nil {
		types, category = record.Types, record.Group
		if archetype := idx.equipmentFor(record.Archetype); !record.IsArchetype && archetype != nil {
			if len(types) == 0 {
				types = archetype.Types
			}
			if category == "" {
				category = archetype.Group
			}
		}
		return types, category, true
	}
	if duplicate, ok := idx.equipmentDuplicateFor(name); ok {
		if archetype := idx.equipmentFor(duplicate.Archetype); archetype != nil {
			category = archetype.Group
		}
		return duplicate.Types, category, true
	}

	word := strings.ToLower(name)
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, record := range layer.equipment {
			if !record.IsArchetype {
				continue
			}
			for _, kind := range record.Types {
				if kind == word {
					return []string{word}, record.Group, true
				}
			}
		}
	}
	return nil, "", false
}

func (idx *Index) traitAppliesTo(trait *paradox.Block, group string) bool {
	if trait == nil {
		return false
	}
	for _, field := range trait.Entries {
		if field == nil || field.Key == nil || field.Block == nil || !strings.EqualFold(*field.Key, "limit_to_equipment_type") {
			continue
		}
		types, _ := idx.equipmentGroupFor(group)
		for _, limit := range blockTokens(field.Block.Entries) {
			if limit == group {
				return true
			}
			for _, name := range types {
				if limit == name {
					return true
				}
			}
		}
		return false
	}
	return true
}

func (idx *Index) equipmentGroupIcon(name string) string {
	if icon := "GFX_" + name; strings.HasPrefix(name, "mio_cat_eq") {
		if _, ok := idx.spriteRecordFor(icon); ok {
			return icon
		}
	}
	return "GFX_military_industrial_organization_" + name
}

func hasScriptBlock(block *paradox.Block, key string) bool {
	if block == nil {
		return false
	}
	for _, field := range block.Entries {
		if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, key) {
			return true
		}
	}
	return false
}

func (idx *Index) IndustrialOrgsPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	state := idx.decisionStartOf(tag, date)
	state.from = tag
	items := []map[string]any{}
	for _, name := range idx.industrialOrgNames() {
		org := idx.resolveIndustrialOrg(name)
		if org == nil || !state.holds(org.Allowed) || !state.holds(org.Visible) {
			continue
		}
		label := name
		if text, ok := idx.LocaliseFor(name, tag, date); ok {
			label = text
		}
		item := map[string]any{
			"token":    name,
			"name":     label,
			"category": idx.industrialOrgCategory(org.EquipmentGroups),
			"size":     0,
		}
		if org.Icon != "" {
			item["icon"] = org.Icon
			item["icon_sprite"] = idx.spritePayload(org.Icon)
		}
		if org.Initial != nil {
			if key := ciScalar(org.Initial.Entries, "name"); key != "" {
				if text, ok := idx.LocaliseFor(key, tag, date); ok {
					item["initial_trait"] = text
				} else {
					item["initial_trait"] = key
				}
			}
		}
		design := hasScriptBlock(org.Initial, "equipment_bonus")
		production := hasScriptBlock(org.Initial, "production_bonus")
		equipment := []map[string]any{}
		for _, group := range org.EquipmentGroups {
			icon := idx.equipmentGroupIcon(group)
			entry := map[string]any{"group": group, "icon": icon, "icon_sprite": idx.spritePayload(icon), "design": 0, "production": 0, "level": 0}
			if text, ok := idx.Localise(group); ok {
				entry["label"] = text
			}
			if idx.traitAppliesTo(org.Initial, group) {
				if design {
					entry["design"] = 1
				}
				if production {
					entry["production"] = 1
				}
				if design || production {
					entry["level"] = 1
				}
			}
			equipment = append(equipment, entry)
		}
		item["equipment"] = equipment
		items = append(items, item)
	}
	return map[string]any{"tag": tag, "items": items}
}
