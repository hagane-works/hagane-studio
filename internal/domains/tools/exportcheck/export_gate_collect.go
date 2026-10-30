package exportcheck

import (
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/exportplan/timeline"
	"haganeworks/internal/hoi4/importsvc/world"
	"regexp"
	"strings"
)

type ExportGateBookmark struct {
	BookmarkID          string
	Date                string
	Source              string
	DefaultFlag         bool
	DefaultCountry      string
	SelectableCountries []map[string]any
	Filters             []string
	EffectivePath       string
	EffectiveSource     string
}

type ExportGateProject struct {
	ID                    any
	Name                  string
	CountryTag            string
	StartDate             string
	EndDate               string
	ScenarioHistoryPolicy string
	ExportLanguages       []string
	Dependencies          []string
	BasemodTemplateID     any
}

func bookmarkCountries(selectable []map[string]any) []BookmarkCountryInput {
	result := []BookmarkCountryInput{}
	for _, item := range selectable {
		tag := strings.ToUpper(strings.TrimSpace(anyToString(item["tag"])))
		if tag == "" {
			continue
		}
		labels := []string{}
		switch raw := item["label"].(type) {
		case string:
			if trimmed := strings.TrimSpace(raw); trimmed != "" {
				labels = append(labels, trimmed)
			}
		default:
			for _, value := range anyToStringSlice(item["label"]) {
				if trimmed := strings.TrimSpace(value); trimmed != "" {
					labels = append(labels, trimmed)
				}
			}
		}
		result = append(result, BookmarkCountryInput{
			Tag:      tag,
			Ideology: strings.TrimSpace(anyToString(item["ideology"])),
			Labels:   labels,

			Interesting: true,
		})
	}
	return result
}

func effectiveBookmarkRows(
	bookmarks []ExportGateBookmark, rawFiles []FocusGateRawFile,
) ([]ExportGateBookmark, []map[string]any) {
	byPath := map[string][]ExportGateBookmark{}
	pathOrder := []string{}
	addPath := func(key string, rows []ExportGateBookmark) {
		if _, seen := byPath[key]; !seen {
			pathOrder = append(pathOrder, key)
		}
		byPath[key] = rows
	}
	for _, bookmark := range bookmarks {
		id := strings.TrimSpace(bookmark.BookmarkID)
		if id == "" {
			continue
		}
		addPath(strings.ToLower("common/bookmarks/"+id+".txt"),
			[]ExportGateBookmark{bookmark})
	}

	findings := []map[string]any{}
	rawPathByKey := map[string][2]string{}
	collisionKeys := map[string]bool{}

	for _, row := range rawFiles {
		path := SafeRawPath(row.DestPath)
		lower := strings.ToLower(path)
		if !strings.HasPrefix(lower, "common/bookmarks/") ||
			!strings.HasSuffix(lower, ".txt") || row.IsBinary {
			continue
		}
		rawID := anyToString(row.ID)
		if row.ID == nil || rawID == "" {
			rawID = path
		}
		if prior, exists := rawPathByKey[lower]; exists && !collisionKeys[lower] {

			collisionKeys[lower] = true
			paths := map[string]bool{prior[0]: true, path: true}
			findings = append(findings, gateFinding(
				"bookmark.raw_path_case_collision", "blocker",
				findingSourceMap("bookmark_raw_file", path, path, "project_raw"),
				map[string]any{
					"paths":        sortedKeys(paths),
					"artifact_ids": []string{prior[1], rawID},
				},
				"bookmark.raw_file.rename_unique_path"))
		} else if !exists {
			rawPathByKey[lower] = [2]string{path, rawID}
		}

		parsed, summary := world.ImportBookmarks(row.Content)

		if len(parsed) == 0 && !IsDeliberateVanillaShadow(row.Content) {
			provider := row.Source
			if provider == "" {
				provider = "project_preserved_raw"
			}
			warnings := []string{}
			if summary != nil {
				warnings = append(warnings, summary.Warnings...)
			}
			findings = append(findings, gateFinding(
				"bookmark.raw_no_bookmark", "blocker",
				findingSourceMap("bookmark_raw_file", path, path, provider),
				map[string]any{"warnings": warnings},
				"bookmark.raw_file.repair_structure"))
		}

		effectiveSource := row.Source
		if effectiveSource == "" {
			effectiveSource = "preserved_raw"
		}
		rows := make([]ExportGateBookmark, 0, len(parsed))
		for _, item := range parsed {
			rows = append(rows, bookmarkFromPyDict(item, path, effectiveSource))
		}
		addPath(lower, rows)
	}

	out := []ExportGateBookmark{}
	for _, key := range pathOrder {
		out = append(out, byPath[key]...)
	}
	return out, findings
}

