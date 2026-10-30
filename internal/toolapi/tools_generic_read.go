package toolapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const defaultEntityListLimit = 200

type EntitySummary struct {
	EditRevision     string `json:"edit_revision,omitempty"`
	TreeEditRevision string `json:"tree_edit_revision,omitempty"`
	ID               int64  `json:"id"`
	Key              string `json:"key"`
	Name             string `json:"name"`
	Kind             string `json:"kind"`

	ResolvedLocale string `json:"resolved_locale"`
	CountryTag     string `json:"country_tag,omitempty"`

	Summary map[string]any `json:"summary,omitempty"`
}

type listEntitiesIn struct {
	ProjectID  int64  `json:"project_id"`
	Kind       string `json:"kind" jsonschema:"content type, e.g. event / focus_tree / character"`
	CountryTag string `json:"country_tag,omitempty" jsonschema:"country filter where supported; for kind=focus this does not replace parent_id"`
	ParentID   int64  `json:"parent_id,omitempty" jsonschema:"REQUIRED for kind=focus: the numeric id from list_entities kind=focus_tree; country_tag alone cannot select a tree"`
	Locale     string `json:"locale,omitempty" jsonschema:"language for name, default en"`
	Limit      int    `json:"limit,omitempty"`
}

type listEntitiesOut struct {
	ProjectID int64           `json:"project_id"`
	Kind      string          `json:"kind"`
	Entities  []EntitySummary `json:"entities"`
	Returned  int             `json:"returned"`
	Total     int             `json:"total"`
	Truncated bool            `json:"truncated"`
	Note      string          `json:"note,omitempty"`
}

func registerListEntities(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "list_entities",
		Description: "List any content in a project — the general read tool. `kind` is one of: " +
			strings.Join(entityKindNames(), ", ") + ". Every result has the same shape: `id` " +
			"(use it to change or delete this thing), `key` (use it to reference this thing from " +
			"other content), `name` (resolved to a readable name), plus a few fields specific to " +
			"the kind. Read before you write: if you change something without looking, you may " +
			"overwrite what the author just did in the program's window. Filter with country_tag where " +
			"supported. For kind=focus, first list_entities kind=focus_tree and pass that tree's numeric id as parent_id; country_tag alone cannot select a tree.",
	}, func(ctx context.Context, _ *CallToolRequest, in listEntitiesIn) (*CallToolResult, listEntitiesOut, error) {
		kind, err := lookupEntityKind(in.Kind)
		if err != nil {
			return nil, listEntitiesOut{}, err
		}
		countryTag := strings.ToUpper(strings.TrimSpace(in.CountryTag))
		if kind.RequireCountry != "" && countryTag == "" {
			return nil, listEntitiesOut{}, fmt.Errorf("%s: %s", kind.Kind, kind.RequireCountry)
		}
		path, err := entityListPath(in.ProjectID, kind, in.ParentID)
		if err != nil {
			return nil, listEntitiesOut{}, err
		}
		if countryTag != "" && kind.CountryFilter {
			if kind.CountryPath != "" {
				path = fmt.Sprintf("/api/projects/%d/%s", in.ProjectID,
					strings.ReplaceAll(kind.CountryPath, "{country_tag}", url.PathEscape(countryTag)))
			} else {
				path += "?" + url.Values{"country_tag": {countryTag}}.Encode()
			}
		}
		rows, err := proxyAuthedCallAny(ctx, proxy, "GET", path, nil)
		if err != nil {
			return nil, listEntitiesOut{}, err
		}

		counts := newChildCounter(ctx, proxy, in.ProjectID)

		limit := in.Limit
		if limit <= 0 || limit > defaultEntityListLimit {
			limit = defaultEntityListLimit
		}
		out := listEntitiesOut{
			ProjectID: in.ProjectID, Kind: kind.Kind, Entities: []EntitySummary{},
			Total: len(rows),
		}
		for _, row := range rows {
			if len(out.Entities) >= limit {
				break
			}
			out.Entities = append(out.Entities, summarize(kind, row, counts))
		}
		out.Returned = len(out.Entities)
		out.Truncated = out.Returned < out.Total
		switch {
		case out.Truncated:

			out.Note = fmt.Sprintf("Showing %d of %d. Narrow with country_tag, or raise limit — "+
				"do not conclude something is absent from this list alone.", out.Returned, out.Total)
		case kind.NoKeyReason != "" && kind.KeyField == "" && kind.Kind != "super_event":
			out.Note = kind.NoKeyReason
		}
		return nil, out, nil
	})
}

