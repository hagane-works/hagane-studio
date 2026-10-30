package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func stateRESTResponse(displayName, locs map[string]any) map[string]any {
	row := map[string]any{
		"id": 42, "state_id": 613, "name": "STATE_613",
		"owner": "ENG", "controller": "ENG", "source": "created",
		"display_name_locs": locs,
	}
	if displayName != nil {
		row["display_name"] = displayName
	}
	return row
}

func editStateAgainst(t *testing.T, row map[string]any, args map[string]any) (*CallToolResult, map[string]any) {
	t.Helper()
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			gotBody, _ = readJSONBody(r)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(row)
	}))
	t.Cleanup(upstream.Close)

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerEditState(s, proxy) })
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "edit_state", Arguments: args,
	})
	if err != nil {

		t.Fatalf("CallTool returned a transport/validation error: %v", err)
	}
	return result, gotBody
}

func TestEditState_UpstreamMapDisplayNameSurvivesAndDoesNotFailValidation(t *testing.T) {
	row := stateRESTResponse(
		map[string]any{"en": "Borneo", "zh": "婆罗洲", "ru": "Борнео"},
		map[string]any{"zh": "婆罗洲", "ru": "Борнео"},
	)
	result, _ := editStateAgainst(t, row, map[string]any{
		"project_id": 7, "state_db_id": 42, "owner": "ENG",
	})
	if result.IsError {
		t.Fatalf("tool reported an error: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	disp, ok := out["display_name"].(map[string]any)
	if !ok {
		t.Fatalf("display_name = %#v, want the per-language map the route sent", out["display_name"])
	}
	for lang, want := range map[string]string{"en": "Borneo", "zh": "婆罗洲", "ru": "Борнео"} {
		if disp[lang] != want {
			t.Errorf("display_name[%q] = %v, want %q", lang, disp[lang], want)
		}
	}
}

func TestEditState_MissingUpstreamDisplayNameStillValidates(t *testing.T) {
	row := stateRESTResponse(nil, map[string]any{})
	result, _ := editStateAgainst(t, row, map[string]any{
		"project_id": 7, "state_db_id": 42, "owner": "ENG",
	})
	if result.IsError {
		t.Fatalf("tool reported an error: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	disp, ok := out["display_name"].(map[string]any)
	if !ok || len(disp) != 0 {
		t.Fatalf("display_name = %#v, want an empty object (nil map marshals to null and fails output validation)", out["display_name"])
	}
}

func TestEditState_RenameMergesAnyLanguageIntoExistingLocs(t *testing.T) {
	row := stateRESTResponse(
		map[string]any{"en": "Corsica", "zh": "科西嘉"},
		map[string]any{"zh": "科西嘉"},
	)
	_, body := editStateAgainst(t, row, map[string]any{
		"project_id": 7, "state_db_id": 42,
		"display_name_locs": map[string]any{"ru": "Корсика"},
	})
	locs, ok := body["display_name_locs"].(map[string]any)
	if !ok {
		t.Fatalf("PUT body display_name_locs = %#v, want a merged map", body["display_name_locs"])
	}
	if locs["ru"] != "Корсика" {
		t.Errorf("display_name_locs[ru] = %v, want Корсика", locs["ru"])
	}
	if locs["zh"] != "科西嘉" {
		t.Errorf("display_name_locs[zh] = %v, want the untouched language carried over", locs["zh"])
	}

	if _, leaked := locs["en"]; leaked {
		t.Errorf("display_name_locs[en] = %v — vanilla fallback leaked into the author's own overrides", locs["en"])
	}
}

func TestEditState_ShorthandStillWinsOverLocsMap(t *testing.T) {
	row := stateRESTResponse(map[string]any{"en": "Corsica"}, map[string]any{})
	_, body := editStateAgainst(t, row, map[string]any{
		"project_id": 7, "state_db_id": 42,
		"display_name_locs": map[string]any{"en": "from-locs", "ru": "Корсика"},
		"display_name_en":   "from-shorthand",
	})
	locs := body["display_name_locs"].(map[string]any)
	if locs["en"] != "from-shorthand" {
		t.Errorf("display_name_locs[en] = %v, want from-shorthand", locs["en"])
	}
	if locs["ru"] != "Корсика" {
		t.Errorf("display_name_locs[ru] = %v, want Корсика", locs["ru"])
	}
}

func TestDisplayNameText_FoldsTheMapShapeDownToOneString(t *testing.T) {
	for _, tc := range []struct {
		name   string
		entity map[string]any
		want   string
	}{
		{"string shape untouched", map[string]any{"display_name": "Borneo"}, "Borneo"},
		{"map shape uses the resolved language", map[string]any{
			"display_name": map[string]any{"en": "Borneo", "zh": "婆罗洲"}, "resolved_locale": "zh",
		}, "婆罗洲"},
		{"map shape falls back to English", map[string]any{
			"display_name": map[string]any{"en": "Borneo"}, "resolved_locale": "ru",
		}, "Borneo"},
		{"nothing at all", map[string]any{}, ""},
	} {
		if got := displayNameText(tc.entity); got != tc.want {
			t.Errorf("%s: displayNameText = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestEnrichDisplayName_CompletesUpstreamMapWithLanguagesItDropped(t *testing.T) {
	entity := map[string]any{
		"name":              "STATE_613",
		"display_name":      map[string]any{"en": "Borneo", "zh": "婆罗洲"},
		"display_name_locs": map[string]any{"zh": "婆罗洲", "ru": "Борнео", "de": "   "},
	}
	if !enrichDisplayName(entity, "ru") {
		t.Fatal("enrichDisplayName returned false for a localisable row")
	}
	disp, ok := entity["display_name"].(map[string]any)
	if !ok {
		t.Fatalf("display_name = %#v, want the map preserved", entity["display_name"])
	}
	if disp["ru"] != "Борнео" {
		t.Errorf("display_name[ru] = %v, want Борнео (author-stored language the upstream resolver dropped)", disp["ru"])
	}
	if disp["en"] != "Borneo" {
		t.Errorf("display_name[en] = %v, want Borneo (upstream value kept)", disp["en"])
	}
	if _, blank := disp["de"]; blank {
		t.Errorf("display_name[de] = %v, want the whitespace-only slot dropped", disp["de"])
	}
	if entity["resolved_locale"] != "ru" {
		t.Errorf("resolved_locale = %v, want ru", entity["resolved_locale"])
	}
}
