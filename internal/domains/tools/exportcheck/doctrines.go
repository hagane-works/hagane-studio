package exportcheck

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
