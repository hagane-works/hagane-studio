package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const (
	aiStrategyPlansTable = "ai_strategy_plans"
	aiTemplatesTable     = "ai_templates"
	aiUnavailable        = "AI planning store unavailable"
)

var aiPlanBlockFields = []string{
	"allowed", "enable", "abort", "ai_national_focuses", "focus_factors",
	"research", "ideas", "traits", "ai_strategy", "weight",
}

var aiTemplateBlockFields = []string{"blocked_for", "available_for", "upgrade_prio", "enable"}

func aiValidateScriptID(value, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s must be one safe HOI4 script token", label)
	}
	for _, r := range value {
		if unicode.IsSpace(r) || strings.ContainsRune("{}=#\"'", r) {
			return "", fmt.Errorf("%s must be one safe HOI4 script token", label)
		}
	}
	return value, nil
}

func aiValidateBlock(value, label string) (string, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return "", fmt.Errorf("%s must be one brace block", label)
	}
	block, recoveries, err := paradox.Parse("__hagane_value = " + value)
	if err != nil {
		return "", fmt.Errorf("%s must be one parseable brace block", label)
	}

	for _, recovery := range recoveries {
		if recovery.Code == paradox.RecoveryUnclosedBlock {
			return "", fmt.Errorf("%s must be one parseable brace block", label)
		}
	}
	if len(block.Entries) != 1 {
		return "", fmt.Errorf("%s must be one parseable brace block", label)
	}
	entry := block.Entries[0]
	if entry.Key == nil || *entry.Key != "__hagane_value" || entry.Block == nil {
		return "", fmt.Errorf("%s must be one parseable brace block", label)
	}
	return value, nil
}

func aiHasExactOuterBlock(value string) bool {
	if !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return false
	}
	depth := 0
	quoted, escaped, comment := false, false, false
	for i := 0; i < len(value); i++ {
		char := value[i]
		if comment {
			if char == '\n' {
				comment = false
			}
			continue
		}
		if quoted {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
			} else if char == '"' {
				quoted = false
			}
			continue
		}
		switch char {
		case '#':
			comment = true
		case '"':
			quoted = true
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 || depth == 0 && i != len(value)-1 {
				return false
			}
		}
	}
	return !quoted && depth == 0
}

func aiNormalizeLegacyBlock(value, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	normalized, err := aiValidateBlock(value, label)
	if err == nil {
		return normalized, nil
	}
	if strings.HasPrefix(value, "{") {

		return "", err
	}
	candidate := "{\n" + value + "\n}"
	if !aiHasExactOuterBlock(candidate) {
		return "", err
	}
	normalized, repairErr := aiValidateBlock(candidate, label)
	if repairErr != nil {
		return "", err
	}
	return normalized, nil
}

func aiNormalizeRequiredLegacyBlock(value, label string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return aiValidateBlock("{\n}", label)
	}
	return aiNormalizeLegacyBlock(value, label)
}

func aiValidatePlanOrder(values []string) error {
	allowed := map[string]bool{"name": true, "desc": true}
	for _, field := range aiPlanBlockFields {
		allowed[field] = true
	}
	for _, value := range values {
		if !allowed[value] {
			return fmt.Errorf("field_order contains an unknown plan field")
		}
	}
	return nil
}

func aiValidateTemplateOrder(values []string) error {
	allowed := map[string]bool{"role": true, "front_role_override": true}
	for _, field := range aiTemplateBlockFields {
		allowed[field] = true
	}
	for _, value := range values {
		if allowed[value] {
			continue
		}
		if strings.HasPrefix(value, "target:") {
			if _, err := aiValidateScriptID(strings.TrimPrefix(value, "target:"), "field_order target id"); err == nil {
				continue
			}
		}
		return fmt.Errorf("field_order contains an unknown template field")
	}
	return nil
}

func aiObject(data []byte, allowed map[string]bool) (map[string]json.RawMessage, error) {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, fmt.Errorf("payload must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("payload must be an object")
	}
	for key, raw := range fields {
		if !allowed[key] {
			return nil, fmt.Errorf("unknown field %q", key)
		}
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil, fmt.Errorf("%s cannot be null", key)
		}
	}
	return fields, nil
}

