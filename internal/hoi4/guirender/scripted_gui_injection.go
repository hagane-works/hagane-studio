package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

var (
	scriptedGUIWrappers    = stringSet("scripted_gui")
	scriptedEffectWrappers = stringSet("scripted_effects", "scripted_effect")
	dynamicListKeys        = stringSet("dynamic_list", "dynamic_lists")
	gateConditionKeys      = stringSet("has_idea", "has_country_flag", "has_global_flag", "has_government", "is_in_array", "has_dlc", "original_tag", "tag")
	gateHostBlocks         = stringSet("limit", "visible", "trigger", "triggers")
)

var memberEffectNameHints = []string{"_join_", "_add_", "join_", "_member"}

type InjectionDynamicList struct {
	EntryContainer *string `json:"entry_container"`
	ArrayName      *string `json:"array_name"`
}

type InjectionPoint struct {
	WindowName       string                 `json:"window_name"`
	Extensible       bool                   `json:"extensible"`
	EntryContainer   *string                `json:"entry_container"`
	ArrayName        *string                `json:"array_name"`
	Gate             []string               `json:"gate"`
	MemberAddEffects []string               `json:"member_add_effects"`
	DynamicLists     []InjectionDynamicList `json:"dynamic_lists"`
	MemberAddIdeas   []string               `json:"member_add_ideas"`
	MemberRecipe     *string                `json:"member_recipe"`
}

type NamedInjectionSource struct {
	Name string
	Text string
}

func stringSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func inStringSet(values map[string]struct{}, value string) bool {
	_, ok := values[strings.ToLower(value)]
	return ok
}

func injectionBlock(entry *paradox.Entry) []*paradox.Entry {
	if entry == nil || entry.Block == nil {
		return nil
	}
	return entry.Block.Entries
}

func injectionScalar(entries []*paradox.Entry, key string) (string, bool) {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block != nil {
			continue
		}
		if strings.EqualFold(*entry.Key, key) {
			return entry.Scalar.StrValue(), true
		}
	}
	return "", false
}

func injectionTrimQuoted(value string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(value), `"`))
}

func injectionUnwrap(value string) (string, bool) {
	value = injectionTrimQuoted(value)
	if value == "" {
		return "", false
	}
	if dot := strings.LastIndex(value, "."); dot >= 0 {
		value = value[dot+1:]
	}
	if value == "" {
		return "", false
	}
	return value, true
}

func injectionTopBlocks(ast *paradox.Block, wrappers map[string]struct{}) []*paradox.Entry {
	out := []*paradox.Entry{}
	sawWrapper := false
	if ast == nil {
		return out
	}
	for _, entry := range ast.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !inStringSet(wrappers, *entry.Key) {
			continue
		}
		sawWrapper = true
		for _, child := range entry.Block.Entries {
			if child != nil && child.Key != nil && child.Block != nil {
				out = append(out, child)
			}
		}
	}
	if !sawWrapper {
		for _, entry := range ast.Entries {
			if entry != nil && entry.Key != nil && entry.Block != nil {
				out = append(out, entry)
			}
		}
	}
	return out
}

func injectionDynamicLists(entries []*paradox.Entry) []InjectionDynamicList {
	out := []InjectionDynamicList{}
	var consume func([]*paradox.Entry)
	consume = func(body []*paradox.Entry) {
		rawContainer, ok := injectionScalar(body, "entry_container")
		if !ok || strings.TrimSpace(rawContainer) == "" {
			return
		}
		container := injectionTrimQuoted(rawContainer)
		if container == "" {
			return
		}
		var array *string
		rawName, nameOK := injectionScalar(body, "name")
		if !nameOK || rawName == "" {
			rawName, nameOK = injectionScalar(body, "array")
		}
		if nameOK {
			if unwrapped, ok := injectionUnwrap(rawName); ok {
				array = stringPointer(unwrapped)
			}
		}
		out = append(out, InjectionDynamicList{EntryContainer: stringPointer(container), ArrayName: array})
	}
	var walk func([]*paradox.Entry)
	walk = func(current []*paradox.Entry) {
		for _, entry := range current {
			if entry == nil || entry.Block == nil {
				continue
			}
			body := entry.Block.Entries
			if entry.Key != nil && inStringSet(dynamicListKeys, *entry.Key) {
				if _, ok := injectionScalar(body, "entry_container"); ok {
					consume(body)
				} else {
					for _, grid := range body {
						if grid != nil && grid.Key != nil && grid.Block != nil {
							consume(grid.Block.Entries)
						}
					}
				}
			}
			walk(body)
		}
	}
	walk(entries)
	return out
}

