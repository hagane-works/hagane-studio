package techsurfaceprofiles

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func auditedEnvironment(state DLCState) Environment {
	return NewEnvironment(AuditedSourceFingerprint(), state)
}

func TestProductionEntriesAreFourStableEntrances(t *testing.T) {
	want := []string{EntryArmy, EntryAir, EntryNavy, EntryIndustryEquipment}
	for run := 0; run < 3; run++ {
		entries := ProductionEntries()
		if len(entries) != len(want) {
			t.Fatalf("entry count=%d, want %d", len(entries), len(want))
		}
		got := make([]string, len(entries))
		for i := range entries {
			got[i] = entries[i].ID
		}
		if !slices.Equal(got, want) {
			t.Fatalf("entry order=%v, want %v", got, want)
		}
	}
}

func TestResolveBaseAndDLCVariantsDeterministically(t *testing.T) {
	tests := []struct {
		name  string
		state DLCState
		want  map[string]string
	}{
		{
			name: "base", state: DLCState{},
			want: map[string]string{
				EntryArmy + "/infantry":                 "infantry_folder/base",
				EntryArmy + "/armour":                   "armour_folder/base",
				EntryAir + "/airframes":                 "air_techs_folder/base",
				EntryNavy + "/naval_main":               "naval_folder/base",
				EntryIndustryEquipment + "/industry":    "industry_folder/base",
				EntryIndustryEquipment + "/electronics": "electronics_folder/base",
			},
		},
		{
			name: "all relevant DLC", state: DLCState{NoStepBack: true, ByBloodAlone: true, ManTheGuns: true},
			want: map[string]string{
				EntryArmy + "/infantry":                 "infantry_folder/base",
				EntryArmy + "/armour":                   "nsb_armour_folder/nsb",
				EntryAir + "/airframes":                 "bba_air_techs_folder/bba",
				EntryNavy + "/naval_main":               "mtgnavalfolder/mtg",
				EntryNavy + "/naval_support":            "mtgnavalsupportfolder/mtg",
				EntryIndustryEquipment + "/industry":    "industry_folder/base",
				EntryIndustryEquipment + "/electronics": "electronics_folder/base",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first, err := Resolve(auditedEnvironment(test.state))
			if err != nil {
				t.Fatal(err)
			}
			second, err := Resolve(auditedEnvironment(test.state))
			if err != nil {
				t.Fatal(err)
			}
			firstJSON, _ := json.Marshal(first)
			secondJSON, _ := json.Marshal(second)
			if string(firstJSON) != string(secondJSON) {
				t.Fatalf("selection is not deterministic:\n%s\n%s", firstJSON, secondJSON)
			}
			got := map[string]string{}
			for _, entry := range first.Entries {
				for _, target := range entry.Targets {
					got[entry.ID+"/"+target.ID] = target.FolderID + "/" + target.VariantID
				}
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("selected targets=%v, want %v", got, test.want)
			}
		})
	}
}

