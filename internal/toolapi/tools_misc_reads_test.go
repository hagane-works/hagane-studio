package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListRawTemplates_ProxiesAndReshapes(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"templates": []map[string]any{
				{
					"key": "scripted_gui_overview", "label_en": "Overview Panel", "label_zh": "总览面板",
					"desc_en": "A country overview panel", "desc_zh": "国家总览面板",
					"labels":       map[string]any{"en": "Overview Panel", "zh": "总览面板"},
					"descriptions": map[string]any{"en": "A country overview panel", "zh": "国家总览面板"},
					"main_param":   "seed", "route_path": "",
				},
			},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerListRawTemplates(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "list_raw_templates", Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", contentText(result))
	}

	if gotMethod != http.MethodGet || gotPath != "/api/projects/7/raw-files/templates" {
		t.Errorf("upstream request = %s %s, want GET /api/projects/7/raw-files/templates", gotMethod, gotPath)
	}
	if gotAuth != "" {
		t.Errorf("upstream saw Authorization = %q; the tools forward no credential", gotAuth)
	}

	out := result.StructuredContent.(map[string]any)
	if out["project_id"] != float64(7) {
		t.Errorf("project_id = %v, want 7", out["project_id"])
	}
	templates := out["templates"].([]any)
	if len(templates) != 1 {
		t.Fatalf("templates = %v, want 1 entry", templates)
	}
	entry := templates[0].(map[string]any)
	if entry["key"] != "scripted_gui_overview" {
		t.Errorf("templates[0].key = %v, want scripted_gui_overview", entry["key"])
	}
	labels := entry["labels"].(map[string]any)
	if labels["zh"] != "总览面板" {
		t.Errorf("templates[0].labels.zh = %v, want 总览面板", labels["zh"])
	}
	if entry["main_param"] != "seed" {
		t.Errorf("templates[0].main_param = %v, want seed", entry["main_param"])
	}
}

func TestListUiPanels_ProxiesAndResolvesWindowName(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"panels": []map[string]any{
				{
					"id": 3, "token": "EA_legitimacy", "context_type": "decision_category",
					"country_tag": "EA", "name_en": "Legitimacy", "name_zh": "合法性",
					"derived": map[string]any{"window_name": "EA_legitimacy_window"},
				},
			},
			"total": 1,
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerListUiPanels(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "list_ui_panels", Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", contentText(result))
	}
	if gotPath != "/api/projects/7/ui-panels" {
		t.Errorf("upstream path = %s, want /api/projects/7/ui-panels", gotPath)
	}

	out := result.StructuredContent.(map[string]any)
	if out["total"] != float64(1) {
		t.Errorf("total = %v, want 1", out["total"])
	}
	panels := out["panels"].([]any)
	panel := panels[0].(map[string]any)
	if panel["panel_id"] != float64(3) {
		t.Errorf("panel_id = %v, want 3", panel["panel_id"])
	}
	if panel["window_name"] != "EA_legitimacy_window" {
		t.Errorf("window_name = %v, want EA_legitimacy_window (from response.derived.window_name)", panel["window_name"])
	}
}

func TestGetSubUnits_ProxiesAndForwardsCountryTagFilter(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id": 55, "sub_unit_id": "zzz_mtn_inf_1", "name_en": "Mountain Infantry",
				"name_zh": "山地步兵", "group": "infantry", "active": true, "country_tag": "ITA",
			},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetSubUnits(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_sub_units", Arguments: map[string]any{"project_id": 7, "country_tag": "ita"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", contentText(result))
	}
	if gotPath != "/api/projects/7/sub-units?country_tag=ita" {
		t.Errorf("upstream path = %s, want /api/projects/7/sub-units?country_tag=ita (raw, unnormalised — REST does the .strip().upper())", gotPath)
	}

	out := result.StructuredContent.(map[string]any)
	if out["total"] != float64(1) {
		t.Errorf("total = %v, want 1", out["total"])
	}
	rows := out["sub_units"].([]any)
	row := rows[0].(map[string]any)
	if row["sub_unit_db_id"] != float64(55) || row["sub_unit_id"] != "zzz_mtn_inf_1" {
		t.Errorf("sub_units[0] = %+v, want id 55 / sub_unit_id zzz_mtn_inf_1", row)
	}
	if row["active"] != true {
		t.Errorf("active = %v, want true", row["active"])
	}
}

func TestGetSubUnits_NoCountryTagOmitsQueryString(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetSubUnits(s, proxy) })
	session := connectClient(t, srv.URL)

	_, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_sub_units", Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if gotPath != "/api/projects/7/sub-units" {
		t.Errorf("upstream path = %s, want /api/projects/7/sub-units with NO query string when country_tag is omitted", gotPath)
	}
}