func decodeAIPayload(r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("payload must contain one JSON value")
		}
		return err
	}
	return nil
}

type aiTargetTemplate struct {
	ID   string `json:"id"`
	Body string `json:"body"`
}

func (t *aiTargetTemplate) UnmarshalJSON(data []byte) error {
	if _, err := aiObject(data, map[string]bool{"id": true, "body": true}); err != nil {
		return err
	}
	type plain aiTargetTemplate
	return json.Unmarshal(data, (*plain)(t))
}

type aiStrategyPlan struct {
	ID                int64    `json:"id"`
	ProjectID         int64    `json:"project_id"`
	PlanID            string   `json:"plan_id"`
	Name              string   `json:"name"`
	Desc              string   `json:"desc"`
	Allowed           []string `json:"allowed"`
	Enable            []string `json:"enable"`
	Abort             []string `json:"abort"`
	AINationalFocuses []string `json:"ai_national_focuses"`
	FocusFactors      []string `json:"focus_factors"`
	Research          []string `json:"research"`
	Ideas             []string `json:"ideas"`
	Traits            []string `json:"traits"`
	AIStrategy        []string `json:"ai_strategy"`
	Weight            []string `json:"weight"`
	FieldOrder        []string `json:"field_order"`
	RawBlock          string   `json:"raw_block"`
	SourceFile        string   `json:"source_file"`
	Source            string   `json:"source"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

type aiStrategyPlanCreate struct {
	PlanID            string   `json:"plan_id"`
	Name              string   `json:"name"`
	Desc              string   `json:"desc"`
	Allowed           []string `json:"allowed"`
	Enable            []string `json:"enable"`
	Abort             []string `json:"abort"`
	AINationalFocuses []string `json:"ai_national_focuses"`
	FocusFactors      []string `json:"focus_factors"`
	Research          []string `json:"research"`
	Ideas             []string `json:"ideas"`
	Traits            []string `json:"traits"`
	AIStrategy        []string `json:"ai_strategy"`
	Weight            []string `json:"weight"`
	FieldOrder        []string `json:"field_order"`
}

func newAIStrategyPlanCreate() aiStrategyPlanCreate {
	return aiStrategyPlanCreate{
		Allowed: []string{"{\n}"}, Enable: []string{"{\n}"}, Abort: []string{"{\n}"},
		AINationalFocuses: []string{}, FocusFactors: []string{}, Research: []string{},
		Ideas: []string{}, Traits: []string{}, AIStrategy: []string{},
		Weight:     []string{"{\n\tfactor = 1\n}"},
		FieldOrder: []string{"name", "desc", "allowed", "enable", "abort", "weight"},
	}
}

func (in *aiStrategyPlanCreate) UnmarshalJSON(data []byte) error {
	allowed := map[string]bool{"plan_id": true, "name": true, "desc": true, "field_order": true}
	for _, field := range aiPlanBlockFields {
		allowed[field] = true
	}
	if _, err := aiObject(data, allowed); err != nil {
		return err
	}
	type plain aiStrategyPlanCreate
	value := newAIStrategyPlanCreate()
	if err := json.Unmarshal(data, (*plain)(&value)); err != nil {
		return err
	}
	*in = value
	return nil
}

func (in *aiStrategyPlanCreate) blockLists() []*[]string {
	return []*[]string{&in.Allowed, &in.Enable, &in.Abort, &in.AINationalFocuses, &in.FocusFactors,
		&in.Research, &in.Ideas, &in.Traits, &in.AIStrategy, &in.Weight}
}

func aiNormalizeBlockList(name string, values []string) error {
	for i, raw := range values {
		value, err := aiValidateBlock(raw, name)
		if err != nil {
			return err
		}
		values[i] = value
	}
	return nil
}

func (in *aiStrategyPlanCreate) normalize() error {
	var err error
	if in.PlanID, err = aiValidateScriptID(in.PlanID, "plan_id"); err != nil {
		return err
	}
	for i, list := range in.blockLists() {
		if err := aiNormalizeBlockList(aiPlanBlockFields[i], *list); err != nil {
			return err
		}
	}
	return aiValidatePlanOrder(in.FieldOrder)
}

type aiStrategyPlanPatch struct {
	PlanID            workbench.Optional[string]   `json:"plan_id"`
	Name              workbench.Optional[string]   `json:"name"`
	Desc              workbench.Optional[string]   `json:"desc"`
	Allowed           workbench.Optional[[]string] `json:"allowed"`
	Enable            workbench.Optional[[]string] `json:"enable"`
	Abort             workbench.Optional[[]string] `json:"abort"`
	AINationalFocuses workbench.Optional[[]string] `json:"ai_national_focuses"`
	FocusFactors      workbench.Optional[[]string] `json:"focus_factors"`
	Research          workbench.Optional[[]string] `json:"research"`
	Ideas             workbench.Optional[[]string] `json:"ideas"`
	Traits            workbench.Optional[[]string] `json:"traits"`
	AIStrategy        workbench.Optional[[]string] `json:"ai_strategy"`
	Weight            workbench.Optional[[]string] `json:"weight"`
	FieldOrder        workbench.Optional[[]string] `json:"field_order"`
}

func (p *aiStrategyPlanPatch) blockLists() []*workbench.Optional[[]string] {
	return []*workbench.Optional[[]string]{&p.Allowed, &p.Enable, &p.Abort, &p.AINationalFocuses,
		&p.FocusFactors, &p.Research, &p.Ideas, &p.Traits, &p.AIStrategy, &p.Weight}
}

func (p *aiStrategyPlanPatch) normalize() error {
	const nullError = "AI strategy plan update fields cannot be null"
	for _, value := range []workbench.Optional[string]{p.PlanID, p.Name, p.Desc} {
		if value.Set && value.Value == nil {
			return errors.New(nullError)
		}
	}
	for _, value := range append(p.blockLists(), &p.FieldOrder) {
		if value.Set && value.Value == nil {
			return errors.New(nullError)
		}
	}
	if p.PlanID.Set {
		value, err := aiValidateScriptID(*p.PlanID.Value, "plan_id")
		if err != nil {
			return err
		}
		p.PlanID.Value = &value
	}
	for i, list := range p.blockLists() {
		if !list.Set {
			continue
		}
		values := append([]string(nil), *list.Value...)
		if err := aiNormalizeBlockList(aiPlanBlockFields[i], values); err != nil {
			return err
		}
		list.Value = &values
	}
	if p.FieldOrder.Set {
		return aiValidatePlanOrder(*p.FieldOrder.Value)
	}
	return nil
}

func aiStringList(row project.Row, column string) ([]string, error) {
	raw, err := json.Marshal(row[column])
	if err != nil {
		return nil, err
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	if values == nil {
		return nil, fmt.Errorf("%s is not a list", column)
	}
	return values, nil
}

func aiStrategyPlanFromRow(row project.Row) (aiStrategyPlan, error) {
	item := aiStrategyPlan{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		PlanID: text(row, "plan_id"), Name: text(row, "name"), Desc: text(row, "desc"),
		RawBlock: text(row, "raw_block"), SourceFile: text(row, "source_file"), Source: text(row, "source"),
		CreatedAt: isoTime(row["created_at"]), UpdatedAt: isoTime(row["updated_at"]),
	}
	lists := []*[]string{&item.Allowed, &item.Enable, &item.Abort, &item.AINationalFocuses,
		&item.FocusFactors, &item.Research, &item.Ideas, &item.Traits, &item.AIStrategy, &item.Weight}
	for i, list := range lists {
		values, err := aiStringList(row, aiPlanBlockFields[i])
		if err != nil {
			return item, err
		}
		for j, raw := range values {
			if values[j], err = aiNormalizeRequiredLegacyBlock(raw, aiPlanBlockFields[i]); err != nil {
				return item, err
			}
		}
		*list = values
	}
	order, err := aiStringList(row, "field_order")
	if err != nil {
		return item, err
	}
	if err := aiValidatePlanOrder(order); err != nil {
		return item, err
	}
	item.FieldOrder = order
	return item, nil
}

var errPlanExists = statusErr(http.StatusConflict, "AI strategy plan ID already exists")

func (h *handlers) aiStrategyPlans(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	const missing = "AI strategy plan not found"
	crud{
		listKey: "ai_strategy_plans", missing: missing, unavailable: aiUnavailable,
		list: func(p *project.Project, _ *http.Request) (any, int, error) {
			rows, err := p.Rows(aiStrategyPlansTable)
			if err != nil {
				return nil, 0, err
			}
			sortRows(rows, byText("plan_id"))
			items := []aiStrategyPlan{}
			for _, row := range rows {
				item, err := aiStrategyPlanFromRow(row)
				if err != nil {
					return nil, 0, err
				}
				items = append(items, item)
			}
			return items, len(items), nil
		},
		create: createAIStrategyPlan,
		update: updateAIStrategyPlan,
		remove: removeRow(aiStrategyPlansTable, missing),
	}.serve(p, w, r, rest)
}

func createAIStrategyPlan(p *project.Project, r *http.Request) (any, error) {
	var in aiStrategyPlanCreate
	if err := decodeAIPayload(r, &in); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	if err := in.normalize(); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	row, err := writeRow(p, aiStrategyPlansTable, func(tx *project.Tx) (project.Row, error) {
		if taken(tx.Rows(aiStrategyPlansTable), 0, func(row project.Row) bool { return text(row, "plan_id") == in.PlanID }) {
			return nil, errPlanExists
		}
		values := project.Row{
			"plan_id": in.PlanID, "name": in.Name, "desc": in.Desc,
			"field_order": jsonValue(in.FieldOrder),
			"raw_block":   "", "source_file": "", "source": "created",
		}
		for i, list := range in.blockLists() {
			values[aiPlanBlockFields[i]] = jsonValue(*list)
		}
		return insert(tx, aiStrategyPlansTable, values)
	})
	if err != nil {
		return nil, err
	}
	return aiStrategyPlanFromRow(row)
}

func updateAIStrategyPlan(p *project.Project, r *http.Request, id int64) (any, error) {
	var patch aiStrategyPlanPatch
	if err := decodeAIPayload(r, &patch); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	if err := patch.normalize(); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	cur, err := getRow(p, aiStrategyPlansTable, id, "AI strategy plan not found")
	if err != nil {
		return nil, err
	}
	sets := project.Row{}
	setOpt(sets, "plan_id", patch.PlanID)
	setOpt(sets, "name", patch.Name)
	setOpt(sets, "desc", patch.Desc)
	for i, list := range patch.blockLists() {
		setOptJSON(sets, aiPlanBlockFields[i], *list)
	}
	setOptJSON(sets, "field_order", patch.FieldOrder)
	if text(cur, "source") != "created" {
		sets["source"] = "created"
	}
	if text(cur, "raw_block") != "" {
		sets["raw_block"] = ""
	}
	row, err := updateRow(p, aiStrategyPlansTable, id, sets, func(tx *project.Tx) error {
		if patch.PlanID.Set && taken(tx.Rows(aiStrategyPlansTable), id, func(row project.Row) bool {
			return text(row, "plan_id") == *patch.PlanID.Value
		}) {
			return errPlanExists
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return aiStrategyPlanFromRow(row)
}

type aiTemplate struct {
	ID                int64              `json:"id"`
	ProjectID         int64              `json:"project_id"`
	TemplateID        string             `json:"template_id"`
	Role              string             `json:"role"`
	BlockedFor        string             `json:"blocked_for"`
	AvailableFor      string             `json:"available_for"`
	UpgradePrio       string             `json:"upgrade_prio"`
	Enable            string             `json:"enable"`
	FrontRoleOverride string             `json:"front_role_override"`
	TargetTemplates   []aiTargetTemplate `json:"target_templates"`
	FieldOrder        []string           `json:"field_order"`
	RawBlock          string             `json:"raw_block"`
	SourceFile        string             `json:"source_file"`
	Source            string             `json:"source"`
	CreatedAt         string             `json:"created_at"`
	UpdatedAt         string             `json:"updated_at"`
}

type aiTemplateCreate struct {
	TemplateID        string             `json:"template_id"`
	Role              string             `json:"role"`
	BlockedFor        string             `json:"blocked_for"`
	AvailableFor      string             `json:"available_for"`
	UpgradePrio       string             `json:"upgrade_prio"`
	Enable            string             `json:"enable"`
	FrontRoleOverride string             `json:"front_role_override"`
	TargetTemplates   []aiTargetTemplate `json:"target_templates"`
	FieldOrder        []string           `json:"field_order"`
}

func newAITemplateCreate() aiTemplateCreate {
	return aiTemplateCreate{
		Role: "infantry", UpgradePrio: "{\n\tfactor = 1\n}",
		TargetTemplates: []aiTargetTemplate{{
			ID:   "infantry_default",
			Body: "{\n\tupgrade_prio = {\n\t\tfactor = 1\n\t}\n\ttarget_template = {\n\t\tsupport = {\n\t\t}\n\t\tregiments = {\n\t\t\tinfantry = 6\n\t\t}\n\t}\n}",
		}},
		FieldOrder: []string{"role", "upgrade_prio", "target:infantry_default"},
	}
}

func (in *aiTemplateCreate) UnmarshalJSON(data []byte) error {
	allowed := map[string]bool{
		"template_id": true, "role": true, "blocked_for": true, "available_for": true,
		"upgrade_prio": true, "enable": true, "front_role_override": true,
		"target_templates": true, "field_order": true,
	}
	if _, err := aiObject(data, allowed); err != nil {
		return err
	}
	type plain aiTemplateCreate
	value := newAITemplateCreate()
	if err := json.Unmarshal(data, (*plain)(&value)); err != nil {
		return err
	}
	*in = value
	return nil
}

func aiNormalizeTargetsWith(values []aiTargetTemplate, legacyRead bool) ([]aiTargetTemplate, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one target template is required")
	}
	seen := map[string]bool{}
	for i := range values {
		var err error
		if values[i].ID, err = aiValidateScriptID(values[i].ID, "target template id"); err != nil {
			return nil, err
		}
		if legacyRead {
			values[i].Body, err = aiNormalizeRequiredLegacyBlock(values[i].Body, "target template body")
		} else {
			values[i].Body, err = aiValidateBlock(values[i].Body, "target template body")
		}
		if err != nil {
			return nil, err
		}

		if seen[values[i].ID] && !legacyRead {
			return nil, fmt.Errorf("target template ids must be unique within a role")
		}
		seen[values[i].ID] = true
	}
	return values, nil
}

func aiNormalizeTargets(values []aiTargetTemplate) ([]aiTargetTemplate, error) {
	return aiNormalizeTargetsWith(values, false)
}

func aiNormalizeLegacyTargets(values []aiTargetTemplate) ([]aiTargetTemplate, error) {
	return aiNormalizeTargetsWith(values, true)
}

func (in *aiTemplateCreate) normalize() error {
	var err error
	if in.TemplateID, err = aiValidateScriptID(in.TemplateID, "template_id"); err != nil {
		return err
	}
	if in.Role, err = aiValidateScriptID(in.Role, "role"); err != nil {
		return err
	}
	if strings.TrimSpace(in.FrontRoleOverride) == "" {
		in.FrontRoleOverride = ""
	} else if in.FrontRoleOverride, err = aiValidateScriptID(in.FrontRoleOverride, "front_role_override"); err != nil {
		return err
	}
	for i, value := range []*string{&in.BlockedFor, &in.AvailableFor, &in.UpgradePrio, &in.Enable} {
		if strings.TrimSpace(*value) == "" {
			*value = ""
			continue
		}
		if *value, err = aiValidateBlock(*value, aiTemplateBlockFields[i]); err != nil {
			return err
		}
	}
	if in.TargetTemplates, err = aiNormalizeTargets(in.TargetTemplates); err != nil {
		return err
	}
	return aiValidateTemplateOrder(in.FieldOrder)
}

type aiTemplatePatch struct {
	TemplateID        workbench.Optional[string]             `json:"template_id"`
	Role              workbench.Optional[string]             `json:"role"`
	BlockedFor        workbench.Optional[string]             `json:"blocked_for"`
	AvailableFor      workbench.Optional[string]             `json:"available_for"`
	UpgradePrio       workbench.Optional[string]             `json:"upgrade_prio"`
	Enable            workbench.Optional[string]             `json:"enable"`
	FrontRoleOverride workbench.Optional[string]             `json:"front_role_override"`
	TargetTemplates   workbench.Optional[[]aiTargetTemplate] `json:"target_templates"`
	FieldOrder        workbench.Optional[[]string]           `json:"field_order"`
}

func (p *aiTemplatePatch) blocks() []*workbench.Optional[string] {
	return []*workbench.Optional[string]{&p.BlockedFor, &p.AvailableFor, &p.UpgradePrio, &p.Enable}
}

func (p *aiTemplatePatch) normalize() error {
	const nullError = "AI template update fields cannot be null"
	for _, value := range append(p.blocks(), &p.TemplateID, &p.Role, &p.FrontRoleOverride) {
		if value.Set && value.Value == nil {
			return errors.New(nullError)
		}
	}
	if p.TargetTemplates.Set && p.TargetTemplates.Value == nil || p.FieldOrder.Set && p.FieldOrder.Value == nil {
		return errors.New(nullError)
	}
	for _, field := range []struct {
		name string
		opt  *workbench.Optional[string]
	}{{"template_id", &p.TemplateID}, {"role", &p.Role}} {
		if field.opt.Set {
			value, err := aiValidateScriptID(*field.opt.Value, field.name)
			if err != nil {
				return err
			}
			field.opt.Value = &value
		}
	}
	if p.FrontRoleOverride.Set {
		value := strings.TrimSpace(*p.FrontRoleOverride.Value)
		if value != "" {
			var err error
			if value, err = aiValidateScriptID(value, "front_role_override"); err != nil {
				return err
			}
		}
		p.FrontRoleOverride.Value = &value
	}
	for i, block := range p.blocks() {
		if !block.Set {
			continue
		}
		value := strings.TrimSpace(*block.Value)
		if value != "" {
			var err error
			if value, err = aiValidateBlock(value, aiTemplateBlockFields[i]); err != nil {
				return err
			}
		}
		block.Value = &value
	}
	if p.TargetTemplates.Set {
		values, err := aiNormalizeTargets(*p.TargetTemplates.Value)
		if err != nil {
			return err
		}
		p.TargetTemplates.Value = &values
	}
	if p.FieldOrder.Set {
		return aiValidateTemplateOrder(*p.FieldOrder.Value)
	}
	return nil
}

func aiTemplateFromRow(row project.Row) (aiTemplate, error) {
	item := aiTemplate{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		TemplateID: text(row, "template_id"), Role: text(row, "role"),
		BlockedFor: text(row, "blocked_for"), AvailableFor: text(row, "available_for"),
		UpgradePrio: text(row, "upgrade_prio"), Enable: text(row, "enable"),
		FrontRoleOverride: text(row, "front_role_override"), RawBlock: text(row, "raw_block"),
		SourceFile: text(row, "source_file"), Source: text(row, "source"),
		CreatedAt: isoTime(row["created_at"]), UpdatedAt: isoTime(row["updated_at"]),
	}
	var err error
	if item.Role, err = aiValidateScriptID(item.Role, "role"); err != nil {
		return item, err
	}
	if strings.TrimSpace(item.FrontRoleOverride) == "" {
		item.FrontRoleOverride = ""
	} else if item.FrontRoleOverride, err = aiValidateScriptID(item.FrontRoleOverride, "front_role_override"); err != nil {
		return item, err
	}
	for i, value := range []*string{&item.BlockedFor, &item.AvailableFor, &item.UpgradePrio, &item.Enable} {
		if *value, err = aiNormalizeLegacyBlock(*value, aiTemplateBlockFields[i]); err != nil {
			return item, err
		}
	}
	raw, err := json.Marshal(row["target_templates"])
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(raw, &item.TargetTemplates); err != nil {
		return item, err
	}
	if item.TargetTemplates == nil {
		return item, fmt.Errorf("target_templates is not a list")
	}
	if item.TargetTemplates, err = aiNormalizeLegacyTargets(item.TargetTemplates); err != nil {
		return item, err
	}
	if item.FieldOrder, err = aiStringList(row, "field_order"); err != nil {
		return item, err
	}
	item.FieldOrder = aiLegacyTargetMarkers(item.FieldOrder, item.TargetTemplates)
	if err := aiValidateTemplateOrder(item.FieldOrder); err != nil {
		return item, err
	}
	return item, nil
}

func aiLegacyTargetMarkers(order []string, targets []aiTargetTemplate) []string {
	ids := make(map[string]bool, len(targets))
	for _, target := range targets {
		ids[target.ID] = true
	}
	for i, value := range order {
		if ids[value] && !slices.Contains(aiTemplateBlockFields, value) && value != "role" && value != "front_role_override" {
			order[i] = "target:" + value
		}
	}
	return order
}

var errTemplateExists = statusErr(http.StatusConflict, "AI template ID already exists")

func (h *handlers) aiTemplates(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	const missing = "AI template not found"
	crud{
		listKey: "ai_templates", missing: missing, unavailable: aiUnavailable,
		list: func(p *project.Project, _ *http.Request) (any, int, error) {
			rows, err := p.Rows(aiTemplatesTable)
			if err != nil {
				return nil, 0, err
			}
			sortRows(rows, byText("template_id"))
			items := []aiTemplate{}
			for _, row := range rows {
				item, err := aiTemplateFromRow(row)
				if err != nil {
					return nil, 0, err
				}
				items = append(items, item)
			}
			return items, len(items), nil
		},
		create: createAITemplate,
		update: updateAITemplate,
		remove: removeRow(aiTemplatesTable, missing),
	}.serve(p, w, r, rest)
}

func createAITemplate(p *project.Project, r *http.Request) (any, error) {
	var in aiTemplateCreate
	if err := decodeAIPayload(r, &in); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	if err := in.normalize(); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	row, err := writeRow(p, aiTemplatesTable, func(tx *project.Tx) (project.Row, error) {
		if taken(tx.Rows(aiTemplatesTable), 0, func(row project.Row) bool { return text(row, "template_id") == in.TemplateID }) {
			return nil, errTemplateExists
		}
		return insert(tx, aiTemplatesTable, project.Row{
			"template_id": in.TemplateID, "role": in.Role,
			"blocked_for": in.BlockedFor, "available_for": in.AvailableFor,
			"upgrade_prio": in.UpgradePrio, "enable": in.Enable,
			"front_role_override": in.FrontRoleOverride,
			"target_templates":    jsonValue(in.TargetTemplates),
			"field_order":         jsonValue(in.FieldOrder),
			"raw_block":           "", "source_file": "", "source": "created",
		})
	})
	if err != nil {
		return nil, err
	}
	return aiTemplateFromRow(row)
}

func updateAITemplate(p *project.Project, r *http.Request, id int64) (any, error) {
	var patch aiTemplatePatch
	if err := decodeAIPayload(r, &patch); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	if err := patch.normalize(); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	cur, err := getRow(p, aiTemplatesTable, id, "AI template not found")
	if err != nil {
		return nil, err
	}
	sets := project.Row{}
	setOpt(sets, "template_id", patch.TemplateID)
	setOpt(sets, "role", patch.Role)
	for i, block := range patch.blocks() {
		setOpt(sets, aiTemplateBlockFields[i], *block)
	}
	setOpt(sets, "front_role_override", patch.FrontRoleOverride)
	setOptJSON(sets, "target_templates", patch.TargetTemplates)
	setOptJSON(sets, "field_order", patch.FieldOrder)
	if text(cur, "source") != "created" {
		sets["source"] = "created"
	}
	if text(cur, "raw_block") != "" {
		sets["raw_block"] = ""
	}
	row, err := updateRow(p, aiTemplatesTable, id, sets, func(tx *project.Tx) error {
		if patch.TemplateID.Set && taken(tx.Rows(aiTemplatesTable), id, func(row project.Row) bool {
			return text(row, "template_id") == *patch.TemplateID.Value
		}) {
			return errTemplateExists
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return aiTemplateFromRow(row)
}
