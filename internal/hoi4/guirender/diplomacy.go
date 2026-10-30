package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type factionTemplate struct {
	Name string
	Icon string

	Manifest string
	Goals    []string
}

type factionGoal struct {
	Name     string
	Category string
	Group    string
	Manifest bool
}

func (idx *Index) indexFactionTemplateFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		token := strings.ToLower(*entry.Key)
		name := strings.Trim(ciScalar(entry.Block.Entries, "name"), "\"")
		if name == "" {
			continue
		}
		category := strings.ToLower(ciScalar(entry.Block.Entries, "category"))
		manifest := strings.EqualFold(ciScalar(entry.Block.Entries, "is_manifest"), "yes")
		if category != "" || manifest {
			if _, taken := idx.factionGoals[token]; !taken {
				idx.factionGoals[token] = factionGoal{
					Name: name, Category: category, Manifest: manifest,
					Group: strings.Trim(ciScalar(entry.Block.Entries, "group"), "\""),
				}
			}
			continue
		}
		if _, taken := idx.factionTemplates[token]; taken {
			continue
		}
		template := factionTemplate{
			Name:     name,
			Icon:     strings.Trim(ciScalar(entry.Block.Entries, "icon"), "\""),
			Manifest: strings.ToLower(ciScalar(entry.Block.Entries, "manifest")),
		}
		for _, goal := range scalarList(entry.Block.Entries, "goals") {
			template.Goals = append(template.Goals, strings.ToLower(strings.Trim(goal, "\"")))
		}
		idx.factionTemplates[token] = template
	}
}

type datedFaction struct {
	Date  string
	Value string

	DLC dlcGuard
}

func (idx *Index) indexFactions(tag string, entries []*paradox.Entry) {
	self := strings.ToUpper(tag)
	var walk func(list []*paradox.Entry, scope, outer, date string, guard dlcGuard)
	walk = func(list []*paradox.Entry, scope, outer, date string, guard dlcGuard) {
		branch := branchState{}
		for _, entry := range list {
			if entry == nil || entry.Key == nil {
				continue
			}
			key := strings.ToLower(*entry.Key)
			if entry.Block == nil {
				raw := strings.Trim(toString(scalarValue(entry.Scalar)), "\"")
				value := strings.ToUpper(raw)
				switch value {
				case "PREV":

					value = outer
				case "ROOT", "THIS":
					value = self
				}
				switch key {
				case "create_faction_from_template":
					idx.factionLed[scope] = append(idx.factionLed[scope],
						datedFaction{Date: date, Value: strings.ToLower(raw), DLC: guard})
				case "create_faction":

					idx.factionLed[scope] = append(idx.factionLed[scope],
						datedFaction{Date: date, Value: "name:" + raw, DLC: guard})
				case "add_to_faction":
					if value != "" {
						idx.factionMember[value] = append(idx.factionMember[value],
							datedFaction{Date: date, Value: scope, DLC: guard})
					}
				}
				continue
			}
			switch {
			case key == "if" || key == "else_if":
				if inner, ok := branch.enterIf(entry.Block.Entries, guard); ok {
					walk(entry.Block.Entries, scope, outer, date, inner)
				}
				continue
			case key == "else":
				if inner, ok := branch.enterElse(); ok {
					walk(entry.Block.Entries, scope, outer, date, inner)
				}
				continue
			case key == "limit":
				continue
			case isGameDate(key):
				walk(entry.Block.Entries, scope, outer, key, guard)
				continue
			}

			if isTagLike(*entry.Key) {
				walk(entry.Block.Entries, strings.ToUpper(*entry.Key), scope, date, guard)
				continue
			}
			walk(entry.Block.Entries, scope, outer, date, guard)
		}
	}
	walk(entries, self, self, "", dlcGuard{})
}

