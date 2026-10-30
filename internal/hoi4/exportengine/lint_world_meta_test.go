package exportengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportmsg"
)

func loadLintWorldSnapshot(t *testing.T) Snapshot {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "lint_world", "snapshot.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return snapshot
}

var lintWorldExpected = map[string]string{
	"expected_POLITICS_EXPORT_WARNINGS.log": `# Politics and ideology export warnings
# Things noticed while writing the country history files and common/ideologies that did not stop the export. Check each line before you release the mod: a dropped ruling party, two rows for the same country, or how your own ideologies were written.

ZZ1: the history's ruling_party "monarchism_custom" does not belong to any ideology group that can be exported, so it was left out (writing the sub-ideology itself would be invalid in the game). Set the country's ideology to a sub-ideology of a defined group, or define your own group and its sub-ideology in the ideology list.
`,
	"expected_STATE_ORPHAN_TAG_LINT.log": `# States that name a country that does not exist
# Each line is a state whose owner, controller or add_core_of names a tag that is neither a base-game country nor a country in this project. HOI4 reports an error when a state's owner is an unknown country. Fix it by adding the country to the project, correcting the typo, or importing the missing country files again.

State 5 (Testland): owner = XQZ — no country in this project or the base game has this tag; HOI4 reports an error when loading.
State 5 (Testland): add_core_of = XQZ — no country in this project or the base game has this tag; HOI4 reports an error when loading.
`,
}

func loadLintWorldExpected(t *testing.T, name string) string {
	t.Helper()
	want, ok := lintWorldExpected[name]
	if !ok {
		t.Fatalf("no expected text for %s", name)
	}
	return want
}

func worldMetaFor(t *testing.T, snapshot Snapshot, includeImported bool) worldout.Meta {
	t.Helper()
	project := worldout.NewOMap()
	if err := project.UnmarshalJSON(snapshot.Project); err != nil {
		t.Fatalf("decode project: %v", err)
	}
	rows := func(table string, filterSource bool) []*worldout.OMap {
		out := []*worldout.OMap{}
		for _, raw := range snapshot.Tables[table] {
			row := worldout.NewOMap()
			if err := row.UnmarshalJSON(raw); err != nil {
				t.Fatalf("decode %s: %v", table, err)
			}
			if filterSource && !includeImported && worldString(row, "source") == "imported" {
				continue
			}
			out = append(out, row)
		}
		return out
	}
	countries := rows("countries", false)
	historyFiles := rows("history_files", true)
	states := rows("states", true)
	bookmarks := rows("bookmarks", true)
	contentByTag := worldout.NewOMap()
	for _, set := range [][]*worldout.OMap{countries, historyFiles} {
		for _, row := range set {
			if ctag := strings.ToUpper(strings.TrimSpace(firstWorldString(row, "tag", "country_tag"))); ctag != "" {
				contentByTag.Set(ctag, true)
			}
		}
	}
	recruitByCtag, clByCtag, err := BuildRecruitAndLeaderMaps(snapshot, includeImported)
	if err != nil {
		t.Fatalf("recruit maps: %v", err)
	}
	tables, err := testWorldTables(t)
	if err != nil {
		t.Fatalf("tables: %v", err)
	}
	scenario := &worldout.Scenario{
		Project: project, Tag: strings.ToUpper(strings.TrimSpace(worldString(project, "country_tag"))),
		ModName: worldString(project, "name"), ModDirName: "Demo",
		IncludeImported: includeImported,
		ExistenceSets:   map[string]map[string]bool{"idea": {}, "focus": {}},
		ContentByTag:    contentByTag, RecruitByCtag: recruitByCtag, ClByCtag: clByCtag,
		Countries: countries, HistoryFiles: historyFiles, States: states, Bookmarks: bookmarks,
	}
	_, meta := worldout.EmitScenario(tables, scenario)
	return meta
}

func TestApplyWorldLintsIsQuietOnZeroMeta(t *testing.T) {
	artifact := NativeArtifact{Filename: "Demo.zip", Entries: []Entry{{Path: "Demo.mod"}}}
	ApplyWorldLints(&artifact, worldout.Meta{}, english)
	if len(artifact.Warnings) != 0 || len(artifact.Coverage) != 0 || len(artifact.Entries) != 1 {
		t.Fatalf("zero Meta must add nothing: warnings=%v coverage=%v entries=%d",
			artifact.Warnings, artifact.Coverage, len(artifact.Entries))
	}
	ApplyWorldLints(nil, worldout.Meta{Politics: []exportmsg.Message{{Key: "politics.ruling_party_dropped", Args: exportmsg.Args{"subject": "GER"}}}}, english)
}

func TestApplyWorldLintsPushesHeadlinesLogsAndCoverage(t *testing.T) {
	snapshot := loadLintWorldSnapshot(t)
	meta := worldMetaFor(t, snapshot, false)
	artifact := NativeArtifact{Filename: "Demo.zip", Entries: []Entry{{Path: "Demo.mod"}}}
	ApplyWorldLints(&artifact, meta, english)

	if len(artifact.Warnings) != 2 {
		t.Fatalf("expected exactly 2 headlines (politics, orphan), got %v", artifact.Warnings)
	}
	if !strings.HasPrefix(artifact.Warnings[0], "Politics export: warnings for ZZ1 (1)") {
		t.Fatalf("politics headline: %q", artifact.Warnings[0])
	}
	if !strings.HasPrefix(artifact.Warnings[1], "State owners: states that name an unknown country tag as owner, controller or core (1; tags: XQZ)") {
		t.Fatalf("orphan headline: %q", artifact.Warnings[1])
	}
	for _, name := range []string{PoliticsLogName, StateOrphanTagLogName} {
		entry, ok := findEntry(artifact, "Demo/"+name)
		if !ok {
			t.Fatalf("%s not written; entries=%v", name, entryPaths(artifact))
		}
		want := loadLintWorldExpected(t, "expected"+name)
		if string(entry.Bytes) != want {
			t.Fatalf("%s differs from the expected English log:\n--- got\n%s\n--- want\n%s", name, entry.Bytes, want)
		}
	}
	joined := strings.Join(artifact.Coverage, ",")
	if !strings.Contains(joined, "politics_export_lint") || !strings.Contains(joined, "state_orphan_tag_lint") {
		t.Fatalf("coverage tags missing: %v", artifact.Coverage)
	}
}
