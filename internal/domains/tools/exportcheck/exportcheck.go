package exportcheck

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/hoi4/exportjob"
	"haganeworks/internal/hoi4/runcheck"
	"haganeworks/internal/project"
)

func Register(mux *http.ServeMux, deps domains.Deps) {
	setGame(deps.Game)
	h := &handler{open: deps.Open}
	for _, method := range []string{"GET", "POST"} {
		mux.HandleFunc(method+" /api/go/projects/{id}/export/check", h.check)
		mux.HandleFunc(method+" /api/go/projects/{id}/export/validate", h.validate)
	}
	mux.HandleFunc("GET /api/go/projects/{id}/export/base-installs", h.baseInstalls)
	mux.HandleFunc("GET /api/go/run-check/projects/{id}", h.runCheck)
}

type handler struct {
	open apiutil.Opener
}

func detail(w http.ResponseWriter, status int, message string) {
	apiutil.JSON(w, status, map[string]string{"detail": message})
}

func (h *handler) openProject(w http.ResponseWriter, r *http.Request) (*project.Project, bool) {
	p, err := h.open(r)
	if err != nil {
		var notFound apiutil.NotFoundError
		if errors.Is(err, project.ErrNotFound) || errors.As(err, &notFound) {
			detail(w, http.StatusNotFound, "Project not found")
		} else {
			apiutil.Fail(w, err)
		}
		return nil, false
	}
	return p, true
}

func parseReplacePaths(raw string) []string {
	out := []string{}
	for _, part := range strings.Split(raw, ",") {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func issueOptions(r *http.Request) ExportIssuesOptions {
	q := r.URL.Query()
	return ExportIssuesOptions{
		IncludeImported: q.Get("include_imported") == "true",
		ReplacePaths:    parseReplacePaths(q.Get("replace_paths")),
	}
}

func (h *handler) check(w http.ResponseWriter, r *http.Request) {
	p, ok := h.openProject(w, r)
	if !ok {
		return
	}
	s := newStore(p)
	issues, found, err := s.CollectExportIssues(r.Context(), p.ID(), issueOptions(r))
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "export gate unavailable")
		return
	}
	if !found {
		detail(w, http.StatusNotFound, "Project not found")
		return
	}
	apiutil.JSON(w, http.StatusOK, SummarizeExportIssues(issues))
}

func (h *handler) validate(w http.ResponseWriter, r *http.Request) {
	p, ok := h.openProject(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	opts := issueOptions(r)
	compact := q.Get("view") == "compact"
	var page validationPageRequest
	if compact {
		var pageErr *validationPageError
		page, pageErr = parseValidationPage(q)
		if pageErr != nil {
			apiutil.JSON(w, pageErr.status, map[string]any{"code": pageErr.code, "detail": pageErr.detail})
			return
		}
	}
	s := newStore(p)
	issues, found, err := s.CollectExportIssues(r.Context(), p.ID(), opts)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "export gate unavailable")
		return
	}
	if !found {
		detail(w, http.StatusNotFound, "Project not found")
		return
	}
	counts, nodeCount, err := s.ExportValidateSummary(r.Context(), p.ID())
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "export gate unavailable")
		return
	}

	summary := map[string]any{}
	for k, v := range counts {
		summary[k] = v
	}
	summary["focus_nodes"] = nodeCount
	if compact {
		payload, pageErr := compactValidationPage(issues, summary, page)
		if pageErr != nil {
			apiutil.JSON(w, pageErr.status, map[string]any{"code": pageErr.code, "detail": pageErr.detail})
			return
		}
		apiutil.JSON(w, http.StatusOK, payload)
		return
	}

	leveled := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		copied := map[string]any{}
		for k, v := range issue {
			copied[k] = v
		}
		copied["level"] = validationLevel(issue)
		leveled = append(leveled, copied)
	}

	blobs := p.Blobs()
	head := func(key string) (bool, error) { return blobs.Exists(key), nil }
	assetAudit, auditErr := s.AuditProjectAssetsFor(r.Context(), p.ID(), opts.IncludeImported, head)
	assetPayload := map[string]any{
		"idea_pictures_missing":  assetAudit.IdeaPicturesMissing,
		"event_pictures_missing": assetAudit.EventPicturesMissing,
		"decision_icons_missing": assetAudit.DecisionIconsMissing,
		"tech_icons_missing":     assetAudit.TechIconsMissing,
		"focus_icons_missing":    assetAudit.FocusIconsMissing,
		"portraits_missing":      assetAudit.PortraitsMissing,
		"total_missing":          assetAudit.TotalMissing,
	}
	if auditErr != nil {
		assetPayload["_error"] = auditErr.Error()
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"issues":      leveled,
		"summary":     summary,
		"asset_audit": assetPayload,

		"artifact": map[string]any{
			"status":  "deferred",
			"message": "Final artifact checks run when the ZIP is first built.",
		},
	})
}

