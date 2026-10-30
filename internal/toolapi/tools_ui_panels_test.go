package toolapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"haganeworks/internal/hoi4/exportengine"
)

func newUIPanelTestServer(t *testing.T, upstreamURL string) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	getServer := func(*http.Request) *Server {
		server := NewServer()
		proxy := NewRESTProxy(upstreamURL)
		registerCreateUiPanel(server, proxy)
		registerListUiPanels(server, proxy)
		registerUIPanelTools(server, proxy)
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

func uiPanelFixture() map[string]any {
	return map[string]any{
		"id": 4, "project_id": 7, "token": "CC1_war_status", "country_tag": "CC1",
		"name_en": "War status", "name_zh": "战争局势",
		"display_name_locs": map[string]any{"en": "War status", "zh": "战争局势"},
		"context_type":      "decision_category", "parent_window_token": "decision_tab",
		"visible_trigger": "always = yes",
		"window_props":    map[string]any{"x": 20, "y": 30, "width": 400, "height": 180},
		"controls": []any{
			map[string]any{
				"kind": "container", "name": "body", "x": 0, "y": 0, "w": 380, "h": 130,
				"children": []any{
					map[string]any{"kind": "text", "name": "title", "x": 8, "y": 8, "text": "Tension"},
					map[string]any{"kind": "icon", "name": "badge", "x": 8, "y": 36, "sprite": "GFX_CC1_badge"},
				},
			},
			map[string]any{"kind": "button", "name": "action", "x": 20, "y": 140, "text": "Act", "click_effect": "add_political_power = 1"},
			map[string]any{"kind": "icon", "name": "obsolete", "x": 300, "y": 10, "sprite": "GFX_CC1_old"},
		},
		"raw_extras": map[string]any{}, "source": "created",
		"created_at": "2026-10-14T10:00:00Z", "updated_at": "2026-10-14T10:00:00Z",
		"derived": map[string]any{
			"window_name": "CC1_war_status_panel",
			"gui_path":    "interface/zz_uipanel_CC1_war_status.gui",
			"txt_path":    "common/scripted_guis/zz_uipanel_CC1_war_status.txt",
		},
		"warnings": []any{"backend advisory"},
	}
}

func uiPanelFilesFixture() map[string]any {
	return map[string]any{
		"token": "CC1_war_status", "window_name": "CC1_war_status_panel",
		"gui_path": "interface/zz_uipanel_CC1_war_status.gui",
		"gui_text": `guiTypes = { containerWindowType = { name = "CC1_war_status_panel"
			containerWindowType = { name = "CC1_war_status_body"
				instantTextBoxType = { name = "CC1_war_status_title" }
				iconType = { name = "CC1_war_status_badge" spriteType = "GFX_CC1_badge" }
			}
			buttonType = { name = "CC1_war_status_action" }
			iconType = { name = "CC1_war_status_obsolete" spriteType = "GFX_CC1_old" }
		} }`,
		"txt_path":  "common/scripted_guis/zz_uipanel_CC1_war_status.txt",
		"txt_text":  "scripted_gui = { CC1_war_status = { window_name = \"CC1_war_status_panel\" effects = { CC1_war_status_action_click = { } } } }",
		"loc":       map[string]any{"en": "l_english:"},
		"loc_paths": []any{"localisation/english/replace/CC1_ui_panels_l_english.yml"},
	}
}

func TestGetUIPanelReturnsFullReadbackAndCapabilityGaps(t *testing.T) {
	var method, path string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_ = json.NewEncoder(w).Encode(uiPanelFixture())
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_ui_panel", Arguments: map[string]any{"project_id": 7, "panel_id": 4},
	})
	if err != nil || result.IsError {
		t.Fatalf("get_ui_panel: err=%v result=%s", err, contentText(result))
	}
	if method != http.MethodGet || path != "/api/projects/7/ui-panels/4" {
		t.Fatalf("upstream = %s %s", method, path)
	}
	out := result.StructuredContent.(map[string]any)
	if out["panel_id"] != float64(4) || out["updated_at"] != "2026-10-14T10:00:00Z" {
		t.Fatalf("readback identifiers = %#v", out)
	}
	if _, ok := out["controls"].([]any); !ok {
		t.Fatalf("controls missing from full readback: %T", out["controls"])
	}
	if _, ok := out["derived"].(map[string]any); !ok {
		t.Fatalf("derived missing: %T", out["derived"])
	}
	if len(out["capability_gaps"].([]any)) != 1 {
		t.Fatalf("capability gaps not explicit: %#v", out["capability_gaps"])
	}
	guard := out["revision_guard"].(map[string]any)
	if guard["server_enforced"] != true || guard["conflict_code"] != "ui_panel_revision_conflict" {
		t.Fatalf("revision guard contract missing: %#v", guard)
	}
}

