package lint

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type LogSignature struct {
	Module string
	Body   string
}

type LogEntry struct {
	Sig       LogSignature
	Bucket    string
	SubBucket string
	Label     string
	RawSample string
	RawCount  int
}

var logLineRe = regexp.MustCompile(`^\[([^\]]*)\]\[([^\]]*)\]\[([^\]]*)\]:\s*(.*)$`)

type logSigRaw struct {
	sig LogSignature
	raw string
}

func iterLogSignatures(text string) []logSigRaw {
	var out []logSigRaw
	var prevSig *LogSignature
	var prevRaw []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\n")
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if m := logLineRe.FindStringSubmatch(line); m != nil {
			if prevSig != nil {
				out = append(out, logSigRaw{*prevSig, strings.Join(prevRaw, "\n")})
			}
			prevSig = &LogSignature{Module: strings.TrimSpace(m[3]), Body: strings.TrimSpace(m[4])}
			prevRaw = []string{line}
		} else {
			if prevSig == nil {
				continue
			}
			prevRaw = append(prevRaw, line)
		}
	}
	if prevSig != nil {
		out = append(out, logSigRaw{*prevSig, strings.Join(prevRaw, "\n")})
	}
	return out
}

type logClassifier struct {
	re     *regexp.Regexp
	bucket string
	sub    string
	label  string
}

