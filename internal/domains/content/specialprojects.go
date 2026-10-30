package content

import (
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

const (
	specializationsTable  = "special_project_specializations"
	specialProjectsTable  = "special_projects"
	specialProjectsGone   = "special project store unavailable"
	specializationMissing = "Specialization not found"
	specialProjectMissing = "Special project not found"
)

func rowJSON(row project.Row, columns []string, defaults map[string]any) map[string]any {
	out := make(map[string]any, len(columns))
	for _, column := range columns {
		value := row[column]
		if value == nil {
			value = defaults[column]
		}
		out[column] = value
	}
	return out
}

var specializationColumns = []string{
	"id", "project_id", "country_tag", "specialization_id", "name_en", "name_zh", "desc_en", "desc_zh",
	"display_name_locs", "description_locs", "color", "blueprint_image", "program_background",
	"facility_model", "sub_specializations", "raw_extras", "source", "created_at", "updated_at",
}

var specializationDefaults = map[string]any{
	"country_tag": "", "name_en": "", "name_zh": "", "desc_en": "", "desc_zh": "",
	"blueprint_image": "", "program_background": "", "facility_model": "", "source": "created",
}

var specialProjectColumns = []string{
	"id", "project_id", "country_tag", "project_token", "specialization", "name_en", "name_zh",
	"desc_en", "desc_zh", "display_name_locs", "description_locs", "icon", "blueprint_image",
	"complexity", "prototype_time", "project_tags", "special_project_parent", "allowed", "visible",
	"available", "resource_cost", "project_output", "generic_prototype_rewards", "x", "y",
	"raw_extras", "source", "created_at", "updated_at",
}

var specialProjectDefaults = map[string]any{
	"country_tag": "", "specialization": "", "name_en": "", "name_zh": "", "desc_en": "", "desc_zh": "",
	"icon": "", "blueprint_image": "", "complexity": "", "prototype_time": "", "allowed": "",
	"visible": "", "available": "", "resource_cost": "", "project_output": "",
	"x": int64(0), "y": int64(0), "source": "created",
}

var specializationJSONColumns = map[string]bool{
	"display_name_locs": true, "description_locs": true, "color": true,
	"sub_specializations": true, "raw_extras": true,
}

var specializationUpdateColumns = []string{
	"specialization_id", "country_tag", "name_en", "name_zh", "desc_en", "desc_zh",
	"display_name_locs", "description_locs", "color", "blueprint_image",
	"program_background", "facility_model", "sub_specializations", "raw_extras",
}

var specialProjectJSONColumns = map[string]bool{
	"display_name_locs": true, "description_locs": true, "project_tags": true,
	"special_project_parent": true, "generic_prototype_rewards": true, "raw_extras": true,
}

var specialProjectUpdateColumns = []string{
	"project_token", "country_tag", "specialization", "name_en", "name_zh",
	"desc_en", "desc_zh", "display_name_locs", "description_locs", "icon",
	"blueprint_image", "complexity", "prototype_time", "project_tags",
	"special_project_parent", "allowed", "visible", "available", "resource_cost",
	"project_output", "generic_prototype_rewards", "x", "y", "raw_extras",
}

var specialIntColumns = map[string]bool{"x": true, "y": true}

var numericPrefixToken = regexp.MustCompile(`^-?[0-9]`)

func validateSpecialProjectScalar(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	entry, err := importsvc.ParseHOI4ScalarToken(value, false)
	if err != nil || entry.Quoted {
		return fmt.Errorf("%s must be one unquoted positive integer or script constant", field)
	}
	if entry.Scalar.Kind == paradox.ScalarFloat || entry.Scalar.Kind == paradox.ScalarInt && entry.Scalar.Int.Sign() <= 0 ||
		entry.Scalar.Kind == paradox.ScalarStr && numericPrefixToken.MatchString(entry.Scalar.Str) {
		return fmt.Errorf("%s must be one unquoted positive integer or script constant", field)
	}
	return nil
}

func emittedSpecialProjectBlock(field, value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		return field + " = " + value
	}
	return field + " = {\n" + value + "\n}"
}

func validateSpecialProjectBlock(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	script := emittedSpecialProjectBlock(field, value)
	root, recoveries, err := paradox.Parse(script)
	if err != nil || len(recoveries) != 0 || root == nil || len(root.Entries) != 1 || root.Entries[0].Block == nil {
		return fmt.Errorf("%s must be a valid HOI4 script block", field)
	}
	for _, token := range paradox.Tokenize([]rune(script)) {
		if token.Type == "UNKNOWN" || token.Type == "STRING" && !token.Quoted && numericPrefixToken.MatchString(token.Value) {
			return fmt.Errorf("%s contains an invalid token %q", field, token.Value)
		}
	}
	if field == "resource_cost" {
		for _, entry := range root.Entries[0].Block.Entries {
			if entry.Key != nil && *entry.Key == "resources" && entry.Block != nil {
				return nil
			}
		}
		return fmt.Errorf("resource_cost must contain a resources = { ... } block")
	}
	return nil
}

