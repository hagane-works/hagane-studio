package guirender

import (
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type doctrineFolder struct {
	Token string

	Name       string
	TabGFX     string
	ColorFrame int
	Allowed    *paradox.Block
}

type doctrineTrack struct {
	Token string
	Name  string

	Background, Icon, IconFrame string
}

type subDoctrine struct {
	Token, Track, Name, Description, Icon string
	XPCost                                float64
	XPType                                string
	Visible                               *paradox.Block

	Rewards   []doctrineReward
	RewardGFX string
}

type doctrineReward struct {
	Key     string
	Mastery float64
}

type doctrineIndex struct {
	folders      map[string]doctrineFolder
	folderOrder  []string
	tracks       map[string]doctrineTrack
	subDoctrines map[string]subDoctrine
	subOrder     []string
}

func (idx *Index) doctrineData() *doctrineIndex {
	if idx.doctrines == nil {
		idx.doctrines = &doctrineIndex{
			folders: map[string]doctrineFolder{}, tracks: map[string]doctrineTrack{}, subDoctrines: map[string]subDoctrine{},
		}
	}
	return idx.doctrines
}

func scalarField(entries []*paradox.Entry, key string) string {
	return strings.Trim(ciScalar(entries, key), "\"")
}

func (idx *Index) indexDoctrineFolderFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	data := idx.doctrineData()
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		folder := doctrineFolder{
			Token:   *entry.Key,
			Name:    scalarField(entry.Block.Entries, "name"),
			TabGFX:  scalarField(entry.Block.Entries, "tab_gfx"),
			Allowed: traitBlock(entry.Block, "allowed"),
		}
		if frame, ok := toFloat(ciScalarValue(entry.Block.Entries, "color_frame")); ok {
			folder.ColorFrame = int(frame)
		}
		if _, seen := data.folders[folder.Token]; !seen {
			data.folderOrder = append(data.folderOrder, folder.Token)
		}
		data.folders[folder.Token] = folder
	}
}

func (idx *Index) indexDoctrineTrackFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	data := idx.doctrineData()
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		data.tracks[*entry.Key] = doctrineTrack{
			Token:      *entry.Key,
			Name:       scalarField(entry.Block.Entries, "name"),
			Background: scalarField(entry.Block.Entries, "background"),
			Icon:       scalarField(entry.Block.Entries, "icon"),
			IconFrame:  scalarField(entry.Block.Entries, "icon_frame"),
		}
	}
}

func (idx *Index) indexSubDoctrineFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	data := idx.doctrineData()
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		sub := subDoctrine{
			Token:       *entry.Key,
			Track:       scalarField(entry.Block.Entries, "track"),
			Name:        scalarField(entry.Block.Entries, "name"),
			Description: scalarField(entry.Block.Entries, "description"),
			Icon:        scalarField(entry.Block.Entries, "icon"),
			XPType:      scalarField(entry.Block.Entries, "xp_type"),
			Visible:     traitBlock(entry.Block, "visible"),
			RewardGFX:   scalarField(entry.Block.Entries, "reward_gfx"),
		}
		if rewards := traitBlock(entry.Block, "rewards"); rewards != nil {
			for _, step := range rewards.Entries {
				if step == nil || step.Block == nil {
					continue
				}
				reward := doctrineReward{}
				if step.Key != nil {
					reward.Key = *step.Key
				}
				if mastery, ok := toFloat(ciScalarValue(step.Block.Entries, "mastery")); ok {
					reward.Mastery = mastery
				}
				sub.Rewards = append(sub.Rewards, reward)
			}
		}
		if cost, ok := toFloat(ciScalarValue(entry.Block.Entries, "xp_cost")); ok {
			sub.XPCost = cost
		}
		if sub.Track == "" {
			continue
		}
		if _, seen := data.subDoctrines[sub.Token]; !seen {
			data.subOrder = append(data.subOrder, sub.Token)
		}
		data.subDoctrines[sub.Token] = sub
	}
}

func (idx *Index) layersBottomUp() []*Index {
	layers := []*Index{}
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append([]*Index{layer}, layers...)
	}
	return layers
}

func (idx *Index) doctrineFolders() []doctrineFolder {
	out := []doctrineFolder{}
	seen := map[string]bool{}
	for _, layer := range idx.layersBottomUp() {
		if layer.doctrines == nil {
			continue
		}
		for _, token := range layer.doctrines.folderOrder {
			if seen[token] {
				continue
			}
			seen[token] = true
			for top := idx; top != nil; top = top.fallback {
				if top.doctrines != nil {
					if folder, ok := top.doctrines.folders[token]; ok {
						out = append(out, folder)
						break
					}
				}
			}
		}
	}
	return out
}

func (idx *Index) doctrineTrackFor(token string) (doctrineTrack, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if layer.doctrines != nil {
			if track, ok := layer.doctrines.tracks[token]; ok {
				return track, true
			}
		}
	}
	return doctrineTrack{}, false
}

