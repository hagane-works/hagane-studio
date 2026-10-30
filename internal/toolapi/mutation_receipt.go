package toolapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"haganeworks/internal/domains/countries"
)

type mutationReceipt struct {
	Tool               string `json:"tool"`
	ProjectID          int64  `json:"project_id"`
	WriteStatus        string `json:"write_status"`
	VerificationStatus string `json:"verification_status"`
	SavedValuesSource  string `json:"saved_values_source"`
	ReportingRule      string `json:"reporting_rule"`
}

type mutationReferencePolicy struct {
	SourceKind string
	IDKeys     []string
	InputKeys  []string
}

var relationScriptFieldsByKind = map[string][]string{
	"focus": {
		"completion_reward", "select_effect", "available", "bypass", "cancel",
	},
	"event": {
		"trigger", "immediate", "after", "mean_time_to_happen", "options",
	},
	"decision": {
		"complete_effect", "remove_effect", "timeout_effect", "cancel_effect", "allowed",
		"available", "visible", "cancel_trigger", "activation", "ai_will_do",
	},
	"idea": {
		"on_add", "on_remove", "do_effect", "allowed", "available", "visible",
		"allowed_civil_war", "allowed_to_remove", "cancel",
	},
	"character": {
		"advisor_on_add", "advisor_on_remove", "advisor_available", "advisor_visible", "advisor_modifier",
	},
}

var mutationReferencePolicies = map[string]mutationReferencePolicy{
	"create_focus": {SourceKind: "focus", IDKeys: []string{"focus_node_id", "node_id"}, InputKeys: []string{"completion_reward"}},
	"update_focus": {SourceKind: "focus", IDKeys: []string{"focus_node_id", "node_id"}, InputKeys: []string{"completion_reward", "available"}},

	"create_event":     {SourceKind: "event", IDKeys: []string{"event_db_id"}, InputKeys: []string{"trigger", "immediate", "after", "mean_time_to_happen", "options"}},
	"update_event":     {SourceKind: "event", IDKeys: []string{"event_db_id"}, InputKeys: []string{"trigger", "immediate", "after", "mean_time_to_happen", "options"}},
	"add_event_option": {SourceKind: "event", IDKeys: []string{"event_db_id"}, InputKeys: []string{"trigger", "effects", "hidden_effects"}},

	"create_decision": {SourceKind: "decision", IDKeys: []string{"decision_db_id"}, InputKeys: []string{"available", "visible", "complete_effect", "ai_will_do"}},
	"update_decision": {SourceKind: "decision", IDKeys: []string{"decision_db_id"}, InputKeys: []string{"available", "visible", "complete_effect", "ai_will_do"}},

	"create_idea": {SourceKind: "idea", IDKeys: []string{"idea_db_id"}, InputKeys: []string{"allowed"}},
	"update_idea": {SourceKind: "idea", IDKeys: []string{"idea_db_id"}, InputKeys: []string{"allowed"}},
}

var mutationPrefixes = []string{
	"acquire_", "add_", "apply_", "assign_", "attach_", "batch_", "cancel_",
	"clear_", "compile_", "compose_", "connect_", "create_", "delete_", "derive_",
	"detach_", "discard_", "edit_", "ensure_", "export_", "generate_", "import_",
	"instantiate_", "materialize_", "name_", "publish_", "queue_", "remove_",
	"rename_", "repair_", "replace_", "reuse_", "save_", "seed_", "select_", "set_", "stage_",
	"start_", "unlist_", "update_", "upload_",
}

