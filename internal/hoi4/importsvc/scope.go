package importsvc

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type ScopeKind string

const (
	ScopeOwn          ScopeKind = "own"
	ScopeExplicit     ScopeKind = "explicit"
	ScopeGlobal       ScopeKind = "global"
	ScopeGlobalExcept ScopeKind = "global_except"
)

type ScopeVerdict struct {
	Kind        ScopeKind
	CountryTags []string
}

func (v ScopeVerdict) TagList() []string {
	out := make([]string, len(v.CountryTags))
	copy(out, v.CountryTags)
	return out
}

func (v ScopeVerdict) StorageTuple() (kind string, countryTag string, shared []string) {
	switch v.Kind {
	case ScopeOwn:
		owner := ""
		if len(v.CountryTags) > 0 {
			owner = v.CountryTags[0]
		}
		return "own", owner, nil
	case ScopeExplicit:
		return "explicit", "", v.TagList()
	case ScopeGlobal:
		return "global", "", nil
	default:
		return "global_except", "", v.TagList()
	}
}

func SynthCountryGate(scopeKind, countryTag string, sharedCountryTags []string) (string, bool) {
	sk := scopeKind
	if sk == "" {
		sk = "own"
	}
	sk = pyLower(sk)
	var tags []string
	for _, t := range sharedCountryTags {
		if t != "" {
			tags = append(tags, pyUpper(t))
		}
	}
	switch sk {
	case "own":
		owner := pyUpper(countryTag)
		if owner == "" {
			return "", false
		}
		return "original_tag = " + owner, true
	case "explicit":
		members := tags
		if len(members) == 0 && countryTag != "" {
			members = []string{pyUpper(countryTag)}
		}
		if len(members) == 0 {
			return "", false
		}
		parts := make([]string, 0, len(members))
		for _, t := range members {
			parts = append(parts, "original_tag = "+t)
		}
		return "OR = { " + strings.Join(parts, " ") + " }", true
	case "global_except":
		if len(tags) == 0 {
			return "", false
		}
		parts := make([]string, 0, len(tags))
		for _, t := range tags {
			parts = append(parts, "original_tag = "+t)
		}
		return "NOT = { OR = { " + strings.Join(parts, " ") + " } }", true
	}

	return "", false
}

var undecidableKeys = map[string]bool{
	"has_country_flag": true, "has_global_flag": true, "has_character_flag": true,
	"custom_trigger_tooltip": true, "hidden_trigger": true,
}

var transparentKeys = map[string]bool{
	"allowed": true, "visible": true, "available": true, "potential": true,
	"trigger": true, "allowed_civil_war": true,
	"if": true, "else_if": true, "else": true,
}

func looksLikeTag(tok string) bool {
	n := len([]rune(tok))
	return n >= 2 && n <= 5 && pyIsAlphaUpperTag(tok)
}

type stringSet map[string]bool

func (s stringSet) update(o stringSet) {
	for k := range o {
		s[k] = true
	}
}

