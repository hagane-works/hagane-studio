package exportcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

func sprintfInt(format string, v int64) string { return fmt.Sprintf(format, v) }

func (s *Store) loadExportIssuesBundle(
	ctx context.Context, projectID int64,
) (exportIssuesBundle, bool, error) {
	var b exportIssuesBundle

	project, found, err := s.GetMetadata(ctx, projectID)
	if err != nil || !found {
		return b, found, err
	}
	b.project = project

	type loader func() error
	for _, load := range []loader{
		func() (e error) { b.characters, e = s.ListCharacters(ctx, projectID, ""); return },
		func() (e error) { b.trees, e = s.loadGateTrees(ctx, projectID); return },
		func() (e error) { b.techs, e = s.loadGateTechs(ctx, projectID); return },
		func() (e error) { b.ideas, e = s.loadGateIdeas(ctx, projectID); return },
		func() (e error) { b.events, e = s.ListEvents(ctx, projectID, nil, ""); return },
		func() (e error) { b.decisions, e = s.ListDecisions(ctx, projectID, nil, ""); return },
		func() (e error) { b.states, e = s.ListStates(ctx, projectID); return },
		func() (e error) { b.historyFiles, e = s.ListHistoryFiles(ctx, projectID); return },
		func() (e error) { b.bookmarks, e = s.ListBookmarks(ctx, projectID); return },
		func() (e error) { b.countries, e = s.ListCountries(ctx, projectID); return },
		func() (e error) { b.rawFiles, e = s.loadGateRawFiles(ctx, projectID); return },
		func() (e error) { b.subUnits, e = s.ListSubUnits(ctx, projectID, nil, nil); return },
		func() (e error) { b.divisions, e = s.ListDivisionTemplates(ctx, projectID, nil, nil); return },
		func() (e error) { b.doctrines, e = s.ListDoctrines(ctx, projectID, ""); return },
		func() (e error) { b.airWings, e = s.ListAirWingGroups(ctx, projectID, nil); return },
		func() (e error) { b.airWingWings, e = s.ListAirWings(ctx, projectID, nil); return },
		func() (e error) { b.policies, e = loadFocusPolicyRows(ctx, s, projectID); return },
	} {
		if err := load(); err != nil {
			return b, true, err
		}
	}
	return b, true, nil
}

func createdRows[T any](rows []T, source func(T) string) []T {
	out := make([]T, 0, len(rows))
	for _, r := range rows {
		if s := source(r); s == "" || s == "created" {
			out = append(out, r)
		}
	}
	return out
}

