package toolapi

import (
	"context"
	"fmt"
	"net/url"
)

type worldMapRecordsIn struct {
	ProjectID int64 `json:"project_id"`
}

func registerWorldMapRecordTools(s *Server, proxy *RESTProxy) {
	records := func(ctx context.Context, projectID int64) (map[string]any, error) {
		response, err := proxyAuthedCall(ctx, proxy, "GET", worldImagePath(projectID, "land"), nil)
		if err != nil {
			return nil, err
		}
		changes, _ := response["map_changes"].(map[string]any)
		return changes, nil
	}

	AddTool(s, &Tool{
		Name: "list_world_map_records",
		Description: "List the records that make the project's map not the base game's, newest last: new land, province edits, " +
			"a new world or an imported MOD map (id, kind, name, created_at), with their count. No records means the map is the " +
			"base game's. The game view's map studio shows the player the same list with an undo and a restore button.",
	}, func(ctx context.Context, _ *CallToolRequest, in worldMapRecordsIn) (*CallToolResult, map[string]any, error) {
		if err := requireRecordsProject(ctx, in.ProjectID); err != nil {
			return nil, nil, err
		}
		changes, err := records(ctx, in.ProjectID)
		if err != nil {
			return nil, nil, err
		}
		out := map[string]any{"count": 0, "records": []any{}}
		if changes != nil {
			out["count"], out["records"] = changes["count"], changes["records"]
		}
		return nil, out, nil
	})

	AddTool(s, &Tool{
		Name: "undo_world_map_change",
		Description: "Undo the project's latest map record only (the newest new land, province edit, new world or imported MOD " +
			"map); the ones before it stay. Call it only when the player has asked you, in this conversation, to undo the latest " +
			"map change — their request is the confirmation, ask nothing more; never on your own initiative. Returns what was " +
			"undone: tell the player its name.",
	}, func(ctx context.Context, _ *CallToolRequest, in worldMapRecordsIn) (*CallToolResult, map[string]any, error) {
		if err := requireRecordsProject(ctx, in.ProjectID); err != nil {
			return nil, nil, err
		}
		response, err := proxyAuthedCall(ctx, proxy, "DELETE", worldImagePath(in.ProjectID, "land")+"?latest=1", nil)
		if err != nil {
			return nil, nil, err
		}
		return nil, response, nil
	})

	AddTool(s, &Tool{
		Name: "restore_world_map",
		Description: "Restore the base game's map: take away every map record (new land, province edits, new worlds, imported MOD " +
			"maps), so the map, its states and strategic regions are the base game's again; countries and other content stay. " +
			"Call it only when the player has asked you, in this conversation, to restore the map (恢复原版地图, undo all the map " +
			"changes, start the map over) — their request is the confirmation, ask nothing more; never on your own initiative, " +
			"not even to save a new world (then list the records and let the player decide). Returns the records it took away " +
			"(removed): tell the player what they were.",
	}, func(ctx context.Context, _ *CallToolRequest, in worldMapRecordsIn) (*CallToolResult, map[string]any, error) {
		if err := requireRecordsProject(ctx, in.ProjectID); err != nil {
			return nil, nil, err
		}
		changes, err := records(ctx, in.ProjectID)
		if err != nil {
			return nil, nil, err
		}
		confirmation, _ := changes["confirmation"].(string)
		if changes == nil || confirmation == "" {
			return nil, map[string]any{"restored": 0, "removed": []any{}, "already_base": true}, nil
		}

		response, err := proxyAuthedCall(ctx, proxy, "DELETE", worldImagePath(in.ProjectID, "land")+"?"+url.Values{"confirm": {confirmation}}.Encode(), nil)
		if err != nil {
			return nil, nil, err
		}
		return nil, response, nil
	})
}

func requireRecordsProject(_ context.Context, projectID int64) error {
	if projectID <= 0 {
		return fmt.Errorf("project_id must be greater than 0")
	}
	return nil
}
