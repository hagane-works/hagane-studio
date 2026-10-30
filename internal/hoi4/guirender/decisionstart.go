package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type decisionStart struct {
	tag     string
	date    string
	party   string
	power   float64
	focuses map[string]bool
	flags   map[string]bool
	ideas   map[string]bool

	popularity map[string]float64

	missions map[string]bool

	from string

	world *decisionWorld

	depth int
}

const scriptedTriggerDepth = 8

type decisionWorld struct {
	idx      *Index
	date     string
	starts   map[string]*decisionStart
	owners   map[string]bool
	wars     map[string]map[string]bool
	subjects map[string]bool

	techs map[string]map[string]bool
}

func (idx *Index) decisionStartOf(tag, date string) decisionStart {
	world := &decisionWorld{idx: idx, date: date, starts: map[string]*decisionStart{}}
	return *world.start(tag)
}

func (world *decisionWorld) researched(tag string) map[string]bool {
	tag = strings.ToUpper(tag)
	if world.techs == nil {
		world.techs = map[string]map[string]bool{}
	}
	if held, ok := world.techs[tag]; ok {
		return held
	}
	held := map[string]bool{}
	for _, name := range world.idx.ResearchedBy(tag, world.date) {
		held[strings.ToLower(name)] = true
	}
	world.techs[tag] = held
	return held
}

func (world *decisionWorld) start(tag string) *decisionStart {
	tag = strings.ToUpper(tag)
	if state, ok := world.starts[tag]; ok {
		return state
	}
	idx, date := world.idx, world.date
	state := &decisionStart{
		tag: tag, date: scenarioDate(date), party: idx.RulingPartyOf(tag),
		focuses: map[string]bool{}, flags: map[string]bool{}, ideas: map[string]bool{},
		missions: map[string]bool{}, world: world,
	}
	world.starts[tag] = state
	if start, ok := idx.countryStartAt(tag, date); ok {
		state.power = start.PoliticalPower
	}
	if politics := idx.countryPoliticsFor(tag); politics != nil {
		for _, call := range effectsRunByOwned(politics.Effects, date, world.idx.ownedDLC) {
			switch call.Command {
			case "complete_national_focus":
				state.focuses[call.Token] = true
			case "set_country_flag":
				state.flags[call.Token] = true
			case "clr_country_flag":
				delete(state.flags, call.Token)
			case "activate_mission":
				state.missions[strings.ToLower(call.Token)] = true
			}
		}
		if len(politics.Popularities) > 0 {
			state.popularity = map[string]float64{}
			for name, share := range politics.Popularities {
				state.popularity[strings.ToLower(name)] = float64(share) / 100
			}
		}
	}
	for _, idea := range idx.StartingIdeas(tag, date) {
		state.ideas[idea] = true
	}
	return state
}

func (world *decisionWorld) owns(tag string) bool {
	if world.owners == nil {
		world.owners = map[string]bool{}
		for _, state := range world.idx.effectiveStates() {
			if owner := world.idx.ownerOfState(state, world.date); owner != "" {
				world.owners[strings.ToUpper(owner)] = true
			}
		}
	}
	return world.owners[strings.ToUpper(tag)]
}

func (world *decisionWorld) relations() {
	if world.wars != nil {
		return
	}
	world.wars, world.subjects = map[string]map[string]bool{}, map[string]bool{}
	atWar := func(a, b string) {
		if world.wars[a] == nil {
			world.wars[a] = map[string]bool{}
		}
		world.wars[a][b] = true
	}
	for _, tag := range world.idx.ListCountryTags() {
		politics := world.idx.countryPoliticsFor(tag)
		if politics == nil {
			continue
		}
		for _, call := range effectsRunByOwned(politics.Effects, world.date, world.idx.ownedDLC) {
			target := strings.ToUpper(call.Token)
			switch call.Command {
			case "declare_war_on":
				atWar(tag, target)
				atWar(target, tag)
			case "set_autonomy", "puppet":
				world.subjects[target] = true
			}
		}
	}
}

func (state decisionStart) holds(block *paradox.Block) bool {
	holds, decided := state.verdict(block)
	return decided && holds
}

func (state decisionStart) verdict(block *paradox.Block) (holds, decided bool) {
	if block == nil {
		return true, true
	}
	return state.eval(block.Entries, false)
}

