package importjob

import (
	"testing"

	"haganeworks/internal/hoi4/emit/locout"
)

// Imported text must be stored under the code the exporter reads, or a
// Korean (Russian, French, ...) mod exports English in its own language.
func TestLocShortMatchesTheExportLanguageCodes(t *testing.T) {
	for code, folder := range locout.LangCodeToHOI4 {
		if got := locShort("l_" + folder); got != code {
			t.Errorf("locShort(%q) = %q, want %q", "l_"+folder, got, code)
		}
	}
	if got := locShort("l_klingon"); got != "klingon" {
		t.Errorf("a language the exporter does not know keeps its name: %q", got)
	}
}
