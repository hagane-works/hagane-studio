package techlifecycle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	CodeProjectInvalid           = "tech_branch_project_invalid"
	CodeProjectNotFound          = "tech_branch_project_not_found"
	CodeBranchNotFound           = "tech_branch_not_found"
	CodeBranchProjectMismatch    = "tech_branch_project_mismatch"
	CodeBranchIdentityInvalid    = "tech_branch_identity_invalid"
	CodeBranchIdentityAmbiguous  = "tech_branch_identity_ambiguous"
	CodeBranchMemberMissing      = "tech_branch_member_missing"
	CodeBranchMemberProject      = "tech_branch_member_project_mismatch"
	CodeImmutableField           = "tech_branch_field_profile_owned"
	CodeUnknownField             = "tech_branch_field_unknown"
	CodeEquipmentInvalid         = "tech_branch_equipment_invalid"
	CodeEquipmentMissing         = "tech_branch_equipment_missing"
	CodeEquipmentProjectMismatch = "tech_branch_equipment_project_mismatch"
	CodeDeleteBlocked            = "tech_branch_delete_blocked"
)

type ContractError struct {
	Code   string `json:"code"`
	Field  string `json:"field,omitempty"`
	Detail string `json:"detail"`
}

func (e ContractError) Error() string { return e.Code + ": " + e.Detail }

func contractError(code, field, detail string) error {
	return ContractError{Code: code, Field: field, Detail: detail}
}

type Field[T any] struct {
	Set   bool
	Value *T
}

func (f *Field[T]) UnmarshalJSON(data []byte) error {
	f.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		f.Value = nil
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	f.Value = &value
	return nil
}

type UpdateRequest struct {
	Name             Field[string]
	NameEn           Field[string]
	NameZh           Field[string]
	DescEn           Field[string]
	DescZh           Field[string]
	DisplayNameLocs  Field[map[string]string]
	DescriptionLocs  Field[map[string]string]
	EnableEquipments Field[[]string]

	ReadOnlyFields []string
	UnknownFields  []string
}

var readOnlyUpdateFields = map[string]bool{
	"project_id": true, "id": true, "profile_id": true,
	"baseline_sha256": true, "schema_version": true,
	"surface": true, "folder": true, "slot": true, "gridbox_id": true,
	"root_technology_id": true, "technology_ids": true, "technology_db_id": true,
	"tech_line": true, "start_year": true, "position_x": true, "position_y": true,
	"folder_placements": true, "path": true, "source": true,
}

func decodeField[T any](raw json.RawMessage, target *Field[T]) error {
	return target.UnmarshalJSON(raw)
}

func (u *UpdateRequest) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*u = UpdateRequest{}
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		var err error
		switch key {
		case "name":
			err = decodeField(raw[key], &u.Name)
		case "name_en":
			err = decodeField(raw[key], &u.NameEn)
		case "name_zh":
			err = decodeField(raw[key], &u.NameZh)
		case "desc_en":
			err = decodeField(raw[key], &u.DescEn)
		case "desc_zh":
			err = decodeField(raw[key], &u.DescZh)
		case "display_name_locs":
			err = decodeField(raw[key], &u.DisplayNameLocs)
		case "description_locs":
			err = decodeField(raw[key], &u.DescriptionLocs)
		case "enable_equipments":
			err = decodeField(raw[key], &u.EnableEquipments)
		default:
			if readOnlyUpdateFields[key] {
				u.ReadOnlyFields = append(u.ReadOnlyFields, key)
			} else {
				u.UnknownFields = append(u.UnknownFields, key)
			}
		}
		if err != nil {
			return fmt.Errorf("decode technology branch field %q: %w", key, err)
		}
	}
	return nil
}

func (u UpdateRequest) MarshalJSON() ([]byte, error) {
	out := map[string]any{}
	put := func(key string, set bool, value any) {
		if set {
			out[key] = value
		}
	}
	put("name", u.Name.Set, u.Name.Value)
	put("name_en", u.NameEn.Set, u.NameEn.Value)
	put("name_zh", u.NameZh.Set, u.NameZh.Value)
	put("desc_en", u.DescEn.Set, u.DescEn.Value)
	put("desc_zh", u.DescZh.Set, u.DescZh.Value)
	put("display_name_locs", u.DisplayNameLocs.Set, u.DisplayNameLocs.Value)
	put("description_locs", u.DescriptionLocs.Set, u.DescriptionLocs.Value)
	put("enable_equipments", u.EnableEquipments.Set, u.EnableEquipments.Value)
	return json.Marshal(out)
}

type State struct {
	Projects           []int64             `json:"projects"`
	Branches           []Branch            `json:"branches"`
	Technologies       []Technology        `json:"technologies"`
	Equipments         []Equipment         `json:"equipments"`
	ExternalReferences []ExternalReference `json:"external_references"`
}

