package specialprojects

import (
	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
	"math/big"
)

func NativeRawExtras(r *importsvc.RawExtras) map[string]any {
	out := map[string]any{}
	if r == nil {
		return out
	}
	for _, k := range r.Keys {
		if v, ok := r.Get(k); ok {
			out[k] = native(v)
		}
	}
	return out
}
func native(v any) any {
	switch x := v.(type) {
	case *big.Int:
		if x.IsInt64() {
			return x.Int64()
		}
		return x.String()
	case paradox.Scalar:
		switch x.Kind {
		case paradox.ScalarInt:
			if x.Int.IsInt64() {
				return x.Int.Int64()
			}
			return x.Int.String()
		case paradox.ScalarFloat:
			return x.Float
		default:
			return x.Str
		}
	case []any:
		o := make([]any, len(x))
		for i := range x {
			o[i] = native(x[i])
		}
		return o
	case map[string]any:
		o := map[string]any{}
		for k, z := range x {
			o[k] = native(z)
		}
		return o
	default:
		return v
	}
}