func TestAuthoringTargetFailsClosedForUncertifiedNSBAndAllowsCertifiedDLC(t *testing.T) {
	profile, err := Resolve(auditedEnvironment(DLCState{NoStepBack: true, ByBloodAlone: true, ManTheGuns: true}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profile.AuthoringTarget(EntryArmy, "armour"); ErrorCodeOf(err) != CodeTargetUncertified {
		t.Fatalf("NSB armour error=%v code=%q, want %q", err, ErrorCodeOf(err), CodeTargetUncertified)
	}
	for _, item := range [][2]string{{EntryArmy, "infantry"}, {EntryAir, "airframes"}, {EntryNavy, "naval_main"}, {EntryNavy, "naval_support"}, {EntryIndustryEquipment, "industry"}, {EntryIndustryEquipment, "electronics"}} {
		target, err := profile.AuthoringTarget(item[0], item[1])
		if err != nil {
			t.Fatalf("%s/%s: %v", item[0], item[1], err)
		}
		if len(target.Slots) == 0 || len(target.Timeline) == 0 {
			t.Fatalf("%s/%s returned without fixed geometry", item[0], item[1])
		}
	}
}

func TestResolveRejectsMissingAndDriftedSource(t *testing.T) {
	missing := auditedEnvironment(DLCState{})
	missing.Source.Files = missing.Source.Files[:1]
	if _, err := Resolve(missing); ErrorCodeOf(err) != CodeSourceMissing {
		t.Fatalf("missing source error=%v code=%q", err, ErrorCodeOf(err))
	}

	drifted := auditedEnvironment(DLCState{})
	drifted.Source.Files[0].SHA256 = strings.Repeat("0", 64)
	if _, err := Resolve(drifted); ErrorCodeOf(err) != CodeSourceDrift {
		t.Fatalf("drifted source error=%v code=%q", err, ErrorCodeOf(err))
	}
}

func TestResolveRejectsMissingOrStaleDLCFingerprint(t *testing.T) {
	missing := auditedEnvironment(DLCState{})
	missing.DLCFingerprint = ""
	if _, err := Resolve(missing); ErrorCodeOf(err) != CodeDLCFingerprint {
		t.Fatalf("missing fingerprint error=%v code=%q", err, ErrorCodeOf(err))
	}

	stale := auditedEnvironment(DLCState{})
	stale.DLC.ByBloodAlone = true
	if _, err := Resolve(stale); ErrorCodeOf(err) != CodeDLCFingerprint {
		t.Fatalf("stale fingerprint error=%v code=%q", err, ErrorCodeOf(err))
	}
}

func TestCatalogueInputsExposeNoFreeCoordinates(t *testing.T) {
	typeOf := reflect.TypeOf(Environment{})
	for i := 0; i < typeOf.NumField(); i++ {
		name := strings.ToLower(typeOf.Field(i).Name)
		if name == "x" || name == "y" || strings.Contains(name, "coordinate") || strings.Contains(name, "position") || strings.Contains(name, "slot") || strings.Contains(name, "folder") {
			t.Fatalf("Environment unexpectedly exposes author geometry field %q", typeOf.Field(i).Name)
		}
	}

	profile, err := Resolve(auditedEnvironment(DLCState{}))
	if err != nil {
		t.Fatal(err)
	}
	target, err := profile.AuthoringTarget(EntryArmy, "infantry")
	if err != nil {
		t.Fatal(err)
	}
	name, err := target.GridboxName("MOD_infantry_extension")
	if err != nil || name != "MOD_infantry_extension_tree" {
		t.Fatalf("gridbox name=%q err=%v", name, err)
	}
	if _, err := target.GridboxName(`bad id { x = 1 }`); err == nil {
		t.Fatal("unsafe author-controlled gridbox id accepted")
	}
}

func TestCatalogueReturnsDefensiveCopies(t *testing.T) {
	entries := ProductionEntries()
	entries[0].ID = "mutated"
	entries[0].TargetGroups[0].Variants[0].Slots[0].X = -999
	clean := ProductionEntries()
	if clean[0].ID != EntryArmy || clean[0].TargetGroups[0].Variants[0].Slots[0].X != 140 {
		t.Fatalf("catalogue was mutated through returned copy: %+v", clean[0])
	}

	profile, err := Resolve(auditedEnvironment(DLCState{}))
	if err != nil {
		t.Fatal(err)
	}
	target, err := profile.AuthoringTarget(EntryArmy, "infantry")
	if err != nil {
		t.Fatal(err)
	}
	target.Slots[0].X = -999
	again, err := profile.AuthoringTarget(EntryArmy, "infantry")
	if err != nil || again.Slots[0].X != 140 {
		t.Fatalf("profile target was mutated through returned copy: %+v err=%v", again, err)
	}
}

func TestBaseNavalProfileKeepsEngineVerifiedGridboxLocalCells(t *testing.T) {
	target := baseNavalTarget()
	if got, want := target.Timeline, []TimelineCell{{StartYear: 1936, X: 0, Y: 7}, {StartYear: 1940, X: 0, Y: 13}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Base naval engine-verified timeline=%v, want %v", got, want)
	}
	if target.Certification.State != "engine_verified_geometry" {
		t.Fatalf("Base naval certification=%q", target.Certification.State)
	}
}

func TestAllPlayerTargetsReturnBaseAndDLCVariants(t *testing.T) {
	air, err := AllPlayerTargets(EntryAir, "airframes")
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{air[0].FolderID, air[1].FolderID}; !reflect.DeepEqual(got, []string{"air_techs_folder", "bba_air_techs_folder"}) {
		t.Fatalf("air compatibility folders=%v", got)
	}
	if _, err := AllPlayerTargets(EntryNavy, "naval_support"); ErrorCodeOf(err) != CodeTargetUncertified {
		t.Fatalf("naval support all-player error=%v", err)
	}
}
