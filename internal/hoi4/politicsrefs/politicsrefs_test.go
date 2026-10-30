package politicsrefs

import (
	"errors"
	"reflect"
	"testing"
)

func testSnapshot() Snapshot {
	return Snapshot{
		ProjectID: "p0-politics",
		Ideologies: []Ideology{
			{Kind: KindRoot, ScriptID: "democratic", DisplayName: "Democratic", Source: SourceVanilla},
			{Kind: KindRoot, ScriptID: "fascism", DisplayName: "Fascism", Source: SourceBaseMod},
			{Kind: KindRoot, ScriptID: "imported_republic", DisplayName: "Imported Republic", Source: SourceImported},
			{Kind: KindRoot, ScriptID: "civic_coalition", DisplayName: "Civic Coalition", Source: SourceProjectManaged},
			{Kind: KindSubtype, ScriptID: "social_liberalism", RootID: "democratic", DisplayName: "Social Liberalism", Source: SourceImported},
			{Kind: KindSubtype, ScriptID: "civic_pluralism", RootID: "civic_coalition", DisplayName: "Civic Pluralism", Source: SourceProjectManaged},
			{Kind: KindSubtype, ScriptID: "new_liberalism", RootID: "democratic", DisplayName: "New Liberalism", Source: SourceProjectManaged},
		},
		References: []Reference{
			{
				Target:  Target{Kind: KindSubtype, ScriptID: "civic_pluralism"},
				Entity:  EntityRef{Kind: "character", ID: "TST_leader", Path: "characters/TST.txt:TST_leader.ideology"},
				Context: "country_leader.ideology",
			},
			{
				Target:  Target{Kind: KindRoot, ScriptID: "civic_coalition"},
				Entity:  EntityRef{Kind: "history_file", ID: "TST", Path: "history/countries/TST.txt:set_politics.ruling_party"},
				Context: "country history ruling ROOT",
			},
		},
		Dependencies: []RootDependency{
			{
				CustomRootID: "civic_coalition", ExpectedRoot: "fascism",
				ReferenceKey: "military_name_group", Disposition: DependencyRequired,
				Entity: EntityRef{Kind: "military_name_group", ID: "GER_MOT_02", Path: "common/units/names_divisions/GER_names_divisions.txt:GER_MOT_02"},
			},
			{
				CustomRootID: "civic_coalition", ExpectedRoot: "fascism",
				ReferenceKey: "has_government", Disposition: DependencyIntentional,
				Entity: EntityRef{Kind: "focus", ID: "TST_old_guard", Path: "common/national_focus/TST.txt:TST_old_guard.available.has_government"},
			},
		},
	}
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	clone := snapshot
	clone.Ideologies = append([]Ideology(nil), snapshot.Ideologies...)
	clone.References = append([]Reference(nil), snapshot.References...)
	clone.Dependencies = append([]RootDependency(nil), snapshot.Dependencies...)
	return clone
}

func TestSourceOwnershipIsImmutableAndP0TakeoverIsForbidden(t *testing.T) {
	snapshot := testSnapshot()
	for _, target := range []Target{
		{Kind: KindRoot, ScriptID: "democratic"},
		{Kind: KindRoot, ScriptID: "fascism"},
		{Kind: KindRoot, ScriptID: "imported_republic"},
		{Kind: KindSubtype, ScriptID: "social_liberalism"},
	} {
		ideology, found := findIdeology(snapshot, target)
		if !found {
			t.Fatalf("missing fixture target %#v", target)
		}
		if decision := OwnershipFor(ideology, ActionDelete); decision.Allowed {
			t.Fatalf("source-owned target unexpectedly deletable: %#v", decision)
		}
		if decision := OwnershipFor(ideology, ActionTakeover); decision.Allowed {
			t.Fatalf("source-owned target unexpectedly takeover-able: %#v", decision)
		}
	}
	if _, err := NewManager().PreviewDelete(snapshot, Target{Kind: KindRoot, ScriptID: "democratic"}); !errors.Is(err, ErrImmutableSource) {
		t.Fatalf("vanilla ROOT preview error = %v, want ErrImmutableSource", err)
	}
	projectRoot, _ := findIdeology(snapshot, Target{Kind: KindRoot, ScriptID: "civic_coalition"})
	if !OwnershipFor(projectRoot, ActionDelete).Allowed {
		t.Fatal("project-managed ROOT must have safe-delete capability")
	}
	if OwnershipFor(projectRoot, ActionTakeover).Allowed {
		t.Fatal("an already project-owned ROOT must not expose a takeover shortcut")
	}
}

