package worldout

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"

	"haganeworks/internal/gameindex"
)

type Tables struct {
	VanillaTags    map[string]bool
	StateFilenames map[int64]string

	StateOwners      map[int64]string
	CountryFilenames map[string]string
	CountryCultures  map[string]*OMap
	CosmeticTags     map[string]bool
	IdeologyBaseText string
}

func LoadTables(path string) (*Tables, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeTables(data)
}

type tablesKey struct{}

func LoadVanillaTables(ctx context.Context) (*Tables, error) {
	t, err := gameindex.Load(ctx, tablesKey{}, gameindex.WorldTables, decodeTables)
	if errors.Is(err, gameindex.ErrNoGame) {
		return emptyTables(), nil
	}
	if err != nil {
		return emptyTables(), err
	}
	return t, nil
}

func DecodeTables(data []byte) (*Tables, error) { return decodeTables(data) }

func emptyTables() *Tables {
	return &Tables{
		VanillaTags: map[string]bool{}, StateFilenames: map[int64]string{}, StateOwners: map[int64]string{},
		CountryFilenames: map[string]string{}, CountryCultures: map[string]*OMap{}, CosmeticTags: map[string]bool{},
	}
}

func decodeTables(data []byte) (*Tables, error) {
	var raw struct {
		VanillaTags      []string                   `json:"vanilla_tags"`
		StateFilenames   map[string]string          `json:"state_filenames"`
		StateOwners      map[string]string          `json:"state_owners"`
		CountryFilenames map[string]string          `json:"country_filenames"`
		CountryCultures  map[string]json.RawMessage `json:"country_cultures"`
		CosmeticTags     []string                   `json:"cosmetic_tags"`
		IdeologyBaseText string                     `json:"ideology_base_text"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	if raw.CountryFilenames == nil {
		raw.CountryFilenames = map[string]string{}
	}
	t := &Tables{
		VanillaTags:      map[string]bool{},
		StateFilenames:   map[int64]string{},
		StateOwners:      map[int64]string{},
		CountryFilenames: raw.CountryFilenames,
		CountryCultures:  map[string]*OMap{},
		CosmeticTags:     map[string]bool{},
		IdeologyBaseText: raw.IdeologyBaseText,
	}
	for _, tag := range raw.VanillaTags {
		t.VanillaTags[tag] = true
	}
	for k, v := range raw.StateFilenames {
		id, err := strconv.ParseInt(k, 10, 64)
		if err == nil {
			t.StateFilenames[id] = v
		}
	}
	for k, v := range raw.StateOwners {
		if id, err := strconv.ParseInt(k, 10, 64); err == nil {
			t.StateOwners[id] = v
		}
	}
	for tag, rm := range raw.CountryCultures {
		om := NewOMap()
		if err := om.UnmarshalJSON(rm); err != nil {
			return nil, err
		}
		t.CountryCultures[tag] = om
	}
	for _, id := range raw.CosmeticTags {
		t.CosmeticTags[id] = true
	}
	return t, nil
}
