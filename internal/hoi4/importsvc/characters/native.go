package characters

import (
	"math/big"

	"haganeworks/internal/hoi4/paradox"
)

func NativeEntities(rows []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, nativeMap(row))
	}
	return out
}

func nativeMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = nativeValue(v)
	}
	return out
}

func nativeValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return nativeMap(x)
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = nativeValue(x[i])
		}
		return out
	case []string:
		return x
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
	default:
		return v
	}
}
