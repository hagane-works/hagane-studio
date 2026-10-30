package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var projectTables = []string{
	"projects",
	"ai_strategy_plans", "ai_templates", "autonomy_states", "bookmarks",
	"characters", "countries", "custom_traits", "decision_categories",
	"decisions", "division_deployments", "division_templates", "doctrines",
	"dynamic_modifiers", "entity_obs_paths", "equipments", "event_namespaces",
	"events", "focus_file_policies", "focus_nodes", "focus_trees",
	"gui_overlays", "history_files", "ideas", "intelligence_agencies",
	"localisation_overrides", "military_industrial_organizations",
	"modifier_definitions", "oob_air_wing_groups", "oob_air_wings",
	"oob_fleets", "oob_ships", "oob_task_forces", "opinion_modifiers",
	"project_cosmetic_tag_names", "project_cosmetic_tags",
	"project_file_baseline_entries", "project_file_baselines",
	"project_loc_entries", "project_obs_assets", "project_obs_icons",
	"project_overlay_state", "project_raw_files", "scripted_localisations",
	"scripted_snippets", "special_project_specializations",
	"special_projects", "states", "sub_units", "super_events",
	"technologies", "ui_panels", "wargoal_types",
}

var runtimeAdditions = map[string][]Column{
	"projects": {{Name: "owned_dlcs", Type: "json"}},
	"world_map_additions": {
		{Name: "id", Type: "bigint", NotNull: true},
		{Name: "project_id", Type: "integer", NotNull: true},
		{Name: "placement", Type: "jsonb", NotNull: true},
		{Name: "seed", Type: "bigint", NotNull: true},
		{Name: "options", Type: "jsonb", NotNull: true, Default: map[string]any{}},
		{Name: "name", Type: "text", NotNull: true, Default: ""},
		{Name: "owner", Type: "text", NotNull: true, Default: ""},
		{Name: "provinces", Type: "jsonb", NotNull: true, Default: []any{}},
		{Name: "state_ids", Type: "jsonb", NotNull: true, Default: []any{}},
		{Name: "region_id", Type: "integer", NotNull: true, Default: 0},
		{Name: "raw_paths", Type: "jsonb", NotNull: true, Default: []any{}},
		{Name: "text_keys", Type: "jsonb", NotNull: true, Default: []any{}},
		{Name: "summary", Type: "jsonb", NotNull: true, Default: map[string]any{}},
		{Name: "created_at", Type: "timestamp with time zone", NotNull: true, Default: "now"},
		{Name: "kind", Type: "text", NotNull: true, Default: "land"},
		{Name: "state_before", Type: "jsonb", NotNull: true, Default: map[string]any{}},
		{Name: "previous", Type: "jsonb", NotNull: true, Default: map[string]any{}},
		{Name: "states_replaced", Type: "json", NotNull: true, Default: []any{}},
		{Name: "histories_before", Type: "jsonb", NotNull: true, Default: map[string]any{}},
	},
}

type Column struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	NotNull bool   `json:"not_null,omitempty"`

	Default any `json:"default,omitempty"`
}

var (
	createRE = regexp.MustCompile(`^CREATE TABLE (?:IF NOT EXISTS )?public\.([a-z0-9_]+) \($`)

	columnRE = regexp.MustCompile(`^\s+"?([a-z0-9_]+)"? (.+?),?$`)
	typeRE   = regexp.MustCompile(`^(character varying|timestamp without time zone|timestamp with time zone|double precision|[a-z]+)(\([0-9, ]+\))?(\[\])?`)
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: schemagen <schema.sql>")
		os.Exit(2)
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()

	wanted := map[string]bool{}
	for _, name := range projectTables {
		wanted[name] = true
	}
	tables := map[string][]Column{}
	var current string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if m := createRE.FindStringSubmatch(line); m != nil {
			current = ""
			if wanted[m[1]] {
				current = m[1]
			}
			continue
		}
		if current == "" {
			continue
		}
		if strings.HasPrefix(line, ")") {
			current = ""
			continue
		}
		m := columnRE.FindStringSubmatch(line)
		if m == nil || strings.HasPrefix(strings.TrimSpace(line), "CONSTRAINT") {
			continue
		}
		column, err := parseColumn(m[1], m[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s.%s: %v\n", current, m[1], err)
			os.Exit(1)
		}
		tables[current] = append(tables[current], column)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for table, columns := range runtimeAdditions {
		tables[table] = append(tables[table], columns...)
	}
	var missing []string
	for _, name := range projectTables {
		if _, ok := tables[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		fmt.Fprintf(os.Stderr, "tables not found in schema: %s\n", strings.Join(missing, ", "))
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(tables); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseColumn(name, rest string) (Column, error) {
	t := typeRE.FindStringSubmatch(rest)
	if t == nil {
		return Column{}, fmt.Errorf("unknown type in %q", rest)
	}
	column := Column{Name: name, Type: t[1] + t[3], NotNull: strings.Contains(rest, "NOT NULL")}
	if i := strings.Index(rest, " DEFAULT "); i >= 0 {
		expr := strings.TrimSpace(strings.TrimSuffix(rest[i+len(" DEFAULT "):], " NOT NULL"))
		if def, ok := constantDefault(expr); ok {
			column.Default = def
		}
	}
	return column, nil
}

func constantDefault(expr string) (any, bool) {
	switch {
	case expr == "true" || expr == "false":
		return expr == "true", true
	case expr == "CURRENT_TIMESTAMP" || expr == "now()":
		return "now", true
	case strings.HasPrefix(expr, "nextval("):
		return nil, false
	}

	if i := strings.LastIndex(expr, "::"); i > 0 && !strings.Contains(expr[i:], "'") {
		expr = expr[:i]
	}
	if strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") {
		text := strings.ReplaceAll(expr[1:len(expr)-1], "''", "'")
		var parsed any
		if (strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[")) && json.Unmarshal([]byte(text), &parsed) == nil {
			return parsed, true
		}
		return text, true
	}
	if n, err := strconv.ParseInt(strings.Trim(expr, "()"), 10, 64); err == nil {
		return n, true
	}
	if f, err := strconv.ParseFloat(strings.Trim(expr, "()"), 64); err == nil {
		return f, true
	}
	if expr == "NULL" {
		return nil, false
	}
	return nil, false
}
