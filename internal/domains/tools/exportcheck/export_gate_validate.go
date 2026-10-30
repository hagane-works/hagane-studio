package exportcheck

import (
	"encoding/json"
	"fmt"
	"haganeworks/internal/hoi4/emit/locout"
	"haganeworks/internal/hoi4/exportengine"
	"strings"
	"unicode/utf8"
)

type ValidateInput struct {
	ProjectID        int64
	ProjectName      string
	CountryTag       string
	Characters       []ValidateCharacter
	Trees            []ValidateFocusTree
	Techs            []ValidateTech
	Ideas            []ValidateIdea
	Events           []ValidateEvent
	Decisions        []ValidateDecision
	StageStateIDs    []int64
	IdeologyRegistry []byte
	Dependencies     []string
}

type ValidateCharacter struct {
	ID                int64
	Tag               string
	CountryTag        string
	RoleType          string
	Traits            []string
	AdvisorTraits     []string
	Source            string
	ReplacesVanillaID bool

	Ideology string

	DisplayNameLocs map[string]any
	NameEn, NameZh  string
	Name            string
}

type ValidateFocusNode struct {
	ID                int64
	FocusID           string
	Prerequisites     [][]string
	MutuallyExclusive []string
}

type ValidateFocusTree struct {
	ID       int64
	TreeID   string
	TreeKind string
	Nodes    []ValidateFocusNode
}

type ValidateTech struct {
	ID            int64
	TechID        string
	Prerequisites []string
}

type ValidateIdea struct {
	ID               int64
	IdeaID           string
	HasModifier      bool
	IsDesigner       bool
	HasRawExtras     bool
	HasResearchBonus bool
}

type ValidateEvent struct {
	ID          int64
	EventID     string
	NamespaceID *int64
	OptionCount int
	Hidden      bool
}

type ValidateDecision struct {
	ID         int64
	DecisionID string
}

var militaryRoles = map[string]bool{
	"general": true, "field_marshal": true, "admiral": true,
}

