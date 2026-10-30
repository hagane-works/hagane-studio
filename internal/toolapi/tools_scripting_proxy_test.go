package toolapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newScriptingTestServer(t *testing.T, upstreamURL string) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}

	getServer := func(*http.Request) *Server {
		server := NewServer()
		proxy := NewRESTProxy(upstreamURL)
		registerCreateCustomTrait(server, proxy)
		registerCreateDynamicModifier(server, proxy)
		registerCreateRawFromTemplate(server, proxy)
		registerCreateUiPanel(server, proxy)
		registerCreateSuperEvent(server, proxy)
		registerCreateScriptedLocalisation(server, proxy)
		registerCreateScriptedSnippet(server, proxy)
		return server
	}
	streamable := toolHTTPHandler(getServer)
	handler := streamable

	srv := httptest.NewUnstartedServer(handler)
	srv.Listener.Close()
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

func TestCreateCustomTrait_ProxiesAndReshapes(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 15, "trait_id": "brave_leader", "trait_type": "country_leader",
		})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_custom_trait",
		Arguments: map[string]any{
			"project_id": 7, "trait_id": "brave_leader", "name_en": "Brave",
			"modifier": map[string]any{"army_attack_factor": 0.1},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/custom-traits" {
		t.Errorf("upstream saw %s %s, want POST /api/projects/7/custom-traits", gotMethod, gotPath)
	}
	if gotBody["trait_type"] != "country_leader" {
		t.Errorf("body[trait_type] = %v, want default country_leader", gotBody["trait_type"])
	}
	if gotBody["targeted_modifier"] == nil {
		t.Errorf("body[targeted_modifier] = nil, want [] (the `targeted_modifier or []` default)")
	}
	out := result.StructuredContent.(map[string]any)
	if out["custom_trait_id"] != float64(15) {
		t.Errorf("custom_trait_id = %v, want 15", out["custom_trait_id"])
	}
}

func TestCreateCustomTrait_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "trait (country_leader, brave_leader) already exists"})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_custom_trait",
		Arguments: map[string]any{"project_id": 7, "trait_id": "brave_leader"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "already exists") {
		t.Errorf("error text = %q", contentText(result))
	}
}

func TestCreateDynamicModifier_ProxiesAndReshapes(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 9, "dm_id": "fng_boost", "warnings": []any{"dm_id collides with a vanilla id"},
		})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_dynamic_modifier",
		Arguments: map[string]any{"project_id": 7, "dm_id": "fng_boost"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/7/dynamic-modifiers" {
		t.Errorf("path = %q, want /api/projects/7/dynamic-modifiers", gotPath)
	}
	if gotBody["modifier"] == nil {
		t.Errorf("body[modifier] = nil, want {} (the `modifier or {}` default)")
	}
	out := result.StructuredContent.(map[string]any)
	if out["dm_db_id"] != float64(9) {
		t.Errorf("dm_db_id = %v, want 9", out["dm_db_id"])
	}
	warnings, _ := out["warnings"].([]any)
	if len(warnings) != 1 || warnings[0] != "dm_id collides with a vanilla id" {
		t.Errorf("warnings = %v", out["warnings"])
	}
}

func TestCreateDynamicModifier_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "dm_id already exists"})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_dynamic_modifier",
		Arguments: map[string]any{"project_id": 7, "dm_id": "fng_boost"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "already exists") {
		t.Errorf("error text = %q", contentText(result))
	}
}

func TestCreateRawFromTemplate_ProxiesToTemplateRoute(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 3, "dest_path": "common/defines/fng_tweaks_defines.lua"},
		})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_raw_from_template",
		Arguments: map[string]any{"project_id": 7, "template": "defines", "key": "fng_tweaks"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/7/raw-files/from-template/defines" {
		t.Errorf("path = %q, want .../from-template/defines", gotPath)
	}
	if gotBody["key"] != "fng_tweaks" {
		t.Errorf("body[key] = %v, want fng_tweaks (defines' main_param is `key`, not `panel_key`)", gotBody["key"])
	}
	out := result.StructuredContent.(map[string]any)
	files, _ := out["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}
	first, _ := files[0].(map[string]any)
	if first["dest_path"] != "common/defines/fng_tweaks_defines.lua" {
		t.Errorf("files[0].dest_path = %v", first["dest_path"])
	}
	if out["file_count"] != float64(1) {
		t.Errorf("file_count = %v, want 1", out["file_count"])
	}
}

