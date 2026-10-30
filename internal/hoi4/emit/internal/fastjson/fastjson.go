package fastjson

import (
	"encoding/json"
	"unicode/utf8"
)

type Builder[V, O any] interface {
	Null() V
	Bool(bool) V
	String(string) V

	Number(literal string) (V, bool)

	Array(items []V) V
	NewObject() O

	Set(object O, key string, value V)
	Object(object O) V
}

func Parse[V, O any](data []byte, b Builder[V, O]) (V, bool) {
	p := parser[V, O]{data: data, b: b}
	p.skipSpace()
	value, ok := p.value()
	if !ok {
		var zero V
		return zero, false
	}
	p.skipSpace()
	if p.pos != len(p.data) {
		var zero V
		return zero, false
	}
	return value, true
}

const maxDepth = 10000

type parser[V, O any] struct {
	data  []byte
	pos   int
	depth int
	b     Builder[V, O]
}

func (p *parser[V, O]) skipSpace() {
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *parser[V, O]) value() (V, bool) {
	var zero V
	if p.pos >= len(p.data) {
		return zero, false
	}
	switch c := p.data[p.pos]; {
	case c == '{':
		return p.object()
	case c == '[':
		return p.array()
	case c == '"':
		s, ok := p.str()
		if !ok {
			return zero, false
		}
		return p.b.String(s), true
	case c == '-' || (c >= '0' && c <= '9'):
		literal, ok := p.number()
		if !ok {
			return zero, false
		}
		return p.b.Number(literal)
	case c == 't':
		return p.b.Bool(true), p.literal("true")
	case c == 'f':
		return p.b.Bool(false), p.literal("false")
	case c == 'n':
		return p.b.Null(), p.literal("null")
	}
	return zero, false
}

func (p *parser[V, O]) literal(word string) bool {
	if len(p.data)-p.pos < len(word) || string(p.data[p.pos:p.pos+len(word)]) != word {
		return false
	}
	p.pos += len(word)
	return true
}

func (p *parser[V, O]) object() (V, bool) {
	var zero V
	p.depth++
	if p.depth > maxDepth {
		return zero, false
	}
	p.pos++
	object := p.b.NewObject()
	p.skipSpace()
	if p.pos < len(p.data) && p.data[p.pos] == '}' {
		p.pos++
		p.depth--
		return p.b.Object(object), true
	}
	for {
		if p.pos >= len(p.data) || p.data[p.pos] != '"' {
			return zero, false
		}
		key, ok := p.str()
		if !ok {
			return zero, false
		}
		p.skipSpace()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return zero, false
		}
		p.pos++
		p.skipSpace()
		value, ok := p.value()
		if !ok {
			return zero, false
		}
		p.b.Set(object, key, value)
		p.skipSpace()
		if p.pos >= len(p.data) {
			return zero, false
		}
		switch p.data[p.pos] {
		case ',':
			p.pos++
			p.skipSpace()
		case '}':
			p.pos++
			p.depth--
			return p.b.Object(object), true
		default:
			return zero, false
		}
	}
}

func (p *parser[V, O]) array() (V, bool) {
	var zero V
	p.depth++
	if p.depth > maxDepth {
		return zero, false
	}
	p.pos++
	items := []V{}
	p.skipSpace()
	if p.pos < len(p.data) && p.data[p.pos] == ']' {
		p.pos++
		p.depth--
		return p.b.Array(items), true
	}
	for {
		value, ok := p.value()
		if !ok {
			return zero, false
		}
		items = append(items, value)
		p.skipSpace()
		if p.pos >= len(p.data) {
			return zero, false
		}
		switch p.data[p.pos] {
		case ',':
			p.pos++
			p.skipSpace()
		case ']':
			p.pos++
			p.depth--
			return p.b.Array(items), true
		default:
			return zero, false
		}
	}
}

func (p *parser[V, O]) str() (string, bool) {
	start := p.pos
	escaped := false
	i := start + 1
	for ; i < len(p.data); i++ {
		c := p.data[i]
		if c == '"' {
			break
		}
		if c < 0x20 {
			return "", false
		}
		if c == '\\' {
			escaped = true
			i++
		}
	}
	if i >= len(p.data) {
		return "", false
	}
	p.pos = i + 1
	raw := p.data[start+1 : i]
	if !utf8.Valid(raw) {
		return slowUnquote(p.data[start : i+1])
	}
	if !escaped {
		return string(raw), true
	}
	if s, ok := unescapeCommon(raw); ok {
		return s, true
	}
	return slowUnquote(p.data[start : i+1])
}

func slowUnquote(quoted []byte) (string, bool) {
	var s string
	if err := json.Unmarshal(quoted, &s); err != nil {
		return "", false
	}
	return s, true
}

func unescapeCommon(raw []byte) (string, bool) {
	out := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c != '\\' {
			out = append(out, c)
			continue
		}
		i++
		if i >= len(raw) {
			return "", false
		}
		switch raw[i] {
		case '"', '\\', '/':
			out = append(out, raw[i])
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'u':
			if i+4 >= len(raw) {
				return "", false
			}
			r := rune(0)
			for _, h := range raw[i+1 : i+5] {
				switch {
				case h >= '0' && h <= '9':
					r = r<<4 | rune(h-'0')
				case h >= 'a' && h <= 'f':
					r = r<<4 | rune(h-'a'+10)
				case h >= 'A' && h <= 'F':
					r = r<<4 | rune(h-'A'+10)
				default:
					return "", false
				}
			}
			if r >= 0xD800 && r <= 0xDFFF {
				return "", false
			}
			out = utf8.AppendRune(out, r)
			i += 4
		default:
			return "", false
		}
	}
	return string(out), true
}

func (p *parser[V, O]) number() (string, bool) {
	start := p.pos
	if p.data[p.pos] == '-' {
		p.pos++
	}
	switch {
	case p.pos < len(p.data) && p.data[p.pos] == '0':
		p.pos++
	case p.pos < len(p.data) && p.data[p.pos] >= '1' && p.data[p.pos] <= '9':
		p.digits()
	default:
		return "", false
	}
	if p.pos < len(p.data) && p.data[p.pos] == '.' {
		p.pos++
		if !p.digits() {
			return "", false
		}
	}
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			p.pos++
		}
		if !p.digits() {
			return "", false
		}
	}
	return string(p.data[start:p.pos]), true
}

func (p *parser[V, O]) digits() bool {
	start := p.pos
	for p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '9' {
		p.pos++
	}
	return p.pos > start
}
