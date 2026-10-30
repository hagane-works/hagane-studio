package decisions

import (
	"errors"
	"io/fs"
	"log"
	"net/http"

	"haganeworks/internal/gamefiles"
	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/paradox"
)

type categoryIDsKey struct{}

func categoryIDs(game func() (fs.FS, error)) func() map[string]bool {
	return func() map[string]bool {
		g, err := gamefiles.Open(game)
		if err != nil {
			return nil
		}
		return gamefiles.Derived(g, categoryIDsKey{}, readCategoryIDs)
	}
}

func readCategoryIDs(g *gamefiles.Game) map[string]bool {
	ids := map[string]bool{}
	for _, name := range g.Files("common/decisions/categories", false, ".txt") {
		raw, err := g.ReadFile(name)
		if err != nil {
			continue
		}
		textValue, ok := gfx.DecodeScriptText(raw)
		if !ok {
			continue
		}
		ast, _, err := paradox.Parse(textValue)
		if err != nil || ast == nil {
			continue
		}
		for _, entry := range ast.Entries {
			if entry.Key != nil && entry.Block != nil && *entry.Key != "" {
				ids[*entry.Key] = true
			}
		}
	}
	return ids
}

func (h *handlers) serveGameSprite(w http.ResponseWriter, sprite string) bool {
	if sprite == "" {
		return false
	}
	g, err := gamefiles.Open(h.deps.Game)
	if err != nil {
		return false
	}
	png, err := g.SpritePNG(sprite)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Printf("decision sprite %s: %v", sprite, err)
		}
		return false
	}
	writeImage(w, png)
	return true
}
