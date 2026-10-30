package build

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/importsvc/world"
	"haganeworks/internal/hoi4/loc"
	"haganeworks/internal/hoi4/paradox"
)

var historyCountryRe = regexp.MustCompile(`^([A-Z0-9]{3})\s*-\s*(.+)\.txt$`)

var countryAliasExtras = map[string][]string{"PRC": {"china_communist"}}

type countryAliases struct {
	Source      string `json:"source"`
	GameVersion string `json:"game_version"`

	Aliases map[string][]string `json:"aliases"`

	Names map[string]string `json:"names"`

	Files map[string]string `json:"files"`
}

func buildCountryAliases(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	out := countryAliases{
		Source: "history/countries/TAG - Name.txt", GameVersion: in.GameVersion,
		Aliases: map[string][]string{}, Names: map[string]string{}, Files: map[string]string{},
	}
	for _, rel := range baseFiles(in, "history/countries", ".txt") {
		m := historyCountryRe.FindStringSubmatch(path.Base(rel))
		if m == nil {
			continue
		}
		tag, name := m[1], m[2]
		if _, seen := out.Names[tag]; seen {
			continue
		}
		out.Names[tag] = name
		out.Files[tag] = path.Base(rel)
		var words []string
		for _, word := range strings.Fields(strings.ToLower(name)) {
			word = strings.Map(func(r rune) rune {
				if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
					return r
				}
				return -1
			}, word)
			if word != "" {
				words = append(words, word)
			}
		}
		var candidates []string
		switch len(words) {
		case 0:
		case 1:
			candidates = words
		default:
			candidates = []string{words[0], strings.Join(words, "_"), strings.Join(words, "")}
		}
		candidates = append(candidates, countryAliasExtras[tag]...)
		seen := map[string]bool{}
		var aliases []string
		for _, alias := range candidates {
			if alias == strings.ToLower(tag) || seen[alias] {
				continue
			}
			seen[alias] = true
			aliases = append(aliases, alias)
		}
		if len(aliases) > 0 {
			out.Aliases[tag] = aliases
		}
	}
	if len(out.Names) == 0 {
		return nil, missingFile("history/countries")
	}
	return marshal(out)
}

func countryAliasesOf(raw []byte) *countryAliases {
	out := &countryAliases{}
	_ = json.Unmarshal(raw, out)
	return out
}

func (a *countryAliases) fileCountry(name string) string {
	stem := strings.TrimSuffix(name, path.Ext(name))
	if len(stem) == 3 && strings.ToUpper(stem) == stem {
		if _, ok := a.Names[stem]; ok {
			return stem
		}
	}
	low := strings.ToLower(stem)
	best, bestLen := "", 0
	tags := sortedKeys(a.Aliases)
	for _, tag := range tags {
		for _, alias := range a.Aliases[tag] {
			if low == alias {
				return tag
			}
			if len(alias) > bestLen && strings.HasPrefix(low, alias+"_") && strings.Contains(alias, "_") {
				best, bestLen = tag, len(alias)
			}
		}
	}
	if best != "" {
		return best
	}
	return "shared"
}

var (
	tagFileRe  = regexp.MustCompile(`^\s*([A-Z0-9]{3})\s*=\s*"countries/(.+?)"`)
	dtagRe     = regexp.MustCompile(`^D[0-9][0-9]\.txt$`)
	dtagColor  = regexp.MustCompile(`color\s*=\s*\{\s*([0-9]+)\s+([0-9]+)\s+([0-9]+)\s*\}`)
	cosmeticRe = regexp.MustCompile(`^([A-Za-z0-9_]+)\s*=\s*\{`)
)

func countryFilenames(in *gameindex.Input) map[string]string {
	out := map[string]string{}
	for _, rel := range baseFiles(in, "common/country_tags", ".txt") {
		text, _ := readBase(in, rel)
		for _, line := range strings.Split(text, "\n") {
			m := tagFileRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			base := m[2]
			if i := strings.LastIndexByte(base, '/'); i >= 0 {
				base = base[i+1:]
			}
			if _, seen := out[m[1]]; !seen {
				out[m[1]] = base
			}
		}
	}
	return out
}

