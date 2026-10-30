package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestContentLanguageClosureParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "content_closure_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name            string          `json:"name"`
			ExportLanguages []string        `json:"export_languages"`
			Groups          json.RawMessage `json:"groups"`
			Expected        json.RawMessage `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Cases) == 0 {
		t.Fatal("fixture 为空")
	}

	for _, c := range fx.Cases {
		var audit ContentLanguageClosureAudit

		var asMap map[string]any
		if err := json.Unmarshal(c.Groups, &asMap); err == nil {
			audit = AuditContentLanguageClosure(c.ExportLanguages, asMap)
		} else {
			var asList []map[string]any
			if err := json.Unmarshal(c.Groups, &asList); err != nil {
				t.Fatalf("case %s: groups 既不是 dict 也不是 list: %v", c.Name, err)
			}
			audit = AuditContentLanguageClosureGeneric(c.ExportLanguages, asList)
		}

		got := jsonNorm(t, map[string]any{
			"enabled_languages":   audit.EnabledLanguages,
			"populated_languages": audit.PopulatedLanguages,
			"missing_languages":   audit.MissingLanguages,
			"is_closed":           audit.IsClosed(),
			"diagnostics":         audit.Diagnostics,
		})
		var want any
		if err := json.Unmarshal(c.Expected, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.MarshalIndent(got, "", " ")
			t.Errorf("case %s\n got %s\nwant %s", c.Name, gb, c.Expected)
		}
	}
}

func TestEnglishAlwaysEnabled(t *testing.T) {
	for _, langs := range [][]string{nil, {}, {"zh"}, {"ZH", " ru "}} {
		got := normaliseEnabledLanguages(langs)
		if len(got) == 0 || got[0] != "en" {
			t.Errorf("normaliseEnabledLanguages(%v) = %v,en 必须恒在且排第一", langs, got)
		}
	}
}

func TestContentSpecsLoaded(t *testing.T) {
	data := contentLanguageSpecsData()
	if len(data.ContentLocales) != 10 {
		t.Errorf("content_locales = %d,应为 10", len(data.ContentLocales))
	}
	if len(data.EntitySpecs) < 20 {
		t.Errorf("entity_specs = %d,明显偏少", len(data.EntitySpecs))
	}
	if len(data.GroupEntitySpecs) < 20 {
		t.Errorf("group_entity_specs = %d,明显偏少", len(data.GroupEntitySpecs))
	}
	if _, ok := data.EntitySpecs["generic"]; !ok {
		t.Error("缺 generic 规格 —— 非 Mapping 入参那一支会整个失效")
	}
}
