package exportcheck

func (idx RegistryIndex) Blocked() bool {
	for _, f := range idx.Findings {
		if f.Severity == "blocker" {
			return true
		}
	}
	return false
}

func (d FocusFileDecision) Blocked() bool {
	for _, f := range d.Findings {
		if f.Severity == "blocker" {
			return true
		}
	}
	return false
}

func (r LayoutResolution) Blocked() bool {
	for _, f := range r.Findings {
		if f.Severity == "blocker" {
			return true
		}
	}
	return false
}
