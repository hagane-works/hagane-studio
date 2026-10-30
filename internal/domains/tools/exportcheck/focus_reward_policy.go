package exportcheck

import (
	"fmt"
	"haganeworks/internal/hoi4/clausewitz"
	"sort"
	"strings"
	"unicode"
)

type EffectRule struct {
	ValueKinds      []clausewitz.ValueKind
	SupportedScopes []string
}

func (r EffectRule) acceptsKind(kind clausewitz.ValueKind) bool {
	for _, k := range r.ValueKinds {
		if k == kind {
			return true
		}
	}
	return false
}

func (r EffectRule) supportsScope(scope string) bool {
	for _, s := range r.SupportedScopes {
		if s == scope {
			return true
		}
	}
	return false
}

func (r EffectRule) scalarOnly() bool {
	return len(r.ValueKinds) == 1 && r.ValueKinds[0] == clausewitz.ValueScalar
}

var scalarCountryEffects = []string{
	"set_country_leader_ideology",
	"set_cosmetic_tag",
	"retire_character",
	"add_to_faction",
	"add_political_power",
	"add_stability",
}

var scalarOrBlockCountryEffects = []string{
	"promote_character",
	"load_focus_tree",
	"set_country_flag",
}

var DefaultEffectRules = func() map[string]EffectRule {
	rules := map[string]EffectRule{
		"add_building_construction": {
			ValueKinds:      []clausewitz.ValueKind{clausewitz.ValueBlock},
			SupportedScopes: []string{"state"},
		},
	}
	for _, name := range scalarCountryEffects {
		rules[name] = EffectRule{
			ValueKinds:      []clausewitz.ValueKind{clausewitz.ValueScalar},
			SupportedScopes: []string{"country"},
		}
	}
	for _, name := range scalarOrBlockCountryEffects {
		rules[name] = EffectRule{
			ValueKinds:      []clausewitz.ValueKind{clausewitz.ValueScalar, clausewitz.ValueBlock},
			SupportedScopes: []string{"country"},
		}
	}
	return rules
}()

var rewardModifierKeys = map[string]bool{
	"army_attack": true, "army_defense": true, "army_attack_factor": true,
	"army_defence_factor": true, "equipment_capture_factor": true,
}

var rewardTriggerKeys = map[string]bool{
	"has_country_flag": true, "has_completed_focus": true,
	"country_exists": true, "has_government": true,
}

var rewardEffectContainers = map[string]bool{
	"completion_reward": true, "hidden_effect": true, "if": true,
	"else_if": true, "else": true, "random": true, "random_list": true,
	"every_country": true, "every_state": true, "every_owned_state": true,
	"every_neighbor_country": true,
}

