package build

import (
	"bytes"
	"context"
	"encoding/json"
	"path"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/gamefiles"
	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/loc"
)

var locRefRe = regexp.MustCompile(`\$([A-Za-z0-9_.\-]+)\$`)

func locFiles(g *gamefiles.Game, dir string) []string {
	var normal, replace []string
	for _, rel := range g.Files("localisation/"+dir, true, ".yml") {
		if strings.Contains(strings.ToLower(rel), "/replace/") {
			replace = append(replace, rel)
		} else {
			normal = append(normal, rel)
		}
	}
	sortFold(normal)
	sortFold(replace)
	return append(normal, replace...)
}

func scanLoc(files [][]byte, want map[string]bool, values map[string]string) {
	for _, data := range files {
		data = bytes.TrimPrefix(data, utf8BOM)
		for len(data) > 0 {
			line := data
			if nl := bytes.IndexByte(data, '\n'); nl >= 0 {
				line, data = data[:nl], data[nl+1:]
			} else {
				data = nil
			}
			line = bytes.TrimSpace(line)
			if len(line) == 0 || line[0] == '#' {
				continue
			}
			colon := bytes.IndexByte(line, ':')
			if colon <= 0 || !want[string(line[:colon])] {
				continue
			}
			first := bytes.IndexByte(line[colon:], '"')
			last := bytes.LastIndexByte(line, '"')
			if first < 0 || colon+first >= last {
				continue
			}
			values[string(line[:colon])] = loc.DecodeLocEscapes(string(line[colon+first+1 : last]))
		}
	}
}

func locNames(ctx context.Context, g *gamefiles.Game, keys []string) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, lang := range locLanguages {
		if ctx.Err() != nil {
			return out
		}
		var files [][]byte
		for _, rel := range locFiles(g, lang[0]) {
			if raw, err := g.ReadFile(rel); err == nil {
				files = append(files, raw)
			}
		}
		if len(files) == 0 {
			continue
		}
		values := map[string]string{}
		want := setOf(keys)
		for pass := 0; pass < 4 && len(want) > 0; pass++ {
			scanLoc(files, want, values)
			next := map[string]bool{}
			for key := range want {
				for _, m := range locRefRe.FindAllStringSubmatch(values[key], -1) {
					if _, known := values[m[1]]; !known {
						next[m[1]] = true
					}
				}
			}
			want = next
		}
		var resolve func(key string, depth int) (string, bool)
		resolve = func(key string, depth int) (string, bool) {
			text, ok := values[key]
			if !ok || depth > 8 {
				return "", false
			}
			resolved := true
			text = locRefRe.ReplaceAllStringFunc(text, func(ref string) string {
				inner, ok := resolve(ref[1:len(ref)-1], depth+1)
				if !ok {
					resolved = false
					return ref
				}
				return inner
			})
			return text, resolved
		}
		for _, key := range keys {

			if text, ok := resolve(key, 0); ok && strings.TrimSpace(text) != "" {
				if out[key] == nil {
					out[key] = map[string]string{}
				}
				out[key][lang[1]] = strings.TrimSpace(text)
			}
		}
	}
	return out
}

func unitTokens(g *gamefiles.Game, dir string, wrappers ...string) []string {
	set := map[string]bool{}
	for _, name := range g.Files(dir, false, ".txt") {
		if path.Dir(name) != dir {
			continue
		}
		for _, top := range readScript(g, name) {
			if top.Key == nil || top.Block == nil {
				continue
			}
			for _, w := range wrappers {
				if *top.Key != w {
					continue
				}
				for _, item := range top.Block.Entries {
					if item.Key != nil && item.Block != nil {
						set[*item.Key] = true
					}
				}
			}
		}
	}
	return sortedKeys(set)
}

func unitNameTokens(g *gamefiles.Game) (subunits, equipment []string) {
	return unitTokens(g, "common/units", "sub_units"),
		unitTokens(g, "common/units/equipment", "equipments", "duplicate_archetypes")
}

func buildGameNames(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	g := game(in)
	subunits, equipment := unitNameTokens(g)
	keys := append(append([]string{}, subunits...), equipment...)
	if raw, err := g.ReadFile(techSectionsGUI); err == nil {
		text := stripComments(pyText(raw))
		for _, span := range guiContainers(text) {
			for _, s := range techSubSections(text[span[0]:span[1]]) {
				keys = append(keys, s.TextKey)
			}
		}
	}
	names := locNames(ctx, g, sortedKeys(setOf(keys)))
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	return marshal(map[string]any{"game_version": in.GameVersion, "names": names})
}

func gameNamesOf(raw []byte) map[string]map[string]string {
	var f struct {
		Names map[string]map[string]string `json:"names"`
	}
	_ = json.Unmarshal(raw, &f)
	return f.Names
}

func buildUnitNames(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	g := game(in)
	subunits, equipment := unitNameTokens(g)
	if len(subunits) == 0 && len(equipment) == 0 {
		return nil, missingFile("common/units")
	}
	names := gameNamesOf(in.Dataset(gameindex.GameNames))
	table := func(tokens []string) *ordered {
		out := newOrdered()
		for _, token := range tokens {
			byLocale := names[token]
			if byLocale == nil {
				byLocale = map[string]string{}
			}
			out.set(token, byLocale)
		}
		return out
	}
	return marshal(map[string]any{"game_version": in.GameVersion, "subunits": table(subunits), "equipment": table(equipment)})
}