type Branch struct {
	ProjectID        int64    `json:"project_id"`
	ID               string   `json:"id"`
	ProfileID        string   `json:"profile_id"`
	BaselineSHA256   string   `json:"baseline_sha256"`
	SchemaVersion    int      `json:"schema_version"`
	Surface          string   `json:"surface"`
	Folder           string   `json:"folder"`
	Slot             string   `json:"slot"`
	GridboxID        string   `json:"gridbox_id"`
	RootTechnologyID string   `json:"root_technology_id"`
	TechnologyIDs    []string `json:"technology_ids"`
}

type FolderPlacement struct {
	Folder string  `json:"folder"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

type PathEdge struct {
	LeadsToTechnology string `json:"leads_to_tech"`
}

type Technology struct {
	DatabaseID       int64             `json:"technology_db_id"`
	ProjectID        int64             `json:"project_id"`
	TechID           string            `json:"tech_id"`
	Source           string            `json:"source"`
	Name             string            `json:"name"`
	NameEn           string            `json:"name_en"`
	NameZh           string            `json:"name_zh"`
	DescEn           string            `json:"desc_en"`
	DescZh           string            `json:"desc_zh"`
	DisplayNameLocs  map[string]string `json:"display_name_locs"`
	DescriptionLocs  map[string]string `json:"description_locs"`
	EnableEquipments []string          `json:"enable_equipments"`
	TechLine         string            `json:"tech_line"`
	Folder           string            `json:"folder"`
	StartYear        int               `json:"start_year"`
	PositionX        float64           `json:"position_x"`
	PositionY        float64           `json:"position_y"`
	FolderPlacements []FolderPlacement `json:"folder_placements"`
	Path             []PathEdge        `json:"path"`
}

type Equipment struct {
	ProjectID   int64  `json:"project_id"`
	EquipmentID string `json:"equipment_id"`
}

type ExternalReference struct {
	ProjectID          int64  `json:"project_id"`
	SourceKind         string `json:"source_kind"`
	SourceID           string `json:"source_id"`
	Field              string `json:"field"`
	TargetTechnologyID string `json:"target_technology_id"`
}

type BranchView struct {
	ProjectID        int64    `json:"project_id"`
	ID               string   `json:"id"`
	ProfileID        string   `json:"profile_id"`
	BaselineSHA256   string   `json:"baseline_sha256"`
	SchemaVersion    int      `json:"schema_version"`
	Surface          string   `json:"surface"`
	Folder           string   `json:"folder"`
	Slot             string   `json:"slot"`
	GridboxID        string   `json:"gridbox_id"`
	RootTechnologyID string   `json:"root_technology_id"`
	TechnologyIDs    []string `json:"technology_ids"`

	TechnologyDatabaseID int64             `json:"technology_db_id"`
	Source               string            `json:"source"`
	Name                 string            `json:"name"`
	NameEn               string            `json:"name_en"`
	NameZh               string            `json:"name_zh"`
	DescEn               string            `json:"desc_en"`
	DescZh               string            `json:"desc_zh"`
	DisplayNameLocs      map[string]string `json:"display_name_locs"`
	DescriptionLocs      map[string]string `json:"description_locs"`
	EnableEquipments     []string          `json:"enable_equipments"`
	TechLine             string            `json:"tech_line"`
	StartYear            int               `json:"start_year"`
	PositionX            float64           `json:"position_x"`
	PositionY            float64           `json:"position_y"`
	FolderPlacements     []FolderPlacement `json:"folder_placements"`
	Path                 []PathEdge        `json:"path"`
}

type MutableTechnology struct {
	Name             string            `json:"name"`
	NameEn           string            `json:"name_en"`
	NameZh           string            `json:"name_zh"`
	DescEn           string            `json:"desc_en"`
	DescZh           string            `json:"desc_zh"`
	DisplayNameLocs  map[string]string `json:"display_name_locs"`
	DescriptionLocs  map[string]string `json:"description_locs"`
	EnableEquipments []string          `json:"enable_equipments"`
}

type UpdatePlan struct {
	ProjectID            int64             `json:"project_id"`
	BranchID             string            `json:"branch_id"`
	TechnologyDatabaseID int64             `json:"technology_db_id"`
	RootTechnologyID     string            `json:"root_technology_id"`
	ChangedFields        []string          `json:"changed_fields"`
	Mutable              MutableTechnology `json:"mutable"`
	View                 BranchView        `json:"view"`
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func cloneMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func clonePlacements(values []FolderPlacement) []FolderPlacement {
	if len(values) == 0 {
		return []FolderPlacement{}
	}
	return append([]FolderPlacement(nil), values...)
}

func clonePaths(values []PathEdge) []PathEdge {
	if len(values) == 0 {
		return []PathEdge{}
	}
	return append([]PathEdge(nil), values...)
}

func normalizedTechnology(value Technology) Technology {
	value.TechID = strings.TrimSpace(value.TechID)
	value.DisplayNameLocs = cloneMap(value.DisplayNameLocs)
	value.DescriptionLocs = cloneMap(value.DescriptionLocs)
	value.EnableEquipments = cloneStrings(value.EnableEquipments)
	value.FolderPlacements = clonePlacements(value.FolderPlacements)
	value.Path = clonePaths(value.Path)
	return value
}
