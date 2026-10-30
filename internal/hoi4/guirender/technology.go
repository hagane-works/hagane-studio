package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type techFolderSlot struct {
	Folder string
	X      int
	Y      int
}

type technologyRecord struct {
	Name string

	Cost      float64
	StartYear int
	Doctrine  bool
	Folders   []techFolderSlot

	LeadsTo []string

	XOR        []string
	Categories []string

	Enables []string

	EnableSubunits []string

	EnableBuildings []string

	UnitModifiers map[string]map[string]float64

	CountryModifiers map[string]float64

	Order int

	SubTechnologies []string
	SubTechIndex    int

	SpecialProject bool

	ForceSmall bool
}

var technologyStructureKeys = map[string]struct{}{
	"folder": {}, "path": {}, "categories": {}, "ai_will_do": {}, "xor": {},
	"dependencies": {}, "allow": {}, "allow_branch": {}, "show_equipment_icon": {},
	"enable_equipments": {}, "enable_equipment_modules": {}, "enable_subunits": {},
	"enable_building": {}, "enable_tactic": {}, "on_research_complete": {},
	"on_research_complete_limit": {}, "research_bonus": {}, "sub_technologies": {},
	"modifier": {}, "equipment_bonus": {}, "desc": {}, "enable_decisions": {},
}

type techFolderRecord struct {
	Name string

	Ledger string
	Order  int

	Available *paradox.Block
}

func (idx *Index) indexTechnologyTagFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil ||
			!strings.EqualFold(*entry.Key, "technology_folders") {
			continue
		}
		for _, folder := range entry.Block.Entries {
			if folder == nil || folder.Key == nil || folder.Block == nil {
				continue
			}
			name := strings.ToLower(*folder.Key)
			if _, seen := idx.techFolders[name]; seen {
				continue
			}
			record := &techFolderRecord{
				Name:   *folder.Key,
				Ledger: strings.Trim(ciScalar(folder.Block.Entries, "ledger"), "\""),
				Order:  len(idx.techFolders),
			}
			for _, field := range folder.Block.Entries {
				if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, "available") {
					record.Available = field.Block
				}
			}
			idx.techFolders[name] = record
			idx.techFolderOrder = append(idx.techFolderOrder, *folder.Key)
		}
	}
}

func fileVariables(entries []*paradox.Entry) map[string]float64 {
	out := map[string]float64{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block != nil ||
			!strings.HasPrefix(*entry.Key, "@") {
			continue
		}
		if value, ok := toFloat(scalarValue(entry.Scalar)); ok {
			out[strings.ToLower(*entry.Key)] = value
		}
	}
	return out
}

func coordinate(entries []*paradox.Entry, key string, variables map[string]float64) int {
	raw := ciScalarValue(entries, key)
	if value, ok := toFloat(raw); ok {
		return int(value)
	}
	token := strings.ToLower(strings.Trim(toString(raw), "\""))
	if value, ok := variables[token]; ok {
		return int(value)
	}
	return 0
}

