package guirender

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/paradox"
)

type BookmarkCountry struct {
	Tag      string
	Ideology string
	History  string
	Ideas    []string
	Focuses  []string

	Minor bool
}

type Bookmark struct {
	Name           string
	Desc           string
	Date           string
	Picture        string
	DefaultCountry string

	Default bool

	Filters   []string
	Countries []BookmarkCountry
}

type datedIdeas struct {
	Date   string
	Tokens []string
	Remove bool

	DLC dlcGuard
}

type countryPolitics struct {
	Ideas []datedIdeas

	Effects      []effectCall
	NamedThreats []namedThreat

	Technologies []datedIdeas

	OOBs []datedIdeas

	GrandDoctrines []datedIdeas

	SubDoctrines []datedIdeas

	Mastery []datedVariable

	Variables []datedVariable

	Variants []equipmentVariant

	Recruits   []datedIdeas
	Promotions []datedIdeas

	Capital          int
	RulingParty      string
	ElectionsAllowed bool
	HasElections     bool

	LastElection      string
	ElectionFrequency int
	Popularities      map[string]int
}

func (idx *Index) indexBookmarkFile(name string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "bookmarks") {
			continue
		}
		for _, mark := range entry.Block.Entries {
			if mark == nil || mark.Key == nil || mark.Block == nil || !strings.EqualFold(*mark.Key, "bookmark") {
				continue
			}
			idx.bookmarks = append(idx.bookmarks, parseBookmark(mark.Block.Entries))
			idx.bookmarkFiles = append(idx.bookmarkFiles, name)
		}
	}
}

func parseBookmark(body []*paradox.Entry) Bookmark {
	mark := Bookmark{
		Name:           ciScalar(body, "name"),
		Desc:           ciScalar(body, "desc"),
		Date:           ciScalar(body, "date"),
		Picture:        ciScalar(body, "picture"),
		DefaultCountry: strings.ToUpper(ciScalar(body, "default_country")),
		Default:        strings.EqualFold(ciScalar(body, "default"), "yes"),
		Filters:        scalarList(body, "filters"),
	}
	seen := map[string]bool{}
	for _, entry := range body {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		tag := strings.ToUpper(strings.Trim(*entry.Key, "\""))
		if len(tag) != 3 || seen[tag] {

			continue
		}
		fields := entry.Block.Entries
		if ciScalar(fields, "ideology") == "" && ciScalar(fields, "history") == "" {
			continue
		}
		seen[tag] = true
		mark.Countries = append(mark.Countries, BookmarkCountry{
			Tag:      tag,
			Ideology: ciScalar(fields, "ideology"),
			History:  strings.Trim(ciScalar(fields, "history"), "\""),
			Ideas:    scalarList(fields, "ideas"),
			Focuses:  scalarList(fields, "focuses"),
			Minor:    strings.EqualFold(ciScalar(fields, "minor"), "yes"),
		})
	}
	return mark
}

func scalarList(entries []*paradox.Entry, key string) []string {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, key) {
			continue
		}
		out := []string{}
		for _, item := range entry.Block.Entries {
			if item == nil || item.Block != nil {
				continue
			}
			if item.Key != nil {
				out = append(out, *item.Key)
				continue
			}
			if value := toString(scalarValue(item.Scalar)); value != "" {
				out = append(out, value)
			}
		}
		return out
	}
	return nil
}

func (idx *Index) indexIdeologyFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "ideologies") {
			continue
		}
		for _, group := range entry.Block.Entries {
			if group == nil || group.Key == nil || group.Block == nil {
				continue
			}
			party := strings.ToLower(*group.Key)
			idx.ideologyParty[party] = party
			idx.ideologyGroups = append(idx.ideologyGroups, ideologyGroup{Name: party, Color: blockColor(group.Block.Entries)})
			for _, field := range group.Block.Entries {
				if field == nil || field.Key == nil || field.Block == nil || !strings.EqualFold(*field.Key, "types") {
					continue
				}
				for _, kind := range field.Block.Entries {
					if kind != nil && kind.Key != nil {
						idx.ideologyParty[strings.ToLower(*kind.Key)] = party
					}
				}
			}
		}
	}
}

type ideologyGroup struct {
	Name  string
	Color [3]int
}

