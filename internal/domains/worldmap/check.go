package worldmap

import (
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/project"
)

func (h *handlers) check(w http.ResponseWriter, r *http.Request, p *project.Project) {
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		date = worldMapDefaultDate
	}
	files, err := h.landFiles(r.Context(), p)
	if err != nil {
		var refused *landRefusal
		if errors.As(err, &refused) {
			refuseMap(w, http.StatusConflict, problemOwnMap, refused.message)
			return
		}
		log.Printf("world map check %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件")
		return
	}
	out, err := checkMap(files, date)
	if err != nil {
		log.Printf("world map check %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func checkMap(files *landFileSet, date string) (map[string]any, error) {
	base, err := files.sharedBase()
	if err != nil {
		return nil, err
	}
	regions, _, err := files.projectRegions()
	if err != nil {
		return nil, err
	}
	return mapCheck(base, regions, files.index.WorldMap(date)), nil
}

func mapCheck(base *mapgen.Base, regions map[int][]int, payload guirender.WorldMapPayload) map[string]any {
	regionOf := map[int]int{}
	for id, provinces := range regions {
		for _, p := range provinces {
			regionOf[p] = id
		}
	}
	type named struct{ name, owner string }
	info := map[int]named{}
	for _, row := range payload.States {
		id, _ := row[0].(int)
		name, _ := row[1].(string)
		owner, _ := row[2].(string)
		info[id] = named{name, owner}
	}
	states := map[int][]int{}
	for _, row := range payload.Provinces {
		id, _ := row[0].(int)
		if state, _ := row[6].(int); state > 0 {
			states[state] = append(states[state], id)
		}
	}
	findings := base.LoadRules(regions, states)

	isLand := base.IsLand()
	ids := make([]int, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	landless, across := []map[string]any{}, []map[string]any{}
	for _, id := range ids {
		provinces := states[id]
		sort.Ints(provinces)
		landed := false
		seen := map[int]bool{}
		for _, p := range provinces {
			landed = landed || isLand(p)
			if r := regionOf[p]; r > 0 {
				seen[r] = true
			}
		}
		if !landed {
			landless = append(landless, map[string]any{"state": id, "name": info[id].name, "owner": info[id].owner, "provinces": provinces})
		}
		if len(seen) > 1 {
			list := make([]int, 0, len(seen))
			for r := range seen {
				list = append(list, r)
			}
			sort.Ints(list)
			across = append(across, map[string]any{"state": id, "name": info[id].name, "regions": list})
		}
	}
	out := map[string]any{
		"ok": len(findings) == 0, "findings": findings,
		"states_without_land": landless, "states_across_regions": across,
	}
	if len(findings) > 0 {
		out["fix"] = "repair_world_map mends all of these (the map studio's Fix button); the player can undo it"
	}
	return out
}
