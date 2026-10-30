package guirender

import (
	"math"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type specialProjectRecord struct {
	Name           string
	Specialization string

	Allowed []*paradox.Entry

	Available []*paradox.Entry

	Parents []string

	Icon string

	BreakthroughCost map[string]float64
	PrototypeTime    string
	Complexity       string

	Output *paradox.Block
	Order  int
	File   string
	Line   int
}

type specializationRecord struct {
	Name  string
	Color []float64
	Order int
}

func (idx *Index) indexSpecialProjectFile(file string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	if idx.specialProjects == nil {
		idx.specialProjects = map[string]*specialProjectRecord{}
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || strings.HasPrefix(*entry.Key, "@") {
			continue
		}
		specialization := strings.ToLower(strings.TrimSpace(ciScalar(entry.Block.Entries, "specialization")))
		if specialization == "" {
			continue
		}
		idx.specialProjectSeq++
		record := &specialProjectRecord{
			Name: *entry.Key, Specialization: specialization, Order: idx.specialProjectSeq, File: file, Line: entry.Line,
			Icon:          strings.Trim(ciScalar(entry.Block.Entries, "icon"), "\""),
			PrototypeTime: strings.Trim(ciScalar(entry.Block.Entries, "prototype_time"), "\""),
			Complexity:    strings.Trim(ciScalar(entry.Block.Entries, "complexity"), "\""),
		}
		if allowed := childBlock(entry.Block.Entries, "allowed"); allowed != nil {
			record.Allowed = append([]*paradox.Entry{}, allowed.Entries...)
		}
		if available := childBlock(entry.Block.Entries, "available"); available != nil {
			record.Available = append([]*paradox.Entry{}, available.Entries...)
		}
		if parents := childBlock(entry.Block.Entries, "special_project_parent"); parents != nil {
			record.Parents = blockTokens(parents.Entries)
		}
		if cost := childBlock(entry.Block.Entries, "breakthrough_cost"); cost != nil {
			record.BreakthroughCost = numberBlock(cost)
		}
		record.Output = childBlock(entry.Block.Entries, "project_output")
		idx.specialProjects[strings.ToLower(record.Name)] = record
	}
}

func (idx *Index) indexSpecializationFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	if idx.specializations == nil {
		idx.specializations = map[string]*specializationRecord{}
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		idx.specializationSeq++
		record := &specializationRecord{Name: strings.ToLower(*entry.Key), Order: idx.specializationSeq}
		if color := childBlock(entry.Block.Entries, "color"); color != nil {
			for _, token := range blockTokens(color.Entries) {
				if value, ok := toFloat(token); ok {
					record.Color = append(record.Color, value)
				}
			}
		}
		idx.specializations[record.Name] = record
	}
}

func (idx *Index) specializationList() []*specializationRecord {
	var layers []*Index
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
	}
	byName := map[string]*specializationRecord{}
	rank := map[string]int{}
	next := 0
	for at := len(layers) - 1; at >= 0; at-- {
		records := make([]*specializationRecord, 0, len(layers[at].specializations))
		for _, record := range layers[at].specializations {
			records = append(records, record)
		}
		sort.Slice(records, func(a, b int) bool { return records[a].Order < records[b].Order })
		for _, record := range records {
			if _, seen := rank[record.Name]; !seen {
				next++
				rank[record.Name] = next
			}
			byName[record.Name] = record
		}
	}
	out := make([]*specializationRecord, 0, len(byName))
	for _, record := range byName {
		out = append(out, record)
	}
	sort.Slice(out, func(a, b int) bool { return rank[out[a].Name] < rank[out[b].Name] })
	return out
}

func (idx *Index) specialProjectList() []*specialProjectRecord {
	byName := map[string]*specialProjectRecord{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for key, record := range layer.specialProjects {
			if _, seen := byName[key]; !seen {
				byName[key] = record
			}
		}
	}
	out := make([]*specialProjectRecord, 0, len(byName))
	for _, record := range byName {
		out = append(out, record)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Order != out[b].Order {
			return out[a].Order < out[b].Order
		}
		return out[a].Name < out[b].Name
	})
	return out
}

func (idx *Index) SpecialProjectSource(name string) (string, []byte, int, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.specialProjects[name]; ok && record.File != "" {
			return record.File, layer.files[record.File], record.Line, true
		}
	}
	return "", nil, 0, false
}

func (idx *Index) facilityBuildings() map[string]*buildingRecord {
	out := map[string]*buildingRecord{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, record := range layer.buildings {
			if record.Specialization == "" {
				continue
			}
			if _, seen := out[record.Specialization]; !seen {
				out[record.Specialization] = record
			}
		}
	}
	return out
}

