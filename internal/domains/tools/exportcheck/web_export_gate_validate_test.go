package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestValidateProjectCoreParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "validate_core_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name    string `json:"name"`
			Project struct {
				ID         int64  `json:"id"`
				Name       string `json:"name"`
				CountryTag string `json:"country_tag"`
			} `json:"project"`
			Characters []struct {
				ID                int64          `json:"id"`
				Tag               string         `json:"tag"`
				CountryTag        string         `json:"country_tag"`
				RoleType          string         `json:"role_type"`
				Traits            []string       `json:"traits"`
				AdvisorTraits     []string       `json:"advisor_traits"`
				Source            string         `json:"source"`
				ReplacesVanillaID bool           `json:"replaces_vanilla_id"`
				DisplayNameLocs   map[string]any `json:"display_name_locs"`
				NameEn            string         `json:"name_en"`
				NameZh            string         `json:"name_zh"`
				Name              string         `json:"name"`
			} `json:"characters"`
			Trees []struct {
				ID       int64  `json:"id"`
				TreeID   string `json:"tree_id"`
				TreeKind string `json:"tree_kind"`
				Nodes    []struct {
					ID                int64      `json:"id"`
					FocusID           string     `json:"focus_id"`
					Prerequisites     [][]string `json:"prerequisites"`
					MutuallyExclusive []string   `json:"mutually_exclusive"`
				} `json:"nodes"`
			} `json:"trees"`
			Techs []struct {
				ID            int64    `json:"id"`
				TechID        string   `json:"tech_id"`
				Prerequisites []string `json:"prerequisites"`
			} `json:"techs"`
			Ideas []struct {
				ID            int64           `json:"id"`
				IdeaID        string          `json:"idea_id"`
				Modifier      json.RawMessage `json:"modifier"`
				IsDesigner    bool            `json:"is_designer"`
				RawExtras     json.RawMessage `json:"raw_extras"`
				ResearchBonus json.RawMessage `json:"research_bonus"`
			} `json:"ideas"`
			Events []struct {
				ID          int64           `json:"id"`
				EventID     string          `json:"event_id"`
				Options     json.RawMessage `json:"options"`
				Hidden      bool            `json:"hidden"`
				NamespaceID *int64          `json:"namespace_id"`
			} `json:"events"`
			Decisions []struct {
				ID         int64  `json:"id"`
				DecisionID string `json:"decision_id"`
			} `json:"decisions"`
			Issues json.RawMessage `json:"issues"`
		} `json:"cases"`
		CoreCodes []string `json:"core_codes"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	truthy := func(raw json.RawMessage) bool {
		if len(raw) == 0 {
			return false
		}
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return false
		}
		switch t := v.(type) {
		case nil:
			return false
		case bool:
			return t
		case string:
			return t != ""
		case float64:
			return t != 0
		case []any:
			return len(t) > 0
		case map[string]any:
			return len(t) > 0
		}
		return true
	}

	seenCodes := map[string]bool{}
	for _, c := range fx.Cases {
		in := ValidateInput{
			ProjectID: c.Project.ID, ProjectName: c.Project.Name,
			CountryTag: c.Project.CountryTag,
		}
		for _, ch := range c.Characters {
			in.Characters = append(in.Characters, ValidateCharacter{
				ID: ch.ID, Tag: ch.Tag, CountryTag: ch.CountryTag,
				RoleType: ch.RoleType, Traits: ch.Traits, AdvisorTraits: ch.AdvisorTraits,
				Source: ch.Source, ReplacesVanillaID: ch.ReplacesVanillaID,
				DisplayNameLocs: ch.DisplayNameLocs, NameEn: ch.NameEn,
				NameZh: ch.NameZh, Name: ch.Name,
			})
		}
		for _, tr := range c.Trees {
			vt := ValidateFocusTree{ID: tr.ID, TreeID: tr.TreeID, TreeKind: tr.TreeKind}
			for _, n := range tr.Nodes {
				vt.Nodes = append(vt.Nodes, ValidateFocusNode{
					ID: n.ID, FocusID: n.FocusID,
					Prerequisites: n.Prerequisites, MutuallyExclusive: n.MutuallyExclusive,
				})
			}
			in.Trees = append(in.Trees, vt)
		}
		for _, tc := range c.Techs {
			in.Techs = append(in.Techs, ValidateTech{
				ID: tc.ID, TechID: tc.TechID, Prerequisites: tc.Prerequisites})
		}
		for _, id := range c.Ideas {
			in.Ideas = append(in.Ideas, ValidateIdea{
				ID: id.ID, IdeaID: id.IdeaID,
				HasModifier: truthy(id.Modifier), IsDesigner: id.IsDesigner,
				HasRawExtras: truthy(id.RawExtras), HasResearchBonus: truthy(id.ResearchBonus),
			})
		}
		for _, ev := range c.Events {
			count := 0
			if len(ev.Options) > 0 {
				var opts []any
				_ = json.Unmarshal(ev.Options, &opts)
				count = len(opts)
			}
			in.Events = append(in.Events, ValidateEvent{
				ID: ev.ID, EventID: ev.EventID, NamespaceID: ev.NamespaceID,
				OptionCount: count, Hidden: ev.Hidden,
			})
		}
		for _, d := range c.Decisions {
			in.Decisions = append(in.Decisions, ValidateDecision{
				ID: d.ID, DecisionID: d.DecisionID})
		}

		got := jsonNorm(t, validateProjectCore(in))
		var want any
		if err := json.Unmarshal(c.Issues, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("case %s\n got %s\nwant %s", c.Name, gb, c.Issues)
		}
		var issues []struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(c.Issues, &issues)
		for _, i := range issues {
			seenCodes[i.Code] = true
		}
	}

	for _, code := range fx.CoreCodes {
		if !seenCodes[code] {
			t.Errorf("夹具没有覆盖 code %q", code)
		}
	}
}

func TestValidateProjectCoreRejectsAndMergeOfMutuallyExclusiveBranches(t *testing.T) {
	base := ValidateInput{Trees: []ValidateFocusTree{{
		ID: 9, TreeID: "per_iran_tree", Nodes: []ValidateFocusNode{
			{ID: 1, FocusID: "per_german", MutuallyExclusive: []string{"per_british"}},
			{ID: 2, FocusID: "per_british", MutuallyExclusive: []string{"per_german"}},
			{ID: 3, FocusID: "per_armor_school", Prerequisites: [][]string{{"per_german"}}},
			{ID: 4, FocusID: "per_refinery", Prerequisites: [][]string{{"per_british"}}},
			{ID: 5, FocusID: "per_arsenal", Prerequisites: [][]string{{"per_armor_school"}, {"per_refinery"}}},
		},
	}}}

	issues := validateProjectCore(base)
	found := false
	for _, issue := range issues {
		if issue["code"] == "focus_prereq_mutex_conflict" {
			found = true
			if issue["level"] != "error" {
				t.Fatalf("level = %v", issue["level"])
			}
		}
	}
	if !found {
		t.Fatalf("missing focus_prereq_mutex_conflict: %#v", issues)
	}

	base.Trees[0].Nodes[4].Prerequisites = [][]string{{"per_armor_school", "per_refinery"}}
	for _, issue := range validateProjectCore(base) {
		if issue["code"] == "focus_prereq_mutex_conflict" {
			t.Fatalf("one OR-group must remain playable: %#v", issue)
		}
	}
}

func TestCharacterMissingNameDelegates(t *testing.T) {
	blank := ValidateCharacter{ID: 1, Tag: "GER_x", Source: "created"}
	if !characterShowsRawTag(blank) {
		t.Error("完全没名字的 created 角色应当被判定为显示裸 tag")
	}
	imported := blank
	imported.Source = "imported"
	if characterShowsRawTag(imported) {
		t.Error("imported 行不 emit,名字不归我们管,不该报")
	}
	replaces := blank
	replaces.ReplacesVanillaID = true
	if characterShowsRawTag(replaces) {
		t.Error("replaces_vanilla_id 行刻意不写 name 行,HOI4 用自己的 loc,不该报")
	}
	named := blank
	named.NameEn = "Erwin Rommel"
	if characterShowsRawTag(named) {
		t.Error("有 name_en 的角色不该报")
	}
}
