package exportcheck

import "testing"

func TestAnyTruthyHandlesTypedNil(t *testing.T) {
	var nilPtr *int64
	var nilMap map[string]any
	var nilSlice []string
	for name, v := range map[string]any{
		"typed-nil-ptr":   nilPtr,
		"typed-nil-map":   nilMap,
		"typed-nil-slice": nilSlice,
		"untyped-nil":     nil,
		"empty-map":       map[string]any{},
		"empty-slice":     []any{},
		"empty-string":    "",
		"zero-int":        0,
		"false":           false,
	} {
		if anyTruthy(v) {
			t.Errorf("%s 应当为 false", name)
		}
	}
	id := int64(42)
	for name, v := range map[string]any{
		"non-nil-ptr": &id,
		"non-empty":   map[string]any{"a": 1},
		"string":      "x",
		"int":         1,
		"true":        true,
		"slice":       []any{1},
	} {
		if !anyTruthy(v) {
			t.Errorf("%s 应当为 true", name)
		}
	}
}
