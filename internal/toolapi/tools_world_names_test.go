package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNameWorldMapForwardsOneBatch(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody struct {
		StateNames []worldMapNameInput `json:"state_names"`
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"updated": 2,
			"total":   2,
			"name_status": map[string]any{
				"complete": true,
			},
		})
	}))
	defer upstream.Close()

	srv := newProxyOnlyTestServer(t, func(s *Server) {
		registerNameWorldMapTool(s, NewRESTProxy(upstream.URL))
	})
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "name_world_map",
		Arguments: map[string]any{
			"project_id": 7,
			"state_names": []any{
				map[string]any{
					"state_id":          1,
					"display_name_locs": map[string]any{"en": "Winterfell", "zh": "临冬城"},
					"capital_name_locs": map[string]any{"en": "Winter Town", "zh": "冬季镇"},
				},
				map[string]any{
					"state_id":          2,
					"display_name_locs": map[string]any{"en": "The Vale", "zh": "谷地"},
				},
			},
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("CallTool err=%v result=%+v", err, result)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/gui-render/world-map/names" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	if len(gotBody.StateNames) != 2 {
		t.Fatalf("state_names = %+v", gotBody.StateNames)
	}
	if first := gotBody.StateNames[0]; first.StateID != 1 || first.DisplayNameLocs["zh"] != "临冬城" || first.CapitalNameLocs["en"] != "Winter Town" {
		t.Fatalf("first name = %+v", first)
	}
	if second := gotBody.StateNames[1]; second.StateID != 2 || second.DisplayNameLocs["en"] != "The Vale" || len(second.CapitalNameLocs) != 0 {
		t.Fatalf("second name = %+v", second)
	}
	out, ok := result.StructuredContent.(map[string]any)
	status, statusOK := out["name_status"].(map[string]any)
	if !ok || !statusOK || status["complete"] != true || out["updated"] != float64(2) {
		t.Fatalf("structured output = %#v", result.StructuredContent)
	}
}

func TestNameWorldMapRejectsEmptyBatchBeforeProxy(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer upstream.Close()

	srv := newProxyOnlyTestServer(t, func(s *Server) {
		registerNameWorldMapTool(s, NewRESTProxy(upstream.URL))
	})
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "name_world_map",
		Arguments: map[string]any{"project_id": 7, "state_names": []any{}},
	})
	if err != nil || !result.IsError {
		t.Fatalf("CallTool err=%v result=%+v", err, result)
	}
	if called {
		t.Fatal("empty names batch reached upstream")
	}
	if text := contentText(result); !strings.Contains(text, "at least one") {
		t.Fatalf("error text = %q", text)
	}
}
