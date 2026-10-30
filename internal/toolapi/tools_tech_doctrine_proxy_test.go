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

func newTechDoctrineTestServer(t *testing.T, proxy *RESTProxy) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}

	getServer := func(*http.Request) *Server {
		server := NewServer()
		registerCreateTechnology(server, proxy)
		registerUpdateTechnology(server, proxy)
		registerCreateEquipment(server, proxy)
		registerCreateEquipmentVariant(server, proxy)
		registerCreateDoctrine(server, proxy)
		registerUpdateDoctrine(server, proxy)
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

func decodeBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decoding forwarded body: %v", err)
	}
	return body
}

func TestCreateTechnology_ProxiesAndAppliesDefaults(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = decodeBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 101, "tech_id": "infantry_new_tech", "warnings": []string{"tech_id collides with vanilla id"},
		})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_technology",
		Arguments: map[string]any{
			"project_id":       7,
			"name":             "New Tech",
			"country_modifier": "army_attack_factor = 0.05",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/technologies" {
		t.Errorf("upstream request = %s %s, want POST /api/projects/7/technologies", gotMethod, gotPath)
	}
	if gotAuth != "" {
		t.Errorf("upstream saw Authorization = %q; the tools forward no credential", gotAuth)
	}
	if gotBody["tech_line"] != "infantry" {
		t.Errorf("tech_line = %v, want the omitted-field default 'infantry'", gotBody["tech_line"])
	}
	if gotBody["research_cost"] != 1.0 {
		t.Errorf("research_cost = %v, want the omitted-field default 1.0", gotBody["research_cost"])
	}
	if gotBody["start_year"] != float64(1936) {
		t.Errorf("start_year = %v, want the omitted-field default 1936", gotBody["start_year"])
	}
	if gotBody["ai_will_do"] != "factor = 1" {
		t.Errorf("ai_will_do = %v, want the omitted-field default 'factor = 1'", gotBody["ai_will_do"])
	}
	mods, _ := gotBody["modifiers"].(map[string]any)
	if mods["__modifier"] != "army_attack_factor = 0.05" {
		t.Errorf("modifiers.__modifier = %v, want country_modifier packed in", mods["__modifier"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["tech_db_id"] != float64(101) {
		t.Errorf("tech_db_id = %v, want 101", out["tech_db_id"])
	}
	if out["tech_id"] != "infantry_new_tech" {
		t.Errorf("tech_id = %v, want infantry_new_tech", out["tech_id"])
	}
	warnings, _ := out["warnings"].([]any)
	if len(warnings) != 1 || warnings[0] != "tech_id collides with vanilla id" {
		t.Errorf("warnings = %v, want the upstream warning relayed", out["warnings"])
	}
}

func TestCreateTechnology_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "无法为 `infantry_new_tech` 找到唯一 tech_id"})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_technology",
		Arguments: map[string]any{"project_id": 7, "name": "New Tech"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "无法为") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}

func TestUpdateTechnology_ProxiesOnlySuppliedFields(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = decodeBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "tech_id": "infantry_new_tech"})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "update_technology",
		Arguments: map[string]any{"project_id": 7, "tech_db_id": 101, "name": "Renamed Tech"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/projects/7/technologies/101" {
		t.Errorf("upstream request = %s %s, want PUT /api/projects/7/technologies/101", gotMethod, gotPath)
	}
	if len(gotBody) != 1 || gotBody["name"] != "Renamed Tech" {
		t.Errorf("body = %+v, want ONLY {name: Renamed Tech} (exclude_unset parity)", gotBody)
	}

	out := result.StructuredContent.(map[string]any)
	if out["tech_db_id"] != float64(101) || out["tech_id"] != "infantry_new_tech" {
		t.Errorf("out = %+v", out)
	}
}

