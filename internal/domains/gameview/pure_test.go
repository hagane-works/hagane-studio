package gameview

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/project"
)

const ideasGFX = `spriteTypes = {
	# advisor seats
	spriteType = {
		name = "GFX_idea_slot_army_chief"
		texturefile = "gfx/interface/ideas/select_advisor.dds"
	}
	spriteType = { name = "GFX_idea_slot_navy_chief" texturefile = "gfx/interface/ideas/select_advisor.dds" }
	SpriteType = {
		name = GFX_topbar_fuel_bar
		textureFile1 = "gfx/interface/topbar/fuel_bar.dds"
		textureFile2 = "gfx/interface/topbar/fuel_bar_bg.dds"
	}
}
`

func TestSpriteTextureEditRepointsOnlyThatSprite(t *testing.T) {
	out, err := spriteTextureEdit(ideasGFX, "GFX_idea_slot_army_chief", "gfx/interface/game_view/gfx_idea_slot_army_chief.dds")
	if err != nil {
		t.Fatal(err)
	}
	if _, values := spriteTextureSpans(ideasGFX, "GFX_idea_slot_army_chief"); values[0] != "gfx/interface/ideas/select_advisor.dds" {
		t.Fatalf("old texture = %q", values)
	}
	want := strings.Replace(ideasGFX,
		"name = \"GFX_idea_slot_army_chief\"\n\t\ttexturefile = \"gfx/interface/ideas/select_advisor.dds\"",
		"name = \"GFX_idea_slot_army_chief\"\n\t\ttexturefile = \"gfx/interface/game_view/gfx_idea_slot_army_chief.dds\"", 1)
	if out != want {
		t.Fatalf("only the army chief's texture changes:\n%s", out)
	}

	bar, err := spriteTextureEdit(ideasGFX, "GFX_topbar_fuel_bar", "gfx/interface/game_view/fuel.dds")
	if err != nil || !strings.Contains(bar, `textureFile1 = "gfx/interface/game_view/fuel.dds"`) ||
		!strings.Contains(bar, `textureFile2 = "gfx/interface/topbar/fuel_bar_bg.dds"`) {
		t.Fatalf("progress bar: err %v\n%s", err, bar)
	}
	if _, err := spriteTextureEdit(ideasGFX, "GFX_nothing", "x.dds"); err == nil {
		t.Fatal("a sprite the file does not declare is an error")
	}
}

type fakeSpriteIndex struct {
	file, body string
}

func (f fakeSpriteIndex) SpriteSource(name string) (string, []byte, bool) {
	if _, values := spriteTextureSpans(f.body, name); len(values) == 0 {
		return "", nil, false
	}
	return f.file, []byte(f.body), true
}

func (f fakeSpriteIndex) ScreenPictureFile(string) (string, bool) { return "", false }

func testProject(t *testing.T) *project.Project {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func rawFileByDest(t *testing.T, p *project.Project, dest string) (RawFile, bool) {
	t.Helper()
	rows, err := p.Rows("project_raw_files")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if str(row["dest_path"]) == dest {
			return rawFileFromRow(row), true
		}
	}
	return RawFile{}, false
}

