package exportmerge

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

var entitySnapshotFields = map[string][]string{
	"character":  strings.Fields(`advisor_available advisor_cost advisor_idea_token advisor_modifier advisor_on_add advisor_on_remove advisor_removal_cost advisor_slot advisor_traits advisor_visible always_show_on_actions_tooltip attack_skill can_be_captured coordination_skill country_tag defense_skill desc_en desc_zh description description_locs expire gender has_advisor_role ideology is_portrait_modified is_replacement large_portrait logistics_skill display_name_locs localisation_key maneuvering_skill military_visible name name_en name_zh operative_available operative_bg operative_nationalities operative_skill_level operative_traits planning_skill raw_extras replaces_vanilla_id role_type scientist_data scientist_visible skill small_portrait tag traits`),
	"event":      strings.Fields(`after auto_fire_after_days auto_fire_on_startup country_tag description description_locs desc_en desc_zh display_name_locs event_id event_number event_type fire_only_once hidden immediate is_triggered_only major mean_time_to_happen on_action_attach options picture raw_extras scope_kind shared_country_tags timeout_days title title_en title_zh trigger`),
	"decision":   strings.Fields(`activation ai_will_do allowed available cancel_effect cancel_trigger complete_effect cost country_tag custom_cost_text custom_cost_trigger days_mission_timeout days_re_enable days_remove decision_id description_locs desc_en desc_zh display_name_locs effect_tooltip_locs fire_only_once icon is_good modifier name_en name_zh priority raw_extras remove_effect scope_kind selectable_mission shared_country_tags state_target target_array target_root_trigger target_trigger targeted_modifier targets timeout_effect visible war_with_on_add war_with_on_remove`),
	"focus_node": strings.Fields(`ai_will_do allow_branch_ast available bypass cancel completion_reward cost description_locs desc_en desc_zh display_name_locs dynamic_icon effect_tooltip_locs focus_id icon load_focus_tree_rules mutually_exclusive name name_en name_zh offset prerequisites raw_extras relative_position_id search_filters select_effect text_icon x y`),
	"idea":       strings.Fields(`allowed allowed_civil_war allowed_to_remove available cancel cancel_if_invalid category_flags country_tag description_locs desc_en desc_zh designer_traits display_name_locs do_effect equipment_bonus idea_id idea_type is_designer ledger level modifier name name_en name_zh on_add on_remove picture raw_extras removal_cost research_bonus rule scope_kind shared_country_tags targeted_modifier visible`),
}