func factionAt(calls []datedFaction, date string, owned map[string]bool) (string, bool) {
	date = scenarioDate(date)
	best, found := datedFaction{}, false
	for _, call := range calls {
		if !dateReached(call.Date, date) || !call.DLC.holds(owned) {
			continue
		}
		if !found || (call.Date != "" && (best.Date == "" || laterDate(call.Date, best.Date))) {
			best, found = call, true
		}
	}
	return best.Value, found
}

func laterDate(a, b string) bool {
	da, db := parsedDate(a), parsedDate(b)
	for i := 0; i < 3; i++ {
		if da[i] != db[i] {
			return da[i] > db[i]
		}
	}
	return false
}

func isTagLike(key string) bool {
	if len(key) != 3 {
		return false
	}
	for _, r := range key {
		if !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func (idx *Index) FactionOf(tag, date string) (name, icon, leader string) {
	name, _, icon, leader = idx.factionNamed(tag, date)
	return name, icon, leader
}

func (idx *Index) FactionNameKey(tag, date string) string {
	_, key, _, _ := idx.factionNamed(tag, date)
	return key
}

func (idx *Index) factionNamed(tag, date string) (name, key, icon, leader string) {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	leader = tag
	if led, ok := idx.factionMemberOf(tag, date); ok {
		leader = led
	} else if _, leads := idx.factionTemplateOf(tag, date); !leads {
		return "", "", "", ""
	}
	token, ok := idx.factionTemplateOf(leader, date)
	if !ok {
		return "", "", "", ""
	}
	if strings.HasPrefix(token, "name:") {
		key = strings.TrimPrefix(token, "name:")
		name = key
		if localised, found := idx.Localise(key); found {
			name = localised
		}
		return name, key, "", leader
	}
	for layer := idx; layer != nil; layer = layer.fallback {
		if template, found := layer.factionTemplates[token]; found {
			name = template.Name
			if localised, ok := idx.Localise(template.Name); ok {
				name = localised
			}
			return name, template.Name, template.Icon, leader
		}
	}
	return "", "", "", leader
}

func (idx *Index) factionMemberOf(tag, date string) (string, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if leader, ok := factionAt(layer.factionMember[tag], date, idx.ownedDLC); ok {
			return leader, true
		}
	}
	return "", false
}

func (idx *Index) factionTemplateOf(tag, date string) (string, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if token, ok := factionAt(layer.factionLed[tag], date, idx.ownedDLC); ok {
			return token, true
		}
	}
	return "", false
}

type diplomaticAction struct {
	ID    string
	Title string

	PP, CP string
}

var diplomaticActionList = []diplomaticAction{
	{"declare_war", "DIPLOMACY_WAR_TITLE", "", ""},
	{"generate_wargoal", "DIPLOMACY_GENERATE_WARGOAL_TITLE", "", ""},
	{"guarantee", "DIPLOMACY_GUARANTEE_TITLE", "NDiplomacy.GUARANTEE_COST", ""},
	{"military_access", "DIPLOMACY_MILACC_TITLE", "", ""},
	{"offer_military_access", "DIPLOMACY_OFFER_MILACC_TITLE", "", ""},
	{"docking_rights", "DIPLOMACY_DOCKING_RIGHTS_TITLE", "", ""},
	{"offer_docking_rights", "DIPLOMACY_OFFER_DOCKING_RIGHTS_TITLE", "", ""},
	{"air_base_access", "DIPLOMACY_AIR_BASE_ACCESS_TITLE", "", ""},
	{"offer_air_base_access", "DIPLOMACY_OFFER_AIR_BASE_ACCESS_TITLE", "", ""},
	{"improve_relations", "DIPLOMACY_IMPROVERELATION_TITLE", "NDiplomacy.BASE_IMPROVE_RELATION_COST", ""},
	{"send_attache", "DIPLOMACY_SEND_ATTACHE_TITLE", "NDiplomacy.BASE_SEND_ATTACHE_COST", "NDiplomacy.BASE_SEND_ATTACHE_CP_COST"},
	{"non_aggression_pact", "DIPLOMACY_NONAGGRESSIONPACT_TITLE", "", ""},
	{"invite_to_faction", "DIPLOMACY_OFFER_JOIN_FACTION_TITLE", "", ""},
	{"licences", "DIPLOMACY_REQUEST_ACCESS_TO_LICENCE_PRODUCTION_TITLE", "", ""},
	{"lend_lease", "DIPLOMACY_LEND_LEASE_TITLE", "", ""},
	{"request_lend_lease", "DIPLOMACY_INCOMING_LEND_LEASE_TITLE", "", ""},
	{"embargo", "DIPLOMACY_EMBARGO_TITLE", "NDiplomacy.EMBARGO_COST", ""},
	{"send_volunteers", "DIPLOMACY_SEND_VOLUNTEERS_TITLE", "", ""},
}

