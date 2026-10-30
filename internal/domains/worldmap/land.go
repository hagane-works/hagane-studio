package worldmap

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "image/jpeg"

	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/project"
)

const (
	landMaxPlacePixels = 2048 * 1024
	landSketchMaxBytes = 12 << 20
)

var landTagPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{2}$`)

func (h *handlers) landRoute(preview bool) func(http.ResponseWriter, *http.Request, *project.Project) {
	return func(w http.ResponseWriter, r *http.Request, p *project.Project) { h.land(w, r, p, preview) }
}

func (h *handlers) land(w http.ResponseWriter, r *http.Request, p *project.Project, preview bool) {
	store := h.store(p)
	if r.Method == http.MethodGet {
		additions, err := store.WorldMapAdditions()
		if err != nil {
			log.Printf("world map land %s: %v", p.Dir(), err)
			refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到新增土地的记录")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"additions": additions, "map_changes": mapChangesOf(additions)})
		return
	}
	if r.Method == http.MethodDelete {
		if r.URL.Query().Get("latest") != "" {
			h.undoLatest(w, p, store)
			return
		}
		if confirmation := r.URL.Query().Get("confirm"); confirmation != "" {
			h.confirmedRestore(w, p, store, confirmation)
			return
		}
		h.restore(w, p, store)
		return
	}
	select {
	case landGate <- struct{}{}:
		defer func() { <-landGate }()
	default:
		refuseMap(w, http.StatusTooManyRequests, problemBusy, "另一块新土地正在生成,稍等几秒再试")
		return
	}
	req, problem := parseLandRequest(w, r)
	if problem != "" {
		writeMapProblem(w, http.StatusUnprocessableEntity, requestProblem(problem))
		return
	}
	files, err := h.landFiles(r.Context(), p)
	if err != nil {
		var refused *landRefusal
		if errors.As(err, &refused) {
			refuseMap(w, http.StatusConflict, problemOwnMap, refused.message)
			return
		}
		log.Printf("world map land %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件(原版地图还没放到服务器上,或项目的地图文件读取失败)")
		return
	}
	started := time.Now()
	base, err := files.base()
	if err != nil {
		log.Printf("world map land %s base: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "地图文件读不出来")
		return
	}
	before := files.findingCounts(base)
	add, err := base.AddLand(req.sketch, req.options)
	var addErr *mapgen.AddError
	if errors.As(err, &addErr) {
		writeMapProblem(w, http.StatusUnprocessableEntity, mapProblemOf(addErr))
		return
	}
	if err != nil {
		log.Printf("world map land %s add: %v", p.Dir(), err)
		refuseMap(w, http.StatusInternalServerError, problemNotSaved, "生成失败")
		return
	}
	findings := newFindings(mapgen.Check(base.CheckInput(nil)), before)
	summary := landSummary(add, time.Since(started))

	seaMoves, err := landSeaRegionMoves(files, base, add, files.index.MaxStrategicRegionID()+1)
	var refusedRegion *landRefusal
	if errors.As(err, &refusedRegion) {
		refuseMap(w, http.StatusConflict, problemOwnMap, refusedRegion.message)
		return
	}
	if problem := mapProblemOf(err); problem != nil {
		writeMapProblem(w, http.StatusUnprocessableEntity, problem)
		return
	}
	if err != nil {
		log.Printf("world map land %s regions: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "地图文件读不出来")
		return
	}
	if len(seaMoves) > 0 {
		summary["sea_region_moves"] = seaMoves
	}
	if preview {
		writeJSON(w, http.StatusOK, map[string]any{
			"summary": summary, "findings": findings,
			"overlay": landOverlay(base, add, req.options.Place),
		})
		return
	}
	if len(mapgen.Errors(findings)) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"code": problemLandBreaksMap, "detail": "新土地会让地图出错,没有保存", "findings": findings})
		return
	}
	if req.owner != "" && !slices.Contains(files.index.ListCountryTags(), req.owner) {
		writeMapProblem(w, http.StatusUnprocessableEntity, newMapProblem(problemNoCountry, "没有国家 "+req.owner+":先在国家工作台新建它,或留空").with("tag", req.owner))
		return
	}
	saved, err := h.saveLand(files, base, add, seaMoves, req, summary)
	var refused *landRefusal
	if errors.As(err, &refused) {
		refuseMap(w, http.StatusConflict, problemOwnMap, refused.message)
		return
	}
	if err != nil {
		log.Printf("world map land %s save: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, "新土地没能存进项目,项目的地图没有改动")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"addition": saved, "summary": summary, "findings": findings})
}

type landRequest struct {
	sketch    image.Image
	options   mapgen.AddOptions
	name      string
	owner     string
	languages []string
}

func parseLandRequest(w http.ResponseWriter, r *http.Request) (landRequest, string) {
	r.Body = http.MaxBytesReader(w, r.Body, landSketchMaxBytes+(1<<20))
	if err := r.ParseMultipartForm(landSketchMaxBytes); err != nil {
		return landRequest{}, "草图要是 12 MB 以内的 PNG 或 JPEG"
	}
	file, _, err := r.FormFile("sketch")
	if err != nil {
		return landRequest{}, "没有收到草图"
	}
	sketch, _, err := image.Decode(file)
	_ = file.Close()
	if err != nil {
		return landRequest{}, "草图读不出来,要是 PNG 或 JPEG"
	}
	number := func(name string) int {
		v, _ := strconv.Atoi(strings.TrimSpace(r.FormValue(name)))
		return v
	}
	place := image.Rect(number("x"), number("y"), number("x")+number("width"), number("y")+number("height"))
	if place.Dx()*place.Dy() > landMaxPlacePixels {
		return landRequest{}, "新土地的范围太大了(最多约 2048×1024 像素),缩小一些"
	}
	req := landRequest{
		sketch: sketch,
		options: mapgen.AddOptions{
			Place: place, LandProvincePixels: number("land"),
		},
		name:  strings.TrimSpace(r.FormValue("name")),
		owner: strings.ToUpper(strings.TrimSpace(r.FormValue("owner"))),
	}
	if seed, err := strconv.ParseInt(r.FormValue("seed"), 10, 64); err == nil {
		req.options.Seed = seed
	}
	if req.owner != "" && !landTagPattern.MatchString(req.owner) {
		return landRequest{}, "拥有者要是三个字母的国家标签,比如 GER"
	}
	if len([]rune(req.name)) > 60 {
		return landRequest{}, "名字最长 60 个字"
	}
	for _, lang := range strings.Split(r.FormValue("languages"), ",") {
		if lang = strings.ToLower(strings.TrimSpace(lang)); lang != "" {
			if _, _, ok := GUITextTarget(lang); !ok {
				return landRequest{}, "不认识的语言 " + lang
			}
			req.languages = append(req.languages, lang)
		}
	}
	if len(req.languages) == 0 {
		req.languages = []string{"en"}
	}
	return req, ""
}

func landSummary(add *mapgen.Addition, took time.Duration) map[string]any {
	counts := map[string]int{"land": 0, "sea": 0, "lake": 0}
	for _, pr := range add.Provinces {
		counts[pr.Kind]++
	}
	covered := make([]int, 0, len(add.Covered))
	for id := range add.Covered {
		covered = append(covered, id)
	}
	sort.Ints(covered)
	return map[string]any{
		"land": counts["land"], "sea": counts["sea"], "lake": counts["lake"],
		"states": len(add.States), "covered": covered, "ignored": add.Ignored,
		"first_province": firstNewProvince(add), "seconds": took.Seconds(),
	}
}

func firstNewProvince(add *mapgen.Addition) int {
	if len(add.Provinces) == 0 {
		return 0
	}
	return add.Provinces[0].ID
}

func landOverlay(base *mapgen.Base, add *mapgen.Addition, place image.Rectangle) map[string]any {
	rect := add.Changed.Union(place).Intersect(image.Rect(0, 0, base.Width, base.Height))
	provinces := image.NewNRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	terrain := image.NewNRGBA(provinces.Rect)
	legend := map[uint8]color.NRGBA{}
	for _, entry := range mapgen.Legend {
		legend[entry.Terrain] = color.NRGBA{entry.Color[0], entry.Color[1], entry.Color[2], 255}
	}
	for _, pr := range add.Provinces {
		c := color.NRGBA{pr.Color[0], pr.Color[1], pr.Color[2], 255}
		for _, p := range add.Pixels(pr.ID) {
			x, y := int(p)%base.Width-rect.Min.X, int(p)/base.Width-rect.Min.Y
			if x < 0 || y < 0 || x >= rect.Dx() || y >= rect.Dy() {
				continue
			}
			provinces.SetNRGBA(x, y, c)
			if pr.Kind == "land" {
				terrain.SetNRGBA(x, y, legend[base.Terrain[p]])
			} else {
				terrain.SetNRGBA(x, y, legend[mapgen.Legend[mapgen.Lake].Terrain])
			}
		}
	}
	encode := func(img image.Image) string {
		var out bytes.Buffer
		_ = (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&out, img)
		return "data:image/png;base64," + base64.StdEncoding.EncodeToString(out.Bytes())
	}
	return map[string]any{
		"rect":      [4]int{rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy()},
		"provinces": encode(provinces),
		"terrain":   encode(terrain),
	}
}

type blobInserts struct {
	files   *landFileSet
	note    string
	inserts []RawFileInsert
	shaOf   map[string]string

	replace func(dest string) bool
}

func (b *blobInserts) binary(dest string, data []byte) bool {
	sha, key, ok := b.files.storeBlob(data)
	if !ok {
		return false
	}
	if b.shaOf == nil {
		b.shaOf = map[string]string{}
	}
	b.shaOf[dest] = sha
	b.inserts = append(b.inserts, RawFileInsert{DestPath: dest, IsBinary: true, RawKind: "verbatim_override", Note: b.note,
		GenerateReplacePath: b.replace != nil && b.replace(dest), ContentSHA256: &sha, ContentOBSKey: &key})
	return true
}

func (b *blobInserts) texts(texts map[string]string) error {
	paths := make([]string, 0, len(texts))
	for dest := range texts {
		paths = append(paths, dest)
	}
	sort.Strings(paths)
	for _, dest := range paths {
		text := texts[dest]
		if len(text) > landTextFileMax {
			if !b.binary(dest, []byte(text)) {
				return fmt.Errorf("%s not stored", dest)
			}
			continue
		}
		b.inserts = append(b.inserts, RawFileInsert{DestPath: dest, Content: text, RawKind: "verbatim_override", Note: b.note,
			GenerateReplacePath: b.replace != nil && b.replace(dest)})
	}
	return nil
}

func (b *blobInserts) paths() []string {
	out := make([]string, 0, len(b.inserts))
	for _, insert := range b.inserts {
		out = append(out, insert.DestPath)
	}
	return out
}

func (b *blobInserts) textures(what string, textures []texturePaint) error {
	for _, texture := range textures {
		data, err := b.files.file(texture.path)
		if err == nil {
			data, err = texture.paint(data)
		}
		if err != nil {
			log.Printf("world map %s %s: %s left as it was: %v", what, b.files.p.Dir(), texture.path, err)
			continue
		}
		if !b.binary(texture.path, data) {
			return fmt.Errorf("%s not stored", texture.path)
		}
	}
	return nil
}

type texturePaint struct {
	path  string
	paint func([]byte) ([]byte, error)
}

func (h *handlers) saveLand(files *landFileSet, base *mapgen.Base, add *mapgen.Addition, seaMoves []mapgen.RegionMove, req landRequest, summary map[string]any) (WorldMapAddition, error) {
	store := files.store
	name := req.name
	if name == "" {
		name = "New Land"
	}
	rows := make([]WorldMapStateRow, len(add.States))
	var pointNames [][3]string
	for k, st := range add.States {
		stateName := name
		if len(add.States) > 1 {
			stateName = fmt.Sprintf("%s %d", name, k+1)
		}
		locs := map[string]string{}
		for _, lang := range req.languages {
			locs[lang] = stateName
		}

		rows[k] = WorldMapStateRow{
			Provinces: st.Provinces, Manpower: int64(20000 * len(st.Provinces)), Category: "rural",
			Owner: req.owner, Buildings: map[string]any{"infrastructure": 1}, Locs: locs,
			VictoryPoints: map[int]int{st.Capital: 1},
		}
		for _, lang := range req.languages {
			pointNames = append(pointNames, [3]string{lang, fmt.Sprintf("VICTORY_POINTS_%d", st.Capital), stateName})
		}
	}
	stateIDs, err := store.CreateWorldMapStates(rows)
	if err != nil {
		return WorldMapAddition{}, err
	}
	undoStates := func() {
		if derr := store.DeleteWorldMapStates(stateIDs); derr != nil {
			log.Printf("world map land %s: states %v left after a failed save: %v", files.p.Dir(), stateIDs, derr)
		}
	}

	regionID := files.index.MaxStrategicRegionID() + 1
	pos := base.Positions(add, stateIDs, regionID, req.options.Seed)

	textFiles := map[string]string{}
	most, mostPixels := 0, -1
	for id, n := range add.Covered {
		if n > mostPixels || (n == mostPixels && id < most) {
			most, mostPixels = id, n
		}
	}
	weather := ""
	if _, file, ok := files.index.StrategicRegionOf(most); ok {
		if body, ok := files.index.Source(file); ok {
			weather = mapgen.WeatherBlock(string(body))
		}
	}
	var regionProvinces []int
	cutInto := map[string][]int{}
	for _, pr := range add.Provinces {
		if pr.Kind == "sea" {
			if _, file, ok := files.index.StrategicRegionOf(add.CutFrom[pr.ID]); ok {
				cutInto[file] = append(cutInto[file], pr.ID)
				continue
			}
		}
		regionProvinces = append(regionProvinces, pr.ID)
	}
	regionFile := fmt.Sprintf("map/strategicregions/%d-%s.txt", regionID, safeFileStem(name))
	textFiles[regionFile] = mapgen.RegionFile(regionID, regionProvinces, weather)
	for file, ids := range cutInto {

		body, refusal, err := files.regionText(file)
		if refusal != "" {
			undoStates()
			return WorldMapAddition{}, &landRefusal{refusal}
		}
		if err != nil {
			undoStates()
			return WorldMapAddition{}, err
		}
		textFiles[file] = mapgen.AddToRegion(body, ids)
	}

	if err := applySeaRegionMoves(files, seaMoves, textFiles, func(id int) (string, bool) {
		if id == regionID {
			return regionFile, true
		}
		return files.index.StrategicRegionFile(id)
	}); err != nil {
		undoStates()
		return WorldMapAddition{}, err
	}

	for _, entry := range []struct {
		name  string
		lines []string
		move  int
	}{
		{"map/buildings.txt", pos.Buildings, 1},
		{"map/unitstacks.txt", pos.UnitStacks, 2},
		{"map/supply_nodes.txt", pos.SupplyNodes, 0},
		{"map/weatherpositions.txt", pos.Weather, 0},
	} {
		body, err := files.file(entry.name)
		if err != nil {
			undoStates()
			return WorldMapAddition{}, fmt.Errorf("%s: %w", entry.name, err)
		}
		text := string(body)
		if entry.move > 0 {
			text, _ = base.MoveCovered(text, add, entry.move == 2)
		}
		textFiles[entry.name] = mapgen.AppendLines(text, entry.lines)
	}
	textFiles["map/definition.csv"] = string(base.DefinitionCSV())

	out := &blobInserts{files: files, note: "世界地图 · 新增土地(" + name + ")"}
	images := map[string]image.Image{
		"provinces.bmp": base.ProvincesImage(), "heightmap.bmp": base.HeightImage(),
		"terrain.bmp": base.TerrainImage(), "rivers.bmp": base.RiversImage(),
	}
	for _, file := range landImageFiles {
		data, err := mapgen.EncodeBMP(images[file])
		if err != nil || !out.binary("map/"+file, data) {
			undoStates()
			return WorldMapAddition{}, fmt.Errorf("map/%s not stored: %v", file, err)
		}
	}

	if err := out.textures("land", []texturePaint{
		{landColormap, func(data []byte) ([]byte, error) { return base.PaintColormap(data, add, req.options.Seed) }},
		{"map/world_normal.bmp", func(data []byte) ([]byte, error) { return base.PaintNormals(data, add) }},
		{"map/trees.bmp", func(data []byte) ([]byte, error) { return base.PaintTrees(data, add, req.options.Seed) }},
	}); err != nil {
		undoStates()
		return WorldMapAddition{}, err
	}
	if err := out.texts(textFiles); err != nil {
		undoStates()
		return WorldMapAddition{}, err
	}
	rawPaths := out.paths()

	regionKey := fmt.Sprintf("STRATEGICREGION_%d", regionID)
	var textKeys []string
	for _, lang := range req.languages {
		textKeys = append(textKeys, lang+":"+regionKey)
	}
	for _, point := range pointNames {
		textKeys = append(textKeys, point[0]+":"+point[1])
	}
	provinces := make([]int, 0, len(add.Provinces))
	for _, pr := range add.Provinces {
		provinces = append(provinces, pr.ID)
	}
	optionsJSON, _ := json.Marshal(map[string]any{"land": req.options.LandProvincePixels})
	summaryJSON, _ := json.Marshal(summary)
	record := WorldMapAddition{
		Kind:      "land",
		Placement: [4]int{req.options.Place.Min.X, req.options.Place.Min.Y, req.options.Place.Dx(), req.options.Place.Dy()},
		Seed:      req.options.Seed, Options: optionsJSON, Name: name, Owner: req.owner,
		Provinces: provinces, StateIDs: stateIDs, RegionID: regionID, RawPaths: rawPaths,
		TextKeys: textKeys, Summary: summaryJSON,
	}
	id, err := store.SaveWorldMapAddition(out.inserts, nil, record)
	if err != nil {
		undoStates()
		return WorldMapAddition{}, err
	}
	record.ID = id
	for _, lang := range req.languages {
		if err := store.SetGUITextOverrideIn(lang, regionKey, name); err != nil {
			log.Printf("world map land %s: region name %s: %v", files.p.Dir(), lang, err)
		}
	}
	for _, point := range pointNames {
		if err := store.SetGUITextOverrideIn(point[0], point[1], point[2]); err != nil {
			log.Printf("world map land %s: %s %s: %v", files.p.Dir(), point[1], point[0], err)
		}
	}

	var picture bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&picture, images["provinces.bmp"]); err == nil {
		projectMapImages.put(files.p.Dir(), out.shaOf["map/provinces.bmp"], picture.Bytes())
	} else {
		projectMapImages.forget(files.p.Dir())
	}
	files.saved()
	return record, nil
}

func (h *handlers) undoLatest(w http.ResponseWriter, p *project.Project, store *mapStore) {
	record, found, err := store.UndoLatestWorldMap()
	if err != nil {
		log.Printf("world map undo %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUndoFailed, "没能撤销,项目没有改动")
		return
	}
	if !found {
		refuseMap(w, http.StatusNotFound, problemNothingToUndo, "没有可以撤销的地图改动")
		return
	}
	projectMapImages.forget(p.Dir())
	h.svc.Forget(p)
	writeJSON(w, http.StatusOK, map[string]any{"undone": map[string]any{"id": record.ID, "kind": record.Kind, "name": record.Name}})
}

func (h *handlers) restore(w http.ResponseWriter, p *project.Project, store *mapStore) {
	additions, err := store.RestoreWorldMap()
	if err != nil {
		log.Printf("world map restore %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUndoFailed, "没能恢复原版地图,项目没有改动")
		return
	}
	projectMapImages.forget(p.Dir())
	h.svc.Forget(p)
	removed := []worldMapChange{}
	if changes := mapChangesOf(additions); changes != nil {
		removed = changes.Records
	}
	writeJSON(w, http.StatusOK, map[string]any{"restored": len(additions), "removed": removed})
}

var unsafeFileRune = regexp.MustCompile(`[^A-Za-z0-9 _-]+`)

func safeFileStem(name string) string {
	stem := strings.TrimSpace(unsafeFileRune.ReplaceAllString(name, ""))
	if stem == "" {
		return "New Land"
	}
	return stem
}
