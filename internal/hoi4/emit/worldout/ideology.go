package worldout

import (
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/clausewitz"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/reftables"
)

func vanillaSubEntry(sub reftables.SubIdeology) *OMap {
	e := NewOMap()
	e.Set("id", sub.ID)
	e.Set("group", sub.Group)
	e.Set("name", NewOMap())
	return e
}

type registry struct {
	groups      []string
	roots       []rootDefinition
	rootEntries []*OMap
	subs        []*OMap
	preset      Value
}

type rootDefinition struct {
	id   string
	body string
}

func normalizeRegistry(raw Value) registry {
	out := registry{groups: []string{}, subs: []*OMap{}}
	rawMap, ok := raw.(*OMap)
	if !ok || rawMap == nil {
		return out
	}
	if g, has := rawMap.Get("groups"); has {
		if gl, isList := g.([]Value); isList {
			for _, item := range gl {
				if s, isStr := item.(string); isStr {
					out.groups = append(out.groups, s)
				}
			}
		}
	}
	if r, has := rawMap.Get("root_definitions"); has {
		if rl, isList := r.([]Value); isList {
			for _, item := range rl {
				root, isMap := item.(*OMap)
				if !isMap {
					continue
				}
				idV, _ := root.Get("id")
				id, isStr := idV.(string)
				id = pyStrip(id)
				if !isStr || id == "" {
					continue
				}
				entry := root.Clone()
				entry.Set("id", id)
				entry.Set("body", pyStrip(pyStrOrEmpty(eGet(entry, "body"))))
				if nm, hasName := entry.Get("name"); !hasName {
					entry.Set("name", NewOMap())
				} else if _, isMap := nm.(*OMap); !isMap {
					entry.Set("name", NewOMap())
				}
				if dv, hasDesc := entry.Get("desc"); hasDesc {
					if _, isMap := dv.(*OMap); !isMap {
						entry.Set("desc", NewOMap())
					}
				}
				body := pyStrip(pyStrOrEmpty(eGet(entry, "body")))
				entry.Set("body", body)
				out.rootEntries = append(out.rootEntries, entry)
				out.roots = append(out.roots, rootDefinition{id: id, body: body})
				found := false
				for _, group := range out.groups {
					if group == id {
						found = true
						break
					}
				}
				if !found {
					out.groups = append(out.groups, id)
				}
			}
		}
	}
	if s, has := rawMap.Get("sub_ideologies"); has {
		if sl, isList := s.([]Value); isList {
			for _, item := range sl {
				switch t := item.(type) {
				case string:
					e := NewOMap()
					e.Set("id", t)
					e.Set("group", "")
					e.Set("name", NewOMap())
					out.subs = append(out.subs, e)
				case *OMap:
					idv, _ := t.Get("id")
					id, isStr := idv.(string)
					if !isStr || id == "" {
						continue
					}
					entry := t.Clone()
					if !entry.Has("group") {
						entry.Set("group", "")
					}
					if nm, hasName := entry.Get("name"); !hasName {
						entry.Set("name", NewOMap())
					} else if _, isMap := nm.(*OMap); !isMap {
						entry.Set("name", NewOMap())
					}
					if dv, hasDesc := entry.Get("desc"); hasDesc {
						if _, isMap := dv.(*OMap); !isMap {
							entry.Set("desc", NewOMap())
						}
					}
					out.subs = append(out.subs, entry)
				}
			}
		}
	}
	if p, has := rawMap.Get("preset"); has {
		out.preset = p
	}
	return out
}

func mergeWithVanilla(raw Value) registry {
	project := normalizeRegistry(raw)
	vanilla := reftables.Ideologies()

	groups := append([]string{}, vanilla.Groups...)
	for _, g := range project.groups {
		found := false
		for _, existing := range groups {
			if existing == g {
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, g)
		}
	}

	byID := map[string]*OMap{}
	order := []string{}
	for _, sub := range vanilla.SubIdeologies {
		if _, exists := byID[sub.ID]; !exists {
			order = append(order, sub.ID)
		}
		byID[sub.ID] = vanillaSubEntry(sub)
	}
	for _, s := range project.subs {
		id := pyStrOrEmpty(eGet(s, "id"))
		if _, exists := byID[id]; !exists {
			order = append(order, id)
		}
		byID[id] = s
	}

	merged := registry{
		groups:      groups,
		roots:       append([]rootDefinition(nil), project.roots...),
		rootEntries: append([]*OMap(nil), project.rootEntries...),
	}
	for _, id := range order {
		merged.subs = append(merged.subs, byID[id])
	}
	merged.preset = nil
	if pyTruthy(project.preset) {
		merged.preset = project.preset
	}
	return merged
}

