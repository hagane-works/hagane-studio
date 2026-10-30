package exportengine

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"haganeworks/internal/gameindex"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

const (
	crossRefLogName = "_CROSS_REF_SKIPPED.log"
	tokenLogName    = "_TOKEN_LINT.log"
)

type LintReport struct {
	CrossRef []string
	Token    []string

	Scripted []ScriptedRef
	msg      exportmsg.Printer
}

func (r LintReport) ScriptedHeadline() string {
	return scriptedHeadline(r.Scripted, r.msg)
}

type refTarget struct {
	id   string
	line int
}

type refExtractor func(entry *paradox.Entry) []refTarget

type crossRefSpec struct {
	targetType string
	extract    refExtractor
}

func scalarRef(entry *paradox.Entry) []refTarget {
	if entry.Block != nil {
		return nil
	}
	return []refTarget{{id: entry.Scalar.StrValue(), line: entry.Line}}
}

func blockField(field string) refExtractor {
	return func(entry *paradox.Entry) []refTarget {
		return fieldsInBlock(entry, field)
	}
}

func scalarOrBlockField(field string) refExtractor {
	return func(entry *paradox.Entry) []refTarget {
		if entry.Block == nil {
			return []refTarget{{id: entry.Scalar.StrValue(), line: entry.Line}}
		}
		return fieldsInBlock(entry, field)
	}
}

func blockAnyField(fields ...string) refExtractor {
	return func(entry *paradox.Entry) []refTarget {
		var out []refTarget
		for _, field := range fields {
			out = append(out, fieldsInBlock(entry, field)...)
		}
		return out
	}
}

func scalarOrList(entry *paradox.Entry) []refTarget {
	if entry.Block == nil {
		return []refTarget{{id: entry.Scalar.StrValue(), line: entry.Line}}
	}
	var out []refTarget
	for _, sub := range entry.Block.Entries {
		if sub.Key == nil && sub.Block == nil {
			out = append(out, refTarget{id: sub.Scalar.StrValue(), line: sub.Line})
		}
	}
	return out
}

func eventScalarOrBlock(entry *paradox.Entry) []refTarget {
	if entry.Block == nil {
		return []refTarget{{id: entry.Scalar.StrValue(), line: entry.Line}}
	}
	return fieldsInBlock(entry, "id")
}

func blockKeys(entry *paradox.Entry) []refTarget {
	if entry.Block == nil {
		return nil
	}
	var out []refTarget
	for _, sub := range entry.Block.Entries {
		if sub.Key == nil || *sub.Key == "popup" {
			continue
		}
		out = append(out, refTarget{id: *sub.Key, line: sub.Line})
	}
	return out
}

func countryLeaderTraits(entry *paradox.Entry) []refTarget {
	if entry.Block == nil {
		return nil
	}
	var out []refTarget
	for _, sub := range entry.Block.Entries {
		if sub.Key == nil || *sub.Key != "traits" || sub.Block == nil {
			continue
		}
		for _, leaf := range sub.Block.Entries {
			if leaf.Key == nil && leaf.Block == nil {
				out = append(out, refTarget{id: leaf.Scalar.StrValue(), line: leaf.Line})
			}
		}
	}
	return out
}

func fieldsInBlock(entry *paradox.Entry, field string) []refTarget {
	if entry.Block == nil {
		return nil
	}
	var out []refTarget
	for _, sub := range entry.Block.Entries {
		if sub.Key == nil || *sub.Key != field || sub.Block != nil {
			continue
		}
		out = append(out, refTarget{id: sub.Scalar.StrValue(), line: sub.Line})
	}
	return out
}

