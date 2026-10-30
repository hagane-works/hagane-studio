package gameview

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/project"
)

func (s *Service) read(w http.ResponseWriter, r *http.Request, p *project.Project, route string) {
	known := route == "status" || route == "windows" || route == "window" || route == "loc" ||
		route == "tech-canvas/layout" || guiDataRoutes[route]
	if !known {
		detail(w, http.StatusNotFound, "Not Found")
		return
	}
	query := r.URL.Query()

	if guiDataRoutes[route] {
		if problem := guiDataQueryError(query); problem != "" {
			detail(w, http.StatusUnprocessableEntity, problem)
			return
		}
	}
	if route == "window" {
		if !validSelector(strings.TrimSpace(query.Get("name"))) {
			detail(w, http.StatusUnprocessableEntity, "name is required")
			return
		}
		if file := strings.TrimSpace(query.Get("file")); file != "" {
			if _, err := exportengine.NormalizeGUISourcePath(file); err != nil {
				detail(w, http.StatusUnprocessableEntity, err.Error())
				return
			}
		}
	}
	if route == "tech-canvas/layout" && !validSelector(strings.TrimSpace(query.Get("folder"))) {
		detail(w, http.StatusUnprocessableEntity, "folder is required")
		return
	}
	index, found, err := s.loadIndex(r.Context(), p)
	if err != nil {
		log.Printf("[gameview] project %s index: %v", p.Dir(), err)
		detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
		return
	}
	index = s.inLanguage(r, index)
	switch route {
	case "status":
		writeJSON(w, http.StatusOK, s.statusPayload(p, index, found))
	case "windows":
		rows := []map[string]any{}
		if found {
			rows = index.ListWindows()
		}
		writeJSON(w, http.StatusOK, map[string]any{"windows": rows})
	case "window":
		if !found {
			detail(w, http.StatusNotFound, "window not found: "+strings.TrimSpace(query.Get("name")))
			return
		}
		s.windowRead(w, r, p, index)
	case "tech-canvas/layout":
		techCanvasLayout(w, r, index, found)
	case "loc":
		keys, ok := locKeys(query.Get("keys"))
		if !ok {
			detail(w, http.StatusUnprocessableEntity, "keys must be 1-64 comma-separated localisation keys")
			return
		}

		markup := query.Get("markup") == "1"
		out := map[string]string{}
		if found {
			for _, key := range keys {
				value, ok := index.Localise(key)
				if markup {
					value, ok = index.LocaliseMarkup(key)
				}
				if ok {
					out[key] = value
				}
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"texts": out})
	default:
		dataRead(w, r, route, index, found)
	}
}

func (s *Service) statusPayload(p *project.Project, index *guirender.Index, found bool) map[string]any {
	windows, sprites := 0, 0
	if found && index != nil {
		windows, sprites = index.TotalWindowCount(), index.TotalSpriteCount()
	}
	payload := map[string]any{
		"mod_root": nil, "project_id": p.ID(), "scope": "project_snapshot",
		"extra_roots": []string{}, "resolved": found,
		"windows": windows, "sprites": sprites, "extra_sprites": 0,
		"loc_language": index.Language(),

		"vanilla_root": nil, "vanilla_sprites": 0,
	}
	if found && index != nil {
		if version := index.GameVersion(); version != "" {
			payload["vanilla_root"] = version
			payload["vanilla_sprites"] = index.VanillaSpriteCount()
		}
	}
	return payload
}

func (s *Service) refresh(w http.ResponseWriter, r *http.Request, p *project.Project) {
	s.Refresh(p)
	index, found, err := s.loadIndex(r.Context(), p)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
		return
	}
	writeJSON(w, http.StatusOK, s.statusPayload(p, index, found))
}

func (s *Service) dataVersion(w http.ResponseWriter, r *http.Request, p *project.Project) {
	version, err := dataStamps.get(p)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "project data version unavailable")
		return
	}
	ref, found, err := guiSnapshot(p)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "project data version unavailable")
		return
	}
	if found {
		version += "|" + ref.Key + "|" + ref.Revision
	}

	if layer, identity := s.gameLayerQuiet(); layer != nil {
		version += "|" + identity
	}
	sum := sha256.Sum256([]byte(version))
	writeJSON(w, http.StatusOK, map[string]string{"version": hex.EncodeToString(sum[:12])})
}

