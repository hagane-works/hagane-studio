package rawfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/clausewitz"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

var referenceIDRe = regexp.MustCompile(`^[A-Za-z0-9_:.@-]+$`)

func sha256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

var repairEntitySpecs = map[string]map[string]bool{
	"focus": {
		"completion_reward": true, "available": true, "bypass": true,
		"cancel": true, "select_effect": true, "ai_will_do": true,
	},
	"event": {
		"trigger": true, "mean_time_to_happen": true, "immediate": true, "after": true,
	},
	"decision": {
		"allowed": true, "available": true, "visible": true, "complete_effect": true,
		"remove_effect": true, "cancel_trigger": true, "cancel_effect": true,
		"modifier": true, "activation": true, "timeout_effect": true,
		"target_root_trigger": true, "target_trigger": true,
		"custom_cost_trigger": true, "ai_will_do": true,
	},
	"idea": {
		"ai_will_do": true, "allowed": true, "allowed_civil_war": true,
		"available": true, "visible": true, "allowed_to_remove": true,
		"cancel": true, "do_effect": true, "on_add": true, "on_remove": true,
	},
}

var repairTables = map[string]string{
	"focus": "focus_nodes", "event": "events", "decision": "decisions", "idea": "ideas",
}

func AllowedCrossRefRepairFields() map[string][]string {
	out := map[string][]string{}
	for kind, fields := range repairEntitySpecs {
		list := make([]string, 0, len(fields))
		for f := range fields {
			list = append(list, f)
		}
		sort.Strings(list)
		out[kind] = list
	}
	return out
}

type CrossRefRepairError struct {
	Code       string
	Message    string
	StatusCode int
	Context    map[string]any
}

func (e *CrossRefRepairError) Error() string { return e.Message }

func (e *CrossRefRepairError) ToDetail() map[string]any {
	ctx := e.Context
	if ctx == nil {
		ctx = map[string]any{}
	}
	return map[string]any{"error": e.Code, "message": e.Message, "context": ctx}
}

func repairErr(code, msg string, status int, ctx map[string]any) *CrossRefRepairError {
	return &CrossRefRepairError{Code: code, Message: msg, StatusCode: status, Context: ctx}
}

func repairContext(projectID int64, kind string, entityID int64, field string) map[string]any {
	return map[string]any{
		"project_id": projectID, "entity_kind": kind,
		"entity_id": entityID, "field": field,
	}
}

func withContext(base map[string]any, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

type CrossRefRepairInput struct {
	EntityKind        string
	EntityID          int64
	Field             string
	TargetID          string
	TargetStartOffset int
	TargetEndOffset   int
	ReplacementID     *string
	DeleteAssignment  bool
	SourceSHA256      *string
}

var repairSHARe = regexp.MustCompile(`^[0-9a-f]{64}$`)

type crossRefRepairBody struct {
	EntityKind        *string `json:"entity_kind"`
	EntityID          *int64  `json:"entity_id"`
	Field             *string `json:"field"`
	TargetID          *string `json:"target_id"`
	TargetStartOffset *int    `json:"target_start_offset"`
	TargetEndOffset   *int    `json:"target_end_offset"`
	ReplacementID     *string `json:"replacement_id"`
	DeleteAssignment  *bool   `json:"delete_assignment"`
	SourceSHA256      *string `json:"source_sha256"`
}

func (b crossRefRepairBody) validate() (CrossRefRepairInput, string) {
	var in CrossRefRepairInput
	if b.EntityKind == nil || repairTables[*b.EntityKind] == "" {
		return in, "entity_kind must be one of focus/event/decision/idea"
	}
	if b.EntityID == nil || *b.EntityID <= 0 {
		return in, "entity_id must be greater than 0"
	}
	if b.Field == nil || len(*b.Field) < 1 || len(*b.Field) > 64 {
		return in, "field length must be between 1 and 64"
	}
	if b.TargetID == nil || len(*b.TargetID) < 1 || len(*b.TargetID) > 512 {
		return in, "target_id length must be between 1 and 512"
	}
	if b.TargetStartOffset == nil || *b.TargetStartOffset < 0 {
		return in, "target_start_offset must be >= 0"
	}
	if b.TargetEndOffset == nil || *b.TargetEndOffset <= 0 {
		return in, "target_end_offset must be > 0"
	}
	if b.ReplacementID != nil && (len(*b.ReplacementID) < 1 || len(*b.ReplacementID) > 512) {
		return in, "replacement_id length must be between 1 and 512"
	}
	if b.SourceSHA256 != nil && !repairSHARe.MatchString(*b.SourceSHA256) {
		return in, "source_sha256 must be 64 lowercase hex characters"
	}
	del := b.DeleteAssignment != nil && *b.DeleteAssignment
	if *b.TargetEndOffset <= *b.TargetStartOffset {
		return in, "target_end_offset must be greater than start offset"
	}
	hasReplacement := b.ReplacementID != nil && *b.ReplacementID != ""
	if hasReplacement == del {
		return in, "choose exactly one action: replacement_id or delete_assignment"
	}
	return CrossRefRepairInput{
		EntityKind: *b.EntityKind, EntityID: *b.EntityID, Field: *b.Field,
		TargetID: *b.TargetID, TargetStartOffset: *b.TargetStartOffset,
		TargetEndOffset: *b.TargetEndOffset, ReplacementID: b.ReplacementID,
		DeleteAssignment: del, SourceSHA256: b.SourceSHA256,
	}, ""
}

func readRepairBody(r *http.Request) (CrossRefRepairInput, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return CrossRefRepairInput{}, apiutil.ValidationError("invalid request body")
	}
	var body crossRefRepairBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return CrossRefRepairInput{}, apiutil.ValidationError("invalid request body")
	}
	in, msg := body.validate()
	if msg != "" {
		return CrossRefRepairInput{}, apiutil.ValidationError(msg)
	}
	return in, nil
}

