package export

import (
	"archive/zip"
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

func newProject(t *testing.T) *project.Project {
	t.Helper()
	store := project.NewStore(t.TempDir())
	record, err := workbench.CreateProject(store, workbench.CreateInput{Name: "Test Mod", CountryTag: "TST", CountryName: "Testland"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.Get(project.Int64(record["id"]))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestBuildNewProject(t *testing.T) {
	p := newProject(t)
	out := filepath.Join(t.TempDir(), "mod.zip")
	result, err := Build(context.Background(), p, Options{}, out)
	if err != nil {
		t.Fatal(err)
	}
	if result.SizeBytes == 0 || len(result.Files) == 0 {
		t.Fatalf("empty export: %+v", result)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	joined := strings.Join(names, "\n")
	if !strings.Contains(joined, ".mod") || !strings.Contains(joined, "/descriptor.mod") {
		t.Fatalf("export has no descriptor:\n%s", joined)
	}
	t.Logf("files:\n%s", joined)

	modDir := t.TempDir()
	installed, err := Install(out, modDir, "uuid-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(installed.Descriptor); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(out, modDir, "uuid-1"); err != nil {
		t.Fatalf("reinstall: %v", err)
	}
	if _, err := Install(out, modDir, "uuid-2"); err == nil {
		t.Fatal("installing over another project's mod folder must fail")
	}
}

func TestBuildWritesNotesInThePlayersLanguage(t *testing.T) {
	p := newProject(t)
	build := func(locale, notes string) exportengine.Manifest {
		result, err := Build(context.Background(), p, Options{Locale: locale}, filepath.Join(t.TempDir(), "mod.zip"))
		if err != nil {
			t.Fatal(err)
		}

		if want := exportmsg.For(notes).Text("job.checks_not_run", nil); len(result.NativeWarnings) == 0 || result.NativeWarnings[0] != want {
			t.Fatalf("%s notes = %q, want %q first", locale, result.NativeWarnings, want)
		}
		return result.NativeManifest
	}
	build("", "en")
	build("xx", "en")
	en, zh := build("en", "en"), build("zh", "zh")
	if en.Fingerprint != zh.Fingerprint || len(en.Entries) != len(zh.Entries) {
		t.Fatalf("the interface language changed the mod's identity: %s/%d vs %s/%d", en.Fingerprint, len(en.Entries), zh.Fingerprint, len(zh.Entries))
	}
	for i, entry := range zh.Entries {
		report := strings.HasSuffix(entry.Path, ".log") || path.Base(entry.Path) == exportengine.ExportFindingsBasename
		if entry.Path != en.Entries[i].Path || !report && entry.SHA256 != en.Entries[i].SHA256 {
			t.Errorf("%s differs between the English and the Chinese export", entry.Path)
		}
	}
}
