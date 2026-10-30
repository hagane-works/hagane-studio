package rawfiles

import "haganeworks/internal/hoi4/paradox"

type refExtractor func(e *paradox.Entry) []string

func scalarStr(e *paradox.Entry) string { return e.Scalar.StrValue() }

func extScalar(e *paradox.Entry) []string {
	if e.Block == nil {
		return []string{scalarStr(e)}
	}
	return nil
}

func extScalarOrList(e *paradox.Entry) []string {
	if e.Block == nil {
		return []string{scalarStr(e)}
	}
	var out []string
	for _, en := range e.Block.Entries {
		if en.Key == nil && en.Block == nil {
			out = append(out, scalarStr(en))
		}
	}
	return out
}

func extBlockWithField(field string) refExtractor {
	return func(e *paradox.Entry) []string {
		var out []string
		if e.Block == nil {
			return nil
		}
		for _, en := range e.Block.Entries {
			if en.Key != nil && *en.Key == field && en.Block == nil {
				out = append(out, scalarStr(en))
			}
		}
		return out
	}
}

func extBlockWithAnyField(fields ...string) refExtractor {
	accepted := map[string]bool{}
	for _, f := range fields {
		accepted[f] = true
	}
	return func(e *paradox.Entry) []string {
		var out []string
		if e.Block == nil {
			return nil
		}
		for _, en := range e.Block.Entries {
			if en.Key != nil && accepted[*en.Key] && en.Block == nil {
				out = append(out, scalarStr(en))
			}
		}
		return out
	}
}

func extScalarOrBlockField(field string) refExtractor {
	blockExt := extBlockWithField(field)
	return func(e *paradox.Entry) []string {
		if e.Block == nil {
			return []string{scalarStr(e)}
		}
		return blockExt(e)
	}
}

func extSwapPair(e *paradox.Entry) []string {
	var out []string
	if e.Block == nil {
		return nil
	}
	for _, en := range e.Block.Entries {
		if en.Key != nil && en.Block == nil {
			switch *en.Key {
			case "add_idea", "remove_idea", "add", "remove":
				out = append(out, scalarStr(en))
			}
		}
	}
	return out
}

func extEventScalarOrBlock(e *paradox.Entry) []string {
	if e.Block == nil {
		return []string{scalarStr(e)}
	}
	var out []string
	for _, en := range e.Block.Entries {
		if en.Key != nil && *en.Key == "id" && en.Block == nil {
			out = append(out, scalarStr(en))
		}
	}
	return out
}

func extBlockKeys(e *paradox.Entry) []string {
	if e.Block == nil {
		return nil
	}
	var out []string
	for _, en := range e.Block.Entries {
		if en.Key == nil || *en.Key == "popup" {
			continue
		}
		out = append(out, *en.Key)
	}
	return out
}

func extSwapTraits(e *paradox.Entry) []string {
	var out []string
	if e.Block == nil {
		return nil
	}
	for _, en := range e.Block.Entries {
		if en.Key != nil && en.Block == nil && (*en.Key == "add" || *en.Key == "remove") {
			out = append(out, scalarStr(en))
		}
	}
	return out
}

func extCreateCountryLeaderTraits(e *paradox.Entry) []string {
	var out []string
	if e.Block == nil {
		return nil
	}
	for _, en := range e.Block.Entries {
		if en.Key != nil && *en.Key == "traits" && en.Block != nil {
			for _, sub := range en.Block.Entries {
				if sub.Key == nil && sub.Block == nil {
					out = append(out, scalarStr(sub))
				}
			}
		}
	}
	return out
}

type refSpec struct {
	targetType string
	extract    refExtractor
}

