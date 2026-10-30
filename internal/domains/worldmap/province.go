package worldmap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/project"
)

var landTerrains = map[string]bool{
	"plains": true, "forest": true, "hills": true, "mountain": true,
	"desert": true, "jungle": true, "urban": true, "marsh": true,
}

type provinceEdit struct {
	Province  int     `json:"province"`
	Kind      *string `json:"kind"`
	Terrain   *string `json:"terrain"`
	Continent *int    `json:"continent"`
	State     *int    `json:"state"`
	Region    *int    `json:"region"`
}

func (h *handlers) province(w http.ResponseWriter, r *http.Request, p *project.Project) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	var in provinceEdit
	if err != nil || json.Unmarshal(body, &in) != nil || in.Province <= 0 {
		refuseMap(w, http.StatusUnprocessableEntity, problemBadRequest, "要说清改哪个省份")
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
		log.Printf("world map province %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件")
		return
	}
	edit, problem, err := editProvince(files, in)
	if problem != nil {
		writeMapProblem(w, http.StatusUnprocessableEntity, problem.about(in.Province))
		return
	}
	if err != nil {
		log.Printf("world map province %s #%d: %v", p.Dir(), in.Province, err)
		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, "改动没能存进项目,项目没有改动")
		return
	}
	writeJSON(w, http.StatusOK, edit)
}

