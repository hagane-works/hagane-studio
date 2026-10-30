package focus

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/gamefiles"
	"haganeworks/internal/hoi4/clausewitz"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/hoi4/vanilladata"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

type LintIssue map[string]any

func newIssue(level, msg, fix, kind, id, field string) LintIssue {
	return LintIssue{
		"level": level, "msg": msg, "suggested_fix": fix,
		"entity_kind": kind, "entity_id": id, "field": field,
	}
}

func emptyOrWhitespace(s string) bool { return strings.TrimSpace(s) == "" }

type prereqGraph struct {
	order []string
	edges map[string][]string
	seen  map[string]map[string]bool
}

func newPrereqGraph() *prereqGraph {
	return &prereqGraph{edges: map[string][]string{}, seen: map[string]map[string]bool{}}
}

func (g *prereqGraph) addNode(id string) {
	if _, ok := g.edges[id]; !ok {
		g.order = append(g.order, id)
		g.edges[id] = nil
		g.seen[id] = map[string]bool{}
	}
}

func (g *prereqGraph) addEdge(node, prereq string) {
	g.addNode(node)
	if g.seen[node][prereq] {
		return
	}
	g.seen[node][prereq] = true
	g.edges[node] = append(g.edges[node], prereq)
}

func (g *prereqGraph) cycles() [][]string {
	const white, gray, black = 0, 1, 2
	color := make(map[string]int, len(g.order))
	for _, n := range g.order {
		color[n] = white
	}
	var out [][]string
	seenSets := map[string]bool{}
	record := func(cycle []string) {
		set := map[string]bool{}
		for _, n := range cycle {
			set[n] = true
		}
		keys := make([]string, 0, len(set))
		for k := range set {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		key := strings.Join(keys, "\x00")
		if seenSets[key] {
			return
		}
		seenSets[key] = true
		out = append(out, cycle)
	}
	var dfs func(node string, stack []string) []string
	dfs = func(node string, stack []string) []string {
		color[node] = gray
		stack = append(stack, node)
		for _, next := range g.edges[node] {
			if _, known := color[next]; !known {
				continue
			}
			if color[next] == gray {
				for i, s := range stack {
					if s == next {
						record(append(append([]string{}, stack[i:]...), next))
						break
					}
				}
			} else if color[next] == white {
				stack = dfs(next, stack)
			}
		}
		stack = stack[:len(stack)-1]
		color[node] = black
		return stack
	}
	for _, n := range g.order {
		if color[n] == white {
			dfs(n, nil)
		}
	}
	return out
}

type tokenLegalityFinding struct {
	Code, Token, Detail string
}

var techCategoryKeys = map[string]bool{"add_tech_bonus": true, "has_tech_bonus": true}

var effectTypoAffixes = []string{"add_", "remove_", "set_", "clr_", "reset_", "modify_"}

func effectTypoSuggestion(key string) string {
	for _, affix := range effectTypoAffixes {
		if strings.HasPrefix(key, affix) {
			base := key[len(affix):]
			if vanilladata.IsVanillaEffect(base) || vanilladata.IsVanillaTrigger(base) {
				return base
			}
		}
	}
	return ""
}

func scanTokenLegality(text string, effectContext bool, categories map[string]bool) []tokenLegalityFinding {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	block, recoveries, err := paradox.Parse(text)
	if err != nil || block == nil {
		return nil
	}
	for _, rec := range recoveries {
		if rec.Code == paradox.RecoveryUnclosedBlock {
			return nil
		}
	}
	known := func(key string) bool { return vanilladata.IsVanillaEffect(key) || vanilladata.IsVanillaTrigger(key) }
	var findings []tokenLegalityFinding
	var walk func(entries []*paradox.Entry)
	walk = func(entries []*paradox.Entry) {
		for _, e := range entries {
			if e.Key != nil {
				key := *e.Key
				if effectContext && key == "research_bonus" && !known(key) {
					findings = append(findings, tokenLegalityFinding{
						Code: "research_bonus_as_effect", Token: key,
						Detail: "research_bonus is not a HOI4 effect — use " +
							"add_tech_bonus = { bonus = N uses = N category = " +
							"<tech_category> } to grant research speed in a " +
							"focus/decision effect",
					})
				}
				if techCategoryKeys[key] && e.Block != nil && categories != nil {
					for _, child := range e.Block.Entries {
						if child.Key == nil || *child.Key != "category" || child.Block != nil {
							continue
						}
						c := strings.TrimSpace(child.Scalar.StrValue())
						if c != "" && !categories[c] {
							findings = append(findings, tokenLegalityFinding{
								Code: "unknown_tech_category", Token: c,
								Detail: fmt.Sprintf("'%s' is not a vanilla technology category "+
									"(e.g. construction_tech, industry, electronics) — fix the "+
									"typo or define a custom category in common/technology_tags/", c),
							})
						}
					}
				}
				if effectContext && !known(key) {
					if base := effectTypoSuggestion(key); base != "" {
						findings = append(findings, tokenLegalityFinding{
							Code: "effect_typo", Token: key,
							Detail: fmt.Sprintf("'%s' is not a HOI4 effect — did you mean "+
								"'%s'? (the program recognises '%s' but not '%s')", key, base, base, key),
						})
					}
				}
			}
			if e.Block != nil {
				walk(e.Block.Entries)
			}
		}
	}
	walk(block.Entries)
	return findings
}

var tokenLintMsg = map[string][2]string{
	"research_bonus_as_effect": {
		"research_bonus 不是 effect",
		"改用 add_tech_bonus = { bonus = N uses = N category = <技术类别> }",
	},
	"unknown_tech_category": {
		"技术类别不存在",
		"改成真类(construction_tech / industry / electronics 等),或在 common/technology_tags/ 定义自定义类别",
	},
}

const rawContentLiteralNewline = "raw_content.literal_escape_newline"

var parserFindingMsg = map[string][2]string{
	"clausewitz.unclosed_block":      {"unclosed `{` — everything after this line is lost", "Close the block; if the `}` sits after a `#` on the same line, move the comment to its own line"},
	"clausewitz.stray_closing_brace": {"stray `}` with no matching `{`", "Remove the extra `}` or add the missing `{`"},
	"clausewitz.unterminated_quote":  {"unterminated quote", "Close the string"},
	rawContentLiteralNewline:         {"literal backslash-n in script text (HOI4 prints the two characters verbatim)", "Use a real line break, or move the text into a localisation yml value"},
}

func parserFindingToLintIssue(f clausewitz.AnalysisFinding, kind, id, field string) LintIssue {
	level := f.Severity
	if level == "blocker" {
		level = "error"
	}
	if level == "" {
		level = "warning"
	}
	head, fix := f.Code, "fix per the finding's evidence"
	if m, ok := parserFindingMsg[f.Code]; ok {
		head, fix = m[0], m[1]
	}
	var span any
	where := ""
	if f.Source != nil {
		if sp, ok := f.Source["span"].(map[string]any); ok {
			span = sp
			if line, ok := sp["start_line"]; ok {
				where = fmt.Sprintf(" (line %v)", line)
			}
		} else if sp, ok := f.Source["span"].(clausewitz.SourceSpan); ok {
			span = sp
			where = fmt.Sprintf(" (line %v)", sp.StartLine)
		}
	}
	evidence := f.Evidence
	if evidence == nil {
		evidence = map[string]any{}
	}
	return LintIssue{
		"level": level, "code": f.Code,
		"msg":           fmt.Sprintf("[%s] %s%s: %s", id, field, where, head),
		"suggested_fix": fix,
		"entity_kind":   kind, "entity_id": id, "field": field,
		"source_span": span, "evidence": evidence, "repair_route": f.RepairRoute,
	}
}

func lineExcerpt(text []rune, offset, width int) string {
	lineStart := 0
	for i := offset - 1; i >= 0; i-- {
		if text[i] == '\n' {
			lineStart = i + 1
			break
		}
	}
	lineEnd := len(text)
	for i := offset; i < len(text); i++ {
		if text[i] == '\n' {
			lineEnd = i
			break
		}
	}
	line := text[lineStart:lineEnd]
	if len(line) <= width {
		return string(line)
	}
	start := offset - lineStart - width/2
	if start > len(line)-width {
		start = len(line) - width
	}
	if start < 0 {
		start = 0
	}
	prefix := ""
	if start > 0 {
		prefix = "…"
	}
	return prefix + string(line[start:start+width])
}

func literalNewlineFindings(src string, rootPath []string) []clausewitz.AnalysisFinding {
	text := []rune(src)
	findings := []clausewitz.AnalysisFinding{}
	inComment := false
	for i := 0; i < len(text)-1; i++ {
		c := text[i]
		switch {
		case c == '\n':
			inComment = false
		case c == '#' && !inComment:
			inComment = true
		case !inComment && c == '\\' && text[i+1] == 'n':
			span := clausewitz.Span(text, i, i+2)
			findings = append(findings, clausewitz.AnalysisFinding{
				Code: rawContentLiteralNewline, Severity: "warning",
				Source: map[string]any{"file": nil, "field_path": append([]string{}, rootPath...), "span": span},
				Evidence: map[string]any{
					"line": span.StartLine, "source_excerpt": lineExcerpt(text, i, 200),
					"reason": "HOI4 script does not interpret backslash escapes; the two characters '\\n' " +
						"are emitted verbatim. Use a real line break, or move the text into a localisation " +
						"yml value where \\n is a newline.",
				},
				RepairRoute: "raw_content.replace_literal_newline",
			})
			if len(findings) >= 20 {
				return findings
			}
			i++
		}
	}
	return findings
}

type treeLint struct {
	categories map[string]bool
}

func (c *treeLint) emitTokenLegality(text, kind, id, field string, out *[]LintIssue) {
	if text == "" {
		return
	}
	for _, f := range scanTokenLegality(text, true, c.categories) {
		head, fix := f.Detail, f.Detail
		if m, ok := tokenLintMsg[f.Code]; ok {
			head, fix = m[0], m[1]
		}
		*out = append(*out, newIssue("warning", fmt.Sprintf("[%s] %s: %s · %s", id, field, f.Token, head), fix, kind, id, field))
	}
	rep, err := clausewitz.AnalyzeScriptPositions(text, []string{kind, id, field}, nil, nil)
	if err != nil {
		return
	}
	for _, f := range rep.Findings {
		*out = append(*out, LintIssue(f.ToLintIssue(kind, id, field)))
	}
	for _, f := range rep.ParserFindings {
		*out = append(*out, parserFindingToLintIssue(f, kind, id, field))
	}
	for _, f := range literalNewlineFindings(text, []string{kind, id, field}) {
		*out = append(*out, parserFindingToLintIssue(f, kind, id, field))
	}
}

type gameLintData struct {
	sprites    map[string]bool
	categories map[string]bool
}

type gameLintKey struct{}

func loadGameLintData(game *gamefiles.Game) *gameLintData {
	return gamefiles.Derived(game, gameLintKey{}, readGameLintData)
}

func readGameLintData(game *gamefiles.Game) *gameLintData {
	data := &gameLintData{sprites: map[string]bool{}, categories: map[string]bool{}}
	for name := range game.Sprites() {
		data.sprites[name] = true
	}
	for _, file := range game.Files("common/technology_tags", false, ".txt") {
		raw, err := game.ReadFile(file)
		if err != nil {
			continue
		}
		text, ok := gfx.DecodeScriptText(raw)
		if !ok {
			continue
		}
		block, _, err := paradox.Parse(text)
		if err != nil || block == nil {
			continue
		}
		for _, entry := range block.Entries {
			if entry.Key == nil || *entry.Key != "technology_categories" || entry.Block == nil {
				continue
			}
			for _, item := range entry.Block.Entries {
				if item.Key == nil && item.Block == nil {
					data.categories[item.Scalar.StrValue()] = true
				}
			}
		}
	}
	if len(data.categories) == 0 {
		data.categories = nil
	}
	return data
}

func (h *handlers) lintProject(p *project.Project) (map[string]any, error) {
	record := p.Record()
	isSubmod := len(workbench.StringList(record["dependencies"])) > 0
	if deps, ok := record["dependencies"].(string); ok && strings.TrimSpace(deps) != "" {
		isSubmod = true
	}
	var game *gameLintData
	if g, err := gamefiles.Open(h.deps.Game); err == nil {
		game = loadGameLintData(g)
	}
	c := &treeLint{}
	if game != nil {
		c.categories = game.categories
	}

	trees, err := p.Rows("focus_trees")
	if err != nil {
		return nil, err
	}
	treeIDs := map[int64]bool{}
	for _, tree := range trees {
		treeIDs[project.Int64(tree["id"])] = true
	}
	nodeRows, err := p.Rows("focus_nodes")
	if err != nil {
		return nil, err
	}

	sort.SliceStable(nodeRows, func(i, j int) bool {
		ti, tj := project.Int64(nodeRows[i]["focus_tree_id"]), project.Int64(nodeRows[j]["focus_tree_id"])
		if ti != tj {
			return ti < tj
		}
		return project.Int64(nodeRows[i]["id"]) < project.Int64(nodeRows[j]["id"])
	})
	var nodes []FocusNode
	for _, row := range nodeRows {
		if treeIDs[project.Int64(row["focus_tree_id"])] {
			nodes = append(nodes, nodeFromRow(row, ""))
		}
	}

	focusIssues, decisionIssues, eventIssues, ideaIssues := []LintIssue{}, []LintIssue{}, []LintIssue{}, []LintIssue{}

	allFocusIDs := map[string]bool{}
	for _, n := range nodes {
		if n.FocusID != "" {
			allFocusIDs[n.FocusID] = true
		}
	}
	graph := newPrereqGraph()
	for _, n := range nodes {
		fid := n.FocusID
		if fid == "" {
			fid = "(unnamed)"
		}
		for _, group := range n.Prerequisites {
			for _, pid := range group {
				if pid != "" && !allFocusIDs[pid] {
					focusIssues = append(focusIssues, newIssue("error",
						fmt.Sprintf("[%s] prereq focus_id '%s' 不存在", fid, pid),
						"删除该 prereq 或先创建 focus", "focus", fid, "prerequisites"))
				}
			}
		}
		for _, mid := range n.MutuallyExclusive {
			if mid != "" && !allFocusIDs[mid] {
				focusIssues = append(focusIssues, newIssue("error",
					fmt.Sprintf("[%s] mutually_exclusive '%s' 不存在", fid, mid),
					"删除该 mutex 或先创建 focus", "focus", fid, "mutually_exclusive"))
			}
		}
		if emptyOrWhitespace(n.CompletionReward) {
			focusIssues = append(focusIssues, newIssue("warning",
				fmt.Sprintf("[%s] completion_reward 为空 · 完成后无效果", fid),
				"加 add_political_power / add_stability / 等 effect", "focus", fid, "completion_reward"))
		}
		if emptyOrWhitespace(n.Icon) {
			focusIssues = append(focusIssues, newIssue("info",
				fmt.Sprintf("[%s] icon 字段为空 · 用 vanilla 默认图标", fid),
				"上传 focus icon 或选用 GFX_focus_generic_*", "focus", fid, "icon"))
		}
		if n.FocusID != "" {
			graph.addNode(n.FocusID)
			for _, group := range n.Prerequisites {
				for _, pid := range group {
					if pid != "" {
						graph.addEdge(n.FocusID, pid)
					}
				}
			}
		}
	}
	for _, cycle := range graph.cycles() {
		head := cycle[0]
		chain := strings.Join(cycle, " → ")
		msg := fmt.Sprintf("[%s] prerequisites 成环 · 环内 focus 永远无法解锁 · %s", head, chain)
		fix := "断开环路:删除环中任一 prereq 边"
		if len(cycle) == 2 && cycle[0] == cycle[1] {
			msg = fmt.Sprintf("[%s] prerequisites 自引用(focus 不能依赖自己)· %s", head, chain)
			fix = "删除指向自身的 prereq"
		}
		focusIssues = append(focusIssues, newIssue("error", msg, fix, "focus", head, "prerequisites"))
	}

	for _, n := range nodes {
		fid := n.FocusID
		if fid == "" {
			fid = "(unnamed)"
		}
		fields := map[string]string{
			"completion_reward": n.CompletionReward, "available": n.Available, "bypass": n.Bypass,
			"select_effect": n.SelectEffect, "cancel": n.Cancel,
		}
		for _, f := range []string{"completion_reward", "available", "bypass", "select_effect", "cancel"} {
			c.emitTokenLegality(fields[f], "focus", fid, f, &focusIssues)
		}
	}

	decisions, err := p.Rows("decisions")
	if err != nil {
		return nil, err
	}
	for _, d := range sortByID(decisions) {
		did := colStr(d, "decision_id", "")
		if did == "" {
			did = "(unnamed)"
		}
		for _, f := range []string{"complete_effect", "remove_effect", "cancel_effect", "allowed", "available", "visible", "ai_will_do", "cancel_trigger"} {
			c.emitTokenLegality(colStr(d, f, ""), "decision", did, f, &decisionIssues)
		}
		if !colBool(d, "selectable_mission", false) && emptyOrWhitespace(colStr(d, "complete_effect", "")) {
			decisionIssues = append(decisionIssues, newIssue("warning",
				fmt.Sprintf("[%s] complete_effect 为空 · decision 触发后无效果", did),
				"加 effect 块(或标 selectable_mission=True 作为 mission)", "decision", did, "complete_effect"))
		}
	}

	events, err := p.Rows("events")
	if err != nil {
		return nil, err
	}
	events = sortByID(events)
	for _, e := range events {
		eid := colStr(e, "event_id", "")
		if eid == "" {
			eid = "(unnamed)"
		}
		for _, f := range []string{"trigger", "immediate", "mean_time_to_happen", "after"} {
			c.emitTokenLegality(colStr(e, f, ""), "event", eid, f, &eventIssues)
		}
		var options []map[string]any
		list, _ := e["options"].([]any)
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				options = append(options, m)
			}
		}
		for i, o := range options {
			for _, of := range []string{"trigger", "effects", "hidden_effects"} {
				text, _ := o[of].(string)
				c.emitTokenLegality(text, "event", eid, fmt.Sprintf("options[%d].%s", i, of), &eventIssues)
			}
		}
		if len(options) == 0 && !colBool(e, "fire_only_once", false) {
			eventIssues = append(eventIssues, newIssue("info",
				fmt.Sprintf("[%s] event 0 option · 玩家无法点击 · 多 reload 才能消失", eid),
				"加至少 1 option · 或确认是 hidden=yes / fire_only_once=yes 的 trigger event", "event", eid, "options"))
		}
	}

	ideas, err := p.Rows("ideas")
	if err != nil {
		return nil, err
	}
	ideas = sortByID(ideas)
	for _, idea := range ideas {
		iid := colStr(idea, "idea_id", "")
		if iid == "" {
			iid = "(unnamed)"
		}
		for _, f := range []string{"ai_will_do", "allowed", "allowed_civil_war", "available", "visible", "allowed_to_remove", "cancel", "do_effect", "on_add", "on_remove"} {
			c.emitTokenLegality(colStr(idea, f, ""), "idea", iid, f, &ideaIssues)
		}
	}

	if !isSubmod && game != nil && len(game.sprites) > 0 {
		projectIcons, err := projectFocusIcons(p)
		if err != nil {
			return nil, err
		}
		projFocusIcons := map[string]bool{}
		for _, ref := range projectIcons {
			projFocusIcons[ref.Name] = true
		}
		assets, err := p.Rows("project_obs_assets")
		if err != nil {
			return nil, err
		}
		projEventPics, projIdeaPics := map[string]bool{}, map[string]bool{}
		for _, a := range assets {
			switch colStr(a, "asset_type", "") {
			case "event_picture":
				projEventPics[colStr(a, "gfx_name", "")] = true
			case "idea_picture":
				projIdeaPics[colStr(a, "gfx_name", "")] = true
			}
		}
		paths, err := p.Rows("entity_obs_paths")
		if err != nil {
			return nil, err
		}
		entityObsKeys := map[string]bool{}
		for _, row := range paths {
			entityObsKeys[fmt.Sprintf("%s:%d", colStr(row, "entity_type", ""), project.Int64(row["entity_id"]))] = true
		}

		for _, n := range nodes {
			fid := n.FocusID
			if fid == "" {
				fid = "(unnamed)"
			}
			derived := ""
			if n.FocusID != "" {
				derived = emitmisc.FocusIconSpriteName(n.FocusID)
			}
			type check struct{ name, field string }
			var checks []check
			if base := strings.TrimSpace(n.Icon); base != "" {
				checks = append(checks, check{base, "icon"})
			}
			for _, entry := range n.DynamicIcon {
				if icon, ok := entry["icon"].(string); ok && strings.TrimSpace(icon) != "" {
					checks = append(checks, check{strings.TrimSpace(icon), "dynamic_icon"})
				}
			}
			if ti := strings.TrimSpace(n.TextIcon); ti != "" {
				checks = append(checks, check{ti, "text_icon"})
			}
			for _, ck := range checks {

				if ck.name == derived || game.sprites[ck.name] || projFocusIcons[ck.name] {
					continue
				}
				focusIssues = append(focusIssues, newIssue("warning",
					fmt.Sprintf("[%s] %s: GFX '%s' 非 vanilla 图标也未上传到项目 · 导出会丢该引用(破图)", fid, ck.field, ck.name),
					"上传该 focus icon 到项目库,或改用 vanilla GFX_goal_* / GFX_focus_*", "focus", fid, ck.field))
			}
		}
		for _, e := range events {
			eid := colStr(e, "event_id", "")
			if eid == "" {
				eid = "(unnamed)"
			}
			pic := strings.TrimSpace(colStr(e, "picture", ""))
			if pic == "" || game.sprites[pic] || projEventPics[pic] || entityObsKeys[fmt.Sprintf("event_picture:%d", project.Int64(e["id"]))] {
				continue
			}
			eventIssues = append(eventIssues, newIssue("warning",
				fmt.Sprintf("[%s] picture: GFX '%s' 非 vanilla event picture 也未上传 · 导出会丢该引用(破图)", eid, pic),
				"上传 event picture,或改用 vanilla GFX_news_event_* / GFX_report_event_*", "event", eid, "picture"))
		}
		for _, idea := range ideas {
			iid := colStr(idea, "idea_id", "")
			if iid == "" {
				iid = "(unnamed)"
			}
			pic := strings.TrimSpace(colStr(idea, "picture", ""))
			if pic == "" {
				continue
			}
			stripped := strings.TrimPrefix(pic, "GFX_idea_")
			if game.sprites["GFX_idea_"+stripped] || game.sprites[pic] || projIdeaPics[pic] || projIdeaPics[stripped] ||
				entityObsKeys[fmt.Sprintf("idea_picture:%d", project.Int64(idea["id"]))] {
				continue
			}
			ideaIssues = append(ideaIssues, newIssue("warning",
				fmt.Sprintf("[%s] picture: GFX '%s' 非 vanilla idea picture 也未上传 · 导出会丢该引用(破图)", iid, pic),
				"上传 idea picture,或改用 vanilla idea 图(去掉 GFX_idea_ 前缀后须在 vanilla 索引内)", "idea", iid, "picture"))
		}
	}

	totals := map[string]int{"error": 0, "warning": 0, "info": 0}
	for _, group := range [][]LintIssue{focusIssues, decisionIssues, eventIssues, ideaIssues} {
		for _, it := range group {
			level, _ := it["level"].(string)
			totals[level]++
		}
	}
	return map[string]any{
		"focus": focusIssues, "decision": decisionIssues,
		"event": eventIssues, "idea": ideaIssues, "totals": totals,
	}, nil
}

func (h *handlers) treeValidation(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	report, err := h.lintProject(p)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, report)
	return nil
}
