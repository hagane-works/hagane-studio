package focus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/domains"
	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/imgcodec"
	"haganeworks/internal/project"
)

type harness struct {
	t   *testing.T
	p   *project.Project
	mux *http.ServeMux
}

func newHarness(t *testing.T, game fs.FS) *harness {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) {
			if game == nil {
				return nil, fs.ErrNotExist
			}
			return game, nil
		},
	})
	return &harness{t: t, p: p, mux: mux}
}

func (h *harness) do(method, path string, body any, headers ...string) *httptest.ResponseRecorder {
	h.t.Helper()
	var reader *bytes.Reader
	switch b := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(b))
	default:
		raw, _ := json.Marshal(b)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	return rec
}

func (h *harness) expect(rec *httptest.ResponseRecorder, status int) map[string]any {
	h.t.Helper()
	if rec.Code != status {
		h.t.Fatalf("status %d, want %d: %s", rec.Code, status, rec.Body.String())
	}
	if rec.Body.Len() == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		return nil
	}
	return out
}

func (h *harness) list(rec *httptest.ResponseRecorder, status int) []any {
	h.t.Helper()
	if rec.Code != status {
		h.t.Fatalf("status %d, want %d: %s", rec.Code, status, rec.Body.String())
	}
	var out []any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		h.t.Fatalf("decode list: %v: %s", err, rec.Body.String())
	}
	return out
}

const base = "/api/go/projects/1/focus-trees"

func (h *harness) createTree(body any) int64 {
	h.t.Helper()
	tree := h.expect(h.do("POST", base, body), http.StatusCreated)
	return int64(tree["id"].(float64))
}

func (h *harness) createNode(treeID int64, body any) map[string]any {
	h.t.Helper()
	return h.expect(h.do("POST", fmt.Sprintf("%s/%d/nodes", base, treeID), body), http.StatusCreated)
}

func diagnosticCodes(v any) []string {
	var out []string
	list, _ := v.([]any)
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m["code"].(string))
		}
	}
	return out
}

func TestTreeLifecycle(t *testing.T) {
	h := newHarness(t, nil)
	rec := h.do("POST", base, map[string]any{})
	tree := h.expect(rec, http.StatusCreated)
	if tree["tree_id"] != "tst_custom_focus" || tree["edit_revision"] != "1" || tree["country_tag"] != "TST" || tree["scope_kind"] != "own" {
		t.Fatalf("created tree: %v", tree)
	}
	if got := rec.Header().Get("X-Edit-Revisions"); got != `{"focus_tree:1":"1"}` {
		t.Fatalf("X-Edit-Revisions = %q", got)
	}
	policy, _ := tree["file_policy"].(map[string]any)
	if policy == nil || policy["emit_mode"] != "namespaced_addition" || policy["source_file"] != "common/national_focus/tst_custom_focus.txt" || policy["origin_kind"] != "created" {
		t.Fatalf("file policy: %v", tree["file_policy"])
	}
	if tree["diagnostics"] == nil || tree["initial_show_position"] != nil || tree["shared_country_tags"] != nil {
		t.Fatalf("tree shape: %v", tree)
	}
	if nodes, _ := tree["nodes"].([]any); nodes == nil || len(nodes) != 0 {
		t.Fatalf("nodes: %v", tree["nodes"])
	}

	if trees := h.list(h.do("GET", base, nil), http.StatusOK); len(trees) != 1 {
		t.Fatalf("list: %v", trees)
	}
	if trees := h.list(h.do("GET", base+"?tree_id=tst_custom_focus", nil), http.StatusOK); len(trees) != 1 {
		t.Fatalf("by tree_id: %v", trees)
	}
	if trees := h.list(h.do("GET", base+"?tree_id=nope", nil), http.StatusOK); len(trees) != 0 {
		t.Fatalf("missing tree_id: %v", trees)
	}
	if trees := h.list(h.do("GET", base+"?country_tag=GER", nil), http.StatusOK); len(trees) != 0 {
		t.Fatalf("country filter: %v", trees)
	}

	rec = h.do("PUT", base+"/1", map[string]any{"continuous_focus_position_x": 7, "edit_revision": "1"},
		"X-Edit-Revisions", `{"focus_tree:1":"1"}`)
	updated := h.expect(rec, http.StatusOK)
	if updated["edit_revision"] != "2" || updated["continuous_focus_position_x"].(float64) != 7 {
		t.Fatalf("updated: %v", updated)
	}
	if got := rec.Header().Get("X-Edit-Revisions"); got != `{"focus_tree:1":"2"}` {
		t.Fatalf("X-Edit-Revisions = %q", got)
	}

	conflict := h.expect(h.do("PUT", base+"/1", map[string]any{"continuous_focus_position_x": 8},
		"X-Edit-Revisions", `{"focus_tree:1":"1"}`), http.StatusConflict)
	detail := conflict["detail"].(map[string]any)
	if detail["code"] != "save_conflict" || detail["expected_revision"] != "1" || detail["current_revision"] != "2" {
		t.Fatalf("conflict: %v", conflict)
	}

	h.expect(h.do("PUT", base+"/1", map[string]any{}, "X-Edit-Revisions", `{"focus_tree:1":"01"}`), http.StatusUnprocessableEntity)

	readOnly := h.expect(h.do("PUT", base+"/1", map[string]any{"tree_id": "other"}), http.StatusUnprocessableEntity)
	if readOnly["detail"].(map[string]any)["error"] != "focus_tree_id_read_only" {
		t.Fatalf("read only: %v", readOnly)
	}

	h.expect(h.do("DELETE", base+"/1", nil), http.StatusNoContent)
	missing := h.expect(h.do("GET", base+"/1", nil), http.StatusNotFound)
	if missing["detail"] != "Focus tree not found" {
		t.Fatalf("missing: %v", missing)
	}
	if rows, _ := h.p.Rows("focus_file_policies"); len(rows) != 0 {
		t.Fatalf("policies left: %v", rows)
	}
}

