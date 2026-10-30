package build

import (
	"context"
	"encoding/json"
	"path"
	"strings"

	"haganeworks/internal/gameindex"
)

func buildCollisionTable(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	icons, bookmarks := map[string]bool{}, map[string]bool{}
	add := func(set map[string]bool, p string) {
		if p = strings.ToLower(strings.ReplaceAll(p, "\\", "/")); p != "" {
			set[p] = true
		}
	}
	var focus struct {
		Icons []focusIcon `json:"icons"`
	}
	_ = json.Unmarshal(in.Dataset(gameindex.FocusIcons), &focus)
	for _, icon := range focus.Icons {
		add(icons, icon.Texturefile)
	}
	var tech struct {
		Paths map[string]string `json:"paths"`
	}
	_ = json.Unmarshal(in.Dataset(gameindex.TechIconPaths), &tech)
	for _, p := range tech.Paths {
		add(icons, p)
	}
	for _, rel := range baseFiles(in, "gfx/interface/decisions", ".dds") {
		base := path.Base(rel)
		add(icons, "gfx/interface/decisions/"+strings.TrimSuffix(base, path.Ext(base))+".dds")
	}
	for _, rel := range baseFiles(in, "common/bookmarks", ".txt") {
		add(bookmarks, rel)
	}
	return marshal(map[string]any{"icons": sortedKeys(icons), "bookmarks": sortedKeys(bookmarks)})
}
