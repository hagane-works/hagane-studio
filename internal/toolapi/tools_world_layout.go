package toolapi

import (
	"context"
	"fmt"
)

func worldFields(value map[string]any, fields ...string) map[string]any {
	out := make(map[string]any, len(fields))
	for _, field := range fields {
		if item, ok := value[field]; ok {
			out[field] = item
		}
	}
	return out
}

func worldRows(value any) []any {
	switch rows := value.(type) {
	case []any:
		return rows
	case []map[string]any:
		out := make([]any, len(rows))
		for i, row := range rows {
			out[i] = row
		}
		return out
	}
	return nil
}

func canonicalWorldLayout(value any) []any {
	rows := worldRows(value)
	if rows == nil {
		return nil
	}
	out := make([]any, len(rows))
	for i, value := range rows {
		row, ok := value.(map[string]any)
		if !ok {

			out[i] = value
			continue
		}
		out[i] = worldFields(row, "state_id", "owner", "capital_province", "image_position", "map_position")
		if position, hasImagePosition := row["image_position"]; hasImagePosition && position != nil {
			delete(out[i].(map[string]any), "map_position")
		}
	}
	return out
}

func worldUnsupportedFields(value map[string]any, supported ...string) map[string]int {
	known := make(map[string]bool, len(supported))
	for _, field := range supported {
		known[field] = true
	}
	out := map[string]int{}
	for field := range value {
		if !known[field] {
			out[field]++
		}
	}
	return out
}

func worldLayoutUnsupported(value any) map[string]int {
	out := map[string]int{}
	for _, value := range worldRows(value) {
		row, ok := value.(map[string]any)
		if !ok {
			out["invalid_row"]++
			continue
		}
		for field, count := range worldUnsupportedFields(row, "state_id", "owner", "capital_province", "image_position", "map_position", "province_count") {
			out[field] += count
		}
	}
	return out
}

func compactWorldAddition(addition map[string]any) map[string]any {
	out := worldFields(addition, "id", "kind", "name", "owner", "seed", "options", "placement", "region_id", "created_at")
	for _, field := range []string{"provinces", "state_ids", "raw_paths", "text_keys"} {
		if rows := worldRows(addition[field]); rows != nil {
			out[field+"_count"] = len(rows)
		}
	}
	if fields := worldUnsupportedFields(addition, "id", "kind", "name", "owner", "seed", "options", "placement", "region_id", "created_at", "provinces", "state_ids", "raw_paths", "text_keys", "summary"); len(fields) > 0 {
		out["unsupported_fields"] = fields
	}
	return out
}

func compactWorldResult(resp map[string]any) map[string]any {
	out := worldFields(resp, "summary", "state_layout", "name_status", "addition", "findings", "previews", "preview", "preview_urls", "map_changes")
	if fields := worldUnsupportedFields(resp, "summary", "state_layout", "name_status", "addition", "findings", "previews", "preview", "preview_urls", "map_changes"); len(fields) > 0 {
		out["unsupported_fields"] = fields
	}
	if summary, ok := resp["summary"].(map[string]any); ok {
		copy := make(map[string]any, len(summary))
		for key, value := range summary {
			if key != "state_layout" && key != "name_status" {
				copy[key] = value
			}
		}
		out["summary"] = copy
		for _, key := range []string{"state_layout", "name_status"} {
			if _, exists := out[key]; !exists {
				if value, exists := summary[key]; exists {
					out[key] = value
				}
			}
		}
	}
	if layout := canonicalWorldLayout(out["state_layout"]); layout != nil {
		if fields := worldLayoutUnsupported(out["state_layout"]); len(fields) > 0 {
			out["state_layout_unsupported_fields"] = fields
		}
		out["state_layout"] = layout
		out["state_layout_count"] = len(layout)
		out["state_layout_complete"] = true
	}
	if addition, ok := resp["addition"].(map[string]any); ok {
		out["addition"] = compactWorldAddition(addition)
		out["state_layout_read_tool"] = "get_world_state_layout"
	}
	return out
}

type worldStateLayoutIn struct {
	ProjectID int64 `json:"project_id"`
	Offset    int   `json:"offset,omitempty" jsonschema:"zero-based index in the saved world's canonical state_layout; start at 0"`
	Limit     int   `json:"limit,omitempty" jsonschema:"states per page, 1..200 (default 100); follow next_offset until complete is true"`
}

func registerWorldStateLayoutTool(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "get_world_state_layout",
		Description: "Read the saved generated world's canonical state naming index without regenerating or changing geometry. " +
			"Each row retains state_id, owner, capital_province and source image/map position. Use after an interrupted " +
			"create_world_from_images or an oversized result file; start offset 0 and follow next_offset until complete=true. " +
			"Collect exactly total states, then call name_world_map once with the whole batch in every project export language. " +
			"This reads the saved geometry's index, not the current name completion status.",
	}, func(ctx context.Context, _ *CallToolRequest, in worldStateLayoutIn) (*CallToolResult, map[string]any, error) {
		if in.ProjectID <= 0 || in.Offset < 0 {
			return nil, nil, fmt.Errorf("project_id must be positive and offset must be 0 or greater")
		}
		if in.Limit == 0 {
			in.Limit = 100
		}
		if in.Limit < 1 || in.Limit > 200 {
			return nil, nil, fmt.Errorf("limit must be 1..200; follow next_offset to read every state")
		}
		resp, err := proxyAuthedCall(ctx, proxy, "GET", worldImagePath(in.ProjectID, "land"), nil)
		if err != nil {
			return nil, nil, err
		}
		var addition map[string]any
		for _, value := range worldRows(resp["additions"]) {
			if row, ok := value.(map[string]any); ok && row["kind"] == "world" {
				addition = row
			}
		}
		if addition == nil {
			return nil, nil, fmt.Errorf("this project has no saved generated world; finish preview and obtain the user's approval before create_world_from_images")
		}
		summary, _ := addition["summary"].(map[string]any)
		layout := canonicalWorldLayout(summary["state_layout"])
		if len(layout) == 0 {
			return nil, nil, fmt.Errorf("the saved world has no canonical state_layout; do not regenerate or guess state IDs")
		}
		if in.Offset >= len(layout) {
			return nil, nil, fmt.Errorf("offset %d is outside the saved world's %d states; use the prior page's next_offset", in.Offset, len(layout))
		}
		end := min(len(layout), in.Offset+in.Limit)
		out := map[string]any{"addition": compactWorldAddition(addition), "state_layout": layout[in.Offset:end],
			"total": len(layout), "offset": in.Offset, "returned": end - in.Offset, "complete": end == len(layout)}
		if end < len(layout) {
			out["next_offset"] = end
		}
		if fields := worldLayoutUnsupported(summary["state_layout"]); len(fields) > 0 {
			out["state_layout_unsupported_fields"] = fields
		}
		call, out := worldImageResult(out)
		return call, out, nil
	})
}