func (h *handler) baseInstalls(w http.ResponseWriter, r *http.Request) {
	p, ok := h.openProject(w, r)
	if !ok {
		return
	}
	var values []any
	decode(colBytes(p.Record(), "dependencies"), &values)
	names := NormalizeBaseDependencies(values)
	if len(names) == 0 {
		apiutil.JSON(w, http.StatusOK, map[string]any{"base_installs": []BaseInstall{}})
		return
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"base_installs": ResolveBaseInstalls(names, []BasemodTemplate{}),
	})
}

func (h *handler) runCheck(w http.ResponseWriter, r *http.Request) {
	p, err := h.open(r)
	if err != nil {
		var notFound apiutil.NotFoundError
		if errors.Is(err, project.ErrNotFound) || errors.As(err, &notFound) {
			apiutil.JSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		} else {
			apiutil.JSON(w, http.StatusServiceUnavailable, map[string]any{"error": "project authorization unavailable"})
		}
		return
	}
	includeImported := strings.EqualFold(r.URL.Query().Get("include_imported"), "true")
	s := newStore(p)
	issues, err := collectRunCheck(s, includeImported)
	if err != nil {
		apiutil.JSON(w, http.StatusServiceUnavailable, map[string]any{"error": "run check unavailable"})
		return
	}
	politicsIssues, err := s.PoliticsLifecycleFindings(r.Context(), p.ID())
	if err != nil {
		apiutil.JSON(w, http.StatusServiceUnavailable, map[string]any{"error": "run check unavailable"})
		return
	}
	issues = append(issues, politicsIssues...)
	if issues == nil {
		issues = []runcheck.Finding{}
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{"issues": issues})
}

func collectRunCheck(s *Store, includeImported bool) ([]runcheck.Finding, error) {
	snapshot := runcheck.Snapshot{}
	namespaces, err := s.rows("event_namespaces")
	if err != nil {
		return nil, err
	}
	nsName := map[int64]string{}
	for _, ns := range namespaces {
		nsName[colInt(ns, "id")] = colStr(ns, "namespace")
	}
	events, err := s.rows("events")
	if err != nil {
		return nil, err
	}
	for _, row := range events {
		name, joined := nsName[colInt(row, "namespace_id")]
		if !joined {
			continue
		}
		if !includeImported && colStr(row, "source") == "imported" {
			continue
		}
		snapshot.Events = append(snapshot.Events, runcheck.Event{
			ID: colInt(row, "id"), NamespaceID: colInt(row, "namespace_id"), EventID: colStr(row, "event_id"),
			EventNumber: colInt(row, "event_number"), Namespace: name,
		})
	}
	trees, err := s.rows("focus_trees")
	if err != nil {
		return nil, err
	}
	treeKind := map[int64]string{}
	for _, t := range trees {
		if !includeImported && colStr(t, "source") == "imported" {
			continue
		}
		treeKind[colInt(t, "id")] = colStr(t, "tree_kind")
	}
	nodes, err := s.rows("focus_nodes")
	if err != nil {
		return nil, err
	}
	for _, row := range nodes {
		kind, joined := treeKind[colInt(row, "focus_tree_id")]
		if !joined {
			continue
		}
		node := runcheck.FocusNode{
			ID: colInt(row, "id"), TreeID: colInt(row, "focus_tree_id"), TreeKind: kind,
			FocusID: colStr(row, "focus_id"), RelativePosition: colStr(row, "relative_position_id"),
		}
		decode(colBytes(row, "prerequisites"), &node.Prerequisites)
		decode(colBytes(row, "mutually_exclusive"), &node.MutuallyExclusive)
		snapshot.FocusNodes = append(snapshot.FocusNodes, node)
	}
	return runcheck.Collect(snapshot), nil
}

func CollectExportIssues(ctx context.Context, p *project.Project, opts exportjob.ExportIssuesOptions) ([]exportjob.ExportIssue, bool, error) {
	issues, found, err := newStore(p).CollectExportIssues(ctx, p.ID(),
		ExportIssuesOptions{IncludeImported: opts.IncludeImported, ReplacePaths: opts.ReplacePaths})
	if err != nil || !found {
		return nil, found, err
	}
	out := make([]exportjob.ExportIssue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, exportjob.ExportIssue(issue))
	}
	return out, true, nil
}

func IssueCollector(p *project.Project) exportjob.IssueCollector { return collector{p} }

type collector struct{ p *project.Project }

func (c collector) CollectExportIssues(ctx context.Context, projectID int64, opts exportjob.ExportIssuesOptions) ([]exportjob.ExportIssue, bool, error) {
	if projectID != c.p.ID() {
		return nil, false, nil
	}
	return CollectExportIssues(ctx, c.p, opts)
}
