package knowledge

import (
	"embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

//go:embed all:articlesdata
var articleFiles embed.FS

type Article struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Slug         string   `json:"slug"`
	Locale       string   `json:"locale"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	PrimaryTopic string   `json:"primary_topic"`
	Topics       []string `json:"topics"`
	Scenarios    []string `json:"scenarios"`
	GameTerms    []string `json:"game_terms"`
	Aliases      []string `json:"aliases"`
	RelatedIDs   []string `json:"related_ids"`
	Verification string   `json:"verification"`
	GameVersion  string   `json:"game_version"`
	UpdatedAt    string   `json:"updated_at"`
	SearchText   string   `json:"search_text"`
	ReferenceKey string   `json:"reference_key,omitempty"`
	Sources      []Source `json:"sources"`
	Body         string   `json:"body"`
	SourcePath   string   `json:"source_path"`
}

type Source struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type SearchOptions struct {
	Query  string
	Locale string
	Kind   string
	Limit  int
}

type SearchHit struct {
	Article Article `json:"article"`
	Score   int     `json:"score"`
}

var (
	loadOnce sync.Once
	loaded   []Article
	loadErr  error
	wordRE   = regexp.MustCompile(`[a-z0-9_]+|[\p{Han}]+`)
)

func All() ([]Article, error) {
	loadOnce.Do(load)
	if loadErr != nil {
		return nil, loadErr
	}
	out := make([]Article, len(loaded))
	copy(out, loaded)
	return out, nil
}

func Search(options SearchOptions) ([]SearchHit, int, error) {
	query := normalize(options.Query)
	if query == "" {
		return nil, 0, errors.New("query is required")
	}
	locale, err := searchLocale(options.Locale, options.Query)
	if err != nil {
		return nil, 0, err
	}
	kind := normalize(options.Kind)
	limit := options.Limit
	if limit == 0 {
		limit = 8
	}
	if limit < 1 || limit > 20 {
		return nil, 0, errors.New("limit must be between 1 and 20")
	}
	articles, err := All()
	if err != nil {
		return nil, 0, err
	}
	terms := queryTerms(query)
	hits := make([]SearchHit, 0)
	for _, article := range articles {
		if locale != "all" && article.Locale != locale {
			continue
		}
		if kind != "" && normalize(article.Kind) != kind {
			continue
		}
		score := scoreArticle(article, query, terms)
		if score > 0 {
			hits = append(hits, SearchHit{Article: article, Score: score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		if hits[i].Article.UpdatedAt != hits[j].Article.UpdatedAt {
			return hits[i].Article.UpdatedAt > hits[j].Article.UpdatedAt
		}
		return hits[i].Article.ID < hits[j].Article.ID
	})
	total := len(hits)
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, total, nil
}

func Get(id, locale string) (Article, error) {
	id = normalize(id)
	if id == "" {
		return Article{}, errors.New("id is required")
	}
	locale = normalize(locale)
	if locale == "" {
		locale = "zh"
	}
	if locale != "zh" && locale != "en" {
		return Article{}, errors.New("locale must be zh or en")
	}
	articles, err := All()
	if err != nil {
		return Article{}, err
	}
	var fallback *Article
	for i := range articles {
		article := &articles[i]
		if normalize(article.ID) != id && normalize(article.Slug) != id && normalize(article.ReferenceKey) != id {
			continue
		}
		if article.Locale == locale {
			return *article, nil
		}
		if fallback == nil {
			copy := *article
			fallback = &copy
		}
	}
	if fallback != nil {
		return *fallback, nil
	}
	return Article{}, fmt.Errorf("knowledge article %q was not found", id)
}

func load() {
	err := fs.WalkDir(articleFiles, "articlesdata", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		raw, err := articleFiles.ReadFile(path)
		if err != nil {
			return err
		}
		article, err := parseArticle(path, string(raw))
		if err != nil {
			return err
		}
		loaded = append(loaded, article)
		return nil
	})
	if err != nil {
		loadErr = err
		return
	}
	sort.Slice(loaded, func(i, j int) bool {
		if loaded[i].ID != loaded[j].ID {
			return loaded[i].ID < loaded[j].ID
		}
		return loaded[i].Locale < loaded[j].Locale
	})
}

func parseArticle(path, raw string) (Article, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return Article{}, fmt.Errorf("%s: frontmatter is missing", path)
	}
	end := strings.Index(raw[4:], "\n---\n")
	if end < 0 {
		return Article{}, fmt.Errorf("%s: frontmatter is not closed", path)
	}
	header := raw[4 : 4+end]
	body := strings.TrimSpace(raw[4+end+5:])
	values := map[string]string{}
	for _, line := range strings.Split(header, "\n") {
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if ok {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	if scalar(values["status"]) != "published" {
		return Article{}, fmt.Errorf("%s: only published articles may be embedded", path)
	}
	article := Article{
		ID: scalar(values["contentId"]), Kind: scalar(values["kind"]), Slug: scalar(values["slug"]),
		Locale: scalar(values["locale"]), Title: scalar(values["title"]), Summary: scalar(values["summary"]),
		PrimaryTopic: scalar(values["primaryTopic"]), Topics: list(values["topics"]), Scenarios: list(values["scenarios"]),
		GameTerms: list(values["gameTerms"]), Aliases: list(values["aliases"]), RelatedIDs: list(values["relatedIds"]),
		Verification: scalar(values["verification"]), GameVersion: scalar(values["gameVersion"]),
		UpdatedAt: scalar(values["updatedAt"]), SearchText: scalar(values["searchText"]),
		ReferenceKey: scalar(values["referenceKey"]), Sources: parseSources(header),
		Body: body, SourcePath: strings.TrimPrefix(path, "articlesdata/"),
	}
	if article.ID == "" || article.Kind == "" || article.Locale == "" || article.Title == "" || article.Body == "" {
		return Article{}, fmt.Errorf("%s: required article metadata is missing", path)
	}
	return article, nil
}

func parseSources(header string) []Source {
	result := []Source{}
	inSources := false
	current := Source{}
	flush := func() {
		if current.Label != "" && current.URL != "" {
			result = append(result, current)
		}
		current = Source{}
	}
	for _, line := range strings.Split(header, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "sources:" {
			inSources = true
			continue
		}
		if !inSources {
			continue
		}
		if line != "" && line[0] != ' ' && line[0] != '\t' {
			break
		}
		if strings.HasPrefix(trimmed, "- label:") {
			flush()
			current.Label = scalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- label:")))
		}
		if strings.HasPrefix(trimmed, "url:") {
			current.URL = scalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "url:")))
		}
	}
	flush()
	return result
}

func scalar(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if decoded, err := strconv.Unquote(value); err == nil {
			return decoded
		}
	}
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
	}
	return value
}

func list(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || value == "[]" {
		return []string{}
	}
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = value[1 : len(value)-1]
	}
	reader := csv.NewReader(strings.NewReader(value))
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	items, err := reader.Read()
	if err != nil {
		return []string{scalar(value)}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item = scalar(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func searchLocale(value, query string) (string, error) {
	value = normalize(value)
	if value == "" || value == "auto" {
		for _, r := range query {
			if unicode.Is(unicode.Han, r) {
				return "zh", nil
			}
		}
		return "en", nil
	}
	if value != "zh" && value != "en" && value != "all" {
		return "", errors.New("locale must be auto, zh, en, or all")
	}
	return value, nil
}

func normalize(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(norm.NFKC.String(value))), " ")
}

func queryTerms(query string) []string {
	seen := map[string]bool{}
	out := []string{}
	add := func(term string) {
		term = normalize(term)
		if term != "" && !seen[term] {
			seen[term] = true
			out = append(out, term)
		}
	}
	for _, token := range wordRE.FindAllString(query, -1) {
		add(token)
		if strings.Contains(token, "_") {
			for _, part := range strings.Split(token, "_") {
				if len(part) >= 2 {
					add(part)
				}
			}
		}
		runes := []rune(token)
		if len(runes) > 2 && unicode.Is(unicode.Han, runes[0]) {
			for i := 0; i < len(runes)-1; i++ {
				add(string(runes[i : i+2]))
			}
		}
	}
	return out
}

func scoreArticle(article Article, query string, terms []string) int {
	id := normalize(strings.Join([]string{article.ID, article.Slug, article.ReferenceKey}, " "))
	title := normalize(article.Title)
	tags := normalize(strings.Join(append(append(append([]string{}, article.GameTerms...), article.Aliases...), append(article.Topics, article.Scenarios...)...), " "))
	summary := normalize(article.Summary + " " + article.SearchText)
	body := normalize(article.Body)
	score := 0
	for _, field := range []struct {
		text   string
		phrase int
		term   int
	}{{id, 140, 28}, {title, 100, 22}, {tags, 70, 15}, {summary, 55, 11}, {body, 16, 3}} {
		if strings.Contains(field.text, query) {
			score += field.phrase
		}
		for _, term := range terms {
			if strings.Contains(field.text, term) {
				score += field.term
			}
		}
	}
	return score
}
