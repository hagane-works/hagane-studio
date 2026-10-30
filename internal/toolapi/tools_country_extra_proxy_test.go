package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateBookmark_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/bookmarks" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 15, "bookmark_id": "the_modern_usa_arrived"})
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerCreateBookmark)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_bookmark",
		Arguments: map[string]any{
			"project_id": 7, "bookmark_id": "the_modern_usa_arrived", "name_zh": "现代美国到来",
			"desc_zh": "新的时代开始了", "display_name_locs": map[string]any{"zh": "现代美国到来"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotAuth != "" {
		t.Errorf("upstream saw Authorization = %q; the tools forward no credential", gotAuth)
	}
	if gotBody["date"] != "1936.1.1.12" {
		t.Errorf("upstream saw date = %v, want the BookmarkCreate default", gotBody["date"])
	}
	if gotBody["picture"] != "GFX_select_date_1936" {
		t.Errorf("upstream saw picture = %v, want the BookmarkCreate default", gotBody["picture"])
	}
	if gotBody["name_zh"] != "现代美国到来" || gotBody["desc_zh"] != "新的时代开始了" {
		t.Errorf("bookmark localisation = %#v", gotBody)
	}
	out := result.StructuredContent.(map[string]any)
	if out["bookmark_db_id"] != float64(15) {
		t.Errorf("bookmark_db_id = %v, want 15", out["bookmark_db_id"])
	}
}

func TestCreateBookmark_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "bookmark_id 'the_modern_usa_arrived' already exists in this project. Pick a unique bookmark_id."})
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerCreateBookmark)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_bookmark",
		Arguments: map[string]any{"project_id": 7, "bookmark_id": "the_modern_usa_arrived"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "already exists in this project") {
		t.Errorf("error text = %q, want the upstream's own detail message relayed", contentText(result))
	}
}

func TestSetCountryHistory_CreatesWhenNoExistingRow(t *testing.T) {
	var postBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/projects/12/history-files":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/projects/12/history-files":
			_ = json.NewDecoder(r.Body).Decode(&postBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 41, "country_tag": "ROM"})
		default:
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerSetCountryHistory)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "set_country_history",
		Arguments: map[string]any{"project_id": 12, "country_tag": "rom", "set_research_slots": 3},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if postBody["country_tag"] != "ROM" {
		t.Errorf("upstream saw country_tag = %v, want it uppercased to ROM", postBody["country_tag"])
	}
	if postBody["set_research_slots"] != float64(3) {
		t.Errorf("upstream saw set_research_slots = %v, want 3", postBody["set_research_slots"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["created"] != true {
		t.Errorf("created = %v, want true (no existing row found by the GET lookup)", out["created"])
	}
	if out["history_file_id"] != float64(41) {
		t.Errorf("history_file_id = %v, want 41", out["history_file_id"])
	}
}

func TestSetCountryHistory_UpdatesWhenExistingRowFound(t *testing.T) {
	var putBody map[string]any
	var putPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/projects/12/history-files":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 7, "country_tag": "ROM"},
			})
		case r.Method == http.MethodPut:
			putPath = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "country_tag": "ROM"})
		default:
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerSetCountryHistory)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "set_country_history",
		Arguments: map[string]any{"project_id": 12, "country_tag": "ROM", "set_stability": 0.6},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if putPath != "/api/projects/12/history-files/7" {
		t.Errorf("PUT path = %q, want /api/projects/12/history-files/7 (the row the GET lookup found)", putPath)
	}
	if putBody["set_stability"] != 0.6 {
		t.Errorf("upstream saw set_stability = %v, want 0.6", putBody["set_stability"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["created"] != false {
		t.Errorf("created = %v, want false (existing row found)", out["created"])
	}
	if out["history_file_id"] != float64(7) {
		t.Errorf("history_file_id = %v, want 7", out["history_file_id"])
	}
}

func TestSetCountryHistory_ForwardsCompletePoliticsDiplomacyAndOOBSurface(t *testing.T) {
	var got map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 7, "country_tag": "CHI"}})
			return
		}
		if r.Method != http.MethodPut || r.URL.Path != "/api/projects/12/history-files/7" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "country_tag": "CHI"})
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerSetCountryHistory)
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "set_country_history",
		Arguments: map[string]any{
			"project_id": 12, "country_tag": "CHI",
			"set_politics_ruling_party": "neutrality", "set_politics_last_election": "1932.1.1",
			"set_politics_election_frequency": 48, "set_politics_elections_allowed": false,
			"set_popularities": map[string]any{"neutrality": 70, "democratic": 30},
			"set_convoys":      25, "set_air_oob": "CHI_1936_air", "set_naval_oob": "CHI_1936_naval",
			"startup_flags": []any{"chi_ready"}, "give_guarantee_tags": []any{"MON"},
			"add_to_faction_tags": []any{},
			"set_autonomy":        []any{map[string]any{"target": "PRC", "autonomy_state": "autonomy_puppet"}},
			"diplomatic_relation": []any{map[string]any{"country": "USA", "relation": "non_aggression_pact", "active": true}},
			"date_blocks":         []any{map[string]any{"date": "1937.7.7", "script": "set_country_flag = marco_polo"}},
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("CallTool err=%v result=%+v", err, result)
	}
	for _, key := range []string{
		"set_politics_last_election", "set_politics_election_frequency", "set_politics_elections_allowed",
		"set_popularities", "set_convoys", "set_air_oob", "set_naval_oob", "startup_flags",
		"give_guarantee_tags", "add_to_faction_tags", "set_autonomy", "diplomatic_relation", "date_blocks",
	} {
		if _, ok := got[key]; !ok {
			t.Errorf("complete country-history payload dropped %q: %+v", key, got)
		}
	}
	if got["set_politics_elections_allowed"] != false || got["set_air_oob"] != "CHI_1936_air" {
		t.Errorf("boolean/OOB fields changed in transit: %+v", got)
	}
	if tags, ok := got["add_to_faction_tags"].([]any); !ok || len(tags) != 0 {
		t.Errorf("explicit empty faction list was not preserved: %#v", got["add_to_faction_tags"])
	}
}

func TestSetCountryHistory_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"detail": "history_file for country 'ROM' already exists in this project. PUT to update or DELETE first.",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerSetCountryHistory)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "set_country_history",
		Arguments: map[string]any{"project_id": 12, "country_tag": "ROM"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "already exists in this project") {
		t.Errorf("error text = %q, want the upstream's own detail message relayed", contentText(result))
	}
}
