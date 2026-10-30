package guirender

import (
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/paradox"
)

type FrontendPicture struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`

	Locale string `json:"locale,omitempty"`

	Overridden bool `json:"overridden,omitempty"`
}

func (idx *Index) FrontendBackgrounds() []FrontendPicture {
	var layers []*Index
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
	}
	seen := map[string]bool{}
	var names []string
	locales := map[string]string{}
	for i := len(layers) - 1; i >= 0; i-- {
		var files []string
		for name := range layers[i].files {
			lower := strings.ToLower(name)
			if strings.HasPrefix(lower, "common/frontend/backgrounds/") && strings.HasSuffix(lower, ".txt") {
				files = append(files, name)
			}
		}
		sort.Strings(files)
		for _, file := range files {
			text, ok := gfx.DecodeScriptText(layers[i].files[file])
			if !ok {
				continue
			}
			root, _, err := paradox.Parse(text)
			if err != nil {
				continue
			}
			for _, entry := range root.Entries {
				if entry == nil || entry.Key == nil || entry.Block == nil {
					continue
				}
				name := strings.TrimSpace(*entry.Key)
				if name == "" || seen[strings.ToLower(name)] {
					continue
				}
				seen[strings.ToLower(name)] = true
				names = append(names, name)
				locales[name] = ciScalar(entry.Block.Entries, "locale")
			}
		}
	}
	out := []FrontendPicture{}
	for _, name := range names {
		if picture, ok := idx.loadingPicture(name); ok {
			picture.Locale = locales[name]
			out = append(out, picture)
		}
	}
	return out
}

func (idx *Index) LoadingPictures() []FrontendPicture {
	out := []FrontendPicture{}
	for n := 1; n <= 99; n++ {
		picture, ok := idx.loadingPicture(fmt.Sprintf("load_%d", n))
		if !ok {
			break
		}
		out = append(out, picture)
	}
	return append(out, idx.FrontendBackgrounds()...)
}

func (idx *Index) loadingPicture(name string) (FrontendPicture, bool) {
	w, h, rel, ok := idx.textureSize(screenPictureDir + name + ".dds")
	if !ok {
		return FrontendPicture{}, false
	}
	if override, replaced := idx.spriteOverride(ScreenPictureSprite(name)); replaced {
		return FrontendPicture{Name: name, URL: override.URL, Width: override.Width, Height: override.Height, Overridden: true}, true
	}
	if rel == "" {
		return FrontendPicture{}, false
	}
	url := idx.vanillaTextureURL(rel)
	if url == "" {
		return FrontendPicture{}, false
	}
	return FrontendPicture{Name: name, URL: url, Width: w, Height: h}, true
}

const screenPictureDir = "gfx/loadingscreens/"

func ScreenPictureSprite(name string) string { return "GFX_" + name }

func (idx *Index) ScreenPictureFile(sprite string) (string, bool) {
	stem, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(sprite)), "gfx_")
	if !ok || stem == "" || strings.ContainsAny(stem, `/\. `) {
		return "", false
	}
	if _, declared := idx.spriteRecordFor(sprite); declared {
		return "", false
	}
	file := screenPictureDir + stem + ".dds"
	if _, _, _, found := idx.textureSize(file); !found {
		return "", false
	}
	return file, true
}
