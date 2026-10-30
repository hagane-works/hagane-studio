package gameview

import (
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/loc"
)

const languagesKept = 2

type languageEntry struct {
	ready chan struct{}
	text  map[string]string
	used  time.Time
}

type languageCache struct {
	sync.Mutex
	entries map[string]*languageEntry
}

var vanillaLanguages = &languageCache{entries: map[string]*languageEntry{}}

func (c *languageCache) text(layer *gameLayer, identity, lang string) map[string]string {
	key := identity + "|" + lang
	c.Lock()
	entry := c.entries[key]
	owner := entry == nil
	if owner {
		entry = &languageEntry{ready: make(chan struct{})}
		c.entries[key] = entry
	}
	entry.used = time.Now()
	c.Unlock()
	if owner {
		started := time.Now()
		entry.text = readLanguage(layer.roots, lang)
		log.Printf("[gameview] base game %s text: %d keys in %s", lang, len(entry.text), time.Since(started).Round(time.Millisecond))
		c.Lock()
		c.trimLocked()
		c.Unlock()
		close(entry.ready)
		startSweeper()
	}
	<-entry.ready
	return entry.text
}

func (c *languageCache) trimLocked() {
	if len(c.entries) <= languagesKept {
		return
	}
	keys := make([]string, 0, len(c.entries))
	for key := range c.entries {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(a, b int) bool { return c.entries[keys[a]].used.Before(c.entries[keys[b]].used) })
	for _, key := range keys[:len(keys)-languagesKept] {
		select {
		case <-c.entries[key].ready:
			delete(c.entries, key)
		default:
		}
	}
}

func (c *languageCache) dropIdle(idle time.Duration) bool {
	c.Lock()
	defer c.Unlock()
	dropped := false
	for key, entry := range c.entries {
		select {
		case <-entry.ready:
			if time.Since(entry.used) >= idle {
				delete(c.entries, key)
				dropped = true
			}
		default:
		}
	}
	return dropped
}

func (c *languageCache) drop() {
	c.Lock()
	defer c.Unlock()
	for key, entry := range c.entries {
		select {
		case <-entry.ready:
			delete(c.entries, key)
		default:
		}
	}
}

func readLanguage(roots []contentRoot, lang string) map[string]string {
	header := "l_" + lang
	text := map[string]string{}
	var replace []func()
	for _, root := range roots {
		fsys := root.fsys
		_ = fs.WalkDir(fsys, "localisation/"+lang, func(rel string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.EqualFold(pathExt(rel), ".yml") {
				return nil
			}
			read := func() {
				body, err := fs.ReadFile(fsys, rel)
				if err != nil {
					return
				}
				decoded, ok := gfx.DecodeScriptText(body)
				if !ok {
					return
				}
				language, entries, _ := loc.ParseLocText(decoded)
				if language != header {
					return
				}
				for k, v := range entries {
					text[k] = v
				}
			}
			if strings.Contains(strings.ToLower(rel), "/replace/") {
				replace = append(replace, read)
			} else {
				read()
			}
			return nil
		})
	}
	for _, read := range replace {
		read()
	}
	if len(text) == 0 {
		return nil
	}
	return text
}

func pathExt(name string) string {
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 && !strings.Contains(name[dot:], "/") {
		return name[dot:]
	}
	return ""
}

func (s *Service) inLanguage(r *http.Request, index *guirender.Index) *guirender.Index {
	lang := requestLanguage(r.URL.Query().Get("lang"))
	if lang == "" || index == nil {
		return index
	}
	var base map[string]string
	if layer, identity := s.gameLayerQuiet(); layer != nil {
		base = vanillaLanguages.text(layer, identity, lang)
	}
	return index.InLanguage(lang, base)
}

func requestLanguage(raw string) string {
	lang := guirender.GameLanguage(raw)
	if lang == "english" {
		return ""
	}
	return lang
}