var logClassifiers = []logClassifier{

	{regexp.MustCompile(`Non assign trigger is not enclosed`), "D_bug5_fixed", "bug5_non_assign_trigger", "raw_extras cancel block cascade (Bug 5)"},
	{regexp.MustCompile(`Non assign effect is not enclosed`), "D_bug5_fixed", "bug5_non_assign_effect", "raw_extras effect block cascade (Bug 5)"},
	{regexp.MustCompile(`Unexpected token: =,`), "D_bug5_fixed", "bug5_unexpected_equal", "raw_extras = cascade (Bug 5)"},
	{regexp.MustCompile(`Unexpected token: \d[\d.]*,`), "D_bug5_fixed", "bug5_unexpected_number", "raw_extras numeric cascade (Bug 5)"},
	{regexp.MustCompile(`Unknown modifier: =,`), "D_bug5_fixed", "bug5_unknown_modifier_eq", "raw_extras stray = in modifier block (Bug 5 variant)"},
	{regexp.MustCompile(`Unknown modifier: ,,`), "D_bug5_fixed", "bug5_unknown_modifier_comma", "raw_extras stray , in modifier block (Bug 5 variant)"},
	{regexp.MustCompile(`Not a valid compare token in trigger '='`), "D_bug5_fixed", "bug5_invalid_compare_token", "event trigger block malformed `=` (Bug 5 variant on events)"},
	{regexp.MustCompile(`Expected hidden_modifier to be followed by a block`), "D_bug5_fixed", "bug5_hidden_modifier_no_block", "hidden_modifier emitted as scalar, not block (Bug 5 variant)"},
	{regexp.MustCompile(`Malformed token: @`), "D_bug5_fixed", "bug5_malformed_at_var", "@var reference not resolved in export (raw_extras / resolver gap)"},
	{regexp.MustCompile(`Malformed token: [A-Za-z][\w]*\.`), "D_bug5_fixed", "bug5_malformed_event_ns_token", "event namespace id split by round-trip (Bug 5 variant)"},
	{regexp.MustCompile(`Malformed token: [A-Za-z_][\w]*,`), "D_bug5_fixed", "bug5_malformed_generic_token", "identifier emitted as stray token (round-trip / raw_extras)"},
	{regexp.MustCompile(`Unexpected token: country_event,`), "D_bug5_fixed", "bug5_unexpected_country_event", "country_event shorthand broken in events (Bug 5 variant)"},
	{regexp.MustCompile(`Unexpected token: news_event,`), "D_bug5_fixed", "bug5_unexpected_news_event", "news_event shorthand broken in events (Bug 5 variant)"},
	{regexp.MustCompile(`Unexpected token: state_event,`), "D_bug5_fixed", "bug5_unexpected_state_event", "state_event shorthand broken in events (Bug 5 variant)"},
	{regexp.MustCompile(`Unexpected token: [A-Za-z_][\w]*, near line: \d+" in file: "common/decisions/`), "D_bug5_fixed", "bug5_unexpected_decision_id", "decision id as stray token (decision shorthand round-trip, Bug 5 variant)"},
	{regexp.MustCompile(`Unexpected token: [A-Za-z_][\w]*, near line: \d+" in file: "events/`), "D_bug5_fixed", "bug5_unexpected_event_id", "event id as stray token (event shorthand round-trip, Bug 5 variant)"},
	{regexp.MustCompile(`Unexpected token: [A-Za-z_][\w]*, near line: \d+" in file: "common/ideas/`), "D_bug5_fixed", "bug5_unexpected_idea_id", "idea id as stray token (Bug 5 variant)"},
	{regexp.MustCompile(`Unknown modifier: '[^']+'`), "D_bug5_fixed", "bug5_unknown_modifier_quoted", "modifier name quoted/malformed (raw_extras roundtrip)"},
	{regexp.MustCompile(`Unknown modifier: [()]`), "D_bug5_fixed", "bug5_unknown_modifier_paren", "stray `(` or `)` in modifier block (Bug 5 variant)"},
	{regexp.MustCompile(`Unexpected token: >,`), "D_bug5_fixed", "bug5_unexpected_gt", "stray `>` from compare-op emitted wrong (Bug 5 variant)"},
	{regexp.MustCompile(`Unexpected token: [A-Za-z_][\w]*, near line: \d+" in file: "common/national_focus/`), "D_bug5_fixed", "bug5_unexpected_focus_trigger", "focus trigger token malformed (Bug 5 variant)"},
	{regexp.MustCompile(`Invalid group: slots`), "D_bug5_fixed", "bug5_invalid_group_slots", "idea slots field malformed (Bug 5 variant on ideas)"},
	{regexp.MustCompile(`Unknown category for : `), "D_bug5_fixed", "bug5_idea_unknown_category", "idea category field malformed in export (Bug 5 variant)"},
	{regexp.MustCompile(`Unknown modifier: ',`), "D_bug5_fixed", "bug5_unknown_modifier_squote", "stray single-quote ' in modifier block (Bug 5 variant)"},
	{regexp.MustCompile(`Error: unexpected token in file: "events/`), "D_bug5_fixed", "bug5_event_unquoted_string", "event title/desc unquoted string with space ( the ) leaked (Bug 1 family — DLC-name-like unquoted)"},
	{regexp.MustCompile(`Error reading division template`), "D_bug5_fixed", "bug5_division_template_error", "division_template malformed in history file (Bug 5 variant)"},
	{regexp.MustCompile(`Unknown modifier: [A-Za-z_][\w]*`), "D_bug5_fixed", "bug5_unknown_modifier_name", "unknown modifier name (tech/idea raw_extras roundtrip)"},
	{regexp.MustCompile(`Unknown building: building`), "D_bug5_fixed", "bug5_unknown_building_keyword", "tech modifier block emitted `building = building` (Bug 5 variant)"},

	{regexp.MustCompile(`has invalid folder`), "C_gap", "technology_tags_folder", "tech folder undefined (common/technology_tags/ workbench missing)"},
	{regexp.MustCompile(`has invalid category`), "C_gap", "technology_tags_category", "tech category undefined (common/technology_tags/ workbench missing)"},
	{regexp.MustCompile(`Failed to create id \d`), "C_gap", "replace_path_event_id", "event id collision w/ vanilla (replace_path strategy missing)"},
	{regexp.MustCompile(`Reverse id lookup:`), "C_gap", "replace_path_event_id_reverse", "event id reverse-lookup (replace_path strategy missing)"},
	{regexp.MustCompile(`Multiple character have the tag`), "C_gap", "replace_path_character", "character dup w/ vanilla (replace_path for common/characters/ missing)"},
	{regexp.MustCompile(`Duplicate idea\.`), "C_gap", "replace_path_idea", "idea dup w/ vanilla (replace_path for common/ideas/ missing)"},
	{regexp.MustCompile(`Duplicate of .* entity added to entity system`), "C_gap", "replace_path_entity", "entity dup (replace_path for common/units/equipment/ missing)"},
	{regexp.MustCompile(`Failed to find entity "[^"]+" for attachment`), "C_gap", "entity_attachment_mod_inherent", "mod unit-pack entity attachment missing (mod-inherent)"},
	{regexp.MustCompile(`Couldn't find texture file`), "C_gap", "asset_texture_missing", "texture PNG missing (asset fallback architecture gap)"},
	{regexp.MustCompile(`Texture Handler encountered missing texture file`), "C_gap", "asset_texture_handler", "texture handler missing file (asset fallback architecture gap)"},
	{regexp.MustCompile(`Invalid scientist trait`), "C_gap", "scientist_specialization", "scientist trait to specialization mapping"},
	{regexp.MustCompile(`Unexpected token: instance`), "C_gap", "character_instance_token", "conditional character instance token"},
	{regexp.MustCompile(`equipmentdesigner/graphic_db`), "C_gap", "equipmentdesigner_graphic_db", "equipmentdesigner graphic_db custom-tag scope (mod-inherent)"},
	{regexp.MustCompile(`does not match any MIO in database`), "C_gap", "mio_missing", "Military Industrial Organisation is missing"},
	{regexp.MustCompile(`Unknown equipment type:`), "C_gap", "unknown_equipment_type", "custom equipment type is missing"},
	{regexp.MustCompile(`has loc key collisions`), "C_gap", "loc_key_collision", "localisation key collision with vanilla"},
	{regexp.MustCompile(`Invalid trigger '[^']+' in`), "C_gap", "scripted_trigger_missing", "custom scripted trigger ref (common/scripted_triggers/ workbench missing)"},
	{regexp.MustCompile(`Trigger failed to validate:`), "C_gap", "scripted_trigger_validation", "scripted trigger/localisation failed validate (scripted_* workbench gap)"},
	{regexp.MustCompile(`Unknown trigger-type:`), "C_gap", "unknown_trigger_type", "unknown trigger-type (scripted_triggers workbench missing)"},
	{regexp.MustCompile(`Invalid effect '[^']+' in`), "C_gap", "scripted_effect_missing", "custom scripted effect ref (common/scripted_effects/ workbench missing)"},
	{regexp.MustCompile(`Unknown effect-type:`), "C_gap", "unknown_effect_type", "unknown effect-type (scripted_effects workbench missing)"},
	{regexp.MustCompile(`Invalid focus scripted in trigger`), "C_gap", "cross_ref_focus_in_trigger", "has_completed_focus references a missing focus"},
	{regexp.MustCompile(`invalid focus for ai strategy`), "C_gap", "cross_ref_ai_strategy_focus", "ai_strategy focus ref broken (cross_ref focus gate)"},
	{regexp.MustCompile(`Invalid Decision ID for unlock_decision_tooltip`), "C_gap", "cross_ref_decision_tooltip", "unlock_decision_tooltip references a missing decision"},
	{regexp.MustCompile(`Invalid Decision ID for `), "C_gap", "cross_ref_decision_generic", "decision reference is missing"},
	{regexp.MustCompile(`invalid database object for effect/trigger:`), "C_gap", "database_scoped_var", "database_scoped_variables — Paradox feature not modeled"},
	{regexp.MustCompile(`Invalid doctrine`), "C_gap", "tech_doctrine_ref", "has_doctrine trigger refs unknown doctrine (tech_tags gap)"},
	{regexp.MustCompile(`add_tech_bonus: Unknown technology category`), "C_gap", "tech_bonus_category", "add_tech_bonus category unknown (tech_tags category gap)"},
	{regexp.MustCompile(`Invalid category or tech in has_tech_bonus`), "C_gap", "tech_bonus_trigger_category", "has_tech_bonus category unknown (tech_tags category gap)"},
	{regexp.MustCompile(`(?i)Duplicate id \d+`), "C_gap", "replace_path_decision_dup", "decision id duplicate w/ vanilla (replace_path strategy missing)"},
	{regexp.MustCompile(`Several character with the same token`), "C_gap", "replace_path_character_several", "character token dup w/ vanilla (replace_path for common/characters/)"},
	{regexp.MustCompile(`Entity referenced in equipment graphic database does not exist`), "C_gap", "equipment_entity_missing", "equipment_graphic_database entity ref missing (equipment workbench gap)"},
	{regexp.MustCompile(`GFX referenced in equipment graphic database does not exist`), "C_gap", "equipment_gfx_missing", "equipment_graphic_database GFX ref missing (equipment workbench gap)"},
	{regexp.MustCompile(`Invalid [Mm]odule "[^"]+" attached to slot`), "C_gap", "equipment_module_missing", "equipment module ref missing (equipment workbench gap)"},
	{regexp.MustCompile(`Tried to get dynamic token that does not exist`), "C_gap", "dynamic_token_missing", "custom dynamic token missing (localisation/tech gap)"},
	{regexp.MustCompile(`ai_strategy\.cpp`), "C_gap", "ai_strategy_ref_misc", "AI strategy misc ref (AI strategy workbench + cross_ref gap)"},
	{regexp.MustCompile(`Decision scripted with modifier, but no days_remove`), "C_gap", "decision_modifier_no_days_remove", "decision has modifier but no days_remove (mod-inherent in most mods)"},
	{regexp.MustCompile(`Invalid dynamic modifier`), "C_gap", "dynamic_modifier_ref", "dynamic_modifier ref (common/dynamic_modifiers/ workbench missing)"},
	{regexp.MustCompile(`missing static modifier definition`), "C_gap", "static_modifier_missing", "static_modifier def missing (common/static_modifiers/ workbench missing)"},
	{regexp.MustCompile(`set_power_balance_gfx: Unable to find power balance`), "C_gap", "power_balance_missing", "power_balance ref (common/bop/ workbench missing)"},
	{regexp.MustCompile(`has_game_rule: game rule`), "C_gap", "game_rule_missing", "game_rule ref (common/game_rules/ workbench missing)"},
	{regexp.MustCompile(`add_country_leader_role: Invalid country leader trait`), "C_gap", "cross_ref_leader_trait", "country leader role references a missing trait"},
	{regexp.MustCompile(`add_country_leader_role: Invalid ideology`), "C_gap", "ideology_ref_missing", "ideology ref (common/ideologies/ workbench missing)"},
	{regexp.MustCompile(`Invalid Decision Category`), "C_gap", "cross_ref_decision_category", "decision category ref broken (cross_ref)"},
	{regexp.MustCompile(`Invalid focus: .*: focuses`), "C_gap", "bookmark_focus_ref", "bookmark→focus ref (cross_ref / bookmark workbench)"},
	{regexp.MustCompile(`Not a valid value: `), "C_gap", "not_a_valid_value", "ideology/doctrine/category value not recognized (ideology/tech_tags workbenches)"},
	{regexp.MustCompile(`is not a valid ideology`), "C_gap", "ideology_not_valid", "character/leader ideology not in common/ideologies/ (ideology workbench gap)"},
	{regexp.MustCompile(`create_country_leader: Invalid ideology`), "C_gap", "create_country_leader_ideology", "create_country_leader ideology invalid (ideology workbench gap)"},
	{regexp.MustCompile(`Duplicate category\. .* is a duplicate\.`), "C_gap", "replace_path_category_dup", "decision/idea category dup w/ vanilla (replace_path strategy missing)"},
	{regexp.MustCompile(`Invalid subunit category:`), "C_gap", "subunit_category_missing", "subunit_category undefined (common/unit_tags/ workbench missing)"},
	{regexp.MustCompile(`add_power_balance_value: Unable to find power balance`), "C_gap", "power_balance_value_missing", "add_power_balance_value ref (common/bop/ workbench missing)"},
	{regexp.MustCompile(`add_power_balance_modifier: Unable to find power balance`), "C_gap", "power_balance_modifier_missing", "add_power_balance_modifier ref (common/bop/ workbench missing)"},
	{regexp.MustCompile(`remove_power_balance: Unable to find power balance`), "C_gap", "power_balance_remove_missing", "remove_power_balance ref (common/bop/ workbench missing)"},
	{regexp.MustCompile(`Invalid trait for idea`), "C_gap", "idea_trait_ref", "idea references a missing trait"},
	{regexp.MustCompile(`Idea: .* unknown category:`), "C_gap", "idea_category_missing", "idea category undefined (common/idea_categories/ gap)"},
	{regexp.MustCompile(`Invalid decision in has_decision trigger`), "C_gap", "cross_ref_has_decision", "has_decision references a missing decision"},
	{regexp.MustCompile(`on_action .* is not a valid event`), "C_gap", "on_action_event_ref", "on_actions event ref broken (cross_ref event gap on history/on_actions)"},
	{regexp.MustCompile(`Unable to find focus tree with ID:`), "C_gap", "focus_tree_id_missing", "load_focus_tree ref to focus tree not in project (cross_ref / multi-country gap)"},
	{regexp.MustCompile(`create_corps_commander: invalid trait`), "C_gap", "corps_commander_trait_ref", "create_corps_commander references a missing trait"},
	{regexp.MustCompile(`is not in trait DB in game/common/scientist_traits`), "C_gap", "scientist_trait_db_missing", "scientist trait definition is missing"},
	{regexp.MustCompile(`in scientist role, trait`), "C_gap", "scientist_trait_role_missing", "scientist role trait definition is missing"},
	{regexp.MustCompile(`Invalid trait: .* has_allowed_idea_with_trait`), "C_gap", "idea_trait_has_allowed", "has_allowed_idea_with_trait references a missing trait"},
	{regexp.MustCompile(`set_power_balance: Unable to find power balance`), "C_gap", "power_balance_set_missing", "set_power_balance ref (common/bop/ workbench missing)"},
	{regexp.MustCompile(`set_power_balance_gfx: Unable to find side with ID`), "C_gap", "power_balance_side_missing", "set_power_balance_gfx side ref (common/bop/ workbench missing)"},
	{regexp.MustCompile(`remove_power_balance_modifier: Unable to find power balance`), "C_gap", "power_balance_remove_modifier", "remove_power_balance_modifier ref (common/bop/ workbench missing)"},
	{regexp.MustCompile(`initial_show_position references invalid focus`), "C_gap", "focus_tree_initial_show_position", "focus_tree initial_show_position focus id invalid (cross_ref focus)"},
	{regexp.MustCompile(`Error in focus: `), "C_gap", "focus_generic_error", "focus definition error (various — mod-inherent or cross_ref)"},
	{regexp.MustCompile(`Event is missing a title`), "D_triage_tail", "event_missing_title", "event missing title loc key (mod-inherent localisation gap in most mods)"},
	{regexp.MustCompile(`Event is missing a desc`), "D_triage_tail", "event_missing_desc", "event missing desc loc key (mod-inherent localisation gap)"},
	{regexp.MustCompile(`Event has no options`), "D_triage_tail", "event_no_options", "event has no options / hidden event mis-flagged (mod-inherent or export gap)"},
	{regexp.MustCompile(`non-hidden event with no options defined`), "D_triage_tail", "event_non_hidden_no_options", "non-hidden event without options (mod-inherent RT56 WTT event)"},
	{regexp.MustCompile(`does not match any Special Project in database`), "C_gap", "special_project_missing", "special_project (sp:X) ref (common/special_projects/ workbench missing)"},
	{regexp.MustCompile(`Failed to find autonomy_state`), "C_gap", "autonomy_state_missing", "autonomy_state ref (common/autonomous_states/ workbench missing)"},
	{regexp.MustCompile(`Invalid subunit\.`), "C_gap", "invalid_subunit", "invalid subunit (common/unit_tags/ workbench missing)"},
	{regexp.MustCompile(`not a province building type`), "C_gap", "province_building_missing", "province_building type invalid (buildings workbench missing)"},
	{regexp.MustCompile(`Unrecognized equipment upgrade`), "C_gap", "equipment_upgrade_missing", "equipment upgrade unknown (equipment workbench missing)"},
	{regexp.MustCompile(`Unknown technology/doctrine category/folder`), "C_gap", "tech_category_folder_in_effect", "tech category/folder in effect (tech_tags gap)"},
	{regexp.MustCompile(`invalid on_cancel event for start border war`), "C_gap", "border_war_on_cancel", "start_border_war event ref (cross_ref event)"},
	{regexp.MustCompile(`invalid on_win event for start border war`), "C_gap", "border_war_on_win", "start_border_war event ref (cross_ref event)"},
	{regexp.MustCompile(`invalid on_lose event for start border war`), "C_gap", "border_war_on_lose", "start_border_war event ref (cross_ref event)"},
	{regexp.MustCompile(`recruit_character: Unknown character`), "C_gap", "history_recruit_unknown_character", "history file recruit_character references unknown char (γ2 character cross_ref)"},
	{regexp.MustCompile(`Value does not belong to designated Script EnumType`), "C_gap", "script_enum_type", "script_enum value unknown (custom enum coverage gap)"},
	{regexp.MustCompile(`Building type not found`), "C_gap", "building_type_not_found", "building type not found (buildings workbench missing)"},
	{regexp.MustCompile(`remove_country_leader_role: Invalid ideology`), "C_gap", "ideology_ref_remove_leader", "remove_country_leader_role ideology (ideology workbench gap)"},
	{regexp.MustCompile(`Several characters have the name`), "C_gap", "replace_path_character_name_dup", "character name dup w/ vanilla (replace_path for common/characters/)"},
	{regexp.MustCompile(`Duplicate focus name will cause database problems`), "C_gap", "replace_path_focus_dup", "focus name dup w/ vanilla (replace_path for common/national_focus/)"},
	{regexp.MustCompile(`Duplicate decision\. .* is a duplicate\.`), "C_gap", "replace_path_decision_name_dup", "decision name dup w/ vanilla (replace_path for common/decisions/)"},
	{regexp.MustCompile(`recruit_character should only happen in game/history files`), "D_triage_tail", "on_actions_recruit_position", "recruit_character is used outside a history file"},
	{regexp.MustCompile(`common/scripted_localisation`), "C_gap", "scripted_localisation_fail", "scripted_localisation failure (common/scripted_localisation/ gap)"},
	{regexp.MustCompile(`ai_strategy`), "C_gap", "ai_strategy_misc_ref", "ai_strategy misc ref (AI strategy + cross_ref gap)"},

	{regexp.MustCompile(`Expected 'default', a continent name or a country tag`), "D_triage_tail", "graphic_db_tag_invalid", "graphic_db unknown TAG (mod-inherent, equipmentdesigner tail)"},
	{regexp.MustCompile(`Unknown trigger-type:`), "D_triage_tail", "unknown_trigger_type", "unknown trigger-type (could be mod custom or regression — inspect)"},
	{regexp.MustCompile(`Invalid Scope, supported: State\|Country`), "D_triage_tail", "invalid_scope_trigger", "invalid scope on trigger (mostly equipmentdesigner mod-inherent)"},
	{regexp.MustCompile(`Failed to find entity`), "D_triage_tail", "entity_attachment_missing", "entity attachment missing (mod unit-pack inherent)"},
	{regexp.MustCompile(`Technology .* not found`), "D_triage_tail", "technology_not_found", "technology reference missing (may be coverage gap or cross_ref miss)"},
	{regexp.MustCompile(`No political party .* for country leader`), "D_triage_tail", "leader_no_party", "country leader is missing an ideology or party binding"},
	{regexp.MustCompile(`Failed to find faction template`), "D_triage_tail", "faction_template_missing", "faction template empty/missing (export-side gap)"},
	{regexp.MustCompile(`set_technology: Invalid tech`), "D_triage_tail", "history_set_technology_invalid", "history file set_technology = unknown tech (cross_ref or coverage)"},
	{regexp.MustCompile(`set_cosmetic_tag: Invalid`), "D_triage_tail", "history_set_cosmetic_tag_invalid", "history file set_cosmetic_tag unknown (cross_ref gap)"},
	{regexp.MustCompile(`Tried to trigger non-existant event`), "D_triage_tail", "triggered_missing_event", "event ref dangling (cross_ref gap)"},

	{regexp.MustCompile(`does not match previous fontfiles`), "mod_noise", "font_mismatch", "font file line-height mismatch (cosmetic, mod-inherent)"},
	{regexp.MustCompile(`incorrect checksum for DLC`), "mod_noise", "dlc_checksum", "DLC checksum warning (achievement lock only, harmless)"},
	{regexp.MustCompile(`Could not find .* in window`), "mod_noise", "gui_window_widget", "GUI window widget not found (mod-inherent UI override)"},
}

