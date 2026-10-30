package gameview

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"haganeworks/internal/gamefiles"
	"io"
	"io/fs"
	"log"
	"path"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"haganeworks/internal/hoi4/guirender"
)

const vanillaBaseURL = "/api/go/gui-vanilla"

type textSubtree struct {
	dir       string
	suffixes  []string
	recursive bool
}

var textSubtrees = []textSubtree{
	{"interface", []string{".gui", ".gfx"}, true},
	{"localisation/english", []string{".yml"}, true},

	{"gfx/fonts", []string{".fnt"}, false},
	{"common/national_focus", []string{".txt"}, true},
	{"common/characters", []string{".txt"}, true},
	{"common/idea_tags", []string{".txt"}, true},
	{"common/bookmarks", []string{".txt"}, true},
	{"common/country_tags", []string{".txt"}, true},
	{"common/ideologies", []string{".txt"}, true},
	{"common/ideas", []string{".txt"}, true},
	{"common/dynamic_modifiers", []string{".txt"}, true},
	{"history/countries", []string{".txt"}, true},
	{"history/states", []string{".txt"}, true},
	{"history/units", []string{".txt"}, true},
	{"history/general", []string{".txt"}, true},
	{"documentation", []string{"modifiers_documentation.md"}, false},
	{"map", []string{"continent.txt", "definition.csv"}, false},
	{"tutorial", []string{"tutorial.txt"}, false},
	{"events", []string{".txt"}, true},
	{"common/countries", []string{".txt"}, true},
	{"common/units", []string{".txt"}, true},
	{"common/technologies", []string{".txt"}, true},
	{"common/technology_tags", []string{".txt"}, true},
	{"common/buildings", []string{".txt"}, true},
	{"common/resources", []string{".txt"}, true},
	{"common/autonomous_states", []string{".txt"}, true},
	{"common/wargoals", []string{".txt"}, true},
	{"common/intelligence_agencies", []string{".txt"}, true},
	{"common/intelligence_agency_upgrades", []string{".txt"}, true},
	{"common/special_projects", []string{".txt"}, true},
	{"common/scientist_traits", []string{".txt"}, true},
	{"map/strategicregions", []string{".txt"}, true},
	{"common/factions", []string{".txt"}, true},
	{"common/decisions", []string{".txt"}, true},
	{"common/military_industrial_organization", []string{".txt"}, true},
	{"common/equipment_groups", []string{".txt"}, true},
	{"common/unit_leader", []string{".txt"}, true},
	{"common/country_leader", []string{".txt"}, true},
	{"common/doctrines", []string{".txt"}, true},
	{"common/modifiers", []string{".txt"}, true},
	{"common/frontend", []string{".txt"}, true},
	{"common/continuous_focus", []string{".txt"}, true},
	{"common/scripted_triggers", []string{".txt"}, true},

	{"common", []string{"alerts.txt"}, false},
	{"common/defines", []string{".lua"}, false},
}

var manifestImageSubtrees = []string{
	"gfx/interface", "gfx/leaders", "gfx/flags", "gfx/texticons", "gfx/event_pictures", "gfx/loadingscreens",
}

type contentRoot struct {
	name string
	fsys fs.FS
}

type gameFile struct {
	root int
	rel  string
}

type gameLayer struct {
	index   *guirender.Index
	roots   []contentRoot
	files   map[string]gameFile
	version string
	built   time.Duration
}

func contentRoots(fsys fs.FS) []contentRoot {
	var roots []contentRoot
	for _, root := range gamefiles.Roots(fsys) {
		roots = append(roots, contentRoot{name: root.Name, fsys: root.FS})
	}
	return roots
}

func gameIdentity(fsys fs.FS) string { return gamefiles.Identity(fsys) }

func gameVersion(fsys fs.FS) string {
	raw, err := fs.ReadFile(fsys, "launcher-settings.json")
	if err == nil {
		var settings struct {
			RawVersion string `json:"rawVersion"`
			Version    string `json:"version"`
		}
		if json.Unmarshal(trimBOM(raw), &settings) == nil {
			if v := strings.TrimSpace(settings.RawVersion); v != "" {
				return v
			}
			if v := strings.TrimSpace(settings.Version); v != "" {
				return v
			}
		}
	}
	return "local"
}

func trimBOM(raw []byte) []byte {
	if len(raw) >= 3 && raw[0] == 0xef && raw[1] == 0xbb && raw[2] == 0xbf {
		return raw[3:]
	}
	return raw
}

func buildGameLayer(fsys fs.FS) (*gameLayer, error) {
	started := time.Now()
	roots := contentRoots(fsys)
	files := collectTexts(roots)
	if len(files) == 0 {
		return nil, errors.New("the game folder has no interface files")
	}
	sizes := loadSizeCache(fsys)
	textures, located := collectManifest(roots, sizes)
	sizes.save()
	layer := &gameLayer{roots: roots, files: located, version: gameVersion(fsys)}
	raw, err := json.Marshal(guirender.VanillaManifest{GameVersion: layer.version, Textures: textures})
	if err != nil {
		return nil, err
	}
	files[guirender.VanillaManifestPath] = raw
	index, err := guirender.NewVanillaIndex(files, vanillaBaseURL)
	if err != nil {
		return nil, err
	}
	layer.index = index
	layer.built = time.Since(started)
	return layer, nil
}

