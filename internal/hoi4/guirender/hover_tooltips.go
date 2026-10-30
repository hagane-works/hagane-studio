package guirender

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

var tooltipNeutralModifiers = map[string]bool{
	"consumer_goods_factor": true, "consumer_goods_expected_value": true,
}

var tooltipBadWhenPositive = regexp.MustCompile(`cost|attrition|penalty|tension|upkeep|consumption|decay|_time|casualties`)

var tooltipGoodWhenPositive = map[string]bool{
	"enemy_justify_war_goal_time": true, "max_planning_factor": true,
}

func (idx *Index) tooltipModifierLine(key string, value float64) string {
	lower := strings.ToLower(key)
	percent := modifierReadsPercent(lower)
	decimals, ok := idx.modifierDecimalsOf(lower)
	if !ok {
		decimals = 2
		if percent {
			decimals = 1
		}
	}
	n := value
	if percent {
		n *= 100
	}
	text := fmt.Sprintf("%.*f", decimals, n)
	if percent {
		text += "%"
	}
	label := idx.modifierLabel(lower)
	if tooltipNeutralModifiers[lower] {
		return fmt.Sprintf("%s: §Y%s§!", label, text)
	}
	if n > 0 {
		text = "+" + text
	}
	bad := tooltipBadWhenPositive.MatchString(lower) && !tooltipGoodWhenPositive[lower]
	colour := "G"
	if (n > 0) == bad {
		colour = "R"
	}
	if n == 0 {
		colour = "Y"
	}
	return fmt.Sprintf("%s: §%s%s§!", label, colour, text)
}

func (idx *Index) tooltipModifierLines(block *paradox.Block) []string {
	if block == nil {
		return nil
	}
	out := []string{}
	for _, entry := range block.Entries {
		if entry == nil || entry.Key == nil || entry.Block != nil {
			continue
		}
		if value, ok := toFloat(scalarValue(entry.Scalar)); ok {
			out = append(out, idx.tooltipModifierLine(*entry.Key, value))
		}
	}
	return out
}

func (idx *Index) localisedOr(key string) string {
	if text, ok := idx.Localise(key); ok && strings.TrimSpace(text) != "" {
		return text
	}
	return key
}

func (idx *Index) IdeaTooltip(token string) string {
	lines := []string{"§H" + idx.localisedOr(token) + "§!"}
	block := idx.ideaModifierBlockFor(token)
	if block == nil {
		block = idx.dynamicModifierBlockFor(token)
	}
	if modifiers := idx.tooltipModifierLines(block); len(modifiers) > 0 {
		lines = append(lines, "")
		lines = append(lines, modifiers...)
	}
	if desc, ok := idx.LocaliseMarkup(token + "_desc"); ok && strings.TrimSpace(desc) != "" {
		lines = append(lines, "--------------", strings.TrimSpace(desc))
	}
	return strings.Join(lines, "\n")
}

func (idx *Index) AdvisorTooltip(name string, traits []string, cost int) string {
	lines := []string{idx.fillTooltipLoc("CHARACTER_NAME_TOOLTIP", "$NAME|Y$", map[string]any{"NAME": name})}
	modifiers := []string{}
	for _, trait := range traits {
		lines = append(lines, "§H"+stripColour(idx.localisedOr(trait))+"§!")
		block := idx.countryLeaderTraitFor(trait)
		if block == nil {
			continue
		}
		kept := &paradox.Block{}
		for _, entry := range block.Entries {
			if entry != nil && entry.Key != nil && advisorTraitSkip[strings.ToLower(*entry.Key)] {
				continue
			}
			kept.Entries = append(kept.Entries, entry)
		}
		modifiers = append(modifiers, idx.tooltipModifierLines(kept)...)
	}
	if len(modifiers) > 0 {
		lines = append(lines, "")
		lines = append(lines, modifiers...)
	}
	if cost > 0 {
		lines = append(lines, "", idx.fillTooltipLoc("POLITICS_ADVISOR_COST", "Cost (£pol_power ): $VALUE|-0$", map[string]any{"VALUE": cost}))
	}
	return strings.Join(lines, "\n")
}

