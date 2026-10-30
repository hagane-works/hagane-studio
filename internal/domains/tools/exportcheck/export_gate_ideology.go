package exportcheck

import (
	"encoding/json"
	"fmt"
	"haganeworks/internal/hoi4/clausewitz"
	"haganeworks/internal/hoi4/paradox"
	"regexp"
	"strings"
)

var rootIDRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var maxRootBodyChars = 100000

type ideologyRootDef struct {
	ID   string
	Body string
}

type ideologySub struct {
	ID    string
	Group string
}

type normalizedIdeologyRegistry struct {
	Groups []string
	Roots  []ideologyRootDef
	Subs   []ideologySub
}

func normalizeIdeologyRegistry(raw []byte) normalizedIdeologyRegistry {
	out := normalizedIdeologyRegistry{Groups: []string{}, Roots: []ideologyRootDef{},
		Subs: []ideologySub{}}
	if len(raw) == 0 {
		return out
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil || doc == nil {
		return out
	}
	if groups, ok := doc["groups"].([]any); ok {
		for _, g := range groups {
			if s, ok := g.(string); ok && strings.TrimSpace(s) != "" {
				out.Groups = append(out.Groups, strings.TrimSpace(s))
			}
		}
	}
	if roots, ok := doc["root_definitions"].([]any); ok {
		for _, r := range roots {
			obj, ok := r.(map[string]any)
			if !ok {
				continue
			}
			id, ok := obj["id"].(string)
			if !ok {
				continue
			}
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			body := ""
			if b, ok := obj["body"].(string); ok {
				body = strings.TrimSpace(b)
			}
			out.Roots = append(out.Roots, ideologyRootDef{ID: id, Body: body})

			found := false
			for _, g := range out.Groups {
				if g == id {
					found = true
					break
				}
			}
			if !found {
				out.Groups = append(out.Groups, id)
			}
		}
	}
	if subs, ok := doc["sub_ideologies"].([]any); ok {
		for _, s := range subs {
			switch v := s.(type) {
			case string:
				out.Subs = append(out.Subs, ideologySub{ID: v, Group: ""})
			case map[string]any:
				id, ok := v["id"].(string)
				if !ok || strings.TrimSpace(id) == "" {
					continue
				}
				group := ""
				if g, ok := v["group"].(string); ok {
					group = strings.TrimSpace(g)
				}
				out.Subs = append(out.Subs, ideologySub{
					ID: strings.TrimSpace(id), Group: group})
			}
		}
	}
	return out
}

func (n normalizedIdeologyRegistry) customRootDefinitions() []ideologyRootDef {
	out := []ideologyRootDef{}
	for _, r := range n.Roots {
		if !vanillaIdeologyGroupIDs()[r.ID] {
			out = append(out, r)
		}
	}
	return out
}

func EffectiveGroupIDs(raw []byte) map[string]bool {
	out := map[string]bool{}
	for g := range vanillaIdeologyGroupIDs() {
		out[g] = true
	}
	for _, g := range normalizeIdeologyRegistry(raw).Groups {
		out[g] = true
	}
	return out
}

func EffectiveSubIDs(raw []byte) map[string]bool {
	out := vanillaSubIdeologyIDs()
	for _, s := range normalizeIdeologyRegistry(raw).Subs {
		out[s.ID] = true
	}
	return out
}

func InvalidStandaloneSubIdeologies(raw []byte) []map[string]any {
	n := normalizeIdeologyRegistry(raw)
	supported := map[string]bool{}
	for g := range vanillaIdeologyGroupIDs() {
		supported[g] = true
	}
	for _, r := range n.customRootDefinitions() {
		supported[r.ID] = true
	}
	vanillaSubs := vanillaSubIdeologyIDs()
	invalid := []map[string]any{}
	seen := map[string]bool{}
	for _, entry := range n.Subs {
		if vanillaSubs[entry.ID] || seen[entry.ID] {
			continue
		}
		seen[entry.ID] = true
		if !supported[entry.Group] {
			invalid = append(invalid, map[string]any{"id": entry.ID, "group": entry.Group})
		}
	}
	return invalid
}

func paradoxBlock(entries []*paradox.Entry, key string) *paradox.Block {
	for _, e := range entries {
		if e.Key != nil && *e.Key == key && e.Block != nil {
			return e.Block
		}
	}
	return nil
}

func InvalidCustomRootDefinitions(raw []byte) []map[string]any {
	invalid := []map[string]any{}
	seen := map[string]bool{}
	for _, root := range normalizeIdeologyRegistry(raw).Roots {
		rootID, body := root.ID, root.Body
		reason := ""
		switch {
		case !rootIDRe.MatchString(rootID):
			reason = "id must start with a letter or underscore and use only letters, numbers, and underscores"
		case vanillaIdeologyGroupIDs()[rootID]:
			reason = "vanilla ROOT groups cannot be redefined"
		case seen[rootID]:
			reason = "duplicate ROOT id"
		case body == "":
			reason = "group body is required"
		case len([]rune(body)) > maxRootBodyChars:
			reason = fmt.Sprintf("group body exceeds %d characters", maxRootBodyChars)
		default:
			wrapped := "ideologies = {\n" + rootID + " = {\n" + body + "\n}\n}\n"
			block, recoveries, err := paradox.Parse(wrapped)

			if failed, msg := clausewitz.ParseErrorMessage(recoveries, err); failed {
				reason = "group body is not valid HOI4 script: " + msg
				break
			}
			outer := paradoxBlock(block.Entries, "ideologies")
			var keyed []*paradox.Entry
			if outer != nil {
				for _, e := range outer.Entries {
					if e.Key != nil {
						keyed = append(keyed, e)
					}
				}
			}
			switch {
			case len(block.Entries) != 1 || outer == nil || len(keyed) != 1 ||
				*keyed[0].Key != rootID || keyed[0].Block == nil:
				reason = "body must stay inside exactly one ROOT group"
			case paradoxBlock(keyed[0].Block.Entries, "types") == nil:
				reason = "group body must contain a types = { ... } block"
			case paradoxBlock(keyed[0].Block.Entries, "color") == nil:
				reason = "group body must contain a color = { r g b } block"
			}
		}
		if reason != "" {
			invalid = append(invalid, map[string]any{"id": rootID, "reason": reason})
		}
		seen[rootID] = true
	}
	return invalid
}

type ValidateIdeologyRef struct {
	Value string
	Who   string
	Ref   map[string]any
}

func validateIdeologyChecks(registry []byte, isSubmod bool,
	refs []ValidateIdeologyRef) []map[string]any {

	issues := []map[string]any{}

	if !isSubmod {
		for _, inv := range InvalidStandaloneSubIdeologies(registry) {
			sid, _ := inv["id"].(string)
			group, _ := inv["group"].(string)
			if group == "" {
				group = "missing"
			}
			issues = append(issues, map[string]any{
				"level": "error", "code": "ideology_registry_parent_missing",
				"entity_ref": nil,
				"detail":     map[string]any{"id": sid, "group": group},
				"severity":   "blocker",
				"fix_hint": "Open the custom ideology registry and choose democratic, " +
					"communism, fascism, or neutrality as the parent group.",
				"message": fmt.Sprintf(
					"Custom ideology '%s' has parent group '%s', which a standalone export "+
						"cannot emit. Export is blocked instead of silently dropping the ideology.",
					sid, group),
			})
		}
	}

	for _, inv := range InvalidCustomRootDefinitions(registry) {
		id, _ := inv["id"].(string)
		reason, _ := inv["reason"].(string)
		shown := id
		if shown == "" {
			shown = "missing"
		}
		issues = append(issues, map[string]any{
			"level": "error", "code": "ideology_root_invalid",
			"entity_ref": nil, "detail": inv, "severity": "blocker",
			"fix_hint": "Open the ideology registry and repair the custom ROOT body. " +
				"It must be one group body containing types and color blocks.",
			"message": fmt.Sprintf(
				"Custom ROOT ideology '%s' is unsafe: %s. Export is blocked instead of "+
					"writing a broken common/ideologies replacement.", shown, reason),
		})
	}

	if isSubmod && len(normalizeIdeologyRegistry(registry).customRootDefinitions()) > 0 {
		issues = append(issues, map[string]any{
			"level": "error", "code": "ideology_root_submod_blocked",
			"entity_ref": nil, "severity": "blocker",
			"fix_hint": "Use a ROOT group supplied by the base mod, or remove the " +
				"dependency and export this project as a standalone mod.",
			"message": "Custom ROOT ideologies are standalone-only. Replacing " +
				"common/ideologies in a submod could erase the base mod's ROOT groups, " +
				"so export is blocked.",
		})
	}

	if !isSubmod {
		knownSubs := EffectiveSubIDs(registry)
		knownGroups := EffectiveGroupIDs(registry)
		warned := map[string]bool{}
		for _, r := range refs {
			x := strings.TrimSpace(r.Value)
			if x == "" || knownSubs[x] || knownGroups[x] || warned[x] {
				continue
			}
			warned[x] = true
			issues = append(issues, map[string]any{
				"level": "warning", "code": "ideology_undefined_standalone",
				"entity_ref": r.Ref, "severity": "warning",
				"fix_hint": fmt.Sprintf(
					"Define '%s' in the project's ideology registry, or point %s at a "+
						"vanilla ideology.", x, r.Who),
				"message": fmt.Sprintf(
					"Custom ideology '%s' is referenced by %s but not defined in the "+
						"project's ideology registry; a standalone mod must define it "+
						"(export emits common/ideologies only for registry entries).",
					x, r.Who),
			})
		}
	}
	return issues
}

func validateCountryLeaderIdeologies(registry []byte,
	characters []ValidateCharacter) []map[string]any {

	issues := []map[string]any{}
	knownRoots := EffectiveGroupIDs(registry)
	knownSubs := EffectiveSubIDs(registry)
	for _, character := range characters {
		if character.RoleType != "country_leader" {
			continue
		}
		ideology := strings.TrimSpace(character.Ideology)
		if ideology == "" || !knownRoots[ideology] || knownSubs[ideology] {
			continue
		}
		issues = append(issues, map[string]any{
			"level": "error", "code": "ideology_leader_not_subtype",
			"entity_ref": map[string]any{"type": "character", "id": character.Tag},
			"detail":     map[string]any{"ideology": ideology}, "severity": "blocker",
			"fix_hint": fmt.Sprintf("Choose a leader political subtype inside ROOT ideology '%s'; country leaders cannot use a ROOT directly.", ideology),
			"message": fmt.Sprintf("Country leader %s uses ROOT ideology '%s', but country_leader.ideology requires a political subtype.",
				character.Tag, ideology),
		})
	}
	return issues
}

func validateHistoryPopularities(registry []byte, isSubmod bool,
	historyFiles []ValidateHistoryFile) []map[string]any {

	issues := []map[string]any{}
	knownRoots := EffectiveGroupIDs(registry)
	knownSubs := EffectiveSubIDs(registry)
	for _, hf := range historyFiles {
		ref := map[string]any{"type": "history_file", "id": hf.CountryTag}

		ruling := strings.TrimSpace(hf.SetPoliticsRulingParty)
		if ruling != "" && knownSubs[ruling] && !knownRoots[ruling] {
			issues = append(issues, map[string]any{
				"level": "error", "code": "ideology_ruling_party_not_root",
				"entity_ref": ref, "detail": map[string]any{"ruling_party": ruling},
				"severity": "blocker",
				"fix_hint": fmt.Sprintf("Choose the ROOT ideology group that contains '%s'; country ruling_party cannot use a leader subtype.", ruling),
				"message": fmt.Sprintf("Country %s uses ideology subtype '%s' as ruling_party, but country history requires a ROOT ideology group.",
					hf.CountryTag, ruling),
			})
		}
		if hf.SetPopularitiesError != "" {
			issues = append(issues, map[string]any{
				"level": "error", "code": "ideology_popularity_invalid",
				"entity_ref": ref, "severity": "blocker",
				"fix_hint": "Set each ROOT support value to a whole number from 0 to 100, and make the total exactly 100.",
				"message": fmt.Sprintf("Country %s has an invalid set_popularities block: %s.",
					hf.CountryTag, hf.SetPopularitiesError),
			})
			continue
		}
		if isSubmod || len(hf.SetPopularities) == 0 {
			continue
		}
		for root := range hf.SetPopularities {
			if knownRoots[root] {
				continue
			}
			issues = append(issues, map[string]any{
				"level": "error", "code": "ideology_popularity_root_missing",
				"entity_ref": ref, "detail": map[string]any{"root": root},
				"severity": "blocker",
				"fix_hint": fmt.Sprintf("Define ROOT ideology '%s' in this project's registry, or replace it with an existing ROOT.", root),
				"message": fmt.Sprintf("Country %s assigns popularity to ROOT ideology '%s', but that ROOT is not defined by this standalone project.",
					hf.CountryTag, root),
			})
		}
	}
	return issues
}
