package states

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/paradox"
)

func normaliseAny(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func rawJSONObjectKeys(raw json.RawMessage) []string {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var keys []string
	if tok, err := decoder.Token(); err != nil || tok != json.Delim('{') {
		return keys
	}
	for decoder.More() {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if key, ok := tok.(string); ok {
			keys = append(keys, key)
			var skip json.RawMessage
			if err := decoder.Decode(&skip); err != nil {
				return keys
			}
		}
	}
	return keys
}

func TestStateBuildingsParity(t *testing.T) {
	raw, err := os.ReadFile("testdata/state_buildings_parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Deserialize []struct {
			Input  string         `json:"input"`
			Output map[string]any `json:"output"`
		} `json:"deserialize"`
		Serialize []struct {
			PB       map[string]any `json:"pb"`
			Existing string         `json:"existing"`
			Output   string         `json:"output"`
		} `json:"serialize"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	var orders struct {
		Serialize []struct {
			PB json.RawMessage `json:"pb"`
		} `json:"serialize"`
	}
	if err := json.Unmarshal(raw, &orders); err != nil {
		t.Fatal(err)
	}
	if len(fx.Deserialize) == 0 || len(fx.Serialize) == 0 {
		t.Fatal("empty fixture")
	}
	for _, c := range fx.Deserialize {
		if got, want := normaliseAny(t, deserializeProvinceBuildings(c.Input)), normaliseAny(t, c.Output); !reflect.DeepEqual(got, want) {
			t.Errorf("deserialize(%q): got %v want %v", c.Input, got, want)
		}
	}
	for i, c := range fx.Serialize {
		if got := serializeProvinceBuildings(c.PB, rawJSONObjectKeys(orders.Serialize[i].PB), c.Existing); got != c.Output {
			t.Errorf("serialize[%d]: got %q want %q", i, got, c.Output)
		}
	}
}

func TestStateSnapshotAndProvenance(t *testing.T) {
	p := membershipProvenance([]any{float64(3), float64(1)}, "GER", "", nil)
	if p["status"] != "legacy_unknown" || p["membership_changed"] != nil {
		t.Fatalf("legacy: %v", p)
	}
	if !reflect.DeepEqual(p["current"], []int{1, 3}) {
		t.Fatalf("current sorted: %v", p["current"])
	}
	snap, err := captureStateSourceSnapshot(500, []any{float64(1), float64(2)}, "GER", nil, "history/states/500-x.txt")
	if err != nil {
		t.Fatal(err)
	}
	snapMap, _ := plain(snap).(map[string]any)
	p2 := membershipProvenance([]any{json.Number("1"), json.Number("3")}, "ITA", "", snapMap)
	if p2["status"] != "known" || p2["membership_changed"] != true || p2["owner_changed"] != true || p2["controller_changed"] != false {
		t.Fatalf("known: %v", p2)
	}
	if !reflect.DeepEqual(p2["added"], []int{3}) || !reflect.DeepEqual(p2["removed"], []int{2}) {
		t.Fatalf("delta: added=%v removed=%v", p2["added"], p2["removed"])
	}
	if normalizeStateSourceSnapshot(map[string]any{"schema_version": "wrong"}) != nil {
		t.Fatal("wrong schema must normalize to nil")
	}
}

func TestResolveDisplayLocs(t *testing.T) {
	names := map[string]map[string]string{"1": {"en": "Corsica", "zh": "科西嘉"}}
	out := resolveDisplayLocs(names, 1, map[string]any{"zh": " 华北 ", "ru": "Корсика", "de": "  ", "pl": nil})
	want := map[string]string{"en": "Corsica", "zh": "华北", "ru": "Корсика"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("resolveDisplayLocs = %v, want %v", out, want)
	}
	if none := resolveDisplayLocs(names, 99, nil); len(none) != 0 {
		t.Fatalf("unknown state must omit: %v", none)
	}
}

func TestStateReadVanillaNamesNeedProvenance(t *testing.T) {
	names := map[string]map[string]string{"1": {"en": "Corsica"}}
	base := map[string]any{
		"state_id": json.Number("1"), "display_name_locs": map[string]any{"x-test": " Winterfell "},
		"source": "created", "source_file": "", "source_snapshot": nil, "raw_extras": map[string]any{}, "provinces": []any{},
	}
	got := stateRead(base, names)["display_name"].(map[string]string)
	if !reflect.DeepEqual(got, map[string]string{"x-test": "Winterfell"}) {
		t.Fatalf("generated state display_name = %v", got)
	}
	imported := cloneRow(base)
	imported["source"] = "imported"
	got = stateRead(imported, names)["display_name"].(map[string]string)
	if got["en"] != "Corsica" || got["x-test"] != "Winterfell" {
		t.Fatalf("imported state display_name = %v", got)
	}
}

func TestProjectStateDisplayNamesUsesPreservedLocalisation(t *testing.T) {
	states := []map[string]any{{
		"name": "STATE_7", "state_id": json.Number("7"), "display_name": map[string]string{"en": "Game", "zh": "游戏"},
		"display_name_locs": map[string]any{"fr": "Auteur"},
	}}
	locRows := []map[string]any{
		{"id": json.Number("1"), "key": "STATE_7", "locale": "l_english", "value": "Old", "source_file": "localisation/english/mod_l_english.yml", "replace_priority": json.Number("0"), "source_order": json.Number("0"), "entry_order": json.Number("0")},
		{"id": json.Number("2"), "key": "STATE_7", "locale": "l_english", "value": "Newer", "source_file": "localisation/english/mod_l_english.yml", "replace_priority": json.Number("1"), "source_order": json.Number("0"), "entry_order": json.Number("0")},
		{"id": json.Number("3"), "key": "STATE_7", "locale": "l_simp_chinese", "value": "", "source_file": "localisation/simp_chinese/mod_l_simp_chinese.yml", "replace_priority": json.Number("0"), "source_order": json.Number("0"), "entry_order": json.Number("0")},
		{"id": json.Number("4"), "key": "STATE_7", "locale": "l_english", "value": "Unsafe", "source_file": "../escape.yml", "replace_priority": json.Number("9"), "source_order": json.Number("0"), "entry_order": json.Number("0")},
	}
	projectStateDisplayNames(locRows, states)
	got := states[0]["display_name"].(map[string]string)
	want := map[string]string{"en": "Newer", "fr": "Auteur"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("display_name = %v, want %v", got, want)
	}
}

func eqTags(t *testing.T, got, want []string, label string) {
	t.Helper()
	if !reflect.DeepEqual(append([]string{}, got...), append([]string{}, want...)) {
		t.Fatalf("%s: got %v want %v", label, got, want)
	}
}

func TestApplyCoreOp(t *testing.T) {
	got, removed := applyCoreOp([]string{"GXC", "CHI"}, stateBulkCores{Mode: coreModeAdd, Tags: []string{"PRC", "CHI"}})
	eqTags(t, got, []string{"GXC", "CHI", "PRC"}, "add")
	if len(removed) != 0 {
		t.Fatalf("add removed %v", removed)
	}
	got, removed = applyCoreOp([]string{"GXC", "CHI", "PRC"}, stateBulkCores{Mode: coreModeRemove, Tags: []string{"CHI", "NOT_THERE"}})
	eqTags(t, got, []string{"GXC", "PRC"}, "remove")
	eqTags(t, removed, []string{"CHI"}, "remove.removed")
	got, removed = applyCoreOp([]string{"GXC", "CHI", "PRC"}, stateBulkCores{Mode: coreModeReplaceOne, From: "CHI", Tags: []string{"MEN"}})
	eqTags(t, got, []string{"GXC", "MEN", "PRC"}, "replace_one")
	eqTags(t, removed, []string{"CHI"}, "replace_one.removed")
	got, removed = applyCoreOp([]string{"GXC", "PRC"}, stateBulkCores{Mode: coreModeReplaceOne, From: "CHI", Tags: []string{"MEN"}})
	eqTags(t, got, []string{"GXC", "PRC"}, "replace_one absent")
	if len(removed) != 0 {
		t.Fatalf("absent from removed %v", removed)
	}
	got, _ = applyCoreOp([]string{"GXC", "CHI", "PRC"}, stateBulkCores{Mode: coreModeReplaceOne, From: "CHI", Tags: []string{"PRC"}})
	eqTags(t, got, []string{"GXC", "PRC"}, "replace_one dedup")
	got, removed = applyCoreOp([]string{"GXC", "CHI", "PRC"}, stateBulkCores{Mode: coreModeReplaceAll, Tags: []string{"SOV"}})
	eqTags(t, got, []string{"SOV"}, "replace_all")
	eqTags(t, removed, []string{"GXC", "CHI", "PRC"}, "replace_all.removed")
	got, _ = applyCoreOp([]string{" GXC ", "GXC"}, stateBulkCores{Mode: coreModeAdd, Tags: []string{" PRC ", "PRC", ""}})
	eqTags(t, got, []string{"GXC", "PRC"}, "trim/dedupe")
}

func TestValidateBulkInput(t *testing.T) {
	owner := "SOV"
	cases := []struct {
		name string
		in   stateBulkInput
		want string
	}{
		{"empty ids", stateBulkInput{Owner: &owner}, "state_db_ids must not be empty"},
		{"nothing to change", stateBulkInput{StateDBIDs: []int64{1}}, "nothing to change"},
		{"owner only", stateBulkInput{StateDBIDs: []int64{1}, Owner: &owner}, ""},
		{"unknown mode", stateBulkInput{StateDBIDs: []int64{1}, Cores: &stateBulkCores{Mode: "clobber", Tags: []string{"SOV"}}}, "cores.mode must be"},
		{"add without tags", stateBulkInput{StateDBIDs: []int64{1}, Cores: &stateBulkCores{Mode: coreModeAdd}}, "cores.tags must not be empty"},
		{"replace_one without from", stateBulkInput{StateDBIDs: []int64{1}, Claims: &stateBulkCores{Mode: coreModeReplaceOne, Tags: []string{"PRC"}}}, "claims.from is required"},
		{"oversize", stateBulkInput{StateDBIDs: make([]int64, stateBulkMaxIDs+1), Owner: &owner}, "exceeds the 2000 limit"},
	}
	for _, c := range cases {
		err := validateBulkInput(c.in)
		if c.want == "" {
			if err != nil {
				t.Errorf("%s: %v", c.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(string(normaliseJSON(t, err)), c.want) {
			t.Errorf("%s: got %v, want %q", c.name, err, c.want)
		}
	}
}

func normaliseJSON(t *testing.T, err error) []byte {
	t.Helper()
	raw, _ := json.Marshal(err)
	return raw
}

func parseStateBlock(t *testing.T, text string) *paradox.Block {
	t.Helper()
	block, _, err := paradox.Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range block.Entries {
		if ent.Key != nil && *ent.Key == "state" && ent.Block != nil {
			return ent.Block
		}
	}
	t.Fatal("no state block")
	return nil
}

const shapeFixture = `
state = {
	id = 9999
	buildings = {
		industrial_complex = 3
		1234 = {
			naval_base = 5
			bunker = 2
			coastal_bunker = 1
			dam = 1
			landmark_test_monument = { level = 1 }
		}
		5678 = { naval_base = 1 }
	}
	history = {
		owner = TST
		1939.1.1 = {
			buildings = { 9012 = { naval_base = 2 } }
			1940.6.1 = { owner = OTH }
		}
		force_link_ownership_to = 42
	}
}
`

func TestScanVanillaStateContent(t *testing.T) {
	got := scanVanillaStateContent(parseStateBlock(t, shapeFixture))
	if got.NavalBases != 3 || got.Bunkers != 1 || got.CoastalBunkers != 1 || got.Dams != 1 {
		t.Errorf("counts %+v", got)
	}
	if !reflect.DeepEqual(got.Landmarks, []string{"landmark_test_monument"}) || got.DatedBlocks != 2 || !got.ForceLink {
		t.Errorf("landmarks/dated/force link %+v", got)
	}
	gated := scanVanillaStateContent(parseStateBlock(t, `state = { id = 8888 history = { buildings = { 4321 = { if = { limit = { has_dlc = "Test Pack" } naval_base = 3 } } } } }`))
	if gated.NavalBases != 1 || gated.HasDLC != 1 {
		t.Errorf("gated %+v", gated)
	}
	unnamed := scanVanillaStateContent(parseStateBlock(t, `state = { id = 6666 history = {
		set_compliance = 20 set_resistance = 10 1939.1.1 = { start_resistance = yes }
		buildings = { 4799 = { dam_mountain = 1 naval_headquarters = { level = 1 allowed = { has_dlc = "Test Pack" } } } } } }`))
	if unnamed.OtherContent != 2 || unnamed.HasDLC != 1 || unnamed.Resistance != 3 || unnamed.Dams != 0 {
		t.Errorf("unnamed %+v", unnamed)
	}
}

func TestBookmarkQualityParity(t *testing.T) {
	raw, err := os.ReadFile("testdata/bookmark_quality_parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name   string `json:"name"`
		Inputs []struct {
			BookmarkID     string `json:"bookmark_id"`
			Path           string `json:"path"`
			Source         string `json:"source"`
			Default        bool   `json:"default"`
			DefaultCountry string `json:"default_country"`
			Countries      []struct {
				Tag         string   `json:"tag"`
				Ideology    *string  `json:"ideology"`
				Labels      []string `json:"labels"`
				Interesting bool     `json:"interesting"`
			} `json:"countries"`
			Filters       []string `json:"filters"`
			TargetVersion string   `json:"target_version"`
			Emitted       bool     `json:"emitted"`
		} `json:"inputs"`
		Result struct {
			TargetVersion string   `json:"target_version"`
			Findings      []any    `json:"findings"`
			BlockerCodes  []string `json:"blocker_codes"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil || len(cases) == 0 {
		t.Fatalf("fixture: %v", err)
	}
	for _, c := range cases {
		inputs := []bookmarkQualityInput{}
		for _, i := range c.Inputs {
			countries := []bookmarkCountryInput{}
			for _, country := range i.Countries {
				ideology := ""
				if country.Ideology != nil {
					ideology = *country.Ideology
				}
				countries = append(countries, bookmarkCountryInput{Tag: country.Tag, Ideology: ideology, Labels: country.Labels, Interesting: country.Interesting})
			}
			inputs = append(inputs, bookmarkQualityInput{
				BookmarkID: i.BookmarkID, Path: i.Path, Source: i.Source, Default: i.Default,
				DefaultCountry: i.DefaultCountry, Countries: countries, Filters: i.Filters,
				TargetVersion: i.TargetVersion, Emitted: i.Emitted,
			})
		}
		result := evaluateBookmarkQuality(inputs)
		if result.TargetVersion != c.Result.TargetVersion {
			t.Errorf("[%s] target_version %q want %q", c.Name, result.TargetVersion, c.Result.TargetVersion)
		}
		if got, want := normaliseAny(t, result.Findings), normaliseAny(t, c.Result.Findings); !reflect.DeepEqual(got, want) {
			t.Errorf("[%s] findings:\n got %v\nwant %v", c.Name, got, want)
		}
		blockers := []string{}
		for _, f := range result.Findings {
			if f.Severity == "blocker" {
				blockers = append(blockers, f.Code)
			}
		}
		if len(blockers) != 0 || len(c.Result.BlockerCodes) != 0 {
			if !reflect.DeepEqual(blockers, c.Result.BlockerCodes) {
				t.Errorf("[%s] blockers %v want %v", c.Name, blockers, c.Result.BlockerCodes)
			}
		}
	}
}

func TestNormalizeBookmarkCountries(t *testing.T) {
	out := normalizeBookmarkCountries([]map[string]any{
		{"tag": "GER", "history_en": "old english", "history_locs": map[string]any{"zh": " 中文钩子 ", "fr": ""}},
		{"tag": "USA", "history_en": "", "history_zh": ""},
	})
	locs, _ := out[0]["history_locs"].(map[string]string)
	if locs["en"] != "old english" || locs["zh"] != "中文钩子" || locs["fr"] != "" {
		t.Fatalf("merged locs: %v", locs)
	}
	if out[0]["history_en"] != "old english" || out[0]["history_zh"] != "中文钩子" {
		t.Fatalf("legacy fields: %v", out[0])
	}
	if _, present := out[1]["history_locs"]; present {
		t.Fatal("empty entry must not grow history_locs")
	}
}

func TestHistoryPopularities(t *testing.T) {
	values, err := normalizeHistoryPopularities(map[string]any{"technocracy": float64(60), "democratic": float64(40)})
	if err != nil || values["technocracy"] != int64(60) || values["democratic"] != int64(40) {
		t.Fatalf("values=%v err=%v", values, err)
	}
	if cleared, err := normalizeHistoryPopularities(map[string]any{}); err != nil || len(cleared) != 0 {
		t.Fatalf("explicit clear: %v %v", cleared, err)
	}
	if _, err := normalizeHistoryPopularities(map[string]any{"technocracy": float64(90)}); err == nil || !strings.Contains(err.Error(), "add up to 100") {
		t.Fatalf("broken total: %v", err)
	}
	payload := map[string]any{"set_popularities": map[string]any{"technocracy": float64(60), "democratic": float64(40)}}
	if err := normalizeHistoryFilePayload(payload); err != nil {
		t.Fatal(err)
	}
	if payload["set_popularities_democratic"] != int64(40) || payload["set_popularities_fascism"] != int64(0) {
		t.Fatalf("legacy mirrors: %#v", payload)
	}
}

func TestValidateHistoryDateBlocks(t *testing.T) {
	if _, err := validateHistoryDateBlocks([]any{map[string]any{"date": "1939.13.1", "script": ""}}); err == nil ||
		err.Error() != "date block 0: date must use a positive year, month 1-12, and day 1-31" {
		t.Fatalf("bad month: %v", err)
	}
	blocks, err := validateHistoryDateBlocks([]any{map[string]any{"date": " 1939.1.1.12 ", "script": "x = y"}})
	if err != nil || blocks[0].(map[string]any)["date"] != "1939.1.1.12" {
		t.Fatalf("blocks=%v err=%v", blocks, err)
	}
}

func TestStateOutlinesOfTwoNeighbours(t *testing.T) {
	rows := []string{"111222", "112222", "111222", "111222"}
	w, h := 6, 4
	province := make([]int32, w*h)
	for y, row := range rows {
		for x, c := range row {
			province[y*w+x] = int32(c - '0')
		}
	}
	shapes := stateOutlines(w, h, province, 1, map[int][]int{10: {1}, 20: {2}, 30: {9}})
	if _, ok := shapes[30]; ok {
		t.Error("state 30 has no pixel")
	}
	left, right := shapes[10], shapes[20]
	if len(left.Rings) != 1 || len(right.Rings) != 1 {
		t.Fatalf("rings: %v / %v", left.Rings, right.Rings)
	}
	if left.BBox != [4]int{0, 0, 3, 4} || right.BBox != [4]int{2, 0, 6, 4} {
		t.Errorf("boxes %v %v", left.BBox, right.BBox)
	}
	shared := map[[2]int]bool{}
	for _, p := range left.Rings[0] {
		shared[p] = true
	}
	common := 0
	for _, p := range right.Rings[0] {
		if shared[p] {
			common++
		}
	}
	if common < 4 {
		t.Errorf("neighbours share %d corners: %v / %v", common, left.Rings, right.Rings)
	}
}
