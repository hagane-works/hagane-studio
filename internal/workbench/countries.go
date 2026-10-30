package workbench

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/emit/worldout"
)

var ErrCountryMissing = errors.New("Country not found in this project")

type ErrCountryConflict struct{ Tag string }

func (e ErrCountryConflict) Error() string {
	return fmt.Sprintf("Country %s already exists in this project", e.Tag)
}

type ErrCountryTagInvalid struct{ Tag string }

func (e ErrCountryTagInvalid) Error() string { return countryTagPatternMessage }

var countryTagRE = regexp.MustCompile(`^[A-Z][A-Z0-9]{2,4}$`)

const countryTagPatternMessage = "tag must match ^[A-Z][A-Z0-9]{2,4}$ (3-5 chars, uppercase letter first; HOI4 tags are at least 3 characters)"

func vanillaCountryTag(tag string) bool {
	t, err := worldout.LoadVanillaTables(context.Background())
	if err != nil {
		return false
	}
	_, ok := t.CountryCultures[strings.ToUpper(tag)]
	return ok
}

func VanillaCountryTag(tag string) bool { return vanillaCountryTag(tag) }

func vanillaTagWarning(tag string) *string {
	if !vanillaCountryTag(tag) {
		return nil
	}
	w := fmt.Sprintf(
		"%s 是 vanilla 国家;建议编辑现有国家而不是新建重复的。 / %s is a vanilla country; consider editing the existing country instead of creating a duplicate.",
		tag, tag)
	return &w
}

func cleanLocs(value map[string]string) map[string]string {
	out := map[string]string{}
	for lang, raw := range value {
		if text := strings.TrimSpace(raw); text != "" {
			out[lang] = text
		}
	}
	return out
}

func mergeLegacyLocs(value map[string]string, en, zh string) map[string]string {
	locs := cleanLocs(value)
	for lang, raw := range map[string]string{"en": en, "zh": zh} {
		if text := strings.TrimSpace(raw); text != "" && locs[lang] == "" {
			locs[lang] = text
		}
	}
	return locs
}

type CountryLocsFields struct {
	AdjectiveEn, AdjectiveZh string
	AdjectiveLocs            map[string]string
	IdeologyNameLocs         map[string]map[string]string
	IdeologyScalars          map[string]*[2]string
}

var ideologyGroups = []string{"democratic", "fascism", "communism", "neutrality"}

func normalizeCountryLocs(f *CountryLocsFields) {
	adj := mergeLegacyLocs(f.AdjectiveLocs, f.AdjectiveEn, f.AdjectiveZh)
	f.AdjectiveLocs = adj
	if len(adj) > 0 {
		if strings.TrimSpace(f.AdjectiveEn) == "" {
			f.AdjectiveEn = adj["en"]
		}
		if strings.TrimSpace(f.AdjectiveZh) == "" {
			f.AdjectiveZh = adj["zh"]
		}
	}
	ideo := map[string]map[string]string{}
	for group, locs := range f.IdeologyNameLocs {
		ideo[group] = cleanLocs(locs)
	}
	for _, group := range ideologyGroups {
		scalars := f.IdeologyScalars[group]
		locs := mergeLegacyLocs(ideo[group], scalars[0], scalars[1])
		if len(locs) > 0 {
			ideo[group] = locs
			if strings.TrimSpace(scalars[0]) == "" {
				scalars[0] = locs["en"]
			}
			if strings.TrimSpace(scalars[1]) == "" {
				scalars[1] = locs["zh"]
			}
		}
	}
	f.IdeologyNameLocs = ideo
}

type CountryCreateInput struct {
	Tag                string                       `json:"tag"`
	Name               *string                      `json:"name"`
	Color              []int64                      `json:"color"`
	ColorUI            []int64                      `json:"color_ui"`
	Ideology           *string                      `json:"ideology"`
	GraphicalCulture   string                       `json:"graphical_culture"`
	GraphicalCulture2D string                       `json:"graphical_culture_2d"`
	UseLegacyAIPPSpend *bool                        `json:"use_legacy_ai_pp_spend"`
	NameEn             string                       `json:"name_en"`
	NameZh             string                       `json:"name_zh"`
	AdjectiveEn        string                       `json:"adjective_en"`
	AdjectiveZh        string                       `json:"adjective_zh"`
	FlagImagePath      string                       `json:"flag_image_path"`
	NameDemocraticEn   string                       `json:"name_democratic_en"`
	NameDemocraticZh   string                       `json:"name_democratic_zh"`
	NameFascismEn      string                       `json:"name_fascism_en"`
	NameFascismZh      string                       `json:"name_fascism_zh"`
	NameCommunismEn    string                       `json:"name_communism_en"`
	NameCommunismZh    string                       `json:"name_communism_zh"`
	NameNeutralityEn   string                       `json:"name_neutrality_en"`
	NameNeutralityZh   string                       `json:"name_neutrality_zh"`
	DisplayNameLocs    map[string]string            `json:"display_name_locs"`
	AdjectiveLocs      map[string]string            `json:"adjective_locs"`
	IdeologyNameLocs   map[string]map[string]string `json:"ideology_name_locs"`
}

func (in *CountryCreateInput) Normalize() error {
	if !countryTagRE.MatchString(in.Tag) {
		return ErrCountryTagInvalid{Tag: in.Tag}
	}
	if in.Name == nil {
		return fmt.Errorf("name is required")
	}
	if in.Color == nil {
		in.Color = []int64{}
	}
	if in.ColorUI == nil {
		in.ColorUI = []int64{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.AdjectiveLocs == nil {
		in.AdjectiveLocs = map[string]string{}
	}
	if in.IdeologyNameLocs == nil {
		in.IdeologyNameLocs = map[string]map[string]string{}
	}
	return nil
}

var groupedContentTables = []struct{ Badge, Table string }{
	{"characters", "characters"},
	{"ideas", "ideas"},
	{"events", "events"},
	{"decisions", "decisions"},
	{"focuses", "focus_trees"},
	{"custom_traits", "custom_traits"},
	{"event_namespaces", "event_namespaces"},
	{"decision_categories", "decision_categories"},
	{"technologies", "technologies"},
}

type OrderedCounts []struct {
	Label string
	N     int64
}

func (o OrderedCounts) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	b.WriteByte('{')
	for i, kv := range o {
		if i > 0 {
			b.WriteByte(',')
		}
		key, _ := json.Marshal(kv.Label)
		b.Write(key)
		b.WriteByte(':')
		fmt.Fprintf(&b, "%d", kv.N)
	}
	b.WriteByte('}')
	return []byte(b.String()), nil
}

func (o OrderedCounts) Total() int64 {
	var t int64
	for _, kv := range o {
		t += kv.N
	}
	return t
}

func strDefault(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

func firstNonBlank(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
