package guirender

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type stateRecord struct {
	ID       int
	Manpower int

	Owners    []datedIdeas
	Levels    []datedLevels
	Resources map[string]int

	Name     string
	Category string
	Cores    []datedIdeas

	Claims        []datedIdeas
	VictoryPoints map[int]int
	Provinces     []int

	ProvinceLevels []datedProvinceLevels
}

type datedProvinceLevels struct {
	Date string
	ByID map[int]map[string]int
}

type datedLevels struct {
	Date      string
	Buildings map[string]int
}

func parseStateRecord(entries []*paradox.Entry) *stateRecord {
	record := &stateRecord{Resources: map[string]int{}}
	if id, ok := toFloat(ciScalarValue(entries, "id")); ok {
		record.ID = int(id)
	}
	if manpower, ok := toFloat(ciScalarValue(entries, "manpower")); ok {
		record.Manpower = int(manpower)
	}
	record.Owners = collectStateOwners(entries, "")
	record.Name = strings.Trim(ciScalar(entries, "name"), "\"")
	record.Category = strings.Trim(ciScalar(entries, "state_category"), "\"")
	record.VictoryPoints = map[int]int{}

	record.Cores = collectCalls(entries, "add_core_of", "")
	record.Claims = collectCalls(entries, "add_claim_by", "")
	var walk func(items []*paradox.Entry, date string)
	walk = func(items []*paradox.Entry, date string) {
		for _, entry := range items {
			if entry == nil || entry.Key == nil || entry.Block == nil {
				continue
			}
			key := strings.ToLower(*entry.Key)
			switch {
			case key == "victory_points":

				points := blockTokens(entry.Block.Entries)
				for at := 0; at+1 < len(points); at += 2 {
					province, err1 := strconv.Atoi(points[at])
					value, err2 := strconv.Atoi(points[at+1])
					if err1 == nil && err2 == nil {
						record.VictoryPoints[province] = value
					}
				}
			case key == "provinces":
				for _, token := range blockTokens(entry.Block.Entries) {
					if id, err := strconv.Atoi(token); err == nil {
						record.Provinces = append(record.Provinces, id)
					}
				}
			case key == "buildings":
				levels := map[string]int{}
				addLevels(levels, entry.Block.Entries)
				record.Levels = append(record.Levels, datedLevels{Date: date, Buildings: levels})
				byID := map[int]map[string]int{}
				for _, field := range entry.Block.Entries {
					if field == nil || field.Key == nil || field.Block == nil {
						continue
					}
					province, err := strconv.Atoi(strings.TrimSpace(*field.Key))
					if err != nil {
						continue
					}
					into := map[string]int{}
					addLevels(into, field.Block.Entries)
					byID[province] = into
				}
				if len(byID) > 0 {
					record.ProvinceLevels = append(record.ProvinceLevels,
						datedProvinceLevels{Date: date, ByID: byID})
				}
			case key == "resources":
				addLevels(record.Resources, entry.Block.Entries)
			case isGameDate(key):
				walk(entry.Block.Entries, key)
			default:
				walk(entry.Block.Entries, date)
			}
		}
	}
	walk(entries, "")
	return record
}

func (record *stateRecord) provinceBuildingsAt(date string) map[int]map[string]int {
	out := map[int]map[string]int{}
	for _, block := range record.ProvinceLevels {
		if !dateReached(block.Date, date) {
			continue
		}
		for province, levels := range block.ByID {
			into := out[province]
			if into == nil {
				into = map[string]int{}
				out[province] = into
			}
			for name, level := range levels {
				into[name] = level
			}
		}
	}
	return out
}

func (record *stateRecord) buildingsAt(date string) map[string]int {
	out := map[string]int{}
	for _, block := range record.Levels {
		if !dateReached(block.Date, date) {
			continue
		}
		for name, level := range block.Buildings {
			out[name] = level
		}
	}
	return out
}

