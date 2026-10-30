package project

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlobPutReportsWhereTheWriteFailed(t *testing.T) {
	root := t.TempDir()
	p, err := NewStore(root).Create(Row{"name": "Demo"}, "Demo")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(p.FilesDir()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.FilesDir(), []byte("not a folder"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = p.Blobs().Put("local/p/x/portrait.png", "image/png", 3, strings.NewReader("png"))
	var write *WriteError
	if !errors.As(err, &write) {
		t.Fatalf("err = %v, want a WriteError", err)
	}
	if !strings.HasPrefix(write.Path, p.FilesDir()+string(filepath.Separator)) || write.Blocked() {
		t.Fatalf("path %q blocked %v", write.Path, write.Blocked())
	}
}

func TestWriteErrorNamesPermissionProblems(t *testing.T) {
	path := filepath.Join("C:", "mods", "files", "portrait.png")
	err := &WriteError{Path: path, Err: &os.LinkError{Op: "rename", Old: path + ".tmp", New: path, Err: fs.ErrPermission}}
	if !err.Blocked() {
		t.Fatal("permission error not reported as blocked")
	}
	if got, want := err.Error(), "could not write "+path+": "+fs.ErrPermission.Error(); got != want {
		t.Fatalf("message %q, want %q", got, want)
	}
	if (&WriteError{Path: path, Err: errors.New("disk full")}).Blocked() {
		t.Fatal("other errors must not count as blocked")
	}
}
