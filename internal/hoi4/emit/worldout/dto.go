package worldout

import (
	"bytes"
	"encoding/json"
	"os"

	"haganeworks/internal/hoi4/cosmetic"
	"haganeworks/internal/hoi4/exportmsg"
)

type Scenario struct {
	Name             string
	Project          *OMap
	Tag              string
	ModName          string
	ModDirName       string
	IsSubmod         bool
	IncludeImported  bool
	ExistenceSets    map[string]map[string]bool
	RawOverride      bool
	FullTextOverride *string

	IdeologyPlan        *IdeologyExportPlan
	BookmarkGfx         [][2]string
	ContentByTag        *OMap
	RecruitByCtag       *OMap
	ClByCtag            *OMap
	Countries           []*OMap
	HistoryFiles        []*OMap
	States              []*OMap
	Bookmarks           []*OMap
	CosmeticDefinitions map[string]cosmetic.Definition
	CosmeticOwners      map[string]string
	PreservedLocKeys    map[string]map[string]bool

	OOBFiles []OOBFile
}

func scenarioFromOMap(o *OMap) *Scenario {
	s := &Scenario{
		Name:            eStr(o, "name"),
		Project:         eMap(o, "project"),
		Tag:             eStr(o, "tag"),
		ModName:         eStr(o, "mod_name"),
		ModDirName:      eStr(o, "mod_dir_name"),
		IsSubmod:        eBool(o, "is_submod"),
		IncludeImported: eBool(o, "include_imported"),
		RawOverride:     eBool(o, "standalone_ideo_raw_override"),
		ContentByTag:    eMap(o, "content_by_tag"),
		RecruitByCtag:   eMap(o, "recruit_chars_by_ctag"),
		ClByCtag:        eMap(o, "cl_replacements_by_ctag"),
	}
	if v, ok := o.Get("standalone_full_text_override"); ok {
		txt := pyStr(v)
		s.FullTextOverride = &txt
	}
	s.ExistenceSets = map[string]map[string]bool{}
	if es := eMap(o, "existence_sets"); es != nil {
		for _, k := range es.Keys() {
			set := map[string]bool{}
			if lv, ok := es.Get(k); ok {
				if l, isList := lv.([]Value); isList {
					for _, item := range l {
						set[pyStr(item)] = true
					}
				}
			}
			s.ExistenceSets[k] = set
		}
	}
	for _, pair := range eList(o, "bookmark_gfx") {
		if p, ok := pair.([]Value); ok && len(p) >= 2 {
			s.BookmarkGfx = append(s.BookmarkGfx,
				[2]string{pyStr(p[0]), pyStr(p[1])})
		}
	}
	collect := func(key string) []*OMap {
		var out []*OMap
		for _, v := range eList(o, key) {
			if m, ok := v.(*OMap); ok {
				out = append(out, m)
			}
		}
		return out
	}
	s.Countries = collect("countries")
	s.HistoryFiles = collect("history_files")
	s.States = collect("states")
	s.Bookmarks = collect("bookmarks")
	return s
}

func LoadCorpus(path string) ([]*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	root, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	list, _ := root.([]Value)
	var out []*Scenario
	for _, item := range list {
		if om, ok := item.(*OMap); ok {
			out = append(out, scenarioFromOMap(om))
		}
	}
	return out, nil
}

type Record struct {
	Path string
	Text string
}

type Meta struct {
	Politics []exportmsg.Message
	Orphan   []exportmsg.Message
	Symmetry []exportmsg.Message
	Skip     map[string]int64

	Ideology IdeologyExportFacts

	OOB []exportmsg.Message
}
