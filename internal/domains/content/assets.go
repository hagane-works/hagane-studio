package content

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const (
	maxCreatorAssetBytes = 10 * 1024 * 1024
	maxStoredAssetBytes  = 32 << 20
)

type assetSpec struct {
	table      string
	entityType string
	obsKind    string
	filename   string
	gfxName    string

	apply func(row project.Row)
}

func (h *handlers) assetRoute(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string,
	upload, read string, spec func(p *project.Project, id int64) (assetSpec, error),
	respond func(w http.ResponseWriter, id int64, spec assetSpec) error,
) bool {
	if len(rest) != 2 || (rest[1] != upload && rest[1] != read) {
		return false
	}
	id, err := strconv.ParseInt(rest[0], 10, 64)
	if err != nil || id <= 0 {
		detail(w, http.StatusNotFound, "Asset target not found")
		return true
	}
	if rest[1] == upload {
		if r.Method != http.MethodPost {
			detail(w, http.StatusMethodNotAllowed, "POST only")
			return true
		}
		target, err := spec(p, id)
		if err != nil {
			fail(w, err, "asset metadata unavailable")
			return true
		}
		if err := uploadAsset(p, w, r, id, target); err != nil {
			fail(w, err, "asset metadata commit failed; retry is safe")
			return true
		}
		if err := respond(w, id, target); err != nil {
			fail(w, err, "asset metadata unavailable")
		}
		return true
	}
	if r.Method != http.MethodGet {
		detail(w, http.StatusMethodNotAllowed, "GET only")
		return true
	}
	target, err := spec(p, id)
	if err != nil {

		detail(w, http.StatusNotFound, "Asset not uploaded")
		return true
	}
	serveAsset(p, w, id, target.entityType)
	return true
}

var errAssetTargetMissing = statusErr(http.StatusNotFound, "Asset target not found")

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

func uploadAsset(p *project.Project, w http.ResponseWriter, r *http.Request, id int64, spec assetSpec) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxCreatorAssetBytes+(2<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		return statusErr(http.StatusUnprocessableEntity, "invalid multipart form")
	}
	defer r.MultipartForm.RemoveAll()
	file, header, err := r.FormFile("file")
	if err != nil {
		return statusErr(http.StatusUnprocessableEntity, "file is required")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxCreatorAssetBytes+1))
	_ = file.Close()
	if err != nil {
		return statusErr(http.StatusBadRequest, "failed to read upload")
	}
	if len(raw) == 0 {
		return statusErr(http.StatusBadRequest, "Empty file")
	}
	if len(raw) > maxCreatorAssetBytes {
		return statusErr(http.StatusRequestEntityTooLarge, "image exceeds the 10 MB size limit")
	}
	crop, err := parseCrop(r)
	if err != nil {
		return statusErr(http.StatusUnprocessableEntity, "invalid crop coordinates")
	}
	pngBytes, err := importassets.NormalizePNGWithCrop(raw, header.Filename, spec.entityType, crop)
	if err != nil {
		return statusErr(http.StatusBadRequest, "Failed to decode image: "+err.Error())
	}
	sum := sha256.Sum256(pngBytes)
	blobs := p.Blobs()
	key, err := importassets.EntityAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), spec.obsKind, id, spec.entityType, hex.EncodeToString(sum[:]))
	if err != nil {
		return statusErr(http.StatusServiceUnavailable, "asset key unavailable")
	}
	if !blobs.Exists(key) {
		if err := blobs.Put(key, "image/png", int64(len(pngBytes)), bytes.NewReader(pngBytes)); err != nil {
			return statusErr(http.StatusServiceUnavailable, "object storage write failed; no database pointer was created")
		}
	}

	return p.Write([]string{"entity_obs_paths", spec.table}, func(tx *project.Tx) error {
		if tx.Find(spec.table, id) == nil {
			return fmt.Errorf("asset target %s/%d vanished", spec.table, id)
		}
		mapped := false
		for _, row := range tx.Rows("entity_obs_paths") {
			if row["entity_type"] == spec.entityType && project.Int64(row["entity_id"]) == id {
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
			tx.Insert("entity_obs_paths", project.Row{"entity_type": spec.entityType, "entity_id": id, "obs_key": key})
		}
		_, err := tx.Update(spec.table, id, spec.apply)
		return err
	})
}

func serveAsset(p *project.Project, w http.ResponseWriter, id int64, entityType string) {
	paths, err := p.Rows("entity_obs_paths")
	if err != nil {
		fail(w, err, "asset metadata unavailable")
		return
	}
	key := ""
	for _, row := range paths {
		if row["entity_type"] == entityType && project.Int64(row["entity_id"]) == id {
			key = strings.TrimSpace(text(row, "obs_key"))
			break
		}
	}
	if key == "" {
		detail(w, http.StatusNotFound, "Asset not uploaded")
		return
	}
	body, err := p.Blobs().Get(key, maxStoredAssetBytes)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "asset object unavailable")
		return
	}
	header := w.Header()
	header.Set("Content-Type", http.DetectContentType(body))
	header.Set("Cache-Control", "no-store")
	header.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
