package importjob

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

type CountryStageOptions struct{ ImportCountries, ImportFactions bool }
type CountryMergeOptions struct{ ConflictPolicy string }

func decodeFactionContent(body []byte) string {
	return decodeScriptBytes(body)
}

type CountryCounts struct {
	Countries, Factions, DynamicSkipped int64
	Warnings                            []string
}
type countryTagDecl struct {
	Tag, Path, RawPath string
	Dynamic            bool
}

type CountryTagDeclaration struct {
	Tag       string
	FilePath  string
	IsDynamic bool
}

var countryTagLineRE = regexp.MustCompile(`(?i)^\s*([a-z0-9]{2,5})\s*=\s*"([^"]+)"\s*(?:#.*)?$`)
var dynamicTagRE = regexp.MustCompile(`(?i)^\s*dynamic_tags\s*=\s*yes\b`)
var rgbRE = regexp.MustCompile(`(?i)\bcolor\s*=\s*(?:rgb\s*)?\{\s*(\d{1,3})\s+(\d{1,3})\s+(\d{1,3})\s*\}`)
var rgbUIRE = regexp.MustCompile(`(?i)\bcolor_ui\s*=\s*(?:rgb\s*)?\{\s*(\d{1,3})\s+(\d{1,3})\s+(\d{1,3})\s*\}`)
var cultureRE = regexp.MustCompile(`(?im)^\s*graphical_culture\b\s*=\s*([A-Za-z0-9_]+)`)
var culture2DRE = regexp.MustCompile(`(?im)^\s*graphical_culture_2d\b\s*=\s*([A-Za-z0-9_]+)`)
var legacyAIRE = regexp.MustCompile(`(?im)^\s*use_legacy_ai_pp_spend\b\s*=\s*(yes|no)\b`)

func parseCountryTags(text string) []countryTagDecl {
	normalized := strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\v', '\f', '\x1c', '\x1d', '\x1e', '\x85', '\u2028', '\u2029':
			return '\n'
		}
		return r
	}, strings.TrimPrefix(text, "\ufeff"))
	s := bufio.NewScanner(strings.NewReader(normalized))
	dynamic := false
	seen := map[string]bool{}
	var out []countryTagDecl
	for s.Scan() {
		line := s.Text()
		if dynamicTagRE.MatchString(line) {
			dynamic = true
			continue
		}
		m := countryTagLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		tag := strings.ToUpper(m[1])
		if seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, countryTagDecl{Tag: tag, Path: normalizeCountryPath(m[2]), RawPath: m[2], Dynamic: dynamic})
	}
	return out
}

func ParseCountryTagDeclarations(text string) []CountryTagDeclaration {
	parsed := parseCountryTags(text)
	out := make([]CountryTagDeclaration, 0, len(parsed))
	for _, row := range parsed {
		out = append(out, CountryTagDeclaration{
			Tag: row.Tag, FilePath: row.RawPath, IsDynamic: row.Dynamic,
		})
	}
	return out
}
func normalizeCountryPath(p string) string {
	p = strings.Trim(strings.ReplaceAll(p, "\\", "/"), " /")
	p = strings.TrimPrefix(strings.ToLower(p), "common/")
	return p
}
func rgbValue(text string) ([]int, bool) {
	m := rgbUIRE.FindStringSubmatch(text)
	if m == nil {
		m = rgbRE.FindStringSubmatch(text)
	}
	if m == nil {
		return nil, false
	}
	v := make([]int, 3)
	for i := range v {
		v[i], _ = strconv.Atoi(m[i+1])
		if v[i] < 0 || v[i] > 255 {
			return nil, false
		}
	}
	return v, true
}

func ParseCountryColor(text string) ([]int, bool) { return rgbValue(text) }
func countryMetadata(text string) map[string]any {
	m := map[string]any{}
	if v, ok := rgbValue(text); ok {
		m["color"] = v
	}
	if x := cultureRE.FindStringSubmatch(text); x != nil {
		m["graphical_culture"] = x[1]
	}
	if x := culture2DRE.FindStringSubmatch(text); x != nil {
		m["graphical_culture_2d"] = x[1]
	}
	if x := legacyAIRE.FindStringSubmatch(text); x != nil {
		m["use_legacy_ai_pp_spend"] = strings.EqualFold(x[1], "yes")
	}
	return m
}

var sharedCountryOpenRE = regexp.MustCompile(`(?m)^([A-Za-z0-9]{2,5})\s*=\s*\{`)