func (idx *Index) ideaGrants(token string) string {
	return strings.Join(idx.tooltipModifierLines(idx.ideaModifierBlockFor(token)), ", ")
}

func (idx *Index) focusDef(id string) *Focus {
	for layer := idx; layer != nil; layer = layer.fallback {
		if focus, ok := layer.focusDefs[id]; ok {
			return focus
		}
	}
	return nil
}

func (idx *Index) focusRewardBlock(id string) []*paradox.Entry {
	for layer := idx; layer != nil; layer = layer.fallback {
		if block, ok := layer.focusRewardBlocks[id]; ok {
			return block
		}
	}
	return nil
}

func (idx *Index) FocusTooltip(id string) string {
	focus := idx.focusDef(id)
	if focus == nil {
		return ""
	}
	lines := []string{"§H" + idx.localisedOr(id) + "§!", ""}
	days := focus.Cost * idx.defineNumber("NFocus.FOCUS_POINT_DAYS")
	if days <= 0 {
		days = focus.Cost * 7
	}
	lines = append(lines, idx.fillTooltipLoc("FOCUS_STATUS_TOOLTIP", "Completion Time: $DAYS|0H$ days", map[string]any{"DAYS": days}))
	if requires := idx.focusRequiresLines(focus.Prereq); len(requires) > 0 || len(focus.Mutex) > 0 {
		lines = append(lines, "")
		lines = append(lines, requires...)
		if len(focus.Mutex) > 0 {
			names := make([]string, 0, len(focus.Mutex))
			for _, other := range focus.Mutex {
				names = append(names, "§Y"+idx.localisedOr(other)+"§!")
			}
			lines = append(lines, idx.localisedOrText("NATIONAL_FOCUS_EXCLUSIVE_INFO_DESC", "Mutually exclusive with")+" "+strings.Join(names, ", "))
		}
	}
	if effects := idx.effectTooltipLines(idx.focusRewardBlock(id)); len(effects) > 0 {
		lines = append(lines, "", idx.fillTooltipLoc("FOCUS_REWARD_TOOLTIP", "§HEffect§!:", nil))
		lines = append(lines, effects...)
	}
	return strings.Join(lines, "\n")
}

func (idx *Index) focusRequiresLines(prereq [][]string) []string {
	name := func(id string) map[string]any { return map[string]any{"FOCUS": idx.localisedOr(id)} }
	trim := func(text string) string { return strings.TrimRight(text, "\n") }
	switch {
	case len(prereq) == 0:
		return nil
	case len(prereq) == 1 && len(prereq[0]) == 1:
		return []string{idx.fillTooltipLoc("FOCUS_SINGLE_PREREQ", "§RRequires $FOCUS$§!", name(prereq[0][0]))}
	case len(prereq) == 1:
		out := []string{trim(idx.fillTooltipLoc("FOCUS_SINGLE_OR_PREREQ_HEADER", "§HRequires one of the following:\n§!", nil))}
		for _, id := range prereq[0] {
			out = append(out, trim(idx.fillTooltipLoc("FOCUS_SINGLE_OR_PREREQ", "§R- $FOCUS$\n§!", name(id))))
		}
		return out
	}
	out := []string{trim(idx.fillTooltipLoc("FOCUS_MIX_PREREQ_HEADER", "§HRequires all of the following:\n§!", nil))}
	for _, group := range prereq {
		if len(group) == 1 {
			out = append(out, trim(idx.fillTooltipLoc("FOCUS_MIX_AND_PREREQ", "§R- $FOCUS$\n§!", name(group[0]))))
			continue
		}
		out = append(out, trim(idx.fillTooltipLoc("FOCUS_MIX_OR_PREREQ_HEADER", "§R- One of the following:\n§!", nil)))
		for _, id := range group {
			out = append(out, trim(idx.fillTooltipLoc("FOCUS_MIX_OR_PREREQ", "     $FOCUS|R$\n", name(id))))
		}
	}
	return out
}