const snapshotDefaultsJSON = `{
"character":{"advisor_available":"","advisor_cost":150,"advisor_idea_token":"","advisor_modifier":"","advisor_on_add":"","advisor_on_remove":"","advisor_removal_cost":-1,"advisor_slot":"","advisor_traits":[],"advisor_visible":"","attack_skill":1,"can_be_captured":true,"coordination_skill":1,"country_tag":"","defense_skill":1,"desc_en":"","desc_zh":"","description":"","description_locs":{},"display_name_locs":{},"expire":"","gender":"","has_advisor_role":false,"ideology":"","is_portrait_modified":false,"is_replacement":false,"large_portrait":"","localisation_key":"","logistics_skill":1,"maneuvering_skill":1,"military_visible":"","name_en":"","name_zh":"","operative_available":"","operative_bg":"","operative_nationalities":[],"operative_skill_level":1,"operative_traits":[],"planning_skill":1,"raw_extras":{},"replaces_vanilla_id":false,"scientist_data":{},"scientist_visible":"","skill":1,"small_portrait":"","traits":[]},
"decision":{"activation":"","ai_will_do":"","allowed":"","available":"","cancel_effect":"","cancel_trigger":"","complete_effect":"","cost":0,"country_tag":"","custom_cost_text":"","custom_cost_trigger":"","days_mission_timeout":0,"days_re_enable":0,"days_remove":0,"desc_en":"","desc_zh":"","description_locs":{},"display_name_locs":{},"effect_tooltip_locs":{},"fire_only_once":false,"icon":"","is_good":false,"modifier":"","name_en":"","name_zh":"","priority":0,"raw_extras":{},"remove_effect":"","scope_kind":"own","selectable_mission":false,"state_target":"","target_array":[],"target_root_trigger":"","target_trigger":"","targeted_modifier":{},"targets":[],"timeout_effect":"","visible":"","war_with_on_add":"","war_with_on_remove":""},
"event":{"after":"","auto_fire_after_days":0,"auto_fire_on_startup":false,"country_tag":"","desc_en":"","desc_zh":"","description":"","description_locs":{},"display_name_locs":{},"event_type":"country_event","fire_only_once":false,"hidden":false,"immediate":"","is_triggered_only":true,"major":false,"mean_time_to_happen":"","on_action_attach":"","options":[],"picture":"","raw_extras":{},"scope_kind":"own","timeout_days":0,"title":"","title_en":"","title_zh":"","trigger":""},
"focus_node":{"ai_will_do":"factor = 1","allow_branch_ast":{},"available":"","bypass":"","cancel":"","completion_reward":"","cost":10.0,"desc_en":"","desc_zh":"","description_locs":{},"display_name_locs":{},"dynamic_icon":[],"effect_tooltip_locs":{},"icon":"GFX_goal_unknown","load_focus_tree_rules":[],"mutually_exclusive":[],"name_en":"","name_zh":"","offset":[],"prerequisites":[],"raw_extras":{},"relative_position_id":"","search_filters":[],"select_effect":"","text_icon":"","x":0,"y":0},
"idea":{"allowed":"","allowed_civil_war":"","allowed_to_remove":"","available":"","cancel":"","cancel_if_invalid":true,"country_tag":"","desc_en":"","desc_zh":"","description_locs":{},"designer_traits":[],"display_name_locs":{},"do_effect":"","equipment_bonus":{},"is_designer":false,"ledger":"","level":0,"modifier":{},"name":"","name_en":"","name_zh":"","on_add":"","on_remove":"","picture":"","raw_extras":{},"removal_cost":-1,"research_bonus":{},"rule":{},"scope_kind":"own","targeted_modifier":[],"visible":""}}
`

var snapshotDefaults map[string]map[string]any

func init() {
	if err := decodeJSON([]byte(snapshotDefaultsJSON), &snapshotDefaults); err != nil {
		panic(err)
	}
}

type DiffOp map[string]any

type FieldConflict struct {
	Path     string `json:"path"`
	UserOp   DiffOp `json:"user_op"`
	TargetOp DiffOp `json:"target_op"`
	Kind     string `json:"kind"`
}

type ConflictReport struct {
	Status      string          `json:"status"`
	Ops         []DiffOp        `json:"ops"`
	Conflicts   []FieldConflict `json:"conflicts"`
	TargetValue map[string]any  `json:"target_value"`
	EntityID    string          `json:"entity_id"`
	EntityType  string          `json:"entity_type"`
}

type ResolutionRequest struct {
	Resolutions  map[string]string `json:"resolutions"`
	ManualValues map[string]any    `json:"manual_values"`
}

func CaptureCurrent(entityType string, row map[string]any) (map[string]any, error) {
	fields, ok := entitySnapshotFields[entityType]
	if !ok {
		return nil, fmt.Errorf("unknown entity_type %q", entityType)
	}
	out := make(map[string]any, len(fields))
	for _, field := range fields {
		out[field] = cloneJSON(row[field])
	}
	return out, nil
}

