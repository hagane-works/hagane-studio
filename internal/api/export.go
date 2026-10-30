package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"haganeworks/internal/domains/tools/exportcheck"
	"haganeworks/internal/export"
	"haganeworks/internal/gamedir"
	"haganeworks/internal/server"
)

type exports struct {
	projects *projects
	jobs     *export.Jobs
}

func registerExports(mux *http.ServeMux, p *projects, jobs *export.Jobs) {
	e := &exports{projects: p, jobs: jobs}
	mux.HandleFunc("POST /api/go/export/jobs", e.start)
	mux.HandleFunc("GET /api/go/export/jobs/{pid}/{job}", e.status)
	mux.HandleFunc("GET /api/go/export/jobs/{pid}/{job}/download", e.download)
	mux.HandleFunc("POST /api/app/projects/{id}/install", e.install)
	registerAdvancedExports(mux, p, exportcheck.IssueCollector)
}

func jobStatus(job export.Job) map[string]any {
	out := map[string]any{"job_id": job.ID, "status": job.Status}
	switch job.Status {
	case "complete":
		out["files"] = job.Result.Files
		out["filename"] = job.Result.Filename
		out["size_bytes"] = job.Result.SizeBytes
		out["native_warnings"] = job.Result.NativeWarnings
	case "failed":
		out["error"] = job.Error
		out["http_status"] = http.StatusInternalServerError
	}
	return out
}

func (e *exports) start(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID int64 `json:"project_id"`
		export.Options
	}
	if err := server.ReadJSON(r, maxJSONBody, &body); err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	store, err := e.projects.current()
	if err != nil {
		fail(w, err)
		return
	}
	p, err := store.Get(body.ProjectID)
	if err != nil {
		fail(w, err)
		return
	}
	body.Options.Issues = exportcheck.IssueCollector
	job := e.jobs.Start(p, body.Options)
	server.JSON(w, http.StatusAccepted, jobStatus(job))
}

func (e *exports) find(r *http.Request) (export.Job, bool) {
	pid, err := strconv.ParseInt(r.PathValue("pid"), 10, 64)
	if err != nil {
		return export.Job{}, false
	}
	return e.jobs.Get(pid, r.PathValue("job"))
}

func (e *exports) status(w http.ResponseWriter, r *http.Request) {
	job, ok := e.find(r)
	if !ok {
		server.Fail(w, http.StatusNotFound, errors.New("export job not found"))
		return
	}
	server.JSON(w, http.StatusOK, jobStatus(job))
}

func (e *exports) download(w http.ResponseWriter, r *http.Request) {
	job, ok := e.find(r)
	if !ok || job.Status != "complete" {
		server.Fail(w, http.StatusNotFound, errors.New("export not ready"))
		return
	}
	file, err := os.Open(job.Path)
	if err != nil {
		fail(w, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", urlPathEscape(job.Result.Filename)))
	http.ServeContent(w, r, job.Result.Filename, info.ModTime(), file)
}

func (e *exports) install(w http.ResponseWriter, r *http.Request) {
	p, err := e.projects.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	var opts export.Options
	if err := server.ReadJSON(r, maxJSONBody, &opts); err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	modDir := e.projects.settings.Get().ModDir
	if modDir == "" {
		modDir = gamedir.ModDir()
	}
	tmp, err := os.MkdirTemp("", "haganeworks-install-")
	if err != nil {
		fail(w, err)
		return
	}
	defer os.RemoveAll(tmp)
	zipPath := filepath.Join(tmp, "mod.zip")
	opts.AcknowledgeBlockers = true
	opts.Issues = exportcheck.IssueCollector
	if _, err := export.Build(context.Background(), p, opts, zipPath); err != nil {
		fail(w, err)
		return
	}
	uuid, _ := p.Record()["uuid"].(string)
	if uuid == "" {
		uuid = fmt.Sprintf("project-%d", p.ID())
	}
	result, err := export.Install(zipPath, modDir, uuid)
	if errors.Is(err, export.ErrFolderTaken) {
		server.Fail(w, http.StatusConflict, err)
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	server.JSON(w, http.StatusOK, result)
}

func urlPathEscape(name string) string {
	const hex = "0123456789ABCDEF"
	out := make([]byte, 0, len(name)*3)
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-' || c == '_' {
			out = append(out, c)
			continue
		}
		out = append(out, '%', hex[c>>4], hex[c&15])
	}
	return string(out)
}
