package countries

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

func TestCosmeticTags(t *testing.T) {
	p := newProject(t)
	seedTwoCountries(t, p)
	seed(t, p, "project_loc_entries",
		project.Row{"locale": "l_english", "key": "TST_FAS", "value": "Old", "source_file": "localisation/english/a_l_english.yml", "replace_priority": 0},
		project.Row{"locale": "l_english", "key": "TST_FAS", "value": "Imported Reich", "source_file": "localisation/english/b_l_english.yml", "replace_priority": 1},
		project.Row{"locale": "l_english", "key": "TST_FAS_ADJ", "value": "Reichish", "source_file": "localisation/english/b_l_english.yml"},
		project.Row{"locale": "l_english", "key": "TST_FAS_DEF", "value": "x", "source_file": "../escape.yml"},
	)
	seed(t, p, "history_files", project.Row{"country_tag": "TST", "set_cosmetic_tag": "TST_FAS"})
	seed(t, p, "project_raw_files", project.Row{"dest_path": "localisation/english/raw_l_english.yml",
		"content": "l_english:\n TST_FAS_DEF:0 \"the Raw Reich\"\n"})
	mux := newMux(p)

	expectStatus(t, call(mux, "GET", "/api/go/projects/1/cosmetic-tags/TST_FAS", ""), http.StatusNotFound)
	r := call(mux, "POST", "/api/go/projects/1/cosmetic-tags", `{"tag":"TST_FAS"}`)
	expectStatus(t, r, http.StatusCreated)
	body := r.json(t)
	if body["revision"] != 2.0 || body["source"] != "created" ||
		asJSON(body["imported_name_locs"]) != `{"l_english":{"":"Imported Reich","ADJ":"Reichish"}}` ||
		asJSON(body["referenced_by"]) != `["TST"]` || asJSON(body["name_locs"]) != `{}` {
		t.Fatalf("created = %s", r.Body.String())
	}
	if asJSON(body["raw_overrides"]) != `[{"file_id":1,"locale":"l_english","path":"localisation/english/raw_l_english.yml","suffix":"DEF","value":"the Raw Reich"}]` {
		t.Fatalf("raw_overrides = %v", asJSON(body["raw_overrides"]))
	}

	r = call(mux, "POST", "/api/go/projects/1/cosmetic-tags", `{"tag":"TST_FAS"}`)
	expectStatus(t, r, http.StatusCreated)
	if r.json(t)["revision"] != 2.0 {
		t.Fatal("re-create changed the tag")
	}
	expectStatus(t, call(mux, "POST", "/api/go/projects/1/cosmetic-tags", `{"tag":"bad tag"}`), http.StatusUnprocessableEntity)
	expectStatus(t, call(mux, "POST", "/api/go/projects/1/cosmetic-tags", `{"tag":"X","extra":1}`), http.StatusUnprocessableEntity)

	r = call(mux, "PATCH", "/api/go/projects/1/cosmetic-tags/TST_FAS", `{"expected_revision":1,"name_ops":[]}`)
	expectStatus(t, r, http.StatusConflict)
	r = call(mux, "PATCH", "/api/go/projects/1/cosmetic-tags/TST_FAS",
		`{"expected_revision":2,"name_ops":[{"locale":"l_english","suffix":"","value":"Our Reich"}],"color":[10,20,30]}`)
	expectStatus(t, r, http.StatusOK)
	body = r.json(t)
	if body["revision"] != 3.0 || asJSON(body["name_locs"]) != `{"l_english":{"":"Our Reich"}}` || asJSON(body["color"]) != "[10,20,30]" {
		t.Fatalf("patched = %s", r.Body.String())
	}
	r = call(mux, "PATCH", "/api/go/projects/1/cosmetic-tags/TST_FAS",
		`{"expected_revision":3,"name_ops":[{"locale":"l_english","suffix":"","value":null}],"color":null}`)
	expectStatus(t, r, http.StatusOK)
	if body := r.json(t); asJSON(body["name_locs"]) != `{}` || body["color"] != nil || body["revision"] != 4.0 {
		t.Fatalf("cleared = %s", r.Body.String())
	}
	for _, bad := range []string{
		`{"expected_revision":0,"name_ops":[]}`,
		`{"expected_revision":4,"name_ops":[{"locale":"l_klingon","suffix":"","value":"x"}]}`,
		`{"expected_revision":4,"name_ops":[{"locale":"l_english","suffix":""}]}`,
		`{"expected_revision":4,"name_ops":[],"color":[1,2]}`,
	} {
		expectStatus(t, call(mux, "PATCH", "/api/go/projects/1/cosmetic-tags/TST_FAS", bad), http.StatusUnprocessableEntity)
	}
	expectStatus(t, call(mux, "PATCH", "/api/go/projects/1/cosmetic-tags/NOPE", `{"expected_revision":1,"name_ops":[]}`), http.StatusNotFound)

	r = call(mux, "GET", "/api/go/projects/1/cosmetic-tags", "")
	expectStatus(t, r, http.StatusOK)
	if body := r.json(t); body["total"] != 1.0 {
		t.Fatalf("list = %s", r.Body.String())
	}

	r = call(mux, "POST", "/api/go/projects/1/cosmetic-tags/TST_FAS/resolve",
		`{"locale":"l_english","name_locs":{"l_english":{"ADJ":"Draftish"}},"color":null,"color_ui":null}`)
	expectStatus(t, r, http.StatusOK)
	body = r.json(t)
	names := body["names"].(map[string]any)
	if names[""].(map[string]any)["value"] != "Imported Reich" || names["ADJ"].(map[string]any)["value"] != "Draftish" {
		t.Fatalf("resolved names = %v", names)
	}
	if asJSON(body["color"]) != "[1,2,3]" {
		t.Fatalf("owner colour should come from the owning country: %v", body["color"])
	}
	expectStatus(t, call(mux, "POST", "/api/go/projects/1/cosmetic-tags/TST_FAS/resolve", `{"locale":"xx"}`), http.StatusUnprocessableEntity)
}