func validateSpecialProjectValues(in map[string]any, requireSpecialization bool) error {
	if value, present := in["specialization"]; requireSpecialization || present {
		if strings.TrimSpace(anyToString(value)) == "" {
			return statusErr(http.StatusBadRequest, "specialization cannot be empty")
		}
	}
	for _, field := range []string{"complexity", "prototype_time"} {
		if value, present := in[field]; present {
			if err := validateSpecialProjectScalar(field, anyToString(value)); err != nil {
				return statusErr(http.StatusUnprocessableEntity, err.Error())
			}
		}
	}
	for _, field := range []string{"resource_cost", "project_output"} {
		if value, present := in[field]; present {
			if err := validateSpecialProjectBlock(field, anyToString(value)); err != nil {
				return statusErr(http.StatusUnprocessableEntity, err.Error())
			}
		}
	}
	return nil
}

func coerceInt(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		var parsed int64
		if _, err := fmt.Sscanf(strings.TrimSpace(t), "%d", &parsed); err == nil {
			return parsed
		}
	}
	return 0
}

func columnValue(column string, value any, jsonColumns map[string]bool) (any, error) {
	if jsonColumns[column] {
		return jsonValue(value), nil
	}
	if value == nil {
		return nil, nil
	}
	if specialIntColumns[column] {
		if f, ok := value.(float64); ok && f == math.Trunc(f) {
			return int64(f), nil
		}
		return nil, errConstraint
	}
	if s, ok := value.(string); ok {
		return s, nil
	}
	return nil, errConstraint
}

func patchSets(in map[string]any, columns []string, jsonColumns map[string]bool) (project.Row, error) {
	sets := project.Row{}
	for _, column := range columns {
		value, present := in[column]
		if !present {
			continue
		}
		converted, err := columnValue(column, value, jsonColumns)
		if err != nil {
			return nil, err
		}
		sets[column] = converted
	}
	return sets, nil
}

func jsonOr(v any, empty any) any {
	if v == nil {
		return empty
	}
	return jsonValue(v)
}

