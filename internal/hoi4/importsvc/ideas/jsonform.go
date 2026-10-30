package ideas

import (
	"fmt"
	"math/big"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func encJSONScalar(s paradox.Scalar) map[string]any {
	switch s.Kind {
	case paradox.ScalarInt:
		return map[string]any{"t": "int", "v": s.Int.String()}
	case paradox.ScalarFloat:
		return map[string]any{"t": "float", "v": importsvc.PyFloatReprStr(s.Float)}
	default:
		return map[string]any{"t": "str", "v": s.Str}
	}
}

func encJSONAny(x any) any {
	switch t := x.(type) {
	case nil:
		return map[string]any{"t": "none"}
	case importsvc.ScopeVerdict:
		sk, tag, shared := t.StorageTuple()
		var sharedAny any
		if shared != nil {
			l := make([]any, 0, len(shared))
			for _, s := range shared {
				l = append(l, s)
			}
			sharedAny = l
		}
		var gateAny any
		if g, ok := importsvc.SynthCountryGate(sk, tag, shared); ok {
			gateAny = g
		}
		tags := []any{}
		for _, s := range t.TagList() {
			tags = append(tags, encJSONAny(s))
		}
		return map[string]any{
			"t": "verdict", "kind": string(t.Kind), "tags": tags,
			"storage": []any{encJSONAny(sk), encJSONAny(tag), encJSONAny(sharedAny)},
			"gate":    encJSONAny(gateAny),
		}
	case *Dict:
		pairs := []any{}
		for _, k := range t.Keys() {
			v, _ := t.Get(k)
			pairs = append(pairs, []any{k, encJSONAny(v)})
		}
		return map[string]any{"t": "dict", "v": pairs}
	case []any:
		l := []any{}
		for _, v := range t {
			l = append(l, encJSONAny(v))
		}
		return map[string]any{"t": "list", "v": l}
	case paradox.Scalar:
		return encJSONScalar(t)
	case *big.Int:
		return map[string]any{"t": "int", "v": t.String()}
	case float64:
		return map[string]any{"t": "float", "v": importsvc.PyFloatReprStr(t)}
	case bool:
		return map[string]any{"t": "bool", "v": t}
	case string:
		return map[string]any{"t": "str", "v": t}
	}
	panic(fmt.Sprintf("ideas jsonform encAny: unsupported value %T", x))
}

func summaryJSON(s *importsvc.ImportSummary) map[string]any {
	preserved := []any{}
	for _, k := range s.PreservedFields.Keys {
		preserved = append(preserved, []any{k, s.PreservedFields.Get(k)})
	}
	return map[string]any{
		"object_type": s.ObjectType, "total": s.Total,
		"preserved": preserved, "warnings": s.Warnings,
	}
}

func sectionJSON(ents []*Dict, summary *importsvc.ImportSummary, err error) map[string]any {
	if err != nil {
		return map[string]any{"error": "OverflowError"}
	}
	encEnts := []any{}
	for _, e := range ents {
		encEnts = append(encEnts, encJSONAny(e))
	}
	return map[string]any{"entities": encEnts, "summary": summaryJSON(summary)}
}

func IdeasJSON(text string) map[string]any { return sectionJSON(ImportIdeas(text)) }

func OpinionModifiersJSON(text string) map[string]any {
	return sectionJSON(ImportOpinionModifiers(text))
}

func DynamicModifiersJSON(text string) map[string]any {
	return sectionJSON(ImportDynamicModifiers(text))
}

func AutonomyStatesJSON(text string) map[string]any {
	return sectionJSON(ImportAutonomyStates(text))
}

func MIOJSON(text string) map[string]any { return sectionJSON(ImportMIO(text)) }

func IntelligenceAgenciesJSON(text string) map[string]any {
	return sectionJSON(ImportIntelligenceAgencies(text))
}

func DoctrinesJSON(text, entityType, sourceFile string) map[string]any {
	return sectionJSON(ImportDoctrines(text, entityType, sourceFile))
}

func ScriptedSnippetsJSON(text, kind, sourceFile string) map[string]any {
	return sectionJSON(ImportScriptedSnippets(text, kind, sourceFile))
}

func ScriptedLocalisationsJSON(text, sourceFile string) map[string]any {
	return sectionJSON(ImportScriptedLocalisations(text, sourceFile))
}
