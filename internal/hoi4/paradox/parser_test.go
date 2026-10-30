package paradox

import (
	"testing"
)

func mustParse(t *testing.T, src string) (*Block, []Recovery) {
	t.Helper()
	ast, rec, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected parse error: %v\nsource:\n%s", err, src)
	}
	return ast, rec
}

func key(e *Entry) string {
	if e.Key == nil {
		return "<nil>"
	}
	return *e.Key
}

func TestParseScalarKinds(t *testing.T) {
	ast, rec := mustParse(t, `
		int_v   = 1936
		neg_v   = -3
		float_v = 0.75
		str_v   = western_european_gfx
		quo_v   = "GFX_select_date_1936"
		date_v  = 1936.1.1
	`)
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	want := []struct {
		k    string
		kind ScalarKind
		str  string
	}{
		{"int_v", ScalarInt, "1936"},
		{"neg_v", ScalarInt, "-3"},
		{"float_v", ScalarFloat, "0.75"},
		{"str_v", ScalarStr, "western_european_gfx"},
		{"quo_v", ScalarStr, "GFX_select_date_1936"},
		{"date_v", ScalarStr, "1936.1.1"},
	}
	if len(ast.Entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(ast.Entries), len(want))
	}
	for i, w := range want {
		e := ast.Entries[i]
		if key(e) != w.k {
			t.Errorf("entry %d key = %s, want %s", i, key(e), w.k)
		}
		if e.Scalar.Kind != w.kind {
			t.Errorf("%s: kind = %d, want %d", w.k, e.Scalar.Kind, w.kind)
		}
		if e.Scalar.StrValue() != w.str {
			t.Errorf("%s: value = %q, want %q", w.k, e.Scalar.StrValue(), w.str)
		}
		if e.Block != nil {
			t.Errorf("%s: scalar entry must not carry a Block", w.k)
		}
	}
	if !ast.Entries[4].Quoted {
		t.Error("quoted value must set Entry.Quoted")
	}
}

func TestParseComparisonOperatorsArePreserved(t *testing.T) {

	ast, rec := mustParse(t, `
		a = 1
		b == 1
		c > 0.75
		d >= 1
		e < 1
		f <= 1
		g != 1
	`)
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	wantOps := []string{"=", "==", ">", ">=", "<", "<=", "!="}
	if len(ast.Entries) != len(wantOps) {
		t.Fatalf("got %d entries, want %d", len(ast.Entries), len(wantOps))
	}
	for i, op := range wantOps {
		e := ast.Entries[i]
		if e.Op == nil || *e.Op != op {
			t.Errorf("entry %d op = %v, want %q", i, e.Op, op)
		}
	}
	if ast.Entries[2].Scalar.Kind != ScalarFloat {
		t.Error("`c > 0.75` must still parse its value as a float")
	}
}

func TestParseNestedBlocks(t *testing.T) {

	ast, rec := mustParse(t, `
ideologies = {
	democratic = {
		types = {
			conservatism = {
			}
			liberalism = {
			}
		}
		can_host_government_in_exile = yes
	}
}
`)
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	if len(ast.Entries) != 1 || ast.Entries[0].Block == nil {
		t.Fatalf("top level must be one block entry, got %d", len(ast.Entries))
	}
	dem := ast.Entries[0].Block.Entries[0]
	if key(dem) != "democratic" || dem.Block == nil {
		t.Fatalf("democratic block missing: %+v", dem)
	}
	types := dem.Block.Entries[0]
	if key(types) != "types" || types.Block == nil {
		t.Fatalf("types block missing")
	}
	if len(types.Block.Entries) != 2 {
		t.Fatalf("types must hold 2 sub-blocks, got %d", len(types.Block.Entries))
	}
	for _, sub := range types.Block.Entries {
		if sub.Block == nil || len(sub.Block.Entries) != 0 {
			t.Fatalf("%s must be an empty block, got %+v", key(sub), sub)
		}
	}
	if last := dem.Block.Entries[1]; key(last) != "can_host_government_in_exile" ||
		last.Scalar.Str != "yes" {
		t.Fatalf("sibling scalar after a nested block was lost: %+v", last)
	}
}

