package guirender

import (
	"fmt"
	"strings"
	"testing"
)

func labelsOf(items []map[string]any) string {
	out := []string{}
	for _, item := range items {
		out = append(out, fmt.Sprint(item["label"]))
	}
	return strings.Join(out, ", ")
}

func TestRealGameDoctrinesMatchTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	const date = "1936.1.1.12"
	land := idx.DoctrinesPayload("GER", date, "")
	if got := labelsOf(land["folders"].([]map[string]any)); got != "Land Doctrine, Naval Doctrine, Air Doctrine, Special Forces" {
		t.Errorf("tabs = %s", got)
	}
	for _, want := range []struct{ folder, grand, tracks string }{
		{"land", "Mobile Warfare", "Infantry, Artillery & Combat Support, Armor, Operations"},
		{"naval", "Trade Interdiction", "Submarines, Screens, Carriers, Capital Ships"},
		{"air", "Battlefield Support", "Fighter Aircraft, Strike Aircraft, Medium Aircraft, Heavy Aircraft"},
	} {
		page := idx.DoctrinesPayload("GER", date, want.folder)
		grand, _ := page["grand"].(map[string]any)
		if grand == nil || grand["label"] != want.grand {
			t.Errorf("%s grand doctrine = %v, want %s", want.folder, grand, want.grand)
			continue
		}
		tracks := page["tracks"].([]map[string]any)
		if got := labelsOf(tracks); got != want.tracks {
			t.Errorf("%s tracks = %s, want %s", want.folder, got, want.tracks)
		}
		for _, track := range tracks {
			if track["sub"] != nil || track["mastery"] != 0 {
				t.Errorf("%s/%v starts with %v, mastery %v", want.folder, track["token"], track["sub"], track["mastery"])
			}
		}
		if want.folder == "air" {
			if got := labelsOf(tracks[0]["choices"].([]map[string]any)); got != "Homeland Air Defense, Medium-Range Escort, Fighter-Bombers, Airspace Dominance, Knights of the Air, Naval Aviation" {
				t.Errorf("fighter subdoctrines = %s", got)
			}
		}
	}
	sf := idx.DoctrinesPayload("GER", date, "special_forces")
	if sf["grand"] != nil {
		t.Errorf("special forces start with %v, want none", sf["grand"])
	}

	grands := sf["grand_choices"].([]map[string]any)
	if got := labelsOf(grands); got != "Expanded Special Forces, Elite Special Forces" {
		t.Errorf("special forces grand doctrines = %s", got)
	}
	if grands[0]["xp_cost"] != 60.0 || grands[0]["xp_type"] != "army" {
		t.Errorf("Expanded Special Forces costs %v %v", grands[0]["xp_cost"], grands[0]["xp_type"])
	}
	if xp := sf["experience"].(map[string]any); xp["army"] != 0.0 {
		t.Errorf("Germany's army experience on the first day = %v", xp["army"])
	}

	got := []string{}
	land39 := idx.DoctrinesPayload("GER", "1939.8.14.12", "land")["tracks"].([]map[string]any)
	for _, track := range land39 {
		sub, _ := track["sub"].(map[string]any)
		got = append(got, fmt.Sprintf("%v=%v/%v", track["token"], sub["token"], track["mastery"]))
	}
	if strings.Join(got, " ") != "infantry=mobile_infantry/200 combat_support=mobile_recon_and_assault/150 armor=<nil>/0 operations=mission_type_tactics/100" {
		t.Errorf("1939 land = %v", got)
	}

	steps := []string{}
	for _, reward := range land39[0]["rewards"].([]map[string]any) {
		steps = append(steps, fmt.Sprintf("%v@%v:%v/%.2f", reward["level"], reward["mastery"], reward["unlocked"], reward["progress"]))
	}
	if strings.Join(steps, " ") != "1@100:true/1.00 2@200:true/1.00 3@300:false/0.00 4@400:false/0.00 5@500:false/0.00" {
		t.Errorf("mobile infantry rewards = %v", steps)
	}
	if first := land39[0]["rewards"].([]map[string]any)[0]; first["label"] != "Modernized Cavalry" {
		t.Errorf("first reward = %v", first["label"])
	}
	if land39[0]["complete"] != false || land39[2]["rewards"] != nil {
		t.Errorf("complete = %v, armor rewards = %v", land39[0]["complete"], land39[2]["rewards"])
	}

	if second := land39[1]["rewards"].([]map[string]any)[1]; second["progress"] != 0.5 || second["unlocked"] != false {
		t.Errorf("recon's second reward = %v", second)
	}
}

func TestDoctrineRewardsUnlockInTurn(t *testing.T) {
	idx := &Index{}
	sub := subDoctrine{Token: "sf", Rewards: []doctrineReward{{Key: "a", Mastery: 60}, {Key: "b"}, {}}}
	steps, complete := idx.doctrineRewards(sub, 260, func(key string) string { return "loc:" + key })
	got := []string{}
	for _, step := range steps {
		got = append(got, fmt.Sprintf("%v@%v:%v/%.2f", step["label"], step["mastery"], step["unlocked"], step["progress"]))
	}
	if strings.Join(got, " ") != "loc:sf_a@60:true/1.00 loc:sf_b@160:true/1.00 @260:true/1.00" || !complete {
		t.Errorf("steps = %v, complete = %v", got, complete)
	}
	if _, complete := idx.doctrineRewards(subDoctrine{}, 500, nil); complete {
		t.Error("a subdoctrine without rewards has no milestone to light")
	}
}