func TestSubtypeDisplayNameAndStableScriptIDContract(t *testing.T) {
	snapshot := testSnapshot()
	target := Target{Kind: KindSubtype, ScriptID: "new_liberalism"}
	proposal := Ideology{
		Kind: KindSubtype, ScriptID: "new_liberalism", RootID: "democratic",
		DisplayName: "Social Liberalism",
		Source:      SourceProjectManaged,
	}
	if err := ValidateEdit(snapshot, target, proposal); err != nil {
		t.Fatalf("valid custom subtype edit rejected: %v", err)
	}
	proposal.ScriptID = "renewed_liberalism"
	if err := ValidateEdit(snapshot, target, proposal); !errors.Is(err, ErrStableScriptID) {
		t.Fatalf("persisted script id change error = %v, want ErrStableScriptID", err)
	}
	proposal.ScriptID = "social_liberalism"
	if err := ValidateEdit(snapshot, target, proposal); !errors.Is(err, ErrStableScriptID) {
		t.Fatalf("persisted script id replacement error = %v, want ErrStableScriptID", err)
	}
	vanillaRoot := Target{Kind: KindRoot, ScriptID: "democratic"}
	if err := ValidateEdit(snapshot, vanillaRoot, Ideology{
		Kind: KindRoot, ScriptID: "democratic_v2", DisplayName: "Democratic", Source: SourceProjectManaged,
	}); !errors.Is(err, ErrImmutableSource) {
		t.Fatalf("source ROOT edit error = %v, want ErrImmutableSource", err)
	}
}

func TestReferenceCatalogAggregatesRootAndSubtypeWithStableOrdering(t *testing.T) {
	snapshot := testSnapshot()

	snapshot.References = []Reference{snapshot.References[1], snapshot.References[0], snapshot.References[1]}
	refs, err := ListReferences(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("reference count = %d, want 2: %#v", len(refs), refs)
	}
	if refs[0].Target.Kind != KindRoot || refs[0].Entity.Kind != "history_file" {
		t.Fatalf("unexpected deterministic first reference: %#v", refs[0])
	}
	impacts, err := AggregateReferences(snapshot, Target{Kind: KindRoot, ScriptID: "civic_coalition"})
	if err != nil {
		t.Fatal(err)
	}
	if len(impacts) != 3 {
		t.Fatalf("ROOT impacts = %d, want 3: %#v", len(impacts), impacts)
	}
	for _, impact := range impacts {
		if impact.Entity.Kind == "" || impact.Entity.ID == "" || impact.Entity.Path == "" {
			t.Fatalf("impact lost object/path identity: %#v", impact)
		}
	}
}

func TestPreviewApplyBlocksReferencesAndNeverRewrites(t *testing.T) {
	snapshot := testSnapshot()
	manager := NewManager()
	preview, err := manager.PreviewDelete(snapshot, Target{Kind: KindSubtype, ScriptID: "civic_pluralism"})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Allowed || len(preview.Impacts) != 1 || preview.AutomaticRewrites {
		t.Fatalf("unsafe preview: %#v", preview)
	}
	authorization, err := manager.ApplyDelete(snapshot, preview.Token)
	if !errors.Is(err, ErrDeleteBlocked) {
		t.Fatalf("apply error = %v, want ErrDeleteBlocked", err)
	}
	if authorization.Authorized || len(authorization.Impacts) != 1 || authorization.AutomaticRewrites {
		t.Fatalf("blocked apply lost display impacts or requested rewrite: %#v", authorization)
	}
	if _, err := manager.ApplyDelete(snapshot, preview.Token); !errors.Is(err, ErrUnknownPreview) {
		t.Fatalf("preview token was not one-time: %v", err)
	}
	if err := RewriteReferences(snapshot, preview.Target, preview.Impacts); !errors.Is(err, ErrAutomaticRewrite) {
		t.Fatalf("rewrite tripwire error = %v, want ErrAutomaticRewrite", err)
	}
}

func TestRootPreviewBlocksChildSubtypeInsteadOfCascading(t *testing.T) {
	snapshot := testSnapshot()
	preview, err := NewManager().PreviewDelete(snapshot, Target{Kind: KindRoot, ScriptID: "civic_coalition"})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Allowed || preview.AutomaticRewrites {
		t.Fatalf("ROOT with child/references must be blocked without cascade: %#v", preview)
	}
	foundChild := false
	for _, impact := range preview.Impacts {
		if impact.Entity.Kind == "ideology_subtype" && impact.Entity.ID == "civic_pluralism" {
			foundChild = true
		}
	}
	if !foundChild {
		t.Fatalf("ROOT preview did not expose child subtype impact: %#v", preview.Impacts)
	}
}

