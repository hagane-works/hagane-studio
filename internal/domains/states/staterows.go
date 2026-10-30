package states

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/loc"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

const (
	statesTable     = "states"
	locEntriesTable = "project_loc_entries"
)

var errStateMissing = apiutil.NotFoundError("State not found")

var errDryRun = errors.New("dry run")

const stateSnapshotSchema = "hagane.state-source-snapshot.v1"

var provinceIDRE = regexp.MustCompile(`^\d+$`)

func resolveDisplayLocs(names map[string]map[string]string, stateID int64, locs map[string]any) map[string]string {
	out := map[string]string{}
	for lang, value := range names[fmt.Sprintf("%d", stateID)] {
		if v := strings.TrimSpace(value); v != "" {
			out[lang] = v
		}
	}
	for lang, value := range locs {
		if v := strings.TrimSpace(str(value)); v != "" {
			out[lang] = v
		}
	}
	return out
}

func usesVanillaDisplayNames(row map[string]any, snapshot map[string]any) bool {
	if strings.EqualFold(strings.TrimSpace(str(row["source"])), "imported") {
		return true
	}
	if strings.TrimSpace(str(row["source_file"])) != "" {
		return true
	}
	return snapshot != nil
}

func storedDisplayLocs(locs map[string]any) map[string]string {
	out := map[string]string{}
	for lang, value := range locs {
		if v := strings.TrimSpace(str(value)); v != "" {
			out[lang] = v
		}
	}
	return out
}

func stateDisplayNameKey(row map[string]any) string {
	if key := strings.TrimSpace(str(row["name"])); key != "" {
		return key
	}
	return fmt.Sprintf("STATE_%d", coerceInt(row["state_id"]))
}

func projectStateDisplayNames(locRows []project.Row, states []map[string]any) {
	if len(states) == 0 {
		return
	}
	keys := map[string]bool{}
	for _, row := range states {
		keys[stateDisplayNameKey(row)] = true
	}
	type winner struct {
		value                               string
		priority, sourceOrder, entryOrd, id int64
	}
	better := func(a, b winner) bool {
		if a.priority != b.priority {
			return a.priority > b.priority
		}
		if a.sourceOrder != b.sourceOrder {
			return a.sourceOrder > b.sourceOrder
		}
		if a.entryOrd != b.entryOrd {
			return a.entryOrd > b.entryOrd
		}
		return a.id > b.id
	}
	best := map[[2]string]winner{}
	for _, entry := range locRows {
		key := str(entry["key"])
		if !keys[key] {
			continue
		}
		if _, ok := exportengine.SafePreservedLocSourceFile(str(entry["source_file"])); !ok {
			continue
		}
		w := winner{
			value: str(entry["value"]), priority: int64(coerceInt(entry["replace_priority"])),
			sourceOrder: int64(coerceInt(entry["source_order"])), entryOrd: int64(coerceInt(entry["entry_order"])),
			id: project.Int64(entry["id"]),
		}
		slot := [2]string{key, str(entry["locale"])}
		if prior, ok := best[slot]; !ok || better(w, prior) {
			best[slot] = w
		}
	}
	byKey := map[string]map[string]string{}
	for slot, w := range best {
		lang, ok := loc.LanguageToSuffix(slot[1])
		if !ok {
			continue
		}
		if byKey[slot[0]] == nil {
			byKey[slot[0]] = map[string]string{}
		}
		byKey[slot[0]][lang] = strings.TrimSpace(w.value)
	}
	for _, row := range states {
		display, _ := row["display_name"].(map[string]string)
		if display == nil {
			display = map[string]string{}
		}
		for lang, value := range byKey[stateDisplayNameKey(row)] {

			delete(display, lang)
			if value != "" {
				display[lang] = value
			}
		}
		overrides, _ := row["display_name_locs"].(map[string]any)
		for lang, value := range storedDisplayLocs(overrides) {
			display[lang] = value
		}
		row["display_name"] = display
	}
}

func isSimpleProvinceBlock(entries []*paradox.Entry) bool {
	for _, sub := range entries {
		if sub.Key == nil || sub.Block != nil {
			return false
		}
	}
	return true
}

func deserializeProvinceBuildings(text string) map[string]any {
	text = strings.TrimSpace(text)
	out := map[string]any{}
	if text == "" {
		return out
	}
	block, _, err := paradox.Parse(text)
	if err != nil {
		return out
	}
	for _, ent := range block.Entries {
		if ent.Key == nil || ent.Block == nil {
			continue
		}
		key := strings.TrimSpace(*ent.Key)
		if !provinceIDRE.MatchString(key) || !isSimpleProvinceBlock(ent.Block.Entries) {
			continue
		}
		levels := map[string]any{}
		for _, sub := range ent.Block.Entries {
			levels[*sub.Key] = coerceInt(sub.Scalar.StrValue())
		}
		out[key] = levels
	}
	return out
}

