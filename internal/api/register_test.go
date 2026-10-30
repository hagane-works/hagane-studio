package api

import (
	"net/http"
	"path/filepath"
	"testing"

	"haganeworks/internal/settings"
)

func TestRegisterHasNoConflicts(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("route conflict: %v", r)
		}
	}()
	Register(http.NewServeMux(), Deps{Settings: store, GameIndexDir: filepath.Join(t.TempDir(), "game-index"), FindGame: noInstalledGame})
}
