package exportengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

const LocFallbackLogName = "_LOC_EN_FALLBACK.log"

var (
	locFallbackLocaleRE       = regexp.MustCompile(`^\s*(l_[A-Za-z0-9_]+)\s*:\s*$`)
	locFallbackEntryRE        = regexp.MustCompile(`^\s*([A-Za-z0-9_.\-']+)\s*:\s*(?:(\d+)\s*)?"((?:[^"\\]|\\.)*)"\s*(?:#.*)?$`)
	locFallbackEntryLenientRE = regexp.MustCompile(`^\s*([A-Za-z0-9_.\-']+)\s*:\s*(?:(\d+)\s*)?"(.*)"\s*(?:#.*)?$`)
	locFallbackLatin3RE       = regexp.MustCompile(`[A-Za-z]{3}`)
)

type LocFallbackFile struct {
	Path     string
	Total    int
	Fallback int
	Keys     []string
}

type LocFallbackLanguage struct {
	Language string
	Total    int
	Fallback int
	Files    map[string]*LocFallbackFile
}

type LocFallbackReport struct {
	Languages map[string]*LocFallbackLanguage
	msg       exportmsg.Printer
}

type LocFallbackSummary struct {
	Fallback      int                                   `json:"fallback"`
	Headline      string                                `json:"headline"`
	LanguageCount int                                   `json:"language_count"`
	Languages     map[string]LocFallbackSummaryLanguage `json:"languages"`
	Percent       string                                `json:"percent"`
	Total         int                                   `json:"total"`
}

type LocFallbackSummaryLanguage struct {
	Fallback int                               `json:"fallback"`
	Files    map[string]LocFallbackSummaryFile `json:"files"`
	Percent  string                            `json:"percent"`
	Total    int                               `json:"total"`
}

type LocFallbackSummaryFile struct {
	Fallback int `json:"fallback"`
	Total    int `json:"total"`
}

type locFallbackParsedEntry struct {
	language string
	key      string
	value    string
}

type locFallbackParsedFile struct {
	path      string
	entries   []locFallbackParsedEntry
	isReplace bool
}

func IsLocYMLPath(path string) bool {
	p := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return strings.Contains(p, "localisation/") && strings.HasSuffix(p, ".yml")
}

func locFallbackRelPath(path string) string {
	p := strings.ReplaceAll(path, "\\", "/")
	if idx := strings.Index(strings.ToLower(p), "localisation/"); idx >= 0 {
		return p[idx:]
	}
	return p
}

func parseLocFallbackFile(path string, data []byte) locFallbackParsedFile {
	text := string(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}))
	language := ""
	var entries []locFallbackParsedEntry
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSuffix(line, "\r")
		stripped := strings.TrimSpace(line)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		if m := locFallbackLocaleRE.FindStringSubmatch(line); m != nil {
			language = m[1][2:]
			continue
		}
		m := locFallbackEntryRE.FindStringSubmatch(line)
		if m == nil {
			m = locFallbackEntryLenientRE.FindStringSubmatch(line)
		}
		if m == nil || language == "" {
			continue
		}
		entries = append(entries, locFallbackParsedEntry{language: language, key: m[1], value: m[3]})
	}
	rel := locFallbackRelPath(path)
	isReplace := false
	for _, part := range strings.Split(rel, "/") {
		if strings.ToLower(part) == "replace" {
			isReplace = true
			break
		}
	}
	return locFallbackParsedFile{path: rel, entries: entries, isReplace: isReplace}
}

func ScanLocFallback(entries []Entry, msg exportmsg.Printer) *LocFallbackReport {
	var parsed []locFallbackParsedFile
	for _, entry := range entries {
		if !IsLocYMLPath(entry.Path) {
			continue
		}
		parsed = append(parsed, parseLocFallbackFile(entry.Path, entry.Bytes))
	}
	sort.SliceStable(parsed, func(i, j int) bool {
		return strings.ToLower(parsed[i].path) < strings.ToLower(parsed[j].path)
	})

	ordered := append([]locFallbackParsedFile(nil), parsed...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].isReplace != ordered[j].isReplace {
			return !ordered[i].isReplace
		}
		return strings.ToLower(ordered[i].path) < strings.ToLower(ordered[j].path)
	})
	english := map[string]string{}
	for _, pf := range ordered {
		for _, e := range pf.entries {
			if e.language == "english" {
				english[e.key] = e.value
			}
		}
	}

	report := &LocFallbackReport{Languages: map[string]*LocFallbackLanguage{}, msg: msg}
	for _, pf := range parsed {
		for _, e := range pf.entries {
			if e.language == "english" {
				continue
			}
			lang := report.Languages[e.language]
			if lang == nil {
				lang = &LocFallbackLanguage{Language: e.language, Files: map[string]*LocFallbackFile{}}
				report.Languages[e.language] = lang
			}
			stats := lang.Files[pf.path]
			if stats == nil {
				stats = &LocFallbackFile{Path: pf.path}
				lang.Files[pf.path] = stats
			}
			stats.Total++
			lang.Total++
			if e.value == "" {
				continue
			}
			en, ok := english[e.key]
			if !ok || en != e.value {
				continue
			}
			if !locFallbackLatin3RE.MatchString(e.value) {
				continue
			}
			stats.Fallback++
			lang.Fallback++
			stats.Keys = append(stats.Keys, e.key)
		}
	}
	return report
}