func repairFailure(err error) error {
	if re, ok := err.(*CrossRefRepairError); ok {
		return &apiutil.StatusError{Status: re.StatusCode, Body: map[string]any{"detail": re.ToDetail()}}
	}
	return err
}

func loadRepairSource(rows []project.Row, projectID int64, in CrossRefRepairInput) (project.Row, string, error) {
	c := repairContext(projectID, in.EntityKind, in.EntityID, in.Field)
	fields, ok := repairEntitySpecs[in.EntityKind]
	if !ok {
		return nil, "", repairErr("cross_ref_repair_entity_kind_not_allowed",
			"Entity kind is not supported by the cross-reference repair API.", 400, c)
	}
	if !fields[in.Field] {
		allowed := make([]string, 0, len(fields))
		for f := range fields {
			allowed = append(allowed, f)
		}
		sort.Strings(allowed)
		return nil, "", repairErr("cross_ref_repair_field_not_allowed",
			"Field is not an allowlisted persisted raw-script field.", 400,
			withContext(c, map[string]any{"allowed_fields": allowed}))
	}
	for _, row := range rows {
		if project.Int64(row["id"]) != in.EntityID {
			continue
		}
		value, _ := row[in.Field].(string)
		return row, value, nil
	}
	return nil, "", repairErr("cross_ref_repair_entity_not_found",
		"Entity was not found in the authorised project.", 404, c)
}

func assertRepairParseable(text string, c map[string]any) error {
	_, recoveries, err := paradox.Parse(text)
	failed := err != nil
	for _, r := range recoveries {
		if r.Code == paradox.RecoveryUnclosedBlock {
			failed = true
		}
	}
	if failed {
		return repairErr("cross_ref_repair_source_unparseable",
			"The current field cannot be safely parsed for a positioned repair.", 422,
			withContext(c, map[string]any{"parser_error": "ParseError"}))
	}
	return nil
}

func validateRepairAction(targetID string, replacement *string, del bool, c map[string]any) error {
	hasReplacement := replacement != nil && *replacement != ""
	if hasReplacement == del {
		return repairErr("cross_ref_repair_action_invalid",
			"Choose exactly one action: replacement_id or delete_assignment.", 400, c)
	}
	if replacement != nil {
		if !referenceIDRe.MatchString(*replacement) {
			return repairErr("cross_ref_repair_replacement_invalid",
				"Replacement id is not a safe HOI4 reference token.", 422, c)
		}
		if *replacement == targetID {
			return repairErr("cross_ref_repair_noop",
				"Replacement id must differ from the current target id.", 400, c)
		}
	}
	return nil
}