func ClassifyLogBody(body string) (bucket, sub, label string) {
	for _, c := range logClassifiers {
		if c.re.MatchString(body) {
			return c.bucket, c.sub, c.label
		}
	}
	return "D_unknown", "unknown", "Couldn't categorize this line"
}

var logBucketMeta = map[string][2]string{
	"D_unknown":     {"warning", "Couldn't categorize this line"},
	"D_bug5_fixed":  {"error", "Malformed script (export cascade)"},
	"C_gap":         {"warning", "Tool coverage gap"},
	"D_triage_tail": {"warning", "Known long-tail issue"},
	"mod_noise":     {"info", "Harmless noise"},
}

func AnalyzeErrorLog(text string) []LogEntry {
	firstIdx := map[LogSignature]int{}
	var entries []LogEntry
	for _, sr := range iterLogSignatures(text) {
		if i, ok := firstIdx[sr.sig]; ok {
			entries[i].RawCount++
			continue
		}
		bucket, sub, label := ClassifyLogBody(sr.sig.Body)
		firstIdx[sr.sig] = len(entries)
		entries = append(entries, LogEntry{
			Sig: sr.sig, Bucket: bucket, SubBucket: sub, Label: label,
			RawSample: sr.raw, RawCount: 1,
		})
	}
	return entries
}

