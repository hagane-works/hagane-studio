package toolapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAIPlanningGetEntityReadsExactRowFromListOnlyAPI(t *testing.T) {
	for _, tc := range []struct{ kind, path, wrapper, key string }{
		{"ai_strategy_plan", "ai-strategy-plans", "ai_strategy_plans", "plan_id"},
		{"ai_template", "ai-templates", "ai_templates", "template_id"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			requests := 0
			missingProject := false
			session := genericReadSession(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				if missingProject {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"detail":"Project not found"}`))
					return
				}
				if r.Method != http.MethodGet || r.URL.Path != "/api/projects/296/"+tc.path {
					w.WriteHeader(http.StatusMethodNotAllowed)
					_, _ = w.Write([]byte(`{"error":"PUT or DELETE only"}`))
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{tc.wrapper: []map[string]any{
					{"id": 41, tc.key: "other", "raw_block": "unrelated row"},
					{"id": 42, tc.key: "wanted", "raw_block": "complete saved blocks", "source": "imported"},
				}})
			})
			result := callTool(t, session, "get_entity", map[string]any{"project_id": 296, "kind": tc.kind, "id": 42})
			if result.IsError {
				t.Fatalf("list-only get_entity failed: %s", contentText(result))
			}
			entity := result.StructuredContent.(map[string]any)["entity"].(map[string]any)
			if entity["id"] != float64(42) || entity[tc.key] != "wanted" || entity["raw_block"] != "complete saved blocks" || entity["source"] != "imported" {
				t.Fatalf("get_entity lost the exact complete row: %#v", entity)
			}
			missing := callTool(t, session, "get_entity", map[string]any{"project_id": 296, "kind": tc.kind, "id": 99})
			if !missing.IsError || !strings.Contains(contentText(missing), "id 99") || strings.Contains(contentText(missing), "unrelated row") {
				t.Fatalf("missing row must be refused, never substituted: %s", contentText(missing))
			}
			missingProject = true
			gone := callTool(t, session, "get_entity", map[string]any{"project_id": 296, "kind": tc.kind, "id": 42})
			if !gone.IsError || strings.Contains(contentText(gone), "complete saved blocks") {
				t.Fatalf("upstream 404 must not return a previous or unrelated row: %s", contentText(gone))
			}
			if requests != 3 {
				t.Fatalf("upstream requests = %d, want exactly one list per read", requests)
			}
		})
	}
}

func TestAIPlanningFieldReferencesAreListedAndReadable(t *testing.T) {
	index := buildFieldsIndex()
	for _, stem := range []string{"ai-strategy-plans", "ai-templates"} {
		if !strings.Contains(index, "`"+stem+"`") {
			t.Fatalf("field index does not advertise %s", stem)
		}
		doc, err := readFieldDoc(stem)
		if err != nil || !strings.Contains(doc, "get_entity") || !strings.Contains(doc, "complete") || !strings.Contains(doc, "list-only") {
			t.Fatalf("field doc does not explain the real input/readback contract: %s", stem)
		}
	}
}
