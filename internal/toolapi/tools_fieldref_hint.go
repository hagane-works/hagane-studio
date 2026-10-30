package toolapi

import (
	"context"
	"sort"
	"strings"
)

var toolFieldRef = map[string]string{

	"create_focus_tree":            "focus",
	"update_focus_tree":            "focus",
	"create_focus":                 "focus",
	"update_focus":                 "focus",
	"connect_focus":                "focus",
	"set_focus_mutual_exclusion":   "focus",
	"create_event":                 "events",
	"update_event":                 "events",
	"create_event_namespace":       "events",
	"update_event_namespace":       "events",
	"add_event_option":             "events",
	"create_super_event":           "super-events",
	"update_super_event":           "super-events",
	"create_character":             "characters",
	"update_character":             "characters",
	"assign_character_role":        "characters",
	"create_custom_trait":          "custom-traits",
	"update_custom_trait":          "custom-traits",
	"create_idea":                  "ideas",
	"update_idea":                  "ideas",
	"create_decision":              "decisions",
	"update_decision":              "decisions",
	"create_decision_category":     "decisions",
	"update_decision_category":     "decisions",
	"create_country":               "countries",
	"update_country":               "countries",
	"set_country_history":          "history-files",
	"create_bookmark":              "bookmarks",
	"update_bookmark":              "bookmarks",
	"create_ai_strategy_plan":      "ai-strategy-plans",
	"update_ai_strategy_plan":      "ai-strategy-plans",
	"create_ai_template":           "ai-templates",
	"update_ai_template":           "ai-templates",
	"ensure_project_settings":      "_timeline-and-replace-path",
	"create_dynamic_modifier":      "dynamic-modifiers",
	"update_dynamic_modifier":      "dynamic-modifiers",
	"create_modifier_definition":   "modifier-definitions",
	"update_modifier_definition":   "modifier-definitions",
	"create_opinion_modifier":      "opinion-modifiers",
	"update_opinion_modifier":      "opinion-modifiers",
	"create_wargoal":               "wargoals",
	"update_wargoal":               "wargoals",
	"create_scripted_snippet":      "scripted-snippets",
	"update_scripted_snippet":      "scripted-snippets",
	"create_scripted_localisation": "scripted-localisations",
	"update_scripted_localisation": "scripted-localisations",

	"edit_state":               "states",
	"seed_states_from_vanilla": "states",
	"create_autonomy_state":    "autonomy-states",
	"update_autonomy_state":    "autonomy-states",

	"create_technology":                      "technologies",
	"update_technology":                      "technologies",
	"create_equipment":                       "equipment",
	"update_equipment":                       "equipment",
	"create_equipment_variant":               "equipment",
	"create_sub_unit":                        "sub-units",
	"update_sub_unit":                        "sub-units",
	"create_division_template":               "division-templates",
	"update_division_template":               "division-templates",
	"add_division_deployment":                "division-deployments",
	"update_division_deployment":             "division-deployments",
	"add_air_wing":                           "oob-units",
	"update_air_wing":                        "oob-units",
	"create_air_wing_group":                  "oob-units",
	"update_air_wing_group":                  "oob-units",
	"create_fleet":                           "oob-units",
	"update_fleet":                           "oob-units",
	"add_ship":                               "oob-units",
	"update_ship":                            "oob-units",
	"add_task_force":                         "oob-units",
	"update_task_force":                      "oob-units",
	"create_mio":                             "mios",
	"update_mio":                             "mios",
	"add_mio_trait":                          "mios",
	"create_doctrine":                        "doctrines",
	"update_doctrine":                        "doctrines",
	"confirm_technology_surface_environment": "technologies",
	"create_technology_surface_branch":       "technologies",
	"update_technology_surface_branch":       "technologies",
	"delete_technology_surface_branch":       "technologies",
	"add_technology_surface_node":            "technologies",
	"repair_technology_surface_timeline":     "technologies",
	"create_intelligence_agency":             "intelligence-agencies",
	"update_intelligence_agency":             "intelligence-agencies",
	"add_agency_upgrade":                     "intelligence-agencies",
	"create_specialization":                  "special-projects",
	"update_specialization":                  "special-projects",
	"create_special_project":                 "special-projects",
	"update_special_project":                 "special-projects",
	"validate_special_projects":              "special-projects",

	"create_ui_panel":           "ui-panels",
	"update_ui_panel":           "ui-panels",
	"apply_ui_panel_operations": "ui-panels",
	"validate_ui_panel":         "ui-panels",
	"create_raw_from_template":  "_raw-script-fields",
	"create_raw_file":           "_raw-script-fields",
	"update_raw_file":           "_raw-script-fields",
	"validate_project":          "_validation-tools",
	"export_project":            "_timeline-and-replace-path",
}

