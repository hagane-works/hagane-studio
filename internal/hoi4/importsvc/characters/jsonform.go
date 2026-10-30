package characters

import (
	"fmt"
	"math/big"

	"haganeworks/internal/hoi4/importsvc"
)

func encJSONValue(x any) any {
	switch t := x.(type) {
	case nil:
		return nil
	case bool:
		return map[string]any{"t": "bool", "v": t}
	case string:
		return map[string]any{"t": "str", "v": t}
	case *big.Int:
		return map[string]any{"t": "int", "v": t.String()}
	case float64:
		return map[string]any{"t": "float", "v": importsvc.PyFloatReprStr(t)}
	case []string:
		out := make([]any, 0, len(t))
		for _, v := range t {
			out = append(out, encJSONValue(v))
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, v := range t {
			out = append(out, encJSONValue(v))
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, v := range t {
			out[k] = encJSONValue(v)
		}
		return out
	default:
		panic(fmt.Sprintf("characters jsonform: unencodable value type %T", x))
	}
}

func runImportJSON(fn func(string) ([]map[string]any, *importsvc.ImportSummary, error), text string) map[string]any {
	entities, summary, err := fn(text)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	encEnts := make([]any, 0, len(entities))
	for _, e := range entities {
		encEnts = append(encEnts, encJSONValue(e))
	}
	preserved := map[string]any{}
	for _, k := range summary.PreservedFields.Keys {
		preserved[k] = summary.PreservedFields.Get(k)
	}
	warnings := summary.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return map[string]any{
		"entities": encEnts,
		"summary": map[string]any{
			"object_type": summary.ObjectType,
			"total":       summary.Total,
			"preserved":   preserved,
			"warnings":    warnings,
		},
	}
}

func CharactersJSON(text string) map[string]any {
	return runImportJSON(ImportCharacters, text)
}

func CountryLeaderTraitsJSON(text string) map[string]any {
	return runImportJSON(ImportCountryLeaderTraits, text)
}

func UnitLeaderTraitsJSON(text string) map[string]any {
	return runImportJSON(ImportUnitLeaderTraits, text)
}

func ScientistTraitsJSON(text string) map[string]any {
	return runImportJSON(ImportScientistTraits, text)
}

func ReplacementLeadersJSON(text string) map[string]any {
	return runImportJSON(func(t string) ([]map[string]any, *importsvc.ImportSummary, error) {
		results, summary := ImportReplacementCountryLeadersFromEvents(t)
		return results, summary, nil
	}, text)
}
