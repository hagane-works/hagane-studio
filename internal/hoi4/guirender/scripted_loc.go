package guirender

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var scriptedCallRE = regexp.MustCompile(`\[([A-Za-z?][A-Za-z0-9_.?]*)(?:\|[^\]]*)?\]`)

var literalTagRE = regexp.MustCompile(`^[A-Z][A-Z0-9]{2}$`)

func (idx *Index) LocaliseFor(key, root, date string) (string, bool) {
	value, ok := idx.locValue(key)
	if !ok {
		return "", false
	}
	value = idx.resolveLocRefs(value, 0, map[string]bool{key: true})
	return cleanMarkup(idx.resolveScripted(value, root, date)), true
}

func (idx *Index) resolveScripted(value, root, date string) string {
	if !strings.Contains(value, "[") {
		return value
	}
	return scriptedCallRE.ReplaceAllStringFunc(value, func(match string) string {
		inner := scriptedCallRE.FindStringSubmatch(match)[1]
		parts := strings.Split(inner, ".")

		if len(parts) <= 2 {
			if text := idx.definedTextNamed(parts[len(parts)-1]); text != nil {
				tag := strings.ToUpper(strings.TrimSpace(root))
				if len(parts) == 2 {
					tag = idx.scopeTag(parts[0], root)
				}
				if tag == "" {
					return match
				}
				if key, ok := idx.definedTextFor(text, tag, date); ok {
					if value, found := idx.locValue(key); found {
						return idx.resolveScripted(idx.resolveLocRefs(value, 0, map[string]bool{key: true}), root, date)
					}
				}
				return match
			}
		}
		if len(parts) < 2 {
			return match
		}
		tag := idx.scopeTag(parts[0], root)
		if tag == "" {
			return match
		}
		getter := parts[len(parts)-1]
		if len(parts) == 3 && strings.EqualFold(parts[1], "Capital") {
			if text, ok := idx.capitalGetter(tag, getter); ok {
				return text
			}
			return match
		}
		if len(parts) != 2 {
			return match
		}
		if text, ok := idx.countryGetter(tag, getter, date); ok {
			return text
		}
		return match
	})
}

func (idx *Index) scopeTag(scope, root string) string {
	switch strings.ToUpper(scope) {
	case "ROOT", "THIS":
		return strings.ToUpper(strings.TrimSpace(root))
	}

	tag := strings.ToUpper(scope)
	if !literalTagRE.MatchString(tag) {
		return ""
	}
	for layer := idx; layer != nil; layer = layer.fallback {
		if _, ok := layer.countryTags[tag]; ok {
			return tag
		}
	}
	return ""
}

func (idx *Index) countryGetter(tag, getter, date string) (string, bool) {
	party := strings.ToLower(idx.RulingPartyOf(tag))
	byParty := func(suffix string) (string, bool) {
		if party != "" {
			if text, ok := idx.Localise(tag + "_" + party + suffix); ok {
				return text, true
			}
		}
		return idx.Localise(tag + suffix)
	}
	switch strings.ToLower(getter) {
	case "getname", "getnamewithflag":
		return idx.CountryName(tag, party), true
	case "getnamedef":
		if text, ok := byParty("_DEF"); ok {
			return text, true
		}
		return idx.CountryName(tag, party), true
	case "getnamedefcap":
		text, ok := byParty("_DEF")
		if !ok {
			text = idx.CountryName(tag, party)
		}
		return capitalizeFirst(text), true
	case "getadjective":
		if text, ok := byParty("_ADJ"); ok {
			return text, true
		}
		return tag, true
	case "getadjectivecap":
		if text, ok := byParty("_ADJ"); ok {
			return capitalizeFirst(text), true
		}
		return tag, true
	case "getnonideologyname":
		return idx.CountryName(tag, ""), true
	case "getnonideologynamedef":
		if text, ok := idx.Localise(tag + "_DEF"); ok {
			return text, true
		}
		return idx.CountryName(tag, ""), true
	case "getleader":
		if name := idx.countryLeaderName(tag, date); name != "" {
			return name, true
		}
		return "", false
	case "getfactionname":
		if name, _, _ := idx.FactionOf(tag, date); name != "" {
			return name, true
		}
		return "", false
	case "getrulingparty":
		if text, ok := idx.Localise(tag + "_" + party + "_party"); ok && party != "" {
			return text, true
		}
		if text, ok := idx.Localise(party); ok {
			return text, true
		}
		return "", false
	case "getrulingpartylong":
		if text, ok := idx.Localise(tag + "_" + party + "_party_long"); ok && party != "" {
			return text, true
		}
		return idx.countryGetter(tag, "GetRulingParty", date)
	case "gettag":
		return tag, true
	case "getflag":

		return "", true
	}
	return "", false
}

func (idx *Index) capitalGetter(tag, getter string) (string, bool) {
	switch strings.ToLower(getter) {
	case "getname", "getcapitalvictorypointname":
	default:
		return "", false
	}
	politics := idx.countryPoliticsFor(tag)
	if politics == nil || politics.Capital == 0 {
		return "", false
	}
	for _, state := range idx.effectiveStates() {
		if state.ID != politics.Capital {
			continue
		}
		if label, ok := idx.Localise(state.Name); ok {
			return label, true
		}
		return "", false
	}
	return "", false
}

func (idx *Index) countryLeaderName(tag, date string) string {
	chosen := idx.countryLeader(tag, date)
	if chosen == nil {
		return ""
	}
	return idx.advisorName(chosen)
}

func (idx *Index) countryLeader(tag, date string) *Character {

	var chosen *Character
	chosenDepth := 0
	for _, person := range idx.peopleOf(tag, date) {
		for _, role := range person.Roles {
			if role != "country_leader" {
				continue
			}
			if person.Ideology != "" && idx.PartyOfIdeology(person.Ideology) != idx.RulingPartyOf(tag) {
				continue
			}
			if depth := idx.characterDepth(person.Token); chosen == nil || depth < chosenDepth {
				chosen, chosenDepth = person, depth
			}
		}
	}
	return chosen
}

func capitalizeFirst(text string) string {
	first, size := utf8.DecodeRuneInString(text)
	if size == 0 {
		return text
	}
	return string(unicode.ToUpper(first)) + text[size:]
}
