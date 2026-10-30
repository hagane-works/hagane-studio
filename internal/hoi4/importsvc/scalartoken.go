package importsvc

import (
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type HOI4ScalarTokenError struct{ Msg string }

func (e *HOI4ScalarTokenError) Error() string { return e.Msg }

func tokenErr(msg string) error { return &HOI4ScalarTokenError{Msg: msg} }

func PyStrOrEmpty(v *paradox.Scalar) string {
	if v == nil {
		return ""
	}
	switch v.Kind {
	case paradox.ScalarInt:
		if v.Int == nil || v.Int.Sign() == 0 {
			return ""
		}
	case paradox.ScalarFloat:
		if v.Float == 0 {
			return ""
		}
	default:
		if v.Str == "" {
			return ""
		}
	}
	return v.StrValue()
}

func ParseHOI4ScalarToken(value string, allowEmpty bool) (*paradox.Entry, error) {
	token := pyStrip(value)
	if token == "" {
		if allowEmpty {
			return nil, nil
		}
		return nil, tokenErr("expected one non-empty HOI4 scalar token")
	}
	if len([]rune(token)) > 1000 || strings.ContainsAny(token, "\n\r") {
		return nil, tokenErr("expected one HOI4 scalar token")
	}

	ast, _, err := paradox.Parse("value = " + token)
	if err != nil {
		return nil, tokenErr("expected one HOI4 scalar token")
	}
	if len(ast.Entries) != 1 {
		return nil, tokenErr("expected one HOI4 scalar token")
	}
	entry := ast.Entries[0]
	if entry.Key == nil || *entry.Key != "value" || entry.Block != nil ||
		(entry.Scalar.Kind == paradox.ScalarStr && entry.Scalar.Str == "") {
		return nil, tokenErr("expected one HOI4 scalar token")
	}

	bare := &paradox.Entry{Scalar: entry.Scalar, Quoted: entry.Quoted}
	if pyStrip(ToScriptEntries([]*paradox.Entry{bare}, 0)) != token {
		return nil, tokenErr("expected one HOI4 scalar token")
	}
	return entry, nil
}

func ValidateHOI4ScalarToken(value string, allowEmpty bool) (string, error) {
	token := pyStrip(value)
	if _, err := ParseHOI4ScalarToken(token, allowEmpty); err != nil {
		return "", err
	}
	return token, nil
}

func RenderHOI4ScalarToken(value string) (string, error) {
	entry, err := ParseHOI4ScalarToken(value, false)
	if err != nil {
		return "", err
	}
	bare := &paradox.Entry{Scalar: entry.Scalar, Quoted: entry.Quoted}
	return pyStrip(ToScriptEntries([]*paradox.Entry{bare}, 0)), nil
}
