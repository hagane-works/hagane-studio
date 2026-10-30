package toolapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func genericReadSession(t *testing.T, handler http.HandlerFunc) *testSession {
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

func callTool(t *testing.T, session *testSession, name string, args map[string]any) *CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	return result
}

func TestListEntitiesReturnsTheCommonEnvelope(t *testing.T) {
	var gotPath, gotQuery string
	session := genericReadSession(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id": 41, "event_id": "sanguo_shu.25", "namespace": "sanguo_shu",
			"country_tag": "HAN", "major": true,
			"display_name_locs": map[string]any{"en": "A Decision at Xuchang"},
		}})
	})
	result := callTool(t, session, "list_entities", map[string]any{
		"project_id": 296, "kind": "event", "country_tag": "han",
	})
	if result.IsError {
		t.Fatalf("tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/296/events" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "country_tag=HAN" {
		t.Errorf("query = %q, want the tag upper-cased", gotQuery)
	}
	out := result.StructuredContent.(map[string]any)
	entities := out["entities"].([]any)
	if len(entities) != 1 {
		t.Fatalf("entities = %v", out["entities"])
	}
	first := entities[0].(map[string]any)
	for field, want := range map[string]any{
		"id": float64(41), "key": "sanguo_shu.25", "name": "A Decision at Xuchang",
		"kind": "event", "country_tag": "HAN", "resolved_locale": "en",
	} {
		if first[field] != want {
			t.Errorf("%s = %v, want %v", field, first[field], want)
		}
	}
	summary := first["summary"].(map[string]any)
	if summary["namespace"] != "sanguo_shu" || summary["major"] != true {
		t.Errorf("summary = %v", summary)
	}
}

func TestSuperEventKeyIsComposedIntoTheUsableGameID(t *testing.T) {
	session := genericReadSession(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id": 9, "country_tag": "HAN", "number": 1100,
			"display_name_locs": map[string]any{"en": "The Yellow Turbans Rise"},
		}})
	})
	result := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "super_event"})
	first := result.StructuredContent.(map[string]any)["entities"].([]any)[0].(map[string]any)
	if first["key"] != "han_se.1100" {
		t.Fatalf("key = %v, want han_se.1100 (the form script must reference)", first["key"])
	}
}

func TestFocusTreeNodeCountIsDerivedAndTheNodesArrayIsDropped(t *testing.T) {
	session := genericReadSession(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id": 3, "tree_id": "han_focus", "tree_kind": "primary",
			"nodes":             []any{map[string]any{"id": 1}, map[string]any{"id": 2}, map[string]any{"id": 3}},
			"display_name_locs": map[string]any{"en": "Han Focus Tree"},
		}})
	})
	result := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "focus_tree"})
	first := result.StructuredContent.(map[string]any)["entities"].([]any)[0].(map[string]any)
	summary := first["summary"].(map[string]any)
	if summary["node_count"] != float64(3) {
		t.Errorf("node_count = %v, want 3", summary["node_count"])
	}
	if _, leaked := summary["nodes"]; leaked {
		t.Errorf("the nodes array reached the caller: %v", summary)
	}
}

func TestDerivedCountIsOmittedRatherThanGuessedAtZero(t *testing.T) {
	session := genericReadSession(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id": 3, "tree_id": "han_focus", "display_name_locs": map[string]any{"en": "Han"},
		}})
	})
	result := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "focus_tree"})
	first := result.StructuredContent.(map[string]any)["entities"].([]any)[0].(map[string]any)
	summary, _ := first["summary"].(map[string]any)
	if _, present := summary["node_count"]; present {
		t.Fatalf("node_count was invented with no nodes in the payload: %v", summary)
	}
}

func TestUnknownKindAnswersWithTheValidList(t *testing.T) {
	session := genericReadSession(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	result := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "focuses"})
	if !result.IsError {
		t.Fatal("an unknown kind must be an error, not an empty list")
	}
	text := contentText(result)
	if !strings.Contains(text, "focus_tree") || !strings.Contains(text, "event") {
		t.Errorf("error must enumerate the valid kinds, got: %s", text)
	}
}