func (idx *Index) indexTechnologyFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil ||
			!strings.EqualFold(*entry.Key, "technologies") {
			continue
		}
		variables := fileVariables(entry.Block.Entries)
		for _, tech := range entry.Block.Entries {
			if tech == nil || tech.Key == nil || tech.Block == nil ||
				strings.HasPrefix(*tech.Key, "@") {
				continue
			}
			idx.techSeq++
			record := &technologyRecord{
				Name:           *tech.Key,
				Doctrine:       strings.EqualFold(ciScalar(tech.Block.Entries, "doctrine"), "yes"),
				Order:          idx.techSeq,
				SpecialProject: strings.EqualFold(ciScalar(tech.Block.Entries, "is_special_project_tech"), "yes"),
				ForceSmall:     strings.EqualFold(ciScalar(tech.Block.Entries, "force_use_small_tech_layout"), "yes"),
			}
			if cost, ok := toFloat(ciScalarValue(tech.Block.Entries, "research_cost")); ok {
				record.Cost = cost
			}
			if year, ok := toFloat(ciScalarValue(tech.Block.Entries, "start_year")); ok {
				record.StartYear = int(year)
			}
			if index, ok := toFloat(ciScalarValue(tech.Block.Entries, "sub_tech_index")); ok {
				record.SubTechIndex = int(index)
			}
			if values := numericEntries(tech.Block.Entries, nil); len(values) > 0 {
				delete(values, "research_cost")
				delete(values, "start_year")
				record.CountryModifiers = values
			}
			for _, field := range tech.Block.Entries {
				if field == nil || field.Key == nil || field.Block == nil {
					continue
				}
				switch strings.ToLower(*field.Key) {
				case "folder":
					slot := techFolderSlot{
						Folder: strings.Trim(ciScalar(field.Block.Entries, "name"), "\""),
					}
					for _, at := range field.Block.Entries {
						if at == nil || at.Key == nil || at.Block == nil ||
							!strings.EqualFold(*at.Key, "position") {
							continue
						}
						slot.X = coordinate(at.Block.Entries, "x", variables)
						slot.Y = coordinate(at.Block.Entries, "y", variables)
					}
					if slot.Folder != "" {
						record.Folders = append(record.Folders, slot)
					}
				case "path":
					if to := strings.Trim(ciScalar(field.Block.Entries, "leads_to_tech"), "\""); to != "" {
						record.LeadsTo = append(record.LeadsTo, to)
					}
				case "enable_equipments":
					record.Enables = append(record.Enables, blockTokens(field.Block.Entries)...)
				case "enable_subunits":
					record.EnableSubunits = append(record.EnableSubunits, blockTokens(field.Block.Entries)...)
				case "enable_building":
					if building := strings.Trim(ciScalar(field.Block.Entries, "building"), "\""); building != "" {
						record.EnableBuildings = append(record.EnableBuildings, strings.ToLower(building))
					}
				case "xor":
					record.XOR = append(record.XOR, blockTokens(field.Block.Entries)...)
				case "categories":
					record.Categories = append(record.Categories, blockTokens(field.Block.Entries)...)
				case "sub_technologies":
					record.SubTechnologies = append(record.SubTechnologies, blockTokens(field.Block.Entries)...)
				default:
					key := strings.ToLower(*field.Key)
					if _, structural := technologyStructureKeys[key]; structural {
						continue
					}
					values := map[string]float64{}
					for _, stat := range field.Block.Entries {
						if stat == nil || stat.Key == nil || stat.Block != nil {
							continue
						}
						if value, ok := toFloat(scalarValue(stat.Scalar)); ok {
							values[strings.ToLower(*stat.Key)] = value
						}
					}
					if len(values) > 0 {
						if record.UnitModifiers == nil {
							record.UnitModifiers = map[string]map[string]float64{}
						}
						record.UnitModifiers[key] = values
					}
				}
			}
			idx.technologies[strings.ToLower(record.Name)] = record
		}
	}
}

func (idx *Index) technologyFor(name string) *technologyRecord {
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.technologies[strings.ToLower(name)]; ok {
			return record
		}
	}
	return nil
}

func (idx *Index) TechFolders() []map[string]any {
	return idx.techFoldersFor("", "")
}

func (idx *Index) techFoldersFor(tag, date string) []map[string]any {
	var state *decisionStart
	if tag != "" {
		start := idx.decisionStartOf(tag, date)
		state = &start
	}
	seen := map[string]bool{}
	out := []map[string]any{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, name := range layer.techFolderOrder {
			key := strings.ToLower(name)
			if seen[key] {
				continue
			}
			seen[key] = true
			folder := layer.techFolders[key]
			if state != nil {
				if holds, decided := state.verdict(folder.Available); decided && !holds {
					continue
				}
			}
			item := map[string]any{"name": name, "ledger": folder.Ledger}
			if label, ok := idx.Localise(name); ok {
				item["label"] = label
			}
			out = append(out, item)
		}
	}
	return out
}

func (idx *Index) ResearchedBy(tag, date string) []string {
	politics := idx.countryPoliticsFor(tag)
	if politics == nil {
		return nil
	}
	return tokensByOwned(politics.Technologies, date, idx.ownedDLC)
}

