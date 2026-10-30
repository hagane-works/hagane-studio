package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"haganeworks/internal/reftables"
)

func TestIdeologyRegistryParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "validate_ideology_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Registries []struct {
			Name              string          `json:"name"`
			Registry          json.RawMessage `json:"registry"`
			EffectiveGroupIDs []string        `json:"effective_group_ids"`
			EffectiveSubIDs   []string        `json:"effective_sub_ids"`
			CustomRootIDs     []string        `json:"custom_root_ids"`
			InvalidStandalone json.RawMessage `json:"invalid_standalone"`
			InvalidRoots      json.RawMessage `json:"invalid_roots"`
		} `json:"registries"`
		Issues []struct {
			Name          string          `json:"name"`
			Registry      json.RawMessage `json:"registry"`
			IsSubmod      bool            `json:"is_submod"`
			CharacterRefs []struct {
				Value string `json:"value"`
				Tag   string `json:"tag"`
			} `json:"character_refs"`
			HistoryRefs []struct {
				Value string `json:"value"`
				Tag   string `json:"tag"`
			} `json:"history_refs"`
			Issues json.RawMessage `json:"issues"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	regBytes := func(r json.RawMessage) []byte {
		if len(r) == 0 || string(r) == "null" {
			return nil
		}
		return r
	}
	sortedKeysOf := func(m map[string]bool) []string {
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}

	for _, c := range fx.Registries {
		reg := regBytes(c.Registry)
		if got := sortedKeysOf(EffectiveGroupIDs(reg)); !reflect.DeepEqual(got, c.EffectiveGroupIDs) {
			t.Errorf("%s effective_group_ids\n got %v\nwant %v", c.Name, got, c.EffectiveGroupIDs)
		}
		if got := sortedKeysOf(EffectiveSubIDs(reg)); !reflect.DeepEqual(got, c.EffectiveSubIDs) {
			t.Errorf("%s effective_sub_ids\n got %v\nwant %v", c.Name, got, c.EffectiveSubIDs)
		}
		gotRoots := []string{}
		for _, r := range normalizeIdeologyRegistry(reg).customRootDefinitions() {
			gotRoots = append(gotRoots, r.ID)
		}
		wantRoots := c.CustomRootIDs
		if wantRoots == nil {
			wantRoots = []string{}
		}
		if !reflect.DeepEqual(gotRoots, wantRoots) {
			t.Errorf("%s custom_root_ids\n got %v\nwant %v", c.Name, gotRoots, wantRoots)
		}
		for _, pair := range []struct {
			label string
			got   any
			want  json.RawMessage
		}{
			{"invalid_standalone", InvalidStandaloneSubIdeologies(reg), c.InvalidStandalone},
			{"invalid_roots", InvalidCustomRootDefinitions(reg), c.InvalidRoots},
		} {
			gotTree := jsonNorm(t, pair.got)
			var wantTree any
			if err := json.Unmarshal(pair.want, &wantTree); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(gotTree, wantTree) {
				gb, _ := json.Marshal(gotTree)
				t.Errorf("%s %s\n got %s\nwant %s", c.Name, pair.label, gb, pair.want)
			}
		}
	}

	for _, c := range fx.Issues {
		refs := []ValidateIdeologyRef{}
		for _, r := range c.CharacterRefs {
			refs = append(refs, ValidateIdeologyRef{
				Value: r.Value, Who: "character " + r.Tag,
				Ref: map[string]any{"type": "character", "id": r.Tag},
			})
		}
		for _, r := range c.HistoryRefs {
			refs = append(refs, ValidateIdeologyRef{
				Value: r.Value, Who: r.Tag + " ruling_party",
				Ref: map[string]any{"type": "history_file", "id": r.Tag},
			})
		}
		got := jsonNorm(t, validateIdeologyChecks(regBytes(c.Registry), c.IsSubmod, refs))
		var want any
		if err := json.Unmarshal(c.Issues, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("case %s\n got %s\nwant %s", c.Name, gb, c.Issues)
		}
	}
}

func TestCustomRootBraceEscapeBlocked(t *testing.T) {
	reg := []byte(`{"root_definitions":[{"id":"tech","body":` +
		`"types = { a = {} }\ncolor = { 1 2 3 }\n}\nevil = {"}]}`)
	invalid := InvalidCustomRootDefinitions(reg)
	if len(invalid) != 1 {
		t.Fatalf("逃逸的 body 应当被判非法,得到 %d 条:%v", len(invalid), invalid)
	}
	reason, _ := invalid[0]["reason"].(string)
	if reason == "" {
		t.Error("非法 ROOT 必须带 creator-facing reason")
	}
}

func TestSubmodRootBlocked(t *testing.T) {
	reg := []byte(`{"root_definitions":[{"id":"technocracy","body":` +
		`"types = {\n a = {}\n}\ncolor = { 1 2 3 }"}]}`)
	submod := validateIdeologyChecks(reg, true, nil)
	found := false
	for _, i := range submod {
		if i["code"] == "ideology_root_submod_blocked" {
			found = true
		}
		if i["code"] == "ideology_registry_parent_missing" {
			t.Error("submod 不该跑独立注册表父组检查")
		}
	}
	if !found {
		t.Errorf("submod 带自定义 ROOT 应当被拦:%v", submod)
	}
	standalone := validateIdeologyChecks(reg, false, nil)
	for _, i := range standalone {
		if i["code"] == "ideology_root_submod_blocked" {
			t.Error("独立项目不该报 submod 专属的那条")
		}
	}
}

func TestUndefinedIdeologyDedup(t *testing.T) {
	refs := []ValidateIdeologyRef{
		{Value: "junzheng", Who: "character A", Ref: map[string]any{"type": "character", "id": "A"}},
		{Value: "junzheng", Who: "character B", Ref: map[string]any{"type": "character", "id": "B"}},
	}
	got := validateIdeologyChecks(nil, false, refs)
	count := 0
	for _, i := range got {
		if i["code"] == "ideology_undefined_standalone" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("同一 token 多处引用应当只报一次,得到 %d", count)
	}
}

func TestIdeologyRegistryChecksWithAnEmptyTable(t *testing.T) {
	raw := []byte(`{"groups": ["fascism"], "sub_ideologies": [{"id": "rexism", "group": "fascism"}, {"id": "neo_fascism", "group": "fascism"}]}`)
	if invalid := InvalidStandaloneSubIdeologies(raw); len(invalid) != 0 {
		t.Fatalf("with the built-in table: %v", invalid)
	}
	defer reftables.Use(map[string][]byte{reftables.IdeologiesFile: []byte(`{"groups": [], "sub_ideologies": []}`)})()
	if groups := EffectiveGroupIDs(raw); !reflect.DeepEqual(groups, map[string]bool{"fascism": true}) {
		t.Errorf("groups = %v", groups)
	}
	if subs := EffectiveSubIDs(raw); !reflect.DeepEqual(subs, map[string]bool{"rexism": true, "neo_fascism": true}) {
		t.Errorf("subs = %v", subs)
	}
	if invalid := InvalidStandaloneSubIdeologies(raw); len(invalid) != 2 {
		t.Errorf("invalid = %v, want both sub-ideologies", invalid)
	}
}
