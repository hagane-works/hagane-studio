package tools

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

var (
	basemodSlugSepRe       = regexp.MustCompile(`[^a-z0-9]+`)
	basemodFlagTagRe       = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)
	basemodFlagIdeologyRe  = regexp.MustCompile(`^[A-Za-z0-9_]{1,32}$`)
	basemodIndexEntityType = map[string]bool{
		"character": true, "focus": true, "decision": true, "country_tag": true,
		"idea": true, "ideology": true, "state": true,
	}
	basemodCollisionTypes = map[string]bool{"character": true, "focus": true, "idea": true, "decision": true}

	basemodFlagSizeDirs = map[string]string{
		"": "", "large": "", "standard": "", "normal": "", "l": "",
		"medium": "medium/", "med": "medium/", "m": "medium/",
		"small": "small/", "sml": "small/", "s": "small/",
	}
)

const (
	basemodMaxIDLen    = 256
	basemodMaxBatchIDs = 5000
	basemodMaxBody     = 1 << 20
)

func registerBasemod(mux *http.ServeMux, deps domains.Deps) {
	b := &basemod{open: deps.Open, game: deps.Game, mods: &basemodMods{}}
	mux.HandleFunc("GET /api/go/basemod/templates", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"templates": []any{}, "items": []any{}, "total": 0})
	})
	mux.HandleFunc("GET /api/go/basemod/refs", b.refs)
	mux.HandleFunc("GET /api/go/basemod/ideology-registry", b.ideologyRegistry)
	mux.HandleFunc("GET /api/go/basemod/id-taken", b.idTakenGET)
	mux.HandleFunc("POST /api/go/basemod/id-taken", b.idTakenPOST)
	mux.HandleFunc("GET /api/go/projects/{id}/basemod/{dep}/gfx/{name}", b.gfx)
	mux.HandleFunc("GET /api/go/projects/{id}/basemod/{dep}/art/{rel...}", b.art)
	mux.HandleFunc("GET /api/go/projects/{id}/basemod/{dep}/flag/{tag}", b.flag)
}

type basemod struct {
	open apiutil.Opener
	game func() (fs.FS, error)
	mods *basemodMods
}

func (b *basemod) project(w http.ResponseWriter, r *http.Request, projectID int64) *project.Project {
	r.SetPathValue("id", strconv.FormatInt(projectID, 10))
	p, err := b.open(r)
	if err != nil {
		apiutil.Fail(w, err)
		return nil
	}
	return p
}

func basemodDependencies(p *project.Project) []string {
	out := []string{}
	for _, name := range workbench.StringList(p.Record()["dependencies"]) {
		if text := strings.TrimSpace(name); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func basemodKey(name string) string {
	return strings.Trim(basemodSlugSepRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "_"), "_")
}

func (b *basemod) refs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	projectID, err := strconv.ParseInt(strings.TrimSpace(q.Get("project_id")), 10, 64)
	if err != nil || projectID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid project_id"})
		return
	}
	entityType := q.Get("type")
	if !basemodIndexEntityType[entityType] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "unknown entity type"})
		return
	}
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		if limit, err := strconv.Atoi(raw); err != nil || limit < 1 || limit > 200 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "limit must be an integer in [1, 200]"})
			return
		}
	}
	if b.project(w, r, projectID) == nil {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"type": entityType, "results": []any{}})
}

func (b *basemod) ideologyRegistry(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("project_id")), 10, 64)
	if err != nil || projectID <= 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if b.project(w, r, projectID) == nil {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": []any{}, "sub_ideologies": []any{}})
}

func (b *basemod) idTakenGET(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	projectID, err := strconv.ParseInt(strings.TrimSpace(q.Get("project_id")), 10, 64)
	if err != nil || projectID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad project_id"})
		return
	}
	if !basemodCollisionTypes[q.Get("type")] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown type"})
		return
	}
	if id := strings.TrimSpace(q.Get("id")); id == "" || len(id) > basemodMaxIDLen {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	if b.project(w, r, projectID) == nil {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"indexed": false, "taken": false})
}

func (b *basemod) idTakenPOST(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ProjectID int64    `json:"project_id"`
		Type      string   `json:"type"`
		IDs       []string `json:"ids"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, basemodMaxBody)).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad JSON body: " + err.Error()})
		return
	}
	if request.ProjectID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad project_id"})
		return
	}
	if !basemodCollisionTypes[request.Type] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown type"})
		return
	}
	if len(request.IDs) > basemodMaxBatchIDs {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "too many ids"})
		return
	}
	if b.project(w, r, request.ProjectID) == nil {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"indexed": false, "taken": map[string]string{}})
}
