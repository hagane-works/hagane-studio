package cosmetic

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizedNameVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/name_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct {
		ID         string       `json:"id"`
		Definition Definition   `json:"definition"`
		Locale     string       `json:"locale"`
		Suffix     string       `json:"suffix"`
		Expected   ResolvedName `json:"expected"`
	}
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, vector := range vectors {
		t.Run(vector.ID, func(t *testing.T) {
			if got := ResolveName(vector.Definition, vector.Locale, vector.Suffix); got != vector.Expected {
				t.Fatalf("resolved = %#v, want %#v", got, vector.Expected)
			}
		})
	}
}

func TestEveryMissingCombinationAndLanguage(t *testing.T) {
	chains := map[string][]string{"": {""}, "ADJ": {"ADJ", ""}, "DEF": {"DEF", ""}}
	for _, ideology := range []string{"democratic", "fascism", "neutrality", "communism"} {
		chains[ideology] = []string{ideology, ""}
		chains[ideology+"_ADJ"] = []string{ideology + "_ADJ", "ADJ", ""}
		chains[ideology+"_DEF"] = []string{ideology + "_DEF", ideology, "DEF", ""}
	}
	for _, locale := range Locales() {
		for suffix, chain := range chains {
			for mask := 0; mask < 1<<len(chain); mask++ {
				for _, missing := range []string{"", " \t "} {
					names := map[string]string{}
					want := ResolvedName{Value: "exact imported", Source: "imported", InheritedFrom: suffix}
					for i := len(chain) - 1; i >= 0; i-- {
						candidate := chain[i]
						names[candidate] = missing
						if mask&(1<<i) != 0 {
							names[candidate] = " 名称 \"quoted\" \\ path " + candidate
							want = ResolvedName{Value: names[candidate], Source: "authored", InheritedFrom: candidate}
						}
					}
					def := Definition{Tag: "ROM_test", NameLocs: Names{locale: names}, ImportedNameLocs: Names{locale: {suffix: "exact imported"}}}
					if got := ResolveName(def, locale, suffix); got != want {
						t.Fatalf("%s/%s mask=%d: got %#v want %#v", locale, suffix, mask, got, want)
					}
				}
			}
		}
	}
}

func TestImportedPresenceAndNoInheritance(t *testing.T) {
	def := Definition{Tag: "ROM_greater_kingdom", ImportedNameLocs: Names{"l_english": {"": "", "ADJ": "ROM Greater Kingdom", "DEF": " "}}}
	before, _ := json.Marshal(def)
	names := ResolveNames(def, "l_english")
	if len(names) != 15 {
		t.Fatalf("forms=%d", len(names))
	}
	for suffix, expected := range map[string]string{"": "", "ADJ": "ROM Greater Kingdom", "DEF": " "} {
		if got := names[suffix]; got.Source != "imported" || got.Value != expected {
			t.Fatalf("explicit %q changed: %#v", suffix, got)
		}
	}
	if got := names["fascism_DEF"]; got.Source != "generated" {
		t.Fatalf("imported forms inherited: %#v", got)
	}
	if got := ResolveName(def, "l_simp_chinese", ""); got.Source != "generated" {
		t.Fatalf("cross-language fallback: %#v", got)
	}
	after, _ := json.Marshal(def)
	if string(before) != string(after) {
		t.Fatal("resolver mutated inputs")
	}
}

