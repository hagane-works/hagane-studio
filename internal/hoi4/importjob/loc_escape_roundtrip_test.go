package importjob

import (
	"testing"

	"haganeworks/internal/hoi4/emit/locout"
)

// What the importer stores must come back out of the exporter as written.
func TestLocValueEscapesRoundTrip(t *testing.T) {
	for _, written := range []string{
		`plain`,
		`line\nbreak`,
		`a \"quoted\" word`,
		`tab\there`,
		`shown as backslash-n: \n`,
		`two backslashes \\ and a break\n`,
	} {
		if got := locout.YmlEscapeValue(decodeLocValue(written)); got != written {
			t.Errorf("%s came back as %s", written, got)
		}
	}
}
