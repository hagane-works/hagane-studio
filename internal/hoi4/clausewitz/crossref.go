package clausewitz

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
)

const (
	StatusResolved          = "resolved"
	StatusTypo              = "typo"
	StatusUnknown           = "unknown"
	StatusBaseModNotIndexed = "base_mod_not_indexed"
	StatusMissing           = "missing"
)

const (
	ImpactReportOnly = "report_only"
	ImpactBlocker    = "blocker"
)

const (
	ActionReplaceTarget    = "replace_target"
	ActionDeleteAssignment = "delete_assignment"
)

type ReferenceCandidate struct {
	EffectName      string
	TargetType      string
	TargetID        string
	SourceSpan      SourceSpan
	ParentPath      []string
	AssignmentSpan  *SourceSpan
	BooleanAncestry []string
}

func (c ReferenceCandidate) Validate() error {
	if c.EffectName == "" {
		return errors.New("effect_name must not be empty")
	}
	if c.TargetType == "" {
		return errors.New("target_type must not be empty")
	}
	if c.TargetID == "" {
		return errors.New("target_id must not be empty")
	}
	return nil
}

func (c ReferenceCandidate) ToMap() map[string]any {
	pp := c.ParentPath
	if pp == nil {
		pp = []string{}
	}
	ba := c.BooleanAncestry
	if ba == nil {
		ba = []string{}
	}
	var as any
	if c.AssignmentSpan != nil {
		as = *c.AssignmentSpan
	}
	return map[string]any{
		"effect_name": c.EffectName, "target_type": c.TargetType,
		"target_id": c.TargetID, "source_span": c.SourceSpan,
		"parent_path": pp, "assignment_span": as, "boolean_ancestry": ba,
	}
}

type CrossRefCatalog struct {
	ProjectIDs         map[string]map[string]bool
	VanillaIDs         map[string]map[string]bool
	BaseModIDs         map[string]map[string]bool
	CompleteTypes      map[string]bool
	ExpectsBaseMod     bool
	BaseIndexedTypes   map[string]bool
	TypoSuggestions    map[[2]string]string
	FatalReferenceKind map[[2]string]bool
}

func (c CrossRefCatalog) KnownIDs(targetType string) map[string]bool {
	out := map[string]bool{}
	for _, m := range []map[string]map[string]bool{c.ProjectIDs, c.VanillaIDs, c.BaseModIDs} {
		for id := range m[targetType] {
			out[id] = true
		}
	}
	return out
}

type CrossRefFinding struct {
	Candidate   ReferenceCandidate
	Status      string
	Impact      string
	Suggestion  *string
	Detail      string
	RepairRoute string
}

func (f CrossRefFinding) ToMap() map[string]any {
	var sug any
	if f.Suggestion != nil {
		sug = *f.Suggestion
	}
	return map[string]any{
		"code": "cross_ref." + f.Status, "status": f.Status, "impact": f.Impact,
		"suggestion": sug, "detail": f.Detail, "repair_route": f.RepairRoute,
		"candidate": f.Candidate.ToMap(),
	}
}

type CrossRefPolicyReport struct {
	SourceText   string
	OutputText   string
	Findings     []CrossRefFinding
	SourceSHA256 string
	OutputSHA256 string
}

func (r CrossRefPolicyReport) TextUnchanged() bool {
	return r.SourceText == r.OutputText && r.SourceSHA256 == r.OutputSHA256
}

func (r CrossRefPolicyReport) Blockers() []CrossRefFinding {
	var out []CrossRefFinding
	for _, f := range r.Findings {
		if f.Impact == ImpactBlocker {
			out = append(out, f)
		}
	}
	return out
}

func (r CrossRefPolicyReport) ToMap() map[string]any {
	fs := make([]map[string]any, 0, len(r.Findings))
	for _, f := range r.Findings {
		fs = append(fs, f.ToMap())
	}
	return map[string]any{
		"findings": fs, "source_sha256": r.SourceSHA256,
		"output_sha256": r.OutputSHA256, "text_unchanged": r.TextUnchanged(),
		"blocker_count": len(r.Blockers()),
	}
}

