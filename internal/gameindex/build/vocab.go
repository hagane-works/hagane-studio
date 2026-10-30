package build

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/exportengine"
)

//go:embed structural_fields_product.json
var structuralProductJSON []byte

//go:embed cwtool_rules.json
var cwtoolRulesJSON []byte

func emittableToken(token string) bool {
	if token == "" {
		return false
	}
	for i, r := range token {
		if i == 0 && !(unicode.IsLetter(r) || r == '_') {
			return false
		}
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return false
		}
	}
	return true
}

func buildStructuralFields(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	product, err := decodeOrdered(structuralProductJSON)
	if err != nil {
		return nil, err
	}
	var blacklist []string
	_ = json.Unmarshal(product.raw("internal_blacklist"), &blacklist)
	black := map[string]bool{}
	for _, b := range blacklist {
		black[b] = true
	}
	byEntity, _ := product.values["by_entity"].(*ordered)
	if byEntity == nil {
		return nil, fmt.Errorf("structural_fields_product.json has no by_entity")
	}
	var modifiers struct {
		Modifiers []string `json:"modifiers"`
	}
	_ = json.Unmarshal(in.Dataset(gameindex.VanillaModifiers), &modifiers)
	equipment := equipmentOf(in.Dataset(gameindex.Equipment))
	archetypes := map[string]bool{}
	for tech, picture := range equipment.Pictures {
		archetypes[tech] = true
		if picture != "" {
			archetypes[picture] = true
		}
	}
	families := [][2]any{
		{"vanilla:vanilla_modifiers", setOf(modifiers.Modifiers)},
		{"vanilla:vanilla_archetypes", archetypes},
		{"vanilla:vanilla_tech_folders", setOf(techTagsOf(in.Dataset(gameindex.TechTags)).Folders.IDs)},
	}
	counts := newOrdered()
	for _, family := range families {
		name, set := family[0].(string), family[1].(map[string]bool)
		var kept []string
		for token := range set {
			if black[token] || !emittableToken(token) || strings.HasSuffix(token, "_at") || strings.HasSuffix(token, "_json") {
				continue
			}
			kept = append(kept, token)
		}
		counts.set(strings.TrimPrefix(name, "vanilla:"), len(set))
		if len(kept) == 0 {
			continue
		}
		sort.Strings(kept)
		byEntity.set(name, kept)
	}
	union := map[string]bool{}
	for _, key := range byEntity.keys {
		var fields []string
		if list, ok := byEntity.values[key].([]string); ok {
			fields = list
		} else {
			_ = json.Unmarshal(mustJSON(byEntity.values[key]), &fields)
		}
		for _, f := range fields {
			union[f] = true
		}
	}
	meta, _ := product.values["_meta"].(*ordered)
	if meta == nil {
		meta = newOrdered()
	}
	meta.set("vanilla_external_counts", counts)
	meta.set("total_unique_fields", len(union))
	meta.set("game_version", in.GameVersion)
	out := newOrdered()
	out.set("_meta", meta)
	out.set("fields", sortedKeys(union))
	out.set("by_entity", byEntity)
	return marshal(out)
}

func setOf(list []string) map[string]bool {
	out := map[string]bool{}
	for _, s := range list {
		if s != "" {
			out[s] = true
		}
	}
	return out
}

func mustJSON(v any) []byte {
	raw, _ := marshalCompact(v)
	return raw
}

func buildYesNoKeys(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	known := map[string]bool{}
	for _, name := range readIDs(in.Dataset(gameindex.VanillaEffects)) {
		known[name] = true
	}
	for _, name := range readIDs(in.Dataset(gameindex.VanillaTriggers)) {
		known[name] = true
	}
	var structural struct {
		Fields []string `json:"fields"`
	}
	_ = json.Unmarshal(in.Dataset(gameindex.StructuralFields), &structural)
	for _, name := range structural.Fields {
		known[name] = true
	}
	keys := map[string]bool{}
	files := 0
	for _, dir := range []string{"common", "events", "history"} {
		for _, rel := range walkFiles(in.FS, dir, ".txt") {
			raw, err := readRaw(in, rel)
			if err != nil {
				continue
			}
			if exportengine.VanillaYesNoKeysOf(raw, func(k string) bool { return known[k] }, keys) {
				files++
			}
		}
		if err := checkCtx(ctx); err != nil {
			return nil, err
		}
	}
	if files == 0 {
		return nil, missingFile("common/")
	}
	list := sortedKeys(keys)
	return marshal(map[string]any{
		"game_version": shortVersion(in.GameVersion),
		"ids":          list,
		"rule":         "every key K of a scalar `K = yes|no` entry anywhere in the game's own script that passes raw scripted-token predicate",
		"source":       "the game's common/ + events/ + history/ *.txt",
		"total":        len(list),
	})
}

var eventPictureRe = regexp.MustCompile(`name\s*=\s*"?(GFX_(?:news|report)_event_[A-Za-z0-9_\-]+)"?`)

