package characters

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	emitcharacters "haganeworks/internal/hoi4/emit/characters"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const table = "characters"

func Register(mux *http.ServeMux, deps domains.Deps) {
	h := func(fn func(*project.Project, http.ResponseWriter, *http.Request) error) http.HandlerFunc {
		return apiutil.Handler(deps.Open, fn)
	}
	const chars = "/api/go/projects/{id}/characters"
	mux.HandleFunc("GET "+chars, h(listCharacters))
	mux.HandleFunc("POST "+chars, h(createCharacter))
	mux.HandleFunc("GET "+chars+"/{cid}", h(getCharacter))
	mux.HandleFunc("PUT "+chars+"/{cid}", h(updateCharacter))
	mux.HandleFunc("DELETE "+chars+"/{cid}", h(deleteCharacter))
	mux.HandleFunc("POST "+chars+"/{cid}/fork", h(forkCharacter))
	mux.HandleFunc("GET "+chars+"/{cid}/preview", h(characterPreview))
	mux.HandleFunc("GET "+chars+"/{cid}/preview-files", h(characterPreviewFiles))
	mux.HandleFunc("POST "+chars+"/{cid}/parse-preview", h(characterParsePreview))
	mux.HandleFunc("POST "+chars+"/{cid}/upload-portrait", h(uploadPortrait))
	mux.HandleFunc("DELETE "+chars+"/{cid}/portrait", h(resetPortrait))
	mux.HandleFunc("POST "+chars+"/{cid}/attach-animated-portrait", h(attachAnimatedPortrait))
	mux.HandleFunc("GET "+chars+"/{cid}/assets/{filename}", h(characterAsset))

	const traits = "/api/go/projects/{id}/custom-traits"
	mux.HandleFunc("GET "+traits, h(listCustomTraits))
	mux.HandleFunc("POST "+traits, h(createCustomTrait))
	mux.HandleFunc("GET "+traits+"/known", knownTraitsHandler(deps))
	mux.HandleFunc("GET "+traits+"/{tid}", h(getCustomTrait))
	mux.HandleFunc("PUT "+traits+"/{tid}", h(updateCustomTrait))
	mux.HandleFunc("DELETE "+traits+"/{tid}", h(deleteCustomTrait))
	mux.HandleFunc("POST "+traits+"/{tid}/upload-icon", h(uploadTraitIcon))
	mux.HandleFunc("GET "+traits+"/{tid}/icon", h(traitIcon))
}

type Character struct {
	ID                         int64             `json:"id"`
	ProjectID                  int64             `json:"project_id"`
	Name                       string            `json:"name"`
	Tag                        string            `json:"tag"`
	LocalisationKey            string            `json:"localisation_key"`
	CountryTag                 string            `json:"country_tag"`
	Description                string            `json:"description"`
	RoleType                   string            `json:"role_type"`
	HasAdvisorRole             bool              `json:"has_advisor_role"`
	IsReplacement              bool              `json:"is_replacement"`
	ReplacesVanillaID          bool              `json:"replaces_vanilla_id"`
	Skill                      int               `json:"skill"`
	AttackSkill                int               `json:"attack_skill"`
	DefenseSkill               int               `json:"defense_skill"`
	PlanningSkill              int               `json:"planning_skill"`
	LogisticsSkill             int               `json:"logistics_skill"`
	ManeuveringSkill           int               `json:"maneuvering_skill"`
	CoordinationSkill          int               `json:"coordination_skill"`
	MilitaryVisible            string            `json:"military_visible"`
	Traits                     []string          `json:"traits"`
	AdvisorTraits              []string          `json:"advisor_traits"`
	AdvisorSlot                string            `json:"advisor_slot"`
	AdvisorIdeaToken           string            `json:"advisor_idea_token"`
	AdvisorCost                int               `json:"advisor_cost"`
	AdvisorAvailable           string            `json:"advisor_available"`
	AdvisorVisible             string            `json:"advisor_visible"`
	AdvisorRemovalCost         int               `json:"advisor_removal_cost"`
	AdvisorModifier            string            `json:"advisor_modifier"`
	AdvisorOnAdd               string            `json:"advisor_on_add"`
	AdvisorOnRemove            string            `json:"advisor_on_remove"`
	Gender                     string            `json:"gender"`
	CanBeCaptured              bool              `json:"can_be_captured"`
	AlwaysShowOnActionsTooltip *bool             `json:"always_show_on_actions_tooltip"`
	ScientistData              map[string]any    `json:"scientist_data"`
	ScientistVisible           string            `json:"scientist_visible"`
	OperativeSkillLevel        int               `json:"operative_skill_level"`
	OperativeTraits            []string          `json:"operative_traits"`
	OperativeNationalities     []string          `json:"operative_nationalities"`
	OperativeAvailable         string            `json:"operative_available"`
	OperativeBg                string            `json:"operative_bg"`
	Ideology                   string            `json:"ideology"`
	Expire                     string            `json:"expire"`
	LargePortrait              string            `json:"large_portrait"`
	SmallPortrait              string            `json:"small_portrait"`
	RawExtras                  map[string]any    `json:"raw_extras"`
	NameEn                     string            `json:"name_en"`
	NameZh                     string            `json:"name_zh"`
	DescEn                     string            `json:"desc_en"`
	DescZh                     string            `json:"desc_zh"`
	DisplayNameLocs            map[string]string `json:"display_name_locs"`
	DescriptionLocs            map[string]string `json:"description_locs"`
	Source                     string            `json:"source"`
	SourceFile                 string            `json:"source_file"`
	IsPortraitModified         bool              `json:"is_portrait_modified"`
	Warnings                   []string          `json:"warnings"`
}

