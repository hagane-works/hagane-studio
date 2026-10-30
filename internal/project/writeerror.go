package project

import (
	"errors"
	"io/fs"
	"os"
)

type WriteError struct {
	Path string
	Err  error
}

func (e *WriteError) Error() string {
	cause := e.Err
	var pathErr *fs.PathError
	var linkErr *os.LinkError
	switch {
	case errors.As(cause, &linkErr):
		cause = linkErr.Err
	case errors.As(cause, &pathErr):
		cause = pathErr.Err
	}
	return "could not write " + e.Path + ": " + cause.Error()
}

func (e *WriteError) Unwrap() error { return e.Err }

func (e *WriteError) Blocked() bool { return errors.Is(e.Err, fs.ErrPermission) }

func writeFailure(path string, err error) error {
	if err == nil {
		return nil
	}
	return &WriteError{Path: path, Err: err}
}