func isMutationToolName(name string) bool {
	if name == "confirm_technology_surface_environment" {
		return true
	}
	switch name {
	case "undo_world_map_change", "restore_world_map":
		return true
	}
	if strings.HasPrefix(name, "preview_") {
		return false
	}
	for _, prefix := range mutationPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func mutationContractSentence(referenceAware bool) string {
	base := "On success inspect mutation_receipt: write_status only confirms persistence; summarize saved values from the tool result, then re-read and validate before claiming gameplay success."
	if referenceAware {
		base += " When this call supplies script-bearing fields, reference_check=failed or unavailable means the task is unfinished."
	}
	return base
}

func mutationReceiptMiddleware(proxy *RESTProxy) Middleware {
	return func(next MethodHandler) MethodHandler {
		return func(ctx context.Context, method string, req Request) (Result, error) {
			if method == "tools/list" {
				result, err := next(ctx, method, req)
				if err == nil {
					addMutationContractToToolList(result)
				}
				return result, err
			}

			var toolName string
			args := map[string]any{}
			if method == "tools/call" {
				if params, ok := req.GetParams().(*CallToolParamsRaw); ok && params != nil {
					toolName = params.Name
					_ = json.Unmarshal(params.Arguments, &args)
				}
			}

			result, err := next(ctx, method, req)
			if err != nil || method != "tools/call" || !isMutationToolName(toolName) {
				return result, err
			}
			call, ok := result.(*CallToolResult)
			if !ok || call == nil || call.IsError || call.StructuredContent == nil {
				return result, err
			}
			fields, ok := structuredFields(call.StructuredContent)
			if !ok {
				return result, err
			}

			if policy, exists := mutationReferencePolicies[toolName]; exists && referenceCheckRequested(args, policy.InputKeys) {
				status := strings.TrimSpace(stringValue(fields["reference_check"]))
				if status == "" || status == "not_applicable" {
					applyMutationReferenceCheck(ctx, proxy, numericProjectID(args["project_id"]), args, fields, policy)
				}
			}

			receipt := buildMutationReceipt(toolName, numericProjectID(args["project_id"]), fields)
			fields["mutation_receipt"] = receipt
			call.StructuredContent = fields
			return result, err
		}
	}
}

func addMutationContractToToolList(result Result) {
	list, ok := result.(*ListToolsResult)
	if !ok || list == nil {
		return
	}
	for index, original := range list.Tools {
		if original == nil || !isMutationToolName(original.Name) {
			continue
		}
		copyTool := *original
		tool := &copyTool
		list.Tools[index] = tool
		referenceAware := false
		if _, exists := mutationReferencePolicies[tool.Name]; exists {
			referenceAware = true
		}
		sentence := mutationContractSentence(referenceAware)
		if !strings.Contains(tool.Description, "mutation_receipt") {
			description := strings.TrimSpace(tool.Description)
			if description != "" && !strings.HasSuffix(description, ".") {
				description += "."
			}
			if description != "" {
				description += " "
			}
			tool.Description = description + sentence
		}
	}
}

func referenceCheckRequested(args map[string]any, keys []string) bool {
	for _, key := range keys {
		if value, present := args[key]; present && containsIndexedReference(value) {
			return true
		}
	}
	return false
}

func relationFieldsTouched(kind string, fields map[string]any) bool {
	for _, key := range relationScriptFieldsByKind[kind] {
		if value, present := fields[key]; present && containsIndexedReference(value) {
			return true
		}
	}
	return false
}

func containsIndexedReference(value any) bool {
	switch typed := value.(type) {
	case string:
		return countries.HasRelationRefs(typed)
	case map[string]any:
		for _, child := range typed {
			if containsIndexedReference(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsIndexedReference(child) {
				return true
			}
		}
	case []map[string]any:
		for _, child := range typed {
			if containsIndexedReference(child) {
				return true
			}
		}
	}
	return false
}

func applyMutationReferenceCheck(ctx context.Context, proxy *RESTProxy, projectID int64, args, fields map[string]any, policy mutationReferencePolicy) {
	sourceID := int64(0)
	for _, key := range policy.IDKeys {
		sourceID = numericProjectID(fields[key])
		if sourceID > 0 {
			break
		}
		sourceID = numericProjectID(args[key])
		if sourceID > 0 {
			break
		}
	}
	status, unresolved, warnings := checkEntityReferences(ctx, proxy, projectID, policy.SourceKind, sourceID, true)
	fields["reference_check"] = status
	fields["unresolved_references"] = unresolved
	if len(warnings) > 0 {
		fields["warnings"] = appendStringValues(fields["warnings"], warnings...)
	}
}

func checkEntityReferences(ctx context.Context, proxy *RESTProxy, projectID int64, sourceKind string, sourceID int64, requested bool) (string, []DanglingReference, []string) {
	unresolved := []DanglingReference{}
	if !requested {
		return "not_applicable", unresolved, nil
	}
	if proxy == nil || projectID <= 0 || sourceID <= 0 || strings.TrimSpace(sourceKind) == "" {
		return "unavailable", unresolved, []string{
			"REFERENCE CHECK UNAVAILABLE: the write committed, but the saved entity could not be identified for post-write verification. Do not report its scripted effects as working; re-read it and run validate_project.",
		}
	}
	resp, err := proxyAuthedCall(ctx, proxy, "GET",
		fmt.Sprintf("/api/projects/%d/relations/entity/%s/%d", projectID, sourceKind, sourceID), nil)
	if err != nil {
		return "unavailable", unresolved, []string{
			"REFERENCE CHECK UNAVAILABLE: the write committed, but its outgoing references were not verified. Do not report its scripted effects as working; re-read it and run validate_project.",
		}
	}
	rows, _ := resp["outgoing"].([]any)
	if len(rows) == 0 {
		return "unavailable", unresolved, []string{
			"REFERENCE CHECK UNAVAILABLE: the submitted script contained indexed references, but the saved entity returned no outgoing edges. Re-read the field and run validate_project before claiming the behavior works.",
		}
	}
	for _, raw := range rows {
		edge, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		resolved, present := edge["resolved"].(bool)
		if !present || resolved {
			continue
		}
		unresolved = append(unresolved, DanglingReference{
			SourceKind: stringField(edge, "source_kind"),
			SourceID:   intField(edge, "source_id"),
			SourceName: stringField(edge, "source_name"),
			FieldName:  stringField(edge, "field_name"),
			EffectName: stringField(edge, "effect_name"),
			TargetType: stringField(edge, "target_type"),
			TargetID:   stringField(edge, "target_id"),
		})
	}
	if len(unresolved) == 0 {
		return "passed", unresolved, nil
	}
	warnings := make([]string, 0, len(unresolved))
	for _, ref := range unresolved {
		warnings = append(warnings, fmt.Sprintf(
			"UNRESOLVED REFERENCE: %s uses %s -> %s %s, but that target does not exist. Create it or remove the reference before claiming success.",
			ref.FieldName, ref.EffectName, ref.TargetType, ref.TargetID,
		))
	}
	return "failed", unresolved, warnings
}

func buildMutationReceipt(toolName string, projectID int64, fields map[string]any) mutationReceipt {
	writeStatus := "committed"
	verificationStatus := "not_run"
	reportingRule := "The write committed, but semantic verification did not run. Report only values returned by this result; re-read and validate before claiming completion."

	failed := integerValue(fields["failed"])
	if failed > 0 {
		succeeded := integerValue(fields["updated"])
		if succeeded == 0 {
			succeeded = integerValue(fields["applied"])
		}
		if succeeded > 0 {
			writeStatus = "partially_committed"
		} else {
			writeStatus = "no_change"
		}
		verificationStatus = "issues_found"
		reportingRule = "The batch contains failed rows. Report the successful and failed result rows separately; never describe the full batch as completed."
	}
	if okValue, present := fields["ok"].(bool); present && !okValue {
		writeStatus = "no_change"
		verificationStatus = "issues_found"
		reportingRule = "The operation returned ok=false. Do not claim that the requested change was applied."
	}

	switch strongestReferenceCheck(fields) {
	case "passed":
		if verificationStatus != "issues_found" {
			verificationStatus = "references_passed"
			reportingRule = "The write committed and all indexed outgoing references resolve. Report saved values from this result; gameplay semantics still require the narrow relevant validation."
		}
	case "failed":
		verificationStatus = "issues_found"
		reportingRule = "The write committed with unresolved references. Do not claim the affected behavior is complete or working until those references are repaired and checked again."
	case "unavailable":
		verificationStatus = "verification_unavailable"
		reportingRule = "The write committed, but reference verification was unavailable. Re-read and run validate_project before claiming completion."
	}

	if diagnosticsHaveActiveFindings(fields["diagnostics"]) || diagnosticsHaveActiveFindings(fields["tree_diagnostics"]) {
		if verificationStatus == "not_run" || verificationStatus == "references_passed" {
			verificationStatus = "diagnostics_found"
			reportingRule = "The write committed with active diagnostics. Report them and validate after repair; do not summarize the change as clean."
		}
	}

	proof := inspectExplicitMutationProof(fields)
	if len(proof.failedPrimary) > 0 {
		verificationStatus = "issues_found"
		reportingRule = fmt.Sprintf(
			"The tool explicitly returned %s. The write may have persisted, but do not claim the requested operation is complete.",
			strings.Join(proof.failedPrimary, ", "),
		)
	} else if len(proof.failedRelated) > 0 && proof.primaryPassed {
		verificationStatus = "issues_found"
		reportingRule = fmt.Sprintf(
			"The tool reported completion or validity, but these explicit verification proofs failed: %s. Do not claim completion until they pass.",
			strings.Join(proof.failedRelated, ", "),
		)
	} else if proof.primaryPassed && proof.relatedPresent && proof.relatedPassed &&
		verificationStatus != "issues_found" && verificationStatus != "diagnostics_found" && verificationStatus != "verification_unavailable" {
		verificationStatus = "verified"
		reportingRule = "The tool explicitly reported completion or validity and its returned proofs passed. Report completion only within those stated scopes; this does not prove in-game or runtime behavior."
	}

	return mutationReceipt{
		Tool:               toolName,
		ProjectID:          projectID,
		WriteStatus:        writeStatus,
		VerificationStatus: verificationStatus,
		SavedValuesSource:  "tool_result",
		ReportingRule:      reportingRule,
	}
}

type explicitMutationProof struct {
	primaryPassed  bool
	failedPrimary  []string
	relatedPresent bool
	relatedPassed  bool
	failedRelated  []string
}

func inspectExplicitMutationProof(fields map[string]any) explicitMutationProof {
	proof := explicitMutationProof{relatedPassed: true}
	for _, key := range []string{"completed", "valid"} {
		value, present := fields[key].(bool)
		if !present {
			continue
		}
		if value {
			proof.primaryPassed = true
		} else {
			proof.failedPrimary = append(proof.failedPrimary, key+"=false")
		}
	}
	for _, key := range []string{
		"persistence_verified",
		"authoritative_validation",
		"readback_verified",
		"references_verified",
		"sprite_references_verified",
		"localisation_verified",
		"structurally_valid",
		"references_valid",
		"compiled_readback_valid",
		"export_valid",
	} {
		value, present := fields[key].(bool)
		if !present {
			continue
		}
		proof.relatedPresent = true
		if !value {
			proof.relatedPassed = false
			proof.failedRelated = append(proof.failedRelated, key+"=false")
		}
	}
	return proof
}

func strongestReferenceCheck(value any) string {
	strongest := ""
	var visit func(any)
	visit = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			if status := strings.TrimSpace(stringValue(typed["reference_check"])); status != "" {
				switch status {
				case "failed":
					strongest = "failed"
				case "unavailable":
					if strongest != "failed" {
						strongest = "unavailable"
					}
				case "passed":
					if strongest == "" || strongest == "not_applicable" {
						strongest = "passed"
					}
				case "not_applicable":
					if strongest == "" {
						strongest = "not_applicable"
					}
				}
			}
			for key, child := range typed {
				if key != "mutation_receipt" && key != "reference_check" {
					visit(child)
				}
			}
		case []any:
			for _, child := range typed {
				visit(child)
			}
		}
	}
	visit(value)
	return strongest
}

func diagnosticsHaveActiveFindings(value any) bool {
	visit := func(row map[string]any) bool {
		return !strings.EqualFold(stringValue(row["change"]), "resolved")
	}
	switch rows := value.(type) {
	case []any:
		for _, raw := range rows {
			if row, ok := raw.(map[string]any); ok && visit(row) {
				return true
			}
		}
	case []map[string]any:
		for _, row := range rows {
			if visit(row) {
				return true
			}
		}
	}
	return false
}

func appendStringValues(existing any, values ...string) []string {
	out := []string{}
	switch typed := existing.(type) {
	case []string:
		out = append(out, typed...)
	case []any:
		for _, value := range typed {
			if text, ok := value.(string); ok {
				out = append(out, text)
			}
		}
	}
	return append(out, values...)
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func integerValue(value any) int64 {
	return numericProjectID(value)
}

func numericProjectID(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return parsed
	}
	return 0
}