func TestValidation(t *testing.T) {
	for _, tag := range []string{"ROM_greater_kingdom", "MiXeD.1-Test", strings.Repeat("x", 100)} {
		if err := ValidateTag(tag); err != nil {
			t.Fatal(tag, err)
		}
	}
	for _, tag := range []string{"", ".", "..", "ROM foo", "ROM\nfoo", "RÖM", "国", strings.Repeat("x", 101)} {
		if err := ValidateTag(tag); err == nil {
			t.Fatalf("accepted invalid tag %q", tag)
		}
	}
	for _, locale := range Locales() {
		for _, suffix := range Suffixes() {
			for _, value := range []string{"", "   ", " 翁法罗斯 ", `Name "quoted" \ $TOKEN$ §Ytext§!`, strings.Repeat("国", 4096)} {
				if err := ValidateName(locale, suffix, value); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	for _, value := range []string{"a\nb", "a\rb", "a\u2028b", "a\u2029b", "a\x00b", string([]byte{0xff}), strings.Repeat("国", 4097)} {
		if err := ValidateName("l_english", "", value); err == nil {
			t.Fatal("accepted invalid value")
		}
	}
	if ValidateName("en", "", "value") == nil || ValidateName("l_english", "despotism", "value") == nil {
		t.Fatal("accepted unsupported locale/suffix")
	}
	got := Suffixes()
	got[0] = "mutated"
	if Suffixes()[0] != "" {
		t.Fatal("mutable whitelist escaped")
	}
}

func TestRGBJSON(t *testing.T) {
	for _, value := range []string{"[1,2]", "[1,2,3,4]", "[1,2,256]", "[-1,2,3]", "[1.5,2,3]", "[null,2,3]", "null", "{}"} {
		var color RGB
		if json.Unmarshal([]byte(value), &color) == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	var def Definition
	if err := json.Unmarshal([]byte(`{"color":null,"color_ui":[0,255,1]}`), &def); err != nil {
		t.Fatal(err)
	}
	if def.Color != nil || *def.ColorUI != (RGB{0, 255, 1}) {
		t.Fatalf("bad nullable RGB: %#v", def)
	}
}

func TestIndependentColorFallbacks(t *testing.T) {
	colors := []RGB{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}}
	for mapMask := 0; mapMask < 8; mapMask++ {
		for uiMask := 0; uiMask < 8; uiMask++ {
			mapSlots, uiSlots := [3]*RGB{}, [3]*RGB{}
			base, _ := ResolveColors(Definition{Tag: "ROM_greater_kingdom"}, nil, nil)
			wantMap, wantUI := base, base
			for i := 2; i >= 0; i-- {
				if mapMask&(1<<i) != 0 {
					mapSlots[i] = &colors[i]
					wantMap = colors[i]
				}
			}
			wantUI = wantMap
			for i := 2; i >= 0; i-- {
				if uiMask&(1<<i) != 0 {
					uiSlots[i] = &colors[i]
					wantUI = colors[i]
				}
			}
			def := Definition{Tag: "ROM_greater_kingdom", Color: mapSlots[0], ImportedColor: mapSlots[1], ColorUI: uiSlots[0], ImportedColorUI: uiSlots[1]}
			gotMap, gotUI := ResolveColors(def, mapSlots[2], uiSlots[2])
			if gotMap != wantMap || gotUI != wantUI {
				t.Fatalf("map=%d UI=%d got %v/%v want %v/%v", mapMask, uiMask, gotMap, gotUI, wantMap, wantUI)
			}
		}
	}
	invalid := RGB{-1, 0, 0}
	valid := RGB{255, 0, 1}
	got, _ := ResolveColors(Definition{Color: &invalid, ImportedColor: &valid}, nil, nil)
	if got != valid {
		t.Fatal("invalid higher color prevented valid fallback")
	}
}

func TestHumanize(t *testing.T) {
	for input, want := range map[string]string{"ROM_greater_kingdom": "ROM Greater Kingdom", "USA_TEST_LONGER": "USA TEST Longer", "": "", "___": "___", "  MiXeD__éCOLE_国名  ": "Mixed École 国名", "ROM.foo-test": "Rom.foo-test", "ROM\u2003test": "ROM Test"} {
		if got := Humanize(input); got != want {
			t.Errorf("Humanize(%q)=%q want %q", input, got, want)
		}
	}
}

func TestResolvedJSONContract(t *testing.T) {
	value := ResolvedName{Value: "value", Source: "authored", InheritedFrom: "ADJ"}
	got, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]string
	_ = json.Unmarshal(got, &fields)
	want := map[string]string{"value": "value", "source": "authored", "inherited_from": "ADJ"}
	if !reflect.DeepEqual(fields, want) {
		t.Fatal(fmt.Sprint(fields))
	}
}
