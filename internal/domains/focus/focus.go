package focus

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

func Register(mux *http.ServeMux, deps domains.Deps) {
	h := &handlers{deps: deps}
	const tree = "/api/go/projects/{id}/focus-trees"
	mux.HandleFunc("GET "+tree, h.read(h.listTrees))
	mux.HandleFunc("POST "+tree, h.write(h.createTree))
	mux.HandleFunc("GET "+tree+"/{tid}", h.read(h.getTree))
	mux.HandleFunc("PUT "+tree+"/{tid}", h.write(h.updateTree))
	mux.HandleFunc("DELETE "+tree+"/{tid}", h.write(h.deleteTree))
	mux.HandleFunc("GET "+tree+"/{tid}/preview", h.read(h.preview))
	mux.HandleFunc("PUT "+tree+"/{tid}/node-positions", h.write(h.nodePositions))
	mux.HandleFunc("GET "+tree+"/{tid}/nodes", h.read(h.listNodes))
	mux.HandleFunc("POST "+tree+"/{tid}/nodes", h.write(h.createNode))
	mux.HandleFunc("GET "+tree+"/{tid}/nodes/{nid}", h.read(h.getNode))
	mux.HandleFunc("PUT "+tree+"/{tid}/nodes/{nid}", h.write(h.updateNode))
	mux.HandleFunc("DELETE "+tree+"/{tid}/nodes/{nid}", h.write(h.deleteNode))
	mux.HandleFunc("PUT "+tree+"/{tid}/nodes/{nid}/mutual-exclusion", h.write(h.mutualExclusion))
	mux.HandleFunc("POST "+tree+"/{tid}/nodes/{nid}/fork", h.write(h.forkNode))
	mux.HandleFunc("POST "+tree+"/{tid}/nodes/{nid}/parse-preview", h.read(h.parsePreview))
	mux.HandleFunc("POST "+tree+"/{tid}/nodes/{nid}/upload-icon", h.write(h.uploadNodeIcon))

	const icons = "/api/go/projects/{id}/focus-icons"
	mux.HandleFunc("GET "+icons, h.read(h.listIcons))
	mux.HandleFunc("GET "+icons+"/library", h.read(h.iconLibrary))
	mux.HandleFunc("POST "+icons+"/upload-named", h.write(h.uploadNamedIcon))
	mux.HandleFunc("GET "+icons+"/{gfx}", h.read(h.serveIcon))
	mux.HandleFunc("GET /api/go/vanilla/focus-icon/{gfx}", h.vanillaIcon)

	mux.HandleFunc("POST /api/go/projects/{id}/import/focus-tree", h.write(h.importFocusTree))
	mux.HandleFunc("POST /api/go/projects/{id}/import/drawio", h.write(h.importDrawIO))

	mux.HandleFunc("GET /api/go/lint/dead-flags/{id}", h.read(h.deadFlags))
	mux.HandleFunc("GET /api/go/lint/tree-validation/{id}", h.read(h.treeValidation))
}

type handlers struct {
	deps domains.Deps
}

func fail(w http.ResponseWriter, err error) {
	var detail DetailError
	if errors.As(err, &detail) {
		apiutil.JSON(w, detail.Status, map[string]any{"detail": detail.Detail})
		return
	}
	apiutil.Fail(w, err)
}

func (h *handlers) read(fn func(p *project.Project, w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := h.deps.Open(r)
		if err != nil {
			fail(w, err)
			return
		}
		if err := fn(p, w, r); err != nil {
			fail(w, err)
		}
	}
}

func (h *handlers) write(fn func(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rv, err := readRevisions(r)
		if err != nil {
			fail(w, err)
			return
		}
		p, err := h.deps.Open(r)
		if err != nil {
			fail(w, err)
			return
		}
		if err := fn(p, rv, w, r); err != nil {
			fail(w, err)
		}
	}
}

func respond(w http.ResponseWriter, rv *revisions, status int, body any) {
	rv.header(w)
	if body == nil {
		w.WriteHeader(status)
		return
	}
	apiutil.JSON(w, status, body)
}

func decodeBody(r *http.Request, v any, detail string) error {
	raw, err := apiutil.ReadBody(r)
	if err != nil {
		return DetailError{Status: 422, Detail: detail}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(v); err != nil {
		return DetailError{Status: 422, Detail: detail}
	}
	return nil
}

func pathTreeID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("tid"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errTreeMissing
	}
	return id, nil
}

func pathNodeID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("nid"), 10, 64)
	if err != nil || id <= 0 || id > maxPersistedFocusNodeID {
		return 0, errNodeNotPersisted
	}
	return id, nil
}

func (h *handlers) listTrees(p *project.Project, w http.ResponseWriter, r *http.Request) error {

	if scriptID := strings.TrimSpace(r.URL.Query().Get("tree_id")); scriptID != "" {
		tree, err := treeByScriptID(p, scriptID)
		if err != nil {
			return err
		}
		if tree == nil {
			apiutil.JSON(w, http.StatusOK, []FocusTree{})
			return nil
		}
		apiutil.JSON(w, http.StatusOK, []FocusTree{*tree})
		return nil
	}
	trees, err := listTrees(p, r.URL.Query().Get("country_tag"), r.URL.Query().Get("tree_kind"))
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, trees)
	return nil
}

