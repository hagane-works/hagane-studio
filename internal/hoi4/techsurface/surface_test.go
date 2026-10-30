package techsurface

import (
	"slices"
	"strings"
	"testing"
)

const baselineGUI = `guiTypes = {
	containerWindowType = {
		name = "countrytechtreeview"
		containerWindowType = {
			name = "infantry_folder"
			gridboxtype = {
				name = "infantry_weapons_tree"
				position = { x = 140 y = 325 }
				slotsize = { width = 70 height = 70 }
				format = "LEFT"
			}
		}
	}
}
`

func TestProductionProfilesProjectOnlyCertifiedCatalogueTargets(t *testing.T) {
	got := Profiles()
	if len(got) != 8 {
		t.Fatalf("production profile count=%d, want 8", len(got))
	}
	wantIDs := []string{
		"p0_prod_nsb0_bba0_mtg0_fec641fb", "p0_prod_nsb0_bba0_mtg1_fec641fb",
		"p0_prod_nsb0_bba1_mtg0_fec641fb", "p0_prod_nsb0_bba1_mtg1_fec641fb",
		"p0_prod_nsb1_bba0_mtg0_fec641fb", "p0_prod_nsb1_bba0_mtg1_fec641fb",
		"p0_prod_nsb1_bba1_mtg0_fec641fb", "p0_prod_nsb1_bba1_mtg1_fec641fb",
	}
	gotIDs := make([]string, len(got))
	for index := range got {
		gotIDs[index] = got[index].ID
		if got[index].BaselineSHA256 != "fec641fb738ab59369cac40ae902fdba793bed7cbaa2c807756207a2e1f01045" {
			t.Fatalf("profile %s baseline=%q", got[index].ID, got[index].BaselineSHA256)
		}
		if _, exists := got[index].Folders["armour_folder"]; exists {
			t.Fatalf("profile %s exposed uncertified base armour", got[index].ID)
		}
		if _, exists := got[index].Folders["nsb_armour_folder"]; exists {
			t.Fatalf("profile %s exposed uncertified NSB armour", got[index].ID)
		}
	}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Fatalf("production profile IDs=%v, want %v", gotIDs, wantIDs)
	}

	base, ok := ProfileByID(wantIDs[0])
	if !ok {
		t.Fatal("base production profile missing")
	}
	assertFolderProfile(t, base, "infantry_folder", "army", "LEFT", 70, 70, "p0_infantry_extension_1", 140, 1225, []TimelineCell{{1936, 0, 2}, {1939, 0, 6}, {1943, 0, 14}})
	assertFolderProfile(t, base, "air_techs_folder", "air", "UP", 60, 70, "p0_base_air_slot", 2240, 32, []TimelineCell{{1933, 0, 2}, {1940, 0, 6}})
	assertFolderProfile(t, base, "naval_folder", "navy", "LEFT", 70, 70, "p0_base_naval_slot", 150, 400, []TimelineCell{{1936, 0, 7}, {1940, 0, 13}})
	assertFolderProfile(t, base, "industry_folder", "industry_equipment", "UP", 70, 70, "p0_base_industry_slot", 1350, 172, []TimelineCell{{1936, 0, 0}, {1937, 0, 4}})
	assertFolderProfile(t, base, "electronics_folder", "industry_equipment", "UP", 60, 60, "p0_base_electronics_slot", 1550, 172, []TimelineCell{{1936, -1, 0}, {1938, -1, 6}})

	allDLC, ok := ProfileByID(wantIDs[7])
	if !ok {
		t.Fatal("all-DLC production profile missing")
	}
	assertFolderProfile(t, allDLC, "bba_air_techs_folder", "air", "UP", 70, 70, "p0_dlc_air_slot", 2100, 32, []TimelineCell{{1933, 0, 2}, {1940, 0, 6}})
	assertFolderProfile(t, allDLC, "mtgnavalfolder", "navy", "LEFT", 70, 70, "p0_dlc_naval_main_slot", 70, 1160, []TimelineCell{{1936, 0, 8}, {1939, 0, 14}})
	assertFolderProfile(t, allDLC, "mtgnavalsupportfolder", "navy", "LEFT", 70, 70, "p0_dlc_naval_support_slot", 150, 1170, []TimelineCell{{1922, 0, 1}, {1936, 0, 5}})
}

func TestLegacyInfantryProfileIsReadableButNotAdvertised(t *testing.T) {
	for _, profile := range Profiles() {
		if profile.ID == "p0_vanilla_infantry_fec641fb" {
			t.Fatal("legacy compatibility profile was advertised as a production choice")
		}
	}
	legacy, ok := ProfileByID("p0_vanilla_infantry_fec641fb")
	if !ok || len(legacy.Folders) != 1 {
		t.Fatalf("legacy profile=%+v ok=%t", legacy, ok)
	}
	if _, ok := legacy.Folders["infantry_folder"]; !ok {
		t.Fatalf("legacy profile folders=%v", legacy.Folders)
	}
}