func serializeProvinceBuildings(pb map[string]any, pbOrder []string, existing string) string {
	var preserved []string
	if text := strings.TrimSpace(existing); text != "" {
		block, recoveries, err := paradox.Parse(text)
		if err != nil || len(recoveries) > 0 {
			preserved = append(preserved, text)
			block = nil
		}
		if block != nil {
			for _, ent := range block.Entries {
				if ent.Key != nil && ent.Block != nil &&
					provinceIDRE.MatchString(strings.TrimSpace(*ent.Key)) &&
					isSimpleProvinceBlock(ent.Block.Entries) {
					continue
				}
				preserved = append(preserved, strings.TrimRight(importsvc.ToScriptEntries([]*paradox.Entry{ent}, 0), "\n"))
			}
		}
	}
	var structured []string
	for _, pid := range pbOrder {
		levels, ok := pb[pid].(map[string]any)
		if !ok {
			continue
		}
		names := make([]string, 0, len(levels))
		for name := range levels {
			names = append(names, name)
		}
		sort.Strings(names)
		var lines []string
		for _, name := range names {
			if strings.TrimSpace(name) == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("\t%s = %d", name, coerceInt(levels[name])))
		}
		if len(lines) == 0 {
			continue
		}
		structured = append(structured, pid+" = {\n"+strings.Join(lines, "\n")+"\n}")
	}
	return strings.Join(append(structured, preserved...), "\n")
}

func normalizeProvinceIDs(values []any) ([]int, error) {
	set := map[int]bool{}
	for _, v := range values {
		switch t := v.(type) {
		case string:
			var n int
			if _, err := fmt.Sscanf(strings.TrimSpace(t), "%d", &n); err != nil || n <= 0 {
				return nil, fmt.Errorf("province ids must be positive integers")
			}
			set[n] = true
		default:
			n, ok := whole(v)
			if _, isBool := v.(bool); isBool || !ok || n <= 0 {
				return nil, fmt.Errorf("province ids must be positive integers")
			}
			set[int(n)] = true
		}
	}
	out := make([]int, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Ints(out)
	return out, nil
}

func nullableText(v any) any {
	if s := str(v); s != "" {
		return s
	}
	return nil
}

func captureStateSourceSnapshot(stateID int64, provinces []any, owner, controller, sourcePath any) (map[string]any, error) {
	ids, err := normalizeProvinceIDs(provinces)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"schema_version":        stateSnapshotSchema,
		"state_id":              stateID,
		"province_ids":          ids,
		"owner":                 nullableText(owner),
		"controller":            nullableText(controller),
		"source_path":           nullableText(sourcePath),
		"source_line":           nil,
		"source_content_sha256": nil,
	}, nil
}

func normalizeStateSourceSnapshot(value map[string]any) map[string]any {
	if value == nil || str(value["schema_version"]) != stateSnapshotSchema {
		return nil
	}
	stateID, ok := num(value["state_id"])
	if !ok {
		return nil
	}
	ids, _ := value["province_ids"].([]any)
	snapshot, err := captureStateSourceSnapshot(int64(stateID), ids, value["owner"], value["controller"], value["source_path"])
	if err != nil {
		return nil
	}
	snapshot["source_line"] = value["source_line"]
	snapshot["source_content_sha256"] = value["source_content_sha256"]
	return snapshot
}

func membershipProvenance(provinces []any, owner, controller string, sourceSnapshot map[string]any) map[string]any {
	current, err := normalizeProvinceIDs(provinces)
	if err != nil {
		current = []int{}
	}
	snapshot := normalizeStateSourceSnapshot(sourceSnapshot)
	if snapshot == nil {
		return map[string]any{
			"status": "legacy_unknown", "source": nil, "current": current,
			"added": []int{}, "removed": []int{}, "changed": []int{},
			"membership_changed": nil, "owner_changed": nil, "controller_changed": nil,
		}
	}
	source := snapshot["province_ids"].([]int)
	inSource, inCurrent := map[int]bool{}, map[int]bool{}
	for _, id := range source {
		inSource[id] = true
	}
	for _, id := range current {
		inCurrent[id] = true
	}
	added, removed := []int{}, []int{}
	for _, id := range current {
		if !inSource[id] {
			added = append(added, id)
		}
	}
	for _, id := range source {
		if !inCurrent[id] {
			removed = append(removed, id)
		}
	}
	changed := append(append([]int{}, added...), removed...)
	sort.Ints(changed)
	return map[string]any{
		"status": "known", "source": source, "current": current,
		"added": added, "removed": removed, "changed": changed,
		"membership_changed": len(changed) > 0,
		"owner_changed":      snapshot["owner"] != nullableText(owner),
		"controller_changed": snapshot["controller"] != nullableText(controller),
	}
}

func stateRead(row project.Row, names map[string]map[string]string) map[string]any {
	m := cloneRow(row)
	buildings := ""
	if extras, ok := m["raw_extras"].(map[string]any); ok {
		buildings = str(extras["buildings"])
	}
	m["province_buildings"] = deserializeProvinceBuildings(buildings)
	stateID, _ := num(m["state_id"])
	snapshotRaw, _ := m["source_snapshot"].(map[string]any)
	normalized := normalizeStateSourceSnapshot(snapshotRaw)
	locs, _ := m["display_name_locs"].(map[string]any)
	if usesVanillaDisplayNames(m, normalized) {
		m["display_name"] = resolveDisplayLocs(names, int64(stateID), locs)
	} else {
		m["display_name"] = storedDisplayLocs(locs)
	}
	if normalized != nil {
		m["source_snapshot"] = normalized
	} else {
		m["source_snapshot"] = nil
	}
	provinces, _ := m["provinces"].([]any)
	m["membership_provenance"] = membershipProvenance(provinces, str(m["owner"]), str(m["controller"]), snapshotRaw)
	return m
}

