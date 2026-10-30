package worldmap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/emit/locout"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/mapgen"
)

const (
	modIndexField   = "mod_index"
	modBundleField  = "mod_bundle"
	modFilePrefix   = "file:"
	modFileMaxBytes = 4 << 20
	modFilesMax     = 6000
	modTextMaxBytes = 48 << 20
)

type modBundleEntry struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

var modFileFolders = []struct{ prefix, ext string }{
	{"map/strategicregions/", ".txt"}, {"history/states/", ".txt"}, {"localisation/", ".yml"},
}

type modWorld struct {
	facts map[int]guirender.StateFact

	stateNames map[string]map[int]string

	names map[string]map[string]string

	landedLakeStates int
}

func landLakeStates(b *mapgen.Base, states []mapgen.WorldPartState, regions []mapgen.Region) int {
	parts := make(map[int]mapgen.RestateState, len(states))
	for _, st := range states {
		parts[st.ID] = mapgen.RestateState{Provinces: st.Provinces}
	}
	regionOf := map[int]int{}
	for _, region := range regions {
		for _, p := range region.Provinces {
			regionOf[p] = region.ID
		}
	}
	land := b.IsLand()
	hasLand := func(provinces []int) bool {
		for _, p := range provinces {
			if land(p) {
				return true
			}
		}
		return false
	}
	plan := b.LandForLakeStates(parts, regionOf)
	landed := 0
	for i, st := range states {
		list, ok := plan.Lists[st.ID]
		if !ok {
			continue
		}
		if !hasLand(st.Provinces) && hasLand(list) {
			landed++
		}
		states[i].Provinces = list
	}
	return landed
}

func readModBundle(r *http.Request) (map[string][]byte, *mapProblem) {
	raw := strings.TrimSpace(r.FormValue(modIndexField))
	_, indexPresent := r.Form[modIndexField]
	_, indexFilePresent := r.MultipartForm.File[modIndexField]
	_, bundlePresent := r.MultipartForm.File[modBundleField]
	_, bundleValuePresent := r.Form[modBundleField]
	newProtocol := indexPresent || indexFilePresent || bundlePresent || bundleValuePresent
	for field := range r.MultipartForm.File {
		if strings.HasPrefix(field, modFilePrefix) && newProtocol {
			return nil, newMapProblem(problemBadRequest, "MOD 文件不能同时使用旧文件分项和新文件包")
		}
	}

	if !newProtocol {
		return readLegacyModFiles(r)
	}
	var index []modBundleEntry
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &index); err != nil {
			return nil, newMapProblem(problemBadRequest, "MOD 文件清单读不出来")
		}
	}
	headers := r.MultipartForm.File[modBundleField]
	if len(index) == 0 || len(headers) == 0 {
		return nil, newMapProblem(problemModNeedsStatesRegions, "没有收到 MOD 的文件")
	}
	total, problem := validateModEntries(index)
	if problem != nil {
		return nil, problem
	}
	if headers[0].Size != total {
		return nil, newMapProblem(problemBadRequest, "MOD 文件包和清单对不上")
	}
	bundle, err := headers[0].Open()
	if err != nil {
		return nil, newMapProblem(problemModFileUnreadable, "MOD 文件读不出来:"+modBundleField).with("path", modBundleField)
	}
	defer bundle.Close()
	files := make(map[string][]byte, len(index))
	for _, entry := range index {
		body := make([]byte, entry.Size)
		if _, err := io.ReadFull(bundle, body); err != nil {
			return nil, newMapProblem(problemModFileUnreadable, "MOD 文件读不出来:"+entry.Path).with("path", entry.Path)
		}
		files[path.Clean(strings.ReplaceAll(entry.Path, "\\", "/"))] = body
	}
	return files, nil
}

func validateModEntries(index []modBundleEntry) (int64, *mapProblem) {
	if len(index) > modFilesMax {
		return 0, newMapProblem(problemModTooManyFiles, fmt.Sprintf("MOD 文件太多:最多 %d 个", modFilesMax)).with("max", modFilesMax)
	}
	var total int64
	for _, entry := range index {
		clean := path.Clean(strings.ReplaceAll(entry.Path, "\\", "/"))
		allowed := false
		for _, folder := range modFileFolders {
			allowed = allowed || (strings.HasPrefix(strings.ToLower(clean), folder.prefix) && strings.HasSuffix(strings.ToLower(clean), folder.ext))
		}
		if !allowed || strings.Contains(clean, "..") {
			return 0, newMapProblem(problemModFileOutside, "MOD 文件只收 map/strategicregions、history/states 和 localisation 下的文本:"+entry.Path).with("path", entry.Path)
		}
		if entry.Size < 0 || entry.Size > modFileMaxBytes {
			return 0, newMapProblem(problemModFileUnreadable, "MOD 文件太大或读不出来:"+entry.Path).with("path", entry.Path)
		}
		if total += entry.Size; total > modTextMaxBytes {
			return 0, newMapProblem(problemModTextTooLarge, "MOD 的文本文件合计太大")
		}
	}
	return total, nil
}