func TestProductionAllPlayerManifestRequiresBaseAndDLCPlacements(t *testing.T) {
	profile, ok := ProfileByID("p0_prod_nsb0_bba0_mtg0_fec641fb")
	if !ok {
		t.Fatal("base production profile missing")
	}
	manifest := Manifest{SchemaVersion: SchemaVersion, Profile: profile.ID, BaselineSHA256: profile.BaselineSHA256, CompatibilityMode: "all_players", Branches: []Branch{{
		ID: "air", Surface: "air", Folder: "air_techs_folder", Slot: "p0_base_air_slot", RootTechnologyID: "MOD_air", TechnologyIDs: []string{"MOD_air"},
		Placements: []BranchPlacement{{Folder: "air_techs_folder", Slot: "p0_base_air_slot"}},
	}}}
	findings := ValidateShape(manifest, false)
	found := false
	for _, finding := range findings {
		if finding.Code == "tech_surface_all_players_placement_missing" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing Base/DLC placement finding absent: %#v", findings)
	}
}

func assertFolderProfile(t *testing.T, profile Profile, folderID, surface, format string, width, height int, slotID string, slotX, slotY int, timeline []TimelineCell) {
	t.Helper()
	folder, ok := profile.Folders[folderID]
	if !ok {
		t.Fatalf("profile %s missing folder %s", profile.ID, folderID)
	}
	if folder.Surface != surface || folder.GridboxFormat != format || folder.SlotWidth != width || folder.SlotHeight != height {
		t.Fatalf("folder %s presentation=%+v", folderID, folder)
	}
	slot, ok := folder.Slots[slotID]
	if !ok || slot.X != slotX || slot.Y != slotY {
		t.Fatalf("folder %s slot=%+v ok=%t", folderID, slot, ok)
	}
	if !slices.Equal(folder.Timeline, timeline) {
		t.Fatalf("folder %s timeline=%v, want %v", folderID, folder.Timeline, timeline)
	}
}

func testProfile(t *testing.T) (Manifest, func()) {
	t.Helper()
	const id = "test_infantry"
	restore := RegisterProfileForTest(Profile{ID: id, BaselineSHA256: Hash([]byte(baselineGUI)), Folders: map[string]FolderProfile{
		"infantry_folder": {ID: "infantry_folder", Surface: "army", Slots: map[string]Slot{
			"slot": {ID: "slot", X: 140, Y: 1225},
		}, Timeline: []TimelineCell{
			{StartYear: 1936, X: 0, Y: 2},
			{StartYear: 1939, X: 0, Y: 6},
		}},
	}})
	return Manifest{SchemaVersion: SchemaVersion, Profile: id, BaselineSHA256: Hash([]byte(baselineGUI)), Branches: []Branch{{
		ID: "rifle", Surface: "army", Folder: "infantry_folder", Slot: "slot", RootTechnologyID: "MOD_rifle", TechnologyIDs: []string{"MOD_rifle", "MOD_rifle_2"},
	}}}, restore
}

func TestCompileAddsOneRootGridboxWithoutReformattingSource(t *testing.T) {
	manifest, restore := testProfile(t)
	defer restore()
	techs := []Technology{
		{ID: "MOD_rifle", Source: "created", Folder: "infantry_folder", StartYear: 1936, FolderPlacements: []FolderPlacement{{Folder: "infantry_folder", X: 0, Y: 2}}, PathTargets: []string{"MOD_rifle_2"}},
		{ID: "MOD_rifle_2", Source: "created", Folder: "infantry_folder", StartYear: 1939, FolderPlacements: []FolderPlacement{{Folder: "infantry_folder", X: 0, Y: 6}}},
	}
	got, err := Compile([]byte(baselineGUI), manifest, techs, false)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "name = \"MOD_rifle_tree\"") || !strings.Contains(text, "position = { x = 140 y = 1225 }") {
		t.Fatalf("generated gridbox missing:\n%s", text)
	}
	if !strings.Contains(text, "name = \"infantry_weapons_tree\"\n\t\t\t\tposition") {
		t.Fatalf("existing GUI source was rewritten:\n%s", text)
	}
	if !strings.HasPrefix(text, strings.TrimSuffix(baselineGUI, "\n")) {

		prefix := baselineGUI[:strings.LastIndex(baselineGUI, "\t\t}")]
		if !strings.HasPrefix(text, prefix) {
			t.Fatalf("source prefix changed")
		}
	}
}

