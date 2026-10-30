package exportengine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/emit/locout"
	"haganeworks/internal/hoi4/exportmsg"
)

var placeholderIDTables = []struct{ table, column string }{
	{"ideas", "idea_id"},
	{"focus_nodes", "focus_id"},
	{"technologies", "tech_id"},
	{"characters", "tag"},
}

var placeholderIDPattern = buildPlaceholderIDPattern()

func buildPlaceholderIDPattern() *regexp.Regexp {
	stems := locout.PlaceholderDisplayNamesSorted()
	quoted := make([]string, 0, len(stems))
	for _, stem := range stems {
		token := strings.ReplaceAll(strings.TrimSpace(stem), " ", "_")
		if token == "" {
			continue
		}
		quoted = append(quoted, regexp.QuoteMeta(token))
	}
	if len(quoted) == 0 {

		return regexp.MustCompile(`\A\z(?:never)`)
	}
	return regexp.MustCompile(`(?i)\A(?:` + strings.Join(quoted, "|") + `)(?:_\d+)?\z`)
}

func IsPlaceholderDerivedID(id string) bool {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return false
	}
	return placeholderIDPattern.MatchString(trimmed)
}

type PlaceholderIDReport struct {
	IDs []string

	Tables map[string]int
	msg    exportmsg.Printer
}

func (r *PlaceholderIDReport) Count() int {
	if r == nil {
		return 0
	}
	return len(r.IDs)
}

func ScanPlaceholderIDs(snapshot Snapshot, msg exportmsg.Printer) *PlaceholderIDReport {
	report := &PlaceholderIDReport{Tables: map[string]int{}, msg: msg}
	seen := map[string]bool{}
	for _, spec := range placeholderIDTables {
		rows := snapshot.Tables[spec.table]
		hits := make([]string, 0, 4)
		for _, row := range rows {
			var probe map[string]json.RawMessage
			if err := json.Unmarshal(row, &probe); err != nil {
				continue
			}
			raw, ok := probe[spec.column]
			if !ok {
				continue
			}
			var id string
			if err := json.Unmarshal(raw, &id); err != nil {
				continue
			}
			id = strings.TrimSpace(id)
			if !IsPlaceholderDerivedID(id) || seen[spec.table+"\x00"+id] {
				continue
			}
			seen[spec.table+"\x00"+id] = true
			hits = append(hits, id)
		}
		if len(hits) == 0 {
			continue
		}
		sort.Strings(hits)
		report.Tables[spec.table] = len(hits)
		report.IDs = append(report.IDs, hits...)
	}
	return report
}

func (r *PlaceholderIDReport) Headline() string {
	if r.Count() == 0 {
		return ""
	}
	n := r.Count()
	tables := make([]string, 0, len(r.Tables))
	for table := range r.Tables {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	parts := make([]string, 0, len(tables))
	for _, table := range tables {
		parts = append(parts, fmt.Sprintf("%s: %d", table, r.Tables[table]))
	}
	return r.msg.Text("placeholder.headline", exportmsg.Args{
		"count": n, "ids": joinCapped(r.IDs, 8, r.msg), "tables": strings.Join(parts, ", "),
	})
}
