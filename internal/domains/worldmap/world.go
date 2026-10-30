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
	"io"
	"log"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"haganeworks/internal/hoi4/emit/locout"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/hoi4/mapsketch"
	"haganeworks/internal/project"
)

const (
	worldMaxTags = 8

	worldLandMin, worldLandMax = 400, 60000

	worldUploadMaxBytes = 160 << 20

	worldOwnersMaxPixels = 4096 * 2048
)

type worldRequest struct {
	source    string
	sketch    image.Image
	upload    mapgen.UploadInput
	seed      int64
	land, sea int
	perState  int
	tags      []string
	languages []string

	owners           []uint8
	ownersW, ownersH int

	focus *[4]float64

	picturePlacement *mapsketch.Placement

	keepPreviews func(previews map[string]string) map[string]string

	modFiles map[string][]byte
	tagMap   map[string]string
	date     string

	fixes map[string]bool
}

var worldFixes = map[string]bool{"black_pixel": true, "undefined_color": true, "state_no_land": true}

func parseWorldRequest(w http.ResponseWriter, r *http.Request) (worldRequest, *mapProblem) {
	r.Body = http.MaxBytesReader(w, r.Body, worldUploadMaxBytes)
	if err := r.ParseMultipartForm(landSketchMaxBytes); err != nil {
		return worldRequest{}, newMapProblem(problemTooLarge, "收不下:草图要在 12 MB 以内,上传的地图图片合计 160 MB 以内")
	}
	number := func(name string) int {
		v, _ := strconv.Atoi(strings.TrimSpace(r.FormValue(name)))
		return v
	}
	req := worldRequest{source: strings.TrimSpace(r.FormValue("source")), land: number("land"), sea: number("sea"), perState: number("per_state")}
	if req.source == "" {
		req.source = "sketch"
	}
	if req.source != "sketch" && req.source != "game" && req.source != "upload" && req.source != "mod" {
		return worldRequest{}, newMapProblem(problemBadRequest, "世界从草图生成(sketch)、从原版地图出发(game)、上传做好的图(upload)或原样导入 MOD 的地图(mod)")
	}
	if seed, err := strconv.ParseInt(r.FormValue("seed"), 10, 64); err == nil {
		req.seed = seed
	}
	if req.source == "sketch" {
		file, _, err := r.FormFile("sketch")
		if err != nil {
			return worldRequest{}, newMapProblem(problemUploadMissing, "没有收到草图").with("file", "sketch")
		}
		req.sketch, _, err = image.Decode(file)
		_ = file.Close()
		if err != nil {
			return worldRequest{}, newMapProblem(problemSketchUnreadable, "草图读不出来,要是 PNG 或 JPEG")
		}
		if req.land == 0 {
			req.land = 2000
		}
		if req.land < worldLandMin || req.land > worldLandMax {
			return worldRequest{}, newMapProblem(problemWorldLandSize, fmt.Sprintf("省份大小要在 %d 到 %d 像素之间", worldLandMin, worldLandMax)).with("min", worldLandMin).with("max", worldLandMax)
		}
		if req.sea <= 0 {
			req.sea = req.land * 5
		}
	}
	req.fixes = map[string]bool{}
	for _, code := range strings.Split(r.FormValue("fix"), ",") {
		if code = strings.TrimSpace(code); worldFixes[code] {
			req.fixes[code] = true
		}
	}
	if req.source == "upload" || req.source == "mod" {
		if problem := parseWorldUpload(r, &req.upload); problem != nil {
			return worldRequest{}, problem
		}
		req.upload.FillStrayPixels = req.fixes["black_pixel"] || req.fixes["undefined_color"]
	}
	if req.source == "mod" {
		if problem := parseWorldMod(r, &req); problem != nil {
			return worldRequest{}, problem
		}
	}
	if req.perState < 0 || req.perState > 30 {
		return worldRequest{}, newMapProblem(problemWorldPerState, "每州省份数要在 1 到 30 之间").with("max", 30)
	}
	seen := map[string]bool{}
	for _, tag := range strings.Split(r.FormValue("tags"), ",") {
		tag = strings.ToUpper(strings.TrimSpace(tag))
		if tag == "" || seen[tag] {
			continue
		}
		if !landTagPattern.MatchString(tag) {
			return worldRequest{}, newMapProblem(problemTagInvalid, "国家要是三个字母的标签,比如 GER").with("tag", tag)
		}
		seen[tag] = true
		req.tags = append(req.tags, tag)
	}

	if req.source == "mod" {
		req.tags = nil
	} else if len(req.tags) == 0 {
		return worldRequest{}, newMapProblem(problemWorldNoCountries, "至少选一个国家:新世界的州分给它们")
	}
	if len(req.tags) > worldMaxTags {
		return worldRequest{}, newMapProblem(problemWorldTooManyCountries, fmt.Sprintf("最多 %d 个国家", worldMaxTags)).with("max", worldMaxTags)
	}
	for _, lang := range strings.Split(r.FormValue("languages"), ",") {
		if lang = strings.ToLower(strings.TrimSpace(lang)); lang != "" {
			if _, _, ok := GUITextTarget(lang); !ok {
				return worldRequest{}, newMapProblem(problemBadRequest, "不认识的语言 "+lang)
			}
			req.languages = append(req.languages, lang)
		}
	}
	if len(req.languages) == 0 {
		req.languages = []string{"en"}
	}
	if problem := parseWorldOwners(r, &req); problem != nil {
		return worldRequest{}, problem
	}
	return req, nil
}

