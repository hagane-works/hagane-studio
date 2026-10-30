package guirender

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/loc"
)

var GameLanguages = []string{"english", "braz_por", "french", "german", "japanese", "korean", "polish", "russian", "simp_chinese", "spanish"}

var siteLanguages = map[string]string{
	"en": "english", "zh": "simp_chinese", "ru": "russian", "ja": "japanese", "ko": "korean",
	"es": "spanish", "fr": "french", "de": "german", "pl": "polish", "pt": "braz_por",
}

func GameLanguage(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	code = strings.TrimPrefix(code, "l_")
	for _, lang := range GameLanguages {
		if code == lang {
			return lang
		}
	}
	if cut := strings.IndexAny(code, "-_"); cut > 0 {
		code = code[:cut]
	}
	return siteLanguages[code]
}

func (idx *Index) Language() string {
	if idx == nil || idx.lang == "" {
		return "english"
	}
	return idx.lang
}

func (idx *Index) dateInLanguage(year, month, day int, monthName string) string {
	switch idx.Language() {
	case "simp_chinese", "japanese":
		if !strings.HasSuffix(monthName, "月") {
			monthName = fmt.Sprintf("%d月", month)
		}
		return fmt.Sprintf("%d年%s%d日", year, monthName, day)
	case "korean":
		if !strings.HasSuffix(monthName, "월") {
			monthName = fmt.Sprintf("%d월", month)
		}
		return fmt.Sprintf("%d년 %s %d일", year, monthName, day)
	}
	return fmt.Sprintf("%d %s, %d", day, monthName, year)
}

func (idx *Index) InLanguage(lang string, base map[string]string) *Index {
	lang = GameLanguage(lang)
	if idx == nil || lang == "" || lang == "english" {
		return idx
	}
	var head, prev *Index
	for layer := idx; layer != nil; layer = layer.fallback {
		view := *layer
		view.lang = lang
		if layer.source == "vanilla" {

			view.langLoc = base
		} else {
			view.langLoc = layerLanguageText(layer, lang)
		}
		if prev == nil {
			head = &view
		} else {
			prev.fallback = &view
		}
		prev = &view
	}
	return head
}

func withLanguageVariants(node map[string]any, lang string) map[string]any {
	if lang == "" || lang == "english" {
		return node
	}
	children := nodeChildren(node)
	if len(children) == 0 {
		return node
	}
	suffix := "_" + lang
	replaced := map[string]bool{}
	for _, child := range children {
		if name, _ := child["name"].(string); strings.HasSuffix(strings.ToLower(name), suffix) {
			replaced[strings.ToLower(name[:len(name)-len(suffix)])] = true
		}
	}
	var out []map[string]any
	for i, child := range children {
		next := withLanguageVariants(child, lang)
		name, _ := child["name"].(string)
		lower := strings.ToLower(name)
		switch {
		case strings.HasSuffix(lower, suffix):
			next = withHide(next, false)
		case replaced[lower]:
			next = withHide(next, true)
		}
		if out == nil && !sameNode(next, child) {
			out = append(make([]map[string]any, 0, len(children)), children[:i]...)
		}
		if out != nil {
			out = append(out, next)
		}
	}
	if out == nil {
		return node
	}
	copied := make(map[string]any, len(node))
	for k, v := range node {
		copied[k] = v
	}
	copied["children"] = out
	return copied
}

func withHide(node map[string]any, hide bool) map[string]any {
	copied := make(map[string]any, len(node))
	for k, v := range node {
		copied[k] = v
	}
	raw := map[string]any{}
	if old, ok := node["raw"].(map[string]any); ok {
		for k, v := range old {
			raw[k] = v
		}
	}
	raw["hide"] = "no"
	if hide {
		raw["hide"] = "yes"
	}
	copied["raw"] = raw
	return copied
}

func sameNode(a, b map[string]any) bool {
	return fmt.Sprintf("%p", a) == fmt.Sprintf("%p", b)
}

func layerLanguageText(layer *Index, lang string) map[string]string {
	key := languageTextKey{layer: layer, lang: lang}
	languageTexts.mu.Lock()
	if entry, ok := languageTexts.entries[key]; ok {
		languageTexts.tick++
		entry.used = languageTexts.tick
		languageTexts.mu.Unlock()
		return entry.text
	}
	languageTexts.mu.Unlock()

	header := "l_" + lang
	text := map[string]string{}

	names := make([]string, 0, 16)
	for name := range layer.files {
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "localisation/") && strings.HasSuffix(lower, ".yml") {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(a, b int) bool {
		ra := strings.Contains(strings.ToLower(names[a]), "/replace/")
		rb := strings.Contains(strings.ToLower(names[b]), "/replace/")
		if ra != rb {
			return rb
		}
		return names[a] < names[b]
	})
	for _, name := range names {
		body := layer.files[name]
		decoded, ok := gfx.DecodeScriptText(body)
		if !ok {
			continue
		}
		language, entries, _ := loc.ParseLocText(decoded)
		if language != header {
			continue
		}
		for k, v := range entries {
			text[k] = v
		}
	}

	languageTexts.mu.Lock()
	defer languageTexts.mu.Unlock()
	languageTexts.tick++
	languageTexts.entries[key] = &languageTextEntry{text: text, used: languageTexts.tick}
	for len(languageTexts.entries) > languageTextsMax {
		var oldest languageTextKey
		var oldestUsed uint64
		first := true
		for k, entry := range languageTexts.entries {
			if first || entry.used < oldestUsed {
				oldest, oldestUsed, first = k, entry.used, false
			}
		}
		delete(languageTexts.entries, oldest)
	}
	return text
}

type languageTextKey struct {
	layer *Index
	lang  string
}

type languageTextEntry struct {
	text map[string]string
	used uint64
}

const languageTextsMax = 24

var languageTexts = struct {
	mu      sync.Mutex
	entries map[languageTextKey]*languageTextEntry
	tick    uint64
}{entries: map[languageTextKey]*languageTextEntry{}}
