package exportcheck

import (
	"fmt"
	"haganeworks/internal/hoi4/clausewitz"
	"haganeworks/internal/hoi4/importsvc/world"
	"haganeworks/internal/hoi4/paradox"
	"sort"
	"strings"
)

type exportGateVanilla struct {
	CountryTags  []string          `json:"country_tags"`
	StateOwners  map[string]string `json:"state_owners"`
	StateIDs     []int             `json:"state_ids"`
	tagSet       map[string]bool
	ownerByState map[int]string

	lakes []int
}

type CountryHistoryBuildInput struct {
	ProjectCountryTag  string
	ProjectCountryTags []string
	CountryTags        []string
	Bookmarks          []CountryHistoryBookmark
	HistoryFiles       []CountryHistoryFileRow
	States             []ValidateState
	RawFiles           []CountryHistoryRawFile

	StructuredArtifactName string
	GeneratedHistoryTags   []string
}

type CountryHistoryBookmark struct {
	DefaultCountry      string
	SelectableCountries []map[string]any
}

type CountryHistoryFileRow struct {
	ID               int64
	CountryTag       string
	Capital          any
	SetResearchSlots any
	Source           string
}

type CountryHistoryRawFile struct {
	ID            any
	DestPath      string
	Content       string
	ContentSHA256 string
	IsBinary      bool
	Source        string
}

