package states

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/cosmetic"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/project"
)

const (
	historyTable       = "history_files"
	cosmeticTagsTable  = "project_cosmetic_tags"
	cosmeticNamesTable = "project_cosmetic_tag_names"
)

var errHistoryMissing = apiutil.NotFoundError("HistoryFile not found")

var historyVanillaGroupOrder = []string{"democratic", "communism", "fascism", "neutrality"}

var (
	hoi4DateRE = regexp.MustCompile(`^([0-9]{1,4})\.([0-9]{1,2})\.([0-9]{1,2})(?:\.([0-9]{1,2}))?$`)
	rootIDRE   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

var historyFileDataColumns = []string{
	"country_tag", "capital", "set_politics_ruling_party", "set_politics_last_election",
	"set_politics_election_frequency", "set_politics_elections_allowed",
	"set_popularities_democratic", "set_popularities_communism",
	"set_popularities_fascism", "set_popularities_neutrality",
	"set_popularities",
	"set_research_slots", "set_convoys", "set_stability", "set_war_support",
	"add_ideas_extra", "remove_ideas", "complete_national_focus_ids",
	"set_technology_ids", "set_cosmetic_tag", "set_oob", "set_air_oob",
	"set_naval_oob", "startup_flags", "give_guarantee_tags", "add_to_faction_tags",
	"add_to_faction", "set_autonomy", "diplomatic_relation", "date_blocks",
	"raw_script", "source",
}

var historyFileJSONColumns = map[string]bool{
	"add_ideas_extra": true, "remove_ideas": true, "complete_national_focus_ids": true,
	"set_technology_ids": true, "startup_flags": true, "give_guarantee_tags": true,
	"add_to_faction_tags": true, "set_autonomy": true, "diplomatic_relation": true,
	"date_blocks": true, "set_popularities": true,
}

func historyFileDefaults() map[string]any {
	return map[string]any{
		"capital": 0, "set_politics_ruling_party": "", "set_politics_last_election": "",
		"set_politics_election_frequency": 0, "set_politics_elections_allowed": nil,
		"set_popularities_democratic": 0, "set_popularities_communism": 0,
		"set_popularities_fascism": 0, "set_popularities_neutrality": 0,
		"set_popularities":   map[string]any{},
		"set_research_slots": 0, "set_convoys": 0, "set_stability": -1.0,
		"set_war_support": -1.0, "add_ideas_extra": []any{}, "remove_ideas": []any{},
		"complete_national_focus_ids": []any{}, "set_technology_ids": []any{},
		"set_cosmetic_tag": "", "set_oob": "", "set_air_oob": "", "set_naval_oob": "",
		"startup_flags": []any{}, "give_guarantee_tags": []any{},
		"add_to_faction_tags": []any{}, "add_to_faction": "", "set_autonomy": []any{},
		"diplomatic_relation": []any{}, "date_blocks": []any{}, "raw_script": "",
		"source": "created",
	}
}

func registerHistoryFiles(mux *http.ServeMux, h *handlers) {
	base := "/api/go/projects/{id}/history-files"
	mux.HandleFunc("GET "+base, h.handle(h.listHistoryFiles))
	mux.HandleFunc("POST "+base, h.handle(h.createHistoryFile))
	mux.HandleFunc("GET "+base+"/{hid}", h.handle(h.getHistoryFile))
	mux.HandleFunc("PUT "+base+"/{hid}", h.handle(h.updateHistoryFile))
	mux.HandleFunc("DELETE "+base+"/{hid}", h.handle(h.deleteHistoryFile))
}

func historyID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("hid"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errHistoryMissing
	}
	return id, nil
}

func decodeHistoryInput(r *http.Request) (map[string]any, error) {
	var in map[string]any
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20)).Decode(&in); err != nil {
		return nil, detailError(http.StatusUnprocessableEntity, "Invalid payload")
	}
	if in == nil {
		in = map[string]any{}
	}
	return in, nil
}

func (h *handlers) listHistoryFiles(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	rows, err := p.Rows(historyTable)
	if err != nil {
		return err
	}
	sort.SliceStable(rows, func(i, j int) bool { return str(rows[i]["country_tag"]) < str(rows[j]["country_tag"]) })
	apiutil.JSON(w, http.StatusOK, rows)
	return nil
}

