package emitmisc

import (
	"reflect"
	"testing"
)

func TestInitialShowPositionLegacyFallback(t *testing.T) {
	got := appendInitialShowPosition(nil, NewOD(), "GER_opening")
	want := []string{
		"", "\tinitial_show_position = {", "\t\tfocus = GER_opening", "\t}",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v want=%#v", got, want)
	}
	if got := appendInitialShowPosition(nil, NewOD(), "+23"); len(got) != 0 {
		t.Fatalf("numeric legacy value must remain ambiguous: %#v", got)
	}
}

func TestInitialShowPositionPreservesTriggerIndentation(t *testing.T) {
	offset := NewOD().Set("x", int64(4)).Set("y", int64(5)).Set(
		"trigger", "OR = {\n\ttag = GER\n\ttag = FRA\n}\n")
	position := NewOD().Set("kind", "coordinates").Set("x", int64(12)).Set(
		"y", int64(-3)).Set("offsets", []any{offset})
	got := appendInitialShowPosition(nil, position, "")
	want := []string{
		"", "\tinitial_show_position = {", "\t\tx = 12", "\t\ty = -3",
		"\t\toffset = {", "\t\t\tx = 4", "\t\t\ty = 5",
		"\t\t\ttrigger = {", "\t\t\t\tOR = {", "\t\t\t\t\ttag = GER",
		"\t\t\t\t\ttag = FRA", "\t\t\t\t}", "\t\t\t}", "\t\t}", "\t}",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v want=%#v", got, want)
	}
}