func editProvince(files *landFileSet, in provinceEdit) (map[string]any, *mapProblem, error) {
	store := files.store
	var base *mapgen.Base
	var definitions []mapgen.Definition
	if in.Kind != nil {
		loaded, err := files.base()
		if err != nil {
			return nil, nil, err
		}
		base, definitions = loaded, loaded.Definitions
	} else {
		body, err := files.file("map/definition.csv")
		if err != nil {
			return nil, nil, err
		}
		definitions = mapgen.ParseDefinitions(body)
	}
	at := -1
	for i := range definitions {
		if definitions[i].ID == in.Province {
			at = i
		}
	}
	if at < 0 {
		return nil, newMapProblem(problemNoProvince, fmt.Sprintf("没有省份 #%d", in.Province)).with("id", in.Province), nil
	}
	row := &definitions[at]
	isLand := row.Kind == "land"
	var changed []string
	definitionChanged, terrainMapChanged := false, false
	var raised *mapgen.Addition

	if in.Kind != nil && *in.Kind != row.Kind {
		kind := *in.Kind
		if kind != "land" && kind != "sea" && kind != "lake" {
			return nil, newMapProblem(problemKindInvalid, "省份类型只能是陆地、海洋或湖泊"), nil
		}
		if isLand {
			request := floodRequest{Date: worldMapDefaultDate, Provinces: []int{in.Province}}
			if kind == "sea" {
				return sink(files, request)
			}
			return flood(files, request)
		}
		if kind == "land" {
			state, region, terrain, continent, problem := raisedProvinceTarget(files, base, definitions, in)
			if problem != nil {
				return nil, problem, nil
			}
			in.State, in.Region, in.Terrain, in.Continent = &state, &region, &terrain, &continent
			var err error
			raised, err = base.Raise(in.Province, terrain, continent)
			if err != nil {
				return nil, mapProblemOr(err, problemBadRequest), nil
			}
			definitions = base.Definitions
			row = &definitions[at]
			isLand = true
			changed = append(changed, "kind", "terrain", "continent", "heightmap", "rivers")
			definitionChanged, terrainMapChanged = true, true
		} else {
			if problem := switchWater(base, row, kind); problem != nil {
				return nil, problem, nil
			}
			changed = append(changed, "kind")
			definitionChanged, terrainMapChanged = true, true
		}
	}
	if in.Terrain != nil && *in.Terrain != row.Terrain {
		if !isLand {
			return nil, newMapProblem(problemWaterTerrain, "海洋和湖泊的地形跟着类型走,不能单独改").with("id", in.Province), nil
		}
		if !landTerrains[*in.Terrain] {
			return nil, newMapProblem(problemTerrainUnknown, "不认识的地形 "+*in.Terrain).with("terrain", *in.Terrain), nil
		}
		row.Terrain = *in.Terrain
		changed = append(changed, "terrain")
		definitionChanged = true
	}
	if in.Continent != nil && *in.Continent != row.Continent {
		count := files.index.ContinentCount()
		if !isLand {
			return nil, newMapProblem(problemWaterContinent, "海洋和湖泊不属于大洲"), nil
		}
		if *in.Continent < 1 || *in.Continent > count {
			return nil, newMapProblem(problemContinentRange, fmt.Sprintf("大洲要在 1 到 %d 之间", count)).with("count", count), nil
		}
		row.Continent = *in.Continent
		changed = append(changed, "continent")
		definitionChanged = true
	}

	stateLists := map[int][]int{}
	before := map[int]StateBefore{}
	if in.State != nil {
		if !isLand {
			return nil, newMapProblem(problemStateLandOnly, "只有陆地省份属于地块"), nil
		}
		target := *in.State
		if _, ok := files.index.StateProvinces(target); !ok {
			return nil, newMapProblem(problemStateMissing, fmt.Sprintf("没有地块 #%d", target)).with("id", target), nil
		}
		current, inState := files.index.StateOfProvince(in.Province)
		if !inState || current != target {
			ids := []int{target}
			if inState {
				currentProvinces, _ := files.index.StateProvinces(current)
				if len(currentProvinces) <= 1 {
					return nil, newMapProblem(problemStateLastProvince, fmt.Sprintf("地块 #%d 只有这一个省份,移走它地块就空了", current)).with("state", current), nil
				}

				kindOf := map[int]string{}
				for _, d := range definitions {
					kindOf[d.ID] = d.Kind
				}
				landLeft := false
				for _, p := range currentProvinces {
					landLeft = landLeft || (p != in.Province && kindOf[p] == "land")
				}
				if !landLeft {
					return nil, newMapProblem(problemStateLastLand, fmt.Sprintf("地块 #%d 只剩这一个陆地省份,移走它就只剩湖泊,放不下机场和要塞,游戏开局会退出", current)).with("state", current), nil
				}
				ids = append(ids, current)
			}
			rows, err := store.WorldMapStates(ids)
			if err != nil {
				return nil, nil, err
			}
			var missing []int
			for _, id := range ids {
				if _, ok := rows[id]; !ok {
					missing = append(missing, id)
				}
			}
			if len(missing) > 0 {
				if err := store.SeedWorldMapStates(missing); err != nil {
					return nil, nil, err
				}
				if rows, err = store.WorldMapStates(ids); err != nil {
					return nil, nil, err
				}
				for _, id := range missing {
					seeded, ok := rows[id]
					if !ok {
						return nil, nil, fmt.Errorf("state %d could not be seeded", id)
					}
					seeded.Seeded = true
					rows[id] = seeded
				}
			}
			for _, id := range ids {
				before[id] = rows[id]
			}
			moved := []int{}
			for _, p := range rows[target].Provinces {
				if p != in.Province {
					moved = append(moved, p)
				}
			}
			stateLists[target] = append(moved, in.Province)
			if inState {
				left := []int{}
				for _, p := range rows[current].Provinces {
					if p != in.Province {
						left = append(left, p)
					}
				}
				stateLists[current] = left
			}
			changed = append(changed, "state")
		}
	}

	textFiles := map[string]string{}
	if in.Region != nil {
		target := *in.Region
		targetFile, ok := files.index.StrategicRegionFile(target)
		if !ok {
			return nil, newMapProblem(problemRegionMissing, fmt.Sprintf("没有战略区 #%d", target)).with("region", target), nil
		}
		current, currentFile, inRegion := files.index.StrategicRegionOf(in.Province)
		if !inRegion || current != target {
			if inRegion {
				text, refusal, err := files.regionText(currentFile)
				if refusal != "" {
					return nil, newMapProblem(problemRegionFileText, refusal).with("file", currentFile), nil
				}
				if err != nil {
					return nil, nil, err
				}
				if len(mapgen.RegionProvinces(text)) <= 1 {
					return nil, newMapProblem(problemRegionLastProvince, fmt.Sprintf("战略区 #%d 只有这一个省份,移走它战略区就空了", current)).with("region", current), nil
				}
				textFiles[currentFile] = mapgen.RemoveFromRegion(text, in.Province)
			}
			text, refusal, err := files.regionText(targetFile)
			if refusal != "" {
				return nil, newMapProblem(problemRegionFileText, refusal).with("file", targetFile), nil
			}
			if err != nil {
				return nil, nil, err
			}

			textFiles[targetFile] = mapgen.AddToRegion(text, []int{in.Province})
			changed = append(changed, "region")
		}
	}

	if problem, err := provinceEditLoadRules(files, base, definitions, in, textFiles); problem != nil || err != nil {
		return nil, problem, err
	}

	if len(changed) == 0 {
		return map[string]any{"changed": []string{}}, nil, nil
	}
	if definitionChanged {
		textFiles["map/definition.csv"] = string(mapgen.DefinitionsCSV(definitions))
	}
	note := fmt.Sprintf("世界地图 · 改省份 #%d", in.Province)
	var inserts []RawFileInsert
	paths := make([]string, 0, len(textFiles))
	for dest := range textFiles {
		paths = append(paths, dest)
	}
	sort.Strings(paths)
	for _, dest := range paths {
		inserts = append(inserts, RawFileInsert{DestPath: dest, Content: textFiles[dest], RawKind: "verbatim_override", Note: note})
	}
	if raised != nil {
		out := &blobInserts{files: files, note: note}
		if err := drownedPictures(out, base, raised, "raise"); err != nil {
			return nil, nil, err
		}
		inserts = append(inserts, out.inserts...)
	} else if terrainMapChanged {
		data, err := mapgen.EncodeBMP(base.TerrainImage())
		if err != nil {
			return nil, nil, err
		}
		sha, key, ok := files.storeBlob(data)
		if !ok {
			return nil, nil, errors.New("map/terrain.bmp not stored")
		}
		inserts = append(inserts, RawFileInsert{DestPath: "map/terrain.bmp", IsBinary: true, RawKind: "verbatim_override", Note: note, ContentSHA256: &sha, ContentOBSKey: &key})
	}
	rawPaths := make([]string, 0, len(inserts))
	for _, insert := range inserts {
		rawPaths = append(rawPaths, insert.DestPath)
	}
	summary, _ := json.Marshal(map[string]any{"changed": changed, "edit": in})
	id, err := store.SaveWorldMapAddition(inserts, stateLists, WorldMapAddition{
		Kind: "edit", Provinces: []int{in.Province}, RawPaths: rawPaths, StateBefore: before, Summary: summary,
	})
	if err != nil {
		return nil, nil, err
	}
	files.saved()
	return map[string]any{"changed": changed, "record": id}, nil, nil
}

