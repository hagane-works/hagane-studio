package gameindex

import (
	"encoding/json"
	"errors"
	"net/http"
)

func WriteHTTPError(w http.ResponseWriter, err error) bool {
	code := ""
	switch {
	case errors.Is(err, ErrBuilding):
		code = "game_index_building"
	case errors.Is(err, ErrFailed):
		code = "game_index_failed"
	default:
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "2")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(map[string]string{"detail": err.Error(), "code": code})
	return true
}
