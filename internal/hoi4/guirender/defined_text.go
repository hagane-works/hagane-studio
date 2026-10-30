package guirender

import (
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type definedText struct {
	options []definedTextOption
}

type definedTextOption struct {
	trigger *paradox.Block
	key     string

	random bool
}

func (idx *Index) indexScriptedLocFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "defined_text") {
			continue
		}
		name := strings.Trim(ciScalar(entry.Block.Entries, "name"), "\"")
		if name == "" {
			continue
		}
		text := &definedText{}
		for _, field := range entry.Block.Entries {
			if field == nil || field.Key == nil || field.Block == nil || !strings.EqualFold(*field.Key, "text") {
				continue
			}
			option := definedTextOption{key: strings.Trim(ciScalar(field.Block.Entries, "localization_key"), "\"")}
			for _, part := range field.Block.Entries {
				if part == nil || part.Key == nil || part.Block == nil {
					continue
				}
				switch strings.ToLower(*part.Key) {
				case "trigger":
					option.trigger = part.Block
				case "random_list":
					option.random = true
				}
			}
			text.options = append(text.options, option)
		}
		idx.definedTexts[strings.ToLower(name)] = text
	}
}

func (idx *Index) definedTextNamed(name string) *definedText {
	key := strings.ToLower(name)
	for layer := idx; layer != nil; layer = layer.fallback {
		if text, ok := layer.definedTexts[key]; ok {
			return text
		}
	}
	return nil
}

func (idx *Index) definedTextFor(text *definedText, tag, date string) (string, bool) {
	var state *decisionStart
	for _, option := range text.options {
		if option.random {
			return "", false
		}
		if option.trigger != nil {
			if state == nil {
				start := idx.decisionStartOf(tag, date)
				state = &start
			}
			holds, decided := state.verdict(option.trigger)
			if !decided {
				return "", false
			}
			if !holds {
				continue
			}
		}
		return option.key, option.key != ""
	}
	return "", false
}