func sharedCountryMetadata(text string) map[string]map[string]any {
	out := map[string]map[string]any{}
	pos := 0
	for {
		loc := sharedCountryOpenRE.FindStringSubmatchIndex(text[pos:])
		if loc == nil {
			break
		}
		start := pos + loc[0]
		tag := strings.ToUpper(text[pos+loc[2] : pos+loc[3]])
		brace := strings.IndexByte(text[start:pos+loc[1]], '{') + start
		if brace < start {
			break
		}
		depth, end := 0, -1
		for i := brace; i < len(text); i++ {
			if text[i] == '{' {
				depth++
			}
			if text[i] == '}' {
				depth--
				if depth == 0 {
					end = i + 1
					break
				}
			}
		}
		if end < 0 {
			break
		}
		m := countryMetadata(text[brace+1 : end-1])
		if len(m) > 0 {
			if _, exists := out[tag]; !exists {
				out[tag] = m
			}
		}
		pos = end
	}
	return out
}

func ParseSharedCountryColors(text string) map[string][]int {
	metadata := sharedCountryMetadata(text)
	out := make(map[string][]int, len(metadata))
	for tag, fields := range metadata {
		if color, ok := fields["color"].([]int); ok {
			out[tag] = append([]int(nil), color...)
		}
	}
	return out
}

func safeModPath(p string) bool {
	p = strings.ReplaceAll(p, "\\", "/")
	return p != "" && !strings.HasPrefix(p, "/") && !strings.Contains(p, "../") && !strings.Contains(p, ":")
}

type countryStage struct {
	rows     []countryStageRow
	factions []countryFactionRow
}

type countryStageRow struct {
	tag, filePath string
	payload       payload
}

type countryFactionRow struct{ destPath, content string }

func (s *session) stageCountries(opt CountryStageOptions) (CountryCounts, error) {
	if opt == (CountryStageOptions{}) {
		opt = CountryStageOptions{true, true}
	}
	var counts CountryCounts
	s.countries = countryStage{}
	root := s.root
	var tagFiles []string
	_ = filepath.WalkDir(filepath.Join(root, "common", "country_tags"), func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".txt") {
			tagFiles = append(tagFiles, p)
		}
		return nil
	})
	sort.Strings(tagFiles)
	decls := map[string]countryTagDecl{}
	ordered := []string{}
	if opt.ImportCountries {
		for _, p := range tagFiles {
			b, e := os.ReadFile(p)
			if e != nil {
				return counts, e
			}
			for _, d := range parseCountryTags(string(b)) {
				if _, ok := decls[d.Tag]; ok {
					continue
				}
				decls[d.Tag] = d
				ordered = append(ordered, d.Tag)
			}
		}
	}
	metadata := map[string]map[string]any{}
	reverse := map[string][]string{}
	for _, tag := range ordered {
		d := decls[tag]
		if !d.Dynamic {
			reverse[d.Path] = append(reverse[d.Path], tag)
		}
	}

	_ = filepath.WalkDir(filepath.Join(root, "common", "countries"), func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".txt") {
			return nil
		}
		rel, _ := filepath.Rel(filepath.Join(root, "common"), p)
		key := normalizeCountryPath(filepath.ToSlash(rel))
		tags := reverse[key]
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		if len(tags) == 0 {
			for tag, m := range sharedCountryMetadata(string(b)) {
				if metadata[tag] == nil {
					metadata[tag] = m
				}
			}
			return nil
		}
		m := countryMetadata(string(b))
		for _, tag := range tags {
			metadata[tag] = m
		}
		return nil
	})
	for _, tag := range ordered {
		d := decls[tag]
		if d.Dynamic {
			counts.DynamicSkipped++
			continue
		}
		m := metadata[tag]
		if m == nil {
			m = map[string]any{}
		}
		b, _ := json.Marshal(m)
		s.countries.rows = append(s.countries.rows, countryStageRow{tag: tag, filePath: d.Path, payload: decodePayload(b)})
		counts.Countries++
	}
	if opt.ImportFactions {
		err := filepath.WalkDir(filepath.Join(root, "common", "factions"), func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".txt") {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			dest := filepath.ToSlash(rel)
			if !safeModPath(dest) {
				counts.Warnings = append(counts.Warnings, "factions_path_unsafe: "+dest)
				return nil
			}
			b, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			s.countries.factions = append(s.countries.factions, countryFactionRow{destPath: dest, content: decodeFactionContent(b)})
			counts.Factions++
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return counts, err
		}
	}
	return counts, nil
}

var countryIdeologies = []string{"democratic", "fascism", "communism", "neutrality"}