var crossRefSpecs = map[string][]crossRefSpec{

	"add_ideas":      {{"idea", scalarOrList}},
	"remove_ideas":   {{"idea", scalarOrList}},
	"swap_ideas":     {{"idea", blockAnyField("add_idea", "remove_idea", "add", "remove")}},
	"add_timed_idea": {{"idea", blockField("idea")}},
	"has_idea":       {{"idea", scalarRef}},

	"add_dynamic_modifier":    {{"dynamic_modifier", blockField("modifier")}},
	"remove_dynamic_modifier": {{"dynamic_modifier", blockField("modifier")}},
	"has_dynamic_modifier":    {{"dynamic_modifier", scalarOrBlockField("modifier")}},

	"set_autonomy": {{
		"autonomy_state", blockAnyField("autonomy_state", "autonomous_state"),
	}},
	"has_autonomy_state": {{"autonomy_state", scalarRef}},

	"country_event":     {{"event", eventScalarOrBlock}},
	"state_event":       {{"event", eventScalarOrBlock}},
	"news_event":        {{"event", eventScalarOrBlock}},
	"unit_leader_event": {{"event", eventScalarOrBlock}},

	"complete_national_focus": {{"focus", scalarOrBlockField("focus")}},
	"has_completed_focus":     {{"focus", scalarRef}},
	"focus_progress":          {{"focus", blockField("focus")}},

	"activate_decision":       {{"decision", scalarRef}},
	"unlock_decision_tooltip": {{"decision", scalarOrBlockField("decision")}},

	"has_trait":                     {{"trait", scalarRef}},
	"has_country_leader_with_trait": {{"trait", scalarRef}},
	"add_country_leader_trait":      {{"trait", scalarOrBlockField("trait")}},
	"remove_country_leader_trait":   {{"trait", scalarOrBlockField("trait")}},
	"add_unit_leader_trait":         {{"trait", scalarRef}},
	"remove_unit_leader_trait":      {{"trait", scalarRef}},
	"swap_ruler_traits":             {{"trait", blockAnyField("add", "remove")}},
	"create_country_leader":         {{"trait", countryLeaderTraits}},

	"add_opinion_modifier":         {{"opinion_modifier", blockField("modifier")}},
	"remove_opinion_modifier":      {{"opinion_modifier", blockField("modifier")}},
	"reverse_add_opinion_modifier": {{"opinion_modifier", blockField("modifier")}},
	"has_opinion_modifier":         {{"opinion_modifier", scalarOrBlockField("modifier")}},

	"has_tech":       {{"technology", scalarRef}},
	"can_research":   {{"technology", scalarRef}},
	"set_technology": {{"technology", blockKeys}},

	"has_character":              {{"character", scalarRef}},
	"is_character":               {{"character", scalarRef}},
	"retire_character":           {{"character", scalarRef}},
	"promote_character":          {{"character", scalarOrBlockField("character")}},
	"remove_country_leader_role": {{"character", scalarOrBlockField("character")}},
	"remove_unit_leader_role":    {{"character", scalarOrBlockField("character")}},
	"add_country_leader_role":    {{"character", blockField("character")}},
	"add_corps_commander_role":   {{"character", blockField("character")}},

	"add_trait": {
		{"character", blockField("character")},
		{"trait", blockField("trait")},
	},

	"add_mio_funds":                               {{"mio", blockField("mio")}},
	"add_mio_size_up":                             {{"mio", blockField("mio")}},
	"add_mio_design_team_assign_cost":             {{"mio", blockField("mio")}},
	"add_mio_industrial_manufacturer_assign_cost": {{"mio", blockField("mio")}},
}

func snapshotIDSet(snapshot Snapshot, table, field string) map[string]bool {
	out := map[string]bool{}
	for _, raw := range snapshot.Tables[table] {
		row, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			continue
		}
		if id := row.Str(field); id != "" {
			out[id] = true
		}
	}
	return out
}

type crossRefVanillaKey struct{}

func loadCrossRefVanillaIDs() map[string]map[string]bool {
	sets, _ := crossRefVanilla()
	return sets
}

