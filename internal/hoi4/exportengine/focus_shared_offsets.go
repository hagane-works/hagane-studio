package exportengine

import (
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/exportmsg"
)

func compiledSharedPoolViews(before, after []*emitmisc.NS) map[*emitmisc.NS]bool {
	changed := map[*emitmisc.NS]bool{}
	for i, tree := range after {
		if i < len(before) && tree != before[i] && tree.Str("tree_kind") == "shared_pool" {
			changed[tree] = true
		}
	}
	return changed
}

func (s *imageSink) warnRawSharedPlacement(raw []Entry) {
	if s == nil || len(s.sharedPlacementPaths) == 0 {
		return
	}
	for _, entry := range raw {
		name, err := SafePath(entry.Path)
		if err == nil && s.sharedPlacementPaths[name] {
			s.say("focus.shared_layout_raw", exportmsg.Args{"file": name})
		}
	}
}