func TestParseListVersusKeyValue(t *testing.T) {

	ast, _ := mustParse(t, `
		filters = { new_content continent }
		colors  = { 0 0 255 }
		kv      = { a = 1 }
		strings = { "FACTION_NAME_DEMOCRATIC_1" "FACTION_NAME_DEMOCRATIC_2" }
	`)
	filters := ast.Entries[0].Block.Entries
	if len(filters) != 2 {
		t.Fatalf("filters: got %d items, want 2", len(filters))
	}
	for _, e := range filters {
		if e.Key != nil || e.Op != nil {
			t.Fatalf("list item must have nil Key and nil Op: %+v", e)
		}
	}
	if filters[0].Scalar.Str != "new_content" || filters[1].Scalar.Str != "continent" {
		t.Fatalf("list items mangled: %+v", filters)
	}

	colors := ast.Entries[1].Block.Entries
	if len(colors) != 3 {
		t.Fatalf("colors: got %d items, want 3", len(colors))
	}
	for _, e := range colors {
		if e.Key != nil || e.Scalar.Kind != ScalarInt {
			t.Fatalf("color channel must be a keyless int: %+v", e)
		}
	}

	kv := ast.Entries[2].Block.Entries
	if len(kv) != 1 || kv[0].Key == nil {
		t.Fatalf("k/v inside a block must keep its key: %+v", kv)
	}

	strs := ast.Entries[3].Block.Entries
	if len(strs) != 2 || !strs[0].Quoted || strs[0].Scalar.Str != "FACTION_NAME_DEMOCRATIC_1" {
		t.Fatalf("quoted list items mangled: %+v", strs)
	}
}

func TestParseQualifiedColorBlock(t *testing.T) {

	for _, q := range []string{"rgb", "hsv", "RGB", "HSV"} {
		ast, rec := mustParse(t, "color = "+q+" { 153 0 51 }\n")
		if len(rec) != 0 {
			t.Fatalf("%s: unexpected recoveries %+v", q, rec)
		}
		e := ast.Entries[0]
		if e.Qualifier == nil || *e.Qualifier != q {
			t.Fatalf("%s: qualifier lost: %+v", q, e.Qualifier)
		}
		if e.Block == nil || len(e.Block.Entries) != 3 {
			t.Fatalf("%s: qualified block body wrong: %+v", q, e.Block)
		}
		if key(e) != "color" || e.Op == nil || *e.Op != "=" {
			t.Fatalf("%s: key/op wrong: %s %v", q, key(e), e.Op)
		}
	}
}

func TestParseUnknownQualifierIsNotSwallowed(t *testing.T) {

	ast, _ := mustParse(t, "color = cmyk { 1 2 3 }\n")
	if len(ast.Entries) != 2 {
		t.Fatalf("want 2 entries (scalar + anonymous block), got %d", len(ast.Entries))
	}
	if ast.Entries[0].Qualifier != nil {
		t.Fatalf("unknown qualifier must not be recorded: %v", ast.Entries[0].Qualifier)
	}
	if ast.Entries[0].Scalar.Str != "cmyk" {
		t.Fatalf("entry 0 value = %q, want cmyk", ast.Entries[0].Scalar.Str)
	}
	if ast.Entries[1].Block == nil || ast.Entries[1].Key != nil {
		t.Fatalf("entry 1 must be an anonymous block: %+v", ast.Entries[1])
	}
}

func TestParseAnonymousBlock(t *testing.T) {
	ast, _ := mustParse(t, "{ a = 1 }\n{ b = 2 }\n")
	if len(ast.Entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(ast.Entries))
	}
	for i, e := range ast.Entries {
		if e.Key != nil || e.Op != nil || e.Block == nil {
			t.Fatalf("entry %d must be an anonymous block: %+v", i, e)
		}
	}
}

func TestParseEmptyAndWhitespaceOnlyInput(t *testing.T) {
	for _, src := range []string{"", "\n", "   \t\r\n  ", "# only a comment\n"} {
		ast, rec, err := Parse(src)
		if err != nil {
			t.Fatalf("%q: %v", src, err)
		}
		if len(rec) != 0 {
			t.Fatalf("%q: unexpected recoveries %+v", src, rec)
		}
		if len(ast.Entries) != 0 {
			t.Fatalf("%q: expected no entries, got %d", src, len(ast.Entries))
		}
	}
}

func TestParseLineNumbersOnEntries(t *testing.T) {
	ast, _ := mustParse(t, "a = 1\n\nb = {\n  c = 2\n}\n")
	if ast.Entries[0].Line != 1 {
		t.Errorf("a line = %d, want 1", ast.Entries[0].Line)
	}
	if ast.Entries[1].Line != 3 {
		t.Errorf("b line = %d, want 3", ast.Entries[1].Line)
	}
	if inner := ast.Entries[1].Block.Entries[0]; inner.Line != 4 {
		t.Errorf("c line = %d, want 4", inner.Line)
	}
}

