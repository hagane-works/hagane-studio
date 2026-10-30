package states

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/project"
)

const maxStateSeedWarnings = 50

type stateSeedInput struct {
	StateIDs []int64
	All      bool
}

type stateSeedResult struct {
	Seeded   int64    `json:"seeded"`
	Skipped  int64    `json:"skipped"`
	Missing  []int64  `json:"missing"`
	Warnings []string `json:"warnings"`
}

var seedRawExtrasKeep = map[string]bool{"buildings": true, "force_link_ownership_to": true}

const seedRawExtrasKeepPrefix = "history."

func pydanticSeedBool(raw json.RawMessage) (bool, bool) {
	var value bool
	if json.Unmarshal(raw, &value) == nil {
		return value, true
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		switch strings.ToLower(strings.TrimSpace(text)) {
		case "1", "true", "t", "yes", "y", "on":
			return true, true
		case "0", "false", "f", "no", "n", "off":
			return false, true
		}
		return false, false
	}
	var number json.Number
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&number) == nil {
		switch number.String() {
		case "0", "0.0":
			return false, true
		case "1", "1.0":
			return true, true
		}
	}
	return false, false
}

func pydanticSeedInt(raw json.RawMessage) (int64, bool) {
	var value bool
	if json.Unmarshal(raw, &value) == nil {
		if value {
			return 1, true
		}
		return 0, true
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		parsed, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		return parsed, err == nil
	}
	var number json.Number
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&number) != nil {
		return 0, false
	}
	if parsed, err := number.Int64(); err == nil {
		return parsed, true
	}
	parsed, err := strconv.ParseFloat(number.String(), 64)
	if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) || math.Trunc(parsed) != parsed || parsed < math.MinInt64 || parsed > math.MaxInt64 {
		return 0, false
	}
	return int64(parsed), true
}

func decodeStateSeedInput(body []byte) (stateSeedInput, bool) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil || object == nil {
		return stateSeedInput{}, false
	}
	input := stateSeedInput{}
	if raw, exists := object["all"]; exists {
		value, ok := pydanticSeedBool(raw)
		if !ok {
			return stateSeedInput{}, false
		}
		input.All = value
	}
	if raw, exists := object["state_ids"]; exists && string(raw) != "null" {
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return stateSeedInput{}, false
		}
		input.StateIDs = make([]int64, 0, len(items))
		for _, item := range items {
			value, ok := pydanticSeedInt(item)
			if !ok {
				return stateSeedInput{}, false
			}
			input.StateIDs = append(input.StateIDs, value)
		}
	}
	return input, true
}

type seedWarnings struct {
	warnings []string
	overflow int
}

func (c *seedWarnings) add(message string) {
	if len(c.warnings) < maxStateSeedWarnings {
		c.warnings = append(c.warnings, message)
		return
	}
	c.overflow++
}

func (c *seedWarnings) result() []string {
	out := append([]string{}, c.warnings...)
	if c.overflow > 0 {
		out = append(out, fmt.Sprintf("... +%d more warnings (truncated)", c.overflow))
	}
	return out
}

func listOrEmpty(v any) []any {
	if list, ok := v.([]any); ok {
		return list
	}
	return []any{}
}