func TestCreateUIPanelAcceptsCompleteStructuredPanelAndNormalizesControls(t *testing.T) {
	var body map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/projects/7/ui-panels" {
			t.Fatalf("unexpected upstream call: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		row := uiPanelFixture()
		row["controls"] = body["controls"]
		row["window_props"] = body["window_props"]
		_ = json.NewEncoder(w).Encode(row)
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_ui_panel", Arguments: map[string]any{
			"project_id": 7, "token": "CC1_war_status", "country_tag": "CC1",
			"context_type": "decision_category", "parent_window_token": "decision_tab",
			"visible_trigger":   "always = yes",
			"display_name_locs": map[string]any{"en": "War status", "zh": "战争局势"},
			"window_props":      map[string]any{"x": 20, "y": 30, "width": 400, "height": 180},
			"controls": []any{map[string]any{
				"kind": "progressbar", "name": "tension", "x": 10, "y": 20,
				"width": 200, "height": 20, "sprite": "GFX_CC1_tension", "frame_var": "CC1_tension_frame",
			}},
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("create full panel: err=%v result=%s", err, contentText(result))
	}
	controls := body["controls"].([]any)
	control := controls[0].(map[string]any)
	if control["w"] != float64(200) || control["h"] != float64(20) {
		t.Fatalf("control aliases were not normalized: %#v", control)
	}
	if _, exists := control["width"]; exists {
		t.Fatalf("width alias leaked into persisted control: %#v", control)
	}
	if body["parent_window_token"] != "decision_tab" || body["visible_trigger"] != "always = yes" {
		t.Fatalf("full panel fields not forwarded: %#v", body)
	}
	out := result.StructuredContent.(map[string]any)
	if out["panel_id"] != float64(4) || out["diagnostics"] == nil {
		t.Fatalf("create lacks full readback/diagnostics: %#v", out)
	}
}