func injectionGate(entries []*paradox.Entry) []string {
	out := []string{}
	seen := map[string]bool{}
	var collect func([]*paradox.Entry)
	collect = func(current []*paradox.Entry) {
		for _, entry := range current {
			if entry == nil {
				continue
			}
			if entry.Key != nil {
				key := strings.ToLower(*entry.Key)
				if inStringSet(gateConditionKeys, key) && !seen[key] {
					seen[key] = true
					out = append(out, key)
				}
			}
			if entry.Block != nil {
				collect(entry.Block.Entries)
			}
		}
	}
	var walk func([]*paradox.Entry)
	walk = func(current []*paradox.Entry) {
		for _, entry := range current {
			if entry == nil || entry.Block == nil {
				continue
			}
			if entry.Key != nil && inStringSet(gateHostBlocks, *entry.Key) {
				collect(entry.Block.Entries)
			}
			walk(entry.Block.Entries)
		}
	}
	walk(entries)
	return out
}

func normalizeInjectionArrays(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		unwrapped, ok := injectionUnwrap(value)
		if !ok || seen[strings.ToLower(unwrapped)] {
			continue
		}
		seen[strings.ToLower(unwrapped)] = true
		out = append(out, unwrapped)
	}
	return out
}

func injectionEffectReferencesArray(entries []*paradox.Entry, array string) bool {
	target, ok := injectionUnwrap(array)
	if !ok {
		return false
	}
	var walk func([]*paradox.Entry) bool
	walk = func(current []*paradox.Entry) bool {
		for _, entry := range current {
			if entry == nil {
				continue
			}
			if entry.Key != nil && strings.EqualFold(*entry.Key, "add_to_array") && entry.Block != nil {
				if raw, found := injectionScalar(entry.Block.Entries, "array"); found {
					if value, valid := injectionUnwrap(raw); valid && strings.EqualFold(value, target) {
						return true
					}
				}
				for _, child := range entry.Block.Entries {
					if child != nil && child.Key != nil {
						if value, valid := injectionUnwrap(*child.Key); valid && strings.EqualFold(value, target) {
							return true
						}
					}
				}
			}
			if entry.Block != nil && walk(entry.Block.Entries) {
				return true
			}
		}
		return false
	}
	return walk(entries)
}

func injectionHasAddToArray(entries []*paradox.Entry) bool {
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if entry.Key != nil && strings.EqualFold(*entry.Key, "add_to_array") {
			return true
		}
		if entry.Block != nil && injectionHasAddToArray(entry.Block.Entries) {
			return true
		}
	}
	return false
}

func injectionNameLooksLikeMemberAdd(name string) bool {
	name = strings.ToLower(name)
	for _, hint := range memberEffectNameHints {
		if strings.Contains(name, hint) {
			return true
		}
	}
	return false
}

func injectionMemberAddEffects(effectBlocks []*paradox.Entry, arrayNames []string) []string {
	arrays := normalizeInjectionArrays(arrayNames)
	out := []string{}
	seen := map[string]bool{}
	for _, block := range effectBlocks {
		if block == nil || block.Key == nil || block.Block == nil || seen[*block.Key] || len(block.Block.Entries) == 0 {
			continue
		}
		linked := false
		for _, array := range arrays {
			if injectionEffectReferencesArray(block.Block.Entries, array) {
				linked = true
				break
			}
		}
		if !linked && injectionNameLooksLikeMemberAdd(*block.Key) && injectionHasAddToArray(block.Block.Entries) {
			linked = true
		}
		if linked {
			seen[*block.Key] = true
			out = append(out, *block.Key)
		}
	}
	return out
}

func injectionIdeasGatingArrayFill(entries []*paradox.Entry, arrays []string) []string {
	targets := map[string]bool{}
	for _, array := range arrays {
		targets[strings.ToLower(array)] = true
	}
	out := []string{}
	seen := map[string]bool{}
	blockFillsTarget := func(body []*paradox.Entry) bool {
		for _, entry := range body {
			if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "add_to_array") {
				continue
			}
			if raw, found := injectionScalar(entry.Block.Entries, "array"); found {
				if value, valid := injectionUnwrap(raw); valid && targets[strings.ToLower(value)] {
					return true
				}
			}
			for _, child := range entry.Block.Entries {
				if child != nil && child.Key != nil {
					if value, valid := injectionUnwrap(*child.Key); valid && targets[strings.ToLower(value)] {
						return true
					}
				}
			}
		}
		return false
	}
	ideasInLimit := func(body []*paradox.Entry) []string {
		found := []string{}
		var collect func([]*paradox.Entry)
		collect = func(current []*paradox.Entry) {
			for _, entry := range current {
				if entry == nil {
					continue
				}
				if entry.Key != nil && entry.Block == nil && strings.EqualFold(*entry.Key, "has_idea") {
					found = append(found, injectionTrimQuoted(entry.Scalar.StrValue()))
				}
				if entry.Block != nil {
					collect(entry.Block.Entries)
				}
			}
		}
		for _, entry := range body {
			if entry != nil && entry.Key != nil && entry.Block != nil && strings.EqualFold(*entry.Key, "limit") {
				collect(entry.Block.Entries)
			}
		}
		return found
	}
	var walk func([]*paradox.Entry)
	walk = func(current []*paradox.Entry) {
		for _, entry := range current {
			if entry == nil || entry.Block == nil {
				continue
			}
			body := entry.Block.Entries
			if blockFillsTarget(body) {
				for _, idea := range ideasInLimit(body) {
					if idea != "" && !seen[idea] {
						seen[idea] = true
						out = append(out, idea)
					}
				}
			}
			walk(body)
		}
	}
	walk(entries)
	return out
}

