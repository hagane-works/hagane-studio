package focus

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
)

var (
	flagSetRe = regexp.MustCompile(
		`(?i)\b(?:set_country_flag|set_global_flag)\s*=\s*(?:\{[^}]*?\bflag\s*=\s*)?([a-zA-Z_][a-zA-Z0-9_]*)(.?)`)
	flagCheckRe = regexp.MustCompile(
		`(?i)\b(?:has_country_flag|has_global_flag)\s*=\s*(?:\{[^}]*?\bflag\s*=\s*)?([a-zA-Z_][a-zA-Z0-9_]*)(.?)`)
)

func dynamicFlagSuffix(s string) bool { return s == "@" || s == "$" }

func walkJSON(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, walkJSON(value[key]))
		}
		return strings.Join(parts, " ")
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			parts = append(parts, walkJSON(item))
		}
		return strings.Join(parts, " ")
	}
	return ""
}

func walkJSONText(v any) string {
	switch v.(type) {
	case map[string]any, []any, string:
		return walkJSON(v)
	}
	return ""
}

type deadFlagState struct {
	setMap   map[string][]string
	checkSet map[string]bool
}

func (st *deadFlagState) harvest(text, label string) {
	if text == "" {
		return
	}
	for _, m := range flagSetRe.FindAllStringSubmatch(text, -1) {
		if !dynamicFlagSuffix(m[2]) {
			st.setMap[m[1]] = append(st.setMap[m[1]], label)
		}
	}
	for _, m := range flagCheckRe.FindAllStringSubmatch(text, -1) {
		if !dynamicFlagSuffix(m[2]) {
			st.checkSet[m[1]] = true
		}
	}
}

func (st *deadFlagState) harvestConditionAST(v any) {
	switch n := v.(type) {
	case map[string]any:
		if op, _ := n["op"].(string); op == "has_country_flag" || op == "has_global_flag" {
			if val, _ := n["value"].(string); strings.TrimSpace(val) != "" {
				m := flagCheckRe.FindStringSubmatch("has_country_flag = " + val)
				if m != nil && !dynamicFlagSuffix(m[2]) {
					st.checkSet[m[1]] = true
				} else if fields := strings.Fields(strings.TrimSpace(val)); len(fields) > 0 {
					if tok := fields[0]; !strings.Contains(tok, "@") && !strings.Contains(tok, "$") {
						st.checkSet[tok] = true
					}
				}
			}
		}
		for _, sub := range n {
			st.harvestConditionAST(sub)
		}
	case []any:
		for _, item := range n {
			st.harvestConditionAST(item)
		}
	}
}