type CrossRefRepairPlan struct {
	Action        string
	Applicable    bool
	SourceSpan    SourceSpan
	Before        string
	After         string
	SourceSHA256  string
	Reason        string
	TargetID      string
	ReplacementID *string
}

func (p CrossRefRepairPlan) ToMap() map[string]any {
	var rid any
	if p.ReplacementID != nil {
		rid = *p.ReplacementID
	}
	return map[string]any{
		"action": p.Action, "applicable": p.Applicable, "source_span": p.SourceSpan,
		"before": p.Before, "after": p.After, "source_sha256": p.SourceSHA256,
		"reason": p.Reason, "target_id": p.TargetID, "replacement_id": rid,
	}
}

func sha256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func spanText(text []rune, span SourceSpan) (string, error) {
	if span.StartOffset < 0 || span.EndOffset < span.StartOffset {
		return "", errors.New("invalid source span")
	}
	if span.EndOffset > len(text) {
		return "", errors.New("source span extends beyond source text")
	}
	return string(text[span.StartOffset:span.EndOffset]), nil
}

func levenshtein(left, right []rune) int {
	if string(left) == string(right) {
		return 0
	}
	if len(left) == 0 {
		return len(right)
	}
	if len(right) == 0 {
		return len(left)
	}
	previous := make([]int, len(right)+1)
	for i := range previous {
		previous[i] = i
	}
	for row := 1; row <= len(left); row++ {
		current := make([]int, 0, len(right)+1)
		current = append(current, row)
		for column := 1; column <= len(right); column++ {
			sub := previous[column-1]
			if left[row-1] != right[column-1] {
				sub++
			}
			m := current[len(current)-1] + 1
			if previous[column]+1 < m {
				m = previous[column] + 1
			}
			if sub < m {
				m = sub
			}
			current = append(current, m)
		}
		previous = current
	}
	return previous[len(previous)-1]
}

func typoSuggestion(c ReferenceCandidate, catalog CrossRefCatalog) *string {
	if explicit, ok := catalog.TypoSuggestions[[2]string{c.TargetType, c.TargetID}]; ok && explicit != "" {
		return &explicit
	}
	target := []rune(c.TargetID)
	if len(target) < 4 {
		return nil
	}
	var matches []string
	for known := range catalog.KnownIDs(c.TargetType) {
		kr := []rune(known)
		d := len(target) - len(kr)
		if d < 0 {
			d = -d
		}
		if d > 1 {
			continue
		}
		if levenshtein(target, kr) == 1 {
			matches = append(matches, known)
		}
	}
	if len(matches) != 1 {
		return nil
	}
	sort.Strings(matches)
	return &matches[0]
}

func classify(c ReferenceCandidate, catalog CrossRefCatalog) (string, *string, string) {
	if catalog.KnownIDs(c.TargetType)[c.TargetID] {
		return StatusResolved, nil,
			"target exists in a project, vanilla, or indexed base-mod catalog"
	}
	if s := typoSuggestion(c, catalog); s != nil {
		return StatusTypo, s,
			fmt.Sprintf("unresolved target is one edit away from known id %s", *s)
	}
	if catalog.ExpectsBaseMod && !catalog.BaseIndexedTypes[c.TargetType] {
		return StatusBaseModNotIndexed, nil,
			"the project declares a base mod, but this target type has no base index"
	}
	if catalog.CompleteTypes[c.TargetType] {
		return StatusMissing, nil,
			"the target is absent from every complete catalog for this type"
	}
	return StatusUnknown, nil,
		"catalog coverage is incomplete, so absence cannot be established"
}

