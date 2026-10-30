package decisions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const (
	maxIconUploadBytes = 10 * 1024 * 1024
	maxIconReadBytes   = 32 * 1024 * 1024

	decisionIconType = "decision_icon"
	categoryIconType = "decision_category_icon"
)

type iconTarget struct {
	category    bool
	id          int64
	logicalID   string
	projectUUID string
}

func (t iconTarget) table() string {
	if t.category {
		return categoriesTable
	}
	return decisionsTable
}

func (t iconTarget) entityType() string {
	if t.category {
		return categoryIconType
	}
	return decisionIconType
}

func (t iconTarget) spec() (filename, gfxName string) {
	slug := emitmisc.Slugify(t.logicalID)
	if t.category {
		return "decision_category_" + slug + ".png", emitmisc.DecisionCategoryIconSpriteName(t.logicalID)
	}
	return "decision_" + slug + ".png", emitmisc.DecisionIconSpriteName(t.logicalID)
}

func (h *handlers) iconRoute(w http.ResponseWriter, r *http.Request, p *project.Project, rest []string) bool {
	var raw, action string
	category := false
	switch {
	case len(rest) == 2 && (rest[1] == "upload-icon" || rest[1] == "icon"):
		raw, action = rest[0], rest[1]
	case len(rest) == 3 && rest[0] == "categories" && (rest[2] == "upload-icon" || rest[2] == "icon"):
		raw, action, category = rest[1], rest[2], true
	default:
		return false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		detail(w, http.StatusNotFound, "Asset target not found")
		return true
	}
	if action == "upload-icon" {
		if r.Method != http.MethodPost {
			detail(w, http.StatusMethodNotAllowed, "POST only")
			return true
		}
		h.uploadIcon(w, r, p, category, id)
		return true
	}
	if r.Method != http.MethodGet {
		detail(w, http.StatusMethodNotAllowed, "GET only")
		return true
	}
	h.readIcon(w, p, category, id)
	return true
}

func prepareTarget(p *project.Project, category bool, id int64) (iconTarget, error) {
	t := iconTarget{category: category, id: id, projectUUID: p.Blobs().ProjectUUID()}
	rows, err := p.Rows(t.table())
	if err != nil {
		return t, err
	}
	for _, row := range rows {
		if project.Int64(row["id"]) != id {
			continue
		}
		if category {
			t.logicalID = text(row["category_id"])
		} else {
			t.logicalID = text(row["decision_id"])
		}
		if strings.TrimSpace(t.projectUUID) == "" {
			return t, errors.New("project UUID missing")
		}
		return t, nil
	}
	return t, errAssetTargetMissing
}

var errAssetTargetMissing = errors.New("asset target not found")

func (h *handlers) uploadIcon(w http.ResponseWriter, r *http.Request, p *project.Project, category bool, id int64) {
	target, err := prepareTarget(p, category, id)
	if errors.Is(err, errAssetTargetMissing) {
		detail(w, http.StatusNotFound, "Asset target not found")
		return
	}
	if err != nil {
		log.Printf("prepare decision icon target: %v", err)
		detail(w, http.StatusServiceUnavailable, "asset metadata unavailable")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxIconUploadBytes+(2<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		detail(w, http.StatusUnprocessableEntity, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		detail(w, http.StatusUnprocessableEntity, "file is required")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxIconUploadBytes+1))
	_ = file.Close()
	if err != nil {
		detail(w, http.StatusBadRequest, "failed to read upload")
		return
	}
	if len(raw) == 0 {
		detail(w, http.StatusBadRequest, "Empty file")
		return
	}
	if len(raw) > maxIconUploadBytes {
		detail(w, http.StatusRequestEntityTooLarge, "image exceeds the 10 MB size limit")
		return
	}
	crop, err := parseCrop(r)
	if err != nil {
		detail(w, http.StatusUnprocessableEntity, "invalid crop coordinates")
		return
	}
	filename, gfxName, err := persistIcon(p, target, raw, header.Filename, crop)
	if err != nil {
		var status *statusDetail
		if errors.As(err, &status) {
			detail(w, status.status, status.detail)
			return
		}
		if errors.Is(err, errAssetTargetMissing) {
			detail(w, http.StatusNotFound, "Asset target not found")
			return
		}
		log.Printf("commit decision icon %d: %v", id, err)
		detail(w, http.StatusServiceUnavailable, "asset metadata commit failed; retry is safe")
		return
	}
	preview := fmt.Sprintf("/api/go/projects/%d/decisions/%d/icon", p.ID(), id)
	if category {
		preview = fmt.Sprintf("/api/go/projects/%d/decisions/categories/%d/icon", p.ID(), id)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"filename": filename, "warnings": []string{}, "icon": gfxName, "preview_url": preview,
	})
}

