package paradox

import (
	"strings"
	"testing"
)

const bomLit = string(rune(0xFEFF))

func sig(toks []*Token) []string {
	out := make([]string, 0, len(toks))
	for _, t := range toks {
		out = append(out, t.Type+":"+t.Value)
	}
	return out
}

func tok(t *testing.T, src string) []*Token {
	t.Helper()
	return Tokenize([]rune(src))
}

func wantSig(t *testing.T, src string, want ...string) {
	t.Helper()
	got := sig(tok(t, src))

	want = append(want, "EOF:")
	if len(got) != len(want) {
		t.Fatalf("token count mismatch for %q\n got: %v\nwant: %v", src, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token %d mismatch for %q\n got: %v\nwant: %v", i, src, got, want)
		}
	}
}

func TestTokenizeOperators(t *testing.T) {

	wantSig(t, "a = 1", "STRING:a", "OP:=", "NUMBER:1")
	wantSig(t, "a == 1", "STRING:a", "OP:==", "NUMBER:1")
	wantSig(t, "a > 1", "STRING:a", "OP:>", "NUMBER:1")
	wantSig(t, "a >= 1", "STRING:a", "OP:>=", "NUMBER:1")
	wantSig(t, "a < 1", "STRING:a", "OP:<", "NUMBER:1")
	wantSig(t, "a <= 1", "STRING:a", "OP:<=", "NUMBER:1")
	wantSig(t, "a != 1", "STRING:a", "OP:!=", "NUMBER:1")
}

func TestTokenizeOperatorsWithoutSurroundingSpace(t *testing.T) {

	wantSig(t, "a=1", "STRING:a", "OP:=", "NUMBER:1")
	wantSig(t, "a>=1", "STRING:a", "OP:>=", "NUMBER:1")
	wantSig(t, "x={y=1}", "STRING:x", "OP:=", "LBRACE:{", "STRING:y", "OP:=", "NUMBER:1", "RBRACE:}")
}

func TestTokenizeBareBangIsUnknown(t *testing.T) {

	wantSig(t, "!", "UNKNOWN:!")
	wantSig(t, "a = !", "STRING:a", "OP:=", "UNKNOWN:!")
}

func TestTokenizeNumberClassification(t *testing.T) {
	cases := []struct {
		src      string
		wantType string
	}{
		{"1", "NUMBER"},
		{"0", "NUMBER"},
		{"-1", "NUMBER"},
		{"1936", "NUMBER"},
		{"1.5", "NUMBER"},
		{"-1.5", "NUMBER"},
		{"0.75", "NUMBER"},

		{"1936.1.1", "STRING"},
		{"1.", "STRING"},
		{".5", "STRING"},
		{"+1", "STRING"},
		{"1e5", "STRING"},
		{"--1", "STRING"},
		{"1_000", "STRING"},
		{"yes", "STRING"},
		{"GFX_thing", "STRING"},
	}
	for _, c := range cases {
		toks := tok(t, c.src)
		if len(toks) != 2 {
			t.Fatalf("%q: expected 1 token + EOF, got %v", c.src, sig(toks))
		}
		if toks[0].Type != c.wantType {
			t.Errorf("%q: got %s, want %s", c.src, toks[0].Type, c.wantType)
		}
		if toks[0].Value != c.src {
			t.Errorf("%q: value round-trip broken, got %q", c.src, toks[0].Value)
		}
	}
}

