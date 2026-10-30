package events

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

type PyError struct{ Type string }

func (e *PyError) Error() string { return e.Type }

func pyIsSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return true
	}
	return unicode.IsSpace(r)
}

func pyStrip(s string) string { return strings.TrimFunc(s, pyIsSpace) }

func pyIntCoerce(sc paradox.Scalar) (*big.Int, bool) {
	a := importsvc.CoerceInt(&sc, 0)
	b := importsvc.CoerceInt(&sc, 1)
	if a.Cmp(b) != 0 {
		return nil, false
	}
	return a, true
}

func pyIntStr(s string) (*big.Int, bool) {
	return pyIntCoerce(paradox.Scalar{Kind: paradox.ScalarStr, Str: s})
}

func pyFloatStr(sc paradox.Scalar) (float64, bool) {
	ss := paradox.Scalar{Kind: paradox.ScalarStr, Str: importsvc.PyScalarStr(sc)}
	a, _ := importsvc.CoerceFloat(&ss, 0)
	b, _ := importsvc.CoerceFloat(&ss, 1)
	if a == b || (math.IsNaN(a) && math.IsNaN(b)) {
		return a, true
	}
	return 0, false
}

func pyIntFloatStr(sc paradox.Scalar) (*big.Int, string) {
	f, ok := pyFloatStr(sc)
	if !ok {
		return nil, "ValueError"
	}
	if math.IsInf(f, 0) {
		return nil, "OverflowError"
	}
	if math.IsNaN(f) {
		return nil, "ValueError"
	}
	z, _ := new(big.Float).SetFloat64(f).Int(nil)
	return z, ""
}

func expandAtConstants(ast *paradox.Block) {
	registry := map[string]paradox.Scalar{}
	for _, e := range ast.Entries {
		if e.Block != nil {
			continue
		}
		if e.Key == nil || *e.Key == "" || !strings.HasPrefix(*e.Key, "@") {
			continue
		}
		registry[*e.Key] = e.Scalar
	}
	if len(registry) == 0 {
		return
	}
	var walk func(entries []*paradox.Entry)
	walk = func(entries []*paradox.Entry) {
		for _, e := range entries {
			if e.Block != nil {
				walk(e.Block.Entries)
			} else if e.Scalar.Kind == paradox.ScalarStr && strings.HasPrefix(e.Scalar.Str, "@") {
				if v, ok := registry[e.Scalar.Str]; ok {
					e.Scalar = v
				}
			}
		}
	}
	walk(ast.Entries)
}

func findOne(entries []*paradox.Entry, key string) *paradox.Entry {
	for _, e := range entries {
		if e.Key != nil && *e.Key == key {
			return e
		}
	}
	return nil
}

func ppScalar(entries []*paradox.Entry, key string) *paradox.Scalar {
	e := findOne(entries, key)
	if e == nil || e.Block != nil {
		return nil
	}
	sc := e.Scalar
	return &sc
}

func ppScalarStr(entries []*paradox.Entry, key string) string {
	if sc := ppScalar(entries, key); sc != nil {
		return importsvc.PyScalarStr(*sc)
	}
	return ""
}

func ppBlock(entries []*paradox.Entry, key string) *paradox.Block {
	e := findOne(entries, key)
	if e == nil || e.Block == nil {
		return nil
	}
	return e.Block
}

func blockTruthy(b *paradox.Block) bool { return b != nil && len(b.Entries) > 0 }

func pyYesEntry(e *paradox.Entry) bool {
	return strings.ToLower(pyStrEntryValue(e)) == "yes"
}

func pyStrEntryValue(e *paradox.Entry) string {
	if e.Block != nil {
		return pyReprEntryList(e.Block)
	}
	return importsvc.PyScalarStr(e.Scalar)
}

func pyReprEntryList(b *paradox.Block) string {
	parts := make([]string, 0, len(b.Entries))
	for _, e := range b.Entries {
		parts = append(parts, pyReprEntry(e))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func pyReprEntry(e *paradox.Entry) string {
	var val string
	if e.Block != nil {
		val = pyReprEntryList(e.Block)
	} else {
		switch e.Scalar.Kind {
		case paradox.ScalarInt:
			val = e.Scalar.Int.String()
		case paradox.ScalarFloat:
			val = importsvc.PyFloatReprStr(e.Scalar.Float)
		default:
			val = pyStrRepr(e.Scalar.Str)
		}
	}
	return fmt.Sprintf("Entry(key=%s, op=%s, value=%s, line=%d, quoted=%s, qualifier=%s, trivia=%s)",
		pyReprStrOrNone(e.Key), pyReprStrOrNone(e.Op), val, e.Line,
		pyBoolRepr(e.Quoted), pyReprStrOrNone(e.Qualifier), pyReprPayload(e.Trivia))
}

func pyReprPayload(p paradox.Payload) string {
	return "TriviaPayload(leading=" + pyReprComments(p.Leading) +
		", trailing=" + pyReprComments(p.Trailing) + ")"
}

func pyReprComments(cs []paradox.Comment) string {
	parts := make([]string, 0, len(cs))
	for _, c := range cs {
		parts = append(parts, fmt.Sprintf(
			"CommentTrivia(text=%s, line=%d, column=%d, inline=%s, whitespace=%s)",
			pyStrRepr(c.Text), c.Line, c.Column, pyBoolRepr(c.Inline), pyStrRepr(c.WS)))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func pyBoolRepr(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

func pyReprStrOrNone(p *string) string {
	if p == nil {
		return "None"
	}
	return pyStrRepr(*p)
}

func pyIsPrintable(r rune) bool {
	if r == ' ' {
		return true
	}
	return unicode.IsGraphic(r) && !unicode.Is(unicode.Zs, r)
}

func pyStrRepr(s string) string {
	quote := rune('\'')
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = '"'
	}
	var b strings.Builder
	b.WriteRune(quote)
	for _, r := range s {
		switch {
		case r == quote || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case pyIsPrintable(r):
			b.WriteRune(r)
		case r < 0x100:
			fmt.Fprintf(&b, `\x%02x`, r)
		case r < 0x10000:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			fmt.Fprintf(&b, `\U%08x`, r)
		}
	}
	b.WriteRune(quote)
	return b.String()
}

func scalarAny(sc paradox.Scalar) any {
	switch sc.Kind {
	case paradox.ScalarInt:
		return sc.Int
	case paradox.ScalarFloat:
		return sc.Float
	default:
		return sc.Str
	}
}
