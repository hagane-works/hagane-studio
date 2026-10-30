package toolapi

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/api"
	"haganeworks/internal/gamedir"
	"haganeworks/internal/gameindex"
	"haganeworks/internal/server"
	"haganeworks/internal/settings"
)

const programTestToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

type programFixture struct {
	t        *testing.T
	base     string
	session  *http.Client
	projects string
}

func newProgram(t *testing.T) *programFixture {
	t.Helper()
	dir := t.TempDir()
	store, err := settings.Open(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	projects := filepath.Join(dir, "projects")
	if _, err := store.Update(func(s *settings.Settings) { s.ProjectsDir = projects }); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	host := listener.Addr().String()
	srv, err := server.New(host, fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}, programTestToken)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(gameindex.SetDefault(nil))
	api.Register(srv.Mux, api.Deps{
		Settings: store, GameIndexDir: filepath.Join(dir, "game-index"), Port: listener.Addr().(*net.TCPAddr).Port,
		FindGame: func() (gamedir.Install, error) { return gamedir.Install{}, gamedir.ErrNotFound },
	})
	Register(srv.Tools, srv.Mux)
	httpServer := &http.Server{Handler: srv.Handler()}
	go func() { _ = httpServer.Serve(listener) }()
	t.Cleanup(func() { _ = httpServer.Close() })
	return &programFixture{t: t, base: "http://" + host, projects: projects}
}

func (p *programFixture) tools(method, path string, body any) (int, map[string]any) {
	p.t.Helper()
	return p.request(method, path, body, map[string]string{"Authorization": "Bearer " + programTestToken})
}