func buildEventPictures(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	news, report := map[string]bool{}, map[string]bool{}
	for _, rel := range baseFiles(in, "interface", ".gfx") {
		base := strings.ToLower(rel[strings.LastIndexByte(rel, '/')+1:])
		if !strings.HasPrefix(base, "eventpictures") && base != "news_event_pictures.gfx" {
			continue
		}
		text, _ := readBase(in, rel)
		for _, m := range eventPictureRe.FindAllStringSubmatch(text, -1) {
			if strings.HasPrefix(m[1], "GFX_news_event_") {
				news[m[1]] = true
			} else {
				report[m[1]] = true
			}
		}
	}
	all := map[string]bool{}
	for k := range news {
		all[k] = true
	}
	for k := range report {
		all[k] = true
	}
	return marshal(map[string]any{
		"source":       "interface/eventpictures*.gfx + news_event_pictures.gfx",
		"game_version": in.GameVersion,
		"total":        len(all),
		"by_prefix":    map[string][]string{"news": sortedKeys(news), "report": sortedKeys(report)},
		"all_pictures": sortedKeys(all),
	})
}

func CwtoolRulesWithoutGame() ([]byte, error) {
	return buildCwtoolWhitelist(context.Background(), &gameindex.Input{})
}

var cwtoolMathOwned = map[string]bool{"add": true, "if": true, "else": true}

func buildCwtoolWhitelist(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	rules, err := decodeOrdered(cwtoolRulesJSON)
	if err != nil {
		return nil, err
	}
	mathIDs, mathSub := mathFunctions(in.Dataset(gameindex.MathFunctions))
	sub := setOf(mathSub)
	mathWords := map[string]bool{}
	for _, id := range mathIDs {
		if !sub[id] && !cwtoolMathOwned[id] {
			mathWords[id] = true
		}
	}
	effects := setOf(readIDs(in.Dataset(gameindex.VanillaEffects)))
	triggers := setOf(readIDs(in.Dataset(gameindex.VanillaTriggers)))
	for w := range mathWords {
		effects[w] = true
		triggers[w] = true
	}
	if enums, ok := rules.values["trigger_enums"].(*ordered); ok {
		for _, key := range enums.keys {
			triggers[key] = true
		}
	}
	var modifiers struct {
		Modifiers []string `json:"modifiers"`
	}
	_ = json.Unmarshal(in.Dataset(gameindex.VanillaModifiers), &modifiers)

	var ideaTags struct {
		All []string `json:"all_categories"`
	}
	_ = json.Unmarshal(in.Dataset(gameindex.IdeaTags), &ideaTags)
	tags := techTagsOf(in.Dataset(gameindex.TechTags))
	var pictures struct {
		All []string `json:"all_pictures"`
	}
	_ = json.Unmarshal(in.Dataset(gameindex.EventPictures), &pictures)
	var traits traitsFile
	_ = json.Unmarshal(in.Dataset(gameindex.Traits), &traits)
	traitTypes := map[string]bool{}
	for _, meta := range traits.MetadataByType["unit_leader"] {
		for _, t := range meta["trait_types"] {
			traitTypes[t] = true
		}
	}
	lists := map[string][]string{
		"idea_categories":         ideaTags.All,
		"decision_categories":     readIDs(in.Dataset(gameindex.DecisionCategories)),
		"tech_categories":         tags.Categories.IDs,
		"tech_folders":            tags.Folders.IDs,
		"equipment_archetypes":    equipmentOf(in.Dataset(gameindex.Equipment)).Archetypes,
		"event_pictures":          pictures.All,
		"unit_leader_trait_types": sortedKeys(traitTypes),
		"country_leader_traits":   traits.IDsByType["country_leader"],
	}
	if enums, ok := rules.values["field_enums"].(*ordered); ok {
		for _, name := range enums.keys {
			enum, ok := enums.values[name].(*ordered)
			if !ok {
				continue
			}
			source, isName := enum.values["whitelist"].(string)
			if !isName {
				continue
			}
			list := append([]string{}, lists[source]...)
			sort.Strings(list)
			enum.set("whitelist", list)
		}
	}
	out := newOrdered()
	for _, key := range rules.keys {
		out.set(key, rules.values[key])
		if key == "modifier_framework_keys" {
			out.set("known_effects", sortedKeys(effects))
			out.set("known_triggers", sortedKeys(triggers))
			out.set("known_modifiers", sortedKeys(setOf(modifiers.Modifiers)))
		}
	}
	return marshal(out)
}

func decodeOrdered(raw []byte) (*ordered, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	o, ok := v.(*ordered)
	if !ok {
		return nil, fmt.Errorf("not a JSON object")
	}
	return o, nil
}

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			o := newOrdered()
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, _ := keyTok.(string)
				v, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				o.set(key, v)
			}
			_, err := dec.Token()
			return o, err
		case '[':
			list := []any{}
			for dec.More() {
				v, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				list = append(list, v)
			}
			_, err := dec.Token()
			return list, err
		}
	}
	return tok, nil
}

func (o *ordered) raw(key string) []byte { return mustJSON(o.values[key]) }
