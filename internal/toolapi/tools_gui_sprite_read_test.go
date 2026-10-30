package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetGUISpritesPreservesExactSelectionOrderAndMetadata(t *testing.T) {
	var gotNames string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/projects/7/gui-render/sprites" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		gotNames = r.URL.Query().Get("names")
		if r.URL.Query().Get("include_usage") != "true" {
			t.Error("the tool did not request functional usage evidence")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"usage_evidence": map[string]any{"GFX_second": map[string]any{"note": "exact runtime condition unknown", "frame_base": 0, "frame_usages": []map[string]any{{"frame": 0, "key": "unselected"}, {"frame": 1, "key": "selected"}, {"frame": 2, "key": "unselected_2"}}}}, "sprites": map[string]any{
			"GFX_second": map[string]any{"resolved": true, "overridden": true, "width": 120, "height": 24, "frames": 3, "texturefile": "gfx/interface/second.dds"},
			"GFX_first":  map[string]any{"resolved": true, "overridden": false, "width": 64, "height": 64, "frames": 1, "texturefile": "gfx/interface/first.dds"},
		}})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetGUISprites(s, proxy) })
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "get_gui_sprites",
		Arguments: map[string]any{"project_id": 7, "names": []any{" GFX_second ", "GFX_first", "GFX_second"}, "include_usage": true, "selected_frames": map[string]any{"GFX_second": []int{1}}},
	})
	if err != nil || result.IsError {
		t.Fatalf("CallTool err=%v result=%+v", err, result)
	}
	if gotNames != "GFX_second,GFX_first" {
		t.Fatalf("names query = %q", gotNames)
	}
	out := result.StructuredContent.(map[string]any)
	rows := out["sprites"].([]any)
	first := rows[0].(map[string]any)
	if first["name"] != "GFX_second" || first["overridden"] != true || first["frames"] != float64(3) || first["preview_url"] != nil {
		t.Fatalf("first sprite = %#v", first)
	}
	if first["usage_evidence"].(map[string]any)["note"] != "exact runtime condition unknown" {
		t.Fatal("usage evidence lost in the reshape")
	}
	frames := first["usage_evidence"].(map[string]any)["frame_usages"].([]any)
	if len(frames) != 1 || frames[0].(map[string]any)["frame"] != float64(1) || frames[0].(map[string]any)["key"] != "selected" {
		t.Fatalf("selected frame meanings expanded to other atlas frames: %+v", frames)
	}
}

func TestGetGUISpritesDefaultDoesNotReadOrReturnUsage(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Has("include_usage") {
			t.Error("ordinary image lookup requested unnecessary usage indexing")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"sprites": map[string]any{"GFX_frame": map[string]any{"frames": 1, "width": 64, "height": 64}}, "usage_evidence": map[string]any{"GFX_frame": map[string]any{"frame_usages": []map[string]any{{"frame": 0, "key": "unrequested"}}}}})
	}))
	defer upstream.Close()
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetGUISprites(s, NewRESTProxy(upstream.URL)) })
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{Name: "get_gui_sprites", Arguments: map[string]any{"project_id": 7, "names": []string{"GFX_frame"}}})
	if err != nil || result.IsError || calls != 1 {
		t.Fatalf("metadata lookup failed: %+v %v calls=%d", result, err, calls)
	}
	row := result.StructuredContent.(map[string]any)["sprites"].([]any)[0].(map[string]any)
	if _, present := row["usage_evidence"]; present {
		t.Fatalf("ordinary lookup returned unrequested usage evidence: %+v", row)
	}
}

func TestGetGUISpritesUsageWithoutFrameScopeDoesNotDumpAtlas(t *testing.T) {
	usage := map[string]any{"note": "unknown live conditions", "references": []any{map[string]any{"name_path": "alerts/icon"}}, "frame_usages": []any{map[string]any{"frame": float64(0)}, map[string]any{"frame": float64(69)}}}
	filtered := selectedGUISpriteUsage(usage, nil)
	if len(filtered["frame_usages"].([]any)) != 0 || filtered["note"] != "unknown live conditions" || len(usage["frame_usages"].([]any)) != 2 {
		t.Fatalf("missing scope expanded atlas or mutated cached evidence: %+v", filtered)
	}
}

func TestGetGUISpritesRejectsMalformedFrameScopeBeforeHTTP(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer upstream.Close()
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetGUISprites(s, NewRESTProxy(upstream.URL)) })
	session := connectClient(t, srv.URL)
	for _, frames := range []map[string][]int{{"GFX_other": {0}}, {"GFX_alert": {}}, {"GFX_alert": {-1}}, {"GFX_alert": {1, 1}}} {
		result, err := session.CallTool(context.Background(), &CallToolParams{Name: "get_gui_sprites", Arguments: map[string]any{"project_id": 7, "names": []string{"GFX_alert"}, "include_usage": true, "selected_frames": frames}})
		if err != nil || !result.IsError || called {
			t.Fatalf("bad scope reached HTTP: %+v err=%v result=%+v called=%v", frames, err, result, called)
		}
	}
}

func TestGetGUISpritesRejectsOversizedSelectionBeforeHTTP(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer upstream.Close()
	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerGetGUISprites(s, proxy) })
	session := connectClient(t, srv.URL)
	names := make([]any, 101)
	for index := range names {
		names[index] = "GFX_" + strings.Repeat("x", index%5+1)
	}
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_gui_sprites", Arguments: map[string]any{"project_id": 7, "names": names},
	})
	if err != nil || !result.IsError || called {
		t.Fatalf("err=%v isError=%v upstreamCalled=%v", err, result.IsError, called)
	}
}
