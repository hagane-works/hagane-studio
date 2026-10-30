package exportcheck

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

type relProject struct {
	countryTag  string
	countryTags []string
	isSubmod    bool
}

func relBestTag(own *string, shared []string, proj relProject) *string {
	if own != nil && *own != "" {
		u := strings.ToUpper(*own)
		return &u
	}
	for _, t := range shared {
		if t != "" {
			u := strings.ToUpper(t)
			return &u
		}
	}
	if proj.countryTag != "" {
		u := strings.ToUpper(proj.countryTag)
		return &u
	}

	var fallback []string
	for _, t := range proj.countryTags {
		if t != "" {
			fallback = append(fallback, strings.ToUpper(t))
		}
	}
	if len(fallback) == 0 {
		return nil
	}
	sort.Strings(fallback)
	return &fallback[0]
}

func relAllTags(own *string, shared []string, proj relProject) []string {
	set := map[string]struct{}{}
	if own != nil && *own != "" {
		set[strings.ToUpper(*own)] = struct{}{}
	}
	for _, t := range shared {
		if t != "" {
			set[strings.ToUpper(t)] = struct{}{}
		}
	}
	if len(set) == 0 && proj.countryTag != "" {
		set[strings.ToUpper(proj.countryTag)] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

type relScriptSource struct {
	kind       string
	sourceID   int64
	sourceName string
	fieldName  string
	text       string
	tag        *string
	tags       []string
}

func scanJSONStrings(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	_ = json.Unmarshal(raw, &out)
	return out
}

type relRowSource struct {
	id       int64
	name     string
	own      *string
	shared   []string
	fields   map[string]string
	fieldSeq []string
}

func relQuerySourcesWith(_ context.Context, db *Store, _ int64, proj relProject) ([]relScriptSource, error) {
	var out []relScriptSource
	emit := func(kind string, r relRowSource) {
		tag := relBestTag(r.own, r.shared, proj)
		tags := relAllTags(r.own, r.shared, proj)
		for _, fld := range r.fieldSeq {
			if text := r.fields[fld]; text != "" {
				out = append(out, relScriptSource{kind, r.id, r.name, fld, text, tag, tags})
			}
		}
	}
	fieldsOf := func(row map[string]any, seq []string) map[string]string {
		fields := map[string]string{}
		for _, name := range seq {
			fields[name] = colStr(row, name)
		}
		return fields
	}

	trees, err := db.rows("focus_trees")
	if err != nil {
		return nil, err
	}
	nodes, err := db.rows("focus_nodes")
	if err != nil {
		return nil, err
	}
	nodesByTree := map[int64][]map[string]any{}
	for _, node := range nodes {
		tid := colInt(node, "focus_tree_id")
		nodesByTree[tid] = append(nodesByTree[tid], node)
	}
	for _, t := range trees {
		own, shared := colStrPtr(t, "country_tag"), scanJSONStrings(colBytes(t, "shared_country_tags"))
		for _, node := range nodesByTree[colInt(t, "id")] {
			seq := []string{"completion_reward", "select_effect", "available", "bypass", "cancel"}
			emit("focus", relRowSource{
				id: colInt(node, "id"), name: colStr(node, "focus_id"), own: own, shared: shared,
				fieldSeq: seq, fields: fieldsOf(node, seq),
			})
		}
	}

	events, err := db.rows("events")
	if err != nil {
		return nil, err
	}
	for _, row := range events {
		id, eventID := colInt(row, "id"), colStr(row, "event_id")
		own, shared := colStrPtr(row, "country_tag"), scanJSONStrings(colBytes(row, "shared_country_tags"))
		seq := []string{"trigger", "immediate", "after", "mean_time_to_happen"}
		emit("event", relRowSource{id: id, name: eventID, own: own, shared: shared, fieldSeq: seq, fields: fieldsOf(row, seq)})
		tag := relBestTag(own, shared, proj)
		tags := relAllTags(own, shared, proj)
		var opts []map[string]any
		_ = json.Unmarshal(colBytes(row, "options"), &opts)
		for idx, o := range opts {
			for _, ofield := range []string{"trigger", "effects", "hidden_effects"} {
				if v, ok := o[ofield]; ok && v != nil {
					if text, ok := v.(string); ok && text != "" {
						out = append(out, relScriptSource{
							"event", id, eventID,
							"option[" + strconv.Itoa(idx) + "]." + ofield, text, tag, tags,
						})
					}
				}
			}
		}
	}

	decisions, err := db.rows("decisions")
	if err != nil {
		return nil, err
	}
	for _, row := range decisions {
		seq := []string{"complete_effect", "remove_effect", "timeout_effect", "cancel_effect",
			"allowed", "available", "visible", "cancel_trigger", "activation", "ai_will_do"}
		emit("decision", relRowSource{
			id: colInt(row, "id"), name: colStr(row, "decision_id"), own: colStrPtr(row, "country_tag"),
			shared: scanJSONStrings(colBytes(row, "shared_country_tags")), fieldSeq: seq, fields: fieldsOf(row, seq),
		})
	}

	ideas, err := db.rows("ideas")
	if err != nil {
		return nil, err
	}
	for _, row := range ideas {
		seq := []string{"on_add", "on_remove", "do_effect", "allowed", "available",
			"visible", "allowed_civil_war", "allowed_to_remove", "cancel"}
		emit("idea", relRowSource{
			id: colInt(row, "id"), name: colStr(row, "idea_id"), own: colStrPtr(row, "country_tag"),
			shared: scanJSONStrings(colBytes(row, "shared_country_tags")), fieldSeq: seq, fields: fieldsOf(row, seq),
		})
	}

	characters, err := db.rows("characters")
	if err != nil {
		return nil, err
	}
	for _, row := range characters {
		name := colStr(row, "tag")
		if name == "" {
			name = colStr(row, "name")
		}
		seq := []string{"advisor_on_add", "advisor_on_remove", "advisor_available",
			"advisor_visible", "advisor_modifier"}
		emit("character", relRowSource{id: colInt(row, "id"), name: name, own: colStrPtr(row, "country_tag"), fieldSeq: seq, fields: fieldsOf(row, seq)})
	}
	return out, nil
}