func raisedProvinceTarget(files *landFileSet, base *mapgen.Base, definitions []mapgen.Definition, in provinceEdit) (int, int, string, int, *mapProblem) {
	definitionOf := map[int]mapgen.Definition{}
	for _, definition := range definitions {
		definitionOf[definition.ID] = definition
	}
	state, region := 0, 0
	terrain, continent := "", 0
	if in.State != nil {
		state = *in.State
		provinces, ok := files.index.StateProvinces(state)
		if !ok {
			return 0, 0, "", 0, newMapProblem(problemStateMissing, fmt.Sprintf("没有地块 #%d", state)).with("id", state)
		}
		for _, province := range provinces {
			definition := definitionOf[province]
			if definition.Kind != "land" {
				continue
			}
			terrain, continent = definition.Terrain, definition.Continent
			region, _, _ = files.index.StrategicRegionOf(province)
			break
		}
	}
	if state == 0 {
		best, bestBorder := 0, -1
		for province, border := range base.Borders([]int{in.Province})[in.Province] {
			definition := definitionOf[province]
			candidateState, inState := files.index.StateOfProvince(province)
			candidateRegion, _, inRegion := files.index.StrategicRegionOf(province)
			if definition.Kind != "land" || !inState || !inRegion || border <= bestBorder {
				continue
			}
			best, bestBorder = province, border
			state, region, terrain, continent = candidateState, candidateRegion, definition.Terrain, definition.Continent
		}
		if best == 0 {
			return 0, 0, "", 0, newMapProblem(problemKindLandWater, "这个水域省份不挨着可加入的陆地地块;请指定 state 和 region,或使用「新增土地」建立岛屿").with("id", in.Province)
		}
	}
	if in.Region != nil {
		region = *in.Region
	}
	if _, ok := files.index.StrategicRegionFile(region); !ok {
		return 0, 0, "", 0, newMapProblem(problemRegionMissing, fmt.Sprintf("没有战略区 #%d", region)).with("region", region)
	}
	if in.Terrain != nil {
		terrain = *in.Terrain
	}
	if !landTerrains[terrain] {
		return 0, 0, "", 0, newMapProblem(problemTerrainUnknown, "不认识的地形 "+terrain).with("terrain", terrain)
	}
	if in.Continent != nil {
		continent = *in.Continent
	}
	if continent < 1 || continent > files.index.ContinentCount() {
		return 0, 0, "", 0, newMapProblem(problemContinentRange, fmt.Sprintf("大洲要在 1 到 %d 之间", files.index.ContinentCount())).with("count", files.index.ContinentCount())
	}
	return state, region, terrain, continent, nil
}

