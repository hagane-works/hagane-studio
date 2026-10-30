package content

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const opinionModifiersTable = "opinion_modifiers"

var opinionBodyFields = []string{
	"value", "decay", "days", "months", "years", "min_trust", "max_trust", "trade",
}

type opinionModifier struct {
	ID         int64          `json:"id"`
	ProjectID  int64          `json:"project_id"`
	ModifierID string         `json:"modifier_id"`
	CountryTag string         `json:"country_tag"`
	Value      any            `json:"value"`
	Decay      any            `json:"decay"`
	Days       any            `json:"days"`
	Months     any            `json:"months"`
	Years      any            `json:"years"`
	MinTrust   any            `json:"min_trust"`
	MaxTrust   any            `json:"max_trust"`
	Trade      *bool          `json:"trade"`
	Body       map[string]any `json:"body"`
	RawExtras  map[string]any `json:"raw_extras"`
	Source     string         `json:"source"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

func opinionTradeFromStorage(value any) *bool {
	if value == nil {
		return nil
	}
	if b, ok := value.(bool); ok {
		return &b
	}
	switch strings.ToLower(strings.TrimSpace(anyToString(value))) {
	case "yes", "true", "1":
		t := true
		return &t
	case "no", "false", "0":
		f := false
		return &f
	}
	return nil
}

func opinionModifierFromRow(row project.Row) opinionModifier {
	body := anyMap(row["body"])
	return opinionModifier{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		ModifierID: text(row, "modifier_id"), CountryTag: text(row, "country_tag"),
		Value: body["value"], Decay: body["decay"], Days: body["days"],
		Months: body["months"], Years: body["years"],
		MinTrust: body["min_trust"], MaxTrust: body["max_trust"],
		Trade: opinionTradeFromStorage(body["trade"]),
		Body:  body, RawExtras: anyMap(row["raw_extras"]),
		Source:    textOr(row, "source", "created"),
		CreatedAt: isoNaive(row["created_at"]), UpdatedAt: isoNaive(row["updated_at"]),
	}
}

func applyOpinionBodyFields(body map[string]any, values map[string]any) map[string]any {
	merged := map[string]any{}
	for k, v := range body {
		merged[k] = v
	}
	for _, field := range opinionBodyFields {
		value, present := values[field]
		if !present {
			continue
		}
		switch {
		case value == nil:
			delete(merged, field)
		case field == "trade":
			if b, ok := value.(bool); ok && b {
				merged[field] = "yes"
			} else {
				merged[field] = "no"
			}
		default:
			merged[field] = value
		}
	}
	return merged
}

func (h *handlers) opinionModifiers(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	const missing = "Opinion modifier not found"
	crud{
		missing: missing, unavailable: "store unavailable",
		list: func(p *project.Project, r *http.Request) (any, int, error) {
			rows, err := p.Rows(opinionModifiersTable)
			if err != nil {
				return nil, 0, err
			}
			sortRows(rows, byText("modifier_id"))
			tag := queryPtr(r, "country_tag")
			items := []opinionModifier{}
			for _, row := range rows {
				if tag != nil && *tag != "" && text(row, "country_tag") != strings.ToUpper(strings.TrimSpace(*tag)) {
					continue
				}
				items = append(items, opinionModifierFromRow(row))
			}
			return items, len(items), nil
		},
		create: createOpinionModifier,
		get: func(p *project.Project, id int64) (any, error) {
			row, err := getRow(p, opinionModifiersTable, id, missing)
			if err != nil {
				return nil, err
			}
			return opinionModifierFromRow(row), nil
		},
		update: updateOpinionModifier,
		remove: removeRow(opinionModifiersTable, missing),
	}.serve(p, w, r, rest)
}

func opinionConflict(modifierID string) error {
	return statusErr(http.StatusConflict, fmt.Sprintf(
		"Opinion modifier '%s' already exists. Pick a unique modifier id.", modifierID))
}

func createOpinionModifier(p *project.Project, r *http.Request) (any, error) {
	var in map[string]any
	if err := decodeBody(r, &in); err != nil {
		return nil, errInvalidPayload
	}
	modifierID := anyToString(in["modifier_id"])
	if modifierID == "" {
		return nil, statusErr(http.StatusUnprocessableEntity, "modifier_id is required")
	}
	row, err := writeRow(p, opinionModifiersTable, func(tx *project.Tx) (project.Row, error) {
		if taken(tx.Rows(opinionModifiersTable), 0, func(row project.Row) bool { return text(row, "modifier_id") == modifierID }) {
			return nil, opinionConflict(modifierID)
		}
		countryTag := anyToString(in["country_tag"])
		if countryTag == "" {
			countryTag = strings.ToUpper(text(tx.Record(), "country_tag"))
		}
		return insert(tx, opinionModifiersTable, project.Row{
			"country_tag": countryTag, "modifier_id": modifierID,
			"body":       jsonValue(applyOpinionBodyFields(map[string]any{}, in)),
			"raw_extras": map[string]any{}, "source": "created",
		})
	})
	if err != nil {
		return nil, err
	}
	return opinionModifierFromRow(row), nil
}

func updateOpinionModifier(p *project.Project, r *http.Request, id int64) (any, error) {
	var patch map[string]any
	if err := decodeBody(r, &patch); err != nil {
		return nil, errInvalidPayload
	}
	curRow, err := getRow(p, opinionModifiersTable, id, "Opinion modifier not found")
	if err != nil {
		return nil, err
	}
	cur := opinionModifierFromRow(curRow)
	modifierID := cur.ModifierID
	if next, present := patch["modifier_id"]; present {
		if nextID := anyToString(next); nextID != "" && nextID != cur.ModifierID {
			modifierID = nextID
		}
	}
	countryTag := cur.CountryTag
	if v, present := patch["country_tag"]; present && v != nil {
		countryTag = anyToString(v)
	}
	sets := project.Row{
		"modifier_id": modifierID, "country_tag": countryTag,
		"body": jsonValue(applyOpinionBodyFields(cur.Body, patch)), "source": "created",
	}
	row, err := updateRow(p, opinionModifiersTable, id, sets, func(tx *project.Tx) error {
		if modifierID != cur.ModifierID && taken(tx.Rows(opinionModifiersTable), id, func(row project.Row) bool {
			return text(row, "modifier_id") == modifierID
		}) {
			return opinionConflict(modifierID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return opinionModifierFromRow(row), nil
}

const modifierDefinitionsTable = "modifier_definitions"

var modifierIDRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const modifierIDMessage = "modifier_id must start with a letter or underscore and contain only letters, numbers, and underscores"

var (
	modifierColorTypes = map[string]bool{"good": true, "bad": true, "neutral": true}
	modifierValueTypes = map[string]bool{
		"number": true, "percentage": true, "percentage_in_hundred": true, "yes_no": true,
	}
	modifierPostfixes  = map[string]bool{"none": true, "days": true, "hours": true, "daily": true}
	modifierCategories = map[string]bool{
		"none": true, "all": true, "country": true, "state": true, "unit_leader": true,
		"army": true, "naval": true, "air": true, "peace": true, "politics": true,
		"ai": true, "defensive": true, "aggressive": true, "war_production": true,
		"military_advancements": true, "military_equipment": true, "autonomy": true,
		"government_in_exile": true, "intelligence_agency": true,
	}
)

type modifierDefinition struct {
	ID             int64    `json:"id"`
	ProjectID      int64    `json:"project_id"`
	ModifierID     string   `json:"modifier_id"`
	ColorType      string   `json:"color_type"`
	ValueType      string   `json:"value_type"`
	Precision      int      `json:"precision"`
	Postfix        string   `json:"postfix"`
	Categories     []string `json:"categories"`
	SourceFile     string   `json:"source_file"`
	ExplicitFields []string `json:"explicit_fields"`
	Source         string   `json:"source"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

func modifierDefinitionFromRow(row project.Row) modifierDefinition {
	item := modifierDefinition{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		ModifierID: text(row, "modifier_id"), ColorType: textOr(row, "color_type", "bad"),
		ValueType: textOr(row, "value_type", "number"), Precision: intOr(row, "precision", 2),
		Postfix: textOr(row, "postfix", "none"), SourceFile: text(row, "source_file"),
		Source:    textOr(row, "source", "created"),
		CreatedAt: isoNaive(row["created_at"]), UpdatedAt: isoNaive(row["updated_at"]),
	}
	decode(row["categories"], &item.Categories)
	decode(row["explicit_fields"], &item.ExplicitFields)
	if item.Categories == nil {
		item.Categories = []string{}
	}
	if item.ExplicitFields == nil {
		item.ExplicitFields = []string{}
	}
	return item
}

func validateModifierEnums(colorType, valueType, postfix *string, precision *int, categories *[]string) error {
	if colorType != nil && !modifierColorTypes[*colorType] {
		return fmt.Errorf("color_type must be one of good/bad/neutral")
	}
	if valueType != nil && !modifierValueTypes[*valueType] {
		return fmt.Errorf("value_type must be one of number/percentage/percentage_in_hundred/yes_no")
	}
	if postfix != nil && !modifierPostfixes[*postfix] {
		return fmt.Errorf("postfix must be one of none/days/hours/daily")
	}
	if precision != nil && (*precision < 0 || *precision > 3) {
		return fmt.Errorf("precision must be between 0 and 3")
	}
	if categories != nil {
		if len(*categories) == 0 {
			return fmt.Errorf("at least one category is required")
		}
		deduped := []string{}
		seen := map[string]bool{}
		for _, c := range *categories {
			if !modifierCategories[c] {
				return fmt.Errorf("unknown modifier category '%s'", c)
			}
			if !seen[c] {
				seen[c] = true
				deduped = append(deduped, c)
			}
		}
		*categories = deduped
	}
	return nil
}

type modifierDefinitionCreate struct {
	ModifierID string    `json:"modifier_id"`
	ColorType  *string   `json:"color_type"`
	ValueType  *string   `json:"value_type"`
	Precision  *int      `json:"precision"`
	Postfix    *string   `json:"postfix"`
	Categories *[]string `json:"categories"`
}

type modifierDefinitionPatch struct {
	ModifierID workbench.Optional[string]   `json:"modifier_id"`
	ColorType  workbench.Optional[string]   `json:"color_type"`
	ValueType  workbench.Optional[string]   `json:"value_type"`
	Precision  workbench.Optional[int]      `json:"precision"`
	Postfix    workbench.Optional[string]   `json:"postfix"`
	Categories workbench.Optional[[]string] `json:"categories"`
}

var errModifierDefinitionExists = statusErr(http.StatusConflict, "Modifier definition ID already exists")

func (h *handlers) modifierDefinitions(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	const missing = "Modifier definition not found"
	crud{
		listKey: "modifier_definitions", missing: missing, unavailable: "store unavailable",
		list: func(p *project.Project, _ *http.Request) (any, int, error) {
			items, err := listRows(p, modifierDefinitionsTable, byText("modifier_id"), modifierDefinitionFromRow)
			return items, len(items), err
		},
		create: createModifierDefinition,
		get: func(p *project.Project, id int64) (any, error) {
			row, err := getRow(p, modifierDefinitionsTable, id, missing)
			if err != nil {
				return nil, err
			}
			return modifierDefinitionFromRow(row), nil
		},
		update: updateModifierDefinition,
		remove: removeRow(modifierDefinitionsTable, missing),
	}.serve(p, w, r, rest)
}

func createModifierDefinition(p *project.Project, r *http.Request) (any, error) {
	var in modifierDefinitionCreate
	if err := decodeBody(r, &in); err != nil {
		return nil, errInvalidPayload
	}
	modifierID := strings.TrimSpace(in.ModifierID)
	if !modifierIDRE.MatchString(modifierID) {
		return nil, statusErr(http.StatusUnprocessableEntity, modifierIDMessage)
	}
	colorType, valueType, precision, postfix := "bad", "number", 2, "none"
	categories := []string{"all"}
	if in.ColorType != nil {
		colorType = *in.ColorType
	}
	if in.ValueType != nil {
		valueType = *in.ValueType
	}
	if in.Precision != nil {
		precision = *in.Precision
	}
	if in.Postfix != nil {
		postfix = *in.Postfix
	}
	if in.Categories != nil {
		categories = *in.Categories
	}
	if err := validateModifierEnums(&colorType, &valueType, &postfix, &precision, &categories); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	row, err := writeRow(p, modifierDefinitionsTable, func(tx *project.Tx) (project.Row, error) {
		if taken(tx.Rows(modifierDefinitionsTable), 0, func(row project.Row) bool { return text(row, "modifier_id") == modifierID }) {
			return nil, errModifierDefinitionExists
		}
		return insert(tx, modifierDefinitionsTable, project.Row{
			"modifier_id": modifierID, "color_type": colorType, "value_type": valueType,
			"precision": int64(precision), "postfix": postfix, "categories": jsonValue(categories),
			"source_file":     "",
			"explicit_fields": jsonValue([]string{"color_type", "value_type", "precision", "postfix", "category"}),
			"source":          "created",
		})
	})
	if err != nil {
		return nil, err
	}
	return modifierDefinitionFromRow(row), nil
}

func updateModifierDefinition(p *project.Project, r *http.Request, id int64) (any, error) {
	var patch modifierDefinitionPatch
	if err := decodeBody(r, &patch); err != nil {
		return nil, errInvalidPayload
	}
	curRow, err := getRow(p, modifierDefinitionsTable, id, "Modifier definition not found")
	if err != nil {
		return nil, err
	}
	cur := modifierDefinitionFromRow(curRow)
	if err := validateModifierEnums(patch.ColorType.Value, patch.ValueType.Value, patch.Postfix.Value,
		patch.Precision.Value, patch.Categories.Value); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	modifierID := cur.ModifierID
	if patch.ModifierID.Set && patch.ModifierID.Value != nil {
		candidate := strings.TrimSpace(*patch.ModifierID.Value)
		if !modifierIDRE.MatchString(candidate) {
			return nil, statusErr(http.StatusUnprocessableEntity, modifierIDMessage)
		}
		modifierID = candidate
	}

	touched := map[string]bool{}
	for _, field := range cur.ExplicitFields {
		if field == "category" {
			field = "categories"
		}
		touched[field] = true
	}
	for name, set := range map[string]bool{
		"modifier_id": patch.ModifierID.Set, "color_type": patch.ColorType.Set,
		"value_type": patch.ValueType.Set, "precision": patch.Precision.Set,
		"postfix": patch.Postfix.Set, "categories": patch.Categories.Set,
	} {
		if set {
			touched[name] = true
		}
	}
	explicitFields := []string{}
	for _, field := range []string{"color_type", "value_type", "precision", "postfix", "categories"} {
		if touched[field] {
			if field == "categories" {
				field = "category"
			}
			explicitFields = append(explicitFields, field)
		}
	}
	sets := project.Row{
		"modifier_id": modifierID, "explicit_fields": jsonValue(explicitFields), "source": "created",
	}
	setOpt(sets, "color_type", patch.ColorType)
	setOpt(sets, "value_type", patch.ValueType)
	setOpt(sets, "precision", patch.Precision)
	setOpt(sets, "postfix", patch.Postfix)
	setOptJSON(sets, "categories", patch.Categories)
	row, err := updateRow(p, modifierDefinitionsTable, id, sets, func(tx *project.Tx) error {
		if modifierID != cur.ModifierID && taken(tx.Rows(modifierDefinitionsTable), id, func(row project.Row) bool {
			return text(row, "modifier_id") == modifierID
		}) {
			return errModifierDefinitionExists
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return modifierDefinitionFromRow(row), nil
}
