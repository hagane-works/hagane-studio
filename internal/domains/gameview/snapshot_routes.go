package gameview

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"unicode/utf8"

	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/project"
)

type zipFile = zip.File

const (
	fontFileLimit      = int64(16 << 20)
	injectionFileLimit = int64(16 << 20)
	injectionTextLimit = int64(128 << 20)
)

type fontSpec struct {
	Path      string
	MediaType string
}

var fontFamilies = map[string]fontSpec{
	"aldrich": {Path: "gfx/fonts/AldrichTNOV13.ttf", MediaType: "font/ttf"},
	"bomb":    {Path: "gfx/fonts/BombardierTNOV6.otf", MediaType: "font/otf"},
	"times":   {Path: "gfx/fonts/TimesNewerRoman-Regular.otf", MediaType: "font/otf"},
}

func (s *Service) fontRead(w http.ResponseWriter, r *http.Request, p *project.Project) {
	values, has := r.URL.Query()["family"]
	if !has || len(values) == 0 {
		detail(w, http.StatusUnprocessableEntity, "family is required")
		return
	}
	family := values[len(values)-1]
	spec, allowed := fontFamilies[family]
	if !allowed {
		detail(w, http.StatusNotFound, "font not found: "+family)
		return
	}
	ref, found, err := guiSnapshot(p)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
		return
	}
	if !found {
		detail(w, http.StatusNotFound, "font not found: "+family)
		return
	}
	archive, err := openSnapshot(p, ref)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
		return
	}
	defer archive.Close()

	var target *zip.File
	for _, member := range archive.File {
		if !member.FileInfo().IsDir() && strings.ReplaceAll(member.Name, "\\", "/") == spec.Path {
			target = member
		}
	}
	if target == nil {
		detail(w, http.StatusNotFound, "font not found: "+family)
		return
	}
	body, err := readMember(target, fontFileLimit)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
		return
	}
	w.Header().Set("Content-Type", spec.MediaType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func injectionSourceKind(name string) (string, string) {
	for _, candidate := range []struct{ prefix, kind string }{
		{"common/scripted_guis/", "gui"},
		{"common/scripted_effects/", "effect"},
	} {
		if !strings.HasPrefix(name, candidate.prefix) {
			continue
		}
		basename := strings.TrimPrefix(name, candidate.prefix)
		if basename != "" && !strings.Contains(basename, "/") && strings.HasSuffix(basename, ".txt") {
			return candidate.kind, basename
		}
	}
	return "", ""
}

func decodeInjectionText(body []byte) string {
	body = trimBOM(body)
	if utf8.Valid(body) {
		return string(body)
	}
	runes := make([]rune, len(body))
	for i, value := range body {
		runes[i] = rune(value)
	}
	return string(runes)
}

func (s *Service) injectionPoints(w http.ResponseWriter, r *http.Request, p *project.Project) {
	ref, found, err := guiSnapshot(p)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
		return
	}
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{"injection_points": []guirender.InjectionPoint{}})
		return
	}
	archive, err := openSnapshot(p, ref)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
		return
	}
	defer archive.Close()
	guiFiles := []guirender.NamedInjectionSource{}
	effectFiles := []guirender.NamedInjectionSource{}
	var retained int64
	for _, member := range archive.File {
		if member.FileInfo().IsDir() {
			continue
		}
		name, err := exportengine.SafePath(strings.ReplaceAll(member.Name, "\\", "/"))
		if err != nil {
			detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
			return
		}
		kind, basename := injectionSourceKind(name)
		if kind == "" {
			continue
		}
		retained += int64(member.UncompressedSize64)
		if retained > injectionTextLimit {
			detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
			return
		}
		body, err := readMember(member, injectionFileLimit)
		if err != nil {
			detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
			return
		}
		source := guirender.NamedInjectionSource{Name: basename, Text: decodeInjectionText(body)}
		if kind == "gui" {
			guiFiles = append(guiFiles, source)
		} else {
			effectFiles = append(effectFiles, source)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"injection_points": guirender.ProjectInjectionPoints(guiFiles, effectFiles),
	})
}

