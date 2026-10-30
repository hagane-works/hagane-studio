package characters

import (
	"math"
	"math/big"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

type PyError struct {
	Type string
	Msg  string
}

func (e *PyError) Error() string { return e.Type + ": " + e.Msg }

func findAll(entries []*paradox.Entry, key string) []*paradox.Entry {
	var out []*paradox.Entry
	for _, e := range entries {
		if e.Key != nil && *e.Key == key {
			out = append(out, e)
		}
	}
	return out
}

func findOne(entries []*paradox.Entry, key string) *paradox.Entry {
	for _, e := range entries {
		if e.Key != nil && *e.Key == key {
			return e
		}
	}
	return nil
}

func scalarOr(entries []*paradox.Entry, key string, def paradox.Scalar) paradox.Scalar {
	e := findOne(entries, key)
	if e == nil || e.Block != nil {
		return def
	}
	return e.Scalar
}

func scalarFound(entries []*paradox.Entry, key string) (paradox.Scalar, bool) {
	e := findOne(entries, key)
	if e == nil || e.Block != nil {
		return paradox.Scalar{}, false
	}
	return e.Scalar, true
}

func blockOf(entries []*paradox.Entry, key string) *paradox.Block {
	e := findOne(entries, key)
	if e == nil || e.Block == nil {
		return nil
	}
	return e.Block
}

func blockTruthy(b *paradox.Block) bool { return b != nil && len(b.Entries) > 0 }

func bareValues(entries []*paradox.Entry) []string {
	out := []string{}
	for _, e := range entries {
		if e.Key == nil && e.Block == nil {
			out = append(out, importsvc.PyScalarStr(e.Scalar))
		}
	}
	return out
}

func expandAtConstants(entries []*paradox.Entry) {
	registry := map[string]paradox.Scalar{}
	for _, e := range entries {
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
	var walk func([]*paradox.Entry)
	walk = func(es []*paradox.Entry) {
		for _, e := range es {
			if e.Block != nil {
				walk(e.Block.Entries)
				continue
			}
			if e.Scalar.Kind == paradox.ScalarStr && strings.HasPrefix(e.Scalar.Str, "@") {
				if v, ok := registry[e.Scalar.Str]; ok {
					e.Scalar = v
				}
			}
		}
	}
	walk(entries)
}

func strScalar(s string) paradox.Scalar {
	return paradox.Scalar{Kind: paradox.ScalarStr, Str: s}
}

func intScalar(n int64) paradox.Scalar {
	return paradox.Scalar{Kind: paradox.ScalarInt, Int: big.NewInt(n)}
}

func pyStr(s paradox.Scalar) string { return importsvc.PyScalarStr(s) }

func sv(s paradox.Scalar) any {
	switch s.Kind {
	case paradox.ScalarInt:
		return s.Int
	case paradox.ScalarFloat:
		return s.Float
	default:
		return s.Str
	}
}

func truthy(s paradox.Scalar) bool {
	switch s.Kind {
	case paradox.ScalarInt:
		return s.Int.Sign() != 0
	case paradox.ScalarFloat:
		return s.Float != 0
	default:
		return s.Str != ""
	}
}

func yesEq(s paradox.Scalar) bool {
	return pyLower(pyStrip(pyStr(s))) == "yes"
}

func pyInt(s paradox.Scalar) (*big.Int, error) {
	switch s.Kind {
	case paradox.ScalarInt:
		return s.Int, nil
	case paradox.ScalarFloat:
		if math.IsInf(s.Float, 0) {
			return nil, &PyError{"OverflowError", "cannot convert float infinity to integer"}
		}
		if math.IsNaN(s.Float) {
			return nil, &PyError{"ValueError", "cannot convert float NaN to integer"}
		}
		sc := s
		return importsvc.CoerceInt(&sc, 0), nil
	default:
		sc := s
		z0 := importsvc.CoerceInt(&sc, 0)
		z1 := importsvc.CoerceInt(&sc, 1)
		if z0.Cmp(z1) != 0 {
			return nil, &PyError{
				"ValueError",
				"invalid literal for int() with base 10: " + pyReprStr(s.Str),
			}
		}
		return z0, nil
	}
}

func pyFloatOfStr(raw string) (float64, bool) {
	sc := strScalar(raw)
	f0, _ := importsvc.CoerceFloat(&sc, 0)
	f1, _ := importsvc.CoerceFloat(&sc, 1)
	if f0 == f1 {
		return f0, true
	}
	if math.IsNaN(f0) && math.IsNaN(f1) {
		return f0, true
	}
	return 0, false
}

func accumulateRawExtra(m map[string]any, key string, piece any) {
	existing, present := m[key]
	if !present || existing == nil || existing == "" {
		m[key] = piece
		return
	}
	if lst, ok := existing.([]any); ok {
		m[key] = append(lst, piece)
		return
	}
	m[key] = []any{existing, piece}
}