func TestCreateTreeRules(t *testing.T) {
	h := newHarness(t, nil)
	isDefault := h.expect(h.do("POST", base, map[string]any{"is_default": true}), http.StatusUnprocessableEntity)
	if isDefault["detail"].(map[string]any)["error"] != "focus_created_default_conflicts_with_effective_base" {
		t.Fatalf("is_default: %v", isDefault)
	}
	imported := h.expect(h.do("POST", base, map[string]any{"source": "imported"}), http.StatusUnprocessableEntity)
	if imported["detail"].(map[string]any)["error"] != "focus_origin_metadata_read_only" {
		t.Fatalf("source: %v", imported)
	}
	foreign := h.expect(h.do("POST", base, map[string]any{"country_tag": "GER"}), http.StatusUnprocessableEntity)
	if foreign["detail"].(map[string]any)["error"] != "focus_scope_blocked" {
		t.Fatalf("foreign country: %v", foreign)
	}
	h.expect(h.do("POST", base, "not json"), http.StatusUnprocessableEntity)

	first := h.expect(h.do("POST", base, map[string]any{"tree_id": "mine"}), http.StatusCreated)
	second := h.expect(h.do("POST", base, map[string]any{"tree_id": "mine"}), http.StatusCreated)
	if first["tree_id"] != "mine" || second["tree_id"] != "mine_2" {
		t.Fatalf("suffix: %v %v", first["tree_id"], second["tree_id"])
	}
	if w := second["warnings"].([]any); len(w) != 1 || !strings.Contains(w[0].(string), "auto-suffixed to 'mine_2'") {
		t.Fatalf("warnings: %v", second["warnings"])
	}
	h.expect(h.do("POST", base, map[string]any{"tree_id": "mine", "on_conflict": "error"}), http.StatusConflict)

	collision := h.expect(h.do("POST", base, map[string]any{"tree_id": "german_focus"}), http.StatusUnprocessableEntity)
	if collision["detail"].(map[string]any)["error"] != "focus_file_policy_conflict" {
		t.Fatalf("primary vanilla id: %v", collision)
	}
	vanilla := h.expect(h.do("POST", base, map[string]any{"tree_id": "german_focus", "tree_kind": "shared_pool"}), http.StatusCreated)
	if w := vanilla["warnings"].([]any); len(w) != 1 || !strings.Contains(w[0].(string), "'TST_german_focus'") {
		t.Fatalf("vanilla collision warning: %v", vanilla["warnings"])
	}
	if vanilla["scope_kind"] != "shared_pool" || vanilla["file_policy"].(map[string]any)["source_file"] != "common/national_focus/TST_german_focus.txt" {
		t.Fatalf("shared pool: %v", vanilla)
	}

	position := h.expect(h.do("POST", base, map[string]any{"tree_id": "pos", "initial_show_position": map[string]any{"kind": "coordinates", "x": "3", "y": 4, "extra": 1}}), http.StatusCreated)
	isp := position["initial_show_position"].(map[string]any)
	if isp["kind"] != "coordinates" || isp["x"].(float64) != 3 || isp["y"].(float64) != 4 || isp["extra"] != nil {
		t.Fatalf("initial_show_position: %v", isp)
	}
	h.expect(h.do("POST", base, map[string]any{"initial_show_position": map[string]any{"kind": "nope"}}), http.StatusUnprocessableEntity)
	blocked := h.expect(h.do("POST", base, map[string]any{"initial_show_focus": "12"}), http.StatusUnprocessableEntity)
	if blocked["detail"].(map[string]any)["error"] != "focus_initial_position_blocked" {
		t.Fatalf("legacy numeric: %v", blocked)
	}
}

