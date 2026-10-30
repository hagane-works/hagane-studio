package toolapi

import (
	"context"
	"fmt"
	"strings"

	"haganeworks/internal/hoi4/exportplan/timeline"
)

type ensureProjectSettingsIn struct {
	ProjectID             int64    `json:"project_id"`
	EnsureExportLanguages []string `json:"ensure_export_languages,omitempty" jsonschema:"language codes to merge into the current export list; existing languages are preserved"`
	StartDate             *string  `json:"start_date,omitempty" jsonschema:"optional explicit HOI4 Y.M.D or Y.M.D.H campaign start"`
	EndDate               *string  `json:"end_date,omitempty" jsonschema:"optional explicit HOI4 Y.M.D or Y.M.D.H campaign end"`
	EnsurePlayableEndDate bool     `json:"ensure_playable_end_date,omitempty" jsonschema:"fill or repair a standalone campaign end date; projects with a base MOD require an explicit end_date"`
	TimelineCoverageDate  string   `json:"timeline_coverage_date,omitempty" jsonschema:"latest bookmark/start date the end date must be after"`
}

type projectSettingsSnapshot struct {
	StartDate       *string  `json:"start_date"`
	EndDate         *string  `json:"end_date"`
	ExportLanguages []string `json:"export_languages"`
}

type ensureProjectSettingsOut struct {
	ProjectID       int64    `json:"project_id"`
	StartDate       *string  `json:"start_date"`
	EndDate         *string  `json:"end_date"`
	ExportLanguages []string `json:"export_languages"`
	ChangedFields   []string `json:"changed_fields"`
	TimelineStatus  string   `json:"timeline_status" jsonschema:"not_checked | configured | requires_explicit_end_date"`
	Warnings        []string `json:"warnings,omitempty"`
}

type projectSettingsEnsureRequest struct {
	Languages     []string
	StartDate     *string
	EndDate       *string
	EnsureEndDate bool
	CoverageDate  string
}

type projectSettingsEnsureResult struct {
	Snapshot       projectSettingsSnapshot
	ChangedFields  []string
	TimelineStatus string
	Warnings       []string
}

var supportedProjectLocales = map[string]bool{
	"en": true, "zh": true, "ru": true, "fr": true, "de": true,
	"es": true, "ja": true, "ko": true, "pl": true, "pt": true,
}

func registerEnsureProjectSettings(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "ensure_project_settings",
		Description: "Merge content languages into project export settings and configure campaign dates. " +
			"Use before localized authoring and scenario/timeline work. Existing export languages are never removed. " +
			"A safe end date can be derived for standalone projects; a base-MOD project needs an explicit end_date.",
	}, func(ctx context.Context, _ *CallToolRequest, in ensureProjectSettingsIn) (*CallToolResult, ensureProjectSettingsOut, error) {
		result, err := ensureProjectSettings(ctx, proxy, in.ProjectID, projectSettingsEnsureRequest{
			Languages: in.EnsureExportLanguages, StartDate: in.StartDate, EndDate: in.EndDate,
			EnsureEndDate: in.EnsurePlayableEndDate, CoverageDate: in.TimelineCoverageDate,
		})
		if err != nil {
			return nil, ensureProjectSettingsOut{}, err
		}
		return nil, ensureProjectSettingsOut{
			ProjectID: in.ProjectID, StartDate: result.Snapshot.StartDate,
			EndDate: result.Snapshot.EndDate, ExportLanguages: result.Snapshot.ExportLanguages,
			ChangedFields: result.ChangedFields, TimelineStatus: result.TimelineStatus,
			Warnings: result.Warnings,
		}, nil
	})
}

