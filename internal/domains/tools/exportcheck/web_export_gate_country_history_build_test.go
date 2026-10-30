package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCountryHistoryBuildParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "country_history_build_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name    string `json:"name"`
			Project struct {
				CountryTag  string   `json:"country_tag"`
				CountryTags []string `json:"country_tags"`
			} `json:"project"`
			Countries []string `json:"countries"`
			Bookmarks []struct {
				DefaultCountry      string           `json:"default_country"`
				SelectableCountries []map[string]any `json:"selectable_countries"`
			} `json:"bookmarks"`
			HistoryFiles []struct {
				ID               int64  `json:"id"`
				CountryTag       string `json:"country_tag"`
				Capital          any    `json:"capital"`
				SetResearchSlots any    `json:"set_research_slots"`
				Source           string `json:"source"`
			} `json:"history_files"`
			States []struct {
				StateID int64  `json:"state_id"`
				Owner   string `json:"owner"`
			} `json:"states"`
			RawFiles []struct {
				ID            any    `json:"id"`
				DestPath      string `json:"dest_path"`
				Content       string `json:"content"`
				Source        string `json:"source"`
				IsBinary      bool   `json:"is_binary"`
				ContentSHA256 string `json:"content_sha256"`
			} `json:"raw_files"`
			Findings json.RawMessage `json:"findings"`
			Entries  []struct {
				Tag         string `json:"tag"`
				Requirement struct {
					Authored    bool `json:"authored"`
					Selectable  bool `json:"selectable"`
					Interesting bool `json:"interesting"`
				} `json:"requirement"`
				ProviderIDs         []string `json:"provider_ids"`
				LoadOrders          []int    `json:"load_orders"`
				FinalWinner         *string  `json:"final_winner"`
				Capital             *int     `json:"capital"`
				CapitalIsValidState bool     `json:"capital_is_valid_state"`
				ResearchSlots       *int     `json:"research_slots"`
			} `json:"entries"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	for _, c := range fx.Cases {
		in := CountryHistoryBuildInput{
			ProjectCountryTag:      c.Project.CountryTag,
			ProjectCountryTags:     c.Project.CountryTags,
			CountryTags:            c.Countries,
			StructuredArtifactName: "Mod",
		}
		for _, b := range c.Bookmarks {
			in.Bookmarks = append(in.Bookmarks, CountryHistoryBookmark{
				DefaultCountry: b.DefaultCountry, SelectableCountries: b.SelectableCountries})
		}
		for _, h := range c.HistoryFiles {
			in.HistoryFiles = append(in.HistoryFiles, CountryHistoryFileRow{
				ID: h.ID, CountryTag: h.CountryTag, Capital: h.Capital,
				SetResearchSlots: h.SetResearchSlots, Source: h.Source})
		}
		for _, s := range c.States {
			in.States = append(in.States, ValidateState{StateID: s.StateID, Owner: s.Owner})
		}
		for _, r := range c.RawFiles {
			in.RawFiles = append(in.RawFiles, CountryHistoryRawFile{
				ID: r.ID, DestPath: r.DestPath, Content: r.Content,
				Source: r.Source, IsBinary: r.IsBinary, ContentSHA256: r.ContentSHA256})
		}

		reg := BuildCountryHistoryForExport(in)

		gotFindings := jsonNorm(t, reg.Findings)
		var wantFindings any
		if err := json.Unmarshal(c.Findings, &wantFindings); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotFindings, wantFindings) {
			gb, _ := json.Marshal(gotFindings)
			t.Errorf("case %s findings\n got %s\nwant %s", c.Name, gb, c.Findings)
		}

		if len(reg.Entries) != len(c.Entries) {
			t.Errorf("case %s entries %d want %d", c.Name, len(reg.Entries), len(c.Entries))
			continue
		}
		for i, want := range c.Entries {
			got := reg.Entries[i]
			if got.Tag != want.Tag {
				t.Errorf("case %s entry[%d].tag = %q want %q", c.Name, i, got.Tag, want.Tag)
			}
			if got.Requirement.Authored != want.Requirement.Authored ||
				got.Requirement.Selectable != want.Requirement.Selectable ||
				got.Requirement.Interesting != want.Requirement.Interesting {
				t.Errorf("case %s entry[%d].requirement = %+v want %+v",
					c.Name, i, got.Requirement, want.Requirement)
			}
			gotIDs := []string{}
			gotOrders := []int{}
			for _, p := range got.Providers {
				gotIDs = append(gotIDs, p.ProviderID)
				gotOrders = append(gotOrders, p.LoadOrder)
			}
			wantIDs := want.ProviderIDs
			if wantIDs == nil {
				wantIDs = []string{}
			}
			wantOrders := want.LoadOrders
			if wantOrders == nil {
				wantOrders = []int{}
			}
			if !reflect.DeepEqual(gotIDs, wantIDs) {
				t.Errorf("case %s entry[%d] provider_ids\n got %v\nwant %v",
					c.Name, i, gotIDs, wantIDs)
			}
			if !reflect.DeepEqual(gotOrders, wantOrders) {
				t.Errorf("case %s entry[%d] load_orders\n got %v\nwant %v",
					c.Name, i, gotOrders, wantOrders)
			}
			var gotWinner *string
			if got.FinalWinner != nil {
				id := got.FinalWinner.ProviderID
				gotWinner = &id
			}
			if (gotWinner == nil) != (want.FinalWinner == nil) ||
				(gotWinner != nil && want.FinalWinner != nil && *gotWinner != *want.FinalWinner) {
				t.Errorf("case %s entry[%d] final_winner mismatch", c.Name, i)
			}
			if (got.Capital == nil) != (want.Capital == nil) ||
				(got.Capital != nil && want.Capital != nil && *got.Capital != *want.Capital) {
				t.Errorf("case %s entry[%d] capital mismatch (got %v want %v)",
					c.Name, i, got.Capital, want.Capital)
			}
			if got.CapitalIsValidState != want.CapitalIsValidState {
				t.Errorf("case %s entry[%d] capital_is_valid_state = %v want %v",
					c.Name, i, got.CapitalIsValidState, want.CapitalIsValidState)
			}
			if (got.ResearchSlots == nil) != (want.ResearchSlots == nil) ||
				(got.ResearchSlots != nil && want.ResearchSlots != nil &&
					*got.ResearchSlots != *want.ResearchSlots) {
				t.Errorf("case %s entry[%d] research_slots mismatch", c.Name, i)
			}
		}
	}
}