func (s *Store) assembleExportIssues(
	ctx context.Context, b exportIssuesBundle, opts ExportIssuesOptions,
) ([]map[string]any, error) {
	pid := b.project.ID
	includeImported := EffectiveIncludeImported(
		opts.IncludeImported, len(b.project.Dependencies) > 0,
		b.project.Loaded, s.hasImportedRows(ctx, pid))

	importedDescriptorPaths := NormalizeDescriptorReplacePaths(
		b.project.ImportedReplacePaths)
	var gateReplacePaths []string
	if includeImported {
		gateReplacePaths = EffectiveDescriptorReplacePaths(
			b.project.ImportedReplacePaths, opts.ReplacePaths)
	} else {
		gateReplacePaths = NormalizeDescriptorReplacePaths(opts.ReplacePaths)
	}

	historyForExport, treesForExport, statesForExport := b.historyFiles, b.trees, b.states
	bookmarksForExport, rawForExport := b.bookmarks, b.rawFiles
	subUnits := b.subUnits
	if !includeImported {
		historyForExport = createdRows(b.historyFiles,
			func(h HistoryFile) string { return anyToString(h["source"]) })
		treesForExport = createdRows(b.trees, func(t FocusTree) string { return t.Source })
		statesForExport = createdRows(b.states,
			func(st StateRow) string { return anyToString(st["source"]) })
		bookmarksForExport = createdRows(b.bookmarks,
			func(bm Bookmark) string { return bm.Source })
		subUnits = createdRows(b.subUnits, func(u SubUnit) string { return u.Source })

		kept := make([]gateRawFile, 0, len(b.rawFiles))
		for _, r := range b.rawFiles {
			if r.Source != "imported" {
				kept = append(kept, r)
			}
		}
		rawForExport = kept
	}

	policyRows := make([]FocusPolicyRow, 0, len(b.policies))
	gatePolicies := make([]FocusGatePolicyRow, 0, len(b.policies))
	for _, row := range b.policies {
		policyRows = append(policyRows, FocusPolicyRow{ID: row.ID, TreeIDs: row.Policy.TreeIDs})
		gatePolicies = append(gatePolicies, FocusGatePolicyRow{ID: row.ID, Policy: row.Policy})
	}

	expandedIDs := map[string]bool{}
	for _, id := range ExpandFocusExportCohorts(
		focusTreeIDs(treesForExport), focusTreeIDs(b.trees), policyRows) {
		expandedIDs[id] = true
	}
	cohort := make([]FocusTree, 0, len(b.trees))
	for _, t := range b.trees {
		if expandedIDs[t.TreeID] {
			cohort = append(cohort, t)
		}
	}
	treesForExport = cohort

	isSubmod := len(b.project.Dependencies) > 0
	baseIndex := s.baseIndexLookup(b.project)

	issues := []map[string]any{}
	issues = append(issues, techSurfaceExportIssues(b.project, b.techs)...)
	politicsSnapshot, err := s.PoliticsReferences(ctx, pid)
	if err != nil {
		return nil, err
	}
	politicsIssues, err := politicsExportIssues(b.project, politicsSnapshot)
	if err != nil {
		return nil, err
	}
	issues = append(issues, politicsIssues...)

	issues = append(issues, ValidateProject(ValidateProjectInput{
		Core: ValidateInput{
			ProjectID: pid, ProjectName: b.project.Name,
			CountryTag:   b.project.CountryTag,
			Dependencies: b.project.Dependencies,
			Characters:   projectValidateCharacters(b.characters),
			Trees:        projectValidateTrees(b.trees),
			Techs:        projectValidateTechs(b.techs),
			Ideas:        projectValidateIdeas(b.ideas),
			Events:       projectValidateEvents(b.events),
			Decisions:    projectValidateDecisions(b.decisions),
		},
		States:           projectValidateStates(b.states),
		HistoryFiles:     projectValidateHistoryFiles(b.historyFiles),
		StageStateIDs:    b.project.StageStateIDs,
		IdeologyRegistry: jsonBytes(b.project.IdeologyRegistry),
		IsSubmod:         isSubmod,
	})...)

	var importedUnitPaths []string
	if includeImported {
		for _, r := range b.rawFiles {
			if r.Source == "imported" {
				importedUnitPaths = append(importedUnitPaths, r.DestPath)
			}
		}
	}
	issues = append(issues, CollectSubUnitExportIssues(
		projectSubUnitRows(subUnits), b.project.CountryTag, importedUnitPaths)...)

	gp := GuardrailProject{ID: pid, CountryTag: b.project.CountryTag,
		Dependencies: b.project.Dependencies}
	gChars := projectGuardrailCharacters(b.characters)
	gHistory := projectGuardrailHistoryFiles(b.historyFiles)
	issues = append(issues, GuardrailBaseCollision(gp, gChars,
		projectGuardrailTrees(b.trees), projectGuardrailIdeas(b.ideas),
		projectGuardrailDecisions(b.decisions), baseIndex)...)
	issues = append(issues, GuardrailSubmodIdeology(gp, gChars, gHistory, isSubmod, baseIndex)...)
	issues = append(issues, GuardrailBaseTagWrites(gp, gHistory,
		projectGuardrailDivisions(b.divisions), isSubmod, baseIndex)...)
	issues = append(issues, GuardrailRecruitRoleFilter(gp, gChars, baseIndex)...)
	issues = append(issues, GuardrailTechAndDoctrine(gp,
		projectGuardrailTechs(b.techs), projectGuardrailDoctrines(b.doctrines), isSubmod)...)

	locIssues, err := s.collectLocalisationReferenceIssues(ctx, pid, includeImported)
	if err != nil {
		return nil, err
	}
	issues = append(issues, locIssues...)

	if len(importedDescriptorPaths) > 0 && !includeImported {
		issues = append(issues, descriptorImportedPathIssue(pid, importedDescriptorPaths))
	}

	p0Findings, _, err := CollectP0ExportFindings(CollectP0ExportFindingsInput{
		Project: ExportGateProject{
			ID: pid, Name: b.project.Name, CountryTag: b.project.CountryTag,
			StartDate: derefString(b.project.StartDate), EndDate: derefString(b.project.EndDate),
			ScenarioHistoryPolicy: b.project.ScenarioHistoryPolicy,
			ExportLanguages:       b.project.ExportLanguages,
			Dependencies:          b.project.Dependencies,
			BasemodTemplateID:     b.project.BaseModTemplateID,
		},
		Trees:              treesForExport,
		Bookmarks:          projectGateBookmarks(bookmarksForExport),
		HistoryFiles:       projectHistoryFileRows(historyForExport),
		States:             projectValidateStates(statesForExport),
		RawFiles:           projectGateRawFiles(rawForExport),
		Countries:          projectCountryTags(b.countries),
		FocusProviderTrees: b.trees,
		FocusFilePolicies:  gatePolicies,
		ReplacePaths:       gateReplacePaths,
	})
	if err != nil {
		return nil, err
	}
	for _, f := range p0Findings {
		issues = append(issues, P0FindingToExportIssue(f))
	}

	eventsForExport, decisionsForExport, ideasForExport := b.events, b.decisions, b.ideas
	if !includeImported {
		eventsForExport = createdRows(b.events, func(e Event) string { return e.Source })
		decisionsForExport = createdRows(b.decisions, func(d Decision) string { return d.Source })
		ideasForExport = createdRows(b.ideas, func(i Idea) string { return i.Source })
	}
	for _, f := range RawScriptFindings(eventsForExport, decisionsForExport, ideasForExport, rawScriptRawFiles(rawForExport)) {
		issues = append(issues, P0FindingToExportIssue(f))
	}

	groups, err := s.closureGroups(ctx, pid, includeImported)
	if err != nil {
		return nil, err
	}
	closure := AuditContentLanguageClosure(b.project.ExportLanguages, groups)
	issues = append(issues, contentLanguageClosureBlockers(pid, closure)...)

	airWings := b.airWings
	if !includeImported {
		kept := make([]AirWingGroup, 0, len(airWings))
		wingsByGroup := map[int64][]AirWing{}
		for _, w := range b.airWingWings {
			wingsByGroup[w.GroupID] = append(wingsByGroup[w.GroupID], w)
		}
		for _, g := range airWings {
			if g.Source != "imported" {
				kept = append(kept, g)
				continue
			}

			for _, w := range wingsByGroup[g.ID] {
				if w.Source != "imported" {
					kept = append(kept, g)
					break
				}
			}
		}
		airWings = kept
	}
	issues = append(issues, airWingLocationIssues(airWings, b.airWingWings)...)

	issues = append(issues, stateNoLandIssues(projectValidateStates(statesForExport), rawForExport)...)

	return issues, nil
}

