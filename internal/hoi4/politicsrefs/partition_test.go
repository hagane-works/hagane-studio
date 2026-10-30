package politicsrefs

import (
	"errors"
	"math/rand"
	"reflect"
	"testing"
)

func vanillaRoots() []Ideology {
	return []Ideology{
		{Kind: KindRoot, ScriptID: "democratic", Source: SourceVanilla},
		{Kind: KindRoot, ScriptID: "communism", Source: SourceVanilla},
		{Kind: KindRoot, ScriptID: "fascism", Source: SourceVanilla},
		{Kind: KindRoot, ScriptID: "neutrality", Source: SourceVanilla},
	}
}

func TestPartitionValidOnRealTestServerShapes(t *testing.T) {
	cases := []struct {
		name         string
		extra        []Ideology
		wantKeptIDs  []string
		wantConflict map[string]ConflictReason
	}{
		{

			name: "A_group_id_used_as_subtype",
			extra: []Ideology{
				{Kind: KindSubtype, ScriptID: "fascism", Source: SourceProjectManaged},
				{Kind: KindSubtype, ScriptID: "communism", Source: SourceProjectManaged},
			},
			wantKeptIDs: []string{"democratic", "communism", "fascism", "neutrality"},
			wantConflict: map[string]ConflictReason{
				"fascism":   ConflictDuplicateID,
				"communism": ConflictDuplicateID,
			},
		},
		{

			name: "B_subtype_without_parent_group",
			extra: []Ideology{
				{Kind: KindSubtype, ScriptID: "totalist", Source: SourceProjectManaged},
				{Kind: KindSubtype, ScriptID: "syndicalist", Source: SourceProjectManaged},
			},
			wantKeptIDs: []string{"democratic", "communism", "fascism", "neutrality"},
			wantConflict: map[string]ConflictReason{
				"totalist":    ConflictMissingRoot,
				"syndicalist": ConflictMissingRoot,
			},
		},
		{

			name: "C_parent_group_filled_in",
			extra: []Ideology{
				{Kind: KindSubtype, ScriptID: "totalist", RootID: "communism", Source: SourceProjectManaged},
			},
			wantKeptIDs:  []string{"democratic", "communism", "fascism", "neutrality", "totalist"},
			wantConflict: map[string]ConflictReason{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Snapshot{ProjectID: "p", Ideologies: append(vanillaRoots(), tc.extra...)}

			if tc.name != "C_parent_group_filled_in" {
				if err := ValidateSnapshot(in); err == nil {
					t.Fatalf("前提不成立:%s 本应触发 ValidateSnapshot 失败", tc.name)
				}
			}

			got := PartitionValid(in)
			if err := ValidateSnapshot(got); err != nil {
				t.Fatalf("清洗后仍不合契约: %v", err)
			}

			var keptIDs []string
			for _, ideology := range got.Ideologies {
				keptIDs = append(keptIDs, ideology.ScriptID)
			}
			if !reflect.DeepEqual(keptIDs, tc.wantKeptIDs) {
				t.Errorf("保留的 id = %v, want %v", keptIDs, tc.wantKeptIDs)
			}

			if len(got.Conflicts) != len(tc.wantConflict) {
				t.Fatalf("冲突条数 = %d (%v), want %d", len(got.Conflicts), got.Conflicts, len(tc.wantConflict))
			}
			for _, conflict := range got.Conflicts {
				want, ok := tc.wantConflict[conflict.ScriptID]
				if !ok {
					t.Errorf("多出冲突 %q", conflict.ScriptID)
					continue
				}
				if conflict.Reason != want {
					t.Errorf("%q 的原因 = %q, want %q", conflict.ScriptID, conflict.Reason, want)
				}
				if conflict.Detail == "" {
					t.Errorf("%q 没有给出 detail —— 作者看不懂要修什么", conflict.ScriptID)
				}
			}
		})
	}
}

