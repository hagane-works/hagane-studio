package characters

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	previewemit "haganeworks/internal/hoi4/emit/preview"
	emitscripts "haganeworks/internal/hoi4/emit/scripts"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

var portraitEntityTypes = []string{"character_portrait", "character_advisor", "character_trait"}

type characterAssetTarget struct {
	id         int64
	tag        string
	countryTag string
}

func prepareCharacterTarget(p *project.Project, cid int64) (characterAssetTarget, error) {
	row, err := findRow(p, table, cid)
	if err != nil {
		return characterAssetTarget{}, err
	}
	if row == nil {
		return characterAssetTarget{}, errAssetTargetMissing
	}
	if strings.TrimSpace(p.Blobs().ProjectUUID()) == "" {
		return characterAssetTarget{}, detail(http.StatusServiceUnavailable, "asset metadata unavailable")
	}
	return characterAssetTarget{
		id: cid, tag: str(row["tag"]),
		countryTag: firstNonBlank(str(row["country_tag"]), projectCountryTag(p)),
	}, nil
}

func portraitSpecs(target characterAssetTarget) []assetSpec {
	tag := strings.ToUpper(target.countryTag)
	charSlug := previewemit.ExtractSlug(target.tag, target.countryTag)
	return []assetSpec{
		{"character_portrait", "character", emitscripts.PortraitFilename(tag, charSlug), ""},
		{"character_advisor", "character", emitscripts.AdvisorIconFilename(tag, charSlug), ""},
		{"character_trait", "character", "trait_" + emitmisc.Slugify(charSlug) + ".png", ""},
	}
}

func uploadPortrait(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, ok := pathID(r, "cid")
	if !ok {
		return errAssetTargetMissing
	}
	target, err := prepareCharacterTarget(p, cid)
	if err != nil {
		return err
	}
	raw, filename, crop, err := readUpload(w, r)
	if err != nil {
		return err
	}
	assets, err := storeRenditions(p, cid, portraitSpecs(target), raw, filename, crop)
	if err != nil {
		return err
	}
	err = p.Write([]string{"entity_obs_paths", table, "project_raw_files"}, func(tx *project.Tx) error {
		if tx.Find(table, cid) == nil {
			return detail(http.StatusServiceUnavailable, "asset metadata commit failed; retry is safe")
		}
		for _, asset := range assets {
			upsertEntityObsPath(tx, asset.entityType, cid, asset.key)
		}

		if _, err := tx.Update(table, cid, func(row project.Row) {
			row["large_portrait"] = assets[0].filename
			row["small_portrait"] = assets[1].filename
			row["is_portrait_modified"] = true
			row["has_obs"] = true
			flipImported(row)
		}); err != nil {
			return err
		}
		return normalizeGeneratedWorldLeader(tx, cid)
	})
	if err != nil {
		return err
	}
	base := "/api/go/projects/" + strconv.FormatInt(p.ID(), 10) + "/characters/" + strconv.FormatInt(cid, 10) + "/assets/"
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"filename": assets[0].filename, "warnings": []string{},
		"portrait_url":     base + url.PathEscape(assets[0].filename),
		"advisor_icon_url": base + url.PathEscape(assets[1].filename),
		"trait_icon_url":   base + url.PathEscape(assets[2].filename),
	})
	return nil
}

