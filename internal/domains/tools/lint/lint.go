package lint

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

const maxLogUploadBytes = 25 * 1024 * 1024

func Register(mux *http.ServeMux, deps domains.Deps) {
	h := &handlers{open: deps.Open}
	mux.HandleFunc("GET /api/go/lint/panel-contract-core/{id}", h.report(h.panelContractCore))
	mux.HandleFunc("GET /api/go/lint/start-dates/{id}", h.report(h.startDates))
	mux.HandleFunc("GET /api/go/lint/perf/{id}", h.report(h.perf))
	mux.HandleFunc("POST /api/go/lint/error-log/{id}", h.errorLog)
	mux.HandleFunc("POST /api/go/lint/playtest-bundle/{id}", h.playtestBundle)
}

type handlers struct {
	open apiutil.Opener
}

func detail(w http.ResponseWriter, status int, text string) {
	apiutil.JSON(w, status, map[string]string{"detail": text})
}

func projectID(w http.ResponseWriter, r *http.Request) bool {
	if _, err := strconv.ParseInt(r.PathValue("id"), 10, 64); err != nil {
		detail(w, http.StatusUnprocessableEntity, "project_id must be an integer")
		return false
	}
	return true
}

func (h *handlers) openProject(w http.ResponseWriter, r *http.Request) (*project.Project, bool) {
	p, err := h.open(r)
	if errors.Is(err, project.ErrNotFound) {
		detail(w, http.StatusNotFound, "Project not found")
		return nil, false
	}
	if err != nil {
		apiutil.Fail(w, err)
		return nil, false
	}
	return p, true
}

func (h *handlers) report(build func(p *project.Project) (map[string]any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !projectID(w, r) {
			return
		}
		p, ok := h.openProject(w, r)
		if !ok {
			return
		}
		out, err := build(p)
		if err != nil {
			apiutil.Fail(w, err)
			return
		}
		apiutil.JSON(w, http.StatusOK, out)
	}
}

func (h *handlers) panelContractCore(p *project.Project) (map[string]any, error) {
	panels, names, err := loadPanels(p)
	if err != nil {
		return nil, err
	}
	return BuildPanelContractCore(panels, names), nil
}

func (h *handlers) startDates(p *project.Project) (map[string]any, error) {
	return buildStartDateReport(p)
}

func (h *handlers) perf(p *project.Project) (map[string]any, error) {
	return profileProject(p)
}

func (h *handlers) errorLog(w http.ResponseWriter, r *http.Request) {
	if !projectID(w, r) {
		return
	}
	text, done := readUploadedLog(w, r, "file", "error.log")
	if done {
		return
	}
	p, ok := h.openProject(w, r)
	if !ok {
		return
	}
	ix, err := loadLogProjectIndex(p)
	if err != nil {
		apiutil.Fail(w, err)
		return
	}
	apiutil.JSON(w, http.StatusOK, BuildLogReportFromIndex(ix, text))
}

func (h *handlers) playtestBundle(w http.ResponseWriter, r *http.Request) {
	if !projectID(w, r) {
		return
	}
	p, ok := h.openProject(w, r)
	if !ok {
		return
	}
	in, done := readPlaytestBundle(w, r)
	if done {
		return
	}
	builtDate, bookmarkID, err := defaultBookmark(p)
	if err != nil {
		apiutil.Fail(w, err)
		return
	}
	apiutil.JSON(w, http.StatusOK, BuildPlaytestReport(in, builtDate, bookmarkID))
}

