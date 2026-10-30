package exportcheck

func (a DisplayNameAnalysis) AuthorVisible() bool { return len(a.Findings) > 0 }

func (a DisplayNameAnalysis) ToDict() map[string]any {
	findings := a.Findings
	if findings == nil {
		findings = []map[string]any{}
	}
	return map[string]any{
		"findings":         findings,
		"authored_locales": nonNilStrings(a.AuthoredLocales),
		"fallback_locales": nonNilStrings(a.FallbackLocales),
		"author_visible":   a.AuthorVisible(),
	}
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