func TestNodeLifecycle(t *testing.T) {
	h := newHarness(t, nil)
	treeID := h.createTree(map[string]any{})

	a := h.createNode(treeID, map[string]any{"focus_id": " TST_a ", "x": 5, "y": 0, "completion_reward": "add_political_power = 10\\nadd_stability = 0.1"})
	if a["focus_id"] != "TST_a" || a["tree_edit_revision"] != "2" || a["icon"] != "GFX_goal_unknown" || a["cost"].(float64) != 10 || a["name"] != "New Focus" {
		t.Fatalf("node a: %v", a)
	}
	if a["completion_reward"] != "add_political_power = 10\nadd_stability = 0.1" {
		t.Fatalf("literal \\n not normalised: %q", a["completion_reward"])
	}
	if a["diagnostics"] == nil || a["tree_diagnostics"] == nil {
		t.Fatalf("diagnostics missing: %v", a)
	}
	dup := h.expect(h.do("POST", fmt.Sprintf("%s/%d/nodes", base, treeID), map[string]any{"focus_id": "TST_a"}), http.StatusConflict)
	if dup["detail"] != "focus_id 'TST_a' already exists in this project" {
		t.Fatalf("duplicate: %v", dup)
	}
	suffixed := h.createNode(treeID, map[string]any{"focus_id": "TST_a", "on_conflict": "suffix", "x": 9})
	if suffixed["focus_id"] != "TST_a_1" || len(suffixed["warnings"].([]any)) != 1 {
		t.Fatalf("suffix: %v", suffixed)
	}
	derived := h.createNode(treeID, map[string]any{"name": "Big Plan", "x": 12})
	if derived["focus_id"] != "TST_Big_Plan" {
		t.Fatalf("derived id: %v", derived["focus_id"])
	}
	h.expect(h.do("POST", fmt.Sprintf("%s/%d/nodes", base, treeID), map[string]any{"focus_id": "x", "on_conflict": "maybe"}), http.StatusUnprocessableEntity)
	h.expect(h.do("POST", base+"/99/nodes", map[string]any{"focus_id": "x"}), http.StatusNotFound)

	aID := int64(a["id"].(float64))
	b := h.createNode(treeID, map[string]any{
		"focus_id": "TST_b", "relative_position_id": "TST_a", "x": 1, "y": 1,
		"prerequisites": [][]string{{"TST_a", "TST_Big_Plan"}, {"TST_a"}}, "mutually_exclusive": []string{"TST_a"},
	})
	bID := int64(b["id"].(float64))

	same := h.createNode(treeID, map[string]any{"focus_id": "TST_same", "x": 5, "y": 0})
	if codes := diagnosticCodes(same["diagnostics"]); !contains(codes, "focus_layout_same_cell") {
		t.Fatalf("same cell diagnostics: %v", same["diagnostics"])
	}

	path := fmt.Sprintf("%s/%d/nodes/%d", base, treeID, aID)
	updated := h.expect(h.do("PUT", path, map[string]any{"x": 20, "name_zh": "甲", "tree_edit_revision": "x"}), http.StatusOK)
	if updated["x"].(float64) != 20 || updated["name_zh"] != "甲" {
		t.Fatalf("updated: %v", updated)
	}
	h.expect(h.do("PUT", path, map[string]any{"focus_id": "  "}), http.StatusBadRequest)
	h.expect(h.do("PUT", path, map[string]any{"focus_id": "TST_b"}), http.StatusConflict)
	notPersisted := h.expect(h.do("PUT", fmt.Sprintf("%s/%d/nodes/%d", base, treeID, int64(1)<<40), map[string]any{}), http.StatusNotFound)
	if notPersisted["detail"] != "Focus node not persisted" {
		t.Fatalf("not persisted: %v", notPersisted)
	}
	h.expect(h.do("PUT", fmt.Sprintf("%s/%d/nodes/999", base, treeID), map[string]any{}), http.StatusNotFound)
	ambiguous := h.expect(h.do("PUT", path, map[string]any{"completion_reward": `log = "a\nb"`}), http.StatusUnprocessableEntity)
	if !strings.Contains(fmt.Sprint(ambiguous["detail"]), "completion_reward") {
		t.Fatalf("ambiguous block text: %v", ambiguous)
	}

	got := h.expect(h.do("GET", path, nil), http.StatusOK)
	if got["x"].(float64) != 20 {
		t.Fatalf("get: %v", got)
	}
	if nodes := h.list(h.do("GET", fmt.Sprintf("%s/%d/nodes", base, treeID), nil), http.StatusOK); len(nodes) != 5 {
		t.Fatalf("nodes: %d", len(nodes))
	}

	forked := h.expect(h.do("POST", path+"/fork", map[string]any{"new_focus_id": "TST_a_copy"}), http.StatusCreated)
	if forked["focus_id"] != "TST_a_copy" || forked["x"].(float64) != 22 || forked["source"] != "created" || forked["name_zh"] != "甲" {
		t.Fatalf("fork: %v", forked)
	}
	h.expect(h.do("POST", path+"/fork", map[string]any{"new_focus_id": "TST_a"}), http.StatusBadRequest)
	h.expect(h.do("POST", path+"/fork", map[string]any{"new_focus_id": "TST_b"}), http.StatusConflict)
	h.expect(h.do("POST", path+"/fork", map[string]any{}), http.StatusUnprocessableEntity)

	h.expect(h.do("DELETE", path, nil), http.StatusNoContent)
	bAfter := h.expect(h.do("GET", fmt.Sprintf("%s/%d/nodes/%d", base, treeID, bID), nil), http.StatusOK)
	if bAfter["relative_position_id"] != "" || bAfter["x"].(float64) != 21 || bAfter["y"].(float64) != 1 {
		t.Fatalf("re-anchor: %v", bAfter)
	}
	if fmt.Sprint(bAfter["prerequisites"]) != "[[TST_Big_Plan]]" || len(bAfter["mutually_exclusive"].([]any)) != 0 {
		t.Fatalf("stripped refs: %v %v", bAfter["prerequisites"], bAfter["mutually_exclusive"])
	}
	h.expect(h.do("DELETE", path, nil), http.StatusNotFound)

	tree := h.expect(h.do("GET", fmt.Sprintf("%s/%d", base, treeID), nil), http.StatusOK)
	if tree["edit_revision"] == "1" {
		t.Fatalf("revision did not advance: %v", tree["edit_revision"])
	}
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

func TestMutualExclusionAndPositions(t *testing.T) {
	h := newHarness(t, nil)
	treeID := h.createTree(map[string]any{})
	a := h.createNode(treeID, map[string]any{"focus_id": "TST_a", "x": 0})
	b := h.createNode(treeID, map[string]any{"focus_id": "TST_b", "x": 3})
	aID, bID := int64(a["id"].(float64)), int64(b["id"].(float64))

	path := fmt.Sprintf("%s/%d/nodes/%d/mutual-exclusion", base, treeID, aID)
	pair := h.list(h.do("PUT", path, map[string]any{"other_node_id": bID, "enabled": true}), http.StatusOK)
	if len(pair) != 2 || fmt.Sprint(pair[0].(map[string]any)["mutually_exclusive"]) != "[TST_b]" ||
		fmt.Sprint(pair[1].(map[string]any)["mutually_exclusive"]) != "[TST_a]" {
		t.Fatalf("mutex on: %v", pair)
	}
	pair = h.list(h.do("PUT", path, map[string]any{"other_node_id": bID, "enabled": false}), http.StatusOK)
	if len(pair[0].(map[string]any)["mutually_exclusive"].([]any)) != 0 {
		t.Fatalf("mutex off: %v", pair)
	}
	h.expect(h.do("PUT", path, map[string]any{"other_node_id": bID}), http.StatusUnprocessableEntity)
	h.expect(h.do("PUT", path, map[string]any{"other_node_id": aID, "enabled": true}), http.StatusUnprocessableEntity)

	before := h.expect(h.do("GET", fmt.Sprintf("%s/%d", base, treeID), nil), http.StatusOK)["edit_revision"]
	moved := h.list(h.do("PUT", fmt.Sprintf("%s/%d/node-positions", base, treeID), map[string]any{
		"positions": []map[string]any{{"id": aID, "x": 4, "y": 2}, {"id": bID, "x": 6, "y": 2}, {"id": 999, "x": 1, "y": 1}},
	}), http.StatusOK)
	if len(moved) != 2 || moved[0].(map[string]any)["x"].(float64) != 4 || moved[1].(map[string]any)["y"].(float64) != 2 {
		t.Fatalf("moved: %v", moved)
	}
	if moved[0].(map[string]any)["tree_diagnostics"] == nil || moved[1].(map[string]any)["tree_diagnostics"] != nil {
		t.Fatalf("tree diagnostics belong to the first node only")
	}
	after := h.expect(h.do("GET", fmt.Sprintf("%s/%d", base, treeID), nil), http.StatusOK)["edit_revision"]
	if before == after {
		t.Fatalf("positions did not advance the revision")
	}

	none := h.do("PUT", fmt.Sprintf("%s/%d/node-positions", base, treeID), map[string]any{"positions": []map[string]any{{"id": 999, "x": 1, "y": 1}}})
	if out := h.list(none, http.StatusOK); len(out) != 0 || none.Header().Get("X-Edit-Revisions") != "" {
		t.Fatalf("no-match positions: %v", out)
	}
	if again := h.expect(h.do("GET", fmt.Sprintf("%s/%d", base, treeID), nil), http.StatusOK)["edit_revision"]; again != after {
		t.Fatalf("revision moved without a change")
	}
	h.expect(h.do("PUT", fmt.Sprintf("%s/%d/node-positions", base, treeID), map[string]any{"positions": []map[string]any{{"id": 0, "x": 1, "y": 1}}}), http.StatusUnprocessableEntity)
}

func TestImportedTreeBecomesCreatedOnEdit(t *testing.T) {
	h := newHarness(t, nil)
	err := h.p.Write([]string{"focus_trees", "focus_nodes"}, func(tx *project.Tx) error {
		tx.Insert("focus_trees", project.Row{"tree_id": "imp", "tree_kind": "primary", "scope_kind": "own", "country_tag": "TST",
			"source": "imported", "source_file": "common/national_focus/imp.txt", "initial_show_position": map[string]any{}})
		tx.Insert("focus_nodes", project.Row{"focus_tree_id": int64(1), "focus_id": "TST_imp", "name": "TST_imp", "x": json.Number("3"), "y": json.Number("0"),
			"source": "imported", "display_name_locs": map[string]any{}, "description_locs": map[string]any{},
			"source_snapshot": map[string]any{"focus_id": "TST_imp", "x": json.Number("3"), "y": json.Number("0")}})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	tree := h.expect(h.do("GET", base+"/1", nil), http.StatusOK)
	if tree["source"] != "imported" || tree["source_file"] != "common/national_focus/imp.txt" {
		t.Fatalf("imported tree: %v", tree)
	}
	h.expect(h.do("PUT", base+"/1/nodes/1", map[string]any{"x": 4}), http.StatusOK)
	tree = h.expect(h.do("GET", base+"/1", nil), http.StatusOK)
	if tree["source"] != "created" {
		t.Fatalf("source after edit: %v", tree["source"])
	}

	if codes := diagnosticCodes(tree["diagnostics"]); !contains(codes, "focus_layout_replacement_global_shift") {
		t.Fatalf("global shift diagnostic: %v", tree["diagnostics"])
	}
}

func TestPreviewAndParsePreview(t *testing.T) {
	h := newHarness(t, nil)
	treeID := h.createTree(map[string]any{})
	node := h.createNode(treeID, map[string]any{"focus_id": "TST_a", "cost": 5, "completion_reward": "add_political_power = 10"})
	nodeID := int64(node["id"].(float64))

	preview := h.expect(h.do("GET", fmt.Sprintf("%s/%d/preview", base, treeID), nil), http.StatusOK)
	script, _ := preview["script"].(string)
	if !strings.Contains(script, "focus_tree = {") || !strings.Contains(script, "id = TST_a") || !strings.Contains(script, "add_political_power = 10") {
		t.Fatalf("tree preview:\n%s", script)
	}
	if preview["tree_edit_revision"] != "2" || preview["tree_id"].(float64) != float64(treeID) {
		t.Fatalf("preview fields: %v", preview)
	}
	one := h.expect(h.do("GET", fmt.Sprintf("%s/%d/preview?node_id=%d", base, treeID, nodeID), nil), http.StatusOK)
	if s := one["script"].(string); !strings.HasPrefix(strings.TrimSpace(s), "focus = {") || one["node_id"].(float64) != float64(nodeID) {
		t.Fatalf("node preview: %v", one)
	}
	h.expect(h.do("GET", fmt.Sprintf("%s/%d/preview?node_id=abc", base, treeID), nil), http.StatusUnprocessableEntity)
	h.expect(h.do("GET", fmt.Sprintf("%s/%d/preview?node_id=999", base, treeID), nil), http.StatusNotFound)
	h.expect(h.do("GET", base+"/99/preview", nil), http.StatusNotFound)

	parsed := h.expect(h.do("POST", fmt.Sprintf("%s/%d/nodes/%d/parse-preview", base, treeID, nodeID),
		map[string]any{"script": "focus = {\n\tid = TST_x\n\tcost = 7\n\tx = 2\n\tcompletion_reward = { add_stability = 0.1 }\n}"}), http.StatusOK)
	entity, _ := parsed["entity"].(map[string]any)
	if entity == nil || entity["focus_id"] != "TST_x" || fmt.Sprint(entity["cost"]) != "7" || entity["name"] != nil {
		t.Fatalf("parse preview: %v", parsed)
	}
	empty := h.expect(h.do("POST", fmt.Sprintf("%s/%d/nodes/%d/parse-preview", base, treeID, nodeID), map[string]any{"script": " "}), http.StatusOK)
	if errs := empty["parse_errors"].([]any); len(errs) != 1 || errs[0].(map[string]any)["severity"] != "info" {
		t.Fatalf("empty script: %v", empty)
	}
	broken := h.expect(h.do("POST", fmt.Sprintf("%s/%d/nodes/%d/parse-preview", base, treeID, nodeID), map[string]any{"script": "focus = { id = a }}"}), http.StatusOK)
	if errs := broken["parse_errors"].([]any); len(errs) == 0 && broken["entity"] == nil {
		t.Fatalf("broken script: %v", broken)
	}
	h.expect(h.do("POST", fmt.Sprintf("%s/%d/nodes/%d/parse-preview", base, treeID, nodeID), map[string]any{}), http.StatusUnprocessableEntity)
	missing := h.expect(h.do("POST", fmt.Sprintf("%s/%d/nodes/999/parse-preview", base, treeID), map[string]any{"script": "x"}), http.StatusNotFound)
	if missing["detail"] != "Focus node not found" {
		t.Fatalf("missing node: %v", missing)
	}
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 4), G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func multipartBody(t *testing.T, fields map[string]string, file []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}
	if file != nil {
		part, err := writer.CreateFormFile("file", "icon.png")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write(file)
	}
	_ = writer.Close()
	return &buf, writer.FormDataContentType()
}

func (h *harness) upload(path string, fields map[string]string, file []byte, headers ...string) *httptest.ResponseRecorder {
	h.t.Helper()
	body, contentType := multipartBody(h.t, fields, file)
	req := httptest.NewRequest("POST", path, body)
	req.Header.Set("Content-Type", contentType)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	return rec
}

func testGame(t *testing.T) fstest.MapFS {
	t.Helper()
	pix := make([]byte, 8*8*4)
	for i := range pix {
		pix[i] = 200
	}
	return fstest.MapFS{
		"interface/goals.gfx": {Data: []byte(`spriteTypes = {
	spriteType = { name = "GFX_goal_generic_test" texturefile = "gfx/interface/goals/goal_generic_test.dds" }
	spriteType = { name = "GFX_focus_TST_special" texturefile = "gfx/interface/goals/focus_tst_special.dds" }
}`)},
		"interface/goals_shine.gfx":                 {Data: []byte(`spriteTypes = { spriteType = { name = "GFX_goal_generic_test_shine" texturefile = "gfx/interface/goals/goal_generic_test.dds" } }`)},
		"interface/eventpictures.gfx":               {Data: []byte(`spriteTypes = { spriteType = { name = "GFX_report_event_test" texturefile = "gfx/event_pictures/test.dds" } }`)},
		"gfx/interface/goals/goal_generic_test.dds": {Data: imgcodec.EncodeDXT5DDS(pix, 8, 8)},
		"common/technology_tags/00_technology.txt":  {Data: []byte("technology_categories = {\n\tindustry\n\tconstruction_tech\n}\n")},
	}
}

func TestFocusIcons(t *testing.T) {
	h := newHarness(t, testGame(t))
	treeID := h.createTree(map[string]any{})
	node := h.createNode(treeID, map[string]any{"focus_id": "TST_a"})
	nodeID := int64(node["id"].(float64))

	rec := h.upload(fmt.Sprintf("%s/%d/nodes/%d/upload-icon", base, treeID, nodeID), nil, testPNG(t, 40, 30))
	uploaded := h.expect(rec, http.StatusOK)
	if uploaded["icon"] != "GFX_focus_tst_a" || uploaded["filename"] != "focus_tst_a.png" ||
		uploaded["preview_url"] != "/api/go/projects/1/focus-icons/GFX_focus_tst_a" {
		t.Fatalf("upload: %v", uploaded)
	}
	if rec.Header().Get("X-Edit-Revisions") == "" {
		t.Fatalf("upload must report the tree revision")
	}
	got := h.expect(h.do("GET", fmt.Sprintf("%s/%d/nodes/%d", base, treeID, nodeID), nil), http.StatusOK)
	if got["icon"] != "GFX_focus_tst_a" {
		t.Fatalf("node icon: %v", got["icon"])
	}
	served := h.do("GET", "/api/go/projects/1/focus-icons/GFX_focus_tst_a", nil)
	if served.Code != http.StatusOK || served.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("serve: %d %s", served.Code, served.Header().Get("Content-Type"))
	}
	if img, err := png.Decode(bytes.NewReader(served.Body.Bytes())); err != nil || img.Bounds().Dx() != 100 || img.Bounds().Dy() != 88 {
		t.Fatalf("served icon is not the canonical 100x88 PNG: %v", err)
	}
	h.expect(h.do("GET", "/api/go/projects/1/focus-icons/GFX_missing", nil), http.StatusNotFound)
	h.expect(h.upload(fmt.Sprintf("%s/%d/nodes/999/upload-icon", base, treeID), nil, testPNG(t, 4, 4)), http.StatusNotFound)
	h.expect(h.upload(fmt.Sprintf("%s/%d/nodes/%d/upload-icon", base, treeID, nodeID), nil, nil), http.StatusUnprocessableEntity)
	h.expect(h.upload(fmt.Sprintf("%s/%d/nodes/%d/upload-icon", base, treeID, nodeID), nil, []byte("not an image")), http.StatusBadRequest)

	named := h.expect(h.upload("/api/go/projects/1/focus-icons/upload-named", map[string]string{"gfx_name": "dir/GFX_focus_custom.png"}, testPNG(t, 10, 10)), http.StatusOK)
	if named["gfx_name"] != "GFX_focus_custom" || named["saved"] != true {
		t.Fatalf("upload named: %v", named)
	}
	h.expect(h.upload("/api/go/projects/1/focus-icons/upload-named", map[string]string{"gfx_name": "bad name!"}, testPNG(t, 10, 10)), http.StatusBadRequest)

	icons := h.expect(h.do("GET", "/api/go/projects/1/focus-icons", nil), http.StatusOK)
	if fmt.Sprint(icons["icons"]) != "[GFX_focus_custom GFX_focus_tst_a]" {
		t.Fatalf("icons: %v", icons)
	}

	library := h.expect(h.do("GET", "/api/go/projects/1/focus-icons/library", nil), http.StatusOK)
	if library["total"].(float64) != 4 {
		t.Fatalf("library: %v", library)
	}
	first := library["icons"].([]any)[0].(map[string]any)
	if first["name"] != "GFX_focus_TST_special" || first["source"] != "vanilla" || first["category"] != "focus" ||
		first["thumbnail_url"] != "/api/go/vanilla/focus-icon/GFX_focus_TST_special" {
		t.Fatalf("first library icon: %v", first)
	}
	projectOnly := h.expect(h.do("GET", "/api/go/projects/1/focus-icons/library?source=project&q=custom", nil), http.StatusOK)
	if projectOnly["returned"].(float64) != 1 || projectOnly["icons"].([]any)[0].(map[string]any)["thumbnail_url"] != "/api/go/projects/1/focus-icons/GFX_focus_custom" {
		t.Fatalf("project library: %v", projectOnly)
	}
	h.expect(h.do("GET", "/api/go/projects/1/focus-icons/library?source=nope", nil), http.StatusBadRequest)
	h.expect(h.do("GET", "/api/go/projects/1/focus-icons/library?limit=x", nil), http.StatusUnprocessableEntity)

	vanilla := h.do("GET", "/api/go/vanilla/focus-icon/GFX_goal_generic_test", nil)
	if vanilla.Code != http.StatusOK || vanilla.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("vanilla icon: %d %s", vanilla.Code, vanilla.Body.String())
	}
	if img, err := png.Decode(bytes.NewReader(vanilla.Body.Bytes())); err != nil || img.Bounds().Dx() != 8 {
		t.Fatalf("vanilla PNG: %v", err)
	}
	missingTexture := h.expect(h.do("GET", "/api/go/vanilla/focus-icon/GFX_focus_TST_special", nil), http.StatusNotFound)
	if !strings.Contains(missingTexture["detail"].(string), "vanilla DDS file missing") {
		t.Fatalf("missing texture: %v", missingTexture)
	}

	h.expect(h.do("GET", "/api/go/vanilla/focus-icon/GFX_goal_generic_test_shine", nil), http.StatusOK)
	h.expect(h.do("GET", "/api/go/vanilla/focus-icon/GFX_nowhere", nil), http.StatusNotFound)
}

