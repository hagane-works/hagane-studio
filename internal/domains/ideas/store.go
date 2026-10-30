package ideas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
	"haganeworks/internal/schema"
	"haganeworks/internal/workbench"
)

const table = "ideas"

var (
	errIdeaMissing = apiutil.NotFoundError("Idea not found")

	errSourceMissing = apiutil.NotFoundError("Source idea not found")

	errStoreUnavailable = &apiutil.StatusError{Status: 503, Body: map[string]string{"detail": "idea store unavailable"}}
)

var designerIdeaTypes = map[string]bool{
	"tank_manufacturer": true, "naval_manufacturer": true, "aircraft_manufacturer": true,
	"materiel_manufacturer": true, "industrial_concern": true,
}

type Idea struct {
	ID                int64             `json:"id"`
	ProjectID         int64             `json:"project_id"`
	IdeaID            string            `json:"idea_id"`
	CountryTag        string            `json:"country_tag"`
	ScopeKind         string            `json:"scope_kind"`
	SharedCountryTags []string          `json:"shared_country_tags"`
	IdeaType          string            `json:"idea_type"`
	Name              string            `json:"name"`
	Picture           string            `json:"picture"`
	Modifier          map[string]any    `json:"modifier"`
	AIWillDo          string            `json:"ai_will_do"`
	RemovalCost       int               `json:"removal_cost"`
	Allowed           string            `json:"allowed"`
	AllowedCivilWar   string            `json:"allowed_civil_war"`
	IsDesigner        bool              `json:"is_designer"`
	ResearchBonus     map[string]any    `json:"research_bonus"`
	DesignerTraits    []string          `json:"designer_traits"`
	RawExtras         map[string]any    `json:"raw_extras"`
	NameEn            string            `json:"name_en"`
	NameZh            string            `json:"name_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	DescriptionLocs   map[string]string `json:"description_locs"`
	Available         string            `json:"available"`
	Visible           string            `json:"visible"`
	AllowedToRemove   string            `json:"allowed_to_remove"`
	Cancel            string            `json:"cancel"`
	DoEffect          string            `json:"do_effect"`
	OnAdd             string            `json:"on_add"`
	OnRemove          string            `json:"on_remove"`
	CancelIfInvalid   bool              `json:"cancel_if_invalid"`
	TargetedModifier  []map[string]any  `json:"targeted_modifier"`
	EquipmentBonus    map[string]any    `json:"equipment_bonus"`
	Rule              map[string]any    `json:"rule"`
	Level             int               `json:"level"`
	Ledger            string            `json:"ledger"`
	Cost              int               `json:"cost"`
	AutoAttachOnStart bool              `json:"auto_attach_on_start"`
	Source            string            `json:"source"`
	Warnings          []string          `json:"warnings"`
}

func ideaFromRow(row project.Row) Idea {
	it := Idea{
		ID:                project.Int64(row["id"]),
		ProjectID:         project.Int64(row["project_id"]),
		IdeaID:            text(row, "idea_id"),
		CountryTag:        text(row, "country_tag"),
		ScopeKind:         textOr(row, "scope_kind", "own"),
		IdeaType:          text(row, "idea_type"),
		Name:              text(row, "name"),
		Picture:           text(row, "picture"),
		AIWillDo:          text(row, "ai_will_do"),
		RemovalCost:       intOr(row, "removal_cost", -1),
		Allowed:           text(row, "allowed"),
		AllowedCivilWar:   text(row, "allowed_civil_war"),
		IsDesigner:        boolOr(row, "is_designer", false),
		NameEn:            text(row, "name_en"),
		NameZh:            text(row, "name_zh"),
		DescEn:            text(row, "desc_en"),
		DescZh:            text(row, "desc_zh"),
		Available:         text(row, "available"),
		Visible:           text(row, "visible"),
		AllowedToRemove:   text(row, "allowed_to_remove"),
		Cancel:            text(row, "cancel"),
		DoEffect:          text(row, "do_effect"),
		OnAdd:             text(row, "on_add"),
		OnRemove:          text(row, "on_remove"),
		CancelIfInvalid:   boolOr(row, "cancel_if_invalid", true),
		Level:             intOr(row, "level", 0),
		Ledger:            text(row, "ledger"),
		Cost:              intOr(row, "cost", 0),
		AutoAttachOnStart: boolOr(row, "auto_attach_on_start", true),
		Source:            textOr(row, "source", "created"),
	}
	decode(row["shared_country_tags"], &it.SharedCountryTags)
	decode(row["modifier"], &it.Modifier)
	decode(row["research_bonus"], &it.ResearchBonus)
	decode(row["designer_traits"], &it.DesignerTraits)
	decode(row["raw_extras"], &it.RawExtras)
	decode(row["display_name_locs"], &it.DisplayNameLocs)
	decode(row["description_locs"], &it.DescriptionLocs)
	decode(row["targeted_modifier"], &it.TargetedModifier)
	decode(row["equipment_bonus"], &it.EquipmentBonus)
	decode(row["rule"], &it.Rule)
	if it.Modifier == nil {
		it.Modifier = map[string]any{}
	}
	if it.ResearchBonus == nil {
		it.ResearchBonus = map[string]any{}
	}
	if it.DesignerTraits == nil {
		it.DesignerTraits = []string{}
	}
	if it.RawExtras == nil {
		it.RawExtras = map[string]any{}
	}
	if it.DisplayNameLocs == nil {
		it.DisplayNameLocs = map[string]string{}
	}
	if it.DescriptionLocs == nil {
		it.DescriptionLocs = map[string]string{}
	}
	if it.TargetedModifier == nil {
		it.TargetedModifier = []map[string]any{}
	}
	if it.EquipmentBonus == nil {
		it.EquipmentBonus = map[string]any{}
	}
	if it.Rule == nil {
		it.Rule = map[string]any{}
	}
	it.Warnings = []string{}
	return it
}

func text(row project.Row, column string) string {
	s, _ := row[column].(string)
	return s
}

func textOr(row project.Row, column, def string) string {
	if s, ok := row[column].(string); ok {
		return s
	}
	return def
}

func intOr(row project.Row, column string, def int) int {
	if row[column] == nil {
		return def
	}
	return int(project.Int64(row[column]))
}

func boolOr(row project.Row, column string, def bool) bool {
	if b, ok := row[column].(bool); ok {
		return b
	}
	return def
}

func decode(v any, dst any) {
	if v == nil {
		return
	}
	raw, err := json.Marshal(v)
	if err == nil {
		_ = json.Unmarshal(raw, dst)
	}
}

func jsonValue(v any) any {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out any
	if err := decoder.Decode(&out); err != nil {
		return nil
	}
	return out
}

func listIdeas(p *project.Project, ideaType, countryTag string) ([]Idea, error) {
	rows, err := p.Rows(table)
	if err != nil {
		return nil, err
	}
	out := []Idea{}
	for _, row := range rows {
		if ideaType != "" && text(row, "idea_type") != ideaType {
			continue
		}
		if countryTag != "" {
			if tag, ok := row["country_tag"].(string); !ok || tag != countryTag {
				continue
			}
		}
		out = append(out, ideaFromRow(row))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IdeaType != out[j].IdeaType {
			return out[i].IdeaType < out[j].IdeaType
		}
		return out[i].IdeaID < out[j].IdeaID
	})
	return out, nil
}

func getIdea(p *project.Project, id int64) (Idea, error) {
	rows, err := p.Rows(table)
	if err != nil {
		return Idea{}, err
	}
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			return ideaFromRow(row), nil
		}
	}
	return Idea{}, errIdeaMissing
}

func ideaNameOrIDRequiredDetail() map[string]any {
	return map[string]any{
		"error": "idea_name_or_id_required",
		"message": "Provide either name or idea_id — the program will not invent one. " +
			"Without a name the idea would be scripted as `new_idea_N` and show that raw token in game.",
	}
}

func ideaCreateScriptID(name, ideaID string) (string, error) {
	if trimmed := strings.TrimSpace(ideaID); trimmed != "" {
		return trimmed, nil
	}
	if strings.TrimSpace(name) == "" {
		return "", &apiutil.StatusError{Status: 422, Body: map[string]any{"detail": ideaNameOrIDRequiredDetail()}}
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.ToLower(name), " ", "_")), nil
}

type createInput struct {
	IdeaID            string            `json:"idea_id"`
	CountryTag        string            `json:"country_tag"`
	ScopeKind         *string           `json:"scope_kind"`
	SharedCountryTags []string          `json:"shared_country_tags"`
	IdeaType          *string           `json:"idea_type"`
	Name              *string           `json:"name"`
	Picture           string            `json:"picture"`
	Modifier          map[string]any    `json:"modifier"`
	AIWillDo          string            `json:"ai_will_do"`
	RemovalCost       *int              `json:"removal_cost"`
	Allowed           string            `json:"allowed"`
	AllowedCivilWar   string            `json:"allowed_civil_war"`
	IsDesigner        bool              `json:"is_designer"`
	ResearchBonus     map[string]any    `json:"research_bonus"`
	DesignerTraits    []string          `json:"designer_traits"`
	NameEn            string            `json:"name_en"`
	NameZh            string            `json:"name_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	DescriptionLocs   map[string]string `json:"description_locs"`
	Available         string            `json:"available"`
	Visible           string            `json:"visible"`
	AllowedToRemove   string            `json:"allowed_to_remove"`
	Cancel            string            `json:"cancel"`
	DoEffect          string            `json:"do_effect"`
	OnAdd             string            `json:"on_add"`
	OnRemove          string            `json:"on_remove"`
	CancelIfInvalid   *bool             `json:"cancel_if_invalid"`
	TargetedModifier  []map[string]any  `json:"targeted_modifier"`
	EquipmentBonus    map[string]any    `json:"equipment_bonus"`
	Rule              map[string]any    `json:"rule"`
	Level             int               `json:"level"`
	Ledger            string            `json:"ledger"`
	Cost              int               `json:"cost"`
	AutoAttachOnStart *bool             `json:"auto_attach_on_start"`
	Source            *string           `json:"source"`
}

