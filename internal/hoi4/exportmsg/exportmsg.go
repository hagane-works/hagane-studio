package exportmsg

import (
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var Locales = []string{"en", "zh", "ru", "de", "es", "fr", "ja", "ko"}

//go:embed messages/*.json
var files embed.FS

var catalogs = load()

func load() map[string]map[string]string {
	out := make(map[string]map[string]string, len(Locales))
	for _, locale := range Locales {
		data, err := files.ReadFile("messages/" + locale + ".json")
		if err != nil {
			panic(err)
		}
		var catalog map[string]string
		if err := json.Unmarshal(data, &catalog); err != nil {
			panic(fmt.Sprintf("exportmsg: messages/%s.json: %v", locale, err))
		}
		out[locale] = catalog
	}
	return out
}

type Printer struct{ locale string }

func For(locale string) Printer {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if _, ok := catalogs[locale]; !ok || locale == "en" {
		return Printer{}
	}
	return Printer{locale: locale}
}

func (p Printer) Locale() string {
	if p.locale == "" {
		return "en"
	}
	return p.locale
}

type Args map[string]any

type Message struct {
	Key  string
	Args Args
}

var placeholder = regexp.MustCompile(`\{([a-z][a-z0-9_]*)\}`)

func (p Printer) Text(key string, args Args) string {
	template, ok := catalogs[p.Locale()][key]
	if !ok {
		return key
	}
	return placeholder.ReplaceAllStringFunc(template, func(match string) string {
		value, ok := args[match[1:len(match)-1]]
		if !ok {
			return match
		}
		if message, ok := value.(Message); ok {
			return p.Render(message)
		}
		return fmt.Sprint(value)
	})
}

func (p Printer) Render(m Message) string { return p.Text(m.Key, m.Args) }