func TestGetOobSnapshot_ProxiesFiveCollectionsAndForwardsOobFile(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"air_wing_groups": []map[string]any{{"id": 1, "oob_file": "USA_1936", "country_tag": "USA", "location": "798", "ordering": 0}},
			"air_wings":       []map[string]any{{"id": 2, "group_id": 1, "equipment_type": "fighter_equipment_0", "amount": "100", "owner": "", "name": ""}},
			"fleets":          []map[string]any{{"id": 3, "oob_file": "USA_1936", "country_tag": "USA", "name": "Pacific Fleet", "naval_base": "9601"}},
			"task_forces":     []map[string]any{{"id": 4, "fleet_id": 3, "name": "TF1", "location": "9601"}},
			"ships":           []map[string]any{{"id": 5, "task_force_id": 4, "name": "USS Test", "definition": "destroyer", "equipment_type": "des_hull", "amount": "1", "owner": "", "version_name": "1936 Model"}},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetOobSnapshot(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_oob_snapshot", Arguments: map[string]any{"project_id": 7, "oob_file": "USA_1936"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", contentText(result))
	}
	if gotPath != "/api/projects/7/oob-units/snapshot?oob_file=USA_1936" {
		t.Errorf("upstream path = %s, want /api/projects/7/oob-units/snapshot?oob_file=USA_1936", gotPath)
	}

	out := result.StructuredContent.(map[string]any)
	for _, key := range []string{"air_wing_groups_total", "air_wings_total", "fleets_total", "task_forces_total", "ships_total"} {
		if out[key] != float64(1) {
			t.Errorf("%s = %v, want 1", key, out[key])
		}
	}
	ships := out["ships"].([]any)
	ship := ships[0].(map[string]any)
	if ship["definition"] != "destroyer" || ship["version_name"] != "1936 Model" {
		t.Errorf("ships[0] = %+v, want definition destroyer / version_name '1936 Model'", ship)
	}
	fleets := out["fleets"].([]any)
	fleet := fleets[0].(map[string]any)
	if fleet["naval_base"] != "9601" {
		t.Errorf("fleets[0].naval_base = %v, want 9601", fleet["naval_base"])
	}
	if out["oob_file"] != "USA_1936" {
		t.Errorf("oob_file = %v, want USA_1936 (echoed input)", out["oob_file"])
	}
}

func TestGetOobSnapshot_CapsEachCollectionAt200ButReportsTrueTotal(t *testing.T) {
	groups := make([]map[string]any, 0, 205)
	for i := 0; i < 205; i++ {
		groups = append(groups, map[string]any{"id": i, "oob_file": "USA_1936", "country_tag": "USA", "location": "1", "ordering": i})
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"air_wing_groups": groups,
			"air_wings":       []map[string]any{},
			"fleets":          []map[string]any{},
			"task_forces":     []map[string]any{},
			"ships":           []map[string]any{},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetOobSnapshot(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_oob_snapshot", Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", contentText(result))
	}

	out := result.StructuredContent.(map[string]any)
	if out["air_wing_groups_total"] != float64(205) {
		t.Errorf("air_wing_groups_total = %v, want 205 (true count, uncapped)", out["air_wing_groups_total"])
	}
	rows := out["air_wing_groups"].([]any)
	if len(rows) != 200 {
		t.Errorf("len(air_wing_groups) = %d, want 200 (capped)", len(rows))
	}
}

func TestGetStates_ListsAllUsingResolvedDisplayNameFromREST(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id": 42, "state_id": 613, "name": "STATE_613",
				"display_name":      map[string]any{"en": "Borneo", "zh": "婆罗洲", "ru": "Борнео"},
				"display_name_locs": map[string]any{"zh": "婆罗洲", "ru": "Борнео"},
				"owner":             "ENG", "controller": "ENG",
				"provinces": []any{float64(9601), float64(9602), float64(9603)},
			},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetStates(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_states", Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", contentText(result))
	}
	if gotPath != "/api/projects/7/states" {
		t.Errorf("upstream path = %s, want /api/projects/7/states (no country_tag -> plain list route)", gotPath)
	}

	out := result.StructuredContent.(map[string]any)
	if out["total"] != float64(1) || out["returned"] != float64(1) {
		t.Errorf("total/returned = %v/%v, want 1/1", out["total"], out["returned"])
	}
	states := out["states"].([]any)
	row := states[0].(map[string]any)
	disp, ok := row["display_name"].(map[string]any)
	if !ok {
		t.Fatalf("display_name = %#v, want the per-language map relayed from REST", row["display_name"])
	}
	for lang, want := range map[string]string{"en": "Borneo", "zh": "婆罗洲", "ru": "Борнео"} {
		if disp[lang] != want {
			t.Errorf("display_name[%q] = %v, want %q (every stored language survives the proxy hop)", lang, disp[lang], want)
		}
	}
	if _, gone := row["display_name_en"]; gone {
		t.Errorf("display_name_en still present — the fixed en/zh pair is what could not report a Russian name")
	}
	if row["province_count"] != float64(3) {
		t.Errorf("province_count = %v, want 3", row["province_count"])
	}
	if _, hasNote := out["note"]; hasNote {
		t.Errorf("note = %v, want absent when results are non-empty and under the cap", out["note"])
	}
}

