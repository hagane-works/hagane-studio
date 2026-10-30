package importjob

import (
	"encoding/json"
	"fmt"
	"haganeworks/internal/project"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	unitsparser "haganeworks/internal/hoi4/importsvc/units"
)

type UnitStageOptions struct {
	CountryTag  string
	ImportOOB   bool
	PreserveRaw bool
}

type UnitMergeOptions struct{ ConflictPolicy string }

type UnitCounts struct {
	RawFiles      int64 `json:"raw_files"`
	AirWingGroups int64 `json:"air_wing_groups"`
	AirWings      int64 `json:"air_wings"`
	Fleets        int64 `json:"fleets"`
	TaskForces    int64 `json:"task_forces"`
	Ships         int64 `json:"ships"`
}

func unitFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		n := strings.ToLower(filepath.ToSlash(rel))
		if strings.HasSuffix(n, ".txt") && (strings.HasPrefix(n, "common/units/") || strings.HasPrefix(n, "history/units/")) {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

func countsFromUnitPayload(c *UnitCounts, p map[string]any) {
	groups, _ := p["groups"].([]any)
	fleets, _ := p["fleets"].([]any)
	c.AirWingGroups += int64(len(groups))
	c.Fleets += int64(len(fleets))
	for _, v := range groups {
		if m, ok := v.(map[string]any); ok {
			if a, ok := m["wings"].([]any); ok {
				c.AirWings += int64(len(a))
			}
		}
	}
	for _, v := range fleets {
		if m, ok := v.(map[string]any); ok {
			if a, ok := m["task_forces"].([]any); ok {
				c.TaskForces += int64(len(a))
				for _, tf := range a {
					if tm, ok := tf.(map[string]any); ok {
						if s, ok := tm["ships"].([]any); ok {
							c.Ships += int64(len(s))
						}
					}
				}
			}
		}
	}
}

type unitFile struct {
	sourceFile, oobFile, countryTag, content string
	payload                                  payload
	mergeStructured                          bool
}

type unitStage struct{ files []unitFile }

func (s *session) stageUnits(countryTag string) (UnitCounts, error) {
	return s.stageStructuredUnits(UnitStageOptions{CountryTag: countryTag, ImportOOB: true, PreserveRaw: true})
}

func (s *session) mergeUnits(tx *project.Tx, conflictPolicy string) error {
	return s.mergeStructuredUnits(tx, UnitMergeOptions{ConflictPolicy: conflictPolicy})
}

func (s *session) stageStructuredUnits(options UnitStageOptions) (UnitCounts, error) {
	var counts UnitCounts
	files, err := unitFiles(s.root)
	if err != nil {
		return counts, err
	}

	type parsedUnitFile struct {
		file unitFile
		oob  map[string]any
	}
	err = inFileOrder(s.ctx, len(files), func(i int) (parsedUnitFile, error) {
		rel, _ := filepath.Rel(s.root, files[i])
		rel = filepath.ToSlash(rel)
		body, err := os.ReadFile(files[i])
		if err != nil {
			return parsedUnitFile{}, fmt.Errorf("read %s: %w", rel, err)
		}
		var parsed parsedUnitFile
		data := map[string]any{"groups": []any{}, "fleets": []any{}}
		if options.ImportOOB && strings.HasPrefix(strings.ToLower(rel), "history/units/") {
			wire := unitsparser.AirNavalOOBJSON(string(body))
			data["groups"] = wire["groups"]
			data["fleets"] = wire["fleets"]
			parsed.oob = data
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return parsedUnitFile{}, err
		}
		oob := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
		tag := strings.ToUpper(strings.TrimSpace(options.CountryTag))
		if tag == "" {
			tag = strings.ToUpper(strings.SplitN(oob, "_", 2)[0])
		}
		if len(tag) > 10 {
			tag = tag[:10]
		}
		content := ""
		if options.PreserveRaw {
			content = string(body)
		}
		parsed.file = unitFile{
			sourceFile: rel, oobFile: oob, countryTag: tag, content: content,
			payload: decodePayload(raw), mergeStructured: true,
		}
		return parsed, nil
	}, func(_ int, parsed parsedUnitFile) error {
		if parsed.oob != nil {
			countsFromUnitPayload(&counts, parsed.oob)
		}
		if options.PreserveRaw {
			counts.RawFiles++
		}
		s.units.files = append(s.units.files, parsed.file)
		return nil
	})
	if err != nil {
		return counts, err
	}

	s.units.files = lastWins(s.units.files, func(f unitFile) string { return f.sourceFile })
	return counts, nil
}

func (s *session) mergeStructuredUnits(tx *project.Tx, options UnitMergeOptions) error {
	policy := strings.ToLower(strings.TrimSpace(options.ConflictPolicy))
	if policy == "" {
		policy = "overwrite"
	}
	if policy != "overwrite" && policy != "skip" {
		return fmt.Errorf("unsupported unit conflict policy %q", options.ConflictPolicy)
	}
	if policy == "skip" {

		existing := map[string]bool{}
		for _, table := range []string{"oob_air_wing_groups", "oob_fleets"} {
			for _, row := range tx.Rows(table) {
				if file, ok := row["source_file"].(string); ok {
					existing[file] = true
				}
			}
		}
		for i := range s.units.files {
			if existing[s.units.files[i].sourceFile] {
				s.units.files[i].mergeStructured = false
			}
		}
	} else {

		staged := make(map[string]bool, len(s.units.files))
		for _, f := range s.units.files {
			staged[f.sourceFile] = true
		}
		groups := unitDeleteBySourceFile(tx, "oob_air_wing_groups", staged)
		unitDeleteByParent(tx, "oob_air_wings", "group_id", groups)
		fleets := unitDeleteBySourceFile(tx, "oob_fleets", staged)
		forces := unitDeleteByParent(tx, "oob_task_forces", "fleet_id", fleets)
		unitDeleteByParent(tx, "oob_ships", "task_force_id", forces)
	}

	for _, f := range s.units.files {
		if !f.mergeStructured {
			continue
		}
		for _, g := range unitElements(f.payload, "groups") {
			tx.Insert("oob_air_wing_groups", project.Row{
				"oob_file": f.oobFile, "source_file": f.sourceFile, "country_tag": f.countryTag,
				"location": g.str("location"), "ordering": g.intOr("ordering", 0),
				"layout": g.jsonOr("layout", []any{}), "source": "imported",
			})
		}
	}

	groupIDs := unitIDsBy(tx, "oob_air_wing_groups", "source_file")
	for _, f := range s.units.files {
		if !f.mergeStructured {
			continue
		}
		for _, g := range unitElements(f.payload, "groups") {
			for _, groupID := range groupIDs[unitJoinKey(f.sourceFile, g.intOr("ordering", 0))] {
				for _, w := range unitElements(g, "wings") {
					tx.Insert("oob_air_wings", project.Row{
						"group_id": groupID, "equipment_type": w.str("equipment_type"), "name": w.str("name"),
						"owner": w.str("owner"), "amount": w.str("amount"), "ordering": w.intOr("ordering", 0),
						"body_ast": w.jsonOr("body_ast", []any{}), "source": "imported",
					})
				}
			}
		}
	}

	for _, f := range s.units.files {
		if !f.mergeStructured {
			continue
		}
		for _, fleet := range unitElements(f.payload, "fleets") {
			tx.Insert("oob_fleets", project.Row{
				"oob_file": f.oobFile, "source_file": f.sourceFile, "country_tag": f.countryTag,
				"name": fleet.str("name"), "naval_base": fleet.str("naval_base"), "ordering": fleet.intOr("ordering", 0),
				"layout": fleet.jsonOr("layout", []any{}), "source": "imported",
			})
		}
	}

	fleetIDs := unitIDsBy(tx, "oob_fleets", "source_file")
	for _, f := range s.units.files {
		if !f.mergeStructured {
			continue
		}
		for _, fleet := range unitElements(f.payload, "fleets") {
			for _, fleetID := range fleetIDs[unitJoinKey(f.sourceFile, fleet.intOr("ordering", 0))] {
				for _, tf := range unitElements(fleet, "task_forces") {
					tx.Insert("oob_task_forces", project.Row{
						"fleet_id": fleetID, "name": tf.str("name"), "location": tf.str("location"),
						"ordering": tf.intOr("ordering", 0), "layout": tf.jsonOr("layout", []any{}), "source": "imported",
					})
				}
			}
		}
	}

	forceIDs := unitIDsBy(tx, "oob_task_forces", "fleet_id")
	for _, f := range s.units.files {
		if !f.mergeStructured {
			continue
		}
		for _, fleet := range unitElements(f.payload, "fleets") {
			for _, tf := range unitElements(fleet, "task_forces") {
				for _, fleetID := range fleetIDs[unitJoinKey(f.sourceFile, fleet.intOr("ordering", 0))] {
					for _, forceID := range forceIDs[unitJoinKey(fleetID, tf.intOr("ordering", 0))] {
						for _, ship := range unitElements(tf, "ships") {
							tx.Insert("oob_ships", project.Row{
								"task_force_id": forceID, "name": ship.str("name"), "definition": ship.str("definition"),
								"start_experience_factor": ship.str("start_experience_factor"),
								"equipment_type":          ship.str("equipment_type"), "amount": ship.str("amount"),
								"owner": ship.str("owner"), "version_name": ship.str("version_name"),
								"ordering": ship.intOr("ordering", 0), "body_ast": ship.jsonOr("body_ast", []any{}),
								"source": "imported",
							})
						}
					}
				}
			}
		}
	}

	when := miscImportedRawFile
	if policy == "skip" {
		when = nil
	}
	raw := newKeyedTable(tx, "project_raw_files", "dest_path")
	for _, f := range s.units.files {
		if f.content == "" {
			continue
		}
		miscUpsertRawFile(raw, project.Row{
			"dest_path": f.sourceFile, "content": f.content, "generate_replace_path": false,
			"note": "preserved on whole-mod import (units)", "is_binary": false, "source": "imported",
		}, []string{"content", "note", "is_binary", "source"}, when)
	}
	return nil
}

func unitElements(p payload, key string) []payload {
	list, _ := p[key].([]any)
	out := make([]payload, 0, len(list))
	for _, item := range list {
		object, _ := item.(map[string]any)
		out = append(out, payload(object))
	}
	return out
}

func unitIDsBy(tx *project.Tx, table, column string) map[string][]int64 {
	out := map[string][]int64{}
	for _, row := range tx.Rows(table) {
		key := unitJoinKey(row[column], project.Int64(row["ordering"]))
		out[key] = append(out[key], project.Int64(row["id"]))
	}
	return out
}

func unitJoinKey(value any, ordering int64) string {
	return keyPart(value) + "\x00" + strconv.FormatInt(ordering, 10)
}

func unitDeleteBySourceFile(tx *project.Tx, table string, files map[string]bool) map[int64]bool {
	ids := map[int64]bool{}
	tx.Delete(table, func(row project.Row) bool {
		file, _ := row["source_file"].(string)
		if files[file] {
			ids[project.Int64(row["id"])] = true
			return true
		}
		return false
	})
	return ids
}

func unitDeleteByParent(tx *project.Tx, table, parent string, parents map[int64]bool) map[int64]bool {
	ids := map[int64]bool{}
	if len(parents) == 0 {
		return ids
	}
	tx.Delete(table, func(row project.Row) bool {
		if parents[project.Int64(row[parent])] {
			ids[project.Int64(row["id"])] = true
			return true
		}
		return false
	})
	return ids
}
