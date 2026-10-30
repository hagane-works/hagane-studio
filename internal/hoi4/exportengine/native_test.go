package exportengine

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"maps"
	"slices"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/techsurface"
)

func TestBuildNativeDescriptorFromSnapshot(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"测试: Demo","country_tag":"ger","version":"2.0","supported_version":"1.19.*","tags":["Alternative History"]}`),
	}
	artifact, err := BuildNativeDescriptor(snapshot, []string{"common/ideas"})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Filename != "Demo.zip" || len(artifact.Entries) != 2 {
		t.Fatalf("artifact=%#v", artifact)
	}
	if artifact.Entries[0].Path != "Demo.mod" || artifact.Entries[1].Path != "Demo/descriptor.mod" {
		t.Fatalf("paths=%q %q", artifact.Entries[0].Path, artifact.Entries[1].Path)
	}
	outer := string(artifact.Entries[0].Bytes)
	if !strings.Contains(outer, `name="测试: Demo"`) || !strings.Contains(outer, `replace_path="common/ideas"`) || !strings.HasSuffix(outer, `path="mod/Demo"`) {
		t.Fatalf("outer descriptor:\n%s", outer)
	}
}

func TestSanitizeModDirNameParity(t *testing.T) {
	cases := map[string]string{
		`a/b\c:d`:   "a_b_c_d",
		"name...  ": "name",
		"纯中文":       "mod_GER",
		"a__b":      "a_b",
	}
	for input, want := range cases {
		if got := SanitizeModDirName(input, "mod_GER"); got != want {
			t.Fatalf("SanitizeModDirName(%q)=%q want %q", input, got, want)
		}
	}
}

func TestBuildNativeShadowEmitsTechnologyScriptsAndFiltersImported(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","start_year":1936}`),
		Tables: map[string][]json.RawMessage{
			"technologies": {
				json.RawMessage(`{"id":1,"tech_id":"GER_created","tech_line":"industry","source":"created","source_file":"","name":"Created"}`),
				json.RawMessage(`{"id":2,"tech_id":"GER_imported","tech_line":"industry","source":"imported","source_file":"common/technologies/original.txt","name":"Imported"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(artifact.Entries) != 6 || artifact.Entries[2].Path != "Demo/common/technologies/GER_industry.txt" || artifact.Entries[5].Path != "Demo/"+LocFallbackLogName {
		t.Fatalf("entries=%#v", artifact.Entries)
	}
	if strings.Contains(string(artifact.Entries[2].Bytes), "GER_imported") || !strings.Contains(string(artifact.Entries[2].Bytes), "GER_created") {
		t.Fatalf("unexpected technology script:\n%s", artifact.Entries[2].Bytes)
	}
	if artifact.Entries[3].Path != "Demo/localisation/english/replace/GER_technology_l_english.yml" ||
		artifact.Entries[4].Path != "Demo/localisation/simp_chinese/replace/GER_technology_l_simp_chinese.yml" {
		t.Fatalf("localisation paths=%q %q", artifact.Entries[3].Path, artifact.Entries[4].Path)
	}
}

func TestBuildNativeShadowCompilesTechnologySurfaceFromSourceSnapshot(t *testing.T) {
	const gui = `guiTypes = {
	containerWindowType = {
		name = "countrytechtreeview"
		containerWindowType = {
			name = "infantry_folder"
			gridboxtype = { name = "infantry_weapons_tree" }
		}
	}
}
`
	const profileID = "native_test_infantry"
	restore := techsurface.RegisterProfileForTest(techsurface.Profile{
		ID: profileID, BaselineSHA256: techsurface.Hash([]byte(gui)),
		Folders: map[string]techsurface.FolderProfile{
			"infantry_folder": {ID: "infantry_folder", Surface: "army", TechLine: "infantry", Slots: map[string]techsurface.Slot{
				"slot": {ID: "slot", X: 140, Y: 1225},
			}, Timeline: []techsurface.TimelineCell{
				{StartYear: 1936, X: 0, Y: 2},
				{StartYear: 1939, X: 0, Y: 6},
			}},
		},
	})
	defer restore()
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	w, err := zw.Create("interface/countrytechtreeview.gui")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(gui)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	project := map[string]any{
		"id": 42, "name": "Demo", "country_tag": "GER",
		"tech_surface_manifest": map[string]any{
			"schema_version": 1, "profile": profileID,
			"baseline_sha256": techsurface.Hash([]byte(gui)),
			"branches": []any{map[string]any{
				"id": "rifle", "surface": "army", "folder": "infantry_folder", "slot": "slot", "root_technology_id": "MOD_rifle", "technology_ids": []string{"MOD_rifle", "MOD_rifle_2"},
			}},
		},
	}
	projectJSON, _ := json.Marshal(project)
	snapshot := Snapshot{ProjectID: 42, Project: projectJSON, Tables: map[string][]json.RawMessage{
		"technologies":       {json.RawMessage(`{"id":1,"tech_id":"MOD_rifle","tech_line":"infantry","source":"created","folder":"infantry_folder","start_year":1936,"folder_placements":[{"folder":"infantry_folder","x":0,"y":2}],"path":[{"leads_to_tech":"MOD_rifle_2"}]}`), json.RawMessage(`{"id":2,"tech_id":"MOD_rifle_2","tech_line":"infantry","source":"created","folder":"infantry_folder","start_year":1939,"folder_placements":[{"folder":"infantry_folder","x":0,"y":6}]}`)},
		"project_obs_assets": {json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"gui-source.zip"}`)},
	}}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, maxBytes int64) ([]byte, error) {
		if key != "gui-source.zip" {
			t.Fatalf("unexpected key %q", key)
		}
		return archive.Bytes(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]string{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = string(entry.Bytes)
	}
	got := entries["Demo/interface/countrytechtreeview.gui"]
	if !strings.Contains(got, `name = "MOD_rifle_tree"`) || !strings.Contains(got, `position = { x = 140 y = 1225 }`) {
		t.Fatalf("surface GUI missing gridbox:\n%s", got)
	}
	if !slices.Contains(artifact.Coverage, "technology_surface") {
		t.Fatalf("coverage=%v", artifact.Coverage)
	}
	if sha := sha256.Sum256([]byte(gui)); techsurface.Hash([]byte(gui)) != fmt.Sprintf("%x", sha) {
		t.Fatal("test hash mismatch")
	}
}

