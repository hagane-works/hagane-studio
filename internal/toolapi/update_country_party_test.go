package toolapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestUpdateCountryWritesARulingPartyChangeToTheStartHistory(t *testing.T) {
	type call struct {
		method, path string
		body         map[string]any
	}
	countries := []map[string]any{
		{"id": 3, "tag": "GER", "ideology": "neutrality", "auto_added": false},
		{"id": 4, "tag": "KRX", "ideology": "neutrality", "auto_added": true},
		{"id": 5, "tag": "ZQX", "ideology": "neutrality", "auto_added": false},
	}
	run := func(t *testing.T, args map[string]any, history []map[string]any) ([]call, map[string]any) {
		var calls []call
		session := updateGapSession(t, func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if r.Method != http.MethodGet {
				calls = append(calls, call{r.Method, r.URL.Path, body})
			}
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/api/projects/296/countries":
				_ = json.NewEncoder(w).Encode(map[string]any{"countries": countries, "items": countries, "total": len(countries)})
			case r.Method == http.MethodGet && r.URL.Path == "/api/projects/296/history-files":
				_ = json.NewEncoder(w).Encode(history)
			case strings.HasPrefix(r.URL.Path, "/api/projects/296/countries/"):
				_ = json.NewEncoder(w).Encode(map[string]any{"id": args["id"], "tag": "GER"})
			case strings.HasPrefix(r.URL.Path, "/api/projects/296/history-files"):
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 41, "country_tag": body["country_tag"]})
			default:
				t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		})
		result := callTool(t, session, "update_country", args)
		if result.IsError {
			t.Fatalf("tool error: %+v", result.Content)
		}
		out, _ := result.StructuredContent.(map[string]any)
		return calls, out
	}

	t.Run("base-game country: the party goes to a new start-history row", func(t *testing.T) {
		calls, out := run(t, map[string]any{"project_id": 296, "id": 3, "ideology": "communism"}, nil)
		if len(calls) != 2 || calls[0].path != "/api/projects/296/countries/3" || calls[0].body["ideology"] != "communism" {
			t.Fatalf("calls = %+v", calls)
		}
		if calls[1].method != http.MethodPost || calls[1].path != "/api/projects/296/history-files" ||
			calls[1].body["country_tag"] != "GER" || calls[1].body["set_politics_ruling_party"] != "communism" || len(calls[1].body) != 2 {
			t.Fatalf("history write = %+v", calls[1])
		}
		party, _ := out["ruling_party"].(map[string]any)
		if party["written"] != true || party["created"] != true || party["history_file_id"] != float64(41) {
			t.Fatalf("ruling_party = %v", party)
		}
	})

	t.Run("imported country: its existing start-history row is updated", func(t *testing.T) {
		calls, _ := run(t, map[string]any{"project_id": 296, "id": 4, "ideology": "fascism"},
			[]map[string]any{{"id": 77, "country_tag": "KRX"}})
		if len(calls) != 2 || calls[1].method != http.MethodPut || calls[1].path != "/api/projects/296/history-files/77" ||
			calls[1].body["set_politics_ruling_party"] != "fascism" {
			t.Fatalf("calls = %+v", calls)
		}
	})

	t.Run("the row's own value echoed back leaves the start history alone", func(t *testing.T) {
		calls, out := run(t, map[string]any{"project_id": 296, "id": 3, "name": "Reich", "ideology": "neutrality"}, nil)
		if len(calls) != 1 {
			t.Fatalf("the history was written: %+v", calls)
		}
		party, _ := out["ruling_party"].(map[string]any)
		if party["written"] != false || !strings.Contains(party["note"].(string), "set_country_history") {
			t.Fatalf("ruling_party = %v", party)
		}
	})

	t.Run("a country the project made keeps its party on the row", func(t *testing.T) {
		calls, out := run(t, map[string]any{"project_id": 296, "id": 5, "ideology": "communism"}, nil)
		if len(calls) != 1 || calls[0].body["ideology"] != "communism" {
			t.Fatalf("calls = %+v", calls)
		}
		if _, has := out["ruling_party"]; has {
			t.Fatalf("ruling_party reported for a project-made country: %v", out)
		}
	})

	t.Run("no ideology: no read, no history", func(t *testing.T) {
		calls, out := run(t, map[string]any{"project_id": 296, "id": 3, "name": "Reich"}, nil)
		if len(calls) != 1 {
			t.Fatalf("calls = %+v", calls)
		}
		if _, has := out["ruling_party"]; has {
			t.Fatalf("ruling_party reported without an ideology: %v", out)
		}
	})
}