func TestCreateRawFromTemplate_ScriptedGuiUsesBareRouteAndPanelKeyField(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "dest_path": "interface/fng_overview_hagane.gui"},
			{"id": 2, "dest_path": "common/scripted_guis/fng_overview_hagane.txt"},
		})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_raw_from_template",
		Arguments: map[string]any{"project_id": 7, "template": "scripted_gui", "key": "fng_overview"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/7/raw-files/from-template" {
		t.Errorf("path = %q, want the bare .../from-template route (route_path=\"\")", gotPath)
	}
	if gotBody["panel_key"] != "fng_overview" {
		t.Errorf("body[panel_key] = %v, want fng_overview", gotBody["panel_key"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["file_count"] != float64(2) {
		t.Errorf("file_count = %v, want 2", out["file_count"])
	}
}

func TestCreateRawFromTemplate_UnknownTemplateNeverReachesUpstream(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_raw_from_template",
		Arguments: map[string]any{"project_id": 7, "template": "not_a_real_template", "key": "fng_tweaks"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for an unknown template")
	}
	if called {
		t.Error("upstream backend was called for an unknown template — should fail before any HTTP call")
	}
	if !strings.Contains(contentText(result), "Unknown template") && !strings.Contains(contentText(result), "unknown template") {
		t.Errorf("error text = %q, want it to mention the unknown template", contentText(result))
	}
}

func TestCreateRawFromTemplate_MalformedKeyNeverReachesUpstream(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_raw_from_template",
		Arguments: map[string]any{"project_id": 7, "template": "defines", "key": "NOT-VALID!"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for a malformed key")
	}
	if called {
		t.Error("upstream backend was called for a malformed key — should fail before any HTTP call")
	}
}

func TestCreateRawFromTemplate_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "invalid defines key"})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_raw_from_template",
		Arguments: map[string]any{"project_id": 7, "template": "defines", "key": "fng_tweaks"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 400")
	}
	if !strings.Contains(contentText(result), "invalid defines key") {
		t.Errorf("error text = %q", contentText(result))
	}
}

func TestCreateUiPanel_ProxiesAndReshapesDerivedWindowName(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 4, "token": "EA_legitimacy", "context_type": "player_context",
			"derived":  map[string]any{"window_name": "EA_legitimacy_window"},
			"warnings": []any{},
		})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_ui_panel",
		Arguments: map[string]any{"project_id": 7, "token": "EA_legitimacy", "context_type": "player_context"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/7/ui-panels" {
		t.Errorf("path = %q, want /api/projects/7/ui-panels", gotPath)
	}
	if gotBody["context_type"] != "player_context" {
		t.Errorf("body[context_type] = %v", gotBody["context_type"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["window_name"] != "EA_legitimacy_window" {
		t.Errorf("window_name = %v, want EA_legitimacy_window (from response.derived.window_name)", out["window_name"])
	}
	if out["panel_id"] != float64(4) {
		t.Errorf("panel_id = %v, want 4", out["panel_id"])
	}
}

func TestCreateUiPanel_DefaultsContextTypeWhenOmitted(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "token": "x", "context_type": "decision_category"})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	_, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_ui_panel",
		Arguments: map[string]any{"project_id": 7, "token": "x"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if gotBody["context_type"] != "decision_category" {
		t.Errorf("body[context_type] = %v, want decision_category default", gotBody["context_type"])
	}
}

func TestCreateUiPanel_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "panel rejected by backend"})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_ui_panel",
		Arguments: map[string]any{"project_id": 7, "token": "valid_panel"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 422")
	}
	if !strings.Contains(contentText(result), "panel rejected by backend") {
		t.Errorf("error text = %q", contentText(result))
	}
}

