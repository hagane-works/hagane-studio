package toolapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type derivedFixture struct {
	mu     sync.Mutex
	hits   map[string]int
	routes map[string]any
}

func newDerivedFixture(routes map[string]any) *derivedFixture {
	return &derivedFixture{hits: map[string]int{}, routes: routes}
}

func (f *derivedFixture) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.hits[r.URL.Path]++
		f.mu.Unlock()
		for suffix, body := range f.routes {
			if strings.HasSuffix(r.URL.Path, suffix) {
				_ = json.NewEncoder(w).Encode(body)
				return
			}
		}
		_ = json.NewEncoder(w).Encode([]any{})
	}
}

func (f *derivedFixture) hitCount(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hits[path]
}

func summaryOf(t *testing.T, fixture *derivedFixture, kind string, args map[string]any) map[string]any {
	t.Helper()
	session := genericReadSession(t, fixture.handler(t))
	call := map[string]any{"project_id": 296, "kind": kind}
	for k, v := range args {
		call[k] = v
	}
	result := callTool(t, session, "list_entities", call)
	if result.IsError {
		t.Fatalf("tool error: %+v", result.Content)
	}
	entities := result.StructuredContent.(map[string]any)["entities"].([]any)
	if len(entities) == 0 {
		t.Fatalf("no entities returned: %v", result.StructuredContent)
	}
	summary, _ := entities[0].(map[string]any)["summary"].(map[string]any)
	return summary
}

func TestFocusTreeSharedRefCountComesFromSharedFocusRefs(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/focus-trees": []map[string]any{{
			"id": 3, "tree_id": "han_pool", "tree_kind": "shared_pool",
			"shared_focus_refs": []any{"a", "b"}, "nodes": []any{map[string]any{"id": 1}},
		}},
	})
	summary := summaryOf(t, fixture, "focus_tree", nil)
	if summary["shared_ref_count"] != float64(2) {
		t.Errorf("shared_ref_count = %v, want 2", summary["shared_ref_count"])
	}
	if summary["node_count"] != float64(1) {
		t.Errorf("node_count = %v, want 1", summary["node_count"])
	}

	for _, leaked := range []string{"shared_focus_refs", "nodes"} {
		if _, present := summary[leaked]; present {
			t.Errorf("%s leaked into the summary", leaked)
		}
	}
}

func TestModifierDefinitionCategoryCount(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/modifier-definitions": []map[string]any{{
			"id": 1, "modifier_id": "han_stab", "value_type": "percentage",
			"categories": []any{"country", "state", "all"},
		}},
	})
	if got := summaryOf(t, fixture, "modifier_definition", nil)["category_count"]; got != float64(3) {
		t.Errorf("category_count = %v, want 3", got)
	}
}

func TestCustomTraitAdvancedExtrasCount(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/custom-traits": []map[string]any{{
			"id": 1, "trait_id": "han_scholar", "trait_type": "country_leader",
			"extra_fields": map[string]any{"type": "land", "skill": 3},
		}},
	})
	if got := summaryOf(t, fixture, "custom_trait", nil)["advanced_extras_count"]; got != float64(2) {
		t.Errorf("advanced_extras_count = %v, want 2", got)
	}
}

func TestDeploymentActionableWarningUsesTheSamePrefixRule(t *testing.T) {
	cases := []struct {
		name     string
		warnings []any
		want     any
	}{
		{"actionable present", []any{"info: fine", "actionable: province not owned"}, true},
		{"only informational", []any{"info: fine"}, false},
		{"none at all", []any{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newDerivedFixture(map[string]any{
				"/division-deployments": []map[string]any{{
					"id": 4, "oob_file": "HAN_1936", "warnings": tc.warnings,
				}},
			})
			if got := summaryOf(t, fixture, "division_deployment", nil)["has_actionable_warning"]; got != tc.want {
				t.Errorf("has_actionable_warning = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestActionableWarningIsAbsentWhenWarningsAreNotInTheRow(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/division-deployments": []map[string]any{{"id": 4, "oob_file": "HAN_1936"}},
	})
	if _, present := summaryOf(t, fixture, "division_deployment", nil)["has_actionable_warning"]; present {
		t.Fatal("has_actionable_warning was invented with no warnings field present")
	}
}

func TestShipPrideOfTheFleetIsReadOutOfBodyAst(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/oob-units/ships": []map[string]any{{
			"id": 22, "name": "NRB Minas Gerasis", "definition": "battleship",
			"body_ast": []any{
				map[string]any{"key": "pride_of_the_fleet", "op": "=", "value": "yes"},
			},
		}},
	})
	if got := summaryOf(t, fixture, "ship", nil)["pride_of_the_fleet"]; got != true {
		t.Errorf("pride_of_the_fleet = %v, want true", got)
	}
}

func TestShipWithoutPrideFlagOmitsIt(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/oob-units/ships": []map[string]any{{
			"id": 23, "name": "NRB Humayta", "definition": "submarine", "body_ast": []any{},
		}},
	})
	if _, present := summaryOf(t, fixture, "ship", nil)["pride_of_the_fleet"]; present {
		t.Fatal("pride_of_the_fleet was reported for a ship that does not carry it")
	}
}

