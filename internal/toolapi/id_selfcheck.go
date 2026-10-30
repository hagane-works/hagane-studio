package toolapi

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
)

func idReassignmentMiddleware() Middleware {
	return func(next MethodHandler) MethodHandler {
		return func(ctx context.Context, method string, req Request) (Result, error) {
			var requested map[string]any
			params, hasParams := req.GetParams().(*CallToolParamsRaw)
			if method == "tools/call" && hasParams && params != nil && len(params.Arguments) > 0 {
				_ = json.Unmarshal(params.Arguments, &requested)
			}

			if locale, ok := requested["locale"].(string); ok {
				ctx = withContentLocale(ctx, locale)
			}

			result, err := next(ctx, method, req)
			if err != nil || method != "tools/call" || requested == nil {
				return result, err
			}
			call, ok := result.(*CallToolResult)
			if !ok || call == nil || call.IsError || call.StructuredContent == nil {
				return result, err
			}
			returned, ok := structuredFields(call.StructuredContent)
			if !ok {
				return result, err
			}
			if notice := reassignmentNotice(requested, returned); notice != "" {
				returned["id_reassigned"] = notice
				call.StructuredContent = returned
			}
			return result, err
		}
	}
}

func structuredFields(structured any) (map[string]any, bool) {
	encoded, err := json.Marshal(structured)
	if err != nil {
		return nil, false
	}
	var fields map[string]any
	if json.Unmarshal(encoded, &fields) != nil {
		return nil, false
	}
	return fields, true
}

func reassignmentNotice(requested, returned map[string]any) string {
	var keys []string
	for key := range requested {
		if isBusinessIDKey(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var lines []string
	for _, key := range keys {
		asked, ok := requested[key].(string)
		if !ok || strings.TrimSpace(asked) == "" {
			continue
		}
		got, ok := returned[key].(string)
		if !ok || strings.TrimSpace(got) == "" {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(asked), strings.TrimSpace(got)) {
			continue
		}
		lines = append(lines, "  "+key+": you asked for \""+asked+"\", the program stored \""+got+"\"")
	}
	if len(lines) == 0 {
		return ""
	}
	return "⚠️ ID REASSIGNED — the program silently renumbered on a collision:\n" +
		strings.Join(lines, "\n") +
		"\nThe stored value is the real one. Every later reference (event calls, " +
		"prerequisites, effects, localisation keys) must use it, not the value you " +
		"requested. Nothing was rejected and the entity exists — do NOT retry the call."
}

func isBusinessIDKey(key string) bool {
	switch key {
	case "project_id", "id":
		return false
	case "namespace", "token", "project_token", "key", "tag", "country_tag",
		"name", "template_name", "entity_key", "wargoal_key", "specialization":
		return true
	}
	if strings.HasSuffix(key, "_db_id") {
		return false
	}
	return strings.HasSuffix(key, "_id") || strings.HasSuffix(key, "_key") ||
		strings.HasSuffix(key, "_token") || strings.HasSuffix(key, "_tag")
}
