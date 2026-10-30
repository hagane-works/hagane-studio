package decisions

import (
	"errors"
	"fmt"
	"sort"

	"haganeworks/internal/workbench"
)

var errAmbiguousBlockText = errors.New("block text contains an ambiguous literal newline")

type blockTextIssue struct {
	Line    int
	Column  int
	Message string
	Repair  string
}

func canonicalBlockText(src string) (string, []blockTextIssue, error) {
	out := make([]byte, 0, len(src))
	var issues []blockTextIssue
	rejected := false
	inComment, inQuote, quoteEscape := false, false, false
	line, column := 1, 1
	for i := 0; i < len(src); {
		c := src[i]
		if c == '\n' {
			out = append(out, c)
			i++
			line, column = line+1, 1
			inComment, quoteEscape = false, false
			continue
		}
		if inComment {
			out = append(out, c)
			i++
			column++
			continue
		}
		if c == '#' && !inQuote {
			inComment = true
			out = append(out, c)
			i++
			column++
			continue
		}
		if c == '"' && !quoteEscape {
			inQuote = !inQuote
			out = append(out, c)
			i++
			column++
			continue
		}
		if c != '\\' {
			quoteEscape = false
			out = append(out, c)
			i++
			column++
			continue
		}
		run := 1
		for i+run < len(src) && src[i+run] == '\\' {
			run++
		}
		if i+run >= len(src) || src[i+run] != 'n' {
			out = append(out, src[i:i+run]...)
			quoteEscape = inQuote && run%2 == 1
			i += run
			column += run
			continue
		}
		issue := blockTextIssue{
			Line:    line,
			Column:  column,
			Message: "该段可能被游戏当成一整行，后续内容不生效。",
			Repair:  fmt.Sprintf("请在第 %d 行第 %d 列附近改成真实换行。", line, column),
		}
		if inQuote || run != 1 {
			rejected = true
			issues = append(issues, issue)
			out = append(out, src[i:i+run+1]...)
			i += run + 1
			column += run + 1
			quoteEscape = false
			continue
		}
		out = append(out, '\n')
		issues = append(issues, issue)
		i += 2
		line, column = line+1, 1
		quoteEscape = false
	}
	if rejected {
		return src, issues, errAmbiguousBlockText
	}
	return string(out), issues, nil
}

func canonicalBlockTextFields(fields map[string]string, assign func(map[string]string)) error {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(fields))
	var issues []blockTextIssue
	for _, key := range keys {
		value, found, err := canonicalBlockText(fields[key])
		for i := range found {
			found[i].Repair = key + ": " + found[i].Repair
		}
		issues = append(issues, found...)
		if err != nil {

			if len(issues) > 0 {
				return inputError(issues[0].Message + " " + issues[0].Repair)
			}
			return inputError(fmt.Sprintf("脚本段落无法保存：%v", fmt.Errorf("%s: %w", key, err)))
		}
		out[key] = value
	}
	assign(out)
	return nil
}

type inputError string

func (e inputError) Error() string { return string(e) }

func collectOptional(fields map[string]string, name string, value workbench.Optional[string]) {
	if value.Set && value.Value != nil {
		fields[name] = *value.Value
	}
}

func assignOptional(values map[string]string, name string, target *workbench.Optional[string]) {
	if target.Set && target.Value != nil {
		value := values[name]
		target.Value = &value
	}
}

func canonicalCategoryCreate(in *categoryCreateInput) error {
	return canonicalBlockTextFields(map[string]string{
		"allowed": in.Allowed, "visible": in.Visible, "on_map_area": in.OnMapArea,
	}, func(v map[string]string) {
		in.Allowed, in.Visible, in.OnMapArea = v["allowed"], v["visible"], v["on_map_area"]
	})
}

func canonicalCategoryPatch(p *categoryPatch) error {
	fields := map[string]string{}
	collectOptional(fields, "allowed", p.Allowed)
	collectOptional(fields, "visible", p.Visible)
	collectOptional(fields, "on_map_area", p.OnMapArea)
	return canonicalBlockTextFields(fields, func(v map[string]string) {
		assignOptional(v, "allowed", &p.Allowed)
		assignOptional(v, "visible", &p.Visible)
		assignOptional(v, "on_map_area", &p.OnMapArea)
	})
}

func canonicalDecisionCreate(in *decisionCreateInput) error {
	fields := map[string]string{
		"allowed": in.Allowed, "available": in.Available, "visible": in.Visible,
		"complete_effect": in.CompleteEffect, "remove_effect": in.RemoveEffect,
		"cancel_trigger": in.CancelTrigger, "cancel_effect": in.CancelEffect,
		"modifier": in.Modifier, "activation": in.Activation, "timeout_effect": in.TimeoutEffect,
		"target_root_trigger": in.TargetRootTrigger, "target_trigger": in.TargetTrigger,
		"custom_cost_trigger": in.CustomCostTrigger, "ai_will_do": in.AIWillDo,
	}
	return canonicalBlockTextFields(fields, func(v map[string]string) {
		in.Allowed, in.Available, in.Visible = v["allowed"], v["available"], v["visible"]
		in.CompleteEffect, in.RemoveEffect = v["complete_effect"], v["remove_effect"]
		in.CancelTrigger, in.CancelEffect = v["cancel_trigger"], v["cancel_effect"]
		in.Modifier, in.Activation, in.TimeoutEffect = v["modifier"], v["activation"], v["timeout_effect"]
		in.TargetRootTrigger, in.TargetTrigger = v["target_root_trigger"], v["target_trigger"]
		in.CustomCostTrigger, in.AIWillDo = v["custom_cost_trigger"], v["ai_will_do"]
	})
}

func (p *decisionPatch) blockFields() []struct {
	name  string
	value *workbench.Optional[string]
} {
	return []struct {
		name  string
		value *workbench.Optional[string]
	}{
		{"allowed", &p.Allowed}, {"available", &p.Available}, {"visible", &p.Visible},
		{"complete_effect", &p.CompleteEffect}, {"remove_effect", &p.RemoveEffect},
		{"cancel_trigger", &p.CancelTrigger}, {"cancel_effect", &p.CancelEffect},
		{"modifier", &p.Modifier}, {"activation", &p.Activation}, {"timeout_effect", &p.TimeoutEffect},
		{"target_root_trigger", &p.TargetRootTrigger}, {"target_trigger", &p.TargetTrigger},
		{"custom_cost_trigger", &p.CustomCostTrigger}, {"ai_will_do", &p.AIWillDo},
	}
}

func canonicalDecisionPatch(p *decisionPatch) error {
	fields := map[string]string{}
	for _, item := range p.blockFields() {
		collectOptional(fields, item.name, *item.value)
	}
	return canonicalBlockTextFields(fields, func(v map[string]string) {
		for _, item := range p.blockFields() {
			assignOptional(v, item.name, item.value)
		}
	})
}
