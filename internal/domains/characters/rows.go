package characters

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

func detail(status int, msg string) error {
	return &apiutil.StatusError{Status: status, Body: map[string]any{"detail": msg}}
}

func pathID(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id, err == nil && id > 0
}

func decodeBody(w http.ResponseWriter, r *http.Request, limit int64, v any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.UseNumber()
	return decoder.Decode(v)
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func intOr(v any, def int) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case int:
		return n
	case float64:
		return int(n)
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
		if f, err := n.Float64(); err == nil {
			return int(f)
		}
	}
	return def
}

func boolOr(v any, def bool) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}

func boolPtr(v any) *bool {
	if b, ok := v.(bool); ok {
		return &b
	}
	return nil
}

func decodeColumn(v any, dst any) {
	if v == nil {
		return
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	_ = decoder.Decode(dst)
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

func stringList(v any) []string {
	switch list := v.(type) {
	case []string:
		return append([]string{}, list...)
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func notNullViolation(table string, changes map[string]any) bool {
	for _, column := range schema.Columns(table) {
		if !column.NotNull {
			continue
		}
		if value, ok := changes[column.Name]; ok && value == nil {
			return true
		}
	}
	return false
}

func firstNonBlank(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func strDefault(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

func intDefault(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

func boolDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func projectCountryTag(p *project.Project) string {
	return str(p.Record()["country_tag"])
}

func findRow(p *project.Project, table string, id int64) (project.Row, error) {
	rows, err := p.Rows(table)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			return row, nil
		}
	}
	return nil, nil
}

func flipImported(row project.Row) {
	if str(row["source"]) == "imported" {
		row["source"] = "created"
	}
}

func upperTrim(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