func TestRelations(t *testing.T) {
	p := newProject(t)
	seedTwoCountries(t, p)
	ideas := seed(t, p, "ideas",
		project.Row{"idea_id": "TST_idea_a", "country_tag": "TST", "on_add": "add_ideas = { TST_idea_b missing_idea } country_event = { id = tst.1 }"},
		project.Row{"idea_id": "TST_idea_b", "country_tag": "TST"},
	)
	trees := seed(t, p, "focus_trees", project.Row{"tree_id": "t", "country_tag": "OTH"})
	seed(t, p, "focus_nodes", project.Row{"focus_tree_id": trees[0], "focus_id": "OTH_focus", "completion_reward": "add_ideas = TST_idea_a has_completed_focus = OTH_focus"})
	mux := newMux(p)

	r := call(mux, "GET", fmt.Sprintf("/api/go/projects/1/relations/entity/idea/%d", ideas[0]), "")
	expectStatus(t, r, http.StatusOK)
	body := r.json(t)
	if body["outgoing_total"] != 3.0 || body["incoming_total"] != 1.0 {
		t.Fatalf("entity relations = %s", r.Body.String())
	}
	out := body["outgoing"].([]any)
	first := out[0].(map[string]any)
	if first["target_id"] != "TST_idea_b" || first["resolved"] != true || first["target_tag"] != "TST" || first["source_tag"] != "TST" || first["field_name"] != "on_add" {
		t.Fatalf("first edge = %v", first)
	}
	if out[1].(map[string]any)["resolved"] != false || out[2].(map[string]any)["target_type"] != "event" {
		t.Fatalf("edges = %v", out)
	}

	r = call(mux, "GET", fmt.Sprintf("/api/go/projects/1/relations/entity/idea/%d", ideas[1]), "")
	if body := r.json(t); body["outgoing_total"] != 0.0 || body["incoming_total"] != 1.0 {
		t.Fatalf("idea b = %s", r.Body.String())
	}
	expectStatus(t, call(mux, "GET", "/api/go/projects/1/relations/entity/idea/x", ""), http.StatusUnprocessableEntity)

	r = call(mux, "GET", "/api/go/projects/1/relations/country/tst", "")
	expectStatus(t, r, http.StatusOK)
	if body := r.json(t); body["outgoing_total"] != 3.0 || body["incoming_total"] != 2.0 {
		t.Fatalf("country relations = %s", r.Body.String())
	}
	r = call(mux, "GET", "/api/go/projects/1/relations/graph", "")
	if body := r.json(t); body["total"] != 5.0 {
		t.Fatalf("graph = %s", r.Body.String())
	}
}