func addLevels(into map[string]int, entries []*paradox.Entry) {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}

		if entry.Block != nil {
			if _, err := strconv.Atoi(strings.TrimSpace(*entry.Key)); err == nil {
				addLevels(into, entry.Block.Entries)
			} else if level, ok := toFloat(ciScalarValue(entry.Block.Entries, "level")); ok {

				into[strings.ToLower(*entry.Key)] += int(level)
			}
			continue
		}
		if value, ok := toFloat(scalarValue(entry.Scalar)); ok {
			into[strings.ToLower(*entry.Key)] += int(value)
		}
	}
}

func (idx *Index) effectiveStates() []*stateRecord {
	byID := map[int]*stateRecord{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for id, state := range layer.states {
			if _, taken := byID[id]; !taken {
				byID[id] = state
			}
		}
	}
	ids := make([]int, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]*stateRecord, 0, len(ids))
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out
}

func (idx *Index) ownerOfState(record *stateRecord, date string) string {
	if date == "" {
		date = "1936.1.1"
	}
	owner, at := "", ""
	for _, block := range record.Owners {
		if len(block.Tokens) == 0 || !dateReached(block.Date, date) || !block.DLC.holds(idx.ownedDLC) {
			continue
		}
		if dateReached(at, block.Date) {
			owner, at = block.Tokens[len(block.Tokens)-1], block.Date
		}
	}
	return strings.ToUpper(owner)
}

func collectStateOwners(entries []*paradox.Entry, date string) []datedIdeas {
	return collectStateOwnersGuarded(entries, date, dlcGuard{})
}

func collectStateOwnersGuarded(entries []*paradox.Entry, date string, guard dlcGuard) []datedIdeas {
	out := []datedIdeas{}
	branch := branchState{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		if entry.Block != nil && (key == "if" || key == "else_if") {
			if inner, ok := branch.enterIf(entry.Block.Entries, guard); ok {
				out = append(out, collectStateOwnersGuarded(entry.Block.Entries, date, inner)...)
			}
			continue
		}
		if entry.Block != nil && key == "else" {
			if inner, ok := branch.enterElse(); ok {
				out = append(out, collectStateOwnersGuarded(entry.Block.Entries, date, inner)...)
			}
			continue
		}
		if entry.Block != nil {
			if isGameDate(key) {
				out = append(out, collectStateOwnersGuarded(entry.Block.Entries, key, guard)...)
				continue
			}
			if tag := strings.TrimSpace(*entry.Key); countryTagRe.MatchString(tag) && transfersThisState(entry.Block.Entries) {
				out = append(out, datedIdeas{Date: date, Tokens: []string{tag}, DLC: guard})
				continue
			}
			out = append(out, collectStateOwnersGuarded(entry.Block.Entries, date, guard)...)
			continue
		}
		if key == "owner" || key == "transfer_state_to" {
			if token := toString(scalarValue(entry.Scalar)); token != "" {
				out = append(out, datedIdeas{Date: date, Tokens: []string{token}, DLC: guard})
			}
		}
	}
	return out
}

var countryTagRe = regexp.MustCompile(`^[A-Z][A-Z0-9]{2}$`)

func transfersThisState(entries []*paradox.Entry) bool {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block != nil || !strings.EqualFold(*entry.Key, "transfer_state") {
			continue
		}
		switch strings.ToUpper(toString(scalarValue(entry.Scalar))) {
		case "PREV", "ROOT", "THIS":
			return true
		}
	}
	return false
}

func (idx *Index) conscriptionFactor(tag, date string) float64 {
	law := ""
	for _, token := range idx.StartingIdeas(tag, date) {
		if idx.ideaCategoryOf(token) == "mobilization_laws" {
			law = token
		}
	}
	if law == "" {
		law = idx.DefaultIdeaOf("mobilization_laws")
	}
	if law == "" {
		return 0
	}
	for layer := idx; layer != nil; layer = layer.fallback {
		if value, ok := layer.ideaConscription[law]; ok {
			return value
		}
	}
	return 0
}

