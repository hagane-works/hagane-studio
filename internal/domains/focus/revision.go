package focus

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
)

const codeConflict = "save_conflict"

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

type revisions struct {
	expected  map[string]string
	advanced  map[int64]string
	committed map[string]string
}

func invalidRevisions() error {
	return &apiutil.StatusError{Status: http.StatusUnprocessableEntity,
		Body: map[string]any{"detail": map[string]string{"code": "invalid_edit_revisions"}}}
}

func readRevisions(r *http.Request) (*revisions, error) {
	rv := &revisions{expected: map[string]string{}, advanced: map[int64]string{}, committed: map[string]string{}}
	raw := r.Header.Get("X-Edit-Revisions")
	if len(raw) > 32768 {
		return nil, invalidRevisions()
	}
	if raw == "" {
		return rv, nil
	}
	var expected map[string]string
	if err := json.Unmarshal([]byte(raw), &expected); err != nil || expected == nil {
		return nil, invalidRevisions()
	}
	for key, value := range expected {
		kind, rawID, found := strings.Cut(key, ":")
		if _, ok := parseRevision(rawID); !found || (kind != "bookmark" && kind != "focus_tree") || !ok {
			return nil, invalidRevisions()
		}
		if _, ok := parseRevision(value); !ok {
			return nil, invalidRevisions()
		}
	}
	rv.expected = expected
	return rv, nil
}

func treeScopeKey(id int64) string { return "focus_tree:" + strconv.FormatInt(id, 10) }

func (rv *revisions) advance(tx *project.Tx, treeID int64) (string, error) {
	if revision, done := rv.advanced[treeID]; done {
		return revision, nil
	}
	scope := revisionScope{Kind: "focus_tree", ID: treeID}
	row := tx.Find("focus_trees", treeID)
	if row == nil {
		return "", problemError(http.StatusNotFound, revisionProblem{Code: "save_target_not_found", Scope: scope})
	}
	current := revisionOf(row)
	if expected, ok := rv.expected[treeScopeKey(treeID)]; ok && expected != formatRevision(current) {
		return "", problemError(http.StatusConflict, revisionProblem{
			Code: codeConflict, Scope: scope, ExpectedRevision: expected, CurrentRevision: formatRevision(current),
		})
	}
	if current == math.MaxInt64 {
		return "", problemError(http.StatusInternalServerError, revisionProblem{Code: "save_revision_exhausted", Scope: scope})
	}
	next := current + 1
	if _, err := tx.Update("focus_trees", treeID, func(r project.Row) { r["edit_revision"] = next }); err != nil {
		return "", err
	}
	rv.advanced[treeID] = formatRevision(next)
	return rv.advanced[treeID], nil
}

func (rv *revisions) created(treeID int64) {
	rv.advanced[treeID] = formatRevision(1)
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
		if _, touched := rv.advanced[id]; touched {
			rv.committed[treeScopeKey(id)] = formatRevision(revisionOf(row))
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
