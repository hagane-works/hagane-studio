package rawfiles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

func newProject(t *testing.T) (*project.Project, *http.ServeMux) {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Register(mux, domains.Deps{Open: func(*http.Request) (*project.Project, error) { return p, nil }})
	return p, mux
}

func do(t *testing.T, mux *http.ServeMux, method, path string, body any, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
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
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) any {
	t.Helper()
	var out any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body is not JSON: %s", rec.Body.String())
	}
	return out
}

func expect(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, want %d: %s", rec.Code, status, rec.Body.String())
	}
}

func multipartBody(t *testing.T, fields map[string]string, file []byte, filename string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	if file != nil {
		part, _ := w.CreateFormFile("file", filename)
		_, _ = part.Write(file)
	}
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

func postMultipart(t *testing.T, mux *http.ServeMux, path string, fields map[string]string, file []byte) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := multipartBody(t, fields, file, "upload.txt")
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const base = "/api/go/projects/1/raw-files"

func TestRawFilesCRUD(t *testing.T) {
	p, mux := newProject(t)

	rec := do(t, mux, "GET", base, nil, nil)
	expect(t, rec, 200)
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("empty list = %s", rec.Body.String())
	}

	rec = do(t, mux, "POST", base, map[string]any{
		"dest_path": `\common\on_actions\b_on_actions.txt`, "content": "on_actions = { }", "note": "n",
	}, nil)
	expect(t, rec, 201)
	created := decode(t, rec).(map[string]any)
	for key, want := range map[string]any{
		"dest_path": "common/on_actions/b_on_actions.txt", "content": "on_actions = { }",
		"is_binary": false, "generate_replace_path": false, "raw_kind": "verbatim_override",
		"target_layer": nil, "target_sha256": nil, "note": "n", "project_id": float64(1),
	} {
		if created[key] != want {
			t.Errorf("created[%s] = %#v, want %#v", key, created[key], want)
		}
	}
	if w, ok := created["warnings"].([]any); !ok || len(w) != 0 {
		t.Errorf("warnings = %#v", created["warnings"])
	}
	if created["created_at"] == "" || created["updated_at"] == "" {
		t.Errorf("timestamps missing: %#v", created)
	}
	id := int64(created["id"].(float64))

	rec = do(t, mux, "POST", base, map[string]any{"dest_path": "common/on_actions/b_on_actions.txt"}, nil)
	expect(t, rec, 409)

	rec = do(t, mux, "POST", base, map[string]any{"dest_path": "a.txt", "content": "x = \"a\\nb\""}, nil)
	expect(t, rec, 201)
	second := decode(t, rec).(map[string]any)
	if w := second["warnings"].([]any); len(w) != 1 || w[0].(map[string]any)["code"] != RawContentLiteralNewline {
		t.Errorf("literal \\n warning = %#v", second["warnings"])
	}
	rec = do(t, mux, "GET", base, nil, nil)
	list := decode(t, rec).([]any)
	if len(list) != 2 || list[0].(map[string]any)["dest_path"] != "a.txt" {
		t.Fatalf("list order = %s", rec.Body.String())
	}
	if w := list[0].(map[string]any)["warnings"].([]any); len(w) != 0 {
		t.Errorf("list warnings must be empty: %#v", w)
	}

	rec = do(t, mux, "GET", fmt.Sprintf("%s/%d", base, id), nil, nil)
	expect(t, rec, 200)
	rec = do(t, mux, "GET", base+"/999", nil, nil)
	expect(t, rec, 404)
	if decode(t, rec).(map[string]any)["detail"] != "RawFile not found" {
		t.Errorf("404 body = %s", rec.Body.String())
	}
	rec = do(t, mux, "GET", base+"/abc", nil, nil)
	expect(t, rec, 422)
	rec = do(t, mux, "GET", base+"/from-template", nil, nil)
	expect(t, rec, 405)

	rec = do(t, mux, "PUT", fmt.Sprintf("%s/%d", base, id), map[string]any{"content": "on_actions = { on_startup = { } }", "note": "m"}, nil)
	expect(t, rec, 200)
	updated := decode(t, rec).(map[string]any)
	if updated["content"] != "on_actions = { on_startup = { } }" || updated["note"] != "m" {
		t.Fatalf("updated = %#v", updated)
	}
	rec = do(t, mux, "PUT", fmt.Sprintf("%s/%d", base, id), map[string]any{"dest_path": "a.txt"}, nil)
	expect(t, rec, 409)
	rec = do(t, mux, "PUT", fmt.Sprintf("%s/%d", base, id), map[string]any{"content": "x = {"}, nil)
	expect(t, rec, 422)
	if decode(t, rec).(map[string]any)["detail"].(map[string]any)["code"] != RawContentBraceCode {
		t.Fatalf("brace 422 = %s", rec.Body.String())
	}
	rec = do(t, mux, "PUT", base+"/999", map[string]any{"content": "x"}, nil)
	expect(t, rec, 404)

	if err := p.Write([]string{table}, func(tx *project.Tx) error {
		_, err := tx.Update(table, id, func(r project.Row) { r["source"] = "imported" })
		return err
	}); err != nil {
		t.Fatal(err)
	}
	expect(t, do(t, mux, "PUT", fmt.Sprintf("%s/%d", base, id), map[string]any{"note": "z"}, nil), 200)
	rows, _ := p.Rows(table)
	for _, row := range rows {
		if project.Int64(row["id"]) == id && row["source"] != "created" {
			t.Errorf("source = %v", row["source"])
		}
	}

	expect(t, do(t, mux, "DELETE", fmt.Sprintf("%s/%d", base, id), nil, nil), 204)
	expect(t, do(t, mux, "DELETE", fmt.Sprintf("%s/%d", base, id), nil, nil), 404)
}

