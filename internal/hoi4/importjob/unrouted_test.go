package importjob

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func writeFile(t *testing.T, root, rel string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUnrouted_FlagsDirectoryNobodyImports(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "common/names/zzz_probe_names.txt")
	writeFile(t, root, "common/names/more.txt")
	writeFile(t, root, "common/on_actions/kept.txt")
	writeFile(t, root, "localisation/english/x.yml")
	writeFile(t, root, "descriptor.mod")

	got, err := UnroutedByDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if got["common/names"] != 2 {
		t.Fatalf("common/names 应报 2 个文件,got %v", got)
	}
	for _, routed := range []string{"common/on_actions", "localisation/english", "<mod root>"} {
		if _, bad := got[routed]; bad {
			t.Fatalf("%s 被误报为未路由: %v", routed, got)
		}
	}
}

func TestUnrouted_SilentOnFullyRoutedMod(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"descriptor.mod",
		"common/national_focus/tree.txt",
		"common/ideas/spirits.txt",
		"common/ideologies/00_ideologies.txt",
		"history/countries/GER.txt",
		"events/ger.txt",
		"localisation/english/a_l_english.yml",
		"gfx/interface/icon.dds",
	} {
		writeFile(t, root, rel)
	}
	got, err := UnroutedByDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("完全在白名单内的 mod 不该报任何未路由目录,got %v", got)
	}
}

func TestUnrouted_RootFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "descriptor.mod")
	writeFile(t, root, "thumbnail.png")
	writeFile(t, root, "mystery.txt")
	got, err := UnroutedByDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if got["<mod root>"] != 1 {
		t.Fatalf("根目录陌生文件应报 1,got %v", got)
	}
}

func TestUnrouted_PrefixListStaysInSyncWithImporters(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	pathLiteral := regexp.MustCompile(`"(common|history|events)/[a-z_]+/`)
	missing := map[string]string{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		body, readErr := os.ReadFile(f)
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, m := range pathLiteral.FindAllString(string(body), -1) {
			prefix := strings.TrimPrefix(m, `"`)
			covered := false
			for _, routed := range routedPrefixes {

				if strings.HasPrefix(prefix, routed) || strings.HasPrefix(routed, prefix) {
					covered = true
					break
				}
			}
			if !covered {
				missing[prefix] = f
			}
		}
	}
	if len(missing) > 0 {
		for prefix, f := range missing {
			t.Errorf("路径前缀 %q(出现在 %s)不在 routedPrefixes 里 —— "+
				"该目录的文件会被误报为「平台没收下」", prefix, f)
		}
	}
}