func (idx *Index) localisedOrText(key, fallback string) string {
	if text, ok := idx.LocaliseMarkup(key); ok && strings.TrimSpace(text) != "" {
		return text
	}
	return fallback
}

func (idx *Index) effectTooltipLines(entries []*paradox.Entry) []string {
	out := []string{}
	number := func(entry *paradox.Entry) (float64, bool) {
		if entry.Block != nil {
			return 0, false
		}
		return toFloat(scalarValue(entry.Scalar))
	}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		switch key {
		case "add_political_power":
			if v, ok := number(entry); ok {
				out = append(out, idx.fillTooltipLoc("POLITICS_ADD_POLITICAL_POWER", "Political Power: $VAL|=+0$.", map[string]any{"VAL": v}))
			}
		case "add_stability":
			if v, ok := number(entry); ok {
				out = append(out, idx.fillTooltipLoc("EFFECT_ADD_STABILITY", "Gain Base Stability: $VAL|+=%$.", map[string]any{"VAL": v}))
			}
		case "add_war_support":
			if v, ok := number(entry); ok {
				out = append(out, idx.fillTooltipLoc("EFFECT_ADD_WAR_SUPPORT", "Gain Base War Support: $VAL|+=%$.", map[string]any{"VAL": v}))
			}
		case "army_experience", "navy_experience", "air_experience":
			if v, ok := number(entry); ok {
				branch := strings.ToUpper(strings.TrimSuffix(key, "_experience"))
				out = append(out, idx.fillTooltipLoc("EFFECT_ADD_"+branch+"_EXPERIENCE", "Experience: $VAL|=+0$.", map[string]any{"VAL": v}))
			}
		case "add_command_power":
			if v, ok := number(entry); ok {
				out = append(out, idx.fillTooltipLoc("EFFECT_ADD_COMMAND_POWER", "Command Power: $VAL|=+1$.", map[string]any{"VAL": v}))
			}
		case "add_research_slot":
			if v, ok := number(entry); ok {
				out = append(out, idx.fillTooltipLoc("EFFECT_ADD_RESEARCH_SLOT", "Gain $COUNT|G$ Research Slot.", map[string]any{"COUNT": v}))
			}
		case "add_ideas":
			for _, token := range effectValues(entry) {
				out = append(out, idx.addIdeaLine(token))
			}
		case "remove_ideas":
			for _, token := range effectValues(entry) {
				out = append(out, idx.fillTooltipLoc("EFFECT_REMOVE_IDEA_NATIONAL_SPIRIT", "Remove National Spirit $NAME|Y$.", map[string]any{"NAME": idx.localisedOr(token)}))
			}
		case "add_timed_idea":
			if entry.Block != nil {
				token := ciScalar(entry.Block.Entries, "idea")
				days, _ := toFloat(ciScalarValue(entry.Block.Entries, "days"))
				if token != "" {
					out = append(out, strings.TrimRight(idx.fillTooltipLoc("EFFECT_ADD_TIMED_IDEA", "Gains '$IDEA|H$' which grants ($DESC$) for $TIME_DESC$.\n", map[string]any{
						"IDEA": idx.localisedOr(token), "DESC": idx.ideaGrants(token), "TIME_DESC": fmt.Sprintf("%d days", int(days)),
					}), "\n"))
				}
			}
		case "add_popularity":
			if entry.Block != nil {
				ideology := ciScalar(entry.Block.Entries, "ideology")
				v, ok := toFloat(ciScalarValue(entry.Block.Entries, "popularity"))
				if ideology != "" && ok {
					label := idx.localisedOr(ideology + "_noun")
					if label == ideology+"_noun" {
						label = idx.localisedOr(ideology)
					}
					out = append(out, idx.fillTooltipLoc("POLITICS_IDEOLOGY_POPULARITY_CHANGE", "Change in popularity of $IDEOLOGY$: $VALUE|+2%$", map[string]any{"IDEOLOGY": label, "VALUE": v}))
				}
			}
		case "country_event":
			id := ""
			if entry.Block != nil {
				id = ciScalar(entry.Block.Entries, "id")
			} else {
				id = entry.Scalar.StrValue()
			}
			if id != "" {
				title := idx.localisedOr(id + ".t")
				out = append(out, strings.TrimRight(idx.fillTooltipLoc("EFFECT_COUNTRY_EVENT", "Gets event '$EVENT|Y$'.\n", map[string]any{"EVENT": title}), "\n"))
			}
		case "custom_effect_tooltip":
			if entry.Block == nil {
				if text, ok := idx.LocaliseMarkup(entry.Scalar.StrValue()); ok && strings.TrimSpace(text) != "" {
					out = append(out, strings.TrimRight(text, "\n"))
				}
			}
		}
	}
	return out
}