func parseWorldOwners(r *http.Request, req *worldRequest) *mapProblem {
	file, _, err := r.FormFile("owners")
	if err != nil {
		return nil
	}
	defer file.Close()
	picture, format, err := image.Decode(file)
	if err != nil || format != "png" {
		return newMapProblem(problemOwnersNotPNG, "国家涂色图要是 PNG")
	}
	bounds := picture.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 1 || h < 1 || w*h > worldOwnersMaxPixels {
		return newMapProblem(problemOwnersTooLarge, fmt.Sprintf("国家涂色图太大:最多 %d 像素", worldOwnersMaxPixels)).with("max", worldOwnersMaxPixels)
	}
	mask := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			red, _, _, alpha := picture.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if alpha == 0 {
				continue
			}

			v := int((red * 0xffff / alpha) >> 8)
			if v > len(req.tags) {
				return newMapProblem(problemOwnersExtraCountry, fmt.Sprintf("国家涂色图里有第 %d 个国家,但只选了 %d 个国家", v, len(req.tags))).with("index", v).with("count", len(req.tags))
			}
			mask[y*w+x] = uint8(v)
		}
	}
	req.owners, req.ownersW, req.ownersH = mask, w, h
	return nil
}

func worldOwners(world *mapgen.World, req worldRequest) (map[int]string, map[string]int, *mapProblem) {
	if req.owners == nil {
		owner, capital := world.Owners(req.tags)
		return owner, capital, nil
	}
	owner, capital, missing := world.OwnersFromMask(req.tags, req.owners, req.ownersW, req.ownersH)
	if len(missing) > 0 {
		return nil, nil, newMapProblem(problemWorldCountryUnpainted, fmt.Sprintf("国家 %s 没有分到任何一个州:在地图上给它涂一块陆地(涂的地方要盖住一个州的大半),或者把它从国家里去掉", strings.Join(missing, "、"))).with("tags", strings.Join(missing, ", "))
	}
	return owner, capital, nil
}

func parseWorldUpload(r *http.Request, in *mapgen.UploadInput) *mapProblem {
	picture := func(field, name string, required bool) (image.Image, *mapProblem) {
		file, _, err := r.FormFile(field)
		if err != nil {
			if required {
				return nil, newMapProblem(problemUploadMissing, "没有收到"+name).with("file", field+".bmp")
			}
			return nil, nil
		}
		defer file.Close()
		img, format, err := image.Decode(file)
		if err != nil {
			return nil, newMapProblem(problemUploadUnreadable, name+"读不出来,要是 PNG 或 BMP").with("file", field+".bmp")
		}
		if format == "jpeg" {
			return nil, newMapProblem(problemUploadJPEG, name+"不能是 JPEG:有损压缩会改掉颜色,要用 PNG 或 BMP").with("file", field+".bmp")
		}
		return img, nil
	}
	var problem *mapProblem
	if in.Provinces, problem = picture("provinces", "省份图", true); problem != nil {
		return problem
	}
	if in.Heightmap, problem = picture("heightmap", "高度图", true); problem != nil {
		return problem
	}
	if in.Terrain, problem = picture("terrain", "地表图", true); problem != nil {
		return problem
	}
	if in.Rivers, problem = picture("rivers", "河流图", false); problem != nil {
		return problem
	}
	if file, _, err := r.FormFile("definition"); err == nil {
		defer file.Close()
		body, err := io.ReadAll(io.LimitReader(file, 8<<20))
		if err != nil {
			return newMapProblem(problemUploadUnreadable, "definition.csv 读不出来").with("file", "definition.csv")
		}
		in.Definitions = body
	}
	return nil
}

