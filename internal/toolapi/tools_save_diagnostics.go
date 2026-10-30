package toolapi

import (
	"encoding/json"
	"haganeworks/internal/hoi4/savediagnostics"
)

type saveDiagnostic = savediagnostics.Diagnostic

func saveDiagnosticsField(payload map[string]any) *[]saveDiagnostic {
	return namedSaveDiagnosticsField(payload, "diagnostics")
}

func namedSaveDiagnosticsField(payload map[string]any, name string) *[]saveDiagnostic {
	value, present := payload[name]
	if !present || value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var result []saveDiagnostic
	if json.Unmarshal(raw, &result) != nil || result == nil {
		return nil
	}
	return &result
}