func mapOrEmpty(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func seedRow(state *vanillaStateFile) (project.Row, error) {
	src := state.Row
	provinces := listOrEmpty(src["provinces"])
	snapshotIDs, err := normalizeSeedProvinces(provinces)
	if err != nil {
		return nil, err
	}
	extras := map[string]any{}
	for k, v := range mapOrEmpty(src["raw_extras"]) {
		if seedRawExtrasKeep[k] || strings.HasPrefix(k, seedRawExtrasKeepPrefix) {
			extras[k] = v
		}
	}
	victoryPoints := []any{}
	for _, item := range listOrEmpty(src["victory_points"]) {
		vp := mapOrEmpty(item)
		victoryPoints = append(victoryPoints, map[string]any{"province_id": vp["province_id"], "value": vp["value"]})
	}
	controllers := []any{}
	for _, item := range listOrEmpty(src["set_province_controller"]) {
		pc := mapOrEmpty(item)
		controllers = append(controllers, map[string]any{"province_id": pc["province_id"], "controller": str(pc["controller"])})
	}
	controller := nullableText(src["controller"])
	owner := str(src["owner"])
	manpower, _ := whole(src["manpower"])
	row := project.Row{
		"state_id":                   state.StateID,
		"name":                       str(src["name"]),
		"display_name_locs":          map[string]any{},
		"manpower":                   manpower,
		"state_category":             str(src["state_category"]),
		"resources":                  plain(mapOrEmpty(src["resources"])),
		"provinces":                  plain(provinces),
		"owner":                      owner,
		"controller":                 controller,
		"add_core_of":                plain(listOrEmpty(src["add_core_of"])),
		"add_claim_by":               plain(listOrEmpty(src["add_claim_by"])),
		"remove_core_of":             plain(listOrEmpty(src["remove_core_of"])),
		"victory_points":             plain(victoryPoints),
		"set_province_controller":    plain(controllers),
		"buildings":                  plain(mapOrEmpty(src["buildings"])),
		"local_supplies":             src["local_supplies"],
		"buildings_max_level_factor": src["buildings_max_level_factor"],
		"impassable":                 src["impassable"],
		"impassable_ignored_links":   src["impassable_ignored_links"],
		"force_link_ownership_to":    nil,
		"raw_extras":                 plain(extras),
		"source_file":                state.Filename,
		"source_snapshot": plain(map[string]any{
			"schema_version": stateSnapshotSchema, "state_id": state.StateID,
			"province_ids": snapshotIDs, "owner": nullableText(owner), "controller": controller,
			"source_path": nullableText(state.Filename), "source_line": nil, "source_content_sha256": nil,
		}),
		"source": "imported",
	}
	return row, nil
}

func normalizeSeedProvinces(values []any) ([]int64, error) {
	seen := map[int64]bool{}
	for _, v := range values {
		n, ok := whole(v)
		if !ok || n <= 0 {
			return nil, errors.New("province ids must be positive integers")
		}
		seen[n] = true
	}
	out := make([]int64, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func seedStatesFromVanilla(p *project.Project, idx *vanillaStates, in stateSeedInput) (stateSeedResult, error) {
	byID := make(map[int64]*vanillaStateFile, len(idx.Catalog))
	for _, state := range idx.Catalog {
		byID[state.StateID] = state
	}
	var targets []*vanillaStateFile
	missing := []int64{}
	if in.All {
		targets = idx.Catalog
	} else {
		for _, id := range in.StateIDs {
			if state, ok := byID[id]; ok {
				targets = append(targets, state)
			} else {
				missing = append(missing, id)
			}
		}
	}
	var warnings seedWarnings
	seen := map[int64]bool{}
	var rows []project.Row
	var duplicates int64
	for _, target := range targets {
		if seen[target.StateID] {
			duplicates++
			continue
		}
		row, err := seedRow(target)
		if err != nil {
			warnings.add(fmt.Sprintf("State add failed id=%d from bundled: %v", target.StateID, err))
			continue
		}
		seen[target.StateID] = true
		rows = append(rows, row)
	}
	var seeded int64
	err := p.Write([]string{statesTable}, func(tx *project.Tx) error {
		existing := map[int64]bool{}
		for _, row := range tx.Rows(statesTable) {
			if id, ok := whole(row["state_id"]); ok {
				existing[id] = true
			}
		}
		for _, row := range rows {
			id := row["state_id"].(int64)
			if existing[id] {
				continue
			}
			existing[id] = true
			tx.Insert(statesTable, row)
			seeded++
		}
		return nil
	})
	if err != nil {
		return stateSeedResult{}, err
	}
	return stateSeedResult{
		Seeded:   seeded,
		Skipped:  duplicates + int64(len(rows)) - seeded,
		Missing:  missing,
		Warnings: warnings.result(),
	}, nil
}