func normaliseRewardIdentifier(identifier string, quoted bool) string {
	if quoted {
		return identifier
	}
	var b strings.Builder
	for _, r := range identifier {
		if !clausewitz.PyIsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func pyIsAlnum(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}

func pyIsDigit(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func looksLikeCountryScope(key string) bool {
	return len([]rune(key)) == 3 && pyIsAlnum(key) && !pyIsDigit(key) &&
		strings.ToUpper(key) == key
}

func looksLikeScopeOrWeight(key string) bool {
	return pyIsDigit(key) || looksLikeCountryScope(key)
}

func isEffectPosition(a clausewitz.AssignmentIR) bool {
	if len(a.ParentPath) == 0 {
		return false
	}

	for _, p := range a.ParentPath {
		if p == "limit" {
			return false
		}
	}
	parent := a.ParentPath[len(a.ParentPath)-1]
	return rewardEffectContainers[parent] || looksLikeScopeOrWeight(parent)
}

func scopeAtAssignment(a clausewitz.AssignmentIR, initialScope string) string {
	scope := initialScope
	parent := ""
	for _, container := range a.ParentPath {
		switch {
		case container == "every_state" || container == "every_owned_state":
			scope = "state"
		case container == "every_country" || container == "every_neighbor_country" ||
			looksLikeCountryScope(container):
			scope = "country"
		case pyIsDigit(container) && parent != "random_list":

			scope = "state"
		}
		parent = container
	}
	return scope
}

func rewardFindingSource(a clausewitz.AssignmentIR, sourceName, focusID *string) map[string]any {
	fieldPath := append(append([]string{}, a.ParentPath...), a.Key)
	var file, id any
	if sourceName != nil {
		file = *sourceName
	}
	if focusID != nil {
		id = *focusID
	}
	return map[string]any{
		"file":       file,
		"entity":     map[string]any{"type": "focus", "id": id},
		"field_path": fieldPath,
		"span":       a.Span,
	}
}

func shapeNames(kinds []clausewitz.ValueKind) []string {
	out := []string{}
	for _, k := range kinds {
		out = append(out, string(k))
	}
	sort.Strings(out)
	return out
}

func quotedScalar(a clausewitz.AssignmentIR) string {
	value := a.Value.PyStr()
	if a.ValueQuoted {
		return `"` + value + `"`
	}
	return value
}

type RewardRepairPlan struct {
	RepairID             string
	FindingCode          string
	RepairRoute          string
	Source               map[string]any
	Operations           []map[string]any
	PreviewBefore        string
	PreviewAfter         *string
	Transactional        bool
	RequiresConfirmation bool
	Applicable           bool
}

type RewardAnalysis struct {
	Assignments  []clausewitz.AssignmentIR
	Findings     []clausewitz.AnalysisFinding
	RepairPlans  []RewardRepairPlan
	SourceSHA256 string
}

func rewardRepairID(code string, span clausewitz.SourceSpan) string {
	return fmt.Sprintf("%s:%d:%d", code, span.StartOffset, span.EndOffset)
}

func scalarBlockRepair(
	finding clausewitz.AnalysisFinding,
	assignment clausewitz.AssignmentIR,
	assignments []clausewitz.AssignmentIR,
	normalisedKey string,
) RewardRepairPlan {
	want := append(append([]string{}, assignment.ParentPath...), assignment.Key)
	children := []clausewitz.AssignmentIR{}
	scalarChildren := []clausewitz.AssignmentIR{}
	for _, child := range assignments {
		if !stringSliceEqual(child.ParentPath, want) {
			continue
		}
		children = append(children, child)
		if child.ValueKind == clausewitz.ValueScalar {
			scalarChildren = append(scalarChildren, child)
		}
	}
	deterministic := len(children) == 1 && len(scalarChildren) == 1

	var replacement *string
	var operations []map[string]any
	if deterministic {
		text := fmt.Sprintf("%s = %s", normalisedKey, quotedScalar(scalarChildren[0]))
		replacement = &text
		operations = []map[string]any{{
			"op":          "replace_source_span",
			"span":        assignment.Span,
			"replacement": text,
		}}
	} else {
		operations = []map[string]any{{
			"op":                  "open_focus_reward_editor",
			"focus_field":         "completion_reward",
			"required_value_kind": "scalar",
		}}
	}
	return RewardRepairPlan{
		RepairID:             rewardRepairID(finding.Code, assignment.Span),
		FindingCode:          finding.Code,
		RepairRoute:          finding.RepairRoute,
		Source:               finding.Source,
		Operations:           operations,
		PreviewBefore:        assignment.Excerpt,
		PreviewAfter:         replacement,
		Transactional:        true,
		RequiresConfirmation: true,
		Applicable:           deterministic,
	}
}

func modifierCarrierRepair(
	finding clausewitz.AnalysisFinding,
	assignment clausewitz.AssignmentIR,
) RewardRepairPlan {
	return RewardRepairPlan{
		RepairID:    rewardRepairID(finding.Code, assignment.Span),
		FindingCode: finding.Code,
		RepairRoute: finding.RepairRoute,
		Source:      finding.Source,
		Operations: []map[string]any{
			{"op": "choose_modifier_carrier", "choices": []string{"idea", "dynamic_modifier"}},
			{"op": "create_or_select_carrier", "requires_author_id": true},
			{"op": "author_carrier_localisation", "requires_content_locales": true},
			{"op": "grant_carrier_with_effect", "effect": "add_ideas_or_add_dynamic_modifier"},
		},
		PreviewBefore:        assignment.Excerpt,
		PreviewAfter:         nil,
		Transactional:        true,
		RequiresConfirmation: true,
		Applicable:           false,
	}
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type CompletionRewardInput struct {
	Text         string
	FocusID      *string
	SourceName   *string
	CurrentScope string
	EffectRules  map[string]EffectRule
}

func AnalyzeCompletionReward(in CompletionRewardInput) RewardAnalysis {
	currentScope := in.CurrentScope
	if currentScope == "" {
		currentScope = "country"
	}
	ir := clausewitz.AnalyzeAssignments(in.Text, []string{"completion_reward"}, in.SourceName)

	rules := map[string]EffectRule{}
	for k, v := range DefaultEffectRules {
		rules[k] = v
	}
	for k, v := range in.EffectRules {
		rules[k] = v
	}

	findings := append([]clausewitz.AnalysisFinding{}, ir.Findings...)
	plans := []RewardRepairPlan{}

	for _, assignment := range ir.Assignments {
		if !isEffectPosition(assignment) {
			continue
		}
		key := normaliseRewardIdentifier(assignment.Key, assignment.KeyQuoted)
		effectiveScope := scopeAtAssignment(assignment, currentScope)
		source := rewardFindingSource(assignment, in.SourceName, in.FocusID)

		if rewardModifierKeys[key] {
			finding := clausewitz.AnalysisFinding{
				Code:     "focus.reward.modifier_in_effect_position",
				Severity: "blocker",
				Source:   source,
				Evidence: map[string]any{
					"identifier":            assignment.Key,
					"normalised_identifier": key,
					"value_kind":            string(assignment.ValueKind),
					"assignment":            assignment.Excerpt,
					"current_scope":         effectiveScope,
				},
				RepairRoute: "focus.reward.create_modifier_carrier",
			}
			findings = append(findings, finding)
			plans = append(plans, modifierCarrierRepair(finding, assignment))
			continue
		}

		if rewardTriggerKeys[key] {
			findings = append(findings, clausewitz.AnalysisFinding{
				Code:     "focus.reward.trigger_in_effect_position",
				Severity: "blocker",
				Source:   source,
				Evidence: map[string]any{
					"identifier":            assignment.Key,
					"normalised_identifier": key,
					"assignment":            assignment.Excerpt,
				},
				RepairRoute: "focus.reward.move_or_replace_trigger",
			})
			continue
		}

		rule, known := rules[key]
		if !known {
			continue
		}

		if !rule.supportsScope(effectiveScope) {
			scopes := append([]string{}, rule.SupportedScopes...)
			sort.Strings(scopes)
			findings = append(findings, clausewitz.AnalysisFinding{
				Code:     "focus.reward.unsupported_scope",
				Severity: "blocker",
				Source:   source,
				Evidence: map[string]any{
					"effect":           key,
					"current_scope":    effectiveScope,
					"supported_scopes": scopes,
					"assignment":       assignment.Excerpt,
				},
				RepairRoute: "focus.reward.wrap_supported_scope",
			})
		}

		if !rule.acceptsKind(assignment.ValueKind) {
			code := "focus.reward.invalid_value_shape"
			if assignment.ValueKind == clausewitz.ValueBlock && rule.scalarOnly() {
				code = "focus.reward.scalar_effect_block_value"
			}
			finding := clausewitz.AnalysisFinding{
				Code:     code,
				Severity: "blocker",
				Source:   source,
				Evidence: map[string]any{
					"effect":               key,
					"actual_value_kind":    string(assignment.ValueKind),
					"expected_value_kinds": shapeNames(rule.ValueKinds),
					"assignment":           assignment.Excerpt,
				},
				RepairRoute: "focus.reward.edit_value_shape",
			}
			findings = append(findings, finding)
			if code == "focus.reward.scalar_effect_block_value" {
				plans = append(plans, scalarBlockRepair(finding, assignment, ir.Assignments, key))
			}
		}
	}

	return RewardAnalysis{
		Assignments:  ir.Assignments,
		Findings:     findings,
		RepairPlans:  plans,
		SourceSHA256: ir.SourceSHA256,
	}
}
