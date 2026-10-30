package rawfiles

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
)

type revisionScope struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
}

type revisionProblem struct {
	Code             string        `json:"code"`
	Scope            revisionScope `json:"scope"`
	ExpectedRevision string        `json:"expected_revision,omitempty"`
	CurrentRevision  string        `json:"current_revision,omitempty"`
}

func problemError(status int, problem revisionProblem) error {
	return &apiutil.StatusError{Status: status, Body: map[string]any{"detail": problem}}
}

func parseRevision(raw string) (int64, bool) {
	if raw == "" || len(raw) > 19 || raw[0] < '1' || raw[0] > '9' {
		return 0, false
	}
	for i := 1; i < len(raw); i++ {
		if raw[i] < '0' || raw[i] > '9' {
			return 0, false
		}
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	return value, err == nil
}

func revisionOf(row project.Row) int64 {
	if n := project.Int64(row["edit_revision"]); n > 0 {
		return n
	}
	return 1
}

func treeScopeKey(id int64) string { return "focus_tree:" + strconv.FormatInt(id, 10) }

type revisions struct {
	expected  map[string]string
	advanced  map[int64]bool
	committed map[string]string
}

func readRevisions(r *http.Request) (*revisions, error) {
	rv := &revisions{expected: map[string]string{}, advanced: map[int64]bool{}, committed: map[string]string{}}
	raw := r.Header.Get("X-Edit-Revisions")
	invalid := &apiutil.StatusError{Status: http.StatusUnprocessableEntity,
		Body: map[string]any{"detail": map[string]string{"code": "invalid_edit_revisions"}}}
	if len(raw) > 32768 {
		return nil, invalid
	}
	if raw == "" {
		return rv, nil
	}
	var expected map[string]string
	if err := json.Unmarshal([]byte(raw), &expected); err != nil || expected == nil {
		return nil, invalid
	}
	for key, value := range expected {
		kind, rawID, found := strings.Cut(key, ":")
		if _, ok := parseRevision(rawID); !found || (kind != "bookmark" && kind != "focus_tree") || !ok {
			return nil, invalid
		}
		if _, ok := parseRevision(value); !ok {
			return nil, invalid
		}
	}
	rv.expected = expected
	return rv, nil
}

func (rv *revisions) advance(tx *project.Tx, treeID int64) (int64, error) {
	scope := revisionScope{Kind: "focus_tree", ID: treeID}
	row := tx.Find("focus_trees", treeID)
	if row == nil {
		return 0, problemError(http.StatusNotFound, revisionProblem{Code: "save_target_not_found", Scope: scope})
	}
	current := revisionOf(row)
	if rv.advanced[treeID] {
		return current, nil
	}
	if expected, ok := rv.expected[treeScopeKey(treeID)]; ok && expected != strconv.FormatInt(current, 10) {
		return 0, problemError(http.StatusConflict, revisionProblem{
			Code: "save_conflict", Scope: scope, ExpectedRevision: expected,
			CurrentRevision: strconv.FormatInt(current, 10),
		})
	}
	if current == math.MaxInt64 {
		return 0, problemError(http.StatusInternalServerError, revisionProblem{Code: "save_revision_exhausted", Scope: scope})
	}
	next := current + 1
	if _, err := tx.Update("focus_trees", treeID, func(r project.Row) { r["edit_revision"] = next }); err != nil {
		return 0, err
	}
	rv.advanced[treeID] = true
	return next, nil
}

func (rv *revisions) commit(p *project.Project) {
	if len(rv.advanced) == 0 {
		return
	}
	rows, err := p.Rows("focus_trees")
	if err != nil {
		return
	}
	for _, row := range rows {
		id := project.Int64(row["id"])
		if rv.advanced[id] {
			rv.committed[treeScopeKey(id)] = strconv.FormatInt(revisionOf(row), 10)
		}
	}
}

func (rv *revisions) header(w http.ResponseWriter) {
	if rv == nil || len(rv.committed) == 0 {
		return
	}
	payload, _ := json.Marshal(rv.committed)
	w.Header().Set("X-Edit-Revisions", string(payload))
}
