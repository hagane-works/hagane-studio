package importassets

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/project"
)

type Target struct {
	Slot       string
	Kind       string
	GFXName    string
	EntityID   int64
	Texture    string
	OutputName string
}

type SavedMapping struct {
	Target Target
	OBSKey string
}

var unsafeFilename = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

func BuildPlan(p *project.Project, gfx map[string]string) (map[string][]Target, string, error) {
	uuid, _ := p.Record()["uuid"].(string)
	type source struct {
		slot, table, column string
		perEntity           bool
	}
	sources := []source{
		{"idea_picture", "ideas", "picture", true},
		{"event_picture", "events", "picture", true},
		{"decision_icon", "decisions", "icon", true},
		{"decision_category_icon", "decision_categories", "icon", true},
		{"bookmark_picture", "bookmarks", "picture", true},
		{"tech_icon", "technologies", "tech_id", true},
		{"character_portrait", "characters", "large_portrait", true},
		{"character_advisor", "characters", "small_portrait", true},
	}
	type ref struct {
		slot     string
		entityID int64
		value    string
	}
	var refs []ref
	nodes, err := p.Rows("focus_nodes")
	if err != nil {
		return nil, "", err
	}
	for _, node := range nodes {
		if icon, _ := node["icon"].(string); icon != "" {
			refs = append(refs, ref{"focus_icon", 0, icon})
		}
	}
	for _, src := range sources {
		rows, err := p.Rows(src.table)
		if err != nil {
			return nil, "", err
		}
		for _, row := range rows {
			if value, _ := row[src.column].(string); value != "" {
				refs = append(refs, ref{src.slot, project.Int64(row["id"]), value})
			}
		}
	}
	plan := map[string][]Target{}
	seen := map[string]bool{}
	for _, r := range refs {
		gfxName, texture := resolveReference(r.slot, r.value, gfx)
		if texture == "" {
			continue
		}
		target := Target{Slot: r.slot, Kind: targetKind(r.slot), GFXName: targetGFXKey(r.slot, r.value, gfxName), EntityID: r.entityID, Texture: texture}
		target.OutputName = safePNGName(target.GFXName)
		identity := targetIdentity(target)
		if seen[identity] {
			continue
		}
		seen[identity] = true
		for _, key := range pathMatchKeys(texture) {
			plan[key] = append(plan[key], target)
		}
	}
	for key := range plan {
		sort.Slice(plan[key], func(i, j int) bool {
			return plan[key][i].Slot+plan[key][i].GFXName < plan[key][j].Slot+plan[key][j].GFXName
		})
	}
	return plan, uuid, nil
}

func ResolveSprite(slot, ref string, gfx map[string]string) string {
	_, texture := resolveReference(slot, ref, gfx)
	return texture
}

func resolveReference(slot, ref string, gfx map[string]string) (string, string) {
	if strings.HasPrefix(strings.ToLower(strings.ReplaceAll(ref, "\\", "/")), "gfx/") {
		return ref, strings.ReplaceAll(ref, "\\", "/")
	}
	var candidates []string
	switch slot {
	case "focus_icon":
		candidates = []string{ref}
	case "event_picture":
		if strings.HasPrefix(ref, "GFX_") {
			candidates = []string{ref}
		}
	case "idea_picture":
		if strings.HasPrefix(ref, "GFX_") {
			candidates = []string{ref}
		} else {
			candidates = []string{"GFX_" + ref, "GFX_idea_" + ref}
		}
	case "decision_icon":
		if strings.HasPrefix(ref, "GFX_") {
			candidates = []string{ref}
		} else {
			candidates = []string{"GFX_decision_" + ref}
		}
	case "decision_category_icon":
		if strings.HasPrefix(ref, "GFX_") {
			candidates = []string{ref}
		} else {
			candidates = []string{"GFX_decision_category_" + ref, "GFX_" + ref}
		}
	case "tech_icon":
		candidates = []string{"GFX_" + ref + "_medium"}
	default:
		if strings.HasPrefix(ref, "GFX_") {
			candidates = []string{ref}
		} else {
			candidates = []string{"GFX_" + ref}
		}
	}
	for _, candidate := range candidates {
		if texture := gfx[candidate]; texture != "" {
			return candidate, texture
		}
	}
	return "", ""
}

func targetKind(slot string) string {
	switch slot {
	case "focus_icon":
		return "focus"
	case "idea_picture":
		return "idea"
	case "event_picture":
		return "event"
	case "decision_icon":
		return "decision"
	case "decision_category_icon":
		return "decision-category"
	case "bookmark_picture":
		return "bookmark"
	case "tech_icon":
		return "tech"
	case "character_portrait", "character_advisor":
		return "character"
	default:
		return "import"
	}
}

func targetGFXKey(slot, ref, resolved string) string {
	if slot == "idea_picture" {
		if strings.HasPrefix(ref, "GFX_") {
			return ref
		}
		return "GFX_" + ref
	}
	if slot == "decision_icon" || slot == "decision_category_icon" || slot == "event_picture" || slot == "bookmark_picture" {
		return ref
	}
	if slot == "tech_icon" {
		return ref
	}
	return resolved

}

func safePNGName(value string) string {
	value = unsafeFilename.ReplaceAllString(strings.TrimSpace(value), "_")
	value = strings.Trim(value, "._")
	if value == "" {
		value = "image"
	}
	return value + ".png"
}

func cleanImagePath(value string) string {
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.ReplaceAll(value, "\\", "/")))), "./")
}

