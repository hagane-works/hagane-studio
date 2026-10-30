package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type guiTextTestUpstream struct {
	base      map[string]string
	overrides map[string]string
	called    int
}

func (u *guiTextTestUpstream) serveHTTP(w http.ResponseWriter, r *http.Request) {
	u.called++
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/projects/7/gui-render/loc":
		texts := map[string]string{}
		for _, key := range strings.Split(r.URL.Query().Get("keys"), ",") {
			if value, ok := u.overrides[key]; ok {
				texts[key] = value
			} else if value, ok := u.base[key]; ok {
				texts[key] = value
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"texts": texts})
	case r.Method == http.MethodGet && r.URL.Path == "/api/projects/7/gui-render/text":
		_ = json.NewEncoder(w).Encode(map[string]any{"overrides": u.overrides})
	case r.Method == http.MethodPost && r.URL.Path == "/api/projects/7/gui-render/text":
		var in struct {
			Key, Value, Lang string
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if in.Lang != "en" {
			http.Error(w, "wrong language", http.StatusUnprocessableEntity)
			return
		}
		u.overrides[in.Key] = in.Value
		_ = json.NewEncoder(w).Encode(map[string]any{"key": in.Key, "value": in.Value})
	case r.Method == http.MethodDelete && r.URL.Path == "/api/projects/7/gui-render/text":
		key := r.URL.Query().Get("key")
		_, removed := u.overrides[key]
		delete(u.overrides, key)
		_ = json.NewEncoder(w).Encode(map[string]any{"key": key, "removed": removed})
	default:
		http.Error(w, "unexpected request "+r.Method+" "+r.URL.RequestURI(), http.StatusNotFound)
	}
}

func newGUITextToolSession(t *testing.T, upstream http.Handler, register func(*Server, *RESTProxy)) (*testSession, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(upstream)
	t.Cleanup(server.Close)
	proxy := NewRESTProxy(server.URL)
	toolServer := newProxyOnlyTestServer(t, func(s *Server) { register(s, proxy) })
	return connectClient(t, toolServer.URL), toolServer
}

func TestGetGUITextsReadsExactRenderedValuesAndOverrides(t *testing.T) {
	upstream := &guiTextTestUpstream{
		base: map[string]string{
			"KOR_TITLE": "§YOriginal title§!",
			"KOR_DESC":  "Original $TOKEN$",
		},
		overrides: map[string]string{"KOR_DESC": "§RChanged $TOKEN$§!"},
	}
	session, _ := newGUITextToolSession(t, http.HandlerFunc(upstream.serveHTTP), func(s *Server, proxy *RESTProxy) {
		registerGetGUITexts(s, proxy)
	})
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_gui_texts",
		Arguments: map[string]any{
			"project_id": 7, "keys": []any{" KOR_DESC ", "KOR_TITLE", "KOR_DESC"},
			"language": "en",
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("CallTool err=%v result=%+v", err, result)
	}
	out := result.StructuredContent.(map[string]any)
	if out["target_kind"] != guiLocalisationTargetKind || out["language"] != "en" {
		t.Fatalf("target/language = %v/%v", out["target_kind"], out["language"])
	}
	if out["resource_file"] != guiTextResourceFile() || out["resource_file"] == "" {
		t.Fatalf("resource_file = %v", out["resource_file"])
	}
	rows := out["texts"].([]any)
	if len(rows) != 2 {
		t.Fatalf("texts = %#v", rows)
	}
	first := rows[0].(map[string]any)
	if first["key"] != "KOR_DESC" || first["value"] != "§RChanged $TOKEN$§!" ||
		first["resolved_value"] != "§RChanged $TOKEN$§!" || first["override_value"] != "§RChanged $TOKEN$§!" ||
		first["found"] != true || first["resolved"] != true || first["overridden"] != true {
		t.Fatalf("first = %#v", first)
	}
	second := rows[1].(map[string]any)
	if second["key"] != "KOR_TITLE" || second["value"] != "§YOriginal title§!" || second["overridden"] != false {
		t.Fatalf("second = %#v", second)
	}
	if _, present := second["override_value"]; present {
		t.Fatalf("non-overridden text returned override_value: %#v", second)
	}
}