func (idx *Index) addIdeaLine(token string) string {
	name := idx.localisedOr(token)
	grants := idx.ideaGrants(token)
	if grants == "" {
		return idx.fillTooltipLoc("EFFECT_ADD_IDEA", "Add $NAME$.", map[string]any{"NAME": "§Y" + name + "§!"})
	}
	if idx.ideaCategoryOf(token) == "country" {
		return idx.fillTooltipLoc("EFFECT_ADD_IDEA_NATIONAL_SPIRIT_TOOLTIP_DETAILED", "Gains National Spirit $NAME|Y$, which grants ($DESC$).", map[string]any{"NAME": name, "DESC": grants})
	}
	return idx.fillTooltipLoc("EFFECT_ADD_IDEA_DETAILED", "Gains '$NAME|Y$' which grants ($DESC$).", map[string]any{"NAME": name, "DESC": grants})
}

func effectValues(entry *paradox.Entry) []string {
	if entry.Block == nil {
		if value := strings.TrimSpace(entry.Scalar.StrValue()); value != "" {
			return []string{value}
		}
		return nil
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
		if value := strings.TrimSpace(toString(scalarValue(item.Scalar))); value != "" {
			out = append(out, value)
		}
	}
	return out
}

var tooltipSlotRE = regexp.MustCompile(`\$([A-Za-z0-9_]+)(?:\|([^$\n]*))?\$`)

func (idx *Index) fillTooltipLoc(key, fallback string, vars map[string]any) string {
	text := fallback
	if value, ok := idx.LocaliseMarkup(key); ok && strings.TrimSpace(value) != "" {
		text = value
	}
	return tooltipSlotRE.ReplaceAllStringFunc(text, func(match string) string {
		parts := tooltipSlotRE.FindStringSubmatch(match)
		value, ok := vars[parts[1]]
		if !ok {
			return ""
		}
		return formatTooltipValue(value, parts[2])
	})
}

func formatTooltipValue(value any, format string) string {
	decimals := -1
	percent, sign, colourBySign, inverted := false, false, false, false
	colour := ""
	for _, ch := range format {
		switch {
		case ch >= '0' && ch <= '9':
			decimals = int(ch - '0')
		case ch == '%':
			percent = true
		case ch == '=':
			sign = true
		case ch == '+':
			colourBySign = true
		case ch == '-':
			colourBySign, inverted = true, true
		case (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z'):
			colour = string(ch)
		}
	}
	text := ""
	switch v := value.(type) {
	case string:
		text = v
	case float64, int:
		n, _ := toFloat(v)
		if percent {
			n *= 100
		}
		switch {
		case decimals >= 0:
			text = fmt.Sprintf("%.*f", decimals, n)
		case n == math.Trunc(n):
			text = fmt.Sprintf("%.0f", n)
		default:
			text = strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", n), "0"), ".")
		}
		if percent {
			text += "%"
		}
		if sign && n > 0 {
			text = "+" + text
		}
		if colour == "" && colourBySign && n != 0 {
			colour = "G"
			if (n < 0) != inverted {
				colour = "R"
			}
		}
	default:
		text = fmt.Sprint(v)
	}
	if colour != "" {
		return "§" + colour + text + "§!"
	}
	return text
}
