package exportengine

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func coverSnapshot(picture string, withAsset bool) Snapshot {
	tables := map[string][]json.RawMessage{}
	if withAsset {
		tables["project_obs_assets"] = []json.RawMessage{

			json.RawMessage(`{"id":1,"project_id":42,"asset_type":"project_cover","gfx_name":"thumbnail.png","obs_key":"cover/thumb.png"}`),
		}
	}
	return Snapshot{
		ProjectID: 42,
		Project:   json.RawMessage(`{"id":42,"name":"Demo","country_tag":"GER","picture":"` + picture + `"}`),
		Tables:    tables,
	}
}

func TestDescriptorOmitsPictureWhenTheCoverCannotBeShipped(t *testing.T) {
	artifact, err := BuildNativeShadowWithBlobs(coverSnapshot("thumbnail.png", false), nil, false,
		func(string, int64) ([]byte, error) { return nil, errBlobMissing })
	if err != nil {
		t.Fatal(err)
	}
	descriptor := descriptorText(t, artifact)
	if strings.Contains(descriptor, "picture=") {
		t.Fatalf("descriptor promises a cover that is not in the ZIP:\n%s", descriptor)
	}
	if _, ok := findEntry(artifact, "Demo/thumbnail.png"); ok {
		t.Fatal("thumbnail entry written despite unresolvable bytes")
	}

	entry, ok := findEntry(artifact, "Demo_PROJECT_COVER_WARNING.log")
	if !ok {
		t.Fatalf("no cover warning log; entries=%v", entryPaths(artifact))
	}
	if !strings.Contains(string(entry.Bytes), "thumbnail.png could not be read") {
		t.Fatalf("warning does not say why: %q", entry.Bytes)
	}
	if !warnsAbout(artifact, "thumbnail.png") {
		t.Fatalf("warning missing from artifact.Warnings: %v", artifact.Warnings)
	}
}

func TestDescriptorKeepsPictureWhenTheCoverShips(t *testing.T) {
	want := []byte("PNGDATA")
	artifact, err := BuildNativeShadowWithBlobs(coverSnapshot("thumbnail.png", true), nil, false,
		func(string, int64) ([]byte, error) { return want, nil })
	if err != nil {
		t.Fatal(err)
	}
	if descriptor := descriptorText(t, artifact); !strings.Contains(descriptor, `picture="thumbnail.png"`) {
		t.Fatalf("descriptor lost the cover reference:\n%s", descriptor)
	}
	entry, ok := findEntry(artifact, "Demo/thumbnail.png")
	if !ok {
		t.Fatalf("cover not shipped; entries=%v", entryPaths(artifact))
	}
	if !bytes.Equal(entry.Bytes, want) {
		t.Fatalf("cover bytes=%q want %q", entry.Bytes, want)
	}
	if _, ok := findEntry(artifact, "Demo_PROJECT_COVER_WARNING.log"); ok {
		t.Fatal("warning written for a cover that shipped fine")
	}
}

func TestNoCoverRequestedIsSilent(t *testing.T) {
	artifact, err := BuildNativeShadow(coverSnapshot("", false), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if descriptor := descriptorText(t, artifact); strings.Contains(descriptor, "picture=") {
		t.Fatalf("picture line invented from nothing:\n%s", descriptor)
	}
	if len(artifact.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", artifact.Warnings)
	}
}

func TestUnsupportedCoverFilenameWarnsAndDropsTheLine(t *testing.T) {
	artifact, err := BuildNativeShadow(coverSnapshot("cover.jpg", false), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if descriptor := descriptorText(t, artifact); strings.Contains(descriptor, "picture=") {
		t.Fatalf("unsupported cover name was written:\n%s", descriptor)
	}
	if !warnsAbout(artifact, "only thumbnail.png is supported") {
		t.Fatalf("warnings=%v", artifact.Warnings)
	}
}

func entryPaths(artifact NativeArtifact) []string {
	out := make([]string, 0, len(artifact.Entries))
	for _, entry := range artifact.Entries {
		out = append(out, entry.Path)
	}
	return out
}
