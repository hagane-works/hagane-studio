package gamesave

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/domains/gameview"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/project"
)

type saveError struct {
	status int
	detail string
}

func (e saveError) Error() string { return e.detail }

func refusal(err error) error {
	var status *apiutil.StatusError
	if errors.As(err, &status) {
		if body, ok := status.Body.(map[string]any); ok {
			if text, ok := body["detail"].(string); ok {
				return saveError{status.Status, text}
			}
		}
	}
	var missing apiutil.NotFoundError
	if errors.As(err, &missing) {
		return saveError{404, string(missing)}
	}
	return err
}

type gameIndex interface {
	designerSaveIndex
	hqSaveIndex
	decisionSaveIndex
	equipmentSaveIndex
	designSaveIndex
	wargoalSaveIndex
	intelSaveIndex
	autonomySaveIndex
	specialProjectSaveIndex
	intelUpgradeSaveIndex
	oobUnitSaveIndex
	subUnitSaveIndex
	mioTakeIndex
}

var _ gameIndex = (*guirender.Index)(nil)

type handlers struct {
	open apiutil.Opener

	index func(r *http.Request, p *project.Project) gameIndex

	forget func(p *project.Project)
}

func Register(mux *http.ServeMux, deps domains.Deps) {
	RegisterWith(mux, deps, gameview.New(deps))
}

func RegisterWith(mux *http.ServeMux, deps domains.Deps, svc *gameview.Service) {
	register(mux, &handlers{
		open: deps.Open,
		index: func(r *http.Request, p *project.Project) gameIndex {
			index, found, err := svc.Index(r.Context(), p)
			if err != nil || !found || index == nil {
				return nil
			}
			return index
		},
		forget: svc.Forget,
	})
}

func register(mux *http.ServeMux, h *handlers) {
	const prefix = "POST /api/go/projects/{id}/gui-render/"
	for name, fn := range map[string]func(*project.Project, http.ResponseWriter, *http.Request){
		"designer-save":         h.designerSave,
		"decision-save":         h.decisionSave,
		"oob-take":              h.oobTake,
		"equipment-save":        h.equipmentSave,
		"equipment-design-save": h.designSave,
		"undo-files":            h.undoFiles,
		"wargoal-save":          h.wargoalSave,
		"intel-agency-save":     h.intelAgencySave,
		"intel-agency-create":   h.intelAgencyCreate,
		"autonomy-save":         h.autonomySave,
		"special-project-save":  h.specialProjectSave,
		"intel-upgrade-save":    h.intelUpgradeSave,
		"division-save":         h.divisionSave,
		"oob-unit-save":         h.oobUnitSave,
		"sub-unit-save":         h.subUnitSave,
		"mio-take":              h.mioTake,
	} {
		mux.HandleFunc(prefix+name, h.route(fn))
	}
}

func (h *handlers) route(fn func(*project.Project, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := h.open(r)
		if err != nil {
			apiutil.Fail(w, err)
			return
		}
		fn(p, w, r)
	}
}

func detail(w http.ResponseWriter, status int, text string) {
	apiutil.JSON(w, status, map[string]string{"detail": text})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, limit int64, v any) error {
	return json.NewDecoder(http.MaxBytesReader(w, r.Body, limit)).Decode(v)
}

func decodeBody(w http.ResponseWriter, r *http.Request, limit int64, v any, invalid string) bool {
	if err := decodeJSON(w, r, limit, v); err != nil {
		detail(w, http.StatusUnprocessableEntity, invalid)
		return false
	}
	return true
}

func (h *handlers) loadIndex(w http.ResponseWriter, r *http.Request, p *project.Project) (gameIndex, bool) {
	index := h.index(r, p)
	if index == nil {
		detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
		return nil, false
	}
	return index, true
}

func write[T any](p *project.Project, tables []string, save func(*store) (T, error)) (T, error) {
	var out T
	err := p.Write(tables, func(tx *project.Tx) error {
		var err error
		out, err = save(newStore(tx))
		return err
	})
	return out, err
}

func (h *handlers) finish(w http.ResponseWriter, p *project.Project, out any, err error, op string, changed bool) {
	var refused saveError
	switch {
	case errors.As(err, &refused):
		detail(w, refused.status, refused.detail)
	case err != nil:
		log.Printf("[gamesave] %s %d: %v", op, p.ID(), err)
		detail(w, http.StatusServiceUnavailable, op+" failed")
	default:
		if changed {
			h.forget(p)
		}
		apiutil.JSON(w, http.StatusOK, out)
	}
}

func upperTag(tag string) string { return strings.ToUpper(strings.TrimSpace(tag)) }
