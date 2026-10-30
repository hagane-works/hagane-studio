package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"haganeworks/internal/gameindex"
)

func checkSanity(t *testing.T, data map[string][]byte) {
	t.Helper()
	floor := func(name, key string, n int, landmarks ...string) {
		t.Helper()
		set := setOf(jsonList(t, data[name], key))
		if len(set) < n {
			t.Errorf("%s.%s: %d entries, want at least %d", name, key, len(set), n)
		}
		for _, l := range landmarks {
			if !set[l] {
				t.Errorf("%s.%s misses %q", name, key, l)
			}
		}
	}
	floor(gameindex.VanillaEffects, "ids", 500, "add_ideas", "country_event")
	floor(gameindex.VanillaTriggers, "ids", 500, "has_idea", "has_war")
	floor(gameindex.VanillaModifiers, "modifiers", 600, "political_power_gain")
	floor(gameindex.MathFunctions, "ids", 30, "less_than", "greater_than", "alpha")
	floor(gameindex.YesNoKeys, "ids", 2000, "instant_build", "keep_completed", "transfer_troops", "designer", "can_be_randomly_selected")
	floor(gameindex.StructuralFields, "fields", 1500, "is_triggered_only", "political_power_gain")
	floor(gameindex.IdeaIndex, "picture_sprites", 3000, "war_economy", "civilian_economy", "generic_fascism_drift_2")
	floor(gameindex.TechIDs, "ids", 500, "infantry_weapons")
	floor(gameindex.CharacterIDs, "ids", 5000, "ROM_ion_antonescu")
	floor(gameindex.EventIDs, "ids", 6000)
	floor(gameindex.DecisionCategories, "ids", 500)
	floor(gameindex.DynamicModifierIDs, "ids", 500)
	floor(gameindex.OpinionModifiers, "ids", 700)
	floor(gameindex.Traits, "ids", 2300, "brilliant_strategist")
	floor(gameindex.AutonomyStates, "ids", 15, "autonomy_dominion", "autonomy_puppet")
	floor(gameindex.UnitFiles, "ids", 40, "infantry.txt")
	floor(gameindex.FocusIcons, "names", 2500, "GFX_goal_generic_construct_civ_factory", "GFX_focus_usa_guarantee_the_american_dream")
	floor(gameindex.CollisionTable, "bookmarks", 2, "common/bookmarks/blitzkrieg.txt")
	floor(gameindex.EventPictures, "all_pictures", 800)
	floor(gameindex.WorldTables, "cosmetic_tags", 300, "RAJ_UK")

	var round2 struct {
		Aliases   map[string][]string          `json:"aliases"`
		States    map[string]json.RawMessage   `json:"states"`
		DLCNames  []string                     `json:"dlc_names"`
		Folders   map[string]techSectionFolder `json:"folders"`
		Subunits  map[string]map[string]string `json:"subunits"`
		Equipment map[string]map[string]string `json:"equipment"`
	}
	for _, name := range []string{gameindex.CountryAliases, gameindex.StateContent, gameindex.DLCStates, gameindex.TechSections, gameindex.UnitNames} {
		var version struct {
			GameVersion string `json:"game_version"`
		}
		if err := json.Unmarshal(data[name], &version); err != nil || version.GameVersion == "" {
			t.Errorf("%s: no game_version (%v)", name, err)
		}
		_ = json.Unmarshal(data[name], &round2)
		switch name {
		case gameindex.CountryAliases:
			if len(round2.Aliases) < 300 || len(round2.Aliases["GER"]) == 0 {
				t.Errorf("country aliases: %d tags, GER %v", len(round2.Aliases), round2.Aliases["GER"])
			}
		case gameindex.StateContent:
			if len(round2.States) < 500 || round2.States["2"] == nil {
				t.Errorf("state content: %d states", len(round2.States))
			}
		case gameindex.DLCStates:
			if len(round2.States) < 50 || len(round2.DLCNames) < 5 {
				t.Errorf("dlc states: %d states, names %v", len(round2.States), round2.DLCNames)
			}
		case gameindex.TechSections:
			infantry := round2.Folders["infantry_folder"]
			if len(round2.Folders) < 10 || infantry.Count < 3 || infantry.SubSections[0].Names["en"] == "" || infantry.SubSections[0].Names["zh"] == "" {
				t.Errorf("tech sections: %d folders, infantry %+v", len(round2.Folders), infantry)
			}
		case gameindex.UnitNames:
			if len(round2.Subunits) < 100 || round2.Subunits["infantry"]["en"] != "Infantry" || round2.Subunits["infantry"]["zh"] == "" ||
				len(round2.Equipment) < 200 || round2.Equipment["infantry_equipment"]["en"] == "" {
				t.Errorf("unit names: %d sub-units (infantry %v), %d equipment", len(round2.Subunits), round2.Subunits["infantry"], len(round2.Equipment))
			}
		}
	}

	var trees struct {
		PrimaryPaths map[string]string `json:"primary_paths"`
	}
	_ = json.Unmarshal(data[gameindex.FocusTrees], &trees)
	if trees.PrimaryPaths["finnish_focus"] != "common/national_focus/finland.txt" {
		t.Errorf("focus registry landmark: %q", trees.PrimaryPaths["finnish_focus"])
	}
	var tables struct {
		CountryFilenames map[string]string         `json:"country_filenames"`
		StateOwners      map[string]string         `json:"state_owners"`
		Cultures         map[string]map[string]any `json:"country_cultures"`
		Ideology         string                    `json:"ideology_base_text"`
	}
	_ = json.Unmarshal(data[gameindex.WorldTables], &tables)
	if tables.CountryFilenames["GER"] != "Germany.txt" || tables.CountryFilenames["BRN"] != "Brunei.txt" ||
		tables.StateOwners["64"] != "GER" || tables.Cultures["GER"]["graphical_culture"] != "western_european_gfx" ||
		!strings.Contains(tables.Ideology, "democratic") {
		t.Errorf("world tables landmarks: GER=%q BRN=%q 64=%q", tables.CountryFilenames["GER"], tables.CountryFilenames["BRN"], tables.StateOwners["64"])
	}
	var cosmetic struct {
		Bytes []byte `json:"bytes"`
	}
	_ = json.Unmarshal(data[gameindex.Cosmetic], &cosmetic)
	if len(cosmetic.Bytes) < 10000 {
		t.Errorf("cosmetic.txt: %d bytes", len(cosmetic.Bytes))
	}
	var dlc struct {
		DLCs []struct{ Name, Kind string } `json:"dlcs"`
	}
	_ = json.Unmarshal(data[gameindex.DLCNames], &dlc)
	kinds := map[string]string{}
	for _, d := range dlc.DLCs {
		kinds[d.Name] = d.Kind
	}
	if kinds["La Resistance"] != "expansion" || kinds["Together for Victory"] != "expansion" {
		t.Errorf("dlc roster landmarks: %v", kinds)
	}
	var countries struct {
		Tags []countryTag `json:"tags"`
	}
	_ = json.Unmarshal(data[gameindex.CountryTags], &countries)
	germany := false
	for _, c := range countries.Tags {
		if c.Tag == "GER" {
			germany = c.NameEn != "" && c.Names["zh"] != "" && c.Continent == "europe"
		}
	}
	if !germany || len(countries.Tags) < 300 {
		t.Errorf("country tags: %d, GER complete %v", len(countries.Tags), germany)
	}
	var whitelist map[string]json.RawMessage
	_ = json.Unmarshal(data[gameindex.CwtoolWhitelist], &whitelist)
	for _, key := range []string{"version", "anti_patterns", "field_enums", "known_effects", "known_triggers", "known_modifiers", "cross_validators"} {
		if len(whitelist[key]) == 0 {
			t.Errorf("cwtool whitelist misses %s", key)
		}
	}
}