func TestPartitionValidOutputAlwaysValidates(t *testing.T) {
	ids := []string{"democratic", "communism", "fascism", "neutrality",
		"totalist", "conservatism", "", "  ", "nazism", "despotism"}
	kinds := []IdeologyKind{KindRoot, KindSubtype, "", "bogus"}
	sources := []SourceOwnership{SourceVanilla, SourceProjectManaged,
		SourceImported, SourceBaseMod, "", "nonsense"}
	roots := []string{"", "democratic", "fascism", "does_not_exist", "  "}

	rng := rand.New(rand.NewSource(20260902))
	for i := 0; i < 4000; i++ {
		n := rng.Intn(7)
		snapshot := Snapshot{ProjectID: "p"}
		for j := 0; j < n; j++ {
			snapshot.Ideologies = append(snapshot.Ideologies, Ideology{
				Kind:     kinds[rng.Intn(len(kinds))],
				ScriptID: ids[rng.Intn(len(ids))],
				RootID:   roots[rng.Intn(len(roots))],
				Source:   sources[rng.Intn(len(sources))],
			})
		}
		if rng.Intn(2) == 0 {
			snapshot.References = append(snapshot.References, Reference{
				Target: Target{Kind: kinds[rng.Intn(len(kinds))], ScriptID: ids[rng.Intn(len(ids))]},
				Entity: EntityRef{Kind: "character", ID: "1", Path: "characters/x.txt"},
			})
		}
		if rng.Intn(3) == 0 {
			snapshot.Dependencies = append(snapshot.Dependencies, RootDependency{
				CustomRootID: ids[rng.Intn(len(ids))], ExpectedRoot: ids[rng.Intn(len(ids))],
				ReferenceKey: "has_government", Disposition: DependencyRequired,
				Entity: EntityRef{Kind: "country", ID: "ROM", Path: "history/countries/x.txt"},
			})
		}

		clean := PartitionValid(snapshot)
		if err := ValidateSnapshot(clean); err != nil {
			t.Fatalf("第 %d 次:清洗后仍不合契约: %v\n输入 = %#v\n输出 = %#v",
				i, err, snapshot, clean)
		}

		if again := PartitionValid(clean); !reflect.DeepEqual(again, clean) {
			t.Fatalf("第 %d 次:PartitionValid 不幂等\n一次 = %#v\n两次 = %#v", i, clean, again)
		}

		if repeat := PartitionValid(snapshot); !reflect.DeepEqual(repeat, clean) {
			t.Fatalf("第 %d 次:PartitionValid 不确定\nA = %#v\nB = %#v", i, clean, repeat)
		}
	}
}

func TestPartitionValidReportsEveryConflictNotJustTheFirst(t *testing.T) {
	in := Snapshot{ProjectID: "p", Ideologies: append(vanillaRoots(),
		Ideology{Kind: KindSubtype, ScriptID: "democratic", Source: SourceProjectManaged},
		Ideology{Kind: KindSubtype, ScriptID: "communism", Source: SourceProjectManaged},
		Ideology{Kind: KindSubtype, ScriptID: "fascism", Source: SourceProjectManaged},
		Ideology{Kind: KindSubtype, ScriptID: "neutrality", Source: SourceProjectManaged},
		Ideology{Kind: KindSubtype, ScriptID: "totalist", Source: SourceProjectManaged},
	)}
	got := PartitionValid(in)
	if len(got.Conflicts) != 5 {
		t.Fatalf("冲突条数 = %d, want 5(4 条重复 id + 1 条缺父组);实际 = %#v",
			len(got.Conflicts), got.Conflicts)
	}
}

func TestValidateSnapshotStillRejectsDirtyDataOnWritePaths(t *testing.T) {
	dirty := Snapshot{ProjectID: "p", Ideologies: append(vanillaRoots(),
		Ideology{Kind: KindSubtype, ScriptID: "fascism", Source: SourceProjectManaged},
	)}
	if err := ValidateSnapshot(dirty); !errors.Is(err, ErrDuplicateScriptID) {
		t.Fatalf("ValidateSnapshot 必须继续拒绝写入脏数据, got %v", err)
	}
	orphan := Snapshot{ProjectID: "p", Ideologies: append(vanillaRoots(),
		Ideology{Kind: KindSubtype, ScriptID: "totalist", Source: SourceProjectManaged},
	)}
	if err := ValidateSnapshot(orphan); !errors.Is(err, ErrInvalidSnapshot) {
		t.Fatalf("ValidateSnapshot 必须继续拒绝无父组的 subtype, got %v", err)
	}
}

func TestPartitionValidDropsReferencesToRejectedIdeologies(t *testing.T) {
	in := Snapshot{
		ProjectID:  "p",
		Ideologies: append(vanillaRoots(), Ideology{Kind: KindSubtype, ScriptID: "totalist", Source: SourceProjectManaged}),
		References: []Reference{
			{Target: Target{Kind: KindSubtype, ScriptID: "totalist"},
				Entity: EntityRef{Kind: "character", ID: "7", Path: "characters/ROM.txt"}},
			{Target: Target{Kind: KindRoot, ScriptID: "fascism"},
				Entity: EntityRef{Kind: "country", ID: "ROM", Path: "history/countries/ROM.txt"}},
		},
	}
	got := PartitionValid(in)
	if len(got.References) != 1 || got.References[0].Target.ScriptID != "fascism" {
		t.Fatalf("悬空引用没有被丢弃: %#v", got.References)
	}
}