func TestCompileAddsSameRootGridboxToEveryCompatibleResearchPage(t *testing.T) {
	const dualGUI = `guiTypes = {
	containerWindowType = {
		name = "countrytechtreeview"
		containerWindowType = { name = "air_techs_folder" }
		containerWindowType = { name = "bba_air_techs_folder" }
	}
}
`
	const profileID = "test_dual_air"
	restore := RegisterProfileForTest(Profile{ID: profileID, BaselineSHA256: Hash([]byte(dualGUI)), Folders: map[string]FolderProfile{
		"air_techs_folder":     {ID: "air_techs_folder", Surface: "air", GridboxFormat: "UP", SlotWidth: 60, SlotHeight: 70, Slots: map[string]Slot{"base": {ID: "base", X: 10, Y: 20}}, Timeline: []TimelineCell{{StartYear: 1933, X: 0, Y: 2}}},
		"bba_air_techs_folder": {ID: "bba_air_techs_folder", Surface: "air", GridboxFormat: "UP", SlotWidth: 70, SlotHeight: 70, Slots: map[string]Slot{"bba": {ID: "bba", X: 30, Y: 40}}, Timeline: []TimelineCell{{StartYear: 1933, X: 0, Y: 2}}},
	}})
	defer restore()
	manifest := Manifest{SchemaVersion: SchemaVersion, Profile: profileID, BaselineSHA256: Hash([]byte(dualGUI)), CompatibilityMode: "all_players", Branches: []Branch{{
		ID: "air", Surface: "air", Folder: "air_techs_folder", Slot: "base", RootTechnologyID: "MOD_air", TechnologyIDs: []string{"MOD_air"},
		Placements: []BranchPlacement{{Folder: "air_techs_folder", Slot: "base"}, {Folder: "bba_air_techs_folder", Slot: "bba"}},
	}}}
	techs := []Technology{{ID: "MOD_air", Source: "created", Folder: "air_techs_folder", StartYear: 1933, FolderPlacements: []FolderPlacement{{Folder: "air_techs_folder", X: 0, Y: 2}, {Folder: "bba_air_techs_folder", X: 0, Y: 2}}}}
	got, err := Compile([]byte(dualGUI), manifest, techs, false)
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(got), `name = "MOD_air_tree"`); count != 2 {
		t.Fatalf("compatible root gridbox count=%d, want 2:\n%s", count, got)
	}
}

