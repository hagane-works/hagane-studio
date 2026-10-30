package technology

import (
	"io/fs"

	"haganeworks/internal/gamefiles"
	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/paradox"
)

type TechFolderMetadata struct {
	Ledger   string  `json:"ledger"`
	Doctrine bool    `json:"doctrine"`
	DLCGate  *string `json:"dlc_gate"`
}

type gameData struct {
	game func() (fs.FS, error)
}

type (
	techIDsKey      struct{}
	techFoldersKey  struct{}
	techPicturesKey struct{}
)

func newGameData(game func() (fs.FS, error)) *gameData {
	return &gameData{game: game}
}

func (g *gameData) current() *gamefiles.Game {
	if g == nil {
		return nil
	}
	game, err := gamefiles.Open(g.game)
	if err != nil {
		return nil
	}
	return game
}

func (g *gameData) techIDs() map[string]bool {
	game := g.current()
	if game == nil {
		return map[string]bool{}
	}
	return gamefiles.Derived(game, techIDsKey{}, readTechIDs)
}

func (g *gameData) folders() map[string]TechFolderMetadata {
	game := g.current()
	if game == nil {
		return map[string]TechFolderMetadata{}
	}
	return gamefiles.Derived(game, techFoldersKey{}, readTechFolders)
}

func readScript(game *gamefiles.Game, name string) []*paradox.Entry {
	raw, err := game.ReadFile(name)
	if err != nil {
		return nil
	}
	text, ok := gfx.DecodeScriptText(raw)
	if !ok {
		return nil
	}
	block, _, err := paradox.Parse(text)
	if err != nil || block == nil {
		return nil
	}
	return block.Entries
}

func keyIs(entry *paradox.Entry, key string) bool {
	return entry != nil && entry.Key != nil && *entry.Key == key
}

func childBlock(entries []*paradox.Entry, key string) []*paradox.Entry {
	for _, entry := range entries {
		if keyIs(entry, key) && entry.Block != nil {
			return entry.Block.Entries
		}
	}
	return nil
}

