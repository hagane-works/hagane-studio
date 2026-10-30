package content

import (
	"fmt"
	"net/http"
	"strings"

	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

var errInvalidPayload = statusErr(http.StatusUnprocessableEntity, "Invalid payload")

const dynamicModifiersTable = "dynamic_modifiers"

type dynamicModifier struct {
	ID              int64             `json:"id"`
	ProjectID       int64             `json:"project_id"`
	DmID            string            `json:"dm_id"`
	CountryTag      string            `json:"country_tag"`
	NameEn          string            `json:"name_en"`
	NameZh          string            `json:"name_zh"`
	DisplayNameLocs map[string]string `json:"display_name_locs"`
	Icon            string            `json:"icon"`
	RemoveTrigger   string            `json:"remove_trigger"`
	Enable          string            `json:"enable"`
	Modifier        map[string]any    `json:"modifier"`
	RawExtras       map[string]any    `json:"raw_extras"`
	Source          string            `json:"source"`
	CreatedAt       string            `json:"created_at"`
	UpdatedAt       string            `json:"updated_at"`
	Warnings        []string          `json:"warnings"`
}

func dynamicModifierFromRow(row project.Row) dynamicModifier {
	return dynamicModifier{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		DmID: text(row, "dm_id"), CountryTag: text(row, "country_tag"),
		NameEn: text(row, "name_en"), NameZh: text(row, "name_zh"),
		DisplayNameLocs: stringMap(row["display_name_locs"]), Icon: text(row, "icon"),
		RemoveTrigger: text(row, "remove_trigger"), Enable: text(row, "enable"),
		Modifier: anyMap(row["modifier"]), RawExtras: anyMap(row["raw_extras"]),
		Source:    textOr(row, "source", "created"),
		CreatedAt: isoNaive(row["created_at"]), UpdatedAt: isoNaive(row["updated_at"]),
		Warnings: []string{},
	}
}

type dynamicModifierCreate struct {
	DmID            string            `json:"dm_id"`
	CountryTag      string            `json:"country_tag"`
	NameEn          string            `json:"name_en"`
	NameZh          string            `json:"name_zh"`
	DisplayNameLocs map[string]string `json:"display_name_locs"`
	Icon            string            `json:"icon"`
	RemoveTrigger   string            `json:"remove_trigger"`
	Enable          string            `json:"enable"`
	Modifier        map[string]any    `json:"modifier"`
	RawExtras       map[string]any    `json:"raw_extras"`
	Source          *string           `json:"source"`
}

type dynamicModifierPatch struct {
	DmID            workbench.Optional[string]            `json:"dm_id"`
	CountryTag      workbench.Optional[string]            `json:"country_tag"`
	NameEn          workbench.Optional[string]            `json:"name_en"`
	NameZh          workbench.Optional[string]            `json:"name_zh"`
	DisplayNameLocs workbench.Optional[map[string]string] `json:"display_name_locs"`
	Icon            workbench.Optional[string]            `json:"icon"`
	RemoveTrigger   workbench.Optional[string]            `json:"remove_trigger"`
	Enable          workbench.Optional[string]            `json:"enable"`
	Modifier        workbench.Optional[map[string]any]    `json:"modifier"`
	RawExtras       workbench.Optional[map[string]any]    `json:"raw_extras"`
	Source          workbench.Optional[string]            `json:"source"`
}

func (h *handlers) dynamicModifiers(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	const missing = "DynamicModifier not found"
	crud{
		listKey: "dynamic_modifiers", missing: missing, unavailable: "store unavailable",
		list: func(p *project.Project, _ *http.Request) (any, int, error) {
			items, err := listRows(p, dynamicModifiersTable, nil, dynamicModifierFromRow)
			return items, len(items), err
		},
		create: h.createDynamicModifier,
		get: func(p *project.Project, id int64) (any, error) {
			row, err := getRow(p, dynamicModifiersTable, id, missing)
			if err != nil {
				return nil, err
			}
			return dynamicModifierFromRow(row), nil
		},
		update: updateDynamicModifier,
		remove: removeRow(dynamicModifiersTable, missing),
	}.serve(p, w, r, rest)
}

func suffixed(rows []project.Row, column, base string) string {
	candidate := base
	for suffix := 2; taken(rows, 0, func(row project.Row) bool { return text(row, column) == candidate }); suffix++ {
		candidate = fmt.Sprintf("%s_%d", base, suffix)
	}
	return candidate
}

func (h *handlers) createDynamicModifier(p *project.Project, r *http.Request) (any, error) {
	var in dynamicModifierCreate
	if err := decodeBody(r, &in); err != nil {
		return nil, errInvalidPayload
	}
	baseID := strings.TrimSpace(in.DmID)
	if baseID == "" {
		return nil, statusErr(http.StatusUnprocessableEntity, "dm_id required")
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.Modifier == nil {
		in.Modifier = map[string]any{}
	}
	if in.RawExtras == nil {
		in.RawExtras = map[string]any{}
	}
	source := workbench.StrDefault(in.Source, "created")
	if source == "" {
		source = "created"
	}
	var dmID string
	row, err := writeRow(p, dynamicModifiersTable, func(tx *project.Tx) (project.Row, error) {
		dmID = suffixed(tx.Rows(dynamicModifiersTable), "dm_id", baseID)
		return insert(tx, dynamicModifiersTable, project.Row{
			"dm_id": dmID, "country_tag": strings.ToUpper(in.CountryTag),
			"name_en": in.NameEn, "name_zh": in.NameZh,
			"display_name_locs": jsonValue(in.DisplayNameLocs), "icon": in.Icon,
			"remove_trigger": in.RemoveTrigger, "enable": in.Enable,
			"modifier": jsonValue(in.Modifier), "raw_extras": jsonValue(in.RawExtras),
			"source": source,
		})
	})
	if err != nil {
		return nil, err
	}
	item := dynamicModifierFromRow(row)

	if warning := h.vanillaDynamicModifierWarning(baseID, item.CountryTag, item.Source); warning != "" {
		item.Warnings = append(item.Warnings, warning)
	}
	if dmID != baseID {
		item.Warnings = append(item.Warnings, fmt.Sprintf("dm_id %s collided · auto-suffixed to %s.", pyRepr(baseID), pyRepr(dmID)))
	}
	return item, nil
}

func (h *handlers) vanillaDynamicModifierWarning(dmID, countryTag, source string) string {
	if dmID == "" || source == "imported" || !isVanillaDynamicModifier(h.deps.Game, dmID) {
		return ""
	}
	prefix := "MOD_"
	if countryTag != "" {
		prefix = strings.ToUpper(countryTag) + "_"
	}
	suggested := dmID
	if !strings.HasPrefix(dmID, prefix) {
		suggested = prefix + dmID
	}
	return fmt.Sprintf("%s 跟 vanilla dynamic_modifier 同名 · HOI4 mod 加载会 silent override "+
		"vanilla · 建议改名 %s(同 country TAG prefix 防 collision)", pyRepr(dmID), pyRepr(suggested))
}

func flipSource(sets project.Row, current string, source workbench.Optional[string]) {
	flip := false
	if current == "imported" {
		final := current
		if source.Set && source.Value != nil {
			final = *source.Value
		}
		flip = final == "imported"
	}
	if source.Set && !flip {
		sets["source"] = optValue(source.Value)
	}
	if flip {
		sets["source"] = "created"
	}
}

func optString(o workbench.Optional[string]) string {
	if o.Value != nil {
		return *o.Value
	}
	return ""
}

func setOpt[T any](sets project.Row, column string, o workbench.Optional[T]) {
	if o.Set {
		sets[column] = optValue(o.Value)
	}
}

func setOptJSON[T any](sets project.Row, column string, o workbench.Optional[T]) {
	if o.Set {
		sets[column] = optJSON(o.Value)
	}
}

func uniqueOn(table, column string, id int64, sets project.Row) func(tx *project.Tx) error {
	return func(tx *project.Tx) error {
		value, ok := sets[column]
		if !ok || value == nil {
			return nil
		}
		if taken(tx.Rows(table), id, func(row project.Row) bool { return row[column] == value }) {
			return errConstraint
		}
		return nil
	}
}

func updateDynamicModifier(p *project.Project, r *http.Request, id int64) (any, error) {
	var patch dynamicModifierPatch
	if err := decodeBody(r, &patch); err != nil {
		return nil, errInvalidPayload
	}
	cur, err := getRow(p, dynamicModifiersTable, id, "DynamicModifier not found")
	if err != nil {
		return nil, err
	}
	sets := project.Row{}
	if patch.DmID.Set {
		sets["dm_id"] = strings.TrimSpace(optString(patch.DmID))
	}
	if patch.CountryTag.Set {
		sets["country_tag"] = strings.ToUpper(optString(patch.CountryTag))
	}
	setOpt(sets, "name_en", patch.NameEn)
	setOpt(sets, "name_zh", patch.NameZh)
	setOptJSON(sets, "display_name_locs", patch.DisplayNameLocs)
	setOpt(sets, "icon", patch.Icon)
	setOpt(sets, "remove_trigger", patch.RemoveTrigger)
	setOpt(sets, "enable", patch.Enable)
	setOptJSON(sets, "modifier", patch.Modifier)
	setOptJSON(sets, "raw_extras", patch.RawExtras)
	flipSource(sets, textOr(cur, "source", "created"), patch.Source)
	row, err := updateRow(p, dynamicModifiersTable, id, sets, uniqueOn(dynamicModifiersTable, "dm_id", id, sets))
	if err != nil {
		return nil, err
	}
	return dynamicModifierFromRow(row), nil
}

const autonomyStatesTable = "autonomy_states"

type autonomyState struct {
	ID                int64             `json:"id"`
	ProjectID         int64             `json:"project_id"`
	AutonomyID        string            `json:"autonomy_id"`
	CountryTag        string            `json:"country_tag"`
	NameEn            string            `json:"name_en"`
	NameZh            string            `json:"name_zh"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	IsPuppet          bool              `json:"is_puppet"`
	IsDefault         bool              `json:"is_default"`
	ManpowerInfluence *float64          `json:"manpower_influence"`
	MinFreedomLevel   int               `json:"min_freedom_level"`
	Rule              string            `json:"rule"`
	AiWeights         map[string]any    `json:"ai_weights"`
	Modifier          map[string]any    `json:"modifier"`
	RawExtras         map[string]any    `json:"raw_extras"`
	Source            string            `json:"source"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
	Warnings          []string          `json:"warnings"`
}

func autonomyStateFromRow(row project.Row) autonomyState {
	return autonomyState{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		AutonomyID: text(row, "autonomy_id"), CountryTag: text(row, "country_tag"),
		NameEn: text(row, "name_en"), NameZh: text(row, "name_zh"),
		DisplayNameLocs: stringMap(row["display_name_locs"]),
		IsPuppet:        boolOr(row, "is_puppet", false), IsDefault: boolOr(row, "is_default", false),
		ManpowerInfluence: floatPtr(row, "manpower_influence"),
		MinFreedomLevel:   intOr(row, "min_freedom_level", 0), Rule: text(row, "rule"),
		AiWeights: anyMap(row["ai_weights"]), Modifier: anyMap(row["modifier"]),
		RawExtras: anyMap(row["raw_extras"]), Source: textOr(row, "source", "created"),
		CreatedAt: isoNaive(row["created_at"]), UpdatedAt: isoNaive(row["updated_at"]),
		Warnings: []string{},
	}
}

type autonomyStateCreate struct {
	AutonomyID        string            `json:"autonomy_id"`
	CountryTag        string            `json:"country_tag"`
	NameEn            string            `json:"name_en"`
	NameZh            string            `json:"name_zh"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	IsPuppet          bool              `json:"is_puppet"`
	IsDefault         bool              `json:"is_default"`
	ManpowerInfluence *float64          `json:"manpower_influence"`
	MinFreedomLevel   int               `json:"min_freedom_level"`
	Rule              string            `json:"rule"`
	AiWeights         map[string]any    `json:"ai_weights"`
	Modifier          map[string]any    `json:"modifier"`
	RawExtras         map[string]any    `json:"raw_extras"`
	Source            *string           `json:"source"`
}

type autonomyStatePatch struct {
	AutonomyID        workbench.Optional[string]            `json:"autonomy_id"`
	CountryTag        workbench.Optional[string]            `json:"country_tag"`
	NameEn            workbench.Optional[string]            `json:"name_en"`
	NameZh            workbench.Optional[string]            `json:"name_zh"`
	DisplayNameLocs   workbench.Optional[map[string]string] `json:"display_name_locs"`
	IsPuppet          workbench.Optional[bool]              `json:"is_puppet"`
	IsDefault         workbench.Optional[bool]              `json:"is_default"`
	ManpowerInfluence workbench.Optional[float64]           `json:"manpower_influence"`
	MinFreedomLevel   workbench.Optional[int]               `json:"min_freedom_level"`
	Rule              workbench.Optional[string]            `json:"rule"`
	AiWeights         workbench.Optional[map[string]any]    `json:"ai_weights"`
	Modifier          workbench.Optional[map[string]any]    `json:"modifier"`
	RawExtras         workbench.Optional[map[string]any]    `json:"raw_extras"`
	Source            workbench.Optional[string]            `json:"source"`
}

func (h *handlers) autonomyStates(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	const missing = "AutonomyState not found"
	crud{
		listKey: "autonomy_states", missing: missing, unavailable: "store unavailable",
		list: func(p *project.Project, _ *http.Request) (any, int, error) {
			items, err := listRows(p, autonomyStatesTable, nil, autonomyStateFromRow)
			return items, len(items), err
		},
		create: createAutonomyState,
		get: func(p *project.Project, id int64) (any, error) {
			row, err := getRow(p, autonomyStatesTable, id, missing)
			if err != nil {
				return nil, err
			}
			return autonomyStateFromRow(row), nil
		},
		update: updateAutonomyState,
		remove: removeRow(autonomyStatesTable, missing),
	}.serve(p, w, r, rest)
}

func createAutonomyState(p *project.Project, r *http.Request) (any, error) {
	var in autonomyStateCreate
	if err := decodeBody(r, &in); err != nil {
		return nil, errInvalidPayload
	}
	baseID := strings.TrimSpace(in.AutonomyID)
	if baseID == "" {
		return nil, statusErr(http.StatusUnprocessableEntity, "autonomy_id required")
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.AiWeights == nil {
		in.AiWeights = map[string]any{}
	}
	if in.Modifier == nil {
		in.Modifier = map[string]any{}
	}
	if in.RawExtras == nil {
		in.RawExtras = map[string]any{}
	}
	var autonomyID string
	row, err := writeRow(p, autonomyStatesTable, func(tx *project.Tx) (project.Row, error) {
		autonomyID = suffixed(tx.Rows(autonomyStatesTable), "autonomy_id", baseID)
		return insert(tx, autonomyStatesTable, project.Row{
			"autonomy_id": autonomyID, "country_tag": strings.ToUpper(in.CountryTag),
			"name_en": in.NameEn, "name_zh": in.NameZh,
			"display_name_locs": jsonValue(in.DisplayNameLocs),
			"is_puppet":         in.IsPuppet, "is_default": in.IsDefault,
			"manpower_influence": optValue(in.ManpowerInfluence),
			"min_freedom_level":  int64(in.MinFreedomLevel), "rule": in.Rule,
			"ai_weights": jsonValue(in.AiWeights), "modifier": jsonValue(in.Modifier),
			"raw_extras": jsonValue(in.RawExtras),
			"source":     workbench.StrDefault(in.Source, "created"),
		})
	})
	if err != nil {
		return nil, err
	}
	item := autonomyStateFromRow(row)
	if autonomyID != baseID {
		item.Warnings = append(item.Warnings, fmt.Sprintf("autonomy_id %s collided · auto-suffixed to %s.", pyRepr(baseID), pyRepr(autonomyID)))
	}
	return item, nil
}

func updateAutonomyState(p *project.Project, r *http.Request, id int64) (any, error) {
	var patch autonomyStatePatch
	if err := decodeBody(r, &patch); err != nil {
		return nil, errInvalidPayload
	}
	cur, err := getRow(p, autonomyStatesTable, id, "AutonomyState not found")
	if err != nil {
		return nil, err
	}
	sets := project.Row{}
	if patch.AutonomyID.Set {
		sets["autonomy_id"] = strings.TrimSpace(optString(patch.AutonomyID))
	}
	if patch.CountryTag.Set {
		sets["country_tag"] = strings.ToUpper(optString(patch.CountryTag))
	}
	setOpt(sets, "name_en", patch.NameEn)
	setOpt(sets, "name_zh", patch.NameZh)
	setOptJSON(sets, "display_name_locs", patch.DisplayNameLocs)
	setOpt(sets, "is_puppet", patch.IsPuppet)
	setOpt(sets, "is_default", patch.IsDefault)
	setOpt(sets, "manpower_influence", patch.ManpowerInfluence)
	setOpt(sets, "min_freedom_level", patch.MinFreedomLevel)
	setOpt(sets, "rule", patch.Rule)
	setOptJSON(sets, "ai_weights", patch.AiWeights)
	setOptJSON(sets, "modifier", patch.Modifier)
	setOptJSON(sets, "raw_extras", patch.RawExtras)
	flipSource(sets, textOr(cur, "source", "created"), patch.Source)
	row, err := updateRow(p, autonomyStatesTable, id, sets, uniqueOn(autonomyStatesTable, "autonomy_id", id, sets))
	if err != nil {
		return nil, err
	}
	return autonomyStateFromRow(row), nil
}

const scriptedSnippetsTable = "scripted_snippets"

func validateSnippetKind(value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "effect" || v == "trigger" {
		return v, nil
	}
	return "", statusErr(http.StatusBadRequest, fmt.Sprintf("kind must be one of ('effect', 'trigger'), got '%s'", value))
}

type scriptedSnippet struct {
	ID         int64          `json:"id"`
	ProjectID  int64          `json:"project_id"`
	Kind       string         `json:"kind"`
	Name       string         `json:"name"`
	Body       string         `json:"body"`
	SourceFile string         `json:"source_file"`
	Source     string         `json:"source"`
	RawExtras  map[string]any `json:"raw_extras"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

func scriptedSnippetFromRow(row project.Row) scriptedSnippet {
	item := scriptedSnippet{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		Kind: text(row, "kind"), Name: text(row, "name"), Body: text(row, "body"),
		SourceFile: text(row, "source_file"), Source: textOr(row, "source", "created"),
		CreatedAt: isoNaive(row["created_at"]), UpdatedAt: isoNaive(row["updated_at"]),
	}
	decode(row["raw_extras"], &item.RawExtras)
	return item
}

type scriptedSnippetCreate struct {
	Kind       *string        `json:"kind"`
	Name       string         `json:"name"`
	Body       string         `json:"body"`
	SourceFile string         `json:"source_file"`
	Source     *string        `json:"source"`
	RawExtras  map[string]any `json:"raw_extras"`
}

type scriptedSnippetPatch struct {
	Kind       workbench.Optional[string]         `json:"kind"`
	Name       workbench.Optional[string]         `json:"name"`
	Body       workbench.Optional[string]         `json:"body"`
	SourceFile workbench.Optional[string]         `json:"source_file"`
	RawExtras  workbench.Optional[map[string]any] `json:"raw_extras"`
}

func (h *handlers) scriptedSnippets(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	const missing = "Scripted snippet not found"
	crud{
		listKey: "snippets", missing: missing, unavailable: "store unavailable",
		list: func(p *project.Project, r *http.Request) (any, int, error) {
			rows, err := p.Rows(scriptedSnippetsTable)
			if err != nil {
				return nil, 0, err
			}
			kind := queryPtr(r, "kind")
			sortRows(rows, func(a, b project.Row) bool {
				if text(a, "kind") != text(b, "kind") {
					return text(a, "kind") < text(b, "kind")
				}
				return text(a, "name") < text(b, "name")
			})
			items := []scriptedSnippet{}
			for _, row := range rows {
				if kind != nil && text(row, "kind") != strings.TrimSpace(*kind) {
					continue
				}
				items = append(items, scriptedSnippetFromRow(row))
			}
			return items, len(items), nil
		},
		create: createScriptedSnippet,
		get: func(p *project.Project, id int64) (any, error) {
			row, err := getRow(p, scriptedSnippetsTable, id, missing)
			if err != nil {
				return nil, err
			}
			return scriptedSnippetFromRow(row), nil
		},
		update: updateScriptedSnippet,
		remove: removeRow(scriptedSnippetsTable, missing),
	}.serve(p, w, r, rest)
}

func createScriptedSnippet(p *project.Project, r *http.Request) (any, error) {
	var in scriptedSnippetCreate
	if err := decodeBody(r, &in); err != nil {
		return nil, errInvalidPayload
	}
	kind := "effect"
	if in.Kind != nil {
		kind = *in.Kind
	}
	kind, err := validateSnippetKind(kind)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, statusErr(http.StatusBadRequest, "name is required (e.g. 'my_effect_name').")
	}
	row, err := writeRow(p, scriptedSnippetsTable, func(tx *project.Tx) (project.Row, error) {
		if taken(tx.Rows(scriptedSnippetsTable), 0, func(row project.Row) bool {
			return text(row, "kind") == kind && text(row, "name") == name
		}) {
			return nil, statusErr(http.StatusConflict, fmt.Sprintf(
				"A scripted %s named '%s' already exists in this project. Pick a unique name.", kind, name))
		}
		return insert(tx, scriptedSnippetsTable, project.Row{
			"kind": kind, "name": name, "body": in.Body, "source_file": in.SourceFile,
			"source": workbench.StrDefault(in.Source, "created"), "raw_extras": jsonValue(in.RawExtras),
		})
	})
	if err != nil {
		return nil, err
	}
	return scriptedSnippetFromRow(row), nil
}

func updateScriptedSnippet(p *project.Project, r *http.Request, id int64) (any, error) {
	var patch scriptedSnippetPatch
	if err := decodeBody(r, &patch); err != nil {
		return nil, errInvalidPayload
	}
	cur, err := getRow(p, scriptedSnippetsTable, id, "Scripted snippet not found")
	if err != nil {
		return nil, err
	}
	sets := project.Row{}
	if patch.Kind.Set {
		kind, err := validateSnippetKind(optString(patch.Kind))
		if err != nil {
			return nil, err
		}
		sets["kind"] = kind
	}
	if patch.Name.Set {
		name := strings.TrimSpace(optString(patch.Name))
		if name == "" {
			return nil, statusErr(http.StatusBadRequest, "name cannot be empty")
		}
		sets["name"] = name
	}
	setOpt(sets, "body", patch.Body)
	setOpt(sets, "source_file", patch.SourceFile)
	setOptJSON(sets, "raw_extras", patch.RawExtras)
	if text(cur, "source") == "imported" {
		sets["source"] = "created"
	}
	row, err := updateRow(p, scriptedSnippetsTable, id, sets, func(tx *project.Tx) error {
		kind, name := textOr(sets, "kind", text(cur, "kind")), textOr(sets, "name", text(cur, "name"))
		if taken(tx.Rows(scriptedSnippetsTable), id, func(row project.Row) bool {
			return text(row, "kind") == kind && text(row, "name") == name
		}) {
			return statusErr(http.StatusConflict, "kind + name conflict in this project")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return scriptedSnippetFromRow(row), nil
}

const scriptedLocalisationsTable = "scripted_localisations"

type scriptedLocalisation struct {
	ID         int64          `json:"id"`
	ProjectID  int64          `json:"project_id"`
	Name       string         `json:"name"`
	Body       string         `json:"body"`
	SourceFile string         `json:"source_file"`
	Source     string         `json:"source"`
	RawExtras  map[string]any `json:"raw_extras"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

func scriptedLocalisationFromRow(row project.Row) scriptedLocalisation {
	item := scriptedLocalisation{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		Name: text(row, "name"), Body: text(row, "body"), SourceFile: text(row, "source_file"),
		Source:    textOr(row, "source", "created"),
		CreatedAt: isoNaive(row["created_at"]), UpdatedAt: isoNaive(row["updated_at"]),
	}
	decode(row["raw_extras"], &item.RawExtras)
	return item
}

type scriptedLocalisationCreate struct {
	Name       string         `json:"name"`
	Body       string         `json:"body"`
	SourceFile string         `json:"source_file"`
	Source     *string        `json:"source"`
	RawExtras  map[string]any `json:"raw_extras"`
}

type scriptedLocalisationPatch struct {
	Name       workbench.Optional[string]         `json:"name"`
	Body       workbench.Optional[string]         `json:"body"`
	SourceFile workbench.Optional[string]         `json:"source_file"`
	RawExtras  workbench.Optional[map[string]any] `json:"raw_extras"`
}

func (h *handlers) scriptedLocalisations(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	const missing = "Scripted localisation not found"
	crud{
		listKey: "localisations", missing: missing, unavailable: "store unavailable",
		list: func(p *project.Project, _ *http.Request) (any, int, error) {
			items, err := listRows(p, scriptedLocalisationsTable, byText("name"), scriptedLocalisationFromRow)
			return items, len(items), err
		},
		create: createScriptedLocalisation,
		get: func(p *project.Project, id int64) (any, error) {
			row, err := getRow(p, scriptedLocalisationsTable, id, missing)
			if err != nil {
				return nil, err
			}
			return scriptedLocalisationFromRow(row), nil
		},
		update: updateScriptedLocalisation,
		remove: removeRow(scriptedLocalisationsTable, missing),
	}.serve(p, w, r, rest)
}

func createScriptedLocalisation(p *project.Project, r *http.Request) (any, error) {
	var in scriptedLocalisationCreate
	if err := decodeBody(r, &in); err != nil {
		return nil, errInvalidPayload
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, statusErr(http.StatusBadRequest, "name is required (e.g. 'GetMyDynamicText').")
	}
	row, err := writeRow(p, scriptedLocalisationsTable, func(tx *project.Tx) (project.Row, error) {
		if taken(tx.Rows(scriptedLocalisationsTable), 0, func(row project.Row) bool { return text(row, "name") == name }) {
			return nil, statusErr(http.StatusConflict, fmt.Sprintf(
				"A scripted localisation named '%s' already exists in this project. Pick a unique name.", name))
		}
		return insert(tx, scriptedLocalisationsTable, project.Row{
			"name": name, "body": in.Body, "source_file": in.SourceFile,
			"source": workbench.StrDefault(in.Source, "created"), "raw_extras": jsonValue(in.RawExtras),
		})
	})
	if err != nil {
		return nil, err
	}
	return scriptedLocalisationFromRow(row), nil
}

func updateScriptedLocalisation(p *project.Project, r *http.Request, id int64) (any, error) {
	var patch scriptedLocalisationPatch
	if err := decodeBody(r, &patch); err != nil {
		return nil, errInvalidPayload
	}
	cur, err := getRow(p, scriptedLocalisationsTable, id, "Scripted localisation not found")
	if err != nil {
		return nil, err
	}
	sets := project.Row{}
	if patch.Name.Set {
		name := strings.TrimSpace(optString(patch.Name))
		if name == "" {
			return nil, statusErr(http.StatusBadRequest, "name cannot be empty")
		}
		sets["name"] = name
	}
	setOpt(sets, "body", patch.Body)
	setOpt(sets, "source_file", patch.SourceFile)
	setOptJSON(sets, "raw_extras", patch.RawExtras)
	if text(cur, "source") == "imported" {
		sets["source"] = "created"
	}
	row, err := updateRow(p, scriptedLocalisationsTable, id, sets, func(tx *project.Tx) error {
		if name, ok := sets["name"].(string); ok && taken(tx.Rows(scriptedLocalisationsTable), id, func(row project.Row) bool {
			return text(row, "name") == name
		}) {
			return statusErr(http.StatusConflict, "name conflict in this project")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return scriptedLocalisationFromRow(row), nil
}
