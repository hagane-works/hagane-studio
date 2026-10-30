package guirender

import (
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

const defaultTemplatePriority = 1

type DivisionTemplate struct {
	Name     string
	Priority int

	NamesGroup string

	Counter    int
	HasCounter bool

	Regiments         []RegimentCount
	Support           []RegimentCount
	RegimentalSupport []RegimentCount

	RegimentCells          []RegimentCell
	SupportCells           []RegimentCell
	RegimentalSupportCells []RegimentCell

	IsArmyHQ bool
}

type RegimentCell struct {
	Type string
	X    int
	Y    int
}

func (t DivisionTemplate) AllCells() []RegimentCell {
	out := append([]RegimentCell{}, t.RegimentCells...)
	out = append(out, t.SupportCells...)
	return append(out, t.RegimentalSupportCells...)
}

func (t DivisionTemplate) allCounts() []RegimentCount {
	out := append([]RegimentCount{}, t.Regiments...)
	out = append(out, t.Support...)
	return append(out, t.RegimentalSupport...)
}

type RegimentCount struct {
	Type  string
	Count int
}

type oobRecord struct {
	templates []DivisionTemplate

	Deployed map[string]int

	Fielded map[string]float64

	Divisions []oobDivision

	TaskForces []oobTaskForce

	AirWings []oobAirWing

	ProductionLines []oobProductionLine
}

type oobProductionLine struct {
	Equipment    string
	Creator      string
	Version      string
	Factories    int
	Progress     float64
	Efficiency   float64
	Manufacturer string

	Amount int
}

func collectProductionLines(entries []*paradox.Entry, record *oobRecord) {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		if !strings.EqualFold(*entry.Key, "add_equipment_production") {
			collectProductionLines(entry.Block.Entries, record)
			continue
		}
		line := oobProductionLine{
			Manufacturer: strings.TrimPrefix(strings.Trim(ciScalar(entry.Block.Entries, "industrial_manufacturer"), "\""), "mio:"),
		}
		for _, field := range entry.Block.Entries {
			if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, "equipment") {
				line.Equipment = strings.ToLower(strings.Trim(ciScalar(field.Block.Entries, "type"), "\""))
				line.Creator = strings.ToUpper(strings.Trim(ciScalar(field.Block.Entries, "creator"), "\""))
				line.Version = strings.Trim(ciScalar(field.Block.Entries, "version_name"), "\"")
			}
		}
		if n, ok := toFloat(ciScalarValue(entry.Block.Entries, "requested_factories")); ok {
			line.Factories = int(n)
		}
		if n, ok := toFloat(ciScalarValue(entry.Block.Entries, "progress")); ok {
			line.Progress = n
		}
		if n, ok := toFloat(ciScalarValue(entry.Block.Entries, "efficiency")); ok {
			line.Efficiency = n
		}
		if n, ok := toFloat(ciScalarValue(entry.Block.Entries, "amount")); ok {
			line.Amount = int(n)
		}
		if line.Equipment != "" {
			record.ProductionLines = append(record.ProductionLines, line)
		}
	}
}

type oobAirWing struct {
	State   int
	Type    string
	Amount  float64
	Version string
	Name    string
}

type oobTaskForce struct {
	Fleet    string
	Name     string
	Location int
	Ships    []oobShip
}

type oobShip struct {
	Name       string
	Definition string
	Pride      bool
	Experience float64
}

type oobDivision struct {
	Name       string
	NameOrder  int
	Template   string
	Location   int
	Experience float64

	Equipment float64

	Officer string
}

