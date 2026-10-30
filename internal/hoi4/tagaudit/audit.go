package tagaudit

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/importjob"
	"haganeworks/internal/hoi4/importsvc"
	importworld "haganeworks/internal/hoi4/importsvc/world"
	"haganeworks/internal/hoi4/loc"
	"haganeworks/internal/hoi4/paradox"
)

type File struct {
	Path string
	Body []byte
}

type Summary struct {
	Declared int `json:"declared"`
	New      int `json:"new"`
	Existing int `json:"existing"`
	Dynamic  int `json:"dynamic"`
	Static   int `json:"static"`
}

type EntityPreview struct {
	Characters int `json:"characters"`
	FocusFiles int `json:"focus_files"`
	Events     int `json:"events"`
	Ideas      int `json:"ideas"`
	Decisions  int `json:"decisions"`
	History    int `json:"history"`
}

type NamePreview struct {
	English *string `json:"en"`
	Chinese *string `json:"zh"`
}

type DetectedTag struct {
	Tag           string        `json:"tag"`
	FilePath      string        `json:"file_path"`
	IsDynamic     bool          `json:"is_dynamic"`
	AlreadyExists bool          `json:"already_exists"`
	EntityPreview EntityPreview `json:"entity_preview"`
	NamePreview   NamePreview   `json:"name_preview"`
	ColorPreview  []int         `json:"color_preview"`
}

type WarningItem struct {
	Code   string         `json:"code"`
	Params map[string]any `json:"params"`
}

type Result struct {
	CountryTagFiles []string      `json:"country_tags_files"`
	Summary         Summary       `json:"summary"`
	Detected        []DetectedTag `json:"detected"`
	Warnings        []string      `json:"warnings"`
	WarningItems    []WarningItem `json:"warning_items"`
	DescriptorName  *string       `json:"descriptor_name"`
}

type decodedFile struct {
	path string
	text string
}

var descriptorNameRE = regexp.MustCompile(`(?m)^\s*name\s*=\s*"([^"]+)"`)

var recognizedRoots = map[string]bool{
	"common": true, "gfx": true, "interface": true, "localisation": true,
	"history": true, "events": true,
}

func NormalizePath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	parts := strings.Split(value, "/")
	if len(parts) > 1 && !recognizedRoots[parts[0]] {
		return strings.Join(parts[1:], "/")
	}
	return value
}

func ParseDescriptorName(text string) *string {
	match := descriptorNameRE.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	name := strings.TrimSpace(match[1])
	return &name
}

