package build

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"haganeworks/internal/gamefiles"
	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/paradox"
)

func Datasets() []gameindex.Dataset {
	return []gameindex.Dataset{

		{Name: gameindex.VanillaEffects, Version: 1, Build: buildVanillaEffects},
		{Name: gameindex.VanillaTriggers, Version: 1, Build: buildVanillaTriggers},
		{Name: gameindex.VanillaModifiers, Version: 1, Build: buildVanillaModifiers},
		{Name: gameindex.MathFunctions, Version: 1, Build: buildMathFunctions},

		{Name: gameindex.Traits, Version: 1, Build: buildTraits},
		{Name: gameindex.CharacterIDs, Version: 1, Build: buildCharacterIDs},
		{Name: gameindex.OpinionModifiers, Version: 1, Build: buildOpinionModifiers},
		{Name: gameindex.DynamicModifierIDs, Version: 1, Build: buildDynamicModifierIDs},
		{Name: gameindex.DecisionCategories, Version: 1, Build: buildDecisionCategories},
		{Name: gameindex.TechIDs, Version: 1, Build: buildTechIDs},
		{Name: gameindex.EventIDs, Version: 1, Build: buildEventIDs},
		{Name: gameindex.AutonomyStates, Version: 2, Build: buildAutonomyStates},
		{Name: gameindex.UnitFiles, Version: 1, Build: buildUnitFiles},
		{Name: gameindex.IdeaIndex, Version: 1, Build: buildIdeaIndex},
		{Name: gameindex.CrossRefIDs, Version: 1, Build: buildCrossRefIDs, Deps: []string{
			gameindex.IdeaIndex, gameindex.EventIDs, gameindex.Traits, gameindex.TechIDs,
			gameindex.CharacterIDs, gameindex.OpinionModifiers, gameindex.DynamicModifierIDs}},

		{Name: gameindex.TechTags, Version: 1, Build: buildTechTags},
		{Name: gameindex.Equipment, Version: 1, Build: buildEquipment},
		{Name: gameindex.TechIconPaths, Version: 1, Build: buildTechIconPaths},
		{Name: gameindex.TechIconSizes, Version: 1, Build: buildTechIconSizes},

		{Name: gameindex.FocusIcons, Version: 1, Build: buildFocusIcons},
		{Name: gameindex.FocusTrees, Version: 2, Build: buildFocusTrees},
		{Name: gameindex.IDCollisions, Version: 1, Build: buildIDCollisions, Deps: []string{
			gameindex.TechIDs, gameindex.FocusTrees, gameindex.DecisionCategories, gameindex.DynamicModifierIDs}},
		{Name: gameindex.CollisionTable, Version: 1, Build: buildCollisionTable, Deps: []string{
			gameindex.FocusIcons, gameindex.TechIconPaths}},

		{Name: gameindex.CountryAliases, Version: 2, Build: buildCountryAliases},
		{Name: gameindex.WorldTables, Version: 1, Build: buildWorldTables, Deps: []string{gameindex.CountryAliases}},
		{Name: gameindex.Cosmetic, Version: 1, Build: buildCosmetic},
		{Name: gameindex.CountryTags, Version: 1, Build: buildCountryTags, Deps: []string{gameindex.WorldTables, gameindex.CountryAliases}},
		{Name: gameindex.DLCNames, Version: 1, Build: buildDLCNames},
		{Name: gameindex.StateContent, Version: 1, Build: buildStateContent},
		{Name: gameindex.DLCStates, Version: 1, Build: buildDLCStates},

		{Name: gameindex.GameNames, Version: 2, Build: buildGameNames},
		{Name: gameindex.TechSections, Version: 2, Build: buildTechSections, Deps: []string{gameindex.TechTags, gameindex.GameNames}},
		{Name: gameindex.UnitNames, Version: 2, Build: buildUnitNames, Deps: []string{gameindex.GameNames}},

		{Name: gameindex.IdeaTags, Version: 1, Build: buildIdeaTags, Deps: []string{gameindex.IdeaIndex}},
		{Name: gameindex.IdeasUI, Version: 1, Build: buildIdeasUI, Deps: []string{gameindex.IdeaIndex, gameindex.CountryAliases}},

		{Name: gameindex.StructuralFields, Version: 1, Build: buildStructuralFields, Deps: []string{
			gameindex.VanillaModifiers, gameindex.Equipment, gameindex.TechTags}},
		{Name: gameindex.YesNoKeys, Version: 1, Build: buildYesNoKeys, Deps: []string{
			gameindex.VanillaEffects, gameindex.VanillaTriggers, gameindex.StructuralFields}},
		{Name: gameindex.EventPictures, Version: 1, Build: buildEventPictures},
		{Name: gameindex.CwtoolWhitelist, Version: 1, Build: buildCwtoolWhitelist, Deps: []string{
			gameindex.VanillaEffects, gameindex.VanillaTriggers, gameindex.VanillaModifiers, gameindex.MathFunctions,
			gameindex.IdeaTags, gameindex.DecisionCategories, gameindex.TechTags, gameindex.Equipment,
			gameindex.EventPictures, gameindex.Traits}},
	}
}