func TestBuildNativeShadowMaterializesTechnologyIcon(t *testing.T) {
	var source bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 0x7f
	}
	if err := png.Encode(&source, img); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"technologies":     {json.RawMessage(`{"id":7,"tech_id":"GER_radio","tech_line":"electronics","source":"created","name_en":"Radio"}`)},
			"entity_obs_paths": {json.RawMessage(`{"id":8,"project_id":42,"entity_type":"tech_icon","entity_id":7,"obs_key":"tech/radio.png"}`)},
		},
	}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, maxBytes int64) ([]byte, error) {
		if key != "tech/radio.png" || maxBytes != 32<<20 {
			t.Fatalf("resolver key=%q max=%d", key, maxBytes)
		}
		return source.Bytes(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = entry.Bytes
	}
	if !bytes.HasPrefix(entries["Demo/gfx/interface/technologies/ger_radio.dds"], []byte("DDS ")) {
		t.Fatal("technology DDS missing")
	}
	gfx := string(entries["Demo/interface/GER_technologies.gfx"])
	if !strings.Contains(gfx, "GFX_ger_radio_medium") || !strings.Contains(gfx, "technologies/ger_radio.dds") {
		t.Fatalf("technology gfx mismatch:\n%s", gfx)
	}
}

func TestBuildNativeShadowEmitsEquipmentAndSubUnitLocalisationWithoutTechnologies(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"equipments": {
				json.RawMessage(`{"id":1,"equipment_id":"GER_tank","name_en":"Tank","desc_en":"Armoured vehicle","source":"created"}`),
			},
			"sub_units": {
				json.RawMessage(`{"id":2,"sub_unit_id":"GER_infantry","name_en":"Infantry","description_en":"Line infantry","source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	if !strings.Contains(contents["Demo/localisation/english/replace/GER_equipment_l_english.yml"], "Tank") {
		t.Fatalf("equipment localisation missing: %#v", contents)
	}
	if !strings.Contains(contents["Demo/localisation/english/replace/GER_equipment_l_english.yml"], `GER_tank_desc: "Armoured vehicle"`) {
		t.Fatalf("equipment description localisation missing: %#v", contents)
	}
	if !strings.Contains(contents["Demo/localisation/english/replace/GER_sub_units_l_english.yml"], "Infantry") {
		t.Fatalf("sub-unit localisation missing: %#v", contents)
	}
	if !slices.Contains(artifact.Coverage, "equipment_localisation") || !slices.Contains(artifact.Coverage, "sub_unit_localisation") {
		t.Fatalf("coverage=%#v", artifact.Coverage)
	}
}

func TestBuildNativeShadowRebuildsNestedAirAndNavalOOB(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"oob_air_wing_groups": {json.RawMessage(`{"id":10,"oob_file":"GER_1936","location":"123","ordering":0,"layout":[],"source":"created"}`)},
			"oob_air_wings":       {json.RawMessage(`{"id":11,"group_id":10,"equipment_type":"fighter_equipment_1","name":"First Wing","owner":"GER","amount":"100","ordering":0,"body_ast":[],"source":"created"}`)},
			"oob_fleets":          {json.RawMessage(`{"id":20,"oob_file":"GER_1936","name":"Home Fleet","naval_base":"456","ordering":0,"layout":[],"source":"created"}`)},
			"oob_task_forces":     {json.RawMessage(`{"id":21,"fleet_id":20,"name":"First Task Force","location":"456","ordering":0,"layout":[],"source":"created"}`)},
			"oob_ships":           {json.RawMessage(`{"id":22,"task_force_id":21,"name":"KMS Test","definition":"GER_destroyer","ordering":0,"body_ast":[],"source":"created"}`)},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	oob := contents["Demo/history/units/GER_1936.txt"]
	for _, want := range []string{"air_wings", "fighter_equipment_1", "Home Fleet", "First Task Force", "KMS Test"} {
		if !strings.Contains(oob, want) {
			t.Fatalf("OOB missing %q:\n%s", want, oob)
		}
	}
	if !slices.Contains(artifact.Coverage, "military_oob") {
		t.Fatalf("coverage=%#v", artifact.Coverage)
	}
}

func TestBuildNativeShadowEmitsCharacterScriptsAndLocalisation(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en","zh"]}`),
		Tables: map[string][]json.RawMessage{
			"characters": {
				json.RawMessage(`{"id":1,"tag":"GER_alice","name":"Alice","name_en":"Alice","name_zh":"爱丽丝","country_tag":"GER","role_type":"country_leader","ideology":"democratic","expire":"1965.1.1","traits":[],"advisor_traits":[],"raw_extras":{},"source":"created","source_file":"common/characters/shared.txt","replaces_vanilla_id":false}`),
				json.RawMessage(`{"id":2,"tag":"GER_imported","name_en":"Old","country_tag":"GER","role_type":"country_leader","traits":[],"advisor_traits":[],"raw_extras":{},"source":"imported","source_file":"common/characters/shared.txt","replaces_vanilla_id":false}`),
				json.RawMessage(`{"id":3,"tag":"GER_vanilla","name_en":"Vanilla","country_tag":"GER","role_type":"country_leader","traits":[],"advisor_traits":[],"raw_extras":{},"source":"created","replaces_vanilla_id":true}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	script := contents["Demo/common/characters/shared.txt"]
	if !strings.Contains(script, "GER_alice") || strings.Contains(script, "GER_imported") || strings.Contains(script, "GER_vanilla") {
		t.Fatalf("character script=%q", script)
	}
	if !strings.Contains(contents["Demo/localisation/english/replace/GER_characters_l_english.yml"], "Alice") {
		t.Fatalf("english localisation missing: %#v", contents)
	}
	if !strings.Contains(contents["Demo/localisation/simp_chinese/replace/GER_characters_l_simp_chinese.yml"], "爱丽丝") {
		t.Fatalf("chinese localisation missing: %#v", contents)
	}
	if !slices.Contains(artifact.Coverage, "character_scripts") {
		t.Fatalf("coverage=%#v", artifact.Coverage)
	}
}

func TestBuildNativeShadowShipsCharacterAssetsWithMatchingReferences(t *testing.T) {
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"characters": {json.RawMessage(`{"id":1,"tag":"GER_alice","name_en":"Alice","country_tag":"GER","role_type":"advisor","has_advisor_role":true,"traits":[],"advisor_traits":[],"raw_extras":{},"source":"created","replaces_vanilla_id":false}`)},
		"entity_obs_paths": {
			json.RawMessage(`{"entity_type":"character_portrait","entity_id":1,"obs_key":"p.png"}`),
			json.RawMessage(`{"entity_type":"character_advisor","entity_id":1,"obs_key":"a.png"}`),
		},
	}}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, _ int64) ([]byte, error) { return []byte("PNG:" + key), nil })
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	portrait := "gfx/leaders/GER/portrait_GER_alice.png"
	advisor := "gfx/interface/ideas/advisor_GER_alice.png"
	if contents["Demo/"+portrait] != "PNG:p.png" || contents["Demo/"+advisor] != "PNG:a.png" {
		t.Fatalf("assets missing: %#v", contents)
	}
	if !strings.Contains(contents["Demo/common/characters/GER_GER.txt"], portrait) || !strings.Contains(contents["Demo/interface/GER_characters.gfx"], advisor) {
		t.Fatalf("references not gated with assets: %#v", contents)
	}
}

func TestBuildNativeShadowEmitsDecisionsAndLocalisation(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"decision_categories": {
				json.RawMessage(`{"id":10,"category_id":"GER_category","country_tag":"GER","name_en":"Category","visibility":"visible"}`),
			},
			"decisions": {
				json.RawMessage(`{"id":11,"category_id":10,"decision_id":"GER_decision","country_tag":"GER","name_en":"Decision","source":"created","cost":25}`),
				json.RawMessage(`{"id":12,"category_id":10,"decision_id":"GER_old","country_tag":"GER","name_en":"Old","source":"imported","cost":25}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	decisionPath := "Demo/common/decisions/GER_GER.txt"
	if !strings.Contains(contents[decisionPath], "GER_decision") || strings.Contains(contents[decisionPath], "GER_old") {
		t.Fatalf("decision output=%q entries=%#v", contents[decisionPath], artifact.Entries)
	}
	if _, ok := contents["Demo/localisation/english/replace/GER_decisions_l_english.yml"]; !ok {
		t.Fatalf("missing decision localisation: %#v", contents)
	}
}

func TestBuildNativeShadowMaterializesDecisionIcons(t *testing.T) {
	var source bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 0xdd
	}
	if err := png.Encode(&source, img); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"decision_categories": {json.RawMessage(`{"id":10,"category_id":"GER_category","country_tag":"GER","name_en":"Category","visibility":"visible","icon":"GFX_decision_category_ger_category"}`)},
		"decisions":           {json.RawMessage(`{"id":11,"category_id":10,"decision_id":"GER_decision","country_tag":"GER","name_en":"Decision","source":"created","icon":"GFX_decision_ger_decision"}`)},
		"entity_obs_paths":    {json.RawMessage(`{"entity_type":"decision_category_icon","entity_id":10,"obs_key":"cat.png"}`), json.RawMessage(`{"entity_type":"decision_icon","entity_id":11,"obs_key":"dec.png"}`)},
	}}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(string, int64) ([]byte, error) { return source.Bytes(), nil })
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = entry.Bytes
	}
	if !bytes.HasPrefix(entries["Demo/gfx/interface/decisions/decision_ger_decision.dds"], []byte("DDS ")) {
		t.Fatal("decision DDS missing")
	}
	if !bytes.HasPrefix(entries["Demo/gfx/interface/decisions/decision_category_ger_category.dds"], []byte("DDS ")) {
		t.Fatal("category DDS missing")
	}
	gfx := string(entries["Demo/interface/GER_decisions.gfx"])
	if !strings.Contains(gfx, "GFX_decision_ger_decision") || !strings.Contains(gfx, "GFX_decision_category_ger_category") {
		t.Fatalf("decision GFX mismatch:\n%s", gfx)
	}
}

