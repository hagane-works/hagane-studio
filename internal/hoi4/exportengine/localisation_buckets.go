package exportengine

import (
	"fmt"
	"strings"

	"haganeworks/internal/hoi4/loc"
)

func BuildNativeLocalisationBuckets(snapshot Snapshot, languages []string, includeImported bool) (buckets map[string]map[string]string, err error) {
	defer func() {
		if value := recover(); value != nil {
			buckets = nil
			err = fmt.Errorf("native localisation collection rejected project data: %v", value)
		}
	}()
	wanted := map[string]bool{"en": true}
	for _, language := range languages {
		if language = strings.ToLower(strings.TrimSpace(language)); language != "" {
			wanted[language] = true
		}
	}
	buckets = make(map[string]map[string]string, len(wanted))
	for language := range wanted {
		buckets[language] = map[string]string{}
	}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, includeImported, manifestOnlyResolver)
	if err != nil {
		return nil, err
	}
	collectLocalisationBuckets(artifact.Entries, wanted, buckets)
	return buckets, nil
}

func collectLocalisationBuckets(entries []Entry, wanted map[string]bool, buckets map[string]map[string]string) {
	for _, entry := range entries {
		path := strings.ToLower(strings.ReplaceAll(entry.Path, `\`, "/"))
		if !strings.HasSuffix(path, ".yml") ||
			!(strings.HasPrefix(path, "localisation/") || strings.Contains(path, "/localisation/")) {
			continue
		}
		header, entries, _ := loc.ParseLocText(string(entry.Bytes))
		language, known := loc.LanguageToSuffix(header)
		if !known || !wanted[language] {
			continue
		}
		for key, value := range entries {
			if _, exists := buckets[language][key]; !exists {
				buckets[language][key] = value
			}
		}
	}
}
