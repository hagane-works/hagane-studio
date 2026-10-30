package states

import (
	"errors"
	"io/fs"

	"haganeworks/internal/project"
)

func SeedFromGame(p *project.Project, game fs.FS, ids []int64) error {
	if game == nil {
		return errors.New("the game folder is not set up")
	}
	if len(ids) == 0 {
		return nil
	}
	_, err := seedStatesFromVanilla(p, vanilla.stateFiles(game), stateSeedInput{StateIDs: ids})
	return err
}

func GameStateIDMax(game fs.FS) int64 {
	if game == nil {
		return 0
	}
	var top int64
	for _, state := range vanilla.stateFiles(game).Catalog {
		top = max(top, state.StateID)
	}
	return top
}
