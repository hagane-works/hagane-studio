package politicsrefs

import (
	"sort"
	"strings"
)

type FindingSeverity string

const (
	SeverityFatalBlocker      FindingSeverity = "fatal_blocker"
	SeverityIntentionalReview FindingSeverity = "intentional_review"
)

type Finding struct {
	Code               string          `json:"code"`
	Severity           FindingSeverity `json:"severity"`
	CustomRootID       string          `json:"custom_root_id"`
	ExpectedRootID     string          `json:"expected_root_id"`
	ReferenceKey       string          `json:"reference_key"`
	Entity             EntityRef       `json:"entity"`
	AutomaticRewrite   bool            `json:"automatic_rewrite"`
	RequiresUserReview bool            `json:"requires_user_review"`
}

func normalizeDependency(dependency RootDependency) RootDependency {
	dependency.CustomRootID = strings.TrimSpace(dependency.CustomRootID)
	dependency.ExpectedRoot = strings.TrimSpace(dependency.ExpectedRoot)
	dependency.ReferenceKey = strings.TrimSpace(dependency.ReferenceKey)
	dependency.Entity.Kind = strings.TrimSpace(dependency.Entity.Kind)
	dependency.Entity.ID = strings.TrimSpace(dependency.Entity.ID)
	dependency.Entity.Path = strings.TrimSpace(dependency.Entity.Path)
	return dependency
}

func dependencyLess(left, right RootDependency) bool {
	if left.Disposition != right.Disposition {
		return left.Disposition < right.Disposition
	}
	if left.CustomRootID != right.CustomRootID {
		return left.CustomRootID < right.CustomRootID
	}
	if left.ExpectedRoot != right.ExpectedRoot {
		return left.ExpectedRoot < right.ExpectedRoot
	}
	if left.ReferenceKey != right.ReferenceKey {
		return left.ReferenceKey < right.ReferenceKey
	}
	return entityLess(left.Entity, right.Entity)
}

func findingLess(left, right Finding) bool {
	if left.Severity != right.Severity {

		return left.Severity == SeverityFatalBlocker
	}
	if entityLess(left.Entity, right.Entity) {
		return true
	}
	if entityLess(right.Entity, left.Entity) {
		return false
	}
	if left.ReferenceKey != right.ReferenceKey {
		return left.ReferenceKey < right.ReferenceKey
	}
	if left.CustomRootID != right.CustomRootID {
		return left.CustomRootID < right.CustomRootID
	}
	return left.ExpectedRootID < right.ExpectedRootID
}

func AnalyzeCustomRootImpacts(snapshot Snapshot) ([]Finding, error) {
	if err := ValidateSnapshot(snapshot); err != nil {
		return nil, err
	}
	managedRoots := map[string]bool{}
	for _, raw := range snapshot.Ideologies {
		ideology := normalizeIdeology(raw)
		if ideology.Kind == KindRoot && ideology.Source == SourceProjectManaged {
			managedRoots[ideology.ScriptID] = true
		}
	}
	seen := map[Finding]bool{}
	findings := make([]Finding, 0, len(snapshot.Dependencies))
	for _, raw := range snapshot.Dependencies {
		dependency := normalizeDependency(raw)
		if !managedRoots[dependency.CustomRootID] || dependency.CustomRootID == dependency.ExpectedRoot {
			continue
		}
		finding := Finding{
			CustomRootID:       dependency.CustomRootID,
			ExpectedRootID:     dependency.ExpectedRoot,
			ReferenceKey:       dependency.ReferenceKey,
			Entity:             dependency.Entity,
			AutomaticRewrite:   false,
			RequiresUserReview: true,
		}
		if dependency.Disposition == DependencyRequired {
			finding.Code = "politics_custom_root_compatibility_blocker"
			finding.Severity = SeverityFatalBlocker
		} else {
			finding.Code = "politics_custom_root_intentional_review"
			finding.Severity = SeverityIntentionalReview
		}
		if seen[finding] {
			continue
		}
		seen[finding] = true
		findings = append(findings, finding)
	}
	sort.Slice(findings, func(i, j int) bool { return findingLess(findings[i], findings[j]) })
	return findings, nil
}