func blockColor(entries []*paradox.Entry) [3]int {
	out := [3]int{}
	for _, field := range entries {
		if field == nil || field.Key == nil || field.Block == nil || !strings.EqualFold(*field.Key, "color") {
			continue
		}
		at := 0
		for _, part := range field.Block.Entries {
			if part == nil || part.Block != nil || at > 2 {
				continue
			}

			raw := ""
			if part.Key != nil {
				raw = *part.Key
			} else {
				raw = toString(scalarValue(part.Scalar))
			}
			if value, err := strconv.ParseFloat(raw, 64); err == nil {
				out[at] = int(value)
				at++
			}
		}
		return out
	}
	return out
}

func (idx *Index) IdeologyGroups() []ideologyGroup {
	layers := []*Index{}
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
	}
	out := []ideologyGroup{}
	at := map[string]int{}
	for i := len(layers) - 1; i >= 0; i-- {
		for _, group := range layers[i].ideologyGroups {
			if j, seen := at[group.Name]; seen {
				out[j].Color = group.Color
				continue
			}
			at[group.Name] = len(out)
			out = append(out, group)
		}
	}
	return out
}

func (idx *Index) PartyOfIdeology(ideology string) string {
	ideology = strings.ToLower(strings.TrimSpace(ideology))
	for layer := idx; layer != nil; layer = layer.fallback {
		if party, ok := layer.ideologyParty[ideology]; ok {
			return party
		}
	}
	return ideology
}

func (idx *Index) RulingPartyOf(tag string) string {
	if politics := idx.countryPoliticsFor(tag); politics != nil {
		return strings.ToLower(politics.RulingParty)
	}
	return ""
}

func (idx *Index) indexIdeaFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "ideas") {
			continue
		}
		for _, category := range entry.Block.Entries {
			if category == nil || category.Block == nil {
				continue
			}
			group := ""
			if category.Key != nil {
				group = strings.ToLower(*category.Key)
			}
			if group != "" && !containsFold(idx.ideaGroupOrder, group) {
				for _, idea := range category.Block.Entries {
					if idea != nil && idea.Key != nil && idea.Block != nil {
						idx.ideaGroupOrder = append(idx.ideaGroupOrder, group)
						break
					}
				}
			}
			for _, idea := range category.Block.Entries {
				if idea == nil || idea.Key == nil || idea.Block == nil {
					continue
				}
				if picture := ciScalar(idea.Block.Entries, "picture"); picture != "" {
					idx.ideaPictures[*idea.Key] = picture
				}

				for _, field := range idea.Block.Entries {
					if field == nil || field.Key == nil || field.Block == nil ||
						!strings.EqualFold(*field.Key, "modifier") {
						continue
					}
					idx.ideaModifierBlocks[*idea.Key] = field.Block
					if value, ok := toFloat(ciScalarValue(field.Block.Entries, "conscription")); ok {
						idx.ideaConscription[*idea.Key] = value
					}

					if value, ok := toFloat(ciScalarValue(field.Block.Entries, "min_export")); ok {
						idx.ideaMinExport[*idea.Key] = value
					}
				}
				idx.recordIdeaChoice(group, *idea.Key, idea.Block.Entries)
				if group != "" {
					idx.ideaGroups[*idea.Key] = group

					if _, taken := idx.ideaDefaults[group]; !taken &&
						strings.EqualFold(ciScalar(idea.Block.Entries, "default"), "yes") {
						idx.ideaDefaults[group] = *idea.Key
					}
				}
			}
		}
	}
}

func (idx *Index) DefaultIdeaOf(group string) string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if token, ok := layer.ideaDefaults[group]; ok {
			return token
		}
	}
	return ""
}

func (idx *Index) ideaPicture(token string) string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if picture, ok := layer.ideaPictures[token]; ok {
			return picture
		}
	}
	return ""
}

func (idx *Index) indexCountryTagFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block != nil {
			continue
		}
		tag := strings.ToUpper(*entry.Key)
		if len(tag) == 3 {
			idx.countryTags[tag] = struct{}{}
			if idx.countryTagSeq == nil {
				idx.countryTagSeq = map[string]int{}
			}
			if _, seen := idx.countryTagSeq[tag]; !seen {
				idx.countryTagNext++
				idx.countryTagSeq[tag] = idx.countryTagNext
			}

			if file := toString(scalarValue(entry.Scalar)); file != "" {
				idx.countryFiles[tag] = strings.Trim(file, "\"")
			}
		}
	}
}

