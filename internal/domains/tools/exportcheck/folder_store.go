package exportcheck

import (
	"context"
	"encoding/json"
	"haganeworks/internal/project"
	"sort"
	"strings"
)

type Store struct {
	p *project.Project

	cache map[string][]project.Row
}

func newStore(p *project.Project) *Store {
	return &Store{p: p, cache: map[string][]project.Row{}}
}

func (s *Store) rows(table string) ([]project.Row, error) {
	if rows, ok := s.cache[table]; ok {
		return rows, nil
	}
	rows, err := s.p.Rows(table)
	if err != nil {
		return nil, err
	}
	s.cache[table] = rows
	return rows, nil
}

func decode(raw []byte, dst any) {
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, dst)
	}
}

func decodeRow(row project.Row, dst any) {
	raw, err := json.Marshal(row)
	if err != nil {
		return
	}
	_ = json.Unmarshal(raw, dst)
}

func colBytes(row project.Row, col string) []byte {
	v, ok := row[col]
	if !ok || v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		trimmed := strings.TrimSpace(s)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			return []byte(s)
		}
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return raw
}

func colStr(row project.Row, col string) string {
	switch v := row[col].(type) {
	case nil:
		return ""
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		raw, _ := json.Marshal(v)
		return string(raw)
	}
}

func colStrPtr(row project.Row, col string) *string {
	if row[col] == nil {
		return nil
	}
	v := colStr(row, col)
	return &v
}

func colInt(row project.Row, col string) int64 { return project.Int64(row[col]) }

func colBool(row project.Row, col string) bool {
	switch v := row[col].(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "t" || v == "1"
	case json.Number:
		return v.String() != "0"
	}
	return false
}

func (s *Store) GetMetadata(_ context.Context, id int64) (Project, bool, error) {
	record := s.p.Record()
	if record == nil || (id != 0 && id != s.p.ID()) {
		return Project{}, false, nil
	}
	var p Project
	decodeRow(record, &p)
	p.ID = s.p.ID()
	p.StageStateIDs = parseStage(colStr(record, "stage_state_ids"))
	if raw := colBytes(record, "owned_dlcs"); raw != nil && string(raw) != "null" {
		var owned []string
		if json.Unmarshal(raw, &owned) == nil {
			p.OwnedDLCs = &owned
		}
	} else {
		p.OwnedDLCs = nil
	}
	normalize(&p)
	return p, true, nil
}

func parseStage(raw string) []int64 {
	var values []any
	if json.Unmarshal([]byte(raw), &values) != nil {
		return nil
	}
	seen := map[int64]bool{}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		number, ok := value.(float64)
		if !ok || number < 1 || number != float64(int64(number)) {
			continue
		}
		id := int64(number)
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalize(p *Project) {
	if p.CountryTags == nil {
		if p.CountryTag != "" {
			p.CountryTags = []string{p.CountryTag}
		} else {
			p.CountryTags = []string{}
		}
	}
	if p.IdeologyIconPaths == nil {
		p.IdeologyIconPaths = map[string]string{}
	}
	if p.Version == "" {
		p.Version = "1.0"
	}
	if len(p.Tags) == 0 {
		p.Tags = []string{"Alternative History"}
	}
	if p.Dependencies == nil {
		p.Dependencies = []string{}
	}
	if p.ImportedReplacePaths == nil {
		p.ImportedReplacePaths = []string{}
	}
	if p.SupportedVersion == "" {
		p.SupportedVersion = "1.19.*"
	}
	if p.ScenarioHistoryPolicy == "" {
		p.ScenarioHistoryPolicy = ScenarioHistoryPolicyInheritBaseHistory
	}
	if len(p.ExportLanguages) == 0 {
		p.ExportLanguages = []string{"en"}
	}
	if p.TechYearColumns == nil {
		p.TechYearColumns = []any{}
	}
	if p.IdeologyRegistry == nil {
		p.IdeologyRegistry = map[string]any{}
	}
	if p.TechSurfaceManifest == nil {
		p.TechSurfaceManifest = map[string]any{}
	}
	if p.IdeologySourceManifest == nil {
		p.IdeologySourceManifest = map[string]any{}
	}
}

func decodeRows[T any](s *Store, table string) ([]T, error) {
	rows, err := s.rows(table)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(rows))
	for _, row := range rows {
		var v T
		decodeRow(row, &v)
		out = append(out, v)
	}
	return out, nil
}

func loadFocusPolicyRows(_ context.Context, s *Store, _ int64) ([]focusPolicyRow, error) {
	rows, err := s.rows("focus_file_policies")
	if err != nil {
		return nil, err
	}
	out := []focusPolicyRow{}
	for _, row := range rows {
		r := focusPolicyRow{ID: colInt(row, "id")}
		r.Policy.SourceFile = colStr(row, "source_file")
		r.Policy.OriginKind = colStr(row, "origin_kind")
		decode(colBytes(row, "tree_ids"), &r.Policy.TreeIDs)
		decode(colBytes(row, "matched_tree_ids"), &r.Policy.MatchedTreeIDs)
		r.Policy.EmitMode = colStr(row, "emit_mode")
		r.Policy.TargetPath = colStr(row, "target_path")
		r.Policy.TargetLayer = colStr(row, "target_layer")
		r.Policy.DecisionSource = colStr(row, "decision_source")
		if r.Policy.TreeIDs == nil {
			r.Policy.TreeIDs = []string{}
		}
		if r.Policy.MatchedTreeIDs == nil {
			r.Policy.MatchedTreeIDs = []string{}
		}
		out = append(out, r)
	}
	return out, nil
}