func (s stringSet) sorted() []string {
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type signals struct {
	direct      stringSet
	orTags      stringSet
	notTags     stringSet
	hasScripted bool
	hasFlag     bool
}

func newSignals() *signals {
	return &signals{direct: stringSet{}, orTags: stringSet{}, notTags: stringSet{}}
}

func collect(block []*paradox.Entry) *signals {
	sig := newSignals()
	for _, e := range block {
		if e.Key == nil {
			continue
		}
		key := pyLower(*e.Key)

		switch {
		case (key == "tag" || key == "original_tag" || key == "has_original_tag") && e.Block == nil:
			val := pyUpper(PyScalarStr(e.Scalar))
			if looksLikeTag(val) {
				sig.direct[val] = true
			}

		case key == "or" && e.Block != nil:
			sub := collect(e.Block.Entries)
			sig.orTags.update(sub.direct)
			sig.orTags.update(sub.orTags)
			sig.hasScripted = sig.hasScripted || sub.hasScripted
			sig.hasFlag = sig.hasFlag || sub.hasFlag

		case key == "not" && e.Block != nil:
			sub := collect(e.Block.Entries)

			sig.notTags.update(sub.direct)
			sig.notTags.update(sub.orTags)
			sig.hasScripted = sig.hasScripted || sub.hasScripted
			sig.hasFlag = sig.hasFlag || sub.hasFlag

		case key == "and" && e.Block != nil:
			sub := collect(e.Block.Entries)
			sig.direct.update(sub.direct)
			sig.orTags.update(sub.orTags)
			sig.notTags.update(sub.notTags)
			sig.hasScripted = sig.hasScripted || sub.hasScripted
			sig.hasFlag = sig.hasFlag || sub.hasFlag

		case undecidableKeys[key]:
			sig.hasFlag = true

		case e.Block != nil:
			if transparentKeys[key] {
				sub := collect(e.Block.Entries)
				sig.direct.update(sub.direct)
				sig.orTags.update(sub.orTags)
				sig.notTags.update(sub.notTags)
				sig.hasScripted = sig.hasScripted || sub.hasScripted
				sig.hasFlag = sig.hasFlag || sub.hasFlag
			} else {

				sig.hasScripted = true
			}
		}
	}
	return sig
}

func ClassifyScope(blocks [][]*paradox.Entry) ScopeVerdict {
	if len(blocks) == 0 {
		return ScopeVerdict{Kind: ScopeGlobal, CountryTags: []string{}}
	}

	direct := stringSet{}
	orTags := stringSet{}
	notTags := stringSet{}
	for _, blk := range blocks {
		s := collect(blk)
		direct.update(s.direct)
		orTags.update(s.orTags)
		notTags.update(s.notTags)

	}

	positives := stringSet{}
	positives.update(direct)
	positives.update(orTags)

	if len(positives) == 0 && len(notTags) == 0 {
		return ScopeVerdict{Kind: ScopeGlobal, CountryTags: []string{}}
	}

	if len(notTags) > 0 && len(positives) == 0 {
		return ScopeVerdict{Kind: ScopeGlobalExcept, CountryTags: notTags.sorted()}
	}

	if len(positives) > 0 && len(notTags) > 0 {
		residual := stringSet{}
		for k := range positives {
			if !notTags[k] {
				residual[k] = true
			}
		}
		if len(residual) == 0 {
			leftoverNot := stringSet{}
			for k := range notTags {
				if !positives[k] {
					leftoverNot[k] = true
				}
			}
			if len(leftoverNot) > 0 {
				return ScopeVerdict{Kind: ScopeGlobalExcept, CountryTags: leftoverNot.sorted()}
			}
			return ScopeVerdict{Kind: ScopeGlobal, CountryTags: []string{}}
		}
		positives = residual
	}

	orMinusNot := false
	for k := range orTags {
		if !notTags[k] {
			orMinusNot = true
			break
		}
	}
	if len(positives) == 1 && !orMinusNot {
		return ScopeVerdict{Kind: ScopeOwn, CountryTags: positives.sorted()}
	}
	return ScopeVerdict{Kind: ScopeExplicit, CountryTags: positives.sorted()}
}

func collectTagsAnywhere(block []*paradox.Entry) stringSet {
	tags := stringSet{}
	for _, e := range block {
		isTagKey := e.Key != nil &&
			(*e.Key == "tag" || *e.Key == "original_tag" || *e.Key == "has_original_tag")
		if isTagKey && e.Block == nil {
			val := pyUpper(pyStrip(PyScalarStr(e.Scalar)))
			if looksLikeTag(val) {
				tags[val] = true
			}
		} else if e.Block != nil {
			tags.update(collectTagsAnywhere(e.Block.Entries))
		}
	}
	return tags
}

func isZeroScalar(v paradox.Scalar) bool {
	f, ok := pyFloat(pyStrip(PyScalarStr(v)))
	return ok && f == 0.0
}

func ClassifyFocusTreeCountryBlock(countryBlock []*paradox.Entry) ScopeVerdict {
	if countryBlock == nil {
		return ScopeVerdict{Kind: ScopeGlobal, CountryTags: []string{}}
	}

	included := stringSet{}
	excluded := stringSet{}
	for _, e := range countryBlock {
		if e.Key == nil || *e.Key != "modifier" || e.Block == nil {
			continue
		}
		modTags := collectTagsAnywhere(e.Block.Entries)
		if len(modTags) == 0 {
			continue
		}
		isExclusion := false
		for _, x := range e.Block.Entries {
			if x.Key != nil && *x.Key == "factor" && x.Block == nil && isZeroScalar(x.Scalar) {
				isExclusion = true
				break
			}
		}
		if isExclusion {
			excluded.update(modTags)
		} else {
			included.update(modTags)
		}
	}

	if len(included) > 0 {
		if len(included) == 1 {
			return ScopeVerdict{Kind: ScopeOwn, CountryTags: included.sorted()}
		}
		return ScopeVerdict{Kind: ScopeExplicit, CountryTags: included.sorted()}
	}
	if len(excluded) > 0 {
		return ScopeVerdict{Kind: ScopeGlobalExcept, CountryTags: excluded.sorted()}
	}
	return ScopeVerdict{Kind: ScopeGlobal, CountryTags: []string{}}
}
