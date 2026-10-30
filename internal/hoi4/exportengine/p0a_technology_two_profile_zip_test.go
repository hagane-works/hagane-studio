package exportengine

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/clausewitz"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/hoi4/techsurface"
)

type p0aTechBranchFixture struct {
	Label         string
	Surface       string
	Folder        string
	TechLine      string
	Slot          techsurface.Slot
	GridboxFormat string
	SlotWidth     int
	SlotHeight    int
	Timeline      []techsurface.TimelineCell
	RootID        string
	ChildID       string
	RootNameEN    string
	RootNameZH    string
	ChildNameEN   string
	ChildNameZH   string
}

type p0aTechCandidate struct {
	ProfileID string
	ModName   string
	OutputEnv string
	Branches  []p0aTechBranchFixture
}

func p0aBaseCandidate() p0aTechCandidate {
	return p0aTechCandidate{
		ProfileID: "p0a_engine_base_no_dlc_fec641fb",
		ModName:   "Hagane P0A Technology Base No DLC Batch",
		OutputEnv: "P0A_TECH_BASE_ZIP_OUT",
		Branches: []p0aTechBranchFixture{
			{
				Label: "Infantry", Surface: "army", Folder: "infantry_folder", TechLine: "p0a_base_infantry",
				Slot: techsurface.Slot{ID: "p0a_base_infantry_slot", X: 140, Y: 1225}, GridboxFormat: "LEFT", SlotWidth: 70, SlotHeight: 70,
				Timeline: []techsurface.TimelineCell{{StartYear: 1936, X: 0, Y: 2}, {StartYear: 1939, X: 0, Y: 6}},
				RootID:   "p0a_base_infantry_1", ChildID: "p0a_base_infantry_2", RootNameEN: "P0 Base Infantry I", RootNameZH: "P0 基础步兵 I", ChildNameEN: "P0 Base Infantry II", ChildNameZH: "P0 基础步兵 II",
			},
			{
				Label: "Air", Surface: "air", Folder: "air_techs_folder", TechLine: "p0a_base_air",
				Slot: techsurface.Slot{ID: "p0a_base_air_slot", X: 2240, Y: 32}, GridboxFormat: "UP", SlotWidth: 60, SlotHeight: 70,
				Timeline: []techsurface.TimelineCell{{StartYear: 1933, X: 0, Y: 2}, {StartYear: 1940, X: 0, Y: 6}},
				RootID:   "p0a_base_air_1", ChildID: "p0a_base_air_2", RootNameEN: "P0 Base Air I", RootNameZH: "P0 基础空军 I", ChildNameEN: "P0 Base Air II", ChildNameZH: "P0 基础空军 II",
			},
			{
				Label: "Naval", Surface: "navy", Folder: "naval_folder", TechLine: "p0a_base_naval",
				Slot: techsurface.Slot{ID: "p0a_base_naval_slot", X: 150, Y: 400}, GridboxFormat: "LEFT", SlotWidth: 70, SlotHeight: 70,
				Timeline: []techsurface.TimelineCell{{StartYear: 1936, X: 0, Y: 7}, {StartYear: 1940, X: 0, Y: 13}},
				RootID:   "p0a_base_naval_1", ChildID: "p0a_base_naval_2", RootNameEN: "P0 Base Naval I", RootNameZH: "P0 基础海军 I", ChildNameEN: "P0 Base Naval II", ChildNameZH: "P0 基础海军 II",
			},
			{
				Label: "Industry", Surface: "civilian", Folder: "industry_folder", TechLine: "p0a_base_industry",
				Slot: techsurface.Slot{ID: "p0a_base_industry_slot", X: 1350, Y: 172}, GridboxFormat: "UP", SlotWidth: 70, SlotHeight: 70,
				Timeline: []techsurface.TimelineCell{{StartYear: 1936, X: 0, Y: 0}, {StartYear: 1937, X: 0, Y: 4}},
				RootID:   "p0a_base_industry_1", ChildID: "p0a_base_industry_2", RootNameEN: "P0 Base Industry I", RootNameZH: "P0 基础工业 I", ChildNameEN: "P0 Base Industry II", ChildNameZH: "P0 基础工业 II",
			},
			{
				Label: "Electronics", Surface: "civilian", Folder: "electronics_folder", TechLine: "p0a_base_electronics",
				Slot: techsurface.Slot{ID: "p0a_base_electronics_slot", X: 1550, Y: 172}, GridboxFormat: "UP", SlotWidth: 60, SlotHeight: 60,
				Timeline: []techsurface.TimelineCell{{StartYear: 1936, X: -1, Y: 0}, {StartYear: 1938, X: -1, Y: 6}},
				RootID:   "p0a_base_electronics_1", ChildID: "p0a_base_electronics_2", RootNameEN: "P0 Base Electronics I", RootNameZH: "P0 基础电子 I", ChildNameEN: "P0 Base Electronics II", ChildNameZH: "P0 基础电子 II",
			},
		},
	}
}

