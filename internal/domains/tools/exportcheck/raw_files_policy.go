package exportcheck

import (
	"strings"
)

func SanitizeDestPath(raw string) (string, string) {
	s := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
	if s == "" {
		return "", "dest_path is required (e.g. 'common/on_actions/TNO_Reich_on_actions.txt')"
	}
	if len(s) >= 2 && s[1] == ':' {
		return "", "dest_path must be mod-relative (no drive letter)"
	}
	s = strings.TrimLeft(s, "/")
	var parts []string
	for _, seg := range strings.Split(s, "/") {
		if seg == "" || seg == "." {
			continue
		}
		if seg == ".." {
			return "", "dest_path may not contain '..' (path traversal)"
		}
		parts = append(parts, seg)
	}
	if len(parts) == 0 {
		return "", "dest_path is empty after normalisation"
	}
	if !strings.Contains(parts[len(parts)-1], ".") {
		return "", "dest_path must point at a file (e.g. common/on_actions/TNO_Reich_on_actions.txt)"
	}
	return strings.Join(parts, "/"), ""
}
