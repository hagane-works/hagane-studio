package locout

import (
	"strings"
	"testing"

	"haganeworks/internal/hoi4/loc"
)

var escapeCases = []string{
	"",
	"plain text",
	`C:\mods\`,
	`ends with one\`,
	`\`,
	`\\`,
	`\\\`,
	`\n`,
	`\\n`,
	`\t`,
	`a\"b`,
	`has "quotes" inside`,
	`trailing quote"`,
	`back\slash mid`,
	`C:\new folder`,
	"tab\there and\nnewline",
	"crlf\r\nhere",
	"bare cr\rhere",
	`中文 名字 \ 反斜杠`,
	`中文\n换行`,
	`\中文`,
	"mixed \\n real\n \\\" quote \" tab\t end\\",
	`§Y coloured §! text\nsecond line`,
	`[GER.GetName]\n[ROOT.GetNameDef]`,
}

func normaliseNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

func decodeViaRealParser(t *testing.T, value string) (probe string, sentinelOK bool) {
	t.Helper()
	text := "l_english:\n" +
		YmlLine("probe_key", Text{S: value}) + "\n" +
		YmlLine("sentinel_key", Text{S: "still here"})
	lang, entries, _ := loc.ParseLocText(text)
	if lang != "l_english" {
		t.Fatalf("header lost for %q · emitted=%q", value, text)
	}
	got, ok := entries["probe_key"]
	if !ok {
		t.Fatalf("probe key lost for %q · emitted=%q", value, text)
	}
	return got, entries["sentinel_key"] == "still here"
}

func trailingBackslashRun(s string) int {
	return len(s) - len(strings.TrimRight(s, `\`))
}

func TestYmlEscapeIsIdempotent(t *testing.T) {
	for _, in := range escapeCases {
		once := YmlEscapeValue(in)
		if twice := YmlEscapeValue(once); twice != once {
			t.Errorf("not idempotent: %q -> %q -> %q", in, once, twice)
		}
	}
}

func TestYmlEscapeDecodeSymmetry(t *testing.T) {
	for _, in := range escapeCases {
		got := loc.DecodeLocEscapes(YmlEscapeValue(in))
		want := normaliseNewlines(loc.DecodeLocEscapes(in))
		if got != want {
			t.Errorf("semantic symmetry broken for %q: decode(encode(x))=%q decode(x)=%q",
				in, got, want)
		}
	}
}

func TestAuthoredBackslashNSurvivesAsNewline(t *testing.T) {
	const in = `First line.\nSecond line.`
	emitted := YmlEscapeValue(in)
	if emitted != in {
		t.Fatalf("authored \\n was rewritten: %q -> %q", in, emitted)
	}
	if strings.Contains(emitted, `\\n`) {
		t.Fatalf("backslash doubled again: %q", emitted)
	}
	probe, sentinel := decodeViaRealParser(t, in)
	if probe != "First line.\nSecond line." {
		t.Fatalf("did not reach the game as a line break: %q", probe)
	}
	if !sentinel {
		t.Fatalf("sentinel line swallowed for %q", in)
	}
}

func TestBackslashFreeTextRoundTripsExactly(t *testing.T) {
	checked := 0
	for _, in := range escapeCases {
		if strings.Contains(in, `\`) || strings.Contains(in, "\r") {
			continue
		}
		if got := loc.DecodeLocEscapes(YmlEscapeValue(in)); got != in {
			t.Errorf("lossy for backslash-free %q: got %q", in, got)
		}
		checked++
	}
	if checked < 5 {
		t.Fatalf("only %d backslash-free cases — corpus too thin", checked)
	}
}

func TestYmlEscapeTrailingBackslashRunIsEven(t *testing.T) {
	for _, in := range escapeCases {
		encoded := YmlEscapeValue(in)
		if run := trailingBackslashRun(encoded); run%2 != 0 {
			t.Errorf("H-35 REGRESSION: %q encoded to %q · trailing backslash run "+
				"is %d (odd) — the closing quote would be eaten and HOI4 drops "+
				"the whole .yml file", in, encoded, run)
		}
	}
}

func TestEmittedLineReparsesWithRealParser(t *testing.T) {
	for _, in := range escapeCases {
		probe, sentinel := decodeViaRealParser(t, in)
		if !sentinel {
			t.Errorf("H-35 REGRESSION: the line after %q was swallowed", in)
			continue
		}
		want := normaliseNewlines(loc.DecodeLocEscapes(in))
		if probe != want {
			t.Errorf("value drifted through the real parser for %q: got %q want %q",
				in, probe, want)
		}
	}
}

func TestEmittedLinesParseStrictlyWithNoDiagnostics(t *testing.T) {
	for _, in := range escapeCases {
		text := "l_english:\n" +
			YmlLine("probe_key", Text{S: in}) + "\n" +
			YmlLine("sentinel_key", Text{S: "still here"})
		_, _, diags := loc.ParseLocText(text)
		if len(diags) != 0 {
			t.Errorf("H-35 REGRESSION: %q emitted a line that is not strictly "+
				"parseable · emitted=%q diags=%v", in, text, diags)
		}
	}
}