type builtWorld struct {
	world *mapgen.World
	game  *mapgen.Base
	took  time.Duration

	images      map[string]image.Image
	definitions []byte
	findings    []mapgen.Finding
	terrain     image.Image

	made []string

	fixed map[string]int

	mod *modWorld
}

func (h *handlers) buildWorld(req worldRequest) (*builtWorld, *mapProblem, error) {
	shared, err := h.vanillaMapBase()
	if err != nil {
		return nil, nil, err
	}

	game := copyBase(shared)
	started := time.Now()
	out := &builtWorld{game: game}
	grouping := mapgen.WorldOptions{Seed: req.seed, ProvincesPerState: req.perState}
	switch req.source {
	case "game":
		out.world = mapgen.WorldOfBase(game, grouping)
	case "upload", "mod":
		up, err := mapgen.ReadUpload(req.upload, game)
		if problem := mapProblemOf(err); problem != nil {
			return nil, problem, nil
		}
		if err != nil {
			return nil, nil, err
		}
		if problem := worldSizeProblem(req.source, up.Base, game); problem != nil {
			return nil, problem, nil
		}
		b := up.Base
		out.images = map[string]image.Image{
			"provinces.bmp": b.ProvincesImage(), "heightmap.bmp": b.HeightImage(),
			"terrain.bmp": b.TerrainImage(), "rivers.bmp": b.RiversImage(),
		}
		out.definitions = b.DefinitionCSV()
		out.findings = mapgen.Check(up.Check)
		out.terrain = b.TerrainImage()
		if up.MadeDefinitions {
			out.made = append(out.made, "definitions")
		}
		if up.MadeRivers {
			out.made = append(out.made, "rivers")
		}
		out.fixed = map[string]int{}
		if up.FilledPixels > 0 {
			out.fixed["pixels"] = up.FilledPixels
		}
		if req.source == "mod" {
			world, mod, problem, err := buildModWorld(b, req)
			if problem != nil || err != nil {
				return nil, problem, err
			}
			out.world, out.mod = world, mod
			if mod.landedLakeStates > 0 {
				out.fixed["lake_states"] = mod.landedLakeStates
			}
			break
		}
		out.world = mapgen.WorldOfBase(b, grouping)
	default:
		m, err := mapgen.Generate(req.sketch, mapgen.Options{Width: game.Width, Height: game.Height, Seed: req.seed, LandProvincePixels: req.land, SeaProvincePixels: req.sea})
		if errors.Is(err, mapgen.ErrSize) {
			return nil, newMapProblem(problemWorldSizeRange, "地图尺寸不在可生成的范围里"), nil
		}
		if err != nil {
			return nil, nil, err
		}
		grey := image.NewPaletted(image.Rect(0, 0, m.Width, m.Height), mapgen.GreyPalette())
		copy(grey.Pix, m.Heights)
		out.images = map[string]image.Image{
			"provinces.bmp": m.ProvincesImage(), "heightmap.bmp": grey,
			"terrain.bmp": m.TerrainImage(game.TerrainPalette), "rivers.bmp": m.RiversImage(game.RiverPalette),
		}
		out.definitions = m.DefinitionCSV()
		out.findings = mapgen.Check(m.CheckInput())
		legendPalette := mapgen.GreyPalette()
		for _, entry := range mapgen.Legend {
			legendPalette[entry.Terrain] = color.RGBA{entry.Color[0], entry.Color[1], entry.Color[2], 255}
		}
		out.terrain = m.TerrainImage(legendPalette)
		out.world, err = mapgen.NewWorld(m, game.TerrainPalette, game.RiverPalette, grouping)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(out.world.All.States) == 0 {
		return nil, newMapProblem(problemWorldNoLand, "地图上没有陆地:至少要有一块陆地"), nil
	}

	out.findings = append(out.findings, out.world.LoadRules()...)
	if len(out.world.All.States) < len(req.tags) {
		return nil, newMapProblem(problemWorldFewStates, fmt.Sprintf("只有 %d 个州,不够分给 %d 个国家", len(out.world.All.States), len(req.tags))).
			with("states", len(out.world.All.States)).with("count", len(req.tags)), nil
	}
	out.took = time.Since(started)
	return out, nil, nil
}

func worldSizeProblem(source string, up, game *mapgen.Base) *mapProblem {
	if source == "mod" || (up.Width == game.Width && up.Height == game.Height) {
		return nil
	}
	return newMapProblem(problemWorldSize, fmt.Sprintf("上传的图要是 %d×%d(原版地图的尺寸;别的尺寸还没在游戏里核实),现在是 %d×%d", game.Width, game.Height, up.Width, up.Height)).
		with("width", game.Width).with("height", game.Height).with("got_width", up.Width).with("got_height", up.Height)
}

func (h *handlers) vanillaMapBase() (*mapgen.Base, error) {
	names := append(append([]string(nil), landImageFiles...), "definition.csv")
	parts := []string{"map", h.gameIdentity()}
	for range names {
		parts = append(parts, "game")
	}
	return decodedBase(baseKey(parts...), func() (*mapgen.Base, error) {
		data := map[string][]byte{}
		for _, name := range names {
			body, err := h.gameMapFile(name)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			data[name] = body
		}
		return mapgen.LoadBase(data["provinces.bmp"], data["heightmap.bmp"], data["terrain.bmp"], data["rivers.bmp"], data["definition.csv"])
	})
}

func (h *handlers) worldRoute(preview bool) func(http.ResponseWriter, *http.Request, *project.Project) {
	return func(w http.ResponseWriter, r *http.Request, p *project.Project) { h.world(w, r, p, preview) }
}

func (h *handlers) world(w http.ResponseWriter, r *http.Request, p *project.Project, preview bool) {
	release, ok := acquireLandGate(w)
	if !ok {
		return
	}
	defer release()
	req, problem := parseWorldRequest(w, r)
	if problem != nil {
		writeMapProblem(w, http.StatusUnprocessableEntity, problem)
		return
	}
	h.makeWorld(w, r, p, preview, req)
}

func acquireLandGate(w http.ResponseWriter) (func(), bool) {
	select {
	case landGate <- struct{}{}:
		return func() { <-landGate }, true
	default:
		refuseMap(w, http.StatusTooManyRequests, problemBusy, "另一张地图正在生成,稍等几秒再试")
		return nil, false
	}
}

func (h *handlers) makeWorld(w http.ResponseWriter, r *http.Request, p *project.Project, preview bool, req worldRequest) {
	ctx := r.Context()
	files, err := h.landFiles(ctx, p)
	if err != nil {
		var refused *landRefusal
		if errors.As(err, &refused) {
			refuseMap(w, http.StatusConflict, problemOwnMap, refused.message)
			return
		}
		log.Printf("world map world %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件(原版地图还没放到服务器上,或项目的地图文件读取失败)")
		return
	}

	if imported, found, err := h.svc.ImportedIndex(ctx, p); err == nil && found && imported != nil && imported != h.svc.BaseIndex() && imported.HasOwnFilesIn("history/states") {
		refuseMap(w, http.StatusConflict, problemImportedStates, "这个项目导入的 mod 自带州,全新世界暂时只支持没有导入州的项目")
		return
	}

	changes := mapChangesOf(files.additions)
	if changes != nil && !preview {
		refuseMapChanged(w, changes)
		return
	}
	known := files.index.ListCountryTags()
	for _, tag := range req.tags {
		if !slices.Contains(known, tag) {
			writeMapProblem(w, http.StatusUnprocessableEntity, newMapProblem(problemNoCountry, "没有国家 "+tag+":先在国家工作台新建它").with("tag", tag))
			return
		}
	}
	built, problem, err := h.buildWorld(req)
	if err != nil {
		log.Printf("world map world %s build: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, "生成失败(原版地图读不到,或生成出错)")
		return
	}
	if problem != nil {
		writeMapProblem(w, http.StatusUnprocessableEntity, problem)
		return
	}
	findings := built.findings
	if findings == nil {
		findings = []mapgen.Finding{}
	}
	var owner map[int]string
	var capital map[string]int
	if built.mod != nil {
		var unknown []modUnknownTag
		owner, capital, req.tags, unknown = modOwners(built.world, built.mod, req, known)
		if len(unknown) > 0 {

			h.svc.Forget(p)
			if again, err := h.landFiles(ctx, p); err == nil {
				files, known = again, again.index.ListCountryTags()
				owner, capital, req.tags, unknown = modOwners(built.world, built.mod, req, known)
			}
		}
		if len(unknown) > 0 {
			tags := make([]string, 0, len(unknown))
			for _, u := range unknown {
				tags = append(tags, u.Tag)
			}
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"code":   problemModUnknownTags,
				"detail": "MOD 里这些国家项目里还没有:先新建它们,或把它们并进项目已有的国家(tag_map):" + strings.Join(tags, "、"),
				"values": map[string]any{"tags": unknown},
			})
			return
		}
	} else {
		owner, capital, problem = worldOwners(built.world, req)
		if problem != nil {
			writeMapProblem(w, http.StatusUnprocessableEntity, problem)
			return
		}
	}
	worldStore := files.store
	ownStarts, err := worldStore.WorldMapHistoryTags(req.tags)
	if err != nil {
		log.Printf("world map world %s histories: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到国家的开局历史")
		return
	}
	replacedStates, err := worldStore.WorldMapStateCount()
	if err != nil {
		log.Printf("world map world %s states: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到项目的州")
		return
	}
	summary := worldSummary(built, req, owner, replacedStates, ownStarts)
	if built.mod != nil {

		named, total := modNamedStates(built.mod, req, len(built.world.All.States)), len(built.world.All.States)
		summary["name_status"] = map[string]any{"languages": req.languages, "named_states": named, "total_states": total, "complete": named == total}
	}
	if preview {
		previews := map[string]string{"political": worldPoliticalPreview(built.world, owner, files.index)}
		if req.focus != nil {
			previews["political_close"] = worldPoliticalCloseUp(built.world, owner, req.tags, *req.focus)
		}
		if built.terrain != nil {
			previews["terrain"] = previewDataURL(built.terrain)
		}
		answer := map[string]any{
			"summary": summary, "findings": findings, "previews": previews,
			"state_layout": summary["state_layout"], "name_status": summary["name_status"],
		}
		if req.keepPreviews != nil {
			answer["preview_urls"] = req.keepPreviews(previews)
		}
		if changes != nil {
			answer["map_changes"] = changes
		}
		writeJSON(w, http.StatusOK, answer)
		return
	}
	if len(mapgen.Errors(findings)) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"code": problemWorldBreaksMap, "detail": "地图有错误,没有保存", "findings": findings})
		return
	}
	saved, missing, err := h.saveWorld(files, built, req, owner, capital, ownStarts, summary)
	if missing != "" {

		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, missing)
		return
	}
	if errors.Is(err, ErrWorldMapNotBase) {

		if additions, err := worldStore.WorldMapAdditions(); err == nil && len(additions) > 0 {
			refuseMapChanged(w, mapChangesOf(additions))
			return
		}
		refuseMap(w, http.StatusConflict, problemMapChanged, "项目的地图已经有新增土地或省份改动:先「恢复原版地图」,再创作整个世界")
		return
	}
	if err != nil {
		log.Printf("world map world %s save: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, "新世界没能存进项目,项目没有改动")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"addition": saved, "summary": summary, "findings": findings,
		"state_layout": summary["state_layout"], "name_status": summary["name_status"],
	})
}

