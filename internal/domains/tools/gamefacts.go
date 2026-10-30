package tools

import (
	"io/fs"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"haganeworks/internal/gamefiles"
	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/paradox"
)

type gameData struct {
	game func() (fs.FS, error)
}

type (
	readAtKey   struct{}
	subunitsKey struct{}
	eventsKey   struct{}
)

func newGameData(game func() (fs.FS, error)) *gameData { return &gameData{game: game} }

func (g *gameData) current() *gamefiles.Game {
	if g == nil {
		return nil
	}
	game, err := gamefiles.Open(g.game)
	if err != nil {
		return nil
	}
	return game
}

func (g *gameData) extractedAt() any {
	game := g.current()
	if game == nil {
		return nil
	}
	return gamefiles.Derived(game, readAtKey{}, func(*gamefiles.Game) time.Time { return time.Now() }).Format("2006-01-02")
}

func readText(game *gamefiles.Game, name string) string {
	raw, err := game.ReadFile(name)
	if err != nil {
		return ""
	}
	text, ok := gfx.DecodeScriptText(raw)
	if !ok {
		return ""
	}
	return text
}

func parseScript(text string) []*paradox.Entry {
	block, _, err := paradox.Parse(text)
	if err != nil || block == nil {
		return nil
	}
	return block.Entries
}

func keyIs(entry *paradox.Entry, key string) bool {
	return entry != nil && entry.Key != nil && *entry.Key == key
}

func isIdent(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

var lineComment = regexp.MustCompile(`#[^\n]*`)

func stripComments(text string) string { return lineComment.ReplaceAllString(text, "") }

func walkBraces(text string, start int) (int, bool) {
	depth, i := 1, start
	for i < len(text) && depth > 0 {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
		}
		i++
	}
	return i, depth == 0
}

var subunitStatKeys = []string{
	"combat_width", "max_strength", "max_organisation", "soft_attack", "hard_attack",
	"defense", "breakthrough", "manpower", "default_morale", "supply_consumption",
}

var (
	subunitLandTypes = map[string]bool{
		"infantry": true, "motorized": true, "mechanized": true, "armor": true, "artillery": true,
		"anti_tank": true, "anti_air": true, "support": true, "cavalry": true, "flame": true, "amphibious": true,
	}
	subunitNonLandTypes = map[string]bool{
		"fighter": true, "cas": true, "naval_bomber": true, "interceptor": true, "suicide": true,
		"tactical_bomber": true, "strategic_bomber": true, "air_transport": true, "missile": true,
		"submarine": true, "screen_ship": true, "capital_ship": true, "carrier": true,
	}
	identRe    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*`)
	identAllRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
	subUnitsRe = regexp.MustCompile(`\bsub_units\s*=\s*\{`)
	valueRe    = regexp.MustCompile(`^[^\s{}]+`)
)

type vanillaSubunit struct {
	Token             string   `json:"token"`
	Abbreviation      string   `json:"abbreviation"`
	IsSupport         bool     `json:"is_support"`
	Group             string   `json:"group"`
	Types             []string `json:"types"`
	CombatWidth       float64  `json:"combat_width"`
	MaxStrength       float64  `json:"max_strength"`
	MaxOrganisation   float64  `json:"max_organisation"`
	SoftAttack        float64  `json:"soft_attack"`
	HardAttack        float64  `json:"hard_attack"`
	Defense           float64  `json:"defense"`
	Breakthrough      float64  `json:"breakthrough"`
	Manpower          float64  `json:"manpower"`
	DefaultMorale     float64  `json:"default_morale"`
	SupplyConsumption float64  `json:"supply_consumption"`
}

func (g *gameData) subunits() map[string]*vanillaSubunit {
	game := g.current()
	if game == nil {
		return map[string]*vanillaSubunit{}
	}
	return gamefiles.Derived(game, subunitsKey{}, readSubunits)
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\v' || c == '\f'
}

func skipSpaces(text string, k int) int {
	for k < len(text) && (text[k] == ' ' || text[k] == '\t' || text[k] == '\r' || text[k] == '\n') {
		k++
	}
	return k
}

func splitTopBlocks(text string) [][2]string {
	text = stripComments(text)
	var out [][2]string
	n, i, depth := len(text), 0, 0
	for i < n {
		ch := text[i]
		switch {
		case ch == '{':
			depth++
			i++
			continue
		case ch == '}':
			depth--
			i++
			continue
		case depth != 0 || isSpace(ch):
			i++
			continue
		}
		m := identRe.FindString(text[i:])
		if m == "" {
			i++
			continue
		}
		end := i + len(m)
		k := skipSpaces(text, end)
		if k >= n || text[k] != '=' {
			i = end
			continue
		}
		k = skipSpaces(text, k+1)
		if k >= n || text[k] != '{' {
			i = end
			continue
		}
		j, closed := walkBraces(text, k+1)
		if !closed {
			break
		}
		out = append(out, [2]string{m, text[k+1 : j-1]})
		i = j
	}
	return out
}

func parseBlockFields(text string) (map[string]string, map[string]string) {
	scalars, blocks := map[string]string{}, map[string]string{}
	n, i := len(text), 0
	for i < n {
		if isSpace(text[i]) {
			i++
			continue
		}
		key := identRe.FindString(text[i:])
		if key == "" {
			i++
			continue
		}
		end := i + len(key)
		k := skipSpaces(text, end)
		if k >= n || text[k] != '=' {
			i = end
			continue
		}
		k = skipSpaces(text, k+1)
		if k >= n {
			break
		}
		if text[k] == '{' {
			j, _ := walkBraces(text, k+1)
			blocks[key] = text[k+1 : max(j-1, k+1)]
			i = j
			continue
		}
		if value := valueRe.FindString(text[k:]); value != "" {
			scalars[key] = strings.Trim(value, `"`)
			i = k + len(value)
		} else {
			i = k + 1
		}
	}
	return scalars, blocks
}