func countryCulture(in *gameindex.Input, file string) *ordered {
	text, ok := readBase(in, "common/countries/"+file)
	if !ok {
		return nil
	}
	out := newOrdered()
	if dtagRe.MatchString(file) {
		if m := dtagColor.FindStringSubmatch(text); m != nil {
			out.set("color", rgbOf(m[1:4]))
		}
		return out
	}
	for _, entry := range parseEntries(text) {
		switch {
		case keyIs(entry, "graphical_culture") && entry.Block == nil:
			out.set("graphical_culture", entry.Scalar.StrValue())
		case keyIs(entry, "graphical_culture_2d") && entry.Block == nil:
			out.set("graphical_culture_2d", entry.Scalar.StrValue())
		case keyIs(entry, "color") && entry.Block != nil && (entry.Qualifier == nil || strings.EqualFold(*entry.Qualifier, "rgb")):
			var parts []string
			for _, item := range entry.Block.Entries {
				if item.Key == nil && item.Block == nil {
					parts = append(parts, item.Scalar.StrValue())
				}
			}
			if len(parts) == 3 {
				if color := rgbOf(parts); color != nil {
					out.set("color", color)
				}
			}
		}
	}
	return out
}

func rgbOf(parts []string) []int {
	out := make([]int, 0, 3)
	for _, p := range parts {
		n, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil || n < 0 || n > 255 {
			return nil
		}
		out = append(out, int(n))
	}
	return out
}

func stateFacts(in *gameindex.Input) (filenames, owners map[string]string) {
	filenames, owners = map[string]string{}, map[string]string{}
	for _, rel := range baseFiles(in, "history/states", ".txt") {
		text, _ := readBase(in, rel)
		rows, _ := world.ImportStateFiles(text, path.Base(rel))
		for _, row := range rows {
			v, _ := row.Get("state_id")
			id, ok := intOf(v)
			if !ok {
				continue
			}
			key := strconv.FormatInt(id, 10)
			if _, seen := filenames[key]; seen {
				continue
			}
			filenames[key] = path.Base(rel)
			if owner, _ := row.Get("owner"); owner != nil {
				if s, ok := owner.(string); ok && strings.TrimSpace(s) != "" {
					owners[key] = strings.TrimSpace(s)
				}
			}
		}
	}
	return filenames, owners
}

func intOf(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case *big.Int:
		if n != nil && n.IsInt64() {
			return n.Int64(), true
		}
	case float64:
		return int64(n), true
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		return i, err == nil
	}
	return 0, false
}

func sortedNumericKeys(m map[string]string) *ordered {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, _ := strconv.Atoi(keys[i])
		b, _ := strconv.Atoi(keys[j])
		return a < b
	})
	out := newOrdered()
	for _, k := range keys {
		out.set(k, m[k])
	}
	return out
}

func buildWorldTables(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	aliases := countryAliasesOf(in.Dataset(gameindex.CountryAliases))
	filenames := countryFilenames(in)
	if len(filenames) == 0 {
		return nil, missingFile("common/country_tags")
	}
	cultures := newOrdered()
	for _, tag := range sortedKeys(filenames) {
		file := filenames[tag]
		if dtagRe.MatchString(tag + ".txt") {

			file = tag + ".txt"
		}
		if culture := countryCulture(in, file); culture != nil && len(culture.keys) > 0 {
			cultures.set(tag, culture)
		}
	}
	stateFiles, stateOwners := stateFacts(in)
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	cosmetic := map[string]bool{}
	if text, ok := readBase(in, "common/countries/cosmetic.txt"); ok {
		for _, line := range strings.Split(text, "\n") {
			if m := cosmeticRe.FindStringSubmatch(line); m != nil {
				cosmetic[m[1]] = true
			}
		}
	}
	ideologyBase := ""
	if raw, err := game(in).ReadFile("common/ideologies/00_ideologies.txt"); err == nil {
		ideologyBase = string(raw)
	}
	countryFiles := newOrdered()
	for _, tag := range sortedKeys(filenames) {
		countryFiles.set(tag, filenames[tag])
	}
	out := newOrdered()
	out.set("vanilla_tags", sortedKeys(aliases.Aliases))
	out.set("state_filenames", sortedNumericKeys(stateFiles))
	out.set("state_owners", sortedNumericKeys(stateOwners))
	out.set("country_filenames", countryFiles)
	out.set("country_cultures", cultures)
	out.set("cosmetic_tags", sortedKeys(cosmetic))
	out.set("ideology_base_text", ideologyBase)
	return marshal(out)
}