func worldStateManifest(built *builtWorld, req worldRequest, owner map[int]string) []map[string]any {
	province := map[int]mapgen.Province{}
	for _, item := range built.world.All.Provinces {
		province[item.ID] = item
	}
	out := make([]map[string]any, 0, len(built.world.All.States))
	for k, state := range built.world.All.States {
		id := k + 1
		var sumX, sumY, weight float64
		for _, provinceID := range state.Provinces {
			item, ok := province[provinceID]
			if !ok {
				continue
			}
			pixels := float64(max(1, item.Pixels))
			sumX += item.X * pixels
			sumY += item.Y * pixels
			weight += pixels
		}
		mapX, mapY := 0, 0
		if weight > 0 {
			mapX, mapY = int(math.Round(sumX/weight)), int(math.Round(sumY/weight))
		}
		row := map[string]any{
			"state_id": id, "owner": owner[id], "capital_province": state.Capital,
			"province_count": len(state.Provinces), "map_position": map[string]int{"x": mapX, "y": mapY},
		}
		if placement := req.picturePlacement; placement != nil && placement.CanvasW > 0 && placement.CanvasH > 0 {
			canvasX := mapX * placement.CanvasW / max(1, built.world.Base.Width)
			canvasY := mapY * placement.CanvasH / max(1, built.world.Base.Height)
			if point, ok := placement.SourceOf(canvasX, canvasY); ok {
				row["image_position"] = map[string]int{"x": point.X, "y": point.Y}
			}
		}
		out = append(out, row)
	}
	return out
}