func (idx *Index) TechTreePayload(tag, date, folder string) map[string]any {
	folders := idx.techFoldersFor(tag, date)
	if folder == "" && len(folders) > 0 {
		folder, _ = folders[0]["name"].(string)
	}
	held := map[string]bool{}
	for _, name := range idx.ResearchedBy(tag, date) {
		held[strings.ToLower(name)] = true
	}

	parents := map[string][]string{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for key, record := range layer.technologies {
			for _, next := range record.LeadsTo {
				parents[strings.ToLower(next)] = append(parents[strings.ToLower(next)], key)
			}
		}
	}
	designTeams := idx.designTeamCategories(tag, date)
	upperTag := strings.ToUpper(strings.TrimSpace(tag))

	seen := map[string]bool{}
	items := []map[string]any{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for key, record := range layer.technologies {
			if seen[key] {
				continue
			}
			for _, slot := range record.Folders {
				if !strings.EqualFold(slot.Folder, folder) {
					continue
				}
				seen[key] = true
				item := map[string]any{
					"name":       record.Name,
					"x":          slot.X,
					"y":          slot.Y,
					"cost":       record.Cost,
					"start_year": record.StartYear,
					"doctrine":   record.Doctrine,
					"researched": held[key],
				}
				if len(record.LeadsTo) > 0 {
					item["leads_to"] = record.LeadsTo
				}
				if len(record.XOR) > 0 {
					item["xor"] = record.XOR
				}
				if label, ok := idx.Localise(record.Name); ok {
					item["label"] = label
				}
				item["order"] = record.Order
				item["state"] = idx.techState(record, held, parents[key])

				if layer == idx && idx.fallback != nil && idx.fallback.technologyFor(record.Name) == nil {
					item["added_by_project"] = true
				}

				item["icon"] = "GFX_" + record.Name + "_medium"
				if _, ok := idx.spriteRecordFor("GFX_" + upperTag + "_" + record.Name + "_medium"); ok {
					item["icon"] = "GFX_" + upperTag + "_" + record.Name + "_medium"
				}

				if len(record.Enables) > 0 && !record.ForceSmall {
					label, key := idx.techCardLabel(record, upperTag)
					item["card"] = true
					item["card_label"] = label
					item["card_label_key"] = key
				}
				if record.SpecialProject {
					item["special_project"] = true
				}

				subs := []map[string]any{}
				for _, name := range record.SubTechnologies {
					sub := idx.technologyFor(name)
					if sub == nil {
						continue
					}
					state := "unavailable"
					if held[strings.ToLower(name)] {
						state = "researched"
					} else if held[key] {
						state = "available"
					}
					subs = append(subs, map[string]any{"name": sub.Name, "slot": sub.SubTechIndex, "state": state})
				}
				if len(subs) > 0 {
					item["sub_technologies"] = subs
				}

				if !held[key] {
					for _, category := range record.Categories {
						if designTeams[strings.ToLower(category)] {
							item["design_team"] = true
							break
						}
					}
				}
				items = append(items, item)
				break
			}
		}
	}
	sort.Slice(items, func(a, b int) bool {
		if items[a]["y"] != items[b]["y"] {
			return items[a]["y"].(int) < items[b]["y"].(int)
		}
		if items[a]["x"] != items[b]["x"] {
			return items[a]["x"].(int) < items[b]["x"].(int)
		}
		return items[a]["name"].(string) < items[b]["name"].(string)
	})

	return map[string]any{
		"tag":            strings.ToUpper(tag),
		"folder":         folder,
		"folders":        folders,
		"technologies":   items,
		"researched":     len(held),
		"research_slots": idx.researchSlotsAt(tag, date),

		"base_tech_cost": idx.defineNumber("NTechnology.BASE_TECH_COST"),

		"special_project_offset": idx.guiPositions("interface/countrytechtreeview.gui")["special_project_tech_icon_offset"],
	}
}

