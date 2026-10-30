package api

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/domains/tools/exportcheck"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/exportjob"
	"haganeworks/internal/project"
	"haganeworks/internal/settings"
)

type advancedFixture struct {
	t        *testing.T
	projects *projects
	mux      *http.ServeMux
	id       int64
}

func newAdvancedFixture(t *testing.T, name string, issues []exportjob.ExportIssue) *advancedFixture {
	t.Helper()
	a := newTestAPI(t)
	var created map[string]any
	if status := a.call("POST", "/api/go/projects", map[string]any{"name": name, "country_tag": "TST", "country_name": "Testland"}, &created); status != http.StatusCreated {
		t.Fatalf("create: %d %v", status, created)
	}
	store, err := settings.Open(filepath.Join(filepath.Dir(a.dir), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	f := &advancedFixture{t: t, projects: &projects{settings: store}, mux: http.NewServeMux(), id: int64(created["id"].(float64))}
	collector := exportcheck.IssueCollector
	if issues != nil {
		collector = func(*project.Project) exportjob.IssueCollector { return issueFixture(issues) }
	}
	registerAdvancedExports(f.mux, f.projects, collector)
	return f
}

type issueFixture []exportjob.ExportIssue

func (f issueFixture) CollectExportIssues(context.Context, int64, exportjob.ExportIssuesOptions) ([]exportjob.ExportIssue, bool, error) {
	return f, true, nil
}

func (f *advancedFixture) addImportedDecision() {
	f.t.Helper()
	store, err := f.projects.current()
	if err != nil {
		f.t.Fatal(err)
	}
	p, err := store.Get(f.id)
	if err != nil {
		f.t.Fatal(err)
	}
	err = p.Write([]string{"decision_categories", "decisions"}, func(tx *project.Tx) error {
		category := tx.Insert("decision_categories", project.Row{"category_id": "TST_cat", "country_tag": "TST"})
		tx.Insert("decisions", project.Row{
			"category_id": category["id"], "decision_id": "TST_decide", "country_tag": "TST", "cost": 100,
			"source_snapshot": map[string]any{"decision_id": "TST_decide", "cost": 50},
		})
		return nil
	})
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *advancedFixture) post(route string, file []byte, fields map[string]string) *httptest.ResponseRecorder {
	f.t.Helper()
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, advancedRequest(f.t, "/api/go/projects/"+jsonNumber(f.id)+"/export/"+route, file, fields))
	return rec
}

func advancedRequest(t *testing.T, url string, file []byte, fields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if file != nil {
		part, err := writer.CreateFormFile("base_mod_file", "base.zip")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(file); err != nil {
			t.Fatal(err)
		}
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func zipBytes(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func readZIP(t *testing.T, body []byte) map[string][]byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[file.Name] = data
	}
	return files
}

func writeModFolder(t *testing.T, name string, files map[string][]byte) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	for rel, body := range files {
		target := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func folderState(t *testing.T, dir string) string {
	t.Helper()
	var out strings.Builder
	err := filepath.WalkDir(dir, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		body, err := os.ReadFile(name)
		fmt.Fprintf(&out, "%s %d %s %x\n", name, info.Size(), info.ModTime(), sha256.Sum256(body))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestAdvancedExportRoutesAreRegistered(t *testing.T) {
	a := newTestAPI(t)
	var created map[string]any
	a.call("POST", "/api/go/projects", map[string]any{"name": "Routes", "country_tag": "TST", "country_name": "Testland"}, &created)
	id := jsonNumber(int64(created["id"].(float64)))
	for _, route := range []string{"diff-preview", "diff-resolve", "diff-export-zip", "append-to-existing"} {
		for path, want := range map[string]int{
			"/api/go/projects/" + id + "/export/" + route: http.StatusUnprocessableEntity,
			"/api/go/projects/999/export/" + route:        http.StatusNotFound,
		} {
			rec := httptest.NewRecorder()
			a.mux.ServeHTTP(rec, advancedRequest(t, path, []byte("not-a-zip"), map[string]string{"acknowledge_blockers": "true"}))
			if rec.Code != want {
				t.Errorf("%s: %d %s, want %d", path, rec.Code, rec.Body.String(), want)
			}
		}
	}
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/go/projects/"+id+"/full-mod-files", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("full-mod-files: %d %.200s", rec.Code, rec.Body.String())
	}
}

func TestAdvancedUploadReturns422And413(t *testing.T) {
	f := newAdvancedFixture(t, "Upload", nil)
	if rec := f.post("diff-preview", []byte("not-a-zip"), nil); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid zip status=%d body=%s", rec.Code, rec.Body.String())
	}

	previous := advancedUploadLimit
	advancedUploadLimit = 8
	defer func() { advancedUploadLimit = previous }()
	if rec := f.post("diff-preview", bytes.Repeat([]byte("x"), 9), nil); rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdvancedExportRefusesWhatIsNotAModFolder(t *testing.T) {
	f := newAdvancedFixture(t, "Folders", nil)
	mod := writeModFolder(t, "Base", map[string][]byte{"descriptor.mod": []byte("name=\"Base Mod\"\n")})
	notAMod := writeModFolder(t, "Pictures", map[string][]byte{"a.png": []byte("png")})
	cases := map[string]struct {
		file   []byte
		folder string
	}{
		"relative path":         {folder: "Base"},
		"a file":                {folder: filepath.Join(mod, "descriptor.mod")},
		"missing folder":        {folder: filepath.Join(mod, "missing")},
		"folder without a mod":  {folder: notAMod},
		"both a zip and folder": {file: zipBytes(t, map[string][]byte{"descriptor.mod": []byte("name=\"Base\"\n")}), folder: mod},
		"neither":               {},
	}
	for name, c := range cases {
		fields := map[string]string{"acknowledge_blockers": "true"}
		if c.folder != "" {
			fields["base_mod_folder"] = c.folder
		}
		for _, route := range []string{"diff-preview", "append-to-existing"} {
			rec := f.post(route, c.file, fields)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("%s via %s: %d %s", name, route, rec.Code, rec.Body.String())
			}
		}
	}
	var refused struct {
		Detail struct {
			Error string `json:"error"`
		} `json:"detail"`
	}
	rec := f.post("diff-preview", nil, map[string]string{"base_mod_folder": notAMod})
	if err := json.Unmarshal(rec.Body.Bytes(), &refused); err != nil || refused.Detail.Error != "not_a_mod" {
		t.Fatalf("a folder without descriptor.mod: %s (%v)", rec.Body.String(), err)
	}
}

func TestDiffMergeFromAZIPOrAFolder(t *testing.T) {
	base := map[string][]byte{
		"descriptor.mod":           []byte("name=\"Base Mod\"\n"),
		"common/decisions/TST.txt": []byte("TST_cat = {\n\tTST_decide = {\n\t\tcost = 300\n\t}\n}\n"),
		"notes/creator.txt":        []byte("Untouched creator bytes\r\n"),
	}
	folder := writeModFolder(t, "Base", base)
	before := folderState(t, folder)
	for _, input := range []struct {
		name     string
		file     []byte
		fields   map[string]string
		filename string
	}{
		{"zip", zipBytes(t, base), map[string]string{}, "merged_base.zip"},
		{"folder", nil, map[string]string{"base_mod_folder": folder}, "merged_Base_Mod.zip"},
	} {
		t.Run(input.name, func(t *testing.T) {
			f := newAdvancedFixture(t, "Diff", nil)
			f.addImportedDecision()
			with := func(extra map[string]string) map[string]string {
				fields := map[string]string{}
				for key, value := range input.fields {
					fields[key] = value
				}
				for key, value := range extra {
					fields[key] = value
				}
				return fields
			}

			rec := f.post("diff-preview", input.file, with(nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("preview: %d %s", rec.Code, rec.Body.String())
			}
			var preview struct {
				Reports map[string][]struct {
					EntityID  string `json:"entity_id"`
					Conflicts []struct {
						Path string `json:"path"`
					} `json:"conflicts"`
				} `json:"reports"`
				Totals struct {
					Conflict int `json:"conflict"`
				} `json:"totals"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &preview); err != nil {
				t.Fatal(err)
			}
			decisions := preview.Reports["decision"]
			if preview.Totals.Conflict != 1 || len(decisions) != 1 || decisions[0].EntityID != "TST_decide" {
				t.Fatalf("preview = %s", rec.Body.String())
			}
			resolutions := map[string]string{}
			for _, conflict := range decisions[0].Conflicts {
				resolutions["decision/TST_decide/"+conflict.Path] = "keep_user"
			}
			if resolutions["decision/TST_decide/cost"] == "" {
				t.Fatalf("no cost conflict: %s", rec.Body.String())
			}
			resolved, _ := json.Marshal(map[string]any{"resolutions": resolutions})

			rec = f.post("diff-resolve", input.file, with(map[string]string{"resolutions_json": string(resolved)}))
			var final struct {
				Ops map[string][]map[string]any `json:"final_ops_by_entity"`
			}
			if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &final) != nil {
				t.Fatalf("resolve: %d %s", rec.Code, rec.Body.String())
			}
			costKept := false
			for _, op := range final.Ops["decision/TST_decide"] {
				costKept = costKept || op["path"] == "cost" && fmt.Sprint(op["value"]) == "100"
			}
			if !costKept {
				t.Fatalf("resolve did not keep the project's cost: %s", rec.Body.String())
			}
			if rec := f.post("diff-resolve", input.file, with(map[string]string{"resolutions_json": `{"resolutions":{"bad":"keep_user"}}`})); rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("malformed resolutions: %d", rec.Code)
			}

			rec = f.post("diff-export-zip", input.file, with(map[string]string{"resolutions_json": string(resolved), "acknowledge_blockers": "true"}))
			if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/zip" || rec.Header().Get("X-Merge-Entity-Count") != "1" {
				t.Fatalf("export: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
			}
			if _, params, err := mime.ParseMediaType(rec.Header().Get("Content-Disposition")); err != nil || params["filename"] != input.filename {
				t.Fatalf("content disposition %q (%v), want %s", rec.Header().Get("Content-Disposition"), err, input.filename)
			}
			files := readZIP(t, rec.Body.Bytes())
			if !strings.Contains(string(files["common/decisions/TST.txt"]), "cost = 100") {
				t.Fatalf("merged decision file:\n%s", files["common/decisions/TST.txt"])
			}
			for _, name := range []string{"descriptor.mod", "notes/creator.txt"} {
				if !bytes.Equal(files[name], base[name]) {
					t.Fatalf("%s changed: %q", name, files[name])
				}
			}
			var manifest []byte
			for name, body := range files {
				if filepath.Base(name) == exportengine.ArtifactManifestBasename {
					manifest = body
				}
			}
			if sum := fmt.Sprintf("%x", sha256.Sum256(manifest)); manifest == nil || rec.Header().Get("X-Artifact-Manifest-SHA256") != sum {
				t.Fatalf("manifest header %q, manifest %d bytes", rec.Header().Get("X-Artifact-Manifest-SHA256"), len(manifest))
			}
		})
	}
	if after := folderState(t, folder); after != before {
		t.Fatalf("the mod folder changed:\n%s\nbecame\n%s", before, after)
	}
}

func TestAppendToExistingFromAZIPOrAFolder(t *testing.T) {
	base := map[string][]byte{
		"descriptor.mod":            []byte("version=\"2\"\nname=\"Base Mod\"\ntags={ \"Base\" }\n"),
		"common/decisions/base.txt": []byte("base_cat = { base_decide = { cost = 5 } }\n"),
		"notes/creator.txt":         []byte("Untouched creator bytes\r\n"),
	}
	zipped := map[string][]byte{}
	for name, body := range base {
		zipped["Base/"+name] = body
	}
	folder := writeModFolder(t, "Base", base)
	if err := os.MkdirAll(filepath.Join(folder, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, ".git", "config"), []byte("[core]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := folderState(t, folder)
	for _, input := range []struct {
		name   string
		file   []byte
		fields map[string]string
	}{
		{"zip", zipBytes(t, zipped), map[string]string{"base_mod_filename": "原始.zip"}},
		{"folder", nil, map[string]string{"base_mod_folder": folder}},
	} {
		t.Run(input.name, func(t *testing.T) {
			f := newAdvancedFixture(t, "追加模组", nil)
			fields := map[string]string{"include_imported": "false", "build_identity": "false", "acknowledge_blockers": "true", "locale": "zh"}
			for key, value := range input.fields {
				fields[key] = value
			}
			rec := f.post("append-to-existing", input.file, fields)
			if rec.Code != http.StatusOK || rec.Header().Get("X-Merge-Conflict-Count") == "" {
				t.Fatalf("append: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
			}
			if _, params, err := mime.ParseMediaType(rec.Header().Get("Content-Disposition")); err != nil || params["filename"] != "追加模组_appended.zip" {
				t.Fatalf("content disposition %q (%v)", rec.Header().Get("Content-Disposition"), err)
			}
			files := readZIP(t, rec.Body.Bytes())
			for name := range files {
				if !strings.HasPrefix(name, "Base/") || strings.HasPrefix(name, "Base/.git/") {
					t.Errorf("entry %q is not part of the base mod's folder", name)
				}
			}
			for _, name := range []string{"common/decisions/base.txt", "notes/creator.txt"} {
				if !bytes.Equal(files["Base/"+name], base[name]) {
					t.Errorf("base file %s changed: %q", name, files["Base/"+name])
				}
			}
			descriptor := string(files["Base/descriptor.mod"])
			if !strings.Contains(descriptor, `name="Base Mod"`) || !strings.Contains(descriptor, `version="2"`) {
				t.Errorf("descriptor:\n%s", descriptor)
			}
			added := 0
			for name := range files {
				if _, isBase := base[strings.TrimPrefix(name, "Base/")]; !isBase {
					added++
				}
			}
			if added < 2 {
				t.Errorf("the project added %d files: %v", added, files)
			}
			var embedded exportengine.Manifest
			if err := json.Unmarshal(files["Base/"+exportengine.ArtifactManifestBasename], &embedded); err != nil || embedded.Timeline == nil || embedded.Timeline.TimelinePlanSHA256 == "" {
				t.Errorf("appended artifact lost its timeline identity: %+v (%v)", embedded.Timeline, err)
			}
		})
	}
	if after := folderState(t, folder); after != before {
		t.Fatalf("the mod folder changed:\n%s\nbecame\n%s", before, after)
	}
}

func TestFullModFiles(t *testing.T) {
	f := newAdvancedFixture(t, "Full Mod", nil)
	get := func(id, query string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		f.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/go/projects/"+id+"/full-mod-files"+query, nil))
		return rec
	}
	rec := get(jsonNumber(f.id), "")
	var body struct {
		ModName         string           `json:"mod_name"`
		PrimaryTag      string           `json:"primary_tag"`
		TotalFiles      int              `json:"total_files"`
		IncludeImported bool             `json:"include_imported"`
		Files           []map[string]any `json:"files"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &body) != nil {
		t.Fatalf("full-mod-files: %d %s", rec.Code, rec.Body.String())
	}
	if body.ModName != "Full Mod" || body.PrimaryTag != "TST" || !body.IncludeImported || body.TotalFiles == 0 || body.TotalFiles != len(body.Files) {
		t.Fatalf("contract mismatch: %s %s %v %d/%d", body.ModName, body.PrimaryTag, body.IncludeImported, body.TotalFiles, len(body.Files))
	}
	manifest, localisation := false, false
	for _, file := range body.Files {
		path, _ := file["path"].(string)
		content, _ := file["content"].(string)
		manifest = manifest || filepath.Base(path) == exportengine.ArtifactManifestBasename
		if strings.HasSuffix(path, ".yml") {
			localisation = true
			if file["is_binary"] != false || !strings.HasPrefix(content, "l_") || fmt.Sprint(file["size"]) != fmt.Sprint(len(content)+3) {
				t.Errorf("%s: binary=%v size=%v content %.40q", path, file["is_binary"], file["size"], content)
			}
		}
	}
	if !manifest || !localisation {
		t.Fatalf("manifest %v, localisation %v in %d files", manifest, localisation, len(body.Files))
	}

	if rec := get(jsonNumber(f.id), "?include_imported=false&locale=zh"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"include_imported":false`) {
		t.Errorf("include_imported=false: %d %.200s", rec.Code, rec.Body.String())
	}
	if rec := get(jsonNumber(f.id), "?include_imported=maybe"); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("include_imported=maybe: %d", rec.Code)
	}
	if rec := get("999", ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown project: %d", rec.Code)
	}
}

func TestAdvancedExportGate(t *testing.T) {
	base := zipBytes(t, map[string][]byte{"Base/descriptor.mod": []byte("name=\"Base\"\n")})
	for _, c := range []struct {
		code        string
		acknowledge string
		want        int
	}{
		{"test_blocker", "false", http.StatusConflict},
		{"test_blocker", "true", http.StatusOK},
		{"raw_binary_bytes_unavailable", "true", http.StatusConflict},
	} {
		f := newAdvancedFixture(t, "Gate", []exportjob.ExportIssue{{"severity": "blocker", "code": c.code, "message": "unsafe"}})
		for _, route := range []string{"diff-export-zip", "append-to-existing"} {
			rec := f.post(route, base, map[string]string{"acknowledge_blockers": c.acknowledge})
			if rec.Code != c.want {
				t.Errorf("%s %s acknowledged=%s: %d %s", route, c.code, c.acknowledge, rec.Code, rec.Body.String())
			}
			if c.want == http.StatusConflict && !bytes.Contains(rec.Body.Bytes(), []byte("ctd_blockers")) {
				t.Errorf("%s: 409 body %s", route, rec.Body.String())
			}
		}
	}
}

func TestAdvancedExportQualityFindingsRequireOneExplicitAcknowledgement(t *testing.T) {
	ordinaryCodes := []string{
		"map_integrity.final_overlay_unverified",
		"descriptor.replace_path_empty",
		"final_artifact.localisation_invalid",
		"final_artifact.glyph_missing",
		"content_language.export_languages_missing",
		"cross_ref.focus_prerequisite_missing",
		"preflight.script_parse_failed",
		"bookmark.raw_path_case_collision",
		"country.history_final_winner_ambiguous",
		"artifact.write_target_ambiguous",
	}

	for _, code := range ordinaryCodes {
		t.Run(code, func(t *testing.T) {
			issue := exportjob.ExportIssue{
				"severity": "blocker", "code": code,
				"message": "会影响玩家拿到的最终 Mod，请检查后决定是否继续。",
			}
			collector := issueFixture{issue}
			req := httptest.NewRequest(http.MethodPost, "/projects/7/export", nil)

			first := httptest.NewRecorder()
			if ok, _ := advancedExportGateWithIssues(first, req, collector, 7, exportjob.ExportIssuesOptions{}, false); ok {
				t.Fatal("first attempt bypassed the explicit confirmation step")
			}
			if first.Code != http.StatusConflict {
				t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
			}

			confirmed := httptest.NewRecorder()
			if ok, _ := advancedExportGateWithIssues(confirmed, req, collector, 7, exportjob.ExportIssuesOptions{}, true); !ok {
				t.Fatalf("acknowledged ordinary finding still blocked: %s", confirmed.Body.String())
			}

			recheck := httptest.NewRecorder()
			if ok, _ := advancedExportGateWithIssues(recheck, req, collector, 7, exportjob.ExportIssuesOptions{}, false); ok {
				t.Fatal("acknowledgement silently repaired or removed the finding")
			}
		})
	}
}

func TestAdvancedExportGateReturnsLosslessAcknowledgedFindingsForArtifact(t *testing.T) {
	issue := exportjob.ExportIssue{
		"severity": "blocker", "code": "cross_ref.focus_prerequisite_missing",
		"source":        map[string]any{"kind": "focus", "id": "DEMO_root"},
		"message":       "前置国策 DEMO_missing 尚未登记。",
		"player_impact": "玩家可能无法沿这条国策路线继续。",
		"repair_route":  "到国策编辑器检查 DEMO_root 的前置条件。",
		"evidence":      map[string]any{"target": "DEMO_missing"},
	}
	req := httptest.NewRequest(http.MethodPost, "/projects/7/export", nil)
	response := httptest.NewRecorder()
	ok, reviewed := advancedExportGateWithIssues(response, req, issueFixture{issue}, 7, exportjob.ExportIssuesOptions{}, true)
	if !ok || len(reviewed) != 1 {
		t.Fatalf("acknowledged gate did not return its exact reviewed issue: ok=%v issues=%v body=%s", ok, reviewed, response.Body.String())
	}
	findings := advancedExportFindings(reviewed, true)
	if len(findings) != 1 || !findings[0].Acknowledged || findings[0].Code != issue["code"] ||
		findings[0].Message != issue["message"] || findings[0].PlayerImpact != issue["player_impact"] ||
		findings[0].EditRoute != issue["repair_route"] {
		t.Fatalf("advanced issue lost fields during artifact adaptation: %+v", findings)
	}
	raw, ok := findings[0].SourceEvidence["producer_issue"].(exportjob.ExportIssue)
	if !ok || raw["evidence"].(map[string]any)["target"] != "DEMO_missing" {
		t.Fatalf("producer evidence was not preserved: %+v", findings[0].SourceEvidence)
	}
}

func TestAdvancedUploadCleanupRemovesTemporaryFile(t *testing.T) {
	req := advancedRequest(t, "/api/go/projects/1/export/diff-preview", zipBytes(t, map[string][]byte{"descriptor.mod": []byte("x")}), nil)
	upload, cleanup, err := receiveAdvancedUpload(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(upload.FilenameOnDisk); err != nil {
		t.Fatal(err)
	}
	root := upload.TempDir
	cleanup()
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("temporary directory still exists: %v", err)
	}
}
