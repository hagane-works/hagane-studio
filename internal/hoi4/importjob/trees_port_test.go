package importjob

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

const treePortFocusFile = `focus_tree = {
	id = TST_focus
	country = { factor = 0 modifier = { add = 10 tag = TST } }
	shared_focus = TST_shared_root
	focus = { id = TST_a icon = GFX_goal_tst x = 1 y = 0 cost = 5 }
	focus = { id = TST_b prerequisite = { focus = TST_a } x = 1 y = 1 }
}
focus_tree = {
	id = GER_focus
	country = { factor = 0 modifier = { add = 10 tag = GER } }
	focus = { id = GER_a x = 0 y = 0 }
}
`

const treePortTechFile = `@1936 = 0
@1940 = 2
technologies = {
	TST_rifle = {
		research_cost = 2
		start_year = 1937
		folder = { name = infantry_folder position = { x = 1 y = @1936 } }
		path = { leads_to_tech = TST_rifle2 research_cost_coeff = 1 }
	}
	TST_rifle2 = {
		research_cost = 1.5
		folder = { name = infantry_folder position = { x = 1 y = @1940 } }
	}
}
`

func treePortWrite(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, text := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func treePortMod(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	treePortWrite(t, root, map[string]string{
		"common/national_focus/tst.txt": treePortFocusFile,
		"common/national_focus/TST_shared.txt": "shared_focus = { id = TST_shared_root x = 0 y = 0 }\n" +
			"shared_focus = { id = TST_shared_child prerequisite = { focus = TST_shared_root } x = 0 y = 1 }\n",
		"common/national_focus/orphan_pool.txt":            "shared_focus = { id = ORPHAN_x x = 0 y = 0 }\n",
		"common/national_focus/renamed.txt":                "focus_tree = { id = finnish_focus focus = { id = FIN_x x = 0 y = 0 } }\n",
		"common/technologies/infantry.txt":                 treePortTechFile,
		"common/technologies/special_projects_tech.txt":    "technologies = { SP_ignored = { research_cost = 1 } }\n",
		"localisation/english/tst_l_english.yml":           "l_english:\n TST_a:0 \"Alpha\"\n TST_a_desc:0 \"Alpha desc\"\n TST_rifle:0 \"Rifle\"\n",
		"localisation/simp_chinese/tst_l_simp_chinese.yml": "l_simp_chinese:\n TST_a:0 \"阿尔法\"\n",
		"interface/tst.gfx": "spriteTypes = {\n" +
			"\tspriteType = { name = \"GFX_goal_tst\" texturefile = \"gfx/interface/goals/tst.dds\" }\n" +
			"\tspriteType = { name = \"GFX_TST_rifle_medium\" texturefile = \"gfx/interface/technologies/rifle.dds\" }\n}\n",
	})
	return root
}

func treePortProject(t *testing.T, record project.Row) *project.Project {
	t.Helper()
	p, err := project.NewStore(t.TempDir()).Create(record, "t")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func treePortDefaultProject(t *testing.T) *project.Project {
	return treePortProject(t, project.Row{
		"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"},
		"tech_year_columns": []any{[]any{"@1950", 9}, []any{"@1936", 5}},
	})
}

func treePortRun(p *project.Project, root string, stage TreeStageOptions, policy string) (*session, TreeEntityCounts, error) {
	s := newSession(context.Background(), root, AllOptions(), p.Record())
	if _, err := s.stageLocalisation(nil); err != nil {
		return nil, TreeEntityCounts{}, err
	}
	if _, err := s.stageGFX(); err != nil {
		return nil, TreeEntityCounts{}, err
	}
	stage.ProjectID = p.ID()
	counts, err := s.stageTrees(stage, nil)
	if err != nil {
		return nil, counts, err
	}
	err = p.Write(schema.Tables(), func(tx *project.Tx) error {
		return s.mergeTrees(tx, TreeMergeOptions{ConflictPolicy: policy})
	})
	return s, counts, err
}

func treePortImport(t *testing.T, p *project.Project, root, policy string) (*session, TreeEntityCounts) {
	t.Helper()
	s, counts, err := treePortRun(p, root, TreeStageOptions{ImportFocus: true, ImportTech: true}, policy)
	if err != nil {
		t.Fatal(err)
	}
	return s, counts
}

func treePortRows(t *testing.T, p *project.Project, table, key string) map[string]project.Row {
	t.Helper()
	rows, err := p.Rows(table)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]project.Row{}
	for _, row := range rows {
		k, _ := row[key].(string)
		if _, dup := out[k]; dup {
			t.Fatalf("%s: two rows with %s=%q", table, key, k)
		}
		out[k] = row
	}
	return out
}

