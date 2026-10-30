package toolapi

import (
	"context"
	"fmt"
	"strings"
)

func int64Field(m map[string]any, key string) int64 {
	if f, ok := m[key].(float64); ok {
		return int64(f)
	}
	return 0
}

func stringSliceField(m map[string]any, key string) []string {
	raw, ok := m[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func stringMapField(m map[string]any, key string) map[string]string {
	raw, ok := m[key].(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func orEmptySlice[T any](xs []T) []T {
	if xs == nil {
		return []T{}
	}
	return xs
}

func orEmptyMap[V any](m map[string]V) map[string]V {
	if m == nil {
		return map[string]V{}
	}
	return m
}

type createMioIn struct {
	ProjectID          int64    `json:"project_id"`
	MioID              string   `json:"mio_id" jsonschema:"unique script id; blank -> 400, duplicate -> 409 (no auto-suffix)"`
	Include            string   `json:"include,omitempty" jsonschema:"a vanilla/base MIO template to inherit, e.g. generic_tank_manufacturer_3"`
	Icon               string   `json:"icon,omitempty"`
	EquipmentType      []string `json:"equipment_type,omitempty"`
	ResearchCategories []string `json:"research_categories,omitempty"`
	Allowed            string   `json:"allowed,omitempty" jsonschema:"RAW trigger block"`
	Available          string   `json:"available,omitempty" jsonschema:"RAW trigger block"`
	Visible            string   `json:"visible,omitempty" jsonschema:"RAW trigger block"`
	CountryTag         string   `json:"country_tag,omitempty"`
	ScopeKind          string   `json:"scope_kind,omitempty" jsonschema:"default own"`
	NameEn             string   `json:"name_en,omitempty"`
	NameZh             string   `json:"name_zh,omitempty"`
}

type createMioOut struct {
	MioDBID int64  `json:"mio_db_id"`
	MioID   string `json:"mio_id"`
}

func registerCreateMio(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_mio",
		Description: "Create a Military Industrial Organization (manufacturer/design company). " +
			"DLC: Trial of Allegiance. A MIO has NO top-level modifier — its bonuses live inside " +
			"trait entries; use add_mio_trait to add them. Returns mio_db_id (numeric, needed by " +
			"add_mio_trait) and mio_id.",
	}, func(ctx context.Context, _ *CallToolRequest, in createMioIn) (*CallToolResult, createMioOut, error) {
		scopeKind := in.ScopeKind
		if scopeKind == "" {
			scopeKind = "own"
		}
		body := map[string]any{
			"mio_id":              in.MioID,
			"include":             in.Include,
			"icon":                in.Icon,
			"equipment_type":      orEmptySlice(in.EquipmentType),
			"research_categories": orEmptySlice(in.ResearchCategories),
			"allowed":             in.Allowed,
			"available":           in.Available,
			"visible":             in.Visible,
			"country_tag":         in.CountryTag,
			"scope_kind":          scopeKind,
			"name_en":             in.NameEn,
			"name_zh":             in.NameZh,
		}
		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/mios", in.ProjectID), body)
		if err != nil {
			return nil, createMioOut{}, err
		}
		return nil, createMioOut{
			MioDBID: int64Field(resp, "id"),
			MioID:   stringField(resp, "mio_id"),
		}, nil
	})
}

type addMioTraitIn struct {
	ProjectID              int64          `json:"project_id"`
	MioDBID                int64          `json:"mio_db_id" jsonschema:"numeric id returned by create_mio"`
	Token                  string         `json:"token" jsonschema:"stable per-trait key"`
	Name                   string         `json:"name,omitempty" jsonschema:"HOI4 loc key, not player-facing text; blank uses the engine default"`
	Icon                   string         `json:"icon,omitempty"`
	Kind                   string         `json:"kind,omitempty" jsonschema:"initial | add | override; default add"`
	Position               map[string]any `json:"position,omitempty"`
	RelativePositionID     string         `json:"relative_position_id,omitempty"`
	AllParents             []string       `json:"all_parents,omitempty" jsonschema:"prerequisite trait tokens (AND)"`
	AnyParent              []string       `json:"any_parent,omitempty" jsonschema:"prerequisite trait tokens (OR)"`
	LimitToEquipmentType   []string       `json:"limit_to_equipment_type,omitempty"`
	EquipmentBonus         map[string]any `json:"equipment_bonus,omitempty"`
	ProductionBonus        map[string]any `json:"production_bonus,omitempty"`
	OrganizationModifier   map[string]any `json:"organization_modifier,omitempty"`
	Available              string         `json:"available,omitempty" jsonschema:"RAW trigger block"`
	Visible                string         `json:"visible,omitempty" jsonschema:"RAW trigger block"`
	OnComplete             string         `json:"on_complete,omitempty" jsonschema:"RAW effect block"`
	AiWillDo               string         `json:"ai_will_do,omitempty" jsonschema:"RAW ai_will_do block"`
	SpecialTraitBackground bool           `json:"special_trait_background,omitempty"`
}

type addMioTraitOut struct {
	MioDBID    int64  `json:"mio_db_id"`
	MioID      string `json:"mio_id"`
	TraitCount int    `json:"trait_count"`
}

func registerAddMioTrait(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "add_mio_trait",
		Description: "Append one trait to a MIO's trait tree (read-modify-write). DLC: Trial of " +
			"Allegiance. MIO modifiers are the NESTED per-trait dict blocks equipment_bonus / " +
			"production_bonus / organization_modifier — NOT a top-level modifier. This reads the " +
			"MIO's existing traits and APPENDS (never send only the new trait — the underlying " +
			"update REPLACES the whole list). Returns mio_db_id, mio_id, trait_count.",
	}, func(ctx context.Context, _ *CallToolRequest, in addMioTraitIn) (*CallToolResult, addMioTraitOut, error) {
		tok := strings.TrimSpace(in.Token)
		if tok == "" {
			return nil, addMioTraitOut{}, fmt.Errorf("token is required (the stable per-trait key)")
		}
		kind := in.Kind
		if kind == "" {
			kind = "add"
		}

		mioPath := fmt.Sprintf("/api/projects/%d/mios/%d", in.ProjectID, in.MioDBID)
		current, err := proxyAuthedCall(ctx, proxy, "GET", mioPath, nil)
		if err != nil {
			return nil, addMioTraitOut{}, err
		}

		trait := map[string]any{"token": tok, "kind": kind}
		if in.Name != "" {
			trait["name"] = in.Name
		}
		if in.Icon != "" {
			trait["icon"] = in.Icon
		}
		if in.Position != nil {
			trait["position"] = in.Position
		}
		if in.RelativePositionID != "" {
			trait["relative_position_id"] = in.RelativePositionID
		}
		if in.AllParents != nil {
			trait["all_parents"] = in.AllParents
		}
		if in.AnyParent != nil {
			trait["any_parent"] = in.AnyParent
		}
		if in.LimitToEquipmentType != nil {
			trait["limit_to_equipment_type"] = in.LimitToEquipmentType
		}
		if in.EquipmentBonus != nil {
			trait["equipment_bonus"] = in.EquipmentBonus
		}
		if in.ProductionBonus != nil {
			trait["production_bonus"] = in.ProductionBonus
		}
		if in.OrganizationModifier != nil {
			trait["organization_modifier"] = in.OrganizationModifier
		}
		if in.Available != "" {
			trait["available"] = in.Available
		}
		if in.Visible != "" {
			trait["visible"] = in.Visible
		}
		if in.OnComplete != "" {
			trait["on_complete"] = in.OnComplete
		}
		if in.AiWillDo != "" {
			trait["ai_will_do"] = in.AiWillDo
		}
		if in.SpecialTraitBackground {
			trait["special_trait_background"] = true
		}

		existing, _ := current["traits"].([]any)
		newTraits := append(append([]any{}, existing...), trait)

		resp, err := proxyAuthedCall(ctx, proxy, "PUT", mioPath, map[string]any{"traits": newTraits})
		if err != nil {
			return nil, addMioTraitOut{}, err
		}

		out := addMioTraitOut{
			MioDBID: int64Field(resp, "id"),
			MioID:   stringField(resp, "mio_id"),
		}
		if traits, ok := resp["traits"].([]any); ok {
			out.TraitCount = len(traits)
		}
		return nil, out, nil
	})
}