func customRootDefinitions(raw Value) []*OMap {
	project := normalizeRegistry(raw)
	out := []*OMap{}
	for _, root := range project.rootEntries {
		id := pyStrip(pyStrOrEmpty(eGet(root, "id")))
		if id == "" || reftables.Ideologies().IsGroup(id) {
			continue
		}
		out = append(out, root)
	}
	return out
}

var customRootIDRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func directBlock(block *paradox.Block, key string) *paradox.Block {
	if block == nil {
		return nil
	}
	for _, entry := range block.Entries {
		if entry.Key != nil && *entry.Key == key && entry.Block != nil {
			return entry.Block
		}
	}
	return nil
}

func validCustomRootDefinitions(raw Value) bool {
	seen := map[string]bool{}
	for _, root := range normalizeRegistry(raw).rootEntries {
		id := pyStrip(pyStrOrEmpty(eGet(root, "id")))
		body := pyStrip(pyStrOrEmpty(eGet(root, "body")))
		if !customRootIDRE.MatchString(id) || reftables.Ideologies().IsGroup(id) || seen[id] ||
			body == "" || len(body) > 100000 {
			return false
		}
		seen[id] = true
		wrapped := "ideologies = {\n" + id + " = {\n" + body + "\n}\n}\n"
		ast, recoveries, err := paradox.Parse(wrapped)
		if err != nil || len(recoveries) != 0 || ast == nil || len(ast.Entries) != 1 {
			return false
		}
		outer := directBlock(ast, "ideologies")
		if outer == nil {
			return false
		}
		var keyed []*paradox.Entry
		for _, entry := range outer.Entries {
			if entry.Key != nil {
				keyed = append(keyed, entry)
			}
		}
		if len(keyed) != 1 || *keyed[0].Key != id || keyed[0].Block == nil ||
			directBlock(keyed[0].Block, "types") == nil ||
			directBlock(keyed[0].Block, "color") == nil {
			return false
		}
	}
	return true
}

func customSubIdeologies(raw Value) []*OMap {
	merged := mergeWithVanilla(raw)
	vanilla := reftables.Ideologies()
	supportedGroups := map[string]bool{}
	for _, g := range merged.groups {
		if vanilla.IsGroup(g) {
			supportedGroups[g] = true
		}
	}
	for _, root := range customRootDefinitions(raw) {
		supportedGroups[pyStrOrEmpty(eGet(root, "id"))] = true
	}
	out := []*OMap{}
	for _, s := range merged.subs {
		sidV, _ := s.Get("id")
		sid, _ := sidV.(string)
		grpV, _ := s.Get("group")
		grp, _ := grpV.(string)
		if sid == "" || vanilla.IsSub(sid) {
			continue
		}
		if grp == "" || !supportedGroups[grp] {
			continue
		}
		out = append(out, s)
	}
	return out
}

func paradoxChildBlock(entries []*paradox.Entry, key string) *paradox.Block {
	for _, entry := range entries {
		if entry.Key != nil && *entry.Key == key && entry.Block != nil {
			return entry.Block
		}
	}
	return nil
}

func safeRootDefinitions(raw Value) []rootDefinition {
	reg := normalizeRegistry(raw)
	out := []rootDefinition{}
	seen := map[string]bool{}
	for _, root := range reg.roots {
		if seen[root.id] || reftables.Ideologies().IsGroup(root.id) || !customRootIDRE.MatchString(root.id) ||
			root.body == "" || len([]rune(root.body)) > 100000 {
			return nil
		}
		wrapped := "ideologies = {\n" + root.id + " = {\n" + root.body + "\n}\n}\n"
		block, recoveries, err := paradox.Parse(wrapped)
		if failed, _ := clausewitz.ParseErrorMessage(recoveries, err); failed {
			return nil
		}
		outer := paradoxChildBlock(block.Entries, "ideologies")
		if len(block.Entries) != 1 || outer == nil || len(outer.Entries) != 1 ||
			outer.Entries[0].Key == nil || *outer.Entries[0].Key != root.id ||
			outer.Entries[0].Block == nil || paradoxChildBlock(outer.Entries[0].Block.Entries, "types") == nil ||
			paradoxChildBlock(outer.Entries[0].Block.Entries, "color") == nil {
			return nil
		}
		seen[root.id] = true
		out = append(out, root)
	}
	return out
}

