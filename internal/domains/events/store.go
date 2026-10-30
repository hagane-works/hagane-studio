package events

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const (
	namespacesTable = "event_namespaces"
	eventsTable     = "events"
)

type optional[T any] = workbench.Optional[T]

var (
	errNamespaceMissing = apiutil.NotFoundError("Namespace not found")
	errEventMissing     = apiutil.NotFoundError("Event not found")
	errSourceMissing    = apiutil.NotFoundError("Source event not found")
	errNamespaceForeign = apiutil.BadRequestError("Namespace not found in this project")
)

func namespaceConflict(name string) error {
	return apiutil.ConflictError(fmt.Sprintf("namespace '%s' already exists in this project. Pick a unique name.", name))
}

func eventIDConflict(eventID string) error {
	return apiutil.ConflictError(fmt.Sprintf("event_id '%s' already exists in this project", eventID))
}

func text(row project.Row, key, def string) string {
	switch v := row[key].(type) {
	case string:
		return v
	case nil:
		return def
	default:
		return fmt.Sprint(v)
	}
}

func flag(row project.Row, key string, def bool) bool {
	if b, ok := row[key].(bool); ok {
		return b
	}
	return def
}

func whole(row project.Row, key string) int { return int(project.Int64(row[key])) }

func stringMap(v any) map[string]string {
	out := map[string]string{}
	if m, ok := v.(map[string]any); ok {
		for k, item := range m {
			if s, ok := item.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

func stringList(v any) []string {
	switch list := v.(type) {
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return append([]string{}, list...)
	}
	return nil
}

func anyList(values []string) any {
	if values == nil {
		return nil
	}
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func anyMap(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}

func strDefault(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

func boolDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func intDefault(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

type namespaceResponse struct {
	ID                int64    `json:"id"`
	ProjectID         int64    `json:"project_id"`
	Namespace         string   `json:"namespace"`
	CountryTag        string   `json:"country_tag"`
	ScopeKind         string   `json:"scope_kind"`
	SharedCountryTags []string `json:"shared_country_tags"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	EventCount        int64    `json:"event_count"`
}

func namespaceFromRow(row project.Row, events []project.Row) namespaceResponse {
	ns := namespaceResponse{
		ID:                project.Int64(row["id"]),
		ProjectID:         project.Int64(row["project_id"]),
		Namespace:         text(row, "namespace", ""),
		CountryTag:        text(row, "country_tag", ""),
		ScopeKind:         text(row, "scope_kind", "own"),
		SharedCountryTags: stringList(row["shared_country_tags"]),
		CreatedAt:         text(row, "created_at", ""),
		UpdatedAt:         text(row, "updated_at", ""),
	}
	for _, ev := range events {
		if project.Int64(ev["namespace_id"]) == ns.ID {
			ns.EventCount++
		}
	}
	return ns
}

func findRow(rows []project.Row, id int64) project.Row {
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			return row
		}
	}
	return nil
}

func listNamespaces(p *project.Project, countryTag string) ([]namespaceResponse, error) {
	rows, err := p.Rows(namespacesTable)
	if err != nil {
		return nil, err
	}
	events, err := p.Rows(eventsTable)
	if err != nil {
		return nil, err
	}
	out := []namespaceResponse{}
	for _, row := range rows {
		if countryTag != "" && row["country_tag"] != countryTag {
			continue
		}
		out = append(out, namespaceFromRow(row, events))
	}
	return out, nil
}

func getNamespace(p *project.Project, nsID int64) (namespaceResponse, error) {
	rows, err := p.Rows(namespacesTable)
	if err != nil {
		return namespaceResponse{}, err
	}
	row := findRow(rows, nsID)
	if row == nil {
		return namespaceResponse{}, errNamespaceMissing
	}
	events, err := p.Rows(eventsTable)
	if err != nil {
		return namespaceResponse{}, err
	}
	return namespaceFromRow(row, events), nil
}

type namespaceCreateInput struct {
	Namespace         *string  `json:"namespace"`
	CountryTag        string   `json:"country_tag"`
	ScopeKind         *string  `json:"scope_kind"`
	SharedCountryTags []string `json:"shared_country_tags"`
}

func createNamespace(p *project.Project, in namespaceCreateInput) (namespaceResponse, error) {
	name := ""
	if in.Namespace != nil {
		name = strings.TrimSpace(*in.Namespace)
	}
	if name == "" {
		return namespaceResponse{}, apiutil.BadRequestError("namespace cannot be empty")
	}
	projectTag := text(p.Record(), "country_tag", "")
	tag := strings.ToUpper(strings.TrimSpace(workbench.FirstNonBlank(in.CountryTag, projectTag)))
	var id int64
	err := p.Write([]string{namespacesTable}, func(tx *project.Tx) error {
		for _, row := range tx.Rows(namespacesTable) {
			if row["namespace"] == name {
				return namespaceConflict(name)
			}
		}
		created := tx.Insert(namespacesTable, project.Row{
			"namespace": name, "country_tag": tag, "scope_kind": strDefault(in.ScopeKind, "own"),
			"shared_country_tags": anyList(in.SharedCountryTags),
		})
		id = project.Int64(created["id"])
		return nil
	})
	if err != nil {
		return namespaceResponse{}, err
	}
	return getNamespace(p, id)
}

type namespacePatch struct {
	Namespace         optional[string]   `json:"namespace"`
	CountryTag        optional[string]   `json:"country_tag"`
	ScopeKind         optional[string]   `json:"scope_kind"`
	SharedCountryTags optional[[]string] `json:"shared_country_tags"`
}

func updateNamespace(p *project.Project, nsID int64, patch namespacePatch) (namespaceResponse, error) {
	changes := project.Row{}
	if patch.Namespace.Set {
		if patch.Namespace.Value == nil {
			return namespaceResponse{}, apiutil.ValidationError("namespace cannot be null")
		}
		changes["namespace"] = strings.TrimSpace(*patch.Namespace.Value)
	}
	if patch.CountryTag.Set {
		if patch.CountryTag.Value == nil {
			changes["country_tag"] = nil
		} else {
			changes["country_tag"] = strings.ToUpper(strings.TrimSpace(*patch.CountryTag.Value))
		}
	}
	if patch.ScopeKind.Set {
		if patch.ScopeKind.Value == nil {
			return namespaceResponse{}, apiutil.ValidationError("scope_kind cannot be null")
		}
		changes["scope_kind"] = *patch.ScopeKind.Value
	}
	if patch.SharedCountryTags.Set {
		if patch.SharedCountryTags.Value == nil {
			changes["shared_country_tags"] = nil
		} else {
			changes["shared_country_tags"] = anyList(*patch.SharedCountryTags.Value)
		}
	}
	err := p.Write([]string{namespacesTable}, func(tx *project.Tx) error {
		if tx.Find(namespacesTable, nsID) == nil {
			return errNamespaceMissing
		}
		if len(changes) == 0 {
			return nil
		}
		if name, ok := changes["namespace"].(string); ok {
			for _, row := range tx.Rows(namespacesTable) {
				if project.Int64(row["id"]) != nsID && row["namespace"] == name {
					return namespaceConflict(name)
				}
			}
		}
		_, err := tx.Update(namespacesTable, nsID, func(row project.Row) {
			for k, v := range changes {
				row[k] = v
			}
		})
		return err
	})
	if err != nil {
		return namespaceResponse{}, err
	}
	return getNamespace(p, nsID)
}

func deleteNamespace(p *project.Project, nsID int64) error {
	return p.Write([]string{namespacesTable, eventsTable}, func(tx *project.Tx) error {
		if tx.Delete(namespacesTable, func(row project.Row) bool { return project.Int64(row["id"]) == nsID }) == 0 {
			return errNamespaceMissing
		}
		tx.Delete(eventsTable, func(row project.Row) bool { return project.Int64(row["namespace_id"]) == nsID })
		return nil
	})
}

type eventResponse struct {
	ID                int64             `json:"id"`
	ProjectID         int64             `json:"project_id"`
	NamespaceID       int64             `json:"namespace_id"`
	EventID           string            `json:"event_id"`
	EventNumber       int               `json:"event_number"`
	EventType         string            `json:"event_type"`
	Title             any               `json:"title"`
	Description       any               `json:"description"`
	Picture           string            `json:"picture"`
	IsTriggeredOnly   bool              `json:"is_triggered_only"`
	FireOnlyOnce      bool              `json:"fire_only_once"`
	Hidden            bool              `json:"hidden"`
	Major             bool              `json:"major"`
	Trigger           string            `json:"trigger"`
	MeanTimeToHappen  string            `json:"mean_time_to_happen"`
	Immediate         string            `json:"immediate"`
	After             string            `json:"after"`
	Options           []any             `json:"options"`
	TimeoutDays       int               `json:"timeout_days"`
	RawExtras         map[string]any    `json:"raw_extras"`
	CountryTag        string            `json:"country_tag"`
	ScopeKind         string            `json:"scope_kind"`
	SharedCountryTags []string          `json:"shared_country_tags"`
	TitleEn           string            `json:"title_en"`
	TitleZh           string            `json:"title_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	AutoFireOnStartup bool              `json:"auto_fire_on_startup"`
	AutoFireAfterDays int               `json:"auto_fire_after_days"`
	OnActionAttach    string            `json:"on_action_attach"`
	Source            string            `json:"source"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	DescriptionLocs   map[string]string `json:"description_locs"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
	Warnings          []string          `json:"warnings"`
}

func eventFromRow(row project.Row) eventResponse {
	ev := eventResponse{
		ID:                project.Int64(row["id"]),
		ProjectID:         project.Int64(row["project_id"]),
		NamespaceID:       project.Int64(row["namespace_id"]),
		EventID:           text(row, "event_id", ""),
		EventNumber:       whole(row, "event_number"),
		EventType:         text(row, "event_type", "country_event"),
		Title:             row["title"],
		Description:       row["description"],
		Picture:           text(row, "picture", ""),
		IsTriggeredOnly:   flag(row, "is_triggered_only", true),
		FireOnlyOnce:      flag(row, "fire_only_once", false),
		Hidden:            flag(row, "hidden", false),
		Major:             flag(row, "major", false),
		Trigger:           text(row, "trigger", ""),
		MeanTimeToHappen:  text(row, "mean_time_to_happen", ""),
		Immediate:         text(row, "immediate", ""),
		After:             text(row, "after", ""),
		TimeoutDays:       whole(row, "timeout_days"),
		CountryTag:        text(row, "country_tag", ""),
		ScopeKind:         text(row, "scope_kind", "own"),
		SharedCountryTags: stringList(row["shared_country_tags"]),
		TitleEn:           text(row, "title_en", ""),
		TitleZh:           text(row, "title_zh", ""),
		DescEn:            text(row, "desc_en", ""),
		DescZh:            text(row, "desc_zh", ""),
		AutoFireOnStartup: flag(row, "auto_fire_on_startup", false),
		AutoFireAfterDays: whole(row, "auto_fire_after_days"),
		OnActionAttach:    text(row, "on_action_attach", ""),
		Source:            text(row, "source", "created"),
		DisplayNameLocs:   stringMap(row["display_name_locs"]),
		DescriptionLocs:   stringMap(row["description_locs"]),
		CreatedAt:         text(row, "created_at", ""),
		UpdatedAt:         text(row, "updated_at", ""),
		Warnings:          []string{},
	}
	if ev.Title == nil {
		ev.Title = ""
	}
	if ev.Description == nil {
		ev.Description = ""
	}
	if options, ok := row["options"].([]any); ok {
		ev.Options = options
	} else {
		ev.Options = []any{}
	}
	if extras, ok := row["raw_extras"].(map[string]any); ok {
		ev.RawExtras = extras
	} else {
		ev.RawExtras = map[string]any{}
	}
	return ev
}

func listEvents(p *project.Project, namespaceID *int64, countryTag string) ([]eventResponse, error) {
	rows, err := p.Rows(eventsTable)
	if err != nil {
		return nil, err
	}
	kept := make([]project.Row, 0, len(rows))
	for _, row := range rows {
		if namespaceID != nil && project.Int64(row["namespace_id"]) != *namespaceID {
			continue
		}
		if countryTag != "" && row["country_tag"] != countryTag {
			continue
		}
		kept = append(kept, row)
	}
	sort.SliceStable(kept, func(i, j int) bool {
		return project.Int64(kept[i]["event_number"]) < project.Int64(kept[j]["event_number"])
	})
	out := make([]eventResponse, len(kept))
	for i, row := range kept {
		out[i] = eventFromRow(row)
	}
	return out, nil
}

func getEvent(p *project.Project, eventID int64) (eventResponse, error) {
	rows, err := p.Rows(eventsTable)
	if err != nil {
		return eventResponse{}, err
	}
	row := findRow(rows, eventID)
	if row == nil {
		return eventResponse{}, errEventMissing
	}
	return eventFromRow(row), nil
}

func eventByScriptID(p *project.Project, eventID string) (eventResponse, bool, error) {
	rows, err := p.Rows(eventsTable)
	if err != nil {
		return eventResponse{}, false, err
	}
	eventID = strings.TrimSpace(eventID)
	for _, row := range rows {
		if row["event_id"] == eventID {
			return eventFromRow(row), true, nil
		}
	}
	return eventResponse{}, false, nil
}

type eventCreateInput struct {
	NamespaceID       *int64            `json:"namespace_id"`
	EventID           *string           `json:"event_id"`
	EventNumber       *int              `json:"event_number"`
	EventType         *string           `json:"event_type"`
	Title             any               `json:"title"`
	Description       any               `json:"description"`
	Picture           string            `json:"picture"`
	IsTriggeredOnly   *bool             `json:"is_triggered_only"`
	FireOnlyOnce      bool              `json:"fire_only_once"`
	Hidden            bool              `json:"hidden"`
	Major             bool              `json:"major"`
	Trigger           string            `json:"trigger"`
	MeanTimeToHappen  string            `json:"mean_time_to_happen"`
	Immediate         *string           `json:"immediate"`
	After             *string           `json:"after"`
	Options           []any             `json:"options"`
	TimeoutDays       int               `json:"timeout_days"`
	CountryTag        string            `json:"country_tag"`
	ScopeKind         *string           `json:"scope_kind"`
	SharedCountryTags []string          `json:"shared_country_tags"`
	TitleEn           string            `json:"title_en"`
	TitleZh           string            `json:"title_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	AutoFireOnStartup bool              `json:"auto_fire_on_startup"`
	AutoFireAfterDays int               `json:"auto_fire_after_days"`
	OnActionAttach    string            `json:"on_action_attach"`
	Source            *string           `json:"source"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	DescriptionLocs   map[string]string `json:"description_locs"`
}

func (in *eventCreateInput) normalize() error {
	if in.NamespaceID == nil {
		return fmt.Errorf("namespace_id is required")
	}
	if in.EventID == nil {
		return fmt.Errorf("event_id is required")
	}
	if in.EventNumber == nil {
		return fmt.Errorf("event_number is required")
	}
	if in.Title == nil {
		in.Title = ""
	}
	if in.Description == nil {
		in.Description = ""
	}
	if in.Options == nil {
		in.Options = []any{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	return canonicalCreateBlocks(in)
}

var eventIDRE = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\.(\d+)$`)

const maxCollisionSuffixAttempts = 1000

func resolveCollisionFreeEventID(taken map[string]bool, requested string, requestedNumber *int) (string, *int, *string, error) {
	candidate := requested
	if m := eventIDRE.FindStringSubmatch(candidate); m != nil {
		nsPart := m[1]
		number := 0
		fmt.Sscan(m[2], &number)
		attempts := 0
		for taken[candidate] {
			attempts++
			if attempts > maxCollisionSuffixAttempts {
				return "", nil, nil, apiutil.ConflictError(fmt.Sprintf("event_id auto-suffix exhausted %d attempts starting at '%s' · pick manually.", maxCollisionSuffixAttempts, requested))
			}
			number++
			candidate = fmt.Sprintf("%s.%d", nsPart, number)
		}
		if attempts == 0 {
			return candidate, requestedNumber, nil, nil
		}
		w := fmt.Sprintf("event_id '%s' collided · auto-suffixed to '%s'.", requested, candidate)
		return candidate, &number, &w, nil
	}
	base := candidate
	attempts := 0
	for taken[candidate] {
		attempts++
		if attempts > maxCollisionSuffixAttempts {
			return "", nil, nil, apiutil.ConflictError(fmt.Sprintf("event_id auto-suffix exhausted %d attempts starting at '%s' · pick manually.", maxCollisionSuffixAttempts, requested))
		}
		candidate = fmt.Sprintf("%s_%d", base, attempts)
	}
	if attempts == 0 {
		return candidate, requestedNumber, nil, nil
	}
	w := fmt.Sprintf("event_id '%s' collided · auto-suffixed to '%s'.", requested, candidate)
	return candidate, requestedNumber, &w, nil
}

func takenEventIDs(rows []project.Row) map[string]bool {
	taken := make(map[string]bool, len(rows))
	for _, row := range rows {
		if id, ok := row["event_id"].(string); ok {
			taken[id] = true
		}
	}
	return taken
}

func namespaceExists(tx *project.Tx, nsID int64) bool {
	return tx.Find(namespacesTable, nsID) != nil
}

func createEvent(p *project.Project, in eventCreateInput) (eventResponse, error) {
	projectTag := text(p.Record(), "country_tag", "")
	eventID := strings.TrimSpace(*in.EventID)
	countryTag := strings.ToUpper(strings.TrimSpace(workbench.FirstNonBlank(in.CountryTag, projectTag)))

	titleEn, descEn := in.TitleEn, in.DescEn
	if t, ok := in.Title.(string); ok && strings.TrimSpace(t) != "" && strings.TrimSpace(titleEn) == "" {
		titleEn = strings.TrimSpace(t)
	}
	if d, ok := in.Description.(string); ok && strings.TrimSpace(d) != "" && strings.TrimSpace(descEn) == "" {
		descEn = strings.TrimSpace(d)
	}
	options := normalizeEventOptionsWithNameBackfill(in.Options)
	var created project.Row
	var warning *string
	err := p.Write([]string{namespacesTable, eventsTable}, func(tx *project.Tx) error {
		if !namespaceExists(tx, *in.NamespaceID) {
			return errNamespaceForeign
		}
		finalID, finalNumber, w, err := resolveCollisionFreeEventID(takenEventIDs(tx.Rows(eventsTable)), eventID, in.EventNumber)
		if err != nil {
			return err
		}
		warning = w
		number := intDefault(in.EventNumber, 0)
		if finalNumber != nil {
			number = *finalNumber
		}
		created = tx.Insert(eventsTable, project.Row{
			"namespace_id": *in.NamespaceID, "event_id": finalID, "event_number": int64(number),
			"event_type": strDefault(in.EventType, "country_event"),
			"title":      in.Title, "description": in.Description, "picture": in.Picture,
			"is_triggered_only": boolDefault(in.IsTriggeredOnly, true), "fire_only_once": in.FireOnlyOnce,
			"hidden": in.Hidden, "major": in.Major, "trigger": in.Trigger,
			"mean_time_to_happen": in.MeanTimeToHappen, "immediate": strDefault(in.Immediate, ""),
			"after": strDefault(in.After, ""), "options": options, "timeout_days": int64(in.TimeoutDays),
			"country_tag": countryTag, "scope_kind": strDefault(in.ScopeKind, "own"),
			"shared_country_tags": anyList(in.SharedCountryTags), "title_en": titleEn,
			"title_zh": in.TitleZh, "desc_en": descEn, "desc_zh": in.DescZh,
			"auto_fire_on_startup": in.AutoFireOnStartup, "auto_fire_after_days": int64(in.AutoFireAfterDays),
			"on_action_attach": in.OnActionAttach, "source": strDefault(in.Source, "created"),
			"display_name_locs": anyMap(in.DisplayNameLocs), "description_locs": anyMap(in.DescriptionLocs),
			"raw_extras": map[string]any{}, "source_snapshot": nil, "has_obs": false,
		})
		return nil
	})
	if err != nil {
		return eventResponse{}, err
	}
	ev := eventFromRow(created)
	if warning != nil {
		ev.Warnings = []string{*warning}
	}
	return ev, nil
}

type eventPatch struct {
	NamespaceID       optional[int64]             `json:"namespace_id"`
	EventID           optional[string]            `json:"event_id"`
	EventNumber       optional[int]               `json:"event_number"`
	EventType         optional[string]            `json:"event_type"`
	Title             optional[any]               `json:"title"`
	Description       optional[any]               `json:"description"`
	Picture           optional[string]            `json:"picture"`
	IsTriggeredOnly   optional[bool]              `json:"is_triggered_only"`
	FireOnlyOnce      optional[bool]              `json:"fire_only_once"`
	Hidden            optional[bool]              `json:"hidden"`
	Major             optional[bool]              `json:"major"`
	Trigger           optional[string]            `json:"trigger"`
	MeanTimeToHappen  optional[string]            `json:"mean_time_to_happen"`
	Immediate         optional[string]            `json:"immediate"`
	After             optional[string]            `json:"after"`
	Options           optional[[]any]             `json:"options"`
	TimeoutDays       optional[int]               `json:"timeout_days"`
	CountryTag        optional[string]            `json:"country_tag"`
	ScopeKind         optional[string]            `json:"scope_kind"`
	SharedCountryTags optional[[]string]          `json:"shared_country_tags"`
	TitleEn           optional[string]            `json:"title_en"`
	TitleZh           optional[string]            `json:"title_zh"`
	DescEn            optional[string]            `json:"desc_en"`
	DescZh            optional[string]            `json:"desc_zh"`
	RawExtras         optional[map[string]any]    `json:"raw_extras"`
	AutoFireOnStartup optional[bool]              `json:"auto_fire_on_startup"`
	AutoFireAfterDays optional[int]               `json:"auto_fire_after_days"`
	OnActionAttach    optional[string]            `json:"on_action_attach"`
	Source            optional[string]            `json:"source"`
	DisplayNameLocs   optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs   optional[map[string]string] `json:"description_locs"`
}

func eventRenamedAutoPicture(cur eventResponse, patch eventPatch, projectTag string) (string, bool) {
	if !patch.EventID.Set || patch.EventID.Value == nil || cur.EventID == "" || *patch.EventID.Value == cur.EventID {
		return "", false
	}
	oldTag := workbench.FirstNonBlank(cur.CountryTag, projectTag)
	newTag := oldTag
	if patch.CountryTag.Set && patch.CountryTag.Value != nil {
		newTag = workbench.FirstNonBlank(*patch.CountryTag.Value, projectTag)
	}
	currentPicture := cur.Picture
	if patch.Picture.Set && patch.Picture.Value != nil {
		currentPicture = *patch.Picture.Value
	}
	if strings.TrimSpace(currentPicture) != eventPictureSpriteName(oldTag, cur.EventID) {
		return "", false
	}
	return eventPictureSpriteName(newTag, *patch.EventID.Value), true
}

func patchColumn[T any](changes project.Row, column string, value optional[T], notNull bool) error {
	if !value.Set {
		return nil
	}
	if value.Value == nil {
		if notNull {
			return apiutil.ValidationError(column + " cannot be null")
		}
		changes[column] = nil
		return nil
	}
	switch v := any(*value.Value).(type) {
	case int:
		changes[column] = int64(v)
	case []string:
		changes[column] = anyList(v)
	case map[string]string:
		changes[column] = anyMap(v)
	default:
		changes[column] = v
	}
	return nil
}

func updateEvent(p *project.Project, eventID int64, patch eventPatch) (eventResponse, error) {
	cur, err := getEvent(p, eventID)
	if err != nil {
		return eventResponse{}, err
	}
	if err := canonicalPatchBlocks(&patch); err != nil {
		return eventResponse{}, apiutil.BadRequestError(err.Error())
	}

	if patch.Title.Set && patch.Title.Value != nil {
		if t, ok := (*patch.Title.Value).(string); ok && strings.TrimSpace(t) != "" {
			if !patch.TitleEn.Set || patch.TitleEn.Value == nil || strings.TrimSpace(*patch.TitleEn.Value) == "" {
				v := strings.TrimSpace(t)
				patch.TitleEn = optional[string]{Set: true, Value: &v}
			}
		}
	}
	if patch.Description.Set && patch.Description.Value != nil {
		if d, ok := (*patch.Description.Value).(string); ok && strings.TrimSpace(d) != "" {
			if !patch.DescEn.Set || patch.DescEn.Value == nil || strings.TrimSpace(*patch.DescEn.Value) == "" {
				v := strings.TrimSpace(d)
				patch.DescEn = optional[string]{Set: true, Value: &v}
			}
		}
	}
	if patch.Options.Set && patch.Options.Value != nil {
		normalized := normalizeEventOptionsWithNameBackfill(*patch.Options.Value)
		patch.Options.Value = &normalized
	}
	if patch.EventID.Set && patch.EventID.Value != nil {
		v := strings.TrimSpace(*patch.EventID.Value)
		patch.EventID.Value = &v
	}
	if patch.CountryTag.Set && patch.CountryTag.Value != nil {
		v := strings.ToUpper(strings.TrimSpace(*patch.CountryTag.Value))
		patch.CountryTag.Value = &v
	}
	if picture, rename := eventRenamedAutoPicture(cur, patch, text(p.Record(), "country_tag", "")); rename {
		patch.Picture = optional[string]{Set: true, Value: &picture}
	}

	empty := ""
	if patch.Immediate.Set && patch.Immediate.Value == nil {
		patch.Immediate.Value = &empty
	}
	if patch.After.Set && patch.After.Value == nil {
		patch.After.Value = &empty
	}

	changes := project.Row{}
	for _, err := range []error{
		patchColumn(changes, "namespace_id", patch.NamespaceID, true),
		patchColumn(changes, "event_id", patch.EventID, true),
		patchColumn(changes, "event_number", patch.EventNumber, true),
		patchColumn(changes, "event_type", patch.EventType, true),
		patchColumn(changes, "title", patch.Title, false),
		patchColumn(changes, "description", patch.Description, false),
		patchColumn(changes, "picture", patch.Picture, false),
		patchColumn(changes, "is_triggered_only", patch.IsTriggeredOnly, false),
		patchColumn(changes, "fire_only_once", patch.FireOnlyOnce, false),
		patchColumn(changes, "hidden", patch.Hidden, false),
		patchColumn(changes, "major", patch.Major, false),
		patchColumn(changes, "trigger", patch.Trigger, false),
		patchColumn(changes, "mean_time_to_happen", patch.MeanTimeToHappen, false),
		patchColumn(changes, "immediate", patch.Immediate, true),
		patchColumn(changes, "after", patch.After, true),
		patchColumn(changes, "options", patch.Options, false),
		patchColumn(changes, "timeout_days", patch.TimeoutDays, false),
		patchColumn(changes, "country_tag", patch.CountryTag, false),
		patchColumn(changes, "scope_kind", patch.ScopeKind, true),
		patchColumn(changes, "shared_country_tags", patch.SharedCountryTags, false),
		patchColumn(changes, "title_en", patch.TitleEn, false),
		patchColumn(changes, "title_zh", patch.TitleZh, false),
		patchColumn(changes, "desc_en", patch.DescEn, false),
		patchColumn(changes, "desc_zh", patch.DescZh, false),
		patchColumn(changes, "raw_extras", patch.RawExtras, false),
		patchColumn(changes, "auto_fire_on_startup", patch.AutoFireOnStartup, true),
		patchColumn(changes, "auto_fire_after_days", patch.AutoFireAfterDays, true),
		patchColumn(changes, "on_action_attach", patch.OnActionAttach, true),
		patchColumn(changes, "display_name_locs", patch.DisplayNameLocs, true),
		patchColumn(changes, "description_locs", patch.DescriptionLocs, true),
	} {
		if err != nil {
			return eventResponse{}, err
		}
	}

	flipToCreated := cur.Source == "imported"
	if patch.Source.Set && patch.Source.Value != nil {
		flipToCreated = *patch.Source.Value == "imported"
	}
	if flipToCreated {
		changes["source"] = "created"
	} else if err := patchColumn(changes, "source", patch.Source, true); err != nil {
		return eventResponse{}, err
	}

	var updated project.Row
	err = p.Write([]string{namespacesTable, eventsTable}, func(tx *project.Tx) error {
		if tx.Find(eventsTable, eventID) == nil {
			return errEventMissing
		}
		if ns, ok := changes["namespace_id"]; ok && !namespaceExists(tx, project.Int64(ns)) {
			return errNamespaceForeign
		}
		if id, ok := changes["event_id"].(string); ok && id != cur.EventID {
			for _, row := range tx.Rows(eventsTable) {
				if project.Int64(row["id"]) != eventID && row["event_id"] == id {
					return eventIDConflict(id)
				}
			}
		}
		if len(changes) == 0 {
			updated = tx.Find(eventsTable, eventID)
			return nil
		}
		var err error
		updated, err = tx.Update(eventsTable, eventID, func(row project.Row) {
			for k, v := range changes {
				row[k] = v
			}
		})
		return err
	})
	if err != nil {
		return eventResponse{}, err
	}
	return eventFromRow(updated), nil
}

func deleteEvent(p *project.Project, eventID int64) error {
	return p.Write([]string{eventsTable}, func(tx *project.Tx) error {
		if tx.Delete(eventsTable, func(row project.Row) bool { return project.Int64(row["id"]) == eventID }) == 0 {
			return errEventMissing
		}
		return nil
	})
}

var forkColumnsSkipped = map[string]bool{
	"id": true, "created_at": true, "updated_at": true, "source": true, "event_id": true, "event_number": true,
}

func forkEvent(p *project.Project, eventID int64, newEventID string) (eventResponse, error) {
	var created project.Row
	err := p.Write([]string{eventsTable}, func(tx *project.Tx) error {
		src := tx.Find(eventsTable, eventID)
		if src == nil {
			return errSourceMissing
		}
		m := eventIDRE.FindStringSubmatch(newEventID)
		if m == nil {
			return apiutil.BadRequestError("new_event_id must match 'namespace.NUMBER' (e.g. 'france.200')")
		}
		if newEventID == text(src, "event_id", "") {
			return apiutil.BadRequestError("new_event_id must differ from the original")
		}
		if takenEventIDs(tx.Rows(eventsTable))[newEventID] {
			return eventIDConflict(newEventID)
		}
		number := 0
		fmt.Sscan(m[2], &number)
		row := project.Row{}
		for k, v := range src {
			if !forkColumnsSkipped[k] {
				row[k] = v
			}
		}
		row["event_id"] = newEventID
		row["event_number"] = int64(number)
		row["source"] = "created"
		created = tx.Insert(eventsTable, row)
		return nil
	})
	if err != nil {
		return eventResponse{}, err
	}
	return eventFromRow(created), nil
}
