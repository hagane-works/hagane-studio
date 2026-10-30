package build

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/paradox"
)

var (
	techFoldersRe = regexp.MustCompile(`technology_folders\s*=\s*\{`)
	wordBlockRe   = regexp.MustCompile(`(\w+)\s*=\s*\{`)
	ledgerRe      = regexp.MustCompile(`ledger\s*=\s*(\w+)`)
	doctrineRe    = regexp.MustCompile(`doctrine\s*=\s*yes`)
	dlcGateRe     = regexp.MustCompile(`has_dlc\s*=\s*"([^"]+)"`)
)

type techFolder struct {
	Ledger   string  `json:"ledger"`
	Doctrine bool    `json:"doctrine"`
	DLCGate  *string `json:"dlc_gate"`
}

func techFoldersOf(text string) ([]string, map[string]techFolder) {
	m := techFoldersRe.FindStringIndex(text)
	if m == nil {
		return nil, nil
	}
	end, _ := walkBraces(text, m[1])
	block := text[m[1]:max(end-1, m[1])]
	var order []string
	out := map[string]techFolder{}
	pos := 0
	for pos < len(block) {
		sub := wordBlockRe.FindStringSubmatchIndex(block[pos:])
		if sub == nil {
			break
		}
		id := block[pos+sub[2] : pos+sub[3]]
		start := pos + sub[1]
		j, _ := walkBraces(block, start)
		body := block[start:max(j-1, start)]
		folder := techFolder{Ledger: "unknown", Doctrine: doctrineRe.MatchString(body)}
		if l := ledgerRe.FindStringSubmatch(body); l != nil {
			folder.Ledger = l[1]
		}
		if d := dlcGateRe.FindStringSubmatch(body); d != nil {
			gate := d[1]
			folder.DLCGate = &gate
		}
		if _, seen := out[id]; !seen {
			order = append(order, id)
			out[id] = folder
		}
		pos = j
	}
	return order, out
}

func buildTechTags(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	categories := map[string]bool{}
	folders := map[string]techFolder{}
	for _, rel := range baseFiles(in, "common/technology_tags", ".txt") {
		text, _ := readBase(in, rel)
		_, found := techFoldersOf(text)
		for id, folder := range found {
			if _, seen := folders[id]; !seen {
				folders[id] = folder
			}
		}
		for _, item := range childBlock(parseEntries(text), "technology_categories") {
			if item.Key == nil && item.Block == nil {
				if id := strings.TrimSpace(item.Scalar.StrValue()); id != "" {
					categories[id] = true
				}
			}
		}
	}
	if len(folders) == 0 && len(categories) == 0 {
		return nil, missingFile("common/technology_tags")
	}
	categoryIDs, folderIDs := sortedKeys(categories), sortedKeys(folders)
	return marshal(map[string]any{
		"source":       "common/technology_tags/*.txt",
		"game_version": in.GameVersion,
		"categories":   map[string]any{"total": len(categoryIDs), "ids": categoryIDs},
		"folders":      map[string]any{"total": len(folderIDs), "ids": folderIDs},
		"folder_meta":  folders,
	})
}

type techTags struct {
	Categories struct {
		IDs []string `json:"ids"`
	} `json:"categories"`
	Folders struct {
		IDs []string `json:"ids"`
	} `json:"folders"`
}

func techTagsOf(raw []byte) techTags {
	var t techTags
	_ = json.Unmarshal(raw, &t)
	return t
}

type equipmentFacts struct {
	archetype, picture string
	isArchetype        bool
}

