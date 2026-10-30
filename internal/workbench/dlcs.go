package workbench

import (
	"fmt"

	"haganeworks/internal/hoi4/hoi4dlc"
)

func NormalizeOwnedDLCs(names *[]string) (*[]string, error) {
	if names == nil {
		return nil, nil
	}
	chosen := map[string]bool{}
	for _, name := range *names {
		canonical, ok := hoi4dlc.Canonical(name)
		if !ok {
			return nil, fmt.Errorf("owned_dlcs: unknown DLC %q", name)
		}
		chosen[canonical] = true
	}
	out := make([]string, 0, len(chosen))
	for _, name := range hoi4dlc.Selectable {
		if chosen[name] {
			out = append(out, name)
		}
	}
	return &out, nil
}