func (p *programFixture) request(method, path string, body any, header map[string]string) (int, map[string]any) {
	p.t.Helper()
	var reader *bytes.Reader
	if raw, ok := body.(string); ok {
		reader = bytes.NewReader([]byte(raw))
	} else if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			p.t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, p.base+path, reader)
	if err != nil {
		p.t.Fatal(err)
	}
	for key, value := range header {
		req.Header.Set(key, value)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		p.t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func (p *programFixture) createProject(name string) int64 {
	p.t.Helper()
	res, err := http.Get(p.base + "/")
	if err != nil {
		p.t.Fatal(err)
	}
	res.Body.Close()
	var cookie string
	for _, c := range res.Cookies() {
		cookie = c.Name + "=" + c.Value
	}
	code, out := p.request("POST", "/api/go/projects", map[string]any{"name": name, "country_tag": "TST", "country_name": "Testland"},
		map[string]string{"Cookie": cookie, "Content-Type": "application/json"})
	if code != http.StatusCreated {
		p.t.Fatalf("create project: %d %v", code, out)
	}
	return int64(out["id"].(float64))
}

func TestProgramToolAPI(t *testing.T) {
	p := newProgram(t)
	projectID := p.createProject("Tool API")

	for _, header := range []map[string]string{nil, {"Authorization": "Bearer nope"}} {
		if code, _ := p.request("GET", "/api/tools", nil, header); code != http.StatusUnauthorized {
			t.Errorf("tools without the token (%v): %d", header, code)
		}
	}

	code, list := p.tools("GET", "/api/tools", nil)
	if code != http.StatusOK {
		t.Fatalf("GET /api/tools: %d %v", code, list)
	}
	tools, _ := list["tools"].([]any)
	if len(tools) != len(desktopToolNames) {
		t.Errorf("%d tools, want %d", len(tools), len(desktopToolNames))
	}
	program, _ := list["program"].(map[string]any)
	if program["port"] == nil || program["projects_dir"] != p.projects {
		t.Errorf("program status %v", program)
	}

	code, described := p.tools("GET", "/api/tools/create_idea", nil)
	if code != http.StatusOK || described["input_schema"] == nil {
		t.Fatalf("describe create_idea: %d %v", code, described)
	}
	if ref, _ := described["field_reference"].(map[string]any); ref["name"] != "ideas" || !strings.Contains(ref["text"].(string), "idea") {
		t.Errorf("create_idea field reference %v", described["field_reference"])
	}
	if code, doc := p.tools("GET", "/api/tools/fields/tokens", nil); code != http.StatusOK || doc["text"] == "" {
		t.Errorf("fields/tokens: %d", code)
	}
	if code, _ := p.tools("GET", "/api/tools/fields/nope", nil); code != http.StatusNotFound {
		t.Errorf("unknown field reference: %d", code)
	}

	code, projects := p.tools("POST", "/api/tools/list_projects", nil)
	if code != http.StatusOK || projects["total"] != float64(1) {
		t.Fatalf("list_projects: %d %v", code, projects)
	}
	code, overview := p.tools("POST", "/api/tools/get_project_overview", map[string]any{"project_id": projectID})
	if code != http.StatusOK || overview["name"] != "Tool API" {
		t.Fatalf("get_project_overview: %d %v", code, overview)
	}

	code, idea := p.tools("POST", "/api/tools/create_idea", map[string]any{
		"project_id": projectID, "name": "Tool Spirit", "idea_id": "TST_tool_spirit", "country_tag": "TST",
		"modifier": map[string]any{"stability_factor": 0.05},
	})
	if code != http.StatusOK || idea["idea_id"] != "TST_tool_spirit" {
		t.Fatalf("create_idea: %d %v", code, idea)
	}
	if receipt, _ := idea["mutation_receipt"].(map[string]any); receipt["write_status"] != "committed" {
		t.Errorf("create_idea receipt %v", idea["mutation_receipt"])
	}
	code, event := p.tools("POST", "/api/tools/create_event_namespace", map[string]any{"project_id": projectID, "namespace": "tst_tool"})
	if code != http.StatusOK {
		t.Fatalf("create_event_namespace: %d %v", code, event)
	}
	matches, _ := filepath.Glob(filepath.Join(p.projects, "*", "data", "ideas.json"))
	if len(matches) != 1 {
		t.Fatalf("ideas.json: %v", matches)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil || !strings.Contains(string(raw), "TST_tool_spirit") {
		t.Errorf("the idea is not in %s: %v %.200s", matches[0], err, raw)
	}
	code, read := p.tools("POST", "/api/tools/get_entity", map[string]any{"project_id": projectID, "kind": "idea", "id": idea["idea_db_id"]})
	if code != http.StatusOK {
		t.Errorf("get_entity idea: %d %v", code, read)
	}

	if code, out := p.tools("POST", "/api/tools/create_idea", map[string]any{"project_id": projectID, "nope": 1}); code != http.StatusBadRequest || out["code"] != "invalid_arguments" {
		t.Errorf("unknown argument: %d %v", code, out)
	}
	if code, out := p.tools("POST", "/api/tools/get_project_overview", map[string]any{"project_id": 999}); code != http.StatusNotFound || out["detail"] == "" {
		t.Errorf("missing project: %d %v", code, out)
	}
	if code, out := p.tools("POST", "/api/tools/nope", nil); code != http.StatusNotFound || out["code"] != "unknown_tool" {
		t.Errorf("unknown tool: %d %v", code, out)
	}
	if code, _ := p.tools("POST", "/api/tools/list_projects", "[1]"); code != http.StatusBadRequest {
		t.Errorf("array body: %d", code)
	}
}

var desktopToolNames = []string{
	"add_agency_upgrade", "add_air_wing", "add_division_deployment", "add_event_option", "add_mio_trait", "add_ship",
	"add_task_force", "add_technology_surface_node", "apply_delete_ideology", "apply_gui_overlay_edits",
	"apply_ui_panel_operations", "assign_character_role", "batch_update_entities", "check_world_map",
	"confirm_technology_surface_environment", "connect_focus", "create_ai_strategy_plan", "create_ai_template",
	"create_air_wing_group", "create_autonomy_state", "create_bookmark", "create_character", "create_country",
	"create_country_split", "create_custom_trait", "create_decision", "create_decision_category",
	"create_division_template", "create_doctrine", "create_dynamic_modifier", "create_equipment",
	"create_equipment_variant", "create_event", "create_event_namespace", "create_fleet", "create_focus",
	"create_focus_tree", "create_idea", "create_ideology_root", "create_ideology_subtype",
	"create_intelligence_agency", "create_mio", "create_modifier_definition", "create_opinion_modifier",
	"create_raw_file", "create_raw_from_template", "create_scripted_localisation", "create_scripted_snippet",
	"create_special_project", "create_specialization", "create_sub_unit", "create_super_event", "create_technology",
	"create_technology_surface_branch", "create_terrain_sketch", "create_ui_panel", "create_wargoal",
	"create_world_from_images", "create_world_image", "delete_entity", "delete_focus",
	"delete_technology_surface_branch", "discard_ideology_conflict", "edit_state", "ensure_project_settings",
	"export_project", "get_characters", "get_entity", "get_focus_tree", "get_gui_overlay", "get_gui_sprites",
	"get_gui_texts", "get_gui_window", "get_knowledge_article", "get_oob_snapshot", "get_politics_references", "get_project_overview",
	"get_states", "get_sub_units", "get_technology_surface_branch", "get_technology_surface_environment",
	"get_ui_panel", "get_ui_panel_files", "get_world_image", "get_world_state_layout", "list_entities",
	"list_projects", "list_raw_templates", "list_technology_surface_branches", "list_ui_panels",
	"list_world_map_records", "name_world_map", "preview_delete_entity", "preview_delete_ideology",
	"preview_gui_overlay", "preview_world_from_images", "repair_technology_surface_timeline", "repair_world_map",
	"restore_world_map", "search_knowledge", "seed_states_from_vanilla", "set_country_history", "set_focus_mutual_exclusion",
	"undo_world_map_change", "update_ai_strategy_plan", "update_ai_template", "update_air_wing",
	"update_air_wing_group", "update_autonomy_state", "update_bookmark", "update_character", "update_country",
	"update_custom_trait", "update_decision", "update_decision_category", "update_division_deployment",
	"update_division_template", "update_doctrine", "update_dynamic_modifier", "update_equipment", "update_event",
	"update_event_namespace", "update_fleet", "update_focus", "update_focus_tree", "update_gui_texts", "update_idea",
	"update_ideology", "update_intelligence_agency", "update_mio", "update_modifier_definition",
	"update_opinion_modifier", "update_raw_file", "update_scripted_localisation", "update_scripted_snippet",
	"update_ship", "update_special_project", "update_specialization", "update_sub_unit", "update_super_event",
	"update_task_force", "update_technology", "update_technology_surface_branch", "update_ui_panel", "update_wargoal",
	"validate_gui_overlay", "validate_project", "validate_special_projects", "validate_ui_panel",
}