func isIdent(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func readTechIDs(game *gamefiles.Game) map[string]bool {
	out := map[string]bool{}
	var collect func(entries []*paradox.Entry)
	collect = func(entries []*paradox.Entry) {
		for _, entry := range entries {
			if entry.Key == nil || entry.Block == nil || !isIdent(*entry.Key) {
				continue
			}
			if *entry.Key != "sub_technologies" {
				out[*entry.Key] = true
			}
			if sub := childBlock(entry.Block.Entries, "sub_technologies"); sub != nil {
				collect(sub)
			}
		}
	}
	for _, name := range game.Files("common/technologies", false, ".txt") {
		collect(childBlock(readScript(game, name), "technologies"))
	}
	return out
}

func readTechFolders(game *gamefiles.Game) map[string]TechFolderMetadata {
	out := map[string]TechFolderMetadata{}
	for _, name := range game.Files("common/technology_tags", false, ".txt") {
		for _, folder := range childBlock(readScript(game, name), "technology_folders") {
			if folder.Key == nil || folder.Block == nil {
				continue
			}
			if _, seen := out[*folder.Key]; seen {
				continue
			}
			meta := TechFolderMetadata{Ledger: "unknown"}
			ledgerFound := false
			var walk func([]*paradox.Entry)
			walk = func(entries []*paradox.Entry) {
				for _, entry := range entries {
					switch {
					case keyIs(entry, "ledger") && entry.Block == nil && !ledgerFound:
						meta.Ledger, ledgerFound = entry.Scalar.StrValue(), true
					case keyIs(entry, "doctrine") && entry.Block == nil && entry.Scalar.StrValue() == "yes":
						meta.Doctrine = true
					case keyIs(entry, "has_dlc") && entry.Block == nil && meta.DLCGate == nil:
						gate := entry.Scalar.StrValue()
						meta.DLCGate = &gate
					}
					if entry.Block != nil {
						walk(entry.Block.Entries)
					}
				}
			}
			walk(folder.Block.Entries)
			out[*folder.Key] = meta
		}
	}
	return out
}

func (g *gameData) iconFacts() (*gamefiles.Game, *techIcons, bool) {
	game := g.current()
	if game == nil {
		return nil, nil, false
	}
	return game, &techIcons{
		sprites:     game.Sprites(),
		techPicture: gamefiles.Derived(game, techPicturesKey{}, readTechArchetypePictures),
	}, true
}

type techIcons struct {
	sprites     map[string]string
	techPicture map[string]string
}

type equipmentFacts struct {
	archetype, picture string
	isArchetype        bool
}

func scalarField(entries []*paradox.Entry, key string) string {
	for _, entry := range entries {
		if keyIs(entry, key) && entry.Block == nil {
			return entry.Scalar.StrValue()
		}
	}
	return ""
}

func readTechArchetypePictures(game *gamefiles.Game) map[string]string {
	equipment := map[string]equipmentFacts{}
	files := append(game.Files("common/units/equipment", false, ".txt"), game.Files("common/units/equipment/modules", false, ".txt")...)
	for _, name := range files {
		for _, wrapper := range readScript(game, name) {
			if wrapper.Key == nil || wrapper.Block == nil {
				continue
			}
			switch *wrapper.Key {
			case "equipments", "equipment_modules", "duplicate_archetypes":
			default:
				continue
			}
			for _, item := range wrapper.Block.Entries {
				if item.Key == nil || item.Block == nil {
					continue
				}
				if _, seen := equipment[*item.Key]; seen {
					continue
				}
				equipment[*item.Key] = equipmentFacts{
					archetype:   scalarField(item.Block.Entries, "archetype"),
					picture:     scalarField(item.Block.Entries, "picture"),
					isArchetype: scalarField(item.Block.Entries, "is_archetype") == "yes",
				}
			}
		}
	}
	var resolve func(id string, depth int) string
	resolve = func(id string, depth int) string {
		if depth > 8 {
			return ""
		}
		item, ok := equipment[id]
		if !ok {
			return ""
		}
		if item.picture != "" {
			return item.picture
		}
		if item.archetype != "" && item.archetype != id {
			return resolve(item.archetype, depth+1)
		}
		if item.isArchetype {
			return "archetype_" + id
		}
		return ""
	}
	out := map[string]string{}
	seen := map[string]bool{}
	for _, name := range game.Files("common/technologies", false, ".txt") {
		for _, tech := range childBlock(readScript(game, name), "technologies") {
			if tech.Key == nil || tech.Block == nil || seen[*tech.Key] {
				continue
			}
			seen[*tech.Key] = true
			for _, equipmentID := range enabledEquipments(tech.Block.Entries) {
				if picture := resolve(equipmentID, 0); picture != "" {
					out[*tech.Key] = picture
					break
				}
			}
		}
	}
	return out
}

func enabledEquipments(entries []*paradox.Entry) []string {
	for _, entry := range entries {
		if keyIs(entry, "enable_equipments") && entry.Block != nil {
			var out []string
			for _, item := range entry.Block.Entries {
				if item.Key == nil && item.Block == nil {
					out = append(out, item.Scalar.StrValue())
				}
			}
			return out
		}
		if entry.Block != nil {
			if nested := enabledEquipments(entry.Block.Entries); nested != nil {
				return nested
			}
		}
	}
	return nil
}

func (f *techIcons) resolveTechIcon(techID string) (rel, resolution string, ok bool) {
	if techID == "" {
		return "", "", false
	}
	if rel := f.sprites["GFX_"+techID+"_medium"]; rel != "" {
		return rel, "direct", true
	}
	if picture := f.techPicture[techID]; picture != "" {
		if rel := f.sprites["GFX_"+picture+"_medium"]; rel != "" {
			return rel, "archetype", true
		}
	}
	return "", "", false
}