func projectTags(in CountryHistoryBuildInput) []string {
	var tags []string
	for _, v := range in.ProjectCountryTags {
		tags = append(tags, normTag(v))
	}
	tags = append(tags, normTag(in.ProjectCountryTag))
	for _, v := range in.CountryTags {
		tags = append(tags, normTag(v))
	}
	out := []string{}
	seen := map[string]bool{}
	for _, t := range tags {
		if t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func rawHistoryFields(text string) (bool, any, bool, any) {
	block, recoveries, err := paradox.Parse(text)
	if failed, _ := clausewitz.ParseErrorMessage(recoveries, err); failed || block == nil {
		return false, nil, false, nil
	}
	var lastCapital, lastResearch *paradox.Entry
	var walk func(entries []*paradox.Entry)
	walk = func(entries []*paradox.Entry) {
		for _, e := range entries {
			if e.Key != nil {
				switch *e.Key {
				case "capital":
					lastCapital = e
				case "set_research_slots":
					lastResearch = e
				}
			}
			if e.Block != nil {
				walk(e.Block.Entries)
			}
		}
	}
	walk(block.Entries)

	capitalPresent := lastCapital != nil && lastCapital.Block == nil
	researchPresent := lastResearch != nil && lastResearch.Block == nil
	var capital, research any
	if capitalPresent {
		capital = lastCapital.Scalar.StrValue()
	}
	if researchPresent {
		research = lastResearch.Scalar.StrValue()
	}
	return capitalPresent, capital, researchPresent, research
}

func BuildCountryHistoryForExport(in CountryHistoryBuildInput) CountryHistoryRegistry {
	vanilla := exportGateVanillaOnce()
	tags := projectTags(in)

	type selectFlags struct{ selectable, interesting bool }
	selected := map[string]selectFlags{}
	var selectedOrder []string
	touch := func(tag string, sel, interesting bool) {
		prev, seen := selected[tag]
		if !seen {
			selectedOrder = append(selectedOrder, tag)
		}
		selected[tag] = selectFlags{
			selectable:  sel || prev.selectable,
			interesting: interesting || prev.interesting,
		}
	}
	for _, bm := range in.Bookmarks {
		for _, item := range bm.SelectableCountries {
			tag := normTag(asString(item["tag"]))
			if tag == "" {
				continue
			}

			touch(tag, true, true)
		}
		if def := normTag(bm.DefaultCountry); def != "" {
			prev := selected[def]
			touch(def, true, prev.interesting)
		}
	}

	requirementTags := []string{}
	seenReq := map[string]bool{}
	for _, t := range append(append([]string{}, tags...), selectedOrder...) {
		if t != "" && !seenReq[t] {
			seenReq[t] = true
			requirementTags = append(requirementTags, t)
		}
	}
	inProjectTags := map[string]bool{}
	for _, t := range tags {
		inProjectTags[t] = true
	}
	requirements := make([]CountryRequirement, 0, len(requirementTags))
	for _, tag := range requirementTags {
		f := selected[tag]
		requirements = append(requirements, CountryRequirement{
			Tag:        tag,
			Authored:   inProjectTags[tag] && !vanilla.tagSet[tag],
			Selectable: f.selectable, Interesting: f.interesting,
		})
	}

	var providers []CountryHistoryProvider
	var artifactCandidates []CountryHistoryProvider

	rawHistoryPaths := map[string]bool{}
	for _, row := range in.RawFiles {
		path := SafeRawPath(row.DestPath)
		low := strings.ToLower(path)
		if path != "" && strings.HasPrefix(low, "history/countries/") &&
			strings.HasSuffix(low, ".txt") && !row.IsBinary {
			rawHistoryPaths[low] = true
		}
	}

	for _, tag := range requirementTags {
		if !vanilla.tagSet[tag] {
			continue
		}
		providers = append(providers, CountryHistoryProvider{
			ProviderID: "vanilla:" + tag, Tag: tag, Kind: "vanilla",
			Path:      "history/countries/" + tag + " - <vanilla>.txt",
			LoadOrder: 100, Applies: true,
			ResearchSlotsPresent: true, ResearchSlots: 4,
		})
	}

	for _, row := range in.HistoryFiles {
		tag := normTag(row.CountryTag)
		if tag == "" {
			continue
		}
		imported := row.Source == "imported"
		artifactPath := fmt.Sprintf("history/countries/%s - z%s.txt",
			tag, in.StructuredArtifactName)
		structuredEmitted := !rawHistoryPaths[strings.ToLower(artifactPath)]
		kind := "project_structured"
		loadOrder := 300
		if imported {
			kind = "base_mod"
			loadOrder = 200
		}
		p := CountryHistoryProvider{
			ProviderID: fmt.Sprintf("structured:%v", row.ID), Tag: tag, Kind: kind,
			Path:      "history/countries/" + tag + " - structured.txt",
			LoadOrder: loadOrder, Applies: structuredEmitted,
			CapitalPresent:       positiveInt(row.Capital) != nil,
			Capital:              row.Capital,
			ResearchSlotsPresent: positiveInt(row.SetResearchSlots) != nil,
			ResearchSlots:        row.SetResearchSlots,
		}
		providers = append(providers, p)
		if structuredEmitted {
			c := p
			c.ProviderID = fmt.Sprintf("final:structured:%v", row.ID)
			c.Kind = "final_artifact"
			c.Path = artifactPath
			artifactCandidates = append(artifactCandidates, c)
		}
	}

	for _, row := range in.RawFiles {
		path := SafeRawPath(row.DestPath)
		tag := world.DeriveCountryTagFromHistoryFilename(path)
		if tag == "" || row.IsBinary || !strings.HasSuffix(strings.ToLower(path), ".txt") {
			continue
		}
		capPresent, capital, resPresent, research := rawHistoryFields(row.Content)
		imported := row.Source == "imported"
		kind := "project_preserved_raw"
		loadOrder := 400
		if imported {
			kind = "base_mod"
			loadOrder = 220
		}
		id := row.ID
		if id == nil {
			id = path
		}
		p := CountryHistoryProvider{
			ProviderID: fmt.Sprintf("raw:%v", id), Tag: tag, Kind: kind,
			Path: path, SourceHash: row.ContentSHA256,
			LoadOrder: loadOrder, Applies: true,
			CapitalPresent: capPresent, Capital: capital,
			ResearchSlotsPresent: resPresent, ResearchSlots: research,
		}
		providers = append(providers, p)
		c := p
		c.ProviderID = fmt.Sprintf("final:raw:%v", id)
		c.Kind = "final_artifact"
		artifactCandidates = append(artifactCandidates, c)
	}

	structuredArtifactTags := map[string]bool{}
	for _, p := range artifactCandidates {
		if strings.HasPrefix(p.ProviderID, "final:structured:") {
			structuredArtifactTags[p.Tag] = true
		}
	}
	seenGenerated := map[string]bool{}
	for _, v := range in.GeneratedHistoryTags {
		tag := normTag(v)
		if tag == "" || seenGenerated[tag] || structuredArtifactTags[tag] {
			seenGenerated[tag] = true
			continue
		}
		seenGenerated[tag] = true
		path := fmt.Sprintf("history/countries/%s - z%s.txt", tag, in.StructuredArtifactName)
		if rawHistoryPaths[strings.ToLower(path)] {
			continue
		}
		artifactCandidates = append(artifactCandidates, CountryHistoryProvider{
			ProviderID: "final:generated:" + tag, Tag: tag, Kind: "final_artifact",
			Path: path, LoadOrder: 0, Applies: true,
		})
	}

	sorted := append([]CountryHistoryProvider(nil), artifactCandidates...)
	sort.SliceStable(sorted, func(i, j int) bool {
		li, lj := strings.ToLower(sorted[i].Path), strings.ToLower(sorted[j].Path)
		if li != lj {
			return li < lj
		}
		ri := 0
		if strings.HasPrefix(sorted[i].ProviderID, "final:raw:") {
			ri = 1
		}
		rj := 0
		if strings.HasPrefix(sorted[j].ProviderID, "final:raw:") {
			rj = 1
		}
		if ri != rj {
			return ri < rj
		}
		return strings.ToLower(sorted[i].ProviderID) < strings.ToLower(sorted[j].ProviderID)
	})
	for index, p := range sorted {
		p.LoadOrder = 1000 + index + 1
		providers = append(providers, p)
	}

	winnerByTag := map[string]int{}
	for i, p := range providers {
		prior, seen := winnerByTag[p.Tag]
		if !seen || providers[prior].LoadOrder <= p.LoadOrder {
			winnerByTag[p.Tag] = i
		}
	}
	for i := range providers {
		providers[i].FinalWinner = winnerByTag[providers[i].Tag] == i
	}

	validStateIDs := append([]int(nil), vanilla.StateIDs...)
	ownerByID := map[int]string{}
	for k, v := range vanilla.ownerByState {
		ownerByID[k] = v
	}
	for _, st := range in.States {
		if st.StateID > 0 {
			validStateIDs = append(validStateIDs, int(st.StateID))
			if owner := normTag(st.Owner); owner != "" {
				ownerByID[int(st.StateID)] = owner
			}
		}
	}

	var rawStateFindings []map[string]any
	for _, row := range in.RawFiles {
		path := SafeRawPath(row.DestPath)
		low := strings.ToLower(path)
		if !strings.HasPrefix(low, "history/states/") ||
			!strings.HasSuffix(low, ".txt") || row.IsBinary {
			continue
		}
		block, recoveries, err := paradox.Parse(row.Content)
		if failed, msg := clausewitz.ParseErrorMessage(recoveries, err); failed {
			provider := row.Source
			if provider == "" {
				provider = "project_raw"
			}
			rawStateFindings = append(rawStateFindings, gateFinding(
				"state.raw_parse_failed", "blocker",
				findingSourceMap("state_raw_file", path, path, provider),
				map[string]any{"error_type": "ParseError", "error": msg},
				"state.raw_file.repair_syntax"))
			continue
		}
		stateBlock := paradoxBlock(block.Entries, "state")
		if stateBlock == nil {
			continue
		}
		var idText string
		for _, e := range stateBlock.Entries {
			if e.Key != nil && *e.Key == "id" && e.Block == nil {
				idText = e.Scalar.StrValue()
			}
		}
		sid := positiveInt(idText)
		if sid == nil {
			continue
		}
		validStateIDs = append(validStateIDs, *sid)
		if hist := paradoxBlock(stateBlock.Entries, "history"); hist != nil {
			var lastOwner *paradox.Entry
			for _, e := range hist.Entries {
				if e.Key != nil && *e.Key == "owner" {
					lastOwner = e
				}
			}
			if lastOwner != nil && lastOwner.Block == nil {
				if owner := normTag(lastOwner.Scalar.StrValue()); owner != "" {
					ownerByID[*sid] = owner
				}
			}
		}
	}

	registry := BuildCountryHistoryRegistry(requirements, providers, validStateIDs, ownerByID)
	registry.Findings = append(registry.Findings, rawStateFindings...)
	return registry
}
