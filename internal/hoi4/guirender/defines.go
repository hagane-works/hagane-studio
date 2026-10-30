package guirender

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	defineGroupRE    = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\{`)
	defineValueRE    = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*,?\s*$`)
	defineOverrideRE = regexp.MustCompile(`^\s*NDefines\.([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*;?\s*$`)
)

func (idx *Index) indexDefinesFile(body []byte) {
	depth, group := 0, ""
	for _, line := range strings.Split(string(body), "\n") {
		if at := strings.Index(line, "--"); at >= 0 {
			line = line[:at]
		}
		if match := defineOverrideRE.FindStringSubmatch(line); match != nil {
			if value, err := strconv.ParseFloat(match[3], 64); err == nil {
				idx.defines[match[1]+"."+match[2]] = value
			}
			continue
		}
		if depth == 1 {
			if match := defineGroupRE.FindStringSubmatch(line); match != nil {
				group = match[1]
			}
		}
		if depth == 2 && group != "" {
			if match := defineValueRE.FindStringSubmatch(line); match != nil {
				if value, err := strconv.ParseFloat(match[2], 64); err == nil {
					idx.defines[group+"."+match[1]] = value
				}
			}
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth < 0 {
			depth = 0
		}
		if depth <= 1 {
			group = ""
		}
	}
}

func (idx *Index) defineNumber(name string) float64 {
	for layer := idx; layer != nil; layer = layer.fallback {
		if value, ok := layer.defines[name]; ok {
			return value
		}
	}
	return 0
}
