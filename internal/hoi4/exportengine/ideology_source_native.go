package exportengine

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/emit/locout"
	"haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/ideologysource"
)

type IdeologySourceNativeError struct {
	Code   string `json:"code"`
	Path   string `json:"path,omitempty"`
	Detail string `json:"detail"`
}

func (e IdeologySourceNativeError) Error() string { return e.Code + ": " + e.Detail }

type ideologySourceProject struct {
	Manifest json.RawMessage `json:"ideology_source_manifest"`
	Registry json.RawMessage `json:"ideology_registry"`
}

type ideologySourceManifestEnvelope struct {
	State     string `json:"state"`
	SourceKey string `json:"source_key"`
}

type ideologySourceSnapshotFile struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	SizeBytes *int64 `json:"size_bytes,omitempty"`
}

type ideologySourceSnapshotProvenance struct {
	SchemaVersion          int                          `json:"schema_version"`
	SourceKey              string                       `json:"source_key"`
	DependencyKey          string                       `json:"dependency_key,omitempty"`
	DescriptorReplacePaths []string                     `json:"descriptor_replace_paths"`
	Files                  []ideologySourceSnapshotFile `json:"files"`
}

type ideologySourceSnapshotRef struct {
	SourceKey  string
	OBSKey     string
	Provenance ideologySourceSnapshotProvenance
}

type ideologySourceRegistry struct {
	Subtypes json.RawMessage `json:"sub_ideologies"`
}

type ideologySourceRegistrySubtype struct {
	ID           string          `json:"id"`
	Parent       string          `json:"group"`
	Names        json.RawMessage `json:"name"`
	Descriptions json.RawMessage `json:"desc"`
	Source       string          `json:"source"`
}

func ideologySourceNativeError(code, detail string) error {
	return IdeologySourceNativeError{Code: code, Detail: detail}
}

const ideologySourceNotReadyCode = "ideology_source_manifest_not_ready"

func staleIdeologySourceLock(err error) bool {
	var blocker IdeologySourceNativeError
	return errors.As(err, &blocker) && blocker.Code == ideologySourceNotReadyCode
}

const ideologySourceArchiveLimit int64 = 520 << 20

func decodeIdeologySourceProject(projectRaw json.RawMessage) (ideologySourceProject, ideologysource.Manifest, string, bool, error) {
	var project ideologySourceProject
	if err := json.Unmarshal(projectRaw, &project); err != nil {
		return project, ideologysource.Manifest{}, "", false, ideologySourceNativeError("ideology_source_project_malformed", err.Error())
	}
	trimmed := bytes.TrimSpace(project.Manifest)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return project, ideologysource.Manifest{}, "", false, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return project, ideologysource.Manifest{}, "", false, ideologySourceNativeError("ideology_source_manifest_malformed", err.Error())
	}
	if len(fields) == 0 {
		return project, ideologysource.Manifest{}, "", false, nil
	}
	var manifest ideologysource.Manifest
	var envelope ideologySourceManifestEnvelope
	if err := json.Unmarshal(trimmed, &manifest); err != nil {
		return project, manifest, "", false, ideologySourceNativeError("ideology_source_manifest_malformed", err.Error())
	}
	if err := json.Unmarshal(trimmed, &envelope); err != nil {
		return project, manifest, "", false, ideologySourceNativeError("ideology_source_manifest_malformed", err.Error())
	}
	if strings.ToLower(strings.TrimSpace(envelope.State)) != "ready" {

		if len(manifest.Entries) == 0 {
			return project, manifest, "", false, nil
		}
		return project, manifest, strings.TrimSpace(envelope.SourceKey), false, ideologySourceNativeError(ideologySourceNotReadyCode, "the inherited ideology source manifest is not ready")
	}
	sourceKey := strings.TrimSpace(envelope.SourceKey)
	if sourceKey == "" || len(manifest.Entries) == 0 {
		return project, manifest, sourceKey, false, ideologySourceNativeError("ideology_source_manifest_malformed", "a ready inherited ideology source manifest needs a source_key and entries")
	}
	return project, manifest, sourceKey, true, nil
}