func projectManagedRootEntries(raw Value) []*OMap {
	safe := safeRootDefinitions(raw)
	if len(safe) == 0 {
		return nil
	}
	safeIDs := make(map[string]bool, len(safe))
	for _, root := range safe {
		safeIDs[root.id] = true
	}
	rawMap, ok := raw.(*OMap)
	if !ok || rawMap == nil {
		return nil
	}
	value, ok := rawMap.Get("root_definitions")
	if !ok {
		return nil
	}
	rows, ok := value.([]Value)
	if !ok {
		return nil
	}
	out := make([]*OMap, 0, len(safe))
	for _, value := range rows {
		row, ok := value.(*OMap)
		if !ok || row == nil {
			continue
		}
		id := pyStrip(pyStrOrEmpty(eGet(row, "id")))
		if safeIDs[id] {
			out = append(out, row)
		}
	}
	return out
}

func resolveRulingPartyGroup(raw Value, isSubmod bool, value string) (string, bool) {
	v := pyStrip(value)
	if v == "" {
		return "", false
	}
	roots := map[string]bool{}
	for _, group := range reftables.Ideologies().Groups {
		roots[group] = true
	}
	if !isSubmod {
		for _, root := range safeRootDefinitions(raw) {
			roots[root.id] = true
		}
	}
	if roots[v] {
		return v, true
	}
	if group := subToGroupMap(raw)[v]; roots[group] {
		return group, true
	}
	return "", false
}

func subToGroupMap(raw Value) map[string]string {
	merged := mergeWithVanilla(raw)
	m := map[string]string{}
	for _, s := range merged.subs {
		idV, _ := s.Get("id")
		id, _ := idV.(string)
		if id == "" {
			continue
		}
		grpV, _ := s.Get("group")
		grp, _ := grpV.(string)
		m[id] = grp
	}
	return m
}

func extractVanillaGroupBlock(baseText, group string) (string, bool) {
	re := regexp.MustCompile(`(?m)^\t` + regexp.QuoteMeta(group) + `\s*=\s*\{`)
	m := re.FindStringIndex(baseText)
	if m == nil {
		return "", false
	}
	depth := 0
	for j := m[1] - 1; j < len(baseText); j++ {
		switch baseText[j] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return baseText[m[0] : j+1], true
			}
		}
	}
	return "", false
}

func injectSubsIntoTypes(groupBlock, subsText string) (string, bool) {
	re := regexp.MustCompile(`(?m)^\t\ttypes\s*=\s*\{`)
	m := re.FindStringIndex(groupBlock)
	if m == nil {
		return "", false
	}
	depth := 0
	for j := m[1] - 1; j < len(groupBlock); j++ {
		switch groupBlock[j] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				open := m[1] - 1

				if !strings.Contains(groupBlock[open+1:j], "\n") {
					return strings.TrimRight(groupBlock[:j], " \t") + "\n" + subsText + "\t\t" + groupBlock[j:], true
				}
				lineStart := strings.LastIndex(groupBlock[:j], "\n") + 1
				return groupBlock[:lineStart] + subsText + groupBlock[lineStart:], true
			}
		}
	}
	return "", false
}

func customSubsText(subs []*OMap) string {
	var b strings.Builder
	for _, s := range subs {
		id := pyStrOrEmpty(eGet(s, "id"))
		b.WriteString("\t\t\t" + id + " = {\n")
		if eBool(s, "random_disabled") {
			b.WriteString("\t\t\t\tcan_be_randomly_selected = no\n")
		}
		b.WriteString("\t\t\t}\n\n")
	}
	return b.String()
}