func TestRawFileSizeAndReferences(t *testing.T) {
	script := "story = {\n    country_event = { id = han_se.1100 }\n    add_ideas = han_mandate\n}\n"
	fixture := newDerivedFixture(map[string]any{
		"/raw-files": []map[string]any{{
			"id": 31, "dest_path": "common/scripted_effects/story.txt",
			"is_binary": false, "content": script, "generate_replace_path": true,
			"raw_kind": "verbatim_override",
		}},
	})
	summary := summaryOf(t, fixture, "raw_file", nil)
	if summary["size"] != float64(len(script)) {
		t.Errorf("size = %v, want %d", summary["size"], len(script))
	}

	refs, ok := summary["references"].([]any)
	if !ok {
		t.Fatalf("references missing: %v", summary)
	}
	got := map[string]bool{}
	for _, ref := range refs {
		got[ref.(string)] = true
	}
	if !got["event:han_se.1100"] || !got["idea:han_mandate"] {
		t.Errorf("references = %v", refs)
	}

	if summary["generate_replace_path"] != true {
		t.Errorf("generate_replace_path = %v", summary["generate_replace_path"])
	}
}

func TestBinaryRawFileReportsNoSizeRatherThanZero(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/raw-files": []map[string]any{{
			"id": 32, "dest_path": "gfx/flag.dds", "is_binary": true, "content": "",
		}},
	})
	summary := summaryOf(t, fixture, "raw_file", nil)
	if _, present := summary["size"]; present {
		t.Fatalf("size reported for a binary file: %v", summary)
	}
	if _, present := summary["references"]; present {
		t.Fatalf("references reported for a binary file: %v", summary)
	}
}

func TestDecisionCategoryCountsItsDecisions(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/decisions/categories": []map[string]any{
			{"id": 7, "category_id": "han_internal"},
			{"id": 8, "category_id": "han_foreign"},
		},
		"/decisions": []map[string]any{
			{"id": 1, "decision_id": "a", "category_id": 7},
			{"id": 2, "decision_id": "b", "category_id": 7},
			{"id": 3, "decision_id": "c", "category_id": 8},
		},
	})
	session := genericReadSession(t, fixture.handler(t))
	result := callTool(t, session, "list_entities", map[string]any{
		"project_id": 296, "kind": "decision_category",
	})
	entities := result.StructuredContent.(map[string]any)["entities"].([]any)
	first := entities[0].(map[string]any)["summary"].(map[string]any)
	second := entities[1].(map[string]any)["summary"].(map[string]any)
	if first["decision_count"] != float64(2) || second["decision_count"] != float64(1) {
		t.Fatalf("counts = %v / %v", first, second)
	}

	if hits := fixture.hitCount("/api/projects/296/decisions"); hits != 1 {
		t.Errorf("child table fetched %d times, want exactly 1", hits)
	}
}

func TestEventNamespaceCountsItsEvents(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/events/namespaces": []map[string]any{{"id": 3, "namespace": "sanguo_shu"}},
		"/events": []map[string]any{
			{"id": 1, "event_id": "sanguo_shu.1", "namespace_id": 3},
			{"id": 2, "event_id": "sanguo_shu.2", "namespace_id": 3},
		},
	})
	if got := summaryOf(t, fixture, "event_namespace", nil)["event_count"]; got != float64(2) {
		t.Errorf("event_count = %v, want 2", got)
	}
}

func TestSpecializationCountsProjectsAndSubSpecializations(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/special-projects/specializations": []map[string]any{{
			"id": 5, "specialization_id": "specialization_nuclear",
			"sub_specializations": []any{"sub_a", "sub_b", "sub_c"},
		}},
		"/special-projects/projects": []map[string]any{
			{"id": 1, "project_token": "sp_reactor", "specialization": "specialization_nuclear"},
			{"id": 2, "project_token": "sp_bomb", "specialization": "specialization_nuclear"},
			{"id": 3, "project_token": "sp_jet", "specialization": "specialization_aero"},
		},
	})
	summary := summaryOf(t, fixture, "specialization", nil)
	if summary["project_count"] != float64(2) {
		t.Errorf("project_count = %v, want 2", summary["project_count"])
	}
	if summary["sub_specialization_count"] != float64(3) {
		t.Errorf("sub_specialization_count = %v, want 3", summary["sub_specialization_count"])
	}
}