func TestSpriteOverrideIntoTheModAndBack(t *testing.T) {
	p := testProject(t)
	index := fakeSpriteIndex{file: "interface/ideas.gfx", body: ideasGFX}
	compile := func(name string, target spriteTarget, sha, key string, previous spriteOverrideMetadata) (map[string]any, spriteOverrideMetadata) {
		var out map[string]any
		var metadata spriteOverrideMetadata
		err := withSpriteMutation(p, func(m *spriteMutation) error {
			var err error
			out, metadata, err = compileSpriteOverride(m, index, target, name, sha, key, previous)
			if err != nil {
				return err
			}
			return m.commit(guiSpriteOverrideName(name), "local/png/"+sha, metadata)
		})
		if err != nil {
			t.Fatal(err)
		}
		return out, metadata
	}
	restore := func(name string, metadata spriteOverrideMetadata) map[string]any {
		var out map[string]any
		err := withSpriteMutation(p, func(m *spriteMutation) error {
			var err error
			if out, err = uncompileSpriteOverride(m, index, name, metadata); err != nil {
				return err
			}
			m.remove(guiSpriteOverrideName(name))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	armyTarget := spriteTarget{Width: 4, Height: 3, Frames: 1, Texture: "gfx/interface/ideas/select_advisor.dds"}
	armyTexture := spriteTextureDest("GFX_idea_slot_army_chief")
	out, armyMetadata := compile("GFX_idea_slot_army_chief", armyTarget, "sha1", "local/rawblobs/sh/sha1", spriteOverrideMetadata{})
	if out["gfx"] != "interface/ideas.gfx" || out["texture"] != armyTexture {
		t.Fatalf("compiled = %v", out)
	}
	texture, ok := rawFileByDest(t, p, armyTexture)
	if !ok || !texture.IsBinary || *texture.ContentOBSKey != "local/rawblobs/sh/sha1" {
		t.Fatalf("the picture is a binary raw file: %+v", texture)
	}
	gfx, ok := rawFileByDest(t, p, "interface/ideas.gfx")
	if !ok || gfx.RawKind != "verbatim_override" || !strings.Contains(gfx.Content, armyTexture) ||
		!strings.Contains(gfx.Content, `name = "GFX_idea_slot_navy_chief" texturefile = "gfx/interface/ideas/select_advisor.dds"`) {
		t.Fatalf("the .gfx copy: %+v", gfx)
	}

	_, navyMetadata := compile("GFX_idea_slot_navy_chief", spriteTarget{Width: 4, Height: 3, Frames: 1, Texture: "gfx/interface/ideas/select_advisor.dds"}, "sha2", "k2", spriteOverrideMetadata{})
	armyTarget.Texture = armyTexture
	_, armyMetadata = compile("GFX_idea_slot_army_chief", armyTarget, "sha3", "k3", armyMetadata)
	rows, _ := p.Rows("project_raw_files")
	if len(rows) != 3 {
		t.Fatalf("one .gfx copy and two pictures, got %d files", len(rows))
	}
	if texture, _ := rawFileByDest(t, p, armyTexture); *texture.ContentOBSKey != "k3" {
		t.Fatalf("the newer picture replaces the older: %v", *texture.ContentOBSKey)
	}

	back := restore("GFX_idea_slot_army_chief", armyMetadata)
	if back["texture_removed"] == nil || back["gfx"] != "interface/ideas.gfx" {
		t.Fatalf("taken back = %v", back)
	}
	gfx, _ = rawFileByDest(t, p, "interface/ideas.gfx")
	if !strings.Contains(gfx.Content, "name = \"GFX_idea_slot_army_chief\"\n\t\ttexturefile = \"gfx/interface/ideas/select_advisor.dds\"") ||
		!strings.Contains(gfx.Content, spriteTextureDest("GFX_idea_slot_navy_chief")) {
		t.Fatalf("army chief back, navy chief still replaced:\n%s", gfx.Content)
	}

	restore("GFX_idea_slot_navy_chief", navyMetadata)
	if rows, _ := p.Rows("project_raw_files"); len(rows) != 0 {
		t.Fatalf("nothing of the replacement is left in the mod: %v", rows)
	}
}

func TestSpriteNameOfOverride(t *testing.T) {
	for _, name := range []string{"GFX_idea_slot_army_chief", "GFX_global_alert_icons", "GFX_mapmode_buttons_small"} {
		got, ok := spriteNameOfOverride(guiSpriteOverrideName(name))
		if !ok || got != name {
			t.Fatalf("%s came back as %q, %v", name, got, ok)
		}
	}
	if _, ok := spriteNameOfOverride("snapshot.zip"); ok {
		t.Fatal("not an override name")
	}
	if _, ok := spriteNameOfOverride(guiSpriteOverrideName("GFX weird/name")); ok {
		t.Fatal("a name that does not survive the round trip is not guessed")
	}
	if got := spriteTextureDest("GFX_Idea_Slot.Army"); got != "gfx/interface/game_view/gfx_idea_slot_army_64cd5fa21b.dds" {
		t.Fatalf("texture dest = %q", got)
	}
	if spriteTextureDest("GFX_a.b") == spriteTextureDest("GFX_a_b") {
		t.Fatal("sanitized sprite names collided")
	}
	a := spriteOverrideRoute(2, "GFX_x", "env/p/gui_sprite_override/a")
	b := spriteOverrideRoute(2, "GFX_x", "env/p/gui_sprite_override/b")
	if !strings.HasPrefix(a, "/api/go/projects/2/gui-render/sprite?name=GFX_x&v=") || a == b {
		t.Fatalf("routes %q %q", a, b)
	}
}

func TestSpriteDDSIsAGameTexture(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 4))
	img.Set(1, 1, color.NRGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	dds, err := spriteDDS(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if string(dds[:4]) != "DDS " || dds[12] != 4 || dds[16] != 8 {
		t.Fatalf("a DXT5 .dds of the picture's size, header %v", dds[:20])
	}
}

func TestPreviewSpecMatchesExportPageIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, project, identity string
		legacy, imported        bool
		base                    importedBase
		replace                 replaceScope
	}{
		{name: "standalone", project: `{}`, identity: "standalone", base: baseVanilla},
		{name: "submod", project: `{"dependencies":["The New Order"]}`, identity: "submod", base: baseImported},
		{name: "loaded", project: `{"is_loaded_existing_mod":true}`, identity: "loaded_existing", imported: true, base: baseImportedUI, replace: replaceVanillaOnly},
		{name: "ambiguous", project: `{"is_loaded_existing_mod":true,"dependencies":["TNO"]}`, identity: "ambiguous", imported: true, base: baseImportedUI, replace: replaceVanillaOnly},
		{name: "legacy", project: `{}`, legacy: true, identity: "legacy_import", imported: true, base: baseImported, replace: replaceVanillaOnly},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := previewSpecFor(exportengine.Snapshot{Project: json.RawMessage(tc.project)}, tc.legacy)
			if got.Identity != tc.identity || got.IncludeImported != tc.imported || got.Base != tc.base || got.ReplaceScope != tc.replace {
				t.Fatalf("spec = %+v", got)
			}
		})
	}
}

