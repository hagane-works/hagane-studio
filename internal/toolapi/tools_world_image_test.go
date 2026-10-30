package toolapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateCountrySplitReturnsThePreviewAsAnImage(t *testing.T) {
	jpeg := []byte{0xff, 0xd8, 0xff, 0xd9}
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.Method + " " + r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"image":     map[string]any{"id": "wimg_0123456789abcdef", "kind": "owners", "tags": []string{"NRT", "STH"}},
			"countries": []any{map[string]any{"tag": "NRT", "share": 0.5}},
			"warnings":  []string{},
			"preview":   "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpeg),
		})
	}))
	defer upstream.Close()
	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerWorldImageTools(s, proxy) })
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_country_split",
		Arguments: map[string]any{"project_id": 7, "image_id": "wimg_aaaaaaaaaaaaaaaa", "terrain_image_id": "wimg_bbbbbbbbbbbbbbbb",
			"seeds": []any{map[string]any{"x": 10, "y": 20, "tag": "NRT"}}},
	})
	if err != nil || result.IsError {
		t.Fatalf("CallTool err=%v result=%+v", err, result)
	}
	if gotPath != "POST /api/projects/7/gui-render/world-map/image-countries" {
		t.Fatalf("path %s", gotPath)
	}
	if gotBody["terrain_image_id"] != "wimg_bbbbbbbbbbbbbbbb" || len(gotBody["seeds"].([]any)) != 1 {
		t.Fatalf("body %v", gotBody)
	}
	if _, sent := gotBody["tolerance"]; sent {
		t.Fatalf("unset tolerance sent: %v", gotBody)
	}
	var text *TextContent
	var picture *ImageContent
	for _, content := range result.Content {
		switch c := content.(type) {
		case *TextContent:
			if text == nil {
				text = c
			}
		case *ImageContent:
			picture = c
		}
	}
	if text == nil || strings.Contains(text.Text, "base64") || !strings.Contains(text.Text, `"previews_attached":["preview"]`) {
		t.Fatalf("text %+v", text)
	}
	if picture == nil || picture.MIMEType != "image/jpeg" || string(picture.Data) != string(jpeg) {
		t.Fatalf("image %+v", picture)
	}
}

func TestPoliticalWorldSchemaAllowsColourSeedsWithoutGuessedCoordinates(t *testing.T) {
	bodies := map[string]map[string]any{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		bodies[r.URL.Path] = body
		_ = json.NewEncoder(w).Encode(map[string]any{"image": map[string]any{"id": "wimg_0123456789abcdef"}})
	}))
	defer upstream.Close()
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerWorldImageTools(s, NewRESTProxy(upstream.URL)) })
	session := connectClient(t, srv.URL)
	for _, call := range []CallToolParams{
		{Name: "create_terrain_sketch", Arguments: map[string]any{"project_id": 7, "image_id": "wimg_aaaaaaaaaaaaaaaa", "source_kind": "political",
			"rules": []any{map[string]any{"colour": "#a1b9cd", "terrain": "ocean", "tolerance": 20}}}},
		{Name: "create_country_split", Arguments: map[string]any{"project_id": 7, "image_id": "wimg_aaaaaaaaaaaaaaaa", "terrain_image_id": "wimg_bbbbbbbbbbbbbbbb",
			"seeds": []any{map[string]any{"tag": "ARD", "colour": "#5a64a0", "box": map[string]any{"x": 0, "y": 900, "width": 400, "height": 300}},
				map[string]any{"tag": "OSK", "colour": "#5476a8", "at": map[string]any{"x": 1000, "y": 200}, "colour_tolerance": 12}}}},
	} {
		result, err := session.CallTool(context.Background(), &call)
		if err != nil || result.IsError {
			t.Fatalf("new source/colour-only schema rejected: %s err=%v result=%+v", call.Name, err, result)
		}
	}
	if bodies["/api/projects/7/gui-render/world-map/image-terrain"]["source_kind"] != "political" {
		t.Fatal("political mode was not forwarded to projectsvc")
	}
	seeds := bodies["/api/projects/7/gui-render/world-map/image-countries"]["seeds"].([]any)
	first := seeds[0].(map[string]any)
	if first["colour"] != "#5a64a0" || first["box"] == nil {
		t.Fatalf("source colour/selector lost: %v", first)
	}
	if _, exists := first["x"]; exists {
		t.Fatal("omitted coordinate silently defaulted to x=0")
	}
}