func (idx *Index) CountryStatsPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	out := map[string]any{"tag": tag}
	start, _ := idx.countryStartAt(tag, date)
	sums := idx.startingModifiers(tag, date)

	out["political_power"] = start.PoliticalPower + idx.dailyPoliticalPower(sums)

	out["command_power"] = start.CommandPower
	out["convoys"] = start.Convoys
	out["research_slots"] = start.ResearchSlots

	out["stability"], out["war_support"] = idx.startingStability(tag, date, sums)
	out["army_experience"] = idx.experience(start.ArmyXP, "NMilitary.MAX_ARMY_EXPERIENCE")
	out["navy_experience"] = idx.experience(start.NavyXP, "NMilitary.MAX_NAVY_EXPERIENCE")
	out["air_experience"] = idx.experience(start.AirXP, "NMilitary.MAX_AIR_EXPERIENCE")

	out["nukes"] = start.Nukes

	population, civilian, military, naval := 0, 0, 0, 0
	resources := map[string]int{}
	built := map[string]int{}
	for _, state := range idx.effectiveStates() {
		if idx.ownerOfState(state, date) != tag {
			continue
		}
		population += state.Manpower
		buildings := state.buildingsAt(date)
		civilian += buildings["industrial_complex"]
		military += buildings["arms_factory"]
		naval += buildings["dockyard"]
		for name, level := range buildings {
			built[name] += level
		}
		for name, amount := range state.Resources {
			resources[name] += amount
		}
	}

	out["factories_total"] = civilian + military + naval
	out["fuel"] = idx.fuelEstimate(start.FuelRatio, resources["oil"], built, sums["fuel_gain_factor"])

	out["supply_ratio"] = 1.0
	out["world_tension"] = idx.WorldTension(date)
	takeable := idx.TakeableDecisions(tag, date)
	out["decisions_takeable"] = takeable
	out["decisions_takeable_count"] = len(takeable)
	out["faction"] = idx.factionBadge(tag, date)
	if politics := idx.politicsLines(tag, date); politics != nil {
		out["politics"] = politics
	}
	if pie := idx.popularityPie(tag); len(pie) > 0 {
		out["popularities"] = pie
	}

	if flag := idx.flagPayload(tag, idx.RulingPartyOf(tag), ""); flag != nil {
		out["flag"] = flag
	}
	if name := idx.CountryName(tag, idx.RulingPartyOf(tag)); name != "" {
		out["name"] = name
	}
	factor := idx.conscriptionFactor(tag, date)
	out["population"] = population

	out["manpower"] = max(0, int(float64(population)*factor)-idx.fieldedManpower(tag, date))
	out["conscription"] = factor
	out["civilian_factories"] = civilian
	out["military_factories"] = military
	out["dockyards"] = naval
	out["resources"] = resources

	balance := idx.resourceBalance(tag, date)
	out["resource_surplus"] = balance["surplus"]
	out["resource_balance"] = balance
	return out
}

func (idx *Index) experience(value float64, capDefine string) int {
	if value < 0 {
		return 0
	}
	if limit := idx.defineNumber(capDefine); limit > 0 && value > limit {
		value = limit
	}
	return int(value)
}

func (idx *Index) buildingRecordFor(name string) *buildingRecord {
	name = strings.ToLower(name)
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.buildings[name]; ok {
			return record
		}
	}
	return nil
}

func (idx *Index) fuelEstimate(ratio float64, oil int, built map[string]int, gainFactor float64) map[string]any {
	capacity := idx.defineNumber("NCountry.BASE_FUEL_CAPACITY")
	gain := idx.defineNumber("NCountry.BASE_FUEL_GAIN") +
		idx.defineNumber("NCountry.BASE_FUEL_GAIN_PER_OIL")*float64(oil)
	names := make([]string, 0, len(built))
	for name := range built {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		record := idx.buildingRecordFor(name)
		if record == nil || built[name] <= 0 {
			continue
		}
		level := float64(built[name])
		capacity += level * record.CountryModifiers["max_fuel_building"] * 1000
		gain += level * record.CountryModifiers["fuel_gain_from_states"]
	}

	gain *= 1 + gainFactor
	out := map[string]any{
		"ratio": ratio, "capacity": math.Round(capacity),
		"gaining": ratio < 1 && gain > 0, "days": nil,
	}
	if ratio < 1 && gain > 0 && capacity > 0 {
		out["days"] = int(math.Ceil(capacity * (1 - ratio) / (24 * gain)))
	}
	return out
}

