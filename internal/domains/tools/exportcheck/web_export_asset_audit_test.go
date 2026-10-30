package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAssetAuditParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "asset_audit_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name  string `json:"name"`
			Input struct {
				Deps            []string `json:"deps"`
				IncludeImported bool     `json:"include_imported"`
				Ideas           []struct {
					ID         int64  `json:"id"`
					IdeaID     string `json:"idea_id"`
					Picture    string `json:"picture"`
					Source     string `json:"source"`
					CountryTag string `json:"country_tag"`
				} `json:"ideas"`
				Events []struct {
					ID         int64  `json:"id"`
					EventID    string `json:"event_id"`
					Picture    string `json:"picture"`
					Source     string `json:"source"`
					CountryTag string `json:"country_tag"`
				} `json:"events"`
				Decisions []struct {
					ID         int64  `json:"id"`
					DecisionID string `json:"decision_id"`
					Icon       string `json:"icon"`
					Source     string `json:"source"`
				} `json:"decisions"`
				Techs []struct {
					ID     int64  `json:"id"`
					TechID string `json:"tech_id"`
					Source string `json:"source"`
				} `json:"techs"`
				Characters []struct {
					ID            int64  `json:"id"`
					Tag           string `json:"tag"`
					LargePortrait string `json:"large_portrait"`
					Source        string `json:"source"`
					CountryTag    string `json:"country_tag"`
				} `json:"characters"`
				Trees []struct {
					Source string `json:"source"`
					Nodes  []struct {
						FocusID string `json:"focus_id"`
						Icon    string `json:"icon"`
					} `json:"nodes"`
				} `json:"trees"`
			} `json:"input"`
			Expected json.RawMessage `json:"expected"`
		} `json:"cases"`
		Namers []struct {
			Name   string   `json:"name"`
			Args   []string `json:"args"`
			Result string   `json:"result"`
		} `json:"namers"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Cases) == 0 {
		t.Fatal("fixture 为空")
	}

	resolveNone := func(string, int64, string) bool { return false }
	vanillaIdea := func(name string) bool { return name == "vanilla_pic" }
	vanillaFocus := func(name string) bool { return name == "GFX_goal_generic_alliance" }

	for _, c := range fx.Cases {
		in := AssetAuditInput{
			ProjectTag:      "GER",
			HasDependencies: len(c.Input.Deps) > 0,
			IncludeImported: c.Input.IncludeImported,
		}
		for _, r := range c.Input.Ideas {
			in.Ideas = append(in.Ideas, AssetAuditEntity{ID: r.ID, IdeaID: r.IdeaID,
				Picture: r.Picture, Source: r.Source, CountryTag: r.CountryTag})
		}
		for _, r := range c.Input.Events {
			in.Events = append(in.Events, AssetAuditEntity{ID: r.ID, EventID: r.EventID,
				Picture: r.Picture, Source: r.Source, CountryTag: r.CountryTag})
		}
		for _, r := range c.Input.Decisions {
			in.Decisions = append(in.Decisions, AssetAuditEntity{ID: r.ID,
				DecisionID: r.DecisionID, Icon: r.Icon, Source: r.Source})
		}
		for _, r := range c.Input.Techs {
			in.Techs = append(in.Techs, AssetAuditEntity{ID: r.ID, TechID: r.TechID,
				Source: r.Source})
		}
		for _, r := range c.Input.Characters {
			in.Characters = append(in.Characters, AssetAuditEntity{ID: r.ID, Tag: r.Tag,
				LargePortrait: r.LargePortrait, Source: r.Source, CountryTag: r.CountryTag})
		}
		for _, tr := range c.Input.Trees {
			gt := AssetAuditTree{Source: tr.Source}
			for _, n := range tr.Nodes {
				gt.Nodes = append(gt.Nodes, AssetAuditEntity{FocusID: n.FocusID, Icon: n.Icon})
			}
			in.Trees = append(in.Trees, gt)
		}

		got := AuditProjectAssets(in, resolveNone, vanillaIdea, vanillaFocus)
		compareJSON(t, "asset-audit "+c.Name, got, c.Expected)
	}

	for _, n := range fx.Namers {
		var got string
		switch n.Name {
		case "idea_picture_filename", "idea_picture_filename-spaces":
			got = ideaPictureFilename(n.Args[0], n.Args[1])
		case "event_picture_filename":
			got = eventPictureFilename(n.Args[0], n.Args[1])
		case "decision_icon_filename":
			got = decisionIconFilename(n.Args[0])
		case "tech_icon_filename":
			got = techIconFilename(n.Args[0])
		case "focus_icon_sprite_name":
			got = focusIconSpriteName(n.Args[0])
		case "portrait_filename":
			got = portraitFilename(n.Args[0], n.Args[1])
		case "extract_slug-prefixed", "extract_slug-plain":
			got = extractPortraitSlug(n.Args[0], n.Args[1])
		default:
			t.Fatalf("namer %s 没有对应的 Go 调用", n.Name)
		}
		if got != n.Result {
			t.Errorf("namer %s = %q want %q", n.Name, got, n.Result)
		}
	}
}

func TestSubmodSkipAppliesToFourKindsOnly(t *testing.T) {
	never := func(string, int64, string) bool { return false }
	no := func(string) bool { return false }
	in := AssetAuditInput{
		ProjectTag: "GER", HasDependencies: true,
		Ideas:      []AssetAuditEntity{{ID: 1, IdeaID: "i", Picture: "GFX_idea_x"}},
		Events:     []AssetAuditEntity{{ID: 2, EventID: "e", Picture: "GFX_e"}},
		Decisions:  []AssetAuditEntity{{ID: 3, DecisionID: "d", Icon: "x"}},
		Techs:      []AssetAuditEntity{{ID: 4, TechID: "t"}},
		Characters: []AssetAuditEntity{{ID: 5, Tag: "GER_a", LargePortrait: "u.png"}},
		Trees:      []AssetAuditTree{{Nodes: []AssetAuditEntity{{FocusID: "f"}}}},
	}
	got := AuditProjectAssets(in, never, no, no)
	if got.IdeaPicturesMissing != 0 || got.EventPicturesMissing != 0 ||
		got.FocusIconsMissing != 0 || got.PortraitsMissing != 0 {
		t.Errorf("submod 下 idea/event/focus/portrait 应全部信任,得到 %+v", got)
	}
	if got.DecisionIconsMissing != 1 || got.TechIconsMissing != 1 {
		t.Errorf("submod 下 decision/tech 仍应计数,得到 %+v", got)
	}
	if got.TotalMissing != 2 {
		t.Errorf("total = %d want 2", got.TotalMissing)
	}
}

func TestEntityCountTablePolicy(t *testing.T) {
	for _, name := range []string{"entity_obs_paths", "project_obs_assets",
		"project_obs_icons", "project_file_baselines", "project_overlay_state"} {
		if !nonContentTables[name] {
			t.Errorf("%s 应在排除表里", name)
		}
	}
	if len(nonContentTables) != 5 {
		t.Errorf("排除表 %d 条,应为 5 —— 多排除一张内容表就是对作者说他没作品",
			len(nonContentTables))
	}
	want := map[string]string{"wargoal_types": "wargoals",
		"project_raw_files": "raw_files", "equipments": "equipment"}
	if !reflect.DeepEqual(countKeyAliases, want) {
		t.Errorf("键别名 = %v want %v(改它会改变 /export/validate 已发布的形状)",
			countKeyAliases, want)
	}

	for _, bad := range []string{"a b", "a;drop", "A", "1x", `a"b`, ""} {
		if sqlIdentifierRE.MatchString(bad) {
			t.Errorf("标识符白名单放行了 %q", bad)
		}
	}
	for _, ok := range []string{"characters", "focus_trees", "_x", "a1"} {
		if !sqlIdentifierRE.MatchString(ok) {
			t.Errorf("标识符白名单误拦了 %q", ok)
		}
	}
}

func TestSlugifyNFKDDivergenceIsKnown(t *testing.T) {
	got := ideaPictureFilename("fra", "Idée-Forte")
	const goValue = "idea_FRA_idée_forte.png"
	const pythonValue = "idea_FRA_idee_forte.png"
	if got != goValue {
		t.Errorf("Go 侧文件名 = %q want %q —— 若这里变了,说明 emit 的 Slugify "+
			"被改动过,应同步复核 emitter 输出与本条注释", got, goValue)
	}
	if goValue == pythonValue {
		t.Error("两侧已一致 —— 分歧被修好了,删掉这条测试并把非 ASCII 用例加回夹具")
	}
}