func buildRepairPlan(sourceText string, in CrossRefRepairInput, c map[string]any) (
	clausewitz.ReferenceCandidate, clausewitz.CrossRefRepairPlan, error) {

	var zeroCand clausewitz.ReferenceCandidate
	var zeroPlan clausewitz.CrossRefRepairPlan
	if err := validateRepairAction(in.TargetID, in.ReplacementID, in.DeleteAssignment, c); err != nil {
		return zeroCand, zeroPlan, err
	}
	current := sha256Hex(sourceText)
	if in.SourceSHA256 != nil && *in.SourceSHA256 != "" && *in.SourceSHA256 != current {
		return zeroCand, zeroPlan, repairErr("cross_ref_repair_stale_source",
			"The script field changed after the repair preview was created.", 409,
			withContext(c, map[string]any{
				"expected_source_sha256": *in.SourceSHA256,
				"current_source_sha256":  current,
			}))
	}
	if err := assertRepairParseable(sourceText, c); err != nil {
		return zeroCand, zeroPlan, err
	}
	var matches []clausewitz.ReferenceCandidate
	for _, cand := range CollectPositionedReferenceCandidates(sourceText, nil) {
		if cand.TargetID == in.TargetID &&
			cand.SourceSpan.StartOffset == in.TargetStartOffset &&
			cand.SourceSpan.EndOffset == in.TargetEndOffset {
			matches = append(matches, cand)
		}
	}
	if len(matches) != 1 {
		return zeroCand, zeroPlan, repairErr("cross_ref_repair_candidate_not_found",
			"The exact positioned reference no longer exists in this field.", 409,
			withContext(c, map[string]any{
				"target_id":           in.TargetID,
				"target_start_offset": in.TargetStartOffset,
				"target_end_offset":   in.TargetEndOffset,
				"match_count":         len(matches),
			}))
	}
	candidate := matches[0]

	report, err := clausewitz.AnalyzeCrossReferences(sourceText,
		[]clausewitz.ReferenceCandidate{candidate}, clausewitz.CrossRefCatalog{})
	if err != nil || len(report.Findings) == 0 {
		return zeroCand, zeroPlan, repairErr("cross_ref_repair_plan_invalid",
			"The requested repair could not be planned against the current source.", 409,
			withContext(c, map[string]any{"reason": "candidate analysis failed"}))
	}
	plan, err := clausewitz.PlanCrossRefRepair(sourceText, report.Findings[0],
		in.ReplacementID, in.DeleteAssignment)
	if err != nil {
		return zeroCand, zeroPlan, repairErr("cross_ref_repair_plan_invalid",
			"The requested repair could not be planned against the current source.", 409,
			withContext(c, map[string]any{"reason": err.Error()}))
	}
	return candidate, plan, nil
}

func repairPlanResponse(in CrossRefRepairInput, cand clausewitz.ReferenceCandidate,
	plan clausewitz.CrossRefRepairPlan) map[string]any {

	ba := cand.BooleanAncestry
	if ba == nil {
		ba = []string{}
	}
	var repl any
	if plan.ReplacementID != nil {
		repl = *plan.ReplacementID
	}
	return map[string]any{
		"entity_kind": in.EntityKind, "entity_id": in.EntityID, "field": in.Field,
		"target_id": cand.TargetID, "action": plan.Action, "applicable": plan.Applicable,
		"before": plan.Before, "after": plan.After,
		"source_span": plan.SourceSpan, "source_sha256": plan.SourceSHA256,
		"reason": plan.Reason, "replacement_id": repl,
		"boolean_ancestry": ba, "applied": false,
	}
}

func previewRepair(p *project.Project, in CrossRefRepairInput) (map[string]any, error) {
	c := repairContext(p.ID(), in.EntityKind, in.EntityID, in.Field)
	rows, err := p.Rows(repairTables[in.EntityKind])
	if err != nil {
		return nil, err
	}
	row, sourceText, err := loadRepairSource(rows, p.ID(), in)
	if err != nil {
		return nil, err
	}
	cand, plan, err := buildRepairPlan(sourceText, in, c)
	if err != nil {
		return nil, err
	}
	out := repairPlanResponse(in, cand, plan)
	revisions := map[string]string{}
	if in.EntityKind == "focus" {
		treeID := project.Int64(row["focus_tree_id"])
		trees, err := p.Rows("focus_trees")
		if err != nil {
			return nil, err
		}
		for _, tree := range trees {
			if project.Int64(tree["id"]) == treeID {
				revisions[treeScopeKey(treeID)] = strconv.FormatInt(revisionOf(tree), 10)
			}
		}
	}
	out["edit_revisions"] = revisions
	return out, nil
}

