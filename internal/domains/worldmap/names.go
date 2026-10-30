package worldmap

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/emit/locout"
	"haganeworks/internal/project"
)

type WorldMapStateNameItemInput struct {
	StateID         int64             `json:"state_id"`
	DisplayNameLocs map[string]string `json:"display_name_locs"`
	CapitalNameLocs map[string]string `json:"capital_name_locs,omitempty"`
}

type WorldMapStateNameReadback struct {
	StateID           int64             `json:"state_id"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	CapitalProvinceID int64             `json:"capital_province_id,omitempty"`
	CapitalNameLocs   map[string]string `json:"capital_name_locs"`
}

type WorldMapStateNameStatus struct {
	Complete    bool     `json:"complete"`
	TotalStates int      `json:"total_states"`
	NamedStates int      `json:"named_states"`
	Languages   []string `json:"languages"`
}

type WorldMapStateNamesResult struct {
	NameStatus        WorldMapStateNameStatus     `json:"name_status"`
	StateCount        int                         `json:"state_count"`
	CapitalCount      int                         `json:"capital_count"`
	LocalizationCount int                         `json:"localization_count"`
	Items             []WorldMapStateNameReadback `json:"items"`
}

type normalizedWorldMapStateNameItem struct {
	stateID         int64
	displayNameLocs map[string]string
	capitalNameLocs map[string]string
}

type worldMapStateNameRow struct {
	id              int64
	displayNameLocs map[string]string
	victoryPoints   string
}

type worldMapCapitalLocUpdate struct {
	Locale, Key, Value, SourceFile string
}

func (h *handlers) names(w http.ResponseWriter, r *http.Request, p *project.Project) {
	var body struct {
		StateNames []WorldMapStateNameItemInput `json:"state_names"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "地图名称请求读不懂"})
		return
	}
	result, err := h.store(p).ApplyWorldMapStateNames(body.StateNames)
	if err != nil {
		var detail *apiutil.StatusError
		if errors.As(err, &detail) {
			apiutil.Fail(w, err)
			return
		}
		log.Printf("world map names %s: %v", p.Dir(), err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "地图名称没有保存,项目没有发生部分修改"})
		return
	}
	h.svc.Forget(p)
	writeJSON(w, http.StatusOK, result)
}

func detailError(status int, detail string) error {
	return &apiutil.StatusError{Status: status, Body: map[string]any{"detail": detail}}
}

func worldMapStateNameInvalid(format string, args ...any) error {
	return detailError(422, fmt.Sprintf(format, args...))
}

func normalizeWorldMapNameLocs(field string, stateID int64, locs map[string]string) (map[string]string, error) {
	out := make(map[string]string, len(locs))
	keys := make([]string, 0, len(locs))
	for key := range locs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, rawLang := range keys {
		lang := strings.ToLower(strings.TrimSpace(rawLang))
		if lang == "" {
			lang = "en"
		}
		if _, _, ok := GUITextTarget(lang); !ok {
			return nil, worldMapStateNameInvalid("state %d %s contains unsupported language %q", stateID, field, rawLang)
		}
		if _, duplicate := out[lang]; duplicate {
			return nil, worldMapStateNameInvalid("state %d %s contains duplicate language %q", stateID, field, lang)
		}
		value := strings.TrimSpace(locs[rawLang])
		if value == "" {
			return nil, worldMapStateNameInvalid("state %d %s[%s] must not be empty", stateID, field, lang)
		}
		if utf8.RuneCountInString(value) > guiTextMaxLen || strings.ContainsRune(value, '\x00') {
			return nil, worldMapStateNameInvalid("state %d %s[%s] is not valid localisation text", stateID, field, lang)
		}
		out[lang] = value
	}
	return out, nil
}

func sameWorldMapLanguageSet(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for lang := range left {
		if _, ok := right[lang]; !ok {
			return false
		}
	}
	return true
}

func generatedStateNamePlaceholder(stateID int64, value string) bool {
	value = strings.TrimSpace(value)
	return strings.EqualFold(value, fmt.Sprintf("State %d", stateID)) ||
		strings.EqualFold(value, fmt.Sprintf("STATE_%d", stateID)) ||
		value == fmt.Sprintf("第 %d 州", stateID)
}

func generatedCapitalNamePlaceholder(stateID int64, value string) bool {
	value = strings.TrimSpace(value)
	return strings.EqualFold(value, fmt.Sprintf("Town %d", stateID)) ||
		value == fmt.Sprintf("城镇 %d", stateID)
}

