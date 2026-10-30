package states

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

type handlers struct {
	deps domains.Deps
}

const maxStateBody = 8 << 20

func Register(mux *http.ServeMux, deps domains.Deps) {
	h := &handlers{deps: deps}
	states := "/api/go/projects/{id}/states"
	mux.HandleFunc("GET "+states+"/by-owner/{tag}", h.handle(h.byOwner))
	mux.HandleFunc("POST "+states+"/seed-from-vanilla", h.handle(h.seedFromVanilla))
	mux.HandleFunc("POST "+states+"/bulk-owner-core", h.handle(h.bulkOwnerCore))
	mux.HandleFunc("POST "+states+"/bulk-restore", h.handle(h.bulkRestore))
	mux.HandleFunc("POST "+states+"/restore-vanilla-content", h.handle(h.restoreVanillaContent))
	mux.HandleFunc("POST "+states+"/bulk-delete", h.handle(h.bulkDelete))
	mux.HandleFunc("GET "+states+"/{sid}", h.handle(h.getState))
	mux.HandleFunc("PUT "+states+"/{sid}", h.handle(h.updateState))
	mux.HandleFunc("DELETE "+states+"/{sid}", h.handle(h.deleteState))
	mux.HandleFunc("GET /api/go/projects/{id}/states-geometry", h.handle(h.statesGeometry))
	registerHistoryFiles(mux, h)
	registerBookmarks(mux, h)
}

func RegisterList(mux *http.ServeMux, deps domains.Deps) {
	mux.HandleFunc("GET /api/go/projects/{id}/states", ListHandler(deps))
}

func ListHandler(deps domains.Deps) http.HandlerFunc {
	h := &handlers{deps: deps}
	return h.handle(h.listStates)
}

func (h *handlers) handle(fn func(p *project.Project, w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := h.deps.Open(r)
		if err != nil {
			apiutil.Fail(w, err)
			return
		}
		if err := fn(p, w, r); err != nil {
			fail(w, err)
		}
	}
}

func fail(w http.ResponseWriter, err error) {
	var (
		validation apiutil.ValidationError
		conflict   apiutil.ConflictError
		notFound   apiutil.NotFoundError
		badRequest apiutil.BadRequestError
		status     *apiutil.StatusError
	)
	switch {
	case errors.As(err, &status), errors.As(err, &validation), errors.As(err, &conflict),
		errors.As(err, &notFound), errors.As(err, &badRequest), errors.Is(err, project.ErrNotFound):
		apiutil.Fail(w, err)
	default:
		log.Printf("states: %v", err)
		apiutil.JSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "store unavailable"})
	}
}

func (h *handlers) game() fs.FS {
	if h.deps.Game == nil {
		return nil
	}
	game, err := h.deps.Game()
	if err != nil {
		return nil
	}
	return game
}

func (h *handlers) stateNames() map[string]map[string]string {
	return vanilla.stateNames(h.game())
}

func readBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	return io.ReadAll(http.MaxBytesReader(w, r.Body, maxStateBody))
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any, message string) error {
	body, err := readBody(w, r)
	if err != nil || json.Unmarshal(body, v) != nil {
		return detailError(http.StatusUnprocessableEntity, message)
	}
	return nil
}

func (h *handlers) readStates(p *project.Project, keep func(project.Row) bool) ([]map[string]any, error) {
	rows, err := p.Rows(statesTable)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, _ := whole(rows[i]["state_id"])
		b, _ := whole(rows[j]["state_id"])
		return a < b
	})
	names := h.stateNames()
	out := []map[string]any{}
	for _, row := range rows {
		if keep == nil || keep(row) {
			out = append(out, stateRead(row, names))
		}
	}
	if len(out) > 0 {
		locRows, err := p.Rows(locEntriesTable)
		if err != nil {
			return nil, err
		}
		projectStateDisplayNames(locRows, out)
	}
	return out, nil
}

func (h *handlers) readState(p *project.Project, id int64) (map[string]any, error) {
	items, err := h.readStates(p, func(row project.Row) bool { return project.Int64(row["id"]) == id })
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errStateMissing
	}
	return items[0], nil
}

func (h *handlers) listStates(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	items, err := h.readStates(p, nil)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, items)
	return nil
}

