package exportengine

import (
	"encoding/json"
	"strings"

	"haganeworks/internal/hoi4/exportplan/timeline"
)

const ArtifactManifestBasename = "_HAGANE_ARTIFACT_MANIFEST.json"

type embeddedManifestEntry struct {
	Path           string  `json:"path"`
	SizeBytes      *int64  `json:"size_bytes"`
	SHA256         *string `json:"sha256"`
	Origin         string  `json:"origin,omitempty"`
	SourceEntity   string  `json:"source_entity,omitempty"`
	SelfHashPolicy string  `json:"self_hash_policy,omitempty"`
}

type embeddedManifest struct {
	Schema      string                     `json:"schema"`
	Fingerprint string                     `json:"fingerprint"`
	Timeline    *timeline.ArtifactMetadata `json:"timeline,omitempty"`
	EntryCount  int                        `json:"entry_count"`
	Entries     []embeddedManifestEntry    `json:"entries"`
}

func artifactManifestPath(entries []Entry) string {
	roots := map[string]bool{}
	for _, entry := range entries {
		value := strings.ReplaceAll(entry.Path, "\\", "/")
		if value == "" || !strings.Contains(value, "/") || isArtifactManifestPath(value) {
			continue
		}
		roots[value[:strings.Index(value, "/")]] = true
	}
	if len(roots) == 1 {
		for root := range roots {
			return root + "/" + ArtifactManifestBasename
		}
	}
	return ArtifactManifestBasename
}

func isArtifactManifestPath(path string) bool {
	value := strings.TrimRight(strings.ReplaceAll(path, "\\", "/"), "/")
	if slash := strings.LastIndex(value, "/"); slash >= 0 {
		value = value[slash+1:]
	}
	return strings.EqualFold(value, ArtifactManifestBasename)
}

func buildEmbeddedManifest(schema, fingerprint, manifestPath string, rows []ManifestEntry) ([]byte, error) {
	return buildEmbeddedManifestWithTimeline(schema, fingerprint, manifestPath, rows, nil)
}

func buildEmbeddedManifestWithTimeline(schema, fingerprint, manifestPath string, rows []ManifestEntry, metadata *timeline.ArtifactMetadata) ([]byte, error) {
	out := embeddedManifest{
		Schema:      schema,
		Fingerprint: fingerprint,
		Timeline:    cloneTimelineMetadata(metadata),
		EntryCount:  len(rows) + 1,
		Entries:     make([]embeddedManifestEntry, 0, len(rows)+1),
	}
	for _, row := range rows {
		size, sum := row.SizeBytes, row.SHA256
		out.Entries = append(out.Entries, embeddedManifestEntry{
			Path: row.Path, SizeBytes: &size, SHA256: &sum,
			Origin: row.Origin, SourceEntity: row.SourceEntity,
		})
	}
	out.Entries = append(out.Entries, embeddedManifestEntry{
		Path: manifestPath, Origin: "artifact_manifest",
		SelfHashPolicy: "excluded_from_embedded_manifest_to_avoid_self_reference",
	})
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}