func generatedWorldNamePlaceholder(stateID int64, value string) bool {
	return generatedStateNamePlaceholder(stateID, value) || generatedCapitalNamePlaceholder(stateID, value)
}

func completeWorldMapCapitalNames(stateID int64, display, capital map[string]string) (map[string]string, error) {
	for lang, value := range display {
		if generatedWorldNamePlaceholder(stateID, value) {
			return nil, worldMapStateNameInvalid("state %d display_name_locs[%s] is still the generated placeholder %q", stateID, lang, value)
		}
	}
	for lang, value := range capital {
		if _, ok := display[lang]; !ok {
			return nil, worldMapStateNameInvalid("state %d capital_name_locs[%s] has no matching state-name language", stateID, lang)
		}
		if generatedWorldNamePlaceholder(stateID, value) {
			return nil, worldMapStateNameInvalid("state %d capital_name_locs[%s] is still the generated placeholder %q", stateID, lang, value)
		}
	}
	completed := make(map[string]string, len(display))
	for lang, stateName := range display {
		if capitalName, ok := capital[lang]; ok {
			completed[lang] = capitalName
		} else {
			completed[lang] = stateName
		}
	}
	return completed, nil
}

func normalizeWorldMapStateNamesInput(in []WorldMapStateNameItemInput) ([]normalizedWorldMapStateNameItem, error) {
	if len(in) == 0 {
		return nil, worldMapStateNameInvalid("items must not be empty")
	}
	seen := make(map[int64]bool, len(in))
	out := make([]normalizedWorldMapStateNameItem, 0, len(in))
	var commonLanguages map[string]string
	for _, item := range in {
		if item.StateID <= 0 {
			return nil, worldMapStateNameInvalid("state_id must be a positive integer")
		}
		if seen[item.StateID] {
			return nil, worldMapStateNameInvalid("duplicate state_id %d", item.StateID)
		}
		seen[item.StateID] = true
		display, err := normalizeWorldMapNameLocs("display_name_locs", item.StateID, item.DisplayNameLocs)
		if err != nil {
			return nil, err
		}
		capital, err := normalizeWorldMapNameLocs("capital_name_locs", item.StateID, item.CapitalNameLocs)
		if err != nil {
			return nil, err
		}
		if len(display) == 0 {
			return nil, worldMapStateNameInvalid("state %d display_name_locs must not be empty", item.StateID)
		}
		if commonLanguages == nil {
			commonLanguages = display
		} else if !sameWorldMapLanguageSet(commonLanguages, display) {
			return nil, worldMapStateNameInvalid("state %d display_name_locs must use the same language set as every other state", item.StateID)
		}
		capital, err = completeWorldMapCapitalNames(item.StateID, display, capital)
		if err != nil {
			return nil, err
		}
		out = append(out, normalizedWorldMapStateNameItem{stateID: item.StateID, displayNameLocs: display, capitalNameLocs: capital})
	}
	return out, nil
}

func normalizeStoredWorldMapNameLocs(stateID int64, locs map[string]string) (map[string]string, error) {
	out := make(map[string]string, len(locs))
	for rawLang, value := range locs {
		lang := strings.ToLower(strings.TrimSpace(rawLang))
		if lang == "" {
			lang = "en"
		}
		if _, _, ok := GUITextTarget(lang); !ok {
			return nil, worldMapStateNameInvalid("state %d already contains unsupported display-name language %q", stateID, rawLang)
		}
		if _, duplicate := out[lang]; duplicate {
			return nil, worldMapStateNameInvalid("state %d already contains duplicate display-name language %q", stateID, lang)
		}
		out[lang] = strings.TrimSpace(value)
	}
	return out, nil
}

func validateWorldMapExistingLanguages(items []normalizedWorldMapStateNameItem, loaded map[int64]worldMapStateNameRow) error {
	if len(items) == 0 {
		return worldMapStateNameInvalid("items must not be empty")
	}
	incoming := items[0].displayNameLocs
	for stateID, row := range loaded {
		for lang := range row.displayNameLocs {
			if _, ok := incoming[lang]; !ok {
				return worldMapStateNameInvalid("display_name_locs must cover existing language %q (present on state %d)", lang, stateID)
			}
		}
	}
	return nil
}

