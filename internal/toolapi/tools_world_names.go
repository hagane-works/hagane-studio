package toolapi

import (
	"context"
	"fmt"
)

type worldMapNameInput struct {
	StateID         int               `json:"state_id" jsonschema:"state number returned in create_world_from_images state_layout"`
	DisplayNameLocs map[string]string `json:"display_name_locs,omitempty" jsonschema:"the state's visible name keyed by every project content language; required for every state"`
	CapitalNameLocs map[string]string `json:"capital_name_locs,omitempty" jsonschema:"the capital or victory-point city's visible name keyed by project content language"`
}

type nameWorldMapIn struct {
	ProjectID  int64               `json:"project_id"`
	StateNames []worldMapNameInput `json:"state_names" jsonschema:"all generated states and their capital/victory-point city names in one batch; at least one item is required"`
}

func registerNameWorldMapTool(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "name_world_map",
		Description: "Name a generated world's states and capital/victory-point cities in one complete batch (never one state per call). Call this " +
			"after create_world_from_images, using its state_layout state_id values and including every generated state. " +
			"Every display_name_locs map must contain exactly all project export languages (English is always required). " +
			"Ordinary HOI4 provinces have no independent visible names; use capital_name_locs only for the state's capital " +
			"or victory-point province; omitted capital languages use that state's name, never Town N. Do not report the " +
			"world-building task complete unless this result says name_status.complete=true.",
	}, func(ctx context.Context, _ *CallToolRequest, in nameWorldMapIn) (*CallToolResult, map[string]any, error) {
		if in.ProjectID <= 0 {
			return nil, nil, fmt.Errorf("project_id must be greater than 0")
		}
		if len(in.StateNames) == 0 {
			return nil, nil, fmt.Errorf("state_names must contain at least one item")
		}
		response, err := proxyAuthedCall(ctx, proxy.withTimeout(worldImageStepTimeout), "POST",
			worldImagePath(in.ProjectID, "names"), map[string]any{"state_names": in.StateNames})
		if err != nil {
			return nil, nil, err
		}
		call, out := worldImageResult(response)
		return call, out, nil
	})
}