func TestCreateSuperEvent_ProxiesAndReshapes(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 21, "number": 4001, "country_tag": "FNG",
		})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_super_event",
		Arguments: map[string]any{
			"project_id": 7, "number": 4001, "country_tag": "FNG", "auto_fire_on_startup": true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/7/super-events" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody["auto_fire_on_startup"] != true {
		t.Errorf("body[auto_fire_on_startup] = %v, want true", gotBody["auto_fire_on_startup"])
	}
	if gotBody["options"] == nil {
		t.Errorf("body[options] = nil, want [] (the `options or []` default)")
	}
	out := result.StructuredContent.(map[string]any)
	if out["super_event_id"] != float64(21) {
		t.Errorf("super_event_id = %v, want 21", out["super_event_id"])
	}
	if out["number"] != float64(4001) {
		t.Errorf("number = %v, want 4001", out["number"])
	}
}

func TestCreateSuperEvent_RequiresReachableTrigger(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_super_event",
		Arguments: map[string]any{"project_id": 7, "number": 4001},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when neither auto_fire_on_startup nor auto_fire_after_days is set")
	}
	if called {
		t.Error("upstream backend was called for a dormant super-event — should fail before any HTTP call")
	}
	if !strings.Contains(contentText(result), "reachable in-game") {
		t.Errorf("error text = %q", contentText(result))
	}
}

func TestCreateSuperEvent_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "number 4001 already used for FNG"})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_super_event",
		Arguments: map[string]any{"project_id": 7, "number": 4001, "auto_fire_after_days": 30},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "already used for FNG") {
		t.Errorf("error text = %q", contentText(result))
	}
}

func TestCreateScriptedLocalisation_ProxiesAndReshapes(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 5, "name": "GetIdeologyTitle"})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_scripted_localisation",
		Arguments: map[string]any{
			"project_id": 7, "name": "GetIdeologyTitle",
			"body": "name = GetIdeologyTitle\ntext = { trigger = { always = yes } localization_key = FNG_TITLE }",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/7/scripted-localisations" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody["name"] != "GetIdeologyTitle" {
		t.Errorf("body[name] = %v", gotBody["name"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["loc_db_id"] != float64(5) {
		t.Errorf("loc_db_id = %v, want 5", out["loc_db_id"])
	}
}

func TestCreateScriptedLocalisation_RejectsBlankBody(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_scripted_localisation",
		Arguments: map[string]any{"project_id": 7, "name": "GetX", "body": "   "},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for a whitespace-only body")
	}
	if called {
		t.Error("upstream backend was called for a blank body — should fail before any HTTP call")
	}
}

func TestCreateScriptedLocalisation_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "name GetX already exists in this project"})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_scripted_localisation",
		Arguments: map[string]any{"project_id": 7, "name": "GetX", "body": "name = GetX"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "already exists in this project") {
		t.Errorf("error text = %q", contentText(result))
	}
}

func TestCreateScriptedSnippet_ProxiesAndReshapes(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 8, "name": "fng_grant_pp", "kind": "effect"})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_scripted_snippet",
		Arguments: map[string]any{"project_id": 7, "name": "fng_grant_pp", "body": "add_political_power = 10"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/7/scripted-snippets" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody["kind"] != "effect" {
		t.Errorf("body[kind] = %v, want effect default", gotBody["kind"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["snippet_db_id"] != float64(8) {
		t.Errorf("snippet_db_id = %v, want 8", out["snippet_db_id"])
	}
	if out["kind"] != "effect" {
		t.Errorf("kind = %v, want effect", out["kind"])
	}
}

func TestCreateScriptedSnippet_TriggerKindForwarded(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "fng_is_eligible", "kind": "trigger"})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	_, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_scripted_snippet",
		Arguments: map[string]any{"project_id": 7, "name": "fng_is_eligible", "kind": "trigger"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if gotBody["kind"] != "trigger" {
		t.Errorf("body[kind] = %v, want trigger", gotBody["kind"])
	}
}

func TestCreateScriptedSnippet_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "name fng_grant_pp already exists for kind=effect"})
	}))
	defer upstream.Close()

	srv := newScriptingTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_scripted_snippet",
		Arguments: map[string]any{"project_id": 7, "name": "fng_grant_pp"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "already exists for kind=effect") {
		t.Errorf("error text = %q", contentText(result))
	}
}
