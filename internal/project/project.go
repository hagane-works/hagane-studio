package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"haganeworks/internal/schema"
)

type Row = map[string]any

const (
	metaFile  = "project.json"
	dataDir   = "data"
	FilesDir  = "files"
	formatV1  = 1
	tableMode = 0o644
)

var ErrNotFound = errors.New("not found")

type meta struct {
	Format int `json:"format"`

	Project Row `json:"project"`

	NextIDs map[string]int64 `json:"next_ids"`

	Counts map[string]int64 `json:"counts,omitempty"`

	BaseMods map[string]string `json:"base_mods,omitempty"`
}

type Project struct {
	dir string

	id   int64
	uuid string

	mu     sync.RWMutex
	meta   meta
	tables map[string][]Row
}

func (p *Project) ID() int64 { return p.id }

func (p *Project) Dir() string { return p.dir }

func (p *Project) FilesDir() string { return filepath.Join(p.dir, FilesDir) }

func (p *Project) Counts() map[string]int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make(map[string]int64, len(p.meta.Counts))
	for table, n := range p.meta.Counts {
		out[table] = n
	}
	return out
}

func (p *Project) Record() Row {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return cloneRow(p.meta.Project)
}

func (p *Project) BaseMods() map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return maps.Clone(p.meta.BaseMods)
}

func (p *Project) UpdateRecord(change func(Row)) (Row, error) {
	return p.UpdateRecordAndBaseMods(func(record Row, _ map[string]string) { change(record) })
}

func (p *Project) UpdateRecordAndBaseMods(change func(record Row, baseMods map[string]string)) (Row, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	next := cloneRow(p.meta.Project)
	baseMods := maps.Clone(p.meta.BaseMods)
	if baseMods == nil {
		baseMods = map[string]string{}
	}
	change(next, baseMods)
	next["id"] = p.meta.Project["id"]
	previous, previousBaseMods := p.meta.Project, p.meta.BaseMods
	p.meta.Project, p.meta.BaseMods = next, baseMods
	if err := p.saveMetaLocked(); err != nil {
		p.meta.Project, p.meta.BaseMods = previous, previousBaseMods
		return nil, err
	}
	return cloneRow(next), nil
}

