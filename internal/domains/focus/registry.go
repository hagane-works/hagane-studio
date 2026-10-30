package focus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/gameindex"
)

type FocusTreePathRecord struct {
	TreeID        string
	CanonicalPath string
	ContentSHA256 string
}

type FocusTreePathRegistry struct {
	Records  []FocusTreePathRecord
	ByTreeID map[string]FocusTreePathRecord
}

type registryKey struct{}

var errNoGameFolder = &apiutil.StatusError{Status: http.StatusConflict, Body: map[string]any{
	"code":   "game_folder_missing",
	"detail": "Set the Hearts of Iron IV folder in Settings first: focus trees are checked against the base game's.",
}}

func loadRegistry() (*FocusTreePathRegistry, error) {
	registry, err := gameindex.Load(context.Background(), registryKey{}, gameindex.FocusTrees, decodeRegistry)
	if errors.Is(err, gameindex.ErrNoGame) {
		return nil, errNoGameFolder
	}
	return registry, err
}

func decodeRegistry(raw []byte) (*FocusTreePathRegistry, error) {
	var payload struct {
		PrimaryPaths map[string]string `json:"primary_paths"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("focus tree registry: %w", err)
	}
	if len(payload.PrimaryPaths) == 0 {
		return nil, fmt.Errorf("focus tree registry is incomplete")
	}
	ids := make([]string, 0, len(payload.PrimaryPaths))
	for id := range payload.PrimaryPaths {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	registry := &FocusTreePathRegistry{ByTreeID: make(map[string]FocusTreePathRecord, len(ids))}
	for _, id := range ids {
		record := FocusTreePathRecord{TreeID: id, CanonicalPath: payload.PrimaryPaths[id]}
		registry.Records = append(registry.Records, record)
		registry.ByTreeID[id] = record
	}
	return registry, nil
}

func LoadFocusTreePathRegistry() (*FocusTreePathRegistry, error) { return loadRegistry() }

func registryIndex(registry *FocusTreePathRegistry) RegistryIndex {
	return BuildPrimaryRegistry(registry.PolicySnapshot(), "")
}
