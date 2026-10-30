package guirender

import (
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

var eventFiringKeys = map[string]bool{
	"country_event": true, "news_event": true, "state_event": true,
	"unit_leader_event": true, "operative_leader_event": true,
}

func firedEventIDs(entries []*paradox.Entry, out *[]string) {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		switch {
		case eventFiringKeys[key]:
			id := ""
			if entry.Block != nil {
				id = ciScalar(entry.Block.Entries, "id")
			} else {
				id = toString(scalarValue(entry.Scalar))
			}
			if id = strings.Trim(strings.TrimSpace(id), "\""); id != "" {
				*out = append(*out, id)
			}
			continue
		case key == "random_events" && entry.Block != nil:
			for _, choice := range entry.Block.Entries {
				if choice == nil || choice.Block != nil {
					continue
				}
				if id := strings.Trim(toString(scalarValue(choice.Scalar)), "\" "); strings.Contains(id, ".") {
					*out = append(*out, id)
				}
			}
			continue
		}
		if entry.Block != nil {
			firedEventIDs(entry.Block.Entries, out)
		}
	}
}

func collectStrictTags(entries []*paradox.Entry, allow, deny *[]string) {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		if entry.Block != nil {
			switch key {
			case "and":
				collectStrictTags(entry.Block.Entries, allow, deny)
			case "or":
				if tags, ok := onlyTags(entry.Block.Entries); ok {
					*allow = append(*allow, tags...)
				}
			case "not", "nor":
				if tags, ok := onlyTags(entry.Block.Entries); ok {
					*deny = append(*deny, tags...)
				}
			}
			continue
		}
		if tag, ok := literalTag(entry); ok {
			*allow = append(*allow, tag)
		}
	}
}

func onlyTags(entries []*paradox.Entry) ([]string, bool) {
	var tags []string
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		if entry.Block != nil {
			if !strings.EqualFold(*entry.Key, "or") {
				return nil, false
			}
			inner, ok := onlyTags(entry.Block.Entries)
			if !ok {
				return nil, false
			}
			tags = append(tags, inner...)
			continue
		}
		tag, ok := literalTag(entry)
		if !ok {
			return nil, false
		}
		tags = append(tags, tag)
	}
	return tags, len(tags) > 0
}

func literalTag(entry *paradox.Entry) (string, bool) {
	key := strings.ToLower(*entry.Key)
	if key != "tag" && key != "original_tag" {
		return "", false
	}
	tag := strings.ToUpper(strings.Trim(toString(scalarValue(entry.Scalar)), "\" "))
	if len(tag) != 3 || strings.ContainsAny(tag, ":.") {
		return "", false
	}
	return tag, true
}

func strictTagAllowed(block *paradox.Block, tag string) bool {
	if block == nil {
		return true
	}
	var allow, deny []string
	collectStrictTags(block.Entries, &allow, &deny)
	return decisionTagAllowed(allow, deny, tag)
}

type EventOwnership struct {
	Specific, Own, Owned map[string]bool
}

func (o EventOwnership) Group(id string) string {
	switch {
	case o.Specific[id]:
		return "country"
	case o.Own[id] || !o.Owned[id]:
		return "generic"
	}
	return ""
}

func (idx *Index) EventOwnership(tag string) EventOwnership {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	events := map[string]*eventRecord{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for id, record := range layer.events {
			if _, ok := events[id]; !ok {
				events[id] = record
			}
		}
	}
	reach := func(seeds []string) map[string]bool {
		seen := map[string]bool{}
		queue := append([]string(nil), seeds...)
		for len(queue) > 0 {
			id := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			record, ok := events[id]
			if !ok || seen[id] {
				continue
			}
			seen[id] = true
			queue = append(queue, record.Fires...)
		}
		return seen
	}

	var specific, common, all []string

	if tag != "" {
		for _, choice := range idx.FocusTreeChoices(tag) {
			seeds := &common
			if record := idx.focusTreeRecord(choice.ID); record != nil {
				if _, named := idx.focusTreeScoresOf(record)[tag]; named {
					seeds = &specific
				}
			}
			for _, focus := range idx.focusTreeFocuses(choice.ID) {
				firedEventIDs(idx.focusRewardBlock(focus.ID), seeds)
			}
		}
	}
	seenFocus := map[string]bool{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for id, block := range layer.focusRewardBlocks {
			if !seenFocus[id] {
				seenFocus[id] = true
				firedEventIDs(block, &all)
			}
		}
	}

	for _, category := range idx.DecisionCategories() {
		categoryHas := tag != "" && decisionTagAllowed(category.Allow, category.Deny, tag) && strictTagAllowed(category.Visible, tag)
		categoryNames := namesTag(category.Allow, tag) || strictTagNamed(category.Visible, tag)
		for _, decision := range idx.DecisionsIn(category.Name) {
			all = append(all, decision.Fires...)
			if !categoryHas || !decisionTagAllowed(decision.Allow, decision.Deny, tag) || !strictTagAllowed(decision.Visible, tag) {
				continue
			}
			if categoryNames || namesTag(decision.Allow, tag) || strictTagNamed(decision.Visible, tag) {
				specific = append(specific, decision.Fires...)
			} else {
				common = append(common, decision.Fires...)
			}
		}
	}

	for id, record := range events {
		if len(record.TriggerAllow) == 0 {
			continue
		}
		all = append(all, id)
		if tag != "" && decisionTagAllowed(record.TriggerAllow, record.TriggerDeny, tag) {
			specific = append(specific, id)
		}
	}
	return EventOwnership{
		Specific: reach(specific),
		Own:      reach(append(append([]string(nil), specific...), common...)),
		Owned:    reach(all),
	}
}

func namesTag(tags []string, tag string) bool {
	for _, named := range tags {
		if strings.EqualFold(named, tag) {
			return true
		}
	}
	return false
}

func strictTagNamed(block *paradox.Block, tag string) bool {
	if block == nil {
		return false
	}
	var allow, deny []string
	collectStrictTags(block.Entries, &allow, &deny)
	return namesTag(allow, tag)
}

func (idx *Index) focusTreeRecord(id string) *focusTreeRecord {
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.focusTrees[id]; ok {
			return record
		}
	}
	return nil
}

func (idx *Index) focusTreeFocuses(id string) []*Focus {
	record := idx.focusTreeRecord(id)
	if record == nil {
		return nil
	}
	return append(append([]*Focus(nil), record.focuses...), idx.expandSharedFocuses(record.shared)...)
}
