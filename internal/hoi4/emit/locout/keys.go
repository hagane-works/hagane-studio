package locout

import "unicode"

func locKeyAllowedRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') || r == '_' || r == '.' || r == '\'' || r == '-'
}

func matchLocKeyAllowed(s string) bool {
	body := s
	if n := len(body); n > 0 && body[n-1] == '\n' {
		body = body[:n-1]
	}
	if body == "" {
		return false
	}
	for _, r := range body {
		if !locKeyAllowedRune(r) {
			return false
		}
	}
	return true
}

func IsLegalLocKey(key string) bool {
	return key != "" && matchLocKeyAllowed(key)
}

func SanitizeLocKey(key string) string {
	if key == "" {
		return ""
	}
	folded := pyNFKDStripCombining(key)
	out := make([]rune, 0, len(folded))
	for _, r := range folded {
		if locKeyAllowedRune(r) {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "_"
	}
	return string(out)
}

func CharacterLocKey(char *Obj) string {
	tag := pyStrip(strOrEmpty(char.Attr("tag")))
	if IsLegalLocKey(tag) {
		return tag
	}
	name := pyStrip(strOrEmpty(char.Attr("name")))
	if IsLegalLocKey(name) {
		return name
	}
	return SanitizeLocKey(tag)
}

func EventOptionLocKey(opt Value, eventID string, optIdx int) (string, string) {
	raw := ""
	if opt.Kind == KObj {
		raw = pyStrip(strOrEmpty(opt.Obj.Attr("name")))
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
		letter = "opt" + itoa(optIdx)
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [24]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func machineTokenID(ident string) bool {
	for _, r := range ident {
		if r == '_' || r == '.' || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