func Audit(files []File, existing map[string]bool) Result {
	result := Result{
		CountryTagFiles: []string{}, Detected: []DetectedTag{},
		Warnings: []string{}, WarningItems: []WarningItem{},
	}
	countryTags := map[string]string{}
	countries := map[string]string{}
	countryOrder := []string{}
	localisations := []loc.LocFile{}
	characters, focuses := map[string]bool{}, map[string]bool{}
	events, ideas := map[string]bool{}, map[string]bool{}
	decisions, histories := map[string]bool{}, map[string]bool{}
	unrouted := map[string]int{}
	unsafeRaid, unsafeUnit := []string{}, []string{}
	raids := map[string]string{}
	var descriptorName *string
	descriptorSeen := false

	for _, file := range files {
		norm := NormalizePath(file.Path)
		if !descriptorSeen && strings.HasSuffix(strings.ToLower(file.Path), "descriptor.mod") {
			descriptorSeen = true
			descriptorName = ParseDescriptorName(decodeUTF8Replace(file.Body))
		}
		if strings.HasPrefix(norm, "gfx/") && hasAnySuffix(strings.ToLower(norm), ".png", ".dds", ".tga") {
			continue
		}
		text, ok := gfx.DecodeScriptText(file.Body)
		if !ok {
			errText := cp1252DecodeError(file.Body)
			params := map[string]any{"path": norm, "error": errText}
			appendWarning(&result, "decode_failed", params,
				[]importsvc.WarnParam{{Key: "path", Value: norm}, {Key: "error", Value: errText}},
				fmt.Sprintf("Couldn't read %s — it was skipped; the rest imported fine.", norm))
			continue
		}
		appendRecoveries(&result, decodedFile{path: norm, text: text})

		switch {
		case strings.HasPrefix(norm, "common/characters/") && strings.HasSuffix(norm, ".txt"):
			characters[norm] = true
		case strings.HasPrefix(norm, "common/national_focus/") && strings.HasSuffix(norm, ".txt"):
			focuses[norm] = true
		case strings.HasPrefix(norm, "common/technologies/") && strings.HasSuffix(norm, ".txt"):

		case strings.HasPrefix(norm, "common/ideas/") && strings.HasSuffix(norm, ".txt"):
			ideas[norm] = true
		case strings.HasPrefix(norm, "events/") && strings.HasSuffix(norm, ".txt"):
			if !importworld.IsSuperEventFile(norm, &text) {
				events[norm] = true
			}
		case strings.HasPrefix(norm, "common/decisions/categories/") && strings.HasSuffix(norm, ".txt"):
		case strings.HasPrefix(norm, "common/decisions/") && strings.HasSuffix(norm, ".txt"):
			decisions[norm] = true
		case recognizedCommonTextPath(norm):
		case strings.HasPrefix(norm, "common/raids/") && strings.HasSuffix(norm, ".txt"):
			if !safeRelativePath(norm) {
				unsafeRaid = append(unsafeRaid, norm)
			} else {
				raids[norm] = text
			}
		case strings.HasPrefix(norm, "common/on_actions/") && strings.HasSuffix(norm, ".txt"):
			if !safeRelativePath(norm) {
				unsafeRaid = append(unsafeRaid, norm)
			}
		case strings.HasPrefix(norm, "common/factions/") && strings.HasSuffix(norm, ".txt"):
			if !safeRelativePath(norm) {
				params := map[string]any{"file": norm}
				appendWarning(&result, "factions_path_unsafe", params,
					[]importsvc.WarnParam{{Key: "file", Value: norm}},
					fmt.Sprintf("Skipped faction file '%s' - its path is not a valid mod-relative file path (unsafe path).", norm))
			}
		case (strings.HasPrefix(norm, "common/units/") || strings.HasPrefix(norm, "history/units/")) && strings.HasSuffix(norm, ".txt"):
			if !safeRelativePath(norm) {
				unsafeUnit = append(unsafeUnit, norm)
			}
		case strings.HasPrefix(norm, "common/bookmarks/") && strings.HasSuffix(norm, ".txt"):
		case strings.HasPrefix(norm, "common/country_tags/") && strings.HasSuffix(norm, ".txt"):
			countryTags[norm] = text
		case strings.HasPrefix(norm, "common/countries/") && strings.HasSuffix(norm, ".txt"):
			if _, seen := countries[norm]; !seen {
				countryOrder = append(countryOrder, norm)
			}
			countries[norm] = text
		case importworld.IsHistoryFile(norm):
			histories[norm] = true
		case importworld.IsStateFile(norm):
		case strings.HasPrefix(norm, "interface/") && strings.HasSuffix(norm, ".gfx"):
		case strings.HasPrefix(norm, "localisation/") && strings.HasSuffix(norm, ".yml"):
			localisations = append(localisations, loc.LocFile{Path: norm, Text: text})
		default:
			dir := "(root)"
			if slash := strings.LastIndex(norm, "/"); slash >= 0 {
				dir = norm[:slash]
			}
			unrouted[dir]++
		}
	}

	appendClassificationWarnings(&result, unrouted, unsafeRaid, unsafeUnit, raids)
	result.DescriptorName = descriptorName

	paths := make([]string, 0, len(countryTags))
	for filePath := range countryTags {
		paths = append(paths, filePath)
	}
	sort.Strings(paths)
	result.CountryTagFiles = paths
	declarations := []importjob.CountryTagDeclaration{}
	declared := map[string]bool{}
	for _, filePath := range paths {
		for _, declaration := range importjob.ParseCountryTagDeclarations(countryTags[filePath]) {
			if declared[declaration.Tag] {
				continue
			}
			declared[declaration.Tag] = true
			declarations = append(declarations, declaration)
		}
	}

	counts := make(map[string]EntityPreview, len(declared))
	for tag := range declared {
		counts[tag] = EntityPreview{}
	}
	countPaths(characters, counts, func(p *EntityPreview) { p.Characters++ })
	countPaths(focuses, counts, func(p *EntityPreview) { p.FocusFiles++ })
	countPaths(events, counts, func(p *EntityPreview) { p.Events++ })
	countPaths(ideas, counts, func(p *EntityPreview) { p.Ideas++ })
	countPaths(decisions, counts, func(p *EntityPreview) { p.Decisions++ })
	countPaths(histories, counts, func(p *EntityPreview) { p.History++ })

	sharedColors := map[string][]int{}
	for _, filePath := range countryOrder {
		for tag, color := range importjob.ParseSharedCountryColors(countries[filePath]) {
			if _, found := sharedColors[tag]; !found {
				sharedColors[tag] = color
			}
		}
	}
	locMap := loc.BuildLocMap(localisations)
	for _, declaration := range declarations {
		name := NamePreview{}
		if value := strings.TrimSpace(locMap["en"][declaration.Tag]); value != "" {
			name.English = stringPointer(value)
		}
		if value := strings.TrimSpace(locMap["zh"][declaration.Tag]); value != "" {
			name.Chinese = stringPointer(value)
		}
		color := sharedColors[declaration.Tag]
		countryPath := declaration.FilePath
		if !strings.HasPrefix(countryPath, "common/") {
			countryPath = "common/" + countryPath
		}
		if text := countries[countryPath]; text != "" {
			if parsed, ok := importjob.ParseCountryColor(text); ok {
				color = parsed
			}
		}
		if color != nil {
			color = append([]int(nil), color...)
		}
		result.Detected = append(result.Detected, DetectedTag{
			Tag: declaration.Tag, FilePath: declaration.FilePath,
			IsDynamic: declaration.IsDynamic, AlreadyExists: existing[declaration.Tag],
			EntityPreview: counts[declaration.Tag], NamePreview: name, ColorPreview: color,
		})
		result.Summary.Declared++
		if existing[declaration.Tag] {
			result.Summary.Existing++
		} else {
			result.Summary.New++
		}
		if declaration.IsDynamic {
			result.Summary.Dynamic++
		} else {
			result.Summary.Static++
		}
	}
	return result
}

