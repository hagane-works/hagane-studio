package exportcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"haganeworks/internal/hoi4/paradox"
	"sort"
	"strconv"
	"strings"
)

const vanillaStartDate = "1936.1.1.12"

type hoi4Date [4]int

func ParseHOI4Date(value string) *hoi4Date {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 && len(parts) != 4 {
		return nil
	}
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		nums[i] = n
	}
	y, m, d := nums[0], nums[1], nums[2]
	h := 12
	if len(nums) == 4 {
		h = nums[3]
	}
	if y <= 0 || m < 1 || m > 12 || d < 1 || d > 31 {
		return nil
	}
	return &hoi4Date{y, m, d, h}
}

func dateLess(a, b hoi4Date) bool {
	for i := 0; i < 4; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func ymdEqual(a, b hoi4Date) bool { return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] }

func normaliseDateValue(v hoi4Date) string {
	return strconv.Itoa(v[0]) + "." + strconv.Itoa(v[1]) + "." + strconv.Itoa(v[2]) + "." + strconv.Itoa(v[3])
}

func planParseDate(token string) *hoi4Date {
	v := ParseHOI4Date(token)
	if v == nil || v[3] < 0 || v[3] > 23 {
		return nil
	}
	return v
}

func IsDeliberateVanillaShadow(content string) bool {
	text := strings.TrimLeft(content, "\uFEFF")
	for _, line := range strings.Split(text, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		if strings.TrimSpace(line) != "" {
			return false
		}
	}
	return true
}

type timelineFindingSource struct {
	Kind       string
	Identifier string
	Path       string
	Provider   string
}

type TimelineFinding struct {
	Code        string
	Severity    string
	Source      timelineFindingSource
	Evidence    map[string]any
	RepairRoute string
}

