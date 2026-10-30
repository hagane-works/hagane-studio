package toolapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type editTokensKey struct{}
type editReceiptKey struct{}

func editRevisionsMiddleware() Middleware {
	return func(next MethodHandler) MethodHandler {
		return func(ctx context.Context, method string, req Request) (Result, error) {
			receipt := map[string]string{}
			ctx = context.WithValue(ctx, editReceiptKey{}, receipt)
			if method == "tools/call" {
				if params, ok := req.GetParams().(*CallToolParamsRaw); ok && params != nil {
					var args map[string]json.RawMessage
					if json.Unmarshal(params.Arguments, &args) == nil {
						if raw, found := args["edit_revisions"]; found {
							var expected map[string]string
							if json.Unmarshal(raw, &expected) != nil || expected == nil {
								return nil, fmt.Errorf("edit_revisions must be a scope-to-decimal-string object")
							}
							for key, token := range expected {
								kind, ident, found := strings.Cut(key, ":")
								if !found || (kind != "bookmark" && kind != "focus_tree") || !canonicalRevision(ident) {
									return nil, fmt.Errorf("invalid edit revision scope")
								}
								if !canonicalRevision(token) {
									return nil, fmt.Errorf("edit revision must be a canonical positive decimal integer")
								}
							}
							ctx = context.WithValue(ctx, editTokensKey{}, expected)
							delete(args, "edit_revisions")
							params.Arguments, _ = json.Marshal(args)
						}
					}
				}
			}
			result, err := next(ctx, method, req)
			if call, ok := result.(*CallToolResult); ok && err == nil && !call.IsError && len(receipt) > 0 {

				if fields, ok := structuredFields(call.StructuredContent); ok && call.StructuredContent != nil {
					if _, taken := fields["edit_revisions"]; !taken {
						fields["edit_revisions"] = receipt
						call.StructuredContent = fields
					}
				}
			}
			if err == nil && method == "tools/list" {
				if list, ok := result.(*ListToolsResult); ok {
					for index, original := range list.Tools {
						if original == nil {
							continue
						}
						copyTool := *original
						tool := &copyTool
						list.Tools[index] = tool
						if tool == nil {
							continue
						}
						encoded, marshalErr := json.Marshal(tool.InputSchema)
						if marshalErr != nil {
							continue
						}
						var schema map[string]any
						if json.Unmarshal(encoded, &schema) != nil || schema == nil {
							continue
						}
						props, _ := schema["properties"].(map[string]any)
						if props == nil {
							props = map[string]any{}
							schema["properties"] = props
						}
						props["edit_revisions"] = map[string]any{
							"type": "object", "additionalProperties": map[string]any{"type": "string", "pattern": "^[1-9][0-9]*$"},
							"description": "For bookmark/focus mutations, pass the edit_revision from the content snapshot you used, keyed bookmark:<database id> or focus_tree:<database id>. Focus nodes use their parent's tree_edit_revision. Stale versions conflict; never read a newer version merely to replay old content.",
						}
						tool.InputSchema = schema
					}
				}
			}
			return result, err
		}
	}
}

func canonicalRevision(raw string) bool {
	if raw == "" || len(raw) > 19 || raw[0] < '1' || raw[0] > '9' {
		return false
	}
	for i := 1; i < len(raw); i++ {
		if raw[i] < '0' || raw[i] > '9' {
			return false
		}
	}
	_, err := strconv.ParseInt(raw, 10, 64)
	return err == nil
}
