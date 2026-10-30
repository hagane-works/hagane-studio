package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var guardrailBaseSets = map[string]map[string]map[string]bool{
	"none":  {},
	"chars": {"character": {"BASE_CHR": true}},
	"tags":  {"country_tag": {"FRA": true, "ITA": true}},
	"ideo":  {"ideology": {"base_ideo": true}},
	"all": {
		"character": {"BASE_CHR": true}, "focus": {"base_focus": true},
		"idea": {"base_idea": true}, "decision": {"base_dec": true},
		"country_tag": {"FRA": true}, "ideology": {"base_ideo": true},
	},
}

func lookupFor(name string) BaseIndexLookup {
	sets := guardrailBaseSets[name]
	return func(entityType string) map[string]bool { return sets[entityType] }
}

func TestExportGuardrailsParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "export_guardrails_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Guardrails []struct {
			Name   string          `json:"name"`
			Base   string          `json:"base"`
			Issues json.RawMessage `json:"issues"`
		} `json:"guardrails"`
		Normalize []struct {
			Name   string          `json:"name"`
			Input  map[string]any  `json:"input"`
			Output json.RawMessage `json:"output"`
		} `json:"normalize"`
		Summary         json.RawMessage `json:"summary"`
		IncludeImported []struct {
			Name                string `json:"name"`
			Requested           bool   `json:"requested"`
			HasDependencies     bool   `json:"has_dependencies"`
			IsLoadedExistingMod bool   `json:"is_loaded_existing_mod"`
			HasImportedRows     bool   `json:"has_imported_rows"`
			Result              bool   `json:"result"`
		} `json:"include_imported"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	deps := []string{"Base"}
	run := map[string]func(base string) []ExportIssue{
		"collision-no-deps": func(b string) []ExportIssue {
			return GuardrailBaseCollision(GuardrailProject{ID: 7, CountryTag: "GER"},
				[]GuardrailCharacter{{Tag: "BASE_CHR", Source: "created", HasAdvisorRole: true}},
				nil, nil, nil, lookupFor(b))
		},
		"collision-character": func(b string) []ExportIssue {
			return GuardrailBaseCollision(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "BASE_CHR", Source: "created"}, {Tag: "MINE", Source: "created"}},
				nil, nil, nil, lookupFor(b))
		},
		"collision-imported-skipped": func(b string) []ExportIssue {
			return GuardrailBaseCollision(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "BASE_CHR", Source: "imported"}},
				nil, nil, nil, lookupFor(b))
		},
		"collision-focus": func(b string) []ExportIssue {
			return GuardrailBaseCollision(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				nil, []GuardrailTree{{Nodes: []GuardrailFocusNode{{"base_focus"}, {"mine"}}}},
				nil, nil, lookupFor(b))
		},
		"collision-idea-decision": func(b string) []ExportIssue {
			return GuardrailBaseCollision(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				nil, nil, []GuardrailIdea{{IdeaID: "base_idea", Source: "created"}},
				[]GuardrailDecision{{DecisionID: "base_dec", Source: "created"}}, lookupFor(b))
		},
		"collision-no-index-trusts": func(b string) []ExportIssue {
			return GuardrailBaseCollision(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "BASE_CHR", Source: "created"}}, nil, nil, nil, lookupFor(b))
		},
		"collision-partial-index": func(b string) []ExportIssue {
			return GuardrailBaseCollision(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "BASE_CHR", Source: "created"}},
				[]GuardrailTree{{Nodes: []GuardrailFocusNode{{"base_focus"}}}}, nil, nil, lookupFor(b))
		},

		"ideology-not-submod": func(b string) []ExportIssue {
			return GuardrailSubmodIdeology(GuardrailProject{ID: 7, CountryTag: "GER"},
				[]GuardrailCharacter{{Tag: "c1", Ideology: "weird"}}, nil, false, lookupFor(b))
		},
		"ideology-no-index-trusts": func(b string) []ExportIssue {
			return GuardrailSubmodIdeology(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "c1", Ideology: "weird"}}, nil, true, lookupFor(b))
		},
		"ideology-undefined": func(b string) []ExportIssue {
			return GuardrailSubmodIdeology(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "c1", Ideology: "weird"}}, nil, true, lookupFor(b))
		},
		"ideology-base-known": func(b string) []ExportIssue {
			return GuardrailSubmodIdeology(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "c1", Ideology: "base_ideo"}}, nil, true, lookupFor(b))
		},
		"ideology-vanilla-known": func(b string) []ExportIssue {
			return GuardrailSubmodIdeology(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "c1", Ideology: "fascism"}}, nil, true, lookupFor(b))
		},
		"ideology-dedup": func(b string) []ExportIssue {
			return GuardrailSubmodIdeology(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "a", Ideology: "weird"}, {Tag: "b", Ideology: "weird"}},
				nil, true, lookupFor(b))
		},
		"ideology-history-ruling-party": func(b string) []ExportIssue {
			return GuardrailSubmodIdeology(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				nil, []GuardrailHistoryFile{{CountryTag: "GER", Source: "created",
					SetPoliticsRulingParty: "strange"}}, true, lookupFor(b))
		},
		"ideology-blank-skipped": func(b string) []ExportIssue {
			return GuardrailSubmodIdeology(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "c1", Ideology: "   "}}, nil, true, lookupFor(b))
		},

		"tagwrites-not-submod": func(b string) []ExportIssue {
			return GuardrailBaseTagWrites(GuardrailProject{ID: 7, CountryTag: "GER"},
				[]GuardrailHistoryFile{{CountryTag: "FRA", Source: "created"}}, nil, false, lookupFor(b))
		},
		"tagwrites-no-index": func(b string) []ExportIssue {
			return GuardrailBaseTagWrites(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailHistoryFile{{CountryTag: "FRA", Source: "created"}}, nil, true, lookupFor(b))
		},
		"tagwrites-history": func(b string) []ExportIssue {
			return GuardrailBaseTagWrites(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailHistoryFile{{CountryTag: "FRA", Source: "created"},
					{CountryTag: "GER", Source: "created"}}, nil, true, lookupFor(b))
		},
		"tagwrites-imported-skipped": func(b string) []ExportIssue {
			return GuardrailBaseTagWrites(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailHistoryFile{{CountryTag: "FRA", Source: "imported"}}, nil, true, lookupFor(b))
		},
		"tagwrites-division": func(b string) []ExportIssue {
			return GuardrailBaseTagWrites(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				nil, []GuardrailDivisionTemplate{{CountryTag: "ITA", Source: "created",
					TemplateName: "Elite", OOBFile: "ita.txt"}}, true, lookupFor(b))
		},

		"recruit-no-deps": func(b string) []ExportIssue {
			return GuardrailRecruitRoleFilter(GuardrailProject{ID: 7, CountryTag: "GER"},
				[]GuardrailCharacter{{Tag: "c1", RoleType: "advisor", CountryTag: "FRA",
					HasAdvisorRole: true}}, lookupFor(b))
		},
		"recruit-pure-advisor-base": func(b string) []ExportIssue {
			return GuardrailRecruitRoleFilter(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "adv", RoleType: "advisor", CountryTag: "FRA",
					HasAdvisorRole: true}}, lookupFor(b))
		},
		"recruit-slotless-skipped": func(b string) []ExportIssue {
			return GuardrailRecruitRoleFilter(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "c1", RoleType: "advisor", CountryTag: "FRA",
					HasAdvisorRole: false}}, lookupFor(b))
		},
		"recruit-non-advisor": func(b string) []ExportIssue {
			return GuardrailRecruitRoleFilter(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "c1", RoleType: "general", CountryTag: "FRA",
					HasAdvisorRole: true}}, lookupFor(b))
		},
		"recruit-non-base-tag": func(b string) []ExportIssue {
			return GuardrailRecruitRoleFilter(GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "c1", RoleType: "advisor", CountryTag: "GER",
					HasAdvisorRole: true}}, lookupFor(b))
		},
		"recruit-falls-back-to-project-tag": func(b string) []ExportIssue {
			return GuardrailRecruitRoleFilter(GuardrailProject{ID: 7, CountryTag: "FRA", Dependencies: deps},
				[]GuardrailCharacter{{Tag: "c1", RoleType: "advisor", HasAdvisorRole: true}}, lookupFor(b))
		},

		"techdoc-submod-silent": func(b string) []ExportIssue {
			return GuardrailTechAndDoctrine(GuardrailProject{ID: 7, Dependencies: deps},
				[]GuardrailTech{{TechID: "t1", Folder: "weird_folder"}}, nil, true)
		},
		"techdoc-unknown-folder": func(b string) []ExportIssue {
			return GuardrailTechAndDoctrine(GuardrailProject{ID: 7},
				[]GuardrailTech{{TechID: "t1", Folder: "weird_folder"},
					{TechID: "t2", Folder: "infantry_folder"}}, nil, false)
		},
		"techdoc-folder-warned-once": func(b string) []ExportIssue {
			return GuardrailTechAndDoctrine(GuardrailProject{ID: 7},
				[]GuardrailTech{{TechID: "t1", Folder: "weird"},
					{TechID: "t2", Folder: "weird"}}, nil, false)
		},
		"techdoc-track-unresolved": func(b string) []ExportIssue {
			return GuardrailTechAndDoctrine(GuardrailProject{ID: 7}, nil,
				[]GuardrailDoctrine{{EntityType: "subdoctrine", EntityKey: "sd1",
					TrackKey: "ghost_track"}}, false)
		},
		"techdoc-track-defined": func(b string) []ExportIssue {
			return GuardrailTechAndDoctrine(GuardrailProject{ID: 7}, nil,
				[]GuardrailDoctrine{{EntityType: "track", EntityKey: "my_track"},
					{EntityType: "subdoctrine", EntityKey: "sd1", TrackKey: "my_track"}}, false)
		},
		"techdoc-track-vanilla": func(b string) []ExportIssue {
			return GuardrailTechAndDoctrine(GuardrailProject{ID: 7}, nil,
				[]GuardrailDoctrine{{EntityType: "subdoctrine", EntityKey: "sd1",
					TrackKey: "armor"}}, false)
		},
	}

	if len(fx.Guardrails) == 0 {
		t.Fatal("fixture 为空")
	}
	for _, c := range fx.Guardrails {
		fn, ok := run[c.Name]
		if !ok {
			t.Fatalf("case %s 没有对应的 Go 调用 —— 生成脚本加了 case 就必须在这里加一条", c.Name)
		}
		compareJSON(t, "guardrail "+c.Name, fn(c.Base), c.Issues)
	}

	for _, c := range fx.Normalize {
		got := NormalizeExportIssue(c.Input)
		if strings.TrimSpace(string(c.Output)) == "null" {
			if got != nil {
				gb, _ := json.Marshal(got)
				t.Errorf("normalize %s 应当被丢弃,得到 %s", c.Name, gb)
			}
			continue
		}
		if got == nil {
			t.Errorf("normalize %s 不该被丢弃", c.Name)
			continue
		}
		compareJSON(t, "normalize "+c.Name, got, c.Output)
	}

	kept := []ExportIssue{}
	for _, c := range fx.Normalize {
		if n := NormalizeExportIssue(c.Input); n != nil {
			kept = append(kept, n)
		}
	}
	compareJSON(t, "summary", SummarizeExportIssues(kept), fx.Summary)

	for _, c := range fx.IncludeImported {
		got := EffectiveIncludeImported(c.Requested, c.HasDependencies,
			c.IsLoadedExistingMod, c.HasImportedRows)
		if got != c.Result {
			t.Errorf("include_imported %s = %v want %v", c.Name, got, c.Result)
		}
	}
}

func TestGuardrailsTrustWhenNoBaseIndex(t *testing.T) {
	empty := func(string) map[string]bool { return nil }
	p := GuardrailProject{ID: 7, CountryTag: "GER", Dependencies: []string{"Base"}}
	chars := []GuardrailCharacter{{Tag: "ANY", Ideology: "totally_made_up",
		RoleType: "advisor", CountryTag: "FRA", HasAdvisorRole: true}}
	hist := []GuardrailHistoryFile{{CountryTag: "FRA", SetPoliticsRulingParty: "nope"}}

	for name, got := range map[string][]ExportIssue{
		"base_collision":  GuardrailBaseCollision(p, chars, nil, nil, nil, empty),
		"submod_ideology": GuardrailSubmodIdeology(p, chars, hist, true, empty),
		"base_tag_writes": GuardrailBaseTagWrites(p, hist, nil, true, empty),
		"recruit_filter":  GuardrailRecruitRoleFilter(p, chars, empty),
	} {
		if len(got) != 0 {
			t.Errorf("%s 在缺 base 索引时必须信任,得到 %d 条", name, len(got))
		}
	}
}

func TestNormalizeDropsInfoAndSuccess(t *testing.T) {
	for _, level := range []string{"info", "success"} {
		if got := NormalizeExportIssue(map[string]any{
			"level": level, "code": "c", "severity": "blocker"}); got != nil {
			t.Errorf("level=%s 必须被丢弃,得到 %v", level, got)
		}
	}
	if got := NormalizeExportIssue(map[string]any{
		"level": "error", "code": "c", "severity": "blocker"}); got == nil {
		t.Error("level=error 不该被丢弃")
	}
}

func TestNormalizeOmitsFalsyKeys(t *testing.T) {
	got := NormalizeExportIssue(map[string]any{
		"code": "c", "severity": "blocker",
		"source": map[string]any{}, "evidence": map[string]any{},
		"repair_route": "", "out_of_stage": false,
	})
	for _, key := range []string{"source", "evidence", "repair_route", "out_of_stage"} {
		if _, present := got[key]; present {
			t.Errorf("空值的 %q 必须整键省略", key)
		}
	}
	full := NormalizeExportIssue(map[string]any{
		"code": "c", "severity": "blocker",
		"source": map[string]any{"kind": "k"}, "evidence": map[string]any{"e": 1},
		"repair_route": "r", "out_of_stage": true,
	})
	for _, key := range []string{"source", "evidence", "repair_route", "out_of_stage"} {
		if _, present := full[key]; !present {
			t.Errorf("有值的 %q 必须保留", key)
		}
	}
	if !reflect.DeepEqual(full["out_of_stage"], true) {
		t.Error("out_of_stage 应当归一成 true 而非原值")
	}
}
