package importjob

import (
	"haganeworks/internal/project"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type MiscCounts map[string]int64
type MiscMergeOptions struct{ ConflictPolicy string }

type miscRawRow struct{ kind, path, content string }

func miscRawKey(row miscRawRow) string { return row.path }

func miscFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		n := strings.ToLower(filepath.ToSlash(rel))
		if filepath.Ext(n) != ".txt" {
			return nil
		}
		if miscKind(n) != "" {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

func miscKind(n string) string {
	for _, x := range []struct{ prefix, kind string }{

		{"common/raids/", "raids"}, {"common/on_actions/", "on_actions"},
	} {
		if strings.HasPrefix(n, x.prefix) {
			return x.kind
		}
	}
	return ""
}

func (s *session) stageMiscEntities(progress func(EntityProgress)) (MiscCounts, error) {
	files, err := miscFiles(s.root)
	if err != nil {
		return nil, err
	}
	counts := MiscCounts{}
	var total int64
	err = inFileOrder(s.ctx, len(files), func(i int) (miscRawRow, error) {
		rel, _ := filepath.Rel(s.root, files[i])
		rel = filepath.ToSlash(rel)
		body, err := os.ReadFile(files[i])
		if err != nil {
			return miscRawRow{}, err
		}
		return miscRawRow{kind: miscKind(strings.ToLower(rel)), path: rel, content: decodeScriptBytes(body)}, nil
	}, func(i int, row miscRawRow) error {
		s.miscRaw = append(s.miscRaw, row)
		counts[row.kind]++
		total++
		if progress != nil {
			progress(EntityProgress{FilesDone: i + 1, FilesTotal: len(files), Rows: total})
		}
		return nil
	})
	if err != nil {
		return counts, err
	}

	s.miscRaw = lastWins(s.miscRaw, miscRawKey)
	return counts, nil
}

func (s *session) mergeMiscEntities(tx *project.Tx, options MiscMergeOptions) error {
	when := miscImportedRawFile
	if strings.EqualFold(options.ConflictPolicy, "skip") {
		when = nil
	}
	raw := newKeyedTable(tx, "project_raw_files", "dest_path")
	for _, r := range s.miscRaw {
		miscUpsertRawFile(raw, project.Row{
			"dest_path": r.path, "content": r.content, "generate_replace_path": false,
			"note": "preserved by Go whole-mod import (" + r.kind + ")", "is_binary": false,
			"content_binary": nil, "source": "imported",
		}, []string{"content", "note", "is_binary", "content_binary", "source"}, when)
	}
	return nil
}

func miscImportedRawFile(row project.Row) bool {
	source, _ := row["source"].(string)
	return source == "imported"
}

func miscUpsertRawFile(raw *keyedTable, row project.Row, update []string, when func(project.Row) bool) {
	id, ok := raw.find(row["dest_path"])
	if !ok {
		raw.insertNew(row)
		return
	}
	if when == nil {
		return
	}
	if existing := raw.tx.Find(raw.table, id); existing == nil || !when(existing) {
		return
	}
	raw.tx.Update(raw.table, id, func(existing project.Row) {
		for _, column := range update {
			existing[column] = row[column]
		}
	})
}
