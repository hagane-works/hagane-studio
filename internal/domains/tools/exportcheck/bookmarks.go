package exportcheck

type Bookmark struct {
	EditRevision        string            `json:"edit_revision"`
	Diagnostics         *[]SaveDiagnostic `json:"diagnostics,omitempty"`
	ID                  int64             `json:"id"`
	ProjectID           int64             `json:"project_id"`
	BookmarkID          string            `json:"bookmark_id"`
	NameEn              string            `json:"name_en"`
	NameZh              string            `json:"name_zh"`
	DescEn              string            `json:"desc_en"`
	DescZh              string            `json:"desc_zh"`
	Date                string            `json:"date"`
	Picture             string            `json:"picture"`
	DefaultCountry      string            `json:"default_country"`
	DefaultFlag         bool              `json:"default_flag"`
	SelectableCountries []map[string]any  `json:"selectable_countries"`
	Filters             []string          `json:"filters"`
	DisplayNameLocs     map[string]string `json:"display_name_locs"`
	DescriptionLocs     map[string]string `json:"description_locs"`
	Source              string            `json:"source"`
	CreatedAt           string            `json:"created_at"`
	UpdatedAt           string            `json:"updated_at"`
}
