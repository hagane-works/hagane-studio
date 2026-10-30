package trees

import (
	"math/big"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

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

func strScalar(s string) paradox.Scalar {
	return paradox.Scalar{Kind: paradox.ScalarStr, Str: s}
}

func intScalar(n int64) paradox.Scalar {
	return paradox.Scalar{Kind: paradox.ScalarInt, Int: big.NewInt(n)}
}

func floatScalar(f float64) paradox.Scalar {
	return paradox.Scalar{Kind: paradox.ScalarFloat, Float: f}
}

func scalarOr(entries []*paradox.Entry, key string, def paradox.Scalar) paradox.Scalar {
	e := findOne(entries, key)
	if e == nil || e.Block != nil {
		return def
	}
	return e.Scalar
}

func scalarOpt(entries []*paradox.Entry, key string) *paradox.Scalar {
	e := findOne(entries, key)
	if e == nil || e.Block != nil {
		return nil
	}
	sc := e.Scalar
	return &sc
}

func ppBlock(entries []*paradox.Entry, key string) *paradox.Block {
	e := findOne(entries, key)
	if e == nil || e.Block == nil {
		return nil
	}
	return e.Block
}

func truthyBlock(b *paradox.Block) bool {
	return b != nil && len(b.Entries) > 0
}

func bareValues(entries []*paradox.Entry) []string {
	out := []string{}
	for _, e := range entries {
		if e.Key == nil && e.Block == nil {
			out = append(out, pyStr(e.Scalar))
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
	var walk func(es []*paradox.Entry)
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

type RawDict struct {
	Keys []string
	m    map[string]any
}

func NewRawDict() *RawDict { return &RawDict{m: map[string]any{}} }

func (d *RawDict) Get(k string) (any, bool) {
	v, ok := d.m[k]
	return v, ok
}

func (d *RawDict) Set(k string, v any) {
	if _, ok := d.m[k]; !ok {
		d.Keys = append(d.Keys, k)
	}
	d.m[k] = v
}

func isEmptyStrVal(v any) bool {
	switch t := v.(type) {
	case string:
		return t == ""
	case paradox.Scalar:
		return t.Kind == paradox.ScalarStr && t.Str == ""
	}
	return false
}

func (d *RawDict) Accumulate(k string, piece any) {
	existing, present := d.m[k]
	if !present || existing == nil || isEmptyStrVal(existing) {
		d.Set(k, piece)
	} else if lst, ok := existing.([]any); ok {
		d.Set(k, append(lst, piece))
	} else {
		d.Set(k, []any{existing, piece})
	}
}