func TestRecoveryStrayRBraceAtTopLevel(t *testing.T) {

	ast, rec := mustParse(t, "a = 1\n}\nb = 2\n")
	if len(rec) != 1 || rec[0].Code != RecoveryStrayRBrace {
		t.Fatalf("expected one stray-rbrace recovery, got %+v", rec)
	}
	if rec[0].Line != 2 {
		t.Errorf("recovery line = %d, want 2", rec[0].Line)
	}
	if len(ast.Entries) != 2 || key(ast.Entries[0]) != "a" || key(ast.Entries[1]) != "b" {
		t.Fatalf("entries around the stray brace were lost: %d entries", len(ast.Entries))
	}
}

func TestRecoveryEmptyValueBeforeCloseBrace(t *testing.T) {

	ast, rec := mustParse(t, "outer = {\n  inner =\n}\ntail = 1\n")
	if len(rec) != 1 || rec[0].Code != RecoveryEmptyValue {
		t.Fatalf("expected one empty-value recovery, got %+v", rec)
	}
	if rec[0].Details["key"] != "inner" || rec[0].Details["terminator"] != "}" {
		t.Fatalf("recovery details wrong: %+v", rec[0].Details)
	}
	if len(ast.Entries) != 2 {
		t.Fatalf("terminator was consumed — outer block swallowed the tail: %d entries", len(ast.Entries))
	}
	inner := ast.Entries[0].Block.Entries[0]
	if key(inner) != "inner" || inner.Scalar.Kind != ScalarStr || inner.Scalar.Str != "" {
		t.Fatalf("empty value entry wrong: %+v", inner)
	}
	if key(ast.Entries[1]) != "tail" {
		t.Fatalf("tail entry lost: %+v", ast.Entries[1])
	}
}

func TestRecoveryEmptyValueAtEOF(t *testing.T) {
	ast, rec := mustParse(t, "a = 1\ndangling =")
	if len(rec) != 1 || rec[0].Code != RecoveryEmptyValue {
		t.Fatalf("expected one empty-value recovery, got %+v", rec)
	}
	if rec[0].Details["terminator"] != "EOF" {
		t.Fatalf("terminator = %v, want EOF", rec[0].Details["terminator"])
	}
	if len(ast.Entries) != 2 || key(ast.Entries[1]) != "dangling" {
		t.Fatalf("dangling entry lost: %d entries", len(ast.Entries))
	}
}

func TestUnclosedBlockAtEOFIsRecoveredNotDropped(t *testing.T) {

	ast, rec := mustParse(t, "a = {\n b = 1\n")
	if len(rec) != 1 || rec[0].Code != RecoveryUnclosedBlock {
		t.Fatalf("expected one unclosed-block recovery, got %+v", rec)
	}

	if rec[0].Line != 1 {
		t.Errorf("recovery line = %d, want 1 (the line of the unclosed `{`)", rec[0].Line)
	}
	if rec[0].Details["key"] != "a" {
		t.Errorf("recovery must name the block: %+v", rec[0].Details)
	}
	if rec[0].Details["eof_line"] != 3 {
		t.Errorf("eof_line = %v, want 3", rec[0].Details["eof_line"])
	}
	if len(ast.Entries) != 1 || key(ast.Entries[0]) != "a" || ast.Entries[0].Block == nil {
		t.Fatalf("outer entry lost: %+v", ast.Entries)
	}
	inner := ast.Entries[0].Block.Entries
	if len(inner) != 1 || key(inner[0]) != "b" || inner[0].Scalar.StrValue() != "1" {
		t.Fatalf("body of the repaired block lost: %+v", inner)
	}
}

