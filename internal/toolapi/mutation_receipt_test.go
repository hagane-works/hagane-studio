package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMutationReceiptSeparatesPersistenceFromVerification(t *testing.T) {
	middleware := mutationReceiptMiddleware(nil)
	handler := middleware(func(context.Context, string, Request) (Result, error) {
		return &CallToolResult{StructuredContent: map[string]any{
			"country_id": float64(12),
			"tag":        "PER",
		}}, nil
	})

	result, err := handler(context.Background(), "tools/call", toolRequest(
		"create_country", map[string]any{"project_id": 69, "tag": "PER"},
	))
	if err != nil {
		t.Fatal(err)
	}
	call := result.(*CallToolResult)
	fields := call.StructuredContent.(map[string]any)
	receipt, ok := fields["mutation_receipt"].(mutationReceipt)
	if !ok {
		t.Fatalf("mutation_receipt type = %T, want mutationReceipt", fields["mutation_receipt"])
	}
	if receipt.WriteStatus != "committed" || receipt.VerificationStatus != "not_run" {
		t.Fatalf("receipt = %+v", receipt)
	}
	if receipt.SavedValuesSource != "tool_result" {
		t.Errorf("saved_values_source = %q", receipt.SavedValuesSource)
	}
}

func TestBusinessFailureDoesNotGetASuccessMutationReceipt(t *testing.T) {
	handler := mutationReceiptMiddleware(nil)(func(context.Context, string, Request) (Result, error) {
		return &CallToolResult{IsError: true, StructuredContent: map[string]any{
			"ok": false, "error": "GFX_alerts frame 3: usage, evidence", "recoverable": true,
			"batch_id": "original-batch", "item_ids": []int64{11, 0},
		}}, nil
	})
	result, err := handler(context.Background(), "tools/call", toolRequest("create_idea", map[string]any{"project_id": 7}))
	if err != nil {
		t.Fatal(err)
	}
	call := result.(*CallToolResult)
	fields := call.StructuredContent.(map[string]any)
	if !call.IsError || fields["mutation_receipt"] != nil || fields["batch_id"] != "original-batch" {
		t.Fatalf("business failure gained a success receipt or lost progress: %+v", call)
	}
}

func TestMutationReceiptChecksEventReferencesAfterWrite(t *testing.T) {
	var relationCalls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/projects/7/events":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 21, "event_id": "per_army.1", "event_number": 1, "warnings": []any{},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/projects/7/relations/entity/event/21":
			relationCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"outgoing": []any{
				map[string]any{
					"source_kind": "event", "source_id": 21, "source_name": "per_army.1",
					"field_name": "immediate", "effect_name": "add_ideas",
					"target_type": "idea", "target_id": "per_missing_idea", "resolved": false,
				},
			}})
		default:
			t.Fatalf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		s.AddReceivingMiddleware(mutationReceiptMiddleware(proxy))
		registerCreateEvent(s, proxy)
	})
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_event",
		Arguments: map[string]any{
			"project_id": 7, "namespace_id": 3, "event_id": "per_army.1", "event_number": 1,
			"immediate": "add_ideas = per_missing_idea",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("write should remain successful despite reference finding: %s", contentText(result))
	}
	if relationCalls != 1 {
		t.Fatalf("relation checks = %d, want 1", relationCalls)
	}
	fields := result.StructuredContent.(map[string]any)
	if fields["reference_check"] != "failed" {
		t.Fatalf("reference_check = %v", fields["reference_check"])
	}
	unresolved, ok := fields["unresolved_references"].([]any)
	if !ok || len(unresolved) != 1 {
		t.Fatalf("unresolved_references = %#v", fields["unresolved_references"])
	}
	receipt := fields["mutation_receipt"].(map[string]any)
	if receipt["write_status"] != "committed" || receipt["verification_status"] != "issues_found" {
		t.Fatalf("mutation_receipt = %#v", receipt)
	}
	if encoded, _ := json.Marshal(unresolved); !strings.Contains(string(encoded), "per_missing_idea") {
		t.Fatal("unresolved target was not named in unresolved_references")
	}
}

func TestMutationReceiptDoesNotTreatMissingRelationEdgesAsPassed(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/projects/7/events":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 21, "event_id": "per_army.1", "event_number": 1, "warnings": []any{},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/projects/7/relations/entity/event/21":
			_ = json.NewEncoder(w).Encode(map[string]any{"outgoing": []any{}})
		default:
			t.Fatalf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		s.AddReceivingMiddleware(mutationReceiptMiddleware(proxy))
		registerCreateEvent(s, proxy)
	})
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_event",
		Arguments: map[string]any{
			"project_id": 7, "namespace_id": 3, "event_id": "per_army.1", "event_number": 1,
			"immediate": "add_ideas = per_missing_idea",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fields := result.StructuredContent.(map[string]any)
	if fields["reference_check"] != "unavailable" {
		t.Fatalf("reference_check = %v, want unavailable", fields["reference_check"])
	}
	receipt := fields["mutation_receipt"].(map[string]any)
	if receipt["verification_status"] != "verification_unavailable" {
		t.Fatalf("mutation_receipt = %#v", receipt)
	}
}