func (in *createInput) normalize() {
	if in.Modifier == nil {
		in.Modifier = map[string]any{}
	}
	if in.ResearchBonus == nil {
		in.ResearchBonus = map[string]any{}
	}
	if in.DesignerTraits == nil {
		in.DesignerTraits = []string{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	if in.TargetedModifier == nil {
		in.TargetedModifier = []map[string]any{}
	}
	if in.EquipmentBonus == nil {
		in.EquipmentBonus = map[string]any{}
	}
	if in.Rule == nil {
		in.Rule = map[string]any{}
	}
}

func createIdea(p *project.Project, in createInput) (Idea, error) {
	if err := canonicalIdeaCreate(&in); err != nil {
		return Idea{}, err
	}
	name := workbench.StrDefault(in.Name, "")
	ideaType := workbench.StrDefault(in.IdeaType, "country")
	ideaID, err := ideaCreateScriptID(name, in.IdeaID)
	if err != nil {
		return Idea{}, err
	}
	isDesigner := in.IsDesigner || designerIdeaTypes[ideaType]
	countryTag := in.CountryTag
	if countryTag == "" && ideaType == "country" {
		if projectTag, ok := p.Record()["country_tag"].(string); ok {
			countryTag = projectTag
		}
	}
	removalCost := -1
	if in.RemovalCost != nil {
		removalCost = *in.RemovalCost
	}
	cancelIfInvalid := true
	if in.CancelIfInvalid != nil {
		cancelIfInvalid = *in.CancelIfInvalid
	}
	autoAttach := true
	if in.AutoAttachOnStart != nil {
		autoAttach = *in.AutoAttachOnStart
	}
	var shared any
	if in.SharedCountryTags != nil {
		shared = jsonValue(in.SharedCountryTags)
	}

	var created Idea
	err = p.Write([]string{table}, func(tx *project.Tx) error {
		taken := map[string]bool{}
		for _, row := range tx.Rows(table) {
			taken[text(row, "idea_id")] = true
		}
		original := ideaID
		for suffix := 1; taken[ideaID]; suffix++ {
			ideaID = fmt.Sprintf("%s_%d", original, suffix)
		}
		stored := tx.Insert(table, project.Row{
			"idea_id": ideaID, "country_tag": countryTag,
			"scope_kind": workbench.StrDefault(in.ScopeKind, "own"), "shared_country_tags": shared,
			"idea_type": ideaType, "name": name, "picture": in.Picture,
			"modifier": jsonValue(in.Modifier), "ai_will_do": in.AIWillDo,
			"removal_cost": int64(removalCost), "allowed": in.Allowed, "allowed_civil_war": in.AllowedCivilWar,
			"is_designer": isDesigner, "research_bonus": jsonValue(in.ResearchBonus),
			"designer_traits": jsonValue(in.DesignerTraits), "name_en": workbench.FirstNonBlank(in.NameEn, name),
			"name_zh": in.NameZh, "desc_en": in.DescEn, "desc_zh": in.DescZh,
			"display_name_locs": jsonValue(in.DisplayNameLocs), "description_locs": jsonValue(in.DescriptionLocs),
			"available": in.Available, "visible": in.Visible, "allowed_to_remove": in.AllowedToRemove,
			"cancel": in.Cancel, "do_effect": in.DoEffect, "on_add": in.OnAdd, "on_remove": in.OnRemove,
			"cancel_if_invalid": cancelIfInvalid, "targeted_modifier": jsonValue(in.TargetedModifier),
			"equipment_bonus": jsonValue(in.EquipmentBonus), "rule": jsonValue(in.Rule),
			"level": int64(in.Level), "ledger": in.Ledger, "cost": int64(in.Cost),
			"auto_attach_on_start": autoAttach, "source": workbench.StrDefault(in.Source, "created"),
			"raw_extras": map[string]any{}, "category_flags": nil, "source_snapshot": nil,
			"source_file": "", "has_obs": false,
		})
		created = ideaFromRow(stored)
		if ideaID != original {
			created.Warnings = []string{fmt.Sprintf("idea_id '%s' collided · auto-suffixed to '%s'.", original, ideaID)}
		}
		return nil
	})
	return created, err
}

type patchInput struct {
	IdeaID            workbench.Optional[string]            `json:"idea_id"`
	CountryTag        workbench.Optional[string]            `json:"country_tag"`
	ScopeKind         workbench.Optional[string]            `json:"scope_kind"`
	SharedCountryTags workbench.Optional[[]string]          `json:"shared_country_tags"`
	IdeaType          workbench.Optional[string]            `json:"idea_type"`
	Name              workbench.Optional[string]            `json:"name"`
	Picture           workbench.Optional[string]            `json:"picture"`
	Modifier          workbench.Optional[map[string]any]    `json:"modifier"`
	AIWillDo          workbench.Optional[string]            `json:"ai_will_do"`
	RemovalCost       workbench.Optional[int]               `json:"removal_cost"`
	Allowed           workbench.Optional[string]            `json:"allowed"`
	AllowedCivilWar   workbench.Optional[string]            `json:"allowed_civil_war"`
	IsDesigner        workbench.Optional[bool]              `json:"is_designer"`
	ResearchBonus     workbench.Optional[map[string]any]    `json:"research_bonus"`
	DesignerTraits    workbench.Optional[[]string]          `json:"designer_traits"`
	NameEn            workbench.Optional[string]            `json:"name_en"`
	NameZh            workbench.Optional[string]            `json:"name_zh"`
	DescEn            workbench.Optional[string]            `json:"desc_en"`
	DescZh            workbench.Optional[string]            `json:"desc_zh"`
	DisplayNameLocs   workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs   workbench.Optional[map[string]string] `json:"description_locs"`
	RawExtras         workbench.Optional[map[string]any]    `json:"raw_extras"`
	Available         workbench.Optional[string]            `json:"available"`
	Visible           workbench.Optional[string]            `json:"visible"`
	AllowedToRemove   workbench.Optional[string]            `json:"allowed_to_remove"`
	Cancel            workbench.Optional[string]            `json:"cancel"`
	DoEffect          workbench.Optional[string]            `json:"do_effect"`
	OnAdd             workbench.Optional[string]            `json:"on_add"`
	OnRemove          workbench.Optional[string]            `json:"on_remove"`
	CancelIfInvalid   workbench.Optional[bool]              `json:"cancel_if_invalid"`
	TargetedModifier  workbench.Optional[[]map[string]any]  `json:"targeted_modifier"`
	EquipmentBonus    workbench.Optional[map[string]any]    `json:"equipment_bonus"`
	Rule              workbench.Optional[map[string]any]    `json:"rule"`
	Level             workbench.Optional[int]               `json:"level"`
	Ledger            workbench.Optional[string]            `json:"ledger"`
	Cost              workbench.Optional[int]               `json:"cost"`
	AutoAttachOnStart workbench.Optional[bool]              `json:"auto_attach_on_start"`
	Source            workbench.Optional[string]            `json:"source"`
}

func optional[T any](o workbench.Optional[T]) any {
	if o.Value == nil {
		return nil
	}
	switch v := any(*o.Value).(type) {
	case string, bool:
		return v
	case int:
		return int64(v)
	default:
		return jsonValue(v)
	}
}

func updateIdea(p *project.Project, id int64, patch patchInput) (Idea, error) {
	if err := canonicalIdeaPatch(&patch); err != nil {
		return Idea{}, err
	}
	if patch.IdeaID.Set && patch.IdeaID.Value != nil {
		trimmed := strings.TrimSpace(*patch.IdeaID.Value)
		patch.IdeaID.Value = &trimmed
	}
	var updated Idea
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		current := tx.Find(table, id)
		if current == nil {
			return errIdeaMissing
		}
		changes := map[string]any{}
		set := func(column string, isSet bool, value any) {
			if isSet {
				changes[column] = value
			}
		}
		set("idea_id", patch.IdeaID.Set, optional(patch.IdeaID))
		set("country_tag", patch.CountryTag.Set, optional(patch.CountryTag))
		set("scope_kind", patch.ScopeKind.Set, optional(patch.ScopeKind))
		set("shared_country_tags", patch.SharedCountryTags.Set, optional(patch.SharedCountryTags))
		set("idea_type", patch.IdeaType.Set, optional(patch.IdeaType))
		set("name", patch.Name.Set, optional(patch.Name))
		set("picture", patch.Picture.Set, optional(patch.Picture))
		set("modifier", patch.Modifier.Set, optional(patch.Modifier))
		set("ai_will_do", patch.AIWillDo.Set, optional(patch.AIWillDo))
		set("removal_cost", patch.RemovalCost.Set, optional(patch.RemovalCost))
		set("allowed", patch.Allowed.Set, optional(patch.Allowed))
		set("allowed_civil_war", patch.AllowedCivilWar.Set, optional(patch.AllowedCivilWar))
		set("is_designer", patch.IsDesigner.Set, optional(patch.IsDesigner))
		set("research_bonus", patch.ResearchBonus.Set, optional(patch.ResearchBonus))
		set("designer_traits", patch.DesignerTraits.Set, optional(patch.DesignerTraits))
		set("name_en", patch.NameEn.Set, optional(patch.NameEn))
		set("name_zh", patch.NameZh.Set, optional(patch.NameZh))
		set("desc_en", patch.DescEn.Set, optional(patch.DescEn))
		set("desc_zh", patch.DescZh.Set, optional(patch.DescZh))
		set("display_name_locs", patch.DisplayNameLocs.Set, optional(patch.DisplayNameLocs))
		set("description_locs", patch.DescriptionLocs.Set, optional(patch.DescriptionLocs))
		set("raw_extras", patch.RawExtras.Set, optional(patch.RawExtras))
		set("available", patch.Available.Set, optional(patch.Available))
		set("visible", patch.Visible.Set, optional(patch.Visible))
		set("allowed_to_remove", patch.AllowedToRemove.Set, optional(patch.AllowedToRemove))
		set("cancel", patch.Cancel.Set, optional(patch.Cancel))
		set("do_effect", patch.DoEffect.Set, optional(patch.DoEffect))
		set("on_add", patch.OnAdd.Set, optional(patch.OnAdd))
		set("on_remove", patch.OnRemove.Set, optional(patch.OnRemove))
		set("cancel_if_invalid", patch.CancelIfInvalid.Set, optional(patch.CancelIfInvalid))
		set("targeted_modifier", patch.TargetedModifier.Set, optional(patch.TargetedModifier))
		set("equipment_bonus", patch.EquipmentBonus.Set, optional(patch.EquipmentBonus))
		set("rule", patch.Rule.Set, optional(patch.Rule))
		set("level", patch.Level.Set, optional(patch.Level))
		set("ledger", patch.Ledger.Set, optional(patch.Ledger))
		set("cost", patch.Cost.Set, optional(patch.Cost))
		set("auto_attach_on_start", patch.AutoAttachOnStart.Set, optional(patch.AutoAttachOnStart))

		effectiveType := text(current, "idea_type")
		if patch.IdeaType.Set && patch.IdeaType.Value != nil {
			effectiveType = *patch.IdeaType.Value
		}
		if designerIdeaTypes[effectiveType] {
			changes["is_designer"] = true
		}

		flipToCreated := text(current, "source") == "imported"
		if patch.Source.Set && patch.Source.Value != nil {
			flipToCreated = *patch.Source.Value == "imported"
		}
		if flipToCreated {
			changes["source"] = "created"
		} else if patch.Source.Set {
			changes["source"] = optional(patch.Source)
		}
		if len(changes) == 0 {
			updated = ideaFromRow(current)
			return nil
		}
		if err := checkConstraints(tx, id, changes); err != nil {
			return err
		}
		row, err := tx.Update(table, id, func(row project.Row) {
			for column, value := range changes {
				row[column] = value
			}
		})
		if err != nil {
			return err
		}
		updated = ideaFromRow(row)
		return nil
	})
	return updated, err
}