func applyRepair(p *project.Project, rv *revisions, in CrossRefRepairInput) (map[string]any, error) {
	c := repairContext(p.ID(), in.EntityKind, in.EntityID, in.Field)
	if in.SourceSHA256 == nil || *in.SourceSHA256 == "" {
		return nil, repairErr("cross_ref_repair_source_sha_required",
			"Apply requires the source_sha256 returned by preview.", 400, c)
	}
	table := repairTables[in.EntityKind]
	tables := []string{table}
	if in.EntityKind == "focus" {
		tables = append(tables, "focus_trees")
	}
	var resp map[string]any
	err := p.Write(tables, func(tx *project.Tx) error {
		if in.EntityKind == "focus" {

			if node := tx.Find(table, in.EntityID); node != nil {
				if _, err := rv.advance(tx, project.Int64(node["focus_tree_id"])); err != nil {
					return err
				}
			}
		}
		_, sourceText, err := loadRepairSource(tx.Rows(table), p.ID(), in)
		if err != nil {
			return err
		}
		cand, plan, err := buildRepairPlan(sourceText, in, c)
		if err != nil {
			return err
		}
		if !plan.Applicable {
			ba := cand.BooleanAncestry
			if ba == nil {
				ba = []string{}
			}
			return repairErr("cross_ref_repair_not_applicable", plan.Reason, 409,
				withContext(c, map[string]any{"target_id": in.TargetID, "boolean_ancestry": ba}))
		}
		repaired, err := clausewitz.ApplyCrossRefRepair(sourceText, plan)
		if err != nil {
			return repairErr("cross_ref_repair_apply_invalid",
				"The repair plan no longer applies to the current source.", 409,
				withContext(c, map[string]any{"reason": err.Error()}))
		}
		if err := assertRepairParseable(repaired, c); err != nil {
			return err
		}
		repairedCands := CollectPositionedReferenceCandidates(repaired, nil)
		if in.ReplacementID != nil {
			present := false
			for _, item := range repairedCands {
				if item.TargetID == *in.ReplacementID &&
					item.EffectName == cand.EffectName &&
					item.TargetType == cand.TargetType &&
					item.SourceSpan.StartOffset == cand.SourceSpan.StartOffset {
					present = true
					break
				}
			}
			if !present {
				return repairErr("cross_ref_repair_verification_failed",
					"The replacement did not reparse as the expected reference.", 409, c)
			}
		}

		if _, err := tx.Update(table, in.EntityID, func(row project.Row) { row[in.Field] = repaired }); err != nil {
			return repairErr("cross_ref_repair_persist_failed",
				"The repaired field could not be persisted transactionally.", 500, c)
		}
		remaining := 0
		for _, item := range repairedCands {
			if item.TargetID == in.TargetID {
				remaining++
			}
		}
		resp = repairPlanResponse(in, cand, plan)
		resp["applied"] = true
		resp["new_source_sha256"] = sha256Hex(repaired)
		resp["remaining_target_count"] = remaining
		resp["reparse_ok"] = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	rv.commit(p)
	return resp, nil
}

func crossRefPreview(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	in, err := readRepairBody(r)
	if err != nil {
		return err
	}
	out, err := previewRepair(p, in)
	if err != nil {
		return repairFailure(err)
	}
	apiutil.JSON(w, http.StatusOK, out)
	return nil
}

func crossRefApply(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	rv, err := readRevisions(r)
	if err != nil {
		return err
	}
	in, err := readRepairBody(r)
	if err != nil {
		return err
	}
	out, err := applyRepair(p, rv, in)
	if err != nil {
		return repairFailure(err)
	}
	rv.header(w)
	apiutil.JSON(w, http.StatusOK, out)
	return nil
}
