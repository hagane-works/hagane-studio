package ideas

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func pyIsPrintable(r rune) bool {
	if r == ' ' {
		return true
	}
	if unicode.In(r, unicode.Cc, unicode.Cf, unicode.Cs, unicode.Co,
		unicode.Zl, unicode.Zp, unicode.Zs) {
		return false
	}

	return unicode.In(r, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S)
}

func pyStrRepr(s string) string {
	quote := '\''
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
		default:
			if pyIsPrintable(r) {
				b.WriteRune(r)
			} else if r < 0x100 {
				fmt.Fprintf(&b, `\x%02x`, r)
			} else if r < 0x10000 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				fmt.Fprintf(&b, `\U%08x`, r)
			}
		}
	}
	b.WriteRune(quote)
	return b.String()
}

func pyOptStrRepr(p *string) string {
	if p == nil {
		return "None"
	}
	return pyStrRepr(*p)
}

func pyBoolRepr(v bool) string {
	if v {
		return "True"
	}
	return "False"
}

func pyScalarRepr(s paradox.Scalar) string {
	if s.Kind == paradox.ScalarStr {
		return pyStrRepr(s.Str)
	}
	return importsvc.PyScalarStr(s)
}

func pyCommentRepr(c paradox.Comment) string {
	return "CommentTrivia(text=" + pyStrRepr(c.Text) +
		", line=" + strconv.Itoa(c.Line) +
		", column=" + strconv.Itoa(c.Column) +
		", inline=" + pyBoolRepr(c.Inline) +
		", whitespace=" + pyStrRepr(c.WS) + ")"
}

func pyCommentListRepr(cs []paradox.Comment) string {
	parts := make([]string, 0, len(cs))
	for _, c := range cs {
		parts = append(parts, pyCommentRepr(c))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func pyPayloadRepr(p paradox.Payload) string {
	return "TriviaPayload(leading=" + pyCommentListRepr(p.Leading) +
		", trailing=" + pyCommentListRepr(p.Trailing) + ")"
}

func pyEntryRepr(e *paradox.Entry) string {
	var value string
	if e.Block != nil {
		value = pyBlockRepr(e.Block)
	} else {
		value = pyScalarRepr(e.Scalar)
	}
	return "Entry(key=" + pyOptStrRepr(e.Key) +
		", op=" + pyOptStrRepr(e.Op) +
		", value=" + value +
		", line=" + strconv.Itoa(e.Line) +
		", quoted=" + pyBoolRepr(e.Quoted) +
		", qualifier=" + pyOptStrRepr(e.Qualifier) +
		", trivia=" + pyPayloadRepr(e.Trivia) + ")"
}

func pyBlockRepr(b *paradox.Block) string {
	parts := make([]string, 0, len(b.Entries))
	for _, e := range b.Entries {
		parts = append(parts, pyEntryRepr(e))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
