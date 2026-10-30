package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestEnsureProjectSettingsMergesLanguageAndDerivesStandaloneEndDate(t *testing.T) {
	var putBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/projects/7":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 7, "start_date": "1936.1.1", "end_date": nil,
				"export_languages": []string{"en"}, "dependencies": []string{}, "basemod_template_id": nil,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/projects/7/bookmarks":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"bookmark_id": "late_start", "date": "1950.1.1"}})
		case r.Method == http.MethodPut && r.URL.Path == "/api/projects/7":
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 7, "start_date": "1936.1.1", "end_date": "2050.1.1",
				"export_languages": []string{"en", "zh"}, "dependencies": []string{},
			})
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer upstream.Close()

	result, err := ensureProjectSettings(context.Background(), NewRESTProxy(upstream.URL), 7,
		projectSettingsEnsureRequest{Languages: []string{"zh"}, EnsureEndDate: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(putBody["export_languages"], []any{"en", "zh"}) || putBody["end_date"] != "2050.1.1" {
		t.Fatalf("PUT body = %#v", putBody)
	}
	if result.Snapshot.EndDate == nil || *result.Snapshot.EndDate != "2050.1.1" ||
		!reflect.DeepEqual(result.Snapshot.ExportLanguages, []string{"en", "zh"}) ||
		!reflect.DeepEqual(result.ChangedFields, []string{"export_languages", "end_date"}) ||
		result.TimelineStatus != "configured" {
		t.Fatalf("result = %#v", result)
	}
}

func TestEnsureProjectSettingsDoesNotGuessBaseModEndDate(t *testing.T) {
	putCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPut {
			putCalled = true
			t.Fatal("dependent project without an explicit end date must not be updated")
		}
		if r.URL.Path == "/api/projects/8/bookmarks" {
			_ = json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 8, "start_date": "1962.1.1", "end_date": nil,
			"export_languages": []string{"en"}, "dependencies": []string{"TNO"},
		})
	}))
	defer upstream.Close()

	result, err := ensureProjectSettings(context.Background(), NewRESTProxy(upstream.URL), 8,
		projectSettingsEnsureRequest{EnsureEndDate: true})
	if err != nil {
		t.Fatal(err)
	}
	if putCalled || result.TimelineStatus != "requires_explicit_end_date" || len(result.Warnings) != 1 {
		t.Fatalf("result=%#v putCalled=%v", result, putCalled)
	}
}

func TestAuthoringDefaultsMergesAuthoredLocaleBeforeMutation(t *testing.T) {
	var sequence []string
	var putBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		sequence = append(sequence, r.Method)
		if r.Method == http.MethodGet && r.URL.Path == "/api/projects/9" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 9, "start_date": nil, "end_date": nil,
				"export_languages": []string{"en"}, "dependencies": []string{},
			})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/projects/9/bookmarks" {
			_ = json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&putBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 9, "start_date": nil, "end_date": "1949.1.1",
			"export_languages": []string{"en", "zh"}, "dependencies": []string{},
		})
	}))
	defer upstream.Close()

	next := func(_ context.Context, _ string, _ Request) (Result, error) {
		sequence = append(sequence, "mutation")
		return &CallToolResult{StructuredContent: map[string]any{"focus_id": "TST_one"}}, nil
	}
	handler := authoringDefaultsMiddleware(NewRESTProxy(upstream.URL))(next)
	request := &CallToolRequest{Params: &CallToolParamsRaw{
		Name: "create_focus", Arguments: json.RawMessage(`{"project_id":9,"focus_tree_id":2,"name":"Test","name_zh":"测试"}`),
	}}
	result, err := handler(context.Background(), "tools/call", request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sequence, []string{http.MethodGet, http.MethodGet, http.MethodPut, "mutation"}) {
		t.Fatalf("sequence = %v", sequence)
	}
	if !reflect.DeepEqual(putBody["export_languages"], []any{"en", "zh"}) || putBody["end_date"] != "1949.1.1" {
		t.Fatalf("PUT body = %#v", putBody)
	}
	call := result.(*CallToolResult)
	fields, ok := structuredFields(call.StructuredContent)
	if !ok {
		t.Fatalf("structured result = %#v", call.StructuredContent)
	}
	defaults, ok := fields["authoring_defaults"].(map[string]any)
	if !ok || defaults["timeline_status"] != "configured" {
		t.Fatalf("authoring_defaults = %#v", fields["authoring_defaults"])
	}
}
