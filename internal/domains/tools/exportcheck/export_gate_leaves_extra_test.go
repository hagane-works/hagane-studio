package exportcheck

import (
	"regexp"
	"strings"
)

var replacePathAssignmentRe = regexp.MustCompile(
	`(?m)^[ \t]*replace_path[ \t]*=[ \t]*` +
		`(?:"([^"\r\n]*)"|([^"#\s{}\r\n]+))` +
		`[ \t]*(?:#[^\r\n]*)?$`)

func ParseDescriptorReplacePaths(descriptorText string) []string {
	if descriptorText == "" {
		return []string{}
	}
	var raw []string
	for _, m := range replacePathAssignmentRe.FindAllStringSubmatch(descriptorText, -1) {

		if strings.Contains(m[0], `"`) {
			raw = append(raw, m[1])
		} else {
			raw = append(raw, m[2])
		}
	}
	return NormalizeDescriptorReplacePaths(raw)
}
