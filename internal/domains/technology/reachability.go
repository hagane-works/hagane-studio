package technology

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

type TechUnlockPath struct {
	SourceKind string `json:"source_kind"`
	SourceID   int64  `json:"source_id"`
	SourceName string `json:"source_name"`
	FieldName  string `json:"field_name"`
}

type techEffectSource struct {
	kind, name, field, text string
	id                      int64
	owners                  map[string]bool
}

type reachabilityProject struct {
	primary string
	tags    []string
}

func ownerTags(own string, shared []string, project reachabilityProject) map[string]bool {
	result := map[string]bool{}
	if own != "" {
		result[strings.ToUpper(own)] = true
		return result
	}
	for _, tag := range shared {
		if tag != "" {
			result[strings.ToUpper(tag)] = true
		}
	}
	if len(result) > 0 {
		return result
	}
	if project.primary != "" {
		result[strings.ToUpper(project.primary)] = true
	}
	for _, tag := range project.tags {
		if tag != "" {
			result[strings.ToUpper(tag)] = true
		}
	}
	return result
}

func stringList(v any) []string {
	var out []string
	decodeColumn(v, &out)
	return out
}

func extractSetTechnology(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	block, _, err := paradox.Parse(text)
	if err != nil || block == nil {
		return nil
	}
	out := make([]string, 0)
	var walk func([]*paradox.Entry)
	walk = func(entries []*paradox.Entry) {
		for _, entry := range entries {
			if entry == nil {
				continue
			}
			if entry.Key != nil && *entry.Key == "set_technology" && entry.Block != nil {
				for _, child := range entry.Block.Entries {
					if child == nil || child.Key == nil || *child.Key == "popup" || *child.Key == "limit_to_owner" {
						continue
					}
					out = append(out, *child.Key)
				}
				continue
			}
			if entry.Block != nil {
				walk(entry.Block.Entries)
			}
		}
	}
	walk(block.Entries)
	return out
}

func loadTechEffectSources(p *project.Project) ([]techEffectSource, error) {
	record := p.Record()
	proj := reachabilityProject{primary: text(record, "country_tag", ""), tags: stringList(record["country_tags"])}
	sources := make([]techEffectSource, 0)

	trees, err := p.Rows("focus_trees")
	if err != nil {
		return nil, err
	}
	nodes, err := p.Rows("focus_nodes")
	if err != nil {
		return nil, err
	}
	nodesByTree := map[int64][]project.Row{}
	for _, node := range nodes {
		treeID := project.Int64(node["focus_tree_id"])
		nodesByTree[treeID] = append(nodesByTree[treeID], node)
	}
	for _, tree := range trees {
		owners := ownerTags(text(tree, "country_tag", ""), stringList(tree["shared_country_tags"]), proj)
		for _, node := range nodesByTree[project.Int64(tree["id"])] {
			for _, field := range []string{"completion_reward", "select_effect"} {
				if value := text(node, field, ""); value != "" && len(owners) > 0 {
					sources = append(sources, techEffectSource{"focus", text(node, "focus_id", ""), field, value, project.Int64(node["id"]), owners})
				}
			}
		}
	}

	events, err := p.Rows("events")
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		id, name := project.Int64(event["id"]), text(event, "event_id", "")
		owners := ownerTags(text(event, "country_tag", ""), stringList(event["shared_country_tags"]), proj)
		for _, field := range []string{"immediate", "after"} {
			if value := text(event, field, ""); value != "" && len(owners) > 0 {
				sources = append(sources, techEffectSource{"event", name, field, value, id, owners})
			}
		}
		var options []map[string]any
		decodeColumn(event["options"], &options)
		for index, option := range options {
			for _, field := range []string{"effects", "hidden_effects"} {
				if value, ok := option[field]; ok && value != nil && fmt.Sprint(value) != "" && len(owners) > 0 {
					sources = append(sources, techEffectSource{"event", name, fmt.Sprintf("option[%d].%s", index, field), fmt.Sprint(value), id, owners})
				}
			}
		}
	}

	decisions, err := p.Rows("decisions")
	if err != nil {
		return nil, err
	}
	for _, decision := range decisions {
		id, name := project.Int64(decision["id"]), text(decision, "decision_id", "")
		owners := ownerTags(text(decision, "country_tag", ""), stringList(decision["shared_country_tags"]), proj)
		for _, field := range []string{"complete_effect", "remove_effect", "timeout_effect", "cancel_effect"} {
			if value := text(decision, field, ""); value != "" && len(owners) > 0 {
				sources = append(sources, techEffectSource{"decision", name, field, value, id, owners})
			}
		}
	}
	return sources, nil
}

func techReachability(p *project.Project, countryTag string) ([]string, error) {
	sources, err := loadTechEffectSources(p)
	if err != nil {
		return nil, err
	}
	tag := strings.ToUpper(countryTag)
	set := map[string]bool{}
	for _, source := range sources {
		if !source.owners[tag] {
			continue
		}
		for _, techID := range extractSetTechnology(source.text) {
			set[techID] = true
		}
	}
	out := make([]string, 0, len(set))
	for techID := range set {
		out = append(out, techID)
	}
	sort.Strings(out)
	return out, nil
}

func techReachabilityPaths(p *project.Project, countryTag, techID string) ([]TechUnlockPath, error) {
	sources, err := loadTechEffectSources(p)
	if err != nil {
		return nil, err
	}
	tag := strings.ToUpper(countryTag)
	seen := map[string]bool{}
	out := make([]TechUnlockPath, 0)
	for _, source := range sources {
		if !source.owners[tag] {
			continue
		}
		found := false
		for _, candidate := range extractSetTechnology(source.text) {
			if candidate == techID {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		key := fmt.Sprintf("%s/%d/%s", source.kind, source.id, source.field)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, TechUnlockPath{source.kind, source.id, source.name, source.field})
	}
	return out, nil
}

func (h *handlers) reachability(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	tag := r.PathValue("tag")
	techIDs, err := techReachability(p, tag)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "technology reachability unavailable")
		return
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{"country_tag": strings.ToUpper(tag), "tech_ids": techIDs})
}

func (h *handlers) reachabilityPaths(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	tag, techID := r.PathValue("tag"), r.PathValue("tech")
	paths, err := techReachabilityPaths(p, tag, techID)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "technology reachability unavailable")
		return
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{"country_tag": strings.ToUpper(tag), "tech_id": techID, "paths": paths})
}