func (idx *Index) diplomaticActions() []map[string]any {
	out := []map[string]any{}
	for _, action := range diplomaticActionList {
		entry := map[string]any{"id": action.ID, "label": idx.localiseOr(action.Title, action.Title)}
		if action.PP != "" {
			entry["pp"] = idx.defineNumber(action.PP)
		}
		if action.CP != "" {
			entry["cp"] = idx.defineNumber(action.CP)
		}
		out = append(out, entry)
	}
	return out
}

func (idx *Index) DiplomacyPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))

	owners := map[string]bool{}
	for _, state := range idx.effectiveStates() {
		if owner := idx.ownerOfState(state, date); owner != "" {
			owners[owner] = true
		}
	}

	row := func(other string) map[string]any {
		party := idx.RulingPartyOf(other)
		item := map[string]any{"tag": other}
		if name, key := idx.countryNamed(other, party); name != "" {
			item["name"] = name

			item["name_key"] = key
		}

		if flag := idx.flagPayload(other, party, "medium"); flag != nil {
			item["flag"] = flag
		}
		if party != "" {
			item["ideology"] = party
		}
		if name, icon, leader := idx.FactionOf(other, date); name != "" {
			item["faction"] = name
			if icon != "" {
				item["faction_icon"] = icon
			}
			item["faction_leader"] = leader
		}
		if overlord, _ := idx.overlordShare(other); overlord != "" {
			item["overlord"] = overlord
		}
		return item
	}

	countries := make([]map[string]any, 0, len(owners))
	for other := range owners {
		if other == tag {
			continue
		}
		countries = append(countries, row(other))
	}
	sort.Slice(countries, func(a, b int) bool {
		na, _ := countries[a]["name"].(string)
		nb, _ := countries[b]["name"].(string)
		if na != nb {
			return na < nb
		}
		return countries[a]["tag"].(string) < countries[b]["tag"].(string)
	})

	self := row(tag)

	if politics := idx.countryPoliticsFor(tag); politics != nil {
		if politics.RulingParty != "" {
			self["party"] = politics.RulingParty

			if label, ok := idx.Localise(politics.RulingParty); ok {
				self["party_label"] = label
			}

			if name, ok := idx.Localise(tag + "_" + politics.RulingParty + "_party"); ok {
				self["party_name"] = name
			}
		}
		if politics.HasElections {
			self["elections"] = politics.ElectionsAllowed
		}
	}
	out := map[string]any{"tag": tag, "countries": countries, "self": self, "autonomy_states": idx.AutonomyLevels()}

	out["actions"] = idx.diplomaticActions()
	out["texts"] = map[string]any{
		"intel_ledger":  idx.localiseOr("INTEL_LEDGER", "Intel Ledger"),
		"unknown_focus": idx.localiseOr("UNKNOWN_INFO", "Unknown Focus"),

		"select_focus": idx.localiseOr("POLITICS_SELECT_FOCUS", "Select a National Focus"),
		"no_faction":   stripColour(idx.localiseOr("DIPLOMACY_NO_FACTION", "No faction")),
	}

	if person := idx.countryLeader(tag, date); person != nil {
		if leader := idx.advisorName(person); leader != "" {
			out["leader"] = leader

			out["leader_token"] = person.Token
		}
	}
	return out
}
