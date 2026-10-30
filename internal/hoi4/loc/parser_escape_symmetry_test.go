package loc

import (
	"strings"
	"testing"
)

func TestDecodeLocEscapesTable(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"plain", "plain"},
		{`\n`, "\n"},
		{`\t`, "\t"},
		{`\"`, `"`},
		{`\\`, `\`},

		{`\\n`, `\n`},
		{`\\\n`, "\\\n"},

		{`\q`, `\q`},
		{`C:\mods`, `C:\mods`},

		{`ends with one\`, `ends with one\`},
		{`\`, `\`},

		{`\中文`, `\中文`},
		{`中文\n换行`, "中文\n换行"},
	}
	for _, tc := range cases {
		if got := DecodeLocEscapes(tc.in); got != tc.want {
			t.Errorf("DecodeLocEscapes(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestDecodeAppliedOnceOnly(t *testing.T) {
	const in = `\\n`
	once := DecodeLocEscapes(in)
	if once != `\n` {
		t.Fatalf("first decode: got %q want %q", once, `\n`)
	}
	if twice := DecodeLocEscapes(once); twice != "\n" {
		t.Fatalf("sanity: second decode should differ, got %q", twice)
	}
}

func TestUnevenTrailingBackslashFallsOutOfStrictParse(t *testing.T) {
	const lenientNote = "recovered via lenient parse"

	uneven := []string{`C:\mods\`, `ends with one\`, `a\\"b`}
	for _, v := range uneven {
		text := "l_english:\n bad_key: \"" + v + "\"\n next_key: \"still here\""
		_, _, diags := ParseLocText(text)
		if !containsNote(diags, lenientNote) {
			t.Errorf("uneven trailing run %q should fall out of strict parse, diags=%v",
				v, diags)
		}
	}

	even := map[string]string{
		`C:\\mods\\`:      `C:\mods\`,
		`ends with one\\`: `ends with one\`,
		`a\"b`:            `a"b`,
	}
	for emitted, want := range even {
		text := "l_english:\n bad_key: \"" + emitted + "\"\n next_key: \"still here\""
		_, entries, diags := ParseLocText(text)
		if len(diags) != 0 {
			t.Errorf("even trailing run %q must parse strictly, diags=%v", emitted, diags)
		}
		if entries["bad_key"] != want {
			t.Errorf("even trailing run %q decoded to %q want %q",
				emitted, entries["bad_key"], want)
		}
		if entries["next_key"] != "still here" {
			t.Errorf("even trailing run %q lost the next line: %v", emitted, entries)
		}
	}
}

func containsNote(diags []string, note string) bool {
	for _, d := range diags {
		if strings.Contains(d, note) {
			return true
		}
	}
	return false
}

func TestDecodeTailSetMatchesEmitter(t *testing.T) {
	for _, tail := range []string{"n", "t", `"`, `\`} {
		in := `\` + tail
		if got := DecodeLocEscapes(in); got == in {
			t.Errorf("tail %q is in the emitter's set but decoder left %q untouched", tail, in)
		}
	}
	for _, tail := range []string{"r", "0", "x", "a"} {
		in := `\` + tail
		if got := DecodeLocEscapes(in); got != in {
			t.Errorf("tail %q is NOT in the emitter's set but decoder turned %q into %q "+
				"— the two sides have drifted", tail, in, got)
		}
	}
}