func TestPoliticsLifecycle(t *testing.T) {
	p := newProject(t)
	seedTwoCountries(t, p)
	mux := newMux(p)

	r := call(mux, "GET", "/api/go/projects/1/politics/references", "")
	expectStatus(t, r, http.StatusOK)
	body := r.json(t)
	if body["project_id"] != "u1" || len(body["ideologies"].([]any)) != 4 {
		t.Fatalf("references = %s", r.Body.String())
	}

	r = call(mux, "POST", "/api/go/projects/1/politics/roots",
		`{"script_id":"technocracy","body":"types = { technocrat = { } }\ncolor = { 1 2 3 }","name":{"en":"Technocracy"}}`)
	expectStatus(t, r, http.StatusCreated)
	if body := r.json(t); body["kind"] != "root" || body["source"] != "project_managed" || asJSON(body["desc"]) != "{}" {
		t.Fatalf("root = %s", r.Body.String())
	}
	registry := p.Record()["ideology_registry"].(map[string]any)
	if asJSON(registry["groups"]) != `["technocracy"]` {
		t.Fatalf("registry = %v", asJSON(registry))
	}
	r = call(mux, "POST", "/api/go/projects/1/politics/roots", `{"script_id":"technocracy","body":"types = { }\ncolor = { 1 2 3 }","name":{"en":"Again"}}`)
	expectStatus(t, r, http.StatusConflict)
	if body := r.json(t); body["code"] != "politics_script_id_conflict" {
		t.Fatalf("conflict = %s", r.Body.String())
	}
	expectStatus(t, call(mux, "POST", "/api/go/projects/1/politics/roots", `{"script_id":"x","bogus":1}`), http.StatusUnprocessableEntity)
	r = call(mux, "POST", "/api/go/projects/1/politics/roots", `{"script_id":"democratic","body":"types = { }\ncolor = { 1 2 3 }","name":{"en":"D"}}`)
	expectStatus(t, r, http.StatusUnprocessableEntity)

	r = call(mux, "POST", "/api/go/projects/1/politics/subtypes",
		`{"script_id":"social_technocracy","root_id":"technocracy","name":{"en":"Social Technocracy"},"random_disabled":true}`)
	expectStatus(t, r, http.StatusCreated)
	if body := r.json(t); body["root_id"] != "technocracy" || body["random_disabled"] != true {
		t.Fatalf("subtype = %s", r.Body.String())
	}
	r = call(mux, "POST", "/api/go/projects/1/politics/subtypes", `{"script_id":"x_sub","root_id":"nowhere","name":{"en":"X"},"random_disabled":false}`)
	expectStatus(t, r, http.StatusNotFound)

	r = call(mux, "POST", "/api/go/projects/1/politics/subtypes",
		`{"script_id":"civic_reformism","root_id":"democratic","name":{"en":"Civic Reformism"},"random_disabled":false}`)
	expectStatus(t, r, http.StatusCreated)
	manifest, _ := p.Record()["ideology_source_manifest"].(map[string]any)
	if manifest["state"] != "ready" || manifest["source_key"] != "vanilla" || len(manifest["entries"].([]any)) != 1 {
		t.Fatalf("manifest = %v", asJSON(manifest))
	}

	r = call(mux, "PATCH", "/api/go/projects/1/politics/definitions/subtype/social_technocracy", `{"name":{"en":"Renamed"}}`)
	expectStatus(t, r, http.StatusOK)
	if body := r.json(t); asJSON(body["name"]) != `{"en":"Renamed"}` {
		t.Fatalf("patch = %s", r.Body.String())
	}
	r = call(mux, "PATCH", "/api/go/projects/1/politics/definitions/subtype/social_technocracy", `{"root_id":"democratic"}`)
	expectStatus(t, r, http.StatusConflict)
	r = call(mux, "PATCH", "/api/go/projects/1/politics/definitions/root/democratic", `{"name":{"en":"x"}}`)
	expectStatus(t, r, http.StatusConflict)
	if body := r.json(t); body["code"] != "politics_source_owned_read_only" {
		t.Fatalf("immutable = %s", r.Body.String())
	}
	expectStatus(t, call(mux, "PATCH", "/api/go/projects/1/politics/definitions/other/x", `{}`), http.StatusNotFound)

	if err := p.Write([]string{"countries"}, func(tx *project.Tx) error {
		_, err := tx.Update("countries", 1, func(row project.Row) { row["ideology"] = "technocracy" })
		return err
	}); err != nil {
		t.Fatal(err)
	}
	r = call(mux, "POST", "/api/go/projects/1/politics/delete-preview", `{"kind":"root","script_id":"technocracy"}`)
	expectStatus(t, r, http.StatusOK)
	preview := r.json(t)
	if preview["allowed"] != false || len(preview["impacts"].([]any)) != 3 {
		t.Fatalf("preview = %s", r.Body.String())
	}
	r = call(mux, "POST", "/api/go/projects/1/politics/delete-apply", `{"preview_token":"`+preview["token"].(string)+`"}`)
	expectStatus(t, r, http.StatusConflict)
	if body := r.json(t); body["code"] != "politics_delete_blocked" || body["authorization"] == nil {
		t.Fatalf("blocked = %s", r.Body.String())
	}

	r = call(mux, "POST", "/api/go/projects/1/politics/delete-preview", `{"kind":"subtype","script_id":"civic_reformism"}`)
	expectStatus(t, r, http.StatusOK)
	token := r.json(t)["token"].(string)
	r = call(mux, "POST", "/api/go/projects/1/politics/delete-apply", `{"preview_token":"`+token+`"}`)
	expectStatus(t, r, http.StatusOK)
	if body := r.json(t); body["authorized"] != true {
		t.Fatalf("apply = %s", r.Body.String())
	}
	if manifest, _ := p.Record()["ideology_source_manifest"].(map[string]any); len(manifest["entries"].([]any)) != 0 || manifest["state"] != nil {
		t.Fatalf("manifest after delete = %v", asJSON(manifest))
	}
	r = call(mux, "POST", "/api/go/projects/1/politics/delete-apply", `{"preview_token":"`+token+`"}`)
	expectStatus(t, r, http.StatusConflict)
	if body := r.json(t); body["code"] != "politics_delete_preview_unknown" {
		t.Fatalf("reuse = %s", r.Body.String())
	}
	expectStatus(t, call(mux, "POST", "/api/go/projects/1/politics/delete-apply", `{"preview_token":" "}`), http.StatusUnprocessableEntity)
	expectStatus(t, call(mux, "POST", "/api/go/projects/1/politics/delete-preview", `{"kind":"root","script_id":"nope"}`), http.StatusNotFound)
}

