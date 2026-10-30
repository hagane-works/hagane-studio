package focus

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var focusCountryTagRE = regexp.MustCompile(`^[A-Z0-9]{2,5}$`)

var focusScopeKinds = map[string]bool{
	"own": true, "explicit": true, "global": true, "global_except": true,
}

type FocusPolicyFinding struct {
	Code        string         `json:"code"`
	Severity    string         `json:"severity"`
	Source      string         `json:"source"`
	Message     string         `json:"message,omitempty"`
	Evidence    map[string]any `json:"evidence"`
	RepairRoute string         `json:"repair_route"`
}

var FocusFindingCodes = []string{
	"focus_shared_pool_scope_ignored",
	"focus_scope_unknown_kind",
	"focus_scope_own_missing_country",
	"focus_scope_explicit_missing_countries",
	"focus_scope_global_except_missing_countries",
	"focus_scope_invalid_country_tag",
	"focus_scope_own_foreign_country",
}

func evidenceTag(evidence map[string]any, key string) string {
	if text, ok := evidence[key].(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func evidenceTags(evidence map[string]any, key string) []string {
	var out []string
	switch typed := evidence[key].(type) {
	case []string:
		out = append(out, typed...)
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
	}
	return out
}

func focusFindingMessage(f FocusPolicyFinding) string {
	switch f.Code {
	case "focus_shared_pool_scope_ignored":
		return "A shared-pool focus file has no country of its own, so the scope_kind / " +
			"country_tag / shared_country_tags you sent were ignored. Nothing is lost — " +
			"countries reach these focuses through their own tree. Drop those fields to " +
			"silence this warning."
	case "focus_scope_unknown_kind":
		kind := evidenceTag(f.Evidence, "scope_kind")
		if kind == "" {
			kind = "(empty)"
		}
		return "scope_kind \"" + kind + "\" is not a scope Hagane Works knows. Use one of: " +
			"own (the tree belongs to a single country), explicit (only the countries you " +
			"list), global (every country), global_except (every country except the ones " +
			"you list)."
	case "focus_scope_own_missing_country":
		return "A scope_kind=own tree belongs to exactly one country, and none was given. " +
			"Set country_tag to the country whose tree this is."
	case "focus_scope_explicit_missing_countries":
		return "A scope_kind=explicit tree is offered only to the countries you list, and " +
			"the list is empty. Put at least one country tag in shared_country_tags."
	case "focus_scope_global_except_missing_countries":
		return "A scope_kind=global_except tree is offered to every country EXCEPT the ones " +
			"you list, and the list is empty. Either list the countries to exclude in " +
			"shared_country_tags, or use scope_kind=global."
	case "focus_scope_invalid_country_tag":
		tags := evidenceTags(f.Evidence, "country_tags")
		return "Not a usable HOI4 country tag: " + strings.Join(tags, ", ") +
			". A tag is 2-5 characters, uppercase letters and digits only (GER, SOV, D01)."
	case "focus_scope_own_foreign_country":
		tag := evidenceTag(f.Evidence, "country_tag")
		known := len(evidenceTags(f.Evidence, "project_country_tags"))
		return "This project has no country " + tag + ", so a focus tree cannot belong to " +
			"it. Create " + tag + " in the project first, then create the tree — or point " +
			"country_tag at one of the " + strconv.Itoa(known) + " countries the project " +
			"already has (they are listed in evidence.project_country_tags)."
	}
	return ""
}

func withFocusFindingMessages(findings []FocusPolicyFinding) []FocusPolicyFinding {
	for i := range findings {
		findings[i].Message = focusFindingMessage(findings[i])
	}
	return findings
}

type FocusScopeCandidate struct {
	TreeKind          string
	TreeID            string
	ScopeKind         string
	CountryTag        string
	SharedCountryTags []string
}

type FocusScopeVerdict struct {
	Kind        string
	CountryTags []string
}

type FocusScopeMaterialization struct {
	TreeKind          string
	ScopeKind         string
	CountryTag        string
	SharedCountryTags []string
	Findings          []FocusPolicyFinding
}

func (m FocusScopeMaterialization) Blocked() bool {
	for _, f := range m.Findings {
		if f.Severity == "blocker" {
			return true
		}
	}
	return false
}

func normaliseFocusTag(v string) string {
	return strings.ToUpper(strings.TrimSpace(v))
}

func normaliseFocusTags(vs []string) []string {
	out := make([]string, 0, len(vs))
	seen := map[string]bool{}
	for _, v := range vs {
		tag := normaliseFocusTag(v)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	return out
}

func nullableTag(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func tagsEvidence(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

func MaterializeFocusBundle(
	tree FocusScopeCandidate,
	projectCountryTags []string,
	verdict *FocusScopeVerdict,
	source string,
) FocusScopeMaterialization {
	if source == "" {
		source = "focus_scope"
	}
	treeKind := strings.TrimSpace(tree.TreeKind)
	if treeKind == "" {
		treeKind = "primary"
	}
	treeID := tree.TreeID
	var findings []FocusPolicyFinding

	if treeKind == "shared_pool" {
		suppliedScopeKind := strings.TrimSpace(tree.ScopeKind)
		suppliedCountry := normaliseFocusTag(tree.CountryTag)
		suppliedShared := normaliseFocusTags(tree.SharedCountryTags)
		if suppliedScopeKind != "" || suppliedCountry != "" || len(suppliedShared) > 0 {
			findings = append(findings, FocusPolicyFinding{
				Code:     "focus_shared_pool_scope_ignored",
				Severity: "warning",
				Source:   source,
				Evidence: map[string]any{
					"tree_id":             treeID,
					"scope_kind":          suppliedScopeKind,
					"country_tag":         nullableTag(suppliedCountry),
					"shared_country_tags": tagsEvidence(suppliedShared),
				},
				RepairRoute: "focus_shared_pool_review",
			})
		}
		return FocusScopeMaterialization{
			TreeKind:          "shared_pool",
			ScopeKind:         "shared_pool",
			SharedCountryTags: []string{},
			Findings:          withFocusFindingMessages(findings),
		}
	}

	hasVerdict := verdict != nil
	scopeKind := strings.TrimSpace(tree.ScopeKind)
	var verdictTags []string
	if hasVerdict {
		scopeKind = strings.TrimSpace(verdict.Kind)
		verdictTags = normaliseFocusTags(verdict.CountryTags)
	}
	persistedCountry := normaliseFocusTag(tree.CountryTag)
	persistedShared := normaliseFocusTags(tree.SharedCountryTags)
	allowed := map[string]bool{}
	for _, tag := range normaliseFocusTags(projectCountryTags) {
		allowed[tag] = true
	}

	if !focusScopeKinds[scopeKind] {
		findings = append(findings, FocusPolicyFinding{
			Code:     "focus_scope_unknown_kind",
			Severity: "blocker",
			Source:   source,
			Evidence: map[string]any{
				"tree_id":    treeID,
				"scope_kind": nullableTag(scopeKind),
			},
			RepairRoute: "focus_scope_repair",
		})
		return FocusScopeMaterialization{
			TreeKind:          treeKind,
			ScopeKind:         scopeKind,
			SharedCountryTags: []string{},
			Findings:          withFocusFindingMessages(findings),
		}
	}

	countryTag := ""
	sharedCountryTags := []string{}
	switch scopeKind {
	case "own":
		var ownTags []string
		if hasVerdict {
			ownTags = verdictTags
		} else if persistedCountry != "" {
			ownTags = []string{persistedCountry}
		}
		if len(ownTags) != 1 {
			findings = append(findings, FocusPolicyFinding{
				Code:     "focus_scope_own_missing_country",
				Severity: "blocker",
				Source:   source,
				Evidence: map[string]any{
					"tree_id":      treeID,
					"country_tags": tagsEvidence(ownTags),
				},
				RepairRoute: "focus_scope_choose_country",
			})
		} else {
			countryTag = ownTags[0]
		}
	case "explicit", "global_except":
		if hasVerdict {
			sharedCountryTags = verdictTags
		} else {
			sharedCountryTags = persistedShared
		}
		if len(sharedCountryTags) == 0 {
			code := "focus_scope_explicit_missing_countries"
			if scopeKind == "global_except" {
				code = "focus_scope_global_except_missing_countries"
			}
			findings = append(findings, FocusPolicyFinding{
				Code:        code,
				Severity:    "blocker",
				Source:      source,
				Evidence:    map[string]any{"tree_id": treeID},
				RepairRoute: "focus_scope_choose_country",
			})
		}
	}

	var tagsToValidate []string
	if countryTag != "" {
		tagsToValidate = append(tagsToValidate, countryTag)
	}
	tagsToValidate = append(tagsToValidate, sharedCountryTags...)
	var invalidTags []string
	for _, tag := range tagsToValidate {
		if !focusCountryTagRE.MatchString(tag) {
			invalidTags = append(invalidTags, tag)
		}
	}
	if len(invalidTags) > 0 {
		findings = append(findings, FocusPolicyFinding{
			Code:     "focus_scope_invalid_country_tag",
			Severity: "blocker",
			Source:   source,
			Evidence: map[string]any{
				"tree_id":      treeID,
				"country_tags": invalidTags,
			},
			RepairRoute: "focus_scope_choose_country",
		})
	}
	if scopeKind == "own" && countryTag != "" && len(allowed) > 0 && !allowed[countryTag] {
		sortedAllowed := make([]string, 0, len(allowed))
		for tag := range allowed {
			sortedAllowed = append(sortedAllowed, tag)
		}
		sort.Strings(sortedAllowed)
		findings = append(findings, FocusPolicyFinding{
			Code:     "focus_scope_own_foreign_country",
			Severity: "blocker",
			Source:   source,
			Evidence: map[string]any{
				"tree_id":              treeID,
				"country_tag":          countryTag,
				"project_country_tags": sortedAllowed,
			},
			RepairRoute: "focus_scope_choose_country",
		})
	}

	return FocusScopeMaterialization{
		TreeKind:          treeKind,
		ScopeKind:         scopeKind,
		CountryTag:        countryTag,
		SharedCountryTags: sharedCountryTags,
		Findings:          withFocusFindingMessages(findings),
	}
}
