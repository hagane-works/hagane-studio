package toolapi

func intField(m map[string]any, key string) int64 {
	if n, ok := m[key].(float64); ok {
		return int64(n)
	}
	return 0
}

type DanglingReference struct {
	SourceKind string `json:"source_kind"`
	SourceID   int64  `json:"source_id"`
	SourceName string `json:"source_name"`
	FieldName  string `json:"field_name"`
	EffectName string `json:"effect_name"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}
