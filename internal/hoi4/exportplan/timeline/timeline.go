package timeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	VanillaStartDate = "1936.1.1.12"
	VanillaEndDate   = "1949.1.1"
	BookmarksPath    = "common/bookmarks"
)

type Date struct {
	Year  int
	Month int
	Day   int
	Hour  int
}

func ParseDate(value string) (Date, bool) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) != 3 && len(parts) != 4 {
		return Date{}, false
	}
	values := make([]int, len(parts))
	for i, part := range parts {
		parsed, err := strconv.Atoi(part)
		if err != nil {
			return Date{}, false
		}
		values[i] = parsed
	}
	date := Date{Year: values[0], Month: values[1], Day: values[2], Hour: 12}
	if len(values) == 4 {
		date.Hour = values[3]
	}
	if date.Year <= 0 || date.Month < 1 || date.Month > 12 ||
		date.Day < 1 || date.Day > 31 || date.Hour < 0 || date.Hour > 23 {
		return Date{}, false
	}
	return date, true
}

func (d Date) Canonical() string {
	return strconv.Itoa(d.Year) + "." + strconv.Itoa(d.Month) + "." +
		strconv.Itoa(d.Day) + "." + strconv.Itoa(d.Hour)
}

func (d Date) DefinesValue() string {
	base := strconv.Itoa(d.Year) + "." + strconv.Itoa(d.Month) + "." + strconv.Itoa(d.Day)
	if d.Hour == 12 {
		return base
	}
	return base + "." + strconv.Itoa(d.Hour)
}

func (d Date) EqualYMD(other Date) bool {
	return d.Year == other.Year && d.Month == other.Month && d.Day == other.Day
}