func (idx *Index) subDoctrineFor(token string) (subDoctrine, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if layer.doctrines != nil {
			if sub, ok := layer.doctrines.subDoctrines[token]; ok {
				return sub, true
			}
		}
	}
	return subDoctrine{}, false
}

func (idx *Index) subDoctrinesOf(track string) []subDoctrine {
	out := []subDoctrine{}
	seen := map[string]bool{}
	for _, layer := range idx.layersBottomUp() {
		if layer.doctrines == nil {
			continue
		}
		for _, token := range layer.doctrines.subOrder {
			if seen[token] {
				continue
			}
			seen[token] = true
			if sub, ok := idx.subDoctrineFor(token); ok && sub.Track == track {
				out = append(out, sub)
			}
		}
	}
	return out
}

func (idx *Index) grandDoctrinesOf(folder string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, layer := range idx.layersBottomUp() {
		for _, token := range layer.grandDoctrineOrder {
			if seen[token] {
				continue
			}
			seen[token] = true
			if grand, ok := idx.grandDoctrineFor(token); ok && grand.Folder == folder {
				out = append(out, token)
			}
		}
	}
	return out
}

func (idx *Index) StartingSubDoctrines(tag, date string) map[string]string {
	out := map[string]string{}
	politics := idx.countryPoliticsFor(tag)
	if politics == nil {
		return out
	}
	for _, token := range tokensByOwned(politics.SubDoctrines, date, idx.ownedDLC) {
		if sub, ok := idx.subDoctrineFor(token); ok {
			out[sub.Track] = token
		}
	}
	return out
}

func collectMastery(entries []*paradox.Entry, date string) []datedVariable {
	return collectMasteryGuarded(entries, date, dlcGuard{})
}

func collectMasteryGuarded(entries []*paradox.Entry, date string, guard dlcGuard) []datedVariable {
	out := []datedVariable{}
	branch := branchState{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		switch {
		case key == "if" || key == "else_if":
			if inner, ok := branch.enterIf(entry.Block.Entries, guard); ok {
				out = append(out, collectMasteryGuarded(entry.Block.Entries, date, inner)...)
			}
		case key == "else":
			if inner, ok := branch.enterElse(); ok {
				out = append(out, collectMasteryGuarded(entry.Block.Entries, date, inner)...)
			}
		case key == "add_mastery":
			sub := scalarField(entry.Block.Entries, "sub_doctrine")
			if amount, ok := toFloat(ciScalarValue(entry.Block.Entries, "amount")); ok && sub != "" {
				out = append(out, datedVariable{DLC: guard, Date: date, Name: sub, Value: amount})
			}
		case isGameDate(key):
			out = append(out, collectMasteryGuarded(entry.Block.Entries, key, guard)...)
		}
	}
	return out
}

func (idx *Index) StartingMastery(tag, date string) map[string]float64 {
	out := map[string]float64{}
	politics := idx.countryPoliticsFor(tag)
	if politics == nil {
		return out
	}
	if date == "" {
		date = "1936.1.1"
	}
	for _, mastery := range politics.Mastery {
		if dateReached(mastery.Date, date) && mastery.DLC.holds(idx.ownedDLC) {
			out[mastery.Name] += mastery.Value
		}
	}
	return out
}

func (idx *Index) doctrineRewards(sub subDoctrine, mastery float64, localise func(string) string) ([]map[string]any, bool) {
	fallback := idx.defineNumber("NDoctrines.DEFAULT_REWARD_MASTERY")
	if fallback <= 0 {
		fallback = 100
	}
	out := make([]map[string]any, 0, len(sub.Rewards))
	total := 0.0
	complete := len(sub.Rewards) > 0
	for i, reward := range sub.Rewards {
		need := reward.Mastery
		if need <= 0 {
			need = fallback
		}
		progress := (mastery - total) / need
		if progress < 0 {
			progress = 0
		} else if progress > 1 {
			progress = 1
		}
		total += need
		unlocked := mastery >= total
		complete = complete && unlocked
		label := ""
		if reward.Key != "" {
			label = localise(sub.Token + "_" + reward.Key)
		}
		out = append(out, map[string]any{
			"token": reward.Key, "label": label, "level": i + 1,
			"mastery": int(total), "unlocked": unlocked, "progress": progress,
		})
	}
	return out, complete
}

