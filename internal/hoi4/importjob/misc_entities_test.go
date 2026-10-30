package importjob

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMiscKindCoverage(t *testing.T) {
	cases := map[string]string{
		"common/raids/categories/a.txt": "raids", "common/on_actions/a.txt": "on_actions",
	}
	for path, want := range cases {
		if got := miscKind(path); got != want {
			t.Errorf("%s: %q != %q", path, got, want)
		}
	}
	if got := miscKind("events/a.txt"); got != "" {
		t.Fatalf("unrelated routed as %q", got)
	}
}

func TestMiscFilesDeterministicAndTextOnly(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"common/raids/z.txt", "common/raids/a.txt", "common/raids/no.dds"} {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(rel), 0644); err != nil {
			t.Fatal(err)
		}
	}
	files, err := miscFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("files=%v", files)
	}
	if filepath.Base(files[0]) != "a.txt" || filepath.Base(files[1]) != "z.txt" {
		t.Fatalf("not sorted: %v", files)
	}
}