func (idx *Index) indexOOBFile(name string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	key := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(name, "history/units/"), ".txt"))
	record := &oobRecord{Deployed: map[string]int{}, Fielded: map[string]float64{}}
	countForces(root.Entries, record)
	collectProductionLines(root.Entries, record)
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil ||
			!strings.EqualFold(*entry.Key, "division_template") {
			continue
		}
		template := idx.parseDivisionTemplate(entry.Block)
		if template.Name != "" {
			record.templates = append(record.templates, template)
		}
	}
	if len(record.templates) > 0 || len(record.Deployed) > 0 || len(record.Fielded) > 0 || len(record.Divisions) > 0 || len(record.TaskForces) > 0 || len(record.AirWings) > 0 || len(record.ProductionLines) > 0 {
		idx.oobs[key] = record
	}

	if len(record.templates) > 0 || len(record.TaskForces) > 0 || len(record.AirWings) > 0 {
		if idx.oobSources == nil {
			idx.oobSources = map[string][]byte{}
		}
		idx.oobSources[key] = body
	}
}

func (idx *Index) OOBSource(name string) ([]byte, bool) {
	key := strings.ToLower(strings.Trim(name, "\""))
	for layer := idx; layer != nil; layer = layer.fallback {
		if body, ok := layer.oobSources[key]; ok {
			return body, true
		}
	}
	return nil, false
}

func (idx *Index) TemplateOOB(tag, date, template string) (string, bool) {
	for _, oob := range idx.OOBOf(tag, date) {
		record, ok := idx.oobRecordFor(oob)
		if !ok || len(record.templates) == 0 {
			continue
		}
		if template == "" {
			return strings.Trim(oob, "\""), true
		}
		for _, candidate := range record.templates {
			if strings.EqualFold(candidate.Name, template) {
				return strings.Trim(oob, "\""), true
			}
		}
	}
	return "", false
}

func (idx *Index) parseDivisionTemplate(block *paradox.Block) DivisionTemplate {
	template := DivisionTemplate{
		Name:       strings.Trim(ciScalar(block.Entries, "name"), "\""),
		Priority:   defaultTemplatePriority,
		NamesGroup: strings.Trim(ciScalar(block.Entries, "division_names_group"), "\""),
		IsArmyHQ:   strings.EqualFold(ciScalar(block.Entries, "is_army_hq"), "yes"),
	}
	if template.Name == "" {
		if key := strings.Trim(ciScalar(block.Entries, "localization_key"), "\""); key != "" {
			template.Name = idx.localiseOr(key, key)
		}
	}
	if priority, ok := toFloat(ciScalarValue(block.Entries, "priority")); ok {
		template.Priority = int(priority)
	}
	if counter, ok := strictInt(ciScalarValue(block.Entries, "template_counter")); ok {
		template.Counter, template.HasCounter = counter, true
	}
	for _, field := range block.Entries {
		if field == nil || field.Key == nil || field.Block == nil {
			continue
		}
		switch strings.ToLower(*field.Key) {
		case "regiments":
			template.Regiments = countRegiments(field.Block.Entries)
			template.RegimentCells = regimentCells(field.Block.Entries)
		case "support":
			template.Support = countRegiments(field.Block.Entries)
			template.SupportCells = regimentCells(field.Block.Entries)
		case "regimental_support":
			template.RegimentalSupport = countRegiments(field.Block.Entries)
			template.RegimentalSupportCells = regimentCells(field.Block.Entries)
		}
	}
	return template
}

