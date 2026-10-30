package guirender

import (
	"strconv"
	"strings"
)

func (idx *Index) indexModifierDocs(body []byte) {
	if idx.modifierDecimals == nil {
		idx.modifierDecimals = map[string]int{}
	}
	current := ""
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if heading, ok := strings.CutPrefix(line, "## "); ok {
			current = strings.ToLower(strings.TrimSpace(heading))
			continue
		}
		rest, ok := strings.CutPrefix(line, "* Number with ")
		if current == "" || !ok {
			continue
		}
		if fields := strings.Fields(rest); len(fields) > 0 {
			if n, err := strconv.Atoi(fields[0]); err == nil {
				idx.modifierDecimals[current] = n
			}
		}
	}
}

func (idx *Index) modifierDecimalsOf(key string) (int, bool) {
	key = strings.ToLower(key)
	for layer := idx; layer != nil; layer = layer.fallback {
		if n, ok := layer.modifierDecimals[key]; ok {
			return n, true
		}
	}
	return 0, false
}
