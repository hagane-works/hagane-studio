package states

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/savediagnostics"
)

type bookmarkCountryInput struct {
	Tag         string
	Ideology    string
	Labels      []string
	Interesting bool
}

type bookmarkQualityInput struct {
	BookmarkID     string
	Path           string
	Source         string
	Default        bool
	DefaultCountry string
	Countries      []bookmarkCountryInput
	Filters        []string
	TargetVersion  string
	Emitted        bool
}

type bookmarkGateFinding struct {
	Code        string         `json:"code"`
	Severity    string         `json:"severity"`
	Source      map[string]any `json:"source"`
	Evidence    map[string]any `json:"evidence"`
	RepairRoute string         `json:"repair_route"`
}

type bookmarkQualityResult struct {
	TargetVersion string
	Findings      []bookmarkGateFinding
}

func bookmarkFilterChoicesForVersion(targetVersion string) []string {
	raw := strings.TrimSpace(targetVersion)
	if raw == "1.19" || strings.HasPrefix(raw, "1.19.") {
		return []string{"new_content", "continent", "label"}
	}
	return nil
}

func bookmarkFindingSource(b bookmarkQualityInput, countryTag string) map[string]any {
	kind, identifier := "bookmark", b.BookmarkID
	if countryTag != "" {
		kind, identifier = "bookmark_country", b.BookmarkID+":"+countryTag
	}
	src := map[string]any{"kind": kind, "identifier": identifier}
	if b.Path != "" {
		src["path"] = b.Path
	}
	if b.Source != "" {
		src["provider"] = b.Source
	}
	return src
}

func bookmarkIsLegacySource(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "imported", "legacy", "preserved_raw", "base_mod", "vanilla":
		return true
	}
	return false
}

func bookmarkValidLabelToken(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f' ||
			r == '{' || r == '}' || r == '=' || r == '#' || r == '"' {
			return false
		}
	}
	return true
}

func emptyIfNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func evaluateBookmarkQuality(bookmarks []bookmarkQualityInput) bookmarkQualityResult {
	var emitted []bookmarkQualityInput
	for _, item := range bookmarks {
		if item.Emitted && strings.TrimSpace(item.BookmarkID) != "" {
			emitted = append(emitted, item)
		}
	}
	var versions []string
	seenVersions := map[string]bool{}
	for _, item := range emitted {
		if !seenVersions[item.TargetVersion] {
			seenVersions[item.TargetVersion] = true
			versions = append(versions, item.TargetVersion)
		}
	}
	targetVersion := ""
	if len(versions) == 1 {
		targetVersion = versions[0]
	}
	var allowed []string
	if targetVersion != "" {
		allowed = bookmarkFilterChoicesForVersion(targetVersion)
	}
	findings := []bookmarkGateFinding{}
	if len(emitted) > 0 && (targetVersion == "" || len(allowed) == 0) {
		findings = append(findings, bookmarkGateFinding{
			Code: "bookmark.filter_registry_missing", Severity: "blocker",
			Source:      bookmarkFindingSource(emitted[0], ""),
			Evidence:    map[string]any{"target_versions": versions},
			RepairRoute: "project.target_version.select_supported",
		})
	}
	var defaults []bookmarkQualityInput
	for _, item := range emitted {
		if item.Default {
			defaults = append(defaults, item)
		}
	}
	if len(defaults) > 1 {
		ids := make([]string, 0, len(defaults))
		paths := make([]string, 0, len(defaults))
		for _, item := range defaults {
			ids = append(ids, item.BookmarkID)
			paths = append(paths, item.Path)
		}
		findings = append(findings, bookmarkGateFinding{
			Code: "bookmark.default_multiple", Severity: "blocker",
			Source:      map[string]any{"kind": "bookmark_set", "identifier": "effective"},
			Evidence:    map[string]any{"default_count": len(defaults), "bookmark_ids": ids, "paths": paths},
			RepairRoute: "bookmarks.default.select_one",
		})
	}
	allowedSet := map[string]bool{}
	for _, v := range allowed {
		allowedSet[v] = true
	}
	for _, bookmark := range emitted {
		var filters []string
		for _, value := range bookmark.Filters {
			if t := strings.TrimSpace(value); t != "" {
				filters = append(filters, t)
			}
		}
		var countries []bookmarkCountryInput
		for _, country := range bookmark.Countries {
			if strings.TrimSpace(country.Tag) != "" {
				countries = append(countries, country)
			}
		}
		interesting := map[string]bool{}
		for _, country := range countries {
			if country.Interesting {
				interesting[strings.ToUpper(strings.TrimSpace(country.Tag))] = true
			}
		}
		defaultCountry := strings.ToUpper(strings.TrimSpace(bookmark.DefaultCountry))
		if defaultCountry != "" && !interesting[defaultCountry] {
			tags := make([]string, 0, len(interesting))
			for tag := range interesting {
				tags = append(tags, tag)
			}
			sort.Strings(tags)
			findings = append(findings, bookmarkGateFinding{
				Code: "bookmark.default_country_not_interesting", Severity: "blocker",
				Source:      bookmarkFindingSource(bookmark, ""),
				Evidence:    map[string]any{"default_country": defaultCountry, "interesting_countries": tags},
				RepairRoute: "bookmark.default_country.choose_interesting",
			})
		}
		var allLabels []string
		seenLabels := map[string]bool{}
		for _, country := range countries {
			for _, label := range country.Labels {
				if t := strings.TrimSpace(label); t != "" && !seenLabels[t] {
					seenLabels[t] = true
					allLabels = append(allLabels, t)
				}
			}
		}
		var invalidLabels []string
		for _, label := range allLabels {
			if !bookmarkValidLabelToken(label) {
				invalidLabels = append(invalidLabels, label)
			}
		}
		if len(invalidLabels) > 0 {
			legacy := bookmarkIsLegacySource(bookmark.Source)
			code, severity := "bookmark.country_label_invalid", "blocker"
			if legacy {
				code, severity = "bookmark.country_label_legacy_invalid_preserved", "warning"
			}
			findings = append(findings, bookmarkGateFinding{
				Code: code, Severity: severity, Source: bookmarkFindingSource(bookmark, ""),
				Evidence:    map[string]any{"invalid_labels": invalidLabels, "preserved": legacy},
				RepairRoute: "bookmark.countries.labels.repair",
			})
		}
		hasLabelFilter := false
		for _, f := range filters {
			if f == "label" {
				hasLabelFilter = true
				break
			}
		}
		if len(allLabels) > 0 && !hasLabelFilter {
			findings = append(findings, bookmarkGateFinding{
				Code: "bookmark.country_labels_filter_missing", Severity: "warning",
				Source:      bookmarkFindingSource(bookmark, ""),
				Evidence:    map[string]any{"country_labels": allLabels, "filters": emptyIfNil(filters)},
				RepairRoute: "bookmark.filters.enable_label",
			})
		}
		var unknownFilters []string
		for _, value := range filters {
			if !allowedSet[value] {
				unknownFilters = append(unknownFilters, value)
			}
		}
		if len(unknownFilters) > 0 {
			legacy := bookmarkIsLegacySource(bookmark.Source)
			code, severity := "bookmark.filter_invalid_for_target", "blocker"
			if legacy {
				code, severity = "bookmark.filter_legacy_unknown_preserved", "warning"
			}
			findings = append(findings, bookmarkGateFinding{
				Code: code, Severity: severity, Source: bookmarkFindingSource(bookmark, ""),
				Evidence: map[string]any{
					"tokens": unknownFilters, "allowed_new_choices": emptyIfNil(allowed),
					"target_version": bookmark.TargetVersion, "preserved": legacy,
				},
				RepairRoute: "bookmark.filters.choose_versioned",
			})
			labelsCasefold := map[string]string{}
			for _, label := range allLabels {
				if _, seen := labelsCasefold[strings.ToLower(label)]; !seen {
					labelsCasefold[strings.ToLower(label)] = label
				}
			}
			var matching, matchingLabels []string
			for _, value := range unknownFilters {
				if display, ok := labelsCasefold[strings.ToLower(value)]; ok {
					matching = append(matching, value)
					matchingLabels = append(matchingLabels, display)
				}
			}
			if len(matching) > 0 {
				findings = append(findings, bookmarkGateFinding{
					Code: "bookmark.filter_country_label_semantics", Severity: "warning",
					Source: bookmarkFindingSource(bookmark, ""),
					Evidence: map[string]any{
						"filter_tokens": matching, "country_labels": matchingLabels, "recommended_filter": "label",
					},
					RepairRoute: "bookmark.filters.replace_country_values_with_label",
				})
			}
		}
		for _, country := range countries {
			if strings.TrimSpace(country.Ideology) != "" {
				continue
			}
			tag := strings.ToUpper(strings.TrimSpace(country.Tag))
			findings = append(findings, bookmarkGateFinding{
				Code: "bookmark.selectable_ideology_missing", Severity: "warning",
				Source: bookmarkFindingSource(bookmark, tag),
				Evidence: map[string]any{
					"country_tag": tag, "interesting": country.Interesting,
					"player_impact": []string{"country_leader", "ideology_icon", "country_name", "flag"},
					"ctd_claim":     false,
				},
				RepairRoute: "bookmark.country.ideology.set",
			})
		}
	}
	return bookmarkQualityResult{TargetVersion: targetVersion, Findings: findings}
}