func readSubunits(game *gamefiles.Game) map[string]*vanillaSubunit {
	out := map[string]*vanillaSubunit{}
	for _, name := range game.Files("common/units", false, ".txt") {
		text := stripComments(readText(game, name))
		for _, loc := range subUnitsRe.FindAllStringIndex(text, -1) {
			end, _ := walkBraces(text, loc[1])
			body := text[loc[1]:max(end-1, loc[1])]
			for _, block := range splitTopBlocks(body) {
				token := block[0]
				scalars, blocks := parseBlockFields(block[1])
				types := identAllRe.FindAllString(blocks["type"], -1)
				if types == nil {
					types = []string{}
				}
				land, nonLand := false, false
				for _, t := range types {
					land = land || subunitLandTypes[t]
					nonLand = nonLand || subunitNonLandTypes[t]
				}
				if !land || nonLand {
					continue
				}
				if _, seen := out[token]; seen {
					continue
				}
				num := func(key string) float64 {
					v, err := strconv.ParseFloat(scalars[key], 64)
					if err != nil {
						return 0
					}
					return v
				}
				support := false
				for _, t := range types {
					support = support || t == "support"
				}
				out[token] = &vanillaSubunit{
					Token: token, Abbreviation: scalars["abbreviation"], IsSupport: support,
					Group: scalars["group"], Types: types,
					CombatWidth: num("combat_width"), MaxStrength: num("max_strength"),
					MaxOrganisation: num("max_organisation"), SoftAttack: num("soft_attack"),
					HardAttack: num("hard_attack"), Defense: num("defense"),
					Breakthrough: num("breakthrough"), Manpower: num("manpower"),
					DefaultMorale: num("default_morale"), SupplyConsumption: num("supply_consumption"),
				}
			}
		}
	}
	return out
}

type vanillaEvent struct {
	NS              string `json:"ns"`
	ID              string `json:"id"`
	Type            string `json:"type"`
	Title           string `json:"title"`
	Picture         string `json:"picture"`
	IsTriggeredOnly bool   `json:"is_triggered_only"`
	FireOnlyOnce    bool   `json:"fire_only_once"`
	Hidden          bool   `json:"hidden"`
	Major           bool   `json:"major"`
	OptionCount     int    `json:"option_count"`
	File            string `json:"file"`
	Line            int    `json:"line"`

	hay string
}