func crossRefVanilla() (map[string]map[string]bool, bool) {
	sets, err := gameindex.Load(context.Background(), crossRefVanillaKey{}, gameindex.CrossRefIDs, func(raw []byte) (map[string]map[string]bool, error) {
		var table struct {
			IDs map[string][]string `json:"ids"`
		}
		if err := json.Unmarshal(raw, &table); err != nil {
			return nil, err
		}
		out := make(map[string]map[string]bool, len(table.IDs)+1)
		for targetType, ids := range table.IDs {
			out[targetType] = stringSetOf(ids)
		}
		return out, nil
	})
	if err != nil {
		return map[string]map[string]bool{}, false
	}
	autonomy, err := gameindex.Load(context.Background(), autonomyStatesKey{}, gameindex.AutonomyStates, func(raw []byte) (map[string]bool, error) {
		var table struct {
			IDs []string `json:"ids"`
		}
		err := json.Unmarshal(raw, &table)
		return stringSetOf(table.IDs), err
	})
	if err == nil {
		withAutonomy := make(map[string]map[string]bool, len(sets)+1)
		for k, v := range sets {
			withAutonomy[k] = v
		}
		withAutonomy["autonomy_state"] = autonomy
		sets = withAutonomy
	}
	return sets, true
}

type autonomyStatesKey struct{}

func stringSetOf(ids []string) map[string]bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id != "" {
			set[id] = true
		}
	}
	return set
}

func crossRefExistenceSets(snapshot Snapshot) map[string]map[string]bool {
	sets := map[string]map[string]bool{

		"focus":    snapshotIDSet(snapshot, "focus_nodes", "focus_id"),
		"decision": snapshotIDSet(snapshot, "decisions", "decision_id"),
		"mio": snapshotIDSet(snapshot, "military_industrial_organizations",
			"mio_id"),

		"idea":             snapshotIDSet(snapshot, "ideas", "idea_id"),
		"event":            snapshotIDSet(snapshot, "events", "event_id"),
		"trait":            snapshotIDSet(snapshot, "custom_traits", "trait_id"),
		"technology":       snapshotIDSet(snapshot, "technologies", "tech_id"),
		"character":        snapshotIDSet(snapshot, "characters", "tag"),
		"opinion_modifier": snapshotIDSet(snapshot, "opinion_modifiers", "modifier_id"),
		"dynamic_modifier": snapshotIDSet(snapshot, "dynamic_modifiers", "dm_id"),
		"autonomy_state":   snapshotIDSet(snapshot, "autonomy_states", "autonomy_id"),
	}

	primaryTag := ""
	if project, err := emitmisc.DecodeEntityJSON(snapshot.Project); err == nil {
		primaryTag = strings.ToLower(project.Str("country_tag"))
	}
	for _, raw := range snapshot.Tables["super_events"] {
		row, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			continue
		}
		tag := strings.ToLower(row.Str("country_tag"))
		if tag == "" {
			tag = primaryTag
		}
		if tag == "" {
			continue
		}
		sets["event"][fmt.Sprintf("%s_se.%d", tag, row.Int("number", 0))] = true
	}
	for targetType, vanilla := range loadCrossRefVanillaIDs() {
		set, ok := sets[targetType]
		if !ok {
			continue
		}
		for id := range vanilla {
			set[id] = true
		}
	}
	return sets
}

func levenshtein(left, right string) int {
	if left == right {
		return 0
	}
	l, r := []rune(left), []rune(right)
	if len(l) == 0 {
		return len(r)
	}
	if len(r) == 0 {
		return len(l)
	}
	previous := make([]int, len(r)+1)
	current := make([]int, len(r)+1)
	for i := range previous {
		previous[i] = i
	}
	for row := 1; row <= len(l); row++ {
		current[0] = row
		for col := 1; col <= len(r); col++ {
			sub := previous[col-1]
			if l[row-1] != r[col-1] {
				sub++
			}
			best := current[col-1] + 1
			if previous[col]+1 < best {
				best = previous[col] + 1
			}
			if sub < best {
				best = sub
			}
			current[col] = best
		}
		previous, current = current, previous
	}
	return previous[len(r)]
}

