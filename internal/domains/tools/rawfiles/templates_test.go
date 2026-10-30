package rawfiles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRawTemplateBuildersParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "raw_files_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		BuilderCases []struct {
			Template string `json:"template"`
			Files    []struct {
				DestPath string `json:"dest_path"`
				Content  string `json:"content"`
			} `json:"files"`
		} `json:"builder_cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	sampleVars := []RawTemplateVar{
		{Key: "my_var", Label: "My Var"},
		{Key: "second_var", Label: "第二个"},
	}

	fgsBase := "guiTypes = {\n" +
		"\tcontainerWindowType = {\n" +
		"\t\tname = \"gamesetup_scenario_window\"\n" +
		"\t\tcontainerWindowType = {\n" +
		"\t\t\tname = \"bookmarks\"\n" +
		"\t\t\tgridBoxType = {\n" +
		"\t\t\t\tname = \"bookmarks_grid\"\n" +
		"\t\t\t\tslotsize = { width = 540 height = 120 }\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t}\n" +
		"\tcontainerWindowType = {\n" +
		"\t\tname = \"other_window\"\n" +
		"\t}\n" +
		"}\n"
	fgsFiles, fgsErr := BuildFrontendGameSetupSlotsize(fgsBase, 200)
	if fgsErr != "" {
		t.Errorf("frontendgamesetup build failed: %s", fgsErr)
	}
	built := map[string][]RawTemplateFile{
		"scripted_gui":      BuildScriptedGuiPanel("my_panel", "My Panel", sampleVars),
		"on_actions":        BuildOnActions("my_hooks", "GER"),
		"game_rules":        BuildGameRules("my_rules", "My Rule Title"),
		"ai_strategy":       BuildAIStrategy("my_strat", "GER"),
		"gfx":               BuildGfxSprite("my_sprite"),
		"defines":           BuildDefines("my_tweaks"),
		"bop":               BuildBop("my_bop"),
		"music":             BuildMusic("my_music"),
		"peace_conference":  BuildPeaceConference("my_peace"),
		"frontendgamesetup": fgsFiles,
	}
	covered := 0
	for _, c := range fx.BuilderCases {
		got, ok := built[c.Template]
		if !ok {
			t.Errorf("no Go builder for fixture template %q", c.Template)
			continue
		}
		covered++
		if len(got) != len(c.Files) {
			t.Errorf("%s: file count %d != %d", c.Template, len(got), len(c.Files))
			continue
		}
		for i, f := range c.Files {
			if got[i].DestPath != f.DestPath {
				t.Errorf("%s[%d]: dest %q != %q", c.Template, i, got[i].DestPath, f.DestPath)
			}
			if got[i].Content != f.Content {
				g, w := got[i].Content, f.Content

				pos := 0
				for pos < len(g) && pos < len(w) && g[pos] == w[pos] {
					pos++
				}
				lo := pos - 40
				if lo < 0 {
					lo = 0
				}
				gEnd, wEnd := pos+40, pos+40
				if gEnd > len(g) {
					gEnd = len(g)
				}
				if wEnd > len(w) {
					wEnd = len(w)
				}
				t.Errorf("%s[%d] %s: content mismatch at byte %d\n got: %q\nwant: %q",
					c.Template, i, f.DestPath, pos, g[lo:gEnd], w[lo:wEnd])
			}
		}
	}
	if covered != 10 {
		t.Errorf("covered %d builders, want 10", covered)
	}

	if _, e := BuildFrontendGameSetupSlotsize("", 500); e != "complete base GUI text is required" {
		t.Errorf("empty base: %q", e)
	}
	if _, e := BuildFrontendGameSetupSlotsize("guiTypes = { containerWindowType = { name = \"only\" } }", 500); e == "" {
		t.Error("single-window doc should be rejected")
	}
}