func stringPointer(value string) *string { return &value }

func countPaths(files map[string]bool, counts map[string]EntityPreview, increment func(*EntityPreview)) {
	for filePath := range files {
		tag := filenamePrefix(filePath)
		preview, ok := counts[tag]
		if !ok {
			continue
		}
		increment(&preview)
		counts[tag] = preview
	}
}

func filenamePrefix(filePath string) string {
	base := filePath
	if slash := strings.LastIndex(base, "/"); slash >= 0 {
		base = base[slash+1:]
	}
	if dot := strings.LastIndex(base, "."); dot >= 0 {
		base = base[:dot]
	}
	if underscore := strings.Index(base, "_"); underscore >= 0 {
		base = base[:underscore]
	}
	base = strings.ToUpper(base)
	if len(base) < 2 || len(base) > 4 {
		return ""
	}
	for _, r := range base {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return ""
		}
	}
	return base
}

func recognizedCommonTextPath(norm string) bool {
	if !strings.HasSuffix(norm, ".txt") {
		return false
	}
	prefixes := []string{
		"common/country_leader/", "common/unit_leader/", "common/scientist_traits/",
		"common/special_projects/specialization/", "common/special_projects/projects/",
		"common/opinion_modifiers/", "common/modifier_definitions/",
		"common/ai_strategy_plans/", "common/ai_templates/",
		"common/dynamic_modifiers/", "common/autonomous_states/",
		"common/military_industrial_organization/organizations/",
		"common/intelligence_agencies/", "common/scripted_effects/",
		"common/scripted_triggers/", "common/scripted_localisation/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(norm, prefix) {
			return true
		}
	}
	if strings.HasPrefix(norm, "common/wargoals/") && strings.Count(norm, "/") == 2 {
		return true
	}
	if strings.HasPrefix(norm, "common/doctrines/") {
		parts := strings.Split(norm, "/")
		return len(parts) >= 4 && map[string]bool{
			"folders": true, "grand_doctrines": true,
			"tracks": true, "subdoctrines": true,
		}[parts[2]]
	}
	return false
}