func TestUpdateGUITextsWritesBatchAndVerifiesEveryKey(t *testing.T) {
	upstream := &guiTextTestUpstream{
		base: map[string]string{
			"KOR_TITLE": "§YHello§! $TOKEN$ £army_experience",
			"KOR_DESC":  "Old description",
			"KOR_CLEAR": "Vanilla restored",
		},
		overrides: map[string]string{"KOR_CLEAR": "Temporary override"},
	}
	session, _ := newGUITextToolSession(t, http.HandlerFunc(upstream.serveHTTP), func(s *Server, proxy *RESTProxy) {
		registerUpdateGUITexts(s, proxy)
	})
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_gui_texts",
		Arguments: map[string]any{
			"project_id": 7,
			"updates": []any{
				map[string]any{"key": "KOR_TITLE", "operation": "set_color", "color": "§R"},
				map[string]any{"key": "KOR_DESC", "operation": "set_value", "value": "New §Gdescription§!"},
				map[string]any{"key": "KOR_CLEAR", "operation": "clear"},
			},
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("CallTool err=%v result=%+v", err, result)
	}
	wantColor := "§RHello $TOKEN$ £army_experience§!"
	if upstream.overrides["KOR_TITLE"] != wantColor || upstream.overrides["KOR_DESC"] != "New §Gdescription§!" {
		t.Fatalf("saved overrides = %#v", upstream.overrides)
	}
	if _, present := upstream.overrides["KOR_CLEAR"]; present {
		t.Fatalf("clear left override: %#v", upstream.overrides)
	}
	out := result.StructuredContent.(map[string]any)
	if out["resource_file"] != guiTextResourceFile() || out["resource_file"] == "" {
		t.Fatalf("resource_file = %v", out["resource_file"])
	}
	if out["requested"] != float64(3) || out["applied"] != float64(3) || out["updated"] != float64(3) || out["failed"] != float64(0) {
		t.Fatalf("counts = %#v", out)
	}
	rows := out["results"].([]any)
	color := rows[0].(map[string]any)
	if color["ok"] != true || color["readback_verified"] != true || color["saved_value"] != wantColor ||
		color["saved_override_value"] != wantColor || color["overridden"] != true || color["target_kind"] != guiLocalisationTargetKind {
		t.Fatalf("color row = %#v", color)
	}
	cleared := rows[2].(map[string]any)
	if cleared["ok"] != true || cleared["saved_value"] != "Vanilla restored" || cleared["overridden"] != false || cleared["changed"] != true {
		t.Fatalf("clear row = %#v", cleared)
	}
}

func TestUpdateGUITextsRejectsInvalidBatchBeforeWriting(t *testing.T) {
	upstream := &guiTextTestUpstream{base: map[string]string{"KOR_TITLE": "Title"}, overrides: map[string]string{}}
	session, _ := newGUITextToolSession(t, http.HandlerFunc(upstream.serveHTTP), func(s *Server, proxy *RESTProxy) {
		registerUpdateGUITexts(s, proxy)
	})
	for _, arguments := range []map[string]any{
		{"project_id": 7, "language": "zh", "updates": []any{map[string]any{"key": "KOR_TITLE", "operation": "clear"}}},
		{"project_id": 7, "updates": []any{map[string]any{"key": "guessed key", "operation": "clear"}}},
		{"project_id": 7, "updates": []any{
			map[string]any{"key": "KOR_TITLE", "operation": "clear"},
			map[string]any{"key": "KOR_TITLE", "operation": "clear"},
		}},
		{"project_id": 7, "updates": []any{map[string]any{"key": "KOR_TITLE", "operation": "set_color", "color": "pink"}}},
	} {
		before := upstream.called
		result, err := session.CallTool(context.Background(), &CallToolParams{Name: "update_gui_texts", Arguments: arguments})
		if err != nil || !result.IsError {
			t.Fatalf("arguments=%#v err=%v result=%+v", arguments, err, result)
		}
		if upstream.called != before {
			t.Fatalf("invalid batch reached upstream: %#v", arguments)
		}
	}
}

func TestReadGUITextStateChunksOneHundredKeysForLocEndpoint(t *testing.T) {
	locCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/loc") {
			locCalls++
			keys := strings.Split(r.URL.Query().Get("keys"), ",")
			if len(keys) > guiLocEndpointBatchLimit || r.URL.Query().Get("markup") != "1" {
				t.Fatalf("loc query = %s", r.URL.RawQuery)
			}
			texts := map[string]string{}
			for _, key := range keys {
				texts[key] = key
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"texts": texts})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"overrides": map[string]string{}})
	}))
	defer upstream.Close()
	keys := make([]string, guiTextBatchLimit)
	for index := range keys {
		keys[index] = "KEY_" + strings.Repeat("X", index/10) + string(rune('A'+index%10))
	}
	state, err := readGUITextState(context.Background(), NewRESTProxy(upstream.URL), 7, keys)
	if err != nil || len(state.Effective) != guiTextBatchLimit || locCalls != 2 {
		t.Fatalf("state=%d locCalls=%d err=%v", len(state.Effective), locCalls, err)
	}
}

func TestUpdateGUITextsIsMutationReceiptCompatible(t *testing.T) {
	if !isMutationToolName("update_gui_texts") {
		t.Fatal("update_gui_texts is not classified as a mutation")
	}
	fields := map[string]any{
		"applied": float64(2), "updated": float64(1), "failed": float64(1),
	}
	receipt := buildMutationReceipt("update_gui_texts", 7, fields)
	if receipt.WriteStatus != "partially_committed" || receipt.SavedValuesSource != "tool_result" {
		t.Fatalf("receipt = %+v", receipt)
	}
}
