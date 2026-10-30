package guirender

import "testing"

func indexWithExternalTextures(t *testing.T, source string, textures map[string]ExternalTexture) *Index {
	t.Helper()
	idx, err := newIndex(map[string][]byte{}, source)
	if err != nil {
		t.Fatal(err)
	}
	idx.external = textures
	idx.externalLoose = buildLooseTextureIndex(textures)
	return idx
}

func eventIDs(index *Index) map[string]bool {
	out := map[string]bool{}
	for _, row := range index.ListEvents("") {
		if id, ok := row["id"].(string); ok {
			out[id] = true
		}
	}
	return out
}

func TestWithoutFilesPreventsShadowedDeclarationsFromResurrecting(t *testing.T) {
	base, err := NewIndex(map[string][]byte{
		"events/Shared.txt": []byte("add_namespace = old\ncountry_event = { id = old.1 }\ncountry_event = { id = old.2 }\n"),
		"events/Other.txt":  []byte("add_namespace = keep\ncountry_event = { id = keep.1 }\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	masked, err := base.WithoutFiles([]string{"EVENTS/shared.TXT"})
	if err != nil {
		t.Fatal(err)
	}
	top, err := NewIndexWithFallback(map[string][]byte{
		"events/shared.txt": []byte("add_namespace = replacement\ncountry_event = { id = replacement.1 }\n"),
	}, masked)
	if err != nil {
		t.Fatal(err)
	}
	ids := eventIDs(top)
	if ids["old.1"] || ids["old.2"] {
		t.Fatalf("same-path lower events were resurrected: %v", ids)
	}
	if !ids["replacement.1"] || !ids["keep.1"] {
		t.Fatalf("replacement or unrelated event missing: %v", ids)
	}
	if base.EventPayload("old.1") == nil {
		t.Fatal("mask mutated the original index")
	}
}

func TestWithoutFoldersRemovesBaseModAndVanillaLayers(t *testing.T) {
	vanilla, err := newIndex(map[string][]byte{
		"events/vanilla.txt": []byte("add_namespace = vanilla\ncountry_event = { id = vanilla.1 }\n"),
	}, "vanilla")
	if err != nil {
		t.Fatal(err)
	}
	base, err := NewIndexWithFallback(map[string][]byte{
		"events/base.txt":       []byte("add_namespace = base\ncountry_event = { id = base.1 }\n"),
		"common/ideas/base.txt": []byte("ideas = { country = { base_idea = { } } }"),
	}, vanilla)
	if err != nil {
		t.Fatal(err)
	}
	masked, err := base.WithoutFolders([]string{"events"})
	if err != nil {
		t.Fatal(err)
	}
	ids := eventIDs(masked)
	if ids["base.1"] || ids["vanilla.1"] {
		t.Fatalf("replace_path left lower events behind: %v", ids)
	}
	if _, ok := masked.Source("common/ideas/base.txt"); !ok {
		t.Fatal("replace_path removed an unrelated directory")
	}
}

func TestWithoutFilesRemovesManifestOnlyTextureAndLooseLookup(t *testing.T) {
	base, err := newIndex(map[string][]byte{
		"events/replaced.txt": []byte("add_namespace = replaced\ncountry_event = { id = replaced.1 }\n"),
	}, "mod")
	if err != nil {
		t.Fatal(err)
	}
	base.external = map[string]ExternalTexture{
		"gfx/interface/goals/goal_hidden.dds": {Width: 100, Height: 88},
		"gfx/interface/topbar/kept.dds":       {Width: 20, Height: 20},
	}
	base.externalLoose = buildLooseTextureIndex(base.external)
	masked, err := base.WithoutFiles([]string{`/GFX\interface\GOALS\goal_hidden.DDS`, "EVENTS/replaced.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, found := masked.textureSize("gfx/interface/goals/goal_hidden.dds"); found {
		t.Fatal("same-path manifest texture survived a file shadow")
	}
	if _, found := masked.looseExternal("GFX_goal_hidden"); found {
		t.Fatal("removed texture survived through the loose external index")
	}
	if _, _, _, found := masked.textureSize("gfx/interface/topbar/kept.dds"); !found {
		t.Fatal("unrelated manifest texture was removed")
	}
	if _, found := base.looseExternal("GFX_goal_hidden"); !found {
		t.Fatal("mask mutated the original external texture index")
	}
}

func TestWithoutFoldersRemovesManifestTexturesFromEveryLowerLayer(t *testing.T) {
	vanilla := indexWithExternalTextures(t, "vanilla", map[string]ExternalTexture{
		"gfx/interface/goals/goal_vanilla.dds":    {Width: 100, Height: 88},
		"gfx/interface/goals/sub/goal_nested.dds": {Width: 100, Height: 88},
		"gfx/interface/topbar/vanilla_kept.dds":   {Width: 20, Height: 20},
	})
	base := indexWithExternalTextures(t, "mod", map[string]ExternalTexture{
		"gfx/interface/goals/goal_imported.dds": {Width: 100, Height: 88},
	})
	base.fallback = vanilla

	masked, err := base.WithoutFolders([]string{"GFX/interface/goals/"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"GFX_goal_imported", "GFX_goal_vanilla"} {
		if _, found := masked.looseExternal(name); found {
			t.Fatalf("%s survived replace_path in externalLoose", name)
		}
		if _, _, rel, found := masked.looseTextureSize(name); found || rel != "" {
			t.Fatalf("%s resurrected from a lower external manifest: %q %v", name, rel, found)
		}
	}
	if _, _, _, found := masked.textureSize("gfx/interface/goals/sub/goal_nested.dds"); !found {
		t.Fatal("direct-folder replace_path incorrectly removed a nested texture")
	}
	if _, _, _, found := masked.textureSize("gfx/interface/topbar/vanilla_kept.dds"); !found {
		t.Fatal("replace_path removed an unrelated external texture")
	}
}

func TestWithoutFilesKeepsOnlySurvivingProjectFileSources(t *testing.T) {
	index, err := NewIndex(map[string][]byte{
		"interface/replaced.gui": []byte(`guiTypes = { containerWindowType = { name = replaced } }`),
		"interface/kept.gui":     []byte(`guiTypes = { containerWindowType = { name = kept } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	index.SetProjectFileSources(map[string]string{
		"interface/replaced.gui":     "project_raw_files:1",
		"interface/kept.gui":         "project_raw_files:2",
		"gfx/interface/replaced.dds": "project_raw_files:3",
	})

	masked, err := index.WithoutFiles([]string{
		"INTERFACE/replaced.GUI",
		"gfx/interface/replaced.dds",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := masked.ProjectFileSource("interface/replaced.gui"); ok {
		t.Fatal("rebuilt index retained provenance for a removed text file")
	}
	if _, ok := masked.ProjectFileSource("gfx/interface/replaced.dds"); ok {
		t.Fatal("rebuilt index retained provenance for a removed binary-only file")
	}
	if source, ok := masked.ProjectFileSource("interface/kept.gui"); !ok || source != "project_raw_files:2" {
		t.Fatalf("surviving provenance = %q, %v", source, ok)
	}
	if source, ok := index.ProjectFileSource("interface/replaced.gui"); !ok || source != "project_raw_files:1" {
		t.Fatal("mask mutated the original index provenance")
	}
}

func TestWithoutFilesReusesTheChainForTheSameRemovals(t *testing.T) {
	base, err := NewIndex(map[string][]byte{
		"events/Shared.txt": []byte("add_namespace = old\ncountry_event = { id = old.1 }\n"),
		"events/Other.txt":  []byte("add_namespace = keep\ncountry_event = { id = keep.1 }\n"),
		"events/Third.txt":  []byte("add_namespace = third\ncountry_event = { id = third.1 }\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := base.WithoutFiles([]string{"events/shared.txt", "events/not_in_base.txt"})
	if err != nil {
		t.Fatal(err)
	}

	again, err := base.WithoutFiles([]string{"EVENTS/Shared.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if first != again {
		t.Fatal("the same removals re-parsed the base instead of reusing it")
	}
	if ids := eventIDs(first); ids["old.1"] || !ids["keep.1"] {
		t.Fatalf("reused chain kept the shadowed file or lost another: %v", ids)
	}

	other, err := base.WithoutFiles([]string{"events/third.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if other == first {
		t.Fatal("different removals returned the same chain")
	}
	if ids := eventIDs(other); !ids["old.1"] || ids["third.1"] {
		t.Fatalf("second mask removed the wrong file: %v", ids)
	}

	if same, err := base.WithoutFiles([]string{"events/absent.txt"}); err != nil || same != base {
		t.Fatalf("a mask that removes nothing changed the chain: %v", err)
	}
}

func TestWithoutFilesAndFoldersMasksBothInOneLayer(t *testing.T) {
	base, err := newIndex(map[string][]byte{
		"events/shadowed.txt":     []byte("add_namespace = a\ncountry_event = { id = a.1 }\n"),
		"events/kept.txt":         []byte("add_namespace = b\ncountry_event = { id = b.1 }\n"),
		"history/events/deep.txt": []byte("add_namespace = c\ncountry_event = { id = c.1 }\n"),
	}, "vanilla")
	if err != nil {
		t.Fatal(err)
	}
	masked, err := base.WithoutFilesAndFolders([]string{"EVENTS/Shadowed.txt"}, []string{"history/events"})
	if err != nil {
		t.Fatal(err)
	}
	if ids := eventIDs(masked); ids["a.1"] || ids["c.1"] || !ids["b.1"] {
		t.Fatalf("masked events = %v", ids)
	}
	if got := base.OwnFiles(); len(got) != 3 || got[0] != "events/kept.txt" {
		t.Fatalf("own files of the original = %v", got)
	}
	if same, err := base.WithoutFilesAndFolders(nil, []string{"interface"}); err != nil || same != base {
		t.Fatalf("a mask removing nothing must return the receiver: %v", err)
	}
}

func TestOverLinksAStandaloneLayerOntoALowerChain(t *testing.T) {
	lower, err := newIndex(map[string][]byte{
		"common/units/equipment/planes.txt": []byte(`equipments = {
	small_plane_airframe = { year = 1936 is_archetype = yes picture = archetype_small_plane }
	small_plane_airframe_1 = { year = 1936 archetype = small_plane_airframe }
}`),
		"events/lower.txt": []byte("add_namespace = low\ncountry_event = { id = low.1 }\n"),
	}, "vanilla")
	if err != nil {
		t.Fatal(err)
	}
	own, err := NewIndexWithFallback(map[string][]byte{
		"common/units/equipment/x_plane_airframes.txt": []byte(`duplicate_archetypes = {
	small_plane_cas_airframe = {
		archetype = small_plane_airframe
		picture = archetype_CAS_equipment
	}
}`),
		"events/own.txt": []byte("add_namespace = own\ncountry_event = { id = own.1 }\n"),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if own.equipmentFor("small_plane_cas_airframe") != nil {
		t.Fatal("a standalone layer has no archetype to copy")
	}
	linked := own.Over(lower)
	if ids := eventIDs(linked); !ids["own.1"] || !ids["low.1"] {
		t.Fatalf("linked events = %v", ids)
	}
	copied := linked.equipmentFor("small_plane_cas_airframe")
	if copied == nil || copied.Picture != "archetype_CAS_equipment" || linked.equipmentFor("small_plane_cas_airframe_1") == nil {
		t.Fatalf("copies of the lower archetype: %+v", copied)
	}
	if own.equipmentFor("small_plane_cas_airframe") != nil || own.fallback != nil {
		t.Fatal("Over changed the standalone layer")
	}
}
