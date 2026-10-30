package gameview

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"

	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const (
	pictureLimit = int64(64 << 20)

	pictureMaxSide   = 8192
	pictureMaxPixels = 16 * 1024 * 1024
)

type pictureContentError struct{ err error }

func (e pictureContentError) Error() string { return e.err.Error() }
func (e pictureContentError) Unwrap() error { return e.err }

func encodePNG(body []byte, name string) ([]byte, int, int, error) {
	w, h, ok := imageHeaderSize(path.Ext(name), body)
	if ok && (w > pictureMaxSide || h > pictureMaxSide || w*h > pictureMaxPixels) {
		return nil, 0, 0, pictureContentError{fmt.Errorf("%s is %dx%d, over the preview limit", name, w, h)}
	}
	picture, err := importassets.DecodeImage(body, name)
	if err != nil {
		return nil, 0, 0, pictureContentError{err}
	}
	var out bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&out, picture); err != nil {
		return nil, 0, 0, err
	}
	bounds := picture.Bounds()
	return out.Bytes(), bounds.Dx(), bounds.Dy(), nil
}

func writePNG(w http.ResponseWriter, body []byte, width, height int) {
	w.Header().Set("Content-Type", "image/png")
	if width > 0 && height > 0 {
		w.Header().Set("X-Image-Width", fmt.Sprint(width))
		w.Header().Set("X-Image-Height", fmt.Sprint(height))
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s *Service) vanillaPNG(rel string) ([]byte, error) {
	layer, err := s.gameLayer()
	if err != nil {
		return nil, err
	}

	cacheKey := layer.version + "|" + fmt.Sprintf("%p", layer) + "|" + rel
	if body, ok := vanillaPNGs.get(cacheKey); ok {
		return body, nil
	}
	stem := strings.TrimSuffix(rel, path.Ext(rel))
	for _, ext := range []string{".dds", ".tga", ".png"} {
		body, actual, err := layer.readFile(stem+ext, pictureLimit)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out, _, _, err := encodePNG(body, actual)
		if err != nil {
			return nil, err
		}
		vanillaPNGs.put(cacheKey, out)
		return out, nil
	}
	return nil, fs.ErrNotExist
}

func (s *Service) handleVanillaAsset(w http.ResponseWriter, r *http.Request) {
	rel := r.PathValue("path")
	safe, err := exportengine.SafePath(rel)
	if err != nil || safe != rel || path.Ext(safe) != ".png" {
		detail(w, http.StatusNotFound, "Not Found")
		return
	}
	body, err := s.vanillaPNG(strings.ToLower(safe))
	if err != nil {
		var content pictureContentError
		switch {
		case errors.As(err, &content):
			detail(w, http.StatusUnprocessableEntity, "base-game picture is invalid")
		case errors.Is(err, fs.ErrNotExist):
			detail(w, http.StatusNotFound, "Not Found")
		default:
			detail(w, http.StatusServiceUnavailable, "base-game picture unavailable: "+err.Error())
		}
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive, noimageindex")
	writePNG(w, body, 0, 0)
}

func (s *Service) spriteRead(w http.ResponseWriter, r *http.Request, p *project.Project) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if !validSelector(name) {
		detail(w, http.StatusUnprocessableEntity, "name is required")
		return
	}

	if key, ok := spriteOverrideRows(p)[guiSpriteOverrideName(name)]; ok {
		if body, err := p.Blobs().Get(key, maxPictureBytes); err == nil {
			writePNG(w, body, 0, 0)
			return
		}
	}

	assets, err := cachedSpriteAssets(p)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "project sprite store unavailable")
		return
	}
	if asset, ok := assets[name]; ok {
		if body, err := p.Blobs().Get(asset.Key, maxPictureBytes); err == nil {
			writePNG(w, body, 0, 0)
			return
		}
	}
	index, found, err := s.loadIndex(r.Context(), p)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
		return
	}
	if !found {
		detail(w, http.StatusNotFound, "sprite not resolved: "+name)
		return
	}
	if body, width, height, ok := index.SpritePNG(name); ok {
		writePNG(w, body, width, height)
		return
	}

	if texture, raw := index.ProjectRawSpriteTexture(name); raw {
		body, err := rawTexturePNG(p, texture)
		if err != nil {
			log.Printf("[gameview] sprite %s from raw file %s: %v", name, texture.Path, err)
			var content pictureContentError
			if errors.As(err, &content) {
				detail(w, http.StatusUnprocessableEntity, "project picture is invalid")
				return
			}
			detail(w, http.StatusServiceUnavailable, "project picture unavailable")
			return
		}
		writePNG(w, body, texture.Width, texture.Height)
		return
	}

	if member, local := index.SpriteTextureMember(name); local {
		key, revision, _ := index.SourceArchiveRevision()
		body, width, height, ok, err := snapshotSprite(p, member, key, revision)
		if err != nil {
			log.Printf("[gameview] sprite %s from imported snapshot: %v", name, err)
			var content pictureContentError
			if errors.As(err, &content) {
				detail(w, http.StatusUnprocessableEntity, "imported sprite is invalid")
				return
			}
			detail(w, http.StatusServiceUnavailable, "imported sprite unavailable")
			return
		}
		if ok {
			writePNG(w, body, width, height)
			return
		}
	}

	if rel, external := index.ExternalSpriteTexture(name); external {
		if pngPath := guirender.VanillaTexturePath(rel); pngPath != "" {
			if body, err := s.vanillaPNG(pngPath); err == nil {
				writePNG(w, body, 0, 0)
				return
			}
		}
	}
	detail(w, http.StatusNotFound, "sprite not resolved: "+name)
}

