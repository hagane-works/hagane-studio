package toolapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type deleteFixture struct {
	mu       sync.Mutex
	deleted  []string
	entity   map[string]any
	edges    []map[string]any
	deleteAt string
}

func (f *deleteFixture) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			f.mu.Lock()
			f.deleted = append(f.deleted, r.URL.Path)
			f.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/relations/graph") {
			_ = json.NewEncoder(w).Encode(map[string]any{"edges": f.edges})
			return
		}
		_ = json.NewEncoder(w).Encode(f.entity)
	}
}

func (f *deleteFixture) deletions() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.deleted...)
}

func newIdeaFixture(edges ...map[string]any) *deleteFixture {
	return &deleteFixture{
		entity: map[string]any{
			"id": 12, "idea_id": "han_mandate", "idea_type": "country",
			"display_name_locs": map[string]any{"en": "Mandate of Heaven"},
		},
		edges: edges,
	}
}

func TestPreviewNamesEveryPlaceThatReferencesTheTarget(t *testing.T) {
	fixture := newIdeaFixture(
		map[string]any{"source_kind": "focus", "source_id": 4, "source_name": "Restore the Han",
			"field_name": "completion_reward", "effect_name": "add_ideas", "target_id": "han_mandate"},
		map[string]any{"source_kind": "event", "source_id": 9, "source_name": "A Council",
			"field_name": "immediate", "effect_name": "add_ideas", "target_id": "something_else"},
	)
	session := genericReadSession(t, fixture.handler(t))
	result := callTool(t, session, "preview_delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 12,
	})
	if result.IsError {
		t.Fatalf("tool error: %+v", result.Content)
	}
	out := result.StructuredContent.(map[string]any)
	if out["reference_count"] != float64(1) || out["safe_to_delete"] != false {
		t.Fatalf("references not detected: %v", out)
	}
	first := out["referenced_by"].([]any)[0].(map[string]any)
	if first["source_name"] != "Restore the Han" || first["field_name"] != "completion_reward" {
		t.Errorf("finding is not actionable: %v", first)
	}

	if got := fixture.deletions(); len(got) != 0 {
		t.Fatalf("preview issued a DELETE: %v", got)
	}
	if !strings.Contains(out["recommendation"].(string), "FIRST") {
		t.Errorf("recommendation must say to fix references first: %v", out["recommendation"])
	}
}

func TestPreviewStatesItsCoverageEvenWhenNothingReferencesTheTarget(t *testing.T) {
	session := genericReadSession(t, newIdeaFixture().handler(t))
	result := callTool(t, session, "preview_delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 12,
	})
	out := result.StructuredContent.(map[string]any)
	if out["safe_to_delete"] != true {
		t.Fatalf("expected safe: %v", out)
	}
	coverage, _ := out["coverage"].(string)
	if !strings.Contains(coverage, "raw script files") {
		t.Fatalf("a clean preview must state what it searched: %q", coverage)
	}
}

func TestDeleteWithoutTheConfirmTokenIsRefused(t *testing.T) {
	fixture := newIdeaFixture()
	session := genericReadSession(t, fixture.handler(t))

	for _, confirm := range []string{"", "true", "yes", "delete-idea-99"} {
		result := callTool(t, session, "delete_entity", map[string]any{
			"project_id": 296, "kind": "idea", "id": 12, "confirm": confirm,
		})
		if !result.IsError {
			t.Errorf("confirm=%q was accepted", confirm)
		}
	}
	if got := fixture.deletions(); len(got) != 0 {
		t.Fatalf("a refused delete still reached the backend: %v", got)
	}
}

func TestRefusalDoesNotLeakTheConfirmToken(t *testing.T) {
	session := genericReadSession(t, newIdeaFixture().handler(t))
	result := callTool(t, session, "delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 12, "confirm": "true",
	})
	if text := contentText(result); strings.Contains(text, "delete-idea-12") {
		t.Fatalf("the refusal handed over the token: %s", text)
	}
}

