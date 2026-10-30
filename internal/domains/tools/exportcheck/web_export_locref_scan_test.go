package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLocRefScanParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "locref_scan_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name    string `json:"name"`
			Sources []struct {
				EntityType string `json:"entity_type"`
				EntityID   string `json:"entity_id"`
				Field      string `json:"field"`
				Value      any    `json:"value"`
			} `json:"sources"`
			Defined []string        `json:"defined"`
			Issues  json.RawMessage `json:"issues"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	for _, c := range fx.Cases {
		var sources []ScriptLocSource
		for _, s := range c.Sources {
			sources = append(sources, ScriptLocSource{
				EntityType: s.EntityType, EntityID: s.EntityID,
				Field: s.Field, Value: s.Value})
		}
		defined := map[string]bool{}
		for _, d := range c.Defined {
			defined[d] = true
		}
		compareJSON(t, "locref "+c.Name,
			ScanMissingLocalisationReferences(sources, defined), c.Issues)
	}
}

func TestLocRefSpecsComplete(t *testing.T) {
	specs := locrefScanSpecs()
	if len(specs.Specs) != 12 {
		t.Errorf("spec 表 %d 条,应为 12", len(specs.Specs))
	}
	for _, spec := range specs.Specs {
		if locrefIDColumns[spec.Kind] == "" {
			t.Errorf("%s(表 %s)缺 id 列映射 —— 这张表会被整个跳过",
				spec.Kind, spec.Table)
		}
		if len(spec.Fields) == 0 {
			t.Errorf("%s 没有脚本字段", spec.Kind)
		}
	}
	if len(specs.SelfDefiningKinds) != 5 {
		t.Errorf("自带 loc map 的类别 %d 个,应为 5", len(specs.SelfDefiningKinds))
	}
}