func characterFromRow(row project.Row) Character {
	c := Character{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		Name: str(row["name"]), Tag: str(row["tag"]), LocalisationKey: str(row["localisation_key"]),
		CountryTag: str(row["country_tag"]), Description: str(row["description"]),
		RoleType: str(row["role_type"]), HasAdvisorRole: boolOr(row["has_advisor_role"], false),
		IsReplacement: boolOr(row["is_replacement"], false), ReplacesVanillaID: boolOr(row["replaces_vanilla_id"], false),
		Skill: intOr(row["skill"], 1), AttackSkill: intOr(row["attack_skill"], 1),
		DefenseSkill: intOr(row["defense_skill"], 1), PlanningSkill: intOr(row["planning_skill"], 1),
		LogisticsSkill: intOr(row["logistics_skill"], 1), ManeuveringSkill: intOr(row["maneuvering_skill"], 1),
		CoordinationSkill: intOr(row["coordination_skill"], 1), MilitaryVisible: str(row["military_visible"]),
		AdvisorSlot: str(row["advisor_slot"]), AdvisorIdeaToken: str(row["advisor_idea_token"]),
		AdvisorCost: intOr(row["advisor_cost"], 150), AdvisorAvailable: str(row["advisor_available"]),
		AdvisorVisible: str(row["advisor_visible"]), AdvisorRemovalCost: intOr(row["advisor_removal_cost"], -1),
		AdvisorModifier: str(row["advisor_modifier"]), AdvisorOnAdd: str(row["advisor_on_add"]),
		AdvisorOnRemove: str(row["advisor_on_remove"]), Gender: str(row["gender"]),
		CanBeCaptured:              boolOr(row["can_be_captured"], true),
		AlwaysShowOnActionsTooltip: boolPtr(row["always_show_on_actions_tooltip"]),
		ScientistVisible:           str(row["scientist_visible"]),
		OperativeSkillLevel:        intOr(row["operative_skill_level"], 1),
		OperativeAvailable:         str(row["operative_available"]), OperativeBg: str(row["operative_bg"]),
		Ideology: str(row["ideology"]), Expire: str(row["expire"]),
		LargePortrait: str(row["large_portrait"]), SmallPortrait: str(row["small_portrait"]),
		NameEn: str(row["name_en"]), NameZh: str(row["name_zh"]), DescEn: str(row["desc_en"]), DescZh: str(row["desc_zh"]),
		Source: str(row["source"]), SourceFile: str(row["source_file"]),
		IsPortraitModified: boolOr(row["is_portrait_modified"], false),
		Warnings:           []string{},
	}
	decodeColumn(row["traits"], &c.Traits)
	decodeColumn(row["advisor_traits"], &c.AdvisorTraits)
	decodeColumn(row["scientist_data"], &c.ScientistData)
	decodeColumn(row["operative_traits"], &c.OperativeTraits)
	decodeColumn(row["operative_nationalities"], &c.OperativeNationalities)
	decodeColumn(row["raw_extras"], &c.RawExtras)
	decodeColumn(row["display_name_locs"], &c.DisplayNameLocs)
	decodeColumn(row["description_locs"], &c.DescriptionLocs)

	if c.Traits == nil {
		c.Traits = []string{}
	}
	if c.AdvisorTraits == nil {
		c.AdvisorTraits = []string{}
	}
	if c.ScientistData == nil {
		c.ScientistData = map[string]any{}
	}
	if c.OperativeTraits == nil {
		c.OperativeTraits = []string{}
	}
	if c.OperativeNationalities == nil {
		c.OperativeNationalities = []string{}
	}
	if c.RawExtras == nil {
		c.RawExtras = map[string]any{}
	}
	if c.DisplayNameLocs == nil {
		c.DisplayNameLocs = map[string]string{}
	}
	if c.DescriptionLocs == nil {
		c.DescriptionLocs = map[string]string{}
	}
	return c
}