func reference(t *testing.T, dir, rel string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Logf("reference %s: %v (skipped)", rel, err)
		return nil
	}
	return raw
}

func differ(t *testing.T, label string, got, want []string) int {
	t.Helper()
	g, w := setOf(got), setOf(want)
	var added, removed []string
	for k := range g {
		if !w[k] {
			added = append(added, k)
		}
	}
	for k := range w {
		if !g[k] {
			removed = append(removed, k)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	if len(added) == 0 && len(removed) == 0 {
		t.Logf("%-34s identical (%d)", label, len(g))
		return 0
	}
	t.Logf("%-34s index %d, bundled %d: +%d %v  -%d %v", label, len(g), len(w), len(added), head(added), len(removed), head(removed))
	return len(removed)
}

func head(list []string) []string {
	if len(list) > 8 {
		return append(append([]string{}, list[:8]...), "…")
	}
	return list
}

func jsonList(t *testing.T, raw []byte, path ...string) []string {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("%v: %v", path, err)
	}
	for _, p := range path {
		m, _ := v.(map[string]any)
		v = m[p]
	}
	var out []string
	switch v := v.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
				continue
			}
			b, _ := json.Marshal(item)
			out = append(out, string(b))
		}
	case map[string]any:
		for k, item := range v {
			b, _ := json.Marshal(item)
			out = append(out, k+"="+string(b))
		}
	}
	return out
}

