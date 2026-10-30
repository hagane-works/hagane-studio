package paradox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const defaultVanillaRoot = `G:\Steam\steamapps\common\Hearts of Iron IV`

const excerptAfarCountry = "\n" +
	"\n" +
	"graphical_culture = african_gfx\n" +
	"graphical_culture_2d = african_2d\n" +
	"\n" +
	"color = rgb { 153 0 51 }"

const excerptIdeologies = `ideologies = {

	democratic = {

		types = {

			conservatism = {
			}

			liberalism = {
			}

		}

		dynamic_faction_names = {
			"FACTION_NAME_DEMOCRATIC_1"
			"FACTION_NAME_DEMOCRATIC_2"
		}

		color = { 0 0 255 }

		rules = {
			can_create_collaboration_government = no
			can_declare_war_on_same_ideology = no
		}

		can_host_government_in_exile = yes

		war_impact_on_world_tension = 0.25		# no major danger, remember this this the 40s!
		faction_impact_on_world_tension = 0.1
	}
}
`

const excerptBookmark = `bookmarks = {
	bookmark = {
		name = "GATHERING_STORM_NAME"
		desc = "GATHERING_STORM_DESC"
		date = 1936.1.1.12
		picture = "GFX_select_date_1936"
		default_country = "GER"
		default = yes

		filters = { new_content continent }
		# filters can also contain 'label'
	}
}
`

const excerptEconomicIdea = `			allowed = {
				always = no
			}
			allowed_to_remove = {
				OR = {
					surrender_progress > 0.75
					NOT = { has_government = communism }
				}
			}
`

const excerptSpriteTypes = `spriteTypes = {

	#Government in Exile

	spriteType = {
		name = "GFX_countryview_government_in_exile_bg"
		texturefile = "gfx/interface/government_in_exile/countryview_government_in_exile_bg.dds"
	}
}
`

func TestVanillaExcerptAfarCountry(t *testing.T) {
	ast, rec := mustParse(t, excerptAfarCountry)
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	if len(ast.Entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(ast.Entries))
	}
	color := ast.Entries[2]
	if key(color) != "color" {
		t.Fatalf("entry 2 key = %q", key(color))
	}

	if color.Qualifier == nil || *color.Qualifier != "rgb" {
		t.Fatalf("rgb qualifier lost on an EOF-terminated file: %v", color.Qualifier)
	}
	if color.Block == nil || len(color.Block.Entries) != 3 {
		t.Fatalf("colour body wrong: %+v", color.Block)
	}
	if color.Line != 6 {
		t.Errorf("color line = %d, want 6", color.Line)
	}
}

func TestVanillaExcerptIdeologies(t *testing.T) {
	ast, rec := mustParse(t, excerptIdeologies)
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	dem := ast.Entries[0].Block.Entries[0]
	byKey := map[string]*Entry{}
	for _, e := range dem.Block.Entries {
		if e.Key != nil {
			byKey[*e.Key] = e
		}
	}
	for _, want := range []string{"types", "dynamic_faction_names", "color", "rules",
		"can_host_government_in_exile", "war_impact_on_world_tension",
		"faction_impact_on_world_tension"} {
		if byKey[want] == nil {
			t.Fatalf("democratic.%s missing — sibling keys were dropped", want)
		}
	}
	names := byKey["dynamic_faction_names"].Block.Entries
	if len(names) != 2 || !names[0].Quoted || names[0].Scalar.Str != "FACTION_NAME_DEMOCRATIC_1" {
		t.Fatalf("quoted list mangled: %+v", names)
	}
	if v := byKey["war_impact_on_world_tension"]; v.Scalar.Kind != ScalarFloat ||
		v.Scalar.StrValue() != "0.25" {
		t.Fatalf("float with a tab-separated trailing comment mangled: %+v", v.Scalar)
	} else if len(v.Trivia.Trailing) != 1 ||
		!strings.HasPrefix(v.Trivia.Trailing[0].Text, "# no major danger") {
		t.Fatalf("trailing comment on that line lost: %+v", v.Trivia.Trailing)
	}
	if v := byKey["can_host_government_in_exile"]; v.Scalar.Str != "yes" {
		t.Fatalf("`yes` must stay a string scalar, got %+v", v.Scalar)
	}
}

