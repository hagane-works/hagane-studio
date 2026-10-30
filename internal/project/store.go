package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const deletedDir = ".deleted"

type Store struct {
	root string

	mu   sync.Mutex
	open map[int64]*Project
	dirs map[int64]string
}

func NewStore(root string) *Store {
	return &Store{root: root, open: map[int64]*Project{}, dirs: map[int64]string{}}
}

func (s *Store) Root() string { return s.root }

func (s *Store) List() ([]Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scanLocked()
}

func (s *Store) scanLocked() ([]Row, error) {
	entries, err := os.ReadDir(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return []Row{}, nil
	}
	if err != nil {
		return nil, err
	}
	type found struct {
		dir  string
		meta meta
		mod  time.Time
	}
	var all []found
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		dir := filepath.Join(s.root, entry.Name())
		m, err := readMeta(dir)
		if err != nil {
			continue
		}
		var modified time.Time
		if info, err := os.Stat(filepath.Join(dir, metaFile)); err == nil {
			modified = info.ModTime()
		}
		all = append(all, found{dir: dir, meta: m, mod: modified})
	}

	sort.Slice(all, func(i, j int) bool { return all[i].dir < all[j].dir })
	var maxID int64
	for _, f := range all {
		if id := toInt64(f.meta.Project["id"]); id > maxID {
			maxID = id
		}
	}
	seen := map[int64]bool{}
	s.dirs = map[int64]string{}
	rows := make([]Row, 0, len(all))
	for i := range all {
		f := &all[i]
		id := toInt64(f.meta.Project["id"])
		if id <= 0 || seen[id] {
			maxID++
			id = maxID
			f.meta.Project["id"] = id
			if err := writeJSON(filepath.Join(f.dir, metaFile), f.meta); err != nil {
				return nil, err
			}
		}
		seen[id] = true
		s.dirs[id] = f.dir
		if p, ok := s.open[id]; ok && p.dir != f.dir {
			delete(s.open, id)
		}
		row := cloneRow(f.meta.Project)
		row["folder"] = filepath.Base(f.dir)
		counts := map[string]any{}
		for table, n := range f.meta.Counts {
			counts[table] = n
		}
		row["_table_counts"] = counts
		if _, ok := row["updated_at"]; !ok {
			row["updated_at"] = f.mod.UTC().Format(time.RFC3339)
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return fmt.Sprint(rows[i]["updated_at"]) > fmt.Sprint(rows[j]["updated_at"])
	})
	return rows, nil
}

func readMeta(dir string) (meta, error) {
	raw, err := os.ReadFile(filepath.Join(dir, metaFile))
	if err != nil {
		return meta{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var m meta
	if err := decoder.Decode(&m); err != nil {
		return meta{}, err
	}
	if m.Project == nil {
		return meta{}, errors.New("project.json has no project record")
	}
	if m.Format > formatV1 {
		return meta{}, fmt.Errorf("project.json format %d is newer than this program", m.Format)
	}
	return m, nil
}

func (s *Store) Get(id int64) (*Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.open[id]; ok {
		return p, nil
	}
	dir, ok := s.dirs[id]
	if !ok {
		if _, err := s.scanLocked(); err != nil {
			return nil, err
		}
		if dir, ok = s.dirs[id]; !ok {
			return nil, ErrNotFound
		}
	}
	m, err := readMeta(dir)
	if err != nil {
		return nil, err
	}
	p := &Project{dir: dir, meta: m, tables: map[string][]Row{}}
	p.id = toInt64(m.Project["id"])
	p.uuid, _ = m.Project["uuid"].(string)
	s.open[id] = p
	return p, nil
}

func (s *Store) Create(record Row, folderHint string) (*Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.scanLocked(); err != nil {
		return nil, err
	}
	var maxID int64
	for id := range s.dirs {
		if id > maxID {
			maxID = id
		}
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return nil, err
	}
	dir, err := reserveFolder(s.root, FolderName(folderHint))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	stored := cloneRow(record)
	stored["id"] = maxID + 1
	if _, ok := stored["created_at"]; !ok {
		stored["created_at"] = now
	}
	stored["updated_at"] = now
	p := &Project{
		dir:    dir,
		meta:   meta{Format: formatV1, Project: stored, NextIDs: map[string]int64{}},
		tables: map[string][]Row{},
	}
	p.id = toInt64(stored["id"])
	p.uuid, _ = stored["uuid"].(string)
	if err := os.MkdirAll(filepath.Join(dir, dataDir), 0o755); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	if err := p.saveMetaLocked(); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	id := maxID + 1
	s.dirs[id] = dir
	s.open[id] = p
	return p, nil
}

func (s *Store) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, ok := s.dirs[id]
	if !ok {
		return ErrNotFound
	}
	trash := filepath.Join(s.root, deletedDir)
	if err := os.MkdirAll(trash, 0o755); err != nil {
		return err
	}
	target := filepath.Join(trash, filepath.Base(dir)+"-"+time.Now().Format("20060102-150405"))
	if err := os.Rename(dir, target); err != nil {
		return err
	}
	delete(s.dirs, id)
	delete(s.open, id)
	return nil
}

func FolderName(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ' ' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	folder := strings.Trim(b.String(), " .")
	if folder == "" {
		folder = "project"
	}
	if runes := []rune(folder); len(runes) > 80 {
		folder = strings.TrimRight(string(runes[:80]), " .")
	}
	return folder
}

func reserveFolder(root, name string) (string, error) {
	for n := 1; n < 1000; n++ {
		candidate := name
		if n > 1 {
			candidate = fmt.Sprintf("%s-%d", name, n)
		}
		dir := filepath.Join(root, candidate)
		err := os.Mkdir(dir, 0o755)
		if err == nil {
			return dir, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("no free folder name for %q", name)
}