func switchWater(base *mapgen.Base, row *mapgen.Definition, kind string) *mapProblem {
	kindOf := map[int]string{}
	for _, d := range base.Definitions {
		kindOf[d.ID] = d.Kind
	}
	id := int32(row.ID)

	neighbours := map[int32]bool{}
	var pixels []int
	for p, v := range base.Province {
		if v != id {
			continue
		}
		pixels = append(pixels, p)
		x := p % base.Width
		for _, q := range [4]int{p - 1, p + 1, p - base.Width, p + base.Width} {
			if q < 0 || q >= len(base.Province) || (q == p-1 && x == 0) || (q == p+1 && x == base.Width-1) {
				continue
			}
			if other := base.Province[q]; other != id && kindOf[int(other)] == "land" {
				neighbours[other] = true
			}
		}
	}
	seas := map[int32]map[int32]bool{}
	for p, a := range base.Province {
		x := p % base.Width
		for _, q := range [2]int{p + 1, p + base.Width} {
			if q >= len(base.Province) || (q == p+1 && x == base.Width-1) {
				continue
			}
			b := base.Province[q]
			if a == b {
				continue
			}
			if neighbours[a] && kindOf[int(b)] == "sea" {
				if seas[a] == nil {
					seas[a] = map[int32]bool{}
				}
				seas[a][b] = true
			}
			if neighbours[b] && kindOf[int(a)] == "sea" {
				if seas[b] == nil {
					seas[b] = map[int32]bool{}
				}
				seas[b][a] = true
			}
		}
	}
	at := map[int]int{}
	for i, d := range base.Definitions {
		at[d.ID] = i
	}
	if kind == "lake" {
		var stranded []string
		for land := range neighbours {
			others := 0
			for sea := range seas[land] {
				if sea != id {
					others++
				}
			}
			if others == 0 && base.Definitions[at[int(land)]].Coastal {
				stranded = append(stranded, fmt.Sprintf("#%d", land))
			}
		}
		if len(stranded) > 0 {
			sort.Strings(stranded)
			return newMapProblem(problemLakeStrandsCoast, "改成湖泊会让这些沿海省份不再挨着海,港口就没处放了:"+strings.Join(stranded, "、")).with("provinces", strings.Join(stranded, ", "))
		}
		row.Kind, row.Terrain, row.Coastal = "lake", "lakes", false
	} else {
		row.Kind, row.Terrain, row.Coastal = "sea", "ocean", len(neighbours) > 0
		for land := range neighbours {
			base.Definitions[at[int(land)]].Coastal = true
		}
	}
	index := mapgen.Legend[mapgen.Sea].Terrain
	if kind == "lake" {
		index = mapgen.Legend[mapgen.Lake].Terrain
	}
	for _, p := range pixels {
		base.Terrain[p] = index
	}
	return nil
}