func TestVanillaExcerptBookmark(t *testing.T) {
	ast, rec := mustParse(t, excerptBookmark)
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	bm := ast.Entries[0].Block.Entries[0].Block
	byKey := map[string]*Entry{}
	for _, e := range bm.Entries {
		if e.Key != nil {
			byKey[*e.Key] = e
		}
	}
	date := byKey["date"]
	if date == nil || date.Scalar.Kind != ScalarStr || date.Scalar.Str != "1936.1.1.12" {
		t.Fatalf("4-part date literal mangled: %+v", date)
	}
	if date.Quoted {
		t.Fatal("bare date must not be marked Quoted (it would be re-emitted in quotes)")
	}
	filters := byKey["filters"]
	if filters == nil || filters.Block == nil || len(filters.Block.Entries) != 2 {
		t.Fatalf("filters list wrong: %+v", filters)
	}
	if byKey["name"].Scalar.Str != "GATHERING_STORM_NAME" || !byKey["name"].Quoted {
		t.Fatalf("quoted loc key mangled: %+v", byKey["name"])
	}
}

func TestVanillaExcerptEconomicIdeaTrigger(t *testing.T) {
	ast, rec := mustParse(t, excerptEconomicIdea)
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	or := ast.Entries[1].Block.Entries[0]
	if key(or) != "OR" || or.Block == nil {
		t.Fatalf("OR block missing: %+v", or)
	}
	cmp := or.Block.Entries[0]
	if key(cmp) != "surrender_progress" || cmp.Op == nil || *cmp.Op != ">" {
		t.Fatalf("comparison operator lost: key=%s op=%v", key(cmp), cmp.Op)
	}
	if cmp.Scalar.Kind != ScalarFloat || cmp.Scalar.StrValue() != "0.75" {
		t.Fatalf("threshold value wrong: %+v", cmp.Scalar)
	}
	not := or.Block.Entries[1]
	if key(not) != "NOT" || not.Block == nil || len(not.Block.Entries) != 1 {
		t.Fatalf("single-line nested block wrong: %+v", not)
	}
}

func TestVanillaExcerptSpriteTypes(t *testing.T) {
	ast, rec := mustParse(t, excerptSpriteTypes)
	if len(rec) != 0 {
		t.Fatalf("unexpected recoveries: %+v", rec)
	}
	sprite := ast.Entries[0].Block.Entries[0]
	if key(sprite) != "spriteType" || sprite.Block == nil {
		t.Fatalf("spriteType entry wrong: %+v", sprite)
	}
	tex := sprite.Block.Entries[1]

	if key(tex) != "texturefile" || !tex.Quoted ||
		tex.Scalar.Str != "gfx/interface/government_in_exile/countryview_government_in_exile_bg.dds" {
		t.Fatalf("texture path mangled: %+v", tex)
	}
	if len(sprite.Trivia.Leading) != 1 ||
		sprite.Trivia.Leading[0].Text != "#Government in Exile" {
		t.Fatalf("section comment lost: %+v", sprite.Trivia.Leading)
	}
}

func vanillaRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv("HOI4_VANILLA_DIR")
	if root == "" {
		root = defaultVanillaRoot
	}
	if st, err := os.Stat(filepath.Join(root, "common")); err != nil || !st.IsDir() {
		t.Skipf("SKIPPED (no vanilla install) · looked in %q · "+
			"set HOI4_VANILLA_DIR to run the full-corpus sweep. "+
			"The embedded 1.19.2 excerpts above still ran.", root)
	}
	return root
}

func vanillaVersion(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, "launcher-settings.json"))
	if err != nil {
		return ""
	}
	var meta struct {
		RawVersion string `json:"rawVersion"`
	}
	if json.Unmarshal(raw, &meta) != nil {
		return ""
	}
	return meta.RawVersion
}

