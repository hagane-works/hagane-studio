package events

import (
	"errors"
	"fmt"
	"math/big"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func encJSONAny(x any) any {
	switch t := x.(type) {
	case nil:
		return nil
	case importsvc.ScopeVerdict:
		return map[string]any{"kind": string(t.Kind), "tags": t.TagList()}
	case bool:
		return t
	case *big.Int:
		return map[string]any{"t": "int", "v": t.String()}
	case int:
		return map[string]any{"t": "int", "v": fmt.Sprintf("%d", t)}
	case float64:
		return map[string]any{"t": "float", "v": importsvc.PyFloatReprStr(t)}
	case string:
		return t
	case []any:
		out := make([]any, 0, len(t))
		for _, v := range t {
			out = append(out, encJSONAny(v))
		}
		return out
	case []map[string]any:
		out := make([]any, 0, len(t))
		for _, v := range t {
			out = append(out, encJSONAny(v))
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, v := range t {
			out[k] = encJSONAny(v)
		}
		return out
	case *importsvc.RawExtras:
		out := map[string]any{}
		for _, k := range t.Keys {
			v, _ := t.Get(k)
			out[k] = encJSONAny(v)
		}
		return out
	default:
		panic(fmt.Sprintf("events jsonform: unexpected type %T", x))
	}
}

func encJSONSummary(s *importsvc.ImportSummary) map[string]any {
	preserved := map[string]any{}
	for _, k := range s.PreservedFields.Keys {
		preserved[k] = s.PreservedFields.Get(k)
	}
	warnings := []any{}
	for _, w := range s.Warnings {
		warnings = append(warnings, w)
	}
	return map[string]any{
		"object_type": s.ObjectType,
		"total":       s.Total,
		"preserved":   preserved,
		"warnings":    warnings,
	}
}

func sectionErrJSON(err error) (map[string]any, bool) {
	if err == nil {
		return nil, false
	}
	var pe *paradox.ParseError
	if errors.As(err, &pe) {
		return map[string]any{"parse_error": pe.Error()}, true
	}
	var pye *PyError
	if errors.As(err, &pye) {
		return map[string]any{"error": pye.Type}, true
	}
	panic(fmt.Sprintf("events jsonform: unexpected error type: %v", err))
}

func EventsJSON(text string) map[string]any {
	ns, evts, summary, err := ImportEvents(text)
	if rec, bad := sectionErrJSON(err); bad {
		return rec
	}
	nsAny := make([]any, 0, len(ns))
	for _, n := range ns {
		nsAny = append(nsAny, n)
	}
	return map[string]any{
		"namespaces": nsAny,
		"events":     encJSONAny(evts),
		"summary":    encJSONSummary(summary),
	}
}

func DecisionCategoriesJSON(text string) map[string]any {
	cats, summary, err := ImportDecisionCategories(text)
	if rec, bad := sectionErrJSON(err); bad {
		return rec
	}
	return map[string]any{
		"categories": encJSONAny(cats),
		"summary":    encJSONSummary(summary),
	}
}

func DecisionsJSON(text string) map[string]any {
	cats, decs, summary, err := ImportDecisions(text)
	if rec, bad := sectionErrJSON(err); bad {
		return rec
	}
	return map[string]any{
		"categories": encJSONAny(cats),
		"decisions":  encJSONAny(decs),
		"summary":    encJSONSummary(summary),
	}
}
