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

const mioTable = "military_industrial_organizations"

var (
	errMIOMissing     = apiutil.NotFoundError("MIO not found")
	errMIOUnavailable = detailError(http.StatusServiceUnavailable, "store unavailable")
)

type MIO struct {
	ID                 int64             `json:"id"`
	ProjectID          int64             `json:"project_id"`
	MioID              string            `json:"mio_id"`
	Include            string            `json:"include"`
	Icon               string            `json:"icon"`
	EquipmentType      []string          `json:"equipment_type"`
	ResearchCategories []string          `json:"research_categories"`
	Allowed            string            `json:"allowed"`
	Available          string            `json:"available"`
	Visible            string            `json:"visible"`
	Traits             []map[string]any  `json:"traits"`
	RawExtras          map[string]any    `json:"raw_extras"`
	CountryTag         string            `json:"country_tag"`
	ScopeKind          string            `json:"scope_kind"`
	SharedCountryTags  []string          `json:"shared_country_tags"`
	Source             string            `json:"source"`
	NameEn             string            `json:"name_en"`
	NameZh             string            `json:"name_zh"`
	DescEn             string            `json:"desc_en"`
	DescZh             string            `json:"desc_zh"`
	DisplayNameLocs    map[string]string `json:"display_name_locs"`
	DescriptionLocs    map[string]string `json:"description_locs"`
	CreatedAt          string            `json:"created_at"`
	UpdatedAt          string            `json:"updated_at"`
}

func mioFromRow(row project.Row) MIO {
	var m MIO
	fromRow(row, map[string]any{"scope_kind": "own", "source": "created"}, &m)
	if m.EquipmentType == nil {
		m.EquipmentType = []string{}
	}
	if m.ResearchCategories == nil {
		m.ResearchCategories = []string{}
	}
	if m.Traits == nil {
		m.Traits = []map[string]any{}
	}
	if m.RawExtras == nil {
		m.RawExtras = map[string]any{}
	}
	if m.DisplayNameLocs == nil {
		m.DisplayNameLocs = map[string]string{}
	}
	if m.DescriptionLocs == nil {
		m.DescriptionLocs = map[string]string{}
	}

	m.CreatedAt = isoNaive(row["created_at"])
	m.UpdatedAt = isoNaive(row["updated_at"])
	return m
}

func listMIOs(p *project.Project) ([]MIO, error) {
	rows, err := p.Rows(mioTable)
	if err != nil {
		return nil, err
	}
	out := make([]MIO, 0, len(rows))
	for _, row := range rows {
		out = append(out, mioFromRow(row))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].MioID < out[j].MioID })
	return out, nil
}

func getMIO(p *project.Project, id int64) (MIO, error) {
	rows, err := p.Rows(mioTable)
	if err != nil {
		return MIO{}, err
	}
	if row := findRow(rows, id); row != nil {
		return mioFromRow(row), nil
	}
	return MIO{}, errMIOMissing
}

type MIOCreateInput struct {
	MioID              string            `json:"mio_id"`
	Include            string            `json:"include"`
	Icon               string            `json:"icon"`
	EquipmentType      []string          `json:"equipment_type"`
	ResearchCategories []string          `json:"research_categories"`
	Allowed            string            `json:"allowed"`
	Available          string            `json:"available"`
	Visible            string            `json:"visible"`
	Traits             []map[string]any  `json:"traits"`
	RawExtras          map[string]any    `json:"raw_extras"`
	CountryTag         string            `json:"country_tag"`
	ScopeKind          *string           `json:"scope_kind"`
	SharedCountryTags  []string          `json:"shared_country_tags"`
	Source             *string           `json:"source"`
	NameEn             string            `json:"name_en"`
	NameZh             string            `json:"name_zh"`
	DescEn             string            `json:"desc_en"`
	DescZh             string            `json:"desc_zh"`
	DisplayNameLocs    map[string]string `json:"display_name_locs"`
	DescriptionLocs    map[string]string `json:"description_locs"`
}

