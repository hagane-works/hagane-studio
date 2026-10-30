package guirender

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func partialTestVanilla(t *testing.T, files map[string]string) *Index {
	t.Helper()
	manifest, err := json.Marshal(VanillaManifest{GameVersion: "1.16.9", Textures: map[string][]int{"gfx/interface/a.dds": {8, 8}}})
	if err != nil {
		t.Fatal(err)
	}
	raw := map[string][]byte{VanillaManifestPath: manifest}
	for name, body := range files {
		raw[name] = []byte(body)
	}
	vanilla, err := NewVanillaIndex(raw, "https://example.invalid/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	return vanilla
}

func rebuiltWhole(t *testing.T, vanilla *Index, files, folders []string) *Index {
	t.Helper()
	plain := *vanilla
	plain.masking = nil
	var out *Index
	var err error
	if len(folders) > 0 {
		out, err = plain.WithoutVanillaFolders(folders)
	} else {
		out, err = plain.WithoutFiles(files)
	}
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sameIndex(t *testing.T, label string, want, got *Index) {
	t.Helper()
	wantCopy, gotCopy := *want, *got
	wantCopy.masking, gotCopy.masking = nil, nil
	if reflect.DeepEqual(&wantCopy, &gotCopy) {
		return
	}
	wv, gv := reflect.ValueOf(&wantCopy).Elem(), reflect.ValueOf(&gotCopy).Elem()
	for i := 0; i < wv.NumField(); i++ {
		if !reflect.DeepEqual(indexField(wv, i).Interface(), indexField(gv, i).Interface()) {
			t.Errorf("%s: field %s differs from a whole rebuild", label, wv.Type().Field(i).Name)
		}
	}
}

var partialTestFiles = map[string]string{
	"interface/topbar.gui": `guiTypes = { containerWindowType = { name = topbar size = { width = 100 height = 20 } } }`,
	"interface/core.gfx":   `spriteTypes = { spriteType = { name = GFX_a texturefile = "gfx/interface/a.dds" } }`,
	"common/ideas/a.txt":   `ideas = { country = { idea_a = { picture = pic_a modifier = { stability_factor = 0.1 } } } }`,
	"common/ideas/b.txt":   `ideas = { country = { idea_b = { picture = pic_b } } }`,
	"events/x.txt":         "add_namespace = x\ncountry_event = { id = x.1 }\n",

	"interface/goals/b.gfx": `spriteTypes = { spriteType = { name = GFX_b texturefile = "gfx/interface/a.dds" } }`,
	"common/military_industrial_organization/organizations/CHI_organization.txt": `CHI_org = { name = CHI_org icon = GFX_a }`,
	"common/units/equipment/planes.txt": `equipments = {
	small_plane_airframe = { year = 1936 is_archetype = yes picture = archetype_small_plane }
	small_plane_airframe_1 = { year = 1936 archetype = small_plane_airframe }
}`,
	"common/units/equipment/x_plane_airframes.txt": `duplicate_archetypes = {
	small_plane_cas_airframe = { archetype = small_plane_airframe picture = archetype_CAS_equipment }
}`,
}

func TestMaskingTheBaseGameMatchesARebuildForEveryFile(t *testing.T) {
	vanilla := partialTestVanilla(t, partialTestFiles)
	names := make([]string, 0, len(partialTestFiles))
	for name := range partialTestFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		got, err := vanilla.WithoutFiles([]string{name})
		if err != nil {
			t.Fatal(err)
		}
		sameIndex(t, name, rebuiltWhole(t, vanilla, []string{name}, nil), got)
	}
	for _, folder := range []string{"common/ideas", "interface/goals", "events"} {
		got, err := vanilla.WithoutVanillaFolders([]string{folder})
		if err != nil {
			t.Fatal(err)
		}
		sameIndex(t, folder+"/", rebuiltWhole(t, vanilla, nil, []string{folder}), got)
	}
}

func TestMaskingTheBaseGameSharesWhatItDoesNotTouch(t *testing.T) {
	vanilla := partialTestVanilla(t, partialTestFiles)
	masked, err := vanilla.WithoutFiles([]string{"common/military_industrial_organization/organizations/CHI_organization.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.ValueOf(masked.windows).Pointer() != reflect.ValueOf(vanilla.windows).Pointer() ||
		reflect.ValueOf(masked.focusTrees).Pointer() != reflect.ValueOf(vanilla.focusTrees).Pointer() {
		t.Fatal("untouched records were read again")
	}
	if len(masked.industrialOrgs) != 0 || len(vanilla.industrialOrgs) != 1 {
		t.Fatalf("design companies: masked %d, base game %d", len(masked.industrialOrgs), len(vanilla.industrialOrgs))
	}
	sprites, err := vanilla.WithoutFiles([]string{"interface/goals/b.gfx"})
	if err != nil {
		t.Fatal(err)
	}
	_, lostB := sprites.sprites["GFX_b"]
	_, keptA := sprites.sprites["GFX_a"]
	_, baseB := vanilla.sprites["GFX_b"]
	if lostB || !keptA || !baseB {
		t.Fatalf("sprites from two folders: %v", sprites.sprites)
	}

	if masked.equipmentFor("small_plane_cas_airframe") == nil || vanilla.equipment["small_plane_cas_airframe"] != nil {
		t.Fatal("duplicated archetypes were written into the base game's table")
	}
}

func TestMaskingTheRealBaseGameMatchesARebuild(t *testing.T) {
	pack := os.Getenv("GUI_VANILLA_PACK_ZIP")
	if pack == "" {
		t.Skip("GUI_VANILLA_PACK_ZIP not set")
	}
	archive, err := zip.OpenReader(pack)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	files := map[string][]byte{}
	for _, member := range archive.File {
		if member.FileInfo().IsDir() {
			continue
		}
		body, err := member.Open()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(body)
		body.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[strings.ReplaceAll(member.Name, "\\", "/")] = raw
	}
	vanilla, err := NewVanillaIndex(files, "")
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	vanilla.PrepareMasking()
	t.Logf("plan: %d groups in %s", len(vanilla.masking.groups), time.Since(started))
	groups := make([]string, 0, len(vanilla.masking.groups))
	for group := range vanilla.masking.groups {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	for _, group := range groups {
		names := append([]string(nil), vanilla.masking.groups[group]...)
		sort.Strings(names)
		name := names[len(names)/2]
		started := time.Now()
		got, err := vanilla.WithoutFiles([]string{name})
		if err != nil {
			t.Fatal(err)
		}
		took := time.Since(started)
		sameIndex(t, name, rebuiltWhole(t, vanilla, []string{name}, nil), got)
		t.Logf("%-70s %s", name, took.Round(time.Millisecond))
	}
}
