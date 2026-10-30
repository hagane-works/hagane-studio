package exportcheck

import (
	"context"
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed gendata/locref_scan_specs.json
var locrefScanSpecsJSON []byte

type locrefSpec struct {
	Kind      string   `json:"kind"`
	Table     string   `json:"table"`
	Fields    []string `json:"fields"`
	HasSource bool     `json:"has_source"`
	IDExpr    string   `json:"id_expr"`
}

type locrefSpecs struct {
	Specs             []locrefSpec `json:"specs"`
	SelfDefiningKinds []string     `json:"self_defining_kinds"`
}

var locrefSpecsOnce sync.Once

var locrefSpecsData locrefSpecs

func locrefScanSpecs() *locrefSpecs {
	locrefSpecsOnce.Do(func() {
		if err := json.Unmarshal(locrefScanSpecsJSON, &locrefSpecsData); err != nil {
			panic("locref_scan_specs.json 解析失败: " + err.Error())
		}

		for _, spec := range locrefSpecsData.Specs {
			if !sqlIdentifierRE.MatchString(spec.Table) {
				panic("locref spec 表名非法: " + spec.Table)
			}
			for _, f := range spec.Fields {
				if !sqlIdentifierRE.MatchString(f) {
					panic("locref spec 字段名非法: " + spec.Table + "." + f)
				}
			}
		}
	})
	return &locrefSpecsData
}

var locrefIDColumns = map[string]string{
	"character": "tag", "custom_trait": "trait_id",
	"dynamic_modifier": "dm_id", "idea": "idea_id", "event": "event_id",
	"decision_category": "category_id", "decision": "decision_id",
	"history_file": "country_tag", "scripted_snippet": "name",
	"scripted_localisation": "name", "ui_panel": "token",

	"super_event": locrefComposedID,
}

const locrefComposedID = "<composed>"

func (s *Store) collectLocalisationReferenceIssues(
	ctx context.Context, projectID int64, includeImported bool,
) ([]map[string]any, error) {
	specs := locrefScanSpecs()
	defined, err := s.locrefDefinedKeys()
	if err != nil {
		return nil, err
	}

	sources := []ScriptLocSource{}
	selfDefining := map[string]bool{}
	for _, k := range specs.SelfDefiningKinds {
		selfDefining[k] = true
	}

	if err := s.scanFocusNodeLocRefs(ctx, projectID, includeImported,
		&sources, defined); err != nil {
		return nil, err
	}

	for _, spec := range specs.Specs {
		idCol := locrefIDColumns[spec.Kind]
		if idCol == "" {
			continue
		}
		cols := []string{"id"}
		if idCol == locrefComposedID {
			cols = append(cols, "country_tag", "number")
		} else {
			cols = append(cols, idCol)
		}
		cols = append(cols, spec.Fields...)

		extra := []string{}
		if selfDefining[spec.Kind] {
			extra = []string{"display_name_locs", "name_en", "name_zh",
				"description_locs", "desc_en", "desc_zh"}
		}
		if spec.Kind == "decision" {
			extra = append(extra, "effect_tooltip_locs")
		}
		if err := s.scanLocRefTable(ctx, projectID, spec, idCol, cols, extra,
			includeImported, &sources, defined); err != nil {
			return nil, err
		}
	}
	return ScanMissingLocalisationReferences(sources, defined), nil
}
