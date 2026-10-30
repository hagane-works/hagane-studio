package exportcheck

import (
	"strings"
)

type BaseInstall struct {
	Dependency  string `json:"dependency"`
	Matched     bool   `json:"matched"`
	DisplayName string `json:"display_name"`
	Version     string `json:"version"`
	WorkshopURL string `json:"workshop_url"`
	ModKey      string `json:"mod_key"`
}

func NormalizeBaseDependencies(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			continue
		}
		if text = strings.TrimSpace(text); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func ResolveBaseInstalls(dependencies []string, templates []BasemodTemplate) []BaseInstall {
	out := make([]BaseInstall, 0, len(dependencies))
	for _, dependency := range dependencies {
		var matched *BasemodTemplate
		for index := range templates {
			template := &templates[index]
			if dependency == strings.TrimSpace(template.DisplayName) || containsBaseInstallName(template.DescriptorNames, dependency) {
				matched = template
				break
			}
		}
		item := BaseInstall{Dependency: dependency, DisplayName: dependency}
		if matched != nil {
			item.Matched = true
			if matched.DisplayName != "" {
				item.DisplayName = matched.DisplayName
			}
			item.Version = matched.Version
			if matched.WorkshopURL != nil {
				item.WorkshopURL = *matched.WorkshopURL
			}
			item.ModKey = matched.ModKey
		}
		out = append(out, item)
	}
	return out
}

func containsBaseInstallName(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
