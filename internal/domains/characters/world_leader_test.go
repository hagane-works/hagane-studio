package characters

import (
	"strings"
	"testing"
)

func TestStripGeneratedWorldLeader(t *testing.T) {
	raw := `capital = 1
set_stability = 0.6

create_country_leader = {
	name = "Leader of ABC"
	desc = ""
	picture = GFX_portrait_ABC_custom
	ideology = despotism
	traits = {
	}
}
`
	got, changed := stripGeneratedWorldLeader(raw, "abc")
	if !changed {
		t.Fatal("generated legacy leader was not removed")
	}
	if strings.Contains(got, "create_country_leader") || !strings.Contains(got, "capital = 1") {
		t.Fatalf("unexpected migrated history:\n%s", got)
	}
}

func TestStripGeneratedWorldLeaderPreservesFollowingCommentsAndStatements(t *testing.T) {
	raw := "capital = 1\r\n\r\ncreate_country_leader = {\r\n" +
		"\tname = \"Leader of ABC\"\r\n\tdesc = \"\"\r\n" +
		"\tpicture = GFX_portrait_Europe_Generic_1\r\n\tideology = despotism\r\n" +
		"\ttraits = { }\r\n}\r\n# Keep this authored note { literally }\r\nadd_political_power = 5\r\n"
	want := "capital = 1\r\n\r\n# Keep this authored note { literally }\r\nadd_political_power = 5\r\n"
	got, changed := stripGeneratedWorldLeader(raw, "ABC")
	if !changed || got != want {
		t.Fatalf("migration changed content outside its block:\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestStripGeneratedWorldLeaderPreservesAuthoredBlocks(t *testing.T) {
	cases := []string{
		`create_country_leader = { name = "Alice" desc = "" picture = GFX_portrait_ABC_Alice ideology = despotism traits = { } }`,
		`create_country_leader = { name = "Leader of ABC" desc = "" picture = GFX_portrait_ABC_Alice ideology = democratic traits = { } }`,
		`create_country_leader = { name = "Leader of ABC" desc = "" picture = GFX_portrait_ABC_Alice ideology = despotism traits = { } expire = "1965.1.1" }`,
		`create_country_leader = { name = "Leader of ABC" desc = "" picture = GFX_portrait_ABC_Alice ideology = despotism traits = { }`,
	}
	for _, raw := range cases {
		if got, changed := stripGeneratedWorldLeader(raw, "ABC"); changed || got != raw {
			t.Fatalf("authored block changed:\n%s", raw)
		}
	}
}

func TestStripGeneratedWorldLeaderKeepsAStatementBeforeTheBlockOnItsLine(t *testing.T) {
	raw := `capital = 1 create_country_leader = {
	name = "Leader of ABC"
	desc = ""
	picture = GFX_portrait_Europe_Generic_1
	ideology = despotism
	traits = { }
}
add_political_power = 5
`
	if got, changed := stripGeneratedWorldLeader(raw, "ABC"); changed || got != raw {
		t.Fatalf("migration deleted a statement on the block's first line:\n%s", got)
	}
}
