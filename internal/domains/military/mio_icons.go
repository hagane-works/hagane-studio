package military

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const (
	maxIconUploadBytes = 10 * 1024 * 1024
	maxIconReadBytes   = 32 * 1024 * 1024

	mioIconType      = "mio_icon"
	mioTraitIconType = "mio_trait_icon"
)

var errAssetTargetMissing = errors.New("asset target not found")

type statusDetail struct {
	status int
	detail string
}

func (e *statusDetail) Error() string { return e.detail }

func (h *handlers) mioIconRoute(w http.ResponseWriter, r *http.Request, p *project.Project, rest []string) bool {
	switch {
	case len(rest) == 4 && rest[1] == "traits":
		id, err := strconv.ParseInt(rest[0], 10, 64)
		if err != nil || id <= 0 {
			detail(w, http.StatusNotFound, "MIO not found")
			return true
		}
		mioTraitIcon(w, r, p, id, rest[2], rest[3])
		return true
	case len(rest) == 2 && (rest[1] == "upload-icon" || rest[1] == "icon"):
		id, err := strconv.ParseInt(rest[0], 10, 64)
		if err != nil || id <= 0 {
			detail(w, http.StatusNotFound, "Asset target not found")
			return true
		}
		if rest[1] == "upload-icon" {
			if r.Method != http.MethodPost {
				detail(w, http.StatusMethodNotAllowed, "POST only")
				return true
			}
			uploadMIOIcon(w, r, p, id)
			return true
		}
		if r.Method != http.MethodGet {
			detail(w, http.StatusMethodNotAllowed, "GET only")
			return true
		}
		readMIOIcon(w, p, id)
		return true
	}
	return false
}

func readUpload(w http.ResponseWriter, r *http.Request) ([]byte, string, importassets.Crop, bool) {
	var crop importassets.Crop
	r.Body = http.MaxBytesReader(w, r.Body, maxIconUploadBytes+(2<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		detail(w, http.StatusUnprocessableEntity, "invalid multipart form")
		return nil, "", crop, false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		detail(w, http.StatusUnprocessableEntity, "file is required")
		return nil, "", crop, false
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxIconUploadBytes+1))
	_ = file.Close()
	if err != nil {
		detail(w, http.StatusBadRequest, "failed to read upload")
		return nil, "", crop, false
	}
	if len(raw) == 0 {
		detail(w, http.StatusBadRequest, "Empty file")
		return nil, "", crop, false
	}
	if len(raw) > maxIconUploadBytes {
		detail(w, http.StatusRequestEntityTooLarge, "image exceeds the 10 MB size limit")
		return nil, "", crop, false
	}
	crop, err = parseCrop(r)
	if err != nil {
		detail(w, http.StatusUnprocessableEntity, "invalid crop coordinates")
		return nil, "", crop, false
	}
	return raw, header.Filename, crop, true
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

func storePNG(p *project.Project, key string, png []byte) error {
	blobs := p.Blobs()
	if blobs.Exists(key) {
		return nil
	}
	if err := blobs.Put(key, "image/png", int64(len(png)), bytes.NewReader(png)); err != nil {
		log.Printf("mio icon write %s: %v", key, err)
		return &statusDetail{http.StatusServiceUnavailable, "object storage write failed; no database pointer was created"}
	}
	return nil
}

func uploadMIOIcon(w http.ResponseWriter, r *http.Request, p *project.Project, id int64) {
	rows, err := p.Rows(mioTable)
	if err != nil {
		log.Printf("prepare mio icon target: %v", err)
		detail(w, http.StatusServiceUnavailable, "asset metadata unavailable")
		return
	}
	row := findRow(rows, id)
	if row == nil {
		detail(w, http.StatusNotFound, "Asset target not found")
		return
	}
	raw, filename, crop, ok := readUpload(w, r)
	if !ok {
		return
	}
	slug := emitmisc.Slugify(text(row, "mio_id"))
	storedName, gfxName := "mio_"+slug+".png", "GFX_idea_mio_"+slug
	png, err := importassets.NormalizePNGWithCrop(raw, filename, mioIconType, crop)
	if err != nil {
		detail(w, http.StatusBadRequest, "Failed to decode image: "+err.Error())
		return
	}
	sum := sha256.Sum256(png)
	blobs := p.Blobs()
	key, err := importassets.EntityAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "mio", id, mioIconType, hex.EncodeToString(sum[:]))
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "asset key unavailable")
		return
	}
	if err := storePNG(p, key, png); err != nil {
		writeUploadError(w, err, "mio icon")
		return
	}
	err = p.Write([]string{"entity_obs_paths", mioTable}, func(tx *project.Tx) error {
		if tx.Find(mioTable, id) == nil {
			return errAssetTargetMissing
		}
		if err := mapEntityKey(tx, mioIconType, id, key); err != nil {
			return err
		}
		_, err := tx.Update(mioTable, id, func(row project.Row) { row["icon"] = gfxName })
		return err
	})
	if err != nil {
		writeUploadError(w, err, "mio icon")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"filename": storedName, "warnings": []string{}, "icon": gfxName,
		"preview_url": fmt.Sprintf("/api/go/projects/%d/mios/%d/icon", p.ID(), id),
	})
}