func TestFocusParentHintNamesARegisteredKindAndCorrectRoute(t *testing.T) {
	var paths []string
	session := genericReadSession(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/projects/296/focus-trees":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 37, "tree_id": "GER_main", "country_tag": "GER", "nodes": []any{}}})
		case "/api/projects/296/focus-trees/37/nodes":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 91, "focus_id": "GER_start", "focus_tree_id": 37}})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	missing := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "focus", "country_tag": "GER"})
	if !missing.IsError || !strings.Contains(contentText(missing), `kind="focus_tree"`) || strings.Contains(contentText(missing), `kind="focus-tree"`) {
		t.Fatalf("parent lookup hint must name the actual registered kind: %s", contentText(missing))
	}
	if len(paths) != 0 {
		t.Fatalf("missing parent must not guess a tree or call upstream: %v", paths)
	}
	trees := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "focus_tree", "country_tag": "GER"})
	if trees.IsError {
		t.Fatalf("the exact hinted kind failed: %s", contentText(trees))
	}
	treeID := trees.StructuredContent.(map[string]any)["entities"].([]any)[0].(map[string]any)["id"]
	nodes := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "focus", "parent_id": treeID})
	if nodes.IsError || len(nodes.StructuredContent.(map[string]any)["entities"].([]any)) != 1 {
		t.Fatalf("explicit parent read failed: %s", contentText(nodes))
	}
	if len(paths) != 2 || paths[1] != "/api/projects/296/focus-trees/37/nodes" {
		t.Fatalf("parent route = %v", paths)
	}
}

func TestStatesRefuseToListWithoutACountryFilter(t *testing.T) {
	session := genericReadSession(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("upstream must not be called at all")
	})
	result := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "state"})
	if !result.IsError {
		t.Fatal("listing every state must be refused")
	}
	if !strings.Contains(contentText(result), "country_tag") {
		t.Errorf("refusal must say how to proceed: %s", contentText(result))
	}
}

func TestTruncationIsAlwaysStated(t *testing.T) {
	session := genericReadSession(t, func(w http.ResponseWriter, _ *http.Request) {
		rows := make([]map[string]any, 5)
		for i := range rows {
			rows[i] = map[string]any{"id": i + 1, "idea_id": "idea_x", "display_name_locs": map[string]any{"en": "X"}}
		}
		_ = json.NewEncoder(w).Encode(rows)
	})
	result := callTool(t, session, "list_entities", map[string]any{
		"project_id": 296, "kind": "idea", "limit": 2,
	})
	out := result.StructuredContent.(map[string]any)
	if out["returned"] != float64(2) || out["total"] != float64(5) || out["truncated"] != true {
		t.Fatalf("truncation not reported: %v", out)
	}
	if note, _ := out["note"].(string); note == "" {
		t.Error("a truncated list must carry a note saying so")
	}
}

func TestBothListShapesAreUnderstood(t *testing.T) {
	session := genericReadSession(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ai_strategy_plans": []map[string]any{{"id": 4, "plan_id": "han_ai", "display_name_locs": map[string]any{"en": "Han AI"}}},
			"total":             1,
		})
	})
	result := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "ai_strategy_plan"})
	entities := result.StructuredContent.(map[string]any)["entities"].([]any)
	if len(entities) != 1 || entities[0].(map[string]any)["key"] != "han_ai" {
		t.Fatalf("object-wrapped list not understood: %v", result.StructuredContent)
	}
}

func TestNestedKindDemandsItsParentID(t *testing.T) {
	session := genericReadSession(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/projects/296/focus-trees/3/nodes") {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 7, "focus_id": "han_reform"}})
	})
	if result := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "focus"}); !result.IsError {
		t.Fatal("a nested kind without parent_id must be refused")
	} else if !strings.Contains(contentText(result), "parent_id") {
		t.Errorf("refusal must name the missing argument: %s", contentText(result))
	}
	result := callTool(t, session, "list_entities", map[string]any{
		"project_id": 296, "kind": "focus", "parent_id": 3,
	})
	if result.IsError {
		t.Fatalf("with parent_id it must work: %+v", result.Content)
	}
}

