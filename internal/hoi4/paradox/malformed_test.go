package paradox

import (
	"strings"
	"testing"
)

func parseNoPanic(t *testing.T, name, src string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s: Parse panicked: %v", name, r)
		}
	}()
	ast, rec, err := Parse(src)
	if err != nil {
		if _, ok := err.(*ParseError); !ok {
			t.Fatalf("%s: error type %T, want *ParseError", name, err)
		}
		return
	}
	if ast == nil {
		t.Fatalf("%s: nil AST with nil error", name)
	}
	_ = rec

	_ = EntriesJSON(ast.Entries)
	_ = RecoveriesJSON(rec)
}

func TestMalformedInputNeverPanics(t *testing.T) {
	cases := map[string]string{
		"empty":                  "",
		"only newlines":          "\n\n\n",
		"only bom":               bomLit,
		"bom run":                bomLit + bomLit + bomLit,
		"lone open brace":        "{",
		"lone close brace":       "}",
		"close then open":        "}{",
		"unbalanced open":        "a = { b = { c = 1 }",
		"unbalanced close":       "a = { b = 1 } }",
		"many stray closes":      strings.Repeat("}", 500),
		"operator only":          "=",
		"operator run":           "= = = >= <= != ==",
		"key then eof":           "a =",
		"key op op":              "a = = 1",
		"value with no key":      "1 2 3 4",
		"block as key":           "{ } = 1",
		"unterminated quote":     `a = "abc`,
		"unterminated quote nl":  "a = \"abc\ndef",
		"quote at eof":           `"`,
		"backslash at eof":       `a = "abc\`,
		"escaped quote only":     `a = "\""`,
		"comment no newline":     "# comment with no newline",
		"hash inside word":       "a#b = 1",
		"bang only":              "!",
		"bang run":               "!!!!!",
		"nul byte":               "a = \x00 b = 1",
		"control chars":          "\x01\x02\x03 a = 1",
		"invalid utf8":           "a = \xff\xfe\x80 b = 1",
		"cr only line endings":   "a = 1\rb = 2\r",
		"crlf":                   "a = 1\r\nb = 2\r\n",
		"tabs everywhere":        "\ta\t=\t{\t1\t}\t",
		"equals in quotes":       `a = "b = { c"`,
		"brace in quotes":        `a = "}"`,
		"comment then eof brace": "a = { # note\n",
		"double negative":        "a = --1",
		"lone minus":             "a = -",
		"huge number":            "a = " + strings.Repeat("9", 4000),
		"long key":               strings.Repeat("k", 100000) + " = 1",
		"long quoted":            `a = "` + strings.Repeat("x", 100000) + `"`,
		"repeated empty blocks":  strings.Repeat("{}", 2000),
		"repeated dangling":      strings.Repeat("a =\n", 1000),
		"unicode identifiers":    "德意志 = 帝国\n",
		"emoji":                  "a = 🇩🇪\n",
		"rgb without block":      "color = rgb\n",
		"rgb then eof":           "color = rgb {",
		"qualifier no close":     "color = rgb { 1 2 3",
	}
	for name, src := range cases {
		parseNoPanic(t, name, src)
	}
}

func TestDeeplyNestedBlocksDoNotBlowTheStack(t *testing.T) {

	const depth = 5000
	src := strings.Repeat("a = {", depth) + strings.Repeat("}", depth)
	parseNoPanic(t, "deep nesting", src)

	parseNoPanic(t, "deep nesting unterminated", strings.Repeat("a = {", depth))
}

func TestPathologicalNestingIsAParseErrorNotAStackOverflow(t *testing.T) {
	for name, src := range map[string]string{
		"anonymous": strings.Repeat("{", 400000),
		"keyed":     strings.Repeat("a = {", 400000),
	} {
		_, _, err := Parse(src)
		perr, ok := err.(*ParseError)
		if !ok || !strings.Contains(perr.Msg, "nested deeper than") {
			t.Fatalf("%s: err=%v, want the nesting ParseError", name, err)
		}
	}

	ast, _, err := Parse(strings.Repeat("a = {", maxBlockDepth))
	if err != nil || ast == nil {
		t.Fatalf("depth %d must still parse: %v", maxBlockDepth, err)
	}
}

func TestTruncatedRealFilePrefixesNeverPanic(t *testing.T) {

	const src = `focus_tree = {
	id = german_focus
	country = {
		factor = 0
		modifier = {
			add = 10
			tag = GER
		}
	}
	shortcut = {
		name = GER_oppose_hitler_shortcut
		scroll_wheel_factor = 0.485
		trigger = { has_dlc = "Gotterdammerung" }
	}
}
`
	runes := []rune(src)
	for i := 0; i <= len(runes); i++ {
		parseNoPanic(t, "truncated at "+itoa(i), string(runes[:i]))
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func FuzzParseNoPanic(f *testing.F) {
	seeds := []string{
		"a = 1",
		"color = rgb { 153 0 51 }",
		"color = { 106  119  89 } # { 76  97  121 }",
		"date = 1936.1.1.12",
		"a = { b = { c = 1 } }",
		"}\na = 1\n",
		"a =\n}",
		`a = "unterminated`,
		"# comment only",
		bomLit + "a = 1",
		"surrender_progress > 0.75",
		"filters = { new_content continent }",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		ast, rec, err := Parse(src)
		if err != nil {
			if _, ok := err.(*ParseError); !ok {
				t.Fatalf("error type %T, want *ParseError", err)
			}
			return
		}
		if ast == nil {
			t.Fatal("nil AST with nil error")
		}
		_ = EntriesJSON(ast.Entries)
		_ = RecoveriesJSON(rec)
	})
}