func TestCreateUIPanelPreservesHistoricalMultilingualTextOnFirstCall(t *testing.T) {
	var body map[string]any
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/api/projects/7/ui-panels" {
			t.Errorf("unexpected route %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		row := uiPanelFixture()
		row["controls"] = body["controls"]
		_ = json.NewEncoder(w).Encode(row)
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result := callTool(t, session, "create_ui_panel", map[string]any{
		"project_id": 7, "token": "GER_regression_panel", "context_type": "player_context",
		"controls": []any{
			map[string]any{"kind": "container", "name": "body", "children": []any{
				map[string]any{"kind": "text", "name": "title", "text": map[string]any{"en": "QA Regression Panel", "zh": "QA回归面板"}},
			}},
			map[string]any{
				"kind": "button", "name": "gain_power", "click_effect": "add_political_power = 1",
				"text":      map[string]any{"en": "Gain Power", "zh": "增加政治点"},
				"text_locs": map[string]any{"en": "Gain Power", "de": "Macht gewinnen"},
			},
		},
	})
	if result.IsError || calls != 1 {
		t.Fatalf("historical request must succeed on its first invocation: calls=%d result=%s", calls, contentText(result))
	}
	controls := body["controls"].([]any)
	title := controls[0].(map[string]any)["children"].([]any)[0].(map[string]any)
	button := controls[1].(map[string]any)
	for _, pair := range []struct {
		control map[string]any
		want    map[string]string
	}{
		{title, map[string]string{"en": "QA Regression Panel", "zh": "QA回归面板"}},
		{button, map[string]string{"en": "Gain Power", "zh": "增加政治点", "de": "Macht gewinnen"}},
	} {
		if _, exists := pair.control["text"]; exists {
			t.Fatalf("locale map was left in scalar text: %#v", pair.control)
		}
		locs := pair.control["text_locs"].(map[string]any)
		if len(locs) != len(pair.want) {
			t.Fatalf("a locale was lost or invented: %#v", locs)
		}
		for locale, text := range pair.want {
			if locs[locale] != text {
				t.Fatalf("%s locale changed: %#v", locale, locs)
			}
		}
	}
	gui, script, err := exportengine.CompileUIPanel(body)
	if err != nil || !strings.Contains(gui, `GER_regression_panel_title`) || !strings.Contains(script, "add_political_power = 1") {
		t.Fatalf("normalized controls must compile with the real exporter: err=%v", err)
	}
	for locale, expected := range map[string][]string{
		"en": {"QA Regression Panel", "Gain Power"},
		"zh": {"QA回归面板", "增加政治点"},
		"de": {"Macht gewinnen"},
	} {
		localisation := exportengine.GenerateUIPanelsLocalisation([]map[string]any{body}, locale)
		for _, text := range expected {
			if !strings.Contains(localisation, text) {
				t.Fatalf("export lost %s text %q: %s", locale, text, localisation)
			}
		}
	}
}

func TestCreateUIPanelRejectsAmbiguousOrInvalidMultilingualTextWithoutWriting(t *testing.T) {
	for _, test := range []struct {
		name string
		text any
		locs any
		want string
	}{
		{"conflicting explicit locale", map[string]any{"en": "A"}, map[string]any{"en": "B"}, "panel.control_text_locs_conflict"},
		{"nonstring locale", map[string]any{"en": 3}, nil, "panel.control_text"},
		{"empty locale", map[string]any{"": "A"}, nil, "panel.control_text"},
		{"empty map", map[string]any{}, nil, "panel.control_text"},
		{"invalid explicit locales", map[string]any{"en": "A"}, "B", "panel.control_text_locs"},
		{"null byte", map[string]any{"en": "A\x00B"}, nil, "panel.control_text"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
			defer upstream.Close()
			session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
			control := map[string]any{"kind": "text", "name": "title", "text": test.text}
			if test.locs != nil {
				control["text_locs"] = test.locs
			}
			result := callTool(t, session, "create_ui_panel", map[string]any{
				"project_id": 7, "token": "GER_regression_panel", "controls": []any{control},
			})
			if !result.IsError || !strings.Contains(contentText(result), test.want) || calls != 0 {
				t.Fatalf("ambiguous text must not be guessed or written: calls=%d result=%s", calls, contentText(result))
			}
		})
	}
}

func TestCreateUIPanelRejectsDuplicateControlNamesBeforePOST(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_ui_panel", Arguments: map[string]any{
			"project_id": 7, "token": "CC1_duplicate",
			"controls": []any{
				map[string]any{"kind": "text", "name": "value"},
				map[string]any{"kind": "icon", "name": "value", "sprite": "GFX_value"},
			},
		},
	})
	if err != nil || !result.IsError || !strings.Contains(contentText(result), "panel.control_name_duplicate") {
		t.Fatalf("duplicate control must be rejected: err=%v result=%s", err, contentText(result))
	}
	if called {
		t.Fatal("invalid complete panel reached POST")
	}
}

func TestUIPanelValidationRejectsFieldsTheExporterWouldDrop(t *testing.T) {
	panel := uiPanelFixture()
	panel["window_props"].(map[string]any)["mystery_anchor"] = "CENTER"
	controls := panel["controls"].([]any)
	controls[0].(map[string]any)["engine_magic"] = "silently lost"
	controls[1].(map[string]any)["raw_extras"] = map[string]any{
		"nested": map[string]any{"level": map[string]any{"too": "deep"}},
	}
	_, diagnostics := validateUIPanelData(panel)
	codes := map[string]bool{}
	for _, diagnostic := range diagnostics {
		codes[diagnostic.Code] = true
	}
	for _, want := range []string{"panel.window_field_unknown", "panel.control_field_unknown", "panel.raw_extras_value"} {
		if !codes[want] {
			t.Fatalf("missing %s in diagnostics: %#v", want, diagnostics)
		}
	}
}

