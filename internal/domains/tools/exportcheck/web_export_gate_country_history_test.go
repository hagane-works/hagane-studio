package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestCountryHistoryRegistryParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "country_history_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name         string `json:"name"`
			Requirements []struct {
				Tag         string `json:"tag"`
				Authored    bool   `json:"authored"`
				Selectable  bool   `json:"selectable"`
				Interesting bool   `json:"interesting"`
			} `json:"requirements"`
			Providers []struct {
				ProviderID           string `json:"provider_id"`
				Tag                  string `json:"tag"`
				Kind                 string `json:"kind"`
				Path                 string `json:"path"`
				SourceHash           string `json:"source_hash"`
				LoadOrder            int    `json:"load_order"`
				Applies              bool   `json:"applies"`
				FinalWinner          bool   `json:"final_winner"`
				CapitalPresent       bool   `json:"capital_present"`
				Capital              any    `json:"capital"`
				ResearchSlotsPresent bool   `json:"research_slots_present"`
				ResearchSlots        any    `json:"research_slots"`
			} `json:"providers"`
			ValidStateIDs  []int             `json:"valid_state_ids"`
			StateOwnerByID map[string]string `json:"state_owner_by_id"`
			Findings       json.RawMessage   `json:"findings"`
			Entries        []struct {
				Tag                   string   `json:"tag"`
				FinalWinner           *string  `json:"final_winner"`
				FinalWinnerExplicit   bool     `json:"final_winner_explicit"`
				CapitalProvider       *string  `json:"capital_provider"`
				Capital               *int     `json:"capital"`
				CapitalIsValidState   bool     `json:"capital_is_valid_state"`
				ResearchSlotsProvider *string  `json:"research_slots_provider"`
				ResearchSlots         *int     `json:"research_slots"`
				SourceHashes          []string `json:"source_hashes"`
			} `json:"entries"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	for _, c := range fx.Cases {
		var reqs []CountryRequirement
		for _, r := range c.Requirements {
			reqs = append(reqs, CountryRequirement{
				Tag: r.Tag, Authored: r.Authored,
				Selectable: r.Selectable, Interesting: r.Interesting})
		}
		var provs []CountryHistoryProvider
		for _, p := range c.Providers {
			provs = append(provs, CountryHistoryProvider{
				ProviderID: p.ProviderID, Tag: p.Tag, Kind: p.Kind, Path: p.Path,
				SourceHash: p.SourceHash, LoadOrder: p.LoadOrder,
				Applies: p.Applies, FinalWinner: p.FinalWinner,
				CapitalPresent: p.CapitalPresent, Capital: p.Capital,
				ResearchSlotsPresent: p.ResearchSlotsPresent,
				ResearchSlots:        p.ResearchSlots,
			})
		}
		owners := map[int]string{}
		for k, v := range c.StateOwnerByID {
			id, err := strconv.Atoi(k)
			if err != nil {
				t.Fatal(err)
			}
			owners[id] = v
		}

		reg := BuildCountryHistoryRegistry(reqs, provs, c.ValidStateIDs, owners)

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
			checkPtrStr := func(label string, gotP *CountryHistoryProvider, wantS *string) {
				var gotS *string
				if gotP != nil {
					id := gotP.ProviderID
					gotS = &id
				}
				if (gotS == nil) != (wantS == nil) ||
					(gotS != nil && wantS != nil && *gotS != *wantS) {
					t.Errorf("case %s entry[%d].%s mismatch", c.Name, i, label)
				}
			}
			checkPtrStr("final_winner", got.FinalWinner, want.FinalWinner)
			checkPtrStr("capital_provider", got.CapitalProvider, want.CapitalProvider)
			checkPtrStr("research_slots_provider", got.ResearchSlotsProvider,
				want.ResearchSlotsProvider)
			if got.FinalWinnerExplicit != want.FinalWinnerExplicit {
				t.Errorf("case %s entry[%d].final_winner_explicit = %v want %v",
					c.Name, i, got.FinalWinnerExplicit, want.FinalWinnerExplicit)
			}
			checkPtrInt := func(label string, g, w *int) {
				if (g == nil) != (w == nil) || (g != nil && w != nil && *g != *w) {
					t.Errorf("case %s entry[%d].%s mismatch (got %v want %v)",
						c.Name, i, label, g, w)
				}
			}
			checkPtrInt("capital", got.Capital, want.Capital)
			checkPtrInt("research_slots", got.ResearchSlots, want.ResearchSlots)
			if got.CapitalIsValidState != want.CapitalIsValidState {
				t.Errorf("case %s entry[%d].capital_is_valid_state = %v want %v",
					c.Name, i, got.CapitalIsValidState, want.CapitalIsValidState)
			}

			gotHashes := []string{}
			for _, p := range got.Providers {
				gotHashes = append(gotHashes, p.SourceHash)
			}
			wantHashes := want.SourceHashes
			if wantHashes == nil {
				wantHashes = []string{}
			}
			if !reflect.DeepEqual(gotHashes, wantHashes) {
				t.Errorf("case %s entry[%d] source_hashes\n got %v\nwant %v",
					c.Name, i, gotHashes, wantHashes)
			}
		}
	}
}

func TestCapitalChecksAreIndependent(t *testing.T) {
	reqs := []CountryRequirement{{Tag: "XXX", Authored: true, Selectable: true}}
	invalid := BuildCountryHistoryRegistry(reqs,
		[]CountryHistoryProvider{{ProviderID: "p", Tag: "XXX", Applies: true,
			CapitalPresent: true, Capital: 99999}},
		[]int{16}, map[int]string{99999: "FRA"})
	codes := map[string]bool{}
	for _, f := range invalid.Findings {
		codes[f["code"].(string)] = true
	}
	if !codes["country.capital_invalid_state"] {
		t.Error("无效首都应当报 capital_invalid_state")
	}
	if codes["country.capital_not_owned"] {
		t.Error("首都无效时不该顺带报归属")
	}

	owned := BuildCountryHistoryRegistry(reqs,
		[]CountryHistoryProvider{{ProviderID: "p", Tag: "XXX", Applies: true,
			CapitalPresent: true, Capital: 16}},
		[]int{16}, map[int]string{16: "FRA"})
	codes2 := map[string]bool{}
	for _, f := range owned.Findings {
		codes2[f["code"].(string)] = true
	}
	if !codes2["country.capital_not_owned"] {
		t.Error("有效但归别人的首都应当提醒")
	}
	if codes2["country.capital_invalid_state"] {
		t.Error("有效首都不该报无效")
	}
}

func TestRequiresBootableHistory(t *testing.T) {
	for _, tc := range []struct {
		req  CountryRequirement
		want bool
	}{
		{CountryRequirement{Authored: true, Selectable: true}, true},
		{CountryRequirement{Authored: true, Interesting: true}, true},
		{CountryRequirement{Authored: true}, false},
		{CountryRequirement{Selectable: true}, false},
		{CountryRequirement{}, false},
	} {
		if got := tc.req.RequiresBootableHistory(); got != tc.want {
			t.Errorf("%+v → %v want %v", tc.req, got, tc.want)
		}
	}
}