var baselineUnclosedAtEOF = []struct {
	path     string
	unclosed int
}{
	{`common\doctrines\subdoctrines\sea\navy_submarine_doctrines.txt`, 1},
	{`common\focus_inlay_windows\ast_right_vs_left_campaign_empty_inlay_window.txt`, 1},
	{`common\ideas\SOV.txt`, 1},
	{`common\ideas\persia.txt`, 1},
	{`common\ideas\switzerland.txt`, 1},
	{`common\military_industrial_organization\organizations\BRA_organization.txt`, 1},
	{`common\scripted_guis\AST_cabinet_trust_scripted_gui.txt`, 1},
	{`common\units\names_divisions\BRA_names_divisions.txt`, 1},
	{`common\units\names_divisions\FIN_names_divisions.txt`, 6},
	{`history\countries\NOR - Norway.txt`, 1},
	{`history\units\AST_1936.txt`, 1},
	{`history\units\FRA_1936.txt`, 1},
	{`history\units\FRA_1936_nsb.txt`, 1},
	{`history\units\FRA_1939_naval_legacy.txt`, 1},
	{`history\units\FRA_1939_naval_mtg.txt`, 1},
	{`interface\backend.gui`, 1},
	{`interface\ger_monroe_doctrine_scripted_gui.gui`, 1},
	{`interface\powerbalanceview.gfx`, 1},
	{`interface\sia_movie_theater_campaigns_scripted_gui.gui`, 1},
	{`interface\sov_propaganda_campaigns_scripted_gui.gui`, 1},
}

func TestVanillaInstallSweep(t *testing.T) {
	root := vanillaRoot(t)

	dirs := []string{"common", "events", "history", "interface", "map"}
	type failure struct {
		path string
		msg  string
	}
	var (
		files, entries int
		failures       []failure
		recoveryCount  int
		recoveryFiles  []string
	)

	unclosed := map[string]int{}
	recoveredEntries := map[string]int{}

	for _, d := range dirs {
		base := filepath.Join(root, d)
		if st, err := os.Stat(base); err != nil || !st.IsDir() {
			continue
		}
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".txt" && ext != ".gfx" && ext != ".gui" && ext != ".asset" {
				return nil
			}
			if info.Size() > 8<<20 {
				return nil
			}
			raw, rerr := os.ReadFile(path)
			if rerr != nil {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			files++
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("PANIC parsing %s: %v", rel, r)
					}
				}()
				ast, rec, perr := Parse(string(raw))
				if perr != nil {
					if _, ok := perr.(*ParseError); !ok {
						t.Errorf("%s: error type %T, want *ParseError", rel, perr)
					}
					failures = append(failures, failure{rel, perr.Error()})
					return
				}
				n := countEntries(ast.Entries)
				entries += n
				if len(rec) > 0 {
					recoveryCount += len(rec)
					recoveryFiles = append(recoveryFiles, fmt.Sprintf("%s (%d)", rel, len(rec)))
					for _, r := range rec {
						if r.Code == RecoveryUnclosedBlock {
							unclosed[rel]++
							recoveredEntries[rel] = n
						}
					}
				}

				_ = EntriesJSON(ast.Entries)
			}()
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", base, err)
		}
	}

	if files < 500 {
		t.Fatalf("only %d files swept from %q — the install looks wrong, "+
			"refusing to report a green sweep on a near-empty corpus", files, root)
	}
	version := vanillaVersion(root)
	regained := 0
	for _, n := range recoveredEntries {
		regained += n
	}

	t.Logf("vanilla sweep · root=%s · version=%s · files=%d · entries=%d · "+
		"parse errors=%d · recoveries=%d · entries regained by EOF repair=%d "+
		"across %d files",
		root, version, files, entries, len(failures), recoveryCount,
		regained, len(recoveredEntries))

	sort.Slice(failures, func(i, j int) bool { return failures[i].path < failures[j].path })

	for _, f := range failures {
		t.Errorf("vanilla file DISCARDED (whole-file content loss): %s → %s", f.path, f.msg)
	}

	if version != "1.19.2.0" {
		for p, n := range unclosed {
			t.Logf("(unpinned version %q) repaired at EOF: %s (%d unclosed, %d entries)",
				version, p, n, recoveredEntries[p])
		}
		t.Logf("SKIPPED the exact-set assertion: baseline is pinned to 1.19.2.0, " +
			"this install is different. Re-baseline before trusting this sweep.")
	} else {
		want := map[string]int{}
		for _, b := range baselineUnclosedAtEOF {
			want[b.path] = b.unclosed
		}
		for p, gotN := range unclosed {
			wantN, ok := want[p]
			if !ok {
				t.Errorf("NEW vanilla file needing EOF repair (not in baseline): %s (%d)", p, gotN)
				continue
			}
			if gotN != wantN {
				t.Errorf("%s: %d unclosed-block recoveries, baseline says %d", p, gotN, wantN)
			}
			if recoveredEntries[p] == 0 {
				t.Errorf("%s: repaired but produced ZERO entries — the file is still "+
					"effectively discarded", p)
			}
		}
		for p := range want {
			if _, ok := unclosed[p]; !ok {
				t.Errorf("baseline file no longer needs EOF repair — delete it from "+
					"baselineUnclosedAtEOF in this same commit: %s", p)
			}
		}
	}

	sort.Strings(recoveryFiles)
	for _, r := range recoveryFiles {
		t.Logf("recovery fired on vanilla file: %s", r)
	}
}

