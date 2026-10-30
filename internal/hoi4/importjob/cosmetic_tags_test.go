package importjob

import (
	"reflect"
	"testing"

	"haganeworks/internal/hoi4/cosmetic"
)

func TestCosmeticColorParserIndependentChannels(t *testing.T) {
	rows, err := ParseCosmeticColors("\ufeff# comment {\n  Rom_greater.Kingdom-1 = { note = \"} # {\" color = rgb { 12 34 56 } color_ui = hsv { 0.5 1.0 1.0 } }\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Tag != "Rom_greater.Kingdom-1" || *rows[0].Color != (cosmetic.RGB{12, 34, 56}) || *rows[0].ColorUI != (cosmetic.RGB{0, 255, 255}) {
		t.Fatalf("parsed=%+v", rows)
	}
}
func TestCosmeticColorParserFormats(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  cosmetic.RGB
	}{
		{"rgb { 0.5 1.0 0.0 }", cosmetic.RGB{128, 255, 0}},
		{"{ 0 1 0 }", cosmetic.RGB{0, 1, 0}},
		{"rgb { 12.5 200.0 0.0 }", cosmetic.RGB{13, 200, 0}},
		{"hsv { 0 1 1 }", cosmetic.RGB{255, 0, 0}},
		{"hsv { 1 1 1 }", cosmetic.RGB{255, 0, 0}},
		{"hsv { 0.5 0 0.5 }", cosmetic.RGB{128, 128, 128}},
	} {
		rows, err := ParseCosmeticColors("Long_tag = { color = " + tc.value + " }")
		if err != nil {
			t.Fatal(err)
		}
		if *rows[0].Color != tc.want {
			t.Errorf("%s=%v want %v", tc.value, *rows[0].Color, tc.want)
		}
	}
}
func TestCosmeticColorParserInvalidAndDuplicate(t *testing.T) {
	for _, body := range []string{"color = { 1 2 }", "color = { 1 2 256 }", "color = hsv { 2 0 1 }", "color = { @x 2 3 }", "color = { 1 2 3 } color = { 4 5 6 }"} {
		if _, err := ParseCosmeticColors("Long_tag = { " + body + " }"); err == nil {
			t.Errorf("invalid accepted: %s", body)
		}
	}
	rows, err := ParseCosmeticColors("Long_tag = {color = { 1 2 3 } color = { 1 2 3 }}\nLong_tag = {color = { 1 2 3 }}")
	if err != nil || len(rows) != 1 || !reflect.DeepEqual(rows[0].Color, &cosmetic.RGB{1, 2, 3}) {
		t.Fatalf("identical duplicates=%v,%v", rows, err)
	}
	if _, err := ParseCosmeticColors("Long_tag = {color = { 1 2 3 }} Long_tag = {color = { 4 5 6 }}"); err == nil {
		t.Fatal("ambiguous duplicate has no diagnostic")
	}
}