var localisationKeyRE = regexp.MustCompile(`^[A-Za-z0-9_.'-]+$`)

func validateLocalisationKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value != "" && !localisationKeyRE.MatchString(value) {
		return "", fmt.Errorf("localisation_key may contain only ASCII letters, numbers, underscores, dots, apostrophes, and hyphens")
	}
	return value, nil
}

type CharacterCreateInput struct {
	Name                       *string           `json:"name"`
	Tag                        string            `json:"tag"`
	LocalisationKey            string            `json:"localisation_key"`
	CountryTag                 string            `json:"country_tag"`
	Description                string            `json:"description"`
	RoleType                   *string           `json:"role_type"`
	HasAdvisorRole             bool              `json:"has_advisor_role"`
	IsReplacement              bool              `json:"is_replacement"`
	ReplacesVanillaID          bool              `json:"replaces_vanilla_id"`
	Skill                      *int              `json:"skill"`
	AttackSkill                *int              `json:"attack_skill"`
	DefenseSkill               *int              `json:"defense_skill"`
	PlanningSkill              *int              `json:"planning_skill"`
	LogisticsSkill             *int              `json:"logistics_skill"`
	ManeuveringSkill           *int              `json:"maneuvering_skill"`
	CoordinationSkill          *int              `json:"coordination_skill"`
	MilitaryVisible            string            `json:"military_visible"`
	Traits                     []string          `json:"traits"`
	AdvisorTraits              []string          `json:"advisor_traits"`
	AdvisorSlot                string            `json:"advisor_slot"`
	AdvisorIdeaToken           string            `json:"advisor_idea_token"`
	AdvisorCost                *int              `json:"advisor_cost"`
	AdvisorAvailable           string            `json:"advisor_available"`
	AdvisorVisible             string            `json:"advisor_visible"`
	AdvisorRemovalCost         *int              `json:"advisor_removal_cost"`
	AdvisorModifier            string            `json:"advisor_modifier"`
	AdvisorOnAdd               string            `json:"advisor_on_add"`
	AdvisorOnRemove            string            `json:"advisor_on_remove"`
	Gender                     string            `json:"gender"`
	CanBeCaptured              *bool             `json:"can_be_captured"`
	AlwaysShowOnActionsTooltip *bool             `json:"always_show_on_actions_tooltip"`
	ScientistData              map[string]any    `json:"scientist_data"`
	ScientistVisible           string            `json:"scientist_visible"`
	OperativeSkillLevel        *int              `json:"operative_skill_level"`
	OperativeTraits            []string          `json:"operative_traits"`
	OperativeNationalities     []string          `json:"operative_nationalities"`
	OperativeAvailable         string            `json:"operative_available"`
	OperativeBg                string            `json:"operative_bg"`
	Ideology                   string            `json:"ideology"`
	Expire                     string            `json:"expire"`
	NameEn                     string            `json:"name_en"`
	NameZh                     string            `json:"name_zh"`
	DescEn                     string            `json:"desc_en"`
	DescZh                     string            `json:"desc_zh"`
	DisplayNameLocs            map[string]string `json:"display_name_locs"`
	DescriptionLocs            map[string]string `json:"description_locs"`
	Source                     *string           `json:"source"`
	SourceFile                 string            `json:"source_file"`
	IsPortraitModified         bool              `json:"is_portrait_modified"`
}

