package paradox

type Token struct {
	Type   string
	Value  string
	Line   int
	Quoted bool
	Trivia Payload
}

func isStopChar(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n', '{', '}', '=', '>', '<', '!', '"', '#':
		return true
	}
	return false
}

func isASCIIDigit(r rune) bool { return r >= '0' && r <= '9' }

func matchesInt(word []rune) bool {
	i := 0
	if len(word) > 0 && word[0] == '-' {
		i = 1
	}
	if i >= len(word) {
		return false
	}
	for ; i < len(word); i++ {
		if !isASCIIDigit(word[i]) {
			return false
		}
	}
	return true
}

func matchesFloat(word []rune) bool {
	i := 0
	if len(word) > 0 && word[0] == '-' {
		i = 1
	}
	start := i
	for i < len(word) && isASCIIDigit(word[i]) {
		i++
	}
	if i == start || i >= len(word) || word[i] != '.' {
		return false
	}
	i++
	start = i
	for i < len(word) && isASCIIDigit(word[i]) {
		i++
	}
	return i > start && i == len(word)
}

func Tokenize(text []rune) []*Token {
	var tokens []*Token
	tracker := &triviaTracker{}

	appendToken := func(t *Token) {
		t.Trivia.Leading = append(t.Trivia.Leading, tracker.takeLeading()...)
		tokens = append(tokens, t)
	}

	i := 0
	line := 1
	length := len(text)

	for i < length {
		c := text[i]

		if c == '\n' {
			line++
			i++
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '#' {
			comment, end := captureComment(text, i, line)
			i = end
			var prev *Payload
			if len(tokens) > 0 {
				prev = &tokens[len(tokens)-1].Trivia
			}
			tracker.record(comment, prev)
			continue
		}
		if c == '{' {
			appendToken(&Token{Type: "LBRACE", Value: "{", Line: line})
			i++
			continue
		}
		if c == '}' {
			appendToken(&Token{Type: "RBRACE", Value: "}", Line: line})
			i++
			continue
		}
		if c == '=' {
			if i+1 < length && text[i+1] == '=' {
				appendToken(&Token{Type: "OP", Value: "==", Line: line})
				i += 2
			} else {
				appendToken(&Token{Type: "OP", Value: "=", Line: line})
				i++
			}
			continue
		}
		if c == '>' {
			if i+1 < length && text[i+1] == '=' {
				appendToken(&Token{Type: "OP", Value: ">=", Line: line})
				i += 2
			} else {
				appendToken(&Token{Type: "OP", Value: ">", Line: line})
				i++
			}
			continue
		}
		if c == '<' {
			if i+1 < length && text[i+1] == '=' {
				appendToken(&Token{Type: "OP", Value: "<=", Line: line})
				i += 2
			} else {
				appendToken(&Token{Type: "OP", Value: "<", Line: line})
				i++
			}
			continue
		}
		if c == '!' && i+1 < length && text[i+1] == '=' {
			appendToken(&Token{Type: "OP", Value: "!=", Line: line})
			i += 2
			continue
		}

		if c == '"' {
			i++
			start := i
			for i < length && text[i] != '"' {
				if text[i] == '\\' && i+1 < length {

					i += 2
				} else {
					if text[i] == '\n' {
						line++
					}
					i++
				}
			}
			value := string(text[start:min(i, length)])
			appendToken(&Token{Type: "STRING", Value: value, Line: line, Quoted: true})
			if i < length {
				i++
			}
			continue
		}

		start := i
		for i < length && !isStopChar(text[i]) {
			i++
		}

		word := text[start:i]
		if len(word) == 0 {
			appendToken(&Token{Type: "UNKNOWN", Value: string(c), Line: line})
			i++
			continue
		}
		switch {
		case matchesInt(word):
			appendToken(&Token{Type: "NUMBER", Value: string(word), Line: line})
		case matchesFloat(word):
			appendToken(&Token{Type: "NUMBER", Value: string(word), Line: line})
		default:
			appendToken(&Token{Type: "STRING", Value: string(word), Line: line})
		}
	}

	appendToken(&Token{Type: "EOF", Value: "", Line: line})
	return tokens
}
