package guirender

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fixtureFiles(t *testing.T) map[string][]byte {
	t.Helper()
	root := filepath.Join("testdata", "full")
	out := map[string][]byte{}
	err := filepath.WalkDir(root, func(name string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		body, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = body
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestProjectWindowContract(t *testing.T) {
	idx, err := NewIndex(fixtureFiles(t))
	if err != nil {
		t.Fatal(err)
	}
	if idx.WindowCount() != 1 || idx.SpriteCount() != 1 {
		t.Fatalf("counts windows=%d sprites=%d", idx.WindowCount(), idx.SpriteCount())
	}
	window, ok := idx.Window("countrytechtreeview")
	if !ok {
		t.Fatal("window missing")
	}
	node := window["node"].(map[string]any)
	if got := node["position"].(map[string]any)["x"]; got != 12 {
		t.Fatalf("recursive @constant not expanded: %#v", got)
	}
	sprite := window["sprites"].(map[string]any)["GFX_panel_100x50"].(map[string]any)
	if sprite["frames"] != 2 || sprite["width"] != 200 || sprite["height"] != 50 || sprite["effect_kind"] != "button_state" {
		t.Fatalf("sprite metadata mismatch: %#v", sprite)
	}
	text := window["texts"].(map[string]any)["TECH_YEAR"].(map[string]any)

	if !reflect.DeepEqual(text, map[string]any{"display": "1936", "kind": "loc", "markup": "§Y1936§!"}) {
		t.Fatalf("text metadata mismatch: %#v", text)
	}
	font := window["fonts"].(map[string]any)["aldrich_24_outline"].(map[string]any)
	if font["family"] != "aldrich" || font["size"] != 24 || font["outline"] != true || font["line_ratio"] != 1.208 {
		t.Fatalf("font metadata mismatch: %#v", font)
	}
	groups := window["overlap_groups"].([]map[string]any)
	if len(groups) != 1 || groups[0]["default"] != "root/1" {
		t.Fatalf("overlap mismatch: %#v", groups)
	}
}

func TestFolderGeometryParity(t *testing.T) {
	idx, _ := NewIndex(fixtureFiles(t))
	window, _ := idx.Window("countrytechtreeview")
	node := window["node"].(map[string]any)
	geometry := FolderGeometry(FindNamedNode(node, "INFANTRY_FOLDER"), "infantry_folder")
	grids := geometry["grids"].([]map[string]any)
	if len(grids) != 1 || grids[0]["originX"] != 5 || grids[0]["originY"] != 6 || grids[0]["slotWidth"] != 80 || grids[0]["slotHeight"] != 90 {
		t.Fatalf("grid mismatch: %#v", grids)
	}
	years := geometry["yearLabels"].([]map[string]any)
	if len(years) != 1 || years[0]["x"] != 20 || years[0]["y"] != 30 {
		t.Fatalf("year mismatch: %#v", years)
	}
}

func TestParserPreservesLosslessRawBlocks(t *testing.T) {
	windows, err := ParseGUI(`guiTypes={containerWindowType={name=W position={x=1 y=2} drag_scroll={left middle} maxWidth=100%%}}`)
	if err != nil {
		t.Fatal(err)
	}
	node := windows[0]
	raw := node["raw"].(map[string]any)
	drag := raw["drag_scroll"].(map[string]any)["__block__"].([]map[string]any)
	if len(drag) != 2 || drag[0]["k"] != nil || drag[0]["v"] != "left" {
		t.Fatalf("raw block lost: %#v", drag)
	}
	if node["maxwidth"] != 0 || !strings.Contains(toString(node["scalar_tokens"].(map[string]any)["maxwidth"]), "%") {
		t.Fatalf("scalar token not preserved: %#v", node)
	}
}