func (idx *Index) indexCountryHistoryFile(name string, body []byte) {

	tag := strings.ToUpper(strings.TrimSpace(path.Base(name)))
	if len(tag) < 3 {
		return
	}
	tag = tag[:3]
	root, ok := parseScript(body)
	if !ok {
		return
	}
	politics := &countryPolitics{Popularities: map[string]int{}}
	if capital, ok := toFloat(ciScalarValue(root.Entries, "capital")); ok {
		politics.Capital = int(capital)
	}

	var walk func(entries []*paradox.Entry)
	walk = func(entries []*paradox.Entry) {
		for _, entry := range entries {
			if entry == nil || entry.Key == nil || entry.Block == nil {
				continue
			}
			switch strings.ToLower(*entry.Key) {
			case "set_politics":
				if politics.RulingParty == "" {
					politics.RulingParty = ciScalar(entry.Block.Entries, "ruling_party")
					allowed := ciScalar(entry.Block.Entries, "elections_allowed")
					politics.HasElections = allowed != ""
					politics.ElectionsAllowed = strings.EqualFold(allowed, "yes")
					politics.LastElection = strings.Trim(ciScalar(entry.Block.Entries, "last_election"), `"`)
					politics.ElectionFrequency, _ = strconv.Atoi(ciScalar(entry.Block.Entries, "election_frequency"))
				}
			case "set_popularities":
				if len(politics.Popularities) == 0 {
					for _, party := range entry.Block.Entries {
						if party == nil || party.Key == nil || party.Block != nil {
							continue
						}
						if value, err := strconv.Atoi(toString(scalarValue(party.Scalar))); err == nil {
							politics.Popularities[strings.ToLower(*party.Key)] = value
						}
					}
				}
			default:
				walk(entry.Block.Entries)
			}
		}
	}
	walk(root.Entries)
	idx.indexSubjects(tag, root.Entries)
	idx.indexFactions(tag, root.Entries)
	politics.Ideas = collectDatedIdeas(root.Entries, "")
	politics.Technologies = collectTechnologies(root.Entries, "")
	politics.Recruits = collectCalls(root.Entries, "recruit_character", "")
	politics.Promotions = collectCalls(root.Entries, "promote_character", "")
	politics.Effects = collectEffects(root.Entries, historyNumberCommands, "")
	politics.NamedThreats = collectNamedThreats(root.Entries, "")
	politics.GrandDoctrines = collectCalls(root.Entries, "set_grand_doctrine", "")
	politics.SubDoctrines = collectCalls(root.Entries, "set_sub_doctrine", "")
	politics.Mastery = collectMastery(root.Entries, "")
	politics.Variables = collectVariables(root.Entries, "")
	politics.Variants = collectVariants(root.Entries, "")
	politics.OOBs = append(collectCalls(root.Entries, "set_oob", ""),
		append(collectCalls(root.Entries, "set_naval_oob", ""),
			collectCalls(root.Entries, "set_air_oob", "")...)...)
	if countryPoliticsEmpty(politics) {
		return
	}
	idx.countryHistories[tag] = append(idx.countryHistories[tag], historyFile{Path: strings.ToLower(path.Base(name)), Politics: politics})
}

func countryPoliticsEmpty(politics *countryPolitics) bool {
	return politics == nil || (politics.RulingParty == "" &&
		len(politics.Popularities) == 0 && politics.Capital == 0 &&
		!politics.HasElections && politics.LastElection == "" && politics.ElectionFrequency == 0 &&
		len(politics.Ideas) == 0 && len(politics.Effects) == 0 && len(politics.NamedThreats) == 0 &&
		len(politics.Technologies) == 0 && len(politics.OOBs) == 0 &&
		len(politics.GrandDoctrines) == 0 && len(politics.SubDoctrines) == 0 &&
		len(politics.Mastery) == 0 && len(politics.Variables) == 0 && len(politics.Variants) == 0 &&
		len(politics.Recruits) == 0 && len(politics.Promotions) == 0)
}

type historyFile struct {
	Path     string
	Politics *countryPolitics
}