type getEntityIn struct {
	ProjectID int64  `json:"project_id"`
	Kind      string `json:"kind"`
	ID        int64  `json:"id" jsonschema:"the row id from list_entities"`
	ParentID  int64  `json:"parent_id,omitempty" jsonschema:"REQUIRED for kind=focus: the numeric focus_tree id owning this node, from list_entities kind=focus_tree"`
	Locale    string `json:"locale,omitempty"`
}

type getEntityOut struct {
	ProjectID int64          `json:"project_id"`
	Kind      string         `json:"kind"`
	Summary   EntitySummary  `json:"summary"`
	Entity    map[string]any `json:"entity"`
}

func registerGetEntity(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "get_entity",
		Description: "Read ONE piece of content in full, by the `id` list_entities gave you. " +
			"Returns the same summary plus every field the project stores, so you can see the " +
			"current values before changing them. `kind` is one of: " +
			strings.Join(entityKindNames(), ", ") + ".",
	}, func(ctx context.Context, _ *CallToolRequest, in getEntityIn) (*CallToolResult, getEntityOut, error) {
		kind, err := lookupEntityKind(in.Kind)
		if err != nil {
			return nil, getEntityOut{}, err
		}
		if in.ID <= 0 {
			return nil, getEntityOut{}, fmt.Errorf(
				"get_entity needs the row id (the `id` field from list_entities), not the game identifier")
		}
		base, err := entityListPath(in.ProjectID, kind, in.ParentID)
		if err != nil {
			return nil, getEntityOut{}, err
		}
		path := fmt.Sprintf("%s/%d", base, in.ID)
		if kind.ReadFromList {
			path = base
		}
		rows, err := proxyAuthedCallAny(ctx, proxy, "GET", path, nil)
		if err != nil {
			return nil, getEntityOut{}, err
		}
		var entity map[string]any
		for _, row := range rows {
			if int64Field(row, "id") == in.ID {
				entity = row
				break
			}
		}
		if entity == nil {
			return nil, getEntityOut{}, fmt.Errorf("no %s with id %d in project %d", kind.Kind, in.ID, in.ProjectID)
		}
		return nil, getEntityOut{
			ProjectID: in.ProjectID, Kind: kind.Kind,
			Summary: summarize(kind, entity, newChildCounter(ctx, proxy, in.ProjectID)), Entity: entity,
		}, nil
	})
}

func lookupEntityKind(name string) (entityKind, error) {
	kind, ok := entityKindByName[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return entityKind{}, fmt.Errorf("unknown kind %q. Valid kinds: %s",
			name, strings.Join(entityKindNames(), ", "))
	}
	return kind, nil
}

func entityListPath(projectID int64, kind entityKind, parentID int64) (string, error) {
	if kind.ParentPath == "" {
		return fmt.Sprintf("/api/projects/%d/%s", projectID, kind.Path), nil
	}
	if parentID <= 0 {
		parentKind := strings.ReplaceAll(strings.TrimSuffix(kind.ParentPath, "s"), "-", "_")
		for _, candidate := range entityKinds {
			if candidate.Path == kind.ParentPath && candidate.ParentPath == "" {
				parentKind = candidate.Kind
				break
			}
		}
		return "", fmt.Errorf("%s lives inside a %s — pass parent_id (list_entities kind=%q gives the ids)",
			kind.Kind, parentKind, parentKind)
	}
	return fmt.Sprintf("/api/projects/%d/%s/%d/%s", projectID, kind.ParentPath, parentID, kind.Path), nil
}

