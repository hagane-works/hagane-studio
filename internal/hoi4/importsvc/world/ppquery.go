package world

import (
	"fmt"
	"math/big"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func scalarAny(s paradox.Scalar) any {
	switch s.Kind {
	case paradox.ScalarInt:
		return s.Int
	case paradox.ScalarFloat:
		return s.Float
	default:
		return s.Str
	}
}

func entVal(e *paradox.Entry) any {
	if e.Block != nil {
		return e.Block
	}
	return scalarAny(e.Scalar)
}

func pyStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case *big.Int:
		return t.String()
	case float64:
		return importsvc.PyFloatReprStr(t)
	case *paradox.Block:
		return entryListRepr(t)
	}
	return fmt.Sprintf("%v", v)
}

func truthy(v any) bool {
	switch t := v.(type) {
	case string:
		return t != ""
	case *big.Int:
		return t.Sign() != 0
	case float64:
		return t != 0
	case bool:
		return t
	case nil:
		return false
	}
	return true
}

func keyIs(e *paradox.Entry, k string) bool {
	return e.Key != nil && *e.Key == k
}

func ppFindAll(entries []*paradox.Entry, key string) []*paradox.Entry {
	var out []*paradox.Entry
	for _, e := range entries {
		if keyIs(e, key) {
			out = append(out, e)
		}
	}
	return out
}

func ppFindOne(entries []*paradox.Entry, key string) *paradox.Entry {
	for _, e := range entries {
		if keyIs(e, key) {
			return e
		}
	}
	return nil
}

func ppScalar(entries []*paradox.Entry, key string) (any, bool) {
	e := ppFindOne(entries, key)
	if e == nil || e.Block != nil {
		return nil, false
	}
	return scalarAny(e.Scalar), true
}

func ppBlock(entries []*paradox.Entry, key string) *paradox.Block {
	e := ppFindOne(entries, key)
	if e == nil || e.Block == nil {
		return nil
	}
	return e.Block
}

func ppBareValues(entries []*paradox.Entry) []string {
	out := []string{}
	for _, e := range entries {
		if e.Key == nil && e.Block == nil {
			out = append(out, pyStr(scalarAny(e.Scalar)))
		}
	}
	return out
}

func expandAtConstants(ast *paradox.Block) {
	registry := map[string]paradox.Scalar{}
	for _, e := range ast.Entries {
		if e.Block != nil {
			continue
		}
		if e.Key == nil || !strings.HasPrefix(*e.Key, "@") {
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
				continue
			}
			if e.Scalar.Kind == paradox.ScalarStr && strings.HasPrefix(e.Scalar.Str, "@") {
				if rep, ok := registry[e.Scalar.Str]; ok {
					e.Scalar = rep
				}
			}
		}
	}
	walk(ast.Entries)
}

func pyReprStr(s string) string {
	quote := byte('\'')
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = '"'
	}
	var b strings.Builder
	b.WriteByte(quote)
	for _, r := range s {
		switch {
		case r == rune(quote) || r == '\\':
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
	b.WriteByte(quote)
	return b.String()
}

func pyIsPrintable(r rune) bool {
	if r == ' ' {
		return true
	}
	return unicode.IsGraphic(r) && !unicode.Is(unicode.Zs, r)
}

func pyReprScalar(s paradox.Scalar) string {
	switch s.Kind {
	case paradox.ScalarInt:
		return s.Int.String()
	case paradox.ScalarFloat:
		return importsvc.PyFloatReprStr(s.Float)
	default:
		return pyReprStr(s.Str)
	}
}

func pyReprOptStr(p *string) string {
	if p == nil {
		return "None"
	}
	return pyReprStr(*p)
}

func pyReprBool(v bool) string {
	if v {
		return "True"
	}
	return "False"
}

func pyReprComment(c paradox.Comment) string {
	return "CommentTrivia(text=" + pyReprStr(c.Text) +
		", line=" + fmt.Sprint(c.Line) +
		", column=" + fmt.Sprint(c.Column) +
		", inline=" + pyReprBool(c.Inline) +
		", whitespace=" + pyReprStr(c.WS) + ")"
}

func pyReprComments(cs []paradox.Comment) string {
	parts := make([]string, 0, len(cs))
	for _, c := range cs {
		parts = append(parts, pyReprComment(c))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func pyReprPayload(p paradox.Payload) string {
	return "TriviaPayload(leading=" + pyReprComments(p.Leading) +
		", trailing=" + pyReprComments(p.Trailing) + ")"
}

func pyReprEntry(e *paradox.Entry) string {
	value := ""
	if e.Block != nil {
		value = entryListRepr(e.Block)
	} else {
		value = pyReprScalar(e.Scalar)
	}
	return "Entry(key=" + pyReprOptStr(e.Key) +
		", op=" + pyReprOptStr(e.Op) +
		", value=" + value +
		", line=" + fmt.Sprint(e.Line) +
		", quoted=" + pyReprBool(e.Quoted) +
		", qualifier=" + pyReprOptStr(e.Qualifier) +
		", trivia=" + pyReprPayload(e.Trivia) + ")"
}

func entryListRepr(b *paradox.Block) string {
	parts := make([]string, 0, len(b.Entries))
	for _, e := range b.Entries {
		parts = append(parts, pyReprEntry(e))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