type createIntelligenceAgencyIn struct {
	ProjectID      int64    `json:"project_id"`
	AgencyID       string   `json:"agency_id" jsonschema:"synthetic per-project storage key; blank -> 400, duplicate -> 409 (no auto-suffix)"`
	Picture        string   `json:"picture,omitempty"`
	Names          []string `json:"names,omitempty" jsonschema:"pool of candidate agency names the engine picks from at random"`
	DefaultTrigger string   `json:"default_trigger,omitempty" jsonschema:"RAW trigger block"`
	Available      string   `json:"available,omitempty" jsonschema:"RAW trigger block"`
	CountryTag     string   `json:"country_tag,omitempty"`
	ScopeKind      string   `json:"scope_kind,omitempty" jsonschema:"default own"`
	NameEn         string   `json:"name_en,omitempty"`
	NameZh         string   `json:"name_zh,omitempty"`
}

type createIntelligenceAgencyOut struct {
	AgencyDBID int64  `json:"agency_db_id"`
	AgencyID   string `json:"agency_id"`
	NameEn     string `json:"name_en"`
}

func registerCreateIntelligenceAgency(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_intelligence_agency",
		Description: "Create an intelligence agency (a country's spy-agency identity). DLC: La " +
			"Résistance. agency_id is not a block identifier; when names is empty, export uses " +
			"agency_id as the sole names token and localisation key. An agency has NO description " +
			"slot — use add_agency_upgrade for the upgrade tree. Returns agency_db_id (numeric, " +
			"needed by add_agency_upgrade) and agency_id.",
	}, func(ctx context.Context, _ *CallToolRequest, in createIntelligenceAgencyIn) (*CallToolResult, createIntelligenceAgencyOut, error) {
		scopeKind := in.ScopeKind
		if scopeKind == "" {
			scopeKind = "own"
		}
		body := map[string]any{
			"agency_id":       in.AgencyID,
			"picture":         in.Picture,
			"names":           orEmptySlice(in.Names),
			"default_trigger": in.DefaultTrigger,
			"available":       in.Available,
			"country_tag":     in.CountryTag,
			"scope_kind":      scopeKind,
			"name_en":         in.NameEn,
			"name_zh":         in.NameZh,
		}
		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/intelligence-agencies", in.ProjectID), body)
		if err != nil {
			return nil, createIntelligenceAgencyOut{}, err
		}
		return nil, createIntelligenceAgencyOut{
			AgencyDBID: int64Field(resp, "id"),
			AgencyID:   stringField(resp, "agency_id"),
			NameEn:     stringField(resp, "name_en"),
		}, nil
	})
}