func parseInitialShowPosition(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, DetailError{Status: 422, Detail: "initial_show_position must be an object"}
	}
	cleaned, err := ValidateInitialShowPositionUnion(m)
	if err != nil {
		return nil, DetailError{Status: 422, Detail: err.Error()}
	}
	return cleaned, nil
}

func (h *handlers) createTree(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	var in FocusTreeCreateInput
	if err := decodeBody(r, &in, "Invalid focus tree payload"); err != nil {
		return err
	}
	position, err := parseInitialShowPosition(in.RawInitialShowPosition)
	if err != nil {
		return err
	}
	in.InitialShowPosition = position
	tree, err := createTree(p, rv, in)
	if err != nil {
		return err
	}
	respond(w, rv, http.StatusCreated, tree)
	return nil
}

func (h *handlers) getTree(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	tree, err := getTree(p, treeID)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, tree)
	return nil
}

func (h *handlers) updateTree(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	var patch FocusTreePatch
	if err := decodeBody(r, &patch, "Invalid focus tree payload"); err != nil {
		return err
	}
	if patch.InitialShowPosition.Set && patch.InitialShowPosition.Value != nil {
		cleaned, err := ValidateInitialShowPositionUnion(*patch.InitialShowPosition.Value)
		if err != nil {
			return DetailError{Status: 422, Detail: err.Error()}
		}
		patch.InitialShowPosition.Value = &cleaned
	}
	tree, err := updateTree(p, rv, treeID, patch)
	if err != nil {
		return err
	}
	respond(w, rv, http.StatusOK, tree)
	return nil
}

func (h *handlers) deleteTree(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	if err := deleteTree(p, rv, treeID); err != nil {
		return err
	}
	respond(w, rv, http.StatusNoContent, nil)
	return nil
}

func (h *handlers) preview(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	return previewTree(p, w, r, treeID)
}

func (h *handlers) nodePositions(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	var payload struct {
		Positions []FocusNodePosition `json:"positions"`
	}
	if err := decodeBody(r, &payload, "Invalid node positions payload"); err != nil {
		return err
	}
	for _, pos := range payload.Positions {
		if pos.ID <= 0 || pos.ID > maxPersistedFocusNodeID {
			return DetailError{Status: 422, Detail: "positions entries require 0 < id <= 2147483647"}
		}
	}
	nodes, err := bulkPositions(p, rv, treeID, payload.Positions)
	if err != nil {
		return err
	}
	respond(w, rv, http.StatusOK, nodes)
	return nil
}

func (h *handlers) listNodes(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	tree, err := getTree(p, treeID)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, tree.Nodes)
	return nil
}

func (h *handlers) createNode(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	var in FocusNodeCreateInput
	if err := decodeBody(r, &in, "Invalid focus node payload"); err != nil {
		return err
	}
	node, err := createNode(p, rv, treeID, in)
	if err != nil {
		return err
	}
	respond(w, rv, http.StatusCreated, node)
	return nil
}

func (h *handlers) getNode(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	nodeID, err := pathNodeID(r)
	if err != nil {
		return err
	}
	node, err := getNode(p, treeID, nodeID)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, node)
	return nil
}

func (h *handlers) updateNode(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	nodeID, err := pathNodeID(r)
	if err != nil {
		return err
	}
	var patch FocusNodePatch
	if err := decodeBody(r, &patch, "Invalid focus node payload"); err != nil {
		return err
	}
	node, err := updateNode(p, rv, treeID, nodeID, patch)
	if err != nil {
		return err
	}
	respond(w, rv, http.StatusOK, node)
	return nil
}

func (h *handlers) deleteNode(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	nodeID, err := pathNodeID(r)
	if err != nil {
		return err
	}
	if err := deleteNode(p, rv, treeID, nodeID); err != nil {
		return err
	}
	respond(w, rv, http.StatusNoContent, nil)
	return nil
}

func (h *handlers) mutualExclusion(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	nodeID, err := pathNodeID(r)
	if err != nil {
		return err
	}
	var payload struct {
		OtherNodeID int64 `json:"other_node_id"`
		Enabled     *bool `json:"enabled"`
	}
	if err := decodeBody(r, &payload, ""); err != nil || payload.OtherNodeID <= 0 || payload.Enabled == nil {
		return DetailError{Status: 422, Detail: "other_node_id and enabled are required"}
	}
	nodes, err := setMutualExclusion(p, rv, treeID, nodeID, payload.OtherNodeID, *payload.Enabled)
	if err != nil {
		return err
	}
	respond(w, rv, http.StatusOK, nodes)
	return nil
}

func (h *handlers) forkNode(p *project.Project, rv *revisions, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	nodeID, err := pathNodeID(r)
	if err != nil {
		return err
	}
	var payload struct {
		NewFocusID string `json:"new_focus_id"`
	}
	if err := decodeBody(r, &payload, ""); err != nil || payload.NewFocusID == "" {
		return DetailError{Status: 422, Detail: "new_focus_id is required"}
	}
	node, err := forkNode(p, rv, treeID, nodeID, payload.NewFocusID)
	if err != nil {
		return err
	}
	respond(w, rv, http.StatusCreated, node)
	return nil
}

func (h *handlers) parsePreview(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	treeID, err := pathTreeID(r)
	if err != nil {
		return err
	}
	nodeID, err := pathNodeID(r)
	if err != nil {
		return err
	}
	result, err := parsePreviewNode(p, r, treeID, nodeID)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}