func TestVanillaUnclosedShapeIsRepairedWithoutTheGame(t *testing.T) {
	const src = `ast_right_vs_left_campaign_empty_inlay_window = {

    window_name = ast_right_vs_left_campaign_empty_ui_window

    internal = yes
    visible = {
        OR = {
            NOT = { has_completed_focus = AST_war_is_coming }
        }
    }
`
	ast, rec, err := Parse(src)
	if err != nil {
		t.Fatalf("this shape must no longer discard the file: %v", err)
	}
	if len(rec) != 1 || rec[0].Code != RecoveryUnclosedBlock {
		t.Fatalf("the repair must be reported, not silent: %+v", rec)
	}
	if rec[0].Line != 1 || rec[0].Details["key"] != "ast_right_vs_left_campaign_empty_inlay_window" {
		t.Errorf("diagnostic must point at the unclosed `{`: %+v", rec[0])
	}

	if len(ast.Entries) != 1 {
		t.Fatalf("want 1 top-level entry, got %d", len(ast.Entries))
	}
	body := ast.Entries[0].Block
	if body == nil || len(body.Entries) != 3 {
		t.Fatalf("window body lost: %+v", body)
	}
	if key(body.Entries[0]) != "window_name" || key(body.Entries[2]) != "visible" {
		t.Fatalf("window body mangled: %+v", body.Entries)
	}
}

func TestVanillaSOVIdeasSurviveTheUnclosedBlock(t *testing.T) {
	root := vanillaRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "common", "ideas", "SOV.txt"))
	if err != nil {
		t.Skipf("SKIPPED (SOV.txt unreadable): %v", err)
	}
	ast, rec, perr := Parse(string(raw))
	if perr != nil {
		t.Fatalf("SOV.txt still discarded: %v", perr)
	}
	if len(rec) != 1 || rec[0].Code != RecoveryUnclosedBlock || rec[0].Line != 1 {
		t.Fatalf("expected one unclosed-block repair on line 1, got %+v", rec)
	}
	if len(ast.Entries) != 1 || key(ast.Entries[0]) != "ideas" {
		t.Fatalf("top-level `ideas` block missing: %+v", ast.Entries)
	}

	if vanillaVersion(root) != "1.19.2.0" {
		t.Logf("SKIPPED the exact counts (install is not 1.19.2.0)")
		return
	}
	perGroup := map[string]int{}
	for _, g := range ast.Entries[0].Block.Entries {
		if g.Key != nil && g.Block != nil {
			perGroup[*g.Key] += len(g.Block.Entries)
		}
	}

	want := map[string]int{
		"country":               154,
		"hidden_ideas":          16,
		"tank_manufacturer":     41,
		"naval_manufacturer":    49,
		"aircraft_manufacturer": 33,
		"materiel_manufacturer": 26,
		"industrial_concern":    23,
	}
	total := 0
	for group, n := range want {
		if perGroup[group] != n {
			t.Errorf("SOV.txt %s = %d ideas, want %d", group, perGroup[group], n)
		}
		total += n
	}
	if got := len(perGroup); got != len(want) {
		t.Errorf("SOV.txt idea groups = %d, want %d (%v)", got, len(want), perGroup)
	}
	t.Logf("SOV.txt recovered: %d national spirits (`country`), %d idea entries "+
		"across %d groups — was 0 before the EOF repair", want["country"], total, len(want))
}

func countEntries(es []*Entry) int {
	n := 0
	for _, e := range es {
		n++
		if e.Block != nil {
			n += countEntries(e.Block.Entries)
		}
	}
	return n
}