func (p *Project) Rows(table string) ([]Row, error) {
	if err := p.load(table); err != nil {
		return nil, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	rows := p.tables[table]
	out := make([]Row, len(rows))
	for i, row := range rows {
		out[i] = cloneRow(row)
	}
	return out, nil
}

type Tx struct {
	p     *Project
	dirty map[string]bool
	ids   bool
	index map[string]map[int64]int
}

func (p *Project) Write(tables []string, change func(*Tx) error) error {
	for _, table := range tables {
		if err := p.load(table); err != nil {
			return err
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	backup := make(map[string][]Row, len(tables))
	for _, table := range tables {
		backup[table] = p.tables[table]
	}
	backupIDs := make(map[string]int64, len(p.meta.NextIDs))
	for k, v := range p.meta.NextIDs {
		backupIDs[k] = v
	}
	backupRecord := p.meta.Project
	tx := &Tx{p: p, dirty: map[string]bool{}, index: map[string]map[int64]int{}}
	restore := func() {
		for table, rows := range backup {
			p.tables[table] = rows
		}
		p.meta.NextIDs = backupIDs
		p.meta.Project = backupRecord
	}
	if err := change(tx); err != nil {
		restore()
		return err
	}
	for table := range tx.dirty {
		if err := p.saveTableLocked(table); err != nil {
			restore()
			return err
		}
	}
	if len(tx.dirty) > 0 {
		if p.meta.Counts == nil {
			p.meta.Counts = map[string]int64{}
		}
		for table := range tx.dirty {
			if n := int64(len(p.tables[table])); n != p.meta.Counts[table] {
				p.meta.Counts[table] = n
				tx.ids = true
			}
		}
	}
	if tx.ids {
		if err := p.saveMetaLocked(); err != nil {
			return err
		}
	}
	return nil
}

func (tx *Tx) table(name string) []Row {
	rows, ok := tx.p.tables[name]
	if !ok {
		panic(fmt.Sprintf("project: table %q was not declared in Write", name))
	}
	return rows
}

func (tx *Tx) own(table string) []Row {
	rows := tx.table(table)
	if !tx.dirty[table] {
		rows = append(make([]Row, 0, len(rows)+16), rows...)
		tx.p.tables[table] = rows
		tx.dirty[table] = true
	}
	return rows
}

func (tx *Tx) position(table string, id int64) int {
	index, ok := tx.index[table]
	if !ok {
		rows := tx.table(table)
		index = make(map[int64]int, len(rows))
		for i, row := range rows {
			index[toInt64(row["id"])] = i
		}
		tx.index[table] = index
	}
	if i, ok := index[id]; ok {
		return i
	}
	return -1
}

func (tx *Tx) Rows(table string) []Row { return tx.table(table) }

func (tx *Tx) Find(table string, id int64) Row {
	if i := tx.position(table, id); i >= 0 {
		return tx.table(table)[i]
	}
	return nil
}

func (tx *Tx) NextID(table string) int64 {
	meta := &tx.p.meta
	if meta.NextIDs == nil {
		meta.NextIDs = map[string]int64{}
	}
	next := meta.NextIDs[table]
	if next == 0 {

		next = 1
		for _, row := range tx.table(table) {
			if id := toInt64(row["id"]); id >= next {
				next = id + 1
			}
		}
	}
	meta.NextIDs[table] = next + 1
	tx.ids = true
	return next
}

func (tx *Tx) Insert(table string, row Row) Row {
	stored := cloneRow(row)
	if toInt64(stored["id"]) == 0 {
		stored["id"] = tx.NextID(table)
	}
	tx.stamp(table, stored, true)
	rows := tx.own(table)
	tx.position(table, 0)
	tx.p.tables[table] = append(rows, stored)
	tx.index[table][toInt64(stored["id"])] = len(rows)
	return stored
}

func (tx *Tx) Update(table string, id int64, change func(Row)) (Row, error) {
	i := tx.position(table, id)
	if i < 0 {
		return nil, ErrNotFound
	}
	rows := tx.own(table)
	next := cloneRow(rows[i])
	change(next)
	next["id"] = rows[i]["id"]
	tx.stamp(table, next, false)
	rows[i] = next
	return next, nil
}

func (tx *Tx) stamp(table string, row Row, insert bool) {
	columns := schema.Columns(table)
	if columns == nil {
		return
	}
	now := time.Now()
	for _, column := range columns {
		switch column.Name {
		case "project_id":
			if insert {
				row["project_id"] = tx.p.meta.Project["id"]
			}
		case "created_at":
			if insert && row["created_at"] == nil {
				row["created_at"] = schema.Timestamp(column.Type, now)
			}
		case "updated_at":
			row["updated_at"] = schema.Timestamp(column.Type, now)
		}
	}
	if insert {
		schema.Fill(table, row, now)
	}
}

func (tx *Tx) Delete(table string, match func(Row) bool) int {
	rows := tx.table(table)
	kept := make([]Row, 0, len(rows))
	for _, row := range rows {
		if !match(row) {
			kept = append(kept, row)
		}
	}
	removed := len(rows) - len(kept)
	if removed > 0 {
		tx.p.tables[table] = kept
		tx.dirty[table] = true
		delete(tx.index, table)
	}
	return removed
}

func (tx *Tx) Replace(table string, rows []Row) {
	tx.table(table)
	tx.p.tables[table] = rows
	tx.dirty[table] = true
	delete(tx.index, table)
}

func (tx *Tx) SetRecord(change func(Row)) {
	next := cloneRow(tx.p.meta.Project)
	change(next)
	next["id"] = tx.p.meta.Project["id"]
	tx.p.meta.Project = next
	tx.ids = true
}

func (tx *Tx) Record() Row { return tx.p.meta.Project }

func (p *Project) load(table string) error {
	if !validTableName(table) {
		return fmt.Errorf("project: invalid table name %q", table)
	}
	p.mu.RLock()
	_, loaded := p.tables[table]
	p.mu.RUnlock()
	if loaded {
		return nil
	}
	rows, err := readTable(filepath.Join(p.dir, dataDir, table+".json"))
	if err != nil {
		return fmt.Errorf("read %s: %w", table, err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, loaded := p.tables[table]; !loaded {
		p.tables[table] = rows
	}
	return nil
}

func readTable(path string) ([]Row, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Row{}, nil
	}
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var rows []Row
	if err := decoder.Decode(&rows); err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []Row{}
	}
	return rows, nil
}

func (p *Project) saveTableLocked(table string) error {
	rows := p.tables[table]
	sort.SliceStable(rows, func(i, j int) bool { return toInt64(rows[i]["id"]) < toInt64(rows[j]["id"]) })
	return writeJSON(filepath.Join(p.dir, dataDir, table+".json"), rows)
}

func (p *Project) saveMetaLocked() error {
	return writeJSON(filepath.Join(p.dir, metaFile), p.meta)
}

func writeJSON(path string, v any) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

func validTableName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}
	return true
}

func cloneRow(row Row) Row {
	if row == nil {
		return nil
	}
	return cloneValue(row).(Row)
}

func cloneValue(v any) any {
	switch value := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, item := range value {
			out[k] = cloneValue(item)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = cloneValue(item)
		}
		return out
	default:
		return value
	}
}

func toInt64(v any) int64 {
	switch value := v.(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	case json.Number:
		n, err := value.Int64()
		if err != nil {
			return 0
		}
		return n
	case string:
		var n int64
		if _, err := fmt.Sscan(strings.TrimSpace(value), &n); err != nil {
			return 0
		}
		return n
	}
	return 0
}

func Int64(v any) int64 { return toInt64(v) }
