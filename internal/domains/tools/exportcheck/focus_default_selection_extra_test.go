package exportcheck

func (s FocusDefaultSelection) Blocked() bool {
	for _, f := range s.Findings {
		if f.Severity == "blocker" {
			return true
		}
	}
	return false
}
