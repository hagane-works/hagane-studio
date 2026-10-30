package emitmisc

import (
	"os"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/importsvc"
)

func oldOOBRenderToken(token string) string {
	s, err := importsvc.RenderHOI4ScalarToken(token)
	if err != nil {
		panic("OOBUnitsValidationError: " + err.Error())
	}
	return s
}

func airWing(id int64, equipment, owner, amount string) *NS {
	return NewNS().
		Set("id", id).
		Set("equipment_type", equipment).
		Set("owner", owner).
		Set("amount", amount).
		Set("ordering", id)
}

func airWingGroup(id int64, location string, wings ...*NS) *NS {
	list := make([]any, 0, len(wings))
	for _, w := range wings {
		list = append(list, w)
	}
	return NewNS().
		Set("id", id).
		Set("country_tag", "SOV").
		Set("oob_file", "SOV_1936").
		Set("source_file", "history/units/SOV_1936.txt").
		Set("location", location).
		Set("ordering", id).
		Set("wings", list)
}

func TestNegativeControlAirWingGroupWithoutLocation(t *testing.T) {

	var panicked any
	func() {
		defer func() { panicked = recover() }()
		oldOOBRenderToken("")
	}()
	if panicked == nil {
		t.Fatal("negative control is void: the pre-change helper no longer panics on \"\"")
	}
	t.Logf("BEFORE (old helper) panic: %v", panicked)

	groups := []*NS{
		airWingGroup(12, "", airWing(1, "fighter_equipment_1", "SOV", "24"),
			airWing(2, "cas_equipment_1", "SOV", "12")),
		airWingGroup(13, "219", airWing(3, "fighter_equipment_1", "SOV", "48")),
	}

	script, notices := GenerateAirWingsBlockTolerant(groups, 0)
	t.Logf("AFTER  notice: %s", strings.Join(notices, "\n"))
	t.Logf("AFTER  script:\n%s", script)

	if len(notices) != 1 {
		t.Fatalf("want exactly 1 notice, got %d: %v", len(notices), notices)
	}
	n := notices[0]

	for _, want := range []string{
		"air wing group #12",
		"SOV",
		"SOV_1936",
		"spawn state is empty",
		"OOB workbench",
		"2 air wing(s)",
		"will not exist in game",
	} {
		if !strings.Contains(n, want) {
			t.Errorf("notice is missing %q; got: %s", want, n)
		}
	}
	if strings.Contains(n, "scalar token") {
		t.Errorf("notice leaks the parser's wording at the author: %s", n)
	}

	if !strings.Contains(script, "219 = {") ||
		!strings.Contains(script, "fighter_equipment_1 = {") {
		t.Errorf("healthy group was lost:\n%s", script)
	}
	if strings.Contains(script, "cas_equipment_1") {
		t.Errorf("skipped group leaked into the script:\n%s", script)
	}

	if strings.Contains(script, "\t = {") || strings.Contains(script, "\n = {") {
		t.Errorf("emitted a keyless block:\n%s", script)
	}
}

func TestAirWingGroupWithMalformedLocationDegrades(t *testing.T) {
	groups := []*NS{airWingGroup(12, "219 220", airWing(1, "fighter_equipment_1", "SOV", "24"))}
	script, notices := GenerateAirWingsBlockTolerant(groups, 0)
	if script != "" {
		t.Errorf("every group was skipped, so nothing should be emitted; got:\n%s", script)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "\"219 220\" is not a single HOI4 token") {
		t.Fatalf("want a notice quoting the rejected value, got: %v", notices)
	}
	t.Logf("notice: %s", notices[0])
}

func TestAirWingWithoutEquipmentTypeIsSkippedButSiblingsSurvive(t *testing.T) {
	groups := []*NS{airWingGroup(12, "219",
		airWing(1, "", "SOV", "24"),
		airWing(2, "fighter_equipment_1", "SOV", "48"))}
	script, notices := GenerateAirWingsBlockTolerant(groups, 0)
	if len(notices) != 1 {
		t.Fatalf("want 1 notice, got %v", notices)
	}
	t.Logf("notice: %s", notices[0])
	for _, want := range []string{"air wing #1", "air wing group #12", "aircraft type is empty"} {
		if !strings.Contains(notices[0], want) {
			t.Errorf("notice missing %q: %s", want, notices[0])
		}
	}
	if !strings.Contains(script, "fighter_equipment_1 = {") {
		t.Errorf("sibling wing was lost:\n%s", script)
	}
	if !strings.Contains(script, "219 = {") {
		t.Errorf("group was lost even though only one wing was bad:\n%s", script)
	}
}