var stateUpdateColumns = []string{
	"display_name_locs", "manpower", "state_category", "resources", "provinces",
	"owner", "controller", "add_core_of", "add_claim_by", "remove_core_of",
	"victory_points", "set_province_controller", "buildings", "local_supplies",
	"buildings_max_level_factor", "impassable", "impassable_ignored_links",
	"force_link_ownership_to",
}

func validateStateProvinces(value any) ([]any, error) {
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("provinces must be a list of positive integers")
	}
	seen := map[int64]bool{}
	ids := []int64{}
	for _, item := range list {
		n, ok := whole(item)
		if _, isBool := item.(bool); isBool || !ok || n <= 0 {
			return nil, fmt.Errorf("provinces must contain positive integers")
		}
		if seen[n] {
			return nil, fmt.Errorf("provinces must not contain duplicate ids")
		}
		seen[n] = true
		ids = append(ids, n)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := make([]any, len(ids))
	for i, n := range ids {
		out[i] = n
	}
	return out, nil
}

func updateStateTx(tx *project.Tx, id int64, in map[string]any, pbOrder []string) error {
	cur := tx.Find(statesTable, id)
	if cur == nil {
		return errStateMissing
	}
	if v, present := in["provinces"]; present && v != nil {
		normalized, err := validateStateProvinces(v)
		if err != nil {
			return detailError(422, err.Error())
		}
		in["provinces"] = normalized
	}
	forceLinkText := ""
	if v, present := in["force_link_ownership_to"]; present && v != nil {
		n, ok := whole(v)
		if _, isBool := v.(bool); isBool || !ok || n < -2147483648 || n > 2147483647 {
			return detailError(422, "force_link_ownership_to must be an integer state id or null")
		}
		in["force_link_ownership_to"] = n
		forceLinkText = fmt.Sprintf("%d", n)
	}

	changes := map[string]any{}
	stateID, _ := whole(cur["state_id"])
	curSnapshot, _ := cur["source_snapshot"].(map[string]any)
	_, provincesTouched := in["provinces"]
	if normalizeStateSourceSnapshot(curSnapshot) == nil && (str(cur["source"]) == "imported" || provincesTouched) {
		provinces, _ := cur["provinces"].([]any)
		if snapshot, err := captureStateSourceSnapshot(stateID, provinces, cur["owner"], cur["controller"], cur["source_file"]); err == nil {
			changes["source_snapshot"] = plain(snapshot)
		}
	}

	extras := map[string]any{}
	if current, ok := cur["raw_extras"].(map[string]any); ok {
		for k, v := range current {
			extras[k] = v
		}
	}
	extrasTouched := false
	if raw, present := in["province_buildings"]; present {
		pb, _ := raw.(map[string]any)
		if text := serializeProvinceBuildings(pb, pbOrder, str(extras["buildings"])); text != "" {
			extras["buildings"] = text
		} else {
			delete(extras, "buildings")
		}
		extrasTouched = true
		delete(in, "province_buildings")
	}
	if _, touched := in["impassable_ignored_links"]; touched {
		if _, has := extras["impassable_ignored_links"]; has {
			delete(extras, "impassable_ignored_links")
			extrasTouched = true
		}
	}
	if v, touched := in["force_link_ownership_to"]; touched {
		if v == nil {
			if _, has := extras["force_link_ownership_to"]; has {
				delete(extras, "force_link_ownership_to")
				extrasTouched = true
			}
		} else if str(extras["force_link_ownership_to"]) != forceLinkText {
			extras["force_link_ownership_to"] = forceLinkText
			extrasTouched = true
		}
	}
	if extrasTouched {
		changes["raw_extras"] = plain(extras)
	}

	for _, column := range stateUpdateColumns {
		value, present := in[column]
		if !present {
			continue
		}
		if value == nil && (column == "owner" || column == "state_category") {
			value = ""
		}
		if value == nil && column == "display_name_locs" {
			value = map[string]any{}
		}
		if s, ok := value.(string); ok && (column == "owner" || column == "controller" || column == "state_category") {
			value = strings.TrimSpace(s)
		}
		stored, err := columnValue(statesTable, column, value)
		if err != nil {
			return detailError(422, "Invalid state payload")
		}
		changes[column] = stored
	}
	flip := str(cur["source"]) == "imported"
	if len(changes) == 0 && !flip {
		return nil
	}
	_, err := tx.Update(statesTable, id, func(row project.Row) {
		for column, value := range changes {
			row[column] = value
		}
		if flip {
			row["source"] = "created"
		}
	})
	return err
}