func game(in *gameindex.Input) *gamefiles.Game { return gamefiles.For(in.FS) }

func marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func pyText(raw []byte) string {
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	text := string(raw)
	if !utf8.ValidString(text) {
		var b strings.Builder
		for len(raw) > 0 {
			r, size := utf8.DecodeRune(raw)
			b.WriteRune(r)
			raw = raw[size:]
		}
		text = b.String()
	}
	if strings.Contains(text, "\r") {
		text = strings.ReplaceAll(text, "\r\n", "\n")
		text = strings.ReplaceAll(text, "\r", "\n")
	}
	return text
}

func readBase(in *gameindex.Input, rel string) (string, bool) {
	raw, err := fs.ReadFile(in.FS, rel)
	if err != nil {
		return "", false
	}
	return pyText(raw), true
}

func baseFiles(in *gameindex.Input, dir, ext string) []string {
	return listFiles(in.FS, dir, ext)
}

func listFiles(fsys fs.FS, dir, ext string) []string {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ext) {
			continue
		}
		out = append(out, path.Join(dir, entry.Name()))
	}
	sortFold(out)
	return out
}

func sortFold(names []string) {
	sort.SliceStable(names, func(i, j int) bool {
		a, b := strings.ToLower(names[i]), strings.ToLower(names[j])
		if a != b {
			return a < b
		}
		return names[i] < names[j]
	})
}

func layerDirs(fsys fs.FS, groups ...string) []string {
	if len(groups) == 0 {
		groups = []string{"dlc", "integrated_dlc"}
	}
	var out []string
	for _, group := range groups {
		entries, err := fs.ReadDir(fsys, group)
		if err != nil {
			continue
		}
		var names []string
		for _, entry := range entries {
			if entry.IsDir() {
				names = append(names, path.Join(group, entry.Name()))
			}
		}
		sortFold(names)
		out = append(out, names...)
	}
	return out
}

func walkFiles(fsys fs.FS, dir, ext string) []string {
	var out []string
	_ = fs.WalkDir(fsys, dir, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(name), ext) {
			out = append(out, name)
		}
		return nil
	})
	sortFold(out)
	return out
}

func parseEntries(text string) []*paradox.Entry {
	block, _, err := paradox.Parse(text)
	if err != nil || block == nil {
		return nil
	}
	return block.Entries
}

func readScript(g *gamefiles.Game, name string) []*paradox.Entry {
	raw, err := g.ReadFile(name)
	if err != nil {
		return nil
	}
	text, ok := gfx.DecodeScriptText(raw)
	if !ok {
		return nil
	}
	return parseEntries(text)
}

func keyIs(entry *paradox.Entry, key string) bool {
	return entry != nil && entry.Key != nil && *entry.Key == key
}

func childBlock(entries []*paradox.Entry, key string) []*paradox.Entry {
	for _, entry := range entries {
		if keyIs(entry, key) && entry.Block != nil {
			return entry.Block.Entries
		}
	}
	return nil
}

func scalarField(entries []*paradox.Entry, key string) string {
	for _, entry := range entries {
		if keyIs(entry, key) && entry.Block == nil {
			return entry.Scalar.StrValue()
		}
	}
	return ""
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

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\v' || c == '\f'
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

func sortedKeys[V any](set map[string]V) []string {
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

type idSet struct {
	Source      string   `json:"source"`
	GameVersion string   `json:"game_version"`
	Total       int      `json:"total"`
	IDs         []string `json:"ids"`
}

func idSetJSON(in *gameindex.Input, source string, ids map[string]bool) ([]byte, error) {
	list := sortedKeys(ids)
	return marshal(idSet{Source: source, GameVersion: in.GameVersion, Total: len(list), IDs: list})
}

func readIDs(raw []byte) []string {
	var set idSet
	_ = json.Unmarshal(raw, &set)
	return set.IDs
}

func shortVersion(raw string) string {
	parts := strings.Split(raw, ".")
	if len(parts) > 3 {
		parts = parts[:3]
	}
	return strings.Join(parts, ".")
}

func checkCtx(ctx context.Context) error { return ctx.Err() }

type ordered struct {
	keys   []string
	values map[string]any
}

func newOrdered() *ordered { return &ordered{values: map[string]any{}} }

func (o *ordered) set(key string, v any) {
	if _, ok := o.values[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = v
}

func (o *ordered) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, key := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		k, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(k)
		buf.WriteByte(':')
		v, err := marshalCompact(o.values[key])
		if err != nil {
			return nil, err
		}
		buf.Write(v)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func marshalCompact(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func readRaw(in *gameindex.Input, rel string) ([]byte, error) { return fs.ReadFile(in.FS, rel) }

func mustBase(in *gameindex.Input, rel string) string {
	text, _ := readBase(in, rel)
	return text
}
