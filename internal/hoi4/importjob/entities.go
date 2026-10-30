package importjob

import (
	"fmt"
	"haganeworks/internal/project"
	"os"
	"path/filepath"
	"strings"

	ideas "haganeworks/internal/hoi4/importsvc/ideas"
)

type EntityProgress struct {
	FilesDone, FilesTotal int
	Rows                  int64
}

type entityRow struct {
	kind, naturalKey, sourceFile string
	payload                      []byte
	decoded                      payload
}

func (r *entityRow) data() payload {
	if r.decoded == nil {
		r.decoded = decodePayload(r.payload)
	}
	return r.decoded
}

func simpleEntityFiles(root string) ([]string, error) {
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
		if strings.HasSuffix(n, ".txt") && (strings.HasPrefix(n, "common/opinion_modifiers/") || strings.HasPrefix(n, "common/dynamic_modifiers/") || strings.HasPrefix(n, "common/autonomous_states/") || strings.HasPrefix(n, "common/scripted_effects/") || strings.HasPrefix(n, "common/scripted_triggers/") || strings.HasPrefix(n, "common/scripted_localisation/")) {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func parseSimpleEntity(rel, text string) ([]entityRow, error) {
	n := strings.ToLower(rel)
	var kind, key string
	var parsed []*ideas.Dict
	var err error
	switch {
	case strings.HasPrefix(n, "common/opinion_modifiers/"):
		kind, key = "opinion_modifier", "modifier_id"
		parsed, _, err = ideas.ImportOpinionModifiers(text)
	case strings.HasPrefix(n, "common/dynamic_modifiers/"):
		kind, key = "dynamic_modifier", "dm_id"
		parsed, _, err = ideas.ImportDynamicModifiers(text)
	case strings.HasPrefix(n, "common/autonomous_states/"):
		kind, key = "autonomy_state", "autonomy_id"
		parsed, _, err = ideas.ImportAutonomyStates(text)
	case strings.HasPrefix(n, "common/scripted_effects/"):
		kind, key = "scripted_effect", "name"
		parsed, _, err = ideas.ImportScriptedSnippets(text, "effect", rel)
	case strings.HasPrefix(n, "common/scripted_triggers/"):
		kind, key = "scripted_trigger", "name"
		parsed, _, err = ideas.ImportScriptedSnippets(text, "trigger", rel)
	case strings.HasPrefix(n, "common/scripted_localisation/"):
		kind, key = "scripted_localisation", "name"
		parsed, _, err = ideas.ImportScriptedLocalisations(text, rel)
	}
	if err != nil {
		return nil, err
	}
	rows := make([]entityRow, 0, len(parsed))
	for _, native := range ideas.NativeEntities(parsed) {
		id, _ := native[key].(string)
		if id == "" {
			continue
		}
		payload, e := marshalPGJSON(native)
		if e != nil {
			return nil, e
		}
		rows = append(rows, entityRow{kind: kind, naturalKey: id, sourceFile: rel, payload: payload})
	}
	return rows, nil
}

func (s *session) stageSimpleEntities(progress func(EntityProgress)) (int64, error) {
	files, err := simpleEntityFiles(s.root)
	if err != nil {
		return 0, err
	}
	var total int64
	err = inFileOrder(s.ctx, len(files), func(i int) (fileRows[entityRow], error) {
		rel, _ := filepath.Rel(s.root, files[i])
		rel = filepath.ToSlash(rel)
		body, err := os.ReadFile(files[i])
		if err != nil {
			return fileRows[entityRow]{}, fmt.Errorf("read %s: %w", rel, err)
		}
		rows, err := parseSimpleEntity(rel, decodeScriptBytes(body))
		return fileRows[entityRow]{rel: rel, rows: rows, invalid: err}, nil
	}, func(i int, file fileRows[entityRow]) error {
		if file.invalid != nil {
			s.warnParse(file.rel, file.invalid)
			return nil
		}
		s.entities = append(s.entities, file.rows...)
		total += int64(len(file.rows))
		if progress != nil {
			progress(EntityProgress{FilesDone: i + 1, FilesTotal: len(files), Rows: total})
		}
		return nil
	})
	if err != nil {
		return total, err
	}

	s.entities = lastWins(s.entities, entityKey)
	return total, nil
}

func entityKey(row entityRow) string { return row.kind + "\x00" + row.naturalKey }

func (s *session) entityCounts() map[string]int64 {
	out := map[string]int64{}
	for _, row := range s.entities {
		out[row.kind]++
	}
	return out
}

func (s *session) mergeSimpleEntities(tx *project.Tx) error {
	opinion := newKeyedTable(tx, "opinion_modifiers", "modifier_id")
	dynamic := newKeyedTable(tx, "dynamic_modifiers", "dm_id")
	autonomy := newKeyedTable(tx, "autonomy_states", "autonomy_id")
	snippets := newKeyedTable(tx, "scripted_snippets", "kind", "name")
	scriptedLoc := newKeyedTable(tx, "scripted_localisations", "name")
	snippetKind := func(kind string) string {
		if kind == "scripted_effect" {
			return "effect"
		}
		return "trigger"
	}
	if s.options.ConflictPolicy == "skip" {
		kept := s.entities[:0]
		for _, e := range s.entities {
			var exists bool
			switch e.kind {
			case "opinion_modifier":
				_, exists = opinion.find(e.naturalKey)
			case "dynamic_modifier":
				_, exists = dynamic.find(e.naturalKey)
			case "autonomy_state":
				_, exists = autonomy.find(e.naturalKey)
			case "scripted_effect", "scripted_trigger":
				_, exists = snippets.find(snippetKind(e.kind), e.naturalKey)
			case "scripted_localisation":
				_, exists = scriptedLoc.find(e.naturalKey)
			}
			if !exists {
				kept = append(kept, e)
			}
		}
		s.entities = kept
	}
	tag := s.projectTag(tx)
	for i := range s.entities {
		e := &s.entities[i]
		p := e.data()
		switch e.kind {
		case "opinion_modifier":
			opinion.upsert(project.Row{
				"country_tag": tag, "modifier_id": e.naturalKey,
				"body": p.jsonOr("body", map[string]any{}), "raw_extras": p.jsonOr("raw_extras", map[string]any{}),
				"source": "imported",
			}, []string{"country_tag", "body", "raw_extras", "source"})
		case "dynamic_modifier":
			names := s.locNamesOrEmpty(e.naturalKey)
			dynamic.upsert(project.Row{
				"country_tag": tag, "dm_id": e.naturalKey,
				"name_en": s.locStr(e.naturalKey, "en"), "name_zh": s.locStr(e.naturalKey, "zh"), "display_name_locs": names,
				"icon": p.str("icon"), "remove_trigger": p.str("remove_trigger"), "enable": p.str("enable"),
				"modifier": p.jsonOr("modifier", map[string]any{}), "raw_extras": p.jsonOr("raw_extras", map[string]any{}),
				"source": "imported",
			}, []string{"country_tag", "name_en", "name_zh", "display_name_locs", "icon", "remove_trigger", "enable", "modifier", "raw_extras", "source"})
		case "autonomy_state":
			names := s.locNamesOrEmpty(e.naturalKey)
			autonomy.upsert(project.Row{
				"country_tag": tag, "autonomy_id": e.naturalKey,
				"name_en": s.locStr(e.naturalKey, "en"), "name_zh": s.locStr(e.naturalKey, "zh"), "display_name_locs": names,
				"is_puppet": p.boolOr("is_puppet", false), "is_default": p.boolOr("is_default", false),
				"manpower_influence": p.nullableFloat("manpower_influence"),
				"min_freedom_level":  p.intOr("min_freedom_level", 0), "rule": p.str("rule"),
				"ai_weights": p.jsonOr("ai_weights", map[string]any{}), "modifier": p.jsonOr("modifier", map[string]any{}),
				"raw_extras": p.jsonOr("raw_extras", map[string]any{}), "source": "imported",
			}, []string{"country_tag", "name_en", "name_zh", "display_name_locs", "is_puppet", "is_default", "manpower_influence", "min_freedom_level", "rule", "ai_weights", "modifier", "raw_extras", "source"})
		case "scripted_effect", "scripted_trigger":
			snippets.upsert(project.Row{
				"kind": snippetKind(e.kind), "name": e.naturalKey, "body": p.str("body"),
				"source_file": p.strOr("source_file", e.sourceFile), "source": "imported",
				"raw_extras": p.jsonOr("raw_extras", nil),
			}, []string{"body", "source_file", "source", "raw_extras"})
		case "scripted_localisation":
			scriptedLoc.upsert(project.Row{
				"name": e.naturalKey, "body": p.str("body"),
				"source_file": p.strOr("source_file", e.sourceFile), "source": "imported",
				"raw_extras": p.jsonOr("raw_extras", nil),
			}, []string{"body", "source_file", "source", "raw_extras"})
		}
	}
	return nil
}