type addAgencyUpgradeIn struct {
	ProjectID               int64    `json:"project_id"`
	AgencyDBID              int64    `json:"agency_db_id" jsonschema:"numeric id returned by create_intelligence_agency"`
	Branch                  string   `json:"branch" jsonschema:"branch token the upgrade sits under (created if it doesn't exist yet)"`
	Token                   string   `json:"token" jsonschema:"upgrade token, unique across branches"`
	Picture                 string   `json:"picture,omitempty"`
	Frame                   string   `json:"frame,omitempty"`
	Sound                   string   `json:"sound,omitempty"`
	Available               string   `json:"available,omitempty" jsonschema:"RAW trigger block"`
	Visible                 string   `json:"visible,omitempty" jsonschema:"RAW trigger block"`
	AiWillDo                string   `json:"ai_will_do,omitempty" jsonschema:"RAW ai_will_do block"`
	ModifiersDuringProgress string   `json:"modifiers_during_progress,omitempty" jsonschema:"RAW modifier block"`
	Levels                  []string `json:"levels,omitempty" jsonschema:"one RAW level-block body per level"`
}

type addAgencyUpgradeOut struct {
	AgencyDBID  int64  `json:"agency_db_id"`
	AgencyID    string `json:"agency_id"`
	BranchCount int    `json:"branch_count"`
}