func (h *handlers) byOwner(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	tag := strings.ToUpper(strings.TrimSpace(r.PathValue("tag")))
	items := []map[string]any{}
	if tag != "" {
		var err error
		items, err = h.readStates(p, func(row project.Row) bool {
			return strings.ToUpper(strings.TrimSpace(str(row["owner"]))) == tag
		})
		if err != nil {
			return err
		}
	}
	apiutil.JSON(w, http.StatusOK, items)
	return nil
}

func stateID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("sid"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errStateMissing
	}
	return id, nil
}

func (h *handlers) getState(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := stateID(r)
	if err != nil {
		return err
	}
	item, err := h.readState(p, id)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, item)
	return nil
}

func pbKeyOrder(body []byte) []string {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if tok, err := decoder.Token(); err != nil || tok != json.Delim('{') {
		return nil
	}
	for decoder.More() {
		keyTok, err := decoder.Token()
		if err != nil {
			return nil
		}
		if key, _ := keyTok.(string); key == "province_buildings" {
			if inner, err := decoder.Token(); err != nil || inner != json.Delim('{') {
				return nil
			}
			var keys []string
			for decoder.More() {
				kTok, err := decoder.Token()
				if err != nil {
					return keys
				}
				if k, ok := kTok.(string); ok {
					keys = append(keys, k)
					var skip json.RawMessage
					if err := decoder.Decode(&skip); err != nil {
						return keys
					}
				}
			}
			return keys
		}
		var skip json.RawMessage
		if err := decoder.Decode(&skip); err != nil {
			return nil
		}
	}
	return nil
}

func (h *handlers) updateState(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := stateID(r)
	if err != nil {
		return err
	}
	body, err := readBody(w, r)
	if err != nil {
		return detailError(http.StatusUnprocessableEntity, "Invalid state payload")
	}
	var in map[string]any
	if err := json.Unmarshal(body, &in); err != nil {
		return detailError(http.StatusUnprocessableEntity, "Invalid state payload")
	}
	if in == nil {
		in = map[string]any{}
	}
	order := pbKeyOrder(body)
	if err := p.Write([]string{statesTable}, func(tx *project.Tx) error {
		return updateStateTx(tx, id, in, order)
	}); err != nil {
		return err
	}
	item, err := h.readState(p, id)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, item)
	return nil
}

func (h *handlers) deleteState(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := stateID(r)
	if err != nil {
		return err
	}
	if err := deleteState(p, id); err != nil {
		return err
	}
	apiutil.NoContent(w)
	return nil
}

func (h *handlers) bulkDelete(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var in stateBulkDeleteInput
	if err := decodeBody(w, r, &in, "Invalid bulk delete payload"); err != nil {
		return err
	}
	res, err := bulkDeleteStates(p, in)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, res)
	return nil
}

func (h *handlers) bulkOwnerCore(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var in stateBulkInput
	if err := decodeBody(w, r, &in, "Invalid bulk payload"); err != nil {
		return err
	}
	res, err := bulkOwnerCore(p, in)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, res)
	return nil
}

func (h *handlers) bulkRestore(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var payload struct {
		Entries []stateRestoreEntry `json:"entries"`
	}
	if err := decodeBody(w, r, &payload, "Invalid restore payload"); err != nil {
		return err
	}
	res, err := bulkRestore(p, payload.Entries)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, res)
	return nil
}

func (h *handlers) seedFromVanilla(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	body, err := readBody(w, r)
	if err != nil {
		return detailError(http.StatusUnprocessableEntity, "Invalid state seed payload")
	}
	in, ok := decodeStateSeedInput(body)
	if !ok {
		return detailError(http.StatusUnprocessableEntity, "Invalid state seed payload")
	}
	if !in.All && len(in.StateIDs) == 0 {
		return detailError(http.StatusBadRequest, "Provide `all: true` or a non-empty `state_ids` list.")
	}
	idx, err := h.loadVanillaStates()
	if err != nil {
		return err
	}
	res, err := seedStatesFromVanilla(p, idx, in)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, res)
	return nil
}

func (h *handlers) restoreVanillaContent(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var in restoreInput
	if err := decodeBody(w, r, &in, "Invalid restore-vanilla payload"); err != nil {
		return err
	}
	if err := validateRestoreInput(in); err != nil {
		return err
	}
	idx, err := h.loadVanillaStates()
	if err != nil {
		return err
	}
	res, err := restoreVanillaContent(p, idx, in)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, res)
	return nil
}