func dedentRootBody(body string) []string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.Trim(body, "\r\n")
	lines := strings.Split(body, "\n")
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return lines
	}
	for i, line := range lines {
		if strings.TrimSpace(line) != "" && len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}
	return lines
}

func formatCustomRootBlock(root *OMap) string {
	id := pyStrOrEmpty(eGet(root, "id"))
	lines := []string{"\t" + id + " = {"}
	for _, line := range dedentRootBody(pyStrOrEmpty(eGet(root, "body"))) {
		if line == "" {
			lines = append(lines, "")
		} else {
			lines = append(lines, "\t\t"+line)
		}
	}
	lines = append(lines, "\t}")
	return strings.Join(lines, "\n")
}

func typeIDsInGroupBlock(groupBlock, groupID string) map[string]bool {
	out := map[string]bool{}
	ast, recoveries, err := paradox.Parse("ideologies = {\n" + groupBlock + "\n}\n")
	if err != nil || len(recoveries) != 0 {
		return out
	}
	group := directBlock(directBlock(ast, "ideologies"), groupID)
	types := directBlock(group, "types")
	if types == nil {
		return out
	}
	for _, entry := range types.Entries {
		if entry.Key != nil {
			out[*entry.Key] = true
		}
	}
	return out
}

func appendRootBlocks(full string, blocks []string) string {
	trimmed := strings.TrimRight(full, " \t\r\n")
	if len(blocks) == 0 || !strings.HasSuffix(trimmed, "}") {
		return full
	}
	return strings.TrimRight(trimmed[:len(trimmed)-1], " \t\r\n") +
		"\n\n" + strings.Join(blocks, "\n\n") + "\n}\n"
}

func customRootBlocks(roots []*OMap, byGroup map[string][]*OMap) ([]string, bool) {
	var rootBlocks []string
	for _, root := range roots {
		rootID := pyStrOrEmpty(eGet(root, "id"))
		groupBlock := formatCustomRootBlock(root)
		existing := typeIDsInGroupBlock(groupBlock, rootID)
		var inject []*OMap
		for _, sub := range byGroup[rootID] {
			if !existing[pyStrOrEmpty(eGet(sub, "id"))] {
				inject = append(inject, sub)
			}
		}
		if len(inject) > 0 {
			var ok bool
			groupBlock, ok = injectSubsIntoTypes(groupBlock, customSubsText(inject))
			if !ok {
				return nil, false
			}
		}
		rootBlocks = append(rootBlocks, groupBlock)
	}
	return rootBlocks, true
}

func CustomRootIdeologiesFile(raw Value, sourceRoots map[string]bool) (string, []*OMap) {
	roots := customRootDefinitions(raw)
	if len(roots) == 0 || !validCustomRootDefinitions(raw) || len(safeRootDefinitions(raw)) != len(roots) {
		return "", nil
	}
	byGroup := map[string][]*OMap{}
	for _, s := range customSubIdeologies(raw) {
		grp := pyStrOrEmpty(eGet(s, "group"))
		byGroup[grp] = append(byGroup[grp], s)
	}
	kept := make([]*OMap, 0, len(roots))
	for _, root := range roots {
		if !sourceRoots[strings.ToLower(pyStrOrEmpty(eGet(root, "id")))] {
			kept = append(kept, root)
		}
	}
	blocks, ok := customRootBlocks(kept, byGroup)
	if !ok || len(blocks) == 0 {
		return "", nil
	}
	return "ideologies = {\n" + strings.Join(blocks, "\n\n") + "\n}\n", kept
}