func mergePolitics(earlier, later *countryPolitics) *countryPolitics {
	out := *later
	if out.Capital == 0 {
		out.Capital = earlier.Capital
	}
	if out.RulingParty == "" {
		out.RulingParty, out.ElectionsAllowed, out.HasElections = earlier.RulingParty, earlier.ElectionsAllowed, earlier.HasElections
		out.LastElection, out.ElectionFrequency = earlier.LastElection, earlier.ElectionFrequency
	}
	if len(out.Popularities) == 0 {
		out.Popularities = earlier.Popularities
	}
	out.Ideas = joinNew(earlier.Ideas, later.Ideas)
	out.Effects = joinNew(earlier.Effects, later.Effects)
	out.NamedThreats = joinNew(earlier.NamedThreats, later.NamedThreats)
	out.Technologies = joinNew(earlier.Technologies, later.Technologies)
	out.OOBs = joinNew(earlier.OOBs, later.OOBs)
	out.GrandDoctrines = joinNew(earlier.GrandDoctrines, later.GrandDoctrines)
	out.SubDoctrines = joinNew(earlier.SubDoctrines, later.SubDoctrines)
	out.Mastery = joinNew(earlier.Mastery, later.Mastery)
	out.Variables = joinNew(earlier.Variables, later.Variables)
	out.Variants = joinNew(earlier.Variants, later.Variants)
	out.Recruits = joinNew(earlier.Recruits, later.Recruits)
	out.Promotions = joinNew(earlier.Promotions, later.Promotions)
	return &out
}

func joinNew[T any](earlier, later []T) []T {
	out := append(make([]T, 0, len(earlier)+len(later)), earlier...)
	have := make(map[string]bool, len(earlier))
	for _, entry := range earlier {
		have[fmt.Sprintf("%#v", entry)] = true
	}
	for _, entry := range later {
		if !have[fmt.Sprintf("%#v", entry)] {
			out = append(out, entry)
		}
	}
	return out
}

func collectDatedIdeas(entries []*paradox.Entry, date string) []datedIdeas {
	return collectDatedIdeasGuarded(entries, date, dlcGuard{})
}

func collectDatedIdeasGuarded(entries []*paradox.Entry, date string, guard dlcGuard) []datedIdeas {
	out := []datedIdeas{}
	branch := branchState{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		if entry.Block != nil && (key == "if" || key == "else_if") {
			if inner, ok := branch.enterIf(entry.Block.Entries, guard); ok {
				out = append(out, collectDatedIdeasGuarded(entry.Block.Entries, date, inner)...)
			}
			continue
		}
		if entry.Block != nil && key == "else" {
			if inner, ok := branch.enterElse(); ok {
				out = append(out, collectDatedIdeasGuarded(entry.Block.Entries, date, inner)...)
			}
			continue
		}
		if key == "add_dynamic_modifier" || key == "remove_dynamic_modifier" {
			modifier := ""
			if entry.Block != nil {
				modifier = ciScalar(entry.Block.Entries, "modifier")
			} else {
				modifier = toString(scalarValue(entry.Scalar))
			}
			if modifier != "" {
				out = append(out, datedIdeas{
					Date: date, Tokens: []string{modifier}, Remove: key == "remove_dynamic_modifier", DLC: guard,
				})
			}
			continue
		}
		if key == "add_ideas" || key == "remove_ideas" {
			tokens := []string{}
			if entry.Block != nil {
				for _, idea := range entry.Block.Entries {
					if idea == nil || idea.Block != nil {
						continue
					}
					if idea.Key != nil {
						tokens = append(tokens, *idea.Key)
					} else if value := toString(scalarValue(idea.Scalar)); value != "" {
						tokens = append(tokens, value)
					}
				}
			} else if value := toString(scalarValue(entry.Scalar)); value != "" {
				tokens = append(tokens, value)
			}
			if len(tokens) > 0 {
				out = append(out, datedIdeas{Date: date, Tokens: tokens, Remove: key == "remove_ideas", DLC: guard})
			}
			continue
		}
		if entry.Block != nil && isGameDate(key) {
			out = append(out, collectDatedIdeasGuarded(entry.Block.Entries, key, guard)...)
		}
	}
	return out
}

func limitHolds(entries []*paradox.Entry) (holds, decidable bool) {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		if strings.EqualFold(*entry.Key, "limit") {
			return evalCondition(entry.Block.Entries)
		}
	}
	return false, false
}

func evalCondition(entries []*paradox.Entry) (holds, decidable bool) {
	result := true
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		switch {
		case key == "has_dlc" && entry.Block == nil:

		case (key == "not" || key == "nor") && entry.Block != nil:
			inner, ok := evalCondition(entry.Block.Entries)
			if !ok {
				return false, false
			}
			result = result && !inner
		case (key == "and" || key == "or") && entry.Block != nil:
			inner, ok := evalCondition(entry.Block.Entries)
			if !ok {
				return false, false
			}
			result = result && inner
		default:
			return false, false
		}
	}
	return result, true
}