func TestPoliticsConflictsAreSeparatedAndDiscardable(t *testing.T) {
	p := newProject(t)
	if _, err := p.UpdateRecord(func(r project.Row) {
		r["ideology_registry"] = map[string]any{"sub_ideologies": []any{
			map[string]any{"id": "democratic", "group": ""},
			map[string]any{"id": "orphan", "group": ""},
			map[string]any{"id": "kept", "group": "fascism"},
		}}
	}); err != nil {
		t.Fatal(err)
	}
	seed(t, p, "characters", project.Row{"tag": "c1", "ideology": "orphan"})
	mux := newMux(p)
	r := call(mux, "GET", "/api/go/projects/1/politics/references", "")
	expectStatus(t, r, http.StatusOK)
	body := r.json(t)
	conflicts := body["conflicts"].([]any)
	if len(conflicts) != 2 || !strings.Contains(r.Body.String(), `"reason":"missing_root"`) || !strings.Contains(r.Body.String(), `"usage_count":1`) {
		t.Fatalf("conflicts = %s", r.Body.String())
	}
	expectStatus(t, call(mux, "DELETE", "/api/go/projects/1/politics/conflicts/kept", ""), http.StatusNotFound)
	r = call(mux, "DELETE", "/api/go/projects/1/politics/conflicts/orphan", "")
	expectStatus(t, r, http.StatusOK)
	if body := r.json(t); body["script_id"] != "orphan" || body["usage_count"] != 1.0 {
		t.Fatalf("discarded = %s", r.Body.String())
	}
	registry := p.Record()["ideology_registry"].(map[string]any)
	if strings.Contains(asJSON(registry), "orphan") || !strings.Contains(asJSON(registry), "kept") {
		t.Fatalf("registry = %s", asJSON(registry))
	}
}