func buildEquipment(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	g := game(in)
	equipment := map[string]equipmentFacts{}
	archetypes := map[string]bool{}
	files := append(g.Files("common/units/equipment", false, ".txt"), g.Files("common/units/equipment/modules", false, ".txt")...)
	for _, name := range files {
		for _, wrapper := range readScript(g, name) {
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
				isArchetype := false
				for _, field := range item.Block.Entries {
					if keyIs(field, "is_archetype") && field.Block == nil && field.Scalar.StrValue() == "yes" {
						isArchetype = true
					}
				}
				if isArchetype && *wrapper.Key == "equipments" && path.Dir(name) == "common/units/equipment" {
					archetypes[*item.Key] = true
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
	if len(equipment) == 0 {
		return nil, missingFile("common/units/equipment")
	}
	var resolve func(id string, depth int) string
	resolve = func(id string, depth int) string {
		item, ok := equipment[id]
		if !ok || depth > 8 {
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
	pictures := map[string]string{}
	seen := map[string]bool{}
	for _, name := range g.Files("common/technologies", false, ".txt") {
		for _, tech := range childBlock(readScript(g, name), "technologies") {
			if tech.Key == nil || tech.Block == nil || seen[*tech.Key] {
				continue
			}
			seen[*tech.Key] = true
			for _, id := range enabledEquipments(tech.Block.Entries) {
				if picture := resolve(id, 0); picture != "" {
					pictures[*tech.Key] = picture
					break
				}
			}
		}
	}
	list := sortedKeys(archetypes)
	return marshal(map[string]any{
		"source":                    "common/units/equipment/*.txt · is_archetype=yes top-level keys",
		"game_version":              in.GameVersion,
		"total":                     len(list),
		"archetypes":                list,
		"tech_to_archetype_picture": pictures,
	})
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

type equipmentData struct {
	Archetypes []string          `json:"archetypes"`
	Pictures   map[string]string `json:"tech_to_archetype_picture"`
}

func equipmentOf(raw []byte) equipmentData {
	var e equipmentData
	_ = json.Unmarshal(raw, &e)
	return e
}

var techIconRe = regexp.MustCompile(`(?is)SpriteType\s*=\s*\{[^}]*?\bname\s*=\s*"(GFX_[A-Za-z0-9_]+_medium)"[^}]*?\btexturefile\s*=\s*"([^"]+)"`)

func buildTechIconPaths(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	paths := map[string]string{}
	add := func(rel string) {
		text, ok := readBase(in, rel)
		if !ok {
			return
		}
		for _, m := range techIconRe.FindAllStringSubmatch(stripComments(text), -1) {
			id := strings.TrimSuffix(strings.TrimPrefix(m[1], "GFX_"), "_medium")
			if _, seen := paths[id]; !seen {
				paths[id] = strings.ReplaceAll(m[2], "\\", "/")
			}
		}
	}
	add("interface/Technologies.gfx")
	for _, dir := range layerDirs(in.FS, "dlc") {
		if !strings.HasPrefix(path.Base(dir), "dlc") {
			continue
		}
		for _, rel := range listFiles(in.FS, dir+"/interface", ".gfx") {
			if strings.Contains(path.Base(rel), "technologies") {
				add(rel)
			}
		}
	}
	return marshal(map[string]any{
		"source": "Technologies.gfx + dlc/*/interface/*_technologies.gfx", "game_version": in.GameVersion,
		"total": len(paths), "paths": paths,
	})
}

var (
	techSizeSpriteRe = regexp.MustCompile(`(?i)name\s*=\s*"?(GFX_[\w\-]+)"?\s+texturefile\s*=\s*"([^"]+)"`)
	techMediumRe     = regexp.MustCompile(`^GFX_(.+)_medium$`)
	tagPrefixRe      = regexp.MustCompile(`^[A-Z]{3}_`)
)

func buildTechIconSizes(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	files := []string{"interface/Technologies.gfx"}
	for _, dir := range layerDirs(in.FS, "dlc") {
		for _, rel := range listFiles(in.FS, dir+"/interface", ".gfx") {
			if strings.HasSuffix(path.Base(rel), "_technologies.gfx") {
				files = append(files, rel)
			}
		}
	}
	type def struct{ texture string }
	var order []string
	defs := map[string][]def{}
	for _, rel := range files {
		raw, err := fs.ReadFile(in.FS, rel)
		if err != nil {
			continue
		}
		for _, m := range techSizeSpriteRe.FindAllStringSubmatch(string(raw), -1) {
			if _, ok := defs[m[1]]; !ok {
				order = append(order, m[1])
			}
			defs[m[1]] = append(defs[m[1]], def{strings.ReplaceAll(m[2], "\\", "/")})
		}
	}
	roots := append([]string{""}, layerDirs(in.FS, "dlc")...)
	dims := func(rel string) []int {
		for _, root := range roots {
			name := path.Join(root, rel)
			f, err := in.FS.Open(name)
			if err != nil {
				continue
			}
			head := make([]byte, 24)
			n, _ := io.ReadFull(f, head)
			f.Close()
			head = head[:n]
			switch {
			case len(head) >= 20 && string(head[:4]) == "DDS ":
				h := binary.LittleEndian.Uint32(head[12:16])
				w := binary.LittleEndian.Uint32(head[16:20])
				return []int{int(w), int(h)}
			case len(head) >= 24 && string(head[:8]) == "\x89PNG\r\n\x1a\n":
				return []int{int(binary.BigEndian.Uint32(head[16:20])), int(binary.BigEndian.Uint32(head[20:24]))}
			}
		}
		return nil
	}
	sizes := map[string][]int{}
	for _, name := range order {
		m := techMediumRe.FindStringSubmatch(name)
		if m == nil || tagPrefixRe.MatchString(m[1]) {
			continue
		}
		for _, d := range defs[name] {
			if size := dims(d.texture); size != nil {
				if size[0] != 64 || size[1] != 64 {
					sizes[m[1]] = size
				}
				break
			}
		}
	}
	return marshal(map[string]any{
		"source": "HOI4 interface/Technologies.gfx + dlc/*/interface/*_technologies.gfx, GFX_<tech>_medium texture sizes; techs not listed use 64x64",
		"sizes":  sizes,
	})
}
