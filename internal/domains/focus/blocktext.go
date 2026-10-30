package focus

import (
	"errors"
	"fmt"
	"sort"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/workbench"
)

const BlockTextLiteralNewlineCode = "block_text.literal_newline"

var ErrAmbiguousBlockText = errors.New("block text contains an ambiguous literal newline")

type BlockTextIssue struct {
	Code    string
	Line    int
	Column  int
	Message string
	Repair  string
}

type BlockTextPolicyResult struct {
	Value      string
	Normalized int
	Rejected   bool
	Issues     []BlockTextIssue
}

func ApplyBlockTextPolicy(src string) BlockTextPolicyResult {
	out := make([]byte, 0, len(src))
	result := BlockTextPolicyResult{Issues: []BlockTextIssue{}}
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

		issue := BlockTextIssue{
			Code:    BlockTextLiteralNewlineCode,
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

func CanonicalBlockText(src string) (string, []BlockTextIssue, error) {
	result := ApplyBlockTextPolicy(src)
	if result.Rejected {
		return src, result.Issues, ErrAmbiguousBlockText
	}
	return result.Value, result.Issues, nil
}

func CanonicalBlockTextFields(fields map[string]string) (map[string]string, []BlockTextIssue, error) {
	out := make(map[string]string, len(fields))
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var issues []BlockTextIssue
	for _, key := range keys {
		value, found, err := CanonicalBlockText(fields[key])
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

func blockTextValidationError(issues []BlockTextIssue, err error) error {
	if len(issues) > 0 {
		return apiutil.ValidationError(issues[0].Message + " " + issues[0].Repair)
	}
	return apiutil.ValidationError(fmt.Sprintf("脚本段落无法保存：%v", err))
}

func canonicalTypedBlockText(fields map[string]string, assign func(map[string]string)) error {
	normalized, issues, err := CanonicalBlockTextFields(fields)
	if err != nil {
		return blockTextValidationError(issues, err)
	}
	assign(normalized)
	return nil
}

func canonicalBlockTextFocusCreate(in *FocusNodeCreateInput) error {
	fields := map[string]string{
		"completion_reward": in.CompletionReward, "available": in.Available,
		"bypass": in.Bypass, "cancel": in.Cancel, "select_effect": in.SelectEffect,
	}
	if in.AiWillDo != nil {
		fields["ai_will_do"] = *in.AiWillDo
	}
	return canonicalTypedBlockText(fields, func(v map[string]string) {
		in.CompletionReward, in.Available = v["completion_reward"], v["available"]
		in.Bypass, in.Cancel, in.SelectEffect = v["bypass"], v["cancel"], v["select_effect"]
		if in.AiWillDo != nil {
			value := v["ai_will_do"]
			in.AiWillDo = &value
		}
	})
}

func canonicalBlockTextFocusPatch(p *FocusNodePatch) error {
	fields := map[string]string{}
	items := []struct {
		name  string
		value *workbench.Optional[string]
	}{
		{"completion_reward", &p.CompletionReward}, {"available", &p.Available},
		{"bypass", &p.Bypass}, {"cancel", &p.Cancel}, {"select_effect", &p.SelectEffect},
		{"ai_will_do", &p.AiWillDo},
	}
	for _, item := range items {
		if item.value.Set && item.value.Value != nil {
			fields[item.name] = *item.value.Value
		}
	}
	return canonicalTypedBlockText(fields, func(v map[string]string) {
		for _, item := range items {
			if item.value.Set && item.value.Value != nil {
				value := v[item.name]
				item.value.Value = &value
			}
		}
	})
}
