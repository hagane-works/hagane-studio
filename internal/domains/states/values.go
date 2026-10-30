package states

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/schema"
)

func str(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return strconv.FormatInt(n, 10)
		}
		if f, err := t.Float64(); err == nil {
			return floatText(f)
		}
		return t.String()
	case float64:
		return floatText(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	case bool:
		if t {
			return "True"
		}
		return "False"
	}
	return ""
}

func floatText(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%v", f)
}

func num(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case int64:
		return float64(t), true
	case int:
		return float64(t), true
	}
	return 0, false
}

func whole(v any) (int64, bool) {
	switch t := v.(type) {
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return n, true
		}
	case int64:
		return t, true
	case int:
		return int64(t), true
	}
	f, ok := num(v)
	if !ok || f != float64(int64(f)) {
		return 0, false
	}
	return int64(f), true
}

func coerceInt(v any) int {
	switch t := v.(type) {
	case string:
		var n int
		if _, err := fmt.Sscanf(strings.TrimSpace(t), "%d", &n); err == nil {
			return n
		}
		return 0
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return int(n)
		}
	}
	f, _ := num(v)
	return int(f)
}

func plain(v any) any {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out any
	if decoder.Decode(&out) != nil {
		return nil
	}
	return out
}

func cloneRow(row map[string]any) map[string]any {
	out, _ := plain(row).(map[string]any)
	if out == nil {
		out = map[string]any{}
	}
	return out
}

func columnValue(table, column string, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	columnType := ""
	for _, c := range schema.Columns(table) {
		if c.Name == column {
			columnType = c.Type
			break
		}
	}
	invalid := apiutil.ValidationError(column + " has the wrong type")
	switch columnType {
	case "integer", "bigint", "smallint":
		n, ok := whole(v)
		if !ok {
			return nil, invalid
		}
		return n, nil
	case "double precision", "real", "numeric":
		f, ok := num(v)
		if !ok {
			return nil, invalid
		}
		return f, nil
	case "boolean":
		b, ok := v.(bool)
		if !ok {
			return nil, invalid
		}
		return b, nil
	case "character varying", "text":
		s, ok := v.(string)
		if !ok {
			return nil, invalid
		}
		return s, nil
	default:
		return plain(v), nil
	}
}

func detailError(status int, detail any) error {
	return &apiutil.StatusError{Status: status, Body: map[string]any{"detail": detail}}
}
