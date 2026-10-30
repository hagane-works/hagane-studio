package exportengine

import (
	"archive/zip"
	"encoding/json"
	"io"
	"path/filepath"
	"testing"

	"haganeworks/internal/hoi4/exportplan/timeline"
)

func TestBuildWithMetadataEmbedsTimelinePlanIdentity(t *testing.T) {
	plan := timeline.Build(timeline.Input{
		ProjectStartDate: "1964.5.1",
		RequestedEndDate: "1976.12.31",
		StructuredBookmarks: []timeline.BookmarkEvidence{{
			ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true,
		}},
		ModDirectory: "Demo",
	})
	metadata := plan.ArtifactMetadata()
	zipPath := filepath.Join(t.TempDir(), "demo.zip")
	manifest, err := BuildWithMetadata(zipPath, "artifact-fingerprint", []Entry{{
		Path: "Demo/descriptor.mod", Bytes: []byte("name=\"Demo\"\n"),
	}}, DefaultArtifactBudget(), &metadata)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Timeline == nil || manifest.Timeline.TimelinePlanSHA256 != plan.PlanSHA256 {
		t.Fatalf("returned manifest=%#v", manifest)
	}

	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	for _, file := range archive.File {
		if file.Name != "Demo/"+ArtifactManifestBasename {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		var embedded struct {
			Timeline *timeline.ArtifactMetadata `json:"timeline"`
		}
		if err := json.Unmarshal(body, &embedded); err != nil {
			t.Fatal(err)
		}
		if embedded.Timeline == nil || embedded.Timeline.TimelinePlanSHA256 != plan.PlanSHA256 {
			t.Fatalf("embedded manifest=%s", body)
		}
		return
	}
	t.Fatal("embedded artifact manifest missing")
}