func bookmarkFromPyDict(item *world.PyDict, path, source string) ExportGateBookmark {
	get := func(key string) any { v, _ := item.Get(key); return v }
	return ExportGateBookmark{
		BookmarkID:          anyToString(get("bookmark_id")),
		Date:                anyToString(get("date")),
		Source:              anyToString(get("source")),
		DefaultFlag:         anyTruthy(get("default_flag")),
		DefaultCountry:      anyToString(get("default_country")),
		SelectableCountries: anyToMapSlice(get("selectable_countries")),
		Filters:             anyToStringSlice(get("filters")),
		EffectivePath:       path,
		EffectiveSource:     source,
	}
}

func modArtifactName(project ExportGateProject) string {
	tag := strings.ToUpper(project.CountryTag)
	fallback := "mod_" + anyToString(project.ID)
	if tag != "" {
		fallback = "mod_" + tag
	}
	return exportengine.SanitizeModDirName(project.Name, fallback)
}

func definesLuaRelPath(modDirName string) string {
	return "common/defines/" + modDirName + "_defines.lua"
}

var definesStartDateRE = regexp.MustCompile(
	`NDefines\.NGame\.START_DATE\s*=\s*["']([^"']+)["']`)

func BuildTimelinePlanForExport(
	project ExportGateProject, bookmarks []ExportGateBookmark,
	rawFiles []FocusGateRawFile,
) TimelinePlan {
	structured := make([]StructuredBookmarkInput, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		id := strings.TrimSpace(bookmark.BookmarkID)
		source := bookmark.Source
		if source == "" {
			source = "project_structured"
		}
		structured = append(structured, StructuredBookmarkInput{
			BookmarkID: bookmark.BookmarkID, Date: bookmark.Date,
			Path: "common/bookmarks/" + id + ".txt", Source: source,
			Emitted: id != "",
		})
	}
	raws := []PreservedRawBookmarkInput{}
	for _, row := range rawFiles {
		path := SafeRawPath(row.DestPath)
		lower := strings.ToLower(path)
		if !strings.HasPrefix(lower, "common/bookmarks/") ||
			!strings.HasSuffix(lower, ".txt") || row.IsBinary {
			continue
		}
		source := row.Source
		if source == "" {
			source = "project_preserved_raw"
		}
		raws = append(raws, PreservedRawBookmarkInput{
			Path: path, Content: row.Content, Source: source,
			SourceHash: row.ContentSHA256, Emitted: true,
		})
	}
	plan := BuildTimelineExportPlan(project.StartDate, structured, raws)

	definesPath := definesLuaRelPath(modArtifactName(project))

	var rawOwner *FocusGateRawFile
	for i := len(rawFiles) - 1; i >= 0; i-- {
		if strings.EqualFold(SafeRawPath(rawFiles[i].DestPath), definesPath) &&
			!rawFiles[i].IsBinary {
			rawOwner = &rawFiles[i]
			break
		}
	}
	if rawOwner == nil {
		return plan
	}

	matches := definesStartDateRE.FindAllStringSubmatch(rawOwner.Content, -1)
	rawDate := ""
	if len(matches) > 0 {
		rawDate = strings.TrimSpace(matches[len(matches)-1][1])
	}
	normalizedRaw := ""
	if rawDate != "" {
		if parsed := ParseHOI4Date(rawDate); parsed != nil {
			normalizedRaw = normaliseDateValue(*parsed)
		}
	}

	plan.ShouldEmitDefines = false
	if normalizedRaw == "" || normalizedRaw != plan.DefinesStartDate {
		provider := rawOwner.Source
		if provider == "" {
			provider = "project_raw"
		}
		var rawEvidence, parsedEvidence any
		if rawDate != "" {
			rawEvidence = rawDate
		}
		if normalizedRaw != "" {
			parsedEvidence = normalizedRaw
		}
		plan.Findings = append(plan.Findings, TimelineFinding{
			Code: "timeline.raw_defines_start_date_conflict", Severity: "blocker",
			Source: timelineFindingSource{Kind: "defines_raw_file",
				Identifier: definesPath, Path: definesPath, Provider: provider},
			Evidence: map[string]any{
				"raw_start_date":        rawEvidence,
				"parsed_raw_start_date": parsedEvidence,
				"expected_start_date":   plan.DefinesStartDate,
			},
			RepairRoute: "timeline.raw_defines.align_start_date",
		})
	}
	return plan
}