func TestCreateTreeWithoutGameFolder(t *testing.T) {
	restore := gameindex.UseStatic(map[string][]byte{})
	defer restore()
	h := newHarness(t, nil)
	for _, body := range []map[string]any{{}, {"tree_id": "mine"}} {
		refused := h.expect(h.do("POST", base, body), http.StatusConflict)
		if refused["code"] != "game_folder_missing" || !strings.Contains(fmt.Sprint(refused["detail"]), "Settings") {
			t.Fatalf("without a game folder %v: %v", body, refused)
		}
	}
}

func TestLibraryWithoutGame(t *testing.T) {
	h := newHarness(t, nil)
	library := h.expect(h.do("GET", "/api/go/projects/1/focus-icons/library", nil), http.StatusOK)
	if library["total"].(float64) != 0 {
		t.Fatalf("library without game: %v", library)
	}
	h.expect(h.do("GET", "/api/go/vanilla/focus-icon/GFX_goal_unknown", nil), http.StatusNotFound)
}

func TestDeadFlags(t *testing.T) {
	h := newHarness(t, nil)
	treeID := h.createTree(map[string]any{})
	h.createNode(treeID, map[string]any{"focus_id": "TST_a", "completion_reward": "set_country_flag = my_flag\nset_country_flag = checked_flag\nset_global_flag = dyn@ROOT"})
	h.createNode(treeID, map[string]any{"focus_id": "TST_b", "x": 3, "available": "has_country_flag = checked_flag"})
	report := h.expect(h.do("GET", "/api/go/lint/dead-flags/1", nil), http.StatusOK)
	dead := report["dead_flags"].([]any)
	if len(dead) != 1 || dead[0].(map[string]any)["flag_id"] != "my_flag" ||
		fmt.Sprint(dead[0].(map[string]any)["set_locations"]) != "[focus TST_a.completion_reward]" {
		t.Fatalf("dead flags: %v", report)
	}
	if report["set_total"].(float64) != 2 || report["check_total"].(float64) != 1 {
		t.Fatalf("totals: %v", report)
	}
}