func typoSuggestion(targetID string, known map[string]bool) bool {
	if utf8.RuneCountInString(targetID) < 4 {
		return false
	}
	want := utf8.RuneCountInString(targetID)
	matches := 0
	for id := range known {
		diff := utf8.RuneCountInString(id) - want
		if diff < -1 || diff > 1 {
			continue
		}
		if levenshtein(targetID, id) == 1 {
			matches++
			if matches > 1 {
				return false
			}
		}
	}
	return matches == 1
}

type crossRefCatalog struct {
	known          map[string]map[string]bool
	completeTypes  map[string]bool
	expectsBaseMod bool
}

func (c crossRefCatalog) classify(targetType, targetID string) string {
	known := c.known[targetType]
	if known[targetID] {
		return "resolved"
	}
	if typoSuggestion(targetID, known) {
		return "typo"
	}

	if c.expectsBaseMod {
		return "base_mod_not_indexed"
	}
	if c.completeTypes[targetType] {
		return "missing"
	}
	return "unknown"
}

type tokenLocator struct {
	text       string
	lineStarts []int
	used       map[int]bool
}

func newTokenLocator(text string) *tokenLocator {
	starts := []int{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return &tokenLocator{text: text, lineStarts: starts, used: map[int]bool{}}
}

func isRefTokenEdge(b byte) bool {
	switch {
	case b >= 'A' && b <= 'Z', b >= 'a' && b <= 'z', b >= '0' && b <= '9':
		return true
	}
	switch b {
	case '_', ':', '.', '@', '-':
		return true
	}
	return false
}

func (l *tokenLocator) offsetOfLine(line int) int {
	if line <= 1 {
		return 0
	}
	if line-1 >= len(l.lineStarts) {
		return len(l.text)
	}
	return l.lineStarts[line-1]
}

func (l *tokenLocator) locate(target string, fromLine int) (int, int, bool) {
	if target == "" {
		return 0, 0, false
	}
	for cursor := l.offsetOfLine(fromLine); cursor <= len(l.text); {
		rel := strings.Index(l.text[cursor:], target)
		if rel < 0 {
			return 0, 0, false
		}
		start := cursor + rel
		end := start + len(target)
		cursor = start + 1
		if start > 0 && isRefTokenEdge(l.text[start-1]) {
			continue
		}
		if end < len(l.text) && isRefTokenEdge(l.text[end]) {
			continue
		}
		if l.used[start] {
			continue
		}
		l.used[start] = true
		line, column := l.position(start)
		return line, column, true
	}
	return 0, 0, false
}

func (l *tokenLocator) position(offset int) (int, int) {
	line := sort.SearchInts(l.lineStarts, offset+1) - 1
	if line < 0 {
		line = 0
	}
	return line + 1, utf8.RuneCountInString(l.text[l.lineStarts[line]:offset]) + 1
}

func walkBlock(block *paradox.Block, visit func(entry *paradox.Entry)) {
	if block == nil {
		return
	}
	for _, entry := range block.Entries {
		visit(entry)
		if entry.Block != nil {
			walkBlock(entry.Block, visit)
		}
	}
}

type techCategoriesKey struct{}

func vanillaTechCategories() (map[string]bool, bool) {
	set, err := gameindex.Load(context.Background(), techCategoriesKey{}, gameindex.TechTags, func(raw []byte) (map[string]bool, error) {
		var tags struct {
			Categories struct {
				IDs []string `json:"ids"`
			} `json:"categories"`
		}
		err := json.Unmarshal(raw, &tags)
		return stringSetOf(tags.Categories.IDs), err
	})
	return set, err == nil && len(set) > 0
}

var techBonusKeys = map[string]bool{"add_tech_bonus": true, "has_tech_bonus": true}

func declaredTechCategories(entries []Entry) map[string]bool {
	out := map[string]bool{}
	for _, entry := range entries {
		lower := strings.ToLower(entry.Path)
		if !strings.Contains(lower, "common/technology_tags/") || !strings.HasSuffix(lower, ".txt") {
			continue
		}
		block, _, err := paradox.Parse(scriptText(entry))
		if err != nil {
			continue
		}
		walkBlock(block, func(e *paradox.Entry) {
			if e.Key == nil || *e.Key != "technology_categories" || e.Block == nil {
				return
			}
			for _, sub := range e.Block.Entries {
				if sub.Key == nil && sub.Block == nil {
					if id := sub.Scalar.StrValue(); id != "" {
						out[id] = true
					}
				}
			}
		})
	}
	return out
}

func scriptText(entry Entry) string {
	return strings.TrimPrefix(string(entry.Bytes), "\uFEFF")
}

func isScriptEntry(entry Entry) bool {
	return len(entry.Bytes) > 0 && strings.HasSuffix(strings.ToLower(entry.Path), ".txt")
}

func BuildLintReport(snapshot Snapshot, entries []Entry, msg exportmsg.Printer) (LintReport, error) {
	if len(entries) == 0 {
		return LintReport{msg: msg}, nil
	}
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return LintReport{}, err
	}
	modRoot := artifactRoot(NativeArtifact{Entries: entries})

	catalog := crossRefCatalog{
		known:          crossRefExistenceSets(snapshot),
		completeTypes:  map[string]bool{},
		expectsBaseMod: len(project.List("dependencies")) > 0,
	}

	if _, indexed := crossRefVanilla(); indexed && !catalog.expectsBaseMod {
		for _, specs := range crossRefSpecs {
			for _, spec := range specs {
				catalog.completeTypes[spec.targetType] = true
			}
		}
	}

	allowedCategory := declaredTechCategories(entries)
	techCategories, techCategoriesKnown := vanillaTechCategories()
	var findings []string
	skipCounter := map[string]int{}
	tokenCounter := map[string]int{}

	for _, entry := range entries {
		if !isScriptEntry(entry) {
			continue
		}
		text := scriptText(entry)
		block, _, err := paradox.Parse(text)
		if err != nil || block == nil {

			continue
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(entry.Path, modRoot), "/")
		locator := newTokenLocator(text)

		walkBlock(block, func(e *paradox.Entry) {
			if e.Key == nil {
				return
			}
			key := *e.Key
			for _, spec := range crossRefSpecs[key] {
				for _, target := range spec.extract(e) {
					if target.id == "" {
						continue
					}
					line, column, ok := locator.locate(target.id, target.line)
					if !ok {
						continue
					}
					status := catalog.classify(spec.targetType, target.id)
					if status == "resolved" {
						continue
					}
					findings = append(findings, fmt.Sprintf(
						"%s · %s -> %s:%s · %s:%d:%d",
						status, key, spec.targetType, target.id, rel, line, column,
					))
				}
			}
			if strings.EqualFold(key, "targeted_modifier") && e.Block != nil {
				hasTag := false
				for _, sub := range e.Block.Entries {
					if sub.Key != nil && strings.EqualFold(*sub.Key, "tag") {
						hasTag = true
						break
					}
				}
				if !hasTag {
					skipCounter["targeted_modifier[missing_tag · "+missingTagPreview(e.Block)+"]"]++
				}
			}
			if techBonusKeys[key] && e.Block != nil {
				for _, category := range fieldsInBlock(e, "category") {
					id := strings.TrimSpace(category.id)
					if id == "" || !techCategoriesKnown || techCategories[id] || allowedCategory[id] {
						continue
					}
					tokenCounter["unknown_tech_category · "+id]++
				}
			}
		})
	}

	report := LintReport{CrossRef: findings, Scripted: ScanScriptedRefs(entries, modRoot), msg: msg}
	for _, name := range sortedKeys(skipCounter) {
		report.CrossRef = append(report.CrossRef,
			fmt.Sprintf("warning · %s: %d", name, skipCounter[name]))
	}
	for _, name := range sortedKeys(tokenCounter) {
		report.Token = append(report.Token, fmt.Sprintf("%s: %d", name, tokenCounter[name]))
	}
	return report, nil
}