func TestBuildNativeShadowEmitsReferencedSuperEvents(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"super_events": {
				json.RawMessage(`{"id":21,"country_tag":"GER","number":7,"title_en":"Victory","description_en":"Done","auto_fire_on_startup":true,"source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	if !strings.Contains(contents["Demo/events/GER_se.txt"], "ger_se.7") {
		t.Fatalf("missing super event script: %#v", contents)
	}
	if !strings.Contains(contents["Demo/localisation/english/replace/GER_se_l_english.yml"], "Victory") {
		t.Fatalf("missing super event localisation: %#v", contents)
	}
	if !slices.Contains(artifact.Coverage, "super_event_scripts") {
		t.Fatalf("coverage=%#v", artifact.Coverage)
	}
}

func TestBuildNativeShadowMaterializesSuperEventPictureAndFrames(t *testing.T) {
	var source bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = 0x77
	}
	if err := png.Encode(&source, img); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"super_events":     {json.RawMessage(`{"id":21,"country_tag":"GER","number":7,"title_en":"Victory","auto_fire_on_startup":true,"source":"created"}`)},
		"entity_obs_paths": {json.RawMessage(`{"entity_type":"super_event_picture","entity_id":21,"obs_key":"se.png"}`)},
	}}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(string, int64) ([]byte, error) { return source.Bytes(), nil })
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = entry.Bytes
	}
	for _, path := range []string{"Demo/gfx/tvbigevent/pics/GER_SE_0007.dds", "Demo/gfx/tvbigevent/SE_bg.dds", "Demo/gfx/tvbigevent/SE_frame.dds", "Demo/gfx/tvbigevent/SE_pic.dds", "Demo/gfx/tvbigevent/SE_btn.dds"} {
		if !bytes.HasPrefix(entries[path], []byte("DDS ")) {
			t.Fatalf("missing %s", path)
		}
	}
	if !strings.Contains(string(entries["Demo/interface/GER_se_pictures.gfx"]), "GFX_GER_SE_0007") {
		t.Fatal("super event GFX missing")
	}
	if !strings.Contains(string(entries["Demo/common/scripted_localisation/GER_SE_getpicture.txt"]), "GER_SE_0007") {
		t.Fatal("super event router missing picture")
	}
}

func TestBuildNativeShadowKeepsSuperEventReferencedFromRawFile(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"super_events": {
				json.RawMessage(`{"id":21,"country_tag":"GER","number":7,"title_en":"Victory","description_en":"Done","source":"created"}`),
			},
			"project_raw_files": {
				json.RawMessage(`{"id":31,"dest_path":"common/scripted_effects/zz_story_registry_generated.txt","raw_kind":"verbatim_override","is_binary":false,"content":"story_fire = {\n    country_event = { id = ger_se.7 hours = 6 }\n}\n","source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	if !strings.Contains(contents["Demo/events/GER_se.txt"], "ger_se.7") {
		t.Fatalf("a SuperEvent referenced from a raw-file was dropped; entries=%v", slices.Sorted(maps.Keys(contents)))
	}
	if !strings.Contains(contents["Demo/events/GER_se.txt"], "add_namespace = ger_se") {
		t.Fatalf("namespace missing, leaving the raw-file's call dangling:\n%s", contents["Demo/events/GER_se.txt"])
	}
	for _, warning := range artifact.Warnings {
		if strings.Contains(warning, "ger_se.7") {
			t.Fatalf("kept SuperEvent must not warn: %q", warning)
		}
	}
}

func TestBuildNativeShadowWarnsOnDormantSuperEventDrop(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"super_events": {
				json.RawMessage(`{"id":21,"country_tag":"GER","number":7,"title_en":"Victory","description_en":"Done","source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var found string
	for _, warning := range artifact.Warnings {
		if strings.Contains(warning, "ger_se.7") {
			found = warning
		}
	}
	if found == "" {
		t.Fatalf("dropping a SuperEvent produced no warning; warnings=%#v", artifact.Warnings)
	}

	if strings.Contains(found, "super_event_dormant:") {
		t.Fatalf("raw counter key leaked to the author: %q", found)
	}
}

func TestBuildNativeShadowIncludesRawBytesAndReplacePath(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER"}`),
		Tables: map[string][]json.RawMessage{
			"project_raw_files": {
				json.RawMessage(`{"id":8,"dest_path":"gfx/raw.bin","raw_kind":"verbatim_override","is_binary":true,"content_binary":"\\x0041ff","generate_replace_path":true,"source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(artifact.Entries) != 4 || artifact.Entries[2].Path != "Demo/gfx/raw.bin" {
		t.Fatalf("entries=%#v", artifact.Entries)
	}
	if artifact.Entries[3].Path != "Demo/_RAW_OVERRIDE_FILES.log" {
		t.Fatalf("raw override log missing: %#v", artifact.Entries)
	}
	if got := artifact.Entries[2].Bytes; len(got) != 3 || got[0] != 0 || got[1] != 0x41 || got[2] != 0xff {
		t.Fatalf("raw bytes=%x", got)
	}
	if !strings.Contains(string(artifact.Entries[1].Bytes), `replace_path="gfx"`) {
		t.Fatalf("descriptor=%s", artifact.Entries[1].Bytes)
	}
}

func TestBuildNativeShadowReadsBoundedPrivateRawBytes(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER"}`),
		Tables: map[string][]json.RawMessage{
			"project_raw_files": {json.RawMessage(`{"id":9,"dest_path":"gfx/private.bin","raw_kind":"verbatim_override","is_binary":true,"content_obs_key":"projects/demo/raw/private.bin","source":"created"}`)},
		},
	}
	var requested string
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, maxBytes int64) ([]byte, error) {
		requested = key
		if maxBytes != 256<<20 {
			t.Fatalf("maxBytes=%d", maxBytes)
		}
		return []byte{0, 1, 2}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if requested != "projects/demo/raw/private.bin" {
		t.Fatalf("requested=%q", requested)
	}
	for _, entry := range artifact.Entries {
		if entry.Path == "Demo/gfx/private.bin" {
			if !slices.Equal(entry.Bytes, []byte{0, 1, 2}) {
				t.Fatalf("bytes=%v", entry.Bytes)
			}
			return
		}
	}
	t.Fatal("private raw entry missing")
}

func TestOverlayEntriesLetsExactRawPathWin(t *testing.T) {
	entries, err := overlayEntries(
		[]Entry{{Path: "Demo/common/x.txt", Bytes: []byte("generated")}},
		[]Entry{{Path: "Demo/common/x.txt", Bytes: []byte("raw"), Origin: "raw_override"}},
	)
	if err != nil || len(entries) != 1 || string(entries[0].Bytes) != "raw" {
		t.Fatalf("entries=%#v err=%v", entries, err)
	}
	if _, err := overlayEntries(
		[]Entry{{Path: "Demo/common/X.txt"}}, []Entry{{Path: "Demo/common/x.txt"}},
	); err == nil {
		t.Fatal("case-only collision accepted")
	}
}

func TestBuildNativeShadowEmitsScriptedFiles(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER"}`),
		Tables: map[string][]json.RawMessage{
			"scripted_snippets": {
				json.RawMessage(`{"id":1,"kind":"effect","name":"demo_effect","body":"add_political_power = 5","source_file":"","source":"created"}`),
			},
			"scripted_localisations": {
				json.RawMessage(`{"id":2,"name":"GetDemo","body":"name = GetDemo\ntext = { localization_key = demo_key }","source_file":"","source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	if !strings.Contains(contents["Demo/common/scripted_effects/zz_Demo_scripted_effects.txt"], "demo_effect") {
		t.Fatalf("scripted effect missing: %#v", contents)
	}
	if !strings.Contains(contents["Demo/common/scripted_localisation/zz_Demo_scripted_localisation.txt"], "GetDemo") {
		t.Fatalf("scripted localisation missing: %#v", contents)
	}
}

func TestBuildNativeShadowEmitsWorldFiles(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"countries": {
				json.RawMessage(`{"id":1,"tag":"ZZ1","name_en":"Test Country","name_zh":"","color":[12,34,56],"graphical_culture":"western_european_gfx","graphical_culture_2d":"western_european_2d","ideology":"democratic"}`),
			},
			"history_files": {
				json.RawMessage(`{"id":2,"country_tag":"ZZ1","capital":1,"source":"created","startup_flags":["demo_started"]}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	if !strings.Contains(contents["Demo/history/countries/ZZ1 - zDemo.txt"], "set_country_flag = demo_started") {
		t.Fatalf("history missing: %#v", contents)
	}
	if !strings.Contains(contents["Demo/common/country_tags/00_Demo_countries.txt"], "ZZ1") {
		t.Fatalf("country registry missing: %#v", contents)
	}
}

func TestBuildNativeShadowMaterializesBookmarkBackdrop(t *testing.T) {
	var source bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 0x99
	}
	if err := png.Encode(&source, img); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"bookmarks":        {json.RawMessage(`{"id":9,"bookmark_id":"demo_start","name_en":"Demo","source":"created","picture":"custom"}`)},
		"entity_obs_paths": {json.RawMessage(`{"entity_type":"bookmark_picture","entity_id":9,"obs_key":"bookmark.png"}`)},
	}}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(string, int64) ([]byte, error) { return source.Bytes(), nil })
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = entry.Bytes
	}
	if !bytes.HasPrefix(entries["Demo/gfx/loadingscreens/select_date_demo_start.dds"], []byte("DDS ")) {
		t.Fatal("bookmark DDS missing")
	}
	if !strings.Contains(string(entries["Demo/interface/hagane_bookmarks.gfx"]), "GFX_hagane_select_date_demo_start") {
		t.Fatal("bookmark GFX missing")
	}
	if !strings.Contains(string(entries["Demo/common/bookmarks/demo_start.txt"]), "GFX_hagane_select_date_demo_start") {
		t.Fatal("bookmark script ref missing")
	}
}

func TestBuildNativeShadowMaterializesFlagsAndIdeologyIcons(t *testing.T) {
	var source bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = 0x88
	}
	if err := png.Encode(&source, img); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"],"ideology_icon_paths":{"democratic":"x","pb04_civic_coalition":"x","not_registered":"x"},"ideology_registry":{"groups":["pb04_civic_coalition"],"root_definitions":[{"id":"pb04_civic_coalition","body":"types = { }"}]}}`), Tables: map[string][]json.RawMessage{
		"countries":          {json.RawMessage(`{"id":5,"tag":"ZZ1","name_en":"Test"}`)},
		"entity_obs_paths":   {json.RawMessage(`{"entity_type":"country_flag","entity_id":5,"obs_key":"flag.png"}`)},
		"project_obs_assets": {json.RawMessage(`{"asset_type":"ideology_icon","gfx_name":"democratic","obs_key":"ideology.png"}`), json.RawMessage(`{"asset_type":"ideology_icon","gfx_name":"pb04_civic_coalition","obs_key":"custom_ideology.png"}`), json.RawMessage(`{"asset_type":"ideology_icon","gfx_name":"not_registered","obs_key":"rogue.png"}`), json.RawMessage(`{"asset_type":"country_flag_ideology","gfx_name":"ZZ1_fascism","obs_key":"fascism.png"}`), json.RawMessage(`{"asset_type":"country_flag_ideology","gfx_name":"ZZ1_pb04_civic_coalition","obs_key":"custom.png"}`)},
	}}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(string, int64) ([]byte, error) { return source.Bytes(), nil })
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = entry.Bytes
	}
	for _, path := range []string{"Demo/gfx/flags/ZZ1.tga", "Demo/gfx/flags/ZZ1_democratic.tga", "Demo/gfx/flags/medium/ZZ1_fascism.tga", "Demo/gfx/flags/small/ZZ1_communism.tga", "Demo/gfx/flags/ZZ1_pb04_civic_coalition.tga", "Demo/gfx/flags/medium/ZZ1_pb04_civic_coalition.tga", "Demo/gfx/flags/small/ZZ1_pb04_civic_coalition.tga"} {
		data := entries[path]
		if len(data) < 18 || data[2] != 2 || data[16] != 32 {
			t.Fatalf("invalid TGA %s", path)
		}
	}
	if !bytes.HasPrefix(entries["Demo/gfx/interface/ideologies/democratic_group.dds"], []byte("DDS ")) {
		t.Fatal("ideology DDS missing")
	}
	if !bytes.HasPrefix(entries["Demo/gfx/interface/ideologies/pb04_civic_coalition_group.dds"], []byte("DDS ")) {
		t.Fatal("custom ROOT ideology DDS missing")
	}
	if _, exists := entries["Demo/gfx/interface/ideologies/not_registered_group.dds"]; exists {
		t.Fatal("unregistered ideology ROOT must not be emitted")
	}
	gfx := string(entries["Demo/interface/hagane_ideology_icons.gfx"])
	if !strings.Contains(gfx, "GFX_ideology_pb04_civic_coalition_group") ||
		!strings.Contains(gfx, "gfx/interface/ideologies/pb04_civic_coalition_group.dds") {
		t.Fatalf("custom ROOT ideology GFX missing: %s", gfx)
	}
	if strings.Contains(gfx, "not_registered") {
		t.Fatalf("unregistered ideology ROOT leaked into GFX: %s", gfx)
	}
}

