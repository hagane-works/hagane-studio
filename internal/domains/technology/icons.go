package technology

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const (
	maxCreatorAssetBytes = 10 * 1024 * 1024
	maxIconBlobBytes     = 32 << 20
	iconEntityType       = "tech_icon"
	obsPathsTable        = "entity_obs_paths"
)

func techIconKey(p *project.Project, techID string) (string, bool, error) {
	techs, err := p.Rows(table)
	if err != nil {
		return "", false, err
	}
	row := findTechByID(techs, techID)
	if row == nil {
		return "", false, nil
	}
	paths, err := p.Rows(obsPathsTable)
	if err != nil {
		return "", false, err
	}
	id := project.Int64(row["id"])
	for _, path := range paths {
		if text(path, "entity_type", "") == iconEntityType && project.Int64(path["entity_id"]) == id {
			key := text(path, "obs_key", "")
			return key, key != "", nil
		}
	}
	return "", false, nil
}

func (h *handlers) serveIcon(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	key, found, err := techIconKey(p, r.PathValue("tech"))
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "asset metadata unavailable")
		return
	}
	var body []byte
	if found {
		body, err = p.Blobs().Get(key, maxIconBlobBytes)
	}
	if !found || err != nil {

		detail(w, http.StatusNotFound, "Asset not uploaded")
		return
	}
	header := w.Header()
	header.Set("Content-Type", http.DetectContentType(body))
	header.Set("Cache-Control", "no-store")
	header.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *handlers) listIcons(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	rows, err := p.Rows(table)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "asset metadata unavailable")
		return
	}
	seen := map[string]bool{}
	names := []string{}
	for _, row := range rows {
		techID := text(row, "tech_id", "")
		if techID == "" || !boolean(row, "has_obs") || seen[techID] {
			continue
		}
		seen[techID] = true
		names = append(names, techID)
	}
	sort.Strings(names)
	apiutil.JSON(w, http.StatusOK, map[string]any{"icons": names})
}

func (h *handlers) iconSize(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.open(w, r); !ok {
		return
	}
	size, base := importassets.TechIconSize(r.PathValue("tech"))
	apiutil.JSON(w, http.StatusOK, map[string]any{"width": size.Width, "height": size.Height, "base_game": base})
}

func parseCrop(r *http.Request) (importassets.Crop, error) {
	read := func(name string) (float64, error) {
		raw := strings.TrimSpace(r.FormValue(name))
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
	width, err := read("crop_width")
	if err != nil {
		return importassets.Crop{}, err
	}
	height, err := read("crop_height")
	if err != nil {
		return importassets.Crop{}, err
	}
	return importassets.Crop{X: x, Y: y, Width: width, Height: height}, nil
}

func (h *handlers) uploadIcon(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	id, valid := techRowID(r)
	if !valid {
		detail(w, http.StatusNotFound, "Asset target not found")
		return
	}
	row := findRow(p, id)
	if row == nil {
		detail(w, http.StatusNotFound, "Asset target not found")
		return
	}
	blobs := p.Blobs()
	if strings.TrimSpace(blobs.ProjectUUID()) == "" {
		detail(w, http.StatusServiceUnavailable, "asset metadata unavailable")
		return
	}
	techID := text(row, "tech_id", "")
	r.Body = http.MaxBytesReader(w, r.Body, maxCreatorAssetBytes+(2<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		detail(w, http.StatusUnprocessableEntity, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		detail(w, http.StatusUnprocessableEntity, "file is required")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxCreatorAssetBytes+1))
	_ = file.Close()
	if err != nil {
		detail(w, http.StatusBadRequest, "failed to read upload")
		return
	}
	if len(raw) == 0 {
		detail(w, http.StatusBadRequest, "Empty file")
		return
	}
	if len(raw) > maxCreatorAssetBytes {
		detail(w, http.StatusRequestEntityTooLarge, "image exceeds the 10 MB size limit")
		return
	}
	crop, err := parseCrop(r)
	if err != nil {
		detail(w, http.StatusUnprocessableEntity, "invalid crop coordinates")
		return
	}
	size, _ := importassets.TechIconSize(techID)
	pngBytes, err := importassets.NormalizePNGWithCropToSize(raw, header.Filename, iconEntityType, crop, size)
	if err != nil {
		detail(w, http.StatusBadRequest, "Failed to decode image: "+err.Error())
		return
	}
	sum := sha256.Sum256(pngBytes)
	key, err := importassets.EntityAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "tech", id, iconEntityType, hex.EncodeToString(sum[:]))
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "asset key unavailable")
		return
	}
	if !blobs.Exists(key) {
		if err := blobs.Put(key, "image/png", int64(len(pngBytes)), bytes.NewReader(pngBytes)); err != nil {
			detail(w, http.StatusServiceUnavailable, "object storage write failed; no database pointer was created")
			return
		}
	}

	err = p.Write([]string{table, obsPathsTable}, func(tx *project.Tx) error {
		if tx.Find(table, id) == nil {
			return errTechMissing
		}
		updated := false
		for _, path := range tx.Rows(obsPathsTable) {
			if text(path, "entity_type", "") == iconEntityType && project.Int64(path["entity_id"]) == id {
				if _, err := tx.Update(obsPathsTable, project.Int64(path["id"]), func(row project.Row) { row["obs_key"] = key }); err != nil {
					return err
				}
				updated = true
				break
			}
		}
		if !updated {
			tx.Insert(obsPathsTable, project.Row{"entity_type": iconEntityType, "entity_id": id, "obs_key": key})
		}
		_, err := tx.Update(table, id, func(row project.Row) { row["has_obs"] = true })
		return err
	})
	if err != nil {
		if err == errTechMissing {
			detail(w, http.StatusNotFound, "Asset target not found")
			return
		}
		detail(w, http.StatusServiceUnavailable, "asset metadata commit failed; retry is safe")
		return
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"filename": emitmisc.Slugify(techID) + ".png", "warnings": []string{},
		"icon":        emitmisc.TechIconSpriteName(techID),
		"preview_url": fmt.Sprintf("/api/go/projects/%d/tech-icons/%s", p.ID(), url.PathEscape(techID)),
	})
}

func findRow(p *project.Project, id int64) project.Row {
	rows, err := p.Rows(table)
	if err != nil {
		return nil
	}
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			return row
		}
	}
	return nil
}

func (h *handlers) vanillaIcon(w http.ResponseWriter, r *http.Request) {
	techID := r.PathValue("tech")
	game, icons, ok := h.game.iconFacts()
	if !ok {
		detail(w, http.StatusServiceUnavailable, errNoGame(h))
		return
	}
	rel, resolution, found := icons.resolveTechIcon(techID)
	if !found {
		detail(w, http.StatusNotFound, "vanilla tech icon not found for "+techID)
		return
	}
	body, err := game.PNG(rel)
	if errors.Is(err, fs.ErrNotExist) {
		detail(w, http.StatusNotFound, "vanilla DDS file missing: "+rel)
		return
	}
	if err != nil {
		detail(w, http.StatusInternalServerError, "failed to decode DDS: "+rel)
		return
	}
	writePNG(w, body, resolution)
}

func writePNG(w http.ResponseWriter, data []byte, resolution string) {
	w.Header().Set("Content-Type", "image/png")
	if resolution != "" {
		w.Header().Set("X-Vanilla-Icon-Resolution", resolution)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func errNoGame(h *handlers) string {
	if h.deps.Game == nil {
		return "Hearts of Iron IV folder is not set"
	}
	if _, err := h.deps.Game(); err != nil {
		return err.Error()
	}
	return "Hearts of Iron IV folder is not readable"
}
