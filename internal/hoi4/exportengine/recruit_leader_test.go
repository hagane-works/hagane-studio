package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func recruitSnapshot() Snapshot {
	return Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","start_year":1936,"export_languages":["english"]}`),
		Tables: map[string][]json.RawMessage{
			"countries": {
				json.RawMessage(`{"id":1,"tag":"GER","name":"Germany","source":"created","ideology":"fascism"}`),
				json.RawMessage(`{"id":2,"tag":"ITA","name":"Italy","source":"created","ideology":"fascism"}`),
			},
			"characters": {
				json.RawMessage(`{"id":11,"tag":"GER_new_leader","character_id":"GER_new_leader","country_tag":"GER","name":"New Leader","source":"created","replaces_vanilla_id":false}`),
				json.RawMessage(`{"id":12,"tag":"GER_new_general","character_id":"GER_new_general","country_tag":"GER","name":"New General","source":"created","replaces_vanilla_id":false}`),

				json.RawMessage(`{"id":13,"tag":"GER_vanilla_swap","character_id":"GER_vanilla_swap","country_tag":"GER","name":"Swap","source":"created","replaces_vanilla_id":true}`),

				json.RawMessage(`{"id":14,"tag":"ITA_imported","character_id":"ITA_imported","country_tag":"ITA","name":"Imported","source":"imported","replaces_vanilla_id":false}`),

				json.RawMessage(`{"id":15,"tag":"GER_unassigned","character_id":"GER_unassigned","country_tag":"","name":"Unassigned","source":"created","replaces_vanilla_id":false}`),
			},
		},
	}
}

func historyFile(t *testing.T, artifact NativeArtifact, suffix string) string {
	t.Helper()
	for _, entry := range artifact.Entries {
		if strings.HasSuffix(entry.Path, suffix) {
			return string(entry.Bytes)
		}
	}
	var paths []string
	for _, entry := range artifact.Entries {
		paths = append(paths, entry.Path)
	}
	t.Fatalf("no entry ending in %q; entries=%v", suffix, paths)
	return ""
}