func rawTexturePNG(p *project.Project, texture guirender.ProjectRawTexture) ([]byte, error) {
	cacheKey := projectKey(p) + "|" + texture.ObjectKey + "#" + texture.Version
	if body, ok := projectPNGs.get(cacheKey); ok {
		return body, nil
	}
	source, err := p.Blobs().Get(texture.ObjectKey, pictureLimit)
	if err != nil {
		return nil, err
	}
	out, _, _, err := encodePNG(source, rawFormatName(texture.Path, source))
	if err != nil {
		return nil, err
	}
	projectPNGs.put(cacheKey, out)
	return out, nil
}

func snapshotSprite(p *project.Project, member, expectedKey, expectedRevision string) ([]byte, int, int, bool, error) {
	ref, attached, err := guiSnapshot(p)
	if err != nil {
		return nil, 0, 0, false, err
	}
	if !attached {
		if expectedKey != "" {
			return nil, 0, 0, false, errors.New("GUI snapshot changed while resolving sprite")
		}
		return nil, 0, 0, false, nil
	}
	if expectedKey != "" && (ref.Key != expectedKey || ref.Revision != expectedRevision) {
		return nil, 0, 0, false, errors.New("GUI snapshot changed while resolving sprite")
	}
	wanted := strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(member), "\\", "/"), "/")
	if _, err := exportengine.SafePath(wanted); err != nil {
		return nil, 0, 0, false, pictureContentError{err}
	}
	switch strings.ToLower(path.Ext(wanted)) {
	case ".dds", ".png", ".tga":
	default:
		return nil, 0, 0, false, nil
	}
	cacheKey := projectKey(p) + "|" + ref.Key + "|" + ref.Revision + "|" + strings.ToLower(wanted)
	if body, ok := projectPNGs.get(cacheKey); ok {
		config, err := png.DecodeConfig(bytes.NewReader(body))
		if err == nil {
			return body, config.Width, config.Height, true, nil
		}
	}
	archive, err := openSnapshot(p, ref)
	if err != nil {
		return nil, 0, 0, false, err
	}
	defer archive.Close()
	folded := strings.ToLower(wanted)
	var target *zipFile
	for _, file := range archive.File {
		if file.FileInfo().IsDir() || strings.ToLower(strings.ReplaceAll(file.Name, "\\", "/")) != folded {
			continue
		}
		if target != nil {
			return nil, 0, 0, false, pictureContentError{fmt.Errorf("GUI snapshot image path is ambiguous: %s", wanted)}
		}
		target = file
	}
	if target == nil {
		if expectedKey != "" {
			return nil, 0, 0, false, errors.New("GUI snapshot changed while resolving sprite")
		}
		return nil, 0, 0, false, nil
	}
	source, err := readMember(target, pictureLimit)
	if err != nil {
		return nil, 0, 0, false, pictureContentError{err}
	}
	body, width, height, err := encodePNG(source, wanted)
	if err != nil {
		return nil, 0, 0, false, err
	}
	projectPNGs.put(cacheKey, body)
	return body, width, height, true, nil
}
