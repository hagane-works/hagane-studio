package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPanelNamesParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "panel_names_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Derive []struct {
			Token   string            `json:"token"`
			Control *string           `json:"control"`
			Names   map[string]string `json:"names"`
		} `json:"derive"`
		Walk []struct {
			Name     string           `json:"name"`
			Controls []map[string]any `json:"controls"`
			Order    []string         `json:"order"`
		} `json:"walk"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Derive) == 0 {
		t.Fatal("fixture 为空")
	}
	for _, c := range fx.Derive {
		ctrl, with := "", false
		if c.Control != nil {
			ctrl, with = *c.Control, true
		}
		got := DerivePanelNames(c.Token, ctrl, with)
		if !reflect.DeepEqual(got, c.Names) {
			gb, _ := json.Marshal(got)
			wb, _ := json.Marshal(c.Names)
			t.Errorf("derive(%q,%v)\n got %s\nwant %s", c.Token, c.Control, gb, wb)
		}
	}
	for _, c := range fx.Walk {
		order := []string{}
		for _, ctrl := range WalkPanelControls(c.Controls) {
			order = append(order, anyToString(ctrl["name"]))
		}
		want := c.Order
		if want == nil {
			want = []string{}
		}
		if !reflect.DeepEqual(order, want) {
			t.Errorf("walk %s\n got %v\nwant %v", c.Name, order, want)
		}
	}
}

func TestWalkOnlyRecursesContainers(t *testing.T) {
	got := WalkPanelControls([]map[string]any{
		{"name": "a", "kind": "text", "children": []any{
			map[string]any{"name": "hidden"}}},
		{"name": "b", "kind": "CONTAINER", "children": []any{
			map[string]any{"name": "shown"}}},
	})
	names := []string{}
	for _, c := range got {
		names = append(names, anyToString(c["name"]))
	}
	want := []string{"a", "b", "shown"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("got %v want %v(kind 比对要忽略大小写与空白)", names, want)
	}
}
