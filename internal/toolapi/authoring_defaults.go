package toolapi

import (
	"context"
	"encoding/json"
	"strings"
)

func authoringDefaultsMiddleware(proxy *RESTProxy) Middleware {
	return func(next MethodHandler) MethodHandler {
		return func(ctx context.Context, method string, req Request) (Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}
			params, ok := req.GetParams().(*CallToolParamsRaw)
			if !ok || params == nil || !usesAutomaticAuthoringDefaults(params.Name) {
				return next(ctx, method, req)
			}
			args := map[string]any{}
			_ = json.Unmarshal(params.Arguments, &args)
			projectID := numericProjectID(args["project_id"])
			if projectID <= 0 {
				return next(ctx, method, req)
			}
			coverage := ""
			if params.Name == "create_bookmark" || params.Name == "update_bookmark" {
				coverage, _ = args["date"].(string)
			}
			languages := authoredLocales(args)
			settings, err := ensureProjectSettings(ctx, proxy, projectID, projectSettingsEnsureRequest{
				Languages: languages, EnsureEndDate: true, CoverageDate: coverage,
			})
			if err != nil {
				return nil, err
			}

			result, err := next(ctx, method, req)
			if err != nil {
				return result, err
			}
			call, ok := result.(*CallToolResult)
			if !ok || call == nil || call.IsError || call.StructuredContent == nil {
				return result, nil
			}
			fields, ok := structuredFields(call.StructuredContent)
			if !ok {
				return result, nil
			}
			meta := map[string]any{
				"export_languages": settings.Snapshot.ExportLanguages,
				"end_date":         settings.Snapshot.EndDate,
				"timeline_status":  settings.TimelineStatus,
				"changed_fields":   settings.ChangedFields,
			}
			if len(settings.Warnings) > 0 {
				meta["warnings"] = settings.Warnings
			}
			fields["authoring_defaults"] = meta
			call.StructuredContent = fields
			return result, nil
		}
	}
}

func authoredLocales(args map[string]any) []string {
	found := map[string]bool{}
	var visit func(string, any)
	visit = func(parent string, value any) {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				lower := strings.ToLower(strings.TrimSpace(key))
				for code := range supportedProjectLocales {
					if (strings.HasSuffix(lower, "_"+code) ||
						(strings.Contains(parent, "loc") || parent == "name" || parent == "desc") && lower == code) &&
						nonBlankLocalisedValue(child) {
						found[code] = true
					}
				}
				visit(lower, child)
			}
		case []any:
			for _, child := range typed {
				visit(parent, child)
			}
		}
	}
	visit("", args)
	order := []string{"en", "zh", "ru", "fr", "de", "es", "ja", "ko", "pl", "pt"}
	out := make([]string, 0, len(found))
	for _, code := range order {
		if found[code] {
			out = append(out, code)
		}
	}
	return out
}

func nonBlankLocalisedValue(value any) bool {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) != ""
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

func usesAutomaticAuthoringDefaults(toolName string) bool {
	if !isMutationToolName(toolName) || toolName == "ensure_project_settings" {
		return false
	}

	for _, prefix := range []string{"delete_", "discard_", "preview_", "queue_", "undo_", "restore_"} {
		if strings.HasPrefix(toolName, prefix) {
			return false
		}
	}

	return !worldImageWorkingTools[toolName]
}