func (in *CharacterCreateInput) Normalize() error {
	if in.Name == nil {
		return fmt.Errorf("name is required")
	}
	key, err := validateLocalisationKey(in.LocalisationKey)
	if err != nil {
		return err
	}
	in.LocalisationKey = key
	if in.Traits == nil {
		in.Traits = []string{}
	}
	if in.AdvisorTraits == nil {
		in.AdvisorTraits = []string{}
	}
	if in.ScientistData == nil {
		in.ScientistData = map[string]any{}
	}
	if in.OperativeTraits == nil {
		in.OperativeTraits = []string{}
	}
	if in.OperativeNationalities == nil {
		in.OperativeNationalities = []string{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	return nil
}

func listCharacters(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	rows, err := p.Rows(table)
	if err != nil {
		return err
	}
	countryTag := r.URL.Query().Get("country_tag")
	out := []Character{}
	for _, row := range rows {
		if countryTag != "" && str(row["country_tag"]) != countryTag {
			continue
		}
		out = append(out, characterFromRow(row))
	}
	apiutil.JSON(w, http.StatusOK, out)
	return nil
}

var errCharacterMissing = detail(http.StatusNotFound, "Character not found")

func getCharacter(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, ok := pathID(r, "cid")
	if !ok {
		return errCharacterMissing
	}
	row, err := findRow(p, table, cid)
	if err != nil {
		return err
	}
	if row == nil {
		return errCharacterMissing
	}
	apiutil.JSON(w, http.StatusOK, characterFromRow(row))
	return nil
}

func createCharacter(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var in CharacterCreateInput
	if err := decodeBody(w, r, 4<<20, &in); err != nil {
		return detail(http.StatusUnprocessableEntity, "Invalid character payload")
	}
	if err := in.Normalize(); err != nil {
		return detail(http.StatusUnprocessableEntity, err.Error())
	}
	countryTag := strings.ToUpper(firstNonBlank(in.CountryTag, projectCountryTag(p)))
	tag := strings.TrimSpace(in.Tag)
	if tag == "" {
		tag = emitcharacters.MakeCharacterID(countryTag, *in.Name)
	}
	warnings := []string{}
	var created project.Row
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		taken := map[string]bool{}
		for _, row := range tx.Rows(table) {
			taken[str(row["tag"])] = true
		}
		originalTag := tag
		for suffix := 1; taken[tag]; suffix++ {
			tag = fmt.Sprintf("%s_%d", originalTag, suffix)
		}
		if tag != originalTag {
			warnings = append(warnings, fmt.Sprintf("tag '%s' collided · auto-suffixed to '%s'.", originalTag, tag))
		}
		created = tx.Insert(table, project.Row{
			"country_tag": countryTag, "name": *in.Name, "tag": tag,
			"localisation_key": in.LocalisationKey, "description": in.Description,
			"role_type": strDefault(in.RoleType, "general"), "has_advisor_role": in.HasAdvisorRole,
			"is_replacement": in.IsReplacement, "replaces_vanilla_id": in.ReplacesVanillaID,
			"skill": int64(intDefault(in.Skill, 1)), "attack_skill": int64(intDefault(in.AttackSkill, 1)),
			"defense_skill": int64(intDefault(in.DefenseSkill, 1)), "planning_skill": int64(intDefault(in.PlanningSkill, 1)),
			"logistics_skill": int64(intDefault(in.LogisticsSkill, 1)), "maneuvering_skill": int64(intDefault(in.ManeuveringSkill, 1)),
			"coordination_skill": int64(intDefault(in.CoordinationSkill, 1)),
			"military_visible":   in.MilitaryVisible, "traits": jsonValue(in.Traits),
			"advisor_traits": jsonValue(in.AdvisorTraits), "advisor_slot": in.AdvisorSlot,
			"advisor_idea_token": in.AdvisorIdeaToken, "advisor_cost": int64(intDefault(in.AdvisorCost, 150)),
			"scientist_data": jsonValue(in.ScientistData), "scientist_visible": in.ScientistVisible,
			"operative_skill_level": int64(intDefault(in.OperativeSkillLevel, 1)),
			"operative_traits":      jsonValue(in.OperativeTraits), "operative_nationalities": jsonValue(in.OperativeNationalities),
			"operative_available": in.OperativeAvailable, "operative_bg": in.OperativeBg,
			"name_en": firstNonBlank(in.NameEn, *in.Name), "name_zh": in.NameZh,
			"desc_en": firstNonBlank(in.DescEn, in.Description), "desc_zh": in.DescZh,
			"display_name_locs": jsonValue(in.DisplayNameLocs), "description_locs": jsonValue(in.DescriptionLocs),
			"advisor_available": in.AdvisorAvailable, "advisor_visible": in.AdvisorVisible,
			"advisor_removal_cost": int64(intDefault(in.AdvisorRemovalCost, -1)),
			"advisor_modifier":     in.AdvisorModifier, "advisor_on_add": in.AdvisorOnAdd,
			"advisor_on_remove": in.AdvisorOnRemove, "gender": in.Gender,
			"can_be_captured":                boolDefault(in.CanBeCaptured, true),
			"always_show_on_actions_tooltip": jsonValue(in.AlwaysShowOnActionsTooltip),
			"ideology":                       in.Ideology, "expire": in.Expire,
			"source": strDefault(in.Source, "created"), "source_file": in.SourceFile,
			"is_portrait_modified": in.IsPortraitModified, "large_portrait": "", "small_portrait": "",
			"raw_extras": map[string]any{}, "has_obs": false,
		})
		return nil
	})
	if err != nil {
		return err
	}
	c := characterFromRow(created)
	c.Warnings = warnings
	apiutil.JSON(w, http.StatusCreated, c)
	return nil
}