func buildCosmetic(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	raw, err := game(in).ReadFile("common/countries/cosmetic.txt")
	if err != nil {
		return nil, err
	}
	return marshal(map[string]any{"path": "common/countries/cosmetic.txt", "game_version": in.GameVersion, "bytes": raw})
}

var locLanguages = [][2]string{
	{"english", "en"}, {"simp_chinese", "zh"}, {"russian", "ru"}, {"german", "de"}, {"spanish", "es"},
	{"french", "fr"}, {"japanese", "ja"}, {"korean", "ko"}, {"polish", "pl"}, {"braz_por", "pt"},
}

var continentGroups = map[string]string{
	"europe": "europe", "north_america": "americas", "south_america": "americas",
	"australia": "oceania", "africa": "africa", "asia": "asia", "middle_east": "asia",
}

type countryTag struct {
	Tag             string            `json:"tag"`
	NameEn          string            `json:"name_en"`
	NameZh          string            `json:"name_zh"`
	I18nKey         string            `json:"i18n_key"`
	IdeologyDefault string            `json:"ideology_default"`
	FlagEmoji       string            `json:"flag_emoji"`
	Continent       string            `json:"continent"`
	Names           map[string]string `json:"names"`
}

func buildCountryTags(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	g := game(in)
	var tables struct {
		CountryFilenames map[string]string `json:"country_filenames"`
	}
	_ = json.Unmarshal(in.Dataset(gameindex.WorldTables), &tables)
	histories := countryAliasesOf(in.Dataset(gameindex.CountryAliases)).Files
	tags := map[string]bool{}
	for tag := range tables.CountryFilenames {
		if !dtagRe.MatchString(tag + ".txt") {
			tags[tag] = true
		}
	}

	locs := map[string]map[string]string{}
	for _, lang := range locLanguages {
		entries := map[string]string{}
		for _, rel := range g.Files("localisation/"+lang[0], true, ".yml") {
			if !strings.Contains(strings.ToLower(path.Base(rel)), "countr") {
				continue
			}
			raw, err := g.ReadFile(rel)
			if err != nil {
				continue
			}
			_, parsed, _ := loc.ParseLocText(pyText(raw))
			for key, value := range parsed {
				if len(key) >= 3 && tags[key[:3]] && (len(key) == 3 || key[3] == '_') && strings.TrimSpace(value) != "" {
					entries[key] = value
				}
			}
		}
		locs[lang[1]] = entries
	}
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	continents := provinceContinents(in)
	stateProvince := stateFirstProvince(in)
	out := []countryTag{}
	for _, tag := range sortedKeys(tags) {
		entry := countryTag{Tag: tag, IdeologyDefault: "neutrality", Continent: "other", Names: map[string]string{}}
		capital, party := 0, ""
		if file := histories[tag]; file != "" {
			entries := parseEntries(mustBase(in, "history/countries/"+file))
			capital = historyCapital(entries)
			party = rulingParty(entries)
			if party != "" {
				entry.IdeologyDefault = party
			}
		}
		for _, lang := range locLanguages {
			code := lang[1]
			for _, key := range []string{tag + "_" + party, tag, tag + "_neutrality", tag + "_democratic", tag + "_fascism", tag + "_communism"} {
				if v := locs[code][key]; v != "" && key != tag+"_" {
					entry.Names[code] = v
					break
				}
			}
		}
		entry.NameEn = entry.Names["en"]
		if entry.NameEn == "" {
			entry.NameEn = tag
		}
		entry.NameZh = entry.Names["zh"]
		if entry.NameZh == "" {
			entry.NameZh = entry.NameEn
		}
		entry.I18nKey = camelKey(entry.NameEn)
		if province, ok := stateProvince[capital]; ok {
			if group := continentGroups[continents[province]]; group != "" {
				entry.Continent = group
			}
		}
		out = append(out, entry)
	}
	return marshal(map[string]any{"game_version": in.GameVersion, "tags": out})
}

