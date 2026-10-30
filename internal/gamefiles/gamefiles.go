package gamefiles

import (
	"container/list"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/importassets"
)

type Root struct {
	Name string
	FS   fs.FS
}

func Roots(fsys fs.FS) []Root {
	roots := []Root{{Name: "", FS: fsys}}
	for _, group := range []string{"dlc", "integrated_dlc"} {
		entries, err := fs.ReadDir(fsys, group)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if sub, err := fs.Sub(fsys, group+"/"+entry.Name()); err == nil {
				roots = append(roots, Root{Name: group + "/" + entry.Name(), FS: sub})
			}
		}
	}
	return roots
}

type Game struct {
	fsys  fs.FS
	roots []Root

	spritesOnce sync.Once
	sprites     map[string]string

	pngs *pngCache

	derivedMu sync.Mutex
	derived   map[any]*derivedValue
}

const MaxFile = 128 << 20

var (
	gamesMu sync.Mutex
	games   = map[string]*Game{}
)

func For(fsys fs.FS) *Game {
	id := Identity(fsys)
	gamesMu.Lock()
	defer gamesMu.Unlock()
	if g, ok := games[id]; ok {
		return g
	}
	g := newGame(fsys, Roots(fsys))

	clear(games)
	games[id] = g
	return g
}

func Mod(fsys fs.FS) *Game { return newGame(fsys, []Root{{Name: "", FS: fsys}}) }

func newGame(fsys fs.FS, roots []Root) *Game {
	return &Game{fsys: fsys, roots: roots, pngs: newPNGCache(512, 64<<20)}
}

func Open(open func() (fs.FS, error)) (*Game, error) {
	if open == nil {
		return nil, errNotSetUp
	}
	fsys, err := open()
	if err != nil {
		return nil, err
	}
	if fsys == nil {
		return nil, errNotSetUp
	}
	return For(fsys), nil
}

var errNotSetUp = errors.New("the game folder is not set up")

func Identity(fsys fs.FS) string {
	id := fmt.Sprintf("%T", fsys)
	value := reflect.ValueOf(fsys)
	switch value.Kind() {
	case reflect.String:
		id += ":" + value.String()
	case reflect.Map, reflect.Pointer, reflect.Slice, reflect.Func:
		id += fmt.Sprintf("@%x", value.Pointer())
	default:
		id += fmt.Sprintf("%v", fsys)
	}
	if info, err := fs.Stat(fsys, "launcher-settings.json"); err == nil {
		id += fmt.Sprintf("|%d|%d", info.Size(), info.ModTime().UnixNano())
	}
	return id
}

func (g *Game) FS() fs.FS { return g.fsys }

func (g *Game) Roots() []Root { return g.roots }

