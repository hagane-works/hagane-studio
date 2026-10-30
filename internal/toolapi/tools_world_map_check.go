package toolapi

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type checkWorldMapIn struct {
	ProjectID int64  `json:"project_id"`
	Date      string `json:"date,omitempty" jsonschema:"scenario start date the owners are read at, e.g. 1936.1.1 (default)"`
}

type repairWorldMapIn struct {
	ProjectID int64    `json:"project_id"`
	Date      string   `json:"date,omitempty" jsonschema:"scenario start date the owners are read at, e.g. 1936.1.1 (default)"`
	Languages []string `json:"languages,omitempty" jsonschema:"content languages for the names of states the repair makes (default en)"`
}

func registerWorldMapCheckTools(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "check_world_map",
		Description: "Check the project's map for what makes Hearts of Iron IV crash: a state left with only lakes (no land for " +
			"its air base, so the game quits when a game starts), a state across strategic regions, a sea region in pieces, " +
			"a land province in no state or region, a gap in state ids. These crashes leave error.log empty on a normal start. " +
			"Returns ok, the findings, and the states named (states_without_land, states_across_regions). Call it before " +
			"export_project whenever the map was edited, and first when the player says the game crashes; when ok is false, " +
			"tell the player what is wrong and call repair_world_map.",
	}, func(ctx context.Context, _ *CallToolRequest, in checkWorldMapIn) (*CallToolResult, map[string]any, error) {
		if in.ProjectID <= 0 {
			return nil, nil, fmt.Errorf("project_id must be greater than 0")
		}
		path := worldImagePath(in.ProjectID, "check")
		if date := strings.TrimSpace(in.Date); date != "" {
			path += "?" + url.Values{"date": {date}}.Encode()
		}
		response, err := proxyAuthedCall(ctx, proxy.withTimeout(worldImageStepTimeout), "GET", path, nil)
		if err != nil {
			return nil, nil, err
		}
		return nil, response, nil
	})

	AddTool(s, &Tool{
		Name: "repair_world_map",
		Description: "Mend what check_world_map found on the project's map, as the map studio's Fix button does: a state of " +
			"lakes alone takes the land province beside its lake (in the same strategic region) and that land's owner; a " +
			"state across strategic regions is split along them; a sea region in pieces is joined to the one beside it. " +
			"Each repair is one record the player can undo in the map studio. Call check_world_map again afterwards, " +
			"then export. Tell the player which states it changed.",
	}, func(ctx context.Context, _ *CallToolRequest, in repairWorldMapIn) (*CallToolResult, map[string]any, error) {
		if in.ProjectID <= 0 {
			return nil, nil, fmt.Errorf("project_id must be greater than 0")
		}
		body := map[string]any{}
		if date := strings.TrimSpace(in.Date); date != "" {
			body["date"] = date
		}
		if len(in.Languages) > 0 {
			body["languages"] = in.Languages
		}
		response, err := proxyAuthedCall(ctx, proxy.withTimeout(worldImageStepTimeout), "POST", worldImagePath(in.ProjectID, "repair"), body)
		if err != nil {
			return nil, nil, err
		}
		return nil, response, nil
	})
}
