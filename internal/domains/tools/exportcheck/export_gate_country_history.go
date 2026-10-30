package exportcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

type CountryRequirement struct {
	Tag         string
	Authored    bool
	Selectable  bool
	Interesting bool
}

func (r CountryRequirement) RequiresBootableHistory() bool {
	return r.Authored && (r.Selectable || r.Interesting)
}

type CountryHistoryProvider struct {
	ProviderID           string
	Tag                  string
	Kind                 string
	Path                 string
	SourceHash           string
	LoadOrder            int
	Applies              bool
	FinalWinner          bool
	CapitalPresent       bool
	Capital              any
	ResearchSlotsPresent bool
	ResearchSlots        any
}

type EffectiveCountryHistory struct {
	Tag                   string
	Requirement           CountryRequirement
	Providers             []CountryHistoryProvider
	FinalWinner           *CountryHistoryProvider
	FinalWinnerExplicit   bool
	CapitalProvider       *CountryHistoryProvider
	Capital               *int
	CapitalIsValidState   bool
	ResearchSlotsProvider *CountryHistoryProvider
	ResearchSlots         *int
	Findings              []map[string]any
}

type CountryHistoryRegistry struct {
	Entries  []EffectiveCountryHistory
	Findings []map[string]any
}

func normTag(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func positiveInt(raw any) *int {
	switch v := raw.(type) {
	case bool:
		return nil
	case int:
		if v > 0 {
			return &v
		}
	case int64:
		i := int(v)
		if i > 0 {
			return &i
		}
	case float64:

		i := int(v)
		if float64(i) == v && i > 0 {
			return &i
		}
	case string:
		token := strings.TrimSpace(v)
		if token != "" && isAllDigits(token) {
			if i, err := strconv.Atoi(token); err == nil && i > 0 {
				return &i
			}
		}
	}
	return nil
}

func providerHash(p CountryHistoryProvider) string {
	payload := map[string]any{
		"provider_id": p.ProviderID, "tag": p.Tag, "kind": p.Kind,
		"path": p.Path, "load_order": p.LoadOrder, "applies": p.Applies,
		"capital_present": p.CapitalPresent, "capital": jsonDefaultStr(p.Capital),
		"research_slots_present": p.ResearchSlotsPresent,
		"research_slots":         jsonDefaultStr(p.ResearchSlots),
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func jsonDefaultStr(v any) any { return v }

func findingSourceMap(kind, identifier, path, provider string) map[string]any {
	out := map[string]any{"kind": kind, "identifier": identifier}
	if path != "" {
		out["path"] = path
	}
	if provider != "" {
		out["provider"] = provider
	}
	return out
}

func countryHistorySource(p CountryHistoryProvider, tag string) map[string]any {
	return findingSourceMap("country_history", tag, p.Path, p.Kind)
}

func gateFinding(code, severity string, source map[string]any,
	evidence map[string]any, repairRoute string) map[string]any {
	return map[string]any{
		"code": code, "severity": severity, "source": source,
		"evidence": evidence, "repair_route": repairRoute,
	}
}

func BuildCountryHistoryRegistry(requirements []CountryRequirement,
	providers []CountryHistoryProvider, validStateIDs []int,
	stateOwnerByID map[int]string) CountryHistoryRegistry {

	requirementByTag := map[string]CountryRequirement{}
	for _, req := range requirements {
		tag := normTag(req.Tag)
		if tag != "" {
			req.Tag = tag
			requirementByTag[tag] = req
		}
	}
	providersByTag := map[string][]CountryHistoryProvider{}
	for _, p := range providers {
		tag := normTag(p.Tag)
		if tag == "" {
			continue
		}
		n := p
		n.Tag = tag
		if strings.TrimSpace(n.SourceHash) == "" {
			n.SourceHash = providerHash(p)
		} else {
			n.SourceHash = strings.TrimSpace(n.SourceHash)
		}
		providersByTag[tag] = append(providersByTag[tag], n)
		if _, exists := requirementByTag[tag]; !exists {
			requirementByTag[tag] = CountryRequirement{Tag: tag}
		}
	}

	validStates := map[int]bool{}
	for _, id := range validStateIDs {
		if id > 0 {
			validStates[id] = true
		}
	}
	owners := map[int]string{}
	for id, owner := range stateOwnerByID {
		owners[id] = normTag(owner)
	}

	tags := make([]string, 0, len(requirementByTag))
	for tag := range requirementByTag {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	entries := []EffectiveCountryHistory{}
	allFindings := []map[string]any{}
	for _, tag := range tags {
		requirement := requirementByTag[tag]
		applicable := []CountryHistoryProvider{}
		for _, item := range providersByTag[tag] {
			if item.Applies {
				applicable = append(applicable, item)
			}
		}
		sort.SliceStable(applicable, func(i, j int) bool {
			if applicable[i].LoadOrder != applicable[j].LoadOrder {
				return applicable[i].LoadOrder < applicable[j].LoadOrder
			}
			return applicable[i].ProviderID < applicable[j].ProviderID
		})
		findings := []map[string]any{}

		var explicitWinners []CountryHistoryProvider
		for _, item := range applicable {
			if item.FinalWinner {
				explicitWinners = append(explicitWinners, item)
			}
		}
		finalWinnerExplicit := len(explicitWinners) == 1
		var finalWinner *CountryHistoryProvider
		switch {
		case len(explicitWinners) > 1:
			w := explicitWinners[len(explicitWinners)-1]
			finalWinner = &w
			ids := []string{}
			paths := []string{}
			for _, item := range explicitWinners {
				ids = append(ids, item.ProviderID)
				paths = append(paths, item.Path)
			}
			findings = append(findings, gateFinding(
				"country.history_final_winner_ambiguous", "blocker",
				findingSourceMap("country_history_set", tag, "", ""),
				map[string]any{"provider_ids": ids, "paths": paths},
				"country.history.manifest.resolve_winner"))
		case len(explicitWinners) == 1:
			w := explicitWinners[0]
			finalWinner = &w
		default:
			if len(applicable) > 0 {
				w := applicable[len(applicable)-1]
				finalWinner = &w
				findings = append(findings, gateFinding(
					"country.history_final_winner_inferred", "warning",
					findingSourceMap("country_history_set", tag, "", ""),
					map[string]any{"provider_id": w.ProviderID, "path": w.Path,
						"load_order": w.LoadOrder},
					"country.history.manifest.attach_final_winner"))
			}
		}

		var capitalProvider *CountryHistoryProvider
		for i := len(applicable) - 1; i >= 0; i-- {
			if applicable[i].CapitalPresent {
				c := applicable[i]
				capitalProvider = &c
				break
			}
		}
		var capital *int
		if capitalProvider != nil {
			capital = positiveInt(capitalProvider.Capital)
		}
		capitalValid := capital != nil && validStates[*capital]

		if capitalProvider != nil && !capitalValid {
			var parsed any
			if capital != nil {
				parsed = *capital
			}
			findings = append(findings, gateFinding(
				"country.capital_invalid_state", "blocker",
				countryHistorySource(*capitalProvider, tag),
				map[string]any{
					"capital_token": capitalProvider.Capital, "parsed_capital": parsed,
					"valid_state_id": false, "provider_id": capitalProvider.ProviderID,
					"source_hash": capitalProvider.SourceHash,
				},
				"country.history.capital.choose_state_id"))
		} else if capitalProvider == nil && requirement.RequiresBootableHistory() {
			provs := []map[string]any{}
			for _, item := range applicable {
				provs = append(provs, map[string]any{
					"provider_id": item.ProviderID, "kind": item.Kind,
					"path": item.Path, "source_hash": item.SourceHash,
				})
			}
			findings = append(findings, gateFinding(
				"country.capital_missing", "blocker",
				findingSourceMap("country", tag, "", ""),
				map[string]any{"authored": requirement.Authored,
					"selectable":  requirement.Selectable,
					"interesting": requirement.Interesting, "providers": provs},
				"country.history.capital.add"))
		}

		if capitalValid {
			if owner, known := owners[*capital]; known && owner != tag {
				findings = append(findings, gateFinding(
					"country.capital_not_owned", "warning",
					countryHistorySource(*capitalProvider, tag),
					map[string]any{"capital": *capital, "owner": owner,
						"country_tag": tag, "provider_id": capitalProvider.ProviderID},
					"country.history.capital.review_ownership"))
			}
		}

		var researchProvider *CountryHistoryProvider
		for i := len(applicable) - 1; i >= 0; i-- {
			if applicable[i].ResearchSlotsPresent {
				r := applicable[i]
				researchProvider = &r
				break
			}
		}
		var researchSlots *int
		if researchProvider != nil {
			researchSlots = positiveInt(researchProvider.ResearchSlots)
		}
		if researchProvider == nil && (requirement.Selectable || requirement.Interesting) {
			ids := []string{}
			for _, item := range applicable {
				ids = append(ids, item.ProviderID)
			}
			findings = append(findings, gateFinding(
				"country.research_slots_missing", "warning",
				findingSourceMap("country", tag, "", ""),
				map[string]any{"providers": ids, "engine_blocker": false},
				"country.history.research_slots.add"))
		} else if researchProvider != nil && researchSlots == nil {
			findings = append(findings, gateFinding(
				"country.research_slots_invalid", "warning",
				countryHistorySource(*researchProvider, tag),
				map[string]any{"research_slots_token": researchProvider.ResearchSlots,
					"provider_id": researchProvider.ProviderID, "engine_blocker": false},
				"country.history.research_slots.repair"))
		}

		entries = append(entries, EffectiveCountryHistory{
			Tag: tag, Requirement: requirement, Providers: applicable,
			FinalWinner: finalWinner, FinalWinnerExplicit: finalWinnerExplicit,
			CapitalProvider: capitalProvider, Capital: capital,
			CapitalIsValidState:   capitalValid,
			ResearchSlotsProvider: researchProvider, ResearchSlots: researchSlots,
			Findings: findings,
		})
		allFindings = append(allFindings, findings...)
	}
	return CountryHistoryRegistry{Entries: entries, Findings: allFindings}
}