func validateWorldMapProjectLanguages(items []normalizedWorldMapStateNameItem, required []string) error {
	if len(items) == 0 {
		return worldMapStateNameInvalid("items must not be empty")
	}
	incoming := items[0].displayNameLocs
	requiredSet := make(map[string]bool, len(required))
	for _, lang := range required {
		requiredSet[lang] = true
	}
	missing := make([]string, 0)
	for lang := range requiredSet {
		if _, ok := incoming[lang]; !ok {
			missing = append(missing, lang)
		}
	}
	unexpected := make([]string, 0)
	for lang := range incoming {
		if !requiredSet[lang] {
			unexpected = append(unexpected, lang)
		}
	}
	sort.Strings(missing)
	sort.Strings(unexpected)
	if len(missing) > 0 || len(unexpected) > 0 {
		return worldMapStateNameInvalid(
			"display_name_locs must exactly match project export languages %v; missing %v, unexpected %v",
			required, missing, unexpected)
	}
	return nil
}

func victoryPointScore(raw json.RawMessage) (float64, bool) {
	text := strings.TrimSpace(string(raw))
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		if unquoted, err := strconv.Unquote(text); err == nil {
			text = strings.TrimSpace(unquoted)
		}
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) {
		return 0, false
	}
	return value, true
}

func primaryVictoryPointProvince(raw string) (int64, bool, error) {
	var points []struct {
		ProvinceID int64           `json:"province_id"`
		Value      json.RawMessage `json:"value"`
	}
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "null" {
		return 0, false, nil
	}
	if err := json.Unmarshal([]byte(raw), &points); err != nil {
		return 0, false, fmt.Errorf("invalid victory_points: %w", err)
	}
	var selected int64
	best := math.Inf(-1)
	hasNumericScore := false
	for _, point := range points {
		if point.ProvinceID <= 0 {
			continue
		}
		if selected == 0 {
			selected = point.ProvinceID
		}
		score, ok := victoryPointScore(point.Value)
		if !ok {
			continue
		}
		if !hasNumericScore || score > best {
			selected, best, hasNumericScore = point.ProvinceID, score, true
		}
	}
	return selected, selected != 0, nil
}

func exportLanguages(record project.Row) ([]string, error) {
	var configured []string
	if raw := record["export_languages"]; raw != nil {
		body, _ := json.Marshal(raw)
		if err := json.Unmarshal(body, &configured); err != nil {
			return nil, fmt.Errorf("project export_languages: %w", err)
		}
	}
	return locout.EnsureEnFirstStrings(configured), nil
}

