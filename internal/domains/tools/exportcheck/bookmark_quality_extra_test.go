package exportcheck

func (r BookmarkQualityResult) Blockers() []bookmarkGateFinding {
	out := []bookmarkGateFinding{}
	for _, f := range r.Findings {
		if f.Severity == "blocker" {
			out = append(out, f)
		}
	}
	return out
}