func TestUIPanelIsNotAnOfferedKind(t *testing.T) {
	if _, present := entityKindByName["ui_panel"]; present {
		t.Fatal("ui_panel must not be offered through generic entity tools")
	}
	for _, name := range entityKindNames() {
		if strings.Contains(name, "panel") {
			t.Errorf("unexpected panel-like kind offered: %s", name)
		}
	}
}

func TestEveryKeylessKindExplainsWhy(t *testing.T) {
	for _, kind := range entityKinds {
		if kind.KeyField == "" && strings.TrimSpace(kind.NoKeyReason) == "" {
			t.Errorf("kind %q has no key field and no explanation", kind.Kind)
		}
	}
}

func TestGetEntityRejectsTheGameIDInPlaceOfTheRowID(t *testing.T) {
	session := genericReadSession(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("upstream must not be called")
	})
	result := callTool(t, session, "get_entity", map[string]any{"project_id": 296, "kind": "event", "id": 0})
	if !result.IsError {
		t.Fatal("id 0 must be refused")
	}
	if !strings.Contains(contentText(result), "list_entities") {
		t.Errorf("refusal must point at where the row id comes from: %s", contentText(result))
	}
}

func TestGetEntityReturnsSummaryPlusEveryStoredField(t *testing.T) {
	session := genericReadSession(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/296/events/41" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 41, "event_id": "sanguo_shu.25", "immediate": "country_event = { id = han_se.1100 }",
			"display_name_locs": map[string]any{"en": "A Decision at Xuchang"},
		})
	})
	result := callTool(t, session, "get_entity", map[string]any{"project_id": 296, "kind": "event", "id": 41})
	out := result.StructuredContent.(map[string]any)
	summary := out["summary"].(map[string]any)
	if summary["key"] != "sanguo_shu.25" || summary["name"] != "A Decision at Xuchang" {
		t.Errorf("summary = %v", summary)
	}

	entity := out["entity"].(map[string]any)
	if entity["immediate"] != "country_event = { id = han_se.1100 }" {
		t.Errorf("full entity not returned: %v", entity)
	}
}

func TestNameFollowsTheRequestedLocale(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id": 1, "idea_id": "han_spirit",
			"display_name_locs": map[string]any{"en": "Mandate of Heaven", "ja": "天命"},
		}})
	}
	session := genericReadSession(t, handler)

	def := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "idea"})
	first := def.StructuredContent.(map[string]any)["entities"].([]any)[0].(map[string]any)
	if first["name"] != "Mandate of Heaven" || first["resolved_locale"] != "en" {
		t.Errorf("default locale: name=%v locale=%v", first["name"], first["resolved_locale"])
	}

	ja := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "idea", "locale": "ja"})
	first = ja.StructuredContent.(map[string]any)["entities"].([]any)[0].(map[string]any)
	if first["name"] != "天命" || first["resolved_locale"] != "ja" {
		t.Errorf("ja locale: name=%v locale=%v", first["name"], first["resolved_locale"])
	}

	de := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "idea", "locale": "de"})
	first = de.StructuredContent.(map[string]any)["entities"].([]any)[0].(map[string]any)
	if first["name"] != "Mandate of Heaven" || first["resolved_locale"] != "en" {
		t.Errorf("de fallback: name=%v locale=%v", first["name"], first["resolved_locale"])
	}
}

func TestHistoryFileSummarisesWhichSettingsAreConfigured(t *testing.T) {
	session := genericReadSession(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id": 2, "country_tag": "HAN",
			"set_politics": "ruling_party = neutrality", "set_stability": 0.4,
			"set_war_support": nil, "add_ideas": []any{},
		}})
	})
	result := callTool(t, session, "list_entities", map[string]any{"project_id": 296, "kind": "history_file"})
	first := result.StructuredContent.(map[string]any)["entities"].([]any)[0].(map[string]any)
	summary := first["summary"].(map[string]any)
	configured, ok := summary["configured_fields"].([]any)
	if !ok {
		t.Fatalf("configured_fields missing: %v", summary)
	}
	got := map[string]bool{}
	for _, field := range configured {
		got[field.(string)] = true
	}
	if !got["set_politics"] || !got["set_stability"] {
		t.Errorf("set fields missing from %v", configured)
	}
	if got["set_war_support"] || got["add_ideas"] {
		t.Errorf("unset fields reported as configured: %v", configured)
	}
}