func missingTagPreview(block *paradox.Block) string {
	inner := strings.TrimSpace(strings.ReplaceAll(importsvc.ToScript(block, 0), "\n", " "))
	if utf8.RuneCountInString(inner) <= 80 {
		return inner
	}
	return string([]rune(inner)[:80])
}

func sortedKeys(counter map[string]int) []string {
	names := make([]string, 0, len(counter))
	for name := range counter {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func checkedRefTypes() []string {
	seen := map[string]bool{}
	var names []string
	for _, specs := range crossRefSpecs {
		for _, spec := range specs {
			if !seen[spec.targetType] {
				seen[spec.targetType] = true
				names = append(names, spec.targetType)
			}
		}
	}
	sort.Strings(names)
	return names
}

func crossRefHeader(msg exportmsg.Printer) []string {
	lines := commentLines(msg.Text("crossref.log.header", exportmsg.Args{"types": strings.Join(checkedRefTypes(), ", ")}))
	return append(lines, "")
}

func tokenHeader(msg exportmsg.Printer) []string {
	return append(commentLines(msg.Text("token.log.header", nil)), "")
}

func (r LintReport) Entries(modRoot string) []Entry {
	if modRoot == "" {
		return nil
	}
	var out []Entry
	if len(r.CrossRef) > 0 || len(r.Scripted) > 0 {
		lines := append(crossRefHeader(r.msg), r.CrossRef...)
		if len(r.Scripted) > 0 {
			if len(r.CrossRef) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, scriptedSectionHeader(r.msg)...)
			lines = append(lines, scriptedLines(r.Scripted, r.msg)...)
		}
		out = append(out, Entry{
			Path:         modRoot + "/" + crossRefLogName,
			Bytes:        []byte(strings.Join(lines, "\n") + "\n"),
			Origin:       "generated",
			SourceEntity: "cross_ref_lint",
		})
	}
	if len(r.Token) > 0 {
		out = append(out, Entry{
			Path:         modRoot + "/" + tokenLogName,
			Bytes:        []byte(strings.Join(append(tokenHeader(r.msg), r.Token...), "\n") + "\n"),
			Origin:       "generated",
			SourceEntity: "token_lint",
		})
	}
	return out
}

type TextRef struct {
	Effect string

	TargetType string

	TargetID string

	Line int
}

func ExtractTextRefs(text string) []TextRef {
	block, _, err := paradox.Parse(text)
	if err != nil || block == nil {
		return nil
	}
	var out []TextRef
	walkBlock(block, func(entry *paradox.Entry) {
		if entry.Key == nil {
			return
		}
		key := *entry.Key
		for _, spec := range crossRefSpecs[key] {
			for _, target := range spec.extract(entry) {
				if target.id == "" {
					continue
				}
				out = append(out, TextRef{
					Effect: key, TargetType: spec.targetType,
					TargetID: target.id, Line: target.line,
				})
			}
		}
	})
	return out
}

func IsVanillaRef(targetType, targetID string) bool {
	return loadCrossRefVanillaIDs()[targetType][targetID]
}

func CheckedRefTypes() []string {
	seen := map[string]bool{}
	var out []string
	for _, specs := range crossRefSpecs {
		for _, spec := range specs {
			if !seen[spec.targetType] {
				seen[spec.targetType] = true
				out = append(out, spec.targetType)
			}
		}
	}
	sort.Strings(out)
	return out
}
