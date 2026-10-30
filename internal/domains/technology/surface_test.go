package technology

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/hoi4/techsurface"
	"haganeworks/internal/hoi4/techsurfaceprofiles"
	"haganeworks/internal/project"
)

func seedGUISource(t *testing.T, p *project.Project, provenance map[string]any) {
	t.Helper()
	if provenance == nil {
		provenance = map[string]any{"technology_surface": map[string]any{
			"schema_version": 1, "source": jsonValue(techsurfaceprofiles.AuditedSourceFingerprint()), "dlc_confirmed": false,
		}}
	}
	err := p.Write([]string{assetsTable}, func(tx *project.Tx) error {
		tx.Insert(assetsTable, project.Row{"asset_type": "gui_source", "gfx_name": "snapshot.zip",
			"obs_key": "local/p/u1/gui-source/snapshot.zip", "provenance": jsonValue(provenance)})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func surfaceStatusOf(t *testing.T, mux http.Handler) TechSurfaceStatus {
	t.Helper()
	rec := call(t, mux, "GET", base+"/surface", nil)
	expect(t, rec, http.StatusOK)
	return decode[TechSurfaceStatus](t, rec)
}

func TestSurfaceStatusWithoutSource(t *testing.T) {
	mux := newMux(newTestProject(t), nil)
	rec := call(t, mux, "GET", base+"/surface", nil)
	expect(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), `"branches":[]`) {
		t.Fatalf("branches must be an array: %s", rec.Body.String())
	}
	status := decode[TechSurfaceStatus](t, rec)
	if status.Enabled || status.BaselineAttached || status.Environment.State != TechSurfaceEnvironmentBaselineMissing ||
		status.Environment.SourceConnection.State != TechnologySourceNotConnected || status.ActiveProfile != nil {
		t.Fatalf("status=%+v", status)
	}
}

func readyProject(t *testing.T) (*project.Project, http.Handler) {
	t.Helper()
	p := newTestProject(t)
	seedGUISource(t, p, nil)
	mux := newMux(p, nil)
	status := surfaceStatusOf(t, mux)
	if status.Environment.State != TechSurfaceEnvironmentDLCUnconfirmed || !status.BaselineAttached ||
		status.Environment.SourceConnection.Kind != TechnologySourceKindProjectMod || status.Environment.SourceConnection.Label != "This project's imported source" {
		t.Fatalf("unconfirmed status=%+v", status)
	}
	expect(t, call(t, mux, "PUT", base+"/surface/environment", map[string]any{"dlc": map[string]any{}, "folder": "x"}), http.StatusUnprocessableEntity)
	missingDLC := call(t, mux, "PUT", base+"/surface/environment", map[string]any{"dlc": map[string]any{}, "compatibility_mode": "requires_dlc"})
	expect(t, missingDLC, http.StatusUnprocessableEntity)
	if !strings.Contains(missingDLC.Body.String(), "Select at least one required DLC") {
		t.Fatalf("requires_dlc=%s", missingDLC.Body.String())
	}
	confirmed := call(t, mux, "PUT", base+"/surface/environment", map[string]any{"dlc": map[string]any{}, "compatibility_mode": "all_players"})
	expect(t, confirmed, http.StatusOK)
	status = decode[TechSurfaceStatus](t, confirmed)
	if status.Environment.State != TechSurfaceEnvironmentReady || status.ActiveProfile == nil || !status.Environment.DLCConfirmed {
		t.Fatalf("ready status=%+v", status)
	}
	return p, mux
}

func createInfantryBranch(t *testing.T, mux http.Handler) TechSurfaceBranchResult {
	t.Helper()
	rec := call(t, mux, "POST", base+"/surface/branches", map[string]any{
		"id": "tst_infantry", "entry_id": "army", "target_id": "infantry",
		"root": map[string]any{"tech_id": "TST_ext_1", "name": "Extension I", "description_locs": map[string]string{"en": "First."}},
	})
	expect(t, rec, http.StatusCreated)
	return decode[TechSurfaceBranchResult](t, rec)
}

func TestSurfaceBranchLifecycle(t *testing.T) {
	p, mux := readyProject(t)

	bad := call(t, mux, "POST", base+"/surface/branches", map[string]any{
		"id": "x", "entry_id": "army", "target_id": "infantry",
		"root": map[string]any{"tech_id": "X", "folder": "infantry_folder", "description_locs": map[string]string{"en": "d"}},
	})
	expect(t, bad, http.StatusUnprocessableEntity)

	created := createInfantryBranch(t, mux)
	if created.Root.TechID != "TST_ext_1" || created.Root.StartYear != 1936 || created.Root.Folder != "infantry_folder" ||
		len(created.Root.FolderPlacements) != 1 || created.Root.FolderPlacements[0]["y"] != 2.0 || created.Branch.Slot == "" ||
		len(created.Manifest.Branches) != 1 {
		t.Fatalf("created=%+v", created)
	}
	child := call(t, mux, "POST", base+"/surface/branches/tst_infantry/technologies", map[string]any{
		"parent_technology_id": "TST_ext_1",
		"technology":           map[string]any{"tech_id": "TST_ext_2", "name": "Extension II", "desc_en": "Second."},
	})
	expect(t, child, http.StatusCreated)
	childResult := decode[TechSurfaceBranchTechnologyResult](t, child)
	if childResult.Technology.StartYear != 1939 || childResult.Technology.FolderPlacements[0]["y"] != 6.0 ||
		len(childResult.Branch.TechnologyIDs) != 2 {
		t.Fatalf("child=%+v", childResult)
	}
	root := findTechByID(mustRows(t, p, table), "TST_ext_1")
	if paths := techFromRow(root).Path; len(paths) != 1 || paths[0]["leads_to_tech"] != "TST_ext_2" {
		t.Fatalf("parent path=%v", paths)
	}
	expect(t, call(t, mux, "POST", base+"/surface/branches/missing/technologies", map[string]any{
		"parent_technology_id": "TST_ext_1", "technology": map[string]any{"tech_id": "Z", "desc_en": "z"},
	}), http.StatusUnprocessableEntity)

	status := surfaceStatusOf(t, mux)
	if !status.Enabled || status.LegacyTimelineRepairNeeded || len(status.Manifest.Branches) != 1 {
		t.Fatalf("enabled status=%+v", status)
	}

	expect(t, call(t, mux, "POST", base, map[string]any{"tech_id": "free", "tech_line": "infantry"}), http.StatusUnprocessableEntity)
	rootID := itoa(created.Root.ID)
	expect(t, call(t, mux, "PUT", base+"/"+rootID, map[string]any{"position_x": 3}), http.StatusUnprocessableEntity)
	expect(t, call(t, mux, "PUT", base+"/"+rootID, map[string]any{"name_zh": "扩展"}), http.StatusOK)
	expect(t, call(t, mux, "DELETE", base+"/"+rootID, nil), http.StatusUnprocessableEntity)

	list := decode[[]map[string]any](t, call(t, mux, "GET", base+"/surface/branches", nil))
	if len(list) != 1 || list[0]["id"] != "tst_infantry" || list[0]["gridbox_id"] != "TST_ext_1_tree" {
		t.Fatalf("branches=%v", list)
	}
	missing := call(t, mux, "GET", base+"/surface/branches/nope", nil)
	expect(t, missing, http.StatusNotFound)
	if !strings.Contains(missing.Body.String(), `"code":"tech_branch_not_found"`) {
		t.Fatalf("missing=%s", missing.Body.String())
	}
	renamed := call(t, mux, "PUT", base+"/surface/branches/tst_infantry", map[string]any{"name": "Renamed"})
	expect(t, renamed, http.StatusOK)
	if decode[map[string]any](t, renamed)["name"] != "Renamed" {
		t.Fatalf("renamed=%s", renamed.Body.String())
	}
	readOnly := call(t, mux, "PUT", base+"/surface/branches/tst_infantry", map[string]any{"folder": "x"})
	expect(t, readOnly, http.StatusUnprocessableEntity)
	if !strings.Contains(readOnly.Body.String(), "tech_branch_field_profile_owned") {
		t.Fatalf("read-only=%s", readOnly.Body.String())
	}
	blocked := call(t, mux, "DELETE", base+"/surface/branches/tst_infantry", nil)
	expect(t, blocked, http.StatusConflict)
	if !strings.Contains(blocked.Body.String(), `"kind":"child_technology"`) {
		t.Fatalf("blocked=%s", blocked.Body.String())
	}

	err := p.Write([]string{table}, func(tx *project.Tx) error {
		row := findTechByID(tx.Rows(table), "TST_ext_2")
		_, err := tx.Update(table, project.Int64(row["id"]), func(row project.Row) {
			row["start_year"] = int64(1950)
			row["folder_placements"] = []any{map[string]any{"folder": "infantry_folder", "x": 9.0, "y": 9.0}}
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if !surfaceStatusOf(t, mux).LegacyTimelineRepairNeeded {
		t.Fatal("drifted timeline not reported")
	}
	repaired := call(t, mux, "POST", base+"/surface/repair", map[string]any{})
	expect(t, repaired, http.StatusOK)
	if result := decode[TechSurfaceRepairResult](t, repaired); len(result.RepairedTechnologyIDs) != 2 {
		t.Fatalf("repair=%+v", result)
	}
	if surfaceStatusOf(t, mux).LegacyTimelineRepairNeeded {
		t.Fatal("repair left the timeline drifted")
	}
}

func TestSurfaceBranchDeleteClearsEmptyManifest(t *testing.T) {
	p, mux := readyProject(t)
	createInfantryBranch(t, mux)

	err := p.Write([]string{"events"}, func(tx *project.Tx) error {
		tx.Insert("events", project.Row{"event_id": "tst.1", "trigger": "has_tech = TST_ext_1"})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	blocked := call(t, mux, "DELETE", base+"/surface/branches/tst_infantry", nil)
	expect(t, blocked, http.StatusConflict)
	if !strings.Contains(blocked.Body.String(), `"kind":"external_reference"`) || !strings.Contains(blocked.Body.String(), `"source_id":"tst.1"`) {
		t.Fatalf("blocked=%s", blocked.Body.String())
	}
	if err := p.Write([]string{"events"}, func(tx *project.Tx) error {
		tx.Delete("events", func(project.Row) bool { return true })
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	expect(t, call(t, mux, "DELETE", base+"/surface/branches/tst_infantry", nil), http.StatusNoContent)
	if manifest, _ := p.Record()["tech_surface_manifest"].(map[string]any); len(manifest) != 0 {
		t.Fatalf("manifest after last delete=%v", manifest)
	}
	if len(mustRows(t, p, table)) != 0 {
		t.Fatal("root technology not removed")
	}
	if surfaceStatusOf(t, mux).Enabled {
		t.Fatal("surface still enabled")
	}
}

func TestSurfaceSourceConnection(t *testing.T) {
	p := newTestProject(t)
	mux := newMux(p, nil)
	expect(t, call(t, mux, "POST", base+"/surface/source", map[string]any{"label": "x"}), http.StatusUnprocessableEntity)
	seedGUISource(t, p, nil)
	expect(t, call(t, mux, "POST", base+"/surface/source", map[string]any{"label": "x", "extra": 1}), http.StatusUnprocessableEntity)
	rec := call(t, mux, "POST", base+"/surface/source", map[string]any{"label": "  My   HOI4 "})
	expect(t, rec, http.StatusOK)
	connection := decode[TechnologySourceConnection](t, rec)
	if connection.State != TechnologySourceConnected || connection.Kind != TechnologySourceKindLocalHOI4 || connection.Label != "My HOI4" ||
		connection.Fingerprint != technologySourceFingerprint(techsurfaceprofiles.AuditedSourceFingerprint()) {
		t.Fatalf("connection=%+v", connection)
	}
	if status := surfaceStatusOf(t, mux); status.Environment.SourceConnection.Kind != TechnologySourceKindLocalHOI4 {
		t.Fatalf("status connection=%+v", status.Environment.SourceConnection)
	}

	existing := call(t, mux, "POST", base+"/surface/source/certified", map[string]any{})
	expect(t, existing, http.StatusOK)
	if decode[TechnologySourceConnection](t, existing).Kind != TechnologySourceKindLocalHOI4 {
		t.Fatalf("existing=%s", existing.Body.String())
	}

	if _, err := p.UpdateRecord(func(r project.Row) { r["is_loaded_existing_mod"] = true }); err != nil {
		t.Fatal(err)
	}
	expect(t, call(t, mux, "POST", base+"/surface/source", map[string]any{"label": "x"}), http.StatusUnprocessableEntity)
}

func TestCertifiedSourceErrors(t *testing.T) {
	p := newTestProject(t)
	unconfigured := call(t, newMux(p, nil), "POST", base+"/surface/source/certified", map[string]any{})
	expect(t, unconfigured, http.StatusServiceUnavailable)
	if !strings.Contains(unconfigured.Body.String(), `"code":"certified_tech_source_unconfigured"`) {
		t.Fatalf("unconfigured=%s", unconfigured.Body.String())
	}

	drifted := fstest.MapFS{
		techsurfaceprofiles.CountryTechTreeGUIPath: {Data: []byte("guiTypes = {}")},
		techsurfaceprofiles.TechnologyTagsPath:     {Data: []byte("technology_folders = {}")},
	}
	mismatch := call(t, newMux(p, drifted), "POST", base+"/surface/source/certified", map[string]any{})
	expect(t, mismatch, http.StatusServiceUnavailable)
	var body struct {
		Detail map[string]any `json:"detail"`
	}
	_ = json.Unmarshal(mismatch.Body.Bytes(), &body)
	if body.Detail["code"] != ErrCodeCertifiedSourceVersionMismatch || body.Detail["path"] != techsurfaceprofiles.CountryTechTreeGUIPath ||
		body.Detail["expected_sha256"] != techsurfaceprofiles.CountryTechTreeGUISHA256 || body.Detail["server_sha256"] == nil {
		t.Fatalf("mismatch=%s", mismatch.Body.String())
	}

	drifted["launcher-settings.json"] = &fstest.MapFile{Data: []byte(`{"rawVersion":"1.19.2.0","version":"Operation Postern v1.19.2.0"}`)}
	content := call(t, newMux(p, drifted), "POST", base+"/surface/source/certified", map[string]any{})
	if !strings.Contains(content.Body.String(), ErrCodeCertifiedSourceContentMismatch) || !strings.Contains(content.Body.String(), `"server_version":"Operation Postern v1.19.2.0"`) {
		t.Fatalf("content mismatch=%s", content.Body.String())
	}

	missing := call(t, newMux(p, fstest.MapFS{}), "POST", base+"/surface/source/certified", map[string]any{})
	if !strings.Contains(missing.Body.String(), ErrCodeCertifiedSourceFileMissing) {
		t.Fatalf("missing=%s", missing.Body.String())
	}
	if rows := mustRows(t, p, assetsTable); len(rows) != 0 {
		t.Fatalf("failed attach wrote rows: %v", rows)
	}

	if _, err := p.UpdateRecord(func(r project.Row) { r["is_loaded_existing_mod"] = true }); err != nil {
		t.Fatal(err)
	}
	expect(t, call(t, newMux(p, drifted), "POST", base+"/surface/source/certified", map[string]any{}), http.StatusUnprocessableEntity)
}

func TestLoadCertifiedTechnologySourceBuildsArchive(t *testing.T) {
	game := fstest.MapFS{"a/b.txt": {Data: []byte("hello")}}
	expected := techsurfaceprofiles.SourceFingerprint{Files: []techsurfaceprofiles.SourceFile{{Path: "a/b.txt",
		SHA256: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"}}}
	archive, source, err := loadCertifiedTechnologySource(game, expected)
	if err != nil || len(archive) == 0 || len(source.Files) != 1 || source.Files[0].SHA256 != expected.Files[0].SHA256 {
		t.Fatalf("archive=%d source=%+v err=%v", len(archive), source, err)
	}
}

func TestCertifiedSourceFromRealGame(t *testing.T) {
	dir := os.Getenv("HOI4_GAME_DIR")
	if dir == "" {
		t.Skip("HOI4_GAME_DIR not set")
	}
	p := newTestProject(t)
	mux := newMux(p, os.DirFS(dir))
	rec := call(t, mux, "POST", base+"/surface/source/certified", map[string]any{})
	expect(t, rec, http.StatusOK)
	if connection := decode[TechnologySourceConnection](t, rec); connection.Kind != TechnologySourceKindPlatform {
		t.Fatalf("connection=%+v", connection)
	}
	row := attachedGUISource(mustRows(t, p, assetsTable))
	if row == nil || !p.Blobs().Exists(text(row, "obs_key", "")) {
		t.Fatalf("snapshot row=%v", row)
	}
	if status := surfaceStatusOf(t, mux); status.Environment.State != TechSurfaceEnvironmentDLCUnconfirmed {
		t.Fatalf("status=%+v", status.Environment)
	}
	icon := call(t, mux, "GET", "/api/go/vanilla/tech-icon/infantry_weapons", nil)
	expect(t, icon, http.StatusOK)
	known := decode[KnownTechnologyFoldersResponse](t, call(t, mux, "GET", base+"/known-folders", nil))
	if len(known.Vanilla) < 10 {
		t.Fatalf("vanilla folders=%v", known.Vanilla)
	}
	if ids := newGameData(func() (fs.FS, error) { return os.DirFS(dir), nil }).techIDs(); len(ids) < 500 || !ids["CAS1"] {
		t.Fatalf("tech ids=%d", len(ids))
	}
}

func mustRows(t *testing.T, p *project.Project, name string) []project.Row {
	t.Helper()
	rows, err := p.Rows(name)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func auditedBaseTechSurfaceProfile(t *testing.T) techsurfaceprofiles.Profile {
	t.Helper()
	profile, err := techsurfaceprofiles.Resolve(techsurfaceprofiles.NewEnvironment(
		techsurfaceprofiles.AuditedSourceFingerprint(), techsurfaceprofiles.DLCState{},
	))
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func assertSurfacePlacement(t *testing.T, placements []map[string]any, wantX, wantY int) {
	t.Helper()
	if len(placements) != 1 {
		t.Fatalf("placements=%#v", placements)
	}
	if got, ok := placements[0]["x"].(int); !ok || got != wantX {
		t.Fatalf("x=%#v want=%d", placements[0]["x"], wantX)
	}
	if got, ok := placements[0]["y"].(int); !ok || got != wantY {
		t.Fatalf("y=%#v want=%d", placements[0]["y"], wantY)
	}
}

func TestPrepareTechSurfaceBranchUsesCertifiedInfantryTimeline(t *testing.T) {
	name := "P0 Infantry Test I"
	candidate, branch, root, err := prepareTechSurfaceBranch(techsurface.Manifest{}, false, auditedBaseTechSurfaceProfile(t), TechSurfaceCompatibilityAllPlayers, TechSurfaceBranchCreateInput{
		ID: "p0_infantry_test", EntryID: "army", TargetID: "infantry",
		Root: CreateInput{TechID: "P0_infantry_extension_1", Name: &name, DescriptionLocs: map[string]string{"en": "First test technology."}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if root.StartYear == nil || *root.StartYear != 1936 {
		t.Fatalf("root year=%v", root.StartYear)
	}
	assertSurfacePlacement(t, root.FolderPlacements, 0, 2)
	childName := "P0 Infantry Test II"
	child, parentID, err := prepareTechSurfaceBranchTechnology(candidate, branch, TechSurfaceBranchTechnologyCreateInput{
		ParentTechnologyID: root.TechID,
		Technology:         CreateInput{TechID: "P0_infantry_extension_2", Name: &childName, DescriptionLocs: map[string]string{"en": "Second test technology."}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if parentID != root.TechID || child.StartYear == nil || *child.StartYear != 1939 {
		t.Fatalf("parent=%q child year=%v", parentID, child.StartYear)
	}
	assertSurfacePlacement(t, child.FolderPlacements, 0, 6)
}

func TestPrepareTechSurfaceBranchRequiresEnglishDescription(t *testing.T) {
	_, _, _, err := prepareTechSurfaceBranch(techsurface.Manifest{}, false, auditedBaseTechSurfaceProfile(t), TechSurfaceCompatibilityAllPlayers, TechSurfaceBranchCreateInput{
		ID: "p0_missing_description", EntryID: "army", TargetID: "infantry",
		Root: CreateInput{TechID: "P0_missing_description"},
	})
	if err == nil {
		t.Fatal("missing description was accepted")
	}
}

func TestPrepareTechSurfaceBranchFollowsPlayerCompatibilityChoice(t *testing.T) {
	name := "P0 Air Compatibility"
	input := TechSurfaceBranchCreateInput{
		ID: "p0_air_compatibility", EntryID: "air", TargetID: "airframes",
		Root: CreateInput{TechID: "P0_air_compatibility_1", Name: &name, DescriptionLocs: map[string]string{"en": "Compatibility test."}},
	}
	_, allPlayersBranch, allPlayersRoot, err := prepareTechSurfaceBranch(techsurface.Manifest{}, false, auditedBaseTechSurfaceProfile(t), TechSurfaceCompatibilityAllPlayers, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(allPlayersBranch.Placements) != 2 || len(allPlayersRoot.FolderPlacements) != 2 ||
		allPlayersBranch.Placements[0].Folder != "air_techs_folder" || allPlayersBranch.Placements[1].Folder != "bba_air_techs_folder" {
		t.Fatalf("all-player branch=%#v root placements=%#v", allPlayersBranch.Placements, allPlayersRoot.FolderPlacements)
	}
	_, requiredBranch, requiredRoot, err := prepareTechSurfaceBranch(techsurface.Manifest{}, false, auditedBaseTechSurfaceProfile(t), TechSurfaceCompatibilityRequiresDLC, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(requiredBranch.Placements) != 1 || len(requiredRoot.FolderPlacements) != 1 || requiredBranch.Placements[0].Folder != "air_techs_folder" {
		t.Fatalf("required-DLC branch=%#v root placements=%#v", requiredBranch.Placements, requiredRoot.FolderPlacements)
	}
}

func TestTechnologySurfaceTimelineAssignmentsUseNativeYearRows(t *testing.T) {
	profile, ok := techsurface.ProfileByID(auditedBaseTechSurfaceProfile(t).ID)
	if !ok {
		t.Fatal("missing P0 profile")
	}
	assignments, err := technologySurfaceTimelineAssignments(techsurface.Manifest{
		SchemaVersion: techsurface.SchemaVersion, Profile: profile.ID, BaselineSHA256: profile.BaselineSHA256,
		Branches: []techsurface.Branch{{
			ID: "p0_infantry_test", Surface: "army", Folder: "infantry_folder", Slot: "p0_infantry_extension_1",
			RootTechnologyID: "P0_infantry_extension_1", TechnologyIDs: []string{"P0_infantry_extension_1", "P0_infantry_extension_2"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 2 || assignments[0].Cell.StartYear != 1936 || assignments[0].Cell.Y != 2 ||
		assignments[1].Cell.StartYear != 1939 || assignments[1].Cell.Y != 6 {
		t.Fatalf("assignments=%#v", assignments)
	}
}

func TestDecodeTechnologySurfaceProvenancePreservesOtherAssetMetadata(t *testing.T) {
	root, provenance, err := decodeTechnologySurfaceProvenance(map[string]any{
		"other": "kept", "technology_surface": map[string]any{"dlc_confirmed": true, "compatibility_mode": "requires_dlc"},
	})
	if err != nil || root["other"] != "kept" || !provenance.DLCConfirmed || provenance.CompatibilityMode != "requires_dlc" {
		t.Fatalf("root=%v provenance=%+v err=%v", root, provenance, err)
	}
}

func TestTechnologySourceConnectionShapes(t *testing.T) {
	local := technologySourceConnection(technologySurfaceProvenance{
		SchemaVersion: 1, Source: techsurfaceprofiles.AuditedSourceFingerprint(),
		TechnologySource: &technologySourceMetadata{SchemaVersion: 1, Kind: TechnologySourceKindLocalHOI4, Label: "Hearts of Iron IV",
			Fingerprint: strings.Repeat("a", 64), Reused: true},
	}, false)
	if local.State != TechnologySourceConnected || local.Kind != TechnologySourceKindLocalHOI4 || !local.Reused {
		t.Fatalf("local=%#v", local)
	}
	raw, _ := json.Marshal(local)
	for _, forbidden := range []string{"obs_key", "project_uuid", "absolute_path"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("connection leaked %q: %s", forbidden, raw)
		}
	}
	if imported := technologySourceConnection(technologySurfaceProvenance{Source: techsurfaceprofiles.AuditedSourceFingerprint()}, true); imported.State != TechnologySourceProjectSource ||
		imported.Kind != TechnologySourceKindProjectMod || imported.Label != "This project's imported Mod" {
		t.Fatalf("imported=%#v", imported)
	}
	left := techsurfaceprofiles.AuditedSourceFingerprint()
	right := techsurfaceprofiles.SourceFingerprint{Files: []techsurfaceprofiles.SourceFile{left.Files[1], left.Files[0]}}
	if got, want := technologySourceFingerprint(left), technologySourceFingerprint(right); got != want || len(got) != 64 {
		t.Fatalf("fingerprints got=%q want=%q", got, want)
	}
	if got := normalizedTechnologySourceLabel("  Hearts   of Iron IV  "); got != "Hearts of Iron IV" {
		t.Fatalf("label=%q", got)
	}
	if got := normalizedTechnologySourceLabel(strings.Repeat("x", 200)); len([]rune(got)) != 100 {
		t.Fatalf("label length=%d", len([]rune(got)))
	}
}

func TestTechnologySurfaceEnvironmentStatus(t *testing.T) {
	status, profile := technologySurfaceEnvironmentStatus(technologySurfaceProvenance{SchemaVersion: 1, Source: techsurfaceprofiles.AuditedSourceFingerprint()}, false)
	if status.State != TechSurfaceEnvironmentDLCUnconfirmed || profile != nil {
		t.Fatalf("unconfirmed status=%#v", status)
	}
	drifted := techsurfaceprofiles.AuditedSourceFingerprint()
	drifted.Files[0].SHA256 = strings.Repeat("0", 64)
	if status, profile := technologySurfaceEnvironmentStatus(technologySurfaceProvenance{SchemaVersion: 1, Source: drifted}, false); status.Code != techsurfaceprofiles.CodeSourceDrift || profile != nil {
		t.Fatalf("drift status=%#v", status)
	}
	dlc := techsurfaceprofiles.DLCState{ByBloodAlone: true, ManTheGuns: true}
	status, profile = technologySurfaceEnvironmentStatus(technologySurfaceProvenance{
		SchemaVersion: 1, Source: techsurfaceprofiles.AuditedSourceFingerprint(), DLCConfirmed: true,
		Environment: techsurfaceprofiles.NewEnvironment(techsurfaceprofiles.AuditedSourceFingerprint(), dlc),
	}, false)
	if status.State != TechSurfaceEnvironmentReady || profile == nil {
		t.Fatalf("ready status=%#v", status)
	}
	result := profileResult(*profile)
	if len(result.Entries) != 4 || result.Entries[3].ID != techsurfaceprofiles.EntryIndustryEquipment {
		t.Fatalf("entries=%#v", result.Entries)
	}
	encoded, _ := json.Marshal(result)
	for _, forbidden := range []string{`"gridbox_format"`, `"slot_size"`, `"slots"`, `"x"`, `"y"`} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("profile response leaked compiler geometry %s: %s", forbidden, encoded)
		}
	}
}