func countForces(entries []*paradox.Entry, record *oobRecord) {
	fleet := ""
	var walk func(list []*paradox.Entry)
	walk = func(list []*paradox.Entry) {
		for _, entry := range list {
			if entry == nil || entry.Key == nil || entry.Block == nil {
				continue
			}
			key := strings.ToLower(*entry.Key)
			switch {
			case key == "division":
				name := strings.Trim(ciScalar(entry.Block.Entries, "division_template"), "\"")
				if name != "" {
					record.Deployed[name]++
				}
				division := oobDivision{Template: name, Equipment: 1, Name: strings.Trim(ciScalar(entry.Block.Entries, "name"), "\"")}
				if block := childBlock(entry.Block.Entries, "division_name"); block != nil {
					if order, ok := toFloat(ciScalarValue(block.Entries, "name_order")); ok {
						division.NameOrder = int(order)
					}
					if named := strings.Trim(ciScalar(block.Entries, "name"), "\""); named != "" && division.Name == "" {
						division.Name = named
					}
				}
				if location, ok := toFloat(ciScalarValue(entry.Block.Entries, "location")); ok {
					division.Location = int(location)
				}
				if experience, ok := toFloat(ciScalarValue(entry.Block.Entries, "start_experience_factor")); ok {
					division.Experience = experience
				}
				if equipment, ok := toFloat(ciScalarValue(entry.Block.Entries, "start_equipment_factor")); ok {
					division.Equipment = equipment
				}
				if officer := childBlock(entry.Block.Entries, "officer"); officer != nil {
					division.Officer = strings.Trim(ciScalar(officer.Entries, "name"), "\" ")
				}
				record.Divisions = append(record.Divisions, division)
				continue
			case key == "fleet":
				fleet = strings.Trim(ciScalar(entry.Block.Entries, "name"), "\" ")
				walk(entry.Block.Entries)
				continue
			case key == "task_force":
				force := oobTaskForce{Fleet: fleet, Name: strings.Trim(ciScalar(entry.Block.Entries, "name"), "\" ")}
				if location, ok := toFloat(ciScalarValue(entry.Block.Entries, "location")); ok {
					force.Location = int(location)
				}
				for _, item := range entry.Block.Entries {
					if item == nil || item.Key == nil || item.Block == nil || !strings.EqualFold(*item.Key, "ship") {
						continue
					}
					ship := oobShip{
						Name:       strings.Trim(ciScalar(item.Block.Entries, "name"), "\""),
						Definition: strings.ToLower(strings.Trim(ciScalar(item.Block.Entries, "definition"), "\"")),
						Pride:      strings.EqualFold(ciScalar(item.Block.Entries, "pride_of_the_fleet"), "yes"),
					}
					if experience, ok := toFloat(ciScalarValue(item.Block.Entries, "start_experience_factor")); ok {
						ship.Experience = experience
					}
					force.Ships = append(force.Ships, ship)
				}
				record.TaskForces = append(record.TaskForces, force)
				walk(entry.Block.Entries)
				continue
			case key == "equipment":

				for _, model := range entry.Block.Entries {
					if model == nil || model.Key == nil || model.Block == nil {
						continue
					}
					if amount, ok := toFloat(ciScalarValue(model.Block.Entries, "amount")); ok {
						record.Fielded[strings.ToLower(*model.Key)] += amount
					}
				}
				continue
			case key == "air_wings":
				for _, base := range entry.Block.Entries {
					if base == nil || base.Key == nil || base.Block == nil {
						continue
					}
					state, ok := toFloat(*base.Key)
					if !ok {
						continue
					}
					wings := []int{}
					for _, item := range base.Block.Entries {
						if item == nil || item.Key == nil {
							continue
						}
						if item.Block == nil {

							if strings.EqualFold(*item.Key, "name") && len(wings) > 0 {
								last := &record.AirWings[wings[len(wings)-1]]
								if last.Name == "" {
									last.Name = strings.Trim(toString(scalarValue(item.Scalar)), "\"")
								}
							}
							continue
						}
						amount, ok := toFloat(ciScalarValue(item.Block.Entries, "amount"))
						if !ok {
							continue
						}
						model := strings.ToLower(*item.Key)
						record.Fielded[model] += amount
						record.AirWings = append(record.AirWings, oobAirWing{
							State: int(state), Type: model, Amount: amount,
							Version: strings.Trim(ciScalar(item.Block.Entries, "version_name"), "\""),
						})
						wings = append(wings, len(record.AirWings)-1)
					}
				}
				continue
			}
			walk(entry.Block.Entries)
		}
	}
	walk(entries)
}

func countRegiments(entries []*paradox.Entry) []RegimentCount {
	order := []string{}
	counts := map[string]int{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		kind := *entry.Key
		if _, seen := counts[kind]; !seen {
			order = append(order, kind)
		}
		counts[kind]++
	}
	out := make([]RegimentCount, 0, len(order))
	for _, kind := range order {
		out = append(out, RegimentCount{Type: kind, Count: counts[kind]})
	}
	return out
}

