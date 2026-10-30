package preview

import (
	"fmt"
	"math/big"
	"regexp"
	"sort"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/importsvc/ideas"
	"haganeworks/internal/hoi4/importsvc/trees"
	"haganeworks/internal/hoi4/paradox"
)

func normalize(v any) any {
	switch t := v.(type) {
	case nil, string, bool, float64:
		return t
	case *OD:
		return t
	case *big.Int:
		return t
	case int:
		return big.NewInt(int64(t))
	case int64:
		return big.NewInt(t)
	case paradox.Scalar:
		switch t.Kind {
		case paradox.ScalarInt:
			return t.Int
		case paradox.ScalarFloat:
			return t.Float
		default:
			return t.Str
		}
	case []any:
		out := make([]any, 0, len(t))
		for _, x := range t {
			out = append(out, normalize(x))
		}
		return out
	case []string:
		out := make([]any, 0, len(t))
		for _, x := range t {
			out = append(out, x)
		}
		return out
	case [][]string:
		out := make([]any, 0, len(t))
		for _, x := range t {
			out = append(out, normalize(x))
		}
		return out
	case []map[string]any:
		out := make([]any, 0, len(t))
		for _, x := range t {
			out = append(out, normalize(x))
		}
		return out
	case map[string]any:
		od := NewOD()
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			od.Set(k, normalize(t[k]))
		}
		return od
	case *importsvc.RawExtras:
		od := NewOD()
		keys := append([]string(nil), t.Keys...)
		sort.Strings(keys)
		for _, k := range keys {
			val, _ := t.Get(k)
			od.Set(k, normalize(val))
		}
		return od
	case *trees.RawDict:
		od := NewOD()
		keys := append([]string(nil), t.Keys...)
		sort.Strings(keys)
		for _, k := range keys {
			val, _ := t.Get(k)
			od.Set(k, normalize(val))
		}
		return od
	case *ideas.Dict:
		od := NewOD()
		keys := append([]string(nil), t.Keys()...)
		sort.Strings(keys)
		for _, k := range keys {
			val, _ := t.Get(k)
			od.Set(k, normalize(val))
		}
		return od
	case map[string]float64:
		od := NewOD()
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			od.Set(k, t[k])
		}
		return od
	}
	panic(fmt.Sprintf("normalize: unexpected type %T", v))
}

type fieldDef struct {
	name string
	def  func() any
}

func sv(s string) func() any  { return func() any { return s } }
func bv(b bool) func() any    { return func() any { return b } }
func iv(n int64) func() any   { return func() any { return big.NewInt(n) } }
func fv(f float64) func() any { return func() any { return f } }
func nilv() func() any        { return func() any { return nil } }
func listv() func() any       { return func() any { return []any{} } }
func dictv() func() any       { return func() any { return NewOD() } }

var characterFields = []fieldDef{
	{"tag", sv("")}, {"name", sv("")}, {"gender", sv("")},
	{"can_be_captured", bv(true)}, {"role_type", sv("advisor")},
	{"traits", listv()}, {"skill", iv(1)}, {"attack_skill", iv(1)},
	{"defense_skill", iv(1)}, {"planning_skill", iv(1)},
	{"logistics_skill", iv(1)}, {"maneuvering_skill", iv(1)},
	{"coordination_skill", iv(1)}, {"military_visible", sv("")},
	{"scientist_data", dictv()}, {"scientist_visible", sv("")},
	{"operative_skill_level", iv(1)}, {"operative_traits", listv()},
	{"operative_nationalities", listv()}, {"operative_available", sv("")},
	{"operative_bg", sv("")}, {"ideology", sv("")}, {"expire", sv("")},
	{"has_advisor_role", bv(false)}, {"advisor_slot", sv("")},
	{"advisor_cost", iv(150)}, {"advisor_idea_token", sv("")},
	{"advisor_traits", listv()}, {"advisor_available", sv("")},
	{"advisor_visible", sv("")}, {"advisor_removal_cost", iv(-1)},
	{"advisor_modifier", sv("")}, {"advisor_on_add", sv("")},
	{"advisor_on_remove", sv("")},
	{"always_show_on_actions_tooltip", nilv()},
	{"raw_extras", dictv()}, {"large_portrait", sv("")},
	{"small_portrait", sv("")},
}

