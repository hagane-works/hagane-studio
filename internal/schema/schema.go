package schema

import (
	_ "embed"
	"encoding/json"
	"sort"
	"time"
)

//go:embed tables.json
var tablesJSON []byte

type Column struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	NotNull bool   `json:"not_null"`
	Default any    `json:"default"`
}

var tables = func() map[string][]Column {
	var t map[string][]Column
	if err := json.Unmarshal(tablesJSON, &t); err != nil {
		panic("schema: tables.json: " + err.Error())
	}
	return t
}()

func Columns(table string) []Column { return tables[table] }

func Known(table string) bool { _, ok := tables[table]; return ok }

func Fill(table string, row map[string]any, now time.Time) {
	for _, column := range tables[table] {
		if _, ok := row[column.Name]; ok {
			continue
		}
		switch {
		case column.Default == "now":
			row[column.Name] = Timestamp(column.Type, now)
		case column.Default != nil:
			row[column.Name] = cloneJSON(column.Default)
		default:
			row[column.Name] = nil
		}
	}
}

func Timestamp(columnType string, t time.Time) string {
	t = t.UTC()
	switch columnType {
	case "timestamp with time zone", "timestamptz":
		return t.Format("2006-01-02T15:04:05.999999") + "+00:00"
	case "date":
		return t.Format("2006-01-02")
	default:
		return t.Format("2006-01-02T15:04:05.999999")
	}
}

func Now(table, column string, t time.Time) string {
	for _, c := range tables[table] {
		if c.Name == column {
			return Timestamp(c.Type, t)
		}
	}
	return Timestamp("timestamp without time zone", t)
}

func cloneJSON(v any) any {
	switch value := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, item := range value {
			out[k] = cloneJSON(item)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = cloneJSON(item)
		}
		return out
	default:
		return value
	}
}

func Tables() []string {
	names := make([]string, 0, len(tables))
	for name := range tables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
