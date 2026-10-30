package characters

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const (
	maxCreatorAssetBytes = 10 * 1024 * 1024

	maxServedAsset = 32 << 20
)

type assetSpec struct {
	entityType string
	obsKind    string
	filename   string
	gfxName    string
}

type uploadedAsset struct {
	entityType string
	gfxName    string
	filename   string
	key        string
}

func readUpload(w http.ResponseWriter, r *http.Request) ([]byte, string, importassets.Crop, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCreatorAssetBytes+(2<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		return nil, "", importassets.Crop{}, detail(http.StatusUnprocessableEntity, "invalid multipart form")
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, "", importassets.Crop{}, detail(http.StatusUnprocessableEntity, "file is required")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxCreatorAssetBytes+1))
	_ = file.Close()
	if err != nil {
		return nil, "", importassets.Crop{}, detail(http.StatusBadRequest, "failed to read upload")
	}
	if len(raw) == 0 {
		return nil, "", importassets.Crop{}, detail(http.StatusBadRequest, "Empty file")
	}
	if len(raw) > maxCreatorAssetBytes {
		return nil, "", importassets.Crop{}, detail(http.StatusRequestEntityTooLarge, "image exceeds the 10 MB size limit")
	}
	crop, err := parseCrop(r)
	if err != nil {
		return nil, "", importassets.Crop{}, detail(http.StatusUnprocessableEntity, "invalid crop coordinates")
	}
	return raw, header.Filename, crop, nil
}

func parseCrop(form *http.Request) (importassets.Crop, error) {
	read := func(name string) (float64, error) {
		raw := strings.TrimSpace(form.FormValue(name))
		if raw == "" {
			return 0, nil
		}
		return strconv.ParseFloat(raw, 64)
	}
	x, err := read("crop_x")
	if err != nil {
		return importassets.Crop{}, err
	}
	y, err := read("crop_y")
	if err != nil {
		return importassets.Crop{}, err
	}
	w, err := read("crop_width")
	if err != nil {
		return importassets.Crop{}, err
	}
	h, err := read("crop_height")
	if err != nil {
		return importassets.Crop{}, err
	}
	return importassets.Crop{X: x, Y: y, Width: w, Height: h}, nil
}

func storeRenditions(p *project.Project, entityID int64, specs []assetSpec, raw []byte, filename string, crop importassets.Crop) ([]uploadedAsset, error) {
	blobs := p.Blobs()
	if strings.TrimSpace(blobs.ProjectUUID()) == "" {
		return nil, detail(http.StatusServiceUnavailable, "asset metadata unavailable")
	}
	var out []uploadedAsset
	for _, spec := range specs {
		pngBytes, err := importassets.NormalizePNGWithCrop(raw, filename, spec.entityType, crop)
		if err != nil {
			return nil, detail(http.StatusBadRequest, "Failed to decode image: "+err.Error())
		}
		sum := sha256.Sum256(pngBytes)
		key, err := importassets.EntityAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), spec.obsKind, entityID, spec.entityType, hex.EncodeToString(sum[:]))
		if err != nil {
			return nil, detail(http.StatusServiceUnavailable, "asset key unavailable")
		}
		if err := putBlob(p, key, "image/png", pngBytes); err != nil {
			return nil, err
		}
		out = append(out, uploadedAsset{entityType: spec.entityType, gfxName: spec.gfxName, filename: spec.filename, key: key})
	}
	return out, nil
}

func putBlob(p *project.Project, key, contentType string, data []byte) error {
	blobs := p.Blobs()
	if blobs.Exists(key) {
		return nil
	}
	if err := blobs.Put(key, contentType, int64(len(data)), bytes.NewReader(data)); err != nil {
		return detail(http.StatusServiceUnavailable, "object storage write failed; no database pointer was created")
	}
	return nil
}

func upsertEntityObsPath(tx *project.Tx, entityType string, entityID int64, key string) {
	for _, row := range tx.Rows("entity_obs_paths") {
		if str(row["entity_type"]) == entityType && project.Int64(row["entity_id"]) == entityID {
			_, _ = tx.Update("entity_obs_paths", project.Int64(row["id"]), func(r project.Row) {
				r["project_id"] = tx.Record()["id"]
				r["obs_key"] = key
			})
			return
		}
	}
	tx.Insert("entity_obs_paths", project.Row{"entity_type": entityType, "entity_id": entityID, "obs_key": key})
}

func entityObsKey(rows []project.Row, entityType string, entityID int64) string {
	for _, row := range rows {
		if str(row["entity_type"]) == entityType && project.Int64(row["entity_id"]) == entityID {
			return str(row["obs_key"])
		}
	}
	return ""
}

func serveBlob(p *project.Project, w http.ResponseWriter, key string) error {
	data, err := p.Blobs().Get(key, maxServedAsset)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, os.ErrNotExist) {
		return errAssetNotUploaded
	}
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", http.DetectContentType(data))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
	return nil
}

var errAssetNotUploaded = detail(http.StatusNotFound, "Asset not uploaded")

var errAssetTargetMissing = detail(http.StatusNotFound, "Asset target not found")
