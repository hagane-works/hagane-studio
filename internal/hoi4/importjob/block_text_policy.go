package importjob

import (
	"fmt"
	"sort"
	"strings"
)

var importedBlockFields = map[string]bool{
	"after": true, "activation": true, "ai_chance": true, "ai_will_do": true,
	"allow": true, "allow_branch": true, "allowed": true,
	"allowed_civil_war": true, "allowed_to_remove": true,
	"available": true, "bypass": true, "cancel": true,
	"cancel_trigger": true, "complete_effect": true,
	"completion_reward": true, "default_trigger": true,
	"do_effect": true, "effects": true, "hidden_effects": true,
	"immediate": true, "mean_time_to_happen": true, "modifier": true,
	"on_add": true, "on_map_area": true, "on_research_complete": true,
	"on_research_complete_limit": true, "on_remove": true,
	"project_output": true, "remove_effect": true, "resource_cost": true,
	"select_effect": true, "trigger": true, "visible": true,
}

type ImportedBlockTextIssue struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

type ImportedBlockTextResult struct {
	Value      any                      `json:"value"`
	Normalized int                      `json:"normalized"`
	Issues     []ImportedBlockTextIssue `json:"issues,omitempty"`
}

func NormalizeImportedBlockPayload(value any) (ImportedBlockTextResult, error) {
	result := ImportedBlockTextResult{}
	normalized, err := normalizeImportedValue(value, nil, &result)
	result.Value = normalized
	return result, err
}

func normalizeImportedValue(value any, path []string, result *ImportedBlockTextResult) (any, error) {
	switch current := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(current))
		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			childPath := appendPath(path, key)
			if text, ok := current[key].(string); ok && isImportedBlockField(key) {
				canonical, count, issue, err := canonicalizeImportedBlockText(text, strings.Join(childPath, "."))
				if issue != nil {
					result.Issues = append(result.Issues, *issue)
				}
				if err != nil {
					return nil, err
				}
				result.Normalized += count
				out[key] = canonical
				continue
			}
			child, err := normalizeImportedValue(current[key], childPath, result)
			if err != nil {
				return nil, err
			}
			out[key] = child
		}
		return out, nil
	case []any:
		out := make([]any, len(current))
		for i := range current {
			child, err := normalizeImportedValue(current[i], appendPath(path, fmt.Sprintf("[%d]", i)), result)
			if err != nil {
				return nil, err
			}
			out[i] = child
		}
		return out, nil
	default:
		return value, nil
	}
}

func appendPath(path []string, item string) []string {
	out := make([]string, len(path)+1)
	copy(out, path)
	out[len(path)] = item
	return out
}

func isImportedBlockField(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	return importedBlockFields[key] || strings.HasSuffix(key, "_trigger") ||
		strings.HasSuffix(key, "_effect") || strings.HasSuffix(key, "_effects")
}

func canonicalizeImportedBlockText(src, fieldPath string) (string, int, *ImportedBlockTextIssue, error) {
	out := make([]byte, 0, len(src))
	inComment, inQuote, quoteEscape := false, false, false
	line, column, count := 1, 1, 0
	for i := 0; i < len(src); {
		c := src[i]
		if c == '\n' {
			out = append(out, c)
			i++
			line, column = line+1, 1
			inComment, quoteEscape = false, false
			continue
		}
		if inComment {
			out = append(out, c)
			i++
			column++
			continue
		}
		if c == '#' && !inQuote {
			inComment = true
			out = append(out, c)
			i++
			column++
			continue
		}
		if c == '"' && !quoteEscape {
			inQuote = !inQuote
			out = append(out, c)
			i++
			column++
			continue
		}
		if c != '\\' {
			quoteEscape = false
			out = append(out, c)
			i++
			column++
			continue
		}
		run := 1
		for i+run < len(src) && src[i+run] == '\\' {
			run++
		}
		if i+run >= len(src) || src[i+run] != 'n' {
			out = append(out, src[i:i+run]...)
			quoteEscape = inQuote && run%2 == 1
			i += run
			column += run
			continue
		}
		issue := &ImportedBlockTextIssue{Path: fieldPath, Line: line, Column: column,
			Message: "该段可能被游戏当成一整行，后续内容不生效；请改成真实换行。"}
		if inQuote || run != 1 {
			return src, count, issue, fmt.Errorf("%s at %d:%d: ambiguous literal newline", fieldPath, line, column)
		}
		out = append(out, '\n')
		count++
		i += 2
		line, column = line+1, 1
		quoteEscape = false
	}
	return string(out), count, nil, nil
}
