package military

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	militaryparser "haganeworks/internal/hoi4/importsvc/military"
	"haganeworks/internal/project"
)

const militaryImportBodyLimit = 16 << 20

type militaryImportWire struct {
	Text       *string `json:"text"`
	SourceFile string  `json:"source_file"`
	OobFile    *string `json:"oob_file"`
	Policy     string  `json:"policy"`
}

func decodeMilitaryImport(w http.ResponseWriter, r *http.Request) (militaryImportWire, bool) {
	var input militaryImportWire
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, militaryImportBodyLimit))
	if err := decoder.Decode(&input); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			detail(w, http.StatusRequestEntityTooLarge, "Import payload is too large")
		} else {
			detail(w, http.StatusUnprocessableEntity, "Invalid military import payload")
		}
		return input, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		detail(w, http.StatusUnprocessableEntity, "Invalid military import payload")
		return input, false
	}
	if input.Text == nil {
		detail(w, http.StatusUnprocessableEntity, "text is required")
		return input, false
	}
	return input, true
}

func writeImportResult(w http.ResponseWriter, result any, err error, op string) {
	if err == nil {
		writeJSON(w, http.StatusOK, result)
		return
	}
	var status *apiutil.StatusError
	if errors.As(err, &status) {
		apiutil.Fail(w, err)
		return
	}
	log.Printf("%s: %v", op, err)
	detail(w, http.StatusInternalServerError, "import commit failed: "+err.Error())
}

