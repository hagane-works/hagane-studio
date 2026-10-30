package exportcheck

import (
	"fmt"
	"strings"
)

func ValidateImportSourceFile(value string) (string, error) {
	path := NormalizeSourceFile(value)
	if path == "" {
		return "", nil
	}
	segments := strings.Split(path, "/")
	bad := len(segments) != 3 ||
		segments[0] != "common" || segments[1] != "units" ||
		!strings.HasSuffix(strings.ToLower(segments[len(segments)-1]), ".txt") ||
		strings.Contains(path, ":")
	if !bad {
		for _, seg := range segments {
			if seg == "" || seg == "." || seg == ".." {
				bad = true
				break
			}
		}
	}
	if bad {
		return "", fmt.Errorf("source_file must be a mod-relative common/units/*.txt path")
	}
	return path, nil
}