func historyCapital(entries []*paradox.Entry) int {
	if c, ok := atoiPrefix(strings.TrimSpace(scalarField(entries, "capital"))); ok {
		return c
	}
	var find func([]*paradox.Entry) int
	find = func(entries []*paradox.Entry) int {
		for _, e := range entries {
			switch {
			case keyIs(e, "capital") && e.Block == nil:
				if c, ok := atoiPrefix(strings.TrimSpace(e.Scalar.StrValue())); ok {
					return c
				}
			case keyIs(e, "set_capital") && e.Block != nil:
				if c, ok := atoiPrefix(strings.TrimSpace(scalarField(e.Block.Entries, "state"))); ok {
					return c
				}
			case e.Block != nil:
				if c := find(e.Block.Entries); c != 0 {
					return c
				}
			}
		}
		return 0
	}
	return find(entries)
}

func rulingParty(entries []*paradox.Entry) string {
	for _, entry := range entries {
		if keyIs(entry, "set_politics") && entry.Block != nil {
			switch party := scalarField(entry.Block.Entries, "ruling_party"); party {
			case "democratic", "fascism", "communism", "neutrality":
				return party
			}
		}
	}
	return ""
}

func provinceContinents(in *gameindex.Input) map[int]string {
	g := game(in)
	var names []string
	if raw, err := g.ReadFile("map/continent.txt"); err == nil {
		for _, entry := range childBlock(parseEntries(pyText(raw)), "continents") {
			if entry.Key == nil && entry.Block == nil {
				names = append(names, entry.Scalar.StrValue())
			}
		}
	}
	out := map[int]string{}
	raw, err := g.ReadFile("map/definition.csv")
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Split(strings.TrimSpace(line), ";")
		if len(fields) < 8 {
			continue
		}
		id, err1 := strconv.Atoi(strings.TrimSpace(fields[0]))
		c, err2 := strconv.Atoi(strings.TrimSpace(fields[7]))
		if err1 != nil || err2 != nil || c < 1 || c > len(names) {
			continue
		}
		out[id] = names[c-1]
	}
	return out
}

func stateFirstProvince(in *gameindex.Input) map[int]int {
	out := map[int]int{}
	for _, rel := range baseFiles(in, "history/states", ".txt") {
		text, _ := readBase(in, rel)
		for _, entry := range parseEntries(text) {
			if !keyIs(entry, "state") || entry.Block == nil {
				continue
			}
			id, ok := atoiPrefix(strings.TrimSpace(scalarField(entry.Block.Entries, "id")))
			if !ok {
				continue
			}
			for _, item := range childBlock(entry.Block.Entries, "provinces") {
				if item.Key == nil && item.Block == nil {
					if p, ok := atoiPrefix(item.Scalar.StrValue()); ok {
						if _, seen := out[id]; !seen {
							out[id] = p
						}
						break
					}
				}
			}
		}
	}
	return out
}

func camelKey(name string) string {
	var words []string
	for _, word := range strings.FieldsFunc(name, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		words = append(words, strings.ToLower(word))
	}
	for i := 1; i < len(words); i++ {
		r := []rune(words[i])
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, "")
}

var dlcPrefixes = map[string]string{
	"Together for Victory": "TFV", "Death or Dishonor": "DOD", "Waking the Tiger": "WTT",
	"Man the Guns": "MTG", "La Resistance": "LAR", "Battle for the Bosporus": "BFTB",
	"No Step Back": "NSB", "By Blood Alone": "BBA", "Arms Against Tyranny": "AAT",
	"Trial of Allegiance": "TOA", "Gotterdammerung": "GOT", "Graveyard of Empires": "GOE",
}

