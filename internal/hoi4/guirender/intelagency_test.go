package guirender

import (
	"strings"
	"testing"
)

func TestRealGameIntelligenceAgencyBeforeCreation(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.IntelAgencyPayload("GER", "1936.1.1.12")
	creation := out["creation"].(map[string]any)
	if creation["factories"] != 5.0 || creation["days"] != 30.0 || creation["days_text"] != "§Y30§! days" {
		t.Errorf("creation = %v", creation)
	}
	insignia := out["insignia"].([]map[string]any)
	pictures := []string{}
	for _, item := range insignia {
		pictures = append(pictures, item["picture"].(string))
	}
	want := []string{"GFX_intelligence_agency_logo_ger"}
	for n := 1; n <= 8; n++ {
		want = append(want, "GFX_intelligence_agency_logo_generic_"+string(rune('0'+n)))
	}
	if len(pictures) != len(want) {
		t.Fatalf("insignia = %v", pictures)
	}
	for i := range want {
		if pictures[i] != want[i] {
			t.Fatalf("insignia = %v", pictures)
		}
	}
	if insignia[0]["available_script"] != "original_tag = GER\nNOT = {\n\thas_government = communism\n}" || insignia[1]["available_script"] != "always = yes" {
		t.Errorf("conditions = %q / %q", insignia[0]["available_script"], insignia[1]["available_script"])
	}
	if out["chosen"] != 0 || out["name"] != "Abwehr" {
		t.Errorf("chosen %v name %v", out["chosen"], out["name"])
	}
	labels := []string{}
	for _, branch := range out["branches"].([]map[string]any) {
		labels = append(labels, branch["label"].(string))
	}
	if len(labels) != 5 || labels[0] != "Intelligence" || labels[1] != "Defense" || labels[2] != "Operations" ||
		labels[3] != "Operative Training" || labels[4] != "Cryptology Department" {
		t.Errorf("branches = %v", labels)
	}

	if name := idx.IntelAgencyPayload("ENG", "1936.1.1.12")["name"]; name != "MI6" {
		t.Errorf("ENG name = %v", name)
	}
}

func TestRealGameIntelligenceAgencyCreatedByHistory(t *testing.T) {
	idx := realTopbarIndex(t, false)
	for _, want := range []struct {
		tag, date string
		created   bool
	}{
		{"GER", "1939.8.14.12", true},
		{"HUN", "1939.8.14.12", true},
		{"GER", "1936.1.1.12", false},
		{"ENG", "1939.8.14.12", false},
	} {
		out := idx.IntelAgencyPayload(want.tag, want.date)
		if out["created"] != want.created {
			t.Errorf("%s %s created = %v", want.tag, want.date, out["created"])
		}
	}
	out := idx.IntelAgencyPayload("GER", "1939.8.14.12")
	texts := out["texts"].(map[string]any)
	if out["operative_slots"] != 1.0 || texts["defense_level"] != "Counter Intelligence: §H0.00§!" || texts["recruit_days"] != "30 days" || out["name"] != "Abwehr" {
		t.Errorf("created panel = %v %v %v %v", out["operative_slots"], texts["defense_level"], texts["recruit_days"], out["name"])
	}
}

func TestRealGameIntelligenceUpgradesAvailable(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.IntelAgencyPayload("GER", "1936.1.1.12")
	locked := []string{}
	for _, branch := range out["branches"].([]map[string]any) {
		for _, upgrade := range branch["upgrades"].([]map[string]any) {
			if upgrade["available"] == false {
				locked = append(locked, upgrade["id"].(string))
			}
		}
	}
	want := "upgrade_portable_radios|upgrade_decryption_boost|upgrade_decryption_boost_2|upgrade_crypto_strength|upgrade_crypto_strength_2"
	if got := strings.Join(locked, "|"); got != want {
		t.Errorf("locked = %s", got)
	}
}
