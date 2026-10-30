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

type terrainRequest struct {
	Provinces map[string]string `json:"provinces"`
	Look      bool              `json:"look"`
}

func parseTerrain(body []byte) (terrainRequest, map[int]string, string) {
	var in terrainRequest
	if err := json.Unmarshal(body, &in); err != nil {
		return in, nil, "读不懂这次改动"
	}
	if len(in.Provinces) == 0 {
		return in, nil, "没有要改的省份"
	}
	if len(in.Provinces) > 20000 {
		return in, nil, "一次改的省份太多"
	}
	out := map[int]string{}
	for key, terrain := range in.Provinces {
		id, err := strconv.Atoi(strings.TrimSpace(key))
		if err != nil || id <= 0 {
			return in, nil, "省份编号不对:" + key
		}
		terrain = strings.ToLower(strings.TrimSpace(terrain))
		if !landTerrains[terrain] {
			return in, nil, "不认识的地形 " + terrain
		}
		out[id] = terrain
	}
	return in, out, ""
}

func (h *handlers) terrain(w http.ResponseWriter, r *http.Request, p *project.Project) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
	if err != nil {
		refuseMap(w, http.StatusUnprocessableEntity, problemTooLarge, "改动太大")
		return
	}
	req, terrains, problem := parseTerrain(body)
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
		log.Printf("world map terrain %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件")
		return
	}
	out, failed, err := reterrain(files, req, terrains)
	if failed != nil {
		writeMapProblem(w, http.StatusUnprocessableEntity, failed)
		return
	}
	if err != nil {
		log.Printf("world map terrain %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, "没能存进项目,项目没有改动")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func reterrain(files *landFileSet, req terrainRequest, terrains map[int]string) (map[string]any, *mapProblem, error) {
	out := &blobInserts{files: files, note: "世界地图 · 地图创作改了地形"}
	var definitions []byte
	painted := 0
	if !req.Look {

		body, err := files.file("map/definition.csv")
		if err != nil {
			return nil, nil, err
		}
		rows := mapgen.ParseDefinitions(body)
		at := map[int]int{}
		for i, row := range rows {
			at[row.ID] = i
		}
		for id, terrain := range terrains {
			i, ok := at[id]
			if !ok {
				return nil, mapProblemOf(mapgen.NoProvince(id)), nil
			}
			if rows[i].Kind != "land" {
				return nil, mapProblemOf(mapgen.WaterTerrain(id)), nil
			}
			rows[i].Terrain = terrain
		}
		definitions = mapgen.DefinitionsCSV(rows)
	} else {
		base, err := files.base()
		if err != nil {
			return nil, nil, err
		}
		seed := files.p.ID()
		add, err := base.Reterrain(terrains, seed)
		if err != nil {
			return nil, mapProblemOr(err, problemBadRequest), nil
		}
		definitions = base.DefinitionCSV()
		for _, image := range []struct {
			path   string
			encode func() ([]byte, error)
		}{
			{"map/terrain.bmp", func() ([]byte, error) { return mapgen.EncodeBMP(base.TerrainImage()) }},
			{"map/heightmap.bmp", func() ([]byte, error) { return mapgen.EncodeBMP(base.HeightImage()) }},
		} {
			data, err := image.encode()
			if err != nil || !out.binary(image.path, data) {
				return nil, nil, fmt.Errorf("%s not stored: %v", image.path, err)
			}
		}

		if err := out.textures("terrain", []texturePaint{
			{landColormap, func(data []byte) ([]byte, error) { return base.PaintColormap(data, add, seed) }},
			{"map/world_normal.bmp", func(data []byte) ([]byte, error) { return base.PaintNormals(data, add) }},
			{"map/trees.bmp", func(data []byte) ([]byte, error) { return base.PaintTrees(data, add, seed) }},
		}); err != nil {
			return nil, nil, err
		}
		painted = len(add.Provinces)
	}
	out.inserts = append(out.inserts, RawFileInsert{DestPath: "map/definition.csv", Content: string(definitions), RawKind: "verbatim_override", Note: out.note})
	ids := make([]int, 0, len(terrains))
	for id := range terrains {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	paths := out.paths()
	summary, _ := json.Marshal(map[string]any{"terrain": len(ids), "look": req.Look})
	record, err := files.store.SaveWorldMapAddition(out.inserts, nil, WorldMapAddition{
		Kind: "edit", Name: "地图创作 · 地形", Provinces: ids, RawPaths: paths, Summary: summary,
	})
	if err != nil {
		return nil, nil, err
	}
	files.saved()
	return map[string]any{"changed": len(ids), "record": record, "look": req.Look, "painted": painted, "files": paths}, nil, nil
}