func BuildBookmarkQualityForExport(
	bookmarks []ExportGateBookmark, rawFiles []FocusGateRawFile,
	targetVersion string,
) (BookmarkQualityResult, []map[string]any) {
	if targetVersion == "" {
		targetVersion = "1.19.2"
	}
	effectiveRows, rawFindings := effectiveBookmarkRows(bookmarks, rawFiles)
	inputs := make([]BookmarkQualityInput, 0, len(effectiveRows))
	for _, bookmark := range effectiveRows {
		path := bookmark.EffectivePath
		if path == "" {
			path = "common/bookmarks/" + strings.TrimSpace(bookmark.BookmarkID) + ".txt"
		}
		source := bookmark.EffectiveSource
		if source == "" {
			source = bookmark.Source
		}
		if source == "" {
			source = "created"
		}
		inputs = append(inputs, BookmarkQualityInput{
			BookmarkID: bookmark.BookmarkID, Path: path, Source: source,
			Default: bookmark.DefaultFlag, DefaultCountry: bookmark.DefaultCountry,
			Countries: bookmarkCountries(bookmark.SelectableCountries),
			Filters:   bookmark.Filters, TargetVersion: targetVersion,
			Emitted: strings.TrimSpace(bookmark.BookmarkID) != "",
		})
	}
	return EvaluateBookmarkQuality(inputs), rawFindings
}

type CollectP0ExportFindingsInput struct {
	Project            ExportGateProject
	Trees              []FocusTree
	Bookmarks          []ExportGateBookmark
	HistoryFiles       []CountryHistoryFileRow
	States             []ValidateState
	RawFiles           []FocusGateRawFile
	Countries          []string
	FocusProviderTrees []FocusTree
	FocusFilePolicies  []FocusGatePolicyRow
	ReplacePaths       []string
}

func CollectP0ExportFindings(
	in CollectP0ExportFindingsInput,
) ([]map[string]any, timeline.Plan, error) {
	timelinePlan := BuildSharedTimelinePlanForExport(
		in.Project, in.Bookmarks, in.RawFiles, in.ReplacePaths,
	)

	legacyTimelinePlan := BuildTimelinePlanForExport(in.Project, in.Bookmarks, in.RawFiles)
	effectiveReplacePaths := legacyTimelinePlan.DescriptorReplacePaths(in.ReplacePaths)
	quality, rawBookmarkFindings := BuildBookmarkQualityForExport(
		in.Bookmarks, in.RawFiles, "")

	artifactName := modArtifactName(in.Project)
	historyInput := CountryHistoryBuildInput{
		ProjectCountryTag:      in.Project.CountryTag,
		CountryTags:            in.Countries,
		HistoryFiles:           in.HistoryFiles,
		States:                 in.States,
		StructuredArtifactName: artifactName,
	}
	for _, bookmark := range in.Bookmarks {
		historyInput.Bookmarks = append(historyInput.Bookmarks, CountryHistoryBookmark{
			DefaultCountry:      bookmark.DefaultCountry,
			SelectableCountries: bookmark.SelectableCountries,
		})
	}
	for _, row := range in.RawFiles {
		historyInput.RawFiles = append(historyInput.RawFiles, CountryHistoryRawFile{
			ID: row.ID, DestPath: row.DestPath, Content: row.Content,
			IsBinary: row.IsBinary, ContentSHA256: row.ContentSHA256,
		})
	}
	countryHistory := BuildCountryHistoryForExport(historyInput)

	findings := []map[string]any{}
	for _, f := range legacyTimelinePlan.Findings {
		findings = append(findings, f.ToMap())
	}
	projection := timelinePlan.ExportIssuesProjection()
	findings = append(findings, timelineProjectionFindings(projection)...)
	if in.Project.ScenarioHistoryPolicy == ScenarioHistoryPolicyProjectOwnsStartingWorld {
		historyPlan := BuildScenarioHistoryReadOnlyPlan(in.Project, timelinePlan)
		findings = append(findings, scenarioHistoryProjectionFindings(historyPlan)...)
	}

	for _, f := range quality.Findings {
		findings = append(findings, f.ToMap())
	}
	findings = append(findings, rawBookmarkFindings...)
	findings = append(findings, countryHistory.Findings...)

	focusFindings, err := FocusFindings(FocusFindingsInput{
		Project: FocusGateProject{
			CountryTag: in.Project.CountryTag, ExportLanguages: in.Project.ExportLanguages,
			Dependencies:      in.Project.Dependencies,
			BasemodTemplateID: in.Project.BasemodTemplateID,
		},
		Trees: in.Trees, Countries: in.Countries, RawFiles: in.RawFiles,
		ProviderTrees: in.FocusProviderTrees, FilePolicies: in.FocusFilePolicies,
		ReplacePaths: effectiveReplacePaths,
	})
	if err != nil {
		return nil, timelinePlan, err
	}
	return append(findings, focusFindings...), timelinePlan, nil
}