func collectCalls(entries []*paradox.Entry, command, date string) []datedIdeas {
	return collectCallsGuarded(entries, command, date, dlcGuard{})
}

func collectCallsGuarded(entries []*paradox.Entry, command, date string, guard dlcGuard) []datedIdeas {
	out := []datedIdeas{}
	branch := branchState{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		if entry.Block != nil && (key == "if" || key == "else_if") {
			if inner, ok := branch.enterIf(entry.Block.Entries, guard); ok {
				out = append(out, collectCallsGuarded(entry.Block.Entries, command, date, inner)...)
			}
			continue
		}
		if entry.Block != nil && key == "else" {
			if inner, ok := branch.enterElse(); ok {
				out = append(out, collectCallsGuarded(entry.Block.Entries, command, date, inner)...)
			}
			continue
		}
		if entry.Block != nil {
			at := date
			if isGameDate(key) {
				at = key
			}
			out = append(out, collectCallsGuarded(entry.Block.Entries, command, at, guard)...)
			continue
		}
		if key == command {
			if token := toString(scalarValue(entry.Scalar)); token != "" {
				out = append(out, datedIdeas{Date: date, Tokens: []string{token}, DLC: guard})
			}
		}
	}
	return out
}

func tokensBy(blocks []datedIdeas, date string) []string {
	return tokensByOwned(blocks, date, nil)
}

func tokensByOwned(blocks []datedIdeas, date string, owned map[string]bool) []string {
	if date == "" {
		date = "1936.1.1"
	}
	out := []string{}
	for _, block := range blocks {
		if dateReached(block.Date, date) && block.DLC.holds(owned) {
			out = append(out, block.Tokens...)
		}
	}
	return out
}

func (idx *Index) RecruitedCharacters(tag, date string) []string {
	if politics := idx.countryPoliticsFor(tag); politics != nil {
		return tokensByOwned(politics.Recruits, date, idx.ownedDLC)
	}
	return nil
}

func (idx *Index) PromotedLeader(tag, date string) string {
	politics := idx.countryPoliticsFor(tag)
	if politics == nil {
		return ""
	}
	promoted := tokensByOwned(politics.Promotions, date, idx.ownedDLC)
	if len(promoted) == 0 {
		return ""
	}
	return promoted[len(promoted)-1]
}

func isGameDate(key string) bool {
	parts := strings.Split(key, ".")
	if len(parts) < 3 || len(parts) > 4 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func dateReached(when, at string) bool {
	if when == "" {
		return true
	}
	parse := func(value string) [3]int {
		out := [3]int{}
		for i, part := range strings.Split(value, ".") {
			if i > 2 {
				break
			}
			out[i], _ = strconv.Atoi(part)
		}
		return out
	}
	a, b := parse(when), parse(at)
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return true
}

func (idx *Index) StartingIdeas(tag, date string) []string {
	politics := idx.countryPoliticsFor(tag)
	if politics == nil {
		return nil
	}
	if date == "" {
		date = "1936.1.1"
	}

	blocks := append([]datedIdeas(nil), politics.Ideas...)
	sort.SliceStable(blocks, func(a, b int) bool {
		if blocks[a].Date == "" || blocks[b].Date == "" {
			return blocks[a].Date == "" && blocks[b].Date != ""
		}
		da, db := parsedDate(blocks[a].Date), parsedDate(blocks[b].Date)
		for i := 0; i < 3; i++ {
			if da[i] != db[i] {
				return da[i] < db[i]
			}
		}
		return false
	})
	out := []string{}
	seen := map[string]bool{}
	for _, block := range blocks {
		if !dateReached(block.Date, date) || !block.DLC.holds(idx.ownedDLC) {
			continue
		}
		for _, token := range block.Tokens {
			if block.Remove {
				if !seen[token] {
					continue
				}
				delete(seen, token)
				kept := out[:0]
				for _, held := range out {
					if held != token {
						kept = append(kept, held)
					}
				}
				out = kept
				continue
			}
			if seen[token] {
				continue
			}
			if group := idx.ideaCategoryOf(token); group != "" && idx.isSlotCategory(group) {
				kept := out[:0]
				for _, held := range out {
					if idx.ideaCategoryOf(held) == group {
						delete(seen, held)
						continue
					}
					kept = append(kept, held)
				}
				out = kept
			}
			seen[token] = true
			out = append(out, token)
		}
	}
	return out
}

func (idx *Index) indexDynamicModifierFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		if icon := ciScalar(entry.Block.Entries, "icon"); icon != "" {
			idx.dynamicModifiers[*entry.Key] = icon
		}
		idx.dynamicModifierBlocks[*entry.Key] = entry.Block
	}
}

