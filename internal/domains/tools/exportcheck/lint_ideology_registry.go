package exportcheck

import "haganeworks/internal/reftables"

func vanillaIdeologyGroupIDs() map[string]bool {
	out := map[string]bool{}
	for _, id := range reftables.Ideologies().Groups {
		out[id] = true
	}
	return out
}

func vanillaSubIdeologyIDs() map[string]bool {
	out := map[string]bool{}
	for _, sub := range reftables.Ideologies().SubIdeologies {
		out[sub.ID] = true
	}
	return out
}