type LogEntityRef struct {
	Kind       string
	EntityID   string
	Name       *string
	CountryTag *string
}

func (r LogEntityRef) toMap() map[string]any {
	m := map[string]any{"kind": r.Kind, "entity_id": r.EntityID}
	if r.Name != nil {
		m["name"] = *r.Name
	} else {
		m["name"] = nil
	}
	if r.CountryTag != nil {
		m["country_tag"] = *r.CountryTag
	} else {
		m["country_tag"] = nil
	}
	return m
}

type LogProjectIndex struct {
	Focus            map[string]LogEntityRef
	Event            map[string]LogEntityRef
	Decision         map[string]LogEntityRef
	DecisionCategory map[string]LogEntityRef
	Character        map[string]LogEntityRef
	Idea             map[string]LogEntityRef
	Technology       map[string]LogEntityRef
	State            map[string]LogEntityRef
	HistoryFile      map[string]LogEntityRef
}

func NewLogProjectIndex() *LogProjectIndex {
	return &LogProjectIndex{
		Focus: map[string]LogEntityRef{}, Event: map[string]LogEntityRef{},
		Decision: map[string]LogEntityRef{}, DecisionCategory: map[string]LogEntityRef{},
		Character: map[string]LogEntityRef{}, Idea: map[string]LogEntityRef{},
		Technology: map[string]LogEntityRef{}, State: map[string]LogEntityRef{},
		HistoryFile: map[string]LogEntityRef{},
	}
}

