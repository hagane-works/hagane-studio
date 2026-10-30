package exportcheck

import (
	"reflect"
	"testing"
)

func TestNormalizeBaseDependenciesPreservesOrderAndDuplicates(t *testing.T) {
	got := NormalizeBaseDependencies([]any{nil, "  The New Order  ", "", 7, "Alias", "Alias", "\tUnknown\n"})
	want := []string{"The New Order", "Alias", "Alias", "Unknown"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
}

func TestResolveBaseInstallsUsesNewestFirstExactContract(t *testing.T) {
	newestURL := "https://steam.example/new"
	templates := []BasemodTemplate{
		{ID: 30, ModKey: "new", Version: "3", DisplayName: "Different", WorkshopURL: &newestURL, DescriptorNames: []string{"Alias"}},
		{ID: 20, ModKey: "tno", Version: "2", DisplayName: " The New Order ", DescriptorNames: []string{"Other"}},
		{ID: 10, ModKey: "old", Version: "1", DisplayName: "Old", DescriptorNames: []string{"Alias"}},
		{ID: 5, ModKey: "blank", Version: "4", DisplayName: "", DescriptorNames: []string{"Blank Name"}},
	}
	got := ResolveBaseInstalls([]string{"The New Order", "Alias", "Alias", "Blank Name", "Unknown"}, templates)
	if len(got) != 5 || got[0].ModKey != "tno" || got[0].DisplayName != " The New Order " {
		t.Fatalf("display match=%+v", got)
	}
	if got[1].ModKey != "new" || got[1].WorkshopURL != newestURL || got[2].ModKey != "new" {
		t.Fatalf("first match / duplicate=%+v", got)
	}
	if !got[3].Matched || got[3].DisplayName != "Blank Name" {
		t.Fatalf("empty display fallback=%+v", got[3])
	}
	if got[4].Matched || got[4].DisplayName != "Unknown" || got[4].Version != "" || got[4].WorkshopURL != "" || got[4].ModKey != "" {
		t.Fatalf("unmatched=%+v", got[4])
	}
	if empty := ResolveBaseInstalls(nil, templates); empty == nil || len(empty) != 0 {
		t.Fatalf("empty must be []: %#v", empty)
	}
}
