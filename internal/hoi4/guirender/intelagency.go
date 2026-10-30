package guirender

import (
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

type intelAgencyRecord struct {
	Picture string
	Names   []string

	Default   []*paradox.Entry
	Available []*paradox.Entry

	Ordinal int
	Line    int
}

type intelBranchRecord struct {
	Name     string
	Upgrades []intelUpgradeRecord
	Order    int
}

type intelUpgradeRecord struct {
	Name    string
	Picture string
	Frame   string

	Levels    int
	Factories float64

	Available []*paradox.Entry

	LevelScripts []string
	File         string
	Line         int
}

func (idx *Index) indexIntelAgencyFile(name string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	if idx.intelAgencies == nil {
		idx.intelAgencies = map[string][]*intelAgencyRecord{}
	}
	records := []*intelAgencyRecord{}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "intelligence_agency") {
			continue
		}
		record := &intelAgencyRecord{Ordinal: len(records), Line: entry.Line}
		record.Picture = strings.Trim(ciScalar(entry.Block.Entries, "picture"), "\"")
		if names := childBlock(entry.Block.Entries, "names"); names != nil {
			record.Names = blockTokens(names.Entries)
		}

		if block := lastChildBlock(entry.Block.Entries, "default"); block != nil {
			record.Default = append([]*paradox.Entry{}, block.Entries...)
		}
		if block := lastChildBlock(entry.Block.Entries, "available"); block != nil {
			record.Available = append([]*paradox.Entry{}, block.Entries...)
		}
		records = append(records, record)
	}
	idx.intelAgencies[name] = records
}

func onlyAlwaysYes(entries []*paradox.Entry) bool {
	seen := false
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		if entry.Block != nil || !strings.EqualFold(*entry.Key, "always") || !strings.EqualFold(toString(scalarValue(entry.Scalar)), "yes") {
			return false
		}
		seen = true
	}
	return seen
}

func lastChildBlock(entries []*paradox.Entry, key string) *paradox.Block {
	var found *paradox.Block
	for _, entry := range entries {
		if entry != nil && entry.Key != nil && entry.Block != nil && strings.EqualFold(*entry.Key, key) {
			found = entry.Block
		}
	}
	return found
}

func (idx *Index) indexIntelUpgradeFile(file string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	if idx.intelBranches == nil {
		idx.intelBranches = map[string]*intelBranchRecord{}
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		idx.intelBranchSeq++
		branch := &intelBranchRecord{Name: *entry.Key, Order: idx.intelBranchSeq}
		for _, item := range entry.Block.Entries {
			if item == nil || item.Key == nil || item.Block == nil {
				continue
			}
			upgrade := intelUpgradeRecord{
				Name:    *item.Key,
				Picture: strings.Trim(ciScalar(item.Block.Entries, "picture"), "\""),
				Frame:   strings.Trim(ciScalar(item.Block.Entries, "frame"), "\""),
				File:    file,
				Line:    item.Line,
			}
			for _, part := range item.Block.Entries {
				if part != nil && part.Key != nil && part.Block != nil && strings.EqualFold(*part.Key, "level") {
					upgrade.Levels++
					upgrade.LevelScripts = append(upgrade.LevelScripts, conditionScript(part.Block.Entries))
				}
			}
			if progress := childBlock(item.Block.Entries, "modifiers_during_progress"); progress != nil {
				upgrade.Factories = numberBlock(progress)["civilian_factory_use"]
			}
			if available := childBlock(item.Block.Entries, "available"); available != nil {
				upgrade.Available = append([]*paradox.Entry{}, available.Entries...)
			}
			branch.Upgrades = append(branch.Upgrades, upgrade)
		}
		idx.intelBranches[strings.ToLower(branch.Name)] = branch
	}
}

func conditionScript(entries []*paradox.Entry) string {
	if len(entries) == 0 {
		return ""
	}
	return strings.TrimSpace(importsvc.ToScriptEntries(entries, 0))
}

func (idx *Index) intelAgencyFiles() []struct {
	Path    string
	Records []*intelAgencyRecord
} {
	seen := map[string]bool{}
	type file = struct {
		Path    string
		Records []*intelAgencyRecord
	}
	out := []file{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for path, records := range layer.intelAgencies {
			key := strings.ToLower(path)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, file{Path: path, Records: records})
		}
	}
	sort.Slice(out, func(a, b int) bool { return strings.ToLower(out[a].Path) < strings.ToLower(out[b].Path) })
	return out
}

