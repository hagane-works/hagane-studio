package exportengine

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildDeterministicArtifact(t *testing.T) {
	entries := []Entry{
		{Path: "descriptor.mod", Bytes: []byte("name=\"Demo\"\n"), Origin: "generated"},
		{Path: "common/ideas/demo.txt", Bytes: []byte("ideas = {}\n"), Origin: "idea", SourceEntity: "ideas:1"},
	}
	one := filepath.Join(t.TempDir(), "one.zip")
	two := filepath.Join(t.TempDir(), "two.zip")
	m1, err := Build(one, "fingerprint", entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatal(err)
	}
	m2, err := Build(two, "fingerprint", []Entry{entries[1], entries[0]}, DefaultArtifactBudget())
	if err != nil {
		t.Fatal(err)
	}
	if m1.ZIPSHA256 != m2.ZIPSHA256 || m1.ZIPBytes != m2.ZIPBytes {
		t.Fatalf("artifact is not deterministic: %#v %#v", m1, m2)
	}
	b1, _ := os.ReadFile(one)
	b2, _ := os.ReadFile(two)
	if !bytes.Equal(b1, b2) {
		t.Fatal("zip bytes differ")
	}
	reader, err := zip.OpenReader(one)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	if len(reader.File) != 3 || reader.File[0].Name != "common/ideas/demo.txt" {
		var names []string
		for _, f := range reader.File {
			names = append(names, f.Name)
		}
		t.Fatalf("entries=%v", names)
	}
	if got := reader.File[2].Name; got != "common/"+ArtifactManifestBasename {
		t.Fatalf("manifest must be written last, beside the single root; got %q", got)
	}
	if m1.ManifestPath != "common/"+ArtifactManifestBasename {
		t.Fatalf("manifest path not reported to the caller: %q", m1.ManifestPath)
	}
}

type cancelAfterChecksContext struct {
	remaining int
	done      chan struct{}
	canceled  bool
}

func (c *cancelAfterChecksContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *cancelAfterChecksContext) Done() <-chan struct{}       { return c.done }
func (c *cancelAfterChecksContext) Value(any) any               { return nil }
func (c *cancelAfterChecksContext) Err() error {
	if c.canceled {
		return context.Canceled
	}
	if c.remaining > 0 {
		c.remaining--
		return nil
	}
	c.canceled = true
	close(c.done)
	return context.Canceled
}

func TestBuildContextCancellationDoesNotPublishPartialArtifact(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.zip")
	if err := os.WriteFile(destination, []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := &cancelAfterChecksContext{remaining: 5, done: make(chan struct{})}
	_, err := BuildContext(ctx, destination, "fp", []Entry{{
		Path: "large.bin", Bytes: bytes.Repeat([]byte("0123456789abcdef"), 128<<10),
	}}, DefaultArtifactBudget())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error=%v", err)
	}
	body, readErr := os.ReadFile(destination)
	if readErr != nil || string(body) != "previous" {
		t.Fatalf("canceled build replaced destination: body=%q err=%v", body, readErr)
	}
	if _, statErr := os.Stat(destination + ".part"); !os.IsNotExist(statErr) {
		t.Fatalf("canceled build left partial artifact: %v", statErr)
	}
}

func TestBuildEmbedsAReadableArtifactManifest(t *testing.T) {
	entries := []Entry{
		{Path: "Demo.mod", Bytes: []byte("name=\"Demo\"\n"), Origin: "generated"},
		{Path: "Demo/descriptor.mod", Bytes: []byte("name=\"Demo\"\n"), Origin: "generated"},
		{Path: "Demo/common/ideas/demo.txt", Bytes: []byte("ideas = {}\n"), Origin: "idea", SourceEntity: "ideas:1"},
	}
	path := filepath.Join(t.TempDir(), "demo.zip")
	manifest, err := Build(path, "fingerprint", entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatal(err)
	}

	if manifest.ManifestPath != "Demo/"+ArtifactManifestBasename {
		t.Fatalf("manifest path=%q want it beside the mod folder", manifest.ManifestPath)
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var body []byte
	for _, f := range reader.File {
		if f.Name != manifest.ManifestPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err = io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(body) == 0 {
		t.Fatal("no artifact manifest inside the ZIP")
	}
	var doc struct {
		Schema      string `json:"schema"`
		Fingerprint string `json:"fingerprint"`
		EntryCount  int    `json:"entry_count"`
		Entries     []struct {
			Path           string  `json:"path"`
			SHA256         *string `json:"sha256"`
			SelfHashPolicy string  `json:"self_hash_policy"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("embedded manifest is not readable JSON: %v\n%s", err, body)
	}
	if doc.Schema != ArtifactManifestSchema || doc.Fingerprint != "fingerprint" {
		t.Fatalf("schema/fingerprint drifted: %+v", doc)
	}
	if doc.EntryCount != 4 || len(doc.Entries) != 4 {
		t.Fatalf("want 3 entries + the manifest's own row, got %d", len(doc.Entries))
	}
	var selfRows int
	for _, row := range doc.Entries {
		if row.Path != manifest.ManifestPath {
			if row.SHA256 == nil || *row.SHA256 == "" {
				t.Fatalf("entry %q has no hash", row.Path)
			}
			continue
		}
		selfRows++

		if row.SHA256 != nil || row.SelfHashPolicy == "" {
			t.Fatalf("the manifest's own row must be hash-free and explain why: %+v", row)
		}
	}
	if selfRows != 1 {
		t.Fatalf("manifest must list itself exactly once, got %d", selfRows)
	}
}

func TestBuildRejectsManifestPathCollision(t *testing.T) {
	entries := []Entry{
		{Path: "Demo/descriptor.mod", Bytes: []byte("x"), Origin: "generated"},
		{Path: "Demo/" + ArtifactManifestBasename, Bytes: []byte("{}"), Origin: "raw_override"},
	}
	if _, err := Build(filepath.Join(t.TempDir(), "x.zip"), "fp", entries, DefaultArtifactBudget()); err == nil {
		t.Fatal("an entry already at the manifest path must fail, not be silently shadowed")
	}
}

func TestBuildRejectsUnsafeAndCollidingPaths(t *testing.T) {
	budget := DefaultArtifactBudget()
	for _, entries := range [][]Entry{
		{{Path: "../escape", Bytes: []byte("x")}},
		{{Path: "A.txt", Bytes: []byte("x")}, {Path: "a.txt", Bytes: []byte("y")}},
	} {
		if _, err := Build(filepath.Join(t.TempDir(), "bad.zip"), "fp", entries, budget); err == nil {
			t.Fatalf("unsafe entries accepted: %#v", entries)
		}
	}
}
