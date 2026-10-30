package paradox

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

type ScalarKind int

const (
	ScalarStr ScalarKind = iota
	ScalarInt
	ScalarFloat
)

type Scalar struct {
	Kind  ScalarKind
	Str   string
	Int   *big.Int
	Float float64
}

func (s Scalar) StrValue() string {
	switch s.Kind {
	case ScalarInt:
		return s.Int.String()
	case ScalarFloat:
		return pyFloatRepr(s.Float)
	default:
		return s.Str
	}
}

type Block struct {
	Entries []*Entry
	Trivia  Payload
}

type Entry struct {
	Key       *string
	Op        *string
	Scalar    Scalar
	Block     *Block
	Line      int
	Quoted    bool
	Qualifier *string
	Trivia    Payload
}

type Recovery struct {
	Code    string
	Line    int
	Details map[string]any
}

const (
	RecoveryStrayRBrace = "parser_recovery_stray_rbrace"
	RecoveryEmptyValue  = "parser_recovery_empty_value"

	RecoveryUnclosedBlock = "parser_recovery_unclosed_block"
)

type ParseError struct{ Msg string }

func (e *ParseError) Error() string { return e.Msg }

var blockQualifiers = map[string]bool{"rgb": true, "hsv": true, "RGB": true, "HSV": true}

const maxBlockDepth = 1000

type parser struct {
	tokens         []*Token
	pos            int
	recoveryTrivia []Comment
	recoveries     []Recovery
	eofTriviaTaken bool
	depth          int
}

var eofToken = Token{Type: "EOF", Value: "", Line: 0}

func (p *parser) peek(offset int) *Token {
	idx := p.pos + offset
	if idx < len(p.tokens) {
		return p.tokens[idx]
	}
	return &eofToken
}

func (p *parser) advance() *Token {
	var t *Token
	if p.pos < len(p.tokens) {
		t = p.tokens[p.pos]
	} else {
		t = &eofToken
	}
	p.pos++
	return t
}

func (p *parser) expect(ttype string) (*Token, error) {
	t := p.advance()
	if t.Type != ttype {
		return nil, &ParseError{Msg: fmt.Sprintf(
			"Expected %s, got %s '%s' at line %d", ttype, t.Type, t.Value, t.Line)}
	}
	return t, nil
}

func toScalar(t *Token) Scalar {
	if t.Type == "NUMBER" {
		if strings.ContainsRune(t.Value, '.') {
			f, _ := strconv.ParseFloat(t.Value, 64)
			return Scalar{Kind: ScalarFloat, Float: f}
		}
		n := new(big.Int)
		n.SetString(t.Value, 10)
		return Scalar{Kind: ScalarInt, Int: n}
	}
	return Scalar{Kind: ScalarStr, Str: t.Value}
}

func (p *parser) takeRecoveryTrivia() []Comment {
	out := p.recoveryTrivia
	p.recoveryTrivia = nil
	return out
}

func (p *parser) takeEOFTrivia(t *Token) []Comment {
	if p.eofTriviaTaken {
		return nil
	}
	p.eofTriviaTaken = true
	return triviaInOrder(t.Trivia)
}

func (p *parser) carryRecoveryTrivia(t *Token) {
	p.recoveryTrivia = append(p.recoveryTrivia, triviaInOrder(t.Trivia)...)
}

func consumedEntryPayload(recovered []Comment, first *Token, remaining ...*Token) Payload {
	trailing := append([]Comment{}, first.Trivia.Trailing...)
	for _, t := range remaining {
		trailing = append(trailing, triviaInOrder(t.Trivia)...)
	}
	leading := append([]Comment{}, recovered...)
	leading = append(leading, first.Trivia.Leading...)
	return Payload{Leading: leading, Trailing: trailing}
}

func blockEntryPayload(recovered []Comment, first, close *Token) Payload {
	leading := append([]Comment{}, recovered...)
	leading = append(leading, first.Trivia.Leading...)
	return Payload{
		Leading:  leading,
		Trailing: append([]Comment{}, close.Trivia.Trailing...),
	}
}

func extendBlockOpenTrivia(block *Block, first *Token, remaining ...*Token) {
	block.Trivia.Leading = append(block.Trivia.Leading, first.Trivia.Trailing...)
	for _, t := range remaining {
		block.Trivia.Leading = append(block.Trivia.Leading, triviaInOrder(t.Trivia)...)
	}
}

func strp(s string) *string { return &s }

func (p *parser) closeBlock(open *Token, key string) (*Token, error) {
	switch t := p.peek(0); t.Type {
	case "RBRACE":
		return p.advance(), nil
	case "EOF":
		details := map[string]any{"key": key, "eof_line": t.Line}
		p.recoveries = append(p.recoveries, Recovery{
			Code: RecoveryUnclosedBlock, Line: open.Line, Details: details,
		})

		return &Token{Type: "RBRACE", Value: "}", Line: t.Line}, nil
	default:

		return p.expect("RBRACE")
	}
}

