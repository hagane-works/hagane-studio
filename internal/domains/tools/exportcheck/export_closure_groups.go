package exportcheck

import (
	"context"
)

func (s *Store) closureGroups(
	ctx context.Context, projectID int64, includeImported bool,
) (map[string]any, error) {
	data := contentLanguageSpecsData()
	out := map[string]any{}

	for group, table := range data.GroupTables {
		if len(table) != 2 {
			continue
		}
		tableName, sourceGated := table[0].(string), table[1].(bool)
		if !sqlIdentifierRE.MatchString(tableName) {
			continue
		}
		specName := data.GroupEntitySpecs[group]
		spec, ok := data.EntitySpecs[specName]
		if !ok {
			continue
		}
		rows, err := s.loadClosureRows(ctx, projectID, tableName, spec,
			data, sourceGated && !includeImported)
		if err != nil {
			return nil, err
		}

		if group == "trees" {
			if err := s.attachFocusNodes(ctx, projectID, rows); err != nil {
				return nil, err
			}
		}
		out[group] = rows
	}

	allChars, err := s.loadClosureRows(ctx, projectID, "characters",
		data.EntitySpecs[data.GroupEntitySpecs["characters"]], data, false)
	if err != nil {
		return nil, err
	}
	if chars := allChars; len(chars) >= 0 {
		out["replacement_country_leaders"] = filterReplacements(chars,
			func(role string) bool { return role == "country_leader" })
		out["replacement_unit_leaders"] = filterReplacements(chars,
			func(role string) bool { return unitLeaderRoles[role] })
		out["replacement_scientists"] = filterReplacements(chars,
			func(role string) bool { return scientistRoles[role] })
	}
	return out, nil
}

var unitLeaderRoles = map[string]bool{
	"general": true, "field_marshal": true, "admiral": true, "corps_commander": true,
}

var scientistRoles = map[string]bool{"scientist": true}

func filterReplacements(rows []map[string]any, keep func(string) bool) []map[string]any {
	out := []map[string]any{}
	for _, r := range rows {
		if anyTruthy(r["replaces_vanilla_id"]) && keep(anyToString(r["role_type"])) {
			out = append(out, r)
		}
	}
	return out
}
