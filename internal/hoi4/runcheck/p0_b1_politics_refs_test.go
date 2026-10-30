package runcheck

import (
	"testing"

	"haganeworks/internal/hoi4/politicsrefs"
)

func TestCollectPoliticsSeparatesBlockerAndReview(t *testing.T) {
	snapshot := politicsrefs.Snapshot{
		ProjectID: "project-8",
		Ideologies: []politicsrefs.Ideology{
			{Kind: politicsrefs.KindRoot, ScriptID: "democratic", Source: politicsrefs.SourceVanilla},
			{Kind: politicsrefs.KindRoot, ScriptID: "civic_coalition", Source: politicsrefs.SourceProjectManaged},
			{Kind: politicsrefs.KindSubtype, ScriptID: "civic_reformism", RootID: "civic_coalition", Source: politicsrefs.SourceProjectManaged},
		},
		Dependencies: []politicsrefs.RootDependency{
			{CustomRootID: "civic_coalition", ExpectedRoot: "democratic", ReferenceKey: "has_government",
				Disposition: politicsrefs.DependencyIntentional, Entity: politicsrefs.EntityRef{Kind: "focus", ID: "11", Path: "focuses.FRA_gate.available"}},
			{CustomRootID: "civic_coalition", ExpectedRoot: "democratic", ReferenceKey: "military_name_group:FRA_INF_01",
				Disposition: politicsrefs.DependencyRequired, Entity: politicsrefs.EntityRef{Kind: "division_template", ID: "12", Path: "history/units/FRA_1936.txt"}},
		},
	}
	findings, err := CollectPolitics(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings=%#v", findings)
	}
	if findings[0].Severity != "blocker" || findings[0].Code != "politics_custom_root_compatibility_blocker" {
		t.Fatalf("required compatibility was not first blocker: %#v", findings[0])
	}
	if findings[1].Severity != "warning" || findings[1].Code != "politics_custom_root_intentional_review" {
		t.Fatalf("intentional gate was not review warning: %#v", findings[1])
	}
	if findings[0].Detail["automatic_rewrite"] != false {
		t.Fatal("run check advertised automatic rewrite")
	}
}

func TestCollectPoliticsBlocksCustomRootWithoutRandomEligibleSubtype(t *testing.T) {
	snapshot := politicsrefs.Snapshot{ProjectID: "project-8", Ideologies: []politicsrefs.Ideology{
		{Kind: politicsrefs.KindRoot, ScriptID: "technocracy", Source: politicsrefs.SourceProjectManaged},
		{Kind: politicsrefs.KindSubtype, ScriptID: "civic_technocracy", RootID: "technocracy", Source: politicsrefs.SourceProjectManaged, RandomDisabled: true},
	}}
	findings, err := CollectPolitics(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Code != "politics_custom_root_no_random_subtype" || findings[0].Severity != "blocker" {
		t.Fatalf("findings=%#v", findings)
	}
	if findings[0].Detail["custom_root_id"] != "technocracy" || findings[0].RepairRoute != "politics.ideologies" {
		t.Fatalf("finding=%#v", findings[0])
	}
}

func TestCollectPoliticsAcceptsCustomRootWithRandomEligibleSubtype(t *testing.T) {
	snapshot := politicsrefs.Snapshot{ProjectID: "project-8", Ideologies: []politicsrefs.Ideology{
		{Kind: politicsrefs.KindRoot, ScriptID: "technocracy", Source: politicsrefs.SourceProjectManaged},
		{Kind: politicsrefs.KindSubtype, ScriptID: "default_technocracy", RootID: "technocracy", Source: politicsrefs.SourceProjectManaged},
		{Kind: politicsrefs.KindSubtype, ScriptID: "civic_technocracy", RootID: "technocracy", Source: politicsrefs.SourceProjectManaged, RandomDisabled: true},
	}}
	findings, err := CollectPolitics(snapshot)
	if err != nil || len(findings) != 0 {
		t.Fatalf("findings=%#v err=%v", findings, err)
	}
}
