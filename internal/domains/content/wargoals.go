package content

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const wargoalsTable = "wargoal_types"

var wargoalSafeScalarKeys = map[string]bool{
	"generate_base_cost": true, "generate_per_state_cost": true,
	"take_states_limit": true, "take_states_cost": true, "puppet_cost": true,
	"force_government_cost": true, "expire": true, "threat": true,
	"take_states_threat_factor": true,
}

var wargoalBlockFields = []string{"allowed", "available", "take_states", "puppet", "liberate"}

func wargoalKnownField(key string) bool {
	if key == "war_name" || wargoalSafeScalarKeys[key] {
		return true
	}
	for _, f := range wargoalBlockFields {
		if key == f {
			return true
		}
	}
	return false
}

type wargoalRecord struct {
	Key    string `json:"key"`
	Script string `json:"script"`
}

func parseWargoalAdvancedEntries(text string) ([]wargoalRecord, error) {
	if strings.TrimSpace(text) == "" {
		return []wargoalRecord{}, nil
	}
	block, recoveries, err := paradox.Parse(text)
	if err != nil || len(recoveries) > 0 {
		return nil, fmt.Errorf("Advanced script contains malformed HOI4 script")
	}
	records := []wargoalRecord{}
	for _, entry := range block.Entries {
		if entry.Key == nil {
			return nil, fmt.Errorf("Advanced script only accepts key = value entries")
		}
		key := *entry.Key
		if wargoalKnownField(key) {
			return nil, fmt.Errorf("%s belongs in the visual fields, not Advanced raw script", key)
		}
		records = append(records, wargoalRecord{
			Key:    key,
			Script: strings.Trim(importsvc.ToScriptEntries([]*paradox.Entry{entry}, 0), "\n"),
		})
	}
	return records, nil
}

func wargoalAdvancedScriptFromRecords(records []wargoalRecord) string {
	var parts []string
	for _, record := range records {
		if wargoalKnownField(record.Key) {
			continue
		}
		if script := strings.Trim(record.Script, "\n"); script != "" {
			parts = append(parts, script)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func wargoalReplaceAdvancedRecords(existing, advanced []wargoalRecord) []wargoalRecord {
	result := []wargoalRecord{}
	inserted := false
	for _, record := range existing {
		if wargoalKnownField(record.Key) {
			result = append(result, record)
		} else if !inserted {
			result = append(result, advanced...)
			inserted = true
		}
	}
	if !inserted {
		result = append(result, advanced...)
	}
	return result
}

func wargoalValidateToken(value, fieldName string) (string, error) {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return "", fmt.Errorf("%s is required", fieldName)
	}
	for _, r := range cleaned {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f' ||
			r == '{' || r == '}' || r == '=' || r == '#' || r == '"' {
			return "", fmt.Errorf("%s must be one HOI4 token", fieldName)
		}
	}
	return cleaned, nil
}

func wargoalValidateScalar(value, fieldName string) (string, error) {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return "", nil
	}
	if strings.ContainsAny(cleaned, "\n\r{}=#") {
		return "", fmt.Errorf("%s must be one safe scalar", fieldName)
	}
	hasSpace := strings.ContainsAny(cleaned, " \t\v\f")
	if hasSpace && !(len(cleaned) >= 2 && strings.HasPrefix(cleaned, `"`) && strings.HasSuffix(cleaned, `"`)) {
		return "", fmt.Errorf("%s must be one token or one quoted string", fieldName)
	}
	return cleaned, nil
}

func wargoalValidateBlock(value, fieldName string) (string, error) {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return "", nil
	}
	block, recoveries, err := paradox.Parse(cleaned)
	if err != nil || len(recoveries) > 0 {
		return "", fmt.Errorf("%s contains malformed HOI4 script", fieldName)
	}
	for _, entry := range block.Entries {
		if entry.Key == nil {
			return "", fmt.Errorf("%s contains malformed HOI4 script", fieldName)
		}
	}
	return cleaned, nil
}

