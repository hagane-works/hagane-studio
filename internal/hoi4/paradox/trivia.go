package paradox

type Comment struct {
	Text   string
	Line   int
	Column int
	Inline bool
	WS     string
}

type Payload struct {
	Leading  []Comment
	Trailing []Comment
}

func captureComment(text []rune, start, line int) (Comment, int) {
	end := start
	for end < len(text) && text[end] != '\r' && text[end] != '\n' {
		end++
	}
	lineStart := 0
	for i := start - 1; i >= 0; i-- {
		if text[i] == '\n' || text[i] == '\r' {
			lineStart = i + 1
			break
		}
	}
	prefix := string(text[lineStart:start])
	inline := trimSpacePy(prefix) != ""
	var ws string
	if inline {
		ws = trailingWhitespace(prefix)
	} else {
		ws = prefix
	}
	return Comment{
		Text:   string(text[start:end]),
		Line:   line,
		Column: start - lineStart,
		Inline: inline,
		WS:     ws,
	}, end
}

func trailingWhitespace(value string) string {
	r := []rune(value)
	pos := len(r)
	for pos > 0 && (r[pos-1] == ' ' || r[pos-1] == '\t') {
		pos--
	}
	return string(r[pos:])
}

func trimSpacePy(s string) string {
	isSpace := func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x1f, 0x85:
			return true
		}

		switch r {
		case 0xa0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000:
			return true
		}
		if r >= 0x2000 && r <= 0x200a {
			return true
		}
		return false
	}
	rs := []rune(s)
	i, j := 0, len(rs)
	for i < j && isSpace(rs[i]) {
		i++
	}
	for j > i && isSpace(rs[j-1]) {
		j--
	}
	return string(rs[i:j])
}

type triviaTracker struct {
	pending []Comment
}

func (t *triviaTracker) record(c Comment, previous *Payload) {
	if c.Inline && previous != nil {
		previous.Trailing = append(previous.Trailing, c)
	} else {
		t.pending = append(t.pending, c)
	}
}

func (t *triviaTracker) takeLeading() []Comment {
	out := t.pending
	t.pending = nil
	return out
}

func triviaInOrder(p Payload) []Comment {
	out := make([]Comment, 0, len(p.Leading)+len(p.Trailing))
	out = append(out, p.Leading...)
	out = append(out, p.Trailing...)
	return out
}
