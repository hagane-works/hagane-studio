package toolapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newCharactersTestServer(t *testing.T, proxyURL string) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}

	getServer := func(*http.Request) *Server {
		server := NewServer()
		proxy := NewRESTProxy(proxyURL)
		registerGetCharacters(server, proxy)
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

func fakeCharactersUpstream(t *testing.T, wantPath string, rows []map[string]any) (*httptest.Server, *string) {
	t.Helper()
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected upstream path: %s, want %s", r.URL.Path, wantPath)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rows)
	}))
	t.Cleanup(srv.Close)
	return srv, &gotQuery
}

func TestGetCharacters_NoFilter_ForwardsBareList(t *testing.T) {
	upstream, gotQuery := fakeCharactersUpstream(t, "/api/projects/7/characters", []map[string]any{
		{"id": 3, "tag": "GER_hitler", "name": "Adolf Hitler", "country_tag": "GER",
			"role_type": "country_leader", "has_advisor_role": false},
		{"id": 4, "tag": "GER_goring", "name": "Hermann Goring", "country_tag": "GER",
			"role_type": "high_command", "has_advisor_role": true},
	})

	srv := newCharactersTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_characters", Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", contentText(result))
	}
	if *gotQuery != "" {
		t.Errorf("upstream query = %q, want empty (no country_tag filter)", *gotQuery)
	}

	out := result.StructuredContent.(map[string]any)
	if out["project_id"] != float64(7) {
		t.Errorf("project_id = %v, want 7", out["project_id"])
	}
	chars := out["characters"].([]any)
	if len(chars) != 2 {
		t.Fatalf("len(characters) = %d, want 2", len(chars))
	}
	first := chars[0].(map[string]any)
	if first["character_id"] != float64(3) {
		t.Errorf("character_id = %v, want 3 (renamed from REST's 'id')", first["character_id"])
	}
	if first["tag"] != "GER_hitler" || first["name"] != "Adolf Hitler" || first["country_tag"] != "GER" {
		t.Errorf("first character = %+v, want GER_hitler/Adolf Hitler/GER", first)
	}
	if first["role_type"] != "country_leader" {
		t.Errorf("role_type = %v, want country_leader", first["role_type"])
	}
	if first["has_advisor_role"] != false {
		t.Errorf("has_advisor_role = %v, want false", first["has_advisor_role"])
	}
	second := chars[1].(map[string]any)
	if second["has_advisor_role"] != true {
		t.Errorf("has_advisor_role = %v, want true", second["has_advisor_role"])
	}
}

func TestGetCharacters_WithCountryTag_ForwardsUppercasedQueryParam(t *testing.T) {
	upstream, gotQuery := fakeCharactersUpstream(t, "/api/projects/7/characters", []map[string]any{
		{"id": 5, "tag": "ita_deb", "name": "Some Advisor", "country_tag": "ITA",
			"role_type": "advisor", "has_advisor_role": true},
	})

	srv := newCharactersTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_characters", Arguments: map[string]any{"project_id": 7, "country_tag": "ita"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", contentText(result))
	}
	if *gotQuery != "country_tag=ITA" {
		t.Errorf("upstream query = %q, want country_tag=ITA (uppercased, case-insensitive parity "+
			"with _characters_payload's country_tag.upper() filter)", *gotQuery)
	}

	out := result.StructuredContent.(map[string]any)
	chars := out["characters"].([]any)
	if len(chars) != 1 || chars[0].(map[string]any)["country_tag"] != "ITA" {
		t.Errorf("characters = %v, want the single ITA row", chars)
	}
}