func worldSummary(built *builtWorld, req worldRequest, owner map[int]string, replacedStates int, ownStarts map[string]bool) map[string]any {
	counts := map[string]int{"land": 0, "sea": 0, "lake": 0}
	for _, pr := range built.world.All.Provinces {
		counts[pr.Kind]++
	}
	perTag := map[string]int{}
	for _, tag := range owner {
		perTag[tag]++
	}
	countries := make([]map[string]any, 0, len(req.tags))
	for _, tag := range req.tags {
		countries = append(countries, map[string]any{"tag": tag, "states": perTag[tag], "own_start": ownStarts[tag]})
	}
	return map[string]any{
		"source": req.source, "seed": req.seed, "seconds": built.took.Seconds(),
		"provinces": counts, "states": len(built.world.All.States), "regions": len(built.world.Regions),
		"countries": countries, "unloaded": mapgen.UnloadedFolders, "replaced_states": replacedStates,
		"state_layout": worldStateManifest(built, req, owner),
		"name_status": map[string]any{
			"languages": req.languages, "named_states": 0, "total_states": len(built.world.All.States),
			"complete": false, "next_tool": "name_world_map",
		},
		"made": append([]string{}, built.made...), "fixed": built.fixed,
	}
}

func worldPoliticalPreview(world *mapgen.World, owner map[int]string, index *guirender.Index) string {
	base := world.Base
	stateOf := map[int32]int{}
	for k, st := range world.All.States {
		for _, id := range st.Provinces {
			stateOf[int32(id)] = k + 1
		}
	}
	kind := map[int32]string{}
	for _, pr := range world.All.Provinces {
		kind[int32(pr.ID)] = pr.Kind
	}
	palette := []color.RGBA{{70, 110, 160, 255}, {170, 80, 60, 255}, {90, 140, 70, 255}, {160, 140, 60, 255}, {120, 80, 150, 255}, {60, 140, 140, 255}}
	colours := map[string]color.RGBA{}
	for _, tag := range owner {
		if _, ok := colours[tag]; ok {
			continue
		}
		if c, ok := index.CountryMapColor(tag); ok {
			colours[tag] = color.RGBA{c[0], c[1], c[2], 255}
		} else {
			colours[tag] = palette[len(colours)%len(palette)]
		}
	}
	width, height := base.Width, base.Height
	if width > sketchPreviewWidth {
		height = height * sketchPreviewWidth / width
		width = sketchPreviewWidth
	}
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		sy := y * base.Height / height
		for x := 0; x < width; x++ {
			id := base.Province[sy*base.Width+x*base.Width/width]
			c := color.RGBA{40, 70, 120, 255}
			switch {
			case kind[id] == "lake":
				c = color.RGBA{80, 120, 170, 255}
			case kind[id] == "land":
				c = color.RGBA{150, 150, 150, 255}
				if tag := owner[stateOf[id]]; tag != "" {
					c = colours[tag]
				}
			}
			out.SetRGBA(x, y, c)
		}
	}
	var body bytes.Buffer
	_ = (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&body, out)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(body.Bytes())
}