func TestUntouchedImportedCountriesStayOut(t *testing.T) {
	row := func(tag, created, updated string) json.RawMessage {
		return json.RawMessage(`{"tag":"` + tag + `","created_at":"` + created + `","updated_at":"` + updated + `"}`)
	}
	snapshot := exportengine.Snapshot{Tables: map[string][]json.RawMessage{
		"countries": {
			row("GER", "2026-09-01T10:00:00", "2026-09-01T10:00:00"),
			row("ita", "2026-09-01T10:00:00", "2026-09-20T08:30:00"),
			json.RawMessage(`{"tag":"FRA","auto_added":false,"created_at":"2026-09-21T09:00:00","updated_at":"2026-09-21T09:00:00"}`),
			row("XYZ", "2026-09-21T09:00:00", "2026-09-21T09:00:00"),
		},
		"events": {json.RawMessage(`{"id":1}`)},
	}}
	kept := keepProjectCountries(snapshot, []string{"GER", "ITA", "FRA"})
	tags := []string{}
	for _, raw := range kept.Tables["countries"] {
		var r struct{ Tag string }
		_ = json.Unmarshal(raw, &r)
		tags = append(tags, r.Tag)
	}
	if strings.Join(tags, ",") != "ita,FRA,XYZ" || len(kept.Tables["events"]) != 1 || len(snapshot.Tables["countries"]) != 4 {
		t.Fatalf("kept %v", tags)
	}
}

func TestKeptKnownCountriesLoseTheirRowIdeology(t *testing.T) {
	snapshot := exportengine.Snapshot{Tables: map[string][]json.RawMessage{
		"countries": {
			json.RawMessage(`{"tag":"ITA","ideology":"neutrality","name_en":"Italy","created_at":"a","updated_at":"b"}`),
			json.RawMessage(`{"tag":"XYZ","ideology":"fascism","created_at":"c","updated_at":"c"}`),
		},
	}}
	kept := keepProjectCountries(snapshot, []string{"ITA"})
	var italy, xyz map[string]any
	_ = json.Unmarshal(kept.Tables["countries"][0], &italy)
	_ = json.Unmarshal(kept.Tables["countries"][1], &xyz)
	if _, ok := italy["ideology"]; ok || italy["name_en"] != "Italy" || xyz["ideology"] != "fascism" {
		t.Fatalf("Italy = %v, XYZ = %v", italy, xyz)
	}
}