func TestEveryAdvertisedKindHasGenericListAndReadPath(t *testing.T) {
	if len(entityKinds) != 39 {
		t.Fatalf("advertised kinds=%d, want the 39-kind public contract", len(entityKinds))
	}
	var paths []string
	session := genericReadSession(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath()+"?"+r.URL.RawQuery)
		kind := kindForMatrixPath(r.URL.Path)
		row := matrixRow(kind)
		if strings.HasSuffix(r.URL.Path, "/17") {
			_ = json.NewEncoder(w).Encode(row)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{row})
	})

	for _, kind := range entityKinds {
		kind := kind
		t.Run(kind.Kind, func(t *testing.T) {
			parentID := int64(0)
			if kind.ParentPath != "" {
				parentID = 3
			}
			countryTag := ""
			listArgs := map[string]any{"project_id": 296, "kind": kind.Kind, "parent_id": parentID}
			if kind.RequireCountry != "" {
				countryTag = "HAN"
				listArgs["country_tag"] = countryTag
			}
			beforeList := len(paths)
			list := callTool(t, session, "list_entities", listArgs)
			if list.IsError {
				t.Fatalf("list_entities failed: %s", contentText(list))
			}
			listPath := matrixListPath(kind, parentID, countryTag)
			if !matrixSawPath(paths[beforeList:], listPath) {
				t.Fatalf("list did not read %q; paths=%v", listPath, paths[beforeList:])
			}

			beforeGet := len(paths)
			get := callTool(t, session, "get_entity", map[string]any{
				"project_id": 296, "kind": kind.Kind, "id": 17, "parent_id": parentID,
			})
			if get.IsError {
				t.Fatalf("get_entity failed: %s", contentText(get))
			}
			getPath := fmt.Sprintf("/api/projects/296/%s/17", kind.Path)
			if kind.ParentPath != "" {
				getPath = fmt.Sprintf("/api/projects/296/%s/%d/%s/17", kind.ParentPath, parentID, kind.Path)
			}
			if kind.ReadFromList {
				getPath = matrixListPath(kind, parentID, "")
			}
			if !matrixSawPath(paths[beforeGet:], getPath) {
				t.Fatalf("get did not read %q; paths=%v", getPath, paths[beforeGet:])
			}
			entity := get.StructuredContent.(map[string]any)["entity"].(map[string]any)
			if entity["id"] != float64(17) {
				t.Fatalf("get readback entity=%v, want id 17", entity)
			}
		})
	}
}

func matrixSawPath(paths []string, want string) bool {
	for _, path := range paths {
		if path == want+"?" || path == want {
			return true
		}
	}
	return false
}

func matrixListPath(kind entityKind, parentID int64, countryTag string) string {
	path, err := entityListPath(296, kind, parentID)
	if err != nil {
		panic(err)
	}
	if countryTag != "" && kind.CountryFilter {
		if kind.CountryPath != "" {
			return "/api/projects/296/" + strings.ReplaceAll(kind.CountryPath, "{country_tag}", countryTag)
		}
		return path + "?country_tag=" + countryTag
	}
	return path
}

func kindForMatrixPath(path string) string {
	for _, kind := range entityKinds {
		if strings.Contains(path, "/"+kind.Path) {
			return kind.Kind
		}
	}
	return "idea"
}

func matrixRow(kind string) map[string]any {
	row := map[string]any{"id": 17, "display_name_locs": map[string]any{"en": "Matrix row"}}
	if spec, err := lookupEntityKind(kind); err == nil && spec.KeyField != "" {
		row[spec.KeyField] = "matrix_key"
	}
	if kind == "super_event" {
		row["country_tag"], row["number"] = "HAN", 1100
	}
	return row
}
