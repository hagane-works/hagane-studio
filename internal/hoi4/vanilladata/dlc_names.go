package vanilladata

import (
	"context"
	"encoding/json"
	"strings"

	"haganeworks/internal/gameindex"
)

type DLCName struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Prefix string `json:"prefix"`
	Source string `json:"source"`
}

type dlcNamesFile struct {
	GameVersion string    `json:"game_version"`
	Sources     []string  `json:"sources"`
	DLCs        []DLCName `json:"dlcs"`
}

type dlcRoster struct {
	file   dlcNamesFile
	exact  map[string]DLCName
	folded map[string]DLCName
}

type dlcRosterKey struct{}

func dlcNames() *dlcRoster {
	roster, err := gameindex.Load(context.Background(), dlcRosterKey{}, gameindex.DLCNames, func(raw []byte) (*dlcRoster, error) {
		var top dlcNamesFile
		if err := json.Unmarshal(raw, &top); err != nil {
			return nil, err
		}
		r := &dlcRoster{file: top, exact: map[string]DLCName{}, folded: map[string]DLCName{}}
		for _, d := range top.DLCs {
			r.exact[d.Name] = d
			r.folded[foldDLCName(d.Name)] = d
		}
		return r, nil
	})
	if err != nil {
		return &dlcRoster{exact: map[string]DLCName{}, folded: map[string]DLCName{}}
	}
	return roster
}

func dlcNamesOnce() dlcNamesFile { return dlcNames().file }

func dlcNamesExactOnce() map[string]DLCName { return dlcNames().exact }

func dlcNamesFoldedOnce() map[string]DLCName { return dlcNames().folded }

func foldDLCName(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

func DLCNames() []DLCName {
	return append([]DLCName(nil), dlcNamesOnce().DLCs...)
}

func DLCNamesGameVersion() string { return dlcNamesOnce().GameVersion }

func DLCNamesSources() []string {
	return append([]string(nil), dlcNamesOnce().Sources...)
}

func IsKnownDLCName(name string) bool {
	_, ok := dlcNamesExactOnce()[name]
	return ok
}

func LookupDLCName(name string) (DLCName, bool) {
	d, ok := dlcNamesExactOnce()[name]
	return d, ok
}

func SuggestDLCName(name string) (string, bool) {
	d, ok := dlcNamesFoldedOnce()[foldDLCName(name)]
	if !ok || d.Name == name {
		return "", false
	}
	return d.Name, true
}