func contentLanguageClosureBlockers(
	projectID int64, audit ContentLanguageClosureAudit,
) []map[string]any {
	out := []map[string]any{}
	for _, d := range audit.Diagnostics {
		out = append(out, map[string]any{
			"level":      "error",
			"code":       "content_language.export_languages_missing",
			"severity":   "blocker",
			"entity_ref": map[string]any{"type": "project", "id": projectID},
			"source": map[string]any{
				"kind": "project", "identifier": strconv.FormatInt(projectID, 10),
			},
			"evidence": map[string]any{
				"language": d.Language, "fields": d.Fields,
			},
			"repair_route": "project.content_languages.add",
			"message": "Structured " + d.Language + " content exists, but that " +
				"language is disabled for this export.",
			"fix_hint": "Add " + d.Language + " to the project's Mod content " +
				"languages, then run the export check again.",
		})
	}
	return out
}

func projectCountryTags(rows []Country) []string {
	out := make([]string, 0, len(rows))
	for _, c := range rows {
		out = append(out, c.Tag)
	}
	return out
}

func rawScriptRawFiles(rows []gateRawFile) []RawScriptRawFile {
	out := make([]RawScriptRawFile, 0, len(rows))
	for _, r := range rows {
		out = append(out, RawScriptRawFile{ID: r.ID, DestPath: r.DestPath, Content: r.Content, IsBinary: r.IsBinary, Source: r.Source})
	}
	return out
}

func projectGateRawFiles(rows []gateRawFile) []FocusGateRawFile {
	out := make([]FocusGateRawFile, 0, len(rows))
	for _, r := range rows {
		out = append(out, FocusGateRawFile{
			ID: r.ID, DestPath: r.DestPath, Content: r.Content,
			IsBinary: r.IsBinary, ContentSHA256: r.ContentSHA256,
			GenerateReplacePath: r.GenerateReplacePath, Source: r.Source,
		})
	}
	return out
}

func focusTreeIDs(rows []FocusTree) []string {
	out := make([]string, 0, len(rows))
	for _, t := range rows {
		out = append(out, t.TreeID)
	}
	return out
}

type gateRawFile struct {
	ID                  int64
	DestPath            string
	Content             string
	IsBinary            bool
	ContentSHA256       string
	GenerateReplacePath bool
	Source              string
}

