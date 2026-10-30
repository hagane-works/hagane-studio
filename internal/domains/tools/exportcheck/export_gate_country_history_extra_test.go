package exportcheck

func (r CountryHistoryRegistry) Blockers() []map[string]any {
	out := []map[string]any{}
	for _, f := range r.Findings {
		if f["severity"] == "blocker" {
			out = append(out, f)
		}
	}
	return out
}