func importIdeologySnapshot(t *testing.T, p *project.Project, files map[string]string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	var provenanceFiles []any
	for path, content := range files {
		w, _ := zw.Create(path)
		_, _ = w.Write([]byte(content))
		sum := sha256.Sum256([]byte(content))
		provenanceFiles = append(provenanceFiles, map[string]any{"path": path, "sha256": hex.EncodeToString(sum[:])})
	}
	_ = zw.Close()
	sum := sha256.Sum256(buf.Bytes())
	blobs := p.Blobs()
	key, err := importassets.IdeologySourceSnapshotKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "imported", hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	if err := blobs.Put(key, "application/zip", int64(buf.Len()), bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	seed(t, p, "project_obs_assets", project.Row{"asset_type": "ideology_source", "gfx_name": "imported", "obs_key": key,
		"provenance": map[string]any{"schema_version": 1, "source_key": "imported", "descriptor_replace_paths": []any{}, "files": provenanceFiles}})
}

func TestImportedIdeologiesSeedTheRegistry(t *testing.T) {
	p := newProject(t)
	source := "ideologies = {\n\ttechnocracy = {\n\t\ttypes = {\n\t\t\tcivic_technocracy = { }\n\t\t}\n\t\tcolor = { 42 110 145 }\n\t}\n}\n"
	importIdeologySnapshot(t, p, map[string]string{"common/ideologies/zz_mod.txt": source})
	seed(t, p, "project_raw_files", project.Row{"dest_path": "common/ideologies/zz_mod.txt", "content": source, "source": "imported"})
	mux := newMux(p)
	r := call(mux, "GET", "/api/go/projects/1/politics/references", "")
	expectStatus(t, r, http.StatusOK)
	if !strings.Contains(r.Body.String(), `{"kind":"root","script_id":"technocracy","display_name":"technocracy","source":"imported"}`) ||
		!strings.Contains(r.Body.String(), `"script_id":"civic_technocracy","root_id":"technocracy"`) {
		t.Fatalf("references = %s", r.Body.String())
	}
	registry := p.Record()["ideology_registry"].(map[string]any)
	if asJSON(registry["groups"]) != `["technocracy"]` || !strings.Contains(asJSON(registry["sub_ideologies"]), `"source":"imported"`) {
		t.Fatalf("registry = %s", asJSON(registry))
	}

	r = call(mux, "PATCH", "/api/go/projects/1/politics/definitions/root/technocracy", `{"name":{"en":"x"}}`)
	expectStatus(t, r, http.StatusConflict)
	r = call(mux, "POST", "/api/go/projects/1/politics/subtypes",
		`{"script_id":"green_technocracy","root_id":"technocracy","name":{"en":"Green"},"random_disabled":false}`)

	expectStatus(t, r, http.StatusConflict)
	if body := r.json(t); body["code"] != "politics_source_raw_owned" {
		t.Fatalf("subtype = %s", r.Body.String())
	}
}

func TestSubtypeUnderImportedRootIsSourceLocked(t *testing.T) {
	p := newProject(t)
	source := "ideologies = {\n\ttechnocracy = {\n\t\ttypes = {\n\t\t\tcivic_technocracy = { }\n\t\t}\n\t\tcolor = { 42 110 145 }\n\t}\n}\n"
	importIdeologySnapshot(t, p, map[string]string{"common/ideologies/zz_mod.txt": source})
	mux := newMux(p)
	r := call(mux, "POST", "/api/go/projects/1/politics/subtypes",
		`{"script_id":"green_technocracy","root_id":"technocracy","name":{"en":"Green"},"random_disabled":false}`)
	expectStatus(t, r, http.StatusCreated)
	manifest, _ := p.Record()["ideology_source_manifest"].(map[string]any)
	entries, _ := manifest["entries"].([]any)
	if manifest["source_key"] != "imported" || len(entries) != 1 ||
		entries[0].(map[string]any)["owner_path"] != "common/ideologies/zz_mod.txt" {
		t.Fatalf("manifest = %s", asJSON(manifest))
	}

	if err := p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		row := tx.Rows("project_obs_assets")[0]
		_, err := tx.Update("project_obs_assets", project.Int64(row["id"]), func(r project.Row) {
			prov := r["provenance"].(map[string]any)
			files := prov["files"].([]any)
			files[0] = map[string]any{"path": "common/ideologies/zz_mod.txt", "sha256": strings.Repeat("0", 64)}
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	r = call(mux, "POST", "/api/go/projects/1/politics/subtypes",
		`{"script_id":"blue_technocracy","root_id":"technocracy","name":{"en":"Blue"},"random_disabled":false}`)
	expectStatus(t, r, http.StatusConflict)
	if body := r.json(t); body["code"] != "politics_source_archive_drift" {
		t.Fatalf("drift = %s", r.Body.String())
	}
}
