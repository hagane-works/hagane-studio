package exportengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	emitlocout "haganeworks/internal/hoi4/emit/locout"
	"haganeworks/internal/hoi4/exportmsg"
)

const bomUTF8 = "\uFEFF"

type preservedLocRow struct {
	ID          int64  `json:"id"`
	Locale      string `json:"locale"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Version     *int   `json:"version"`
	SourceFile  string `json:"source_file"`
	SourceOrder int    `json:"source_order"`
	EntryOrder  int    `json:"entry_order"`
	Source      string `json:"source"`
}

func preservedLocLine(key string, version *int, value string) string {
	versionText := ":"
	if version != nil {
		versionText = fmt.Sprintf(":%d", *version)
	}
	return " " + key + versionText + " \"" + emitlocout.YmlEscapeValue(value) + "\""
}

func safePreservedSourceFile(raw string) (string, bool) {
	s := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if s == "" || strings.HasPrefix(s, "/") || strings.Contains(s, ":") {
		return "", false
	}
	cleaned := path.Clean(s)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", false
	}
	lower := strings.ToLower(cleaned)
	if !strings.HasPrefix(lower, "localisation/") || !strings.HasSuffix(lower, ".yml") {
		return "", false
	}
	return cleaned, true
}

func SafePreservedLocSourceFile(raw string) (string, bool) {
	return safePreservedSourceFile(raw)
}

func eligiblePreservedLocKeys(rows []preservedLocRow, includeImported bool) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, row := range rows {
		if !includeImported && strings.EqualFold(row.Source, "imported") {
			continue
		}
		if row.Locale == "" || row.Key == "" || !emitlocout.IsLegalLocKey(row.Key) {
			continue
		}
		if _, ok := safePreservedSourceFile(row.SourceFile); !ok {
			continue
		}
		if out[row.Locale] == nil {
			out[row.Locale] = map[string]bool{}
		}
		out[row.Locale][row.Key] = true
	}
	return out
}

func ymlLocaleKeys(text string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	locale := ""
	for _, line := range strings.Split(strings.TrimPrefix(text, bomUTF8), "\n") {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, " ") && !strings.HasPrefix(trimmed, "\t") {
			if head := strings.TrimSuffix(strings.TrimSpace(trimmed), ":"); strings.HasPrefix(head, "l_") {
				locale = head
			}
			continue
		}
		if locale == "" {
			continue
		}
		body := strings.TrimSpace(trimmed)
		if body == "" || strings.HasPrefix(body, "#") {
			continue
		}
		idx := strings.IndexByte(body, ':')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(body[:idx])
		if key == "" {
			continue
		}
		if _, ok := out[locale]; !ok {
			out[locale] = map[string]bool{}
		}
		out[locale][key] = true
	}
	return out
}

func NativePreservedLoc(
	snapshot Snapshot, modDir string, includeImported bool, existing []Entry,
) ([]Entry, []exportmsg.Message, error) {
	rows, err := decodePreservedLocRows(snapshot.Tables["project_loc_entries"])
	if err != nil {
		return nil, nil, err
	}
	return nativePreservedLoc(rows, modDir, includeImported, existing)
}

func decodePreservedLocRows(raws []json.RawMessage) ([]preservedLocRow, error) {
	if len(raws) == 0 {
		return nil, nil
	}
	rows := make([]preservedLocRow, len(raws))
	for i, raw := range raws {
		if err := json.Unmarshal(raw, &rows[i]); err != nil {
			return nil, fmt.Errorf("preserved loc: decode row: %w", err)
		}
	}
	return rows, nil
}

func nativePreservedLoc(
	rows []preservedLocRow, modDir string, includeImported bool, existing []Entry,
) ([]Entry, []exportmsg.Message, error) {
	if len(rows) == 0 {
		return nil, nil, nil
	}

	excluded := map[string]map[string]bool{}
	taken := map[string]bool{}
	for _, e := range existing {
		taken[strings.ToLower(e.Path)] = true
		lower := strings.ToLower(e.Path)
		if !strings.Contains(lower, "/localisation/") || !strings.HasSuffix(lower, ".yml") {
			continue
		}
		for locale, keys := range ymlLocaleKeys(string(e.Bytes)) {
			if _, ok := excluded[locale]; !ok {
				excluded[locale] = map[string]bool{}
			}
			for k := range keys {
				excluded[locale][k] = true
			}
		}
	}

	grouped := map[string][]preservedLocRow{}
	var warnings []exportmsg.Message
	var illegalKeys []string
	rejected := 0
	for _, row := range rows {

		if !includeImported && strings.EqualFold(row.Source, "imported") {
			continue
		}
		if row.Locale == "" || row.Key == "" {
			continue
		}

		if !emitlocout.IsLegalLocKey(row.Key) {
			illegalKeys = append(illegalKeys, row.Key)
			continue
		}
		if excluded[row.Locale][row.Key] {
			continue
		}
		rel, ok := safePreservedSourceFile(row.SourceFile)
		if !ok {
			rejected++
			continue
		}
		grouped[rel] = append(grouped[rel], row)
	}
	if rejected > 0 {
		warnings = append(warnings, exportmsg.Message{Key: "preserved_loc.bad_source", Args: exportmsg.Args{"count": rejected}})
	}
	if len(illegalKeys) > 0 {
		sort.Strings(illegalKeys)
		shown := illegalKeys
		if len(shown) > 5 {
			shown = shown[:5]
		}
		warnings = append(warnings, exportmsg.Message{Key: "preserved_loc.bad_key", Args: exportmsg.Args{
			"count": len(illegalKeys), "keys": strings.Join(shown, ", "),
		}})
	}
	if len(grouped) == 0 {
		return nil, warnings, nil
	}

	files := make([]string, 0, len(grouped))
	for rel := range grouped {
		files = append(files, rel)
	}

	sort.Slice(files, func(i, j int) bool {
		li, lj := strings.ToLower(files[i]), strings.ToLower(files[j])
		if li != lj {
			return li < lj
		}
		return files[i] < files[j]
	})

	entries := make([]Entry, 0, len(files))
	for _, rel := range files {
		rowsFor := grouped[rel]

		sort.SliceStable(rowsFor, func(i, j int) bool {
			a, b := rowsFor[i], rowsFor[j]
			if a.SourceOrder != b.SourceOrder {
				return a.SourceOrder < b.SourceOrder
			}
			if a.EntryOrder != b.EntryOrder {
				return a.EntryOrder < b.EntryOrder
			}
			return a.ID < b.ID
		})
		var body bytes.Buffer
		currentLocale := ""
		for _, row := range rowsFor {

			if row.Locale != currentLocale {
				body.WriteString(row.Locale + ":\n")
				currentLocale = row.Locale
			}
			body.WriteString(preservedLocLine(row.Key, row.Version, row.Value) + "\n")
		}
		target := modDir + "/" + rel

		if taken[strings.ToLower(target)] {
			dir, file := path.Split(rel)
			stem := strings.TrimSuffix(file, path.Ext(file))
			ext := path.Ext(file)
			candidate := dir + stem + "_hagane_preserved" + ext
			for i := 2; taken[strings.ToLower(modDir+"/"+candidate)]; i++ {
				candidate = fmt.Sprintf("%s%s_hagane_preserved_%d%s", dir, stem, i, ext)
			}
			target = modDir + "/" + candidate
		}
		taken[strings.ToLower(target)] = true
		entries = append(entries, Entry{
			Path:         target,
			Bytes:        append([]byte(bomUTF8), body.Bytes()...),
			Origin:       "preserved_localisation",
			SourceEntity: "project_loc_file",
		})
	}
	return entries, warnings, nil
}
