package importsvc

import (
	"math"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

func NeedsQuote(s string) bool {
	if s == "" {
		return true
	}
	return strings.ContainsAny(s, " /\\\"")
}

func EmitQuoted(value string, wasQuoted bool) string {
	if wasQuoted {
		return `"` + value + `"`
	}
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func renderLeading(comments []paradox.Comment, indent int) []string {
	prefix := strings.Repeat("\t", indent)
	out := make([]string, 0, len(comments))
	for _, c := range comments {
		out = append(out, prefix+c.Text)
	}
	return out
}

func renderAfterLine(line string, comments []paradox.Comment, indent int) []string {
	rendered := []string{line}
	prefix := strings.Repeat("\t", indent)
	attachedInline := false
	for _, c := range comments {
		if c.Inline && !attachedInline {
			gap := c.WS
			if gap == "" {
				gap = " "
			}
			rendered[0] = rendered[0] + gap + c.Text
			attachedInline = true
		} else {
			rendered = append(rendered, prefix+c.Text)
		}
	}
	return rendered
}

func hasTrivia(b *paradox.Block) bool {
	if len(b.Trivia.Leading) > 0 || len(b.Trivia.Trailing) > 0 {
		return true
	}
	for _, e := range b.Entries {
		if len(e.Trivia.Leading) > 0 || len(e.Trivia.Trailing) > 0 {
			return true
		}
	}
	return false
}

func fmtScriptScalar(s paradox.Scalar, quoted bool) string {
	if s.Kind != paradox.ScalarFloat {
		return PyScalarStr(s)
	}
	text := PyFloatReprStr(s.Float)
	if !strings.ContainsAny(text, "eE") {
		return text
	}
	if math.IsNaN(s.Float) || math.IsInf(s.Float, 0) {

		return text
	}
	return expandScientific(text)
}

func expandScientific(text string) string {
	idx := strings.IndexAny(text, "eE")
	mantissa, expPart := text[:idx], text[idx+1:]
	exp, err := strconv.Atoi(expPart)
	if err != nil {
		return text
	}
	sign := ""
	if strings.HasPrefix(mantissa, "-") {
		sign, mantissa = "-", mantissa[1:]
	}
	intPart, fracPart := mantissa, ""
	if dot := strings.IndexByte(mantissa, '.'); dot >= 0 {
		intPart, fracPart = mantissa[:dot], mantissa[dot+1:]
	}
	digits := intPart + fracPart

	point := len(intPart) + exp
	var out string
	switch {
	case point <= 0:
		out = "0." + strings.Repeat("0", -point) + digits
	case point >= len(digits):
		out = digits + strings.Repeat("0", point-len(digits)) + ".0"
	default:
		out = digits[:point] + "." + digits[point:]
	}
	if !strings.Contains(out, ".") {
		out += ".0"
	}
	return sign + out
}

const keyStopChars = " \t\r\n{}=><!\"#"

func needsQuoteKey(k string) bool {
	if k == "" {
		return true
	}
	if strings.ContainsAny(k, keyStopChars) {
		return true
	}

	for _, r := range k {
		if pyIsSpace(r) {
			return true
		}
	}
	return false
}

func emitKey(k string) string {
	if needsQuoteKey(k) {
		return EmitQuoted(k, false)
	}
	return k
}

func scalarText(s paradox.Scalar, quoted bool) string {
	if s.Kind == paradox.ScalarStr && (quoted || NeedsQuote(s.Str)) {
		return EmitQuoted(s.Str, quoted)
	}
	return fmtScriptScalar(s, quoted)
}

func ToScript(b *paradox.Block, indent int) string {
	lines := toScriptLines(b, indent)
	return strings.Join(lines, "\n")
}

func ToScriptEntries(entries []*paradox.Entry, indent int) string {
	return ToScript(&paradox.Block{Entries: entries}, indent)
}

func toScriptLines(b *paradox.Block, indent int) []string {
	var lines []string
	pfx := strings.Repeat("\t", indent)

	for _, e := range b.Entries {
		lines = append(lines, renderLeading(e.Trivia.Leading, indent)...)
		if e.Key == nil {
			if e.Block != nil {

				children := e.Block
				lines = append(lines, renderAfterLine(
					pfx+"{", children.Trivia.Leading, indent+1)...)
				nested := ToScript(children, indent+1)
				if nested != "" {
					lines = append(lines, nested)
				}
				lines = append(lines, renderAfterLine(pfx+"}", e.Trivia.Trailing, indent)...)
			} else {

				v := scalarText(e.Scalar, e.Quoted)
				lines = append(lines, renderAfterLine(pfx+v, e.Trivia.Trailing, indent)...)
			}
			continue
		}

		op := "="
		if e.Op != nil && *e.Op != "" {
			op = *e.Op
		}
		if e.Block != nil {
			qual := ""
			if e.Qualifier != nil && *e.Qualifier != "" {
				qual = *e.Qualifier + " "
			}
			children := e.Block
			compact := len(children.Entries) <= 6 && !hasTrivia(children)
			if compact {
				for _, c := range children.Entries {
					if c.Key != nil || c.Block != nil {
						compact = false
						break
					}
				}
			}
			if compact {
				parts := make([]string, 0, len(children.Entries))
				for _, c := range children.Entries {
					parts = append(parts, scalarText(c.Scalar, c.Quoted))
				}
				inner := strings.Join(parts, " ")
				line := pfx + emitKey(*e.Key) + " " + op + " " + qual + "{ " + inner + " }"
				lines = append(lines, renderAfterLine(line, e.Trivia.Trailing, indent)...)
			} else {
				openLine := pfx + emitKey(*e.Key) + " " + op + " " + qual + "{"
				lines = append(lines, renderAfterLine(
					openLine, children.Trivia.Leading, indent+1)...)
				nested := ToScript(children, indent+1)
				if nested != "" {
					lines = append(lines, nested)
				}
				lines = append(lines, renderAfterLine(pfx+"}", e.Trivia.Trailing, indent)...)
			}
		} else {
			v := scalarText(e.Scalar, e.Quoted)
			line := pfx + emitKey(*e.Key) + " " + op + " " + v
			lines = append(lines, renderAfterLine(line, e.Trivia.Trailing, indent)...)
		}
	}

	lines = append(lines, renderLeading(b.Trivia.Trailing, indent)...)
	return lines
}