func bundleSnapshot(p *project.Project, file string) (exportengine.Snapshot, error) {
	normalized, err := exportengine.NormalizeGUISourcePath(file)
	if err != nil {
		return exportengine.Snapshot{}, err
	}
	record, err := json.Marshal(p.Record())
	if err != nil {
		return exportengine.Snapshot{}, err
	}
	overlays, err := p.Rows("gui_overlays")
	if err != nil {
		return exportengine.Snapshot{}, err
	}
	bare := strings.TrimPrefix(normalized, "interface/")
	var overlay project.Row
	for _, row := range overlays {
		name := str(row["file"])
		if name == file {
			overlay = row
			break
		}
		if overlay == nil && (name == normalized || name == bare) {
			overlay = row
		}
	}
	if overlay == nil {
		return exportengine.Snapshot{}, errOverlayMissing
	}
	encodedOverlay, err := json.Marshal(overlay)
	if err != nil {
		return exportengine.Snapshot{}, err
	}
	rows, err := p.Rows("project_obs_assets")
	if err != nil {
		return exportengine.Snapshot{}, err
	}
	assets := []json.RawMessage{}
	for _, row := range rows {
		switch str(row["asset_type"]) {
		case "gui_source", "gui_sprite_override", vanillaScaffoldAssetType:
			encoded, err := json.Marshal(row)
			if err != nil {
				return exportengine.Snapshot{}, err
			}
			assets = append(assets, encoded)
		}
	}
	return exportengine.NewSnapshot(p.ID(), record, map[string][]json.RawMessage{
		"gui_overlays": {encodedOverlay}, "project_obs_assets": assets,
	})
}

var errOverlayMissing = errString("GUI overlay not found")

type errString string

func (e errString) Error() string { return string(e) }

func (s *Service) bundleExport(w http.ResponseWriter, r *http.Request, p *project.Project) {
	file := strings.TrimSpace(r.URL.Query().Get("file"))
	normalized, err := exportengine.NormalizeGUISourcePath(file)
	if err != nil {
		detail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	snapshot, err := bundleSnapshot(p, file)
	if err != nil {
		detail(w, http.StatusConflict, err.Error())
		return
	}
	entries, warnings, err := exportengine.BuildNativeGUIBundle(snapshot, p.Blobs().Get)
	if err != nil {
		log.Printf("[gameview] GUI bundle %s: %v", normalized, err)
		detail(w, http.StatusConflict, err.Error())
		return
	}
	if len(entries) == 0 {
		detail(w, http.StatusConflict, "no imported GUI source is available for this file")
		return
	}
	var payload bytes.Buffer
	zw := zip.NewWriter(&payload)
	for _, entry := range entries {
		part, err := zw.Create(entry.Path)
		if err == nil {
			_, err = part.Write(entry.Bytes)
		}
		if err != nil {
			detail(w, http.StatusInternalServerError, "could not create GUI bundle")
			return
		}
	}
	if err := zw.Close(); err != nil {
		detail(w, http.StatusInternalServerError, "could not create GUI bundle")
		return
	}
	if warnings == nil {
		warnings = []string{}
	}
	warningJSON, _ := json.Marshal(warnings)
	w.Header().Set("X-Gui-Export-Lint", url.PathEscape(string(warningJSON)))
	w.Header().Set("Access-Control-Expose-Headers", "X-Gui-Export-Lint")
	stem := strings.TrimSuffix(path.Base(normalized), path.Ext(normalized))
	w.Header().Set("Content-Disposition", `attachment; filename="gui_bundle_`+safeASCIIName(stem)+`.zip"`)
	w.Header().Set("Content-Type", "application/zip")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload.Bytes())
}

func safeASCIIName(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "gui"
	}
	return b.String()
}
