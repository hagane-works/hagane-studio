package toolapi

import (
	"context"
	"strings"
)

const defaultContentLocale = "en"

type localeCtxKey struct{}

func withContentLocale(ctx context.Context, locale string) context.Context {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if locale == "" {
		return ctx
	}
	return context.WithValue(ctx, localeCtxKey{}, locale)
}

func contentLocaleFrom(ctx context.Context) string {
	if locale, ok := ctx.Value(localeCtxKey{}).(string); ok && locale != "" {
		return locale
	}
	return defaultContentLocale
}

func locsValue(locs map[string]any, lang string) string {
	if locs == nil {
		return ""
	}
	if text, ok := locs[lang].(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func resolveDisplayName(locale string, locs map[string]any, legacyEn, legacyZh, fallback string) (string, string) {
	legacyEn = strings.TrimSpace(legacyEn)
	legacyZh = strings.TrimSpace(legacyZh)
	if active := locsValue(locs, locale); active != "" {
		return active, locale
	}
	if locale == "zh" {
		if legacyZh != "" {
			return legacyZh, "zh"
		}
	}
	if english := locsValue(locs, defaultContentLocale); english != "" {
		return english, defaultContentLocale
	}
	if legacyEn != "" {
		return legacyEn, defaultContentLocale
	}
	return fallback, ""
}

func legacyNameColumns(entity map[string]any) (string, string) {
	pick := func(keys ...string) string {
		for _, key := range keys {
			if text, ok := entity[key].(string); ok && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text)
			}
		}
		return ""
	}
	return pick("name_en", "title_en"), pick("name_zh", "title_zh")
}

func displayNameFallback(entity map[string]any) string {
	for _, key := range []string{"name", "title", "tag", "namespace", "token", "project_token"} {
		if text, ok := entity[key].(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	for key, value := range entity {
		if !strings.HasSuffix(key, "_id") || key == "project_id" {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func mergeLocs(base, overlay map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(overlay))
	for _, source := range []map[string]any{base, overlay} {
		for lang, value := range source {
			text, ok := value.(string)
			if !ok {
				continue
			}
			if text = strings.TrimSpace(text); text != "" {
				out[lang] = text
			}
		}
	}
	return out
}

func enrichDisplayName(entity map[string]any, locale string) bool {
	if entity == nil {
		return false
	}
	locs, ok := entity["display_name_locs"].(map[string]any)
	if !ok {
		return false
	}
	if existing, ok := entity["display_name"].(string); ok && strings.TrimSpace(existing) != "" {
		return false
	}
	legacyEn, legacyZh := legacyNameColumns(entity)
	if upstream, ok := entity["display_name"].(map[string]any); ok {
		merged := mergeLocs(locs, upstream)
		entity["display_name"] = merged
		_, resolved := resolveDisplayName(locale, merged, legacyEn, legacyZh, displayNameFallback(entity))
		entity["resolved_locale"] = resolved
		return true
	}
	name, resolved := resolveDisplayName(locale, locs, legacyEn, legacyZh, displayNameFallback(entity))
	entity["display_name"] = name

	entity["resolved_locale"] = resolved
	return true
}

func displayNameText(entity map[string]any) string {
	switch typed := entity["display_name"].(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if lang, ok := entity["resolved_locale"].(string); ok && lang != "" {
			if text := locsValue(typed, lang); text != "" {
				return text
			}
		}
		return locsValue(typed, defaultContentLocale)
	}
	return ""
}

func enrichDisplayNames(value any, locale string) {
	switch typed := value.(type) {
	case map[string]any:
		enrichDisplayName(typed, locale)
		for _, nested := range typed {
			enrichDisplayNames(nested, locale)
		}
	case []any:
		for _, item := range typed {
			enrichDisplayNames(item, locale)
		}
	case []map[string]any:
		for _, item := range typed {
			enrichDisplayNames(item, locale)
		}
	}
}
