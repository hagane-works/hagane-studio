package events

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const blockTextLiteralNewlineCode = "block_text.literal_newline"

var errAmbiguousBlockText = errors.New("block text contains an ambiguous literal newline")

type blockTextIssue struct {
	Code    string
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
			Code:    blockTextLiteralNewlineCode,
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

func canonicalBlockTextPayload(value any) (any, []blockTextIssue, error) {
	return canonicalBlockTextPayloadAt(value, nil)
}

func canonicalBlockTextPayloadAt(value any, path []string) (any, []blockTextIssue, error) {
	switch current := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(current))
		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var issues []blockTextIssue
		for _, key := range keys {
			childPath := appendPath(path, key)
			if text, ok := current[key].(string); ok && isBlockTextField(key) {
				canonical, found, err := canonicalBlockText(text)
				for i := range found {
					found[i].Repair = strings.Join(childPath, ".") + ": " + found[i].Repair
				}
				issues = append(issues, found...)
				if err != nil {
					return nil, issues, err
				}
				out[key] = canonical
				continue
			}
			child, found, err := canonicalBlockTextPayloadAt(current[key], childPath)
			issues = append(issues, found...)
			if err != nil {
				return nil, issues, err
			}
			out[key] = child
		}
		return out, issues, nil
	case []any:
		out := make([]any, len(current))
		var issues []blockTextIssue
		for i := range current {
			child, found, err := canonicalBlockTextPayloadAt(current[i], appendPath(path, fmt.Sprintf("[%d]", i)))
			issues = append(issues, found...)
			if err != nil {
				return nil, issues, err
			}
			out[i] = child
		}
		return out, issues, nil
	default:
		return value, []blockTextIssue{}, nil
	}
}

var blockTextFields = map[string]bool{
	"after": true, "activation": true, "ai_chance": true, "ai_will_do": true,
	"allow": true, "allow_branch": true, "allowed": true,
	"allowed_civil_war": true, "allowed_to_remove": true,
	"available": true, "bypass": true, "cancel": true,
	"cancel_trigger": true, "complete_effect": true,
	"completion_reward": true, "default_trigger": true,
	"do_effect": true, "effects": true, "hidden_effects": true,
	"immediate": true, "mean_time_to_happen": true, "modifier": true,
	"on_add": true, "on_map_area": true, "on_research_complete": true,
	"on_research_complete_limit": true, "on_remove": true,
	"project_output": true, "remove_effect": true, "resource_cost": true,
	"select_effect": true, "trigger": true, "visible": true,
}

func isBlockTextField(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	return blockTextFields[key] || strings.HasSuffix(key, "_trigger") ||
		strings.HasSuffix(key, "_effect") || strings.HasSuffix(key, "_effects")
}

func appendPath(path []string, item string) []string {
	out := make([]string, len(path)+1)
	copy(out, path)
	out[len(path)] = item
	return out
}

func blockTextError(issues []blockTextIssue, err error) string {
	if len(issues) > 0 {
		return issues[0].Message + " " + issues[0].Repair
	}
	return fmt.Sprintf("脚本段落无法保存：%v", err)
}

func canonicalCreateBlocks(in *eventCreateInput) error {
	fields := map[string]string{
		"trigger":             in.Trigger,
		"mean_time_to_happen": in.MeanTimeToHappen,
	}
	if in.Immediate != nil {
		fields["immediate"] = *in.Immediate
	}
	if in.After != nil {
		fields["after"] = *in.After
	}
	normalized, issues, err := canonicalBlockTextFields(fields)
	if err != nil {
		return errors.New(blockTextError(issues, err))
	}
	in.Trigger = normalized["trigger"]
	in.MeanTimeToHappen = normalized["mean_time_to_happen"]
	if in.Immediate != nil {
		value := normalized["immediate"]
		in.Immediate = &value
	}
	if in.After != nil {
		value := normalized["after"]
		in.After = &value
	}
	payload, issues, err := canonicalBlockTextPayload(in.Options)
	if err != nil {
		return errors.New(blockTextError(issues, err))
	}
	in.Options = payload.([]any)
	return nil
}

func canonicalPatchBlocks(p *eventPatch) error {
	fields := map[string]string{}
	set := func(name string, value optional[string]) {
		if value.Set && value.Value != nil {
			fields[name] = *value.Value
		}
	}
	set("trigger", p.Trigger)
	set("mean_time_to_happen", p.MeanTimeToHappen)
	set("immediate", p.Immediate)
	set("after", p.After)
	normalized, issues, err := canonicalBlockTextFields(fields)
	if err != nil {
		return errors.New(blockTextError(issues, err))
	}
	assign := func(name string, value *optional[string]) {
		if value.Set && value.Value != nil {
			canonical := normalized[name]
			value.Value = &canonical
		}
	}
	assign("trigger", &p.Trigger)
	assign("mean_time_to_happen", &p.MeanTimeToHappen)
	assign("immediate", &p.Immediate)
	assign("after", &p.After)
	if p.Options.Set && p.Options.Value != nil {
		payload, found, payloadErr := canonicalBlockTextPayload(*p.Options.Value)
		if payloadErr != nil {
			return errors.New(blockTextError(found, payloadErr))
		}
		options := payload.([]any)
		p.Options.Value = &options
	}
	return nil
}
