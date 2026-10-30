package rawfiles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/importsvc"
	characterimport "haganeworks/internal/hoi4/importsvc/characters"
	treeimport "haganeworks/internal/hoi4/importsvc/trees"
	"haganeworks/internal/hoi4/loc"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

const maxSingleImportBytes = 256 << 20

type SingleImportResult struct {
	Total           int            `json:"total"`
	Created         int            `json:"created"`
	Updated         int            `json:"updated"`
	Skipped         int            `json:"skipped"`
	PreservedFields map[string]int `json:"preserved_fields"`
	Warnings        []string       `json:"warnings"`
	ConflictPolicy  string         `json:"conflict_policy"`
}

func readSingleImportUpload(w http.ResponseWriter, r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSingleImportBytes+(4<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return "", apiutil.ValidationError("invalid multipart form")
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		return "", apiutil.ValidationError("file is required")
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxSingleImportBytes+1))
	if err != nil {
		return "", apiutil.ValidationError("failed to read upload")
	}
	if len(raw) > maxSingleImportBytes {
		return "", &apiutil.StatusError{Status: http.StatusRequestEntityTooLarge,
			Body: map[string]string{"detail": "import file too large"}}
	}
	text, _, ok := loc.DecodeScriptText(raw)
	if !ok {
		return "", apiutil.ValidationError("file must be UTF-8 or Windows-1252 text")
	}
	return text, nil
}

func singleImportPolicy(raw string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(raw))
	if p == "" {
		return "overwrite", nil
	}
	if p != "overwrite" && p != "skip" {
		return "", apiutil.BadRequestError(fmt.Sprintf("conflict_policy must be one of {'overwrite', 'skip'}, got %q", raw))
	}
	return p, nil
}

type singleImportBatchPlan struct {
	Selected []int
	Actions  []string
	Created  int
	Updated  int
	Skipped  int
}

