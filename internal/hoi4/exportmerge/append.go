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

	"haganeworks/internal/hoi4/cosmetic"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/exportplan/timeline"
)

func MergeAppend(ctx context.Context, base Archive, platform []exportengine.Entry, destination, fingerprint string) (exportengine.Manifest, []string, error) {
	return mergeAppend(ctx, base, platform, destination, fingerprint, nil, false, nil, nil, exportmsg.Printer{})
}

func MergeAppendWithFindings(ctx context.Context, base Archive, platform []exportengine.Entry, destination, fingerprint string, findings []exportengine.ExportFinding, msg exportmsg.Printer) (exportengine.Manifest, []string, error) {
	return mergeAppend(ctx, base, platform, destination, fingerprint, findings, true, nil, nil, msg)
}

func MergeAppendWithFindingsAndSnapshot(ctx context.Context, base Archive, platform []exportengine.Entry, destination, fingerprint string, findings []exportengine.ExportFinding, snapshot exportengine.Snapshot, metadata *timeline.ArtifactMetadata, msg exportmsg.Printer) (exportengine.Manifest, []string, error) {
	return mergeAppend(ctx, base, platform, destination, fingerprint, findings, true, &snapshot, metadata, msg)
}

func mergeAppend(ctx context.Context, base Archive, platform []exportengine.Entry, destination, fingerprint string, findings []exportengine.ExportFinding, manageFindings bool, snapshot *exportengine.Snapshot, metadata *timeline.ArtifactMetadata, msg exportmsg.Printer) (exportengine.Manifest, []string, error) {
	if manageFindings {
		var inherited []exportengine.ExportFinding
		var err error
		platform, inherited, err = stripExportFindingsEntries(platform)
		if err != nil {
			return exportengine.Manifest{}, nil, err
		}
		findings = append(inherited, findings...)
	}
	basePaths := make([]string, 0, len(base.Entries))
	for _, entry := range base.Entries {
		if manageFindings && strings.EqualFold(path.Base(entry.Path), exportengine.ExportFindingsBasename) {
			continue
		}
		basePaths = append(basePaths, entry.Path)
	}
	platformPaths := make([]string, 0, len(platform))
	for _, entry := range platform {
		platformPaths = append(platformPaths, entry.Path)
	}
	basePrefix := commonTopLevelPrefix(basePaths)
	platformPrefix := commonTopLevelPrefix(platformPaths)
	reroot := basePrefix != "" && platformPrefix != ""
	baseSet := map[string]bool{}
	baseFold := map[string]string{}
	for _, entry := range base.Entries {
		if strings.EqualFold(path.Base(entry.Path), exportengine.ArtifactManifestBasename) {
			continue
		}
		if manageFindings && strings.EqualFold(path.Base(entry.Path), exportengine.ExportFindingsBasename) {
			continue
		}
		baseSet[entry.Path] = true
		baseFold[strings.ToLower(entry.Path)] = entry.Path
	}

	type rebasedEntry struct {
		Source       string
		Dest         string
		ExactOverlap bool
		Entry        exportengine.Entry
	}
	rebased := make([]rebasedEntry, 0, len(platform))
	for _, entry := range platform {
		if err := ctx.Err(); err != nil {
			return exportengine.Manifest{}, nil, err
		}
		if strings.EqualFold(path.Base(entry.Path), exportengine.ArtifactManifestBasename) {
			continue
		}
		if reroot && !strings.Contains(entry.Path, "/") {
			continue
		}
		destinationPath := entry.Path
		if reroot && strings.HasPrefix(entry.Path, platformPrefix) {
			destinationPath = basePrefix + strings.TrimPrefix(entry.Path, platformPrefix)
		}
		if _, err := exportengine.SafePath(destinationPath); err != nil {
			return exportengine.Manifest{}, nil, err
		}
		exactOverlap := baseSet[destinationPath]

		if actual, exists := baseFold[strings.ToLower(destinationPath)]; exists {
			destinationPath = actual
		}
		rebased = append(rebased, rebasedEntry{Source: entry.Path, Dest: destinationPath, ExactOverlap: exactOverlap, Entry: entry})
	}
	overlap := map[string]bool{}
	for _, entry := range rebased {
		if entry.ExactOverlap {
			overlap[entry.Dest] = true
		}
	}

	entries := map[string]exportengine.Entry{}
	entryFold := map[string]string{}
	for _, entry := range ArchiveEntries(base, true) {
		if manageFindings && strings.EqualFold(path.Base(entry.Path), exportengine.ExportFindingsBasename) {
			continue
		}
		entries[entry.Path] = entry
		entryFold[strings.ToLower(entry.Path)] = entry.Path
	}
	descriptorMerged := map[string]bool{}
	onActionRouted := map[string]bool{}
	for _, current := range rebased {
		if err := ctx.Err(); err != nil {
			return exportengine.Manifest{}, nil, err
		}
		destinationPath := current.Dest
		body := current.Entry.Bytes
		origin := "platform_append"
		if baseSet[destinationPath] && len(current.Entry.CosmeticColorTags) > 0 && strings.HasSuffix("/"+strings.ToLower(destinationPath), "/common/countries/cosmetic.txt") {
			var err error
			body, err = cosmetic.MergeColorFiles(base.ByPath[destinationPath], body, current.Entry.CosmeticColorTags)
			if err != nil {
				return exportengine.Manifest{}, nil, fmt.Errorf("merge cosmetic colors %s: %w", destinationPath, err)
			}
			origin = "platform_cosmetic_tag_merge"
		} else if overlap[destinationPath] && isOnActionsPath(destinationPath) {
			destinationPath = haganeOnActionsSibling(destinationPath)
			onActionRouted[current.Dest] = true
			origin = "platform_on_actions_sibling"
		} else if overlap[destinationPath] && isDescriptorPath(destinationPath) {
			body = []byte(mergeDescriptor(string(base.ByPath[destinationPath]), string(body)))
			descriptorMerged[destinationPath] = true
			origin = "descriptor_union"
		} else if overlap[destinationPath] && isBookmarkScriptPath(destinationPath) {
			if merged, preserved := preserveBookmarkNavigationPlaceholder(base.ByPath[destinationPath], body); preserved {
				body = merged
				origin = "platform_bookmark_navigation_compat"
			}
		}
		if actual, exists := entryFold[strings.ToLower(destinationPath)]; exists {
			destinationPath = actual
		} else {
			entryFold[strings.ToLower(destinationPath)] = destinationPath
		}
		entries[destinationPath] = exportengine.Entry{Path: destinationPath, Bytes: body, Origin: origin, SourceEntity: "platform_zip_entry:" + current.Source}
	}

	conflicts := make([]string, 0)
	for collision := range overlap {
		if !onActionRouted[collision] && !descriptorMerged[collision] {
			conflicts = append(conflicts, collision)
		}
	}
	sort.Strings(conflicts)
	final := make([]exportengine.Entry, 0, len(entries))
	for _, entry := range entries {
		final = append(final, entry)
	}
	if manageFindings {
		final, findings = exportengine.RefreshFinalArtifactFindings(final, findings, snapshot, msg)
		report, err := exportengine.ExportFindingsEntry(final, findings)
		if err != nil {
			return exportengine.Manifest{}, nil, err
		}
		final = append(final, report)
	}
	if fingerprint == "" {
		fingerprint = appendFingerprint(base, platform)
		if manageFindings {
			fingerprint = mergeFindingsFingerprint(fingerprint, findings)
		}
	}
	if err := ctx.Err(); err != nil {
		return exportengine.Manifest{}, nil, err
	}
	manifest, err := exportengine.BuildContextWithMetadata(ctx, destination, fingerprint, final, exportengine.DefaultArtifactBudget(), metadata)
	return manifest, conflicts, err
}