func (idx *Index) researchSlotsAt(tag, date string) int {
	start, ok := idx.countryStartAt(tag, date)
	if !ok {
		return 0
	}
	return start.ResearchSlots
}

func collectTechnologies(entries []*paradox.Entry, date string) []datedIdeas {
	return collectTechnologiesGuarded(entries, date, dlcGuard{})
}

func collectTechnologiesGuarded(entries []*paradox.Entry, date string, guard dlcGuard) []datedIdeas {
	out := []datedIdeas{}
	branch := branchState{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		if entry.Block != nil && (key == "if" || key == "else_if") {
			if inner, ok := branch.enterIf(entry.Block.Entries, guard); ok {
				out = append(out, collectTechnologiesGuarded(entry.Block.Entries, date, inner)...)
			}
			continue
		}
		if entry.Block != nil && key == "else" {
			if inner, ok := branch.enterElse(); ok {
				out = append(out, collectTechnologiesGuarded(entry.Block.Entries, date, inner)...)
			}
			continue
		}
		if key == "set_technology" && entry.Block != nil {
			tokens := []string{}
			for _, tech := range entry.Block.Entries {
				if tech == nil || tech.Key == nil || tech.Block != nil {
					continue
				}
				if value, ok := toFloat(scalarValue(tech.Scalar)); ok && value == 0 {
					continue
				}
				tokens = append(tokens, *tech.Key)
			}
			if len(tokens) > 0 {
				out = append(out, datedIdeas{DLC: guard, Date: date, Tokens: tokens})
			}
			continue
		}
		if entry.Block != nil && isGameDate(key) {
			out = append(out, collectTechnologiesGuarded(entry.Block.Entries, key, guard)...)
		}
	}
	return out
}

func (idx *Index) ResearchedModifiers(tag, date string) map[string]map[string]float64 {
	out := map[string]map[string]float64{}
	for _, name := range idx.ResearchedBy(tag, date) {
		record := idx.technologyFor(name)
		if record == nil {
			continue
		}
		for target, values := range record.UnitModifiers {
			if out[target] == nil {
				out[target] = map[string]float64{}
			}
			for stat, value := range values {
				out[target][stat] += value
			}
		}
	}
	return out
}

func (idx *Index) FieldedEquipment(tag, date string) map[string]bool {
	out := map[string]bool{}
	for _, name := range idx.ResearchedBy(tag, date) {
		record := idx.technologyFor(name)
		if record == nil {
			continue
		}
		for _, model := range record.Enables {
			out[strings.ToLower(model)] = true
		}
	}
	return out
}

func (idx *Index) techState(record *technologyRecord, held map[string]bool, parents []string) string {
	key := strings.ToLower(record.Name)
	if held[key] {
		return "researched"
	}
	for _, other := range record.XOR {
		if held[strings.ToLower(other)] {
			return "unavailable"
		}
	}

	if record.SpecialProject {
		return "unavailable"
	}
	if len(parents) == 0 {
		return "available"
	}
	for _, parent := range parents {
		if held[parent] {
			return "available"
		}
	}
	return "unavailable"
}

func (idx *Index) techCardLabel(record *technologyRecord, upperTag string) (string, string) {
	keys := []string{record.Name + "_short", record.Name}
	if len(record.Enables) == 1 {
		model := record.Enables[0]
		keys = append([]string{upperTag + "_" + model + "_short", upperTag + "_" + model, model + "_short", model}, keys...)
	}
	for _, key := range keys {
		if label, ok := idx.Localise(key); ok {
			return label, key
		}
	}
	return record.Name, record.Name
}

func (idx *Index) designTeamCategories(tag, date string) map[string]bool {
	out := map[string]bool{}
	tag = strings.ToUpper(strings.TrimSpace(tag))
	state := idx.decisionStartOf(tag, date)
	state.from = tag
	for _, name := range idx.industrialOrgNames() {
		org := idx.resolveIndustrialOrg(name)
		if org == nil || !state.holds(org.Allowed) || !state.holds(org.Visible) {
			continue
		}
		for _, category := range org.ResearchCategories {
			out[strings.ToLower(category)] = true
		}
	}
	return out
}
