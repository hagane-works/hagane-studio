package importjob

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"haganeworks/internal/project"
)

type session struct {
	ctx     context.Context
	root    string
	options Options

	record project.Row

	facts *gameFacts

	mu       sync.Mutex
	warnings []string

	locAll []locRow

	loc map[string]map[string]*locRow

	gfx map[string]string

	entities []entityRow

	neededImages map[string]bool

	trees      treeStage
	world      worldStage
	structured structuredStage
	miscRaw    []miscRawRow
	countries  countryStage
	cosmetic   cosmeticStage
	units      unitStage
	mapFiles   mapFileStage
}

func newSession(ctx context.Context, root string, options Options, record project.Row) *session {
	return &session{
		ctx: ctx, root: root, options: options, record: record,
		loc: map[string]map[string]*locRow{}, gfx: map[string]string{},
		neededImages: map[string]bool{},
	}
}

func (s *session) gameFacts() (*gameFacts, error) {
	if s.facts == nil {
		aliases, err := loadCountryAliases(s.ctx)
		if err != nil {
			return nil, err
		}
		focus, focusErr := loadFocusRegistry(s.ctx)
		if err := s.ctx.Err(); err != nil {
			return nil, err
		}
		s.facts = &gameFacts{aliases: aliases, focus: focus, focusErr: focusErr}
	}
	return s.facts, nil
}

func (s *session) warn(message string) {
	s.mu.Lock()
	s.warnings = append(s.warnings, message)
	s.mu.Unlock()
}

func (s *session) warnParse(path string, cause error) {
	s.warn(fmt.Sprintf("Parse failed %s: %v", path, cause))
}

func (s *session) sortedWarnings() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]string(nil), s.warnings...)
	sort.Strings(out)
	return out
}

func (s *session) needImage(path string) {
	if path != "" {
		s.neededImages[path] = true
	}
}

func (s *session) projectTag(tx *project.Tx) string {
	if s.options.CountryTag != "" {
		return s.options.CountryTag
	}
	tag, _ := tx.Record()["country_tag"].(string)
	return tag
}

func (s *session) projectTags() []string {
	seen := map[string]bool{}
	var out []string
	add := func(tag string) {
		tag = strings.ToUpper(strings.TrimSpace(tag))
		if tag != "" && !seen[tag] {
			seen[tag] = true
			out = append(out, tag)
		}
	}
	if list, ok := s.record["country_tags"].([]any); ok {
		for _, item := range list {
			if tag, ok := item.(string); ok {
				add(tag)
			}
		}
	}
	if tag, ok := s.record["country_tag"].(string); ok {
		add(tag)
	}
	return out
}

func locShort(locale string) string {
	switch locale {
	case "l_english":
		return "en"
	case "l_simp_chinese":
		return "zh"
	}
	if len(locale) > 2 {
		return locale[2:]
	}
	return ""
}

func (s *session) locNames(key string) map[string]any {
	byLocale := s.loc[key]
	if len(byLocale) == 0 {
		return nil
	}
	out := make(map[string]any, len(byLocale))
	for locale, row := range byLocale {
		out[locShort(locale)] = row.value
	}
	return out
}

func (s *session) locNamesOrEmpty(key string) map[string]any {
	if names := s.locNames(key); names != nil {
		return names
	}
	return map[string]any{}
}

func (s *session) locText(key, lang string) (string, bool) {
	for locale, row := range s.loc[key] {
		if locShort(locale) == lang {
			return row.value, true
		}
	}
	return "", false
}

func (s *session) locStr(key, lang string) string {
	text, _ := s.locText(key, lang)
	return text
}

type payload map[string]any

func decodePayload(raw []byte) payload {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out payload
	if err := decoder.Decode(&out); err != nil || out == nil {
		return payload{}
	}
	return out
}

func (p payload) text(key string) (string, bool) {
	value, ok := p[key]
	if !ok || value == nil {
		return "", false
	}
	return jsonText(value), true
}

func jsonText(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		raw, _ := json.Marshal(v)
		return string(raw)
	}
}

func (p payload) str(key string) string {
	text, _ := p.text(key)
	return text
}

func (p payload) strOr(key, fallback string) string {
	if text, ok := p.text(key); ok {
		return text
	}
	return fallback
}

