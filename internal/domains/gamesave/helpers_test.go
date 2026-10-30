package gamesave

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"haganeworks/internal/domains/gameview"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/project"
)

var allTables = []string{
	rawFilesTable, divisionTemplateTable, divisionDeploymentTable, historyFilesTable, wargoalsTable,
	agenciesTable, autonomyStatesTable, specialProjectsTable, fleetTable, taskForceTable,
	airWingGroupTable, airWingTable, subUnitTable, mioTable,
}

func newProject(t *testing.T) *project.Project {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func run[T any](p *project.Project, save func(*store) (T, error)) (T, error) {
	return write(p, allTables, save)
}

func seed(t *testing.T, p *project.Project, table string, rows ...project.Row) []int64 {
	t.Helper()
	ids := []int64{}
	if err := p.Write([]string{table}, func(tx *project.Tx) error {
		for _, row := range rows {
			ids = append(ids, project.Int64(tx.Insert(table, row)["id"]))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return ids
}

func rows(t *testing.T, p *project.Project, table string) []project.Row {
	t.Helper()
	out, err := p.Rows(table)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func rawFiles(t *testing.T, p *project.Project) []gameview.RawFile {
	t.Helper()
	var files []gameview.RawFile
	if err := p.Write([]string{rawFilesTable}, func(tx *project.Tx) error {
		files = gameview.NewRawFiles(tx).List()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return files
}

func seedRaw(t *testing.T, p *project.Project, in gameview.RawFileInsert) gameview.RawFile {
	t.Helper()
	var made gameview.RawFile
	if err := p.Write([]string{rawFilesTable}, func(tx *project.Tx) error {
		var err error
		made, err = gameview.NewRawFiles(tx).Create(in)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return made
}

func setRawContent(t *testing.T, p *project.Project, file gameview.RawFile, content string) {
	t.Helper()
	if err := p.Write([]string{rawFilesTable}, func(tx *project.Tx) error {
		_, err := gameview.NewRawFiles(tx).Update(file.ID, gameview.RawFileIntent{
			RawKind: file.RawKind, DestPath: file.DestPath, Content: content,
		}, nil)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func index(t *testing.T, files map[string]string) *guirender.Index {
	t.Helper()
	bytesOf := map[string][]byte{}
	for name, text := range files {
		bytesOf[name] = []byte(text)
	}
	out, err := guirender.NewIndex(bytesOf)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func indexOver(t *testing.T, files map[string]string, below *guirender.Index) *guirender.Index {
	t.Helper()
	bytesOf := map[string][]byte{}
	for name, text := range files {
		bytesOf[name] = []byte(text)
	}
	out, err := guirender.NewIndexWithFallback(bytesOf, below)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func statusOf(err error) int {
	if refused, ok := err.(saveError); ok {
		return refused.status
	}
	if err != nil {
		return -1
	}
	return 0
}

func strPtr(s string) *string     { return &s }
func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool        { return &v }

type server struct {
	t         *testing.T
	mux       *http.ServeMux
	forgotten int
}

func newServer(t *testing.T, p *project.Project, idx gameIndex) *server {
	s := &server{t: t, mux: http.NewServeMux()}
	register(s.mux, &handlers{
		open:   func(*http.Request) (*project.Project, error) { return p, nil },
		index:  func(*http.Request, *project.Project) gameIndex { return idx },
		forget: func(*project.Project) { s.forgotten++ },
	})
	return s
}

func (s *server) post(route string, body any, want int) map[string]any {
	s.t.Helper()
	raw, ok := body.([]byte)
	if !ok {
		raw, _ = json.Marshal(body)
	}
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/go/projects/1/gui-render/"+route, bytes.NewReader(raw)))
	if rec.Code != want {
		s.t.Fatalf("%s: status %d, want %d: %s", route, rec.Code, want, rec.Body.String())
	}
	out := map[string]any{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		s.t.Fatalf("%s: %q: %v", route, rec.Body.String(), err)
	}
	return out
}
