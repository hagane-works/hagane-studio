package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEditRevisionToolTransport(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var tokens map[string]string
		if json.Unmarshal([]byte(r.Header.Get("X-Edit-Revisions")), &tokens) != nil || tokens["focus_tree:21"] != "9007199254740993" {
			t.Error("original token not forwarded", r.Header.Get("X-Edit-Revisions"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, present := body["edit_revisions"]; present {
			t.Error("transport metadata leaked into content")
		}
		w.Header().Set("X-Edit-Revisions", `{"focus_tree:21":"9007199254740994"}`)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":56,"focus_id":"GER_authored","tree_edit_revision":"9007199254740994"}`))
	}))
	defer upstream.Close()
	srv := newAuthedTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	session := connectClient(t, srv.URL)
	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	declared := false
	for _, tool := range listed.Tools {
		if tool.Name == "create_focus" {
			raw, _ := json.Marshal(tool.InputSchema)
			var schema map[string]any
			_ = json.Unmarshal(raw, &schema)
			props, _ := schema["properties"].(map[string]any)
			declared = props["edit_revisions"] != nil
		}
	}
	if !declared {
		t.Fatal("precondition absent from advertised contract")
	}
	result, err := session.CallTool(context.Background(), &CallToolParams{Name: "create_focus", Arguments: map[string]any{"project_id": 7, "focus_tree_id": 21, "name": "Authored", "edit_revisions": map[string]string{"focus_tree:21": "9007199254740993"}}})
	if err != nil || result.IsError {
		t.Fatalf("call: %v %+v", err, result)
	}
	if result.StructuredContent.(map[string]any)["tree_edit_revision"] != "9007199254740994" {
		t.Fatal(result.StructuredContent)
	}
	revisions, _ := result.StructuredContent.(map[string]any)["edit_revisions"].(map[string]any)
	if revisions["focus_tree:21"] != "9007199254740994" {
		t.Fatal(result.StructuredContent)
	}
	_, _ = session.CallTool(context.Background(), &CallToolParams{Name: "create_focus", Arguments: map[string]any{"project_id": 7, "focus_tree_id": 21, "name": "Invalid", "edit_revisions": map[string]any{"focus_tree:21": 1}}})
	if calls != 1 {
		t.Fatal("invalid precondition reached writer", calls)
	}
}
