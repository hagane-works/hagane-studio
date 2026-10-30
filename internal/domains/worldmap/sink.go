package worldmap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/project"
)

func (h *handlers) sinkRoute(w http.ResponseWriter, r *http.Request, p *project.Project) {
	h.drownRoute(w, r, p, "海", sink)
}

type sinkFacts struct {
	kind, state, region map[int]int
	stateName           map[int]string

	points map[int]map[int]int

	capitals map[int][]string

	divisions map[int]int

	borders map[int]map[int]int
}

func (f sinkFacts) capitalProvince(state int) int {
	best, bestValue := 0, -1
	for p, v := range f.points[state] {
		if v > bestValue || (v == bestValue && p < best) {
			best, bestValue = p, v
		}
	}
	return best
}

type sinkPiece struct {
	provinces []int
	region    int
}

func sinkPlan(ids []int, f sinkFacts) ([]sinkPiece, *mapProblem) {
	chosen := map[int]bool{}
	for _, id := range ids {
		chosen[id] = true
	}
	for _, id := range ids {
		if n := f.divisions[id]; n > 0 {
			return nil, newMapProblem(problemSeaDivisions, fmt.Sprintf("开局有 %d 个师驻在省份 #%d,变成海后它们没处站:请先把它们挪到别的省份", n, id)).
				with("id", id).with("count", n).about(id)
		}
		st := f.state[id]
		if tags := f.capitals[st]; len(tags) > 0 && f.capitalProvince(st) == id {
			name := f.stateName[st]
			if name == "" {
				name = "#" + strconv.Itoa(st)
			}
			return nil, newMapProblem(problemSeaCapital, fmt.Sprintf("省份 #%d 是 %s 的首都(「%s」),不能变成海", id, strings.Join(tags, "、"), name)).
				with("id", id).with("tags", strings.Join(tags, ", ")).with("state", name).about(id)
		}
	}
	seen := map[int]bool{}
	var pieces []sinkPiece
	for _, first := range ids {
		if seen[first] {
			continue
		}
		seen[first] = true
		piece := []int{first}
		shore := map[int]int{}
		for i := 0; i < len(piece); i++ {
			for other, n := range f.borders[piece[i]] {
				switch {
				case chosen[other]:
					if !seen[other] {
						seen[other] = true
						piece = append(piece, other)
					}
				case f.kind[other] == 1:
					shore[other] += n
				}
			}
		}
		sort.Ints(piece)
		if len(shore) == 0 {
			return nil, newMapProblem(problemSeaUnreached, fmt.Sprintf("省份 #%d 不挨着海:变成海的省份要连到已有的海上(湖不算)", piece[0])).
				with("id", piece[0]).about(piece...)
		}
		best, bestEdges := 0, -1
		for sea, n := range shore {
			if n > bestEdges || (n == bestEdges && sea < best) {
				best, bestEdges = sea, n
			}
		}
		pieces = append(pieces, sinkPiece{provinces: piece, region: f.region[best]})
	}
	return pieces, nil
}

