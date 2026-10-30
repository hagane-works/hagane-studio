package guirender

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
)

const VanillaManifestPath = "vanilla_textures.json"

type ExternalTexture struct {
	Width  int
	Height int
}

type VanillaManifest struct {
	GameVersion string           `json:"game_version"`
	Textures    map[string][]int `json:"textures"`
}

func NewVanillaIndex(files map[string][]byte, baseURL string) (*Index, error) {
	rawManifest, ok := files[VanillaManifestPath]
	if !ok {
		return nil, fmt.Errorf("vanilla pack is missing %s", VanillaManifestPath)
	}
	var manifest VanillaManifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return nil, fmt.Errorf("vanilla texture manifest: %w", err)
	}

	scripts := make(map[string][]byte, len(files))
	for name, body := range files {
		if name == VanillaManifestPath {
			continue
		}
		scripts[name] = body
	}
	idx, err := newIndex(scripts, "vanilla")
	if err != nil {
		return nil, err
	}
	idx.version = manifest.GameVersion
	idx.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	idx.external = make(map[string]ExternalTexture, len(manifest.Textures))
	for rel, size := range manifest.Textures {
		if len(size) != 2 || size[0] <= 0 || size[1] <= 0 {
			continue
		}
		idx.external[foldTexturePath(rel)] = ExternalTexture{Width: size[0], Height: size[1]}
	}
	idx.externalLoose = buildLooseTextureIndex(idx.external)
	idx.masking = &maskPlan{source: idx}
	return idx, nil
}

func (idx *Index) GameVersion() string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if layer.version != "" {
			return layer.version
		}
	}
	return ""
}

func (idx *Index) Fallback() *Index {
	if idx == nil {
		return nil
	}
	return idx.fallback
}

func (idx *Index) TotalWindowCount() int {
	seen := map[string]struct{}{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for name := range layer.windows {
			seen[name] = struct{}{}
		}
	}
	return len(seen)
}

func (idx *Index) TotalSpriteCount() int {
	seen := map[string]struct{}{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for name := range layer.sprites {
			seen[name] = struct{}{}
		}
	}
	return len(seen)
}

func (idx *Index) VanillaSpriteCount() int {
	count := 0
	for layer := idx; layer != nil; layer = layer.fallback {
		if layer.source == "vanilla" {
			count += len(layer.sprites)
		}
	}
	return count
}

func foldTexturePath(value string) string {
	rel := strings.TrimPrefix(strings.ReplaceAll(value, "\\", "/"), "/")
	for strings.Contains(rel, "//") {
		rel = strings.ReplaceAll(rel, "//", "/")
	}
	return strings.ToLower(rel)
}

func buildLooseTextureIndex(external map[string]ExternalTexture) map[string]string {
	paths := make([]string, 0, len(external))
	for rel := range external {
		if !strings.HasPrefix(rel, "gfx/interface/") {
			continue
		}
		switch path.Ext(rel) {
		case ".dds", ".png", ".tga":
		default:
			continue
		}
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	out := make(map[string]string, len(paths)*2)
	for _, rel := range paths {
		for _, name := range looseTextureNames(rel) {
			if _, taken := out[name]; !taken {
				out[name] = rel
			}
		}
	}
	return out
}

func (idx *Index) spriteRecordFor(name string) (spriteRecord, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.sprites[name]; ok {
			return record, true
		}
	}
	return spriteRecord{}, false
}

func (idx *Index) locValue(key string) (string, bool) {

	if idx.lang != "" {
		for layer := idx; layer != nil; layer = layer.fallback {
			if value, ok := layer.langLoc[key]; ok {
				return value, true
			}
		}
	}
	for layer := idx; layer != nil; layer = layer.fallback {
		if value, ok := layer.loc[key]; ok {
			return value, true
		}
	}
	return "", false
}

func (idx *Index) fontRecordFor(key string) (fontRecord, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.fonts[key]; ok {
			return record, true
		}
	}
	return fontRecord{}, false
}

