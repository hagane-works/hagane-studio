package exportmerge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
	emitpreview "haganeworks/internal/hoi4/emit/preview"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/exportmsg"
)

var diffEntityTypes = map[string]bool{
	"character": true, "event": true, "decision": true, "focus_node": true, "idea": true,
}

func ApplyOpsToArchive(ctx context.Context, archive Archive, target *TargetMod, final map[string][]DiffOp, primaryTag, destination, fingerprint string) (exportengine.Manifest, error) {
	return applyOpsToArchive(ctx, archive, target, final, primaryTag, destination, fingerprint, nil, false, nil, exportmsg.Printer{})
}

func ApplyOpsToArchiveWithFindings(ctx context.Context, archive Archive, target *TargetMod, final map[string][]DiffOp, primaryTag, destination, fingerprint string, findings []exportengine.ExportFinding, msg exportmsg.Printer) (exportengine.Manifest, error) {
	return applyOpsToArchive(ctx, archive, target, final, primaryTag, destination, fingerprint, findings, true, nil, msg)
}

func ApplyOpsToArchiveWithFindingsAndSnapshot(ctx context.Context, archive Archive, target *TargetMod, final map[string][]DiffOp, primaryTag, destination, fingerprint string, findings []exportengine.ExportFinding, snapshot exportengine.Snapshot, msg exportmsg.Printer) (exportengine.Manifest, error) {
	return applyOpsToArchive(ctx, archive, target, final, primaryTag, destination, fingerprint, findings, true, &snapshot, msg)
}

func applyOpsToArchive(ctx context.Context, archive Archive, target *TargetMod, final map[string][]DiffOp, primaryTag, destination, fingerprint string, findings []exportengine.ExportFinding, manageFindings bool, snapshot *exportengine.Snapshot, msg exportmsg.Printer) (exportengine.Manifest, error) {
	if err := ctx.Err(); err != nil {
		return exportengine.Manifest{}, err
	}
	if target == nil {
		target = ParseTarget(ctx, archive)
	}
	primaryTag = strings.ToUpper(strings.TrimSpace(primaryTag))
	if primaryTag == "" {
		primaryTag = "UNK"
	}

	resolved := map[string]map[string]any{}
	for _, key := range sortedFinalKeys(final) {
		kind, id, ok := strings.Cut(key, "/")
		if !ok || id == "" || !diffEntityTypes[kind] {
			return exportengine.Manifest{}, fmt.Errorf("malformed resolved entity key %q", key)
		}
		base := target.ForType(kind)[id]
		if base == nil {
			base = map[string]any{}
		}
		patched, err := ApplyDiff(base, final[key])
		if err != nil {
			return exportengine.Manifest{}, fmt.Errorf("apply %s: %w", key, err)
		}
		resolved[key] = patched
	}

	if err := ValidateFocusOverrides(target.focusSources, resolved); err != nil {
		return exportengine.Manifest{}, err
	}
	fileOverrides := map[string]map[string]map[string]any{}
	for key, row := range resolved {
		kind, id, _ := strings.Cut(key, "/")
		source := target.SourceFiles[key]
		if source == "" {
			source = existingGroupFile(target, kind, row)
		}
		if source == "" {
			source = defaultEntityPath(kind, id, row, primaryTag)
		}
		if kind == "focus_node" && target.focusSources[source] == nil {
			for _, entry := range archive.Entries {
				if strings.EqualFold(entry.Path, source) {
					return exportengine.Manifest{}, fmt.Errorf("cannot address focus in existing unparsed file %q", source)
				}
			}
		}
		if fileOverrides[source] == nil {
			fileOverrides[source] = map[string]map[string]any{}
		}
		fileOverrides[source][key] = row
	}

	entryMap := make(map[string]exportengine.Entry, len(archive.Entries)+len(fileOverrides))
	for _, entry := range ArchiveEntries(archive, true) {
		if manageFindings && strings.EqualFold(path.Base(entry.Path), exportengine.ExportFindingsBasename) {
			continue
		}
		entryMap[entry.Path] = entry
	}
	paths := make([]string, 0, len(fileOverrides))
	for source := range fileOverrides {
		paths = append(paths, source)
	}
	sort.Strings(paths)
	for _, source := range paths {
		if err := ctx.Err(); err != nil {
			return exportengine.Manifest{}, err
		}
		text, err := emitTargetFile(target, source, fileOverrides[source], primaryTag)
		if err != nil {
			return exportengine.Manifest{}, err
		}
		entryMap[source] = exportengine.Entry{Path: source, Bytes: []byte(text), Origin: "differential_generated", SourceEntity: "target_file:" + source}
	}
	mergeLocalisations(archive, entryMap, resolved)

	entries := make([]exportengine.Entry, 0, len(entryMap))
	for _, entry := range entryMap {
		entries = append(entries, entry)
	}
	if manageFindings {
		entries, findings = exportengine.RefreshFinalArtifactFindings(entries, findings, snapshot, msg)
		report, err := exportengine.ExportFindingsEntry(entries, findings)
		if err != nil {
			return exportengine.Manifest{}, err
		}
		entries = append(entries, report)
	}
	if fingerprint == "" {
		fingerprint = mergeFingerprint(archive, final)
		if manageFindings {
			fingerprint = mergeFindingsFingerprint(fingerprint, findings)
		}
	}
	if err := ctx.Err(); err != nil {
		return exportengine.Manifest{}, err
	}
	manifest, err := exportengine.BuildContext(ctx, destination, fingerprint, entries, exportengine.DefaultArtifactBudget())
	if err != nil {
		return exportengine.Manifest{}, err
	}
	if err := ctx.Err(); err != nil {
		return exportengine.Manifest{}, err
	}
	return manifest, nil
}

