package toolapi

import (
	"context"
	"fmt"
)

type createEventNamespaceIn struct {
	ProjectID  int64  `json:"project_id"`
	Namespace  string `json:"namespace" jsonschema:"the namespace.N bucket events live under, e.g. usa_politics"`
	CountryTag string `json:"country_tag,omitempty" jsonschema:"defaults to the project's primary country"`
	ScopeKind  string `json:"scope_kind,omitempty" jsonschema:"default own"`
}

type createEventNamespaceOut struct {
	NamespaceID int64  `json:"namespace_id"`
	Namespace   string `json:"namespace"`
	CountryTag  string `json:"country_tag"`
}

func registerCreateEventNamespace(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_event_namespace",

		Description: "Create only the NAMING PREFIX that event ids are grouped under (e.g. " +
			"'usa_politics', giving ids like usa_politics.1). Produces NO player-visible content " +
			"whatsoever — it is a bucket, not an event. To create an actual event use create_event; " +
			"for a full-screen cinematic use create_super_event. Must exist before create_event. " +
			"Duplicate → 409 (no auto-suffix).",
	}, func(ctx context.Context, _ *CallToolRequest, in createEventNamespaceIn) (*CallToolResult, createEventNamespaceOut, error) {
		scopeKind := in.ScopeKind
		if scopeKind == "" {
			scopeKind = "own"
		}
		body := map[string]any{
			"namespace":   in.Namespace,
			"country_tag": in.CountryTag,
			"scope_kind":  scopeKind,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/events/namespaces", in.ProjectID), body)
		if err != nil {
			return nil, createEventNamespaceOut{}, err
		}

		out := createEventNamespaceOut{
			Namespace:  stringField(resp, "namespace"),
			CountryTag: stringField(resp, "country_tag"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.NamespaceID = int64(id)
		}
		return nil, out, nil
	})
}

type createEventIn struct {
	ProjectID         int64            `json:"project_id"`
	NamespaceID       int64            `json:"namespace_id" jsonschema:"from create_event_namespace"`
	EventID           string           `json:"event_id" jsonschema:"'namespace.N' — N MUST equal event_number"`
	EventNumber       int              `json:"event_number"`
	EventType         string           `json:"event_type,omitempty" jsonschema:"default country_event"`
	Title             string           `json:"title,omitempty"`
	Description       string           `json:"description,omitempty"`
	IsTriggeredOnly   *bool            `json:"is_triggered_only,omitempty" jsonschema:"default true; exclusive with mean_time_to_happen"`
	MeanTimeToHappen  string           `json:"mean_time_to_happen,omitempty"`
	Trigger           string           `json:"trigger,omitempty"`
	Immediate         string           `json:"immediate,omitempty"`
	Options           []map[string]any `json:"options,omitempty"`
	CountryTag        string           `json:"country_tag,omitempty"`
	AutoFireOnStartup bool             `json:"auto_fire_on_startup,omitempty"`
}

type createEventOut struct {
	EventDBID   int64    `json:"event_db_id"`
	EventID     string   `json:"event_id"`
	EventNumber int      `json:"event_number"`
	Warnings    []string `json:"warnings"`
}

func registerCreateEvent(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_event",

		Description: "Create an ordinary EVENT: the popup dialog with a title, description and " +
			"option buttons. NOT a full-screen cinematic (use create_super_event) and NOT an id " +
			"prefix (use create_event_namespace, which must exist first). event_id = 'namespace.N'; " +
			"event_number MUST equal the numeric suffix. is_triggered_only is exclusive with " +
			"mean_time_to_happen. event_id collisions auto-suffix — read back the returned event_id.",
	}, func(ctx context.Context, _ *CallToolRequest, in createEventIn) (*CallToolResult, createEventOut, error) {
		eventType := in.EventType
		if eventType == "" {
			eventType = "country_event"
		}
		options := in.Options
		if options == nil {
			options = []map[string]any{}
		}
		body := map[string]any{
			"namespace_id":         in.NamespaceID,
			"event_id":             in.EventID,
			"event_number":         in.EventNumber,
			"event_type":           eventType,
			"title":                in.Title,
			"description":          in.Description,
			"is_triggered_only":    boolOr(in.IsTriggeredOnly, true),
			"mean_time_to_happen":  in.MeanTimeToHappen,
			"trigger":              in.Trigger,
			"immediate":            in.Immediate,
			"options":              options,
			"country_tag":          in.CountryTag,
			"auto_fire_on_startup": in.AutoFireOnStartup,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/events", in.ProjectID), body)
		if err != nil {
			return nil, createEventOut{}, err
		}

		out := createEventOut{
			EventID:  stringField(resp, "event_id"),
			Warnings: eventStringSliceField(resp, "warnings"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.EventDBID = int64(id)
		}
		if num, ok := resp["event_number"].(float64); ok {
			out.EventNumber = int(num)
		}
		return nil, out, nil
	})
}