func TestTokenizeDateLiteralStaysString(t *testing.T) {

	ast, rec, err := Parse("date = 1936.1.1.12\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	e := ast.Entries[0]
	if e.Scalar.Kind != ScalarStr || e.Scalar.Str != "1936.1.1.12" {
		t.Fatalf("date literal must stay a string scalar, got kind=%d %q",
			e.Scalar.Kind, e.Scalar.StrValue())
	}
	if e.Quoted {
		t.Fatalf("bare date must not be flagged as quoted")
	}
}

func TestTokenizeQuotedString(t *testing.T) {
	toks := tok(t, `name = "GATHERING_STORM_NAME"`)
	v := toks[2]
	if v.Type != "STRING" || v.Value != "GATHERING_STORM_NAME" || !v.Quoted {
		t.Fatalf("quoted value wrong: %+v", v)
	}

	bare := tok(t, `name = GATHERING_STORM_NAME`)
	if bare[2].Quoted {
		t.Fatalf("bare value must not carry Quoted=true")
	}
}

func TestTokenizeQuotedStringKeepsSpacesAndSpecials(t *testing.T) {

	toks := tok(t, `desc = "a = b { c } # d"`)
	if len(toks) != 4 {
		t.Fatalf("quoted blob must stay one token, got %v", sig(toks))
	}
	if toks[2].Value != "a = b { c } # d" {
		t.Fatalf("quoted value mangled: %q", toks[2].Value)
	}
}

func TestTokenizeEscapeSequenceIsNotUnescaped(t *testing.T) {

	toks := tok(t, `x = "a\"b" y = 1`)
	if toks[2].Value != `a\"b` {
		t.Fatalf("escape handling changed: %q (want %q)", toks[2].Value, `a\"b`)
	}
	if toks[3].Value != "y" {
		t.Fatalf("string did not terminate at the right quote: %v", sig(toks))
	}
}

func TestTokenizeUnterminatedQuoteDoesNotHang(t *testing.T) {
	toks := tok(t, `x = "never closed`)
	if toks[2].Type != "STRING" || toks[2].Value != "never closed" || !toks[2].Quoted {
		t.Fatalf("unterminated quote must yield the tail as a quoted string: %+v", toks[2])
	}
	if toks[len(toks)-1].Type != "EOF" {
		t.Fatalf("stream must still terminate with EOF: %v", sig(toks))
	}
}

func TestTokenizeLineNumbers(t *testing.T) {
	src := "a = 1\nb = 2\n\nc = 3"
	toks := tok(t, src)
	want := map[string]int{"a": 1, "b": 2, "c": 4}
	for _, tk := range toks {
		if l, ok := want[tk.Value]; ok && tk.Line != l {
			t.Errorf("token %q line = %d, want %d", tk.Value, tk.Line, l)
		}
	}
	if eof := toks[len(toks)-1]; eof.Line != 4 {
		t.Errorf("EOF line = %d, want 4", eof.Line)
	}
}

func TestTokenizeCRLFCountsLinesOnce(t *testing.T) {

	crlf := tok(t, "a = 1\r\nb = 2\r\nc = 3\r\n")
	lf := tok(t, "a = 1\nb = 2\nc = 3\n")
	if len(crlf) != len(lf) {
		t.Fatalf("CRLF token count %d != LF %d", len(crlf), len(lf))
	}
	for i := range lf {
		if crlf[i].Type != lf[i].Type || crlf[i].Value != lf[i].Value || crlf[i].Line != lf[i].Line {
			t.Fatalf("CRLF/LF divergence at %d: %+v vs %+v", i, crlf[i], lf[i])
		}
	}
}

func TestTokenizeMultilineQuotedStringCountsLines(t *testing.T) {
	toks := tok(t, "x = \"line1\nline2\"\ny = 1")
	var y *Token
	for _, tk := range toks {
		if tk.Value == "y" {
			y = tk
		}
	}
	if y == nil {
		t.Fatal("y token missing")
	}
	if y.Line != 3 {
		t.Fatalf("newline inside a quoted string must advance the line counter: y at %d, want 3", y.Line)
	}
}

func TestTokenizeEscapedNewlineSkipsLineCount_QuirkBugCompatible(t *testing.T) {

	toks := tok(t, "x = \"a\\\nb\"\ny = 1")
	var y *Token
	for _, tk := range toks {
		if tk.Value == "y" {
			y = tk
		}
	}
	if y == nil {
		t.Fatal("y token missing")
	}
	if y.Line != 2 {
		t.Fatalf("quirk changed: y at line %d, want 2 (backslash-escaped newline is not counted)", y.Line)
	}
}

func TestTokenizeBigIntegerKeepsPrecision(t *testing.T) {

	const huge = "123456789012345678901234567890"
	ast, _, err := Parse("x = " + huge + "\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	e := ast.Entries[0]
	if e.Scalar.Kind != ScalarInt || e.Scalar.StrValue() != huge {
		t.Fatalf("big int mangled: kind=%d value=%q", e.Scalar.Kind, e.Scalar.StrValue())
	}
}

func TestParseStripsLeadingBOM(t *testing.T) {
	for _, prefix := range []string{bomLit, bomLit + bomLit, ""} {
		ast, _, err := Parse(prefix + "graphical_culture = western_european_gfx\n")
		if err != nil {
			t.Fatalf("prefix %q: %v", prefix, err)
		}
		if len(ast.Entries) != 1 {
			t.Fatalf("prefix %q: got %d entries", prefix, len(ast.Entries))
		}
		if k := ast.Entries[0].Key; k == nil || *k != "graphical_culture" {
			t.Fatalf("prefix %q: BOM leaked into the key: %v", prefix, k)
		}
	}
}

func TestBOMOnlyAtFileStartIsStripped_DocumentsBehaviour(t *testing.T) {

	ast, _, err := Parse("a = 1\n" + bomLit + "b = 2\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ast.Entries) != 2 {
		t.Fatalf("got %d entries", len(ast.Entries))
	}
	if k := *ast.Entries[1].Key; !strings.HasPrefix(k, bomLit) {
		t.Fatalf("expected interior BOM to stay glued to the key, got %q", k)
	}
}

func TestTokenizeUTF8IsRuneIndexed(t *testing.T) {

	src := "name = \"德意志\" # 注释\n"
	ast, _, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	e := ast.Entries[0]
	if e.Scalar.Str != "德意志" {
		t.Fatalf("multi-byte value mangled: %q", e.Scalar.Str)
	}
	if len(e.Trivia.Trailing) != 1 {
		t.Fatalf("expected 1 trailing comment, got %d", len(e.Trivia.Trailing))
	}
	c := e.Trivia.Trailing[0]

	if c.Column != 13 {
		t.Fatalf("comment Column = %d, want 13 (rune space; byte space would be 19)", c.Column)
	}
	if c.Text != "# 注释" {
		t.Fatalf("comment text mangled: %q", c.Text)
	}
}
