package worldmap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/project"
)

type floodRequest struct {
	Date      string `json:"date"`
	Provinces []int  `json:"provinces"`
}

func parseFlood(body []byte) (floodRequest, string) { return parseDrown(body, "湖") }

func parseDrown(body []byte, what string) (floodRequest, string) {
	var in floodRequest
	if err := json.Unmarshal(body, &in); err != nil {
		return in, "读不懂这次改动"
	}
	if len(in.Provinces) == 0 {
		return in, "没有要变成" + what + "的省份"
	}
	if len(in.Provinces) > 5000 {
		return in, "一次变成" + what + "的省份太多"
	}
	seen := map[int]bool{}
	ids := in.Provinces[:0]
	for _, id := range in.Provinces {
		if id <= 0 {
			return in, "省份编号不对:" + strconv.Itoa(id)
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	in.Provinces = ids
	if strings.TrimSpace(in.Date) == "" {
		in.Date = worldMapDefaultDate
	}
	return in, ""
}

func keepBlocks(text string, gone map[int]bool) string {
	blocks, rest := provinceBlocks(text)
	drop := false
	for p := range blocks {
		if gone[p] {
			drop = true
		}
	}
	if !drop {
		return text
	}
	ids := make([]int, 0, len(blocks))
	for p := range blocks {
		if !gone[p] {
			ids = append(ids, p)
		}
	}
	sort.Ints(ids)
	var parts []string
	for _, p := range ids {
		parts = append(parts, blocks[p])
	}
	return strings.TrimSpace(strings.Join(append(parts, rest...), "\n"))
}

func (h *handlers) drownRoute(w http.ResponseWriter, r *http.Request, p *project.Project, what string, edit func(*landFileSet, floodRequest) (map[string]any, *mapProblem, error)) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		refuseMap(w, http.StatusUnprocessableEntity, problemTooLarge, "改动太大")
		return
	}
	req, problem := parseDrown(body, what)
	if problem != "" {
		writeMapProblem(w, http.StatusUnprocessableEntity, requestProblem(problem))
		return
	}
	select {
	case landGate <- struct{}{}:
		defer func() { <-landGate }()
	default:
		refuseMap(w, http.StatusTooManyRequests, problemBusy, "地图正在改别的地方,稍等几秒再试")
		return
	}
	files, err := h.landFiles(r.Context(), p)
	if err != nil {
		var refused *landRefusal
		if errors.As(err, &refused) {
			refuseMap(w, http.StatusConflict, problemOwnMap, refused.message)
			return
		}
		log.Printf("world map %s %s: %v", what, p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件")
		return
	}
	out, failed, err := edit(files, req)
	if failed != nil {
		writeMapProblem(w, http.StatusUnprocessableEntity, failed)
		return
	}
	if err != nil {
		log.Printf("world map %s %s: %v", what, p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, "没能存进项目,项目没有改动")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) floodRoute(w http.ResponseWriter, r *http.Request, p *project.Project) {
	h.drownRoute(w, r, p, "湖", flood)
}

func floodPlan(provinces [][]any, states [][]any, ids []int) (map[int]bool, []int, *mapProblem) {
	kindOf := map[int]int{}
	stateOf := map[int]int{}
	members := map[int]int{}
	for _, row := range provinces {
		id, _ := row[0].(int)
		kind, _ := row[2].(int)
		state, _ := row[6].(int)
		kindOf[id] = kind + 1
		stateOf[id] = state
		if state > 0 {
			members[state]++
		}
	}
	names := map[int]string{}
	for _, row := range states {
		id, _ := row[0].(int)
		name, _ := row[1].(string)
		names[id] = name
	}
	gone := map[int]bool{}
	flooded := map[int]int{}
	for _, id := range ids {
		switch kindOf[id] {
		case 0:
			return nil, nil, mapProblemOf(mapgen.NoProvince(id))
		case 1:
		default:
			return nil, nil, newMapProblem("already_water", fmt.Sprintf("省份 #%d 已经是水面", id)).with("id", id).about(id)
		}
		gone[id] = true
		if st := stateOf[id]; st > 0 {
			flooded[st]++
		}
	}
	touched := make([]int, 0, len(flooded))
	for st, n := range flooded {
		if n >= members[st] {
			name := names[st]
			if name == "" {
				name = "#" + strconv.Itoa(st)
			}
			var drowned []int
			for _, id := range ids {
				if stateOf[id] == st {
					drowned = append(drowned, id)
				}
			}
			return nil, nil, newMapProblem(problemStateDrowned, fmt.Sprintf("「%s」会整个被淹掉:一个州至少要留一个省份", name)).
				with("state", name).about(drowned...)
		}
		touched = append(touched, st)
	}
	sort.Ints(touched)
	return gone, touched, nil
}

func drownedStates(store *mapStore, touched []int, gone map[int]bool) (updates map[int]WorldMapStateUpdate, before map[int]StateBefore, ensure map[int][]int, pointsLost int, problem *mapProblem, err error) {
	have, err := store.WorldMapStates(touched)
	if err != nil {
		return nil, nil, nil, 0, nil, err
	}
	seeded := map[int]bool{}
	var missing []int
	for _, id := range touched {
		if _, ok := have[id]; !ok {
			missing = append(missing, id)
			seeded[id] = true
		}
	}
	if len(missing) > 0 {
		if err := store.SeedWorldMapStates(missing); err != nil {
			return nil, nil, nil, 0, nil, err
		}
	}
	rows, err := store.WorldMapStateRows(touched)
	if err != nil {
		return nil, nil, nil, 0, nil, err
	}
	updates = map[int]WorldMapStateUpdate{}
	before = map[int]StateBefore{}
	ensure = map[int][]int{}
	for _, id := range touched {
		row, ok := rows[id]
		if !ok {
			return nil, nil, nil, 0, newMapProblem(problemStateMissing, fmt.Sprintf("项目里没有地块 #%d", id)).with("id", id), nil
		}
		var provinces []int
		for _, p := range row.Provinces {
			if !gone[p] {
				provinces = append(provinces, p)
			}
		}
		var points []WorldMapVictoryPoint
		capital, capitalValue := 0, -1.0
		for _, point := range row.VictoryPoints {
			if gone[point.ProvinceID] {
				pointsLost++
				continue
			}
			points = append(points, point)
			if v, err := strconv.ParseFloat(strings.TrimSpace(string(point.Value)), 64); err == nil && v > capitalValue {
				capital, capitalValue = point.ProvinceID, v
			}
		}
		before[id] = StateBefore{Provinces: row.Provinces, Seeded: seeded[id], Row: row.Row}
		updates[id] = WorldMapStateUpdate{
			Provinces: provinces, Owner: row.Owner, Cores: row.Cores, VictoryPoints: points,
			BuildingsText: keepBlocks(row.BuildingsText, gone), Manpower: row.Manpower,
		}
		ensure[id] = capitalFirst(provinces, capital)
	}
	return updates, before, ensure, pointsLost, nil, nil
}

type positionClean struct {
	path  string
	clean func(string) (string, int)
	count string
}

func cleanPositions(files *landFileSet, cleaners []positionClean, texts map[string]string, counts map[string]int) error {
	for _, file := range cleaners {
		body, err := files.current(file.path)
		if err != nil {
			return fmt.Errorf("%s: %w", file.path, err)
		}
		text, n := file.clean(string(body))
		counts[file.count] = n
		if n > 0 {
			texts[file.path] = text
		}
	}
	return nil
}

func flood(files *landFileSet, req floodRequest) (map[string]any, *mapProblem, error) {
	store := files.store
	payload := files.index.WorldMap(req.Date)
	gone, touched, problem := floodPlan(payload.Provinces, payload.States, req.Provinces)
	if problem != nil {
		return nil, problem, nil
	}
	base, err := files.base()
	if err != nil {
		return nil, nil, err
	}

	buildings, err := files.file("map/buildings.txt")
	if err != nil {
		return nil, nil, err
	}
	buildingsText, buildingsDropped := base.DropBuildingsIn(string(buildings), gone)
	add, err := base.Flood(req.Provinces)
	if err != nil {
		return nil, mapProblemOr(err, problemBadRequest), nil
	}

	updates, before, ensure, pointsLost, problem, err := drownedStates(store, touched, gone)
	if problem != nil || err != nil {
		return nil, problem, err
	}

	buildingsText, _, buildingsAdded := base.RestateBuildings(buildingsText, map[int]int{}, ensure, files.p.ID())

	texts := map[string]string{"map/buildings.txt": buildingsText, "map/definition.csv": string(base.DefinitionCSV())}
	counts := map[string]int{"buildings_dropped": buildingsDropped, "buildings_added": buildingsAdded, "points_lost": pointsLost}
	if err := cleanPositions(files, []positionClean{
		{"map/railways.txt", func(t string) (string, int) { return mapgen.CutRailways(t, gone) }, "railways_cut"},
		{"map/supply_nodes.txt", func(t string) (string, int) { return mapgen.DropProvinceLines(t, " ", []int{1}, gone) }, "supply_nodes_dropped"},
		{"map/unitstacks.txt", func(t string) (string, int) { return mapgen.DropProvinceLines(t, ";", []int{0}, gone) }, "unitstacks_dropped"},
		{"map/adjacencies.csv", func(t string) (string, int) { return mapgen.DropProvinceLines(t, ";", []int{0, 1, 3}, gone) }, "adjacencies_dropped"},
	}, texts, counts); err != nil {
		return nil, nil, err
	}

	out := &blobInserts{files: files, note: "世界地图 · 地图创作把省份变成了湖"}
	if err := drownedPictures(out, base, add, "flood"); err != nil {
		return nil, nil, err
	}
	if err := out.texts(texts); err != nil {
		return nil, nil, err
	}
	summary, _ := json.Marshal(map[string]any{"lakes": len(req.Provinces), "states": len(touched), "counts": counts})
	record, err := store.SaveWorldMapRestate(out.inserts, updates, WorldMapAddition{
		Kind: "edit", Name: "地图创作 · 湖", Provinces: req.Provinces, RawPaths: out.paths(), StateBefore: before, Summary: summary,
	})
	if err != nil {
		return nil, nil, err
	}
	files.saved()
	answer := map[string]any{"changed": len(req.Provinces), "record": record, "states": len(touched)}
	for k, v := range counts {
		answer[k] = v
	}
	return answer, nil, nil
}

func drownedPictures(out *blobInserts, base *mapgen.Base, add *mapgen.Addition, what string) error {
	for _, image := range []struct {
		path   string
		encode func() ([]byte, error)
	}{
		{"map/terrain.bmp", func() ([]byte, error) { return mapgen.EncodeBMP(base.TerrainImage()) }},
		{"map/heightmap.bmp", func() ([]byte, error) { return mapgen.EncodeBMP(base.HeightImage()) }},
		{"map/rivers.bmp", func() ([]byte, error) { return mapgen.EncodeBMP(base.RiversImage()) }},
	} {
		data, err := image.encode()
		if err != nil || !out.binary(image.path, data) {
			return fmt.Errorf("%s not stored: %v", image.path, err)
		}
	}
	return out.textures(what, []texturePaint{
		{"map/world_normal.bmp", func(data []byte) ([]byte, error) { return base.PaintNormals(data, add) }},
		{"map/trees.bmp", func(data []byte) ([]byte, error) { return base.ClearTrees(data, add) }},
	})
}

func capitalFirst(provinces []int, capital int) []int {
	order := []int{}
	if capital > 0 && containsInt(provinces, capital) {
		order = append(order, capital)
	}
	for _, p := range provinces {
		if p != capital {
			order = append(order, p)
		}
	}
	return order
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
