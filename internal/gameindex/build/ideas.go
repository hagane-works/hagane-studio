package build

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"haganeworks/internal/gameindex"
)

var (
	ideaBlockNameRe = regexp.MustCompile(`([A-Za-z_][\p{L}\p{N}_.\-]*)\s*=\s*\{`)
	outerIdeasRe    = regexp.MustCompile(`(?m)^\s*ideas\s*=\s*\{`)
	ideaNameRe      = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	ideaPictureRe   = regexp.MustCompile(`(?m)^\s*picture\s*=\s*([A-Za-z_][\p{L}\p{N}_.\-]*)\s*$`)
	ideaSpriteRe    = regexp.MustCompile(`name\s*=\s*"?GFX_idea_([A-Za-z0-9_.\-]+)"?`)
)

var ideaCategoryFlags = map[string]bool{"law": true, "designer": true, "use_list_view": true}

type ideaBlock struct {
	name, body string
	line       int
}

func walkIdeaBlocks(text string) []ideaBlock {
	var out []ideaBlock
	i := 0
	for i < len(text) {
		m := ideaBlockNameRe.FindStringSubmatchIndex(text[i:])
		if m == nil {
			break
		}
		name := text[i+m[2] : i+m[3]]
		line := strings.Count(text[:i+m[0]], "\n") + 1
		start := i + m[1]
		end, closed := walkBraces(text, start)
		if !closed {
			break
		}
		out = append(out, ideaBlock{name, text[start : end-1], line})
		i = end
	}
	return out
}

type ideaRow struct {
	Category string  `json:"category"`
	IdeaID   string  `json:"idea_id"`
	Picture  *string `json:"picture"`
	File     string  `json:"file"`
	Line     int     `json:"line"`
}

func scanIdeaFile(rel, text string) []ideaRow {
	loc := outerIdeasRe.FindStringIndex(text)
	if loc == nil {
		return nil
	}
	end, closed := walkBraces(text, loc[1])
	if !closed {
		return nil
	}
	region := text[loc[1] : end-1]
	var rows []ideaRow
	for _, category := range walkIdeaBlocks(region) {
		if ideaCategoryFlags[category.name] {
			continue
		}
		for _, idea := range walkIdeaBlocks(category.body) {
			if !ideaNameRe.MatchString(idea.name) || ideaCategoryFlags[idea.name] {
				continue
			}
			var picture *string
			if m := ideaPictureRe.FindStringSubmatch(idea.body); m != nil {
				p := m[1]
				picture = &p
			}
			rows = append(rows, ideaRow{Category: category.name, IdeaID: idea.name, Picture: picture, File: rel, Line: idea.line})
		}
	}
	return rows
}

type counter struct {
	keys   []string
	counts map[string]int
}

func (c *counter) add(key string) {
	if c.counts == nil {
		c.counts = map[string]int{}
	}
	if _, ok := c.counts[key]; !ok {
		c.keys = append(c.keys, key)
	}
	c.counts[key]++
}

func (c *counter) byCountDesc() *ordered {
	keys := append([]string(nil), c.keys...)
	sort.SliceStable(keys, func(i, j int) bool { return c.counts[keys[i]] > c.counts[keys[j]] })
	out := newOrdered()
	for _, key := range keys {
		out.set(key, c.counts[key])
	}
	return out
}

func buildIdeaIndex(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	var ideas []ideaRow
	var byCategory, byFile counter
	scan := func(rel string) {
		text, ok := readBase(in, rel)
		if !ok {
			return
		}
		for _, row := range scanIdeaFile(rel, text) {
			ideas = append(ideas, row)
			byCategory.add(row.Category)
			byFile.add(rel)
		}
	}
	for _, rel := range baseFiles(in, "common/ideas", ".txt") {
		scan(rel)
	}
	for _, dir := range layerDirs(in.FS) {
		for _, rel := range listFiles(in.FS, dir+"/common/ideas", ".txt") {
			scan(rel)
		}
	}
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}

	sprites := map[string]bool{}
	gfxFiles := 0
	for _, top := range []string{"interface", "dlc", "integrated_dlc"} {
		for _, rel := range walkFiles(in.FS, top, ".gfx") {
			gfxFiles++
			text, _ := readBase(in, rel)
			for _, m := range ideaSpriteRe.FindAllStringSubmatch(text, -1) {
				sprites[m[1]] = true
			}
		}
	}
	referenced := map[string]bool{}
	for _, idea := range ideas {
		if idea.Picture != nil {
			if name := strings.TrimPrefix(*idea.Picture, "GFX_idea_"); name != "" {
				referenced[name] = true
			}
		}
	}
	orphans := []string{}
	for name := range referenced {
		if !sprites[name] {
			orphans = append(orphans, name)
		}
	}
	sort.Strings(orphans)
	if len(ideas) == 0 {
		return nil, fmt.Errorf("no ideas found in common/ideas")
	}
	if ideas == nil {
		ideas = []ideaRow{}
	}
	out := newOrdered()
	out.set("total", len(ideas))
	out.set("game_version", shortVersion(in.GameVersion))
	out.set("generated_at", time.Now().UTC().Format(time.RFC3339))
	out.set("note", "picture_sprites = every `name = \"GFX_idea_XYZ\"` spriteType in interface/*.gfx + dlc/*/**/*.gfx + integrated_dlc/*/**/*.gfx, prefix stripped: the legal `picture = X` values. ideas[].picture is only the referenced subset; consumers use the union picture_sprites ∪ {ideas[].picture} (picture_sprites_orphans lists referenced names with no spriteType).")
	out.set("picture_sprites_total", len(sprites))
	out.set("picture_sprites", sortedKeys(sprites))
	out.set("picture_sprites_orphans", orphans)
	out.set("picture_sprites_scanned_files", gfxFiles)
	out.set("by_category", byCategory.byCountDesc())
	out.set("by_file", byFile.byCountDesc())
	out.set("ideas", ideas)
	return marshal(out)
}