func TestBuildNativeShadowEmitsFocusWithPersistedPolicy(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"focus_trees": {
				json.RawMessage(`{"id":1,"tree_id":"ger_demo_tree","country_tag":"GER","tree_kind":"primary","source":"created","source_file":"demo.txt"}`),
			},
			"focus_nodes": {
				json.RawMessage(`{"id":2,"focus_tree_id":1,"focus_id":"GER_demo_focus","name":"Demo Focus","name_en":"Demo Focus","icon":"GFX_goal_unknown","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[],"ai_will_do":"factor = 1"}`),
			},
			"focus_file_policies": {
				json.RawMessage(`{"id":3,"tree_ids":["ger_demo_tree"],"target_path":"common/national_focus/custom_demo.txt"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	if !strings.Contains(contents["Demo/common/national_focus/custom_demo.txt"], "GER_demo_focus") {
		t.Fatalf("focus script missing: %#v", contents)
	}
	if _, ok := contents["Demo/localisation/english/replace/GER_focus_l_english.yml"]; !ok {
		t.Fatalf("focus localisation missing: %#v", contents)
	}
}

func TestBuildNativeShadowMaterializesFocusIconsWithMatchingReferences(t *testing.T) {
	var source bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	if err := png.Encode(&source, img); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"focus_trees":       {json.RawMessage(`{"id":1,"tree_id":"ger_demo_tree","country_tag":"GER","tree_kind":"primary","source":"created"}`)},
			"focus_nodes":       {json.RawMessage(`{"id":2,"focus_tree_id":1,"focus_id":"GER_demo_focus","name_en":"Demo","icon":"GFX_focus_ger_demo_focus","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[]}`)},
			"project_obs_icons": {json.RawMessage(`{"id":3,"project_id":42,"gfx_name":"GFX_focus_ger_demo_focus","obs_key":"focus/demo.png"}`)},
		},
	}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, maxBytes int64) ([]byte, error) {
		if key != "focus/demo.png" || maxBytes != 32<<20 {
			t.Fatalf("resolver key=%q max=%d", key, maxBytes)
		}
		return source.Bytes(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = entry.Bytes
	}
	dds := entries["Demo/gfx/interface/goals/focus_ger_demo_focus.dds"]
	if !bytes.HasPrefix(dds, []byte("DDS ")) {
		t.Fatalf("focus DDS missing: %q", dds)
	}
	gfx := string(entries["Demo/interface/GER_focus.gfx"])
	if !strings.Contains(gfx, "GFX_focus_ger_demo_focus") || !strings.Contains(gfx, "focus_ger_demo_focus.dds") {
		t.Fatalf("gfx mismatch:\n%s", gfx)
	}

	script := string(entries["Demo/common/national_focus/ger_demo_tree.txt"])
	if !strings.Contains(script, "icon = GFX_focus_ger_demo_focus") {
		t.Fatalf("script icon missing:\n%s", script)
	}
}

func TestBuildNativeShadowAppliesGUIOverlayWithPrivateAssets(t *testing.T) {
	var pngBytes bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 0xaa
	}
	if err := png.Encode(&pngBytes, img); err != nil {
		t.Fatal(err)
	}
	var sourceZip bytes.Buffer
	zw := zip.NewWriter(&sourceZip)
	w, _ := zw.Create("interface/demo.gui")
	_, _ = w.Write([]byte("guiTypes = { containerWindowType = { name = Top instantTextBoxType = { name = Label text = OLD sprite = GFX_old position = { x = 1 y = 2 } } } }"))
	w, _ = zw.Create("common/scripted_guis/demo.txt")
	_, _ = w.Write([]byte("scripted_gui = { demo = { } }\n"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"gui_overlays":       {json.RawMessage(`{"id":1,"file":"interface/demo.gui","edits":[{"target":"Top/Label","op":"set_text","payload":{"key":"DEMO_LABEL","text":"Updated","is_new":true}},{"target":"Top/Label","op":"set_layout","payload":{"x":20,"y":30}},{"target":"Top/Label","op":"swap_sprite","payload":{"gfx_ref":"GFX_old","override_file":"demo_override.png"}}]}`)},
		"project_obs_assets": {json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"gui/source.zip"}`), json.RawMessage(`{"asset_type":"gui_sprite_override","gfx_name":"demo_override.png","obs_key":"gui/demo.png"}`)},
	}}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, max int64) ([]byte, error) {
		if key == "gui/source.zip" {
			return sourceZip.Bytes(), nil
		}
		if key == "gui/demo.png" {
			return pngBytes.Bytes(), nil
		}
		return nil, fmt.Errorf("unexpected key %s", key)
	})
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = entry.Bytes
	}
	gui := string(entries["Demo/interface/demo.gui"])
	if !strings.Contains(gui, `text = "GUISUB_DEMO_LABEL"`) || !strings.Contains(gui, `sprite = "GFX_mysub_demo_override"`) || !strings.Contains(gui, "x = 20") {
		t.Fatalf("GUI overlay mismatch:\n%s", gui)
	}
	if !bytes.HasPrefix(entries["Demo/gfx/interface/demo_override.dds"], []byte("DDS ")) {
		t.Fatal("GUI DDS missing")
	}
	if !strings.Contains(string(entries["Demo/interface/demo_sprites.gfx"]), "GFX_mysub_demo_override") {
		t.Fatal("GUI GFX missing")
	}
	if !bytes.HasPrefix(entries["Demo/localisation/english/replace/demo_gui_submod_l_english.yml"], []byte{0xef, 0xbb, 0xbf}) {
		t.Fatal("GUI localisation missing BOM")
	}
	if localisation := string(entries["Demo/localisation/english/replace/demo_gui_submod_l_english.yml"]); !strings.Contains(localisation, "GUISUB_DEMO_LABEL") || !strings.Contains(localisation, "Updated") {
		t.Fatalf("GUI localisation does not match the emitted GUI key: %s", localisation)
	}
	if !strings.Contains(string(entries["Demo/common/scripted_guis/demo.txt"]), "scripted_gui") {
		t.Fatal("paired scripted GUI missing")
	}
}

func TestBuildNativeShadowSkipsGUIOverlayWithoutScaffold(t *testing.T) {

	panelRow := json.RawMessage(`{"id":1,"token":"GER_power","name_en":"Power","display_name_locs":{"en":"Power"},"context_type":"player_context","visible_trigger":"always = yes","window_props":{"x":10,"y":20,"width":400,"height":300},"controls":[],"raw_extras":{},"source":"created"}`)

	overlayRow := json.RawMessage(`{"id":55,"file":"countrydiplomacyview.gui","source":"vanilla","edits":[{"target":"countrydiplomacyview/diplomacy_title","op":"set_layout","payload":{"x":18,"y":9}}]}`)

	var sourceZip bytes.Buffer
	zw := zip.NewWriter(&sourceZip)
	w, _ := zw.Create("interface/zz_uipanel_my_panel.gui")
	_, _ = w.Write([]byte("guiTypes = { }\n"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name   string
		assets []json.RawMessage
	}{
		{"snapshot present but screen absent", []json.RawMessage{json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"gui/source.zip"}`)}},
		{"no gui_source snapshot at all", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
				"gui_overlays":       {overlayRow},
				"ui_panels":          {panelRow},
				"project_obs_assets": tc.assets,
			}}
			artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, max int64) ([]byte, error) {
				if key == "gui/source.zip" {
					return sourceZip.Bytes(), nil
				}
				return nil, fmt.Errorf("unexpected key %s", key)
			})

			if err != nil {
				t.Fatalf("export must not fail on an unbuildable overlay: %v", err)
			}

			contents := map[string]string{}
			for _, entry := range artifact.Entries {
				contents[entry.Path] = string(entry.Bytes)
			}
			if !strings.Contains(contents["Demo/interface/zz_uipanel_GER_power.gui"], "GER_power_panel") {
				t.Fatalf("unrelated content was dropped: %#v", contents)
			}

			if !slices.ContainsFunc(artifact.Warnings, func(warning string) bool {
				return strings.Contains(warning, "Interface edits to")
			}) {
				t.Fatalf("skip must be announced, warnings=%#v", artifact.Warnings)
			}

			if slices.Contains(artifact.Coverage, "gui_overlays") {
				t.Fatalf("coverage must not claim gui_overlays: %#v", artifact.Coverage)
			}
		})
	}
}

