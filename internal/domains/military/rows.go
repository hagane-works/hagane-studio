package military

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"haganeworks/internal/project"
	"haganeworks/internal/schema"
	"haganeworks/internal/workbench"
)

func text(row project.Row, column string) string {
	s, _ := row[column].(string)
	return s
}

func textOr(row project.Row, column, def string) string {
	if s, ok := row[column].(string); ok {
		return s
	}
	return def
}

func fromRow(row project.Row, defaults map[string]any, dst any) {
	if len(defaults) > 0 {
		filled := make(project.Row, len(row))
		for k, v := range row {
			filled[k] = v
		}
		for column, value := range defaults {
			if filled[column] == nil {
				filled[column] = value
			}
		}
		row = filled
	}
	raw, err := json.Marshal(row)
	if err == nil {

		_ = json.Unmarshal(raw, dst)
	}
}

func decode(v any, dst any) {
	if v == nil {
		return
	}
	raw, err := json.Marshal(v)
	if err == nil {
		_ = json.Unmarshal(raw, dst)
	}
}

func jsonValue(v any) any {
	switch value := v.(type) {
	case nil:
		return nil
	case string, bool, json.Number:
		return value
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out any
	if err := decoder.Decode(&out); err != nil {
		return nil
	}
	return out
}

func opt[T any](o workbench.Optional[T]) any {
	if o.Value == nil {
		return nil
	}
	return jsonValue(*o.Value)
}

func optStr(o workbench.Optional[string]) string {
	if o.Value != nil {
		return *o.Value
	}
	return ""
}

func findRow(rows []project.Row, id int64) project.Row {
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			return row
		}
	}
	return nil
}

func nullViolation(table string, changes map[string]any) bool {
	for _, column := range schema.Columns(table) {
		if value, ok := changes[column.Name]; ok && value == nil && column.NotNull {
			return true
		}
	}
	return false
}

func number(v any) float64 {
	switch value := v.(type) {
	case json.Number:
		f, _ := value.Float64()
		return f
	case float64:
		return value
	case int64:
		return float64(value)
	case int:
		return float64(value)
	}
	return 0
}

func isoNaive(v any) string {
	raw, _ := v.(string)
	if raw == "" {
		return ""
	}
	for _, layout := range []string{"2006-01-02T15:04:05.999999999", "2006-01-02T15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999"} {
		t, err := time.Parse(layout, raw)
		if err != nil {
			continue
		}
		if t.Nanosecond() == 0 {
			return t.Format("2006-01-02T15:04:05")
		}
		return t.Format("2006-01-02T15:04:05") + fmt.Sprintf(".%06d", t.Nanosecond()/1000)
	}
	return raw
}

func upper(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func stringPtr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}
