package countries

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

type revisions struct {
	expected  map[string]string
	advanced  map[string]bool
	committed map[string]string
}

var scopeKinds = map[string]string{"focus_trees": "focus_tree", "bookmarks": "bookmark"}

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

func invalidRevisions() error {
	return &apiutil.StatusError{Status: http.StatusUnprocessableEntity,
		Body: map[string]any{"detail": map[string]string{"code": "invalid_edit_revisions"}}}
}

func readRevisions(r *http.Request) (*revisions, error) {
	rv := &revisions{expected: map[string]string{}, advanced: map[string]bool{}, committed: map[string]string{}}
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

func revisionOf(row project.Row) int64 {
	if n := project.Int64(row["edit_revision"]); n > 0 {
		return n
	}
	return 1
}

func scopeKey(table string, id int64) string {
	return scopeKinds[table] + ":" + strconv.FormatInt(id, 10)
}

func (rv *revisions) advance(tx *project.Tx, table string, id int64) error {
	key := scopeKey(table, id)
	if rv.advanced[key] {
		return nil
	}
	scope := revisionScope{Kind: scopeKinds[table], ID: id}
	row := tx.Find(table, id)
	if row == nil {
		return &apiutil.StatusError{Status: http.StatusNotFound,
			Body: map[string]any{"detail": revisionProblem{Code: "save_target_not_found", Scope: scope}}}
	}
	current := revisionOf(row)
	if expected, ok := rv.expected[key]; ok && expected != strconv.FormatInt(current, 10) {
		return &apiutil.StatusError{Status: http.StatusConflict, Body: map[string]any{"detail": revisionProblem{
			Code: "save_conflict", Scope: scope, ExpectedRevision: expected, CurrentRevision: strconv.FormatInt(current, 10),
		}}}
	}
	if current == math.MaxInt64 {
		return &apiutil.StatusError{Status: http.StatusInternalServerError,
			Body: map[string]any{"detail": revisionProblem{Code: "save_revision_exhausted", Scope: scope}}}
	}
	if _, err := tx.Update(table, id, func(r project.Row) { r["edit_revision"] = current + 1 }); err != nil {
		return err
	}
	rv.advanced[key] = true
	return nil
}

func (rv *revisions) commit(p *project.Project) {
	if len(rv.advanced) == 0 {
		return
	}
	for table := range scopeKinds {
		rows, err := p.Rows(table)
		if err != nil {
			continue
		}
		for _, row := range rows {
			key := scopeKey(table, project.Int64(row["id"]))
			if rv.advanced[key] {
				rv.committed[key] = strconv.FormatInt(revisionOf(row), 10)
			}
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
