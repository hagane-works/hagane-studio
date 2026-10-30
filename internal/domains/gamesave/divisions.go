package gamesave

import (
	"strings"

	"haganeworks/internal/domains/military"
	"haganeworks/internal/project"
)

const (
	divisionTemplateTable   = "division_templates"
	divisionDeploymentTable = "division_deployments"
)

type divisionTemplate = military.DivisionTemplate

func templateFromRow(row project.Row) divisionTemplate {
	var t divisionTemplate
	decodeColumn(row, &t)
	if t.Source == "" {
		t.Source = "created"
	}
	if t.Regiments == nil {
		t.Regiments = []map[string]any{}
	}
	if t.Support == nil {
		t.Support = []map[string]any{}
	}
	if t.RawExtras == nil {
		t.RawExtras = map[string]any{}
	}
	t.Warnings = []string{}
	return t
}

func (s *store) templates(oob, tag *string) []divisionTemplate {
	rows := s.sortedRows(divisionTemplateTable, func(row project.Row) bool {
		return (oob == nil || text(row, "oob_file") == *oob) && (tag == nil || text(row, "country_tag") == *tag)
	}, func(a, b project.Row) bool {
		if x, y := text(a, "oob_file"), text(b, "oob_file"); x != y {
			return x < y
		}
		if x, y := number(a["ordering"]), number(b["ordering"]); x != y {
			return x < y
		}
		return text(a, "template_name") < text(b, "template_name")
	})
	out := make([]divisionTemplate, 0, len(rows))
	for _, row := range rows {
		out = append(out, templateFromRow(row))
	}
	return out
}

type deployment struct {
	ID               int64
	DivisionTemplate string
}

func (s *store) deployments(oob string) []deployment {
	rows := s.sortedRows(divisionDeploymentTable, func(row project.Row) bool { return text(row, "oob_file") == oob }, byOrdering)
	out := make([]deployment, 0, len(rows))
	for _, row := range rows {
		out = append(out, deployment{ID: project.Int64(row["id"]), DivisionTemplate: text(row, "division_template")})
	}
	return out
}

func (s *store) setDeploymentTemplate(id int64, name string) error {
	_, err := s.update(divisionDeploymentTable, id, map[string]any{"division_template": strings.TrimSpace(name)})
	return err
}