type statusDetail struct {
	status int
	detail string
}

func (e *statusDetail) Error() string { return e.detail }

func persistIcon(p *project.Project, target iconTarget, raw []byte, filename string, crop importassets.Crop) (string, string, error) {
	storedName, gfxName := target.spec()
	png, err := importassets.NormalizePNGWithCrop(raw, filename, target.entityType(), crop)
	if err != nil {
		return "", "", &statusDetail{http.StatusBadRequest, "Failed to decode image: " + err.Error()}
	}
	sum := sha256.Sum256(png)
	blobs := p.Blobs()
	key, err := importassets.EntityAssetKey(blobs.EnvPrefix(), target.projectUUID, "decision", target.id, target.entityType(), hex.EncodeToString(sum[:]))
	if err != nil {
		return "", "", &statusDetail{http.StatusServiceUnavailable, "asset key unavailable"}
	}
	if !blobs.Exists(key) {
		if err := blobs.Put(key, "image/png", int64(len(png)), bytes.NewReader(png)); err != nil {
			log.Printf("decision icon write %s: %v", key, err)
			return "", "", &statusDetail{http.StatusServiceUnavailable, "object storage write failed; no database pointer was created"}
		}
	}
	projectID := p.ID()
	err = p.Write([]string{"entity_obs_paths", target.table()}, func(tx *project.Tx) error {
		if tx.Find(target.table(), target.id) == nil {
			return errAssetTargetMissing
		}
		entityType := target.entityType()
		mapped := false
		for _, row := range tx.Rows("entity_obs_paths") {
			if text(row["entity_type"]) == entityType && project.Int64(row["entity_id"]) == target.id {
				if _, err := tx.Update("entity_obs_paths", project.Int64(row["id"]), func(r project.Row) {
					r["project_id"] = projectID
					r["obs_key"] = key
				}); err != nil {
					return err
				}
				mapped = true
				break
			}
		}
		if !mapped {
			tx.Insert("entity_obs_paths", project.Row{"entity_type": entityType, "entity_id": target.id, "obs_key": key})
		}
		_, err := tx.Update(target.table(), target.id, func(row project.Row) {
			row["icon"] = gfxName
			if !target.category {
				row["has_obs"] = true
			}
		})
		return err
	})
	if err != nil {
		return "", "", err
	}
	return storedName, gfxName, nil
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

func mappedKey(paths []project.Row, entityType string, id int64) string {
	for _, row := range paths {
		if text(row["entity_type"]) == entityType && project.Int64(row["entity_id"]) == id {
			if key := text(row["obs_key"]); key != "" {
				return key
			}
		}
	}
	return ""
}

func importedKey(assets []project.Row, gfxName string, assetTypes ...string) string {
	if gfxName == "" {
		return ""
	}
	for _, assetType := range assetTypes {
		for _, row := range assets {
			if text(row["asset_type"]) == assetType && text(row["gfx_name"]) == gfxName {
				if key := text(row["obs_key"]); key != "" {
					return key
				}
			}
		}
	}
	return ""
}

func (h *handlers) readIcon(w http.ResponseWriter, p *project.Project, category bool, id int64) {
	target := iconTarget{category: category, id: id}
	rows, err := p.Rows(target.table())
	if err != nil {
		h.iconUnavailable(w, err)
		return
	}
	var row project.Row
	for _, candidate := range rows {
		if project.Int64(candidate["id"]) == id {
			row = candidate
			break
		}
	}
	if row == nil {
		detail(w, http.StatusNotFound, "Asset not uploaded")
		return
	}
	paths, err := p.Rows("entity_obs_paths")
	if err != nil {
		h.iconUnavailable(w, err)
		return
	}
	assets, err := p.Rows("project_obs_assets")
	if err != nil {
		h.iconUnavailable(w, err)
		return
	}
	icon := text(row["icon"])
	key := mappedKey(paths, target.entityType(), id)
	if key == "" {

		key = importedKey(assets, icon, target.entityType(), decisionIconType)
	}
	if key != "" && serveBlob(w, p, key) {
		return
	}
	prefix := "GFX_decision_"
	if category {
		prefix = "GFX_decision_category_"
	}
	if h.serveGameSprite(w, spriteName(icon, prefix)) {
		return
	}
	detail(w, http.StatusNotFound, "Asset not uploaded")
}

func (h *handlers) iconUnavailable(w http.ResponseWriter, err error) {
	log.Printf("resolve decision icon: %v", err)
	detail(w, http.StatusServiceUnavailable, "asset metadata unavailable")
}

func serveBlob(w http.ResponseWriter, p *project.Project, key string) bool {
	data, err := p.Blobs().Get(key, maxIconReadBytes)
	if err != nil {
		log.Printf("decision icon %s: %v", key, err)
		return false
	}
	writeImage(w, data)
	return true
}

func writeImage(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", http.DetectContentType(data))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func spriteName(token, prefix string) string {
	token = strings.TrimSpace(token)
	switch {
	case token == "":
		return ""
	case strings.HasPrefix(token, "GFX_"):
		return token
	case strings.HasPrefix(token, "decision_"):
		return "GFX_" + token
	default:
		return prefix + token
	}
}

func (h *handlers) iconCollection(w http.ResponseWriter, r *http.Request) {
	p, err := h.deps.Open(r)
	if err != nil {
		apiutil.Fail(w, err)
		return
	}
	if r.Method != http.MethodGet {
		detail(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	tables := map[string][]project.Row{}
	for _, table := range []string{decisionsTable, categoriesTable, "entity_obs_paths", "project_obs_assets"} {
		if tables[table], err = p.Rows(table); err != nil {
			h.iconUnavailable(w, err)
			return
		}
	}
	paths, assets := tables["entity_obs_paths"], tables["project_obs_assets"]
	name := strings.TrimSpace(r.PathValue("name"))
	if r.PathValue("name") == "" {
		writeJSON(w, http.StatusOK, map[string]any{"icons": listIconNames(tables[decisionsTable], paths, assets)})
		return
	}
	if key := resolveIconName(name, tables[decisionsTable], tables[categoriesTable], paths, assets); key != "" && serveBlob(w, p, key) {
		return
	}
	if h.serveGameSprite(w, spriteName(name, "GFX_decision_")) ||
		h.serveGameSprite(w, spriteName(name, "GFX_decision_category_")) {
		return
	}
	detail(w, http.StatusNotFound, "Decision icon not found")
}

func listIconNames(decisions, paths, assets []project.Row) []string {
	type ref struct {
		name, key string
		priority  int
		entity    int64
	}
	var refs []ref
	for _, d := range decisions {
		name := text(d["icon"])
		if !boolOf(d["has_obs"]) || name == "" {
			continue
		}
		id := project.Int64(d["id"])
		if key := mappedKey(paths, decisionIconType, id); key != "" {
			refs = append(refs, ref{name, key, 0, id})
		}
	}
	for _, a := range assets {
		if text(a["asset_type"]) == decisionIconType && text(a["gfx_name"]) != "" && text(a["obs_key"]) != "" {
			refs = append(refs, ref{text(a["gfx_name"]), text(a["obs_key"]), 1, 0})
		}
	}
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].name != refs[j].name {
			return refs[i].name < refs[j].name
		}
		if refs[i].priority != refs[j].priority {
			return refs[i].priority < refs[j].priority
		}
		return refs[i].entity < refs[j].entity
	})
	names := []string{}
	last := ""
	for _, r := range refs {
		if r.name == last {
			continue
		}
		last = r.name
		names = append(names, r.name)
	}
	return names
}

func resolveIconName(name string, decisions, categories, paths, assets []project.Row) string {
	if name == "" {
		return ""
	}
	for _, d := range decisions {
		if text(d["icon"]) == name && boolOf(d["has_obs"]) {
			if key := mappedKey(paths, decisionIconType, project.Int64(d["id"])); key != "" {
				return key
			}
		}
	}
	for _, c := range categories {
		if text(c["icon"]) == name {
			if key := mappedKey(paths, categoryIconType, project.Int64(c["id"])); key != "" {
				return key
			}
		}
	}
	return importedKey(assets, name, decisionIconType)
}
