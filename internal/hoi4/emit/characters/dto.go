package characters

import (
	"math/big"
	"sort"
	"strings"
	"unicode"
)

type KV struct {
	Key string
	Val any
}

func kvLookup(kvs []KV, key string) (any, bool) {
	for _, kv := range kvs {
		if kv.Key == key {
			return kv.Val, true
		}
	}
	return nil, false
}

type SciSpec struct {
	Spec  string
	Level *big.Int
}

type Character struct {
	ID     int
	Tag    string
	Name   string
	NameEn string
	NameZh string
	DescEn string
	DescZh string

	DisplayNameLocs []KV
	DescriptionLocs []KV

	CountryTag     string
	RoleType       string
	HasAdvisorRole bool

	Traits        []string
	AdvisorTraits []string

	Skill             *big.Int
	AttackSkill       *big.Int
	DefenseSkill      *big.Int
	PlanningSkill     *big.Int
	LogisticsSkill    *big.Int
	ManeuveringSkill  *big.Int
	CoordinationSkill *big.Int

	AdvisorSlot      string
	AdvisorCost      *big.Int
	AdvisorIdeaToken string

	ScientistSpecs []SciSpec

	Gender                     string
	CanBeCaptured              *bool
	LargePortrait              string
	SmallPortrait              string
	Ideology                   string
	Expire                     string
	MilitaryVisible            string
	ScientistVisible           string
	OperativeSkillLevel        *big.Int
	OperativeTraits            []string
	OperativeNationalities     []string
	OperativeAvailable         string
	OperativeBg                string
	AdvisorAvailable           string
	AdvisorVisible             string
	AdvisorModifier            string
	AdvisorOnAdd               string
	AdvisorOnRemove            string
	AdvisorRemovalCost         *big.Int
	AlwaysShowOnActionsTooltip *bool

	RawExtras []KV

	SourceFile         string
	ReplacesVanillaID  bool
	IsPortraitModified bool
}

type CustomTrait struct {
	TraitID    string
	TraitType  string
	CountryTag string
}

type EventNamespace struct {
	Namespace string
	ID        int
}

type RawFile struct {
	DestPath string
	Source   string
}

type LocLanguage struct {
	Code   string
	Subdir string
	Suffix string
}

type Input struct {
	Tag      string
	IsSubmod bool

	Characters                []*Character
	ReplacementCountryLeaders []*Character
	ReplacementUnitLeaders    []*Character
	ReplacementScientists     []*Character

	CustomTraits    []CustomTrait
	EventNamespaces []EventNamespace
	RawFiles        []RawFile
	ExportLanguages []LocLanguage

	VanillaTraits map[string]bool

	PortraitObs func(id int) string
	AdvisorObs  func(id int) string
}

type CLOut struct {
	CharTag       string   `json:"char_tag"`
	Ideology      string   `json:"ideology"`
	Traits        []string `json:"traits"`
	Expire        string   `json:"expire"`
	PictureSprite string   `json:"picture_sprite"`
	LocEmitted    bool     `json:"loc_emitted"`
}

type TraitOverlayArtifacts struct {
	Namespace                    string
	EventPath                    string
	EventLocalisationPaths       []string
	ReplacementPortraitsPath     string
	ReplacementLocalisationPaths []string
}

type TextOutput struct {
	Path string
	Text string
}

type Result struct {
	Texts           map[string]string
	Recruit         map[string][]string
	CL              map[string][]CLOut
	Skips           map[string]int
	StructuredPaths []string
	TraitOverlays   map[string]TraitOverlayArtifacts
}

func (r *Result) TraitOverlayNamespaces() map[string]string {
	out := map[string]string{}
	if r == nil {
		return out
	}
	for ctag, artifacts := range r.TraitOverlays {
		namespace := strings.TrimSpace(artifacts.Namespace)
		if namespace == "" || strings.TrimSpace(artifacts.EventPath) == "" {
			continue
		}
		if _, exists := r.Texts[artifacts.EventPath]; !exists {
			continue
		}
		out[ctag] = namespace
	}
	return out
}

func (r *Result) TextOutputs() []TextOutput {
	if r == nil {
		return nil
	}
	paths := make([]string, 0, len(r.Texts))
	for path := range r.Texts {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	out := make([]TextOutput, 0, len(paths))
	for _, path := range paths {
		out = append(out, TextOutput{Path: path, Text: r.Texts[path]})
	}
	return out
}

func bigOf(v any) (*big.Int, bool) {
	z, ok := v.(*big.Int)
	return z, ok
}

func isLetterRune(r rune) bool { return unicode.IsLetter(r) }
func isNumberRune(r rune) bool { return unicode.IsNumber(r) }
