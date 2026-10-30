package paradox

import (
	"strconv"
	"testing"
)

func TestPyFloatReprMatchesCPython(t *testing.T) {
	cases := []struct{ in, want string }{
		{"0.75", "0.75"},
		{"0.25", "0.25"},
		{"0.1", "0.1"},
		{"1.0", "1.0"},
		{"100.0", "100.0"},
		{"1937.0", "1937.0"},
		{"-0.5", "-0.5"},
		{"-0.0", "-0.0"},
		{"0.0", "0.0"},
		{"0.0001", "0.0001"},
		{"0.00001", "1e-05"},
		{"1000000000000000.0", "1000000000000000.0"},
		{"10000000000000000.0", "1e+16"},
		{"123456789.123456789", "123456789.12345679"},
		{"3.14159265358979", "3.14159265358979"},
		{"2.675", "2.675"},
	}
	for _, c := range cases {
		f, err := strconv.ParseFloat(c.in, 64)
		if err != nil {
			t.Fatalf("bad test input %q: %v", c.in, err)
		}
		if got := pyFloatRepr(f); got != c.want {
			t.Errorf("pyFloatRepr(%s) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFloatScalarRendersWithPythonRepr(t *testing.T) {

	ast, _ := mustParse(t, "a = 0.75\nb = 1.0\nc = -0.5\nd = 0.00001\n")
	want := []string{"0.75", "1.0", "-0.5", "1e-05"}
	for i, w := range want {
		e := ast.Entries[i]
		if e.Scalar.Kind != ScalarFloat {
			t.Fatalf("entry %d: kind = %d, want float", i, e.Scalar.Kind)
		}
		if got := e.Scalar.StrValue(); got != w {
			t.Errorf("entry %d: StrValue = %q, want %q", i, got, w)
		}
	}
}

func TestIntScalarRendersWithoutDecimalPoint(t *testing.T) {
	ast, _ := mustParse(t, "a = 1\nb = -3\nc = 0\n")
	want := []string{"1", "-3", "0"}
	for i, w := range want {
		if got := ast.Entries[i].Scalar.StrValue(); got != w {
			t.Errorf("entry %d: %q, want %q", i, got, w)
		}
	}
}

func TestJSONFormCarriesNumbersAsStrings(t *testing.T) {
	ast, _ := mustParse(t, "i = 1936\nf = 0.75\ns = yes\nb = { x = 1 }\n")
	js := EntriesJSON(ast.Entries)
	if len(js) != 4 {
		t.Fatalf("got %d entries", len(js))
	}
	if js[0].Value.T != "int" || js[0].Value.V != "1936" {
		t.Errorf("int JSON = %+v, want t=int v=\"1936\"", js[0].Value)
	}
	if js[1].Value.T != "float" || js[1].Value.V != "0.75" {
		t.Errorf("float JSON = %+v, want t=float v=\"0.75\"", js[1].Value)
	}
	if js[2].Value.T != "str" || js[2].Value.V != "yes" {
		t.Errorf("str JSON = %+v", js[2].Value)
	}
	if js[3].Value.T != "block" || js[3].Value.Trivia == nil {
		t.Errorf("block JSON must carry its own trivia: %+v", js[3].Value)
	}
}

func TestJSONFormNeverEmitsNullSlices(t *testing.T) {

	ast, _ := mustParse(t, "a = 1\n")
	js := EntriesJSON(ast.Entries)
	if js[0].Trivia.Leading == nil || js[0].Trivia.Trailing == nil {
		t.Fatalf("trivia slices must be non-nil: %+v", js[0].Trivia)
	}
	if RecoveriesJSON(nil) == nil {
		t.Fatal("RecoveriesJSON(nil) must be an empty slice, not nil")
	}
	if EntriesJSON(nil) == nil {
		t.Fatal("EntriesJSON(nil) must be an empty slice, not nil")
	}
}
