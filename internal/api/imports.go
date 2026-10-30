package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"haganeworks/internal/domains/countries"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/hoi4/importjob"
	"haganeworks/internal/hoi4/tagaudit"
	"haganeworks/internal/project"
	"haganeworks/internal/server"
	"haganeworks/internal/workbench"
)

const (
	maxStagingRequest = 1 << 30
	maxAuditFileBytes = 8 << 20
	maxAuditRequest   = 256 << 20
)

type imports struct {
	projects *projects
	jobs     *importjob.Manager
	staging  string
	locks    sync.Map
}

func registerImports(mux *http.ServeMux, p *projects) error {
	staging, err := os.MkdirTemp("", "haganeworks-import-")
	if err != nil {
		return err
	}
	jobs := importjob.NewManager()
	jobs.AfterImport = countries.SeedImportedPolitics
	i := &imports{projects: p, jobs: jobs, staging: staging}
	mux.HandleFunc("POST /api/go/import/staging/{pid}/{sid}", i.stage)
	mux.HandleFunc("DELETE /api/go/import/staging/{pid}/{sid}", i.dropStage)
	mux.HandleFunc("POST /api/go/import/jobs", i.startJob)
	mux.HandleFunc("GET /api/go/import/jobs/{pid}/{sid}", i.jobStatus)
	mux.HandleFunc("POST /api/go/import/jobs/{pid}/{sid}/cancel", i.cancelJob)
	mux.HandleFunc("POST /api/go/import/images/{pid}/{sid}", i.images)
	mux.HandleFunc("POST /api/go/projects/{id}/import/audit-tags", i.auditTags)
	return nil
}

func (i *imports) session(r *http.Request) (*project.Project, string, error) {
	pid, err := strconv.ParseInt(r.PathValue("pid"), 10, 64)
	sid := r.PathValue("sid")
	if err != nil || pid <= 0 || !importjob.ValidSessionID(sid) {
		return nil, "", project.ErrNotFound
	}
	store, err := i.projects.current()
	if err != nil {
		return nil, "", err
	}
	p, err := store.Get(pid)
	return p, sid, err
}

func (i *imports) sessionDir(projectID int64, sessionID string) string {
	return filepath.Join(i.staging, fmt.Sprintf("%d_%s", projectID, sessionID))
}

func safeStageTarget(root, relative string) (string, bool) {
	relative = strings.ReplaceAll(strings.TrimSpace(relative), "\\", "/")
	clean := filepath.Clean(filepath.FromSlash(relative))
	if relative == "" || strings.HasPrefix(relative, "/") || filepath.IsAbs(clean) || filepath.VolumeName(clean) != "" ||
		clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", false
	}
	return filepath.Join(root, clean), true
}

type stagedFile struct {
	relative string
	path     string
}

func readFiles(w http.ResponseWriter, r *http.Request, root string, limit int64) ([]stagedFile, []string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, nil, errors.New("multipart body required")
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, nil, err
	}
	received, err := os.MkdirTemp(filepath.Dir(root), filepath.Base(root)+".received-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(received)
	var paths []string
	count := 0
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, errors.New("invalid or oversized multipart upload")
		}
		switch part.FormName() {
		case "paths":
			value, err := io.ReadAll(io.LimitReader(part, 16<<10))
			part.Close()
			if err != nil {
				return nil, nil, errors.New("invalid path field")
			}
			paths = append(paths, string(value))
		case "files":
			err := writeStaged(filepath.Join(received, strconv.Itoa(count)), part)
			part.Close()
			if err != nil {
				return nil, nil, err
			}
			count++
		default:
			part.Close()
		}
	}
	if count != len(paths) {
		return nil, nil, errors.New("files and paths count mismatch")
	}
	var skipped []string
	var files []stagedFile
	for n, rel := range paths {
		target, safe := safeStageTarget(root, rel)
		if !safe {
			skipped = append(skipped, rel)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, skipped, err
		}
		if err := os.Rename(filepath.Join(received, strconv.Itoa(n)), target); err != nil {
			return nil, skipped, err
		}
		files = append(files, stagedFile{relative: rel, path: target})
	}
	return files, skipped, nil
}

func writeStaged(target string, body io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	tmp := target + ".part"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, body)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(tmp)
		return errors.New("cannot finish staged file")
	}
	return os.Rename(tmp, target)
}

func stagingStats(root string) (int, int64) {
	var count int
	var size int64
	_ = filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if info, err := d.Info(); err == nil {
				count++
				size += info.Size()
			}
		}
		return nil
	})
	return count, size
}

