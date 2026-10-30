package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetProjectOverview_MergesThreeUpstreamCalls(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/projects/7":
			json.NewEncoder(w).Encode(map[string]any{
				"id": 7, "name": "German Spring", "start_date": "1936.1.1", "end_date": "1956.1.1",
				"export_languages": []string{"en", "zh"}, "dependencies": []string{"base_mod"},
			})
		case "/api/projects/7/countries":
			json.NewEncoder(w).Encode([]map[string]any{{"tag": "GER", "name": "Germany"}})
		case "/api/projects/7/focus-trees":
			json.NewEncoder(w).Encode([]map[string]any{{"id": 3, "tree_id": "ger_focus", "country_tag": "GER"}})
		default:
			t.Errorf("unexpected upstream path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	srv := newAuthedTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_project_overview", Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if out["name"] != "German Spring" {
		t.Errorf("name = %v, want German Spring", out["name"])
	}
	if out["end_date"] != "1956.1.1" {
		t.Errorf("end_date = %v", out["end_date"])
	}
	languages := out["export_languages"].([]any)
	dependencies := out["dependencies"].([]any)
	if len(languages) != 2 || languages[1] != "zh" || len(dependencies) != 1 || dependencies[0] != "base_mod" {
		t.Errorf("languages=%v dependencies=%v", languages, dependencies)
	}
	countries := out["countries"].([]any)
	if len(countries) != 1 || countries[0].(map[string]any)["tag"] != "GER" {
		t.Errorf("countries = %v", countries)
	}
	trees := out["focus_trees"].([]any)
	if len(trees) != 1 || trees[0].(map[string]any)["tree_id"] != "ger_focus" {
		t.Errorf("focus_trees = %v", trees)
	}
}
