package workbench

import (
	"context"
	"sort"
	"strings"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/emit/worldout"
)

type countryTag struct {
	Tag  string `json:"tag"`
	Name string `json:"name"`
}

var tagCatalogExcluded = map[string]bool{
	"ATJ": true, "BLI": true, "BOU": true, "CPS": true,
	"GAL": true, "GOW": true, "POK": true, "RUT": true,
	"SDL": true, "SSI": true, "TNE": true, "VAN": true,
}

var tagCatalogNameOverrides = map[string]string{
	"ALT": "Altai Republic",
	"CIN": "Chechnya-Ingushetia",
	"ENG": "Britain",
	"FER": "Far Eastern Republic",
	"GAM": "The Gambia",
	"KBK": "Kabardino-Balkaria",
	"KOM": "Komi Republic",
	"MEL": "Mari El",
	"NOA": "North Ossetia-Alania",
	"OVO": "Ostyakia Vogulia",
	"SOL": "Solomon Islands",
	"XIC": "Xi'An",
}

type countryTagsKey struct{}

func loadCountryTags(ctx context.Context) ([]countryTag, error) {
	tags, err := gameindex.Load(ctx, countryTagsKey{}, gameindex.WorldTables, func(raw []byte) ([]countryTag, error) {
		tables, err := worldout.DecodeTables(raw)
		if err != nil {
			return nil, err
		}
		countryTags := make([]countryTag, 0, len(tables.CountryFilenames))
		for tag, filename := range tables.CountryFilenames {
			if isDynamicCountryTag(tag) || tagCatalogExcluded[tag] {
				continue
			}
			name := strings.TrimSuffix(filename, ".txt")
			if override := tagCatalogNameOverrides[tag]; override != "" {
				name = override
			}
			countryTags = append(countryTags, countryTag{Tag: tag, Name: name})
		}
		sort.Slice(countryTags, func(i, j int) bool {
			return countryTags[i].Tag < countryTags[j].Tag
		})
		return countryTags, nil
	})
	if err == gameindex.ErrNoGame {
		return []countryTag{}, nil
	}
	return tags, err
}

func isDynamicCountryTag(tag string) bool {
	return len(tag) == 3 && tag[0] == 'D' && tag[1] >= '0' && tag[1] <= '9' && tag[2] >= '0' && tag[2] <= '9'
}

func SearchCountryTags(ctx context.Context, query string) (map[string]any, error) {
	tags, err := loadCountryTags(ctx)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(query)
	if query == "" {
		return map[string]any{"tags": tags, "total": len(tags)}, nil
	}
	filtered := make([]countryTag, 0)
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag.Tag), query) || strings.Contains(strings.ToLower(tag.Name), query) {
			filtered = append(filtered, tag)
		}
	}
	return map[string]any{"tags": filtered, "total": len(filtered)}, nil
}