func TestTechnologyLocalisationAcceptedThroughCreateAndPatchTools(t *testing.T) {
	var calls []map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, decodeBody(t, r))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "tech_id": "GER_fixture"})
	}))
	defer upstream.Close()
	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)
	for _, tc := range []struct {
		name      string
		arguments map[string]any
	}{
		{"create_technology", map[string]any{"project_id": 7, "name": "Fixture", "desc_en": "Research description", "desc_zh": "研究说明", "display_name_locs": map[string]string{"en": "Fixture", "zh": "测试"}, "description_locs": map[string]string{"en": "Research description", "zh": "研究说明"}}},
		{"update_technology", map[string]any{"project_id": 7, "tech_db_id": 101, "desc_en": "", "desc_zh": "更新说明", "description_locs": map[string]string{}}},
	} {
		result, err := session.CallTool(context.Background(), &CallToolParams{Name: tc.name, Arguments: tc.arguments})
		if err != nil || result.IsError {
			t.Fatalf("%s failed: %v %+v", tc.name, err, result)
		}
	}
	if len(calls) != 2 || calls[0]["desc_zh"] != "研究说明" || calls[0]["display_name_locs"].(map[string]any)["zh"] != "测试" {
		t.Fatalf("create localisation lost: %#v", calls)
	}
	if len(calls[1]) != 3 || calls[1]["desc_en"] != "" || calls[1]["desc_zh"] != "更新说明" || len(calls[1]["description_locs"].(map[string]any)) != 0 {
		t.Fatalf("patch did not preserve omitted versus clear: %#v", calls[1])
	}
}

func TestUpdateTechnology_UpstreamNotFoundSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Technology not found"})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "update_technology",
		Arguments: map[string]any{"project_id": 7, "tech_db_id": 999, "name": "x"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 404")
	}
	if !strings.Contains(contentText(result), "Technology not found") {
		t.Errorf("error text = %q", contentText(result))
	}
}

func TestCreateEquipment_ProxiesAndDefaultsIsArchetypeTrue(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = decodeBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 55, "equipment_id": "infantry_equipment_1", "is_archetype": true, "archetype": "",
			"warnings": []string{},
		})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_equipment",
		Arguments: map[string]any{"project_id": 7, "name": "Infantry Equipment", "equipment_type": "infantry_equipment"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/equipment" {
		t.Errorf("upstream request = %s %s, want POST /api/projects/7/equipment", gotMethod, gotPath)
	}
	if gotBody["is_archetype"] != true {
		t.Errorf("is_archetype = %v, want the omitted-field default true", gotBody["is_archetype"])
	}
	if _, present := gotBody["soft_attack"]; present {
		t.Errorf("soft_attack should be ABSENT (None = inherit), got %v", gotBody["soft_attack"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["equipment_db_id"] != float64(55) {
		t.Errorf("equipment_db_id = %v, want 55", out["equipment_db_id"])
	}
	if out["equipment_id"] != "infantry_equipment_1" {
		t.Errorf("equipment_id = %v", out["equipment_id"])
	}
	if out["is_archetype"] != true {
		t.Errorf("is_archetype = %v, want true", out["is_archetype"])
	}
}

func TestCreateEquipment_SendsExplicitStatsAndIsArchetypeFalse(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody = decodeBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 56, "equipment_id": "standalone_1", "is_archetype": false, "archetype": ""})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	_, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_equipment",
		Arguments: map[string]any{
			"project_id": 7, "name": "Standalone", "is_archetype": false, "soft_attack": 0,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if gotBody["is_archetype"] != false {
		t.Errorf("is_archetype = %v, want explicit false honored (not overwritten by the true default)", gotBody["is_archetype"])
	}
	if gotBody["soft_attack"] != 0.0 {
		t.Errorf("soft_attack = %v, want explicit 0 forwarded (caller opted OUT of inherit)", gotBody["soft_attack"])
	}
}

func TestCreateEquipment_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "equipment_id 'infantry_equipment' already exists"})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_equipment",
		Arguments: map[string]any{"project_id": 7, "equipment_id": "infantry_equipment"},
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