func (i *imports) lock(projectID int64, sessionID string) func() {
	value, _ := i.locks.LoadOrStore(fmt.Sprintf("%d:%s", projectID, sessionID), &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func (i *imports) stage(w http.ResponseWriter, r *http.Request) {
	p, sid, err := i.session(r)
	if err != nil {
		fail(w, err)
		return
	}
	defer i.lock(p.ID(), sid)()
	root := i.sessionDir(p.ID(), sid)
	files, skipped, err := readFiles(w, r, root, maxStagingRequest)
	if err != nil {
		server.Fail(w, http.StatusBadRequest, err)
		return
	}
	total, size := stagingStats(root)
	if skipped == nil {
		skipped = []string{}
	}
	server.JSON(w, http.StatusOK, map[string]any{"staged": len(files), "skipped": skipped, "total_staged": total, "total_bytes": size})
}

func (i *imports) dropStage(w http.ResponseWriter, r *http.Request) {
	p, sid, err := i.session(r)
	if err != nil {
		fail(w, err)
		return
	}
	defer i.lock(p.ID(), sid)()
	_ = os.RemoveAll(i.sessionDir(p.ID(), sid))
	server.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

type importJobRequest struct {
	ProjectID          int64  `json:"project_id"`
	SessionID          string `json:"session_id"`
	CountryTag         string `json:"country_tag"`
	ConflictPolicy     string `json:"conflict_policy"`
	ImportCharacters   *bool  `json:"import_characters"`
	ImportFocus        *bool  `json:"import_focus"`
	ImportTech         *bool  `json:"import_tech"`
	ImportIdeas        *bool  `json:"import_ideas"`
	ImportEvents       *bool  `json:"import_events"`
	ImportDecisions    *bool  `json:"import_decisions"`
	ImportBookmarks    *bool  `json:"import_bookmarks"`
	ImportHistoryFiles *bool  `json:"import_history_files"`
	ImportStates       *bool  `json:"import_states"`
}

func defaultTrue(value *bool) bool { return value == nil || *value }

func (i *imports) startJob(w http.ResponseWriter, r *http.Request) {
	var req importJobRequest
	if err := server.ReadJSON(r, 1<<20, &req); err != nil {
		server.Fail(w, http.StatusBadRequest, errors.New("invalid job request"))
		return
	}
	store, err := i.projects.current()
	if err != nil {
		fail(w, err)
		return
	}
	p, err := store.Get(req.ProjectID)
	if err != nil {
		fail(w, err)
		return
	}
	if !importjob.ValidSessionID(req.SessionID) {
		server.Fail(w, http.StatusBadRequest, errors.New("invalid project_id or session_id"))
		return
	}
	state, err := i.jobs.Start(p, req.SessionID, i.sessionDir(p.ID(), req.SessionID), importjob.Options{
		CountryTag: req.CountryTag, ConflictPolicy: req.ConflictPolicy,
		Characters: defaultTrue(req.ImportCharacters), Focus: defaultTrue(req.ImportFocus), Tech: defaultTrue(req.ImportTech),
		Ideas: defaultTrue(req.ImportIdeas), Events: defaultTrue(req.ImportEvents), Decisions: defaultTrue(req.ImportDecisions),
		Bookmarks: defaultTrue(req.ImportBookmarks), HistoryFiles: defaultTrue(req.ImportHistoryFiles), States: defaultTrue(req.ImportStates),
	})
	if err != nil {
		server.Fail(w, http.StatusBadRequest, err)
		return
	}
	server.JSON(w, http.StatusAccepted, state)
}

func (i *imports) jobStatus(w http.ResponseWriter, r *http.Request) {
	p, sid, err := i.session(r)
	if err != nil {
		fail(w, err)
		return
	}
	state, ok := i.jobs.Get(p.ID(), sid)
	if !ok {
		server.Fail(w, http.StatusNotFound, errors.New("not found"))
		return
	}
	server.JSON(w, http.StatusOK, state)
}

func (i *imports) cancelJob(w http.ResponseWriter, r *http.Request) {
	p, sid, err := i.session(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !i.jobs.Cancel(p.ID(), sid) {
		server.Fail(w, http.StatusNotFound, errors.New("not found"))
		return
	}
	server.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (i *imports) images(w http.ResponseWriter, r *http.Request) {
	p, sid, err := i.session(r)
	if err != nil {
		fail(w, err)
		return
	}
	gfx, ok := i.jobs.AssetContext(p.ID(), sid)
	if !ok {
		server.Fail(w, http.StatusConflict, errors.New("completed import job with GFX context required"))
		return
	}
	plan, uuid, err := importassets.BuildPlan(p, gfx)
	if err != nil {
		fail(w, err)
		return
	}
	batchRoot, err := os.MkdirTemp(i.staging, fmt.Sprintf("images_%d_", p.ID()))
	if err != nil {
		fail(w, err)
		return
	}
	defer os.RemoveAll(batchRoot)
	files, skipped, err := readFiles(w, r, batchRoot, maxStagingRequest)
	if err != nil {
		server.Fail(w, http.StatusBadRequest, err)
		return
	}
	relatives := make([]string, len(files))
	for index, file := range files {
		relatives[index] = file.relative
	}
	assigned := importassets.AssignTargets(plan, relatives)
	blobs := p.Blobs()
	warnings := append([]string{}, skipped...)
	counts := map[string]int{}
	var mappings []importassets.SavedMapping
	rawSaved := 0
	for index, file := range files {
		raw, err := os.ReadFile(file.path)
		if err != nil {
			warnings = append(warnings, file.relative+": "+err.Error())
			continue
		}
		if key, err := importassets.RawKey(blobs.EnvPrefix(), uuid, filepath.ToSlash(file.relative)); err == nil {
			if err := blobs.Put(key, "application/octet-stream", int64(len(raw)), bytes.NewReader(raw)); err == nil {
				rawSaved++
			} else {
				warnings = append(warnings, file.relative+" raw: "+err.Error())
			}
		}
		for _, target := range assigned[index] {
			png, err := importassets.NormalizePNG(raw, file.relative, canonicalKind(target.Slot))
			if err != nil {
				warnings = append(warnings, file.relative+" -> "+target.Slot+": "+err.Error())
				continue
			}
			key, err := importassets.UploadKey(blobs.EnvPrefix(), uuid, target.Kind, target.OutputName)
			if err != nil {
				warnings = append(warnings, err.Error())
				continue
			}
			if err := blobs.Put(key, "image/png", int64(len(png)), bytes.NewReader(png)); err != nil {
				warnings = append(warnings, file.relative+": "+err.Error())
				continue
			}
			mappings = append(mappings, importassets.SavedMapping{Target: target, OBSKey: key})
			counts[resultCountKey(target.Slot)]++
		}
	}
	err = p.Write([]string{"project_obs_icons", "project_obs_assets", "entity_obs_paths", "technologies", "characters"}, func(tx *project.Tx) error {
		importassets.PersistMappings(tx, mappings)
		return nil
	})
	if err != nil {
		fail(w, err)
		return
	}
	result := map[string]any{
		"received": len(files), "raw_saved": rawSaved, "raw_obs_saved": rawSaved, "warnings": warnings,
		"art_obs_persisted": len(mappings), "portraits_matched": counts["portraits_matched"],
		"goal_icons_saved": counts["goal_icons_saved"],
	}
	for key, value := range counts {
		result[key] = value
	}
	server.JSON(w, http.StatusOK, result)
}

func canonicalKind(slot string) string {
	if slot == "tech_icon" || slot == "event_picture" {
		return ""
	}
	return slot
}

func resultCountKey(slot string) string {
	switch slot {
	case "focus_icon":
		return "goal_icons_saved"
	case "idea_picture":
		return "idea_pictures_saved"
	case "event_picture":
		return "event_pictures_saved"
	case "decision_icon":
		return "decision_icons_saved"
	case "decision_category_icon":
		return "decision_category_icons_saved"
	case "bookmark_picture":
		return "bookmark_pictures_saved"
	case "tech_icon":
		return "tech_icons_saved"
	case "character_portrait", "character_advisor":
		return "portraits_matched"
	}
	return "images_saved"
}

func (i *imports) auditTags(w http.ResponseWriter, r *http.Request) {
	p, err := i.projects.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	root, err := os.MkdirTemp(i.staging, fmt.Sprintf("audit_%d_", p.ID()))
	if err != nil {
		fail(w, err)
		return
	}
	defer os.RemoveAll(root)
	staged, _, err := readFiles(w, r, root, maxAuditRequest)
	if err != nil {
		server.Fail(w, http.StatusBadRequest, err)
		return
	}
	files := make([]tagaudit.File, 0, len(staged))
	for _, file := range staged {
		body, err := os.ReadFile(file.path)
		if err != nil || len(body) > maxAuditFileBytes {
			continue
		}
		files = append(files, tagaudit.File{Path: file.relative, Body: body})
	}
	existing, err := workbench.ExistingCountryTags(p)
	if err != nil {
		fail(w, err)
		return
	}
	server.JSON(w, http.StatusOK, tagaudit.Audit(files, existing))
}