var (
	containerRe     = regexp.MustCompile(`containerWindowType\s*=\s*\{`)
	quotedNameRe    = regexp.MustCompile(`\bname\s*=\s*"([^"]+)"`)
	bareNameRe      = regexp.MustCompile(`\bname\s*=\s*([A-Za-z_][A-Za-z0-9_]*)`)
	textBoxRe       = regexp.MustCompile(`\binstantTextBoxType\s*=\s*\{`)
	subtitleNameRe  = regexp.MustCompile(`^(.+?)_subtitle_(.+)$`)
	positionYRe     = regexp.MustCompile(`\bposition\s*=\s*\{[^}]*\by\s*=\s*(-?\d+)`)
	quotedTextRe    = regexp.MustCompile(`\btext\s*=\s*"([^"]+)"`)
	bareTextKeyRe   = regexp.MustCompile(`\btext\s*=\s*([A-Z_][A-Z_0-9]*)`)
	firstIntegerRe  = regexp.MustCompile(`^-?\d+$`)
	techSectionsGUI = "interface/countrytechtreeview.gui"
)

type techSectionFolder struct {
	Count       int              `json:"count"`
	SubSections []techSubSection `json:"sub_sections"`
}

type techSubSection struct {
	Name    string            `json:"name"`
	SubKey  string            `json:"sub_key"`
	Y       int               `json:"y"`
	TextKey string            `json:"text_key"`
	Names   map[string]string `json:"names"`
}

func guiContainers(text string) map[string][2]int {
	containers := map[string][2]int{}
	for _, m := range containerRe.FindAllStringIndex(text, -1) {
		end, _ := walkBraces(text, m[1])
		body := text[m[1]:max(end-1, m[1])]
		head := body
		if len(head) > 600 {
			head = head[:600]
		}
		name, ok := guiName(head)
		if !ok {
			name = "_unnamed_"
		}
		if _, seen := containers[name]; !seen {
			containers[name] = [2]int{m[1], max(end-1, m[1])}
		}
	}
	return containers
}

func guiName(body string) (string, bool) {
	if m := quotedNameRe.FindStringSubmatch(body); m != nil {
		return m[1], true
	}
	if m := bareNameRe.FindStringSubmatch(body); m != nil {
		return m[1], true
	}
	return "", false
}

func techSubSections(body string) []techSubSection {
	var out []techSubSection
	for pos := 0; pos < len(body); {
		m := textBoxRe.FindStringIndex(body[pos:])
		if m == nil {
			break
		}
		start := pos + m[1]
		end, _ := walkBraces(body, start)
		box := body[start:max(end-1, start)]
		pos = end
		name, ok := guiName(box)
		if !ok {
			continue
		}
		sub := subtitleNameRe.FindStringSubmatch(name)
		if sub == nil {
			continue
		}
		y := positionYRe.FindStringSubmatch(box)
		text := quotedTextRe.FindStringSubmatch(box)
		if text == nil {
			text = bareTextKeyRe.FindStringSubmatch(box)
		}
		if y == nil || text == nil || !firstIntegerRe.MatchString(y[1]) {
			continue
		}
		n, _ := atoiSigned(y[1])
		out = append(out, techSubSection{Name: name, SubKey: sub[2], Y: n, TextKey: text[1]})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Y < out[j].Y })
	return out
}

func atoiSigned(s string) (int, bool) {
	if strings.HasPrefix(s, "-") {
		n, ok := atoiPrefix(s[1:])
		return -n, ok
	}
	return atoiPrefix(s)
}

func buildTechSections(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	g := game(in)
	raw, err := g.ReadFile(techSectionsGUI)
	if err != nil {
		return nil, err
	}
	text := stripComments(pyText(raw))
	containers := guiContainers(text)
	folders := techTagsOf(in.Dataset(gameindex.TechTags)).Folders.IDs
	result := map[string]*techSectionFolder{}
	for _, folder := range folders {
		span, ok := containers[folder]
		if !ok {
			continue
		}
		subs := techSubSections(text[span[0]:span[1]])
		if subs == nil {
			subs = []techSubSection{}
		}
		result[folder] = &techSectionFolder{Count: len(subs), SubSections: subs}
	}
	names := gameNamesOf(in.Dataset(gameindex.GameNames))
	for _, f := range result {
		for i := range f.SubSections {
			byLocale := map[string]string{}
			for locale, value := range names[f.SubSections[i].TextKey] {
				byLocale[locale] = strings.ReplaceAll(value, "\n", " ")
			}
			f.SubSections[i].Names = byLocale
		}
	}
	out := newOrdered()
	for _, folder := range sortedKeys(result) {
		out.set(folder, result[folder])
	}
	return marshal(map[string]any{"game_version": in.GameVersion, "folders": out})
}

var utf8BOM = []byte{0xef, 0xbb, 0xbf}