func scanDeadFlags(p *project.Project) (map[string]any, error) {
	st := &deadFlagState{setMap: map[string][]string{}, checkSet: map[string]bool{}}
	rows := func(table string) ([]project.Row, error) {
		r, err := p.Rows(table)
		return sortByID(r), err
	}
	text := func(row project.Row, key string) string { return colStr(row, key, "") }

	trees, err := p.Rows("focus_trees")
	if err != nil {
		return nil, err
	}
	known := map[int64]bool{}
	for _, tree := range trees {
		known[project.Int64(tree["id"])] = true
	}
	nodes, err := rows("focus_nodes")
	if err != nil {
		return nil, err
	}
	for _, n := range nodes {
		if !known[project.Int64(n["focus_tree_id"])] {
			continue
		}
		label := text(n, "focus_id")
		for _, name := range []string{"completion_reward", "select_effect", "bypass", "cancel", "available", "ai_will_do"} {
			st.harvest(text(n, name), "focus "+label+"."+name)
		}
		st.harvest(walkJSONText(n["raw_extras"]), "focus "+label+".raw_extras")
		st.harvestConditionAST(n["allow_branch_ast"])
	}

	events, err := rows("events")
	if err != nil {
		return nil, err
	}
	for _, e := range events {
		label := text(e, "event_id")
		for _, name := range []string{"immediate", "trigger", "mean_time_to_happen", "after"} {
			st.harvest(text(e, name), "event "+label+"."+name)
		}
		st.harvest(walkJSONText(e["raw_extras"]), "event "+label+".raw_extras")
		options, _ := e["options"].([]any)
		for i, option := range options {
			if _, isDict := option.(map[string]any); isDict {
				st.harvest(walkJSON(option), "event "+label+".option["+strconv.Itoa(i)+"]")
			}
		}
	}

	scanFields := func(table, kind, ident string, fields []string, rawExtras bool, extraJSON []string) error {
		list, err := rows(table)
		if err != nil {
			return err
		}
		for _, row := range list {
			label := text(row, ident)
			for _, field := range fields {
				st.harvest(text(row, field), kind+" "+label+"."+field)
			}
			for _, name := range extraJSON {
				st.harvest(walkJSONText(row[name]), kind+" "+label+"."+name)
			}
			if rawExtras {
				st.harvest(walkJSONText(row["raw_extras"]), kind+" "+label+".raw_extras")
			}
		}
		return nil
	}
	if err := scanFields("decisions", "decision", "decision_id", []string{"allowed", "available", "visible",
		"complete_effect", "remove_effect", "cancel_trigger", "cancel_effect", "modifier", "activation",
		"timeout_effect", "ai_will_do", "target_root_trigger", "target_trigger", "custom_cost_trigger"}, true, nil); err != nil {
		return nil, err
	}

	histories, err := rows("history_files")
	if err != nil {
		return nil, err
	}
	for _, h := range histories {
		label := text(h, "country_tag")
		startup, _ := h["startup_flags"].([]any)
		for _, v := range startup {
			if flag, ok := v.(string); ok && strings.TrimSpace(flag) != "" {
				st.setMap[strings.TrimSpace(flag)] = append(st.setMap[strings.TrimSpace(flag)], "history_file "+label+".startup_flags")
			}
		}
		st.harvest(text(h, "raw_script"), "history_file "+label+".raw_script")
	}

	superEvents, err := rows("super_events")
	if err != nil {
		return nil, err
	}
	for _, se := range superEvents {
		tag := strings.TrimSpace(text(se, "country_tag"))
		if tag == "" {
			tag = "UNK"
		}
		seID := strings.ToLower(tag) + "_se." + strconv.FormatInt(project.Int64(se["number"]), 10)
		options, _ := se["options"].([]any)
		for i, option := range options {
			if _, isDict := option.(map[string]any); isDict {
				st.harvest(walkJSON(option), "super_event "+seID+".option["+strconv.Itoa(i)+"]")
			}
		}
	}

	if err := scanFields("ideas", "idea", "idea_id", []string{"available", "visible", "allowed",
		"allowed_civil_war", "allowed_to_remove", "cancel", "do_effect", "on_add", "on_remove"}, true, nil); err != nil {
		return nil, err
	}

	characters, err := rows("characters")
	if err != nil {
		return nil, err
	}
	for _, c := range characters {
		label := text(c, "tag")
		if label == "" {
			label = text(c, "name")
		}
		for _, field := range []string{"advisor_available", "advisor_visible", "advisor_modifier",
			"advisor_on_add", "advisor_on_remove", "military_visible", "scientist_visible", "operative_available"} {
			st.harvest(text(c, field), "character "+label+"."+field)
		}
	}

	if err := scanFields("dynamic_modifiers", "dynamic_modifier", "dm_id", []string{"enable", "remove_trigger"}, true, nil); err != nil {
		return nil, err
	}
	if err := scanFields("military_industrial_organizations", "mio", "mio_id", []string{"allowed", "available", "visible"}, true, []string{"traits"}); err != nil {
		return nil, err
	}

	setTotal := 0
	flags := make([]string, 0, len(st.setMap))
	for flag, locations := range st.setMap {
		setTotal += len(locations)
		flags = append(flags, flag)
	}
	sort.Strings(flags)
	deadFlags := []map[string]any{}
	for _, flag := range flags {
		if st.checkSet[flag] {
			continue
		}
		deadFlags = append(deadFlags, map[string]any{"flag_id": flag, "set_locations": st.setMap[flag]})
	}
	return map[string]any{"set_total": setTotal, "check_total": len(st.checkSet), "dead_flags": deadFlags}, nil
}

func (h *handlers) deadFlags(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	report, err := scanDeadFlags(p)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, report)
	return nil
}
