package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"haganeworks/internal/project"
	"haganeworks/internal/schema"
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

func intOr(row project.Row, column string, def int) int {
	if row[column] == nil {
		return def
	}
	return int(project.Int64(row[column]))
}

func boolOr(row project.Row, column string, def bool) bool {
	if b, ok := row[column].(bool); ok {
		return b
	}
	return def
}

func floatPtr(row project.Row, column string) *float64 {
	switch v := row[column].(type) {
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return &f
		}
	case float64:
		return &v
	case int64:
		f := float64(v)
		return &f
	case int:
		f := float64(v)
		return &f
	}
	return nil
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

func parseTimestamp(v any) (time.Time, bool) {
	raw, _ := v.(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func isoNaive(v any) string {
	t, ok := parseTimestamp(v)
	if !ok {
		s, _ := v.(string)
		return s
	}
	if t.Nanosecond() == 0 {
		return t.Format("2006-01-02T15:04:05")
	}
	return t.Format("2006-01-02T15:04:05") + fmt.Sprintf(".%06d", t.Nanosecond()/1000)
}

func isoTime(v any) string {
	t, ok := parseTimestamp(v)
	if !ok {
		s, _ := v.(string)
		return s
	}
	return t.Format(time.RFC3339Nano)
}

func anyToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return anyToString(f)
		}
		return t.String()
	case bool:
		if t {
			return "True"
		}
		return "False"
	}
	return ""
}

func pyRepr(s string) string {
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		return `"` + s + `"`
	}
	return "'" + s + "'"
}

func findRow(rows []project.Row, id int64) project.Row {
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			return row
		}
	}
	return nil
}

func sortRows(rows []project.Row, less func(a, b project.Row) bool) {
	sort.SliceStable(rows, func(i, j int) bool { return less(rows[i], rows[j]) })
}

func byText(column string) func(a, b project.Row) bool {
	return func(a, b project.Row) bool { return text(a, column) < text(b, column) }
}

func taken(rows []project.Row, skipID int64, match func(project.Row) bool) bool {
	for _, row := range rows {
		if project.Int64(row["id"]) != skipID && match(row) {
			return true
		}
	}
	return false
}

func checkNotNull(table string, row project.Row) error {
	for _, column := range schema.Columns(table) {
		if column.NotNull && row[column.Name] == nil {
			return fmt.Errorf("%w: %s.%s is null", errConstraint, table, column.Name)
		}
	}
	return nil
}

func insert(tx *project.Tx, table string, row project.Row) (project.Row, error) {
	stored := tx.Insert(table, row)
	if err := checkNotNull(table, stored); err != nil {
		return nil, err
	}
	return stored, nil
}

func update(tx *project.Tx, table string, id int64, sets project.Row) (project.Row, error) {
	notNull := map[string]bool{}
	for _, column := range schema.Columns(table) {
		notNull[column.Name] = column.NotNull
	}
	for column, value := range sets {
		if value == nil && notNull[column] {
			return nil, fmt.Errorf("%w: %s.%s is null", errConstraint, table, column)
		}
	}
	return tx.Update(table, id, func(row project.Row) {
		for column, value := range sets {
			row[column] = value
		}
	})
}

func optValue[T any](value *T) any {
	if value == nil {
		return nil
	}
	return stored(*value)
}

func optJSON[T any](value *T) any {
	if value == nil {
		return nil
	}
	return jsonValue(*value)
}

func stored(v any) any {
	switch value := v.(type) {
	case int:
		return int64(value)
	case float64, string, bool, int64:
		return value
	case nil:
		return nil
	}
	return jsonValue(v)
}

func stringMap(v any) map[string]string {
	var out map[string]string
	decode(v, &out)
	if out == nil {
		out = map[string]string{}
	}
	return out
}

func anyMap(v any) map[string]any {
	var out map[string]any
	decode(v, &out)
	if out == nil {
		out = map[string]any{}
	}
	return out
}