func (idx *Index) SpecialProjectsPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	state := idx.decisionStartOf(tag, date)

	completed := map[string]bool{}
	progress := map[string]float64{}
	if politics := idx.countryPoliticsFor(tag); politics != nil {
		for _, call := range effectsRunByOwned(politics.Effects, date, idx.ownedDLC) {
			switch call.Command {
			case "complete_special_project":
				completed[strings.ToLower(call.Token)] = true
			case "add_breakthrough_progress":
				progress[call.Token] += call.Value
			}
		}
	}
	counts := map[string]int{}
	buildings := idx.facilityBuildings()

	asFrom := state
	asFrom.from = tag
	projects := []map[string]any{}
	pictures := []string{"GFX_sp_facility_card_bg", "GFX_buildings_strip", "GFX_unavailable_project"}
	for _, project := range idx.specialProjectList() {
		if completed[strings.ToLower(project.Name)] {
			continue
		}
		if project.Allowed != nil {
			if value, decided := state.eval(project.Allowed, false); !decided || !value {
				continue
			}
		}
		counts[project.Specialization]++
		available := true
		for _, parent := range project.Parents {
			if !completed[strings.ToLower(parent)] {
				available = false
			}
		}
		if available && project.Available != nil {
			value, decided := asFrom.eval(project.Available, false)
			available = decided && value
		}
		generic := idx.localiseOr(project.Name, project.Name)
		label, labelKey := generic, project.Name
		if own, ok := idx.Localise(tag + "_" + project.Name); ok {
			label, labelKey = own, tag+"_"+project.Name
		}
		icon := project.Icon
		if icon == "" {
			if _, ok := idx.spriteRecordFor("GFX_" + project.Name); ok {
				icon = "GFX_" + project.Name
			}
		}
		if icon != "" {
			pictures = append(pictures, icon)
		}
		costs := []map[string]any{}
		specs := make([]string, 0, len(project.BreakthroughCost))
		for spec := range project.BreakthroughCost {
			specs = append(specs, spec)
		}
		sort.Strings(specs)
		for _, spec := range specs {
			cost := map[string]any{"specialization": spec, "amount": project.BreakthroughCost[spec]}
			if building := buildings[spec]; building != nil {
				cost["icon_frame"] = building.IconFrame
			}
			costs = append(costs, cost)
		}
		item := map[string]any{
			"id": project.Name, "label": label, "label_key": labelKey, "specialization": project.Specialization,
			"specialization_name": idx.localiseOr(project.Specialization, project.Specialization),
			"available":           available, "icon": icon, "breakthrough_cost": costs,
			"prototype_time": project.PrototypeTime, "complexity": project.Complexity, "file": project.File,
			"allowed_script": conditionScript(project.Allowed), "available_script": conditionScript(project.Available),
		}
		if label != generic {
			item["generic"] = generic
		}
		projects = append(projects, item)
	}
	specializations := []map[string]any{}
	for _, record := range idx.specializationList() {
		item := map[string]any{
			"id": record.Name, "label": idx.localiseOr(record.Name+"_short", idx.localiseOr(record.Name, record.Name)),
			"name": idx.localiseOr(record.Name, record.Name), "projects": counts[record.Name], "color": record.Color,
			"points": int(math.Floor(progress[record.Name] + 1e-9)), "progress": progress[record.Name] - math.Floor(progress[record.Name]+1e-9),
		}
		if building := buildings[record.Name]; building != nil {
			item["icon_frame"] = building.IconFrame
			item["building"] = building.Name
		}
		specializations = append(specializations, item)
	}
	facilities := []map[string]any{}
	for _, stateRecord := range idx.effectiveStates() {
		if idx.ownerOfState(stateRecord, date) != tag {
			continue
		}
		levels := stateRecord.buildingsAt(date)
		provinces := stateRecord.provinceBuildingsAt(date)
		ids := make([]int, 0, len(provinces))
		for province := range provinces {
			ids = append(ids, province)
		}
		sort.Ints(ids)
		add := func(name string, level, province int) {
			if level <= 0 {
				return
			}
			var building *buildingRecord
			for _, candidate := range buildings {
				if strings.EqualFold(candidate.Name, name) {
					building = candidate
				}
			}

			if building == nil || building.IsProvince != (province != 0) {
				return
			}
			for i := 0; i < level; i++ {
				facilities = append(facilities, map[string]any{
					"state": stateRecord.ID, "state_name": idx.stateLabel(stateRecord, tag), "province": province,
					"building": building.Name, "label": idx.localiseOr(building.Name, building.Name),
					"specialization": building.Specialization, "icon_frame": building.IconFrame,
				})
			}
		}
		names := make([]string, 0, len(levels))
		for name := range levels {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			add(name, levels[name], 0)
		}
		for _, province := range ids {
			names := make([]string, 0, len(provinces[province]))
			for name := range provinces[province] {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				add(name, provinces[province][name], province)
			}
		}
	}
	return map[string]any{
		"tag": tag, "specializations": specializations, "facilities": facilities, "projects": projects,

		"sprites": idx.SpritesPayload(pictures),
	}
}
