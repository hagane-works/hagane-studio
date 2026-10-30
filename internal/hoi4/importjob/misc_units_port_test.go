package importjob

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

func miscWriteMod(t *testing.T, files map[string][]byte) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, body, 0o640); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func miscNewProject(t *testing.T) *project.Project {
	t.Helper()
	p, err := project.NewStore(t.TempDir()).Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func miscWrite(t *testing.T, p *project.Project, change func(tx *project.Tx)) {
	t.Helper()
	if err := p.Write(schema.Tables(), func(tx *project.Tx) error { change(tx); return nil }); err != nil {
		t.Fatal(err)
	}
}

type miscPortRun struct {
	s          *session
	structured StructuredMiscResult
	units      UnitCounts
	misc       MiscCounts
	mapFiles   int
}

func miscPortImport(t *testing.T, p *project.Project, root, policy string) miscPortRun {
	t.Helper()
	options := AllOptions()
	options.ConflictPolicy = policy
	run := miscPortRun{s: newSession(context.Background(), root, options, p.Record())}
	s := run.s
	var err error
	if _, err = s.stageLocalisation(nil); err != nil {
		t.Fatal(err)
	}
	if run.structured, err = s.stageStructuredMisc(nil); err != nil {
		t.Fatal(err)
	}
	if run.units, err = s.stageUnits(options.CountryTag); err != nil {
		t.Fatal(err)
	}
	if run.misc, err = s.stageMiscEntities(nil); err != nil {
		t.Fatal(err)
	}
	if run.mapFiles, err = s.stageMapFiles(p.Blobs()); err != nil {
		t.Fatal(err)
	}
	err = p.Write(schema.Tables(), func(tx *project.Tx) error {
		if err := s.mergeStructuredMisc(tx, MiscMergeOptions{ConflictPolicy: policy}); err != nil {
			return err
		}
		if err := s.mergeUnits(tx, policy); err != nil {
			return err
		}
		return s.mergeMapFiles(tx, MiscMergeOptions{ConflictPolicy: policy})
	})
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func miscRowsBy(t *testing.T, p *project.Project, table, key string) map[string]project.Row {
	t.Helper()
	rows, err := p.Rows(table)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]project.Row{}
	for _, row := range rows {
		k, _ := row[key].(string)
		out[k] = row
	}
	return out
}

func miscJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func TestMiscPortStructuredAndRawFiles(t *testing.T) {
	p := miscNewProject(t)
	var seededModifier int64
	miscWrite(t, p, func(tx *project.Tx) {
		seededModifier = project.Int64(tx.Insert("modifier_definitions", project.Row{
			"modifier_id": "my_mod", "color_type": "bad", "value_type": "number", "precision": 2, "postfix": "none",
			"categories": []any{"all"}, "explicit_fields": []any{}, "source_file": "x.txt", "source": "created",
		})["id"])
		tx.Insert("project_raw_files", project.Row{
			"dest_path": "common/on_actions/o.txt", "content": "author", "generate_replace_path": false,
			"note": "mine", "is_binary": false, "source": "created",
		})
	})
	bad := "broken_mod = { nested = { x = 1 } }\n"
	mod := map[string][]byte{
		"common/modifier_definitions/a.txt":   []byte("my_mod = {\n color_type = good\n value_type = percentage\n precision = 1\n category = army\n}\n"),
		"common/modifier_definitions/bad.txt": []byte(bad),
		"common/ai_strategy_plans/p.txt":      []byte("TST_plan = {\n name = \"Plan\"\n desc = \"Plan desc\"\n allowed = { original_tag = TST }\n enable = { always = yes }\n}\n"),
		"common/ai_templates/t.txt":           []byte("TST_template = {\n role = armor\n tank_target = { upgrade_prio = { factor = 1 } }\n}\n"),
		"common/wargoals/w.txt":               []byte("wargoal_types = {\n take_tst = {\n  war_name = WAR_NAME_TST\n  expire = 0\n  allowed = { always = yes }\n }\n}\n"),
		"common/raids/r.txt":                  []byte("raid_cat = { }\n"),
		"common/on_actions/o.txt":             []byte("on_actions = { on_startup = { } }\n"),
	}
	run := miscPortImport(t, p, miscWriteMod(t, mod), "overwrite")

	if c := run.structured.Counts; c["modifier_definitions"] != 1 || c["ai_strategy_plans"] != 1 || c["ai_templates"] != 1 || c["wargoals"] != 1 {
		t.Fatalf("structured counts %v", c)
	}
	if len(run.structured.Warnings) != 1 || !strings.HasPrefix(run.structured.Warnings[0], "common/modifier_definitions/bad.txt kept as raw file: ") {
		t.Fatalf("warnings %v", run.structured.Warnings)
	}
	if run.misc["raids"] != 1 || run.misc["on_actions"] != 1 {
		t.Fatalf("misc counts %v", run.misc)
	}

	mods := miscRowsBy(t, p, "modifier_definitions", "modifier_id")
	m := mods["my_mod"]
	if project.Int64(m["id"]) != seededModifier || m["color_type"] != "good" || m["value_type"] != "percentage" ||
		project.Int64(m["precision"]) != 1 || m["postfix"] != "none" || miscJSON(m["categories"]) != `["army"]` ||
		miscJSON(m["explicit_fields"]) != `["color_type","value_type","precision","category"]` ||
		m["source"] != "imported" || m["source_file"] != "common/modifier_definitions/a.txt" {
		t.Fatalf("existing modifier definition is updated in place: %v", m)
	}
	if _, ok := mods["broken_mod"]; ok {
		t.Fatal("a rejected file's definitions are not structured")
	}
	plan := miscRowsBy(t, p, "ai_strategy_plans", "plan_id")["TST_plan"]
	if plan["name"] != `"Plan"` || plan["desc"] != `"Plan desc"` || len(plan["allowed"].([]any)) != 1 || len(plan["enable"].([]any)) != 1 ||
		miscJSON(plan["abort"]) != `[]` || miscJSON(plan["field_order"]) != `["name","desc","allowed","enable"]` ||
		!strings.Contains(plan["raw_block"].(string), "TST_plan") || plan["source"] != "imported" || plan["source_file"] != "common/ai_strategy_plans/p.txt" {
		t.Fatalf("ai plan %v", plan)
	}
	template := miscRowsBy(t, p, "ai_templates", "template_id")["TST_template"]
	if template["role"] != "armor" || len(template["target_templates"].([]any)) != 1 ||
		miscJSON(template["field_order"]) != `["role","target:tank_target"]` || template["blocked_for"] != "" || template["source"] != "imported" {
		t.Fatalf("ai template %v", template)
	}
	wargoal := miscRowsBy(t, p, "wargoal_types", "wargoal_key")["take_tst"]
	if wargoal["war_name"] != "WAR_NAME_TST" || miscJSON(wargoal["safe_scalars"]) != `{"expire":"0"}` ||
		!strings.Contains(wargoal["allowed"].(string), "always") || wargoal["puppet"] != "" || len(wargoal["ordered_entries"].([]any)) != 3 ||
		miscJSON(wargoal["file_trivia"]) != `{}` || wargoal["source_snapshot"] != nil || wargoal["source"] != "imported" {
		t.Fatalf("wargoal %v", wargoal)
	}

	raw := miscRowsBy(t, p, "project_raw_files", "dest_path")
	if r := raw["common/modifier_definitions/bad.txt"]; r["content"] != bad || r["note"] != "preserved by Go whole-mod import (modifier_definitions)" ||
		r["is_binary"] != false || r["content_binary"] != nil || r["generate_replace_path"] != false ||
		r["raw_kind"] != "verbatim_override" || r["source"] != "imported" {
		t.Fatalf("rejected structured file kept raw: %v", r)
	}
	if r := raw["common/raids/r.txt"]; r["content"] != "raid_cat = { }\n" || r["note"] != "preserved by Go whole-mod import (raids)" || r["source"] != "imported" {
		t.Fatalf("raids raw file %v", r)
	}
	if r := raw["common/on_actions/o.txt"]; r["content"] != "author" || r["source"] != "created" || r["note"] != "mine" {
		t.Fatalf("the author's raw override wins: %v", r)
	}

	mod["common/modifier_definitions/a.txt"] = []byte("my_mod = { color_type = neutral }\nnew_mod = { color_type = good }\n")
	mod["common/raids/r.txt"] = []byte("raid_cat = { changed = yes }\n")
	miscPortImport(t, p, miscWriteMod(t, mod), "skip")
	mods = miscRowsBy(t, p, "modifier_definitions", "modifier_id")
	if mods["my_mod"]["color_type"] != "good" {
		t.Fatalf("skip keeps the existing definition: %v", mods["my_mod"])
	}
	if n := mods["new_mod"]; n == nil || n["color_type"] != "good" || n["value_type"] != "number" || project.Int64(n["precision"]) != 2 ||
		miscJSON(n["categories"]) != `["all"]` || n["source"] != "imported" {
		t.Fatalf("skip still adds a new definition: %v", n)
	}
	if r := miscRowsBy(t, p, "project_raw_files", "dest_path")["common/raids/r.txt"]; r["content"] != "raid_cat = { }\n" {
		t.Fatalf("skip keeps the imported raw file: %v", r)
	}

	miscPortImport(t, p, miscWriteMod(t, mod), "overwrite")
	mods = miscRowsBy(t, p, "modifier_definitions", "modifier_id")
	if m := mods["my_mod"]; m["color_type"] != "neutral" || miscJSON(m["categories"]) != `["all"]` || project.Int64(m["id"]) != seededModifier {
		t.Fatalf("overwrite updates the definition: %v", m)
	}
	raw = miscRowsBy(t, p, "project_raw_files", "dest_path")
	if r := raw["common/raids/r.txt"]; r["content"] != "raid_cat = { changed = yes }\n" {
		t.Fatalf("overwrite refreshes the imported raw file: %v", r)
	}
	if r := raw["common/on_actions/o.txt"]; r["content"] != "author" || r["source"] != "created" {
		t.Fatalf("the author's raw override still wins: %v", r)
	}
	if n := len(miscRowsBy(t, p, "modifier_definitions", "modifier_id")); n != 2 {
		t.Fatalf("modifier definitions %d, want 2", n)
	}
}

const unitPortOOB = `air_wings = { 1 = { fighter = { owner = TST amount = 10 } name = "A" } }
units = { fleet = { name = "F" naval_base = 1 task_force = { name = "T" location = 1 ship = { name = "S" definition = destroyer } } } }
`

func TestUnitsPortHierarchyAndReimport(t *testing.T) {
	p := miscNewProject(t)
	mod := map[string][]byte{
		"history/units/TST_1936_naval.txt": []byte(unitPortOOB),
		"common/units/infantry.txt":        []byte("sub_units = { infantry = { } }\n"),
		"history/units/TST_empty.txt":      {},
	}
	run := miscPortImport(t, p, miscWriteMod(t, mod), "overwrite")
	if c := run.units; c.RawFiles != 3 || c.AirWingGroups != 1 || c.AirWings != 1 || c.Fleets != 1 || c.TaskForces != 1 || c.Ships != 1 {
		t.Fatalf("unit counts %#v", c)
	}
	const naval = "history/units/TST_1936_naval.txt"
	check := func(ships ...string) {
		t.Helper()
		groups, _ := p.Rows("oob_air_wing_groups")
		wings, _ := p.Rows("oob_air_wings")
		fleets, _ := p.Rows("oob_fleets")
		forces, _ := p.Rows("oob_task_forces")
		shipRows, _ := p.Rows("oob_ships")
		if len(groups) != 1 || len(wings) != 1 || len(fleets) != 1 || len(forces) != 1 || len(shipRows) != len(ships) {
			t.Fatalf("rows: %d groups, %d wings, %d fleets, %d task forces, %d ships", len(groups), len(wings), len(fleets), len(forces), len(shipRows))
		}
		g, w, f, tf := groups[0], wings[0], fleets[0], forces[0]
		if g["oob_file"] != "TST_1936_naval" || g["source_file"] != naval || g["country_tag"] != "TST" || g["location"] != "1" ||
			project.Int64(g["ordering"]) != 0 || len(g["layout"].([]any)) == 0 || g["source"] != "imported" {
			t.Fatalf("air wing group %v", g)
		}
		if project.Int64(w["group_id"]) != project.Int64(g["id"]) || w["equipment_type"] != "fighter" || w["name"] != "A" ||
			w["owner"] != "TST" || w["amount"] != "10" || len(w["body_ast"].([]any)) == 0 || w["source"] != "imported" {
			t.Fatalf("air wing %v", w)
		}
		if f["oob_file"] != "TST_1936_naval" || f["source_file"] != naval || f["country_tag"] != "TST" || f["name"] != "F" ||
			f["naval_base"] != "1" || f["source"] != "imported" {
			t.Fatalf("fleet %v", f)
		}
		if project.Int64(tf["fleet_id"]) != project.Int64(f["id"]) || tf["name"] != "T" || tf["location"] != "1" || tf["source"] != "imported" {
			t.Fatalf("task force %v", tf)
		}
		for i, ship := range shipRows {
			if project.Int64(ship["task_force_id"]) != project.Int64(tf["id"]) || ship["name"] != ships[i] ||
				ship["definition"] != "destroyer" || project.Int64(ship["ordering"]) != int64(i) || ship["source"] != "imported" {
				t.Fatalf("ship %d %v", i, ship)
			}
		}
	}
	check("S")
	raw := miscRowsBy(t, p, "project_raw_files", "dest_path")
	if r := raw[naval]; r["content"] != unitPortOOB || r["note"] != "preserved on whole-mod import (units)" ||
		r["is_binary"] != false || r["generate_replace_path"] != false || r["raw_kind"] != "verbatim_override" || r["source"] != "imported" {
		t.Fatalf("naval raw file %v", r)
	}
	if _, ok := raw["common/units/infantry.txt"]; !ok {
		t.Fatal("common/units file kept raw")
	}
	if _, ok := raw["history/units/TST_empty.txt"]; ok {
		t.Fatal("an empty file has no raw copy (content<>'')")
	}
	firstGroup := project.Int64(miscRowsBy(t, p, "oob_air_wing_groups", "source_file")[naval]["id"])

	miscWrite(t, p, func(tx *project.Tx) {
		for _, row := range tx.Rows("project_raw_files") {
			if row["dest_path"] == "common/units/infantry.txt" {
				tx.Update("project_raw_files", project.Int64(row["id"]), func(r project.Row) { r["source"], r["content"] = "created", "author" })
			}
		}
	})

	twoShips := strings.Replace(unitPortOOB, `definition = destroyer }`, `definition = destroyer } ship = { name = "S2" definition = destroyer }`, 1)
	mod["history/units/TST_1936_naval.txt"] = []byte(twoShips)
	mod["common/units/infantry.txt"] = []byte("sub_units = { changed = { } }\n")
	miscPortImport(t, p, miscWriteMod(t, mod), "overwrite")
	check("S", "S2")
	if id := project.Int64(miscRowsBy(t, p, "oob_air_wing_groups", "source_file")[naval]["id"]); id == firstGroup {
		t.Fatal("overwrite replaces the file's air wing group")
	}
	raw = miscRowsBy(t, p, "project_raw_files", "dest_path")
	if raw[naval]["content"] != twoShips {
		t.Fatalf("overwrite refreshes the imported raw copy: %v", raw[naval])
	}
	if r := raw["common/units/infantry.txt"]; r["content"] != "author" || r["source"] != "created" {
		t.Fatalf("the author's raw file wins: %v", r)
	}

	mod["history/units/TST_1936_naval.txt"] = []byte(unitPortOOB)
	mod["history/units/TST_new.txt"] = []byte(`units = { fleet = { name = "N" } }`)
	miscPortImport(t, p, miscWriteMod(t, mod), "skip")
	fleets := miscRowsBy(t, p, "oob_fleets", "source_file")
	if len(fleets) != 2 || fleets["history/units/TST_new.txt"]["name"] != "N" || fleets["history/units/TST_new.txt"]["oob_file"] != "TST_new" {
		t.Fatalf("skip merges only the new file: %v", fleets)
	}
	if ships, _ := p.Rows("oob_ships"); len(ships) != 2 {
		t.Fatalf("skip keeps the existing file's ships: %d", len(ships))
	}
	if raw := miscRowsBy(t, p, "project_raw_files", "dest_path"); raw[naval]["content"] != twoShips || raw["history/units/TST_new.txt"] == nil {
		t.Fatalf("skip keeps existing raw copies and adds new ones: %v", raw[naval])
	}

	s := newSession(context.Background(), t.TempDir(), AllOptions(), p.Record())
	if err := p.Write(schema.Tables(), func(tx *project.Tx) error { return s.mergeUnits(tx, "merge") }); err == nil || !strings.Contains(err.Error(), "unsupported unit conflict policy") {
		t.Fatalf("unknown policy: %v", err)
	}
}

func TestMapFilesPortIsModMapFile(t *testing.T) {
	for rel, want := range map[string]bool{
		"map/provinces.bmp":                               true,
		"map/definition.csv":                              true,
		"map/default.map":                                 true,
		"map/strategicregions/1-A.txt":                    true,
		"MAP/Provinces.BMP":                               true,
		`map\buildings.txt`:                               true,
		"map/terrain/colormap_rgb_cityemissivemask_a.dds": true,
		"map/terrain/atlas0.dds":                          false,
		"map/water/colormap_water_0.dds":                  false,
		"common/national_focus/a.txt":                     false,
		"gfx/map/x.bmp":                                   false,
	} {
		if got := IsModMapFile(rel); got != want {
			t.Errorf("IsModMapFile(%q) = %v, want %v", rel, got, want)
		}
	}
	if got, _ := UnroutedByDir(miscWriteMod(t, map[string][]byte{
		"map/provinces.bmp":      {1},
		"map/terrain/atlas0.dds": {1},
	})); got["map"] != 0 || got["map/terrain"] != 1 {
		t.Fatalf("the kept map files are not reported, the rest of map/ is: %v", got)
	}
}

type mapFileRecordingStore struct{ puts map[string][]byte }

func (s *mapFileRecordingStore) EnvPrefix() string { return "env" }
func (s *mapFileRecordingStore) Put(key, _ string, _ int64, body io.Reader) error {
	data, err := io.ReadAll(body)
	s.puts[key] = data
	return err
}
func (s *mapFileRecordingStore) Head(key string) (bool, error) { _, ok := s.puts[key]; return ok, nil }

func TestMapFilesPortPictureStoredByContent(t *testing.T) {
	picture := []byte("BM fake")
	root := miscWriteMod(t, map[string][]byte{"map/provinces.bmp": picture})
	store := &mapFileRecordingStore{puts: map[string][]byte{}}
	row, problem, err := mapFileRow(filepath.Join(root, "map", "provinces.bmp"), "map/provinces.bmp", nil, store)
	if err != nil || problem != "" {
		t.Fatalf("%v %q", err, problem)
	}
	sum := sha256.Sum256(picture)
	sha := hex.EncodeToString(sum[:])
	want := "env/rawblobs/" + sha[:2] + "/" + sha
	if row[4] != want || string(store.puts[want]) != string(picture) {
		t.Fatalf("row %v, puts %v; want the bytes at %s", row, store.puts, want)
	}
	if _, problem, _ := mapFileRow(filepath.Join(root, "map", "provinces.bmp"), "map/provinces.bmp", nil, nil); !strings.Contains(problem, "no object storage") {
		t.Fatalf("without storage the picture is reported: %q", problem)
	}
}

func TestMapFilesPortKeepsTheModsOwnMap(t *testing.T) {
	p := miscNewProject(t)
	miscWrite(t, p, func(tx *project.Tx) {
		raw := func(path, content, source, note string) {
			tx.Insert("project_raw_files", project.Row{
				"dest_path": path, "content": content, "generate_replace_path": false, "note": note, "is_binary": false, "source": source,
			})
		}
		raw("map/heightmap.bmp", "edited", "created", "a map edit")
		raw("map/gone.txt", "old", "imported", mapFilesNote)
		raw("map/other_lane.txt", "kept", "imported", "another importer")
	})
	picture := []byte("BM fake provinces")
	latin1 := []byte{'n', 0xe9, '\n'}
	definition := "0;0;0;0;land;false;unknown;0\n1;10;20;30;land;false;plains;1\n"
	mod := map[string][]byte{
		"map/provinces.bmp":            picture,
		"map/heightmap.bmp":            []byte("BM mod heightmap"),
		"map/definition.csv":           []byte(definition),
		"map/strategicregions/1-A.txt": []byte("strategic_region={ id=1 }\n"),
		"map/latin1.txt":               latin1,
		"map/terrain/atlas0.dds":       []byte("DDS texture"),
	}
	run := miscPortImport(t, p, miscWriteMod(t, mod), "overwrite")
	if run.mapFiles != 5 {
		t.Fatalf("kept %d map files, want 5", run.mapFiles)
	}
	for _, warning := range run.s.warnings {
		if strings.HasPrefix(warning, "Map file ") {
			t.Fatalf("unexpected warning %q", warning)
		}
	}
	blobKey := func(body []byte) (string, string) {
		sum := sha256.Sum256(body)
		sha := hex.EncodeToString(sum[:])
		return sha, project.BlobEnvPrefix + "/rawblobs/" + sha[:2] + "/" + sha
	}
	blobs := p.Blobs()
	rows := miscRowsBy(t, p, "project_raw_files", "dest_path")
	if r := rows["map/definition.csv"]; r["content"] != definition || r["is_binary"] != false || r["source"] != "imported" ||
		r["note"] != mapFilesNote || r["raw_kind"] != "verbatim_override" || r["generate_replace_path"] != false ||
		r["content_sha256"] != nil || r["content_obs_key"] != nil || r["content_binary"] != nil {
		t.Fatalf("definition.csv kept as text: %v", r)
	}
	for path, body := range map[string][]byte{"map/provinces.bmp": picture, "map/latin1.txt": latin1} {
		sha, key := blobKey(body)
		r := rows[path]
		if r["is_binary"] != true || r["content"] != "" || r["content_sha256"] != sha || r["content_obs_key"] != key || r["source"] != "imported" || r["note"] != mapFilesNote {
			t.Fatalf("%s points at its stored bytes: %v", path, r)
		}
		if stored, err := blobs.Get(key, 1<<20); err != nil || string(stored) != string(body) {
			t.Fatalf("%s bytes at %s: %q, %v", path, key, stored, err)
		}
	}
	if r := rows["map/heightmap.bmp"]; r["source"] != "created" || r["content"] != "edited" || r["note"] != "a map edit" || r["is_binary"] != false {
		t.Fatalf("the author's map edit is not overwritten by a re-import: %v", r)
	}
	if _, ok := rows["map/gone.txt"]; ok {
		t.Fatal("an earlier import's map file the mod no longer ships is removed")
	}
	if _, ok := rows["map/other_lane.txt"]; !ok {
		t.Fatal("another importer's row is not this lane's to remove")
	}
	if _, ok := rows["map/terrain/atlas0.dds"]; ok {
		t.Fatal("render-only textures are not kept")
	}
	countMapRows := func() int {
		n := 0
		for _, r := range miscRowsBy(t, p, "project_raw_files", "dest_path") {
			if r["note"] == mapFilesNote {
				n++
			}
		}
		return n
	}

	miscPortImport(t, p, miscWriteMod(t, map[string][]byte{"common/ideas/a.txt": []byte("ideas={}\n")}), "overwrite")
	if n := countMapRows(); n != 4 {
		t.Fatalf("map rows after an import without a map: %d, want 4", n)
	}

	delete(mod, "map/strategicregions/1-A.txt")
	mod["map/definition.csv"] = []byte("changed\n")
	miscPortImport(t, p, miscWriteMod(t, mod), "skip")
	rows = miscRowsBy(t, p, "project_raw_files", "dest_path")
	if rows["map/definition.csv"]["content"] != definition || rows["map/strategicregions/1-A.txt"] == nil {
		t.Fatalf("skip keeps the map rows: %v", rows["map/definition.csv"])
	}

	delete(mod, "map/latin1.txt")
	miscPortImport(t, p, miscWriteMod(t, mod), "overwrite")
	rows = miscRowsBy(t, p, "project_raw_files", "dest_path")
	if rows["map/definition.csv"]["content"] != "changed\n" {
		t.Fatalf("overwrite refreshes the text: %v", rows["map/definition.csv"])
	}
	for _, gone := range []string{"map/strategicregions/1-A.txt", "map/latin1.txt"} {
		if _, ok := rows[gone]; ok {
			t.Fatalf("%s is no longer shipped and is removed", gone)
		}
	}
	if rows["map/heightmap.bmp"]["content"] != "edited" || rows["map/provinces.bmp"] == nil || countMapRows() != 2 {
		t.Fatalf("after overwrite: %d map rows", countMapRows())
	}

	s := newSession(context.Background(), miscWriteMod(t, map[string][]byte{"map/provinces.bmp": picture, "map/default.map": []byte("x\n")}), AllOptions(), p.Record())
	kept, err := s.stageMapFiles(nil)
	if err != nil || kept != 1 || len(s.warnings) != 1 || s.warnings[0] != "Map file map/provinces.bmp was not kept: there is no object storage for its bytes" {
		t.Fatalf("kept %d, %v, warnings %v", kept, err, s.warnings)
	}
}
