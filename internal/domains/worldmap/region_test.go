package worldmap

import (
	"strings"
	"testing"

	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/mapgen"
)

func TestRegionTextWritesIntoTheProjectsOwnRegionFiles(t *testing.T) {
	const (
		caribbean = "map/strategicregions/53-Caribbean.txt"
		base      = "map/strategicregions/54-Gulf.txt"
		written   = "map/strategicregions/314-New.txt"
		binary    = "map/strategicregions/55-Packed.txt"
		patched   = "map/strategicregions/56-Patched.txt"
	)
	game := "strategic_region={\n\tid=53\n\tname=\"STRATEGICREGION_53\"\n\tprovinces={\n\t\t1 2\n\t}\n}\n"
	imported := "strategic_region={\n\tid=53\n\tname=\"STRATEGICREGION_53\"\n\tprovinces={\n\t\t1 2 3\n\t}\n}\n"
	ours := "strategic_region={\n\tid=314\n\tname=\"STRATEGICREGION_314\"\n\tprovinces={\n\t\t9001\n\t}\n}\n"
	index, err := guirender.NewIndex(map[string][]byte{caribbean: []byte(game), base: []byte(game)})
	if err != nil {
		t.Fatal(err)
	}
	rowOf := func(path, content string, binary bool, kind string) RawFile {
		return RawFile{DestPath: path, Content: content, IsBinary: binary, RawKind: kind}
	}
	files := &landFileSet{
		index: index,
		raw: map[string]RawFile{
			strings.ToLower(caribbean): rowOf(caribbean, imported, false, "verbatim_override"),
			strings.ToLower(written):   rowOf(written, ours, false, "verbatim_override"),
			strings.ToLower(binary):    rowOf(binary, "", true, "verbatim_override"),
			strings.ToLower(patched):   rowOf(patched, "x", false, "empty_shadow"),
		},
		ours: map[string]bool{strings.ToLower(written): true},
	}

	text, refusal, err := files.regionText(caribbean)
	if err != nil || refusal != "" || text != imported {
		t.Fatalf("the project's own imported region file must be written into as it is in the project: %q, %q, %v", text, refusal, err)
	}
	if got := mapgen.RegionProvinces(mapgen.AddToRegion(text, []int{9100})); !regionHasAll(got, 1, 2, 3, 9100) {
		t.Fatalf("the cut water joins the imported file's provinces, got %v", got)
	}
	if text, refusal, err := files.regionText(written); err != nil || refusal != "" || text != ours {
		t.Fatalf("an addition's own region file comes from its row: %q, %q, %v", text, refusal, err)
	}
	if text, refusal, err := files.regionText(base); err != nil || refusal != "" || text != game {
		t.Fatalf("without a project row the base game's file is used: %q, %q, %v", text, refusal, err)
	}
	for _, file := range []string{binary, patched} {
		if _, refusal, err := files.regionText(file); err != nil || !strings.Contains(refusal, file) {
			t.Fatalf("%s: a project file that is not a whole-text override must be refused by name, got %q, %v", file, refusal, err)
		}
	}
	if _, refusal, err := files.regionText("map/strategicregions/999-Nowhere.txt"); err == nil || refusal != "" {
		t.Fatalf("a region file nobody has is an error, got %q, %v", refusal, err)
	}
}

func regionHasAll(have []int, want ...int) bool {
	set := map[int]bool{}
	for _, id := range have {
		set[id] = true
	}
	for _, id := range want {
		if !set[id] {
			return false
		}
	}
	return true
}