var activeCrossRefs = map[string][]refSpec{
	"add_ideas":                     []refSpec{{"idea", extScalarOrList}},
	"remove_ideas":                  []refSpec{{"idea", extScalarOrList}},
	"swap_ideas":                    []refSpec{{"idea", extSwapPair}},
	"add_timed_idea":                []refSpec{{"idea", extBlockWithField("idea")}},
	"has_idea":                      []refSpec{{"idea", extScalar}},
	"add_dynamic_modifier":          []refSpec{{"dynamic_modifier", extBlockWithField("modifier")}},
	"remove_dynamic_modifier":       []refSpec{{"dynamic_modifier", extBlockWithField("modifier")}},
	"has_dynamic_modifier":          []refSpec{{"dynamic_modifier", extScalarOrBlockField("modifier")}},
	"set_autonomy":                  []refSpec{{"autonomy_state", extBlockWithAnyField("autonomy_state", "autonomous_state")}},
	"has_autonomy_state":            []refSpec{{"autonomy_state", extScalar}},
	"country_event":                 []refSpec{{"event", extEventScalarOrBlock}},
	"state_event":                   []refSpec{{"event", extEventScalarOrBlock}},
	"news_event":                    []refSpec{{"event", extEventScalarOrBlock}},
	"unit_leader_event":             []refSpec{{"event", extEventScalarOrBlock}},
	"complete_national_focus":       []refSpec{{"focus", extScalarOrBlockField("focus")}},
	"has_completed_focus":           []refSpec{{"focus", extScalar}},
	"focus_progress":                []refSpec{{"focus", extBlockWithField("focus")}},
	"activate_decision":             []refSpec{{"decision", extScalar}},
	"unlock_decision_tooltip":       []refSpec{{"decision", extScalarOrBlockField("decision")}},
	"has_trait":                     []refSpec{{"trait", extScalar}},
	"has_country_leader_with_trait": []refSpec{{"trait", extScalar}},
	"add_country_leader_trait":      []refSpec{{"trait", extScalarOrBlockField("trait")}},
	"remove_country_leader_trait":   []refSpec{{"trait", extScalarOrBlockField("trait")}},
	"add_unit_leader_trait":         []refSpec{{"trait", extScalar}},
	"remove_unit_leader_trait":      []refSpec{{"trait", extScalar}},
	"swap_ruler_traits":             []refSpec{{"trait", extSwapTraits}},
	"create_country_leader":         []refSpec{{"trait", extCreateCountryLeaderTraits}},
	"add_opinion_modifier":          []refSpec{{"opinion_modifier", extBlockWithField("modifier")}},
	"remove_opinion_modifier":       []refSpec{{"opinion_modifier", extBlockWithField("modifier")}},
	"reverse_add_opinion_modifier":  []refSpec{{"opinion_modifier", extBlockWithField("modifier")}},
	"has_opinion_modifier":          []refSpec{{"opinion_modifier", extScalarOrBlockField("modifier")}},
	"has_tech":                      []refSpec{{"technology", extScalar}},
	"can_research":                  []refSpec{{"technology", extScalar}},
	"set_technology":                []refSpec{{"technology", extBlockKeys}},
	"has_character":                 []refSpec{{"character", extScalar}},
	"is_character":                  []refSpec{{"character", extScalar}},
	"retire_character":              []refSpec{{"character", extScalar}},
	"promote_character":             []refSpec{{"character", extScalarOrBlockField("character")}},
	"remove_country_leader_role":    []refSpec{{"character", extScalarOrBlockField("character")}},
	"remove_unit_leader_role":       []refSpec{{"character", extScalarOrBlockField("character")}},
	"add_country_leader_role":       []refSpec{{"character", extBlockWithField("character")}},
	"add_corps_commander_role":      []refSpec{{"character", extBlockWithField("character")}},
	"add_trait": []refSpec{
		{"character", extBlockWithField("character")},
		{"trait", extBlockWithField("trait")},
	},
	"add_mio_funds":                               []refSpec{{"mio", extBlockWithField("mio")}},
	"add_mio_size_up":                             []refSpec{{"mio", extBlockWithField("mio")}},
	"add_mio_design_team_assign_cost":             []refSpec{{"mio", extBlockWithField("mio")}},
	"add_mio_industrial_manufacturer_assign_cost": []refSpec{{"mio", extBlockWithField("mio")}},
}
