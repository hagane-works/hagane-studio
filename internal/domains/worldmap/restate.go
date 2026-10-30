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

	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/project"
)

type restateRequest struct {
	Date      string            `json:"date"`
	Provinces map[string]string `json:"provinces"`
	AddCores  bool              `json:"add_cores"`
	Languages []string          `json:"languages"`
}

func parseRestate(body []byte) (restateRequest, map[int]string, string) {
	var in restateRequest
	if err := json.Unmarshal(body, &in); err != nil {
		return in, nil, "读不懂这次改动"
	}
	if len(in.Provinces) == 0 {
		return in, nil, "没有要改的省份"
	}
	if len(in.Provinces) > 20000 {
		return in, nil, "一次改的省份太多"
	}
	assign := map[int]string{}
	for key, tag := range in.Provinces {
		id, err := strconv.Atoi(strings.TrimSpace(key))
		if err != nil || id <= 0 {
			return in, nil, "省份编号不对:" + key
		}
		tag = strings.ToUpper(strings.TrimSpace(tag))
		if !landTagPattern.MatchString(tag) {
			return in, nil, "国家要是三个字母的标签,比如 GER"
		}
		assign[id] = tag
	}
	if strings.TrimSpace(in.Date) == "" {
		in.Date = worldMapDefaultDate
	}
	if len(in.Languages) == 0 {
		in.Languages = []string{"en"}
	}
	return in, assign, ""
}