func p0aDLCCandidate() p0aTechCandidate {
	return p0aTechCandidate{
		ProfileID: "p0a_engine_all_dlc_fec641fb",
		ModName:   "Hagane P0A Technology All DLC Batch",
		OutputEnv: "P0A_TECH_DLC_ZIP_OUT",
		Branches: []p0aTechBranchFixture{
			{
				Label: "BBA Air", Surface: "air", Folder: "bba_air_techs_folder", TechLine: "p0a_dlc_air",
				Slot: techsurface.Slot{ID: "p0a_dlc_air_slot", X: 2100, Y: 32}, GridboxFormat: "UP", SlotWidth: 70, SlotHeight: 70,
				Timeline: []techsurface.TimelineCell{{StartYear: 1933, X: 0, Y: 2}, {StartYear: 1940, X: 0, Y: 6}},
				RootID:   "p0a_dlc_air_1", ChildID: "p0a_dlc_air_2", RootNameEN: "P0 DLC Air I", RootNameZH: "P0 DLC 空军 I", ChildNameEN: "P0 DLC Air II", ChildNameZH: "P0 DLC 空军 II",
			},
			{
				Label: "MTG Naval Main", Surface: "navy", Folder: "mtgnavalfolder", TechLine: "p0a_dlc_naval_main",
				Slot: techsurface.Slot{ID: "p0a_dlc_naval_main_slot", X: 70, Y: 1160}, GridboxFormat: "LEFT", SlotWidth: 70, SlotHeight: 70,
				Timeline: []techsurface.TimelineCell{{StartYear: 1936, X: 0, Y: 8}, {StartYear: 1939, X: 0, Y: 14}},
				RootID:   "p0a_dlc_naval_main_1", ChildID: "p0a_dlc_naval_main_2", RootNameEN: "P0 DLC Naval Main I", RootNameZH: "P0 DLC 海军主体 I", ChildNameEN: "P0 DLC Naval Main II", ChildNameZH: "P0 DLC 海军主体 II",
			},
			{
				Label: "MTG Naval Support", Surface: "navy", Folder: "mtgnavalsupportfolder", TechLine: "p0a_dlc_naval_support",
				Slot: techsurface.Slot{ID: "p0a_dlc_naval_support_slot", X: 150, Y: 1170}, GridboxFormat: "LEFT", SlotWidth: 70, SlotHeight: 70,
				Timeline: []techsurface.TimelineCell{{StartYear: 1922, X: 0, Y: 1}, {StartYear: 1936, X: 0, Y: 5}},
				RootID:   "p0a_dlc_naval_support_1", ChildID: "p0a_dlc_naval_support_2", RootNameEN: "P0 DLC Naval Support I", RootNameZH: "P0 DLC 海军支援 I", ChildNameEN: "P0 DLC Naval Support II", ChildNameZH: "P0 DLC 海军支援 II",
			},
		},
	}
}