func (idx *Index) intelBranchList() []*intelBranchRecord {
	var layers []*Index
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
	}
	byName := map[string]*intelBranchRecord{}
	order := map[string]int{}
	next := 0
	for at := len(layers) - 1; at >= 0; at-- {
		branches := make([]*intelBranchRecord, 0, len(layers[at].intelBranches))
		for _, branch := range layers[at].intelBranches {
			branches = append(branches, branch)
		}
		sort.Slice(branches, func(a, b int) bool { return branches[a].Order < branches[b].Order })
		for _, branch := range branches {
			key := strings.ToLower(branch.Name)
			if _, seen := order[key]; !seen {
				next++
				order[key] = next
			}
			byName[key] = branch
		}
	}
	out := make([]*intelBranchRecord, 0, len(byName))
	for _, branch := range byName {
		out = append(out, branch)
	}
	sort.Slice(out, func(a, b int) bool { return order[strings.ToLower(out[a].Name)] < order[strings.ToLower(out[b].Name)] })
	return out
}

func (idx *Index) IntelUpgradeSource(name string) (string, []byte, int, bool) {
	for _, branch := range idx.intelBranchList() {
		for _, upgrade := range branch.Upgrades {
			if !strings.EqualFold(upgrade.Name, name) || upgrade.File == "" {
				continue
			}
			for layer := idx; layer != nil; layer = layer.fallback {
				if body, ok := layer.files[upgrade.File]; ok {
					return upgrade.File, body, upgrade.Line, true
				}
			}
		}
	}
	return "", nil, 0, false
}

func (idx *Index) IntelAgencySource(path string, ordinal int) (string, []byte, int, bool) {
	for _, file := range idx.intelAgencyFiles() {
		if !strings.EqualFold(file.Path, path) || ordinal < 0 || ordinal >= len(file.Records) {
			continue
		}
		for layer := idx; layer != nil; layer = layer.fallback {
			if records, ok := layer.intelAgencies[file.Path]; ok && len(records) > ordinal && records[ordinal] == file.Records[ordinal] {
				return file.Path, layer.files[file.Path], records[ordinal].Line, true
			}
		}
	}
	return "", nil, 0, false
}

