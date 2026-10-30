package guirender

import (
	"math"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type autonomyRecord struct {
	ID         string
	MinFreedom float64

	ManpowerInfluence float64
	HasManpower       bool

	Allowed []*paradox.Entry

	Rule     string
	Modifier string
	Order    int

	File string
	Line int
}

func (idx *Index) indexAutonomyLevel(id, file string, line int, block *paradox.Block) {
	if idx.autonomyLevels == nil {
		idx.autonomyLevels = map[string]*autonomyRecord{}
	}
	idx.autonomySeq++
	record := &autonomyRecord{ID: id, Order: idx.autonomySeq, File: file, Line: line}
	record.MinFreedom, _ = toFloat(ciScalarValue(block.Entries, "min_freedom_level"))
	record.ManpowerInfluence, record.HasManpower = toFloat(ciScalarValue(block.Entries, "manpower_influence"))
	if allowed := childBlock(block.Entries, "allowed"); allowed != nil {
		record.Allowed = append([]*paradox.Entry{}, allowed.Entries...)
	}
	if rule := childBlock(block.Entries, "rule"); rule != nil {
		record.Rule = conditionScript(rule.Entries)
	}
	if modifier := childBlock(block.Entries, "modifier"); modifier != nil {
		record.Modifier = conditionScript(modifier.Entries)
	}
	idx.autonomyLevels[id] = record
}

func (idx *Index) AutonomySource(id string) (string, []byte, int, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.autonomyLevels[id]; ok && record.File != "" {
			return record.File, layer.files[record.File], record.Line, true
		}
	}
	return "", nil, 0, false
}

func (idx *Index) autonomyLevelPayload(record *autonomyRecord) map[string]any {
	out := map[string]any{
		"id": record.ID, "label": idx.localiseOr(record.ID, record.ID), "file": record.File,
		"min_freedom_level": record.MinFreedom, "rule": record.Rule, "modifier": record.Modifier,
	}
	if record.HasManpower {
		out["manpower_influence"] = record.ManpowerInfluence
	}
	return out
}

func (idx *Index) autonomyLevelList() []*autonomyRecord {
	var layers []*Index
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
	}
	byID := map[string]*autonomyRecord{}
	rank := map[string]int{}
	next := 0
	for at := len(layers) - 1; at >= 0; at-- {
		records := make([]*autonomyRecord, 0, len(layers[at].autonomyLevels))
		for _, record := range layers[at].autonomyLevels {
			records = append(records, record)
		}
		sort.Slice(records, func(a, b int) bool { return records[a].Order < records[b].Order })
		for _, record := range records {
			if _, seen := rank[record.ID]; !seen {
				next++
				rank[record.ID] = next
			}
			byID[record.ID] = record
		}
	}
	out := make([]*autonomyRecord, 0, len(byID))
	for _, record := range byID {
		out = append(out, record)
	}
	sort.Slice(out, func(a, b int) bool { return rank[out[a].ID] < rank[out[b].ID] })
	return out
}

func (idx *Index) overlordOf(tag string) string {
	tag = strings.ToUpper(tag)
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.subjectOf[tag]; ok {
			return record.Overlord
		}
	}
	return ""
}

func (idx *Index) countryOrder(tag string) (int, int) {
	var layers []*Index
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
	}
	for at := len(layers) - 1; at >= 0; at-- {
		if seq, ok := layers[at].countryTagSeq[tag]; ok {
			return len(layers) - 1 - at, seq
		}
	}
	return len(layers), math.MaxInt32
}

func (idx *Index) subjectsOf(tag, date string) map[string]subjectRecord {
	out := map[string]subjectRecord{}
	politics := idx.countryPoliticsFor(tag)
	if politics == nil {
		return out
	}
	for _, call := range effectsRunByOwned(politics.Effects, date, idx.ownedDLC) {
		if call.Command != "set_autonomy" && call.Command != "puppet" {
			continue
		}
		target := strings.ToUpper(call.Token)
		if target == "" || target == tag {
			continue
		}
		for layer := idx; layer != nil; layer = layer.fallback {
			if record, ok := layer.subjectOf[target]; ok {
				if strings.EqualFold(record.Overlord, tag) {
					out[target] = record
				}
				break
			}
		}
	}
	return out
}

func (idx *Index) cosmeticTagOf(tag, date string) string {
	politics := idx.countryPoliticsFor(tag)
	if politics == nil {
		return ""
	}
	cosmetic := ""
	for _, call := range effectsRunByOwned(politics.Effects, date, idx.ownedDLC) {
		switch call.Command {
		case "set_cosmetic_tag":
			cosmetic = call.Token
		case "drop_cosmetic_tag":
			cosmetic = ""
		}
	}
	return cosmetic
}

