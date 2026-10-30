package worldmap

import (
	"crypto/sha256"
	"encoding/hex"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"haganeworks/internal/hoi4/mapgen"
)

const (
	cacheIdle = 10 * time.Minute

	sweepEvery = time.Minute

	gameMapFileMax = 256 << 20

	gameMapFileKept = 4 << 20
)

var gameMapFiles struct {
	sync.Mutex
	game string
	body map[string][]byte
	used time.Time
}

func (h *handlers) gameMapFile(name string) ([]byte, error) {
	identity := h.gameIdentity()
	gameMapFiles.Lock()
	if gameMapFiles.game == identity {
		if body, ok := gameMapFiles.body[name]; ok {
			gameMapFiles.used = time.Now()
			gameMapFiles.Unlock()
			return body, nil
		}
	}
	gameMapFiles.Unlock()
	body, err := h.svc.ReadGameFile("map/"+name, gameMapFileMax)
	if err != nil || len(body) > gameMapFileKept {
		return body, err
	}
	gameMapFiles.Lock()
	if gameMapFiles.game != identity || gameMapFiles.body == nil {
		gameMapFiles.game, gameMapFiles.body = identity, map[string][]byte{}
	}
	gameMapFiles.body[name], gameMapFiles.used = body, time.Now()
	gameMapFiles.Unlock()
	startSweeper()
	return body, nil
}

var mapBases struct {
	sync.Mutex
	key  string
	base *mapgen.Base
	used time.Time
}

func decodedBase(key string, load func() (*mapgen.Base, error)) (*mapgen.Base, error) {
	mapBases.Lock()
	if mapBases.base != nil && mapBases.key == key {
		mapBases.used = time.Now()
		base := mapBases.base
		mapBases.Unlock()
		return base, nil
	}
	mapBases.Unlock()
	base, err := load()
	if err != nil {
		return nil, err
	}
	mapBases.Lock()
	mapBases.key, mapBases.base, mapBases.used = key, base, time.Now()
	mapBases.Unlock()
	startSweeper()
	return base, nil
}

func copyBase(b *mapgen.Base) *mapgen.Base {
	out := *b
	out.Province = slices.Clone(b.Province)
	out.Heights = slices.Clone(b.Heights)
	out.Terrain = slices.Clone(b.Terrain)
	out.Rivers = slices.Clone(b.Rivers)
	out.TerrainPalette = slices.Clone(b.TerrainPalette)
	out.RiverPalette = slices.Clone(b.RiverPalette)
	out.Definitions = slices.Clone(b.Definitions)
	return &out
}

func contentKey(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:12])
}

func baseKey(parts ...string) string { return strings.Join(parts, "|") }

var gameImage struct {
	sync.Mutex
	game     string
	body     []byte
	tag      string
	used     time.Time
	failedAt time.Time
	err      error
}

var projectMapImages = &projectImageCache{entries: map[string]projectImage{}}

type projectImage struct {
	sha  string
	body []byte
	used time.Time
}

type projectImageCache struct {
	sync.Mutex
	entries map[string]projectImage
}

func (c *projectImageCache) get(project, sha string) ([]byte, bool) {
	c.Lock()
	defer c.Unlock()
	entry, ok := c.entries[project]
	if !ok || entry.sha != sha {
		return nil, false
	}
	entry.used = time.Now()
	c.entries[project] = entry
	return entry.body, true
}

func (c *projectImageCache) put(project, sha string, body []byte) {
	c.Lock()
	if _, ok := c.entries[project]; !ok && len(c.entries) >= 4 {
		oldest := ""
		for key, entry := range c.entries {
			if oldest == "" || entry.used.Before(c.entries[oldest].used) {
				oldest = key
			}
		}
		delete(c.entries, oldest)
	}
	c.entries[project] = projectImage{sha: sha, body: body, used: time.Now()}
	c.Unlock()
	startSweeper()
}

func (c *projectImageCache) forget(project string) {
	c.Lock()
	delete(c.entries, project)
	c.Unlock()
}

var vanillaFindings struct {
	sync.Mutex
	game   string
	counts map[string]int
}

var sweeperOnce sync.Once

func startSweeper() {
	sweeperOnce.Do(func() {
		go func() {
			for range time.Tick(sweepEvery) {
				sweepIdle(cacheIdle)
			}
		}()
	})
}

func sweepIdle(idle time.Duration) {
	dropped := false
	gameMapFiles.Lock()
	if gameMapFiles.body != nil && time.Since(gameMapFiles.used) >= idle {
		gameMapFiles.body, gameMapFiles.game = nil, ""
		dropped = true
	}
	gameMapFiles.Unlock()
	mapBases.Lock()
	if mapBases.base != nil && time.Since(mapBases.used) >= idle {
		mapBases.base, mapBases.key = nil, ""
		dropped = true
	}
	mapBases.Unlock()
	gameImage.Lock()
	if gameImage.body != nil && time.Since(gameImage.used) >= idle {
		gameImage.body, gameImage.game, gameImage.tag = nil, "", ""
		dropped = true
	}
	gameImage.Unlock()
	projectMapImages.Lock()
	for key, entry := range projectMapImages.entries {
		if time.Since(entry.used) >= idle {
			delete(projectMapImages.entries, key)
			dropped = true
		}
	}
	projectMapImages.Unlock()
	if dropped {
		debug.FreeOSMemory()
	}
}