func regimentCells(entries []*paradox.Entry) []RegimentCell {
	out := []RegimentCell{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		cell := RegimentCell{Type: *entry.Key}
		if x, ok := toFloat(ciScalarValue(entry.Block.Entries, "x")); ok {
			cell.X = int(x)
		}
		if y, ok := toFloat(ciScalarValue(entry.Block.Entries, "y")); ok {
			cell.Y = int(y)
		}
		out = append(out, cell)
	}
	return out
}

func (idx *Index) OOBOf(tag, date string) []string {
	politics := idx.countryPoliticsFor(tag)
	if politics == nil {
		return nil
	}
	return tokensByOwned(politics.OOBs, date, idx.ownedDLC)
}

func (idx *Index) oobRecordFor(name string) (*oobRecord, bool) {
	key := strings.ToLower(strings.Trim(name, "\""))
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.oobs[key]; ok {
			return record, true
		}
	}
	return nil, false
}

func (idx *Index) DivisionTemplatesOf(tag, date string) []DivisionTemplate {
	out := []DivisionTemplate{}
	seen := map[string]bool{}
	for _, oob := range idx.OOBOf(tag, date) {
		record, ok := idx.oobRecordFor(oob)
		if !ok {
			continue
		}
		for _, template := range record.templates {
			if seen[template.Name] || template.IsArmyHQ {
				continue
			}
			seen[template.Name] = true
			out = append(out, template)
		}
	}
	return out
}

func (idx *Index) DivisionsPayload(tag, date string) map[string]any {

	deployed := map[string]int{}
	for _, oob := range idx.OOBOf(tag, date) {
		if record, ok := idx.oobRecordFor(oob); ok {
			for name, count := range record.Deployed {
				deployed[name] += count
			}
		}
	}
	templates := []map[string]any{}
	for _, template := range idx.DivisionTemplatesOf(tag, date) {
		item := map[string]any{
			"name":      template.Name,
			"icon":      idx.templateIcon(template),
			"priority":  template.Priority,
			"regiments": regimentPayload(idx, template.Regiments),
			"support":   regimentPayload(idx, template.Support),
			"width":     regimentTotal(template.Regiments),
			"deployed":  deployed[template.Name],
		}
		templates = append(templates, item)
	}
	return map[string]any{
		"tag": strings.ToUpper(tag), "oob": idx.OOBOf(tag, date), "templates": templates,
	}
}

func regimentPayload(idx *Index, counts []RegimentCount) []map[string]any {
	out := []map[string]any{}
	for _, count := range counts {
		item := map[string]any{"type": count.Type, "count": count.Count}

		if label, ok := idx.Localise(count.Type); ok {
			item["label"] = label
		}
		out = append(out, item)
	}
	return out
}

func regimentTotal(counts []RegimentCount) int {
	total := 0
	for _, count := range counts {
		total += count.Count
	}
	return total
}

func (idx *Index) ListOOBs() []string {
	seen := map[string]struct{}{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for name := range layer.oobs {
			seen[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (idx *Index) templateIcon(template DivisionTemplate) string {
	if template.HasCounter {
		symbol := "GFX_div_templ_" + strconv.Itoa(template.Counter) + "_large"
		if _, ok := idx.spriteRecordFor(symbol); ok {
			return symbol
		}
	}
	_, lead := idx.leadUnit(template)
	return idx.subUnitIcon(lead)
}

func (idx *Index) leadUnit(template DivisionTemplate) (string, *subUnitRecord) {
	var lead *subUnitRecord
	for _, kind := range templateUnits(template) {
		unit := idx.subUnitFor(kind)
		if unit != nil && (lead == nil || unit.Priority > lead.Priority) {
			lead = unit
		}
	}
	if lead == nil {
		return "", nil
	}
	return lead.Name, lead
}