type updateEventIn struct {
	ProjectID        int64            `json:"project_id"`
	EventDBID        int64            `json:"event_db_id"`
	Title            *string          `json:"title,omitempty"`
	Description      *string          `json:"description,omitempty"`
	Trigger          *string          `json:"trigger,omitempty"`
	Immediate        *string          `json:"immediate,omitempty"`
	After            *string          `json:"after,omitempty"`
	Options          []map[string]any `json:"options,omitempty" jsonschema:"REPLACES the whole list — use add_event_option to append one"`
	IsTriggeredOnly  *bool            `json:"is_triggered_only,omitempty"`
	MeanTimeToHappen *string          `json:"mean_time_to_happen,omitempty"`
	EventID          *string          `json:"event_id,omitempty"`
	NamespaceID      *int64           `json:"namespace_id,omitempty"`
	CountryTag       *string          `json:"country_tag,omitempty"`
}

type updateEventOut struct {
	EventDBID int64  `json:"event_db_id"`
	EventID   string `json:"event_id"`
}

func registerUpdateEvent(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_event",
		Description: "Patch an event by numeric event_db_id. Only supplied fields change. options " +
			"REPLACES the whole list (use add_event_option to append one).",
	}, func(ctx context.Context, _ *CallToolRequest, in updateEventIn) (*CallToolResult, updateEventOut, error) {
		body := map[string]any{}
		if in.Title != nil {
			body["title"] = *in.Title
		}
		if in.Description != nil {
			body["description"] = *in.Description
		}
		if in.Trigger != nil {
			body["trigger"] = *in.Trigger
		}
		if in.Immediate != nil {
			body["immediate"] = *in.Immediate
		}
		if in.After != nil {
			body["after"] = *in.After
		}
		if in.Options != nil {
			body["options"] = in.Options
		}
		if in.IsTriggeredOnly != nil {
			body["is_triggered_only"] = *in.IsTriggeredOnly
		}
		if in.MeanTimeToHappen != nil {
			body["mean_time_to_happen"] = *in.MeanTimeToHappen
		}
		if in.EventID != nil {
			body["event_id"] = *in.EventID
		}
		if in.NamespaceID != nil {
			body["namespace_id"] = *in.NamespaceID
		}
		if in.CountryTag != nil {
			body["country_tag"] = *in.CountryTag
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/events/%d", in.ProjectID, in.EventDBID), body)
		if err != nil {
			return nil, updateEventOut{}, err
		}

		out := updateEventOut{
			EventID: stringField(resp, "event_id"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.EventDBID = int64(id)
		}
		return nil, out, nil
	})
}

type addEventOptionIn struct {
	ProjectID     int64          `json:"project_id"`
	EventDBID     int64          `json:"event_db_id"`
	Name          string         `json:"name"`
	Effects       string         `json:"effects,omitempty"`
	Trigger       string         `json:"trigger,omitempty"`
	HiddenEffects string         `json:"hidden_effects,omitempty"`
	AIChance      map[string]any `json:"ai_chance,omitempty"`
}

type addEventOptionOut struct {
	EventDBID   int64  `json:"event_db_id"`
	EventID     string `json:"event_id"`
	OptionCount int    `json:"option_count"`
}

func registerAddEventOption(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "add_event_option",
		Description: "Append one option (answer button) to an event (read-modify-write over update).",
	}, func(ctx context.Context, _ *CallToolRequest, in addEventOptionIn) (*CallToolResult, addEventOptionOut, error) {
		getPath := fmt.Sprintf("/api/projects/%d/events/%d", in.ProjectID, in.EventDBID)
		current, err := proxyAuthedCall(ctx, proxy, "GET", getPath, nil)
		if err != nil {
			return nil, addEventOptionOut{}, err
		}

		var opts []any
		if existing, ok := current["options"].([]any); ok {
			opts = existing
		}
		aiChance := in.AIChance
		if aiChance == nil {
			aiChance = map[string]any{}
		}
		opts = append(opts, map[string]any{
			"name":           in.Name,
			"trigger":        in.Trigger,
			"effects":        in.Effects,
			"hidden_effects": in.HiddenEffects,
			"ai_chance":      aiChance,
		})

		resp, err := proxyAuthedCall(ctx, proxy, "PUT", getPath, map[string]any{"options": opts})
		if err != nil {
			return nil, addEventOptionOut{}, err
		}

		out := addEventOptionOut{
			EventID: stringField(resp, "event_id"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.EventDBID = int64(id)
		}
		if opts, ok := resp["options"].([]any); ok {
			out.OptionCount = len(opts)
		}
		return nil, out, nil
	})
}

func eventStringSliceField(m map[string]any, key string) []string {
	raw, ok := m[key].([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