func TestProjectFlagURLs(t *testing.T) {
	rows := []json.RawMessage{
		json.RawMessage(`{"id":2,"tag":"GER","flag_image_path":"flags/GER.png","flag_paths_by_ideology":{"fascism":"flags/GER_fascism.png","communism":""},"updated_at":"2026-09-30T05:00:00Z"}`),
		json.RawMessage(`{"id":3,"tag":"ITA","flag_image_path":"","flag_paths_by_ideology":{},"updated_at":"x"}`),
		json.RawMessage(`{"id":4,"tag":"ENG","flag_image_path":"flags/ENG.png","flag_paths_by_ideology":null,"updated_at":"y"}`),
	}
	got := projectFlagURLs(7, exportengine.Snapshot{Tables: map[string][]json.RawMessage{"countries": rows}})
	want := map[string]string{
		"GER":         "/api/go/projects/7/countries/2/flag?v=2026-09-30T05%3A00%3A00Z",
		"GER_fascism": "/api/go/projects/7/countries/2/flag?ideology=fascism&v=2026-09-30T05%3A00%3A00Z",
		"ENG":         "/api/go/projects/7/countries/4/flag?v=y",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flags = %v", got)
	}
}

func TestUIPanelWindowEnvelopeCarriesExactControlPaths(t *testing.T) {
	panel := uiPanel{
		ID: 20, Token: "CC1 war", UpdatedAt: time.Date(2026, 10, 14, 10, 11, 12, 123, time.UTC),
		Controls: []map[string]any{{
			"kind": "container", "name": "body",
			"children": []any{map[string]any{"kind": "text", "name": "tension value"}},
		}},
	}
	envelope, ok := uiPanelWindowEnvelopeFor([]uiPanel{panel}, "interface/zz_uipanel_CC1_war.gui", "CC1_war_panel")
	if !ok || envelope.PanelID != 20 || envelope.Token != "CC1 war" || envelope.Revision != "2026-10-14T10:11:12.000000123Z" {
		t.Fatalf("identity = %#v", envelope)
	}
	want := map[string]uiPanelWindowControlEnvelope{
		"CC1_war_panel/CC1_war_body":                       {Name: "body", Path: "body"},
		"CC1_war_panel/CC1_war_body/CC1_war_tension_value": {Name: "tension value", Path: "body/tension value"},
	}
	if !reflect.DeepEqual(envelope.Controls, want) {
		t.Fatalf("controls = %#v", envelope.Controls)
	}
	if _, ok := uiPanelWindowEnvelopeFor([]uiPanel{panel, panel}, "interface/zz_uipanel_CC1_war.gui", "CC1_war_panel"); ok {
		t.Fatal("two rows compiling the same window are not a writable target")
	}
}

func TestInjectionTextDecoding(t *testing.T) {
	if got := decodeInjectionText(append([]byte{0xef, 0xbb, 0xbf}, "window"...)); got != "window" {
		t.Fatalf("utf-8 = %q", got)
	}
	if got := decodeInjectionText([]byte{0x63, 0x61, 0x66, 0xe9}); got != "café" {
		t.Fatalf("latin-1 = %q", got)
	}
	if kind, base := injectionSourceKind("common/scripted_guis/a.txt"); kind != "gui" || base != "a.txt" {
		t.Fatal(kind, base)
	}
	if kind, _ := injectionSourceKind("common/scripted_effects/sub/a.txt"); kind != "" {
		t.Fatal("not recursive")
	}
}

func TestIndexCacheKeepsAProjectsLatestBuilds(t *testing.T) {
	cache := newIndexCache(10, 2)
	build := func(context.Context) (*guirender.Index, error) { return guirender.NewIndex(map[string][]byte{}) }
	for _, key := range []string{"7|a", "7|b", "8|x", "7|c"} {
		if _, err := cache.load(context.Background(), key, build); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond)
	}
	kept := sortedKeys(cache.entries)
	if !reflect.DeepEqual(kept, []string{"7|b", "7|c", "8|x"}) {
		t.Fatalf("kept %v", kept)
	}

	small := newIndexCache(2, 2)
	for _, key := range []string{"1|a", "2|a", "3|a"} {
		_, _ = small.load(context.Background(), key, build)
		time.Sleep(2 * time.Millisecond)
	}
	if kept := sortedKeys(small.entries); !reflect.DeepEqual(kept, []string{"2|a", "3|a"}) {
		t.Fatalf("kept %v", kept)
	}
	if !small.dropIdle(0) || len(small.entries) != 0 {
		t.Fatal("idle builds were not dropped")
	}
}