func (ix *LogProjectIndex) byKind(kind string) map[string]LogEntityRef {
	switch kind {
	case "event":
		return ix.Event
	case "focus":
		return ix.Focus
	case "decision":
		return ix.Decision
	case "decision_category":
		return ix.DecisionCategory
	case "idea":
		return ix.Idea
	case "technology":
		return ix.Technology
	case "character":
		return ix.Character
	case "history_file":
		return ix.HistoryFile
	case "state":
		return ix.State
	}
	return nil
}

var (
	logFileInLogRe = regexp.MustCompile(`in file:\s*"([^"]+)"`)
	logTokenRe     = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_.]{2,}`)
	logEventIDRe   = regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*\.\d+)\b`)
	logStateIDRe   = regexp.MustCompile(`(?i)\bstate(?:\s+|\s*=\s*|_id\s*[:=]?\s*)?(\d{1,4})\b`)
	logQuotedIDRe  = regexp.MustCompile(`['"]([A-Za-z_][A-Za-z0-9_.]+)['"]`)
)

var logFilePrefixPriority = []struct {
	prefix string
	kinds  []string
}{
	{"events/", []string{"event"}},
	{"common/national_focus/", []string{"focus"}},
	{"common/decisions/categories/", []string{"decision_category"}},
	{"common/decisions/", []string{"decision", "decision_category"}},
	{"common/characters/", []string{"character"}},
	{"common/ideas/", []string{"idea"}},
	{"common/technologies/", []string{"technology"}},
	{"history/states/", []string{"state"}},
	{"history/countries/", []string{"history_file", "character"}},
	{"common/on_actions/", []string{"history_file", "character", "event"}},
}