type textFile struct {
	root int
	rel  string
}

func collectTexts(roots []contentRoot) map[string][]byte {
	var list []textFile
	for _, sub := range textSubtrees {
		for i, root := range roots {
			list = append(list, listText(i, root.fsys, sub)...)
		}
	}
	bodies := make([][]byte, len(list))
	var wait sync.WaitGroup
	jobs := make(chan int)
	for worker := 0; worker < min(8, runtime.GOMAXPROCS(0)); worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for i := range jobs {
				if body, err := fs.ReadFile(roots[list[i].root].fsys, list[i].rel); err == nil {
					bodies[i] = body
				}
			}
		}()
	}
	for i := range list {
		jobs <- i
	}
	close(jobs)
	wait.Wait()
	files := make(map[string][]byte, len(list))
	for i, file := range list {
		if bodies[i] != nil {
			files[file.rel] = bodies[i]
		}
	}
	return files
}

func listText(root int, fsys fs.FS, sub textSubtree) []textFile {
	wanted := func(name string) bool {
		lower := strings.ToLower(name)
		ext := path.Ext(lower)
		for _, suffix := range sub.suffixes {
			if suffix == ext || suffix == lower {
				return true
			}
		}
		return false
	}
	var out []textFile
	if !sub.recursive {
		entries, err := fs.ReadDir(fsys, sub.dir)
		if err != nil {
			return nil
		}
		for _, entry := range entries {
			if !entry.IsDir() && wanted(entry.Name()) {
				out = append(out, textFile{root, sub.dir + "/" + entry.Name()})
			}
		}
		return out
	}
	_ = fs.WalkDir(fsys, sub.dir, func(rel string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && wanted(entry.Name()) {
			out = append(out, textFile{root, rel})
		}
		return nil
	})
	return out
}

func collectManifest(roots []contentRoot, sizes *sizeCache) (map[string][]int, map[string]gameFile) {
	type candidate struct {
		folded string
		file   gameFile
		stamp  [2]int64
	}
	var candidates []candidate
	add := func(i int, rel string, entry fs.DirEntry) {
		c := candidate{folded: strings.ToLower(rel), file: gameFile{i, rel}}
		if info, err := entry.Info(); err == nil {
			c.stamp = [2]int64{info.Size(), info.ModTime().UnixNano()}
		}
		candidates = append(candidates, c)
	}
	for i, root := range roots {
		if entries, err := fs.ReadDir(root.fsys, "gfx/fonts"); err == nil {
			for _, entry := range entries {
				ext := strings.ToLower(path.Ext(entry.Name()))
				if !entry.IsDir() && (ext == ".dds" || ext == ".tga") {
					add(i, "gfx/fonts/"+entry.Name(), entry)
				}
			}
		}
		for _, sub := range manifestImageSubtrees {
			_ = fs.WalkDir(root.fsys, sub, func(rel string, entry fs.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return nil
				}
				switch strings.ToLower(path.Ext(rel)) {
				case ".dds", ".tga", ".png":
					add(i, rel, entry)
				}
				return nil
			})
		}
	}
	measured := make([][2]int, len(candidates))
	var missing []int
	for i, c := range candidates {
		if size, ok := sizes.lookup(roots[c.file.root].name+"|"+c.file.rel, c.stamp); ok {
			measured[i] = size
		} else {
			missing = append(missing, i)
		}
	}

	var wait sync.WaitGroup
	jobs := make(chan int)
	for worker := 0; worker < min(8, runtime.GOMAXPROCS(0)); worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for i := range jobs {
				c := candidates[i]
				if w, h, ok := headerSize(roots[c.file.root].fsys, c.file.rel); ok {
					measured[i] = [2]int{w, h}
				}
			}
		}()
	}
	for _, i := range missing {
		jobs <- i
	}
	close(jobs)
	wait.Wait()
	manifest := make(map[string][]int, len(candidates))
	located := make(map[string]gameFile, len(candidates))
	for i, c := range candidates {
		sizes.store(roots[c.file.root].name+"|"+c.file.rel, c.stamp, measured[i])
		if measured[i][0] <= 0 || measured[i][1] <= 0 {
			continue
		}
		manifest[c.folded] = []int{measured[i][0], measured[i][1]}
		located[c.folded] = c.file
	}
	return manifest, located
}

func headerSize(fsys fs.FS, rel string) (int, int, bool) {
	file, err := fsys.Open(rel)
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()
	head := make([]byte, 32)
	n, _ := io.ReadFull(file, head)
	return imageHeaderSize(path.Ext(rel), head[:n])
}

