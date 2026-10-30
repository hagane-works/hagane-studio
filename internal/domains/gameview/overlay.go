package gameview

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/project"
)

const (
	scaffoldImported = "import"
	scaffoldVanilla  = "vanilla"

	vanillaScaffoldAssetType = "gui_vanilla_scaffold"

	importedSourceLimit = int64(16 << 20)

	timestampLayout = "2006-01-02T15:04:05.999999"
)

type guiOverlay struct {
	ID            int64
	ProjectID     int64
	File          string
	Source        string
	ScaffoldHash  string
	SchemaVersion int
	Edits         []map[string]any
	Revision      string
}

func overlayRevision(updatedAt any) string {
	raw := strings.TrimSpace(str(updatedAt))
	if raw == "" {
		return "0"
	}
	for _, layout := range []string{timestampLayout, time.RFC3339Nano} {
		if t, err := time.Parse(layout, strings.TrimSuffix(raw, "+00:00")); err == nil {
			return t.UTC().Format(time.RFC3339Nano)
		}
	}
	return raw
}

func plainEdits(value any) []map[string]any {
	out := []map[string]any{}
	if value == nil {
		return out
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return out
	}
	if json.Unmarshal(raw, &out) != nil || out == nil {
		return []map[string]any{}
	}
	return out
}

func overlayFromRow(row project.Row) guiOverlay {
	return guiOverlay{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		File: str(row["file"]), Source: str(row["source"]), ScaffoldHash: str(row["scaffold_hash"]),
		SchemaVersion: int(project.Int64(row["schema_version"])), Edits: plainEdits(row["edits"]),
		Revision: overlayRevision(row["updated_at"]),
	}
}

func overlayByFile(p *project.Project, file string) (guiOverlay, bool, error) {
	rows, err := p.Rows("gui_overlays")
	if err != nil {
		return guiOverlay{}, false, err
	}
	for _, row := range rows {
		if str(row["file"]) == file {
			return overlayFromRow(row), true, nil
		}
	}
	return guiOverlay{}, false, nil
}

func overlayForFile(p *project.Project, requested, normalized string) (guiOverlay, bool, error) {
	aliases := []string{
		strings.Trim(strings.ReplaceAll(requested, "\\", "/"), "/"),
		strings.TrimPrefix(normalized, "interface/"),
		normalized,
	}
	seen := map[string]bool{}
	for _, file := range aliases {
		if file == "" || seen[file] {
			continue
		}
		seen[file] = true
		overlay, exists, err := overlayByFile(p, file)
		if err != nil || exists {
			return overlay, exists, err
		}
	}
	return guiOverlay{}, false, nil
}

type overlayScaffold struct {
	Source []byte
	Hash   string
	Origin string
}

func newScaffold(source []byte, origin string) overlayScaffold {
	sum := sha256.Sum256(source)
	return overlayScaffold{Source: source, Hash: hex.EncodeToString(sum[:]), Origin: origin}
}

func (s *Service) overlayScaffoldFor(p *project.Project, file string) (overlayScaffold, bool, error) {
	normalized, err := exportengine.NormalizeGUISourcePath(file)
	if err != nil {
		return overlayScaffold{}, false, err
	}
	ref, attached, err := guiSnapshot(p)
	if err != nil {
		return overlayScaffold{}, false, err
	}
	if attached {
		source, found, err := snapshotFile(p, ref, normalized, importedSourceLimit)
		if err != nil {
			return overlayScaffold{}, false, err
		}
		if found && len(source) > 0 {
			return newScaffold(source, scaffoldImported), true, nil
		}
	}
	if vanilla := s.BaseIndex(); vanilla != nil && vanilla.HasOwnFile(normalized) {
		if source, ok := vanilla.Source(normalized); ok && len(source) > 0 {
			return newScaffold(source, scaffoldVanilla), true, nil
		}
	}
	return overlayScaffold{}, false, nil
}

func (s *Service) overlayScaffoldHash(p *project.Project, file string) (string, bool, error) {
	scaffold, found, err := s.overlayScaffoldFor(p, file)
	if err != nil || !found {
		return "", false, err
	}
	return scaffold.Hash, true, nil
}

