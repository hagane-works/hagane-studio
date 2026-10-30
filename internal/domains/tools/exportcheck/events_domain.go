package exportcheck

type Event struct {
	ID                int64             `json:"id"`
	ProjectID         int64             `json:"project_id"`
	NamespaceID       int64             `json:"namespace_id"`
	EventID           string            `json:"event_id"`
	EventNumber       int               `json:"event_number"`
	EventType         string            `json:"event_type"`
	Title             any               `json:"title"`
	Description       any               `json:"description"`
	Picture           string            `json:"picture"`
	IsTriggeredOnly   bool              `json:"is_triggered_only"`
	FireOnlyOnce      bool              `json:"fire_only_once"`
	Hidden            bool              `json:"hidden"`
	Major             bool              `json:"major"`
	Trigger           string            `json:"trigger"`
	MeanTimeToHappen  string            `json:"mean_time_to_happen"`
	Immediate         string            `json:"immediate"`
	After             string            `json:"after"`
	Options           []any             `json:"options"`
	TimeoutDays       int               `json:"timeout_days"`
	RawExtras         map[string]any    `json:"raw_extras"`
	CountryTag        string            `json:"country_tag"`
	ScopeKind         string            `json:"scope_kind"`
	SharedCountryTags []string          `json:"shared_country_tags"`
	TitleEn           string            `json:"title_en"`
	TitleZh           string            `json:"title_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	AutoFireOnStartup bool              `json:"auto_fire_on_startup"`
	AutoFireAfterDays int               `json:"auto_fire_after_days"`
	OnActionAttach    string            `json:"on_action_attach"`
	Source            string            `json:"source"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	DescriptionLocs   map[string]string `json:"description_locs"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
	Warnings          []string          `json:"warnings"`
}