func compareWithReference(t *testing.T, data map[string][]byte, dir string) {
	t.Helper()

	olderGame := map[string]bool{"cwtool field archetype": true, "equipment archetypes (web UI)": true, "unit-names equipment tokens": true}
	check := func(label string, got, want []string) {
		t.Helper()
		if want == nil {
			return
		}
		if removed := differ(t, label, got, want); removed*10 > len(want) && !olderGame[label] {
			t.Errorf("%s lost %d of %d bundled entries", label, removed, len(want))
		}
	}
	type pair struct {
		label, dataset string
		path           []string
		rel            string
		refPath        []string
	}
	for _, p := range []pair{
		{"effects", gameindex.VanillaEffects, []string{"ids"}, "internal/hoi4/vanilladata/tokens_vanilla_effects.json", []string{"ids"}},
		{"triggers", gameindex.VanillaTriggers, []string{"ids"}, "internal/hoi4/vanilladata/tokens_vanilla_triggers.json", []string{"ids"}},
		{"structural fields", gameindex.StructuralFields, []string{"fields"}, "internal/hoi4/vanilladata/tokens_entity_structural_fields.json", []string{"fields"}},
		{"yes/no keys", gameindex.YesNoKeys, []string{"ids"}, "internal/hoi4/vanilladata/tokens_vanilla_yesno_keys.json", []string{"ids"}},
		{"idea picture sprites", gameindex.IdeaIndex, []string{"picture_sprites"}, "internal/hoi4/vanilladata/vanilla_idea_index.json", []string{"picture_sprites"}},
		{"idea picture orphans", gameindex.IdeaIndex, []string{"picture_sprites_orphans"}, "internal/hoi4/vanilladata/vanilla_idea_index.json", []string{"picture_sprites_orphans"}},
		{"focus icon names", gameindex.FocusIcons, []string{"names"}, "internal/hoi4/exportengine/focus_vanilla_icons.json", []string{"names"}},
		{"focus primary paths", gameindex.FocusTrees, []string{"primary_paths"}, "internal/hoi4/exportengine/focus_vanilla_trees.json", []string{"primary_paths"}},
		{"collision icons", gameindex.CollisionTable, []string{"icons"}, "internal/hoi4/exportengine/vanilla_collision_table.json", []string{"icons"}},
		{"collision bookmarks", gameindex.CollisionTable, []string{"bookmarks"}, "internal/hoi4/exportengine/vanilla_collision_table.json", []string{"bookmarks"}},
		{"tech icon sizes", gameindex.TechIconSizes, []string{"sizes"}, "internal/hoi4/importassets/vanilla_tech_icon_sizes.json", []string{"sizes"}},
		{"world vanilla_tags", gameindex.WorldTables, []string{"vanilla_tags"}, "internal/hoi4/emit/worldout/tables.json", []string{"vanilla_tags"}},
		{"world state_filenames", gameindex.WorldTables, []string{"state_filenames"}, "internal/hoi4/emit/worldout/tables.json", []string{"state_filenames"}},
		{"world state_owners", gameindex.WorldTables, []string{"state_owners"}, "internal/hoi4/emit/worldout/tables.json", []string{"state_owners"}},
		{"world country_filenames", gameindex.WorldTables, []string{"country_filenames"}, "internal/hoi4/emit/worldout/tables.json", []string{"country_filenames"}},
		{"world country_cultures", gameindex.WorldTables, []string{"country_cultures"}, "internal/hoi4/emit/worldout/tables.json", []string{"country_cultures"}},
		{"world cosmetic_tags", gameindex.WorldTables, []string{"cosmetic_tags"}, "internal/hoi4/emit/worldout/tables.json", []string{"cosmetic_tags"}},
		{"cwtool known_effects", gameindex.CwtoolWhitelist, []string{"known_effects"}, "internal/domains/tools/cwtool_whitelist.json", []string{"known_effects"}},
		{"cwtool known_triggers", gameindex.CwtoolWhitelist, []string{"known_triggers"}, "internal/domains/tools/cwtool_whitelist.json", []string{"known_triggers"}},
		{"cwtool known_modifiers", gameindex.CwtoolWhitelist, []string{"known_modifiers"}, "internal/domains/tools/cwtool_whitelist.json", []string{"known_modifiers"}},
		{"idea categories (registry)", gameindex.IdeaTags, []string{"idea_categories"}, "internal/hoi4/exportengine/registry_validation_cwtools.json", []string{"idea_categories"}},
		{"idea slots (registry)", gameindex.IdeaTags, []string{"idea_slots"}, "internal/hoi4/exportengine/registry_validation_cwtools.json", []string{"idea_slots"}},
		{"ideas UI all_ids", gameindex.IdeasUI, []string{"all_ids"}, "web/src/lib/vanilla_ideas.json", []string{"all_ids"}},
	} {
		ref := reference(t, dir, p.rel)
		if ref == nil {
			continue
		}
		check(p.label, jsonList(t, data[p.dataset], p.path...), jsonList(t, ref, p.refPath...))
	}
	if ref := reference(t, dir, "internal/domains/tools/cwtool_whitelist.json"); ref != nil {
		var got, want struct {
			FieldEnums map[string]struct {
				Whitelist []string `json:"whitelist"`
			} `json:"field_enums"`
		}
		_ = json.Unmarshal(data[gameindex.CwtoolWhitelist], &got)
		_ = json.Unmarshal(ref, &want)
		for _, name := range sortedKeys(want.FieldEnums) {
			check("cwtool field "+name, got.FieldEnums[name].Whitelist, want.FieldEnums[name].Whitelist)
		}
	}
	if ref := reference(t, dir, "internal/hoi4/vanilladata/vanilla_idea_index.json"); ref != nil {
		rows := func(raw []byte) []string {
			var out []string
			for _, idea := range ideaIndex(raw) {
				out = append(out, idea.Category+"/"+idea.IdeaID)
			}
			return out
		}
		check("ideas (category/id)", rows(data[gameindex.IdeaIndex]), rows(ref))
	}
	if ref := reference(t, dir, "internal/hoi4/exportengine/cross_ref_vanilla_ids.json"); ref != nil {
		for _, kind := range []string{"idea", "event", "trait", "technology", "character", "opinion_modifier", "dynamic_modifier"} {
			check("cross-ref "+kind, jsonList(t, data[gameindex.CrossRefIDs], "ids", kind), jsonList(t, ref, "ids", kind))
		}
	}
	if ref := reference(t, dir, "internal/hoi4/vanilladata/dlc_names.json"); ref != nil {
		names := func(raw []byte) []string {
			var f struct {
				DLCs []struct{ Name string } `json:"dlcs"`
			}
			_ = json.Unmarshal(raw, &f)
			var out []string
			for _, d := range f.DLCs {
				out = append(out, d.Name)
			}
			return out
		}
		check("dlc names", names(data[gameindex.DLCNames]), names(ref))
	}
	if ref := reference(t, dir, "internal/hoi4/emit/worldout/tables.json"); ref != nil {
		var got, want struct {
			Text string `json:"ideology_base_text"`
		}
		_ = json.Unmarshal(data[gameindex.WorldTables], &got)
		_ = json.Unmarshal(ref, &want)
		t.Logf("%-34s identical %v (%d / %d bytes)", "world ideology_base_text", got.Text == want.Text, len(got.Text), len(want.Text))
	}
	if ref := reference(t, dir, "internal/hoi4/cosmetic/vanilla_cosmetic_1_19_2.txt"); ref != nil {
		var got struct {
			Bytes []byte `json:"bytes"`
		}
		_ = json.Unmarshal(data[gameindex.Cosmetic], &got)
		t.Logf("%-34s identical %v (%d / %d bytes)", "cosmetic.txt", string(got.Bytes) == string(ref), len(got.Bytes), len(ref))
	}

	source := func(rel string) string { return string(reference(t, dir, rel)) }
	quoted := func(text string) []string {
		var out []string
		for _, m := range regexp.MustCompile(`"([A-Za-z_0-9.\-]+)"`).FindAllStringSubmatch(text, -1) {
			out = append(out, m[1])
		}
		return out
	}
	block := func(text, start string) string {
		i := strings.Index(text, start)
		if i < 0 {
			return ""
		}
		return text[i : i+strings.Index(text[i:], "\n}\n")]
	}
	if text := source("internal/hoi4/importjob/events_world.go"); text != "" {
		if m := regexp.MustCompile(`const eventCountryAliasSpec = "([^"]*)"`).FindStringSubmatch(text); m != nil {
			aliases := countryAliasesOf(data[gameindex.CountryAliases])
			var got []string
			for _, tag := range sortedKeys(aliases.Aliases) {
				got = append(got, tag+":"+strings.Join(aliases.Aliases[tag], ","))
			}
			check("country aliases", got, strings.Split(m[1], ";"))
		}
	}
	if text := source("internal/hoi4/importjob/focus_policy_registry_gen.go"); text != "" {
		var ids []string
		for _, m := range regexp.MustCompile(`\t"([a-z_]+)":\s+"common/national_focus/`).FindAllStringSubmatch(text, -1) {
			ids = append(ids, m[1])
		}
		check("importjob focus registry ids", jsonList(t, data[gameindex.FocusTrees], "ids"), ids)
	}
	if text := source("internal/hoi4/exportengine/cross_ref_lint.go"); text != "" {
		check("tech categories", jsonList(t, data[gameindex.TechTags], "categories", "ids"), quoted(block(text, "var vanillaTechCategories")))
		check("autonomy states", jsonList(t, data[gameindex.AutonomyStates], "ids"), quoted(block(text, "var vanillaAutonomyStates")))
	}
	if text := source("internal/domains/tools/exportcheck/export_gate_leaves.go"); text != "" {
		check("unit files", jsonList(t, data[gameindex.UnitFiles], "ids"), quoted(block(text, "var vanillaUnitFilenames")))
	}
	if text := source("web/src/lib/vanillaEquipmentArchetypes.ts"); text != "" {
		var want []string
		for _, m := range regexp.MustCompile(`\n  "([A-Za-z_0-9]+)",`).FindAllStringSubmatch(text, -1) {
			want = append(want, m[1])
		}
		differ(t, "equipment archetypes (web UI)", jsonList(t, data[gameindex.Equipment], "archetypes"), want)
	}
	if text := source("web/src/lib/vanillaCountryTags.ts"); text != "" {
		var want []string
		for _, m := range regexp.MustCompile(`tag: "(\w+)"`).FindAllStringSubmatch(text, -1) {
			want = append(want, m[1])
		}
		var tags struct {
			Tags []countryTag `json:"tags"`
		}
		_ = json.Unmarshal(data[gameindex.CountryTags], &tags)
		var got []string
		for _, c := range tags.Tags {
			got = append(got, c.Tag)
		}
		differ(t, "country tags (web UI curated list)", got, want)
	}
	compareWebTables(t, data, dir, check)
}

