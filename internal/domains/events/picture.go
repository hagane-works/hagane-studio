package events

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
	"haganeworks/internal/workbench"
)

const (
	maxCreatorAssetBytes = 10 * 1024 * 1024
	maxPictureBytes      = 32 << 20

	pictureEntityType = "event_picture"
)

var (
	errAssetTargetMissing = apiutil.NotFoundError("Asset target not found")
	errAssetMissing       = apiutil.NotFoundError("Asset not uploaded")
)

type pictureTarget struct {
	ProjectUUID string
	EventID     string
	CountryTag  string
	EventType   string
}

func prepareTarget(p *project.Project, eventID int64) (pictureTarget, error) {
	rows, err := p.Rows(eventsTable)
	if err != nil {
		return pictureTarget{}, err
	}
	row := findRow(rows, eventID)
	if row == nil {
		return pictureTarget{}, errAssetTargetMissing
	}
	record := p.Record()
	return pictureTarget{
		ProjectUUID: p.Blobs().ProjectUUID(),
		EventID:     text(row, "event_id", ""),
		CountryTag:  workbench.FirstNonBlank(text(row, "country_tag", ""), text(record, "country_tag", "")),
		EventType:   text(row, "event_type", ""),
	}, nil
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

func statusError(status int, detail string) error {
	return &apiutil.StatusError{Status: status, Body: map[string]string{"detail": detail}}
}

func (h *handlers) uploadPicture(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	eventID, err := pathID(r, "eid", errAssetTargetMissing)
	if err != nil {
		return err
	}
	target, err := prepareTarget(p, eventID)
	if err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxCreatorAssetBytes+(2<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		return apiutil.ValidationError("invalid multipart form")
	}
	defer r.MultipartForm.RemoveAll()
	file, header, err := r.FormFile("file")
	if err != nil {
		return apiutil.ValidationError("file is required")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxCreatorAssetBytes+1))
	_ = file.Close()
	if err != nil {
		return apiutil.BadRequestError("failed to read upload")
	}
	if len(raw) == 0 {
		return apiutil.BadRequestError("Empty file")
	}
	if len(raw) > maxCreatorAssetBytes {
		return statusError(http.StatusRequestEntityTooLarge, "image exceeds the 10 MB size limit")
	}
	crop, err := parseCrop(r)
	if err != nil {
		return apiutil.ValidationError("invalid crop coordinates")
	}

	tag := strings.ToUpper(target.CountryTag)
	slug := emitmisc.Slugify(target.EventID)
	filename := "event_" + tag + "_" + slug + ".png"
	gfxName := "GFX_report_event_" + tag + "_" + slug
	var pngBytes []byte
	if target.EventType == "news_event" {
		pngBytes, err = importassets.NormalizePNGWithCropToSize(raw, header.Filename, pictureEntityType, crop, importassets.NewsEventPictureSize)
	} else {
		pngBytes, err = importassets.NormalizePNGWithCrop(raw, header.Filename, pictureEntityType, crop)
	}
	if err != nil {
		return apiutil.BadRequestError("Failed to decode image: " + err.Error())
	}
	sum := sha256.Sum256(pngBytes)
	blobs := p.Blobs()
	key, err := importassets.EntityAssetKey(blobs.EnvPrefix(), target.ProjectUUID, "event", eventID, pictureEntityType, hex.EncodeToString(sum[:]))
	if err != nil {
		return statusError(http.StatusServiceUnavailable, "asset key unavailable")
	}
	if !blobs.Exists(key) {
		if err := blobs.Put(key, "image/png", int64(len(pngBytes)), bytes.NewReader(pngBytes)); err != nil {
			return fmt.Errorf("store event picture: %w", err)
		}
	}

	err = p.Write([]string{"entity_obs_paths", eventsTable}, func(tx *project.Tx) error {
		if tx.Find(eventsTable, eventID) == nil {
			return errAssetTargetMissing
		}
		mapped := false
		for _, row := range tx.Rows("entity_obs_paths") {
			if row["entity_type"] == pictureEntityType && project.Int64(row["entity_id"]) == eventID {
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
			tx.Insert("entity_obs_paths", project.Row{"entity_type": pictureEntityType, "entity_id": eventID, "obs_key": key})
		}
		_, err := tx.Update(eventsTable, eventID, func(row project.Row) {
			row["picture"] = gfxName
			row["has_obs"] = true
		})
		return err
	})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"filename": filename, "warnings": []string{}, "picture": gfxName,
		"preview_url": fmt.Sprintf("/api/go/projects/%d/events/%d/picture", p.ID(), eventID),
	})
	return nil
}

func resolvePictureKey(p *project.Project, eventID int64) (key, picture string, err error) {
	rows, err := p.Rows(eventsTable)
	if err != nil {
		return "", "", err
	}
	event := findRow(rows, eventID)
	if event == nil {
		return "", "", errAssetMissing
	}
	picture = text(event, "picture", "")
	paths, err := p.Rows("entity_obs_paths")
	if err != nil {
		return "", "", err
	}
	for _, row := range paths {
		if row["entity_type"] == pictureEntityType && project.Int64(row["entity_id"]) == eventID {
			if key := text(row, "obs_key", ""); key != "" {
				return key, picture, nil
			}
		}
	}
	assets, err := p.Rows("project_obs_assets")
	if err != nil {
		return "", "", err
	}
	for _, row := range assets {
		if row["asset_type"] == pictureEntityType && row["gfx_name"] == picture {
			if key := text(row, "obs_key", ""); key != "" {
				return key, picture, nil
			}
		}
	}
	return "", picture, nil
}

func (h *handlers) picture(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	eventID, err := pathID(r, "eid", errAssetTargetMissing)
	if err != nil {
		return err
	}
	key, picture, err := resolvePictureKey(p, eventID)
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
