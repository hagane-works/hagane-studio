package guirender

import (
	"sort"
	"testing"
)

func threeLayers(t *testing.T) *Index {
	t.Helper()
	vanilla, err := NewIndex(map[string][]byte{
		"interface/base.gui": []byte(`guiTypes = { containerWindowType = { name = base_window } }`),
		"events/base.txt": []byte("add_namespace = base\n" +
			"country_event = { id = base.1 title = base.1.t }\n" +
			"country_event = { id = base.2 title = base.2.t }\n"),
		"localisation/english/base_l_english.yml": []byte("l_english:\n base.2.t:0 \"Base\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	mod, err := NewIndexWithFallback(map[string][]byte{
		"interface/mod.gui": []byte(`guiTypes = { containerWindowType = { name = mod_window } }`),

		"events/mod.txt": []byte("add_namespace = base\n" +
			"country_event = { id = base.2 title = base.2.modded }\n"),
		"localisation/english/mod_l_english.yml": []byte("l_english:\n base.2.modded:0 \"Modded\"\n"),
	}, vanilla)
	if err != nil {
		t.Fatal(err)
	}
	project, err := NewIndexWithFallback(map[string][]byte{
		"interface/project.gui": []byte(`guiTypes = { containerWindowType = { name = project_window } }`),
		"events/project.txt": []byte("add_namespace = project\n" +
			"country_event = { id = project.1 title = project.1.t }\n"),
	}, mod)
	if err != nil {
		t.Fatal(err)
	}
	return project
}

func TestAThreeLayerIndexStillReachesTheBaseGame(t *testing.T) {
	idx := threeLayers(t)

	names := []string{}
	for _, row := range idx.ListWindows() {
		names = append(names, row["name"].(string))
	}
	sort.Strings(names)
	if got := join(names); got != "base_window,mod_window,project_window" {
		t.Errorf("windows = %s", got)
	}
	if _, ok := idx.Window("base_window"); !ok {
		t.Error("a vanilla-only window is not found through two layers")
	}

	titles := map[string]string{}
	for _, row := range idx.ListEvents("") {
		id := row["id"].(string)
		if _, twice := titles[id]; twice {
			t.Errorf("%s listed twice", id)
		}
		titles[id], _ = row["title"].(string)
	}
	if len(titles) != 3 {
		t.Errorf("events = %v, want base.1, base.2 and project.1", titles)
	}

	if titles["base.2"] != "Modded" {
		t.Errorf("base.2 title = %q, want the mod's", titles["base.2"])
	}
}

func join(values []string) string {
	out := ""
	for i, v := range values {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}