func TestPreviewApplyAuthorizesOnlyExactUnreferencedTarget(t *testing.T) {
	snapshot := testSnapshot()
	manager := NewManager()
	preview, err := manager.PreviewDelete(snapshot, Target{Kind: KindSubtype, ScriptID: "new_liberalism"})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Allowed || len(preview.Impacts) != 0 {
		t.Fatalf("unreferenced project subtype should be allowed: %#v", preview)
	}
	authorization, err := manager.ApplyDelete(snapshot, preview.Token)
	if err != nil {
		t.Fatal(err)
	}
	if !authorization.Authorized || authorization.Target != preview.Target || authorization.AutomaticRewrites {
		t.Fatalf("unexpected authorization: %#v", authorization)
	}

	if _, found := findIdeology(snapshot, preview.Target); !found {
		t.Fatal("domain apply mutated the caller snapshot")
	}
}

func TestStaleAndOneTimePreviewToken(t *testing.T) {
	snapshot := testSnapshot()
	manager := NewManager()
	preview, err := manager.PreviewDelete(snapshot, Target{Kind: KindSubtype, ScriptID: "new_liberalism"})
	if err != nil {
		t.Fatal(err)
	}
	changed := cloneSnapshot(snapshot)
	for i := range changed.Ideologies {
		if changed.Ideologies[i].ScriptID == "new_liberalism" {
			changed.Ideologies[i].DisplayName = "Changed after preview"
		}
	}
	if _, err := manager.ApplyDelete(changed, preview.Token); !errors.Is(err, ErrStalePreview) {
		t.Fatalf("changed snapshot apply error = %v, want ErrStalePreview", err)
	}
	if _, err := manager.ApplyDelete(snapshot, preview.Token); !errors.Is(err, ErrUnknownPreview) {
		t.Fatalf("stale attempt did not consume token: %v", err)
	}
}

func TestSnapshotHashIsDeterministicAndContentSensitive(t *testing.T) {
	first := testSnapshot()
	second := cloneSnapshot(first)
	for left, right := 0, len(second.Ideologies)-1; left < right; left, right = left+1, right-1 {
		second.Ideologies[left], second.Ideologies[right] = second.Ideologies[right], second.Ideologies[left]
	}
	second.References[0], second.References[1] = second.References[1], second.References[0]
	second.Dependencies[0], second.Dependencies[1] = second.Dependencies[1], second.Dependencies[0]
	hashA, err := SnapshotHash(first)
	if err != nil {
		t.Fatal(err)
	}
	hashB, err := SnapshotHash(second)
	if err != nil {
		t.Fatal(err)
	}
	if hashA != hashB {
		t.Fatalf("equivalent snapshots hashed differently: %s != %s", hashA, hashB)
	}
	second.Dependencies[0].Entity.Path += ".changed"
	hashC, err := SnapshotHash(second)
	if err != nil {
		t.Fatal(err)
	}
	if hashA == hashC {
		t.Fatal("path change did not invalidate snapshot hash")
	}
}

func TestCustomRootFindingsSeparateFatalAndIntentionalReview(t *testing.T) {
	snapshot := testSnapshot()

	snapshot.Dependencies = []RootDependency{
		snapshot.Dependencies[1], snapshot.Dependencies[0], snapshot.Dependencies[1],
	}
	wantDependencies := append([]RootDependency(nil), snapshot.Dependencies...)
	findings, err := AnalyzeCustomRootImpacts(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("finding count = %d, want 2: %#v", len(findings), findings)
	}
	if findings[0].Severity != SeverityFatalBlocker || findings[0].Entity.Kind != "military_name_group" {
		t.Fatalf("fatal compatibility finding was not first: %#v", findings[0])
	}
	if findings[1].Severity != SeverityIntentionalReview || findings[1].ReferenceKey != "has_government" {
		t.Fatalf("intentional review finding wrong: %#v", findings[1])
	}
	for _, finding := range findings {
		if finding.Entity.ID == "" || finding.Entity.Path == "" || finding.AutomaticRewrite || !finding.RequiresUserReview {
			t.Fatalf("finding lost object/path or requested automatic repair: %#v", finding)
		}
	}

	if !reflect.DeepEqual(snapshot.Dependencies, wantDependencies) {
		t.Fatal("impact analysis mutated dependencies")
	}
}

func TestSnapshotRejectsDuplicateScriptIDs(t *testing.T) {
	snapshot := testSnapshot()
	snapshot.Ideologies = append(snapshot.Ideologies, Ideology{
		Kind: KindSubtype, ScriptID: "democratic", RootID: "democratic",
		DisplayName: "Duplicate", Source: SourceProjectManaged,
	})
	if err := ValidateSnapshot(snapshot); !errors.Is(err, ErrDuplicateScriptID) {
		t.Fatalf("duplicate script id error = %v, want ErrDuplicateScriptID", err)
	}
}
