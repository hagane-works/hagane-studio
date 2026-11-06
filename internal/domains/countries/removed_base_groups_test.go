package countries

import (
	"reflect"
	"sort"
	"testing"

	"haganeworks/internal/hoi4/ideologysource"
)

func TestRemovedBaseGameGroupsFollowTheModsIdeologyFile(t *testing.T) {
	discovery := ideologysource.Discovery{Roots: []ideologysource.RootOwner{{RootID: "fascism"}, {RootID: "solidarism"}}}
	layer := func(replace []string, paths ...string) []ideologysource.Layer {
		files := []ideologysource.SourceFile{}
		for _, path := range paths {
			files = append(files, ideologysource.SourceFile{Path: path})
		}
		return []ideologysource.Layer{
			{SourceKind: ideologysource.SourceVanilla},
			{SourceKind: ideologysource.SourceDependency, ReplacePaths: replace, Files: files},
		}
	}
	for name, tc := range map[string]struct {
		layers []ideologysource.Layer
		want   []any
	}{
		"same-name file":       {layer(nil, "common/ideologies/00_ideologies.txt"), []any{"communism", "democratic", "neutrality"}},
		"replace_path":         {layer([]string{"common/ideologies"}, "common/ideologies/mine.txt"), []any{"communism", "democratic", "neutrality"}},
		"beside the base game": {layer(nil, "common/ideologies/zz_mine.txt"), nil},
	} {
		got := removedBaseGameGroups(tc.layers, discovery)
		sort.Slice(got, func(i, j int) bool { return got[i].(string) < got[j].(string) })
		if !reflect.DeepEqual(got, tc.want) && !(len(got) == 0 && len(tc.want) == 0) {
			t.Errorf("%s: removed = %v, want %v", name, got, tc.want)
		}
	}
}