func (h *handlers) equipmentImport(w http.ResponseWriter, r *http.Request, p *project.Project) {
	if r.Method != http.MethodPost {
		detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	input, ok := decodeMilitaryImport(w, r)
	if !ok {
		return
	}
	result, err := importEquipmentText(p, *input.Text, input.SourceFile, input.Policy)
	writeImportResult(w, result, err, "equipment import")
}

func (h *handlers) subUnitImport(w http.ResponseWriter, r *http.Request, p *project.Project) {
	if r.Method != http.MethodPost {
		detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	input, ok := decodeMilitaryImport(w, r)
	if !ok {
		return
	}
	result, err := importSubUnitsText(p, *input.Text, input.SourceFile, input.Policy)
	writeImportResult(w, result, err, "sub-unit import")
}

func (h *handlers) divisionTemplateImport(w http.ResponseWriter, r *http.Request, p *project.Project) {
	if r.Method != http.MethodPost {
		detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	input, ok := decodeMilitaryImport(w, r)
	if !ok {
		return
	}
	if input.OobFile == nil {
		detail(w, http.StatusUnprocessableEntity, "oob_file is required")
		return
	}
	result, err := importDivisionTemplatesText(p.Write, *input.Text, *input.OobFile, input.Policy)
	writeImportResult(w, result, err, "division template import")
}

type EquipmentImportResult struct {
	Created      int      `json:"created"`
	Updated      int      `json:"updated"`
	Skipped      int      `json:"skipped"`
	Warnings     []string `json:"warnings"`
	EquipmentIDs []string `json:"equipment_ids"`
}

type SubUnitImportResult struct {
	Created    int      `json:"created"`
	Updated    int      `json:"updated"`
	Skipped    int      `json:"skipped"`
	Warnings   []string `json:"warnings"`
	SubUnitIDs []string `json:"sub_unit_ids"`
}

type DivisionTemplateImportResult struct {
	Created       int      `json:"created"`
	Updated       int      `json:"updated"`
	Skipped       int      `json:"skipped"`
	Warnings      []string `json:"warnings"`
	TemplateNames []string `json:"template_names"`
}

func militaryOverwrite(policy string) bool {
	if policy == "" {
		policy = "overwrite"
	}
	return strings.ToLower(policy) != "skip"
}

type importPlan struct {
	Selected []int
	Actions  []string
	Created  int
	Updated  int
	Skipped  int
}

func planImport(keys []string, existing map[string]bool, overwrite bool) importPlan {
	plan := importPlan{Actions: make([]string, len(keys))}
	seen := make(map[string]bool, len(existing)+len(keys))
	for key, present := range existing {
		seen[key] = present
	}
	selectedByKey := map[string]int{}
	for index, key := range keys {
		if key == "" {
			continue
		}
		if seen[key] {
			if !overwrite {
				plan.Actions[index] = "skipped"
				plan.Skipped++
				continue
			}
			plan.Actions[index] = "updated"
			plan.Updated++
		} else {
			plan.Actions[index] = "created"
			plan.Created++
			seen[key] = true
		}
		selectedByKey[key] = index
	}
	plan.Selected = make([]int, 0, len(selectedByKey))
	for _, index := range selectedByKey {
		plan.Selected = append(plan.Selected, index)
	}
	sort.Ints(plan.Selected)
	return plan
}

func successfulImportKeys(keys []string, actions []string) []string {
	out := make([]string, 0, len(keys))
	for index, action := range actions {
		if action == "created" || action == "updated" {
			out = append(out, keys[index])
		}
	}
	return out
}

func selectedImportKeys(keys []string, selected []int) []string {
	out := make([]string, 0, len(selected))
	for _, index := range selected {
		out = append(out, keys[index])
	}
	return out
}

func wireRow(row map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(row)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out map[string]any
	err = decoder.Decode(&out)
	return out, err
}

func wireText(p map[string]any, key string) (string, bool) {
	switch value := p[key].(type) {
	case nil:
		return "", false
	case string:
		return value, true
	case json.Number:
		return value.String(), true
	case bool:
		return strconv.FormatBool(value), true
	default:
		raw, _ := json.Marshal(value)
		return string(raw), true
	}
}

func wireString(p map[string]any, key string) string {
	s, _ := wireText(p, key)
	return s
}

func wireInt(p map[string]any, key string) (any, error) {
	s, ok := wireText(p, key)
	if !ok || s == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return nil, errors.New("invalid input syntax for type integer: \"" + s + "\"")
	}
	return json.Number(strconv.FormatInt(n, 10)), nil
}

func wireFloat(p map[string]any, key string) (any, error) {
	s, ok := wireText(p, key)
	if !ok || s == "" {
		return nil, nil
	}
	if _, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err != nil {
		return nil, errors.New("invalid input syntax for type double precision: \"" + s + "\"")
	}
	return json.Number(strings.TrimSpace(s)), nil
}

func wireBool(p map[string]any, key string) (any, error) {
	s, ok := wireText(p, key)
	if !ok || s == "" {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "t", "yes", "y", "on", "1":
		return true, nil
	case "false", "f", "no", "n", "off", "0":
		return false, nil
	}
	return nil, errors.New("invalid input syntax for type boolean: \"" + s + "\"")
}

func wireJSON(p map[string]any, key string, def any) any {
	if value, ok := p[key]; ok {
		return value
	}
	return def
}

type importColumns struct {
	strings, ints, floats, bools []string
}

func (c importColumns) fill(row project.Row, p map[string]any) error {
	for _, key := range c.strings {
		row[key] = wireString(p, key)
	}
	for _, group := range []struct {
		keys []string
		cast func(map[string]any, string) (any, error)
	}{{c.ints, wireInt}, {c.floats, wireFloat}, {c.bools, wireBool}} {
		for _, key := range group.keys {
			value, err := group.cast(p, key)
			if err != nil {
				return err
			}
			row[key] = value
		}
	}
	return nil
}

var equipmentImportColumns = importColumns{
	strings: []string{"equipment_type", "picture", "archetype", "parent", "source_file"},
	ints:    []string{"year", "manpower"},
	floats: []string{
		"soft_attack", "hard_attack", "ap_attack", "air_attack", "breakthrough",
		"defense", "armor_value", "hardness", "max_strength", "build_cost_ic",
		"reliability", "maximum_speed", "naval_speed", "fire_range",
		"lg_armor_piercing", "lg_attack", "hg_armor_piercing", "hg_attack",
		"torpedo_attack", "anti_air_attack", "shore_bombardment", "evasion",
		"surface_detection", "sub_attack", "sub_detection", "surface_visibility",
		"sub_visibility", "naval_range", "port_capacity_usage",
		"search_and_destroy_coordination", "convoy_raiding_coordination",
		"air_defence", "air_range", "air_agility", "air_ground_attack",
		"air_bombing", "air_superiority", "naval_strike_attack",
		"naval_strike_targetting", "carrier_size", "default_carrier_composition_weight",
	},
	bools: []string{"is_buildable", "can_license", "carrier_capable"},
}

var subUnitImportColumns = importColumns{
	strings: []string{"source_file", "abbreviation", "sprite", "map_icon_category", "transport", "group"},
	ints:    []string{"priority", "ai_priority"},
	bools: []string{
		"active", "cavalry", "special_forces", "marines", "can_exfiltrate_from_coast",
		"mountaineers", "can_be_parachuted", "affects_speed",
	},
}

type importRows struct {
	table   string
	rows    []map[string]any
	keys    []string
	inScope func(project.Row) bool
	keyOf   func(project.Row) string
	build   func(p map[string]any) (project.Row, error)
}

func (b importRows) apply(write writer, overwrite bool) (importPlan, error) {
	var plan importPlan
	err := write([]string{b.table}, func(tx *project.Tx) error {
		existing := map[string]bool{}
		for _, row := range tx.Rows(b.table) {
			if b.inScope(row) {
				existing[b.keyOf(row)] = true
			}
		}
		wanted := map[string]bool{}
		for _, key := range b.keys {
			if key != "" {
				wanted[key] = true
			}
		}
		for key := range existing {
			if !wanted[key] {
				delete(existing, key)
			}
		}
		plan = planImport(b.keys, existing, overwrite)
		if len(plan.Selected) == 0 {
			return nil
		}
		built := make([]project.Row, 0, len(plan.Selected))
		for _, index := range plan.Selected {
			payload, err := wireRow(b.rows[index])
			if err != nil {
				return err
			}
			row, err := b.build(payload)
			if err != nil {
				return err
			}
			built = append(built, row)
		}
		if overwrite {
			selected := map[string]bool{}
			for _, key := range selectedImportKeys(b.keys, plan.Selected) {
				selected[key] = true
			}
			tx.Delete(b.table, func(row project.Row) bool { return b.inScope(row) && selected[b.keyOf(row)] })
		}
		for _, row := range built {
			tx.Insert(b.table, row)
		}
		return nil
	})
	return plan, err
}

func importEquipmentText(p *project.Project, body, sourceFile, policy string) (EquipmentImportResult, error) {
	result := EquipmentImportResult{Warnings: []string{}, EquipmentIDs: []string{}}
	parsed := militaryparser.ImportEquipment(body, sourceFile)
	result.Warnings = parsed.Warnings
	keys := make([]string, len(parsed.Rows))
	for index, row := range parsed.Rows {
		keys[index], _ = row["equipment_id"].(string)
	}
	batch := importRows{
		table: equipmentTable, rows: parsed.Rows, keys: keys,
		inScope: func(project.Row) bool { return true },
		keyOf:   func(row project.Row) string { return text(row, "equipment_id") },
		build: func(payload map[string]any) (project.Row, error) {
			row := project.Row{
				"country_tag": "", "equipment_id": wireString(payload, "equipment_id"), "name": "",
				"resources":  wireJSON(payload, "resources", map[string]any{}),
				"raw_extras": wireJSON(payload, "raw_extras", map[string]any{}),
				"source":     "created", "name_en": "", "name_zh": "", "desc_en": "", "desc_zh": "",
				"display_name_locs": map[string]any{}, "description_locs": map[string]any{},
			}
			isArchetype, err := wireBool(payload, "is_archetype")
			if err != nil {
				return nil, err
			}
			if isArchetype == nil {
				isArchetype = false
			}
			row["is_archetype"] = isArchetype
			return row, equipmentImportColumns.fill(row, payload)
		},
	}
	plan, err := batch.apply(p.Write, militaryOverwrite(policy))
	if err != nil {
		return result, err
	}
	result.Created, result.Updated, result.Skipped = plan.Created, plan.Updated, plan.Skipped
	result.EquipmentIDs = successfulImportKeys(keys, plan.Actions)
	return result, nil
}

func normalizeImportSourceFile(value string) (string, error) {
	path := strings.Trim(strings.TrimSpace(strings.ReplaceAll(value, "\\", "/")), "/")
	if path == "" {
		return "", nil
	}
	parts := strings.Split(path, "/")
	valid := len(parts) == 3 && parts[0] == "common" && parts[1] == "units" &&
		strings.HasSuffix(strings.ToLower(parts[2]), ".txt") && !strings.Contains(path, ":")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			valid = false
		}
	}
	if !valid {
		return "", detailError(http.StatusUnprocessableEntity, "source_file must be a mod-relative common/units/*.txt path")
	}
	return path, nil
}

