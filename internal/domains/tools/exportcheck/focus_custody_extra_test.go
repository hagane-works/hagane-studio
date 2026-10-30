package exportcheck

func (m FocusScopeMaterialization) Blocked() bool {
	for _, f := range m.Findings {
		if f.Severity == "blocker" {
			return true
		}
	}
	return false
}
