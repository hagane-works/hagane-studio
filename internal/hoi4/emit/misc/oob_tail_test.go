package emitmisc

import (
	"strings"
	"testing"
)

func TestTakenInUnitsTailLosesItsByteOrderMark(t *testing.T) {
	env := testEnv("GER")
	template := NewNS().
		Set("template_name", "Infanterie-Division").
		Set("oob_file", "GER_1936_nsb").
		Set("source_file", "history/units/GER_1936_nsb.txt").
		Set("country_tag", "GER").
		Set("source", "created").
		Set("ordering", int64(0)).
		Set("regiments", []any{}).
		Set("support", []any{}).
		Set("raw_extras", NewOD()).
		Set("oob_units_tail", "\ufeff\n\nunits = {\n\tdivision = {\n\t\tname = \"1. Infanterie-Division\"\n\t\tlocation = 6488\n\t\tdivision_template = \"Infanterie-Division\"\n\t}\n}\n")

	EmitTreesTechMilitary(env, nil, nil, nil, nil, []*NS{template}, nil, nil, nil, nil)

	got := writeAt(t, env, "mod/history/units/GER_1936_nsb.txt")
	if strings.Contains(got, "\ufeff") {
		t.Fatalf("the byte-order mark was written into the file:\n%q", got)
	}
	if !strings.Contains(got, "name = \"1. Infanterie-Division\"") {
		t.Fatalf("the taken-in divisions are missing:\n%s", got)
	}
}
