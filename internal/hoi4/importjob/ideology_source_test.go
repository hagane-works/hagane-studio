package importjob

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeIdeologySourceTestFile(t *testing.T, root, rel, body string) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBuildIdeologySourceArchiveIsExactAndDeterministic(t *testing.T) {
	root := t.TempDir()
	writeIdeologySourceTestFile(t, root, "common/ideologies/00_base.txt", "ideologies = {}\n")
	writeIdeologySourceTestFile(t, root, "common/ideologies/nested/addon.TXT", "addon = {}\n")
	writeIdeologySourceTestFile(t, root, "common/ideologies/ignored.yml", "ignored")
	writeIdeologySourceTestFile(t, root, "common/ideas/not_an_ideology.txt", "ignored")

	first, manifest, err := BuildIdeologySourceArchive(root, "dependency-key", "base-mod", []string{" common/ideologies ", `COMMON\IDEOLOGIES`, "history/countries"})
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(first)
	firstBytes, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	second, secondManifest, err := BuildIdeologySourceArchive(root, "dependency-key", "base-mod", []string{"history/countries", "common/ideologies"})
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(second)
	secondBytes, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBytes, secondBytes) || !reflect.DeepEqual(manifest, secondManifest) {
		t.Fatal("archive or provenance changed across identical captures")
	}
	if manifest.SourceKey != "dependency-key" || manifest.DependencyKey != "base-mod" ||
		!reflect.DeepEqual(manifest.DescriptorReplacePaths, []string{"common/ideologies", "history/countries"}) || len(manifest.Files) != 2 {
		t.Fatalf("manifest=%#v", manifest)
	}
	zr, err := zip.OpenReader(first)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	got := []string{}
	for _, file := range zr.File {
		got = append(got, file.Name)
	}
	want := []string{"common/ideologies/00_base.txt", "common/ideologies/nested/addon.TXT"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("members=%v want=%v", got, want)
	}
	for i, file := range manifest.Files {
		if file.Path != want[i] || len(file.SHA256) != 64 {
			t.Fatalf("file provenance=%#v", file)
		}
	}
}

func TestBuildIdeologySourceArchiveRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	writeIdeologySourceTestFile(t, root, "common/ideologies/00_base.txt", "x")
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("escape"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "common", "ideologies", "linked.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if archive, _, err := BuildIdeologySourceArchive(root, "vanilla", "", nil); err == nil {
		if archive != "" {
			_ = os.Remove(archive)
		}
		t.Fatal("symlink was accepted")
	}
}

func TestBuildIdeologySourceArchiveEnforcesInputLimit(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "common", "ideologies", "oversize.txt")
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(ideologySourceSnapshotInputLimit + 1); err != nil {
		file.Close()
		t.Skipf("sparse files unavailable: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if archive, _, err := BuildIdeologySourceArchive(root, "vanilla", "", nil); err == nil {
		if archive != "" {
			_ = os.Remove(archive)
		}
		t.Fatal("oversized input was accepted")
	}
}

func TestBuildIdeologySourceArchiveReturnsNoSnapshotWhenAbsent(t *testing.T) {
	archive, manifest, err := BuildIdeologySourceArchive(t.TempDir(), "vanilla", "", nil)
	if err != nil || archive != "" || len(manifest.Files) != 0 {
		t.Fatalf("archive=%q manifest=%#v err=%v", archive, manifest, err)
	}
}

func TestDiscoverImportedIdeologySourceKeepsRootBodyAndSubtypeIDs(t *testing.T) {
	root := t.TempDir()
	writeIdeologySourceTestFile(t, root, "common/ideologies/technocracy.txt", `ideologies = {
	technocracy = {
		types = { civic_technocracy = { can_be_randomly_selected = no } }
		color = { 42 110 145 }
	}
}`)
	discovery, err := discoverImportedIdeologySource(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Roots) != 1 || discovery.Roots[0].RootID != "technocracy" ||
		len(discovery.Roots[0].SubtypeIDs) != 1 || discovery.Roots[0].SubtypeIDs[0] != "civic_technocracy" ||
		!strings.Contains(discovery.Roots[0].Body, "color = { 42 110 145 }") {
		t.Fatalf("discovery=%#v", discovery.Roots)
	}
}