func mergeFindingsFingerprint(base string, findings []exportengine.ExportFinding) string {
	h := sha256.New()
	_, _ = h.Write([]byte(base))
	doc, err := exportengine.NewExportFindingsDocument(findings)
	if err == nil {
		raw, _ := json.Marshal(doc)
		_, _ = h.Write(raw)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func mergeFingerprint(archive Archive, final map[string][]DiffOp) string {
	h := sha256.New()
	for _, entry := range archive.Entries {
		_, _ = h.Write([]byte(entry.Path))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(entry.Bytes)
	}
	for _, key := range sortedFinalKeys(final) {
		_, _ = h.Write([]byte(key))
		raw, _ := json.Marshal(final[key])
		_, _ = h.Write(raw)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func existingGroupFile(target *TargetMod, kind string, row map[string]any) string {
	switch kind {
	case "event":
		ns := asString(row["namespace"])
		if ns == "" {
			ns = eventNamespace(asString(row["event_id"]))
		}
		for id, otherNS := range target.EventNamespaces {
			if otherNS == ns {
				return target.SourceFiles[entityKey("event", id)]
			}
		}
	case "decision":
		category := asString(row["category_id"])
		for source, categories := range target.DecisionCategoriesInFile {
			for _, candidate := range categories {
				if candidate == category {
					return source
				}
			}
		}
		for id, other := range target.Decisions {
			if asString(other["category_id"]) == category {
				return target.SourceFiles[entityKey("decision", id)]
			}
		}
	case "focus_node":
		treeID := asString(row["tree_id"])
		for id, other := range target.FocusNodes {
			if asString(other["tree_id"]) == treeID {
				return target.SourceFiles[entityKey("focus_node", id)]
			}
		}
	}
	return ""
}

func defaultEntityPath(kind, id string, row map[string]any, primaryTag string) string {
	switch kind {
	case "character":
		tag := strings.ToUpper(asString(row["country_tag"]))
		if tag == "" {
			tag = primaryTag
		}
		return "common/characters/" + tag + ".txt"
	case "event":
		ns := asString(row["namespace"])
		if ns == "" {
			ns = eventNamespace(asString(row["event_id"]))
		}
		if ns == "" {
			ns = "hagane_events"
		}
		return "events/" + ns + ".txt"
	case "decision":
		category := asString(row["category_id"])
		if category == "" {
			category = "hagane_decisions"
		}
		return "common/decisions/" + category + ".txt"
	case "focus_node":
		treeID := asString(row["tree_id"])
		if treeID == "" {
			treeID = "hagane_focus"
		}
		return "common/national_focus/" + treeID + ".txt"
	case "idea":
		return "common/ideas/hagane_ideas.txt"
	default:
		return "common/" + kind + "/" + id + ".txt"
	}
}

func eventNamespace(id string) string {
	ns, _, _ := strings.Cut(id, ".")
	if ns == id {
		return ""
	}
	return ns
}

func emitTargetFile(target *TargetMod, source string, overrides map[string]map[string]any, primaryTag string) (string, error) {
	kind := classifyTargetPath(source)
	switch kind {
	case "character":
		return emitCharacterFile(target, source, overrides, primaryTag)
	case "idea":
		return emitIdeaFile(target, source, overrides)
	case "event":
		return emitEventFile(target, source, overrides)
	case "decision":
		return emitDecisionFile(target, source, overrides)
	case "focus_node":
		return emitFocusFile(target, source, overrides, primaryTag)
	default:
		return "", fmt.Errorf("cannot determine entity type for %q", source)
	}
}

func rowsForSource(target *TargetMod, kind, source string, overrides map[string]map[string]any, idField string) []map[string]any {
	pool := target.ForType(kind)
	rows := make([]map[string]any, 0)
	handled := map[string]bool{}
	for id, row := range pool {
		if target.SourceFiles[entityKey(kind, id)] != source {
			continue
		}
		if replacement := overrides[entityKey(kind, id)]; replacement != nil {
			rows = append(rows, cloneJSON(replacement).(map[string]any))
		} else {
			rows = append(rows, cloneJSON(row).(map[string]any))
		}
		handled[id] = true
	}
	for key, row := range overrides {
		_, id, _ := strings.Cut(key, "/")
		if !handled[id] {
			rows = append(rows, cloneJSON(row).(map[string]any))
		}
	}

	sort.SliceStable(rows, func(i, j int) bool { return asString(rows[i][idField]) < asString(rows[j][idField]) })
	return rows
}

func emitCharacterFile(target *TargetMod, source string, overrides map[string]map[string]any, primaryTag string) (string, error) {
	ctag := strings.ToUpper(strings.TrimSuffix(path.Base(source), path.Ext(source)))
	if ctag == "" {
		ctag = primaryTag
	}
	rows := rowsForSource(target, "character", source, overrides, "tag")
	blocks := make([]string, 0, len(rows))
	for _, row := range rows {
		id := asString(row["tag"])
		original := target.Characters[id]
		portrait := asString(original["large_portrait"])
		advisor := asString(original["small_portrait"])
		body, _ := json.Marshal(map[string]any{
			"char": row, "country_tag": ctag,
			"has_portrait_image": portrait != "", "has_advisor_image": advisor != "",
			"portrait_path_override": portrait, "advisor_path_override": advisor,
		})
		text, err := emitpreview.EmitCharacterJSON(body)
		if err != nil {
			return "", err
		}
		blocks = append(blocks, text)
	}
	return strings.Join(blocks, "\n\n") + "\n", nil
}

func emitIdeaFile(target *TargetMod, source string, overrides map[string]map[string]any) (string, error) {
	rows := rowsForSource(target, "idea", source, overrides, "idea_id")
	for _, row := range rows {
		id := asString(row["idea_id"])
		if asString(row["picture"]) != asString(target.Ideas[id]["picture"]) {
			row["picture"] = ""
		}
	}
	body, _ := json.Marshal(map[string]any{"ideas": rows, "picture_refs": nil})
	text, err := emitpreview.EmitIdeasJSON(body)
	return text + "\n", err
}

func emitEventFile(target *TargetMod, source string, overrides map[string]map[string]any) (string, error) {
	rows := rowsForSource(target, "event", source, overrides, "event_id")
	namespace := ""
	validPictures := []string{}
	for _, row := range rows {
		id := asString(row["event_id"])
		if namespace == "" {
			namespace = target.EventNamespaces[id]
		}
		if picture := asString(target.Events[id]["picture"]); picture != "" {
			validPictures = append(validPictures, picture)
		}
	}
	if namespace == "" && len(rows) > 0 {
		namespace = eventNamespace(asString(rows[0]["event_id"]))
	}
	if namespace == "" {
		namespace = "hagane_events"
	}
	body, _ := json.Marshal(map[string]any{"namespace": namespace, "events": rows, "valid_pictures": validPictures})
	text, err := emitpreview.EmitEventsJSON(body)
	return text + "\n", err
}

func emitDecisionFile(target *TargetMod, source string, overrides map[string]map[string]any) (string, error) {
	rows := rowsForSource(target, "decision", source, overrides, "decision_id")
	byCategory := map[string][]map[string]any{}
	for _, row := range rows {
		id := asString(row["decision_id"])
		if asString(row["icon"]) != asString(target.Decisions[id]["icon"]) {
			row["icon"] = ""
		}
		category := asString(row["category_id"])
		if category == "" {
			category = "hagane_decisions"
		}
		byCategory[category] = append(byCategory[category], row)
	}
	categoryIDs := append([]string(nil), target.DecisionCategoriesInFile[source]...)
	seen := map[string]bool{}
	for _, category := range categoryIDs {
		seen[category] = true
	}
	extra := []string{}
	for category := range byCategory {
		if !seen[category] {
			extra = append(extra, category)
		}
	}
	sort.Strings(extra)
	categoryIDs = append(categoryIDs, extra...)
	categories := make([]map[string]any, 0, len(categoryIDs))
	decisions := make([]map[string]any, 0, len(rows))
	for index, category := range categoryIDs {
		pk := index + 1
		categories = append(categories, map[string]any{"id": pk, "category_id": category})
		for _, row := range byCategory[category] {
			row["category_id"] = pk
			decisions = append(decisions, row)
		}
	}
	body, _ := json.Marshal(map[string]any{"categories": categories, "decisions": decisions, "icon_gates": nil})
	text, err := emitmisc.EmitDecisionsJSON(body)
	return text + "\n", err
}

func emitFocusFile(target *TargetMod, source string, overrides map[string]map[string]any, primaryTag string) (string, error) {
	if file := target.focusSources[source]; file != nil {
		return file.Render(overrides)
	}
	rows := rowsForSource(target, "focus_node", source, overrides, "focus_id")
	treeID := strings.TrimSuffix(path.Base(source), path.Ext(source))
	if len(rows) > 0 && asString(rows[0]["tree_id"]) != "" {
		treeID = asString(rows[0]["tree_id"])
	}
	validIcons := map[string]bool{"GFX_goal_unknown": true}
	for _, original := range target.FocusNodes {
		if target.SourceFiles[entityKey("focus_node", asString(original["focus_id"]))] != source {
			continue
		}
		for _, key := range []string{"icon", "text_icon"} {
			if value := asString(original[key]); value != "" {
				validIcons[value] = true
			}
		}
		if dynamic, ok := original["dynamic_icon"].([]any); ok {
			for _, item := range dynamic {
				if row, ok := item.(map[string]any); ok {
					if value := asString(row["icon"]); value != "" {
						validIcons[value] = true
					}
				}
			}
		}
	}
	icons := make([]string, 0, len(validIcons))
	for icon := range validIcons {
		icons = append(icons, icon)
	}
	body, _ := json.Marshal(map[string]any{"tree": map[string]any{"tree_id": treeID, "country_tag": primaryTag}, "nodes": rows, "valid_icons": icons})
	text, err := emitmisc.EmitFocusTreeJSON(body)
	return text + "\n", err
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}
