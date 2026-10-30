package gameview

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/png"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
	emitscripts "haganeworks/internal/hoi4/emit/scripts"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const maxPictureBytes = 10 << 20

type projectSprite struct {
	Key  string
	Kind string
}

func str(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}

func entityPictures(p *project.Project) (map[string]map[int64]string, error) {
	rows, err := p.Rows("entity_obs_paths")
	if err != nil {
		return nil, err
	}
	out := map[string]map[int64]string{}
	for _, row := range rows {
		kind, key := str(row["entity_type"]), strings.TrimSpace(str(row["obs_key"]))
		if kind == "" || key == "" {
			continue
		}
		if out[kind] == nil {
			out[kind] = map[int64]string{}
		}
		out[kind][project.Int64(row["entity_id"])] = key
	}
	return out, nil
}

func projectSpriteAssets(p *project.Project) (map[string]projectSprite, error) {
	out := map[string]projectSprite{}
	add := func(name, key, kind, countryTag string) {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(key) == "" {
			return
		}
		for _, spriteName := range projectSpriteNames(name, kind, countryTag) {
			out[spriteName] = projectSprite{Key: key, Kind: kind}
		}
	}
	icons, err := p.Rows("project_obs_icons")
	if err != nil {
		return nil, err
	}
	for _, row := range icons {
		add(str(row["gfx_name"]), str(row["obs_key"]), "focus_icon", "")
	}
	pictures, err := entityPictures(p)
	if err != nil {
		return nil, err
	}
	for _, source := range []struct{ table, column, kind string }{
		{"ideas", "picture", "idea_picture"},
		{"decisions", "icon", "decision_icon"},
		{"events", "picture", "event_picture"},
		{"technologies", "tech_id", "tech_icon"},
	} {
		rows, err := p.Rows(source.table)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if key, ok := pictures[source.kind][project.Int64(row["id"])]; ok {
				add(str(row[source.column]), key, source.kind, "")
			}
		}
	}
	record := p.Record()
	iconPaths, _ := record["ideology_icon_paths"].(map[string]any)
	assets, err := p.Rows("project_obs_assets")
	if err != nil {
		return nil, err
	}
	for _, row := range assets {
		name := str(row["gfx_name"])
		if str(row["asset_type"]) != "ideology_icon" || name == "" || strings.TrimSpace(str(iconPaths[name])) == "" {
			continue
		}
		add("GFX_ideology_"+name+"_group", str(row["obs_key"]), "ideology_icon", "")
	}
	characters, err := p.Rows("characters")
	if err != nil {
		return nil, err
	}
	projectTag := strings.TrimSpace(str(record["country_tag"]))
	for _, row := range characters {
		tag := strings.TrimSpace(str(row["country_tag"]))
		if tag == "" {
			tag = projectTag
		}
		if tag == "" {
			tag = "UNK"
		}
		for _, kind := range []string{"character_portrait", "character_advisor"} {
			if key, ok := pictures[kind][project.Int64(row["id"])]; ok {
				add(str(row["tag"]), key, kind, tag)
			}
		}
	}
	return out, nil
}

var spriteAssetsMemo = struct {
	sync.Mutex
	entries map[string]spriteAssetsEntry
}{entries: map[string]spriteAssetsEntry{}}

type spriteAssetsEntry struct {
	stamp  string
	assets map[string]projectSprite
}

func cachedSpriteAssets(p *project.Project) (map[string]projectSprite, error) {
	stamp, err := dataStamps.get(p)
	if err != nil {
		return projectSpriteAssets(p)
	}
	key := projectKey(p)
	spriteAssetsMemo.Lock()
	entry, ok := spriteAssetsMemo.entries[key]
	spriteAssetsMemo.Unlock()
	if ok && entry.stamp == stamp {
		return entry.assets, nil
	}
	assets, err := projectSpriteAssets(p)
	if err != nil {
		return nil, err
	}
	spriteAssetsMemo.Lock()
	if len(spriteAssetsMemo.entries) > 16 {
		spriteAssetsMemo.entries = map[string]spriteAssetsEntry{}
	}
	spriteAssetsMemo.entries[key] = spriteAssetsEntry{stamp: stamp, assets: assets}
	spriteAssetsMemo.Unlock()
	return assets, nil
}

func projectSpriteNames(name, kind, countryTag string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	switch kind {
	case "idea_picture":
		if strings.HasPrefix(name, "GFX_idea_") {
			return []string{name}
		}
		return []string{name, "GFX_idea_" + name}
	case "tech_icon":
		exported := emitmisc.TechIconSpriteName(name)
		preview := "GFX_" + name + "_medium"
		if exported == preview {
			return []string{exported}
		}
		return []string{exported, preview}
	case "character_portrait", "character_advisor":
		countryTag = strings.ToUpper(strings.TrimSpace(countryTag))
		if countryTag == "" {
			countryTag = "UNK"
		}
		slug := emitscripts.ExtractSlug(name, countryTag)

		if kind == "character_portrait" {
			return []string{
				emitscripts.PortraitSpriteName(countryTag, slug),
				"gfx/leaders/" + countryTag + "/" + emitscripts.PortraitFilename(countryTag, slug),
			}
		}
		return []string{
			emitscripts.AdvisorSpriteName(countryTag, slug),
			"gfx/interface/ideas/" + emitscripts.AdvisorIconFilename(countryTag, slug),
		}
	default:
		return []string{name}
	}
}

