package states

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/project"
)

const stateBulkMaxIDs = 2000

const (
	coreModeAdd        = "add"
	coreModeRemove     = "remove"
	coreModeReplaceOne = "replace_one"
	coreModeReplaceAll = "replace_all"
)

type stateBulkCores struct {
	Mode string   `json:"mode"`
	Tags []string `json:"tags"`
	From string   `json:"from"`
}

type stateBulkInput struct {
	StateDBIDs  []int64         `json:"state_db_ids"`
	Owner       *string         `json:"owner"`
	Controller  *string         `json:"controller"`
	Cores       *stateBulkCores `json:"cores"`
	Claims      *stateBulkCores `json:"claims"`
	RemoveCores *stateBulkCores `json:"remove_cores"`
	DryRun      bool            `json:"dry_run"`
}

type stateBulkDetail struct {
	ID                 int64    `json:"id"`
	StateID            int64    `json:"state_id"`
	Changed            bool     `json:"changed"`
	SkippedReason      string   `json:"skipped_reason,omitempty"`
	BeforeOwner        string   `json:"before_owner"`
	AfterOwner         string   `json:"after_owner"`
	BeforeCores        []string `json:"before_cores"`
	AfterCores         []string `json:"after_cores"`
	BeforeController   string   `json:"before_controller"`
	AfterController    string   `json:"after_controller"`
	BeforeClaims       []string `json:"before_claims"`
	AfterClaims        []string `json:"after_claims"`
	RemovedClaims      []string `json:"removed_claims,omitempty"`
	BeforeRemoveCores  []string `json:"before_remove_cores"`
	AfterRemoveCores   []string `json:"after_remove_cores"`
	RemovedRemoveCores []string `json:"removed_remove_cores,omitempty"`
	RemovedCores       []string `json:"removed_cores,omitempty"`
	BeforeSource       string   `json:"before_source"`
}

type stateBulkResult struct {
	DryRun  bool              `json:"dry_run"`
	Changed int               `json:"changed"`
	Skipped int               `json:"skipped"`
	Details []stateBulkDetail `json:"details"`
}

type stateRestoreEntry struct {
	ID          int64     `json:"id"`
	Owner       *string   `json:"owner"`
	Cores       *[]string `json:"add_core_of"`
	Controller  *string   `json:"controller"`
	Claims      *[]string `json:"add_claim_by"`
	RemoveCores *[]string `json:"remove_core_of"`
	Source      *string   `json:"source"`
}