func TestUnclosedAnonymousAndQualifiedBlocksAreRecovered(t *testing.T) {

	anon, rec := mustParse(t, "{ a = 1\n")
	if len(rec) != 1 || rec[0].Code != RecoveryUnclosedBlock || rec[0].Details["key"] != "" {
		t.Fatalf("anonymous block: %+v", rec)
	}
	if len(anon.Entries) != 1 || anon.Entries[0].Block == nil ||
		len(anon.Entries[0].Block.Entries) != 1 {
		t.Fatalf("anonymous block body lost: %+v", anon.Entries)
	}
	qual, rec := mustParse(t, "color = rgb { 153 0 51\n")
	if len(rec) != 1 || rec[0].Code != RecoveryUnclosedBlock {
		t.Fatalf("qualified block: %+v", rec)
	}
	e := qual.Entries[0]
	if e.Qualifier == nil || *e.Qualifier != "rgb" || e.Block == nil ||
		len(e.Block.Entries) != 3 {
		t.Fatalf("rgb qualifier or body lost: %+v", e)
	}
}

func TestNestedUnclosedBlocksReportOneRecoveryEach(t *testing.T) {

	ast, rec := mustParse(t, "a = {\n b = {\n  c = {\n   d = 1\n")
	if len(rec) != 3 {
		t.Fatalf("want 3 recoveries (one per unclosed block), got %+v", rec)
	}
	wantLines := []int{3, 2, 1}
	for i, r := range rec {
		if r.Code != RecoveryUnclosedBlock || r.Line != wantLines[i] {
			t.Fatalf("recovery %d = %+v, want unclosed-block at line %d", i, r, wantLines[i])
		}
	}
	deepest := ast.Entries[0].Block.Entries[0].Block.Entries[0].Block.Entries
	if len(deepest) != 1 || key(deepest[0]) != "d" {
		t.Fatalf("deepest entry lost: %+v", deepest)
	}
}

func TestEOFCommentsAreNotDuplicatedPerUnclosedBlock(t *testing.T) {

	ast, rec := mustParse(t, "a = {\n b = {\n# tail note\n")
	if len(rec) != 2 {
		t.Fatalf("want 2 recoveries, got %+v", rec)
	}
	var seen int
	var walk func(*Block)
	walk = func(b *Block) {
		for _, c := range append(append([]Comment{}, b.Trivia.Leading...), b.Trivia.Trailing...) {
			if c.Text == "# tail note" {
				seen++
			}
		}
		for _, e := range b.Entries {
			for _, c := range append(append([]Comment{}, e.Trivia.Leading...), e.Trivia.Trailing...) {
				if c.Text == "# tail note" {
					seen++
				}
			}
			if e.Block != nil {
				walk(e.Block)
			}
		}
	}
	walk(ast)
	if seen != 1 {
		t.Fatalf("end-of-file comment appears %d times, want exactly 1", seen)
	}
}

func TestRecoveriesBeforeAnUnclosedBlockAreKept(t *testing.T) {

	_, rec := mustParse(t, "}\na = {\n")
	if len(rec) != 2 {
		t.Fatalf("want stray-rbrace + unclosed-block, got %+v", rec)
	}
	if rec[0].Code != RecoveryStrayRBrace || rec[1].Code != RecoveryUnclosedBlock {
		t.Fatalf("codes wrong / order wrong: %+v", rec)
	}
}

func TestCloseBlockGuardKeepsTheVerbatimParseErrorMessage(t *testing.T) {

	p := &parser{tokens: []*Token{{Type: "STRING", Value: "junk", Line: 7}}}
	_, err := p.closeBlock(&Token{Type: "LBRACE", Value: "{", Line: 3}, "k")
	if err == nil {
		t.Fatal("the guard must still fail rather than invent a repair")
	}
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("error type = %T, want *ParseError", err)
	}
	const want = "Expected RBRACE, got STRING 'junk' at line 7"
	if pe.Msg != want {
		t.Fatalf("message = %q, want %q", pe.Msg, want)
	}
	if len(p.recoveries) != 0 {
		t.Fatalf("the guard must not record a repair it did not make: %+v", p.recoveries)
	}
}

func TestStrayRBraceInsideBlockClosesIt(t *testing.T) {

	ast, rec := mustParse(t, "outer = { a = 1 }\n")
	if len(rec) != 0 {
		t.Fatalf("nested close brace must not be reported as stray: %+v", rec)
	}
	if len(ast.Entries) != 1 || ast.Entries[0].Block == nil {
		t.Fatalf("outer block wrong: %+v", ast.Entries)
	}
}

func TestUnknownTokensAreSkippedNotFatal(t *testing.T) {

	ast, rec := mustParse(t, "a = 1\n!\nb = 2\n")
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	if len(ast.Entries) != 2 {
		t.Fatalf("an UNKNOWN token must not eat neighbouring entries: %d entries", len(ast.Entries))
	}
}
