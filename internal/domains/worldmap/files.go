package worldmap

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"path"
	"strings"

	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/project"
)

var landImageFiles = []string{"provinces.bmp", "heightmap.bmp", "terrain.bmp", "rivers.bmp"}

const landColormap = "map/terrain/colormap_rgb_cityemissivemask_a.dds"

const (
	landTextFileMax = 1 << 20

	projectBlobMax = 256 << 20
)

var landGate = make(chan struct{}, 1)

type landRefusal struct{ message string }

func (e *landRefusal) Error() string { return e.message }

type landFileSet struct {
	h         *handlers
	p         *project.Project
	store     *mapStore
	raw       map[string]RawFile
	ours      map[string]bool
	index     *guirender.Index
	additions []WorldMapAddition
}

func (h *handlers) landFiles(ctx context.Context, p *project.Project) (*landFileSet, error) {
	set, err := h.landMapFiles(ctx, p)
	if err != nil {
		return nil, err
	}
	index, _, err := h.svc.Index(ctx, p)
	if err != nil {
		return nil, err
	}
	if index == nil {
		return nil, fmt.Errorf("no game data for the project")
	}
	set.index = index
	return set, nil
}

func (h *handlers) landMapFiles(ctx context.Context, p *project.Project) (*landFileSet, error) {
	set, err := h.landFileRows(p)
	if err != nil {
		return nil, err
	}
	imported, found, err := h.svc.ImportedIndex(ctx, p)
	if err != nil {
		return nil, err
	}

	_, ownPicture := set.raw["map/provinces.bmp"]
	if vanilla := h.svc.BaseIndex(); !ownPicture && found && imported != nil && imported != vanilla && imported.HasOwnFile("map/definition.csv") {
		own, _ := imported.Source("map/definition.csv")
		var game []byte
		if vanilla != nil {
			game, _ = vanilla.Source("map/definition.csv")
		}
		if !bytes.Equal(bytes.TrimSpace(own), bytes.TrimSpace(game)) {
			return nil, &landRefusal{"这个项目导入的 mod 自带地图,但它的地图文件还没有存进项目:重新导入一次 mod 后就能在它的地图上编辑"}
		}
	}
	return set, nil
}

func (h *handlers) landFileRows(p *project.Project) (*landFileSet, error) {
	store := h.store(p)
	rows, err := store.RawFiles()
	if err != nil {
		return nil, err
	}
	additions, err := store.WorldMapAdditions()
	if err != nil {
		return nil, err
	}
	return newLandFileSet(h, p, store, rows, additions)
}

func newLandFileSet(h *handlers, p *project.Project, store *mapStore, rows []RawFile, additions []WorldMapAddition) (*landFileSet, error) {
	set := &landFileSet{h: h, p: p, store: store, raw: map[string]RawFile{}, ours: map[string]bool{}, additions: additions}
	for _, row := range rows {
		set.raw[strings.ToLower(row.DestPath)] = row
	}
	for _, a := range additions {
		for _, p := range a.RawPaths {
			set.ours[strings.ToLower(p)] = true
		}
	}
	for lower, row := range set.raw {
		if strings.HasPrefix(lower, "map/") && !set.ours[lower] && isLandMapPath(lower) {
			if problem := unreadableMapFile(row); problem != "" {
				return nil, &landRefusal{fmt.Sprintf("项目里的地图文件 %s %s,地图编辑读不了它", row.DestPath, problem)}
			}
		}
	}
	return set, nil
}

func unreadableMapFile(row RawFile) string {
	if row.RawKind != "" && row.RawKind != "verbatim_override" {
		return "不是整份文件(" + row.RawKind + ")"
	}
	if row.IsBinary && (row.ContentSHA256 == nil || strings.TrimSpace(*row.ContentSHA256) == "" ||
		row.ContentOBSKey == nil || strings.TrimSpace(*row.ContentOBSKey) == "") {
		return "没有存储位置"
	}
	return ""
}

func (f *landFileSet) ownMap() bool {
	for lower := range f.raw {
		if isLandMapPath(lower) {
			return true
		}
	}
	return false
}

func isLandMapPath(lower string) bool {
	if lower == landColormap {
		return true
	}
	switch path.Base(lower) {
	case "provinces.bmp", "heightmap.bmp", "terrain.bmp", "rivers.bmp", "definition.csv",
		"buildings.txt", "unitstacks.txt", "supply_nodes.txt", "weatherpositions.txt",
		"world_normal.bmp", "trees.bmp":
		return path.Dir(lower) == "map"
	}
	return false
}

func (f *landFileSet) file(name string) ([]byte, error) {
	lower := strings.ToLower(name)
	if row, ok := f.raw[lower]; ok && (f.ours[lower] || unreadableMapFile(row) == "") {
		if !row.IsBinary {
			return []byte(row.Content), nil
		}
		if row.ContentOBSKey == nil || f.p == nil {
			return nil, fmt.Errorf("%s: stored bytes not found", name)
		}
		body, err := f.p.Blobs().Get(*row.ContentOBSKey, projectBlobMax)
		if err != nil {
			return nil, fmt.Errorf("%s: stored bytes not found (%v)", name, err)
		}
		return body, nil
	}
	if f.h == nil {
		return nil, fmt.Errorf("%s: no game folder", name)
	}
	return f.h.gameMapFile(strings.TrimPrefix(name, "map/"))
}