func (state decisionStart) eval(entries []*paradox.Entry, anyOf bool) (holds, decidable bool) {
	unknown := false

	var chainTaken, chainUnknown bool
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		var value, ok bool
		switch {
		case entry.Block != nil && key == "if":
			value, ok, chainTaken, chainUnknown = state.branch(entry.Block.Entries, false, false)
		case entry.Block != nil && key == "else_if":
			value, ok, chainTaken, chainUnknown = state.branch(entry.Block.Entries, chainTaken, chainUnknown)
		case entry.Block != nil && key == "else":
			switch {
			case chainUnknown:
				value, ok = false, false
			case chainTaken:
				value, ok = true, true
			default:
				value, ok = state.eval(entry.Block.Entries, false)
			}
			chainTaken, chainUnknown = false, false
		case key == "limit":
			continue
		default:
			value, ok = state.trigger(entry)
		}
		switch {
		case !ok:
			unknown = true
		case anyOf && value:
			return true, true
		case !anyOf && !value:
			return false, true
		}
	}
	if unknown {
		return false, false
	}
	return !anyOf, true
}

func (state decisionStart) branch(entries []*paradox.Entry, taken, unknown bool) (value, ok, nowTaken, nowUnknown bool) {
	if taken {
		return true, true, true, unknown
	}
	if unknown {
		return false, false, false, true
	}
	var limit *paradox.Block
	for _, entry := range entries {
		if entry != nil && entry.Key != nil && entry.Block != nil && strings.EqualFold(*entry.Key, "limit") {
			limit = entry.Block
			break
		}
	}
	applies, decided := state.verdict(limit)
	switch {
	case !decided:
		return false, false, false, true
	case !applies:
		return true, true, false, false
	}
	value, ok = state.eval(entries, false)
	return value, ok, true, false
}

func (state decisionStart) scoped(key string, entries []*paradox.Entry) (value, ok, handled bool) {
	switch {
	case strings.EqualFold(key, "ROOT") || strings.EqualFold(key, "THIS"):
		value, ok = state.eval(entries, false)
		return value, ok, true
	case strings.EqualFold(key, "FROM") && state.from != "" && state.world != nil:
		value, ok = state.world.start(state.from).eval(entries, false)
		return value, ok, true
	case strings.EqualFold(key, "OVERLORD") && state.world != nil:

		overlord := state.world.idx.overlordOf(state.tag)
		if overlord == "" {
			return false, true, true
		}
		value, ok = state.world.start(overlord).eval(entries, false)
		return value, ok, true
	case isTagLike(key) && state.world != nil:
		value, ok = state.world.start(key).eval(entries, false)
		return value, ok, true
	}
	if person := state.character(key); person != nil {
		value, ok = state.characterTriggers(person, entries)
		return value, ok, true
	}
	return false, false, false
}

func (state decisionStart) character(token string) *Character {
	if state.world == nil {
		return nil
	}
	for layer := state.world.idx; layer != nil; layer = layer.fallback {
		if person, ok := layer.charactersByToken[token]; ok {
			return person
		}
	}
	return nil
}

func (state decisionStart) characterTriggers(person *Character, entries []*paradox.Entry) (holds, decided bool) {
	hired := state.ideas[person.Token] || (person.IdeaToken != "" && state.ideas[person.IdeaToken])
	result := true
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		if !strings.EqualFold(*entry.Key, "is_hired_as_advisor") || entry.Block != nil {
			return false, false
		}
		want := strings.EqualFold(strings.Trim(toString(scalarValue(entry.Scalar)), "\" "), "yes")
		result = result && hired == want
	}
	return result, true
}

func compareAt(op string, have, want float64) (bool, bool) {
	switch op {
	case ">":
		return have > want, true
	case "<":
		return have < want, true
	case ">=":
		return have >= want, true
	case "<=":
		return have <= want, true
	case "=", "":
		return have == want, true
	}
	return false, false
}