func TestIndexCacheCoalescesConcurrentBuilds(t *testing.T) {
	cache := newIndexCache(4, 2)
	var builds atomic.Int32
	release := make(chan struct{})
	build := func(context.Context) (*guirender.Index, error) {
		builds.Add(1)
		<-release
		return guirender.NewIndex(map[string][]byte{})
	}
	var wait sync.WaitGroup
	results := make([]*guirender.Index, 8)
	for i := range results {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results[i], _ = cache.load(context.Background(), "1|x", build)
		}()
	}
	time.Sleep(20 * time.Millisecond)
	close(release)
	wait.Wait()
	if builds.Load() != 1 {
		t.Fatalf("%d builds for one key", builds.Load())
	}
	for _, index := range results {
		if index == nil || index != results[0] {
			t.Fatal("waiters did not share the build")
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = cache.load(ctx, "1|y", func(context.Context) (*guirender.Index, error) {
		return guirender.NewIndex(map[string][]byte{})
	})
	if index, err := cache.load(context.Background(), "1|y", build); err != nil || index == nil {
		t.Fatal(index, err)
	}
}

func TestRawFiles(t *testing.T) {
	p := testProject(t)
	sha, key := "abc", "local/rawblobs/ab/abc"
	err := p.Write([]string{"project_raw_files"}, func(tx *project.Tx) error {
		raw := NewRawFiles(tx)
		first, err := raw.Create(RawFileInsert{DestPath: "common/x.txt", Content: "a = 1", RawKind: "verbatim_override", Note: "n"})
		if err != nil {
			return err
		}
		if _, err := raw.Create(RawFileInsert{DestPath: "common/x.txt", Content: "b"}); err != ErrRawFileConflict {
			t.Fatalf("same path twice: %v", err)
		}
		if _, err := raw.Create(RawFileInsert{DestPath: "gfx/x.dds", IsBinary: true}); err == nil {
			t.Fatal("a binary row needs its bytes")
		}
		if _, err := raw.Create(RawFileInsert{DestPath: "gfx/x.dds", IsBinary: true, ContentSHA256: &sha, ContentOBSKey: &key}); err != nil {
			return err
		}
		updated, err := raw.Update(first.ID, RawFileIntent{RawKind: "verbatim_override", DestPath: "common/x.txt", Content: "a = 2"}, nil)
		if err != nil || updated.Content != "a = 2" || updated.Note != "n" || updated.Source != "created" {
			t.Fatalf("update = %+v %v", updated, err)
		}
		if list := raw.List(); len(list) != 2 || list[0].DestPath != "common/x.txt" {
			t.Fatalf("list = %+v", list)
		}
		if raw.Delete(first.ID) != nil || raw.Delete(first.ID) != ErrRawFileMissing {
			t.Fatal("delete")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, bad := SanitizeDestPath(`\interface\.\x.gfx`); got != "interface/x.gfx" || bad != "" {
		t.Fatal(got, bad)
	}
	for _, path := range []string{"", "C:/x.txt", "a/../b.txt", "common/folder"} {
		if _, bad := SanitizeDestPath(path); bad == "" {
			t.Fatalf("%q accepted", path)
		}
	}
	if RejectReservedDest("descriptor.mod") == "" || RejectReservedDest("common/x.txt") != "" {
		t.Fatal("reserved paths")
	}
	if GateRawContent("interface/x.gfx", "a = { b = { }", false) == nil {
		t.Fatal("an unclosed block passed")
	}
	if GateRawContent("interface/x.gfx", "a = { # } \n }", false) != nil {
		t.Fatal("a brace in a comment counted")
	}
	if GateRawContent("localisation/x.yml", "{", false) != nil {
		t.Fatal("localisation is not script")
	}
}

func TestStoreRawBlobIsContentAddressed(t *testing.T) {
	p := testProject(t)
	sha, key, err := storeRawBlob(p, []byte("dds bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "local/rawblobs/"+sha[:2]+"/") || !p.Blobs().Exists(key) {
		t.Fatal(sha, key)
	}
	again, againKey, _ := storeRawBlob(p, []byte("dds bytes"))
	if again != sha || againKey != key {
		t.Fatal("the same bytes got another key")
	}
}

func TestProjectStampFollowsWrites(t *testing.T) {
	p := testProject(t)
	before, err := projectStamp(p)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := projectStamp(p); again != before {
		t.Fatal("the stamp changed without a write")
	}
	if err := p.Write([]string{"gui_overlays"}, func(tx *project.Tx) error {
		tx.Insert("gui_overlays", project.Row{"file": "x.gui"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if after, _ := projectStamp(p); after == before {
		t.Fatal("the stamp did not follow a write")
	}
}
