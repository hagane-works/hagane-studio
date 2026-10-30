package guirender

import (
	"math"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type effectCall struct {
	DLC     dlcGuard
	Date    string
	Command string
	Value   float64

	Token string
}

type namedThreat struct {
	DLC    dlcGuard
	Date   string
	Threat float64
	Name   string
}

var historyNumberCommands = map[string]bool{
	"add_political_power": true, "add_command_power": true,
	"set_convoys": true, "add_convoys": true,
	"set_research_slots": true, "add_research_slot": true,
	"set_stability": true, "add_stability": true,
	"set_war_support": true, "add_war_support": true,
	"set_fuel_ratio":  true,
	"army_experience": true, "navy_experience": true, "air_experience": true,
	"add_nuclear_bombs": true,

	"complete_national_focus": true, "set_country_flag": true, "clr_country_flag": true,

	"declare_war_on": true, "set_autonomy": true, "puppet": true,

	"activate_mission": true,

	"set_cosmetic_tag": true, "drop_cosmetic_tag": true,

	"complete_special_project": true, "add_breakthrough_progress": true,

	"create_intelligence_agency": true,
}

var tokenCommands = map[string]bool{
	"complete_national_focus": true, "set_country_flag": true, "clr_country_flag": true,
	"puppet": true, "activate_mission": true,
	"set_cosmetic_tag": true, "drop_cosmetic_tag": true,
	"create_intelligence_agency": true,
}

var focusRewardCommands = map[string]bool{
	"add_political_power": true, "add_command_power": true,
	"army_experience": true, "navy_experience": true, "air_experience": true,
	"add_stability": true, "add_war_support": true,
}

func scopeSwitch(key string) bool {
	lower := strings.ToLower(key)
	switch {
	case lower == "limit" || lower == "prev" || lower == "root" || lower == "from":
		return true
	case strings.HasPrefix(lower, "every_") || strings.HasPrefix(lower, "random_") ||
		strings.HasPrefix(lower, "any_") || strings.HasSuffix(lower, "_scope"):
		return true
	case isTagLike(key):
		return true
	}
	for _, r := range key {
		if r < '0' || r > '9' {
			return false
		}
	}
	return key != ""
}

func collectEffects(entries []*paradox.Entry, commands map[string]bool, date string) []effectCall {
	return collectEffectsGuarded(entries, commands, date, dlcGuard{})
}

func collectEffectsGuarded(entries []*paradox.Entry, commands map[string]bool, date string, guard dlcGuard) []effectCall {
	out := []effectCall{}
	add := func(call effectCall) {
		call.DLC = guard
		out = append(out, call)
	}
	branch := branchState{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		if entry.Block != nil && (key == "if" || key == "else_if") {
			if inner, ok := branch.enterIf(entry.Block.Entries, guard); ok {
				out = append(out, collectEffectsGuarded(entry.Block.Entries, commands, date, inner)...)
			}
			continue
		}
		if entry.Block != nil && key == "else" {
			if inner, ok := branch.enterElse(); ok {
				out = append(out, collectEffectsGuarded(entry.Block.Entries, commands, date, inner)...)
			}
			continue
		}
		if entry.Block != nil && (key == "declare_war_on" || key == "set_autonomy") && commands[key] {
			if target := strings.Trim(ciScalar(entry.Block.Entries, "target"), "\""); target != "" {
				add(effectCall{Date: date, Command: key, Token: strings.ToUpper(target)})
			}
			continue
		}
		if entry.Block != nil && key == "complete_special_project" && commands[key] {

			project := strings.TrimPrefix(strings.Trim(ciScalar(entry.Block.Entries, "project"), "\""), "sp:")
			if project != "" {
				add(effectCall{Date: date, Command: key, Token: project})
			}
			continue
		}
		if entry.Block != nil && key == "add_breakthrough_progress" && commands[key] {

			specialization := strings.Trim(ciScalar(entry.Block.Entries, "specialization"), "\"")
			if value, ok := toFloat(ciScalarValue(entry.Block.Entries, "value")); ok && specialization != "" {
				add(effectCall{Date: date, Command: key, Token: strings.ToLower(specialization), Value: value})
			}
			continue
		}
		if entry.Block != nil && key == "set_country_flag" && commands[key] {

			if flag := strings.Trim(ciScalar(entry.Block.Entries, "flag"), "\""); flag != "" {
				add(effectCall{Date: date, Command: key, Token: flag})
			}
			continue
		}
		if entry.Block != nil {
			if isGameDate(key) {
				out = append(out, collectEffectsGuarded(entry.Block.Entries, commands, key, guard)...)
				continue
			}
			if scopeSwitch(*entry.Key) {
				continue
			}
			out = append(out, collectEffectsGuarded(entry.Block.Entries, commands, date, guard)...)
			continue
		}
		if !commands[key] {
			continue
		}
		if tokenCommands[key] {
			if token := strings.Trim(toString(scalarValue(entry.Scalar)), "\""); token != "" {
				add(effectCall{Date: date, Command: key, Token: token})
			}
			continue
		}
		if value, ok := toFloat(scalarValue(entry.Scalar)); ok {
			add(effectCall{Date: date, Command: key, Value: value})
		}
	}
	return out
}

func collectNamedThreats(entries []*paradox.Entry, date string) []namedThreat {
	return collectNamedThreatsGuarded(entries, date, dlcGuard{})
}

func collectNamedThreatsGuarded(entries []*paradox.Entry, date string, guard dlcGuard) []namedThreat {
	out := []namedThreat{}
	branch := branchState{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		switch {
		case key == "if" || key == "else_if":
			if inner, ok := branch.enterIf(entry.Block.Entries, guard); ok {
				out = append(out, collectNamedThreatsGuarded(entry.Block.Entries, date, inner)...)
			}
		case key == "else":
			if inner, ok := branch.enterElse(); ok {
				out = append(out, collectNamedThreatsGuarded(entry.Block.Entries, date, inner)...)
			}
		case key == "limit":
		case key == "add_named_threat":
			if threat, ok := toFloat(ciScalarValue(entry.Block.Entries, "threat")); ok {
				out = append(out, namedThreat{DLC: guard,
					Date: date, Threat: threat,
					Name: strings.Trim(ciScalar(entry.Block.Entries, "name"), "\""),
				})
			}
		case isGameDate(key):
			out = append(out, collectNamedThreatsGuarded(entry.Block.Entries, key, guard)...)
		default:
			out = append(out, collectNamedThreatsGuarded(entry.Block.Entries, date, guard)...)
		}
	}
	return out
}

func scenarioDate(date string) string {
	if strings.TrimSpace(date) == "" {
		return "1936.1.1"
	}
	return date
}

func parsedDate(value string) [3]int {
	out := [3]int{}
	for i, part := range strings.Split(value, ".") {
		if i > 2 {
			break
		}
		out[i] = toInt(part, 0)
	}
	return out
}

func effectsRunBy(calls []effectCall, date string) []effectCall {
	return effectsRunByOwned(calls, date, nil)
}

func effectsRunByOwned(calls []effectCall, date string, owned map[string]bool) []effectCall {
	date = scenarioDate(date)
	out := make([]effectCall, 0, len(calls))
	for _, call := range calls {
		if dateReached(call.Date, date) && call.DLC.holds(owned) {
			out = append(out, call)
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].Date == "" || out[b].Date == "" {
			return out[a].Date == "" && out[b].Date != ""
		}
		da, db := parsedDate(out[a].Date), parsedDate(out[b].Date)
		for i := 0; i < 3; i++ {
			if da[i] != db[i] {
				return da[i] < db[i]
			}
		}
		return false
	})
	return out
}

