package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestWorldMapRecordTools(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	changed := true
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		mu.Unlock()
		switch {
		case r.Method == http.MethodGet && changed:
			_ = json.NewEncoder(w).Encode(map[string]any{"additions": []any{}, "map_changes": map[string]any{
				"count": 2, "confirmation": "restore-2-41",
				"records": []any{map[string]any{"id": 40, "kind": "world", "name": "导入 MOD 地图"}, map[string]any{"id": 41, "kind": "edit", "name": "x"}},
			}})
		case r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"additions": []any{}, "map_changes": nil})
		case r.URL.Query().Get("latest") != "":
			_ = json.NewEncoder(w).Encode(map[string]any{"undone": map[string]any{"id": 41, "kind": "edit", "name": "x"}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"restored": 2, "removed": []any{}})
		}
	}))
	defer upstream.Close()
	srv := newProxyOnlyTestServer(t, func(s *Server) {
		registerWorldMapRecordTools(s, NewRESTProxy(upstream.URL))
	})
	session := connectClient(t, srv.URL)
	call := func(name string) map[string]any {
		t.Helper()
		result, err := session.CallTool(context.Background(), &CallToolParams{Name: name, Arguments: map[string]any{"project_id": 7}})
		if err != nil || result.IsError {
			t.Fatalf("%s: err=%v result=%+v", name, err, result)
		}
		out, _ := result.StructuredContent.(map[string]any)
		return out
	}
	took := func() []string {
		mu.Lock()
		defer mu.Unlock()
		out := calls
		calls = nil
		return out
	}

	if listed := call("list_world_map_records"); listed["records"] == nil {
		t.Fatalf("list %+v", listed)
	}
	took()

	call("undo_world_map_change")
	if got := took(); len(got) != 1 || got[0] != "DELETE /api/projects/7/gui-render/world-map/land?latest=1" {
		t.Fatalf("undo %v", got)
	}

	if restored := call("restore_world_map"); restored["restored"] != float64(2) {
		t.Fatalf("restore %+v", restored)
	}
	if got := took(); len(got) != 2 || got[1] != "DELETE /api/projects/7/gui-render/world-map/land?confirm=restore-2-41" {
		t.Fatalf("restore %v", got)
	}

	changed = false
	if restored := call("restore_world_map"); restored["already_base"] != true {
		t.Fatalf("base map %+v", restored)
	}
	for _, c := range took() {
		if strings.HasPrefix(c, "DELETE") {
			t.Fatalf("deleted on the base map: %s", c)
		}
	}
}