func safeRelativePath(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "\\") {
		return false
	}
	for _, component := range strings.Split(strings.ReplaceAll(value, "\\", "/"), "/") {
		if component == ".." {
			return false
		}
	}
	return true
}

func hasAnySuffix(value string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}
	return false
}

func decodeUTF8Replace(body []byte) string {
	if len(body) >= 3 && body[0] == 0xef && body[1] == 0xbb && body[2] == 0xbf {
		body = body[3:]
	}
	if utf8.Valid(body) {
		return string(body)
	}
	return strings.ToValidUTF8(string(body), "�")
}

func cp1252DecodeError(body []byte) string {
	for index, value := range body {
		switch value {
		case 0x81, 0x8d, 0x8f, 0x90, 0x9d:
			return fmt.Sprintf("'charmap' codec can't decode byte 0x%02x in position %d: character maps to <undefined>", value, index)
		}
	}
	return "unable to decode file as UTF-8 or cp1252"
}

func appendRecoveries(result *Result, file decodedFile) {
	lower := strings.ToLower(file.path)
	if !hasAnySuffix(lower, ".txt", ".gui", ".gfx", ".asset", ".mod") {
		return
	}
	_, recoveries, err := paradox.Parse(file.text)
	if err != nil {
		return
	}
	for _, recovery := range recoveries {
		params := map[string]any{"path": file.path, "line": recovery.Line}
		ordered := []importsvc.WarnParam{{Key: "path", Value: file.path}, {Key: "line", Value: recovery.Line}}
		for _, key := range []string{"key", "terminator", "eof_line"} {
			if value, ok := recovery.Details[key]; ok {
				params[key] = value
				ordered = append(ordered, importsvc.WarnParam{Key: key, Value: value})
			}
		}
		location := fmt.Sprintf("%s:%d", file.path, recovery.Line)
		fallback := fmt.Sprintf("Recovered malformed script at %s.", location)
		switch recovery.Code {
		case paradox.RecoveryStrayRBrace:
			fallback = fmt.Sprintf("Recovered malformed script at %s: ignored an unmatched top-level closing brace and continued. Review this source line before export.", location)
		case paradox.RecoveryEmptyValue:
			fallback = fmt.Sprintf("Recovered malformed script at %s: '%v' has no value before '%v', so it was preserved with an empty value. Review this source line before export.", location, recovery.Details["key"], recovery.Details["terminator"])
		}
		appendWarning(result, recovery.Code, params, ordered, fallback)
	}
}