func (g *Game) ReadFile(rel string) ([]byte, error) {
	rel = Clean(rel)
	for i := len(g.roots) - 1; i >= 0; i-- {
		data, err := readLimited(g.roots[i].FS, rel)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("%s: %w", rel, fs.ErrNotExist)
}

func (g *Game) Files(dir string, recursive bool, exts ...string) []string {
	dir = Clean(dir)
	seen := map[string]bool{}
	for _, root := range g.roots {
		walk := func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() {
				if name != dir && !recursive {
					return fs.SkipDir
				}
				return nil
			}
			if hasExt(name, exts) {
				seen[name] = true
			}
			return nil
		}
		_ = fs.WalkDir(root.FS, dir, walk)
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func hasExt(name string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	lower := strings.ToLower(name)
	for _, ext := range exts {
		if strings.HasSuffix(lower, strings.ToLower(ext)) {
			return true
		}
	}
	return false
}

func (g *Game) Sprites() map[string]string {
	g.spritesOnce.Do(func() {

		var texts []string
		for _, root := range g.roots {
			var names []string
			_ = fs.WalkDir(root.FS, "interface", func(name string, entry fs.DirEntry, err error) error {
				if err == nil && !entry.IsDir() && hasExt(name, []string{".gfx"}) {
					names = append(names, name)
				}
				return nil
			})
			sort.Strings(names)
			for _, name := range names {
				raw, err := readLimited(root.FS, name)
				if err != nil {
					continue
				}
				if text, ok := gfx.DecodeScriptText(raw); ok {
					texts = append(texts, text)
				}
			}
		}
		sprites := gfx.ParseMultipleGfxFiles(texts)
		for name, texture := range sprites {
			sprites[name] = Clean(texture)
		}
		g.sprites = sprites
	})
	return g.sprites
}

func (g *Game) Texture(rel string) (string, bool) {
	rel = Clean(rel)
	if rel == "" {
		return "", false
	}
	candidates := []string{rel}
	stem := strings.TrimSuffix(rel, path.Ext(rel))
	for _, ext := range []string{".dds", ".tga", ".png"} {
		if candidate := stem + ext; candidate != rel {
			candidates = append(candidates, candidate)
		}
	}
	for _, candidate := range candidates {
		for i := len(g.roots) - 1; i >= 0; i-- {
			if info, err := fs.Stat(g.roots[i].FS, candidate); err == nil && !info.IsDir() {
				return candidate, true
			}
		}
	}
	return "", false
}

func (g *Game) PNG(rel string) ([]byte, error) {
	found, ok := g.Texture(rel)
	if !ok {
		return nil, fmt.Errorf("%s: %w", Clean(rel), fs.ErrNotExist)
	}
	if png, ok := g.pngs.get(found); ok {
		return png, nil
	}
	raw, err := g.ReadFile(found)
	if err != nil {
		return nil, err
	}
	png, err := importassets.NormalizePNG(raw, found, "")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", found, err)
	}
	g.pngs.put(found, png)
	return png, nil
}

func (g *Game) SpritePNG(sprite string) ([]byte, error) {
	texture, ok := g.Sprites()[sprite]
	if !ok {
		return nil, fmt.Errorf("sprite %s: %w", sprite, fs.ErrNotExist)
	}
	return g.PNG(texture)
}

func Clean(rel string) string {
	rel = strings.TrimSpace(strings.ReplaceAll(rel, "\\", "/"))
	rel = path.Clean("/" + rel)
	return strings.TrimPrefix(rel, "/")
}

func readLimited(fsys fs.FS, rel string) ([]byte, error) {
	if info, err := fs.Stat(fsys, rel); err != nil {
		return nil, err
	} else if info.IsDir() {
		return nil, fmt.Errorf("%s: %w", rel, fs.ErrNotExist)
	} else if info.Size() > MaxFile {
		return nil, fmt.Errorf("%s is larger than %d bytes", rel, MaxFile)
	}
	return fs.ReadFile(fsys, rel)
}

type pngCache struct {
	mu         sync.Mutex
	maxEntries int
	maxBytes   int
	bytes      int
	order      *list.List
	entries    map[string]*list.Element
}

type pngEntry struct {
	key  string
	data []byte
}

func newPNGCache(maxEntries, maxBytes int) *pngCache {
	return &pngCache{maxEntries: maxEntries, maxBytes: maxBytes, order: list.New(), entries: map[string]*list.Element{}}
}

func (c *pngCache) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	element, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(element)
	return element.Value.(*pngEntry).data, true
}

func (c *pngCache) put(key string, data []byte) {
	if len(data) > c.maxBytes {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if element, ok := c.entries[key]; ok {
		c.bytes -= len(element.Value.(*pngEntry).data)
		c.order.Remove(element)
	}
	c.entries[key] = c.order.PushFront(&pngEntry{key: key, data: data})
	c.bytes += len(data)
	for c.order.Len() > c.maxEntries || c.bytes > c.maxBytes {
		oldest := c.order.Back()
		entry := oldest.Value.(*pngEntry)
		c.order.Remove(oldest)
		delete(c.entries, entry.key)
		c.bytes -= len(entry.data)
	}
}
