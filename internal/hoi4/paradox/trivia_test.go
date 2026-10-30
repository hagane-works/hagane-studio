package paradox

import (
	"testing"
)

func TestLeadingCommentAttachesToNextEntry(t *testing.T) {
	ast, _ := mustParse(t, "# header one\n# header two\na = 1\n")
	e := ast.Entries[0]
	if len(e.Trivia.Leading) != 2 {
		t.Fatalf("got %d leading comments, want 2: %+v", len(e.Trivia.Leading), e.Trivia.Leading)
	}
	if e.Trivia.Leading[0].Text != "# header one" || e.Trivia.Leading[1].Text != "# header two" {
		t.Fatalf("leading comment text/order wrong: %+v", e.Trivia.Leading)
	}
	for i, c := range e.Trivia.Leading {
		if c.Inline {
			t.Errorf("comment %d on its own line must not be Inline", i)
		}
		if c.Line != i+1 {
			t.Errorf("comment %d line = %d, want %d", i, c.Line, i+1)
		}
		if c.Column != 0 {
			t.Errorf("comment %d column = %d, want 0", i, c.Column)
		}
	}
}

func TestTrailingCommentAttachesToPrecedingEntry(t *testing.T) {
	ast, _ := mustParse(t, "a = 1 # after a\nb = 2\n")
	a, b := ast.Entries[0], ast.Entries[1]
	if len(a.Trivia.Trailing) != 1 || a.Trivia.Trailing[0].Text != "# after a" {
		t.Fatalf("trailing comment lost from `a`: %+v", a.Trivia)
	}
	if !a.Trivia.Trailing[0].Inline {
		t.Fatal("a same-line comment must be Inline=true")
	}
	if len(b.Trivia.Leading) != 0 {
		t.Fatalf("inline comment leaked forward onto `b`: %+v", b.Trivia.Leading)
	}
}

func TestCommentWhitespaceIsRecorded(t *testing.T) {

	ast, _ := mustParse(t, "a = 1\t\t# tabbed\n\t\t# indented own-line\nb = 2\n")
	tr := ast.Entries[0].Trivia.Trailing
	if len(tr) != 1 || tr[0].WS != "\t\t" {
		t.Fatalf("inline comment WS = %q, want two tabs: %+v", tr[0].WS, tr)
	}
	ld := ast.Entries[1].Trivia.Leading
	if len(ld) != 1 || ld[0].WS != "\t\t" {
		t.Fatalf("own-line comment WS = %q, want two tabs: %+v", ld[0].WS, ld)
	}
	if ld[0].Column != 2 {
		t.Fatalf("own-line comment Column = %d, want 2", ld[0].Column)
	}
}

func TestCommentContainingBracesIsNotParsed(t *testing.T) {

	ast, rec := mustParse(t, "color = { 106  119  89 } # { 76  97  121 }\n")
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	if len(ast.Entries) != 1 {
		t.Fatalf("comment produced extra entries: %d", len(ast.Entries))
	}
	body := ast.Entries[0].Block
	if body == nil || len(body.Entries) != 3 {
		t.Fatalf("colour block must hold exactly 3 channels, got %+v", body)
	}
	got := []string{}
	for _, e := range body.Entries {
		got = append(got, e.Scalar.StrValue())
	}
	if got[0] != "106" || got[1] != "119" || got[2] != "89" {
		t.Fatalf("colour channels = %v, want [106 119 89]", got)
	}
	tr := ast.Entries[0].Trivia.Trailing
	if len(tr) != 1 || tr[0].Text != "# { 76  97  121 }" {
		t.Fatalf("the brace-bearing comment was not captured verbatim: %+v", tr)
	}
}

func TestCommentInsideBlockIsKept(t *testing.T) {
	ast, _ := mustParse(t, "outer = {\n\t# why this exists\n\ta = 1 # inline\n}\n")
	inner := ast.Entries[0].Block.Entries[0]
	if len(inner.Trivia.Leading) != 1 || inner.Trivia.Leading[0].Text != "# why this exists" {
		t.Fatalf("in-block leading comment lost: %+v", inner.Trivia.Leading)
	}
	if len(inner.Trivia.Trailing) != 1 || inner.Trivia.Trailing[0].Text != "# inline" {
		t.Fatalf("in-block inline comment lost: %+v", inner.Trivia.Trailing)
	}
}

