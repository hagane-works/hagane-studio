package importjob

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLocFilePreservesOccurrences(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "localisation", "english", "replace", "x.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := "\ufeffl_english:\n key:0 \"one\"\n key: \"two\\nline\"\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	rows, err := parseLocFile(path, "localisation/english/replace/x.yml", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].key != "key" || rows[1].key != "key" {
		t.Fatal("duplicate lost")
	}
	if rows[0].version == nil || *rows[0].version != 0 {
		t.Fatal("explicit :0 lost")
	}
	if rows[1].version != nil {
		t.Fatal("omitted version changed")
	}
	if rows[1].value != "two\nline" {
		t.Fatalf("decode=%q", rows[1].value)
	}
	if !rows[0].isReplace || rows[0].replacePriority != 1 {
		t.Fatal("replace rank lost")
	}
}

func TestLocalisationFilesSorted(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"z.yml", "a.yml", "skip.txt"} {
		path := filepath.Join(root, "localisation", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	files, err := localisationFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || filepath.Base(files[0]) != "a.yml" {
		t.Fatalf("files=%v", files)
	}
}
