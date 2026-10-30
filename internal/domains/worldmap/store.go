package worldmap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains/states"
	"haganeworks/internal/hoi4/emit/locout"
	"haganeworks/internal/project"
	"haganeworks/internal/reftables"
)

const (
	rawFilesTable  = "project_raw_files"
	additionsTable = "world_map_additions"
	statesTable    = "states"
	locTable       = "project_loc_entries"
	historyTable   = "history_files"
	characterTable = "characters"
)

var ErrWorldMapNotBase = errors.New("world map: the project's map is no longer the base game's")

var errTextInvalid = errors.New("invalid GUI text override")

type RawFile struct {
	DestPath            string
	Content             string
	IsBinary            bool
	GenerateReplacePath bool
	RawKind             string
	Note                string
	ContentSHA256       *string
	ContentOBSKey       *string
	Source              string
}

type RawFileInsert struct {
	DestPath            string
	Content             string
	IsBinary            bool
	GenerateReplacePath bool
	RawKind             string
	Note                string
	ContentSHA256       *string
	ContentOBSKey       *string
}

type previousRawFile struct {
	DestPath            string  `json:"dest_path"`
	Content             string  `json:"content"`
	IsBinary            bool    `json:"is_binary"`
	GenerateReplacePath bool    `json:"generate_replace_path"`
	Note                string  `json:"note"`
	SHA256              *string `json:"sha256"`
	OBSKey              *string `json:"obs_key"`
	Source              string  `json:"source"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type WorldMapAddition struct {
	ID        int64           `json:"id"`
	Kind      string          `json:"kind"`
	Placement [4]int          `json:"placement"`
	Seed      int64           `json:"seed"`
	Options   json.RawMessage `json:"options"`
	Name      string          `json:"name"`
	Owner     string          `json:"owner"`
	Provinces []int           `json:"provinces"`
	StateIDs  []int           `json:"state_ids"`
	RegionID  int             `json:"region_id"`
	RawPaths  []string        `json:"raw_paths"`
	TextKeys  []string        `json:"text_keys"`
	Summary   json.RawMessage `json:"summary"`
	CreatedAt time.Time       `json:"created_at"`

	StateBefore map[int]StateBefore `json:"-"`
}

type StateBefore struct {
	Provinces []int `json:"provinces"`

	Text   string `json:"text"`
	Source string `json:"source"`

	Seeded bool `json:"seeded"`

	Row json.RawMessage `json:"row,omitempty"`
}

type WorldMapVictoryPoint struct {
	ProvinceID int             `json:"province_id"`
	Value      json.RawMessage `json:"value"`
}

type WorldMapStateRow struct {
	Provinces []int
	Manpower  int64
	Category  string
	Owner     string
	Buildings map[string]any

	Locs map[string]string

	VictoryPoints map[int]int

	Cores []string

	ProvinceBuildings string

	Points []WorldMapVictoryPoint

	Resources map[string]any
}

type WorldMapStateFull struct {
	Row           json.RawMessage
	Name          string
	Locs          map[string]string
	Provinces     []int
	Owner         string
	Cores         []string
	VictoryPoints []WorldMapVictoryPoint
	Manpower      int64
	Category      string
	Buildings     map[string]any

	BuildingsText string

	Extras map[string]any
}

type WorldMapStateUpdate struct {
	Provinces     []int
	Owner         string
	Cores         []string
	VictoryPoints []WorldMapVictoryPoint

	BuildingsText string
	Manpower      int64

	ExtrasSet  map[string]any
	ExtrasDrop []string
}

type HistoryBefore struct {
	Capital *int   `json:"capital"`
	Source  string `json:"source"`
}

type mapStore struct {
	p    *project.Project
	game func() (fs.FS, error)
}

func str(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}

func intOf(v any) int { return int(project.Int64(v)) }

func plain(v any) any {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return plainJSON(raw)
}

func plainJSON(raw []byte) any {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out any
	if decoder.Decode(&out) != nil {
		return nil
	}
	return out
}

func jsonText(v any) string {
	body, _ := json.Marshal(v)
	return string(body)
}

func optionalString(v any) *string {
	if v == nil {
		return nil
	}
	s := str(v)
	return &s
}

func optionalValue(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func intSet(ids []int) map[int]bool {
	out := make(map[int]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out
}

func rawFileFromRow(row project.Row) RawFile {
	isBinary, _ := row["is_binary"].(bool)
	replace, _ := row["generate_replace_path"].(bool)
	return RawFile{
		DestPath: str(row["dest_path"]), Content: str(row["content"]), IsBinary: isBinary,
		GenerateReplacePath: replace, RawKind: str(row["raw_kind"]), Note: str(row["note"]),
		ContentSHA256: optionalString(row["content_sha256"]), ContentOBSKey: optionalString(row["content_obs_key"]),
		Source: str(row["source"]),
	}
}

func (s *mapStore) RawFiles() ([]RawFile, error) {
	rows, err := s.p.Rows(rawFilesTable)
	if err != nil {
		return nil, err
	}
	out := make([]RawFile, 0, len(rows))
	for _, row := range rows {
		out = append(out, rawFileFromRow(row))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DestPath < out[j].DestPath })
	return out, nil
}

func (s *mapStore) RawFileBlob(destPath string) (sha, key string, found bool, err error) {
	rows, err := s.p.Rows(rawFilesTable)
	if err != nil {
		return "", "", false, err
	}
	for _, row := range rows {
		if !strings.EqualFold(str(row["dest_path"]), destPath) || row["is_binary"] != true {
			continue
		}
		if row["content_sha256"] == nil || row["content_obs_key"] == nil {
			return "", "", false, nil
		}
		return str(row["content_sha256"]), str(row["content_obs_key"]), true, nil
	}
	return "", "", false, nil
}

func validateRawFileStorage(in RawFileInsert) error {
	if in.IsBinary && (in.ContentSHA256 == nil || strings.TrimSpace(*in.ContentSHA256) == "" ||
		in.ContentOBSKey == nil || strings.TrimSpace(*in.ContentOBSKey) == "") {
		return apiutil.ValidationError("binary raw files require an OBS/COS content pointer")
	}
	return nil
}

func writeRawFiles(tx *project.Tx, files []RawFileInsert) map[string]previousRawFile {
	previous := map[string]previousRawFile{}
	for _, file := range files {
		for _, row := range tx.Rows(rawFilesTable) {
			if !strings.EqualFold(str(row["dest_path"]), file.DestPath) {
				continue
			}
			isBinary, _ := row["is_binary"].(bool)
			replace, _ := row["generate_replace_path"].(bool)
			source := str(row["source"])
			if source == "" {
				source = "created"
			}
			previous[strings.ToLower(file.DestPath)] = previousRawFile{
				DestPath: str(row["dest_path"]), Content: str(row["content"]), IsBinary: isBinary,
				GenerateReplacePath: replace, Note: str(row["note"]),
				SHA256: optionalString(row["content_sha256"]), OBSKey: optionalString(row["content_obs_key"]),
				Source: source, CreatedAt: str(row["created_at"]), UpdatedAt: str(row["updated_at"]),
			}
			break
		}
		tx.Delete(rawFilesTable, func(row project.Row) bool { return strings.EqualFold(str(row["dest_path"]), file.DestPath) })
		tx.Insert(rawFilesTable, project.Row{
			"dest_path": file.DestPath, "content": file.Content, "is_binary": file.IsBinary,
			"generate_replace_path": file.GenerateReplacePath, "raw_kind": "verbatim_override",
			"target_layer": nil, "target_sha256": nil, "note": file.Note, "content_binary": nil,
			"content_sha256": optionalValue(file.ContentSHA256), "content_obs_key": optionalValue(file.ContentOBSKey),
			"source": "created",
		})
	}
	return previous
}

func putBackRawFile(tx *project.Tx, old previousRawFile) {
	row := project.Row{
		"dest_path": old.DestPath, "content": old.Content, "is_binary": old.IsBinary,
		"generate_replace_path": old.GenerateReplacePath, "raw_kind": "verbatim_override",
		"target_layer": nil, "target_sha256": nil, "note": old.Note, "content_binary": nil,
		"content_sha256": optionalValue(old.SHA256), "content_obs_key": optionalValue(old.OBSKey),
		"source": old.Source,
	}
	if old.CreatedAt != "" {
		row["created_at"] = old.CreatedAt
	}
	tx.Insert(rawFilesTable, row)
}

func deleteRawFile(tx *project.Tx, path string) {
	tx.Delete(rawFilesTable, func(row project.Row) bool { return strings.EqualFold(str(row["dest_path"]), path) })
}

func GUITextTarget(lang string) (locale, file string, ok bool) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" || lang == "en" {
		return "l_english", "localisation/english/replace/zz_game_view_l_english.yml", true
	}
	hoi4, ok := locout.LangCodeToHOI4[lang]
	if !ok {
		return "", "", false
	}
	return "l_" + hoi4, "localisation/" + hoi4 + "/replace/zz_game_view_l_" + hoi4 + ".yml", true
}

const guiTextMaxLen = 8000

func validTextKey(key string) bool {
	if key == "" || len(key) > 512 {
		return false
	}
	for _, ch := range key {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9', ch == '_', ch == '.', ch == '-':
		default:
			return false
		}
	}
	return true
}

func setText(tx *project.Tx, locale, key, value, file string) {
	for _, row := range tx.Rows(locTable) {
		if str(row["locale"]) == locale && str(row["key"]) == key && str(row["source_file"]) == file &&
			project.Int64(row["entry_order"]) == 0 {
			_, _ = tx.Update(locTable, project.Int64(row["id"]), func(r project.Row) {
				r["value"], r["source"] = value, "created"
			})
			return
		}
	}
	tx.Insert(locTable, project.Row{
		"locale": locale, "key": key, "value": value, "version": nil, "source": "created",
		"source_file": file, "replace_priority": int64(1), "source_order": int64(0), "entry_order": int64(0),
		"is_replace": true,
	})
}

func clearText(tx *project.Tx, locale, key, file string) bool {
	return tx.Delete(locTable, func(row project.Row) bool {
		return str(row["locale"]) == locale && str(row["key"]) == key && str(row["source_file"]) == file &&
			str(row["source"]) == "created"
	}) > 0
}

func (s *mapStore) SetGUITextOverrideIn(lang, key, value string) error {
	key = strings.TrimSpace(key)
	locale, file, ok := GUITextTarget(lang)
	if !ok || !validTextKey(key) || utf8.RuneCountInString(value) > guiTextMaxLen || strings.ContainsRune(value, '\x00') {
		return errTextInvalid
	}
	return s.p.Write([]string{locTable}, func(tx *project.Tx) error {
		setText(tx, locale, key, value, file)
		return nil
	})
}

func clearTextKeys(tx *project.Tx, entries []string) {
	seen := map[string]bool{}
	for _, entry := range entries {
		lang, key, ok := strings.Cut(entry, ":")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			continue
		}
		locale, file, ok := GUITextTarget(lang)
		if !ok {
			continue
		}
		identity := locale + "\x00" + key + "\x00" + file
		if seen[identity] {
			continue
		}
		seen[identity] = true
		clearText(tx, locale, key, file)
	}
}

func additionFromRow(row project.Row) WorldMapAddition {
	raw := func(column string) []byte {
		body, _ := json.Marshal(row[column])
		return body
	}
	a := WorldMapAddition{
		ID: project.Int64(row["id"]), Kind: str(row["kind"]), Seed: project.Int64(row["seed"]),
		Options: raw("options"), Name: str(row["name"]), Owner: str(row["owner"]),
		RegionID: intOf(row["region_id"]), Summary: raw("summary"),
	}
	if a.Kind == "" {
		a.Kind = "land"
	}
	if created, err := time.Parse("2006-01-02T15:04:05.999999-07:00", str(row["created_at"])); err == nil {
		a.CreatedAt = created.UTC()
	} else if created, err := time.Parse("2006-01-02T15:04:05.999999", str(row["created_at"])); err == nil {
		a.CreatedAt = created.UTC()
	}
	_ = json.Unmarshal(raw("state_before"), &a.StateBefore)
	for id, state := range a.StateBefore {
		if state.Provinces == nil {
			state.Provinces = []int{}
			a.StateBefore[id] = state
		}
	}
	_ = json.Unmarshal(raw("placement"), &a.Placement)
	_ = json.Unmarshal(raw("provinces"), &a.Provinces)
	_ = json.Unmarshal(raw("state_ids"), &a.StateIDs)
	_ = json.Unmarshal(raw("raw_paths"), &a.RawPaths)
	_ = json.Unmarshal(raw("text_keys"), &a.TextKeys)
	if a.Provinces == nil {
		a.Provinces = []int{}
	}
	if a.StateIDs == nil {
		a.StateIDs = []int{}
	}
	if a.RawPaths == nil {
		a.RawPaths = []string{}
	}
	if a.TextKeys == nil {
		a.TextKeys = []string{}
	}
	return a
}

func additionsOf(rows []project.Row) []WorldMapAddition {
	out := make([]WorldMapAddition, 0, len(rows))
	for _, row := range rows {
		out = append(out, additionFromRow(row))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *mapStore) WorldMapAdditions() ([]WorldMapAddition, error) {
	rows, err := s.p.Rows(additionsTable)
	if err != nil {
		return nil, err
	}
	return additionsOf(rows), nil
}

func hasWorld(tx *project.Tx) bool {
	for _, row := range tx.Rows(additionsTable) {
		if str(row["kind"]) == "world" {
			return true
		}
	}
	return false
}

func insertRecord(tx *project.Tx, a WorldMapAddition, kind string, previous map[string]previousRawFile, extra project.Row) int64 {
	options, summary := a.Options, a.Summary
	if len(options) == 0 {
		options = json.RawMessage(`{}`)
	}
	if len(summary) == 0 {
		summary = json.RawMessage(`{}`)
	}
	if a.Kind != "" {
		kind = a.Kind
	}
	before := a.StateBefore
	if before == nil {
		before = map[int]StateBefore{}
	}
	ints := func(list []int) []int {
		if list == nil {
			return []int{}
		}
		return list
	}
	strs := func(list []string) []string {
		if list == nil {
			return []string{}
		}
		return list
	}
	row := project.Row{
		"kind": kind, "placement": plain(a.Placement), "seed": a.Seed, "options": plainJSON(options),
		"name": a.Name, "owner": a.Owner, "provinces": plain(ints(a.Provinces)), "state_ids": plain(ints(a.StateIDs)),
		"region_id": int64(a.RegionID), "raw_paths": plain(strs(a.RawPaths)), "text_keys": plain(strs(a.TextKeys)),
		"summary": plainJSON(summary), "state_before": plain(before), "previous": plain(previous),
	}
	for key, value := range extra {
		row[key] = value
	}
	return project.Int64(tx.Insert(additionsTable, row)["id"])
}

func stateRowsByID(tx *project.Tx) map[int]project.Row {
	out := map[int]project.Row{}
	for _, row := range tx.Rows(statesTable) {
		out[intOf(row["state_id"])] = row
	}
	return out
}

func (s *mapStore) WorldMapStates(stateIDs []int) (map[int]StateBefore, error) {
	rows, err := s.p.Rows(statesTable)
	if err != nil {
		return nil, err
	}
	want := intSet(stateIDs)
	out := map[int]StateBefore{}
	for _, row := range rows {
		id := intOf(row["state_id"])
		if !want[id] {
			continue
		}
		text := jsonText(row["provinces"])
		var before StateBefore
		_ = json.Unmarshal([]byte(text), &before.Provinces)
		if before.Provinces == nil {
			before.Provinces = []int{}
		}
		before.Text, before.Source = text, str(row["source"])
		out[id] = before
	}
	return out, nil
}

func (s *mapStore) SeedWorldMapStates(stateIDs []int) error {
	if s.game == nil {
		return errors.New("the game folder is not set up")
	}
	game, err := s.game()
	if err != nil {
		return err
	}
	ids := make([]int64, len(stateIDs))
	for i, id := range stateIDs {
		ids[i] = int64(id)
	}
	return states.SeedFromGame(s.p, game, ids)
}

func (s *mapStore) CreateWorldMapStates(rows []WorldMapStateRow) ([]int, error) {
	var ids []int
	err := s.p.Write([]string{statesTable, additionsTable}, func(tx *project.Tx) error {
		var projectTop int64
		for _, row := range tx.Rows(statesTable) {
			projectTop = max(projectTop, project.Int64(row["state_id"]))
		}

		var top int64
		if !hasWorld(tx) {
			if s.game == nil {
				return errors.New("the game folder is not set up")
			}
			game, err := s.game()
			if err != nil {
				return err
			}
			top = states.GameStateIDMax(game)
		}
		var err error
		ids, err = insertWorldMapStates(tx, max(top, projectTop)+1, rows)
		return err
	})
	return ids, err
}

func insertWorldMapStates(tx *project.Tx, first int64, rows []WorldMapStateRow) ([]int, error) {
	taken := map[int64]bool{}
	for _, row := range tx.Rows(statesTable) {
		taken[project.Int64(row["state_id"])] = true
	}
	ids := make([]int, 0, len(rows))
	next := first
	for _, row := range rows {
		id := next
		next++
		if taken[id] {
			return nil, fmt.Errorf("world map states: state %d is already in the project", id)
		}
		provinces := make([]any, len(row.Provinces))
		for i, p := range row.Provinces {
			provinces[i] = json.Number(strconv.Itoa(p))
		}
		cores := []any{}
		if row.Owner != "" {
			cores = append(cores, row.Owner)
		}
		for _, tag := range row.Cores {
			if tag != "" && tag != row.Owner {
				cores = append(cores, tag)
			}
		}
		locs := row.Locs
		if locs == nil {
			locs = map[string]string{}
		}
		buildings := row.Buildings
		if buildings == nil {
			buildings = map[string]any{}
		}
		points := []any{}
		if row.Points != nil {
			for _, point := range row.Points {
				points = append(points, map[string]any{"province_id": json.Number(strconv.Itoa(point.ProvinceID)), "value": plainJSON(point.Value)})
			}
		} else {
			provincesWithPoints := make([]int, 0, len(row.VictoryPoints))
			for province := range row.VictoryPoints {
				provincesWithPoints = append(provincesWithPoints, province)
			}
			sort.Ints(provincesWithPoints)
			for _, province := range provincesWithPoints {
				points = append(points, map[string]any{"province_id": json.Number(strconv.Itoa(province)), "value": json.Number(strconv.Itoa(row.VictoryPoints[province]))})
			}
		}
		resources := row.Resources
		if resources == nil {
			resources = map[string]any{}
		}
		extras := map[string]any{}
		if row.ProvinceBuildings != "" {
			extras["buildings"] = row.ProvinceBuildings
		}
		tx.Insert(statesTable, project.Row{
			"state_id": id, "name": fmt.Sprintf("STATE_%d", id), "display_name_locs": plain(locs),
			"manpower": row.Manpower, "state_category": row.Category, "resources": plain(resources),
			"provinces": provinces, "owner": row.Owner, "controller": nil, "add_core_of": cores,
			"add_claim_by": []any{}, "remove_core_of": []any{}, "victory_points": points,
			"set_province_controller": []any{}, "buildings": plain(buildings),
			"local_supplies": nil, "buildings_max_level_factor": nil, "impassable": nil,
			"impassable_ignored_links": nil, "force_link_ownership_to": nil, "raw_extras": plain(extras),

			"source_file": nil, "source_snapshot": nil, "source": "created",
		})
		taken[id] = true
		ids = append(ids, int(id))
	}
	return ids, nil
}

func (s *mapStore) DeleteWorldMapStates(stateIDs []int) error {
	want := intSet(stateIDs)
	return s.p.Write([]string{statesTable}, func(tx *project.Tx) error {
		tx.Delete(statesTable, func(row project.Row) bool {
			return want[intOf(row["state_id"])] && str(row["source"]) == "created"
		})
		return nil
	})
}

func (s *mapStore) WorldMapStateRows(stateIDs []int) (map[int]WorldMapStateFull, error) {
	rows, err := s.p.Rows(statesTable)
	if err != nil {
		return nil, err
	}
	want := intSet(stateIDs)
	out := map[int]WorldMapStateFull{}
	for _, row := range rows {
		id := intOf(row["state_id"])
		if !want[id] {
			continue
		}
		text, err := json.Marshal(row)
		if err != nil {
			return nil, fmt.Errorf("state %d row: %w", id, err)
		}
		var raw struct {
			Name          string                 `json:"name"`
			Locs          map[string]string      `json:"display_name_locs"`
			Provinces     []int                  `json:"provinces"`
			Owner         string                 `json:"owner"`
			Cores         []string               `json:"add_core_of"`
			VictoryPoints []WorldMapVictoryPoint `json:"victory_points"`
			Manpower      int64                  `json:"manpower"`
			Category      string                 `json:"state_category"`
			Buildings     map[string]any         `json:"buildings"`
			Extras        map[string]any         `json:"raw_extras"`
		}
		decoder := json.NewDecoder(bytes.NewReader(text))
		decoder.UseNumber()
		if err := decoder.Decode(&raw); err != nil {
			return nil, fmt.Errorf("state %d row: %w", id, err)
		}
		full := WorldMapStateFull{
			Row: json.RawMessage(text), Name: raw.Name, Locs: raw.Locs, Provinces: raw.Provinces, Owner: raw.Owner,
			Cores: raw.Cores, VictoryPoints: raw.VictoryPoints, Manpower: raw.Manpower, Category: raw.Category,
			Buildings: raw.Buildings, Extras: raw.Extras,
		}
		if text, ok := raw.Extras["buildings"].(string); ok {
			full.BuildingsText = text
		}
		out[id] = full
	}
	return out, nil
}

func updateStateProvinces(tx *project.Tx, byID map[int]project.Row, state int, provinces []int) error {
	row, ok := byID[state]
	if !ok {
		return fmt.Errorf("state %d: no row to change", state)
	}
	_, err := tx.Update(statesTable, project.Int64(row["id"]), func(r project.Row) {
		r["provinces"], r["source"] = plain(provinces), "created"
	})
	return err
}

func (s *mapStore) SaveWorldMapAddition(files []RawFileInsert, states map[int][]int, a WorldMapAddition) (int64, error) {
	for _, file := range files {
		if err := validateRawFileStorage(file); err != nil {
			return 0, err
		}
	}
	var id int64
	err := s.p.Write([]string{rawFilesTable, statesTable, additionsTable}, func(tx *project.Tx) error {
		previous := writeRawFiles(tx, files)
		byID := stateRowsByID(tx)
		ids := make([]int, 0, len(states))
		for state := range states {
			ids = append(ids, state)
		}
		sort.Ints(ids)
		for _, state := range ids {
			if err := updateStateProvinces(tx, byID, state, states[state]); err != nil {
				return err
			}
		}
		id = insertRecord(tx, a, "land", previous, nil)
		return nil
	})
	return id, err
}

func (s *mapStore) SaveWorldMapRestate(files []RawFileInsert, updates map[int]WorldMapStateUpdate, a WorldMapAddition) (int64, error) {
	for _, file := range files {
		if err := validateRawFileStorage(file); err != nil {
			return 0, err
		}
	}
	var id int64
	err := s.p.Write([]string{rawFilesTable, statesTable, additionsTable}, func(tx *project.Tx) error {
		previous := writeRawFiles(tx, files)
		byID := stateRowsByID(tx)
		ids := make([]int, 0, len(updates))
		for state := range updates {
			ids = append(ids, state)
		}
		sort.Ints(ids)
		for _, state := range ids {
			u := updates[state]
			row, ok := byID[state]
			if !ok {
				return fmt.Errorf("state %d: no row to change", state)
			}
			cores := u.Cores
			if cores == nil {
				cores = []string{}
			}
			points := u.VictoryPoints
			if points == nil {
				points = []WorldMapVictoryPoint{}
			}
			if _, err := tx.Update(statesTable, project.Int64(row["id"]), func(r project.Row) {
				extras, _ := plain(r["raw_extras"]).(map[string]any)
				if extras == nil {
					extras = map[string]any{}
				}
				delete(extras, "buildings")
				for _, key := range u.ExtrasDrop {
					delete(extras, key)
				}
				for key, value := range u.ExtrasSet {
					extras[key] = plain(value)
				}
				if u.BuildingsText != "" {
					extras["buildings"] = u.BuildingsText
				}
				r["provinces"], r["owner"], r["add_core_of"], r["victory_points"] = plain(u.Provinces), u.Owner, plain(cores), plain(points)
				r["raw_extras"], r["manpower"], r["source"] = extras, u.Manpower, "created"
			}); err != nil {
				return err
			}
		}
		id = insertRecord(tx, a, "edit", previous, nil)
		return nil
	})
	return id, err
}

func restoreStateRow(tx *project.Tx, byID map[int]project.Row, stateID int, stored json.RawMessage) error {
	row, ok := byID[stateID]
	if !ok {
		return errors.New("state " + strconv.Itoa(stateID) + ": no row to put back")
	}
	was, _ := plainJSON(stored).(map[string]any)
	_, err := tx.Update(statesTable, project.Int64(row["id"]), func(r project.Row) {
		for _, column := range []string{"provinces", "owner", "controller", "add_core_of", "victory_points", "raw_extras", "manpower", "source"} {
			r[column] = was[column]
		}
	})
	return err
}

func putBackState(tx *project.Tx, id int, before StateBefore) error {
	if before.Seeded {
		tx.Delete(statesTable, func(row project.Row) bool { return intOf(row["state_id"]) == id })
		return nil
	}
	byID := stateRowsByID(tx)
	if len(before.Row) > 0 && strings.TrimSpace(string(before.Row)) != "null" {
		return restoreStateRow(tx, byID, id, before.Row)
	}
	provinces := before.Text
	if provinces == "" {
		provinces = jsonText(before.Provinces)
	}
	row, ok := byID[id]
	if !ok {
		return nil
	}
	_, err := tx.Update(statesTable, project.Int64(row["id"]), func(r project.Row) {
		r["provinces"], r["source"] = plainJSON([]byte(provinces)), before.Source
	})
	return err
}

func putBackWorldStates(tx *project.Tx, replaced any, histories map[string]HistoryBefore) error {
	rows := []project.Row{}
	list, _ := replaced.([]any)
	for _, item := range list {
		if row, ok := plain(item).(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	tx.Replace(statesTable, rows)
	for id, before := range histories {
		historyID, err := strconv.ParseInt(id, 10, 64)
		if err != nil || tx.Find(historyTable, historyID) == nil {
			continue
		}
		var capital any
		if before.Capital != nil {
			capital = int64(*before.Capital)
		}
		if _, err := tx.Update(historyTable, historyID, func(r project.Row) {
			r["capital"], r["source"] = capital, before.Source
		}); err != nil {
			return err
		}
	}
	return nil
}

func putBackOriginalRawFiles(tx *project.Tx, records []project.Row) {
	seen := map[string]bool{}
	for _, row := range records {
		a := additionFromRow(row)
		replaced := map[string]previousRawFile{}
		body, _ := json.Marshal(row["previous"])
		_ = json.Unmarshal(body, &replaced)
		for _, path := range a.RawPaths {
			lower := strings.ToLower(path)
			if seen[lower] {
				continue
			}
			seen[lower] = true
			if old, ok := replaced[lower]; ok {
				putBackRawFile(tx, old)
			}
		}
	}
}

func sortedRecords(tx *project.Tx) []project.Row {
	rows := append([]project.Row(nil), tx.Rows(additionsTable)...)
	sort.Slice(rows, func(i, j int) bool { return project.Int64(rows[i]["id"]) < project.Int64(rows[j]["id"]) })
	return rows
}

func historiesOf(row project.Row) map[string]HistoryBefore {
	histories := map[string]HistoryBefore{}
	body, _ := json.Marshal(row["histories_before"])
	_ = json.Unmarshal(body, &histories)
	return histories
}

var restoreTables = []string{rawFilesTable, additionsTable, statesTable, locTable, historyTable}

func (s *mapStore) RestoreWorldMap() ([]WorldMapAddition, error) {
	var additions []WorldMapAddition
	err := s.p.Write(restoreTables, func(tx *project.Tx) error {
		records := sortedRecords(tx)
		additions = additionsOf(records)
		if len(additions) == 0 {
			return nil
		}
		var paths, textKeys []string
		var stateIDs []int
		for _, a := range additions {
			paths = append(paths, a.RawPaths...)
			stateIDs = append(stateIDs, a.StateIDs...)
			textKeys = append(textKeys, a.TextKeys...)
		}
		clearTextKeys(tx, textKeys)
		for _, path := range paths {
			deleteRawFile(tx, path)
		}
		putBackOriginalRawFiles(tx, records)

		for _, row := range records {
			if str(row["kind"]) != "world" {
				continue
			}
			if err := putBackWorldStates(tx, row["states_replaced"], historiesOf(row)); err != nil {
				return err
			}
			tx.Delete(additionsTable, func(project.Row) bool { return true })
			return nil
		}
		created := intSet(stateIDs)
		tx.Delete(statesTable, func(row project.Row) bool { return created[intOf(row["state_id"])] })

		first := map[int]StateBefore{}
		for _, a := range additions {
			for id, before := range a.StateBefore {
				if _, seen := first[id]; !seen && !created[id] {
					first[id] = before
				}
			}
		}
		ids := make([]int, 0, len(first))
		for id := range first {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			if err := putBackState(tx, id, first[id]); err != nil {
				return err
			}
		}
		tx.Delete(additionsTable, func(project.Row) bool { return true })
		return nil
	})
	if err != nil {
		return nil, err
	}
	return additions, nil
}

func (s *mapStore) UndoLatestWorldMap() (WorldMapAddition, bool, error) {
	var a WorldMapAddition
	found := false
	err := s.p.Write(restoreTables, func(tx *project.Tx) error {
		records := sortedRecords(tx)
		if len(records) == 0 {
			return nil
		}
		row := records[len(records)-1]
		a, found = additionFromRow(row), true
		previous := map[string]previousRawFile{}
		body, _ := json.Marshal(row["previous"])
		_ = json.Unmarshal(body, &previous)
		clearTextKeys(tx, a.TextKeys)
		for _, path := range a.RawPaths {
			deleteRawFile(tx, path)
			if old, existed := previous[strings.ToLower(path)]; existed {
				putBackRawFile(tx, old)
			}
		}
		if a.Kind == "world" {
			if err := putBackWorldStates(tx, row["states_replaced"], historiesOf(row)); err != nil {
				return err
			}
		} else if len(a.StateIDs) > 0 {
			made := intSet(a.StateIDs)
			tx.Delete(statesTable, func(r project.Row) bool { return made[intOf(r["state_id"])] })
		}
		ids := make([]int, 0, len(a.StateBefore))
		for id := range a.StateBefore {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			if err := putBackState(tx, id, a.StateBefore[id]); err != nil {
				return err
			}
		}
		recordID := a.ID
		tx.Delete(additionsTable, func(r project.Row) bool { return project.Int64(r["id"]) == recordID })
		return nil
	})
	if err != nil {
		return WorldMapAddition{}, false, err
	}
	return a, found, nil
}

func (s *mapStore) WorldMapHistoryTags(tags []string) (map[string]bool, error) {
	rows, err := s.p.Rows(historyTable)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, tag := range tags {
		want[strings.ToUpper(tag)] = true
	}
	out := map[string]bool{}
	for _, row := range rows {
		if tag := strings.ToUpper(str(row["country_tag"])); want[tag] {
			out[tag] = true
		}
	}
	return out, nil
}

func (s *mapStore) WorldMapStateCount() (int, error) {
	rows, err := s.p.Rows(statesTable)
	return len(rows), err
}

func (s *mapStore) SaveWorldMapWorld(files []RawFileInsert, rows []WorldMapStateRow, capitals map[string]int, leaderTags []string, a WorldMapAddition) (int64, []int, error) {
	for _, file := range files {
		if err := validateRawFileStorage(file); err != nil {
			return 0, nil, err
		}
	}
	var id int64
	var ids []int
	tables := []string{rawFilesTable, additionsTable, statesTable, historyTable, characterTable}
	err := s.p.Write(tables, func(tx *project.Tx) error {
		if len(tx.Rows(additionsTable)) > 0 {
			return ErrWorldMapNotBase
		}

		replaced := append([]project.Row(nil), tx.Rows(statesTable)...)
		sort.SliceStable(replaced, func(i, j int) bool {
			return project.Int64(replaced[i]["state_id"]) < project.Int64(replaced[j]["state_id"])
		})
		kept := make([]any, 0, len(replaced))
		for _, row := range replaced {
			kept = append(kept, plain(row))
		}
		tx.Delete(statesTable, func(project.Row) bool { return true })
		var err error
		if ids, err = insertWorldMapStates(tx, 1, rows); err != nil {
			return err
		}

		histories := map[string]HistoryBefore{}
		tags := make([]string, 0, len(capitals))
		for tag := range capitals {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		for _, tag := range tags {
			state := capitals[tag]
			for _, row := range tx.Rows(historyTable) {
				if !strings.EqualFold(str(row["country_tag"]), tag) {
					continue
				}
				before := HistoryBefore{Source: str(row["source"])}
				if row["capital"] != nil {
					capital := intOf(row["capital"])
					before.Capital = &capital
				}
				historyID := project.Int64(row["id"])
				histories[strconv.FormatInt(historyID, 10)] = before
				if _, err := tx.Update(historyTable, historyID, func(r project.Row) {
					r["capital"], r["source"] = int64(state), "created"
				}); err != nil {
					return err
				}
			}
		}

		seedCharacterIDs := seedWorldMapLeaders(tx, leaderTags)

		previous := writeRawFiles(tx, files)
		options, err := worldMapOptionsWithSeedCharacters(a.Options, seedCharacterIDs)
		if err != nil {
			return err
		}
		a.Options, a.Kind, a.RegionID, a.StateIDs, a.StateBefore = options, "world", 0, ids, map[int]StateBefore{}
		id = insertRecord(tx, a, "world", previous, project.Row{
			"states_replaced": kept, "histories_before": plain(histories),
		})
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	return id, ids, nil
}

func seedWorldMapLeaders(tx *project.Tx, tags []string) []int64 {
	seen := map[string]bool{}
	ids := make([]int64, 0, len(tags))

	neutral := map[string]bool{}
	for _, ideology := range reftables.Ideologies().Members("neutrality") {
		neutral[ideology] = true
	}
	for _, rawTag := range tags {
		countryTag := strings.ToUpper(strings.TrimSpace(rawTag))
		if countryTag == "" || seen[countryTag] {
			continue
		}
		seen[countryTag] = true
		hasLeader := false
		taken := map[string]bool{}
		for _, row := range tx.Rows(characterTable) {
			taken[str(row["tag"])] = true
			if strings.ToUpper(str(row["country_tag"])) == countryTag && str(row["role_type"]) == "country_leader" &&
				str(row["source"]) != "imported" && neutral[strings.ToLower(str(row["ideology"]))] {
				hasLeader = true
			}
		}
		if hasLeader {
			continue
		}
		name := "Leader of " + countryTag
		baseTag := countryTag + "_new_world_leader"
		characterTag := baseTag
		for suffix := 1; taken[characterTag]; suffix++ {
			characterTag = fmt.Sprintf("%s_%d", baseTag, suffix+1)
		}
		row := tx.Insert(characterTable, project.Row{
			"country_tag": countryTag, "name": name, "tag": characterTag, "localisation_key": characterTag,
			"description": "", "role_type": "country_leader",
			"has_advisor_role": false, "is_replacement": false, "replaces_vanilla_id": false,
			"skill": int64(1), "attack_skill": int64(1), "defense_skill": int64(1), "planning_skill": int64(1),
			"logistics_skill": int64(1), "maneuvering_skill": int64(1), "coordination_skill": int64(1),
			"military_visible": "", "traits": []any{}, "advisor_traits": []any{}, "advisor_slot": "",
			"advisor_idea_token": "", "advisor_cost": int64(150),
			"scientist_data": map[string]any{}, "scientist_visible": "", "operative_skill_level": int64(1),
			"operative_traits": []any{}, "operative_nationalities": []any{}, "operative_available": "",
			"operative_bg": "", "name_en": name, "name_zh": "", "desc_en": "", "desc_zh": "",
			"display_name_locs": map[string]any{"en": name}, "description_locs": map[string]any{},
			"advisor_available": "", "advisor_visible": "", "advisor_removal_cost": int64(-1),
			"advisor_modifier": "", "advisor_on_add": "", "advisor_on_remove": "", "gender": "",
			"can_be_captured": true, "ideology": "despotism", "expire": "", "source": "created",
			"source_file": "", "is_portrait_modified": false,
			"large_portrait": "GFX_portrait_Europe_Generic_1", "small_portrait": "",
			"raw_extras": map[string]any{"generated_by": "world_map"}, "has_obs": false,
		})
		ids = append(ids, project.Int64(row["id"]))
	}
	return ids
}

func worldMapOptionsWithSeedCharacters(raw json.RawMessage, ids []int64) (json.RawMessage, error) {
	options := map[string]json.RawMessage{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &options); err != nil {
			return nil, fmt.Errorf("world map options: %w", err)
		}
		if options == nil {
			options = map[string]json.RawMessage{}
		}
	}
	if ids == nil {
		ids = []int64{}
	}
	seedIDs, _ := json.Marshal(ids)
	options["seed_character_ids"] = seedIDs
	body, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("world map seed characters: %w", err)
	}
	return body, nil
}