func stringSliceField(row map[string]any, key string) []string {
	if values, ok := row[key].([]string); ok {
		return values
	}
	return []string{}
}

func importSubUnitsText(p *project.Project, body, rawSourceFile, policy string) (SubUnitImportResult, error) {
	result := SubUnitImportResult{Warnings: []string{}, SubUnitIDs: []string{}}
	sourceFile, err := normalizeImportSourceFile(rawSourceFile)
	if err != nil {
		return result, err
	}
	parsed := militaryparser.ImportSubUnits(body, sourceFile)
	result.Warnings = parsed.Warnings
	keys := make([]string, len(parsed.Rows))
	for index, row := range parsed.Rows {
		keys[index], _ = row["sub_unit_id"].(string)
		unitTypes, err := validateSubUnitTypes(stringSliceField(row, "unit_types"))
		if err != nil {
			return result, detailError(http.StatusUnprocessableEntity, err.Error())
		}
		categories, err := validateSubUnitCategories(stringSliceField(row, "categories"))
		if err != nil {
			return result, detailError(http.StatusUnprocessableEntity, err.Error())
		}
		row["unit_types"], row["categories"] = unitTypes, categories
	}
	batch := importRows{
		table: subUnitTable, rows: parsed.Rows, keys: keys,
		inScope: func(project.Row) bool { return true },
		keyOf:   func(row project.Row) string { return text(row, "sub_unit_id") },
		build: func(payload map[string]any) (project.Row, error) {
			row := project.Row{
				"sub_unit_id": wireString(payload, "sub_unit_id"), "country_tag": "", "source": "created",
				"name": "", "name_en": "", "name_zh": "", "desc_en": "", "desc_zh": "",
				"display_name_locs": map[string]any{}, "description_locs": map[string]any{},
				"unit_types":        wireJSON(payload, "unit_types", []any{}),
				"categories":        wireJSON(payload, "categories", []any{}),
				"essential":         wireJSON(payload, "essential", []any{}),
				"need":              wireJSON(payload, "need", map[string]any{}),
				"stats":             wireJSON(payload, "stats", map[string]any{}),
				"terrain_modifiers": wireJSON(payload, "terrain_modifiers", map[string]any{}),
				"raw_extras":        wireJSON(payload, "raw_extras", map[string]any{}),
			}
			ordering, err := wireInt(payload, "ordering")
			if err != nil {
				return nil, err
			}
			if ordering == nil {
				ordering = json.Number("0")
			}
			row["ordering"] = ordering
			return row, subUnitImportColumns.fill(row, payload)
		},
	}
	plan, err := batch.apply(p.Write, militaryOverwrite(policy))
	if err != nil {
		return result, err
	}
	result.Created, result.Updated, result.Skipped = plan.Created, plan.Updated, plan.Skipped
	result.SubUnitIDs = successfulImportKeys(keys, plan.Actions)
	return result, nil
}

