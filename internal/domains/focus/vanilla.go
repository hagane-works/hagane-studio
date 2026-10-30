package focus

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/gamefiles"
	"haganeworks/internal/hoi4/gfx"
)

type vanillaFocusIcon struct {
	Name        string
	TextureFile string
	Category    *string
}

type vanillaIconIndex struct {
	icons []vanillaFocusIcon
	paths map[string]string
}

type iconIndexKey struct{}

func buildVanillaIconIndex(g *gamefiles.Game) *vanillaIconIndex {
	index := &vanillaIconIndex{paths: map[string]string{}}
	for _, file := range g.Files("interface", false, ".gfx") {
		base := strings.ToLower(path.Base(file))
		if !strings.Contains(base, "goals") || strings.Contains(base, "shine") {
			continue
		}
		raw, err := g.ReadFile(file)
		if err != nil {
			continue
		}
		text, ok := gfx.DecodeScriptText(raw)
		if !ok {
			continue
		}
		for name, texture := range gfx.ParseGfxMappings(text) {
			if !strings.HasPrefix(name, "GFX_focus_") && !strings.HasPrefix(name, "GFX_goal_") {
				continue
			}
			if strings.HasSuffix(name, "_shine") {
				continue
			}
			index.paths[name] = gamefiles.Clean(texture)
		}
	}
	names := make([]string, 0, len(index.paths))
	for name := range index.paths {
		names = append(names, name)
	}
	sort.Strings(names)
	focus, goal := "focus", "goal"
	for _, name := range names {
		category := &focus
		if strings.HasPrefix(name, "GFX_goal_") {
			category = &goal
		}
		index.icons = append(index.icons, vanillaFocusIcon{Name: name, TextureFile: index.paths[name], Category: category})
	}
	return index
}

func (h *handlers) vanillaIcons() (*gamefiles.Game, *vanillaIconIndex, error) {
	game, err := gamefiles.Open(h.deps.Game)
	if err != nil {
		return nil, nil, err
	}
	return game, gamefiles.Derived(game, iconIndexKey{}, buildVanillaIconIndex), nil
}

func (h *handlers) vanillaIcon(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("gfx")
	notFound := func(detail string) {
		fail(w, apiutil.NotFoundError(detail))
	}
	game, index, err := h.vanillaIcons()
	if err != nil {
		notFound("vanilla focus icon not found: " + name)
		return
	}
	rel, found := index.paths[name]
	if !found {
		rel, found = game.Sprites()[name]
	}
	if !found {
		notFound("vanilla focus icon not found: " + name)
		return
	}
	if rel == "" {
		notFound("vanilla focus icon has no texturefile: " + name)
		return
	}
	pngBytes, err := game.PNG(rel)
	if errors.Is(err, fs.ErrNotExist) {
		notFound("vanilla DDS file missing: " + rel)
		return
	}
	if err != nil {
		fail(w, statusError(http.StatusInternalServerError, "failed to decode DDS: "+rel))
		return
	}
	writePNG(w, pngBytes)
}

func writePNG(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
