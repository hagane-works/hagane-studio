package toolapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestFocusAndSuperEventGetEntityUseSingleGet(t *testing.T) {
	for _, name := range []string{"focus", "super_event"} {
		kind, err := lookupEntityKind(name)
		if err != nil {
			t.Fatal(err)
		}
		if kind.ReadFromList {
			t.Fatalf("%s: ReadFromList must be false because a single GET exists", name)
		}
	}

	var paths []string
	session := genericReadSession(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/projects/296/super-events/25":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 25, "project_id": 296, "number": 1001, "country_tag": "ALB", "title_en": "T"})
		case "/api/projects/296/focus-trees/1591/nodes/93011":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 93011, "focus_tree_id": 1591, "focus_id": "ALB_w4tmp_first", "cost": 5})
		default:

			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	})

	se := callTool(t, session, "get_entity", map[string]any{"project_id": 296, "kind": "super_event", "id": 25})
	if se.IsError {
		t.Fatalf("super_event get_entity: %s", contentText(se))
	}
	if key := se.StructuredContent.(map[string]any)["summary"].(map[string]any)["key"]; key != "alb_se.1001" {
		t.Errorf("super_event key=%v want alb_se.1001", key)
	}
	focus := callTool(t, session, "get_entity", map[string]any{"project_id": 296, "kind": "focus", "id": 93011, "parent_id": 1591})
	if focus.IsError {
		t.Fatalf("focus get_entity: %s", contentText(focus))
	}
	if key := focus.StructuredContent.(map[string]any)["summary"].(map[string]any)["key"]; key != "ALB_w4tmp_first" {
		t.Errorf("focus key=%v want ALB_w4tmp_first", key)
	}
	for _, p := range paths {
		if p == "/api/projects/296/super-events" || p == "/api/projects/296/focus-trees/1591/nodes" {
			t.Fatalf("get_entity still read the list path %q; paths=%v", p, paths)
		}
	}
	saw := map[string]bool{}
	for _, p := range paths {
		saw[p] = true
	}
	if !saw["/api/projects/296/super-events/25"] || !saw["/api/projects/296/focus-trees/1591/nodes/93011"] {
		t.Fatalf("single GET paths not requested; paths=%v", paths)
	}
}
