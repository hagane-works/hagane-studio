package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type wargoalRecord struct {
	Name string

	Open         bool
	TakeStates   []*paradox.Entry
	BaseCost     float64
	PerStateCost float64
	Limit        float64
	Threat       float64
	Order        int
}

func (idx *Index) indexWargoalFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	if idx.wargoals == nil {
		idx.wargoals = map[string]*wargoalRecord{}
	}
	for _, group := range root.Entries {
		if group == nil || group.Key == nil || group.Block == nil || !strings.EqualFold(*group.Key, "wargoal_types") {
			continue
		}
		for _, item := range group.Block.Entries {
			if item == nil || item.Key == nil || item.Block == nil {
				continue
			}
			record := &wargoalRecord{Name: *item.Key, Open: true}
			if allowed := childBlock(item.Block.Entries, "allowed"); allowed != nil && len(allowed.Entries) > 0 {
				record.Open = false
			}
			if states := childBlock(item.Block.Entries, "take_states"); states != nil {
				record.TakeStates = states.Entries
			}
			numbers := numberBlock(item.Block)
			record.BaseCost = numbers["generate_base_cost"]
			record.PerStateCost = numbers["generate_per_state_cost"]
			record.Limit = numbers["take_states_limit"]
			record.Threat = numbers["threat"]
			idx.wargoalSeq++
			record.Order = idx.wargoalSeq
			idx.wargoals[strings.ToLower(record.Name)] = record
		}
	}
}

func (idx *Index) wargoalList() []*wargoalRecord {
	var layers []*Index
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
	}
	byName := map[string]*wargoalRecord{}
	order := map[string]int{}
	next := 0
	for at := len(layers) - 1; at >= 0; at-- {
		records := make([]*wargoalRecord, 0, len(layers[at].wargoals))
		for _, record := range layers[at].wargoals {
			records = append(records, record)
		}
		sort.Slice(records, func(a, b int) bool { return records[a].Order < records[b].Order })
		for _, record := range records {
			key := strings.ToLower(record.Name)
			if _, seen := order[key]; !seen {
				next++
				order[key] = next
			}
			byName[key] = record
		}
	}
	out := make([]*wargoalRecord, 0, len(byName))
	for _, record := range byName {
		out = append(out, record)
	}
	sort.Slice(out, func(a, b int) bool { return order[strings.ToLower(out[a].Name)] < order[strings.ToLower(out[b].Name)] })
	return out
}

func (idx *Index) WargoalSource(name string) (string, []byte, int, bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	for layer := idx; layer != nil; layer = layer.fallback {
		paths := []string{}
		for path := range layer.files {
			lower := strings.ToLower(path)
			if strings.HasPrefix(lower, "common/wargoals/") && strings.HasSuffix(lower, ".txt") {
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
				if group == nil || group.Key == nil || group.Block == nil || !strings.EqualFold(*group.Key, "wargoal_types") {
					continue
				}
				for _, item := range group.Block.Entries {
					if item != nil && item.Key != nil && item.Block != nil && strings.EqualFold(*item.Key, want) {
						return path, layer.files[path], item.Line, true
					}
				}
			}
		}
	}
	return "", nil, 0, false
}

func (idx *Index) stateTaken(state *stateRecord, conditions []*paradox.Entry, root, prev, date string) bool {
	owner := idx.ownerOfState(state, date)
	if len(conditions) == 0 {
		return owner == prev
	}
	resolve := func(value string) string {
		switch strings.ToUpper(strings.Trim(value, `"`)) {
		case "ROOT":
			return root
		case "PREV", "FROM":
			return prev
		}
		return strings.ToUpper(strings.Trim(value, `"`))
	}
	has := func(list []string, tag string) bool {
		for _, item := range list {
			if strings.EqualFold(item, tag) {
				return true
			}
		}
		return false
	}
	for _, condition := range conditions {
		if condition == nil || condition.Key == nil || condition.Block != nil {
			return false
		}
		value := toString(scalarValue(condition.Scalar))
		switch strings.ToLower(*condition.Key) {
		case "is_core_of":
			if !has(tokensByOwned(state.Cores, date, idx.ownedDLC), resolve(value)) {
				return false
			}
		case "is_claimed_by":
			if !has(tokensByOwned(state.Claims, date, idx.ownedDLC), resolve(value)) {
				return false
			}
		case "is_owned_by", "is_controlled_by":
			if owner != resolve(value) {
				return false
			}
		case "always":
			if !strings.EqualFold(value, "yes") {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (idx *Index) stateLabel(state *stateRecord, owner string) string {
	if name, ok := idx.Localise(owner + "_" + state.Name); ok {
		return name
	}
	return idx.localiseOr(state.Name, state.Name)
}

func (idx *Index) WarGoalPayload(tag, target, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	target = strings.ToUpper(strings.TrimSpace(target))
	states := idx.effectiveStates()
	sort.Slice(states, func(a, b int) bool { return states[a].ID < states[b].ID })
	types := []map[string]any{}
	for _, goal := range idx.wargoalList() {
		if !goal.Open || target == "" || target == tag {
			continue
		}
		picked := []map[string]any{}
		for _, state := range states {

			if idx.ownerOfState(state, date) != target {
				continue
			}
			if idx.stateTaken(state, goal.TakeStates, tag, target, date) {
				picked = append(picked, map[string]any{"id": state.ID, "name": idx.stateLabel(state, target)})
			}
		}
		if len(picked) == 0 {
			continue
		}
		types = append(types, map[string]any{
			"id": goal.Name, "label": idx.localiseOr(goal.Name, goal.Name),
			"base_cost": goal.BaseCost, "per_state_cost": goal.PerStateCost,
			"limit": goal.Limit, "threat": goal.Threat, "states": picked,
		})
	}

	costFormat, _ := idx.locValue("DIPLOMACY_GENERATE_WARGOAL_DESC")
	return map[string]any{
		"tag": tag, "target": target, "types": types,
		"daily_pp": idx.defineNumber("NDiplomacy.BASE_GENERATE_WARGOAL_DAILY_PP"),
		"texts": map[string]any{
			"title":       idx.localiseOr("DIPLOMACY_GENERATE_WARGOAL_TITLE", "Justify War Goal"),
			"no_wargoal":  idx.localiseOr("DIPLOMACY_NO_WARGOAL", "No war goal selected"),
			"select_type": idx.localiseOr("DIPLOMACY_SELECT_WARGOAL_TYPE", "Select a war goal type"),

			"cost": costFormat,
		},

		"text_keys": map[string]any{
			"title":       "DIPLOMACY_GENERATE_WARGOAL_TITLE",
			"no_wargoal":  "DIPLOMACY_NO_WARGOAL",
			"select_type": "DIPLOMACY_SELECT_WARGOAL_TYPE",
			"cost":        "DIPLOMACY_GENERATE_WARGOAL_DESC",
		},
	}
}