func newValidateIssue(level, code string, entityRef any, detail map[string]any,
	message string) map[string]any {
	if detail == nil {
		detail = map[string]any{}
	}
	return map[string]any{
		"level": level, "code": code, "entity_ref": entityRef,
		"detail": detail, "message": message,
	}
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func validateProjectCore(in ValidateInput) []map[string]any {
	issues := []map[string]any{}

	if in.ProjectName != "" && !isASCII(in.ProjectName) {
		utag := strings.ToUpper(in.CountryTag)
		fallback := fmt.Sprintf("mod_%d", in.ProjectID)
		if utag != "" {
			fallback = "mod_" + utag
		}
		folder := exportengine.SanitizeModDirName(in.ProjectName, fallback)
		issues = append(issues, newValidateIssue("warning", "mod_folder_name", nil,
			map[string]any{"name": in.ProjectName, "folder": folder},
			fmt.Sprintf("Mod folder name: project name '%s' is not ASCII — "+
				"HOI4 needs an ASCII mod path, so the exported folder / .mod will "+
				"be '%s' (the launcher display name stays '%s'). "+
				"Set an ASCII project name to control the folder name.",
				in.ProjectName, folder, in.ProjectName)))
	}

	for _, char := range in.Characters {
		if characterShowsRawTag(char) {
			issues = append(issues, newValidateIssue("error", "character_missing_name",
				map[string]any{"type": "character", "id": char.ID},
				map[string]any{"tag": char.Tag},
				fmt.Sprintf("Character %s: no display name in any language — "+
					"the export writes no name line, so HOI4 shows the raw tag "+
					"'%s' in-game.", char.Tag, char.Tag)))
		}
		if len(char.Traits) == 0 && militaryRoles[char.RoleType] {
			issues = append(issues, newValidateIssue("warning", "character_no_military_traits",
				map[string]any{"type": "character", "id": char.ID},
				map[string]any{"tag": char.Tag},
				fmt.Sprintf("Character %s: no traits selected", char.Tag)))
		}

		if char.RoleType == "advisor" && len(char.Traits) == 0 && len(char.AdvisorTraits) > 0 {
			issues = append(issues, newValidateIssue("warning",
				"character_advisor_traits_fallback",
				map[string]any{"type": "character", "id": char.ID},
				map[string]any{"tag": char.Tag},
				fmt.Sprintf("Character %s: advisor traits filled but primary traits empty "+
					"— using advisor_traits", char.Tag)))
		}
	}

	allFocusIDs := map[string]bool{}
	focusMutex := map[string]map[string]bool{}
	focusNodesByID := map[string]ValidateFocusNode{}
	for _, t := range in.Trees {
		for _, n := range t.Nodes {
			allFocusIDs[n.FocusID] = true
			focusNodesByID[n.FocusID] = n
			if focusMutex[n.FocusID] == nil {
				focusMutex[n.FocusID] = map[string]bool{}
			}
			for _, other := range n.MutuallyExclusive {
				if other == "" {
					continue
				}
				focusMutex[n.FocusID][other] = true
				if focusMutex[other] == nil {
					focusMutex[other] = map[string]bool{}
				}
				focusMutex[other][n.FocusID] = true
			}
		}
	}
	for _, tree := range in.Trees {
		if len(tree.Nodes) == 0 {
			issues = append(issues, newValidateIssue("warning", "focus_tree_no_nodes",
				map[string]any{"type": "focus", "id": tree.ID},
				map[string]any{"tree_id": tree.TreeID},
				fmt.Sprintf("Focus tree '%s': no nodes", tree.TreeID)))
		}
		refLevel := "error"
		if tree.TreeKind == "shared_pool" {
			refLevel = "warning"
		}
		for _, node := range tree.Nodes {
			if node.FocusID == "" {
				issues = append(issues, newValidateIssue("error", "focus_missing_id",
					map[string]any{"type": "focus", "id": node.ID, "tree_id": tree.ID},
					map[string]any{"node_id": node.ID},
					fmt.Sprintf("Focus node ID=%d: missing focus_id", node.ID)))
			}
			for _, group := range node.Prerequisites {
				for _, prereq := range group {
					if prereq != "" && !allFocusIDs[prereq] {
						issues = append(issues, newValidateIssue(refLevel,
							"focus_prereq_not_found",
							map[string]any{"type": "focus", "id": node.ID, "tree_id": tree.ID},
							map[string]any{"focus_id": node.FocusID, "prereq": prereq},
							fmt.Sprintf("Focus '%s': prerequisite '%s' not found in project",
								node.FocusID, prereq)))
					}
				}
			}
			for _, me := range node.MutuallyExclusive {
				if me != "" && !allFocusIDs[me] {
					issues = append(issues, newValidateIssue(refLevel, "focus_mutex_not_found",
						map[string]any{"type": "focus", "id": node.ID, "tree_id": tree.ID},
						map[string]any{"focus_id": node.FocusID, "mutex": me},
						fmt.Sprintf("Focus '%s': mutually_exclusive '%s' not found in project",
							node.FocusID, me)))
				}
			}
			if len(node.Prerequisites) > 1 && focusPrerequisitesKnown(node.Prerequisites, allFocusIDs) {
				playable, conflicts := focusPrerequisitesSatisfiable(node.Prerequisites, focusNodesByID, focusMutex)
				if playable || len(conflicts) == 0 {
					continue
				}
				issues = append(issues, newValidateIssue("error", "focus_prereq_mutex_conflict",
					map[string]any{"type": "focus", "id": node.ID, "tree_id": tree.ID},
					map[string]any{"focus_id": node.FocusID, "prerequisites": node.Prerequisites,
						"conflicting_focus_ids": conflicts},
					fmt.Sprintf("Focus '%s': prerequisite AND-groups are mutually exclusive, so this focus can never unlock; put alternative branches in one OR group", node.FocusID)))
			}
		}
	}

	techIDs := map[string]bool{}
	for _, t := range in.Techs {
		techIDs[t.TechID] = true
	}
	for _, tech := range in.Techs {
		if tech.TechID == "" {
			issues = append(issues, newValidateIssue("error", "tech_missing_id",
				map[string]any{"type": "technology", "id": tech.ID}, nil,
				fmt.Sprintf("Technology ID=%d: missing tech_id", tech.ID)))
		}
		for _, prereq := range tech.Prerequisites {
			if prereq != "" && !techIDs[prereq] {
				issues = append(issues, newValidateIssue("warning", "tech_prereq_not_found",
					map[string]any{"type": "technology", "id": tech.ID},
					map[string]any{"tech_id": tech.TechID, "prereq": prereq},
					fmt.Sprintf("Tech '%s': prerequisite '%s' not found (may be vanilla tech)",
						tech.TechID, prereq)))
			}
		}
	}

	for _, idea := range in.Ideas {
		if idea.IdeaID == "" {
			issues = append(issues, newValidateIssue("error", "idea_missing_id",
				map[string]any{"type": "idea", "id": idea.ID}, nil,
				fmt.Sprintf("Idea ID=%d: missing idea_id", idea.ID)))
		}

		if !idea.HasModifier && !idea.IsDesigner && !idea.HasRawExtras && !idea.HasResearchBonus {
			issues = append(issues, newValidateIssue("warning", "idea_no_modifier",
				map[string]any{"type": "idea", "id": idea.ID},
				map[string]any{"idea_id": idea.IdeaID},
				fmt.Sprintf("Idea '%s': no modifier defined", idea.IdeaID)))
		}
	}

	for _, ev := range in.Events {
		ref := map[string]any{"type": "event", "id": ev.ID}
		if ev.NamespaceID != nil {
			ref["namespace_id"] = *ev.NamespaceID
		} else {
			ref["namespace_id"] = nil
		}
		if ev.EventID == "" {
			issues = append(issues, newValidateIssue("error", "event_missing_id",
				copyAnyMap(ref), nil,
				fmt.Sprintf("Event ID=%d: missing event_id", ev.ID)))
		}

		if ev.OptionCount == 0 && !ev.Hidden {
			issues = append(issues, newValidateIssue("warning", "event_no_options",
				copyAnyMap(ref), map[string]any{"event_id": ev.EventID},
				fmt.Sprintf("Event '%s': no options defined", ev.EventID)))
		}
	}

	for _, dec := range in.Decisions {
		if dec.DecisionID == "" {
			issues = append(issues, newValidateIssue("error", "decision_missing_id",
				map[string]any{"type": "decision", "id": dec.ID}, nil,
				fmt.Sprintf("Decision ID=%d: missing decision_id", dec.ID)))
		}
	}

	return issues
}

func focusPrerequisitesKnown(groups [][]string, known map[string]bool) bool {
	for _, group := range groups {
		if len(group) == 0 {
			return false
		}
		for _, focusID := range group {
			if focusID == "" || !known[focusID] {
				return false
			}
		}
	}
	return true
}

func focusPrerequisitesSatisfiable(groups [][]string, nodes map[string]ValidateFocusNode,
	mutex map[string]map[string]bool) (bool, [][2]string) {
	const maxSearchSteps = 50_000
	remaining, exhausted := maxSearchSteps, false
	type state map[string]bool
	clone := func(in state) state {
		out := make(state, len(in))
		for key, value := range in {
			out[key] = value
		}
		return out
	}
	mergeConflicts := func(dst [][2]string, src [][2]string) [][2]string {
		seen := map[[2]string]bool{}
		for _, pair := range dst {
			seen[pair] = true
		}
		for _, pair := range src {
			if pair[1] < pair[0] {
				pair[0], pair[1] = pair[1], pair[0]
			}
			if !seen[pair] {
				dst = append(dst, pair)
				seen[pair] = true
			}
		}
		return dst
	}

	var require func(string, state, state) (state, bool, [][2]string)
	var satisfy func([][]string, int, state, state) (state, bool, [][2]string)
	satisfy = func(required [][]string, index int, selected, visiting state) (state, bool, [][2]string) {
		remaining--
		if remaining < 0 {
			exhausted = true
			return nil, false, nil
		}
		if index == len(required) {
			return selected, true, nil
		}
		conflicts := [][2]string{}
		for _, candidate := range required[index] {
			afterCandidate, ok, candidateConflicts := require(candidate, clone(selected), clone(visiting))
			conflicts = mergeConflicts(conflicts, candidateConflicts)
			if !ok {
				continue
			}
			afterGroups, ok, groupConflicts := satisfy(required, index+1, afterCandidate, visiting)
			conflicts = mergeConflicts(conflicts, groupConflicts)
			if ok {
				return afterGroups, true, conflicts
			}
		}
		return nil, false, conflicts
	}
	require = func(focusID string, selected, visiting state) (state, bool, [][2]string) {
		remaining--
		if remaining < 0 {
			exhausted = true
			return nil, false, nil
		}
		if visiting[focusID] {
			return nil, false, nil
		}
		if selected[focusID] {
			return selected, true, nil
		}
		for existing := range selected {
			if focusID != existing && (mutex[focusID][existing] || mutex[existing][focusID]) {
				return nil, false, [][2]string{{focusID, existing}}
			}
		}
		node, exists := nodes[focusID]
		if !exists {
			return nil, false, nil
		}
		selected[focusID] = true
		visiting[focusID] = true
		return satisfy(node.Prerequisites, 0, selected, visiting)
	}

	_, ok, conflicts := satisfy(groups, 0, state{}, state{})
	if exhausted {

		return true, nil
	}
	return ok, conflicts
}

func copyAnyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func characterShowsRawTag(char ValidateCharacter) bool {
	source := char.Source
	if source == "" {
		source = "created"
	}
	if source != "created" {
		return false
	}

	payload := map[string]any{
		"tag": char.Tag, "country_tag": char.CountryTag,
		"replaces_vanilla_id": char.ReplacesVanillaID,
		"name_en":             char.NameEn, "name_zh": char.NameZh, "name": char.Name,
		"display_name_locs": char.DisplayNameLocs,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	v, err := locout.ParseJSON(raw)
	if err != nil || v.Kind != locout.KObj {
		return false
	}
	return len(locout.DetectCharacterNameFallback([]*locout.Obj{v.Obj})) > 0
}
