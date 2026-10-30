package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var updateGapTools = []string{
	"update_super_event", "update_bookmark", "update_custom_trait",
	"update_dynamic_modifier", "update_scripted_snippet", "update_scripted_localisation",
	"update_autonomy_state", "update_division_template", "update_division_deployment",
	"update_equipment", "update_country", "update_focus_tree", "update_sub_unit",
}

func updateGapSession(t *testing.T, handler http.HandlerFunc) *testSession {
	t.Helper()
	upstream := httptest.NewServer(handler)
	t.Cleanup(upstream.Close)
	srv := newAuthedTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	return connectClient(t, srv.URL)
}

func TestUpdateSendsOnlyTheFieldsSupplied(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	session := updateGapSession(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 12, "equipment_id": "han_rifle"})
	})
	result := callTool(t, session, "update_equipment", map[string]any{
		"project_id": 296, "id": 12, "soft_attack": 4.5,
	})
	if result.IsError {
		t.Fatalf("tool error: %+v", result.Content)
	}
	if gotMethod != "PUT" || gotPath != "/api/projects/296/equipment/12" {
		t.Errorf("%s %s", gotMethod, gotPath)
	}
	if len(gotBody) != 1 || gotBody["soft_attack"] != 4.5 {
		t.Fatalf("body must carry ONLY the supplied field, got %v", gotBody)
	}
}

func TestUpdateFocusTreeAcceptsCreateResultID(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	session := updateGapSession(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 73, "tree_id": "AOI_tree", "country_tag": "AOI",
		})
	})
	result := callTool(t, session, "update_focus_tree", map[string]any{
		"project_id": 296, "focus_tree_id": 73, "country_tag": "AOI",
	})
	if result.IsError {
		t.Fatalf("focus_tree_id alias was rejected: %s", contentText(result))
	}
	if gotPath != "/api/projects/296/focus-trees/73" {
		t.Fatalf("path=%q, want create_focus_tree's numeric id", gotPath)
	}
	if len(gotBody) != 1 || gotBody["country_tag"] != "AOI" {
		t.Fatalf("body=%v", gotBody)
	}
}

func TestUpdateFocusTreeDoesNotAdvertiseOrSilentlyAcceptUnsupportedLocalisation(t *testing.T) {
	session := updateGapSession(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unsupported tree localisation must never reach upstream as a no-op write")
	})
	catalog, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range catalog.Tools {
		if tool.Name == "update_focus_tree" {
			schema, _ := json.Marshal(tool.InputSchema)
			if strings.Contains(string(schema), "display_name_locs") || strings.Contains(string(schema), "description_locs") {
				t.Fatalf("tree metadata advertises unsupported localisation: %s", schema)
			}
		}
	}
	result := callTool(t, session, "update_focus_tree", map[string]any{
		"project_id": 296, "focus_tree_id": 73,
		"display_name_locs": map[string]any{"en": "Unsupported tree title"},
	})
	if !result.IsError {
		t.Fatal("unsupported tree localisation was reported as a successful update")
	}
}

func TestUpdateWithNoFieldsIsRefused(t *testing.T) {
	session := updateGapSession(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("upstream must not be called when there is nothing to change")
	})
	result := callTool(t, session, "update_country", map[string]any{"project_id": 296, "id": 3})
	if !result.IsError {
		t.Fatal("an update with no fields must be refused, not reported as success")
	}
	if !strings.Contains(contentText(result), "at least one") {
		t.Errorf("refusal must say what is missing: %s", contentText(result))
	}
}

func TestUpdateRejectsAMissingRowID(t *testing.T) {
	session := updateGapSession(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("upstream must not be called")
	})
	result := callTool(t, session, "update_bookmark", map[string]any{
		"project_id": 296, "id": 0, "name_en": "Spring 1936",
	})
	if !result.IsError {
		t.Fatal("id 0 must be refused")
	}
	if !strings.Contains(contentText(result), "list_entities") {
		t.Errorf("refusal must point at where the row id comes from: %s", contentText(result))
	}
}

func TestUpdateReadsTheKeyBackFromTheResponse(t *testing.T) {
	session := updateGapSession(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "trait_id": "han_scholar_2"})
	})
	result := callTool(t, session, "update_custom_trait", map[string]any{
		"project_id": 296, "id": 7, "trait_id": "han_scholar",
	})
	out := result.StructuredContent.(map[string]any)
	if out["key"] != "han_scholar_2" {
		t.Fatalf("key = %v, want the stored value not the requested one", out["key"])
	}

	if notice, _ := out["id_reassigned"].(string); !strings.Contains(notice, "ID REASSIGNED") {
		t.Errorf("a renumbered id produced no notice: %v", out)
	}
}

func TestUpdateSuperEventReportsTheComposedGameID(t *testing.T) {
	session := updateGapSession(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 9, "country_tag": "HAN", "number": 1100})
	})
	result := callTool(t, session, "update_super_event", map[string]any{
		"project_id": 296, "id": 9, "title_en": "The Yellow Turbans Rise",
	})
	out := result.StructuredContent.(map[string]any)
	if out["key"] != "han_se.1100" {
		t.Fatalf("key = %v, want han_se.1100", out["key"])
	}
}

func TestUpdateForwardsTheCanonicalLocsMap(t *testing.T) {
	var gotBody map[string]any
	session := updateGapSession(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 5, "idea_id": "x"})
	})
	callTool(t, session, "update_sub_unit", map[string]any{
		"project_id": 296, "id": 5,
		"display_name_locs": map[string]any{"en": "Han Militia", "zh": "汉军民兵"},
	})
	locs, ok := gotBody["display_name_locs"].(map[string]any)
	if !ok || locs["zh"] != "汉军民兵" {
		t.Fatalf("locs map not forwarded: %v", gotBody)
	}
}

func TestAllThirteenUpdateToolsAreRegistered(t *testing.T) {
	session := updateGapSession(t, func(w http.ResponseWriter, _ *http.Request) {})
	result, err := session.ListTools(context.Background(), &ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	registered := map[string]string{}
	for _, tool := range result.Tools {
		registered[tool.Name] = tool.Description
	}
	for _, name := range updateGapTools {
		if _, ok := registered[name]; !ok {
			t.Errorf("%s is not registered", name)
		}
	}

	if !strings.Contains(registered["update_focus_tree"], "update_focus") {
		t.Errorf("update_focus_tree must distinguish itself from update_focus: %q", registered["update_focus_tree"])
	}
	if !strings.Contains(registered["update_super_event"], "update_event") {
		t.Errorf("update_super_event must distinguish itself from update_event: %q", registered["update_super_event"])
	}
}

func TestRenameConsequencesAreStatedUpFront(t *testing.T) {
	session := updateGapSession(t, func(w http.ResponseWriter, _ *http.Request) {})
	result, err := session.ListTools(context.Background(), &ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	descriptions := map[string]string{}
	for _, tool := range result.Tools {
		descriptions[tool.Name] = tool.Description
	}
	for _, name := range []string{
		"update_scripted_snippet", "update_division_template", "update_country", "update_sub_unit",
	} {
		if !strings.Contains(descriptions[name], "NOT") {
			t.Errorf("%s must warn that renaming leaves references behind: %q", name, descriptions[name])
		}
	}
}
