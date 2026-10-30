package exportcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"haganeworks/internal/gameindex"
)

type FocusTreeRegistryError struct{ msg string }

func (e *FocusTreeRegistryError) Error() string { return e.msg }

type FocusTreePathRecord struct {
	TreeID        string
	CanonicalPath string
	SourceFile    string
	ContentSHA256 string
}

type FocusTreePathRegistry struct {
	Records  []FocusTreePathRecord
	ByTreeID map[string]FocusTreePathRecord
}

type focusRegistryKey struct{}

func loadFocusRegistry() (*FocusTreePathRegistry, error) {
	registry, err := gameindex.Load(context.Background(), focusRegistryKey{}, gameindex.FocusTrees, decodeFocusRegistry)
	if err != nil {
		var registryErr *FocusTreeRegistryError
		if errors.As(err, &registryErr) {
			return nil, err
		}
		return nil, &FocusTreeRegistryError{msg: fmt.Sprintf("focus tree registry: %v", err)}
	}
	return registry, nil
}

func decodeFocusRegistry(raw []byte) (*FocusTreePathRegistry, error) {
	var payload struct {
		PrimaryPaths map[string]string `json:"primary_paths"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &FocusTreeRegistryError{msg: fmt.Sprintf("focus tree registry: %v", err)}
	}
	if len(payload.PrimaryPaths) == 0 {
		return nil, &FocusTreeRegistryError{msg: "focus tree registry is incomplete"}
	}
	ids := make([]string, 0, len(payload.PrimaryPaths))
	for id := range payload.PrimaryPaths {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	registry := &FocusTreePathRegistry{ByTreeID: make(map[string]FocusTreePathRecord, len(ids))}
	for _, id := range ids {
		path := payload.PrimaryPaths[id]
		record := FocusTreePathRecord{TreeID: id, CanonicalPath: path, SourceFile: path}
		registry.Records = append(registry.Records, record)
		registry.ByTreeID[id] = record
	}
	return registry, nil
}

func LoadFocusTreePathRegistry() (*FocusTreePathRegistry, error) { return loadFocusRegistry() }