func pathMatchKeys(value string) []string {
	normalized := strings.ToLower(cleanImagePath(value))
	ext := filepath.Ext(normalized)
	stem := strings.TrimSuffix(normalized, ext)
	if stem == normalized {
		return []string{normalized}
	}
	return []string{normalized, stem}
}

func MatchTargets(plan map[string][]Target, relative string) []Target {
	keys := pathMatchKeys(relative)
	seen := map[string]bool{}
	var out []Target
	for _, key := range keys {
		for _, target := range plan[key] {
			id := targetIdentity(target)
			if !seen[id] {
				seen[id] = true
				out = append(out, target)
			}
		}
	}
	return out
}

func AssignTargets(plan map[string][]Target, relatives []string) [][]Target {
	type pick struct{ index, rank int }
	matched := make([][]Target, len(relatives))
	best := map[string]pick{}
	for i, relative := range relatives {
		matched[i] = MatchTargets(plan, relative)
		for _, target := range matched[i] {
			id := targetIdentity(target)
			rank := matchRank(relative, target.Texture)
			current, seen := best[id]
			if !seen || rank > current.rank || (rank == current.rank && relative < relatives[current.index]) {
				best[id] = pick{index: i, rank: rank}
			}
		}
	}
	assigned := make([][]Target, len(relatives))
	for i, targets := range matched {
		for _, target := range targets {
			if best[targetIdentity(target)].index == i {
				assigned[i] = append(assigned[i], target)
			}
		}
	}
	return assigned
}

func matchRank(relative, texture string) int {
	file, want := cleanImagePath(relative), cleanImagePath(texture)
	switch {
	case file == want:
		return 3
	case strings.EqualFold(file, want):
		return 2
	}
	return 1
}

func targetIdentity(target Target) string {
	if target.Slot == "tech_icon" || strings.HasPrefix(target.Slot, "character_") {
		return fmt.Sprintf("%s|%d", target.Slot, target.EntityID)
	}
	return target.Slot + "|" + target.GFXName
}

func PersistMappings(tx *project.Tx, mappings []SavedMapping) {
	if len(mappings) == 0 {
		return
	}
	sorted := append([]SavedMapping(nil), mappings...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].OBSKey < sorted[j].OBSKey })
	firstPer := func(key func(SavedMapping) (string, bool)) []SavedMapping {
		seen := map[string]bool{}
		var out []SavedMapping
		for _, m := range sorted {
			k, ok := key(m)
			if ok && !seen[k] {
				seen[k] = true
				out = append(out, m)
			}
		}
		return out
	}
	upsert := func(table string, match func(project.Row) bool, row project.Row, update func(project.Row)) {
		for _, existing := range tx.Rows(table) {
			if match(existing) {
				_, _ = tx.Update(table, project.Int64(existing["id"]), update)
				return
			}
		}
		tx.Insert(table, row)
	}
	for _, m := range firstPer(func(m SavedMapping) (string, bool) { return m.Target.GFXName, m.Target.Slot == "focus_icon" }) {
		upsert("project_obs_icons", func(r project.Row) bool { return r["gfx_name"] == m.Target.GFXName },
			project.Row{"gfx_name": m.Target.GFXName, "obs_key": m.OBSKey},
			func(r project.Row) { r["obs_key"] = m.OBSKey })
	}
	assetType := func(slot string) string {
		if slot == "decision_category_icon" {
			return "decision_icon"
		}
		return slot
	}
	for _, m := range firstPer(func(m SavedMapping) (string, bool) {
		switch m.Target.Slot {
		case "idea_picture", "event_picture", "decision_icon", "decision_category_icon", "bookmark_picture":
			return assetType(m.Target.Slot) + "\x00" + m.Target.GFXName, true
		}
		return "", false
	}) {
		kind := assetType(m.Target.Slot)
		upsert("project_obs_assets", func(r project.Row) bool { return r["asset_type"] == kind && r["gfx_name"] == m.Target.GFXName },
			project.Row{"asset_type": kind, "gfx_name": m.Target.GFXName, "obs_key": m.OBSKey},
			func(r project.Row) { r["obs_key"] = m.OBSKey })
	}
	for _, m := range firstPer(func(m SavedMapping) (string, bool) {
		switch m.Target.Slot {
		case "tech_icon", "character_portrait", "character_advisor":
			return fmt.Sprintf("%s\x00%d", m.Target.Slot, m.Target.EntityID), true
		}
		return "", false
	}) {
		upsert("entity_obs_paths", func(r project.Row) bool {
			return r["entity_type"] == m.Target.Slot && project.Int64(r["entity_id"]) == m.Target.EntityID
		}, project.Row{"entity_type": m.Target.Slot, "entity_id": m.Target.EntityID, "obs_key": m.OBSKey},
			func(r project.Row) { r["obs_key"] = m.OBSKey })
	}
	for _, m := range mappings {
		switch m.Target.Slot {
		case "tech_icon":
			_, _ = tx.Update("technologies", m.Target.EntityID, func(r project.Row) { r["has_obs"] = true })
		case "character_portrait":
			_, _ = tx.Update("characters", m.Target.EntityID, func(r project.Row) { r["large_portrait"] = m.Target.OutputName; r["has_obs"] = true })
		case "character_advisor":
			_, _ = tx.Update("characters", m.Target.EntityID, func(r project.Row) { r["small_portrait"] = m.Target.OutputName; r["has_obs"] = true })
		}
	}
}