func readLegacyModFiles(r *http.Request) (map[string][]byte, *mapProblem) {
	var index []modBundleEntry
	for field, headers := range r.MultipartForm.File {
		if strings.HasPrefix(field, modFilePrefix) && len(headers) > 0 {
			index = append(index, modBundleEntry{Path: strings.TrimPrefix(field, modFilePrefix), Size: headers[0].Size})
		}
	}
	if _, problem := validateModEntries(index); problem != nil {
		return nil, problem
	}
	files := make(map[string][]byte, len(index))
	for _, entry := range index {
		file, err := r.MultipartForm.File[modFilePrefix+entry.Path][0].Open()
		if err != nil {
			return nil, newMapProblem(problemModFileUnreadable, "MOD 文件读不出来:"+entry.Path).with("path", entry.Path)
		}
		body, err := io.ReadAll(io.LimitReader(file, modFileMaxBytes+1))
		_ = file.Close()
		if err != nil || int64(len(body)) != entry.Size {
			return nil, newMapProblem(problemModFileUnreadable, "MOD 文件读不出来:"+entry.Path).with("path", entry.Path)
		}
		files[path.Clean(strings.ReplaceAll(entry.Path, "\\", "/"))] = body
	}
	return files, nil
}

func parseWorldMod(r *http.Request, req *worldRequest) *mapProblem {
	if r.MultipartForm == nil {
		return newMapProblem(problemModNeedsStatesRegions, "没有收到 MOD 的文件")
	}
	files, problem := readModBundle(r)
	if problem != nil {
		return problem
	}
	req.modFiles = files
	states, regions := 0, 0
	for name := range req.modFiles {
		switch {
		case strings.HasPrefix(strings.ToLower(name), "history/states/"):
			states++
		case strings.HasPrefix(strings.ToLower(name), "map/strategicregions/"):
			regions++
		}
	}
	if states == 0 || regions == 0 {
		return newMapProblem(problemModNeedsStatesRegions, "原样导入要有 MOD 的 history/states(州)和 map/strategicregions(战略区)文件")
	}
	if len(req.upload.Definitions) == 0 {
		return newMapProblem(problemModNeedsDefinition, "原样导入要有 MOD 的 map/definition.csv:州是按它的省份编号写的")
	}
	req.tagMap = map[string]string{}
	for _, pair := range strings.Split(r.FormValue("tag_map"), ",") {
		from, to, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if !ok {
			continue
		}
		from, to = strings.ToUpper(strings.TrimSpace(from)), strings.ToUpper(strings.TrimSpace(to))
		if !landTagPattern.MatchString(from) || !landTagPattern.MatchString(to) {
			return newMapProblem(problemBadRequest, "国家对应要写成 MOD标签=项目标签,比如 KOR=SHU")
		}
		req.tagMap[from] = to
	}
	req.date = strings.TrimSpace(r.FormValue("date"))
	if req.date == "" {
		req.date = "1936.1.1"
	}
	return nil
}

func buildModWorld(b *mapgen.Base, req worldRequest) (*mapgen.World, *modWorld, *mapProblem, error) {
	files := map[string][]byte{"map/definition.csv": b.DefinitionCSV()}
	for name, body := range req.modFiles {
		files[name] = body
	}
	idx, err := guirender.NewIndex(files)
	if err != nil {
		return nil, nil, nil, err
	}
	payload := idx.WorldMap(req.date)
	land := map[int]bool{}
	members := map[int][]int{}
	for _, row := range payload.Provinces {
		id, _ := row[0].(int)
		if kind, _ := row[2].(int); kind == 0 {
			land[id] = true
		}
		if region, _ := row[7].(int); region > 0 {
			members[region] = append(members[region], id)
		}
	}
	var regions []mapgen.Region
	for id, provinces := range members {
		weather := ""
		if file, ok := idx.StrategicRegionFile(id); ok {
			weather = mapgen.WeatherBlock(string(files[file]))
		}
		regions = append(regions, mapgen.Region{ID: id, Provinces: provinces, Weather: weather})
	}
	mod := &modWorld{facts: map[int]guirender.StateFact{}, stateNames: map[string]map[int]string{}, names: map[string]map[string]string{}}
	var states []mapgen.WorldPartState
	for _, fact := range idx.StateFacts(req.date) {
		mod.facts[fact.ID] = fact
		capital, best := 0, -1
		for _, p := range fact.Provinces {
			if v, ok := fact.VictoryPoints[p]; ok && land[p] && v > best {
				capital, best = p, v
			}
		}
		states = append(states, mapgen.WorldPartState{ID: fact.ID, Provinces: fact.Provinces, Capital: capital})
	}
	if req.fixes["state_no_land"] {
		mod.landedLakeStates = landLakeStates(b, states, regions)
	}
	world, err := mapgen.WorldOfParts(b, states, regions)
	if problem := mapProblemOf(err); problem != nil {
		return nil, nil, problem, nil
	}
	if err != nil {
		return nil, nil, nil, err
	}

	for _, lang := range req.languages {
		view := idx.InLanguage(lang, nil)
		hoi4 := locout.LangCodeToHOI4[lang]
		names := map[int]string{}
		for _, fact := range view.StateFacts(req.date) {
			if text, ok := view.Localise(fact.Name); ok && strings.TrimSpace(text) != "" {
				names[fact.ID] = text
			}
		}
		mod.stateNames[lang] = names
		if hoi4 == "" {
			continue
		}
		own := map[string]string{}
		wm := view.WorldMap(req.date)
		for _, row := range wm.Regions {
			id, _ := row[0].(int)
			name, _ := row[1].(string)
			if key := fmt.Sprintf("STRATEGICREGION_%d", id); name != "" && !strings.HasPrefix(name, "STRATEGICREGION_") {
				own[key] = name
			}
		}
		for _, row := range wm.Points {
			province, _ := row[0].(int)
			if name, _ := row[2].(string); strings.TrimSpace(name) != "" {
				own[fmt.Sprintf("VICTORY_POINTS_%d", province)] = name
			}
		}
		mod.names[hoi4] = own
	}
	return world, mod, nil, nil
}

