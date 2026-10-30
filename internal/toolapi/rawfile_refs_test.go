package toolapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

type rawFileFixture struct {
	rawFiles    []map[string]any
	superEvents []map[string]any
	events      []map[string]any
	ideas       []map[string]any
	edges       []map[string]any
	entity      map[string]any
}

func (f rawFileFixture) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		list := func(rows []map[string]any) {
			if rows == nil {
				rows = []map[string]any{}
			}
			_ = json.NewEncoder(w).Encode(rows)
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/raw-files"):
			list(f.rawFiles)
		case strings.HasSuffix(r.URL.Path, "/super-events"):
			list(f.superEvents)
		case strings.HasSuffix(r.URL.Path, "/events"):
			list(f.events)
		case strings.HasSuffix(r.URL.Path, "/ideas"):
			list(f.ideas)
		case strings.HasSuffix(r.URL.Path, "/relations/graph"):
			edges := f.edges
			if edges == nil {
				edges = []map[string]any{}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"edges": edges})
		default:
			if f.entity != nil {
				_ = json.NewEncoder(w).Encode(f.entity)
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	}
}

func storyRegistry(body string) map[string]any {
	return map[string]any{
		"id": 31, "dest_path": "common/scripted_effects/zz_sanguo_story_registry_generated.txt",
		"is_binary": false, "content": body,
	}
}

func TestDeletePreviewFindsRawFileReferences(t *testing.T) {
	fixture := rawFileFixture{
		entity: map[string]any{
			"id": 12, "idea_id": "han_mandate",
			"display_name_locs": map[string]any{"en": "Mandate of Heaven"},
		},
		rawFiles: []map[string]any{storyRegistry(
			"grant = {\n    add_ideas = han_mandate\n}\n")},
	}
	session := genericReadSession(t, fixture.handler(t))
	result := callTool(t, session, "preview_delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 12,
	})
	if result.IsError {
		t.Fatalf("tool error: %+v", result.Content)
	}
	out := result.StructuredContent.(map[string]any)
	if out["safe_to_delete"] != false {
		t.Fatalf("a raw-file reference did not block the safe verdict: %v", out)
	}
	rawRefs, _ := out["referenced_by_raw_files"].([]any)
	if len(rawRefs) != 1 {
		t.Fatalf("referenced_by_raw_files = %v", out["referenced_by_raw_files"])
	}
	if out["reference_count"] != float64(1) {
		t.Errorf("reference_count = %v, must include raw-file references", out["reference_count"])
	}
}