func mapEntityKey(tx *project.Tx, entityType string, id int64, key string) error {
	for _, row := range tx.Rows("entity_obs_paths") {
		if text(row, "entity_type") == entityType && project.Int64(row["entity_id"]) == id {
			_, err := tx.Update("entity_obs_paths", project.Int64(row["id"]), func(r project.Row) { r["obs_key"] = key })
			return err
		}
	}
	tx.Insert("entity_obs_paths", project.Row{"entity_type": entityType, "entity_id": id, "obs_key": key})
	return nil
}

func writeUploadError(w http.ResponseWriter, err error, kind string) {
	var status *statusDetail
	switch {
	case errors.As(err, &status):
		detail(w, status.status, status.detail)
	case errors.Is(err, errAssetTargetMissing):
		detail(w, http.StatusNotFound, "Asset target not found")
	case errors.Is(err, errMIOTraitMissing):
		detail(w, http.StatusNotFound, errMIOTraitMissing.Error())
	case errors.Is(err, errMIOMissing):
		detail(w, http.StatusNotFound, errMIOMissing.Error())
	default:
		log.Printf("commit %s metadata: %v", kind, err)
		detail(w, http.StatusServiceUnavailable, "asset metadata commit failed; retry is safe")
	}
}

func readMIOIcon(w http.ResponseWriter, p *project.Project, id int64) {
	rows, err := p.Rows(mioTable)
	if err == nil && findRow(rows, id) == nil {
		detail(w, http.StatusNotFound, "Asset not uploaded")
		return
	}
	var paths []project.Row
	if err == nil {
		paths, err = p.Rows("entity_obs_paths")
	}
	if err != nil {
		log.Printf("resolve mio icon %d: %v", id, err)
		detail(w, http.StatusServiceUnavailable, "asset metadata unavailable")
		return
	}
	key := ""
	for _, row := range paths {
		if text(row, "entity_type") == mioIconType && project.Int64(row["entity_id"]) == id {
			key = strings.TrimSpace(text(row, "obs_key"))
			break
		}
	}
	if key == "" {
		detail(w, http.StatusNotFound, "Asset not uploaded")
		return
	}
	serveBlob(w, p, key)
}

