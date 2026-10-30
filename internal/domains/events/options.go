package events

import (
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"

	"haganeworks/internal/hoi4/emit/locout"
)

var (
	slugNonWord = regexp.MustCompile(`[^\p{L}\p{N}_\s-]`)
	slugSpaces  = regexp.MustCompile(`[\s-]+`)
	slugUnders  = regexp.MustCompile(`_+`)
)

func slugifyID(text string) string {
	t := strings.TrimSpace(strings.ToLower(text))
	t = norm.NFKD.String(t)
	t = slugNonWord.ReplaceAllString(t, "")
	t = slugSpaces.ReplaceAllString(t, "_")
	t = slugUnders.ReplaceAllString(t, "_")
	t = strings.Trim(t, "_")
	if t == "" {
		return "unnamed"
	}
	return t
}

func eventPictureSpriteName(countryTag, eventID string) string {
	return "GFX_report_event_" + strings.ToUpper(countryTag) + "_" + slugifyID(eventID)
}

func cleanLocsAny(v any) map[string]string {
	out := map[string]string{}
	if m, ok := v.(map[string]any); ok {
		for lang, raw := range m {
			if s, ok := raw.(string); ok {
				if t := strings.TrimSpace(s); t != "" {
					out[lang] = t
				}
			}
		}
	}
	return out
}

func mergeLegacyLocsAny(v, en, zh any) map[string]string {
	locs := cleanLocsAny(v)
	for lang, raw := range map[string]any{"en": en, "zh": zh} {
		s, _ := raw.(string)
		if t := strings.TrimSpace(s); t != "" && locs[lang] == "" {
			locs[lang] = t
		}
	}
	return locs
}

func locsToAny(m map[string]string) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func normalizeEventOptions(options []any) []any {
	normalized := make([]any, 0, len(options))
	for _, raw := range options {
		m, ok := raw.(map[string]any)
		if !ok {
			normalized = append(normalized, raw)
			continue
		}
		opt := map[string]any{}
		for k, v := range m {
			opt[k] = v
		}
		names := mergeLegacyLocsAny(opt["display_name_locs"], opt["name_en"], opt["name_zh"])
		displayText := ""
		if s, ok := opt["display_text"].(string); ok {
			displayText = strings.TrimSpace(s)
		}
		if displayText != "" && names["en"] == "" {
			names["en"] = displayText
		}
		if len(names) > 0 {
			opt["display_name_locs"] = locsToAny(names)
			if _, present := opt["name_en"]; !present {
				opt["name_en"] = names["en"]
			}
			if _, present := opt["name_zh"]; !present {
				opt["name_zh"] = names["zh"]
			}
		}
		tooltips := mergeLegacyLocsAny(opt["effect_tooltip_locs"], opt["effect_tooltip_en"], opt["effect_tooltip_zh"])
		if len(tooltips) > 0 {
			opt["effect_tooltip_locs"] = locsToAny(tooltips)
			if _, present := opt["effect_tooltip_en"]; !present {
				opt["effect_tooltip_en"] = tooltips["en"]
			}
			if _, present := opt["effect_tooltip_zh"]; !present {
				opt["effect_tooltip_zh"] = tooltips["zh"]
			}
		}
		normalized = append(normalized, opt)
	}
	return normalized
}

func normalizeEventOptionsWithNameBackfill(options []any) []any {
	prepped := make([]any, 0, len(options))
	for _, raw := range options {
		m, ok := raw.(map[string]any)
		if !ok {
			prepped = append(prepped, raw)
			continue
		}
		opt := map[string]any{}
		for k, v := range m {
			opt[k] = v
		}
		bareName, _ := opt["name"].(string)
		nameEn, _ := opt["name_en"].(string)
		displayText, _ := opt["display_text"].(string)

		if name := strings.TrimSpace(bareName); name != "" && !locout.IsLegalLocKey(name) &&
			strings.TrimSpace(nameEn) == "" && strings.TrimSpace(displayText) == "" {
			opt["name_en"] = name
		}
		prepped = append(prepped, opt)
	}
	return normalizeEventOptions(prepped)
}
