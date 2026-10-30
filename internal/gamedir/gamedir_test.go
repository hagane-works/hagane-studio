package gamedir

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLibraryFolderPaths(t *testing.T) {
	vdf := filepath.Join(t.TempDir(), "libraryfolders.vdf")
	body := `"libraryfolders"
{
	"0"
	{
		"path"		"C:\\Program Files (x86)\\Steam"
		"label"		""
	}
	"1"
	{
		"path"		"D:\\SteamLibrary"
		"apps"
		{
			"394360"		"123"
		}
	}
}
`
	if err := os.WriteFile(vdf, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got := libraryFolderPaths(vdf)
	want := []string{`C:\Program Files (x86)\Steam`, `D:\SteamLibrary`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %q, want %q", got, want)
	}
}

func TestInspectReadsVersion(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"common", "interface", "gfx", "localisation"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "launcher-settings.json"), []byte(`{"rawVersion":"1.19.3.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	install, err := Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if install.Version != "1.19.3.0" {
		t.Fatalf("version = %q", install.Version)
	}
	if _, err := Inspect(t.TempDir()); err != ErrNotFound {
		t.Fatalf("empty folder: err = %v, want ErrNotFound", err)
	}
}