func sink(files *landFileSet, req floodRequest) (map[string]any, *mapProblem, error) {
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
	facts := sinkFacts{
		kind: map[int]int{}, state: map[int]int{}, region: map[int]int{}, stateName: map[int]string{},
		points: map[int]map[int]int{}, capitals: files.index.CapitalStates(),
		divisions: files.index.DivisionLocations(req.Date), borders: base.Borders(req.Provinces),
	}
	for _, row := range payload.Provinces {
		id, _ := row[0].(int)
		facts.kind[id], _ = row[2].(int)
		facts.state[id], _ = row[6].(int)
		facts.region[id], _ = row[7].(int)
	}
	for _, row := range payload.States {
		id, _ := row[0].(int)
		facts.stateName[id], _ = row[1].(string)
	}
	for _, row := range payload.Points {
		province, _ := row[0].(int)
		value, _ := row[1].(int)
		st := facts.state[province]
		if facts.points[st] == nil {
			facts.points[st] = map[int]int{}
		}
		facts.points[st][province] = value
	}
	pieces, problem := sinkPlan(req.Provinces, facts)
	if problem != nil {
		return nil, problem, nil
	}

	textFiles := map[string]string{}
	regionText := func(file string) (string, string, error) {
		if text, ok := textFiles[file]; ok {
			return text, "", nil
		}
		return files.regionText(file)
	}
	regionsChanged := 0
	for _, piece := range pieces {
		targetFile, ok := files.index.StrategicRegionFile(piece.region)
		if !ok {
			return nil, newMapProblem(problemSeaNoRegion, fmt.Sprintf("省份 #%d 旁边的海不属于任何战略区,没法把它并进去", piece.provinces[0])).
				with("id", piece.provinces[0]).about(piece.provinces...), nil
		}
		var moved []int
		for _, p := range piece.provinces {
			current, currentFile, inRegion := files.index.StrategicRegionOf(p)
			if inRegion && current == piece.region {
				continue
			}
			if inRegion {
				text, refusal, err := regionText(currentFile)
				if refusal != "" {
					return nil, newMapProblem(problemOwnMap, refusal), nil
				}
				if err != nil {
					return nil, nil, err
				}
				text = mapgen.RemoveFromRegion(text, p)
				if len(mapgen.RegionProvinces(text)) == 0 {
					return nil, newMapProblem(problemRegionEmpty, fmt.Sprintf("战略区 #%d 的省份会全部变成海,战略区就空了", current)).
						with("region", current).about(piece.provinces...), nil
				}
				textFiles[currentFile] = text
			}
			moved = append(moved, p)
		}
		if len(moved) == 0 {
			continue
		}
		text, refusal, err := regionText(targetFile)
		if refusal != "" {
			return nil, newMapProblem(problemOwnMap, refusal), nil
		}
		if err != nil {
			return nil, nil, err
		}
		textFiles[targetFile] = mapgen.AddToRegion(text, moved)
		regionsChanged++
	}

	buildings, err := files.file("map/buildings.txt")
	if err != nil {
		return nil, nil, err
	}
	buildingsText, buildingsDropped := base.DropBuildingsIn(string(buildings), gone)
	add, err := base.Sink(req.Provinces)
	if err != nil {
		return nil, mapProblemOr(err, problemBadRequest), nil
	}

	updates, before, ensure, pointsLost, problem, err := drownedStates(store, touched, gone)
	if problem != nil || err != nil {
		return nil, problem, err
	}

	for _, p := range add.NowCoastal {
		st := facts.state[p]
		if _, done := ensure[st]; done || st == 0 {
			continue
		}
		provinces, ok := files.index.StateProvinces(st)
		if !ok {
			continue
		}
		ensure[st] = capitalFirst(provinces, facts.capitalProvince(st))
	}
	buildingsText, _, buildingsAdded := base.RestateBuildings(buildingsText, map[int]int{}, ensure, files.p.ID())
	buildingsText, coastalAdded := base.AddCoastalBuildings(buildingsText, add.NowCoastal, facts.state, files.p.ID())

	texts := map[string]string{"map/buildings.txt": buildingsText, "map/definition.csv": string(base.DefinitionCSV())}
	for file, text := range textFiles {
		texts[file] = text
	}
	counts := map[string]int{"buildings_dropped": buildingsDropped, "buildings_added": buildingsAdded + coastalAdded,
		"points_lost": pointsLost, "now_coastal": len(add.NowCoastal), "regions_changed": regionsChanged}
	unitStacks := base.UnitStacks(req.Provinces)
	if err := cleanPositions(files, []positionClean{
		{"map/railways.txt", func(t string) (string, int) { return mapgen.CutRailways(t, gone) }, "railways_cut"},
		{"map/supply_nodes.txt", func(t string) (string, int) { return mapgen.DropProvinceLines(t, " ", []int{1}, gone) }, "supply_nodes_dropped"},
		{"map/unitstacks.txt", func(t string) (string, int) {

			text, n := mapgen.DropProvinceLines(t, ";", []int{0}, gone)
			if len(unitStacks) == 0 {
				return text, n
			}
			if text != "" && !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			return text + strings.Join(unitStacks, "\n") + "\n", n + len(unitStacks)
		}, "unitstacks_changed"},
		{"map/adjacencies.csv", func(t string) (string, int) { return mapgen.DropProvinceLines(t, ";", []int{0, 1, 3}, gone) }, "adjacencies_dropped"},
	}, texts, counts); err != nil {
		return nil, nil, err
	}

	out := &blobInserts{files: files, note: "世界地图 · 地图创作把省份变成了海"}
	if err := drownedPictures(out, base, add, "sink"); err != nil {
		return nil, nil, err
	}
	if err := out.texts(texts); err != nil {
		return nil, nil, err
	}
	summary, _ := json.Marshal(map[string]any{"seas": len(req.Provinces), "states": len(touched), "counts": counts})
	record, err := store.SaveWorldMapRestate(out.inserts, updates, WorldMapAddition{
		Kind: "edit", Name: "地图创作 · 海", Provinces: req.Provinces, RawPaths: out.paths(), StateBefore: before, Summary: summary,
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
