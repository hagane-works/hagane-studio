package worldmap

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/project"
)

const riversMaxPixels = 400_000

func parseRivers(body []byte) (map[int]uint8, string) {
	var in struct {
		Pixels [][2]int `json:"pixels"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, "读不懂这次改动"
	}
	if len(in.Pixels) == 0 {
		return nil, "没有要改的河流"
	}
	if len(in.Pixels) > riversMaxPixels {
		return nil, "一次改的河流太多"
	}
	out := make(map[int]uint8, len(in.Pixels))
	for _, pair := range in.Pixels {
		at, value := pair[0], pair[1]
		if at < 0 {
			return nil, "河流的位置不对"
		}
		if !(value >= 0 && value <= mapgen.RiverWide) && value != mapgen.RiverLand {
			return nil, "河流图里没有这个值"
		}
		out[at] = uint8(value)
	}
	return out, ""
}

func riverSteps(at []int) []int {
	out := make([]int, len(at))
	prev := 0
	for i, p := range at {
		out[i] = p - prev
		prev = p
	}
	return out
}

func (h *handlers) riversRead(w http.ResponseWriter, r *http.Request, p *project.Project) {
	files, err := h.landFiles(r.Context(), p)
	if err != nil {
		var refused *landRefusal
		if errors.As(err, &refused) {
			refuseMap(w, http.StatusConflict, problemOwnMap, refused.message)
			return
		}
		log.Printf("world map rivers %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件")
		return
	}
	data, err := files.file("map/rivers.bmp")
	if err != nil {
		log.Printf("world map rivers %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到河流图(map/rivers.bmp)")
		return
	}
	width, height, rivers, err := mapgen.DecodeRivers(data)
	if err != nil {
		writeMapProblem(w, http.StatusUnprocessableEntity, mapProblemOr(err, problemUnreadable))
		return
	}
	at, values := mapgen.RiverPixels(rivers)
	ints := make([]int, len(values))
	for i, v := range values {
		ints[i] = int(v)
	}
	w.Header().Set("Cache-Control", "private, no-cache")
	writeJSON(w, http.StatusOK, map[string]any{"width": width, "height": height, "at": riverSteps(at), "values": ints})
}

func (h *handlers) rivers(w http.ResponseWriter, r *http.Request, p *project.Project) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 12<<20))
	if err != nil {
		refuseMap(w, http.StatusUnprocessableEntity, problemTooLarge, "改动太大")
		return
	}
	pixels, problem := parseRivers(body)
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
		log.Printf("world map rivers %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件")
		return
	}
	base, err := files.base()
	if err != nil {
		log.Printf("world map rivers %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图文件")
		return
	}
	changed, err := base.EditRivers(pixels)
	if err != nil {
		writeMapProblem(w, http.StatusUnprocessableEntity, mapProblemOr(err, problemBadRequest))
		return
	}
	if changed == 0 {
		refuseMap(w, http.StatusUnprocessableEntity, problemNothing, "河流没有变化")
		return
	}
	data, err := mapgen.EncodeBMP(base.RiversImage())
	if err != nil {
		log.Printf("world map rivers %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, "没能存进项目,项目没有改动")
		return
	}
	sha, key, stored := files.storeBlob(data)
	if !stored {
		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, "没能存进项目,项目没有改动")
		return
	}
	summary, _ := json.Marshal(map[string]any{"rivers": changed})
	record, err := files.store.SaveWorldMapAddition([]RawFileInsert{{
		DestPath: "map/rivers.bmp", IsBinary: true, RawKind: "verbatim_override",
		Note: "世界地图 · 地图创作改了河流", ContentSHA256: &sha, ContentOBSKey: &key,
	}}, nil, WorldMapAddition{Kind: "edit", Name: "地图创作 · 河流", RawPaths: []string{"map/rivers.bmp"}, Summary: summary})
	if err != nil {
		log.Printf("world map rivers %s: %v", p.Dir(), err)
		refuseMap(w, http.StatusServiceUnavailable, problemNotSaved, "没能存进项目,项目没有改动")
		return
	}
	files.saved()
	writeJSON(w, http.StatusOK, map[string]any{"changed": changed, "record": record})
}