func (idx *Index) ideaModifierBlockFor(token string) *paradox.Block {
	for layer := idx; layer != nil; layer = layer.fallback {
		if block, ok := layer.ideaModifierBlocks[token]; ok {
			return block
		}
	}
	return nil
}

func (idx *Index) dynamicModifierBlockFor(token string) *paradox.Block {
	for layer := idx; layer != nil; layer = layer.fallback {
		if block, ok := layer.dynamicModifierBlocks[token]; ok {
			return block
		}
	}
	return nil
}

func (idx *Index) dynamicModifierIcon(token string) (string, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if icon, ok := layer.dynamicModifiers[token]; ok {
			return icon, true
		}
	}
	return "", false
}

func (idx *Index) ideaCategoryOf(token string) string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if group, ok := layer.ideaGroups[token]; ok {
			return group
		}
	}
	return ""
}

func parseScript(body []byte) (*paradox.Block, bool) {
	text, ok := gfx.DecodeScriptText(body)
	if !ok {
		return nil, false
	}
	root, _, err := paradox.Parse(text)
	if err != nil {
		return nil, false
	}
	return root, true
}

func (idx *Index) FlagTexture(tag, ideology, size string) (rel string, width, height int, ok bool) {
	tag = strings.ToUpper(strings.TrimSpace(tag))

	if len(tag) != 3 && !cosmeticTagPattern.MatchString(tag) {
		return "", 0, 0, false
	}
	dir := "gfx/flags"
	if size != "" {
		dir += "/" + size
	}
	names := []string{}
	if ideology != "" {
		names = append(names, tag+"_"+strings.ToLower(ideology))
	}
	names = append(names, tag)
	for _, name := range names {
		for _, ext := range []string{".tga", ".dds", ".png"} {
			candidate := dir + "/" + name + ext
			if w, h, external, found := idx.textureSize(candidate); found {
				if external == "" {
					external = candidate
				}
				return external, w, h, true
			}
		}
	}
	return "", 0, 0, false
}

var cosmeticTagPattern = regexp.MustCompile(`^[A-Z0-9]{3}_[A-Z0-9_]{1,40}$`)

func (idx *Index) flagPayload(tag, ideology, size string) map[string]any {
	rel, width, height, ok := idx.FlagTexture(tag, ideology, size)

	if url, _, uploaded := idx.projectFlagURL(tag, ideology); uploaded {

		width, height = flagBox(size)
		return map[string]any{"file": "", "width": width, "height": height, "url": url, "project": true}
	}
	if !ok {
		return nil
	}
	return map[string]any{
		"file": rel, "width": width, "height": height,
		"url": idx.vanillaTextureURL(rel),
	}
}

func (idx *Index) SetProjectFlags(flags map[string]string) {
	idx.projectFlags = flags
}

func (idx *Index) SetProjectSprites(sprites map[string][2]int) {
	idx.projectSprites = sprites
}

func (idx *Index) SetProjectSpriteVersions(versions map[string]string) {
	idx.projectSpriteVersions = versions
}

func (idx *Index) projectSpriteVersion(name string) string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if version, ok := layer.projectSpriteVersions[name]; ok {
			return version
		}
	}
	return ""
}

func (idx *Index) projectSpriteSize(name string) ([2]int, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if size, ok := layer.projectSprites[name]; ok {
			return size, true
		}
	}
	return [2]int{}, false
}

func (idx *Index) projectFlagURL(tag, ideology string) (url, key string, ok bool) {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	for layer := idx; layer != nil; layer = layer.fallback {

		if ideology != "" {
			key := tag + "_" + strings.ToLower(ideology)
			if url := layer.projectFlags[key]; url != "" {
				return url, key, true
			}
		}
		if url := layer.projectFlags[tag]; url != "" {
			return url, tag, true
		}
	}
	return "", "", false
}

func flagBox(size string) (int, int) {
	switch size {
	case "medium":
		return 41, 26
	case "small":
		return 10, 7
	}
	return 82, 52
}