func injectionMemberAddIdeas(effectBlocks []*paradox.Entry, arrayNames []string) []string {
	arrays := normalizeInjectionArrays(arrayNames)
	out := []string{}
	seen := map[string]bool{}
	for _, block := range effectBlocks {
		if block == nil || block.Block == nil {
			continue
		}
		for _, idea := range injectionIdeasGatingArrayFill(block.Block.Entries, arrays) {
			if idea != "" && !seen[idea] {
				seen[idea] = true
				out = append(out, idea)
			}
		}
	}
	return out
}

func injectionRecipe(effects, ideas []string) *string {
	if len(ideas) > 0 {
		return stringPointer("add idea " + strings.Join(ideas[:min(3, len(ideas))], " / "))
	}
	if len(effects) > 0 {
		return stringPointer("run effect " + strings.Join(effects[:min(3, len(effects))], " / "))
	}
	return nil
}

func stringPointer(value string) *string {
	copy := value
	return &copy
}

func AnalyzeInjectionPoints(guiText, effectText string) []InjectionPoint {
	guiAST, _, err := paradox.Parse(guiText)
	if err != nil {
		return []InjectionPoint{}
	}
	effectBlocks := []*paradox.Entry{}
	if effectText != "" {
		if effectAST, _, effectErr := paradox.Parse(effectText); effectErr == nil {
			effectBlocks = injectionTopBlocks(effectAST, scriptedEffectWrappers)
		}
	}
	results := []InjectionPoint{}
	for _, block := range injectionTopBlocks(guiAST, scriptedGUIWrappers) {
		body := injectionBlock(block)
		rawWindow, ok := injectionScalar(body, "window_name")
		if !ok {
			continue
		}
		lists := injectionDynamicLists(body)
		var entryContainer, arrayName *string
		if len(lists) > 0 {
			entryContainer, arrayName = lists[0].EntryContainer, lists[0].ArrayName
		}
		arrays := []string{}
		for _, list := range lists {
			if list.ArrayName != nil {
				arrays = append(arrays, *list.ArrayName)
			}
		}
		extensible := entryContainer != nil
		effects, ideas := []string{}, []string{}
		if extensible && len(effectBlocks) > 0 {
			effects = injectionMemberAddEffects(effectBlocks, arrays)
			ideas = injectionMemberAddIdeas(effectBlocks, arrays)
		}
		results = append(results, InjectionPoint{
			WindowName: injectionTrimQuoted(rawWindow), Extensible: extensible,
			EntryContainer: entryContainer, ArrayName: arrayName,
			Gate: injectionGate(body), MemberAddEffects: effects,
			DynamicLists: lists, MemberAddIdeas: ideas,
			MemberRecipe: injectionRecipe(effects, ideas),
		})
	}
	return results
}

func ProjectInjectionPoints(guiFiles, effectFiles []NamedInjectionSource) []InjectionPoint {
	sort.SliceStable(guiFiles, func(i, j int) bool { return guiFiles[i].Name < guiFiles[j].Name })
	sort.SliceStable(effectFiles, func(i, j int) bool { return effectFiles[i].Name < effectFiles[j].Name })
	effectChunks := []string{}
	seenEffectFiles := map[string]bool{}
	for _, source := range effectFiles {
		key := strings.ToLower(source.Name)
		if seenEffectFiles[key] {
			continue
		}
		seenEffectFiles[key] = true
		if source.Text != "" {
			effectChunks = append(effectChunks, source.Text)
		}
	}
	effectText := strings.Join(effectChunks, "\n")
	results := []InjectionPoint{}
	seenWindows := map[string]bool{}
	for _, source := range guiFiles {
		if source.Text == "" {
			continue
		}
		for _, point := range AnalyzeInjectionPoints(source.Text, effectText) {
			if seenWindows[point.WindowName] {
				continue
			}
			seenWindows[point.WindowName] = true
			results = append(results, point)
		}
	}
	return results
}
