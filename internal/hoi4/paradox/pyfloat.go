package paradox

import (
	"strconv"
	"strings"
)

func pyFloatRepr(f float64) string {
	sci := strconv.FormatFloat(f, 'e', -1, 64)
	neg := strings.HasPrefix(sci, "-")
	if neg {
		sci = sci[1:]
	}
	ePos := strings.IndexByte(sci, 'e')
	mant := sci[:ePos]
	exp, _ := strconv.Atoi(sci[ePos+1:])
	digits := strings.Replace(mant, ".", "", 1)

	var out string
	switch {
	case exp < -4 || exp >= 16:

		out = mant + "e" + pySciExp(exp)
	case exp >= len(digits)-1:
		out = digits + strings.Repeat("0", exp-(len(digits)-1)) + ".0"
	case exp >= 0:
		out = digits[:exp+1] + "." + digits[exp+1:]
	default:
		out = "0." + strings.Repeat("0", -exp-1) + digits
	}
	if neg {
		out = "-" + out
	}
	return out
}

func pySciExp(exp int) string {
	sign := "+"
	if exp < 0 {
		sign = "-"
		exp = -exp
	}
	s := strconv.Itoa(exp)
	if len(s) < 2 {
		s = "0" + s
	}
	return sign + s
}