func (p payload) json(key string) (any, bool) {
	value, ok := p[key]
	return value, ok
}

func (p payload) jsonOr(key string, fallback any) any {
	if value, ok := p[key]; ok {
		return value
	}
	return fallback
}

func (p payload) boolOr(key string, fallback bool) bool {
	text, ok := p.text(key)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "true", "t", "yes", "y", "on", "1":
		return true
	case "false", "f", "no", "n", "off", "0":
		return false
	}
	return fallback
}

func (p payload) intOr(key string, fallback int64) int64 {
	if n, ok := p.int(key); ok {
		return n
	}
	return fallback
}

func (p payload) int(key string) (int64, bool) {
	text, ok := p.text(key)
	if !ok || strings.TrimSpace(text) == "" {
		return 0, false
	}
	if n, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64); err == nil {
		return n, true
	}

	if f, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil {
		return int64(f + 0.5), true
	}
	return 0, false
}

func (p payload) float(key string) (float64, bool) {
	text, ok := p.text(key)
	if !ok || strings.TrimSpace(text) == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	return f, err == nil
}

func (p payload) nullableFloat(key string) any {
	if f, ok := p.float(key); ok {
		return f
	}
	return nil
}

func (p payload) nullableInt(key string) any {
	if n, ok := p.int(key); ok {
		return n
	}
	return nil
}

func (p payload) nullableText(key string) any {
	if text, ok := p.text(key); ok {
		return text
	}
	return nil
}

func lastWins[T any](rows []T, key func(T) string) []T {
	last := make(map[string]int, len(rows))
	for i, row := range rows {
		last[key(row)] = i
	}
	out := make([]T, 0, len(last))
	for i, row := range rows {
		if last[key(row)] == i {
			out = append(out, row)
		}
	}
	return out
}

type keyedTable struct {
	tx      *project.Tx
	table   string
	columns []string
	ids     map[string]int64
}

func newKeyedTable(tx *project.Tx, table string, columns ...string) *keyedTable {
	k := &keyedTable{tx: tx, table: table, columns: columns, ids: map[string]int64{}}
	for _, row := range tx.Rows(table) {
		k.ids[k.keyOf(row)] = project.Int64(row["id"])
	}
	return k
}

func keyPart(value any) string {
	switch v := value.(type) {
	case nil:
		return "\x01null"
	case string:
		return v
	case json.Number:
		return v.String()
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		raw, _ := json.Marshal(v)
		return string(raw)
	}
}

func (k *keyedTable) keyOf(row project.Row) string {
	parts := make([]string, len(k.columns))
	for i, column := range k.columns {
		parts[i] = keyPart(row[column])
	}
	return strings.Join(parts, "\x00")
}

func (k *keyedTable) find(values ...any) (int64, bool) {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = keyPart(value)
	}
	id, ok := k.ids[strings.Join(parts, "\x00")]
	return id, ok
}

func (k *keyedTable) exists(row project.Row) bool {
	_, ok := k.ids[k.keyOf(row)]
	return ok
}

func (k *keyedTable) upsert(row project.Row, update []string) (project.Row, bool) {
	if id, ok := k.ids[k.keyOf(row)]; ok {
		stored, err := k.tx.Update(k.table, id, func(existing project.Row) {
			for _, column := range update {
				existing[column] = row[column]
			}
		})
		if err == nil {
			return stored, false
		}
	}
	stored := k.tx.Insert(k.table, row)
	k.ids[k.keyOf(stored)] = project.Int64(stored["id"])
	return stored, true
}

func (k *keyedTable) insertNew(row project.Row) (project.Row, bool) {
	if id, ok := k.ids[k.keyOf(row)]; ok {
		return k.tx.Find(k.table, id), false
	}
	stored := k.tx.Insert(k.table, row)
	k.ids[k.keyOf(stored)] = project.Int64(stored["id"])
	return stored, true
}

func columnsOf(row project.Row, except ...string) []string {
	skip := map[string]bool{"id": true, "project_id": true, "created_at": true}
	for _, column := range except {
		skip[column] = true
	}
	out := make([]string, 0, len(row))
	for column := range row {
		if !skip[column] {
			out = append(out, column)
		}
	}
	sort.Strings(out)
	return out
}

func notPorted(s *session, what string) error {
	s.warn("Import of " + what + " is not available in this version yet.")
	return nil
}
