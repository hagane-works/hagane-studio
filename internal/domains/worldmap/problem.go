package worldmap

import (
	"errors"
	"net/http"

	"haganeworks/internal/hoi4/mapgen"
)

const (
	problemBusy            = "busy"
	problemUnreadable      = "unreadable"
	problemNotSaved        = "not_saved"
	problemBadRequest      = "bad_request"
	problemTooLarge        = "too_large"
	problemNothing         = "nothing"
	problemOwnMap          = "own_map"
	problemNoCountry       = "no_country"
	problemStateDrowned    = "state_drowned"
	problemStateMissing    = "state_missing"
	problemSeaDivisions    = "sea_divisions"
	problemSeaCapital      = "sea_capital"
	problemSeaUnreached    = "sea_unreached"
	problemSeaNoRegion     = "sea_no_region"
	problemRegionEmpty     = "region_empty"
	problemLandBreaksMap   = "land_breaks_map"
	problemWorldBreaksMap  = "world_breaks_map"
	problemUndoFailed      = "undo_failed"
	problemNothingToUndo   = "nothing_to_undo"
	problemMapChanged      = "map_changed"
	problemMapChangesMoved = "map_changes_moved"
	problemImportedStates  = "imported_states"

	problemUploadMissing         = "upload_missing"
	problemUploadUnreadable      = "upload_unreadable"
	problemUploadJPEG            = "upload_jpeg"
	problemSketchUnreadable      = "sketch_unreadable"
	problemWorldLandSize         = "world_land_size"
	problemWorldPerState         = "world_per_state"
	problemTagInvalid            = "tag_invalid"
	problemWorldNoCountries      = "world_no_countries"
	problemWorldTooManyCountries = "world_too_many_countries"
	problemOwnersNotPNG          = "owners_not_png"
	problemOwnersTooLarge        = "owners_too_large"
	problemOwnersExtraCountry    = "owners_extra_country"
	problemWorldCountryUnpainted = "world_country_unpainted"
	problemWorldSize             = "world_size"
	problemWorldSizeRange        = "world_size_range"
	problemWorldNoLand           = "world_no_land"
	problemWorldFewStates        = "world_few_states"

	problemModNeedsStatesRegions = "mod_needs_states_regions"
	problemModNeedsDefinition    = "mod_needs_definition"
	problemModFileOutside        = "mod_file_outside"
	problemModTooManyFiles       = "mod_too_many_files"
	problemModFileUnreadable     = "mod_file_unreadable"
	problemModTextTooLarge       = "mod_text_too_large"
	problemModUnknownTags        = "mod_unknown_tags"

	problemKindLandWater      = "kind_land_water"
	problemKindInvalid        = "kind_invalid"
	problemLakeStrandsCoast   = "lake_strands_coast"
	problemTerrainUnknown     = "terrain_unknown"
	problemWaterContinent     = "water_continent"
	problemContinentRange     = "continent_range"
	problemStateLandOnly      = "state_land_only"
	problemStateLastProvince  = "state_last_province"
	problemStateLastLand      = "state_last_land"
	problemRegionMissing      = "region_missing"
	problemRegionLastProvince = "region_last_province"
	problemRegionFileText     = "region_file_text"
	problemProvinceRegion     = "province_state_region"
	problemProvinceSeaSplit   = "province_sea_split"
	problemNoProvince         = "no_province"
	problemWaterTerrain       = "water_terrain"
)

type mapProblem struct {
	Code      string         `json:"code"`
	Detail    string         `json:"detail"`
	Values    map[string]any `json:"values,omitempty"`
	Provinces []int          `json:"provinces,omitempty"`
}

func (p *mapProblem) Error() string { return p.Detail }

func newMapProblem(code, detail string) *mapProblem { return &mapProblem{Code: code, Detail: detail} }

func (p *mapProblem) with(key string, value any) *mapProblem {
	if p.Values == nil {
		p.Values = map[string]any{}
	}
	p.Values[key] = value
	return p
}

func (p *mapProblem) about(provinces ...int) *mapProblem {
	p.Provinces = append(p.Provinces, provinces...)
	return p
}

func mapProblemOf(err error) *mapProblem {
	var edit *mapgen.AddError
	if errors.As(err, &edit) {
		code := edit.Code
		if code == "" {
			code = problemBadRequest
		}
		return &mapProblem{Code: code, Detail: edit.Message, Values: edit.Values, Provinces: edit.Provinces}
	}
	var upload *mapgen.UploadError
	if errors.As(err, &upload) {
		code := upload.Code
		if code == "" {
			code = problemBadRequest
		}
		return &mapProblem{Code: code, Detail: upload.Message, Values: upload.Values}
	}
	var refused *landRefusal
	if errors.As(err, &refused) {
		return newMapProblem(problemOwnMap, refused.message)
	}
	return nil
}

func writeMapProblem(w http.ResponseWriter, status int, problem *mapProblem) {
	writeJSON(w, status, problem)
}

func refuseMap(w http.ResponseWriter, status int, code, detail string) {
	writeMapProblem(w, status, newMapProblem(code, detail))
}

func requestProblem(detail string) *mapProblem {
	switch {
	case hasAnyPrefix(detail, "一次", "新土地的范围太大"):
		return newMapProblem(problemTooLarge, detail)
	case hasAnyPrefix(detail, "没有要", "河流没有变化", "没有收到"):
		return newMapProblem(problemNothing, detail)
	}
	return newMapProblem(problemBadRequest, detail)
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func mapProblemOr(err error, code string) *mapProblem {
	if problem := mapProblemOf(err); problem != nil {
		return problem
	}
	return newMapProblem(code, err.Error())
}