type CharacterPatch struct {
	Name                       workbench.Optional[string]            `json:"name"`
	Tag                        workbench.Optional[string]            `json:"tag"`
	LocalisationKey            workbench.Optional[string]            `json:"localisation_key"`
	CountryTag                 workbench.Optional[string]            `json:"country_tag"`
	Description                workbench.Optional[string]            `json:"description"`
	RoleType                   workbench.Optional[string]            `json:"role_type"`
	HasAdvisorRole             workbench.Optional[bool]              `json:"has_advisor_role"`
	IsReplacement              workbench.Optional[bool]              `json:"is_replacement"`
	ReplacesVanillaID          workbench.Optional[bool]              `json:"replaces_vanilla_id"`
	Skill                      workbench.Optional[int]               `json:"skill"`
	AttackSkill                workbench.Optional[int]               `json:"attack_skill"`
	DefenseSkill               workbench.Optional[int]               `json:"defense_skill"`
	PlanningSkill              workbench.Optional[int]               `json:"planning_skill"`
	LogisticsSkill             workbench.Optional[int]               `json:"logistics_skill"`
	ManeuveringSkill           workbench.Optional[int]               `json:"maneuvering_skill"`
	CoordinationSkill          workbench.Optional[int]               `json:"coordination_skill"`
	MilitaryVisible            workbench.Optional[string]            `json:"military_visible"`
	Traits                     workbench.Optional[[]string]          `json:"traits"`
	AdvisorTraits              workbench.Optional[[]string]          `json:"advisor_traits"`
	AdvisorSlot                workbench.Optional[string]            `json:"advisor_slot"`
	AdvisorIdeaToken           workbench.Optional[string]            `json:"advisor_idea_token"`
	AdvisorCost                workbench.Optional[int]               `json:"advisor_cost"`
	AdvisorAvailable           workbench.Optional[string]            `json:"advisor_available"`
	AdvisorVisible             workbench.Optional[string]            `json:"advisor_visible"`
	AdvisorRemovalCost         workbench.Optional[int]               `json:"advisor_removal_cost"`
	AdvisorModifier            workbench.Optional[string]            `json:"advisor_modifier"`
	AdvisorOnAdd               workbench.Optional[string]            `json:"advisor_on_add"`
	AdvisorOnRemove            workbench.Optional[string]            `json:"advisor_on_remove"`
	Gender                     workbench.Optional[string]            `json:"gender"`
	CanBeCaptured              workbench.Optional[bool]              `json:"can_be_captured"`
	AlwaysShowOnActionsTooltip workbench.Optional[bool]              `json:"always_show_on_actions_tooltip"`
	ScientistData              workbench.Optional[map[string]any]    `json:"scientist_data"`
	ScientistVisible           workbench.Optional[string]            `json:"scientist_visible"`
	OperativeSkillLevel        workbench.Optional[int]               `json:"operative_skill_level"`
	OperativeTraits            workbench.Optional[[]string]          `json:"operative_traits"`
	OperativeNationalities     workbench.Optional[[]string]          `json:"operative_nationalities"`
	OperativeAvailable         workbench.Optional[string]            `json:"operative_available"`
	OperativeBg                workbench.Optional[string]            `json:"operative_bg"`
	Ideology                   workbench.Optional[string]            `json:"ideology"`
	Expire                     workbench.Optional[string]            `json:"expire"`
	NameEn                     workbench.Optional[string]            `json:"name_en"`
	NameZh                     workbench.Optional[string]            `json:"name_zh"`
	DescEn                     workbench.Optional[string]            `json:"desc_en"`
	DescZh                     workbench.Optional[string]            `json:"desc_zh"`
	DisplayNameLocs            workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs            workbench.Optional[map[string]string] `json:"description_locs"`
	RawExtras                  workbench.Optional[map[string]any]    `json:"raw_extras"`
	Source                     workbench.Optional[string]            `json:"source"`
	SourceFile                 workbench.Optional[string]            `json:"source_file"`
	IsPortraitModified         workbench.Optional[bool]              `json:"is_portrait_modified"`
}

