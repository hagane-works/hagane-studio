package importjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"haganeworks/internal/gameindex"
)

type gameFacts struct {
	aliases countryAliasData

	focus    *focusRegistry
	focusErr error
}

func gameIndexBuilding() bool {
	for _, name := range []string{gameindex.CountryAliases, gameindex.FocusTrees} {
		if errors.Is(gameindex.CheckReady(name), gameindex.ErrBuilding) {
			return true
		}
	}
	return false
}

type focusRegistry struct {
	pathByTreeID map[string]string
}

type focusRegistryKey struct{}

func loadFocusRegistry(ctx context.Context) (*focusRegistry, error) {
	registry, err := gameindex.Load(ctx, focusRegistryKey{}, gameindex.FocusTrees, func(raw []byte) (*focusRegistry, error) {
		var payload struct {
			PrimaryPaths map[string]string `json:"primary_paths"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		if len(payload.PrimaryPaths) == 0 {
			return nil, errors.New("focus tree registry is incomplete")
		}
		return &focusRegistry{pathByTreeID: payload.PrimaryPaths}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("the base game's focus tree registry is unavailable: %w", err)
	}
	return registry, nil
}

type countryAliasData struct {
	aliases map[string][]string
	tags    map[string]bool
}

type countryAliasesKey struct{}

func loadCountryAliases(ctx context.Context) (countryAliasData, error) {
	data, err := gameindex.Load(ctx, countryAliasesKey{}, gameindex.CountryAliases, func(raw []byte) (countryAliasData, error) {
		var payload struct {
			Aliases map[string][]string `json:"aliases"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return countryAliasData{}, err
		}
		out := countryAliasData{aliases: payload.Aliases, tags: map[string]bool{}}
		for tag := range payload.Aliases {
			out.tags[tag] = true
		}
		return out, nil
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return countryAliasData{}, ctxErr
		}
		return countryAliasData{aliases: map[string][]string{}, tags: map[string]bool{}}, nil
	}
	return data, nil
}
