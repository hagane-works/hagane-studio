package characters

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type TraitOverlayRows struct {
	Project         json.RawMessage
	Characters      []json.RawMessage
	CustomTraits    []json.RawMessage
	EventNamespaces []json.RawMessage
	EntityOBSPaths  []json.RawMessage
	IncludeImported bool
	VanillaTraits   map[string]bool

	OBSAvailable func(key string) (bool, error)
}

func rawObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func rawString(obj map[string]json.RawMessage, key string) string {
	var out string
	_ = json.Unmarshal(obj[key], &out)
	return out
}

func rawBool(obj map[string]json.RawMessage, key string) bool {
	var out bool
	_ = json.Unmarshal(obj[key], &out)
	return out
}

func rawInt(obj map[string]json.RawMessage, key string) int {
	var out int
	_ = json.Unmarshal(obj[key], &out)
	return out
}

func rawStrings(obj map[string]json.RawMessage, key string) []string {
	var out []string
	_ = json.Unmarshal(obj[key], &out)
	return out
}

func rawKVs(obj map[string]json.RawMessage, key string) []KV {
	var values map[string]any
	if err := json.Unmarshal(obj[key], &values); err != nil {
		return nil
	}
	keys := make([]string, 0, len(values))
	for name := range values {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	out := make([]KV, 0, len(keys))
	for _, name := range keys {
		out = append(out, KV{Key: name, Val: values[name]})
	}
	return out
}

func decodeOverlayCharacter(raw json.RawMessage) (*Character, string, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, "", err
	}
	return &Character{
		ID:                 rawInt(obj, "id"),
		Tag:                rawString(obj, "tag"),
		Name:               rawString(obj, "name"),
		NameEn:             rawString(obj, "name_en"),
		NameZh:             rawString(obj, "name_zh"),
		DescEn:             rawString(obj, "desc_en"),
		DescZh:             rawString(obj, "desc_zh"),
		DisplayNameLocs:    rawKVs(obj, "display_name_locs"),
		DescriptionLocs:    rawKVs(obj, "description_locs"),
		CountryTag:         rawString(obj, "country_tag"),
		RoleType:           rawString(obj, "role_type"),
		HasAdvisorRole:     rawBool(obj, "has_advisor_role"),
		Traits:             rawStrings(obj, "traits"),
		AdvisorTraits:      rawStrings(obj, "advisor_traits"),
		Ideology:           rawString(obj, "ideology"),
		Expire:             rawString(obj, "expire"),
		RawExtras:          rawKVs(obj, "raw_extras"),
		SourceFile:         rawString(obj, "source_file"),
		ReplacesVanillaID:  rawBool(obj, "replaces_vanilla_id"),
		IsPortraitModified: rawBool(obj, "is_portrait_modified"),
	}, rawString(obj, "source"), nil
}

var overlayLangRegistry = map[string][2]string{
	"en": {"english/replace", "l_english"},
	"zh": {"simp_chinese/replace", "l_simp_chinese"},
	"ru": {"russian/replace", "l_russian"},
	"fr": {"french/replace", "l_french"},
	"de": {"german/replace", "l_german"},
	"es": {"spanish/replace", "l_spanish"},
	"ja": {"japanese/replace", "l_japanese"},
	"ko": {"korean/replace", "l_korean"},
	"pl": {"polish/replace", "l_polish"},
	"pt": {"braz_por/replace", "l_braz_por"},
}

func overlayLanguages(codes []string) []LocLanguage {
	if len(codes) == 0 {
		codes = []string{"en", "zh"}
	}
	ordered := append([]string{"en"}, codes...)
	seen := map[string]bool{}
	var out []LocLanguage
	for _, code := range ordered {
		code = strings.ToLower(strings.TrimSpace(code))
		if seen[code] {
			continue
		}
		dest, ok := overlayLangRegistry[code]
		if !ok {
			continue
		}
		seen[code] = true
		out = append(out, LocLanguage{Code: code, Subdir: dest[0], Suffix: dest[1]})
	}
	return out
}

func EmitTraitOverlayRows(rows TraitOverlayRows) (*Result, error) {
	project, err := rawObject(rows.Project)
	if err != nil {
		return nil, fmt.Errorf("decode project: %w", err)
	}
	tag := strings.ToUpper(strings.TrimSpace(rawString(project, "country_tag")))
	if tag == "" {
		tag = "UNK"
	}
	var dependencies []string
	_ = json.Unmarshal(project["dependencies"], &dependencies)
	var exportLanguages []string
	_ = json.Unmarshal(project["export_languages"], &exportLanguages)

	in := &Input{
		Tag:             tag,
		IsSubmod:        len(dependencies) > 0,
		VanillaTraits:   rows.VanillaTraits,
		ExportLanguages: overlayLanguages(exportLanguages),
	}
	for _, raw := range rows.Characters {
		character, source, err := decodeOverlayCharacter(raw)
		if err != nil {
			return nil, fmt.Errorf("decode character: %w", err)
		}
		if !rows.IncludeImported && source == "imported" {
			continue
		}
		in.Characters = append(in.Characters, character)
		if !character.ReplacesVanillaID {
			continue
		}
		switch character.RoleType {
		case "country_leader":
			in.ReplacementCountryLeaders = append(in.ReplacementCountryLeaders, character)
		case "general", "field_marshal", "admiral":
			in.ReplacementUnitLeaders = append(in.ReplacementUnitLeaders, character)
		case "scientist":
			in.ReplacementScientists = append(in.ReplacementScientists, character)
		}
	}
	for _, raw := range rows.CustomTraits {
		obj, err := rawObject(raw)
		if err != nil {
			return nil, fmt.Errorf("decode custom trait: %w", err)
		}
		in.CustomTraits = append(in.CustomTraits, CustomTrait{
			TraitID: rawString(obj, "trait_id"), TraitType: rawString(obj, "trait_type"), CountryTag: rawString(obj, "country_tag"),
		})
	}
	for _, raw := range rows.EventNamespaces {
		obj, err := rawObject(raw)
		if err != nil {
			return nil, fmt.Errorf("decode event namespace: %w", err)
		}
		in.EventNamespaces = append(in.EventNamespaces, EventNamespace{
			Namespace: rawString(obj, "namespace"), ID: rawInt(obj, "id"),
		})
	}
	portraitOBS, advisorOBS := map[int]string{}, map[int]string{}
	for _, raw := range rows.EntityOBSPaths {
		obj, err := rawObject(raw)
		if err != nil {
			return nil, fmt.Errorf("decode character obs path: %w", err)
		}
		kind := rawString(obj, "entity_type")
		if kind != "character_portrait" && kind != "character_advisor" {
			continue
		}
		id, key := rawInt(obj, "entity_id"), rawString(obj, "obs_key")
		available := false
		if rows.OBSAvailable != nil && strings.TrimSpace(key) != "" {
			available, err = rows.OBSAvailable(key)
			if err != nil {
				return nil, fmt.Errorf("resolve character obs path %q: %w", key, err)
			}
		}
		if !available {
			continue
		}
		switch kind {
		case "character_portrait":
			portraitOBS[id] = key
		case "character_advisor":
			advisorOBS[id] = key
		}
	}
	in.PortraitObs = func(id int) string { return portraitOBS[id] }
	in.AdvisorObs = func(id int) string { return advisorOBS[id] }
	return EmitTraitOverlayArtifacts(in), nil
}
