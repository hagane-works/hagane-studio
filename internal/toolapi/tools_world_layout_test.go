package toolapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func largeWorldResult(states, provinces int) map[string]any {
	layout := make([]any, states)
	ids := make([]any, states)
	for i := range layout {
		layout[i] = map[string]any{"state_id": i + 1, "owner": fmt.Sprintf("A%02d", i%32), "capital_province": i*10 + 1,
			"province_count": 10, "map_position": map[string]int{"x": i * 2, "y": i}, "image_position": map[string]int{"x": i, "y": i / 2}}
		ids[i] = i + 1
	}
	allProvinces := make([]any, provinces)
	for i := range allProvinces {
		allProvinces[i] = i + 1
	}
	status := map[string]any{"complete": false, "total_states": states, "languages": []string{"en", "zh"}}
	summary := map[string]any{"states": states, "provinces": map[string]int{"land": provinces}, "state_layout": layout, "name_status": status}
	return map[string]any{"addition": map[string]any{"id": 77, "kind": "world", "seed": 20261020, "summary": summary,
		"provinces": allProvinces, "state_ids": ids, "raw_paths": []any{"map/provinces.bmp", "map/definition.csv"}},
		"summary": summary, "state_layout": layout, "name_status": status, "findings": []any{},
		"preview_urls": map[string]string{"political": "https://example.test/preview.png"}}
}

func TestLargeWorldResultKeepsEveryNamingRowOnceAndSaveReceipt(t *testing.T) {
	original := largeWorldResult(900, 20000)
	before, _ := json.Marshal(original)
	out := compactWorldResult(original)
	call, out := worldImageResult(out)
	body, _ := json.Marshal(out)
	if len(body) >= 100000 || len(body)*3 >= len(before) {
		t.Fatalf("world result not compact: before=%d after=%d", len(before), len(body))
	}
	if strings.Count(string(body), `"state_id":`) != 900 || len(out["state_layout"].([]any)) != 900 || out["state_layout_complete"] != true {
		t.Fatalf("canonical layout lost or repeated: bytes=%d", len(body))
	}
	if strings.Contains(string(body), `"province_count"`) || strings.Contains(string(body), `"state_ids":[`) || strings.Contains(string(body), `"provinces":[`) {
		t.Fatal("full geometry leaked into the naming result")
	}
	addition := out["addition"].(map[string]any)
	if addition["id"] != 77 || addition["kind"] != "world" || addition["provinces_count"] != 20000 || addition["state_ids_count"] != 900 {
		t.Fatalf("saved geometry receipt lost: %v", addition)
	}
	text := call.Content[0].(*TextContent).Text
	if text != string(body) {
		t.Fatal("model text does not match compact structured content")
	}

	after, _ := json.Marshal(original)
	if string(before) != string(after) {
		t.Fatal("compaction mutated upstream saved geometry")
	}
	t.Logf("900 states/20000 provinces: %d -> %d JSON bytes; all 900 canonical state IDs preserved", len(before), len(body))
}

