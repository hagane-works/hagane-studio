package guirender

import "testing"

func TestTechTreeMarksTechnologiesOnlyTheTopLayerHas(t *testing.T) {
	base, err := NewIndex(map[string][]byte{
		"common/technologies/infantry.txt": []byte("technologies = {\n" +
			"\tinfantry_weapons = {\n\t\tpath = { leads_to_tech = infantry_weapons1 }\n" +
			"\t\tfolder = { name = infantry_folder position = { x = 0 y = 0 } }\n\t}\n" +
			"\tinfantry_weapons1 = {\n\t\tfolder = { name = infantry_folder position = { x = 0 y = 2 } }\n\t}\n}\n"),
	})
	if err != nil {
		t.Fatalf("base: %v", err)
	}
	top, err := NewIndexWithFallback(map[string][]byte{
		"common/technologies/infantry.txt": []byte("technologies = {\n" +
			"\tinfantry_weapons = {\n\t\tpath = { leads_to_tech = infantry_weapons1 }\n" +
			"\t\tpath = { leads_to_tech = rifle_drill }\n" +
			"\t\tfolder = { name = infantry_folder position = { x = 0 y = 0 } }\n\t}\n" +
			"\trifle_drill = {\n\t\tfolder = { name = infantry_folder position = { x = 1 y = 2 } }\n\t}\n}\n"),
	}, base)
	if err != nil {
		t.Fatalf("top: %v", err)
	}
	marked := map[string]bool{}
	for _, item := range top.TechTreePayload("GER", "1936.1.1", "infantry_folder")["technologies"].([]map[string]any) {
		name, _ := item["name"].(string)
		marked[name], _ = item["added_by_project"].(bool)
	}
	if len(marked) != 3 {
		t.Fatalf("technologies on the page: %v", marked)
	}
	if !marked["rifle_drill"] || marked["infantry_weapons"] || marked["infantry_weapons1"] {
		t.Fatalf("added_by_project = %v, want rifle_drill only", marked)
	}

	for _, item := range base.TechTreePayload("GER", "1936.1.1", "infantry_folder")["technologies"].([]map[string]any) {
		if item["added_by_project"] != nil {
			t.Fatalf("%v marked with no layer below", item["name"])
		}
	}
}
