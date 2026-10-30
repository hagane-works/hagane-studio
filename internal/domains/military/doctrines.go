package military

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const doctrineTable = "doctrines"

var (
	errDoctrineMissing     = apiutil.NotFoundError("Doctrine not found")
	errDoctrineUnavailable = detailError(http.StatusServiceUnavailable, "doctrine store unavailable")
)

var doctrineEntityTypes = []string{"folder", "grand_doctrine", "track", "subdoctrine"}

type Doctrine struct {
	ID                    int64             `json:"id"`
	ProjectID             int64             `json:"project_id"`
	EntityType            string            `json:"entity_type"`
	EntityKey             string            `json:"entity_key"`
	FolderKey             string            `json:"folder_key"`
	TrackKey              string            `json:"track_key"`
	Name                  string            `json:"name"`
	Description           string            `json:"description"`
	Icon                  string            `json:"icon"`
	XpCost                int               `json:"xp_cost"`
	XpType                string            `json:"xp_type"`
	Xor                   *string           `json:"xor"`
	AllowInMultipleTracks *bool             `json:"allow_in_multiple_tracks"`
	RawExtras             map[string]any    `json:"raw_extras"`
	SourceFile            string            `json:"source_file"`
	Source                string            `json:"source"`
	DisplayNameLocs       map[string]string `json:"display_name_locs"`
	DescriptionLocs       map[string]string `json:"description_locs"`
	CreatedAt             string            `json:"created_at"`
	UpdatedAt             string            `json:"updated_at"`
}

func doctrineFromRow(row project.Row) Doctrine {
	var d Doctrine
	fromRow(row, map[string]any{"entity_type": "grand_doctrine", "source": "created"}, &d)
	if d.RawExtras == nil {
		d.RawExtras = map[string]any{}
	}
	if d.DisplayNameLocs == nil {
		d.DisplayNameLocs = map[string]string{}
	}
	if d.DescriptionLocs == nil {
		d.DescriptionLocs = map[string]string{}
	}
	d.CreatedAt = isoNaive(row["created_at"])
	d.UpdatedAt = isoNaive(row["updated_at"])
	return d
}

func sanitizeSourceFile(raw string) (string, error) {
	s := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
	if s == "" {
		return "", nil
	}
	if len(s) >= 2 && s[1] == ':' {
		return "", fmt.Errorf("source_file must be mod-relative (no drive letter)")
	}
	s = strings.TrimLeft(s, "/")
	var parts []string
	for _, seg := range strings.Split(s, "/") {
		if seg == "" || seg == "." {
			continue
		}
		if seg == ".." {
			return "", fmt.Errorf("source_file may not contain '..' (path traversal)")
		}
		parts = append(parts, seg)
	}
	return strings.Join(parts, "/"), nil
}

func validateDoctrineEntityType(value string) (string, error) {
	v := strings.TrimSpace(value)
	for _, t := range doctrineEntityTypes {
		if v == t {
			return v, nil
		}
	}
	return "", detailError(http.StatusBadRequest, fmt.Sprintf(
		"entity_type must be one of ('folder', 'grand_doctrine', 'track', 'subdoctrine'), got '%s'", value))
}

func listDoctrines(p *project.Project, entityType string) ([]Doctrine, error) {
	rows, err := p.Rows(doctrineTable)
	if err != nil {
		return nil, err
	}
	filter := entityType != ""
	entityType = strings.TrimSpace(entityType)
	out := []Doctrine{}
	for _, row := range rows {
		if filter && !stringEquals(row["entity_type"], entityType) {
			continue
		}
		out = append(out, doctrineFromRow(row))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].EntityType != out[j].EntityType {
			return out[i].EntityType < out[j].EntityType
		}
		return out[i].EntityKey < out[j].EntityKey
	})
	return out, nil
}

func getDoctrine(p *project.Project, id int64) (Doctrine, error) {
	rows, err := p.Rows(doctrineTable)
	if err != nil {
		return Doctrine{}, err
	}
	if row := findRow(rows, id); row != nil {
		return doctrineFromRow(row), nil
	}
	return Doctrine{}, errDoctrineMissing
}

type doctrineCreateInput struct {
	EntityType            *string           `json:"entity_type"`
	EntityKey             string            `json:"entity_key"`
	FolderKey             string            `json:"folder_key"`
	TrackKey              string            `json:"track_key"`
	Name                  string            `json:"name"`
	Description           string            `json:"description"`
	Icon                  string            `json:"icon"`
	XpCost                int               `json:"xp_cost"`
	XpType                string            `json:"xp_type"`
	Xor                   *string           `json:"xor"`
	AllowInMultipleTracks *bool             `json:"allow_in_multiple_tracks"`
	RawExtras             map[string]any    `json:"raw_extras"`
	SourceFile            string            `json:"source_file"`
	Source                *string           `json:"source"`
	DisplayNameLocs       map[string]string `json:"display_name_locs"`
	DescriptionLocs       map[string]string `json:"description_locs"`
}

