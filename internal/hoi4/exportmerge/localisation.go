package exportmerge

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/emit/locout"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/loc"
)

var mergeLocales = []string{"en", "zh", "ru", "de", "es", "fr", "ja", "ko"}

var localeHeaders = map[string]string{
	"en": "l_english", "zh": "l_simp_chinese", "ru": "l_russian", "de": "l_german",
	"es": "l_spanish", "fr": "l_french", "ja": "l_japanese", "ko": "l_korean",
}

var localeDirs = map[string]string{
	"en": "english", "zh": "simp_chinese", "ru": "russian", "de": "german",
	"es": "spanish", "fr": "french", "ja": "japanese", "ko": "korean",
}

func mergeLocalisations(archive Archive, entries map[string]exportengine.Entry, resolved map[string]map[string]any) {
	affected := map[string]map[string]string{}
	for _, language := range mergeLocales {
		affected[language] = map[string]string{}
	}
	for key, row := range resolved {
		kind, _, _ := strings.Cut(key, "/")
		pairs := entityLocalisations(kind, row)
		for language, values := range pairs {
			for locKey, value := range values {
				affected[language][locKey] = value
			}
		}
	}

	for _, language := range mergeLocales {
		values := affected[language]
		if len(values) == 0 {
			continue
		}
		header := localeHeaders[language]
		sink := "localisation/" + localeDirs[language] + "/replace/zz_platform_merged_" + header + ".yml"
		placed := map[string]bool{}
		for _, item := range archive.Entries {
			if !strings.HasSuffix(strings.ToLower(item.Path), ".yml") || item.Path == sink {
				continue
			}
			text := strings.TrimPrefix(string(item.Bytes), "\ufeff")
			fileHeader, existing, _ := loc.ParseLocText(text)
			if fileHeader != header {
				continue
			}
			changed := false
			for key, value := range values {
				if _, exists := existing[key]; exists {
					existing[key] = value
					placed[key] = true
					changed = true
				}
			}
			if changed {
				entries[item.Path] = exportengine.Entry{Path: item.Path, Bytes: serializeLocalisation(header, existing), Origin: "differential_localisation", SourceEntity: "localisation_file:" + item.Path}
			}
		}
		newValues := map[string]string{}
		for key, value := range values {
			if !placed[key] {
				newValues[key] = value
			}
		}
		if len(newValues) == 0 {
			continue
		}
		sinkValues := map[string]string{}
		if raw, exists := archive.ByPath[sink]; exists {
			fileHeader, parsed, _ := loc.ParseLocText(strings.TrimPrefix(string(raw), "\ufeff"))
			if fileHeader == header {
				for key, value := range parsed {
					sinkValues[key] = value
				}
			}
		}
		for key, value := range newValues {
			sinkValues[key] = value
		}
		entries[sink] = exportengine.Entry{Path: sink, Bytes: serializeLocalisation(header, sinkValues), Origin: "differential_localisation", SourceEntity: "localisation_file:" + sink}
	}
}

func entityLocalisations(kind string, row map[string]any) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, language := range mergeLocales {
		out[language] = map[string]string{}
	}
	put := func(key, mapField, enField, zhField string, source map[string]any) {
		if key == "" {
			return
		}
		if values, ok := source[mapField].(map[string]any); ok {
			for _, language := range mergeLocales {
				if value := strings.TrimSpace(asString(values[language])); value != "" {
					out[language][key] = value
				}
			}
		}
		if value := strings.TrimSpace(asString(source[enField])); value != "" && out["en"][key] == "" {
			out["en"][key] = value
		}
		if value := strings.TrimSpace(asString(source[zhField])); value != "" && out["zh"][key] == "" {
			out["zh"][key] = value
		}
	}
	switch kind {
	case "character":
		put(asString(row["tag"]), "display_name_locs", "name_en", "name_zh", row)
	case "idea":
		id := asString(row["idea_id"])
		put(id, "display_name_locs", "name_en", "name_zh", row)
		put(id+"_desc", "description_locs", "desc_en", "desc_zh", row)
	case "focus_node":
		id := asString(row["focus_id"])
		put(id, "display_name_locs", "name_en", "name_zh", row)
		put(id+"_desc", "description_locs", "desc_en", "desc_zh", row)
		put(id+"_et_tt", "effect_tooltip_locs", "effect_tooltip_en", "effect_tooltip_zh", row)
	case "decision":
		id := asString(row["decision_id"])
		put(id, "display_name_locs", "name_en", "name_zh", row)
		put(id+"_desc", "description_locs", "desc_en", "desc_zh", row)
		put(id+"_et_tt", "effect_tooltip_locs", "effect_tooltip_en", "effect_tooltip_zh", row)
	case "event":
		id := asString(row["event_id"])
		put(id+".t", "display_name_locs", "title_en", "title_zh", row)
		put(id+".d", "description_locs", "desc_en", "desc_zh", row)
		if options, ok := row["options"].([]any); ok {
			for _, option := range options {
				if value, ok := option.(map[string]any); ok {
					key := asString(value["name"])
					put(key, "display_name_locs", "name_en", "name_zh", value)
					put(key+"_tt", "effect_tooltip_locs", "effect_tooltip_en", "effect_tooltip_zh", value)
				}
			}
		}
	}
	return out
}

func serializeLocalisation(header string, values map[string]string) []byte {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var body strings.Builder
	body.WriteString(header)
	body.WriteString(":\n")
	for _, key := range keys {

		value := locout.YmlEscapeValue(values[key])
		body.WriteByte(' ')
		body.WriteString(key)
		body.WriteString(": \"")
		body.WriteString(value)
		body.WriteString("\"\n")
	}
	return append([]byte{0xef, 0xbb, 0xbf}, []byte(body.String())...)
}