func ideaIndex(raw []byte) []ideaRow {
	var idx struct {
		Ideas []ideaRow `json:"ideas"`
	}
	_ = json.Unmarshal(raw, &idx)
	return idx.Ideas
}

func buildIdeaTags(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	g := game(in)
	block, slots := map[string]bool{}, map[string]bool{}
	for _, name := range g.Files("common/idea_tags", false, ".txt") {
		for _, category := range childBlock(readScript(g, name), "idea_categories") {
			if category.Key == nil || category.Block == nil {
				continue
			}
			block[*category.Key] = true
			for _, field := range category.Block.Entries {
				if (keyIs(field, "slot") || keyIs(field, "character_slot")) && field.Block == nil {
					if slot := strings.TrimSpace(field.Scalar.StrValue()); slot != "" {
						slots[slot] = true
					}
				}
			}
		}
	}
	if len(block) == 0 {
		return nil, fmt.Errorf("no idea_categories found in common/idea_tags")
	}
	runtime := map[string]bool{}
	for _, idea := range ideaIndex(in.Dataset(gameindex.IdeaIndex)) {
		runtime[idea.Category] = true
	}
	all := map[string]bool{}
	for k := range block {
		all[k] = true
	}
	for k := range runtime {
		all[k] = true
	}
	return marshal(map[string]any{
		"source":             "common/idea_tags/*.txt idea_categories + the categories of common/ideas/*.txt",
		"game_version":       shortVersion(in.GameVersion),
		"idea_categories":    sortedKeys(block),
		"idea_slots":         sortedKeys(slots),
		"runtime_categories": sortedKeys(runtime),
		"all_categories":     sortedKeys(all),
	})
}

var ideaGroups = map[string]string{
	"country":                  "country_spirits",
	"hidden_ideas":             "hidden_spirits",
	"economy":                  "laws",
	"trade_laws":               "laws",
	"mobilization_laws":        "laws",
	"aircraft_manufacturer":    "design_companies",
	"tank_manufacturer":        "design_companies",
	"naval_manufacturer":       "design_companies",
	"materiel_manufacturer":    "design_companies",
	"industrial_concern":       "design_companies",
	"academy_spirit":           "officer_corps_spirits",
	"army_spirit":              "officer_corps_spirits",
	"division_command_spirit":  "officer_corps_spirits",
	"naval_academy_spirit":     "officer_corps_spirits",
	"navy_spirit":              "officer_corps_spirits",
	"naval_command_spirit":     "officer_corps_spirits",
	"air_force_academy_spirit": "officer_corps_spirits",
	"air_force_spirit":         "officer_corps_spirits",
	"air_force_command_spirit": "officer_corps_spirits",
	"political_advisor":        "old_advisors",
	"theorist":                 "old_advisors",
	"army_chief":               "old_advisors",
	"navy_chief":               "old_advisors",
	"air_chief":                "old_advisors",
	"high_command":             "old_advisors",
}

func buildIdeasUI(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	aliases := countryAliasesOf(in.Dataset(gameindex.CountryAliases))
	byGroup, byCountry := newOrdered(), newOrdered()
	groupCounts := newOrdered()
	ids := map[string]bool{}
	ideas := ideaIndex(in.Dataset(gameindex.IdeaIndex))
	for _, idea := range ideas {
		group := ideaGroups[idea.Category]
		if group == "" {
			group = "country_spirits"
		}
		country := aliases.fileCountry(path.Base(idea.File))
		rows, _ := byGroup.values[group].([][3]string)
		byGroup.set(group, append(rows, [3]string{idea.IdeaID, idea.Category, country}))
		crow, _ := byCountry.values[country].([][3]string)
		byCountry.set(country, append(crow, [3]string{idea.IdeaID, idea.Category, group}))
		n, _ := groupCounts.values[group].(int)
		groupCounts.set(group, n+1)
		ids[idea.IdeaID] = true
	}
	out := newOrdered()
	out.set("meta", map[string]any{
		"total":  len(ideas),
		"source": "HOI4 " + shortVersion(in.GameVersion) + " common/ideas/",
		"groups": groupCounts,
	})
	out.set("by_group", byGroup)
	out.set("by_country", byCountry)
	out.set("all_ids", sortedKeys(ids))
	return marshal(out)
}