func TestUIPanelValidationRejectsUnsafeAndCaseAmbiguousRawExtras(t *testing.T) {
	panel := uiPanelFixture()
	panel["raw_extras"] = map[string]any{
		"effects": "set_variable = { which = tension value = 1",
		"Effects": "add_political_power = 5",
	}
	controls := panel["controls"].([]any)
	controls[0].(map[string]any)["raw_extras"] = map[string]any{
		"Foo": "yes",
		"foo": "no",
		"nested": map[string]any{
			"Bar": "value = {",
			"bar": "value = yes",
		},
	}

	_, diagnostics := validateUIPanelData(panel)
	codes := map[string]int{}
	for _, diagnostic := range diagnostics {
		codes[diagnostic.Code]++
	}
	if codes["panel.raw_extras_key_duplicate"] < 3 {
		t.Fatalf("case-ambiguous raw extras were not all rejected: %#v", diagnostics)
	}
	unsafe := false
	for code := range codes {
		if strings.Contains(code, "raw_extras.") {
			unsafe = true
			break
		}
	}
	if !unsafe {
		t.Fatalf("unbalanced raw script was not rejected: %#v", diagnostics)
	}
}

func TestCreateUIPanelRequiresSafeTokenBeforePOST(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_ui_panel", Arguments: map[string]any{"project_id": 7, "token": ""},
	})
	if err != nil || !result.IsError || !strings.Contains(contentText(result), "token required") {
		t.Fatalf("empty token must be rejected: err=%v result=%s", err, contentText(result))
	}
	if called {
		t.Fatal("invalid token reached POST")
	}
}

func TestUpdateUIPanelIsPartialAndForwardsRevisionGuard(t *testing.T) {
	var mu sync.Mutex
	var putBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(uiPanelFixture())
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			row := uiPanelFixture()
			row["name_zh"] = putBody["name_zh"]
			row["updated_at"] = "2026-10-14T10:01:00Z"
			_ = json.NewEncoder(w).Encode(row)
		}
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_ui_panel", Arguments: map[string]any{
			"project_id": 7, "panel_id": 4,
			"expected_updated_at": "2026-10-14T10:00:00Z", "name_zh": "战争态势",
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("update_ui_panel: err=%v result=%s", err, contentText(result))
	}
	mu.Lock()
	defer mu.Unlock()
	if len(putBody) != 2 || putBody["name_zh"] != "战争态势" || putBody["expected_updated_at"] != "2026-10-14T10:00:00Z" {
		t.Fatalf("PUT must contain only partial field + guard: %#v", putBody)
	}
	if _, present := putBody["controls"]; present {
		t.Fatal("update_ui_panel must never accept or send controls")
	}
	out := result.StructuredContent.(map[string]any)
	if out["updated_at"] != "2026-10-14T10:01:00Z" || out["readback"] == nil {
		t.Fatalf("missing post-write readback: %#v", out)
	}
}

func TestUpdateUIPanelMergesObjectFieldsKeyByKey(t *testing.T) {
	var putBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			_ = json.NewDecoder(r.Body).Decode(&putBody)
		}
		_ = json.NewEncoder(w).Encode(uiPanelFixture())
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_ui_panel", Arguments: map[string]any{
			"project_id": 7, "panel_id": 4, "expected_updated_at": "2026-10-14T10:00:00Z",
			"window_props": map[string]any{"x": 50, "y": nil},
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("update_ui_panel: err=%v result=%s", err, contentText(result))
	}
	props, _ := putBody["window_props"].(map[string]any)
	if props["x"] != float64(50) || props["width"] != float64(400) || props["height"] != float64(180) {
		t.Fatalf("window_props lost the keys it did not change: %#v", props)
	}
	if _, kept := props["y"]; kept {
		t.Fatalf("null did not remove y: %#v", props)
	}
}

func TestUpdateUIPanelTokenCollisionIsNotReportedAsStale(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(uiPanelFixture())
			return
		}
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "token 'CC1_other' already exists in this project"})
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_ui_panel", Arguments: map[string]any{
			"project_id": 7, "panel_id": 4, "expected_updated_at": "2026-10-14T10:00:00Z", "token": "CC1_other",
		},
	})
	text := contentText(result)
	if err != nil || !result.IsError || strings.Contains(text, "ui_panel_revision_conflict") || !strings.Contains(text, "already exists") {
		t.Fatalf("token collision reported as a stale revision: err=%v result=%s", err, text)
	}
}