func serveBlob(w http.ResponseWriter, p *project.Project, key string) {
	data, err := p.Blobs().Get(key, maxIconReadBytes)
	if err != nil {
		log.Printf("mio icon %s: %v", key, err)
		detail(w, http.StatusNotFound, "Asset not uploaded")
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(data))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

var errMIOTraitMissing = errors.New("MIO trait not found")

func mioTraitIconName(mioID, token string) string {
	return "GFX_idea_mio_trait_" + emitmisc.Slugify(mioID) + "_" + emitmisc.Slugify(token)
}

func findMIOTrait(traits []map[string]any, token string) (int, bool) {
	for i, trait := range traits {
		if value, _ := trait["token"].(string); value == token {
			return i, true
		}
	}
	return 0, false
}

func mioTraitTarget(p *project.Project, id int64, rawToken string) (mioID, token string, err error) {
	token = strings.TrimSpace(rawToken)
	if token == "" {
		return "", "", &statusDetail{http.StatusBadRequest, "trait token is required"}
	}
	rows, err := p.Rows(mioTable)
	if err != nil {
		return "", "", err
	}
	row := findRow(rows, id)
	if row == nil {
		return "", "", errMIOMissing
	}
	var traits []map[string]any
	decode(row["traits"], &traits)
	if _, found := findMIOTrait(traits, token); !found {
		return "", "", errMIOTraitMissing
	}
	return text(row, "mio_id"), token, nil
}

func mioTraitIcon(w http.ResponseWriter, r *http.Request, p *project.Project, id int64, rawToken, action string) {
	switch action {
	case "upload-icon":
		if r.Method != http.MethodPost {
			detail(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		mioID, token, err := mioTraitTarget(p, id, rawToken)
		if err != nil {
			writeTraitTargetError(w, err)
			return
		}
		raw, filename, crop, ok := readUpload(w, r)
		if !ok {
			return
		}
		gfxName := mioTraitIconName(mioID, token)
		png, err := importassets.NormalizePNGWithCrop(raw, filename, mioTraitIconType, crop)
		if err != nil {
			detail(w, http.StatusBadRequest, "Failed to decode image: "+err.Error())
			return
		}
		sum := sha256.Sum256(png)
		blobs := p.Blobs()
		key, err := importassets.NamedAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "mio_trait", gfxName, hex.EncodeToString(sum[:]))
		if err != nil {
			detail(w, http.StatusServiceUnavailable, "asset key unavailable")
			return
		}
		if err := storePNG(p, key, png); err != nil {
			writeUploadError(w, err, "MIO trait")
			return
		}
		if gfxName, err = commitMIOTraitIcon(p, id, mioID, token, key); err != nil {
			writeUploadError(w, err, "MIO trait")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"filename": gfxName + ".png", "icon": gfxName, "warnings": []string{},
			"preview_url": fmt.Sprintf("/api/go/projects/%d/mios/%d/traits/%s/icon", p.ID(), id, url.PathEscape(token)),
		})
	case "icon":
		if r.Method != http.MethodGet {
			detail(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		const missing = "MIO trait icon not uploaded"
		mioID, token, err := mioTraitTarget(p, id, rawToken)
		if errors.Is(err, errMIOMissing) || errors.Is(err, errMIOTraitMissing) {
			detail(w, http.StatusNotFound, missing)
			return
		}
		if err != nil {
			log.Printf("resolve MIO trait icon: %v", err)
			detail(w, http.StatusServiceUnavailable, "asset metadata unavailable")
			return
		}
		gfxName := mioTraitIconName(mioID, token)
		icons, err := p.Rows("project_obs_icons")
		if err != nil {
			log.Printf("resolve MIO trait icon: %v", err)
			detail(w, http.StatusServiceUnavailable, "asset metadata unavailable")
			return
		}
		key := ""
		for _, row := range icons {
			if text(row, "gfx_name") == gfxName {
				key = strings.TrimSpace(text(row, "obs_key"))
				break
			}
		}
		blobs := p.Blobs()
		if key == "" || !importassets.IsNamedAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "mio_trait", gfxName, key) {
			detail(w, http.StatusNotFound, missing)
			return
		}
		serveBlob(w, p, key)
	default:
		detail(w, http.StatusNotFound, "MIO trait asset not found")
	}
}

func writeTraitTargetError(w http.ResponseWriter, err error) {
	var status *statusDetail
	switch {
	case errors.Is(err, errMIOMissing):
		detail(w, http.StatusNotFound, "MIO not found")
	case errors.Is(err, errMIOTraitMissing):
		detail(w, http.StatusNotFound, errMIOTraitMissing.Error())
	case errors.As(err, &status):
		detail(w, status.status, status.detail)
	default:
		log.Printf("prepare named asset: %v", err)
		detail(w, http.StatusServiceUnavailable, "asset metadata unavailable")
	}
}

func commitMIOTraitIcon(p *project.Project, id int64, mioID, token, key string) (string, error) {
	var gfxName string
	err := p.Write([]string{mioTable, "project_obs_icons"}, func(tx *project.Tx) error {
		row := tx.Find(mioTable, id)
		if row == nil {
			return errMIOMissing
		}
		current := text(row, "mio_id")
		if current != mioID {
			return errors.New("MIO identity changed during icon upload; retry is safe")
		}
		var traits []map[string]any
		decode(row["traits"], &traits)
		index, found := findMIOTrait(traits, token)
		if !found {
			return errMIOTraitMissing
		}
		gfxName = mioTraitIconName(current, token)
		traits[index]["icon"] = gfxName
		if _, err := tx.Update(mioTable, id, func(row project.Row) { row["traits"] = jsonValue(traits) }); err != nil {
			return err
		}
		for _, icon := range tx.Rows("project_obs_icons") {
			if text(icon, "gfx_name") == gfxName {
				_, err := tx.Update("project_obs_icons", project.Int64(icon["id"]), func(r project.Row) { r["obs_key"] = key })
				return err
			}
		}
		tx.Insert("project_obs_icons", project.Row{"gfx_name": gfxName, "obs_key": key})
		return nil
	})
	return gfxName, err
}
