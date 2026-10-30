package exportcheck

import (
	"encoding/json"
	"fmt"
	"haganeworks/internal/hoi4/emit/locout"
	"strings"
)

type DisplayNameRow struct {
	ID     string
	Source string
	Attrs  map[string]any
}

type DisplayNameKind struct {
	Kind            string
	Singular        string
	Plural          string
	Rows            []DisplayNameRow
	LiteralAttr     string
	RowSourceScoped bool
}

func unnamedIDs(rows []DisplayNameRow, literalAttr string, rowSourceScoped bool) []string {
	named := map[string]bool{}
	out := []string{}
	for _, row := range rows {
		if rowSourceScoped {
			src := row.Source
			if src == "" {
				src = "created"
			}
			if src != "created" {
				continue
			}
		}
		ident := strings.TrimSpace(row.ID)
		if ident == "" {
			continue
		}
		if hasMeaningfulDisplayName(row.Attrs, ident, literalAttr) {
			named[ident] = true
		} else {
			seen := false
			for _, o := range out {
				if o == ident {
					seen = true
					break
				}
			}
			if !seen {
				out = append(out, ident)
			}
		}
	}
	filtered := out[:0]
	for _, i := range out {
		if !named[i] {
			filtered = append(filtered, i)
		}
	}
	return filtered
}

func hasMeaningfulDisplayName(attrs map[string]any, ident, literalAttr string) bool {
	if attrs == nil {
		attrs = map[string]any{}
	}
	raw, err := json.Marshal(attrs)
	if err != nil {
		return false
	}
	v, err := locout.ParseJSON(raw)
	if err != nil || v.Kind != locout.KObj {
		return false
	}
	return locout.HasMeaningfulDisplayNameLiteral(v.Obj, locout.Str(ident), literalAttr)
}

func validateDisplayNamePlaceholder(kinds []DisplayNameKind) []map[string]any {
	issues := []map[string]any{}
	for _, k := range kinds {
		missing := unnamedIDs(k.Rows, k.LiteralAttr, k.RowSourceScoped)
		if len(missing) == 0 {
			continue
		}
		n := len(missing)
		sample := missing
		if n > 5 {
			sample = missing[:5]
		}
		sampleTxt := strings.Join(sample, ", ")
		if n > len(sample) {
			sampleTxt += ", …"
		}
		one, many := k.Singular, k.Plural
		noun := many
		has := "have"
		it := "them"
		if n == 1 {
			noun = one
			has = "has"
			it = "it"
		}
		issues = append(issues, map[string]any{
			"level": "info", "code": "display_name_placeholder",
			"entity_ref": nil,
			"detail": map[string]any{
				"kind": k.Kind, "count": n, "sample_ids": sample,
			},
			"message": fmt.Sprintf(
				"%d %s %s no display name yet (%s) — the export writes no name line "+
					"for %s, so HOI4 shows the id in-game. That is intended when the id "+
					"is a vanilla one (the game's own name shows instead); otherwise "+
					"fill in a name.",
				n, noun, has, sampleTxt, it),
		})
	}
	return issues
}