func registerAddAgencyUpgrade(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "add_agency_upgrade",
		Description: "Append one upgrade to an intelligence agency's upgrade tree (read-modify-write). " +
			"DLC: La Résistance. NB upgrades are mod-GLOBAL: they ride on the agency row for editing " +
			"convenience, but export merges every agency's branches into one upgrades file de-duped " +
			"by token. This reads the agency's existing upgrade_branches and APPENDS (never send only " +
			"the new upgrade). Returns agency_db_id, agency_id, branch_count.",
	}, func(ctx context.Context, _ *CallToolRequest, in addAgencyUpgradeIn) (*CallToolResult, addAgencyUpgradeOut, error) {
		br := strings.TrimSpace(in.Branch)
		tok := strings.TrimSpace(in.Token)
		if br == "" {
			return nil, addAgencyUpgradeOut{}, fmt.Errorf("branch is required (the branch token)")
		}
		if tok == "" {
			return nil, addAgencyUpgradeOut{}, fmt.Errorf("token is required (the upgrade token)")
		}

		agencyPath := fmt.Sprintf("/api/projects/%d/intelligence-agencies/%d", in.ProjectID, in.AgencyDBID)
		current, err := proxyAuthedCall(ctx, proxy, "GET", agencyPath, nil)
		if err != nil {
			return nil, addAgencyUpgradeOut{}, err
		}

		upgrade := map[string]any{"token": tok}
		if in.Picture != "" {
			upgrade["picture"] = in.Picture
		}
		if in.Frame != "" {
			upgrade["frame"] = in.Frame
		}
		if in.Sound != "" {
			upgrade["sound"] = in.Sound
		}
		if in.Available != "" {
			upgrade["available"] = in.Available
		}
		if in.Visible != "" {
			upgrade["visible"] = in.Visible
		}
		if in.AiWillDo != "" {
			upgrade["ai_will_do"] = in.AiWillDo
		}
		if in.ModifiersDuringProgress != "" {
			upgrade["modifiers_during_progress"] = in.ModifiersDuringProgress
		}
		if in.Levels != nil {
			upgrade["levels"] = in.Levels
		}

		existingBranches, _ := current["upgrade_branches"].([]any)
		branches := make([]any, 0, len(existingBranches)+1)
		found := false
		for _, b := range existingBranches {
			bm, ok := b.(map[string]any)
			if !ok {
				branches = append(branches, b)
				continue
			}

			cp := make(map[string]any, len(bm))
			for k, v := range bm {
				cp[k] = v
			}
			if name, ok := cp["branch"].(string); ok && name == br {
				upgrades, _ := cp["upgrades"].([]any)
				cp["upgrades"] = append(append([]any{}, upgrades...), upgrade)
				found = true
			}
			branches = append(branches, cp)
		}
		if !found {
			branches = append(branches, map[string]any{"branch": br, "upgrades": []any{upgrade}})
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT", agencyPath, map[string]any{"upgrade_branches": branches})
		if err != nil {
			return nil, addAgencyUpgradeOut{}, err
		}

		out := addAgencyUpgradeOut{
			AgencyDBID: int64Field(resp, "id"),
			AgencyID:   stringField(resp, "agency_id"),
		}
		if b, ok := resp["upgrade_branches"].([]any); ok {
			out.BranchCount = len(b)
		}
		return nil, out, nil
	})
}

type createSpecialProjectIn struct {
	ProjectID            int64    `json:"project_id"`
	ProjectToken         string   `json:"project_token" jsonschema:"unique DAG-node id; blank -> 400, duplicate -> 409 (no auto-suffix)"`
	CountryTag           string   `json:"country_tag,omitempty"`
	Specialization       string   `json:"specialization" jsonschema:"required specialization/track id; create the track first"`
	NameEn               string   `json:"name_en,omitempty"`
	NameZh               string   `json:"name_zh,omitempty"`
	DescEn               string   `json:"desc_en,omitempty"`
	DescZh               string   `json:"desc_zh,omitempty"`
	Icon                 string   `json:"icon,omitempty"`
	Complexity           string   `json:"complexity,omitempty" jsonschema:"one positive integer or script constant such as sp_complexity.medium"`
	PrototypeTime        string   `json:"prototype_time,omitempty" jsonschema:"one positive integer or script constant such as sp_time.prototype.medium"`
	SpecialProjectParent []string `json:"special_project_parent,omitempty" jsonschema:"parent project_tokens = the DAG prerequisite edges (soft string refs)"`
	Allowed              string   `json:"allowed,omitempty" jsonschema:"RAW trigger block"`
	Visible              string   `json:"visible,omitempty" jsonschema:"RAW trigger block"`
	Available            string   `json:"available,omitempty" jsonschema:"RAW trigger block"`
	ResourceCost         string   `json:"resource_cost,omitempty" jsonschema:"HOI4 block body containing resources = { steel = 10 }; validated before save"`
	ProjectOutput        string   `json:"project_output,omitempty" jsonschema:"HOI4 output/effect block body; validated before save; rewards go here"`
	X                    int      `json:"x,omitempty"`
	Y                    int      `json:"y,omitempty"`
}

type createSpecialProjectOut struct {
	SpecialProjectDBID int64  `json:"special_project_db_id"`
	ProjectToken       string `json:"project_token"`
	Specialization     string `json:"specialization"`
}

