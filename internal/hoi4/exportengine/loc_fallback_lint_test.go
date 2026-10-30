package exportengine

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const locFallbackTestdata = "testdata/locfallback"

func locFallbackFixtureEntries(t *testing.T) []Entry {
	t.Helper()
	root := filepath.Join(locFallbackTestdata, "input")
	var entries []Entry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entries = append(entries, Entry{Path: filepath.ToSlash(rel), Bytes: data, Origin: "generated"})
		return nil
	})
	if err != nil {
		t.Fatalf("walk fixture: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("fixture tree %s is empty", root)
	}

	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Path > entries[j].Path })
	return entries
}

const locFallbackExpectedLog = `# Untranslated text
# A key is listed when its text is not empty, is exactly the same as the English text of the same key, and has at least three Latin letters in a row. The English text was written on purpose: in game it is better than a bare key. Translate the text to clear a line. This never blocks the export.
#
# Untranslated text: 10 of 20 keys (50.0%) still show the English text in french, german, polish, simp_chinese, spanish. Translate them to clear the list in _LOC_EN_FALLBACK.log.
#
# language          untranslated / total    percent
# french            0 / 2                   0.0%
# german            6 / 12                  50.0%
# polish            1 / 1                   100.0%
# simp_chinese      2 / 3                   66.7%
# spanish           1 / 2                   50.0%

## l_french  0 / 2 (0.0%)

## l_german  6 / 12 (50.0%)
### localisation/german/demo_l_german.yml  6 / 12
demo_focus_a
demo_focus_b
demo_var
demo_quote
demo_ver
demo_lenient

## l_polish  1 / 1 (100.0%)
### localisation/polish/demo_l_polish.yml  1 / 1
demo_focus_a

## l_simp_chinese  2 / 3 (66.7%)
### localisation/simp_chinese/replace/demo_l_simp_chinese.yml  2 / 3
demo_focus_b
demo_shared

## l_spanish  1 / 2 (50.0%)
### localisation/polish/demo_l_polish.yml  1 / 2
demo_focus_a

`

func readGolden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(locFallbackTestdata, name))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")))
}

func firstDiff(a, b string) string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(al) || i < len(bl); i++ {
		var x, y string
		if i < len(al) {
			x = al[i]
		}
		if i < len(bl) {
			y = bl[i]
		}
		if x != y {
			return "line idx " + itoa(i) + ":\n got: " + x + "\nwant: " + y
		}
	}
	return "(no line differs; length differs)"
}

func itoa(n int) string {
	return locFallbackThousands(n)
}

func TestLocFallbackMatchesGoldenFiles(t *testing.T) {
	report := ScanLocFallback(locFallbackFixtureEntries(t), english)

	if got, want := report.LogText(), locFallbackExpectedLog; got != want {
		t.Fatalf("log text differs from the golden file:\n%s", firstDiff(got, want))
	}
	if got, want := report.Headline(), strings.TrimRight(readGolden(t, "expected_headline.txt"), "\n"); got != want {
		t.Fatalf("headline differs: got %q want %q", got, want)
	}
	gotJSON, err := report.SummaryJSON()
	if err != nil {
		t.Fatalf("summary json: %v", err)
	}
	if got, want := string(gotJSON), strings.TrimRight(readGolden(t, "expected_summary.json"), "\n"); got != want {
		t.Fatalf("summary json differs:\n got: %s\nwant: %s", got, want)
	}
}

