package exportmerge

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeFolder(t *testing.T, files map[string][]byte) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestReadFolderReadsLikeAZIPOfTheFolder(t *testing.T) {
	mod := map[string][]byte{
		"descriptor.mod":               []byte("name=\"Base\"\n"),
		"common/decisions/x.txt":       []byte("x_cat = { x = { cost = 5 } }\n"),
		"localisation/english/x.yml":   []byte("\xef\xbb\xbfl_english:\n x:0 \"X\"\n"),
		"gfx/interface/empty_file.dds": {},
	}
	files := map[string][]byte{".git/config": []byte("[core]"), ".haganeworks-install": []byte("uuid\n"), "common/.notes": []byte("private")}
	for name, body := range mod {
		files[name] = body
	}
	dir := writeFolder(t, files)

	want, err := ReadArchive(context.Background(), writeZIP(t, mod), DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadFolder(context.Background(), dir, "", DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.ByPath, want.ByPath) || len(got.Entries) != len(want.Entries) {
		t.Fatalf("folder read %v, the ZIP %v", got.ByPath, want.ByPath)
	}

	rooted, err := ReadFolder(context.Background(), dir, "Base", DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range rooted.Entries {
		if !strings.HasPrefix(entry.Path, "Base/") || string(rooted.ByPath[entry.Path]) != string(mod[strings.TrimPrefix(entry.Path, "Base/")]) {
			t.Fatalf("entry %q is not the folder's file under Base/", entry.Path)
		}
	}
	if len(rooted.Entries) != len(mod) {
		t.Fatalf("%d entries under Base/, want %d", len(rooted.Entries), len(mod))
	}
}

func TestReadFolderBudgetsAndCancellation(t *testing.T) {
	dir := writeFolder(t, map[string][]byte{"a.txt": []byte("aaaa"), "b.txt": []byte("bbbb")})
	for name, change := range map[string]func(*ArchiveBudget){
		"entries":     func(b *ArchiveBudget) { b.MaxEntries = 1 },
		"entry bytes": func(b *ArchiveBudget) { b.MaxEntryBytes = 3 },
		"total bytes": func(b *ArchiveBudget) { b.MaxUncompressedBytes = 7 },
	} {
		budget := DefaultArchiveBudget()
		change(&budget)
		if _, err := ReadFolder(context.Background(), dir, "", budget); !errors.Is(err, ErrArchiveTooLarge) {
			t.Errorf("%s budget err=%v", name, err)
		}
	}
	if _, err := ReadFolder(context.Background(), dir, "..", DefaultArchiveBudget()); !errors.Is(err, ErrUnsafeArchive) {
		t.Errorf("unsafe prefix err=%v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ReadFolder(canceled, dir, "", DefaultArchiveBudget()); !errors.Is(err, context.Canceled) {
		t.Errorf("cancel err=%v", err)
	}
}

func TestReadFolderRefusesLinks(t *testing.T) {
	dir := writeFolder(t, map[string][]byte{"descriptor.mod": []byte("name=\"Base\"\n")})
	if err := os.Symlink(filepath.Join(dir, "descriptor.mod"), filepath.Join(dir, "link.mod")); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	if _, err := ReadFolder(context.Background(), dir, "", DefaultArchiveBudget()); !errors.Is(err, ErrUnsafeArchive) {
		t.Fatalf("symlink err=%v", err)
	}
}
