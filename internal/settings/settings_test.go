package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToolsToken(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Hagane Works")
	path := filepath.Join(dir, "settings.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadToolsToken(path); err == nil {
		t.Fatal("a token before the program made one")
	}
	first, err := store.ToolsToken()
	if err != nil || len(first) != 64 {
		t.Fatalf("token %q: %v", first, err)
	}
	if again, _ := store.ToolsToken(); again != first {
		t.Errorf("the token changed: %q then %q", first, again)
	}
	if read, err := ReadToolsToken(path); err != nil || read != first {
		t.Errorf("ReadToolsToken = %q, %v", read, err)
	}
	if ToolsTokenPath(path) != filepath.Join(dir, "tools-token") {
		t.Errorf("token path %s", ToolsTokenPath(path))
	}
	if err := os.WriteFile(ToolsTokenPath(path), []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if renewed, err := store.ToolsToken(); err != nil || renewed == first || len(renewed) != 64 {
		t.Errorf("an empty file gave %q, %v", renewed, err)
	}
}
