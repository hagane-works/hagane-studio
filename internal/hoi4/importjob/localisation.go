package importjob

import (
	"bufio"
	"fmt"
	"haganeworks/internal/project"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	localeRE       = regexp.MustCompile(`^\s*(l_[A-Za-z0-9_]+)\s*:\s*$`)
	entryRE        = regexp.MustCompile(`^\s*([A-Za-z0-9_.-]+)\s*:\s*([0-9]*)\s*"((?:[^"\\]|\\.)*)"\s*(?:#.*)?$`)
	lenientEntryRE = regexp.MustCompile(`^\s*([A-Za-z0-9_.-]+)\s*:\s*([0-9]*)\s*"(.*)"\s*(?:#.*)?$`)
)

type LocProgress struct {
	FilesDone  int
	FilesTotal int
	Rows       int64
}

type locRow struct {
	locale          string
	key             string
	value           string
	version         *int32
	sourceFile      string
	replacePriority int32
	sourceOrder     int32
	entryOrder      int32
	isReplace       bool
}

func localisationFiles(root string) ([]string, error) {
	base := filepath.Join(root, "localisation")
	files := []string{}
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".yml") {
			files = append(files, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return files, nil
	}
	sort.Strings(files)
	return files, err
}

func parseLocFile(path, sourceFile string, sourceOrder int32) ([]locRow, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	replace := false
	for _, part := range strings.Split(strings.ToLower(filepath.ToSlash(sourceFile)), "/") {
		if part == "replace" {
			replace = true
			break
		}
	}
	priority := int32(0)
	if replace {
		priority = 1
	}
	rows := make([]locRow, 0, 256)
	locale := ""
	scanner := bufio.NewScanner(strings.NewReader(decodeScriptBytes(body)))
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	lineNumber := int32(0)
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimPrefix(scanner.Text(), "\ufeff")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if match := localeRE.FindStringSubmatch(line); match != nil {
			locale = match[1]
			continue
		}
		match := entryRE.FindStringSubmatch(line)
		if match == nil {
			match = lenientEntryRE.FindStringSubmatch(line)
		}
		if match == nil || locale == "" {
			continue
		}
		var version *int32
		if match[2] != "" {
			parsed, err := strconv.ParseInt(match[2], 10, 32)
			if err != nil {
				continue
			}
			value := int32(parsed)
			version = &value
		}
		rows = append(rows, locRow{
			locale: locale, key: match[1], value: decodeLocValue(match[3]),
			version: version, sourceFile: filepath.ToSlash(sourceFile),
			replacePriority: priority, sourceOrder: sourceOrder,
			entryOrder: lineNumber, isReplace: replace,
		})
	}
	return rows, scanner.Err()
}

func decodeLocValue(value string) string {
	var out strings.Builder
	for index := 0; index < len(value); index++ {
		if value[index] == '\\' && index+1 < len(value) {
			if decoded, ok := map[byte]byte{'n': '\n', 'r': '\r', 't': '\t', '"': '"', '\\': '\\'}[value[index+1]]; ok {
				out.WriteByte(decoded)
				index++
				continue
			}
		}
		out.WriteByte(value[index])
	}
	return out.String()
}

func (s *session) stageLocalisation(progress func(LocProgress)) (int64, error) {
	files, err := localisationFiles(s.root)
	if err != nil {
		return 0, err
	}
	var total int64
	err = inFileOrder(s.ctx, len(files), func(index int) ([]locRow, error) {
		rel, err := filepath.Rel(s.root, files[index])
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		rows, err := parseLocFile(files[index], rel, int32(index))
		if err != nil {
			return nil, fmt.Errorf("parse localisation %s: %w", rel, err)
		}
		return rows, nil
	}, func(index int, rows []locRow) error {
		s.locAll = append(s.locAll, rows...)
		total += int64(len(rows))
		if progress != nil {
			progress(LocProgress{FilesDone: index + 1, FilesTotal: len(files), Rows: total})
		}
		return nil
	})
	if err != nil {
		return total, err
	}
	for i := range s.locAll {
		row := &s.locAll[i]
		byLocale := s.loc[row.key]
		if byLocale == nil {
			byLocale = map[string]*locRow{}
			s.loc[row.key] = byLocale
		}
		if current := byLocale[row.locale]; current == nil || locRanksAbove(row, current) {
			byLocale[row.locale] = row
		}
	}
	return total, nil
}

func locRanksAbove(a, b *locRow) bool {
	if a.replacePriority != b.replacePriority {
		return a.replacePriority > b.replacePriority
	}
	if a.sourceOrder != b.sourceOrder {
		return a.sourceOrder > b.sourceOrder
	}
	return a.entryOrder > b.entryOrder
}

func (s *session) mergeLocalisation(tx *project.Tx) error {
	tx.Delete("project_loc_entries", func(row project.Row) bool { return row["source"] == "imported" })
	for _, row := range s.locAll {
		var version any
		if row.version != nil {
			version = *row.version
		}
		tx.Insert("project_loc_entries", project.Row{
			"locale": row.locale, "key": row.key, "value": row.value, "version": version,
			"source": "imported", "source_file": row.sourceFile,
			"replace_priority": row.replacePriority, "source_order": row.sourceOrder,
			"entry_order": row.entryOrder, "is_replace": row.isReplace,
		})
	}
	return nil
}