func TestCommentBeforeCloseBraceIsKept(t *testing.T) {

	ast, _ := mustParse(t, "outer = {\n\ta = 1\n\t# trailing note\n}\n")
	blk := ast.Entries[0].Block
	found := false
	for _, c := range blk.Trivia.Trailing {
		if c.Text == "# trailing note" {
			found = true
		}
	}
	if !found {
		t.Fatalf("comment before `}` was dropped: block trivia = %+v", blk.Trivia)
	}
}

func TestCommentAtEndOfFileIsKept(t *testing.T) {
	ast, _ := mustParse(t, "a = 1\n# last word\n")
	found := false
	for _, c := range ast.Trivia.Trailing {
		if c.Text == "# last word" {
			found = true
		}
	}
	if !found {
		t.Fatalf("EOF comment dropped: top-level trivia = %+v", ast.Trivia)
	}
}

func TestCommentTextExcludesCR(t *testing.T) {

	ast, _ := mustParse(t, "a = 1 # note\r\nb = 2\r\n")
	tr := ast.Entries[0].Trivia.Trailing
	if len(tr) != 1 || tr[0].Text != "# note" {
		t.Fatalf("CR leaked into comment text: %q", tr[0].Text)
	}
}

func TestBannerCommentBlockSurvives(t *testing.T) {

	src := "################################################\n" +
		"### search_filters = {FOCUS_FILTER_POLITICAL}\n" +
		"### search_filters = {FOCUS_FILTER_RESEARCH}\n" +
		"\n" +
		"# CONSTANT FOR INNER CIRCLE TIMES\n" +
		"@inner_circle_time_tier_1 = 20\n" +
		"@inner_circle_time_tier_2 = 20\n"
	ast, rec := mustParse(t, src)
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	if len(ast.Entries) != 2 {
		t.Fatalf("banner comments produced %d entries, want 2", len(ast.Entries))
	}
	first := ast.Entries[0]
	if key(first) != "@inner_circle_time_tier_1" {
		t.Fatalf("@constant key mangled: %q", key(first))
	}
	if first.Scalar.Kind != ScalarInt || first.Scalar.StrValue() != "20" {
		t.Fatalf("@constant value wrong: %+v", first.Scalar)
	}
	if len(first.Trivia.Leading) != 4 {
		t.Fatalf("banner comments lost: got %d, want 4: %+v",
			len(first.Trivia.Leading), first.Trivia.Leading)
	}

	if first.Trivia.Leading[1].Text != "### search_filters = {FOCUS_FILTER_POLITICAL}" {
		t.Fatalf("assignment-shaped comment mangled: %q", first.Trivia.Leading[1].Text)
	}
}

func TestTriviaInOrderConcatenatesLeadingThenTrailing(t *testing.T) {
	p := Payload{
		Leading:  []Comment{{Text: "# l1"}, {Text: "# l2"}},
		Trailing: []Comment{{Text: "# t1"}},
	}
	got := triviaInOrder(p)
	if len(got) != 3 || got[0].Text != "# l1" || got[1].Text != "# l2" || got[2].Text != "# t1" {
		t.Fatalf("trivia_in_order wrong: %+v", got)
	}

	got = append(got, Comment{Text: "# x"})
	if len(p.Leading) != 2 {
		t.Fatal("triviaInOrder result aliases p.Leading")
	}
}

func TestCaptureCommentInlineDetection(t *testing.T) {
	cases := []struct {
		src        string
		wantInline bool
		wantWS     string
		wantCol    int
	}{
		{"# top", false, "", 0},
		{"  # indented", false, "  ", 2},
		{"a = 1 # after", true, " ", 6},
		{"a = 1\t# after", true, "\t", 6},
	}
	for _, c := range cases {
		runes := []rune(c.src)
		idx := -1
		for i, r := range runes {
			if r == '#' {
				idx = i
				break
			}
		}
		if idx < 0 {
			t.Fatalf("no # in %q", c.src)
		}
		got, end := captureComment(runes, idx, 1)
		if end != len(runes) {
			t.Errorf("%q: end = %d, want %d", c.src, end, len(runes))
		}
		if got.Inline != c.wantInline {
			t.Errorf("%q: Inline = %v, want %v", c.src, got.Inline, c.wantInline)
		}
		if got.WS != c.wantWS {
			t.Errorf("%q: WS = %q, want %q", c.src, got.WS, c.wantWS)
		}
		if got.Column != c.wantCol {
			t.Errorf("%q: Column = %d, want %d", c.src, got.Column, c.wantCol)
		}
	}
}
