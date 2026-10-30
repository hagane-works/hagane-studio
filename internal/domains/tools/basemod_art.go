package tools

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/gamefiles"
	"haganeworks/internal/project"
)

func (b *basemod) mod(w http.ResponseWriter, r *http.Request) *gamefiles.Game {
	p, err := b.open(r)
	if err != nil {
		apiutil.Fail(w, err)
		return nil
	}
	dep := r.PathValue("dep")
	mod := b.mods.files(p, dep)
	if mod == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "no base-mod folder for dependency '" + dep + "'"})
	}
	return mod
}

func (b *basemod) gfx(w http.ResponseWriter, r *http.Request) {
	mod := b.mod(w, r)
	if mod == nil {
		return
	}
	name := r.PathValue("name")
	texture := mod.Sprites()[name]
	if texture == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "base-mod gfx name not found: " + name})
		return
	}
	png, err := mod.PNG(texture)
	if errors.Is(err, fs.ErrNotExist) {
		if game, gameErr := gamefiles.Open(b.game); gameErr == nil {
			png, err = game.PNG(texture)
		}
	}
	writeBasemodPicture(w, png, err, texture)
}

func (b *basemod) art(w http.ResponseWriter, r *http.Request) {
	mod := b.mod(w, r)
	if mod == nil {
		return
	}
	rel := strings.ReplaceAll(r.PathValue("rel"), "\\", "/")
	if rel == "" || basemodHasDotDot(rel) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid rel_path"})
		return
	}
	if !strings.Contains(rel, "/") {
		resolved := gamefiles.Derived(mod, basemodBasenamesKey{}, basemodBasenames)[basemodStem(rel)]
		if resolved == "" {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "base-mod art bare-name not found in index: " + rel})
			return
		}
		rel = resolved
	}
	png, err := mod.PNG(rel)
	writeBasemodPicture(w, png, err, rel)
}

func (b *basemod) flag(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tag, ideology := r.PathValue("tag"), q.Get("ideology")
	size := "large"
	if q.Has("size") {
		size = q.Get("size")
	}
	if !basemodFlagTagRe.MatchString(tag) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid flag tag"})
		return
	}
	if ideology != "" && !basemodFlagIdeologyRe.MatchString(ideology) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid ideology"})
		return
	}
	sizeDir, ok := basemodFlagSizeDirs[strings.ToLower(strings.TrimSpace(size))]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid flag size: '" + size + "' (large|medium|small)"})
		return
	}
	mod := b.mod(w, r)
	if mod == nil {
		return
	}
	candidates := []string{"gfx/flags/" + sizeDir + tag + ".tga"}
	if ideology != "" {
		candidates = append([]string{"gfx/flags/" + sizeDir + tag + "_" + ideology + ".tga"}, candidates...)
	}
	for _, rel := range candidates {
		png, err := mod.PNG(rel)
		if !errors.Is(err, fs.ErrNotExist) {
			writeBasemodPicture(w, png, err, rel)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"detail": "base-mod flag not found: " + tag})
}

func writeBasemodPicture(w http.ResponseWriter, png []byte, err error, rel string) {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "base-mod file missing: " + rel})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "failed to decode base-mod picture: " + rel})
	default:

		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(png)
	}
}

func basemodHasDotDot(rel string) bool {
	for _, segment := range strings.Split(rel, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

type basemodBasenamesKey struct{}

func basemodBasenames(mod *gamefiles.Game) map[string]string {
	rank := func(texture string) int {
		lower := strings.ToLower(texture)
		switch {
		case strings.HasPrefix(lower, "gfx/leaders/"):
			return 0
		case strings.HasPrefix(lower, "gfx/"):
			return 1
		}
		return 2
	}
	best := map[string]string{}
	for _, texture := range mod.Sprites() {
		stem := basemodStem(texture)
		current, taken := best[stem]
		if stem != "" && (!taken || rank(texture) < rank(current) || rank(texture) == rank(current) && texture < current) {
			best[stem] = texture
		}
	}
	return best
}

func basemodStem(rel string) string {
	name := path.Base(rel)
	return strings.ToLower(strings.TrimSuffix(name, path.Ext(name)))
}

type basemodMods struct {
	mu     sync.Mutex
	opened map[string]*gamefiles.Game
}

func (m *basemodMods) files(p *project.Project, dep string) *gamefiles.Game {
	key := basemodKey(dep)
	if key == "" {
		return nil
	}
	folders := p.BaseMods()
	folder := ""
	for _, name := range basemodDependencies(p) {
		if folders[name] != "" && basemodKey(name) == key {
			folder = folders[name]
			break
		}
	}
	if folder == "" {
		return nil
	}
	if info, err := os.Stat(folder); err != nil || !info.IsDir() {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	files, ok := m.opened[folder]
	if !ok {
		if m.opened == nil {
			m.opened = map[string]*gamefiles.Game{}
		}
		files = gamefiles.Mod(os.DirFS(folder))
		m.opened[folder] = files
	}
	return files
}