func (h *handlers) specialProjects(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	if len(rest) == 0 {
		detail(w, http.StatusNotFound, "Not Found")
		return
	}
	if rest[0] == "validate" && len(rest) == 1 && r.Method == http.MethodGet {
		result, err := validateSpecialProjectDAG(p, queryPtr(r, "country_tag"))
		if err != nil {
			fail(w, err, specialProjectsGone)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	switch rest[0] {
	case "specializations":
		crud{
			missing: specializationMissing, unavailable: specialProjectsGone,
			list: func(p *project.Project, r *http.Request) (any, int, error) {
				items, err := listSpecial(p, specializationsTable, "specialization_id", nil, queryPtr(r, "country_tag"))
				return items, len(items), err
			},
			create: createSpecialization,
			update: func(p *project.Project, r *http.Request, id int64) (any, error) {
				return updateSpecial(p, r, id, specializationsTable, specializationMissing, "specialization_id",
					specializationUpdateColumns, specializationJSONColumns, specializationColumns, specializationDefaults)
			},
			remove: removeRow(specializationsTable, specializationMissing),
		}.serve(p, w, r, rest[1:])
	case "projects":
		crud{
			missing: specialProjectMissing, unavailable: specialProjectsGone,
			list: func(p *project.Project, r *http.Request) (any, int, error) {
				items, err := listSpecial(p, specialProjectsTable, "project_token", queryPtr(r, "specialization"), queryPtr(r, "country_tag"))
				return items, len(items), err
			},
			create: createSpecialProject,
			update: func(p *project.Project, r *http.Request, id int64) (any, error) {
				return updateSpecial(p, r, id, specialProjectsTable, specialProjectMissing, "project_token",
					specialProjectUpdateColumns, specialProjectJSONColumns, specialProjectColumns, specialProjectDefaults)
			},
			remove: removeRow(specialProjectsTable, specialProjectMissing),
		}.serve(p, w, r, rest[1:])
	default:
		detail(w, http.StatusNotFound, "Not Found")
	}
}

func specialShape(table string, row project.Row) map[string]any {
	if table == specializationsTable {
		return rowJSON(row, specializationColumns, specializationDefaults)
	}
	return rowJSON(row, specialProjectColumns, specialProjectDefaults)
}

func listSpecial(p *project.Project, table, orderBy string, specialization, countryTag *string) ([]map[string]any, error) {
	rows, err := p.Rows(table)
	if err != nil {
		return nil, err
	}
	sortRows(rows, byText(orderBy))
	out := []map[string]any{}
	for _, row := range rows {
		if specialization != nil && *specialization != "" && text(row, "specialization") != *specialization {
			continue
		}
		if countryTag != nil && *countryTag != "" && text(row, "country_tag") != strings.ToUpper(*countryTag) {
			continue
		}
		out = append(out, specialShape(table, row))
	}
	return out, nil
}

func createSpecialization(p *project.Project, r *http.Request) (any, error) {
	var in map[string]any
	if err := decodeBody(r, &in); err != nil {
		return nil, errInvalidPayload
	}
	specID := strings.TrimSpace(anyToString(in["specialization_id"]))
	if specID == "" {
		return nil, statusErr(http.StatusBadRequest, "specialization_id cannot be empty")
	}
	row, err := writeRow(p, specializationsTable, func(tx *project.Tx) (project.Row, error) {
		if taken(tx.Rows(specializationsTable), 0, func(row project.Row) bool { return text(row, "specialization_id") == specID }) {
			return nil, statusErr(http.StatusConflict, fmt.Sprintf("Specialization '%s' already exists", specID))
		}
		countryTag := anyToString(in["country_tag"])
		if countryTag == "" {
			countryTag = text(tx.Record(), "country_tag")
		}
		nameEn := anyToString(in["name_en"])
		if nameEn == "" {
			nameEn = specID
		}
		return insert(tx, specializationsTable, project.Row{
			"country_tag": strings.ToUpper(countryTag), "specialization_id": specID,
			"name_en": nameEn, "name_zh": anyToString(in["name_zh"]),
			"desc_en": anyToString(in["desc_en"]), "desc_zh": anyToString(in["desc_zh"]),
			"display_name_locs":   jsonOr(in["display_name_locs"], map[string]any{}),
			"description_locs":    jsonOr(in["description_locs"], map[string]any{}),
			"color":               jsonOr(in["color"], []any{}),
			"blueprint_image":     anyToString(in["blueprint_image"]),
			"program_background":  anyToString(in["program_background"]),
			"facility_model":      anyToString(in["facility_model"]),
			"sub_specializations": jsonOr(in["sub_specializations"], []any{}),
			"raw_extras":          map[string]any{}, "source": "created",
		})
	})
	if err != nil {
		return nil, err
	}
	return specialShape(specializationsTable, row), nil
}

func createSpecialProject(p *project.Project, r *http.Request) (any, error) {
	var in map[string]any
	if err := decodeBody(r, &in); err != nil {
		return nil, errInvalidPayload
	}
	if err := validateSpecialProjectValues(in, true); err != nil {
		return nil, err
	}
	token := strings.TrimSpace(anyToString(in["project_token"]))
	if token == "" {
		return nil, statusErr(http.StatusBadRequest, "project_token cannot be empty")
	}
	row, err := writeRow(p, specialProjectsTable, func(tx *project.Tx) (project.Row, error) {
		if taken(tx.Rows(specialProjectsTable), 0, func(row project.Row) bool { return text(row, "project_token") == token }) {
			return nil, statusErr(http.StatusConflict, fmt.Sprintf("Special project '%s' already exists", token))
		}
		countryTag := anyToString(in["country_tag"])
		if countryTag == "" {
			countryTag = text(tx.Record(), "country_tag")
		}
		nameEn := anyToString(in["name_en"])
		if nameEn == "" {
			nameEn = token
		}
		values := project.Row{
			"country_tag": strings.ToUpper(countryTag), "project_token": token, "name_en": nameEn,
			"display_name_locs":         jsonOr(in["display_name_locs"], map[string]any{}),
			"description_locs":          jsonOr(in["description_locs"], map[string]any{}),
			"project_tags":              jsonOr(in["project_tags"], []any{}),
			"special_project_parent":    jsonOr(in["special_project_parent"], []any{}),
			"generic_prototype_rewards": jsonOr(in["generic_prototype_rewards"], []any{}),
			"x":                         coerceInt(in["x"]), "y": coerceInt(in["y"]),
			"raw_extras": map[string]any{}, "source": "created",
		}
		for _, column := range []string{"specialization", "name_zh", "desc_en", "desc_zh", "icon",
			"blueprint_image", "complexity", "prototype_time", "allowed", "visible", "available",
			"resource_cost", "project_output"} {
			values[column] = anyToString(in[column])
		}
		return insert(tx, specialProjectsTable, values)
	})
	if err != nil {
		return nil, err
	}
	return specialShape(specialProjectsTable, row), nil
}

func updateSpecial(p *project.Project, r *http.Request, id int64, table, missing, idColumn string,
	updateColumns []string, jsonColumns map[string]bool, columns []string, defaults map[string]any,
) (any, error) {
	var in map[string]any
	if err := decodeBody(r, &in); err != nil {
		return nil, errInvalidPayload
	}
	cur, err := getRow(p, table, id, missing)
	if err != nil {
		return nil, err
	}
	if in == nil {
		in = map[string]any{}
	}
	if table == specialProjectsTable {
		if err := validateSpecialProjectValues(in, false); err != nil {
			return nil, err
		}
	}
	if v, present := in[idColumn]; present {
		cleaned := strings.TrimSpace(anyToString(v))
		if cleaned == "" {
			return nil, statusErr(http.StatusBadRequest, idColumn+" cannot be empty")
		}
		in[idColumn] = cleaned
	}
	if v, present := in["country_tag"]; present {
		if s, ok := v.(string); ok {
			in["country_tag"] = strings.ToUpper(strings.TrimSpace(s))
		}
	}
	sets, err := patchSets(in, updateColumns, jsonColumns)
	if err != nil {
		return nil, err
	}
	if text(cur, "source") == "imported" {
		sets["source"] = "created"
	}
	row, err := updateRow(p, table, id, sets, uniqueOn(table, idColumn, id, sets))
	if err != nil {
		return nil, err
	}
	return rowJSON(row, columns, defaults), nil
}

func validateSpecialProjectDAG(p *project.Project, countryTag *string) (map[string]any, error) {
	rows, err := p.Rows(specialProjectsTable)
	if err != nil {
		return nil, err
	}
	parents := map[string][]string{}
	projectRows := map[string]project.Row{}
	total := 0
	for _, row := range rows {
		if countryTag != nil && *countryTag != "" && text(row, "country_tag") != strings.ToUpper(*countryTag) {
			continue
		}
		var list []string
		decode(row["special_project_parent"], &list)
		token := text(row, "project_token")
		parents[token] = list
		projectRows[token] = row
		total++
	}
	tokens := map[string]bool{}
	sortedTokens := make([]string, 0, len(parents))
	for token := range parents {
		tokens[token] = true
		sortedTokens = append(sortedTokens, token)
	}
	sort.Strings(sortedTokens)

	actionable := []map[string]any{}
	info := []map[string]any{}
	for _, token := range sortedTokens {
		row := projectRows[token]
		if strings.TrimSpace(text(row, "specialization")) == "" {
			actionable = append(actionable, map[string]any{
				"kind": "missing_specialization", "project_token": token,
				"detail": "specialization is required by the game",
			})
		}
		for _, field := range []string{"complexity", "prototype_time"} {
			if err := validateSpecialProjectScalar(field, text(row, field)); err != nil {
				actionable = append(actionable, map[string]any{
					"kind": "invalid_script_field", "project_token": token, "field": field, "detail": err.Error(),
				})
			}
		}
		for _, field := range []string{"resource_cost", "project_output"} {
			if err := validateSpecialProjectBlock(field, text(row, field)); err != nil {
				actionable = append(actionable, map[string]any{
					"kind": "invalid_script_field", "project_token": token, "field": field, "detail": err.Error(),
				})
			}
		}
		for _, parent := range parents[token] {
			if !tokens[parent] {
				actionable = append(actionable, map[string]any{
					"kind": "missing_parent", "project_token": token,
					"parent": parent, "detail": fmt.Sprintf("parent '%s' does not exist", parent),
				})
			}
		}
	}

	const white, gray, black = 0, 1, 2
	color := map[string]int{}
	var dfs func(node string, stack []string) ([]string, bool)
	dfs = func(node string, stack []string) ([]string, bool) {
		color[node] = gray
		stack = append(stack, node)
		for _, parent := range parents[node] {
			if !tokens[parent] {
				continue
			}
			if color[parent] == gray {
				start := 0
				for i, v := range stack {
					if v == parent {
						start = i
						break
					}
				}
				return append(append([]string{}, stack[start:]...), parent), true
			}
			if color[parent] == white {
				if cycle, found := dfs(parent, stack); found {
					return cycle, true
				}
			}
		}
		color[node] = black
		return nil, false
	}
	for _, token := range sortedTokens {
		if color[token] != white {
			continue
		}
		if cycle, found := dfs(token, nil); found {
			actionable = append(actionable, map[string]any{
				"kind":   "cycle",
				"detail": "dependency cycle: " + strings.Join(cycle, " → "),
				"nodes":  cycle,
			})
			break
		}
	}

	hasChild := map[string]bool{}
	for _, list := range parents {
		for _, parent := range list {
			if tokens[parent] {
				hasChild[parent] = true
			}
		}
	}
	for _, token := range sortedTokens {
		if len(parents[token]) == 0 && !hasChild[token] {
			info = append(info, map[string]any{
				"kind": "orphan", "project_token": token,
				"detail": "isolated project (no parents, no children)",
			})
		}
	}
	return map[string]any{"actionable": actionable, "info": info, "total": total}, nil
}