func generateFullIdeologiesFile(raw Value, baseText string) string {
	customs := customSubIdeologies(raw)
	roots := customRootDefinitions(raw)
	if len(customs) == 0 && len(roots) == 0 {
		return ""
	}
	if !validCustomRootDefinitions(raw) {
		return ""
	}
	if len(safeRootDefinitions(raw)) != len(roots) {
		return ""
	}
	if baseText == "" {
		return ""
	}

	byGroup := map[string][]*OMap{}
	var order []string
	for _, s := range customs {
		grp := pyStrOrEmpty(eGet(s, "group"))
		if _, ok := byGroup[grp]; !ok {
			order = append(order, grp)
		}
		byGroup[grp] = append(byGroup[grp], s)
	}

	full := baseText
	vanilla := reftables.Ideologies()
	for _, grp := range order {
		if !vanilla.IsGroup(grp) {
			continue
		}
		groupBlock, ok := extractVanillaGroupBlock(full, grp)
		if !ok {
			return ""
		}
		injected, ok := injectSubsIntoTypes(groupBlock, customSubsText(byGroup[grp]))
		if !ok {
			return ""
		}
		full = strings.Replace(full, groupBlock, injected, 1)
	}
	rootBlocks, ok := customRootBlocks(roots, byGroup)
	if !ok {
		return ""
	}
	full = appendRootBlocks(full, rootBlocks)

	for _, g := range vanilla.Groups {
		re := regexp.MustCompile(`(?m)^\t` + regexp.QuoteMeta(g) + `\s*=\s*\{`)
		if len(re.FindAllStringIndex(full, -1)) != 1 {
			return ""
		}
	}
	for _, root := range roots {
		id := pyStrOrEmpty(eGet(root, "id"))
		re := regexp.MustCompile(`(?m)^\t` + regexp.QuoteMeta(id) + `\s*=\s*\{`)
		if len(re.FindAllStringIndex(full, -1)) != 1 {
			return ""
		}
	}

	if !strings.HasSuffix(full, "\n") {
		full += "\n"
	}
	return full
}

type IdeologyExportFacts struct {
	Standalone           bool
	HasCustomDefinitions bool
	RawOverride          bool
	FullFileGenerated    bool
}

func (f IdeologyExportFacts) RequiresReplacePath() bool {
	return f.Standalone && f.HasCustomDefinitions && !f.RawOverride && f.FullFileGenerated
}

type IdeologyExportPlan struct {
	Facts    IdeologyExportFacts
	FullText string

	InheritedSource bool
	SourceRoots     map[string]bool
}

func BuildIdeologyExportPlan(t *Tables, project *OMap, isSubmod, rawOverride bool) IdeologyExportPlan {
	regRaw, _ := project.Get("ideology_registry")
	hasCustom := len(customSubIdeologies(regRaw)) > 0 || len(customRootDefinitions(regRaw)) > 0
	plan := IdeologyExportPlan{Facts: IdeologyExportFacts{
		Standalone:           !isSubmod,
		HasCustomDefinitions: hasCustom,
		RawOverride:          rawOverride,
	}}
	if plan.Facts.Standalone && hasCustom && !rawOverride && t != nil {
		plan.FullText = generateFullIdeologiesFile(regRaw, t.IdeologyBaseText)
	}
	plan.Facts.FullFileGenerated = plan.FullText != ""
	return plan
}

func buildCustomRootPoliticalPortraits(tags []string, roots []*OMap) string {
	rootIDs := make([]string, 0, len(roots))
	seenRoots := map[string]bool{}
	for _, root := range roots {
		id := pyStrip(pyStrOrEmpty(eGet(root, "id")))
		if id == "" || reftables.Ideologies().IsGroup(id) || seenRoots[id] {
			continue
		}
		seenRoots[id] = true
		rootIDs = append(rootIDs, id)
	}
	if len(rootIDs) == 0 {
		return ""
	}
	sort.Strings(rootIDs)

	tagIDs := make([]string, 0, len(tags))
	seenTags := map[string]bool{}
	for _, raw := range tags {
		tag := toUpper(pyStrip(raw))
		if !regexp.MustCompile(`^[A-Z0-9_]{3,64}$`).MatchString(tag) || seenTags[tag] {
			continue
		}
		seenTags[tag] = true
		tagIDs = append(tagIDs, tag)
	}
	if len(tagIDs) == 0 {
		return ""
	}
	sort.Strings(tagIDs)

	var blocks []string
	for _, tag := range tagIDs {
		lines := []string{tag + " = {", "\tpolitical = {"}
		for _, rootID := range rootIDs {
			lines = append(lines,
				"\t\t"+rootID+" = {",
				"\t\t\tmale = {",
				"\t\t\t\t\"GFX_Portrait_Europe_Generic_2\"",
				"\t\t\t}",
				"\t\t}",
			)
		}
		lines = append(lines, "\t}", "}")
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	return strings.Join(blocks, "\n\n") + "\n"
}
