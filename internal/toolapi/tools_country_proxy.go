package toolapi

import (
	"context"
	"fmt"
)

type createCountryIn struct {
	ProjectID int64  `json:"project_id"`
	Tag       string `json:"tag" jsonschema:"2-5 chars, first A-Z, e.g. ITA"`
	Name      string `json:"name"`
	Ideology  string `json:"ideology,omitempty" jsonschema:"default neutrality"`
}

type createCountryOut struct {
	CountryID int64  `json:"country_id"`
	Tag       string `json:"tag"`
	Warning   string `json:"warning,omitempty"`
}

func registerCreateCountry(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "create_country",
		Description: "Create a country. tag = 2-5 chars, first A-Z (e.g. ITA). name = display name.",
	}, func(ctx context.Context, _ *CallToolRequest, in createCountryIn) (*CallToolResult, createCountryOut, error) {
		ideology := in.Ideology
		if ideology == "" {
			ideology = "neutrality"

		}
		body := map[string]any{"tag": in.Tag, "name": in.Name, "ideology": ideology}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/countries", in.ProjectID), body)
		if err != nil {
			return nil, createCountryOut{}, err
		}

		out := createCountryOut{
			Tag: stringField(resp, "tag"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.CountryID = int64(id)
		}
		if w, ok := resp["warning"].(string); ok && w != "" {
			out.Warning = w
		}
		return nil, out, nil
	})
}

func stringField(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}