func (idx *Index) IntelAgencyPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	state := idx.decisionStartOf(tag, date)
	holds := func(conditions []*paradox.Entry) bool {
		value, decided := state.eval(conditions, false)
		return decided && value
	}
	insignia := []map[string]any{}

	chosen, generic := -1, -1
	for _, file := range idx.intelAgencyFiles() {
		for _, record := range file.Records {
			if record.Available != nil && !holds(record.Available) {
				continue
			}
			names := make([]string, 0, len(record.Names))
			for _, name := range record.Names {
				names = append(names, idx.localiseOr(name, name))
			}
			isDefault := record.Default != nil && holds(record.Default)
			if isDefault && onlyAlwaysYes(record.Default) {
				if generic < 0 {
					generic = len(insignia)
				}
			} else if isDefault && chosen < 0 {
				chosen = len(insignia)
			}
			insignia = append(insignia, map[string]any{
				"file": file.Path, "ordinal": record.Ordinal, "picture": record.Picture,
				"names": names, "raw_names": record.Names, "default": isDefault,

				"available_script": conditionScript(record.Available),
				"default_script":   conditionScript(record.Default),
			})
		}
	}
	pictures := make([]string, 0, len(insignia))
	for _, item := range insignia {
		pictures = append(pictures, item["picture"].(string))
	}
	if chosen < 0 {
		chosen = generic
	}
	if chosen < 0 && len(insignia) > 0 {
		chosen = 0
	}
	name := ""
	if chosen >= 0 {
		if names := insignia[chosen]["names"].([]string); len(names) > 0 {
			name = names[0]
		}
	}
	branches := []map[string]any{}
	for _, branch := range idx.intelBranchList() {
		upgrades := []map[string]any{}
		for _, upgrade := range branch.Upgrades {
			item := map[string]any{
				"id": upgrade.Name, "label": idx.localiseOr(upgrade.Name, upgrade.Name),
				"picture": upgrade.Picture, "frame": upgrade.Frame,
				"levels": upgrade.Levels, "factories": upgrade.Factories,

				"available": upgrade.Available == nil || holds(upgrade.Available),

				"available_script": conditionScript(upgrade.Available),
				"levels_script":    upgrade.LevelScripts,
				"file":             upgrade.File,
			}

			if upgrade.Frame != "" {
				item["frame_sprite"] = upgrade.Frame + "_unavailable"
				pictures = append(pictures, upgrade.Frame+"_unavailable")
			}
			if upgrade.Picture != "" {
				pictures = append(pictures, upgrade.Picture)
			}
			upgrades = append(upgrades, item)
		}
		branches = append(branches, map[string]any{
			"id": branch.Name, "label": idx.localiseOr(branch.Name, branch.Name), "upgrades": upgrades,
		})
	}
	days := idx.defineNumber("NOperatives.AGENCY_CREATION_DAYS")

	daysFormat, ok := idx.locValue("EVENT_AMOUNT_DAYS")
	if !ok || !strings.Contains(daysFormat, "$AMOUNT$") {
		daysFormat = "$AMOUNT$ days"
	}
	created := false
	if politics := idx.countryPoliticsFor(tag); politics != nil {
		for _, call := range effectsRunByOwned(politics.Effects, date, idx.ownedDLC) {
			if call.Command == "create_intelligence_agency" && strings.EqualFold(call.Token, "yes") {
				created = true
			}
		}
	}
	defense, ok := idx.locValue("AGENCY_DEFENSE_LEVEL")
	if !ok {
		defense = "Counter Intelligence: $LEVEL|H$"
	}
	rating := strconv.FormatFloat(idx.defineNumber("NOperatives.BASE_COUNTER_INTELLIGENCE_RATING"), 'f', 2, 64)
	slots := 0.0
	if block := idx.staticModifierFor("created_intelligence_agency"); block != nil {
		slots = numberBlock(block)["operative_slot"]
	}
	recruit := idx.defineNumber("NOperatives.AGENCY_OPERATIVE_RECRUITMENT_TIME")
	pictures = append(pictures, "GFX_agency_upgrade_researched_icon", "GFX_agency_upgrade_repetition_icon", "GFX_agency_upgrade_btn_unavailable")
	duration, ok := idx.locValue("AGENCY_UPGRADES_DURATION")
	if !ok {
		duration = "Branch Upgrades take $DAYS|H$ days to complete."
	}
	return map[string]any{
		"tag": tag, "insignia": insignia, "chosen": chosen, "name": name, "branches": branches,
		"sprites": idx.SpritesPayload(pictures),

		"created":         created,
		"operative_slots": slots,
		"creation": map[string]any{
			"factories": idx.defineNumber("NOperatives.AGENCY_CREATION_FACTORIES"),
			"days":      days,

			"days_text": strings.Replace(daysFormat, "$AMOUNT$", "§Y"+strconv.Itoa(int(days))+"§!", 1),
		},
		"texts": map[string]any{
			"create":       idx.localiseOr("CREATE_AGENCY_BUTTON", "Create Agency"),
			"forming":      idx.localiseOr("CREATE_AGENCY_BUTTON_PROGRESS", "Agency being formed..."),
			"no_operative": idx.localiseOr("STATUS_NO_AGENCY_NO_OPERATIVE", "Operatives can't be recruited until you have created the Agency."),
			"choose":       idx.localiseOr("CHOOSE_INSIGNIA", "Choose Insignia"),

			"defense_level": locSlot.ReplaceAllString(defense, "§H"+rating+"§!"),
			"no_upgrade":    idx.localiseOr("STATUS_NO_AGENCY_UPGRADE", "No upgrade in progress."),
			"recruiting":    idx.localiseOr("STATUS_RECRUIT_OPERATIVE", "Recruiting operative"),
			"recruit_days":  strings.Replace(daysFormat, "$AMOUNT$", strconv.Itoa(int(recruit)), 1),

			"upgrades_duration": locSlot.ReplaceAllString(duration, "§H"+strconv.Itoa(int(idx.defineNumber("NOperatives.AGENCY_UPGRADE_DAYS")))+"§!"),
		},
	}
}
