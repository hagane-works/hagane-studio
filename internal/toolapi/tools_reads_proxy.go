package toolapi

import (
	"context"
	"fmt"
)

type getProjectOverviewIn struct {
	ProjectID int64 `json:"project_id"`
}

type overviewCountry struct {
	Tag  string `json:"tag"`
	Name string `json:"name"`
}

type overviewFocusTree struct {
	ID         int64  `json:"id"`
	TreeID     string `json:"tree_id"`
	CountryTag string `json:"country_tag"`
}

type getProjectOverviewOut struct {
	ProjectID       int64               `json:"project_id"`
	Name            string              `json:"name"`
	StartDate       *string             `json:"start_date"`
	EndDate         *string             `json:"end_date"`
	ExportLanguages []string            `json:"export_languages"`
	Dependencies    []string            `json:"dependencies"`
	Countries       []overviewCountry   `json:"countries"`
	FocusTrees      []overviewFocusTree `json:"focus_trees"`
}

func registerGetProjectOverview(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "get_project_overview",
		Description: "Read project dates, export languages, dependencies, countries, and focus trees so you can act without duplicating or missing timeline/localisation settings.",
	}, func(ctx context.Context, _ *CallToolRequest, in getProjectOverviewIn) (*CallToolResult, getProjectOverviewOut, error) {
		out, err := fetchProjectOverview(ctx, proxy, in.ProjectID)
		return nil, out, err
	})
}

func fetchProjectOverview(ctx context.Context, proxy *RESTProxy, projectID int64) (getProjectOverviewOut, error) {
	project, err := proxyAuthedCall(ctx, proxy, "GET", fmt.Sprintf("/api/projects/%d", projectID), nil)
	if err != nil {
		return getProjectOverviewOut{}, err
	}
	countriesRaw, err := proxyAuthedCallList(ctx, proxy, "GET", fmt.Sprintf("/api/projects/%d/countries", projectID), nil)
	if err != nil {
		return getProjectOverviewOut{}, err
	}
	treesRaw, err := proxyAuthedCallList(ctx, proxy, "GET", fmt.Sprintf("/api/projects/%d/focus-trees", projectID), nil)
	if err != nil {
		return getProjectOverviewOut{}, err
	}

	out := getProjectOverviewOut{
		ProjectID: projectID, Name: stringField(project, "name"),
		ExportLanguages: stringSliceField(project, "export_languages"),
		Dependencies:    stringSliceField(project, "dependencies"),
	}
	if sd, ok := project["start_date"].(string); ok {
		out.StartDate = &sd
	}
	if ed, ok := project["end_date"].(string); ok {
		out.EndDate = &ed
	}
	for _, c := range countriesRaw {
		out.Countries = append(out.Countries, overviewCountry{Tag: stringField(c, "tag"), Name: stringField(c, "name")})
	}
	for _, t := range treesRaw {
		out.FocusTrees = append(out.FocusTrees, overviewFocusTree{
			ID: int64Field(t, "id"), TreeID: stringField(t, "tree_id"), CountryTag: stringField(t, "country_tag"),
		})
	}
	return out, nil
}
