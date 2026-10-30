package gamesave

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"haganeworks/internal/project"
)

const agenciesTable = "intelligence_agencies"

type intelSaveIndex interface {
	IntelAgencySource(path string, ordinal int) (string, []byte, int, bool)
}

type intelSaveInput struct {
	File    string `json:"file"`
	Ordinal int    `json:"ordinal"`

	Picture  string   `json:"picture"`
	OldNames []string `json:"old_names"`

	Names     *[]string `json:"names"`
	Available *string   `json:"available"`
	Default   *string   `json:"default"`
}

type intelCreateInput struct {
	Tag     string `json:"tag"`
	Picture string `json:"picture"`
	Name    string `json:"name"`
}

const intelRawFileNote = "游戏界面情报机构徽记(整份文件收进项目)"

var (
	intelPicturePattern = regexp.MustCompile(`^GFX_[A-Za-z0-9_]+$`)
	intelTagPattern     = regexp.MustCompile(`^[A-Z][A-Z0-9]{2}$`)
	intelIDClean        = regexp.MustCompile(`[^a-z0-9]+`)
)

func intelNames(names []string) ([]string, error) {
	out := []string{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if len([]rune(name)) > 80 || strings.ContainsAny(name, "\"{}#\r\n") {
			return nil, fmt.Errorf("a name cannot hold quotes, braces or #: %s", name)
		}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil, errors.New("an insignia needs at least one name")
	}
	return out, nil
}

func intelNamesText(names []string) string {
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, `"`+name+`"`)
	}
	return "names = { " + strings.Join(quoted, " ") + " }"
}

var errIntelChanged = errors.New("the insignia file has changed; reopen the window")

func intelEdit(text string, line int, picture string, names *[]string, available, deflt *string) (string, error) {
	words := scriptWords(text)
	open, close, err := blockBounds(words, text, line, "intelligence_agency")
	if err != nil {
		return "", err
	}
	if at, found := assignments(words, open, close)["picture"]; !found || strings.Trim(words[at[1]].text, `"`) != picture {
		return "", errIntelChanged
	}
	inner := blockInner(text, words, open, close)
	edits := []edit{}
	added := []string{}
	if names != nil {
		line := intelNamesText(*names)
		if found := childBlocks(words, open, close, "names"); len(found) > 0 {
			edits = append(edits, edit{words[found[0][0]].start, words[found[0][1]].end, line})
		} else {
			added = append(added, line)
		}
	}
	for _, part := range []struct {
		key   string
		value *string
	}{{"available", available}, {"default", deflt}} {
		more, add := scriptBlockEdits(text, words, open, close, part.key, part.value, inner)
		edits = append(edits, more...)
		added = append(added, add...)
	}
	edits = append(edits, insertBeforeClose(text, words, close, added, inner)...)
	return applyEdits(text, edits), nil
}

func sameNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}

func agencyNames(row project.Row) []string {
	var names []string
	decodeColumn(row["names"], &names)
	return names
}

func saveIntelAgency(st *store, index intelSaveIndex, in intelSaveInput) (map[string]any, error) {
	if in.Names == nil && in.Available == nil && in.Default == nil {
		return nil, saveError{422, "nothing to change"}
	}
	if !intelPicturePattern.MatchString(in.Picture) {
		return nil, saveError{422, "an insignia picture is required"}
	}
	if in.Names != nil {
		names, err := intelNames(*in.Names)
		if err != nil {
			return nil, saveError{422, err.Error()}
		}
		in.Names = &names
	}
	for _, condition := range []*string{in.Available, in.Default} {
		if condition == nil {
			continue
		}
		cleaned, err := cleanScript(*condition)
		if err != nil {
			return nil, saveError{422, err.Error()}
		}
		*condition = cleaned
	}
	for _, row := range st.sortedRows(agenciesTable, nil, byText("agency_id")) {
		names := agencyNames(row)
		if len(names) == 0 && text(row, "agency_id") != "" {
			names = []string{text(row, "agency_id")}
		}
		if textOr(row, "source", "created") == "imported" || text(row, "picture") != in.Picture || !sameNames(names, in.OldNames) {
			continue
		}
		changes := map[string]any{}
		if in.Names != nil {
			changes["names"] = jsonValue(*in.Names)
		}
		if in.Available != nil {
			changes["available"] = *in.Available
		}
		if in.Default != nil {
			changes["default_trigger"] = *in.Default
		}
		id := project.Int64(row["id"])
		if _, err := st.update(agenciesTable, id, changes); err != nil {
			return nil, err
		}
		return map[string]any{"row": id, "imported": false}, nil
	}
	file, body, line, ok := index.IntelAgencySource(in.File, in.Ordinal)
	if !ok {
		return nil, saveError{404, "the insignia is not declared where the window found it"}
	}
	dest, err := sanitizedDest(file)
	if err != nil {
		return nil, err
	}
	text, err := intelEdit(string(body), line, in.Picture, in.Names, in.Available, in.Default)
	if errors.Is(err, errIntelChanged) {
		return nil, saveError{409, err.Error()}
	}
	if err != nil {
		return nil, saveError{422, err.Error()}
	}
	if err := gateText(dest, text); err != nil {
		return nil, err
	}
	created, err := st.raw.put(dest, text, intelRawFileNote)
	if err != nil {
		return nil, err
	}
	return map[string]any{"file": dest, "imported": created}, nil
}