func checkConstraints(tx *project.Tx, id int64, changes map[string]any) error {
	for _, column := range schema.Columns(table) {
		if value, ok := changes[column.Name]; ok && value == nil && column.NotNull {
			return errStoreUnavailable
		}
	}
	if ideaID, ok := changes["idea_id"].(string); ok {
		for _, row := range tx.Rows(table) {
			if project.Int64(row["id"]) != id && text(row, "idea_id") == ideaID {
				return errStoreUnavailable
			}
		}
	}
	return nil
}

func deleteIdea(p *project.Project, id int64) error {
	return p.Write([]string{table}, func(tx *project.Tx) error {
		if tx.Delete(table, func(row project.Row) bool { return project.Int64(row["id"]) == id }) == 0 {
			return errIdeaMissing
		}
		return nil
	})
}

func forkIdea(p *project.Project, id int64, newIdeaID string) (Idea, error) {
	var forked Idea
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		source := tx.Find(table, id)
		if source == nil {
			return errSourceMissing
		}
		for _, row := range tx.Rows(table) {
			if text(row, "idea_id") == newIdeaID {
				return apiutil.ConflictError(fmt.Sprintf("idea_id '%s' already exists in this project", newIdeaID))
			}
		}
		copied := project.Row{}
		for column, value := range source {
			switch column {
			case "id", "created_at", "updated_at", "source", "idea_id":
			default:
				copied[column] = value
			}
		}
		copied["idea_id"] = newIdeaID
		copied["source"] = "created"
		forked = ideaFromRow(tx.Insert(table, copied))
		return nil
	})
	return forked, err
}
