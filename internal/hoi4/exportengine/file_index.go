package exportengine

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

type FileIndex struct {
	Filename string
	Paths    []string
}

func FileIndexFromManifest(manifest Manifest, filename string) FileIndex {
	paths := make([]string, 0, len(manifest.Entries)+1)
	entries := make([]Entry, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		paths = append(paths, entry.Path)
		entries = append(entries, Entry{Path: entry.Path})
	}
	manifestPath := manifest.ManifestPath
	if manifestPath == "" {
		manifestPath = artifactManifestPath(entries)
	}
	if manifestPath != "" {
		paths = append(paths, manifestPath)
	}
	return FileIndex{Filename: filename, Paths: paths}
}

var manifestOnlyBlob = func() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		panic("encode manifest-only image marker: " + err.Error())
	}

	out.WriteString("\x00HAGANE_NATIVE_FILE_INDEX_ONLY_v1")
	return out.Bytes()
}()

func manifestOnlyResolver(string, int64) ([]byte, error) {
	return append([]byte(nil), manifestOnlyBlob...), nil
}

func isManifestOnlyBlob(value []byte) bool {
	if len(value) != len(manifestOnlyBlob) {
		return false
	}
	for i := range value {
		if value[i] != manifestOnlyBlob[i] {
			return false
		}
	}
	return true
}

func BuildNativeFileIndex(snapshot Snapshot, replacePaths []string, includeImported bool) (index FileIndex, err error) {

	defer func() {
		if value := recover(); value != nil {
			index = FileIndex{}
			err = fmt.Errorf("native file index rejected project data: %v", value)
		}
	}()
	artifact, err := BuildNativeShadowWithBlobs(snapshot, replacePaths, includeImported, manifestOnlyResolver)
	if err != nil {
		return FileIndex{}, err
	}
	paths := make([]string, 0, len(artifact.Entries)+1)
	for _, entry := range artifact.Entries {
		paths = append(paths, entry.Path)
	}
	if manifestPath := artifactManifestPath(artifact.Entries); manifestPath != "" {
		for _, path := range paths {
			if equalArtifactPath(path, manifestPath) {
				return FileIndex{}, fmt.Errorf("artifact manifest path collides with an emitted entry: %q", path)
			}
		}
		paths = append(paths, manifestPath)
	}
	return FileIndex{Filename: artifact.Filename, Paths: paths}, nil
}

func equalArtifactPath(one, two string) bool {
	if len(one) != len(two) {
		return false
	}
	for i := range one {
		a, b := one[i], two[i]
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}