func TestOOBNestingCounts(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/oob-units/air-wing-groups": []map[string]any{{"id": 10, "oob_file": "HAN_1936", "location": "500"}},
		"/oob-units/air-wings": []map[string]any{
			{"id": 11, "group_id": 10}, {"id": 12, "group_id": 10}, {"id": 13, "group_id": 99},
		},
		"/oob-units/fleets":      []map[string]any{{"id": 20, "name": "Home Fleet", "naval_base": "10980"}},
		"/oob-units/task-forces": []map[string]any{{"id": 21, "fleet_id": 20}, {"id": 24, "fleet_id": 20}},
		"/oob-units/ships": []map[string]any{
			{"id": 22, "task_force_id": 21}, {"id": 23, "task_force_id": 21},
		},
	})
	if got := summaryOf(t, fixture, "air_wing_group", nil)["wing_count"]; got != float64(2) {
		t.Errorf("wing_count = %v, want 2", got)
	}
	if got := summaryOf(t, fixture, "fleet", nil)["task_force_count"]; got != float64(2) {
		t.Errorf("task_force_count = %v, want 2", got)
	}
	if got := summaryOf(t, fixture, "task_force", nil)["ship_count"]; got != float64(2) {
		t.Errorf("ship_count = %v, want 2", got)
	}
}

func TestUnreadableChildTableYieldsNoCountRatherThanZero(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/decisions/categories": []map[string]any{{"id": 7, "category_id": "han_internal"}},
	})
	session := genericReadSession(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/decisions") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		fixture.handler(t)(w, r)
	})
	result := callTool(t, session, "list_entities", map[string]any{
		"project_id": 296, "kind": "decision_category",
	})
	if result.IsError {
		t.Fatalf("an unreadable child table broke the whole listing: %+v", result.Content)
	}
	entities := result.StructuredContent.(map[string]any)["entities"].([]any)
	summary, _ := entities[0].(map[string]any)["summary"].(map[string]any)
	if _, present := summary["decision_count"]; present {
		t.Fatalf("decision_count was reported despite an unreadable child table: %v", summary)
	}
}

func TestEverySummaryFieldHasAProducer(t *testing.T) {

	derived := map[string]bool{
		"node_count": true, "shared_ref_count": true, "category_count": true,
		"sub_specialization_count": true, "advanced_extras_count": true,
		"has_actionable_warning": true, "pride_of_the_fleet": true,
		"size": true, "references": true, "configured_fields": true,
		"category_key": true,

		"decision_count": true, "event_count": true, "project_count": true,
		"wing_count": true, "task_force_count": true, "ship_count": true,
	}
	childCountFields := map[string]bool{}
	for _, kind := range entityKinds {
		if kind.ChildCount != nil {
			childCountFields[kind.ChildCount.Field] = true
			if !contains(strings.Join(kind.Summary, ","), kind.ChildCount.Field) {
				t.Errorf("%s declares child count %q but does not list it in Summary",
					kind.Kind, kind.ChildCount.Field)
			}
		}
	}
	for _, kind := range entityKinds {
		for _, field := range kind.Summary {
			if !derived[field] {
				continue
			}
			if childCountFields[field] {
				continue
			}

			if _, ok := summaryValue(map[string]any{}, field); ok {
				t.Errorf("%s.%s produced a value from an empty row", kind.Kind, field)
			}
		}
	}
}

func TestDecisionShowsItsCategoryByName(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/decisions": []map[string]any{{
			"id": 1, "decision_id": "han_reform", "category_id": 17, "cost": 50,
		}},
		"/decisions/categories": []map[string]any{
			{"id": 17, "category_id": "han_internal"},
			{"id": 18, "category_id": "han_foreign"},
		},
	})
	summary := summaryOf(t, fixture, "decision", nil)
	if summary["category_key"] != "han_internal" {
		t.Fatalf("category_key = %v, want the category's own name — an int row id "+
			"is exactly what this revision existed to remove", summary["category_key"])
	}

	if hits := fixture.hitCount("/api/projects/296/decisions/categories"); hits != 1 {
		t.Errorf("parent table fetched %d times, want exactly 1", hits)
	}
}

func TestUnresolvableParentOmitsTheName(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/decisions":            []map[string]any{{"id": 1, "decision_id": "x", "category_id": 999}},
		"/decisions/categories": []map[string]any{{"id": 17, "category_id": "han_internal"}},
	})
	if _, present := summaryOf(t, fixture, "decision", nil)["category_key"]; present {
		t.Fatal("a dangling category link produced a name")
	}
}

func TestOOBParentsCarryBothIDAndName(t *testing.T) {
	fixture := newDerivedFixture(map[string]any{
		"/oob-units/fleets":      []map[string]any{{"id": 20, "name": "Home Fleet"}},
		"/oob-units/task-forces": []map[string]any{{"id": 21, "fleet_id": 20, "name": "1st Task Force"}},
		"/oob-units/ships":       []map[string]any{{"id": 22, "task_force_id": 21, "name": "KMS Test"}},
	})
	tf := summaryOf(t, fixture, "task_force", nil)
	if tf["fleet_id"] != float64(20) || tf["fleet_name"] != "Home Fleet" {
		t.Errorf("task_force summary = %v", tf)
	}
	ship := summaryOf(t, fixture, "ship", nil)
	if ship["task_force_id"] != float64(21) || ship["task_force_name"] != "1st Task Force" {
		t.Errorf("ship summary = %v", ship)
	}
}

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }
