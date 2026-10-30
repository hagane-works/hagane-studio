package importjob

import (
	"encoding/json"
	"fmt"
	"haganeworks/internal/project"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/importsvc/miscstructured"
)

type StructuredMiscResult struct {
	Counts   map[string]int64
	Warnings []string
}

func parseStructured(kind, text, source string) ([]map[string]any, error) {
	switch kind {
	case "modifier_definitions":
		return miscstructured.ModifierDefinitions(text, source)
	case "ai_strategy_plans":
		return miscstructured.AIPlans(text, source)
	case "ai_templates":
		return miscstructured.AITemplates(text, source)
	case "wargoals":
		return miscstructured.Wargoals(text, source)
	}
	return nil, fmt.Errorf("unsupported kind")
}
func structuredKey(k string) string {
	return map[string]string{"modifier_definitions": "modifier_id", "ai_strategy_plans": "plan_id", "ai_templates": "template_id", "wargoals": "wargoal_key"}[k]
}
func structuredKind(n string) string {
	for _, x := range []struct{ p, k string }{{"common/modifier_definitions/", "modifier_definitions"}, {"common/ai_strategy_plans/", "ai_strategy_plans"}, {"common/ai_templates/", "ai_templates"}, {"common/wargoals/", "wargoals"}} {
		if strings.HasPrefix(n, x.p) {
			return x.k
		}
	}
	return ""
}
func structuredMiscFiles(root string) ([]string, error) {
	var out []string
	e := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(root, path)
			n := strings.ToLower(filepath.ToSlash(rel))
			if strings.HasSuffix(n, ".txt") && structuredKind(n) != "" {
				out = append(out, path)
			}
		}
		return nil
	})
	sort.Strings(out)
	return out, e
}

type structuredRow struct {
	kind, naturalKey, sourceFile string
	payload                      payload
}

type structuredStage struct{ rows []structuredRow }

func (s *session) stageStructuredMisc(progress func(EntityProgress)) (StructuredMiscResult, error) {
	res := StructuredMiscResult{Counts: map[string]int64{}}
	files, err := structuredMiscFiles(s.root)
	if err != nil {
		return res, err
	}

	type structuredFile struct {
		rows    []structuredRow
		raw     *miscRawRow
		warning string
	}
	var total int64
	err = inFileOrder(s.ctx, len(files), func(i int) (structuredFile, error) {
		rel, _ := filepath.Rel(s.root, files[i])
		rel = filepath.ToSlash(rel)
		body, err := os.ReadFile(files[i])
		if err != nil {
			return structuredFile{}, err
		}
		kind := structuredKind(strings.ToLower(rel))
		rows, err := parseStructured(kind, string(body), rel)
		if err != nil {
			return structuredFile{
				raw:     &miscRawRow{kind: kind, path: rel, content: string(body)},
				warning: fmt.Sprintf("%s kept as raw file: %v", rel, err),
			}, nil
		}
		var file structuredFile
		for _, r := range rows {

			raw, _ := json.Marshal(r)
			file.rows = append(file.rows, structuredRow{
				kind: kind, naturalKey: fmt.Sprint(r[structuredKey(kind)]), sourceFile: rel, payload: decodePayload(raw),
			})
		}
		return file, nil
	}, func(i int, file structuredFile) error {
		if file.raw != nil {
			res.Warnings = append(res.Warnings, file.warning)
			s.miscRaw = append(s.miscRaw, *file.raw)
		}
		for _, row := range file.rows {
			s.structured.rows = append(s.structured.rows, row)
			res.Counts[row.kind]++
			total++
		}
		if progress != nil {
			progress(EntityProgress{FilesDone: i + 1, FilesTotal: len(files), Rows: total})
		}
		return nil
	})
	if err != nil {
		return res, err
	}

	s.structured.rows = lastWins(s.structured.rows, func(r structuredRow) string { return r.kind + "\x00" + r.naturalKey })

	s.miscRaw = lastWins(s.miscRaw, miscRawKey)
	return res, nil
}

func (s *session) mergeStructuredMisc(tx *project.Tx, options MiscMergeOptions) error {
	tables := map[string]*keyedTable{
		"modifier_definitions": newKeyedTable(tx, "modifier_definitions", "modifier_id"),
		"ai_strategy_plans":    newKeyedTable(tx, "ai_strategy_plans", "plan_id"),
		"ai_templates":         newKeyedTable(tx, "ai_templates", "template_id"),
		"wargoals":             newKeyedTable(tx, "wargoal_types", "wargoal_key"),
	}
	rows := s.structured.rows
	if strings.EqualFold(options.ConflictPolicy, "skip") {

		kept := make([]structuredRow, 0, len(rows))
		for _, r := range rows {
			if table := tables[r.kind]; table != nil {
				if _, exists := table.find(r.naturalKey); exists {
					continue
				}
			}
			kept = append(kept, r)
		}
		rows = kept
	}

	for _, kind := range []string{"modifier_definitions", "ai_strategy_plans", "ai_templates", "wargoals"} {
		for _, r := range rows {
			if r.kind == kind {
				row, update := structuredMergeRow(r)
				tables[kind].upsert(row, update)
			}
		}
	}
	return s.mergeMiscEntities(tx, options)
}

func structuredMergeRow(r structuredRow) (project.Row, []string) {
	p := r.payload
	switch r.kind {
	case "modifier_definitions":

		return project.Row{
			"modifier_id": r.naturalKey,
			"color_type":  p.strOr("color_type", "bad"), "value_type": p.strOr("value_type", "number"),
			"precision": p.intOr("precision", 2), "postfix": p.strOr("postfix", "none"),
			"categories": p.jsonOr("categories", []any{"all"}), "source_file": r.sourceFile,
			"explicit_fields": p.jsonOr("explicit_fields", []any{}), "source": "imported",
		}, []string{"color_type", "value_type", "precision", "postfix", "categories", "source_file", "explicit_fields", "source"}
	case "ai_strategy_plans":

		lists := []string{"allowed", "enable", "abort", "ai_national_focuses", "focus_factors", "research", "ideas", "traits", "ai_strategy", "weight", "field_order"}
		row := project.Row{
			"plan_id": r.naturalKey, "name": p.str("name"), "desc": p.str("desc"),
			"raw_block": p.str("raw_block"), "source_file": r.sourceFile, "source": "imported",
		}
		for _, column := range lists {
			row[column] = p.jsonOr(column, []any{})
		}
		update := append([]string{"name", "desc"}, lists...)
		return row, append(update, "raw_block", "source_file", "source")
	case "ai_templates":

		return project.Row{
				"template_id": r.naturalKey, "role": p.strOr("role", "infantry"),
				"blocked_for": p.str("blocked_for"), "available_for": p.str("available_for"),
				"upgrade_prio": p.str("upgrade_prio"), "enable": p.str("enable"),
				"front_role_override": p.str("front_role_override"),
				"target_templates":    p.jsonOr("target_templates", []any{}), "field_order": p.jsonOr("field_order", []any{}),
				"raw_block": p.str("raw_block"), "source_file": r.sourceFile, "source": "imported",
			}, []string{"role", "blocked_for", "available_for", "upgrade_prio", "enable", "front_role_override",
				"target_templates", "field_order", "raw_block", "source_file", "source"}
	default:

		return project.Row{
				"wargoal_key": r.naturalKey, "war_name": p.str("war_name"),
				"safe_scalars": p.jsonOr("safe_scalars", map[string]any{}),
				"allowed":      p.str("allowed"), "available": p.str("available"), "take_states": p.str("take_states"),
				"puppet": p.str("puppet"), "liberate": p.str("liberate"),
				"ordered_entries":              p.jsonOr("ordered_entries", []any{}),
				"definition_leading_comments":  p.jsonOr("definition_leading_comments", []any{}),
				"definition_trailing_comments": p.jsonOr("definition_trailing_comments", []any{}),
				"file_trivia":                  p.jsonOr("file_trivia", map[string]any{}),
				"source_file":                  r.sourceFile, "source": "imported",
			}, []string{"war_name", "safe_scalars", "allowed", "available", "take_states", "puppet", "liberate",
				"ordered_entries", "definition_leading_comments", "definition_trailing_comments", "file_trivia", "source_file", "source"}
	}
}