func doctrineKeyTaken(tx *project.Tx, id int64, entityType, entityKey string) bool {
	for _, row := range tx.Rows(doctrineTable) {
		if project.Int64(row["id"]) != id && text(row, "entity_type") == entityType && text(row, "entity_key") == entityKey {
			return true
		}
	}
	return false
}

func createDoctrine(p *project.Project, in doctrineCreateInput) (Doctrine, error) {
	entityType, err := validateDoctrineEntityType(stringPtr(in.EntityType, "grand_doctrine"))
	if err != nil {
		return Doctrine{}, err
	}
	entityKey := strings.TrimSpace(in.EntityKey)
	if entityKey == "" {
		return Doctrine{}, detailError(http.StatusBadRequest, "entity_key is required (e.g. 'land', 'new_mobile_warfare').")
	}
	sourceFile, err := sanitizeSourceFile(in.SourceFile)
	if err != nil {
		return Doctrine{}, detailError(http.StatusUnprocessableEntity, err.Error())
	}
	if in.RawExtras == nil {
		in.RawExtras = map[string]any{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	var created Doctrine
	err = p.Write([]string{doctrineTable}, func(tx *project.Tx) error {
		if doctrineKeyTaken(tx, 0, entityType, entityKey) {
			return detailError(http.StatusConflict, fmt.Sprintf(
				"A %s with key '%s' already exists in this project. Pick a unique entity_key.", entityType, entityKey))
		}
		row := tx.Insert(doctrineTable, project.Row{
			"entity_type": entityType, "entity_key": entityKey,
			"folder_key": in.FolderKey, "track_key": in.TrackKey, "name": in.Name,
			"description": in.Description, "icon": in.Icon, "xp_cost": jsonValue(in.XpCost),
			"xp_type": in.XpType, "xor": jsonValue(in.Xor),
			"allow_in_multiple_tracks": jsonValue(in.AllowInMultipleTracks),
			"raw_extras":               jsonValue(in.RawExtras), "source_file": sourceFile,
			"source":            stringPtr(in.Source, "created"),
			"display_name_locs": jsonValue(in.DisplayNameLocs), "description_locs": jsonValue(in.DescriptionLocs),
		})
		created = doctrineFromRow(row)
		return nil
	})
	return created, err
}

type doctrinePatch struct {
	EntityType            workbench.Optional[string]            `json:"entity_type"`
	EntityKey             workbench.Optional[string]            `json:"entity_key"`
	FolderKey             workbench.Optional[string]            `json:"folder_key"`
	TrackKey              workbench.Optional[string]            `json:"track_key"`
	Name                  workbench.Optional[string]            `json:"name"`
	Description           workbench.Optional[string]            `json:"description"`
	Icon                  workbench.Optional[string]            `json:"icon"`
	XpCost                workbench.Optional[int]               `json:"xp_cost"`
	XpType                workbench.Optional[string]            `json:"xp_type"`
	Xor                   workbench.Optional[string]            `json:"xor"`
	AllowInMultipleTracks workbench.Optional[bool]              `json:"allow_in_multiple_tracks"`
	RawExtras             workbench.Optional[map[string]any]    `json:"raw_extras"`
	SourceFile            workbench.Optional[string]            `json:"source_file"`
	DisplayNameLocs       workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs       workbench.Optional[map[string]string] `json:"description_locs"`
}

func updateDoctrine(p *project.Project, id int64, patch doctrinePatch) (Doctrine, error) {
	var updated Doctrine
	err := p.Write([]string{doctrineTable}, func(tx *project.Tx) error {
		current := tx.Find(doctrineTable, id)
		if current == nil {
			return errDoctrineMissing
		}
		cur := doctrineFromRow(current)
		changes := map[string]any{}
		if patch.EntityType.Set {
			validated, err := validateDoctrineEntityType(optStr(patch.EntityType))
			if err != nil {
				return err
			}
			changes["entity_type"] = validated
		}
		if patch.EntityKey.Set {
			trimmed := strings.TrimSpace(optStr(patch.EntityKey))
			if trimmed == "" {
				return detailError(http.StatusBadRequest, "entity_key cannot be empty")
			}
			changes["entity_key"] = trimmed
		}
		if patch.SourceFile.Set {
			sanitized, err := sanitizeSourceFile(optStr(patch.SourceFile))
			if err != nil {
				return detailError(http.StatusUnprocessableEntity, err.Error())
			}
			changes["source_file"] = sanitized
		}

		finalRawExtras := cur.RawExtras
		rawExtrasTouched := false
		if patch.RawExtras.Set {
			rawExtrasTouched = true
			if patch.RawExtras.Value != nil {
				finalRawExtras = *patch.RawExtras.Value
			} else {
				finalRawExtras = nil
			}
		}
		promoted := []string{}
		if patch.Xor.Set {
			promoted = append(promoted, "xor")
		}
		if patch.AllowInMultipleTracks.Set {
			promoted = append(promoted, "allow_in_multiple_tracks")
		}
		if len(promoted) > 0 && finalRawExtras != nil {
			hasCopy := false
			for _, k := range promoted {
				if _, ok := finalRawExtras[k]; ok {
					hasCopy = true
				}
			}
			if hasCopy {
				purged := map[string]any{}
				for k, v := range finalRawExtras {
					purged[k] = v
				}
				for _, k := range promoted {
					delete(purged, k)
				}
				finalRawExtras = purged
				rawExtrasTouched = true
			}
		}

		set := func(column string, isSet bool, value any) {
			if isSet {
				changes[column] = value
			}
		}
		set("folder_key", patch.FolderKey.Set, opt(patch.FolderKey))
		set("track_key", patch.TrackKey.Set, opt(patch.TrackKey))
		set("name", patch.Name.Set, opt(patch.Name))
		set("description", patch.Description.Set, opt(patch.Description))
		set("icon", patch.Icon.Set, opt(patch.Icon))
		set("xp_cost", patch.XpCost.Set, opt(patch.XpCost))
		set("xp_type", patch.XpType.Set, opt(patch.XpType))
		set("xor", patch.Xor.Set, opt(patch.Xor))
		set("allow_in_multiple_tracks", patch.AllowInMultipleTracks.Set, opt(patch.AllowInMultipleTracks))
		if rawExtrasTouched {
			var value any
			if finalRawExtras != nil {
				value = jsonValue(finalRawExtras)
			}
			changes["raw_extras"] = value
		}
		set("display_name_locs", patch.DisplayNameLocs.Set, opt(patch.DisplayNameLocs))
		set("description_locs", patch.DescriptionLocs.Set, opt(patch.DescriptionLocs))
		if cur.Source == "imported" {
			changes["source"] = "created"
		}
		if len(changes) == 0 {
			updated = cur
			return nil
		}
		if nullViolation(doctrineTable, changes) {
			return errDoctrineUnavailable
		}
		entityType, entityKey := cur.EntityType, cur.EntityKey
		if v, ok := changes["entity_type"].(string); ok {
			entityType = v
		}
		if v, ok := changes["entity_key"].(string); ok {
			entityKey = v
		}
		if doctrineKeyTaken(tx, id, entityType, entityKey) {
			return detailError(http.StatusConflict, "entity_type + entity_key conflict in this project")
		}
		row, err := tx.Update(doctrineTable, id, func(row project.Row) {
			for column, value := range changes {
				row[column] = value
			}
		})
		if err != nil {
			return err
		}
		updated = doctrineFromRow(row)
		return nil
	})
	return updated, err
}

func (h *handlers) doctrines(w http.ResponseWriter, r *http.Request, p *project.Project, rest []string) {
	failDoctrine := func(err error, op string) { fail(w, err, op, "doctrine store unavailable") }
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			items, err := listDoctrines(p, r.URL.Query().Get("entity_type"))
			if err != nil {
				failDoctrine(err, "list doctrines")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"doctrines": items, "items": items, "total": len(items)})
		case http.MethodPost:
			var in doctrineCreateInput
			if err := decodeBody(r, 4<<20, &in); err != nil {
				detail(w, http.StatusUnprocessableEntity, "Invalid doctrine payload")
				return
			}
			item, err := createDoctrine(p, in)
			if err != nil {
				failDoctrine(err, "create doctrine")
				return
			}
			writeJSON(w, http.StatusCreated, item)
		default:
			methodError(w, "GET or POST only")
		}
		return
	}
	id, ok := itemID(rest)
	if !ok {
		detail(w, http.StatusNotFound, "Doctrine not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := getDoctrine(p, id)
		if err != nil {
			failDoctrine(err, "get doctrine")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var patch doctrinePatch
		if err := decodeBody(r, 4<<20, &patch); err != nil {
			detail(w, http.StatusUnprocessableEntity, "Invalid doctrine payload")
			return
		}
		item, err := updateDoctrine(p, id, patch)
		if err != nil {
			failDoctrine(err, "update doctrine")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := deleteRow(p, doctrineTable, id, errDoctrineMissing); err != nil {
			failDoctrine(err, "delete doctrine")
			return
		}
		apiutil.NoContent(w)
	default:
		methodError(w, "GET, PUT or DELETE only")
	}
}
