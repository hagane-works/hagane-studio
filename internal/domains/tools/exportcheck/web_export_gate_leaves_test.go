package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestExportGateLeavesParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "exportgate_leaves_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Normalize []struct {
			Input  []string `json:"input"`
			Output []string `json:"output"`
		} `json:"normalize"`
		Parse []struct {
			Text   string   `json:"text"`
			Output []string `json:"output"`
		} `json:"parse"`
		Effective []struct {
			Imported []string `json:"imported"`
			Selected []string `json:"selected"`
			Output   []string `json:"output"`
		} `json:"effective"`
		SubUnits []struct {
			Name             string   `json:"name"`
			ProjectTag       string   `json:"project_tag"`
			ImportedRawPaths []string `json:"imported_raw_paths"`
			Rows             []struct {
				SubUnitID  string   `json:"sub_unit_id"`
				UnitTypes  []string `json:"unit_types"`
				Categories []string `json:"categories"`
				SourceFile string   `json:"source_file"`
				Source     string   `json:"source"`
			} `json:"rows"`
			Issues json.RawMessage `json:"issues"`
		} `json:"sub_units"`
		SourceFiles []struct {
			Input  string  `json:"input"`
			Output *string `json:"output"`
			Error  *string `json:"error"`
		} `json:"source_files"`
		LocExtract []struct {
			Value json.RawMessage `json:"value"`
			Keys  []string        `json:"keys"`
		} `json:"loc_extract"`
		LocScan []struct {
			Name    string   `json:"name"`
			Defined []string `json:"defined"`
			Sources []struct {
				EntityType string          `json:"entity_type"`
				EntityID   string          `json:"entity_id"`
				Field      string          `json:"field"`
				Value      json.RawMessage `json:"value"`
			} `json:"sources"`
			Issues json.RawMessage `json:"issues"`
		} `json:"loc_scan"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	eqStrings := func(got, want []string) bool {
		if len(got) == 0 && len(want) == 0 {
			return true
		}
		return reflect.DeepEqual(got, want)
	}

	for i, c := range fx.Normalize {
		if got := NormalizeDescriptorReplacePaths(c.Input); !eqStrings(got, c.Output) {
			t.Errorf("normalize[%d] %v\n got %v\nwant %v", i, c.Input, got, c.Output)
		}
	}
	for i, c := range fx.Parse {
		if got := ParseDescriptorReplacePaths(c.Text); !eqStrings(got, c.Output) {
			t.Errorf("parse[%d] %q\n got %v\nwant %v", i, c.Text, got, c.Output)
		}
	}
	for i, c := range fx.Effective {
		got := EffectiveDescriptorReplacePaths(c.Imported, c.Selected)
		if !eqStrings(got, c.Output) {
			t.Errorf("effective[%d]\n got %v\nwant %v", i, got, c.Output)
		}
	}

	for _, c := range fx.SubUnits {
		rows := make([]SubUnitRow, 0, len(c.Rows))
		for _, r := range c.Rows {
			rows = append(rows, SubUnitRow{
				SubUnitID: r.SubUnitID, UnitTypes: r.UnitTypes,
				Categories: r.Categories, SourceFile: r.SourceFile, Source: r.Source,
			})
		}
		got := jsonNorm(t, CollectSubUnitExportIssues(rows, c.ProjectTag, c.ImportedRawPaths))
		var want any
		if err := json.Unmarshal(c.Issues, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("sub_units %s\n got %s\nwant %s", c.Name, gb, c.Issues)
		}
	}

	for _, c := range fx.SourceFiles {
		out, err := ValidateImportSourceFile(c.Input)
		if c.Error != nil {
			if err == nil {
				t.Errorf("source_file %q 应当报错 %q", c.Input, *c.Error)
			} else if err.Error() != *c.Error {
				t.Errorf("source_file %q\n got %q\nwant %q", c.Input, err.Error(), *c.Error)
			}
			continue
		}
		if err != nil {
			t.Errorf("source_file %q 意外报错 %v", c.Input, err)
		} else if c.Output != nil && out != *c.Output {
			t.Errorf("source_file %q = %q want %q", c.Input, out, *c.Output)
		}
	}

	for i, c := range fx.LocExtract {
		var value any
		if err := json.Unmarshal(c.Value, &value); err != nil {
			t.Fatal(err)
		}
		keys := ExtractLocalisationReferences(value)
		got := make([]string, 0, len(keys))
		for k := range keys {
			got = append(got, k)
		}
		sort.Strings(got)
		if !eqStrings(got, c.Keys) {
			t.Errorf("loc_extract[%d] %s\n got %v\nwant %v", i, c.Value, got, c.Keys)
		}
	}

	for _, c := range fx.LocScan {
		sources := make([]ScriptLocSource, 0, len(c.Sources))
		for _, s := range c.Sources {
			var v any
			if err := json.Unmarshal(s.Value, &v); err != nil {
				t.Fatal(err)
			}
			sources = append(sources, ScriptLocSource{
				EntityType: s.EntityType, EntityID: s.EntityID, Field: s.Field, Value: v,
			})
		}
		defined := map[string]bool{}
		for _, k := range c.Defined {
			defined[k] = true
		}
		got := jsonNorm(t, ScanMissingLocalisationReferences(sources, defined))
		var want any
		if err := json.Unmarshal(c.Issues, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("loc_scan %s\n got %s\nwant %s", c.Name, gb, c.Issues)
		}
	}
}
