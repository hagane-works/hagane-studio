package cosmetic

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Names map[string]map[string]string
type RGB [3]int

type Definition struct {
	Tag              string `json:"tag"`
	NameLocs         Names  `json:"name_locs"`
	ImportedNameLocs Names  `json:"imported_name_locs"`
	Color            *RGB   `json:"color"`
	ColorUI          *RGB   `json:"color_ui"`
	ImportedColor    *RGB   `json:"imported_color"`
	ImportedColorUI  *RGB   `json:"imported_color_ui"`
}

type ResolvedName struct {
	Value         string `json:"value"`
	Source        string `json:"source"`
	InheritedFrom string `json:"inherited_from"`
}

var suffixes = []string{"", "DEF", "ADJ", "democratic", "democratic_DEF", "democratic_ADJ", "fascism", "fascism_DEF", "fascism_ADJ", "neutrality", "neutrality_DEF", "neutrality_ADJ", "communism", "communism_DEF", "communism_ADJ"}
var locales = []string{"l_english", "l_simp_chinese", "l_russian", "l_french", "l_german", "l_spanish", "l_japanese", "l_korean", "l_polish", "l_braz_por"}

func Suffixes() []string { return append([]string(nil), suffixes...) }
func Locales() []string  { return append([]string(nil), locales...) }

func ValidateTag(tag string) error {
	if tag == "." || tag == ".." {
		return fmt.Errorf("cosmetic tag cannot be a reserved URL path segment")
	}
	if len(tag) < 1 || len(tag) > 100 {
		return fmt.Errorf("cosmetic tag must contain 1 to 100 ASCII characters")
	}
	for _, r := range tag {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-') {
			return fmt.Errorf("cosmetic tag contains an invalid character")
		}
	}
	return nil
}

func ValidateLocale(locale string) error {
	for _, allowed := range locales {
		if locale == allowed {
			return nil
		}
	}
	return fmt.Errorf("unsupported cosmetic name locale %q", locale)
}

func ValidateSuffix(suffix string) error {
	for _, allowed := range suffixes {
		if suffix == allowed {
			return nil
		}
	}
	return fmt.Errorf("unsupported cosmetic name suffix %q", suffix)
}

func ValidateName(locale, suffix, value string) error {
	if err := ValidateLocale(locale); err != nil {
		return err
	}
	if err := ValidateSuffix(suffix); err != nil {
		return err
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > 4096 {
		return fmt.Errorf("cosmetic name must contain at most 4096 valid Unicode codepoints")
	}
	if strings.ContainsAny(value, "\x00\r\n\v\f\x1c\x1d\x1e\u0085\u2028\u2029") {
		return fmt.Errorf("cosmetic name must be a single line")
	}
	return nil
}

func ValidateRGB(color RGB) error {
	for _, channel := range color {
		if channel < 0 || channel > 255 {
			return fmt.Errorf("RGB channels must be integers between 0 and 255")
		}
	}
	return nil
}

func (color *RGB) UnmarshalJSON(data []byte) error {
	var channels []*int
	if err := json.Unmarshal(data, &channels); err != nil {
		return fmt.Errorf("RGB must contain three integer channels: %w", err)
	}
	if len(channels) != 3 || channels[0] == nil || channels[1] == nil || channels[2] == nil {
		return fmt.Errorf("RGB must contain three integer channels")
	}
	next := RGB{*channels[0], *channels[1], *channels[2]}
	if err := ValidateRGB(next); err != nil {
		return err
	}
	*color = next
	return nil
}

func inheritance(suffix string) []string {
	if suffix == "" {
		return []string{""}
	}
	if suffix == "ADJ" || suffix == "DEF" {
		return []string{suffix, ""}
	}
	if strings.HasSuffix(suffix, "_ADJ") {
		return []string{suffix, "ADJ", ""}
	}
	if strings.HasSuffix(suffix, "_DEF") {
		return []string{suffix, strings.TrimSuffix(suffix, "_DEF"), "DEF", ""}
	}
	return []string{suffix, ""}
}

func ResolveName(def Definition, locale, suffix string) ResolvedName {
	for _, candidate := range inheritance(suffix) {
		if value, ok := def.NameLocs[locale][candidate]; ok && strings.TrimSpace(value) != "" {
			return ResolvedName{Value: value, Source: "authored", InheritedFrom: candidate}
		}
	}
	if value, ok := def.ImportedNameLocs[locale][suffix]; ok {
		return ResolvedName{Value: value, Source: "imported", InheritedFrom: suffix}
	}
	return ResolvedName{Value: Humanize(def.Tag), Source: "generated"}
}

func ResolveNames(def Definition, locale string) map[string]ResolvedName {
	names := make(map[string]ResolvedName, len(suffixes))
	for _, suffix := range suffixes {
		names[suffix] = ResolveName(def, locale, suffix)
	}
	return names
}

func Humanize(tag string) string {
	words := strings.FieldsFunc(tag, func(r rune) bool { return r == '_' || unicode.IsSpace(r) })
	for i, word := range words {
		runes := []rune(word)
		upper, hasCased := true, false
		for _, r := range runes {
			if unicode.IsLower(r) || unicode.IsTitle(r) {
				upper = false
			}
			hasCased = hasCased || unicode.IsUpper(r)
		}
		if !upper || !hasCased || len(runes) > 4 {
			words[i] = strings.ToUpper(string(runes[0])) + strings.ToLower(string(runes[1:]))
		}
	}
	if len(words) == 0 {
		return tag
	}
	return strings.Join(words, " ")
}

func ResolveColors(def Definition, ownerColor, ownerUI *RGB) (color, colorUI RGB) {
	digest := md5.Sum([]byte(strings.ToUpper(def.Tag)))
	for i := range color {
		color[i] = 40 + int(digest[i])*176/255
	}
	for _, candidate := range []*RGB{def.Color, def.ImportedColor, ownerColor} {
		if candidate != nil && ValidateRGB(*candidate) == nil {
			color = *candidate
			break
		}
	}
	colorUI = color
	for _, candidate := range []*RGB{def.ColorUI, def.ImportedColorUI, ownerUI} {
		if candidate != nil && ValidateRGB(*candidate) == nil {
			colorUI = *candidate
			break
		}
	}
	return color, colorUI
}