func treePortKeys(rows map[string]project.Row) []string {
	keys := make([]string, 0, len(rows))
	for key := range rows {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func treePortJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func treePortUpdate(t *testing.T, p *project.Project, table string, id int64, change func(project.Row)) {
	t.Helper()
	if err := p.Write([]string{table}, func(tx *project.Tx) error {
		_, err := tx.Update(table, id, change)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func TestTreesPortImportsFocusTechnologyAndPolicies(t *testing.T) {
	p := treePortDefaultProject(t)

	if err := p.Write([]string{"focus_file_policies"}, func(tx *project.Tx) error {
		tx.Insert("focus_file_policies", project.Row{"source_file": "common/national_focus/old_tst.txt", "tree_ids": []any{"TST_focus"}})
		tx.Insert("focus_file_policies", project.Row{"source_file": "common/national_focus/other.txt", "tree_ids": []any{"OTHER_tree"}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	s, counts := treePortImport(t, p, treePortMod(t), "overwrite")

	if counts != (TreeEntityCounts{FocusTrees: 4, FocusNodes: 6, Technologies: 2, YearColumns: 3}) {
		t.Fatalf("counts %+v", counts)
	}

	trees := treePortRows(t, p, "focus_trees", "tree_id")

	if got := treePortKeys(trees); !reflect.DeepEqual(got, []string{"TST_focus", "TST_shared", "finnish_focus"}) {
		t.Fatalf("trees %v", got)
	}
	tree := trees["TST_focus"]
	if tree["country_tag"] != "TST" || tree["scope_kind"] != "own" || tree["tree_kind"] != "primary" || tree["source"] != "imported" ||
		tree["source_file"] != "common/national_focus/tst.txt" || project.Int64(tree["edit_revision"]) != 1 {
		t.Fatalf("TST_focus %v", tree)
	}
	if got := treePortJSON(tree["shared_focus_refs"]); !strings.Contains(got, `"focus_id":"TST_shared_root"`) {
		t.Fatalf("shared_focus_refs %s", got)
	}
	if pool := trees["TST_shared"]; pool["tree_kind"] != "shared_pool" || pool["country_tag"] != "TST" {
		t.Fatalf("TST_shared %v", pool)
	}
	if global := trees["finnish_focus"]; global["country_tag"] != nil || global["scope_kind"] != "global" {
		t.Fatalf("finnish_focus %v", global)
	}

	nodes := treePortRows(t, p, "focus_nodes", "focus_id")
	if got := treePortKeys(nodes); !reflect.DeepEqual(got, []string{"FIN_x", "TST_a", "TST_b", "TST_shared_child", "TST_shared_root"}) {
		t.Fatalf("nodes %v", got)
	}
	a := nodes["TST_a"]
	if project.Int64(a["focus_tree_id"]) != project.Int64(tree["id"]) || a["name_en"] != "Alpha" || a["name_zh"] != "阿尔法" ||
		a["desc_en"] != "Alpha desc" || a["desc_zh"] != "" || a["icon"] != "GFX_goal_tst" || treePortJSON(a["cost"]) != "5" ||
		a["source"] != "imported" || a["cancel_if_invalid"] != true {
		t.Fatalf("TST_a %v", a)
	}
	if got := treePortJSON(a["display_name_locs"]); got != `{"en":"Alpha","zh":"阿尔法"}` {
		t.Fatalf("display_name_locs %s", got)
	}
	if got := treePortJSON(a["description_locs"]); got != `{"en":"Alpha desc"}` {
		t.Fatalf("description_locs %s", got)
	}
	if got := treePortJSON(a["offset"]); got != "[]" {
		t.Fatalf("offset %s", got)
	}
	snapshot, _ := a["source_snapshot"].(map[string]any)
	if snapshot["name_en"] != "Alpha" || snapshot["desc_en"] != "Alpha desc" || snapshot["focus_id"] != "TST_a" {
		t.Fatalf("source_snapshot %v", snapshot)
	}
	b := nodes["TST_b"]
	if b["name_en"] != "TST_b" || b["name"] != "TST_b" || treePortJSON(b["prerequisites"]) != `[["TST_a"]]` || treePortJSON(b["description_locs"]) != "{}" {
		t.Fatalf("TST_b %v", b)
	}
	if project.Int64(nodes["TST_shared_root"]["focus_tree_id"]) != project.Int64(trees["TST_shared"]["id"]) {
		t.Fatal("shared pool node not linked to its pool tree")
	}

	techs := treePortRows(t, p, "technologies", "tech_id")
	if got := treePortKeys(techs); !reflect.DeepEqual(got, []string{"TST_rifle", "TST_rifle2"}) {
		t.Fatalf("technologies %v", got)
	}
	rifle := techs["TST_rifle"]
	if rifle["name"] != "Rifle" || rifle["name_en"] != "Rifle" || rifle["country_tag"] != "" || rifle["source"] != "imported" ||
		rifle["tech_line"] != "infantry" || treePortJSON(rifle["research_cost"]) != "2" || project.Int64(rifle["start_year"]) != 1937 ||
		rifle["source_file"] != "common/technologies/infantry.txt" || rifle["is_banner"] != false || rifle["has_obs"] != false {
		t.Fatalf("TST_rifle %v", rifle)
	}
	rifle2 := techs["TST_rifle2"]
	if rifle2["name"] != "TST_rifle2" || treePortJSON(rifle2["prerequisites"]) != `["TST_rifle"]` || treePortJSON(rifle2["research_cost"]) != "1.5" {
		t.Fatalf("TST_rifle2 %v", rifle2)
	}

	if got := treePortJSON(p.Record()["tech_year_columns"]); got != `[["@1936",0],["@1940",2],["@1950",9]]` {
		t.Fatalf("tech_year_columns %s", got)
	}
	images := s.neededImageList()
	for _, want := range []string{"gfx/interface/goals/tst.dds", "gfx/interface/technologies/rifle.dds"} {
		found := false
		for _, image := range images {
			found = found || image == want
		}
		if !found {
			t.Fatalf("needed images %v lack %s", images, want)
		}
	}

	policies := treePortRows(t, p, "focus_file_policies", "source_file")
	if got := treePortKeys(policies); !reflect.DeepEqual(got, []string{
		"common/national_focus/TST_shared.txt", "common/national_focus/finland.txt",
		"common/national_focus/other.txt", "common/national_focus/tst.txt",
	}) {
		t.Fatalf("policies %v", got)
	}
	vanilla := policies["common/national_focus/finland.txt"]
	if vanilla["origin_kind"] != "imported_vanilla" || vanilla["target_layer"] != "vanilla" || vanilla["decision_source"] != "derived_exact_match" ||
		treePortJSON(vanilla["tree_ids"]) != `["finnish_focus"]` || treePortJSON(vanilla["matched_tree_ids"]) != `["finnish_focus"]` ||
		vanilla["registry_version"] != "" || vanilla["registry_sha256"] != "" {
		t.Fatalf("finland policy %v", vanilla)
	}
	own := policies["common/national_focus/tst.txt"]
	if own["origin_kind"] != "legacy_unknown" || own["emit_mode"] != "mandatory_shadow_same_path" || own["target_layer"] != "project" ||
		treePortJSON(own["tree_ids"]) != `["TST_focus"]` || own["matched_tree_ids"] != nil {
		t.Fatalf("tst policy %v", own)
	}
	if shared := policies["common/national_focus/TST_shared.txt"]; shared["emit_mode"] != "shared_pool_same_path" {
		t.Fatalf("shared policy %v", shared)
	}
}

func TestTreesPortOverwriteKeepsTreeRowsAndReplacesNodes(t *testing.T) {
	p := treePortDefaultProject(t)
	root := treePortMod(t)
	treePortImport(t, p, root, "overwrite")
	tree := treePortRows(t, p, "focus_trees", "tree_id")["TST_focus"]
	rifle := treePortRows(t, p, "technologies", "tech_id")["TST_rifle"]
	policy := treePortRows(t, p, "focus_file_policies", "source_file")["common/national_focus/tst.txt"]
	treePortUpdate(t, p, "focus_trees", project.Int64(tree["id"]), func(row project.Row) {
		row["edit_revision"] = 7
		row["initial_show_focus"] = "user"
	})

	treePortWrite(t, root, map[string]string{"common/national_focus/tst.txt": strings.Replace(
		strings.Replace(treePortFocusFile, "cost = 5", "cost = 7", 1),
		"focus = { id = TST_b prerequisite = { focus = TST_a } x = 1 y = 1 }", "focus = { id = TST_c x = 2 y = 1 }", 1)})
	treePortImport(t, p, root, "overwrite")

	after := treePortRows(t, p, "focus_trees", "tree_id")["TST_focus"]

	if project.Int64(after["id"]) != project.Int64(tree["id"]) || project.Int64(after["edit_revision"]) != 7 || after["initial_show_focus"] != "" {
		t.Fatalf("TST_focus after overwrite %v", after)
	}
	nodes := treePortRows(t, p, "focus_nodes", "focus_id")
	if _, ok := nodes["TST_b"]; ok {
		t.Fatal("TST_b survived the overwrite")
	}
	if c := nodes["TST_c"]; project.Int64(c["focus_tree_id"]) != project.Int64(tree["id"]) {
		t.Fatalf("TST_c %v", c)
	}
	if got := treePortJSON(nodes["TST_a"]["cost"]); got != "7" {
		t.Fatalf("TST_a cost %s", got)
	}

	techs := treePortRows(t, p, "technologies", "tech_id")
	if len(techs) != 2 || project.Int64(techs["TST_rifle"]["id"]) == project.Int64(rifle["id"]) {
		t.Fatalf("technologies %v (old TST_rifle id %v)", techs, rifle["id"])
	}
	policies := treePortRows(t, p, "focus_file_policies", "source_file")
	if len(policies) != 3 || project.Int64(policies["common/national_focus/tst.txt"]["id"]) != project.Int64(policy["id"]) {
		t.Fatalf("policies %v", policies)
	}
}

func TestTreesPortExplicitCountryTagReplacesProjectTag(t *testing.T) {
	p := treePortProject(t, project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{}})
	root := t.TempDir()
	treePortWrite(t, root, map[string]string{"common/national_focus/tst.txt": treePortFocusFile})
	if _, _, err := treePortRun(p, root, TreeStageOptions{CountryTag: " ger ", ImportFocus: true}, "overwrite"); err != nil {
		t.Fatal(err)
	}
	trees := treePortRows(t, p, "focus_trees", "tree_id")
	if got := treePortKeys(trees); !reflect.DeepEqual(got, []string{"GER_focus"}) || trees["GER_focus"]["country_tag"] != "GER" {
		t.Fatalf("trees %v", trees)
	}

	if got := treePortJSON(p.Record()["tech_year_columns"]); got != "[]" {
		t.Fatalf("tech_year_columns %s", got)
	}
}

func TestTreesPortPolicySourceCollisionFailsImport(t *testing.T) {

	p := treePortDefaultProject(t)
	root := t.TempDir()
	treePortWrite(t, root, map[string]string{
		"common/national_focus/finland.txt": "focus_tree = { id = custom_finland focus = { id = CF_a x = 0 y = 0 } }\n",
		"common/national_focus/renamed.txt": "focus_tree = { id = finnish_focus focus = { id = FIN_x x = 0 y = 0 } }\n",
	})
	_, _, err := treePortRun(p, root, TreeStageOptions{ImportFocus: true}, "overwrite")
	if err == nil || !strings.Contains(err.Error(), "uq_focus_file_policy_project_source") {
		t.Fatalf("error %v", err)
	}
	if rows, _ := p.Rows("focus_trees"); len(rows) != 0 {
		t.Fatalf("failed import left %d trees", len(rows))
	}
}

func TestTreesPortRejectsUnknownConflictPolicy(t *testing.T) {
	p := treePortDefaultProject(t)
	_, _, err := treePortRun(p, t.TempDir(), TreeStageOptions{}, "merge")
	if err == nil || !strings.Contains(err.Error(), "unsupported tree conflict policy") {
		t.Fatalf("error %v", err)
	}
}

func TestTreesPortStageImportedFocusPoliciesSplitsUpdatesAndInserts(t *testing.T) {
	policies := []importedFocusFilePolicy{
		{sourceFile: "common/national_focus/A.txt", originKind: "legacy_unknown", treeIDs: []string{"a"}, emitMode: "mandatory_shadow_same_path", targetPath: "common/national_focus/A.txt", targetLayer: "project", decisionSource: "legacy_repair"},
		{sourceFile: "common/national_focus/b.txt", originKind: "imported_vanilla", treeIDs: []string{"b"}, matchedTreeIDs: []string{"b"}},
	}
	staged := stageImportedFocusPolicies(policies, map[string]int64{"common/national_focus/a.txt": 4})
	if len(staged) != 2 || staged[0].existingID != 4 || staged[1].existingID != 0 {
		t.Fatalf("staged %+v", staged)
	}
	first := staged[0].row
	if first["source_file"] != "common/national_focus/A.txt" || treePortJSON(first["tree_ids"]) != `["a"]` || first["matched_tree_ids"] != nil ||
		first["target_layer"] != "project" || len(first) != 10 {
		t.Fatalf("row %v", first)
	}
	if treePortJSON(staged[1].row["matched_tree_ids"]) != `["b"]` {
		t.Fatalf("row %v", staged[1].row)
	}
}