func (p *parser) parseBlock(topLevel bool) (*Block, error) {
	if !topLevel {
		p.depth++
		defer func() { p.depth-- }()
		if p.depth > maxBlockDepth {
			return nil, &ParseError{Msg: fmt.Sprintf(
				"Blocks nested deeper than %d levels at line %d", maxBlockDepth, p.peek(0).Line)}
		}
	}
	entries := &Block{}

	for {
		cur := p.peek(0)
		if cur.Type == "EOF" {
			entries.Trivia.Trailing = append(entries.Trivia.Trailing, p.takeRecoveryTrivia()...)
			entries.Trivia.Trailing = append(entries.Trivia.Trailing, p.takeEOFTrivia(cur)...)
			break
		}
		if cur.Type == "RBRACE" {
			if topLevel {

				p.recoveries = append(p.recoveries, Recovery{
					Code: RecoveryStrayRBrace, Line: cur.Line, Details: map[string]any{},
				})
				p.carryRecoveryTrivia(p.advance())
				continue
			}
			entries.Trivia.Trailing = append(entries.Trivia.Trailing, p.takeRecoveryTrivia()...)
			break
		}

		if cur.Type == "STRING" || cur.Type == "NUMBER" {
			recovered := p.takeRecoveryTrivia()
			nxt := p.peek(1)

			if nxt.Type == "OP" {
				keyTok := p.advance()
				opTok := p.advance()

				after := p.peek(0)
				switch {
				case after.Type == "LBRACE":
					openTok := p.advance()
					block, err := p.parseBlock(false)
					if err != nil {
						return nil, err
					}
					closeTok, err := p.closeBlock(openTok, keyTok.Value)
					if err != nil {
						return nil, err
					}
					extendBlockOpenTrivia(block, keyTok, opTok, openTok)
					block.Trivia.Trailing = append(block.Trivia.Trailing, closeTok.Trivia.Leading...)
					entry := &Entry{
						Key: strp(keyTok.Value), Op: strp(opTok.Value),
						Block: block, Line: keyTok.Line,
					}
					entry.Trivia = blockEntryPayload(recovered, keyTok, closeTok)
					entries.Entries = append(entries.Entries, entry)

				case after.Type == "STRING" && blockQualifiers[after.Value] && p.peek(1).Type == "LBRACE":

					qualTok := p.advance()
					openTok := p.advance()
					block, err := p.parseBlock(false)
					if err != nil {
						return nil, err
					}
					closeTok, err := p.closeBlock(openTok, keyTok.Value)
					if err != nil {
						return nil, err
					}
					extendBlockOpenTrivia(block, keyTok, opTok, qualTok, openTok)
					block.Trivia.Trailing = append(block.Trivia.Trailing, closeTok.Trivia.Leading...)
					entry := &Entry{
						Key: strp(keyTok.Value), Op: strp(opTok.Value),
						Block: block, Line: keyTok.Line,
						Qualifier: strp(qualTok.Value),
					}
					entry.Trivia = blockEntryPayload(recovered, keyTok, closeTok)
					entries.Entries = append(entries.Entries, entry)

				case after.Type == "RBRACE" || after.Type == "EOF":

					terminator := "}"
					if after.Type == "EOF" {
						terminator = "EOF"
					}
					p.recoveries = append(p.recoveries, Recovery{
						Code: RecoveryEmptyValue, Line: keyTok.Line,
						Details: map[string]any{"key": keyTok.Value, "terminator": terminator},
					})
					entry := &Entry{
						Key: strp(keyTok.Value), Op: strp(opTok.Value),
						Scalar: Scalar{Kind: ScalarStr, Str: ""}, Line: keyTok.Line,
					}
					entry.Trivia = consumedEntryPayload(recovered, keyTok, opTok)
					entries.Entries = append(entries.Entries, entry)

				default:
					valTok := p.advance()
					entry := &Entry{
						Key: strp(keyTok.Value), Op: strp(opTok.Value),
						Scalar: toScalar(valTok), Line: keyTok.Line,
						Quoted: valTok.Quoted,
					}
					entry.Trivia = consumedEntryPayload(recovered, keyTok, opTok, valTok)
					entries.Entries = append(entries.Entries, entry)
				}
			} else {

				tok := p.advance()
				entry := &Entry{
					Scalar: toScalar(tok), Line: tok.Line, Quoted: tok.Quoted,
				}
				entry.Trivia = consumedEntryPayload(recovered, tok)
				entries.Entries = append(entries.Entries, entry)
			}

		} else if cur.Type == "LBRACE" {

			recovered := p.takeRecoveryTrivia()
			openTok := p.advance()
			block, err := p.parseBlock(false)
			if err != nil {
				return nil, err
			}
			closeTok, err := p.closeBlock(openTok, "")
			if err != nil {
				return nil, err
			}
			block.Trivia.Leading = append(block.Trivia.Leading, openTok.Trivia.Trailing...)
			block.Trivia.Trailing = append(block.Trivia.Trailing, closeTok.Trivia.Leading...)
			entry := &Entry{Block: block, Line: cur.Line}
			entry.Trivia = blockEntryPayload(recovered, openTok, closeTok)
			entries.Entries = append(entries.Entries, entry)

		} else {
			p.carryRecoveryTrivia(p.advance())
		}
	}

	return entries, nil
}

func Parse(text string) (*Block, []Recovery, error) {
	runes := []rune(text)
	for len(runes) > 0 && runes[0] == 0xFEFF {
		runes = runes[1:]
	}
	tokens := Tokenize(runes)
	p := &parser{tokens: tokens}
	ast, err := p.parseBlock(true)
	if err != nil {
		return nil, p.recoveries, err
	}
	return ast, p.recoveries, nil
}
