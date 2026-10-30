package exportcheck

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type bookmarkParityCase struct {
	Name   string `json:"name"`
	Inputs []struct {
		BookmarkID     string `json:"bookmark_id"`
		Path           string `json:"path"`
		Source         string `json:"source"`
		Default        bool   `json:"default"`
		DefaultCountry string `json:"default_country"`
		Countries      []struct {
			Tag         string   `json:"tag"`
			Ideology    *string  `json:"ideology"`
			Labels      []string `json:"labels"`
			Interesting bool     `json:"interesting"`
		} `json:"countries"`
		Filters       []string `json:"filters"`
		TargetVersion string   `json:"target_version"`
		Emitted       bool     `json:"emitted"`
	} `json:"inputs"`
	Result struct {
		TargetVersion string   `json:"target_version"`
		Findings      []any    `json:"findings"`
		BlockerCodes  []string `json:"blocker_codes"`
	} `json:"result"`
}

func TestBookmarkQualityParity(t *testing.T) {
	raw, err := os.ReadFile("testdata/bookmark_quality_parity.json")
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var cases []bookmarkParityCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("fixture unreadable: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("empty fixture")
	}
	for _, c := range cases {
		inputs := make([]BookmarkQualityInput, 0, len(c.Inputs))
		for _, i := range c.Inputs {
			countries := make([]BookmarkCountryInput, 0, len(i.Countries))
			for _, country := range i.Countries {
				ideology := ""
				if country.Ideology != nil {
					ideology = *country.Ideology
				}
				countries = append(countries, BookmarkCountryInput{
					Tag: country.Tag, Ideology: ideology,
					Labels: country.Labels, Interesting: country.Interesting,
				})
			}
			inputs = append(inputs, BookmarkQualityInput{
				BookmarkID: i.BookmarkID, Path: i.Path, Source: i.Source,
				Default: i.Default, DefaultCountry: i.DefaultCountry,
				Countries: countries, Filters: i.Filters,
				TargetVersion: i.TargetVersion, Emitted: i.Emitted,
			})
		}
		result := EvaluateBookmarkQuality(inputs)
		if result.TargetVersion != c.Result.TargetVersion {
			t.Errorf("[%s] target_version = %q, python %q",
				c.Name, result.TargetVersion, c.Result.TargetVersion)
		}
		got := normaliseAny(t, result.Findings)
		want := normaliseAny(t, c.Result.Findings)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("[%s] findings drift:\n got: %v\nwant: %v", c.Name, got, want)
		}
		gotBlockers := []string{}
		for _, f := range result.Blockers() {
			gotBlockers = append(gotBlockers, f.Code)
		}
		if !reflect.DeepEqual(gotBlockers, append([]string{}, c.Result.BlockerCodes...)) &&
			!(len(gotBlockers) == 0 && len(c.Result.BlockerCodes) == 0) {
			t.Errorf("[%s] blockers = %v, python %v", c.Name, gotBlockers, c.Result.BlockerCodes)
		}
	}
}
