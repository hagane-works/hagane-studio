package ideas

import (
	"errors"
	"fmt"
	"sort"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/workbench"
)

var errAmbiguousBlockText = errors.New("block text contains an ambiguous literal newline")

type blockTextIssue struct {
	Line    int
	Column  int
	Message string
	Repair  string
}

type blockTextPolicyResult struct {
	Value      string
	Normalized int
	Rejected   bool
	Issues     []blockTextIssue
}

func applyBlockTextPolicy(src string) blockTextPolicyResult {
	out := make([]byte, 0, len(src))
	result := blockTextPolicyResult{Issues: []blockTextIssue{}}
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
			result.Rejected = true
			result.Issues = append(result.Issues, issue)
			out = append(out, src[i:i+run+1]...)
			i += run + 1
			column += run + 1
			quoteEscape = false
			continue
		}

		out = append(out, '\n')
		result.Normalized++
		result.Issues = append(result.Issues, issue)
		i += 2
		line, column = line+1, 1
		quoteEscape = false
	}
	result.Value = string(out)
	return result
}

func canonicalBlockText(src string) (string, []blockTextIssue, error) {
	result := applyBlockTextPolicy(src)
	if result.Rejected {
		return src, result.Issues, errAmbiguousBlockText
	}
	return result.Value, result.Issues, nil
}

func canonicalBlockTextFields(fields map[string]string) (map[string]string, []blockTextIssue, error) {
	out := make(map[string]string, len(fields))
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var issues []blockTextIssue
	for _, key := range keys {
		value, found, err := canonicalBlockText(fields[key])
		for i := range found {
			found[i].Repair = key + ": " + found[i].Repair
		}
		issues = append(issues, found...)
		if err != nil {
			return nil, issues, fmt.Errorf("%s: %w", key, err)
		}
		out[key] = value
	}
	return out, issues, nil
}

func blockTextError(issues []blockTextIssue, err error) error {
	if len(issues) > 0 {
		return apiutil.BadRequestError(issues[0].Message + " " + issues[0].Repair)
	}
	return apiutil.BadRequestError(fmt.Sprintf("脚本段落无法保存：%v", err))
}

func canonicalTypedBlockText(fields map[string]string, assign func(map[string]string)) error {
	normalized, issues, err := canonicalBlockTextFields(fields)
	if err != nil {
		return blockTextError(issues, err)
	}
	assign(normalized)
	return nil
}

func canonicalIdeaCreate(in *createInput) error {
	return canonicalTypedBlockText(map[string]string{
		"ai_will_do": in.AIWillDo, "allowed": in.Allowed, "allowed_civil_war": in.AllowedCivilWar,
		"available": in.Available, "visible": in.Visible, "allowed_to_remove": in.AllowedToRemove,
		"cancel": in.Cancel, "do_effect": in.DoEffect, "on_add": in.OnAdd, "on_remove": in.OnRemove,
	}, func(v map[string]string) {
		in.AIWillDo, in.Allowed, in.AllowedCivilWar = v["ai_will_do"], v["allowed"], v["allowed_civil_war"]
		in.Available, in.Visible, in.AllowedToRemove = v["available"], v["visible"], v["allowed_to_remove"]
		in.Cancel, in.DoEffect, in.OnAdd, in.OnRemove = v["cancel"], v["do_effect"], v["on_add"], v["on_remove"]
	})
}

func canonicalIdeaPatch(p *patchInput) error {
	blocks := []struct {
		name  string
		value *workbench.Optional[string]
	}{
		{"ai_will_do", &p.AIWillDo}, {"allowed", &p.Allowed}, {"allowed_civil_war", &p.AllowedCivilWar},
		{"available", &p.Available}, {"visible", &p.Visible}, {"allowed_to_remove", &p.AllowedToRemove},
		{"cancel", &p.Cancel}, {"do_effect", &p.DoEffect}, {"on_add", &p.OnAdd}, {"on_remove", &p.OnRemove},
	}
	fields := map[string]string{}
	for _, block := range blocks {
		if block.value.Set && block.value.Value != nil {
			fields[block.name] = *block.value.Value
		}
	}
	return canonicalTypedBlockText(fields, func(v map[string]string) {
		for _, block := range blocks {
			if block.value.Set && block.value.Value != nil {
				value := v[block.name]
				block.value.Value = &value
			}
		}
	})
}