func planSingleImportBatch(keys []string, existing map[string]bool, policy string) singleImportBatchPlan {
	plan := singleImportBatchPlan{Actions: make([]string, len(keys))}
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
			if policy == "skip" {
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

func summaryFields(s *importsvc.ImportSummary) map[string]int {
	out := map[string]int{}
	if s == nil || s.PreservedFields == nil {
		return out
	}
	for _, key := range s.PreservedFields.Keys {
		out[key] = s.PreservedFields.Get(key)
	}
	return out
}

func jsonValue(v any) any {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	raw = bytes.ReplaceAll(raw, []byte(`\u0000`), nil)
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out any
	if err := decoder.Decode(&out); err != nil {
		return nil
	}
	return out
}

type payload map[string]any

func (p payload) text(key string) (string, bool) {
	value, ok := p[key]
	if !ok || value == nil {
		return "", false
	}
	switch v := value.(type) {
	case string:
		return v, true
	case json.Number:
		return v.String(), true
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

func (p payload) textOr(key, fallback string) string {
	if v, ok := p.text(key); ok {
		return v
	}
	return fallback
}

func (p payload) textOrNull(key string) any {
	if v, ok := p.text(key); ok {
		return v
	}
	return nil
}

func (p payload) json(key string, fallback any) any {
	if value, ok := p[key]; ok {
		return value
	}
	return fallback
}

type errCast struct{}

func (errCast) Error() string { return "cast failed" }

func (p payload) integer(key string, fallback any) (any, error) {
	v, ok := p.text(key)
	if !ok {
		return fallback, nil
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil || n < math.MinInt32 || n > math.MaxInt32 {
		return nil, errCast{}
	}
	return n, nil
}

func (p payload) float(key string, fallback any) (any, error) {
	v, ok := p.text(key)
	if !ok {
		return fallback, nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return nil, errCast{}
	}
	return f, nil
}

func (p payload) boolean(key string, fallback any) (any, error) {
	v, ok := p.text(key)
	if !ok {
		return fallback, nil
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "t", "true", "y", "yes", "on", "1":
		return true, nil
	case "f", "false", "n", "no", "off", "0":
		return false, nil
	}
	return nil, errCast{}
}

func (p payload) nullableBoolean(key string) (any, error) {
	if v, ok := p.text(key); !ok || v == "" {
		return nil, nil
	}
	return p.boolean(key, nil)
}

func unavailable(family string) error {
	return &apiutil.StatusError{Status: http.StatusServiceUnavailable,
		Body: map[string]string{"detail": family + " import unavailable"}}
}

func replaceByKey(p *project.Project, tableName, keyColumn string, keys []string, policy string,
	result *SingleImportResult, extra func(tx *project.Tx), build func(index int) (project.Row, error)) error {

	tables := []string{tableName}
	return p.Write(tables, func(tx *project.Tx) error {
		if extra != nil {
			extra(tx)
		}
		existing := map[string]bool{}
		for _, row := range tx.Rows(tableName) {
			existing[str(row, keyColumn)] = true
		}
		plan := planSingleImportBatch(keys, existing, policy)
		result.Created, result.Updated, result.Skipped = plan.Created, plan.Updated, plan.Skipped
		if policy == "overwrite" && len(plan.Selected) > 0 {
			replaced := map[string]bool{}
			for _, index := range plan.Selected {
				replaced[keys[index]] = true
			}
			tx.Delete(tableName, func(row project.Row) bool { return replaced[str(row, keyColumn)] })
			for key := range replaced {
				delete(existing, key)
			}
		}
		for _, index := range plan.Selected {
			row, err := build(index)
			if err != nil {
				return err
			}
			key := str(row, keyColumn)
			if existing[key] {
				continue
			}
			existing[key] = true
			tx.Insert(tableName, row)
		}
		return nil
	})
}

func importCharacters(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	text, err := readSingleImportUpload(w, r)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err != nil {
		return err
	}
	result, err := importCharacterScript(p, text, r.URL.Query().Get("country_tag"), r.FormValue("conflict_policy"))
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}

var characterOwnerPrefixRE = regexp.MustCompile(`^[A-Z]{2,5}$`)

var dlcPrefixes = map[string]bool{
	"MTG": true, "MtG": true, "WTT": true, "WP": true, "LAR": true, "BBA": true,
	"NSB": true, "TFV": true, "BFTB": true, "AAT": true, "TOA": true, "GOT": true,
	"GOE": true, "GTD": true, "DOD": true, "AOA": true, "WUW": true, "MPP": true,
	"GW": true, "GFX": true, "TFR": true,
}

func singleString(row map[string]any, key, fallback string) string {
	if value, ok := row[key].(string); ok {
		return value
	}
	return fallback
}

func singleBool(row map[string]any, key string, fallback bool) bool {
	if value, ok := row[key].(bool); ok {
		return value
	}
	return fallback
}

func singleValue(row map[string]any, key string, fallback any) any {
	if value, ok := row[key]; ok && value != nil {
		return value
	}
	return fallback
}

func characterImportCountry(tag, fallback string) string {
	prefix, _, found := strings.Cut(tag, "_")
	if !found || !characterOwnerPrefixRE.MatchString(prefix) || dlcPrefixes[prefix] {
		return fallback
	}
	return prefix
}

var characterSnapshotKeys = []string{
	"advisor_available", "advisor_cost", "advisor_idea_token", "advisor_modifier",
	"advisor_on_add", "advisor_on_remove", "advisor_removal_cost", "advisor_slot",
	"advisor_traits", "advisor_visible", "always_show_on_actions_tooltip", "attack_skill",
	"can_be_captured", "coordination_skill", "country_tag", "defense_skill", "desc_en",
	"desc_zh", "description", "description_locs", "expire", "gender", "has_advisor_role",
	"ideology", "is_portrait_modified", "is_replacement", "large_portrait",
	"logistics_skill", "display_name_locs", "localisation_key", "maneuvering_skill",
	"military_visible", "name", "name_en", "name_zh", "operative_available",
	"operative_bg", "operative_nationalities", "operative_skill_level", "operative_traits",
	"planning_skill", "raw_extras", "replaces_vanilla_id", "role_type", "scientist_data",
	"scientist_visible", "skill", "small_portrait", "tag", "traits",
}

func characterImportRecord(parsed map[string]any, scopeTag string) map[string]any {
	tag := singleString(parsed, "tag", "")
	name := singleString(parsed, "name", tag)
	record := map[string]any{
		"country_tag": characterImportCountry(tag, scopeTag), "tag": tag,
		"name": name, "localisation_key": singleString(parsed, "localisation_key", ""),
		"description": "", "name_en": singleString(parsed, "name_en", tag),
		"name_zh": singleString(parsed, "name_zh", ""), "desc_en": "", "desc_zh": "",
		"display_name_locs": singleValue(parsed, "display_name_locs", map[string]any{}),
		"description_locs":  singleValue(parsed, "description_locs", map[string]any{}),
		"role_type":         singleString(parsed, "role_type", "advisor"),
		"has_advisor_role":  singleBool(parsed, "has_advisor_role", false),
		"is_replacement":    true, "replaces_vanilla_id": true, "is_portrait_modified": false,
		"skill":                          singleValue(parsed, "skill", int64(1)),
		"attack_skill":                   singleValue(parsed, "attack_skill", int64(1)),
		"defense_skill":                  singleValue(parsed, "defense_skill", int64(1)),
		"planning_skill":                 singleValue(parsed, "planning_skill", int64(1)),
		"logistics_skill":                singleValue(parsed, "logistics_skill", int64(1)),
		"maneuvering_skill":              singleValue(parsed, "maneuvering_skill", int64(1)),
		"coordination_skill":             singleValue(parsed, "coordination_skill", int64(1)),
		"military_visible":               singleString(parsed, "military_visible", ""),
		"traits":                         singleValue(parsed, "traits", []any{}),
		"advisor_traits":                 singleValue(parsed, "advisor_traits", []any{}),
		"advisor_slot":                   singleString(parsed, "advisor_slot", ""),
		"advisor_idea_token":             singleString(parsed, "advisor_idea_token", ""),
		"advisor_cost":                   singleValue(parsed, "advisor_cost", int64(150)),
		"advisor_available":              singleString(parsed, "advisor_available", ""),
		"advisor_visible":                singleString(parsed, "advisor_visible", ""),
		"advisor_removal_cost":           singleValue(parsed, "advisor_removal_cost", int64(-1)),
		"advisor_modifier":               singleString(parsed, "advisor_modifier", ""),
		"advisor_on_add":                 singleString(parsed, "advisor_on_add", ""),
		"advisor_on_remove":              singleString(parsed, "advisor_on_remove", ""),
		"gender":                         singleString(parsed, "gender", ""),
		"can_be_captured":                singleBool(parsed, "can_be_captured", true),
		"always_show_on_actions_tooltip": parsed["always_show_on_actions_tooltip"],
		"ideology":                       singleString(parsed, "ideology", ""), "expire": singleString(parsed, "expire", ""),
		"scientist_data":        singleValue(parsed, "scientist_data", map[string]any{}),
		"scientist_visible":     singleString(parsed, "scientist_visible", ""),
		"operative_skill_level": int64(1), "operative_traits": []any{},
		"operative_nationalities": []any{}, "operative_available": "", "operative_bg": "",
		"raw_extras":     singleValue(parsed, "raw_extras", map[string]any{}),
		"source_file":    singleString(parsed, "source_file", ""),
		"large_portrait": singleString(parsed, "large_portrait", ""),
		"small_portrait": singleString(parsed, "small_portrait", ""),
	}
	snapshot := make(map[string]any, len(characterSnapshotKeys))
	for _, key := range characterSnapshotKeys {
		snapshot[key] = record[key]
	}
	record["_source_snapshot"] = snapshot
	return record
}

func characterRow(record map[string]any) (project.Row, error) {
	values, ok := jsonValue(record).(map[string]any)
	if !ok {
		return nil, errCast{}
	}
	p := payload(values)
	row := project.Row{
		"country_tag": p.textOrNull("country_tag"), "tag": p.textOrNull("tag"),
		"name": p.textOrNull("name"), "localisation_key": p.textOrNull("localisation_key"),
		"description": p.textOrNull("description"), "role_type": p.textOrNull("role_type"),
		"is_replacement": true, "replaces_vanilla_id": true, "is_portrait_modified": false,
		"military_visible": p.textOrNull("military_visible"),
		"traits":           p.json("traits", nil), "advisor_traits": p.json("advisor_traits", nil),
		"advisor_slot": p.textOrNull("advisor_slot"), "advisor_idea_token": p.textOrNull("advisor_idea_token"),
		"advisor_available": p.textOrNull("advisor_available"), "advisor_visible": p.textOrNull("advisor_visible"),
		"advisor_modifier": p.textOrNull("advisor_modifier"), "advisor_on_add": p.textOrNull("advisor_on_add"),
		"advisor_on_remove": p.textOrNull("advisor_on_remove"), "gender": p.textOrNull("gender"),
		"ideology": p.textOrNull("ideology"), "expire": p.textOrNull("expire"),
		"scientist_data": p.json("scientist_data", nil), "scientist_visible": p.textOrNull("scientist_visible"),
		"operative_traits": p.json("operative_traits", nil), "operative_nationalities": p.json("operative_nationalities", nil),
		"operative_available": p.textOrNull("operative_available"), "operative_bg": p.textOrNull("operative_bg"),
		"raw_extras": p.json("raw_extras", nil), "source": "created", "source_file": p.textOrNull("source_file"),
		"source_snapshot": p.json("_source_snapshot", nil),
		"large_portrait":  p.textOrNull("large_portrait"), "small_portrait": p.textOrNull("small_portrait"),
		"name_en": p.textOrNull("name_en"), "name_zh": p.textOrNull("name_zh"),
		"desc_en": p.textOrNull("desc_en"), "desc_zh": p.textOrNull("desc_zh"),
		"display_name_locs": p.json("display_name_locs", nil), "description_locs": p.json("description_locs", nil),
		"has_obs": false,
	}
	for _, key := range []string{"skill", "attack_skill", "defense_skill", "planning_skill", "logistics_skill",
		"maneuvering_skill", "coordination_skill", "advisor_cost", "advisor_removal_cost", "operative_skill_level"} {
		n, err := p.integer(key, nil)
		if err != nil {
			return nil, err
		}
		row[key] = n
	}
	for _, key := range []string{"has_advisor_role", "can_be_captured"} {
		b, err := p.boolean(key, nil)
		if err != nil {
			return nil, err
		}
		row[key] = b
	}
	tooltip, err := p.nullableBoolean("always_show_on_actions_tooltip")
	if err != nil {
		return nil, err
	}
	row["always_show_on_actions_tooltip"] = tooltip

	for _, key := range []string{"name", "tag", "localisation_key", "role_type", "source_file"} {
		if row[key] == nil {
			return nil, errCast{}
		}
	}
	return row, nil
}

func importCharacterScript(p *project.Project, text, countryTag, rawPolicy string) (SingleImportResult, error) {
	policy, err := singleImportPolicy(rawPolicy)
	if err != nil {
		return SingleImportResult{}, err
	}
	parsed, summary, err := characterimport.ImportCharactersStrict(text)
	if err != nil {
		return SingleImportResult{}, apiutil.ValidationError(err.Error())
	}
	result := SingleImportResult{Total: summary.Total, PreservedFields: summaryFields(summary),
		Warnings: append([]string(nil), summary.Warnings...), ConflictPolicy: policy}
	if len(parsed) == 0 && len(result.Warnings) == 0 {
		result.Warnings = []string{"No characters found in file"}
	}
	scopeTag := countryTag
	if scopeTag == "" {
		scopeTag, _ = p.Record()["country_tag"].(string)
	}
	nativeRows := characterimport.NativeEntities(parsed)
	keys := make([]string, len(nativeRows))
	for i := range nativeRows {
		keys[i] = singleString(nativeRows[i], "tag", "")
	}
	err = replaceByKey(p, "characters", "tag", keys, policy, &result, nil, func(index int) (project.Row, error) {
		return characterRow(characterImportRecord(nativeRows[index], scopeTag))
	})
	if _, ok := err.(errCast); ok {
		return SingleImportResult{}, unavailable("character")
	}
	if err != nil {
		return SingleImportResult{}, err
	}
	return result, nil
}

func importTechnologies(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	techLine := strings.TrimSpace(r.URL.Query().Get("tech_line"))
	folderName := strings.TrimSpace(r.URL.Query().Get("folder_name"))
	if techLine == "" || folderName == "" {
		return apiutil.ValidationError("tech_line and folder_name are required")
	}
	text, err := readSingleImportUpload(w, r)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err != nil {
		return err
	}
	result, err := importTechnologyScript(p, text, techLine, folderName, r.FormValue("conflict_policy"))
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}

func nativeTreeValue(value any) any {
	switch v := value.(type) {
	case *treeimport.RawDict:
		out := make(map[string]any, len(v.Keys))
		for _, key := range v.Keys {
			item, _ := v.Get(key)
			out[key] = nativeTreeValue(item)
		}
		return out
	case paradox.Scalar:
		switch v.Kind {
		case paradox.ScalarInt:
			if v.Int.IsInt64() {
				return v.Int.Int64()
			}
			return v.Int.String()
		case paradox.ScalarFloat:
			return v.Float
		default:
			return v.Str
		}
	case *big.Int:
		if v.IsInt64() {
			return v.Int64()
		}
		return v.String()
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = nativeTreeValue(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = nativeTreeValue(v[i])
		}
		return out
	case []map[string]any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = nativeTreeValue(v[i])
		}
		return out
	case []string:
		return v
	default:
		return value
	}
}

type orderedYear struct {
	Name string
	Row  int
}

func mergeYearColumns(stored any, detected []treeimport.YearColumn) []orderedYear {
	rows := []orderedYear{}
	positions := map[string]int{}
	list, _ := stored.([]any)
	for _, item := range list {
		pair, ok := item.([]any)
		if !ok || len(pair) != 2 {
			continue
		}
		name, ok := pair[0].(string)
		if !ok {
			continue
		}
		var row int
		switch n := pair[1].(type) {
		case float64:
			row = int(n)
		case int64:
			row = int(n)
		case int:
			row = n
		case json.Number:
			parsed, err := n.Int64()
			if err != nil {
				continue
			}
			row = int(parsed)
		default:
			continue
		}
		if pos, exists := positions[name]; exists {
			rows[pos].Row = row
		} else {
			positions[name] = len(rows)
			rows = append(rows, orderedYear{Name: name, Row: row})
		}
	}
	for _, item := range detected {
		if !item.Row.IsInt64() {
			continue
		}
		row := int(item.Row.Int64())
		if pos, exists := positions[item.Name]; exists {
			rows[pos].Row = row
		} else {
			positions[item.Name] = len(rows)
			rows = append(rows, orderedYear{Name: item.Name, Row: row})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Row < rows[j].Row })
	return rows
}

func technologyRow(parsed map[string]any, techLine, folderName string) (project.Row, error) {
	values, ok := jsonValue(nativeTreeValue(parsed)).(map[string]any)
	if !ok {
		return nil, errCast{}
	}
	p := payload(values)
	techID, ok := p.text("tech_id")
	if !ok {
		return nil, errCast{}
	}
	row := project.Row{
		"country_tag": "", "tech_id": techID, "name": p.textOr("name", techID),
		"name_en": p.textOr("name_en", techID), "name_zh": "", "desc_en": "", "desc_zh": "",
		"display_name_locs": map[string]any{}, "description_locs": map[string]any{},
		"tech_line": p.textOr("tech_line", techLine), "folder": p.textOr("folder", folderName),
		"folder_placements": nil,
		"prerequisites":     p.json("prerequisites", []any{}), "categories": p.json("categories", []any{}),
		"modifiers": p.json("modifiers", map[string]any{}), "ai_will_do": p.textOr("ai_will_do", "factor = 1"),
		"enable_equipments": p.json("enable_equipments", []any{}), "enable_subunits": p.json("enable_subunits", []any{}),
		"enable_equipment_modules": p.json("enable_equipment_modules", []any{}),
		"enable_building":          p.textOr("enable_building", ""), "enable_tactic": p.textOr("enable_tactic", ""),
		"xor": p.json("xor", []any{}), "dependencies": p.json("dependencies", map[string]any{}),
		"sub_technologies": p.json("sub_technologies", []any{}),
		"allow":            p.textOr("allow", ""), "allow_branch": p.textOr("allow_branch", ""),
		"on_research_complete":       p.textOr("on_research_complete", ""),
		"on_research_complete_limit": p.textOr("on_research_complete_limit", ""),
		"path":                       p.json("path", []any{}), "xp_research_type": p.textOr("xp_research_type", ""),
		"xp_research_bonus": p.json("xp_research_bonus", map[string]any{}),
		"raw_extras":        p.json("raw_extras", map[string]any{}),
		"source":            "created", "is_banner": false, "has_obs": false, "source_file": "",
	}
	type cast struct {
		key      string
		fallback any
		fn       func(string, any) (any, error)
	}
	for _, c := range []cast{
		{"research_cost", float64(1), p.float}, {"start_year", int64(1936), p.integer},
		{"position_x", float64(0), p.float}, {"position_y", float64(0), p.float},
		{"doctrine", false, p.boolean}, {"xp_boost_cost", int64(0), p.integer},
		{"force_use_small_tech_layout", false, p.boolean},
	} {
		value, err := c.fn(c.key, c.fallback)
		if err != nil {
			return nil, err
		}
		row[c.key] = value
	}
	explicit, err := p.nullableBoolean("position_y_is_explicit")
	if err != nil {
		return nil, err
	}
	row["position_y_is_explicit"] = explicit
	return row, nil
}

func importTechnologyScript(p *project.Project, text, techLine, folderName, rawPolicy string) (SingleImportResult, error) {
	policy, err := singleImportPolicy(rawPolicy)
	if err != nil {
		return SingleImportResult{}, err
	}
	techs, years, summary, err := treeimport.ImportTechnologies(text, techLine, folderName)
	if err != nil {
		return SingleImportResult{}, apiutil.ValidationError(err.Error())
	}
	result := SingleImportResult{
		Total: summary.Total, PreservedFields: summaryFields(summary),
		Warnings: append([]string(nil), summary.Warnings...), ConflictPolicy: policy,
	}
	if len(techs) == 0 && len(result.Warnings) == 0 {
		result.Warnings = []string{"No technologies found in file"}
	}
	keys := make([]string, len(techs))
	for i := range techs {
		keys[i], _ = techs[i]["tech_id"].(string)
	}

	mergeYears := func(tx *project.Tx) {
		if len(years) == 0 {
			return
		}
		merged := mergeYearColumns(tx.Record()["tech_year_columns"], years)
		wire := make([]any, 0, len(merged))
		for _, item := range merged {
			wire = append(wire, []any{item.Name, int64(item.Row)})
		}
		tx.SetRecord(func(record project.Row) {
			record["tech_year_columns"] = wire
			record["updated_at"] = schema.Now("projects", "updated_at", time.Now())
		})
	}
	err = replaceByKey(p, "technologies", "tech_id", keys, policy, &result, mergeYears, func(index int) (project.Row, error) {
		return technologyRow(techs[index], techLine, folderName)
	})
	if _, ok := err.(errCast); ok {
		return SingleImportResult{}, unavailable("technology")
	}
	if err != nil {
		return SingleImportResult{}, err
	}
	return result, nil
}
