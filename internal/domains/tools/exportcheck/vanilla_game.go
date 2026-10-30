package exportcheck

import (
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"haganeworks/internal/gamefiles"
	"haganeworks/internal/hoi4/paradox"
)

var (
	gameMu sync.Mutex
	gameFS func() (fs.FS, error)
)

func setGame(game func() (fs.FS, error)) {
	gameMu.Lock()
	defer gameMu.Unlock()
	gameFS = game
}

var emptyGateVanilla = &exportGateVanilla{tagSet: map[string]bool{}, ownerByState: map[int]string{}}

type gateVanillaKey struct{}

func exportGateVanillaOnce() *exportGateVanilla {
	gameMu.Lock()
	open := gameFS
	gameMu.Unlock()
	game, err := gamefiles.Open(open)
	if err != nil {
		return emptyGateVanilla
	}
	return gamefiles.Derived(game, gateVanillaKey{}, readGateVanilla)
}

var countryTagKey = regexp.MustCompile(`^[A-Z0-9]{3}$`)

func readGateVanilla(game *gamefiles.Game) *exportGateVanilla {
	d := &exportGateVanilla{StateOwners: map[string]string{}, tagSet: map[string]bool{}, ownerByState: map[int]string{}}
	tags := map[string]bool{}
	for _, name := range game.Files("common/country_tags", false, ".txt") {
		for _, entry := range parseGameFile(game, name) {
			if entry.Key == nil {
				continue
			}
			key := strings.TrimSpace(*entry.Key)
			if key == "dynamic_tags" && entry.Block == nil && strings.EqualFold(entry.Scalar.StrValue(), "yes") {
				break
			}
			if countryTagKey.MatchString(key) {
				tags[key] = true
			}
		}
	}
	for tag := range tags {
		d.CountryTags = append(d.CountryTags, tag)
	}
	sort.Strings(d.CountryTags)
	for _, name := range game.Files("history/states", false, ".txt") {
		for _, entry := range parseGameFile(game, name) {
			if entry.Key == nil || *entry.Key != "state" || entry.Block == nil {
				continue
			}
			id, owner := 0, ""
			for _, field := range entry.Block.Entries {
				if field.Key == nil {
					continue
				}
				switch *field.Key {
				case "id":
					if field.Block == nil {
						id, _ = strconv.Atoi(strings.TrimSpace(field.Scalar.StrValue()))
					}
				case "history":
					if field.Block == nil {
						continue
					}
					for _, h := range field.Block.Entries {
						if h.Key != nil && *h.Key == "owner" && h.Block == nil && owner == "" {
							owner = strings.TrimSpace(h.Scalar.StrValue())
						}
					}
				}
			}
			if id > 0 {
				d.StateIDs = append(d.StateIDs, id)
				if owner != "" {
					d.StateOwners[strconv.Itoa(id)] = owner
					d.ownerByState[id] = owner
				}
			}
		}
	}
	sort.Ints(d.StateIDs)
	if raw, err := game.ReadFile("map/definition.csv"); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			fields := strings.Split(strings.TrimSpace(line), ";")
			if len(fields) < 5 || strings.TrimSpace(fields[4]) != "lake" {
				continue
			}
			if id, err := strconv.Atoi(strings.TrimSpace(fields[0])); err == nil && id > 0 {
				d.lakes = append(d.lakes, id)
			}
		}
	}
	for _, t := range d.CountryTags {
		d.tagSet[t] = true
	}
	return d
}

func parseGameFile(game *gamefiles.Game, name string) []*paradox.Entry {
	raw, err := game.ReadFile(name)
	if err != nil {
		return nil
	}
	block, _, err := paradox.Parse(strings.TrimPrefix(string(raw), "\uFEFF"))
	if err != nil || block == nil {
		return nil
	}
	return block.Entries
}