func TestMalformedScalarDropsOnlyThatKey(t *testing.T) {
	groups := []*NS{airWingGroup(12, "219", airWing(1, "fighter_equipment_1", "SOV BRA", "24"))}
	script, notices := GenerateAirWingsBlockTolerant(groups, 0)
	if len(notices) != 1 {
		t.Fatalf("want 1 notice, got %v", notices)
	}
	t.Logf("notice: %s", notices[0])
	if strings.Contains(script, "owner") {
		t.Errorf("unrenderable owner was emitted anyway:\n%s", script)
	}
	if !strings.Contains(script, "amount = 24") {
		t.Errorf("the good sibling key was dropped too:\n%s", script)
	}
	if !strings.Contains(script, "fighter_equipment_1 = {") {
		t.Errorf("whole wing was dropped for one bad key:\n%s", script)
	}
}

func TestFleetFieldTokenDegradesWithoutLosingTheFleet(t *testing.T) {
	fleet := NewNS().
		Set("id", 3).
		Set("country_tag", "SOV").
		Set("oob_file", "SOV_1936").
		Set("name", "Baltic Fleet").
		Set("naval_base", "88 89").
		Set("ordering", int64(1))
	script, notices := GenerateFleetsBlockTolerant([]*NS{fleet}, 0)
	if len(notices) != 1 {
		t.Fatalf("want 1 notice, got %v", notices)
	}
	t.Logf("notice: %s", notices[0])
	for _, want := range []string{"fleet #3", "naval_base", "home port"} {
		if !strings.Contains(notices[0], want) {
			t.Errorf("notice missing %q: %s", want, notices[0])
		}
	}
	if !strings.Contains(script, `name = "Baltic Fleet"`) {
		t.Errorf("fleet was lost over one bad field:\n%s", script)
	}
	if strings.Contains(script, "naval_base") {
		t.Errorf("unrenderable naval_base was emitted:\n%s", script)
	}
}

func TestHealthyInputBytesUnchanged(t *testing.T) {
	groups := []*NS{airWingGroup(12, "219",
		airWing(1, "fighter_equipment_1", "SOV", "24"))}
	script, notices := GenerateAirWingsBlockTolerant(groups, 0)
	if len(notices) != 0 {
		t.Fatalf("healthy input produced notices: %v", notices)
	}
	want := "air_wings = {\n\t219 = {\n\t\tfighter_equipment_1 = {\n" +
		"\t\t\towner = SOV\n\t\t\tamount = 24\n\t\t}\n\t}\n}"
	if script != want {
		t.Errorf("byte drift on the happy path\n want: %q\n got:  %q", want, script)
	}

	if legacy := GenerateAirWingsBlock(groups, 0); legacy != script {
		t.Errorf("wrapper drifted from the tolerant form\n %q\n %q", legacy, script)
	}
}

func TestCentralSeamConsumesOOBNotices(t *testing.T) {
	src, err := os.ReadFile("emitters.go")
	if err != nil {
		t.Fatalf("cannot read the seam file: %v", err)
	}
	text := string(src)
	for _, want := range []string{
		"GenerateAirWingsBlockTolerant",
		"MergeUnitsBlocksTolerant",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("SEAM NOT LANDED: emitters.go still calls the notice-discarding "+
				"form instead of %s — every skipped air wing / fleet / division is "+
				"currently dropped without telling the author.", want)
		}
	}
	if !strings.Contains(text, "env.YmlWarnings = append(env.YmlWarnings, airNotices...)") {
		t.Error("SEAM NOT LANDED: emitters.go does not forward the OOB notices into " +
			"env.YmlWarnings, so they never reach the author.")
	}
}