func (s *mapStore) ApplyWorldMapStateNames(in []WorldMapStateNameItemInput) (WorldMapStateNamesResult, error) {
	items, err := normalizeWorldMapStateNamesInput(in)
	if err != nil {
		return WorldMapStateNamesResult{}, err
	}
	var result WorldMapStateNamesResult
	err = s.p.Write([]string{statesTable, locTable, additionsTable}, func(tx *project.Tx) error {
		if !hasWorld(tx) {
			return detailError(409, "state naming requires an existing generated world")
		}
		requiredLanguages, err := exportLanguages(tx.Record())
		if err != nil {
			return err
		}
		if err := validateWorldMapProjectLanguages(items, requiredLanguages); err != nil {
			return err
		}
		loaded := map[int64]worldMapStateNameRow{}
		for _, row := range tx.Rows(statesTable) {
			stateID := project.Int64(row["state_id"])
			display := map[string]string{}
			if raw := row["display_name_locs"]; raw != nil {
				body, _ := json.Marshal(raw)
				if err := json.Unmarshal(body, &display); err != nil {
					return fmt.Errorf("state %d display_name_locs: %w", stateID, err)
				}
			}
			display, err := normalizeStoredWorldMapNameLocs(stateID, display)
			if err != nil {
				return err
			}
			points := "[]"
			if row["victory_points"] != nil {
				points = jsonText(row["victory_points"])
			}
			loaded[stateID] = worldMapStateNameRow{id: project.Int64(row["id"]), displayNameLocs: display, victoryPoints: points}
		}

		requested := make(map[int64]bool, len(items))
		unknown := make([]int64, 0)
		for _, item := range items {
			requested[item.stateID] = true
			if _, ok := loaded[item.stateID]; !ok {
				unknown = append(unknown, item.stateID)
			}
		}
		if len(unknown) > 0 {
			return detailError(404, fmt.Sprintf("states not found in project %d: %v", s.p.ID(), unknown))
		}
		missing := make([]int64, 0)
		for stateID := range loaded {
			if !requested[stateID] {
				missing = append(missing, stateID)
			}
		}
		sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
		if len(missing) > 0 {
			return worldMapStateNameInvalid("items must cover every state in project %d; missing state_ids: %v", s.p.ID(), missing)
		}
		if err := validateWorldMapExistingLanguages(items, loaded); err != nil {
			return err
		}

		type stateUpdate struct {
			id   int64
			locs map[string]string
		}
		var stateUpdates []stateUpdate
		capitalUpdates := map[string]worldMapCapitalLocUpdate{}
		worldTextKeySet := map[string]bool{}
		languageSet := map[string]bool{}
		result = WorldMapStateNamesResult{
			NameStatus: WorldMapStateNameStatus{Languages: []string{}},
			Items:      make([]WorldMapStateNameReadback, 0, len(items)),
		}
		for _, item := range items {
			stored := loaded[item.stateID]
			for lang, value := range item.displayNameLocs {
				stored.displayNameLocs[lang] = value
			}
			stateUpdates = append(stateUpdates, stateUpdate{id: stored.id, locs: stored.displayNameLocs})
			readback := WorldMapStateNameReadback{
				StateID: item.stateID, DisplayNameLocs: stored.displayNameLocs, CapitalNameLocs: map[string]string{},
			}
			if len(item.displayNameLocs) > 0 {
				result.StateCount++
				result.LocalizationCount += len(item.displayNameLocs)
				for lang := range item.displayNameLocs {
					languageSet[lang] = true
				}
			}
			capitalID, hasCapital, err := primaryVictoryPointProvince(stored.victoryPoints)
			if err != nil {
				return fmt.Errorf("state %d: %w", item.stateID, err)
			}
			if !hasCapital {
				return worldMapStateNameInvalid("state %d has no victory point to name as its capital", item.stateID)
			}
			readback.CapitalProvinceID = capitalID
			if len(item.capitalNameLocs) > 0 {
				readback.CapitalNameLocs = item.capitalNameLocs
				result.CapitalCount++
				key := fmt.Sprintf("VICTORY_POINTS_%d", capitalID)
				for lang, value := range item.capitalNameLocs {
					locale, file, _ := GUITextTarget(lang)
					identity := locale + "\x00" + key + "\x00" + file
					if old, duplicate := capitalUpdates[identity]; duplicate && old.Value != value {
						return worldMapStateNameInvalid("capital %d has conflicting %s names", capitalID, lang)
					}
					capitalUpdates[identity] = worldMapCapitalLocUpdate{Locale: locale, Key: key, Value: value, SourceFile: file}

					worldTextKeySet[lang+":"+key] = true
				}
			}
			result.Items = append(result.Items, readback)
		}
		result.NameStatus = WorldMapStateNameStatus{
			Complete:    result.StateCount == len(loaded) && result.CapitalCount == len(loaded),
			TotalStates: len(loaded), NamedStates: result.StateCount,
			Languages: make([]string, 0, len(languageSet)),
		}
		for lang := range languageSet {
			result.NameStatus.Languages = append(result.NameStatus.Languages, lang)
		}
		sort.Strings(result.NameStatus.Languages)
		identities := make([]string, 0, len(capitalUpdates))
		for identity := range capitalUpdates {
			identities = append(identities, identity)
		}
		sort.Strings(identities)
		result.LocalizationCount += len(identities)
		worldTextKeys := make([]string, 0, len(worldTextKeySet))
		for key := range worldTextKeySet {
			worldTextKeys = append(worldTextKeys, key)
		}
		sort.Strings(worldTextKeys)

		for _, update := range stateUpdates {
			locs := update.locs
			if _, err := tx.Update(statesTable, update.id, func(r project.Row) { r["display_name_locs"] = plain(locs) }); err != nil {
				return err
			}
		}
		for _, identity := range identities {
			u := capitalUpdates[identity]
			setText(tx, u.Locale, u.Key, u.Value, u.SourceFile)
		}
		worlds := 0
		for _, row := range tx.Rows(additionsTable) {
			if str(row["kind"]) != "world" {
				continue
			}
			worlds++
			status := plain(result.NameStatus)
			keys := plain(worldTextKeys)
			if _, err := tx.Update(additionsTable, project.Int64(row["id"]), func(r project.Row) {
				summary, _ := plain(r["summary"]).(map[string]any)
				if summary == nil {
					summary = map[string]any{}
				}
				summary["name_status"] = status
				r["summary"], r["text_keys"] = summary, keys
			}); err != nil {
				return err
			}
		}
		if worlds != 1 {
			return fmt.Errorf("world map state names: updated %d world records", worlds)
		}
		return nil
	})
	if err != nil {
		return WorldMapStateNamesResult{}, err
	}
	return result, nil
}