func pinVanillaScaffold(p *project.Project, scaffold overlayScaffold) error {
	blobs := p.Blobs()
	if len(scaffold.Hash) != 64 || strings.TrimSpace(blobs.ProjectUUID()) == "" {
		return errors.New("invalid GUI scaffold identity")
	}
	key := fmt.Sprintf("%s/p/%s/gui-scaffolds/%s.gui", blobs.EnvPrefix(), blobs.ProjectUUID(), scaffold.Hash)
	if !blobs.Exists(key) {
		if err := blobs.Put(key, "text/plain; charset=utf-8", int64(len(scaffold.Source)), bytes.NewReader(scaffold.Source)); err != nil {
			return err
		}
	}
	return p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		upsertAsset(tx, vanillaScaffoldAssetType, scaffold.Hash, key, nil)
		return nil
	})
}

func upsertAsset(tx *project.Tx, assetType, name, key string, provenance any) project.Row {
	for _, row := range tx.Rows("project_obs_assets") {
		if str(row["asset_type"]) == assetType && str(row["gfx_name"]) == name {
			updated, _ := tx.Update("project_obs_assets", project.Int64(row["id"]), func(r project.Row) {
				r["obs_key"] = key
				if provenance != nil {
					r["provenance"] = provenance
					r["created_at"] = nowTimestamp()
				}
			})
			return updated
		}
	}
	row := project.Row{"asset_type": assetType, "gfx_name": name, "obs_key": key}
	if provenance != nil {
		row["provenance"] = provenance
	}
	return tx.Insert("project_obs_assets", row)
}

func nowTimestamp() string { return time.Now().UTC().Format(timestampLayout) }

func (s *Service) overlayRead(w http.ResponseWriter, r *http.Request, p *project.Project) {
	file := strings.TrimSpace(r.URL.Query().Get("file"))
	normalized, err := exportengine.NormalizeGUISourcePath(file)
	if err != nil {
		detail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	overlay, exists, err := overlayForFile(p, file, normalized)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI overlay store unavailable")
		return
	}
	if !exists {
		scaffold, materializable, err := s.overlayScaffoldFor(p, file)
		if err != nil {
			detail(w, http.StatusServiceUnavailable, "GUI scaffold unavailable")
			return
		}
		source := "vanilla"
		if materializable {
			source = scaffold.Origin
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true, "project_id": p.ID(), "file": file, "source": source,
			"scaffold_hash": scaffold.Hash, "scaffold_hash_pinned": false,
			"materializable": materializable, "schema_version": 1,
			"edits": []map[string]any{}, "exists": false, "revision": "0",
		})
		return
	}
	currentHash, materializable, err := s.overlayScaffoldHash(p, overlay.File)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI scaffold unavailable")
		return
	}
	clientHash := overlay.ScaffoldHash
	if clientHash == "" && materializable {

		clientHash = currentHash
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "id": overlay.ID, "project_id": overlay.ProjectID,
		"file": overlay.File, "source": overlay.Source, "scaffold_hash": clientHash,
		"schema_version": overlay.SchemaVersion, "edits": overlay.Edits, "exists": true,
		"revision": overlay.Revision, "scaffold_hash_pinned": overlay.ScaffoldHash != "",
		"current_scaffold_hash": currentHash, "materializable": materializable,
		"scaffold_drift": materializable && overlay.ScaffoldHash != "" && overlay.ScaffoldHash != currentHash,
	})
}

