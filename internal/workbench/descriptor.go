package workbench

import (
	"regexp"
	"strings"
)

type DescriptorParse struct {
	Name             *string  `json:"name"`
	Tags             []string `json:"tags"`
	Picture          *string  `json:"picture"`
	Version          *string  `json:"version"`
	SupportedVersion *string  `json:"supported_version"`
	ReplacePaths     []string `json:"replace_paths"`
}

var (
	descriptorTagsRE = regexp.MustCompile(`(?ms)^[ \t]*tags\s*=\s*\{([^}]*)\}`)
	descriptorQuoted = regexp.MustCompile(`"([^"]+)"`)

	descriptorReplacePathRE = regexp.MustCompile(`(?m)^[ \t]*replace_path[ \t]*=[ \t]*(?:"([^"\r\n]*)"|([^"#\s{}\r\n]+))[ \t]*(?:#[^\r\n]*)?$`)
	descriptorScalarRE      = map[string]*regexp.Regexp{}
)

func init() {
	for _, key := range []string{"name", "picture", "version", "supported_version"} {
		descriptorScalarRE[key] = regexp.MustCompile(`(?m)^\s*` + key + `\s*=\s*"([^"` + "\n" + `]*)"\s*$`)
	}
}

func descriptorScalar(text, key string) *string {
	m := descriptorScalarRE[key].FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	v := strings.TrimSpace(m[1])
	if v == "" {
		return nil
	}
	return &v
}

func ParseDescriptor(text string) DescriptorParse {
	if runes := []rune(text); len(runes) > 65536 {

		text = string(runes[:65536])
	}
	tags := []string{}
	if m := descriptorTagsRE.FindStringSubmatch(text); m != nil {
		for _, q := range descriptorQuoted.FindAllStringSubmatch(m[1], -1) {
			if t := strings.TrimSpace(q[1]); t != "" {
				tags = append(tags, t)
			}
		}
	}
	raw := []string{}
	for _, idx := range descriptorReplacePathRE.FindAllStringSubmatchIndex(text, -1) {
		if idx[2] >= 0 {
			raw = append(raw, text[idx[2]:idx[3]])
		} else if idx[4] >= 0 {
			raw = append(raw, text[idx[4]:idx[5]])
		}
	}
	return DescriptorParse{
		Name:             descriptorScalar(text, "name"),
		Tags:             tags,
		Picture:          descriptorScalar(text, "picture"),
		Version:          descriptorScalar(text, "version"),
		SupportedVersion: descriptorScalar(text, "supported_version"),
		ReplacePaths:     normalizeReplacePaths(raw),
	}
}