func readHistoryFile(p *project.Project, id int64) (project.Row, error) {
	rows, err := p.Rows(historyTable)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			return row, nil
		}
	}
	return nil, errHistoryMissing
}

func (h *handlers) getHistoryFile(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := historyID(r)
	if err != nil {
		return err
	}
	row, err := readHistoryFile(p, id)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, row)
	return nil
}

func validateHistoryDate(value string) (string, error) {
	raw := strings.TrimSpace(value)
	m := hoi4DateRE.FindStringSubmatch(raw)
	if m == nil {
		return "", fmt.Errorf("date must match Y.M.D or Y.M.D.H (for example 1939.1.1.12)")
	}
	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	hour := 12
	if m[4] != "" {
		hour, _ = strconv.Atoi(m[4])
	}
	if year < 1 || month < 1 || month > 12 || day < 1 || day > 31 {
		return "", fmt.Errorf("date must use a positive year, month 1-12, and day 1-31")
	}
	if hour < 0 || hour > 23 {
		return "", fmt.Errorf("date hour must be between 0 and 23")
	}
	return raw, nil
}

func validateHistoryDateBlocks(list []any) ([]any, error) {
	out := make([]any, 0, len(list))
	for index, item := range list {
		block, _ := item.(map[string]any)
		if block == nil {
			return nil, fmt.Errorf("date block %d must be a HistoryDateBlock or mapping", index)
		}
		dateRaw, hasDate := block["date"]
		scriptRaw, hasScript := block["script"]
		if !hasDate || !hasScript {
			return nil, fmt.Errorf("date block %d must contain date and script", index)
		}
		dateText, ok := dateRaw.(string)
		if !ok {
			return nil, fmt.Errorf("date block %d: date must be a string", index)
		}
		date, err := validateHistoryDate(dateText)
		if err != nil {
			return nil, fmt.Errorf("date block %d: %s", index, err.Error())
		}
		script, ok := scriptRaw.(string)
		if !ok {
			return nil, fmt.Errorf("date block %d: script must be a string", index)
		}
		out = append(out, map[string]any{"date": date, "script": script})
	}
	return out, nil
}

func normalizeHistoryPopularities(raw any) (map[string]any, error) {
	if raw == nil {
		return map[string]any{}, nil
	}
	values, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("set_popularities must be an object keyed by ROOT ideology id")
	}
	out := make(map[string]any, len(values))
	total := int64(0)
	for rawID, rawValue := range values {
		id := strings.TrimSpace(rawID)
		if !rootIDRE.MatchString(id) {
			return nil, fmt.Errorf("set_popularities contains invalid ROOT ideology id %q", rawID)
		}
		value, ok := whole(rawValue)
		if _, isBool := rawValue.(bool); isBool || !ok || value < 0 || value > 100 {
			return nil, fmt.Errorf("set_popularities[%q] must be an integer from 0 to 100", id)
		}
		out[id] = value
		total += value
	}
	if len(out) > 0 && total != 100 {
		return nil, fmt.Errorf("set_popularities must add up to 100 (got %d)", total)
	}
	return out, nil
}

func normalizeHistoryFilePayload(in map[string]any) error {
	if raw, present := in["set_cosmetic_tag"]; present && raw != nil {
		tag, ok := raw.(string)
		if !ok {
			return fmt.Errorf("set_cosmetic_tag must be a string")
		}
		if tag != "" {
			if err := cosmetic.ValidateTag(tag); err != nil {
				return err
			}
		}
	}
	if raw, present := in["set_popularities"]; present {
		popularities, err := normalizeHistoryPopularities(raw)
		if err != nil {
			return err
		}
		in["set_popularities"] = popularities
		for _, group := range historyVanillaGroupOrder {
			value, _ := whole(popularities[group])
			in["set_popularities_"+group] = value
		}
	}
	if raw, present := in["date_blocks"]; present && raw != nil {
		list, ok := raw.([]any)
		if !ok {
			return fmt.Errorf("date_blocks must be a list")
		}
		blocks, err := validateHistoryDateBlocks(list)
		if err != nil {
			return err
		}
		in["date_blocks"] = blocks
	}
	if _, present := in["add_to_faction_tags"]; present {
		in["add_to_faction"] = ""
	}
	return nil
}

