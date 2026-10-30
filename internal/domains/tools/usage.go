package tools

import (
	"net/http"
	"strconv"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

var usageEntitySources = []struct{ Label, Table string }{
	{"characters", "characters"},
	{"ideas", "ideas"},
	{"focus_trees", "focus_trees"},
	{"events", "events"},
	{"event_namespaces", "event_namespaces"},
	{"decisions", "decisions"},
	{"decision_categories", "decision_categories"},
	{"technologies", "technologies"},
	{"super_events", "super_events"},
	{"bookmarks", "bookmarks"},
	{"history_files", "history_files"},
	{"dynamic_modifiers", "dynamic_modifiers"},
	{"custom_traits", "custom_traits"},
	{"states", "states"},
	{"military_industrial_orgs", "military_industrial_organizations"},
	{"opinion_modifiers", "opinion_modifiers"},
	{"modifier_definitions", "modifier_definitions"},
}

func usage(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	counts := make(workbench.OrderedCounts, 0, len(usageEntitySources))
	for _, src := range usageEntitySources {
		rows, err := p.Rows(src.Table)
		if err != nil {
			return err
		}
		counts = append(counts, struct {
			Label string
			N     int64
		}{src.Label, int64(len(rows))})
	}
	name, _ := p.Record()["name"].(string)
	writeJSON(w, http.StatusOK, map[string]any{
		"project_id":    p.ID(),
		"project_name":  name,
		"entity_counts": counts,
		"entity_total":  counts.Total(),
		"audit_count":   0,
		"tokens": map[string]any{
			"scope":            "owner_user_total_approximate",
			"tokens_in_total":  0,
			"tokens_out_total": 0,
			"total_calls":      0,
		},
	})
	return nil
}

func auditLog(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	limit, offset := 50, 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > 200 {
			return apiutil.ValidationError("limit must be between 1 and 200")
		}
		limit = v
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			return apiutil.ValidationError("offset must be >= 0")
		}
		offset = v
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project_id": p.ID(), "total": 0,
		"limit": limit, "offset": offset, "logs": []any{},
	})
	return nil
}