func appendClassificationWarnings(result *Result, unrouted map[string]int, unsafeRaid, unsafeUnit []string, raids map[string]string) {
	directories := make([]string, 0, len(unrouted))
	for directory := range unrouted {
		directories = append(directories, directory)
	}
	sort.Strings(directories)
	for _, directory := range directories {
		count := unrouted[directory]
		params := map[string]any{"count": count, "dir": directory}
		appendWarning(result, "unrouted_dir", params,
			[]importsvc.WarnParam{{Key: "count", Value: count}, {Key: "dir", Value: directory}},
			fmt.Sprintf("%d file(s) under '%s/' aren't edited here — they stay in your original mod, untouched", count, directory))
	}
	sort.Strings(unsafeRaid)
	for _, filePath := range unsafeRaid {
		params := map[string]any{"file": filePath}
		appendWarning(result, "raid_path_unsafe", params,
			[]importsvc.WarnParam{{Key: "file", Value: filePath}},
			fmt.Sprintf("Skipped the raid file '%s' — its path isn't a valid mod-relative file path, so it can't be preserved", filePath))
	}
	sort.Strings(unsafeUnit)
	for _, filePath := range unsafeUnit {
		params := map[string]any{"file": filePath}
		appendWarning(result, "unit_path_unsafe", params,
			[]importsvc.WarnParam{{Key: "file", Value: filePath}},
			fmt.Sprintf("Skipped the unit file '%s' — its path isn't a valid mod-relative file path, so it can't be preserved", filePath))
	}
	appendRaidDependencyWarning(result, raids)
}

var (
	raidLocKeyRE     = regexp.MustCompile(`\btarget_loc_key\s*=\s*([A-Za-z_][A-Za-z0-9_.]*)`)
	raidSpriteRE     = regexp.MustCompile(`=\s*(GFX_[A-Za-z_0-9]+)`)
	raidTypesBlockRE = regexp.MustCompile(`(?m)^types\s*=\s*\{`)
	raidTypeNameRE   = regexp.MustCompile(`(?m)^\t([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\{`)
)

func appendRaidDependencyWarning(result *Result, raids map[string]string) {
	if len(raids) == 0 {
		return
	}
	locKeys, sprites := map[string]bool{}, map[string]bool{}
	for _, text := range raids {
		cleaned := gfx.StripCommentsAndSkipQuotes(text)
		for _, match := range raidLocKeyRE.FindAllStringSubmatch(cleaned, -1) {
			locKeys[match[1]] = true
		}
		for _, match := range raidSpriteRE.FindAllStringSubmatch(cleaned, -1) {
			sprites[match[1]] = true
		}
		if raidTypesBlockRE.MatchString(cleaned) {
			matches := raidTypeNameRE.FindAllStringSubmatchIndex(cleaned, -1)
			for i, match := range matches {
				end := len(cleaned)
				if i+1 < len(matches) {
					end = matches[i+1][0]
				}
				if !strings.Contains(cleaned[match[1]:end], "custom_map_icon") {
					sprites["GFX_raid_type_icon_"+cleaned[match[2]:match[3]]] = true
				}
			}
		}
	}
	if len(locKeys) == 0 && len(sprites) == 0 {
		return
	}
	locList, spriteList := sortedKeys(locKeys), sortedKeys(sprites)
	samples := append(firstN(locList, 3), firstN(spriteList, 3)...)
	sample := strings.Join(samples, ", ")
	params := map[string]any{"loc_count": len(locKeys), "sprite_count": len(sprites), "sample": sample}
	appendWarning(result, "raid_deps_not_carried", params,
		[]importsvc.WarnParam{{Key: "loc_count", Value: len(locKeys)}, {Key: "sprite_count", Value: len(sprites)}, {Key: "sample", Value: sample}},
		fmt.Sprintf("Your raid files are preserved, but the %d text key(s) and %d sprite(s) they reference are not — only vanilla ones will resolve in game (e.g. %s). Add the mod's localisation/ and interface/ definitions as raw override files if the raids are custom.", len(locKeys), len(sprites), sample))
}

func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func firstN(values []string, limit int) []string {
	if len(values) <= limit {
		return append([]string(nil), values...)
	}
	return append([]string(nil), values[:limit]...)
}

func appendWarning(result *Result, code string, params map[string]any, ordered []importsvc.WarnParam, fallback string) {
	result.Warnings = append(result.Warnings, importsvc.CodedWarning(code, ordered, fallback))
	result.WarningItems = append(result.WarningItems, WarningItem{Code: code, Params: params})
}
