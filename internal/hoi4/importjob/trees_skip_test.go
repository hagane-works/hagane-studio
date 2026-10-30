package importjob

import (
	"reflect"
	"strings"
	"testing"

	"haganeworks/internal/project"
)

func TestSkipTreeConflictsLeaveExistingRowsUntouched(t *testing.T) {
	p := treePortDefaultProject(t)
	root := treePortMod(t)
	treePortImport(t, p, root, "overwrite")
	tree := treePortRows(t, p, "focus_trees", "tree_id")["TST_focus"]
	node := treePortRows(t, p, "focus_nodes", "focus_id")["TST_a"]
	rifle := treePortRows(t, p, "technologies", "tech_id")["TST_rifle"]
	treePortUpdate(t, p, "focus_trees", project.Int64(tree["id"]), func(row project.Row) { row["initial_show_focus"] = "user" })
	treePortUpdate(t, p, "focus_nodes", project.Int64(node["id"]), func(row project.Row) { row["name_en"] = "Mine" })
	treePortUpdate(t, p, "technologies", project.Int64(rifle["id"]), func(row project.Row) { row["research_cost"] = 99 })
	before := map[string]project.Row{
		"tree":  treePortRows(t, p, "focus_trees", "tree_id")["TST_focus"],
		"node":  treePortRows(t, p, "focus_nodes", "focus_id")["TST_a"],
		"rifle": treePortRows(t, p, "technologies", "tech_id")["TST_rifle"],
	}

	treePortWrite(t, root, map[string]string{
		"common/national_focus/tst.txt": strings.Replace(treePortFocusFile, "cost = 5", "cost = 7", 1) +
			"focus_tree = { id = TST_more country = { factor = 0 modifier = { add = 10 tag = TST } } focus = { id = TST_c x = 0 y = 0 } }\n",
		"common/national_focus/new.txt": "focus_tree = { id = TST_new country = { factor = 0 modifier = { add = 10 tag = TST } } focus = { id = TST_new_a x = 0 y = 0 } }\n",
		"common/technologies/infantry.txt": strings.Replace(treePortTechFile, "technologies = {",
			"@1944 = 4\ntechnologies = {\n\tTST_rifle3 = { research_cost = 3 }", 1),
	})
	treePortImport(t, p, root, "skip")

	after := map[string]project.Row{
		"tree":  treePortRows(t, p, "focus_trees", "tree_id")["TST_focus"],
		"node":  treePortRows(t, p, "focus_nodes", "focus_id")["TST_a"],
		"rifle": treePortRows(t, p, "technologies", "tech_id")["TST_rifle"],
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("skip changed existing rows:\nbefore %v\nafter  %v", before, after)
	}
	trees := treePortRows(t, p, "focus_trees", "tree_id")
	nodes := treePortRows(t, p, "focus_nodes", "focus_id")
	if _, ok := trees["TST_new"]; !ok {
		t.Fatal("new tree TST_new was not imported")
	}
	if _, ok := trees["TST_more"]; !ok {
		t.Fatal("new tree TST_more was not imported")
	}
	if project.Int64(nodes["TST_c"]["focus_tree_id"]) != project.Int64(trees["TST_more"]["id"]) {
		t.Fatalf("TST_c %v", nodes["TST_c"])
	}
	if _, ok := treePortRows(t, p, "technologies", "tech_id")["TST_rifle3"]; !ok {
		t.Fatal("new technology TST_rifle3 was not imported")
	}

	if got := treePortJSON(p.Record()["tech_year_columns"]); got != `[["@1936",0],["@1940",2],["@1944",4],["@1950",9]]` {
		t.Fatalf("tech_year_columns %s", got)
	}
	if _, ok := treePortRows(t, p, "focus_file_policies", "source_file")["common/national_focus/new.txt"]; !ok {
		t.Fatal("no policy for new.txt")
	}
}