func wargoalValidateSafeScalars(value map[string]string) (map[string]string, error) {
	var unknown []string
	for key := range value {
		if !wargoalSafeScalarKeys[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("unsupported safe scalar keys: %s", strings.Join(unknown, ", "))
	}
	out := map[string]string{}
	for key, raw := range value {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		cleaned, err := wargoalValidateScalar(raw, key)
		if err != nil {
			return nil, err
		}
		out[key] = cleaned
	}
	return out, nil
}

type wargoal struct {
	ID                int64             `json:"id"`
	ProjectID         int64             `json:"project_id"`
	WargoalKey        string            `json:"wargoal_key"`
	WarName           string            `json:"war_name"`
	SafeScalars       map[string]string `json:"safe_scalars"`
	Allowed           string            `json:"allowed"`
	Available         string            `json:"available"`
	TakeStates        string            `json:"take_states"`
	Puppet            string            `json:"puppet"`
	Liberate          string            `json:"liberate"`
	AdvancedRawScript string            `json:"advanced_raw_script"`
	SourceFile        string            `json:"source_file"`
	Source            string            `json:"source"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
}

func wargoalEntries(row project.Row) []wargoalRecord {
	var records []wargoalRecord
	decode(row["ordered_entries"], &records)
	return records
}

func wargoalFromRow(row project.Row) wargoal {
	return wargoal{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		WargoalKey: text(row, "wargoal_key"), WarName: text(row, "war_name"),
		SafeScalars: stringMap(row["safe_scalars"]),
		Allowed:     text(row, "allowed"), Available: text(row, "available"),
		TakeStates: text(row, "take_states"), Puppet: text(row, "puppet"), Liberate: text(row, "liberate"),
		AdvancedRawScript: wargoalAdvancedScriptFromRecords(wargoalEntries(row)),
		SourceFile:        text(row, "source_file"), Source: textOr(row, "source", "created"),
		CreatedAt: isoNaive(row["created_at"]), UpdatedAt: isoNaive(row["updated_at"]),
	}
}

type wargoalCreate struct {
	WargoalKey        string            `json:"wargoal_key"`
	WarName           string            `json:"war_name"`
	SafeScalars       map[string]string `json:"safe_scalars"`
	Allowed           string            `json:"allowed"`
	Available         string            `json:"available"`
	TakeStates        string            `json:"take_states"`
	Puppet            string            `json:"puppet"`
	Liberate          string            `json:"liberate"`
	AdvancedRawScript string            `json:"advanced_raw_script"`
	SourceFile        string            `json:"source_file"`
}

func (in *wargoalCreate) normalize() error {
	var err error
	if in.WargoalKey, err = wargoalValidateToken(in.WargoalKey, "wargoal_key"); err != nil {
		return err
	}
	if in.WarName, err = wargoalValidateScalar(in.WarName, "war_name"); err != nil {
		return err
	}
	if in.SafeScalars == nil {
		in.SafeScalars = map[string]string{}
	}
	if in.SafeScalars, err = wargoalValidateSafeScalars(in.SafeScalars); err != nil {
		return err
	}
	blocks := []*string{&in.Allowed, &in.Available, &in.TakeStates, &in.Puppet, &in.Liberate}
	for i, name := range wargoalBlockFields {
		if *blocks[i], err = wargoalValidateBlock(*blocks[i], name); err != nil {
			return err
		}
	}
	in.SourceFile = strings.ReplaceAll(strings.TrimSpace(in.SourceFile), "\\", "/")
	return nil
}

type wargoalPatch struct {
	WargoalKey        workbench.Optional[string]            `json:"wargoal_key"`
	WarName           workbench.Optional[string]            `json:"war_name"`
	SafeScalars       workbench.Optional[map[string]string] `json:"safe_scalars"`
	Allowed           workbench.Optional[string]            `json:"allowed"`
	Available         workbench.Optional[string]            `json:"available"`
	TakeStates        workbench.Optional[string]            `json:"take_states"`
	Puppet            workbench.Optional[string]            `json:"puppet"`
	Liberate          workbench.Optional[string]            `json:"liberate"`
	AdvancedRawScript workbench.Optional[string]            `json:"advanced_raw_script"`
	SourceFile        workbench.Optional[string]            `json:"source_file"`
}

func (p *wargoalPatch) blocks() []*workbench.Optional[string] {
	return []*workbench.Optional[string]{&p.Allowed, &p.Available, &p.TakeStates, &p.Puppet, &p.Liberate}
}

func (p *wargoalPatch) normalize() error {
	for _, o := range append(p.blocks(), &p.WargoalKey, &p.WarName, &p.AdvancedRawScript, &p.SourceFile) {
		if o.Set && o.Value == nil {
			return fmt.Errorf("Wargoal update fields cannot be null")
		}
	}
	if p.SafeScalars.Set && p.SafeScalars.Value == nil {
		return fmt.Errorf("Wargoal update fields cannot be null")
	}
	if p.WargoalKey.Set {
		v, err := wargoalValidateToken(*p.WargoalKey.Value, "wargoal_key")
		if err != nil {
			return err
		}
		p.WargoalKey.Value = &v
	}
	if p.WarName.Set {
		v, err := wargoalValidateScalar(*p.WarName.Value, "war_name")
		if err != nil {
			return err
		}
		p.WarName.Value = &v
	}
	if p.SafeScalars.Set {
		v, err := wargoalValidateSafeScalars(*p.SafeScalars.Value)
		if err != nil {
			return err
		}
		p.SafeScalars.Value = &v
	}
	for i, block := range p.blocks() {
		if block.Set {
			v, err := wargoalValidateBlock(*block.Value, wargoalBlockFields[i])
			if err != nil {
				return err
			}
			block.Value = &v
		}
	}
	if p.SourceFile.Set {
		v := strings.ReplaceAll(strings.TrimSpace(*p.SourceFile.Value), "\\", "/")
		p.SourceFile.Value = &v
	}
	return nil
}

func wargoalConflict(key string) error {
	return statusErr(http.StatusConflict, fmt.Sprintf("Wargoal '%s' already exists. Pick a unique wargoal id.", key))
}

func (h *handlers) wargoals(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	const missing = "Wargoal not found"
	crud{
		listKey: "wargoals", missing: missing, unavailable: "wargoal store unavailable",
		list: func(p *project.Project, _ *http.Request) (any, int, error) {
			items, err := listRows(p, wargoalsTable, byText("wargoal_key"), wargoalFromRow)
			return items, len(items), err
		},
		create: createWargoal,
		get: func(p *project.Project, id int64) (any, error) {
			row, err := getRow(p, wargoalsTable, id, missing)
			if err != nil {
				return nil, err
			}
			return wargoalFromRow(row), nil
		},
		update: updateWargoal,
		remove: removeRow(wargoalsTable, missing),
	}.serve(p, w, r, rest)
}

func createWargoal(p *project.Project, r *http.Request) (any, error) {
	var in wargoalCreate
	if err := decodeBody(r, &in); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, "Invalid wargoal payload")
	}
	if err := in.normalize(); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	row, err := writeRow(p, wargoalsTable, func(tx *project.Tx) (project.Row, error) {
		if taken(tx.Rows(wargoalsTable), 0, func(row project.Row) bool { return text(row, "wargoal_key") == in.WargoalKey }) {
			return nil, wargoalConflict(in.WargoalKey)
		}
		advanced, err := parseWargoalAdvancedEntries(in.AdvancedRawScript)
		if err != nil {
			return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
		}
		return insert(tx, wargoalsTable, project.Row{
			"wargoal_key": in.WargoalKey, "war_name": in.WarName,
			"safe_scalars": jsonValue(in.SafeScalars),
			"allowed":      in.Allowed, "available": in.Available, "take_states": in.TakeStates,
			"puppet": in.Puppet, "liberate": in.Liberate,
			"ordered_entries":              jsonValue(advanced),
			"definition_leading_comments":  []any{},
			"definition_trailing_comments": []any{},
			"file_trivia":                  map[string]any{},
			"source_file":                  in.SourceFile, "source": "created", "source_snapshot": nil,
		})
	})
	if err != nil {
		return nil, err
	}
	return wargoalFromRow(row), nil
}

func updateWargoal(p *project.Project, r *http.Request, id int64) (any, error) {
	var patch wargoalPatch
	if err := decodeBody(r, &patch); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, "Invalid wargoal payload")
	}
	if err := patch.normalize(); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
	}
	cur, err := getRow(p, wargoalsTable, id, "Wargoal not found")
	if err != nil {
		return nil, err
	}
	keyClash := func(rows []project.Row) error {
		if patch.WargoalKey.Set && *patch.WargoalKey.Value != "" && *patch.WargoalKey.Value != text(cur, "wargoal_key") &&
			taken(rows, id, func(row project.Row) bool { return text(row, "wargoal_key") == *patch.WargoalKey.Value }) {
			return wargoalConflict(*patch.WargoalKey.Value)
		}
		return nil
	}
	rows, err := p.Rows(wargoalsTable)
	if err != nil {
		return nil, err
	}
	if err := keyClash(rows); err != nil {
		return nil, err
	}
	sets := project.Row{}
	if patch.AdvancedRawScript.Set {
		advanced, err := parseWargoalAdvancedEntries(*patch.AdvancedRawScript.Value)
		if err != nil {
			return nil, statusErr(http.StatusUnprocessableEntity, err.Error())
		}
		sets["ordered_entries"] = jsonValue(wargoalReplaceAdvancedRecords(wargoalEntries(cur), advanced))
	}
	setOpt(sets, "wargoal_key", patch.WargoalKey)
	setOpt(sets, "war_name", patch.WarName)
	setOptJSON(sets, "safe_scalars", patch.SafeScalars)
	for i, block := range patch.blocks() {
		setOpt(sets, wargoalBlockFields[i], *block)
	}
	setOpt(sets, "source_file", patch.SourceFile)
	if text(cur, "source") == "imported" {
		sets["source"] = "created"
	}
	row, err := updateRow(p, wargoalsTable, id, sets, func(tx *project.Tx) error {
		return keyClash(tx.Rows(wargoalsTable))
	})
	if err != nil {
		return nil, err
	}
	return wargoalFromRow(row), nil
}