var toolFieldRefExempt = map[string]string{
	"create_ideology_root":      "canonical P0 command schema is self-describing; the legacy ideology field doc still describes unsafe generic project writes",
	"create_ideology_subtype":   "canonical P0 command schema is self-describing; the legacy ideology field doc still describes unsafe generic project writes",
	"update_ideology":           "canonical P0 command schema is self-describing; the legacy ideology field doc still describes unsafe generic project writes",
	"preview_delete_ideology":   "two-step safe-delete command takes only a stable target; the legacy ideology field doc is not applicable",
	"apply_delete_ideology":     "single-use safe-delete token only; the legacy ideology field doc is not applicable",
	"discard_ideology_conflict": "explicitly confirmed removal of one invalid registry entry; the legacy ideology field doc is not applicable",
	"create_world_image":        "world from a picture: takes a picture, not entity fields",
	"create_terrain_sketch":     "world from a picture: pixel rules, not entity fields",
	"create_country_split":      "world from a picture: pixel seeds, not entity fields",
	"preview_world_from_images": "world from a picture: map generation preview, writes nothing",
	"create_world_from_images":  "world from a picture: whole-map generation from stored pictures, no workbench field schema",
	"get_world_state_layout":    "read-only pagination of the saved world's canonical state naming index; no entity field schema",
	"name_world_map":            "second world-from-picture step: self-describing batch state and capital localisation payload",
	"repair_world_map":          "mends the map's crash findings as the map studio's Fix button; no entity fields",
	"undo_world_map_change":     "takes back the latest map record, as the map studio's undo; no entity fields",
	"restore_world_map":         "takes back every map record, as the map studio's restore; no entity fields",
	"update_gui_texts":          "exact-key static .gui localisation override; structured entities use their own tools, so no entity field reference applies",
	"batch_update_entities":     "cross-workbench batch tool; each operation follows its entity field documentation",
	"delete_entity":             "deletion takes an id, not workbench fields",
	"delete_focus":              "deletion takes an id, not workbench fields",
	"preview_delete_entity":     "deletion preview takes an id, not workbench fields",
	"preview_gui_overlay":       "validates the tool's self-describing NAME-PATH overlay schema without writing",
	"apply_gui_overlay_edits":   "uses the tool's self-describing NAME-PATH overlay schema and edit-hoi4-gui workflow, not the UI-panel entity schema",
	"validate_gui_overlay":      "validates a persisted NAME-PATH overlay and takes no workbench fields",
	"test":                      "harness-only tool",
}

func isReadOnlyToolName(name string) bool {
	for _, p := range []string{"check_", "find_", "list_", "get_", "preflight_", "project_", "quote_", "search_"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func fieldRefSentence(workbench string) string {
	return "Read " + fieldRefURI(workbench) + " for what each field expects before filling unfamiliar or raw-script fields."
}

func fieldRefURI(workbench string) string {
	return "`hw fields " + workbench + "`"
}

func fieldReferenceHintMiddleware() Middleware {
	return func(next MethodHandler) MethodHandler {
		return func(ctx context.Context, method string, req Request) (Result, error) {
			result, err := next(ctx, method, req)
			if err != nil || method != "tools/list" {
				return result, err
			}
			list, ok := result.(*ListToolsResult)
			if !ok || list == nil {
				return result, err
			}
			for _, tool := range list.Tools {
				if tool == nil {
					continue
				}
				workbench, mapped := toolFieldRef[tool.Name]
				if !mapped || strings.Contains(tool.Description, fieldRefURI(workbench)) {
					continue
				}
				desc := strings.TrimRight(tool.Description, " ")
				if desc != "" && !strings.HasSuffix(desc, ".") {
					desc += "."
				}
				if desc != "" {
					desc += " "
				}
				tool.Description = desc + fieldRefSentence(workbench)
			}
			return result, err
		}
	}
}

func fieldRefWorkbenches() []string {
	seen := map[string]bool{}
	out := []string{}
	for _, wb := range toolFieldRef {
		if !seen[wb] {
			seen[wb] = true
			out = append(out, wb)
		}
	}
	sort.Strings(out)
	return out
}
