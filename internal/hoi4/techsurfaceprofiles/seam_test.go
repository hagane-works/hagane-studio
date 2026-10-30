package techsurfaceprofiles

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestP0B1CentralSeams(t *testing.T) {
	if os.Getenv("HOI4_REQUIRE_P0_B1_SEAMS") != "1" {
		t.Skip("set HOI4_REQUIRE_P0_B1_SEAMS=1 in the central integration worktree")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate seam assertion source")
	}
	goRoot := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
	assertSourceContains(t, filepath.Join(goRoot, "techsurface", "surface.go"), "productionProfilesFromCatalogue()")
	assertSourceContains(t, filepath.Join(goRoot, "projectapi", "tech_surface_branches.go"), "techsurfaceprofiles.Resolve(")
}

func assertSourceContains(t *testing.T, path, marker string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read central seam %s: %v", path, err)
	}
	if !strings.Contains(string(body), marker) {
		t.Errorf("central seam missing in %s: expected marker %q", filepath.ToSlash(path), marker)
	}
}