func importDivisionTemplatesText(write writer, body, rawOobFile, policy string) (DivisionTemplateImportResult, error) {
	result := DivisionTemplateImportResult{Warnings: []string{}, TemplateNames: []string{}}
	oobFile := strings.TrimSpace(rawOobFile)
	if oobFile == "" {
		return result, detailError(http.StatusUnprocessableEntity, "oob_file is required")
	}
	parsed := militaryparser.ImportDivisionTemplates(body)
	result.Warnings = parsed.Warnings
	keys := make([]string, len(parsed.Rows))
	for index, row := range parsed.Rows {
		keys[index], _ = row["template_name"].(string)
	}
	batch := importRows{
		table: divisionTemplateTable, rows: parsed.Rows, keys: keys,
		inScope: func(row project.Row) bool { return text(row, "oob_file") == oobFile },
		keyOf:   func(row project.Row) string { return text(row, "template_name") },
		build: func(payload map[string]any) (project.Row, error) {
			row := project.Row{
				"oob_file": oobFile, "country_tag": tagFromOOB(oobFile), "oob_units_tail": parsed.Tail,
				"source": "created", "template_name": wireString(payload, "template_name"),
				"division_names_group": wireString(payload, "division_names_group"),
				"regiments":            wireJSON(payload, "regiments", []any{}),
				"support":              wireJSON(payload, "support", []any{}),
				"raw_extras":           wireJSON(payload, "raw_extras", map[string]any{}),
			}
			var err error
			if row["is_locked"], err = wireBool(payload, "is_locked"); err != nil {
				return nil, err
			}
			if row["priority"], err = wireInt(payload, "priority"); err != nil {
				return nil, err
			}
			ordering, err := wireInt(payload, "ordering")
			if err != nil {
				return nil, err
			}
			if ordering == nil {
				ordering = json.Number("0")
			}
			row["ordering"] = ordering
			return row, nil
		},
	}
	plan, err := batch.apply(write, militaryOverwrite(policy))
	if err != nil {
		return result, err
	}
	result.Created, result.Updated, result.Skipped = plan.Created, plan.Updated, plan.Skipped
	result.TemplateNames = successfulImportKeys(keys, plan.Actions)
	return result, nil
}