var (
	hasDLCRe      = regexp.MustCompile(`has_dlc\s*=\s*"([^"]+)"`)
	dlcChecksumRe = regexp.MustCompile(`(?m)^\s*affects_checksum\s*=\s*yes`)
	dlcNumRe      = regexp.MustCompile(`^dlc(\d+)`)
	dlcNameRe     = regexp.MustCompile(`(?m)^\s*name\s*=\s*"([^"]*)"`)
)

type dlcRow struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Prefix string `json:"prefix"`
	Source string `json:"source"`
	order  int
}

func buildDLCNames(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	counts := map[string]int{}
	for _, dir := range []string{"common", "events", "history"} {
		for _, rel := range walkFiles(in.FS, dir, ".txt") {
			text, _ := readBase(in, rel)
			for _, m := range hasDLCRe.FindAllStringSubmatch(text, -1) {
				counts[m[1]]++
			}
		}
		if err := checkCtx(ctx); err != nil {
			return nil, err
		}
	}
	rows := map[string]*dlcRow{}
	for _, dir := range layerDirs(in.FS) {
		for _, rel := range listFiles(in.FS, dir, ".dlc") {
			text, _ := readBase(in, rel)
			m := dlcNameRe.FindStringSubmatch(text)
			if m == nil || m[1] == "" {
				continue
			}
			base := path.Base(dir)
			number := 1 << 20
			if n := dlcNumRe.FindStringSubmatch(base); n != nil {
				number, _ = strconv.Atoi(n[1])
			}
			checksum := dlcChecksumRe.MatchString(text)
			kind := "cosmetic"
			switch {
			case checksum:
				kind = "expansion"
			case counts[m[1]] > 0:
				kind = "content_pack"
			}
			source := base
			if c := counts[m[1]]; c > 0 {
				source += fmt.Sprintf(" · vanilla %dx", c)
			}
			if _, seen := rows[m[1]]; !seen {
				rows[m[1]] = &dlcRow{Name: m[1], Kind: kind, Prefix: dlcPrefixes[m[1]], Source: source, order: number}
			}
		}
	}

	for _, rel := range listFiles(in.FS, "dlc_metadata/dlc_info", ".txt") {
		for _, entry := range childBlock(parseEntries(mustBase(in, rel)), "dlcs") {
			if entry.Key == nil || entry.Block == nil {
				continue
			}
			name := strings.TrimSpace(scalarField(entry.Block.Entries, "name"))
			if name == "" {
				continue
			}
			if _, seen := rows[name]; seen {
				continue
			}
			number := 1 << 20
			if n := dlcNumRe.FindStringSubmatch(*entry.Key); n != nil {
				number, _ = strconv.Atoi(n[1])
			}
			kind, source := "cosmetic", *entry.Key+" (dlc_info, not installed)"
			if c := counts[name]; c > 0 {
				kind = "expansion"
				source += fmt.Sprintf(" · vanilla %dx", c)
			}
			rows[name] = &dlcRow{Name: name, Kind: kind, Prefix: dlcPrefixes[name], Source: source, order: number}
		}
	}
	for name, c := range counts {
		if _, seen := rows[name]; !seen {
			rows[name] = &dlcRow{Name: name, Kind: "expansion", Prefix: dlcPrefixes[name],
				Source: fmt.Sprintf("vanilla %dx (no dlc folder)", c), order: 1 << 21}
		}
	}
	list := make([]*dlcRow, 0, len(rows))
	for _, row := range rows {
		list = append(list, row)
	}
	sort.Slice(list, func(i, j int) bool {
		ci, cj := list[i].Kind == "cosmetic", list[j].Kind == "cosmetic"
		if ci != cj {
			return !ci
		}
		if list[i].order != list[j].order {
			return list[i].order < list[j].order
		}
		return list[i].Name < list[j].Name
	})
	return marshal(map[string]any{
		"game_version": shortVersion(in.GameVersion),
		"sources": []string{
			"dlc/*/*.dlc and integrated_dlc/*/*.dlc `name =` (the installation)",
			"dlc_metadata/dlc_info/*.txt (every DLC the game knows)",
			"`has_dlc = \"...\"` in the game's common/ events/ history/ scripts",
			"prefix: the product's namespace prefixes (informational only)",
		},
		"dlcs": list,
	})
}