func TestRawFilesCreateValidation(t *testing.T) {
	_, mux := newProject(t)
	rec := do(t, mux, "POST", base, map[string]any{"path": "common/x.txt"}, nil)
	expect(t, rec, 422)
	detail := decode(t, rec).(map[string]any)["detail"].([]any)[0].(map[string]any)
	if detail["type"] != "value_error" || !strings.Contains(detail["msg"].(string), "not 'path'") {
		t.Fatalf("alias guard = %#v", detail)
	}
	rec = do(t, mux, "POST", base, map[string]any{"dest_path": "descriptor.mod"}, nil)
	expect(t, rec, 422)
	rec = do(t, mux, "POST", base, map[string]any{"dest_path": "../x.txt"}, nil)
	expect(t, rec, 422)
	rec = do(t, mux, "POST", base, "{not json", nil)
	expect(t, rec, 422)
	rec = do(t, mux, "POST", base, map[string]any{"dest_path": "events/x.txt", "content": "a = {"}, nil)
	expect(t, rec, 422)
	if d := decode(t, rec).(map[string]any)["detail"].(map[string]any); d["code"] != RawContentBraceCode ||
		d["repair_route"] != RawContentBraceRepairRoute || d["findings"] == nil {
		t.Fatalf("brace detail = %#v", d)
	}

	rec = do(t, mux, "POST", base, map[string]any{
		"dest_path": "common/x.txt", "raw_kind": "empty_shadow", "content": "",
		"target_layer": "base game", "target_sha256": strings.Repeat("a", 64),
	}, nil)
	expect(t, rec, 422)
	if decode(t, rec).(map[string]any)["detail"].(map[string]any)["code"] != "empty_shadow_game_version_required" {
		t.Fatalf("empty shadow = %s", rec.Body.String())
	}
}

func TestRawFilesEmptyShadowPolicyConflict(t *testing.T) {
	p, mux := newProject(t)
	if _, err := p.UpdateRecord(func(r project.Row) { r["supported_version"] = "1.19.2" }); err != nil {
		t.Fatal(err)
	}
	rec := do(t, mux, "POST", base, map[string]any{
		"dest_path": "common/x.txt", "raw_kind": "empty_shadow", "content": "",
		"target_layer": "base game", "target_sha256": strings.Repeat("a", 64),
	}, nil)
	expect(t, rec, 409)
	d := decode(t, rec).(map[string]any)["detail"].(map[string]any)
	if d["code"] != "empty_shadow_policy_unverified" || d["project_supported_version"] != "1.19.2" ||
		d["repair_route"] != "raw_file.empty_shadow_evidence" {
		t.Fatalf("detail = %#v", d)
	}
}

