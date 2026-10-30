package gameview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sync"
)

var sizeCacheDir = func() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Hagane Works", "gameview")
}

const sizeCacheFormat = 1

type sizeCache struct {
	mu      sync.Mutex
	path    string
	entries map[string][4]int64
	changed bool
}

type sizeCacheFile struct {
	Format  int                 `json:"format"`
	Entries map[string][4]int64 `json:"entries"`
}

func loadSizeCache(fsys fs.FS) *sizeCache {
	cache := &sizeCache{entries: map[string][4]int64{}}
	value := reflect.ValueOf(fsys)
	base := sizeCacheDir()
	if value.Kind() != reflect.String || base == "" {
		return cache
	}
	sum := sha256.Sum256([]byte(filepath.Clean(value.String())))
	cache.path = filepath.Join(base, "textures-"+hex.EncodeToString(sum[:8])+".json")
	raw, err := os.ReadFile(cache.path)
	if err != nil {
		return cache
	}
	var file sizeCacheFile
	if json.Unmarshal(raw, &file) == nil && file.Format == sizeCacheFormat && file.Entries != nil {
		cache.entries = file.Entries
	}
	return cache
}

func (c *sizeCache) lookup(key string, stamp [2]int64) ([2]int, bool) {
	if stamp == ([2]int64{}) {
		return [2]int{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || entry[0] != stamp[0] || entry[1] != stamp[1] {
		return [2]int{}, false
	}
	return [2]int{int(entry[2]), int(entry[3])}, true
}

func (c *sizeCache) store(key string, stamp [2]int64, size [2]int) {
	if stamp == ([2]int64{}) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := [4]int64{stamp[0], stamp[1], int64(size[0]), int64(size[1])}
	if c.entries[key] != entry {
		c.entries[key] = entry
		c.changed = true
	}
}

func (c *sizeCache) save() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.path == "" || !c.changed {
		return
	}
	raw, err := json.Marshal(sizeCacheFile{Format: sizeCacheFormat, Entries: c.entries})
	if err != nil || os.MkdirAll(filepath.Dir(c.path), 0o755) != nil {
		return
	}
	tmp := c.path + ".tmp"
	if os.WriteFile(tmp, raw, 0o644) == nil && os.Rename(tmp, c.path) == nil {
		c.changed = false
	}
}