func bookmarkQualityCountries(entries []map[string]any) []bookmarkCountryInput {
	var out []bookmarkCountryInput
	for _, item := range entries {
		if item == nil {
			continue
		}
		tag := strings.ToUpper(strings.TrimSpace(str(item["tag"])))
		if tag == "" {
			continue
		}
		var labels []string
		switch raw := item["label"].(type) {
		case string:
			if t := strings.TrimSpace(raw); t != "" {
				labels = []string{t}
			}
		case []any:
			for _, value := range raw {
				if t := strings.TrimSpace(str(value)); t != "" {
					labels = append(labels, t)
				}
			}
		}
		out = append(out, bookmarkCountryInput{
			Tag: tag, Ideology: strings.TrimSpace(str(item["ideology"])), Labels: labels, Interesting: true,
		})
	}
	return out
}

func diagnosticStrings(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func bookmarkSaveDiagnostics(projectID int64, rows []bookmark) []savediagnostics.Diagnostic {
	inputs := make([]bookmarkQualityInput, 0, len(rows))
	for _, b := range rows {
		inputs = append(inputs, bookmarkQualityInput{
			BookmarkID: b.BookmarkID, Path: "common/bookmarks/" + b.BookmarkID + ".txt",
			Source: b.Source, Default: b.DefaultFlag, DefaultCountry: b.DefaultCountry,
			Countries: bookmarkQualityCountries(b.SelectableCountries), Filters: b.Filters,
			TargetVersion: "1.19.2", Emitted: strings.TrimSpace(b.BookmarkID) != "",
		})
	}
	out := []savediagnostics.Diagnostic{}
	for _, f := range evaluateBookmarkQuality(inputs).Findings {
		kind, _ := f.Source["kind"].(string)
		subject, _ := f.Source["identifier"].(string)
		affected := []string{subject}
		if kind == "bookmark_set" {
			subject = strconv.FormatInt(projectID, 10)
			affected = diagnosticStrings(f.Evidence["bookmark_ids"])
		}
		out = append(out, savediagnostics.Diagnostic{
			Key:  fmt.Sprintf("%s:%d:%s:%s", f.Code, projectID, kind, subject),
			Code: f.Code, Severity: "warning",
			EntityRef: savediagnostics.EntityRef{Type: kind, ID: subject},
			Evidence:  f.Evidence, RepairRoute: f.RepairRoute, Affected: affected, Source: f.Source,
		})
	}
	return savediagnostics.Current(out)
}

func bookmarkRowDiagnostics(all []savediagnostics.Diagnostic, b bookmark) *[]savediagnostics.Diagnostic {
	out := []savediagnostics.Diagnostic{}
	for _, d := range all {
		if d.EntityRef.Type == "bookmark_set" || d.EntityRef.ID == b.BookmarkID ||
			(d.EntityRef.Type == "bookmark_country" && strings.HasPrefix(d.EntityRef.ID, b.BookmarkID+":")) {
			out = append(out, d)
		}
	}
	return savediagnostics.Evaluated(out)
}