func AnalyzeCrossReferences(text string, candidates []ReferenceCandidate,
	catalog CrossRefCatalog) (CrossRefPolicyReport, error) {

	runes := []rune(text)
	findings := make([]CrossRefFinding, 0, len(candidates))
	for _, c := range candidates {
		if err := c.Validate(); err != nil {
			return CrossRefPolicyReport{}, err
		}
		observed, err := spanText(runes, c.SourceSpan)
		if err != nil {
			return CrossRefPolicyReport{}, err
		}
		if observed != c.TargetID {
			return CrossRefPolicyReport{}, fmt.Errorf(
				"candidate source span does not select target_id: expected %q, observed %q",
				c.TargetID, observed)
		}
		status, suggestion, detail := classify(c, catalog)
		impact := ImpactReportOnly
		if status != StatusResolved && catalog.FatalReferenceKind[[2]string{c.EffectName, c.TargetType}] {
			impact = ImpactBlocker
		}
		findings = append(findings, CrossRefFinding{
			Candidate: c, Status: status, Impact: impact, Suggestion: suggestion,
			Detail: detail, RepairRoute: "cross_ref.explicit_repair",
		})
	}
	h := sha256Hex(text)
	return CrossRefPolicyReport{
		SourceText: text, OutputText: text, Findings: findings,
		SourceSHA256: h, OutputSHA256: h,
	}, nil
}

func PlanCrossRefRepair(sourceText string, finding CrossRefFinding,
	replacementID *string, deleteAssignment bool) (CrossRefRepairPlan, error) {

	hasReplacement := replacementID != nil && *replacementID != ""
	if hasReplacement == deleteAssignment {
		return CrossRefRepairPlan{}, errors.New(
			"choose exactly one repair: replacement_id or delete_assignment")
	}
	runes := []rune(sourceText)
	h := sha256Hex(sourceText)
	c := finding.Candidate

	if replacementID != nil {
		before, err := spanText(runes, c.SourceSpan)
		if err != nil {
			return CrossRefRepairPlan{}, err
		}
		if before != c.TargetID {
			return CrossRefRepairPlan{}, errors.New("repair target span no longer matches the finding")
		}
		return CrossRefRepairPlan{
			Action: ActionReplaceTarget, Applicable: true, SourceSpan: c.SourceSpan,
			Before: before, After: *replacementID, SourceSHA256: h,
			Reason: "explicit target replacement requested", TargetID: c.TargetID,
			ReplacementID: replacementID,
		}, nil
	}

	span := c.SourceSpan
	if c.AssignmentSpan != nil {
		span = *c.AssignmentSpan
	}
	before, err := spanText(runes, span)
	if err != nil {
		return CrossRefRepairPlan{}, err
	}
	if len(c.BooleanAncestry) > 0 {
		ancestry := ""
		for i, a := range c.BooleanAncestry {
			if i > 0 {
				ancestry += " > "
			}
			ancestry += a
		}
		return CrossRefRepairPlan{
			Action: ActionDeleteAssignment, Applicable: false, SourceSpan: span,
			Before: before, After: before, SourceSHA256: h,
			Reason:   "automatic deletion refused because boolean ancestry may change semantics: " + ancestry,
			TargetID: c.TargetID,
		}, nil
	}
	if c.AssignmentSpan == nil {
		return CrossRefRepairPlan{
			Action: ActionDeleteAssignment, Applicable: false, SourceSpan: span,
			Before: before, After: before, SourceSHA256: h,
			Reason: "assignment span is required for deletion", TargetID: c.TargetID,
		}, nil
	}
	return CrossRefRepairPlan{
		Action: ActionDeleteAssignment, Applicable: true, SourceSpan: span,
		Before: before, After: "", SourceSHA256: h,
		Reason: "explicit assignment deletion requested", TargetID: c.TargetID,
	}, nil
}

func ApplyCrossRefRepair(sourceText string, plan CrossRefRepairPlan) (string, error) {
	if !plan.Applicable {
		return "", fmt.Errorf("repair plan is not applicable: %s", plan.Reason)
	}
	if sha256Hex(sourceText) != plan.SourceSHA256 {
		return "", errors.New("repair source changed after the plan was created")
	}
	runes := []rune(sourceText)
	observed, err := spanText(runes, plan.SourceSpan)
	if err != nil {
		return "", err
	}
	if observed != plan.Before {
		return "", errors.New("repair span content changed after planning")
	}
	return string(runes[:plan.SourceSpan.StartOffset]) + plan.After +
		string(runes[plan.SourceSpan.EndOffset:]), nil
}