var logFallbackKindOrder = []string{
	"event", "focus", "decision", "decision_category",
	"idea", "technology", "character", "history_file", "state",
}

var logBodyKeywordBias = []struct {
	re    *regexp.Regexp
	kinds []string
}{
	{regexp.MustCompile(`(?i)\bcharacter\b|recruit_character|country_leader|corps_commander|create_field_marshal`), []string{"character"}},
	{regexp.MustCompile(`(?i)\bfocus\b`), []string{"focus"}},
	{regexp.MustCompile(`(?i)\bdecision\b`), []string{"decision", "decision_category"}},
	{regexp.MustCompile(`(?i)\bidea\b`), []string{"idea"}},
	{regexp.MustCompile(`(?i)\btech(?:nology)?\b`), []string{"technology"}},
	{regexp.MustCompile(`(?i)\bevent\b`), []string{"event"}},
	{regexp.MustCompile(`(?i)\bstate\b`), []string{"state"}},
}

func MapLogEntry(entry LogEntry, ix *LogProjectIndex) (*LogEntityRef, *string) {
	body := entry.Sig.Body
	var filePath *string
	var kindOrder []string

	m := logFileInLogRe.FindStringSubmatch(body)
	if m == nil {
		m = logFileInLogRe.FindStringSubmatch(entry.RawSample)
	}
	if m != nil {
		fp := m[1]
		filePath = &fp
		norm := strings.ReplaceAll(fp, "\\", "/")
		for _, p := range logFilePrefixPriority {
			if strings.Contains(norm, p.prefix) {
				kindOrder = append([]string(nil), p.kinds...)
				break
			}
		}
	}
	if kindOrder == nil {
		for _, b := range logBodyKeywordBias {
			if b.re.MatchString(body) {
				kindOrder = append([]string(nil), b.kinds...)
				break
			}
		}
	}

	for _, em := range logEventIDRe.FindAllStringSubmatch(body, -1) {
		if ref, ok := ix.Event[strings.ToLower(em[1])]; ok {
			return &ref, filePath
		}
	}

	for _, sm := range logStateIDRe.FindAllStringSubmatch(body, -1) {
		if ref, ok := ix.State[sm[1]]; ok {
			return &ref, filePath
		}
	}

	searchKinds := append([]string(nil), kindOrder...)
	for _, k := range logFallbackKindOrder {
		seen := false
		for _, ko := range kindOrder {
			if ko == k {
				seen = true
				break
			}
		}
		if !seen {
			searchKinds = append(searchKinds, k)
		}
	}
	var tokens []string
	tokenSeen := map[string]bool{}
	for _, t := range logTokenRe.FindAllString(body, -1) {
		if !tokenSeen[t] {
			tokenSeen[t] = true
			tokens = append(tokens, t)
		}
	}
	sort.SliceStable(tokens, func(i, j int) bool { return len(tokens[i]) > len(tokens[j]) })
	for _, kind := range searchKinds {
		idmap := ix.byKind(kind)
		if len(idmap) == 0 {
			continue
		}
		for _, tok := range tokens {
			if ref, ok := idmap[strings.ToLower(tok)]; ok {
				return &ref, filePath
			}
		}
	}

	for _, qm := range logQuotedIDRe.FindAllStringSubmatch(body, -1) {
		for _, kind := range searchKinds {
			if ref, ok := ix.byKind(kind)[strings.ToLower(qm[1])]; ok {
				return &ref, filePath
			}
		}
	}
	return nil, filePath
}