func (s *Store) baseIndexLookup(p Project) BaseIndexLookup {
	return func(entityType string) map[string]bool { return nil }
}

func projectGateBookmarks(rows []Bookmark) []ExportGateBookmark {
	out := make([]ExportGateBookmark, 0, len(rows))
	for _, b := range rows {
		out = append(out, ExportGateBookmark{
			BookmarkID: b.BookmarkID, Date: b.Date, Source: b.Source,
			DefaultFlag: b.DefaultFlag, DefaultCountry: b.DefaultCountry,
			SelectableCountries: b.SelectableCountries, Filters: b.Filters,
		})
	}
	return out
}

func projectGuardrailCharacters(rows []Character) []GuardrailCharacter {
	out := make([]GuardrailCharacter, 0, len(rows))
	for _, c := range rows {
		out = append(out, GuardrailCharacter{
			Tag: c.Tag, Source: c.Source, Ideology: c.Ideology,
			RoleType: c.RoleType, CountryTag: c.CountryTag,
			HasAdvisorRole: c.HasAdvisorRole,
		})
	}
	return out
}

func projectGuardrailTrees(rows []FocusTree) []GuardrailTree {
	out := make([]GuardrailTree, 0, len(rows))
	for _, t := range rows {
		gt := GuardrailTree{}
		for _, n := range t.Nodes {
			gt.Nodes = append(gt.Nodes, GuardrailFocusNode{FocusID: n.FocusID})
		}
		out = append(out, gt)
	}
	return out
}

func projectGuardrailIdeas(rows []Idea) []GuardrailIdea {
	out := make([]GuardrailIdea, 0, len(rows))
	for _, i := range rows {
		out = append(out, GuardrailIdea{IdeaID: i.IdeaID, Source: i.Source})
	}
	return out
}

func projectGuardrailDecisions(rows []Decision) []GuardrailDecision {
	out := make([]GuardrailDecision, 0, len(rows))
	for _, d := range rows {
		out = append(out, GuardrailDecision{DecisionID: d.DecisionID, Source: d.Source})
	}
	return out
}

func projectGuardrailHistoryFiles(rows []HistoryFile) []GuardrailHistoryFile {
	out := make([]GuardrailHistoryFile, 0, len(rows))
	for _, h := range rows {
		out = append(out, GuardrailHistoryFile{
			CountryTag:             anyToString(h["country_tag"]),
			Source:                 anyToString(h["source"]),
			SetPoliticsRulingParty: anyToString(h["set_politics_ruling_party"]),
		})
	}
	return out
}

func projectGuardrailDivisions(rows []DivisionTemplate) []GuardrailDivisionTemplate {
	out := make([]GuardrailDivisionTemplate, 0, len(rows))
	for _, d := range rows {
		out = append(out, GuardrailDivisionTemplate{
			CountryTag: d.CountryTag, Source: d.Source,
			TemplateName: d.TemplateName, OOBFile: d.OobFile,
		})
	}
	return out
}

func projectGuardrailTechs(rows []Technology) []GuardrailTech {
	out := make([]GuardrailTech, 0, len(rows))
	for _, t := range rows {
		out = append(out, GuardrailTech{TechID: t.TechID, Folder: t.Folder})
	}
	return out
}

func projectGuardrailDoctrines(rows []Doctrine) []GuardrailDoctrine {
	out := make([]GuardrailDoctrine, 0, len(rows))
	for _, d := range rows {
		out = append(out, GuardrailDoctrine{
			EntityType: d.EntityType, EntityKey: d.EntityKey, TrackKey: d.TrackKey,
		})
	}
	return out
}

func projectHistoryFileRows(rows []HistoryFile) []CountryHistoryFileRow {
	out := make([]CountryHistoryFileRow, 0, len(rows))
	for _, h := range rows {
		out = append(out, CountryHistoryFileRow{
			ID:               int64(coerceInt(h["id"])),
			CountryTag:       anyToString(h["country_tag"]),
			Capital:          h["capital"],
			SetResearchSlots: h["set_research_slots"],
			Source:           anyToString(h["source"]),
		})
	}
	return out
}