var storedSizes sync.Map

func storedPictureSize(p *project.Project, key string) ([2]int, bool) {
	path, err := p.Blobs().Path(key)
	if err != nil {
		return [2]int{}, false
	}
	if size, ok := storedSizes.Load(path); ok {
		return size.([2]int), true
	}
	body, err := p.Blobs().Get(key, maxPictureBytes)
	if err != nil {
		return [2]int{}, false
	}
	w, h, err := importassets.ImageDimensions(body, "picture.png")
	if err != nil || w <= 0 || h <= 0 {
		return [2]int{}, false
	}
	storedSizes.Store(path, [2]int{w, h})
	return [2]int{w, h}, true
}

func projectSpriteSizes(p *project.Project) (map[string][2]int, map[string]string) {
	assets, err := cachedSpriteAssets(p)
	if err != nil || len(assets) == 0 {
		return nil, nil
	}
	sizes := make(map[string][2]int, len(assets))
	versions := make(map[string]string, len(assets))
	for name, asset := range assets {
		size := importassets.CanonicalSizes[asset.Kind]
		sizes[name] = [2]int{size.Width, size.Height}

		if asset.Kind == "tech_icon" || asset.Kind == "event_picture" || asset.Kind == "ideology_icon" {
			if real, ok := storedPictureSize(p, asset.Key); ok {
				sizes[name] = real
			} else if asset.Kind == "tech_icon" {

				size, _ = importassets.TechIconSize(strings.TrimSuffix(strings.TrimPrefix(name, "GFX_"), "_medium"))
				sizes[name] = [2]int{size.Width, size.Height}
			}
		}
		versions[name] = keyVersion(asset.Key)
	}
	return sizes, versions
}

func keyVersion(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:6])
}

func spriteOverrideRows(p *project.Project) map[string]string {
	rows, err := p.Rows("project_obs_assets")
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for _, row := range rows {
		if str(row["asset_type"]) == "gui_sprite_override" {
			if key := strings.TrimSpace(str(row["obs_key"])); key != "" {
				out[str(row["gfx_name"])] = key
			}
		}
	}
	return out
}

func spriteOverrideRoute(projectID int64, name, key string) string {
	return fmt.Sprintf("/api/go/projects/%d/gui-render/sprite?name=%s&v=%s", projectID, url.QueryEscape(name), keyVersion(key))
}

func spriteOverrides(p *project.Project) map[string]guirender.SpriteOverride {
	keys := spriteOverrideRows(p)
	if len(keys) == 0 {
		return nil
	}
	out := make(map[string]guirender.SpriteOverride, len(keys))
	for basename, key := range keys {
		name, ok := spriteNameOfOverride(basename)
		if !ok {
			continue
		}
		path, err := p.Blobs().Path(key)
		if err != nil {
			continue
		}
		var dims [2]int
		if size, ok := storedSizes.Load(path); ok {
			dims = size.([2]int)
		} else {
			data, err := p.Blobs().Get(key, maxPictureBytes)
			if err != nil {
				continue
			}
			config, err := png.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				continue
			}
			dims = [2]int{config.Width, config.Height}
			storedSizes.Store(path, dims)
		}
		out[name] = guirender.SpriteOverride{Width: dims[0], Height: dims[1], URL: spriteOverrideRoute(p.ID(), name, key)}
	}
	return out
}

const rawHeaderBytes = 4096

func measureRawTextures(p *project.Project, textures []rawTexture) []guirender.ProjectRawTexture {
	out := make([]guirender.ProjectRawTexture, 0, len(textures))
	for _, texture := range textures {
		path, err := p.Blobs().Path(texture.ObjectKey)
		if err != nil {
			continue
		}
		var size [2]int
		if known, ok := storedSizes.Load(path); ok {
			size = known.([2]int)
		} else {
			head, err := readPrefix(path, rawHeaderBytes)
			if err != nil {
				continue
			}
			w, h, ok := guirender.TextureImageSize(rawFormatName(texture.Path, head), head)
			if !ok {
				continue
			}
			size = [2]int{w, h}
			storedSizes.Store(path, size)
		}
		version := texture.SHA
		if len(version) > 12 {
			version = version[:12]
		}
		if version == "" {
			version = keyVersion(texture.ObjectKey)
		}
		out = append(out, guirender.ProjectRawTexture{
			Path: texture.Path, Width: size[0], Height: size[1], Version: version, ObjectKey: texture.ObjectKey,
		})
	}
	return out
}

func readPrefix(path string, n int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, n))
}

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

func rawFormatName(name string, head []byte) string {
	switch {
	case bytes.HasPrefix(head, pngSignature):
		return "x.png"
	case bytes.HasPrefix(head, []byte("DDS ")):
		return "x.dds"
	}
	return name
}
