package tagaudit

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestAuditPythonFixtureContract(t *testing.T) {
	files := []File{
		{Path: "Fixture/common/country_tags/zz_dynamic.txt", Body: []byte("dynamic_tags = yes\nD01 = \"countries/D01.txt\"\n")},
		{Path: "Fixture/common/country_tags/00_countries.txt", Body: []byte("GER = \"countries/Germany.txt\"\nFRA = \"countries/France.txt\"\n")},
		{Path: "Fixture/common/countries/Germany.txt", Body: []byte("color = { 138 155 116 }\n")},
		{Path: "Fixture/common/countries/colors.txt", Body: []byte("FRA = { color = rgb { 57 113 228 } }\n")},
		{Path: "Fixture/localisation/english/countries_l_english.yml", Body: []byte("l_english:\n GER:0 \"Germany\"\n FRA:0 \"France\"\n")},
		{Path: "Fixture/localisation/simp_chinese/countries_l_simp_chinese.yml", Body: []byte("l_simp_chinese:\n GER:0 \"德国\"\n")},
		{Path: "Fixture/common/characters/GER_leaders.txt", Body: []byte("characters = { }\n")},
		{Path: "Fixture/common/national_focus/GER_focus.txt", Body: []byte("focus_tree = { id = GER_main }\n")},
		{Path: "Fixture/events/GER_news.txt", Body: []byte("add_namespace = GER_news\n")},
		{Path: "Fixture/common/ideas/FRA_ideas.txt", Body: []byte("ideas = { }\n")},
		{Path: "Fixture/descriptor.mod", Body: []byte("name = \"Fixture Mod\"\n")},
	}
	got := Audit(files, map[string]bool{"FRA": true})
	if !reflect.DeepEqual(got.CountryTagFiles, []string{
		"common/country_tags/00_countries.txt", "common/country_tags/zz_dynamic.txt",
	}) {
		t.Fatalf("country tag file order = %#v", got.CountryTagFiles)
	}
	if got.Summary != (Summary{Declared: 3, New: 2, Existing: 1, Dynamic: 1, Static: 2}) {
		t.Fatalf("summary = %+v", got.Summary)
	}
	if len(got.Detected) != 3 || got.Detected[0].Tag != "GER" || got.Detected[1].Tag != "FRA" || got.Detected[2].Tag != "D01" {
		t.Fatalf("detected order = %#v", got.Detected)
	}
	ger, fra, dynamic := got.Detected[0], got.Detected[1], got.Detected[2]
	if ger.NamePreview.English == nil || *ger.NamePreview.English != "Germany" || ger.NamePreview.Chinese == nil || *ger.NamePreview.Chinese != "德国" {
		t.Fatalf("GER names = %+v", ger.NamePreview)
	}
	if !reflect.DeepEqual(ger.ColorPreview, []int{138, 155, 116}) || ger.EntityPreview.Characters != 1 || ger.EntityPreview.FocusFiles != 1 || ger.EntityPreview.Events != 1 {
		t.Fatalf("GER preview = %+v color=%v", ger.EntityPreview, ger.ColorPreview)
	}
	if !fra.AlreadyExists || fra.NamePreview.English == nil || *fra.NamePreview.English != "France" || fra.NamePreview.Chinese != nil || !reflect.DeepEqual(fra.ColorPreview, []int{57, 113, 228}) || fra.EntityPreview.Ideas != 1 {
		t.Fatalf("FRA preview = %+v", fra)
	}
	if !dynamic.IsDynamic || dynamic.ColorPreview != nil || dynamic.NamePreview.English != nil || dynamic.NamePreview.Chinese != nil {
		t.Fatalf("dynamic preview = %+v", dynamic)
	}
	if got.DescriptorName == nil || *got.DescriptorName != "Fixture Mod" {
		t.Fatalf("descriptor = %v", got.DescriptorName)
	}

	if len(got.WarningItems) != 1 || got.WarningItems[0].Code != "unrouted_dir" || got.WarningItems[0].Params["dir"] != "(root)" {
		t.Fatalf("warnings = %#v", got.WarningItems)
	}
}

func BenchmarkAuditSixHundredTags(b *testing.B) {
	var declarations strings.Builder
	files := make([]File, 0, 602)
	for index := 0; index < 600; index++ {
		tag := fmt.Sprintf("X%03d", index)
		fmt.Fprintf(&declarations, "%s = \"countries/%s.txt\"\n", tag, tag)
		files = append(files, File{
			Path: "common/countries/" + tag + ".txt",
			Body: []byte("color = { 10 20 30 }\n"),
		})
	}
	files = append(files, File{Path: "common/country_tags/00.txt", Body: []byte(declarations.String())})
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		result := Audit(files, map[string]bool{})
		if result.Summary.Declared != 600 {
			b.Fatalf("declared=%d", result.Summary.Declared)
		}
	}
}

func TestAuditRecoveryWarningAndCrossFileFirstWins(t *testing.T) {
	files := []File{
		{Path: "common/country_tags/00.txt", Body: []byte("\ufeffUSA = \"countries/First.txt\"\rUSA = \"countries/Second.txt\"\u2028dynamic_tags = yes\u0085D01 = \"countries/D01.txt\"")},
		{Path: "common/country_tags/10.txt", Body: []byte("USA = \"countries/Third.txt\"\n")},
		{Path: "common/ideas/USA.txt", Body: []byte("ideas = { key =\n}\n}\n")},
	}
	got := Audit(files, map[string]bool{})
	if len(got.Detected) != 2 || got.Detected[0].FilePath != "countries/First.txt" || !got.Detected[1].IsDynamic {
		t.Fatalf("first-wins/dynamic drift: %#v", got.Detected)
	}
	if len(got.WarningItems) != 2 || got.WarningItems[0].Code != "parser_recovery_empty_value" || got.WarningItems[1].Code != "parser_recovery_stray_rbrace" {
		t.Fatalf("recovery warnings = %#v", got.WarningItems)
	}
	if !strings.Contains(got.Warnings[0], `§§parser_recovery_empty_value§§{"path": "common/ideas/USA.txt", "line": 1, "key": "key", "terminator": "}"}§§`) {
		t.Fatalf("coded warning mismatch: %q", got.Warnings[0])
	}
}