func TestRawFilesBinary(t *testing.T) {
	p, mux := newProject(t)
	data := []byte("DDS fake bytes")
	rec := postMultipart(t, mux, base+"/binary", map[string]string{"dest_path": "gfx/x.dds", "note": "img"}, data)
	expect(t, rec, 201)
	item := decode(t, rec).(map[string]any)
	if item["is_binary"] != true || item["content"] != "" || item["note"] != "img" {
		t.Fatalf("binary item = %#v", item)
	}
	if _, leaked := item["content_obs_key"]; leaked {
		t.Fatal("storage pointer must not be serialised")
	}
	rows, _ := p.Rows(table)
	key, _ := rows[0]["content_obs_key"].(string)
	if !strings.HasPrefix(key, "local/rawblobs/") || rows[0]["content_sha256"] == nil {
		t.Fatalf("pointer = %#v", rows[0])
	}
	stored, err := p.Blobs().Get(key, 1<<20)
	if err != nil || !bytes.Equal(stored, data) {
		t.Fatalf("blob = %q, %v", stored, err)
	}
	rec = postMultipart(t, mux, base+"/binary", map[string]string{"dest_path": "gfx/x.dds"}, data)
	expect(t, rec, 409)
	rec = postMultipart(t, mux, base+"/binary", map[string]string{"dest_path": "gfx/y.dds"}, nil)
	expect(t, rec, 422)
	rec = postMultipart(t, mux, base+"/binary", map[string]string{"dest_path": "noext"}, data)
	expect(t, rec, 422)
}

func TestRawFilesTemplates(t *testing.T) {
	p, mux := newProject(t)
	rec := do(t, mux, "GET", base+"/templates", nil, nil)
	expect(t, rec, 200)
	templates := decode(t, rec).(map[string]any)["templates"].([]any)
	if len(templates) != 9 || templates[0].(map[string]any)["key"] != "scripted_gui" {
		t.Fatalf("templates = %d", len(templates))
	}

	rec = do(t, mux, "POST", base+"/from-template", map[string]any{
		"panel_key": "my_panel", "title": "My Panel", "variables": []any{map[string]any{"key": "v", "label": "V"}},
	}, nil)
	expect(t, rec, 201)
	files := decode(t, rec).([]any)
	if len(files) != 10 || files[0].(map[string]any)["dest_path"] != "interface/my_panel_hagane.gui" ||
		files[0].(map[string]any)["note"] != "generated from scripted_gui template" {
		t.Fatalf("scripted_gui files = %s", rec.Body.String())
	}

	expect(t, do(t, mux, "POST", base+"/from-template", map[string]any{"panel_key": "my_panel"}, nil), 201)
	rows, _ := p.Rows(table)
	if len(rows) != 10 {
		t.Fatalf("rows after upsert = %d", len(rows))
	}

	rec = do(t, mux, "POST", base+"/from-template/on-actions", map[string]any{"panel_key": "hooks", "country_tag": "GER"}, nil)
	expect(t, rec, 201)
	if f := decode(t, rec).([]any); len(f) != 1 || f[0].(map[string]any)["dest_path"] != "common/on_actions/hooks_on_actions.txt" {
		t.Fatalf("on_actions = %s", rec.Body.String())
	}
	expect(t, do(t, mux, "POST", base+"/from-template/defines", map[string]any{"key": "my_tweaks"}, nil), 201)
	expect(t, do(t, mux, "POST", base+"/from-template/defines", map[string]any{"key": "X"}, nil), 422)
	expect(t, do(t, mux, "POST", base+"/from-template/on-actions", map[string]any{}, nil), 422)
	expect(t, do(t, mux, "POST", base+"/from-template/frontendgamesetup", map[string]any{"base_gui_text": "x", "slot_width": 100}, nil), 422)
	expect(t, do(t, mux, "POST", base+"/from-template/nope", map[string]any{"key": "abc"}, nil), 404)
}

