package toolapi

import (
	"context"
)

type listProjectsIn struct{}

type projectOut struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type listProjectsOut struct {
	Total    int          `json:"total"`
	Projects []projectOut `json:"projects"`
}

func registerListProjects(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "list_projects",
		Description: "List your HOI4 mod projects (id, name).",
	}, func(ctx context.Context, _ *CallToolRequest, _ listProjectsIn) (*CallToolResult, listProjectsOut, error) {
		resp, err := proxyAuthedCall(ctx, proxy, "GET", "/api/projects", nil)
		if err != nil {
			return nil, listProjectsOut{}, err
		}
		rows, _ := resp["projects"].([]any)
		out := listProjectsOut{Total: len(rows), Projects: make([]projectOut, 0, len(rows))}
		for _, raw := range rows {
			row, _ := raw.(map[string]any)
			out.Projects = append(out.Projects, projectOut{ID: int64Field(row, "id"), Name: stringField(row, "name")})
		}
		return nil, out, nil
	})
}