type eventIndex struct {
	entries     []vanillaEvent
	namespaces  int
	generatedAt any
}

var (
	eventTypeRe = regexp.MustCompile(`(?m)^\s*(country_event|news_event|state_event|unit_leader_event|operative_leader_event)\s*=\s*\{`)
	addNSRe     = regexp.MustCompile(`(?m)^\s*add_namespace\s*=\s*([\w.]+)`)
	optionRe    = regexp.MustCompile(`(?m)^\s*option\s*=\s*\{`)
	fieldRes    sync.Map
)

func fieldRe(kind, key string) *regexp.Regexp {
	cacheKey := kind + ":" + key
	if re, ok := fieldRes.Load(cacheKey); ok {
		return re.(*regexp.Regexp)
	}
	var pattern string
	switch kind {
	case "field":
		pattern = `(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*([\w.\-]+)`
	case "quoted":
		pattern = `(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*"([^"]*)"`
	default:
		pattern = `(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*yes\b`
	}
	re := regexp.MustCompile(pattern)
	fieldRes.Store(cacheKey, re)
	return re
}

func eventField(block, key string) string {
	if m := fieldRe("field", key).FindStringSubmatch(block); m != nil {
		return m[1]
	}
	return ""
}

func eventQuoted(block, key string) string {
	if m := fieldRe("quoted", key).FindStringSubmatch(block); m != nil {
		return m[1]
	}
	return ""
}

func (g *gameData) events() *eventIndex {
	game := g.current()
	if game == nil {
		return &eventIndex{generatedAt: nil}
	}
	return gamefiles.Derived(game, eventsKey{}, readEventIndex)
}

func readEventIndex(game *gamefiles.Game) *eventIndex {
	index := &eventIndex{generatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	byNamespace := map[string]bool{}
	for _, name := range game.Files("events", false, ".txt") {
		raw, err := game.ReadFile(name)
		if err != nil {
			continue
		}
		text := strings.ToValidUTF8(strings.TrimPrefix(string(raw), "\uFEFF"), "\uFFFD")
		var namespaces []string
		for _, m := range addNSRe.FindAllStringSubmatch(text, -1) {
			namespaces = append(namespaces, m[1])
		}
		if len(namespaces) == 0 {
			continue
		}
		file := path.Base(name)
		for i := 0; ; {

			if i > 0 && text[i-1] != '\n' {
				nl := strings.IndexByte(text[i:], '\n')
				if nl < 0 {
					break
				}
				i += nl + 1
			}
			loc := eventTypeRe.FindStringSubmatchIndex(text[i:])
			if loc == nil {
				break
			}
			start, end := i+loc[0], i+loc[1]
			evType := text[i+loc[2] : i+loc[3]]
			line := strings.Count(text[:start], "\n") + 1
			j, closed := walkBraces(text, end)
			if !closed {
				break
			}
			i = j
			body := text[end : j-1]
			id := eventField(body, "id")
			if id == "" {
				continue
			}
			ns := namespaces[0]
			if before, _, ok := strings.Cut(id, "."); ok {
				for _, declared := range namespaces {
					if declared == before {
						ns = before
						break
					}
				}
			}
			entry := vanillaEvent{
				NS: ns, ID: id, Type: evType,
				Title:           firstNonEmpty(eventQuoted(body, "title"), eventField(body, "title")),
				Picture:         firstNonEmpty(eventQuoted(body, "picture"), eventField(body, "picture")),
				IsTriggeredOnly: fieldRe("yes", "is_triggered_only").MatchString(body),
				FireOnlyOnce:    fieldRe("yes", "fire_only_once").MatchString(body),
				Hidden:          fieldRe("yes", "hidden").MatchString(body),
				Major:           fieldRe("yes", "major").MatchString(body),
				OptionCount:     len(optionRe.FindAllStringIndex(body, -1)),
				File:            file, Line: line,
			}
			entry.hay = strings.ToLower(strings.Join([]string{entry.ID, entry.NS, entry.Title, entry.Picture, entry.File}, "\n"))
			index.entries = append(index.entries, entry)
			byNamespace[ns] = true
		}
	}
	index.namespaces = len(byNamespace)
	return index
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