func TestTreeValidation(t *testing.T) {
	h := newHarness(t, testGame(t))
	treeID := h.createTree(map[string]any{})
	h.createNode(treeID, map[string]any{"focus_id": "TST_a", "prerequisites": [][]string{{"TST_missing"}},
		"completion_reward": "add_tech_bonus = { bonus = 0.5 category = construction }", "icon": "GFX_goal_generic_test"})
	h.createNode(treeID, map[string]any{"focus_id": "TST_b", "x": 3, "prerequisites": [][]string{{"TST_c"}}, "completion_reward": "add_stability = 0.1", "icon": "GFX_nowhere"})
	h.createNode(treeID, map[string]any{"focus_id": "TST_c", "x": 6, "prerequisites": [][]string{{"TST_b"}}, "completion_reward": "add_stability = 0.1"})
	report := h.expect(h.do("GET", "/api/go/lint/tree-validation/1", nil), http.StatusOK)
	issues := report["focus"].([]any)
	var messages []string
	for _, item := range issues {
		messages = append(messages, item.(map[string]any)["msg"].(string))
	}
	joined := strings.Join(messages, "\n")
	for _, want := range []string{"prereq focus_id 'TST_missing' 不存在", "prerequisites 成环", "技术类别不存在", "GFX 'GFX_nowhere'"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "GFX_goal_generic_test'") {
		t.Fatalf("a base-game icon was reported:\n%s", joined)
	}
	totals := report["totals"].(map[string]any)
	if totals["error"].(float64) < 2 {
		t.Fatalf("totals: %v", totals)
	}
	for _, key := range []string{"decision", "event", "idea"} {
		if _, ok := report[key].([]any); !ok {
			t.Fatalf("%s list missing: %v", key, report)
		}
	}
}