func stripExportFindingsEntries(entries []exportengine.Entry) ([]exportengine.Entry, []exportengine.ExportFinding, error) {
	kept := make([]exportengine.Entry, 0, len(entries))
	var inherited []exportengine.ExportFinding
	found := false
	for _, entry := range entries {
		if !strings.EqualFold(path.Base(strings.ReplaceAll(entry.Path, "\\", "/")), exportengine.ExportFindingsBasename) {
			kept = append(kept, entry)
			continue
		}
		if found {
			return nil, nil, fmt.Errorf("generated entries contain more than one export findings record")
		}
		found = true
		doc, err := exportengine.DecodeExportFindingsDocument(entry.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("decode generated export findings: %w", err)
		}
		inherited = append(inherited, doc.Findings...)
	}
	return kept, inherited, nil
}

func appendFingerprint(base Archive, platform []exportengine.Entry) string {
	hasher := sha256.New()
	for _, entry := range base.Entries {
		_, _ = hasher.Write([]byte(entry.Path))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write(entry.Bytes)
	}
	ordered := append([]exportengine.Entry(nil), platform...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	for _, entry := range ordered {
		_, _ = hasher.Write([]byte(entry.Path))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write(entry.Bytes)
		if len(entry.CosmeticColorTags) > 0 {
			tags := append([]string(nil), entry.CosmeticColorTags...)
			sort.Strings(tags)
			policy, _ := json.Marshal(tags)
			_, _ = hasher.Write([]byte("\x00cosmetic-tag-merge-v1\x00"))
			_, _ = hasher.Write(policy)
		}
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func commonTopLevelPrefix(paths []string) string {
	tops := map[string]bool{}
	for _, name := range paths {
		name = strings.ReplaceAll(name, "\\", "/")
		head, _, found := strings.Cut(name, "/")
		if !found {
			continue
		}
		tops[head+"/"] = true
	}
	if len(tops) != 1 {
		return ""
	}
	for prefix := range tops {
		return prefix
	}
	return ""
}

func isOnActionsPath(name string) bool {
	name = strings.ToLower(strings.ReplaceAll(name, "\\", "/"))
	return strings.Contains(name, "common/on_actions/") && strings.HasSuffix(name, ".txt")
}

func haganeOnActionsSibling(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	if strings.Contains(name, "_hagane_on_actions.txt") {
		return name
	}
	if strings.Contains(name, "_on_actions.txt") {
		return strings.Replace(name, "_on_actions.txt", "_hagane_on_actions.txt", 1)
	}
	if strings.HasSuffix(strings.ToLower(name), ".txt") {
		return name[:len(name)-4] + "_hagane.txt"
	}
	return name + "_hagane"
}

func isDescriptorPath(name string) bool {
	name = strings.ToLower(strings.ReplaceAll(name, "\\", "/"))
	return strings.Contains(name, "/") && strings.HasSuffix(name, "/descriptor.mod")
}

type descriptorFields struct {
	Scalars map[string]string
	Blocks  map[string][]string
}

func parseDescriptor(text string) descriptorFields {
	out := descriptorFields{Scalars: map[string]string{}, Blocks: map[string][]string{}}
	for position := 0; position < len(text); {
		equal := strings.IndexByte(text[position:], '=')
		if equal < 0 {
			break
		}
		equal += position
		keyEnd := equal
		for keyEnd > 0 && (text[keyEnd-1] == ' ' || text[keyEnd-1] == '\t') {
			keyEnd--
		}
		keyStart := keyEnd
		for keyStart > 0 && ((text[keyStart-1] >= 'A' && text[keyStart-1] <= 'Z') || (text[keyStart-1] >= 'a' && text[keyStart-1] <= 'z') || text[keyStart-1] == '_') {
			keyStart--
		}
		key := strings.TrimSpace(text[keyStart:keyEnd])
		if key == "" {
			position = equal + 1
			continue
		}
		valueStart := equal + 1
		for valueStart < len(text) && (text[valueStart] == ' ' || text[valueStart] == '\t') {
			valueStart++
		}
		if valueStart < len(text) && text[valueStart] == '{' {
			depth, end := 0, valueStart
			for ; end < len(text); end++ {
				switch text[end] {
				case '{':
					depth++
				case '}':
					depth--
					if depth == 0 {
						end++
						goto blockDone
					}
				}
			}
		blockDone:
			out.Blocks[key] = quotedDescriptorItems(text[valueStart+1 : end-1])
			position = end
			continue
		}
		end := strings.IndexByte(text[valueStart:], '\n')
		if end < 0 {
			end = len(text)
		} else {
			end += valueStart
		}
		value := strings.TrimSpace(text[valueStart:end])
		value = strings.Trim(value, "\"")
		if key == "replace_path" {
			out.Blocks[key] = append(out.Blocks[key], value)
		} else {
			out.Scalars[key] = value
		}
		position = end + 1
	}
	return out
}

func quotedDescriptorItems(body string) []string {
	items := []string{}
	for position := 0; position < len(body); {
		start := strings.IndexByte(body[position:], '"')
		if start < 0 {
			break
		}
		start += position + 1
		end := strings.IndexByte(body[start:], '"')
		if end < 0 {
			break
		}
		end += start
		items = append(items, body[start:end])
		position = end + 1
	}
	if len(items) == 0 {
		items = strings.Fields(body)
	}
	return items
}

func mergeDescriptor(baseText, platformText string) string {
	base, platform := parseDescriptor(strings.TrimPrefix(baseText, "\ufeff")), parseDescriptor(strings.TrimPrefix(platformText, "\ufeff"))
	scalar := func(key string) string {
		if base.Scalars[key] != "" {
			return base.Scalars[key]
		}
		return platform.Scalars[key]
	}
	union := func(key string) []string {
		out := append([]string(nil), base.Blocks[key]...)
		seen := map[string]bool{}
		for _, item := range out {
			seen[item] = true
		}
		for _, item := range platform.Blocks[key] {
			if !seen[item] {
				out = append(out, item)
				seen[item] = true
			}
		}
		return out
	}
	lines := []string{}
	if value := scalar("version"); value != "" {
		lines = append(lines, fmt.Sprintf("version=%s", quoteDescriptor(value)))
	}
	if values := union("tags"); len(values) > 0 {
		lines = appendDescriptorBlock(lines, "tags", values)
	}
	if value := scalar("name"); value != "" {
		lines = append(lines, fmt.Sprintf("name=%s", quoteDescriptor(value)))
	}
	if value := scalar("supported_version"); value != "" {
		lines = append(lines, fmt.Sprintf("supported_version=%s", quoteDescriptor(value)))
	}
	if value := scalar("picture"); value != "" {
		lines = append(lines, fmt.Sprintf("picture=%s", quoteDescriptor(value)))
	}
	if values := union("dependencies"); len(values) > 0 {
		lines = appendDescriptorBlock(lines, "dependencies", values)
	}
	for _, value := range union("replace_path") {
		lines = append(lines, fmt.Sprintf("replace_path=%s", quoteDescriptor(value)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func quoteDescriptor(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func appendDescriptorBlock(lines []string, key string, values []string) []string {
	lines = append(lines, key+"={")
	for _, value := range values {
		lines = append(lines, "\t"+quoteDescriptor(value))
	}
	return append(lines, "}")
}
