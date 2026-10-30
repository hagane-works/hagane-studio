package characters

import (
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
)

func (e *testEnv) createTrait(body map[string]any) CustomTrait {
	e.t.Helper()
	rec := e.json("POST", "/custom-traits", body)
	expect(e.t, rec, http.StatusCreated, "")
	return decode[CustomTrait](e.t, rec)
}

func TestCustomTraitCRUD(t *testing.T) {
	e := newEnv(t, nil)
	a := e.createTrait(map[string]any{"trait_id": "  zeta_trait ", "modifier": map[string]any{"stability_factor": 0.05}})
	if a.TraitID != "zeta_trait" || a.TraitType != "country_leader" || a.CountryTag != "TST" || a.NameEn != "zeta_trait" ||
		!a.Random || a.TargetedModifier == nil || a.Modifier["stability_factor"] != 0.05 {
		t.Fatalf("defaults: %+v", a)
	}
	b := e.createTrait(map[string]any{"trait_id": "alpha", "trait_type": "unit_leader", "country_tag": "ger", "random": false,
		"extra_fields": map[string]any{"type": "navy"}})
	c := e.createTrait(map[string]any{"trait_id": "beta", "trait_type": "country_leader"})
	if b.CountryTag != "GER" || b.Random {
		t.Fatalf("unit leader trait: %+v", b)
	}
	expect(t, e.json("POST", "/custom-traits", map[string]any{"trait_id": "alpha", "trait_type": "unit_leader"}), 409,
		"Custom trait 'alpha' already exists for type 'unit_leader'")
	expect(t, e.json("POST", "/custom-traits", map[string]any{"trait_id": "x", "trait_type": "general"}), 400,
		"Invalid trait_type. Must be one of: ['country_leader', 'scientist', 'unit_leader']")
	expect(t, e.json("POST", "/custom-traits", map[string]any{"trait_id": "  "}), 400, "trait_id cannot be empty")
	expect(t, e.json("POST", "/custom-traits", map[string]any{}), 422, "trait_id is required")
	expect(t, e.json("POST", "/custom-traits", "[1]"), 422, "Invalid trait payload")

	list := decode[[]CustomTrait](t, e.json("GET", "/custom-traits", nil))
	if len(list) != 3 || list[0].TraitID != "beta" || list[1].TraitID != "zeta_trait" || list[2].TraitID != "alpha" {
		t.Fatalf("list order: %+v", list)
	}
	filtered := decode[[]CustomTrait](t, e.json("GET", "/custom-traits?trait_type=unit_leader", nil))
	if len(filtered) != 1 || filtered[0].ID != b.ID {
		t.Fatalf("filtered: %+v", filtered)
	}

	rec := e.json("PUT", "/custom-traits/"+itoa(a.ID), map[string]any{"trait_id": " zeta_2 ", "country_tag": " fra ", "ai_will_do": "factor = 2"})
	expect(t, rec, 200, "")
	updated := decode[CustomTrait](t, rec)
	if updated.TraitID != "zeta_2" || updated.CountryTag != "FRA" || updated.AIWillDo != "factor = 2" {
		t.Fatalf("update: %+v", updated)
	}
	expect(t, e.json("PUT", "/custom-traits/"+itoa(a.ID), map[string]any{"trait_type": "general"}), 400, "Invalid trait_type")
	expect(t, e.json("PUT", "/custom-traits/"+itoa(a.ID), map[string]any{"trait_id": ""}), 400, "trait_id cannot be empty")
	expect(t, e.json("PUT", "/custom-traits/"+itoa(a.ID), map[string]any{"trait_id": "beta"}), 503, "trait store unavailable")
	expect(t, e.json("PUT", "/custom-traits/999", map[string]any{"name_en": "x"}), 404, "Custom trait not found")
	expect(t, e.json("GET", "/custom-traits/"+itoa(c.ID), nil), 200, `"trait_id":"beta"`)

	if rec := e.json("DELETE", "/custom-traits/"+itoa(c.ID), nil); rec.Code != 204 {
		t.Fatalf("delete: %d", rec.Code)
	}
	expect(t, e.json("GET", "/custom-traits/"+itoa(c.ID), nil), 404, "Custom trait not found")
	expect(t, e.json("DELETE", "/custom-traits/"+itoa(c.ID), nil), 404, "Custom trait not found")
}

