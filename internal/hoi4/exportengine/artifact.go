package exportengine

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"haganeworks/internal/hoi4/exportplan/timeline"
)

const ArtifactManifestSchema = "hoi4.artifact-manifest.v1"

type Entry struct {
	Path         string
	Bytes        []byte
	Origin       string
	SourceEntity string

	CosmeticColorTags []string
}

type ManifestEntry struct {
	Path         string `json:"path"`
	SizeBytes    int64  `json:"size_bytes"`
	SHA256       string `json:"sha256"`
	Origin       string `json:"origin,omitempty"`
	SourceEntity string `json:"source_entity,omitempty"`
}

type Manifest struct {
	Schema      string          `json:"schema"`
	Fingerprint string          `json:"fingerprint"`
	Entries     []ManifestEntry `json:"entries"`

	Timeline  *timeline.ArtifactMetadata `json:"timeline,omitempty"`
	ZIPBytes  int64                      `json:"zip_bytes"`
	ZIPSHA256 string                     `json:"zip_sha256"`

	ManifestPath string `json:"manifest_path,omitempty"`
}

type ArtifactBudget struct {
	MaxEntries    int
	MaxEntryBytes int64
	MaxTotalBytes int64
}

func DefaultArtifactBudget() ArtifactBudget {
	return ArtifactBudget{MaxEntries: 20000, MaxEntryBytes: 256 << 20, MaxTotalBytes: 4 << 30}
}

func Build(destination, fingerprint string, entries []Entry, budget ArtifactBudget) (Manifest, error) {
	return BuildWithMetadata(destination, fingerprint, entries, budget, nil)
}

func BuildWithMetadata(destination, fingerprint string, entries []Entry, budget ArtifactBudget, metadata *timeline.ArtifactMetadata) (Manifest, error) {
	return buildContextWithMetadata(context.Background(), destination, fingerprint, entries, budget, metadata)
}

func BuildContext(ctx context.Context, destination, fingerprint string, entries []Entry, budget ArtifactBudget) (Manifest, error) {
	return buildContextWithMetadata(ctx, destination, fingerprint, entries, budget, nil)
}

func BuildContextWithMetadata(ctx context.Context, destination, fingerprint string, entries []Entry, budget ArtifactBudget, metadata *timeline.ArtifactMetadata) (Manifest, error) {
	return buildContextWithMetadata(ctx, destination, fingerprint, entries, budget, metadata)
}