func TestCompileUsesCertifiedVerticalGridboxPresentation(t *testing.T) {
	const profileID = "test_air_vertical"
	restore := RegisterProfileForTest(Profile{ID: profileID, BaselineSHA256: Hash([]byte(baselineGUI)), Folders: map[string]FolderProfile{
		"infantry_folder": {
			ID: "infantry_folder", Surface: "air", GridboxFormat: "UP", SlotWidth: 60, SlotHeight: 70,
			Slots:    map[string]Slot{"slot": {ID: "slot", X: 2100, Y: 32}},
			Timeline: []TimelineCell{{StartYear: 1933, X: 0, Y: 2}, {StartYear: 1940, X: 0, Y: 6}},
		},
	}})
	defer restore()
	manifest := Manifest{SchemaVersion: SchemaVersion, Profile: profileID, BaselineSHA256: Hash([]byte(baselineGUI)), Branches: []Branch{{
		ID: "air", Surface: "air", Folder: "infantry_folder", Slot: "slot", RootTechnologyID: "MOD_air", TechnologyIDs: []string{"MOD_air", "MOD_air_2"},
	}}}
	techs := []Technology{
		{ID: "MOD_air", Source: "created", Folder: "infantry_folder", StartYear: 1933, FolderPlacements: []FolderPlacement{{Folder: "infantry_folder", X: 0, Y: 2}}, PathTargets: []string{"MOD_air_2"}},
		{ID: "MOD_air_2", Source: "created", Folder: "infantry_folder", StartYear: 1940, FolderPlacements: []FolderPlacement{{Folder: "infantry_folder", X: 0, Y: 6}}},
	}
	got, err := Compile([]byte(baselineGUI), manifest, techs, false)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, want := range []string{`name = "MOD_air_tree"`, `slotsize = { width = 60 height = 70 }`, `format = "UP"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated vertical gridbox missing %q:\n%s", want, text)
		}
	}
}

func TestCompileRejectsUncertifiedGridboxPresentation(t *testing.T) {
	manifest, restore := testProfile(t)
	defer restore()
	profile, _ := ProfileByID(manifest.Profile)
	folder := profile.Folders["infantry_folder"]
	folder.GridboxFormat = "DIAGONAL"
	profile.Folders["infantry_folder"] = folder
	restoreInvalid := RegisterProfileForTest(profile)
	defer restoreInvalid()
	techs := []Technology{
		{ID: "MOD_rifle", Source: "created", Folder: "infantry_folder", StartYear: 1936, FolderPlacements: []FolderPlacement{{Folder: "infantry_folder", X: 0, Y: 2}}, PathTargets: []string{"MOD_rifle_2"}},
		{ID: "MOD_rifle_2", Source: "created", Folder: "infantry_folder", StartYear: 1939, FolderPlacements: []FolderPlacement{{Folder: "infantry_folder", X: 0, Y: 6}}},
	}
	if _, err := Compile([]byte(baselineGUI), manifest, techs, false); err == nil || !strings.Contains(err.Error(), "unsupported gridbox format") {
		t.Fatalf("Compile error=%v, want unsupported gridbox format", err)
	}
}

func TestValidateRejectsAOneCellPseudoTimeline(t *testing.T) {
	manifest, restore := testProfile(t)
	defer restore()
	techs := []Technology{
		{ID: "MOD_rifle", Source: "created", Folder: "infantry_folder", StartYear: 1936, FolderPlacements: []FolderPlacement{{Folder: "infantry_folder", X: 0, Y: 0}}, PathTargets: []string{"MOD_rifle_2"}},
		{ID: "MOD_rifle_2", Source: "created", Folder: "infantry_folder", StartYear: 1936, FolderPlacements: []FolderPlacement{{Folder: "infantry_folder", X: 0, Y: 1}}},
	}
	codes := map[string]bool{}
	for _, finding := range Validate(manifest, techs, false) {
		codes[finding.Code] = true
	}
	if !codes["tech_surface_timeline_position_mismatch"] || !codes["tech_surface_timeline_year_mismatch"] {
		t.Fatalf("findings=%v", codes)
	}
}

func TestValidateRejectsFreeSlotAndPathChildOutsideFolder(t *testing.T) {
	manifest, restore := testProfile(t)
	defer restore()
	manifest.Branches[0].Slot = "freehand"
	techs := []Technology{
		{ID: "MOD_rifle", Source: "created", Folder: "infantry_folder", PathTargets: []string{"MOD_rifle_2"}},
		{ID: "MOD_rifle_2", Source: "created", Folder: "support_folder"},
	}
	codes := map[string]bool{}
	for _, f := range Validate(manifest, techs, false) {
		codes[f.Code] = true
	}
	if !codes["tech_surface_slot_not_certified"] {
		t.Fatalf("findings=%v", codes)
	}
}

func TestValidateRequiresEveryCreatedFolderRootToBeDeclared(t *testing.T) {
	manifest, restore := testProfile(t)
	defer restore()
	techs := []Technology{
		{ID: "MOD_rifle", Source: "created", Folder: "infantry_folder"},
		{ID: "MOD_other", Source: "created", Folder: "infantry_folder"},
	}
	codes := map[string]bool{}
	for _, f := range Validate(manifest, techs, false) {
		codes[f.Code+":"+f.TechID] = true
	}
	if !codes["tech_surface_gridbox_missing:MOD_other"] {
		t.Fatalf("findings=%v", codes)
	}
}

func TestValidateRejectsUntrackedOrExternallyLinkedBranchTechnology(t *testing.T) {
	manifest, restore := testProfile(t)
	defer restore()
	manifest.Branches[0].TechnologyIDs = []string{"MOD_rifle", "MOD_rifle_2"}
	techs := []Technology{
		{ID: "MOD_rifle", Source: "created", Folder: "infantry_folder", PathTargets: []string{"MOD_rifle_2"}},
		{ID: "MOD_rifle_2", Source: "created", Folder: "infantry_folder", PathTargets: []string{"MOD_vanilla"}},
		{ID: "MOD_vanilla", Source: "imported", Folder: "infantry_folder"},
		{ID: "MOD_untracked", Source: "created", Folder: "infantry_folder", PathTargets: []string{"MOD_rifle"}},
	}
	codes := map[string]bool{}
	for _, f := range Validate(manifest, techs, false) {
		codes[f.Code] = true
	}
	if !codes["tech_surface_branch_path_external"] || !codes["tech_surface_branch_member_missing"] {
		t.Fatalf("findings=%v", codes)
	}
}

func TestValidateRequiresRootAsFirstBranchMember(t *testing.T) {
	manifest, restore := testProfile(t)
	defer restore()
	manifest.Branches[0].TechnologyIDs = []string{"MOD_rifle_2", "MOD_rifle"}
	techs := []Technology{
		{ID: "MOD_rifle", Source: "created", Folder: "infantry_folder", PathTargets: []string{"MOD_rifle_2"}},
		{ID: "MOD_rifle_2", Source: "created", Folder: "infantry_folder"},
	}
	codes := map[string]bool{}
	for _, f := range Validate(manifest, techs, false) {
		codes[f.Code] = true
	}
	if !codes["tech_surface_root_not_first_member"] {
		t.Fatalf("findings=%v", codes)
	}
}