func mioIDTaken(tx *project.Tx, id int64, mioID string) bool {
	for _, row := range tx.Rows(mioTable) {
		if project.Int64(row["id"]) != id && text(row, "mio_id") == mioID {
			return true
		}
	}
	return false
}

func createMIO(write writer, in MIOCreateInput) (MIO, error) {
	mioID := strings.TrimSpace(in.MioID)
	if mioID == "" {
		return MIO{}, detailError(http.StatusBadRequest, "mio_id is required (e.g. 'USA_marmon_herrington_organization')")
	}
	if in.EquipmentType == nil {
		in.EquipmentType = []string{}
	}
	if in.ResearchCategories == nil {
		in.ResearchCategories = []string{}
	}
	if in.Traits == nil {
		in.Traits = []map[string]any{}
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
	var created MIO
	err := write([]string{mioTable}, func(tx *project.Tx) error {
		if mioIDTaken(tx, 0, mioID) {
			return detailError(http.StatusConflict, fmt.Sprintf(
				"mio_id '%s' already exists in this project. Pick a unique mio_id.", mioID))
		}
		row := tx.Insert(mioTable, project.Row{
			"mio_id": mioID, "include": in.Include, "icon": in.Icon,
			"equipment_type": jsonValue(in.EquipmentType), "research_categories": jsonValue(in.ResearchCategories),
			"allowed": in.Allowed, "available": in.Available, "visible": in.Visible,
			"traits": jsonValue(in.Traits), "raw_extras": jsonValue(in.RawExtras),
			"country_tag": in.CountryTag, "scope_kind": stringPtr(in.ScopeKind, "own"),
			"shared_country_tags": jsonValue(in.SharedCountryTags),
			"source":              stringPtr(in.Source, "created"),
			"name_en":             in.NameEn, "name_zh": in.NameZh, "desc_en": in.DescEn, "desc_zh": in.DescZh,
			"display_name_locs": jsonValue(in.DisplayNameLocs), "description_locs": jsonValue(in.DescriptionLocs),
		})
		created = mioFromRow(row)
		return nil
	})
	return created, err
}

type mioPatch struct {
	MioID              workbench.Optional[string]            `json:"mio_id"`
	Include            workbench.Optional[string]            `json:"include"`
	Icon               workbench.Optional[string]            `json:"icon"`
	EquipmentType      workbench.Optional[[]string]          `json:"equipment_type"`
	ResearchCategories workbench.Optional[[]string]          `json:"research_categories"`
	Allowed            workbench.Optional[string]            `json:"allowed"`
	Available          workbench.Optional[string]            `json:"available"`
	Visible            workbench.Optional[string]            `json:"visible"`
	Traits             workbench.Optional[[]map[string]any]  `json:"traits"`
	RawExtras          workbench.Optional[map[string]any]    `json:"raw_extras"`
	CountryTag         workbench.Optional[string]            `json:"country_tag"`
	ScopeKind          workbench.Optional[string]            `json:"scope_kind"`
	SharedCountryTags  workbench.Optional[[]string]          `json:"shared_country_tags"`
	NameEn             workbench.Optional[string]            `json:"name_en"`
	NameZh             workbench.Optional[string]            `json:"name_zh"`
	DescEn             workbench.Optional[string]            `json:"desc_en"`
	DescZh             workbench.Optional[string]            `json:"desc_zh"`
	DisplayNameLocs    workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs    workbench.Optional[map[string]string] `json:"description_locs"`
}

func updateMIO(p *project.Project, id int64, patch mioPatch) (MIO, error) {
	var updated MIO
	err := p.Write([]string{mioTable}, func(tx *project.Tx) error {
		current := tx.Find(mioTable, id)
		if current == nil {
			return errMIOMissing
		}
		changes := map[string]any{}
		if patch.MioID.Set {
			trimmed := strings.TrimSpace(optStr(patch.MioID))
			if trimmed == "" {
				return detailError(http.StatusBadRequest, "mio_id cannot be empty")
			}
			changes["mio_id"] = trimmed
		}
		set := func(column string, isSet bool, value any) {
			if isSet {
				changes[column] = value
			}
		}
		set("include", patch.Include.Set, opt(patch.Include))
		set("icon", patch.Icon.Set, opt(patch.Icon))
		set("equipment_type", patch.EquipmentType.Set, opt(patch.EquipmentType))
		set("research_categories", patch.ResearchCategories.Set, opt(patch.ResearchCategories))
		set("allowed", patch.Allowed.Set, opt(patch.Allowed))
		set("available", patch.Available.Set, opt(patch.Available))
		set("visible", patch.Visible.Set, opt(patch.Visible))
		set("traits", patch.Traits.Set, opt(patch.Traits))
		set("raw_extras", patch.RawExtras.Set, opt(patch.RawExtras))
		set("country_tag", patch.CountryTag.Set, opt(patch.CountryTag))
		set("scope_kind", patch.ScopeKind.Set, opt(patch.ScopeKind))
		set("shared_country_tags", patch.SharedCountryTags.Set, opt(patch.SharedCountryTags))
		set("name_en", patch.NameEn.Set, opt(patch.NameEn))
		set("name_zh", patch.NameZh.Set, opt(patch.NameZh))
		set("desc_en", patch.DescEn.Set, opt(patch.DescEn))
		set("desc_zh", patch.DescZh.Set, opt(patch.DescZh))
		set("display_name_locs", patch.DisplayNameLocs.Set, opt(patch.DisplayNameLocs))
		set("description_locs", patch.DescriptionLocs.Set, opt(patch.DescriptionLocs))
		if textOr(current, "source", "created") == "imported" {
			changes["source"] = "created"
		}
		if len(changes) == 0 {
			updated = mioFromRow(current)
			return nil
		}
		if nullViolation(mioTable, changes) {
			return errMIOUnavailable
		}
		if mioID, ok := changes["mio_id"].(string); ok && mioIDTaken(tx, id, mioID) {
			return detailError(http.StatusConflict, "mio_id conflict in this project")
		}
		row, err := tx.Update(mioTable, id, func(row project.Row) {
			for column, value := range changes {
				row[column] = value
			}
		})
		if err != nil {
			return err
		}
		updated = mioFromRow(row)
		return nil
	})
	return updated, err
}

func (h *handlers) mios(w http.ResponseWriter, r *http.Request, p *project.Project, rest []string) {
	const missing = "MIO not found"
	if h.mioIconRoute(w, r, p, rest) {
		return
	}
	failMIO := func(err error, op string) { fail(w, err, op, "store unavailable") }
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			items, err := listMIOs(p)
			if err != nil {
				failMIO(err, "list mios")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"mios": items, "items": items, "total": len(items)})
		case http.MethodPost:
			var in MIOCreateInput
			if err := decodeBody(r, 4<<20, &in); err != nil {
				detail(w, http.StatusUnprocessableEntity, "Invalid MIO payload")
				return
			}
			item, err := createMIO(p.Write, in)
			if err != nil {
				failMIO(err, "create mio")
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
		detail(w, http.StatusNotFound, missing)
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := getMIO(p, id)
		if err != nil {
			failMIO(err, "get mio")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var patch mioPatch
		if err := decodeBody(r, 4<<20, &patch); err != nil {
			detail(w, http.StatusUnprocessableEntity, "Invalid MIO payload")
			return
		}
		item, err := updateMIO(p, id, patch)
		if err != nil {
			failMIO(err, "update mio")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := deleteRow(p, mioTable, id, errMIOMissing); err != nil {
			failMIO(err, "delete mio")
			return
		}
		apiutil.NoContent(w)
	default:
		methodError(w, "GET, PUT or DELETE only")
	}
}