func (idx *Index) subjectName(tag, cosmetic, level, ideology string) string {
	bases := []string{tag}
	if cosmetic != "" && !strings.EqualFold(cosmetic, tag) {
		bases = []string{cosmetic, tag}
	}
	for _, base := range bases {
		keys := []string{base + "_" + level}
		if ideology != "" {
			keys = append(keys, base+"_"+strings.ToLower(ideology))
		}
		keys = append(keys, base)
		for _, key := range keys {
			if name, ok := idx.Localise(key); ok {
				return name
			}
		}
	}
	return tag
}

func (idx *Index) autonomyStep(id string) map[string]any {
	label, icon := idx.localiseOr(id, id), "GFX_"+id+"_icon"
	switch id {
	case "free":
		label, icon = idx.localiseOr("AUTONOMY_FREE", "Free"), "GFX_autonomy_free_icon"
	case "annex":
		label, icon = idx.localiseOr("AUTONOMY_ANNEXED", "Annexed"), "GFX_autonomy_annex_icon"
	}
	return map[string]any{"id": id, "label": label, "icon": icon}
}

func (idx *Index) SubjectsPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	subjects := idx.subjectsOf(tag, date)
	targets := make([]string, 0, len(subjects))
	for target := range subjects {
		targets = append(targets, target)
	}
	sort.Slice(targets, func(a, b int) bool {
		la, sa := idx.countryOrder(targets[a])
		lb, sb := idx.countryOrder(targets[b])
		if la != lb {
			return la < lb
		}
		if sa != sb {
			return sa < sb
		}
		return targets[a] < targets[b]
	})
	levels := idx.autonomyLevelList()
	world := &decisionWorld{idx: idx, date: date, starts: map[string]*decisionStart{}}
	rows := []map[string]any{}
	for _, target := range targets {
		record := subjects[target]
		level := record.Level
		if level == "" {
			level = idx.defaultAutonomyLevel()
		}
		state := world.start(target)
		allowed := func(candidate *autonomyRecord) bool {
			if candidate.Allowed == nil {
				return true
			}
			value, decided := state.eval(candidate.Allowed, false)
			return decided && value
		}
		var current *autonomyRecord
		for _, candidate := range levels {
			if strings.EqualFold(candidate.ID, level) {
				current = candidate
			}
		}
		down, up := "annex", "free"
		if current != nil {
			var below, above *autonomyRecord
			for _, candidate := range levels {
				if candidate == current || !allowed(candidate) {
					continue
				}
				if candidate.MinFreedom < current.MinFreedom && (below == nil || candidate.MinFreedom > below.MinFreedom) {
					below = candidate
				}
				if candidate.MinFreedom > current.MinFreedom && (above == nil || candidate.MinFreedom < above.MinFreedom) {
					above = candidate
				}
			}
			if below != nil {
				down = below.ID
			}
			if above != nil {
				up = above.ID
			}
		}
		party := idx.RulingPartyOf(target)
		if party == "" {

			if groups := idx.IdeologyGroups(); len(groups) > 0 {
				party = strings.ToLower(groups[0].Name)
			}
		}
		cosmetic := idx.cosmeticTagOf(target, date)
		row := map[string]any{
			"tag": target, "name": idx.subjectName(target, cosmetic, level, party),
			"level": idx.autonomyStep(level), "down": idx.autonomyStep(down), "up": idx.autonomyStep(up),
			"freedom": record.Freedom,
		}

		if cosmetic != "" {
			if flag := idx.flagPayload(cosmetic, party, ""); flag != nil {
				row["flag"] = flag
			}
		}
		if row["flag"] == nil {
			if flag := idx.flagPayload(target, party, ""); flag != nil {
				row["flag"] = flag
			}
		}
		rows = append(rows, row)
	}

	icons := []string{}
	seen := map[string]bool{}
	for _, row := range rows {
		for _, key := range []string{"level", "down", "up"} {
			icon := row[key].(map[string]any)["icon"].(string)
			if !seen[icon] {
				seen[icon] = true
				icons = append(icons, icon)
			}
		}
	}

	shown := map[string]any{}
	for _, row := range rows {
		id := row["level"].(map[string]any)["id"].(string)
		for _, candidate := range levels {
			if strings.EqualFold(candidate.ID, id) {
				shown[id] = idx.autonomyLevelPayload(candidate)
			}
		}
	}
	return map[string]any{"tag": tag, "subjects": rows, "sprites": idx.SpritesPayload(icons), "levels": shown}
}
