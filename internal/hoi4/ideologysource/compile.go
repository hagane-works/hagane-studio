package ideologysource

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type insertion struct {
	offset int
	text   []byte
}

func managedRegion(source []byte, types blockLocation, ids []string) []byte {
	eol := lineEnding(source)
	closeIndent := lineIndent(source, types.closeOffset)
	itemIndent := childIndent(source, types)
	fieldIndent := itemIndent + "\t"
	var out strings.Builder
	out.WriteString(eol)
	out.WriteString(itemIndent)
	out.Write(managedBegin)
	out.WriteString(eol)
	for _, id := range ids {
		out.WriteString(itemIndent)
		out.WriteString(id)
		out.WriteString(" = {")
		out.WriteString(eol)
		out.WriteString(fieldIndent)
		out.WriteString("can_be_randomly_selected = no")
		out.WriteString(eol)
		out.WriteString(itemIndent)
		out.WriteString("}")
		out.WriteString(eol)
	}
	out.WriteString(itemIndent)
	out.Write(managedEnd)
	out.WriteString(eol)
	out.WriteString(closeIndent)
	return []byte(out.String())
}

func buildPatchedFiles(state prepared, manifest Manifest) ([]PatchedFile, error) {
	entriesByFileRoot := map[string][]ManifestEntry{}
	for _, raw := range manifest.Entries {
		entry := normalizedEntry(raw)
		entriesByFileRoot[ownerKey(entry)] = append(entriesByFileRoot[ownerKey(entry)], entry)
	}
	insertionsByFile := map[string][]insertion{}
	fileByKey := map[string]discoveredFile{}
	for _, entries := range entriesByFileRoot {
		sort.Slice(entries, func(i, j int) bool { return entries[i].ScriptID < entries[j].ScriptID })
		entry := entries[0]
		location, exists := state.discovery.locations[keyForOwner(entry.ParentRootID, entry.OwnerPath)]
		if !exists || len(location.Types) != 1 {
			return nil, fmt.Errorf("validated ROOT location disappeared for %s", entry.ParentRootID)
		}
		file := state.discovery.files[strings.ToLower(entry.OwnerPath)]
		fileKey := entry.SourceKind + "\x00" + stringsKey(entry.DependencyKey) + "\x00" + stringsKey(entry.OwnerPath)
		fileByKey[fileKey] = file
		ids := make([]string, len(entries))
		for i := range entries {
			ids[i] = entries[i].ScriptID
		}
		insertionsByFile[fileKey] = append(insertionsByFile[fileKey], insertion{offset: location.Types[0].closeOffset, text: managedRegion(file.Bytes, location.Types[0], ids)})
	}
	keys := make([]string, 0, len(insertionsByFile))
	for key := range insertionsByFile {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]PatchedFile, 0, len(keys))
	for _, key := range keys {
		file := fileByKey[key]
		insertions := insertionsByFile[key]
		sort.Slice(insertions, func(i, j int) bool { return insertions[i].offset > insertions[j].offset })
		patched := append([]byte(nil), file.Bytes...)
		last := len(patched) + 1
		for _, insert := range insertions {
			if insert.offset < 0 || insert.offset > len(patched) || insert.offset >= last {
				return nil, fmt.Errorf("invalid or duplicate insertion offset for %s", file.Path)
			}
			patched = append(patched[:insert.offset], append(insert.text, patched[insert.offset:]...)...)
			last = insert.offset
		}
		parseBytes := patched
		if bytes.HasPrefix(parseBytes, []byte{0xef, 0xbb, 0xbf}) {
			parseBytes = parseBytes[3:]
		}
		if _, recoveries, err := paradox.Parse(string(parseBytes)); err != nil || len(recoveries) > 0 {
			return nil, findingsError(Finding{Code: "ideology_source_generated_malformed", OwnerPath: file.Path, Detail: "patched file is not a strict Paradox document"})
		}
		if _, err := scanSource(file.Path, patched, file.SourceKind, file.DependencyKey); err != nil {
			return nil, findingsError(Finding{Code: "ideology_source_generated_malformed", OwnerPath: file.Path, Detail: err.Error()})
		}
		result = append(result, PatchedFile{Path: file.Path, Bytes: patched})
	}
	return result, nil
}

func originalFile(layers []Layer, entry ManifestEntry) ([]byte, bool) {
	for _, layer := range layers {
		if layerKind(layer.SourceKind) != entry.SourceKind || strings.TrimSpace(layer.DependencyKey) != entry.DependencyKey {
			continue
		}
		for _, file := range layer.Files {
			if strings.EqualFold(normalizePath(file.Path), entry.OwnerPath) {
				return file.Bytes, true
			}
		}
	}
	return nil, false
}

func validateManagedRegions(layers []Layer, state prepared, manifest Manifest, files []PatchedFile) []Finding {
	expected := map[string][]byte{}
	for _, file := range files {
		expected[strings.ToLower(file.Path)] = file.Bytes
	}
	findings := []Finding{}
	seenFile := map[string]bool{}
	for _, raw := range manifest.Entries {
		entry := normalizedEntry(raw)
		pathKey := strings.ToLower(entry.OwnerPath)
		if seenFile[pathKey] {
			continue
		}
		seenFile[pathKey] = true
		regionKey := entry.SourceKind + "\x00" + stringsKey(entry.DependencyKey) + "\x00" + stringsKey(entry.OwnerPath)
		if state.regions[regionKey] == 0 {
			continue
		}
		current, ok := originalFile(layers, entry)
		if !ok || !bytes.Equal(current, expected[pathKey]) {
			findings = append(findings, withEntry(finding("ideology_source_managed_region_drift", "existing managed region differs from deterministic compiler output"), entry))
		}
	}
	sortFindings(findings)
	return findings
}

func localisationRows(subtypes []Subtype) []LocalisationRow {
	rows := []LocalisationRow{}
	for _, subtype := range subtypes {
		id := strings.TrimSpace(subtype.ScriptID)
		for language, value := range subtype.Names {
			if value = strings.TrimSpace(value); value != "" {
				rows = append(rows, LocalisationRow{Language: strings.ToLower(strings.TrimSpace(language)), Key: id, Value: value})
			}
		}
		for language, value := range subtype.Descriptions {
			if value = strings.TrimSpace(value); value != "" {
				rows = append(rows, LocalisationRow{Language: strings.ToLower(strings.TrimSpace(language)), Key: id + "_desc", Value: value})
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Language != rows[j].Language {
			return rows[i].Language < rows[j].Language
		}
		if rows[i].Key != rows[j].Key {
			return rows[i].Key < rows[j].Key
		}
		return rows[i].Value < rows[j].Value
	})
	return rows
}

func Compile(layers []Layer, manifest Manifest, subtypes []Subtype) (Compilation, error) {
	state, err := prepare(layers)
	if err != nil {
		return Compilation{}, err
	}
	findings := coreFindings(state.discovery, manifest)
	findings = append(findings, subtypeSetFindings(manifest, subtypes)...)
	if len(findings) > 0 {
		return Compilation{}, findingsError(findings...)
	}
	files, err := buildPatchedFiles(state, manifest)
	if err != nil {
		return Compilation{}, err
	}
	if findings := validateManagedRegions(layers, state, manifest, files); len(findings) > 0 {
		return Compilation{}, findingsError(findings...)
	}
	return Compilation{Files: files, Localisations: localisationRows(subtypes)}, nil
}