func TestUpdateUIPanelRejectsStaleSnapshotBeforePUT(t *testing.T) {
	puts := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			puts++
		}
		_ = json.NewEncoder(w).Encode(uiPanelFixture())
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_ui_panel", Arguments: map[string]any{
			"project_id": 7, "panel_id": 4,
			"expected_updated_at": "2026-10-14T09:00:00Z", "name_en": "Stale",
		},
	})
	if err != nil || !result.IsError || !strings.Contains(contentText(result), "stale ui-panel snapshot") {
		t.Fatalf("expected stale-snapshot tool error: err=%v result=%s", err, contentText(result))
	}
	if puts != 0 {
		t.Fatalf("stale edit reached PUT %d time(s)", puts)
	}
}

func TestUpdateUIPanelSurfacesAuthoritativeServerRevisionConflict(t *testing.T) {
	var putBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(uiPanelFixture())
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&putBody)
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"detail": "stale ui-panel revision: expected_updated_at=2026-10-14T10:00:00Z, actual_updated_at=2026-10-14T10:00:01Z", "code": "ui_panel_revision_conflict",
			"panel_id": 4, "expected_updated_at": "2026-10-14T10:00:00Z",
			"actual_updated_at": "2026-10-14T10:00:01Z",
		})
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_ui_panel", Arguments: map[string]any{
			"project_id": 7, "panel_id": 4,
			"expected_updated_at": "2026-10-14T10:00:00Z", "name_en": "Concurrent",
		},
	})
	if err != nil || !result.IsError || !strings.Contains(contentText(result), "ui_panel_revision_conflict") || !strings.Contains(contentText(result), "get_ui_panel") {
		t.Fatalf("409 must surface as a stale-revision tool error: err=%v result=%s", err, contentText(result))
	}
	if putBody["expected_updated_at"] != "2026-10-14T10:00:00Z" {
		t.Fatalf("authoritative guard was not forwarded: %#v", putBody)
	}
}

func TestApplyUIPanelOperationsCoversAllFiveOperationsInOnePUT(t *testing.T) {
	var putBody map[string]any
	puts := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(uiPanelFixture())
			return
		}
		puts++
		_ = json.NewDecoder(r.Body).Decode(&putBody)
		row := uiPanelFixture()
		row["controls"] = putBody["controls"]
		row["updated_at"] = "2026-10-14T10:02:00Z"
		_ = json.NewEncoder(w).Encode(row)
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "apply_ui_panel_operations", Arguments: map[string]any{
			"project_id": 7, "panel_id": 4, "expected_updated_at": "2026-10-14T10:00:00Z",
			"operations": []any{
				map[string]any{"operation": "add", "parent_control_path": "body", "index": 1, "control": map[string]any{"kind": "progressbar", "name": "tension", "x": 10, "y": 32, "sprite": "GFX_CC1_tension", "frame_var": "CC1_tension_frame"}},
				map[string]any{"operation": "update", "control_path": "body/title", "fields": map[string]any{"text": "World tension", "format": "center"}},
				map[string]any{"operation": "move", "control_name": "badge", "index": 0},
				map[string]any{"operation": "reorder", "control_name": "action", "index": 0},
				map[string]any{"operation": "delete", "control_name": "obsolete"},
			},
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("apply_ui_panel_operations: err=%v result=%s", err, contentText(result))
	}
	if puts != 1 {
		t.Fatalf("operations must commit with one PUT, got %d", puts)
	}
	if putBody["expected_updated_at"] != "2026-10-14T10:00:00Z" {
		t.Fatalf("revision guard not forwarded: %#v", putBody)
	}
	controls := putBody["controls"].([]any)
	if len(controls) != 3 || controls[0].(map[string]any)["name"] != "action" || controls[1].(map[string]any)["name"] != "badge" || controls[2].(map[string]any)["name"] != "body" {
		t.Fatalf("root order/result wrong: %#v", controls)
	}
	children := controls[2].(map[string]any)["children"].([]any)
	if len(children) != 2 || children[0].(map[string]any)["text"] != "World tension" || children[1].(map[string]any)["name"] != "tension" {
		t.Fatalf("nested operation result wrong: %#v", children)
	}
	out := result.StructuredContent.(map[string]any)
	if out["applied_count"] != float64(5) || out["readback"] == nil || out["diagnostics"] == nil {
		t.Fatalf("operation result lacks readback/diagnostics: %#v", out)
	}
}