func (idx *Index) CountryName(tag, ideology string) string {
	name, _ := idx.countryNamed(tag, ideology)
	return name
}

func (idx *Index) CountryNameKey(tag, ideology string) string {
	_, key := idx.countryNamed(tag, ideology)
	return key
}

func (idx *Index) countryNamed(tag, ideology string) (name, key string) {
	tag = strings.ToUpper(tag)
	if ideology != "" {
		key = tag + "_" + strings.ToLower(ideology)
		if name, ok := idx.Localise(key); ok {
			return name, key
		}
	}
	if name, ok := idx.Localise(tag); ok {
		return name, tag
	}
	return tag, tag
}

func (idx *Index) countryPoliticsFor(tag string) *countryPolitics {
	tag = strings.ToUpper(tag)
	var files []historyFile
	seen := map[string]bool{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, file := range layer.countryHistories[tag] {
			if !seen[file.Path] {
				seen[file.Path] = true
				files = append(files, file)
			}
		}
	}
	switch len(files) {
	case 0:
		return nil
	case 1:
		return files[0].Politics
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	politics := files[0].Politics
	for _, file := range files[1:] {
		politics = mergePolitics(politics, file.Politics)
	}
	return politics
}

func (idx *Index) countryHistoryTags() []string {
	seen := map[string]bool{}
	var out []string
	for layer := idx; layer != nil; layer = layer.fallback {
		for tag := range layer.countryHistories {
			if !seen[tag] {
				seen[tag] = true
				out = append(out, tag)
			}
		}
	}
	sort.Strings(out)
	return out
}

func (idx *Index) bookmarkList() []Bookmark {
	marks := []Bookmark{}
	seen := map[string]bool{}
	for layer := idx; layer != nil; layer = layer.fallback {
		files := map[string]bool{}
		for at, mark := range layer.bookmarks {
			file := ""
			if at < len(layer.bookmarkFiles) {
				file = layer.bookmarkFiles[at]
			}
			if file != "" && seen[file] {
				continue
			}
			files[file] = true
			marks = append(marks, mark)
		}
		for file := range files {
			seen[file] = true
		}
	}
	if len(marks) == 0 {
		return nil
	}

	sort.SliceStable(marks, func(a, b int) bool { return marks[a].Date < marks[b].Date })
	return marks
}

func (idx *Index) DefaultScenarioDate() string {
	marks := idx.bookmarkList()
	for _, mark := range marks {
		if mark.Default && mark.Date != "" {
			return mark.Date
		}
	}
	for _, mark := range marks {
		if mark.Date != "" {
			return mark.Date
		}
	}
	return "1936.1.1.12"
}

func (idx *Index) ListCountryTags() []string {
	seen := map[string]struct{}{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for tag := range layer.countryTags {
			seen[tag] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for tag := range seen {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func (idx *Index) hasCountryTag(tag string) bool {
	for layer := idx; layer != nil; layer = layer.fallback {
		if _, ok := layer.countryTags[tag]; ok {
			return true
		}
	}
	return false
}

func (idx *Index) countryCard(tag, ideology string) map[string]any {
	politics := idx.countryPoliticsFor(tag)
	if ideology == "" && politics != nil {

		ideology = politics.RulingParty
	}
	name, nameKey := idx.countryNamed(tag, ideology)

	card := map[string]any{
		"tag":      tag,
		"name":     name,
		"name_key": nameKey,
	}
	if tree := idx.FocusTreeForTag(tag); tree != "" {

		card["focus_tree"] = tree
	}
	if ideology != "" {
		card["ideology"] = ideology
		if label, ok := idx.Localise(ideology); ok {
			card["ideology_label"] = label
		}
	}
	if flag := idx.flagPayload(tag, ideology, "medium"); flag != nil {
		card["flag"] = flag
	}
	if continent := idx.ContinentOf(tag); continent != "" {
		card["continent"] = continent
		if label, ok := idx.Localise(continent); ok {
			card["continent_label"] = label
		}
	}
	if politics != nil {
		party := politics.RulingParty
		card["ruling_party"] = party
		if label, ok := idx.Localise(party); ok {
			card["ruling_party_label"] = label
		}
		if politics.HasElections {
			card["elections_allowed"] = politics.ElectionsAllowed
		}
		if share, found := politics.Popularities[strings.ToLower(party)]; found {
			card["ruling_party_share"] = share
		}
	}
	return card
}

func (idx *Index) BookmarksPayload() map[string]any {
	marks := []map[string]any{}
	featured := map[string]struct{}{}
	for _, mark := range idx.bookmarkList() {
		countries := []map[string]any{}
		for _, country := range mark.Countries {
			featured[country.Tag] = struct{}{}
			card := idx.countryCard(country.Tag, country.Ideology)
			if country.Tag == "---" || idx.countryPoliticsFor(country.Tag) == nil {

				card["placeholder"] = true
			}
			if text, ok := idx.Localise(country.History); ok {
				card["history"] = text
			}
			card["history_key"] = country.History
			card["ideas"] = idx.ideaLabels(country.Ideas, "idea")
			card["focuses"] = idx.ideaLabels(country.Focuses, "focus")

			if leader := idx.leaderCard(country.Tag, country.Ideology, mark.Date); leader != nil {
				card["leader"] = leader
			}
			countries = append(countries, card)
		}
		entry := map[string]any{
			"name": mark.Name, "date": mark.Date, "default_country": mark.DefaultCountry,
			"countries": countries, "filters": mark.Filters, "default": mark.Default,
		}
		if contains(mark.Filters, "continent") {

			continents := []map[string]any{}
			for _, name := range idx.continentList() {
				item := map[string]any{"name": name}
				if label, ok := idx.Localise(name); ok {
					item["label"] = label
				}
				continents = append(continents, item)
			}
			entry["continents"] = continents
		}
		if label, ok := idx.Localise(mark.Name); ok {
			entry["label"] = label
		}
		if desc, ok := idx.Localise(mark.Desc); ok {
			entry["desc"] = desc
			entry["desc_key"] = mark.Desc
		}
		if mark.Picture != "" {
			entry["picture"] = mark.Picture
			entry["picture_sprite"] = idx.spritePayload(mark.Picture)
		}
		marks = append(marks, entry)
	}
	minors := []map[string]any{}

	mark0Date := ""
	if marks := idx.bookmarkList(); len(marks) > 0 {
		mark0Date = marks[0].Date
	}

	ownsLand := map[string]bool{}
	ownerDate := mark0Date
	if ownerDate == "" {
		ownerDate = "1936.1.1"
	}
	for _, state := range idx.effectiveStates() {
		if owner := idx.ownerOfState(state, ownerDate); owner != "" {
			ownsLand[owner] = true
		}
	}
	for _, tag := range idx.ListCountryTags() {
		if _, isMajor := featured[tag]; isMajor {
			continue
		}
		if len(ownsLand) > 0 && !ownsLand[tag] {
			continue
		}

		if idx.countryPoliticsFor(tag) == nil {
			continue
		}
		minor := idx.countryCard(tag, "")

		if leader := idx.leaderCard(tag, "", mark0Date); leader != nil {
			minor["leader"] = leader
		}
		minors = append(minors, minor)
	}
	return map[string]any{"bookmarks": marks, "minors": minors}
}

func (idx *Index) leaderCard(tag, ideology, date string) map[string]any {
	people, ok := idx.CharactersPayloadAt(tag, ideology, date)
	if !ok {
		return nil
	}
	leader, _ := people["leader"].(map[string]any)
	return leader
}

func (idx *Index) ideaLabels(tokens []string, kind string) []map[string]any {
	out := []map[string]any{}
	for _, token := range tokens {
		item := map[string]any{"token": token}
		if label, ok := idx.Localise(token); ok {
			item["label"] = label
		}
		name := "GFX_idea_" + token
		if picture := idx.ideaPicture(token); kind == "idea" && picture != "" {
			name = "GFX_idea_" + picture
		}
		if kind == "focus" {
			if icon := idx.focusIcon(token); icon != "" {
				name = icon
			} else {
				name = "GFX_focus_" + token
			}
		}
		if _, found := idx.spriteRecordFor(name); found {
			item["icon"] = name
			item["icon_sprite"] = idx.spritePayload(name)
		}

		tooltip := ""
		switch kind {
		case "idea":
			tooltip = idx.IdeaTooltip(token)
		case "focus":
			tooltip = idx.FocusTooltip(token)
		}
		if tooltip != "" {
			item["tooltip"] = tooltip
		}
		out = append(out, item)
	}
	return out
}

func (idx *Index) focusIcon(id string) string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if icon, ok := layer.focusIcons[id]; ok {
			return icon
		}
	}
	return ""
}