func between(text, start, end string) string {
	i := strings.Index(text, start)
	if i < 0 {
		return ""
	}
	j := strings.Index(text[i:], end)
	if j < 0 {
		return text[i:]
	}
	return text[i : i+j]
}

func canonical(t *testing.T, raw []byte) string {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
	out, _ := json.Marshal(v)
	return string(out)
}

var (
	tsAliasRe      = regexp.MustCompile(`(?m)^  ([A-Z0-9]{3}): \[([^\]]*)\],`)
	tsQuotedRe     = regexp.MustCompile(`"([^"]*)"`)
	tsStateRe      = regexp.MustCompile(`(?m)^  (\d+): (\{.*\}),\s*$`)
	tsDLCStateRe   = regexp.MustCompile(`(?m)^  (\d+): (\[.*\]),\s*$`)
	tsObjectKeyRe  = regexp.MustCompile(`([{,]\s*)([A-Za-z]+):`)
	tsGlossEntryRe = regexp.MustCompile(`(?m)^  "?([A-Za-z_0-9]+)"?: "([^"]*)",`)
)

func compareWebTables(t *testing.T, data map[string][]byte, dir string, check func(label string, got, want []string)) {
	t.Helper()
	source := func(rel string) string { return string(reference(t, dir, rel)) }

	if text := between(source("web/src/lib/country-aliases.ts"), "export const COUNTRY_ALIASES", "\n};"); text != "" {
		var want []string
		for _, m := range tsAliasRe.FindAllStringSubmatch(text, -1) {
			var names []string
			for _, q := range tsQuotedRe.FindAllStringSubmatch(m[2], -1) {
				names = append(names, q[1])
			}
			want = append(want, m[1]+":"+strings.Join(names, ","))
		}
		aliases := countryAliasesOf(data[gameindex.CountryAliases])
		var got []string
		for _, tag := range sortedKeys(aliases.Aliases) {
			got = append(got, tag+":"+strings.Join(aliases.Aliases[tag], ","))
		}
		check("country-aliases (web UI)", got, want)
	}

	if text := between(source("web/src/lib/data/vanillaStateContent.ts"), "export const VANILLA_STATE_CONTENT:", "\n};"); text != "" {
		var want []string
		for _, m := range tsStateRe.FindAllStringSubmatch(text, -1) {
			want = append(want, m[1]+"="+canonical(t, []byte(tsObjectKeyRe.ReplaceAllString(m[2], `$1"$2":`))))
		}
		check("state-content states", jsonList(t, data[gameindex.StateContent], "states"), want)
	}

	if text := source("web/src/lib/data/vanillaDlcStates.ts"); text != "" {
		var names, states []string
		for _, q := range tsQuotedRe.FindAllStringSubmatch(between(text, "export const VANILLA_DLC_NAMES", "\n];"), -1) {
			names = append(names, q[1])
		}
		for _, m := range tsDLCStateRe.FindAllStringSubmatch(between(text, "export const VANILLA_DLC_STATES", "\n};"), -1) {
			states = append(states, m[1]+"="+canonical(t, []byte(m[2])))
		}
		check("dlc-states dlc_names", jsonList(t, data[gameindex.DLCStates], "dlc_names"), names)
		check("dlc-states states", jsonList(t, data[gameindex.DLCStates], "states"), states)
	}

	if ref := reference(t, dir, "web/src/data/visual_section_mapping.json"); ref != nil {
		type section struct {
			Name    string            `json:"name"`
			SubKey  string            `json:"sub_key"`
			Y       int               `json:"y"`
			TextKey string            `json:"text_key"`
			TextEn  string            `json:"text_en"`
			TextZh  string            `json:"text_zh"`
			Names   map[string]string `json:"names"`
		}
		type folders struct {
			Folders map[string]struct {
				Count       int       `json:"count"`
				SubSections []section `json:"sub_sections"`
			} `json:"folders"`
		}
		var got, want folders
		_ = json.Unmarshal(data[gameindex.TechSections], &got)
		_ = json.Unmarshal(ref, &want)
		rows := func(f folders) []string {
			var out []string
			for name, folder := range f.Folders {
				out = append(out, fmt.Sprintf("%s count=%d", name, folder.Count))
				for i, s := range folder.SubSections {
					out = append(out, fmt.Sprintf("%s#%d %s %s y=%d %s", name, i, s.Name, s.SubKey, s.Y, s.TextKey))
				}
			}
			return out
		}
		check("tech-sections structure", rows(got), rows(want))
		titles, changed := 0, 0
		for _, name := range sortedKeys(want.Folders) {
			for i, old := range want.Folders[name].SubSections {
				if i >= len(got.Folders[name].SubSections) {
					break
				}
				s := got.Folders[name].SubSections[i]
				titles++
				for _, lang := range [][3]string{{"en", old.TextEn, s.Names["en"]}, {"zh", old.TextZh, s.Names["zh"]}} {
					if lang[1] != lang[2] {
						changed++
						t.Logf("  tech-sections %s %s %s: bundled %q, game %q", name, s.TextKey, lang[0], lang[1], lang[2])
					}
				}
			}
		}
		t.Logf("%-34s %d titles, %d en/zh texts differ", "tech-sections names", titles, changed)
	}

	glosses := []struct {
		label, rel, start, key string
	}{
		{"unit-names sub-unit", "web/src/lib/i18n/subunitGloss.ts", "export const subunitGlossZh", "subunits"},
		{"unit-names equipment", "web/src/lib/i18n/equipmentGloss.ts", "export const equipmentArchetypeGlossZh", "equipment"},
	}
	var names map[string]map[string]map[string]string
	_ = json.Unmarshal(data[gameindex.UnitNames], &names)
	for _, g := range glosses {
		text := between(source(g.rel), g.start, "\n};")
		if text == "" {
			continue
		}
		gloss := map[string]string{}
		for _, m := range tsGlossEntryRe.FindAllStringSubmatch(text, -1) {
			gloss[m[1]] = m[2]
		}
		index := names[g.key]
		check(g.label+" tokens", sortedKeys(index), sortedKeys(gloss))
		same := 0
		for _, token := range sortedKeys(gloss) {
			entry, ok := index[token]
			switch {
			case !ok:
			case entry["zh"] == gloss[token]:
				same++
			default:
				t.Logf("  %s %s: bundled %q, game %q", g.label, token, gloss[token], entry["zh"])
			}
		}
		t.Logf("%-34s %d of %d bundled zh names equal the game's", g.label+" zh", same, len(gloss))
	}
}