func (p *CharacterPatch) Validate() error {
	if p.LocalisationKey.Set && p.LocalisationKey.Value != nil {
		key, err := validateLocalisationKey(*p.LocalisationKey.Value)
		if err != nil {
			return err
		}
		p.LocalisationKey.Value = &key
	}
	return nil
}

func set[T any](changes map[string]any, column string, o workbench.Optional[T]) {
	if !o.Set {
		return
	}
	if o.Value == nil {
		changes[column] = nil
		return
	}
	switch v := any(*o.Value).(type) {
	case int:
		changes[column] = int64(v)
	case string, bool:
		changes[column] = v
	default:
		changes[column] = jsonValue(v)
	}
}

func updateCharacter(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, ok := pathID(r, "cid")
	if !ok {
		return errCharacterMissing
	}
	var patch CharacterPatch
	if err := decodeBody(w, r, 4<<20, &patch); err != nil {
		return detail(http.StatusUnprocessableEntity, "Invalid character payload")
	}
	if err := patch.Validate(); err != nil {
		return detail(http.StatusUnprocessableEntity, err.Error())
	}
	var updated project.Row
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		current := tx.Find(table, cid)
		if current == nil {
			return errCharacterMissing
		}

		if patch.Description.Set && !patch.DescEn.Set && patch.Description.Value != nil &&
			strings.TrimSpace(*patch.Description.Value) != "" &&
			strings.TrimSpace(str(current["desc_en"])) == "" {
			v := *patch.Description.Value
			patch.DescEn = workbench.Optional[string]{Set: true, Value: &v}
		}
		if patch.Tag.Set && patch.Tag.Value != nil {
			v := strings.TrimSpace(*patch.Tag.Value)
			patch.Tag.Value = &v
		}
		if patch.LocalisationKey.Set && patch.LocalisationKey.Value != nil {
			v := strings.TrimSpace(*patch.LocalisationKey.Value)
			patch.LocalisationKey.Value = &v
		}
		if patch.CountryTag.Set && patch.CountryTag.Value != nil {
			v := upperTrim(*patch.CountryTag.Value)
			patch.CountryTag.Value = &v
		}
		if patch.SourceFile.Set && patch.SourceFile.Value == nil {
			v := ""
			patch.SourceFile.Value = &v
		}
		changes := map[string]any{}
		set(changes, "name", patch.Name)
		set(changes, "tag", patch.Tag)
		set(changes, "localisation_key", patch.LocalisationKey)
		set(changes, "country_tag", patch.CountryTag)
		set(changes, "description", patch.Description)
		set(changes, "role_type", patch.RoleType)
		set(changes, "has_advisor_role", patch.HasAdvisorRole)
		set(changes, "is_replacement", patch.IsReplacement)
		set(changes, "replaces_vanilla_id", patch.ReplacesVanillaID)
		set(changes, "skill", patch.Skill)
		set(changes, "attack_skill", patch.AttackSkill)
		set(changes, "defense_skill", patch.DefenseSkill)
		set(changes, "planning_skill", patch.PlanningSkill)
		set(changes, "logistics_skill", patch.LogisticsSkill)
		set(changes, "maneuvering_skill", patch.ManeuveringSkill)
		set(changes, "coordination_skill", patch.CoordinationSkill)
		set(changes, "military_visible", patch.MilitaryVisible)
		set(changes, "traits", patch.Traits)
		set(changes, "advisor_traits", patch.AdvisorTraits)
		set(changes, "advisor_slot", patch.AdvisorSlot)
		set(changes, "advisor_idea_token", patch.AdvisorIdeaToken)
		set(changes, "advisor_cost", patch.AdvisorCost)
		set(changes, "advisor_available", patch.AdvisorAvailable)
		set(changes, "advisor_visible", patch.AdvisorVisible)
		set(changes, "advisor_removal_cost", patch.AdvisorRemovalCost)
		set(changes, "advisor_modifier", patch.AdvisorModifier)
		set(changes, "advisor_on_add", patch.AdvisorOnAdd)
		set(changes, "advisor_on_remove", patch.AdvisorOnRemove)
		set(changes, "gender", patch.Gender)
		set(changes, "can_be_captured", patch.CanBeCaptured)
		set(changes, "always_show_on_actions_tooltip", patch.AlwaysShowOnActionsTooltip)
		set(changes, "scientist_data", patch.ScientistData)
		set(changes, "scientist_visible", patch.ScientistVisible)
		set(changes, "operative_skill_level", patch.OperativeSkillLevel)
		set(changes, "operative_traits", patch.OperativeTraits)
		set(changes, "operative_nationalities", patch.OperativeNationalities)
		set(changes, "operative_available", patch.OperativeAvailable)
		set(changes, "operative_bg", patch.OperativeBg)
		set(changes, "ideology", patch.Ideology)
		set(changes, "expire", patch.Expire)
		set(changes, "name_en", patch.NameEn)
		set(changes, "name_zh", patch.NameZh)
		set(changes, "desc_en", patch.DescEn)
		set(changes, "desc_zh", patch.DescZh)
		set(changes, "display_name_locs", patch.DisplayNameLocs)
		set(changes, "description_locs", patch.DescriptionLocs)
		set(changes, "raw_extras", patch.RawExtras)
		set(changes, "source", patch.Source)
		set(changes, "source_file", patch.SourceFile)
		set(changes, "is_portrait_modified", patch.IsPortraitModified)

		if patch.Source.Set && patch.Source.Value != nil {
			if *patch.Source.Value == "imported" {
				changes["source"] = "created"
			}
		} else if str(current["source"]) == "imported" {
			changes["source"] = "created"
		}
		if len(changes) == 0 {
			updated = current
			return nil
		}
		if notNullViolation(table, changes) {
			return detail(http.StatusServiceUnavailable, "character store unavailable")
		}

		if tag, ok := changes["tag"].(string); ok {
			for _, row := range tx.Rows(table) {
				if project.Int64(row["id"]) != cid && str(row["tag"]) == tag {
					return detail(http.StatusConflict, fmt.Sprintf(
						"tag '%s' already exists in this project. Pick a unique tag.", tag))
				}
			}
		}
		var err error
		updated, err = tx.Update(table, cid, func(row project.Row) {
			for column, value := range changes {
				row[column] = value
			}
		})
		return err
	})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, characterFromRow(updated))
	return nil
}

