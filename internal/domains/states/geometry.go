package states

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
)

const (
	geometryFormat = "hoi4-states-geo"
	geometryFactor = 1
)

type statesGeometry struct {
	Format        string                `json:"format"`
	SchemaVersion int                   `json:"schema_version"`
	Source        string                `json:"source"`
	GameVersion   string                `json:"game_version"`
	GeneratedAt   string                `json:"generated_at,omitempty"`
	Factor        int                   `json:"factor"`
	Width         int                   `json:"width"`
	Height        int                   `json:"height"`
	StateCount    int                   `json:"state_count"`
	States        map[string]stateShape `json:"states"`
}

type statesGeometryResponse struct {
	Geometry  *statesGeometry `json:"geometry"`
	Source    string          `json:"source"`
	Available bool            `json:"available"`
}

type baseGeometry struct {
	geometry *statesGeometry
}

var (
	geoIDRE        = regexp.MustCompile(`\bid\s*=\s*(\d+)`)
	geoProvincesRE = regexp.MustCompile(`(?s)provinces\s*=\s*\{([^}]*)\}`)
)

func geometryStateProvinces(game fs.FS) map[int][]int {
	names, _ := stateFileNames(game)
	stateOf := map[int]int{}
	for _, name := range names {
		raw, err := fs.ReadFile(game, statesDir+"/"+name)
		if err != nil {
			continue
		}
		text := decodeGameScript(raw)
		id := geoIDRE.FindStringSubmatch(text)
		provinces := geoProvincesRE.FindStringSubmatch(text)
		if id == nil || provinces == nil {
			continue
		}
		sid, err := strconv.Atoi(id[1])
		if err != nil {
			continue
		}
		for _, token := range strings.Fields(provinces[1]) {
			if strings.Trim(token, "0123456789") == "" {
				if p, err := strconv.Atoi(token); err == nil {
					stateOf[p] = sid
				}
			}
		}
	}
	out := map[int][]int{}
	for p, s := range stateOf {
		out[s] = append(out[s], p)
	}
	return out
}

func gameShortVersion(v gameVersion) string {
	parts := strings.Split(v.Raw, ".")
	if len(parts) > 3 {
		parts = parts[:3]
	}
	return strings.Join(parts, ".")
}

func buildBaseGeometry(game fs.FS) (*statesGeometry, error) {
	provinces, err := fs.ReadFile(game, "map/provinces.bmp")
	if err != nil {
		return nil, err
	}
	definitions, err := fs.ReadFile(game, "map/definition.csv")
	if err != nil {
		return nil, err
	}
	states := geometryStateProvinces(game)
	if len(states) == 0 {
		return nil, fmt.Errorf("parsed 0 states")
	}
	width, height, province, err := loadProvinces(provinces, definitions)
	if err != nil {
		return nil, err
	}
	shapes := stateOutlines(width, height, province, geometryFactor, states)
	if len(shapes) == 0 {
		return nil, fmt.Errorf("vectorized 0 states")
	}
	out := &statesGeometry{
		Format: geometryFormat, SchemaVersion: 1, Source: "vanilla",
		GameVersion: gameShortVersion(readGameVersion(game)),
		GeneratedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000000") + "+00:00",
		Factor:      geometryFactor,
		Width:       (width + geometryFactor - 1) / geometryFactor,
		Height:      (height + geometryFactor - 1) / geometryFactor,
		States:      make(map[string]stateShape, len(shapes)),
	}
	for id, shape := range shapes {
		out.States[strconv.Itoa(id)] = shape
	}
	out.StateCount = len(out.States)
	return out, nil
}

func (v *vanillaGame) baseGeometryOf(game fs.FS) *statesGeometry {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.use(game)
	if v.geometry == nil {
		started := time.Now()
		geometry, err := buildBaseGeometry(game)
		if err != nil {
			log.Printf("states geometry: %v", err)
		} else {
			log.Printf("states geometry: traced %d states in %s", geometry.StateCount, time.Since(started).Round(time.Millisecond))
		}
		v.geometry = &baseGeometry{geometry: geometry}
	}
	return v.geometry.geometry
}

func projectDependency(record project.Row) (string, bool) {
	switch deps := record["dependencies"].(type) {
	case nil:
		return "", false
	case []any:
		if len(deps) == 0 {
			return "", false
		}
		if s, ok := deps[0].(string); ok {
			return s, true
		}
		return fmt.Sprint(deps[0]), true
	default:
		return strings.TrimSpace(fmt.Sprint(deps)), true
	}
}

func (h *handlers) statesGeometry(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	if dependency, ok := projectDependency(p.Record()); ok {
		apiutil.JSON(w, http.StatusOK, statesGeometryResponse{Source: dependency})
		return nil
	}
	game := h.game()
	if game == nil {
		apiutil.JSON(w, http.StatusOK, statesGeometryResponse{Source: "vanilla"})
		return nil
	}
	base := vanilla.baseGeometryOf(game)
	if base == nil {
		apiutil.JSON(w, http.StatusOK, statesGeometryResponse{Source: "vanilla"})
		return nil
	}
	geometry := base
	if merged, ok := h.projectOutlines(p, game, base); ok {
		geometry = merged
	}
	apiutil.JSON(w, http.StatusOK, statesGeometryResponse{Geometry: geometry, Source: "vanilla", Available: true})
	return nil
}