var eventFields = []fieldDef{
	{"event_type", sv("country_event")}, {"event_id", sv("")},
	{"title", sv("")}, {"description", sv("")}, {"picture", sv("")},
	{"major", bv(false)}, {"is_triggered_only", bv(true)},
	{"fire_only_once", bv(false)}, {"hidden", bv(false)},
	{"timeout_days", iv(0)}, {"trigger", sv("")},
	{"mean_time_to_happen", sv("")}, {"immediate", sv("")},
	{"after", sv("")}, {"options", listv()}, {"raw_extras", dictv()},
	{"title_en", sv("")}, {"title_zh", sv("")}, {"desc_en", sv("")},
	{"desc_zh", sv("")}, {"display_name_locs", dictv()},
	{"description_locs", dictv()},
}

var decisionCategoryFields = []fieldDef{
	{"category_id", sv("")}, {"icon", sv("")}, {"picture", sv("")},
	{"priority", iv(0)}, {"on_map_area", sv("")}, {"allowed", sv("")},
	{"visible", sv("")}, {"raw_extras", dictv()}, {"scope_kind", sv("own")},
	{"country_tag", sv("")}, {"shared_country_tags", nilv()},
}

var decisionFields = []fieldDef{
	{"decision_id", sv("")}, {"icon", sv("")}, {"allowed", sv("")},
	{"available", sv("")}, {"visible", sv("")}, {"cost", iv(0)},
	{"fire_only_once", bv(false)}, {"cancel_if_not_visible", bv(false)},
	{"days_remove", iv(0)}, {"days_re_enable", iv(0)},
	{"days_mission_timeout", iv(0)}, {"selectable_mission", bv(false)},
	{"is_good", bv(false)}, {"modifier", sv("")}, {"activation", sv("")},
	{"timeout_effect", sv("")}, {"effect_tooltip_locs", dictv()},
	{"complete_effect", sv("")}, {"remove_effect", sv("")},
	{"cancel_trigger", sv("")}, {"cancel_effect", sv("")},
	{"ai_will_do", sv("")}, {"priority", iv(0)}, {"state_target", sv("")},
	{"targets", listv()}, {"target_array", listv()},
	{"target_root_trigger", sv("")}, {"target_trigger", sv("")},
	{"targeted_modifier", dictv()}, {"custom_cost_trigger", sv("")},
	{"custom_cost_text", sv("")}, {"on_map_mode", sv("")},
	{"highlight_states", listv()}, {"war_with_on_remove", sv("")},
	{"raw_extras", dictv()}, {"source", sv("imported")},
	{"scope_kind", sv("own")}, {"country_tag", sv("")},
	{"shared_country_tags", nilv()},
}

var focusNodeFields = []fieldDef{
	{"focus_id", sv("")}, {"icon", sv("GFX_goal_unknown")},
	{"dynamic_icon", listv()}, {"text_icon", sv("")}, {"x", iv(0)},
	{"y", iv(0)}, {"relative_position_id", sv("")}, {"offset", listv()},
	{"cost", fv(10.0)}, {"prerequisites", listv()},
	{"mutually_exclusive", listv()}, {"allow_branch_ast", dictv()},
	{"available", sv("")}, {"bypass", sv("")}, {"cancel", sv("")},
	{"raw_extras", dictv()}, {"cancel_if_invalid", bv(true)},
	{"continue_if_invalid", bv(false)},
	{"available_if_capitulated", bv(false)},
	{"will_lead_to_war_with", listv()}, {"search_filters", listv()},
	{"select_effect", sv("")}, {"ai_will_do", sv("factor = 1")},
	{"completion_reward", sv("")}, {"effect_tooltip_locs", dictv()},
}

var focusTreeFields = []fieldDef{
	{"tree_id", sv("")}, {"tree_kind", sv("primary")},
	{"is_default", bv(false)}, {"initial_show_position", dictv()},
	{"initial_show_focus", sv("")},
	{"continuous_focus_position_x", iv(50)},
	{"continuous_focus_position_y", iv(1000)},
	{"shared_focus_refs", listv()}, {"scope_kind", sv("own")},
	{"country_tag", nilv()}, {"shared_country_tags", nilv()},
}

