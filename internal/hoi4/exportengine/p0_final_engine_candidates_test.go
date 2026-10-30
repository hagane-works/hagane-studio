package exportengine

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/ideologysource"
	"haganeworks/internal/hoi4/techsurface"
)

const p0FinalBaseEquipmentID = "TST_p0_acceptance_rifle"

func TestP0FinalStrictBaseTechnologyEquipmentZIP(t *testing.T) {
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

	candidate := p0aBaseCandidate()
	candidate.ModName = "Hagane P0 Final Base Technology Equipment"
	snapshot := p0aTechSnapshot(t, candidate, baseline)
	var rootTechnology map[string]any
	if err := json.Unmarshal(snapshot.Tables["technologies"][0], &rootTechnology); err != nil {
		t.Fatal(err)
	}
	rootTechnology["enable_equipments"] = []string{p0FinalBaseEquipmentID}
	snapshot.Tables["technologies"][0] = mustJSON(t, rootTechnology)
	snapshot.Tables["equipments"] = []json.RawMessage{mustJSON(t, map[string]any{
		"id": 601, "equipment_id": p0FinalBaseEquipmentID,
		"name_en": "P0 Acceptance Rifle", "name_zh": "P0 验收步枪",
		"desc_en": "Unlocked by P0 Base Infantry I.",
		"desc_zh": "由 P0 基础步兵 I 解锁。",
		"source":  "created", "country_tag": "TST", "year": 1936,
		"archetype": "infantry_equipment", "parent": "infantry_equipment_0",
		"is_archetype": false, "is_buildable": true,
		"soft_attack": 1.0, "defense": 1.0, "build_cost_ic": 0.1,
	})}
	snapshot.Fingerprint, err = SnapshotFingerprint(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	restore := techsurface.RegisterProfileForTest(p0aTechProfile(candidate, baseline))
	defer restore()
	archive := p0aGUISnapshotZIP(t, baseline)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, _ int64) ([]byte, error) {
		if key != "p0a-tech-gui-source.zip" {
			t.Fatalf("unexpected blob key %q", key)
		}
		return archive, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	root := artifactRoot(artifact)
	byPath := make(map[string][]byte, len(artifact.Entries))
	for _, entry := range artifact.Entries {
		byPath[entry.Path] = entry.Bytes
		lower := strings.ToLower(filepath.ToSlash(entry.Path))
		for _, forbidden := range []string{"/common/ideologies/", "/common/characters/", "/history/countries/"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("politics content leaked into strict Base candidate: %s", entry.Path)
			}
		}
	}
	technologyPath := root + "/common/technologies/TST_p0a_base_infantry.txt"
	navalTechnologyPath := root + "/common/technologies/TST_p0a_base_naval.txt"
	equipmentPath := root + "/common/units/equipment/zz_TST_equipment.txt"
	for path, needle := range map[string]string{
		technologyPath:      "enable_equipments = {\n\t\t\t" + p0FinalBaseEquipmentID,
		navalTechnologyPath: "position = { x = 0 y = 7 }",
		equipmentPath:       "archetype = infantry_equipment",
		root + "/interface/countrytechtreeview.gui":                                  "p0a_base_electronics_1_tree",
		root + "/localisation/english/replace/TST_equipment_l_english.yml":           `TST_p0_acceptance_rifle_desc: "Unlocked by P0 Base Infantry I."`,
		root + "/localisation/simp_chinese/replace/TST_equipment_l_simp_chinese.yml": `TST_p0_acceptance_rifle_desc: "由 P0 基础步兵 I 解锁。"`,
	} {
		if !bytes.Contains(byPath[path], []byte(needle)) {
			t.Fatalf("strict Base artifact %s missing %q:\n%s", path, needle, byPath[path])
		}
	}
	for _, expected := range []string{"position = { x = 0 y = 7 }", "position = { x = 0 y = 13 }"} {
		if !bytes.Contains(byPath[navalTechnologyPath], []byte(expected)) {
			t.Fatalf("strict Base naval artifact missing engine-verified gridbox-local cell %q:\n%s", expected, byPath[navalTechnologyPath])
		}
	}
	p0aParseClausewitz(t, "strict Base technologies", byPath[technologyPath])
	p0aParseClausewitz(t, "strict Base naval technologies", byPath[navalTechnologyPath])
	p0aParseClausewitz(t, "strict Base equipment", byPath[equipmentPath])
	p0FinalWriteCandidate(t, "P0_FINAL_BASE_TECH_EQUIPMENT_ZIP_OUT", "P0_FINAL_BASE_TECH_EQUIPMENT.zip", snapshot.Fingerprint, artifact.Entries)
}