func resetPortrait(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, ok := pathID(r, "cid")
	if !ok {
		return errCharacterMissing
	}
	var unreferenced []string
	tables := []string{"entity_obs_paths", table, "project_obs_icons", "project_obs_assets", "project_raw_files"}
	err := p.Write(tables, func(tx *project.Tx) error {
		if tx.Find(table, cid) == nil {
			return errCharacterMissing
		}
		isPortraitSlot := func(row project.Row) bool {
			if project.Int64(row["entity_id"]) != cid {
				return false
			}
			for _, kind := range portraitEntityTypes {
				if str(row["entity_type"]) == kind {
					return true
				}
			}
			return false
		}
		var keys []string
		seen := map[string]bool{}
		for _, row := range tx.Rows("entity_obs_paths") {
			if key := str(row["obs_key"]); isPortraitSlot(row) && key != "" && !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
		tx.Delete("entity_obs_paths", isPortraitSlot)
		if _, err := tx.Update(table, cid, func(row project.Row) {
			row["large_portrait"] = ""
			row["small_portrait"] = ""
			row["is_portrait_modified"] = false
			row["has_obs"] = false
		}); err != nil {
			return err
		}
		for _, key := range keys {
			if !keyReferenced(tx, key) {
				unreferenced = append(unreferenced, key)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	blobs := p.Blobs()
	deleted := make([]string, 0, len(unreferenced))
	for _, key := range unreferenced {
		owned := false
		for _, kind := range portraitEntityTypes {
			if importassets.IsEntityAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "character", cid, kind, key) {
				owned = true
				break
			}
		}
		if !owned {
			log.Printf("reset character portrait refused non-project object key %s", key)
			continue
		}
		path, err := blobs.Path(key)
		if err == nil {
			err = os.Remove(path)
		}
		if err != nil && !os.IsNotExist(err) {

			log.Printf("reset character portrait delete %s: %v", key, err)
			continue
		}
		deleted = append(deleted, "obs:"+key)
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": deleted, "is_portrait_modified": false})
	return nil
}

func keyReferenced(tx *project.Tx, key string) bool {
	for _, ref := range []struct{ table, column string }{
		{"entity_obs_paths", "obs_key"}, {"project_obs_icons", "obs_key"},
		{"project_obs_assets", "obs_key"}, {"project_raw_files", "content_obs_key"},
	} {
		for _, row := range tx.Rows(ref.table) {
			if str(row[ref.column]) == key {
				return true
			}
		}
	}
	return false
}

const maxAnimatedPortraitBytes = 20 * 1024 * 1024

var animatedGFXNameRe = regexp.MustCompile(`^GFX_[A-Za-z0-9_]{1,180}$`)

func attachAnimatedPortrait(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, ok := pathID(r, "cid")
	if !ok {
		return detail(http.StatusNotFound, "Character not found in this project")
	}
	if _, err := prepareCharacterTarget(p, cid); err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAnimatedPortraitBytes+(2<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		return detail(http.StatusUnprocessableEntity, "invalid multipart form")
	}
	gfxName := strings.TrimSpace(r.FormValue("gfx_name"))
	smallGFXName := strings.TrimSpace(r.FormValue("small_gfx_name"))
	frameCount, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("frame_count")))
	fps, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("fps")))
	if !animatedGFXNameRe.MatchString(gfxName) || !animatedGFXNameRe.MatchString(smallGFXName) || gfxName == smallGFXName {
		return detail(http.StatusUnprocessableEntity, "invalid sprite names")
	}
	if frameCount < 2 || frameCount > 48 || fps < 2 || fps > 15 {
		return detail(http.StatusUnprocessableEntity, "frame_count must be 2..48 and fps must be 2..15")
	}
	readPart := func(name string, limit int) ([]byte, bool) {
		file, _, err := r.FormFile(name)
		if err != nil {
			return nil, false
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
		if err != nil || len(data) == 0 || len(data) > limit {
			return nil, false
		}
		return data, true
	}
	large, okLarge := readPart("large_dds", 12<<20)
	small, okSmall := readPart("small_dds", 2<<20)
	preview, okPreview := readPart("preview_png", 4<<20)
	if !okLarge || !okSmall || !okPreview {
		return detail(http.StatusUnprocessableEntity, "large_dds, small_dds and preview_png are required")
	}
	if !bytes.HasPrefix(large, []byte("DDS ")) || !bytes.HasPrefix(small, []byte("DDS ")) {
		return detail(http.StatusBadRequest, "portrait textures must be DDS files")
	}
	gfxText := []byte(exportengine.BuildAnimatedPortraitGFX(gfxName, smallGFXName, frameCount, fps))
	blobs := p.Blobs()
	put := func(entityType, contentType string, data []byte) (string, error) {
		sum := sha256.Sum256(data)
		key, err := importassets.EntityAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "character", cid, entityType, hex.EncodeToString(sum[:]))
		if err != nil {
			return "", detail(http.StatusServiceUnavailable, "object storage write failed; no database pointer was created")
		}
		return key, putBlob(p, key, contentType, data)
	}
	keys := map[string]string{}
	for _, part := range []struct {
		entityType, contentType string
		data                    []byte
	}{
		{"animated_portrait", "image/vnd-ms.dds", large},
		{"animated_portrait_small", "image/vnd-ms.dds", small},
		{"animated_portrait_gfx", "text/plain; charset=utf-8", gfxText},
		{"character_animated_preview", "image/png", preview},
	} {
		key, err := put(part.entityType, part.contentType, part.data)
		if err != nil {
			return err
		}
		keys[part.entityType] = key
	}

	err := p.Write([]string{"project_obs_assets", "entity_obs_paths", table, "project_raw_files"}, func(tx *project.Tx) error {
		if tx.Find(table, cid) == nil {
			return detail(http.StatusNotFound, "Character not found")
		}
		for _, assetType := range []string{"animated_portrait", "animated_portrait_small", "animated_portrait_gfx"} {
			upsertProjectObsAsset(tx, assetType, gfxName, keys[assetType])
		}
		upsertEntityObsPath(tx, "character_animated_preview", cid, keys["character_animated_preview"])
		if _, err := tx.Update(table, cid, func(row project.Row) {
			row["large_portrait"] = gfxName
			row["small_portrait"] = smallGFXName
			row["is_portrait_modified"] = true
			row["has_obs"] = true
			flipImported(row)
		}); err != nil {
			return err
		}
		return normalizeGeneratedWorldLeader(tx, cid)
	})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"ok": true, "gfx_name": gfxName, "small_gfx_name": smallGFXName,
		"frame_count": frameCount, "fps": fps, "is_portrait_modified": true,
	})
	return nil
}

func upsertProjectObsAsset(tx *project.Tx, assetType, gfxName, key string) {
	for _, row := range tx.Rows("project_obs_assets") {
		if str(row["asset_type"]) == assetType && str(row["gfx_name"]) == gfxName {
			_, _ = tx.Update("project_obs_assets", project.Int64(row["id"]), func(r project.Row) { r["obs_key"] = key })
			return
		}
	}
	tx.Insert("project_obs_assets", project.Row{"asset_type": assetType, "gfx_name": gfxName, "obs_key": key})
}

func characterAsset(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, ok := pathID(r, "cid")
	if !ok {
		return errAssetTargetMissing
	}
	lookup := r.PathValue("filename")
	row, err := findRow(p, table, cid)
	if err != nil {
		return err
	}
	if row == nil {
		return errAssetNotUploaded
	}
	paths, err := p.Rows("entity_obs_paths")
	if err != nil {
		return err
	}
	slot := "character_portrait"
	switch lower := strings.ToLower(lookup); {
	case strings.HasPrefix(lookup, "GFX_"):
		slot = "character_animated_preview"
	case lookup == str(row["small_portrait"]) || strings.HasPrefix(lower, "advisor_"):
		slot = "character_advisor"
	case strings.HasPrefix(lower, "trait_"):
		slot = "character_trait"
	}
	key := entityObsKey(paths, slot, cid)
	if key == "" {
		return errAssetNotUploaded
	}
	return serveBlob(p, w, key)
}
