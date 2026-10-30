package guirender

import (
	"path"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/paradox"
)

type Character struct {
	Token string   `json:"token"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
	Slot  string   `json:"slot,omitempty"`

	IdeaToken string `json:"idea_token,omitempty"`
	Ideology  string `json:"ideology,omitempty"`

	Traits []string `json:"traits,omitempty"`
	Cost   int      `json:"cost,omitempty"`

	AdvisorTraits   []string `json:"-"`
	CommanderTraits []string `json:"-"`

	CommanderVisible *paradox.Block `json:"-"`
	Order            int            `json:"-"`

	LeaderTraits []string       `json:"-"`
	Skills       map[string]int `json:"skills,omitempty"`

	AdvisorAllowed   *paradox.Block `json:"-"`
	AdvisorVisible   *paradox.Block `json:"-"`
	AdvisorAvailable *paradox.Block `json:"-"`
	Large            string         `json:"large,omitempty"`
	Small            string         `json:"small,omitempty"`

	ScientistTraits []string       `json:"-"`
	ScientistSkills map[string]int `json:"-"`
}

type characterRecord struct {
	tag        string
	characters []*Character
}

func (idx *Index) traitLabels(tokens []string) []map[string]any {
	out := make([]map[string]any, 0, len(tokens))
	for _, token := range tokens {
		item := map[string]any{"token": token}
		if label, ok := idx.Localise(token); ok {
			item["label"] = label
		}
		out = append(out, item)
	}
	return out
}

func blockTraits(entries []*paradox.Entry) []string {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil ||
			!strings.EqualFold(*entry.Key, "traits") {
			continue
		}
		return blockTokens(entry.Block.Entries)
	}
	return nil
}

func commanderSkills(entries []*paradox.Entry) map[string]int {
	out := map[string]int{}
	for _, key := range []string{
		"skill", "attack_skill", "defense_skill", "planning_skill", "logistics_skill",
		"maneuvering_skill", "coordination_skill",
	} {
		if value, ok := toFloat(ciScalarValue(entries, key)); ok {
			out[key] = int(value)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (idx *Index) indexCharacterFile(name string, body []byte) {

	tag := strings.ToUpper(strings.TrimSuffix(path.Base(name), path.Ext(name)))
	if len(tag) != 3 {
		tag = ""
	}
	text, ok := gfx.DecodeScriptText(body)
	if !ok {
		return
	}
	root, _, err := paradox.Parse(text)
	if err != nil {
		return
	}
	var record *characterRecord
	if tag != "" {
		record = idx.characters[tag]
		if record == nil {
			record = &characterRecord{tag: tag}
			idx.characters[tag] = record
		}
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "characters") {
			continue
		}
		for _, person := range entry.Block.Entries {
			if person == nil || person.Key == nil || person.Block == nil {
				continue
			}
			if parsed := parseCharacter(*person.Key, person.Block.Entries); parsed != nil {
				idx.characterSeq++
				parsed.Order = idx.characterSeq
				if record != nil {
					record.characters = append(record.characters, parsed)
				}
				idx.charactersByToken[parsed.Token] = parsed
			}
		}
	}
}

func parseCharacter(token string, body []*paradox.Entry) *Character {
	person := &Character{Token: token, Name: ciScalar(body, "name")}
	if person.Name == "" {
		person.Name = token
	}
	collect := func(entries []*paradox.Entry) {
		for _, entry := range entries {
			if entry == nil || entry.Key == nil {
				continue
			}
			key := strings.ToLower(*entry.Key)
			switch {
			case key == "portraits" && entry.Block != nil:
				large, small := portraitSprites(entry.Block.Entries)
				if person.Large == "" {
					person.Large = large
				}
				if person.Small == "" {
					person.Small = small
				}
			case key == "country_leader" && entry.Block != nil:
				person.Roles = appendOnce(person.Roles, "country_leader")
				if len(person.LeaderTraits) == 0 {
					person.LeaderTraits = blockTraits(entry.Block.Entries)
				}
				if person.Ideology == "" {
					person.Ideology = ciScalar(entry.Block.Entries, "ideology")
				}
			case key == "advisor" && entry.Block != nil:
				person.Roles = appendOnce(person.Roles, "advisor")
				if person.Slot == "" {
					person.Slot = ciScalar(entry.Block.Entries, "slot")
				}
				if person.IdeaToken == "" {
					person.IdeaToken = ciScalar(entry.Block.Entries, "idea_token")
				}
				if len(person.AdvisorTraits) == 0 {
					person.AdvisorTraits = blockTraits(entry.Block.Entries)
				}
				if person.Cost == 0 {
					if cost, ok := toFloat(ciScalarValue(entry.Block.Entries, "cost")); ok {
						person.Cost = int(cost)
					}
				}
				for _, field := range entry.Block.Entries {
					if field == nil || field.Key == nil || field.Block == nil {
						continue
					}
					switch strings.ToLower(*field.Key) {
					case "allowed":
						if person.AdvisorAllowed == nil {
							person.AdvisorAllowed = field.Block
						}
					case "visible":
						if person.AdvisorVisible == nil {
							person.AdvisorVisible = field.Block
						}
					case "available":
						if person.AdvisorAvailable == nil {
							person.AdvisorAvailable = field.Block
						}
					}
				}
			case key == "scientist" && entry.Block != nil:
				person.Roles = appendOnce(person.Roles, "scientist")
				if len(person.ScientistTraits) == 0 {
					person.ScientistTraits = blockTraits(entry.Block.Entries)
				}
				if person.ScientistSkills == nil {
					if skills := childBlock(entry.Block.Entries, "skills"); skills != nil {
						person.ScientistSkills = map[string]int{}
						for key, level := range numberBlock(skills) {
							person.ScientistSkills[strings.ToLower(key)] = int(level)
						}
					}
				}
			case (key == "field_marshal" || key == "corps_commander" || key == "navy_leader") && entry.Block != nil:
				person.Roles = appendOnce(person.Roles, key)
				if len(person.CommanderTraits) == 0 {
					person.CommanderTraits = blockTraits(entry.Block.Entries)
				}
				if person.CommanderVisible == nil {
					for _, field := range entry.Block.Entries {
						if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, "visible") {
							person.CommanderVisible = field.Block
						}
					}
				}
				if person.Skills == nil {
					person.Skills = commanderSkills(entry.Block.Entries)
				}
			}
		}
	}
	collect(body)

	for _, entry := range body {
		if entry != nil && entry.Key != nil && entry.Block != nil && strings.EqualFold(*entry.Key, "instance") {
			collect(entry.Block.Entries)
		}
	}
	if len(person.Roles) == 0 && person.Large == "" && person.Small == "" {
		return nil
	}
	person.Traits = person.AdvisorTraits
	if len(person.Traits) == 0 {
		person.Traits = person.CommanderTraits
	}

	sort.SliceStable(person.Roles, func(a, b int) bool { return roleRank[person.Roles[a]] < roleRank[person.Roles[b]] })
	return person
}

var roleRank = map[string]int{"country_leader": 0, "advisor": 1, "field_marshal": 2, "corps_commander": 3, "navy_leader": 4, "scientist": 5}

func portraitSprites(entries []*paradox.Entry) (large, small string) {
	for _, group := range []string{"civilian", "army", "navy"} {
		for _, entry := range entries {
			if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, group) {
				continue
			}
			if large == "" {
				large = ciScalar(entry.Block.Entries, "large")
			}
			if small == "" {
				small = ciScalar(entry.Block.Entries, "small")
			}
		}
	}
	return large, small
}

func appendOnce(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func (idx *Index) peopleOf(tag, date string) []*Character {
	seen := map[string]bool{}
	out := []*Character{}
	for _, token := range idx.RecruitedCharacters(tag, date) {
		if seen[token] {
			continue
		}
		for layer := idx; layer != nil; layer = layer.fallback {
			if person, ok := layer.charactersByToken[token]; ok {
				seen[token] = true
				out = append(out, person)
				break
			}
		}
	}
	if len(out) > 0 {

		return out
	}
	for layer := idx; layer != nil; layer = layer.fallback {
		record, ok := layer.characters[tag]
		if !ok {
			continue
		}
		for _, person := range record.characters {
			if !seen[person.Token] {
				seen[person.Token] = true
				out = append(out, person)
			}
		}
	}
	return out
}

func (idx *Index) characterDepth(token string) int {
	depth := 0
	for layer := idx; layer != nil; layer = layer.fallback {
		if _, ok := layer.charactersByToken[token]; ok {
			return depth
		}
		depth++
	}
	return depth
}

func (idx *Index) ListCharacterTags() []string {
	seen := map[string]struct{}{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for tag := range layer.characters {
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

func (idx *Index) CharacterNameKey(token string) (nameKey, origin string, ok bool) {
	want := strings.TrimSpace(token)
	if want == "" {
		return "", "", false
	}
	for layer := idx; layer != nil; layer = layer.fallback {
		person, found := layer.charactersByToken[want]
		if !found {
			continue
		}
		nameKey = person.Name
		if nameKey == "" {
			nameKey = person.Token
		}
		return nameKey, layerOrigin(layer), true
	}
	return "", "", false
}

func (idx *Index) CharactersPayload(tag string) (map[string]any, bool) {
	return idx.CharactersPayloadFor(tag, "")
}

func (idx *Index) CharactersPayloadFor(tag, wantParty string) (map[string]any, bool) {
	return idx.CharactersPayloadAt(tag, wantParty, "")
}

func (idx *Index) CharactersPayloadAt(tag, wantParty, date string) (map[string]any, bool) {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	if wantParty == "" {
		wantParty = idx.RulingPartyOf(tag)
	}
	wantParty = idx.PartyOfIdeology(wantParty)
	people := idx.peopleOf(tag, date)

	if len(people) == 0 && !idx.hasCountryTag(tag) {
		return nil, false
	}
	render := func(person *Character, sprite string) map[string]any {
		name := person.Name
		if localised, ok := idx.Localise(person.Name); ok {
			name = localised
		}
		out := map[string]any{"token": person.Token, "name": name, "roles": person.Roles}
		if person.Slot != "" {
			out["slot"] = person.Slot
		}
		if person.IdeaToken != "" {
			out["idea_token"] = person.IdeaToken
		}
		if person.Ideology != "" {
			out["ideology"] = person.Ideology
		}
		if len(person.Traits) > 0 {
			out["traits"] = idx.traitLabels(person.Traits)
		}
		if person.Cost > 0 {
			out["cost"] = person.Cost
		}
		if len(person.Skills) > 0 {
			out["skills"] = person.Skills
		}
		if sprite != "" {
			out["portrait"] = sprite
			out["portrait_sprite"] = idx.spritePayload(sprite)
		}
		return out
	}

	promoted := idx.PromotedLeader(tag, date)
	var leader map[string]any
	var leaderFallback map[string]any

	leaderDepth := 0
	advisors := []map[string]any{}
	commanders := []map[string]any{}

	start := idx.decisionStartOf(tag, date)
	for _, person := range people {
		for _, role := range person.Roles {
			switch role {
			case "country_leader":
				rendered := render(person, firstNonEmpty(person.Large, person.Small))
				if leaderFallback == nil {
					leaderFallback = rendered
				}
				if promoted != "" {
					if person.Token == promoted {
						leader = rendered
					}
					continue
				}
				if wantParty != "" && idx.PartyOfIdeology(person.Ideology) == wantParty {
					if depth := idx.characterDepth(person.Token); leader == nil || depth < leaderDepth {
						leader, leaderDepth = rendered, depth
					}
				}
			case "advisor":
				item := render(person, firstNonEmpty(person.Small, person.Large))

				if person.Cost == 0 {
					if cost := idx.characterSlotCost(person.Slot); cost > 0 {
						item["cost"] = cost
					}
				}

				cost, _ := item["cost"].(int)
				name, _ := item["name"].(string)
				item["tooltip"] = idx.AdvisorTooltip(name, person.AdvisorTraits, cost)
				item["shown"] = start.holds(person.AdvisorAllowed) && start.holds(person.AdvisorVisible)
				item["available"] = start.holds(person.AdvisorAvailable)
				advisors = append(advisors, item)
			case "field_marshal", "corps_commander", "navy_leader":

				item := render(person, firstNonEmpty(person.Small, person.Large))
				delete(item, "traits")
				if len(person.CommanderTraits) > 0 {
					item["traits"] = idx.traitLabels(person.CommanderTraits)
				}
				commanders = append(commanders, item)
			}
		}
	}
	if leader == nil {
		leader = leaderFallback
	}

	slotIdeas := map[string][]map[string]any{}
	hired := map[string]bool{}
	spirits := []map[string]any{}

	modifiers := []map[string]any{}
	starting := idx.StartingIdeas(tag, date)
	for _, token := range starting {
		if idx.ideaCategoryOf(token) == "country" {
			spirits = append(spirits, idx.ideaLabels([]string{token}, "idea")...)
			continue
		}

		if group := idx.ideaCategoryOf(token); idx.isSlotCategory(group) {
			slotIdeas[group] = append(slotIdeas[group], idx.ideaLabels([]string{token}, "idea")...)
			continue
		}
		if icon, ok := idx.dynamicModifierIcon(token); ok {
			item := map[string]any{"token": token, "icon": icon, "icon_sprite": idx.spritePayload(icon), "tooltip": idx.IdeaTooltip(token)}
			if label, found := idx.Localise(token); found {
				item["label"] = label
			}
			modifiers = append(modifiers, item)
			continue
		}
		hired[token] = true
	}
	spirits = append(spirits, modifiers...)
	for _, advisor := range advisors {
		token, _ := advisor["idea_token"].(string)
		if token == "" {
			token, _ = advisor["token"].(string)
		}
		if hired[token] {
			advisor["hired"] = true
		}
	}
	return map[string]any{
		"tag": tag, "leader": leader, "advisors": advisors, "commanders": commanders,
		"categories": idx.ideaCategoriesPayload(), "spirits": spirits,
		"slot_ideas": idx.withDefaultLaws(slotIdeas),

		"starting_ideas": starting,
	}, true
}

func (idx *Index) withDefaultLaws(filled map[string][]map[string]any) map[string][]map[string]any {
	for _, category := range idx.IdeaCategories() {
		for _, slot := range category.Slots {
			if slot.Character || len(filled[slot.Name]) > 0 {
				continue
			}
			if token := idx.DefaultIdeaOf(slot.Name); token != "" {
				filled[slot.Name] = idx.ideaLabels([]string{token}, "idea")
			}
		}
	}
	return filled
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