func finalPhysicalLine(text string) string {
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

func TestBuildNativeShadowRecruitsNewCharactersIntoHistoryFile(t *testing.T) {
	artifact, err := BuildNativeShadow(recruitSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	ger := historyFile(t, artifact, "/history/countries/GER - zDemo.txt")
	for _, want := range []string{
		"recruit_character = GER_new_leader",
		"recruit_character = GER_new_general",
		"recruit_character = GER_unassigned",
	} {
		if !strings.Contains(ger, want) {
			t.Fatalf("GER history file missing %q:\n%s", want, ger)
		}
	}
	if strings.Contains(ger, "GER_vanilla_swap") {
		t.Fatalf("replacement character must not be recruited again:\n%s", ger)
	}
	if strings.Contains(ger, "ITA_imported") {
		t.Fatalf("imported character leaked into GER:\n%s", ger)
	}

	if last := finalPhysicalLine(ger); strings.HasPrefix(last, "recruit_character") {
		t.Fatalf("recruit_character must not be the last line (got %q):\n%s", last, ger)
	}
}

func TestBuildNativeShadowSkipsImportedRecruitsUnlessIncluded(t *testing.T) {
	excluded, err := BuildNativeShadow(recruitSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range excluded.Entries {
		if strings.HasSuffix(entry.Path, "/history/countries/ITA - zDemo.txt") && strings.Contains(string(entry.Bytes), "ITA_imported") {
			t.Fatalf("imported character recruited with includeImported=false:\n%s", entry.Bytes)
		}
	}
	included, err := BuildNativeShadow(recruitSnapshot(), nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if ita := historyFile(t, included, "/history/countries/ITA - zDemo.txt"); !strings.Contains(ita, "recruit_character = ITA_imported") {
		t.Fatalf("imported character not recruited with includeImported=true:\n%s", ita)
	}
}

func TestRecruitedCharactersAreAlsoDefined(t *testing.T) {
	artifact, err := BuildNativeShadow(recruitSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var defined string
	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, "/common/characters/") {
			defined += string(entry.Bytes)
		}
	}
	for _, entry := range artifact.Entries {
		if !strings.Contains(entry.Path, "/history/countries/") {
			continue
		}
		for _, line := range strings.Split(string(entry.Bytes), "\n") {
			line = strings.TrimSpace(line)
			token, found := strings.CutPrefix(line, "recruit_character = ")
			if !found {
				continue
			}
			if !strings.Contains(defined, strings.Trim(token, `"`)) {
				t.Fatalf("%s recruits %q but no common/characters file defines it", entry.Path, token)
			}
		}
	}
}

func leaderSnapshot() Snapshot {
	snapshot := recruitSnapshot()
	snapshot.Tables["characters"] = append(snapshot.Tables["characters"],

		json.RawMessage(`{"id":21,"tag":"FRA_petain","country_tag":"FRA","role_type":"country_leader","name_en":"Petain","desc_en":"The old marshal.","ideology":"despotism","expire":"1949.1.1","traits":["defeatist","old_guard","defeatist"],"raw_extras":{},"source":"created","source_snapshot":null,"is_portrait_modified":true,"replaces_vanilla_id":true}`),

		json.RawMessage(`{"id":22,"tag":"GER_hitler_swap","country_tag":"GER","role_type":"country_leader","name_en":"Swap","ideology":"","expire":"","traits":[],"raw_extras":{},"source":"created","replaces_vanilla_id":true}`),

		json.RawMessage(`{"id":31,"tag":"FRA_gamelin","country_tag":"FRA","role_type":"field_marshal","traits":[],"raw_extras":{},"source":"created","replaces_vanilla_id":true}`),

		json.RawMessage(`{"id":32,"tag":"FRA_planned","country_tag":"FRA","role_type":"country_leader","traits":[],"raw_extras":{},"source":"created","source_snapshot":{"tag":"FRA_planned"},"replaces_vanilla_id":true}`),

		json.RawMessage(`{"id":33,"tag":"FRA_imported","country_tag":"FRA","role_type":"country_leader","traits":[],"raw_extras":{},"source":"imported","replaces_vanilla_id":true}`),

		json.RawMessage(`{"id":34,"tag":"FRA_optout","country_tag":"FRA","role_type":"country_leader","traits":[],"raw_extras":{"emit_trait_overlay_event":false},"source":"created","replaces_vanilla_id":true}`),

		json.RawMessage(`{"id":35,"tag":"FRA_newbie","country_tag":"FRA","role_type":"country_leader","traits":[],"raw_extras":{},"source":"created","replaces_vanilla_id":false}`),
	)
	return snapshot
}

func TestBuildNativeShadowEmitsCountryLeaderReplacement(t *testing.T) {
	artifact, err := BuildNativeShadow(leaderSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	fra := historyFile(t, artifact, "/history/countries/FRA - zDemo.txt")

	retire := strings.Index(fra, "retire_country_leader = yes")
	create := strings.Index(fra, "create_country_leader = {")
	if retire < 0 || create < 0 || retire > create {
		t.Fatalf("FRA needs retire_country_leader before create_country_leader:\n%s", fra)
	}
	for _, want := range []string{
		"name = FRA_petain",

		`expire = "1949.1.1"`,

		"ideology = despotism",

		"traits = { defeatist old_guard }",
	} {
		if !strings.Contains(fra, want) {
			t.Fatalf("FRA leader replacement missing %q:\n%s", want, fra)
		}
	}

	ger := historyFile(t, artifact, "/history/countries/GER - zDemo.txt")
	for _, want := range []string{
		"name = GER_hitler_swap",
		`expire = "1965.1.1"`,
		"ideology = neutrality",
	} {
		if !strings.Contains(ger, want) {
			t.Fatalf("GER leader replacement missing default %q:\n%s", want, ger)
		}
	}

	if strings.Contains(ger, "traits = { }") || strings.Contains(ger, "traits = {  }") {
		t.Fatalf("empty trait list must omit the traits line:\n%s", ger)
	}
}

func TestBuildNativeShadowLeaderReplacementExcludesNonLegacyRows(t *testing.T) {
	artifact, err := BuildNativeShadow(leaderSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var history string
	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, "/history/countries/") {
			history += string(entry.Bytes)
		}
	}
	for _, tag := range []string{
		"FRA_gamelin",
		"FRA_planned",
		"FRA_imported",
		"FRA_optout",
	} {
		if strings.Contains(history, "name = "+tag) {
			t.Fatalf("%s must not reach create_country_leader:\n%s", tag, history)
		}
	}

	if !strings.Contains(history, "name = FRA_petain") {
		t.Fatalf("exclusion test is vacuous — no leader replacement emitted at all:\n%s", history)
	}
}

func TestBuildNativeShadowLeaderReplacementIsNeverRecruited(t *testing.T) {
	artifact, err := BuildNativeShadow(leaderSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	replaced := map[string]bool{}
	recruited := map[string]bool{}
	for _, entry := range artifact.Entries {
		if !strings.Contains(entry.Path, "/history/countries/") {
			continue
		}
		for _, line := range strings.Split(string(entry.Bytes), "\n") {
			line = strings.TrimSpace(line)
			if tag, ok := strings.CutPrefix(line, "name = "); ok {
				replaced[strings.Trim(tag, `"`)] = true
			}
			if tag, ok := strings.CutPrefix(line, "recruit_character = "); ok {
				recruited[strings.Trim(tag, `"`)] = true
			}
		}
	}
	if !replaced["FRA_petain"] || !recruited["GER_new_leader"] {
		t.Fatalf("fixture no longer exercises both halves: replaced=%v recruited=%v", replaced, recruited)
	}
	for tag := range replaced {
		if recruited[tag] {
			t.Fatalf("%s is both retired+created and recruited", tag)
		}
	}
	if !recruited["FRA_newbie"] || replaced["FRA_newbie"] {
		t.Fatalf("FRA_newbie must be recruited, not replaced: replaced=%v recruited=%v", replaced, recruited)
	}
}

func TestNativeShadowLeaderReplacementReferencesResolve(t *testing.T) {
	artifact, err := BuildNativeShadow(leaderSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var loc, gfx string
	for _, entry := range artifact.Entries {
		switch {
		case strings.Contains(entry.Path, "/localisation/"):
			loc += string(entry.Bytes)
		case strings.HasSuffix(entry.Path, ".gfx"):
			gfx += string(entry.Bytes)
		}
	}
	references := 0
	for _, entry := range artifact.Entries {
		if !strings.Contains(entry.Path, "/history/countries/") {
			continue
		}
		for _, line := range strings.Split(string(entry.Bytes), "\n") {
			line = strings.TrimSpace(line)
			if key, ok := strings.CutPrefix(line, "desc = "); ok {
				references++
				if !strings.Contains(loc, strings.Trim(key, `"`)+":") {
					t.Errorf("%s writes %q but no localisation entry defines that key", entry.Path, line)
				}
			}
			if sprite, ok := strings.CutPrefix(line, "picture = "); ok {
				references++
				if !strings.Contains(gfx, "name = "+strings.Trim(sprite, `"`)) {
					t.Errorf("%s writes %q but no .gfx entry registers that spriteType", entry.Path, line)
				}
			}
		}
	}

	t.Logf("leader-replacement references emitted: %d (expected 0 until native.go:863 stops skipping replacement rows)", references)
}