func registerCreateSpecialProject(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_special_project",
		Description: "Create a special-project DAG node (secret-weapons research project). DLC: " +
			"Götterdämmerung. specialization is required. special_project_parent is the list of " +
			"parent project tokens. complexity and prototype_time accept one positive integer or " +
			"script constant; resource_cost and project_output accept validated HOI4 block bodies. " +
			"There is no modifier field; rewards go through project_output. Returns " +
			"special_project_db_id, project_token, specialization.",
	}, func(ctx context.Context, _ *CallToolRequest, in createSpecialProjectIn) (*CallToolResult, createSpecialProjectOut, error) {
		body := map[string]any{
			"project_token":          in.ProjectToken,
			"country_tag":            in.CountryTag,
			"specialization":         in.Specialization,
			"name_en":                in.NameEn,
			"name_zh":                in.NameZh,
			"desc_en":                in.DescEn,
			"desc_zh":                in.DescZh,
			"icon":                   in.Icon,
			"complexity":             in.Complexity,
			"prototype_time":         in.PrototypeTime,
			"special_project_parent": orEmptySlice(in.SpecialProjectParent),
			"allowed":                in.Allowed,
			"visible":                in.Visible,
			"available":              in.Available,
			"resource_cost":          in.ResourceCost,
			"project_output":         in.ProjectOutput,
			"x":                      in.X,
			"y":                      in.Y,
		}
		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/special-projects/projects", in.ProjectID), body)
		if err != nil {
			return nil, createSpecialProjectOut{}, err
		}
		return nil, createSpecialProjectOut{
			SpecialProjectDBID: int64Field(resp, "id"),
			ProjectToken:       stringField(resp, "project_token"),
			Specialization:     stringField(resp, "specialization"),
		}, nil
	})
}

type createAutonomyStateIn struct {
	ProjectID         int64          `json:"project_id"`
	AutonomyID        string         `json:"autonomy_id" jsonschema:"script id (required); a project-scoped collision AUTO-SUFFIXES (_2/_3...) and surfaces an advisory in warnings"`
	CountryTag        string         `json:"country_tag,omitempty"`
	NameEn            string         `json:"name_en,omitempty"`
	NameZh            string         `json:"name_zh,omitempty"`
	IsPuppet          bool           `json:"is_puppet,omitempty"`
	IsDefault         bool           `json:"is_default,omitempty"`
	ManpowerInfluence *float64       `json:"manpower_influence,omitempty"`
	MinFreedomLevel   int            `json:"min_freedom_level,omitempty" jsonschema:"stored as freedom x 1000 (freedom 0.25 -> pass 250); export divides it back"`
	Rule              string         `json:"rule,omitempty" jsonschema:"RAW autonomy-rule block"`
	Modifier          map[string]any `json:"modifier,omitempty" jsonschema:"DICT {stat: value} (like ideas), NOT a raw string"`
	AiWeights         map[string]any `json:"ai_weights,omitempty" jsonschema:"maps each HOI4 weight key to RAW inner script text, e.g. {ai_subject_wants_higher: 'factor = 1.0'}"`
}

type createAutonomyStateOut struct {
	AutonomyDBID int64    `json:"autonomy_db_id"`
	AutonomyID   string   `json:"autonomy_id"`
	Warnings     []string `json:"warnings,omitempty"`
}

func registerCreateAutonomyState(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_autonomy_state",
		Description: "Create an autonomy/subject level (the puppet-rule + modifier bundle). DLC: " +
			"Together for Victory. Unlike the other three creates in this cluster, a project-scoped " +
			"autonomy_id collision is NOT a 409 — it AUTO-SUFFIXES and surfaces an advisory in " +
			"warnings (read it back). modifier is a DICT {stat: value}, NOT a raw string. TRAP: " +
			"min_freedom_level is stored as freedom x 1000. Returns autonomy_db_id, autonomy_id, warnings.",
	}, func(ctx context.Context, _ *CallToolRequest, in createAutonomyStateIn) (*CallToolResult, createAutonomyStateOut, error) {
		body := map[string]any{
			"autonomy_id":        in.AutonomyID,
			"country_tag":        in.CountryTag,
			"name_en":            in.NameEn,
			"name_zh":            in.NameZh,
			"is_puppet":          in.IsPuppet,
			"is_default":         in.IsDefault,
			"manpower_influence": in.ManpowerInfluence,
			"min_freedom_level":  in.MinFreedomLevel,
			"rule":               in.Rule,
			"modifier":           orEmptyMap(in.Modifier),
			"ai_weights":         orEmptyMap(in.AiWeights),
		}
		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/autonomy-states", in.ProjectID), body)
		if err != nil {
			return nil, createAutonomyStateOut{}, err
		}
		return nil, createAutonomyStateOut{
			AutonomyDBID: int64Field(resp, "id"),
			AutonomyID:   stringField(resp, "autonomy_id"),
			Warnings:     stringSliceField(resp, "warnings"),
		}, nil
	})
}
