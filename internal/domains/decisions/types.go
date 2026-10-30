package decisions

import (
	"encoding/json"
	"strings"
	"time"

	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

type DecisionCategory struct {
	ID                int64             `json:"id"`
	ProjectID         int64             `json:"project_id"`
	CategoryID        string            `json:"category_id"`
	Icon              string            `json:"icon"`
	Picture           string            `json:"picture"`
	Priority          int               `json:"priority"`
	Allowed           string            `json:"allowed"`
	Visible           string            `json:"visible"`
	OnMapArea         string            `json:"on_map_area"`
	RawExtras         map[string]any    `json:"raw_extras"`
	CountryTag        string            `json:"country_tag"`
	ScopeKind         string            `json:"scope_kind"`
	SharedCountryTags []string          `json:"shared_country_tags"`
	NameEn            string            `json:"name_en"`
	NameZh            string            `json:"name_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	DescriptionLocs   map[string]string `json:"description_locs"`
	Visibility        string            `json:"visibility"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
	DecisionCount     int64             `json:"decision_count"`
	Warnings          []string          `json:"warnings"`
}

type Decision struct {
	ID                 int64             `json:"id"`
	ProjectID          int64             `json:"project_id"`
	CategoryID         int64             `json:"category_id"`
	DecisionID         string            `json:"decision_id"`
	Icon               string            `json:"icon"`
	Allowed            string            `json:"allowed"`
	Available          string            `json:"available"`
	Visible            string            `json:"visible"`
	Cost               int               `json:"cost"`
	FireOnlyOnce       bool              `json:"fire_only_once"`
	CancelIfNotVisible bool              `json:"cancel_if_not_visible"`
	DaysRemove         int               `json:"days_remove"`
	DaysReEnable       int               `json:"days_re_enable"`
	CompleteEffect     string            `json:"complete_effect"`
	RemoveEffect       string            `json:"remove_effect"`
	CancelTrigger      string            `json:"cancel_trigger"`
	CancelEffect       string            `json:"cancel_effect"`
	Modifier           string            `json:"modifier"`
	Activation         string            `json:"activation"`
	TimeoutEffect      string            `json:"timeout_effect"`
	DaysMissionTimeout int               `json:"days_mission_timeout"`
	SelectableMission  bool              `json:"selectable_mission"`
	IsGood             bool              `json:"is_good"`
	TargetedModifier   map[string]any    `json:"targeted_modifier"`
	Targets            []string          `json:"targets"`
	TargetArray        []string          `json:"target_array"`
	TargetRootTrigger  string            `json:"target_root_trigger"`
	TargetTrigger      string            `json:"target_trigger"`
	StateTarget        string            `json:"state_target"`
	HighlightStates    []string          `json:"highlight_states"`
	OnMapMode          string            `json:"on_map_mode"`
	Priority           int               `json:"priority"`
	CustomCostTrigger  string            `json:"custom_cost_trigger"`
	CustomCostText     string            `json:"custom_cost_text"`
	WarWithOnAdd       string            `json:"war_with_on_add"`
	WarWithOnRemove    string            `json:"war_with_on_remove"`
	AIWillDo           string            `json:"ai_will_do"`
	RawExtras          map[string]any    `json:"raw_extras"`
	CountryTag         string            `json:"country_tag"`
	ScopeKind          string            `json:"scope_kind"`
	SharedCountryTags  []string          `json:"shared_country_tags"`
	NameEn             string            `json:"name_en"`
	NameZh             string            `json:"name_zh"`
	DescEn             string            `json:"desc_en"`
	DescZh             string            `json:"desc_zh"`
	DisplayNameLocs    map[string]string `json:"display_name_locs"`
	DescriptionLocs    map[string]string `json:"description_locs"`
	EffectTooltipLocs  map[string]string `json:"effect_tooltip_locs"`
	Source             string            `json:"source"`
	CreatedAt          string            `json:"created_at"`
	UpdatedAt          string            `json:"updated_at"`
	Warnings           []string          `json:"warnings"`
}

type categoryCreateInput struct {
	CategoryID        *string           `json:"category_id"`
	Icon              string            `json:"icon"`
	Picture           string            `json:"picture"`
	Priority          int               `json:"priority"`
	Allowed           string            `json:"allowed"`
	Visible           string            `json:"visible"`
	OnMapArea         string            `json:"on_map_area"`
	CountryTag        string            `json:"country_tag"`
	ScopeKind         *string           `json:"scope_kind"`
	SharedCountryTags []string          `json:"shared_country_tags"`
	NameEn            string            `json:"name_en"`
	NameZh            string            `json:"name_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	DescriptionLocs   map[string]string `json:"description_locs"`
	Visibility        *string           `json:"visibility"`
}

type categoryPatch struct {
	CategoryID        workbench.Optional[string]            `json:"category_id"`
	Icon              workbench.Optional[string]            `json:"icon"`
	Picture           workbench.Optional[string]            `json:"picture"`
	Priority          workbench.Optional[int]               `json:"priority"`
	Allowed           workbench.Optional[string]            `json:"allowed"`
	Visible           workbench.Optional[string]            `json:"visible"`
	OnMapArea         workbench.Optional[string]            `json:"on_map_area"`
	CountryTag        workbench.Optional[string]            `json:"country_tag"`
	ScopeKind         workbench.Optional[string]            `json:"scope_kind"`
	SharedCountryTags workbench.Optional[[]string]          `json:"shared_country_tags"`
	NameEn            workbench.Optional[string]            `json:"name_en"`
	NameZh            workbench.Optional[string]            `json:"name_zh"`
	DescEn            workbench.Optional[string]            `json:"desc_en"`
	DescZh            workbench.Optional[string]            `json:"desc_zh"`
	DisplayNameLocs   workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs   workbench.Optional[map[string]string] `json:"description_locs"`
	Visibility        workbench.Optional[string]            `json:"visibility"`
}

type decisionCreateInput struct {
	CategoryID         *int64            `json:"category_id"`
	DecisionID         *string           `json:"decision_id"`
	Icon               string            `json:"icon"`
	Allowed            string            `json:"allowed"`
	Available          string            `json:"available"`
	Visible            string            `json:"visible"`
	Cost               int               `json:"cost"`
	FireOnlyOnce       bool              `json:"fire_only_once"`
	CancelIfNotVisible bool              `json:"cancel_if_not_visible"`
	DaysRemove         int               `json:"days_remove"`
	DaysReEnable       int               `json:"days_re_enable"`
	CompleteEffect     string            `json:"complete_effect"`
	RemoveEffect       string            `json:"remove_effect"`
	CancelTrigger      string            `json:"cancel_trigger"`
	CancelEffect       string            `json:"cancel_effect"`
	Modifier           string            `json:"modifier"`
	Activation         string            `json:"activation"`
	TimeoutEffect      string            `json:"timeout_effect"`
	DaysMissionTimeout int               `json:"days_mission_timeout"`
	SelectableMission  bool              `json:"selectable_mission"`
	IsGood             bool              `json:"is_good"`
	TargetedModifier   map[string]any    `json:"targeted_modifier"`
	Targets            []string          `json:"targets"`
	TargetArray        []string          `json:"target_array"`
	TargetRootTrigger  string            `json:"target_root_trigger"`
	TargetTrigger      string            `json:"target_trigger"`
	StateTarget        string            `json:"state_target"`
	HighlightStates    []string          `json:"highlight_states"`
	OnMapMode          string            `json:"on_map_mode"`
	Priority           int               `json:"priority"`
	CustomCostTrigger  string            `json:"custom_cost_trigger"`
	CustomCostText     string            `json:"custom_cost_text"`
	WarWithOnAdd       string            `json:"war_with_on_add"`
	WarWithOnRemove    string            `json:"war_with_on_remove"`
	AIWillDo           string            `json:"ai_will_do"`
	CountryTag         string            `json:"country_tag"`
	ScopeKind          *string           `json:"scope_kind"`
	SharedCountryTags  []string          `json:"shared_country_tags"`
	NameEn             string            `json:"name_en"`
	NameZh             string            `json:"name_zh"`
	DescEn             string            `json:"desc_en"`
	DescZh             string            `json:"desc_zh"`
	DisplayNameLocs    map[string]string `json:"display_name_locs"`
	DescriptionLocs    map[string]string `json:"description_locs"`
	EffectTooltipLocs  map[string]string `json:"effect_tooltip_locs"`
	Source             *string           `json:"source"`
}

func (in *decisionCreateInput) normalize() error {
	if in.CategoryID == nil {
		return inputError("category_id is required")
	}
	if in.DecisionID == nil {
		return inputError("decision_id is required")
	}
	if in.TargetedModifier == nil {
		in.TargetedModifier = map[string]any{}
	}
	if in.Targets == nil {
		in.Targets = []string{}
	}
	if in.TargetArray == nil {
		in.TargetArray = []string{}
	}
	if in.HighlightStates == nil {
		in.HighlightStates = []string{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	if in.EffectTooltipLocs == nil {
		in.EffectTooltipLocs = map[string]string{}
	}
	return canonicalDecisionCreate(in)
}

type decisionPatch struct {
	CategoryID         workbench.Optional[int64]             `json:"category_id"`
	DecisionID         workbench.Optional[string]            `json:"decision_id"`
	Icon               workbench.Optional[string]            `json:"icon"`
	Allowed            workbench.Optional[string]            `json:"allowed"`
	Available          workbench.Optional[string]            `json:"available"`
	Visible            workbench.Optional[string]            `json:"visible"`
	Cost               workbench.Optional[int]               `json:"cost"`
	FireOnlyOnce       workbench.Optional[bool]              `json:"fire_only_once"`
	CancelIfNotVisible workbench.Optional[bool]              `json:"cancel_if_not_visible"`
	DaysRemove         workbench.Optional[int]               `json:"days_remove"`
	DaysReEnable       workbench.Optional[int]               `json:"days_re_enable"`
	CompleteEffect     workbench.Optional[string]            `json:"complete_effect"`
	RemoveEffect       workbench.Optional[string]            `json:"remove_effect"`
	CancelTrigger      workbench.Optional[string]            `json:"cancel_trigger"`
	CancelEffect       workbench.Optional[string]            `json:"cancel_effect"`
	Modifier           workbench.Optional[string]            `json:"modifier"`
	Activation         workbench.Optional[string]            `json:"activation"`
	TimeoutEffect      workbench.Optional[string]            `json:"timeout_effect"`
	DaysMissionTimeout workbench.Optional[int]               `json:"days_mission_timeout"`
	SelectableMission  workbench.Optional[bool]              `json:"selectable_mission"`
	IsGood             workbench.Optional[bool]              `json:"is_good"`
	TargetedModifier   workbench.Optional[map[string]any]    `json:"targeted_modifier"`
	Targets            workbench.Optional[[]string]          `json:"targets"`
	TargetArray        workbench.Optional[[]string]          `json:"target_array"`
	TargetRootTrigger  workbench.Optional[string]            `json:"target_root_trigger"`
	TargetTrigger      workbench.Optional[string]            `json:"target_trigger"`
	StateTarget        workbench.Optional[string]            `json:"state_target"`
	HighlightStates    workbench.Optional[[]string]          `json:"highlight_states"`
	OnMapMode          workbench.Optional[string]            `json:"on_map_mode"`
	Priority           workbench.Optional[int]               `json:"priority"`
	CustomCostTrigger  workbench.Optional[string]            `json:"custom_cost_trigger"`
	CustomCostText     workbench.Optional[string]            `json:"custom_cost_text"`
	WarWithOnAdd       workbench.Optional[string]            `json:"war_with_on_add"`
	WarWithOnRemove    workbench.Optional[string]            `json:"war_with_on_remove"`
	AIWillDo           workbench.Optional[string]            `json:"ai_will_do"`
	CountryTag         workbench.Optional[string]            `json:"country_tag"`
	ScopeKind          workbench.Optional[string]            `json:"scope_kind"`
	SharedCountryTags  workbench.Optional[[]string]          `json:"shared_country_tags"`
	NameEn             workbench.Optional[string]            `json:"name_en"`
	NameZh             workbench.Optional[string]            `json:"name_zh"`
	DescEn             workbench.Optional[string]            `json:"desc_en"`
	DescZh             workbench.Optional[string]            `json:"desc_zh"`
	DisplayNameLocs    workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs    workbench.Optional[map[string]string] `json:"description_locs"`
	EffectTooltipLocs  workbench.Optional[map[string]string] `json:"effect_tooltip_locs"`
	RawExtras          workbench.Optional[map[string]any]    `json:"raw_extras"`
	Source             workbench.Optional[string]            `json:"source"`
}

func text(v any) string { return textOr(v, "") }

func textOr(v any, def string) string {
	switch s := v.(type) {
	case nil:
		return def
	case string:
		return s
	case json.Number:
		return s.String()
	}
	return def
}

func intOf(v any) int { return int(project.Int64(v)) }

func boolOf(v any) bool { b, _ := v.(bool); return b }

func stringsOf(v any) []string {
	switch list := v.(type) {
	case []string:
		return append([]string{}, list...)
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func stringsOrEmpty(v any) []string {
	if out := stringsOf(v); out != nil {
		return out
	}
	return []string{}
}

func stringMapOf(v any) map[string]string {
	out := map[string]string{}
	switch m := v.(type) {
	case map[string]string:
		for k, s := range m {
			out[k] = s
		}
	case map[string]any:
		for k, item := range m {
			if s, ok := item.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

func anyMapOf(v any) map[string]any {
	if m, ok := v.(map[string]any); ok && m != nil {
		return m
	}
	return map[string]any{}
}

func isoNaive(v any) string {
	raw := text(v)
	if raw == "" {
		return ""
	}
	for _, layout := range []string{"2006-01-02T15:04:05", time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, strings.TrimSpace(raw)); err == nil {
			t = t.UTC()
			if t.Nanosecond() == 0 {
				return t.Format("2006-01-02T15:04:05")
			}
			return t.Format("2006-01-02T15:04:05.000000")
		}
	}
	return raw
}

func stored(v any) any {
	switch value := v.(type) {
	case int:
		return int64(value)
	case []string:
		out := make([]any, len(value))
		for i, s := range value {
			out[i] = s
		}
		return out
	case map[string]string:
		out := make(map[string]any, len(value))
		for k, s := range value {
			out[k] = s
		}
		return out
	}
	return v
}

func setOptional[T any](sets map[string]any, column string, o workbench.Optional[T]) {
	if !o.Set {
		return
	}
	if o.Value == nil {
		sets[column] = nil
		return
	}
	sets[column] = stored(any(*o.Value))
}

func apply(sets map[string]any) func(project.Row) {
	return func(row project.Row) {
		for column, value := range sets {
			row[column] = value
		}
	}
}
