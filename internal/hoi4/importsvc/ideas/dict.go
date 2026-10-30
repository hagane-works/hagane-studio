package ideas

import (
	"errors"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

var ErrOverflow = errors.New("OverflowError")

type Dict struct {
	keys   []string
	values map[string]any
}

func NewDict() *Dict { return &Dict{values: map[string]any{}} }

func (d *Dict) Set(k string, v any) {
	if _, ok := d.values[k]; !ok {
		d.keys = append(d.keys, k)
	}
	d.values[k] = v
}

func (d *Dict) Get(k string) (any, bool) {
	v, ok := d.values[k]
	return v, ok
}

func (d *Dict) Keys() []string { return d.keys }

func (d *Dict) Len() int { return len(d.keys) }

func (d *Dict) Copy() *Dict {
	out := NewDict()
	for _, k := range d.keys {
		out.Set(k, d.values[k])
	}
	return out
}

func isEmptyStrAny(v any) bool {
	switch t := v.(type) {
	case string:
		return t == ""
	case paradox.Scalar:
		return t.Kind == paradox.ScalarStr && t.Str == ""
	}
	return false
}

func accumulateRawExtra(rawExtras *Dict, key string, piece any) {
	existing, present := rawExtras.Get(key)
	if !present || existing == nil || isEmptyStrAny(existing) {
		rawExtras.Set(key, piece)
	} else if lst, ok := existing.([]any); ok {
		rawExtras.Set(key, append(lst, piece))
	} else {
		rawExtras.Set(key, []any{existing, piece})
	}
}

func keyStr(e *paradox.Entry) string {
	if e.Key == nil {
		return "None"
	}
	return *e.Key
}

func keyOrEmpty(e *paradox.Entry) string {
	if e.Key == nil {
		return ""
	}
	return *e.Key
}

func hasKey(e *paradox.Entry) bool { return e.Key != nil && *e.Key != "" }

func valueStr(e *paradox.Entry) string {
	if e.Block != nil {
		return pyBlockRepr(e.Block)
	}
	return importsvc.PyScalarStr(e.Scalar)
}

func ppBlock(entries []*paradox.Entry, key string) ([]*paradox.Entry, bool) {
	for _, e := range entries {
		if e.Key != nil && *e.Key == key {
			if e.Block == nil {
				return nil, false
			}
			return e.Block.Entries, true
		}
	}
	return nil, false
}

func ppScalar(entries []*paradox.Entry, key string) paradox.Scalar {
	for _, e := range entries {
		if e.Key != nil && *e.Key == key {
			if e.Block != nil {
				return paradox.Scalar{Kind: paradox.ScalarStr, Str: ""}
			}
			return e.Scalar
		}
	}
	return paradox.Scalar{Kind: paradox.ScalarStr, Str: ""}
}

func ppBareValues(entries []*paradox.Entry) []any {
	out := []any{}
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

func blockText(b *paradox.Block) string { return importsvc.BlockToText(b, 0) }
