package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/hoi4dlc"
	"haganeworks/internal/hoi4/paradox"
)

var (
	SelectableDLCs = hoi4dlc.Selectable
	IntegratedDLCs = hoi4dlc.Integrated
)

func (idx *Index) SetOwnedDLCs(names []string, all bool) {
	if all {
		idx.ownedDLC = nil
		return
	}
	owned := map[string]bool{}
	for _, name := range IntegratedDLCs {
		owned[dlcKey(name)] = true
	}
	for _, name := range names {
		owned[dlcKey(name)] = true
	}
	idx.ownedDLC = owned
}

func (idx *Index) OwnsDLC(name string) bool {
	return idx.ownedDLC == nil || idx.ownedDLC[dlcKey(name)]
}

func dlcKey(name string) string { return strings.ToLower(strings.TrimSpace(strings.Trim(name, "\""))) }

type dlcGuard struct {
	Need    []string
	Without []string
	AnyOf   [][]string
}

func (g dlcGuard) empty() bool { return len(g.Need) == 0 && len(g.Without) == 0 && len(g.AnyOf) == 0 }

func (g dlcGuard) and(o dlcGuard) dlcGuard {
	if o.empty() {
		return g
	}
	if g.empty() {
		return o
	}
	return dlcGuard{
		Need:    append(append([]string(nil), g.Need...), o.Need...),
		Without: append(append([]string(nil), g.Without...), o.Without...),
		AnyOf:   append(append([][]string(nil), g.AnyOf...), o.AnyOf...),
	}
}

func (g dlcGuard) negate() (dlcGuard, bool) {
	switch {
	case len(g.Need) == 1 && len(g.Without) == 0 && len(g.AnyOf) == 0:
		return dlcGuard{Without: []string{g.Need[0]}}, true
	case len(g.Need) == 0 && len(g.Without) == 1 && len(g.AnyOf) == 0:
		return dlcGuard{Need: []string{g.Without[0]}}, true
	case len(g.Need) == 0 && len(g.Without) == 0 && len(g.AnyOf) == 1:
		return dlcGuard{Without: append([]string(nil), g.AnyOf[0]...)}, true
	}
	return dlcGuard{}, false
}

func (g dlcGuard) holds(owned map[string]bool) bool {
	has := func(name string) bool { return owned == nil || owned[dlcKey(name)] }
	for _, name := range g.Need {
		if !has(name) {
			return false
		}
	}
	for _, name := range g.Without {
		if has(name) {
			return false
		}
	}
	for _, group := range g.AnyOf {
		any := false
		for _, name := range group {
			if has(name) {
				any = true
				break
			}
		}
		if !any {
			return false
		}
	}
	return true
}

func limitGuard(entries []*paradox.Entry) (dlcGuard, bool) {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		if strings.EqualFold(*entry.Key, "limit") {
			return conditionGuard(entry.Block.Entries)
		}
	}
	return dlcGuard{}, false
}

func conditionGuard(entries []*paradox.Entry) (dlcGuard, bool) {
	guard := dlcGuard{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		switch {
		case key == "has_dlc" && entry.Block == nil:
			guard.Need = append(guard.Need, toString(scalarValue(entry.Scalar)))
		case (key == "not" || key == "nor") && entry.Block != nil:

			names, ok := plainDLCs(entry.Block.Entries)
			if !ok {
				return dlcGuard{}, false
			}
			guard.Without = append(guard.Without, names...)
		case key == "and" && entry.Block != nil:
			inner, ok := conditionGuard(entry.Block.Entries)
			if !ok {
				return dlcGuard{}, false
			}
			guard = guard.and(inner)
		case key == "or" && entry.Block != nil:
			names, ok := plainDLCs(entry.Block.Entries)
			if !ok {
				return dlcGuard{}, false
			}
			if len(names) == 1 {
				guard.Need = append(guard.Need, names[0])
			} else if len(names) > 1 {
				guard.AnyOf = append(guard.AnyOf, names)
			}
		default:
			return dlcGuard{}, false
		}
	}
	return guard, true
}

func plainDLCs(entries []*paradox.Entry) ([]string, bool) {
	var names []string
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		if !strings.EqualFold(*entry.Key, "has_dlc") || entry.Block != nil {
			return nil, false
		}
		names = append(names, toString(scalarValue(entry.Scalar)))
	}
	return names, true
}

type branchState struct {
	elseGuard *dlcGuard
}

func (b *branchState) enterIf(entries []*paradox.Entry, parent dlcGuard) (dlcGuard, bool) {
	guard, decidable := limitGuard(entries)
	if !decidable {
		b.elseGuard = nil
		return dlcGuard{}, false
	}
	if negated, ok := guard.negate(); ok {
		combined := parent.and(negated)
		b.elseGuard = &combined
	} else if guard.holds(nil) {

		b.elseGuard = nil
	} else {
		combined := parent
		b.elseGuard = &combined
	}
	return parent.and(guard), true
}

func (b *branchState) enterElse() (dlcGuard, bool) {
	if b.elseGuard == nil {
		return dlcGuard{}, false
	}
	guard := *b.elseGuard
	b.elseGuard = nil
	return guard, true
}

func sortedDLCs(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
