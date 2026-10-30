package gamesave

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"haganeworks/internal/domains/gameview"
	"haganeworks/internal/project"
)

const rawFilesTable = "project_raw_files"

type store struct {
	tx  *project.Tx
	raw *rawRecorder
}

func newStore(tx *project.Tx) *store {
	return &store{tx: tx, raw: &rawRecorder{tx: tx, listed: map[int64]gameview.RawFile{}}}
}

type rawFileStep struct {
	ID   int64  `json:"id"`
	File string `json:"file"`

	Before     *string `json:"before"`
	BeforeNote string  `json:"before_note,omitempty"`

	AfterSHA256 string `json:"after_sha256"`
}

func textSHA256(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

type rawRecorder struct {
	tx     *project.Tx
	listed map[int64]gameview.RawFile
	steps  []rawFileStep
}

func (r *rawRecorder) files() *gameview.RawFiles { return gameview.NewRawFiles(r.tx) }

func (r *rawRecorder) list() []gameview.RawFile {
	files := r.files().List()
	for _, file := range files {
		if _, seen := r.listed[file.ID]; !seen {
			r.listed[file.ID] = file
		}
	}
	return files
}

func (r *rawRecorder) get(id int64) (gameview.RawFile, error) {
	for _, file := range r.files().List() {
		if file.ID == id {
			return file, nil
		}
	}
	return gameview.RawFile{}, gameview.ErrRawFileMissing
}

func (r *rawRecorder) create(in gameview.RawFileInsert) (gameview.RawFile, error) {
	made, err := r.files().Create(in)
	if err == nil {
		r.steps = append(r.steps, rawFileStep{ID: made.ID, File: made.DestPath, AfterSHA256: textSHA256(in.Content)})
	}
	return made, err
}

func (r *rawRecorder) update(id int64, intent gameview.RawFileIntent, note *string) (gameview.RawFile, error) {
	saved, err := r.files().Update(id, intent, note)
	if err != nil {
		return saved, err
	}
	for i := range r.steps {
		if r.steps[i].ID == id {
			r.steps[i].AfterSHA256 = textSHA256(intent.Content)
			return saved, nil
		}
	}

	if before, ok := r.listed[id]; ok {
		text := before.Content
		r.steps = append(r.steps, rawFileStep{
			ID: id, File: before.DestPath, Before: &text, BeforeNote: before.Note, AfterSHA256: textSHA256(intent.Content),
		})
	}
	return saved, nil
}

func (r *rawRecorder) delete(id int64) error { return r.files().Delete(id) }

func (r *rawRecorder) put(dest, text, note string) (bool, error) {
	for _, raw := range r.list() {
		if !strings.EqualFold(raw.DestPath, dest) {
			continue
		}
		kept := raw.Note
		if kept == "" {
			kept = note
		}
		_, err := r.update(raw.ID, gameview.RawFileIntent{
			RawKind: raw.RawKind, DestPath: raw.DestPath, Content: text, GenerateReplacePath: raw.GenerateReplacePath,
			TargetLayer: raw.TargetLayer, TargetSHA256: raw.TargetSHA256,
		}, &kept)
		return false, err
	}
	_, err := r.create(gameview.RawFileInsert{DestPath: dest, Content: text, RawKind: "verbatim_override", Note: note})
	return true, err
}

func (r *rawRecorder) withUndo(out map[string]any) map[string]any {
	if out == nil || len(r.steps) == 0 {
		return out
	}
	out["undo"] = map[string]any{"files": r.steps}
	return out
}

func sanitizedPath(file string) (string, error) {
	dest, bad := gameview.SanitizeDestPath(file)
	if bad != "" {
		return "", saveError{422, bad}
	}
	return dest, nil
}

func sanitizedDest(file string) (string, error) {
	dest, err := sanitizedPath(file)
	if err != nil {
		return "", err
	}
	if reserved := gameview.RejectReservedDest(dest); reserved != "" {
		return "", saveError{422, reserved}
	}
	return dest, nil
}

func gateText(dest, text string) error {
	if err := gameview.GateRawContent(dest, text, false); err != nil {
		return saveError{422, err.Error()}
	}
	return nil
}

func text(row project.Row, column string) string {
	s, _ := row[column].(string)
	return s
}

func textOr(row project.Row, column, def string) string {
	if s, ok := row[column].(string); ok {
		return s
	}
	return def
}

func jsonValue(v any) any {
	switch value := v.(type) {
	case nil:
		return nil
	case string, bool, json.Number:
		return value
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out any
	if err := decoder.Decode(&out); err != nil {
		return nil
	}
	return out
}

func decodeColumn(v any, dst any) {
	if v == nil {
		return
	}
	if raw, err := json.Marshal(v); err == nil {
		_ = json.Unmarshal(raw, dst)
	}
}

func (s *store) sortedRows(table string, keep func(project.Row) bool, less func(a, b project.Row) bool) []project.Row {
	out := []project.Row{}
	for _, row := range s.tx.Rows(table) {
		if keep == nil || keep(row) {
			out = append(out, row)
		}
	}
	if less != nil {
		sort.SliceStable(out, func(i, j int) bool { return less(out[i], out[j]) })
	}
	return out
}

func byText(column string) func(a, b project.Row) bool {
	return func(a, b project.Row) bool { return text(a, column) < text(b, column) }
}

func number(v any) float64 {
	switch value := v.(type) {
	case json.Number:
		f, _ := value.Float64()
		return f
	case float64:
		return value
	case int64:
		return float64(value)
	case int:
		return float64(value)
	}
	return 0
}

func byOrdering(a, b project.Row) bool {
	if x, y := number(a["ordering"]), number(b["ordering"]); x != y {
		return x < y
	}
	return project.Int64(a["id"]) < project.Int64(b["id"])
}

func (s *store) update(table string, id int64, changes map[string]any) (project.Row, error) {
	return s.tx.Update(table, id, func(row project.Row) {
		for column, value := range changes {
			row[column] = value
		}
		if textOr(row, "source", "created") == "imported" {
			row["source"] = "created"
		}
	})
}
