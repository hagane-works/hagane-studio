package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func characterUpstream(t *testing.T, response map[string]any) (*httptest.Server, *string, *map[string]any) {
	t.Helper()
	var gotRoute string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRoute = r.Method + " " + r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(upstream.Close)
	return upstream, &gotRoute, &gotBody
}

func TestCreateCharacter_ForwardsWithDefaults(t *testing.T) {
	upstream, gotRoute, gotBody := characterUpstream(t, map[string]any{
		"id": 99, "tag": "ENG_test_char", "role_type": "general", "has_advisor_role": false,
	})
	srv := newAuthedTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_character",
		Arguments: map[string]any{
			"project_id": 7, "name": "Test Character", "country_tag": "eng",
			"traits": []string{"silent_workhorse"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if out["character_id"] != float64(99) {
		t.Errorf("character_id = %v, want 99", out["character_id"])
	}
	if *gotRoute != "POST /api/projects/7/characters" {
		t.Errorf("upstream saw %s", *gotRoute)
	}
	body := *gotBody
	for key, want := range map[string]any{"role_type": "general", "skill": float64(1), "advisor_cost": float64(150), "can_be_captured": true} {
		if body[key] != want {
			t.Errorf("body[%s] = %v, want %v (default)", key, body[key], want)
		}
	}
}

func TestCreateCharacter_RouteWarningsSurface(t *testing.T) {
	upstream, gotRoute, gotBody := characterUpstream(t, map[string]any{
		"id": 55, "tag": "GER_reroute_test_1", "role_type": "general", "has_advisor_role": false,
		"warnings": []string{"'GER_reroute_test' already exists in TNO base · rerouted to 'GER_reroute_test_1'."},
	})
	srv := newAuthedTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_character",
		Arguments: map[string]any{
			"project_id": 9, "name": "Reroute Test", "tag": "GER_reroute_test", "country_tag": "GER",
		},
	})
	if err != nil {
		t.Fatalf("CallTool transport error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", contentText(result))
	}
	if *gotRoute != "POST /api/projects/9/characters" {
		t.Errorf("upstream saw %s, want POST /api/projects/9/characters", *gotRoute)
	}
	if (*gotBody)["tag"] != "GER_reroute_test" {
		t.Errorf("upstream body[tag] = %v, want the caller's requested tag", (*gotBody)["tag"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["character_id"] != float64(55) {
		t.Errorf("character_id = %v, want 55 (from the route's response)", out["character_id"])
	}
	warnings, _ := out["warnings"].([]any)
	if len(warnings) != 1 || !strings.Contains(warnings[0].(string), "rerouted") {
		t.Errorf("warnings = %v, want the base-mod collision reroute notice", out["warnings"])
	}
}

func TestCreateCharacter_DisplayNameLocsRoundTrip(t *testing.T) {
	upstream, _, gotBody := characterUpstream(t, map[string]any{"id": 1, "tag": "T", "role_type": "general"})
	srv := newAuthedTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	session := connectClient(t, srv.URL)

	args := map[string]any{"project_id": 1, "name": "X", "display_name_locs": map[string]string{"en": "Hello", "zh": "你好", "pl": "Cześć"}}
	if _, err := session.CallTool(context.Background(), &CallToolParams{Name: "create_character", Arguments: args}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	locs, _ := (*gotBody)["display_name_locs"].(map[string]any)
	if locs["zh"] != "你好" || locs["pl"] != "Cześć" {
		t.Errorf("display_name_locs = %v, want zh and pl kept (ContentLocale includes pl/pt)", locs)
	}
}