func buildContextWithMetadata(ctx context.Context, destination, fingerprint string, entries []Entry, budget ArtifactBudget, metadata *timeline.ArtifactMetadata) (Manifest, error) {
	if err := ctx.Err(); err != nil {
		return Manifest{}, err
	}
	if destination == "" || fingerprint == "" {
		return Manifest{}, errors.New("export artifact requires destination and fingerprint")
	}
	if budget.MaxEntries < 1 || budget.MaxEntryBytes < 1 || budget.MaxTotalBytes < 1 {
		return Manifest{}, errors.New("invalid artifact budget")
	}
	if len(entries) > budget.MaxEntries {
		return Manifest{}, fmt.Errorf("artifact has %d entries; budget is %d", len(entries), budget.MaxEntries)
	}
	ordered := append([]Entry(nil), entries...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })

	manifestPath := artifactManifestPath(ordered)
	seen := make(map[string]string, len(ordered))
	var total int64
	for _, entry := range ordered {
		if err := ctx.Err(); err != nil {
			return Manifest{}, err
		}
		clean, err := SafePath(entry.Path)
		if err != nil {
			return Manifest{}, err
		}
		folded := strings.ToLower(clean)
		if previous, exists := seen[folded]; exists {
			return Manifest{}, fmt.Errorf("artifact path collision: %q and %q", previous, clean)
		}
		seen[folded] = clean
		size := int64(len(entry.Bytes))
		if size > budget.MaxEntryBytes {
			return Manifest{}, fmt.Errorf("artifact entry %q exceeds byte budget", clean)
		}
		total += size
		if total > budget.MaxTotalBytes {
			return Manifest{}, errors.New("artifact exceeds total uncompressed byte budget")
		}
	}

	if previous, exists := seen[strings.ToLower(manifestPath)]; exists {
		return Manifest{}, fmt.Errorf("artifact manifest path collides with an emitted entry: %q", previous)
	}

	tmp := destination + ".part"
	_ = os.Remove(tmp)
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return Manifest{}, err
	}
	hasher := sha256.New()
	zw := zip.NewWriter(io.MultiWriter(file, hasher))
	manifest := Manifest{Schema: ArtifactManifestSchema, Fingerprint: fingerprint, Timeline: cloneTimelineMetadata(metadata)}
	fixedTime := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, entry := range ordered {
		if err := ctx.Err(); err != nil {
			_ = file.Close()
			_ = os.Remove(tmp)
			return Manifest{}, err
		}
		header := &zip.FileHeader{Name: entry.Path, Method: zip.Deflate}
		header.SetModTime(fixedTime)
		writer, createErr := zw.CreateHeader(header)
		if createErr != nil {
			_ = zw.Close()
			_ = file.Close()
			_ = os.Remove(tmp)
			return Manifest{}, createErr
		}
		for offset := 0; offset < len(entry.Bytes); {
			if err := ctx.Err(); err != nil {
				_ = file.Close()
				_ = os.Remove(tmp)
				return Manifest{}, err
			}
			end := offset + 256<<10
			if end > len(entry.Bytes) {
				end = len(entry.Bytes)
			}
			written, writeErr := writer.Write(entry.Bytes[offset:end])
			if writeErr != nil {
				_ = zw.Close()
				_ = file.Close()
				_ = os.Remove(tmp)
				return Manifest{}, writeErr
			}
			if written == 0 {
				_ = file.Close()
				_ = os.Remove(tmp)
				return Manifest{}, io.ErrShortWrite
			}
			offset += written
		}
		sum := sha256.Sum256(entry.Bytes)
		manifest.Entries = append(manifest.Entries, ManifestEntry{
			Path: entry.Path, SizeBytes: int64(len(entry.Bytes)), SHA256: hex.EncodeToString(sum[:]),
			Origin: entry.Origin, SourceEntity: entry.SourceEntity,
		})
	}
	if err := ctx.Err(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return Manifest{}, err
	}

	manifestBytes, err := buildEmbeddedManifestWithTimeline(
		ArtifactManifestSchema, fingerprint, manifestPath, manifest.Entries, manifest.Timeline,
	)
	if err != nil {
		_ = zw.Close()
		_ = file.Close()
		_ = os.Remove(tmp)
		return Manifest{}, err
	}
	manifestHeader := &zip.FileHeader{Name: manifestPath, Method: zip.Deflate}
	manifestHeader.SetModTime(fixedTime)
	manifestWriter, err := zw.CreateHeader(manifestHeader)
	if err != nil {
		_ = zw.Close()
		_ = file.Close()
		_ = os.Remove(tmp)
		return Manifest{}, err
	}
	if _, err := manifestWriter.Write(manifestBytes); err != nil {
		_ = zw.Close()
		_ = file.Close()
		_ = os.Remove(tmp)
		return Manifest{}, err
	}
	manifest.ManifestPath = manifestPath
	if err := ctx.Err(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return Manifest{}, err
	}
	if err := zw.Close(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return Manifest{}, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return Manifest{}, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return Manifest{}, err
	}
	info, err := os.Stat(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return Manifest{}, err
	}
	manifest.ZIPBytes = info.Size()
	manifest.ZIPSHA256 = hex.EncodeToString(hasher.Sum(nil))
	if err := ctx.Err(); err != nil {
		_ = os.Remove(tmp)
		return Manifest{}, err
	}
	if err := os.Rename(tmp, destination); err != nil {
		_ = os.Remove(tmp)
		return Manifest{}, err
	}
	return manifest, nil
}

func cloneTimelineMetadata(metadata *timeline.ArtifactMetadata) *timeline.ArtifactMetadata {
	if metadata == nil {
		return nil
	}
	copy := *metadata
	copy.Fingerprint.EffectiveReplacePaths = append(
		[]string(nil), metadata.Fingerprint.EffectiveReplacePaths...)
	return &copy
}

func SafePath(name string) (string, error) {
	if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return "", fmt.Errorf("unsafe artifact path %q", name)
	}
	clean := path.Clean(name)
	if clean != name || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe artifact path %q", name)
	}
	return clean, nil
}

func MarshalManifest(manifest Manifest) ([]byte, error) {
	return json.MarshalIndent(manifest, "", "  ")
}
