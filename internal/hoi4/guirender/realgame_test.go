package guirender

import (
	"encoding/binary"
	"encoding/json"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"haganeworks/internal/hoi4/gfx"
)

func TestRealGameInterfaceCorpus(t *testing.T) {
	root := strings.TrimSpace(os.Getenv("HOI4_GAME_ROOT"))
	if root == "" {
		t.Skip("HOI4_GAME_ROOT unset; set it to a Hearts of Iron IV install to run the real-corpus gate")
	}
	files := map[string][]byte{}
	for _, sub := range []string{"interface", "localisation/english", "gfx/fonts"} {
		dir := filepath.Join(root, filepath.FromSlash(sub))
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", sub, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			switch strings.ToLower(filepath.Ext(entry.Name())) {
			case ".gui", ".gfx", ".yml", ".fnt":
			default:
				continue
			}
			body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			files[sub+"/"+entry.Name()] = body
		}
	}

	manifest := realGameManifest(t, root)
	raw, err := json.Marshal(VanillaManifest{GameVersion: "real", Textures: manifest})
	if err != nil {
		t.Fatal(err)
	}
	files[VanillaManifestPath] = raw

	start := time.Now()
	idx, err := NewVanillaIndex(files, "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatalf("index the real game: %v", err)
	}
	build := time.Since(start)

	parsed, failed := 0, []string{}
	for name, body := range files {
		if !strings.HasSuffix(name, ".gui") {
			continue
		}
		text, ok := decodeForTest(body)
		if !ok {
			failed = append(failed, name+" (not decodable text)")
			continue
		}
		if _, err := ParseGUI(text); err != nil {
			failed = append(failed, name+": "+err.Error())
			continue
		}
		parsed++
	}
	sort.Strings(failed)
	t.Logf("parsed %d/%d .gui files", parsed, parsed+len(failed))
	for _, entry := range failed {
		t.Logf("  unparsed: %s", entry)
	}

	rows := idx.ListWindows()
	t.Logf("indexed %d texture sizes and %d windows / %d sprites in %v",
		len(manifest), len(rows), idx.TotalSpriteCount(), build)

	if len(rows) < 800 {
		t.Fatalf("only %d windows indexed from the real game", len(rows))
	}
	if build > 30*time.Second {
		t.Fatalf("indexing the base game took %v; it runs once per process but not that once", build)
	}

	for _, name := range []string{
		"topbar", "countrypoliticsview", "nationalfocusview", "EventWindow",
		"countrydecisionview", "countrytechnologyview", "countryproductionlineview",
		"countrydeploymentview", "countryofficercorpview", "countrytechtreeview",
	} {
		window, ok := idx.Window(name)
		if !ok {
			t.Errorf("%s: not indexed", name)
			continue
		}
		sprites := window["sprites"].(map[string]any)
		sized, withURL := 0, 0
		for _, entry := range sprites {
			sprite := entry.(map[string]any)
			if sprite["resolved"] == true {
				sized++
			}
			if _, ok := sprite["vanilla_url"]; ok {
				withURL++
			}
		}
		texts := window["texts"].(map[string]any)
		translated := 0
		for _, entry := range texts {
			if entry.(map[string]any)["kind"] == "loc" {
				translated++
			}
		}
		t.Logf("%-28s sprites %3d (sized %3d, url %3d)  texts %3d (localised %3d)",
			name, len(sprites), sized, withURL, len(texts), translated)
		if len(sprites) > 0 && sized*4 < len(sprites)*3 {
			t.Errorf("%s: only %d/%d sprites sized", name, sized, len(sprites))
		}
		if sized != withURL {
			t.Errorf("%s: %d sized sprites but %d carry a URL", name, sized, withURL)
		}
	}
}

func realGameManifest(t *testing.T, root string) map[string][]int {
	t.Helper()
	out := map[string][]int{}
	base := filepath.Join(root, "gfx", "interface")
	err := filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".dds", ".tga", ".png":
		default:
			return nil
		}
		w, h, ok := imageHeaderSize(p)
		if !ok {
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return nil
		}
		out[strings.ToLower(filepath.ToSlash(rel))] = []int{w, h}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 5000 {
		t.Fatalf("only %d textures found under gfx/interface; is HOI4_GAME_ROOT a full install?", len(out))
	}
	return out
}

func imageHeaderSize(p string) (int, int, bool) {
	file, err := os.Open(p)
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()

	head := make([]byte, 32)
	n, _ := file.Read(head)
	head = head[:n]
	switch strings.ToLower(filepath.Ext(p)) {
	case ".dds":
		if len(head) < 20 || string(head[:4]) != "DDS " {
			return 0, 0, false
		}
		h := int(binary.LittleEndian.Uint32(head[12:16]))
		w := int(binary.LittleEndian.Uint32(head[16:20]))
		return w, h, w > 0 && h > 0
	case ".tga":
		if len(head) < 18 {
			return 0, 0, false
		}
		w := int(binary.LittleEndian.Uint16(head[12:14]))
		h := int(binary.LittleEndian.Uint16(head[14:16]))
		return w, h, w > 0 && h > 0
	}
	if _, err := file.Seek(0, 0); err != nil {
		return 0, 0, false
	}
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, false
	}
	return config.Width, config.Height, config.Width > 0 && config.Height > 0
}

func decodeForTest(body []byte) (string, bool) {
	return gfx.DecodeScriptText(body)
}