func (state decisionStart) trigger(entry *paradox.Entry) (value, ok bool) {
	key := strings.ToLower(*entry.Key)
	if entry.Block != nil {
		switch key {
		case "not", "nor":
			inner, decided := state.eval(entry.Block.Entries, true)
			return !inner, decided
		case "or":
			return state.eval(entry.Block.Entries, true)
		case "and", "hidden_trigger":

			return state.eval(entry.Block.Entries, false)
		case "has_dynamic_modifier":
			if len(entry.Block.Entries) != 1 {
				return false, false
			}
			modifier := strings.Trim(ciScalar(entry.Block.Entries, "modifier"), "\"")
			return state.ideas[modifier], modifier != ""
		case "has_country_flag":
			if len(entry.Block.Entries) != 1 {
				return false, false
			}
			flag := strings.Trim(ciScalar(entry.Block.Entries, "flag"), "\"")
			return state.flags[flag], flag != ""
		}
		if value, ok, handled := state.scoped(*entry.Key, entry.Block.Entries); handled {
			return value, ok
		}
		return false, false
	}
	token := strings.Trim(toString(scalarValue(entry.Scalar)), "\" ")
	op := "="
	if entry.Op != nil {
		op = *entry.Op
	}

	if state.popularity != nil && state.isIdeology(key) {
		want, isNumber := toFloat(scalarValue(entry.Scalar))
		if !isNumber {
			return false, false
		}
		return compareAt(op, state.popularity[key], want)
	}
	if key == "date" {
		have, want := parsedDate(state.date), parsedDate(token)
		cmp := 0
		for i := 0; i < 3 && cmp == 0; i++ {
			switch {
			case have[i] < want[i]:
				cmp = -1
			case have[i] > want[i]:
				cmp = 1
			}
		}
		return compareAt(op, float64(cmp), 0)
	}
	if op != "=" {
		return false, false
	}
	yes := strings.EqualFold(token, "yes")
	switch key {
	case "tag", "original_tag":
		other := strings.ToUpper(token)
		return other == state.tag, len(other) == 3 && !strings.ContainsAny(other, ":.")
	case "has_dlc":

		if state.world == nil || state.world.idx == nil {
			return true, true
		}
		return state.world.idx.OwnsDLC(token), true
	case "always":
		return yes, true
	case "is_ai":

		return !yes, true
	case "is_debug":
		return !yes, true
	case "is_major":
		major, known := state.world.idx.bookmarkMajor(state.tag, state.date)
		return major == yes, known
	case "has_government":
		switch strings.ToUpper(token) {
		case "ROOT", "THIS", "FROM", "PREV", "OWNER", "CONTROLLER":
			return false, false
		}
		return state.party != "" && strings.EqualFold(token, state.party), true
	case "has_completed_focus":
		return state.focuses[token], token != ""
	case "has_tech":
		if token == "" || state.world == nil {
			return false, false
		}
		return state.world.researched(state.tag)[strings.ToLower(token)], true
	case "has_doctrine":

		if token == "" || state.world == nil {
			return false, false
		}
		politics := state.world.idx.countryPoliticsFor(state.tag)
		if politics == nil {
			return false, true
		}
		for _, list := range [][]datedIdeas{politics.GrandDoctrines, politics.SubDoctrines} {
			for _, doctrine := range tokensByOwned(list, state.date, state.world.idx.ownedDLC) {
				if strings.EqualFold(doctrine, token) {
					return true, true
				}
			}
		}
		return false, true
	case "has_country_flag":
		return state.flags[token], token != ""
	case "has_idea":
		return state.ideas[token], token != ""
	}
	if state.world == nil {
		return false, false
	}
	switch key {
	case "exists":
		return state.world.owns(state.tag) == yes, true
	case "country_exists":
		other := strings.ToUpper(token)
		return state.world.owns(other), isTagLike(other)
	case "is_subject":
		state.world.relations()
		return state.world.subjects[state.tag] == yes, true
	case "has_war":
		state.world.relations()
		return (len(state.world.wars[state.tag]) > 0) == yes, true
	case "has_war_with":
		other := strings.ToUpper(token)
		if !isTagLike(other) {
			return false, false
		}
		state.world.relations()
		return state.world.wars[state.tag][other], true
	}

	if yes || strings.EqualFold(token, "no") {
		if body := state.world.idx.scriptedTrigger(*entry.Key); body != nil && state.depth < scriptedTriggerDepth {
			inner := state
			inner.depth++
			holds, decided := inner.eval(body.Entries, false)
			return holds == yes, decided
		}
	}
	return false, false
}

