package toolapi

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type getFocusTreeIn struct {
	ProjectID  int64  `json:"project_id"`
	CountryTag string `json:"country_tag,omitempty" jsonschema:"optional filter, case-insensitive"`
}

type focusTreeNode struct {
	TreeEditRevision   string            `json:"tree_edit_revision,omitempty"`
	Diagnostics        *[]saveDiagnostic `json:"diagnostics,omitempty"`
	NodeID             int64             `json:"node_id"`
	FocusID            string            `json:"focus_id"`
	Name               string            `json:"name"`
	Icon               string            `json:"icon"`
	Cost               float64           `json:"cost"`
	X                  int64             `json:"x"`
	Y                  int64             `json:"y"`
	RelativePositionID string            `json:"relative_position_id,omitempty"`
	Prerequisites      [][]string        `json:"prerequisites"`
	PrerequisiteExpr   string            `json:"prerequisite_expression"`
	MutuallyExclusive  []string          `json:"mutually_exclusive"`
	CompletionReward   string            `json:"completion_reward"`
	Available          string            `json:"available"`
	Bypass             string            `json:"bypass"`
	Cancel             string            `json:"cancel"`
	SelectEffect       string            `json:"select_effect"`
	AIWillDo           string            `json:"ai_will_do"`
	NameEn             string            `json:"name_en"`
	NameZh             string            `json:"name_zh"`
	DescEn             string            `json:"desc_en"`
	DescZh             string            `json:"desc_zh"`
	DisplayNameLocs    map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs    map[string]string `json:"description_locs,omitempty"`
	EffectTooltipLocs  map[string]string `json:"effect_tooltip_locs,omitempty"`
}

type focusTreeEntry struct {
	EditRevision string            `json:"edit_revision,omitempty"`
	Diagnostics  *[]saveDiagnostic `json:"diagnostics,omitempty"`
	FocusTreeID  int64             `json:"focus_tree_id"`
	TreeID       string            `json:"tree_id"`
	CountryTag   string            `json:"country_tag"`
	Nodes        []focusTreeNode   `json:"nodes"`
}

type getFocusTreeOut struct {
	ProjectID  int64            `json:"project_id"`
	FocusTrees []focusTreeEntry `json:"focus_trees"`
}

func registerGetFocusTree(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "get_focus_tree",
		Description: "Read focus trees + their complete authoring state (ids, name, icon, cost, x/y, " +
			"prerequisites, mutual exclusions, completion_reward and trigger/effect fields). Use this after writes: " +
			"final reports must quote these returned values rather than the intended inputs. focus_id strings are " +
			"what connect_focus references.",
	}, func(ctx context.Context, _ *CallToolRequest, in getFocusTreeIn) (*CallToolResult, getFocusTreeOut, error) {
		out, err := fetchFocusTreePayload(ctx, proxy, in.ProjectID, in.CountryTag)
		return nil, out, err
	})
}

func fetchFocusTreePayload(ctx context.Context, proxy *RESTProxy, projectID int64, countryTag string) (getFocusTreeOut, error) {
	path := fmt.Sprintf("/api/projects/%d/focus-trees", projectID)
	if tag := strings.TrimSpace(countryTag); tag != "" {
		path += "?" + url.Values{"country_tag": {strings.ToUpper(tag)}}.Encode()
	}

	trees, err := proxyAuthedCallList(ctx, proxy, "GET", path, nil)
	if err != nil {
		return getFocusTreeOut{}, err
	}

	out := getFocusTreeOut{ProjectID: projectID, FocusTrees: []focusTreeEntry{}}
	for _, t := range trees {
		treeID := int64Field(t, "id")

		nodesRaw := entityRows(t["nodes"])

		entry := focusTreeEntry{
			EditRevision: stringField(t, "edit_revision"),
			Diagnostics:  saveDiagnosticsField(t),
			FocusTreeID:  treeID,
			TreeID:       stringField(t, "tree_id"),
			CountryTag:   stringField(t, "country_tag"),
			Nodes:        []focusTreeNode{},
		}
		for _, n := range nodesRaw {
			prerequisites := focusStringMatrixField(n, "prerequisites")
			entry.Nodes = append(entry.Nodes, focusTreeNode{
				TreeEditRevision: stringField(n, "tree_edit_revision"),
				Diagnostics:      saveDiagnosticsField(n),

				NodeID:             int64Field(n, "id"),
				FocusID:            stringField(n, "focus_id"),
				Name:               stringField(n, "name"),
				Icon:               stringField(n, "icon"),
				Cost:               focusFloat64Field(n, "cost"),
				X:                  int64Field(n, "x"),
				Y:                  int64Field(n, "y"),
				RelativePositionID: stringField(n, "relative_position_id"),
				Prerequisites:      prerequisites,
				PrerequisiteExpr:   focusPrerequisiteExpression(prerequisites),
				MutuallyExclusive:  focusStringSliceField(n, "mutually_exclusive"),
				CompletionReward:   stringField(n, "completion_reward"),
				Available:          stringField(n, "available"),
				Bypass:             stringField(n, "bypass"),
				Cancel:             stringField(n, "cancel"),
				SelectEffect:       stringField(n, "select_effect"),
				AIWillDo:           stringField(n, "ai_will_do"),
				NameEn:             stringField(n, "name_en"),
				NameZh:             stringField(n, "name_zh"),
				DescEn:             stringField(n, "desc_en"),
				DescZh:             stringField(n, "desc_zh"),
				DisplayNameLocs:    nonNilStringMap(stringMapField(n, "display_name_locs")),
				DescriptionLocs:    nonNilStringMap(stringMapField(n, "description_locs")),
				EffectTooltipLocs:  nonNilStringMap(stringMapField(n, "effect_tooltip_locs")),
			})
		}
		out.FocusTrees = append(out.FocusTrees, entry)
	}
	return out, nil
}

func nonNilStringMap(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}

func focusFloat64Field(m map[string]any, key string) float64 {
	value, _ := m[key].(float64)
	return value
}
