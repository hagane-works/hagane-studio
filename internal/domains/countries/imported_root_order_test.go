package countries

import (
	"strings"
	"testing"

	"haganeworks/internal/hoi4/ideologysource"
)

// The game lists ideology groups in the order a mod defines them, left to
// right on the political spectrum. Import keeps that order instead of
// sorting the groups by ID.
func TestImportedIdeologyGroupsKeepSourceOrder(t *testing.T) {
	source := `ideologies = {
	totalitarian_socialism = { types = { stalinism = { } } color = { 1 0 0 } }
	authoritarian_socialism = { types = { leninism = { } } color = { 2 0 0 } }
	liberalism_group = { types = { liberalism = { } } color = { 3 0 0 } }
	bonapartism = { types = { caesarism = { } } color = { 4 0 0 } }
}
`
	discovery, err := ideologysource.Discover([]ideologysource.Layer{
		{SourceKind: ideologysource.SourceVanilla, LoadOrder: 0},
		{SourceKind: ideologysource.SourceDependency, DependencyKey: "imported", LoadOrder: 1,
			ReplacePaths: []string{"common/ideologies"},
			Files:        []ideologysource.SourceFile{{Path: "common/ideologies/00_ideologies.txt", Bytes: []byte(source)}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := mergeImportedPoliticsSource(map[string]any{}, discovery)
	if err != nil {
		t.Fatal(err)
	}
	want := "totalitarian_socialism authoritarian_socialism liberalism_group bonapartism"
	var roots, groups []string
	for _, raw := range updated["root_definitions"].([]any) {
		roots = append(roots, raw.(map[string]any)["id"].(string))
	}
	for _, raw := range updated["groups"].([]any) {
		groups = append(groups, raw.(string))
	}
	if got := strings.Join(roots, " "); got != want {
		t.Errorf("root_definitions order = %s, want %s", got, want)
	}
	if got := strings.Join(groups, " "); got != want {
		t.Errorf("groups order = %s, want %s", got, want)
	}
}