func TestWorldInspectionSourceSampleReachesTheFirstSplitRequest(t *testing.T) {
	var inspectCount, splitCount int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch r.URL.Path {
		case "/api/projects/7/gui-render/world-map/image-inspect":
			inspectCount++
			if body["max_colours"] != float64(40) {
				t.Errorf("complete source inspection lost max_colours: %v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"colours": []any{map[string]any{
				"hex": "#566ea4", "sample_source_colour": "#5476a8", "sample": map[string]any{"x": 1128, "y": 311},
			}}})
		case "/api/projects/7/gui-render/world-map/image-countries":
			splitCount++
			seed := body["seeds"].([]any)[0].(map[string]any)
			at := seed["at"].(map[string]any)
			if seed["colour"] != "#5476a8" || at["x"] != float64(1128) || at["y"] != float64(311) {
				t.Errorf("source colour/location was replaced by the display cluster or guessed coordinates: %v", seed)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"image": map[string]any{"id": "wimg_0123456789abcdef"}})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerWorldImageTools(s, NewRESTProxy(upstream.URL)) })
	session := connectClient(t, srv.URL)
	inspection := callTool(t, session, "get_world_image", map[string]any{"project_id": 7, "image_id": "wimg_aaaaaaaaaaaaaaaa", "max_colours": 40})
	if inspection.IsError {
		t.Fatalf("source inspection failed: %s", contentText(inspection))
	}
	swatch := inspection.StructuredContent.(map[string]any)["colours"].([]any)[0].(map[string]any)
	if swatch["hex"] != "#566ea4" {
		t.Fatal("legacy display colour was changed")
	}
	split := callTool(t, session, "create_country_split", map[string]any{
		"project_id": 7, "image_id": "wimg_aaaaaaaaaaaaaaaa", "terrain_image_id": "wimg_bbbbbbbbbbbbbbbb",
		"seeds": []any{map[string]any{"tag": "OSK", "colour": swatch["sample_source_colour"], "at": swatch["sample"]}},
	})
	if split.IsError || inspectCount != 1 || splitCount != 1 {
		t.Fatalf("sample contract must work without a failed split and retry: inspect=%d split=%d result=%s", inspectCount, splitCount, contentText(split))
	}
}

func TestWorldImageResultAttachesEveryPreview(t *testing.T) {
	png := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("png"))
	call, out := worldImageResult(map[string]any{"summary": map[string]any{"states": 3}, "previews": map[string]any{"political": png, "terrain": png}})
	if len(call.Content) != 3 {
		t.Fatalf("content %d", len(call.Content))
	}
	if _, kept := out["previews"]; kept {
		t.Fatalf("previews kept in the JSON: %v", out)
	}
	if names := out["previews_attached"].([]string); len(names) != 2 || names[0] != "political" {
		t.Fatalf("names %v", names)
	}
	if dataURLImage("https://example.com/x.png") != nil || dataURLImage("data:text/plain;base64,eA==") != nil {
		t.Fatal("non-image URL accepted")
	}
}

func TestWorldPreviewMapChangesGetTheRestoreLink(t *testing.T) {
	changed := true
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer := map[string]any{"summary": map[string]any{"states": 3}, "findings": []any{}}
		if changed {
			answer["map_changes"] = map[string]any{"count": 1, "confirmation": "restore-1-4", "records": []any{}}
		}
		_ = json.NewEncoder(w).Encode(answer)
	}))
	defer upstream.Close()
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerWorldImageTools(s, NewRESTProxy(upstream.URL)) })
	session := connectClient(t, srv.URL)
	call := func() string {
		result, err := session.CallTool(context.Background(), &CallToolParams{
			Name: "preview_world_from_images", Arguments: map[string]any{"project_id": 7, "terrain_image_id": "wimg_bbbbbbbbbbbbbbbb"},
		})
		if err != nil || result.IsError {
			t.Fatalf("err=%v result=%+v", err, result)
		}
		for _, content := range result.Content {
			if text, ok := content.(*TextContent); ok {
				return text.Text
			}
		}
		t.Fatalf("no text in %+v", result)
		return ""
	}

	if text := call(); !strings.Contains(text, "map_changes") || strings.Contains(text, "restore_link") {
		t.Fatalf("map changes = %s", text)
	}
	changed = false
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if strings.Contains(tool.Name, "restore") || strings.Contains(tool.Name, "discard") {
			t.Fatalf("a tool restores the map: %s", tool.Name)
		}
	}
}