func planNormalisePath(raw string) string {
	path := strings.TrimLeft(strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/"), "/")
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p == "" || p == "." {
			continue
		}
		if p == ".." {
			return ""
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "/")
}

func planSemanticHash(payload map[string]string) string {
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		vb, _ := json.Marshal(payload[k])
		b.Write(kb)
		b.WriteByte(':')
		b.Write(vb)
	}
	b.WriteByte('}')
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func ExtractRawBookmarkDateTokens(content string) ([]string, string) {
	block, recoveries, err := paradox.Parse(content)
	if err != nil || block == nil {
		return nil, "bookmark_parse_error:ParseError"
	}
	for _, rec := range recoveries {
		if rec.Code == paradox.RecoveryUnclosedBlock {
			return nil, "bookmark_parse_error:ParseError"
		}
	}
	var tokens []string
	bookmarkCount := 0
	for _, wrapper := range block.Entries {
		if wrapper.Key == nil || *wrapper.Key != "bookmarks" || wrapper.Block == nil {
			continue
		}
		for _, bm := range wrapper.Block.Entries {
			if bm.Key == nil || *bm.Key != "bookmark" || bm.Block == nil {
				continue
			}
			bookmarkCount++
			for _, e := range bm.Block.Entries {
				if e.Key == nil || *e.Key != "date" {
					continue
				}
				if e.Block != nil {
					tokens = append(tokens, "<block>")
				} else {
					tokens = append(tokens, strings.Trim(strings.TrimSpace(e.Scalar.StrValue()), `"`))
				}
			}
		}
	}
	if bookmarkCount == 0 {
		return nil, "bookmark_block_missing"
	}
	if len(tokens) == 0 {
		return nil, "bookmark_date_missing"
	}
	return tokens, ""
}

type StructuredBookmarkInput struct {
	BookmarkID string
	Date       string
	Path       string
	Source     string
	Emitted    bool
}

type PreservedRawBookmarkInput struct {
	Path    string
	Content string
	Source  string
	Emitted bool

	SourceHash string
}

type timelineSourceEvidence struct {
	SourceKind              string
	Provider                string
	SourceID                string
	Path                    string
	SourceHash              string
	DateTokens              []string
	ParseableDates          []string
	ParseError              string
	Emitted                 bool
	Winner                  bool
	ShadowedBy              string
	DeliberateVanillaShadow bool
}

type TimelinePlan struct {
	ProjectStartDate    string
	ProjectParsedDate   string
	EffectiveSources    []timelineSourceEvidence
	EarliestEmittedDate string
	EmitsBookmark       bool
	ShouldEmitDefines   bool

	DefinesStartDate         string
	NeedsBookmarkReplacePath bool
	Findings                 []TimelineFinding
}

func structuredEvidence(item StructuredBookmarkInput, defaultDate string) timelineSourceEvidence {
	bookmarkID := strings.TrimSpace(item.BookmarkID)
	path := planNormalisePath(item.Path)
	if path == "" && bookmarkID != "" {
		path = "common/bookmarks/" + bookmarkID + ".txt"
	}
	token := strings.TrimSpace(item.Date)
	if token == "" {
		token = defaultDate
	}
	parsed := planParseDate(token)
	sourceHash := planSemanticHash(map[string]string{
		"bookmark_id": bookmarkID, "date": token, "path": path, "source": item.Source,
	})
	sourceID := bookmarkID
	if sourceID == "" {
		sourceID = path
	}
	if sourceID == "" {
		sourceID = "<blank-structured-bookmark>"
	}
	ev := timelineSourceEvidence{
		SourceKind: "structured", Provider: item.Source, SourceID: sourceID,
		Path: path, SourceHash: sourceHash, DateTokens: []string{token},
		Emitted: item.Emitted && bookmarkID != "" && path != "",
	}
	if parsed != nil {
		ev.ParseableDates = []string{normaliseDateValue(*parsed)}
	} else {
		ev.ParseError = "bookmark_date_invalid"
	}
	return ev
}

func rawEvidence(item PreservedRawBookmarkInput) (bool, string, timelineSourceEvidence) {
	path := planNormalisePath(item.Path)
	sum := sha256.Sum256([]byte(item.Content))
	tokens, parseError := ExtractRawBookmarkDateTokens(item.Content)
	var parsed []string
	invalid := false
	for _, tok := range tokens {
		if v := planParseDate(tok); v != nil {
			parsed = append(parsed, normaliseDateValue(*v))
		} else {
			invalid = true
		}
	}
	if invalid && parseError == "" {
		parseError = "bookmark_date_invalid"
	}
	sourceID := path
	if sourceID == "" {
		sourceID = "<unsafe-raw-bookmark-path>"
	}
	exactHash := hex.EncodeToString(sum[:])
	declared := strings.ToLower(strings.TrimSpace(item.SourceHash))
	hashMatches := declared == "" || declared == exactHash
	return hashMatches, declared, timelineSourceEvidence{
		SourceKind: "preserved_raw", Provider: item.Source, SourceID: sourceID,
		Path: path, SourceHash: exactHash, DateTokens: tokens,
		ParseableDates: parsed, ParseError: parseError,
		Emitted:                 item.Emitted && strings.HasPrefix(strings.ToLower(path), "common/bookmarks/"),
		DeliberateVanillaShadow: IsDeliberateVanillaShadow(item.Content),
	}
}

func strSlice(v []string) []any {
	out := make([]any, len(v))
	for i, s := range v {
		out[i] = s
	}
	return out
}

func BuildTimelineExportPlan(projectStartDate string,
	structured []StructuredBookmarkInput, raws []PreservedRawBookmarkInput) TimelinePlan {

	evidence := []timelineSourceEvidence{}
	for _, item := range structured {
		evidence = append(evidence, structuredEvidence(item, vanillaStartDate))
	}
	rawHashMatches := map[string]bool{}
	rawDeclaredHashes := map[string]string{}
	for _, item := range raws {
		matches, declared, ev := rawEvidence(item)

		rawHashMatches[ev.SourceID] = matches
		rawDeclaredHashes[ev.SourceID] = declared
		evidence = append(evidence, ev)
	}

	winnerByPath := map[string]int{}
	for i, src := range evidence {
		if src.Emitted {
			winnerByPath[strings.ToLower(src.Path)] = i
		}
	}
	for i := range evidence {
		if !evidence[i].Emitted {
			continue
		}
		w := winnerByPath[strings.ToLower(evidence[i].Path)]
		if i == w {
			evidence[i].Winner = true
		} else {
			evidence[i].ShadowedBy = evidence[w].SourceID
		}
	}

	var effective []timelineSourceEvidence
	for _, src := range evidence {
		if src.Emitted && src.Winner {
			effective = append(effective, src)
		}
	}
	var parsedValues []hoi4Date
	for _, src := range effective {
		for _, tok := range src.ParseableDates {
			if v := planParseDate(tok); v != nil {
				parsedValues = append(parsedValues, *v)
			}
		}
	}
	var earliest *hoi4Date
	for i := range parsedValues {
		if earliest == nil || dateLess(parsedValues[i], *earliest) {
			e := parsedValues[i]
			earliest = &e
		}
	}
	projectValue := planParseDate(projectStartDate)
	vanillaValue := planParseDate(vanillaStartDate)

	var findings []TimelineFinding
	for _, src := range effective {
		matches, tracked := rawHashMatches[src.SourceID]
		if src.SourceKind == "preserved_raw" && tracked && !matches {
			findings = append(findings, TimelineFinding{
				Code: "timeline.raw_bookmark_hash_mismatch", Severity: "blocker",
				Source: timelineFindingSource{Kind: "bookmark_raw_file",
					Identifier: src.SourceID, Path: src.Path, Provider: src.Provider},
				Evidence: map[string]any{
					"declared_hash": rawDeclaredHashes[src.SourceID],
					"actual_hash":   src.SourceHash,
				},
				RepairRoute: "bookmark.raw_file.refresh_source_hash",
			})
		}
		if (src.ParseError != "" || len(src.ParseableDates) == 0) && !src.DeliberateVanillaShadow {
			code := "timeline.structured_bookmark_date_unknown"
			kind := "bookmark"
			repair := "bookmark.date.repair"
			if src.SourceKind == "preserved_raw" {
				code = "timeline.raw_bookmark_date_unknown"
				kind = "bookmark_raw_file"
				repair = "bookmark.raw_file.repair_date"
			}
			parseErr := src.ParseError
			if parseErr == "" {
				parseErr = "no_parseable_date"
			}
			findings = append(findings, TimelineFinding{
				Code: code, Severity: "blocker",
				Source: timelineFindingSource{Kind: kind, Identifier: src.SourceID,
					Path: src.Path, Provider: src.Provider},
				Evidence: map[string]any{
					"date_tokens": strSlice(src.DateTokens),
					"parse_error": parseErr,
					"source_hash": src.SourceHash,
				},
				RepairRoute: repair,
			})
		}
	}
	projectParsed := ""
	if projectValue != nil {
		projectParsed = normaliseDateValue(*projectValue)
	}
	if strings.TrimSpace(projectStartDate) != "" && projectValue == nil {
		findings = append(findings, TimelineFinding{
			Code: "timeline.project_start_date_unknown", Severity: "warning",
			Source:      timelineFindingSource{Kind: "project", Identifier: "start_date"},
			Evidence:    map[string]any{"project_start_date": projectStartDate},
			RepairRoute: "project.start_date.repair",
		})
	}
	earliestStr := ""
	if earliest != nil {
		earliestStr = normaliseDateValue(*earliest)
	}
	if projectValue != nil && earliest != nil && !ymdEqual(*projectValue, *earliest) {
		paths := make([]any, 0, len(effective))
		for _, src := range effective {
			paths = append(paths, src.Path)
		}
		findings = append(findings, TimelineFinding{
			Code: "timeline.project_earliest_bookmark_mismatch", Severity: "warning",
			Source: timelineFindingSource{Kind: "timeline_plan", Identifier: "effective"},
			Evidence: map[string]any{
				"project_start_date":    projectParsed,
				"earliest_emitted_date": earliestStr,
				"source_paths":          paths,
			},
			RepairRoute: "timeline.align_project_to_earliest_bookmark",
		})
	}
	for _, src := range evidence {
		if src.ShadowedBy == "" {
			continue
		}
		var winner *timelineSourceEvidence
		for i := range effective {
			if effective[i].SourceID == src.ShadowedBy {
				winner = &effective[i]
				break
			}
		}
		if winner == nil {
			continue
		}
		if strings.Join(src.ParseableDates, "\x00") != strings.Join(winner.ParseableDates, "\x00") {
			findings = append(findings, TimelineFinding{
				Code: "timeline.bookmark_path_shadow_date_mismatch", Severity: "warning",
				Source: timelineFindingSource{Kind: "bookmark_path", Identifier: src.Path,
					Path: src.Path, Provider: winner.Provider},
				Evidence: map[string]any{
					"shadowed_source": src.SourceID,
					"shadowed_dates":  strSlice(src.ParseableDates),
					"winning_source":  winner.SourceID,
					"winning_dates":   strSlice(winner.ParseableDates),
				},
				RepairRoute: "bookmark.path_collision.review_winner",
			})
		}
	}

	emitsBookmark := len(effective) > 0
	effectiveDefines := earliest
	if effectiveDefines == nil {
		effectiveDefines = projectValue
	}
	customEffectiveStart := effectiveDefines != nil && vanillaValue != nil &&
		!ymdEqual(*effectiveDefines, *vanillaValue)
	definesStartDate := ""
	if effectiveDefines != nil {
		definesStartDate = normaliseDateValue(*effectiveDefines)
	}
	differsFromVanilla := func(v *hoi4Date) bool {
		return v != nil && vanillaValue != nil && !ymdEqual(*v, *vanillaValue)
	}
	needsReplace := emitsBookmark &&
		(differsFromVanilla(earliest) ||
			(earliest == nil && differsFromVanilla(projectValue)))

	if projectValue != nil && customEffectiveStart && !emitsBookmark {
		findings = append(findings, TimelineFinding{
			Code: "timeline.custom_start_without_bookmark", Severity: "warning",
			Source:      timelineFindingSource{Kind: "project", Identifier: "start_date"},
			Evidence:    map[string]any{"project_start_date": projectParsed},
			RepairRoute: "bookmarks.create_for_project_start",
		})
	}

	return TimelinePlan{
		ProjectStartDate:    projectStartDate,
		ProjectParsedDate:   projectParsed,
		EffectiveSources:    effective,
		EarliestEmittedDate: earliestStr,
		EmitsBookmark:       emitsBookmark,
		ShouldEmitDefines:   customEffectiveStart,
		DefinesStartDate:    definesStartDate,

		NeedsBookmarkReplacePath: needsReplace,
		Findings:                 findings,
	}
}

func (p TimelinePlan) DescriptorReplacePaths(existing []string) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, item := range existing {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	if p.NeedsBookmarkReplacePath && !seen["common/bookmarks"] {
		result = append(result, "common/bookmarks")
	}
	return result
}

func (f TimelineFinding) ToMap() map[string]any {
	evidence := f.Evidence
	if evidence == nil {
		evidence = map[string]any{}
	}
	return map[string]any{
		"code": f.Code, "severity": f.Severity,
		"source": findingSourceMap(f.Source.Kind, f.Source.Identifier,
			f.Source.Path, f.Source.Provider),
		"evidence": evidence, "repair_route": f.RepairRoute,
	}
}
