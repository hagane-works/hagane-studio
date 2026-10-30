package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type eventRecord struct {
	ID   string
	Kind string

	Namespace string
	Title     string
	Desc      string
	Picture   string

	Options       []string
	Hidden        bool
	Major         bool
	TriggeredOnly bool
	FireOnlyOnce  bool

	Fires                     []string
	TriggerAllow, TriggerDeny []string

	Order int
}

func (idx *Index) indexEventFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}

	declared := map[string]string{}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block != nil ||
			!strings.EqualFold(*entry.Key, "add_namespace") {
			continue
		}
		if name := strings.Trim(toString(scalarValue(entry.Scalar)), "\""); name != "" {
			declared[strings.ToLower(name)] = name
		}
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		kind, ok := eventKind(*entry.Key)
		if !ok {
			continue
		}
		fields := entry.Block.Entries
		id := strings.Trim(ciScalar(fields, "id"), "\"")
		if id == "" {
			continue
		}
		if _, taken := idx.events[id]; taken {
			continue
		}
		record := &eventRecord{
			ID:            id,
			Kind:          kind,
			Namespace:     eventNamespace(id, declared),
			Title:         eventTextKey(fields, "title"),
			Desc:          eventTextKey(fields, "desc"),
			Picture:       strings.Trim(ciScalar(fields, "picture"), "\""),
			Options:       eventOptionKeys(fields),
			Hidden:        eventFlag(fields, "hidden"),
			Major:         eventFlag(fields, "major"),
			TriggeredOnly: eventFlag(fields, "is_triggered_only"),
			FireOnlyOnce:  eventFlag(fields, "fire_only_once"),
			Order:         len(idx.events),
		}
		firedEventIDs(fields, &record.Fires)
		for _, field := range fields {
			if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, "trigger") {
				collectStrictTags(field.Block.Entries, &record.TriggerAllow, &record.TriggerDeny)
			}
		}
		idx.events[id] = record
	}
}

func eventKind(key string) (string, bool) {
	lower := strings.ToLower(key)
	if !strings.HasSuffix(lower, "_event") || lower == "_event" {
		return "", false
	}
	return strings.TrimSuffix(lower, "_event"), true
}

func eventNamespace(id string, declared map[string]string) string {
	at := strings.LastIndex(id, ".")
	if at <= 0 {
		return ""
	}
	prefix := id[:at]
	if canonical, ok := declared[strings.ToLower(prefix)]; ok {
		return canonical
	}
	return prefix
}

func eventTextKey(entries []*paradox.Entry, field string) string {
	first := ""
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || !strings.EqualFold(*entry.Key, field) {
			continue
		}
		key, decided := "", true
		if entry.Block == nil {
			key = strings.Trim(toString(scalarValue(entry.Scalar)), "\"")
		} else {
			key = strings.Trim(ciScalar(entry.Block.Entries, "text"), "\"")
			for _, child := range entry.Block.Entries {
				if child == nil || child.Key == nil || child.Block == nil ||
					!strings.EqualFold(*child.Key, "trigger") {
					continue
				}
				holds, known := evalCondition(child.Block.Entries)
				decided = known && holds
				break
			}
		}
		if key == "" {
			continue
		}
		if decided {
			return key
		}
		if first == "" {
			first = key
		}
	}
	return first
}

func eventOptionKeys(entries []*paradox.Entry) []string {
	out := []string{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil ||
			!strings.EqualFold(*entry.Key, "option") {
			continue
		}
		out = append(out, strings.Trim(ciScalar(entry.Block.Entries, "name"), "\""))
	}
	return out
}

func eventFlag(entries []*paradox.Entry, key string) bool {
	return strings.EqualFold(ciScalar(entries, key), "yes")
}

func (idx *Index) eventRecordFor(id string) (*eventRecord, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.events[id]; ok {
			return record, true
		}
	}
	return nil, false
}

func (idx *Index) EventPayload(id string) map[string]any {
	return idx.EventPayloadFor(id, "", "")
}

func (idx *Index) EventPayloadFor(id, root, date string) map[string]any {
	id = strings.TrimSpace(id)
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.events[id]; ok {
			payload := idx.eventPayloadFor(record, root, date)

			payload["origin"] = layerOrigin(layer)
			return payload
		}
	}
	return nil
}