func (s *Service) overlayWrite(w http.ResponseWriter, r *http.Request, p *project.Project) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	var in struct {
		File                 string           `json:"file"`
		Edits                []map[string]any `json:"edits"`
		ScaffoldHash         string           `json:"scaffold_hash"`
		ExpectedRevision     string           `json:"expected_revision"`
		ExpectedScaffoldHash string           `json:"expected_scaffold_hash"`
		Source               string           `json:"source"`
		SchemaVersion        int              `json:"schema_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		detail(w, http.StatusUnprocessableEntity, "invalid JSON body")
		return
	}
	in.File = strings.TrimSpace(in.File)
	if in.File == "" {
		detail(w, http.StatusBadRequest, "`file` is required")
		return
	}
	normalized, err := exportengine.NormalizeGUISourcePath(in.File)
	if err != nil {
		detail(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(in.Edits) > 10_000 {
		detail(w, http.StatusRequestEntityTooLarge, "too many GUI edits")
		return
	}
	if in.Edits == nil {
		in.Edits = []map[string]any{}
	}
	if in.SchemaVersion == 0 {
		in.SchemaVersion = 1
	}
	if strings.TrimSpace(in.ExpectedRevision) == "" {
		detail(w, http.StatusPreconditionRequired, "expected_revision is required; load the overlay before saving")
		return
	}
	if strings.TrimSpace(in.ExpectedScaffoldHash) == "" || strings.TrimSpace(in.ScaffoldHash) == "" {
		detail(w, http.StatusUnprocessableEntity, "scaffold_hash and expected_scaffold_hash from the loaded overlay are required")
		return
	}

	fileKey := strings.Trim(strings.ReplaceAll(in.File, "\\", "/"), "/")
	current, exists, err := overlayForFile(p, in.File, normalized)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI overlay store unavailable")
		return
	}
	if exists && strings.TrimSpace(current.File) != "" {
		fileKey = current.File
	}
	currentRevision, currentScaffold, storedScaffold := "0", "", ""
	if exists {
		currentRevision, currentScaffold, storedScaffold = current.Revision, current.ScaffoldHash, current.ScaffoldHash
	}
	scaffold, materializable, err := s.overlayScaffoldFor(p, fileKey)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI scaffold unavailable")
		return
	}
	if !materializable {
		detail(w, http.StatusConflict, "no imported GUI scaffold is available for this file")
		return
	}

	if !exists || currentScaffold == "" {
		currentScaffold = scaffold.Hash
	}
	if currentRevision != in.ExpectedRevision || currentScaffold != in.ExpectedScaffoldHash {
		writeJSON(w, http.StatusConflict, map[string]any{
			"detail":           "GUI overlay revision conflict; re-read before writing",
			"current_revision": currentRevision, "current_scaffold_hash": currentScaffold,
		})
		return
	}
	if in.ScaffoldHash != scaffold.Hash {
		detail(w, http.StatusConflict, "GUI scaffold changed; preview the overlay again")
		return
	}

	if index, found, err := s.loadIndex(r.Context(), p); err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI render index unavailable")
		return
	} else if found && index != nil {
		if winner, won := index.ProjectFileSource(normalized); won && winner != "gui_overlays" {
			detail(w, http.StatusConflict, "a project file at this path replaces this screen, so an overlay on it would never be applied; edit that file instead")
			return
		}
	}
	in.Source = scaffold.Origin
	if scaffold.Origin == scaffoldVanilla {
		if err := pinVanillaScaffold(p, scaffold); err != nil {
			detail(w, http.StatusServiceUnavailable, "could not keep the base-game GUI file this edit applies to; nothing was saved")
			return
		}
	}
	row, matched, err := saveOverlayIfCurrent(p, guiOverlay{
		File: fileKey, Edits: in.Edits, ScaffoldHash: in.ScaffoldHash, Source: in.Source, SchemaVersion: in.SchemaVersion,
	}, in.ExpectedRevision, storedScaffold)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI overlay store unavailable")
		return
	}
	if !matched {
		detail(w, http.StatusConflict, "GUI overlay revision conflict; re-read before writing")
		return
	}
	s.Forget(p)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "id": row.ID, "project_id": row.ProjectID, "file": row.File, "source": row.Source,
		"scaffold_hash": row.ScaffoldHash, "schema_version": row.SchemaVersion, "edits": row.Edits, "revision": row.Revision,
	})
}

func saveOverlayIfCurrent(p *project.Project, in guiOverlay, expectedRevision, expectedScaffold string) (guiOverlay, bool, error) {
	edits := make([]any, len(in.Edits))
	for i, edit := range in.Edits {
		edits[i] = edit
	}
	if in.Source == "" {
		in.Source = "vanilla"
	}
	var out guiOverlay
	matched := false
	err := p.Write([]string{"gui_overlays"}, func(tx *project.Tx) error {
		var existing project.Row
		for _, row := range tx.Rows("gui_overlays") {
			if str(row["file"]) == in.File {
				existing = row
				break
			}
		}
		if expectedRevision == "0" {
			if existing != nil {
				return nil
			}
			out = overlayFromRow(tx.Insert("gui_overlays", project.Row{
				"file": in.File, "source": in.Source, "scaffold_hash": in.ScaffoldHash,
				"edits": edits, "schema_version": int64(in.SchemaVersion),
			}))
			matched = true
			return nil
		}
		if existing == nil || overlayRevision(existing["updated_at"]) != expectedRevision ||
			str(existing["scaffold_hash"]) != expectedScaffold {
			return nil
		}
		id := project.Int64(existing["id"])
		for attempt := 0; attempt < 3; attempt++ {
			updated, err := tx.Update("gui_overlays", id, func(row project.Row) {
				row["source"], row["scaffold_hash"], row["edits"] = in.Source, in.ScaffoldHash, edits
				row["schema_version"] = int64(in.SchemaVersion)
			})
			if err != nil {
				return err
			}
			out = overlayFromRow(updated)

			if out.Revision != expectedRevision {
				break
			}
			time.Sleep(time.Millisecond)
		}
		matched = true
		return nil
	})
	return out, matched, err
}

type overlayCandidate struct {
	File                 string           `json:"file"`
	WindowName           string           `json:"window_name"`
	ExpectedRevision     string           `json:"expected_revision"`
	ExpectedScaffoldHash string           `json:"expected_scaffold_hash"`
	Edits                []map[string]any `json:"edits"`
}

func (s *Service) overlayPreview(w http.ResponseWriter, r *http.Request, p *project.Project) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	var in overlayCandidate
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		detail(w, http.StatusUnprocessableEntity, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.File) == "" || strings.TrimSpace(in.WindowName) == "" {
		detail(w, http.StatusUnprocessableEntity, "file and window_name are required")
		return
	}
	if strings.TrimSpace(in.ExpectedRevision) == "" {
		detail(w, http.StatusUnprocessableEntity, "expected_revision is required")
		return
	}
	if len(in.Edits) > 10_000 {
		detail(w, http.StatusRequestEntityTooLarge, "too many GUI edits")
		return
	}
	if in.Edits == nil {
		in.Edits = []map[string]any{}
	}
	payload, status, err := s.previewCandidate(r, p, in)
	if err != nil {
		detail(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Service) overlayValidate(w http.ResponseWriter, r *http.Request, p *project.Project) {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	var request struct {
		File             string `json:"file"`
		WindowName       string `json:"window_name"`
		ExpectedRevision string `json:"expected_revision"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		detail(w, http.StatusUnprocessableEntity, "invalid JSON body")
		return
	}
	normalized, err := exportengine.NormalizeGUISourcePath(request.File)
	if err != nil || strings.TrimSpace(request.WindowName) == "" {
		detail(w, http.StatusUnprocessableEntity, "file and window_name are required")
		return
	}
	overlay, exists, err := overlayForFile(p, request.File, normalized)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "GUI overlay store unavailable")
		return
	}
	if !exists {
		detail(w, http.StatusNotFound, "GUI overlay not found")
		return
	}
	revision := overlay.Revision
	if strings.TrimSpace(request.ExpectedRevision) != "" && request.ExpectedRevision != revision {
		detail(w, http.StatusConflict, "GUI overlay revision conflict; re-read before validating")
		return
	}
	expectedHash := overlay.ScaffoldHash
	if expectedHash == "" {
		currentHash, materializable, err := s.overlayScaffoldHash(p, normalized)
		if err != nil {
			detail(w, http.StatusServiceUnavailable, "GUI scaffold unavailable")
			return
		}
		if !materializable {
			detail(w, http.StatusConflict, "no imported GUI scaffold is available for this file")
			return
		}
		expectedHash = currentHash
	}
	payload, status, err := s.previewCandidate(r, p, overlayCandidate{
		File: request.File, WindowName: request.WindowName,
		ExpectedRevision: revision, ExpectedScaffoldHash: expectedHash, Edits: overlay.Edits,
	})
	if err != nil {
		writeJSON(w, status, map[string]any{
			"ok": false, "valid": false, "materializable": false, "detail": err.Error(), "revision": revision,
		})
		return
	}
	payload["revision"] = revision
	payload["valid"] = true
	payload["saved"] = true
	writeJSON(w, http.StatusOK, payload)
}