func historyColumnValue(column string, value any) (any, error) {
	if historyFileJSONColumns[column] {
		if value == nil {
			if column == "set_popularities" {
				return map[string]any{}, nil
			}
			return []any{}, nil
		}
		return plain(value), nil
	}
	stored, err := columnValue(historyTable, column, value)
	if err != nil {
		return nil, detailError(http.StatusUnprocessableEntity, "Invalid payload")
	}
	return stored, nil
}

func historyTagTaken(tx *project.Tx, tag string, except int64) bool {
	for _, row := range tx.Rows(historyTable) {
		if project.Int64(row["id"]) != except && str(row["country_tag"]) == tag {
			return true
		}
	}
	return false
}

func (h *handlers) createHistoryFile(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	in, err := decodeHistoryInput(r)
	if err != nil {
		return err
	}
	if err := normalizeHistoryFilePayload(in); err != nil {
		return detailError(http.StatusUnprocessableEntity, err.Error())
	}
	countryTag := strings.ToUpper(strings.TrimSpace(str(in["country_tag"])))
	if countryTag == "" {
		return detailError(http.StatusBadRequest, "country_tag is required (e.g. 'USA')")
	}
	in["country_tag"] = countryTag
	if v, present := in["set_convoys"]; present {
		if n, ok := num(v); ok && (n < 0 || n > 2147483647) {
			return detailError(http.StatusUnprocessableEntity, "set_convoys must be between 0 and 2147483647")
		}
	}
	defaults := historyFileDefaults()
	row := project.Row{}
	for _, column := range historyFileDataColumns {
		value, present := in[column]
		if !present {
			value = defaults[column]
		}
		stored, err := historyColumnValue(column, value)
		if err != nil {
			return err
		}
		row[column] = stored
	}
	var created project.Row
	err = p.Write([]string{historyTable, cosmeticTagsTable, cosmeticNamesTable, locEntriesTable}, func(tx *project.Tx) error {
		if historyTagTaken(tx, countryTag, 0) {
			return detailError(http.StatusConflict, fmt.Sprintf(
				"history_file for country '%s' already exists in this project. PUT to update or DELETE first.", countryTag))
		}
		created = tx.Insert(historyTable, row)
		return registerCosmeticTagReference(tx, str(row["set_cosmetic_tag"]), "created")
	})
	if err != nil {
		return err
	}
	stored, err := readHistoryFile(p, project.Int64(created["id"]))
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusCreated, stored)
	return nil
}

func (h *handlers) updateHistoryFile(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := historyID(r)
	if err != nil {
		return err
	}
	cur, err := readHistoryFile(p, id)
	if err != nil {
		return err
	}
	in, err := decodeHistoryInput(r)
	if err != nil {
		return err
	}
	if err := normalizeHistoryFilePayload(in); err != nil {
		return detailError(http.StatusUnprocessableEntity, err.Error())
	}
	if _, writesMap := in["set_popularities"]; !writesMap && writesLegacyPopularities(in) {
		if current, ok := cur["set_popularities"].(map[string]any); ok && len(current) > 0 {
			return detailError(http.StatusUnprocessableEntity, "this country uses the canonical set_popularities map; update that map instead of the legacy four fields")
		}
	}
	if v, present := in["country_tag"]; present {
		normalized := strings.ToUpper(strings.TrimSpace(str(v)))
		if normalized == "" {
			return detailError(http.StatusBadRequest, "country_tag cannot be empty")
		}
		in["country_tag"] = normalized
	}
	changes := map[string]any{}
	for _, column := range historyFileDataColumns {
		value, present := in[column]
		if !present {
			continue
		}
		stored, err := historyColumnValue(column, value)
		if err != nil {
			return err
		}
		changes[column] = stored
	}
	if str(cur["source"]) == "imported" {
		finalSource := "imported"
		if v, present := in["source"]; present {
			finalSource = str(v)
		}
		if finalSource == "imported" {
			changes["source"] = "created"
		}
	}
	err = p.Write([]string{historyTable, cosmeticTagsTable, cosmeticNamesTable, locEntriesTable}, func(tx *project.Tx) error {
		if tx.Find(historyTable, id) == nil {
			return errHistoryMissing
		}
		if len(changes) > 0 {
			if tag, ok := changes["country_tag"].(string); ok && historyTagTaken(tx, tag, id) {
				return detailError(http.StatusConflict, "country_tag conflict in this project")
			}
			if _, err := tx.Update(historyTable, id, func(row project.Row) {
				for column, value := range changes {
					row[column] = value
				}
			}); err != nil {
				return err
			}
		}
		if value, present := in["set_cosmetic_tag"]; present {
			return registerCosmeticTagReference(tx, str(value), "created")
		}
		return nil
	})
	if err != nil {
		return err
	}
	stored, err := readHistoryFile(p, id)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, stored)
	return nil
}

