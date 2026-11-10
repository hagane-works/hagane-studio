package worldout

import "testing"

func TestHistoryCountryFilenameKeepsTheImportedName(t *testing.T) {
	for _, tc := range []struct{ tag, source, want string }{
		{"GRE", "history/countries/GRE - Greece.txt", "GRE - Greece.txt"},
		{"GRE", `history\countries\GRE - Greece.txt`, "GRE - Greece.txt"},
		{"ABK", "history/countries/ABK - Abkhazia.txt", "ABK - Abkhazia.txt"},
		{"GRE", "", "GRE - zmod.txt"},
		{"GRE", "history/countries/ENG - Britain.txt", "GRE - zmod.txt"},
		{"GRE", "history/countries/sub/GRE - Greece.txt", "GRE - zmod.txt"},
		{"GRE", "history/countries/GRE - Greece.yml", "GRE - zmod.txt"},
		{"GRE", "history/states/GRE - Greece.txt", "GRE - zmod.txt"},
		{"GRE", "history/countries/GRE - <bad>.txt", "GRE - zmod.txt"},
	} {
		if got := HistoryCountryFilename(tc.tag, tc.source, "mod"); got != tc.want {
			t.Errorf("HistoryCountryFilename(%q, %q) = %q, want %q", tc.tag, tc.source, got, tc.want)
		}
	}
}

func TestCountryDefinitionFilenameKeepsTheImportedName(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"common/countries/fds_Ruritania.txt", "fds_Ruritania.txt"},
		{`common\countries\fds_Ruritania.txt`, "fds_Ruritania.txt"},
		{"", "RUR.txt"},
		{"common/countries/sub/fds.txt", "RUR.txt"},
		{"common/countries/colors.txt", "RUR.txt"},
		{"common/countries/Cosmetic.txt", "RUR.txt"},
		{"common/countries/fds.yml", "RUR.txt"},
		{"common/ideas/fds.txt", "RUR.txt"},
	} {
		if got := CountryDefinitionFilename("RUR", tc.source); got != tc.want {
			t.Errorf("CountryDefinitionFilename(%q) = %q, want %q", tc.source, got, tc.want)
		}
	}
}