const genericFactionMiniature = "GFX_faction_logo_generic_miniature"

func (idx *Index) factionBadge(tag, date string) map[string]any {
	name, key, icon, leader := idx.factionNamed(tag, date)
	if name == "" {
		return nil
	}

	badge := map[string]any{"name": name, "name_key": key, "leader": leader, "icon": "", "initiative": 0}
	if icon != "" {
		if _, found := idx.spriteRecordFor(icon + "_miniature"); found {
			badge["icon"] = icon + "_miniature"
		}
	}

	if badge["icon"] == "" {
		if _, found := idx.spriteRecordFor(genericFactionMiniature); found {
			badge["icon"] = genericFactionMiniature
		}
	}
	return badge
}

func (idx *Index) politicsLines(tag, date string) map[string]any {
	politics := idx.countryPoliticsFor(tag)
	if politics == nil || politics.RulingParty == "" {
		return nil
	}
	party := strings.ToLower(politics.RulingParty)
	name, ok := idx.locValue(party)
	if !ok {
		name = party
	}

	out := map[string]any{"ruling_party": party, "ideology": cleanMarkup(name)}
	if politics.HasElections && !politics.ElectionsAllowed {
		if text, ok := idx.locValue("POLITICS_NOELECTIONS"); ok {
			out["elections"] = cleanMarkup(text)
			out["elections_key"] = "POLITICS_NOELECTIONS"
		}
		return out
	}
	next := nextElection(politics.LastElection, politics.ElectionFrequency, scenarioDate(date))
	if next == "" {
		return out
	}
	out["next_election"] = next
	if text, ok := idx.locValue("POLITICS_NEXT_ELECTION"); ok {
		out["elections"] = cleanMarkup(strings.ReplaceAll(text, "$DATE$", idx.gameDateText(next)))
		out["elections_key"] = "POLITICS_NEXT_ELECTION"
	}
	return out
}

func nextElection(last string, months int, date string) string {
	year, month, day, ok := splitGameDate(last)
	if !ok || months <= 0 {
		return ""
	}
	for step := 0; step < 1000; step++ {
		month += months
		for month > 12 {
			month -= 12
			year++
		}
		next := fmt.Sprintf("%d.%d.%d", year, month, day)
		if !dateReached(next, date) {
			return next
		}
	}
	return ""
}

func splitGameDate(raw string) (year, month, day int, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(raw), `"`), ".")
	if len(parts) < 3 {
		return 0, 0, 0, false
	}
	var err error
	if year, err = strconv.Atoi(parts[0]); err != nil {
		return 0, 0, 0, false
	}
	if month, err = strconv.Atoi(parts[1]); err != nil || month < 1 || month > 12 {
		return 0, 0, 0, false
	}
	if day, err = strconv.Atoi(parts[2]); err != nil {
		return 0, 0, 0, false
	}
	return year, month, day, true
}

func (idx *Index) gameDateText(raw string) string {
	year, month, day, ok := splitGameDate(raw)
	if !ok {
		return raw
	}
	short := [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}[month-1]
	if value, found := idx.locValue(short); found {
		short = cleanMarkup(value)
	}
	return idx.dateInLanguage(year, month, day, short)
}

func (idx *Index) popularityPie(tag string) []map[string]any {
	politics := idx.countryPoliticsFor(tag)
	if politics == nil || len(politics.Popularities) == 0 {
		return nil
	}
	out := []map[string]any{}
	for _, group := range idx.IdeologyGroups() {
		value := politics.Popularities[group.Name]
		if value <= 0 {
			continue
		}

		labelKey := tag + "_" + group.Name + "_party"
		label, ok := idx.Localise(labelKey)
		if !ok {
			label, _ = idx.Localise(group.Name)
		}
		out = append(out, map[string]any{
			"party":     group.Name,
			"label":     label,
			"label_key": labelKey,
			"value":     value,
			"color":     []int{group.Color[0], group.Color[1], group.Color[2]},
		})
	}
	return out
}