type modUnknownTag struct {
	Tag    string `json:"tag"`
	States int    `json:"states"`
}

func modOwners(world *mapgen.World, mod *modWorld, req worldRequest, known []string) (map[int]string, map[string]int, []string, []modUnknownTag) {
	has := map[string]bool{}
	for _, tag := range known {
		has[strings.ToUpper(tag)] = true
	}
	owner := map[int]string{}
	unknown := map[string]int{}
	for k := range world.All.States {
		id := k + 1
		tag := strings.ToUpper(mod.facts[id].Owner)
		if tag == "" {
			continue
		}
		if mapped, ok := req.tagMap[tag]; ok {
			tag = mapped
		}
		if !has[tag] {
			unknown[tag]++
			continue
		}
		owner[id] = tag
	}
	if len(unknown) > 0 {
		out := make([]modUnknownTag, 0, len(unknown))
		for tag, n := range unknown {
			out = append(out, modUnknownTag{Tag: tag, States: n})
		}
		sort.Slice(out, func(i, j int) bool {
			return out[i].States > out[j].States || (out[i].States == out[j].States && out[i].Tag < out[j].Tag)
		})
		return nil, nil, nil, out
	}
	type score struct{ points, provinces, id int }
	best := map[string]score{}
	capital := map[string]int{}
	for id, tag := range owner {
		points := 0
		for _, v := range mod.facts[id].VictoryPoints {
			points += v
		}
		s := score{points, len(world.All.States[id-1].Provinces), id}
		b, seen := best[tag]
		if !seen || s.points > b.points || (s.points == b.points && (s.provinces > b.provinces || (s.provinces == b.provinces && s.id < b.id))) {
			best[tag], capital[tag] = s, id
		}
	}
	tags := make([]string, 0, len(capital))
	for tag := range capital {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return owner, capital, tags, nil
}

func modStateRows(world *mapgen.World, mod *modWorld, req worldRequest, owner map[int]string, known []string) []WorldMapStateRow {
	has := map[string]bool{}
	for _, tag := range known {
		has[strings.ToUpper(tag)] = true
	}
	rows := make([]WorldMapStateRow, len(world.All.States))
	for k, st := range world.All.States {
		id := k + 1
		fact := mod.facts[id]
		locs := map[string]string{}
		for _, lang := range req.languages {
			if name := mod.stateNames[lang][id]; name != "" {
				locs[lang] = name
			} else {
				locs[lang] = worldStateName(locout.LangCodeToHOI4[lang], id)
			}
		}
		points := map[int]int{}
		for _, p := range st.Provinces {
			if v, ok := fact.VictoryPoints[p]; ok {
				points[p] = v
			}
		}
		if len(points) == 0 {
			points[st.Capital] = 1
		}
		var cores []string
		for _, tag := range fact.Cores {
			if mapped, ok := req.tagMap[tag]; ok {
				tag = mapped
			}
			if has[tag] && tag != owner[id] {
				cores = append(cores, tag)
			}
		}
		buildings := map[string]any{}
		for name, level := range fact.Buildings {
			buildings[name] = level
		}
		if _, ok := buildings["infrastructure"]; !ok {
			buildings["infrastructure"] = 2
		}
		resources := map[string]any{}
		for name, amount := range fact.Resources {
			resources[name] = amount
		}
		manpower := int64(fact.Manpower)
		if manpower <= 0 {
			manpower = int64(200000 + 50000*len(st.Provinces))
		}
		category := fact.Category
		if category == "" {
			category = "town"
		}
		rows[k] = WorldMapStateRow{
			Provinces: st.Provinces, Manpower: manpower, Category: category, Owner: owner[id],
			Buildings: buildings, Locs: locs, VictoryPoints: points, Cores: cores, Resources: resources,
		}
	}
	return rows
}

func modNamedStates(mod *modWorld, req worldRequest, states int) int {
	named := 0
	for id := 1; id <= states; id++ {
		all := true
		for _, lang := range req.languages {
			all = all && mod.stateNames[lang][id] != ""
		}
		if all {
			named++
		}
	}
	return named
}
