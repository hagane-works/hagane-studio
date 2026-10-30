package content

import (
	"fmt"
	"net/http"
	"strings"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
	emitscripts "haganeworks/internal/hoi4/emit/scripts"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const agenciesTable = "intelligence_agencies"

type intelligenceAgency struct {
	ID                int64             `json:"id"`
	ProjectID         int64             `json:"project_id"`
	AgencyID          string            `json:"agency_id"`
	Picture           string            `json:"picture"`
	Names             []string          `json:"names"`
	DefaultTrigger    string            `json:"default_trigger"`
	Available         string            `json:"available"`
	UpgradeBranches   []map[string]any  `json:"upgrade_branches"`
	RawExtras         map[string]any    `json:"raw_extras"`
	CountryTag        string            `json:"country_tag"`
	ScopeKind         string            `json:"scope_kind"`
	SharedCountryTags []string          `json:"shared_country_tags"`
	Source            string            `json:"source"`
	NameEn            string            `json:"name_en"`
	NameZh            string            `json:"name_zh"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
}

func agencyFromRow(row project.Row) intelligenceAgency {
	item := intelligenceAgency{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		AgencyID: text(row, "agency_id"), Picture: text(row, "picture"),
		DefaultTrigger: text(row, "default_trigger"), Available: text(row, "available"),
		RawExtras: anyMap(row["raw_extras"]), CountryTag: text(row, "country_tag"),
		ScopeKind: textOr(row, "scope_kind", "own"), Source: textOr(row, "source", "created"),
		NameEn: text(row, "name_en"), NameZh: text(row, "name_zh"),
		DisplayNameLocs: stringMap(row["display_name_locs"]),
		CreatedAt:       isoNaive(row["created_at"]), UpdatedAt: isoNaive(row["updated_at"]),
	}
	decode(row["names"], &item.Names)
	decode(row["upgrade_branches"], &item.UpgradeBranches)
	decode(row["shared_country_tags"], &item.SharedCountryTags)
	if item.Names == nil {
		item.Names = []string{}
	}
	if item.UpgradeBranches == nil {
		item.UpgradeBranches = []map[string]any{}
	}
	return item
}

type agencyCreate struct {
	AgencyID          string            `json:"agency_id"`
	Picture           string            `json:"picture"`
	Names             []string          `json:"names"`
	DefaultTrigger    string            `json:"default_trigger"`
	Available         string            `json:"available"`
	UpgradeBranches   []map[string]any  `json:"upgrade_branches"`
	RawExtras         map[string]any    `json:"raw_extras"`
	CountryTag        string            `json:"country_tag"`
	ScopeKind         *string           `json:"scope_kind"`
	SharedCountryTags []string          `json:"shared_country_tags"`
	Source            *string           `json:"source"`
	NameEn            string            `json:"name_en"`
	NameZh            string            `json:"name_zh"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
}

type agencyPatch struct {
	AgencyID          workbench.Optional[string]            `json:"agency_id"`
	Picture           workbench.Optional[string]            `json:"picture"`
	Names             workbench.Optional[[]string]          `json:"names"`
	DefaultTrigger    workbench.Optional[string]            `json:"default_trigger"`
	Available         workbench.Optional[string]            `json:"available"`
	UpgradeBranches   workbench.Optional[[]map[string]any]  `json:"upgrade_branches"`
	RawExtras         workbench.Optional[map[string]any]    `json:"raw_extras"`
	CountryTag        workbench.Optional[string]            `json:"country_tag"`
	ScopeKind         workbench.Optional[string]            `json:"scope_kind"`
	SharedCountryTags workbench.Optional[[]string]          `json:"shared_country_tags"`
	NameEn            workbench.Optional[string]            `json:"name_en"`
	NameZh            workbench.Optional[string]            `json:"name_zh"`
	DisplayNameLocs   workbench.Optional[map[string]string] `json:"display_name_locs"`
}

func (h *handlers) agencies(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	if h.assetRoute(p, w, r, rest, "upload-logo", "logo", agencyLogoSpec, func(w http.ResponseWriter, id int64, spec assetSpec) error {
		writeJSON(w, http.StatusOK, map[string]any{
			"filename": spec.filename, "warnings": []string{}, "picture": spec.gfxName,
			"preview_url": fmt.Sprintf("/api/go/projects/%d/intelligence-agencies/%d/logo", p.ID(), id),
		})
		return nil
	}) {
		return
	}
	const missing = "Intelligence agency not found"
	crud{
		listKey: "agencies", missing: missing, unavailable: "store unavailable",
		list: func(p *project.Project, _ *http.Request) (any, int, error) {
			items, err := listRows(p, agenciesTable, byText("agency_id"), agencyFromRow)
			return items, len(items), err
		},
		create: createAgency,
		get: func(p *project.Project, id int64) (any, error) {
			row, err := getRow(p, agenciesTable, id, missing)
			if err != nil {
				return nil, err
			}
			return agencyFromRow(row), nil
		},
		update: updateAgency,
		remove: removeRow(agenciesTable, missing),
	}.serve(p, w, r, rest)
}

func agencyLogoSpec(p *project.Project, id int64) (assetSpec, error) {
	rows, err := p.Rows(agenciesTable)
	if err != nil {
		return assetSpec{}, err
	}
	row := findRow(rows, id)
	if row == nil {
		return assetSpec{}, errAssetTargetMissing
	}
	agencyID := text(row, "agency_id")
	gfxName := emitscripts.IntelAgencyLogoSpriteName(agencyID)
	return assetSpec{
		table: agenciesTable, entityType: "intel_agency_logo", obsKind: "agency",
		filename: "intel_agency_" + emitmisc.Slugify(agencyID) + ".png", gfxName: gfxName,
		apply: func(row project.Row) { row["picture"] = gfxName },
	}, nil
}

func createAgency(p *project.Project, r *http.Request) (any, error) {
	var in agencyCreate
	if err := decodeBody(r, &in); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, "Invalid agency payload")
	}
	agencyID := strings.TrimSpace(in.AgencyID)
	if agencyID == "" {
		return nil, statusErr(http.StatusBadRequest, "agency_id is required (e.g. 'usa_oss')")
	}
	if in.Names == nil {
		in.Names = []string{}
	}
	if in.UpgradeBranches == nil {
		in.UpgradeBranches = []map[string]any{}
	}
	if in.RawExtras == nil {
		in.RawExtras = map[string]any{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	scopeKind := "own"
	if in.ScopeKind != nil {
		scopeKind = *in.ScopeKind
	}
	row, err := writeRow(p, agenciesTable, func(tx *project.Tx) (project.Row, error) {
		if taken(tx.Rows(agenciesTable), 0, func(row project.Row) bool { return text(row, "agency_id") == agencyID }) {
			return nil, statusErr(http.StatusConflict, fmt.Sprintf(
				"agency_id '%s' already exists in this project. Pick a unique agency_id.", agencyID))
		}
		return insert(tx, agenciesTable, project.Row{
			"agency_id": agencyID, "picture": in.Picture, "names": jsonValue(in.Names),
			"default_trigger": in.DefaultTrigger, "available": in.Available,
			"upgrade_branches": jsonValue(in.UpgradeBranches), "raw_extras": jsonValue(in.RawExtras),
			"country_tag": in.CountryTag, "scope_kind": scopeKind,
			"shared_country_tags": jsonValue(in.SharedCountryTags),
			"source":              workbench.StrDefault(in.Source, "created"),
			"name_en":             in.NameEn, "name_zh": in.NameZh,
			"display_name_locs": jsonValue(in.DisplayNameLocs),

			"description_locs": map[string]any{},
		})
	})
	if err != nil {
		return nil, err
	}
	return agencyFromRow(row), nil
}

func updateAgency(p *project.Project, r *http.Request, id int64) (any, error) {
	var patch agencyPatch
	if err := decodeBody(r, &patch); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, "Invalid agency payload")
	}
	cur, err := getRow(p, agenciesTable, id, "Intelligence agency not found")
	if err != nil {
		return nil, err
	}
	sets := project.Row{}
	if patch.AgencyID.Set {
		agencyID := strings.TrimSpace(optString(patch.AgencyID))
		if agencyID == "" {
			return nil, statusErr(http.StatusBadRequest, "agency_id cannot be empty")
		}
		sets["agency_id"] = agencyID
	}
	setOpt(sets, "picture", patch.Picture)
	setOptJSON(sets, "names", patch.Names)
	setOpt(sets, "default_trigger", patch.DefaultTrigger)
	setOpt(sets, "available", patch.Available)
	setOptJSON(sets, "upgrade_branches", patch.UpgradeBranches)
	setOptJSON(sets, "raw_extras", patch.RawExtras)
	setOpt(sets, "country_tag", patch.CountryTag)
	setOpt(sets, "scope_kind", patch.ScopeKind)
	setOptJSON(sets, "shared_country_tags", patch.SharedCountryTags)
	setOpt(sets, "name_en", patch.NameEn)
	setOpt(sets, "name_zh", patch.NameZh)
	setOptJSON(sets, "display_name_locs", patch.DisplayNameLocs)
	if text(cur, "source") == "imported" {
		sets["source"] = "created"
	}
	row, err := updateRow(p, agenciesTable, id, sets, func(tx *project.Tx) error {
		if agencyID, ok := sets["agency_id"].(string); ok && taken(tx.Rows(agenciesTable), id, func(row project.Row) bool {
			return text(row, "agency_id") == agencyID
		}) {
			return statusErr(http.StatusConflict, "agency_id conflict in this project")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return agencyFromRow(row), nil
}