func (h *handlers) saveWorld(files *landFileSet, built *builtWorld, req worldRequest, owner map[int]string, capital map[string]int, ownStarts map[string]bool, summary map[string]any) (WorldMapAddition, string, error) {
	store, p := files.store, files.p
	vanilla := h.svc.BaseIndex()
	gameText := func(rel string) ([]byte, error) {
		if vanilla != nil {
			if body, ok := vanilla.Source(rel); ok {
				return body, nil
			}
		}
		return h.svc.ReadGameFile(rel, 128<<20)
	}
	tutorial, err := gameText("tutorial/tutorial.txt")
	if err != nil {
		log.Printf("world map world %s tutorial: %v", p.Dir(), err)
		return WorldMapAddition{}, "原版教程文件 tutorial/tutorial.txt 还没放到服务器上(新世界要一份改写过的教程,否则小世界开局就崩溃)", nil
	}
	countryTags, err := gameText("common/country_tags/00_countries.txt")
	if err != nil {
		return WorldMapAddition{}, "原版国家标签表 common/country_tags/00_countries.txt 读不到", nil
	}
	dynamicTags, _ := gameText("common/country_tags/zz_dynamic_countries.txt")
	gameLine := map[string]string{}
	for _, line := range strings.Split(string(countryTags), "\n") {
		if fields := strings.Fields(line); len(fields) > 0 {
			gameLine[fields[0]] = strings.TrimSpace(line)
		}
	}
	var tagLines, histories []string
	var leaderTags []string
	capitals := map[string]int{}
	for _, tag := range req.tags {
		if line, ok := gameLine[tag]; ok {
			tagLines = append(tagLines, line)
		}
		if ownStarts[tag] {
			capitals[tag] = capital[tag]
		} else {
			histories = append(histories, tag)
			if capital[tag] != 0 {
				leaderTags = append(leaderTags, tag)
			}
		}
	}
	landWeather, seaWeather := worldWeather(vanilla)
	if landWeather == "" {
		if body, err := h.svc.ReadGameFile("map/strategicregions/1-Southern England.txt", 1<<20); err == nil {
			landWeather = mapgen.WeatherBlock(string(body))
			seaWeather = landWeather
		}
	}
	var hoi4Languages []string
	for _, lang := range req.languages {
		if name, ok := locout.LangCodeToHOI4[lang]; ok && !slices.Contains(hoi4Languages, name) {
			hoi4Languages = append(hoi4Languages, name)
		}
	}
	worldMod := mapgen.WorldMod{
		Tags: req.tags, Owner: owner, Capital: capital, Seed: req.seed,
		LandWeather: landWeather, SeaWeather: seaWeather, Tutorial: string(tutorial),
		TagLines: tagLines, DynamicTags: string(dynamicTags), Histories: histories,
		Languages: hoi4Languages,
	}
	if built.mod != nil {
		worldMod.Names = built.mod.names
	}
	text := built.world.WorldFiles(worldMod)
	if built.definitions != nil {
		text["map/definition.csv"] = string(built.definitions)
	}
	written := make([]string, 0, len(text))
	for name := range text {
		written = append(written, name)
	}
	for name, marker := range mapgen.UnloadMarkers(written) {
		text[name] = marker
	}

	out := &blobInserts{files: files, note: "世界地图 · 全新世界", replace: mapgen.Unloads}
	var provincesImage image.Image
	if built.images != nil {
		provincesImage = built.images["provinces.bmp"]
		for _, file := range landImageFiles {
			data, err := mapgen.EncodeBMP(built.images[file])
			if err != nil || !out.binary("map/"+file, data) {
				return WorldMapAddition{}, "", fmt.Errorf("map/%s not stored: %v", file, err)
			}
		}

		for _, texture := range []struct {
			path  string
			paint func() ([]byte, error)
		}{
			{landColormap, func() ([]byte, error) {
				data, err := files.file(landColormap)
				if err != nil {
					return nil, err
				}
				return built.world.Base.PaintWorldColormap(built.game, data, req.seed)
			}},
			{"map/world_normal.bmp", built.world.Base.WorldNormals},
			{"map/trees.bmp", func() ([]byte, error) {
				data, err := files.file("map/trees.bmp")
				if err != nil {
					return nil, err
				}
				return built.world.Base.PaintWorldTrees(built.game, data, req.seed)
			}},
			{mapgen.FogTexture, func() ([]byte, error) {
				data, err := files.file(mapgen.FogTexture)
				if err != nil {
					return nil, err
				}
				return built.world.Base.PaintWorldFog(built.game, data, req.seed)
			}},
		} {
			data, err := texture.paint()
			if err != nil {
				log.Printf("world map world %s: %s left as the base game's: %v", p.Dir(), texture.path, err)
				continue
			}
			if !out.binary(texture.path, data) {
				return WorldMapAddition{}, "", fmt.Errorf("%s not stored", texture.path)
			}
		}

		water, err := files.file(mapgen.WaterColormapFiles[0])
		var levels [][]byte
		if err == nil {
			levels, err = built.world.Base.PaintWorldWater(built.game, water, req.seed)
		}
		if err != nil {
			log.Printf("world map world %s: the water colour maps left as the base game's: %v", p.Dir(), err)
		}
		for i, level := range levels {
			if !out.binary(mapgen.WaterColormapFiles[i], level) {
				return WorldMapAddition{}, "", fmt.Errorf("%s not stored", mapgen.WaterColormapFiles[i])
			}
		}
	}
	if err := out.texts(text); err != nil {
		return WorldMapAddition{}, "", err
	}
	rawPaths := out.paths()

	rows := make([]WorldMapStateRow, len(built.world.All.States))
	if built.mod != nil {
		rows = modStateRows(built.world, built.mod, req, owner, files.index.ListCountryTags())
	}
	for k, st := range built.world.All.States {
		if built.mod != nil {
			break
		}
		id := k + 1
		locs := map[string]string{}
		for _, lang := range req.languages {
			hoi4 := locout.LangCodeToHOI4[lang]
			locs[lang] = worldStateName(hoi4, id)
		}
		category := "town"
		if capital[owner[id]] == id {
			category = "city"
		}
		rows[k] = WorldMapStateRow{
			Provinces: st.Provinces, Manpower: int64(200000 + 50000*len(st.Provinces)), Category: category,
			Owner: owner[id], Buildings: map[string]any{"infrastructure": 2}, Locs: locs,
			VictoryPoints: map[int]int{st.Capital: 1},
		}
	}
	provinces := make([]int, 0, len(built.world.All.Provinces))
	for _, pr := range built.world.All.Provinces {
		provinces = append(provinces, pr.ID)
	}
	optionsJSON, _ := json.Marshal(map[string]any{"source": req.source, "land": req.land, "sea": req.sea, "per_state": req.perState, "tags": req.tags})
	summaryJSON, _ := json.Marshal(summary)
	record := WorldMapAddition{
		Kind: "world", Seed: req.seed, Options: optionsJSON, Name: worldRecordName(req), Owner: strings.Join(req.tags, ","),
		Provinces: provinces, RawPaths: rawPaths, TextKeys: []string{}, Summary: summaryJSON,
	}
	id, stateIDs, err := store.SaveWorldMapWorld(out.inserts, rows, capitals, leaderTags, record)
	if err != nil {
		return WorldMapAddition{}, "", err
	}
	record.ID, record.StateIDs = id, stateIDs
	if provincesImage != nil {
		var picture bytes.Buffer
		if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&picture, provincesImage); err == nil {
			projectMapImages.put(p.Dir(), out.shaOf["map/provinces.bmp"], picture.Bytes())
		} else {
			projectMapImages.forget(p.Dir())
		}
	} else {
		projectMapImages.forget(p.Dir())
	}
	files.saved()
	return record, "", nil
}

func worldRecordName(req worldRequest) string {
	if req.source == "mod" {
		return "导入 MOD 地图"
	}
	return "全新世界"
}

func worldWeather(vanilla *guirender.Index) (land, sea string) {
	if vanilla == nil {
		return "", ""
	}
	for id := 1; id <= vanilla.MaxStrategicRegionID(); id++ {
		file, ok := vanilla.StrategicRegionFile(id)
		if !ok {
			continue
		}
		body, ok := vanilla.Source(file)
		if !ok {
			continue
		}
		switch {
		case land == "" && strings.Contains(file, "Southern England"):
			land = mapgen.WeatherBlock(string(body))
		case sea == "" && strings.Contains(file, "North Sea"):
			sea = mapgen.WeatherBlock(string(body))
		}
	}
	if sea == "" {
		sea = land
	}
	return land, sea
}

func worldStateName(hoi4Language string, id int) string {
	if hoi4Language == "simp_chinese" {
		return fmt.Sprintf("第 %d 州", id)
	}
	return fmt.Sprintf("State %d", id)
}
