package importjob

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"haganeworks/internal/project"
)

func TestInFileOrder(t *testing.T) {
	previous := runtime.GOMAXPROCS(6)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	const n = 200
	var active, most atomic.Int64
	parse := func(i int) (int, error) {
		now := active.Add(1)
		for old := most.Load(); now > old && !most.CompareAndSwap(old, now); old = most.Load() {
		}
		time.Sleep(time.Duration(rand.Intn(300)) * time.Microsecond)
		active.Add(-1)
		return i * i, nil
	}
	var order []int
	err := inFileOrder(context.Background(), n, parse, func(i, value int) error {
		if value != i*i {
			t.Errorf("file %d got %d", i, value)
		}
		order = append(order, i)
		return nil
	})
	if err != nil || len(order) != n {
		t.Fatalf("err %v, used %d of %d", err, len(order), n)
	}
	for i, got := range order {
		if got != i {
			t.Fatalf("used %d in place %d", got, i)
		}
	}
	if most.Load() < 2 || most.Load() > 6 {
		t.Errorf("at most %d parses at once, want 2..6", most.Load())
	}

	parseErr, useErr := errors.New("cannot read file 40"), errors.New("cannot use file 70")
	used := 0
	err = inFileOrder(context.Background(), n, func(i int) (int, error) {
		if i == 40 || i == 120 {
			return 0, parseErr
		}
		return i, nil
	}, func(i, _ int) error {
		used++
		if i == 70 {
			return useErr
		}
		return nil
	})
	if !errors.Is(err, parseErr) || used != 40 {
		t.Errorf("err %v after %d files, want the parse error after 40", err, used)
	}

	ctx, cancel := context.WithCancel(context.Background())
	err = inFileOrder(ctx, n, func(i int) (int, error) { return i, nil }, func(i, _ int) error {
		if i == 10 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("after cancel: %v", err)
	}
}

func parallelTestMod() map[string]string {
	files := map[string]string{}
	tags := []string{"TST", "AAA", "BBB", "CCC"}
	for f := 0; f < 24; f++ {
		tag := tags[f%len(tags)]
		var events, decisions, chars, ideas, focus, loc, gfx, effects, modifiers strings.Builder
		fmt.Fprintf(&events, "add_namespace = %s_ns%d\n", strings.ToLower(tag), f%5)
		for e := 0; e < 15+f; e++ {
			fmt.Fprintf(&events, "country_event = { id = %s_ns%d.%d title = ev_%d_%d picture = GFX_pic_%d trigger = { tag = %s } option = { name = ok add_political_power = %d } }\n",
				strings.ToLower(tag), f%5, e, f, e, e%7, tag, f)
			fmt.Fprintf(&decisions, "%s_cat_%d = { %s_decision_%d = { icon = GFX_dec_%d cost = %d allowed = { original_tag = %s } } }\n", tag, e%3, tag, e, e%4, f+e, tag)
			fmt.Fprintf(&chars, "%s_char_%d = { name = %s_char_%d portraits = { civilian = { large = GFX_portrait_%d } } country_leader = { ideology = despotism } }\n", tag, e, tag, e, f)
			fmt.Fprintf(&ideas, "%s_idea_%d = { picture = idea_%d modifier = { stability_factor = 0.0%d } }\n", tag, e, f, e%9)
			fmt.Fprintf(&loc, " %s_char_%d:0 \"Name %d from file %d\"\n", tag, e, e, f)
			fmt.Fprintf(&gfx, "\tspriteType = { name = \"GFX_pic_%d\" texturefile = \"gfx/event_pictures/pic_%d_%d.dds\" }\n", e%7, f, e%7)
			fmt.Fprintf(&effects, "tst_effect_%d = { add_political_power = %d }\n", e, f)
			fmt.Fprintf(&modifiers, "tst_modifier_%d = { color_type = good value_type = number precision = %d }\n", e, f%3)
		}
		fmt.Fprintf(&focus, "focus_tree = {\n\tid = tree_%d\n\tcountry = { factor = 0 modifier = { add = 10 tag = %s } }\n", f%6, tag)
		for n := 0; n < 10; n++ {
			fmt.Fprintf(&focus, "\tfocus = { id = %s_focus_%d_%d icon = GFX_goal_%d x = %d y = %d cost = 10 completion_reward = { add_political_power = %d } }\n", tag, f%6, n, n, n, f%3, f)
		}
		focus.WriteString("}\n")
		name := fmt.Sprintf("%02d_%s", f, strings.ToLower(tag))
		files["events/"+name+".txt"] = events.String()
		files["common/decisions/"+name+".txt"] = decisions.String()
		files["common/characters/"+name+".txt"] = "characters = {\n" + chars.String() + "}\n"
		files["common/ideas/"+name+".txt"] = "ideas = {\n\tcountry = {\n" + ideas.String() + "\t}\n}\n"
		files["common/national_focus/"+name+".txt"] = focus.String()
		files["common/technologies/"+name+".txt"] = fmt.Sprintf("technologies = {\n\ttech_%d = { research_cost = %d start_year = 1936 folder = { name = infantry_folder position = { x = %d y = 0 } } }\n}\n", f%8, f+1, f)
		files["common/scripted_effects/"+name+".txt"] = effects.String()
		files["common/modifier_definitions/"+name+".txt"] = modifiers.String()
		files["common/on_actions/"+name+".txt"] = fmt.Sprintf("on_actions = { on_startup = { effect = { set_global_flag = flag_%d } } }\n", f)
		files["common/units/"+name+".txt"] = fmt.Sprintf("sub_units = { unit_%d = { type = land } }\n", f)
		files["history/countries/"+tag+" - "+name+".txt"] = fmt.Sprintf("capital = %d\nset_research_slots = %d\n", f+1, f%4)
		files[fmt.Sprintf("history/states/%d-%s.txt", f%10+1, name)] = fmt.Sprintf("state = { id = %d name = \"STATE_%d\" manpower = %d state_category = town provinces = { %d } history = { owner = %s } }\n", f%10+1, f, 1000+f, f+1, tag)
		files["localisation/"+name+"_l_english.yml"] = "l_english:\n" + loc.String()
		files["interface/"+name+".gfx"] = "spriteTypes = {\n" + gfx.String() + "}\n"
	}
	for _, rel := range []string{"events/zz_broken.txt", "common/characters/zz_broken.txt", "common/ideas/zz_broken.txt", "common/national_focus/zz_broken.txt", "common/scripted_effects/zz_broken.txt", "common/modifier_definitions/zz_broken.txt", "common/decisions/zz_broken.txt"} {
		files[rel] = "broken = { a = { = } } } }\n\"unterminated"
	}
	return files
}

func TestParallelImportMatchesSequential(t *testing.T) {
	previous := runtime.GOMAXPROCS(0)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	mod := parallelTestMod()
	run := func(procs int) (map[string][]byte, State) {
		runtime.GOMAXPROCS(procs)
		root := writeWorldTestMod(t, mod)
		p, err := project.NewStore(t.TempDir()).Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST", "AAA", "BBB"}}, "t")
		if err != nil {
			t.Fatal(err)
		}
		m := NewManager()
		if _, err := m.Start(p, "s1", root, AllOptions()); err != nil {
			t.Fatal(err)
		}
		state := waitForJob(t, m, p.ID(), "s1")
		if state.Status != "complete" {
			t.Fatalf("GOMAXPROCS %d: import %s: %s", procs, state.Status, state.Error)
		}
		return projectFiles(t, p.Dir()), state
	}
	sequentialFiles, sequential := run(1)
	parallelFiles, parallel := run(8)

	if len(sequentialFiles) != len(parallelFiles) {
		t.Errorf("%d files sequentially, %d in parallel", len(sequentialFiles), len(parallelFiles))
	}
	for name, want := range sequentialFiles {
		if got := parallelFiles[name]; !bytes.Equal(got, want) {
			at := 0
			for at < len(got) && at < len(want) && got[at] == want[at] {
				at++
			}
			t.Errorf("%s differs at byte %d: %q vs %q", name, at, want[max(0, at-60):min(len(want), at+60)], got[max(0, at-60):min(len(got), at+60)])
		}
	}
	if fmt.Sprint(sequential.Result) != fmt.Sprint(parallel.Result) || sequential.Rows != parallel.Rows {
		t.Errorf("results differ:\n%v\n%v", sequential.Result, parallel.Result)
	}

	for _, table := range []string{"events", "decisions", "characters", "ideas", "focus_trees", "technologies", "scripted_snippets", "project_loc_entries", "states", "history_files", "project_raw_files"} {
		if len(sequentialFiles["data/"+table+".json"]) < 10 {
			t.Errorf("nothing imported into %s", table)
		}
	}
	if warnings, _ := sequential.Result["warnings"].([]string); len(warnings) == 0 {
		t.Error("the broken files were not reported")
	}
}

func waitForJob(t *testing.T, m *Manager, projectID int64, sessionID string) State {
	t.Helper()
	deadline := time.Now().Add(time.Minute)
	for time.Now().Before(deadline) {
		if state, _ := m.Get(projectID, sessionID); state.Status != "running" {
			return state
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the import did not end")
	return State{}
}

var timestampRE = regexp.MustCompile(`"\d{4}-\d\d-\d\d[T ]\d\d:\d\d:\d\d(\.\d+)?(Z|[+-]\d\d:\d\d)?"`)

func projectFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		out[filepath.ToSlash(rel)] = timestampRE.ReplaceAll(body, []byte(`"<time>"`))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