func (idx *Index) ListEvents(namespace string) []map[string]any {
	wanted := strings.ToLower(strings.TrimSpace(namespace))
	rows := []map[string]any{}
	appendLayer := func(layer *Index, shadowedBy map[string]*eventRecord) {
		picked := make([]*eventRecord, 0, len(layer.events))
		for id, record := range layer.events {
			if _, shadowed := shadowedBy[id]; shadowed {
				continue
			}
			if wanted != "" && strings.ToLower(record.Namespace) != wanted {
				continue
			}
			picked = append(picked, record)
		}
		sort.Slice(picked, func(i, j int) bool { return picked[i].Order < picked[j].Order })
		for _, record := range picked {
			rows = append(rows, idx.eventPayload(record))
		}
	}

	seen := map[string]*eventRecord{}
	for layer := idx; layer != nil; layer = layer.fallback {
		appendLayer(layer, seen)
		for id, record := range layer.events {
			if _, ok := seen[id]; !ok {
				seen[id] = record
			}
		}
	}
	return rows
}

type EventChoice struct {
	ID    string `json:"id"`
	Title string `json:"title"`

	Group string `json:"group,omitempty"`
}

func (idx *Index) SearchEvents(q string, limit int) []EventChoice {
	page, _ := idx.SearchEventsPage(q, 0, limit)
	return page
}

func (idx *Index) SearchEventsPage(q string, offset, limit int) ([]EventChoice, int) {
	return idx.searchEventsPage(q, offset, limit, nil)
}

func (idx *Index) SearchCountryEventsPage(q string, offset, limit int, tag string) ([]EventChoice, int) {
	ownership := idx.EventOwnership(tag)
	return idx.searchEventsPage(q, offset, limit, func(id string) (string, bool) {
		group := ownership.Group(id)
		return group, group != ""
	})
}

func (idx *Index) searchEventsPage(q string, offset, limit int, listed func(id string) (string, bool)) ([]EventChoice, int) {
	q = strings.ToLower(strings.TrimSpace(q))
	if limit <= 0 {
		limit = 30
	}
	offset = max(0, offset)
	type hit struct {
		choice EventChoice
		group  int
		rank   int
		order  int
		layer  int
	}
	var hits []hit
	seen := map[string]bool{}
	depth := 0
	for layer := idx; layer != nil; layer = layer.fallback {
		for id, record := range layer.events {
			if seen[id] {
				continue
			}
			seen[id] = true
			group, groupRank := "", 0
			if listed != nil {
				var ok bool
				if group, ok = listed(id); !ok {
					continue
				}
				if group != "country" {
					groupRank = 1
				}
			}
			title, _ := idx.Localise(record.Title)
			lowerID := strings.ToLower(id)
			rank := -1
			switch {
			case q == "" || strings.HasPrefix(lowerID, q):
				rank = 0
			case strings.Contains(lowerID, q):
				rank = 1
			case strings.Contains(strings.ToLower(title), q):
				rank = 2
			}
			if rank >= 0 {
				hits = append(hits, hit{EventChoice{ID: id, Title: title, Group: group}, groupRank, rank, record.Order, depth})
			}
		}
		depth++
	}
	sort.Slice(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		if a.group != b.group {
			return a.group < b.group
		}
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		if a.layer != b.layer {
			return a.layer < b.layer
		}
		if a.order != b.order {
			return a.order < b.order
		}
		return a.choice.ID < b.choice.ID
	})
	total := len(hits)
	hits = hits[min(offset, total):]
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]EventChoice, len(hits))
	for i, h := range hits {
		out[i] = h.choice
	}
	return out, total
}

func (idx *Index) eventPayload(record *eventRecord) map[string]any {
	return idx.eventPayloadFor(record, "", "")
}

func (idx *Index) eventPayloadFor(record *eventRecord, root, date string) map[string]any {
	payload := map[string]any{
		"id":             record.ID,
		"kind":           record.Kind,
		"namespace":      record.Namespace,
		"picture":        idx.eventPicture(record.Picture),
		"options":        idx.eventOptionPayload(record.Options, root, date),
		"hidden":         record.Hidden,
		"major":          record.Major,
		"triggered_only": record.TriggeredOnly,
		"fire_only_once": record.FireOnlyOnce,
	}

	if text, ok := idx.LocaliseFor(record.Title, root, date); ok {
		payload["title"] = text
	}
	if text, ok := idx.LocaliseFor(record.Desc, root, date); ok {
		payload["desc"] = text
	}

	if record.Title != "" {
		payload["title_key"] = record.Title
	}
	if record.Desc != "" {
		payload["desc_key"] = record.Desc
	}
	return payload
}

func (idx *Index) eventOptionPayload(names []string, root, date string) []map[string]any {
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		option := map[string]any{"name": name}
		if label, ok := idx.LocaliseFor(name, root, date); ok {
			option["label"] = label
		}
		out = append(out, option)
	}
	return out
}

func (idx *Index) eventPicture(raw string) string {
	if raw == "" || strings.HasPrefix(raw, "GFX_") {
		return raw
	}
	if _, ok := idx.spriteRecordFor(raw); ok {
		return raw
	}
	prefixed := "GFX_" + raw
	if _, ok := idx.spriteRecordFor(prefixed); ok {
		return prefixed
	}
	return raw
}
