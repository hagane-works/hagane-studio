package characters

import (
	"fmt"
	"strings"

	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
	"haganeworks/internal/reftables"
)

const worldHistoryNote = "世界地图 · 全新世界"

func legacyScalar(entries []*paradox.Entry, key string) (string, bool) {
	for _, entry := range entries {
		if entry.Key != nil && *entry.Key == key && entry.Block == nil {
			return strings.TrimSpace(entry.Scalar.StrValue()), true
		}
	}
	return "", false
}

func legacyBlockEndLine(lines []string, start int) (int, bool) {
	depth := 0
	sawOpen := false
	inQuote := false
	escaped := false
	for lineIndex := start; lineIndex < len(lines); lineIndex++ {
		line := lines[lineIndex]
		for offset := 0; offset < len(line); offset++ {
			char := line[offset]
			if inQuote {
				if escaped {
					escaped = false
					continue
				}
				if char == '\\' {
					escaped = true
					continue
				}
				if char == '"' {
					inQuote = false
				}
				continue
			}
			switch char {
			case '#':
				offset = len(line)
			case '"':
				inQuote = true
			case '{':
				depth++
				sawOpen = true
			case '}':
				if depth == 0 {
					return 0, false
				}
				depth--
				if sawOpen && depth == 0 {
					rest := strings.TrimSpace(line[offset+1:])
					if rest != "" && !strings.HasPrefix(rest, "#") {
						return 0, false
					}
					return lineIndex + 1, true
				}
			}
		}
	}
	return 0, false
}

func stripGeneratedWorldLeader(content, countryTag string) (string, bool) {
	ast, recoveries, err := paradox.Parse(content)
	if err != nil || len(recoveries) != 0 {
		return content, false
	}
	wantedName := "Leader of " + strings.ToUpper(strings.TrimSpace(countryTag))
	for _, entry := range ast.Entries {
		if entry.Key == nil || *entry.Key != "create_country_leader" || entry.Block == nil {
			continue
		}
		body := entry.Block.Entries
		name, hasName := legacyScalar(body, "name")
		desc, hasDesc := legacyScalar(body, "desc")
		picture, hasPicture := legacyScalar(body, "picture")
		ideology, hasIdeology := legacyScalar(body, "ideology")
		if !hasName || name != wantedName || !hasDesc || desc != "" ||
			!hasPicture || !strings.HasPrefix(picture, "GFX_portrait_") ||
			!hasIdeology || ideology != "despotism" {
			continue
		}
		known := 0
		traitsOK := false
		for _, child := range body {
			if child.Key == nil {
				continue
			}
			switch *child.Key {
			case "name", "desc", "picture", "ideology":
				known++
			case "traits":
				known++
				traitsOK = child.Block != nil && len(child.Block.Entries) == 0
			default:
				known = -100
			}
		}
		if known != 5 || !traitsOK {
			continue
		}

		newline := "\n"
		normalized := content
		if strings.Contains(content, "\r\n") {
			newline = "\r\n"
			normalized = strings.ReplaceAll(content, "\r\n", "\n")
		}
		lines := strings.Split(normalized, "\n")
		start := entry.Line - 1

		if start < 0 || start >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[start]), "create_country_leader") {
			return content, false
		}
		end, ok := legacyBlockEndLine(lines, start)
		if !ok {
			return content, false
		}
		kept := append([]string{}, lines[:start]...)
		kept = append(kept, lines[end:]...)
		return strings.Join(kept, newline), true
	}
	return content, false
}

func normalizeGeneratedWorldLeader(tx *project.Tx, characterID int64) error {
	row := tx.Find(table, characterID)
	if row == nil {
		return errCharacterMissing
	}
	countryTag := strings.ToUpper(str(row["country_tag"]))
	if str(row["role_type"]) != "country_leader" || strings.ToLower(str(row["ideology"])) != "despotism" || countryTag == "" {
		return nil
	}

	neutral := map[string]bool{}
	for _, ideology := range reftables.Ideologies().Members("neutrality") {
		neutral[ideology] = true
	}
	neutralLeaders := 0
	for _, other := range tx.Rows(table) {
		if strings.ToUpper(str(other["country_tag"])) == countryTag &&
			str(other["role_type"]) == "country_leader" &&
			str(other["source"]) != "imported" &&
			neutral[strings.ToLower(str(other["ideology"]))] {
			neutralLeaders++
		}
	}
	if neutralLeaders != 1 {
		return nil
	}
	path := strings.ToLower(fmt.Sprintf("history/countries/%s - New World.txt", countryTag))
	for _, raw := range tx.Rows("project_raw_files") {
		if strings.ToLower(str(raw["dest_path"])) != path || raw["is_binary"] == true || str(raw["note"]) != worldHistoryNote {
			continue
		}
		cleaned, changed := stripGeneratedWorldLeader(str(raw["content"]), countryTag)
		if !changed {
			return nil
		}
		_, err := tx.Update("project_raw_files", project.Int64(raw["id"]), func(r project.Row) {
			r["content"] = cleaned
			flipImported(r)
		})
		return err
	}
	return nil
}
