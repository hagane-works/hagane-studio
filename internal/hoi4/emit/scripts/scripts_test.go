package emitscripts

import (
	"strings"
	"testing"
)

func TestOnActionsStepsIntoOwnerForLeaderScopedHooks(t *testing.T) {
	out := GenerateOnActions("GER", false, "", nil, []AutoFire{
		{EventID: "ger.1", Days: 0, Hook: "on_unit_leader_promote_from_ranks"},
		{EventID: "ger.2", Days: 5, Hook: "on_army_leader_promoted"},
	}, nil)

	for _, hook := range []string{
		"on_unit_leader_promote_from_ranks_veteran",
		"on_unit_leader_promote_from_ranks_green",
		"on_army_leader_promoted",
	} {
		block := hookBlock(t, out, hook)
		if !strings.Contains(block, "owner = {") {
			t.Fatalf("%s must step into the country scope first:\n%s", hook, block)
		}
		ownerAt := strings.Index(block, "owner = {")
		gateAt := strings.Index(block, "limit = { tag = GER }")
		if gateAt < 0 || gateAt < ownerAt {
			t.Fatalf("%s evaluates the country gate outside owner:\n%s", hook, block)
		}
		if eventAt := strings.Index(block, "country_event"); eventAt < 0 || eventAt < ownerAt {
			t.Fatalf("%s fires country_event outside the country scope:\n%s", hook, block)
		}
	}
}

func TestOnActionsLeavesCountryScopedHooksUnwrapped(t *testing.T) {
	out := GenerateOnActions("GER", false, "", nil, []AutoFire{
		{EventID: "ger.3", Days: 0, Hook: "on_war"},
		{EventID: "ger.4", Days: 2, Hook: "on_capitulation"},
	}, nil)
	for _, hook := range []string{"on_war", "on_capitulation"} {
		block := hookBlock(t, out, hook)
		if strings.Contains(block, "owner = {") {
			t.Fatalf("%s is already a country scope; owner must not be added:\n%s", hook, block)
		}
		if !strings.Contains(block, "limit = { tag = GER }") {
			t.Fatalf("%s lost its country gate:\n%s", hook, block)
		}
	}
}

func TestLeaderScopedTableMatchesTheSpecList(t *testing.T) {
	want := []string{
		"on_unit_leader_created",
		"on_army_leader_daily",
		"on_army_leader_won_combat",
		"on_army_leader_lost_combat",
		"on_unit_leader_level_up",
		"on_army_leader_promoted",
		"on_unit_leader_promote_from_ranks_veteran",
		"on_unit_leader_promote_from_ranks_green",
	}
	if len(leaderScopedOnActions) != len(want) {
		t.Fatalf("table has %d hooks, spec lists %d", len(leaderScopedOnActions), len(want))
	}
	for _, hook := range want {
		if !leaderScopedOnActions[hook] {
			t.Fatalf("%s is ROOT-is-leader in the spec but missing from the table", hook)
		}
	}
}

func hookBlock(t *testing.T, out, hook string) string {
	t.Helper()
	start := strings.Index(out, "\t"+hook+" = {")
	if start < 0 {
		t.Fatalf("no %s block in:\n%s", hook, out)
	}
	rest := out[start:]
	if end := strings.Index(rest, "\n\t}\n"); end >= 0 {
		return rest[:end]
	}
	return rest
}