func deleteCharacter(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, ok := pathID(r, "cid")
	if !ok {
		return errCharacterMissing
	}
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		if tx.Delete(table, func(row project.Row) bool { return project.Int64(row["id"]) == cid }) == 0 {
			return errCharacterMissing
		}
		return nil
	})
	if err != nil {
		return err
	}
	apiutil.NoContent(w)
	return nil
}

func forkCharacter(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, ok := pathID(r, "cid")
	if !ok {
		return errCharacterMissing
	}
	var body struct {
		NewTag string `json:"new_tag"`
	}
	if err := decodeBody(w, r, 1<<20, &body); err != nil || body.NewTag == "" {
		return detail(http.StatusUnprocessableEntity, "new_tag is required")
	}
	newTag := strings.TrimSpace(body.NewTag)
	if newTag == "" {
		return detail(http.StatusBadRequest, "new_tag cannot be empty")
	}
	var created project.Row
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		source := tx.Find(table, cid)
		if source == nil {
			return detail(http.StatusNotFound, "Source character not found")
		}
		if newTag == str(source["tag"]) {
			return detail(http.StatusBadRequest, "new_tag must differ from the original tag")
		}
		for _, row := range tx.Rows(table) {
			if str(row["tag"]) == newTag {
				return detail(http.StatusConflict, fmt.Sprintf("tag '%s' already exists in this project", newTag))
			}
		}
		clone := project.Row{}
		for column, value := range source {
			switch column {
			case "id", "created_at", "updated_at":
				continue
			}
			clone[column] = value
		}
		clone["tag"] = newTag
		clone["source"] = "created"
		created = tx.Insert(table, clone)
		return nil
	})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusCreated, characterFromRow(created))
	return nil
}
