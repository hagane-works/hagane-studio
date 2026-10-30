package ideas

import (
	"math/big"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func Native(v any) any {
	switch x := v.(type) {
	case *Dict:
		out := make(map[string]any, x.Len())
		for _, key := range x.Keys() {
			value, _ := x.Get(key)
			out[key] = Native(value)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = Native(x[i])
		}
		return out
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
	case importsvc.ScopeVerdict:
		kind, tag, shared := x.StorageTuple()
		return map[string]any{"kind": kind, "country_tag": tag, "shared_country_tags": shared}
	default:
		return v
	}
}

func NativeEntities(rows []*Dict) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, Native(row).(map[string]any))
	}
	return out
}