func normalizeTags(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func sameTags(a, b []string) bool {
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

func applyCoreOp(current []string, op stateBulkCores) ([]string, []string) {
	cur := normalizeTags(current)
	tags := normalizeTags(op.Tags)
	from := strings.TrimSpace(op.From)
	switch op.Mode {
	case coreModeAdd:
		out := append([]string{}, cur...)
		have := map[string]bool{}
		for _, t := range cur {
			have[t] = true
		}
		for _, t := range tags {
			if !have[t] {
				have[t] = true
				out = append(out, t)
			}
		}
		return out, nil
	case coreModeRemove:
		drop := map[string]bool{}
		for _, t := range tags {
			drop[t] = true
		}
		out, removed := []string{}, []string{}
		for _, t := range cur {
			if drop[t] {
				removed = append(removed, t)
				continue
			}
			out = append(out, t)
		}
		return out, removed
	case coreModeReplaceOne:
		idx := -1
		for i, t := range cur {
			if t == from {
				idx = i
				break
			}
		}
		if idx < 0 {
			return cur, nil
		}
		have := map[string]bool{}
		for i, t := range cur {
			if i != idx {
				have[t] = true
			}
		}
		out := append([]string{}, cur[:idx]...)
		for _, t := range tags {
			if !have[t] {
				have[t] = true
				out = append(out, t)
			}
		}
		out = append(out, cur[idx+1:]...)
		return out, []string{from}
	case coreModeReplaceAll:
		keep := map[string]bool{}
		for _, t := range tags {
			keep[t] = true
		}
		removed := []string{}
		for _, t := range cur {
			if !keep[t] {
				removed = append(removed, t)
			}
		}
		return tags, removed
	}
	return cur, nil
}

func validateBulkInput(in stateBulkInput) error {
	if len(in.StateDBIDs) == 0 {
		return detailError(422, "state_db_ids must not be empty")
	}
	if len(in.StateDBIDs) > stateBulkMaxIDs {
		return detailError(422, fmt.Sprintf("state_db_ids exceeds the %d limit", stateBulkMaxIDs))
	}
	if in.Owner == nil && in.Controller == nil && in.Cores == nil && in.Claims == nil && in.RemoveCores == nil {
		return detailError(422, "nothing to change: pass owner / controller / cores / claims / remove_cores")
	}
	for _, f := range []struct {
		name string
		op   *stateBulkCores
	}{{"cores", in.Cores}, {"claims", in.Claims}, {"remove_cores", in.RemoveCores}} {
		if f.op == nil {
			continue
		}
		switch f.op.Mode {
		case coreModeAdd, coreModeRemove, coreModeReplaceAll:
			if len(normalizeTags(f.op.Tags)) == 0 {
				return detailError(422, f.name+".tags must not be empty")
			}
		case coreModeReplaceOne:
			if strings.TrimSpace(f.op.From) == "" {
				return detailError(422, f.name+".from is required for replace_one")
			}
			if len(normalizeTags(f.op.Tags)) == 0 {
				return detailError(422, f.name+".tags must not be empty")
			}
		default:
			return detailError(422, f.name+".mode must be add / remove / replace_one / replace_all")
		}
	}
	return nil
}

func tagsFromRow(row project.Row, column string) []string {
	raw, _ := row[column].([]any)
	out := []string{}
	for _, v := range raw {
		if s := strings.TrimSpace(str(v)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func anyList(tags []string) []any {
	out := make([]any, len(tags))
	for i, t := range tags {
		out[i] = t
	}
	return out
}

type bulkTagField struct {
	column  string
	op      *stateBulkCores
	before  *[]string
	after   *[]string
	removed *[]string
}

func applyTagFields(row project.Row, payload map[string]any, fields []bulkTagField) {
	for _, f := range fields {
		cur := tagsFromRow(row, f.column)
		*f.before, *f.after = cur, cur
		if f.op == nil {
			continue
		}
		next, removed := applyCoreOp(cur, *f.op)
		if sameTags(next, cur) {
			continue
		}
		payload[f.column] = anyList(next)
		*f.after = next
		*f.removed = removed
	}
}

func bulkOwnerCore(p *project.Project, in stateBulkInput) (*stateBulkResult, error) {
	if err := validateBulkInput(in); err != nil {
		return nil, err
	}
	res := &stateBulkResult{DryRun: in.DryRun, Details: []stateBulkDetail{}}
	err := p.Write([]string{statesTable}, func(tx *project.Tx) error {
		for _, id := range in.StateDBIDs {
			row := tx.Find(statesTable, id)
			if row == nil {
				return errStateMissing
			}
			detail := stateBulkDetail{ID: id}
			detail.StateID, _ = whole(row["state_id"])
			curOwner := strings.TrimSpace(str(row["owner"]))
			curController := strings.TrimSpace(str(row["controller"]))
			detail.BeforeOwner, detail.AfterOwner = curOwner, curOwner
			detail.BeforeController, detail.AfterController = curController, curController
			detail.BeforeSource = str(row["source"])
			payload := map[string]any{}
			if in.Owner != nil {
				if want := strings.TrimSpace(*in.Owner); want != curOwner {
					payload["owner"] = want
					detail.AfterOwner = want
				}
			}
			if in.Controller != nil {
				if want := strings.TrimSpace(*in.Controller); want != curController {
					payload["controller"] = want
					detail.AfterController = want
				}
			}
			applyTagFields(row, payload, []bulkTagField{
				{"add_core_of", in.Cores, &detail.BeforeCores, &detail.AfterCores, &detail.RemovedCores},
				{"add_claim_by", in.Claims, &detail.BeforeClaims, &detail.AfterClaims, &detail.RemovedClaims},
				{"remove_core_of", in.RemoveCores, &detail.BeforeRemoveCores, &detail.AfterRemoveCores, &detail.RemovedRemoveCores},
			})
			if len(payload) == 0 {
				detail.SkippedReason = "already_target"
				res.Skipped++
				res.Details = append(res.Details, detail)
				continue
			}
			if err := updateStateTx(tx, id, payload, nil); err != nil {
				return err
			}
			detail.Changed = true
			res.Changed++
			res.Details = append(res.Details, detail)
		}
		if in.DryRun {
			return errDryRun
		}
		return nil
	})
	if err != nil && !errors.Is(err, errDryRun) {
		return nil, err
	}
	return res, nil
}

func bulkRestore(p *project.Project, entries []stateRestoreEntry) (*stateBulkResult, error) {
	if len(entries) == 0 {
		return nil, detailError(422, "entries must not be empty")
	}
	if len(entries) > stateBulkMaxIDs {
		return nil, detailError(422, fmt.Sprintf("entries exceeds the %d limit", stateBulkMaxIDs))
	}
	res := &stateBulkResult{Details: []stateBulkDetail{}}
	err := p.Write([]string{statesTable}, func(tx *project.Tx) error {
		for _, e := range entries {
			row := tx.Find(statesTable, e.ID)
			if row == nil {
				return errStateMissing
			}
			detail := stateBulkDetail{ID: e.ID}
			detail.StateID, _ = whole(row["state_id"])
			curOwner := strings.TrimSpace(str(row["owner"]))
			curController := strings.TrimSpace(str(row["controller"]))
			detail.BeforeOwner, detail.AfterOwner = curOwner, curOwner
			detail.BeforeController, detail.AfterController = curController, curController
			payload := map[string]any{}
			if e.Owner != nil {
				if want := strings.TrimSpace(*e.Owner); want != curOwner {
					payload["owner"] = want
					detail.AfterOwner = want
				}
			}
			if e.Controller != nil {
				if want := strings.TrimSpace(*e.Controller); want != curController {
					payload["controller"] = want
					detail.AfterController = want
				}
			}
			for _, f := range []struct {
				column        string
				want          *[]string
				before, after *[]string
			}{
				{"add_core_of", e.Cores, &detail.BeforeCores, &detail.AfterCores},
				{"add_claim_by", e.Claims, &detail.BeforeClaims, &detail.AfterClaims},
				{"remove_core_of", e.RemoveCores, &detail.BeforeRemoveCores, &detail.AfterRemoveCores},
			} {
				cur := tagsFromRow(row, f.column)
				*f.before, *f.after = cur, cur
				if f.want == nil {
					continue
				}
				want := normalizeTags(*f.want)
				if sameTags(want, cur) {
					continue
				}
				payload[f.column] = anyList(want)
				*f.after = want
			}
			wantSource := ""
			if e.Source != nil && (*e.Source == "imported" || *e.Source == "created") {
				wantSource = *e.Source
			}
			if len(payload) == 0 && (wantSource == "" || wantSource == str(row["source"])) {
				detail.SkippedReason = "already_target"
				res.Skipped++
				res.Details = append(res.Details, detail)
				continue
			}
			if len(payload) > 0 {
				if err := updateStateTx(tx, e.ID, payload, nil); err != nil {
					return err
				}
			}
			if wantSource != "" {
				if _, err := tx.Update(statesTable, e.ID, func(r project.Row) { r["source"] = wantSource }); err != nil {
					return err
				}
			}
			detail.Changed = true
			res.Changed++
			res.Details = append(res.Details, detail)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

type stateBulkDeleteInput struct {
	StateIDs []int64 `json:"state_ids"`
}

type stateBulkDeleteResult struct {
	Deleted        int     `json:"deleted"`
	SkippedCreated []int64 `json:"skipped_created"`
}

func deleteState(p *project.Project, id int64) error {
	return p.Write([]string{statesTable}, func(tx *project.Tx) error {
		if tx.Delete(statesTable, func(r project.Row) bool { return project.Int64(r["id"]) == id }) == 0 {
			return errStateMissing
		}
		return nil
	})
}

func bulkDeleteStates(p *project.Project, in stateBulkDeleteInput) (*stateBulkDeleteResult, error) {
	if len(in.StateIDs) == 0 {
		return nil, detailError(422, "state_ids must not be empty")
	}
	if len(in.StateIDs) > stateBulkMaxIDs {
		return nil, detailError(422, fmt.Sprintf("state_ids exceeds the %d limit", stateBulkMaxIDs))
	}
	wanted := map[int64]bool{}
	for _, id := range in.StateIDs {
		wanted[id] = true
	}
	res := &stateBulkDeleteResult{SkippedCreated: []int64{}}
	err := p.Write([]string{statesTable}, func(tx *project.Tx) error {
		for _, row := range tx.Rows(statesTable) {
			if id := project.Int64(row["id"]); wanted[id] && str(row["source"]) != "imported" {
				res.SkippedCreated = append(res.SkippedCreated, id)
			}
		}
		sort.Slice(res.SkippedCreated, func(i, j int) bool { return res.SkippedCreated[i] < res.SkippedCreated[j] })
		res.Deleted = tx.Delete(statesTable, func(r project.Row) bool {
			return wanted[project.Int64(r["id"])] && str(r["source"]) == "imported"
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
