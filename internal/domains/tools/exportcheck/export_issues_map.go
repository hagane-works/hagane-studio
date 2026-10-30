package exportcheck

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func waterProvinces(raw []gateRawFile) (map[int64]bool, bool) {
	for _, r := range raw {
		if !strings.EqualFold(strings.TrimSpace(r.DestPath), "map/definition.csv") {
			continue
		}
		if r.IsBinary {
			return nil, false
		}
		water := map[int64]bool{}
		for _, line := range strings.Split(r.Content, "\n") {
			fields := strings.Split(strings.TrimSpace(line), ";")
			if len(fields) < 5 {
				continue
			}
			id, err := strconv.ParseInt(fields[0], 10, 64)
			if err != nil || id <= 0 {
				continue
			}
			if kind := strings.TrimSpace(fields[4]); kind != "land" {
				water[id] = true
			}
		}
		return water, true
	}
	lakes := exportGateVanillaOnce().lakes
	if len(lakes) == 0 {

		return nil, false
	}
	water := make(map[int64]bool, len(lakes))
	for _, id := range lakes {
		water[int64(id)] = true
	}
	return water, true
}

func stateNoLandIssues(states []ValidateState, raw []gateRawFile) []map[string]any {
	water, ok := waterProvinces(raw)
	if !ok {
		return nil
	}
	var bad []ValidateState
	for _, st := range states {
		if len(st.Provinces) == 0 {
			continue
		}
		landed := false
		for _, p := range st.Provinces {
			landed = landed || !water[p]
		}
		if !landed {
			bad = append(bad, st)
		}
	}
	sort.Slice(bad, func(i, j int) bool { return bad[i].StateID < bad[j].StateID })
	out := make([]map[string]any, 0, len(bad))
	for _, st := range bad {
		name := st.Name
		if name == "" {
			name = fmt.Sprintf("#%d", st.StateID)
		}
		out = append(out, map[string]any{
			"level":        "error",
			"code":         "map.state_no_land",
			"severity":     "blocker",
			"entity_ref":   map[string]any{"type": "state", "id": st.StateID},
			"source":       map[string]any{"kind": "state", "identifier": strconv.FormatInt(st.StateID, 10)},
			"evidence":     map[string]any{"state_id": st.StateID, "name": name, "provinces": st.Provinces},
			"repair_route": "map_studio.repair",
			"message": fmt.Sprintf("State %s (#%d) is only a lake, with no land province. "+
				"HOI4 needs land in every state for its air base, rocket site and stronghold, "+
				"and quits when a game starts (error.log stays empty unless the game runs with -debug).",
				name, st.StateID),
			"fix_hint": "Open the map studio and click Fix (or ask the assistant to run repair_world_map): " +
				"the state takes the land province beside its lake. Then export again.",
		})
	}
	return out
}