func TestConfirmTokenIsBoundToOneExactTarget(t *testing.T) {
	fixture := newIdeaFixture()
	session := genericReadSession(t, fixture.handler(t))
	preview := callTool(t, session, "preview_delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 12,
	})
	token := preview.StructuredContent.(map[string]any)["confirm_token"].(string)

	wrong := callTool(t, session, "delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 13, "confirm": token,
	})
	if !wrong.IsError {
		t.Fatal("a token from another target was accepted")
	}

	wrongKind := callTool(t, session, "delete_entity", map[string]any{
		"project_id": 296, "kind": "technology", "id": 12, "confirm": token,
	})
	if !wrongKind.IsError {
		t.Fatal("a token from another kind was accepted")
	}
	if got := fixture.deletions(); len(got) != 0 {
		t.Fatalf("a mismatched token still deleted something: %v", got)
	}
}

func TestDeleteWithTheMatchingTokenGoesThrough(t *testing.T) {
	fixture := newIdeaFixture()
	session := genericReadSession(t, fixture.handler(t))
	preview := callTool(t, session, "preview_delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 12,
	})
	token := preview.StructuredContent.(map[string]any)["confirm_token"].(string)

	result := callTool(t, session, "delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 12, "confirm": token,
	})
	if result.IsError {
		t.Fatalf("tool error: %+v", result.Content)
	}
	out := result.StructuredContent.(map[string]any)
	if out["deleted"] != true || out["key"] != "han_mandate" {
		t.Errorf("out = %v", out)
	}
	deletions := fixture.deletions()
	if len(deletions) != 1 || deletions[0] != "/api/projects/296/ideas/12" {
		t.Fatalf("DELETE path = %v", deletions)
	}
}

func TestDeletingSomethingStillReferencedSaysWhatItLeftBehind(t *testing.T) {
	fixture := newIdeaFixture(map[string]any{
		"source_kind": "focus", "source_id": 4, "source_name": "Restore the Han",
		"field_name": "completion_reward", "target_id": "han_mandate",
	})
	session := genericReadSession(t, fixture.handler(t))
	preview := callTool(t, session, "preview_delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 12,
	})
	token := preview.StructuredContent.(map[string]any)["confirm_token"].(string)

	result := callTool(t, session, "delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 12, "confirm": token,
	})
	out := result.StructuredContent.(map[string]any)
	if out["references_left"] != float64(1) {
		t.Fatalf("references_left = %v, want 1", out["references_left"])
	}
	note, _ := out["note"].(string)
	if !strings.Contains(note, "silently") {
		t.Errorf("note must say the leftover references fail silently: %q", note)
	}
}

func TestKeylessKindsPreviewWithoutConsultingTheGraph(t *testing.T) {
	fixture := &deleteFixture{entity: map[string]any{"id": 4, "oob_file": "HAN_1936", "count": 3}}
	graphCalls := 0
	session := genericReadSession(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/relations/graph") {
			graphCalls++
		}
		fixture.handler(t)(w, r)
	})
	result := callTool(t, session, "preview_delete_entity", map[string]any{
		"project_id": 296, "kind": "division_deployment", "id": 4,
	})
	if result.IsError {
		t.Fatalf("tool error: %+v", result.Content)
	}
	if graphCalls != 0 {
		t.Errorf("graph consulted %d times for a keyless kind", graphCalls)
	}
	if result.StructuredContent.(map[string]any)["safe_to_delete"] != true {
		t.Errorf("out = %v", result.StructuredContent)
	}
}

func TestPreviewRejectsAMissingRowID(t *testing.T) {
	fixture := newIdeaFixture()
	session := genericReadSession(t, fixture.handler(t))
	result := callTool(t, session, "preview_delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 0,
	})
	if !result.IsError {
		t.Fatal("id 0 must be refused")
	}
	if !strings.Contains(contentText(result), "list_entities") {
		t.Errorf("refusal must point at where the row id comes from: %s", contentText(result))
	}
}

func TestPreviewReportsTheTargetItLoaded(t *testing.T) {
	session := genericReadSession(t, newIdeaFixture().handler(t))
	result := callTool(t, session, "preview_delete_entity", map[string]any{
		"project_id": 296, "kind": "idea", "id": 12,
	})
	target := result.StructuredContent.(map[string]any)["target"].(map[string]any)
	if target["key"] != "han_mandate" || target["name"] != "Mandate of Heaven" {
		t.Fatalf("target = %v", target)
	}
}

var _ = httptest.NewServer