func createIntelAgency(st *store, in intelCreateInput) (map[string]any, error) {
	tag := upperTag(in.Tag)
	if !intelTagPattern.MatchString(tag) {
		return nil, saveError{422, "a country tag is required"}
	}
	if !intelPicturePattern.MatchString(in.Picture) {
		return nil, saveError{422, "an insignia picture is required"}
	}
	names, err := intelNames([]string{in.Name})
	if err != nil {
		return nil, saveError{422, err.Error()}
	}
	base := strings.Trim(intelIDClean.ReplaceAllString(strings.ToLower(names[0]), "_"), "_")
	if base == "" {
		base = "agency"
	}
	base = strings.ToLower(tag) + "_" + base
	taken := map[string]bool{}
	for _, row := range st.tx.Rows(agenciesTable) {
		taken[text(row, "agency_id")] = true
	}
	for attempt := 1; attempt <= 20; attempt++ {
		id := base
		if attempt > 1 {
			id = fmt.Sprintf("%s_%d", base, attempt)
		}
		if taken[id] {
			continue
		}
		row := st.tx.Insert(agenciesTable, project.Row{
			"agency_id": id, "picture": in.Picture, "names": jsonValue(names),
			"default_trigger": "", "available": "original_tag = " + tag,
			"upgrade_branches": []any{}, "raw_extras": map[string]any{},
			"country_tag": tag, "scope_kind": "own", "shared_country_tags": nil,
			"source": "created", "name_en": "", "name_zh": "",
			"display_name_locs": map[string]any{}, "description_locs": map[string]any{},
		})
		return map[string]any{"row": project.Int64(row["id"]), "agency_id": id}, nil
	}
	return nil, saveError{409, "no free agency id for this name"}
}

func (h *handlers) intelAgencySave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in intelSaveInput
	if !decodeBody(w, r, 1<<16, &in, "invalid insignia payload") {
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}
	out, err := write(p, []string{rawFilesTable, agenciesTable}, func(st *store) (map[string]any, error) {
		out, err := saveIntelAgency(st, index, in)
		return st.raw.withUndo(out), err
	})
	h.finish(w, p, out, err, "insignia save", true)
}

func (h *handlers) intelAgencyCreate(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in intelCreateInput
	if !decodeBody(w, r, 1<<14, &in, "invalid insignia payload") {
		return
	}
	out, err := write(p, []string{agenciesTable}, func(st *store) (map[string]any, error) {
		return createIntelAgency(st, in)
	})
	h.finish(w, p, out, err, "insignia create", true)
}

type intelUpgradeSaveIndex interface {
	IntelUpgradeSource(name string) (string, []byte, int, bool)
}

type intelUpgradeSaveInput struct {
	ID string `json:"id"`

	Factories *float64 `json:"factories"`
	Available *string  `json:"available"`

	Levels *[]string `json:"levels"`
}

const intelUpgradeRawFileNote = "游戏界面情报机构升级(整份文件收进项目)"

var factoryUseLine = regexp.MustCompile(`(?m)^(\s*civilian_factory_use\s*=\s*)[-0-9.]+`)

func progressText(text string, factories float64) string {
	value := numberText(factories)
	if factoryUseLine.MatchString(text) {
		return factoryUseLine.ReplaceAllString(text, "${1}"+value)
	}
	if strings.TrimSpace(text) == "" {
		return "civilian_factory_use = " + value
	}
	return strings.TrimRight(text, "\n") + "\ncivilian_factory_use = " + value
}

func levelsText(levels []string, indent string) string {
	blocks := make([]string, 0, len(levels))
	for _, level := range levels {
		blocks = append(blocks, conditionText("level", level, indent))
	}
	return strings.Join(blocks, "\n"+indent)
}

func intelUpgradeEdit(text string, line int, name string, in intelUpgradeSaveInput) (string, error) {
	words := scriptWords(text)
	open, close, err := blockBounds(words, text, line, name)
	if err != nil {
		return "", err
	}
	inner := blockInner(text, words, open, close)
	edits := []edit{}
	added := []string{}
	if in.Factories != nil {
		if found := childBlocks(words, open, close, "modifiers_during_progress"); len(found) > 0 {
			at := found[len(found)-1]
			value := *in.Factories
			edits = append(edits, setNumbers(text, words, at[0]+2, at[1], map[string]*float64{"civilian_factory_use": &value}, blockInner(text, words, at[0]+2, at[1]))...)
		} else {
			added = append(added, conditionText("modifiers_during_progress", "civilian_factory_use = "+numberText(*in.Factories), inner))
		}
	}
	more, add := scriptBlockEdits(text, words, open, close, "available", in.Available, inner)
	edits = append(edits, more...)
	added = append(added, add...)
	if in.Levels != nil {
		levels := childBlocks(words, open, close, "level")
		if len(levels) == 0 {
			added = append(added, levelsText(*in.Levels, inner))
		} else {

			first := levels[0]
			edits = append(edits, edit{words[first[0]].start, words[first[1]].end, levelsText(*in.Levels, inner)})
			for _, at := range levels[1:] {
				edits = append(edits, removal(text, words, at[0], at[1]))
			}
		}
	}
	edits = append(edits, insertBeforeClose(text, words, close, added, inner)...)
	return applyEdits(text, edits), nil
}