func TestLocFallbackRuleOnFixture(t *testing.T) {
	report := ScanLocFallback(locFallbackFixtureEntries(t), english)
	de := report.Languages["german"]
	if de == nil {
		t.Fatalf("german missing: %v", report.Languages)
	}
	f := de.Files["localisation/german/demo_l_german.yml"]
	if f == nil {
		t.Fatalf("german file missing: %v", de.Files)
	}
	want := []string{"demo_focus_a", "demo_focus_b", "demo_var", "demo_quote", "demo_ver", "demo_lenient"}
	if strings.Join(f.Keys, ",") != strings.Join(want, ",") {
		t.Fatalf("german fallback keys = %v, want %v", f.Keys, want)
	}

	for _, k := range f.Keys {
		if k == "demo_shared" {
			t.Fatalf("demo_shared counted although English replace/ overrides it")
		}
	}

	zh := report.Languages["simp_chinese"]
	if zh == nil || zh.Fallback != 2 {
		t.Fatalf("simp_chinese fallback = %+v, want 2 (BOM + CRLF file, replace/ dir)", zh)
	}

	es := report.Languages["spanish"]
	if es == nil || es.Files["localisation/polish/demo_l_polish.yml"] == nil {
		t.Fatalf("spanish entries in demo_l_polish.yml not bucketed by header: %+v", es)
	}

	if fr := report.Languages["french"]; fr == nil || fr.Fallback != 0 || fr.Total != 2 {
		t.Fatalf("french = %+v, want 0/2", fr)
	}

	if _, ok := report.Languages["russian"]; ok {
		t.Fatalf("header-less yml must not create a language bucket")
	}
	if got := report.Headline(); got != "Untranslated text: 10 of 20 keys (50.0%) still show the English text in "+
		"french, german, polish, simp_chinese, spanish. Translate them to clear the list in _LOC_EN_FALLBACK.log." {
		t.Fatalf("headline = %q", got)
	}
}

func TestLocFallbackLogEntryShape(t *testing.T) {
	report := ScanLocFallback(locFallbackFixtureEntries(t), english)
	entry := LocFallbackLogEntry("mod_demo", report)
	if entry.Path != "mod_demo/_LOC_EN_FALLBACK.log" {
		t.Fatalf("path = %q", entry.Path)
	}
	if entry.Origin != "generated" || entry.SourceEntity != "loc_fallback_lint" {
		t.Fatalf("origin/source = %q/%q", entry.Origin, entry.SourceEntity)
	}
	if !bytes.HasPrefix(entry.Bytes, []byte("# Untranslated text\n")) {
		t.Fatalf("log body: %q", entry.Bytes[:60])
	}

	if e := LocFallbackLogEntry("mod_demo", ScanLocFallback(nil, english)); e.Path != "" {
		t.Fatalf("empty report must not produce a log entry, got %q", e.Path)
	}
	empty := ScanLocFallback([]Entry{{Path: "m/localisation/english/x_l_english.yml", Bytes: []byte("l_english:\n a: \"Hello\"\n")}}, english)
	if e := LocFallbackLogEntry("m", empty); e.Path != "" {
		t.Fatalf("english-only mod must not produce a log entry, got %q", e.Path)
	}
	if empty.Headline() != "" {
		t.Fatalf("english-only headline must be empty, got %q", empty.Headline())
	}

	for _, banned := range []string{"ERROR", "FAIL", "INVALID", "BLOCK"} {
		if strings.Contains(strings.ToUpper(report.Headline()), banned) {
			t.Fatalf("headline must not read as an error: %q", report.Headline())
		}
	}
}

func TestLocFallbackFormattingHelpers(t *testing.T) {
	cases := map[int]string{0: "0", 999: "999", 1000: "1,000", 10052: "10,052", 42762: "42,762", 1234567: "1,234,567"}
	for n, want := range cases {
		if got := locFallbackThousands(n); got != want {
			t.Errorf("thousands(%d) = %q want %q", n, got, want)
		}
	}
	if got := locFallbackPercent(10052, 42762); got != "23.5%" {
		t.Errorf("percent = %q", got)
	}
	if got := locFallbackPercent(0, 0); got != "0.0%" {
		t.Errorf("percent(0,0) = %q", got)
	}
	if got := locFallbackPercent(2, 3); got != "66.7%" {
		t.Errorf("percent(2,3) = %q", got)
	}
}