func normalizedIdeologyOutputPath(value string) string {
	value = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(value, `\`, "/")))
	for strings.HasPrefix(value, "./") {
		value = strings.TrimPrefix(value, "./")
	}
	for strings.Contains(value, "//") {
		value = strings.ReplaceAll(value, "//", "/")
	}
	return strings.Trim(value, "/")
}

func rejectIdeologySourceRawConflict(paths []string) error {
	if clean, owned := ideologyDirectoryOwner(paths); owned {
		return IdeologySourceNativeError{
			Code: "ideology_source_raw_conflict", Path: clean,
			Detail: fmt.Sprintf("project raw output %q already owns the inherited ideology directory", clean),
		}
	}
	return nil
}

func ideologyDirectoryOwner(paths []string) (string, bool) {
	ordered := append([]string(nil), paths...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return normalizedIdeologyOutputPath(ordered[i]) < normalizedIdeologyOutputPath(ordered[j])
	})
	for _, raw := range ordered {
		clean := normalizedIdeologyOutputPath(raw)
		if strings.HasPrefix(clean, ideologysource.IdeologyDirectory+"/") ||
			strings.Contains(clean, "/"+ideologysource.IdeologyDirectory+"/") {
			return clean, true
		}
	}
	return "", false
}

func inheritedSubtypeList(registry json.RawMessage, manifest ideologysource.Manifest, msg exportmsg.Printer) string {
	subtypes, err := projectIdeologySourceSubtypes(registry, manifest)
	if err != nil || len(subtypes) == 0 {
		return msg.Text("ideology.no_subtypes", nil)
	}
	ids := make([]string, 0, len(subtypes))
	for _, subtype := range subtypes {
		ids = append(ids, subtype.ScriptID)
	}
	return strings.Join(ids, ", ")
}

func decodeIdeologyLocaleMap(raw json.RawMessage, field string) (map[string]string, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return map[string]string{}, nil
	}
	var decoded map[string]string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, ideologySourceNativeError("ideology_source_registry_malformed", fmt.Sprintf("%s must be a locale-to-text object: %v", field, err))
	}
	result := make(map[string]string, len(decoded))
	for language, value := range decoded {
		language = strings.ToLower(strings.TrimSpace(language))
		if language == "" {
			return nil, ideologySourceNativeError("ideology_source_registry_malformed", field+" contains an empty locale")
		}
		if _, exists := result[language]; exists {
			return nil, ideologySourceNativeError("ideology_source_registry_malformed", field+" contains duplicate normalized locales")
		}
		result[language] = value
	}
	return result, nil
}

func projectIdeologySourceSubtypes(raw json.RawMessage, manifest ideologysource.Manifest) ([]ideologysource.Subtype, error) {
	var registry ideologySourceRegistry
	if len(bytes.TrimSpace(raw)) != 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		if err := json.Unmarshal(raw, &registry); err != nil {
			return nil, ideologySourceNativeError("ideology_source_registry_malformed", err.Error())
		}
	}
	var rows []json.RawMessage
	if len(bytes.TrimSpace(registry.Subtypes)) != 0 && !bytes.Equal(bytes.TrimSpace(registry.Subtypes), []byte("null")) {
		if err := json.Unmarshal(registry.Subtypes, &rows); err != nil {
			return nil, ideologySourceNativeError("ideology_source_registry_malformed", "sub_ideologies must be an array: "+err.Error())
		}
	}
	wanted := make(map[string]bool, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		wanted[strings.TrimSpace(entry.ScriptID)] = true
	}
	result := make([]ideologysource.Subtype, 0, len(manifest.Entries))
	for index, rawRow := range rows {
		var row ideologySourceRegistrySubtype
		if err := json.Unmarshal(rawRow, &row); err != nil {
			return nil, ideologySourceNativeError("ideology_source_registry_malformed", fmt.Sprintf("sub_ideologies[%d]: %v", index, err))
		}
		row.ID, row.Parent, row.Source = strings.TrimSpace(row.ID), strings.TrimSpace(row.Parent), strings.ToLower(strings.TrimSpace(row.Source))
		if !wanted[row.ID] || (row.Source != "" && row.Source != "created" && row.Source != "project_managed") {
			continue
		}
		names, err := decodeIdeologyLocaleMap(row.Names, fmt.Sprintf("sub_ideologies[%d].name", index))
		if err != nil {
			return nil, err
		}
		descriptions, err := decodeIdeologyLocaleMap(row.Descriptions, fmt.Sprintf("sub_ideologies[%d].desc", index))
		if err != nil {
			return nil, err
		}
		result = append(result, ideologysource.Subtype{
			ScriptID: row.ID, ParentRootID: row.Parent, Names: names, Descriptions: descriptions,
		})
	}
	return result, nil
}

func ideologySourceLocalisationEntries(modDir string, rows []ideologysource.LocalisationRow) ([]Entry, error) {
	byLanguage := map[string][]ideologysource.LocalisationRow{}
	for _, row := range rows {
		language := strings.ToLower(strings.TrimSpace(row.Language))
		if _, ok := emitmisc.LangCodeToHOI4[language]; !ok {
			return nil, IdeologySourceNativeError{Code: "ideology_source_localisation_language_invalid", Detail: fmt.Sprintf("unsupported content locale %q", row.Language)}
		}
		byLanguage[language] = append(byLanguage[language], row)
	}
	languages := make([]string, 0, len(byLanguage))
	for language := range byLanguage {
		languages = append(languages, language)
	}
	sort.Strings(languages)
	entries := make([]Entry, 0, len(languages))
	for _, language := range languages {
		hoi4Language := emitmisc.LangCodeToHOI4[language]
		lines := []string{"l_" + hoi4Language + ":"}
		for _, row := range byLanguage[language] {
			lines = append(lines, locout.YmlLine(row.Key, locout.Text{S: row.Value}))
		}
		body := append([]byte{0xef, 0xbb, 0xbf}, []byte(strings.Join(lines, "\n")+"\n")...)
		entries = append(entries, Entry{
			Path:  modDir + "/localisation/" + hoi4Language + "/replace/hagane_inherited_ideology_l_" + hoi4Language + ".yml",
			Bytes: body, Origin: "generated", SourceEntity: "ideology_source_localisation:" + language,
		})
	}
	return entries, nil
}

func safeIdeologySourceArchivePath(value string) (string, bool) {
	if strings.Contains(value, `\`) {
		return "", false
	}
	clean, err := SafePath(value)
	if err != nil {
		return "", false
	}
	lower := strings.ToLower(clean)
	return clean, strings.HasPrefix(lower, ideologysource.IdeologyDirectory+"/") && strings.HasSuffix(lower, ".txt")
}

func snapshotIdeologySourceRef(snapshot Snapshot, sourceKey string) (ideologySourceSnapshotRef, error) {
	var selected *ideologySourceSnapshotRef
	for _, raw := range snapshot.Tables["project_obs_assets"] {
		var row struct {
			AssetType  string          `json:"asset_type"`
			GFXName    string          `json:"gfx_name"`
			OBSKey     string          `json:"obs_key"`
			Provenance json.RawMessage `json:"provenance"`
		}
		if err := json.Unmarshal(raw, &row); err != nil {
			return ideologySourceSnapshotRef{}, ideologySourceNativeError("ideology_source_snapshot_provenance_invalid", err.Error())
		}
		if row.AssetType != "ideology_source" || strings.TrimSpace(row.GFXName) != sourceKey {
			continue
		}
		if selected != nil {
			return ideologySourceSnapshotRef{}, ideologySourceNativeError("ideology_source_snapshot_provenance_invalid", "multiple private snapshots match the manifest source_key")
		}
		var provenance ideologySourceSnapshotProvenance
		if len(bytes.TrimSpace(row.Provenance)) == 0 || json.Unmarshal(row.Provenance, &provenance) != nil {
			return ideologySourceSnapshotRef{}, ideologySourceNativeError("ideology_source_snapshot_provenance_invalid", "the private snapshot provenance is missing or malformed")
		}
		selected = &ideologySourceSnapshotRef{SourceKey: strings.TrimSpace(row.GFXName), OBSKey: strings.TrimSpace(row.OBSKey), Provenance: provenance}
	}
	if selected == nil {
		return ideologySourceSnapshotRef{}, ideologySourceNativeError("ideology_source_snapshot_missing", "the ready inherited ideology manifest has no matching private source snapshot")
	}
	ref := *selected
	if ref.OBSKey == "" || ref.Provenance.SchemaVersion != 1 ||
		strings.TrimSpace(ref.Provenance.SourceKey) != ref.SourceKey || len(ref.Provenance.Files) == 0 ||
		ref.Provenance.DescriptorReplacePaths == nil {
		return ideologySourceSnapshotRef{}, ideologySourceNativeError("ideology_source_snapshot_provenance_invalid", "the private snapshot pointer and provenance do not form a complete v1 record")
	}
	seen := map[string]bool{}
	for _, file := range ref.Provenance.Files {
		clean, ok := safeIdeologySourceArchivePath(file.Path)
		digest, hashErr := hex.DecodeString(strings.TrimSpace(file.SHA256))
		key := strings.ToLower(clean)
		if !ok || seen[key] || hashErr != nil || len(digest) != sha256.Size ||
			(file.SizeBytes != nil && *file.SizeBytes < 0) {
			return ideologySourceSnapshotRef{}, ideologySourceNativeError("ideology_source_snapshot_provenance_invalid", "the private snapshot provenance has an unsafe, duplicate, or invalid file record")
		}
		seen[key] = true
	}
	return ref, nil
}

func ideologySourceLayersFromArchive(ref ideologySourceSnapshotRef, archive []byte) ([]ideologysource.Layer, error) {
	if int64(len(archive)) > ideologySourceArchiveLimit {
		return nil, ideologySourceNativeError("ideology_source_snapshot_archive_invalid", "the private source archive exceeds its byte limit")
	}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, ideologySourceNativeError("ideology_source_snapshot_archive_invalid", "the private source snapshot is not a readable ZIP archive")
	}
	type expectedFile struct {
		path      string
		sha256    string
		sizeBytes *int64
	}
	expected := make(map[string]expectedFile, len(ref.Provenance.Files))
	for _, file := range ref.Provenance.Files {
		clean, ok := safeIdeologySourceArchivePath(file.Path)
		if !ok {
			return nil, ideologySourceNativeError("ideology_source_snapshot_provenance_invalid", "the private snapshot provenance contains an unsafe file path")
		}
		expected[strings.ToLower(clean)] = expectedFile{path: clean, sha256: strings.ToLower(strings.TrimSpace(file.SHA256)), sizeBytes: file.SizeBytes}
	}
	files := make([]ideologysource.SourceFile, 0, len(expected))
	seen := map[string]bool{}
	var total int64
	for _, member := range reader.File {
		clean, ok := safeIdeologySourceArchivePath(member.Name)
		key := strings.ToLower(clean)
		want, known := expected[key]
		if !ok || !known || seen[key] || member.FileInfo().IsDir() || !member.Mode().IsRegular() {
			return nil, IdeologySourceNativeError{Code: "ideology_source_snapshot_archive_invalid", Path: member.Name, Detail: "the private source archive member set or type differs from provenance"}
		}
		if member.UncompressedSize64 > uint64(ideologySourceArchiveLimit) || total > ideologySourceArchiveLimit-int64(member.UncompressedSize64) ||
			(want.sizeBytes != nil && uint64(*want.sizeBytes) != member.UncompressedSize64) {
			return nil, IdeologySourceNativeError{Code: "ideology_source_snapshot_archive_drift", Path: clean, Detail: "the private source archive member size differs from provenance or exceeds its limit"}
		}
		input, openErr := member.Open()
		if openErr != nil {
			return nil, ideologySourceNativeError("ideology_source_snapshot_archive_invalid", openErr.Error())
		}
		data, readErr := io.ReadAll(io.LimitReader(input, int64(member.UncompressedSize64)+1))
		closeErr := input.Close()
		if readErr != nil || closeErr != nil {
			return nil, ideologySourceNativeError("ideology_source_snapshot_archive_invalid", "the private source archive member could not be read")
		}
		digest := fmt.Sprintf("%x", sha256.Sum256(data))
		if uint64(len(data)) != member.UncompressedSize64 || digest != want.sha256 ||
			(want.sizeBytes != nil && int64(len(data)) != *want.sizeBytes) {
			return nil, IdeologySourceNativeError{Code: "ideology_source_snapshot_archive_drift", Path: clean, Detail: "the private source archive member bytes differ from provenance"}
		}
		total += int64(len(data))
		seen[key] = true
		files = append(files, ideologysource.SourceFile{Path: want.path, Bytes: data})
	}
	if len(seen) != len(expected) {
		return nil, ideologySourceNativeError("ideology_source_snapshot_archive_drift", "the private source archive is missing one or more provenance members")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	if strings.EqualFold(ref.SourceKey, ideologysource.SourceVanilla) {
		return []ideologysource.Layer{{SourceKind: ideologysource.SourceVanilla, LoadOrder: 0, Files: files}}, nil
	}
	dependencyKey := strings.TrimSpace(ref.Provenance.DependencyKey)
	if dependencyKey == "" {
		dependencyKey = ref.SourceKey
	}
	return []ideologysource.Layer{
		{SourceKind: ideologysource.SourceVanilla, LoadOrder: 0, Files: []ideologysource.SourceFile{}},
		{SourceKind: ideologysource.SourceDependency, DependencyKey: dependencyKey, LoadOrder: 1, ReplacePaths: append([]string(nil), ref.Provenance.DescriptorReplacePaths...), Files: files},
	}, nil
}

func nativeInheritedIdeologySourceFromSnapshot(snapshot Snapshot, rawEntries []Entry, modDir string, resolve BlobResolver) ([]Entry, error) {
	_, _, sourceKey, active, err := decodeIdeologySourceProject(snapshot.Project)
	if err != nil || !active {
		return nil, err
	}
	layers, err := inheritedIdeologySourceLayers(snapshot, sourceKey, resolve)
	if err != nil {
		return nil, err
	}
	return nativeInheritedIdeologySource(snapshot.Project, layers, ideologyRawPaths(rawEntries), modDir)
}

func inheritedIdeologySourceLayers(snapshot Snapshot, sourceKey string, resolve BlobResolver) ([]ideologysource.Layer, error) {
	if strings.EqualFold(sourceKey, ideologysource.SourceVanilla) {
		tables, loadErr := worldout.LoadVanillaTables(context.Background())
		if loadErr != nil || strings.TrimSpace(tables.IdeologyBaseText) == "" {
			return nil, ideologySourceNativeError("ideology_source_snapshot_unavailable", "the base game's ideology file is unavailable (set the Hearts of Iron IV folder)")
		}
		return []ideologysource.Layer{{
			SourceKind: ideologysource.SourceVanilla, LoadOrder: 0,
			Files: []ideologysource.SourceFile{{Path: "common/ideologies/00_ideologies.txt", Bytes: []byte(tables.IdeologyBaseText)}},
		}}, nil
	}
	ref, err := snapshotIdeologySourceRef(snapshot, sourceKey)
	if err != nil {
		return nil, err
	}
	if resolve == nil {
		return nil, ideologySourceNativeError("ideology_source_snapshot_resolver_required", "private object access is required for a ready inherited ideology manifest")
	}
	archive, err := resolve(ref.OBSKey, ideologySourceArchiveLimit)
	if err != nil {
		return nil, ideologySourceNativeError("ideology_source_snapshot_unavailable", "the private inherited ideology snapshot could not be read")
	}
	return ideologySourceLayersFromArchive(ref, archive)
}

func ideologyRawPaths(rawEntries []Entry) []string {
	paths := make([]string, len(rawEntries))
	for index := range rawEntries {
		paths[index] = rawEntries[index].Path
	}
	return paths
}

func relockedInheritedIdeologySource(snapshot Snapshot, project ideologySourceProject, manifest ideologysource.Manifest, sourceKey string, rawPaths []string, modDir string, resolve BlobResolver) ([]Entry, error) {
	if err := rejectIdeologySourceRawConflict(rawPaths); err != nil {
		return nil, err
	}
	layers, err := inheritedIdeologySourceLayers(snapshot, sourceKey, resolve)
	if err != nil {
		return nil, err
	}
	subtypes, err := projectIdeologySourceSubtypes(project.Registry, manifest)
	if err != nil {
		return nil, err
	}
	relocked, err := ideologysource.Relock(layers, subtypes)
	if err != nil {
		return nil, err
	}
	compiled, err := ideologysource.Compile(layers, relocked, subtypes)
	if err != nil {
		return nil, err
	}
	return compiledIdeologySourceEntries(compiled, modDir)
}

func ideologyRelockReason(err error) string {
	var findings ideologysource.FindingsError
	if errors.As(err, &findings) {
		codes := make([]string, 0, len(findings.Findings))
		for _, finding := range findings.Findings {
			codes = append(codes, finding.Code)
		}
		return strings.Join(codes, ", ")
	}
	var blocker IdeologySourceNativeError
	if errors.As(err, &blocker) {
		return blocker.Code
	}
	return err.Error()
}

func applyNativeInheritedIdeologySource(snapshot Snapshot, effectiveReplacePaths []string, rawEntries []Entry, resolve BlobResolver, artifact NativeArtifact, msg exportmsg.Printer) (NativeArtifact, error) {
	project, manifest, sourceKey, active, err := decodeIdeologySourceProject(snapshot.Project)
	stale := staleIdeologySourceLock(err)
	if err != nil && !stale {
		return artifact, err
	}
	if !stale && !active {
		return artifact, nil
	}

	skip := func(why exportmsg.Message) (NativeArtifact, error) {
		artifact.Warnings = append(artifact.Warnings, "ideology_source_skipped: "+msg.Text("ideology.skipped", exportmsg.Args{
			"reason": why, "subtypes": inheritedSubtypeList(project.Registry, manifest, msg),
		}))
		return artifact, nil
	}
	if containsCanonicalString(effectiveReplacePaths, ideologysource.IdeologyDirectory) {
		return skip(exportmsg.Message{Key: "ideology.reason.replace_path"})
	}
	rawPaths := ideologyRawPaths(rawEntries)
	if owner, owned := ideologyDirectoryOwner(rawPaths); owned {
		return skip(exportmsg.Message{Key: "ideology.reason.own_file", Args: exportmsg.Args{"file": owner}})
	}
	var entries []Entry
	if stale {

		entries, err = relockedInheritedIdeologySource(snapshot, project, manifest, sourceKey, rawPaths, artifactRoot(artifact), resolve)
		if err != nil {
			return skip(exportmsg.Message{Key: "ideology.reason.reimported", Args: exportmsg.Args{"codes": ideologyRelockReason(err)}})
		}
		artifact.Warnings = append(artifact.Warnings, "ideology_source_relocked: "+msg.Text("ideology.relocked", exportmsg.Args{
			"subtypes": inheritedSubtypeList(project.Registry, manifest, msg),
		}))
	} else {
		entries, err = nativeInheritedIdeologySourceFromSnapshot(snapshot, rawEntries, artifactRoot(artifact), resolve)
		if err != nil {
			return NativeArtifact{}, err
		}
	}
	artifact.Entries, err = overlayEntries(artifact.Entries, entries)
	if err != nil {
		return NativeArtifact{}, err
	}
	artifact.Coverage = append(artifact.Coverage, "ideology_source")
	return artifact, nil
}

func nativeInheritedIdeologySource(projectRaw json.RawMessage, sourceLayers []ideologysource.Layer, rawProjectOutputPaths []string, modDir string) ([]Entry, error) {
	project, manifest, _, active, err := decodeIdeologySourceProject(projectRaw)
	if err != nil || !active {
		return nil, err
	}
	if err := rejectIdeologySourceRawConflict(rawProjectOutputPaths); err != nil {
		return nil, err
	}
	subtypes, err := projectIdeologySourceSubtypes(project.Registry, manifest)
	if err != nil {
		return nil, err
	}
	compiled, err := ideologysource.Compile(sourceLayers, manifest, subtypes)
	if err != nil {
		return nil, err
	}
	return compiledIdeologySourceEntries(compiled, modDir)
}

func compiledIdeologySourceEntries(compiled ideologysource.Compilation, modDir string) ([]Entry, error) {
	entries := make([]Entry, 0, len(compiled.Files)+len(compiled.Localisations))
	for _, file := range compiled.Files {
		entries = append(entries, Entry{
			Path: modDir + "/" + file.Path, Bytes: file.Bytes,
			Origin: "generated", SourceEntity: "ideology_source:" + file.Path,
		})
	}
	locEntries, err := ideologySourceLocalisationEntries(modDir, compiled.Localisations)
	if err != nil {
		return nil, err
	}
	entries = append(entries, locEntries...)
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}