func (h *handlers) restateRoute(w http.ResponseWriter, r *http.Request, p *project.Project) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<20))
	if err != nil {
		refuseMap(w, http.StatusUnprocessableEntity, problemTooLarge, "改动太大")
		return
	}
	req, assign, problem := parseRestate(body)
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
	ctx := r.Context()
	files, err := h.landFiles(ctx, p)
	if err != nil {
		var refused *landRefusal
		if errors.As(err, &refused) {
			refuseMap(w, http.StatusConflict, problemOwnMap, refused.message)
			return
		}
		log.Printf("world map restate %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件")
		return
	}

	unknown := func() string {
		known := files.index.ListCountryTags()
		for _, tag := range assign {
			if !containsTag(known, tag) {
				return tag
			}
		}
		return ""
	}
	if unknown() != "" {
		h.svc.Forget(p)
		if files, err = h.landFiles(ctx, p); err != nil {
			log.Printf("world map restate %s: %v", p.Dir(), err)
			refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件")
			return
		}
	}
	if tag := unknown(); tag != "" {
		writeMapProblem(w, http.StatusUnprocessableEntity, newMapProblem(problemNoCountry, "没有国家 "+tag+":先新建它").with("tag", tag))
		return
	}
	out, failed, err := restate(files, req, assign)
	if failed != nil {
		writeMapProblem(w, http.StatusUnprocessableEntity, failed)
		return
	}
	if err != nil {
		log.Printf("world map restate %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, "没能存进项目,项目没有改动")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func restate(files *landFileSet, req restateRequest, assign map[int]string) (map[string]any, *mapProblem, error) {
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

	regionOf := map[int]int{}
	for _, row := range payload.Provinces {
		id, _ := row[0].(int)
		state, _ := row[6].(int)
		if region, _ := row[7].(int); region > 0 {
			regionOf[id] = region
		}
		if st, ok := states[state]; ok && state > 0 {
			st.Provinces = append(st.Provinces, id)
			states[state] = st
		}
	}
	base, err := files.base()
	if err != nil {
		return nil, nil, err
	}
	plan := mapgen.PlanRestate(states, base.Neighbours(), assign, regionOf, base.IsLand())
	if len(plan.Owner) == 0 && len(plan.Lists) == 0 && len(plan.New) == 0 {
		return map[string]any{"changed": 0}, nil, nil
	}
	painted := make([]int, 0, len(assign))
	for p := range assign {
		painted = append(painted, p)
	}
	sort.Ints(painted)
	return applyRestatePlan(files, base, plan, names, req, painted, "地图创作")
}

func applyRestatePlan(files *landFileSet, base *mapgen.Base, plan mapgen.RestatePlan, names map[int]string, req restateRequest, provinces []int, recordName string) (map[string]any, *mapProblem, error) {
	store := files.store
	projectID := files.p.ID()

	touched := map[int]bool{}
	for id := range plan.Owner {
		touched[id] = true
	}
	for id := range plan.Lists {
		touched[id] = true
	}
	for _, n := range plan.New {
		touched[n.From] = true
	}
	ids := make([]int, 0, len(touched))
	for id := range touched {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	have, err := store.WorldMapStates(ids)
	if err != nil {
		return nil, nil, err
	}
	seeded := map[int]bool{}
	var missing []int
	for _, id := range ids {
		if _, ok := have[id]; !ok {
			missing = append(missing, id)
			seeded[id] = true
		}
	}
	if len(missing) > 0 {
		if err := store.SeedWorldMapStates(missing); err != nil {
			return nil, nil, err
		}
	}
	rows, err := store.WorldMapStateRows(ids)
	if err != nil {
		return nil, nil, err
	}
	for _, id := range ids {
		if _, ok := rows[id]; !ok {
			return nil, newMapProblem(problemStateMissing, fmt.Sprintf("项目里没有地块 #%d", id)).with("id", id), nil
		}
	}

	points := map[int]WorldMapVictoryPoint{}
	blocks := map[int]string{}
	rests := map[int][]string{}
	share := map[int]int64{}
	for _, id := range ids {
		row := rows[id]
		for _, point := range row.VictoryPoints {
			points[point.ProvinceID] = point
		}
		byProvince, rest := provinceBlocks(row.BuildingsText)
		for p, text := range byProvince {
			blocks[p] = text
		}
		rests[id] = rest
		if n := int64(len(row.Provinces)); n > 0 {
			for _, p := range row.Provinces {
				share[p] = row.Manpower / n
			}
		}
	}
	carry := func(provinces []int) ([]WorldMapVictoryPoint, string, int64) {
		var vps []WorldMapVictoryPoint
		var texts []string
		var manpower int64
		for _, p := range provinces {
			if point, ok := points[p]; ok {
				vps = append(vps, point)
			}
			if text, ok := blocks[p]; ok {
				texts = append(texts, text)
			}
			manpower += share[p]
		}
		return vps, strings.Join(texts, "\n"), manpower
	}

	updates := map[int]WorldMapStateUpdate{}
	before := map[int]StateBefore{}
	for _, id := range ids {
		row := rows[id]
		before[id] = StateBefore{Provinces: row.Provinces, Seeded: seeded[id], Row: row.Row}
		provinces := row.Provinces
		if list, ok := plan.Lists[id]; ok {
			provinces = list
		}
		owner := row.Owner
		cores := append([]string(nil), row.Cores...)
		var extrasSet map[string]any
		var extrasDrop []string
		if tag, ok := plan.Owner[id]; ok {
			owner = tag
			if req.AddCores && !containsTag(cores, tag) {
				cores = append(cores, tag)
			}

			extrasSet, extrasDrop = worldout.StartTransfersLeftOut(row.Extras)
		}
		vps, text, manpower := carry(provinces)
		if _, ok := plan.Lists[id]; !ok {

			vps, text, manpower = row.VictoryPoints, row.BuildingsText, row.Manpower
		} else if rest := rests[id]; len(rest) > 0 {
			text = strings.TrimSpace(strings.Join(append([]string{text}, rest...), "\n"))
		}
		updates[id] = WorldMapStateUpdate{Provinces: provinces, Owner: owner, Cores: cores, VictoryPoints: vps, BuildingsText: text, Manpower: manpower,
			ExtrasSet: extrasSet, ExtrasDrop: extrasDrop}
	}

	newRows := make([]WorldMapStateRow, len(plan.New))
	numbered := map[string]int{}
	for _, name := range names {
		if stem, n := mapgen.RestateStem(name); n > numbered[stem] {
			numbered[stem] = n
		}
	}
	for k, n := range plan.New {
		parent := rows[n.From]
		stem, _ := mapgen.RestateStem(names[n.From])
		number := max(numbered[stem], 1) + 1
		numbered[stem] = number
		locs := map[string]string{}
		for _, lang := range req.Languages {
			name := parent.Locs[lang]
			if name == "" {
				name = names[n.From]
			}
			own, _ := mapgen.RestateStem(name)
			locs[lang] = mapgen.RestateName(own, number)
		}
		vps, text, manpower := carry(n.Provinces)
		var cores []string
		for _, tag := range parent.Cores {
			if tag != n.Owner {
				cores = append(cores, tag)
			}
		}
		buildings := map[string]any{}
		if infrastructure, ok := parent.Buildings["infrastructure"]; ok {
			buildings["infrastructure"] = infrastructure
		}
		category := parent.Category
		if category == "" {
			category = "rural"
		}
		newRows[k] = WorldMapStateRow{
			Provinces: n.Provinces, Manpower: manpower, Category: category, Owner: n.Owner, Buildings: buildings,
			Locs: locs, Cores: cores, ProvinceBuildings: text, Points: vps,
		}
		if !req.AddCores {

			newRows[k].Cores = parent.Cores
		}
	}
	var newIDs []int
	if len(newRows) > 0 {
		if newIDs, err = store.CreateWorldMapStates(newRows); err != nil {
			return nil, nil, err
		}
	}
	undoStates := func() {
		if len(newIDs) == 0 {
			return
		}
		if derr := store.DeleteWorldMapStates(newIDs); derr != nil {
			log.Printf("world map restate %s: states %v left after a failed save: %v", files.p.Dir(), newIDs, derr)
		}
	}

	stateOf := map[int]int{}
	ensure := map[int][]int{}
	capitalFirst := func(provinces []int) []int {
		best, bestValue := provinces[0], -1.0
		for _, p := range provinces {
			if point, ok := points[p]; ok {
				if v, err := strconv.ParseFloat(strings.TrimSpace(string(point.Value)), 64); err == nil && v > bestValue {
					best, bestValue = p, v
				}
			}
		}
		out := []int{best}
		for _, p := range provinces {
			if p != best {
				out = append(out, p)
			}
		}
		return out
	}
	for id, list := range plan.Lists {
		ensure[id] = capitalFirst(list)
		for _, p := range list {
			stateOf[p] = id
		}
	}
	for k, n := range plan.New {
		ensure[newIDs[k]] = capitalFirst(n.Provinces)
		for _, p := range n.Provinces {
			stateOf[p] = newIDs[k]
		}
	}
	var inserts []RawFileInsert
	var rawPaths []string
	renamed, added := 0, 0
	if len(ensure) > 0 {
		text, err := files.file("map/buildings.txt")
		if err != nil {
			undoStates()
			return nil, nil, err
		}
		next, r1, r2 := base.RestateBuildings(string(text), stateOf, ensure, projectID)
		renamed, added = r1, r2
		inserts = append(inserts, RawFileInsert{DestPath: "map/buildings.txt", Content: next, RawKind: "verbatim_override", Note: "世界地图 · 地图创作改了州"})
		rawPaths = append(rawPaths, "map/buildings.txt")
	}

	summary, _ := json.Marshal(map[string]any{"owners": len(plan.Owner), "lists": len(plan.Lists), "new_states": newIDs, "renamed": renamed, "added": added})
	record, err := store.SaveWorldMapRestate(inserts, updates, WorldMapAddition{
		Kind: "edit", Name: recordName, Provinces: provinces, StateIDs: newIDs, RawPaths: rawPaths, StateBefore: before, Summary: summary,
	})
	if err != nil {
		undoStates()
		return nil, nil, err
	}
	files.saved()

	files.remember(inserts)
	return map[string]any{
		"changed": len(updates) + len(newIDs), "record": record, "owners": len(plan.Owner),
		"cut": len(plan.Lists), "new_states": newIDs, "buildings_renamed": renamed, "buildings_added": added,
	}, nil, nil
}

func provinceBlocks(text string) (map[int]string, []string) {
	out := map[int]string{}
	var rest []string
	i, n := 0, len(text)
	skipSpace := func() {
		for i < n && (text[i] == ' ' || text[i] == '\t' || text[i] == '\r' || text[i] == '\n') {
			i++
		}
	}
	for {
		skipSpace()
		if i >= n {
			break
		}
		start := i

		for i < n && text[i] == '#' {
			for i < n && text[i] != '\n' {
				i++
			}
			skipSpace()
		}
		keyStart := i
		for i < n && text[i] != '=' && text[i] != '{' && text[i] != '\n' {
			i++
		}
		key := strings.TrimSpace(text[keyStart:i])
		if i < n && text[i] == '=' {
			i++
		}
		skipSpace()
		if i >= n || text[i] != '{' {

			for i < n && text[i] != '\n' {
				i++
			}
			rest = append(rest, strings.TrimSpace(text[start:i]))
			continue
		}
		depth := 0
		for i < n {
			switch text[i] {
			case '"':
				i++
				for i < n && text[i] != '"' {
					i++
				}
			case '#':
				for i < n && text[i] != '\n' {
					i++
				}
				continue
			case '{':
				depth++
			case '}':
				depth--
			}
			i++
			if depth == 0 {
				break
			}
		}
		entry := strings.TrimSpace(text[start:i])
		if id, err := strconv.Atoi(key); err == nil && id > 0 {
			out[id] = entry
		} else {
			rest = append(rest, entry)
		}
	}
	return out, rest
}
