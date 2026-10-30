package worldout

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

type OOBFile struct {
	Path string

	Tags []string

	Locations []int64

	NavalProvinces []int64

	AirStates []int64

	RawOverridden bool
}

func OOBStem(path string) string {
	clean := strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
	clean = strings.TrimPrefix(clean, "history/units/")
	clean = strings.TrimSuffix(clean, ".txt")
	return strings.TrimSpace(clean)
}

var oobRefRe = regexp.MustCompile(
	`(?i)(?:^|[^A-Za-z0-9_])(set_oob|set_air_oob|set_naval_oob|load_oob|oob)[ \t]*=[ \t]*(?:"([^"]*)"|([^\s"{}=#]+))`)

func parseOOBRefs(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var live []string
	for _, line := range strings.Split(text, "\n") {
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		live = append(live, line)
	}
	var out []string
	for _, m := range oobRefRe.FindAllStringSubmatch(strings.Join(live, "\n"), -1) {
		value := m[2]
		if value == "" {
			value = m[3]
		}
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, OOBStem(value))
		}
	}
	return out
}

type oobWireAction struct {
	Tag  string
	Stem string

	Layer int

	Detail string
}

type oobWiringPlan struct {
	Actions  []oobWireAction
	Warnings []exportmsg.Message
}

func oobGeoReason(kinds string) exportmsg.Message {
	phrase := map[string]string{
		"army": "oob.reason.army",
		"navy": "oob.reason.navy",
		"air":  "oob.reason.air",
	}
	var reason exportmsg.Message
	for _, kind := range strings.Split(kinds, "+") {
		key := phrase[kind]
		if key == "" {
			continue
		}
		if reason.Key == "" {
			reason = exportmsg.Message{Key: key}
			continue
		}
		reason = exportmsg.Message{Key: "oob.reason.and", Args: exportmsg.Args{"first": reason, "second": exportmsg.Message{Key: key}}}
	}
	if reason.Key == "" {
		return exportmsg.Message{Key: "oob.reason.stationed"}
	}
	return reason
}

func oobLayerReason(layer int, detail string) exportmsg.Message {
	switch layer {
	case 1:
		return exportmsg.Message{Key: "oob.reason.own_rows"}
	case 2:
		return exportmsg.Message{Key: "oob.reason.file_prefix", Args: exportmsg.Args{"prefix": detail}}
	case 3:
		return oobGeoReason(detail)
	default:
		return exportmsg.Message{Key: "oob.reason.main_country"}
	}
}

func oobStemTag(stem string) string {
	base := stem
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	if idx := strings.Index(base, "_"); idx >= 0 {
		base = base[:idx]
	}
	return toUpper(pyStrip(base))
}

func oobModeTag(values []string) string {
	counts := map[string]int{}
	var order []string
	for _, v := range values {
		if _, seen := counts[v]; !seen {
			order = append(order, v)
		}
		counts[v]++
	}
	best, bestN := "", 0
	for _, v := range order {
		if counts[v] > bestN {
			best, bestN = v, counts[v]
		}
	}
	return best
}

func oobProvinceID(v Value) (int64, bool) {
	if id, ok := intVal(v); ok && id != 0 {
		return id, true
	}
	if s, ok := v.(string); ok {
		if id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil && id != 0 {
			return id, true
		}
	}
	return 0, false
}

type oobOwnerIndex struct {
	province map[int64]string
	state    map[int64]string
}

func (o *oobOwnerIndex) provinceOwner(id int64) string {
	if o == nil {
		return ""
	}
	return o.province[id]
}

func (o *oobOwnerIndex) stateOwner(id int64) string {
	if o == nil {
		return ""
	}
	return o.state[id]
}

func buildProvinceOwner(states []*OMap) *oobOwnerIndex {
	out := &oobOwnerIndex{province: map[int64]string{}, state: map[int64]string{}}
	for _, st := range states {
		owner := toUpper(pyStrip(pyStrOrEmpty(eGet(st, "owner"))))
		if owner == "" {
			continue
		}
		for _, pv := range eList(st, "provinces") {
			if id, ok := oobProvinceID(pv); ok {
				out.province[id] = owner
			}
		}

		if id, ok := oobProvinceID(eGet(st, "state_id")); ok {
			out.state[id] = owner
		} else if id, ok := oobProvinceID(eGet(st, "id")); ok {
			out.state[id] = owner
		}
	}
	return out
}

