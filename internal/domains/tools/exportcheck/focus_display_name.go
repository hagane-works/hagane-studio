package exportcheck

import (
	"strings"
)

type DisplayNameFallbackProvider struct {
	Locale       string `json:"locale"`
	Key          string `json:"key"`
	ProviderKind string `json:"provider_kind"`
	Path         string `json:"path"`
	ContentHash  string `json:"content_hash"`
}

func (p DisplayNameFallbackProvider) Proven() bool {
	return (p.ProviderKind == "vanilla" || p.ProviderKind == "base_mod") &&
		p.Locale != "" && p.Key != "" && p.Path != "" && p.ContentHash != ""
}

func (p DisplayNameFallbackProvider) toDict() map[string]any {
	return map[string]any{
		"locale":        p.Locale,
		"key":           p.Key,
		"provider_kind": p.ProviderKind,
		"path":          p.Path,
		"content_hash":  p.ContentHash,
	}
}

type DisplayNameAnalysis struct {
	Findings        []map[string]any
	AuthoredLocales []string
	FallbackLocales []string
}

func displayNameMeaningful(value any, focusID string) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	text := strings.TrimSpace(s)
	if text == "" {
		return false
	}
	return text != focusID && text != "New Focus"
}

func enabledDisplayLocales(locales []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, locale := range locales {
		code := strings.ToLower(strings.TrimSpace(locale))
		if code == "" || seen[code] {
			continue
		}
		out = append(out, code)
		seen[code] = true
	}
	if len(out) == 0 {
		return []string{"en"}
	}
	return out
}

type DisplayNameSeverityPolicy struct {
	Map   map[string]string
	Fixed string
}

func (p DisplayNameSeverityPolicy) severityFor(locale string) string {
	value := p.Fixed
	if p.Map != nil {
		var ok bool
		if value, ok = p.Map[locale]; !ok {
			value = "warning"
		}
	}
	switch value {
	case "blocker", "warning", "info":
		return value
	}
	return "warning"
}

var displayNameSeverityRank = map[string]int{"info": 0, "warning": 1, "blocker": 2}

func displayNameSource(focusID string, sourceName *string, locale string) map[string]any {
	fieldPath := []any{"focus", focusID, "display_name_locs"}
	if locale != "" {
		fieldPath = append(fieldPath, locale)
	}
	var file any
	if sourceName != nil {
		file = *sourceName
	}
	return map[string]any{
		"file":       file,
		"entity":     map[string]any{"type": "focus", "id": focusID},
		"field_path": fieldPath,
		"span":       nil,
	}
}

type FocusDisplayNameInput struct {
	FocusID               string
	Source                string
	DisplayNameLocs       map[string]any
	NameEN                string
	NameZH                string
	EnabledContentLocales []string
	IsNewID               bool
	ReusesBaseID          bool
	MaterialNameChanged   bool
	FallbackProviders     []DisplayNameFallbackProvider
	MissingLocaleSeverity DisplayNameSeverityPolicy
	SourceName            *string
}

func AnalyzeFocusDisplayName(in FocusDisplayNameInput) DisplayNameAnalysis {
	locales := enabledDisplayLocales(in.EnabledContentLocales)

	names := map[string]any{}
	for k, v := range in.DisplayNameLocs {
		names[k] = v
	}

	if displayNameMeaningful(in.NameEN, in.FocusID) && !displayNameMeaningful(names["en"], in.FocusID) {
		names["en"] = in.NameEN
	}
	if displayNameMeaningful(in.NameZH, in.FocusID) && !displayNameMeaningful(names["zh"], in.FocusID) {
		names["zh"] = in.NameZH
	}

	authored := []string{}
	authoredSet := map[string]bool{}
	for _, locale := range locales {
		if displayNameMeaningful(names[locale], in.FocusID) {
			authored = append(authored, locale)
			authoredSet[locale] = true
		}
	}

	providersByLocale := map[string][]DisplayNameFallbackProvider{}
	for _, provider := range in.FallbackProviders {
		if provider.Key != in.FocusID || !provider.Proven() {
			continue
		}
		code := strings.ToLower(provider.Locale)
		providersByLocale[code] = append(providersByLocale[code], provider)
	}

	fallbackAllowed := (in.Source == "imported" || in.ReusesBaseID) &&
		!in.IsNewID && !in.MaterialNameChanged

	fallbackLocales := []string{}
	fallbackSet := map[string]bool{}
	for _, locale := range locales {
		if !authoredSet[locale] && fallbackAllowed && len(providersByLocale[locale]) > 0 {
			fallbackLocales = append(fallbackLocales, locale)
			fallbackSet[locale] = true
		}
	}

	missing := []string{}
	for _, locale := range locales {
		if !authoredSet[locale] && !fallbackSet[locale] {
			missing = append(missing, locale)
		}
	}
	if len(missing) == 0 {
		return DisplayNameAnalysis{AuthoredLocales: authored, FallbackLocales: fallbackLocales}
	}

	findings := []map[string]any{}
	authorOwned := in.IsNewID || !fallbackAllowed

	if authorOwned && len(authored) == 0 {

		severity := "blocker"
		if !in.IsNewID {
			severity = ""
			best := -1
			for _, locale := range missing {
				s := in.MissingLocaleSeverity.severityFor(locale)
				if rank := displayNameSeverityRank[s]; rank > best {
					best, severity = rank, s
				}
			}
		}
		findings = append(findings, map[string]any{
			"code":     "focus.display_name.missing_created",
			"severity": severity,
			"source":   displayNameSource(in.FocusID, in.SourceName, ""),
			"evidence": map[string]any{
				"source":                  in.Source,
				"focus_id":                in.FocusID,
				"enabled_content_locales": locales,
				"missing_locales":         missing,
				"is_new_id":               in.IsNewID,
				"reuses_base_id":          in.ReusesBaseID,
				"material_name_changed":   in.MaterialNameChanged,
			},
			"repair_route": "focus.display_name.author_enabled_locale",
		})
		return DisplayNameAnalysis{
			Findings: findings, AuthoredLocales: authored, FallbackLocales: fallbackLocales,
		}
	}

	for _, locale := range missing {
		providers := []map[string]any{}
		for _, p := range providersByLocale[locale] {
			providers = append(providers, p.toDict())
		}
		code := "focus.display_name.missing_enabled_locale"
		route := "focus.display_name.author_enabled_locale"
		if fallbackAllowed {

			code = "focus.display_name.unproven_fallback"
			route = "focus.display_name.register_or_author_fallback"
		}
		findings = append(findings, map[string]any{
			"code":     code,
			"severity": in.MissingLocaleSeverity.severityFor(locale),
			"source":   displayNameSource(in.FocusID, in.SourceName, locale),
			"evidence": map[string]any{
				"source":                in.Source,
				"focus_id":              in.FocusID,
				"locale":                locale,
				"fallback_allowed":      fallbackAllowed,
				"fallback_candidates":   providers,
				"material_name_changed": in.MaterialNameChanged,
			},
			"repair_route": route,
		})
	}

	return DisplayNameAnalysis{
		Findings: findings, AuthoredLocales: authored, FallbackLocales: fallbackLocales,
	}
}