func TestCreateEquipmentVariant_ProxiesArchetypeAndForcesIsArchetypeFalse(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = decodeBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 57, "equipment_id": "infantry_equipment_mk2", "archetype": "infantry_equipment", "is_archetype": false,
		})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_equipment_variant",
		Arguments: map[string]any{
			"project_id": 7, "archetype": "infantry_equipment", "name": "Infantry Equipment Mk2", "soft_attack": 2.5,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/equipment" {
		t.Errorf("upstream request = %s %s, want POST /api/projects/7/equipment (SAME route as create_equipment)", gotMethod, gotPath)
	}
	if gotBody["archetype"] != "infantry_equipment" {
		t.Errorf("archetype = %v, want the parent's equipment_id string forwarded", gotBody["archetype"])
	}
	if gotBody["is_archetype"] != false {
		t.Errorf("is_archetype = %v, want hardcoded false (no is_archetype param on this tool)", gotBody["is_archetype"])
	}
	if gotBody["soft_attack"] != 2.5 {
		t.Errorf("soft_attack = %v, want the explicit override forwarded", gotBody["soft_attack"])
	}
	if _, present := gotBody["hard_attack"]; present {
		t.Errorf("hard_attack should be ABSENT (unset = inherit from archetype)")
	}

	out := result.StructuredContent.(map[string]any)
	if out["equipment_id"] != "infantry_equipment_mk2" || out["archetype"] != "infantry_equipment" {
		t.Errorf("out = %+v", out)
	}
	if _, hasWarnings := out["warnings"]; hasWarnings {
		t.Errorf("create_equipment_variant's response has no warnings key in Python — got %+v", out)
	}
}

func TestCreateEquipmentVariant_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "create failed: db error"})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_equipment_variant",
		Arguments: map[string]any{"project_id": 7, "archetype": "infantry_equipment"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 500")
	}
	if !strings.Contains(contentText(result), "create failed") {
		t.Errorf("error text = %q", contentText(result))
	}
}

func TestCreateDoctrine_ProxiesAndAppliesDefaults(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = decodeBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 9, "entity_type": "grand_doctrine", "entity_key": "land", "folder_key": "", "track_key": "",
		})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_doctrine",
		Arguments: map[string]any{"project_id": 7, "entity_key": "land"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/doctrines" {
		t.Errorf("upstream request = %s %s, want POST /api/projects/7/doctrines", gotMethod, gotPath)
	}
	if gotBody["entity_type"] != "grand_doctrine" {
		t.Errorf("entity_type = %v, want the omitted-field default 'grand_doctrine'", gotBody["entity_type"])
	}
	if gotBody["entity_key"] != "land" {
		t.Errorf("entity_key = %v, want land", gotBody["entity_key"])
	}
	rawExtras, _ := gotBody["raw_extras"].(map[string]any)
	if rawExtras == nil {
		t.Errorf("raw_extras should default to {} (not null/absent), got %v", gotBody["raw_extras"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["doctrine_db_id"] != float64(9) {
		t.Errorf("doctrine_db_id = %v, want 9", out["doctrine_db_id"])
	}
	if out["entity_key"] != "land" {
		t.Errorf("entity_key = %v", out["entity_key"])
	}
}

func TestCreateDoctrine_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"detail": "A grand_doctrine with key 'land' already exists in this project. Pick a unique entity_key.",
		})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_doctrine",
		Arguments: map[string]any{"project_id": 7, "entity_key": "land"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "Pick a unique entity_key") {
		t.Errorf("error text = %q", contentText(result))
	}
}

func TestUpdateDoctrine_ProxiesOnlySuppliedFields(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = decodeBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 9, "entity_type": "subdoctrine", "entity_key": "mobile_warfare",
			"xor": "other_sub_a other_sub_b", "allow_in_multiple_tracks": true,
		})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_doctrine",
		Arguments: map[string]any{
			"project_id": 7, "doctrine_db_id": 9, "xor": "other_sub_a other_sub_b", "allow_in_multiple_tracks": true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/projects/7/doctrines/9" {
		t.Errorf("upstream request = %s %s, want PUT /api/projects/7/doctrines/9", gotMethod, gotPath)
	}
	if len(gotBody) != 2 || gotBody["xor"] != "other_sub_a other_sub_b" || gotBody["allow_in_multiple_tracks"] != true {
		t.Errorf("body = %+v, want ONLY the two supplied fields (exclude_unset parity)", gotBody)
	}

	out := result.StructuredContent.(map[string]any)
	if out["xor"] != "other_sub_a other_sub_b" {
		t.Errorf("xor = %v", out["xor"])
	}
	if out["allow_in_multiple_tracks"] != true {
		t.Errorf("allow_in_multiple_tracks = %v", out["allow_in_multiple_tracks"])
	}
}

func TestUpdateDoctrine_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "entity_type + entity_key conflict in this project"})
	}))
	defer upstream.Close()

	srv := newTechDoctrineTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "update_doctrine",
		Arguments: map[string]any{"project_id": 7, "doctrine_db_id": 9, "entity_key": "land"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "conflict in this project") {
		t.Errorf("error text = %q", contentText(result))
	}
}