func saveIntelUpgrade(st *store, index intelUpgradeSaveIndex, in intelUpgradeSaveInput) (map[string]any, error) {
	name := strings.TrimSpace(in.ID)
	if name == "" || strings.ContainsAny(name, " \t\r\n{}=\"#") {
		return nil, saveError{422, "an upgrade id is required"}
	}
	if in.Factories == nil && in.Available == nil && in.Levels == nil {
		return nil, saveError{422, "nothing to change"}
	}
	if in.Factories != nil && (*in.Factories < 0 || *in.Factories > 100) {
		return nil, saveError{422, "civilian factories are 0 to 100"}
	}
	if in.Available != nil {
		cleaned, err := cleanScript(*in.Available)
		if err != nil {
			return nil, saveError{422, err.Error()}
		}
		in.Available = &cleaned
	}
	if in.Levels != nil {
		if len(*in.Levels) == 0 {
			return nil, saveError{422, "an upgrade has at least one level"}
		}
		cleaned := make([]string, 0, len(*in.Levels))
		for _, level := range *in.Levels {
			text, err := cleanScript(level)
			if err != nil {
				return nil, saveError{422, "a level is not valid script"}
			}
			cleaned = append(cleaned, text)
		}
		in.Levels = &cleaned
	}
	for _, row := range st.sortedRows(agenciesTable, nil, byText("agency_id")) {
		if textOr(row, "source", "created") == "imported" {
			continue
		}
		var current []map[string]any
		decodeColumn(row["upgrade_branches"], &current)
		branches, found := upgradeInRow(current, name, in)
		if !found {
			continue
		}
		id := project.Int64(row["id"])
		if _, err := st.update(agenciesTable, id, map[string]any{"upgrade_branches": jsonValue(branches)}); err != nil {
			return nil, err
		}
		return map[string]any{"row": id, "imported": false}, nil
	}
	file, body, line, ok := index.IntelUpgradeSource(name)
	if !ok {
		return nil, saveError{404, "the upgrade is not declared in any upgrades file"}
	}
	dest, err := sanitizedDest(file)
	if err != nil {
		return nil, err
	}
	text, err := intelUpgradeEdit(string(body), line, name, in)
	if err != nil {
		return nil, saveError{422, err.Error()}
	}
	if err := gateText(dest, text); err != nil {
		return nil, err
	}
	created, err := st.raw.put(dest, text, intelUpgradeRawFileNote)
	if err != nil {
		return nil, err
	}
	return map[string]any{"file": dest, "imported": created}, nil
}

func upgradeInRow(branches []map[string]any, name string, in intelUpgradeSaveInput) ([]map[string]any, bool) {
	out := make([]map[string]any, 0, len(branches))
	found := false
	for _, branch := range branches {
		copied := map[string]any{}
		for key, value := range branch {
			copied[key] = value
		}
		upgrades, _ := branch["upgrades"].([]any)
		changed := make([]any, 0, len(upgrades))
		for _, item := range upgrades {
			upgrade, ok := item.(map[string]any)
			token, _ := upgrade["token"].(string)
			if !ok || !strings.EqualFold(strings.TrimSpace(token), name) {
				changed = append(changed, item)
				continue
			}
			found = true
			next := map[string]any{}
			for key, value := range upgrade {
				next[key] = value
			}
			if in.Factories != nil {
				progress, _ := upgrade["modifiers_during_progress"].(string)
				next["modifiers_during_progress"] = progressText(progress, *in.Factories)
			}
			if in.Available != nil {
				next["available"] = *in.Available
			}
			if in.Levels != nil {
				levels := make([]any, 0, len(*in.Levels))
				for _, level := range *in.Levels {
					levels = append(levels, level)
				}
				next["levels"] = levels
			}
			changed = append(changed, next)
		}
		copied["upgrades"] = changed
		out = append(out, copied)
	}
	return out, found
}

func (h *handlers) intelUpgradeSave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in intelUpgradeSaveInput
	if !decodeBody(w, r, 1<<16, &in, "invalid upgrade payload") {
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}
	out, err := write(p, []string{rawFilesTable, agenciesTable}, func(st *store) (map[string]any, error) {
		out, err := saveIntelUpgrade(st, index, in)
		return st.raw.withUndo(out), err
	})
	h.finish(w, p, out, err, "intel upgrade save", true)
}