func ensureProjectSettings(
	ctx context.Context,
	proxy *RESTProxy,
	projectID int64,
	request projectSettingsEnsureRequest,
) (projectSettingsEnsureResult, error) {
	if projectID <= 0 {
		return projectSettingsEnsureResult{}, fmt.Errorf("project_id must be positive")
	}
	path := fmt.Sprintf("/api/projects/%d", projectID)
	project, err := proxyAuthedCall(ctx, proxy, "GET", path, nil)
	if err != nil {
		return projectSettingsEnsureResult{}, err
	}

	currentLanguages := stringSliceField(project, "export_languages")
	if len(currentLanguages) == 0 {
		currentLanguages = []string{"en"}
	}
	mergedLanguages, languageChanged, err := mergeProjectLocales(currentLanguages, request.Languages)
	if err != nil {
		return projectSettingsEnsureResult{}, err
	}
	body := map[string]any{}
	changedFields := []string{}
	startDate := optionalProjectString(project, "start_date")
	endDate := optionalProjectString(project, "end_date")
	if languageChanged {
		body["export_languages"] = mergedLanguages
		changedFields = append(changedFields, "export_languages")
	}
	if request.StartDate != nil {
		requested := strings.TrimSpace(*request.StartDate)
		if projectStringValue(startDate) != requested {
			body["start_date"] = requested
			changedFields = append(changedFields, "start_date")
		}
		startDate = trimmedStringPointer(requested)
	}
	if request.EndDate != nil {
		requested := strings.TrimSpace(*request.EndDate)
		if projectStringValue(endDate) != requested {
			body["end_date"] = requested
			changedFields = append(changedFields, "end_date")
		}
		endDate = trimmedStringPointer(requested)
	}

	result := projectSettingsEnsureResult{TimelineStatus: "not_checked"}
	if request.EnsureEndDate {
		result.TimelineStatus = "configured"
		coverageDate, err := projectTimelineCoverage(ctx, proxy, projectID, startDate, request.CoverageDate)
		if err != nil {
			return projectSettingsEnsureResult{}, err
		}
		endNeedsRepair := true
		if endDate != nil {
			if parsedEnd, valid := timeline.ParseDate(*endDate); valid && parsedEnd.Compare(coverageDate) > 0 {
				endNeedsRepair = false
			}
		}
		if endNeedsRepair {
			hasBase := len(stringSliceField(project, "dependencies")) > 0 || numericProjectID(project["basemod_template_id"]) > 0
			if hasBase && request.EndDate == nil {
				result.TimelineStatus = "requires_explicit_end_date"
				result.Warnings = append(result.Warnings,
					"Campaign end date was not guessed because this project inherits a base MOD. Call ensure_project_settings with that base MOD's verified end_date before reporting the timeline complete.")
			} else if request.EndDate != nil {
				return projectSettingsEnsureResult{}, fmt.Errorf("end_date must be later than the campaign start/latest bookmark date %s", coverageDate.Canonical())
			} else {
				derived := compatibleCampaignEndDate(coverageDate)
				body["end_date"] = derived
				endDate = &derived
				if !containsString(changedFields, "end_date") {
					changedFields = append(changedFields, "end_date")
				}
			}
		}
	}

	updated := project
	if len(body) > 0 {
		updated, err = proxyAuthedCall(ctx, proxy, "PUT", path, body)
		if err != nil {
			return projectSettingsEnsureResult{}, err
		}

		for key, value := range body {
			if _, present := updated[key]; !present {
				updated[key] = value
			}
		}
	}
	result.Snapshot = projectSettingsSnapshot{
		StartDate:       optionalProjectString(updated, "start_date"),
		EndDate:         optionalProjectString(updated, "end_date"),
		ExportLanguages: stringSliceField(updated, "export_languages"),
	}
	if len(result.Snapshot.ExportLanguages) == 0 {
		result.Snapshot.ExportLanguages = mergedLanguages
	}
	result.ChangedFields = changedFields
	return result, nil
}

func projectTimelineCoverage(
	ctx context.Context,
	proxy *RESTProxy,
	projectID int64,
	startDate *string,
	requestCoverage string,
) (timeline.Date, error) {
	coverage, _ := timeline.ParseDate(timeline.VanillaStartDate)
	consider := func(value, label string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		parsed, ok := timeline.ParseDate(value)
		if !ok {
			return fmt.Errorf("%s must use a valid HOI4 Y.M.D or Y.M.D.H date", label)
		}
		if parsed.Compare(coverage) > 0 {
			coverage = parsed
		}
		return nil
	}
	if startDate != nil {
		if err := consider(*startDate, "start_date"); err != nil {
			return timeline.Date{}, err
		}
	}
	if err := consider(requestCoverage, "timeline_coverage_date"); err != nil {
		return timeline.Date{}, err
	}
	bookmarks, err := proxyAuthedCallList(ctx, proxy, "GET",
		fmt.Sprintf("/api/projects/%d/bookmarks", projectID), nil)
	if err != nil {
		return timeline.Date{}, err
	}
	for _, bookmark := range bookmarks {
		date := stringField(bookmark, "date")
		label := "bookmark date"
		if id := strings.TrimSpace(stringField(bookmark, "bookmark_id")); id != "" {
			label = fmt.Sprintf("bookmark %q date", id)
		}
		if err := consider(date, label); err != nil {
			return timeline.Date{}, err
		}
	}
	return coverage, nil
}

func mergeProjectLocales(current, requested []string) ([]string, bool, error) {
	out := make([]string, 0, len(current)+len(requested))
	seen := map[string]bool{}
	appendLocale := func(raw string, validate bool) error {
		code := strings.ToLower(strings.TrimSpace(raw))
		if code == "" {
			return nil
		}
		if !supportedProjectLocales[code] {
			if validate {
				return fmt.Errorf("unknown HOI4 locale code %q", raw)
			}
			return nil
		}
		if !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
		return nil
	}
	for _, code := range current {
		_ = appendLocale(code, false)
	}
	before := len(out)
	for _, code := range requested {
		if err := appendLocale(code, true); err != nil {
			return nil, false, err
		}
	}
	return out, len(out) != before, nil
}

func compatibleCampaignEndDate(coverage timeline.Date) string {
	vanillaEnd, _ := timeline.ParseDate(timeline.VanillaEndDate)
	if coverage.Year > vanillaEnd.Year ||
		coverage.Year == vanillaEnd.Year && coverage.Month > vanillaEnd.Month ||
		coverage.Year == vanillaEnd.Year && coverage.Month == vanillaEnd.Month && coverage.Day >= vanillaEnd.Day {
		return fmt.Sprintf("%d.1.1", coverage.Year+100)
	}
	return timeline.VanillaEndDate
}

func optionalProjectString(project map[string]any, key string) *string {
	value, ok := project[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}
	value = strings.TrimSpace(value)
	return &value
}

func trimmedStringPointer(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func projectStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
