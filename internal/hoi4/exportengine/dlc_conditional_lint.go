package exportengine

import (
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/hoi4/vanilladata"
)

const DLCConditionalLogName = "_DLC_CONDITIONAL_LINT.log"

const dlcHistoryStatesDir = "history/states"

const dlcMaxGuardedKeys = 8

type DLCHit struct {
	DLC     string
	Known   bool
	Suggest string
	Negated bool
	Path    string
	Dir     string
	Line    int
	Context string
	Guarded []string
}

type DLCReport struct {
	Hits        []DLCHit
	ByDLC       map[string]int
	ByDir       map[string]int
	ParseFailed []string
	msg         exportmsg.Printer
}

type dlcAncestor struct {
	entry *paradox.Entry
	block *paradox.Block
}

var dlcEffectContainers = map[string]bool{
	"if": true, "else_if": true, "while_loop_effect": true, "for_each_scope_loop": true,
	"random_list": false,
}

var dlcSkipKeysInIf = map[string]bool{"limit": true, "else": true, "else_if": true}

func ScanDLCConditionals(entries []Entry, msg exportmsg.Printer) *DLCReport {
	report := &DLCReport{ByDLC: map[string]int{}, ByDir: map[string]int{}, msg: msg}
	root := artifactRootFromEntries(entries)
	for _, entry := range entries {
		if !isScriptEntry(entry) {
			continue
		}
		text := scriptText(entry)
		if !strings.Contains(text, "has_dlc") {
			continue
		}
		rel := dlcRelPath(entry.Path, root)
		block, _, err := paradox.Parse(text)
		if err != nil || block == nil {
			report.ParseFailed = append(report.ParseFailed, rel)
			continue
		}
		dlcWalk(block, nil, func(e *paradox.Entry, stack []dlcAncestor) {
			hit, ok := dlcHitFromEntry(e, stack)
			if !ok {
				return
			}
			hit.Path = rel
			hit.Dir = dlcDirOf(rel)
			report.Hits = append(report.Hits, hit)
			report.ByDLC[hit.DLC]++
			report.ByDir[hit.Dir]++
		})
	}
	sort.SliceStable(report.Hits, func(i, j int) bool {
		a, b := report.Hits[i], report.Hits[j]
		if a.Path != b.Path {
			return dlcDirRank(a.Dir) < dlcDirRank(b.Dir) || (dlcDirRank(a.Dir) == dlcDirRank(b.Dir) && a.Path < b.Path)
		}
		return a.Line < b.Line
	})
	return report
}

func dlcWalk(block *paradox.Block, stack []dlcAncestor, visit func(*paradox.Entry, []dlcAncestor)) {
	if block == nil {
		return
	}
	for _, entry := range block.Entries {
		visit(entry, stack)
		if entry.Block != nil {
			dlcWalk(entry.Block, append(stack, dlcAncestor{entry: entry, block: entry.Block}), visit)
		}
	}
}

func dlcKey(e *paradox.Entry) string {
	if e == nil || e.Key == nil {
		return ""
	}
	return *e.Key
}

func dlcHitFromEntry(e *paradox.Entry, stack []dlcAncestor) (DLCHit, bool) {
	if dlcKey(e) != "has_dlc" || e.Block != nil {
		return DLCHit{}, false
	}
	name := strings.TrimSpace(e.Scalar.StrValue())
	if name == "" {
		return DLCHit{}, false
	}
	hit := DLCHit{DLC: name, Line: e.Line}
	hit.Known = vanilladata.IsKnownDLCName(name)
	if !hit.Known {
		hit.Suggest, _ = vanilladata.SuggestDLCName(name)
	}

	notCount := 0
	limitIdx := -1
	for i := len(stack) - 1; i >= 0; i-- {
		key := dlcKey(stack[i].entry)
		switch {
		case key == "NOT":
			notCount++
			continue
		case key == "AND" || key == "OR" || key == "hidden_trigger" || key == "custom_trigger_tooltip":
			continue
		case key == "limit" && i > 0 && dlcEffectContainers[dlcKey(stack[i-1].entry)]:
			limitIdx = i
		}
		break
	}
	hit.Negated = notCount%2 == 1

	if limitIdx > 0 {
		container := stack[limitIdx-1]
		hit.Context = dlcKey(container.entry)
		hit.Guarded = dlcFirstLevelEffects(container.block)
		return hit, true
	}

	for i := len(stack) - 1; i >= 0; i-- {
		key := dlcKey(stack[i].entry)
		if key == "" || key == "NOT" || key == "AND" || key == "OR" {
			continue
		}
		ctx := key
		if i > 0 {
			owner := dlcKey(stack[i-1].entry)
			if owner != "" {
				ctx += " of " + owner
				if id := dlcBlockID(stack[i-1].block); id != "" {
					ctx += " id=" + id
				}
			}
		}
		hit.Context = ctx
		return hit, true
	}
	hit.Context = "(top level)"
	return hit, true
}

func dlcFirstLevelEffects(block *paradox.Block) []string {
	if block == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	extra := 0
	for _, e := range block.Entries {
		key := dlcKey(e)
		if key == "" || dlcSkipKeysInIf[key] || seen[key] {
			continue
		}
		seen[key] = true
		if len(out) >= dlcMaxGuardedKeys {
			extra++
			continue
		}
		out = append(out, key)
	}
	if extra > 0 {
		out = append(out, fmt.Sprintf("+%d more", extra))
	}
	return out
}

func dlcBlockID(block *paradox.Block) string {
	if block == nil {
		return ""
	}
	for _, e := range block.Entries {
		if dlcKey(e) == "id" && e.Block == nil {
			return e.Scalar.StrValue()
		}
	}
	return ""
}