func (idx *Index) DoctrinesPayload(tag, date, folder string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	state := idx.decisionStartOf(tag, date)
	state.from = tag
	localise := func(key string) string {
		if text, ok := idx.LocaliseFor(key, tag, date); ok {
			return text
		}
		return key
	}
	sprites := map[string]any{}
	sprite := func(name string) {
		if name != "" {
			if _, done := sprites[name]; !done {
				sprites[name] = idx.spritePayload(name)
			}
		}
	}
	folders := []map[string]any{}
	colorFrame := 1
	for _, f := range idx.doctrineFolders() {
		if f.Allowed != nil {
			if holds, decided := state.verdict(f.Allowed); decided && !holds {
				continue
			}
		}
		if folder == "" {
			folder = f.Token
		}
		if f.Token == folder && f.ColorFrame > 0 {
			colorFrame = f.ColorFrame
		}
		sprite(f.TabGFX)
		folders = append(folders, map[string]any{
			"token": f.Token, "label": localise(f.Name), "tab": f.TabGFX, "color_frame": f.ColorFrame,
		})
	}
	masteryFormat, _ := idx.locValue("DOCTRINE_MASTERY")
	raw := func(key string) string {
		text, _ := idx.locValue(key)
		return text
	}
	start, _ := idx.countryStartAt(tag, date)
	out := map[string]any{"tag": tag, "folder": folder, "folders": folders, "color_frame": colorFrame,

		"texts": map[string]any{
			"no_grand":     localise("NO_GRAND_DOCTRINE_PROMPT"),
			"select_grand": localise("GRAND_DOCTRINE_SELECTION_TITLE"),
			"select_sub":   localise("SUB_DOCTRINE_SELECTION_TITLE"),

			"mastery": masteryFormat,

			"grand_desc": localise("GRAND_DOCTRINE_DESCRIPTION"),
			"sub_desc":   localise("SUBDOCTRINE_DESCRIPTION"),
			"army_cost":  raw("DOCTRINE_ARMY_XP_COST"),
			"navy_cost":  raw("DOCTRINE_NAVY_XP_COST"),
			"air_cost":   raw("DOCTRINE_AIR_XP_COST"),
		},

		"experience": map[string]any{"army": start.ArmyXP, "navy": start.NavyXP, "air": start.AirXP},
	}
	grandChoices := []map[string]any{}
	for _, token := range idx.grandDoctrinesOf(folder) {
		grand, _ := idx.grandDoctrineFor(token)
		sprite(grand.Icon)
		grandChoices = append(grandChoices, map[string]any{
			"token": token, "label": localise(grand.Name), "icon": grand.Icon, "xp_cost": grand.XPCost, "xp_type": grand.XPType,
		})
	}
	out["grand_choices"] = grandChoices
	chosen := idx.StartingSubDoctrines(tag, date)
	mastery := idx.StartingMastery(tag, date)
	if token := idx.StartingGrandDoctrines(tag, date)[folder]; token != "" {
		if grand, ok := idx.grandDoctrineFor(token); ok {
			sprite(grand.Icon)
			grandOut := map[string]any{"token": token, "label": localise(grand.Name), "icon": grand.Icon}
			if grand.MaxTrackColumns > 0 {
				grandOut["max_track_columns"] = grand.MaxTrackColumns
			}
			if grand.MaxTrackRows > 0 {
				grandOut["max_track_rows"] = grand.MaxTrackRows
			}
			out["grand"] = grandOut
			tracks := []map[string]any{}
			for _, key := range grand.Tracks {
				track, _ := idx.doctrineTrackFor(key)
				sprite(track.Background)
				sprite(track.Icon)
				sprite(track.IconFrame)
				item := map[string]any{
					"token": key, "label": localise(firstNonEmpty(track.Name, key)),
					"background": track.Background, "icon": track.Icon, "icon_frame": track.IconFrame, "mastery": 0,
				}
				if subToken := chosen[key]; subToken != "" {
					if sub, ok := idx.subDoctrineFor(subToken); ok {
						sprite(sub.Icon)
						item["sub"] = map[string]any{"token": subToken, "label": localise(sub.Name), "icon": sub.Icon}
						item["mastery"] = int(mastery[subToken])
						steps, complete := idx.doctrineRewards(sub, mastery[subToken], localise)
						item["rewards"] = steps

						item["complete"] = complete
						sprite(firstNonEmpty(sub.RewardGFX, "GFX_mastery_reward_icons"))
						sprite("GFX_mastery_progress_bar_bg")
						sprite("GFX_mastery_progress_bar_animated_" + folder)
					}
				}
				choices := []map[string]any{}
				for _, sub := range idx.subDoctrinesOf(key) {
					if sub.Visible != nil {
						if holds, decided := state.verdict(sub.Visible); decided && !holds {
							continue
						}
					}
					sprite(sub.Icon)
					choices = append(choices, map[string]any{
						"token": sub.Token, "label": localise(sub.Name), "icon": sub.Icon, "xp_cost": sub.XPCost, "xp_type": sub.XPType,
					})
				}
				item["choices"] = choices
				tracks = append(tracks, item)
			}
			out["tracks"] = tracks
		}
	}
	out["sprites"] = sprites
	return out
}