func changedStateProvinces(rows []project.Row, catalog []*vanillaStateFile) map[int][]int {
	game := make(map[int][]int, len(catalog))
	for _, state := range catalog {
		list := []int{}
		for _, v := range listOrEmpty(state.Row["provinces"]) {
			n, _ := whole(v)
			list = append(list, int(n))
		}
		slices.Sort(list)
		game[int(state.StateID)] = list
	}
	out := map[int][]int{}
	for _, row := range rows {
		id, _ := whole(row["state_id"])
		list := []int{}
		for _, v := range listOrEmpty(row["provinces"]) {
			n, _ := whole(v)
			list = append(list, int(n))
		}
		slices.Sort(list)
		if base, ok := game[int(id)]; !ok || !slices.Equal(base, list) {
			out[int(id)] = list
		}
	}
	return out
}

type mapFile struct {
	identity string
	read     func() ([]byte, error)
}

func projectMapFile(p *project.Project, game fs.FS, name string) mapFile {
	rows, _ := p.Rows("project_raw_files")
	for _, row := range rows {
		if !strings.EqualFold(strings.TrimSpace(str(row["dest_path"])), name) {
			continue
		}
		if kind := str(row["raw_kind"]); kind != "" && kind != "verbatim_override" {
			continue
		}
		if row["is_binary"] == true {
			key := strings.TrimSpace(str(row["content_obs_key"]))
			if key == "" {
				continue
			}
			return mapFile{identity: name + " own " + str(row["content_sha256"]) + " " + key,
				read: func() ([]byte, error) { return p.Blobs().Get(key, 256<<20) }}
		}
		content := str(row["content"])
		sum := sha256.Sum256([]byte(content))
		return mapFile{identity: name + " own " + hex.EncodeToString(sum[:]),
			read: func() ([]byte, error) { return []byte(content), nil }}
	}
	return mapFile{identity: name + " game", read: func() ([]byte, error) { return fs.ReadFile(game, name) }}
}

func (h *handlers) projectOutlines(p *project.Project, game fs.FS, base *statesGeometry) (*statesGeometry, bool) {
	rows, err := p.Rows(statesTable)
	if err != nil || len(rows) == 0 {
		return nil, false
	}
	changed := changedStateProvinces(rows, vanilla.stateFiles(game).Catalog)
	if len(changed) == 0 {
		return nil, false
	}
	provincesFile := projectMapFile(p, game, "map/provinces.bmp")
	definitionsFile := projectMapFile(p, game, "map/definition.csv")
	sum := sha256.New()
	fmt.Fprintf(sum, "%s\nfactor %d\nbase %s\n%s\n%s\n", p.Dir(), base.Factor, base.GeneratedAt, provincesFile.identity, definitionsFile.identity)
	ids := make([]int, 0, len(changed))
	for id := range changed {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		fmt.Fprintf(sum, "%d %v\n", id, changed[id])
	}
	key := hex.EncodeToString(sum.Sum(nil))
	if merged, ok := outlineCache.get(p.ID(), key); ok {
		return merged, true
	}
	provinces, err := provincesFile.read()
	if err != nil {
		log.Printf("states geometry %d: provinces.bmp: %v", p.ID(), err)
		return nil, false
	}
	definitions, err := definitionsFile.read()
	if err != nil {
		log.Printf("states geometry %d: definition.csv: %v", p.ID(), err)
		return nil, false
	}
	width, height, province, err := loadProvinces(provinces, definitions)
	if err != nil {
		log.Printf("states geometry %d: %v", p.ID(), err)
		return nil, false
	}
	shapes := stateOutlines(width, height, province, base.Factor, changed)
	merged := *base
	merged.States = make(map[string]stateShape, len(base.States)+len(changed))
	for id, shape := range base.States {
		merged.States[id] = shape
	}
	for id := range changed {
		name := strconv.Itoa(id)
		if shape, ok := shapes[id]; ok {
			merged.States[name] = shape
		} else {
			delete(merged.States, name)
		}
	}
	merged.StateCount = len(merged.States)
	outlineCache.put(p.ID(), key, &merged)
	return &merged, true
}

var outlineCache = &projectOutlineCache{entries: map[int64]outlineEntry{}}

type outlineEntry struct {
	key    string
	merged *statesGeometry
}

type projectOutlineCache struct {
	sync.Mutex
	entries map[int64]outlineEntry
}

func (c *projectOutlineCache) get(projectID int64, key string) (*statesGeometry, bool) {
	c.Lock()
	defer c.Unlock()
	entry, ok := c.entries[projectID]
	if !ok || entry.key != key {
		return nil, false
	}
	return entry.merged, true
}

func (c *projectOutlineCache) put(projectID int64, key string, merged *statesGeometry) {
	c.Lock()
	defer c.Unlock()
	if _, ok := c.entries[projectID]; !ok && len(c.entries) >= 8 {
		for id := range c.entries {
			delete(c.entries, id)
			break
		}
	}
	c.entries[projectID] = outlineEntry{key: key, merged: merged}
}
