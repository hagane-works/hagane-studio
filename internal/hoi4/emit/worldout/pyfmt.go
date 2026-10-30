package worldout

import (
	"crypto/md5"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

func pyStrip(s string) string { return strings.TrimSpace(s) }

func pySplitlines(s string) []string {
	var out []string
	var cur strings.Builder
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch r {
		case '\r':
			out = append(out, cur.String())
			cur.Reset()
			if i+1 < len(rs) && rs[i+1] == '\n' {
				i++
			}
		case '\n', '\v', '\f', '\x1c', '\x1d', '\x1e', '\u0085', '\u2028', '\u2029':
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func fmtHoi4Float(v float64) string {
	if math.IsNaN(v) {
		return "nan"
	}
	if math.IsInf(v, 1) {
		return "inf"
	}
	if math.IsInf(v, -1) {
		return "-inf"
	}
	if v == math.Trunc(v) {
		if v == 0 {
			return "0"
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	text := strconv.FormatFloat(v, 'f', 12, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" {
		return "0"
	}
	return text
}

func pyG(v Value) string {
	if n, ok := v.(json.Number); ok {
		if !numIsFloatLiteral(n) {
			return n.String()
		}
		f, _ := n.Float64()
		return strconv.FormatFloat(f, 'g', 6, 64)
	}
	f, _ := numFloat(v)
	return strconv.FormatFloat(f, 'g', 6, 64)
}

func hoi4FloatOf(v Value) string {
	f, _ := numFloat(v)
	return fmtHoi4Float(f)
}

func pyIsUpper(s string) bool {
	hasCased := false
	for _, r := range s {
		if unicode.IsLower(r) || unicode.IsTitle(r) {
			return false
		}
		if unicode.IsUpper(r) {
			hasCased = true
		}
	}
	return hasCased
}

func pyCapWord(w string) string {
	rs := []rune(w)
	if len(rs) == 0 {
		return ""
	}
	return strings.ToUpper(string(rs[0])) + strings.ToLower(string(rs[1:]))
}

func deriveTagColor(tag string) [3]int64 {
	digest := md5.Sum([]byte(strings.ToUpper(tag)))
	var out [3]int64
	for i := 0; i < 3; i++ {
		out[i] = 40 + int64(digest[i])*176/255
	}
	return out
}

var translitTable = map[rune]rune{
	'à': 'a', 'á': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'å': 'a',
	'è': 'e', 'é': 'e', 'ê': 'e', 'ë': 'e',
	'ì': 'i', 'í': 'i', 'î': 'i', 'ï': 'i',
	'ò': 'o', 'ó': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ù': 'u', 'ú': 'u', 'û': 'u', 'ü': 'u',
	'ý': 'y', 'ÿ': 'y', 'ñ': 'n', 'ç': 'c',
	'À': 'A', 'Á': 'A', 'Â': 'A', 'Ã': 'A', 'Ä': 'A', 'Å': 'A',
	'È': 'E', 'É': 'E', 'Ê': 'E', 'Ë': 'E',
	'Ì': 'I', 'Í': 'I', 'Î': 'I', 'Ï': 'I',
	'Ò': 'O', 'Ó': 'O', 'Ô': 'O', 'Õ': 'O', 'Ö': 'O',
	'Ù': 'U', 'Ú': 'U', 'Û': 'U', 'Ü': 'U',
	'Ý': 'Y', 'Ñ': 'N', 'Ç': 'C',
}

func nfkdStrip(s string) string {
	var b strings.Builder
	for _, r := range s {
		if t, ok := translitTable[r]; ok {
			b.WriteRune(t)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var (
	slugRemoveRe   = regexp.MustCompile(`[^\p{L}\p{N}_\s\x0b-]`)
	slugSpaceRe    = regexp.MustCompile(`[\s\x0b-]+`)
	slugCollapseRe = regexp.MustCompile(`_+`)
)

func slugify(text string) string {
	t := pyStrip(strings.ToLower(text))
	t = nfkdStrip(t)
	t = slugRemoveRe.ReplaceAllString(t, "")
	t = slugSpaceRe.ReplaceAllString(t, "_")
	t = slugCollapseRe.ReplaceAllString(t, "_")
	t = strings.Trim(t, "_")
	if t == "" {
		return "unnamed"
	}
	return t
}

func bookmarkPictureSpriteName(bookmarkID string) string {
	return "GFX_hagane_select_date_" + slugify(bookmarkID)
}