func TestP0FinalInheritedPoliticsVisibleZIP(t *testing.T) {
	tables, err := testWorldTables(t)
	if err != nil {
		t.Fatal(err)
	}
	layers := []ideologysource.Layer{{
		SourceKind: ideologysource.SourceVanilla,
		Files:      []ideologysource.SourceFile{{Path: ideologySourceNativePath, Bytes: []byte(tables.IdeologyBaseText)}},
	}}
	discovery, err := ideologysource.Discover(layers)
	if err != nil {
		t.Fatal(err)
	}
	var owner ideologysource.RootOwner
	for _, candidate := range discovery.Roots {
		if candidate.RootID == "democratic" {
			owner = candidate
			break
		}
	}
	if owner.RootID == "" {
		t.Fatal("bundled vanilla snapshot has no democratic ROOT")
	}
	manifest := ideologysource.Manifest{SchemaVersion: ideologysource.SchemaVersion, Entries: []ideologysource.ManifestEntry{{
		SourceKind: owner.SourceKind, OwnerPath: owner.OwnerPath,
		FileSHA256: owner.FileSHA256, RootSHA256: owner.RootSHA256,
		ScriptID: "pc01_social_liberalism", ParentRootID: owner.RootID,
	}}}
	project := ideologySourceNativeProject(t, manifest, []map[string]any{{
		"id": "pc01_social_liberalism", "group": "democratic", "source": "project_managed",
		"name": map[string]string{"en": "PC-01 Social Liberalism", "zh": "PC-01 社会自由主义"},
	}})
	var decoded map[string]any
	if err := json.Unmarshal(project, &decoded); err != nil {
		t.Fatal(err)
	}
	decoded["id"], decoded["name"], decoded["country_tag"] = 602, "P0 Final Inherited Politics Visible", "GER"
	decoded["export_languages"] = []string{"en", "zh"}
	project = mustJSON(t, decoded)

	event := `add_namespace = p0_inherited_acceptance

country_event = {
	id = p0_inherited_acceptance.1
	title = p0_inherited_acceptance.1.t
	desc = p0_inherited_acceptance.1.d
	picture = GFX_report_event_election_vote
	is_triggered_only = yes

	option = {
		name = p0_inherited_acceptance.1.a
		create_country_leader = {
			name = "P0 Social Liberal Test Leader"
			desc = P0_INHERITED_LEADER_DESC
			picture = GFX_portrait_Alf_Landon
			expire = "1965.1.1"
			ideology = pc01_social_liberalism
			traits = { }
		}
		set_party_name = {
			ideology = democratic
			name = P0_INHERITED_PARTY_NAME
			long_name = P0_INHERITED_PARTY_LONG_NAME
		}
		set_popularities = {
			democratic = 100
			communism = 0
			fascism = 0
			neutrality = 0
		}
		set_politics = {
			ruling_party = democratic
			elections_allowed = yes
		}
	}
}
`
	english := "\uFEFFl_english:\n p0_inherited_acceptance.1.t: \"P0 Inherited Ideology Acceptance\"\n p0_inherited_acceptance.1.d: \"This event creates a leader bound to the inherited democratic subtype.\"\n p0_inherited_acceptance.1.a: \"Install the PC-01 Social Liberal test leader\"\n P0_INHERITED_LEADER_DESC: \"A visible engine acceptance leader for the inherited subtype.\"\n P0_INHERITED_PARTY_NAME: \"PC-01 Social Liberals\"\n P0_INHERITED_PARTY_LONG_NAME: \"PC-01 Social Liberal Acceptance Party\"\n"
	chinese := "\uFEFFl_simp_chinese:\n p0_inherited_acceptance.1.t: \"P0 继承意识形态验收\"\n p0_inherited_acceptance.1.d: \"该事件会创建一名绑定至原版民主 ROOT 新增 subtype 的测试领袖。\"\n p0_inherited_acceptance.1.a: \"启用 PC-01 社会自由主义测试领袖\"\n P0_INHERITED_LEADER_DESC: \"用于验证继承 subtype 的可见测试领袖。\"\n P0_INHERITED_PARTY_NAME: \"PC-01 社会自由党\"\n P0_INHERITED_PARTY_LONG_NAME: \"PC-01 社会自由主义验收党\"\n"
	snapshot := Snapshot{
		Schema: SnapshotSchema, ProjectID: 602,
		Fingerprint: ideologysource.Hash([]byte(tables.IdeologyBaseText)),
		Project:     project,
		Tables: map[string][]json.RawMessage{
			"project_raw_files": {
				mustJSON(t, map[string]any{"id": 603, "dest_path": "events/p0_inherited_acceptance.txt", "raw_kind": "verbatim_override", "is_binary": false, "content": event, "source": "created"}),
				mustJSON(t, map[string]any{"id": 604, "dest_path": "localisation/english/replace/p0_inherited_acceptance_l_english.yml", "raw_kind": "verbatim_override", "is_binary": false, "content": english, "source": "created"}),
				mustJSON(t, map[string]any{"id": 605, "dest_path": "localisation/simp_chinese/replace/p0_inherited_acceptance_l_simp_chinese.yml", "raw_kind": "verbatim_override", "is_binary": false, "content": chinese, "source": "created"}),
			},
		},
	}
	snapshot.Fingerprint, err = SnapshotFingerprint(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	root := artifactRoot(artifact)
	byPath := make(map[string][]byte, len(artifact.Entries))
	for _, entry := range artifact.Entries {
		byPath[entry.Path] = entry.Bytes
	}
	ownerPath := root + "/" + owner.OwnerPath
	if !bytes.Contains(byPath[ownerPath], []byte("pc01_social_liberalism = {")) || !bytes.Contains(byPath[ownerPath], []byte("conservatism = {")) {
		t.Fatalf("inherited ROOT was not preserved and patched at %s", ownerPath)
	}
	if strings.Contains(string(byPath[root+".mod"]), `replace_path="common/ideologies"`) {
		t.Fatal("inherited subtype candidate incorrectly requested ideology directory takeover")
	}
	for path, needle := range map[string]string{
		root + "/events/p0_inherited_acceptance.txt":                                           "ideology = pc01_social_liberalism",
		root + "/localisation/english/replace/p0_inherited_acceptance_l_english.yml":           "PC-01 Social Liberal test leader",
		root + "/localisation/simp_chinese/replace/p0_inherited_acceptance_l_simp_chinese.yml": "PC-01 社会自由主义测试领袖",
	} {
		if !bytes.Contains(byPath[path], []byte(needle)) {
			t.Fatalf("visible inherited candidate %s missing %q", path, needle)
		}
	}
	p0aParseClausewitz(t, "visible inherited politics event", byPath[root+"/events/p0_inherited_acceptance.txt"])
	p0FinalWriteCandidate(t, "P0_FINAL_INHERITED_POLITICS_ZIP_OUT", "P0_FINAL_INHERITED_POLITICS_VISIBLE.zip", snapshot.Fingerprint, artifact.Entries)
}

func p0FinalWriteCandidate(t *testing.T, outputEnv, fallback, fingerprint string, entries []Entry) Manifest {
	t.Helper()
	destination := os.Getenv(outputEnv)
	if destination == "" {
		destination = filepath.Join(t.TempDir(), fallback)
	}
	first, err := Build(destination, fingerprint, entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatalf("write final P0 candidate: %v", err)
	}
	readback := p0aZIPEntries(t, destination)
	if len(readback) != len(entries)+1 {
		t.Fatalf("final P0 ZIP entries=%d want=%d", len(readback), len(entries)+1)
	}
	for _, entry := range entries {
		if !bytes.Equal(readback[entry.Path], entry.Bytes) {
			t.Fatalf("final P0 ZIP readback differs at %s", entry.Path)
		}
	}
	duplicate := filepath.Join(t.TempDir(), "determinism.zip")
	second, err := Build(duplicate, fingerprint, entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatalf("write deterministic comparison: %v", err)
	}
	if first.ZIPSHA256 != second.ZIPSHA256 || first.ZIPBytes != second.ZIPBytes {
		t.Fatalf("final P0 ZIP is not deterministic: first=%+v second=%+v", first, second)
	}
	t.Logf("P0_FINAL_ZIP path=%s bytes=%d sha256=%s entries=%d", destination, first.ZIPBytes, first.ZIPSHA256, len(readback))
	return first
}