func TestMutationContractAppearsOnEveryRegisteredMutation(t *testing.T) {
	for _, tool := range listAllTools(t) {
		if !isMutationToolName(tool.Name) {
			continue
		}
		if !strings.Contains(tool.Description, "mutation_receipt") {
			t.Errorf("mutation tool %q has no mutation_receipt contract", tool.Name)
		}
	}
}

func TestEveryWriteToolIsClassifiedForMutationReceipts(t *testing.T) {
	for _, tool := range listAllTools(t) {
		name := tool.Name
		if isReadOnlyToolName(name) || strings.HasPrefix(name, "validate_") ||
			strings.HasPrefix(name, "preview_") || name == "export_project" {
			continue
		}
		if _, mapped := toolFieldRef[name]; !mapped {
			if _, exempt := toolFieldRefExempt[name]; !exempt {
				continue
			}
		}
		if name != "test" && !isMutationToolName(name) {
			t.Errorf("write tool %q is not classified for mutation receipts", name)
		}
	}
}

func TestBatchFailureReceiptIsPartialNotCleanSuccess(t *testing.T) {
	fields := map[string]any{
		"requested": float64(3), "updated": float64(2), "failed": float64(1),
	}
	receipt := buildMutationReceipt("batch_update_entities", 7, fields)
	if receipt.WriteStatus != "partially_committed" || receipt.VerificationStatus != "issues_found" {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestNestedBatchReferenceFailureMarksReceiptAsIssuesFound(t *testing.T) {
	fields := map[string]any{
		"requested": float64(1), "updated": float64(1), "failed": float64(0),
		"results": []any{map[string]any{
			"ok": true, "reference_check": "failed",
		}},
	}
	receipt := buildMutationReceipt("batch_update_entities", 7, fields)
	if receipt.WriteStatus != "committed" || receipt.VerificationStatus != "issues_found" {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestMutationReceiptExplicitIncompleteCannotClaimCompletion(t *testing.T) {
	fields := map[string]any{
		"completed":           false,
		"readback_verified":   true,
		"references_verified": true,
	}
	receipt := buildMutationReceipt("apply_gui_overlay_edits", 7, fields)
	if receipt.VerificationStatus != "issues_found" {
		t.Fatalf("verification_status = %q, want issues_found", receipt.VerificationStatus)
	}
	if !strings.Contains(receipt.ReportingRule, "completed=false") ||
		!strings.Contains(receipt.ReportingRule, "do not claim") {
		t.Fatalf("reporting_rule does not expose incomplete result: %q", receipt.ReportingRule)
	}
}

func TestMutationReceiptExplicitInvalidCannotClaimCompletion(t *testing.T) {
	fields := map[string]any{
		"valid":               false,
		"readback_verified":   true,
		"references_verified": true,
	}
	receipt := buildMutationReceipt("apply_gui_overlay_edits", 7, fields)
	if receipt.VerificationStatus != "issues_found" {
		t.Fatalf("verification_status = %q, want issues_found", receipt.VerificationStatus)
	}
	if !strings.Contains(receipt.ReportingRule, "valid=false") ||
		!strings.Contains(receipt.ReportingRule, "do not claim") {
		t.Fatalf("reporting_rule does not expose invalid result: %q", receipt.ReportingRule)
	}
}

func TestMutationReceiptVerifiedOnlyWithinReturnedProofScope(t *testing.T) {
	fields := map[string]any{
		"completed":                  true,
		"persistence_verified":       true,
		"authoritative_validation":   true,
		"readback_verified":          true,
		"references_verified":        true,
		"sprite_references_verified": true,
		"localisation_verified":      true,
	}
	receipt := buildMutationReceipt("apply_gui_overlay_edits", 7, fields)
	if receipt.VerificationStatus != "verified" {
		t.Fatalf("verification_status = %q, want verified; receipt=%+v", receipt.VerificationStatus, receipt)
	}
	if !strings.Contains(receipt.ReportingRule, "stated scopes") ||
		!strings.Contains(receipt.ReportingRule, "does not prove in-game or runtime behavior") {
		t.Fatalf("reporting_rule overstates verification: %q", receipt.ReportingRule)
	}
}

func TestMutationReceiptFailedRelatedProofBlocksExplicitCompletion(t *testing.T) {
	fields := map[string]any{
		"completed":           true,
		"readback_verified":   false,
		"references_verified": true,
	}
	receipt := buildMutationReceipt("apply_gui_overlay_edits", 7, fields)
	if receipt.VerificationStatus != "issues_found" {
		t.Fatalf("verification_status = %q, want issues_found", receipt.VerificationStatus)
	}
	if !strings.Contains(receipt.ReportingRule, "readback_verified=false") {
		t.Fatalf("reporting_rule does not name failed proof: %q", receipt.ReportingRule)
	}
}

func TestDiagnosticsHaveActiveFindingsSupportsTypedMapSlices(t *testing.T) {
	if !diagnosticsHaveActiveFindings([]map[string]any{{
		"severity": "error", "code": "broken",
	}}) {
		t.Fatal("typed []map diagnostics were ignored")
	}
	if diagnosticsHaveActiveFindings([]map[string]any{{
		"severity": "info", "code": "fixed", "change": "resolved",
	}}) {
		t.Fatal("resolved typed []map diagnostic was treated as active")
	}
	if !diagnosticsHaveActiveFindings([]any{map[string]any{
		"severity": "warning", "code": "still_active",
	}}) {
		t.Fatal("[]any diagnostics regression")
	}
}
