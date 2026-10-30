package worldmap

import (
	"bytes"
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

func landSeaRegionMoves(files *landFileSet, base *mapgen.Base, add *mapgen.Addition, regionID int) ([]mapgen.RegionMove, error) {
	regions, _, err := files.projectRegions()
	if err != nil {
		return nil, err
	}
	for _, pr := range add.Provinces {
		region := regionID
		if pr.Kind == "sea" {
			if id, _, ok := files.index.StrategicRegionOf(add.CutFrom[pr.ID]); ok {
				region = id
			}
		}
		regions[region] = append(regions[region], pr.ID)
	}
	return base.MendSeaRegions(regions, files.stateOf())
}

func applySeaRegionMoves(files *landFileSet, moves []mapgen.RegionMove, texts map[string]string, fileOf func(int) (string, bool)) error {
	read := func(region int) (string, string, error) {
		file, ok := fileOf(region)
		if !ok {
			return "", "", fmt.Errorf("strategic region %d has no file", region)
		}
		if text, ok := texts[file]; ok {
			return file, text, nil
		}
		text, refusal, err := files.regionText(file)
		if refusal != "" {
			return "", "", &landRefusal{refusal}
		}
		return file, text, err
	}
	for _, move := range moves {
		from, text, err := read(move.From)
		if err != nil {
			return err
		}
		for _, p := range move.Provinces {
			text = mapgen.RemoveFromRegion(text, p)
		}
		texts[from] = text
		to, text, err := read(move.To)
		if err != nil {
			return err
		}
		texts[to] = mapgen.AddToRegion(text, move.Provinces)
	}
	return nil
}

type repairRequest struct {
	Date      string   `json:"date"`
	Languages []string `json:"languages"`
}

func (h *handlers) repair(w http.ResponseWriter, r *http.Request, p *project.Project) {
	var req repairRequest
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil || (len(bytes.TrimSpace(body)) > 0 && json.Unmarshal(body, &req) != nil) {
		refuseMap(w, http.StatusUnprocessableEntity, problemBadRequest, "读不懂这次改动")
		return
	}
	if strings.TrimSpace(req.Date) == "" {
		req.Date = worldMapDefaultDate
	}
	if len(req.Languages) == 0 {
		req.Languages = []string{"en"}
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
		log.Printf("world map repair %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件")
		return
	}
	out, failed, err := repairMap(files, req)
	if failed != nil {
		writeMapProblem(w, http.StatusUnprocessableEntity, failed)
		return
	}
	if err != nil {
		log.Printf("world map repair %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, "没能存进项目,项目没有改动")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func repairMap(files *landFileSet, req repairRequest) (map[string]any, *mapProblem, error) {
	base, err := files.base()
	if err != nil {
		return nil, nil, err
	}
	regions, regionFiles, err := files.projectRegions()
	if err != nil {
		return nil, nil, err
	}
	regionOf := map[int]int{}
	for id, provinces := range regions {
		for _, p := range provinces {
			regionOf[p] = id
		}
	}
	out := map[string]any{}
	records := 0

	payload := files.index.WorldMap(req.Date)
	states := map[int]mapgen.RestateState{}
	names := map[int]string{}
	for _, row := range payload.States {
		id, _ := row[0].(int)
		name, _ := row[1].(string)
		owner, _ := row[2].(string)
		states[id] = mapgen.RestateState{Owner: owner}
		names[id] = name
	}
	stateOf := map[int]int{}
	for _, row := range payload.Provinces {
		id, _ := row[0].(int)
		state, _ := row[6].(int)
		if st, ok := states[state]; ok && state > 0 {
			st.Provinces = append(st.Provinces, id)
			states[state] = st
			stateOf[id] = state
		}
	}

	capitalOf, best := map[int]int{}, map[int]float64{}
	for _, row := range payload.Points {
		province, _ := row[0].(int)
		value, _ := row[1].(float64)
		if v, ok := row[1].(int); ok {
			value = float64(v)
		}
		if state := stateOf[province]; state > 0 && (capitalOf[state] == 0 || value > best[state]) {
			capitalOf[state], best[state] = province, value
		}
	}

	lakePlan := base.LandForLakeStates(states, regionOf)
	if len(lakePlan.Lists) > 0 {
		var moved []int
		for id, list := range lakePlan.Lists {
			st := states[id]
			for _, p := range list {
				if stateOf[p] != id {
					moved = append(moved, p)
					stateOf[p] = id
				}
			}
			st.Provinces = list
			if tag, ok := lakePlan.Owner[id]; ok {
				st.Owner = tag
			}
			states[id] = st
		}
		sort.Ints(moved)
		applied, problem, err := applyRestatePlan(files, base, lakePlan, names,
			restateRequest{Date: req.Date, Languages: req.Languages}, moved, "地图创作 · 给只剩湖泊的州补上陆地")
		if problem != nil || err != nil {
			return nil, problem, err
		}
		records++
		out["lakes"] = applied
	}
	plan := mapgen.SplitStatesByRegion(states, base.Neighbours(), regionOf, capitalOf)
	if len(plan.Lists) > 0 {
		var moved []int
		for k, n := range plan.New {
			moved = append(moved, n.Provinces...)
			for _, p := range n.Provinces {
				stateOf[p] = -(k + 1)
			}
		}
		sort.Ints(moved)
		applied, problem, err := applyRestatePlan(files, base, plan, names,
			restateRequest{Date: req.Date, Languages: req.Languages}, moved, "地图创作 · 拆开跨战略区的州")
		if problem != nil || err != nil {
			return nil, problem, err
		}
		records++
		out["states"] = applied
	}

	moves, err := base.MendSeaRegions(regions, stateOf)
	if problem := mapProblemOf(err); problem != nil {
		return nil, problem, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if len(moves) > 0 {
		record, problem, err := saveSeaRegionMoves(files, moves, regionFiles)
		if problem != nil || err != nil {
			return nil, problem, err
		}
		records++
		out["moves"], out["record"] = moves, record
	}
	if records == 0 {
		return nil, newMapProblem(problemNothing, "地图没有会让游戏读地图时退出的问题,没有要修的"), nil
	}
	out["records"] = records
	return out, nil, nil
}

func saveSeaRegionMoves(files *landFileSet, moves []mapgen.RegionMove, regionFiles map[int]string) (int64, *mapProblem, error) {
	texts := map[string]string{}
	err := applySeaRegionMoves(files, moves, texts, func(id int) (string, bool) { file, ok := regionFiles[id]; return file, ok })
	if problem := mapProblemOf(err); problem != nil {
		return 0, problem, nil
	}
	if err != nil {
		return 0, nil, err
	}
	paths := make([]string, 0, len(texts))
	for dest := range texts {
		paths = append(paths, dest)
	}
	sort.Strings(paths)
	note := "世界地图 · 地图创作把断开的海上战略区并好"
	var inserts []RawFileInsert
	for _, dest := range paths {
		inserts = append(inserts, RawFileInsert{DestPath: dest, Content: texts[dest], RawKind: "verbatim_override", Note: note})
	}
	var provinces []int
	for _, move := range moves {
		provinces = append(provinces, move.Provinces...)
	}
	summary, _ := json.Marshal(map[string]any{"sea_region_moves": moves})
	record, err := files.store.SaveWorldMapRestate(inserts, nil, WorldMapAddition{
		Kind: "edit", Name: "地图创作 · 修复海上战略区", Provinces: provinces, RawPaths: paths, Summary: summary,
	})
	if err != nil {
		return 0, nil, err
	}
	files.saved()
	return record, nil, nil
}

func provinceEditLoadRules(files *landFileSet, base *mapgen.Base, definitions []mapgen.Definition, in provinceEdit, textFiles map[string]string) (*mapProblem, error) {
	if in.State == nil && in.Region == nil {
		return nil, nil
	}
	region, _, _ := files.index.StrategicRegionOf(in.Province)
	if in.Region != nil {
		region = *in.Region
	}
	state, inState := files.index.StateOfProvince(in.Province)
	if in.State != nil {
		state, inState = *in.State, true
	}
	if inState {
		provinces, _ := files.index.StateProvinces(state)
		for _, p := range provinces {
			if p == in.Province {
				continue
			}
			if other, _, ok := files.index.StrategicRegionOf(p); ok && other != region {
				return newMapProblem(problemProvinceRegion, fmt.Sprintf("地块 #%d 的省份在战略区 #%d,省份 #%d 会在战略区 #%d:一个地块的省份要在同一个战略区里,不然游戏读地图时会直接退出", state, other, in.Province, region)).
					with("state", state).with("other", other).with("id", in.Province).with("region", region), nil
			}
		}
	}
	if in.Region == nil {
		return nil, nil
	}
	kind := map[int]string{}
	for _, d := range definitions {
		kind[d.ID] = d.Kind
	}
	paths := make([]string, 0, len(textFiles))
	for file := range textFiles {
		paths = append(paths, file)
	}
	sort.Strings(paths)
	for _, file := range paths {
		provinces := mapgen.RegionProvinces(textFiles[file])
		naval := false
		for _, p := range provinces {
			naval = naval || kind[p] == "sea"
		}
		if !naval {
			continue
		}
		if base == nil {
			loaded, err := files.base()
			if err != nil {
				return nil, err
			}
			base = loaded
		}
		if cut := base.SeaRegionCutOff(provinces); cut != nil {
			return newMapProblem(problemProvinceSeaSplit, fmt.Sprintf("这样改会让 %s 里的海断成几块(省份 #%d 等连不上):海上战略区的海要连成一片,不然游戏读地图时会直接退出", file, cut[0])).
				with("file", file).with("id", cut[0]).about(cut...), nil
		}
	}
	return nil, nil
}