func (r *LocFallbackReport) TotalKeys() int {
	if r == nil {
		return 0
	}
	n := 0
	for _, lang := range r.Languages {
		n += lang.Total
	}
	return n
}

func (r *LocFallbackReport) FallbackKeys() int {
	if r == nil {
		return 0
	}
	n := 0
	for _, lang := range r.Languages {
		n += lang.Fallback
	}
	return n
}

func (r *LocFallbackReport) LanguageCount() int {
	if r == nil {
		return 0
	}
	return len(r.Languages)
}

func (r *LocFallbackReport) sortedLanguages() []*LocFallbackLanguage {
	names := make([]string, 0, len(r.Languages))
	for name := range r.Languages {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*LocFallbackLanguage, 0, len(names))
	for _, name := range names {
		out = append(out, r.Languages[name])
	}
	return out
}

func sortedLocFallbackFiles(lang *LocFallbackLanguage) []*LocFallbackFile {
	paths := make([]string, 0, len(lang.Files))
	for path := range lang.Files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	out := make([]*LocFallbackFile, 0, len(paths))
	for _, path := range paths {
		out = append(out, lang.Files[path])
	}
	return out
}

func (r *LocFallbackReport) Headline() string {
	if r == nil || len(r.Languages) == 0 {
		return ""
	}
	languages := make([]string, 0, len(r.Languages))
	for _, lang := range r.sortedLanguages() {
		languages = append(languages, lang.Language)
	}
	return r.msg.Text("loc_fallback.headline", exportmsg.Args{
		"fallback":  locFallbackThousands(r.FallbackKeys()),
		"total":     locFallbackThousands(r.TotalKeys()),
		"percent":   locFallbackPercent(r.FallbackKeys(), r.TotalKeys()),
		"languages": strings.Join(languages, ", "),
		"log":       LocFallbackLogName,
	})
}

func (r *LocFallbackReport) Summary() LocFallbackSummary {
	out := LocFallbackSummary{Languages: map[string]LocFallbackSummaryLanguage{}}
	if r == nil {
		out.Percent = locFallbackPercent(0, 0)
		return out
	}
	for _, lang := range r.sortedLanguages() {
		files := map[string]LocFallbackSummaryFile{}
		for _, f := range sortedLocFallbackFiles(lang) {
			if f.Fallback > 0 {
				files[f.Path] = LocFallbackSummaryFile{Fallback: f.Fallback, Total: f.Total}
			}
		}
		out.Languages[lang.Language] = LocFallbackSummaryLanguage{
			Fallback: lang.Fallback,
			Files:    files,
			Percent:  locFallbackPercent(lang.Fallback, lang.Total),
			Total:    lang.Total,
		}
	}
	out.Fallback = r.FallbackKeys()
	out.Headline = r.Headline()
	out.LanguageCount = r.LanguageCount()
	out.Percent = locFallbackPercent(r.FallbackKeys(), r.TotalKeys())
	out.Total = r.TotalKeys()
	return out
}

func (r *LocFallbackReport) SummaryJSON() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r.Summary()); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func (r *LocFallbackReport) LogText() string {
	var msg exportmsg.Printer
	if r != nil {
		msg = r.msg
	}
	lines := commentLines(msg.Text("loc_fallback.log.header", nil))
	lines = append(lines, "#", "# "+r.Headline(), "#")
	lines = append(lines, "# "+
		locFallbackPad(msg.Text("loc_fallback.log.language", nil), 18)+
		locFallbackPad(msg.Text("loc_fallback.log.counts", nil), 24)+
		msg.Text("loc_fallback.log.percent", nil))
	if r != nil {
		for _, lang := range r.sortedLanguages() {
			lines = append(lines, "# "+
				locFallbackPad(lang.Language, 18)+
				locFallbackPad(locFallbackThousands(lang.Fallback)+" / "+locFallbackThousands(lang.Total), 24)+
				locFallbackPercent(lang.Fallback, lang.Total))
		}
	}
	lines = append(lines, "")
	if r != nil {
		for _, lang := range r.sortedLanguages() {
			lines = append(lines, fmt.Sprintf("## l_%s  %s / %s (%s)",
				lang.Language,
				locFallbackThousands(lang.Fallback),
				locFallbackThousands(lang.Total),
				locFallbackPercent(lang.Fallback, lang.Total)))
			for _, f := range sortedLocFallbackFiles(lang) {
				if f.Fallback == 0 {
					continue
				}
				lines = append(lines, fmt.Sprintf("### %s  %s / %s",
					f.Path, locFallbackThousands(f.Fallback), locFallbackThousands(f.Total)))
				lines = append(lines, f.Keys...)
			}
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func LocFallbackLogEntry(modDir string, report *LocFallbackReport) Entry {
	if modDir == "" || report == nil || report.FallbackKeys() == 0 {
		return Entry{}
	}
	return Entry{
		Path:         modDir + "/" + LocFallbackLogName,
		Bytes:        []byte(report.LogText()),
		Origin:       "generated",
		SourceEntity: "loc_fallback_lint",
	}
}

func locFallbackThousands(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func locFallbackPercent(part, whole int) string {
	if whole <= 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", float64(part*100)/float64(whole))
}

func locFallbackPad(text string, width int) string {
	n := width - len([]rune(text))
	if n < 1 {
		n = 1
	}
	return text + strings.Repeat(" ", n)
}