func TestApplyUIPanelOperationsRefusesDestructiveContainerDelete(t *testing.T) {
	puts := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			puts++
		}
		_ = json.NewEncoder(w).Encode(uiPanelFixture())
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "apply_ui_panel_operations", Arguments: map[string]any{
			"project_id": 7, "panel_id": 4, "expected_updated_at": "2026-10-14T10:00:00Z",
			"operations": []any{map[string]any{"operation": "delete", "control_name": "body"}},
		},
	})
	if err != nil || !result.IsError || !strings.Contains(contentText(result), "cascade=true") {
		t.Fatalf("expected guarded delete error: err=%v result=%s", err, contentText(result))
	}
	if puts != 0 {
		t.Fatal("rejected delete must not write")
	}
}

func TestGetAndValidateUIPanelFilesUseProductionCompileEndpoint(t *testing.T) {
	var paths []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/sprite-references") {
			sprites := map[string]any{
				"GFX_CC1_badge": map[string]any{"resolved": true, "width": 32, "height": 32, "frames": 1},
				"GFX_CC1_old":   map[string]any{"resolved": true, "width": 32, "height": 32, "frames": 1},
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"sprites": sprites})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/files") {
			_ = json.NewEncoder(w).Encode(uiPanelFilesFixture())
			return
		}
		_ = json.NewEncoder(w).Encode(uiPanelFixture())
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	filesResult, err := session.CallTool(context.Background(), &CallToolParams{Name: "get_ui_panel_files", Arguments: map[string]any{"project_id": 7, "panel_id": 4}})
	if err != nil || filesResult.IsError {
		t.Fatalf("get files: err=%v result=%s", err, contentText(filesResult))
	}
	files := filesResult.StructuredContent.(map[string]any)
	if files["gui_path"] != "interface/zz_uipanel_CC1_war_status.gui" || !strings.Contains(files["gui_text"].(string), "CC1_war_status_panel") {
		t.Fatalf("compiled files missing: %#v", files)
	}
	checksums, ok := files["checksums"].(map[string]any)
	if !ok || len(stringField(checksums, "interface/zz_uipanel_CC1_war_status.gui")) != 64 {
		t.Fatalf("compiled file checksums missing: %#v", files["checksums"])
	}
	validationResult, err := session.CallTool(context.Background(), &CallToolParams{Name: "validate_ui_panel", Arguments: map[string]any{"project_id": 7, "panel_id": 4}})
	if err != nil || validationResult.IsError {
		t.Fatalf("validate: err=%v result=%s", err, contentText(validationResult))
	}
	validation := validationResult.StructuredContent.(map[string]any)
	if validation["valid"] != true || validation["export_valid"] != true || validation["references_valid"] != true || validation["compiled_readback_valid"] != true || validation["verification_status"] != "export_valid_with_runtime_gap" || validation["runtime_verified"] != false {
		t.Fatalf("validation must separate compile from runtime: %#v", validation)
	}
	if len(paths) != 4 || paths[0] != "/api/projects/7/ui-panels/4/files" || paths[1] != "/api/projects/7/ui-panels/4" || paths[2] != "/api/projects/7/ui-panels/4/sprite-references" || paths[3] != "/api/projects/7/ui-panels/4/files" {
		t.Fatalf("unexpected validation calls: %#v", paths)
	}
}