func foldHistory(calls []effectCall, setCommand, addCommand string, start float64) (value float64, set bool) {
	value = start
	for _, call := range calls {
		switch call.Command {
		case setCommand:
			value, set = call.Value, true
		case addCommand:
			value += call.Value
		}
	}
	return value, set
}

type countryStart struct {
	PoliticalPower float64
	CommandPower   float64
	ArmyXP         float64
	NavyXP         float64
	AirXP          float64
	Convoys        int
	ResearchSlots  int
	Stability      float64
	WarSupport     float64
	FuelRatio      float64
	Nukes          int
}

func (idx *Index) countryStartAt(tag, date string) (countryStart, bool) {
	politics := idx.countryPoliticsFor(tag)
	ok := politics != nil
	if !ok {
		politics = &countryPolitics{}
	}
	calls := idx.withFocusRewards(effectsRunByOwned(politics.Effects, date, idx.ownedDLC))
	out := countryStart{}
	out.PoliticalPower, _ = foldHistory(calls, "", "add_political_power", 0)
	out.CommandPower, _ = foldHistory(calls, "", "add_command_power",
		idx.defineNumber("NCountry.STARTING_COMMAND_POWER"))
	out.ArmyXP, _ = foldHistory(calls, "", "army_experience", 0)
	out.NavyXP, _ = foldHistory(calls, "", "navy_experience", 0)
	out.AirXP, _ = foldHistory(calls, "", "air_experience", 0)
	nukes, _ := foldHistory(calls, "", "add_nuclear_bombs", 0)
	convoys, _ := foldHistory(calls, "set_convoys", "add_convoys", 0)
	slots, _ := foldHistory(calls, "set_research_slots", "add_research_slot", 0)
	out.Nukes, out.Convoys, out.ResearchSlots = int(nukes), int(convoys), int(slots)

	stability, set := foldHistory(calls, "set_stability", "add_stability", 0)
	if !set {
		stability += idx.defineNumber("NCountry.DEFAULT_STABILITY")
	}
	warSupport, set := foldHistory(calls, "set_war_support", "add_war_support", 0)
	if !set {
		warSupport += idx.defineNumber("NCountry.DEFAULT_WAR_SUPPORT")
	}
	out.Stability, out.WarSupport = clamp01(stability), clamp01(warSupport)
	fuel, set := foldHistory(calls, "set_fuel_ratio", "", 0)
	if !set {
		fuel = idx.defineNumber("NCountry.STARTING_FUEL_RATIO")
	}
	out.FuelRatio = clamp01(fuel)
	return out, ok
}