func summarize(kind entityKind, row map[string]any, counts *childCounter) EntitySummary {
	summary := EntitySummary{
		EditRevision:     stringField(row, "edit_revision"),
		TreeEditRevision: stringField(row, "tree_edit_revision"),
		ID:               intField(row, "id"),
		Key:              entityKey(kind, row),
		Name:             displayNameText(row),
		Kind:             kind.Kind,
		ResolvedLocale:   stringField(row, "resolved_locale"),
		CountryTag:       stringField(row, "country_tag"),
	}
	if summary.Name == "" {

		summary.Name = displayNameFallback(row)
	}
	for _, field := range kind.Summary {
		value, ok := summaryValue(row, field)
		if !ok && kind.ChildCount != nil && kind.ChildCount.Field == field {
			value, ok = counts.count(*kind.ChildCount, row)
		}
		if !ok && kind.ParentLookup != nil && kind.ParentLookup.Field == field {
			value, ok = counts.lookup(*kind.ParentLookup, row)
		}
		if !ok {
			continue
		}
		if summary.Summary == nil {
			summary.Summary = map[string]any{}
		}
		summary.Summary[field] = value
	}
	return summary
}

func summaryValue(row map[string]any, field string) (any, bool) {
	switch field {
	case "node_count":

		nodes, ok := row["nodes"].([]any)
		if !ok {
			return nil, false
		}
		return len(nodes), true
	case "configured_fields":

		return configuredHistoryFields(row)
	case "shared_ref_count":

		return countList(row, "shared_focus_refs")
	case "category_count":
		return countList(row, "categories")
	case "sub_specialization_count":
		return countList(row, "sub_specializations")
	case "advanced_extras_count":

		return countList(row, "extra_fields")
	case "has_actionable_warning":
		return hasActionableWarning(row)
	case "pride_of_the_fleet":
		return prideOfTheFleet(row)
	case "size":
		return rawFileSize(row)
	case "references":
		return rawFileReferences(row)
	case "category_key":

		if key, ok := row["category_key"].(string); ok && key != "" {
			return key, true
		}
		return nil, false
	}
	value, ok := row[field]
	if !ok || value == nil {
		return nil, false
	}
	if text, isText := value.(string); isText && strings.TrimSpace(text) == "" {
		return nil, false
	}
	return value, true
}

var historyFileSettings = []string{
	"set_politics", "set_popularities", "set_research_slots", "set_stability",
	"set_war_support", "add_ideas", "remove_ideas", "complete_national_focus", "raw_script",
}

func configuredHistoryFields(row map[string]any) (any, bool) {
	var configured []string
	for _, field := range historyFileSettings {
		switch value := row[field].(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				configured = append(configured, field)
			}
		case []any:
			if len(value) > 0 {
				configured = append(configured, field)
			}
		case map[string]any:
			if len(value) > 0 {
				configured = append(configured, field)
			}
		case float64, bool:
			configured = append(configured, field)
		}
	}
	if configured == nil {
		return nil, false
	}
	return configured, true
}

func entityKey(kind entityKind, row map[string]any) string {
	if kind.Kind == "super_event" {

		tag := strings.ToLower(strings.TrimSpace(stringField(row, "country_tag")))
		number := intField(row, "number")
		if tag == "" || number == 0 {
			return ""
		}
		return fmt.Sprintf("%s_se.%d", tag, number)
	}
	if kind.KeyField == "" {
		return ""
	}
	return stringField(row, kind.KeyField)
}

func proxyAuthedCallAny(ctx context.Context, proxy *RESTProxy, method, path string, body any) ([]map[string]any, error) {
	raw, err := proxy.callRaw(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return []map[string]any{}, nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decoding upstream response: %w", err)
	}
	enrichDisplayNames(decoded, contentLocaleFrom(ctx))
	return entityRows(decoded), nil
}

func entityRows(decoded any) []map[string]any {
	switch typed := decoded.(type) {
	case []any:
		return objectsIn(typed)
	case map[string]any:

		if _, hasID := typed["id"]; hasID {
			return []map[string]any{typed}
		}

		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if list, ok := typed[key].([]any); ok {
				if rows := objectsIn(list); len(rows) > 0 {
					return rows
				}
			}
		}
		return []map[string]any{}
	}
	return []map[string]any{}
}

func objectsIn(list []any) []map[string]any {
	rows := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows
}
