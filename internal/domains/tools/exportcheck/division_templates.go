package exportcheck

type DivisionTemplate struct {
	ID                 int64            `json:"id"`
	ProjectID          int64            `json:"project_id"`
	OobFile            string           `json:"oob_file"`
	TemplateName       string           `json:"template_name"`
	CountryTag         string           `json:"country_tag"`
	DivisionNamesGroup string           `json:"division_names_group"`
	Regiments          []map[string]any `json:"regiments"`
	Support            []map[string]any `json:"support"`
	IsLocked           *bool            `json:"is_locked"`
	Priority           *int             `json:"priority"`
	Ordering           int              `json:"ordering"`
	OobUnitsTail       string           `json:"oob_units_tail"`
	RawExtras          map[string]any   `json:"raw_extras"`
	Source             string           `json:"source"`
	Warnings           []string         `json:"warnings"`
}