func imageHeaderSize(ext string, head []byte) (int, int, bool) {
	switch strings.ToLower(ext) {
	case ".dds":
		if len(head) < 20 || string(head[:4]) != "DDS " {
			return 0, 0, false
		}
		h := int(binary.LittleEndian.Uint32(head[12:16]))
		w := int(binary.LittleEndian.Uint32(head[16:20]))
		return w, h, w > 0 && h > 0
	case ".tga":
		if len(head) < 18 {
			return 0, 0, false
		}
		w := int(binary.LittleEndian.Uint16(head[12:14]))
		h := int(binary.LittleEndian.Uint16(head[14:16]))
		return w, h, w > 0 && h > 0
	case ".png":
		if len(head) < 24 || string(head[:8]) != "\x89PNG\r\n\x1a\n" {
			return 0, 0, false
		}
		w := int(binary.BigEndian.Uint32(head[16:20]))
		h := int(binary.BigEndian.Uint32(head[20:24]))
		return w, h, w > 0 && h > 0
	}
	return 0, 0, false
}

func (g *gameLayer) readFile(rel string, maxBytes int64) ([]byte, string, error) {
	rel = strings.TrimPrefix(strings.ReplaceAll(rel, "\\", "/"), "/")
	if !fs.ValidPath(rel) {
		return nil, "", fs.ErrNotExist
	}
	if file, ok := g.files[strings.ToLower(rel)]; ok {
		body, err := readLimited(g.roots[file.root].fsys, file.rel, maxBytes)
		return body, file.rel, err
	}
	for i := len(g.roots) - 1; i >= 0; i-- {
		body, err := readLimited(g.roots[i].fsys, rel, maxBytes)
		if err == nil {
			return body, rel, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, "", err
		}
	}

	dir, base := path.Split(rel)
	for i := len(g.roots) - 1; i >= 0; i-- {
		entries, err := fs.ReadDir(g.roots[i].fsys, cmpOr(strings.TrimSuffix(dir, "/"), "."))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() && entry.Name() != base && strings.EqualFold(entry.Name(), base) {
				body, err := readLimited(g.roots[i].fsys, dir+entry.Name(), maxBytes)
				return body, dir + entry.Name(), err
			}
		}
	}
	return nil, "", fs.ErrNotExist
}

func readLimited(fsys fs.FS, rel string, maxBytes int64) ([]byte, error) {
	file, err := fsys.Open(rel)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if info, err := file.Stat(); err == nil && info.IsDir() {
		return nil, fs.ErrNotExist
	}
	body, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("%s is larger than %d bytes", rel, maxBytes)
	}
	return body, nil
}

var gameCache struct {
	sync.Mutex
	key      string
	layer    *gameLayer
	used     time.Time
	flight   *gameFlight
	failed   string
	failedAt time.Time
	failure  error
}

type gameFlight struct {
	key   string
	done  chan struct{}
	layer *gameLayer
	err   error
}

const gameRetryCooldown = 30 * time.Second

var errNoGame = errors.New("the Hearts of Iron IV folder is not set up")

func gameLayerFor(open func() (fs.FS, error)) (*gameLayer, error) {
	if open == nil {
		return nil, errNoGame
	}
	fsys, err := open()
	if err != nil {
		return nil, err
	}
	key := gameIdentity(fsys)
	gameCache.Lock()
	if gameCache.layer != nil && gameCache.key == key {
		gameCache.used = time.Now()
		layer := gameCache.layer
		gameCache.Unlock()
		return layer, nil
	}
	if gameCache.failed == key && time.Since(gameCache.failedAt) < gameRetryCooldown {
		err := gameCache.failure
		gameCache.Unlock()
		return nil, err
	}
	flight := gameCache.flight
	if flight == nil || flight.key != key {
		flight = &gameFlight{key: key, done: make(chan struct{})}
		gameCache.flight = flight
		go finishGameFlight(flight, fsys)
	}
	gameCache.Unlock()
	<-flight.done
	return flight.layer, flight.err
}

func finishGameFlight(flight *gameFlight, fsys fs.FS) {
	flight.layer, flight.err = buildGameLayer(fsys)
	gameCache.Lock()
	if gameCache.flight == flight {
		gameCache.flight = nil
		if flight.err == nil {
			gameCache.key, gameCache.layer, gameCache.used = flight.key, flight.layer, time.Now()
			gameCache.failed = ""
			log.Printf("[gameview] base game %s ready in %s: %d windows, %d sprites, %d textures",
				flight.layer.version, flight.layer.built.Round(time.Millisecond),
				flight.layer.index.TotalWindowCount(), flight.layer.index.TotalSpriteCount(), len(flight.layer.files))
			startSweeper()
		} else {
			gameCache.failed, gameCache.failedAt, gameCache.failure = flight.key, time.Now(), flight.err
			log.Printf("[gameview] base game unavailable, screens draw without it: %v", flight.err)
		}
	}
	gameCache.Unlock()
	close(flight.done)
}

func dropIdleGame(idle time.Duration) bool {
	gameCache.Lock()
	defer gameCache.Unlock()
	if gameCache.layer == nil || time.Since(gameCache.used) < idle {
		return false
	}
	gameCache.layer, gameCache.key = nil, ""
	vanillaLanguages.drop()
	return true
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cmpOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