func logEntryToFinding(entry LogEntry, ref *LogEntityRef, filePath *string) LintFinding {
	meta, ok := logBucketMeta[entry.Bucket]
	severity, headline := "error", "Error"
	if ok {
		severity, headline = meta[0], meta[1]
	}
	reason := entry.Label
	if entry.RawCount > 1 {
		reason = fmt.Sprintf("%s (logged %d× this run)", entry.Label, entry.RawCount)
	}
	var suggestion string
	switch {
	case ref != nil:
		display := ref.EntityID
		if ref.Name != nil && *ref.Name != "" {
			display = *ref.Name
		}
		suggestion = fmt.Sprintf("Open the %s '%s' and check the field the engine complained about.", ref.Kind, display)
	case entry.Bucket == "D_unknown":
		suggestion = "We couldn't categorize this line — it may be harmless or come from another mod. Skim the raw log line below; act on it only if it names one of your entities."
	case severity == "warning" && entry.Bucket == "C_gap":
		suggestion = "This is a known coverage gap (a file type the editor does not yet model). If the base mod provides it, keep it via an override file."
	case severity == "info":
		suggestion = "No action needed — harmless engine noise."
	default:
		suggestion = "Inspect the raw log line; could not auto-map it to an entity."
	}
	var refMap map[string]any
	if ref != nil {
		refMap = ref.toMap()
	}
	return LintFinding{
		Severity: severity,
		Code:     "log." + entry.Bucket + "." + entry.SubBucket,
		Title:    headline + ": " + entry.Label,
		Reason:   reason, Suggestion: suggestion,
		EntityRef: refMap, File: filePath,
		Detail: map[string]any{
			"module": entry.Sig.Module, "bucket": entry.Bucket,
			"sub_bucket": entry.SubBucket, "raw_line": entry.RawSample,
			"raw_count": entry.RawCount,
		},
	}
}

func BuildLogReportFromIndex(ix *LogProjectIndex, text string) map[string]any {
	entries := AnalyzeErrorLog(text)
	findings := make([]LintFinding, 0, len(entries))
	mapped := 0
	byBucket := map[string]int{}
	linesParsed := 0
	for _, e := range entries {
		ref, fp := MapLogEntry(e, ix)
		if ref != nil {
			mapped++
		}
		findings = append(findings, logEntryToFinding(e, ref, fp))
		byBucket[e.Bucket]++
		linesParsed += e.RawCount
	}
	return LintFindingsReport(findings, map[string]any{
		"lines_parsed":      linesParsed,
		"unique_signatures": len(entries),
		"mapped":            mapped,
		"unmapped":          len(entries) - mapped,
		"by_bucket":         byBucket,
	})
}
