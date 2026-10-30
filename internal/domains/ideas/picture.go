package ideas

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const (
	maxPictureBytes = 10 * 1024 * 1024

	pictureEntityType = "idea_picture"
	maxStoredPicture  = 32 << 20
)

var (
	errTargetMissing = apiutil.NotFoundError("Asset target not found")
	errNotUploaded   = apiutil.NotFoundError("Asset not uploaded")
)

func status(code int, detail string) error {
	return &apiutil.StatusError{Status: code, Body: map[string]string{"detail": detail}}
}

func pictureID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("iid"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errTargetMissing
	}
	return id, nil
}

func pictureNames(ideaID, countryTag string) (filename, gfxName string) {
	slug := emitmisc.Slugify(ideaID)
	tag := strings.ToUpper(countryTag)
	return "idea_" + tag + "_" + slug + ".png", "GFX_idea_" + tag + "_" + slug
}

func uploadPicture(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := pictureID(r)
	if err != nil {
		return err
	}
	rows, err := p.Rows(table)
	if err != nil {
		return err
	}
	var idea project.Row
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			idea = row
			break
		}
	}
	if idea == nil {
		return errTargetMissing
	}
	blobs := p.Blobs()
	if strings.TrimSpace(blobs.ProjectUUID()) == "" {
		return status(http.StatusServiceUnavailable, "asset metadata unavailable")
	}
	countryTag := text(idea, "country_tag")
	if countryTag == "" {
		countryTag, _ = p.Record()["country_tag"].(string)
	}
	logicalID := text(idea, "idea_id")

	r.Body = http.MaxBytesReader(w, r.Body, maxPictureBytes+(2<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		return apiutil.ValidationError("invalid multipart form")
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return apiutil.ValidationError("file is required")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxPictureBytes+1))
	_ = file.Close()
	if err != nil {
		return apiutil.BadRequestError("failed to read upload")
	}
	if len(raw) == 0 {
		return apiutil.BadRequestError("Empty file")
	}
	if len(raw) > maxPictureBytes {
		return status(http.StatusRequestEntityTooLarge, "image exceeds the 10 MB size limit")
	}
	crop, err := parseCrop(r)
	if err != nil {
		return apiutil.ValidationError("invalid crop coordinates")
	}

	pngBytes, err := importassets.NormalizePNGWithCrop(raw, header.Filename, pictureEntityType, crop)
	if err != nil {
		return apiutil.BadRequestError("Failed to decode image: " + err.Error())
	}
	sum := sha256.Sum256(pngBytes)
	key, err := importassets.EntityAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "idea", id, pictureEntityType, hex.EncodeToString(sum[:]))
	if err != nil {
		return status(http.StatusServiceUnavailable, "asset key unavailable")
	}
	if !blobs.Exists(key) {
		if err := blobs.Put(key, "image/png", int64(len(pngBytes)), bytes.NewReader(pngBytes)); err != nil {
			return status(http.StatusServiceUnavailable, "object storage write failed; no database pointer was created")
		}
	}
	filename, gfxName := pictureNames(logicalID, countryTag)

	err = p.Write([]string{"entity_obs_paths", table}, func(tx *project.Tx) error {
		if tx.Find(table, id) == nil {
			return errTargetMissing
		}
		upsertEntityPath(tx, id, key)
		_, err := tx.Update(table, id, func(row project.Row) {
			row["picture"] = gfxName
			row["has_obs"] = true
		})
		return err
	})
	if err != nil {
		if errors.Is(err, errTargetMissing) {
			return status(http.StatusServiceUnavailable, "asset metadata commit failed; retry is safe")
		}
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"filename":    filename,
		"warnings":    []string{},
		"picture":     gfxName,
		"preview_url": "/api/go/projects/" + strconv.FormatInt(p.ID(), 10) + "/ideas/" + strconv.FormatInt(id, 10) + "/picture",
	})
	return nil
}

func upsertEntityPath(tx *project.Tx, id int64, key string) {
	for _, row := range tx.Rows("entity_obs_paths") {
		if text(row, "entity_type") == pictureEntityType && project.Int64(row["entity_id"]) == id {
			_, _ = tx.Update("entity_obs_paths", project.Int64(row["id"]), func(row project.Row) {
				row["obs_key"] = key
			})
			return
		}
	}
	tx.Insert("entity_obs_paths", project.Row{
		"entity_type": pictureEntityType, "entity_id": id, "obs_key": key,
	})
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
	if crop.Height, err = read("crop_height"); err != nil {
		return crop, err
	}
	return crop, nil
}

func picture(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := pictureID(r)
	if err != nil {
		return err
	}
	key, err := pictureKey(p, id)
	if err != nil {
		return err
	}
	data, err := p.Blobs().Get(key, maxStoredPicture)
	if err != nil {
		return errNotUploaded
	}
	contentType := "image/png"
	if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		contentType = http.DetectContentType(data)
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
	return nil
}

func pictureKey(p *project.Project, id int64) (string, error) {
	ideas, err := p.Rows(table)
	if err != nil {
		return "", err
	}
	var idea project.Row
	for _, row := range ideas {
		if project.Int64(row["id"]) == id {
			idea = row
			break
		}
	}
	if idea == nil {
		return "", errNotUploaded
	}
	paths, err := p.Rows("entity_obs_paths")
	if err != nil {
		return "", err
	}
	for _, row := range paths {
		if text(row, "entity_type") == pictureEntityType && project.Int64(row["entity_id"]) == id {
			if key := text(row, "obs_key"); key != "" {
				return key, nil
			}
			return "", errNotUploaded
		}
	}
	assets, err := p.Rows("project_obs_assets")
	if err != nil {
		return "", err
	}

	name := text(idea, "picture")
	names := map[string]bool{name: true}
	if name != "" && !strings.HasPrefix(name, "GFX_") {
		names["GFX_"+name] = true
		names["GFX_idea_"+name] = true
	}
	for _, row := range assets {
		if text(row, "asset_type") == pictureEntityType && names[text(row, "gfx_name")] {
			if key := text(row, "obs_key"); key != "" {
				return key, nil
			}
		}
	}
	return "", errNotUploaded
}