func TestKnownTraitsReadsTheGame(t *testing.T) {
	game := fstest.MapFS{
		"common/country_leader/00_traits.txt": {Data: []byte("leader_traits = {\n\tbrave_leader = { stability_factor = 0.1 }\n\tcautious = { }\n}\n")},
		"common/unit_leader/00_traits.txt": {Data: []byte(`leader_traits = {
	brilliant_strategist = { type = land trait_type = personality_trait }
	sea_wolf = { type = { navy operative } trait_type = basic_trait }
}
`)},
		"common/unit_leader/00_skills.txt":       {Data: []byte("leader_skills = { attack = { } }\n")},
		"common/scientist_traits/00_traits.txt":  {Data: []byte("scientist_traits = { brilliant_mind = { } }\n")},
		"common/country_leader/readme.md":        {Data: []byte("ignored")},
		"common/country_leader/01_more.txt":      {Data: []byte("leader_traits = { brave_leader = { } }\n")},
		"common/country_leader/02_unwrapped.txt": {Data: []byte("odd_trait = { }\n")},
	}
	e := newEnv(t, game)
	e.createTrait(map[string]any{"trait_id": "my_navy", "trait_type": "unit_leader", "name_en": "My navy",
		"extra_fields": map[string]any{"type": "navy", "trait_type": "assignable_trait"}})
	e.createTrait(map[string]any{"trait_id": "a_leader", "trait_type": "unit_leader"})
	e.createTrait(map[string]any{"trait_id": "cl", "trait_type": "country_leader"})

	got := decode[knownTraitsResponse](t, e.json("GET", "/custom-traits/known?trait_type=unit_leader", nil))
	if strings.Join(got.Vanilla, ",") != "brilliant_strategist,sea_wolf" {
		t.Fatalf("vanilla: %v", got.Vanilla)
	}
	if m := got.VanillaMetadata["sea_wolf"]; strings.Join(m.Types, ",") != "navy,operative" || strings.Join(m.TraitTypes, ",") != "basic_trait" {
		t.Fatalf("metadata: %+v", got.VanillaMetadata)
	}
	if len(got.Custom) != 2 || got.Custom[0].TraitID != "a_leader" || strings.Join(got.Custom[0].Metadata.Types, ",") != "land" ||
		got.Custom[1].TraitID != "my_navy" || got.Custom[1].Metadata.Types[0] != "navy" || got.Custom[1].Metadata.TraitTypes[0] != "assignable_trait" {
		t.Fatalf("custom: %+v", got.Custom)
	}

	leaders := decode[knownTraitsResponse](t, e.json("GET", "/custom-traits/known", nil))
	if strings.Join(leaders.Vanilla, ",") != "brave_leader,cautious,odd_trait" || len(leaders.VanillaMetadata) != 0 ||
		len(leaders.Custom) != 1 || len(leaders.Custom[0].Metadata.Types) != 0 {
		t.Fatalf("country leaders: %+v", leaders)
	}
	scientists := decode[knownTraitsResponse](t, e.json("GET", "/custom-traits/known?trait_type=scientist", nil))
	if strings.Join(scientists.Vanilla, ",") != "brilliant_mind" || scientists.Custom == nil {
		t.Fatalf("scientists: %+v", scientists)
	}
	expect(t, e.json("GET", "/custom-traits/known?trait_type=general", nil), 400,
		"trait_type must be one of ['country_leader', 'scientist', 'unit_leader']")
	expect(t, e.json("GET", "/custom-traits/known?trait_type=", nil), 400, "trait_type must be one of")
}

func TestKnownTraitsWithoutTheGame(t *testing.T) {
	e := newEnv(t, nil)
	e.createTrait(map[string]any{"trait_id": "cl"})
	got := decode[knownTraitsResponse](t, e.json("GET", "/custom-traits/known?trait_type=country_leader", nil))
	if got.Vanilla == nil || len(got.Vanilla) != 0 || got.VanillaMetadata == nil || len(got.Custom) != 1 {
		t.Fatalf("no game: %+v", got)
	}
}

func TestCustomTraitIcon(t *testing.T) {
	e := newEnv(t, nil)
	leader := e.createTrait(map[string]any{"trait_id": "cl"})
	unit := e.createTrait(map[string]any{"trait_id": "Sea Wolf", "trait_type": "unit_leader"})
	body, ctype := multipartBody(t, map[string][]byte{"file": testPNG(t, 40, 40)}, nil)
	expect(t, e.do("POST", "/custom-traits/"+itoa(leader.ID)+"/upload-icon", ctype, body), 400,
		"Icon upload is only supported for unit_leader and scientist traits")
	expect(t, e.do("POST", "/custom-traits/999/upload-icon", ctype, body), 404, "Asset target not found")
	expect(t, e.json("GET", "/custom-traits/"+itoa(unit.ID)+"/icon", nil), 404, "Asset not uploaded")

	rec := e.do("POST", "/custom-traits/"+itoa(unit.ID)+"/upload-icon", ctype, body)
	expect(t, rec, 200, "")
	resp := decode[map[string]any](t, rec)
	if resp["filename"] != "customtrait_unit_leader_sea_wolf.png" || resp["sprite"] != "GFX_trait_Sea Wolf" ||
		resp["preview_url"] != "/api/go/projects/"+itoa(e.p.ID())+"/custom-traits/"+itoa(unit.ID)+"/icon" {
		t.Fatalf("upload response: %v", resp)
	}
	rec = e.json("GET", "/custom-traits/"+itoa(unit.ID)+"/icon", nil)
	if rec.Code != 200 {
		t.Fatalf("icon: %d %s", rec.Code, rec.Body.String())
	}
	if w, h := imageSize(t, rec.Body.Bytes()); w != 23 || h != 33 {
		t.Fatalf("icon is %dx%d", w, h)
	}
}
