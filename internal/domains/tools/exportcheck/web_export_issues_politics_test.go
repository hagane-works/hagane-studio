package exportcheck

import (
	"os"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/politicsrefs"
)

func TestExportAssemblyConsumesCanonicalPoliticsSnapshot(t *testing.T) {
	source, err := os.ReadFile("export_issues_load.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"s.PoliticsReferences(ctx, pid)", "politicsExportIssues(b.project, politicsSnapshot)"} {
		if !strings.Contains(string(source), required) {
			t.Fatalf("export assembly lost canonical politics seam %q", required)
		}
	}
}

func inheritedPoliticsSnapshot() politicsrefs.Snapshot {
	return politicsrefs.Snapshot{
		ProjectID: "project-8",
		Ideologies: []politicsrefs.Ideology{
			{Kind: politicsrefs.KindRoot, ScriptID: "democratic", Source: politicsrefs.SourceVanilla},
			{Kind: politicsrefs.KindRoot, ScriptID: "civic_coalition", Source: politicsrefs.SourceProjectManaged},
			{Kind: politicsrefs.KindSubtype, ScriptID: "civic_centrism", RootID: "civic_coalition", Source: politicsrefs.SourceProjectManaged},
			{Kind: politicsrefs.KindSubtype, ScriptID: "civic_reformism", RootID: "democratic", Source: politicsrefs.SourceProjectManaged},
		},
		Dependencies: []politicsrefs.RootDependency{
			{CustomRootID: "civic_coalition", ExpectedRoot: "democratic", ReferenceKey: "has_government", Disposition: politicsrefs.DependencyIntentional, Entity: politicsrefs.EntityRef{Kind: "focus", ID: "11", Path: "focuses.FRA_gate.available"}},
			{CustomRootID: "civic_coalition", ExpectedRoot: "democratic", ReferenceKey: "military_name_group:FRA_INF_01", Disposition: politicsrefs.DependencyRequired, Entity: politicsrefs.EntityRef{Kind: "division_template", ID: "12", Path: "history/units/FRA_1936.txt"}},
		},
	}
}

func TestPoliticsLifecycleFindingsShareCanonicalCompatibilityClassification(t *testing.T) {
	findings, err := CollectPoliticsLifecycleFindings(Project{IdeologySourceManifest: map[string]any{
		"state": "ready", "source_key": "vanilla",
	}}, inheritedPoliticsSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 || findings[0].Code != "politics_custom_root_compatibility_blocker" || findings[0].Severity != "blocker" ||
		findings[1].Code != "politics_custom_root_intentional_review" || findings[1].Severity != "warning" {
		t.Fatalf("findings=%#v", findings)
	}
}

func TestPoliticsLifecycleFindingsBlockMissingAndWarnStaleInheritedSource(t *testing.T) {
	for _, tc := range []struct {
		name, code, severity string
		automatic            bool
		manifest             map[string]any
	}{
		{name: "missing", code: politicsSourceMissingCode, severity: "blocker", manifest: map[string]any{}},
		{name: "stale", code: politicsSourceStaleCode, severity: "warning", automatic: true, manifest: map[string]any{"state": "stale", "source_key": "vanilla"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings, err := CollectPoliticsLifecycleFindings(Project{IdeologySourceManifest: tc.manifest}, inheritedPoliticsSnapshot())
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, finding := range findings {
				if finding.Code == tc.code {
					found = finding.Severity == tc.severity && finding.Detail["automatic_rewrite"] == tc.automatic
				}
			}
			if !found {
				t.Fatalf("missing %s in %#v", tc.code, findings)
			}
		})
	}
}

func TestPoliticsLifecycleDoesNotDemandSourceForProjectOwnedRoot(t *testing.T) {
	snapshot := politicsrefs.Snapshot{ProjectID: "project-8", Ideologies: []politicsrefs.Ideology{
		{Kind: politicsrefs.KindRoot, ScriptID: "civic_coalition", Source: politicsrefs.SourceProjectManaged},
		{Kind: politicsrefs.KindSubtype, ScriptID: "civic_reformism", RootID: "civic_coalition", Source: politicsrefs.SourceProjectManaged},
	}}
	findings, err := CollectPoliticsLifecycleFindings(Project{}, snapshot)
	if err != nil || len(findings) != 0 {
		t.Fatalf("findings=%#v err=%v", findings, err)
	}
}

func TestPoliticsExportIssuesExplainMissingRandomEligibleSubtype(t *testing.T) {
	snapshot := politicsrefs.Snapshot{ProjectID: "project-8", Ideologies: []politicsrefs.Ideology{
		{Kind: politicsrefs.KindRoot, ScriptID: "technocracy", Source: politicsrefs.SourceProjectManaged},
		{Kind: politicsrefs.KindSubtype, ScriptID: "civic_technocracy", RootID: "technocracy", Source: politicsrefs.SourceProjectManaged, RandomDisabled: true},
	}}
	issues, err := politicsExportIssues(Project{}, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0]["code"] != "politics_custom_root_no_random_subtype" || issues[0]["severity"] != "blocker" {
		t.Fatalf("issues=%#v", issues)
	}
	if issues[0]["message"] == "" || issues[0]["fix_hint"] == "" {
		t.Fatalf("issue lacks player wording: %#v", issues[0])
	}
}

func TestPoliticsExportIssuesCarryUserReadableWordingAndSameCodes(t *testing.T) {
	issues, err := politicsExportIssues(Project{IdeologySourceManifest: map[string]any{
		"state": "stale", "source_key": "vanilla",
	}}, inheritedPoliticsSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{
		"politics_custom_root_compatibility_blocker": false,
		"politics_custom_root_intentional_review":    false,
		politicsSourceStaleCode:                      false,
	}
	for _, issue := range issues {
		code, _ := issue["code"].(string)
		if _, tracked := wanted[code]; !tracked {
			continue
		}
		message, _ := issue["message"].(string)
		hint, _ := issue["fix_hint"].(string)
		if message == "" || hint == "" {
			t.Fatalf("issue lacks user wording: %#v", issue)
		}
		wanted[code] = true
	}
	for code, found := range wanted {
		if !found {
			t.Fatalf("missing issue %s: %#v", code, issues)
		}
	}
}