func SourceSnapshot(row map[string]any) (map[string]any, bool) {
	v, present := row["source_snapshot"]
	if !present || v == nil {
		return nil, false
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	return cloneJSON(m).(map[string]any), true
}

func NormalizeTarget(snapshot map[string]any, target map[string]any) map[string]any {
	if target == nil {
		return nil
	}
	out := map[string]any{}
	for k, v := range snapshot {
		out[k] = cloneJSON(v)
	}
	for k, v := range target {
		out[k] = cloneJSON(v)
	}
	return out
}

func MergeEntityDiff(entityType, entityID string, row, targetValue map[string]any) (ConflictReport, error) {
	current, err := CaptureCurrent(entityType, row)
	if err != nil {
		return ConflictReport{}, err
	}
	snapshot, hasSnapshot := SourceSnapshot(row)
	targetForDiff := targetValue
	if hasSnapshot {
		seeded := map[string]any{}
		for _, field := range entitySnapshotFields[entityType] {
			if _, exists := snapshot[field]; exists {
				continue
			}
			seeded[field] = cloneJSON(snapshotDefaults[entityType][field])
		}
		if len(seeded) > 0 {
			for k, v := range seeded {
				snapshot[k] = v
			}
			if targetValue != nil {
				targetForDiff = map[string]any{}
				for k, v := range seeded {
					targetForDiff[k] = cloneJSON(v)
				}
				for k, v := range targetValue {
					targetForDiff[k] = cloneJSON(v)
				}
			}
		}
	}
	report := ConflictReport{Ops: []DiffOp{}, Conflicts: []FieldConflict{}, TargetValue: targetValue, EntityID: entityID, EntityType: entityType}
	if !hasSnapshot && targetValue != nil {
		report.Status = "clean_merge"
		report.Ops = DiffEntity(map[string]any{}, current)
		return report, nil
	}
	baseline := snapshot
	if baseline == nil {
		baseline = map[string]any{}
	}
	userDiff := DiffEntity(baseline, current)
	if targetValue == nil {
		report.Status = "clean_merge"
		report.Ops = DiffEntity(map[string]any{}, current)
		return report, nil
	}
	if len(userDiff) == 0 {
		report.Status = "no_change"
		return report, nil
	}
	targetDiff := DiffEntity(baseline, targetForDiff)
	if len(targetDiff) == 0 {
		report.Status, report.Ops = "clean_merge", userDiff
		return report, nil
	}
	report.Ops = userDiff
	report.Conflicts = DetectFieldConflicts(userDiff, targetDiff)
	if len(report.Conflicts) == 0 {
		report.Status = "auto_mergeable"
	} else {
		report.Status = "conflict"
	}
	return report, nil
}

func DiffEntity(old, current map[string]any) []DiffOp {
	return diffDict("", old, current, 0)
}

func diffDict(prefix string, old, current map[string]any, depth int) []DiffOp {
	keys := make([]string, 0, len(old)+len(current))
	seen := map[string]bool{}
	for key := range old {
		seen[key] = true
		keys = append(keys, key)
	}
	for key := range current {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	ops := make([]DiffOp, 0)
	for _, key := range keys {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		ov, ook := old[key]
		nv, nok := current[key]
		switch {
		case !ook:
			ops = append(ops, DiffOp{"op": "add", "path": path, "value": cloneJSON(nv)})
		case !nok:
			ops = append(ops, DiffOp{"op": "remove", "path": path, "old": cloneJSON(ov)})
		case reflect.DeepEqual(ov, nv):
			continue
		default:
			om, okOld := ov.(map[string]any)
			nm, okNew := nv.(map[string]any)
			if okOld && okNew && depth+1 < 3 {
				ops = append(ops, diffDict(path, om, nm, depth+1)...)
			} else {
				ops = append(ops, DiffOp{"op": "replace", "path": path, "value": cloneJSON(nv), "old": cloneJSON(ov)})
			}
		}
	}
	return ops
}

func DetectFieldConflicts(userOps, targetOps []DiffOp) []FieldConflict {
	u, t := map[string]DiffOp{}, map[string]DiffOp{}
	for _, op := range userOps {
		u[fmt.Sprint(op["path"])] = op
	}
	for _, op := range targetOps {
		t[fmt.Sprint(op["path"])] = op
	}
	paths := []string{}
	for path := range u {
		if _, ok := t[path]; ok {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	out := []FieldConflict{}
	for _, path := range paths {
		uo, to := u[path], t[path]
		if uo["op"] == to["op"] && reflect.DeepEqual(uo["value"], to["value"]) {
			continue
		}
		kind := "value"
		if uo["op"] == "remove" || to["op"] == "remove" {
			kind = "remove_vs_change"
		}
		out = append(out, FieldConflict{Path: path, UserOp: uo, TargetOp: to, Kind: kind})
	}
	return out
}

func ApplyResolutions(report ConflictReport, resolutions map[string]string, manual map[string]any) ([]DiffOp, error) {
	byPath := map[string]DiffOp{}
	for _, op := range report.Ops {
		byPath[fmt.Sprint(op["path"])] = op
	}
	handled := map[string]bool{}
	final := []DiffOp{}
	for _, conflict := range report.Conflicts {
		path := conflict.Path
		handled[path] = true
		switch resolutions[path] {
		case "keep_user":
			if op := byPath[path]; op != nil {
				final = append(final, op)
			}
		case "manual":
			value, ok := manual[path]
			if !ok {
				return nil, fmt.Errorf("manual resolution chosen for %q but no value supplied", path)
			}
			old := conflict.TargetOp["old"]
			if old == nil {
				old = conflict.UserOp["old"]
			}
			final = append(final, DiffOp{"op": "replace", "path": path, "value": cloneJSON(value), "old": cloneJSON(old)})
		case "keep_target", "":

		default:
			return nil, fmt.Errorf("unsupported resolution %q", resolutions[path])
		}
	}
	for _, op := range report.Ops {
		if !handled[fmt.Sprint(op["path"])] {
			final = append(final, op)
		}
	}
	return final, nil
}

func ApplyDiff(target map[string]any, ops []DiffOp) (map[string]any, error) {
	result := cloneJSON(target).(map[string]any)
	for _, op := range ops {
		parts := strings.Split(fmt.Sprint(op["path"]), ".")
		if len(parts) == 0 || parts[0] == "" {
			return nil, fmt.Errorf("empty diff path")
		}
		switch op["op"] {
		case "add", "replace":
			setPath(result, parts, cloneJSON(op["value"]))
		case "remove":
			delPath(result, parts)
		default:
			return nil, fmt.Errorf("unknown diff op %q", op["op"])
		}
	}
	return result, nil
}

func setPath(m map[string]any, parts []string, value any) {
	cursor := m
	for _, p := range parts[:len(parts)-1] {
		next, ok := cursor[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			cursor[p] = next
		}
		cursor = next
	}
	cursor[parts[len(parts)-1]] = value
}

func delPath(m map[string]any, parts []string) {
	cursor := m
	for _, p := range parts[:len(parts)-1] {
		next, ok := cursor[p].(map[string]any)
		if !ok {
			return
		}
		cursor = next
	}
	delete(cursor, parts[len(parts)-1])
}

func cloneJSON(value any) any {
	switch value := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			out[k] = cloneJSON(v)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, v := range value {
			out[i] = cloneJSON(v)
		}
		return out
	default:
		return value
	}
}

func decodeJSON(raw []byte, dst any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	canonicalizeNumbers(reflect.ValueOf(dst))
	return nil
}

func canonicalizeNumbers(v reflect.Value) {
	if !v.IsValid() {
		return
	}
	if v.Kind() == reflect.Pointer {
		if !v.IsNil() {
			canonicalizeNumbers(v.Elem())
		}
		return
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		if n, ok := v.Interface().(json.Number); ok {
			if i, err := n.Int64(); err == nil {
				v.Set(reflect.ValueOf(any(i)))
				return
			}
			if f, err := n.Float64(); err == nil {
				v.Set(reflect.ValueOf(any(f)))
				return
			}
		}
		canonicalizeNumbers(v.Elem())
		return
	}
	switch v.Kind() {
	case reflect.Map:
		for _, key := range v.MapKeys() {
			item := v.MapIndex(key)
			holder := reflect.New(item.Type()).Elem()
			holder.Set(item)
			canonicalizeNumbers(holder)
			v.SetMapIndex(key, holder)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			canonicalizeNumbers(v.Index(i))
		}
	}
}
