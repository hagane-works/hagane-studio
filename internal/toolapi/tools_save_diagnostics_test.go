package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGateParityToolDiagnosticsTransport(t *testing.T) {
	for _, mode := range []string{"absent", "empty", "finding"} {
		t.Run(mode, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body := map[string]any{"id": 56, "focus_id": "GER_draft"}
				if mode != "absent" {
					body["diagnostics"] = []any{}
					body["tree_diagnostics"] = []any{}
				}
				if mode == "finding" {
					body["diagnostics"] = []any{map[string]any{
						"key": "overlap:56", "code": "focus_layout_same_cell", "severity": "warning",
						"entity_ref":   map[string]any{"type": "focus_node", "id": "56"},
						"evidence":     map[string]any{"focus_ids": []string{"GER_draft", "GER_other"}},
						"repair_route": "focus-tree", "affected": []string{"GER_draft", "GER_other"}, "change": "introduced",
					}}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(body)
			}))
			defer upstream.Close()
			server := newFocusProxyTestServer(t, func(host string) Deps {
				return Deps{Proxy: NewRESTProxy(upstream.URL)}
			})
			session := connectClient(t, server.URL)
			result, err := session.CallTool(context.Background(), &CallToolParams{Name: "create_focus", Arguments: map[string]any{"project_id": 7, "focus_tree_id": 21, "name": "Draft", "focus_id": "GER_draft"}})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("tool error: %+v", result.Content)
			}
			out := result.StructuredContent.(map[string]any)
			for _, field := range []string{"diagnostics", "tree_diagnostics"} {
				value, present := out[field]
				if mode == "absent" {
					if present {
						t.Fatalf("%s fabricated: %v", field, value)
					}
					continue
				}
				rows, ok := value.([]any)
				if !present || !ok {
					t.Fatalf("%s lost explicit array: %+v", field, out)
				}
				expected := 0
				if mode == "finding" && field == "diagnostics" {
					expected = 1
				}
				if len(rows) != expected {
					t.Fatalf("%s=%v", field, rows)
				}
				if expected == 1 {
					row := rows[0].(map[string]any)
					if row["change"] != "introduced" || row["code"] != "focus_layout_same_cell" {
						t.Fatalf("receipt altered: %v", row)
					}
				}
			}
		})
	}
}