func planOOBWiring(files []OOBFile, refsByTag map[string][]string,
	knownTags map[string]bool, owners *oobOwnerIndex,
	primaryTag string) oobWiringPlan {

	var plan oobWiringPlan
	wirable := 0
	for _, f := range files {
		if !f.RawOverridden {
			wirable++
		}
	}
	if wirable == 0 {

		return plan
	}

	present := map[string]bool{}
	for _, f := range files {
		if stem := OOBStem(f.Path); stem != "" {
			present[stem] = true
		}
	}

	referenced := map[string]bool{}
	for _, stems := range refsByTag {
		for _, stem := range stems {
			if stem != "" {
				referenced[stem] = true
			}
		}
	}

	primary := toUpper(pyStrip(primaryTag))
	for _, f := range files {
		stem := OOBStem(f.Path)
		if stem == "" {
			continue
		}
		if f.RawOverridden {

			continue
		}
		if referenced[stem] {
			continue
		}

		tag, layer, detail := "", 0, ""
		var candidates []string
		for _, t := range f.Tags {
			if t != "" && knownTags[t] {
				candidates = append(candidates, t)
			}
		}
		if picked := oobModeTag(candidates); picked != "" {
			tag, layer = picked, 1
		}
		if tag == "" {

			if prefix := oobStemTag(stem); prefix != "" && knownTags[prefix] {
				tag, layer, detail = prefix, 2, prefix
			}
		}
		if tag == "" {

			type geoVote struct{ owner, kind string }
			var votes []geoVote
			for _, province := range f.Locations {
				if owner := owners.provinceOwner(province); owner != "" {
					votes = append(votes, geoVote{owner, "army"})
				}
			}
			for _, province := range f.NavalProvinces {
				if owner := owners.provinceOwner(province); owner != "" {
					votes = append(votes, geoVote{owner, "navy"})
				}
			}
			for _, state := range f.AirStates {
				if owner := owners.stateOwner(state); owner != "" {
					votes = append(votes, geoVote{owner, "air"})
				}
			}
			ballots := make([]string, 0, len(votes))
			for _, v := range votes {
				ballots = append(ballots, v.owner)
			}
			if picked := oobModeTag(ballots); picked != "" {
				backing := map[string]bool{}
				for _, v := range votes {
					if v.owner == picked {
						backing[v.kind] = true
					}
				}
				var kinds []string
				for _, kind := range []string{"army", "navy", "air"} {
					if backing[kind] {
						kinds = append(kinds, kind)
					}
				}
				tag, layer, detail = picked, 3, strings.Join(kinds, "+")
			}
		}
		if tag == "" && primary != "" {
			tag, layer = primary, 4
		}
		if tag == "" {
			plan.Warnings = append(plan.Warnings, exportmsg.Message{Key: "oob.unassigned", Args: exportmsg.Args{"file": f.Path, "stem": stem}})
			continue
		}

		plan.Actions = append(plan.Actions,
			oobWireAction{Tag: tag, Stem: stem, Layer: layer, Detail: detail})
		referenced[stem] = true
		plan.Warnings = append(plan.Warnings, exportmsg.Message{Key: "oob.wired", Args: exportmsg.Args{
			"file": f.Path, "stem": stem, "tag": tag, "reason": oobLayerReason(layer, detail),
		}})
	}

	tags := make([]string, 0, len(refsByTag))
	for tag := range refsByTag {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	var shipped []string
	for _, f := range files {
		if stem := OOBStem(f.Path); stem != "" {
			shipped = append(shipped, stem)
		}
	}
	for _, tag := range tags {
		seen := map[string]bool{}
		for _, stem := range refsByTag[tag] {
			if stem == "" || present[stem] || seen[stem] {
				continue
			}
			seen[stem] = true
			plan.Warnings = append(plan.Warnings, exportmsg.Message{Key: "oob.dangling", Args: exportmsg.Args{
				"tag": tag, "stem": stem, "files": strings.Join(shipped, ", "),
			}})
		}
	}
	return plan
}