func projectValidateHistoryFiles(rows []HistoryFile) []ValidateHistoryFile {
	out := make([]ValidateHistoryFile, 0, len(rows))
	for _, h := range rows {
		popularities, popularityErr := normalizeHistoryPopularities(h["set_popularities"])
		out = append(out, ValidateHistoryFile{
			CountryTag:             anyToString(h["country_tag"]),
			Capital:                int64(coerceInt(h["capital"])),
			SetPoliticsRulingParty: anyToString(h["set_politics_ruling_party"]),
			SetPopularities:        popularities,
			SetPopularitiesError:   errorString(popularityErr),
		})
	}
	return out
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func projectValidateStates(rows []StateRow) []ValidateState {
	out := make([]ValidateState, 0, len(rows))
	for _, st := range rows {
		v := ValidateState{
			StateID: int64(coerceInt(st["state_id"])),
			Owner:   anyToString(st["owner"]),
			Name:    anyToString(st["name"]),
		}
		for _, p := range anyToIntSlice(st["provinces"]) {
			v.Provinces = append(v.Provinces, p)
		}
		out = append(out, v)
	}
	return out
}

func anyToIntSlice(v any) []int64 {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]int64, 0, len(items))
	for _, item := range items {
		out = append(out, int64(coerceInt(item)))
	}
	return out
}

func projectSubUnitRows(rows []SubUnit) []SubUnitRow {
	out := make([]SubUnitRow, 0, len(rows))
	for _, u := range rows {
		out = append(out, SubUnitRow{
			SubUnitID: u.SubUnitID, UnitTypes: u.UnitTypes, Categories: u.Categories,
			SourceFile: u.SourceFile, Source: u.Source,
		})
	}
	return out
}

func projectValidateCharacters(rows []Character) []ValidateCharacter {
	out := make([]ValidateCharacter, 0, len(rows))
	for _, c := range rows {
		locs := map[string]any{}
		for k, v := range c.DisplayNameLocs {
			locs[k] = v
		}
		out = append(out, ValidateCharacter{
			ID: c.ID, Tag: c.Tag, CountryTag: c.CountryTag, RoleType: c.RoleType,
			Traits: c.Traits, AdvisorTraits: c.AdvisorTraits, Source: c.Source,
			ReplacesVanillaID: c.ReplacesVanillaID, Ideology: c.Ideology,
			DisplayNameLocs: locs, NameEn: c.NameEn, NameZh: c.NameZh, Name: c.Name,
		})
	}
	return out
}

func projectValidateTechs(rows []Technology) []ValidateTech {
	out := make([]ValidateTech, 0, len(rows))
	for _, t := range rows {
		out = append(out, ValidateTech{
			ID: t.ID, TechID: t.TechID, Prerequisites: flattenPrereqs(t.Prerequisites),
		})
	}
	return out
}

func flattenPrereqs(v any) []string {
	switch value := v.(type) {
	case []string:
		return value
	case []any:
		return anyToStringSlice(value)
	}
	return nil
}

func projectValidateIdeas(rows []Idea) []ValidateIdea {
	out := make([]ValidateIdea, 0, len(rows))
	for _, i := range rows {
		out = append(out, ValidateIdea{
			ID: i.ID, IdeaID: i.IdeaID,
			HasModifier: anyTruthy(i.Modifier), IsDesigner: i.IsDesigner,
			HasRawExtras: anyTruthy(i.RawExtras), HasResearchBonus: anyTruthy(i.ResearchBonus),
		})
	}
	return out
}

func projectValidateEvents(rows []Event) []ValidateEvent {
	out := make([]ValidateEvent, 0, len(rows))
	for _, e := range rows {
		out = append(out, ValidateEvent{
			ID: e.ID, EventID: e.EventID, NamespaceID: &e.NamespaceID,
			OptionCount: len(e.Options), Hidden: e.Hidden,
		})
	}
	return out
}

func projectValidateDecisions(rows []Decision) []ValidateDecision {
	out := make([]ValidateDecision, 0, len(rows))
	for _, d := range rows {
		out = append(out, ValidateDecision{ID: d.ID, DecisionID: d.DecisionID})
	}
	return out
}

func projectValidateTrees(rows []FocusTree) []ValidateFocusTree {
	out := make([]ValidateFocusTree, 0, len(rows))
	for _, t := range rows {
		vt := ValidateFocusTree{ID: t.ID, TreeID: t.TreeID, TreeKind: t.TreeKind}
		for _, n := range t.Nodes {
			vt.Nodes = append(vt.Nodes, ValidateFocusNode{
				ID: n.ID, FocusID: n.FocusID, Prerequisites: n.Prerequisites,
				MutuallyExclusive: n.MutuallyExclusive,
			})
		}
		out = append(out, vt)
	}
	return out
}

func jsonBytes(v map[string]any) []byte {
	if len(v) == 0 {
		return nil
	}
	out, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return out
}