func TestWorldLayoutCompactionReportsUnknownFieldsAndAttachesImages(t *testing.T) {
	resp := largeWorldResult(1, 10)
	resp["future_geometry"] = []any{1, 2, 3}
	resp["state_layout"].([]any)[0].(map[string]any)["future_row_geometry"] = []int{1, 2}
	resp["previews"] = map[string]any{"terrain": "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("png"))}
	call, out := worldImageResult(compactWorldResult(resp))
	if out["unsupported_fields"].(map[string]int)["future_geometry"] != 1 || out["state_layout_unsupported_fields"].(map[string]int)["future_row_geometry"] != 1 {
		t.Fatal("unknown upstream fields silently dropped")
	}
	if len(call.Content) != 2 || strings.Contains(call.Content[0].(*TextContent).Text, "base64") {
		t.Fatal("preview was repeated in model text")
	}
}

func TestSavedWorldStateLayoutPagesCoverEveryStateWithoutRegenerating(t *testing.T) {
	resp := largeWorldResult(271, 9000)
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.URL.Path != "/api/projects/7/gui-render/world-map/land" {
			t.Errorf("read tool attempted %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"additions": []any{resp["addition"]}})
	}))
	defer upstream.Close()
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerWorldImageTools(s, NewRESTProxy(upstream.URL)) })
	session := connectClient(t, srv.URL)
	seen := map[int]bool{}
	for offset := 0; ; {
		result, err := session.CallTool(context.Background(), &CallToolParams{Name: "get_world_state_layout", Arguments: map[string]any{"project_id": 7, "offset": offset, "limit": 100}})
		if err != nil || result.IsError {
			t.Fatalf("page offset %d: err=%v result=%+v", offset, err, result)
		}
		var page map[string]any
		if err := json.Unmarshal([]byte(result.Content[0].(*TextContent).Text), &page); err != nil {
			t.Fatal(err)
		}
		if page["total"] != float64(271) || page["offset"] != float64(offset) {
			t.Fatalf("page contract: %v", page)
		}
		for _, value := range page["state_layout"].([]any) {
			row := value.(map[string]any)
			id := int(row["state_id"].(float64))
			if seen[id] || row["owner"] == nil || row["capital_province"] == nil || row["image_position"] == nil {
				t.Fatalf("duplicate or incomplete state row: %v", row)
			}
			seen[id] = true
		}
		if page["complete"] == true {
			if _, exists := page["next_offset"]; exists {
				t.Fatal("last page advertises another page")
			}
			break
		}
		offset = int(page["next_offset"].(float64))
	}
	if len(seen) != 271 || requests != 3 {
		t.Fatalf("read coverage: %d states, %d GETs", len(seen), requests)
	}
}

func TestCreateWorldToolReturnsCompactResultWithoutLosingNamingScope(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/projects/7/gui-render/world-map/image-world" {
			t.Errorf("unexpected create route %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(largeWorldResult(900, 20000))
	}))
	defer upstream.Close()
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerWorldImageTools(s, NewRESTProxy(upstream.URL)) })
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{Name: "create_world_from_images", Arguments: map[string]any{"project_id": 7, "terrain_image_id": "wimg_aaaaaaaaaaaaaaaa", "seed": 20261020}})
	if err != nil || result.IsError {
		t.Fatalf("create err=%v result=%+v", err, result)
	}
	body := result.Content[0].(*TextContent).Text
	if len(body) >= 100000 || strings.Count(body, `"state_id":`) != 900 || !strings.Contains(body, `"state_layout_read_tool":"get_world_state_layout"`) {
		t.Fatalf("tool naming result still oversized or incomplete: %d bytes", len(body))
	}
}

func TestWorldResultPrivateFixtureReplay(t *testing.T) {
	path := os.Getenv("HOI4_WORLD_RESULT_FIXTURE")
	if path == "" {
		t.Skip("set HOI4_WORLD_RESULT_FIXTURE to an existing private result JSON")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	out := compactWorldResult(resp)
	compact, _ := json.Marshal(out)
	before := worldRows(resp["state_layout"])
	after := worldRows(out["state_layout"])
	if len(before) == 0 || len(before) != len(after) || len(compact)*2 >= len(body) {
		t.Fatalf("private fixture not compact or complete: %d -> %d bytes, %d -> %d states", len(body), len(compact), len(before), len(after))
	}
	for i, value := range before {
		row := value.(map[string]any)
		got := after[i].(map[string]any)
		for _, field := range []string{"state_id", "owner", "capital_province", "image_position"} {
			if !reflect.DeepEqual(row[field], got[field]) {
				t.Fatalf("canonical field %s changed in row %d", field, i)
			}
		}
	}
	t.Logf("private replay: %d -> %d JSON bytes; all %d naming rows and required fields unchanged", len(body), len(compact), len(before))
}