func (s *Service) windowRead(w http.ResponseWriter, r *http.Request, p *project.Project, index *guirender.Index) {
	query := r.URL.Query()
	name := strings.TrimSpace(query.Get("name"))
	file := strings.TrimSpace(query.Get("file"))
	normalized := ""
	if file != "" {
		var err error
		if normalized, err = exportengine.NormalizeGUISourcePath(file); err != nil {
			detail(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	var result map[string]any
	var ok bool
	if file != "" {

		if source, sourceOK := index.Source(normalized); sourceOK && len(source) > 0 {
			var err error
			result, ok, err = index.WindowFromSource(name, normalized, source)
			if err != nil {
				detail(w, http.StatusUnprocessableEntity, "GUI source could not be parsed")
				return
			}
		}
	} else {
		result, ok = index.Window(name)
	}
	if !ok {
		message := "window not found: " + name
		if file != "" {
			message = "window not found in requested file: " + name
		}
		detail(w, http.StatusNotFound, message)
		return
	}
	if file == "" {
		file, _ = result["file"].(string)
		var err error
		if normalized, err = exportengine.NormalizeGUISourcePath(file); err != nil {
			detail(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	if file == "" {
		writeJSON(w, http.StatusOK, result)
		return
	}
	overlay, exists, err := overlayForFile(p, file, normalized)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI overlay store unavailable")
		return
	}

	if source, won := index.ProjectFileSource(normalized); won && source != "gui_overlays" {
		if source == "ui_panels" {
			result = attachUIPanelEnvelope(p, normalized, name, result)
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	actualHash, materializable, hashErr := s.overlayScaffoldHash(p, normalized)
	if hashErr != nil {
		log.Printf("[gameview] window %s overlay scaffold: %v", normalized, hashErr)
		materializable = false
	}
	revision := "0"
	selectionHash := actualHash
	drift := false
	if exists {
		revision = overlay.Revision
		if overlay.ScaffoldHash != "" {
			selectionHash = overlay.ScaffoldHash
			drift = overlay.ScaffoldHash != actualHash
		}
	}
	attachEnvelope := func(window map[string]any) map[string]any {
		if !materializable || drift || selectionHash == "" {
			return window
		}
		out := make(map[string]any, len(window)+4)
		for key, value := range window {
			out[key] = value
		}
		out["file"] = normalized
		out["scaffold_hash"] = selectionHash
		out["current_scaffold_hash"] = actualHash
		out["revision"] = revision
		return out
	}
	if exists && len(overlay.Edits) > 0 {

		scaffold, found, err := s.overlayScaffoldFor(p, normalized)
		if err != nil {
			log.Printf("[gameview] overlay %s scaffold: %v", normalized, err)
			writeJSON(w, http.StatusOK, result)
			return
		}
		if found && len(scaffold.Source) > 0 {
			edited, err := exportengine.ApplyGUIPreviewEdits(scaffold.Source, overlay.Edits)
			if err != nil {
				detail(w, http.StatusUnprocessableEntity, "GUI overlay could not be applied")
				return
			}
			editedResult, windowOK, err := index.WindowFromSource(name, normalized, edited)
			if err != nil {
				detail(w, http.StatusUnprocessableEntity, "GUI overlay result could not be parsed")
				return
			}
			if windowOK {
				enrichCandidateLocalisations(editedResult, overlay.Edits, query.Get("lang"))
				writeJSON(w, http.StatusOK, attachEnvelope(editedResult))
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, attachEnvelope(result))
}

func techCanvasLayout(w http.ResponseWriter, r *http.Request, index *guirender.Index, found bool) {
	folder := strings.TrimSpace(r.URL.Query().Get("folder"))
	empty := map[string]any{"folder": folder, "grids": []map[string]any{}, "yearLabels": []map[string]any{}, "source": nil}
	if !found || index == nil {
		writeJSON(w, http.StatusOK, empty)
		return
	}
	window, ok := index.Window("countrytechtreeview")
	if !ok {
		writeJSON(w, http.StatusOK, empty)
		return
	}
	node, _ := window["node"].(map[string]any)
	geometry := guirender.FolderGeometry(guirender.FindNamedNode(node, folder), folder)
	if geometry == nil {
		writeJSON(w, http.StatusOK, empty)
		return
	}
	writeJSON(w, http.StatusOK, geometry)
}

var locKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_.\-]{1,128}$`)

func locKeys(raw string) ([]string, bool) {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	if len(parts) == 0 || len(parts) > 64 {
		return nil, false
	}
	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		key := strings.TrimSpace(part)
		if !locKeyPattern.MatchString(key) {
			return nil, false
		}
		keys = append(keys, key)
	}
	return keys, true
}