func (idx *Index) sourceOfFile(file string) string {
	clean := strings.ToLower(strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "/"))
	for layer := idx; layer != nil; layer = layer.fallback {
		if _, ok := layer.filesFolded[clean]; ok {
			return layer.source
		}
	}
	return idx.source
}

func (idx *Index) SetVanillaTextureURL(fn func(rel string) string) { idx.textureURL = fn }

func (idx *Index) vanillaTextureURL(rel string) string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if layer.textureURL == nil && layer.baseURL == "" {
			continue
		}
		if layer.textureURL != nil {
			if clean := VanillaTexturePath(rel); clean != "" {
				if url := layer.textureURL(clean); url != "" {
					return url
				}
			}
		}
		return VanillaTextureURL(layer.baseURL, rel)
	}
	return ""
}

func (idx *Index) vanillaBase() string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if layer.baseURL != "" {
			return layer.baseURL
		}
	}
	return ""
}

func (idx *Index) textureSize(value string) (width, height int, externalRel string, ok bool) {
	variants := textureVariants(value)
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, variant := range variants {
			if actual, found := layer.filesFolded[variant]; found {
				if w, h, decoded := imageSize(actual, layer.files[actual]); decoded {
					return w, h, "", true
				}
			}
		}
		for _, variant := range variants {
			if meta, found := layer.external[variant]; found {
				return meta.Width, meta.Height, variant, true
			}
		}
	}
	return 0, 0, "", false
}

func (idx *Index) looseTextureSize(ref string) (width, height int, externalRel string, ok bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if actual, body, found := layer.resolveLooseTexture(ref); found {
			if w, h, decoded := imageSize(actual, body); decoded {
				return w, h, "", true
			}
		}
		if rel, found := layer.looseExternal(ref); found {
			meta := layer.external[rel]
			return meta.Width, meta.Height, rel, true
		}
	}

	if file, ok := idx.ScreenPictureFile(ref); ok {
		return idx.textureSize(file)
	}
	return 0, 0, "", false
}

func (idx *Index) looseExternal(ref string) (string, bool) {
	if len(idx.externalLoose) == 0 {
		return "", false
	}
	wanted := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ref)), "gfx_")
	for _, key := range []string{wanted, "gfx_" + wanted} {
		if rel, ok := idx.externalLoose[key]; ok {
			return rel, true
		}
	}
	return "", false
}

func (idx *Index) ExternalSpriteTexture(name string) (string, bool) {
	if record, ok := idx.spriteRecordFor(name); ok {
		for _, candidate := range record.TextureFiles {
			if w, h, rel, found := idx.textureSize(candidate); found {
				_, _ = w, h
				return rel, rel != ""
			}
		}
		return "", false
	}
	_, _, rel, found := idx.looseTextureSize(name)
	return rel, found && rel != ""
}

func VanillaTextureURL(base, rel string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	clean := VanillaTexturePath(rel)
	if base == "" || clean == "" {
		return ""
	}
	return base + "/" + clean
}

func VanillaTexturePath(rel string) string {
	if strings.TrimSpace(rel) == "" {
		return ""
	}
	clean := foldTexturePath(rel)
	if ext := path.Ext(clean); ext != ".png" {
		clean = strings.TrimSuffix(clean, ext) + ".png"
	}
	return clean
}

func (idx *Index) VanillaSpriteURL(name string) string {
	rel, ok := idx.ExternalSpriteTexture(name)
	if !ok {
		return ""
	}
	return idx.vanillaTextureURL(rel)
}

func (idx *Index) SpritesPayload(names []string) map[string]any {
	out := map[string]any{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out[name] = idx.spritePayload(name)
	}
	return out
}

func (idx *Index) Localise(key string) (string, bool) {
	value, ok := idx.locValue(key)
	if !ok {
		return "", false
	}

	value = idx.resolveLocRefs(value, 0, map[string]bool{key: true})
	return cleanMarkup(idx.resolveScripted(value, "", "")), true
}
