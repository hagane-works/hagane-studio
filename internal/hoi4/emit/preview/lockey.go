package preview

import "strings"

func locKeyAllowedRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') || r == '_' || r == '.' || r == '\'' || r == '-'
}

func IsLegalLocKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if !locKeyAllowedRune(r) {
			return false
		}
	}
	return true
}

func SanitizeLocKey(key string) string {
	if key == "" {
		return ""
	}
	folded := stripCombining(pyNFKD(key))
	var b strings.Builder
	for _, r := range folded {
		if locKeyAllowedRune(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "_"
	}
	return out
}

func CharacterLocKey(ent *OD) string {
	tag := pyStrip(getStr(ent, "tag"))
	if IsLegalLocKey(tag) {
		return tag
	}
	name := pyStrip(getStr(ent, "name"))
	if IsLegalLocKey(name) {
		return name
	}
	return SanitizeLocKey(tag)
}

func EventOptionLocKey(opt *OD, eventID string, optIdx int) (string, string) {
	raw := ""
	if opt != nil {
		if v, ok := opt.Get("name"); ok && v != nil {
			raw = pyStrip(orEmptyStr(v))
		}
	}
	reservedT := eventID + ".t"
	reservedD := eventID + ".d"
	if IsLegalLocKey(raw) && raw != reservedT && raw != reservedD {
		return raw, ""
	}
	var letter string
	if optIdx < 26 {
		letter = string(rune('a' + optIdx))
	} else {
		letter = "opt" + pyStr(bigFromInt(optIdx))
	}
	derived := eventID + "." + letter
	if derived == reservedT || derived == reservedD {
		derived = eventID + ".opt" + letter
	}
	literal := ""
	if raw != "" && !IsLegalLocKey(raw) {
		literal = raw
	}
	return derived, literal
}
