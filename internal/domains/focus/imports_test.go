package focus

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func (h *harness) uploadFile(path, filename string, data []byte, fields map[string]string) *httptest.ResponseRecorder {
	h.t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}
	part, _ := writer.CreateFormFile("file", filename)
	_, _ = part.Write(data)
	_ = writer.Close()
	req := httptest.NewRequest("POST", path, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	return rec
}

const focusScript = `focus_tree = {
	id = tst_imported_tree
	country = { factor = 0 modifier = { add = 10 tag = TST } }
	focus = {
		id = TST_one
		icon = GFX_goal_generic_test
		x = 3
		y = 0
		cost = 7
		completion_reward = { add_political_power = 50 }
	}
	focus = {
		id = TST_two
		prerequisite = { focus = TST_one }
		relative_position_id = TST_one
		x = 0
		y = 1
		completion_reward = { add_stability = 0.05 }
	}
}
`

func TestImportFocusTreeScript(t *testing.T) {
	h := newHarness(t, nil)
	rec := h.uploadFile("/api/go/projects/1/import/focus-tree", "tst.txt", []byte(focusScript), nil)
	result := h.expect(rec, http.StatusOK)
	if result["tree_id"] != "tst_imported_tree" || result["tree_action"] != "created" || result["created_nodes"].(float64) != 2 || result["conflict_policy"] != "overwrite" {
		t.Fatalf("import: %v", result)
	}
	if rec.Header().Get("X-Edit-Revisions") == "" {
		t.Fatalf("import must report the new tree's revision")
	}
	trees := h.list(h.do("GET", base, nil), http.StatusOK)
	if len(trees) != 1 {
		t.Fatalf("trees: %v", trees)
	}
	tree := trees[0].(map[string]any)
	nodes := tree["nodes"].([]any)
	if tree["country_tag"] != "TST" || tree["source_file"] != "tst.txt" || len(nodes) != 2 {
		t.Fatalf("imported tree: %v", tree)
	}
	two := nodes[1].(map[string]any)
	if two["focus_id"] != "TST_two" || fmt.Sprint(two["prerequisites"]) != "[[TST_one]]" || two["relative_position_id"] != "TST_one" || two["source"] != "created" {
		t.Fatalf("imported node: %v", two)
	}
	if policy, _ := tree["file_policy"].(map[string]any); policy == nil {
		t.Fatalf("no file policy: %v", tree)
	}

	skipped := h.expect(h.uploadFile("/api/go/projects/1/import/focus-tree", "tst.txt", []byte(focusScript), map[string]string{"conflict_policy": "skip"}), http.StatusOK)
	if skipped["tree_action"] != "skipped" || skipped["created_nodes"].(float64) != 0 {
		t.Fatalf("skip: %v", skipped)
	}
	replaced := h.expect(h.uploadFile("/api/go/projects/1/import/focus-tree", "tst.txt", []byte(focusScript), nil), http.StatusOK)
	if replaced["tree_action"] != "updated" {
		t.Fatalf("overwrite: %v", replaced)
	}
	if trees := h.list(h.do("GET", base, nil), http.StatusOK); len(trees) != 1 || len(trees[0].(map[string]any)["nodes"].([]any)) != 2 {
		t.Fatalf("after overwrite: %v", trees)
	}
	h.expect(h.uploadFile("/api/go/projects/1/import/focus-tree", "tst.txt", []byte(focusScript), map[string]string{"conflict_policy": "merge"}), http.StatusBadRequest)
	none := h.expect(h.uploadFile("/api/go/projects/1/import/focus-tree", "x.txt", []byte("ideas = { }"), nil), http.StatusOK)
	if none["tree_action"] != "none" || len(none["warnings"].([]any)) != 1 {
		t.Fatalf("no tree: %v", none)
	}
}

const drawIOXML = `<mxGraphModel><root>
<mxCell id="0"/>
<mxCell id="a" value="Industry" vertex="1"><mxGeometry x="0" y="0"/></mxCell>
<mxCell id="b" value="Factories" vertex="1"><mxGeometry x="0" y="100"/></mxCell>
<mxCell id="e" edge="1" source="a" target="b"/>
</root></mxGraphModel>`

func TestImportDrawIO(t *testing.T) {
	h := newHarness(t, nil)
	result := h.expect(h.uploadFile("/api/go/projects/1/import/drawio", "tree.drawio", []byte(drawIOXML), map[string]string{"tree_name": "plan"}), http.StatusOK)
	if result["tree_id"] != "tst_plan" || result["focuses_created"].(float64) != 2 || result["prereq_edges_created"].(float64) != 1 {
		t.Fatalf("drawio: %v", result)
	}
	tree := h.expect(h.do("GET", fmt.Sprintf("%s/%v", base, result["tree_db_id"]), nil), http.StatusOK)
	nodes := tree["nodes"].([]any)
	if len(nodes) != 2 || nodes[0].(map[string]any)["focus_id"] != "TST_Industry" ||
		fmt.Sprint(nodes[1].(map[string]any)["prerequisites"]) != "[[TST_Industry]]" {
		t.Fatalf("drawio nodes: %v", nodes)
	}
	again := h.expect(h.uploadFile("/api/go/projects/1/import/drawio", "tree.drawio", []byte(drawIOXML), map[string]string{"tree_name": "plan"}), http.StatusOK)
	if again["tree_id"] != "tst_plan_2" {
		t.Fatalf("second import: %v", again["tree_id"])
	}
	bad := h.expect(h.uploadFile("/api/go/projects/1/import/drawio", "tree.drawio", []byte("<mxGraphModel><root>"), nil), http.StatusBadRequest)
	if !strings.Contains(bad["detail"].(string), "Draw.io XML parse failed") {
		t.Fatalf("bad xml: %v", bad)
	}
}
