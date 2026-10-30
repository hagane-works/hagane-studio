package content

import (
	"io/fs"
	"strings"

	"haganeworks/internal/gamefiles"
	"haganeworks/internal/hoi4/paradox"
)

type dynamicModifierIDsKey struct{}

const utf8BOM = "\xef\xbb\xbf"

func isVanillaDynamicModifier(game func() (fs.FS, error), id string) bool {
	g, err := gamefiles.Open(game)
	if err != nil {
		return false
	}
	return gamefiles.Derived(g, dynamicModifierIDsKey{}, func(g *gamefiles.Game) map[string]bool {
		return readTopLevelKeys(g, "common/dynamic_modifiers")
	})[id]
}

func readTopLevelKeys(game *gamefiles.Game, dir string) map[string]bool {
	ids := map[string]bool{}
	for _, name := range game.Files(dir, false, ".txt") {
		raw, err := game.ReadFile(name)
		if err != nil {
			continue
		}
		block, _, err := paradox.Parse(strings.TrimPrefix(string(raw), utf8BOM))
		if err != nil || block == nil {
			continue
		}
		for _, item := range block.Entries {
			if item.Key != nil {
				ids[*item.Key] = true
			}
		}
	}
	return ids
}
