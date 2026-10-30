package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type dnRow map[string]any

func (r dnRow) str(key string) string {
	s, _ := r[key].(string)
	return s
}

func buildKind(t *testing.T, kind string, rows []dnRow, idKey string) DisplayNameKind {
	t.Helper()
	sing, plur, literal, scoped, ok := DisplayNameKindSpec(kind)
	if !ok {
		t.Fatalf("未知 kind %q", kind)
	}
	out := DisplayNameKind{Kind: kind, Singular: sing, Plural: plur,
		LiteralAttr: literal, RowSourceScoped: scoped}
	for _, r := range rows {
		id := r.str(idKey)
		if kind == "doctrine" {
			id = DoctrineLocKey(r.str("name"))
		}
		out.Rows = append(out.Rows, DisplayNameRow{
			ID: id, Source: r.str("source"), Attrs: r,
		})
	}
	return out
}

func TestDisplayNamePlaceholderParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "displayname_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name  string `json:"name"`
			Trees []struct {
				Source string  `json:"source"`
				Nodes  []dnRow `json:"nodes"`
			} `json:"trees"`
			Techs              []dnRow         `json:"techs"`
			Ideas              []dnRow         `json:"ideas"`
			Decisions          []dnRow         `json:"decisions"`
			Equipments         []dnRow         `json:"equipments"`
			CustomTraits       []dnRow         `json:"custom_traits"`
			DynamicModifiers   []dnRow         `json:"dynamic_modifiers"`
			AutonomyStates     []dnRow         `json:"autonomy_states"`
			Bookmarks          []dnRow         `json:"bookmarks"`
			Mios               []dnRow         `json:"mios"`
			DecisionCategories []dnRow         `json:"decision_categories"`
			Doctrines          []dnRow         `json:"doctrines"`
			Issues             json.RawMessage `json:"issues"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	for _, c := range fx.Cases {
		var kinds []DisplayNameKind

		var focusRows []dnRow
		for _, tr := range c.Trees {
			src := tr.Source
			if src == "" {
				src = "created"
			}
			if src != "created" {
				continue
			}
			focusRows = append(focusRows, tr.Nodes...)
		}
		kinds = append(kinds, buildKind(t, "focus", focusRows, "focus_id"))
		kinds = append(kinds, buildKind(t, "technology", c.Techs, "tech_id"))
		kinds = append(kinds, buildKind(t, "idea", c.Ideas, "idea_id"))
		kinds = append(kinds, buildKind(t, "decision", c.Decisions, "decision_id"))
		kinds = append(kinds, buildKind(t, "equipment", c.Equipments, "equipment_id"))
		kinds = append(kinds, buildKind(t, "custom_trait", c.CustomTraits, "trait_id"))
		kinds = append(kinds, buildKind(t, "dynamic_modifier", c.DynamicModifiers, "dm_id"))
		kinds = append(kinds, buildKind(t, "autonomy_state", c.AutonomyStates, "autonomy_id"))
		kinds = append(kinds, buildKind(t, "bookmark", c.Bookmarks, "bookmark_id"))
		kinds = append(kinds, buildKind(t, "mio", c.Mios, "mio_id"))

		var categories []dnRow
		for _, d := range c.Decisions {
			src := d.str("source")
			if src == "" {
				src = "created"
			}
			if src == "created" {
				categories = c.DecisionCategories
				break
			}
		}
		kinds = append(kinds, buildKind(t, "decision_category", categories, "category_id"))
		kinds = append(kinds, buildKind(t, "doctrine", c.Doctrines, "name"))

		got := jsonNorm(t, validateDisplayNamePlaceholder(kinds))
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

func TestUnnamedIDsIsKeyedNotRowed(t *testing.T) {
	rows := []DisplayNameRow{
		{ID: "GD_X", Attrs: map[string]any{"display_name_locs": map[string]any{}}},
		{ID: "GD_X", Attrs: map[string]any{
			"display_name_locs": map[string]any{"en": "Real"}}},
	}
	if got := unnamedIDs(rows, "", true); len(got) != 0 {
		t.Errorf("同键有一行命名就不该报,得到 %v", got)
	}
	both := []DisplayNameRow{
		{ID: "GD_X", Attrs: map[string]any{"display_name_locs": map[string]any{}}},
		{ID: "GD_Y", Attrs: map[string]any{"display_name_locs": map[string]any{}}},
	}
	if got := unnamedIDs(both, "", true); len(got) != 2 {
		t.Errorf("两个不同键都无名应当报 2 条,得到 %v", got)
	}
}

func TestDoctrineLocKey(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"GRAND_DOCTRINE_X", "GRAND_DOCTRINE_X"},
		{"  padded  ", "padded"},
		{"", ""},
		{"Two Words", ""},
		{"   ", ""},
	} {
		if got := DoctrineLocKey(tc.in); got != tc.want {
			t.Errorf("DoctrineLocKey(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestCharacterAbsentFromDisplayNameTable(t *testing.T) {
	for _, banned := range []string{"character", "intelligence_agency", "state"} {
		if _, _, _, _, ok := DisplayNameKindSpec(banned); ok {
			t.Errorf("%q 刻意不该在占位名表里", banned)
		}
	}
}