func p0aTechProfile(candidate p0aTechCandidate, baseline []byte) techsurface.Profile {
	folders := make(map[string]techsurface.FolderProfile, len(candidate.Branches))
	for _, branch := range candidate.Branches {
		folders[branch.Folder] = techsurface.FolderProfile{
			ID: branch.Folder, Surface: branch.Surface, TechLine: branch.TechLine,
			Slots: map[string]techsurface.Slot{branch.Slot.ID: branch.Slot}, Timeline: branch.Timeline,
			GridboxFormat: branch.GridboxFormat, SlotWidth: branch.SlotWidth, SlotHeight: branch.SlotHeight,
		}
	}
	return techsurface.Profile{ID: candidate.ProfileID, BaselineSHA256: techsurface.Hash(baseline), Folders: folders}
}

func p0aTechSnapshot(t *testing.T, candidate p0aTechCandidate, baseline []byte) Snapshot {
	t.Helper()
	profile := p0aTechProfile(candidate, baseline)
	branches := make([]techsurface.Branch, 0, len(candidate.Branches))
	rows := make([]json.RawMessage, 0, len(candidate.Branches)*2)
	for index, fixture := range candidate.Branches {
		branches = append(branches, techsurface.Branch{
			ID: strings.ReplaceAll(strings.ToLower(fixture.Label), " ", "_") + "_branch", Surface: fixture.Surface,
			Folder: fixture.Folder, Slot: fixture.Slot.ID, RootTechnologyID: fixture.RootID,
			TechnologyIDs: []string{fixture.RootID, fixture.ChildID},
		})
		for techIndex, technology := range []struct {
			ID, NameEN, NameZH string
		}{
			{fixture.RootID, fixture.RootNameEN, fixture.RootNameZH},
			{fixture.ChildID, fixture.ChildNameEN, fixture.ChildNameZH},
		} {
			cell := fixture.Timeline[techIndex]
			row := map[string]any{
				"id": index*2 + techIndex + 1, "tech_id": technology.ID, "tech_line": fixture.TechLine,
				"source": "created", "folder": fixture.Folder, "start_year": cell.StartYear,
				"folder_placements": []any{map[string]any{"folder": fixture.Folder, "x": cell.X, "y": cell.Y}},
				"research_cost":     0.1, "ai_will_do": "factor = 0", "categories": []any{},
				"modifiers": map[string]any{"__modifier": "research_speed_factor = 0.001"},
				"name_en":   technology.NameEN, "name_zh": technology.NameZH,
				"desc_en": technology.NameEN + " engine acceptance node.",
				"desc_zh": technology.NameZH + " 实机验收节点。",
			}
			if techIndex == 0 {
				row["path"] = []any{map[string]any{"leads_to_tech": fixture.ChildID, "research_cost_coeff": 1}}
			} else {
				row["path"] = []any{}
			}
			raw, err := json.Marshal(row)
			if err != nil {
				t.Fatalf("marshal %s: %v", technology.ID, err)
			}
			rows = append(rows, raw)
		}
	}
	manifest := techsurface.Manifest{
		SchemaVersion: techsurface.SchemaVersion, Profile: candidate.ProfileID,
		BaselineSHA256: profile.BaselineSHA256, Branches: branches,
	}
	project := map[string]any{
		"id": 501, "name": candidate.ModName, "country_tag": "TST", "version": "1.0",
		"supported_version": "1.19.*", "tags": []any{"Utilities"}, "export_languages": []any{"en", "zh"},
		"tech_surface_manifest": manifest,
	}
	projectRaw, err := json.Marshal(project)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{
		Schema: SnapshotSchema, ProjectID: 501, Project: projectRaw,
		Tables: map[string][]json.RawMessage{
			"technologies":       rows,
			"project_obs_assets": {json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"p0a-tech-gui-source.zip"}`)},
		},
	}
	snapshot.Fingerprint, err = SnapshotFingerprint(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func p0aGUISnapshotZIP(t *testing.T, baseline []byte) []byte {
	t.Helper()
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	w, err := zw.Create(techsurface.BaselinePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(baseline); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func p0aParseClausewitz(t *testing.T, label string, body []byte) {
	t.Helper()
	_, recoveries, err := paradox.Parse(string(body))
	if failed, detail := clausewitz.ParseErrorMessage(recoveries, err); failed {
		t.Fatalf("%s source readback failed: %s", label, detail)
	}
}

func p0aZIPEntries(t *testing.T, path string) map[string][]byte {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open P0-A ZIP: %v", err)
	}
	defer reader.Close()
	out := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open P0-A ZIP entry %s: %v", file.Name, err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read P0-A ZIP entry %s: %v", file.Name, err)
		}
		out[file.Name] = body
	}
	return out
}

func p0aBuildAndCheckCandidate(t *testing.T, candidate p0aTechCandidate, baseline []byte) {
	t.Helper()
	profile := p0aTechProfile(candidate, baseline)
	restore := techsurface.RegisterProfileForTest(profile)
	defer restore()
	snapshot := p0aTechSnapshot(t, candidate, baseline)
	guiArchive := p0aGUISnapshotZIP(t, baseline)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, maxBytes int64) ([]byte, error) {
		if key != "p0a-tech-gui-source.zip" {
			return nil, fmt.Errorf("unexpected P0-A blob key %q", key)
		}
		return guiArchive, nil
	})
	if err != nil {
		t.Fatalf("build native candidate: %v", err)
	}
	if !slices.Contains(artifact.Coverage, "technology_surface") {
		t.Fatalf("coverage=%v, missing technology_surface", artifact.Coverage)
	}
	root := artifactRoot(artifact)
	entryMap := make(map[string][]byte, len(artifact.Entries))
	for _, entry := range artifact.Entries {
		entryMap[entry.Path] = entry.Bytes
		lower := strings.ToLower(strings.ReplaceAll(entry.Path, "\\", "/"))
		for _, forbidden := range []string{"/common/ideologies/", "/common/characters/", "/history/countries/", "/gfx/flags/"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("politics content leaked into technology-only candidate: %s", entry.Path)
			}
		}
	}
	guiPath := root + "/" + techsurface.BaselinePath
	gui := entryMap[guiPath]
	if len(gui) == 0 {
		t.Fatalf("generated GUI missing at %s", guiPath)
	}
	p0aParseClausewitz(t, "countrytechtreeview.gui", gui)
	for _, fixture := range candidate.Branches {
		for _, want := range []string{
			`name = "` + fixture.RootID + `_tree"`,
			fmt.Sprintf("position = { x = %d y = %d }", fixture.Slot.X, fixture.Slot.Y),
			fmt.Sprintf("slotsize = { width = %d height = %d }", fixture.SlotWidth, fixture.SlotHeight),
			`format = "` + fixture.GridboxFormat + `"`,
		} {
			if !strings.Contains(string(gui), want) {
				t.Fatalf("%s GUI missing %q", fixture.Label, want)
			}
		}
		techPath := root + "/common/technologies/TST_" + fixture.TechLine + ".txt"
		body := entryMap[techPath]
		if len(body) == 0 {
			t.Fatalf("technology source missing at %s", techPath)
		}
		p0aParseClausewitz(t, fixture.Label+" technologies", body)
		for index, id := range []string{fixture.RootID, fixture.ChildID} {
			cell := fixture.Timeline[index]
			for _, want := range []string{id + " = {", "name = " + fixture.Folder, fmt.Sprintf("position = { x = %d y = %d }", cell.X, cell.Y)} {
				if !strings.Contains(string(body), want) {
					t.Fatalf("%s technology source missing %q:\n%s", fixture.Label, want, body)
				}
			}
		}
		if !strings.Contains(string(body), "leads_to_tech = "+fixture.ChildID) {
			t.Fatalf("%s visual path missing", fixture.Label)
		}
	}
	for _, descriptorPath := range []string{root + ".mod", root + "/descriptor.mod"} {
		descriptor := string(entryMap[descriptorPath])
		if descriptor == "" {
			t.Fatalf("descriptor missing at %s", descriptorPath)
		}
		if strings.Contains(descriptor, "replace_path") || strings.Contains(descriptor, "dependencies={") {
			t.Fatalf("technology candidate descriptor must be additive and standalone:\n%s", descriptor)
		}
	}
	english := string(entryMap[root+"/localisation/english/replace/TST_technology_l_english.yml"])
	chinese := string(entryMap[root+"/localisation/simp_chinese/replace/TST_technology_l_simp_chinese.yml"])
	if !bytes.HasPrefix([]byte(english), []byte("\xef\xbb\xbf")) || !bytes.HasPrefix([]byte(chinese), []byte("\xef\xbb\xbf")) {
		t.Fatal("P0-A technology localisation must be UTF-8 BOM")
	}
	for _, fixture := range candidate.Branches {
		for _, want := range []string{fixture.RootNameEN, fixture.ChildNameEN, fixture.RootID + "_desc", fixture.ChildID + "_desc"} {
			if !strings.Contains(english, want) {
				t.Fatalf("English localisation missing %q", want)
			}
		}
		for _, want := range []string{fixture.RootNameZH, fixture.ChildNameZH, fixture.RootID + "_desc", fixture.ChildID + "_desc"} {
			if !strings.Contains(chinese, want) {
				t.Fatalf("Chinese localisation missing %q", want)
			}
		}
	}
	destination := os.Getenv(candidate.OutputEnv)
	if destination == "" {
		destination = filepath.Join(t.TempDir(), root+".zip")
	}
	manifest, err := Build(destination, snapshot.Fingerprint, artifact.Entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatalf("write P0-A candidate ZIP: %v", err)
	}
	zipEntries := p0aZIPEntries(t, destination)
	if len(zipEntries) != len(artifact.Entries)+1 {
		t.Fatalf("ZIP entry count=%d, want %d", len(zipEntries), len(artifact.Entries)+1)
	}
	for _, entry := range artifact.Entries {
		if !bytes.Equal(zipEntries[entry.Path], entry.Bytes) {
			t.Fatalf("ZIP readback differs at %s", entry.Path)
		}
	}
	duplicate := filepath.Join(t.TempDir(), "determinism.zip")
	second, err := Build(duplicate, snapshot.Fingerprint, artifact.Entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatal(err)
	}
	if second.ZIPSHA256 != manifest.ZIPSHA256 || second.ZIPBytes != manifest.ZIPBytes {
		t.Fatalf("candidate ZIP is not deterministic: first=%+v second=%+v", manifest, second)
	}
	raw, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != manifest.ZIPSHA256 {
		t.Fatalf("returned ZIP hash=%s actual=%s", manifest.ZIPSHA256, got)
	}
	t.Logf("P0A_TECH_ZIP profile=%s path=%s bytes=%d sha256=%s fingerprint=%s entries=%d",
		candidate.ProfileID, destination, manifest.ZIPBytes, manifest.ZIPSHA256, snapshot.Fingerprint, len(zipEntries))
}

func TestP0ATechnologyTwoProfileZIPCandidates(t *testing.T) {
	baselinePath := os.Getenv("P0A_TECH_GUI_BASELINE")
	if baselinePath == "" {
		t.Skip("set P0A_TECH_GUI_BASELINE to the audited original countrytechtreeview.gui")
	}
	baseline, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read original GUI baseline: %v", err)
	}
	const auditedSHA256 = "fec641fb738ab59369cac40ae902fdba793bed7cbaa2c807756207a2e1f01045"
	if got := techsurface.Hash(baseline); got != auditedSHA256 {
		t.Fatalf("original GUI baseline hash=%s, want %s", got, auditedSHA256)
	}
	for _, candidate := range []p0aTechCandidate{p0aBaseCandidate(), p0aDLCCandidate()} {
		candidate := candidate
		t.Run(candidate.ProfileID, func(t *testing.T) {
			p0aBuildAndCheckCandidate(t, candidate, baseline)
		})
	}
}
