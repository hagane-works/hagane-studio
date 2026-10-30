package toolapi

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type getCharactersIn struct {
	ProjectID  int64  `json:"project_id"`
	CountryTag string `json:"country_tag,omitempty" jsonschema:"optional filter, case-insensitive"`
}

type characterSummary struct {
	CharacterID    int64  `json:"character_id"`
	Tag            string `json:"tag"`
	Name           string `json:"name"`
	CountryTag     string `json:"country_tag"`
	RoleType       string `json:"role_type"`
	HasAdvisorRole bool   `json:"has_advisor_role"`
}

type getCharactersOut struct {
	ProjectID  int64              `json:"project_id"`
	Characters []characterSummary `json:"characters"`
}

func registerGetCharacters(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "get_characters",
		Description: "Read existing characters (id, tag, name, role) so you don't duplicate.",
	}, func(ctx context.Context, _ *CallToolRequest, in getCharactersIn) (*CallToolResult, getCharactersOut, error) {
		out, err := fetchCharactersPayload(ctx, proxy, in.ProjectID, in.CountryTag)
		return nil, out, err
	})
}

func fetchCharactersPayload(ctx context.Context, proxy *RESTProxy, projectID int64, countryTag string) (getCharactersOut, error) {
	path := fmt.Sprintf("/api/projects/%d/characters", projectID)
	if tag := strings.TrimSpace(countryTag); tag != "" {
		path += "?" + url.Values{"country_tag": {strings.ToUpper(tag)}}.Encode()
	}

	rows, err := proxyAuthedCallList(ctx, proxy, "GET", path, nil)
	if err != nil {
		return getCharactersOut{}, err
	}

	out := getCharactersOut{ProjectID: projectID, Characters: []characterSummary{}}
	for _, c := range rows {
		out.Characters = append(out.Characters, characterSummary{

			CharacterID:    int64Field(c, "id"),
			Tag:            stringField(c, "tag"),
			Name:           stringField(c, "name"),
			CountryTag:     stringField(c, "country_tag"),
			RoleType:       stringField(c, "role_type"),
			HasAdvisorRole: charBoolField(c, "has_advisor_role"),
		})
	}
	return out, nil
}

func charBoolField(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