func TestBuildNativeShadowEmitsIdeasAndEvents(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"ideas": {
				json.RawMessage(`{"id":1,"idea_id":"GER_demo_idea","idea_type":"country","country_tag":"GER","name_en":"Demo Idea","source":"created","source_file":""}`),
			},
			"event_namespaces": {
				json.RawMessage(`{"id":2,"namespace":"ger_demo","country_tag":"GER"}`),
			},
			"events": {
				json.RawMessage(`{"id":3,"namespace_id":2,"event_id":"ger_demo.1","event_type":"country_event","title_en":"Demo Event","desc_en":"Description","options":[],"source":"created","source_file":"events/custom.txt"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	if !strings.Contains(contents["Demo/common/ideas/ger_ger.txt"], "GER_demo_idea") {
		t.Fatalf("idea script missing: %#v", contents)
	}
	if !strings.Contains(contents["Demo/events/custom.txt"], "ger_demo.1") {
		t.Fatalf("event script missing: %#v", contents)
	}
	if _, ok := contents["Demo/localisation/english/replace/GER_events_l_english.yml"]; !ok {
		t.Fatalf("event localisation missing: %#v", contents)
	}
}

func TestBuildNativeShadowMaterializesIdeaAndEventPictures(t *testing.T) {
	var source bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 0xcc
	}
	if err := png.Encode(&source, img); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"ideas":            {json.RawMessage(`{"id":1,"idea_id":"GER_demo","country_tag":"GER","idea_type":"country","name_en":"Demo","source":"created"}`)},
		"event_namespaces": {json.RawMessage(`{"id":2,"namespace":"demo","country_tag":"GER"}`)},
		"events":           {json.RawMessage(`{"id":3,"namespace_id":2,"event_id":"demo.1","event_type":"country_event","title_en":"Demo","options":[],"source":"created"}`)},
		"entity_obs_paths": {json.RawMessage(`{"entity_type":"idea_picture","entity_id":1,"obs_key":"idea.png"}`), json.RawMessage(`{"entity_type":"event_picture","entity_id":3,"obs_key":"event.png"}`)},
	}}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(string, int64) ([]byte, error) { return source.Bytes(), nil })
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = entry.Bytes
	}
	if !bytes.HasPrefix(entries["Demo/gfx/interface/ideas/idea_GER_ger_demo.dds"], []byte("DDS ")) {
		t.Fatal("idea DDS missing")
	}
	if !bytes.HasPrefix(entries["Demo/gfx/event_pictures/event_GER_demo1.dds"], []byte("DDS ")) {
		t.Fatal("event DDS missing")
	}
	if !strings.Contains(string(entries["Demo/common/ideas/ger_ger.txt"]), "picture = GER_ger_demo") {
		t.Fatalf("idea picture ref missing:\n%s", entries["Demo/common/ideas/ger_ger.txt"])
	}
	if !strings.Contains(string(entries["Demo/events/GER_demo.txt"]), "picture = GFX_report_event_GER_demo1") {
		t.Fatalf("event picture ref missing:\n%s", entries["Demo/events/GER_demo.txt"])
	}
}

func TestBuildNativeShadowEmitsModifierFamilies(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"dynamic_modifiers": {
				json.RawMessage(`{"id":1,"dm_id":"GER_demo_modifier","name_en":"Demo Modifier","modifier":{"political_power_gain":0.1},"source":"created"}`),
			},
			"autonomy_states": {
				json.RawMessage(`{"id":2,"autonomy_id":"GER_demo_autonomy","name_en":"Demo Autonomy","default":false,"source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	if !strings.Contains(contents["Demo/common/dynamic_modifiers/GER_dynamic_modifiers.txt"], "GER_demo_modifier") {
		t.Fatalf("dynamic modifier missing: %#v", contents)
	}
	if !strings.Contains(contents["Demo/common/autonomous_states/GER_autonomous_states.txt"], "GER_demo_autonomy") {
		t.Fatalf("autonomy state missing: %#v", contents)
	}
}

func TestBuildNativeShadowEmitsModifierDefinitions(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER"}`),
		Tables: map[string][]json.RawMessage{
			"modifier_definitions": {json.RawMessage(`{"id":1,"modifier_id":"GER_test_modifier","color_type":"good","value_type":"percentage","precision":1,"postfix":"none","categories":["country","politics"],"explicit_fields":[],"source":"created"}`)},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	text := contents["Demo/common/modifier_definitions/GER_modifier_definitions.txt"]
	if !strings.Contains(text, "GER_test_modifier") || !strings.Contains(text, "category = politics") {
		t.Fatalf("modifier definition missing: %q", text)
	}
}

func TestBuildNativeShadowEmitsOrderedWargoals(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER"}`),
		Tables: map[string][]json.RawMessage{
			"wargoal_types": {json.RawMessage(`{"id":1,"wargoal_key":"GER_test_goal","war_name":"WAR_NAME","safe_scalars":{"threat":"5"},"allowed":"always = yes","available":"","take_states":"","puppet":"","liberate":"","ordered_entries":[{"key":"custom_unknown","script":"custom_unknown = yes"},{"key":"allowed","script":"allowed = { always = no }"},{"key":"war_name","script":"war_name = OLD"}],"definition_leading_comments":[{"text":"# before goal","inline":false}],"definition_trailing_comments":[{"text":"# goal end","inline":true,"whitespace":"  "}],"file_trivia":{"wrapper_leading":[{"text":"# file start","inline":false}],"block_leading":[{"text":"# block start","inline":false}],"block_trailing":[],"wrapper_trailing":[{"text":"# wrapper end","inline":true,"whitespace":" "}],"root_trailing":[{"text":"# file end","inline":false}]},"source_file":"common/wargoals/custom.txt","source":"created"}`)},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	text := contents["Demo/common/wargoals/custom.txt"]
	for _, want := range []string{"wargoal_types", "GER_test_goal", "custom_unknown = yes", "always = yes", "war_name = WAR_NAME", "threat = 5", "# file start", "# before goal", "\t}  # goal end", "} # wrapper end", "# file end"} {
		if !strings.Contains(text, want) {
			t.Fatalf("wargoal missing %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "custom_unknown") > strings.Index(text, "allowed =") {
		t.Fatalf("ordered entries changed:\n%s", text)
	}
	if !slices.Contains(artifact.Coverage, "wargoal_types") {
		t.Fatalf("coverage=%#v", artifact.Coverage)
	}
}

func TestBuildNativeShadowEmitsUIPanelTriplet(t *testing.T) {
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"ui_panels": {json.RawMessage(`{"id":1,"token":"GER_power","name_en":"Power","display_name_locs":{"en":"Power"},"context_type":"decision_category","parent_window_token":"decision_tab","visible_trigger":"always = yes","window_props":{"x":10,"y":20,"width":400,"height":300},"controls":[{"kind":"text","name":"value","x":10,"y":10,"text":"Power: [?GER_power]","text_locs":{"en":"Power: [?GER_power]"}},{"kind":"button","name":"gain","x":20,"y":40,"text":"Gain","click_effect":"add_political_power = 5","click_enabled_trigger":"has_political_power > 0"}],"raw_extras":{},"source":"created"}`)},
	}}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	if !strings.Contains(contents["Demo/interface/zz_uipanel_GER_power.gui"], "GER_power_panel") {
		t.Fatalf("GUI missing: %#v", contents)
	}
	if !strings.Contains(contents["Demo/common/scripted_guis/zz_uipanel_GER_power.txt"], "GER_power_gain_click") {
		t.Fatalf("scripted GUI missing: %#v", contents)
	}
	if !strings.Contains(contents["Demo/localisation/english/replace/GER_ui_panels_l_english.yml"], "GER_power_value_loc") {
		t.Fatalf("localisation missing: %#v", contents)
	}
	if !slices.Contains(artifact.Coverage, "ui_panels") {
		t.Fatalf("coverage=%#v", artifact.Coverage)
	}
}

func TestBuildNativeShadowEmitsAdvancedScriptFamilies(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"custom_traits":                     {json.RawMessage(`{"id":1,"trait_id":"GER_test_trait","trait_type":"country_leader","country_tag":"GER","name_en":"Test Trait","source":"created"}`)},
			"military_industrial_organizations": {json.RawMessage(`{"id":2,"mio_id":"GER_test_mio","name_en":"Test MIO","source":"created"}`)},
			"intelligence_agencies":             {json.RawMessage(`{"id":3,"agency_id":"GER_test_agency","names":["GER_TEST_AGENCY"],"source":"created"}`)},
			"special_projects":                  {json.RawMessage(`{"id":4,"project_token":"GER_test_project","specialization":"land","source":"created"}`)},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	checks := map[string]string{
		"Demo/common/country_leader/GER_custom_traits.txt":                                 "GER_test_trait",
		"Demo/common/military_industrial_organization/organizations/GER_organizations.txt": "GER_test_mio",
		"Demo/common/intelligence_agencies/GER_intelligence_agencies.txt":                  "GER_TEST_AGENCY",
		"Demo/common/special_projects/projects/GER_projects.txt":                           "GER_test_project",
	}
	for path, token := range checks {
		if !strings.Contains(contents[path], token) {
			t.Fatalf("%s missing %q: %#v", path, token, contents)
		}
	}
}

func TestBuildNativeShadowMaterializesAdvancedImages(t *testing.T) {
	var source bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 0xee
	}
	if err := png.Encode(&source, img); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"military_industrial_organizations": {json.RawMessage(`{"id":2,"mio_id":"GER_mio","icon":"GFX_idea_mio_ger_mio","name_en":"MIO","source":"created"}`)},
		"intelligence_agencies":             {json.RawMessage(`{"id":3,"agency_id":"GER_agency","picture":"GFX_intelligence_agency_logo_ger_agency","names":["GER_AGENCY"],"source":"created"}`)},
		"custom_traits":                     {json.RawMessage(`{"id":4,"trait_id":"GER_trait","trait_type":"unit_leader","country_tag":"GER","name_en":"Trait","source":"created"}`)},
		"entity_obs_paths":                  {json.RawMessage(`{"entity_type":"mio_icon","entity_id":2,"obs_key":"mio.png"}`), json.RawMessage(`{"entity_type":"intel_agency_logo","entity_id":3,"obs_key":"agency.png"}`), json.RawMessage(`{"entity_type":"custom_trait_icon","entity_id":4,"obs_key":"trait.png"}`)},
	}}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(string, int64) ([]byte, error) { return source.Bytes(), nil })
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = entry.Bytes
	}
	for _, path := range []string{"Demo/gfx/interface/ideas/mio_ger_mio.dds", "Demo/gfx/interface/intelligence_agencies/intel_agency_ger_agency.dds", "Demo/gfx/interface/traits/customtrait_unit_leader_ger_trait.dds"} {
		if !bytes.HasPrefix(entries[path], []byte("DDS ")) {
			t.Fatalf("missing %s", path)
		}
	}
	gfx := string(entries["Demo/interface/GER_advanced_assets.gfx"])
	for _, name := range []string{"GFX_idea_mio_ger_mio", "GFX_intelligence_agency_logo_ger_agency", "GFX_trait_GER_trait"} {
		if !strings.Contains(gfx, name) {
			t.Fatalf("advanced GFX missing %s:\n%s", name, gfx)
		}
	}
}