var countryUpdateColumns = []string{
	"name", "color", "graphical_culture", "graphical_culture_2d", "use_legacy_ai_pp_spend",
	"name_en", "name_zh", "adjective_en", "adjective_zh",
	"display_name_locs", "adjective_locs", "ideology_name_locs",
	"name_democratic_en", "name_democratic_zh", "name_fascism_en", "name_fascism_zh",
	"name_communism_en", "name_communism_zh", "name_neutrality_en", "name_neutrality_zh",
}

const countryFactionNote = "preserved on whole-mod import (common/factions)"

func (s *session) countryLocOrNull(key, lang string) any {
	if text, ok := s.locText(key, lang); ok {
		return text
	}
	return nil
}

func (s *session) mergeCountries(tx *project.Tx, opt CountryMergeOptions) error {
	skip := strings.EqualFold(opt.ConflictPolicy, "skip")

	countries := newKeyedTable(tx, "countries", "tag")
	for _, c := range s.countries.rows {
		p := c.payload
		name := c.tag
		if text, ok := s.locText(c.tag, "en"); ok {
			name = text
		} else if text, ok := s.locText(c.tag, "zh"); ok {
			name = text
		}
		nameEN := c.tag
		if text, ok := s.locText(c.tag, "en"); ok {
			nameEN = text
		}

		var legacyAI any
		if text, ok := p.text("use_legacy_ai_pp_spend"); ok && text != "" {
			legacyAI = p.boolOr("use_legacy_ai_pp_spend", false)
		}
		adjective := c.tag + "_ADJ"
		ideologyNames := map[string]any{}
		row := project.Row{
			"tag": c.tag, "name": name,
			"color":                  p.jsonOr("color", []any{json.Number("128"), json.Number("128"), json.Number("128")}),
			"color_ui":               p.jsonOr("color_ui", []any{}),
			"ideology":               "neutrality",
			"graphical_culture":      p.str("graphical_culture"),
			"graphical_culture_2d":   p.str("graphical_culture_2d"),
			"use_legacy_ai_pp_spend": legacyAI,
			"name_en":                nameEN,
			"name_zh":                s.locStr(c.tag, "zh"),
			"adjective_en":           s.locStr(adjective, "en"),
			"adjective_zh":           s.locStr(adjective, "zh"),
			"display_name_locs":      s.locNamesOrEmpty(c.tag),
			"adjective_locs":         s.locNamesOrEmpty(adjective),
			"ideology_name_locs":     ideologyNames,
			"flag_image_path":        "",
			"flag_paths_by_ideology": map[string]any{},
			"has_obs":                false,
			"auto_added":             true,
		}
		for _, ideology := range countryIdeologies {
			key := c.tag + "_" + ideology
			ideologyNames[ideology] = s.locNamesOrEmpty(key)
			row["name_"+ideology+"_en"] = s.countryLocOrNull(key, "en")
			row["name_"+ideology+"_zh"] = s.countryLocOrNull(key, "zh")
		}
		if skip {
			countries.insertNew(row)
		} else {
			countries.upsert(row, countryUpdateColumns)
		}
	}

	raw := newKeyedTable(tx, "project_raw_files", "dest_path")
	for _, f := range s.countries.factions {
		if id, ok := raw.find(f.destPath); ok {
			if skip || tx.Find("project_raw_files", id)["source"] != "imported" {
				continue
			}
			if _, err := tx.Update("project_raw_files", id, func(row project.Row) {
				row["content"], row["note"], row["is_binary"], row["source"] = f.content, countryFactionNote, false, "imported"
			}); err != nil {
				return fmt.Errorf("merge factions: %w", err)
			}
			continue
		}
		raw.insertNew(project.Row{
			"dest_path": f.destPath, "content": f.content, "generate_replace_path": false,
			"note": countryFactionNote, "is_binary": false, "source": "imported",
		})
	}

	all := append([]project.Row(nil), tx.Rows("countries")...)
	sort.SliceStable(all, func(i, j int) bool { return project.Int64(all[i]["id"]) < project.Int64(all[j]["id"]) })
	tags := make([]any, 0, len(all))
	for _, row := range all {
		tags = append(tags, row["tag"])
	}
	if len(tags) > 0 {
		tx.SetRecord(func(record project.Row) {
			current := record["country_tag"]
			keep := false
			for _, tag := range tags {
				if current != nil && tag == current {
					keep = true
					break
				}
			}
			record["country_tags"] = tags
			if !keep {
				record["country_tag"] = tags[0]
			}
			record["updated_at"] = schema.Now("projects", "updated_at", time.Now())
		})
	}

	if current := tx.Record()["country_tag"]; current != nil {
		for _, row := range all {
			if row["tag"] == current {
				tx.SetRecord(func(record project.Row) { record["country_name"] = row["name"] })
				break
			}
		}
	}
	return nil
}