func (d Date) Compare(other Date) int {
	left := [...]int{d.Year, d.Month, d.Day, d.Hour}
	right := [...]int{other.Year, other.Month, other.Day, other.Hour}
	for i := range left {
		if left[i] < right[i] {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
	}
	return 0
}

type BookmarkEvidence struct {
	ID         string
	Path       string
	Source     string
	SourceHash string
	DateTokens []string
	Emitted    bool
}

type RawDefinesEvidence struct {
	ID         string
	Path       string
	Source     string
	SourceHash string
	Content    string
	StartDate  string
	EndDate    string
	Emitted    bool
}

type Input struct {
	ProjectStartDate string
	RequestedEndDate string
	HasDependencies  bool

	StructuredBookmarks []BookmarkEvidence
	RawBookmarks        []BookmarkEvidence
	RawDefines          []RawDefinesEvidence

	RequestedReplacePaths []string
	InheritedReplacePaths []string

	VanillaStartDate string
	VanillaEndDate   string
	ModDirectory     string
	DefinesPath      string
}

type Finding struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	SourceKind string `json:"source_kind,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
	Path       string `json:"path,omitempty"`
	Expected   string `json:"expected,omitempty"`
	Actual     string `json:"actual,omitempty"`
}

type BookmarkSourceEvidence struct {
	Kind           string   `json:"kind"`
	ID             string   `json:"id"`
	Path           string   `json:"path"`
	Source         string   `json:"source"`
	SourceHash     string   `json:"source_hash"`
	DateTokens     []string `json:"date_tokens"`
	ParseableDates []string `json:"parseable_dates"`
	ParseError     string   `json:"parse_error,omitempty"`
	Emitted        bool     `json:"emitted"`
	Winner         bool     `json:"winner"`
	ShadowedBy     string   `json:"shadowed_by,omitempty"`
}

type RawDefinesWinner struct {
	ID                  string `json:"id"`
	Path                string `json:"path"`
	Source              string `json:"source"`
	SourceHash          string `json:"source_hash"`
	StartDate           string `json:"start_date,omitempty"`
	EndDate             string `json:"end_date,omitempty"`
	ParsedStartDate     string `json:"parsed_start_date,omitempty"`
	ParsedEndDate       string `json:"parsed_end_date,omitempty"`
	ReplacesGenerated   bool   `json:"replaces_generated"`
	LoadsAfterGenerated bool   `json:"loads_after_generated"`
	StartID             string `json:"start_id,omitempty"`
	StartPath           string `json:"start_path,omitempty"`
	StartSource         string `json:"start_source,omitempty"`
	StartSourceHash     string `json:"start_source_hash,omitempty"`
	EndID               string `json:"end_id,omitempty"`
	EndPath             string `json:"end_path,omitempty"`
	EndSource           string `json:"end_source,omitempty"`
	EndSourceHash       string `json:"end_source_hash,omitempty"`
}

type DefinesArtifact struct {
	Path         string `json:"path"`
	Content      string `json:"content"`
	StartDate    string `json:"start_date,omitempty"`
	EndDate      string `json:"end_date,omitempty"`
	EmitsEndDate bool   `json:"emits_end_date"`
}

type Plan struct {
	EffectiveStartDate    string                   `json:"effective_start_date,omitempty"`
	EffectiveEndDate      string                   `json:"effective_end_date,omitempty"`
	EndDateSource         string                   `json:"end_date_source,omitempty"`
	EmitsBookmark         bool                     `json:"emits_bookmark"`
	EmitDefines           bool                     `json:"emit_defines"`
	Defines               DefinesArtifact          `json:"defines"`
	EffectiveReplacePaths []string                 `json:"effective_replace_paths"`
	Findings              []Finding                `json:"findings"`
	BookmarkSources       []BookmarkSourceEvidence `json:"bookmark_sources"`
	RawDefinesWinner      *RawDefinesWinner        `json:"raw_defines_winner,omitempty"`
	PlanSHA256            string                   `json:"plan_sha256"`
}

var defineDateAssignmentRE = regexp.MustCompile(
	`NDefines\.NGame\.(START_DATE|END_DATE)\s*=\s*["']([^"']+)["']`)

func ExtractDefineDates(content string) (startDate, endDate string) {
	lines := make([]string, 0)
	for _, line := range strings.Split(content, "\n") {
		if comment := strings.Index(line, "--"); comment >= 0 {
			line = line[:comment]
		}
		lines = append(lines, line)
	}
	for _, match := range defineDateAssignmentRE.FindAllStringSubmatch(strings.Join(lines, "\n"), -1) {
		if len(match) != 3 {
			continue
		}
		switch match[1] {
		case "START_DATE":
			startDate = strings.TrimSpace(match[2])
		case "END_DATE":
			endDate = strings.TrimSpace(match[2])
		}
	}
	return startDate, endDate
}

func DefinesPathForModDirectory(modDirectory string) string {
	directory := strings.TrimSpace(strings.ReplaceAll(modDirectory, "\\", "/"))
	directory = strings.Trim(directory, "/")
	if directory == "" {
		directory = "hagane"
	}
	return "common/defines/" + directory + "_defines.lua"
}

func Build(input Input) Plan {
	vanillaStart := parseOrDefault(input.VanillaStartDate, VanillaStartDate)
	vanillaEnd := parseOrDefault(input.VanillaEndDate, VanillaEndDate)
	definesPath := normalisePath(input.DefinesPath)
	if definesPath == "" {
		definesPath = DefinesPathForModDirectory(input.ModDirectory)
	}

	sources := make([]BookmarkSourceEvidence, 0, len(input.StructuredBookmarks)+len(input.RawBookmarks))
	for _, item := range input.StructuredBookmarks {
		sources = append(sources, evidenceFromBookmark("structured", item))
	}
	for _, item := range input.RawBookmarks {
		sources = append(sources, evidenceFromBookmark("preserved_raw", item))
	}

	winnerIndexByPath := map[string]int{}
	for i := range sources {
		if sources[i].Emitted {
			winnerIndexByPath[strings.ToLower(sources[i].Path)] = i
		}
	}
	for i := range sources {
		if !sources[i].Emitted {
			continue
		}
		winner := winnerIndexByPath[strings.ToLower(sources[i].Path)]
		if winner == i {
			sources[i].Winner = true
		} else {
			sources[i].ShadowedBy = sources[winner].ID
		}
	}

	effectiveSources := make([]BookmarkSourceEvidence, 0, len(sources))
	for _, source := range sources {
		if source.Emitted && source.Winner {
			effectiveSources = append(effectiveSources, source)
		}
	}

	findings := make([]Finding, 0)
	for _, source := range effectiveSources {
		if source.ParseError != "" || len(source.ParseableDates) == 0 {
			code := "timeline.structured_bookmark_date_unknown"
			if source.Kind == "preserved_raw" {
				code = "timeline.raw_bookmark_date_unknown"
			}
			findings = append(findings, Finding{
				Code: code, Severity: "attention", SourceKind: "bookmark", SourceID: source.ID,
				Path: source.Path, Actual: source.ParseError,
			})
		}
	}

	projectDate, projectDateOK := ParseDate(input.ProjectStartDate)
	if strings.TrimSpace(input.ProjectStartDate) != "" && !projectDateOK {
		findings = append(findings, Finding{
			Code: "timeline.project_start_date_unknown", Severity: "attention",
			SourceKind: "project", SourceID: "start_date", Actual: strings.TrimSpace(input.ProjectStartDate),
		})
	}

	var earliest *Date
	var latest *Date
	for _, source := range effectiveSources {
		for _, token := range source.ParseableDates {
			date, ok := ParseDate(token)
			if !ok {
				continue
			}
			if earliest == nil || date.Compare(*earliest) < 0 {
				candidate := date
				earliest = &candidate
			}
			if latest == nil || date.Compare(*latest) > 0 {
				candidate := date
				latest = &candidate
			}
		}
	}
	if projectDateOK && earliest != nil && !projectDate.EqualYMD(*earliest) {
		findings = append(findings, Finding{
			Code: "timeline.project_earliest_bookmark_mismatch", Severity: "attention",
			SourceKind: "timeline_plan", SourceID: "effective",
			Expected: projectDate.Canonical(), Actual: earliest.Canonical(),
		})
	}
	for _, source := range sources {
		if source.ShadowedBy == "" {
			continue
		}
		winner := winnerForPath(effectiveSources, source.Path)
		if winner != nil && !sameStrings(source.ParseableDates, winner.ParseableDates) {
			findings = append(findings, Finding{
				Code: "timeline.bookmark_path_shadow_date_mismatch", Severity: "attention",
				SourceKind: "bookmark_path", SourceID: source.Path, Path: source.Path,
				Expected: strings.Join(winner.ParseableDates, ","), Actual: strings.Join(source.ParseableDates, ","),
			})
		}
	}

	effectiveStart := earliest
	if effectiveStart == nil && projectDateOK {
		candidate := projectDate
		effectiveStart = &candidate
	}
	emitsBookmark := len(effectiveSources) > 0
	customStart := effectiveStart != nil && !effectiveStart.EqualYMD(vanillaStart)
	if projectDateOK && customStart && !emitsBookmark {
		findings = append(findings, Finding{
			Code: "timeline.custom_start_without_bookmark", Severity: "attention",
			SourceKind: "project", SourceID: "start_date", Actual: projectDate.Canonical(),
		})
	}
	if customStart {
		findings = append(findings, Finding{
			Code: "timeline.tension_scale_not_reviewed", Severity: "info",
			SourceKind: "timeline_plan", SourceID: "effective", Actual: effectiveStart.Canonical(),
		})
	}

	effectiveEnd := vanillaEnd
	endDateSource := "vanilla"
	coverageDate := effectiveStart
	if latest != nil {
		coverageDate = latest
	}
	if coverageDate == nil {
		candidate := vanillaStart
		coverageDate = &candidate
	}
	requestedText := strings.TrimSpace(input.RequestedEndDate)
	requestedEnd, requestedEndOK := ParseDate(requestedText)
	requiresEndDate := false
	if requestedText != "" {
		if coverageDate != nil && requestedEndOK && requestedEnd.Compare(*coverageDate) > 0 {
			effectiveEnd = requestedEnd
			endDateSource = "explicit"
			requiresEndDate = true
		} else {
			expected := "a valid date after the campaign start"
			if coverageDate != nil {
				expected = "after " + coverageDate.Canonical()
			}
			findings = append(findings, Finding{
				Code: "timeline.requested_end_date_invalid", Severity: "attention",
				SourceKind: "project", SourceID: "end_date", Expected: expected, Actual: requestedText,
			})
		}
	}
	if endDateSource != "explicit" && coverageDate != nil && compareYMD(*coverageDate, vanillaEnd) >= 0 && !input.HasDependencies {
		requiresEndDate = true
		effectiveEnd = Date{Year: coverageDate.Year + 100, Month: 1, Day: 1, Hour: 12}
		endDateSource = "compatibility_fallback"
		findings = append(findings, Finding{
			Code: "timeline.end_date_defaulted", Severity: "warning",
			SourceKind: "project", SourceID: "end_date", Actual: effectiveEnd.Canonical(),
		})
	} else if endDateSource != "explicit" && input.HasDependencies {
		endDateSource = "unregistered"
		expected := "a verified base-mod campaign end date"
		if coverageDate != nil {
			expected = "a verified base-mod end date after " + coverageDate.Canonical()
		}
		findings = append(findings, Finding{
			Code: "timeline.base_timeline_source_unregistered", Severity: "attention",
			SourceKind: "project", SourceID: "dependencies",
			Expected: expected,
			Actual:   "source_unregistered",
		})
	}

	emitDefines := (customStart || requiresEndDate) && !rawDefinesReplacesGenerated(input.RawDefines, definesPath)
	rawWinner := rawDefinesWinner(input.RawDefines, definesPath, emitDefines)
	defines := DefinesArtifact{Path: definesPath}
	if effectiveStart != nil {
		defines.StartDate = effectiveStart.Canonical()
	}
	if requiresEndDate {
		defines.EndDate = effectiveEnd.Canonical()
		defines.EmitsEndDate = true
	}
	if emitDefines {
		defines.Content = renderDefines(effectiveStart, effectiveEnd, requiresEndDate, endDateSource)
	}
	if rawWinner != nil {
		expectedStart := vanillaStart
		if effectiveStart != nil {
			expectedStart = *effectiveStart
		}
		startPath, startID := rawWinner.StartPath, rawWinner.StartID
		if startPath == "" {
			startPath, startID = rawWinner.Path, rawWinner.ID
		}
		if rawWinner.ParsedStartDate == "" {
			if (customStart && rawWinner.ReplacesGenerated) || rawWinner.StartDate != "" {
				findings = append(findings, Finding{
					Code: "timeline.raw_defines_start_date_conflict", Severity: "attention",
					SourceKind: "defines_raw_file", SourceID: startID, Path: startPath,
					Expected: expectedStart.Canonical(), Actual: rawWinner.StartDate,
				})
			}
		} else {
			rawStart, ok := ParseDate(rawWinner.ParsedStartDate)
			if !ok || !rawStart.EqualYMD(expectedStart) {
				findings = append(findings, Finding{
					Code: "timeline.raw_defines_start_date_conflict", Severity: "attention",
					SourceKind: "defines_raw_file", SourceID: startID, Path: startPath,
					Expected: expectedStart.Canonical(), Actual: rawWinner.ParsedStartDate,
				})
			}
		}

		actualStart := expectedStart
		if parsed, ok := ParseDate(rawWinner.ParsedStartDate); ok {
			actualStart = parsed
		}
		expectedEnd := vanillaEnd
		if requiresEndDate {
			expectedEnd = effectiveEnd
		}
		endPath, endID := rawWinner.EndPath, rawWinner.EndID
		if endPath == "" {
			endPath, endID = rawWinner.Path, rawWinner.ID
		}
		finalEndIsRaw := rawWinner.EndDate != ""
		finalEndMissingForLateStart := endDateSource != "unregistered" && !finalEndIsRaw && !defines.EmitsEndDate &&
			compareYMD(actualStart, vanillaEnd) >= 0
		if finalEndMissingForLateStart || (requiresEndDate && rawWinner.ReplacesGenerated && rawWinner.EndDate == "") {
			findings = append(findings, Finding{
				Code: "timeline.raw_defines_end_date_missing_or_invalid", Severity: "attention",
				SourceKind: "defines_raw_file", SourceID: endID, Path: endPath,
				Expected: "after " + actualStart.Canonical(), Actual: rawWinner.EndDate,
			})
		} else if finalEndIsRaw {
			rawEnd, rawEndOK := ParseDate(rawWinner.ParsedEndDate)
			if !rawEndOK || rawEnd.Compare(actualStart) <= 0 {
				findings = append(findings, Finding{
					Code: "timeline.raw_defines_end_date_missing_or_invalid", Severity: "attention",
					SourceKind: "defines_raw_file", SourceID: endID, Path: endPath,
					Expected: "after " + actualStart.Canonical(), Actual: rawWinner.EndDate,
				})
			} else if endDateSource != "unregistered" && rawEnd.Compare(expectedEnd) != 0 {
				findings = append(findings, Finding{
					Code: "timeline.raw_defines_end_date_conflict", Severity: "attention",
					SourceKind: "defines_raw_file", SourceID: endID, Path: endPath,
					Expected: expectedEnd.Canonical(), Actual: rawEnd.Canonical(),
				})
			}
		}
	}

	replacePaths := effectiveReplacePaths(input.RequestedReplacePaths, input.InheritedReplacePaths)
	if emitsBookmark && customStart {
		replacePaths = appendUniquePath(replacePaths, BookmarksPath)
	}

	sortFindings(findings)
	plan := Plan{
		EmitsBookmark:         emitsBookmark,
		EmitDefines:           emitDefines,
		Defines:               defines,
		EffectiveReplacePaths: append([]string(nil), replacePaths...),
		Findings:              append([]Finding(nil), findings...),
		BookmarkSources:       copyBookmarkSources(sources),
		RawDefinesWinner:      rawWinner,
	}
	if effectiveStart != nil {
		plan.EffectiveStartDate = effectiveStart.Canonical()
	}
	if endDateSource == "unregistered" {
		plan.EndDateSource = endDateSource
	} else if effectiveStart != nil || requiresEndDate {
		plan.EffectiveEndDate = effectiveEnd.Canonical()
		plan.EndDateSource = endDateSource
	}
	plan.PlanSHA256 = planHash(plan)
	return plan
}

func parseOrDefault(value, fallback string) Date {
	if parsed, ok := ParseDate(value); ok {
		return parsed
	}
	parsed, ok := ParseDate(fallback)
	if !ok {
		panic("timeline package default date is invalid")
	}
	return parsed
}

func evidenceFromBookmark(kind string, item BookmarkEvidence) BookmarkSourceEvidence {
	path := normalisePath(item.Path)
	if path == "" && strings.TrimSpace(item.ID) != "" {
		path = BookmarksPath + "/" + strings.TrimSpace(item.ID) + ".txt"
	}
	id := strings.TrimSpace(item.ID)
	if id == "" {
		id = path
	}
	if id == "" {
		id = "<blank-bookmark>"
	}
	tokens := copyStrings(item.DateTokens)
	parseable := make([]string, 0, len(tokens))
	invalid := false
	for _, token := range tokens {
		date, ok := ParseDate(token)
		if !ok {
			invalid = true
			continue
		}
		parseable = append(parseable, date.Canonical())
	}
	parseError := ""
	if len(tokens) == 0 {
		parseError = "bookmark_date_missing"
	} else if invalid {
		parseError = "bookmark_date_invalid"
	}
	return BookmarkSourceEvidence{
		Kind: kind, ID: id, Path: path, Source: strings.TrimSpace(item.Source),
		SourceHash: strings.TrimSpace(item.SourceHash), DateTokens: tokens,
		ParseableDates: parseable, ParseError: parseError,
		Emitted: item.Emitted && isBookmarkFile(path),
	}
}

func rawDefinesReplacesGenerated(raws []RawDefinesEvidence, generatedPath string) bool {
	for _, raw := range raws {
		if raw.Emitted && strings.EqualFold(normalisePath(raw.Path), generatedPath) {
			return true
		}
	}
	return false
}

func rawDefinesWinner(raws []RawDefinesEvidence, generatedPath string, generatedEmitted bool) *RawDefinesWinner {
	candidates := make([]RawDefinesWinner, 0, len(raws))
	for _, raw := range raws {
		path := normalisePath(raw.Path)
		lower := strings.ToLower(path)
		if !raw.Emitted || !strings.HasPrefix(lower, "common/defines/") || !strings.HasSuffix(lower, ".lua") {
			continue
		}
		start, end := strings.TrimSpace(raw.StartDate), strings.TrimSpace(raw.EndDate)
		if raw.Content != "" {
			contentStart, contentEnd := ExtractDefineDates(raw.Content)
			if start == "" {
				start = contentStart
			}
			if end == "" {
				end = contentEnd
			}
		}
		id := strings.TrimSpace(raw.ID)
		if id == "" {
			id = path
		}
		replaces := strings.EqualFold(path, generatedPath)
		loadsAfter := path > generatedPath
		if !replaces && ((generatedEmitted && !loadsAfter) || (start == "" && end == "")) {
			continue
		}
		candidate := RawDefinesWinner{
			ID: id, Path: path, Source: strings.TrimSpace(raw.Source), SourceHash: strings.TrimSpace(raw.SourceHash),
			StartDate: start, EndDate: end, ReplacesGenerated: replaces, LoadsAfterGenerated: loadsAfter,
		}
		if start != "" {
			candidate.StartID, candidate.StartPath = candidate.ID, candidate.Path
			candidate.StartSource, candidate.StartSourceHash = candidate.Source, candidate.SourceHash
		}
		if end != "" {
			candidate.EndID, candidate.EndPath = candidate.ID, candidate.Path
			candidate.EndSource, candidate.EndSourceHash = candidate.Source, candidate.SourceHash
		}
		if parsed, ok := ParseDate(start); ok {
			candidate.ParsedStartDate = parsed.Canonical()
		}
		if parsed, ok := ParseDate(end); ok {
			candidate.ParsedEndDate = parsed.Canonical()
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Path < candidates[j].Path })
	resolved := RawDefinesWinner{}
	for _, candidate := range candidates {
		if candidate.ReplacesGenerated {
			resolved = candidate
			resolved.ReplacesGenerated = true
			continue
		}
		if resolved.ID == "" {
			resolved = candidate
		} else {
			resolved.ID = candidate.ID
			resolved.Path = candidate.Path
			resolved.Source = candidate.Source
			resolved.SourceHash = candidate.SourceHash
			resolved.LoadsAfterGenerated = true
		}
		if candidate.StartDate != "" {
			resolved.StartDate = candidate.StartDate
			resolved.ParsedStartDate = candidate.ParsedStartDate
			resolved.StartID, resolved.StartPath = candidate.ID, candidate.Path
			resolved.StartSource, resolved.StartSourceHash = candidate.Source, candidate.SourceHash
		}
		if candidate.EndDate != "" {
			resolved.EndDate = candidate.EndDate
			resolved.ParsedEndDate = candidate.ParsedEndDate
			resolved.EndID, resolved.EndPath = candidate.ID, candidate.Path
			resolved.EndSource, resolved.EndSourceHash = candidate.Source, candidate.SourceHash
		}
	}
	return &resolved
}

func renderDefines(start *Date, end Date, emitEnd bool, endDateSource string) string {
	lines := make([]string, 0, 2)
	if start != nil {
		lines = append(lines, "NDefines.NGame.START_DATE = \""+start.Canonical()+"\"")
	}
	if emitEnd {
		endValue := end.DefinesValue()
		if endDateSource == "explicit" {
			endValue = end.Canonical()
		}
		lines = append(lines, "NDefines.NGame.END_DATE = \""+endValue+"\"")
	}
	return strings.Join(lines, "\n") + "\n"
}

func effectiveReplacePaths(requested, inherited []string) []string {
	paths := make([]string, 0, len(requested)+len(inherited))
	for _, path := range requested {
		paths = appendUniquePath(paths, path)
	}
	for _, path := range inherited {
		paths = appendUniquePath(paths, path)
	}
	return paths
}

func appendUniquePath(paths []string, path string) []string {
	clean := normalisePath(path)
	if clean == "" {
		return paths
	}
	for _, existing := range paths {
		if strings.EqualFold(existing, clean) {
			return paths
		}
	}
	return append(paths, clean)
}

func normalisePath(path string) string {
	path = strings.TrimLeft(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"), "/")
	parts := make([]string, 0)
	for _, part := range strings.Split(path, "/") {
		switch part {
		case "", ".":
			continue
		case "..":
			return ""
		default:
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "/")
}

func isBookmarkFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasPrefix(lower, BookmarksPath+"/") && strings.HasSuffix(lower, ".txt")
}

func compareYMD(left, right Date) int {
	leftYMD := [...]int{left.Year, left.Month, left.Day}
	rightYMD := [...]int{right.Year, right.Month, right.Day}
	for i := range leftYMD {
		if leftYMD[i] < rightYMD[i] {
			return -1
		}
		if leftYMD[i] > rightYMD[i] {
			return 1
		}
	}
	return 0
}

func winnerForPath(sources []BookmarkSourceEvidence, path string) *BookmarkSourceEvidence {
	for i := range sources {
		if strings.EqualFold(sources[i].Path, path) {
			return &sources[i]
		}
	}
	return nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func copyStrings(values []string) []string {
	return append([]string(nil), values...)
}

func copyBookmarkSources(values []BookmarkSourceEvidence) []BookmarkSourceEvidence {
	cloned := make([]BookmarkSourceEvidence, len(values))
	for i, value := range values {
		cloned[i] = value
		cloned[i].DateTokens = copyStrings(value.DateTokens)
		cloned[i].ParseableDates = copyStrings(value.ParseableDates)
	}
	return cloned
}

func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		left, right := findings[i], findings[j]
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.SourceID != right.SourceID {
			return left.SourceID < right.SourceID
		}
		return left.Actual < right.Actual
	})
}

func planHash(plan Plan) string {
	plan.PlanSHA256 = ""
	payload, err := json.Marshal(plan)
	if err != nil {
		panic("timeline plan serialisation: " + err.Error())
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