func TestValidateUIPanelRejectsUnresolvedSpriteFromProjectScopedResolver(t *testing.T) {
	var paths []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, "/sprite-references"):
			_ = json.NewEncoder(w).Encode(map[string]any{"sprites": map[string]any{
				"GFX_CC1_badge": map[string]any{"resolved": true, "missing_file": false},
				"GFX_CC1_old":   map[string]any{"resolved": false, "missing_file": true},
			}})
		case strings.HasSuffix(r.URL.Path, "/files"):
			_ = json.NewEncoder(w).Encode(uiPanelFilesFixture())
		default:
			_ = json.NewEncoder(w).Encode(uiPanelFixture())
		}
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "validate_ui_panel", Arguments: map[string]any{"project_id": 7, "panel_id": 4},
	})
	if err != nil || result.IsError {
		t.Fatalf("validate: err=%v result=%s", err, contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if out["valid"] != false || out["references_valid"] != false || out["verification_status"] != "failed" {
		t.Fatalf("unresolved sprite incorrectly completed validation: %#v", out)
	}
	if strings.Contains(strings.Join(paths, "\n"), "/gui-render/") {
		t.Fatalf("validate_ui_panel must not call the game view's routes: %#v", paths)
	}
	encoded, _ := json.Marshal(out)
	if !strings.Contains(string(encoded), "panel.sprite_unresolved") || !strings.Contains(string(encoded), "GFX_CC1_old") {
		t.Fatalf("missing unresolved diagnostic: %s", encoded)
	}
}

func TestUpdateUIPanelSchemaCannotReplaceControls(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	tools, err := session.ListTools(context.Background(), &ListToolsParams{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name != "update_ui_panel" {
			continue
		}
		raw, _ := json.Marshal(tool.InputSchema)
		if strings.Contains(string(raw), `"controls"`) {
			t.Fatalf("update_ui_panel schema exposes unsafe whole-tree replacement: %s", raw)
		}
		if !strings.Contains(string(raw), `"expected_updated_at"`) {
			t.Fatalf("update_ui_panel schema lacks stale-write guard: %s", raw)
		}
		return
	}
	t.Fatal("update_ui_panel not registered")
}

func TestUIPanelScriptBraceCheckSkipsComments(t *testing.T) {
	panel := uiPanelFixture()
	controls := panel["controls"].([]any)
	controls[1].(map[string]any)["click_effect"] = "#{{\n}\n}\nx = {\ny = {\n#}}"
	_, diagnostics := validateUIPanelData(panel)
	rejected := false
	for _, diagnostic := range diagnostics {
		rejected = rejected || strings.Contains(diagnostic.Code, "click_effect")
	}
	if !rejected {
		t.Fatalf("braces hidden behind comments were accepted: %#v", diagnostics)
	}

	panel = uiPanelFixture()
	controls = panel["controls"].([]any)
	controls[1].(map[string]any)["click_effect"] = "add_political_power = 1 # a { in a comment\nadd_stability = 0.01"
	_, diagnostics = validateUIPanelData(panel)
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Code, "click_effect") {
			t.Fatalf("a brace in a comment was counted: %#v", diagnostics)
		}
	}
}

func TestApplyUIPanelOperationsRefusesChildrenOnANonContainer(t *testing.T) {
	puts := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			puts++
		}
		_ = json.NewEncoder(w).Encode(uiPanelFixture())
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "apply_ui_panel_operations", Arguments: map[string]any{
			"project_id": 7, "panel_id": 4, "expected_updated_at": "2026-10-14T10:00:00Z",
			"operations": []any{map[string]any{"operation": "add", "control": map[string]any{
				"kind": "icon", "name": "medal", "sprite": "GFX_CC1_badge",
				"children": []any{map[string]any{"kind": "text", "name": "medal_label", "text": "Lost?"}},
			}}},
		},
	})
	if err != nil || !result.IsError || !strings.Contains(contentText(result), "children") {
		t.Fatalf("children on an icon were not refused: err=%v result=%s", err, contentText(result))
	}
	if puts != 0 {
		t.Fatalf("refused batch reached PUT %d time(s)", puts)
	}
}

func TestApplyUIPanelOperationsRefusesFieldsTheOperationDoesNotTake(t *testing.T) {
	controls, _ := uiPanelMapSlice(uiPanelFixture()["controls"])
	index := 0
	for name, op := range map[string]uiPanelOperation{
		"move with fields":     {Operation: "move", ControlName: "action", ParentControlName: "body", Fields: map[string]any{"x": 10}},
		"update with control":  {Operation: "update", ControlName: "action", Fields: map[string]any{"x": 10}, Control: map[string]any{"kind": "icon"}},
		"reorder with cascade": {Operation: "reorder", ControlName: "action", Index: &index, Cascade: true},
		"add with fields":      {Operation: "add", Control: map[string]any{"kind": "icon", "name": "n"}, Fields: map[string]any{"x": 1}},
	} {
		if _, _, err := applyUIPanelControlOperations(controls, []uiPanelOperation{op}); err == nil || !strings.Contains(err.Error(), "does not take") {
			t.Errorf("%s: err=%v", name, err)
		}
	}
	if _, _, err := applyUIPanelControlOperations(controls, []uiPanelOperation{
		{Operation: "move", ControlName: "action", ParentControlName: "body"},
		{Operation: "update", ControlName: "action", Fields: map[string]any{"x": float64(10)}},
	}); err != nil {
		t.Fatalf("move then update was refused: %v", err)
	}
}