var techFields = []fieldDef{
	{"tech_id", sv("")}, {"tech_line", sv("")}, {"research_cost", fv(1.0)},
	{"start_year", iv(1936)}, {"folder", sv("")}, {"position_x", iv(0)},
	{"position_y", iv(0)}, {"position_y_is_explicit", nilv()},
	{"prerequisites", listv()}, {"categories", listv()},
	{"modifiers", dictv()}, {"ai_will_do", sv("factor = 1")},
	{"raw_extras", dictv()}, {"enable_equipments", listv()},
	{"enable_subunits", listv()}, {"enable_equipment_modules", listv()},
	{"enable_building", sv("")}, {"enable_tactic", sv("")},
	{"xor", listv()}, {"dependencies", dictv()},
	{"sub_technologies", listv()}, {"doctrine", bv(false)},
	{"allow", sv("")}, {"allow_branch", sv("")},
	{"on_research_complete", sv("")}, {"on_research_complete_limit", sv("")},
	{"path", listv()}, {"xp_research_type", sv("")},
	{"xp_boost_cost", iv(0)}, {"xp_research_bonus", dictv()},
	{"force_use_small_tech_layout", bv(false)},
}

var ideaFields = []fieldDef{
	{"idea_id", sv("")}, {"idea_type", sv("country")}, {"name", sv("")},
	{"picture", sv("")}, {"modifier", dictv()}, {"ai_will_do", sv("")},
	{"removal_cost", iv(-1)}, {"allowed", sv("")},
	{"allowed_civil_war", sv("")}, {"research_bonus", dictv()},
	{"designer_traits", listv()}, {"available", sv("")},
	{"visible", sv("")}, {"allowed_to_remove", sv("")}, {"cancel", sv("")},
	{"do_effect", sv("")}, {"on_add", sv("")}, {"on_remove", sv("")},
	{"cancel_if_invalid", bv(true)}, {"targeted_modifier", listv()},
	{"equipment_bonus", dictv()}, {"rule", dictv()}, {"level", iv(0)},
	{"ledger", sv("")}, {"cost", iv(0)}, {"raw_extras", dictv()},
	{"category_flags", nilv()},
}

type srcGet func(key string) (any, bool)

func mapSrc(m map[string]any) srcGet {
	return func(k string) (any, bool) { v, ok := m[k]; return v, ok }
}

func ideaSrc(d *ideas.Dict) srcGet {
	return func(k string) (any, bool) { return d.Get(k) }
}

func buildEnt(fields []fieldDef, src srcGet) *OD {
	ent := NewOD()
	for _, f := range fields {
		if v, ok := src(f.name); ok {
			ent.Set(f.name, normalize(v))
		} else {
			ent.Set(f.name, f.def())
		}
	}
	return ent
}

func applyVerdict(ent *OD, src srcGet) {
	v, ok := src("_scope_verdict")
	if !ok {
		return
	}
	verdict, ok := v.(importsvc.ScopeVerdict)
	if !ok {
		return
	}
	kind, tag, shared := verdict.StorageTuple()
	ent.Set("scope_kind", kind)
	ent.Set("country_tag", tag)
	if shared == nil {
		ent.Set("shared_country_tags", nil)
	} else {
		lst := make([]any, 0, len(shared))
		for _, t := range shared {
			lst = append(lst, t)
		}
		ent.Set("shared_country_tags", lst)
	}
}

var tagRe = regexp.MustCompile(`^[A-Za-z]{3}_`)

func deriveCountryTag(charTag string) string {
	if tagRe.MatchString(charTag) {
		return pyUpper(charTag[:3])
	}
	return "GEN"
}

func attr(ent *OD, k string) any { v, _ := ent.Get(k); return v }

func getStr(ent *OD, k string) string {
	v := attr(ent, k)
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	return pyStr(v)
}

func orEmptyStr(v any) string {
	if v == nil || !pyTruthy(v) {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return pyStr(v)
}

func getList(ent *OD, k string) []any {
	if l, ok := attr(ent, k).([]any); ok {
		return l
	}
	return nil
}

func getOD(ent *OD, k string) *OD {
	if d, ok := attr(ent, k).(*OD); ok {
		return d
	}
	return nil
}

func asOD(v any) *OD {
	if d, ok := v.(*OD); ok {
		return d
	}
	return nil
}

func asList(v any) []any {
	if l, ok := v.([]any); ok {
		return l
	}
	return nil
}
