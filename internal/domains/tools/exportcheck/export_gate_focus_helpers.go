package exportcheck

import (
	"haganeworks/internal/hoi4/clausewitz"
	"haganeworks/internal/hoi4/importsvc/world"
	"reflect"
)

func analysisFindingDict(f clausewitz.AnalysisFinding) map[string]any {
	evidence := f.Evidence
	if evidence == nil {
		evidence = map[string]any{}
	}
	return map[string]any{
		"code": f.Code, "severity": f.Severity, "source": f.Source,
		"evidence": evidence, "repair_route": f.RepairRoute,
	}
}

func contractsEqual(a, b map[string]any) bool { return reflect.DeepEqual(a, b) }

func pyExceptionName(kind string) string { return kind }

const pyErrFocusRegistry = "FocusTreeRegistryError"

const pyErrParse = "ParseError"

func anyToStringSlice(v any) []string {
	items, ok := v.([]any)
	if !ok {
		if direct, ok := v.([]string); ok {
			return direct
		}
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, anyToString(item))
	}
	return out
}

func anyToGroups(v any) [][]string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := [][]string{}
	for _, item := range items {
		if group, ok := item.([]any); ok {
			values := make([]string, 0, len(group))
			for _, value := range group {
				values = append(values, anyToString(value))
			}
			out = append(out, values)
			continue
		}
		out = append(out, []string{anyToString(item)})
	}
	return out
}

func anyToMapSlice(v any) []map[string]any {
	items, ok := v.([]any)
	if !ok {
		if direct, ok := v.([]map[string]any); ok {
			return direct
		}
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		switch value := item.(type) {
		case map[string]any:
			out = append(out, value)
		case *world.PyDict:

			out = append(out, pyDictToMap(value))
		}
	}
	return out
}

func pyDictToMap(d *world.PyDict) map[string]any {
	if d == nil {
		return nil
	}
	out := make(map[string]any, d.Len())
	for _, key := range d.Keys() {
		out[key] = pyDictValue(d.MustGet(key))
	}
	return out
}

func pyDictValue(v any) any {
	switch value := v.(type) {
	case *world.PyDict:
		return pyDictToMap(value)
	case []any:
		out := make([]any, 0, len(value))
		for _, item := range value {
			out = append(out, pyDictValue(item))
		}
		return out
	}
	return v
}

func anyTruthy(v any) bool {
	switch value := v.(type) {
	case nil:
		return false
	case bool:
		return value
	case string:
		return value != ""
	case int:
		return value != 0
	case int64:
		return value != 0
	case float64:
		return value != 0
	case map[string]any:
		return len(value) > 0
	case []any:
		return len(value) > 0
	case []string:
		return len(value) > 0
	case []map[string]any:
		return len(value) > 0
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return false
		}
		return anyTruthy(rv.Elem().Interface())
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() > 0
	}
	return true
}