func (idx *Index) withFocusRewards(calls []effectCall) []effectCall {
	out := make([]effectCall, 0, len(calls))
	for _, call := range calls {
		out = append(out, call)
		if call.Command != "complete_national_focus" {
			continue
		}
		reward := idx.focusRewardOf(call.Token)
		commands := make([]string, 0, len(reward))
		for command := range reward {
			commands = append(commands, command)
		}
		sort.Strings(commands)
		for _, command := range commands {
			out = append(out, effectCall{Date: call.Date, Command: command, Value: reward[command]})
		}
	}
	return out
}

func (idx *Index) WorldTension(date string) float64 {
	date = scenarioDate(date)
	total := 0.0
	for _, tag := range idx.countryHistoryTags() {
		for _, threat := range idx.countryPoliticsFor(tag).NamedThreats {
			if dateReached(threat.Date, date) && threat.DLC.holds(idx.ownedDLC) {
				total += threat.Threat
			}
		}
	}
	if total < 0 {
		return 0
	}
	if total > 100 {
		return 100
	}
	return total
}

func (idx *Index) recordFocusReward(id string, body []*paradox.Entry) {
	if id == "" {
		return
	}
	if _, taken := idx.focusRewards[id]; taken {
		return
	}
	reward := map[string]float64{}
	for _, entry := range body {
		if entry == nil || entry.Key == nil || entry.Block == nil ||
			!strings.EqualFold(*entry.Key, "completion_reward") {
			continue
		}
		if _, kept := idx.focusRewardBlocks[id]; !kept {
			idx.focusRewardBlocks[id] = entry.Block.Entries
		}
		for _, effect := range entry.Block.Entries {
			if effect == nil || effect.Key == nil || effect.Block != nil {
				continue
			}
			key := strings.ToLower(*effect.Key)
			if !focusRewardCommands[key] {
				continue
			}
			if value, ok := toFloat(scalarValue(effect.Scalar)); ok {
				reward[key] += value
			}
		}
	}
	idx.focusRewards[id] = reward
}

func (idx *Index) focusRewardOf(id string) map[string]float64 {
	for layer := idx; layer != nil; layer = layer.fallback {
		if reward, ok := layer.focusRewards[id]; ok {
			return reward
		}
	}
	return nil
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return math.Round(value*1e6) / 1e6
}
