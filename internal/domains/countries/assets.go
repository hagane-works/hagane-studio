package countries

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
	"haganeworks/internal/reftables"
	"haganeworks/internal/workbench"
)

const (
	maxCreatorAssetBytes       = 10 * 1024 * 1024
	maxProjectCoverSourceBytes = 32 * 1024 * 1024
	maxServedAsset             = 32 << 20
)

var errAssetNotUploaded = detail(http.StatusNotFound, "Asset not uploaded")

func serveBlob(p *project.Project, w http.ResponseWriter, key, missing, cacheControl string) error {
	data, err := p.Blobs().Get(key, maxServedAsset)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, os.ErrNotExist) {
		return detail(http.StatusNotFound, missing)
	}
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", http.DetectContentType(data))
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
	return nil
}

func putBlob(p *project.Project, key string, data []byte) error {
	blobs := p.Blobs()
	if blobs.Exists(key) {
		return nil
	}
	if err := blobs.Put(key, "image/png", int64(len(data)), bytes.NewReader(data)); err != nil {
		return detail(http.StatusServiceUnavailable, "object storage write failed; no database pointer was created")
	}
	return nil
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

func parseCrop(r *http.Request) (importassets.Crop, error) {
	var values [4]float64
	for i, name := range []string{"crop_x", "crop_y", "crop_width", "crop_height"} {
		raw := strings.TrimSpace(r.FormValue(name))
		if raw == "" {
			continue
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return importassets.Crop{}, err
		}
		values[i] = v
	}
	return importassets.Crop{X: values[0], Y: values[1], Width: values[2], Height: values[3]}, nil
}

func upsertProjectObsAsset(tx *project.Tx, assetType, gfxName, key string) {
	for _, row := range tx.Rows("project_obs_assets") {
		if str(row["asset_type"]) == assetType && str(row["gfx_name"]) == gfxName {
			_, _ = tx.Update("project_obs_assets", project.Int64(row["id"]), func(r project.Row) { r["obs_key"] = key })
			return
		}
	}
	tx.Insert("project_obs_assets", project.Row{"asset_type": assetType, "gfx_name": gfxName, "obs_key": key, "provenance": map[string]any{}})
}

func projectObsAssetKey(rows []project.Row, assetType, gfxName string) string {
	for _, row := range rows {
		if str(row["asset_type"]) == assetType && str(row["gfx_name"]) == gfxName {
			return strings.TrimSpace(str(row["obs_key"]))
		}
	}
	return ""
}

func validProjectCoverKey(env, projectUUID, key string) bool {
	if importassets.IsNamedAssetKey(env, projectUUID, "project_cover", "thumbnail", key) {
		return true
	}
	return key == strings.Trim(strings.TrimSpace(env), "/")+"/p/"+projectUUID+"/upload/project_cover/thumbnail.png"
}

func uploadCoverHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	blobs := p.Blobs()
	if strings.TrimSpace(blobs.ProjectUUID()) == "" {
		return detail(http.StatusServiceUnavailable, "asset metadata unavailable")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxProjectCoverSourceBytes+(2<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		return detail(http.StatusUnprocessableEntity, "invalid multipart form")
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return detail(http.StatusUnprocessableEntity, "file is required")
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, maxProjectCoverSourceBytes+1))
	_ = file.Close()
	switch {
	case readErr != nil:
		return detail(http.StatusUnprocessableEntity, "cover could not be read")
	case len(raw) == 0:
		return detail(http.StatusUnprocessableEntity, "cover is empty")
	case len(raw) > maxProjectCoverSourceBytes:
		return detail(http.StatusUnprocessableEntity, "cover source is too large")
	}
	pngBytes, err := importassets.NormalizeProjectCover(raw, header.Filename)
	if err != nil {
		return detail(http.StatusUnprocessableEntity, err.Error())
	}
	sum := sha256.Sum256(pngBytes)
	key, err := importassets.NamedAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "project_cover", "thumbnail", hex.EncodeToString(sum[:]))
	if err != nil {
		return detail(http.StatusServiceUnavailable, "asset key unavailable")
	}
	if err := putBlob(p, key, pngBytes); err != nil {
		return err
	}

	err = p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		tx.SetRecord(func(record project.Row) {
			record["picture"] = "thumbnail.png"
			record["updated_at"] = workbench.NowTimestamp()
		})
		upsertProjectObsAsset(tx, "project_cover", "thumbnail.png", key)
		return nil
	})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"filename": "thumbnail.png", "size": len(pngBytes),
		"preview_url": fmt.Sprintf("/api/go/projects/%d/cover?v=%d", p.ID(), time.Now().UnixNano()),
	})
	return nil
}

func readCoverHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	const missing = "Project cover not found"
	rows, err := p.Rows("project_obs_assets")
	if err != nil {
		return err
	}
	key := projectObsAssetKey(rows, "project_cover", "thumbnail.png")
	blobs := p.Blobs()
	if key == "" || !validProjectCoverKey(blobs.EnvPrefix(), blobs.ProjectUUID(), key) {
		return detail(http.StatusNotFound, missing)
	}
	return serveBlob(p, w, key, missing, "private, no-store")
}

func deleteCoverHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var key string
	err := p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		key = projectObsAssetKey(tx.Rows("project_obs_assets"), "project_cover", "thumbnail.png")
		tx.Delete("project_obs_assets", func(row project.Row) bool {
			return str(row["asset_type"]) == "project_cover" && str(row["gfx_name"]) == "thumbnail.png"
		})
		tx.SetRecord(func(record project.Row) {
			record["picture"] = ""
			record["updated_at"] = workbench.NowTimestamp()
		})
		return nil
	})
	if err != nil {
		return err
	}
	blobs := p.Blobs()
	if key != "" && validProjectCoverKey(blobs.EnvPrefix(), blobs.ProjectUUID(), key) {
		if path, err := blobs.Path(key); err == nil {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				log.Printf("delete project cover object %s: %v", key, err)
			}
		}
	}
	apiutil.NoContent(w)
	return nil
}

func countryFlagFilename(tag string) string { return strings.ToUpper(tag) + ".png" }

func countryFlagVariantName(tag, ideology string) string {
	return strings.ToUpper(tag) + "_" + strings.ToLower(ideology)
}

func canonicalIdeologyGroup(registry any, requested string) (string, bool) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", false
	}
	if vanilla := strings.ToLower(requested); reftables.Ideologies().IsGroup(vanilla) {
		return vanilla, true
	}
	if !workbench.RootIDRE.MatchString(requested) {
		return "", false
	}
	for _, group := range registryGroups(registry) {
		if group == requested {
			return group, true
		}
	}
	return "", false
}

func registryGroups(registry any) []string {
	doc, _ := registry.(map[string]any)
	var groups []string
	has := func(id string) bool {
		for _, g := range groups {
			if g == id {
				return true
			}
		}
		return false
	}
	if list, ok := doc["groups"].([]any); ok {
		for _, g := range list {
			if s, ok := g.(string); ok && strings.TrimSpace(s) != "" {
				groups = append(groups, strings.TrimSpace(s))
			}
		}
	}
	if list, ok := doc["root_definitions"].([]any); ok {
		for _, item := range list {
			obj, _ := item.(map[string]any)
			id := strings.TrimSpace(str(obj["id"]))
			if id != "" && !has(id) {
				groups = append(groups, id)
			}
		}
	}
	return groups
}

func uploadFlagHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, err := countryID(r)
	if err != nil {
		return err
	}
	if ideology := strings.TrimSpace(r.URL.Query().Get("ideology")); ideology != "" {
		return uploadFlagVariant(p, w, r, cid, ideology)
	}
	country, err := workbench.GetCountry(p, cid)
	if err != nil {
		return detail(http.StatusNotFound, "Asset target not found")
	}
	blobs := p.Blobs()
	if strings.TrimSpace(blobs.ProjectUUID()) == "" {
		return detail(http.StatusServiceUnavailable, "asset metadata unavailable")
	}
	raw, filename, crop, err := readUpload(w, r)
	if err != nil {
		return err
	}
	pngBytes, err := importassets.NormalizePNGWithCrop(raw, filename, "country_flag", crop)
	if err != nil {
		return detail(http.StatusBadRequest, "Failed to decode image: "+err.Error())
	}
	sum := sha256.Sum256(pngBytes)
	key, err := importassets.EntityAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "flag", cid, "country_flag", hex.EncodeToString(sum[:]))
	if err != nil {
		return detail(http.StatusServiceUnavailable, "asset key unavailable")
	}
	if err := putBlob(p, key, pngBytes); err != nil {
		return err
	}
	tag := str(country["tag"])

	err = p.Write([]string{"entity_obs_paths", "countries"}, func(tx *project.Tx) error {
		if tx.Find("countries", cid) == nil {
			return detail(http.StatusNotFound, "Asset target not found")
		}
		upsertEntityObsPath(tx, "country_flag", cid, key)
		_, err := tx.Update("countries", cid, func(row project.Row) {
			row["flag_image_path"] = "flags/" + strings.TrimSuffix(countryFlagFilename(tag), ".png") + ".tga"
			row["has_obs"] = true
		})
		return err
	})
	if err != nil {
		return err
	}
	item, err := workbench.GetCountry(p, cid)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, item)
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

func readFlagHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, err := countryID(r)
	if err != nil {
		return err
	}
	if ideology := strings.TrimSpace(r.URL.Query().Get("ideology")); ideology != "" {
		return readFlagVariant(p, w, cid, ideology)
	}
	if _, err := workbench.GetCountry(p, cid); err != nil {
		return errAssetNotUploaded
	}
	paths, err := p.Rows("entity_obs_paths")
	if err != nil {
		return err
	}
	key := ""
	for _, row := range paths {
		if str(row["entity_type"]) == "country_flag" && project.Int64(row["entity_id"]) == cid {
			key = strings.TrimSpace(str(row["obs_key"]))
		}
	}
	if key == "" {
		return errAssetNotUploaded
	}
	return serveBlob(p, w, key, "Asset not uploaded", "no-store")
}

func prepareFlagVariant(p *project.Project, cid int64, ideology string) (tag, group string, err error) {
	country, err := workbench.GetCountry(p, cid)
	if err != nil {
		return "", "", errCountryMissing
	}
	group, ok := canonicalIdeologyGroup(p.Record()["ideology_registry"], ideology)
	if !ok {
		return "", "", detail(http.StatusBadRequest, "invalid or unregistered ideology ROOT")
	}
	return str(country["tag"]), group, nil
}

func uploadFlagVariant(p *project.Project, w http.ResponseWriter, r *http.Request, cid int64, ideology string) error {
	tag, group, err := prepareFlagVariant(p, cid, strings.ToLower(ideology))
	if err != nil {
		return err
	}
	key, err := persistNamedImage(p, w, r, "flag", countryFlagVariantName(tag, group), "country_flag")
	if err != nil {
		return err
	}

	err = p.Write([]string{"countries", "project_obs_assets"}, func(tx *project.Tx) error {
		row := tx.Find("countries", cid)
		if row == nil {
			return errCountryMissing
		}
		if str(row["tag"]) != tag {
			return detail(http.StatusServiceUnavailable, "asset metadata commit failed; retry is safe")
		}
		paths := map[string]any{}
		for k, v := range stringMapOf(row["flag_paths_by_ideology"]) {
			paths[k] = v
		}
		paths[group] = "flags/" + strings.ToUpper(tag) + "_" + group + ".tga"
		if _, err := tx.Update("countries", cid, func(r project.Row) { r["flag_paths_by_ideology"] = paths }); err != nil {
			return err
		}
		upsertProjectObsAsset(tx, "country_flag_ideology", countryFlagVariantName(tag, group), key)
		return nil
	})
	if err != nil {
		return err
	}
	item, err := workbench.GetCountry(p, cid)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, item)
	return nil
}

