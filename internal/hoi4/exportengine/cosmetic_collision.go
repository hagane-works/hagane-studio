package exportengine

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/cosmetic"
	"haganeworks/internal/hoi4/emit/worldout"
)

type CosmeticNameCollisionError struct {
	Path, Locale, Key string
	Tags              []string
	Lines             []int
	Values            []string
}

func (e *CosmeticNameCollisionError) Error() string {
	return fmt.Sprintf("cosmetic name collision: %s locale=%s key=%s tags=%s lines=%v values=%q; edit the affected name overrides to agree, then export again",
		e.Path, e.Locale, e.Key, strings.Join(e.Tags, ","), e.Lines, e.Values)
}

var cosmeticGeneratedNameLine = regexp.MustCompile(`^ ([A-Za-z0-9_.-]+):0 "(.*)"$`)

func validateCosmeticGeneratedCollisions(snapshot Snapshot, includeImported bool, entries []Entry) error {
	definitions, err := nativeCosmeticDefinitions(snapshot, includeImported)
	if err != nil {
		return err
	}
	if len(definitions) == 0 {
		return nil
	}
	tags := map[string]bool{}
	for tag := range definitions {
		tags[tag] = true
	}
	if len(snapshot.Tables["history_files"]) > 0 {
		tables, err := worldout.LoadVanillaTables(context.Background())
		if err != nil {
			return err
		}
		for _, raw := range snapshot.Tables["history_files"] {
			var row struct {
				Tag    string `json:"set_cosmetic_tag"`
				Source string `json:"source"`
			}
			if err := json.Unmarshal(raw, &row); err != nil {
				return err
			}

			tag := strings.TrimSpace(row.Tag)
			if tag != "" && (includeImported || row.Source != "imported") && !tables.CosmeticTags[tag] {
				tags[tag] = true
			}
		}
	}
	owners := map[string][]string{}
	registryKeys := map[string]bool{}
	for tag := range tags {
		_, registry := definitions[tag]
		for _, suffix := range cosmetic.Suffixes() {
			key := tag
			if suffix != "" {
				key += "_" + suffix
			}
			owners[key] = append(owners[key], tag)
			registryKeys[key] = registryKeys[key] || registry
		}
	}
	ordered := append([]Entry(nil), entries...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	for _, entry := range ordered {
		if entry.Origin != "generated" || entry.SourceEntity != "world" {
			continue
		}
		parts := strings.Split(entry.Path, "/")
		if len(parts) != 5 || parts[1] != "localisation" || parts[3] != "replace" {
			continue
		}
		locale := "l_" + parts[2]
		if cosmetic.ValidateLocale(locale) != nil || parts[4] != "zz_"+parts[0]+"_cosmetic_"+locale+".yml" {
			continue
		}
		lines := strings.Split(strings.TrimPrefix(string(entry.Bytes), "\ufeff"), "\n")
		if strings.TrimSuffix(lines[0], "\r") != locale+":" {
			continue
		}
		first := map[string]struct {
			line  int
			value string
		}{}
		for index, line := range lines {
			match := cosmeticGeneratedNameLine.FindStringSubmatch(strings.TrimSuffix(line, "\r"))
			if match == nil || len(owners[match[1]]) < 2 || !registryKeys[match[1]] {
				continue
			}
			key, value := match[1], match[2]
			if prior, ok := first[key]; ok && prior.value != value {
				sort.Strings(owners[key])
				return &CosmeticNameCollisionError{Path: entry.Path, Locale: locale, Key: key,
					Tags: owners[key], Lines: []int{prior.line, index + 1}, Values: []string{prior.value, value}}
			} else if !ok {
				first[key] = struct {
					line  int
					value string
				}{index + 1, value}
			}
		}
	}
	return nil
}
