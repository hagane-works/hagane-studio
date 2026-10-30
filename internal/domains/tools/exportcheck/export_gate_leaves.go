package exportcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"haganeworks/internal/gameindex"
)

var slashRunRe = regexp.MustCompile(`/{2,}`)

func normalizeDescriptorReplacePath(value string, isString bool) (string, bool) {
	if !isString {
		return "", false
	}
	if strings.ContainsAny(value, "\r\n\x00") {
		return "", false
	}
	n := strings.ReplaceAll(strings.TrimSpace(value), `\`, "/")
	n = slashRunRe.ReplaceAllString(n, "/")
	for strings.HasPrefix(n, "./") {
		n = n[2:]
	}
	n = strings.TrimRight(n, "/")
	if n == "" {
		return "", false
	}
	return n, true
}

func NormalizeDescriptorReplacePaths(values []string) []string {
	ordered := []string{}
	seen := map[string]bool{}
	for _, raw := range values {
		n, ok := normalizeDescriptorReplacePath(raw, true)
		if !ok || seen[n] {
			continue
		}
		seen[n] = true
		ordered = append(ordered, n)
	}
	return ordered
}

func EffectiveDescriptorReplacePaths(imported, selected []string) []string {
	merged := append(append([]string{}, NormalizeDescriptorReplacePaths(imported)...),
		NormalizeDescriptorReplacePaths(selected)...)
	return NormalizeDescriptorReplacePaths(merged)
}

type unitFilesKey struct{}

func vanillaUnitFilenames() map[string]bool {
	names, err := gameindex.Load(context.Background(), unitFilesKey{}, gameindex.UnitFiles, func(raw []byte) (map[string]bool, error) {
		var payload struct {
			IDs []string `json:"ids"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		out := map[string]bool{}
		for _, name := range payload.IDs {
			out[strings.ToLower(name)] = true
		}
		return out, nil
	})
	if err != nil {
		return map[string]bool{}
	}
	return names
}

func NormalizeSourceFile(value string) string {
	return strings.Trim(strings.TrimSpace(strings.ReplaceAll(value, `\`, "/")), "/")
}

func ValidateUnitTypes(values []string) error {
	_, err := validateSubUnitTypes(values)
	return err
}

func ValidateCategories(values []string) error {
	_, err := validateSubUnitCategories(values)
	return err
}

var tagSanitizeRe = regexp.MustCompile(`[^A-Za-z0-9_]`)

func safeCreatedPath(path, projectTag string, importedPaths map[string]bool) bool {
	if path == "" || importedPaths[path] {
		return true
	}
	tag := strings.Trim(tagSanitizeRe.ReplaceAllString(projectTag, "_"), "_")
	safe := map[string]bool{
		"common/units/hagane_units.txt":     true,
		"common/units/hagane_sub_units.txt": true,
	}
	if tag != "" {
		safe["common/units/"+tag+"_sub_units.txt"] = true
	}
	return safe[path]
}

type SubUnitRow struct {
	SubUnitID  string
	UnitTypes  []string
	Categories []string
	SourceFile string
	Source     string
}

func CollectSubUnitExportIssues(rows []SubUnitRow, projectTag string,
	importedRawPaths []string) []map[string]any {

	imported := map[string]bool{}
	for _, p := range importedRawPaths {
		imported[NormalizeSourceFile(p)] = true
	}
	issues := []map[string]any{}
	for _, row := range rows {
		rowID := row.SubUnitID
		if err := ValidateUnitTypes(row.UnitTypes); err != nil {
			issues = append(issues, subUnitClassificationIssue(rowID, err))
			continue
		}
		if err := ValidateCategories(row.Categories); err != nil {
			issues = append(issues, subUnitClassificationIssue(rowID, err))
			continue
		}
		path := NormalizeSourceFile(row.SourceFile)
		source := row.Source
		if source == "" {
			source = "created"
		}
		if source != "imported" && !safeCreatedPath(path, projectTag, imported) {
			filename := path
			if i := strings.LastIndex(path, "/"); i >= 0 {
				filename = path[i+1:]
			}
			issues = append(issues, map[string]any{
				"level": "error", "code": "sub_unit_source_file_collision",
				"entity_ref": map[string]any{"type": "sub_unit", "id": rowID},
				"severity":   "blocker",
				"detail": map[string]any{
					"source_file":       path,
					"vanilla_collision": vanillaUnitFilenames()[strings.ToLower(filename)],
				},
				"message": fmt.Sprintf(
					"Sub-unit '%s' would write to '%s', replacing an existing "+
						"unit definition file instead of adding a project-owned file.",
					rowID, path),
				"fix_hint": "Let Hagane Works assign the output filename, or import the complete original " +
					"file before editing it.",
			})
		}
	}
	return issues
}

func subUnitClassificationIssue(rowID string, err error) map[string]any {
	return map[string]any{
		"level": "error", "code": "sub_unit_classification_invalid",
		"entity_ref": map[string]any{"type": "sub_unit", "id": rowID},
		"severity":   "blocker",
		"message": fmt.Sprintf("Sub-unit '%s' has an invalid classification: %s.",
			rowID, err.Error()),
		"fix_hint": "Use only supported internal types. Put special_forces and other " +
			"yes/no properties in their dedicated fields; custom categories remain allowed.",
	}
}
