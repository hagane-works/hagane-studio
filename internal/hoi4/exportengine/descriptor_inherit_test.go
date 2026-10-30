package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func descriptorText(t *testing.T, artifact NativeArtifact) string {
	t.Helper()
	for _, entry := range artifact.Entries {
		if strings.HasSuffix(entry.Path, "/descriptor.mod") {
			return string(entry.Bytes)
		}
	}
	t.Fatal("no descriptor.mod entry")
	return ""
}

func inheritSnapshot(replacePathsJSON string) Snapshot {
	return Snapshot{
		ProjectID: 44,
		Project: json.RawMessage(`{"id":44,"name":"Demo","country_tag":"GER","imported_descriptor_replace_paths":` +
			replacePathsJSON + `}`),
		Tables: map[string][]json.RawMessage{},
	}
}

func TestDescriptorInheritsImportedReplacePaths(t *testing.T) {
	snapshot := inheritSnapshot(`["common/ideas","history/countries"]`)
	artifact, err := BuildNativeShadow(snapshot, []string{"common/units"}, false)
	if err != nil {
		t.Fatal(err)
	}
	descriptor := descriptorText(t, artifact)
	for _, want := range []string{
		`replace_path="common/ideas"`,
		`replace_path="history/countries"`,
		`replace_path="common/units"`,
	} {
		if !strings.Contains(descriptor, want) {
			t.Fatalf("descriptor missing %s:\n%s", want, descriptor)
		}
	}
}

func TestDescriptorInheritanceIgnoresIncludeImported(t *testing.T) {
	snapshot := inheritSnapshot(`["common/ideas"]`)
	for _, includeImported := range []bool{false, true} {
		artifact, err := BuildNativeShadow(snapshot, nil, includeImported)
		if err != nil {
			t.Fatal(err)
		}
		if descriptor := descriptorText(t, artifact); !strings.Contains(descriptor, `replace_path="common/ideas"`) {
			t.Fatalf("includeImported=%v dropped the inherited path:\n%s", includeImported, descriptor)
		}
	}
}

func TestInheritedReplacePathsTolerantOfStoredShapes(t *testing.T) {
	cases := map[string][]string{
		`["a/b","c/d"]`:  {"a/b", "c/d"},
		`null`:           nil,
		`[]`:             {},
		`"a/b"`:          {"a/b"},
		`"[\"a/b\"]"`:    {"a/b"},
		`""`:             nil,
		`{"not":"list"}`: nil,
	}
	for stored, want := range cases {
		got := InheritedReplacePaths(inheritSnapshot(stored))
		if len(got) != len(want) {
			t.Fatalf("stored=%s got=%v want=%v", stored, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("stored=%s got=%v want=%v", stored, got, want)
			}
		}
	}
}

func TestInheritedReplacePathsAbsentColumn(t *testing.T) {
	snapshot := Snapshot{ProjectID: 45, Project: json.RawMessage(`{"id":45,"name":"Demo","country_tag":"GER"}`)}
	if got := InheritedReplacePaths(snapshot); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
