package guirender

import (
	"haganeworks/internal/hoi4/paradox"
	"strings"
)

type focusCondition uint8

const (
	focusUnknown focusCondition = iota
	focusFalse
	focusTrue
)

func focusNot(v focusCondition) focusCondition {
	if v == focusTrue {
		return focusFalse
	}
	if v == focusFalse {
		return focusTrue
	}
	return focusUnknown
}

func (idx *Index) focusBranchCondition(entries []*paradox.Entry, any bool, depth int) focusCondition {
	if depth > 32 {
		return focusUnknown
	}
	unknown := false
	for _, e := range entries {
		v := focusUnknown
		if e != nil && e.Key != nil {
			key := strings.ToLower(*e.Key)
			if e.Block != nil {
				switch key {
				case "and":
					v = idx.focusBranchCondition(e.Block.Entries, false, depth+1)
				case "or":
					v = idx.focusBranchCondition(e.Block.Entries, true, depth+1)
				case "not", "nor":
					v = focusNot(idx.focusBranchCondition(e.Block.Entries, true, depth+1))
				case "nand":
					v = focusNot(idx.focusBranchCondition(e.Block.Entries, false, depth+1))
				}
			} else {
				value := e.Scalar.StrValue()
				switch key {
				case "has_dlc":
					v = focusFalse
					if idx.OwnsDLC(value) {
						v = focusTrue
					}
				case "always":
					if strings.EqualFold(value, "yes") {
						v = focusTrue
					}
					if strings.EqualFold(value, "no") {
						v = focusFalse
					}
				default:
					if strings.EqualFold(value, "yes") || strings.EqualFold(value, "no") {
						if body := idx.scriptedTrigger(*e.Key); body != nil {
							v = idx.focusBranchCondition(body.Entries, false, depth+1)
							if strings.EqualFold(value, "no") {
								v = focusNot(v)
							}
						}
					}
				}
			}
		}
		if any && v == focusTrue {
			return focusTrue
		}
		if !any && v == focusFalse {
			return focusFalse
		}
		unknown = unknown || v == focusUnknown
	}
	if unknown {
		return focusUnknown
	}
	if any {
		return focusFalse
	}
	return focusTrue
}

func (idx *Index) visibleFocusBranches(focuses []*Focus) []*Focus {
	hidden := map[string]bool{}
	for _, f := range focuses {
		if idx.focusBranchCondition(f.allowBranch, false, 0) == focusFalse {
			hidden[f.ID] = true
		}
	}

	for changed := true; changed; {
		changed = false
		for _, f := range focuses {
			if hidden[f.ID] {
				continue
			}
			for _, group := range f.Prereq {
				blocked := len(group) > 0
				for _, parent := range group {
					if !hidden[parent] {
						blocked = false
						break
					}
				}
				if blocked {
					hidden[f.ID] = true
					changed = true
					break
				}
			}
		}
	}
	visible := make([]*Focus, 0, len(focuses))
	for _, f := range focuses {
		if !hidden[f.ID] {
			visible = append(visible, f)
		}
	}
	return visible
}