func readFlagVariant(p *project.Project, w http.ResponseWriter, cid int64, ideology string) error {
	const missing = "No per-ideology flag for this ideology"
	tag, group, err := prepareFlagVariant(p, cid, strings.ToLower(ideology))
	if err != nil {

		var status *apiutil.StatusError
		if errors.As(err, &status) && status.Status == http.StatusNotFound {
			return detail(http.StatusNotFound, missing)
		}
		return detail(http.StatusServiceUnavailable, "asset metadata unavailable")
	}
	country, _ := workbench.GetCountry(p, cid)
	if stringMapOf(country["flag_paths_by_ideology"])[group] == "" {
		return detail(http.StatusNotFound, missing)
	}
	rows, err := p.Rows("project_obs_assets")
	if err != nil {
		return err
	}
	logical := countryFlagVariantName(tag, group)
	key := projectObsAssetKey(rows, "country_flag_ideology", logical)
	blobs := p.Blobs()
	if key == "" || !importassets.IsNamedAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "flag", logical, key) {
		return detail(http.StatusNotFound, missing)
	}
	return serveBlob(p, w, key, missing, "no-store")
}

func persistNamedImage(p *project.Project, w http.ResponseWriter, r *http.Request, objectKind, logicalName, imageKind string) (string, error) {
	raw, filename, crop, err := readUpload(w, r)
	if err != nil {
		return "", err
	}
	pngBytes, err := importassets.NormalizePNGWithCrop(raw, filename, imageKind, crop)
	if err != nil {
		return "", detail(http.StatusBadRequest, "Failed to decode image: "+err.Error())
	}
	blobs := p.Blobs()
	sum := sha256.Sum256(pngBytes)
	key, err := importassets.NamedAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), objectKind, logicalName, hex.EncodeToString(sum[:]))
	if err != nil {
		return "", detail(http.StatusServiceUnavailable, "asset key unavailable")
	}
	return key, putBlob(p, key, pngBytes)
}

func uploadIdeologyIconHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	group, ok := canonicalIdeologyGroup(p.Record()["ideology_registry"], r.PathValue("group"))
	if !ok {
		return detail(http.StatusBadRequest, "invalid or unregistered ideology ROOT")
	}
	key, err := persistNamedImage(p, w, r, "ideology_icon", group, "ideology_icon")
	if err != nil {
		return err
	}

	err = p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		record := tx.Record()
		if current, ok := canonicalIdeologyGroup(record["ideology_registry"], group); !ok || current != group {
			return detail(http.StatusServiceUnavailable, "asset metadata commit failed; retry is safe")
		}
		paths := map[string]any{}
		for k, v := range stringMapOf(record["ideology_icon_paths"]) {
			paths[k] = v
		}
		paths[group] = "ideology_icons/" + group + ".dds"
		tx.SetRecord(func(next project.Row) {
			next["ideology_icon_paths"] = paths
			next["updated_at"] = workbench.NowTimestamp()
		})
		upsertProjectObsAsset(tx, "ideology_icon", group, key)
		return nil
	})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, workbench.ProjectRecord(p))
	return nil
}

func readIdeologyIconHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	const missing = "No icon uploaded for this ideology group"
	record := p.Record()
	group, ok := canonicalIdeologyGroup(record["ideology_registry"], r.PathValue("group"))
	if !ok {
		return detail(http.StatusBadRequest, "invalid or unregistered ideology ROOT")
	}
	if stringMapOf(record["ideology_icon_paths"])[group] == "" {
		return detail(http.StatusNotFound, missing)
	}
	rows, err := p.Rows("project_obs_assets")
	if err != nil {
		return err
	}
	key := projectObsAssetKey(rows, "ideology_icon", group)
	blobs := p.Blobs()
	if key == "" || !importassets.IsNamedAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "ideology_icon", group, key) {
		return detail(http.StatusNotFound, missing)
	}
	return serveBlob(p, w, key, missing, "no-store")
}
