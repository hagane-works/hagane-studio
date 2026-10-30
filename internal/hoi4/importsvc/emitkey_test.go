package importsvc

import (
	"strings"
	"testing"

	"haganeworks/internal/hoi4/paradox"
)

func TestEmitKeyQuotesOnlyWhenNecessary(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want string
	}{

		{"space", "two words", `"two words"`},
		{"empty", "", `""`},

		{"newline", "bad\nloc", "\"bad\nloc\""},
		{"tab", "a\tb", "\"a\tb\""},
		{"rbrace", "has}brace", `"has}brace"`},
		{"lbrace", "has{brace", `"has{brace"`},
		{"equals", "has=eq", `"has=eq"`},
		{"hash", "has#hash", `"has#hash"`},

		{"slash", "gfx/interface/x", "gfx/interface/x"},
		{"backslash", `a\b`, `a\b`},

		{"quote", `a"b`, `"a\"b"`},
		{"unit_sep", "a\x1fb", "\"a\x1fb\""},

		{"plain", "infantry_equipment_0", "infantry_equipment_0"},
		{"digits", "500", "500"},
		{"at_const", "@speed", "@speed"},
		{"dotted", "usa_trait_overlay.1", "usa_trait_overlay.1"},
		{"cjk", "中文键", "中文键"},
	}
	for _, c := range cases {
		if got := emitKey(c.key); got != c.want {
			t.Errorf("%s: emitKey(%q) = %q, want %q", c.name, c.key, got, c.want)
		}
	}
}

func TestBareKeyStaysOneToken(t *testing.T) {
	var candidates []rune
	for r := rune(1); r < 128; r++ {
		candidates = append(candidates, r)
	}
	candidates = append(candidates, 0x1c, 0x1f, 0x85, 0xa0, 0x2028, '中')

	for _, r := range candidates {
		key := "a" + string(r) + "b"

		var got []*paradox.Token
		for _, tk := range paradox.Tokenize([]rune(emitKey(key))) {
			if tk.Type != "EOF" {
				got = append(got, tk)
			}
		}

		wantExact := !strings.Contains(key, `"`)
		if len(got) != 1 || (wantExact && got[0].Value != key) {
			vals := make([]string, 0, len(got))
			for _, tk := range got {
				vals = append(vals, tk.Value)
			}
			t.Errorf("U+%04X: emitKey(%q) = %q,重新 tokenize 得到 %d 个 token %q,期望单个 %q",
				r, key, emitKey(key), len(got), vals, key)
		}

		if !needsQuoteKey(key) {
			var bare []*paradox.Token
			for _, tk := range paradox.Tokenize([]rune(key)) {
				if tk.Type != "EOF" {
					bare = append(bare, tk)
				}
			}
			if len(bare) != 1 || bare[0].Value != key {
				t.Errorf("U+%04X: needsQuoteKey(%q)=false 但裸写被撕成 %d 个 token",
					r, key, len(bare))
			}
		}
	}
}
