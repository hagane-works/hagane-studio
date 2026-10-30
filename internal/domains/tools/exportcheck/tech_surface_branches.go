package exportcheck

func stringFrom(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
