package exportengine

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportmsg"
)

const flagTGADescriptorBottomLeft = 0x08

func countryFlagHasSomethingOnRecord(country *worldout.OMap) bool {
	if country == nil {
		return false
	}
	if strings.TrimSpace(worldString(country, "flag_image_path")) != "" {
		return true
	}
	if worldValueNonEmpty(country, "flag_paths_by_ideology") {
		return true
	}
	return worldBool(country, "has_obs")
}

func worldBool(row *worldout.OMap, field string) bool {
	value, ok := row.Get(field)
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		return s == "true" || s == "t" || s == "1"
	}
	return false
}

func worldValueNonEmpty(row *worldout.OMap, field string) bool {
	value, ok := row.Get(field)
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case *worldout.OMap:
		return v.Len() > 0
	case []worldout.Value:
		return len(v) > 0
	case bool:
		return v
	}
	return true
}

const flagsMissingWarningLimit = 8

func flagsMissingWarning(tags []string, msg exportmsg.Printer) string {
	unique := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		tag = strings.ToUpper(strings.TrimSpace(tag))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		unique = append(unique, tag)
	}
	if len(unique) == 0 {
		return ""
	}
	sort.Strings(unique)
	return msg.Text("flags.missing", exportmsg.Args{"count": len(unique), "tags": joinCapped(unique, flagsMissingWarningLimit, msg)})
}
