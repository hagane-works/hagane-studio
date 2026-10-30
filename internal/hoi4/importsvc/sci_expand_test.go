package importsvc

import "testing"

func TestExpandScientificMatchesPython(t *testing.T) {
	cases := map[string]string{
		"5e-06":      "0.000005",
		"1e-05":      "0.00001",
		"-1.234e-07": "-0.0000001234",
		"1e+16":      "10000000000000000.0",
		"1.5e+20":    "150000000000000000000.0",
		"1e-20":      "0.00000000000000000001",
	}
	for in, want := range cases {
		if got := expandScientific(in); got != want {
			t.Errorf("expandScientific(%q) = %q, want %q", in, got, want)
		}
	}
}
