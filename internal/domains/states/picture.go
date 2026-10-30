package states

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/gamefiles"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const (
	maxCreatorAssetBytes = 10 * 1024 * 1024
	maxPictureBytes      = 32 << 20

	bookmarkPictureType = "bookmark_picture"
)

var (
	errAssetTargetMissing = apiutil.NotFoundError("Asset target not found")
	errAssetMissing       = apiutil.NotFoundError("Asset not uploaded")
)

func statusError(status int, detail string) error {
	return &apiutil.StatusError{Status: status, Body: map[string]string{"detail": detail}}
}

func parseCrop(r *http.Request) (importassets.Crop, error) {
	read := func(name string) (float64, error) {
		raw := strings.TrimSpace(r.FormValue(name))
		if raw == "" {
			return 0, nil
		}
		return strconv.ParseFloat(raw, 64)
	}
	var crop importassets.Crop
	var err error
	if crop.X, err = read("crop_x"); err != nil {
		return crop, err
	}
	if crop.Y, err = read("crop_y"); err != nil {
		return crop, err
	}
	if crop.Width, err = read("crop_width"); err != nil {
		return crop, err
	}
	crop.Height, err = read("crop_height")
	return crop, err
}

func (h *handlers) uploadBookmarkPicture(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.ParseInt(r.PathValue("bid"), 10, 64)
	if err != nil || id <= 0 {
		return errAssetTargetMissing
	}
	rows, err := p.Rows(bookmarksTable)
	if err != nil {
		return err
	}
	var target project.Row
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			target = row
			break
		}
	}
	if target == nil {
		return errAssetTargetMissing
	}
	rv, err := readRevisions(r)
	if err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxCreatorAssetBytes+(2<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		return detailError(http.StatusUnprocessableEntity, "invalid multipart form")
	}
	defer r.MultipartForm.RemoveAll()
	file, header, err := r.FormFile("file")
	if err != nil {
		return detailError(http.StatusUnprocessableEntity, "file is required")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxCreatorAssetBytes+1))
	_ = file.Close()
	if err != nil {
		return statusError(http.StatusBadRequest, "failed to read upload")
	}
	if len(raw) == 0 {
		return statusError(http.StatusBadRequest, "Empty file")
	}
	if len(raw) > maxCreatorAssetBytes {
		return statusError(http.StatusRequestEntityTooLarge, "image exceeds the 10 MB size limit")
	}
	crop, err := parseCrop(r)
	if err != nil {
		return detailError(http.StatusUnprocessableEntity, "invalid crop coordinates")
	}

	slug := emitmisc.Slugify(str(target["bookmark_id"]))
	filename := "bookmark_" + slug + ".png"
	gfxName := "GFX_hagane_select_date_" + slug
	pngBytes, err := importassets.NormalizePNGWithCropToSize(raw, header.Filename, bookmarkPictureType, crop, importassets.Size{Width: 180, Height: 104})
	if err != nil {
		return statusError(http.StatusBadRequest, "Failed to decode image: "+err.Error())
	}
	sum := sha256.Sum256(pngBytes)
	blobs := p.Blobs()
	key, err := importassets.EntityAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "bookmark", id, bookmarkPictureType, hex.EncodeToString(sum[:]))
	if err != nil {
		return statusError(http.StatusServiceUnavailable, "asset key unavailable")
	}
	if !blobs.Exists(key) {
		if err := blobs.Put(key, "image/png", int64(len(pngBytes)), bytes.NewReader(pngBytes)); err != nil {
			return fmt.Errorf("store bookmark picture: %w", err)
		}
	}

	err = p.Write([]string{"entity_obs_paths", bookmarksTable}, func(tx *project.Tx) error {
		if _, err := rv.advance(tx, id); err != nil {
			return err
		}
		mapped := false
		for _, row := range tx.Rows("entity_obs_paths") {
			if row["entity_type"] == bookmarkPictureType && project.Int64(row["entity_id"]) == id {
				if _, err := tx.Update("entity_obs_paths", project.Int64(row["id"]), func(row project.Row) {
					row["project_id"] = tx.Record()["id"]
					row["obs_key"] = key
				}); err != nil {
					return err
				}
				mapped = true
				break
			}
		}
		if !mapped {
			tx.Insert("entity_obs_paths", project.Row{"entity_type": bookmarkPictureType, "entity_id": id, "obs_key": key})
		}
		_, err := tx.Update(bookmarksTable, id, func(row project.Row) { row["picture"] = gfxName })
		return err
	})
	if err != nil {
		return err
	}
	rv.commit(p)
	rv.header(w)
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"filename": filename, "warnings": []string{}, "picture": gfxName,
		"preview_url": fmt.Sprintf("/api/go/projects/%d/bookmarks/%d/picture", p.ID(), id),
	})
	return nil
}

func bookmarkPictureKey(p *project.Project, id int64) (key, picture string, err error) {
	rows, err := p.Rows(bookmarksTable)
	if err != nil {
		return "", "", err
	}
	var row project.Row
	for _, candidate := range rows {
		if project.Int64(candidate["id"]) == id {
			row = candidate
			break
		}
	}
	if row == nil {
		return "", "", errAssetMissing
	}
	picture = str(row["picture"])
	paths, err := p.Rows("entity_obs_paths")
	if err != nil {
		return "", "", err
	}
	for _, mapping := range paths {
		if mapping["entity_type"] == bookmarkPictureType && project.Int64(mapping["entity_id"]) == id {
			if key := str(mapping["obs_key"]); key != "" {
				return key, picture, nil
			}
		}
	}
	assets, err := p.Rows("project_obs_assets")
	if err != nil {
		return "", "", err
	}
	for _, asset := range assets {
		if asset["asset_type"] == bookmarkPictureType && str(asset["gfx_name"]) == picture {
			if key := str(asset["obs_key"]); key != "" {
				return key, picture, nil
			}
		}
	}
	return "", picture, nil
}

func (h *handlers) bookmarkPicture(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.ParseInt(r.PathValue("bid"), 10, 64)
	if err != nil || id <= 0 {
		return errAssetTargetMissing
	}
	key, picture, err := bookmarkPictureKey(p, id)
	if err != nil {
		return err
	}
	var body []byte
	if key != "" {
		if data, err := p.Blobs().Get(key, maxPictureBytes); err == nil {
			body, _ = browserImage(data, key)
		}
	}
	if body == nil && picture != "" {
		if game, err := gamefiles.Open(h.deps.Game); err == nil {
			body, _ = game.SpritePNG(picture)
		}
	}
	if body == nil {
		return errAssetMissing
	}
	header := w.Header()
	header.Set("Content-Type", http.DetectContentType(body))
	header.Set("Cache-Control", "no-store")
	header.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return nil
}

func browserImage(data []byte, name string) ([]byte, error) {
	switch http.DetectContentType(data) {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return data, nil
	}
	img, err := importassets.DecodeImage(data, name)
	if err != nil {
		return nil, fmt.Errorf("decode picture %s: %w", name, err)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