func writesLegacyPopularities(in map[string]any) bool {
	for _, group := range historyVanillaGroupOrder {
		if _, ok := in["set_popularities_"+group]; ok {
			return true
		}
	}
	return false
}

func (h *handlers) deleteHistoryFile(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := historyID(r)
	if err != nil {
		return err
	}
	err = p.Write([]string{historyTable}, func(tx *project.Tx) error {
		if tx.Delete(historyTable, func(row project.Row) bool { return project.Int64(row["id"]) == id }) == 0 {
			return errHistoryMissing
		}
		return nil
	})
	if err != nil {
		return err
	}
	apiutil.NoContent(w)
	return nil
}

func registerCosmeticTagReference(tx *project.Tx, tag, source string) error {
	if tag == "" {
		return nil
	}
	if err := cosmetic.ValidateTag(tag); err != nil {
		return apiutil.ValidationError(err.Error())
	}
	for _, row := range tx.Rows(cosmeticTagsTable) {
		if str(row["tag"]) == tag {
			return nil
		}
	}
	created := tx.Insert(cosmeticTagsTable, project.Row{"tag": tag, "source": source, "revision": int64(1)})
	tagID := project.Int64(created["id"])
	type winner struct {
		value                               string
		priority, sourceOrder, entryOrd, id int64
	}
	locales := map[string]bool{}
	for _, l := range cosmetic.Locales() {
		locales[l] = true
	}
	keyed := map[string]string{}
	for _, suffix := range cosmetic.Suffixes() {
		key := tag
		if suffix != "" {
			key += "_" + suffix
		}
		keyed[key] = suffix
	}
	best := map[[2]string]winner{}
	for _, entry := range tx.Rows(locEntriesTable) {
		suffix, ok := keyed[str(entry["key"])]
		if !ok || !locales[str(entry["locale"])] {
			continue
		}
		if _, safe := exportengine.SafePreservedLocSourceFile(str(entry["source_file"])); !safe {
			continue
		}
		w := winner{value: str(entry["value"]), priority: int64(coerceInt(entry["replace_priority"])),
			sourceOrder: int64(coerceInt(entry["source_order"])), entryOrd: int64(coerceInt(entry["entry_order"])),
			id: project.Int64(entry["id"])}
		slot := [2]string{str(entry["locale"]), suffix}
		prior, seen := best[slot]
		if !seen || w.priority > prior.priority ||
			(w.priority == prior.priority && (w.sourceOrder > prior.sourceOrder ||
				(w.sourceOrder == prior.sourceOrder && (w.entryOrd > prior.entryOrd ||
					(w.entryOrd == prior.entryOrd && w.id > prior.id))))) {
			best[slot] = w
		}
	}
	if len(best) == 0 {
		return nil
	}
	slots := make([][2]string, 0, len(best))
	for slot := range best {
		slots = append(slots, slot)
	}
	sort.Slice(slots, func(i, j int) bool {
		if slots[i][0] != slots[j][0] {
			return slots[i][0] < slots[j][0]
		}
		return slots[i][1] < slots[j][1]
	})
	for _, slot := range slots {
		tx.Insert(cosmeticNamesTable, project.Row{
			"cosmetic_tag_id": tagID, "locale": slot[0], "suffix": slot[1],
			"value": best[slot].value, "source": "imported",
		})
	}

	_, err := tx.Update(cosmeticTagsTable, tagID, func(row project.Row) { row["revision"] = int64(2) })
	return err
}
