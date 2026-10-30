package apiutil

import (
	"encoding/json"
	"errors"
	"haganeworks/internal/gameindex"
	"io"
	"net/http"
	"strconv"

	"haganeworks/internal/project"
	"haganeworks/internal/server"
)

type Opener func(r *http.Request) (*project.Project, error)

const MaxJSONBody = 4 << 20

type (
	ValidationError string

	ConflictError string

	NotFoundError string

	BadRequestError string
)

func (e ValidationError) Error() string { return string(e) }
func (e ConflictError) Error() string   { return string(e) }
func (e NotFoundError) Error() string   { return string(e) }
func (e BadRequestError) Error() string { return string(e) }

type StatusError struct {
	Status int
	Body   any
}

func (e *StatusError) Error() string { return http.StatusText(e.Status) }

func JSON(w http.ResponseWriter, status int, body any) { server.JSON(w, status, body) }

func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

func Fail(w http.ResponseWriter, err error) {
	var (
		validation ValidationError
		conflict   ConflictError
		notFound   NotFoundError
		badRequest BadRequestError
		status     *StatusError
	)
	if gameindex.WriteHTTPError(w, err) {
		return
	}
	switch {
	case errors.As(err, &status):
		server.JSON(w, status.Status, status.Body)
	case errors.Is(err, project.ErrNotFound):
		server.Fail(w, http.StatusNotFound, errors.New("Not found"))
	case errors.As(err, &notFound):
		server.Fail(w, http.StatusNotFound, err)
	case errors.As(err, &validation):
		server.Fail(w, http.StatusUnprocessableEntity, err)
	case errors.As(err, &conflict):
		server.Fail(w, http.StatusConflict, err)
	case errors.As(err, &badRequest):
		server.Fail(w, http.StatusBadRequest, err)
	default:
		server.Fail(w, http.StatusInternalServerError, err)
	}
}

func ReadJSON(r *http.Request, v any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, MaxJSONBody))
	decoder.UseNumber()
	if err := decoder.Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return ValidationError("invalid JSON body: " + err.Error())
	}
	return nil
}

func ReadBody(r *http.Request) ([]byte, error) {
	return io.ReadAll(http.MaxBytesReader(nil, r.Body, MaxJSONBody))
}

func PathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, NotFoundError("Not found")
	}
	return id, nil
}

func Handler(open Opener, fn func(p *project.Project, w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := open(r)
		if err != nil {
			Fail(w, err)
			return
		}
		if err := fn(p, w, r); err != nil {
			Fail(w, err)
		}
	}
}
