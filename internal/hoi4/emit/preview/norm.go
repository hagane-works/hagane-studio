package preview

import "strings"

const (
	hangulSBase = 0xAC00
	hangulLBase = 0x1100
	hangulVBase = 0x1161
	hangulTBase = 0x11A7
	hangulLCnt  = 19
	hangulVCnt  = 21
	hangulTCnt  = 28
	hangulNCnt  = hangulVCnt * hangulTCnt
	hangulSCnt  = hangulLCnt * hangulNCnt
)

func combiningClass(r rune) uint8 { return cccMap[r] }

func nfkdRunes(out []rune, r rune) []rune {
	if r >= hangulSBase && r < hangulSBase+hangulSCnt {
		sIdx := r - hangulSBase
		l := hangulLBase + sIdx/hangulNCnt
		v := hangulVBase + (sIdx%hangulNCnt)/hangulTCnt
		t := hangulTBase + sIdx%hangulTCnt
		out = append(out, l, v)
		if t != hangulTBase {
			out = append(out, t)
		}
		return out
	}
	if d, ok := nfkdOne[r]; ok {

		for _, dr := range d {
			out = append(out, dr)
		}
		return out
	}
	return append(out, r)
}

func canonicalReorder(rs []rune) {
	for i := 1; i < len(rs); i++ {
		cc := combiningClass(rs[i])
		if cc == 0 {
			continue
		}
		j := i
		for j > 0 && combiningClass(rs[j-1]) > cc {
			rs[j-1], rs[j] = rs[j], rs[j-1]
			j--
		}
	}
}

func pyNFKD(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		out = nfkdRunes(out, r)
	}
	canonicalReorder(out)
	return string(out)
}

func composePair(a, b rune) (rune, bool) {

	if a >= hangulLBase && a < hangulLBase+hangulLCnt &&
		b >= hangulVBase && b < hangulVBase+hangulVCnt {
		return hangulSBase + (a-hangulLBase)*hangulNCnt + (b-hangulVBase)*hangulTCnt, true
	}
	if a >= hangulSBase && a < hangulSBase+hangulSCnt && (a-hangulSBase)%hangulTCnt == 0 &&
		b > hangulTBase && b < hangulTBase+hangulTCnt {
		return a + (b - hangulTBase), true
	}
	c, ok := compPair[uint64(a)<<32|uint64(b)]
	return c, ok
}

func pyNFKC(s string) string {
	rs := []rune(pyNFKD(s))
	if len(rs) == 0 {
		return ""
	}
	out := make([]rune, 0, len(rs))
	lastStarter := -1
	lastStarterCC := uint8(255)
	for _, r := range rs {
		cc := combiningClass(r)
		if lastStarter >= 0 {

			prevCC := uint8(0)
			if len(out)-1 > lastStarter {
				prevCC = combiningClass(out[len(out)-1])
			}
			blocked := len(out)-1 > lastStarter && (prevCC == 0 || prevCC >= cc)
			if !blocked {
				if comp, ok := composePair(out[lastStarter], r); ok {
					out[lastStarter] = comp
					lastStarterCC = 0
					_ = lastStarterCC
					continue
				}
			}
		}
		out = append(out, r)
		if cc == 0 {
			lastStarter = len(out) - 1
		}
	}
	return string(out)
}

func stripCombining(s string) string {
	var b strings.Builder
	for _, r := range s {
		if combiningClass(r) == 0 {
			b.WriteRune(r)
		}
	}
	return b.String()
}
