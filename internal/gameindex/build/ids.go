package build

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/paradox"
)

func missingFile(rel string) error {
	return fmt.Errorf("%s is missing from the game folder", rel)
}

var topIDRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\{`)

func topLevelIDs(text string) []string {
	body := stripComments(text)
	var out []string
	pos := 0
	for pos < len(body) {
		for pos < len(body) && isSpace(body[pos]) {
			pos++
		}
		if pos >= len(body) {
			break
		}
		m := topIDRe.FindStringSubmatchIndex(body[pos:])
		if m == nil {
			if nl := strings.IndexByte(body[pos:], '\n'); nl >= 0 {
				pos += nl + 1
			} else {
				pos = len(body)
			}
			continue
		}
		out = append(out, body[pos+m[2]:pos+m[3]])
		pos, _ = walkBraces(body, pos+m[1])
	}
	return out
}

func baseDirIDs(in *gameindex.Input, dir string) map[string]bool {
	ids := map[string]bool{}
	for _, name := range baseFiles(in, dir, ".txt") {
		text, _ := readBase(in, name)
		for _, id := range topLevelIDs(text) {
			ids[id] = true
		}
	}
	return ids
}

func buildDynamicModifierIDs(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	return idSetJSON(in, "common/dynamic_modifiers/*.txt", baseDirIDs(in, "common/dynamic_modifiers"))
}

func buildDecisionCategories(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	return idSetJSON(in, "common/decisions/categories/*.txt", baseDirIDs(in, "common/decisions/categories"))
}

func buildTechIDs(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	ids := map[string]bool{}
	for _, name := range baseFiles(in, "common/technologies", ".txt") {
		text, _ := readBase(in, name)
		body, ok := topBlock(text, "technologies")
		if !ok {
			continue
		}
		for _, id := range techIDsOfBlock(body) {
			ids[id] = true
		}
	}
	return idSetJSON(in, "common/technologies/*.txt", ids)
}

func topBlock(text, key string) (string, bool) {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(key) + `\s*=\s*\{`)
	m := re.FindStringIndex(text)
	if m == nil {
		return "", false
	}
	end, _ := walkBraces(text, m[1])
	return text[m[1]:max(end-1, m[1])], true
}

var subTechRe = regexp.MustCompile(`\bsub_technologies\s*=\s*\{`)

func techIDsOfBlock(body string) []string {
	body = stripComments(body)
	var out []string
	pos := 0
	for pos < len(body) {
		for pos < len(body) && (isSpace(body[pos]) || body[pos] == '@') {
			if body[pos] == '@' {
				nl := strings.IndexByte(body[pos:], '\n')
				if nl < 0 {
					pos = len(body)
					break
				}
				pos += nl + 1
			} else {
				pos++
			}
		}
		if pos >= len(body) {
			break
		}
		m := topIDRe.FindStringSubmatchIndex(body[pos:])
		if m == nil {
			if nl := strings.IndexByte(body[pos:], '\n'); nl >= 0 {
				pos += nl + 1
			} else {
				pos = len(body)
			}
			continue
		}
		ident := body[pos+m[2] : pos+m[3]]
		start := pos + m[1]
		end, _ := walkBraces(body, start)
		sub := body[start:max(end-1, start)]
		if ident != "sub_technologies" {
			out = append(out, ident)
		}
		if loc := subTechRe.FindStringIndex(sub); loc != nil {
			subEnd, _ := walkBraces(sub, loc[1])
			out = append(out, techIDsOfBlock(sub[loc[1]:max(subEnd-1, loc[1])])...)
		}
		pos = end
	}
	return out
}

var (
	charAssignRe   = regexp.MustCompile(`^([A-Za-z0-9_.\-]+)\s*=\s*\{`)
	charactersOpen = regexp.MustCompile(`^characters\s*=\s*\{`)
)

func buildCharacterIDs(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	ids := map[string]bool{}
	for _, name := range baseFiles(in, "common/characters", ".txt") {
		text, _ := readBase(in, name)
		depth, inCharacters, charactersDepth := 0, false, -1
		for _, raw := range strings.Split(text, "\n") {
			line := raw
			if i := strings.IndexByte(line, '#'); i >= 0 {
				line = line[:i]
			}
			stripped := strings.TrimSpace(line)
			if stripped == "" {
				continue
			}
			delta := strings.Count(stripped, "{") - strings.Count(stripped, "}")
			if !inCharacters {
				if charactersOpen.MatchString(stripped) {
					inCharacters, charactersDepth = true, depth
				}
				depth += delta
				continue
			}
			if depth == charactersDepth+1 {
				if m := charAssignRe.FindStringSubmatch(stripped); m != nil {
					ids[m[1]] = true
				}
			}
			depth += delta
			if depth <= charactersDepth {
				inCharacters, charactersDepth = false, -1
			}
		}
	}
	return idSetJSON(in, "common/characters/*.txt", ids)
}

var (
	opinionBlockRe = regexp.MustCompile(`opinion_modifiers\s*=\s*\{`)
	identBlockRe   = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\{`)
)

func buildOpinionModifiers(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	ids := map[string]bool{}
	for _, name := range baseFiles(in, "common/opinion_modifiers", ".txt") {
		text, _ := readBase(in, name)
		text = stripComments(text)
		m := opinionBlockRe.FindStringIndex(text)
		if m == nil {
			continue
		}
		end, _ := walkBraces(text, m[1])
		body := text[m[1]:max(end-1, m[1])]
		pos := 0
		for pos < len(body) {
			sub := identBlockRe.FindStringSubmatchIndex(body[pos:])
			if sub == nil {
				break
			}
			ids[body[pos+sub[2]:pos+sub[3]]] = true
			pos, _ = walkBraces(body, pos+sub[1])
		}
	}
	return idSetJSON(in, "common/opinion_modifiers/*.txt", ids)
}

type traitsFile struct {
	Source         string                                    `json:"source"`
	GameVersion    string                                    `json:"game_version"`
	Total          int                                       `json:"total"`
	ByType         map[string]int                            `json:"by_type"`
	IDs            []string                                  `json:"ids"`
	IDsByType      map[string][]string                       `json:"ids_by_type"`
	MetadataByType map[string]map[string]map[string][]string `json:"metadata_by_type"`
}

func buildTraits(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	sources := [][2]string{
		{"common/country_leader", "country_leader"},
		{"common/unit_leader", "unit_leader"},
		{"common/scientist_traits", "scientist"},
	}
	byType := map[string]map[string]bool{"country_leader": {}, "unit_leader": {}, "scientist": {}}
	meta := map[string]map[string]map[string]bool{}
	for _, source := range sources {
		dir, kind := source[0], source[1]
		for _, name := range baseFiles(in, dir, ".txt") {
			if kind == "unit_leader" && strings.Contains(path.Base(name), "_skills") {
				continue
			}
			text, _ := readBase(in, name)
			entries := parseEntries(text)
			collect := func(children []*paradox.Entry) {
				for _, e := range children {
					if e.Key == nil || e.Block == nil {
						continue
					}
					key := *e.Key
					byType[kind][key] = true
					if kind != "unit_leader" {
						continue
					}
					m := meta[key]
					if m == nil {
						m = map[string]map[string]bool{"types": {}, "trait_types": {}}
						meta[key] = m
					}
					for _, child := range e.Block.Entries {
						if child.Key == nil || (*child.Key != "type" && *child.Key != "trait_type") {
							continue
						}
						target := "types"
						if *child.Key == "trait_type" {
							target = "trait_types"
						}
						var values []string
						if child.Block != nil {
							for _, item := range child.Block.Entries {
								if item.Key == nil && item.Block == nil {
									values = append(values, item.Scalar.StrValue())
								}
							}
						} else {
							values = []string{child.Scalar.StrValue()}
						}
						for _, v := range values {
							if v = strings.TrimSpace(v); v != "" {
								m[target][v] = true
							}
						}
					}
				}
			}
			wrapped := false
			for _, e := range entries {
				if e.Key == nil || e.Block == nil {
					continue
				}
				switch *e.Key {
				case "leader_traits", "scientist_traits", "unit_leader_traits":
					collect(e.Block.Entries)
					wrapped = true
				}
			}
			if !wrapped {
				collect(entries)
			}
		}
	}
	out := traitsFile{
		Source: "common/{country_leader,unit_leader,scientist_traits}", GameVersion: in.GameVersion,
		ByType: map[string]int{}, IDsByType: map[string][]string{},
		MetadataByType: map[string]map[string]map[string][]string{"unit_leader": {}},
	}
	merged := map[string]bool{}
	for kind, ids := range byType {
		out.IDsByType[kind] = sortedKeys(ids)
		out.ByType[kind] = len(ids)
		for id := range ids {
			merged[id] = true
		}
	}
	out.IDs = sortedKeys(merged)
	out.Total = len(out.IDs)
	for trait, m := range meta {
		out.MetadataByType["unit_leader"][trait] = map[string][]string{
			"types": sortedKeys(m["types"]), "trait_types": sortedKeys(m["trait_types"]),
		}
	}
	return marshal(out)
}

func buildEventIDs(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	ids := map[string]bool{}
	for _, name := range baseFiles(in, "events", ".txt") {
		text, _ := readBase(in, name)
		if !addNSRe.MatchString(text) {
			continue
		}
		for i := 0; ; {
			if i > 0 && text[i-1] != '\n' {
				nl := strings.IndexByte(text[i:], '\n')
				if nl < 0 {
					break
				}
				i += nl + 1
			}
			loc := eventTypeRe.FindStringIndex(text[i:])
			if loc == nil {
				break
			}
			end := i + loc[1]
			j, closed := walkBraces(text, end)
			if !closed {
				break
			}
			i = j
			if m := eventIDRe.FindStringSubmatch(text[end : j-1]); m != nil {
				ids[m[1]] = true
			}
		}
	}
	return idSetJSON(in, "events/*.txt", ids)
}

var (
	eventTypeRe = regexp.MustCompile(`(?m)^\s*(country_event|news_event|state_event|unit_leader_event|operative_leader_event)\s*=\s*\{`)
	addNSRe     = regexp.MustCompile(`(?m)^\s*add_namespace\s*=\s*([\w.]+)`)
	eventIDRe   = regexp.MustCompile(`(?m)^\s*id\s*=\s*([\w.\-]+)`)
)

func buildAutonomyStates(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	g := game(in)
	ids := map[string]bool{"autonomy_free": true}
	for _, name := range g.Files("common/autonomous_states", false, ".txt") {
		for _, entry := range readScript(g, name) {
			if keyIs(entry, "autonomy_state") && entry.Block != nil {
				if id := strings.TrimSpace(scalarField(entry.Block.Entries, "id")); id != "" {
					ids[id] = true
				}
			}
		}
	}
	return idSetJSON(in, "common/autonomous_states/*.txt", ids)
}

func buildUnitFiles(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	ids := map[string]bool{}
	for _, name := range game(in).Files("common/units", false, ".txt") {
		ids[path.Base(name)] = true
	}
	return idSetJSON(in, "common/units/*.txt", ids)
}

func buildCrossRefIDs(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	var ideas struct {
		Ideas []struct {
			IdeaID string `json:"idea_id"`
		} `json:"ideas"`
	}
	_ = json.Unmarshal(in.Dataset(gameindex.IdeaIndex), &ideas)
	ideaIDs := map[string]bool{}
	for _, idea := range ideas.Ideas {
		if idea.IdeaID != "" {
			ideaIDs[idea.IdeaID] = true
		}
	}
	var traits traitsFile
	_ = json.Unmarshal(in.Dataset(gameindex.Traits), &traits)
	ids := map[string][]string{
		"idea":             sortedKeys(ideaIDs),
		"event":            readIDs(in.Dataset(gameindex.EventIDs)),
		"trait":            traits.IDs,
		"technology":       readIDs(in.Dataset(gameindex.TechIDs)),
		"character":        readIDs(in.Dataset(gameindex.CharacterIDs)),
		"opinion_modifier": readIDs(in.Dataset(gameindex.OpinionModifiers)),
		"dynamic_modifier": readIDs(in.Dataset(gameindex.DynamicModifierIDs)),
	}
	for kind, list := range ids {
		if list == nil {
			ids[kind] = []string{}
		}
	}
	return marshal(map[string]any{
		"source": "ids of the base game's ideas, events, traits, technologies, characters, opinion modifiers and dynamic modifiers · game " + in.GameVersion,
		"ids":    ids,
	})
}

func buildIDCollisions(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	var trees struct {
		IDs []string `json:"ids"`
	}
	_ = json.Unmarshal(in.Dataset(gameindex.FocusTrees), &trees)
	orEmpty := func(list []string) []string {
		if list == nil {
			return []string{}
		}
		return list
	}
	return marshal(map[string]any{
		"extracted_at":          time.Now().Format("2006-01-02"),
		"tech_ids":              orEmpty(readIDs(in.Dataset(gameindex.TechIDs))),
		"focus_tree_ids":        orEmpty(trees.IDs),
		"decision_category_ids": orEmpty(readIDs(in.Dataset(gameindex.DecisionCategories))),
		"dynamic_modifier_ids":  orEmpty(readIDs(in.Dataset(gameindex.DynamicModifierIDs))),
	})
}

func atoiPrefix(s string) (int, bool) {
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(s[:end])
	return n, err == nil
}