func (f *landFileSet) current(name string) ([]byte, error) {
	if row, ok := f.raw[strings.ToLower(name)]; ok && !row.IsBinary && !f.ours[strings.ToLower(name)] {
		return []byte(row.Content), nil
	}
	return f.file(name)
}

func (f *landFileSet) base() (*mapgen.Base, error) {
	shared, err := f.sharedBase()
	if err != nil {
		return nil, err
	}
	return copyBase(shared), nil
}

func (f *landFileSet) sharedBase() (*mapgen.Base, error) {
	names := append(append([]string(nil), landImageFiles...), "definition.csv")
	parts := []string{"map"}
	if f.h != nil {
		parts = append(parts, f.h.gameIdentity())
	}
	for _, name := range names {
		lower := "map/" + name
		row, own := f.raw[lower]
		switch {
		case !own || !(f.ours[lower] || unreadableMapFile(row) == ""):
			parts = append(parts, "game")
		case row.IsBinary && row.ContentSHA256 != nil:
			parts = append(parts, "blob:"+*row.ContentSHA256)
		default:
			parts = append(parts, "text:"+contentKey(row.Content))
		}
	}
	return decodedBase(baseKey(parts...), func() (*mapgen.Base, error) {
		data := map[string][]byte{}
		for _, name := range names {
			body, err := f.file("map/" + name)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			data[name] = body
		}
		return mapgen.LoadBase(data["provinces.bmp"], data["heightmap.bmp"], data["terrain.bmp"], data["rivers.bmp"], data["definition.csv"])
	})
}

func (f *landFileSet) findingCounts(base *mapgen.Base) map[string]int {
	vanilla := len(f.additions) == 0 && !f.ownMap() && f.h != nil
	identity := ""
	if vanilla {
		identity = f.h.gameIdentity()
		vanillaFindings.Lock()
		defer vanillaFindings.Unlock()
		if vanillaFindings.counts != nil && vanillaFindings.game == identity {
			return vanillaFindings.counts
		}
	}
	counts := map[string]int{}
	for _, finding := range mapgen.Check(base.CheckInput(nil)) {
		counts[finding.Code] = finding.Count
	}
	if vanilla {
		vanillaFindings.counts, vanillaFindings.game = counts, identity
	}
	return counts
}

func newFindings(after []mapgen.Finding, before map[string]int) []mapgen.Finding {
	out := []mapgen.Finding{}
	for _, finding := range after {
		if more := finding.Count - before[finding.Code]; more > 0 {
			finding.Count = more
			out = append(out, finding)
		}
	}
	return out
}

func (f *landFileSet) regionText(file string) (text, refusal string, err error) {
	if row, has := f.raw[strings.ToLower(file)]; has {
		if row.IsBinary || (row.RawKind != "" && row.RawKind != "verbatim_override") {
			return "", fmt.Sprintf("项目里的战略区文件 %s 不是整份文本,这里不覆盖它", row.DestPath), nil
		}
		return row.Content, "", nil
	}
	body, ok := f.index.Source(file)
	if !ok {
		return "", "", fmt.Errorf("strategic region file %s unreadable", file)
	}
	return string(body), "", nil
}

func (f *landFileSet) projectRegions() (map[int][]int, map[int]string, error) {
	regions, files := map[int][]int{}, map[int]string{}
	for id := 1; id <= f.index.MaxStrategicRegionID(); id++ {
		file, ok := f.index.StrategicRegionFile(id)
		if !ok {
			continue
		}
		text, refusal, err := f.regionText(file)
		if err != nil {
			return nil, nil, err
		}
		if refusal != "" {
			log.Printf("world map: strategic region %d left out: %s", id, refusal)
			continue
		}
		regions[id], files[id] = mapgen.RegionProvinces(text), file
	}
	return regions, files, nil
}

func (f *landFileSet) stateOf() map[int]int {
	out := map[int]int{}
	for _, row := range f.index.WorldMap(worldMapDefaultDate).Provinces {
		id, _ := row[0].(int)
		if state, _ := row[6].(int); state > 0 {
			out[id] = state
		}
	}
	return out
}

func (f *landFileSet) storeBlob(data []byte) (sha, key string, ok bool) {
	sha, key, err := f.h.svc.StoreRawBlob(f.p, data)
	if err != nil {
		log.Printf("world map %s: blob not stored: %v", f.p.Dir(), err)
		return "", "", false
	}
	return sha, key, true
}

func (f *landFileSet) saved() {
	f.h.svc.Forget(f.p)
}

func (f *landFileSet) remember(inserts []RawFileInsert) {
	for _, insert := range inserts {
		lower := strings.ToLower(insert.DestPath)
		f.raw[lower] = RawFile{DestPath: insert.DestPath, Content: insert.Content, IsBinary: insert.IsBinary,
			RawKind: insert.RawKind, Note: insert.Note, ContentSHA256: insert.ContentSHA256, ContentOBSKey: insert.ContentOBSKey}
		f.ours[lower] = true
	}
}
