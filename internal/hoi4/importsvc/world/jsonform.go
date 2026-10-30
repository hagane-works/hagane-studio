package world

import (
	"fmt"
	"math/big"

	"haganeworks/internal/hoi4/importsvc"
)

func EncodeValueJSON(x any) any {
	switch t := x.(type) {
	case nil:
		return map[string]any{"t": "none"}
	case bool:
		return map[string]any{"t": "bool", "v": t}
	case *big.Int:
		return map[string]any{"t": "int", "v": t.String()}
	case float64:
		return map[string]any{"t": "float", "v": importsvc.PyFloatReprStr(t)}
	case string:
		return map[string]any{"t": "str", "v": t}
	case []any:
		arr := []any{}
		for _, i := range t {
			arr = append(arr, EncodeValueJSON(i))
		}
		return map[string]any{"t": "list", "v": arr}
	case *PyDict:
		m := map[string]any{}
		for _, k := range t.Keys() {
			v, _ := t.Get(k)
			m[k] = EncodeValueJSON(v)
		}
		return map[string]any{"t": "dict", "v": m}
	}
	panic(fmt.Sprintf("world jsonform: unencodable %T", x))
}

func encJSONSummary(s *importsvc.ImportSummary) map[string]any {
	pres := map[string]any{}
	for _, k := range s.PreservedFields.Keys {
		pres[k] = s.PreservedFields.Get(k)
	}
	warns := s.Warnings
	if warns == nil {
		warns = []string{}
	}
	return map[string]any{"object_type": s.ObjectType, "total": s.Total,
		"preserved": pres, "warnings": warns}
}

func ImportRecJSON(rows []*PyDict, summary *importsvc.ImportSummary, err error) map[string]any {
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	encRows := []any{}
	for _, r := range rows {
		encRows = append(encRows, EncodeValueJSON(r))
	}
	return map[string]any{"rows": encRows, "summary": encJSONSummary(summary)}
}

func HistoryJSON(text, filename string) map[string]any {
	return ImportRecJSON(ImportHistoryFiles(text, filename))
}

func StatesJSON(text, filename string) map[string]any {
	rows, summary := ImportStateFiles(text, filename)
	return ImportRecJSON(rows, summary, nil)
}

func BookmarksJSON(text string) map[string]any {
	rows, summary := ImportBookmarks(text)
	return ImportRecJSON(rows, summary, nil)
}

func SuperEventsJSON(text string) map[string]any {
	return ImportRecJSON(ImportSuperEvents(text))
}
