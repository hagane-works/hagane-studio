package focus

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const maxCreatorAssetBytes = 10 * 1024 * 1024

var focusGFXToken = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

type projectIconRef struct {
	Name   string
	OBSKey string
}

func projectFocusIcons(p *project.Project) ([]projectIconRef, error) {
	rows, err := p.Rows("project_obs_icons")
	if err != nil {
		return nil, err
	}
	out := []projectIconRef{}
	for _, row := range rows {
		ref := projectIconRef{Name: colStr(row, "gfx_name", ""), OBSKey: colStr(row, "obs_key", "")}
		if ref.Name != "" {
			out = append(out, ref)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (h *handlers) listIcons(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	refs, err := projectFocusIcons(p)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		names = append(names, ref.Name)
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{"icons": names})
	return nil
}

func (h *handlers) serveIcon(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("gfx")
	refs, err := projectFocusIcons(p)
	if err != nil {
		return err
	}
	key := ""
	for _, ref := range refs {
		if ref.Name == name {
			key = ref.OBSKey
			break
		}
	}
	if strings.TrimSpace(key) == "" {
		return apiutil.NotFoundError("Asset not uploaded")
	}
	data, err := p.Blobs().Get(key, 32<<20)
	if err != nil {
		return apiutil.NotFoundError("Asset not uploaded")
	}
	w.Header().Set("Content-Type", http.DetectContentType(data))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
	return nil
}

type FocusIconLibraryEntry struct {
	Name         string  `json:"name"`
	ThumbnailURL string  `json:"thumbnail_url"`
	Source       string  `json:"source"`
	DLC          *string `json:"dlc"`
	Category     *string `json:"category"`
}

type FocusIconLibraryResponse struct {
	Icons    []FocusIconLibraryEntry `json:"icons"`
	Total    int                     `json:"total"`
	Returned int                     `json:"returned"`
	Offset   int                     `json:"offset"`
	Limit    int                     `json:"limit"`
}

func buildFocusIconLibrary(projectID int64, vanilla []vanillaFocusIcon, refs []projectIconRef, source, query string, limit, offset int) (FocusIconLibraryResponse, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		source = "all"
	}
	if source != "all" && source != "vanilla" && source != "project" {
		return FocusIconLibraryResponse{}, fmt.Errorf("invalid source: %q (vanilla|project|all)", source)
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 1
	} else if limit > 5000 {
		limit = 5000
	}
	all := make([]FocusIconLibraryEntry, 0)
	if source == "all" || source == "vanilla" {
		for _, entry := range vanilla {
			if query != "" && !strings.Contains(strings.ToLower(entry.Name), query) {
				continue
			}
			all = append(all, FocusIconLibraryEntry{
				Name: entry.Name, ThumbnailURL: "/api/go/vanilla/focus-icon/" + url.PathEscape(entry.Name),
				Source: "vanilla", Category: entry.Category,
			})
		}
	}
	if source == "all" || source == "project" {
		for _, ref := range refs {
			if query != "" && !strings.Contains(strings.ToLower(ref.Name), query) {
				continue
			}
			category := "other"
			if strings.HasPrefix(ref.Name, "GFX_focus_") {
				category = "focus"
			} else if strings.HasPrefix(ref.Name, "GFX_goal_") {
				category = "goal"
			}
			all = append(all, FocusIconLibraryEntry{
				Name:         ref.Name,
				ThumbnailURL: fmt.Sprintf("/api/go/projects/%d/focus-icons/%s", projectID, url.PathEscape(ref.Name)),
				Source:       "project", Category: &category,
			})
		}
	}
	response := FocusIconLibraryResponse{Total: len(all), Offset: offset, Limit: limit, Icons: []FocusIconLibraryEntry{}}
	if offset < len(all) {
		end := min(len(all), offset+limit)
		response.Icons = append(response.Icons, all[offset:end]...)
	}
	response.Returned = len(response.Icons)
	return response, nil
}

func parseLibraryInt(r *http.Request, key string, fallback int) (int, error) {
	raw, present := r.URL.Query()[key]
	if !present {
		return fallback, nil
	}
	if len(raw) != 1 {
		return 0, errors.New("invalid query value")
	}
	return strconv.Atoi(raw[0])
}

func (h *handlers) iconLibrary(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	limit, err := parseLibraryInt(r, "limit", 1000)
	if err != nil {
		return apiutil.ValidationError("limit must be an integer")
	}
	offset, err := parseLibraryInt(r, "offset", 0)
	if err != nil {
		return apiutil.ValidationError("offset must be an integer")
	}
	refs, err := projectFocusIcons(p)
	if err != nil {
		return err
	}

	var vanilla []vanillaFocusIcon
	if _, index, err := h.vanillaIcons(); err == nil {
		vanilla = index.icons
	}
	response, err := buildFocusIconLibrary(p.ID(), vanilla, refs, r.URL.Query().Get("source"), r.URL.Query().Get("q"), limit, offset)
	if err != nil {
		return apiutil.BadRequestError(err.Error())
	}
	w.Header().Set("Cache-Control", "private, no-store")
	apiutil.JSON(w, http.StatusOK, response)
	return nil
}

func normalizeFocusGFXName(raw string) (string, error) {
	original := raw
	name := strings.TrimSpace(raw)
	if dot := strings.LastIndexByte(name, '.'); dot > 0 {
		ext := strings.ToLower(name[dot+1:])
		if ext == "dds" || ext == "png" || ext == "tga" {
			name = name[:dot]
		}
	}
	name = strings.TrimSpace(strings.ReplaceAll(name, `\`, "/"))
	if slash := strings.LastIndexByte(name, '/'); slash >= 0 {
		name = strings.TrimSpace(name[slash+1:])
	}
	if name == "" {
		return "", errors.New("gfx_name is required")
	}
	if !focusGFXToken.MatchString(name) {
		return "", fmt.Errorf("gfx_name may only contain letters, digits, and underscores (got %q)", original)
	}
	return name, nil
}

func statusError(status int, detail string) error {
	return &apiutil.StatusError{Status: status, Body: map[string]string{"detail": detail}}
}

func putBlob(p *project.Project, key string, png []byte) error {
	blobs := p.Blobs()
	if blobs.Exists(key) {
		return nil
	}
	if err := blobs.Put(key, "image/png", int64(len(png)), bytes.NewReader(png)); err != nil {
		return statusError(http.StatusServiceUnavailable, "object storage write failed; no database pointer was created")
	}
	return nil
}

func containsString(v any, s string) bool {
	switch value := v.(type) {
	case string:
		return value == s
	case map[string]any:
		for _, item := range value {
			if containsString(item, s) {
				return true
			}
		}
	case []any:
		for _, item := range value {
			if containsString(item, s) {
				return true
			}
		}
	}
	return false
}

func guardFocusIconConsumers(tx *project.Tx, rv *revisions, gfxName string) error {
	trees := map[int64]bool{}
	for _, node := range tx.Rows("focus_nodes") {
		if colStr(node, "icon", "") == gfxName || colStr(node, "text_icon", "") == gfxName ||
			containsString(node["dynamic_icon"], gfxName) || containsString(node["raw_extras"], gfxName) {
			trees[project.Int64(node["focus_tree_id"])] = true
		}
	}
	ids := make([]int64, 0, len(trees))
	for id := range trees {
		if tx.Find("focus_trees", id) != nil {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		if _, err := rv.advance(tx, id); err != nil {
			return err
		}
	}
	return nil
}

func upsertProjectIcon(tx *project.Tx, gfxName, key string) error {
	for _, row := range tx.Rows("project_obs_icons") {
		if colStr(row, "gfx_name", "") == gfxName {
			_, err := tx.Update("project_obs_icons", project.Int64(row["id"]), func(r project.Row) { r["obs_key"] = key })
			return err
		}
	}
	tx.Insert("project_obs_icons", project.Row{"gfx_name": gfxName, "obs_key": key})
	return nil
}

func parseMultipart(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxCreatorAssetBytes+(2<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		return apiutil.ValidationError("invalid multipart form")
	}
	return nil
}

func formFile(r *http.Request) (raw []byte, filename string, err error) {
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, "", apiutil.ValidationError("file is required")
	}
	defer file.Close()
	raw, err = io.ReadAll(io.LimitReader(file, maxCreatorAssetBytes+1))
	if err != nil {
		return nil, "", apiutil.BadRequestError("failed to read upload")
	}
	return raw, header.Filename, nil
}

func (h *handlers) uploadNamedIcon(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	if err := parseMultipart(w, r); err != nil {
		return err
	}
	gfxName, err := normalizeFocusGFXName(r.FormValue("gfx_name"))
	if err != nil {
		return apiutil.BadRequestError(err.Error())
	}
	raw, filename, err := formFile(r)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return apiutil.BadRequestError("empty file")
	}
	if len(raw) > maxCreatorAssetBytes {
		return apiutil.BadRequestError("image exceeds the 10 MB size limit")
	}
	pngBytes, err := importassets.NormalizePNG(raw, filename, "focus_icon")
	if err != nil {
		return apiutil.BadRequestError("could not decode image")
	}
	sum := sha256.Sum256(pngBytes)
	blobs := p.Blobs()
	key, err := importassets.NamedAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "focus", gfxName, hex.EncodeToString(sum[:]))
	if err != nil {
		return statusError(http.StatusServiceUnavailable, "asset key unavailable")
	}
	if err := putBlob(p, key, pngBytes); err != nil {
		return err
	}
	err = p.Write([]string{"focus_trees", "focus_nodes", "project_obs_icons"}, func(tx *project.Tx) error {
		if err := guardFocusIconConsumers(tx, rv, gfxName); err != nil {
			return err
		}
		return upsertProjectIcon(tx, gfxName, key)
	})
	if err != nil {
		return err
	}
	rv.commit(p)
	respond(w, rv, http.StatusOK, map[string]any{"gfx_name": gfxName, "saved": true})
	return nil
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

func (h *handlers) uploadNodeIcon(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	targetMissing := apiutil.NotFoundError("Asset target not found")
	treeID, err := strconv.ParseInt(r.PathValue("tid"), 10, 64)
	if err != nil || treeID <= 0 {
		return targetMissing
	}
	nodeID, err := strconv.ParseInt(r.PathValue("nid"), 10, 64)
	if err != nil || nodeID <= 0 {
		return targetMissing
	}

	nodes, err := p.Rows("focus_nodes")
	if err != nil {
		return err
	}
	focusID, found := "", false
	for _, node := range nodes {
		if project.Int64(node["id"]) == nodeID && project.Int64(node["focus_tree_id"]) == treeID {
			focusID, found = colStr(node, "focus_id", ""), true
			break
		}
	}
	if !found {
		return targetMissing
	}
	if err := parseMultipart(w, r); err != nil {
		return err
	}
	raw, filename, err := formFile(r)
	if err != nil {
		return err
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
	pngBytes, err := importassets.NormalizePNGWithCrop(raw, filename, "focus_icon", crop)
	if err != nil {
		return apiutil.BadRequestError("Failed to decode image: " + err.Error())
	}
	gfxName := emitmisc.FocusIconSpriteName(focusID)
	sum := sha256.Sum256(pngBytes)
	blobs := p.Blobs()
	key, err := importassets.EntityAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "focus", nodeID, "focus_icon", hex.EncodeToString(sum[:]))
	if err != nil {
		return statusError(http.StatusServiceUnavailable, "asset key unavailable")
	}
	if err := putBlob(p, key, pngBytes); err != nil {
		return err
	}

	err = p.Write([]string{"focus_trees", "focus_nodes", "project_obs_icons"}, func(tx *project.Tx) error {
		if tx.Find("focus_trees", treeID) == nil {
			return targetMissing
		}
		if _, err := rv.advance(tx, treeID); err != nil {
			return err
		}
		if err := guardFocusIconConsumers(tx, rv, gfxName); err != nil {
			return err
		}
		if err := upsertProjectIcon(tx, gfxName, key); err != nil {
			return err
		}
		node := tx.Find("focus_nodes", nodeID)
		if node == nil || project.Int64(node["focus_tree_id"]) != treeID {
			return targetMissing
		}
		_, err := tx.Update("focus_nodes", nodeID, func(row project.Row) { row["icon"] = gfxName })
		return err
	})
	if err != nil {
		return err
	}
	rv.commit(p)
	respond(w, rv, http.StatusOK, map[string]any{
		"filename": "focus_" + emitmisc.Slugify(focusID) + ".png", "warnings": []string{},
		"icon":        gfxName,
		"preview_url": fmt.Sprintf("/api/go/projects/%d/focus-icons/%s", p.ID(), url.PathEscape(gfxName)),
	})
	return nil
}
