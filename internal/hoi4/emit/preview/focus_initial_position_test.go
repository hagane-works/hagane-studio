package preview

import (
	"math/big"
	"reflect"
	"testing"
)

func TestPreviewInitialShowPositionLegacyFallback(t *testing.T) {
	got := appendInitialShowPosition(nil, NewOD(), "GER_opening")
	want := []string{
		"", "\tinitial_show_position = {", "\t\tfocus = GER_opening", "\t}",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v want=%#v", got, want)
	}
	if got := appendInitialShowPosition(nil, NewOD(), "-23"); len(got) != 0 {
		t.Fatalf("numeric legacy value must remain ambiguous: %#v", got)
	}
}

func TestPreviewInitialShowPositionPreservesTriggerIndentation(t *testing.T) {
	offset := NewOD()
	offset.Set("x", big.NewInt(4))
	offset.Set("y", big.NewInt(5))
	offset.Set("trigger", "OR = {\n\ttag = GER\n\ttag = FRA\n}\n")
	position := NewOD()
	position.Set("kind", "coordinates")
	position.Set("x", big.NewInt(12))
	position.Set("y", big.NewInt(-3))
	position.Set("offsets", []any{offset})
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
