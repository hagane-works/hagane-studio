package exportengine

import (
	"encoding/json"
	"strings"
)

func InheritedReplacePaths(snapshot Snapshot) []string {
	if len(snapshot.Project) == 0 {
		return nil
	}
	var project map[string]json.RawMessage
	if err := json.Unmarshal(snapshot.Project, &project); err != nil {
		return nil
	}
	raw, ok := project["imported_descriptor_replace_paths"]
	if !ok {
		return nil
	}
	return decodeReplacePathList(raw)
}

func decodeReplacePathList(raw json.RawMessage) []string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return list
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if strings.HasPrefix(text, "[") {
		var nested []string
		if err := json.Unmarshal([]byte(text), &nested); err == nil {
			return nested
		}
		return nil
	}
	return []string{text}
}