func TestUIPanelCompiledReadbackMatchesWholeNames(t *testing.T) {
	panel := map[string]any{"token": "CC1_x", "controls": []any{
		map[string]any{"kind": "text", "name": "a"},
		map[string]any{"kind": "text", "name": "panel"},
	}}
	files := map[string]any{
		"gui_text": "guiTypes = {\n\tcontainerWindowType = {\n\t\tname = \"CC1_x_panel\"\n\t\tinstantTextBoxType = {\n\t\t\tname = \"CC1_x_ab\"\n\t\t}\n\t}\n}\n",
		"txt_text": "scripted_gui = {\n}\n",
	}
	var diagnostics []uiPanelDiagnostic
	if validateUIPanelCompiledReadback(panel, files, &diagnostics) {
		t.Fatal("controls the compiler never wrote passed the readback")
	}
	missing := 0
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "panel.compile_control_missing" {
			missing++
		}
	}
	if missing != 1 {
		t.Fatalf("want only control a reported missing (panel is the window's name, a is not ab): %#v", diagnostics)
	}
}

func TestUpdateUIPanelSendsTheNormalizedValues(t *testing.T) {
	var putBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			_ = json.NewDecoder(r.Body).Decode(&putBody)
		}
		_ = json.NewEncoder(w).Encode(uiPanelFixture())
	}))
	defer upstream.Close()
	session := connectClient(t, newUIPanelTestServer(t, upstream.URL).URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_ui_panel", Arguments: map[string]any{
			"project_id": 7, "panel_id": 4, "expected_updated_at": "2026-10-14T10:00:00Z",
			"window_props": map[string]any{"orientation": "center"},
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("update_ui_panel: err=%v result=%s", err, contentText(result))
	}
	props, _ := putBody["window_props"].(map[string]any)
	if props["orientation"] != "CENTER" || props["width"] != float64(400) {
		t.Fatalf("window_props not sent normalized and merged: %#v", props)
	}
}

func TestUIPanelValidationAcceptsScrollingContainers(t *testing.T) {
	codes := func(scroll any) map[string]bool {
		_, diagnostics := normalizeUIPanelControls([]map[string]any{{
			"kind": "container", "name": "list", "x": 0, "y": 0, "w": 200, "h": 120, "scroll": scroll,
			"children": []any{},
		}})
		out := map[string]bool{}
		for _, d := range diagnostics {
			out[d.Code] = true
		}
		return out
	}
	if got := codes(true); got["panel.control_field_unknown"] || got["panel.control_scroll"] {
		t.Fatalf("a scrolling container was refused: %v", got)
	}
	if got := codes("yes"); !got["panel.control_scroll"] {
		t.Fatalf("a non-boolean scroll was let through: %v", got)
	}
}

func TestUIPanelValidationAcceptsLists(t *testing.T) {
	codes := func(control map[string]any) map[string]bool {
		_, diagnostics := normalizeUIPanelControls([]map[string]any{control})
		out := map[string]bool{}
		for _, d := range diagnostics {
			out[d.Code] = true
		}
		return out
	}
	list := map[string]any{"kind": "container", "name": "items", "x": 0, "y": 0, "w": 200, "h": 150,
		"array": "my_items", "columns": 2, "slot_w": 100, "slot_h": 30, "change_scope": true, "children": []any{}}
	if got := codes(list); got["panel.control_field_unknown"] || got["panel.control_list"] || got["panel.control_array"] {
		t.Fatalf("a list was refused: %v", got)
	}
	bad := map[string]any{"kind": "container", "name": "items", "x": 0, "y": 0, "array": "my items!", "columns": 0, "children": []any{}}
	if got := codes(bad); !got["panel.control_array"] || !got["panel.control_list"] {
		t.Fatalf("a bad list was let through: %v", got)
	}
}