func (s *Service) previewCandidate(r *http.Request, p *project.Project, in overlayCandidate) (map[string]any, int, error) {
	normalized, err := exportengine.NormalizeGUISourcePath(in.File)
	if err != nil {
		return nil, http.StatusUnprocessableEntity, err
	}
	overlay, exists, err := overlayForFile(p, in.File, normalized)
	if err != nil {
		return nil, http.StatusServiceUnavailable, errors.New("GUI overlay store unavailable")
	}
	currentRevision, currentScaffold := "0", ""
	if exists {
		currentRevision, currentScaffold = overlay.Revision, overlay.ScaffoldHash
	}
	if in.ExpectedRevision != currentRevision {
		return nil, http.StatusConflict, errors.New("GUI overlay revision conflict; re-read before previewing")
	}
	scaffold, found, err := s.overlayScaffoldFor(p, normalized)
	if err != nil {
		return nil, http.StatusServiceUnavailable, errors.New("GUI source snapshot unavailable")
	}
	if !found {
		return nil, http.StatusConflict, errors.New("no GUI scaffold (imported or base game) is available for this file")
	}
	if !exists || currentScaffold == "" {
		currentScaffold = scaffold.Hash
	}
	if in.ExpectedScaffoldHash != currentScaffold {
		return nil, http.StatusConflict, errors.New("GUI overlay scaffold conflict; re-read before previewing")
	}
	if currentScaffold != "" && currentScaffold != scaffold.Hash {
		return nil, http.StatusConflict, errors.New("GUI scaffold changed; re-read and rebuild the overlay")
	}
	edited, err := exportengine.ApplyGUIPreviewEdits(scaffold.Source, in.Edits)
	if err != nil {
		return nil, http.StatusUnprocessableEntity, errors.New("GUI overlay could not be applied")
	}
	index, _, err := s.loadIndex(r.Context(), p)
	if err != nil || index == nil {
		return nil, http.StatusServiceUnavailable, errors.New("GUI render index unavailable")
	}
	window, found, err := index.WindowFromSource(strings.TrimSpace(in.WindowName), normalized, edited)
	if err != nil {
		return nil, http.StatusUnprocessableEntity, errors.New("GUI overlay result could not be parsed")
	}
	if !found {
		return nil, http.StatusUnprocessableEntity, errors.New("window_name is not present after applying the candidate overlay")
	}

	enrichCandidateLocalisations(window, in.Edits, "")
	file := normalized
	if exists && strings.TrimSpace(overlay.File) != "" {
		file = overlay.File
	}
	return map[string]any{
		"ok": true, "valid": true, "write_performed": false,
		"project_id": p.ID(), "file": file,
		"normalized_file": normalized, "window_name": strings.TrimSpace(in.WindowName),
		"current_revision": currentRevision, "current_scaffold_hash": currentScaffold,
		"scaffold_hash": scaffold.Hash, "materializable": true, "scaffold_source": scaffold.Origin,
		"edits": in.Edits, "window": window, "diagnostics": []map[string]any{},
	}, http.StatusOK, nil
}

func enrichCandidateLocalisations(window map[string]any, edits []map[string]any, lang string) {
	generated := exportengine.GUIOverlayPreviewLocalisationsIn(edits, overlayTextLanguage(lang))
	if len(generated) == 0 || window == nil {
		return
	}
	texts, _ := window["texts"].(map[string]any)
	for key, value := range generated {
		if _, referenced := texts[key]; referenced {
			texts[key] = map[string]any{"display": value, "kind": "loc"}
		}
	}
}

func overlayTextLanguage(raw string) string {
	code := strings.ToLower(strings.TrimSpace(raw))
	if cut := strings.IndexAny(code, "-_"); cut >= 0 {
		code = code[:cut]
	}
	if code == "" {
		return "en"
	}
	return code
}