func (state decisionStart) isIdeology(key string) bool {
	if state.world == nil {
		return false
	}
	for _, group := range state.world.idx.IdeologyGroups() {
		if strings.EqualFold(group.Name, key) {
			return true
		}
	}
	return false
}

func (idx *Index) TakeableDecisions(tag, date string) []map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	state := idx.decisionStartOf(tag, date)
	out := []map[string]any{}
	for _, category := range decisionCategoriesByPriority(idx.DecisionCategories()) {
		if !idx.everyCategoryLayerHolds(state, category.Name) {
			continue
		}
		for _, decision := range idx.DecisionsIn(category.Name) {
			if !idx.everyDecisionLayerHolds(state, category.Name, decision.Name) {
				continue
			}
			name := decision.Name
			if label, ok := idx.LocaliseMarkup(decision.LabelKey); ok {
				name = label
			}
			out = append(out, map[string]any{"id": decision.Name, "name": name})
		}
	}
	return out
}

func (idx *Index) everyCategoryLayerHolds(state decisionStart, name string) bool {
	key := strings.ToLower(name)
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, category := range layer.decisionCategories {
			if strings.ToLower(category.Name) != key {
				continue
			}
			if category.Debug || !state.holds(category.Allowed) || !state.holds(category.Visible) {
				return false
			}
		}
	}
	return true
}

func (idx *Index) everyDecisionLayerHolds(state decisionStart, category, name string) bool {
	key := strings.ToLower(name)
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, decision := range layer.decisions[strings.ToLower(category)] {
			if strings.ToLower(decision.Name) != key {
				continue
			}
			if decision.Debug || decision.Mission || decision.Targeted || decision.CustomCost ||
				!state.holds(decision.Allowed) || !state.holds(decision.Visible) ||
				!state.holds(decision.Available) {
				return false
			}
			if cost, ok := idx.decisionNumber(decision.Cost); ok && cost > state.power {
				return false
			} else if !ok && decision.Cost != "" {
				return false
			}
		}
	}
	return true
}

func (idx *Index) everyDecisionLayerShows(state decisionStart, category, name string) bool {
	key := strings.ToLower(name)
	found := false
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, decision := range layer.decisions[strings.ToLower(category)] {
			if strings.ToLower(decision.Name) != key {
				continue
			}
			found = true
			if decision.Debug {
				return false
			}
			if decision.Mission {
				if state.missions[key] {
					continue
				}
				if decision.Activation == nil || !state.holds(decision.Allowed) || !state.holds(decision.Activation) {
					return false
				}
				continue
			}
			if !state.holds(decision.Allowed) || !state.holds(decision.Visible) {
				return false
			}
			if decision.TargetedOther || (decision.TargetTrigger != nil && len(decision.Targets) == 0) {
				return false
			}
			if len(decision.Targets) > 0 && !state.anyTarget(decision.Targets, decision.TargetTrigger) {
				return false
			}
		}
	}
	return found
}

func (state decisionStart) anyTarget(targets []string, trigger *paradox.Block) bool {
	for _, target := range targets {
		target = strings.ToUpper(strings.TrimSpace(target))
		if !isTagLike(target) || state.world == nil || !state.world.owns(target) {
			continue
		}
		withFrom := state
		withFrom.from = target
		if withFrom.holds(trigger) {
			return true
		}
	}
	return false
}

func decisionCategoriesByPriority(categories []DecisionCategory) []DecisionCategory {
	out := append([]DecisionCategory(nil), categories...)
	sort.SliceStable(out, func(a, b int) bool { return out[a].Priority > out[b].Priority })
	return out
}

func (idx *Index) bookmarkMajor(tag, date string) (major, known bool) {
	marks := idx.bookmarkList()
	if len(marks) == 0 {
		return false, false
	}
	pick := marks[0]
	at := parsedDate(scenarioDate(date))
	for _, mark := range marks {
		day := parsedDate(mark.Date)
		if day[0] < at[0] || day[0] == at[0] && (day[1] < at[1] || day[1] == at[1] && day[2] <= at[2]) {
			pick = mark
		}
	}
	for _, country := range pick.Countries {
		if strings.EqualFold(country.Tag, tag) && !country.Minor {
			return true, true
		}
	}
	return false, true
}