func TestBuildNativeShadowEmitsAIPlanningFiles(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER"}`),
		Tables: map[string][]json.RawMessage{
			"ai_strategy_plans": {json.RawMessage(`{"id":1,"plan_id":"GER_plan","name":"Plan","desc":"Test","allowed":["{ always = yes }"],"enable":[],"abort":[],"ai_national_focuses":[],"focus_factors":[],"research":[],"ideas":[],"traits":[],"ai_strategy":[],"weight":[],"field_order":["name","allowed"],"source_file":"common/ai_strategy_plans/custom.txt","source":"created"}`)},
			"ai_templates":      {json.RawMessage(`{"id":2,"template_id":"GER_role","role":"infantry","blocked_for":"","available_for":"","upgrade_prio":"","enable":"","front_role_override":"","target_templates":[{"id":"GER_line","body":"{ target_template = GER_line }"}],"field_order":["role","target:GER_line"],"source":"created"}`)},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	if !strings.Contains(contents["Demo/common/ai_strategy_plans/custom.txt"], "GER_plan") || !strings.Contains(contents["Demo/common/ai_strategy_plans/custom.txt"], "always = yes") {
		t.Fatalf("strategy plan missing: %#v", contents)
	}
	if !strings.Contains(contents["Demo/common/ai_templates/ger_ai_templates.txt"], "GER_line") {
		t.Fatalf("AI template missing: %#v", contents)
	}
	if !slices.Contains(artifact.Coverage, "ai_strategy_plans") || !slices.Contains(artifact.Coverage, "ai_templates") {
		t.Fatalf("coverage=%#v", artifact.Coverage)
	}
}

func TestSuperEventPlaceholderFramesKeepTheirOwnFill(t *testing.T) {
	frame, ok := sePlaceholderFrames["SE_frame.dds"]
	if !ok {
		t.Fatal("SE_frame.dds placeholder missing")
	}
	if frame.Fill[3] != 0 {
		t.Fatalf("SE_frame.dds interior alpha = %d, want 0 — it is drawn over the "+
			"author's picture and must not cover it", frame.Fill[3])
	}

	if frame.Border[3] != 255 {
		t.Errorf("SE_frame.dds border alpha = %d, want 255", frame.Border[3])
	}

	bg := sePlaceholderFrames["SE_bg.dds"]
	if bg.Fill[3] < 200 {
		t.Errorf("SE_bg.dds is the backing plate; alpha = %d, want opaque", bg.Fill[3])
	}

	want := map[string]placeholderFrame{
		"SE_bg.dds":    {500, 700, [4]byte{22, 26, 34, 240}, [4]byte{70, 78, 92, 255}},
		"SE_frame.dds": {500, 700, [4]byte{0, 0, 0, 0}, [4]byte{150, 156, 170, 255}},
		"SE_pic.dds":   {460, 252, [4]byte{52, 58, 70, 255}, [4]byte{96, 104, 120, 255}},
		"SE_btn.dds":   {120, 32, [4]byte{60, 72, 96, 255}, [4]byte{120, 136, 164, 255}},
	}
	if len(sePlaceholderFrames) != len(want) {
		t.Fatalf("placeholder count = %d, want %d", len(sePlaceholderFrames), len(want))
	}
	for name, expected := range want {
		if got := sePlaceholderFrames[name]; got != expected {
			t.Errorf("%s = %+v, want %+v", name, got, expected)
		}
	}

	raw := makePlaceholderDDS(frame)
	if len(raw) == 0 {
		t.Fatal("frame placeholder encoded to nothing")
	}
}
