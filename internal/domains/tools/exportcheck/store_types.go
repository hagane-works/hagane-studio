package exportcheck

import "time"

type Project struct {
	ID                     int64             `json:"id"`
	UUID                   *string           `json:"uuid"`
	Name                   string            `json:"name"`
	CountryTag             string            `json:"country_tag"`
	CountryName            string            `json:"country_name"`
	CountryTags            []string          `json:"country_tags"`
	ModPath                string            `json:"mod_path"`
	IdeologyIconPaths      map[string]string `json:"ideology_icon_paths"`
	Version                string            `json:"version"`
	Picture                string            `json:"picture"`
	Tags                   []string          `json:"tags"`
	Dependencies           []string          `json:"dependencies"`
	SupportedVersion       string            `json:"supported_version"`
	ImportedReplacePaths   []string          `json:"imported_descriptor_replace_paths"`
	BaseModTemplateID      *int64            `json:"basemod_template_id"`
	BaseModPinnedVersion   *string           `json:"basemod_pinned_version"`
	Loaded                 bool              `json:"is_loaded_existing_mod"`
	StartDate              *string           `json:"start_date"`
	EndDate                *string           `json:"end_date"`
	ScenarioHistoryPolicy  string            `json:"scenario_history_policy"`
	StageStateIDs          []int64           `json:"-"`
	TechYearColumns        []any             `json:"tech_year_columns"`
	ExportLanguages        []string          `json:"export_languages"`
	IdeologyRegistry       map[string]any    `json:"ideology_registry"`
	TechSurfaceManifest    map[string]any    `json:"tech_surface_manifest"`
	IdeologySourceManifest map[string]any    `json:"ideology_source_manifest"`
	OwnedDLCs              *[]string         `json:"-"`
	CreatedAt              time.Time         `json:"-"`
	UpdatedAt              time.Time         `json:"-"`
}