func TestGetStates_CountryTagUsesByOwnerRouteAndTrimsButDoesNotUppercaseOutput(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetStates(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_states", Arguments: map[string]any{"project_id": 7, "country_tag": " eng "},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if gotPath != "/api/projects/7/states/by-owner/eng" {
		t.Errorf("upstream path = %s, want /api/projects/7/states/by-owner/eng (trimmed, REST does its own .upper())", gotPath)
	}

	out := result.StructuredContent.(map[string]any)
	if out["country_tag"] != "eng" {
		t.Errorf("country_tag = %v, want the trimmed-but-NOT-uppercased 'eng'", out["country_tag"])
	}
	note, _ := out["note"].(string)
	if note != "No states owned by ENG (check the tag, or seed states first)." {
		t.Errorf("note = %q, want the zero-states-for-this-owner message with an UPPERCASED tag", note)
	}
}

func TestGetStates_NameQueryFiltersClientSide(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "state_id": 613, "name": "STATE_613", "display_name": map[string]any{"en": "Borneo", "zh": "婆罗洲"}, "owner": "ENG", "controller": "ENG", "provinces": []any{}},
			{"id": 2, "state_id": 700, "name": "STATE_700", "display_name": map[string]any{"en": "Java", "zh": "爪哇"}, "owner": "ENG", "controller": "ENG", "provinces": []any{}},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetStates(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_states", Arguments: map[string]any{"project_id": 7, "name_query": "婆罗洲"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	out := result.StructuredContent.(map[string]any)
	if out["total"] != float64(1) {
		t.Fatalf("total = %v, want 1 (only Borneo matches the zh substring)", out["total"])
	}
	states := out["states"].([]any)
	if states[0].(map[string]any)["state_id"] != float64(613) {
		t.Errorf("matched state_id = %v, want 613", states[0].(map[string]any)["state_id"])
	}

	result2, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_states", Arguments: map[string]any{"project_id": 7, "name_query": "700"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	out2 := result2.StructuredContent.(map[string]any)
	if out2["total"] != float64(1) {
		t.Fatalf("total = %v, want 1 (exact state_id match on '700')", out2["total"])
	}
}

func TestGetStates_NoStatesAtAllNotesSeedHint(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetStates(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_states", Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	out := result.StructuredContent.(map[string]any)
	note, _ := out["note"].(string)
	if note == "" || !strings.Contains(note, "seed_states_from_vanilla") {
		t.Errorf("note = %q, want it to mention seed_states_from_vanilla", note)
	}
}

func TestValidateSpecialProjects_ProxiesAndSummarizes(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"actionable": []map[string]any{
				{"kind": "missing_parent", "project_token": "proj_a", "parent": "proj_missing", "detail": "parent 'proj_missing' does not exist"},
			},
			"info":  []map[string]any{},
			"total": 1,
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerValidateSpecialProjects(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "validate_special_projects", Arguments: map[string]any{"project_id": 7, "country_tag": "eng"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", contentText(result))
	}
	if gotPath != "/api/projects/7/special-projects/validate?country_tag=eng" {
		t.Errorf("upstream path = %s, want .../validate?country_tag=eng (raw, unnormalised — REST does its own .upper())", gotPath)
	}

	out := result.StructuredContent.(map[string]any)
	if out["country_tag"] != "ENG" {
		t.Errorf("country_tag = %v, want ENG (uppercased in the OUTPUT, mirroring the Python wrapper's own `.upper()`)", out["country_tag"])
	}
	summary := out["summary"].(map[string]any)
	if summary["status"] != "action_required" {
		t.Errorf("summary.status = %v, want action_required", summary["status"])
	}
	if summary["actionable_count"] != float64(1) || summary["info_count"] != float64(0) {
		t.Errorf("summary counts = %+v, want actionable_count=1 info_count=0", summary)
	}
	actionable := out["actionable"].([]any)
	if actionable[0].(map[string]any)["kind"] != "missing_parent" {
		t.Errorf("actionable[0] = %+v, want kind=missing_parent", actionable[0])
	}
}

func TestValidateSpecialProjects_NoCountryTagOmitsQueryAndCountryTagIsNull(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query string %q, want none when country_tag is omitted", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"actionable": []map[string]any{}, "info": []map[string]any{}, "total": 0})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerValidateSpecialProjects(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "validate_special_projects", Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	out := result.StructuredContent.(map[string]any)
	if out["country_tag"] != nil {
		t.Errorf("country_tag = %v, want null/nil when omitted", out["country_tag"])
	}
	summary := out["summary"].(map[string]any)
	if summary["status"] != "ok" {
		t.Errorf("summary.status = %v, want ok when actionable is empty", summary["status"])
	}
}