func readUploadedLog(w http.ResponseWriter, r *http.Request, field, label string) (string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLogUploadBytes+(4<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		detail(w, http.StatusUnprocessableEntity, "invalid multipart form")
		return "", true
	}
	defer r.MultipartForm.RemoveAll()
	f, _, err := r.FormFile(field)
	if err != nil {
		detail(w, http.StatusUnprocessableEntity, field+" is required")
		return "", true
	}
	data, err := io.ReadAll(io.LimitReader(f, maxLogUploadBytes+1))
	_ = f.Close()
	if err != nil {
		detail(w, http.StatusUnprocessableEntity, "could not read "+label)
		return "", true
	}
	if len(data) == 0 {
		detail(w, http.StatusBadRequest, label+" is empty")
		return "", true
	}
	if len(data) > maxLogUploadBytes {
		detail(w, http.StatusRequestEntityTooLarge,
			label+" too large (> "+strconv.Itoa(maxLogUploadBytes/(1024*1024))+" MB)")
		return "", true
	}
	return decodeUTF8Replace(data), false
}

func readPlaytestBundle(w http.ResponseWriter, r *http.Request) (PlaytestBundleInput, bool) {
	var in PlaytestBundleInput
	r.Body = http.MaxBytesReader(w, r.Body, 5*maxLogUploadBytes+(4<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		detail(w, http.StatusUnprocessableEntity, "invalid multipart form")
		return in, true
	}
	defer r.MultipartForm.RemoveAll()
	slots := []struct {
		field, label string
		dst          **string
	}{
		{"game_log", "game.log", &in.GameLog},
		{"exception_txt", "exception.txt", &in.ExceptionTxt},
		{"dlc_load_json", "dlc_load.json", &in.DLCLoadJSON},
		{"system_log", "system.log", &in.SystemLog},
		{"mod_inventory_json", "mod inventory JSON", &in.ModInventoryJSON},
	}
	present := false
	for _, slot := range slots {
		f, _, err := r.FormFile(slot.field)
		if err != nil {
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(f, maxLogUploadBytes+1))
		_ = f.Close()
		if readErr != nil {
			detail(w, http.StatusUnprocessableEntity, "could not read "+slot.label)
			return in, true
		}
		if len(data) > maxLogUploadBytes {
			detail(w, http.StatusRequestEntityTooLarge,
				slot.label+" too large (> "+strconv.Itoa(maxLogUploadBytes/(1024*1024))+" MB)")
			return in, true
		}
		if len(data) == 0 {
			continue
		}
		text := decodeUTF8Replace(data)
		*slot.dst = &text
		present = true
	}
	if !present {
		detail(w, http.StatusBadRequest, "Upload at least one of game.log / exception.txt / dlc_load.json "+
			"/ system.log / mod inventory JSON")
		return in, true
	}
	in.ExpectedName = r.FormValue("expected_mod_name")
	in.ExpectedPath = r.FormValue("expected_mod_path")
	in.ExpectedBuildID = r.FormValue("expected_build_id")
	return in, false
}

func decodeUTF8Replace(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	var sb strings.Builder
	sb.Grow(len(b))
	for i := 0; i < len(b); {
		c := b[i]
		if c < 0x80 {
			sb.WriteByte(c)
			i++
			continue
		}

		var need int
		var lo, hi byte = 0x80, 0xBF
		switch {
		case c >= 0xC2 && c <= 0xDF:
			need = 1
		case c == 0xE0:
			need, lo = 2, 0xA0
		case c >= 0xE1 && c <= 0xEC, c >= 0xEE && c <= 0xEF:
			need = 2
		case c == 0xED:
			need, hi = 2, 0x9F
		case c == 0xF0:
			need, lo = 3, 0x90
		case c >= 0xF1 && c <= 0xF3:
			need = 3
		case c == 0xF4:
			need, hi = 3, 0x8F
		default:
			sb.WriteRune(0xFFFD)
			i++
			continue
		}
		ok, j := true, i+1
		for k := 0; k < need; k++ {
			l, h := byte(0x80), byte(0xBF)
			if k == 0 {
				l, h = lo, hi
			}
			if j >= len(b) || b[j] < l || b[j] > h {
				ok = false
				break
			}
			j++
		}
		if !ok {
			sb.WriteRune(0xFFFD)
			i = j
			continue
		}
		sb.Write(b[i : i+need+1])
		i += need + 1
	}
	return sb.String()
}
