package apiutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"haganeworks/internal/project"
)

func TestFailExplainsFileWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "files", "portrait.png")
	cases := []struct {
		cause  error
		status int
		code   string
	}{
		{&fs.PathError{Op: "open", Path: path, Err: fs.ErrPermission}, http.StatusForbidden, "write_blocked"},
		{errors.New("disk full"), http.StatusInternalServerError, "write_failed"},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		Fail(rec, fmt.Errorf("store picture: %w", &project.WriteError{Path: path, Err: c.cause}))
		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if rec.Code != c.status || body["code"] != c.code || body["folder"] != filepath.Dir(path) || body["detail"] == "" {
			t.Fatalf("%v: status %d body %v", c.cause, rec.Code, body)
		}
	}
}