func dlcRelPath(path, root string) string {
	p := strings.ReplaceAll(path, "\\", "/")
	if root != "" && strings.HasPrefix(p, root) {
		return p[len(root):]
	}
	return p
}

func dlcDirOf(rel string) string {
	parts := strings.Split(rel, "/")
	if len(parts) >= 3 {
		return parts[0] + "/" + parts[1]
	}
	if len(parts) == 2 {
		return parts[0]
	}
	return "."
}

func dlcDirRank(dir string) int {
	if dir == dlcHistoryStatesDir {
		return 0
	}
	return 1
}

func (r *DLCReport) Total() int {
	if r == nil {
		return 0
	}
	return len(r.Hits)
}

func (r *DLCReport) DLCCount() int {
	if r == nil {
		return 0
	}
	return len(r.ByDLC)
}

func (r *DLCReport) UnknownNames() []string {
	if r == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, h := range r.Hits {
		if !h.Known && !seen[h.DLC] {
			seen[h.DLC] = true
			out = append(out, h.DLC)
		}
	}
	sort.Strings(out)
	return out
}

func (r *DLCReport) HistoryStatesCount() int {
	if r == nil {
		return 0
	}
	return r.ByDir[dlcHistoryStatesDir]
}

func (r *DLCReport) sortedDLCs() []string {
	names := make([]string, 0, len(r.ByDLC))
	for n := range r.ByDLC {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if r.ByDLC[names[i]] != r.ByDLC[names[j]] {
			return r.ByDLC[names[i]] > r.ByDLC[names[j]]
		}
		return names[i] < names[j]
	})
	return names
}

func (r *DLCReport) sortedDirs() []string {
	dirs := make([]string, 0, len(r.ByDir))
	for d := range r.ByDir {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool {
		if dlcDirRank(dirs[i]) != dlcDirRank(dirs[j]) {
			return dlcDirRank(dirs[i]) < dlcDirRank(dirs[j])
		}
		return dirs[i] < dirs[j]
	})
	return dirs
}

func (r *DLCReport) Headline() string {
	if r == nil || len(r.Hits) == 0 {
		return ""
	}
	names := r.sortedDLCs()
	shown := names
	ellipsis := ""
	if len(shown) > 3 {
		shown = shown[:3]
		ellipsis = "…"
	}
	head := r.msg.Text("dlc.headline", exportmsg.Args{
		"blocks": len(r.Hits), "count": len(names), "names": strings.Join(shown, ", ") + ellipsis,
		"log": DLCConditionalLogName, "dir": dlcHistoryStatesDir, "dir_count": r.HistoryStatesCount(),
	})
	if unknown := r.UnknownNames(); len(unknown) > 0 {
		head += " " + r.msg.Text("dlc.headline_unknown", exportmsg.Args{"count": len(unknown), "names": strings.Join(unknown, ", ")})
	}
	return head
}

func (r *DLCReport) LogText() string {
	msg := r.msg
	unknown := "  [" + msg.Text("dlc.log.unknown", nil) + "]"
	lines := commentLines(msg.Text("dlc.log.header", nil))
	lines = append(lines, "#", "# "+r.Headline(), "#")
	lines = append(lines, "# "+msg.Text("dlc.log.by_dlc", nil))
	for _, name := range r.sortedDLCs() {
		tag := ""
		if !vanilladata.IsKnownDLCName(name) {
			tag = unknown
			for _, h := range r.Hits {
				if h.DLC == name && h.Suggest != "" {
					tag += " " + msg.Text("dlc.log.did_you_mean", exportmsg.Args{"name": h.Suggest})
					break
				}
			}
		}
		lines = append(lines, fmt.Sprintf("#   %6d  %s%s", r.ByDLC[name], name, tag))
	}
	lines = append(lines, "# "+msg.Text("dlc.log.by_folder", nil))
	for _, dir := range r.sortedDirs() {
		lines = append(lines, fmt.Sprintf("#   %6d  %s", r.ByDir[dir], dir))
	}
	if len(r.ParseFailed) > 0 {
		lines = append(lines, "# "+msg.Text("dlc.log.unreadable", nil))
		for _, p := range r.ParseFailed {
			lines = append(lines, "#   "+p)
		}
	}
	lines = append(lines, "")
	for _, name := range r.sortedDLCs() {
		known := ""
		if !vanilladata.IsKnownDLCName(name) {
			known = unknown
		}
		lines = append(lines, fmt.Sprintf("## %s  (%d)%s", name, r.ByDLC[name], known))
		for _, h := range r.Hits {
			if h.DLC != name {
				continue
			}
			lines = append(lines, dlcHitLine(h, msg))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n") + "\n"
}

func dlcHitLine(h DLCHit, msg exportmsg.Printer) string {
	neg := ""
	if h.Negated {
		neg = "NOT "
	}
	guard := ""
	if len(h.Guarded) > 0 {
		guard = "  -> " + strings.Join(h.Guarded, ", ")
	} else if h.Context == "if" || h.Context == "else_if" {
		guard = "  -> (" + msg.Text("dlc.log.no_effects", nil) + ")"
	}
	return fmt.Sprintf("%s:%d  %shas_dlc \"%s\"  %s%s", h.Path, h.Line, neg, h.DLC, h.Context, guard)
}

func DLCConditionalLogEntry(modDir string, report *DLCReport) Entry {
	if modDir == "" || report == nil || report.Total() == 0 {
		return Entry{}
	}
	return Entry{
		Path:         modDir + "/" + DLCConditionalLogName,
		Bytes:        []byte(report.LogText()),
		Origin:       "generated",
		SourceEntity: "dlc_conditional_lint",
	}
}