func TestCrossRefRepairPreviewApply(t *testing.T) {
	p, mux := newProject(t)
	text := "add_ideas = GER_war_economi\n"
	var ideaID int64
	if err := p.Write([]string{"ideas"}, func(tx *project.Tx) error {
		ideaID = project.Int64(tx.Insert("ideas", project.Row{"idea_id": "x", "on_add": text})["id"])
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cands := CollectPositionedReferenceCandidates(text, nil)
	if len(cands) != 1 {
		t.Fatalf("candidates = %d", len(cands))
	}
	body := map[string]any{
		"entity_kind": "idea", "entity_id": ideaID, "field": "on_add", "target_id": "GER_war_economi",
		"target_start_offset": cands[0].SourceSpan.StartOffset, "target_end_offset": cands[0].SourceSpan.EndOffset,
		"replacement_id": "GER_war_economy",
	}
	rec := do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/preview", body, nil)
	expect(t, rec, 200)
	preview := decode(t, rec).(map[string]any)
	if preview["applied"] != false || preview["source_sha256"] != sha256Hex(text) ||
		len(preview["edit_revisions"].(map[string]any)) != 0 {
		t.Fatalf("preview = %#v", preview)
	}

	rec = do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/apply", body, nil)
	expect(t, rec, 400)
	if d := decode(t, rec).(map[string]any)["detail"].(map[string]any); d["error"] != "cross_ref_repair_source_sha_required" || d["context"] == nil {
		t.Fatalf("detail = %#v", d)
	}
	stale := map[string]any{}
	for k, v := range body {
		stale[k] = v
	}
	stale["source_sha256"] = strings.Repeat("0", 64)
	rec = do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/apply", stale, nil)
	expect(t, rec, 409)

	body["source_sha256"] = preview["source_sha256"]
	rec = do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/apply", body, nil)
	expect(t, rec, 200)
	applied := decode(t, rec).(map[string]any)
	if applied["applied"] != true || applied["reparse_ok"] != true || applied["remaining_target_count"] != float64(0) {
		t.Fatalf("apply = %#v", applied)
	}
	rows, _ := p.Rows("ideas")
	if rows[0]["on_add"] != "add_ideas = GER_war_economy\n" {
		t.Fatalf("on_add = %q", rows[0]["on_add"])
	}

	expect(t, do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/preview", "{bad", nil), 422)
	bad := map[string]any{}
	for k, v := range body {
		bad[k] = v
	}
	bad["entity_kind"] = "country"
	expect(t, do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/preview", bad, nil), 422)
	bad["entity_kind"], bad["entity_id"], bad["field"] = "event", 99, "immediate"
	rec = do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/preview", bad, nil)
	expect(t, rec, 404)
	bad["entity_kind"], bad["field"] = "idea", "name"
	rec = do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/preview", bad, nil)
	expect(t, rec, 400)
	expect(t, do(t, mux, "GET", "/api/go/projects/1/cross-ref-repairs/preview", nil, nil), 405)
	expect(t, do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/bogus", body, nil), 404)
}

func TestCrossRefRepairFocusRevision(t *testing.T) {
	p, mux := newProject(t)
	text := "completion_reward = {\n\tadd_ideas = GER_war_economi\n}\n"
	var treeID, nodeID int64
	if err := p.Write([]string{"focus_trees", "focus_nodes"}, func(tx *project.Tx) error {
		treeID = project.Int64(tx.Insert("focus_trees", project.Row{"tree_id": "t", "edit_revision": int64(3)})["id"])
		nodeID = project.Int64(tx.Insert("focus_nodes", project.Row{"focus_tree_id": treeID, "focus_id": "f", "completion_reward": text})["id"])
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cand := CollectPositionedReferenceCandidates(text, nil)[0]
	body := map[string]any{
		"entity_kind": "focus", "entity_id": nodeID, "field": "completion_reward", "target_id": cand.TargetID,
		"target_start_offset": cand.SourceSpan.StartOffset, "target_end_offset": cand.SourceSpan.EndOffset,
		"delete_assignment": true,
	}
	rec := do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/preview", body, nil)
	expect(t, rec, 200)
	preview := decode(t, rec).(map[string]any)
	key := fmt.Sprintf("focus_tree:%d", treeID)
	if preview["edit_revisions"].(map[string]any)[key] != "3" {
		t.Fatalf("edit_revisions = %#v", preview["edit_revisions"])
	}
	body["source_sha256"] = preview["source_sha256"]

	rec = do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/apply", body,
		map[string]string{"X-Edit-Revisions": fmt.Sprintf(`{%q:"2"}`, key)})
	expect(t, rec, 409)
	if decode(t, rec).(map[string]any)["detail"].(map[string]any)["code"] != "save_conflict" {
		t.Fatalf("conflict = %s", rec.Body.String())
	}
	expect(t, do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/apply", body,
		map[string]string{"X-Edit-Revisions": "{bad"}), 422)

	rec = do(t, mux, "POST", "/api/go/projects/1/cross-ref-repairs/apply", body,
		map[string]string{"X-Edit-Revisions": fmt.Sprintf(`{%q:"3"}`, key)})
	expect(t, rec, 200)
	if got := rec.Header().Get("X-Edit-Revisions"); got != fmt.Sprintf(`{%q:"4"}`, key) {
		t.Fatalf("revision header = %q", got)
	}
	nodes, _ := p.Rows("focus_nodes")
	if strings.Contains(nodes[0]["completion_reward"].(string), "GER_war_economi") {
		t.Fatalf("reward = %q", nodes[0]["completion_reward"])
	}
}

func TestImportCharacters(t *testing.T) {
	p, mux := newProject(t)
	script := []byte(`characters = {
 GER_demo = {
  name = GER_demo_name
  corps_commander = { skill = 4 attack_skill = 3 }
 }
 OTHER = { name = other advisor = { slot = political_advisor cost = 90 } }
}`)
	rec := postMultipart(t, mux, "/api/go/projects/1/import/characters", nil, script)
	expect(t, rec, 200)
	result := decode(t, rec).(map[string]any)
	if result["created"] != float64(2) || result["updated"] != float64(0) || result["conflict_policy"] != "overwrite" {
		t.Fatalf("result = %#v", result)
	}
	rows, _ := p.Rows("characters")
	if len(rows) != 2 {
		t.Fatalf("rows = %d", len(rows))
	}
	byTag := map[string]project.Row{}
	for _, row := range rows {
		byTag[row["tag"].(string)] = row
	}
	ger := byTag["GER_demo"]
	if ger["country_tag"] != "GER" || project.Int64(ger["skill"]) != 4 || ger["source"] != "created" ||
		ger["replaces_vanilla_id"] != true || ger["source_snapshot"] == nil {
		t.Fatalf("GER row = %#v", ger)
	}
	if byTag["OTHER"]["country_tag"] != "TST" || project.Int64(byTag["OTHER"]["advisor_cost"]) != 90 {
		t.Fatalf("OTHER row = %#v", byTag["OTHER"])
	}

	rec = postMultipart(t, mux, "/api/go/projects/1/import/characters?country_tag=FRA",
		map[string]string{"conflict_policy": "skip"}, script)
	expect(t, rec, 200)
	if r := decode(t, rec).(map[string]any); r["skipped"] != float64(2) || r["created"] != float64(0) {
		t.Fatalf("skip = %#v", r)
	}
	rec = postMultipart(t, mux, "/api/go/projects/1/import/characters", nil, script)
	expect(t, rec, 200)
	if r := decode(t, rec).(map[string]any); r["updated"] != float64(2) {
		t.Fatalf("overwrite = %#v", r)
	}
	if rows, _ := p.Rows("characters"); len(rows) != 2 {
		t.Fatalf("rows after overwrite = %d", len(rows))
	}

	expect(t, postMultipart(t, mux, "/api/go/projects/1/import/characters", map[string]string{"conflict_policy": "merge"}, script), 400)
	expect(t, postMultipart(t, mux, "/api/go/projects/1/import/characters", nil, nil), 422)
	expect(t, postMultipart(t, mux, "/api/go/projects/1/import/characters", nil, []byte("characters = { X = { name = X")), 422)
	rec = postMultipart(t, mux, "/api/go/projects/1/import/characters", nil, []byte("# nothing\n"))
	expect(t, rec, 200)
	if r := decode(t, rec).(map[string]any); r["created"] != float64(0) || len(r["warnings"].([]any)) != 1 {
		t.Fatalf("empty file = %#v", r)
	}
}

func TestImportTechnologies(t *testing.T) {
	p, mux := newProject(t)
	script := []byte(`@1936 = 2
technologies = {
 tech_a = {
  research_cost = 1.5
  start_year = 1936
  folder = { name = infantry_folder position = { x = 2 y = @1936 } }
  categories = { infantry_weapons }
 }
}`)
	expect(t, postMultipart(t, mux, "/api/go/projects/1/import/technologies", nil, script), 422)
	rec := postMultipart(t, mux, "/api/go/projects/1/import/technologies?tech_line=infantry&folder_name=infantry_folder", nil, script)
	expect(t, rec, 200)
	if r := decode(t, rec).(map[string]any); r["created"] != float64(1) || r["total"] != float64(1) {
		t.Fatalf("result = %s", rec.Body.String())
	}
	rows, _ := p.Rows("technologies")
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	tech := rows[0]
	if tech["tech_id"] != "tech_a" || tech["tech_line"] != "infantry" || tech["source"] != "created" ||
		tech["research_cost"] != 1.5 || project.Int64(tech["start_year"]) != 1936 ||
		tech["position_y_is_explicit"] != true || tech["country_tag"] != "" {
		t.Fatalf("tech row = %#v", tech)
	}
	years, _ := p.Record()["tech_year_columns"].([]any)
	if len(years) != 1 {
		t.Fatalf("tech_year_columns = %#v", p.Record()["tech_year_columns"])
	}
	pair := years[0].([]any)
	if pair[0] != "@1936" || project.Int64(pair[1]) != 2 {
		t.Fatalf("year pair = %#v", pair)
	}
}
